// Package server is the network edge of WineCraft: WebSocket connections,
// the hub that owns the game state, and the static file server for the web
// client. Game rules live in package game; this package only moves bytes.
package server

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Config tunes a hub. Zero values pick the defaults.
type Config struct {
	JoinCode     string
	Tick         time.Duration // world broadcast interval; default 100 ms (10 Hz)
	Heartbeat    time.Duration // server Ping interval; default 15 s (Cloudflare drops idle sockets)
	Idle         time.Duration // drop a client that sends nothing for this long; default 45 s
	HelloTimeout time.Duration // time allowed to send Hello after connecting; default 10 s
	SendBuffer   int           // queued messages per client before it counts as too slow; default 512
	MaxPerIP     int           // open connections per client IP; default 4
	MsgInterval  time.Duration // sustained client message rate: one per interval; default 20 ms (50/s)
	MsgBurst     int           // client message burst; default 100
	FailInterval time.Duration // a wrong join code is forgiven after this long; default 30 s
	FailBurst    int           // wrong join codes per IP before a lockout; default 5
	Log          *slog.Logger
}

func (c *Config) defaults() {
	def := func(d *time.Duration, v time.Duration) {
		if *d == 0 {
			*d = v
		}
	}
	defInt := func(n *int, v int) {
		if *n == 0 {
			*n = v
		}
	}
	def(&c.Tick, 100*time.Millisecond)
	def(&c.Heartbeat, 15*time.Second)
	def(&c.Idle, 45*time.Second)
	def(&c.HelloTimeout, 10*time.Second)
	def(&c.MsgInterval, 20*time.Millisecond)
	def(&c.FailInterval, 30*time.Second)
	defInt(&c.SendBuffer, 512)
	defInt(&c.MaxPerIP, 4)
	defInt(&c.MsgBurst, 100)
	defInt(&c.FailBurst, 5)
	if c.Log == nil {
		c.Log = slog.Default()
	}
}

// Hub owns the game state. Exactly one goroutine (Run) touches it; connection
// goroutines talk to it through channels.
type Hub struct {
	state   *game.State
	cfg     Config
	joins   chan joinReq
	leaves  chan *client
	inbound chan inbound
	done    chan struct{}
	clients map[uint32]*client
	online  atomic.Int32
	gate    *ipGate
}

type joinReq struct {
	c     *client
	name  string
	reply chan error
}

type inbound struct {
	c   *client
	msg proto.Msg
}

// NewHub creates a hub for state.
func NewHub(state *game.State, cfg Config) *Hub {
	cfg.defaults()
	return &Hub{
		state:   state,
		cfg:     cfg,
		joins:   make(chan joinReq),
		leaves:  make(chan *client, 64),
		inbound: make(chan inbound, 1024),
		done:    make(chan struct{}),
		clients: make(map[uint32]*client),
		gate:    newIPGate(),
	}
}

// Online returns the number of connected players.
func (h *Hub) Online() int { return int(h.online.Load()) }

// Run is the game loop. It returns when ctx is cancelled, closing every
// connection.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)
	tick := time.NewTicker(h.cfg.Tick)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			for _, c := range h.clients {
				// Close waits for the peer's reply; don't let one slow client hold up the rest.
				go c.conn.Close(websocket.StatusGoingAway, "server shutting down")
			}
			return
		case j := <-h.joins:
			j.reply <- h.join(j.c, j.name)
		case c := <-h.leaves:
			h.leave(c)
		case in := <-h.inbound:
			h.handle(in.c, in.msg)
		case <-tick.C:
			h.flush()
		}
	}
}

func (h *Hub) join(c *client, name string) error {
	p, err := h.state.Join(name)
	if err != nil {
		return err
	}
	c.id = p.ID
	c.name = p.Name
	h.clients[p.ID] = c
	h.online.Store(int32(len(h.clients)))

	m := h.state.Map
	h.sendTo(c, &proto.Welcome{ID: p.ID, X: int32(p.Pos.X), Y: int32(p.Pos.Y), Bounds: m.Bounds, Sandbox: m.Sandbox})
	lo, _, _ := world.ChunkOf(m.Bounds.X, m.Bounds.Y)
	hi, _, _ := world.ChunkOf(m.Bounds.X+m.Bounds.W-1, m.Bounds.Y+m.Bounds.H-1)
	for cy := lo.Y; cy <= hi.Y; cy++ {
		for cx := lo.X; cx <= hi.X; cx++ {
			if ch := m.World.Chunk(world.ChunkCoord{X: cx, Y: cy}); ch != nil {
				h.sendTo(c, &proto.Chunk{X: cx, Y: cy, Layers: ch.Layers})
			}
		}
	}
	for _, other := range h.state.Players() {
		if other.ID != p.ID {
			h.sendTo(c, playerState(*other))
		}
	}
	h.cfg.Log.Info("player joined", "id", p.ID, "name", p.Name, "ip", c.ip, "online", len(h.clients))
	return nil
}

func (h *Hub) leave(c *client) {
	if c.id == 0 || h.clients[c.id] != c {
		return
	}
	delete(h.clients, c.id)
	h.online.Store(int32(len(h.clients)))
	h.state.Leave(c.id)
	close(c.send)
	h.broadcast(&proto.PlayerLeft{ID: c.id})
	h.cfg.Log.Info("player left", "id", c.id, "name", c.name, "online", len(h.clients))
}

func (h *Hub) handle(c *client, m proto.Msg) {
	if h.clients[c.id] != c {
		return // already gone
	}
	switch m := m.(type) {
	case *proto.Ping:
		// client heartbeat: receiving it is the point
	case *proto.Move:
		h.state.Move(c.id, m.Dir, m.Seq, time.Now())
	case *proto.Edit:
		ch, err := h.state.Edit(c.id, int(m.X), int(m.Y), m.Layer, m.Tile, time.Now())
		switch {
		case errors.Is(err, game.ErrRateLimited):
			h.sendTo(c, &proto.Error{Code: proto.ErrRateLimited})
		case err != nil:
			h.sendTo(c, &proto.Error{Code: proto.ErrNotAllowed})
		default:
			h.broadcast(&proto.TileUpdate{X: int32(ch.X), Y: int32(ch.Y), Layer: ch.Layer, Tile: ch.Tile})
		}
	default:
		// anything else from a client is ignored
	}
}

// flush broadcasts every player whose state changed since the last tick.
func (h *Hub) flush() {
	for _, p := range h.state.TakeDirty() {
		h.broadcast(playerState(p))
	}
}

func playerState(p game.Player) *proto.PlayerState {
	return &proto.PlayerState{ID: p.ID, X: int32(p.Pos.X), Y: int32(p.Pos.Y), Facing: p.Facing, Seq: p.Seq, Name: p.Name}
}

func (h *Hub) broadcast(m proto.Msg) {
	b, err := proto.Encode(m)
	if err != nil {
		h.cfg.Log.Error("encode broadcast", "type", m.Type(), "err", err)
		return
	}
	for _, c := range h.clients {
		h.enqueue(c, b)
	}
}

func (h *Hub) sendTo(c *client, m proto.Msg) {
	b, err := proto.Encode(m)
	if err != nil {
		h.cfg.Log.Error("encode", "type", m.Type(), "err", err)
		return
	}
	h.enqueue(c, b)
}

// enqueue never blocks the game loop: a client whose queue is full is too
// slow to keep up and gets disconnected.
func (h *Hub) enqueue(c *client, b []byte) {
	select {
	case c.send <- b:
	default:
		h.cfg.Log.Warn("dropping slow client", "id", c.id, "ip", c.ip)
		c.conn.CloseNow()
	}
}

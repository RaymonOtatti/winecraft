package client

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/RaymonOtatti/winecraft/internal/proto"
)

// NetState is the connection's state, for the HUD.
type NetState int32

const (
	Connecting NetState = iota
	Online
	Offline  // dropped; reconnecting after a backoff
	Rejected // the server refused us for good (wrong code, version or name)
)

func (s NetState) String() string {
	return [...]string{"conectando", "en línea", "sin conexión", "rechazado"}[s]
}

// NetConfig says where and as whom to connect.
type NetConfig struct {
	URL        string // ws:// or wss:// …/ws
	Name       string
	JoinCode   string
	Token      string        // keeps your progress across visits; "" plays anonymously
	Heartbeat  time.Duration // client Ping interval; default 15 s (the server drops 45 s of silence)
	Backoff    time.Duration // first reconnect wait; default 1 s, doubling to MaxBackoff
	MaxBackoff time.Duration // default 10 s
}

// Net owns the WebSocket. It dials, reads and writes on its own goroutines
// (never inside a browser callback, where blocking would deadlock) and hands
// messages to the game loop through channels the game polls without blocking.
// It reconnects by itself until the server rejects it for good.
type Net struct {
	cfg   NetConfig
	in    chan proto.Msg
	out   chan proto.Msg
	state atomic.Int32
}

// StartNet connects in the background and keeps reconnecting until ctx ends.
func StartNet(ctx context.Context, cfg NetConfig) *Net {
	if cfg.Heartbeat == 0 {
		cfg.Heartbeat = 15 * time.Second
	}
	if cfg.Backoff == 0 {
		cfg.Backoff = time.Second
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = 10 * time.Second
	}
	n := &Net{cfg: cfg, in: make(chan proto.Msg, 4096), out: make(chan proto.Msg, 256)}
	go n.run(ctx)
	return n
}

// State reports the connection state.
func (n *Net) State() NetState { return NetState(n.state.Load()) }

// Recv returns the next server message, if one is waiting. It never blocks.
func (n *Net) Recv() (proto.Msg, bool) {
	select {
	case m := <-n.in:
		return m, true
	default:
		return nil, false
	}
}

// Send queues m for the server. It never blocks: while offline, or if the
// queue is full, m is dropped (the next Welcome or PlayerState resyncs us).
func (n *Net) Send(m proto.Msg) bool {
	if n.State() != Online {
		return false
	}
	select {
	case n.out <- m:
		return true
	default:
		return false
	}
}

func (n *Net) run(ctx context.Context) {
	backoff := n.cfg.Backoff
	for ctx.Err() == nil {
		n.state.Store(int32(Connecting))
		welcomed, rejected := n.session(ctx)
		if rejected {
			n.state.Store(int32(Rejected))
			return
		}
		n.state.Store(int32(Offline))
		if welcomed {
			backoff = n.cfg.Backoff // it worked a moment ago: retry soon
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, n.cfg.MaxBackoff)
	}
}

// session runs one connection from dial to drop.
func (n *Net) session(ctx context.Context) (welcomed, rejected bool) {
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, _, err := websocket.Dial(dctx, n.cfg.URL, nil)
	cancel()
	if err != nil {
		return false, false
	}
	defer conn.CloseNow()
	conn.SetReadLimit(proto.MaxMessage)

	// Moves queued for the previous connection belong to a player who no
	// longer exists on the server: never replay them to the new one.
	for drained := false; !drained; {
		select {
		case <-n.out:
		default:
			drained = true
		}
	}

	hello, err := proto.Encode(&proto.Hello{Version: proto.Version, Name: n.cfg.Name, JoinCode: n.cfg.JoinCode, Token: n.cfg.Token})
	if err != nil || write(ctx, conn, hello) != nil {
		return false, false
	}

	sctx, stop := context.WithCancel(ctx)
	defer stop()
	go n.writer(sctx, conn)

	for {
		_, b, err := conn.Read(sctx)
		if err != nil {
			return welcomed, rejected
		}
		m, err := proto.Decode(b)
		if err != nil {
			return welcomed, rejected // a server we cannot understand: reconnect
		}
		switch m := m.(type) {
		case *proto.Ping:
			continue // the server's heartbeat; our own Pings keep us alive
		case *proto.Welcome:
			welcomed = true
			n.state.Store(int32(Online))
		case *proto.Error:
			if !welcomed && fatal(m.Code) {
				rejected = true
			}
		}
		select {
		case n.in <- m:
		case <-sctx.Done():
			return welcomed, rejected
		}
	}
}

// fatal errors will not change by retrying.
func fatal(code uint8) bool {
	return code == proto.ErrBadJoinCode || code == proto.ErrBadVersion || code == proto.ErrBadName
}

func (n *Net) writer(ctx context.Context, conn *websocket.Conn) {
	beat := time.NewTicker(n.cfg.Heartbeat)
	defer beat.Stop()
	var nonce uint32
	for {
		var m proto.Msg
		select {
		case <-ctx.Done():
			return
		case m = <-n.out:
		case <-beat.C:
			nonce++
			m = &proto.Ping{Nonce: nonce}
		}
		b, err := proto.Encode(m)
		if err != nil {
			continue
		}
		if write(ctx, conn, b) != nil {
			conn.CloseNow()
			return
		}
	}
}

func write(ctx context.Context, conn *websocket.Conn, b []byte) error {
	wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return conn.Write(wctx, websocket.MessageBinary, b)
}

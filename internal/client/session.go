package client

import (
	"strings"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Session is the client's view of the shared world, built only from what the
// server sends. Only the game goroutine touches it; the network goroutine
// hands messages over through a channel.
type Session struct {
	World     *world.World
	Me        *Walker
	MyID      uint32
	Joined    bool
	Bounds    world.Rect
	BuildZone world.Rect
	Players   map[uint32]*Remote
	Inv       map[world.ItemID]int // what the server says we carry
	Hotbar    *Hotbar
	Notice    string // the last server error, for the HUD
	NoticeSeq int    // bumps on every error, so the same message can toast twice
	Chat      string // the latest mentor chat line, for the chat panel
	ChatSeq   int    // bumps on every new chat line
}

// NewSession starts empty, waiting for a Welcome.
func NewSession() *Session {
	w := world.New()
	return &Session{World: w, Me: NewWalker(w, world.Point{}), Players: make(map[uint32]*Remote), Inv: make(map[world.ItemID]int), Hotbar: NewHotbar()}
}

var errorText = map[uint8]string{
	proto.ErrBadVersion:  "Versión distinta: recargá la página",
	proto.ErrBadJoinCode: "Código incorrecto",
	proto.ErrServerFull:  "El servidor está lleno",
	proto.ErrBadName:     "Nombre no válido",
	proto.ErrRateLimited: "Más despacio",
	proto.ErrNotAllowed:  "No se puede ahí",
	proto.ErrNoMaterial:  "Te faltan materiales",
}

// Apply updates the session with one server message.
func (s *Session) Apply(m proto.Msg) {
	switch m := m.(type) {
	case *proto.Welcome:
		s.MyID, s.Joined = m.ID, true
		s.Bounds, s.BuildZone = m.Bounds, m.BuildZone
		s.Me.Reset(world.Point{X: int(m.X), Y: int(m.Y)})
		clear(s.Players)
		s.Notice = ""
	case *proto.Chunk:
		s.World.PutChunk(world.ChunkCoord{X: m.X, Y: m.Y}, &m.Layers)
	case *proto.PlayerState:
		pos := world.Point{X: int(m.X), Y: int(m.Y)}
		if m.ID == s.MyID {
			s.Me.Reconcile(pos, m.Facing, m.Seq)
			return
		}
		r := s.Players[m.ID]
		if r == nil {
			r = &Remote{ID: m.ID, Pos: pos, from: pos}
			s.Players[m.ID] = r
		}
		r.Name = m.Name
		r.moveTo(pos, m.Facing)
	case *proto.PlayerLeft:
		delete(s.Players, m.ID)
	case *proto.TileUpdate:
		if m.Tile == world.None {
			s.World.Clear(m.Layer, int(m.X), int(m.Y))
		} else {
			s.World.Set(int(m.X), int(m.Y), m.Tile)
		}
	case *proto.Hotbar:
		s.Hotbar.Slots = m.Slots
	case *proto.Inventory:
		clear(s.Inv)
		for _, it := range m.Items {
			s.Inv[it.ID] = int(it.Count)
		}
	case *proto.Error:
		s.NoticeSeq++
		s.Notice = errorText[m.Code]
		if s.Notice == "" {
			s.Notice = "Error del servidor"
		}
	case *proto.Chat:
		s.ChatSeq++
		s.Chat = m.Text
	}
}

// notify shows a message locally, the way server errors are shown.
func (s *Session) notify(text string) {
	s.NoticeSeq++
	s.Notice = text
}

// Tick advances the other players' walking animations.
func (s *Session) Tick(dt time.Duration) {
	for _, r := range s.Players {
		r.tick(dt)
	}
}

// Remote is another player, drawn where the server last said they were,
// sliding there over one step.
type Remote struct {
	ID      uint32
	Name    string
	Pos     world.Point
	Facing  world.Dir
	from    world.Point
	moving  bool
	elapsed time.Duration
	dur     time.Duration
}

func (r *Remote) moveTo(p world.Point, facing world.Dir) {
	r.Facing = facing
	if p == r.Pos {
		return
	}
	dist := abs(p.X-r.Pos.X) + abs(p.Y-r.Pos.Y)
	if dist > 2 { // a correction or a reconnect: snap instead of sliding across the map
		r.Pos, r.from, r.moving = p, p, false
		return
	}
	r.from, r.Pos, r.moving, r.elapsed = r.Pos, p, true, 0
	r.dur = time.Duration(dist) * StepDuration
}

func (r *Remote) tick(dt time.Duration) {
	if !r.moving {
		return
	}
	if r.elapsed += dt; r.elapsed >= r.dur {
		r.moving, r.elapsed = false, 0
	}
}

// Moving reports whether the player is mid-step.
func (r *Remote) Moving() bool { return r.moving }

// DrawPos is the interpolated world-pixel position of the sprite's top-left.
func (r *Remote) DrawPos() (float64, float64) {
	if !r.moving {
		return float64(r.Pos.X * art.Tile), float64(r.Pos.Y * art.Tile)
	}
	p := float64(r.elapsed) / float64(r.dur)
	return (float64(r.from.X) + float64(r.Pos.X-r.from.X)*p) * art.Tile,
		(float64(r.from.Y) + float64(r.Pos.Y-r.from.Y)*p) * art.Tile
}

// Frame is the walk frame, as for the local player.
func (r *Remote) Frame() int {
	if !r.moving {
		return 0
	}
	if p := float64(r.elapsed) / float64(r.dur); p >= 0.25 && p < 0.75 {
		return 1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var fold = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
)

// DisplayName folds Spanish accents for the ASCII debug font used until real
// pixel fonts arrive; anything else non-ASCII shows as '?'.
func DisplayName(name string) string {
	name = fold.Replace(name)
	b := []byte(name)
	out := b[:0]
	for _, r := range name {
		if r < 0x20 || r > 0x7e {
			r = '?'
		}
		out = append(out, byte(r))
	}
	return string(out)
}

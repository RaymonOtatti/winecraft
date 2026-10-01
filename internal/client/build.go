package client

import (
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Hotbar is the row of tiles the player can place.
type Hotbar struct {
	Slots []world.TileID
	sel   int
}

// NewHotbar holds every placeable tile, in registry order.
func NewHotbar() *Hotbar {
	h := &Hotbar{}
	for id := 1; id < world.NumTiles(); id++ {
		if world.Def(world.TileID(id)).Placeable {
			h.Slots = append(h.Slots, world.TileID(id))
		}
	}
	return h
}

// Select picks slot i; out-of-range does nothing.
func (h *Hotbar) Select(i int) {
	if i >= 0 && i < len(h.Slots) {
		h.sel = i
	}
}

func (h *Hotbar) Next()                  { h.sel = (h.sel + 1) % len(h.Slots) }
func (h *Hotbar) Prev()                  { h.sel = (h.sel + len(h.Slots) - 1) % len(h.Slots) }
func (h *Hotbar) Index() int             { return h.sel }
func (h *Hotbar) Selected() world.TileID { return h.Slots[h.sel] }

// Action is what the A and B buttons do.
type Action int

const (
	ActionPlace Action = iota // A
	ActionBreak               // B
)

// Target is the tile the player faces: where A and B act.
func (s *Session) Target() world.Point {
	dx, dy := s.Me.Facing.Delta()
	return world.Point{X: s.Me.Pos.X + dx, Y: s.Me.Pos.Y + dy}
}

// CanBuildAt reports whether the faced tile is one the server could accept:
// standing still, inside the sandbox. The server still decides.
func (s *Session) CanBuildAt(p world.Point) bool {
	return s.Joined && !s.Me.Moving() && s.Sandbox.Contains(p.X, p.Y)
}

// EditFor turns a button press into the Edit to send, or false when there is
// clearly nothing to do (outside the sandbox, mid-step, nothing to break), to
// save the server's message budget. Breaking takes what stands on the tile
// first, then a floor.
func (s *Session) EditFor(a Action, tile world.TileID) (*proto.Edit, bool) {
	p := s.Target()
	if !s.CanBuildAt(p) {
		return nil, false
	}
	e := &proto.Edit{X: int32(p.X), Y: int32(p.Y)}
	switch a {
	case ActionPlace:
		e.Layer, e.Tile = world.Def(tile).Layer, tile
	case ActionBreak:
		switch {
		case world.Def(s.World.At(world.Object, p.X, p.Y)).Breakable:
			e.Layer = world.Object
		case world.Def(s.World.At(world.Ground, p.X, p.Y)).Breakable:
			e.Layer = world.Ground
		default:
			return nil, false
		}
	}
	return e, true
}

// CountFor is how many of the item that places tile we carry.
func (s *Session) CountFor(tile world.TileID) int {
	it, ok := world.ItemForTile(tile)
	if !ok {
		return 0
	}
	return s.Inv[it]
}

// Primary is the A button: harvest the vine you face, otherwise place the
// selected tile. Without the material it tells the player and sends nothing.
func (s *Session) Primary(tile world.TileID) (proto.Msg, bool) {
	p := s.Target()
	if !s.Joined || s.Me.Moving() {
		return nil, false
	}
	if s.World.At(world.Object, p.X, p.Y) == world.Vine {
		return &proto.Interact{X: int32(p.X), Y: int32(p.Y)}, true
	}
	e, ok := s.EditFor(ActionPlace, tile)
	if !ok {
		return nil, false
	}
	if s.CountFor(tile) < 1 {
		s.notify(errorText[proto.ErrNoMaterial])
		return nil, false
	}
	return e, true
}

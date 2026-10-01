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

// Action is what a button does to the faced tile.
type Action int

const (
	ActionUse   Action = iota // Space: harvest a vine (later: talk, open)
	ActionBuild               // C: place the selected hotbar item
	ActionBreak               // X: break what stands there, else the floor
)

// Target is the tile the player faces: where A and B act.
func (s *Session) Target() world.Point {
	dx, dy := s.Me.Facing.Delta()
	return world.Point{X: s.Me.Pos.X + dx, Y: s.Me.Pos.Y + dy}
}

// inZone reports whether p is somewhere players may build.
func (s *Session) inZone(p world.Point) bool { return s.BuildZone.Contains(p.X, p.Y) }

// openGround reports whether something can be built on p: bare soil with
// nothing standing on it.
func (s *Session) openGround(p world.Point) bool {
	g := s.World.At(world.Ground, p.X, p.Y)
	return s.World.At(world.Object, p.X, p.Y) == world.None && (g == world.Grass || g == world.Dirt || g == world.Sand)
}

// CanBuildAt reports whether the server could accept a build at p: standing
// still, inside the build zone, on open ground. The cursor shows this; the
// server still decides.
func (s *Session) CanBuildAt(p world.Point) bool {
	return s.Joined && !s.Me.Moving() && s.inZone(p) && s.openGround(p)
}

// EditFor turns a button press into the Edit to send, or false when there is
// clearly nothing to do (outside the build zone, mid-step, nothing to break), to
// save the server's message budget. Breaking takes what stands on the tile
// first, then a floor.
func (s *Session) EditFor(a Action, tile world.TileID) (*proto.Edit, bool) {
	p := s.Target()
	if !s.Joined || s.Me.Moving() || !s.inZone(p) {
		return nil, false
	}
	e := &proto.Edit{X: int32(p.X), Y: int32(p.Y)}
	switch a {
	case ActionBuild:
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

// Act turns a button press into the message to send. When the action can't
// happen it says why in a notice and sends nothing (presses mid-step are
// ignored quietly). The server still validates everything.
func (s *Session) Act(a Action, tile world.TileID) (proto.Msg, bool) {
	if !s.Joined || s.Me.Moving() {
		return nil, false
	}
	p := s.Target()
	switch a {
	case ActionUse:
		if s.World.At(world.Object, p.X, p.Y) == world.Vine {
			return &proto.Interact{X: int32(p.X), Y: int32(p.Y)}, true
		}
		s.notify("No hay nada para usar acá")
	case ActionBreak:
		if !s.inZone(p) {
			s.notify("Acá no se puede construir")
		} else if e, ok := s.EditFor(ActionBreak, 0); ok {
			return e, true
		} else {
			s.notify("No hay nada para romper")
		}
	case ActionBuild:
		switch {
		case !s.inZone(p):
			s.notify("Acá no se puede construir")
		case s.World.At(world.Object, p.X, p.Y) != world.None:
			s.notify("Ese lugar está ocupado")
		case !s.openGround(p):
			s.notify("No se puede construir sobre eso")
		case s.CountFor(tile) < 1:
			s.notify(errorText[proto.ErrNoMaterial])
		default:
			return s.EditFor(ActionBuild, tile)
		}
	}
	return nil, false
}

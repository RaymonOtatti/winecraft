package game

import (
	"errors"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Edit rate limit: a burst of EditBurst edits, then one per EditInterval.
const (
	EditInterval = 250 * time.Millisecond
	EditBurst    = 5
)

var (
	ErrNotAllowed  = errors.New("edit not allowed")
	ErrRateLimited = errors.New("too many edits")
)

// Change is one tile that changed, for broadcasting.
type Change struct {
	X, Y  int
	Layer world.Layer
	Tile  world.TileID
}

// Edit places tile on layer at (x, y), or breaks what is there when tile is
// None. Players may only edit inside the sandbox, on a tile directly next to
// them. Breaking a floor leaves dirt, never a hole. The real world outside
// the sandbox is read-only (PLAN.md §5).
func (s *State) Edit(id uint32, x, y int, layer world.Layer, tile world.TileID, now time.Time) (Change, error) {
	p := s.players[id]
	if p == nil {
		return Change{}, ErrNotAllowed
	}
	// Every attempt costs a token, so a flood of invalid edits is capped too.
	if !p.edits.Take(now, EditInterval, EditBurst, 1) {
		return Change{}, ErrRateLimited
	}
	if layer >= world.NumLayers {
		return Change{}, ErrNotAllowed
	}
	if !s.Map.Sandbox.Contains(x, y) || manhattan(p.Pos, world.Point{X: x, Y: y}) != 1 {
		return Change{}, ErrNotAllowed
	}
	w := s.Map.World
	cur := w.At(layer, x, y)

	var result world.TileID
	if tile == world.None {
		if cur == world.None || !world.Def(cur).Breakable {
			return Change{}, ErrNotAllowed
		}
		if layer == world.Ground {
			if w.At(world.Object, x, y) != world.None {
				return Change{}, ErrNotAllowed // lift what stands on a floor first
			}
			result = world.Dirt
		}
	} else {
		def := world.Def(tile)
		if !def.Placeable || def.Layer != layer {
			return Change{}, ErrNotAllowed
		}
		if layer == world.Object {
			if cur != world.None || !world.Def(w.At(world.Ground, x, y)).Walkable || s.occupied(x, y) {
				return Change{}, ErrNotAllowed
			}
		} else if !isSoil(cur) || w.At(world.Object, x, y) != world.None {
			return Change{}, ErrNotAllowed
		}
		result = tile
	}

	if result == world.None {
		w.Clear(layer, x, y)
	} else {
		w.Set(x, y, result)
	}
	return Change{X: x, Y: y, Layer: layer, Tile: result}, nil
}

// occupied reports whether any player stands on (x, y).
func (s *State) occupied(x, y int) bool {
	for _, p := range s.players {
		if p.Pos.X == x && p.Pos.Y == y {
			return true
		}
	}
	return false
}

// isSoil reports whether a floor may be laid on ground tile g.
func isSoil(g world.TileID) bool {
	return g == world.Grass || g == world.Dirt || g == world.Sand
}

func manhattan(a, b world.Point) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

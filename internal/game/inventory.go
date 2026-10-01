package game

import (
	"errors"
	"slices"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// RegrowAfter is game tuning for the dev map, not a real-world fact.
const RegrowAfter = 2 * time.Minute

// GrapesPerHarvest is what one vine gives (from the tile registry).
var GrapesPerHarvest = world.Def(world.Vine).GatherN

// StarterKit is what every new player carries, so building can start at once.
var StarterKit = map[world.ItemID]int{
	world.ItemPlanks: 10,
	world.ItemFence:  10,
	world.ItemStone:  5,
	world.ItemCrate:  3,
}

var ErrNoMaterial = errors.New("not enough material")

// Stack is a count of one item.
type Stack struct {
	Item  world.ItemID
	Count int
}

// Inventory lists the player's non-empty stacks by item id.
func (p *Player) Inventory() []Stack {
	out := make([]Stack, 0, len(p.Inv))
	for it, n := range p.Inv {
		if n > 0 {
			out = append(out, Stack{Item: it, Count: n})
		}
	}
	slices.SortFunc(out, func(a, b Stack) int { return int(a.Item) - int(b.Item) })
	return out
}

type regrowth struct {
	at   world.Point
	when time.Time
}

// Harvest gathers from the tile at (x, y), next to the player: grapes from a
// vine, and whatever else the tile registry says a tile yields. The tile
// stays, spent, and grows back after RegrowAfter: gathering never destroys a
// vineyard or a quarry. It shares the edit rate limit.
func (s *State) Harvest(id uint32, x, y int, now time.Time) (Change, error) {
	p := s.players[id]
	if p == nil {
		return Change{}, ErrNotAllowed
	}
	if !p.edits.Take(now, EditInterval, EditBurst, 1) {
		return Change{}, ErrRateLimited
	}
	def := world.Def(s.Map.World.At(world.Object, x, y))
	if manhattan(p.Pos, world.Point{X: x, Y: y}) != 1 || def.Gather == world.ItemNone || def.Spent == world.None {
		return Change{}, ErrNotAllowed
	}
	s.Map.World.Set(x, y, def.Spent)
	s.regrow = append(s.regrow, regrowth{at: world.Point{X: x, Y: y}, when: now.Add(RegrowAfter)})
	s.give(p, def.Gather, def.GatherN)
	return Change{X: x, Y: y, Layer: world.Object, Tile: def.Spent}, nil
}

// Tick advances timed world changes (vines regrowing) and returns what changed.
func (s *State) Tick(now time.Time) []Change {
	var out []Change
	kept := s.regrow[:0]
	for _, r := range s.regrow {
		if now.Before(r.when) {
			kept = append(kept, r)
			continue
		}
		if to := world.Def(s.Map.World.At(world.Object, r.at.X, r.at.Y)).RegrowsTo; to != world.None {
			s.Map.World.Set(r.at.X, r.at.Y, to)
			out = append(out, Change{X: r.at.X, Y: r.at.Y, Layer: world.Object, Tile: to})
		}
	}
	s.regrow = kept
	return out
}

// TakeInventoryChanges returns the players whose inventory changed since the
// last call, by id, and clears the set.
func (s *State) TakeInventoryChanges() []uint32 {
	out := make([]uint32, 0, len(s.invDirty))
	for id := range s.invDirty {
		if s.players[id] != nil {
			out = append(out, id)
		}
	}
	clear(s.invDirty)
	slices.Sort(out)
	return out
}

func (s *State) give(p *Player, it world.ItemID, n int) {
	p.Inv[it] += n
	s.invDirty[p.ID] = true
}

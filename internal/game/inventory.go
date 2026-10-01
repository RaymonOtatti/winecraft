package game

import (
	"errors"
	"slices"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Game tuning for the dev map, not real-world facts.
const (
	GrapesPerHarvest = 2
	RegrowAfter      = 2 * time.Minute
)

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

// Harvest picks the grapes of the vine at (x, y), next to the player. The vine
// stays, harvested, and bears again after RegrowAfter: harvesting must never
// destroy a vineyard. It shares the edit rate limit.
func (s *State) Harvest(id uint32, x, y int, now time.Time) (Change, error) {
	p := s.players[id]
	if p == nil {
		return Change{}, ErrNotAllowed
	}
	if !p.edits.Take(now, EditInterval, EditBurst, 1) {
		return Change{}, ErrRateLimited
	}
	if manhattan(p.Pos, world.Point{X: x, Y: y}) != 1 || s.Map.World.At(world.Object, x, y) != world.Vine {
		return Change{}, ErrNotAllowed
	}
	s.Map.World.Set(x, y, world.VineHarvested)
	s.regrow = append(s.regrow, regrowth{at: world.Point{X: x, Y: y}, when: now.Add(RegrowAfter)})
	s.give(p, world.ItemGrapes, GrapesPerHarvest)
	return Change{X: x, Y: y, Layer: world.Object, Tile: world.VineHarvested}, nil
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
		if s.Map.World.At(world.Object, r.at.X, r.at.Y) == world.VineHarvested {
			s.Map.World.Set(r.at.X, r.at.Y, world.Vine)
			out = append(out, Change{X: r.at.X, Y: r.at.Y, Layer: world.Object, Tile: world.Vine})
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

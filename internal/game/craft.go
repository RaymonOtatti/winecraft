package game

import (
	"errors"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

var (
	// CraftInterval limits how often a player can craft.
	CraftInterval = 250 * time.Millisecond
	CraftBurst    = 5

	ErrUnknownRecipe = errors.New("unknown recipe")
)

// Craft turns raw materials into building materials for player id. It shares
// the edit rate bucket, so a flood of craft attempts is capped like edits.
func (s *State) Craft(id uint32, rid world.RecipeID, now time.Time) error {
	p := s.players[id]
	if p == nil {
		return ErrNotAllowed
	}
	if !p.edits.Take(now, CraftInterval, CraftBurst, 1) {
		return ErrRateLimited
	}
	recipe, ok := world.RecipeByID(rid)
	if !ok {
		return ErrUnknownRecipe
	}
	if !recipe.CanCraft(p.Inv) {
		return ErrNoMaterial
	}
	recipe.Craft(p.Inv)
	s.invDirty[id] = true
	return nil
}

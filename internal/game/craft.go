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

// MissingMaterial lists the shortfall for one recipe input.
type MissingMaterial struct {
	Item  world.ItemID
	Have  int
	Need  int
}

// Craft turns raw materials into building materials for player id. It shares
// the edit rate bucket, so a flood of craft attempts is capped like edits.
func (s *State) Craft(id uint32, rid world.RecipeID, now time.Time) ([]MissingMaterial, error) {
	p := s.players[id]
	if p == nil {
		return nil, ErrNotAllowed
	}
	if !p.edits.Take(now, CraftInterval, CraftBurst, 1) {
		return nil, ErrRateLimited
	}
	recipe, ok := world.RecipeByID(rid)
	if !ok {
		return nil, ErrUnknownRecipe
	}
	raw := recipe.Missing(p.Inv)
	if len(raw) > 0 {
		missing := make([]MissingMaterial, len(raw))
		for i, m := range raw {
			missing[i] = MissingMaterial(m)
		}
		return missing, ErrNoMaterial
	}
	recipe.Craft(p.Inv)
	s.invDirty[id] = true
	return nil, nil
}

package game

import "github.com/RaymonOtatti/winecraft/internal/world"

// HotbarSlots is how many items the hotbar holds (keys 1-4).
const HotbarSlots = 4

// DefaultHotbar is a new player's hotbar: the building materials of the
// starter kit.
var DefaultHotbar = [HotbarSlots]world.ItemID{world.ItemPlanks, world.ItemFence, world.ItemStone, world.ItemCrate}

// SetHotbar sets which item each hotbar slot holds. Any known item may go
// in a slot (even one the player has none of yet); ItemNone empties it.
func (s *State) SetHotbar(id uint32, slots [HotbarSlots]world.ItemID) error {
	p := s.players[id]
	if p == nil {
		return ErrNotAllowed
	}
	for _, it := range slots {
		if int(it) >= world.NumItems() {
			return ErrNotAllowed
		}
	}
	p.Hotbar = slots
	return nil
}

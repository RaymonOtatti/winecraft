package world

// ItemID identifies something a player can carry. Building materials map to
// the tile they place; grapes are carried only (for winemaking, PLAN.md §6).
// The order is the wire format: append new items, never reorder.
type ItemID uint16

const (
	ItemNone ItemID = iota
	ItemGrapes
	ItemPlanks
	ItemFence
	ItemStone
	ItemCrate
	numItems
)

// ItemInfo describes an item.
type ItemInfo struct {
	ID     ItemID
	Name   string // shown to players (Spanish first)
	Places TileID // the tile it builds, or None
}

var items = [numItems]ItemInfo{
	ItemNone:   {ID: ItemNone, Name: "Nada"},
	ItemGrapes: {ID: ItemGrapes, Name: "Racimo de uva"},
	ItemPlanks: {ID: ItemPlanks, Name: "Tablones", Places: Planks},
	ItemFence:  {ID: ItemFence, Name: "Cerca", Places: Fence},
	ItemStone:  {ID: ItemStone, Name: "Piedra", Places: StoneWall},
	ItemCrate:  {ID: ItemCrate, Name: "Cajón", Places: Crate},
}

// ItemDef returns an item's description; unknown ids return the zero value.
func ItemDef(id ItemID) ItemInfo {
	if int(id) >= len(items) {
		return ItemInfo{}
	}
	return items[id]
}

// NumItems is the number of defined item ids.
func NumItems() int { return len(items) }

// ItemForTile is the item that places tile t (also what breaking it gives).
func ItemForTile(t TileID) (ItemID, bool) {
	for _, it := range items {
		if it.Places != None && it.Places == t {
			return it.ID, true
		}
	}
	return ItemNone, false
}

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

	// Raw materials from the west and east zones (PLAN.md §4.5)
	ItemCanto
	ItemArena
	ItemAgua
	ItemJarilla
	ItemLimo
	ItemPoste
	ItemCana
	ItemRollizo
	ItemPaja

	// Crafted building materials (traditional Mendoza construction)
	ItemBarro
	ItemBarroPaja
	ItemAdobe
	ItemCanizo
	ItemQuincha
	ItemTecho

	// Fishing
	ItemFishingRod
	ItemFish

	// Winemaking story (B8)
	ItemWorkbench
	ItemCellar
	ItemPress
	ItemBarrel
	ItemYeast
	ItemWine
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
	ItemStone:  {ID: ItemStone, Name: "Cimiento de piedra", Places: StoneWall},
	ItemCrate:  {ID: ItemCrate, Name: "Cajón", Places: Crate},

	ItemCanto:   {ID: ItemCanto, Name: "Canto rodado"},
	ItemArena:   {ID: ItemArena, Name: "Arena"},
	ItemAgua:    {ID: ItemAgua, Name: "Agua de deshielo"},
	ItemJarilla: {ID: ItemJarilla, Name: "Jarilla"},
	ItemLimo:    {ID: ItemLimo, Name: "Limo"},
	ItemPoste:   {ID: ItemPoste, Name: "Poste"},
	ItemCana:    {ID: ItemCana, Name: "Caña"},
	ItemRollizo: {ID: ItemRollizo, Name: "Rollizo de álamo"},
	ItemPaja:    {ID: ItemPaja, Name: "Paja"},

	ItemBarro:     {ID: ItemBarro, Name: "Barro"},
	ItemBarroPaja: {ID: ItemBarroPaja, Name: "Barro con paja"},
	ItemAdobe:     {ID: ItemAdobe, Name: "Adobe", Places: AdobeWall},
	ItemCanizo:    {ID: ItemCanizo, Name: "Cañizo"},
	ItemQuincha:   {ID: ItemQuincha, Name: "Quincha", Places: QuinchaWall},
	ItemTecho:      {ID: ItemTecho, Name: "Techo de torta", Places: TortaRoof},
	ItemFishingRod: {ID: ItemFishingRod, Name: "Caña de pescar"},
	ItemFish:       {ID: ItemFish, Name: "Pez"},

	ItemWorkbench: {ID: ItemWorkbench, Name: "Banco de trabajo", Places: Workbench},
	ItemCellar:    {ID: ItemCellar, Name: "Bodega", Places: Cellar},
	ItemPress:     {ID: ItemPress, Name: "Prensa", Places: Press},
	ItemBarrel:    {ID: ItemBarrel, Name: "Barril", Places: Barrel},
	ItemYeast:     {ID: ItemYeast, Name: "Levadura"},
	ItemWine:      {ID: ItemWine, Name: "Vino"},
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

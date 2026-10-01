// Package world holds the tile map shared by the client and the server:
// the tile registry, 32×32 chunks with a ground and an object layer, and
// world-coordinate access across chunk borders.
package world

// TileID identifies a tile type. Ground and object tiles share one id space;
// each tile's definition says which layer it lives on.
type TileID uint16

// Layer is one of the stacked tile layers of a chunk.
type Layer uint8

const (
	Ground Layer = iota // what you stand on: grass, road, water…
	Object              // what stands on the ground: vines, walls, trees…
	NumLayers
)

// Tile ids. The order is the wire format: append new tiles, never reorder.
const (
	None TileID = iota

	// ground
	Grass
	Dirt
	Road
	Sand
	Water
	Plaza
	Bridge
	LedgeSouth // a one-way drop: you can hop down it heading south, never climb it
	Planks

	// objects
	Vine
	VineHarvested
	Poplar
	Fence
	StoneWall
	Rock
	Crate

	numTiles
)

// TileDef describes how a tile behaves.
type TileDef struct {
	ID    TileID
	Name  string // shown to players (Spanish first)
	Layer Layer
	// Walkable: for ground, you can stand on it; for objects, you can pass through it.
	Walkable  bool
	Breakable bool   // can be removed in a build zone
	Placeable bool   // can be placed from the hotbar in a build zone
	Drop      TileID // what breaking it gives back; None for nothing
}

var defs = [numTiles]TileDef{
	None:          {ID: None, Name: "Vacío", Layer: Ground},
	Grass:         {ID: Grass, Name: "Pasto", Layer: Ground, Walkable: true},
	Dirt:          {ID: Dirt, Name: "Tierra", Layer: Ground, Walkable: true},
	Road:          {ID: Road, Name: "Camino de tierra", Layer: Ground, Walkable: true},
	Sand:          {ID: Sand, Name: "Arena", Layer: Ground, Walkable: true},
	Water:         {ID: Water, Name: "Agua", Layer: Ground},
	Plaza:         {ID: Plaza, Name: "Plaza", Layer: Ground, Walkable: true},
	Bridge:        {ID: Bridge, Name: "Puente", Layer: Ground, Walkable: true},
	LedgeSouth:    {ID: LedgeSouth, Name: "Desnivel", Layer: Ground, Walkable: true},
	Planks:        {ID: Planks, Name: "Tablones", Layer: Ground, Walkable: true, Breakable: true, Placeable: true, Drop: Planks},
	Vine:          {ID: Vine, Name: "Vid", Layer: Object},
	VineHarvested: {ID: VineHarvested, Name: "Vid cosechada", Layer: Object},
	Poplar:        {ID: Poplar, Name: "Álamo", Layer: Object},
	Fence:         {ID: Fence, Name: "Cerca", Layer: Object, Breakable: true, Placeable: true, Drop: Fence},
	StoneWall:     {ID: StoneWall, Name: "Pared de piedra", Layer: Object, Breakable: true, Placeable: true, Drop: StoneWall},
	Rock:          {ID: Rock, Name: "Roca", Layer: Object, Breakable: true, Drop: StoneWall},
	Crate:         {ID: Crate, Name: "Cajón", Layer: Object, Breakable: true, Placeable: true, Drop: Crate},
}

// Def returns the definition of a tile. Unknown ids return the zero TileDef,
// which is not walkable, breakable or placeable.
func Def(id TileID) TileDef {
	if int(id) >= len(defs) {
		return TileDef{}
	}
	return defs[id]
}

// NumTiles is the number of defined tile ids.
func NumTiles() int { return len(defs) }

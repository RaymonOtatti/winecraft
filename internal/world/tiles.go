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

	// West zone ground: the Andean front (PLAN.md §4.5)
	Gravel // alluvial fan gravel
	Snow   // glacier ice: impassable

	// East zone ground: the Huayquerías badlands
	Silt  // reddish silt
	Cliff // sandstone walls: impassable

	// Gatherable resources, each followed by its spent state
	Boulder      // granite canto rodado on the fans
	BoulderGone  //
	SandBank     // river sand
	SandBankDug  //
	Spring       // meltwater spring below the glacier
	SpringLow    //
	Jarilla      // Larrea divaricata, 980–1,600 m
	JarillaCut   //
	SiltBank     // Huayquerías silt
	SiltDug      //
	Algarrobo    // posts and beams
	AlgarroboCut //
	Chanar       // posts for quincha frames
	ChanarCut    //
	CaneStand    // caña de Castilla along the canals
	CaneCut      //
	TimberPoplar // álamo planted for timber along canals
	PoplarStump  //
	Cortadera    // fiber for daub
	CortaderaCut //

	// Crafted building pieces (traditional Mendoza construction)
	AdobeWall
	QuinchaWall
	TortaRoof

	numTiles
)

// TileDef describes how a tile behaves.
type TileDef struct {
	ID    TileID
	Name  string // shown to players (Spanish first)
	Layer Layer
	// Walkable: for ground, you can stand on it; for objects, you can pass through it.
	Walkable  bool
	Breakable bool   // can be removed in the build zone
	Placeable bool   // can be placed from the hotbar in the build zone
	Drop      ItemID // what breaking it gives; ItemNone for nothing

	// Gathering (Use on it, anywhere): it gives GatherN of Gather, turns into
	// Spent, and Spent grows back into it after a while (RegrowsTo). Gathered
	// things are never breakable: a vineyard or a quarry is never used up.
	Gather    ItemID
	GatherN   int
	Spent     TileID
	RegrowsTo TileID
}

// gatherable describes a resource and its spent state together.
func gatherable(id, spent TileID, name, spentName string, it ItemID, n int) (TileDef, TileDef) {
	return TileDef{ID: id, Name: name, Layer: Object, Gather: it, GatherN: n, Spent: spent},
		TileDef{ID: spent, Name: spentName, Layer: Object, RegrowsTo: id}
}

var defs = func() [numTiles]TileDef {
	d := [numTiles]TileDef{
		None:          {ID: None, Name: "Vacío", Layer: Ground},
		Grass:         {ID: Grass, Name: "Pasto", Layer: Ground, Walkable: true},
		Dirt:          {ID: Dirt, Name: "Tierra", Layer: Ground, Walkable: true},
		Road:          {ID: Road, Name: "Camino de tierra", Layer: Ground, Walkable: true},
		Sand:          {ID: Sand, Name: "Arena", Layer: Ground, Walkable: true},
		Water:         {ID: Water, Name: "Agua", Layer: Ground},
		Plaza:         {ID: Plaza, Name: "Plaza", Layer: Ground, Walkable: true},
		Bridge:        {ID: Bridge, Name: "Puente", Layer: Ground, Walkable: true},
		LedgeSouth:    {ID: LedgeSouth, Name: "Desnivel", Layer: Ground, Walkable: true},
		Planks:        {ID: Planks, Name: "Tablones", Layer: Ground, Walkable: true, Breakable: true, Placeable: true, Drop: ItemPlanks},
		Vine:          {ID: Vine, Name: "Vid", Layer: Object, Gather: ItemGrapes, GatherN: 2, Spent: VineHarvested},
		VineHarvested: {ID: VineHarvested, Name: "Vid cosechada", Layer: Object, RegrowsTo: Vine},
		Poplar:        {ID: Poplar, Name: "Álamo", Layer: Object},
		Fence:         {ID: Fence, Name: "Cerca", Layer: Object, Breakable: true, Placeable: true, Drop: ItemFence},
		StoneWall:     {ID: StoneWall, Name: "Cimiento de piedra", Layer: Object, Breakable: true, Placeable: true, Drop: ItemStone},
		Rock:          {ID: Rock, Name: "Roca", Layer: Object, Breakable: true, Drop: ItemCanto},
		Crate:         {ID: Crate, Name: "Cajón", Layer: Object, Breakable: true, Placeable: true, Drop: ItemCrate},

		Gravel: {ID: Gravel, Name: "Ripio", Layer: Ground, Walkable: true},
		Snow:   {ID: Snow, Name: "Hielo del glaciar", Layer: Ground},
		Silt:   {ID: Silt, Name: "Limo rojo", Layer: Ground, Walkable: true},
		Cliff:  {ID: Cliff, Name: "Barranca de arenisca", Layer: Ground},

		AdobeWall:   {ID: AdobeWall, Name: "Pared de adobe", Layer: Object, Breakable: true, Placeable: true, Drop: ItemAdobe},
		QuinchaWall: {ID: QuinchaWall, Name: "Pared de quincha", Layer: Object, Breakable: true, Placeable: true, Drop: ItemQuincha},
		TortaRoof:   {ID: TortaRoof, Name: "Techo de torta", Layer: Object, Breakable: true, Placeable: true, Drop: ItemTecho},
	}
	for _, g := range []struct {
		id, spent       TileID
		name, spentName string
		it              ItemID
		n               int
	}{
		{Boulder, BoulderGone, "Canto rodado", "Pedregullo", ItemCanto, 1},
		{SandBank, SandBankDug, "Banco de arena", "Arena removida", ItemArena, 2},
		{Spring, SpringLow, "Vertiente de deshielo", "Vertiente baja", ItemAgua, 1},
		{Jarilla, JarillaCut, "Jarilla", "Jarilla cortada", ItemJarilla, 2},
		{SiltBank, SiltDug, "Barranca de limo", "Limo extraído", ItemLimo, 2},
		{Algarrobo, AlgarroboCut, "Algarrobo", "Algarrobo podado", ItemPoste, 1},
		{Chanar, ChanarCut, "Chañar", "Chañar podado", ItemPoste, 1},
		{CaneStand, CaneCut, "Cañaveral", "Cañaveral cortado", ItemCana, 3},
		{TimberPoplar, PoplarStump, "Álamo de corte", "Tocón de álamo", ItemRollizo, 1},
		{Cortadera, CortaderaCut, "Cortadera", "Cortadera cortada", ItemPaja, 2},
	} {
		d[g.id], d[g.spent] = gatherable(g.id, g.spent, g.name, g.spentName, g.it, g.n)
	}
	return d
}()

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

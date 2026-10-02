package world

import "math/rand/v2"

// The two gathering zones beside the dev valley (PLAN.md §4.5). Layout is
// schematic; what grows and lies there is real for each place.

// crossRoadY is the valley's east–west road, extended into both zones.
const crossRoadY = 60

// genWest lays out the Andean front: a gravel fan with granite cantos and
// jarilla, rock outcrops, the meltwater stream with sand banks, and the
// glacier with springs at its foot.
func genWest(w *World, rng *rand.Rand) {
	const glacier = westX0 + 8 // x ≤ this is ice
	for y := 0; y < devSize; y++ {
		for x := westX0; x < 0; x++ {
			g := Gravel
			switch {
			case x <= glacier:
				g = Snow
			case y == 0 || y == devSize-1:
				g = Cliff
			}
			w.Set(x, y, g)
		}
	}
	// rock outcrops in the higher fan, clear of the road and the stream
	for i := 0; i < 26; i++ {
		x, y := glacier+3+rng.IntN(22), 3+rng.IntN(devSize-6)
		if abs(y-crossRoadY) < 4 || (y > 64 && y < 77) {
			continue
		}
		for k := 0; k < 2+rng.IntN(4); k++ {
			w.Set(x+k%2, y+k/2, Cliff)
		}
	}
	// the meltwater stream flows from the glacier into the valley's stream
	for x := glacier + 1; x < 0; x++ {
		w.Set(x, 70, Water)
		w.Set(x, 71, Water)
		if x%5 == 0 && x > glacier+3 {
			w.Set(x, 68, SandBank)
			w.Set(x+2, 73, SandBank)
		}
	}
	// a glacial lake where the stream widens before the valley
	lakeCX, lakeCY := glacier+8, 70
	for dy := -2; dy <= 2; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if dx*dx+dy*dy > 13 {
				continue
			}
			w.Set(lakeCX+dx, lakeCY+dy, Water)
		}
	}
	// sand banks around the lake shore
	for _, p := range []Point{{lakeCX - 4, lakeCY}, {lakeCX + 4, lakeCY + 1}, {lakeCX, lakeCY - 3}} {
		w.Set(p.X, p.Y, SandBank)
	}
	// springs at the foot of the ice
	for _, y := range []int{18, 34, 48, 58, 82} {
		w.Set(glacier+1, y, Spring)
	}
	// the road up the fan
	for x := glacier + 6; x < 0; x++ {
		w.Set(x, crossRoadY, Road)
	}
	// cantos and jarilla scattered over the fan
	for y := 1; y < devSize-1; y++ {
		for x := glacier + 2; x < 0; x++ {
			if w.At(Ground, x, y) != Gravel || w.At(Object, x, y) != None || abs(y-crossRoadY) < 2 {
				continue
			}
			switch r := rng.IntN(100); {
			case r < 4:
				w.Set(x, y, Boulder)
			case r < 9 && x > -36:
				w.Set(x, y, Jarilla) // jarilla grows on the lower piedmont
			}
		}
	}
}

// genEast lays out the Huayquerías badlands (red silt, sandstone walls, silt
// banks, chañar, algarrobo, jarilla) and then the eastern oasis (a canal with
// caña, álamos planted for timber, cortadera).
func genEast(w *World, rng *rand.Rand) {
	for y := 0; y < devSize; y++ {
		for x := valleyW; x < eastX1; x++ {
			g := Silt
			switch {
			case x >= oasisX0:
				g = Grass
				if rng.IntN(100) < 8 {
					g = Dirt
				}
			case y == 0 || y == devSize-1:
				g = Cliff
			}
			w.Set(x, y, g)
			if x >= oasisX0 && (y == 0 || y == devSize-1 || x == eastX1-1) {
				w.Set(x, y, Poplar)
			}
		}
	}
	// sandstone walls cut by water, with silt banks at their feet
	for i := 0; i < 34; i++ {
		x, y := valleyW+3+rng.IntN(oasisX0-valleyW-8), 3+rng.IntN(devSize-6)
		if abs(y-crossRoadY) < 4 {
			continue
		}
		horizontal, n := rng.IntN(2) == 0, 3+rng.IntN(6)
		for k := 0; k < n; k++ {
			cx, cy := x, y+k
			if horizontal {
				cx, cy = x+k, y
			}
			w.Set(cx, cy, Cliff)
			if k%2 == 0 {
				bx, by := cx+1, cy
				if horizontal {
					bx, by = cx, cy+1
				}
				if w.At(Ground, bx, by) == Silt && w.At(Object, bx, by) == None {
					w.Set(bx, by, SiltBank)
				}
			}
		}
	}
	// silt banks and trees beside the road, so the first trip finds them
	for x := valleyW + 4; x < oasisX0-2; x += 6 {
		w.Set(x, crossRoadY-2, SiltBank)
		w.Set(x+3, crossRoadY+2, Chanar)
		w.Set(x+1, crossRoadY+3, Algarrobo)
	}
	// sparse Monte plants on the badlands
	for y := 1; y < devSize-1; y++ {
		for x := valleyW; x < oasisX0; x++ {
			if w.At(Ground, x, y) != Silt || w.At(Object, x, y) != None || abs(y-crossRoadY) < 2 {
				continue
			}
			switch r := rng.IntN(100); {
			case r < 2:
				w.Set(x, y, Chanar)
			case r < 4:
				w.Set(x, y, Algarrobo)
			case r < 6:
				w.Set(x, y, Jarilla)
			}
		}
	}
	// the oasis: an irrigation canal with caña on its banks and álamos in rows
	const canal = 147
	for y := 1; y < devSize-1; y++ {
		w.Set(canal, y, Water)
		if y%4 == 0 && abs(y-crossRoadY) > 2 {
			w.Set(canal-1, y, CaneStand)
			w.Set(canal+1, y, CaneStand)
		}
		if y%2 == 1 && abs(y-crossRoadY) > 2 {
			w.Set(oasisX0+3, y, TimberPoplar)
			w.Set(eastX1-4, y, TimberPoplar)
		}
	}
	w.Set(canal, crossRoadY, Bridge)
	// a reservoir pond south-east of the canal, for fishing and irrigation
	pondCX, pondCY := 154, 78
	for dy := -2; dy <= 2; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if dx*dx+dy*dy > 13 {
				continue
			}
			w.Set(pondCX+dx, pondCY+dy, Water)
		}
	}
	// a tiny sand/clay bank on the pond edge
	w.Set(pondCX-4, pondCY, SandBank)
	for y := 1; y < devSize-1; y++ {
		for x := oasisX0 + 4; x < eastX1-4; x++ {
			if w.At(Object, x, y) == None && w.At(Ground, x, y) != Water && abs(y-crossRoadY) > 1 && rng.IntN(100) < 3 {
				w.Set(x, y, Cortadera)
			}
		}
	}
	// the road east, through the badlands to the oasis
	for x := valleyW; x < eastX1-1; x++ {
		if w.At(Ground, x, crossRoadY) == Water {
			continue
		}
		w.Set(x, crossRoadY, Road)
		w.Clear(Object, x, crossRoadY)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

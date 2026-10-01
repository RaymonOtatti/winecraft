package world

import "math/rand/v2"

// Point is a tile coordinate.
type Point struct{ X, Y int }

// Rect is an axis-aligned tile rectangle: [X, X+W) × [Y, Y+H).
type Rect struct{ X, Y, W, H int }

// Contains reports whether (x, y) lies inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// DevMap is the placeholder world used until the real Valle de Uco pipeline
// (PLAN.md Phase 2) exists. Its layout is invented and makes no claim about
// any real place.
type DevMap struct {
	World     *World
	Bounds    Rect
	Spawn     Point
	BuildZone Rect // where players may build: the whole valley inside the border
}

const devSize = 96

// GenerateDevMap builds the dev map. The layout is fixed; the seed only moves
// scattered details (ground speckles, rocks, trees), so the same seed always
// gives the same bytes.
func GenerateDevMap(seed uint64) *DevMap {
	w := New()
	rng := rand.New(rand.NewPCG(seed, 0x5eed_c0de))
	m := &DevMap{
		World:     w,
		Bounds:    Rect{westX0, 0, eastX1 - westX0, devSize},
		BuildZone: Rect{1, 1, devSize - 2, devSize - 2},
	}

	// Base ground: grass with seeded patches of dirt and sand.
	for y := 0; y < devSize; y++ {
		for x := 0; x < devSize; x++ {
			g := Grass
			switch r := rng.IntN(100); {
			case r < 6:
				g = Dirt
			case r < 8:
				g = Sand
			}
			w.Set(x, y, g)
		}
	}

	// A windbreak of poplars around the edge keeps players on the map.
	for i := 0; i < devSize; i++ {
		w.Set(i, 0, Poplar)
		w.Set(i, devSize-1, Poplar)
		w.Set(0, i, Poplar)
		w.Set(devSize-1, i, Poplar)
	}

	// A ledge band across the north: hop down heading south, come back up by the road.
	for x := 1; x < devSize-1; x++ {
		w.Set(x, 30, LedgeSouth)
	}

	// Vineyards north of the ledge: a vine row every other line, open at both ends.
	for _, block := range []Rect{{6, 6, 35, 21}, {54, 6, 36, 21}} {
		for y := block.Y; y < block.Y+block.H; y += 2 {
			for x := block.X; x < block.X+block.W; x++ {
				w.Set(x, y, Dirt)
				w.Set(x, y, Vine)
			}
		}
	}

	// A stream across the south, crossed by two bridges.
	for x := 1; x < devSize-1; x++ {
		w.Set(x, 70, Water)
		w.Set(x, 71, Water)
	}

	// The main north–south road with a cross road; it cuts through the ledge
	// band and bridges the stream.
	for y := 1; y < devSize-1; y++ {
		w.Set(46, y, Road)
		w.Set(47, y, Road)
	}
	for x := 8; x < 88; x++ {
		w.Set(x, 60, Road)
	}
	for _, p := range []Point{{46, 70}, {47, 70}, {46, 71}, {47, 71}, {72, 70}, {72, 71}} {
		w.Set(p.X, p.Y, Bridge)
	}

	// A small plaza on the road: the spawn point.
	for y := 43; y < 51; y++ {
		for x := 42; x < 52; x++ {
			w.Set(x, y, Plaza)
		}
	}
	m.Spawn = Point{47, 47}

	// A fenced yard with a gate on its east side and a lane to the main road
	// (it was the only build zone in the first test; now it is just a place).
	sb := Rect{10, 77, 24, 13}
	for y := sb.Y - 1; y <= sb.Y+sb.H; y++ {
		for x := sb.X - 1; x <= sb.X+sb.W; x++ {
			w.Set(x, y, Dirt)
			edge := x == sb.X-1 || x == sb.X+sb.W || y == sb.Y-1 || y == sb.Y+sb.H
			if edge {
				w.Set(x, y, Fence)
			}
		}
	}
	gateY := sb.Y + sb.H/2
	w.Clear(Object, sb.X+sb.W, gateY)
	for x := sb.X + sb.W; x < 46; x++ {
		w.Set(x, gateY, Road)
	}
	// rocks inside the yard to break for stone
	for i := 0; i < 6; i++ {
		w.Set(sb.X+2+rng.IntN(sb.W-4), sb.Y+2+rng.IntN(sb.H-4), Rock)
	}

	// Seeded rocks and poplar clumps in the south-east field, kept off roads.
	for i := 0; i < 40; i++ {
		x, y := 56+rng.IntN(36), 74+rng.IntN(20)
		if w.At(Ground, x, y) == Road || w.At(Ground, x-1, y) == Road || w.At(Ground, x+1, y) == Road {
			continue
		}
		if rng.IntN(2) == 0 {
			w.Set(x, y, Rock)
		} else {
			w.Set(x, y, Poplar)
		}
	}
	// The zones beside the valley, and the gaps in its poplar border where
	// the cross road and the meltwater stream pass through.
	genWest(w, rng)
	genEast(w, rng)
	for x := westX0 + 14; x < 8; x++ {
		w.Set(x, crossRoadY, Road)
	}
	for x := 87; x < valleyW; x++ {
		w.Set(x, crossRoadY, Road)
	}
	for _, p := range []Point{{0, crossRoadY}, {valleyW - 1, crossRoadY}, {0, 70}, {0, 71}} {
		w.Clear(Object, p.X, p.Y)
	}
	w.Set(0, 70, Water)
	w.Set(0, 71, Water)
	return m
}

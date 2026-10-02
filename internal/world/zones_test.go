package world

import "testing"

func TestZonesAreNamedByTheirRealPlaces(t *testing.T) {
	cases := map[Point]string{
		{-30, 50}: ZoneWest,
		{10, 50}:  ZoneValley,
		{110, 50}: ZoneBadlands,
		{150, 50}: ZoneOasis,
	}
	for p, want := range cases {
		if got := ZoneAt(p.X, p.Y); got != want {
			t.Errorf("ZoneAt(%v) = %q, want %q", p, got, want)
		}
	}
}

func TestBothZonesAreReachableFromSpawn(t *testing.T) {
	m := GenerateDevMap(1)
	reach := reachable(m.World, m.Spawn, m.Bounds)
	seen := map[string]bool{}
	for p := range reach {
		seen[ZoneAt(p.X, p.Y)] = true
	}
	for _, z := range []string{ZoneWest, ZoneValley, ZoneBadlands, ZoneOasis} {
		if !seen[z] {
			t.Errorf("cannot walk from spawn into %s", z)
		}
	}
}

func TestEveryZoneOffersItsRealMaterials(t *testing.T) {
	m := GenerateDevMap(1)
	reach := reachable(m.World, m.Spawn, m.Bounds)
	want := map[string][]ItemID{
		ZoneWest:     {ItemCanto, ItemArena, ItemAgua, ItemJarilla},
		ZoneBadlands: {ItemLimo, ItemPoste},
		ZoneOasis:    {ItemCana, ItemRollizo, ItemPaja},
	}
	for zone, items := range want {
		got := map[ItemID]bool{}
		// a material counts if a tile yielding it stands next to a tile you can reach
		for p := range reach {
			if ZoneAt(p.X, p.Y) != zone {
				continue
			}
			for _, d := range []Dir{North, South, West, East} {
				dx, dy := d.Delta()
				if it := Def(m.World.At(Object, p.X+dx, p.Y+dy)).Gather; it != ItemNone {
					got[it] = true
				}
			}
		}
		for _, it := range items {
			if !got[it] {
				t.Errorf("%s: no reachable source of %s", zone, ItemDef(it).Name)
			}
		}
	}
}

func TestTheZonesAreNotBuildable(t *testing.T) {
	m := GenerateDevMap(1)
	for _, p := range []Point{{-10, 60}, {120, 60}, {150, 30}} {
		if m.BuildZone.Contains(p.X, p.Y) {
			t.Errorf("%v (%s) must not be in the build zone: building is in the valley", p, ZoneAt(p.X, p.Y))
		}
	}
	if !m.Bounds.Contains(-60, 10) || !m.Bounds.Contains(155, 90) {
		t.Fatalf("bounds %v must include both zones", m.Bounds)
	}
}

func TestBothZonesHaveFishingLakes(t *testing.T) {
	m := GenerateDevMap(1)
	// Require a real lake in each zone: a connected water cluster of at least
	// 20 tiles whose bounding box is wider and taller than a canal (so it is
	// an actual lake, not just the stream or canal).
	wantZones := map[string]bool{ZoneWest: true, ZoneOasis: true}
	for z := range wantZones {
		found := false
		seen := map[Point]bool{}
		for y := 0; y < devSize && !found; y++ {
			for x := westX0; x < eastX1 && !found; x++ {
				if ZoneAt(x, y) != z || m.World.At(Ground, x, y) != Water || seen[Point{x, y}] {
					continue
				}
				size, box := floodWaterBox(m.World, x, y, seen)
				if size >= 20 && box.W > 2 && box.H > 2 {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s must contain a fishing lake of at least 20 water tiles with width > 2 and height > 2", z)
		}
	}
}

func floodWaterBox(w *World, x0, y0 int, seen map[Point]bool) (int, Rect) {
	n := 0
	minX, minY, maxX, maxY := x0, y0, x0, y0
	stack := []Point{{x0, y0}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[p] {
			continue
		}
		seen[p] = true
		n++
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
		for _, d := range []Dir{North, South, West, East} {
			dx, dy := d.Delta()
			q := Point{p.X + dx, p.Y + dy}
			if !seen[q] && w.At(Ground, q.X, q.Y) == Water {
				stack = append(stack, q)
			}
		}
	}
	return n, Rect{minX, minY, maxX - minX + 1, maxY - minY + 1}
}

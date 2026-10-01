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

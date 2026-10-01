package world

import "testing"

func TestDevMapIsDeterministic(t *testing.T) {
	a := GenerateDevMap(1).World.Digest()
	b := GenerateDevMap(1).World.Digest()
	if a != b {
		t.Fatal("same seed must give the same bytes")
	}
	if c := GenerateDevMap(2).World.Digest(); c == a {
		t.Fatal("a different seed should change the scattered details")
	}
}

func TestDevMapSpawnIsStandable(t *testing.T) {
	m := GenerateDevMap(1)
	if !m.World.Standable(m.Spawn.X, m.Spawn.Y) {
		t.Fatalf("spawn %v is not standable", m.Spawn)
	}
	if !m.Bounds.Contains(m.Spawn.X, m.Spawn.Y) {
		t.Fatalf("spawn %v is outside the map", m.Spawn)
	}
}

func TestDevMapKeyPlacesAreReachableFromSpawn(t *testing.T) {
	m := GenerateDevMap(1)
	reach := reachable(m.World, m.Spawn, m.Bounds)

	inBuildZone := false
	for p := range reach {
		if m.BuildZone.Contains(p.X, p.Y) {
			inBuildZone = true
			break
		}
	}
	if !inBuildZone {
		t.Fatal("the build zone must be reachable from spawn")
	}

	nextToVine := false
	for p := range reach {
		for _, d := range []Dir{North, South, West, East} {
			dx, dy := d.Delta()
			if m.World.At(Object, p.X+dx, p.Y+dy) == Vine {
				nextToVine = true
			}
		}
	}
	if !nextToVine {
		t.Fatal("a vine must be reachable for harvesting")
	}

	hopped := false
	for p := range reach {
		if s, ok := m.World.CanStep(p.X, p.Y, South); ok && s.Hop {
			hopped = true
			break
		}
	}
	if !hopped {
		t.Fatal("the ledge band must be reachable and hoppable")
	}
}

func TestDevMapStaysInsideItsBounds(t *testing.T) {
	m := GenerateDevMap(1)
	for p := range reachable(m.World, m.Spawn, m.Bounds) {
		if !m.Bounds.Contains(p.X, p.Y) {
			t.Fatalf("player can walk out of the map at %v", p)
		}
	}
}

// reachable walks the map with the real movement rules, the same ones the
// server enforces. It never leaves twice the bounds, so a leak still ends.
func reachable(w *World, start Point, bounds Rect) map[Point]bool {
	limit := Rect{bounds.X - bounds.W, bounds.Y - bounds.H, bounds.W * 3, bounds.H * 3}
	seen := map[Point]bool{start: true}
	queue := []Point{start}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range []Dir{North, South, West, East} {
			s, ok := w.CanStep(p.X, p.Y, d)
			n := Point{s.X, s.Y}
			if !ok || seen[n] || !limit.Contains(n.X, n.Y) {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return seen
}

func TestBuildZoneIsTheWholeValleyInterior(t *testing.T) {
	m := GenerateDevMap(1)
	for _, p := range []Point{
		{m.Spawn.X + 6, m.Spawn.Y},         // open ground beside the plaza
		{20, 10},                           // the north-west vineyard
		{20, 80},                           // the old fenced yard
		{80, 80},                           // the south-east field
		{1, 1}, {devSize - 2, devSize - 2}, // just inside the poplar border
	} {
		if !m.BuildZone.Contains(p.X, p.Y) {
			t.Errorf("%v should be in the build zone %v", p, m.BuildZone)
		}
	}
	for _, p := range []Point{{0, 0}, {devSize - 1, 40}, {40, devSize - 1}} {
		if m.BuildZone.Contains(p.X, p.Y) {
			t.Errorf("the border tile %v must stay outside the build zone", p)
		}
	}
}

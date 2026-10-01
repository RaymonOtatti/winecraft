package game

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestTickRevealsFogAroundPlayer(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	p, _, err := s.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	if p.Fog == nil {
		t.Fatal("new player must have a fog mask")
	}
	// Spawn is revealed by the initial join reveal.
	if !p.Fog.Seen(p.Pos.X, p.Pos.Y) {
		t.Fatal("spawn tile should be seen after join")
	}
	// Move the player far from spawn and tick; the new area opens.
	far := world.Point{X: 80, Y: 80}
	p.Pos = far
	if p.Fog.Seen(far.X, far.Y) {
		t.Fatal("tile at the new position must start hidden before it is visited")
	}
	s.Tick(t0)
	if !p.Fog.Seen(far.X, far.Y) {
		t.Fatal("tick must reveal around the player's new position")
	}
}

func TestFogSurvivesSaveAndRejoin(t *testing.T) {
	s1 := New(world.GenerateDevMap(1))
	p, _, err := s1.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	s1.Tick(t0)
	if !p.Fog.Seen(p.Pos.X, p.Pos.Y) {
		t.Fatal("expected fog after tick")
	}

	saved := s1.Save(t0)
	s2 := New(world.GenerateDevMap(1))
	if err := s2.Restore(saved); err != nil {
		t.Fatal(err)
	}
	q, _, err := s2.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	if q.Fog == nil {
		t.Fatal("rejoined player must have fog restored")
	}
	if !q.Fog.Seen(p.Pos.X, p.Pos.Y) {
		t.Fatalf("restored fog must still see the saved tile %v", p.Pos)
	}
}

func TestAnonymousPlayersGetFogButAreNotSaved(t *testing.T) {
	s := New(editMap())
	p, err := s.Join("Franco")
	if err != nil {
		t.Fatal(err)
	}
	s.Tick(t0)
	if p.Fog == nil {
		t.Fatal("anonymous player must still have fog")
	}
	saved := s.Save(t0)
	if len(saved.Profiles) != 0 {
		t.Fatalf("anonymous players must not create profiles, got %d", len(saved.Profiles))
	}
}

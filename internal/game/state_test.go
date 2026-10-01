package game

import (
	"strings"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestJoinPlacesPlayersAtSpawnWithUniqueIDs(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	a, err := s.Join("Franco")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Join("Raymon")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.ID == 0 || b.ID == 0 {
		t.Fatalf("ids must be unique and non-zero: %d, %d", a.ID, b.ID)
	}
	spawn := s.Map.Spawn
	if a.Pos != spawn || b.Pos != spawn {
		t.Fatalf("players must start at spawn %v: got %v and %v", spawn, a.Pos, b.Pos)
	}
	if len(s.Players()) != 2 {
		t.Fatalf("Players() = %d, want 2", len(s.Players()))
	}
}

func TestJoinRespectsMaxPlayers(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	s.MaxPlayers = 2
	s.Join("a")
	s.Join("b")
	if _, err := s.Join("c"); err != ErrFull {
		t.Fatalf("third join = %v, want ErrFull", err)
	}
}

func TestLeaveRemovesThePlayer(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	p, _ := s.Join("Franco")
	s.Leave(p.ID)
	if len(s.Players()) != 0 || s.Player(p.ID) != nil {
		t.Fatal("player still present after Leave")
	}
	s.Leave(p.ID) // leaving twice is harmless
}

func TestDirtyPlayersAreReportedOnce(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	p, _ := s.Join("Franco")
	d := s.TakeDirty()
	if len(d) != 1 || d[0].ID != p.ID {
		t.Fatalf("TakeDirty after join = %v, want the new player", d)
	}
	if len(s.TakeDirty()) != 0 {
		t.Fatal("TakeDirty must clear the dirty set")
	}
}

func TestCleanName(t *testing.T) {
	ok := map[string]string{
		"Franco":       "Franco",
		"  Raymón  ":   "Raymón",
		"Ana_B-2.0":    "Ana_B-2.0",
		"Doña Ñandú":   "Doña Ñandú",
		"vino   tinto": "vino tinto",
	}
	for in, want := range ok {
		got, err := CleanName(in)
		if err != nil || got != want {
			t.Errorf("CleanName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	bad := []string{"", "   ", "a\x00b", "<script>", "tab\there", strings.Repeat("x", 25), "emoji🍷"}
	for _, in := range bad {
		if _, err := CleanName(in); err == nil {
			t.Errorf("CleanName(%q) accepted", in)
		}
	}
}

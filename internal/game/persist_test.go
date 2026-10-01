package game

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

const (
	tokA = "0123456789abcdef0123456789abcdef"
	tokB = "fedcba9876543210fedcba9876543210"
)

func TestSaveAndRestoreKeepTheWorldAndThePlayers(t *testing.T) {
	s1 := New(editMap())
	p, _, err := s1.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Edit(p.ID, 4, 5, world.Object, world.Fence, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Harvest(p.ID, 6, 5, t0); err != nil {
		t.Fatal(err)
	}
	if !s1.Move(p.ID, world.North, 1, t0) { // (5,5) → (5,4)? a rock is there: bump only
		p.Facing = world.North
	}
	wantInv, wantPos := copyInv(p.Inv), p.Pos

	saved := s1.Save(t0)
	s2 := New(editMap())
	if err := s2.Restore(saved); err != nil {
		t.Fatal(err)
	}
	if s2.Map.World.At(world.Object, 4, 5) != world.Fence || s2.Map.World.At(world.Object, 6, 5) != world.VineHarvested {
		t.Fatal("the world's edits and harvested vines must survive a restart")
	}
	q, _, err := s2.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	if q.Pos != wantPos || !sameInv(q.Inv, wantInv) {
		t.Fatalf("returning player: pos %v inv %v, want %v %v", q.Pos, q.Inv, wantPos, wantInv)
	}
	if ch := s2.Tick(t0.Add(RegrowAfter)); len(ch) != 1 {
		t.Fatalf("the regrow timer must survive too: %+v", ch)
	}
}

func TestReconnectingWithTheSameTokenReplacesTheOldConnection(t *testing.T) {
	s := New(editMap())
	a, _, _ := s.JoinAs("Franco", tokA)
	a.Inv[world.ItemGrapes] = 9
	b, replaced, err := s.JoinAs("Franco", tokA)
	if err != nil {
		t.Fatal(err)
	}
	if replaced != a.ID || len(s.Players()) != 1 || s.Player(a.ID) != nil {
		t.Fatalf("replaced %d, online %d: the old connection must be dropped", replaced, len(s.Players()))
	}
	if b.Inv[world.ItemGrapes] != 9 {
		t.Fatal("progress carries over to the new connection")
	}
}

func TestOnlyPlayersWithAValidTokenAreSaved(t *testing.T) {
	s := New(editMap())
	s.Join("anon")
	s.JoinAs("bad", "not-a-token")
	s.JoinAs("good", tokB)
	saved := s.Save(t0)
	if len(saved.Profiles) != 1 || saved.Profiles[tokB].Name != "good" {
		t.Fatalf("profiles %v: only the valid token is kept", saved.Profiles)
	}
}

func TestSaveStoresOnlyChangedChunks(t *testing.T) {
	s := New(world.GenerateDevMap(1))
	if n := len(s.Save(t0).Chunks); n != 0 {
		t.Fatalf("a fresh map saves %d chunks, want 0 (it is regenerated from the seed)", n)
	}
	p, _ := s.Join("Franco")
	s.Map.World.Set(p.Pos.X, p.Pos.Y+20, world.Crate)
	if n := len(s.Save(t0).Chunks); n != 1 {
		t.Fatalf("one edit saves %d chunks, want 1", n)
	}
}

func TestARestoredPlayerOnABlockedTileStartsAtSpawn(t *testing.T) {
	s1 := New(editMap())
	p, _, _ := s1.JoinAs("Franco", tokA)
	s1.Move(p.ID, world.West, 1, t0) // to (4,5)
	saved := s1.Save(t0)
	s2 := New(editMap())
	s2.Restore(saved)
	s2.Map.World.Set(4, 5, world.StoneWall) // someone built over that spot meanwhile
	q, _, _ := s2.JoinAs("Franco", tokA)
	if q.Pos != s2.Map.Spawn {
		t.Fatalf("restored onto a wall at %v; must fall back to spawn", q.Pos)
	}
}

func copyInv(in map[world.ItemID]int) map[world.ItemID]int {
	out := make(map[world.ItemID]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sameInv(a, b map[world.ItemID]int) bool {
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	return true
}

package game

import (
	"errors"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestNewPlayersGetTheDefaultHotbar(t *testing.T) {
	_, p := editor(t)
	if p.Hotbar != DefaultHotbar {
		t.Fatalf("hotbar %v, want %v", p.Hotbar, DefaultHotbar)
	}
}

func TestSetHotbarAcceptsKnownItemsOnly(t *testing.T) {
	s, p := editor(t)
	want := [HotbarSlots]world.ItemID{world.ItemGrapes, world.ItemStone, world.ItemNone, world.ItemPlanks}
	if err := s.SetHotbar(p.ID, want); err != nil || p.Hotbar != want {
		t.Fatalf("SetHotbar: err %v, hotbar %v", err, p.Hotbar)
	}
	bad := [HotbarSlots]world.ItemID{world.ItemID(200)}
	if err := s.SetHotbar(p.ID, bad); !errors.Is(err, ErrNotAllowed) || p.Hotbar != want {
		t.Fatalf("an unknown item must be refused and change nothing: err %v hotbar %v", err, p.Hotbar)
	}
	if err := s.SetHotbar(999, want); !errors.Is(err, ErrNotAllowed) {
		t.Fatal("unknown player")
	}
}

func TestTheHotbarIsSavedWithTheProfile(t *testing.T) {
	s1 := New(editMap())
	p, _, _ := s1.JoinAs("Franco", tokA)
	want := [HotbarSlots]world.ItemID{world.ItemCrate, world.ItemCrate, world.ItemNone, world.ItemGrapes}
	s1.SetHotbar(p.ID, want)
	s2 := New(editMap())
	s2.Restore(s1.Save(t0))
	q, _, _ := s2.JoinAs("Franco", tokA)
	if q.Hotbar != want {
		t.Fatalf("restored hotbar %v, want %v", q.Hotbar, want)
	}
}

package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestServerHotbarFillsTheSlots(t *testing.T) {
	s := builder(t)
	want := [proto.HotbarSlots]world.ItemID{world.ItemGrapes, world.ItemNone, world.ItemStone, world.ItemCrate}
	s.Apply(&proto.Hotbar{Slots: want})
	if s.Hotbar.Slots != want {
		t.Fatalf("slots %v, want %v", s.Hotbar.Slots, want)
	}
}

func TestBuildingWithANonBuildableItemSaysSo(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemGrapes, Count: 4}}})
	if got := notified(s, func() { s.Act(ActionBuild, world.ItemGrapes) }); got != "Eso no se puede construir" {
		t.Fatalf("notice %q", got)
	}
	if got := notified(s, func() { s.Act(ActionBuild, world.ItemNone) }); got != "Ese espacio está vacío" {
		t.Fatalf("empty slot notice %q", got)
	}
}

func TestInventoryPanelAssignsItemsToSlots(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemStone, Count: 2}, {ID: world.ItemGrapes, Count: 6}}})
	var p InvPanel
	if _, ok := p.Assign(s, 0); ok {
		t.Fatal("a closed panel assigns nothing")
	}
	p.Toggle()
	items := p.Items(s)
	if len(items) != 2 || items[0] != world.ItemGrapes || items[1] != world.ItemStone {
		t.Fatalf("panel lists %v, want grapes then stone (by item id)", items)
	}
	p.Move(s, +1)
	p.Move(s, +1) // stays on the last row
	m, ok := p.Assign(s, 3)
	if !ok || m.Slots[3] != world.ItemStone || s.Hotbar.Slots[3] != world.ItemStone {
		t.Fatalf("assign stone to slot 4: %+v ok=%v slots %v", m, ok, s.Hotbar.Slots)
	}
	p.Move(s, -5) // back to the first row, never above it
	if m, _ := p.Assign(s, 0); m.Slots[0] != world.ItemGrapes {
		t.Fatalf("assign grapes to slot 1: %+v", m)
	}
	if _, ok := p.Assign(s, 7); ok {
		t.Fatal("slot 8 does not exist")
	}
}

func TestEmptyBagPanelIsHarmless(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{})
	var p InvPanel
	p.Toggle()
	p.Move(s, +1)
	if _, ok := p.Assign(s, 0); ok {
		t.Fatal("nothing to assign from an empty bag")
	}
}

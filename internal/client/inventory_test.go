package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestInventoryMessageReplacesTheInventory(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemFence, Count: 4}, {ID: world.ItemGrapes, Count: 2}}})
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemGrapes, Count: 6}}})
	if s.Inv[world.ItemGrapes] != 6 || s.Inv[world.ItemFence] != 0 {
		t.Fatalf("inventory %v: each message is the whole inventory", s.Inv)
	}
}

func TestAHarvestsAVineYouFace(t *testing.T) {
	s := builder(t)
	s.World.Set(6, 5, world.Vine) // east of (5,5), which we face
	m, ok := s.Act(ActionUse, world.ItemFence)
	if it, isInteract := m.(*proto.Interact); !ok || !isInteract || it.X != 6 || it.Y != 5 {
		t.Fatalf("A facing a vine must harvest: %#v ok=%v", m, ok)
	}
}

func TestAPlacesWhenYouHaveTheMaterial(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemFence, Count: 1}}})
	m, ok := s.Act(ActionBuild, world.ItemFence)
	if e, isEdit := m.(*proto.Edit); !ok || !isEdit || e.Tile != world.Fence {
		t.Fatalf("A with a fence in the bag must place it: %#v ok=%v", m, ok)
	}
}

func TestAWithoutMaterialSaysSoAndSendsNothing(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{})
	seq := s.NoticeSeq
	if m, ok := s.Act(ActionBuild, world.ItemFence); ok {
		t.Fatalf("sent %#v without a fence in the bag", m)
	}
	if s.NoticeSeq == seq || s.Notice != "Te faltan materiales" {
		t.Fatalf("notice %q: the player must be told why", s.Notice)
	}
}

func TestHotbarCountsComeFromTheInventory(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemStone, Count: 7}}})
	if got := s.CountFor(world.ItemStone); got != 7 {
		t.Fatalf("stone wall slot shows %d, want the 7 stones", got)
	}
	if got := s.CountFor(world.ItemCrate); got != 0 {
		t.Fatalf("crate slot shows %d, want 0", got)
	}
}

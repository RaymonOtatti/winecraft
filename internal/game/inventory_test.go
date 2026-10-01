package game

import (
	"errors"
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestNewPlayersGetTheStarterKit(t *testing.T) {
	_, p := editor(t)
	for it, n := range StarterKit {
		if p.Inv[it] != n {
			t.Errorf("%s: %d, want %d", world.ItemDef(it).Name, p.Inv[it], n)
		}
	}
}

func TestBuildingCostsMaterialsAndBreakingGivesThemBack(t *testing.T) {
	s, p := editor(t)
	p.Inv[world.ItemFence] = 1
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.Fence, t0); err != nil {
		t.Fatal(err)
	}
	if p.Inv[world.ItemFence] != 0 {
		t.Fatalf("placing must use the fence: have %d", p.Inv[world.ItemFence])
	}
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.None, t0); err != nil || p.Inv[world.ItemFence] != 1 {
		t.Fatalf("breaking it gives it back: err %v, have %d", err, p.Inv[world.ItemFence])
	}
	cantos := p.Inv[world.ItemCanto]
	if _, err := s.Edit(p.ID, 5, 4, world.Object, world.None, t0); err != nil || p.Inv[world.ItemCanto] != cantos+1 {
		t.Fatalf("a rock breaks into one canto: err %v, %d → %d", err, cantos, p.Inv[world.ItemCanto])
	}
}

func TestNoMaterialNoBuilding(t *testing.T) {
	s, p := editor(t)
	p.Inv[world.ItemCrate] = 0
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.Crate, t0); !errors.Is(err, ErrNoMaterial) {
		t.Fatalf("err = %v, want ErrNoMaterial", err)
	}
	if s.Map.World.At(world.Object, 4, 5) != world.None {
		t.Fatal("nothing may be placed without the material")
	}
}

func TestHarvestKeepsTheVineAndGivesGrapes(t *testing.T) {
	s, p := editor(t)
	ch, err := s.Harvest(p.ID, 6, 5, t0)
	if err != nil {
		t.Fatalf("harvesting the vine next to you: %v", err)
	}
	if ch != (Change{X: 6, Y: 5, Layer: world.Object, Tile: world.VineHarvested}) || s.Map.World.At(world.Object, 6, 5) != world.VineHarvested {
		t.Fatalf("the vine must stay, harvested: change %+v", ch)
	}
	if p.Inv[world.ItemGrapes] != GrapesPerHarvest {
		t.Fatalf("grapes: %d, want %d", p.Inv[world.ItemGrapes], GrapesPerHarvest)
	}
	if _, err := s.Harvest(p.ID, 6, 5, t0.Add(time.Second)); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("a harvested vine has no grapes left: err %v", err)
	}
	if p.Inv[world.ItemGrapes] != GrapesPerHarvest {
		t.Fatal("no double harvest")
	}
}

func TestHarvestNeedsAVineWithinReach(t *testing.T) {
	s, p := editor(t)
	if _, err := s.Harvest(p.ID, 5, 4, t0); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("a rock is not a vine: %v", err)
	}
	s.Map.World.Set(8, 5, world.Vine)
	if _, err := s.Harvest(p.ID, 8, 5, t0); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("a vine three tiles away is out of reach: %v", err)
	}
}

func TestHarvestPoplarGivesWoodAndRegrows(t *testing.T) {
	s, p := editor(t)
	// The editor map places a Poplar at (5,6), one tile south of spawn.
	ch, err := s.Harvest(p.ID, 5, 6, t0)
	if err != nil {
		t.Fatalf("harvesting the poplar next to you: %v", err)
	}
	if ch != (Change{X: 5, Y: 6, Layer: world.Object, Tile: world.PoplarStump}) || s.Map.World.At(world.Object, 5, 6) != world.PoplarStump {
		t.Fatalf("poplar must become a stump: change %+v", ch)
	}
	if p.Inv[world.ItemRollizo] != 1 {
		t.Fatalf("rollizo: %d, want 1", p.Inv[world.ItemRollizo])
	}
	if _, err := s.Harvest(p.ID, 5, 6, t0.Add(time.Second)); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("a harvested poplar has no wood left: err %v", err)
	}
	regrow := s.Tick(t0.Add(RegrowAfter))
	found := false
	for _, c := range regrow {
		if c == (Change{X: 5, Y: 6, Layer: world.Object, Tile: world.Poplar}) {
			found = true
		}
	}
	if !found {
		t.Fatalf("poplar must regrow, got %+v", regrow)
	}
}

func TestVinesRegrowOnATimer(t *testing.T) {
	s, p := editor(t)
	s.Harvest(p.ID, 6, 5, t0)
	if ch := s.Tick(t0.Add(RegrowAfter - time.Second)); len(ch) != 0 {
		t.Fatalf("regrew too early: %+v", ch)
	}
	ch := s.Tick(t0.Add(RegrowAfter))
	if len(ch) != 1 || ch[0] != (Change{X: 6, Y: 5, Layer: world.Object, Tile: world.Vine}) {
		t.Fatalf("regrow changes %+v", ch)
	}
	if len(s.Tick(t0.Add(2*RegrowAfter))) != 0 {
		t.Fatal("a vine regrows once")
	}
	if _, err := s.Harvest(p.ID, 6, 5, t0.Add(RegrowAfter+time.Second)); err != nil {
		t.Fatalf("a regrown vine can be harvested again: %v", err)
	}
}

func TestInventoryChangesAreReportedPerPlayer(t *testing.T) {
	s, p := editor(t)
	q, _ := s.Join("Raymon")
	s.TakeInventoryChanges() // the joins themselves
	s.Harvest(p.ID, 6, 5, t0)
	got := s.TakeInventoryChanges()
	if len(got) != 1 || got[0] != p.ID {
		t.Fatalf("changed inventories %v, want only %d (not %d)", got, p.ID, q.ID)
	}
	if len(s.TakeInventoryChanges()) != 0 {
		t.Fatal("TakeInventoryChanges must clear")
	}
}

func TestInventoryListIsSortedAndSkipsEmpty(t *testing.T) {
	_, p := editor(t)
	p.Inv[world.ItemCrate] = 0
	list := p.Inventory()
	for i, st := range list {
		if st.Count <= 0 {
			t.Fatalf("empty stack listed: %+v", st)
		}
		if i > 0 && list[i-1].Item >= st.Item {
			t.Fatal("stacks must be sorted by item id")
		}
	}
}

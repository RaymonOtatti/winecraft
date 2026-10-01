package world

import "testing"

func TestItemRegistryIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for i, d := range items {
		if d.ID != ItemID(i) || d.Name == "" || seen[d.Name] {
			t.Fatalf("items[%d] = %+v: ids must match the index and names be unique", i, d)
		}
		seen[d.Name] = true
		if d.Places != None && !Def(d.Places).Placeable {
			t.Errorf("item %s places %s, which is not placeable", d.Name, Def(d.Places).Name)
		}
	}
}

func TestEveryPlaceableTileHasAnItem(t *testing.T) {
	for id := 1; id < NumTiles(); id++ {
		tile := TileID(id)
		if !Def(tile).Placeable {
			continue
		}
		it, ok := ItemForTile(tile)
		if !ok || ItemDef(it).Places != tile {
			t.Errorf("placeable %s has no item that places it", Def(tile).Name)
		}
	}
}

func TestBreakableTilesDropRealItems(t *testing.T) {
	for id := 1; id < NumTiles(); id++ {
		d := Def(TileID(id))
		if !d.Breakable {
			continue
		}
		if d.Drop == ItemNone || int(d.Drop) >= NumItems() {
			t.Errorf("breaking %s drops %d, which is no item", d.Name, d.Drop)
		}
		if d.Placeable && d.Drop != func() ItemID { it, _ := ItemForTile(TileID(id)); return it }() {
			t.Errorf("breaking a placed %s must give back the item that placed it", d.Name)
		}
	}
	if Def(Rock).Drop != ItemCanto {
		t.Error("a rock breaks into a granite canto")
	}
}

func TestGrapesAreAnItemNotATile(t *testing.T) {
	if ItemDef(ItemGrapes).Places != None {
		t.Fatal("grapes are carried, never placed")
	}
}

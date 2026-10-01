package art

import (
	"bytes"
	"image"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestAtlasHasATileForEveryID(t *testing.T) {
	a := Atlas()
	want := image.Rect(0, 0, AtlasCols*Tile, ((world.NumTiles()+AtlasCols-1)/AtlasCols)*Tile)
	if a.Bounds() != want {
		t.Fatalf("atlas bounds %v, want %v", a.Bounds(), want)
	}
	for id := 1; id < world.NumTiles(); id++ {
		if opaquePixels(a, TileRect(world.TileID(id))) == 0 {
			t.Errorf("tile %d (%s) is fully transparent", id, world.Def(world.TileID(id)).Name)
		}
	}
	if opaquePixels(a, TileRect(world.None)) != 0 {
		t.Error("None must be fully transparent")
	}
}

func TestGroundTilesAreOpaqueAndObjectsAreNot(t *testing.T) {
	a := Atlas()
	full := Tile * Tile
	for id := 1; id < world.NumTiles(); id++ {
		tid := world.TileID(id)
		n := opaquePixels(a, TileRect(tid))
		switch world.Def(tid).Layer {
		case world.Ground:
			if n != full {
				t.Errorf("ground tile %s has %d/%d opaque pixels; ground must cover its cell", world.Def(tid).Name, n, full)
			}
		case world.Object:
			if tid != world.StoneWall && tid != world.Crate && n == full {
				t.Errorf("object %s covers its whole cell; it should let the ground show through", world.Def(tid).Name)
			}
		}
	}
}

func TestTilesAreDistinct(t *testing.T) {
	a := Atlas()
	seen := map[string]world.TileID{}
	for id := 1; id < world.NumTiles(); id++ {
		key := string(pixels(a, TileRect(world.TileID(id))))
		if prev, dup := seen[key]; dup {
			t.Errorf("tiles %s and %s look identical", world.Def(prev).Name, world.Def(world.TileID(id)).Name)
		}
		seen[key] = world.TileID(id)
	}
}

func TestAtlasIsDeterministic(t *testing.T) {
	if !bytes.Equal(Atlas().Pix, Atlas().Pix) {
		t.Fatal("the atlas must be identical on every run")
	}
}

func TestPlayerSpritesFaceEveryDirection(t *testing.T) {
	s := Player()
	if s.Bounds() != image.Rect(0, 0, 2*Tile, 4*Tile) {
		t.Fatalf("player sheet bounds %v, want 2 frames × 4 directions", s.Bounds())
	}
	seen := map[string]bool{}
	for d := 0; d < 4; d++ {
		for f := 0; f < 2; f++ {
			r := PlayerRect(world.Dir(d), f)
			if opaquePixels(s, r) == 0 {
				t.Fatalf("player frame dir=%d f=%d is empty", d, f)
			}
			seen[string(pixels(s, r))] = true
		}
	}
	if len(seen) < 6 {
		t.Fatalf("only %d distinct player frames; directions and walk frames should differ", len(seen))
	}
}

func opaquePixels(img *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.RGBAAt(x, y).A == 255 {
				n++
			}
		}
	}
	return n
}

func pixels(img *image.RGBA, r image.Rectangle) []byte {
	var out []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := img.PixOffset(r.Min.X, y)
		out = append(out, img.Pix[i:i+r.Dx()*4]...)
	}
	return out
}

func TestEveryItemHasAnIcon(t *testing.T) {
	a := Items()
	seen := map[string]world.ItemID{}
	for id := 1; id < world.NumItems(); id++ {
		r := ItemRect(world.ItemID(id))
		if opaquePixels(a, r) == 0 {
			t.Errorf("item %s has no icon", world.ItemDef(world.ItemID(id)).Name)
		}
		key := string(pixels(a, r))
		if prev, dup := seen[key]; dup {
			t.Errorf("items %s and %s share an icon", world.ItemDef(prev).Name, world.ItemDef(world.ItemID(id)).Name)
		}
		seen[key] = world.ItemID(id)
	}
	if opaquePixels(a, ItemRect(world.ItemNone)) != 0 {
		t.Error("ItemNone must be empty")
	}
}

package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
)

func TestCameraCentersOnTheTarget(t *testing.T) {
	c := CenterOn(47*art.Tile+8, 47*art.Tile+8, 384, 216)
	sx, sy := c.ToScreen(47*art.Tile+8, 47*art.Tile+8)
	if sx != 192 || sy != 108 {
		t.Fatalf("target drawn at (%v,%v), want the screen centre (192,108)", sx, sy)
	}
}

func TestVisibleTilesCoverTheScreen(t *testing.T) {
	c := CenterOn(47*art.Tile+8, 47*art.Tile+8, 384, 216)
	x0, y0, x1, y1 := c.VisibleTiles()
	// every screen pixel's tile must be inside [x0,x1)×[y0,y1)
	for _, p := range [][2]float64{{0, 0}, {383, 0}, {0, 215}, {383, 215}} {
		tx, ty := floorDiv(c.X+p[0], art.Tile), floorDiv(c.Y+p[1], art.Tile)
		if tx < x0 || tx >= x1 || ty < y0 || ty >= y1 {
			t.Errorf("screen pixel %v is tile (%d,%d), outside the visible range [%d,%d)×[%d,%d)", p, tx, ty, x0, x1, y0, y1)
		}
	}
	if w, h := x1-x0, y1-y0; w > 384/art.Tile+2 || h > 216/art.Tile+2 {
		t.Errorf("visible range %d×%d tiles is wasteful for a 24×13.5-tile screen", w, h)
	}
}

func TestVisibleTilesNearNegativeCoordinates(t *testing.T) {
	c := CenterOn(0, 0, 384, 216) // camera top-left is at negative world pixels
	x0, y0, _, _ := c.VisibleTiles()
	if x0 != -12 || y0 != -7 {
		t.Fatalf("range starts at (%d,%d), want (-12,-7): floor division, not truncation", x0, y0)
	}
}

func TestPixelScaleIsAnIntegerThatFits(t *testing.T) {
	cases := []struct{ w, h, want int }{
		{384, 216, 1},
		{768, 432, 2},
		{1920, 1080, 5},
		{1000, 500, 2}, // height limits it
		{200, 100, 1},  // never below 1
	}
	for _, c := range cases {
		if got := PixelScale(c.w, c.h); got != c.want {
			t.Errorf("PixelScale(%d,%d) = %d, want %d", c.w, c.h, got, c.want)
		}
	}
}

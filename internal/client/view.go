// Package client is the WineCraft game client: Ebitengine rendering, input
// and (later) the network connection. It runs natively for development and
// as WebAssembly in the browser.
package client

import (
	"math"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
)

// Base logical resolution: 24 × 13.5 tiles, a 16:9 cousin of the GBA's 15 × 10.
const (
	BaseW = 384
	BaseH = 216
)

// Camera is the world-pixel position of the screen's top-left corner and the
// logical screen size.
type Camera struct {
	X, Y float64
	W, H int
}

// CenterOn puts world pixel (px, py) in the middle of a w×h screen.
func CenterOn(px, py float64, w, h int) Camera {
	return Camera{X: px - float64(w)/2, Y: py - float64(h)/2, W: w, H: h}
}

// ToScreen converts world pixels to screen pixels.
func (c Camera) ToScreen(wx, wy float64) (float64, float64) {
	return wx - c.X, wy - c.Y
}

// VisibleTiles is the half-open tile range [x0,x1)×[y0,y1) that touches the
// screen. Only these tiles are drawn.
func (c Camera) VisibleTiles() (x0, y0, x1, y1 int) {
	x0, y0 = floorDiv(c.X, art.Tile), floorDiv(c.Y, art.Tile)
	x1 = floorDiv(c.X+float64(c.W)-1, art.Tile) + 1
	y1 = floorDiv(c.Y+float64(c.H)-1, art.Tile) + 1
	return
}

// PixelScale is the largest whole-number scale at which the base resolution
// fits in a w×h window, so pixels stay crisp squares (never below 1).
func PixelScale(w, h int) int {
	return max(1, min(w/BaseW, h/BaseH))
}

func floorDiv(v float64, d int) int { return int(math.Floor(v / float64(d))) }

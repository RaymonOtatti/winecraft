package client

import (
	"hash/fnv"

	"github.com/hajimehoshi/ebiten/v2"
)

// HUDCache renders a HUD layer once and reuses the image until the caller's
// generation (a hash of whatever controls the content) changes. This removes
// the per-frame vector and debug-font allocations for mostly-static HUD parts.
type HUDCache struct {
	gen  uint64
	img  *ebiten.Image
	w, h int
}

// NewHUDCache starts an empty cache.
func NewHUDCache() *HUDCache { return &HUDCache{} }

// Bounds holds the logical size the cached image was rendered for.
type Bounds struct{ W, H int }

// Bounds returns the cached image size, or zero when empty.
func (c *HUDCache) Bounds() Bounds { return Bounds{W: c.w, H: c.h} }

// Draw renders paint onto an internal image when gen changes, then draws that
// image onto screen at (0,0). The gen should be a hash of everything that
// affects paint's content; callers can use MakeGen.
func (c *HUDCache) Draw(screen *ebiten.Image, gen uint64, paint func(*ebiten.Image)) {
	w, h := screen.Size()
	if c.img == nil || c.w != w || c.h != h || c.gen != gen {
		if c.img == nil || c.w != w || c.h != h {
			c.img = ebiten.NewImage(w, h)
		} else {
			c.img.Clear()
		}
		c.w, c.h = w, h
		c.gen = gen
		paint(c.img)
	}
	screen.DrawImage(c.img, nil)
}

// HUDHash hashes a sequence of strings into a generation value.
func HUDHash(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return h.Sum64()
}

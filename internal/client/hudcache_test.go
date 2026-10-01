package client

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestHUDCacheRendersOnceAndReuses(t *testing.T) {
	h := NewHUDCache()
	screen := ebiten.NewImage(BaseW, BaseH)

	gen1 := HUDHash("hello")
	gen2 := HUDHash("world")

	// First draw renders and caches.
	h.Draw(screen, gen1, func(e *ebiten.Image) {
		drawText(e, "hello", 10, 10)
	})

	// Second draw with the same generation must return the cached image.
	hit := false
	h.Draw(screen, gen1, func(e *ebiten.Image) {
		hit = true
		drawText(e, "hello", 10, 10)
	})
	if hit {
		t.Fatal("second identical HUD draw should reuse the cache, not render again")
	}

	// A different generation forces a re-render.
	h.Draw(screen, gen2, func(e *ebiten.Image) {
		hit = true
		drawText(e, "world", 10, 10)
	})
	if !hit {
		t.Fatal("different generation must re-render")
	}
}

func TestHUDCacheBounds(t *testing.T) {
	h := NewHUDCache()
	if got := h.Bounds(); got != (Bounds{}) {
		t.Fatalf("empty cache bounds should be zero, got %+v", got)
	}
}

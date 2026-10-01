package world

import (
	"bytes"
	"testing"
)

func TestFogRevealsAroundAPoint(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 10, H: 10}
	f := NewFog(bounds)
	if f == nil {
		t.Fatal("NewFog returned nil")
	}
	f.Reveal(5, 5, 2)
	want := map[Point]bool{
		{5, 5}: true,
		{5, 3}: true,
		{3, 5}: true,
		{7, 5}: true,
		{5, 7}: true,
	}
	for p := range want {
		if !f.Seen(p.X, p.Y) {
			t.Fatalf("tile %v should be seen", p)
		}
	}
	// A tile just outside the radius must stay hidden.
	if f.Seen(5, 2) {
		t.Fatal("tile at distance 3 should not be seen")
	}
}

func TestFogRespectsBounds(t *testing.T) {
	bounds := Rect{X: -10, Y: -5, W: 20, H: 10}
	f := NewFog(bounds)
	f.Reveal(0, 0, 5)
	// Outside the bounds is never seen and does not panic.
	if f.Seen(-11, 0) {
		t.Fatal("outside bounds should not be seen")
	}
	if f.Seen(0, 5) {
		t.Fatal("outside bounds should not be seen")
	}
	// A tile inside the reveal radius and inside the bounds should work.
	if !f.Seen(5, 4) {
		t.Fatalf("tile inside radius and bounds should be seen")
	}
}

func TestFogBytesRoundTrip(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 12, H: 8}
	f := NewFog(bounds)
	f.Reveal(3, 3, 2)
	f.Reveal(9, 6, 1)
	b := f.Bytes()
	if len(b) == 0 {
		t.Fatal("Bytes must not be empty")
	}
	g := NewFog(bounds)
	g.SetBytes(b)
	for y := 0; y < bounds.H; y++ {
		for x := 0; x < bounds.W; x++ {
			if got, want := g.Seen(x, y), f.Seen(x, y); got != want {
				t.Fatalf("(%d,%d) seen=%v, want %v", x, y, got, want)
			}
		}
	}
}

func TestFogSetBytesIgnoresWrongSize(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 10, H: 10}
	f := NewFog(bounds)
	f.SetBytes([]byte{0xff, 0xff}) // too short
	if f.Seen(0, 0) || f.Seen(9, 9) {
		t.Fatal("fog must stay empty after a size mismatch")
	}
}

func TestFogBytesLengthDependsOnlyOnBounds(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 7, H: 7}
	f := NewFog(bounds)
	f.Reveal(3, 3, 2)
	want := (bounds.W*bounds.H + 7) / 8
	if got := len(f.Bytes()); got != want {
		t.Fatalf("byte length = %d, want %d", got, want)
	}
}

func TestFogDoesNotSaveWhenEmpty(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 16, H: 16}
	f := NewFog(bounds)
	if !bytes.Equal(f.Bytes(), []byte{}) {
		t.Fatalf("empty fog should serialize to empty bytes, got %d bytes", len(f.Bytes()))
	}
}

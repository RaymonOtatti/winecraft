package world

// Fog is a per-player discovered-tile mask over a fixed Rect. It is stored as
// a bitset so a 96x96 map needs 1,152 bytes; the dev world with its side zones
// needs about 2,700 bytes. A nil Fog is safe to use and reports every tile as
// unseen.
type Fog struct {
	bounds Rect
	bits   []byte
}

// NewFog creates an empty visibility mask for bounds. It returns nil when the
// bounds have zero area, which Draw code should treat as "no map yet".
func NewFog(bounds Rect) *Fog {
	if bounds.W <= 0 || bounds.H <= 0 {
		return nil
	}
	n := (bounds.W*bounds.H + 7) / 8
	return &Fog{bounds: bounds, bits: make([]byte, n)}
}

// Bounds returns the rectangle this fog covers.
func (f *Fog) Bounds() Rect { return f.bounds }

// Seen reports whether tile (x, y) has been discovered. Coordinates outside the
// bounds are always unseen.
func (f *Fog) Seen(x, y int) bool {
	if f == nil || f.bits == nil {
		return false
	}
	if !f.bounds.Contains(x, y) {
		return false
	}
	i := (y-f.bounds.Y)*f.bounds.W + (x - f.bounds.X)
	return f.bits[i/8]&(1<<(i%8)) != 0
}

// Reveal marks every tile within radius (Chebyshev distance) of (x, y) as seen.
// Tiles outside the bounds are ignored.
func (f *Fog) Reveal(x, y, radius int) {
	if f == nil || f.bits == nil {
		return
	}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			tx, ty := x+dx, y+dy
			if !f.bounds.Contains(tx, ty) {
				continue
			}
			i := (ty-f.bounds.Y)*f.bounds.W + (tx - f.bounds.X)
			f.bits[i/8] |= 1 << (i % 8)
		}
	}
}

// Empty reports whether nothing has been discovered yet.
func (f *Fog) Empty() bool {
	if f == nil {
		return true
	}
	for _, b := range f.bits {
		if b != 0 {
			return false
		}
	}
	return true
}

// Bytes returns the bitset. Empty fog returns an empty slice so omitempty can
// drop it from saves.
func (f *Fog) Bytes() []byte {
	if f == nil || f.Empty() {
		return nil
	}
	b := make([]byte, len(f.bits))
	copy(b, f.bits)
	return b
}

// SetBytes replaces the bitset with b. If the length does not match the
// expected size, the fog is left empty.
func (f *Fog) SetBytes(b []byte) {
	if f == nil {
		return
	}
	if len(b) != len(f.bits) {
		return
	}
	copy(f.bits, b)
}

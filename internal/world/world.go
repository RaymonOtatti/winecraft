package world

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"slices"
)

// ChunkSize is the width and height of a chunk in tiles. It must stay a
// power of two: ChunkOf relies on shifts and masks.
const (
	ChunkSize  = 32
	chunkShift = 5
	chunkMask  = ChunkSize - 1
)

// ChunkCoord addresses a chunk in chunk units.
type ChunkCoord struct{ X, Y int32 }

// Chunk stores its layers as flat arrays indexed by ly*ChunkSize+lx.
type Chunk struct {
	Layers [NumLayers][ChunkSize * ChunkSize]TileID
}

// World is a sparse map of chunks. Missing chunks read as None.
type World struct {
	chunks map[ChunkCoord]*Chunk
}

// New returns an empty world.
func New() *World {
	return &World{chunks: make(map[ChunkCoord]*Chunk)}
}

// ChunkOf splits a world tile coordinate into its chunk and the local
// coordinate inside it. Arithmetic shifts floor-divide, so negative
// coordinates land in the right chunk.
func ChunkOf(x, y int) (cc ChunkCoord, lx, ly int) {
	return ChunkCoord{int32(x >> chunkShift), int32(y >> chunkShift)}, x & chunkMask, y & chunkMask
}

// At returns the tile on layer l at world coordinate (x, y).
func (w *World) At(l Layer, x, y int) TileID {
	cc, lx, ly := ChunkOf(x, y)
	c := w.chunks[cc]
	if c == nil {
		return None
	}
	return c.Layers[l][ly*ChunkSize+lx]
}

// Set writes tile t at (x, y) on the layer its definition names.
func (w *World) Set(x, y int, t TileID) {
	w.put(Def(t).Layer, x, y, t)
}

// Clear empties layer l at (x, y).
func (w *World) Clear(l Layer, x, y int) {
	w.put(l, x, y, None)
}

func (w *World) put(l Layer, x, y int, t TileID) {
	cc, lx, ly := ChunkOf(x, y)
	c := w.chunks[cc]
	if c == nil {
		if t == None {
			return
		}
		c = new(Chunk)
		w.chunks[cc] = c
	}
	c.Layers[l][ly*ChunkSize+lx] = t
}

// PutChunk replaces the chunk at cc with a copy of layers (a client loading
// what the server sent).
func (w *World) PutChunk(cc ChunkCoord, layers *[NumLayers][ChunkSize * ChunkSize]TileID) {
	c := w.chunks[cc]
	if c == nil {
		c = new(Chunk)
		w.chunks[cc] = c
	}
	c.Layers = *layers
}

// Chunk returns the chunk at cc, or nil if nothing was ever written there.
func (w *World) Chunk(cc ChunkCoord) *Chunk { return w.chunks[cc] }

// ChunkCount returns how many chunks hold data.
func (w *World) ChunkCount() int { return len(w.chunks) }

// Digest hashes every chunk in a fixed order (by Y, then X), so two worlds
// with the same tiles always produce the same digest.
func (w *World) Digest() [sha256.Size]byte {
	coords := make([]ChunkCoord, 0, len(w.chunks))
	for cc := range w.chunks {
		coords = append(coords, cc)
	}
	slices.SortFunc(coords, func(a, b ChunkCoord) int {
		if a.Y != b.Y {
			return cmp.Compare(a.Y, b.Y)
		}
		return cmp.Compare(a.X, b.X)
	})
	h := sha256.New()
	var buf [2]byte
	for _, cc := range coords {
		binary.Write(h, binary.LittleEndian, cc)
		for l := range w.chunks[cc].Layers {
			for _, t := range w.chunks[cc].Layers[l] {
				binary.LittleEndian.PutUint16(buf[:], uint16(t))
				h.Write(buf[:])
			}
		}
	}
	var out [sha256.Size]byte
	h.Sum(out[:0])
	return out
}

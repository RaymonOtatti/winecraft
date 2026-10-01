// Package art draws WineCraft's placeholder pixel art in code: a 16×16 tile
// atlas and a player sprite sheet. Drawing it in code means no external
// assets and no license questions until real art arrives (PLAN.md §13).
// Everything here is plain image.RGBA, so it is testable without a window.
package art

import (
	"image"
	"image/color"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

const (
	Tile      = 16 // pixels per tile side
	AtlasCols = 8
)

// TileRect is where tile id sits in the atlas.
func TileRect(id world.TileID) image.Rectangle {
	x, y := int(id)%AtlasCols*Tile, int(id)/AtlasCols*Tile
	return image.Rect(x, y, x+Tile, y+Tile)
}

// PlayerRect is the sprite for facing d and walk frame f (0 or 1).
func PlayerRect(d world.Dir, f int) image.Rectangle {
	x, y := (f&1)*Tile, int(d)*Tile
	return image.Rect(x, y, x+Tile, y+Tile)
}

// Atlas draws every tile into one image, so the renderer can batch draws.
func Atlas() *image.RGBA {
	rows := (world.NumTiles() + AtlasCols - 1) / AtlasCols
	img := image.NewRGBA(image.Rect(0, 0, AtlasCols*Tile, rows*Tile))
	for id := 1; id < world.NumTiles(); id++ {
		tid := world.TileID(id)
		if paint, ok := painters[tid]; ok {
			paint(&cell{img: img, o: TileRect(tid).Min, salt: uint32(id)})
		}
	}
	return img
}

var (
	outline = rgb(0x2a, 0x1e, 0x1a)

	grassBase, grassDark, grassLight, grassTuft = rgb(0x9a, 0xb0, 0x52), rgb(0x7f, 0x96, 0x40), rgb(0xb6, 0xc7, 0x6a), rgb(0x62, 0x7c, 0x32)
	dirtBase, dirtDark, pebble                  = rgb(0xa8, 0x7d, 0x55), rgb(0x8a, 0x63, 0x42), rgb(0xd8, 0xcf, 0xbf)
	roadBase, roadRut, roadDust                 = rgb(0xcf, 0xb9, 0x8c), rgb(0xb2, 0x9a, 0x6e), rgb(0xe0, 0xcd, 0xa2)
	sandBase, sandDot                           = rgb(0xe6, 0xd3, 0x9e), rgb(0xcf, 0xb8, 0x80)
	waterBase, waterDeep, waterWave             = rgb(0x3d, 0x7d, 0xc4), rgb(0x2f, 0x66, 0xa8), rgb(0x8c, 0xc0, 0xec)
	stoneBase, stoneJoint, stoneLight           = rgb(0xc2, 0xba, 0xaa), rgb(0x9a, 0x92, 0x84), rgb(0xd8, 0xd1, 0xc2)
	woodBase, woodDark, woodDeep                = rgb(0xb5, 0x84, 0x50), rgb(0x8a, 0x60, 0x36), rgb(0x5a, 0x3a, 0x1e)
	leaf, leafDark, leafDry                     = rgb(0x5d, 0x8f, 0x34), rgb(0x45, 0x6e, 0x26), rgb(0x9c, 0x9a, 0x3e)
	grape, grapeLight, wire                     = rgb(0x4e, 0x22, 0x5e), rgb(0x7d, 0x3f, 0x93), rgb(0x8c, 0x8c, 0x8c)
	poplarLeaf, poplarLight, trunk              = rgb(0x4a, 0x76, 0x2c), rgb(0x6c, 0x9a, 0x3c), rgb(0x6b, 0x4a, 0x2b)
	rockBase, rockLight, rockShadow             = rgb(0x8f, 0x8a, 0x80), rgb(0xb5, 0xb0, 0xa6), rgb(0x6a, 0x65, 0x5c)
	wallBase, wallJoint                         = rgb(0x9c, 0x96, 0x8c), rgb(0x6e, 0x69, 0x60)

	skin, hatStraw, hatBand, hair = rgb(0xe8, 0xb8, 0x8e), rgb(0xc9, 0xa2, 0x5c), rgb(0x7a, 0x2e, 0x2e), rgb(0x4a, 0x32, 0x22)
	shirt, pants, boot, eye       = rgb(0x8e, 0x23, 0x3a), rgb(0x3b, 0x4a, 0x6b), rgb(0x3a, 0x2a, 0x20), rgb(0x22, 0x18, 0x14)
)

var painters = map[world.TileID]func(*cell){
	world.Grass: paintGrass,
	world.Dirt: func(c *cell) {
		c.noise(dirtBase, dirtDark, 30)
		c.speckle(pebble, 6)
	},
	world.Road: func(c *cell) {
		c.noise(roadBase, roadDust, 25)
		for y := 0; y < Tile; y++ { // two wheel ruts
			c.set(4, y, roadRut)
			c.set(11, y, roadRut)
		}
	},
	world.Sand: func(c *cell) {
		c.fill(sandBase)
		c.speckle(sandDot, 14)
	},
	world.Water: func(c *cell) {
		c.noise(waterBase, waterDeep, 30)
		for _, y := range []int{3, 9, 14} {
			for x := (y * 3) % 5; x < Tile; x += 7 {
				c.set(x, y, waterWave)
				c.set(x+1, y, waterWave)
				c.set(x+2, y-1, waterWave)
			}
		}
	},
	world.Plaza: func(c *cell) {
		c.fill(stoneBase)
		for i := 0; i < Tile; i++ {
			c.set(i, 7, stoneJoint)
			c.set(i, 15, stoneJoint)
			c.set(7, i/2, stoneJoint) // offset joints, like laid pavers
			c.set(15, 8+i/2, stoneJoint)
		}
		c.speckle(stoneLight, 5)
	},
	world.Bridge: func(c *cell) {
		c.fill(woodBase)
		for x := 3; x < Tile; x += 4 {
			for y := 0; y < Tile; y++ {
				c.set(x, y, woodDark)
			}
		}
		for x := 0; x < Tile; x++ { // rails
			c.set(x, 0, woodDeep)
			c.set(x, 1, woodDeep)
			c.set(x, 14, woodDeep)
			c.set(x, 15, woodDeep)
		}
	},
	world.LedgeSouth: func(c *cell) {
		paintGrass(c)
		for x := 0; x < Tile; x++ { // a lit lip, then the drop's face in shadow
			c.set(x, 10, grassLight)
			c.set(x, 11, dirtBase)
			for y := 12; y < Tile; y++ {
				c.set(x, y, dirtDark)
			}
		}
	},
	world.Planks: func(c *cell) {
		c.noise(woodBase, woodDark, 12)
		for x := 0; x < Tile; x += 4 {
			for y := 0; y < Tile; y++ {
				c.set(x, y, woodDark)
			}
		}
		c.set(2, 5, woodDeep)
		c.set(9, 11, woodDeep)
	},
	world.Vine:          func(c *cell) { paintVine(c, true) },
	world.VineHarvested: func(c *cell) { paintVine(c, false) },
	world.Poplar: func(c *cell) {
		for y := 12; y < Tile; y++ {
			c.set(7, y, trunk)
			c.set(8, y, trunk)
		}
		for y := 0; y < 13; y++ { // a narrow column, as Uco's windbreak poplars are
			half := 2 + min(y, 12-y)/2
			for x := 8 - half; x < 8+half; x++ {
				col := poplarLeaf
				if c.hash(x, y)%5 == 0 || x == 8-half {
					col = poplarLight
				}
				c.set(x, y, col)
			}
		}
		c.outline()
	},
	world.Fence: func(c *cell) {
		for y := 3; y < 15; y++ {
			c.set(2, y, woodDark)
			c.set(13, y, woodDark)
		}
		for x := 0; x < Tile; x++ {
			c.set(x, 5, woodBase)
			c.set(x, 6, woodDark)
			c.set(x, 10, woodBase)
			c.set(x, 11, woodDark)
		}
	},
	world.StoneWall: func(c *cell) {
		c.fill(wallBase)
		for y := 0; y < Tile; y += 4 {
			for x := 0; x < Tile; x++ {
				c.set(x, y, wallJoint)
			}
			off := (y / 4 % 2) * 4
			for yy := y; yy < y+4; yy++ {
				c.set(off, yy, wallJoint)
				c.set(off+8, yy, wallJoint)
			}
		}
	},
	world.Rock: func(c *cell) {
		for y := 4; y < 15; y++ {
			for x := 2; x < 14; x++ {
				dx, dy := float64(x)-7.5, (float64(y)-9.5)*1.2
				if dx*dx+dy*dy > 36 {
					continue
				}
				col := rockBase
				switch {
				case dx+dy < -4:
					col = rockLight
				case dx+dy > 4:
					col = rockShadow
				}
				c.set(x, y, col)
			}
		}
		c.outline()
	},
	world.Crate: func(c *cell) {
		c.fill(woodBase)
		for i := 0; i < Tile; i++ {
			c.set(i, 0, woodDeep)
			c.set(i, 15, woodDeep)
			c.set(0, i, woodDeep)
			c.set(15, i, woodDeep)
			c.set(i, i, woodDark)
			c.set(15-i, i, woodDark)
		}
	},
}

func paintGrass(c *cell) {
	c.noise(grassBase, grassDark, 28)
	c.speckle(grassLight, 6)
	for i := 0; i < 3; i++ { // a few tufts
		x, y := int(c.hash(i, 99)%14)+1, int(c.hash(99, i)%13)+2
		c.set(x, y, grassTuft)
		c.set(x-1, y-1, grassTuft)
		c.set(x+1, y-1, grassTuft)
	}
}

func paintVine(c *cell, grapes bool) {
	for x := 0; x < Tile; x++ { // the trellis wire
		c.set(x, 4, wire)
	}
	for y := 3; y < 14; y++ { // the post
		c.set(1, y, trunk)
	}
	for y := 2; y < 12; y++ {
		for x := 0; x < Tile; x++ {
			if h := c.hash(x, y); h%7 < 5 {
				col := leaf
				switch {
				case h%11 == 0:
					col = leafDark
				case !grapes && h%5 == 0:
					col = leafDry
				}
				c.set(x, y, col)
			}
		}
	}
	if grapes {
		for _, gx := range []int{4, 10} { // two hanging bunches
			for y := 7; y < 13; y++ {
				w := (13 - y) / 2
				for x := gx - w; x <= gx+w; x++ {
					col := grape
					if (x+y)%3 == 0 {
						col = grapeLight
					}
					c.set(x, y, col)
				}
			}
		}
	}
	c.outline()
}

// Player draws the sprite sheet: one row per Dir (South, North, West, East),
// two walk frames per row.
func Player() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2*Tile, 4*Tile))
	for d := world.Dir(0); d < 4; d++ {
		for f := 0; f < 2; f++ {
			paintPlayer(&cell{img: img, o: PlayerRect(d, f).Min}, d, f)
		}
	}
	return img
}

func paintPlayer(c *cell, d world.Dir, frame int) {
	c.rect(5, 1, 11, 3, hatStraw) // hat crown
	c.rect(5, 2, 11, 3, hatBand)
	c.rect(3, 3, 13, 4, hatStraw) // brim
	if d == world.North {
		c.rect(5, 4, 11, 8, hair)
	} else {
		c.rect(5, 4, 11, 8, skin)
		switch d {
		case world.South:
			c.set(6, 5, eye)
			c.set(9, 5, eye)
		case world.West:
			c.set(5, 5, eye)
			c.rect(10, 4, 11, 8, hair)
		case world.East:
			c.set(10, 5, eye)
			c.rect(5, 4, 6, 8, hair)
		}
	}
	c.rect(4, 8, 12, 12, shirt)
	c.rect(3, 8, 4, 11, skin) // hands at the sides
	c.rect(12, 8, 13, 11, skin)
	c.rect(5, 12, 11, 14, pants)
	if frame == 0 {
		c.rect(5, 14, 7, 16, boot)
		c.rect(9, 14, 11, 16, boot)
	} else { // mid-step: one foot forward, one back
		c.rect(5, 13, 7, 15, boot)
		c.rect(9, 14, 11, 16, boot)
		c.set(4, 9, shirt) // the arms swing
		c.set(12, 10, shirt)
	}
	c.outline()
}

// cell paints one 16×16 tile at origin o.
type cell struct {
	img  *image.RGBA
	o    image.Point
	salt uint32
}

func (c *cell) set(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= Tile || y >= Tile {
		return
	}
	c.img.SetRGBA(c.o.X+x, c.o.Y+y, col)
}

func (c *cell) at(x, y int) color.RGBA {
	if x < 0 || y < 0 || x >= Tile || y >= Tile {
		return color.RGBA{}
	}
	return c.img.RGBAAt(c.o.X+x, c.o.Y+y)
}

func (c *cell) fill(col color.RGBA) { c.rect(0, 0, Tile, Tile, col) }

func (c *cell) rect(x0, y0, x1, y1 int, col color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.set(x, y, col)
		}
	}
}

// noise fills with base and scatters alt over about pct percent of pixels.
func (c *cell) noise(base, alt color.RGBA, pct uint32) {
	for y := 0; y < Tile; y++ {
		for x := 0; x < Tile; x++ {
			col := base
			if c.hash(x, y)%100 < pct {
				col = alt
			}
			c.set(x, y, col)
		}
	}
}

func (c *cell) speckle(col color.RGBA, n int) {
	for i := 0; i < n; i++ {
		c.set(int(c.hash(i, 7)%Tile), int(c.hash(7, i+31)%Tile), col)
	}
}

// outline darkens transparent pixels that touch drawn ones, so objects read
// clearly against any ground.
func (c *cell) outline() {
	var edge []image.Point
	for y := 0; y < Tile; y++ {
		for x := 0; x < Tile; x++ {
			if c.at(x, y).A != 0 {
				continue
			}
			if c.at(x-1, y).A != 0 || c.at(x+1, y).A != 0 || c.at(x, y-1).A != 0 || c.at(x, y+1).A != 0 {
				edge = append(edge, image.Pt(x, y))
			}
		}
	}
	for _, p := range edge {
		c.set(p.X, p.Y, outline)
	}
}

// hash is a small integer hash: deterministic noise without a random source.
func (c *cell) hash(x, y int) uint32 {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77 ^ c.salt*0xc2b2ae3d
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	return h
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }

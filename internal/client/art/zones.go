package art

import (
	"image/color"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Art for the west and east gathering zones and the crafted building pieces
// (PLAN.md §4.5). Placeholder pixel art, drawn in code like the rest.

var (
	gravelBase, gravelDark, gravelLight = rgb(0x9a, 0x92, 0x84), rgb(0x76, 0x6f, 0x63), rgb(0xc4, 0xbd, 0xb0)
	snowBase, snowShade, snowCrack      = rgb(0xe8, 0xf1, 0xf6), rgb(0xc5, 0xd9, 0xe6), rgb(0x9f, 0xbf, 0xd4)
	siltBase, siltDark, siltLight       = rgb(0xb5, 0x6e, 0x4f), rgb(0x94, 0x55, 0x3b), rgb(0xcf, 0x8d, 0x6c)
	sandstone, sandstoneDark            = rgb(0xc9, 0x9a, 0x6e), rgb(0x9e, 0x72, 0x4c)
	granite, graniteDark, caliche       = rgb(0x8c, 0x8a, 0x86), rgb(0x5e, 0x5c, 0x58), rgb(0xf0, 0xec, 0xe2)
	jarillaLeaf, jarillaDark, flower    = rgb(0x6e, 0x8c, 0x2e), rgb(0x4e, 0x67, 0x1f), rgb(0xf2, 0xc8, 0x2c)
	algarroboLeaf, algarroboDark        = rgb(0x3f, 0x66, 0x2c), rgb(0x2c, 0x4b, 0x1f)
	chanarLeaf, chanarFruit, chanarBark = rgb(0x7a, 0x96, 0x3c), rgb(0xd9, 0x8a, 0x2b), rgb(0x9b, 0x84, 0x5a)
	cane, caneLeaf                      = rgb(0xc8, 0xc0, 0x6a), rgb(0x8f, 0xa8, 0x48)
	timberLeaf, timberLight             = rgb(0x7c, 0xa6, 0x48), rgb(0xa2, 0xc4, 0x63)
	plume, plumeStalk                   = rgb(0xf3, 0xee, 0xdc), rgb(0xa8, 0xa2, 0x6a)
	adobe, adobeDark                    = rgb(0xc8, 0x9b, 0x68), rgb(0x9a, 0x70, 0x47)
	mud, mudDark                        = rgb(0x8f, 0x6a, 0x46), rgb(0x6b, 0x4d, 0x31)
	waterSpark                          = rgb(0xd4, 0xec, 0xff)
)

func init() {
	for id, p := range map[world.TileID]func(*cell){
		world.Gravel: func(c *cell) {
			c.noise(gravelBase, gravelDark, 30)
			c.speckle(gravelLight, 10)
		},
		world.Snow: func(c *cell) {
			c.noise(snowBase, snowShade, 18)
			for x := 2; x < 14; x++ { // a crevasse
				c.set(x, 6+x/4, snowCrack)
			}
		},
		world.Silt: func(c *cell) {
			c.noise(siltBase, siltDark, 25)
			c.speckle(siltLight, 6)
		},
		world.Cliff: func(c *cell) { // layered sandstone, Huayquerías Fm
			for y := 0; y < Tile; y++ {
				col := sandstone
				if y%5 == 0 || y%5 == 3 {
					col = sandstoneDark
				}
				for x := 0; x < Tile; x++ {
					c.set(x, y, col)
				}
			}
			c.speckle(siltDark, 8)
		},
		world.Boulder: func(c *cell) {
			blob(c, 8, 9, 6, 5, granite, graniteDark, caliche)
			c.set(5, 6, caliche) // the white calcium-carbonate coat of Uco's cantos
			c.set(6, 6, caliche)
			c.set(10, 7, caliche)
			c.outline()
		},
		world.BoulderGone: func(c *cell) { pebbles(c, granite, caliche) },
		world.SandBank: func(c *cell) {
			blob(c, 8, 10, 6, 4, sandBase, sandDot, roadDust)
			c.outline()
		},
		world.SandBankDug: func(c *cell) {
			blob(c, 8, 11, 6, 3, sandBase, sandDot, sandBase)
			blob(c, 8, 11, 3, 1, sandDot, sandDot, sandDot)
		},
		world.Spring: func(c *cell) {
			blob(c, 8, 9, 6, 5, waterBase, waterDeep, waterWave)
			c.set(6, 7, waterSpark)
			c.set(10, 10, waterSpark)
			ring(c, 8, 9, 6, 5, granite)
		},
		world.SpringLow: func(c *cell) {
			blob(c, 8, 10, 3, 2, waterBase, waterDeep, waterWave)
			ring(c, 8, 10, 5, 4, granite)
		},
		world.Jarilla:    func(c *cell) { shrub(c, jarillaLeaf, jarillaDark, flower) },
		world.JarillaCut: func(c *cell) { stubs(c, jarillaDark, 5) },
		world.SiltBank: func(c *cell) {
			blob(c, 8, 9, 6, 5, siltBase, siltDark, siltLight)
			for x := 3; x < 13; x++ {
				c.set(x, 8, siltDark)
			}
			c.outline()
		},
		world.SiltDug: func(c *cell) {
			ring(c, 8, 10, 5, 3, siltDark)
			blob(c, 8, 10, 3, 2, mudDark, mudDark, mudDark)
		},
		world.Algarrobo: func(c *cell) { tree(c, 7, algarroboLeaf, algarroboDark, trunk, color.RGBA{}) },
		world.AlgarroboCut: func(c *cell) {
			tree(c, 4, algarroboLeaf, algarroboDark, trunk, color.RGBA{})
			c.set(11, 9, woodBase) // a fresh cut
		},
		world.Chanar: func(c *cell) { tree(c, 6, chanarLeaf, algarroboLeaf, chanarBark, chanarFruit) },
		world.ChanarCut: func(c *cell) {
			tree(c, 3, chanarLeaf, algarroboLeaf, chanarBark, color.RGBA{})
			c.set(5, 10, woodBase)
		},
		world.CaneStand: func(c *cell) { canes(c, 1) },
		world.CaneCut:   func(c *cell) { stubs(c, cane, 6) },
		world.TimberPoplar: func(c *cell) {
			for y := 12; y < Tile; y++ {
				c.set(7, y, trunk)
				c.set(8, y, trunk)
			}
			for y := 1; y < 13; y++ {
				half := 1 + min(y, 12-y)/3
				for x := 8 - half; x < 8+half; x++ {
					col := timberLeaf
					if c.hash(x, y)%4 == 0 {
						col = timberLight
					}
					c.set(x, y, col)
				}
			}
			c.outline()
		},
		world.PoplarStump: func(c *cell) {
			c.rect(6, 10, 10, 14, trunk)
			c.rect(6, 10, 10, 11, woodBase)
			c.outline()
		},
		world.Cortadera: func(c *cell) {
			for x := 3; x < 13; x += 2 {
				for y := 6; y < 15; y++ {
					c.set(x, y, plumeStalk)
				}
				c.rect(x-1, 2+x%3, x+2, 6+x%3, plume)
			}
			c.outline()
		},
		world.CortaderaCut: func(c *cell) { stubs(c, plumeStalk, 4) },
		world.AdobeWall:    func(c *cell) { bricks(c, adobe, adobeDark, 8, 4) },
		world.QuinchaWall: func(c *cell) {
			c.noise(mud, mudDark, 20)
			for i := 0; i < Tile; i += 4 { // the caña lattice showing through the daub
				for k := 0; k < Tile; k++ {
					if c.hash(i, k)%3 != 0 {
						c.set(i, k, cane)
					}
				}
			}
		},
		world.TortaRoof: func(c *cell) {
			c.noise(mud, mudDark, 30)
			for x := 0; x < Tile; x += 3 {
				c.set(x, 0, cane)
				c.set(x, 15, cane)
			}
			c.speckle(flower, 5) // straw in the mud
		},
	} {
		painters[id] = p
	}
	for id, p := range map[world.ItemID]func(*cell){
		world.ItemCanto: func(c *cell) {
			blob(c, 8, 9, 4, 3, granite, graniteDark, caliche)
			c.outline()
		},
		world.ItemArena: func(c *cell) {
			blob(c, 8, 11, 6, 3, sandBase, sandDot, roadDust)
			c.outline()
		},
		world.ItemAgua: func(c *cell) { // a clay cántaro of meltwater
			blob(c, 8, 10, 5, 5, siltBase, siltDark, siltLight)
			c.rect(6, 2, 10, 6, siltBase)
			c.rect(7, 2, 9, 4, waterBase)
			c.outline()
		},
		world.ItemJarilla: func(c *cell) {
			for i := 0; i < 5; i++ {
				for y := 4; y < 14; y++ {
					c.set(5+i*2+(y%3)/2, y, jarillaDark)
				}
				c.set(5+i*2, 3, flower)
			}
			c.rect(4, 9, 13, 10, trunk)
			c.outline()
		},
		world.ItemLimo: func(c *cell) {
			blob(c, 8, 10, 5, 4, siltBase, siltDark, siltLight)
			c.outline()
		},
		world.ItemPoste: func(c *cell) {
			c.rect(7, 1, 10, 15, trunk)
			c.rect(7, 1, 10, 2, woodBase)
			c.outline()
		},
		world.ItemCana: func(c *cell) {
			for i := 0; i < 3; i++ {
				for k := 0; k < 12; k++ {
					c.set(3+i*3+k/4, 2+k, cane)
				}
			}
			c.outline()
		},
		world.ItemRollizo: func(c *cell) {
			c.rect(2, 6, 14, 11, trunk)
			c.rect(12, 6, 14, 11, woodBase) // the cut end
			c.set(13, 8, woodDark)
			c.outline()
		},
		world.ItemPaja: func(c *cell) {
			for x := 3; x < 13; x++ {
				for y := 3; y < 14; y++ {
					if (x+y)%3 != 0 {
						c.set(x, y, flower)
					}
				}
			}
			c.rect(3, 8, 13, 9, trunk) // the tie
			c.outline()
		},
		world.ItemBarro: func(c *cell) {
			blob(c, 8, 10, 6, 4, mud, mudDark, mud)
			c.outline()
		},
		world.ItemBarroPaja: func(c *cell) {
			blob(c, 8, 10, 6, 4, mud, mudDark, mud)
			for i := 0; i < 6; i++ {
				c.set(4+i*2, 8+i%3, flower)
			}
			c.outline()
		},
		world.ItemCanizo: func(c *cell) {
			for y := 2; y < 14; y++ {
				for x := 2; x < 14; x++ {
					col := cane
					if (x/2+y/2)%2 == 0 {
						col = caneLeaf
					}
					c.set(x, y, col)
				}
			}
			c.outline()
		},
	} {
		itemPainters[id] = p
	}
}

// blob fills an ellipse centred on (cx, cy) with radii rx, ry, shaded
// lighter to the top-left and darker to the bottom-right.
func blob(c *cell, cx, cy, rx, ry int, base, dark, light color.RGBA) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx, dy := float64(x-cx)/float64(rx), float64(y-cy)/float64(ry)
			if dx*dx+dy*dy > 1 {
				continue
			}
			col := base
			switch {
			case dx+dy < -0.7:
				col = light
			case dx+dy > 0.7:
				col = dark
			}
			c.set(x, y, col)
		}
	}
}

// ring draws an ellipse outline (stones around a spring, a dug pit's rim).
func ring(c *cell, cx, cy, rx, ry int, col color.RGBA) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx, dy := float64(x-cx)/float64(rx), float64(y-cy)/float64(ry)
			if d := dx*dx + dy*dy; d <= 1 && d > 0.6 && c.at(x, y).A == 0 {
				c.set(x, y, col)
			}
		}
	}
}

func pebbles(c *cell, col, accent color.RGBA) {
	for i := 0; i < 5; i++ {
		x, y := 3+int(c.hash(i, 3)%10), 5+int(c.hash(3, i)%8)
		c.set(x, y, col)
		c.set(x+1, y, col)
		if i%2 == 0 {
			c.set(x, y-1, accent)
		}
	}
	c.outline()
}

func shrub(c *cell, leafCol, dark, accent color.RGBA) {
	blob(c, 8, 9, 6, 5, leafCol, dark, leafCol)
	for i := 0; i < 7; i++ {
		c.set(3+int(c.hash(i, 11)%11), 5+int(c.hash(11, i)%8), accent)
	}
	c.outline()
}

func stubs(c *cell, col color.RGBA, n int) {
	for i := 0; i < n; i++ {
		x := 3 + i*10/n
		c.rect(x, 11, x+1, 14, col)
	}
	c.outline()
}

func tree(c *cell, r int, leafCol, dark, bark, fruit color.RGBA) {
	for y := 9; y < Tile; y++ {
		c.set(7, y, bark)
		c.set(8, y, bark)
	}
	blob(c, 8, 7, r, r*2/3+1, leafCol, dark, leafCol)
	if fruit.A != 0 {
		for i := 0; i < 5; i++ {
			c.set(8-r+1+int(c.hash(i, 21)%uint32(2*r-1)), 5+int(c.hash(21, i)%5), fruit)
		}
	}
	c.outline()
}

func canes(c *cell, _ int) {
	for x := 2; x < 15; x += 3 {
		for y := 1; y < 15; y++ {
			c.set(x, y, cane)
		}
		c.set(x+1, 3+x%4, caneLeaf)
		c.set(x-1, 6+x%5, caneLeaf)
	}
	c.outline()
}

func bricks(c *cell, base, mortar color.RGBA, bw, bh int) {
	c.fill(base)
	for y := 0; y < Tile; y += bh {
		for x := 0; x < Tile; x++ {
			c.set(x, y, mortar)
		}
		off := (y / bh % 2) * (bw / 2)
		for yy := y; yy < y+bh; yy++ {
			for x := off; x < Tile; x += bw {
				c.set(x, yy, mortar)
			}
		}
	}
}

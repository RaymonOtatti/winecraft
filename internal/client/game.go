package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

var voidColor = color.RGBA{0x1b, 0x14, 0x10, 0xff} // outside the map

// Game is the Ebitengine game: it draws the world and the player.
type Game struct {
	World  *world.World
	Pos    world.Point // the local player's tile
	Facing world.Dir

	// SnapPath, when set, saves frame SnapFrame as a PNG and quits.
	SnapPath  string
	SnapFrame int

	tiles  [64]*ebiten.Image // sub-images of one atlas, so draws batch
	player [4][2]*ebiten.Image
	w, h   int
	frame  int
	done   bool
	err    error
}

// NewGame prepares the art for w, drawn at pos.
func NewGame(w *world.World, pos world.Point) *Game {
	g := &Game{World: w, Pos: pos, Facing: world.South, SnapFrame: 30, w: BaseW, h: BaseH}
	atlas := ebiten.NewImageFromImage(art.Atlas())
	for id := 1; id < world.NumTiles(); id++ {
		g.tiles[id] = atlas.SubImage(art.TileRect(world.TileID(id))).(*ebiten.Image)
	}
	sheet := ebiten.NewImageFromImage(art.Player())
	for d := world.Dir(0); d < 4; d++ {
		for f := 0; f < 2; f++ {
			g.player[d][f] = sheet.SubImage(art.PlayerRect(d, f)).(*ebiten.Image)
		}
	}
	return g
}

func (g *Game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	g.frame++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(voidColor)
	cam := CenterOn(float64(g.Pos.X*art.Tile+art.Tile/2), float64(g.Pos.Y*art.Tile+art.Tile/2), g.w, g.h)
	x0, y0, x1, y1 := cam.VisibleTiles()

	var op ebiten.DrawImageOptions
	for layer := world.Layer(0); layer < world.NumLayers; layer++ {
		for ty := y0; ty < y1; ty++ {
			for tx := x0; tx < x1; tx++ {
				id := g.World.At(layer, tx, ty)
				if id == world.None || int(id) >= len(g.tiles) || g.tiles[id] == nil {
					continue
				}
				sx, sy := cam.ToScreen(float64(tx*art.Tile), float64(ty*art.Tile))
				op.GeoM.Reset()
				op.GeoM.Translate(sx, sy)
				screen.DrawImage(g.tiles[id], &op)
			}
		}
	}

	sx, sy := cam.ToScreen(float64(g.Pos.X*art.Tile), float64(g.Pos.Y*art.Tile))
	op.GeoM.Reset()
	op.GeoM.Translate(sx, sy)
	screen.DrawImage(g.player[g.Facing][0], &op)

	if g.SnapPath != "" && g.frame >= g.SnapFrame && !g.done {
		g.err = savePNG(screen, g.SnapPath)
		g.done = true
	}
}

// Layout picks a whole-number pixel scale, so tiles stay crisp at any window
// size; a bigger window shows more of the world.
func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	s := PixelScale(outsideW, outsideH)
	g.w, g.h = outsideW/s, outsideH/s
	return g.w, g.h
}

func savePNG(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

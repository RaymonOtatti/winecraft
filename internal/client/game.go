package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

var voidColor = color.RGBA{0x1b, 0x14, 0x10, 0xff} // outside the map

// Game is the Ebitengine game: it moves the local player and draws the world.
type Game struct {
	World  *world.World
	Me     *Walker
	Input  Input
	Outbox func(Move) // where step attempts go; nil when playing offline

	// SnapPath, when set, saves frame SnapFrame as a PNG and quits.
	SnapPath  string
	SnapFrame int
	ShowDpad  bool // draw the touch D-pad even before a touch (snapshots)

	script []scriptStep
	tiles  [64]*ebiten.Image // sub-images of one atlas, so draws batch
	player [4][2]*ebiten.Image
	w, h   int
	frame  int
	done   bool
	err    error
}

type scriptStep struct {
	dir   world.Dir
	ticks int
}

// NewGame prepares the art for w, with the local player at pos.
func NewGame(w *world.World, pos world.Point) *Game {
	g := &Game{World: w, Me: NewWalker(w, pos), SnapFrame: 30, w: BaseW, h: BaseH}
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

// Script makes the player walk a fixed route, e.g. "W4,N2" (four steps west,
// then two north), for snapshots. The snapshot is taken after it ends.
func (g *Game) Script(route string) error {
	tps := ebiten.DefaultTPS
	for _, part := range strings.Split(route, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var c rune
		var n int
		if _, err := fmt.Sscanf(part, "%c%d", &c, &n); err != nil || n < 1 {
			return fmt.Errorf("bad route step %q (want e.g. W4)", part)
		}
		d, ok := map[rune]world.Dir{'N': world.North, 'S': world.South, 'W': world.West, 'E': world.East}[c]
		if !ok {
			return fmt.Errorf("bad direction %q in %q", c, part)
		}
		hold := time.Duration(n)*StepDuration + TurnDelay
		g.script = append(g.script, scriptStep{dir: d, ticks: int(hold*time.Duration(tps)/time.Second) + 1})
	}
	total := 0
	for _, s := range g.script {
		total += s.ticks
	}
	g.SnapFrame = total + 30
	return nil
}

func (g *Game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	g.frame++

	g.Input.ScriptHeld = false
	if len(g.script) > 0 {
		g.Input.ScriptDir, g.Input.ScriptHeld = g.script[0].dir, true
		if g.script[0].ticks--; g.script[0].ticks <= 0 {
			g.script = g.script[1:]
		}
	}
	dir, held := g.Input.Poll(NewDpad(g.w, g.h))
	dt := time.Second / time.Duration(ebiten.TPS())
	if m, ok := g.Me.Update(dt, dir, held); ok && g.Outbox != nil {
		g.Outbox(m)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(voidColor)
	px, py := g.Me.DrawPos()
	cam := CenterOn(px+art.Tile/2, py+art.Tile/2, g.w, g.h)
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

	sx, sy := cam.ToScreen(px, py)
	op.GeoM.Reset()
	op.GeoM.Translate(math.Round(sx), math.Round(sy))
	screen.DrawImage(g.player[g.Me.Facing][g.Me.Frame()], &op)

	if g.ShowDpad || g.Input.TouchSeen() {
		drawDpad(screen, NewDpad(g.w, g.h))
	}

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

// Non-premultiplied: color.RGBA would need its channels pre-scaled by alpha.
var (
	padFill = color.NRGBA{0xf4, 0xe9, 0xd8, 0x55}
	padEdge = color.NRGBA{0x2a, 0x1e, 0x1a, 0x99}
)

func drawDpad(screen *ebiten.Image, d Dpad) {
	for _, r := range d.Buttons {
		x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
		vector.FillRect(screen, x, y, w, h, padFill, false)
		vector.StrokeRect(screen, x, y, w, h, 1, padEdge, false)
	}
}

package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

var voidColor = color.RGBA{0x1b, 0x14, 0x10, 0xff} // outside the map

// Game is the Ebitengine game: it moves the local player and draws the world
// and everyone in it.
type Game struct {
	S     *Session
	Net   *Net // nil when playing offline
	Input Input

	// SnapPath, when set, saves frame SnapFrame as a PNG and quits.
	SnapPath  string
	SnapFrame int
	Stay      time.Duration // keep playing this long after the snapshot (multi-client checks)
	ShowDpad  bool          // draw the touch D-pad even before a touch (snapshots)

	script  []scriptStep
	tiles   [64]*ebiten.Image // sub-images of one atlas, so draws batch
	player  [4][2]*ebiten.Image
	w, h    int
	frame   int
	snapped bool
	quitAt  time.Time
	done    bool
	err     error
}

type scriptStep struct {
	dir   world.Dir
	ticks int
}

// OfflineSession plays the dev map alone, without a server.
func OfflineSession(m *world.DevMap, pos world.Point) *Session {
	s := NewSession()
	s.World, s.Me = m.World, NewWalker(m.World, pos)
	s.Bounds, s.Sandbox, s.Joined = m.Bounds, m.Sandbox, true
	return s
}

// NewGame prepares the art for session s; n is nil when offline.
func NewGame(s *Session, n *Net) *Game {
	g := &Game{S: s, Net: n, SnapFrame: 30, w: BaseW, h: BaseH}
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
	if g.snapped && !g.done && !time.Now().Before(g.quitAt) {
		g.done = true
	}
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	if g.Net != nil {
		for i := 0; i < 512; i++ { // bounded per frame, so a burst of chunks can't stall a frame
			m, ok := g.Net.Recv()
			if !ok {
				break
			}
			g.S.Apply(m)
		}
	}
	dt := time.Second / time.Duration(ebiten.TPS())
	g.S.Tick(dt)
	if !g.S.Joined {
		return nil
	}
	g.frame++ // counts frames since joining, so scripts and snapshots wait for the server

	g.Input.ScriptHeld = false
	if len(g.script) > 0 {
		g.Input.ScriptDir, g.Input.ScriptHeld = g.script[0].dir, true
		if g.script[0].ticks--; g.script[0].ticks <= 0 {
			g.script = g.script[1:]
		}
	}
	dir, held := g.Input.Poll(NewDpad(g.w, g.h))
	if m, ok := g.S.Me.Update(dt, dir, held); ok && g.Net != nil {
		g.Net.Send(&proto.Move{Dir: m.Dir, Seq: m.Seq})
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(voidColor)
	me := g.S.Me
	px, py := me.DrawPos()
	cam := CenterOn(px+art.Tile/2, py+art.Tile/2, g.w, g.h)
	x0, y0, x1, y1 := cam.VisibleTiles()

	var op ebiten.DrawImageOptions
	for layer := world.Layer(0); layer < world.NumLayers; layer++ {
		for ty := y0; ty < y1; ty++ {
			for tx := x0; tx < x1; tx++ {
				id := g.S.World.At(layer, tx, ty)
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

	// Everyone else first (sorted by y so lower sprites overlap higher ones),
	// then us on top, then the name tags over all sprites.
	others := make([]*Remote, 0, len(g.S.Players))
	for _, r := range g.S.Players {
		others = append(others, r)
	}
	sort.Slice(others, func(i, j int) bool {
		_, yi := others[i].DrawPos()
		_, yj := others[j].DrawPos()
		return yi < yj || (yi == yj && others[i].ID < others[j].ID)
	})
	for _, r := range others {
		rx, ry := r.DrawPos()
		g.drawSprite(screen, cam, rx, ry, r.Facing, r.Frame())
	}
	g.drawSprite(screen, cam, px, py, me.Facing, me.Frame())
	for _, r := range others {
		rx, ry := r.DrawPos()
		sx, sy := cam.ToScreen(rx, ry)
		drawNameTag(screen, DisplayName(r.Name), int(math.Round(sx))+art.Tile/2, int(math.Round(sy))-2)
	}

	if g.ShowDpad || g.Input.TouchSeen() {
		drawDpad(screen, NewDpad(g.w, g.h))
	}

	if g.SnapPath != "" && g.frame >= g.SnapFrame && !g.snapped {
		g.err = savePNG(screen, g.SnapPath)
		g.snapped, g.quitAt = true, time.Now().Add(g.Stay)
	}
}

func (g *Game) drawSprite(screen *ebiten.Image, cam Camera, x, y float64, facing world.Dir, frame int) {
	sx, sy := cam.ToScreen(x, y)
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(math.Round(sx), math.Round(sy))
	screen.DrawImage(g.player[facing][frame], &op)
}

var tagBack = color.NRGBA{0x1b, 0x14, 0x10, 0xb0}

// drawNameTag centres name above (cx, bottom) on a dark plate. The debug
// font is 6×16 px per glyph, with the glyph in the plate's middle rows.
func drawNameTag(screen *ebiten.Image, name string, cx, bottom int) {
	w := 6*len(name) + 4
	x, y := cx-w/2, bottom-12
	vector.FillRect(screen, float32(x), float32(y), float32(w), 11, tagBack, false)
	ebitenutil.DebugPrintAt(screen, name, x+2, y-3)
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

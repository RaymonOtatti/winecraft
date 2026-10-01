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
	S      *Session
	Net    *Net // nil when playing offline
	Input  Input
	Hotbar *Hotbar

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

type scriptKind int

const (
	scriptWalk   scriptKind = iota // hold dir until n moves were sent
	scriptWait                     // hold nothing for n ticks
	scriptButton                   // press btn once, when standing still
)

type scriptStep struct {
	kind   scriptKind
	dir    world.Dir
	n      int
	btn    Buttons
	budget int // walk steps give up after this many ticks (a wall can stop them)
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
	g := &Game{S: s, Net: n, Hotbar: NewHotbar(), SnapFrame: 30, w: BaseW, h: BaseH}
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

// Script runs a fixed sequence for snapshots, e.g. "S36,W14,H2,A,Z60":
// N/S/W/E<n> walk n steps, A places, B breaks, H<n> picks hotbar slot n,
// Z<n> waits n ticks. The snapshot is taken after it ends.
func (g *Game) Script(route string) error {
	tps := ebiten.DefaultTPS
	for _, part := range strings.Split(route, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "":
			continue
		case part == "A":
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{A: true, Slot: -1}})
			continue
		case part == "B":
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{B: true, Slot: -1}})
			continue
		}
		var c rune
		var n int
		if _, err := fmt.Sscanf(part, "%c%d", &c, &n); err != nil || n < 1 {
			return fmt.Errorf("bad script step %q (want e.g. W4, A, B, H2, Z60)", part)
		}
		switch c {
		case 'H':
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{Slot: n - 1}})
			continue
		case 'Z':
			g.script = append(g.script, scriptStep{kind: scriptWait, n: n})
			continue
		}
		d, ok := map[rune]world.Dir{'N': world.North, 'S': world.South, 'W': world.West, 'E': world.East}[c]
		if !ok {
			return fmt.Errorf("bad direction %q in %q", c, part)
		}
		limit := 3*time.Duration(n)*StepDuration + time.Second
		g.script = append(g.script, scriptStep{kind: scriptWalk, dir: d, n: n, budget: int(limit * time.Duration(tps) / time.Second)})
	}
	g.SnapFrame = math.MaxInt // taken 30 frames after the script ends
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

	g.Input.Scripted = len(g.script) > 0
	var st *scriptStep
	if g.Input.Scripted {
		st = &g.script[0]
		g.Input.ScriptHeld = false
		switch st.kind {
		case scriptButton:
			if !g.S.Me.Moving() {
				g.Input.ScriptBtn = st.btn
				g.nextScriptStep()
			}
		case scriptWait:
			if st.n--; st.n <= 0 {
				g.nextScriptStep()
			}
		case scriptWalk:
			g.Input.ScriptDir, g.Input.ScriptHeld = st.dir, true
		}
	}
	pads := NewPads(g.w, g.h, len(g.Hotbar.Slots))
	dir, held := g.Input.Poll(pads.Dpad)
	m, sent := g.S.Me.Update(dt, dir, held)
	if sent && g.Net != nil {
		g.Net.Send(&proto.Move{Dir: m.Dir, Seq: m.Seq})
	}
	if st != nil && st.kind == scriptWalk {
		if sent {
			st.n--
		}
		if st.budget--; st.n <= 0 || st.budget <= 0 {
			g.nextScriptStep()
		}
	}

	b := g.Input.PollButtons(pads)
	g.Hotbar.Select(b.Slot)
	if b.Next {
		g.Hotbar.Next()
	}
	if b.Prev {
		g.Hotbar.Prev()
	}
	for _, act := range []struct {
		pressed bool
		a       Action
	}{{b.A, ActionPlace}, {b.B, ActionBreak}} {
		if !act.pressed {
			continue
		}
		if e, ok := g.S.EditFor(act.a, g.Hotbar.Selected()); ok {
			g.sendEdit(e)
		}
	}
	return nil
}

func (g *Game) nextScriptStep() {
	g.script = g.script[1:]
	if len(g.script) == 0 {
		g.SnapFrame = g.frame + 30
	}
}

// sendEdit sends an edit to the server, whose TileUpdate will change the map.
// Offline, the edit applies directly, mirroring the server's rule that a
// broken floor leaves dirt.
func (g *Game) sendEdit(e *proto.Edit) {
	if g.Net != nil {
		g.Net.Send(e)
		return
	}
	tile := e.Tile
	if tile == world.None && e.Layer == world.Ground {
		tile = world.Dirt
	}
	g.S.Apply(&proto.TileUpdate{X: e.X, Y: e.Y, Layer: e.Layer, Tile: tile})
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

	if t := g.S.Target(); g.S.CanBuildAt(t) {
		cx, cy := cam.ToScreen(float64(t.X*art.Tile), float64(t.Y*art.Tile))
		vector.StrokeRect(screen, float32(math.Round(cx))+0.5, float32(math.Round(cy))+0.5, art.Tile-1, art.Tile-1, 1, cursorColor, false)
	}

	pads := NewPads(g.w, g.h, len(g.Hotbar.Slots))
	g.drawHotbar(screen, pads)
	if g.ShowDpad || g.Input.TouchSeen() {
		drawDpad(screen, pads.Dpad)
		drawButton(screen, pads.A, "A")
		drawButton(screen, pads.B, "B")
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

var (
	tagBack     = color.NRGBA{0x1b, 0x14, 0x10, 0xb0}
	cursorColor = color.NRGBA{0xff, 0xf4, 0xd6, 0xe0}
	slotBack    = color.NRGBA{0x1b, 0x14, 0x10, 0xa0}
	slotPick    = color.NRGBA{0xe8, 0xc5, 0x6a, 0xff}
)

func (g *Game) drawHotbar(screen *ebiten.Image, p Pads) {
	var op ebiten.DrawImageOptions
	for i, r := range p.Hotbar {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotBack, false)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(r.Min.X+(r.Dx()-art.Tile)/2), float64(r.Min.Y+(r.Dy()-art.Tile)/2))
		screen.DrawImage(g.tiles[g.Hotbar.Slots[i]], &op)
		if i == g.Hotbar.Index() {
			vector.StrokeRect(screen, float32(r.Min.X)+1, float32(r.Min.Y)+1, float32(r.Dx())-2, float32(r.Dy())-2, 2, slotPick, false)
		}
	}
}

func drawButton(screen *ebiten.Image, r image.Rectangle, label string) {
	cx, cy := float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+r.Dy()/2)
	vector.FillCircle(screen, cx, cy, float32(r.Dx()/2), padFill, true)
	vector.StrokeCircle(screen, cx, cy, float32(r.Dx()/2), 1, padEdge, true)
	ebitenutil.DebugPrintAt(screen, label, int(cx)-3, int(cy)-8)
}

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

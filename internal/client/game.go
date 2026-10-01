package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"slices"
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
	MyName string
	Prefs  Prefs // per-device settings; NewGame starts with memory-only

	// SnapPath, when set, saves frame SnapFrame as a PNG and quits.
	SnapPath  string
	SnapFrame int
	Stay      time.Duration // keep playing this long after the snapshot (multi-client checks)
	Bench     bool          // time each frame\'s Update+Draw and report on exit
	ShowDpad  bool          // draw the touch D-pad even before a touch (snapshots)

	toast      Toast
	noticeSeen int
	unjoined   int             // frames spent before joining (a rejected client still snapshots)
	frameCost  []time.Duration // Update+Draw per frame, when benchmarking
	updateCost time.Duration

	script  []scriptStep
	tiles   [64]*ebiten.Image // sub-images of one atlas, so draws batch
	icons   [64]*ebiten.Image // item icons, one atlas
	panel   InvPanel
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
	s.Bounds, s.BuildZone, s.Joined = m.Bounds, m.BuildZone, true
	return s
}

// NewGame prepares the art for session s; n is nil when offline.
func NewGame(s *Session, n *Net) *Game {
	g := &Game{S: s, Net: n, Prefs: MemPrefs{}, SnapFrame: 30, w: BaseW, h: BaseH}
	icons := ebiten.NewImageFromImage(art.Items())
	for id := 1; id < world.NumItems(); id++ {
		g.icons[id] = icons.SubImage(art.ItemRect(world.ItemID(id))).(*ebiten.Image)
	}
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

// Script runs a fixed sequence for snapshots, e.g. "S36,W14,H2,C,Z60":
// N/S/W/E<n> walk n steps, A uses, C builds, X breaks, I toggles the
// inventory, H<n> picks hotbar slot n (or, with the inventory open, puts the
// highlighted item there), Z<n> waits n ticks. The snapshot is taken after.
func (g *Game) Script(route string) error {
	tps := ebiten.DefaultTPS
	for _, part := range strings.Split(route, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "":
			continue
		case part == "A": // use (harvest)
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{Use: true, Slot: -1}})
			continue
		case part == "C": // build
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{Build: true, Slot: -1}})
			continue
		case part == "I": // inventory panel
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{Inv: true, Slot: -1}})
			continue
		case part == "X", part == "B": // break
			g.script = append(g.script, scriptStep{kind: scriptButton, btn: Buttons{Break: true, Slot: -1}})
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
	if g.Bench {
		start := time.Now()
		defer func() { g.updateCost = time.Since(start) }()
	}
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
	if g.S.NoticeSeq != g.noticeSeen {
		g.noticeSeen = g.S.NoticeSeq
		g.toast.Show(g.S.Notice, time.Now())
	}
	if !g.S.Joined {
		g.unjoined++
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
	pads := NewPads(g.w, g.h, proto.HotbarSlots)
	b := g.Input.PollButtons(pads)
	if b.Inv {
		g.panel.Toggle()
	}
	dir, held := g.Input.Poll(pads.Dpad)
	if g.panel.Open {
		held = false // the open panel takes the arrows
	}
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

	if b.Help {
		SetHelpVisible(g.Prefs, !HelpVisible(g.Prefs))
	}
	if g.panel.Open {
		g.updatePanel(b)
		return nil
	}
	g.S.Hotbar.Select(b.Slot)
	if b.Next {
		g.S.Hotbar.Next()
	}
	if b.Prev {
		g.S.Hotbar.Prev()
	}
	for _, act := range []struct {
		pressed bool
		a       Action
	}{{b.Use, ActionUse}, {b.Build, ActionBuild}, {b.Break, ActionBreak}} {
		if act.pressed {
			if m, ok := g.S.Act(act.a, g.S.Hotbar.Selected()); ok {
				g.send(m)
			}
		}
	}
	return nil
}

// updatePanel handles input while the inventory panel is open: arrows move
// the highlight, 1-4 (or a tap on a slot) put the item there, a tap on a row
// highlights it, Esc closes.
func (g *Game) updatePanel(b Buttons) {
	if b.Up {
		g.panel.Move(g.S, -1)
	}
	if b.Down {
		g.panel.Move(g.S, +1)
	}
	for _, t := range b.Taps {
		for i, r := range panelRows(g.w, g.h, len(g.panel.Items(g.S))) {
			if t.In(r) {
				g.panel.Select(g.S, i)
			}
		}
	}
	if hb, ok := g.panel.Assign(g.S, b.Slot); ok {
		g.send(hb)
	}
	if b.Esc {
		g.panel.Open = false
	}
}

func (g *Game) nextScriptStep() {
	g.script = g.script[1:]
	if len(g.script) == 0 {
		g.SnapFrame = g.frame + 30
	}
}

// send sends an action to the server, whose TileUpdate will change the map.
// Offline (the dev map, no server), edits apply directly, mirroring the
// server's rule that a broken floor leaves dirt; harvesting needs a server.
func (g *Game) send(m proto.Msg) {
	if g.Net != nil {
		g.Net.Send(m)
		return
	}
	e, ok := m.(*proto.Edit)
	if !ok {
		return
	}
	tile := e.Tile
	if tile == world.None && e.Layer == world.Ground {
		tile = world.Dirt
	}
	g.S.Apply(&proto.TileUpdate{X: e.X, Y: e.Y, Layer: e.Layer, Tile: tile})
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.Bench {
		start := time.Now()
		defer func() { g.frameCost = append(g.frameCost, g.updateCost+time.Since(start)) }()
	}
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

	if t := g.S.Target(); g.S.Joined && !me.Moving() {
		col := cursorNo
		if g.S.CanBuildAt(t) {
			col = cursorColor
		}
		cx, cy := cam.ToScreen(float64(t.X*art.Tile), float64(t.Y*art.Tile))
		vector.StrokeRect(screen, float32(math.Round(cx))+0.5, float32(math.Round(cy))+0.5, art.Tile-1, art.Tile-1, 1, col, false)
	}

	pads := NewPads(g.w, g.h, proto.HotbarSlots)
	if g.S.Joined {
		g.drawHotbar(screen, pads)
		if g.panel.Open {
			g.drawPanel(screen)
		}
	}
	g.drawHUD(screen, pads)
	if g.ShowDpad || g.Input.TouchSeen() {
		drawDpad(screen, pads.Dpad)
		drawButton(screen, pads.Use, "A")
		drawButton(screen, pads.Build, "C")
		drawButton(screen, pads.Break, "X")
	}

	if g.SnapPath != "" && !g.snapped && (g.frame >= g.SnapFrame || g.unjoined >= 120) {
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
	cursorNo    = color.NRGBA{0xd9, 0x3b, 0x3b, 0xd0}
	slotBack    = color.NRGBA{0x1b, 0x14, 0x10, 0xa0}
	slotPick    = color.NRGBA{0xe8, 0xc5, 0x6a, 0xff}
)

func (g *Game) drawHotbar(screen *ebiten.Image, p Pads) {
	var op ebiten.DrawImageOptions
	for i, r := range p.Hotbar {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotBack, false)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(r.Min.X+(r.Dx()-art.Tile)/2), float64(r.Min.Y+(r.Dy()-art.Tile)/2))
		if it := g.S.Hotbar.Slots[i]; it != world.ItemNone && g.icons[it] != nil {
			screen.DrawImage(g.icons[it], &op)
		}
		if g.Net != nil {
			n := fmt.Sprint(g.S.CountFor(g.S.Hotbar.Slots[i]))
			ebitenutil.DebugPrintAt(screen, n, r.Max.X-6*len(n)-1, r.Max.Y-14)
		}
		if i == g.S.Hotbar.Index() {
			vector.StrokeRect(screen, float32(r.Min.X)+1, float32(r.Min.Y)+1, float32(r.Dx())-2, float32(r.Dy())-2, 2, slotPick, false)
		}
	}
}

var dim = color.NRGBA{0x1b, 0x14, 0x10, 0xc8}

// drawHUD draws the connection status, who is online, the selected tile's
// name and any notice. Before joining it dims the screen and says why.
func (g *Game) drawHUD(screen *ebiten.Image, p Pads) {
	state := Online
	if g.Net != nil {
		state = g.Net.State()
	}
	status := StatusLine(g.S, g.Net != nil, state)
	if !g.S.Joined {
		vector.FillRect(screen, 0, 0, float32(g.w), float32(g.h), dim, false)
		drawText(screen, status, (g.w-6*len(status))/2, g.h/2-8)
		return
	}
	// Top bar: the current goal, across the whole width.
	goal := GoalLine(g.S)
	vector.FillRect(screen, 0, 0, float32(g.w), 13, tagBack, false)
	drawText(screen, goal, (g.w-6*len(goal))/2, 1)

	drawPlate(screen, status, 4, 17)
	if g.Net != nil {
		drawPlate(screen, fmt.Sprintf("Uvas: %d", g.S.Inv[world.ItemGrapes]), 4, 29)
		for i, n := range PlayerList(g.S, g.MyName) {
			drawPlate(screen, n, g.w-4-6*len(n)-4, 17+i*12)
		}
	}
	if HelpVisible(g.Prefs) && !g.panel.Open {
		drawHelp(screen, 4, 47)
	}
	if t := g.toast.Text(time.Now()); t != "" {
		drawPlate(screen, t, (g.w-6*len(t))/2, 33)
	}
	if len(p.Hotbar) > 0 {
		label := HotbarLabel(g.S.Hotbar)
		drawPlate(screen, label, (g.w-6*len(label))/2-2, p.Hotbar[0].Min.Y-14)
	}
}

// panelRows lays out n inventory rows in a centred box.
func panelRows(w, h, n int) []image.Rectangle {
	const rowH, width = 18, 168
	top := max(48, (h-n*rowH)/2)
	out := make([]image.Rectangle, n)
	for i := range out {
		x, y := (w-width)/2, top+i*rowH
		out[i] = image.Rect(x, y, x+width, y+rowH)
	}
	return out
}

func (g *Game) drawPanel(screen *ebiten.Image) {
	items := g.panel.Items(g.S)
	rows := panelRows(g.w, g.h, len(items))
	title := "Inventario: 1-4 pone en la barra, I cierra"
	if len(items) == 0 {
		title = "Inventario vacio (I cierra)"
	}
	box := image.Rect((g.w-176)/2, 30, (g.w+176)/2, 46)
	if len(rows) > 0 {
		box = box.Union(rows[len(rows)-1].Inset(-4))
	}
	vector.FillRect(screen, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), dim, false)
	drawText(screen, title, (g.w-6*len(title))/2, 33)
	var op ebiten.DrawImageOptions
	for i, it := range items {
		r := rows[i]
		if i == g.panel.Cursor(g.S) {
			vector.StrokeRect(screen, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx())-1, float32(r.Dy())-1, 1, slotPick, false)
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(r.Min.X+2), float64(r.Min.Y+1))
		if g.icons[it] != nil {
			screen.DrawImage(g.icons[it], &op)
		}
		label := fmt.Sprintf("%-16s %3d", DisplayName(world.ItemDef(it).Name), g.S.CountFor(it))
		drawText(screen, label, r.Min.X+22, r.Min.Y+4)
	}
}

// drawHelp draws the command side bar with its top-left at (x, y).
func drawHelp(screen *ebiten.Image, x, y int) {
	lines := HelpLines()
	w := 0
	for _, l := range lines {
		w = max(w, 6*len(l))
	}
	vector.FillRect(screen, float32(x), float32(y), float32(w+8), float32(12*len(lines)+4), tagBack, false)
	for i, l := range lines {
		drawText(screen, l, x+4, y+2+12*i)
	}
}

// drawPlate writes s on a dark plate whose top-left is (x, y).
func drawPlate(screen *ebiten.Image, s string, x, y int) {
	vector.FillRect(screen, float32(x), float32(y), float32(6*len(s)+4), 11, tagBack, false)
	drawText(screen, s, x+2, y)
}

// drawText uses the debug font: 6 px per glyph, drawn 3 px above y.
func drawText(screen *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(screen, s, x, y-3)
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
	drawPlate(screen, name, cx-(6*len(name)+4)/2, bottom-12)
}

// Layout picks a whole-number pixel scale, so tiles stay crisp at any window
// size; a bigger window shows more of the world.
func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	s := PixelScale(outsideW, outsideH)
	g.w, g.h = outsideW/s, outsideH/s
	return g.w, g.h
}

// BenchReport summarizes the CPU time per frame (Update + Draw) recorded
// with Bench: average, 95th percentile and worst, skipping the first 30
// frames (start-up).
func (g *Game) BenchReport() string {
	c := g.frameCost
	if len(c) > 30 {
		c = c[30:]
	}
	if len(c) == 0 {
		return "no frames"
	}
	s := slices.Clone(c)
	slices.Sort(s)
	var sum time.Duration
	for _, d := range s {
		sum += d
	}
	return fmt.Sprintf("%d frames, CPU per frame: avg %v, p95 %v, max %v",
		len(s), sum/time.Duration(len(s)), s[len(s)*95/100], s[len(s)-1])
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

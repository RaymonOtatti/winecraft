package client

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Dpad is the on-screen direction pad for touch screens, bottom-left, in
// logical screen pixels. Buttons is indexed by world.Dir.
type Dpad struct {
	Buttons [4]image.Rectangle
}

const dpadButton = 22

// NewDpad lays the pad out for a w×h logical screen.
func NewDpad(w, h int) Dpad {
	cx, cy, s := 12+dpadButton*3/2, h-12-dpadButton*3/2, dpadButton
	at := func(dx, dy int) image.Rectangle {
		x, y := cx+dx*s-s/2, cy+dy*s-s/2
		return image.Rect(x, y, x+s, y+s)
	}
	var d Dpad
	d.Buttons[world.North] = at(0, -1)
	d.Buttons[world.South] = at(0, 1)
	d.Buttons[world.West] = at(-1, 0)
	d.Buttons[world.East] = at(1, 0)
	return d
}

// Hit returns the button under (x, y), with a little slack for thumbs.
func (d Dpad) Hit(x, y int) (world.Dir, bool) {
	p := image.Pt(x, y)
	for dir, r := range d.Buttons {
		if p.In(r.Inset(-3)) {
			return world.Dir(dir), true
		}
	}
	return 0, false
}

// keys maps arrows and WASD to directions.
var keys = []struct {
	k ebiten.Key
	d world.Dir
}{
	{ebiten.KeyArrowUp, world.North}, {ebiten.KeyW, world.North},
	{ebiten.KeyArrowDown, world.South}, {ebiten.KeyS, world.South},
	{ebiten.KeyArrowLeft, world.West}, {ebiten.KeyA, world.West},
	{ebiten.KeyArrowRight, world.East}, {ebiten.KeyD, world.East},
}

// Input tracks which direction is held. With several keys down, the most
// recently pressed wins, the way Pokémon handles it.
type Input struct {
	order      []world.Dir
	touchSeen  bool
	touchIDs   []ebiten.TouchID
	ScriptDir  world.Dir // set by a scripted run (snapshots); overrides real input
	ScriptHeld bool
}

// Poll reads the keyboard and touches and returns the held direction.
func (in *Input) Poll(pad Dpad) (world.Dir, bool) {
	if in.ScriptHeld {
		return in.ScriptDir, true
	}
	for _, kd := range keys {
		if inpututil.IsKeyJustPressed(kd.k) {
			in.order = append(in.order, kd.d)
		}
	}
	// drop directions with no key still down
	kept := in.order[:0]
	for _, d := range in.order {
		if in.dirDown(d) {
			kept = append(kept, d)
		}
	}
	in.order = kept

	in.touchIDs = ebiten.AppendTouchIDs(in.touchIDs[:0])
	for _, id := range in.touchIDs {
		in.touchSeen = true
		if d, ok := pad.Hit(ebiten.TouchPosition(id)); ok {
			return d, true
		}
	}
	if n := len(in.order); n > 0 {
		return in.order[n-1], true
	}
	return 0, false
}

// TouchSeen reports whether this device has touched the screen, which is
// when the D-pad appears.
func (in *Input) TouchSeen() bool { return in.touchSeen }

func (in *Input) dirDown(d world.Dir) bool {
	for _, kd := range keys {
		if kd.d == d && ebiten.IsKeyPressed(kd.k) {
			return true
		}
	}
	return false
}

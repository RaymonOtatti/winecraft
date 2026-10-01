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

// Pads lays out every touch control for a w×h logical screen: the D-pad
// bottom-left, A and B bottom-right, the hotbar bottom-centre.
type Pads struct {
	Dpad   Dpad
	A, B   image.Rectangle
	Hotbar []image.Rectangle
}

const (
	padButton = 26
	slotSize  = 20
)

// NewPads lays out the controls for n hotbar slots.
func NewPads(w, h, n int) Pads {
	p := Pads{Dpad: NewDpad(w, h)}
	p.A = image.Rect(w-12-padButton, h-12-padButton*3/2-padButton/2, w-12, h-12-padButton*3/2+padButton/2)
	p.B = p.A.Sub(image.Pt(padButton+6, -padButton*3/4))
	x0 := (w - n*slotSize) / 2
	for i := 0; i < n; i++ {
		x := x0 + i*slotSize
		p.Hotbar = append(p.Hotbar, image.Rect(x, h-slotSize-4, x+slotSize, h-4))
	}
	return p
}

// Buttons is what was pressed this frame (besides walking).
type Buttons struct {
	A, B       bool
	Slot       int // hotbar slot picked directly, or -1
	Next, Prev bool
}

// Input tracks which direction is held. With several keys down, the most
// recently pressed wins, the way Pokémon handles it.
type Input struct {
	order     []world.Dir
	touchSeen bool
	touchIDs  []ebiten.TouchID
	justIDs   []ebiten.TouchID

	// A scripted run (snapshots) replaces real input while Scripted is set.
	Scripted   bool
	ScriptDir  world.Dir
	ScriptHeld bool
	ScriptBtn  Buttons // consumed by one PollButtons call
}

var (
	keysA    = []ebiten.Key{ebiten.KeySpace, ebiten.KeyZ, ebiten.KeyEnter}
	keysB    = []ebiten.Key{ebiten.KeyX, ebiten.KeyBackspace, ebiten.KeyDelete}
	keysSlot = []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
		ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9}
)

// PollButtons reads A, B and the hotbar from keys, the mouse wheel and touches.
func (in *Input) PollButtons(p Pads) Buttons {
	if in.Scripted {
		b := in.ScriptBtn
		in.ScriptBtn = Buttons{Slot: -1}
		return b
	}
	b := Buttons{Slot: -1}
	b.A = anyJustPressed(keysA)
	b.B = anyJustPressed(keysB)
	for i, k := range keysSlot {
		if inpututil.IsKeyJustPressed(k) {
			b.Slot = i
		}
	}
	b.Next = inpututil.IsKeyJustPressed(ebiten.KeyE)
	b.Prev = inpututil.IsKeyJustPressed(ebiten.KeyQ)
	if _, wy := ebiten.Wheel(); wy < 0 {
		b.Next = true
	} else if wy > 0 {
		b.Prev = true
	}
	in.justIDs = inpututil.AppendJustPressedTouchIDs(in.justIDs[:0])
	for _, id := range in.justIDs {
		pt := image.Pt(ebiten.TouchPosition(id))
		switch {
		case pt.In(p.A.Inset(-4)):
			b.A = true
		case pt.In(p.B.Inset(-4)):
			b.B = true
		default:
			for i, r := range p.Hotbar {
				if pt.In(r) {
					b.Slot = i
				}
			}
		}
	}
	return b
}

func anyJustPressed(ks []ebiten.Key) bool {
	for _, k := range ks {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// Poll reads the keyboard and touches and returns the held direction.
func (in *Input) Poll(pad Dpad) (world.Dir, bool) {
	if in.Scripted {
		return in.ScriptDir, in.ScriptHeld
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

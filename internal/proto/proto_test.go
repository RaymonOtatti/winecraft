package proto

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func sampleChunk() *Chunk {
	c := &Chunk{X: -3, Y: 7}
	for i := range c.Layers[world.Ground] {
		c.Layers[world.Ground][i] = world.Grass
	}
	c.Layers[world.Object][5] = world.Vine
	c.Layers[world.Object][1023] = world.Fence
	return c
}

func samples() []Msg {
	return []Msg{
		&Hello{Version: Version, Name: "Raymon", JoinCode: "malbec-42", Token: "tok_abc123"},
		&Welcome{ID: 7, X: 47, Y: 47, Bounds: world.Rect{X: 0, Y: 0, W: 96, H: 96}, BuildZone: world.Rect{X: 10, Y: 77, W: 24, H: 13}},
		sampleChunk(),
		&Move{Dir: world.East, Seq: 1234},
		&PlayerState{ID: 7, X: -5, Y: 99, Facing: world.North, Seq: 1234, Name: "Franco"},
		&PlayerLeft{ID: 9},
		&Edit{X: 12, Y: 80, Layer: world.Object, Tile: world.StoneWall},
		&TileUpdate{X: 12, Y: 80, Layer: world.Object, Tile: world.None},
		&Inventory{Items: []Item{{ID: world.ItemGrapes, Count: 3}, {ID: world.ItemStone, Count: 12}}},
		&Interact{X: 6, Y: -5},
		&Hotbar{Slots: [HotbarSlots]world.ItemID{world.ItemStone, world.ItemNone, world.ItemGrapes, world.ItemFence}},
		&Inventory{},
		&Ping{Nonce: 99},
		&Error{Code: ErrBadJoinCode, Text: "código incorrecto"},
	}
}

func TestRoundTripEveryMessage(t *testing.T) {
	seen := map[Type]bool{}
	for _, m := range samples() {
		b, err := Encode(m)
		if err != nil {
			t.Fatalf("Encode(%T): %v", m, err)
		}
		if Type(b[0]) != m.Type() {
			t.Fatalf("%T: first byte %d, want type %d", m, b[0], m.Type())
		}
		got, err := Decode(b)
		if err != nil {
			t.Fatalf("Decode(%T): %v", m, err)
		}
		if !reflect.DeepEqual(normalize(got), normalize(m)) {
			t.Fatalf("round trip %T:\n got %+v\nwant %+v", m, got, m)
		}
		seen[m.Type()] = true
	}
	for typ := Type(1); typ < numTypes; typ++ {
		if _, ok := names[typ]; ok && !seen[typ] {
			t.Errorf("message type %v has no round-trip sample", typ)
		}
	}
}

// normalize treats a nil and an empty item list as equal.
func normalize(m Msg) Msg {
	if inv, ok := m.(*Inventory); ok && len(inv.Items) == 0 {
		return &Inventory{}
	}
	return m
}

func TestChunkFitsUnderTheBrowserReadLimit(t *testing.T) {
	b, err := Encode(sampleChunk())
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > MaxMessage || MaxMessage > 32<<10 {
		t.Fatalf("chunk is %d bytes, MaxMessage %d: must stay under the 32 KiB default read limit", len(b), MaxMessage)
	}
}

func TestDecodeRejectsMalformedInput(t *testing.T) {
	good, _ := Encode(&Move{Dir: world.South, Seq: 1})
	cases := map[string][]byte{
		"empty":          {},
		"unknown type":   {200, 1, 2, 3},
		"truncated":      good[:len(good)-1],
		"trailing bytes": append(append([]byte{}, good...), 0),
		"too big":        bytes.Repeat([]byte{byte(TypePing)}, MaxMessage+1),
		"bad direction":  {byte(TypeMove), 9, 0, 0, 0, 0},
		"invalid utf8":   mustEncodeRaw(t, TypeHello, func(w *writer) { w.u16(Version); w.str("\xff\xfe"); w.str("x"); w.str("y") }),
		"name too long":  mustEncodeRaw(t, TypeHello, func(w *writer) { w.u16(Version); w.str(strings.Repeat("a", MaxName+1)); w.str("x"); w.str("y") }),
		"bad layer":      mustEncodeRaw(t, TypeEdit, func(w *writer) { w.i32(1); w.i32(1); w.u8(7); w.u16(uint16(world.Fence)) }),
		"inventory lies": {byte(TypeInventory), 5, 0, 0},
		"unknown item":   {byte(TypeInventory), 1, 200, 0, 1, 0},
		"hotbar unknown": {byte(TypeHotbar), 1, 0, 2, 0, 200, 0, 3, 0},
	}
	for name, b := range cases {
		if _, err := Decode(b); err == nil {
			t.Errorf("%s: Decode accepted %v", name, b)
		}
	}
}

func TestEncodeRejectsOversizedFields(t *testing.T) {
	if _, err := Encode(&Hello{Version: Version, Name: strings.Repeat("a", MaxName+1)}); err == nil {
		t.Fatal("Encode must refuse a name longer than MaxName")
	}
	if _, err := Encode(&Inventory{Items: make([]Item, MaxItems+1)}); err == nil {
		t.Fatal("Encode must refuse more than MaxItems items")
	}
}

func mustEncodeRaw(t *testing.T, typ Type, f func(*writer)) []byte {
	t.Helper()
	w := &writer{}
	w.u8(uint8(typ))
	f(w)
	if w.err != nil {
		t.Fatal(w.err)
	}
	return w.buf
}

// FuzzDecode feeds arbitrary bytes to the decoder, which faces the public
// internet. It must never panic, and anything it accepts must survive a
// re-encode unchanged.
func FuzzDecode(f *testing.F) {
	for _, m := range samples() {
		b, _ := Encode(m)
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := Decode(data)
		if err != nil {
			return
		}
		b, err := Encode(m)
		if err != nil {
			t.Fatalf("decoded %T but cannot re-encode: %v", m, err)
		}
		m2, err := Decode(b)
		if err != nil || !reflect.DeepEqual(normalize(m), normalize(m2)) {
			t.Fatalf("unstable round trip for %T", m)
		}
	})
}

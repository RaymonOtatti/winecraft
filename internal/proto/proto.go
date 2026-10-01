// Package proto is the binary wire protocol between the WineCraft client and
// server. Each game message travels as exactly one WebSocket binary message
// (the socket frames it, so there is no length prefix): one type byte, then
// fixed little-endian fields. Strings are a one-byte length plus UTF-8 bytes.
//
// The decoder faces the public internet: it bounds every length, rejects
// unknown types, invalid enums, truncated input and trailing bytes, and is
// fuzzed (FuzzDecode).
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Version is sent in Hello; the server refuses clients on another version.
// 2: inventory entries are items, not tiles; Interact added.
const Version uint16 = 2

// Limits. MaxMessage keeps every message far below the 32 KiB default read
// limit of the browser WebSocket wrapper.
const (
	MaxMessage  = 8 << 10
	MaxName     = 24
	MaxJoinCode = 32
	MaxToken    = 64
	MaxText     = 200
	MaxItems    = 64
)

// Type is the first byte of every message.
type Type uint8

const (
	TypeHello       Type = iota + 1 // client → server, first message
	TypeWelcome                     // server → client, reply to Hello
	TypeChunk                       // server → client
	TypeMove                        // client → server
	TypePlayerState                 // server → client (also corrects your own position)
	TypePlayerLeft                  // server → client
	TypeEdit                        // client → server: place a tile, or break with Tile=None
	TypeTileUpdate                  // server → client: a tile changed
	TypeInventory                   // server → client
	TypePing                        // both ways: game-level heartbeat
	TypeError                       // server → client
	TypeInteract                    // client → server: use what is at (X, Y), e.g. harvest a vine
	numTypes
)

var names = map[Type]string{
	TypeHello: "Hello", TypeWelcome: "Welcome", TypeChunk: "Chunk", TypeMove: "Move",
	TypePlayerState: "PlayerState", TypePlayerLeft: "PlayerLeft", TypeEdit: "Edit",
	TypeTileUpdate: "TileUpdate", TypeInventory: "Inventory", TypePing: "Ping", TypeError: "Error",
	TypeInteract: "Interact",
}

func (t Type) String() string {
	if n, ok := names[t]; ok {
		return n
	}
	return fmt.Sprintf("Type(%d)", uint8(t))
}

// Error codes carried by the Error message.
const (
	ErrBadVersion uint8 = iota + 1
	ErrBadJoinCode
	ErrServerFull
	ErrBadName
	ErrRateLimited
	ErrNotAllowed
	ErrNoMaterial
)

// Msg is one protocol message.
type Msg interface {
	Type() Type
	encode(w *writer)
	decode(r *reader)
}

type Hello struct {
	Version  uint16
	Name     string
	JoinCode string
	Token    string // identifies a returning player; empty on first visit
}

type Welcome struct {
	ID        uint32
	X, Y      int32
	Bounds    world.Rect
	BuildZone world.Rect
}

type Chunk struct {
	X, Y   int32
	Layers [world.NumLayers][world.ChunkSize * world.ChunkSize]world.TileID
}

type Move struct {
	Dir world.Dir
	Seq uint32 // client's move counter; echoed in PlayerState for reconciliation
}

type PlayerState struct {
	ID     uint32
	X, Y   int32
	Facing world.Dir
	Seq    uint32 // last Move seq the server applied for this player
	Name   string
}

type PlayerLeft struct{ ID uint32 }

type Edit struct {
	X, Y  int32
	Layer world.Layer
	Tile  world.TileID // None breaks what is on Layer
}

type TileUpdate struct {
	X, Y  int32
	Layer world.Layer
	Tile  world.TileID
}

type Item struct {
	ID    world.ItemID
	Count uint16
}

type Inventory struct{ Items []Item }

type Ping struct{ Nonce uint32 }

type Interact struct{ X, Y int32 }

type Error struct {
	Code uint8
	Text string
}

func (*Hello) Type() Type       { return TypeHello }
func (*Welcome) Type() Type     { return TypeWelcome }
func (*Chunk) Type() Type       { return TypeChunk }
func (*Move) Type() Type        { return TypeMove }
func (*PlayerState) Type() Type { return TypePlayerState }
func (*PlayerLeft) Type() Type  { return TypePlayerLeft }
func (*Edit) Type() Type        { return TypeEdit }
func (*TileUpdate) Type() Type  { return TypeTileUpdate }
func (*Inventory) Type() Type   { return TypeInventory }
func (*Ping) Type() Type        { return TypePing }
func (*Error) Type() Type       { return TypeError }
func (*Interact) Type() Type    { return TypeInteract }

// Encode serializes m. It fails if a field exceeds its limit.
func Encode(m Msg) ([]byte, error) {
	w := &writer{buf: make([]byte, 0, 64)}
	w.u8(uint8(m.Type()))
	m.encode(w)
	if w.err != nil {
		return nil, fmt.Errorf("encode %v: %w", m.Type(), w.err)
	}
	if len(w.buf) > MaxMessage {
		return nil, fmt.Errorf("encode %v: %d bytes exceeds MaxMessage", m.Type(), len(w.buf))
	}
	return w.buf, nil
}

// Decode parses one message.
func Decode(b []byte) (Msg, error) {
	if len(b) == 0 {
		return nil, errors.New("empty message")
	}
	if len(b) > MaxMessage {
		return nil, fmt.Errorf("message of %d bytes exceeds MaxMessage", len(b))
	}
	var m Msg
	switch Type(b[0]) {
	case TypeHello:
		m = new(Hello)
	case TypeWelcome:
		m = new(Welcome)
	case TypeChunk:
		m = new(Chunk)
	case TypeMove:
		m = new(Move)
	case TypePlayerState:
		m = new(PlayerState)
	case TypePlayerLeft:
		m = new(PlayerLeft)
	case TypeEdit:
		m = new(Edit)
	case TypeTileUpdate:
		m = new(TileUpdate)
	case TypeInventory:
		m = new(Inventory)
	case TypePing:
		m = new(Ping)
	case TypeError:
		m = new(Error)
	case TypeInteract:
		m = new(Interact)
	default:
		return nil, fmt.Errorf("unknown message type %d", b[0])
	}
	r := &reader{buf: b[1:]}
	m.decode(r)
	if r.err != nil {
		return nil, fmt.Errorf("decode %v: %w", m.Type(), r.err)
	}
	if len(r.buf) != 0 {
		return nil, fmt.Errorf("decode %v: %d trailing bytes", m.Type(), len(r.buf))
	}
	return m, nil
}

func (m *Hello) encode(w *writer) {
	w.u16(m.Version)
	w.strMax(m.Name, MaxName)
	w.strMax(m.JoinCode, MaxJoinCode)
	w.strMax(m.Token, MaxToken)
}

func (m *Hello) decode(r *reader) {
	m.Version = r.u16()
	m.Name = r.str(MaxName)
	m.JoinCode = r.str(MaxJoinCode)
	m.Token = r.str(MaxToken)
}

func (m *Welcome) encode(w *writer) {
	w.u32(m.ID)
	w.i32(m.X)
	w.i32(m.Y)
	w.rect(m.Bounds)
	w.rect(m.BuildZone)
}

func (m *Welcome) decode(r *reader) {
	m.ID = r.u32()
	m.X = r.i32()
	m.Y = r.i32()
	m.Bounds = r.rect()
	m.BuildZone = r.rect()
}

func (m *Chunk) encode(w *writer) {
	w.i32(m.X)
	w.i32(m.Y)
	for l := range m.Layers {
		for _, t := range m.Layers[l] {
			w.u16(uint16(t))
		}
	}
}

func (m *Chunk) decode(r *reader) {
	m.X = r.i32()
	m.Y = r.i32()
	for l := range m.Layers {
		for i := range m.Layers[l] {
			m.Layers[l][i] = r.tile()
		}
	}
}

func (m *Move) encode(w *writer) {
	w.u8(uint8(m.Dir))
	w.u32(m.Seq)
}

func (m *Move) decode(r *reader) {
	m.Dir = r.dir()
	m.Seq = r.u32()
}

func (m *PlayerState) encode(w *writer) {
	w.u32(m.ID)
	w.i32(m.X)
	w.i32(m.Y)
	w.u8(uint8(m.Facing))
	w.u32(m.Seq)
	w.strMax(m.Name, MaxName)
}

func (m *PlayerState) decode(r *reader) {
	m.ID = r.u32()
	m.X = r.i32()
	m.Y = r.i32()
	m.Facing = r.dir()
	m.Seq = r.u32()
	m.Name = r.str(MaxName)
}

func (m *PlayerLeft) encode(w *writer) { w.u32(m.ID) }
func (m *PlayerLeft) decode(r *reader) { m.ID = r.u32() }

func (m *Edit) encode(w *writer) {
	w.i32(m.X)
	w.i32(m.Y)
	w.u8(uint8(m.Layer))
	w.u16(uint16(m.Tile))
}

func (m *Edit) decode(r *reader) {
	m.X = r.i32()
	m.Y = r.i32()
	m.Layer = r.layer()
	m.Tile = r.tile()
}

func (m *TileUpdate) encode(w *writer) {
	w.i32(m.X)
	w.i32(m.Y)
	w.u8(uint8(m.Layer))
	w.u16(uint16(m.Tile))
}

func (m *TileUpdate) decode(r *reader) {
	m.X = r.i32()
	m.Y = r.i32()
	m.Layer = r.layer()
	m.Tile = r.tile()
}

func (m *Inventory) encode(w *writer) {
	if len(m.Items) > MaxItems {
		w.fail(fmt.Errorf("%d items exceeds MaxItems", len(m.Items)))
		return
	}
	w.u8(uint8(len(m.Items)))
	for _, it := range m.Items {
		w.u16(uint16(it.ID))
		w.u16(it.Count)
	}
}

func (m *Inventory) decode(r *reader) {
	n := int(r.u8())
	if n > MaxItems {
		r.fail(fmt.Errorf("%d items exceeds MaxItems", n))
		return
	}
	for i := 0; i < n && r.err == nil; i++ {
		m.Items = append(m.Items, Item{ID: r.item(), Count: r.u16()})
	}
}

func (m *Interact) encode(w *writer) {
	w.i32(m.X)
	w.i32(m.Y)
}

func (m *Interact) decode(r *reader) {
	m.X = r.i32()
	m.Y = r.i32()
}

func (m *Ping) encode(w *writer) { w.u32(m.Nonce) }
func (m *Ping) decode(r *reader) { m.Nonce = r.u32() }

func (m *Error) encode(w *writer) {
	w.u8(m.Code)
	w.strMax(m.Text, MaxText)
}

func (m *Error) decode(r *reader) {
	m.Code = r.u8()
	m.Text = r.str(MaxText)
}

// writer appends little-endian fields; the first failure sticks.
type writer struct {
	buf []byte
	err error
}

func (w *writer) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *writer) u8(v uint8)   { w.buf = append(w.buf, v) }
func (w *writer) u16(v uint16) { w.buf = binary.LittleEndian.AppendUint16(w.buf, v) }
func (w *writer) u32(v uint32) { w.buf = binary.LittleEndian.AppendUint32(w.buf, v) }
func (w *writer) i32(v int32)  { w.u32(uint32(v)) }

func (w *writer) rect(r world.Rect) {
	w.i32(int32(r.X))
	w.i32(int32(r.Y))
	w.i32(int32(r.W))
	w.i32(int32(r.H))
}

func (w *writer) str(s string) { w.strMax(s, 255) }

func (w *writer) strMax(s string, max int) {
	if len(s) > max {
		w.fail(fmt.Errorf("string of %d bytes exceeds limit %d", len(s), max))
		return
	}
	w.u8(uint8(len(s)))
	w.buf = append(w.buf, s...)
}

// reader consumes little-endian fields; the first failure sticks and every
// later read returns zero values.
type reader struct {
	buf []byte
	err error
}

var errShort = errors.New("message truncated")

func (r *reader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if len(r.buf) < n {
		r.fail(errShort)
		return nil
	}
	b := r.buf[:n]
	r.buf = r.buf[n:]
	return b
}

func (r *reader) u8() uint8 {
	if b := r.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *reader) u16() uint16 {
	if b := r.take(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}

func (r *reader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}

func (r *reader) i32() int32 { return int32(r.u32()) }

func (r *reader) rect() world.Rect {
	return world.Rect{X: int(r.i32()), Y: int(r.i32()), W: int(r.i32()), H: int(r.i32())}
}

func (r *reader) str(max int) string {
	n := int(r.u8())
	if n > max {
		r.fail(fmt.Errorf("string of %d bytes exceeds limit %d", n, max))
		return ""
	}
	b := r.take(n)
	if b == nil {
		return ""
	}
	if !utf8.Valid(b) {
		r.fail(errors.New("string is not valid UTF-8"))
		return ""
	}
	return string(b)
}

func (r *reader) dir() world.Dir {
	d := world.Dir(r.u8())
	if r.err == nil && !d.Valid() {
		r.fail(fmt.Errorf("invalid direction %d", d))
	}
	return d
}

func (r *reader) layer() world.Layer {
	l := world.Layer(r.u8())
	if r.err == nil && l >= world.NumLayers {
		r.fail(fmt.Errorf("invalid layer %d", l))
	}
	return l
}

func (r *reader) tile() world.TileID {
	t := world.TileID(r.u16())
	if r.err == nil && int(t) >= world.NumTiles() {
		r.fail(fmt.Errorf("unknown tile %d", t))
	}
	return t
}

func (r *reader) item() world.ItemID {
	it := world.ItemID(r.u16())
	if r.err == nil && int(it) >= world.NumItems() {
		r.fail(fmt.Errorf("unknown item %d", it))
	}
	return it
}

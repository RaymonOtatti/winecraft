package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

const testCode = "malbec-42"

// startServer runs a hub and its HTTP handler on an httptest server.
func startServer(t *testing.T, webDir string) (*httptest.Server, *Hub) {
	t.Helper()
	hub := NewHub(game.New(world.GenerateDevMap(1)), Config{JoinCode: testCode, Tick: 20 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	srv := httptest.NewServer(Handler(hub, webDir))
	t.Cleanup(func() { srv.Close(); cancel() })
	return srv, hub
}

type testClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func dial(t *testing.T, srv *httptest.Server) *testClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadLimit(proto.MaxMessage)
	t.Cleanup(func() { conn.CloseNow() })
	return &testClient{t, conn}
}

func (c *testClient) send(m proto.Msg) {
	c.t.Helper()
	b, err := proto.Encode(m)
	if err != nil {
		c.t.Fatal(err)
	}
	if err := c.conn.Write(context.Background(), websocket.MessageBinary, b); err != nil {
		c.t.Fatal(err)
	}
}

// next reads messages until match returns true, or fails after a timeout.
func (c *testClient) next(what string, match func(proto.Msg) bool) proto.Msg {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		_, b, err := c.conn.Read(ctx)
		if err != nil {
			c.t.Fatalf("waiting for %s: %v", what, err)
		}
		m, err := proto.Decode(b)
		if err != nil {
			c.t.Fatalf("server sent an undecodable message: %v", err)
		}
		if match(m) {
			return m
		}
	}
}

func (c *testClient) join(name string) *proto.Welcome {
	c.send(&proto.Hello{Version: proto.Version, Name: name, JoinCode: testCode})
	return c.next("Welcome", func(m proto.Msg) bool { _, ok := m.(*proto.Welcome); return ok }).(*proto.Welcome)
}

func (c *testClient) waitPlayer(id uint32) *proto.PlayerState {
	return c.next("PlayerState", func(m proto.Msg) bool {
		ps, ok := m.(*proto.PlayerState)
		return ok && ps.ID == id
	}).(*proto.PlayerState)
}

func TestTwoPlayersSeeEachOther(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	a := dial(t, srv)
	wa := a.join("Franco")
	b := dial(t, srv)
	wb := b.join("Raymon")

	if wa.ID == wb.ID {
		t.Fatal("players got the same id")
	}
	if got := a.waitPlayer(wb.ID); got.Name != "Raymon" {
		t.Fatalf("A saw player %q, want Raymon", got.Name)
	}
	if got := b.waitPlayer(wa.ID); got.Name != "Franco" {
		t.Fatalf("B saw player %q, want Franco", got.Name)
	}
}

func TestJoinSendsTheMapChunks(t *testing.T) {
	srv, hub := startServer(t, t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	want := hub.state.Map.World.ChunkCount()
	for got := 0; got < want; got++ {
		c.next("Chunk", func(m proto.Msg) bool { _, ok := m.(*proto.Chunk); return ok })
	}
}

func TestPlayerLeftIsBroadcast(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	a := dial(t, srv)
	a.join("Franco")
	b := dial(t, srv)
	wb := b.join("Raymon")
	a.waitPlayer(wb.ID)
	b.conn.Close(websocket.StatusNormalClosure, "bye")
	a.next("PlayerLeft", func(m proto.Msg) bool {
		pl, ok := m.(*proto.PlayerLeft)
		return ok && pl.ID == wb.ID
	})
}

func TestHandshakeRejections(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	cases := []struct {
		name  string
		hello proto.Msg
		code  uint8
	}{
		{"wrong join code", &proto.Hello{Version: proto.Version, Name: "x", JoinCode: "nope"}, proto.ErrBadJoinCode},
		{"wrong version", &proto.Hello{Version: proto.Version + 1, Name: "x", JoinCode: testCode}, proto.ErrBadVersion},
		{"bad name", &proto.Hello{Version: proto.Version, Name: "<b>", JoinCode: testCode}, proto.ErrBadName},
	}
	for _, tc := range cases {
		c := dial(t, srv)
		c.send(tc.hello)
		e := c.next("Error", func(m proto.Msg) bool { _, ok := m.(*proto.Error); return ok }).(*proto.Error)
		if e.Code != tc.code {
			t.Errorf("%s: error code %d, want %d", tc.name, e.Code, tc.code)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if _, _, err := c.conn.Read(ctx); err == nil {
			t.Errorf("%s: connection must be closed after the error", tc.name)
		}
		cancel()
	}
}

func TestFirstMessageMustBeHello(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	c := dial(t, srv)
	c.send(&proto.Ping{Nonce: 1})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for {
		_, b, err := c.conn.Read(ctx)
		if err != nil {
			return // closed: good
		}
		if m, _ := proto.Decode(b); m != nil {
			if _, ok := m.(*proto.Welcome); ok {
				t.Fatal("server welcomed a client that never said Hello")
			}
		}
	}
}

func TestHealthz(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status %d", res.StatusCode)
	}
}

func TestWasmIsServedGzipped(t *testing.T) {
	dir := t.TempDir()
	payload := bytes.Repeat([]byte("\x00asm wine "), 4096)
	if err := os.WriteFile(filepath.Join(dir, "winecraft.wasm"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	srv, _ := startServer(t, dir)

	req, _ := http.NewRequest("GET", srv.URL+"/winecraft.wasm", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := http.DefaultTransport.RoundTrip(req) // the raw transport does not auto-decompress
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Header.Get("Content-Encoding") != "gzip" || res.Header.Get("Content-Type") != "application/wasm" {
		t.Fatalf("headers: encoding %q type %q", res.Header.Get("Content-Encoding"), res.Header.Get("Content-Type"))
	}
	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if !bytes.Equal(got, payload) {
		t.Fatal("gzipped wasm does not decompress to the file")
	}
}

func TestClientIPTrustsCloudflareOnlyFromLoopback(t *testing.T) {
	cases := []struct {
		remote, header, want string
	}{
		{"127.0.0.1:5555", "203.0.113.9", "203.0.113.9"},
		{"[::1]:5555", "203.0.113.9", "203.0.113.9"},
		{"127.0.0.1:5555", "", "127.0.0.1"},
		{"198.51.100.7:4444", "203.0.113.9", "198.51.100.7"},
		{"127.0.0.1:5555", "not-an-ip", "127.0.0.1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/ws", nil)
		r.RemoteAddr = c.remote
		if c.header != "" {
			r.Header.Set("CF-Connecting-IP", c.header)
		}
		if got := clientIP(r); got != c.want {
			t.Errorf("clientIP(remote %s, header %q) = %s, want %s", c.remote, c.header, got, c.want)
		}
	}
}

func TestNameLimitsAgree(t *testing.T) {
	if game.MaxNameLen > proto.MaxName {
		t.Fatalf("game accepts %d-byte names but the protocol carries only %d", game.MaxNameLen, proto.MaxName)
	}
}

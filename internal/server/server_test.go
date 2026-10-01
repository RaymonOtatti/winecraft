package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
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
	return startServerWith(t, game.New(world.GenerateDevMap(1)), webDir)
}

func startServerWith(t *testing.T, state *game.State, webDir string) (*httptest.Server, *Hub) {
	t.Helper()
	srv, hub, _ := startCfg(t, state, Config{}, webDir)
	return srv, hub
}

// startCfg starts a hub with cfg (join code and a fast tick filled in) and
// returns a function that stops the hub.
func startCfg(t *testing.T, state *game.State, cfg Config, webDir string) (*httptest.Server, *Hub, context.CancelFunc) {
	t.Helper()
	cfg.JoinCode = testCode
	if cfg.Tick == 0 {
		cfg.Tick = 20 * time.Millisecond
	}
	hub := NewHub(state, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	srv := httptest.NewServer(Handler(hub, webDir))
	t.Cleanup(func() { srv.Close(); cancel() })
	return srv, hub, cancel
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

func TestMovesAreValidatedAndBroadcast(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	a := dial(t, srv)
	wa := a.join("Franco")
	b := dial(t, srv)
	wb := b.join("Raymon")
	b.waitPlayer(wa.ID)
	a.waitPlayer(wb.ID)

	a.send(&proto.Move{Dir: world.East, Seq: 1})
	own := a.next("own state after move", func(m proto.Msg) bool {
		ps, ok := m.(*proto.PlayerState)
		return ok && ps.ID == wa.ID && ps.Seq == 1
	}).(*proto.PlayerState)
	if own.X != wa.X+1 || own.Y != wa.Y || own.Facing != world.East {
		t.Fatalf("server moved A to (%d,%d) facing %v, want (%d,%d) East", own.X, own.Y, own.Facing, wa.X+1, wa.Y)
	}
	seen := b.next("A's move", func(m proto.Msg) bool {
		ps, ok := m.(*proto.PlayerState)
		return ok && ps.ID == wa.ID && ps.Seq == 1
	}).(*proto.PlayerState)
	if seen.X != own.X {
		t.Fatalf("B sees A at x=%d, A is at x=%d", seen.X, own.X)
	}
}

func TestSpeedHackOverTheNetworkIsCapped(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	a := dial(t, srv)
	wa := a.join("Franco")
	for seq := uint32(1); seq <= 10; seq++ {
		a.send(&proto.Move{Dir: world.East, Seq: seq})
	}
	final := a.next("state after seq 10", func(m proto.Msg) bool {
		ps, ok := m.(*proto.PlayerState)
		return ok && ps.ID == wa.ID && ps.Seq == 10
	}).(*proto.PlayerState)
	if moved := final.X - wa.X; moved > game.StepBurst+1 {
		t.Fatalf("10 instant moves advanced %d tiles; the burst allows %d", moved, game.StepBurst)
	}
}

// buildMap: grass everywhere, spawn (5,5) inside a sandbox covering x<6.
func buildMap() *world.DevMap {
	w := world.New()
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			w.Set(x, y, world.Grass)
		}
	}
	return &world.DevMap{World: w, Bounds: world.Rect{W: 10, H: 10}, Spawn: world.Point{X: 5, Y: 5},
		Sandbox: world.Rect{W: 6, H: 10}}
}

func isError(code uint8) func(proto.Msg) bool {
	return func(m proto.Msg) bool { e, ok := m.(*proto.Error); return ok && e.Code == code }
}

func TestEditsReachEveryone(t *testing.T) {
	srv, _ := startServerWith(t, game.New(buildMap()), t.TempDir())
	a := dial(t, srv)
	wa := a.join("Franco")
	b := dial(t, srv)
	b.join("Raymon")
	b.waitPlayer(wa.ID)

	a.send(&proto.Edit{X: 4, Y: 5, Layer: world.Object, Tile: world.Fence})
	want := func(m proto.Msg) bool {
		u, ok := m.(*proto.TileUpdate)
		return ok && u.X == 4 && u.Y == 5 && u.Layer == world.Object && u.Tile == world.Fence
	}
	a.next("A's own TileUpdate", want)
	b.next("B sees A's fence", want)
}

func TestEditOutsideTheSandboxIsRefused(t *testing.T) {
	srv, _ := startServerWith(t, game.New(buildMap()), t.TempDir())
	a := dial(t, srv)
	a.join("Franco")
	a.send(&proto.Edit{X: 6, Y: 5, Layer: world.Object, Tile: world.Fence})
	a.next("ErrNotAllowed", isError(proto.ErrNotAllowed))
}

func TestEditFloodGetsRateLimited(t *testing.T) {
	srv, _ := startServerWith(t, game.New(buildMap()), t.TempDir())
	a := dial(t, srv)
	a.join("Franco")
	tiles := []world.TileID{world.Fence, world.None}
	for i := 0; i < 20; i++ {
		a.send(&proto.Edit{X: 4, Y: 5, Layer: world.Object, Tile: tiles[i%2]})
	}
	a.next("ErrRateLimited", isError(proto.ErrRateLimited))
}

// closeErr reads until the server closes the connection and returns the
// read error, or fails the test if it is still open after d.
func (c *testClient) closeErr(d time.Duration) error {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	for {
		if _, _, err := c.conn.Read(ctx); err != nil {
			if ctx.Err() != nil {
				c.t.Fatalf("connection still open after %v", d)
			}
			return err
		}
	}
}

func TestNinthPlayerIsTurnedAway(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{MaxPerIP: 16}, t.TempDir())
	for i := 0; i < 8; i++ {
		dial(t, srv).join(fmt.Sprintf("p%d", i))
	}
	c := dial(t, srv)
	c.send(&proto.Hello{Version: proto.Version, Name: "ninth", JoinCode: testCode})
	c.next("ErrServerFull", isError(proto.ErrServerFull))
}

func TestOversizedFrameClosesTheConnection(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	big := make([]byte, proto.MaxMessage+1)
	big[0] = byte(proto.TypePing)
	c.conn.Write(context.Background(), websocket.MessageBinary, big)
	if got := websocket.CloseStatus(c.closeErr(2 * time.Second)); got != websocket.StatusMessageTooBig {
		t.Fatalf("close status %v, want StatusMessageTooBig", got)
	}
}

func TestIdleClientIsDroppedButAHeartbeatKeepsItAlive(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{Idle: 300 * time.Millisecond}, t.TempDir())
	alive := dial(t, srv)
	wAlive := alive.join("Franco")
	idle := dial(t, srv)
	wIdle := idle.join("Raymon")

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				b, _ := proto.Encode(&proto.Ping{Nonce: 1})
				alive.conn.Write(context.Background(), websocket.MessageBinary, b)
			}
		}
	}()

	idle.closeErr(2 * time.Second)
	alive.next("PlayerLeft for the idle player", func(m proto.Msg) bool {
		pl, ok := m.(*proto.PlayerLeft)
		return ok && pl.ID == wIdle.ID
	})
	time.Sleep(400 * time.Millisecond) // longer than Idle: the heartbeating client must survive
	alive.send(&proto.Move{Dir: world.East, Seq: 1})
	alive.next("still connected", func(m proto.Msg) bool {
		ps, ok := m.(*proto.PlayerState)
		return ok && ps.ID == wAlive.ID && ps.Seq == 1
	})
}

func TestSilentConnectionIsDroppedBeforeHello(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{HelloTimeout: 200 * time.Millisecond}, t.TempDir())
	dial(t, srv).closeErr(2 * time.Second)
}

func TestMessageFloodClosesTheConnection(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{MsgInterval: 50 * time.Millisecond, MsgBurst: 20}, t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	b, _ := proto.Encode(&proto.Ping{Nonce: 1})
	for i := 0; i < 200; i++ {
		if c.conn.Write(context.Background(), websocket.MessageBinary, b) != nil {
			break
		}
	}
	if got := websocket.CloseStatus(c.closeErr(2 * time.Second)); got != websocket.StatusPolicyViolation {
		t.Fatalf("close status %v, want StatusPolicyViolation", got)
	}
}

func TestTooManyConnectionsFromOneIP(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{MaxPerIP: 2}, t.TempDir())
	dial(t, srv).join("a")
	dial(t, srv).join("b")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err == nil {
		conn.CloseNow()
		t.Fatal("a third connection from the same IP must be refused")
	}
	if res == nil || res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want HTTP 429, got %v", res)
	}
}

func TestJoinCodeGuessingIsLockedOut(t *testing.T) {
	srv, _, _ := startCfg(t, game.New(world.GenerateDevMap(1)), Config{FailBurst: 3, MaxPerIP: 16}, t.TempDir())
	for i := 0; i < 3; i++ {
		c := dial(t, srv)
		c.send(&proto.Hello{Version: proto.Version, Name: "x", JoinCode: fmt.Sprintf("guess-%d", i)})
		c.next("ErrBadJoinCode", isError(proto.ErrBadJoinCode))
	}
	c := dial(t, srv)
	c.send(&proto.Hello{Version: proto.Version, Name: "x", JoinCode: testCode})
	c.next("locked out even with the right code", isError(proto.ErrRateLimited))
}

func TestCrossOriginPageIsRefused(t *testing.T) {
	srv, _ := startServer(t, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	opts := &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"https://evil.example"}}}
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", opts)
	if err == nil {
		conn.CloseNow()
		t.Fatal("a page from another origin must not open a game socket")
	}
}

func TestShutdownTellsClientsTheServerIsGoingAway(t *testing.T) {
	srv, _, stop := startCfg(t, game.New(world.GenerateDevMap(1)), Config{}, t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	stop()
	if got := websocket.CloseStatus(c.closeErr(2 * time.Second)); got != websocket.StatusGoingAway {
		t.Fatalf("close status %v, want StatusGoingAway", got)
	}
}

func isInventory(want func(map[world.ItemID]uint16) bool) func(proto.Msg) bool {
	return func(m proto.Msg) bool {
		inv, ok := m.(*proto.Inventory)
		if !ok {
			return false
		}
		have := map[world.ItemID]uint16{}
		for _, it := range inv.Items {
			have[it.ID] = it.Count
		}
		return want(have)
	}
}

func TestJoinSendsTheStarterInventory(t *testing.T) {
	srv, _ := startServerWith(t, game.New(buildMap()), t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	c.next("starter inventory", isInventory(func(h map[world.ItemID]uint16) bool {
		return int(h[world.ItemFence]) == game.StarterKit[world.ItemFence]
	}))
}

func TestHarvestOverTheNetwork(t *testing.T) {
	m := buildMap()
	m.World.Set(5, 4, world.Vine) // north of spawn (5,5)
	srv, _ := startServerWith(t, game.New(m), t.TempDir())
	a := dial(t, srv)
	a.join("Franco")
	b := dial(t, srv)
	b.join("Raymon")

	a.send(&proto.Interact{X: 5, Y: 4})
	harvested := func(m proto.Msg) bool {
		u, ok := m.(*proto.TileUpdate)
		return ok && u.X == 5 && u.Y == 4 && u.Tile == world.VineHarvested
	}
	a.next("the vine turns harvested", harvested)
	b.next("everyone sees it", harvested)
	a.next("grapes in the inventory", isInventory(func(h map[world.ItemID]uint16) bool {
		return int(h[world.ItemGrapes]) == game.GrapesPerHarvest
	}))

	a.send(&proto.Interact{X: 5, Y: 4})
	a.next("no double harvest", isError(proto.ErrNotAllowed))
}

func TestBuildingWithoutMaterialIsRefused(t *testing.T) {
	saved := game.StarterKit[world.ItemCrate]
	game.StarterKit[world.ItemCrate] = 0 // this player starts without crates
	t.Cleanup(func() { game.StarterKit[world.ItemCrate] = saved })

	srv, _ := startServerWith(t, game.New(buildMap()), t.TempDir())
	c := dial(t, srv)
	c.join("Franco")
	c.send(&proto.Edit{X: 4, Y: 5, Layer: world.Object, Tile: world.Crate})
	c.next("ErrNoMaterial", isError(proto.ErrNoMaterial))
}

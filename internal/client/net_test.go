package client

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/server"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

const code = "malbec-1"

func gameServer(t *testing.T, cfg server.Config) string {
	t.Helper()
	cfg.JoinCode = code
	cfg.Tick = 20 * time.Millisecond
	hub := server.NewHub(game.New(world.GenerateDevMap(1)), cfg)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	srv := httptest.NewServer(server.Handler(hub, t.TempDir()))
	t.Cleanup(func() { srv.Close(); cancel() })
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func startNet(t *testing.T, url, name, joinCode string, heartbeat time.Duration) *Net {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return StartNet(ctx, NetConfig{URL: url, Name: name, JoinCode: joinCode, Heartbeat: heartbeat, Backoff: 50 * time.Millisecond})
}

// pump applies incoming messages to s until cond holds or time runs out.
func pump(t *testing.T, n *Net, s *Session, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for {
			m, ok := n.Recv()
			if !ok {
				break
			}
			s.Apply(m)
		}
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s (state %v)", what, n.State())
}

func TestNetJoinsAndTwoClientsSeeEachOther(t *testing.T) {
	url := gameServer(t, server.Config{})
	a, sa := startNet(t, url, "Franco", code, 0), NewSession()
	b, sb := startNet(t, url, "Raymon", code, 0), NewSession()
	pump(t, a, sa, "A welcomed", func() bool { return sa.Joined && a.State() == Online })
	pump(t, b, sb, "B welcomed", func() bool { return sb.Joined })
	pump(t, a, sa, "A sees Raymon", func() bool { r := sa.Players[sb.MyID]; return r != nil && r.Name == "Raymon" })
	pump(t, b, sb, "B sees Franco", func() bool { r := sb.Players[sa.MyID]; return r != nil && r.Name == "Franco" })

	if !a.Send(&proto.Move{Dir: world.East, Seq: 1}) {
		t.Fatal("Send while online must queue the message")
	}
	want := world.Point{X: sa.Me.Pos.X + 1, Y: sa.Me.Pos.Y}
	pump(t, b, sb, "B sees A step east", func() bool { r := sb.Players[sa.MyID]; return r != nil && r.Pos == want })
}

func TestNetReconnectsAfterTheServerDropsIt(t *testing.T) {
	// The server drops clients silent for 300 ms; this client only heartbeats
	// every 10 s, so it gets dropped and must come back on its own.
	url := gameServer(t, server.Config{Idle: 300 * time.Millisecond})
	n, s := startNet(t, url, "Franco", code, 10*time.Second), NewSession()
	pump(t, n, s, "first welcome", func() bool { return s.Joined })
	first := s.MyID
	pump(t, n, s, "a second welcome after the drop", func() bool { return s.MyID != first })
}

func TestNetStopsRetryingAWrongJoinCode(t *testing.T) {
	url := gameServer(t, server.Config{})
	n, s := startNet(t, url, "Franco", "wrong", 0), NewSession()
	pump(t, n, s, "rejection", func() bool { return n.State() == Rejected })
	if s.Notice == "" {
		t.Fatal("the rejection reason must reach the HUD")
	}
}

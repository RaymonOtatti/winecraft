package client

import (
	"strings"
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestStatusLine(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.PlayerState{ID: 9, X: 1, Y: 1, Name: "Raymon"})
	cases := []struct {
		online bool
		state  NetState
		want   string
	}{
		{true, Online, "En linea - 2 jugadores"},
		{true, Connecting, "Conectando..."},
		{true, Offline, "Sin conexion, reconectando..."},
		{false, Online, "Sin servidor (solo)"},
	}
	for _, c := range cases {
		if got := StatusLine(s, c.online, c.state); got != c.want {
			t.Errorf("StatusLine(online=%v, %v) = %q, want %q", c.online, c.state, got, c.want)
		}
	}
	solo := welcomed(t)
	if got := StatusLine(solo, true, Online); got != "En linea - 1 jugador" {
		t.Errorf("one player: %q", got)
	}
}

func TestRejectedStatusExplainsWhy(t *testing.T) {
	s := NewSession()
	s.Apply(&proto.Error{Code: proto.ErrBadJoinCode})
	if got := StatusLine(s, true, Rejected); !strings.Contains(got, "Codigo incorrecto") {
		t.Fatalf("rejected status %q must say why", got)
	}
}

func TestPlayerListIsSortedWithYouFirst(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.PlayerState{ID: 12, Name: "zoe"})
	s.Apply(&proto.PlayerState{ID: 9, Name: "Ánibal"})
	got := PlayerList(s, "Franco")
	want := []string{"Franco (vos)", "Anibal", "zoe"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("PlayerList = %v, want %v", got, want)
	}
}

func TestToastShowsForAWhile(t *testing.T) {
	var tst Toast
	t0 := time.Unix(0, 0)
	if tst.Text(t0) != "" {
		t.Fatal("an empty toast shows nothing")
	}
	tst.Show("No se puede ahí", t0)
	if tst.Text(t0.Add(time.Second)) != "No se puede ahi" {
		t.Fatalf("visible after 1 s: %q", tst.Text(t0.Add(time.Second)))
	}
	if tst.Text(t0.Add(ToastFor+time.Millisecond)) != "" {
		t.Fatal("hidden after ToastFor")
	}
}

func TestHotbarLabelNamesTheSelectedItem(t *testing.T) {
	h := NewHotbar()
	h.Select(2)
	if got := HotbarLabel(h); got != DisplayName(world.ItemDef(h.Selected()).Name) {
		t.Fatalf("label %q", got)
	}
}

func TestZoneNoticeAnnouncesTheZoneYouStepInto(t *testing.T) {
	cases := []struct{ from, to, want string }{
		{"", world.ZoneValley, "Zona: " + world.ZoneValley},
		{world.ZoneValley, world.ZoneValley, ""},
		{world.ZoneValley, world.ZoneWest, "Zona: " + world.ZoneWest},
		{world.ZoneValley, world.ZoneBadlands, "Zona: " + world.ZoneBadlands},
		{world.ZoneBadlands, world.ZoneOasis, "Zona: " + world.ZoneOasis},
		{world.ZoneOasis, world.ZoneValley, "Zona: " + world.ZoneValley},
	}
	for _, c := range cases {
		if got := ZoneNotice(c.from, c.to); got != c.want {
			t.Errorf("ZoneNotice(%q → %q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

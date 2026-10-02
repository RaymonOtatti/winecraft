package client

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestKeyBindings(t *testing.T) {
	if !slices.Contains(keysBuild, ebiten.KeyC) {
		t.Error("C must build")
	}
	if !slices.Contains(keysBreak, ebiten.KeyX) {
		t.Error("X must break")
	}
	if !slices.Contains(keysUse, ebiten.KeySpace) {
		t.Error("Space must use (harvest)")
	}
	for _, k := range keysBuild {
		if slices.Contains(keysUse, k) || slices.Contains(keysBreak, k) {
			t.Errorf("key %v does two things", k)
		}
	}
}

// notified runs f and returns the notice it left, or "" if none.
func notified(s *Session, f func()) string {
	seq := s.NoticeSeq
	f()
	if s.NoticeSeq == seq {
		return ""
	}
	return s.Notice
}

func TestRefusedActionsSayWhy(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Session)
		a     Action
		want  string
	}{
		{"use with nothing to use", func(*Session) {}, ActionUse, "No hay nada para usar acá"},
		{"break with nothing to break", func(*Session) {}, ActionBreak, "No hay nada para romper"},
		{"build outside the build zone", func(s *Session) { s.Me.Reset(world.Point{X: 7, Y: 5}); s.Me.Facing = world.East }, ActionBuild, "Acá no se puede construir"},
		{"build on an occupied tile", func(s *Session) { s.World.Set(6, 5, world.Rock) }, ActionBuild, "Ese lugar está ocupado"},
		{"build without material", func(s *Session) { s.Apply(&proto.Inventory{}) }, ActionBuild, "Te faltan materiales"},
	}
	for _, c := range cases {
		s := builder(t)
		s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemFence, Count: 5}}})
		c.setup(s)
		var sent bool
		got := notified(s, func() { _, sent = s.Act(c.a, world.ItemFence) })
		if sent || got != c.want {
			t.Errorf("%s: sent=%v notice %q, want %q", c.name, sent, got, c.want)
		}
	}
}

func TestNoNoticeWhileWalking(t *testing.T) {
	s := builder(t)
	s.Me.Facing = world.West
	hold(s.Me, world.West, StepDuration/2)
	if got := notified(s, func() { s.Act(ActionBuild, world.ItemFence) }); got != "" {
		t.Fatalf("mid-step presses are ignored quietly, got notice %q", got)
	}
}

func TestBreakSendsTheEdit(t *testing.T) {
	s := builder(t)
	s.World.Set(6, 5, world.Crate)
	m, ok := s.Act(ActionBreak, 0)
	if e, isEdit := m.(*proto.Edit); !ok || !isEdit || e.Tile != world.None || e.Layer != world.Object {
		t.Fatalf("break a crate: %#v ok=%v", m, ok)
	}
}

func TestCursorStateFollowsTheTarget(t *testing.T) {
	s := builder(t)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemFence, Count: 1}}})
	if !s.CanBuildAt(s.Target()) {
		t.Fatal("open grass in the build zone: the cursor must say buildable")
	}
	s.World.Set(6, 5, world.Rock)
	if s.CanBuildAt(s.Target()) {
		t.Fatal("an occupied tile: the cursor must say not buildable")
	}
}

func TestUseOnWaterFishesWhenHoldingRod(t *testing.T) {
	s := builder(t)
	s.Me.Facing = world.North
	s.World.Set(5, 4, world.Water)
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemFishingRod, Count: 1}}})
	m, ok := s.Act(ActionUse, world.ItemNone)
	if !ok {
		t.Fatal("fishing with a rod on water must send an Interact")
	}
	if _, isInteract := m.(*proto.Interact); !isInteract {
		t.Fatalf("want Interact, got %T", m)
	}
}

func TestUseOnWaterWithoutRodIsRefused(t *testing.T) {
	s := builder(t)
	s.Me.Facing = world.North
	s.World.Set(5, 4, world.Water)
	m, ok := s.Act(ActionUse, world.ItemNone)
	if ok {
		t.Fatalf("fishing without a rod must be refused, got %T", m)
	}
}

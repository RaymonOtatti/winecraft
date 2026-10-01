package client

import (
	"strings"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestHelpListsTheMainCommands(t *testing.T) {
	text := strings.Join(HelpLines(), "\n")
	for _, want := range []string{"WASD", "Esp", "C ", "X ", "1-4", "H "} {
		if !strings.Contains(text, want) {
			t.Errorf("help is missing %q:\n%s", want, text)
		}
	}
	for _, l := range HelpLines() {
		if DisplayName(l) != l {
			t.Errorf("help line %q must be plain ASCII for the debug font", l)
		}
	}
}

func TestGoalFollowsProgress(t *testing.T) {
	s := builder(t)
	if g := GoalLine(s); !strings.Contains(g, "cosecha") {
		t.Fatalf("no grapes yet: goal %q should send you to harvest", g)
	}
	s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemGrapes, Count: GoalGrapes}}})
	if g := GoalLine(s); !strings.Contains(g, "construi") {
		t.Fatalf("with %d bunches: goal %q should move on to building", GoalGrapes, g)
	}
	if !strings.HasPrefix(GoalLine(s), "Objetivo: ") {
		t.Fatal("the top bar always starts with 'Objetivo: '")
	}
}

func TestHelpPreferenceIsRemembered(t *testing.T) {
	p := MemPrefs{}
	if !HelpVisible(p) {
		t.Fatal("first visit: help is shown")
	}
	SetHelpVisible(p, false)
	if HelpVisible(p) {
		t.Fatal("hidden help stays hidden on the next visit")
	}
}

func TestBarsFitTheSmallestScreen(t *testing.T) {
	s := builder(t)
	for _, inv := range []int{0, GoalGrapes} {
		s.Apply(&proto.Inventory{Items: []proto.Item{{ID: world.ItemGrapes, Count: uint16(inv)}}})
		if w := 6*len(GoalLine(s)) + 8; w > BaseW {
			t.Errorf("goal %q is %d px wide; the screen is %d", GoalLine(s), w, BaseW)
		}
	}
	for _, l := range HelpLines() {
		if w := 6*len(l) + 8; w > 120 {
			t.Errorf("help line %q is %d px: keep the side bar narrow so it never covers the player", l, w)
		}
	}
}

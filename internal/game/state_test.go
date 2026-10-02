package game

import (
	"github.com/RaymonOtatti/winecraft/internal/world"
	"testing"
)

func TestQuestProgressionOnSandBankGather(t *testing.T) {
	// Setup deterministic dev map
	m := world.GenerateDevMap(0)
	s := New(m)
	// Join a player (anonymous token)
	p, _, err := s.JoinAs("tester", "")
	if err != nil {
		t.Fatalf("JoinAs error: %v", err)
	}
	// Simulate that step 1 is already completed (grapes harvested)
	p.QuestStep = 1
	// Simulate a successful gather from a SandBank: it becomes SandBankDug.
	ch := Change{X: p.Pos.X + 1, Y: p.Pos.Y, Layer: world.Object, Tile: world.SandBankDug}
	chat := s.CheckQuestEdit(p.ID, ch)
	if chat == nil {
		t.Fatalf("Expected quest chat after gathering sand, got nil")
	}
	if p.QuestStep != 2 {
		t.Fatalf("QuestStep not updated, expected 2 got %d", p.QuestStep)
	}
}

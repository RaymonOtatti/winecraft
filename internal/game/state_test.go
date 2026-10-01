package game

import (
	"github.com/RaymonOtatti/winecraft/internal/world"
	"testing"
)

func TestQuestProgressionEdit(t *testing.T) {
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
	// Give player the required material to place a SandBank
	s.give(p, world.ItemArena, 1)
	// Choose a location adjacent to player's spawn (47,47) -> (48,47)
	x, y := p.Pos.X+1, p.Pos.Y
	// Perform the edit to place a SandBank on the Object layer
	// Simulate a successful edit that placed a SandBank
	ch := Change{X: x, Y: y, Layer: world.Object, Tile: world.SandBank}
	// Verify quest progression after the edit
	chat := s.CheckQuestEdit(p.ID, ch)
	if chat == nil {
		t.Fatalf("Expected quest chat after building SandBank, got nil")
	}
	if p.QuestStep != 2 {
		t.Fatalf("QuestStep not updated, expected 2 got %d", p.QuestStep)
	}
}

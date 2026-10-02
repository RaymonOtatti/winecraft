package game

import (
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestCheckQuestAdvancesOnFirstGrapes(t *testing.T) {
	s, p := editor(t)
	if got := s.CheckQuest(p.ID); got != nil {
		t.Fatalf("no grapes yet: got %q", got.Text)
	}
	if _, err := s.Harvest(p.ID, 6, 5, t0); err != nil {
		t.Fatalf("harvest: %v", err)
	}
	chat := s.CheckQuest(p.ID)
	if chat == nil {
		t.Fatal("first harvest must trigger a mentor chat")
	}
	if chat.Type() != proto.TypeChat {
		t.Fatalf("want TypeChat, got %v", chat.Type())
	}
	if p.QuestStep != 1 {
		t.Fatalf("QuestStep = %d, want 1", p.QuestStep)
	}
	// A second check must not re-trigger.
	if got := s.CheckQuest(p.ID); got != nil {
		t.Fatalf("already advanced: got %q", got.Text)
	}
}

func TestQuestStepSurvivesSave(t *testing.T) {
	s := New(editMap())
	p, _, err := s.JoinAs("Franco", "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Harvest(p.ID, 6, 5, t0); err != nil {
		t.Fatal(err)
	}
	s.CheckQuest(p.ID)
	s.Leave(p.ID)

	s2 := New(s.Map)
	if err := s2.Restore(s.Save(time.Now())); err != nil {
		t.Fatalf("restore: %v", err)
	}
	q, _, err := s2.JoinAs("Franco", "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("rejoin: %v", err)
	}
	if q.QuestStep != 1 {
		t.Fatalf("QuestStep not restored: %d", q.QuestStep)
	}
	if q.Inv[world.ItemGrapes] != GrapesPerHarvest {
		t.Fatalf("inventory not restored: %d", q.Inv[world.ItemGrapes])
	}
}

func TestQuestAdvancesOnGatheringSandBank(t *testing.T) {
	s, p := editor(t)
	if _, err := s.Harvest(p.ID, 6, 5, t0); err != nil {
		t.Fatalf("harvest grapes: %v", err)
	}
	s.CheckQuest(p.ID)
	if p.QuestStep != 1 {
		t.Fatalf("after grapes QuestStep = %d, want 1", p.QuestStep)
	}

	// Place a SandBank next to the player so we can gather from it.
	s.Map.World.Set(6, 5, world.SandBank)
	ch, err := s.Harvest(p.ID, 6, 5, t0)
	if err != nil {
		t.Fatalf("gather sand bank: %v", err)
	}
	if ch.Tile != world.SandBankDug {
		t.Fatalf("sand bank spent state = %v, want SandBankDug", ch.Tile)
	}
	chat := s.CheckQuestEdit(p.ID, ch)
	if chat == nil {
		t.Fatal("gathering sand must advance quest to step 2")
	}
	if p.QuestStep != 2 {
		t.Fatalf("QuestStep = %d, want 2", p.QuestStep)
	}
}

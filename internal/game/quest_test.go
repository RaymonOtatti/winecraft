package game

import (
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestCheckQuestAdvancesOnSixGrapes(t *testing.T) {
	s, p := editor(t)
	if got := s.CheckQuest(p.ID); got != nil {
		t.Fatalf("no grapes yet: got %q", got.Text)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.Harvest(p.ID, 6, 5, t0.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("harvest %d: %v", i, err)
		}
		s.Tick(t0.Add(time.Duration(i)*time.Second).Add(RegrowAfter)) // regrow the vine for the next harvest
	}
	chat := s.CheckQuest(p.ID)
	if chat == nil {
		t.Fatal("six grapes must trigger a mentor chat")
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
	for i := 0; i < 3; i++ {
		if _, err := s.Harvest(p.ID, 6, 5, t0.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("harvest %d: %v", i, err)
		}
		s.Tick(t0.Add(time.Duration(i)*time.Second).Add(RegrowAfter))
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
	if q.Inv[world.ItemGrapes] != 6 {
		t.Fatalf("inventory not restored: %d", q.Inv[world.ItemGrapes])
	}
}

func TestQuestAdvancesOnGatheringSandBank(t *testing.T) {
	s, p := editor(t)
	for i := 0; i < 3; i++ {
		if _, err := s.Harvest(p.ID, 6, 5, t0.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("harvest grapes %d: %v", i, err)
		}
		s.Tick(t0.Add(time.Duration(i)*time.Second).Add(RegrowAfter))
	}
	s.CheckQuest(p.ID)
	if p.QuestStep != 1 {
		t.Fatalf("after grapes QuestStep = %d, want 1", p.QuestStep)
	}

	// Place a SandBank next to the player so we can gather from it.
	s.Map.World.Set(6, 5, world.SandBank)
	ch, err := s.Harvest(p.ID, 6, 5, t0.Add(3*time.Second))
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

func TestStoryQuestReachesWine(t *testing.T) {
	s, p := editor(t)
	tick := 0
	next := func() time.Time {
		tick++
		return t0.Add(time.Duration(tick) * time.Second)
	}
	// Step 1: harvest 6 grapes.
	for i := 0; i < 3; i++ {
		now := next()
		if _, err := s.Harvest(p.ID, 6, 5, now); err != nil {
			t.Fatalf("harvest grapes %d: %v", i, err)
		}
		s.Tick(now.Add(RegrowAfter))
	}
	s.CheckQuest(p.ID) // 0 -> 1
	// Step 2: gather sand.
	s.Map.World.Set(6, 5, world.SandBank)
	ch, err := s.Harvest(p.ID, 6, 5, next())
	if err != nil {
		t.Fatalf("gather sand: %v", err)
	}
	s.CheckQuestEdit(p.ID, ch) // 1 -> 2
	// Step 3: craft 4 planks.
	p.Inv[world.ItemRollizo] = 2
	if _, err := s.Craft(p.ID, world.RecipePlanks, next()); err != nil {
		t.Fatalf("craft planks: %v", err)
	}
	chat := s.CheckQuest(p.ID)
	if chat == nil || p.QuestStep != 3 {
		t.Fatalf("planks step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 4: craft a workbench from the planks, then place it.
	if _, err := s.Craft(p.ID, world.RecipeWorkbench, next()); err != nil {
		t.Fatalf("craft workbench: %v", err)
	}
	ch, err = s.Edit(p.ID, 4, 5, world.Object, world.Workbench, next())
	if err != nil {
		t.Fatalf("place workbench: %v", err)
	}
	chat = s.CheckQuestEdit(p.ID, ch)
	if chat == nil || p.QuestStep != 4 {
		t.Fatalf("workbench step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 5: craft 2 stone walls.
	p.Inv[world.ItemCanto] = 4
	for i := 0; i < 2; i++ {
		if _, err := s.Craft(p.ID, world.RecipeStone, next()); err != nil {
			t.Fatalf("craft stone %d: %v", i, err)
		}
	}
	chat = s.CheckQuest(p.ID)
	if chat == nil || p.QuestStep != 5 {
		t.Fatalf("stone step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 6: craft a cellar, break the workbench, then place the cellar.
	if _, err := s.Craft(p.ID, world.RecipeCellar, next()); err != nil {
		t.Fatalf("craft cellar: %v", err)
	}
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.None, next()); err != nil {
		t.Fatalf("break workbench: %v", err)
	}
	ch, err = s.Edit(p.ID, 4, 5, world.Object, world.Cellar, next())
	if err != nil {
		t.Fatalf("place cellar: %v", err)
	}
	chat = s.CheckQuestEdit(p.ID, ch)
	if chat == nil || p.QuestStep != 6 {
		t.Fatalf("cellar step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 7: craft press.
	p.Inv[world.ItemPlanks] += 2
	p.Inv[world.ItemStone] += 2
	if _, err := s.Craft(p.ID, world.RecipePress, next()); err != nil {
		t.Fatalf("craft press: %v", err)
	}
	chat = s.CheckQuest(p.ID)
	if chat == nil || p.QuestStep != 7 {
		t.Fatalf("press craft step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 8: break the cellar, then place press.
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.None, next()); err != nil {
		t.Fatalf("break cellar: %v", err)
	}
	ch, err = s.Edit(p.ID, 4, 5, world.Object, world.Press, next())
	if err != nil {
		t.Fatalf("place press: %v", err)
	}
	chat = s.CheckQuestEdit(p.ID, ch)
	if chat == nil || p.QuestStep != 8 {
		t.Fatalf("press place step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 9: craft barrel.
	p.Inv[world.ItemPlanks] += 4
	p.Inv[world.ItemRollizo] += 2
	if _, err := s.Craft(p.ID, world.RecipeBarrel, next()); err != nil {
		t.Fatalf("craft barrel: %v", err)
	}
	chat = s.CheckQuest(p.ID)
	if chat == nil || p.QuestStep != 9 {
		t.Fatalf("barrel craft step: chat=%v step=%d", chat, p.QuestStep)
	}
	// Step 10: break the press, place barrel, then ferment.
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.None, next()); err != nil {
		t.Fatalf("break press: %v", err)
	}
	ch, err = s.Edit(p.ID, 4, 5, world.Object, world.Barrel, next())
	if err != nil {
		t.Fatalf("place barrel: %v", err)
	}
	p.Inv[world.ItemGrapes] = 6
	p.Inv[world.ItemYeast] = 1
	if _, err := s.Harvest(p.ID, 4, 5, next()); err != nil {
		t.Fatalf("ferment: %v", err)
	}
	chat = s.CheckQuestInteract(p.ID, world.Barrel)
	if chat == nil || p.QuestStep != 10 {
		t.Fatalf("ferment step: chat=%v step=%d", chat, p.QuestStep)
	}
	if p.Inv[world.ItemWine] != 1 {
		t.Fatalf("wine = %d, want 1", p.Inv[world.ItemWine])
	}
}

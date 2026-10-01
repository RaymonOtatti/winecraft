package game

import (
	"errors"
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestCraftConsumesAndProduces(t *testing.T) {
	m := world.GenerateDevMap(1)
	s := New(m)
	p, err := s.Join("vintner")
	if err != nil {
		t.Fatal(err)
	}
	p.Inv = map[world.ItemID]int{world.ItemRollizo: 1}

	if err := s.Craft(p.ID, world.RecipePlanks, time.Now()); err != nil {
		t.Fatalf("craft planks: %v", err)
	}
	if p.Inv[world.ItemRollizo] != 0 {
		t.Fatalf("rollizo not consumed, got %d", p.Inv[world.ItemRollizo])
	}
	if p.Inv[world.ItemPlanks] != 2 {
		t.Fatalf("planks = %d, want 2", p.Inv[world.ItemPlanks])
	}
}

func TestCraftNeedsMaterials(t *testing.T) {
	m := world.GenerateDevMap(1)
	s := New(m)
	p, err := s.Join("vintner")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Craft(p.ID, world.RecipePlanks, time.Now()); !errors.Is(err,ErrNoMaterial) {
		t.Fatalf("empty inventory craft: got %v, want ErrNoMaterial", err)
	}
}

func TestCraftUnknownRecipe(t *testing.T) {
	m := world.GenerateDevMap(1)
	s := New(m)
	p, err := s.Join("vintner")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Craft(p.ID, world.RecipeID(world.NumRecipes()), time.Now()); !errors.Is(err,ErrUnknownRecipe) {
		t.Fatalf("unknown recipe: got %v, want ErrUnknownRecipe", err)
	}
}

func TestCraftRateLimited(t *testing.T) {
	m := world.GenerateDevMap(1)
	s := New(m)
	p, err := s.Join("vintner")
	if err != nil {
		t.Fatal(err)
	}
	p.Inv[world.ItemRollizo] = 100
	now := time.Now()
	for i := 0; i < CraftBurst; i++ {
		if err := s.Craft(p.ID, world.RecipePlanks, now); err != nil {
			t.Fatalf("burst craft %d: %v", i, err)
		}
	}
	if err := s.Craft(p.ID, world.RecipePlanks, now); !errors.Is(err,ErrRateLimited) {
		t.Fatalf("after burst: got %v, want ErrRateLimited", err)
	}
}


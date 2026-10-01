package world

import "testing"

func TestRecipeByID(t *testing.T) {
	for id := RecipeID(0); id < RecipeID(NumRecipes()); id++ {
		r, ok := RecipeByID(id)
		if !ok {
			t.Fatalf("recipe %d missing", id)
		}
		if r.ID != id {
			t.Fatalf("recipe %d has wrong ID %d", id, r.ID)
		}
		if r.Name == "" {
			t.Fatalf("recipe %d has no name", id)
		}
		if len(r.Inputs) == 0 || len(r.Outputs) == 0 {
			t.Fatalf("recipe %q has empty inputs or outputs", r.Name)
		}
	}
	if _, ok := RecipeByID(RecipeID(NumRecipes())); ok {
		t.Fatal("out-of-range recipe accepted")
	}
}

func TestRecipesConsumeAndProduce(t *testing.T) {
	for _, r := range AllRecipes() {
		inv := make(map[ItemID]int)
		for _, in := range r.Inputs {
			inv[in.Item] += in.Count
		}
		if !r.CanCraft(inv) {
			t.Fatalf("recipe %q should be craftable with exact inputs", r.Name)
		}
		r.Craft(inv)
		for _, in := range r.Inputs {
			if inv[in.Item] != 0 {
				t.Fatalf("recipe %q did not consume all of %q", r.Name, ItemDef(in.Item).Name)
			}
		}
		for _, out := range r.Outputs {
			if inv[out.Item] != out.Count {
				t.Fatalf("recipe %q produced %d of %q, want %d", r.Name, inv[out.Item], ItemDef(out.Item).Name, out.Count)
			}
		}
	}
}

func TestRecipesNeedInputs(t *testing.T) {
	for _, r := range AllRecipes() {
		if r.CanCraft(make(map[ItemID]int)) {
			t.Fatalf("recipe %q craftable with empty inventory", r.Name)
		}
	}
}

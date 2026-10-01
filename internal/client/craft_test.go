package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestCraftPanelListsAllRecipes(t *testing.T) {
	p := CraftPanel{}
	if got, want := len(p.Recipes()), world.NumRecipes(); got != want {
		t.Fatalf("recipes = %d, want %d", got, want)
	}
}

func TestCraftPanelCursorStaysInRange(t *testing.T) {
	p := CraftPanel{}
	p.Move(+100)
	if p.Cursor() != len(p.Recipes())-1 {
		t.Fatalf("cursor overshot: %d", p.Cursor())
	}
	p.Move(-100)
	if p.Cursor() != 0 {
		t.Fatalf("cursor undershot: %d", p.Cursor())
	}
}

func TestCraftPanelSendsSelectedRecipe(t *testing.T) {
	p := CraftPanel{Open: true, cursor: 2}
	c, ok := p.Craft()
	if !ok {
		t.Fatal("craft panel open should craft")
	}
	if c.Recipe != uint8(world.RecipeStone) {
		t.Fatalf("recipe = %d, want %d (stone)", c.Recipe, world.RecipeStone)
	}
}

func TestCraftPanelClosedSendsNothing(t *testing.T) {
	p := CraftPanel{Open: false}
	if _, ok := p.Craft(); ok {
		t.Fatal("closed panel must not craft")
	}
}

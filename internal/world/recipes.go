package world

// RecipeID identifies a crafting recipe. Recipes turn raw gathering materials
// into building materials and placeable tiles (PLAN.md §4.5). The order is the
// wire format: append new recipes, never reorder.
type RecipeID uint8

const (
	RecipePlanks RecipeID = iota // Rollizo -> Tablones
	RecipeFence                  // Poste x2 -> Cerca
	RecipeStone                  // Canto x2 -> Cimiento de piedra
	RecipeBarro                  // Limo + Agua de deshielo -> Barro
	RecipeBarroPaja              // Barro + Paja -> Barro con paja
	RecipeAdobe                  // Barro con paja -> Adobe
	RecipeCanizo                 // Caña x3 -> Cañizo
	RecipeQuincha                // Poste + Cañizo + Barro -> Quincha
	RecipeTecho                  // Rollizo + Cañizo + Barro con paja -> Techo de torta
	RecipeFishingRod             // Rollizo + Paja -> Caña de pescar
	RecipeWorkbench              // Tablones -> Banco de trabajo
	RecipeCellar                 // Cimientos -> Bodega
	RecipePress                  // Tablones + Cimiento -> Prensa
	RecipeBarrel                 // Tablones + Rollizo -> Barril
	numRecipes
)

// RecipeStack is a fixed count of one item used by a recipe.
type RecipeStack struct {
	Item  ItemID
	Count int
}

// Recipe describes one crafting transformation.
type Recipe struct {
	ID      RecipeID
	Name    string // shown in the crafting panel
	Inputs  []RecipeStack
	Outputs []RecipeStack
}

// CanCraft reports whether inv holds at least the required inputs.
func (r Recipe) CanCraft(inv map[ItemID]int) bool {
	return len(r.Missing(inv)) == 0
}

// Missing lists every input that inv does not have enough of.
func (r Recipe) Missing(inv map[ItemID]int) []struct {
	Item ItemID
	Have int
	Need int
} {
	var out []struct {
		Item ItemID
		Have int
		Need int
	}
	for _, in := range r.Inputs {
		if inv[in.Item] < in.Count {
			out = append(out, struct {
				Item ItemID
				Have int
				Need int
			}{in.Item, inv[in.Item], in.Count})
		}
	}
	return out
}

// Craft consumes the inputs and adds the outputs to inv.
func (r Recipe) Craft(inv map[ItemID]int) {
	for _, in := range r.Inputs {
		inv[in.Item] -= in.Count
	}
	for _, out := range r.Outputs {
		inv[out.Item] += out.Count
	}
}

var recipes = func() [numRecipes]Recipe {
	var r [numRecipes]Recipe
	r[RecipePlanks] = Recipe{
		ID: RecipePlanks, Name: "Tablones",
		Inputs:  []RecipeStack{{ItemRollizo, 1}},
		Outputs: []RecipeStack{{ItemPlanks, 2}},
	}
	r[RecipeFence] = Recipe{
		ID: RecipeFence, Name: "Cerca",
		Inputs:  []RecipeStack{{ItemPoste, 2}},
		Outputs: []RecipeStack{{ItemFence, 1}},
	}
	r[RecipeStone] = Recipe{
		ID: RecipeStone, Name: "Cimiento de piedra",
		Inputs:  []RecipeStack{{ItemCanto, 2}},
		Outputs: []RecipeStack{{ItemStone, 1}},
	}
	r[RecipeBarro] = Recipe{
		ID: RecipeBarro, Name: "Barro",
		Inputs:  []RecipeStack{{ItemLimo, 1}, {ItemAgua, 1}},
		Outputs: []RecipeStack{{ItemBarro, 2}},
	}
	r[RecipeBarroPaja] = Recipe{
		ID: RecipeBarroPaja, Name: "Barro con paja",
		Inputs:  []RecipeStack{{ItemBarro, 1}, {ItemPaja, 1}},
		Outputs: []RecipeStack{{ItemBarroPaja, 2}},
	}
	r[RecipeAdobe] = Recipe{
		ID: RecipeAdobe, Name: "Adobe",
		Inputs:  []RecipeStack{{ItemBarroPaja, 1}},
		Outputs: []RecipeStack{{ItemAdobe, 2}},
	}
	r[RecipeCanizo] = Recipe{
		ID: RecipeCanizo, Name: "Cañizo",
		Inputs:  []RecipeStack{{ItemCana, 3}},
		Outputs: []RecipeStack{{ItemCanizo, 2}},
	}
	r[RecipeQuincha] = Recipe{
		ID: RecipeQuincha, Name: "Quincha",
		Inputs:  []RecipeStack{{ItemPoste, 1}, {ItemCanizo, 1}, {ItemBarro, 1}},
		Outputs: []RecipeStack{{ItemQuincha, 1}},
	}
	r[RecipeTecho] = Recipe{
		ID: RecipeTecho, Name: "Techo de torta",
		Inputs:  []RecipeStack{{ItemRollizo, 1}, {ItemCanizo, 1}, {ItemBarroPaja, 1}},
		Outputs: []RecipeStack{{ItemTecho, 1}},
	}
	r[RecipeFishingRod] = Recipe{
		ID: RecipeFishingRod, Name: "Caña de pescar",
		Inputs:  []RecipeStack{{ItemRollizo, 1}, {ItemPaja, 1}},
		Outputs: []RecipeStack{{ItemFishingRod, 1}},
	}
	r[RecipeWorkbench] = Recipe{
		ID: RecipeWorkbench, Name: "Banco de trabajo",
		Inputs:  []RecipeStack{{ItemPlanks, 4}},
		Outputs: []RecipeStack{{ItemWorkbench, 1}},
	}
	r[RecipeCellar] = Recipe{
		ID: RecipeCellar, Name: "Bodega",
		Inputs:  []RecipeStack{{ItemStone, 2}},
		Outputs: []RecipeStack{{ItemCellar, 1}},
	}
	r[RecipePress] = Recipe{
		ID: RecipePress, Name: "Prensa",
		Inputs:  []RecipeStack{{ItemPlanks, 2}, {ItemStone, 2}},
		Outputs: []RecipeStack{{ItemPress, 1}},
	}
	r[RecipeBarrel] = Recipe{
		ID: RecipeBarrel, Name: "Barril",
		Inputs:  []RecipeStack{{ItemPlanks, 4}, {ItemRollizo, 2}},
		Outputs: []RecipeStack{{ItemBarrel, 1}},
	}
	return r
}()

// RecipeByID returns the recipe with id, or false if unknown.
func RecipeByID(id RecipeID) (Recipe, bool) {
	if int(id) >= len(recipes) {
		return Recipe{}, false
	}
	return recipes[id], true
}

// NumRecipes is the number of defined recipes.
func NumRecipes() int { return len(recipes) }

// AllRecipes returns every defined recipe, in id order.
func AllRecipes() []Recipe {
	out := make([]Recipe, len(recipes))
	for i := range recipes {
		out[i] = recipes[i]
	}
	return out
}

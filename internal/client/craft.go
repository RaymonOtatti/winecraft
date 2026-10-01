package client

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// CraftPanel is the crafting panel (K): it lists every recipe and crafts the
// highlighted one.
type CraftPanel struct {
	Open   bool
	cursor int
}

// Toggle opens or closes the panel.
func (p *CraftPanel) Toggle() { p.Open = !p.Open }

// Recipes lists every defined recipe, in id order.
func (p *CraftPanel) Recipes() []world.Recipe {
	return world.AllRecipes()
}

// Cursor is the highlighted row, kept inside the list.
func (p *CraftPanel) Cursor() int {
	return max(0, min(p.cursor, len(p.Recipes())-1))
}

// Move moves the highlight by d rows, stopping at the ends.
func (p *CraftPanel) Move(d int) {
	p.cursor = max(0, min(p.Cursor()+d, len(p.Recipes())-1))
}

// Select highlights row i (a tap or click on it).
func (p *CraftPanel) Select(i int) {
	if i >= 0 && i < len(p.Recipes()) {
		p.cursor = i
	}
}

// Craft returns the Craft message for the highlighted recipe, or false if
// the panel is closed or no recipe is selected.
func (p *CraftPanel) Craft() (*proto.Craft, bool) {
	if !p.Open {
		return nil, false
	}
	recipes := p.Recipes()
	if len(recipes) == 0 {
		return nil, false
	}
	return &proto.Craft{Recipe: uint8(recipes[p.Cursor()].ID)}, true
}

// updateCraftPanel handles input while the crafting panel is open.
func (g *Game) updateCraftPanel(b Buttons) {
	if b.Up {
		g.craft.Move(-1)
	}
	if b.Down {
		g.craft.Move(+1)
	}
	for _, t := range b.Taps {
		for i, r := range craftRows(g.w, g.h, len(g.craft.Recipes())) {
			if t.In(r) {
				g.craft.Select(i)
			}
		}
	}
	if b.Use || b.Craft {
		if m, ok := g.craft.Craft(); ok {
			g.send(m)
		}
	}
	if b.Esc {
		g.craft.Open = false
	}
}

// drawCraftPanel draws the recipe list with the first output icon and the
// counts the player has versus needs.
func (g *Game) drawCraftPanel(screen *ebiten.Image) {
	recipes := g.craft.Recipes()
	rows := craftRows(g.w, g.h, len(recipes))
	title := "Craftear: K/Enter crea, Esc cierra"
	box := image.Rect((g.w-176)/2, 30, (g.w+176)/2, 46)
	if len(rows) > 0 {
		box = box.Union(rows[len(rows)-1].Inset(-4))
	}
	vector.FillRect(screen, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), dim, false)
	drawText(screen, title, (g.w-6*len(title))/2, 33)
	var op ebiten.DrawImageOptions
	for i, r := range recipes {
		row := rows[i]
		if i == g.craft.Cursor() {
			vector.StrokeRect(screen, float32(row.Min.X)+0.5, float32(row.Min.Y)+0.5, float32(row.Dx())-1, float32(row.Dy())-1, 1, slotPick, false)
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(row.Min.X+2), float64(row.Min.Y+1))
		out := r.Outputs[0].Item
		if g.icons[out] != nil {
			screen.DrawImage(g.icons[out], &op)
		}
		name := DisplayName(r.Name)
		if len(name) > 14 {
			name = name[:14]
		}
		drawText(screen, name, row.Min.X+22, row.Min.Y+4)
		// show first input: have / need
		if len(r.Inputs) > 0 {
			in := r.Inputs[0]
			have := g.S.CountFor(in.Item)
			need := fmt.Sprintf("%d/%d", have, in.Count)
			drawText(screen, need, row.Max.X-6*len(need)-2, row.Min.Y+4)
		}
	}
}

// craftRows returns the inventory-style rows for drawing the recipe list.
func craftRows(w, h, n int) []image.Rectangle {
	const rowH, width = 18, 168
	top := max(48, (h-n*rowH)/2)
	out := make([]image.Rectangle, n)
	for i := range out {
		x, y := (w-width)/2, top+i*rowH
		out[i] = image.Rect(x, y, x+width, y+rowH)
	}
	return out
}

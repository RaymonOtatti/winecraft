package client

import (
	"slices"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// InvPanel is the inventory panel (I or Tab): it lists what you carry and
// puts the highlighted item into a hotbar slot.
type InvPanel struct {
	Open   bool
	cursor int
}

// Toggle opens or closes the panel.
func (p *InvPanel) Toggle() { p.Open = !p.Open }

// Items lists every item you carry, by item id.
func (p *InvPanel) Items(s *Session) []world.ItemID {
	out := make([]world.ItemID, 0, len(s.Inv))
	for it, n := range s.Inv {
		if n > 0 {
			out = append(out, it)
		}
	}
	slices.Sort(out)
	return out
}

// Cursor is the highlighted row, kept inside the list.
func (p *InvPanel) Cursor(s *Session) int {
	return max(0, min(p.cursor, len(p.Items(s))-1))
}

// Move moves the highlight by d rows, stopping at the ends.
func (p *InvPanel) Move(s *Session, d int) {
	p.cursor = max(0, min(p.Cursor(s)+d, len(p.Items(s))-1))
}

// Select highlights row i (a tap or click on it).
func (p *InvPanel) Select(s *Session, i int) {
	if i >= 0 && i < len(p.Items(s)) {
		p.cursor = i
	}
}

// Assign puts the highlighted item into slot (0-based) and returns the
// Hotbar message that saves it on the server.
func (p *InvPanel) Assign(s *Session, slot int) (*proto.Hotbar, bool) {
	items := p.Items(s)
	if !p.Open || len(items) == 0 || slot < 0 || slot >= proto.HotbarSlots {
		return nil, false
	}
	s.Hotbar.Slots[slot] = items[p.Cursor(s)]
	return &proto.Hotbar{Slots: s.Hotbar.Slots}, true
}

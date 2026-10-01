package client

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// The HUD's text, kept apart from drawing so it can be tested. Everything is
// folded to ASCII for the debug font (see DisplayName).

// StatusLine says how the connection is doing. networked is false when
// playing the dev map alone.
func StatusLine(s *Session, networked bool, st NetState) string {
	if !networked {
		return "Sin servidor (solo)"
	}
	switch st {
	case Online:
		n := len(s.Players) + 1
		if n == 1 {
			return "En linea - 1 jugador"
		}
		return fmt.Sprintf("En linea - %d jugadores", n)
	case Connecting:
		return "Conectando..."
	case Offline:
		return "Sin conexion, reconectando..."
	default:
		return DisplayName("No se pudo entrar: " + s.Notice)
	}
}

// PlayerList is everyone online: you first, then the others by name.
func PlayerList(s *Session, me string) []string {
	others := make([]string, 0, len(s.Players))
	for _, r := range s.Players {
		others = append(others, DisplayName(r.Name))
	}
	slices.SortFunc(others, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return append([]string{DisplayName(me) + " (vos)"}, others...)
}

// HotbarLabel names the selected item.
func HotbarLabel(h *Hotbar) string {
	if h.Selected() == world.ItemNone {
		return "(vacio)"
	}
	return DisplayName(world.ItemDef(h.Selected()).Name)
}

// ToastFor is how long a notice stays on screen.
const ToastFor = 3 * time.Second

// Toast is a short message that fades after ToastFor.
type Toast struct {
	text  string
	until time.Time
}

// Show displays text from now.
func (t *Toast) Show(text string, now time.Time) {
	t.text, t.until = DisplayName(text), now.Add(ToastFor)
}

// Text is what to draw at now, or "".
func (t *Toast) Text(now time.Time) string {
	if now.After(t.until) {
		return ""
	}
	return t.text
}

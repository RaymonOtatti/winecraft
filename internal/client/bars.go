package client

import "github.com/RaymonOtatti/winecraft/internal/world"

// Prefs keeps small per-device settings (browser localStorage, a file
// natively). Values are strings; missing keys read as "".
type Prefs interface {
	Get(key string) string
	Set(key, val string)
}

// MemPrefs keeps preferences in memory (tests, and when no storage works).
type MemPrefs map[string]string

func (m MemPrefs) Get(k string) string { return m[k] }
func (m MemPrefs) Set(k, v string)     { m[k] = v }

// HelpVisible: the command side bar shows until the player hides it.
func HelpVisible(p Prefs) bool { return p.Get("help") != "off" }

// SetHelpVisible remembers the player's choice for the next visit.
func SetHelpVisible(p Prefs, v bool) {
	if v {
		p.Set("help", "on")
	} else {
		p.Set("help", "off")
	}
}

// HelpLines is the side bar: the main commands, plain ASCII for the debug
// font.
func HelpLines() []string {
	return []string{
		"WASD  mover",
		"Esp   usar",
		"C     construir",
		"X     romper",
		"1-4   bloque",
		"I     inventario",
		"K     craftear",
		"H     ayuda",
	}
}

// GoalGrapes is how many bunches the first goal asks for.
const GoalGrapes = 6

// GoalLine is the top bar. Until the story's quest engine drives it (B8),
// it follows simple progress: harvest first, then build.
func GoalLine(s *Session) string {
	if s.Inv[world.ItemGrapes] < GoalGrapes {
		return "Objetivo: cosecha 6 racimos (vinedo, al norte)"
	}
	return "Objetivo: construi algo en el valle (tecla C)"
}

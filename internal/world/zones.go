package world

// Zone names, as the game shows them. The valley is schematic until the real
// pipeline (PLAN.md Phase 2); the zones beside it sit in their real
// directions: the Andean front to the west, the Huayquerías badlands and
// then the eastern oasis to the east.
const (
	ZoneWest     = "Frente andino (Cordón del Portillo)"
	ZoneValley   = "Valle de Uco"
	ZoneBadlands = "Huayquerías"
	ZoneOasis    = "Oasis Este"
)

// Dev map zone edges, in tiles.
const (
	westX0  = -64 // the glacier
	valleyW = devSize
	oasisX0 = 136
	eastX1  = 160
)

// ZoneAt names the zone at (x, y).
func ZoneAt(x, y int) string {
	switch {
	case x < 0:
		return ZoneWest
	case x < valleyW:
		return ZoneValley
	case x < oasisX0:
		return ZoneBadlands
	default:
		return ZoneOasis
	}
}

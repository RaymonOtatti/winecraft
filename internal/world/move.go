package world

// Dir is a facing / movement direction on the tile grid. North is -y.
type Dir uint8

const (
	South Dir = iota // facing the camera, the default
	North
	West
	East
	numDirs
)

var dirDelta = [numDirs][2]int{South: {0, 1}, North: {0, -1}, West: {-1, 0}, East: {1, 0}}

// Valid reports whether d is one of the four directions.
func (d Dir) Valid() bool { return d < numDirs }

// Delta returns the tile offset of one step in direction d.
func (d Dir) Delta() (dx, dy int) {
	if !d.Valid() {
		return 0, 0
	}
	return dirDelta[d][0], dirDelta[d][1]
}

// Step is where a move ends. Hop is set when the move jumped down a ledge,
// which covers two tiles.
type Step struct {
	X, Y int
	Hop  bool
}

// CanStep reports where a player standing at (x, y) ends up after moving one
// step in direction d, and whether the move is allowed. The client predicts
// with it and the server validates with it, so both always agree.
func (w *World) CanStep(x, y int, d Dir) (Step, bool) {
	if !d.Valid() {
		return Step{}, false
	}
	dx, dy := d.Delta()
	nx, ny := x+dx, y+dy

	if w.At(Ground, nx, ny) == LedgeSouth {
		// Ledges drop to the south: hop over the ledge tile and land beyond it.
		if d != South {
			return Step{}, false
		}
		lx, ly := nx, ny+1
		if w.At(Ground, lx, ly) == LedgeSouth || !w.standable(lx, ly) {
			return Step{}, false
		}
		return Step{X: lx, Y: ly, Hop: true}, true
	}

	if !w.standable(nx, ny) {
		return Step{}, false
	}
	return Step{X: nx, Y: ny}, true
}

// standable reports whether a player can occupy (x, y): walkable ground and
// either no object or a walkable one.
func (w *World) standable(x, y int) bool {
	if !Def(w.At(Ground, x, y)).Walkable {
		return false
	}
	obj := w.At(Object, x, y)
	return obj == None || Def(obj).Walkable
}

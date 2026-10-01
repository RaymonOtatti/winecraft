package game

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Store keeps a saved game between server runs. The game owns this port; an
// adapter implements it (a JSON file today, SQLite later).
type Store interface {
	Load() (*Saved, error) // (nil, nil) when nothing has been saved yet
	Save(*Saved) error
}

// SaveVersion changes when Saved's shape does.
const SaveVersion = 1

// Saved is everything needed to bring a game back after a restart: the
// chunks that changed since the map was generated, vines waiting to regrow,
// and the progress of every player who has a token.
type Saved struct {
	Version   int
	SavedAt   time.Time
	MapDigest string `json:",omitempty"` // the layout the chunks belong to
	Chunks    []SavedChunk
	Regrow    []SavedRegrow
	Profiles  map[string]Profile
}

// SavedChunk is one changed chunk; each layer is ChunkSize² little-endian
// uint16 tile ids.
type SavedChunk struct {
	X, Y           int32
	Ground, Object []byte
}

// SavedRegrow is a harvested vine and when it bears again.
type SavedRegrow struct {
	X, Y int
	When time.Time
}

// Profile is a player's saved progress.
type Profile struct {
	Name      string
	X, Y      int
	Facing    world.Dir
	Inv       map[world.ItemID]int
	Hotbar    [HotbarSlots]world.ItemID
	QuestStep int    `json:",omitempty"` // B8 quest progress; missing = 0
	Fog       []byte `json:",omitempty"` // discovered-tile mask from world.Fog
}

var tokenRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ValidToken reports whether t is a player token: 128 random bits in hex,
// made once by the client and kept on the device.
func ValidToken(t string) bool { return tokenRE.MatchString(t) }

func profileOf(p *Player) Profile {
	inv := make(map[world.ItemID]int, len(p.Inv))
	for it, n := range p.Inv {
		if n > 0 {
			inv[it] = n
		}
	}
	prof := Profile{Name: p.Name, X: p.Pos.X, Y: p.Pos.Y, Facing: p.Facing, Inv: inv, Hotbar: p.Hotbar, QuestStep: p.QuestStep}
	if p.Fog != nil {
		prof.Fog = p.Fog.Bytes()
	}
	return prof
}

// Save captures the game. Online players are saved as they are right now.
func (s *State) Save(now time.Time) *Saved {
	sv := &Saved{Version: SaveVersion, SavedAt: now, MapDigest: s.mapDigest, Profiles: make(map[string]Profile, len(s.profiles))}
	for tok, prof := range s.profiles {
		sv.Profiles[tok] = prof
	}
	for _, p := range s.players {
		if p.token != "" {
			sv.Profiles[p.token] = profileOf(p)
		}
	}
	for _, cc := range s.Map.World.Modified() {
		c := s.Map.World.Chunk(cc)
		if c == nil {
			continue
		}
		sv.Chunks = append(sv.Chunks, SavedChunk{X: cc.X, Y: cc.Y,
			Ground: packLayer(&c.Layers[world.Ground]), Object: packLayer(&c.Layers[world.Object])})
	}
	for _, r := range s.regrow {
		sv.Regrow = append(sv.Regrow, SavedRegrow{X: r.at.X, Y: r.at.Y, When: r.when})
	}
	return sv
}

// Restore loads a saved game into a freshly generated one, before anyone joins.
func (s *State) Restore(sv *Saved) error {
	if sv.Version != SaveVersion {
		return fmt.Errorf("saved game version %d, this server reads %d", sv.Version, SaveVersion)
	}
	// Chunks and timers saved on another map layout would be pasted onto the
	// wrong terrain: drop them, keep the players. (Older saves have no digest.)
	if sv.MapDigest != "" && sv.MapDigest != s.mapDigest {
		s.MapChanged = true
		sv = &Saved{Version: sv.Version, Profiles: sv.Profiles}
	}
	for _, sc := range sv.Chunks {
		var layers [world.NumLayers][world.ChunkSize * world.ChunkSize]world.TileID
		if err := unpackLayer(sc.Ground, &layers[world.Ground]); err != nil {
			return fmt.Errorf("chunk (%d,%d): %w", sc.X, sc.Y, err)
		}
		if err := unpackLayer(sc.Object, &layers[world.Object]); err != nil {
			return fmt.Errorf("chunk (%d,%d): %w", sc.X, sc.Y, err)
		}
		s.Map.World.PutChunk(world.ChunkCoord{X: sc.X, Y: sc.Y}, &layers)
	}
	for _, r := range sv.Regrow {
		s.regrow = append(s.regrow, regrowth{at: world.Point{X: r.X, Y: r.Y}, when: r.When})
	}
	for tok, prof := range sv.Profiles {
		if ValidToken(tok) {
			s.profiles[tok] = prof
		}
	}
	return nil
}

func packLayer(l *[world.ChunkSize * world.ChunkSize]world.TileID) []byte {
	b := make([]byte, 0, 2*len(l))
	for _, t := range l {
		b = binary.LittleEndian.AppendUint16(b, uint16(t))
	}
	return b
}

func unpackLayer(b []byte, l *[world.ChunkSize * world.ChunkSize]world.TileID) error {
	if len(b) != 2*len(l) {
		return fmt.Errorf("layer is %d bytes, want %d", len(b), 2*len(l))
	}
	for i := range l {
		t := world.TileID(binary.LittleEndian.Uint16(b[2*i:]))
		if int(t) >= world.NumTiles() {
			return fmt.Errorf("unknown tile %d", t)
		}
		l[i] = t
	}
	return nil
}

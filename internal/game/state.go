// Package game holds the authoritative game state and its rules: who is
// playing, where they stand, and which moves and edits are allowed. It knows
// nothing about networks or encodings; the server package drives it from a
// single goroutine, so it needs no locks.
package game

import (
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/RaymonOtatti/winecraft/internal/rate"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// MaxNameLen is the longest player name in bytes. It must not exceed
// proto.MaxName (checked by a server test).
const MaxNameLen = 24

var (
	ErrFull    = errors.New("server is full")
	ErrBadName = errors.New("invalid player name")
)

// Player is one connected player.
type Player struct {
	ID     uint32
	Name   string
	Pos    world.Point
	Facing world.Dir
	Seq    uint32 // last client move sequence applied
	Inv    map[world.ItemID]int
	Hotbar [HotbarSlots]world.ItemID

	token string // stable identity across reconnects and restarts; "" for anonymous

	steps rate.Bucket // movement rate limit, see Move
	edits rate.Bucket // edit rate limit, see Edit
}

// State is the whole authoritative game.
type State struct {
	Map        *world.DevMap
	MaxPlayers int

	players  map[uint32]*Player
	dirty    map[uint32]bool
	invDirty map[uint32]bool
	regrow   []regrowth
	profiles map[string]Profile // saved progress by token, for players not online
	nextID   uint32
}

// New starts an empty game on map m.
func New(m *world.DevMap) *State {
	m.World.MarkBaseline() // the map as generated; only later changes are saved
	return &State{
		Map:        m,
		MaxPlayers: 8,
		players:    make(map[uint32]*Player),
		dirty:      make(map[uint32]bool),
		invDirty:   make(map[uint32]bool),
		profiles:   make(map[string]Profile),
	}
}

// Join adds an anonymous player at the spawn point; nothing is saved for them.
func (s *State) Join(name string) (*Player, error) {
	p, _, err := s.JoinAs(name, "")
	return p, err
}

// JoinAs adds a player identified by token (see ValidToken). A returning
// token gets back its saved position and inventory; if that token is already
// online, the old connection is replaced and its id returned so the server can
// close it. An empty or malformed token plays anonymously.
func (s *State) JoinAs(name, token string) (p *Player, replaced uint32, err error) {
	name, err = CleanName(name)
	if err != nil {
		return nil, 0, err
	}
	if !ValidToken(token) {
		token = ""
	}
	if token != "" {
		for _, other := range s.players {
			if other.token == token {
				replaced = other.ID
				s.Leave(other.ID)
				break
			}
		}
	}
	if len(s.players) >= s.MaxPlayers {
		return nil, replaced, ErrFull
	}
	s.nextID++
	p = &Player{ID: s.nextID, Name: name, Pos: s.Map.Spawn, Facing: world.South, Inv: make(map[world.ItemID]int), Hotbar: DefaultHotbar, token: token}
	if prof, ok := s.profiles[token]; ok && token != "" {
		p.Facing = prof.Facing
		if pos := (world.Point{X: prof.X, Y: prof.Y}); s.Map.Bounds.Contains(pos.X, pos.Y) && s.Map.World.Standable(pos.X, pos.Y) {
			p.Pos = pos
		}
		for it, n := range prof.Inv {
			p.Inv[it] = n
		}
		if prof.Hotbar != ([HotbarSlots]world.ItemID{}) { // older saves have none
			p.Hotbar = prof.Hotbar
		}
	} else {
		for it, n := range StarterKit {
			p.Inv[it] = n
		}
	}
	s.players[p.ID] = p
	s.dirty[p.ID] = true
	s.invDirty[p.ID] = true
	return p, replaced, nil
}

// Leave removes a player, keeping their progress if they have a token.
// Unknown ids are ignored.
func (s *State) Leave(id uint32) {
	if p := s.players[id]; p != nil && p.token != "" {
		s.profiles[p.token] = profileOf(p)
	}
	delete(s.players, id)
	delete(s.dirty, id)
	delete(s.invDirty, id)
}

// Player returns the player with id, or nil.
func (s *State) Player(id uint32) *Player { return s.players[id] }

// Players returns all players ordered by id.
func (s *State) Players() []*Player {
	out := make([]*Player, 0, len(s.players))
	for _, p := range s.players {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b *Player) int { return int(a.ID) - int(b.ID) })
	return out
}

// TakeDirty returns copies of the players that changed since the last call,
// ordered by id, and clears the set.
func (s *State) TakeDirty() []Player {
	out := make([]Player, 0, len(s.dirty))
	for id := range s.dirty {
		if p := s.players[id]; p != nil {
			out = append(out, *p)
		}
	}
	clear(s.dirty)
	slices.SortFunc(out, func(a, b Player) int { return int(a.ID) - int(b.ID) })
	return out
}

// CleanName trims a player name, collapses runs of spaces, and accepts only
// letters, digits, spaces and - _ . (accents and ñ included). Names are drawn
// by the game, never inserted into HTML, but a tight charset keeps them
// readable and stops look-alike tricks.
func CleanName(name string) (string, error) {
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", ErrBadName // includes tabs, newlines and other control characters
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len(name) > MaxNameLen {
		return "", ErrBadName
	}
	return name, nil
}

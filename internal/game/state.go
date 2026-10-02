// Package game holds the authoritative game state and its rules: who is
// playing, where they stand, and which moves and edits are allowed. It knows
// nothing about networks or encodings; the server package drives it from a
// single goroutine, so it needs no locks.
package game

import (
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/RaymonOtatti/winecraft/internal/proto"
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
	ID        uint32
	Name      string
	Pos       world.Point
	Facing    world.Dir
	Seq       uint32 // last client move sequence applied
	Inv       map[world.ItemID]int
	Hotbar    [HotbarSlots]world.ItemID
	QuestStep int      // simple quest progression
	Fog       *world.Fog // discovered-tile mask, saved with the profile

	token string // stable identity across reconnects and restarts; "" for anonymous

	steps rate.Bucket // movement rate limit, see Move
	edits rate.Bucket // edit rate limit, see Edit
}

// State is the whole authoritative game.
type State struct {
	Map        *world.DevMap
	MaxPlayers int
	// MapChanged is set by Restore when the save was made on another map
	// layout: players were kept, world edits dropped.
	MapChanged bool

	players   map[uint32]*Player
	dirty     map[uint32]bool
	invDirty  map[uint32]bool
	regrow    []regrowth
	profiles  map[string]Profile // saved progress by token, for players not online
	mapDigest string             // the generated layout, to tell whether a save fits it
	nextID    uint32
}

// New starts an empty game on map m.
func New(m *world.DevMap) *State {
	digest := m.World.Digest()
	m.World.MarkBaseline() // the map as generated; only later changes are saved
	return &State{
		Map:        m,
		mapDigest:  hex.EncodeToString(digest[:]),
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
	p.Fog = world.NewFog(s.Map.Bounds)
	if p.Fog != nil {
		p.Fog.Reveal(p.Pos.X, p.Pos.Y, MapRevealRadius)
	}
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
		p.QuestStep = prof.QuestStep
		if p.Fog != nil && len(prof.Fog) > 0 {
			p.Fog.SetBytes(prof.Fog)
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

// CheckQuest checks the player's quest progress after an inventory change.
// It returns a Chat message when a step is reached.
func (s *State) CheckQuest(id uint32) *proto.Chat {
	p := s.players[id]
	if p == nil {
		return nil
	}
	switch p.QuestStep {
	case 0:
		if p.Inv[world.ItemGrapes] >= 6 {
			p.QuestStep = 1
			return &proto.Chat{Text: "¡Has cosechado seis racimos! El primer paso de cualquier bodega de Valle de Uco es la viña. Ahora busca arena en el cauce del arroyo para preparar el banco de trabajo."}
		}
	case 2:
		if p.Inv[world.ItemPlanks] >= 4 {
			p.QuestStep = 3
			return &proto.Chat{Text: "¡Excelente! Las tablas son la base de cualquier estructura. Ahora construyamos un banco de trabajo en el valle."}
		}
	case 4:
		if p.Inv[world.ItemStone] >= 2 {
			p.QuestStep = 5
			return &proto.Chat{Text: "Las paredes de piedra mantendrán tu bodega estable. Construye la bodega en el valle para proteger la cosecha."}
		}
	case 6:
		if p.Inv[world.ItemPress] >= 1 {
			p.QuestStep = 7
			return &proto.Chat{Text: "La prensa es esencial para extraer el jugo. Colócala dentro de la bodega para iniciar la fermentación."}
		}
	case 8:
		if p.Inv[world.ItemBarrel] >= 1 {
			p.QuestStep = 9
			return &proto.Chat{Text: "El barril permite que el jugo se convierta en vino. Colócalo en la bodega; ya llevas levadura en la bolsa."}
		}
	}
	return nil
}

// CheckQuestEdit checks quest progress after a successful world change.
func (s *State) CheckQuestEdit(id uint32, ch Change) *proto.Chat {
	p := s.players[id]
	if p == nil {
		return nil
	}
	switch p.QuestStep {
	case 1:
		// Step 1 → 2: gather sand from a river sand bank in the valley.
		if ch.Tile == world.SandBankDug && ch.Layer == world.Object {
			p.QuestStep = 2
			return &proto.Chat{Text: "¡Bien hecho! Has recogido arena del río. Con ella podemos hacer el banco de trabajo."}
		}
	case 3:
		// Step 3 → 4: place a workbench.
		if ch.Tile == world.Workbench && ch.Layer == world.Object {
			p.QuestStep = 4
			return &proto.Chat{Text: "¡El banco de trabajo está listo! Con él podrás ensamblar herramientas más complejas."}
		}
	case 5:
		// Step 5 → 6: build a cellar.
		if ch.Tile == world.Cellar && ch.Layer == world.Object {
			p.QuestStep = 6
			return &proto.Chat{Text: "¡Tu bodega está en pie! Ahora almacena tus uvas y prepáralas para el proceso de fermentación."}
		}
	case 7:
		// Step 7 → 8: place the press.
		if ch.Tile == world.Press && ch.Layer == world.Object {
			p.QuestStep = 8
			return &proto.Chat{Text: "¡La prensa está lista! Ahora vamos a fermentar el mosto."}
		}
	}
	return nil
}

// CheckQuestInteract checks quest progress after a successful Interact that
// does not change a tile (e.g. fishing, fermenting in a barrel).
func (s *State) CheckQuestInteract(id uint32, tile world.TileID) *proto.Chat {
	p := s.players[id]
	if p == nil {
		return nil
	}
	// Story completion: ferment in a placed barrel.
	if p.QuestStep == 9 && tile == world.Barrel {
		p.QuestStep = 10
		return &proto.Chat{Text: "¡Felicidades! Has completado la primera fase de la elaboración del vino. Próximamente, aprenderás a embotellar y vender tu producción."}
	}
	return nil
}

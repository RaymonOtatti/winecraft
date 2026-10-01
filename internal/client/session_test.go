package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func welcomed(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	s.Apply(&proto.Welcome{ID: 7, X: 5, Y: 5, Bounds: world.Rect{W: 10, H: 10}, BuildZone: world.Rect{W: 6, H: 10}})
	ch := &proto.Chunk{}
	for i := range ch.Layers[world.Ground] {
		ch.Layers[world.Ground][i] = world.Grass
	}
	s.Apply(ch)
	return s
}

func TestWelcomeJoinsAtTheServersPosition(t *testing.T) {
	s := welcomed(t)
	if !s.Joined || s.MyID != 7 || s.Me.Pos != (world.Point{X: 5, Y: 5}) {
		t.Fatalf("after Welcome: joined %v id %d pos %v", s.Joined, s.MyID, s.Me.Pos)
	}
	if s.BuildZone != (world.Rect{W: 6, H: 10}) {
		t.Fatalf("build zone %v not taken from Welcome", s.BuildZone)
	}
}

func TestChunkFillsTheWorld(t *testing.T) {
	s := welcomed(t)
	if s.World.At(world.Ground, 3, 3) != world.Grass {
		t.Fatal("chunk tiles not applied")
	}
}

func TestOtherPlayersAppearMoveAndLeave(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.PlayerState{ID: 9, X: 2, Y: 2, Facing: world.South, Name: "Raymon"})
	r := s.Players[9]
	if r == nil || r.Name != "Raymon" || r.Pos != (world.Point{X: 2, Y: 2}) {
		t.Fatalf("remote player: %+v", r)
	}
	s.Apply(&proto.PlayerState{ID: 9, X: 3, Y: 2, Facing: world.East, Seq: 1, Name: "Raymon"})
	if !r.Moving() || r.Pos != (world.Point{X: 3, Y: 2}) || r.Facing != world.East {
		t.Fatal("a one-tile change must animate as a step")
	}
	s.Tick(StepDuration)
	if r.Moving() {
		t.Fatal("the step must finish after StepDuration")
	}
	s.Apply(&proto.PlayerState{ID: 9, X: 9, Y: 9, Name: "Raymon"})
	if r.Moving() || r.Pos != (world.Point{X: 9, Y: 9}) {
		t.Fatal("a long jump (reconnect, correction) must snap, not slide across the map")
	}
	s.Apply(&proto.PlayerLeft{ID: 9})
	if s.Players[9] != nil {
		t.Fatal("PlayerLeft must remove the player")
	}
}

func TestOwnStateReconcilesTheWalker(t *testing.T) {
	s := welcomed(t)
	s.Me.Facing = world.West
	hold(s.Me, world.West, StepDuration)
	idle(s.Me, StepDuration)
	if s.Me.Pos != (world.Point{X: 4, Y: 5}) || s.Me.PendingMoves() != 1 {
		t.Fatalf("setup: %v pending %d", s.Me.Pos, s.Me.PendingMoves())
	}
	s.Apply(&proto.PlayerState{ID: 7, X: 5, Y: 5, Facing: world.West, Seq: 1, Name: "me"})
	if s.Me.Pos != (world.Point{X: 5, Y: 5}) || s.Players[7] != nil {
		t.Fatalf("own refused move: pos %v, and we must not draw ourselves as a remote", s.Me.Pos)
	}
}

func TestTileUpdateAndError(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.TileUpdate{X: 4, Y: 5, Layer: world.Object, Tile: world.Fence})
	if s.World.At(world.Object, 4, 5) != world.Fence {
		t.Fatal("TileUpdate not applied")
	}
	s.Apply(&proto.TileUpdate{X: 4, Y: 5, Layer: world.Object, Tile: world.None})
	if s.World.At(world.Object, 4, 5) != world.None {
		t.Fatal("a None TileUpdate must clear the tile")
	}
	s.Apply(&proto.Error{Code: proto.ErrNotAllowed})
	if s.Notice == "" {
		t.Fatal("an Error must leave a notice for the HUD")
	}
}

func TestChatBumpsChatSeq(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.Chat{Text: "Primera instrucción"})
	if s.ChatSeq != 1 || s.Chat != "Primera instrucción" {
		t.Fatalf("Chat not recorded: seq %d text %q", s.ChatSeq, s.Chat)
	}
	s.Apply(&proto.Chat{Text: "Segunda instrucción"})
	if s.ChatSeq != 2 || s.Chat != "Segunda instrucción" {
		t.Fatalf("Chat seq must bump: got %d %q", s.ChatSeq, s.Chat)
	}
}

func TestWelcomeCreatesFog(t *testing.T) {
	s := welcomed(t)
	if s.Fog == nil {
		t.Fatal("Welcome must create a fog mask from the map bounds")
	}
	if s.Fog.Bounds() != (world.Rect{W: 10, H: 10}) {
		t.Fatalf("fog bounds %v, want map bounds", s.Fog.Bounds())
	}
}

func TestMapMessageRestoresFog(t *testing.T) {
	s := welcomed(t)
	bits := make([]byte, (10*10+7)/8)
	bits[0] = 0xff // first 8 tiles seen
	s.Apply(&proto.Map{Bounds: world.Rect{W: 10, H: 10}, Bits: bits})
	if !s.Fog.Seen(0, 0) {
		t.Fatal("Map must restore seen tiles")
	}
	if s.Fog.Seen(9, 9) {
		t.Fatal("unseen tiles must stay hidden")
	}
}

func TestOwnPlayerStateRevealsFog(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.PlayerState{ID: 7, X: 8, Y: 8, Facing: world.South, Seq: 1, Name: "me"})
	if !s.Fog.Seen(8, 8) {
		t.Fatal("our own position from the server must reveal fog")
	}
}

func TestReconnectWelcomeResetsEverything(t *testing.T) {
	s := welcomed(t)
	s.Apply(&proto.PlayerState{ID: 9, X: 2, Y: 2, Name: "Raymon"})
	s.Me.Facing = world.West
	hold(s.Me, world.West, StepDuration)
	s.Apply(&proto.Welcome{ID: 11, X: 5, Y: 5, Bounds: world.Rect{W: 10, H: 10}})
	if s.MyID != 11 || len(s.Players) != 0 || s.Me.PendingMoves() != 0 || s.Me.Pos != (world.Point{X: 5, Y: 5}) {
		t.Fatalf("after a new Welcome: id %d players %d pending %d pos %v", s.MyID, len(s.Players), s.Me.PendingMoves(), s.Me.Pos)
	}
}

func TestDisplayNameFoldsToTheDebugFont(t *testing.T) {
	if got := DisplayName("Raymón Ñandú"); got != "Raymon Nandu" {
		t.Fatalf("DisplayName = %q", got)
	}
}

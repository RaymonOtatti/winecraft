package filestore

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func sample() *game.Saved {
	return &game.Saved{
		Version: game.SaveVersion,
		SavedAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		Chunks:  []game.SavedChunk{{X: -1, Y: 2, Ground: make([]byte, 2048), Object: make([]byte, 2048)}},
		Regrow:  []game.SavedRegrow{{X: 4, Y: 5, When: time.Date(2026, 10, 1, 12, 2, 0, 0, time.UTC)}},
		Profiles: map[string]game.Profile{
			"0123456789abcdef0123456789abcdef": {Name: "Raymón", X: 3, Y: 4, Facing: world.West, Inv: map[world.ItemID]int{world.ItemGrapes: 6}},
		},
	}
}

func TestSaveThenLoadGivesBackTheSameGame(t *testing.T) {
	st := New(filepath.Join(t.TempDir(), "world.json"))
	want := sample()
	if err := st.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestNoSaveYetIsNotAnError(t *testing.T) {
	got, err := New(filepath.Join(t.TempDir(), "world.json")).Load()
	if got != nil || err != nil {
		t.Fatalf("Load before any Save = %v, %v; want nil, nil", got, err)
	}
}

func TestACorruptSaveIsAnErrorNotAnEmptyWorld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.json")
	os.WriteFile(path, []byte(`{"Version":1,"Chunks":[{`), 0o600)
	if _, err := New(path).Load(); err == nil {
		t.Fatal("a truncated file must fail loudly; starting empty would overwrite real progress on the next save")
	}
}

func TestSaveReplacesTheFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "world.json")
	st := New(path)
	first := sample()
	st.Save(first)
	// a crash during an earlier write left a half-written temp file behind
	os.WriteFile(filepath.Join(dir, ".world.json-crashed.tmp"), []byte("{garbage"), 0o600)
	second := sample()
	second.Profiles["0123456789abcdef0123456789abcdef"] = game.Profile{Name: "nuevo"}
	if err := st.Save(second); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil || got.Profiles["0123456789abcdef0123456789abcdef"].Name != "nuevo" {
		t.Fatalf("after the second save: %v, %v", got, err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Fatalf("save file mode %v: player tokens are secrets, keep it 0600", fi.Mode().Perm())
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".world.json-*.tmp"))
	if len(matches) != 1 { // only the old crashed one; ours was renamed into place
		t.Fatalf("temp files left: %v", matches)
	}
}

// Package filestore saves the game to one JSON file. It implements
// game.Store; SQLite will be a second adapter behind the same port.
package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/RaymonOtatti/winecraft/internal/game"
)

// Store keeps the game at Path.
type Store struct{ Path string }

// New returns a store for the file at path.
func New(path string) *Store { return &Store{Path: path} }

var _ game.Store = (*Store)(nil)

// Load reads the saved game. A missing file means nothing was saved yet and
// returns (nil, nil); an unreadable one is an error, because starting empty
// would overwrite real progress on the next save.
func (s *Store) Load() (*game.Saved, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sv game.Saved
	if err := json.Unmarshal(b, &sv); err != nil {
		return nil, fmt.Errorf("%s is damaged: %w (move it aside to start a new world)", s.Path, err)
	}
	return &sv, nil
}

// Save writes the game atomically: a temp file in the same directory,
// flushed to disk, then renamed over the old one. A crash at any point
// leaves either the old save or the new one, never half of each. The file
// is 0600 because it holds player tokens.
func (s *Store) Save(sv *game.Saved) error {
	b, err := json.Marshal(sv)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(s.Path)+"-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op once renamed
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), s.Path); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil { // make the rename itself durable
		d.Sync()
		d.Close()
	}
	return nil
}

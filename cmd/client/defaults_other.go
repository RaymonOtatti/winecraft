//go:build !(js && wasm)

package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/RaymonOtatti/winecraft/internal/client"
)

// defaults: natively there is no page, so flags say everything.
func defaults() startup { return startup{token: "auto"} }

// prefs keeps settings as small files next to the token.
type prefs struct{ dir string }

func newPrefs() client.Prefs {
	dir, err := os.UserConfigDir()
	if err != nil {
		return client.MemPrefs{}
	}
	return prefs{dir: filepath.Join(dir, "winecraft", "prefs")}
}

func (p prefs) Get(k string) string {
	b, _ := os.ReadFile(filepath.Join(p.dir, k))
	return strings.TrimSpace(string(b))
}

func (p prefs) Set(k, v string) {
	if os.MkdirAll(p.dir, 0o700) == nil {
		os.WriteFile(filepath.Join(p.dir, k), []byte(v+"\n"), 0o600)
	}
}

// deviceToken returns this device's player token, creating it on first use.
// It lives in the user config dir (0600): it is what keeps your progress.
func deviceToken() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("no config dir, playing anonymously: %v", err)
		return ""
	}
	path := filepath.Join(dir, "winecraft", "token")
	if b, err := os.ReadFile(path); err == nil {
		return strings.TrimSpace(string(b))
	}
	var raw [16]byte
	rand.Read(raw[:])
	tok := hex.EncodeToString(raw[:])
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
		os.WriteFile(path, []byte(tok+"\n"), 0o600)
	}
	return tok
}

//go:build !(js && wasm)

package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// defaults: natively there is no page, so flags say everything.
func defaults() startup { return startup{token: "auto"} }

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

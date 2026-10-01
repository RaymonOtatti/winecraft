package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Handler routes the game's HTTP surface: the WebSocket, a health check,
// and the web client's static files.
func Handler(h *Hub, webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", h.serveWS)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "ok %d online\n", h.Online())
	})
	mux.Handle("GET /", newStatic(webDir))
	return mux
}

// static serves web/ and sends .wasm files gzip-compressed (a Go wasm build
// is ~14 MB raw, ~3.5 MB gzipped). The compressed bytes are cached until the
// file changes.
type static struct {
	dir   http.Dir
	files http.Handler

	mu sync.Mutex
	gz map[string]gzEntry
}

type gzEntry struct {
	mod  time.Time
	size int64
	data []byte
}

func newStatic(dir string) *static {
	return &static{dir: http.Dir(dir), files: http.FileServer(http.Dir(dir)), gz: make(map[string]gzEntry)}
}

func (s *static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The client is under active development: always revalidate.
	w.Header().Set("Cache-Control", "no-cache")
	if !strings.HasSuffix(r.URL.Path, ".wasm") {
		s.files.ServeHTTP(w, r)
		return
	}
	f, err := s.dir.Open(r.URL.Path) // http.Dir rejects paths that escape dir
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Add("Vary", "Accept-Encoding")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		http.ServeContent(w, r, st.Name(), st.ModTime(), f)
		return
	}
	data, err := s.gzipped(r.URL.Path, f, st.ModTime(), st.Size())
	if err != nil {
		http.Error(w, "compress failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	http.ServeContent(w, r, st.Name(), st.ModTime(), bytes.NewReader(data))
}

func (s *static) gzipped(path string, f io.Reader, mod time.Time, size int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.gz[path]; ok && e.mod.Equal(mod) && e.size == size {
		return e.data, nil
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := io.Copy(zw, f); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	s.gz[path] = gzEntry{mod: mod, size: size, data: buf.Bytes()}
	return buf.Bytes(), nil
}

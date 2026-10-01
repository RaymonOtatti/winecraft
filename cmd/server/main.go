// Command server runs the WineCraft game server: the shared world, the
// WebSocket endpoint and the web client's files.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/game"
	"github.com/RaymonOtatti/winecraft/internal/server"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address (keep it on loopback; a tunnel publishes it)")
	webDir := flag.String("web", "web", "directory with index.html, wasm_exec.js and winecraft.wasm")
	joinCode := flag.String("join", "", "join code players must enter (default: a random one, printed at start)")
	seed := flag.Uint64("seed", 1, "dev map seed")
	maxPlayers := flag.Int("max-players", 8, "maximum players online")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if *joinCode == "" {
		*joinCode = randomJoinCode()
	}

	state := game.New(world.GenerateDevMap(*seed))
	state.MaxPlayers = *maxPlayers
	hub := server.NewHub(state, server.Config{JoinCode: *joinCode, Log: log})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	hubCtx, stopHub := context.WithCancel(context.Background())
	go hub.Run(hubCtx)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(hub, *webDir),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		stopHub()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()

	fmt.Printf("WineCraft server on http://%s  join code: %s\n", *addr, *joinCode)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// randomJoinCode returns a code like "malbec-482913": easy to read aloud,
// about 23 bits of entropy, and rate-limited on the server.
func randomJoinCode() string {
	words := []string{"malbec", "cabernet", "torrontes", "semillon", "bonarda", "syrah", "merlot", "pinot", "chardonnay", "tempranillo"}
	w, _ := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	n, _ := rand.Int(rand.Reader, big.NewInt(1_000_000))
	return fmt.Sprintf("%s-%06d", words[w.Int64()], n.Int64())
}

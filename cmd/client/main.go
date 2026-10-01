// Command client is the WineCraft game client. Natively it opens a window;
// built with GOOS=js GOARCH=wasm it runs in the browser (see web/).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/RaymonOtatti/winecraft/internal/client"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func main() {
	def := defaults() // the page URL in the browser, nothing natively
	server := flag.String("server", def.server, "game server WebSocket URL, e.g. ws://127.0.0.1:8787/ws (empty: play the dev map offline)")
	name := flag.String("name", def.name, "your player name")
	code := flag.String("code", def.code, "the server's join code")
	snap := flag.String("snap", "", "save a frame as a PNG at this path and quit")
	seed := flag.Uint64("seed", 1, "dev map seed (offline mode)")
	at := flag.String("at", "", "start at tile x,y instead of spawn (offline snapshots)")
	walk := flag.String("walk", "", `scripted route for snapshots, e.g. "W4,N2"`)
	touch := flag.Bool("touch", false, "show the touch D-pad without a touch screen")
	stay := flag.Duration("stay", 0, "keep playing this long after -snap before quitting")
	flag.Parse()

	var g *client.Game
	if *server == "" {
		m := world.GenerateDevMap(*seed)
		pos := m.Spawn
		if *at != "" {
			if _, err := fmt.Sscanf(*at, "%d,%d", &pos.X, &pos.Y); err != nil {
				log.Fatalf("-at wants x,y: %v", err)
			}
		}
		g = client.NewGame(client.OfflineSession(m, pos), nil)
	} else {
		if *name == "" || *code == "" {
			log.Fatal("playing on a server needs -name and -code")
		}
		n := client.StartNet(context.Background(), client.NetConfig{URL: *server, Name: *name, JoinCode: *code})
		g = client.NewGame(client.NewSession(), n)
		g.MyName = *name
	}
	g.SnapPath = *snap
	g.ShowDpad = *touch
	g.Stay = *stay
	if *walk != "" {
		if err := g.Script(*walk); err != nil {
			log.Fatal(err)
		}
	}

	ebiten.SetWindowTitle("WineCraft — Valle de Uco")
	ebiten.SetWindowSize(2*client.BaseW, 2*client.BaseH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}

type startup struct{ server, name, code string }

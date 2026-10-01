// Command client is the WineCraft game client. Natively it opens a window;
// built with GOOS=js GOARCH=wasm it runs in the browser (see web/).
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/RaymonOtatti/winecraft/internal/client"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func main() {
	snap := flag.String("snap", "", "save frame 30 as a PNG at this path and quit")
	seed := flag.Uint64("seed", 1, "dev map seed (offline mode)")
	at := flag.String("at", "", "start at tile x,y instead of spawn (for snapshots)")
	flag.Parse()

	// Offline mode until networking lands (BUILD.md B4.3): the dev map, at spawn.
	m := world.GenerateDevMap(*seed)
	pos := m.Spawn
	if *at != "" {
		if _, err := fmt.Sscanf(*at, "%d,%d", &pos.X, &pos.Y); err != nil {
			log.Fatalf("-at wants x,y: %v", err)
		}
	}
	g := client.NewGame(m.World, pos)
	g.SnapPath = *snap

	ebiten.SetWindowTitle("WineCraft — Valle de Uco")
	ebiten.SetWindowSize(2*client.BaseW, 2*client.BaseH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}

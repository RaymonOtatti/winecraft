package client

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// BenchmarkDraw measures the CPU time and allocations of a single frame.
// It uses the offline dev map and a small Ebitengine screen so the benchmark
// does not depend on a real window.
func BenchmarkDraw(b *testing.B) {
	m := world.GenerateDevMap(1)
	s := OfflineSession(m, m.Spawn)
	s.Me.Facing = world.South
	g := NewGame(s, nil)
	g.w, g.h = BaseW, BaseH
	screen := ebiten.NewImage(BaseW, BaseH)

	// Warm up: create any lazily-allocated caches.
	for i := 0; i < 5; i++ {
		screen.Fill(voidColor)
		g.Draw(screen)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.Fill(voidColor)
		g.Draw(screen)
	}
}

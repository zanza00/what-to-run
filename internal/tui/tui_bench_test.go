package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// benchSink keeps the rendered frame alive, so that the compiler cannot drop
// the work under test.
var benchSink string

// benchmarkModel returns a filtered model with enough tasks that the ranking
// cost shows up in the frame time.
func benchmarkModel() model {
	tasks := make([]core.Task, 400)
	for i := range tasks {
		name := fmt.Sprintf("build-%03d", i)
		tasks[i] = core.Task{
			Source: "package.json",
			Name:   name,
			Desc:   "Compile the project with the production settings",
			Args:   []string{"npm", "run", name},
		}
	}
	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 120, Height: 40})
	return press(m, "build")
}

// BenchmarkRenderFilteredCached measures a frame that reuses the snapshot.
func BenchmarkRenderFilteredCached(b *testing.B) {
	m := benchmarkModel()
	b.ReportAllocs()
	for b.Loop() {
		benchSink = m.View().Content
	}
}

// BenchmarkRenderFilteredRecompute measures the same frame when every read
// rebuilds the results, as the picker did before the snapshot.
func BenchmarkRenderFilteredRecompute(b *testing.B) {
	m := benchmarkModel()
	b.ReportAllocs()
	for b.Loop() {
		m.snapshot.valid = false
		benchSink = m.View().Content
	}
}

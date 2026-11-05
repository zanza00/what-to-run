package tui

// The rendering benchmarks of the picker: the frames, the filter, the
// ranking and the single units of the row algebra. They are manual
// instrumentation, never part of `task all`; run them with
// `go test ./internal/tui -bench . -benchmem`. The source first measured
// the 2026-11-04 review (notes/arch-bench-2026-11.md); the row cache kept
// the suite and added its own legs: the steady frame against the first
// frame of an era.

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// Sinks keep the results alive, so the compiler cannot drop the work.
var (
	frameSink string
	rowSink   string
	maskSink  []bool
	listSink  []string
	tasksSink []core.Task
)

// benchTasks returns n deterministic tasks with the shape of a real project:
// three kinds of names and descriptions, four source files. The command line
// is the joined Args, as CommandLineFor reports it.
func benchTasks(n int) []core.Task {
	sources := []string{"package.json", "Makefile", "justfile", "Taskfile.yml"}
	tasks := make([]core.Task, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("build-%03d", i)
		desc := "Compile the project with the production settings"
		args := []string{"npm", "run", name}
		switch i % 3 {
		case 1:
			name = fmt.Sprintf("test-%03d", i)
			desc = "Run the unit tests with the race detector and the shuffle on"
		case 2:
			name = fmt.Sprintf("lint-%03d", i)
			desc = "Check the sources"
		}
		tasks = append(tasks, core.Task{
			Source: sources[i%len(sources)],
			Name:   name,
			Desc:   desc,
			Args:   args,
		})
	}
	return tasks
}

// benchWarning is the warning that opens the warning modal at startup.
func benchWarning() core.Warning {
	return core.NewSourceWarning("make", "Makefile", "a parse warning for the benchmark run")
}

// benchModel returns a ready model. Fullscreen takes the whole screen; inline
// starts on row 5 of a 40-row terminal, as the shell widgets place it.
func benchModel(b testing.TB, inline bool, tasks []core.Task, w int, warning bool) model {
	opts := Options{}
	if inline {
		opts.Inline = true
	}
	var warnings []core.Warning
	if warning {
		warnings = []core.Warning{benchWarning()}
	}
	m := update(newModel(opts, tasks, warnings), tea.WindowSizeMsg{Width: w, Height: 40})
	if inline {
		m = update(m, tea.CursorPositionMsg{Y: 5})
	}
	if warning && !m.warningModal {
		b.Fatal("the warning modal did not open")
	}
	return m
}

// benchSizes is the matrix of the full-frame benchmark.
var benchSizes = []int{10, 100, 1000}

// benchWidths is the terminal width dimension of the matrix.
var benchWidths = []int{80, 120, 200}

// BenchmarkRenderFullscreen measures the whole frame in fullscreen: View,
// render, the visible window, the help line and, with the modal open, the
// box overlay.
func BenchmarkRenderFullscreen(b *testing.B) {
	for _, n := range benchSizes {
		for _, w := range benchWidths {
			for _, modal := range []bool{false, true} {
				b.Run(fmt.Sprintf("tasks=%d/width=%d/warnmodal=%v", n, w, modal), func(b *testing.B) {
					m := benchModel(b, false, benchTasks(n), w, modal)
					b.ReportAllocs()
					for b.Loop() {
						frameSink = m.View().Content
					}
				})
			}
		}
	}
}

// BenchmarkRenderInline measures the whole frame in inline mode, where the
// list gets the rows between the prompt and the bottom of the screen.
func BenchmarkRenderInline(b *testing.B) {
	for _, n := range benchSizes {
		for _, w := range benchWidths {
			for _, modal := range []bool{false, true} {
				b.Run(fmt.Sprintf("tasks=%d/width=%d/warnmodal=%v", n, w, modal), func(b *testing.B) {
					m := benchModel(b, true, benchTasks(n), w, modal)
					b.ReportAllocs()
					for b.Loop() {
						frameSink = m.View().Content
					}
				})
			}
		}
	}
}

// BenchmarkRenderFiltered measures a frame with a filter, so every visible row
// derives the match mask and paints the segments.
func BenchmarkRenderFiltered(b *testing.B) {
	for _, n := range benchSizes {
		for _, w := range benchWidths {
			b.Run(fmt.Sprintf("tasks=%d/width=%d", n, w), func(b *testing.B) {
				m := benchModel(b, false, benchTasks(n), w, false)
				m.setFilter("build")
				b.ReportAllocs()
				for b.Loop() {
					frameSink = m.View().Content
				}
			})
		}
	}
}

// BenchmarkTaskRow measures one row alone: the derivation and the paint of
// the unselected and the selected row, with and without a filter to
// highlight.
func BenchmarkTaskRow(b *testing.B) {
	m := benchModel(b, false, benchTasks(100), 120, false)
	t := m.list.visible()[0]
	b.Run("plain", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d := deriveRow(t, m.shell, m.list.filterTokens(), m.list.commandWidth(), 120)
			rowSink = m.taskRow(t, d, false, 120)
		}
	})
	b.Run("selected", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d := deriveRow(t, m.shell, m.list.filterTokens(), m.list.commandWidth(), 120)
			rowSink = m.taskRow(t, d, true, 120)
		}
	})
	mf := benchModel(b, false, benchTasks(100), 120, false)
	mf.setFilter("build")
	tf := mf.list.visible()[0]
	b.Run("filtered", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d := deriveRow(tf, mf.shell, mf.list.filterTokens(), mf.list.commandWidth(), 120)
			rowSink = mf.taskRow(tf, d, false, 120)
		}
	})
	b.Run("selected-filtered", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d := deriveRow(tf, mf.shell, mf.list.filterTokens(), mf.list.commandWidth(), 120)
			rowSink = mf.taskRow(tf, d, true, 120)
		}
	})
}

// BenchmarkRowCacheSteadyFrame measures the frame that a steady key move
// costs: the open era already holds the derivations of the window, so the
// render reads them and paints them under the styles of the frame.
func BenchmarkRowCacheSteadyFrame(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("tasks=%d/width=%d", n, 120), func(b *testing.B) {
			m := benchModel(b, false, benchTasks(n), 120, false)
			m.setFilter("build")
			frameSink = m.View().Content // the first frame of the era fills the cache
			b.ReportAllocs()
			for b.Loop() {
				frameSink = m.View().Content
			}
		})
	}
}

// BenchmarkRowCacheFirstFrame measures the frame that a filter keystroke
// costs: the bump of the generation ends the era, so the render derives the
// visible window again.
func BenchmarkRowCacheFirstFrame(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("tasks=%d/width=%d", n, 120), func(b *testing.B) {
			m := benchModel(b, false, benchTasks(n), 120, false)
			m.setFilter("build")
			b.ReportAllocs()
			for b.Loop() {
				first := m
				first.rowCache = &rowCache{}
				frameSink = first.View().Content
			}
		})
	}
}

// BenchmarkTruncateMatched measures the truncation helper alone.
func BenchmarkTruncateMatched(b *testing.B) {
	cmd := "npm run build-004"
	long := strings.Repeat("build the production bundle with the sourcemap and the checksum ", 3)
	tokens := []string{"bui"}
	matched := matchPositions(long, tokens)
	b.Run("fits", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink, maskSink = truncateMatched(cmd, nil, 40)
		}
	})
	b.Run("truncate-unmatched", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink, maskSink = truncateMatched(long, nil, 24)
		}
	})
	b.Run("truncate-matched", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink, maskSink = truncateMatched(long, matched, 24)
		}
	})
}

// BenchmarkMatchPositions measures the per-row mask derivation that the row
// cache runs for the command line and the description of a visible row.
func BenchmarkMatchPositions(b *testing.B) {
	cmd := "npm run build-004"
	desc := "Compile the project with the production settings"
	for _, tokens := range [][]string{{"bui"}, {"bui", "prod"}} {
		b.Run(fmt.Sprintf("tokens=%d", len(tokens)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				maskSink = matchPositions(cmd, tokens)
				maskSink = matchPositions(desc, tokens)
			}
		})
	}
}

// BenchmarkSetFilter measures the per-keystroke filter cost: tokenize, score
// and rank every task, then measure the command column.
func BenchmarkSetFilter(b *testing.B) {
	for _, n := range benchSizes {
		for _, text := range []string{"build", "build prod"} {
			b.Run(fmt.Sprintf("tasks=%d/filter=%q", n, text), func(b *testing.B) {
				m := benchModel(b, false, benchTasks(n), 120, false)
				b.ReportAllocs()
				for b.Loop() {
					m.setFilter(text)
				}
			})
		}
	}
}

// BenchmarkRankTasks measures the pure filter core: subsequence scoring of
// every task and the stable sort of the matches. The model of the leg carries
// an empty filter, so its ranked slice holds every task.
func BenchmarkRankTasks(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("tasks=%d", n), func(b *testing.B) {
			m := benchModel(b, false, benchTasks(n), 120, false)
			tokens := strings.Fields("build prod")
			b.ReportAllocs()
			for b.Loop() {
				tasksSink = rankTasks(m.list.visible(), tokens, m.shell)
			}
		})
	}
}

// BenchmarkListLines measures the visible window of the task list alone. The
// frame counts the list rows for the boxes instead of rendering them twice,
// so an open modal adds no second pass here. With the row cache, the first
// call of an era fills the derivations and the rest read them, so the loop
// measures the steady frame of the list.
func BenchmarkListLines(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("tasks=%d", n), func(b *testing.B) {
			m := benchModel(b, false, benchTasks(n), 120, false)
			b.ReportAllocs()
			for b.Loop() {
				listSink = m.listLines(120)
			}
		})
	}
}

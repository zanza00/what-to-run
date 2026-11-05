# TUI rendering benchmarks (2026-11-04)

The throwaway benchmark suite the measurement agent wrote for the
2026-11-04 review ([arch-review-2026-11](arch-review-2026-11.md)). It lived
in a detached worktree and is gone; this page preserves the source, so a
future piece can rebuild the same benchmarks and compare against the same
baselines (first candidates: the FP2 and FP5 pieces). Method: medians,
benchstat, `-count=10 -benchtime=0.5s -benchmem`, Go 1.27.3, linux/amd64,
Intel N150 — ratios transfer, nanoseconds do not. The widget protocol needs
a program loop for anything beyond these pure paths; the repo rule forbids
PTY tests.

2026-11-05: the suite no longer needs rebuilding — the row-cache piece
landed it at the real seams (PR #130, merge commit `35ba523`) as
`internal/tui/render_bench_test.go`: the benchmark names kept where the
workload survived, the inserts adapted to the merged modules (the list
module, the display module, the typed warnings), plus
`BenchmarkRowCacheSteadyFrame` and `BenchmarkRowCacheFirstFrame`. The
paired benchstat table of that piece lives in [row-cache](row-cache.md).
The source block below stays as the 2026-11-04 record of the suite at HEAD
`cd1e933`, preserved verbatim; its pre-merge inserts are the page's own
record, not the landed file.

## Baselines (medians)

| Benchmark | 10 tasks | 100 tasks | 1000 tasks |
| --- | --- | --- | --- |
| Fullscreen render w80/120/200, modal closed | 152/145/165 µs | 450/443/497 µs | 444/433/478 µs |
| Fullscreen render, modal open | 0.94/1.18/1.84 ms | 1.82/2.13/3.09 ms | 1.79/2.07/3.17 ms |
| Inline render, modal closed | 146/146/160 µs | 377/374/415 µs | 376/368/421 µs |
| Inline render, modal open | 0.84/1.05/1.61 ms | 1.50/1.78/2.59 ms | 1.52/1.87/2.81 ms |
| Filtered frame w80/120/200 | 224/221/241 µs | 753/732/789 µs | 760/757/790 µs |
| setFilter, "build" | 45.6 µs | 473 µs | 4.83 ms |
| setFilter, "build prod" | 65.3 µs | 652 µs | 6.71 ms |
| rankTasks (2 tokens) | 61.5 µs | 620 µs | 6.31 ms |
| listLines (w120, 36-row window) | 103 µs | 356 µs | 348 µs |

Single units: taskRow plain 8.78 µs, selected 16.15 µs, filtered 16.11 µs,
selected-filtered 29.28 µs. truncateMatched: 321 ns (fits), 5.03 µs
(truncate unmatched), 5.24 µs (truncate matched). matchPositions over a
cmd+desc pair: 2.18 µs (1 token), 3.05 µs (2 tokens). Memory: a
modal-closed frame at 1000 tasks/w200 = 41.6 KiB/op, 762 allocs; a
modal-open frame = 1.10 MiB/op, 2228 allocs; a filtered frame 211-217
KiB/op. Pre-existing suite: RenderFilteredCached 694 µs vs
RenderFilteredRecompute 66.6 ms at 400 tasks (96×).

## Profile highlights

- Modal-open frame, 1000 tasks, w200 (3.35 ms, 1.16 MB, 2228 allocs):
  CPU cumulative `render` 87.2% → `overlay` 63.6% → `Compositor.Render`
  46.5% → `ansi.StringWidth` 29.6% → `ansi.FirstGraphemeCluster` 25.5% →
  `strings.splitSeq` 23.9% → `Canvas.Compose`/`Compositor.Draw` 23.1%.
  Allocated bytes: `ultraviolet.NewBuffer` 83.1% (the compositor's cell
  grid, ~135 B per cell); by objects `taskRow` 59.2% cum, the ansi
  `Style.ForegroundColor` string building 41.4% cum.
- Modal-closed frame (478 µs): `taskRow` 59.3% cum; `ansi.stringWidth`
  59.0%; `FirstGraphemeCluster` 50.6%; lipgloss `Style.Render` 33.3%.
  Line-level inside `taskRow`: the description's
  `truncateMatched`+`matchPositions` (tui.go:1485) 250 ms of 2.1 s;
  `renderMatched(cmd)` (tui.go:1509) 220 ms; `lipgloss.Width(cmd)` +
  `strings.Repeat` (tui.go:1470) 90 ms; the command truncation
  (tui.go:1469) 80 ms; the selected styling (tui.go:1497-1506) ~120 ms.
- Keystroke path (`setFilter`, 1000 tasks, 4.8 ms): `rebuildSnapshot`
  68.8% cum; `scoreSubsequence` 40.7% (its DP `subsequenceMatch` 30.7%);
  GC ~15%; `commandWidthFor`'s `lipgloss.Width` pass ~10%.

## Source

```go
package tui

// Throwaway benchmarks for the architecture review. They size the render cost
// of the picker at HEAD; they are not part of the test suite.

import (
	"errors"
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

// benchWarn is the warning that opens the warning modal at startup.
func benchWarn() error {
	return errors.New("a parse warning for the benchmark run")
}

// benchModel returns a ready model. Fullscreen takes the whole screen; inline
// starts on row 5 of a 40-row terminal, as the shell widgets place it.
func benchModel(b testing.TB, inline bool, tasks []core.Task, w int, warning bool) model {
	opts := Options{}
	if inline {
		opts.Inline = true
	}
	var warnings []error
	if warning {
		warnings = []error{benchWarn()}
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

// BenchmarkTaskRow measures one row alone: the unselected and the selected
// layout, with and without a filter to highlight.
func BenchmarkTaskRow(b *testing.B) {
	m := benchModel(b, false, benchTasks(100), 120, false)
	t := m.visible()[0]
	b.Run("plain", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink = m.taskRow(t, false, 120)
		}
	})
	b.Run("selected", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink = m.taskRow(t, true, 120)
		}
	})
	mf := benchModel(b, false, benchTasks(100), 120, false)
	mf.setFilter("build")
	tf := mf.visible()[0]
	b.Run("filtered", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink = mf.taskRow(tf, false, 120)
		}
	})
	b.Run("selected-filtered", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rowSink = mf.taskRow(tf, true, 120)
		}
	})
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

// BenchmarkMatchPositions measures the per-row mask derivation that taskRow
// runs for the command line and the description of every visible row.
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
// every task and the stable sort of the matches.
func BenchmarkRankTasks(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("tasks=%d", n), func(b *testing.B) {
			m := benchModel(b, false, benchTasks(n), 120, false)
			tokens := strings.Fields("build prod")
			b.ReportAllocs()
			for b.Loop() {
				tasksSink = rankTasks(m.tasks, tokens, m.shell)
			}
		})
	}
}

// BenchmarkListLines measures the visible window of the task list alone. The
// warning modal placement re-runs this on every frame that the modal is open.
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
```

To rebuild: save the source as `internal/tui/render_bench_test.go` (it uses
only the package's own helpers; no name collides at HEAD `cd1e933`), then
`go test ./internal/tui -bench . -benchmem`. Delete it or land it at the
real seams when the piece is done.

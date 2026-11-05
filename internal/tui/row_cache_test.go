package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// The table for the row cache. The pins are hand-computed examples of the
// derivation algebra, not values that the test derives through the same code.
// The cache hands a derivation to the painter of one row, and the painter
// paints it under the styles of the frame, so a stale derivation must show
// up in the rendered rows. Every leg below breaks when the cache hands back
// a derivation of an old era.

// windowLegTasks is the task count of the window leg of the cache table: 20
// tasks, so a list height of 5 leaves 15 rows that the one frame of the leg
// never shows.
const windowLegTasks = 20

// poisonDerivation is a derivation that the algebra can never produce. A test
// stores it in the cache and then pins whether the render shows it: the second
// read of an era must show it, and a read of a new era must not.
func poisonDerivation() rowDerivations {
	return rowDerivations{cmd: "cache-poison"}
}

// checkDerivations pins one derivation against the hand-computed example. The
// masks are compared with DeepEqual, which keeps a nil mask apart from an
// empty one, so the pin holds the exact value that the cache stores.
func checkDerivations(t *testing.T, got, want rowDerivations) {
	t.Helper()
	if got.badgeW != want.badgeW {
		t.Errorf("badgeW = %d, want %d", got.badgeW, want.badgeW)
	}
	if got.cmdW != want.cmdW {
		t.Errorf("cmdW = %d, want %d", got.cmdW, want.cmdW)
	}
	if got.cmd != want.cmd {
		t.Errorf("cmd = %q, want %q", got.cmd, want.cmd)
	}
	if !reflect.DeepEqual(got.cmdMask, want.cmdMask) {
		t.Errorf("cmdMask = %v, want %v", got.cmdMask, want.cmdMask)
	}
	if got.descLine != want.descLine {
		t.Errorf("descLine = %q, want %q", got.descLine, want.descLine)
	}
	if !reflect.DeepEqual(got.descMask, want.descMask) {
		t.Errorf("descMask = %v, want %v", got.descMask, want.descMask)
	}
}

// cacheTestModel returns a fullscreen model of tasks sized to an 80×30
// terminal, the shape that every leg below renders in.
func cacheTestModel(tasks []core.Task) model {
	return update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 30})
}

// cacheTestTask returns one task whose derivations the examples below pin: a
// badge of 12 columns, a command line of 14 columns and a description of 19
// columns.
func cacheTestTask() core.Task {
	return core.Task{
		Source: "package.json",
		Name:   "build",
		Desc:   "Compile the project",
		Args:   []string{"npm", "run", "build"},
	}
}

// contained reports whether the rendered row shows the text, whatever the
// styles around it.
func contained(row, text string) bool {
	return strings.Contains(row, text)
}

// TestRowCacheFill pins the first read of an era: the render of a frame fills
// the derivation through the one derivation function, and the worked example
// comes from the algebra by hand. The badge of "package.json" is 12 columns
// wide, so a row at width 40 keeps 27 columns left of the badge separator and
// 25 after the marker; the command of 13 columns does not pad, and the
// description of 19 columns truncates to "Compile th…" in the 11 columns that
// the description keeps.
func TestRowCacheFill(t *testing.T) {
	// The description of the example keeps 11 columns at width 40, and the
	// command keeps the whole 13, so the truncation keeps a full command and
	// an ellipsis-ended description.
	tasks := []core.Task{cacheTestTask()}
	m := cacheTestModel(tasks)
	m.listLines(40)

	d, hit := m.rowCache.lookup(era{gen: m.list.generation(), width: 40}, 0)
	if !hit {
		t.Fatal("the first render of an era did not fill the row")
	}
	checkDerivations(t, d, rowDerivations{
		badgeW:   12,
		cmdW:     13,
		cmd:      "npm run build",
		cmdMask:  nil,
		descLine: " Compile th…",
		descMask: nil,
	})
}

// TestRowCacheFillNarrow pins a second worked example, where the width pushes
// the derivation into its guarded branches: at width 20 the command column
// caps at 5 columns, so the command truncates to "npm …", and the description
// loses its room, so the row drops it.
func TestRowCacheFillNarrow(t *testing.T) {
	tasks := []core.Task{cacheTestTask()}
	m := cacheTestModel(tasks)
	m.listLines(20)

	d, hit := m.rowCache.lookup(era{gen: m.list.generation(), width: 20}, 0)
	if !hit {
		t.Fatal("the first render at width 20 did not fill the row")
	}
	checkDerivations(t, d, rowDerivations{
		badgeW:   12,
		cmdW:     5,
		cmd:      "npm …",
		cmdMask:  nil,
		descLine: "",
		descMask: nil,
	})
}

// TestRowCacheFillPad pins the pad of the command column: the widest command
// holds the column, so a shorter command pads to the same width, and the mask
// stays on the text before the pad.
func TestRowCacheFillPad(t *testing.T) {
	tasks := []core.Task{
		cacheTestTask(),
		{Source: "Makefile", Name: "test", Desc: "Run the tests", Args: []string{"make", "test"}},
	}
	m := cacheTestModel(tasks)
	m.listLines(80)

	d, hit := m.rowCache.lookup(era{gen: m.list.generation(), width: 80}, 1)
	if !hit {
		t.Fatal("the first render did not fill the second row")
	}
	checkDerivations(t, d, rowDerivations{
		badgeW:   8,
		cmdW:     13,
		cmd:      "make test    ",
		cmdMask:  nil,
		descLine: " Run the tests",
		descMask: nil,
	})
}

// TestRowCacheFillFiltered pins the masks of a filtered row: the filter token
// "test" marks the runes of the command and the description that carry it, and
// the description mask also carries the leading space of the description line.
func TestRowCacheFillFiltered(t *testing.T) {
	tasks := []core.Task{{
		Source: "Makefile",
		Name:   "test",
		Desc:   "Run the tests",
		Args:   []string{"go", "test"},
	}}
	m := cacheTestModel(tasks)
	m = press(m, "test")
	m.listLines(80)

	d, hit := m.rowCache.lookup(era{gen: m.list.generation(), width: 80}, 0)
	if !hit {
		t.Fatal("the first filtered render did not fill the row")
	}
	checkDerivations(t, d, rowDerivations{
		badgeW:   8,
		cmdW:     7,
		cmd:      "go test",
		cmdMask:  []bool{false, false, false, true, true, true, true},
		descLine: " Run the tests",
		descMask: []bool{false, false, false, false, false, false, false, false, false, true, true, true, true, false},
	})
}

// TestRowCacheHitNoRebuild pins the second read of an era: the render must
// return the stored derivation without a rebuild. The poison cannot come from
// the derivation function, so the row can only show it when the render reads
// the stored derivation of the second read.
func TestRowCacheHitNoRebuild(t *testing.T) {
	tasks := []core.Task{cacheTestTask()}
	m := cacheTestModel(tasks)
	m.listLines(40)
	m.rowCache.fill(era{gen: m.list.generation(), width: 40}, 0, poisonDerivation())

	rows := m.listLines(40)
	if len(rows) == 0 {
		t.Fatal("the render produced no rows")
	}
	if !contained(rows[0], "cache-poison") {
		t.Errorf("the second read rebuilt the row, want the stored derivation:\n%s", rows[0])
	}
}

// TestRowCacheFillOutsideTheWindow pins that the fill is lazy: the first
// frame of an era fills only the rows that it shows. A list of 20 tasks in a
// window of 5 rows fills 5 rows and leaves the 15 rows outside the window
// unfilled.
func TestRowCacheFillOutsideTheWindow(t *testing.T) {
	tasks := make([]core.Task, windowLegTasks)
	for i := range tasks {
		tasks[i] = core.Task{
			Source: "package.json",
			Name:   fmt.Sprintf("bench-%02d", i),
			Desc:   "Run the benchmarks",
			Args:   []string{"npm", "run", fmt.Sprintf("bench-%02d", i)},
		}
	}
	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 60, Height: 9})
	m.listLines(60)

	e := era{gen: m.list.generation(), width: 60}
	hits := 0
	for pos := 0; pos < windowLegTasks; pos++ {
		if _, ok := m.rowCache.lookup(e, pos); ok {
			hits++
		}
	}
	// The terminal of 9 rows keeps a list height of 5 after the chrome of 4.
	if hits != 5 {
		t.Errorf("the fill filled %d rows, want the 5 rows of the window", hits)
	}
}

// TestRowCacheEra empties the cache wholesale at a door bump. Each case
// stores a poison derivation of the current era, changes the era through its
// real door, and pins that the render refuses the poison: the rows show the
// fresh derivation instead. A stale cache would hand the poison back, and
// the pin would catch it.
func TestRowCacheEra(t *testing.T) {
	poison := poisonDerivation()
	tests := []struct {
		name string
		// bump changes the era through its real door.
		bump func(m *model)
		// want pins the fresh derivation of the first row after the bump.
		width int
		want  rowDerivations
	}{
		{
			name:  "filter",
			bump:  func(m *model) { m.setFilter("test") },
			width: 40,
			// The filter token keeps its masks on both lines.
			want: rowDerivations{
				badgeW:   8,
				cmdW:     7,
				cmd:      "go test",
				cmdMask:  []bool{false, false, false, true, true, true, true},
				descLine: " Run the tests",
				descMask: []bool{false, false, false, false, false, false, false, false, false, true, true, true, true, false},
			},
		},
		{
			name: "tasks",
			bump: func(m *model) {
				m.setTasks([]core.Task{{
					Source: "Makefile",
					Name:   "test",
					Desc:   "Run the tests",
					Args:   []string{"go", "test"},
				}})
			},
			width: 40,
			want: rowDerivations{
				badgeW:   8,
				cmdW:     7,
				cmd:      "go test",
				cmdMask:  nil,
				descLine: " Run the tests",
				descMask: nil,
			},
		},
		{
			// The width is part of the key of the era, not a door of the
			// list, so the case changes it through a render at another width.
			name:  "width",
			bump:  func(m *model) {},
			width: 20,
			want: rowDerivations{
				badgeW:   12,
				cmdW:     5,
				cmd:      "npm …",
				cmdMask:  nil,
				descLine: "",
				descMask: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []core.Task{cacheTestTask(), {Source: "Makefile", Name: "test", Desc: "Run the tests", Args: []string{"go", "test"}}}
			m := cacheTestModel(tasks)
			m.listLines(40)
			m.rowCache.fill(era{gen: m.list.generation(), width: 40}, 0, poison)
			tt.bump(&m)
			rows := m.listLines(tt.width)
			if contained(rows[0], "cache-poison") {
				t.Fatalf("the render served the stale derivation after the bump:\n%s", rows[0])
			}
			d, hit := m.rowCache.lookup(era{gen: m.list.generation(), width: tt.width}, 0)
			if !hit {
				t.Fatal("the render of the new era did not fill the row")
			}
			checkDerivations(t, d, tt.want)
		})
	}
}

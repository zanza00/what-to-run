package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// The window tests pin the module seam directly, without the model: every
// operation lands clamped and scrolled, and the read clamps a drifted offset.
// The notifications test pins the model's named calls of the module. The view
// tests in tui_test.go stay the outermost, byte-level check.

// manyTasks returns n tasks named task-00 …, enough to scroll.
func manyTasks(n int) []core.Task {
	tasks := make([]core.Task, n)
	for i := range tasks {
		name := fmt.Sprintf("task-%02d", i)
		tasks[i] = core.Task{Source: "Makefile", Name: name, Args: []string{"make", name}}
	}
	return tasks
}

// TestWindowOperations pins the branch table of the window operations: the
// sticky scroll at the window edges, the page steps at the current height,
// and home and end at every boundary. Every expected pair follows the sticky
// rule by hand: the window moves only when the cursor crosses its edge.
func TestWindowOperations(t *testing.T) {
	tests := []struct {
		name string
		n    int // the list length
		h    int // the rows that the box shows
		cur  int // the cursor before the move
		off  int // the offset before the move
		op   func(*window, int, int)
		want window
	}{
		{name: "down inside the window keeps the offset",
			n: 10, h: 3, cur: 0, off: 0, op: (*window).down, want: window{cursor: 1, offset: 0}},
		{name: "down crossing the window's bottom scrolls one row",
			n: 10, h: 3, cur: 2, off: 0, op: (*window).down, want: window{cursor: 3, offset: 1}},
		{name: "down at the list bottom stays",
			n: 10, h: 3, cur: 9, off: 7, op: (*window).down, want: window{cursor: 9, offset: 7}},
		{name: "up inside the window keeps the offset",
			n: 10, h: 3, cur: 3, off: 1, op: (*window).up, want: window{cursor: 2, offset: 1}},
		{name: "up crossing the window's top scrolls one row",
			n: 10, h: 3, cur: 1, off: 1, op: (*window).up, want: window{cursor: 0, offset: 0}},
		{name: "up at the list top stays on the first row",
			n: 10, h: 3, cur: 0, off: 0, op: (*window).up, want: window{cursor: 0, offset: 0}},
		{name: "pgdown jumps one page of the current height",
			n: 10, h: 3, cur: 0, off: 0, op: (*window).pageDown, want: window{cursor: 3, offset: 1}},
		{name: "pgdown from the middle follows the cursor",
			n: 10, h: 3, cur: 6, off: 4, op: (*window).pageDown, want: window{cursor: 9, offset: 7}},
		{name: "pgdown past the bottom clamps to the last row",
			n: 10, h: 3, cur: 9, off: 7, op: (*window).pageDown, want: window{cursor: 9, offset: 7}},
		{name: "pgup from the bottom returns one page",
			n: 10, h: 3, cur: 9, off: 7, op: (*window).pageUp, want: window{cursor: 6, offset: 6}},
		{name: "pgup past the top clamps to the first row",
			n: 10, h: 3, cur: 1, off: 0, op: (*window).pageUp, want: window{cursor: 0, offset: 0}},
		{name: "home from a scrolled window shows the first row",
			n: 10, h: 3, cur: 9, off: 7, op: (*window).home, want: window{cursor: 0, offset: 0}},
		{name: "home from a drifted offset still shows the first row",
			n: 10, h: 3, cur: 5, off: 9, op: (*window).home, want: window{cursor: 0, offset: 0}},
		{name: "end from the top scrolls the last page in",
			n: 10, h: 3, cur: 0, off: 0, op: (*window).end, want: window{cursor: 9, offset: 7}},
		{name: "end from the bottom stays",
			n: 10, h: 3, cur: 9, off: 7, op: (*window).end, want: window{cursor: 9, offset: 7}},
		{name: "end with the list shorter than the window keeps the whole list",
			n: 3, h: 5, cur: 0, off: 0, op: (*window).end, want: window{cursor: 2, offset: 0}},
		{name: "end on the empty list lands on the first row",
			n: 0, h: 3, cur: 0, off: 0, op: (*window).end, want: window{}},
		{name: "down on the empty list lands at the top",
			n: 0, h: 3, cur: 4, off: 2, op: (*window).down, want: window{}},
		{name: "one row list ignores the moves",
			n: 1, h: 1, cur: 0, off: 0, op: (*window).down, want: window{}},
		{name: "one row window follows the cursor",
			n: 10, h: 1, cur: 0, off: 0, op: (*window).down, want: window{cursor: 1, offset: 1}},
		{name: "one row window keeps the last row on end",
			n: 10, h: 1, cur: 0, off: 0, op: (*window).end, want: window{cursor: 9, offset: 9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := window{cursor: tt.cur, offset: tt.off}
			tt.op(&w, tt.n, tt.h)
			if w != tt.want {
				t.Errorf("after the move the window holds (%d, %d), want (%d, %d)",
					w.cursor, w.offset, tt.want.cursor, tt.want.offset)
			}
		})
	}
}

// TestWindowVisibleRange pins the read of the two list renders and its
// defensive clamp: a drifted offset, one that a missed geometry or list
// change can leave behind, reads clamped to the list, so the frame stays
// sane.
func TestWindowVisibleRange(t *testing.T) {
	tests := []struct {
		name string
		n, h int
		w    window
		want [2]int // the offset and the end row
	}{
		{name: "an offset inside the list reads unchanged",
			n: 20, h: 5, w: window{cursor: 3, offset: 3}, want: [2]int{3, 8}},
		{name: "an offset past the last full window clamps to it",
			n: 20, h: 5, w: window{cursor: 8, offset: 40}, want: [2]int{15, 20}},
		{name: "a box taller than the list shows the whole list",
			n: 4, h: 9, w: window{cursor: 2, offset: 3}, want: [2]int{0, 4}},
		{name: "an offset one row past the window clamps",
			n: 10, h: 3, w: window{cursor: 9, offset: 8}, want: [2]int{7, 10}},
		{name: "an empty list reads no rows",
			n: 0, h: 5, w: window{cursor: 2, offset: 3}, want: [2]int{0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			off, end := tt.w.visibleRange(tt.n, tt.h)
			if off != tt.want[0] || end != tt.want[1] {
				t.Errorf("visibleRange(%d, %d) of window %+v = (%d, %d), want (%d, %d)",
					tt.n, tt.h, tt.w, off, end, tt.want[0], tt.want[1])
			}
		})
	}
}

// TestWindowRow pins the row read: the clamped cursor. The warning box shows
// one warning at a time, and it picks that row by the cursor, so neither a
// drifted nor a negative cursor reaches the list.
func TestWindowRow(t *testing.T) {
	tests := []struct {
		name string
		n    int
		w    window
		want int
	}{
		{name: "a cursor inside the list reads unchanged",
			n: 3, w: window{cursor: 1, offset: 1}, want: 1},
		{name: "a negative cursor clamps to the first row",
			n: 3, w: window{cursor: -2, offset: -2}, want: 0},
		{name: "a cursor past the list clamps to the last row",
			n: 3, w: window{cursor: 9, offset: 9}, want: 2},
		{name: "the height one window of the warning box reads its row",
			n: 5, w: window{cursor: 2, offset: 2}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.row(tt.n); got != tt.want {
				t.Errorf("row(%d) of window %+v = %d, want %d", tt.n, tt.w, got, tt.want)
			}
		})
	}
}

// TestWindowNotifications pins the model side: the events that change the
// facts of a list go through the window module, by name, instead of relying
// on a remembered call at the end of the key handler. A filter edit resets
// the list to the top, a theme save settles both lists against the relisted
// tasks and warnings, and a resize settles at the new height. The
// notifications are observed on the model state, and the view tests in
// tui_test.go pin the frames that they produce.
func TestWindowNotifications(t *testing.T) {
	list := manyTasks(30)
	warnings := []core.Warning{
		core.NewSourceWarning("make", "Makefile", "first warning"),
		core.NewSourceWarning("make", "GNUmakefile", "second warning"),
		core.NewSourceWarning("make", "makefile.mk", "third warning"),
	}
	tests := []struct {
		name string
		warn []core.Warning
		// scrollDowns are the down keys that move the list into a scrolled
		// window before the case plays its own messages.
		scrollDowns int
		msgs        []tea.Msg
		wantList    window
		wantWarn    window
	}{
		{
			name:        "a filter edit resets the list to the top",
			scrollDowns: 8,
			msgs: []tea.Msg{
				tea.KeyPressMsg{Code: '0', Text: "0"},
				tea.KeyPressMsg{Code: '9', Text: "9"},
			},
			wantList: window{},
			wantWarn: window{},
		},
		{
			name: "a theme save settles both windows against the new lists",
			warn: warnings,
			msgs: func() []tea.Msg {
				msgs := []tea.Msg{escKey} // the start opened the box, close it again
				for range 8 {
					// Move the list into a scrolled window: cursor 8, offset 3
					// at six rows per frame.
					msgs = append(msgs, downKey)
				}
				msgs = append(msgs, ctrlWKey, downKey, downKey)
				msgs = append(msgs, themeSavedMsg{
					name: "nord", path: "/rc/config.toml",
					tasks: manyTasks(3), warnings: warnings[:2],
				})
				return msgs
			}(),
			wantList: window{cursor: 2, offset: 0},
			wantWarn: window{cursor: 1, offset: 1},
		},
		{
			name:        "a resize settles at the new height",
			scrollDowns: 8,
			msgs:        []tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 8}},
			wantList:    window{cursor: 8, offset: 5},
			wantWarn:    window{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := update(newModel(Options{}, list, tt.warn), tea.WindowSizeMsg{Width: 80, Height: 10})
			for range tt.scrollDowns {
				m = update(m, downKey)
			}
			for _, msg := range tt.msgs {
				m = update(m, msg)
			}
			if m.listWin != tt.wantList {
				t.Errorf("task window holds %+v, want %+v", m.listWin, tt.wantList)
			}
			if m.warnWin != tt.wantWarn {
				t.Errorf("warning window holds %+v, want %+v", m.warnWin, tt.wantWarn)
			}
		})
	}
}

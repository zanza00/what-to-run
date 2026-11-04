package tui

import (
	"slices"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// The task list tests pin the module seam directly, without the model: the
// list derives its facts at construction and at every mutation, and its
// reads return those facts. The model tests pin the delegation of the
// model's setFilter and setTasks doors. The view tests in tui_test.go stay
// the outermost, byte-level check.

// TestTaskListMutations pins that the list derives its facts at construction
// and derives them again at every mutation. The oracle recomputes the ranked
// tasks and the command width through the free ranking functions, which carry
// their own tests.
func TestTaskListMutations(t *testing.T) {
	tasks := sampleTasks()
	shell := core.DefaultShellDialect()

	tests := []struct {
		name   string
		mutate func(*taskList) // the mutation that the case applies after construction
		tasks  []core.Task     // the tasks that the reads must rank
		filter string          // the filter text that the reads must return
	}{
		{
			name:   "construction derives the facts of the empty filter",
			mutate: func(*taskList) {},
			tasks:  tasks,
			filter: "",
		},
		{
			name:   "setFilter rebuilds the facts from the new text",
			mutate: func(l *taskList) { l.setFilter("make build") },
			tasks:  tasks,
			filter: "make build",
		},
		{
			name: "setTasks rebuilds the facts from the new slice with the filter held",
			mutate: func(l *taskList) {
				l.setFilter("make")
				l.setTasks(tasks[:2])
			},
			tasks:  tasks[:2],
			filter: "make",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newTaskList(tasks, shell)
			tt.mutate(&l)

			tokens := filterTokensOf(tt.filter)
			if got, want := l.filterText(), tt.filter; got != want {
				t.Errorf("filterText() = %q, want %q", got, want)
			}
			if got, want := l.filterTokens(), tokens; !slices.Equal(got, want) {
				t.Errorf("filterTokens() = %v, want %v", got, want)
			}
			if got, want := l.taskCount(), len(tt.tasks); got != want {
				t.Errorf("taskCount() = %d, want %d", got, want)
			}
			wantRanked := rankTasks(tt.tasks, tokens, shell)
			if got := taskNames(l.visible()); !slices.Equal(got, taskNames(wantRanked)) {
				t.Errorf("visible() = %v, want %v", got, taskNames(wantRanked))
			}
			if got, want := l.commandWidth(), commandWidthFor(wantRanked, shell); got != want {
				t.Errorf("commandWidth() = %d, want %d", got, want)
			}
		})
	}
}

// TestTaskListReadsShareTheRankedSlice pins that the reads are stable calls: a
// repeated read returns the same slice, and the list never takes a defensive
// copy. The callers treat the ranked slice as read-only, so sharing it keeps
// the cost of a frame free of a new ranked list per read.
func TestTaskListReadsShareTheRankedSlice(t *testing.T) {
	tasks := sampleTasks()
	l := newTaskList(tasks, core.DefaultShellDialect())

	// An empty filter ranks nothing, so the ranked slice is the tasks that the
	// constructor took.
	if got := l.visible(); &got[0] != &tasks[0] {
		t.Error("visible() of the empty filter returned a copy of the tasks, want the tasks themselves")
	}
	if a, b := l.visible(), l.visible(); &a[0] != &b[0] {
		t.Error("visible() returned a new slice per read, want the same shared slice")
	}

	l.setFilter("make")
	if a, b := l.visible(), l.visible(); &a[0] != &b[0] {
		t.Error("visible() of a filter returned a new slice per read, want the same shared slice")
	}
}

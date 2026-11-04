package tui

import (
	"strings"

	"github.com/zanza00/what-to-run/internal/core"
)

// The task list owns the tasks, the filter text and the facts that they
// derive: the filter tokens, the ranked tasks and the command width. The list
// derives the facts once, at construction, and it derives them again at every
// mutation. No mutation can skip that derivation, so the facts are never
// stale, and every read below is a plain return of a fact: the list carries no
// validity flag and no fallback path.
//
// The model holds one list and mutates it only through setFilter and setTasks
// (see tui.go). The reads are the model's only way to the tasks, the filter
// text and their facts. The free functions rankTasks and commandWidthFor
// carry the derivations themselves, with their own tests in tui_test.go.

// taskList holds the tasks, the filter text and the facts that they derive.
type taskList struct {
	tasks  []core.Task       // the tasks of the project, in discovery order
	filter string            // the filter text; empty means no filtering
	shell  core.ShellDialect // quotes the command lines that the derivations read

	// The facts below come from the filter text and the tasks. The ranked
	// slice stays shared with the callers of visible: they treat it as
	// read-only, and the list never copies it.
	tokens []string    // the filter text as lowercase whitespace tokens
	ranked []core.Task // the tasks that match the filter, from the best match to the worst one
	cmdW   int         // the widest command line among the ranked tasks
}

// newTaskList builds the list over tasks and derives the facts of the empty
// filter. The model builds its list once, at construction.
func newTaskList(tasks []core.Task, shell core.ShellDialect) taskList {
	l := taskList{tasks: tasks, shell: shell}
	l.rebuild()
	return l
}

// setFilter writes the filter text and derives the facts again from it.
func (l *taskList) setFilter(text string) {
	l.filter = text
	l.rebuild()
}

// setTasks replaces the tasks and derives the facts again from them. The
// filter text stays, so the new ranking filters the new tasks.
func (l *taskList) setTasks(tasks []core.Task) {
	l.tasks = tasks
	l.rebuild()
}

// rebuild derives the facts from the filter text and the tasks, each fact from
// the one before it: the tokens, then the ranking over the full task list,
// then the command width over the ranked tasks.
func (l *taskList) rebuild() {
	l.tokens = strings.Fields(strings.ToLower(l.filter))
	l.ranked = rankTasks(l.tasks, l.tokens, l.shell)
	l.cmdW = commandWidthFor(l.ranked, l.shell)
}

// visible returns the tasks that match the current filter, ranked from the
// best match to the worst one. The read shares the ranked slice with its
// caller, and it never copies the slice.
func (l taskList) visible() []core.Task {
	return l.ranked
}

// filterText returns the current filter text; empty means no filtering.
func (l taskList) filterText() string {
	return l.filter
}

// filterTokens returns the lowercase whitespace-separated tokens of the
// current filter.
func (l taskList) filterTokens() []string {
	return l.tokens
}

// commandWidth returns the width of the command column: the widest visible
// command line, so that the descriptions of all rows start together. taskRow
// caps it to the room that the row has, and only a command longer than that
// is truncated.
func (l taskList) commandWidth() int {
	return l.cmdW
}

// taskCount returns the number of tasks that the list holds.
func (l taskList) taskCount() int {
	return len(l.tasks)
}

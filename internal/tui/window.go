package tui

// The task list, the theme picker and the warning list each show a window
// over their rows (the term lives in GLOSSARY.md): the cursor row that the
// keys move, and the rows that the box shows of the list. The window owns the
// arithmetic that keeps both facts inside the list and inside the box, so no
// key handler clamps a cursor and no render rechecks an offset. The type
// reads nothing: every method takes the facts of the moment, the list length
// and the box height, as its arguments. The model passes its own facts, so
// one type serves all three lists and the tests never need the model.

// window is the cursor and the offset over one list: the selected row and
// the first row that the box shows. The task list, the theme picker and the
// warning list hold one window each; the warning list holds its instance at
// height one, where the window shows exactly the row that the cursor holds.
type window struct {
	cursor int // the selected row of the list
	offset int // the first row that the box shows
}

// up moves the cursor one row up and settles the window (see settle).
func (w *window) up(n, h int) {
	w.cursor--
	w.settle(n, h)
}

// down moves the cursor one row down and settles the window.
func (w *window) down(n, h int) {
	w.cursor++
	w.settle(n, h)
}

// pageUp moves the cursor by the height of the box and settles the window.
func (w *window) pageUp(n, h int) {
	w.cursor -= h
	w.settle(n, h)
}

// pageDown moves the cursor by the height of the box and settles the window.
func (w *window) pageDown(n, h int) {
	w.cursor += h
	w.settle(n, h)
}

// home moves the cursor to the first row and settles the window.
func (w *window) home(n, h int) {
	w.cursor = 0
	w.settle(n, h)
}

// end moves the cursor to the last row and settles the window. An empty list
// holds row zero.
func (w *window) end(n, h int) {
	w.cursor = n - 1
	w.settle(n, h)
}

// top resets the window to the first row. A filter edit starts from the top
// whatever the list and the box hold, so this operation takes no facts.
func (w *window) top() {
	w.cursor, w.offset = 0, 0
}

// settle settles the window after a key, a geometry change or a list change.
// An empty list holds row zero. The cursor lands inside the list, and the
// offset lands inside the list too. Then the window moves only when the
// cursor crosses its edge: the scroll is sticky, so the list does not
// recentre under the cursor at every keystroke.
func (w *window) settle(n, h int) {
	if n == 0 {
		w.cursor, w.offset = 0, 0
		return
	}
	w.cursor = min(max(w.cursor, 0), n-1)
	w.offset = min(max(w.offset, 0), max(n-h, 0))
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+h {
		w.offset = w.cursor - h + 1
	}
}

// visibleRange returns the rows that one box shows of a list n rows long and
// h rows tall: the first row and the end row, so the visible rows are
// list[off:end]. The read clamps the offset itself, because a geometry or
// list change can miss its settle on a path that no test covers: the frame
// stays sane, exactly as the inline clamps kept it before the module.
func (w window) visibleRange(n, h int) (off, end int) {
	off = min(w.offset, max(n-h, 0))
	return off, min(off+h, n)
}

// row returns the cursor clamped to a list n rows long. The warning box
// shows one warning at a time, and it picks that row by the cursor, not by
// the offset: the read keeps the two facts of the window apart.
func (w window) row(n int) int {
	return min(max(w.cursor, 0), n-1)
}

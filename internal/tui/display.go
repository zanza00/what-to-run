package tui

// display owns where the picker draws: the screen that the frame draws on,
// and the geometry of the inline area. The model holds one display and never
// recombines raw booleans: the five named operations are the only writes, and
// the named reads report the facts.
//
// Four laws hold:
//   - The alternate screen restores itself. Only a frame that draws in the
//     area leaves a block on the main screen, so a quit clears only that
//     block.
//   - The width of the main screen is remembered while the frame draws in the
//     area. The alternate screen can report one column more, and one column
//     too many wraps every row.
//   - The layout uses the minimum inline height when fewer rows are free.
//   - A terminal that never answers the cursor query falls back once, and the
//     picker never returns to the area.
type display struct {
	// inline holds how the picker was started. An inline start draws in the
	// area below the prompt and quits with the task for the shell. A
	// fullscreen start takes the whole terminal and runs the task in place.
	inline bool

	// full holds the move to the alternate screen, made by expand or by the
	// fallback. It applies only to an inline start: a fullscreen start takes
	// the alternate screen from the start.
	full bool

	// noPosition holds the fallback: the terminal never answered the cursor
	// query, so the area is gone and the picker stays on the alternate
	// screen.
	noPosition bool

	// width and height hold the size that the terminal last reported.
	width  int
	height int

	// ready reports that the terminal reported its size.
	ready bool

	// startRow holds the anchor row: the 0-based row below the shell prompt
	// that the area starts on, as the terminal reports it.
	startRow int

	// startRowKnown reports that the terminal answered the cursor query.
	startRowKnown bool

	// mainWidth holds the width of the main screen, remembered while the
	// frame draws in the area.
	mainWidth int
}

// newDisplay builds the display of a picker that the edge starts inline or
// fullscreen. Nothing is known yet: the terminal reports its size and its
// cursor row afterwards.
func newDisplay(inline bool) display {
	return display{inline: inline}
}

// startedInline reports how the picker was started. An inline start puts the
// task into the shell after the picker quits, and a fullscreen start runs the
// task in place.
func (d display) startedInline() bool {
	return d.inline
}

// anchored reports that the frame draws in the area: the space between the
// line below the prompt and the bottom of the screen, and no alternate screen
// is active.
func (d display) anchored() bool {
	return d.inline && !d.full
}

// expanded reports the alternate screen: the frame draws on a screen that the
// terminal restores by itself.
func (d display) expanded() bool {
	return !d.anchored()
}

// resize writes the size that the terminal reported and marks the size known.
// While the frame draws in the area, it also remembers the width of the main
// screen (see mainWidth).
func (d *display) resize(w, h int) {
	d.width, d.height = w, h
	d.ready = true
	if d.anchored() {
		d.mainWidth = w
	}
}

// learnPosition writes the anchor row that the terminal reported and marks the
// row known. The answer also clears the fallback, because the terminal proved
// that it answers the cursor query, so the picker can collapse back into the
// area.
func (d *display) learnPosition(y int) {
	d.startRow = y
	d.startRowKnown = true
	d.noPosition = false
}

// fallBack expands when the terminal never answered the cursor query, and
// reports whether the frame moved. It happens once, and it keeps the width
// of the main screen exactly as expand does.
func (d *display) fallBack() bool {
	if !d.inline || d.startRowKnown || d.noPosition {
		return false
	}
	d.noPosition = true
	if !d.full {
		d.mainWidth = d.width
		d.full = true
	}
	return true
}

// expand goes to the alternate screen and remembers the width of the main
// screen first. It reports whether the frame moved. It refuses when there is
// no area to return to: the picker was started fullscreen, or the terminal
// never answered the cursor query. It also refuses when the frame already
// draws on the alternate screen, so that the alternate screen does not
// overwrite the remembered width.
func (d *display) expand() bool {
	if !d.inline || d.noPosition || d.full {
		return false
	}
	d.mainWidth = d.width
	d.full = true
	return true
}

// collapse returns to the area, and reports whether the frame moved. It
// refuses when the frame was not moved to the alternate screen, because then
// there is nothing to return from. It also refuses when there is no area to
// return to: the picker was started fullscreen, or the terminal never
// answered the cursor query.
func (d *display) collapse() bool {
	if !d.inline || d.noPosition || !d.full {
		return false
	}
	d.full = false
	return true
}

// areaHeight returns the number of rows that the anchored frame occupies: the
// space between the anchor row and the bottom of the screen. It returns 0
// while the size or the anchor row is still unknown.
func (d display) areaHeight() int {
	if !d.ready || !d.startRowKnown {
		return 0
	}
	if h := d.height - d.startRow; h > minInlineHeight {
		return h
	}
	return minInlineHeight
}

// inlineWidth returns the width that the anchored frame is laid out in: the
// remembered width of the main screen, or the reported width when nothing is
// remembered.
func (d display) inlineWidth() int {
	if !d.anchored() || d.mainWidth <= 0 {
		return d.width
	}
	return min(d.width, d.mainWidth)
}

// viewHeight returns the height of the frame on the alternate screen: the
// terminal height, or the natural size while the terminal has not reported
// its size. naturalRows is the natural height of the list, which the model
// derives from its tasks.
func (d display) viewHeight(naturalRows int) int {
	if !d.ready {
		return chromeRows + naturalRows
	}
	return d.height
}

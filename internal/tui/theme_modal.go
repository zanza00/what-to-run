package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The theme picker (ctrl+t) is a modal box drawn over the picker. It lists the
// themes and windows the list when the space is short. Each row wears its own
// palette, and the previewed row is highlighted. The box is composited cell by
// cell (see overlay), so the styled rows around it stay intact.

// themeModalListRows is how many theme rows the box shows at a time. The list
// scrolls under the cursor when it holds more themes.
const themeModalListRows = 5

// themeModalHeight returns the number of rows that the picker box has
// available: the whole terminal in fullscreen, or the inline area otherwise.
func (m model) themeModalHeight() int {
	if m.opts.Inline {
		return m.areaHeight()
	}
	return m.height
}

// themeModalHints is the key hint line. It stays short, so the box stays
// narrow.
const themeModalHints = "↑/↓ · enter · esc"

// themeModalTitle is the widest top border title, and it sizes the box: the
// "themes 14/14" counter with both scroll arrows.
const themeModalTitle = " ↑ themes 14/14 ↓ "

// themeModalRows returns how many theme rows the box shows at height h: five,
// or fewer when the space is too short for them.
func themeModalRows(h int) int {
	return min(len(themes), themeModalListRows, max(h-boxChrome, 1))
}

// themeModalWindow returns how many theme rows the box shows right now.
func (m model) themeModalWindow() int {
	return themeModalRows(m.themeModalHeight())
}

// keepThemeVisible scrolls the window so that the previewed theme stays inside
// it.
func (m *model) keepThemeVisible() {
	h := m.themeModalWindow()
	m.themeOffset = min(max(m.themeOffset, 0), max(len(themes)-h, 0))
	if m.themeIdx < m.themeOffset {
		m.themeOffset = m.themeIdx
	}
	if m.themeIdx >= m.themeOffset+h {
		m.themeOffset = m.themeIdx - h + 1
	}
}

// themeModalLines renders the theme picker box: the title and the position on
// the top border, one row per visible theme, and the key hints in the footer.
// The box module draws the borders. It returns nil when the space is too
// small for a usable box.
func (m model) themeModalLines(h, w int) []string {
	nameCol, innerW := themeModalLayout(w)
	n := themeModalRows(h)
	off := min(m.themeOffset, max(len(themes)-n, 0))

	up, down := "", ""
	if off > 0 {
		up = "↑ "
	}
	if off+n < len(themes) {
		down = " ↓"
	}
	title := fmt.Sprintf(" %sthemes %d/%d%s ", up, m.themeIdx+1, len(themes), down)

	// One row holds one visible theme. The themeModalRow call already pads
	// each row to innerW, so the box adds no padding of its own.
	rows := make([]string, 0, n)
	for i := off; i < off+n; i++ {
		rows = append(rows, themeModalRow(m.themeAt(i), i == m.themeIdx, nameCol, innerW))
	}

	// The hints are content, so the caller drops them when they do not fit.
	// The dropped hints still go through the dim style, because lipgloss
	// renders the empty string as a zero-width escape pair, and the box must
	// not replace that pair with bare text.
	hints := themeModalHints
	if lipgloss.Width(hints) > innerW {
		hints = ""
	}

	b := box{
		title:  title,
		body:   rows,
		footer: m.styles.dim.Render(hints),
		border: m.themeAt(m.themeIdx).accent,
		innerW: innerW,
	}
	return b.render(h, w)
}

// themeModalLayout returns the name column and the content width of the box
// for a terminal w columns wide. It accounts for the names, the room that the
// title needs on the top border, and the hints line.
func themeModalLayout(w int) (nameCol, innerW int) {
	nameW := 0
	for _, th := range themes {
		nameW = max(nameW, lipgloss.Width(th.name))
	}
	nameCol = nameW
	innerW = max(2+nameCol, lipgloss.Width(themeModalTitle)-1, lipgloss.Width(themeModalHints))
	if maxW := w - 4; innerW > maxW {
		innerW = max(maxW, 1)
	}
	nameCol = min(nameCol, max(innerW-2, 1))
	return nameCol, innerW
}

// themeModalRow renders one theme row: the cursor marker and the name in the
// accent of the theme, over the tint of the theme when the row is being
// previewed.
func themeModalRow(th theme, selected bool, nameCol, innerW int) string {
	st := newStyles(th)

	name := ansi.Truncate(th.name, nameCol, "…")
	name += strings.Repeat(" ", max(nameCol-lipgloss.Width(name), 0))

	if selected {
		left := st.marker.Render("▌ ") + st.selectedName.Render(name)
		pad := strings.Repeat(" ", max(innerW-lipgloss.Width(left), 0))
		if pad != "" {
			pad = st.selectedPad.Render(pad)
		}
		return left + pad
	}

	left := "  " + st.title.Render(name)
	return left + strings.Repeat(" ", max(innerW-lipgloss.Width(left), 0))
}

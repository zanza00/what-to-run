package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The theme picker (ctrl+t) is a modal box drawn over the picker: the list of
// themes, windowed when the space is short, each row in its own palette and
// the previewed one highlighted. The box is composited cell by cell (see
// overlayThemeModal), so the styled rows around it stay intact.

// themeModalChrome is the number of rows the box adds around the theme list:
// two borders and the hints line.
const themeModalChrome = 3

// themeModalListRows is how many theme rows the box shows at a time: the list
// scrolls under the cursor when it has more themes.
const themeModalListRows = 5

// themeModalHeight returns the rows the picker box has available: the whole
// terminal in fullscreen, the inline area otherwise.
func (m model) themeModalHeight() int {
	if m.opts.Inline {
		return m.areaHeight()
	}
	return m.height
}

// themeModalHints is the key hint line, kept short so the box stays narrow.
const themeModalHints = "↑/↓ · enter · esc"

// themeModalTitle is the widest top border title, used to size the box: the
// "themes 13/13" counter with both scroll arrows.
const themeModalTitle = " ↑ themes 13/13 ↓ "

// themeModalRows returns how many theme rows the box shows at height h: five,
// or fewer when the space is too short for them.
func themeModalRows(h int) int {
	return min(len(themes), themeModalListRows, max(h-themeModalChrome, 1))
}

// themeModalWindow returns how many theme rows the box shows right now.
func (m model) themeModalWindow() int {
	return themeModalRows(m.themeModalHeight())
}

// keepThemeVisible scrolls the window so the previewed theme is inside it.
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

// themeModalLines renders the theme picker box: the title and position on the
// top border, one row per visible theme, the key hints, the bottom border.
// It returns nil when the space is too small for a usable box.
func (m model) themeModalLines(h, w int) []string {
	if h < themeModalChrome+1 || w < 10 {
		return nil
	}

	nameCol, innerW := themeModalLayout(w)
	bw := innerW + 4

	border := lipgloss.NewStyle()
	if accent := themes[m.themeIdx].accent; accent != nil {
		border = border.Foreground(accent)
	}

	n := themeModalRows(h)
	off := min(m.themeOffset, max(len(themes)-n, 0))

	up, down := "", ""
	if off > 0 {
		up = "↑ "
	}
	if off+n < len(themes) {
		down = " ↓"
	}
	title := border.Bold(true).Render(fmt.Sprintf(" %sthemes %d/%d%s ", up, m.themeIdx+1, len(themes), down))
	title = ansi.Truncate(title, max(bw-3, 0), "")
	lines := make([]string, 0, n+themeModalChrome)
	lines = append(lines,
		border.Render("╭─")+title+
			border.Render(strings.Repeat("─", max(bw-3-lipgloss.Width(title), 0)))+
			border.Render("╮"))
	for i := off; i < off+n; i++ {
		lines = append(lines,
			border.Render("│")+" "+themeModalRow(themes[i], i == m.themeIdx, nameCol, innerW)+" "+border.Render("│"))
	}
	lines = append(lines,
		border.Render("│")+" "+m.themeModalFooter(innerW)+" "+border.Render("│"),
		border.Render("╰")+border.Render(strings.Repeat("─", bw-2))+border.Render("╯"))
	return lines
}

// themeModalLayout returns the name column and the content width of the box
// for a terminal w columns wide: the names, the room the title needs on the
// top border and the hints line.
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
// theme's own accent, over the theme's tint when the row is being previewed.
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

// themeModalFooter renders the hints line, padded to the content width. The
// hints are dropped when the box is too narrow.
func (m model) themeModalFooter(innerW int) string {
	hints := themeModalHints
	if lipgloss.Width(hints) > innerW {
		hints = ""
	}
	return m.styles.dim.Render(hints) + strings.Repeat(" ", max(innerW-lipgloss.Width(hints), 0))
}

// overlayThemeModal draws the theme picker over the rendered lines with the
// canvas compositor: it replaces cell by cell, so a background that crosses
// the box edges is not broken (splicing styled lines manually would).
func (m model) overlayThemeModal(lines []string, h, w int) string {
	box := m.themeModalLines(h, w)
	if box == nil {
		return strings.Join(lines, "\n")
	}
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(strings.Join(lines, "\n")),
		lipgloss.NewLayer(strings.Join(box, "\n")).
			X(max((w-lipgloss.Width(box[0]))/2, 0)).
			Y(max((h-len(box))/2, 0)).
			Z(1),
	)
	return comp.Render()
}

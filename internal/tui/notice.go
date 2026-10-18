package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The transient notice (toast): a one-line chip in the top-right corner of the
// picker, for ephemeral confirmations like saving a theme. It never touches
// the chrome (the help line), it expires on its own, and a new notice replaces
// the previous one together with its timer. Problems are not notices: they
// stay in the status line until the next event.

// noticeTTL is how long a notice stays on screen.
const noticeTTL = 2 * time.Second

// noticeExpiredMsg expires the notice with the given id: ids keep a replaced
// notice's timer from clearing its successor.
type noticeExpiredMsg struct{ id int }

// showNotice shows text as a toast for noticeTTL and returns its timer.
func (m *model) showNotice(text string) tea.Cmd {
	m.notice = text
	m.noticeID++
	id := m.noticeID
	return tea.Tick(noticeTTL, func(time.Time) tea.Msg { return noticeExpiredMsg{id: id} })
}

// noticeBox renders the toast as a one-line chip in the top-right corner, in
// the active theme's tint colors (reverse video in mono). It returns nil when
// there is no notice or the space is too small.
func (m model) noticeBox(w int) *placedBox {
	if m.notice == "" {
		return nil
	}
	text := m.notice
	if max := w - 4; max > 0 && lipgloss.Width(text) > max {
		text = ansi.Truncate(text, max, "…")
	}
	chip := m.styles.selectedText.Render(" " + text + " ")
	if lipgloss.Width(chip) > w {
		return nil
	}
	return &placedBox{lines: []string{chip}, x: max(w-lipgloss.Width(chip)-1, 0), y: 0}
}

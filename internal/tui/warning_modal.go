package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/zanza00/what-to-run/internal/config"
)

// The warning modal (ctrl+w, or opened automatically at startup when an
// undismissed warning is present) is a box drawn over the picker. It wraps the
// warning message across multiple lines, so long file paths and parser
// messages stay readable without replacing the help line at the bottom.
//
// Pressing d marks the warning as dismissed in the cache directory of the
// user, so a later run (such as a repeated inline widget invocation) does not
// pop the modal open again for the same warning. The warning count and ctrl+w
// stay available in the bottom chrome.

// warningModalChrome is the number of rows that the box adds around the
// wrapped warning text: two borders and the key hints line.
const warningModalChrome = 3

// warningModalMaxInnerWidth caps the text column of the warning box on a wide
// terminal, so the wrapped lines stay comfortable to read.
const warningModalMaxInnerWidth = 60

// Key hint lines for the bottom of the warning box, from widest to narrowest.
const (
	warningModalHints       = "enter/esc close · d don't show again"
	warningModalScrollHints = "↑/↓ · enter/esc close · d don't show again"
	warningModalShortHints  = "enter · esc · d"
)

// openWarningsAtStart reports whether the warning modal should open at startup,
// and which warning index to show first. The index is the first warning that
// the user has not dismissed with d. The dismissed set comes from the edge.
func openWarningsAtStart(warnings []error, dismissed map[string]bool) (bool, int) {
	if len(warnings) == 0 {
		return false, 0
	}
	for i, w := range warnings {
		if !dismissed[warningKey(w)] {
			return true, i
		}
	}
	return false, 0
}

// warningCount formats the number of warnings for the bottom chrome.
func warningCount(n int) string {
	if n == 1 {
		return "1 warning"
	}
	return fmt.Sprintf("%d warnings", n)
}

// dismissCurrentWarning returns a command that records the displayed warning
// as dismissed, so a later run does not pop the modal open for it. The result
// message carries the warning index and the reloaded dismissed set.
func (m model) dismissCurrentWarning() tea.Cmd {
	if len(m.warnings) == 0 {
		return nil
	}
	idx := min(max(m.warningIdx, 0), len(m.warnings)-1)
	env := m.env
	warning := m.warnings[idx]
	return func() tea.Msg {
		if err := saveDismissedWarning(env.Store, env.CacheFile, warning); err != nil {
			return warningDismissedMsg{idx: idx, err: err}
		}
		return warningDismissedMsg{idx: idx, dismissed: loadDismissedWarnings(env.Store, env.CacheFile)}
	}
}

// warningDismissedMsg reports the result of saving a dismissed warning.
type warningDismissedMsg struct {
	idx       int
	dismissed map[string]bool
	err       error
}

// applyWarningDismissed advances to the next undismissed warning, and it closes
// the box when every warning is dismissed. A failed save shows the error in the
// status line and closes the box.
func (m *model) applyWarningDismissed(msg warningDismissedMsg) {
	if msg.err != nil {
		m.status = m.styles.err.Render(fmt.Sprintf("cannot save dismissed warning: %v", msg.err))
		m.warningModal = false
		return
	}
	m.env.Dismissed = msg.dismissed
	if len(m.warnings) == 0 {
		m.warningModal = false
		return
	}
	from := min(max(msg.idx, 0), len(m.warnings)-1)
	for i := range len(m.warnings) {
		idx := (from + 1 + i) % len(m.warnings)
		if !m.env.Dismissed[warningKey(m.warnings[idx])] {
			m.warningIdx = idx
			return
		}
	}
	m.warningModal = false
}

// warningModalLines renders the warning box: the title (and the position when
// there are several warnings) on the top border, the wrapped warning text, the
// key hints and the bottom border. It returns nil when there are no warnings or
// when the space is too small for a usable box.
func (m model) warningModalLines(h, w int) []string {
	if len(m.warnings) == 0 || h < warningModalChrome+1 || w < 10 {
		return nil
	}

	idx := min(max(m.warningIdx, 0), len(m.warnings)-1)
	maxInnerW := min(max(w-4, 1), warningModalMaxInnerWidth)
	rawTitle := m.warningModalTitle(idx)
	hints := m.warningModalHintText(maxInnerW)

	body := wrapWarning(warningKey(m.warnings[idx]), maxInnerW, max(h-warningModalChrome, 1))
	innerW := max(lipgloss.Width(rawTitle)-1, lipgloss.Width(hints))
	for _, line := range body {
		innerW = max(innerW, lipgloss.Width(line))
	}
	innerW = min(max(innerW, 1), maxInnerW)
	bw := innerW + 4

	border := lipgloss.NewStyle()
	if errColor := m.themeAt(m.themeIdx).err; errColor != nil {
		border = border.Foreground(errColor)
	}

	title := border.Bold(true).Render(rawTitle)
	title = ansi.Truncate(title, max(bw-3, 0), "")

	lines := make([]string, 0, len(body)+warningModalChrome)
	lines = append(lines,
		border.Render("╭─")+title+
			border.Render(strings.Repeat("─", max(bw-3-lipgloss.Width(title), 0)))+
			border.Render("╮"))
	for _, line := range body {
		line = ansi.Truncate(line, innerW, "…")
		pad := strings.Repeat(" ", max(innerW-lipgloss.Width(line), 0))
		lines = append(lines, border.Render("│")+" "+line+pad+" "+border.Render("│"))
	}
	if len(body)+warningModalChrome < h {
		lines = append(lines, border.Render("│")+strings.Repeat(" ", innerW+2)+border.Render("│"))
	}
	lines = append(lines,
		border.Render("│")+" "+m.warningModalFooter(innerW)+" "+border.Render("│"),
		border.Render("╰")+border.Render(strings.Repeat("─", bw-2))+border.Render("╯"))
	return lines
}

// warningModalTitle returns the plain title that goes on the top border of the
// warning box.
func (m model) warningModalTitle(idx int) string {
	if len(m.warnings) <= 1 {
		return " warning "
	}
	up, down := "", ""
	if idx > 0 {
		up = "↑ "
	}
	if idx+1 < len(m.warnings) {
		down = " ↓"
	}
	return fmt.Sprintf(" %swarnings %d/%d%s ", up, idx+1, len(m.warnings), down)
}

// wrapWarning wraps msg to width columns and at most maxLines rows. It breaks
// on spaces, hyphens and path slashes, so a long file path wraps cleanly.
func wrapWarning(msg string, width, maxLines int) []string {
	wrapped := strings.Split(ansi.Wrap(msg, width, "/"), "\n")
	if len(wrapped) <= maxLines {
		return wrapped
	}
	kept := wrapped[:maxLines]
	last := kept[maxLines-1]
	if lipgloss.Width(last) >= width {
		kept[maxLines-1] = ansi.Truncate(last, width, "…")
	} else {
		kept[maxLines-1] = last + "…"
	}
	return kept
}

// warningModalHintText picks the richest hint string that fits in innerW.
func (m model) warningModalHintText(innerW int) string {
	full := warningModalHints
	if len(m.warnings) > 1 {
		full = warningModalScrollHints
	}
	switch {
	case lipgloss.Width(full) <= innerW:
		return full
	case lipgloss.Width(warningModalShortHints) <= innerW:
		return warningModalShortHints
	default:
		return ""
	}
}

// warningModalFooter renders the hints line, padded to innerW.
func (m model) warningModalFooter(innerW int) string {
	hints := m.warningModalHintText(innerW)
	return m.styles.dim.Render(hints) + strings.Repeat(" ", max(innerW-lipgloss.Width(hints), 0))
}

// warningKey normalizes a warning error into a single line, for display and for
// the dismissed-warnings file.
func warningKey(err error) string {
	if err == nil {
		return ""
	}
	return strings.Join(strings.Fields(err.Error()), " ")
}

// loadDismissedWarnings reads the set of dismissed warning messages from path
// through the store. A missing or unreadable file yields an empty set.
func loadDismissedWarnings(store config.Store, path string) map[string]bool {
	if path == "" {
		return nil
	}
	data, err := store.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	dismissed := make(map[string]bool, len(lines))
	for _, line := range lines {
		if k := strings.TrimSpace(line); k != "" {
			dismissed[k] = true
		}
	}
	return dismissed
}

// saveDismissedWarning adds err to the dismissed-warnings file at path, and it
// creates the file and its directory when they are missing.
func saveDismissedWarning(store config.Store, path string, w error) error {
	key := warningKey(w)
	if key == "" {
		return nil
	}
	if path == "" {
		return errors.New("cannot resolve the user cache directory")
	}
	if loadDismissedWarnings(store, path)[key] {
		return nil
	}
	data, err := store.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := store.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var out string
	if trimmed := strings.TrimRight(string(data), "\n"); trimmed != "" {
		out = trimmed + "\n" + key + "\n"
	} else {
		out = key + "\n"
	}
	return store.WriteFile(path, []byte(out), 0o600)
}

// clearResolvedDismissals removes from the cache file the warnings that were
// present in old and are no longer present in current. It loads the file again,
// so a dismissal written by another process is not missed. It returns the set
// that is now in use.
func clearResolvedDismissals(store config.Store, path string, old, current []error) map[string]bool {
	if path == "" {
		return nil
	}
	dismissed := loadDismissedWarnings(store, path)
	if len(old) == 0 || len(dismissed) == 0 {
		return dismissed
	}
	stillActive := make(map[string]bool, len(current))
	for _, w := range current {
		stillActive[warningKey(w)] = true
	}
	next := make(map[string]bool, len(dismissed))
	changed := false
	for k := range dismissed {
		next[k] = true
	}
	for _, w := range old {
		k := warningKey(w)
		if !stillActive[k] && next[k] {
			delete(next, k)
			changed = true
		}
	}
	if !changed {
		return dismissed
	}
	if len(next) == 0 {
		_ = store.Remove(path)
		return next
	}
	var b strings.Builder
	for k := range next {
		b.WriteString(k)
		b.WriteByte('\n')
	}
	_ = store.WriteFile(path, []byte(b.String()), 0o600)
	return next
}

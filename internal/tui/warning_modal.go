package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The warning modal (ctrl+w, or opened automatically at startup when an
// undismissed warning is present) is a box drawn over the picker: it wraps the
// warning message across multiple lines so long file paths and parser messages
// stay readable without replacing the help line at the bottom.
//
// Pressing d marks the warning as dismissed in the user's cache directory so
// subsequent runs (such as repeated inline widget invocations) do not pop the
// modal open again for the same warning, while keeping the warning count and
// ctrl+w available in the bottom chrome.

// warningModalChrome is the number of rows the box adds around the wrapped
// warning text: two borders and the key hints line.
const warningModalChrome = 3

// warningModalMaxInnerWidth caps the text column of the warning box on wide
// terminals so wrapped lines stay comfortable to read.
const warningModalMaxInnerWidth = 60

// Key hint lines for the bottom of the warning box, from widest to narrowest.
const (
	warningModalHints       = "enter/esc close · d don't show again"
	warningModalScrollHints = "↑/↓ · enter/esc close · d don't show again"
	warningModalShortHints  = "enter · esc · d"
)

// openWarningsAtStart reports whether the warning modal should open on
// startup and which warning index to show first: the first warning that has
// not been dismissed with d.
func openWarningsAtStart(warnings []error) (bool, int) {
	if len(warnings) == 0 {
		return false, 0
	}
	dismissed := loadDismissedWarnings()
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

// dismissCurrentWarning records the displayed warning as dismissed so future
// runs do not pop the modal open for it. If another warning in the session is
// still undismissed, the box advances to it; otherwise the box closes.
func (m *model) dismissCurrentWarning() {
	if len(m.warnings) == 0 {
		m.warningModal = false
		return
	}
	m.warningIdx = min(max(m.warningIdx, 0), len(m.warnings)-1)
	if err := saveDismissedWarning(m.warnings[m.warningIdx]); err != nil {
		m.status = m.styles.err.Render(fmt.Sprintf("cannot save dismissed warning: %v", err))
		m.warningModal = false
		return
	}
	dismissed := loadDismissedWarnings()
	for i := range len(m.warnings) {
		idx := (m.warningIdx + 1 + i) % len(m.warnings)
		if !dismissed[warningKey(m.warnings[idx])] {
			m.warningIdx = idx
			return
		}
	}
	m.warningModal = false
}

// warningModalLines renders the warning box: the title (and position when
// there are multiple warnings) on the top border, the wrapped warning text,
// the key hints, and the bottom border. It returns nil when there are no
// warnings or the space is too small for a usable box.
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

// warningModalTitle returns the plain title placed on the top border of the
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

// wrapWarning wraps msg to width columns and at most maxLines rows, breaking
// on spaces, hyphens and path slashes so long file paths wrap cleanly.
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

// warningKey normalizes a warning error into a single line for display and
// persistence.
func warningKey(err error) string {
	if err == nil {
		return ""
	}
	return strings.Join(strings.Fields(err.Error()), " ")
}

// dismissedWarningsFile returns the path of the file that stores dismissed
// warnings in the user's cache directory.
func dismissedWarningsFile() string {
	cacheDir, err := userCacheDir()
	if err != nil || cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, "what-to-run", "dismissed-warnings")
}

// loadDismissedWarnings reads the set of dismissed warning messages from the
// user's cache directory. A missing or unreadable file yields an empty set.
func loadDismissedWarnings() map[string]bool {
	path := dismissedWarningsFile()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
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

// saveDismissedWarning adds err to the user's dismissed warnings file,
// creating the file and its directory when missing.
func saveDismissedWarning(w error) error {
	key := warningKey(w)
	if key == "" {
		return nil
	}
	path := dismissedWarningsFile()
	if path == "" {
		return errors.New("cannot resolve the user cache directory")
	}
	if loadDismissedWarnings()[key] {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var out string
	if trimmed := strings.TrimRight(string(data), "\n"); trimmed != "" {
		out = trimmed + "\n" + key + "\n"
	} else {
		out = key + "\n"
	}
	return os.WriteFile(path, []byte(out), 0o600)
}

// clearResolvedDismissals removes warnings that were present in old but are no
// longer present in current from the dismissed-warnings file.
func clearResolvedDismissals(old, current []error) {
	if len(old) == 0 {
		return
	}
	path := dismissedWarningsFile()
	if path == "" {
		return
	}
	dismissed := loadDismissedWarnings()
	if len(dismissed) == 0 {
		return
	}
	stillActive := make(map[string]bool, len(current))
	for _, w := range current {
		stillActive[warningKey(w)] = true
	}
	changed := false
	for _, w := range old {
		k := warningKey(w)
		if !stillActive[k] && dismissed[k] {
			delete(dismissed, k)
			changed = true
		}
	}
	if !changed {
		return
	}
	if len(dismissed) == 0 {
		_ = os.Remove(path)
		return
	}
	var b strings.Builder
	for k := range dismissed {
		b.WriteString(k)
		b.WriteByte('\n')
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o600)
}

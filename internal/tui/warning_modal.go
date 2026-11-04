package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/zanza00/what-to-run/internal/config"
	"github.com/zanza00/what-to-run/internal/core"
)

// The warning modal (ctrl+w, or opened automatically at startup when an
// undismissed warning is present) is a box drawn over the picker. It wraps the
// warning message across multiple lines, so long file paths and parser
// messages stay readable without replacing the help line at the bottom.
//
// Pressing d marks the warning as dismissed in the cache directory of the
// user, so a later run (such as a repeated inline widget invocation) does not
// pop the modal open again for the same warning. The dismissal of a warning
// keys on the warning's identity, not on its message, so a reword of the
// message in a release does not bring the warning back. Old entries, which
// hold the message text, keep matching too. The warning count and ctrl+w
// stay available in the bottom chrome.

// warningModalMaxInnerWidth caps the text column of the warning box on a wide
// terminal, so the wrapped lines stay comfortable to read.
const warningModalMaxInnerWidth = 60

// Key hint lines for the bottom of the warning box, from widest to narrowest.
const (
	warningModalHints       = "enter/esc close · d don't show again"
	warningModalScrollHints = "↑/↓ · enter/esc close · d don't show again"
	warningModalShortHints  = "enter · esc · d"
)

// dedupeWarnings keeps the first warning of each key, so the list holds one
// warning per underlying problem. The first occurrence's message wins the
// display, because it names the file that the later one only refers to. Two
// rc files that carry the same wrong disable entry produce equal keys, and so
// do the rc loader's validation and discovery's safety net.
func dedupeWarnings(warnings []core.Warning) []core.Warning {
	seen := make(map[string]bool, len(warnings))
	out := make([]core.Warning, 0, len(warnings))
	for _, w := range warnings {
		key := w.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, w)
	}
	return out
}

// warningText normalizes the warning message into a single line. The modal
// displays this text. It is also the legacy entry of the dismissed-warnings
// file: before the keyed format, the picker stored this form of the message.
func warningText(w core.Warning) string {
	return strings.Join(strings.Fields(w.Error()), " ")
}

// warningForms returns the two entries that can dismiss a warning: the key
// that the picker writes today, and the legacy text that an older version
// wrote.
func warningForms(w core.Warning) [2]string {
	return [2]string{w.Key(), warningText(w)}
}

// warnDismissed reports whether the dismissed set holds the warning, by its
// key or by its legacy text. The lookup takes both forms, so a dismissal made
// before the keyed format still matches.
func warnDismissed(dismissed map[string]bool, w core.Warning) bool {
	for _, form := range warningForms(w) {
		if dismissed[form] {
			return true
		}
	}
	return false
}

// openWarningsAtStart reports whether the warning modal should open at startup,
// and which warning index to show first. The index is the first warning that
// the user has not dismissed with d. The dismissed set comes from the edge.
func openWarningsAtStart(warnings []core.Warning, dismissed map[string]bool) (bool, int) {
	if len(warnings) == 0 {
		return false, 0
	}
	for i, w := range warnings {
		if !warnDismissed(dismissed, w) {
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
// as dismissed, so a later run does not pop the modal open for it. The save
// returns the resulting dismissed set, which the model then uses, so the
// dismiss path does not read the file again.
func (m model) dismissCurrentWarning() tea.Cmd {
	if len(m.warnings) == 0 {
		return nil
	}
	idx := min(max(m.warningIdx, 0), len(m.warnings)-1)
	env := m.env
	warning := m.warnings[idx]
	return func() tea.Msg {
		dismissed, err := saveDismissedWarning(env.Store, env.CacheFile, warning)
		if err != nil {
			return warningDismissedMsg{idx: idx, err: err}
		}
		return warningDismissedMsg{idx: idx, dismissed: dismissed}
	}
}

// warningDismissedMsg reports the result of saving a dismissed warning. The
// dismissed set is the one that the save produced.
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
		m.status = status{kind: statusErr, text: fmt.Sprintf("cannot save dismissed warning: %v", msg.err)}
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
		if !warnDismissed(m.env.Dismissed, m.warnings[idx]) {
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
	if len(m.warnings) == 0 {
		return nil
	}

	idx := min(max(m.warningIdx, 0), len(m.warnings)-1)
	maxInnerW := min(max(w-4, 1), warningModalMaxInnerWidth)
	rawTitle := m.warningModalTitle(idx)
	hints := m.warningModalHintText(maxInnerW)

	body := wrapWarning(warningText(m.warnings[idx]), maxInnerW, max(h-boxChrome, 1))
	innerW := max(lipgloss.Width(rawTitle)-1, lipgloss.Width(hints))
	for _, line := range body {
		innerW = max(innerW, lipgloss.Width(line))
	}
	innerW = min(max(innerW, 1), maxInnerW)
	// The blank row becomes the spacer between the text and the hints when the
	// space has a row to spare.
	if len(body)+boxChrome < h {
		body = append(body, "")
	}

	b := box{
		title:  rawTitle,
		body:   body,
		footer: m.styles.dim.Render(hints),
		border: m.themeAt(m.themeIdx).err,
		innerW: innerW,
	}
	return b.render(h, w)
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

// parseDismissedLines returns the entries of the dismissed-warnings file, one
// per line. A blank line holds nothing.
func parseDismissedLines(data []byte) map[string]bool {
	lines := strings.Split(string(data), "\n")
	dismissed := make(map[string]bool, len(lines))
	for _, line := range lines {
		if k := strings.TrimSpace(line); k != "" {
			dismissed[k] = true
		}
	}
	return dismissed
}

// loadDismissedWarnings reads the set of dismissed warnings from path through
// the store. A missing or unreadable file yields an empty set.
func loadDismissedWarnings(store config.Store, path string) map[string]bool {
	if path == "" {
		return nil
	}
	data, err := store.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseDismissedLines(data)
}

// saveDismissedWarning adds the warning's key to the dismissed-warnings file
// at path, and it creates the file and its directory when they are missing.
// It reads the file once and writes at most once, and it returns the set that
// now holds the warning: the dismiss path uses that set instead of reading
// the file again. A warning that either form already dismisses stays as it
// is, so a legacy line does not grow a twin key line.
func saveDismissedWarning(store config.Store, path string, w core.Warning) (map[string]bool, error) {
	if path == "" {
		return nil, errors.New("cannot resolve the user cache directory")
	}
	data, err := store.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	dismissed := parseDismissedLines(data)
	if warnDismissed(dismissed, w) {
		return dismissed, nil
	}
	key := w.Key()
	if err := store.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	var out string
	if trimmed := strings.TrimRight(string(data), "\n"); trimmed != "" {
		out = trimmed + "\n" + key + "\n"
	} else {
		out = key + "\n"
	}
	if err := store.WriteFile(path, []byte(out), 0o600); err != nil {
		return nil, err
	}
	next := maps.Clone(dismissed)
	next[key] = true
	return next, nil
}

// clearResolvedDismissals removes from the cache file the warnings that were
// present in old and are no longer present in current. It loads the file again,
// so a dismissal written by another process is not missed. Both forms of an
// entry age out, the key and the legacy text, so old-format lines die under
// the same rule as the new keys. It returns the set that is now in use.
func clearResolvedDismissals(store config.Store, path string, old, current []core.Warning) map[string]bool {
	if path == "" {
		return nil
	}
	dismissed := loadDismissedWarnings(store, path)
	if len(old) == 0 || len(dismissed) == 0 {
		return dismissed
	}
	stillActive := make(map[string]bool, len(current))
	for _, w := range current {
		stillActive[w.Key()] = true
	}
	next := maps.Clone(dismissed)
	changed := false
	for _, w := range old {
		if stillActive[w.Key()] {
			continue
		}
		// The warning healed, so both of its entries can go.
		for _, form := range warningForms(w) {
			if next[form] {
				delete(next, form)
				changed = true
			}
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

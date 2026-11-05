package tui

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The painter owns the assembly of the selectable rows: the task rows of the
// list and the theme rows of the theme picker. The callers own the budgets
// of their rows, and the painter owns the three laws of the assembly:
//
//   - A row paints segment by segment. A nested style reset would drop the
//     highlight before the row padding, so every segment paints through the
//     style that it needs.
//   - The selected row never takes the dim color. The dim color can be
//     nearly identical to the tint of the theme, so the selected row paints
//     with the selected styles only.
//   - The pad goes through its style only when it is not empty. lipgloss
//     renders the empty string as a zero-width escape pair, so an empty pad
//     must stay bare.
//
// The styles carry the paints of each branch, and the selected flag names
// the branch whose assembly runs. The three laws are the same in both
// branches.

// rowPiece is one piece of a row: its text, its matched runes and its styles.
type rowPiece struct {
	text      string
	matched   []bool
	base      lipgloss.Style
	highlight lipgloss.Style
}

// rowStyles holds the styles of the parts that a row paints through one
// style each: the cursor marker, the pad and the badge. The pieces carry
// their own styles, so each row keeps the palette choice of every piece.
type rowStyles struct {
	marker lipgloss.Style
	pad    lipgloss.Style
	badge  lipgloss.Style
}

// paintRow paints one row: the marker, the pieces, the pad and the badge.
// The budget is the number of columns that the row may fill: the task row
// passes its room, and the theme row passes its inner width.
//
// The selected branch renders the marker through its style, and it closes the
// row with the badge separator inside the badge style, so the last segment
// carries the highlight to the right edge of the row. The plain branch
// renders a bare marker and a bare pad, and it keeps the badge separator
// outside the badge style. A row without a badge passes the empty badge, so
// the pad fills the whole budget.
func paintRow(selected bool, marker string, pieces []rowPiece, badge string, budget int, st rowStyles) string {
	var left string
	if selected {
		left = st.marker.Render(marker)
	} else {
		left = marker
	}
	for _, p := range pieces {
		left += renderMatched(p.text, p.matched, p.base, p.highlight)
	}

	pad := strings.Repeat(" ", max(budget-lipgloss.Width(left), 0))
	if selected && pad != "" {
		pad = st.pad.Render(pad)
	}

	if badge == "" {
		return left + pad
	}
	if selected {
		return left + pad + st.badge.Render(" "+badge)
	}
	return left + pad + " " + st.badge.Render(badge)
}

// truncateMatched fits s within maxW terminal columns while it keeps the
// matched runes visible. When a right truncation would cut off a match, the
// visible window shifts toward the matched span and gets a leading ellipsis.
func truncateMatched(s string, matched []bool, maxW int) (string, []bool) {
	if maxW <= 0 || s == "" {
		return "", nil
	}
	if lipgloss.Width(s) <= maxW {
		return s, matched
	}
	if maxW == 1 {
		return "…", nil
	}

	firstMatch, lastMatch := -1, -1
	for i, ok := range matched {
		if ok {
			if firstMatch < 0 {
				firstMatch = i
			}
			lastMatch = i
		}
	}

	runes := []rune(s)
	width, end := 0, 0
	for end < len(runes) {
		rw := ansi.StringWidth(string(runes[end]))
		if width+rw > maxW-1 {
			break
		}
		width += rw
		end++
	}
	if firstMatch < 0 || lastMatch < end {
		var m []bool
		if len(matched) > 0 {
			m = append(slices.Clone(matched[:end]), false)
		}
		return string(runes[:end]) + "…", m
	}

	// A match lies past the right truncation point. Try a suffix that is
	// anchored to the end of s first, then a middle window framed by two
	// ellipses.
	suffixW := 0
	start := len(runes)
	for start > 0 {
		rw := ansi.StringWidth(string(runes[start-1]))
		if 1+suffixW+rw > maxW {
			break
		}
		suffixW += rw
		start--
	}
	if start <= firstMatch {
		m := make([]bool, 1+len(runes)-start)
		copy(m[1:], matched[start:])
		return "…" + string(runes[start:]), m
	}

	if maxW < 3 {
		return ansi.Truncate(s, maxW, "…"), nil
	}
	start, end = firstMatch, firstMatch
	midW := 0
	for end < len(runes) {
		rw := ansi.StringWidth(string(runes[end]))
		if midW+rw > maxW-2 {
			break
		}
		midW += rw
		end++
	}
	for start > 0 {
		rw := ansi.StringWidth(string(runes[start-1]))
		if midW+rw > maxW-2 {
			break
		}
		midW += rw
		start--
	}
	m := make([]bool, 0, (end-start)+2)
	m = append(m, false)
	m = append(m, matched[start:end]...)
	m = append(m, false)
	return "…" + string(runes[start:end]) + "…", m
}

// renderMatched renders s and applies highlight to the runes marked in matched
// and base to the other runes. Adjacent runes with the same state are grouped
// into a single segment, so the renderer emits no unnecessary ANSI resets.
func renderMatched(s string, matched []bool, base, highlight lipgloss.Style) string {
	if s == "" {
		return ""
	}
	if !slices.Contains(matched, true) {
		return base.Render(s)
	}

	runes := []rune(s)
	var b strings.Builder
	start := 0
	for start < len(runes) {
		isMatch := start < len(matched) && matched[start]
		end := start + 1
		for end < len(runes) && (end < len(matched) && matched[end]) == isMatch {
			end++
		}
		seg := string(runes[start:end])
		if isMatch {
			b.WriteString(highlight.Render(seg))
		} else {
			b.WriteString(base.Render(seg))
		}
		start = end
	}
	return b.String()
}

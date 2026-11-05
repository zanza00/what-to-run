package tui

import (
	"slices"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestPaintRow pins the painter's contract by its raw SGR bytes. The marker,
// every piece, the pad and the badge contribute one open+close pair each, so
// the literals below state both the segment boundaries and the pad law: an
// empty pad renders bare, and never as a zero-width escape pair.
func TestPaintRow(t *testing.T) {
	// One hex color per style keeps the generated sequences readable: the
	// marker paints #101010, the command #202020, the text #303030, the
	// match #404040, the pad #505050, the dim text #606060, the plain match
	// #707070 and the plain name #808080.
	markerSt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#101010"))
	selName := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#202020"))
	selText := lipgloss.NewStyle().Foreground(lipgloss.Color("#303030"))
	selMatch := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#404040"))
	selPad := lipgloss.NewStyle().Foreground(lipgloss.Color("#505050"))
	dim := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#606060"))
	match := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#707070"))
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))

	// The command matches "build" and the description matches "uni" after
	// its separator space, as the filter tokens would mark them.
	cmdPiece := rowPiece{
		text:      "make build",
		matched:   []bool{false, false, false, false, false, true, true, true, true, true},
		base:      selName,
		highlight: selMatch,
	}
	plainCmdPiece := rowPiece{
		text:      "make build",
		matched:   []bool{false, false, false, false, false, true, true, true, true, true},
		base:      lipgloss.Style{},
		highlight: match,
	}
	descPiece := rowPiece{
		text: " run unit tests",
		matched: []bool{
			false, false, false, false,
			true, true, true, true,
			false, false, false, false, false, false, false,
		},
		base:      selText,
		highlight: selMatch,
	}
	plainDescPiece := rowPiece{
		text: " run unit tests",
		matched: []bool{
			false, false, false, false,
			true, true, true, true,
			false, false, false, false, false, false, false,
		},
		base:      dim,
		highlight: match,
	}

	tests := []struct {
		name     string
		selected bool
		marker   string
		pieces   []rowPiece
		badge    string
		budget   int
		styles   rowStyles
		want     string
	}{
		{
			name:     "selected row with pieces, pad and badge",
			selected: true,
			marker:   "▌ ",
			pieces:   []rowPiece{cmdPiece, descPiece},
			badge:    "make",
			budget:   40,
			styles:   rowStyles{marker: markerSt, pad: selPad, badge: selText},
			want: "\x1b[1;38;2;16;16;16m▌ \x1b[m" +
				"\x1b[1;38;2;32;32;32mmake \x1b[m" +
				"\x1b[1;38;2;64;64;64mbuild\x1b[m" +
				"\x1b[38;2;48;48;48m run\x1b[m" +
				"\x1b[1;38;2;64;64;64m uni\x1b[m" +
				"\x1b[38;2;48;48;48mt tests\x1b[m" +
				"\x1b[38;2;80;80;80m             \x1b[m" +
				"\x1b[38;2;48;48;48m make\x1b[m",
		},
		{
			name:     "selected row without pad room",
			selected: true,
			marker:   "▌ ",
			pieces:   []rowPiece{cmdPiece, descPiece},
			badge:    "make",
			budget:   24,
			styles:   rowStyles{marker: markerSt, pad: selPad, badge: selText},
			want: "\x1b[1;38;2;16;16;16m▌ \x1b[m" +
				"\x1b[1;38;2;32;32;32mmake \x1b[m" +
				"\x1b[1;38;2;64;64;64mbuild\x1b[m" +
				"\x1b[38;2;48;48;48m run\x1b[m" +
				"\x1b[1;38;2;64;64;64m uni\x1b[m" +
				"\x1b[38;2;48;48;48mt tests\x1b[m" +
				"\x1b[38;2;48;48;48m make\x1b[m",
		},
		{
			name:     "selected row at the exact budget paints no pad pair",
			selected: true,
			marker:   "▌ ",
			pieces:   []rowPiece{cmdPiece},
			badge:    "make",
			budget:   12,
			styles:   rowStyles{marker: markerSt, pad: selPad, badge: selText},
			want: "\x1b[1;38;2;16;16;16m▌ \x1b[m" +
				"\x1b[1;38;2;32;32;32mmake \x1b[m" +
				"\x1b[1;38;2;64;64;64mbuild\x1b[m" +
				"\x1b[38;2;48;48;48m make\x1b[m",
		},
		{
			name:   "plain row with pieces, pad and badge",
			marker: "  ",
			pieces: []rowPiece{plainCmdPiece, plainDescPiece},
			badge:  "make",
			budget: 40,
			styles: rowStyles{badge: dim},
			want: "  make " +
				"\x1b[1;38;2;112;112;112mbuild\x1b[m" +
				"\x1b[2;38;2;96;96;96m run\x1b[m" +
				"\x1b[1;38;2;112;112;112m uni\x1b[m" +
				"\x1b[2;38;2;96;96;96mt tests\x1b[m" +
				"              " +
				"\x1b[2;38;2;96;96;96mmake\x1b[m",
		},
		{
			name:   "plain row at the exact budget paints a bare separator",
			marker: "  ",
			pieces: []rowPiece{plainCmdPiece},
			badge:  "make",
			budget: 12,
			styles: rowStyles{badge: dim},
			want: "  make " +
				"\x1b[1;38;2;112;112;112mbuild\x1b[m" +
				" " +
				"\x1b[2;38;2;96;96;96mmake\x1b[m",
		},
		{
			name:     "selected theme row without a badge",
			selected: true,
			marker:   "▌ ",
			pieces:   []rowPiece{{text: "nord", base: selName}},
			badge:    "",
			budget:   10,
			styles:   rowStyles{marker: markerSt, pad: selPad},
			want: "\x1b[1;38;2;16;16;16m▌ \x1b[m" +
				"\x1b[1;38;2;32;32;32mnord\x1b[m" +
				"\x1b[38;2;80;80;80m    \x1b[m",
		},
		{
			name:     "selected theme row at the exact budget paints no pad pair",
			selected: true,
			marker:   "▌ ",
			pieces:   []rowPiece{{text: "nord", base: selName}},
			badge:    "",
			budget:   6,
			styles:   rowStyles{marker: markerSt, pad: selPad},
			want: "\x1b[1;38;2;16;16;16m▌ \x1b[m" +
				"\x1b[1;38;2;32;32;32mnord\x1b[m",
		},
		{
			name:   "plain theme row without a badge",
			marker: "  ",
			pieces: []rowPiece{{text: "nord", base: title}},
			badge:  "",
			budget: 10,
			styles: rowStyles{},
			want: "  " +
				"\x1b[38;2;128;128;128mnord\x1b[m" +
				"    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paintRow(tt.selected, tt.marker, tt.pieces, tt.badge, tt.budget, tt.styles); got != tt.want {
				t.Errorf("paintRow(selected: %v, ...) =\n%q\nwant\n%q", tt.selected, got, tt.want)
			}
		})
	}
}

// TestTruncateMatched pins the truncation window: the plain right cut, the
// window that shifts toward a match with its ellipsis, and the guards at the
// narrow edges of the width.
func TestTruncateMatched(t *testing.T) {
	tests := []struct {
		name        string
		s           string
		matched     []bool
		maxW        int
		wantS       string
		wantMatched []bool
	}{
		{
			name:        "the string that fits comes back unchanged",
			s:           "make build",
			matched:     []bool{false, false, false, false, true, true, true, true, true, true},
			maxW:        10,
			wantS:       "make build",
			wantMatched: []bool{false, false, false, false, true, true, true, true, true, true},
		},
		{
			name:  "right cut without a mask",
			s:     "make build",
			maxW:  5,
			wantS: "make…",
		},
		{
			name: "right cut marks the ellipsis as unmatched",
			s:    "make build",
			matched: []bool{
				false, false, false, false, false,
				false, false, false, false, false,
			},
			maxW:        5,
			wantS:       "make…",
			wantMatched: []bool{false, false, false, false, false},
		},
		{
			name: "the window shifts toward a match at the end",
			s:    "make build",
			matched: []bool{
				false, false, false, false, false,
				false, false, false, false, true,
			},
			maxW:        4,
			wantS:       "…ild",
			wantMatched: []bool{false, false, false, true},
		},
		{
			name: "a match in the middle frames a two-ellipsis window",
			s:    "make build",
			matched: []bool{
				false, false, false, false, false,
				true, true, true, true, true,
			},
			maxW:        4,
			wantS:       "…bu…",
			wantMatched: []bool{false, true, true, false},
		},
		{
			name: "a width too narrow for a window falls back to the right cut",
			s:    "abcde",
			matched: []bool{
				false, false, true, false, false,
			},
			maxW:  2,
			wantS: "a…",
		},
		{
			name:  "a width of one renders only the ellipsis",
			s:     "make build",
			maxW:  1,
			wantS: "…",
		},
		{
			name:  "the empty string truncates to itself",
			s:     "",
			maxW:  5,
			wantS: "",
		},
		{
			name:  "a width of zero truncates to the empty string",
			s:     "make",
			maxW:  0,
			wantS: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotS, gotMatched := truncateMatched(tt.s, tt.matched, tt.maxW)
			if gotS != tt.wantS {
				t.Errorf("truncateMatched(%q, %v, %d) s = %q, want %q", tt.s, tt.matched, tt.maxW, gotS, tt.wantS)
			}
			if !slices.Equal(gotMatched, tt.wantMatched) {
				t.Errorf("truncateMatched(%q, %v, %d) matched = %v, want %v", tt.s, tt.matched, tt.maxW, gotMatched, tt.wantMatched)
			}
		})
	}
}

// TestRenderMatched pins the per-segment pairs: the base renders once when no
// rune matches, and each matched run opens and closes exactly one pair.
func TestRenderMatched(t *testing.T) {
	// One hex color per style keeps the generated sequences readable.
	base := lipgloss.NewStyle().Foreground(lipgloss.Color("#202020"))
	highlight := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#404040"))

	tests := []struct {
		name    string
		s       string
		matched []bool
		want    string
	}{
		{
			name: "the empty string renders empty",
			want: "",
		},
		{
			name: "no match renders the base once",
			s:    "make build",
			want: "\x1b[38;2;32;32;32mmake build\x1b[m",
		},
		{
			name:    "one matched run opens one pair",
			s:       "make build",
			matched: []bool{false, false, false, false, false, true, true, true, true, true},
			want: "\x1b[38;2;32;32;32mmake \x1b[m" +
				"\x1b[1;38;2;64;64;64mbuild\x1b[m",
		},
		{
			name:    "adjacent runs split at every boundary",
			s:       "abxcd",
			matched: []bool{true, true, false, true, true},
			want: "\x1b[1;38;2;64;64;64mab\x1b[m" +
				"\x1b[38;2;32;32;32mx\x1b[m" +
				"\x1b[1;38;2;64;64;64mcd\x1b[m",
		},
		{
			name:    "a mask shorter than the text renders the tail unmatched",
			s:       "make",
			matched: []bool{true, true},
			want: "\x1b[1;38;2;64;64;64mma\x1b[m" +
				"\x1b[38;2;32;32;32mke\x1b[m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderMatched(tt.s, tt.matched, base, highlight); got != tt.want {
				t.Errorf("renderMatched(%q, %v, ...) =\n%q\nwant\n%q", tt.s, tt.matched, got, tt.want)
			}
		})
	}
}

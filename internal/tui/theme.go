package tui

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/lipgloss/v2"
)

// theme is one named look of the picker: the palette plus the structural
// choices the variants experiment with (how the row under the cursor stands
// out). Themes are pure data: newStyles turns one into the styles a model
// renders with.
type theme struct {
	name string
	desc string
	// accent highlights the title, the filter prompt and the selected row;
	// nil means the terminal's own foreground (the mono variant).
	accent color.Color
	// dim, ok and err colour secondary text, task success and task failure;
	// a nil colour leaves the terminal's own foreground.
	dim, ok, err color.Color
	// selBg is the background of the whole selected row in bar selection:
	// a lighter surface tone, so the row still reads as selected without
	// the saturated accent of the fill.
	selBg color.Color
	sel   selection
}

// selection is how the row under the cursor stands out.
type selection int

const (
	selFill    selection = iota // whole-row accent background, light text
	selBar                      // accent bar and name over a selBg-tinted row
	selReverse                  // terminal reverse video, no colors
)

// defaultTheme is the theme used when none is requested.
const defaultTheme = "classic"

// themes is the registry, in the order ctrl+t cycles through it. The first
// entry is the default, so the picker looks exactly the same when no theme
// is requested.
var themes = []theme{
	{
		name:   "classic",
		desc:   "the default violet look, filled selection",
		accent: lipgloss.Color("#7D56F4"),
		dim:    lipgloss.Color("241"),
		ok:     lipgloss.Color("2"),
		err:    lipgloss.Color("1"),
		sel:    selFill,
	},
	{
		name:   "nord",
		desc:   "arctic blues and frost, bar selection on a tinted row",
		accent: lipgloss.Color("#88C0D0"),
		dim:    lipgloss.Color("#4C566A"),
		ok:     lipgloss.Color("#A3BE8C"),
		err:    lipgloss.Color("#BF616A"),
		selBg:  lipgloss.Color("#434C5E"),
		sel:    selBar,
	},
	{
		name:   "dracula",
		desc:   "high-contrast plum, bar selection on a tinted row",
		accent: lipgloss.Color("#BD93F9"),
		dim:    lipgloss.Color("#6272A4"),
		ok:     lipgloss.Color("#50FA7B"),
		err:    lipgloss.Color("#FF5555"),
		selBg:  lipgloss.Color("#44475A"),
		sel:    selBar,
	},
	{
		name:   "tokyo",
		desc:   "night blues on ink, bar selection on a tinted row",
		accent: lipgloss.Color("#7AA2F7"),
		dim:    lipgloss.Color("#565F89"),
		ok:     lipgloss.Color("#9ECE6A"),
		err:    lipgloss.Color("#F7768E"),
		selBg:  lipgloss.Color("#3B4261"),
		sel:    selBar,
	},
	{
		name:   "gruvbox",
		desc:   "warm retro amber, bar selection on a tinted row",
		accent: lipgloss.Color("#FABD2F"),
		dim:    lipgloss.Color("#928374"),
		ok:     lipgloss.Color("#B8BB26"),
		err:    lipgloss.Color("#FB4934"),
		selBg:  lipgloss.Color("#504945"),
		sel:    selBar,
	},
	{
		name: "mono",
		desc: "no colors at all, reverse-video selection",
		sel:  selReverse,
	},
}

// themeIndex returns the index of the named theme; an empty name means the
// default one, and an unknown name reports false.
func themeIndex(name string) (int, bool) {
	if name == "" {
		return 0, true
	}
	for i, t := range themes {
		if t.name == name {
			return i, true
		}
	}
	return 0, false
}

// ListThemes writes the available themes to w, one per line, with the
// default marked. It backs '<prog> --list-themes'.
func ListThemes(w io.Writer) {
	width := 0
	for _, t := range themes {
		width = max(width, len(t.name))
	}
	for i, t := range themes {
		mark := " "
		if i == 0 {
			mark = "*"
		}
		fmt.Fprintf(w, "%s %-*s  %s\n", mark, width, t.name, t.desc)
	}
}

// styles are the concrete lipgloss styles a model renders with. They are
// derived from a theme and stateless once built, so a model can swap them
// (ctrl+t) without touching anything else.
type styles struct {
	title        lipgloss.Style // the program name
	filter       lipgloss.Style // the filter prompt and its cursor
	selected     lipgloss.Style // a whole selected row (fill or reverse)
	selectedName lipgloss.Style // bar selection: the name on the tinted row
	selectedDim  lipgloss.Style // bar selection: desc and badge on the tinted row
	selectedPad  lipgloss.Style // bar selection: the padding between them
	marker       lipgloss.Style // bar selection: the cursor marker
	dim          lipgloss.Style
	ok           lipgloss.Style
	err          lipgloss.Style

	markerText string // marker on the selected row ("› " or "▌ ")
	bar        bool   // selection is a bar over a tinted row
}

// newStyles builds the styles for a theme. A nil color leaves the
// terminal's own foreground (or faint, for dim).
func newStyles(t theme) styles {
	accent := lipgloss.NewStyle()
	if t.accent != nil {
		accent = accent.Foreground(t.accent)
	}

	colour := func(c color.Color, fallback lipgloss.Style) lipgloss.Style {
		if c == nil {
			return fallback
		}
		return lipgloss.NewStyle().Foreground(c)
	}

	s := styles{
		title:      accent.Bold(true),
		filter:     accent.Bold(true),
		dim:        colour(t.dim, lipgloss.NewStyle().Faint(true)),
		ok:         colour(t.ok, lipgloss.NewStyle()),
		err:        colour(t.err, lipgloss.NewStyle()),
		markerText: "› ",
	}
	switch t.sel {
	case selBar:
		s.bar = true
		s.markerText = "▌ "

		// Every segment of the tinted row carries the background itself:
		// a nested style's reset would otherwise drop it before the pads.
		onTint := func(fg color.Color, bold bool) lipgloss.Style {
			st := lipgloss.NewStyle()
			if t.selBg != nil {
				st = st.Background(t.selBg)
			}
			if fg != nil {
				st = st.Foreground(fg)
			}
			if bold {
				st = st.Bold(true)
			}
			return st
		}
		s.marker = onTint(t.accent, true)
		s.selectedName = onTint(t.accent, true)
		s.selectedDim = onTint(t.dim, false)
		s.selectedPad = onTint(nil, false)
	case selReverse:
		s.selected = lipgloss.NewStyle().Reverse(true)
	default: // selFill
		if t.accent != nil {
			s.selected = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(t.accent)
		} else {
			s.selected = lipgloss.NewStyle().Reverse(true)
		}
	}
	return s
}

package tui

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/lipgloss/v2"
)

// theme is one named look of the picker: the palette plus the structural
// choices the variants experiment with (how the row under the cursor stands
// out, whether the whole view sits in a frame). Themes are pure data:
// newStyles turns one into the styles a model renders with.
type theme struct {
	name string
	desc string
	// accent highlights the title, the filter prompt and the selected row;
	// nil means the terminal's own foreground (the mono variant).
	accent color.Color
	// dim, ok and err colour secondary text, task success and task failure;
	// a nil colour leaves the terminal's own foreground.
	dim, ok, err color.Color
	sel          selection
	boxed        bool
}

// selection is how the row under the cursor stands out.
type selection int

const (
	selFill    selection = iota // full-row accent background
	selBar                      // accent bar and name, no background
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
		desc:   "arctic blues and frost, filled selection",
		accent: lipgloss.Color("#88C0D0"),
		dim:    lipgloss.Color("#4C566A"),
		ok:     lipgloss.Color("#A3BE8C"),
		err:    lipgloss.Color("#BF616A"),
		sel:    selFill,
	},
	{
		name:   "dracula",
		desc:   "high-contrast plum, filled selection",
		accent: lipgloss.Color("#BD93F9"),
		dim:    lipgloss.Color("#6272A4"),
		ok:     lipgloss.Color("#50FA7B"),
		err:    lipgloss.Color("#FF5555"),
		sel:    selFill,
	},
	{
		name:   "tokyo",
		desc:   "night blues on ink, bar selection",
		accent: lipgloss.Color("#7AA2F7"),
		dim:    lipgloss.Color("#565F89"),
		ok:     lipgloss.Color("#9ECE6A"),
		err:    lipgloss.Color("#F7768E"),
		sel:    selBar,
	},
	{
		name:   "gruvbox",
		desc:   "warm retro amber, bar selection",
		accent: lipgloss.Color("#FABD2F"),
		dim:    lipgloss.Color("#928374"),
		ok:     lipgloss.Color("#B8BB26"),
		err:    lipgloss.Color("#FB4934"),
		sel:    selBar,
	},
	{
		name:   "frame",
		desc:   "the classic palette in a box, bar selection",
		accent: lipgloss.Color("#7D56F4"),
		dim:    lipgloss.Color("241"),
		ok:     lipgloss.Color("2"),
		err:    lipgloss.Color("1"),
		sel:    selBar,
		boxed:  true,
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
	selectedName lipgloss.Style // the name on a bar-selected row
	marker       lipgloss.Style // the cursor marker on a bar-selected row
	dim          lipgloss.Style
	ok           lipgloss.Style
	err          lipgloss.Style
	frame        lipgloss.Style // the box around the view, when the theme has one

	markerText string // marker on the selected row ("› " or "▌ ")
	bar        bool   // selection is a bar and colored text, not a fill
	boxed      bool   // draw frame around the whole view
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
		s.marker = accent.Bold(true)
		s.selectedName = accent.Bold(true)
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
	if t.boxed {
		s.boxed = true
		border := t.accent
		if border == nil {
			border = t.dim
		}
		if border == nil {
			border = lipgloss.Color("240")
		}
		s.frame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border)
	}
	return s
}

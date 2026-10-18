package tui

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/lipgloss/v2"
)

// theme is one named look of the picker: the palette plus the tint of the row
// under the cursor. Every theme shares the same selection style (an accent
// bar and name over a tinted row); the colorless one tints with reverse
// video. Themes are pure data: newStyles turns one into the styles a model
// renders with.
type theme struct {
	name string
	desc string
	// accent highlights the title, the filter prompt and the marker of the
	// selected row; nil means the terminal's own foreground (the mono variant).
	accent color.Color
	// dim, ok and err colour secondary text, task success and task failure;
	// a nil colour leaves the terminal's own foreground.
	dim, ok, err color.Color
	// selBg is the background of the whole selected row: a lighter surface
	// tone, so the row reads as selected without a saturated accent
	// background. nil in the colorless theme, which tints with reverse video.
	selBg color.Color
	// selFg is the foreground of the name, description and badge on that
	// tinted row: the palette's light tone, kept readable on selBg (the
	// dim color may be nearly identical to it, e.g. nord). nil in mono.
	selFg color.Color
}

// defaultTheme is the theme used when none is requested.
const defaultTheme = "classic"

// themes is the registry, in the order ctrl+t cycles through it. The first
// entry is the default, so the picker looks exactly the same when no theme
// is requested.
var themes = []theme{
	{
		name:   "classic",
		desc:   "the default violet look",
		accent: lipgloss.Color("#7D56F4"),
		dim:    lipgloss.Color("241"),
		ok:     lipgloss.Color("2"),
		err:    lipgloss.Color("1"),
		selBg:  lipgloss.Color("#463C6E"),
		selFg:  lipgloss.Color("#EDE9FE"),
	},
	{
		name:   "nord",
		desc:   "arctic blues and frost",
		accent: lipgloss.Color("#88C0D0"),
		dim:    lipgloss.Color("#4C566A"),
		ok:     lipgloss.Color("#A3BE8C"),
		err:    lipgloss.Color("#BF616A"),
		selBg:  lipgloss.Color("#434C5E"),
		selFg:  lipgloss.Color("#D8DEE9"),
	},
	{
		name:   "dracula",
		desc:   "high-contrast plum",
		accent: lipgloss.Color("#BD93F9"),
		dim:    lipgloss.Color("#6272A4"),
		ok:     lipgloss.Color("#50FA7B"),
		err:    lipgloss.Color("#FF5555"),
		selBg:  lipgloss.Color("#44475A"),
		selFg:  lipgloss.Color("#F8F8F2"),
	},
	{
		name:   "tokyo",
		desc:   "night blues on ink",
		accent: lipgloss.Color("#7AA2F7"),
		dim:    lipgloss.Color("#565F89"),
		ok:     lipgloss.Color("#9ECE6A"),
		err:    lipgloss.Color("#F7768E"),
		selBg:  lipgloss.Color("#3B4261"),
		selFg:  lipgloss.Color("#C0CAF5"),
	},
	{
		name:   "gruvbox",
		desc:   "warm retro amber",
		accent: lipgloss.Color("#FABD2F"),
		dim:    lipgloss.Color("#928374"),
		ok:     lipgloss.Color("#B8BB26"),
		err:    lipgloss.Color("#FB4934"),
		selBg:  lipgloss.Color("#504945"),
		selFg:  lipgloss.Color("#EBDBB2"),
	},
	{
		name:   "catppuccin",
		desc:   "soft pastel mocha",
		accent: lipgloss.Color("#CBA6F7"),
		dim:    lipgloss.Color("#7F849C"),
		ok:     lipgloss.Color("#A6E3A1"),
		err:    lipgloss.Color("#F38BA8"),
		selBg:  lipgloss.Color("#45475A"),
		selFg:  lipgloss.Color("#CDD6F4"),
	},
	{
		name:   "one-dark",
		desc:   "the editor classic: cool grays and blue",
		accent: lipgloss.Color("#61AFEF"),
		dim:    lipgloss.Color("#5C6370"),
		ok:     lipgloss.Color("#98C379"),
		err:    lipgloss.Color("#E06C75"),
		selBg:  lipgloss.Color("#3E4451"),
		selFg:  lipgloss.Color("#ABB2BF"),
	},
	{
		name:   "monokai",
		desc:   "vivid pink, orange and green",
		accent: lipgloss.Color("#FD971F"),
		dim:    lipgloss.Color("#75715E"),
		ok:     lipgloss.Color("#A6E22E"),
		err:    lipgloss.Color("#F92672"),
		selBg:  lipgloss.Color("#49483E"),
		selFg:  lipgloss.Color("#F8F8F2"),
	},
	{
		name:   "solarized",
		desc:   "the low-contrast classic: teal and blue",
		accent: lipgloss.Color("#268BD2"),
		dim:    lipgloss.Color("#586E75"),
		ok:     lipgloss.Color("#859900"),
		err:    lipgloss.Color("#DC322F"),
		selBg:  lipgloss.Color("#073642"),
		selFg:  lipgloss.Color("#93A1A1"),
	},
	{
		name:   "everforest",
		desc:   "muted forest greens",
		accent: lipgloss.Color("#A7C080"),
		dim:    lipgloss.Color("#859289"),
		ok:     lipgloss.Color("#83C092"),
		err:    lipgloss.Color("#E67E80"),
		selBg:  lipgloss.Color("#3D484D"),
		selFg:  lipgloss.Color("#D3C6AA"),
	},
	{
		name:   "kanagawa",
		desc:   "ukiyo-e ink blues and sakura",
		accent: lipgloss.Color("#7E9CD8"),
		dim:    lipgloss.Color("#727169"),
		ok:     lipgloss.Color("#98BB6C"),
		err:    lipgloss.Color("#E46876"),
		selBg:  lipgloss.Color("#2D4F67"),
		selFg:  lipgloss.Color("#DCD7BA"),
	},
	{
		name:   "rose-pine",
		desc:   "dusty rose, pine and iris",
		accent: lipgloss.Color("#C4A7E7"),
		dim:    lipgloss.Color("#6E6A86"),
		ok:     lipgloss.Color("#9CCFD8"),
		err:    lipgloss.Color("#EB6F92"),
		selBg:  lipgloss.Color("#403D52"),
		selFg:  lipgloss.Color("#E0DEF4"),
	},
	{
		name: "mono",
		desc: "no colors at all, reverse-video tint",
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
	selectedName lipgloss.Style // the command on the selected row
	selectedText lipgloss.Style // desc and badge on the selected row
	selectedPad  lipgloss.Style // the padding between them
	marker       lipgloss.Style // the cursor marker
	dim          lipgloss.Style
	ok           lipgloss.Style
	err          lipgloss.Style
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

	// Every segment of the selected row carries the highlight itself: a
	// nested style's reset would otherwise drop it before the pads. The
	// highlight is the theme's tint, or reverse video for the colorless
	// theme.
	onRow := func(fg color.Color, bold bool) lipgloss.Style {
		st := lipgloss.NewStyle()
		if t.selBg != nil {
			st = st.Background(t.selBg)
		} else {
			st = st.Reverse(true)
		}
		if fg != nil {
			st = st.Foreground(fg)
		}
		if bold {
			st = st.Bold(true)
		}
		return st
	}

	return styles{
		title:        accent.Bold(true),
		filter:       accent.Bold(true),
		dim:          colour(t.dim, lipgloss.NewStyle().Faint(true)),
		ok:           colour(t.ok, lipgloss.NewStyle()),
		err:          colour(t.err, lipgloss.NewStyle()),
		marker:       onRow(t.accent, true),
		selectedName: onRow(t.selFg, true),
		selectedText: onRow(t.selFg, false),
		selectedPad:  onRow(nil, false),
	}
}

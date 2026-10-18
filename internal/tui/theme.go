package tui

import (
	"fmt"
	"image/color"
	"io"
	"math"

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
	systemTheme(nil, nil),
	{
		name: "mono",
		desc: "no colors at all, reverse-video tint",
	},
}

// systemTheme builds the "system" theme like OpenCode's generateSystem:
// ANSI palette slots for chromatic roles (cyan accent, green ok, red err) and
// a surface tint and muted gray derived from the terminal's background and
// foreground colors (falling back to dark ANSI defaults before the terminal
// reports them).
func systemTheme(bg, fg color.Color) theme {
	bgR, bgG, bgB := 0.0, 0.0, 0.0
	if bg != nil {
		bgR, bgG, bgB = rgb8(bg)
	}
	isDark := colorLuminance(bgR, bgG, bgB) <= 127.5
	selBg := systemGray(bgR, bgG, bgB, isDark, 3)
	return theme{
		name:   "system",
		desc:   "the terminal's own palette and background",
		accent: lipgloss.Color("6"),
		dim:    systemMutedColor(bgR, bgG, bgB, isDark),
		ok:     lipgloss.Color("2"),
		err:    lipgloss.Color("1"),
		selBg:  selBg,
		selFg:  systemSelectedFg(fg, selBg, isDark),
	}
}

// rgb8 returns the 0..255 sRGB channels of c.
func rgb8(c color.Color) (r, g, b float64) {
	r16, g16, b16, _ := c.RGBA()
	return float64(r16 >> 8), float64(g16 >> 8), float64(b16 >> 8)
}

// colorLuminance returns the perceived brightness of an 8-bit RGB color
// (OpenCode's 0.299*R + 0.587*G + 0.114*B formula in 0..255).
func colorLuminance(r, g, b float64) float64 {
	return 0.299*r + 0.587*g + 0.114*b
}

// systemGray returns step i (1..12) of OpenCode's generateGrayScale: a surface
// shade shifted toward white on dark backgrounds or toward black on light ones
// while preserving the background's hue.
func systemGray(bgR, bgG, bgB float64, isDark bool, step int) color.Color {
	factor := float64(step) / 12.0
	lum := colorLuminance(bgR, bgG, bgB)

	if isDark && lum < 10 {
		g := int(math.Floor(factor * 0.4 * 255))
		return hexColor(g, g, g)
	}
	if !isDark && lum > 245 {
		g := int(math.Floor(255 - factor*0.4*255))
		return hexColor(g, g, g)
	}

	next := lum * (1 - factor*0.4)
	if isDark {
		next = lum + (255-lum)*factor*0.4
	}
	ratio := next / lum
	clamp := func(v float64) int {
		return int(math.Floor(min(max(v, 0), 255)))
	}
	return hexColor(clamp(bgR*ratio), clamp(bgG*ratio), clamp(bgB*ratio))
}

// systemMutedColor ports OpenCode's generateMutedTextColor: a neutral gray
// scaled to stay legible against the terminal background.
func systemMutedColor(bgR, bgG, bgB float64, isDark bool) color.Color {
	lum := colorLuminance(bgR, bgG, bgB)
	if isDark {
		g := 180
		if lum >= 10 {
			g = min(int(math.Floor(160+lum*0.3)), 200)
		}
		return hexColor(g, g, g)
	}
	g := 75
	if lum <= 245 {
		g = max(int(math.Floor(100-(255-lum)*0.2)), 60)
	}
	return hexColor(g, g, g)
}

// systemSelectedFg chooses the foreground for the selected row in the system
// theme: the terminal's own foreground (or OpenCode's ANSI 7/0 fallback),
// falling back to pure white or black if a mid-tone background leaves the
// contrast below WCAG AA (4.5:1).
func systemSelectedFg(fg, selBg color.Color, isDark bool) color.Color {
	var candidate color.Color
	if fg != nil {
		r, g, b := rgb8(fg)
		candidate = hexColor(int(r), int(g), int(b))
	} else if isDark {
		candidate = lipgloss.Color("#C0C0C0")
	} else {
		candidate = lipgloss.Color("#000000")
	}
	if contrastRatio(candidate, selBg) >= 4.5 {
		return candidate
	}
	white := lipgloss.Color("#FFFFFF")
	black := lipgloss.Color("#000000")
	if contrastRatio(white, selBg) >= contrastRatio(black, selBg) {
		return white
	}
	return black
}

// hexColor formats 8-bit RGB channels as a lipgloss hex color.
func hexColor(r, g, b int) color.Color {
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
}

// contrastRatio returns the WCAG contrast ratio between two colors.
func contrastRatio(a, b color.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance returns the WCAG relative luminance of a color.
func relativeLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA() // 16-bit channels
	linear := func(v uint32) float64 {
		s := float64(v) / 0xFFFF
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
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
	title         lipgloss.Style // the program name
	filter        lipgloss.Style // the filter prompt and its cursor
	selectedName  lipgloss.Style // the command on the selected row
	selectedText  lipgloss.Style // desc and badge on the selected row
	selectedMatch lipgloss.Style // matched filter characters on the selected row
	selectedPad   lipgloss.Style // the padding between them
	marker        lipgloss.Style // the cursor marker
	match         lipgloss.Style // matched filter characters on unselected rows
	dim           lipgloss.Style
	ok            lipgloss.Style
	err           lipgloss.Style
}

// newStyles builds the styles for a theme. A nil color leaves the
// terminal's own foreground (or faint, for dim).
func newStyles(t theme) styles {
	accent := lipgloss.NewStyle()
	if t.accent != nil {
		accent = accent.Foreground(t.accent)
	}

	match := accent.Bold(true)
	if t.accent == nil {
		match = match.Underline(true)
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
		title:         accent.Bold(true),
		filter:        accent.Bold(true),
		dim:           colour(t.dim, lipgloss.NewStyle().Faint(true)),
		ok:            colour(t.ok, lipgloss.NewStyle()),
		err:           colour(t.err, lipgloss.NewStyle()),
		marker:        onRow(t.accent, true),
		match:         match,
		selectedName:  onRow(t.selFg, true),
		selectedText:  onRow(t.selFg, false),
		selectedMatch: onRow(t.accent, true).Underline(true),
		selectedPad:   onRow(nil, false),
	}
}

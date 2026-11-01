package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The warning modal and the theme modal drew the same box by hand, so a border
// or fitting bug had to be fixed twice. The box module owns the border protocol
// once, so the fix lands in one place. The caller owns the content: the title
// text, the body rows, the footer text and the inner width, which is a content
// fact. The placement is the caller's business (see placedBox in tui.go).

// boxChrome is the number of rows that the box adds around the body: two
// borders and the footer.
const boxChrome = 3

// box is a bordered drawing laid over the frame: a title on the top border,
// body rows, a footer and the two borders, all fitted to one column width.
type box struct {
	title  string      // plain text; render draws it bold in the border color
	body   []string    // rendered rows; render truncates and pads each to innerW
	footer string      // rendered; render pads it to innerW or drops it when wider
	border color.Color // nil = the terminal's default foreground
	innerW int         // content width; the caller computes it from its content
}

// render draws the box in h x w and returns its lines, top border first.
// It returns nil when the space is too small for a usable box.
func (b box) render(h, w int) []string {
	if h < boxChrome+1 || w < 10 {
		return nil
	}
	innerW := min(max(b.innerW, 1), max(w-4, 1))
	bw := innerW + 4

	edge := lipgloss.NewStyle()
	if b.border != nil {
		edge = edge.Foreground(b.border)
	}

	title := edge.Bold(true).Render(b.title)
	title = ansi.Truncate(title, max(bw-3, 0), "")

	lines := make([]string, 0, len(b.body)+boxChrome)
	lines = append(lines,
		edge.Render("╭─")+title+
			edge.Render(strings.Repeat("─", max(bw-3-lipgloss.Width(title), 0)))+
			edge.Render("╮"))
	for _, line := range b.body {
		line = ansi.Truncate(line, innerW, "…")
		pad := strings.Repeat(" ", max(innerW-lipgloss.Width(line), 0))
		lines = append(lines, edge.Render("│")+" "+line+pad+" "+edge.Render("│"))
	}
	footer := b.footer
	if lipgloss.Width(footer) > innerW {
		footer = ""
	}
	lines = append(lines,
		edge.Render("│")+" "+footer+strings.Repeat(" ", max(innerW-lipgloss.Width(footer), 0))+" "+edge.Render("│"),
		edge.Render("╰")+edge.Render(strings.Repeat("─", bw-2))+edge.Render("╯"))
	return lines
}

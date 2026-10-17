package tui

import (
	"bytes"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var ctrlTKey = tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}

// titleLine returns the first rendered line, without ANSI sequences.
func titleLine(view string) string {
	return ansi.Strip(strings.SplitN(view, "\n", 2)[0])
}

func TestThemeRegistry(t *testing.T) {
	if themes[0].name != defaultTheme {
		t.Errorf("first theme = %q, want the default %q", themes[0].name, defaultTheme)
	}
	seen := map[string]bool{}
	for _, th := range themes {
		if th.name == "" || th.desc == "" {
			t.Errorf("theme %+v has an empty name or description", th)
		}
		if seen[th.name] {
			t.Errorf("duplicate theme name %q", th.name)
		}
		if th.sel == selBar && (th.selBg == nil || th.selFg == nil) {
			t.Errorf("theme %q uses bar selection but misses selBg or selFg", th.name)
		}
		seen[th.name] = true
	}
}

// TestThemeContrast keeps the selected row readable: the light text of a bar
// theme must stand out from its tint (WCAG AA, 4.5:1). The nord theme once
// failed this with the dim color on a nearly identical tint.
func TestThemeContrast(t *testing.T) {
	for _, th := range themes {
		if th.sel != selBar {
			continue
		}
		if got := contrastRatio(th.selFg, th.selBg); got < 4.5 {
			t.Errorf("theme %q: selFg on selBg contrast = %.1f:1, want at least 4.5:1", th.name, got)
		}
	}
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

func TestThemeIndex(t *testing.T) {
	tests := []struct {
		name string
		want int
		ok   bool
	}{
		{"", 0, true},
		{defaultTheme, 0, true},
		{"nord", 1, true},
		{"nope", 0, false},
	}
	for _, tt := range tests {
		got, ok := themeIndex(tt.name)
		if got != tt.want || ok != tt.ok {
			t.Errorf("themeIndex(%q) = %d, %v; want %d, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestListThemes(t *testing.T) {
	var buf bytes.Buffer
	ListThemes(&buf)

	for i, th := range themes {
		if !strings.Contains(buf.String(), th.name) || !strings.Contains(buf.String(), th.desc) {
			t.Errorf("ListThemes does not mention %q", th.name)
		}
		mark := " "
		if i == 0 {
			mark = "*"
		}
		if !strings.Contains(buf.String(), mark+" "+th.name) {
			t.Errorf("ListThemes does not mark theme %q with %q", th.name, mark)
		}
	}
}

func TestUnknownThemeWarnsAndFallsBack(t *testing.T) {
	m := newModel(Options{Theme: "nope"}, sampleTasks(), nil)
	if got := themes[m.themeIdx].name; got != defaultTheme {
		t.Errorf("theme after an unknown name = %q, want %q", got, defaultTheme)
	}
	if len(m.warnings) == 0 || !strings.Contains(m.warnings[0].Error(), `"nope"`) {
		t.Fatalf("warnings after an unknown name = %v, want one mentioning it", m.warnings)
	}

	got := sized(Options{Theme: "nope"}).View().Content
	if !strings.Contains(got, "unknown theme") || !strings.Contains(got, "nope") {
		t.Errorf("View() does not report the unknown theme:\n%s", got)
	}
}

func TestCtrlTCyclesThemes(t *testing.T) {
	m := sized(Options{})
	first := themes[m.themeIdx].name

	seen := map[string]bool{first: true}
	for range len(themes) - 1 {
		next, _ := m.Update(ctrlTKey)
		m = next.(model)
		seen[themes[m.themeIdx].name] = true
	}
	if len(seen) != len(themes) {
		t.Errorf("ctrl+t visited %d themes, want %d: %v", len(seen), len(themes), seen)
	}

	// One more press wraps back to the first theme.
	next, _ := m.Update(ctrlTKey)
	if m = next.(model); themes[m.themeIdx].name != first {
		t.Errorf("theme after a full cycle = %q, want %q", themes[m.themeIdx].name, first)
	}
	if m.filter != "" {
		t.Errorf("filter after ctrl+t = %q, want empty (it cycles, it does not type)", m.filter)
	}
}

func TestViewShowsActiveTheme(t *testing.T) {
	if got := titleLine(sized(Options{}).View().Content); got != "wtr" {
		t.Errorf("default title = %q, want %q", got, "wtr")
	}
	if got := titleLine(sized(Options{Theme: "nord"}).View().Content); got != "wtr · nord" {
		t.Errorf("title with the nord theme = %q, want %q", got, "wtr · nord")
	}
}

func TestBarSelectionTintsTheWholeRow(t *testing.T) {
	view := sized(Options{Theme: "tokyo"}).View().Content

	var selected string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "▌") {
			selected = line
			break
		}
	}
	if selected == "" {
		t.Fatalf("bar-selected view does not use the bar marker:\n%s", view)
	}

	// The tint arrives as a background color, and the text on it is the
	// palette's light tone: never the dim color, which can be nearly
	// identical to the tint (the nord bug) and make the row unreadable.
	if !strings.Contains(selected, "48;2;59;66;97") { // #3B4261
		t.Errorf("selected row does not paint the theme tint:\n%q", selected)
	}
	if !strings.Contains(selected, "38;2;192;202;245") { // #C0CAF5
		t.Errorf("selected row text is not the theme's light tone:\n%q", selected)
	}
	if strings.Contains(selected, "38;2;86;95;137") { // tokyo dim #565F89
		t.Errorf("selected row still uses the dim color for part of its text:\n%q", selected)
	}

	if strings.Contains(sized(Options{}).View().Content, "▌") {
		t.Error("the default theme uses the bar marker, want the arrow")
	}
}

func TestEveryThemeRenders(t *testing.T) {
	for _, th := range themes {
		t.Run(th.name, func(t *testing.T) {
			full := sized(Options{Theme: th.name}).View().Content
			if !strings.Contains(full, "build") || !strings.Contains(full, "4 tasks") {
				t.Errorf("fullscreen view of theme %q misses the task list:\n%s", th.name, full)
			}

			inline := sized(Options{Inline: true, Theme: th.name}).View().Content
			if !strings.Contains(inline, "build") {
				t.Errorf("inline view of theme %q misses the task list:\n%s", th.name, inline)
			}
		})
	}
}

func TestHelpThemeHintFits(t *testing.T) {
	if got := sized(Options{}).View().Content; !strings.Contains(got, "ctrl+t theme") {
		t.Errorf("fullscreen help line does not mention ctrl+t:\n%s", got)
	}

	// Inline at 80 columns the hint would push the line over the edge: it is
	// dropped, and the keys before it stay readable.
	if got := sized(Options{Inline: true}).View().Content; strings.Contains(got, "ctrl+t theme") {
		t.Errorf("inline help line at 80 columns keeps the ctrl+t hint, want it dropped:\n%s", got)
	}

	m := update(newModel(Options{Inline: true}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})
	if got := m.View().Content; !strings.Contains(got, "ctrl+t theme") {
		t.Errorf("inline help line at 120 columns dropped the ctrl+t hint:\n%s", got)
	}
}

func TestPreviewRendersAFrame(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // ignore the user's rc file
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\techo building\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Preview(&buf, Options{Theme: "tokyo"}, 80, 24); err != nil {
		t.Fatalf("Preview: %v", err)
	}

	got := buf.String()
	if n := strings.Count(got, "\n"); n != 24 {
		t.Errorf("preview height = %d rows, want 24", n)
	}
	for _, want := range []string{"build", "tokyo"} {
		if !strings.Contains(got, want) {
			t.Errorf("preview does not contain %q:\n%s", want, got)
		}
	}
}

func TestPreviewWithoutTasks(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := Preview(&bytes.Buffer{}, Options{}, 80, 24); err == nil {
		t.Error("Preview without tasks returned nil, want an error")
	}
}

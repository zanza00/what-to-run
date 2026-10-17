package tui

import (
	"bytes"
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
		if th.sel == selBar && th.selBg == nil {
			t.Errorf("theme %q uses bar selection but has no selBg tint", th.name)
		}
		seen[th.name] = true
	}
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
	got := sized(Options{Theme: "tokyo"}).View().Content
	if !strings.Contains(got, "▌") {
		t.Errorf("bar-selected view does not use the bar marker:\n%s", got)
	}
	// The tint of the selected row arrives as a background color; the bar
	// itself must not be the only highlight.
	if !strings.Contains(got, "48;2;59;66;97") { // #3B4261
		t.Errorf("bar-selected view does not paint the row with the theme tint:\n%q", got)
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

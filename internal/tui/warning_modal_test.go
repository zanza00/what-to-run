package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	ctrlWKey = tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}
	dKey     = tea.KeyPressMsg{Code: 'd', Text: "d"}
)

func TestWarningModalOpensAtStartAndKeepsBottomChrome(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	longErr := errors.New("rc file /home/zanza00/.config/what-to-run/config.toml: strict mode: fields in the document are missing in the target struct")
	m := update(newModel(Options{}, sampleTasks(), []error{longErr}),
		tea.WindowSizeMsg{Width: 80, Height: 24})

	if !m.warningModal {
		t.Fatal("warningModal at startup = false, want true for an undismissed warning")
	}

	got := ansi.Strip(m.View().Content)
	// The box is composited over the rows, so rebuild its text from the
	// interior of its borders before checking wrapped messages.
	var box []string
	for _, line := range strings.Split(got, "\n") {
		if i, j := strings.Index(line, "│"), strings.LastIndex(line, "│"); i >= 0 && j > i {
			box = append(box, line[i+len("│"):j])
		}
	}
	text := strings.Join(strings.Fields(strings.Join(box, " ")), " ")
	for _, want := range []string{
		"rc file /home/zanza00/.config/what-to-run/",
		"config.toml: strict mode: fields in the document are missing in the target struct",
		warningModalHints,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("warning modal does not contain %q:\n%s", want, got)
		}
	}
	for _, want := range []string{
		"warning",
		"4 tasks · 1 warning",
		"enter run · esc quit",
		"ctrl+w warnings",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("View() with warning modal does not contain %q:\n%s", want, got)
		}
	}
}

func TestWarningModalCloseAndReopenWithCtrlW(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	for _, closeKey := range []tea.KeyPressMsg{enterKey, escKey, ctrlWKey} {
		m := update(newModel(Options{}, sampleTasks(), []error{errors.New("Cargo.toml: boom")}),
			tea.WindowSizeMsg{Width: 80, Height: 24})
		m.filter = "build"

		next, cmd := m.Update(closeKey)
		m = next.(model)
		if m.warningModal {
			t.Errorf("warningModal after %v = true, want false", closeKey)
		}
		if cmd != nil {
			t.Errorf("closing the warning modal with %v returned a command, want none", closeKey)
		}
		if m.filter != "build" {
			t.Errorf("filter after closing warning modal = %q, want %q", m.filter, "build")
		}
		got := ansi.Strip(m.View().Content)
		if strings.Contains(got, "╰") {
			t.Errorf("warning box is still drawn after closing:\n%s", got)
		}
		if !strings.Contains(got, "1 warning") || !strings.Contains(got, "ctrl+w warnings") {
			t.Errorf("bottom chrome lost the warning indicator after closing:\n%s", got)
		}

		// ctrl+w reopens the warning box.
		m = update(m, ctrlWKey)
		if !m.warningModal {
			t.Error("ctrl+w did not reopen the warning modal")
		}
	}

	// Without warnings, ctrl+w does nothing.
	m := update(sized(Options{}), ctrlWKey)
	if m.warningModal {
		t.Error("ctrl+w opened the warning modal when there are no warnings")
	}
}

func TestWarningModalIgnoresTyping(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	m := update(newModel(Options{}, sampleTasks(), []error{errors.New("Cargo.toml: boom")}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	m = press(m, "npm")
	if m.filter != "" {
		t.Errorf("filter while warning modal is open = %q, want empty", m.filter)
	}
	if !m.warningModal {
		t.Error("typing closed the warning modal")
	}
}

func TestWarningModalDismissWithD(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	warn := errors.New("rc file ~/.config/what-to-run/config.toml: unknown theme \"nope\"")
	m := update(newModel(Options{}, sampleTasks(), []error{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.warningModal {
		t.Fatal("warningModal did not open for a new warning")
	}

	m = update(m, dKey)
	if m.warningModal {
		t.Error("pressing d did not close the warning modal")
	}

	data, err := os.ReadFile(filepath.Join(cacheDir, "what-to-run", "dismissed-warnings"))
	if err != nil {
		t.Fatalf("reading dismissed-warnings: %v", err)
	}
	if !strings.Contains(string(data), warn.Error()) {
		t.Errorf("dismissed-warnings = %q, want %q", data, warn.Error())
	}

	// Next run with the same warning: the modal stays closed at startup,
	// while the bottom chrome still shows "1 warning" and ctrl+w opens it.
	again := update(newModel(Options{}, sampleTasks(), []error{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if again.warningModal {
		t.Error("dismissed warning reopened the modal at startup")
	}
	got := ansi.Strip(again.View().Content)
	if !strings.Contains(got, "4 tasks · 1 warning") || !strings.Contains(got, "ctrl+w warnings") {
		t.Errorf("bottom chrome does not show the dismissed warning count:\n%s", got)
	}
	again = update(again, ctrlWKey)
	if !again.warningModal {
		t.Error("ctrl+w did not open a dismissed warning")
	}

	// A different warning still opens the modal at startup.
	other := newModel(Options{}, sampleTasks(), []error{errors.New("Makefile: syntax error")})
	if !other.warningModal {
		t.Error("a new undismissed warning did not open the modal at startup")
	}
}

func TestWarningModalMultipleWarnings(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	warnings := []error{
		errors.New("first warning"),
		errors.New("second warning"),
	}
	m := update(newModel(Options{}, sampleTasks(), warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})

	got := ansi.Strip(m.View().Content)
	if !strings.Contains(got, "warnings 1/2 ↓") || !strings.Contains(got, "first warning") {
		t.Errorf("modal does not show the first of two warnings:\n%s", got)
	}
	if !strings.Contains(got, "4 tasks · 2 warnings") {
		t.Errorf("bottom chrome does not report 2 warnings:\n%s", got)
	}

	m = update(m, downKey)
	got = ansi.Strip(m.View().Content)
	if !strings.Contains(got, "↑ warnings 2/2") || !strings.Contains(got, "second warning") {
		t.Errorf("modal after down does not show the second warning:\n%s", got)
	}

	m = update(m, homeKey)
	if m.warningIdx != 0 {
		t.Errorf("warningIdx after home = %d, want 0", m.warningIdx)
	}
	m = update(m, endKey)
	if m.warningIdx != 1 {
		t.Errorf("warningIdx after end = %d, want 1", m.warningIdx)
	}

	// Pressing d on warning 2 dismisses it and advances to the remaining
	// undismissed warning 1; pressing d again dismisses warning 1 and
	// closes the modal.
	m = update(m, dKey)
	if !m.warningModal || m.warningIdx != 0 {
		t.Errorf("after dismissing warning 2: modal = %v, idx = %d; want open on 0", m.warningModal, m.warningIdx)
	}
	m = update(m, dKey)
	if m.warningModal {
		t.Error("after dismissing all warnings: modal is still open")
	}
}

func TestWarningModalFitsInlineAndNarrowTerminals(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	longErr := errors.New("rc file /home/zanza00/.config/what-to-run/config.toml: strict mode: fields in the document are missing in the target struct")
	m := newModel(Options{Inline: true}, sampleTasks(), []error{longErr})
	m = update(m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m = update(m, tea.CursorPositionMsg{Y: 12}) // 15-row inline area

	got := m.View().Content
	if n := strings.Count(got, "\n"); n != 14 {
		t.Errorf("inline view with warning modal has %d lines, want 15", n+1)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := ansi.StringWidth(line); w > 40 {
			t.Errorf("narrow view line is %d columns wide, want at most 40: %q", w, line)
		}
	}
}

func TestSavingThemeReParsesConfigAndClearsResolvedWarning(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfgDir := filepath.Join(dir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))

	userDir := filepath.Join(cfgDir, "what-to-run")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, userDir, "config.toml", "theme = \"nope\"\n")
	writeFile(t, dir, "Makefile", "build:\n\techo ok\n")

	tasks, cfg, warnings := discover()
	if len(warnings) != 1 {
		t.Fatalf("initial warnings = %v, want 1", warnings)
	}
	m := update(newModel(Options{Theme: cfg.Theme}, tasks, warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.warningModal {
		t.Fatal("warningModal did not open for the broken theme")
	}

	// Close the warning modal, open the theme picker, and confirm a valid
	// theme: saving re-parses config.toml and clears the resolved warning.
	m = update(m, escKey)
	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	m = update(m, enterKey)

	if len(m.warnings) != 0 {
		t.Errorf("warnings after saving a valid theme = %v, want none", m.warnings)
	}
	got := ansi.Strip(m.View().Content)
	if strings.Contains(got, "1 warning") || strings.Contains(got, "ctrl+w") {
		t.Errorf("bottom chrome still shows a warning after fixing config.toml:\n%s", got)
	}
}

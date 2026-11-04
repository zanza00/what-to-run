package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/zanza00/what-to-run/internal/core"
)

var (
	ctrlWKey = tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}
	dKey     = tea.KeyPressMsg{Code: 'd', Text: "d"}
)

func TestWarningModalOpensAtStartAndKeepsBottomChrome(t *testing.T) {

	longErr := core.NewRCWarning("user", "syntax", "",
		"rc file /home/zanza00/.config/what-to-run/config.toml: strict mode: fields in the document are missing in the target struct")
	m := update(newModel(Options{}, sampleTasks(), []core.Warning{longErr}),
		tea.WindowSizeMsg{Width: 80, Height: 24})

	if !m.warningModal {
		t.Fatal("warningModal at startup = false, want true for an undismissed warning")
	}

	got := ansi.Strip(m.View().Content)
	// The box is composited over the rows, so the test rebuilds its text from
	// the interior of its borders before it checks the wrapped messages.
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

	for _, closeKey := range []tea.KeyPressMsg{enterKey, escKey, ctrlWKey} {
		m := update(newModel(Options{}, sampleTasks(), []core.Warning{
			core.NewSourceWarning("cargo-make", "Makefile.toml", "Makefile.toml: boom")}),
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

	m := update(newModel(Options{}, sampleTasks(), []core.Warning{
		core.NewSourceWarning("cargo-make", "Makefile.toml", "Makefile.toml: boom")}),
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
	env, files := testEnv(t)
	warn := core.NewThemeWarning("user", "nope",
		"rc file ~/.config/what-to-run/config.toml: unknown theme \"nope\"")
	m := update(newModel(Options{Env: env}, sampleTasks(), []core.Warning{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.warningModal {
		t.Fatal("warningModal did not open for a new warning")
	}

	next, cmd := m.Update(dKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.warningModal {
		t.Error("pressing d did not close the warning modal")
	}

	data, ok := files[env.CacheFile]
	if !ok {
		t.Fatalf("dismissed-warnings = missing, want %q", warn.Key())
	}
	if got, want := data, warn.Key()+"\n"; got != want {
		t.Errorf("dismissed-warnings = %q, want the key alone on its line: %q", got, want)
	}

	// The next run with the same warning keeps the modal closed at startup,
	// while the bottom chrome still shows "1 warning" and ctrl+w opens it.
	againEnv := *env
	againEnv.Dismissed = m.env.Dismissed
	again := update(newModel(Options{Env: &againEnv}, sampleTasks(), []core.Warning{warn}),
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
	other := newModel(Options{}, sampleTasks(), []core.Warning{sampleWarning("Makefile: syntax error")})
	if !other.warningModal {
		t.Error("a new undismissed warning did not open the modal at startup")
	}
}

func TestWarningModalMultipleWarnings(t *testing.T) {
	env, _ := testEnv(t)
	warnings := []core.Warning{
		core.NewSourceWarning("make", "Makefile", "first warning"),
		core.NewSourceWarning("make", "GNUmakefile", "second warning"),
	}
	m := update(newModel(Options{Env: env}, sampleTasks(), warnings),
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

	// Pressing d on warning 2 dismisses it and advances to warning 1, which is
	// still undismissed. Pressing d again dismisses warning 1 and closes the
	// modal.
	next, cmd := m.Update(dKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if !m.warningModal || m.warningIdx != 0 {
		t.Errorf("after dismissing warning 2: modal = %v, idx = %d; want open on 0", m.warningModal, m.warningIdx)
	}
	next, cmd = m.Update(dKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.warningModal {
		t.Error("after dismissing all warnings: modal is still open")
	}
}

func TestWarningModalFitsInlineAndNarrowTerminals(t *testing.T) {
	longErr := core.NewRCWarning("user", "syntax", "",
		"rc file /home/zanza00/.config/what-to-run/config.toml: strict mode: fields in the document are missing in the target struct")
	m := newModel(Options{Inline: true}, sampleTasks(), []core.Warning{longErr})
	m = update(m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m = update(m, tea.CursorPositionMsg{Y: 12}) // The cursor position leaves a 15-row inline area.

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

// TestWarningModalPlacement pins the three branches of the warning box
// placement, and the crossings where the branches exchange. Each case is
// arithmetic over the frame facts alone: h is the frame height, listRows is
// the count of list rows, and boxH is the height of the placed box. The
// heights stay inside the range that the box module draws.
func TestWarningModalPlacement(t *testing.T) {
	tests := []struct {
		name     string
		h        int
		listRows int
		boxH     int
		want     int
	}{
		{"centered above a tall list", 24, 20, 5, 9},
		{"one row of room lands on row 1", 24, 3, 5, 1},
		{"flush at the top when the box reaches the help line", 24, 2, 5, 0},
		{"right below the help line when the box is taller than the list", 24, 2, 8, 6},
		{"below the help line while the list holds three rows", 24, 3, 7, 7},
		{"the crossing: one more list row centers above the help line", 24, 4, 7, 0},
		{"the last row of room keeps the box below the help line", 14, 2, 8, 6},
		{"the crossing: no row below the box centers it in the view", 13, 2, 8, 2},
		{"the help row stops one row above the view height", 10, 50, 4, 2},
		{"centered in the view on a tiny terminal", 10, 1, 8, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := warningModalY(tt.h, tt.listRows, tt.boxH); got != tt.want {
				t.Errorf("warningModalY(h=%d, listRows=%d, boxH=%d) = %d, want %d",
					tt.h, tt.listRows, tt.boxH, got, tt.want)
			}
		})
	}
}

func TestSavingThemeReParsesConfigAndClearsResolvedWarning(t *testing.T) {
	env, files := testEnv(t)
	files[env.UserFile] = "theme = \"nope\"\n"
	files[env.ProjectFile] = "[[tasks]]\nname = \"build\"\ncommand = \"echo ok\"\n"

	tasks, cfg, warnings := env.discover()
	if len(warnings) != 1 {
		t.Fatalf("initial warnings = %v, want 1", warnings)
	}
	m := update(newModel(Options{Theme: cfg.Theme, Env: env}, tasks, warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.warningModal {
		t.Fatal("warningModal did not open for the broken theme")
	}

	// With the warning modal closed, the theme picker confirms a valid theme.
	// Saving re-parses config.toml and clears the resolved warning.
	m = update(m, escKey)
	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	if len(m.warnings) != 0 {
		t.Errorf("warnings after saving a valid theme = %v, want none", m.warnings)
	}
	if got := files[env.UserFile]; !strings.Contains(got, `theme = "nord"`) {
		t.Errorf("user rc = %q, want the nord theme", got)
	}
	got := ansi.Strip(m.View().Content)
	if strings.Contains(got, "1 warning") || strings.Contains(got, "ctrl+w") {
		t.Errorf("bottom chrome still shows a warning after fixing config.toml:\n%s", got)
	}
}

// TestLegacyDismissedLineStillMatches pins the migration: a dismissal that an
// older version wrote holds the message text, not the key. The startup check
// still matches it, so an upgrade does not pop the modal open again, and the
// prune still clears it once the warning heals.
func TestLegacyDismissedLineStillMatches(t *testing.T) {
	env, files := testEnv(t)
	warn := core.NewThemeWarning("user", "nope",
		`rc file ~/.config/what-to-run/config.toml: unknown theme "nope"`)
	files[env.CacheFile] = warningText(warn) + "\n"
	env.Dismissed = loadDismissedWarnings(files, env.CacheFile)

	m := update(newModel(Options{Env: env}, sampleTasks(), []core.Warning{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.warningModal {
		t.Error("a legacy entry did not dismiss the warning at startup")
	}
}

// TestModelCollapsesWarningsWithOneKey pins the identity of the listing: the
// picker holds one warning per underlying problem. A bad disable entry can
// arrive twice, from the rc loader's validation and from core's safety net,
// and the same typo can sit in two rc files. Equal keys make one warning, and
// the first occurrence's message wins the display, because it names the file.
func TestModelCollapsesWarningsWithOneKey(t *testing.T) {
	env, files := testEnv(t)
	files[env.UserFile] = "disable = [\"nmake\"]\n"
	files[env.ProjectFile] = "disable = [\"nmake\"]\n"

	_, _, configWarnings := env.discover()
	// The safety net of core sees a hand-built Settings that still carries
	// the entry the loader would drop.
	_, sourceWarnings := core.Discover(env.Files, env.Dir, core.Settings{Disable: []string{"nmake"}})
	warnings := append(configWarnings, sourceWarnings...)
	if len(warnings) < 2 {
		t.Fatalf("producers = %v, want the disable entry twice", warnings)
	}

	m := update(newModel(Options{Env: env}, sampleTasks(), warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(m.warnings) != 1 {
		t.Fatalf("model warnings = %v, want the twin collapsed into one", m.warnings)
	}
	if m.warnings[0].Kind() != core.KindRegistry {
		t.Errorf("kind = %s, want registry", m.warnings[0].Kind())
	}
	if !strings.Contains(m.warnings[0].Error(), "config.toml") {
		t.Errorf("kept message = %q, want the first occurrence, which names the rc file", m.warnings[0].Error())
	}
}

// TestThemeSaveRunsOnlyForThemeWarnings pins the guard of the theme-save
// re-discovery on the warning kind. A warning that merely mentions "theme" in
// its message does not trigger the save; a theme warning does, even with the
// chosen theme unchanged.
func TestThemeSaveRunsOnlyForThemeWarnings(t *testing.T) {
	env, files := testEnv(t)

	// A source warning whose message mentions "theme".
	m := update(newModel(Options{Env: env}, sampleTasks(), []core.Warning{
		core.NewSourceWarning("make", "GNUmakefile", "Makefile theme is broken")}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	m = update(m, escKey) // close the warning modal that opened at startup
	m = update(m, ctrlTKey)
	next, cmd := m.Update(enterKey)
	if cmd != nil {
		t.Errorf("the theme save ran for a source warning: %v", cmd())
	}
	m = next.(model)
	if _, ok := files[env.UserFile]; ok {
		t.Error("the theme save wrote the rc file for a source warning")
	}

	// A theme warning, with the chosen theme unchanged.
	m = update(newModel(Options{Env: env}, sampleTasks(), []core.Warning{
		core.NewThemeWarning("user", "nope", `rc file ~/.config/what-to-run/config.toml: unknown theme "nope"`)}),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	m = update(m, escKey) // close the warning modal that opened at startup
	m = update(m, ctrlTKey)
	next, cmd = m.Update(enterKey)
	if cmd == nil {
		t.Fatal("the theme save did not run for a theme warning")
	}
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if len(m.warnings) != 0 {
		t.Errorf("warnings after the theme-save re-discovery = %v, want none", m.warnings)
	}
	if got := files[env.UserFile]; !strings.Contains(got, `theme = "classic"`) {
		t.Errorf("user rc = %q, want the default theme written", got)
	}
}

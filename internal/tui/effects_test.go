package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/zanza00/what-to-run/internal/config"
)

// TestThemeSaveEffectReportsTheSavedTheme pins the success message of the
// theme save, and the state that it produces.
func TestThemeSaveEffectReportsTheSavedTheme(t *testing.T) {
	env, files := testEnv(t)
	files[env.ProjectFile] = "[[tasks]]\nname = \"from-rc\"\ncommand = \"echo ok\"\n"

	m := update(newModel(Options{Env: env}, sampleTasks(), nil),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)

	msg, ok := execCmd(t, cmd).(themeSavedMsg)
	if !ok {
		t.Fatal("theme save command did not return a themeSavedMsg")
	}
	if msg.err != nil {
		t.Fatalf("theme save error = %v, want none", msg.err)
	}
	if msg.name != "nord" {
		t.Errorf("message name = %q, want nord", msg.name)
	}
	if msg.path != env.UserFile {
		t.Errorf("message path = %q, want %q", msg.path, env.UserFile)
	}
	if got, want := taskNames(msg.tasks), []string{"from-rc"}; !slices.Equal(got, want) {
		t.Errorf("message tasks = %v, want %v", got, want)
	}
	if len(msg.warnings) != 0 {
		t.Errorf("message warnings = %v, want none", msg.warnings)
	}
	if got := files[env.UserFile]; !strings.Contains(got, `theme = "nord"`) {
		t.Errorf("user rc = %q, want the nord theme", got)
	}

	m = update(m, msg)
	if got := themes[m.themeIdx].name; got != "nord" {
		t.Errorf("theme after the save = %q, want nord", got)
	}
	if m.opts.Theme != "nord" {
		t.Errorf("opts.Theme after the save = %q, want nord", m.opts.Theme)
	}
	wantNotice := fmt.Sprintf("theme nord saved to %s", config.ShortPath(env.UserFile, env.Home))
	if m.notice != wantNotice {
		t.Errorf("notice after the save = %q, want %q", m.notice, wantNotice)
	}
	if m.status != "" {
		t.Errorf("status after the save = %q, want it untouched", ansi.Strip(m.status))
	}
	if got, want := taskNames(m.tasks), []string{"from-rc"}; !slices.Equal(got, want) {
		t.Errorf("tasks after the save = %v, want %v", got, want)
	}
}

// TestThemeSaveEffectReportsTheError pins the failure message of the theme
// save, and that the model keeps the previous theme.
func TestThemeSaveEffectReportsTheError(t *testing.T) {
	env, files := testEnv(t)
	env.Store = errStore{Store: env.Store, err: errors.New("disk full")}

	m := update(newModel(Options{Env: env}, sampleTasks(), nil),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)

	msg, ok := execCmd(t, cmd).(themeSavedMsg)
	if !ok {
		t.Fatal("theme save command did not return a themeSavedMsg")
	}
	if msg.err == nil {
		t.Fatal("theme save error = nil, want the write error")
	}
	if msg.name != "nord" {
		t.Errorf("message name = %q, want nord", msg.name)
	}

	m = update(m, msg)
	if m.notice != "" {
		t.Errorf("notice after a failed save = %q, want none", m.notice)
	}
	if got, want := ansi.Strip(m.status), "cannot save theme nord: disk full"; got != want {
		t.Errorf("status after a failed save = %q, want %q", got, want)
	}
	if m.opts.Theme != "" {
		t.Errorf("opts.Theme after a failed save = %q, want it unchanged", m.opts.Theme)
	}
	if _, ok := files[env.UserFile]; ok {
		t.Error("a failed save wrote the user rc file")
	}
}

// TestThemeSaveClearsResolvedDismissals pins that a save removes a dismissed
// warning that is no longer active from the cache and from the model.
func TestThemeSaveClearsResolvedDismissals(t *testing.T) {
	env, files := testEnv(t)
	warn := errors.New(`rc file ~/.config/what-to-run/config.toml: unknown theme "nope"`)
	files[env.UserFile] = "theme = \"nope\"\n"
	files[env.CacheFile] = warningKey(warn) + "\n"
	env.Dismissed = loadDismissedWarnings(files, env.CacheFile)

	_, cfg, warnings := env.discover()
	if len(warnings) != 1 {
		t.Fatalf("discover() warnings = %v, want the broken theme", warnings)
	}
	m := update(newModel(Options{Theme: cfg.Theme, Env: env}, sampleTasks(), warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.warningModal {
		t.Fatal("a dismissed warning opened the modal at startup")
	}

	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	if _, ok := files[env.CacheFile]; ok {
		t.Error("the resolved dismissal is still in the cache")
	}
	if len(m.env.Dismissed) != 0 {
		t.Errorf("dismissed set after the save = %v, want empty", m.env.Dismissed)
	}
}

// TestThemeSaveSeesADismissalFromAnotherProcess pins that the save reloads the
// cache before it clears a resolved warning. The model loaded an empty set, and
// another process dismissed the warning after that.
func TestThemeSaveSeesADismissalFromAnotherProcess(t *testing.T) {
	env, files := testEnv(t)
	files[env.UserFile] = "theme = \"nope\"\n"

	_, cfg, warnings := env.discover()
	if len(warnings) != 1 {
		t.Fatalf("discover() warnings = %v, want the broken theme", warnings)
	}
	m := update(newModel(Options{Theme: cfg.Theme, Env: env}, sampleTasks(), warnings),
		tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(m.env.Dismissed) != 0 {
		t.Fatalf("model dismissed set = %v, want empty", m.env.Dismissed)
	}

	// Another process dismisses the warning after the model was built.
	files[env.CacheFile] = warningKey(warnings[0]) + "\n"

	m = update(m, escKey) // close the warning modal that opened at startup
	m = update(m, ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	if _, ok := files[env.CacheFile]; ok {
		t.Error("the resolved dismissal is still in the cache")
	}
	if len(m.env.Dismissed) != 0 {
		t.Errorf("dismissed set after the save = %v, want empty", m.env.Dismissed)
	}
}

// TestWarningDismissEffectReportsTheDismissedWarning pins the success message of
// the warning dismissal, and the state that it produces.
func TestWarningDismissEffectReportsTheDismissedWarning(t *testing.T) {
	env, files := testEnv(t)
	warn := errors.New("first warning")
	m := update(newModel(Options{Env: env}, sampleTasks(), []error{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})

	next, cmd := m.Update(dKey)
	m = next.(model)
	msg, ok := execCmd(t, cmd).(warningDismissedMsg)
	if !ok {
		t.Fatal("dismiss command did not return a warningDismissedMsg")
	}
	if msg.err != nil {
		t.Fatalf("dismiss error = %v, want none", msg.err)
	}
	if msg.idx != 0 {
		t.Errorf("message idx = %d, want 0", msg.idx)
	}
	if !msg.dismissed[warningKey(warn)] {
		t.Errorf("message dismissed = %v, want the warning", msg.dismissed)
	}
	if got, want := files[env.CacheFile], warningKey(warn)+"\n"; got != want {
		t.Errorf("cache = %q, want %q", got, want)
	}

	m = update(m, msg)
	if m.warningModal {
		t.Error("the modal is still open after the dismissal")
	}
	if !m.env.Dismissed[warningKey(warn)] {
		t.Errorf("model dismissed set = %v, want the warning", m.env.Dismissed)
	}
}

// TestWarningDismissEffectReportsTheError pins the failure message of the
// warning dismissal, and that the modal closes.
func TestWarningDismissEffectReportsTheError(t *testing.T) {
	env, files := testEnv(t)
	env.Store = errStore{Store: env.Store, err: errors.New("cache locked")}
	warn := errors.New("first warning")
	m := update(newModel(Options{Env: env}, sampleTasks(), []error{warn}),
		tea.WindowSizeMsg{Width: 80, Height: 24})

	next, cmd := m.Update(dKey)
	m = next.(model)
	msg, ok := execCmd(t, cmd).(warningDismissedMsg)
	if !ok {
		t.Fatal("dismiss command did not return a warningDismissedMsg")
	}
	if msg.err == nil {
		t.Fatal("dismiss error = nil, want the write error")
	}

	m = update(m, msg)
	if m.warningModal {
		t.Error("the modal is still open after a failed save")
	}
	if got, want := ansi.Strip(m.status), "cannot save dismissed warning: cache locked"; got != want {
		t.Errorf("status after a failed save = %q, want %q", got, want)
	}
	if _, ok := files[env.CacheFile]; ok {
		t.Error("a failed save wrote the cache file")
	}
}

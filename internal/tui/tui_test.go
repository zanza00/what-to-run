package tui

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// update feeds msg to m and returns the resulting concrete model.
func update(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

// press feeds one key press per rune of s, as if the user typed it.
func press(m model, s string) model {
	for _, r := range s {
		m = update(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

// sampleTasks returns a small task list for the UI tests.
func sampleTasks() []Task {
	return []Task{
		{Source: "package.json", Name: "build", Desc: "go build ./...", Args: []string{"npm", "run", "build"}},
		{Source: "Makefile", Name: "build-all", Desc: "Build everything", Args: []string{"make", "build-all"}},
		{Source: "Makefile", Name: "test", Desc: "Run the tests", Args: []string{"make", "test"}},
		{Source: "Cargo.toml", Name: "clean", Desc: "", Args: []string{"cargo", "make", "clean"}},
	}
}

// sized returns a model after the terminal size (and, inline, the cursor
// position query) has been answered, as in a running session.
func sized(opts Options) model {
	m := update(newModel(opts, sampleTasks(), nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	if opts.Inline {
		m = update(m, tea.CursorPositionMsg{Y: 5})
	}
	return m
}

var (
	enterKey  = tea.KeyPressMsg{Code: tea.KeyEnter}
	tabKey    = tea.KeyPressMsg{Code: tea.KeyTab}
	escKey    = tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlCKey  = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlUKey  = tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	upKey     = tea.KeyPressMsg{Code: tea.KeyUp}
	downKey   = tea.KeyPressMsg{Code: tea.KeyDown}
	homeKey   = tea.KeyPressMsg{Code: tea.KeyHome}
	endKey    = tea.KeyPressMsg{Code: tea.KeyEnd}
	pgUpKey   = tea.KeyPressMsg{Code: tea.KeyPgUp}
	pgDownKey = tea.KeyPressMsg{Code: tea.KeyPgDown}
)

func TestViewRendersTasks(t *testing.T) {
	got := sized(Options{}).View().Content
	for _, want := range []string{"wtr", "build", "test", "clean", "type to filter", "4 tasks"} {
		if !strings.Contains(got, want) {
			t.Errorf("View() content does not contain %q:\n%s", want, got)
		}
	}
}

func TestViewShowsCommandLines(t *testing.T) {
	got := sized(Options{}).View().Content
	for _, want := range []string{
		"npm run build", "make build-all", "cargo make clean", // commands
		"go build ./...", "Build everything", "Run the tests", // descriptions
	} {
		if !strings.Contains(got, want) {
			t.Errorf("View() does not show %q:\n%s", want, got)
		}
	}
}

func TestViewHidesRepeatedDescription(t *testing.T) {
	// An rc task without a custom description defaults to its command: the
	// row must not print it twice.
	tasks := []Task{{Source: "rc", Name: "status", Desc: "git status", Args: []string{"sh", "-c", "git status"}}}
	got := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10}).View().Content
	if n := strings.Count(got, "git status"); n != 1 {
		t.Errorf("View() shows %q %d times, want once:\n%s", "git status", n, got)
	}
}

func TestFilteringByCommandLine(t *testing.T) {
	// The name is not a column any more: filtering has to find what the
	// rows show, e.g. the runner.
	m := press(sized(Options{}), "npm")
	if vis := m.visible(); len(vis) != 1 || vis[0].Name != "build" {
		t.Errorf("visible tasks with filter %q = %+v, want just the build task", m.filter, vis)
	}
}

func TestViewKeepsCommandOnNarrowTerminals(t *testing.T) {
	// At 44 columns the description barely fits: the command must stay
	// readable, since it is what will actually run.
	m := update(newModel(Options{}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 44, Height: 10})
	got := m.View().Content
	if !strings.Contains(got, "npm run build") {
		t.Errorf("narrow View() dropped the command line:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 44 {
			t.Errorf("narrow View() line is %d columns wide, want at most 44: %q", w, line)
		}
	}
}

func TestViewTitleUsesInvokedName(t *testing.T) {
	got := sized(Options{Prog: "what-to-run"}).View().Content
	if !strings.Contains(got, "what-to-run") {
		t.Errorf("View() content does not contain the invoked name:\n%s", got)
	}
}

func TestViewFullscreenUsesAltScreen(t *testing.T) {
	if !sized(Options{}).View().AltScreen {
		t.Error("View() in fullscreen mode should set AltScreen")
	}
}

func TestViewInlineWaitsForCursorPosition(t *testing.T) {
	m := update(newModel(Options{Inline: true}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	if got := m.View().Content; got != "" {
		t.Errorf("View() before cursor position is known = %q, want empty", got)
	}

	m = update(m, tea.CursorPositionMsg{Y: 5})
	if got := m.View().Content; got == "" {
		t.Error("View() after cursor position is known is empty, want content")
	}
}

func TestViewInlineHeight(t *testing.T) {
	tests := []struct {
		name     string
		height   int
		startRow int
		want     int
	}{
		{"fills the space below the cursor", 40, 10, 30},
		{"exactly the minimum", 30, 15, 15},
		{"scrolls up to the minimum", 24, 23, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := update(newModel(Options{Inline: true}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 80, Height: tt.height})
			m = update(m, tea.CursorPositionMsg{Y: tt.startRow})

			v := m.View()
			if v.AltScreen {
				t.Error("View() in inline mode should not set AltScreen")
			}
			if got := strings.Count(v.Content, "\n") + 1; got != tt.want {
				t.Errorf("inline view height = %d, want %d", got, tt.want)
			}
			if got := strings.Count(v.Content, "╭"); got != 0 {
				t.Errorf("inline view has %d top-left corners, want 0 (the area border is opt-in)", got)
			}
		})
	}
}

func TestViewInlineDebugArea(t *testing.T) {
	m := update(newModel(Options{Inline: true, DebugArea: true}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5}) // 25 rows of UI

	v := m.View()
	if got := strings.Count(v.Content, "\n") + 1; got != 25 {
		t.Errorf("inline view height with the debug border = %d, want 25", got)
	}
	if got := strings.Count(v.Content, "╭"); got != 1 {
		t.Errorf("inline view has %d top-left corners, want 1 (the area border)", got)
	}
}

func TestViewFullscreenFillsHeightAndScrolls(t *testing.T) {
	tasks := make([]Task, 30)
	for i := range tasks {
		tasks[i] = Task{Source: "Makefile", Name: fmt.Sprintf("task-%02d", i), Args: []string{"make", fmt.Sprintf("task-%02d", i)}}
	}

	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10})

	if got := strings.Count(m.View().Content, "\n") + 1; got != 10 {
		t.Errorf("fullscreen view height = %d, want 10", got)
	}

	for range 8 {
		m = update(m, downKey)
	}
	if m.cursor != 8 {
		t.Fatalf("cursor = %d, want 8", m.cursor)
	}

	got := m.View().Content
	if !strings.Contains(got, "task-08") {
		t.Errorf("View() does not show the selected task-08:\n%s", got)
	}
	if strings.Contains(got, "task-00") || strings.Contains(got, "task-02") {
		t.Errorf("View() shows rows scrolled out of the window:\n%s", got)
	}
	if !strings.Contains(got, "task-03") {
		t.Errorf("View() does not show the first row of the window (task-03):\n%s", got)
	}
}

func TestViewShowsWarnings(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	m := update(newModel(Options{}, sampleTasks(), []error{errors.New("Cargo.toml: boom")}),
		tea.WindowSizeMsg{Width: 80, Height: 30})
	if got := m.View().Content; !strings.Contains(got, "Cargo.toml: boom") {
		t.Errorf("View() does not show the discovery warning:\n%s", got)
	}
}

func TestRestoreCursor(t *testing.T) {
	m := sized(Options{Inline: true}) // 25 rows of UI, cursor at the bottom

	var buf bytes.Buffer
	restoreCursor(&buf, m)
	if got, want := buf.String(), "\x1b[24A\x1b[J"; got != want {
		t.Errorf("restoreCursor wrote %q, want %q", got, want)
	}

	buf.Reset()
	restoreCursor(&buf, newModel(Options{Inline: true}, sampleTasks(), nil)) // never rendered
	if got := buf.String(); got != "" {
		t.Errorf("restoreCursor on an unrendered model wrote %q, want nothing", got)
	}
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"esc", escKey},
		{"ctrl+c", ctrlCKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := sized(Options{}).Update(tt.key)
			if cmd == nil {
				t.Fatalf("Update(%s) returned nil command, want tea.Quit", tt.name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("Update(%s) command returned %T, want tea.QuitMsg", tt.name, cmd())
			}
		})
	}
}

func TestQTypesInsteadOfQuitting(t *testing.T) {
	next, cmd := sized(Options{}).Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd != nil {
		t.Error("q returned a command, want none (it types into the filter)")
	}
	if got := next.(model).filter; got != "q" {
		t.Errorf("filter after q = %q, want %q", got, "q")
	}
}

func TestFiltering(t *testing.T) {
	m := press(sized(Options{}), "ts")
	if vis := m.visible(); len(vis) != 1 || vis[0].Name != "test" {
		t.Errorf("visible tasks with filter %q = %+v, want just the test task", m.filter, vis)
	}
	if got := m.View().Content; !strings.Contains(got, "1/4 tasks") {
		t.Errorf("View() does not show the filtered count:\n%s", got)
	}

	m = update(m, escKey)
	if m.filter != "" {
		t.Errorf("filter after esc = %q, want empty", m.filter)
	}
	if vis := m.visible(); len(vis) != 4 {
		t.Errorf("visible tasks after esc = %d, want 4", len(vis))
	}

	// esc with an empty filter quits.
	if _, cmd := m.Update(escKey); cmd == nil {
		t.Error("esc with an empty filter returned no command, want tea.Quit")
	}

	m = press(m, "cl")
	m = update(m, ctrlUKey)
	if m.filter != "" || len(m.visible()) != 4 {
		t.Errorf("ctrl+u did not clear the filter: filter=%q, visible=%d", m.filter, len(m.visible()))
	}
}

func TestFilterBackspace(t *testing.T) {
	m := press(sized(Options{}), "cl")
	m = update(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.filter != "c" {
		t.Errorf("filter after backspace = %q, want %q", m.filter, "c")
	}
}

func TestViewNoMatchingTask(t *testing.T) {
	m := press(sized(Options{}), "zzz")
	if got := m.View().Content; !strings.Contains(got, "no task matches") {
		t.Errorf("View() does not show the empty state:\n%s", got)
	}
}

func TestNavigation(t *testing.T) {
	m := sized(Options{})

	m = update(m, downKey)
	m = update(m, downKey)
	if m.cursor != 2 {
		t.Errorf("cursor after two downs = %d, want 2", m.cursor)
	}

	m = update(m, upKey)
	m = update(m, upKey)
	m = update(m, upKey) // clamped at the top
	if m.cursor != 0 {
		t.Errorf("cursor after ups = %d, want 0", m.cursor)
	}

	m = update(m, endKey)
	if m.cursor != 3 {
		t.Errorf("cursor after end = %d, want 3", m.cursor)
	}
	m = update(m, downKey) // clamped at the bottom
	if m.cursor != 3 {
		t.Errorf("cursor after down at the bottom = %d, want 3", m.cursor)
	}

	m = update(m, homeKey)
	if m.cursor != 0 {
		t.Errorf("cursor after home = %d, want 0", m.cursor)
	}

	m = update(m, pgDownKey)
	if m.cursor != 3 {
		t.Errorf("cursor after pgdown = %d, want 3", m.cursor)
	}
	m = update(m, pgUpKey)
	if m.cursor != 0 {
		t.Errorf("cursor after pgup = %d, want 0", m.cursor)
	}
}

func TestEnterRunsInPlaceFullscreen(t *testing.T) {
	next, cmd := sized(Options{}).Update(enterKey)
	if cmd == nil {
		t.Fatal("enter returned no command")
	}

	m := next.(model)
	if m.chosen != nil {
		t.Error("fullscreen enter should not mark a task to run after quitting")
	}
	if !strings.Contains(m.status, "running build") {
		t.Errorf("status after enter = %q, want it to mention the running task", m.status)
	}
}

func TestTaskDoneStatus(t *testing.T) {
	m := update(sized(Options{}), enterKey)

	done := update(m, taskDoneMsg{task: m.tasks[0]})
	if !strings.Contains(done.status, "✓ build") {
		t.Errorf("status after a successful run = %q, want it to start with a check mark", done.status)
	}

	failed := update(m, taskDoneMsg{task: m.tasks[0], err: errors.New("exit status 1")})
	if !strings.Contains(failed.status, "✗ build") || !strings.Contains(failed.status, "exit status 1") {
		t.Errorf("status after a failed run = %q, want the error", failed.status)
	}
}

func TestEnterQuitsWithChosenTaskInline(t *testing.T) {
	next, cmd := sized(Options{Inline: true}).Update(enterKey)
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("inline enter command returned %T, want tea.QuitMsg", cmd())
	}

	m := next.(model)
	if m.chosen == nil || m.chosen.Name != "build" {
		t.Fatalf("chosen task = %+v, want build", m.chosen)
	}
	if got, want := m.chosen.command().Args, []string{"npm", "run", "build"}; !slices.Equal(got, want) {
		t.Errorf("chosen command args = %v, want %v", got, want)
	}
}

func TestEnterRunsFilteredTask(t *testing.T) {
	m := press(sized(Options{Inline: true}), "cl")
	next, _ := m.Update(enterKey)
	if m := next.(model); m.chosen == nil || m.chosen.Name != "clean" {
		t.Errorf("chosen task = %+v, want clean", m.chosen)
	}
}

func TestEnterWithoutMatchesDoesNothing(t *testing.T) {
	m := press(sized(Options{}), "zzz")
	if _, cmd := m.Update(enterKey); cmd != nil {
		t.Errorf("enter with no matches returned %T, want no command", cmd)
	}
}

func TestTabChoosesInsertInline(t *testing.T) {
	next, cmd := sized(Options{Inline: true}).Update(tabKey)
	if cmd == nil {
		t.Fatal("inline tab returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("inline tab command returned %T, want tea.QuitMsg", cmd())
	}

	m := next.(model)
	if m.chosen == nil || m.chosen.Name != "build" {
		t.Fatalf("chosen task = %+v, want build", m.chosen)
	}
	if !m.chosenInsert {
		t.Error("chosenInsert after tab = false, want true")
	}
	if got, want := m.chosen.commandLine(), "npm run build"; got != want {
		t.Errorf("chosen command line = %q, want %q", got, want)
	}
}

func TestTabIgnoredFullscreen(t *testing.T) {
	next, cmd := sized(Options{}).Update(tabKey)
	if cmd != nil {
		t.Errorf("fullscreen tab returned %T, want no command", cmd)
	}
	if m := next.(model); m.chosen != nil {
		t.Errorf("fullscreen tab chose %+v, want nothing", m.chosen)
	}
}

func TestTabWithoutMatchesDoesNothing(t *testing.T) {
	m := press(sized(Options{Inline: true}), "zzz")
	if _, cmd := m.Update(tabKey); cmd != nil {
		t.Errorf("tab with no matches returned %T, want no command", cmd)
	}
}

func TestEnterChosenIsRunInline(t *testing.T) {
	next, _ := sized(Options{Inline: true}).Update(enterKey)
	if m := next.(model); m.chosenInsert {
		t.Error("chosenInsert after enter = true, want false")
	}
}

func TestReportChoice(t *testing.T) {
	tests := []struct {
		name   string
		insert bool
		want   string
	}{
		{"run", false, "run\nnpm run build\n"},
		{"insert", true, "insert\nnpm run build\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reportChoice(&buf, sampleTasks()[0], tt.insert)
			if got := buf.String(); got != tt.want {
				t.Errorf("reportChoice wrote %q, want %q", got, tt.want)
			}
		})
	}
}

func TestViewInlineHelpMentionsTab(t *testing.T) {
	if got := sized(Options{Inline: true}).View().Content; !strings.Contains(got, "tab to prompt") {
		t.Errorf("inline help line does not mention tab:\n%s", got)
	}
	if got := sized(Options{}).View().Content; strings.Contains(got, "tab to prompt") {
		t.Errorf("fullscreen help line mentions tab:\n%s", got)
	}
}

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
	"github.com/zanza00/what-to-run/internal/core"
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
func sampleTasks() []core.Task {
	return []core.Task{
		{Source: "package.json", Name: "build", Desc: "go build ./...", Args: []string{"npm", "run", "build"}},
		{Source: "Makefile", Name: "build-all", Desc: "Build everything", Args: []string{"make", "build-all"}},
		{Source: "Makefile", Name: "test", Desc: "Run the tests", Args: []string{"make", "test"}},
		{Source: "Makefile.toml", Name: "clean", Desc: "", Args: []string{"cargo", "make", "clean"}},
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
	ctrlFKey  = tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
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
	tasks := []core.Task{{Source: "rc", Name: "status", Desc: "git status", Args: []string{"sh", "-c", "git status"}}}
	got := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10}).View().Content
	if n := strings.Count(got, "git status"); n != 1 {
		t.Errorf("View() shows %q %d times, want once:\n%s", "git status", n, got)
	}
}

func TestViewShowsScriptBodyMatchingTaskName(t *testing.T) {
	// A package.json script like "knip": "knip" has the same text as the
	// script name in "pnpm run knip", but its script body must still be
	// shown in the description column.
	tasks := []core.Task{
		{Source: "package.json", Name: "test:watch", Desc: "vitest", Args: []string{"pnpm", "run", "test:watch"}},
		{Source: "package.json", Name: "knip", Desc: "knip", Args: []string{"pnpm", "run", "knip"}},
	}
	got := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10}).View().Content
	if n := strings.Count(got, "knip"); n != 2 {
		t.Errorf("View() shows %q %d times, want 2 (command and description):\n%s", "knip", n, got)
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

func TestFilteringByDescription(t *testing.T) {
	tasks := []core.Task{
		{Source: "package.json", Name: "build", Desc: "tsc && vite build", Args: []string{"npm", "run", "build"}},
		{Source: "Makefile", Name: "test", Desc: "Run the tests", Args: []string{"make", "test"}},
	}
	base := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10})

	for _, query := range []string{"tsc", "vite", "npm vite", "run tests"} {
		t.Run(query, func(t *testing.T) {
			m := press(base, query)
			vis := m.visible()
			if len(vis) != 1 {
				t.Fatalf("visible tasks with filter %q = %+v, want 1 match", query, vis)
			}
			want := "build"
			if query == "run tests" {
				want = "test"
			}
			if vis[0].Name != want {
				t.Errorf("matched task for %q = %q, want %q", query, vis[0].Name, want)
			}
		})
	}
}

func TestFilteringRanksAndRejectsScatteredCrossWordNoise(t *testing.T) {
	tasks := []core.Task{
		{
			Source: "package.json",
			Name:   "test:watch",
			Desc:   "vitest",
			Args:   []string{"pnpm", "run", "test:watch"},
		},
		{
			Source: "package.json",
			Name:   "type-strict-check",
			Desc:   "check types",
			Args:   []string{"pnpm", "run", "type-strict-check"},
		},
		{
			Source: "package.json",
			Name:   "build",
			Desc:   "tsc --noEmit && tsc --noEmit -p src/edge && vite build",
			Args:   []string{"pnpm", "run", "build"},
		},
		{
			Source: "package.json",
			Name:   "codes",
			Desc:   "node scripts/codes.mjs",
			Args:   []string{"pnpm", "run", "codes"},
		},
		{
			Source: "package.json",
			Name:   "icons",
			Desc:   "node scripts/icons.mjs",
			Args:   []string{"pnpm", "run", "icons"},
		},
		{
			Source: "package.json",
			Name:   "check",
			Desc:   "biome check && eslint src && pnpm run knip",
			Args:   []string{"pnpm", "run", "check"},
		},
		{
			Source: "package.json",
			Name:   "preview",
			Desc:   "vite --mode dev",
			Args:   []string{"pnpm", "run", "preview"},
		},
		{
			Source: "package.json",
			Name:   "dev:edge",
			Desc:   "wrangler dev",
			Args:   []string{"pnpm", "run", "dev:edge"},
		},
		{
			Source: "package.json",
			Name:   "dev",
			Desc:   "vite --host 0.0.0.0",
			Args:   []string{"pnpm", "run", "dev"},
		},
	}
	base := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 15})

	// "dev": exact task name first, then prefix name ("dev:edge"), then
	// description match ("preview"), while "build" ("src/edge && vite") is
	// rejected as cross-word mid-word noise.
	m := press(base, "dev")
	vis := m.visible()
	var names []string
	for _, task := range vis {
		names = append(names, task.Name)
	}
	if want := []string{"dev", "dev:edge", "preview"}; !slices.Equal(names, want) {
		t.Errorf("ranked tasks for %q = %v, want %v", "dev", names, want)
	}

	// "tsc": contiguous "tsc" in build's description beats a non-contiguous
	// acronym match in a task name ("type-strict-check"), and scattered
	// cross-word noise ("test:watch", "codes", "icons", "check") is rejected.
	m = press(base, "tsc")
	vis = m.visible()
	names = nil
	for _, task := range vis {
		names = append(names, task.Name)
	}
	if want := []string{"build", "type-strict-check"}; !slices.Equal(names, want) {
		t.Errorf("ranked tasks for %q = %v, want %v", "tsc", names, want)
	}
}

func TestMatchPositionsPrefersContiguousAndWordBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		tokens []string
		want   string // matched runes uppercased in text
	}{
		{
			name:   "contiguous substring wins over earlier scattered letter",
			text:   "env=dev && vite build",
			tokens: []string{"vite"},
			want:   "env=dev && VITE build",
		},
		{
			name:   "word boundaries win for initials",
			text:   "npm run build",
			tokens: []string{"nb"},
			want:   "Npm run Build",
		},
		{
			name:   "multiple tokens highlight both",
			text:   "tsc && vite build",
			tokens: []string{"tsc", "vite"},
			want:   "TSC && VITE build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mask := matchPositions(tt.text, tt.tokens)
			runes := []rune(tt.text)
			var b strings.Builder
			for i, r := range runes {
				if i < len(mask) && mask[i] {
					b.WriteString(strings.ToUpper(string(r)))
				} else {
					b.WriteRune(r)
				}
			}
			if got := b.String(); got != tt.want {
				t.Errorf("matchPositions(%q, %v) marked %q, want %q", tt.text, tt.tokens, got, tt.want)
			}
		})
	}
}

func TestViewHighlightsMatches(t *testing.T) {
	tasks := []core.Task{
		{Source: "package.json", Name: "build", Desc: "tsc && vite build", Args: []string{"npm", "run", "build"}},
		{Source: "package.json", Name: "dev", Desc: "vite --open", Args: []string{"npm", "run", "dev"}},
	}
	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10})
	m = press(m, "vite")

	got := m.View().Content
	// Selected row (build) highlights "vite" with selectedMatch; unselected
	// row (dev) highlights "vite" with match.
	wantSel := m.styles.selectedMatch.Render("vite")
	wantUnsel := m.styles.match.Render("vite")
	if !strings.Contains(got, wantSel) {
		t.Errorf("View() does not highlight %q on the selected row:\n%q", "vite", got)
	}
	if !strings.Contains(got, wantUnsel) {
		t.Errorf("View() does not highlight %q on the unselected row:\n%q", "vite", got)
	}
}

func TestViewWindowsLongDescriptionAroundMatch(t *testing.T) {
	tasks := []core.Task{
		{
			Source: "package.json",
			Name:   "build",
			Desc:   "tsc --noEmit && vite build --mode production && node scripts/postbuild.mjs",
			Args:   []string{"npm", "run", "build"},
		},
	}
	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 50, Height: 10})
	m = press(m, "postbuild")

	got := m.View().Content
	want := m.styles.selectedMatch.Render("postbuild")
	if !strings.Contains(got, want) {
		t.Errorf("narrow View() did not keep the matched description span visible:\n%s", got)
	}
}

func TestViewFlattensMultiLineDescriptions(t *testing.T) {
	// A Taskfile `desc: |` block and a just `[doc("a\nb")]` keep their
	// newlines in Task.Desc, but a row is one line: the description has to
	// stay in the description column and the source badge on the same row.
	tests := []struct {
		name string
		desc string
	}{
		{"lf", "Build the app\nSecond line"},
		{"crlf", "Build the app\r\nSecond line"},
		{"block scalar with blank line", "Build the app\n\nSecond line\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []core.Task{{
				Source: "Taskfile.yml",
				Name:   "build",
				Desc:   tt.desc,
				Args:   []string{"task", "build"},
			}}
			m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10})

			row := m.taskRow(tasks[0], false, 80)
			if strings.ContainsAny(row, "\r\n") {
				t.Errorf("taskRow() = %q, want no embedded newline", row)
			}
			if !strings.Contains(row, "Build the app Second line") {
				t.Errorf("taskRow() = %q, want both description lines joined by a space", row)
			}
			if !strings.HasSuffix(row, m.styles.dim.Render("Taskfile.yml")) {
				t.Errorf("taskRow() = %q, want the source badge on the same row", row)
			}
			if w := lipgloss.Width(row); w > 80 {
				t.Errorf("taskRow() is %d columns wide, want at most 80: %q", w, row)
			}
			if !strings.Contains(m.View().Content, "Build the app Second line") {
				t.Errorf("View() does not show the flattened description:\n%s", m.View().Content)
			}
		})
	}
}

func TestViewHighlightsMatchOnSecondDescriptionLine(t *testing.T) {
	// The match mask is computed on the flattened description, so a token
	// found on the second line of a `desc: |` block still highlights the
	// runes the row actually shows.
	tasks := []core.Task{{
		Source: "Taskfile.yml",
		Name:   "build",
		Desc:   "Build the app\nSecond line",
		Args:   []string{"task", "build"},
	}}
	m := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10})
	m = press(m, "second")

	got := m.View().Content
	if !strings.Contains(got, m.styles.selectedMatch.Render("Second")) {
		t.Errorf("View() did not highlight the match on the flattened description:\n%s", got)
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

func TestViewInlineFullscreenToggle(t *testing.T) {
	m := sized(Options{Inline: true}) // 30 rows, starting at row 5: 25 inline rows

	if got := strings.Count(m.View().Content, "\n") + 1; got != 25 {
		t.Fatalf("inline view height = %d, want 25", got)
	}

	next, cmd := m.Update(ctrlFKey)
	m = next.(model)
	if cmd != nil {
		t.Errorf("ctrl+f returned %T, want no command", cmd)
	}
	v := m.View()
	if !v.AltScreen {
		t.Error("ctrl+f in inline mode should switch to the alternate screen")
	}
	if got := strings.Count(v.Content, "\n") + 1; got != 30 {
		t.Errorf("fullscreen view height = %d, want 30 (the whole terminal)", got)
	}
	if !strings.Contains(v.Content, "ctrl+f inline") {
		t.Errorf("fullscreen help line does not offer the way back:\n%s", v.Content)
	}

	// ctrl+f again restores the inline area at the same anchor row.
	next, _ = m.Update(ctrlFKey)
	m = next.(model)
	v = m.View()
	if v.AltScreen {
		t.Error("second ctrl+f should leave the alternate screen")
	}
	if got := strings.Count(v.Content, "\n") + 1; got != 25 {
		t.Errorf("inline view height after toggling back = %d, want 25", got)
	}
	if m.startRow != 5 {
		t.Errorf("startRow after toggling back = %d, want 5", m.startRow)
	}
}

// The alternate screen can report a different terminal size (a scrollbar
// comes and goes, for example). Toggling back must lay the inline frame out at
// the main screen's size: a column too many wraps every row and leaves the
// previous frame on screen.
func TestViewInlineUsesMainScreenSizeAfterFullscreen(t *testing.T) {
	m := sized(Options{Inline: true}) // 80x30, with the inline area starting at row 5

	next, _ := m.Update(ctrlFKey)
	m = next.(model)
	m = update(m, tea.WindowSizeMsg{Width: 81, Height: 30}) // the alternate screen's width

	next, _ = m.Update(ctrlFKey)
	m = next.(model)
	if got := m.inlineWidth(); got != 80 {
		t.Errorf("inline width after toggling back = %d, want 80 (the main screen)", got)
	}
	v := m.View()
	if got := strings.Count(v.Content, "\n") + 1; got != 25 {
		t.Errorf("inline view height after toggling back = %d, want 25", got)
	}
	for _, line := range strings.Split(v.Content, "\n") {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("inline view line is %d columns wide, want at most 80: %q", w, line)
		}
	}

	// A resize of the main screen after toggling back is adopted.
	m = update(m, tea.WindowSizeMsg{Width: 90, Height: 32})
	if got := m.inlineWidth(); got != 90 {
		t.Errorf("inline width after a main-screen resize = %d, want 90", got)
	}
	if got := strings.Count(m.View().Content, "\n") + 1; got != 27 {
		t.Errorf("inline view height after a main-screen resize = %d, want 27", got)
	}
}

func TestCtrlFIgnoredFullscreen(t *testing.T) {
	next, cmd := sized(Options{}).Update(ctrlFKey)
	if cmd != nil {
		t.Errorf("ctrl+f in fullscreen mode returned %T, want no command", cmd)
	}
	m := next.(model)
	if m.fullscreen {
		t.Error("ctrl+f switched a fullscreen run into the inline fullscreen state")
	}
	if !m.View().AltScreen {
		t.Error("ctrl+f left the fullscreen view without AltScreen")
	}
}

func TestViewInlineHelpMentionsFullscreen(t *testing.T) {
	got := sized(Options{Inline: true}).View().Content
	if !strings.Contains(got, "ctrl+f fullscreen") {
		t.Errorf("inline help line does not mention ctrl+f at 80 columns:\n%s", got)
	}
	if strings.Contains(got, "ctrl+t theme") {
		t.Errorf("inline help line at 80 columns kept the ctrl+t hint, want it dropped:\n%s", got)
	}

	m := update(newModel(Options{Inline: true}, sampleTasks(), nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})
	got = m.View().Content
	if !strings.Contains(got, "ctrl+f fullscreen") || !strings.Contains(got, "ctrl+t theme") {
		t.Errorf("inline help line at 120 columns dropped a hint:\n%s", got)
	}

	if got := sized(Options{}).View().Content; strings.Contains(got, "ctrl+f") {
		t.Errorf("fullscreen help line mentions ctrl+f:\n%s", got)
	}
}

func TestViewFullscreenFillsHeightAndScrolls(t *testing.T) {
	tasks := make([]core.Task, 30)
	for i := range tasks {
		tasks[i] = core.Task{Source: "Makefile", Name: fmt.Sprintf("task-%02d", i), Args: []string{"make", fmt.Sprintf("task-%02d", i)}}
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
	m := update(newModel(Options{}, sampleTasks(), []error{errors.New("Makefile.toml: boom")}),
		tea.WindowSizeMsg{Width: 80, Height: 30})
	if got := m.View().Content; !strings.Contains(got, "Makefile.toml: boom") {
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

func TestViewNoTasksShowsPersistentEmptyState(t *testing.T) {
	m := update(newModel(Options{Inline: true}, nil, nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})

	got := m.View().Content
	for _, want := range []string{
		"no tasks found here",
		"package.json",
		"Makefile.toml",
		"Taskfile.yml",
		"justfile",
		".cargo/config.toml",
		"devbox.json",
		"composer.json",
		"deno.json",
		"mise.toml",
		".what-to-run.toml",
		"enter/esc close",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("View() does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "type to filter") {
		t.Errorf("View() invites filtering with no tasks:\n%s", got)
	}
}

func TestNoTasksEmptyStateOnlyDismisses(t *testing.T) {
	m := update(newModel(Options{Inline: true}, nil, nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})

	m = press(m, "abc")
	if m.filter != "" {
		t.Errorf("filter in the empty state = %q, want empty", m.filter)
	}

	for _, key := range []tea.KeyPressMsg{enterKey, escKey} {
		next, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%s in the empty state returned no command, want tea.Quit", key.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s in the empty state returned %T, want tea.QuitMsg", key.String(), cmd())
		}
		if chosen := next.(model).chosen; chosen != nil {
			t.Errorf("%s in the empty state chose %+v, want nothing", key.String(), chosen)
		}
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
	if got, want := m.chosen.Command().Args, []string{"npm", "run", "build"}; !slices.Equal(got, want) {
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
	if got, want := m.chosen.CommandLine(), "npm run build"; got != want {
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
	task := core.Task{Source: "rc", Name: "greet", Args: []string{"echo", "it's a test"}}
	tests := []struct {
		name   string
		insert bool
		shell  core.ShellDialect
		want   string
	}{
		{"run", false, core.ShellPOSIX, "run\necho 'it'\\''s a test'\n"},
		{"insert", true, core.ShellPOSIX, "insert\necho 'it'\\''s a test'\n"},
		{"powershell", false, core.ShellPowerShell, "run\necho 'it''s a test'\n"},
		{"cmd", false, core.ShellCmd, "run\necho \"it's a test\"\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reportChoice(&buf, task, tt.insert, tt.shell)
			if got := buf.String(); got != tt.want {
				t.Errorf("reportChoice(..., %s) wrote %q, want %q", tt.shell, got, tt.want)
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

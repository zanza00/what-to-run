package tui

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

// sized returns a model after the terminal has answered the size query and, in
// inline mode, the cursor position query, as in a running session.
func sized(opts Options) model {
	m := update(newModel(opts, sampleTasks(), nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	if opts.Inline {
		m = update(m, tea.CursorPositionMsg{Y: 5})
	}
	return m
}

func TestNewModelShellDialect(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	tests := []struct {
		name        string
		shell       string
		want        core.ShellDialect
		wantWarning bool
	}{
		{name: "empty means the platform default", shell: "", want: core.DefaultShellDialect()},
		{name: "cmd", shell: "cmd", want: core.ShellCmd},
		{
			name:        "unknown falls back to the platform default",
			shell:       "fish",
			want:        core.DefaultShellDialect(),
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel(Options{Shell: tt.shell}, nil, nil)
			if m.shell != tt.want {
				t.Errorf("newModel(Shell: %q).shell = %v, want %v", tt.shell, m.shell, tt.want)
			}
			if !tt.wantWarning {
				if len(m.warnings) != 0 {
					t.Errorf("warnings = %v, want none", m.warnings)
				}
				return
			}
			if len(m.warnings) != 1 {
				t.Fatalf("warnings = %v, want exactly one", m.warnings)
			}
			if got := m.warnings[0].Error(); !strings.Contains(got, "unknown shell") {
				t.Errorf("warning = %q, want it to mention the unknown shell", got)
			}
		})
	}
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
	// An rc task without a custom description defaults to its command, so the
	// row must not print it twice.
	tasks := []core.Task{{Source: "rc", Name: "status", Desc: "git status", Args: []string{"sh", "-c", "git status"}}}
	got := update(newModel(Options{}, tasks, nil), tea.WindowSizeMsg{Width: 80, Height: 10}).View().Content
	if n := strings.Count(got, "git status"); n != 1 {
		t.Errorf("View() shows %q %d times, want once:\n%s", "git status", n, got)
	}
}

func TestViewShowsScriptBodyMatchingTaskName(t *testing.T) {
	// A package.json script such as "knip": "knip" has the same text as the
	// script name in "pnpm run knip", but the picker must still show its script
	// body in the description column.
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
	// The name is not a column any more, so filtering has to find what the rows
	// show, e.g. the runner.
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

	// "dev": the exact task name comes first, then the prefix name
	// ("dev:edge"), then the description match ("preview"). The "build" task
	// ("src/edge && vite") is rejected as cross-word mid-word noise.
	m := press(base, "dev")
	vis := m.visible()
	var names []string
	for _, task := range vis {
		names = append(names, task.Name)
	}
	if want := []string{"dev", "dev:edge", "preview"}; !slices.Equal(names, want) {
		t.Errorf("ranked tasks for %q = %v, want %v", "dev", names, want)
	}

	// "tsc": a contiguous "tsc" in the description of build beats a
	// non-contiguous acronym match in a task name ("type-strict-check"), and
	// the scattered cross-word noise ("test:watch", "codes", "icons", "check")
	// is rejected.
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
	// The selected row (build) highlights "vite" with selectedMatch, and the
	// unselected row (dev) highlights "vite" with match.
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
	// A Taskfile `desc: |` block and a just `[doc("a\nb")]` keep their newlines
	// in Task.Desc, but a row is one line. The description has to stay in the
	// description column, and the source badge has to stay on the same row.
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
	// The match mask is computed on the flattened description, so a token that
	// is found on the second line of a `desc: |` block still highlights the
	// runes that the row shows.
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
	// At 44 columns the description barely fits, and the command must stay
	// readable, because it is what runs.
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
	got := sized(Options{Prog: "task-picker"}).View().Content
	if !strings.Contains(got, "task-picker") {
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

// The alternate screen can report a different terminal size (a scrollbar comes
// and goes, for example). Toggling back must lay the inline frame out at the
// size of the main screen: one column too many wraps every row and leaves the
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

	// The picker adopts a resize of the main screen after it toggles back.
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

// taskNames returns the names of tasks in order, so a test can compare a
// ranking without depending on the rest of the task.
func taskNames(tasks []core.Task) []string {
	names := make([]string, len(tasks))
	for i, task := range tasks {
		names[i] = task.Name
	}
	return names
}

// TestModelWithoutSnapshotRenders pins that a model built without newModel
// still renders and measures its commands. The snapshot is an optimisation,
// not a requirement.
func TestModelWithoutSnapshotRenders(t *testing.T) {
	m := model{tasks: sampleTasks()}
	m.width, m.height, m.ready = 80, 10, true

	if got := m.View().Content; !strings.Contains(got, "npm run build") {
		t.Errorf("View() without a snapshot does not render the tasks:\n%s", got)
	}
	if got, want := m.commandWidth(), len("cargo make clean"); got != want {
		t.Errorf("commandWidth() without a snapshot = %d, want %d", got, want)
	}

	// A direct edit without the helper falls back to a full computation.
	m.filter = "clean"
	if got := taskNames(m.visible()); !slices.Equal(got, []string{"clean"}) {
		t.Errorf("visible tasks after a direct filter edit = %v, want [clean]", got)
	}
}

// TestTaskSnapshotFollowsRefresh pins that a reload of the task sources
// replaces the cached ranking and keeps the cursor inside the shorter list.
func TestTaskSnapshotFollowsRefresh(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	writeFile(t, dir, "Makefile", "build:\n\t@echo build\n")

	m := press(sized(Options{}), "build")
	m.cursor, m.offset = 1, 1

	m.refreshConfigWarnings()

	if !m.snapshot.valid {
		t.Fatal("snapshot after the task refresh is invalid")
	}
	want := rankTasks(m.tasks, strings.Fields(strings.ToLower(m.filter)), m.shell)
	got := taskNames(m.visible())
	if !slices.Equal(got, taskNames(want)) {
		t.Errorf("visible tasks after the refresh = %v, want %v", got, taskNames(want))
	}
	if !slices.Equal(got, []string{"build"}) {
		t.Errorf("visible tasks after the refresh = %v, want the new Makefile target", got)
	}
	if m.cursor != 0 || m.offset != 0 {
		t.Errorf("cursor and offset after the refresh = %d and %d, want 0 and 0", m.cursor, m.offset)
	}
	if got, want := m.snapshot.commandWidth, commandWidthFor(want, m.shell); got != want {
		t.Errorf("command width after the refresh = %d, want %d", got, want)
	}
}

// TestFilterSnapshotFollowsFilterChanges pins that every filter edit refreshes
// the cached ranking and command width. A stale cache would show old rows after
// a key press.
func TestFilterSnapshotFollowsFilterChanges(t *testing.T) {
	m := sized(Options{})

	steps := []struct {
		name string
		msg  tea.Msg
	}{
		{"type m", tea.KeyPressMsg{Code: 'm', Text: "m"}},
		{"type a", tea.KeyPressMsg{Code: 'a', Text: "a"}},
		{"type k", tea.KeyPressMsg{Code: 'k', Text: "k"}},
		{"type e", tea.KeyPressMsg{Code: 'e', Text: "e"}},
		{"type t", tea.KeyPressMsg{Code: 't', Text: "t"}},
		{"backspace", tea.KeyPressMsg{Code: tea.KeyBackspace}},
		{"ctrl+u", ctrlUKey},
		{"type z", tea.KeyPressMsg{Code: 'z', Text: "z"}},
		{"type z", tea.KeyPressMsg{Code: 'z', Text: "z"}},
		{"type z", tea.KeyPressMsg{Code: 'z', Text: "z"}},
		{"esc", escKey},
	}

	for _, step := range steps {
		m = update(m, step.msg)

		if !m.snapshot.valid {
			t.Fatalf("after %s the snapshot is invalid, want the new results", step.name)
		}
		want := rankTasks(m.tasks, strings.Fields(strings.ToLower(m.filter)), m.shell)
		if got := taskNames(m.snapshot.tasks); !slices.Equal(got, taskNames(want)) {
			t.Fatalf("after %s ranked tasks = %v, want %v", step.name, got, taskNames(want))
		}
		if got, want := m.snapshot.commandWidth, commandWidthFor(want, m.shell); got != want {
			t.Fatalf("after %s command width = %d, want %d", step.name, got, want)
		}
	}
}

func TestViewNoMatchingTask(t *testing.T) {
	m := press(sized(Options{}), "zzz")
	if got := m.View().Content; !strings.Contains(got, "no task matches") {
		t.Errorf("View() does not show the empty state:\n%s", got)
	}
}

// emptyStateFileLines returns the lines of the "looked for task files"
// sentence of the empty screen, with the ANSI escapes and the indentation
// removed.
func emptyStateFileLines(t *testing.T, view string) []string {
	t.Helper()

	lines := strings.Split(ansi.Strip(view), "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, "looked for task files:") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("View() has no \"looked for task files\" sentence:\n%s", ansi.Strip(view))
	}

	var sentence []string
	for _, line := range lines[start:] {
		if strings.Contains(line, "add [[tasks]]") {
			return sentence
		}
		sentence = append(sentence, strings.TrimSpace(line))
	}
	t.Fatalf("View() file list is not followed by the rc hint:\n%s", ansi.Strip(view))
	return nil
}

// emptyStateFiles returns the file names in the "looked for task files"
// sentence of the empty screen, with the wrapping removed.
func emptyStateFiles(t *testing.T, view string) []string {
	t.Helper()

	sentence := strings.Join(emptyStateFileLines(t, view), " ")
	const prefix = "looked for task files: "
	if !strings.HasPrefix(sentence, prefix) {
		t.Fatalf("empty-screen sentence = %q, want prefix %q", sentence, prefix)
	}
	sentence = strings.TrimPrefix(sentence, prefix)

	var files []string
	for _, item := range strings.Split(sentence, ",") {
		for _, name := range strings.Split(item, " and ") {
			if name = strings.TrimSpace(name); name != "" {
				files = append(files, name)
			}
		}
	}
	return files
}

// assertEmptyStateFiles compares the files that the empty screen names with the
// display files of the registry, as a set. A substring check would let a
// missing Makefile pass because of Makefile.toml.
func assertEmptyStateFiles(t *testing.T, view string) {
	t.Helper()

	want := make(map[string]bool, len(sourceInfos))
	for _, src := range sourceInfos {
		if file := src.DisplayFile(); file != "" {
			want[file] = true
		}
	}

	got := make(map[string]bool, len(want))
	for _, file := range emptyStateFiles(t, view) {
		if got[file] {
			t.Errorf("empty screen lists %q more than once:\n%s", file, ansi.Strip(view))
		}
		got[file] = true
	}
	for file := range want {
		if !got[file] {
			t.Errorf("empty screen does not list display file %q:\n%s", file, ansi.Strip(view))
		}
	}
	for file := range got {
		if !want[file] {
			t.Errorf("empty screen lists %q, which is not a registry display file:\n%s", file, ansi.Strip(view))
		}
	}
}

// assertFileListNotTruncated fails when a line of the file list ends in the
// ellipsis that render adds to a line that is wider than the picker.
func assertFileListNotTruncated(t *testing.T, view string) {
	t.Helper()

	for _, line := range emptyStateFileLines(t, view) {
		if strings.Contains(line, "…") {
			t.Errorf("file list line is truncated: %q", line)
		}
	}
}

func TestViewNoTasksShowsPersistentEmptyState(t *testing.T) {
	m := update(newModel(Options{Inline: true}, nil, nil), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})

	got := m.View().Content
	for _, want := range []string{
		"no tasks found here",
		".what-to-run.toml",
		"enter/esc close",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("View() does not contain %q:\n%s", want, got)
		}
	}
	assertEmptyStateFiles(t, got)
	assertFileListNotTruncated(t, got)
	if strings.Contains(got, "type to filter") {
		t.Errorf("View() invites filtering with no tasks:\n%s", got)
	}
}

func TestViewNoTasksNarrowWidth(t *testing.T) {
	m := update(newModel(Options{Inline: true}, nil, nil), tea.WindowSizeMsg{Width: 40, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5})

	got := m.View().Content
	if !strings.Contains(got, "no tasks found here") {
		t.Errorf("View() at 40 columns does not show the empty state:\n%s", got)
	}
	assertEmptyStateFiles(t, got)
	assertFileListNotTruncated(t, got)
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
	if got, want := m.chosen.CommandLineFor(m.shell), "npm run build"; got != want {
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

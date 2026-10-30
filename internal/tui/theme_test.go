package tui

import (
	"bytes"
	"errors"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var ctrlTKey = tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}

// titleLine returns the first rendered line of the view, without the ANSI
// sequences.
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
		// A colored theme needs both tint colors: the background of the
		// selected row, and a foreground that stays readable on it.
		if th.accent != nil && (th.selBg == nil || th.selFg == nil) {
			t.Errorf("theme %q has colors but misses selBg or selFg", th.name)
		}
		seen[th.name] = true
	}
}

// TestThemeContrast keeps the selected row readable. The light text of a bar
// theme must stand out from its tint (WCAG AA, 4.5:1). The nord theme once
// failed this test, with the dim color on a nearly identical tint.
func TestThemeContrast(t *testing.T) {
	for _, th := range themes {
		if th.selBg == nil || th.selFg == nil {
			continue // The colorless theme has no tint to check.
		}
		if got := contrastRatio(th.selFg, th.selBg); got < 4.5 {
			t.Errorf("theme %q: selFg on selBg contrast = %.1f:1, want at least 4.5:1", th.name, got)
		}
	}
}

func TestSystemThemeAdaptsToTerminalColors(t *testing.T) {
	tests := []struct {
		name      string
		bg        color.Color
		fg        color.Color
		wantSelBg string // SGR background sequence on the selected row
		wantSelFg string // SGR foreground sequence on the selected row
		wantDim   string // SGR foreground sequence on unselected secondary text
	}{
		{
			name:      "default fallback before terminal reply",
			bg:        nil,
			fg:        nil,
			wantSelBg: "48;2;25;25;25",    // #191919 (grays[3] from #000000)
			wantSelFg: "38;2;192;192;192", // #C0C0C0 (ANSI 7 fallback)
			wantDim:   "38;2;180;180;180", // #B4B4B4 (muted gray on #000000)
		},
		{
			name:      "dark terminal background and foreground",
			bg:        lipgloss.Color("#1A1B26"),
			fg:        lipgloss.Color("#C0CAF5"),
			wantSelBg: "48;2;47;48;68",    // #2F3044 (grays[3] from #1A1B26)
			wantSelFg: "38;2;192;202;245", // #C0CAF5
			wantDim:   "38;2;168;168;168", // #A8A8A8 (muted gray on #1A1B26)
		},
		{
			name:      "light terminal background and foreground",
			bg:        lipgloss.Color("#FBF1C7"),
			fg:        lipgloss.Color("#3C3836"),
			wantSelBg: "48;2;225;216;179", // #E1D8B3 (grays[3] from #FBF1C7)
			wantSelFg: "38;2;60;56;54",    // #3C3836
			wantDim:   "38;2;96;96;96",    // #606060 (muted gray on #FBF1C7)
		},
		{
			name:      "pure white terminal background without foreground",
			bg:        lipgloss.Color("#FFFFFF"),
			fg:        nil,
			wantSelBg: "48;2;229;229;229", // #E5E5E5 (grays[3] from #FFFFFF)
			wantSelFg: "38;2;0;0;0",       // #000000 (ANSI 0 fallback)
			wantDim:   "38;2;75;75;75",    // #4B4B4B (muted gray on #FFFFFF)
		},
		{
			name:      "mid-tone background enforces WCAG contrast",
			bg:        lipgloss.Color("#7F7F7F"),
			fg:        lipgloss.Color("#909090"),
			wantSelBg: "48;2;139;139;139",
			wantSelFg: "38;2;0;0;0", // low-contrast #909090 falls back to #000000 (6.2:1 vs 3.4:1)
			wantDim:   "38;2;198;198;198",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := systemTheme(tt.bg, tt.fg)
			if got := contrastRatio(th.selFg, th.selBg); got < 4.5 {
				t.Errorf("systemTheme(%v, %v) contrast = %.1f:1, want at least 4.5:1", tt.bg, tt.fg, got)
			}

			m := sized(Options{Theme: "system"})
			if tt.bg != nil {
				m = update(m, tea.BackgroundColorMsg{Color: tt.bg})
			}
			if tt.fg != nil {
				m = update(m, tea.ForegroundColorMsg{Color: tt.fg})
			}

			view := m.View().Content
			selected := selectedRow(t, view)
			if !strings.Contains(selected, tt.wantSelBg) {
				t.Errorf("selected row does not paint selBg %q:\n%q", tt.wantSelBg, selected)
			}
			if !strings.Contains(selected, tt.wantSelFg) {
				t.Errorf("selected row does not paint selFg %q:\n%q", tt.wantSelFg, selected)
			}
			if !strings.Contains(view, tt.wantDim) {
				t.Errorf("view does not use dim %q:\n%q", tt.wantDim, view)
			}
		})
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

func TestCtrlTOpensTheThemePicker(t *testing.T) {
	m := update(sized(Options{}), ctrlTKey)
	if !m.themeModal {
		t.Fatal("ctrl+t did not open the theme picker")
	}
	if m.themeIdx != 0 || m.themeBefore != 0 {
		t.Errorf("opening the picker changed the theme: idx = %d, before = %d", m.themeIdx, m.themeBefore)
	}
	if m.filter != "" {
		t.Errorf("filter after ctrl+t = %q, want empty (it opens the picker, it does not type)", m.filter)
	}
}

func TestThemePickerPreviewsLive(t *testing.T) {
	m := update(sized(Options{}), ctrlTKey)

	m = update(m, downKey)
	if got := themes[m.themeIdx].name; got != "nord" {
		t.Errorf("theme after one down = %q, want nord", got)
	}
	// The preview is live, so the task list behind the box wears the tint of
	// the theme (nord selBg #434C5E).
	got := m.View().Content
	var row string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "▌") && strings.Contains(line, "npm run build") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("no selected task row in the previewed view:\n%s", got)
	}
	if !strings.Contains(row, "48;2;67;76;94") {
		t.Errorf("the task list does not wear the previewed theme's tint:\n%q", row)
	}

	m = update(m, upKey)
	if got := themes[m.themeIdx].name; got != defaultTheme {
		t.Errorf("theme after one up = %q, want %q", got, defaultTheme)
	}

	// The cursor stops at the ends of the list.
	for range len(themes) {
		m = update(m, upKey)
	}
	if m.themeIdx != 0 {
		t.Errorf("theme after scrolling past the top = %d, want 0", m.themeIdx)
	}
	m = update(m, endKey)
	if m.themeIdx != len(themes)-1 {
		t.Errorf("theme after end = %d, want %d", m.themeIdx, len(themes)-1)
	}
	m = update(m, homeKey)
	if m.themeIdx != 0 {
		t.Errorf("theme after home = %d, want 0", m.themeIdx)
	}
}

func TestThemePickerConfirmAndCancel(t *testing.T) {
	env, _ := testEnv(t)

	// The enter key keeps the previewed theme and closes the box. The save
	// arrives as a toast, not on the help line.
	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.themeModal || m.themeIdx != 1 {
		t.Errorf("after enter: modal = %v, theme = %d; want closed on 1", m.themeModal, m.themeIdx)
	}
	if m.notice == "" {
		t.Error("enter did not confirm the save with a notice")
	}
	if m.status != "" {
		t.Errorf("status after saving = %q, want it untouched", ansi.Strip(m.status))
	}
	if got := m.View().Content; strings.Contains(got, "╰") {
		t.Errorf("the box is still drawn after enter:\n%s", got)
	}

	// A second ctrl+t closes the picker and keeps the preview.
	m = update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	next, cmd = m.Update(ctrlTKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.themeModal || m.themeIdx != 1 {
		t.Errorf("after ctrl+t: modal = %v, theme = %d; want closed on 1", m.themeModal, m.themeIdx)
	}

	// esc cancels. The theme from before the picker comes back, and the press
	// neither quits nor clears the filter.
	m = press(sized(Options{Env: env}), "npm")
	m = update(m, ctrlTKey)
	m = update(m, downKey)
	next, cmd = m.Update(escKey)
	m = next.(model)
	if m.themeModal || m.themeIdx != 0 {
		t.Errorf("after esc: modal = %v, theme = %d; want closed on the original 0", m.themeModal, m.themeIdx)
	}
	if m.filter != "npm" {
		t.Errorf("filter after esc in the picker = %q, want %q", m.filter, "npm")
	}
	if cmd != nil {
		t.Error("esc in the theme picker returned a command, want none (it must not quit)")
	}
}

func TestThemePickerIgnoresTyping(t *testing.T) {
	m := press(update(sized(Options{}), ctrlTKey), "npm")
	if m.filter != "" {
		t.Errorf("filter while the picker is open = %q, want empty", m.filter)
	}
	if !m.themeModal {
		t.Error("typing closed the theme picker")
	}
	if m.themeIdx != 0 {
		t.Errorf("theme after typing = %d, want 0", m.themeIdx)
	}
}

func TestViewTitleIgnoresTheTheme(t *testing.T) {
	if got := titleLine(sized(Options{}).View().Content); got != "wtr" {
		t.Errorf("default title = %q, want %q", got, "wtr")
	}
	// The title stays the program name whatever the theme is, because the
	// picker shows the look itself.
	if got := titleLine(sized(Options{Theme: "nord"}).View().Content); got != "wtr" {
		t.Errorf("title with the nord theme = %q, want %q", got, "wtr")
	}
}

// selectedRow returns the rendered line of the bar-selected row.
func selectedRow(t *testing.T, view string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "▌") {
			return line
		}
	}
	t.Fatalf("bar-selected view does not use the bar marker:\n%s", view)
	return ""
}

// boxHeight returns the number of rows that the theme picker box occupies,
// from its top border to its bottom border.
func boxHeight(t *testing.T, view string) int {
	t.Helper()
	top, bottom := -1, -1
	for i, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "╭") {
			top = i
		}
		if strings.Contains(line, "╰") {
			bottom = i
		}
	}
	if top < 0 || bottom < 0 {
		t.Fatalf("no box in the view:\n%s", view)
	}
	return bottom - top + 1
}

func TestBarSelectionTintsTheWholeRow(t *testing.T) {
	selected := selectedRow(t, sized(Options{Theme: "tokyo"}).View().Content)

	// The tint arrives as a background color, and the text on it is the light
	// tone of the palette. It is never the dim color, which can be nearly
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

	// The default theme is a bar theme like the others: the same marker, and
	// its own violet tint and light tone.
	selected = selectedRow(t, sized(Options{}).View().Content)
	if !strings.Contains(selected, "48;2;70;60;110") { // #463C6E
		t.Errorf("default selected row does not paint its tint:\n%q", selected)
	}
	if !strings.Contains(selected, "38;2;237;233;254") { // #EDE9FE
		t.Errorf("default selected row text is not the theme's light tone:\n%q", selected)
	}
	if strings.Contains(selected, "38;5;241") { // classic dim
		t.Errorf("default selected row still uses the dim color for part of its text:\n%q", selected)
	}

	// The colorless theme keeps the same layout. The row carries no colors at
	// all, and reverse video acts as its tint.
	selected = selectedRow(t, sized(Options{Theme: "mono"}).View().Content)
	if !strings.Contains(selected, "[7m") && !strings.Contains(selected, ";7m") {
		t.Errorf("mono selected row does not use reverse video:\n%q", selected)
	}
	if strings.Contains(selected, "38;") || strings.Contains(selected, "48;") {
		t.Errorf("mono selected row uses colors:\n%q", selected)
	}
}

func TestThemePickerRenders(t *testing.T) {
	got := update(sized(Options{}), ctrlTKey).View().Content

	for _, want := range []string{"╭", "╰", "themes", "classic", "nord", "tokyo", themeModalHints} {
		if !strings.Contains(got, want) {
			t.Errorf("theme picker view does not contain %q:\n%s", want, got)
		}
	}
	if h := boxHeight(t, got); h != themeModalListRows+themeModalChrome {
		t.Errorf("box height = %d rows, want %d", h, themeModalListRows+themeModalChrome)
	}

	// The rows hold only the names, so the box shows no theme description.
	if strings.Contains(got, "the default violet look") {
		t.Errorf("theme picker shows a theme description:\n%s", got)
	}

	// The previewed row wears the palette of the active theme.
	var row string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "▌") && strings.Contains(line, "classic") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("no selected theme row in the picker:\n%s", got)
	}
	if !strings.Contains(row, "48;2;70;60;110") { // classic tint #463C6E
		t.Errorf("selected theme row does not paint the theme tint:\n%q", row)
	}

	// The other rows use their own palette, so the name of nord carries its
	// accent.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "nord") && !strings.Contains(line, "38;2;136;192;208") { // #88C0D0
			t.Errorf("nord row does not use its own accent:\n%q", line)
		}
	}
}

func TestThemePickerRendersEveryTheme(t *testing.T) {
	for _, th := range themes {
		t.Run(th.name, func(t *testing.T) {
			got := update(sized(Options{Theme: th.name}), ctrlTKey).View().Content
			if !strings.Contains(got, "themes") || !strings.Contains(got, "╰") {
				t.Errorf("theme picker with theme %q does not render:\n%s", th.name, got)
			}
		})
	}
}

func TestThemePickerWindowsTheList(t *testing.T) {
	// The box shows five theme rows, and the user reaches the rest by
	// scrolling.
	m := update(sized(Options{}), ctrlTKey)
	got := m.View().Content
	if !strings.Contains(got, "themes 1/14 ↓") {
		t.Errorf("the title does not mark the hidden rows below:\n%s", got)
	}
	if strings.Contains(got, "catppuccin") || strings.Contains(got, " mono ") {
		t.Errorf("a theme outside the window is rendered:\n%s", got)
	}

	m = update(m, endKey)
	if want := len(themes) - themeModalListRows; m.themeOffset != want {
		t.Errorf("offset after scrolling to the end = %d, want %d", m.themeOffset, want)
	}
	got = m.View().Content
	if !strings.Contains(got, " mono ") {
		t.Errorf("the last theme is not revealed after scrolling:\n%s", got)
	}
	if strings.Contains(got, " classic ") {
		t.Errorf("the first theme is still rendered after scrolling:\n%s", got)
	}
	if !strings.Contains(got, "↑ themes 14/14") {
		t.Errorf("the title does not mark the hidden rows above:\n%s", got)
	}
}

func TestThemePickerFitsInlineArea(t *testing.T) {
	// In the inline area of 15 rows the box is smaller than the area, and the
	// view keeps its 15 rows.
	m := newModel(Options{Inline: true}, sampleTasks(), nil)
	m = update(m, tea.WindowSizeMsg{Width: 80, Height: 20})
	m = update(m, tea.CursorPositionMsg{Y: 12})
	m = update(m, ctrlTKey)

	got := m.View().Content
	if n := strings.Count(got, "\n"); n != 14 {
		t.Errorf("inline picker has %d lines, want 15", n+1)
	}
	if h := boxHeight(t, got); h != themeModalListRows+themeModalChrome {
		t.Errorf("box height = %d rows, want %d", h, themeModalListRows+themeModalChrome)
	}
}

// TestThemePickerSavesTheChoice checks that keeping a theme from the picker
// writes it to the user rc file. A toast confirms where it went, and the help
// line stays untouched.
func TestThemePickerSavesTheChoice(t *testing.T) {
	env, files := testEnv(t)

	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey) // nord
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	if !strings.Contains(m.notice, "theme nord saved to") {
		t.Errorf("notice after saving = %q, want the save message", m.notice)
	}
	if got := m.View().Content; !strings.Contains(got, "saved to") {
		t.Errorf("the view does not show the toast:\n%s", got)
	}
	if m.status != "" {
		t.Errorf("status after saving = %q, want it untouched", ansi.Strip(m.status))
	}
	data, ok := files[env.UserFile]
	if !ok {
		t.Fatal("saved theme not written to the user rc file")
	}
	if !strings.Contains(data, `theme = "nord"`) {
		t.Errorf("user rc = %q, want the nord theme", data)
	}
}

// TestThemePickerSavesInTheUserFile pins that the picker saves the theme in the
// user rc file (like OpenCode's cli.json) and never in the project one. ctrl+t
// keeps the choice like enter does.
func TestThemePickerSavesInTheUserFile(t *testing.T) {
	env, files := testEnv(t)
	files[env.ProjectFile] = "# my tasks\n"

	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	next, cmd := m.Update(ctrlTKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	data, ok := files[env.UserFile]
	if !ok {
		t.Fatal("user rc not written")
	}
	if !strings.Contains(data, `theme = "nord"`) {
		t.Errorf("user rc = %q, want the nord theme", data)
	}
	if got := files[env.ProjectFile]; got != "# my tasks\n" {
		t.Errorf("project rc = %q, want it untouched", got)
	}
	if !strings.Contains(m.notice, "saved to") {
		t.Errorf("notice = %q, want the save message", m.notice)
	}
}

// TestNoticeExpires checks that the toast clears itself on its timer, and that
// a stale timer does not clear a newer notice.
func TestNoticeExpires(t *testing.T) {
	env, _ := testEnv(t)

	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.notice == "" {
		t.Fatal("no notice after saving")
	}
	first := m.noticeID

	// A newer notice replaces the first one and keeps its own timer.
	m = update(m, ctrlTKey)
	m = update(m, downKey)
	next, cmd = m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))
	if m.noticeID == first {
		t.Fatalf("notice id after a second save = %d, want it to grow", m.noticeID)
	}

	// The stale timer of the first notice must not clear the current one.
	m = update(m, noticeExpiredMsg{id: first})
	if m.notice == "" {
		t.Error("a stale timer cleared the current notice")
	}

	// The current timer clears it.
	m = update(m, noticeExpiredMsg{id: m.noticeID})
	if m.notice != "" {
		t.Errorf("notice after its timer = %q, want empty", m.notice)
	}
	if got := m.View().Content; strings.Contains(got, "saved to") {
		t.Errorf("the expired toast is still drawn:\n%s", got)
	}
}

// TestThemePickerSaveFailureStaysInTheStatusLine pins that a failed save is not
// a toast. It stays in the status line like any other problem.
func TestThemePickerSaveFailureStaysInTheStatusLine(t *testing.T) {
	env, files := testEnv(t)
	env.Store = errStore{Store: env.Store, err: errors.New("disk full")}

	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	next, cmd := m.Update(enterKey)
	m = next.(model)
	m = update(m, execCmd(t, cmd))

	if m.notice != "" {
		t.Errorf("notice after a failed save = %q, want none", m.notice)
	}
	if got := ansi.Strip(m.status); !strings.Contains(got, "cannot save theme nord: disk full") {
		t.Errorf("status after a failed save = %q, want the error", got)
	}
	if _, ok := files[env.UserFile]; ok {
		t.Error("a failed save wrote the user rc file")
	}
}

// TestThemePickerDoesNotSaveWithoutAChange pins that cancelling with esc writes
// nothing, and that keeping the theme that was already active writes nothing
// too.
func TestThemePickerDoesNotSaveWithoutAChange(t *testing.T) {
	env, files := testEnv(t)

	m := update(sized(Options{Env: env}), ctrlTKey)
	m = update(m, downKey)
	m = update(m, escKey)
	if _, ok := files[env.UserFile]; ok {
		t.Error("esc wrote the theme to the rc file")
	}

	m = update(sized(Options{Env: env}), ctrlTKey)
	next, cmd := m.Update(enterKey)
	m = next.(model)
	if cmd != nil {
		t.Errorf("confirming the active theme returned %T, want no command", cmd)
	}
	if _, ok := files[env.UserFile]; ok {
		t.Error("confirming the active theme wrote the rc file")
	}
}

func TestThemePickerFitsNarrowTerminals(t *testing.T) {
	m := newModel(Options{}, sampleTasks(), nil)
	m = update(m, tea.WindowSizeMsg{Width: 40, Height: 24})
	m = update(m, ctrlTKey)

	got := m.View().Content
	if !strings.Contains(got, "themes") {
		t.Fatalf("no theme picker in the view:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := ansi.StringWidth(line); w > 40 {
			t.Errorf("picker line is %d columns wide, want at most 40: %q", w, line)
		}
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

	// In the inline area at 80 columns the hint would push the line over the
	// edge. The view drops it, and the keys before it stay readable.
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
	env := diskEnv(t)
	writeFile(t, env.Dir, "Makefile", "build:\n\techo building\n")

	var buf bytes.Buffer
	if err := Preview(&buf, Options{Theme: "tokyo", Env: env}, 80, 24); err != nil {
		t.Fatalf("Preview: %v", err)
	}

	got := buf.String()
	if n := strings.Count(got, "\n"); n != 24 {
		t.Errorf("preview height = %d rows, want 24", n)
	}
	if !strings.Contains(got, "build") {
		t.Errorf("preview does not contain the task:\n%s", got)
	}
	// The frame wears the requested theme, so its accent colors the title
	// (tokyo #7AA2F7).
	if !strings.Contains(got, "38;2;122;162;247") {
		t.Errorf("preview does not use the tokyo accent:\n%s", got)
	}
}

// TestPreviewUsesTheConfiguredTheme pins that the theme of the user rc file is
// the starting look, and that --theme still wins for a single run.
func TestPreviewUsesTheConfiguredTheme(t *testing.T) {
	env := diskEnv(t)
	writeFile(t, env.Dir, "Makefile", "build:\n\techo building\n")
	writeFile(t, filepath.Dir(env.UserFile), "config.toml", "theme = \"nord\"\n")

	var buf bytes.Buffer
	if err := Preview(&buf, Options{Env: env}, 80, 24); err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !strings.Contains(buf.String(), "38;2;136;192;208") { // nord accent #88C0D0
		t.Errorf("preview does not use the rc theme:\n%s", buf.String())
	}

	buf.Reset()
	if err := Preview(&buf, Options{Theme: "tokyo", Env: env}, 80, 24); err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !strings.Contains(buf.String(), "38;2;122;162;247") { // tokyo accent #7AA2F7
		t.Errorf("preview does not honor --theme over the rc file:\n%s", buf.String())
	}
}

func TestPreviewWithoutTasks(t *testing.T) {
	env := diskEnv(t)

	if err := Preview(&bytes.Buffer{}, Options{Env: env}, 80, 24); err == nil {
		t.Error("Preview without tasks returned nil, want an error")
	}
}

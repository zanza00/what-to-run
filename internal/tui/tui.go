// Package tui implements the terminal user interface of wtr.
package tui

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// areaStyle draws the border around the inline area when DebugArea is set: a
// development aid to check that the UI covers exactly the rows between the
// prompt and the bottom of the screen.
var areaStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240"))

// chromeRows is the number of rows the picker reserves around the task list:
// title, filter line, blank separator and help line.
const chromeRows = 4

// minInlineHeight is the smallest area reserved for the inline UI. Television
// uses the same value: when fewer rows are free below the cursor, the
// terminal is left to scroll up instead of squeezing the UI.
const minInlineHeight = 15

// Options configures the TUI.
type Options struct {
	// Prog is the name the program was invoked as ("wtr",
	// "what-to-run"): it identifies the picker and its warnings.
	// Defaults to "wtr".
	Prog string

	// Inline renders the UI between the current cursor line and the bottom of
	// the screen instead of taking over the whole screen. It is meant for
	// shell widgets that launch the binary right below the prompt (the zsh
	// integration from 'wtr init zsh', see ZshInit) so that
	// the shell history above stays visible.
	Inline bool

	// DebugArea draws a border around the whole inline area so the region
	// managed by the app is visible. It is a development aid and is ignored
	// in fullscreen mode.
	DebugArea bool

	// Theme selects one of the named looks (see ListThemes); empty means
	// the default. An unknown name is a warning, not an error: the picker
	// opens with the default and reports it in the help line.
	Theme string
}

// prog is the name the program identifies with: the invoked name, or "wtr"
// when it was not set.
func (o Options) prog() string {
	if o.Prog == "" {
		return "wtr"
	}
	return o.Prog
}

// Run starts the picker and blocks until the user quits or launches a task.
// It returns an error only if the program fails to start or crashes.
//
// In fullscreen mode a task chosen with enter runs in place, in the picker:
// the terminal is handed over to the task (tea.ExecProcess) and taken back
// afterwards, with the outcome shown in the status line.
//
// In inline mode the picker only chooses. A shell widget (the zsh
// integration, see ZshInit) runs it with stdout captured, so Bubble Tea
// renders on the terminal itself and the result is reported on stdout as two
// lines: the mode ("run" after enter, "insert" after tab) and the command
// line. The widget then executes the command or places it in the prompt, so
// the command is visible and lands in the shell history. When stdout is the
// terminal (no widget), enter runs the task here at the prompt like a
// hand-typed command, propagating its exit code, and tab prints the command
// line, as there is no prompt to insert it into.
func Run(opts Options) error {
	if opts.Inline && !inlineInputAvailable() {
		fmt.Fprintf(os.Stderr, "%s: --inline requires an interactive terminal, running fullscreen\n", opts.prog())
		opts.Inline = false
	}

	// A widget captures stdout to receive the result; the UI then renders on
	// the terminal itself, like fzf does inside shell integrations.
	var progOpts []tea.ProgramOption
	out := io.Writer(os.Stdout)
	if opts.Inline && !term.IsTerminal(os.Stdout.Fd()) {
		in, ttyOut, err := tea.OpenTTY()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot open the terminal, running fullscreen\n", opts.prog())
			opts.Inline = false
		} else {
			defer func() { _ = in.Close(); _ = ttyOut.Close() }()
			progOpts = []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(ttyOut)}
			out = ttyOut
		}
	}

	tasks, cfg, warnings := discover()
	if len(tasks) == 0 {
		return noTasksError(warnings)
	}
	// The rc theme is the starting look; --theme wins for a single run.
	if opts.Theme == "" {
		opts.Theme = cfg.Theme
	}

	final, err := tea.NewProgram(newModel(opts, tasks, warnings), progOpts...).Run()
	if opts.Inline {
		if m, ok := final.(model); ok {
			restoreCursor(out, m)
		}
	}
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}

	m, ok := final.(model)
	if !ok || m.chosen == nil {
		return nil
	}

	if opts.Inline && out != io.Writer(os.Stdout) {
		// Widget mode: the shell runs the command (enter) or inserts it in
		// the prompt (tab).
		reportChoice(os.Stdout, *m.chosen, m.chosenInsert)
		return nil
	}

	if m.chosenInsert {
		// No widget to insert into: print the command line.
		fmt.Println(m.chosen.commandLine())
		return nil
	}

	cmd := m.chosen.command()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("task %s: %w", m.chosen.Name, err)
	}
	return nil
}

// reportChoice writes the picker result for the calling shell widget: the
// mode line ("run" or "insert") followed by the command line.
func reportChoice(w io.Writer, t Task, insert bool) {
	mode := "run"
	if insert {
		mode = "insert"
	}
	fmt.Fprintf(w, "%s\n%s\n", mode, t.commandLine())
}

// discover gathers the tasks of the current directory and the rc settings:
// the rc file first (its problems become warnings), then every source. The
// warnings are returned even when tasks were found, for the picker's status
// line.
func discover() ([]Task, config, []error) {
	cfg, warnings := loadConfig(defaultConfigFiles("."))
	tasks, sourceWarnings := discoverTasks(".", cfg)
	return tasks, cfg, append(warnings, sourceWarnings...)
}

// noTasksError is the error of a directory with nothing to run.
func noTasksError(warnings []error) error {
	if len(warnings) > 0 {
		return fmt.Errorf("no tasks found: %w", errors.Join(warnings...))
	}
	return errors.New("no tasks found: no package.json scripts, Makefile targets, cargo-make tasks or rc file tasks")
}

// Preview renders one frame of the picker with the real tasks of the current
// directory and writes it to w: a development and theming aid ('<prog>
// --preview'), handy to compare themes side by side or paste a frame
// somewhere. The frame is fixed at width×height cells, whatever the terminal
// size, and is drawn fullscreen (no inline anchoring, no runner).
func Preview(w io.Writer, opts Options, width, height int) error {
	tasks, cfg, warnings := discover()
	if len(tasks) == 0 {
		return noTasksError(warnings)
	}
	// The rc theme is the starting look; --theme wins for a single run.
	if opts.Theme == "" {
		opts.Theme = cfg.Theme
	}

	opts.Inline = false
	opts.DebugArea = false
	m := newModel(opts, tasks, warnings)
	if f, ok := w.(term.File); ok && term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(f.Fd()) {
		if bg, err := lipgloss.BackgroundColor(os.Stdin, f); err == nil {
			m.termBg = bg
			m.styles = newStyles(m.themeAt(m.themeIdx))
		}
	}
	m.width, m.height, m.ready = width, height, true
	fmt.Fprintln(w, m.View().Content)
	return nil
}

// inlineInputAvailable reports whether Bubble Tea will be able to read input
// (including the terminal's reply to the cursor position query) when running
// inline. Shell widgets (e.g. zle bindings) run commands with stdin not
// connected to the terminal, in which case Bubble Tea opens /dev/tty itself,
// so that counts too.
func inlineInputAvailable() bool {
	if term.IsTerminal(os.Stdin.Fd()) {
		return true
	}
	in, out, err := tea.OpenTTY()
	if err != nil {
		return false
	}
	_ = in.Close()
	_ = out.Close()
	return true
}

// restoreCursor returns the cursor from the bottom of the rendered block (where
// Bubble Tea leaves it) to the block's first row and clears the block, so the
// shell can redraw its prompt right below the line the user was typing on.
func restoreCursor(w io.Writer, m model) {
	h := m.areaHeight()
	if h <= 1 {
		return
	}
	fmt.Fprintf(w, "\x1b[%dA\x1b[J", h-1)
}

// taskDoneMsg reports that a task run in place inside the fullscreen picker
// has finished.
type taskDoneMsg struct {
	task Task
	err  error
}

// model is the root Bubble Tea model: a filterable list of the project's
// tasks.
type model struct {
	opts     Options
	tasks    []Task
	warnings []error

	width  int
	height int
	ready  bool // true once the first tea.WindowSizeMsg has arrived

	// startRow is the 0-based row the inline UI starts on: the line right
	// below the shell prompt, as reported by the terminal.
	startRow      int
	startRowKnown bool // true once tea.CursorPositionMsg has arrived

	filter string // current filter text; empty means no filtering
	cursor int    // index into visible()
	offset int    // index of the first visible row of the list

	status  string    // outcome of the last in-place run, shown in the help line
	started time.Time // when the current in-place run started
	chosen  *Task     // task chosen in inline mode, to run or insert after quitting

	// notice is the transient confirmation shown as a toast (see notice.go);
	// noticeID invalidates the timer of a replaced notice.
	notice   string
	noticeID int

	// chosenInsert reports that chosen was picked with tab: the command goes
	// to the shell prompt instead of being executed.
	chosenInsert bool

	// themeIdx is the active theme (an index into themes) and styles are
	// its concrete styles. ctrl+t opens the theme picker: themeModal reports
	// that it is open, themeBefore is the theme to restore when it is
	// cancelled with esc, and themeOffset the first row shown while the
	// list is windowed.
	themeIdx    int
	styles      styles
	termBg      color.Color
	termFg      color.Color
	themeModal  bool
	themeBefore int
	themeOffset int

	// warningModal reports that the warning box (ctrl+w, or opened at
	// startup when a warning has not been dismissed) is shown over the
	// list, and warningIdx is the warning currently displayed when there is
	// more than one.
	warningModal bool
	warningIdx   int
}

func newModel(opts Options, tasks []Task, warnings []error) model {
	idx, ok := themeIndex(opts.Theme)
	if !ok {
		warnings = append([]error{fmt.Errorf(
			"unknown theme %q, using %q (--list-themes shows the available ones)",
			opts.Theme, defaultTheme)}, warnings...)
	}
	warnOpen, warnIdx := openWarningsAtStart(warnings)
	return model{
		opts:         opts,
		tasks:        tasks,
		warnings:     warnings,
		themeIdx:     idx,
		styles:       newStyles(themes[idx]),
		warningModal: warnOpen,
		warningIdx:   warnIdx,
	}
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	if m.opts.Inline {
		// Find out which row the UI starts on, i.e. which line to anchor the
		// bottom part of the screen to.
		return tea.Batch(tea.RequestCursorPosition, tea.RequestBackgroundColor, tea.RequestForegroundColor)
	}
	return tea.Batch(tea.RequestBackgroundColor, tea.RequestForegroundColor)
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.keepCursorVisible()

	case tea.CursorPositionMsg:
		m.startRow = msg.Y
		m.startRowKnown = true
		m.keepCursorVisible()

	case tea.BackgroundColorMsg:
		m.termBg = msg.Color
		m.styles = newStyles(m.themeAt(m.themeIdx))

	case tea.ForegroundColorMsg:
		m.termFg = msg.Color
		m.styles = newStyles(m.themeAt(m.themeIdx))

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case taskDoneMsg:
		elapsed := time.Since(m.started).Round(100 * time.Millisecond)
		if msg.err != nil {
			m.status = m.styles.err.Render(fmt.Sprintf("✗ %s (%s): %v", msg.task.Name, elapsed, msg.err))
		} else {
			m.status = m.styles.ok.Render(fmt.Sprintf("✓ %s (%s)", msg.task.Name, elapsed))
		}

	case noticeExpiredMsg:
		if msg.id == m.noticeID {
			m.notice = ""
		}
	}

	return m, nil
}

// handleKey reacts to a key press.
func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.warningModal {
		return m.handleWarningKey(msg)
	}
	if m.themeModal {
		return m.handleThemeKey(msg)
	}

	switch key {
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.cursor, m.offset = 0, 0
			return m, nil
		}
		return m, tea.Quit

	case "enter":
		return m.run()

	case "tab":
		// Inline only: put the command in the shell prompt without running it.
		// There is no prompt to fill in fullscreen mode, so tab does nothing.
		if m.opts.Inline {
			return m.pick(true)
		}
		return m, nil

	case "up", "ctrl+k", "ctrl+p":
		m.cursor--

	case "down", "ctrl+j", "ctrl+n":
		m.cursor++

	case "pgup":
		m.cursor -= m.listHeight()

	case "pgdown":
		m.cursor += m.listHeight()

	case "home":
		m.cursor = 0

	case "end":
		m.cursor = len(m.visible()) - 1

	case "ctrl+u":
		m.filter = ""
		m.cursor, m.offset = 0, 0

	case "ctrl+w":
		if len(m.warnings) > 0 {
			m.warningModal = true
		}
		return m, nil

	// Open the theme picker over the list; the theme in use is kept if the
	// choice is cancelled with esc.
	case "ctrl+t":
		m.themeModal = true
		m.themeBefore = m.themeIdx
		m.keepThemeVisible()

	case "backspace":
		m.filter = trimLastRune(m.filter)
		m.cursor, m.offset = 0, 0

	default:
		// Printable input goes into the filter: q types, it does not quit.
		if msg.Text == "" || msg.Text == "\t" || msg.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
			return m, nil
		}
		m.filter += msg.Text
		m.cursor, m.offset = 0, 0
	}

	m.keepCursorVisible()
	return m, nil
}

// handleWarningKey reacts to a key while the warning box is open: enter, esc
// and ctrl+w close it, d marks the current warning as dismissed so future runs
// do not pop the box open for it, and ↑/↓ step through multiple warnings.
func (m model) handleWarningKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "ctrl+w":
		m.warningModal = false

	case "d":
		m.dismissCurrentWarning()

	case "up", "ctrl+k", "ctrl+p", "left", "ctrl+h":
		m.warningIdx = max(m.warningIdx-1, 0)

	case "down", "ctrl+j", "ctrl+n", "right", "ctrl+l":
		m.warningIdx = min(m.warningIdx+1, len(m.warnings)-1)

	case "home":
		m.warningIdx = 0

	case "end":
		m.warningIdx = max(len(m.warnings)-1, 0)
	}
	return m, nil
}

// handleThemeKey reacts to a key while the theme picker is open: ↑/↓ scroll
// the list and preview each theme live, enter and ctrl+t keep the previewed
// one, esc restores the theme that was active when the picker opened.
func (m model) handleThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.themeModal = false
		m.setTheme(m.themeBefore)
		return m, nil

	case "enter", "ctrl+t":
		m.themeModal = false
		cmd := m.persistTheme()
		return m, cmd

	case "up", "ctrl+k", "ctrl+p":
		m.setTheme(max(m.themeIdx-1, 0))

	case "down", "ctrl+j", "ctrl+n":
		m.setTheme(min(m.themeIdx+1, len(themes)-1))

	case "pgup":
		m.setTheme(max(m.themeIdx-m.themeModalWindow(), 0))

	case "pgdown":
		m.setTheme(min(m.themeIdx+m.themeModalWindow(), len(themes)-1))

	case "home":
		m.setTheme(0)

	case "end":
		m.setTheme(len(themes) - 1)

	default:
		// The picker owns the keyboard: typing does not reach the filter.
		return m, nil
	}

	m.keepThemeVisible()
	return m, nil
}

// setTheme activates the theme at index i and rebuilds its styles.
func (m *model) setTheme(i int) {
	m.themeIdx = i
	m.styles = newStyles(m.themeAt(i))
}

// themeAt returns the theme at index i, resolving the "system" theme against
// the terminal's detected background and foreground colors when available.
func (m model) themeAt(i int) theme {
	th := themes[i]
	if th.name == "system" && (m.termBg != nil || m.termFg != nil) {
		return systemTheme(m.termBg, m.termFg)
	}
	return th
}

// persistTheme saves the previewed theme when it differs from the one that
// was active when the picker opened (or when a theme warning is active). A
// saved theme re-parses the rc files to clear resolved warnings and confirms
// the save with a toast (showNotice); a failure is a status message, never
// fatal, and stays there.
func (m *model) persistTheme() tea.Cmd {
	if m.themeIdx == m.themeBefore && !m.hasThemeWarning() {
		return nil
	}
	name := themes[m.themeIdx].name
	path, err := saveTheme(name)
	if err != nil {
		m.status = m.styles.err.Render(fmt.Sprintf("cannot save theme %s: %v", name, err))
		return nil
	}
	m.opts.Theme = name
	m.refreshConfigWarnings()
	return m.showNotice(fmt.Sprintf("theme %s saved to %s", name, shortPath(path)))
}

// hasThemeWarning reports whether any current warning is about a theme.
func (m model) hasThemeWarning() bool {
	for _, w := range m.warnings {
		if strings.Contains(w.Error(), "theme") {
			return true
		}
	}
	return false
}

// refreshConfigWarnings re-parses the rc files and task sources after an
// action that modifies the config file, dropping resolved warnings.
func (m *model) refreshConfigWarnings() {
	old := m.warnings
	tasks, _, warnings := discover()
	if len(tasks) > 0 {
		m.tasks = tasks
	}
	clearResolvedDismissals(old, warnings)
	m.warnings = warnings
	if len(m.warnings) == 0 {
		m.warningModal = false
		m.warningIdx = 0
	} else {
		m.warningIdx = min(m.warningIdx, len(m.warnings)-1)
	}
	m.keepCursorVisible()
}

// run starts the task under the cursor.
func (m model) run() (tea.Model, tea.Cmd) {
	vis := m.visible()
	if len(vis) == 0 {
		return m, nil
	}
	t := vis[m.cursor]

	if m.opts.Inline {
		// The picker is only a launcher here: it quits and the shell (or Run,
		// when there is no widget) deals with the task afterwards.
		return m.pick(false)
	}

	m.status = m.styles.dim.Render("running " + t.Name + "…")
	m.started = time.Now()
	return m, tea.ExecProcess(t.command(), func(err error) tea.Msg {
		return taskDoneMsg{task: t, err: err}
	})
}

// pick marks the task under the cursor as the picker's result and quits.
// insert selects the tab behaviour: the command goes to the prompt instead
// of being executed.
func (m model) pick(insert bool) (tea.Model, tea.Cmd) {
	vis := m.visible()
	if len(vis) == 0 {
		return m, nil
	}
	t := vis[m.cursor]
	m.chosen = &t
	m.chosenInsert = insert
	return m, tea.Quit
}

// filterTokens returns the lowercase whitespace-separated tokens of the
// current filter.
func (m model) filterTokens() []string {
	return strings.Fields(strings.ToLower(m.filter))
}

// visible returns the tasks matching the current filter, ranked from best to
// worst match (ties keep discovery order).
func (m model) visible() []Task {
	tokens := m.filterTokens()
	if len(tokens) == 0 {
		return m.tasks
	}

	type scoredTask struct {
		task  Task
		score int
	}
	matches := make([]scoredTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		if score, ok := t.matchScore(tokens); ok {
			matches = append(matches, scoredTask{task: t, score: score})
		}
	}
	slices.SortStableFunc(matches, func(a, b scoredTask) int {
		return b.score - a.score
	})

	out := make([]Task, len(matches))
	for i, m := range matches {
		out[i] = m.task
	}
	return out
}

// matches reports whether the task matches all tokens, as case-insensitive
// subsequences of its name, command line or description: the rows show the
// command and the description (e.g. a package.json script body), so typing
// "npm", "tsc" or "vite" must find it.
func (t Task) matches(tokens []string) bool {
	_, ok := t.matchScore(tokens)
	return ok
}

// matchScore scores how well t matches all tokens. Contiguous matches always
// outrank non-contiguous subsequence matches, and within the same tier a
// match in the task name or command line outranks one in the description.
func (t Task) matchScore(tokens []string) (int, bool) {
	const (
		contigCmdTier  = 2000
		contigDescTier = 1000
		fuzzyCmdTier   = 500
		nameTierBonus  = 300
		exactNameBonus = 500
		prefixBonus    = 200
	)

	cmdLine := t.commandLine()
	nameLower := strings.ToLower(t.Name)
	total := 0

	for _, token := range tokens {
		nameScore, nameContig, nameOK := scoreSubsequence(t.Name, token)
		cmdScore, cmdContig, cmdOK := scoreSubsequence(cmdLine, token)
		descScore, descContig, descOK := scoreSubsequence(t.Desc, token)
		if !nameOK && !cmdOK && !descOK {
			return 0, false
		}

		best := -1 << 30
		if nameOK || cmdOK {
			tier := fuzzyCmdTier
			if nameContig || cmdContig {
				tier = contigCmdTier
			}
			score := tier + cmdScore - min(len(cmdLine), 80)
			if nameOK {
				score += nameTierBonus + nameScore - min(len(t.Name), 50)
				switch {
				case nameLower == token:
					score += exactNameBonus
				case strings.HasPrefix(nameLower, token):
					score += prefixBonus
				}
			}
			best = max(best, score)
		}
		if descOK {
			score := descScore
			if descContig {
				score += contigDescTier
			}
			best = max(best, score)
		}
		total += best
	}
	return total, true
}

// scoreSubsequence reports the best alignment score of needle (already
// lowercase) in s, whether the match is contiguous, and whether it matched.
func scoreSubsequence(s, needle string) (int, bool, bool) {
	if s == "" || needle == "" {
		return 0, false, false
	}
	runes := []rune(s)
	lower := make([]rune, len(runes))
	for i, r := range runes {
		lower[i] = unicode.ToLower(r)
	}
	pos, score, contig := subsequenceMatch(runes, lower, []rune(needle))
	if len(pos) == 0 {
		return 0, false, false
	}
	return score, contig, true
}

// matchPositions returns a boolean mask over the runes of s marking the
// characters matched by any token in tokens, or nil when no token matches s.
func matchPositions(s string, tokens []string) []bool {
	if s == "" || len(tokens) == 0 {
		return nil
	}
	runes := []rune(s)
	lower := make([]rune, len(runes))
	for i, r := range runes {
		lower[i] = unicode.ToLower(r)
	}

	var matched []bool
	for _, token := range tokens {
		pos, _, _ := subsequenceMatch(runes, lower, []rune(token))
		if len(pos) == 0 {
			continue
		}
		if matched == nil {
			matched = make([]bool, len(runes))
		}
		for _, idx := range pos {
			matched[idx] = true
		}
	}
	return matched
}

// subsequenceMatch finds the best fzf-style alignment of needle (already
// lowercase) in lower. It prefers contiguous substring matches first; when no
// contiguous match exists, it falls back to a word-anchored subsequence
// (starting at a word boundary, advancing within the same word or jumping to
// later word boundaries) so scattered mid-word letters across unrelated words
// do not produce false positives.
func subsequenceMatch(runes, lower, needle []rune) ([]int, int, bool) {
	m, n := len(needle), len(lower)
	if m == 0 || m > n {
		return nil, 0, false
	}

	const (
		unmatched        = -1 << 30
		baseBonus        = 16
		consecutiveBonus = 16
		boundaryBonus    = 10
		wholeWordBonus   = 12
	)

	// 1. Prefer a contiguous substring match anywhere in lower.
	bestStart := -1
	bestScore := unmatched
	for i := 0; i <= n-m; i++ {
		if !slices.Equal(lower[i:i+m], needle) {
			continue
		}
		score := m*baseBonus + (m-1)*consecutiveBonus
		if isWordBoundary(runes, i) {
			score += boundaryBonus
			if isWordEnd(runes, i+m-1) {
				score += wholeWordBonus
			}
		}
		if i == 0 {
			score += 4
		}
		if score > bestScore {
			bestScore = score
			bestStart = i
		}
	}
	if bestStart >= 0 {
		pos := make([]int, m)
		for k := range m {
			pos[k] = bestStart + k
		}
		return pos, bestScore, true
	}

	// 2. Fall back to a word-anchored subsequence: the match must start on a
	// word boundary and may only skip characters within the same word or jump
	// to the boundary of a later word.
	scores := make([]int, m*n)
	prev := make([]int, m*n)
	for i := range scores {
		scores[i] = unmatched
		prev[i] = -1
	}

	for j := range n {
		if lower[j] == needle[0] && isWordBoundary(runes, j) {
			scores[j] = baseBonus + boundaryBonus
		}
	}

	for k := 1; k < m; k++ {
		row := k * n
		prevRow := (k - 1) * n
		for j := k; j < n; j++ {
			if lower[j] != needle[k] {
				continue
			}
			bestScore := unmatched
			bestPrev := -1
			jBoundary := isWordBoundary(runes, j)
			for p := k - 1; p < j; p++ {
				pScore := scores[prevRow+p]
				if pScore == unmatched {
					continue
				}
				var candidate int
				switch {
				case p == j-1:
					candidate = pScore + baseBonus + consecutiveBonus
				case jBoundary:
					candidate = pScore + baseBonus + boundaryBonus - (j - p - 1)
				case sameWord(runes, p, j):
					candidate = pScore + baseBonus - (j - p - 1)
				default:
					continue
				}
				if candidate > bestScore || (candidate == bestScore && p > bestPrev) {
					bestScore = candidate
					bestPrev = p
				}
			}
			scores[row+j] = bestScore
			prev[row+j] = bestPrev
		}
	}

	lastRow := (m - 1) * n
	bestEnd := -1
	bestScore = unmatched
	for j := m - 1; j < n; j++ {
		if scores[lastRow+j] > bestScore {
			bestScore = scores[lastRow+j]
			bestEnd = j
		}
	}
	if bestEnd < 0 {
		return nil, 0, false
	}

	pos := make([]int, m)
	curr := bestEnd
	for k := m - 1; k >= 0; k-- {
		pos[k] = curr
		curr = prev[k*n+curr]
	}
	return pos, bestScore, false
}

// isWordBoundary reports whether rune j sits at the start of a word (start of
// string, after a non-alphanumeric separator, or a camelCase transition).
func isWordBoundary(runes []rune, j int) bool {
	if j == 0 {
		return true
	}
	prev := runes[j-1]
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsLower(prev) && unicode.IsUpper(runes[j])
}

// isWordEnd reports whether rune j sits at the end of a word.
func isWordEnd(runes []rune, j int) bool {
	if j+1 >= len(runes) {
		return true
	}
	next := runes[j+1]
	return !unicode.IsLetter(next) && !unicode.IsDigit(next)
}

// sameWord reports whether all runes in runes[start:end+1] belong to the same
// alphanumeric word.
func sameWord(runes []rune, start, end int) bool {
	for i := start; i <= end; i++ {
		if !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) {
			return false
		}
	}
	return true
}

// keepCursorVisible clamps the cursor and scrolls the list so that the
// cursor row is inside the window.
func (m *model) keepCursorVisible() {
	vis := m.visible()
	if len(vis) == 0 {
		m.cursor, m.offset = 0, 0
		return
	}

	m.cursor = min(max(m.cursor, 0), len(vis)-1)

	h := m.listHeight()
	m.offset = min(max(m.offset, 0), max(len(vis)-h, 0))
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
}

// trimLastRune removes the last rune of s.
func trimLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

// View implements tea.Model.
func (m model) View() tea.View {
	if m.opts.Inline {
		h := m.areaHeight()
		if h <= 0 {
			// The terminal size or the cursor position is still unknown:
			// render nothing rather than flash the UI in the wrong spot.
			return tea.NewView("")
		}
		if m.opts.DebugArea {
			// The border takes 2 columns and 2 rows of the area for itself.
			return tea.NewView(areaStyle.Render(m.render(max(h-2, 0), max(m.width-2, 0))))
		}
		return tea.NewView(m.render(h, m.width))
	}

	v := tea.NewView(m.render(m.viewHeight(), m.width))
	v.AltScreen = true
	return v
}

// viewHeight returns the number of rows the fullscreen view occupies, or the
// natural size while the terminal size is still unknown.
func (m model) viewHeight() int {
	if !m.ready {
		return chromeRows + m.listHeight()
	}
	return m.height
}

// listHeight returns the number of task rows the list can show.
func (m model) listHeight() int {
	switch {
	case m.opts.Inline:
		return max(m.areaHeight()-chromeRows, 1)
	case !m.ready:
		return max(len(m.visible()), 1)
	default:
		return max(m.height-chromeRows, 1)
	}
}

// render lays the picker out in exactly h rows and w columns: title, filter
// line, task list and help line, padded or trimmed to fit.
func (m model) render(h, w int) string {
	lines := make([]string, 0, h)
	lines = append(lines,
		m.titleLine(),
		m.filterLine(),
		"",
	)
	lines = append(lines, m.listLines(w)...)
	lines = append(lines, m.helpLine(w))

	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i, line := range lines {
		if w > 0 {
			// One row per line, no wrapping, whatever the content.
			lines[i] = ansi.Truncate(line, w, "…")
		}
	}
	if boxes := m.boxes(h, w); len(boxes) > 0 {
		return overlay(lines, boxes...)
	}
	return strings.Join(lines, "\n")
}

// placedBox is a rendered box and the cell its first line starts on.
type placedBox struct {
	lines []string
	x, y  int
}

// overlay draws the boxes over the rendered lines with the canvas compositor:
// it replaces cell by cell, so a background that crosses a box edge is not
// broken (splicing styled lines manually would). Later boxes draw on top.
func overlay(lines []string, boxes ...placedBox) string {
	layers := make([]*lipgloss.Layer, 0, len(boxes)+1)
	layers = append(layers, lipgloss.NewLayer(strings.Join(lines, "\n")))
	for i, b := range boxes {
		layers = append(layers, lipgloss.NewLayer(strings.Join(b.lines, "\n")).X(b.x).Y(b.y).Z(i+1))
	}
	return lipgloss.NewCompositor(layers...).Render()
}

// boxes returns the overlays to draw over the picker: the transient notice in
// the top-right corner, the warning box, and the centered theme picker.
func (m model) boxes(h, w int) []placedBox {
	var boxes []placedBox
	if box := m.noticeBox(w); box != nil {
		boxes = append(boxes, *box)
	}
	if m.warningModal {
		if lines := m.warningModalLines(h, w); lines != nil {
			boxes = append(boxes, placedBox{
				lines: lines,
				x:     max((w-lipgloss.Width(lines[0]))/2, 0),
				y:     m.warningModalY(h, w, len(lines)),
			})
		}
	}
	if m.themeModal {
		if lines := m.themeModalLines(h, w); lines != nil {
			boxes = append(boxes, placedBox{
				lines: lines,
				x:     max((w-lipgloss.Width(lines[0]))/2, 0),
				y:     max((h-len(lines))/2, 0),
			})
		}
	}
	return boxes
}

// warningModalY places the warning box so the bottom help line stays visible:
// centered above the help line when the task list is tall enough, right below
// it when only a couple of tasks are present, or centered in the view on tiny
// terminals.
func (m model) warningModalY(h, w, boxH int) int {
	helpRow := min(3+len(m.listLines(w)), max(h-1, 0))
	switch {
	case helpRow >= boxH:
		return max((helpRow-boxH)/2, min(1, helpRow-boxH))
	case helpRow+1+boxH <= h:
		return helpRow + 1
	default:
		return max((h-boxH)/2, 0)
	}
}

// titleLine renders the picker's title: the invoked name.
func (m model) titleLine() string {
	return m.styles.title.Render(m.opts.prog())
}

// filterLine renders the filter input.
func (m model) filterLine() string {
	if m.filter == "" {
		return m.styles.filter.Render("» ") + m.styles.dim.Render("type to filter")
	}
	return m.styles.filter.Render("» ") + m.filter + m.styles.filter.Render("▏")
}

// listLines renders the visible window of the filtered task list.
func (m model) listLines(w int) []string {
	vis := m.visible()
	if len(vis) == 0 {
		if m.filter != "" {
			return []string{m.styles.dim.Render("no task matches " + strconv.Quote(m.filter))}
		}
		return []string{m.styles.dim.Render("no tasks found")}
	}

	h := m.listHeight()
	off := min(m.offset, max(len(vis)-h, 0))
	end := min(off+h, len(vis))

	rows := make([]string, 0, end-off)
	for i, t := range vis[off:end] {
		rows = append(rows, m.taskRow(t, off+i == m.cursor, w))
	}
	return rows
}

// taskRow renders one list row: the cursor marker, the command line that will
// run, the description and the source file, right-aligned. The command comes
// first and keeps its whole text — it already contains the task name for
// every runner — then the description takes the room that is left, and is
// dropped before the command shrinks. The whole selected row carries the
// theme's tint, or reverse video in the colorless theme.
func (m model) taskRow(t Task, selected bool, w int) string {
	badge := t.Source
	badgeW := lipgloss.Width(badge)

	marker := "  "
	if selected {
		marker = "▌ "
	}

	room := max(w-badgeW-1, 0) // columns left of the badge separator
	avail := max(room-2, 0)    // columns after the marker
	tokens := m.filterTokens()

	cmdW := min(m.commandWidth(), avail)
	cmdRaw := t.commandLine()
	cmd, cmdMatch := truncateMatched(cmdRaw, matchPositions(cmdRaw, tokens), cmdW)
	cmd += strings.Repeat(" ", max(cmdW-lipgloss.Width(cmd), 0))

	desc := t.Desc
	if desc != "" && strings.Contains(strings.ToLower(cmdRaw), strings.ToLower(desc)) {
		// The command already says it: rc tasks without a custom
		// description default to their own command.
		desc = ""
	}
	var (
		descLine  string
		descMatch []bool
	)
	if descW := avail - cmdW - 1; desc != "" && descW > 0 {
		var dm []bool
		desc, dm = truncateMatched(desc, matchPositions(desc, tokens), descW)
		descLine = " " + desc
		if dm != nil {
			descMatch = append([]bool{false}, dm...)
		}
	}

	if selected {
		// The marker in the accent color, the rest of the row in the
		// theme's light tone over its tint (reverse video, in mono).
		left := m.styles.marker.Render(marker) +
			renderMatched(cmd, cmdMatch, m.styles.selectedName, m.styles.selectedMatch)
		if descLine != "" {
			left += renderMatched(descLine, descMatch, m.styles.selectedText, m.styles.selectedMatch)
		}
		pad := strings.Repeat(" ", max(room-lipgloss.Width(left), 0))
		if pad != "" {
			pad = m.styles.selectedPad.Render(pad)
		}
		return left + pad + m.styles.selectedText.Render(" "+badge)
	}

	left := marker + renderMatched(cmd, cmdMatch, lipgloss.Style{}, m.styles.match)
	if descLine != "" {
		left += renderMatched(descLine, descMatch, m.styles.dim, m.styles.match)
	}
	pad := strings.Repeat(" ", max(room-lipgloss.Width(left), 0))
	return left + pad + " " + m.styles.dim.Render(badge)
}

// truncateMatched fits s within maxW terminal columns while keeping any
// matched runes visible: if right-truncating would cut off a match, the
// visible window shifts toward the matched span with a leading ellipsis.
func truncateMatched(s string, matched []bool, maxW int) (string, []bool) {
	if maxW <= 0 || s == "" {
		return "", nil
	}
	if lipgloss.Width(s) <= maxW {
		return s, matched
	}
	if maxW == 1 {
		return "…", nil
	}

	firstMatch, lastMatch := -1, -1
	for i, ok := range matched {
		if ok {
			if firstMatch < 0 {
				firstMatch = i
			}
			lastMatch = i
		}
	}

	runes := []rune(s)
	width, end := 0, 0
	for end < len(runes) {
		rw := ansi.StringWidth(string(runes[end]))
		if width+rw > maxW-1 {
			break
		}
		width += rw
		end++
	}
	if firstMatch < 0 || lastMatch < end {
		var m []bool
		if len(matched) > 0 {
			m = append(slices.Clone(matched[:end]), false)
		}
		return string(runes[:end]) + "…", m
	}

	// A match lies past the right truncation point: try a suffix anchored
	// to the end of s first, then a middle window framed by two ellipses.
	suffixW := 0
	start := len(runes)
	for start > 0 {
		rw := ansi.StringWidth(string(runes[start-1]))
		if 1+suffixW+rw > maxW {
			break
		}
		suffixW += rw
		start--
	}
	if start <= firstMatch {
		m := make([]bool, 1+len(runes)-start)
		copy(m[1:], matched[start:])
		return "…" + string(runes[start:]), m
	}

	if maxW < 3 {
		return ansi.Truncate(s, maxW, "…"), nil
	}
	start, end = firstMatch, firstMatch
	midW := 0
	for end < len(runes) {
		rw := ansi.StringWidth(string(runes[end]))
		if midW+rw > maxW-2 {
			break
		}
		midW += rw
		end++
	}
	for start > 0 {
		rw := ansi.StringWidth(string(runes[start-1]))
		if midW+rw > maxW-2 {
			break
		}
		midW += rw
		start--
	}
	m := make([]bool, 0, (end-start)+2)
	m = append(m, false)
	m = append(m, matched[start:end]...)
	m = append(m, false)
	return "…" + string(runes[start:end]) + "…", m
}

// renderMatched renders s applying highlight to the runes marked in matched
// and base to the rest. Adjacent runes with the same state are grouped into a
// single segment so no unnecessary ANSI resets are emitted.
func renderMatched(s string, matched []bool, base, highlight lipgloss.Style) string {
	if s == "" {
		return ""
	}
	if !slices.Contains(matched, true) {
		return base.Render(s)
	}

	runes := []rune(s)
	var b strings.Builder
	start := 0
	for start < len(runes) {
		isMatch := start < len(matched) && matched[start]
		end := start + 1
		for end < len(runes) && (end < len(matched) && matched[end]) == isMatch {
			end++
		}
		seg := string(runes[start:end])
		if isMatch {
			b.WriteString(highlight.Render(seg))
		} else {
			b.WriteString(base.Render(seg))
		}
		start = end
	}
	return b.String()
}

// commandWidth returns the width of the command column: the widest visible
// command line, so the descriptions of all rows start together. taskRow caps
// it to the room the row has; only a command longer than that is truncated.
func (m model) commandWidth() int {
	cmdW := 0
	for _, t := range m.visible() {
		cmdW = max(cmdW, lipgloss.Width(t.commandLine()))
	}
	return cmdW
}

// helpLine renders the bottom line in w columns: the outcome of the last
// in-place run if there is one, otherwise the task count (and warning count,
// when present) followed by the key hints. Optional hints are dropped when the
// line would not fit (inline mode or narrow terminals), so the keys before
// them stay readable.
func (m model) helpLine(w int) string {
	if m.status != "" {
		return m.status
	}

	count := fmt.Sprintf("%d tasks", len(m.tasks))
	keys := "type to filter · ↑/↓ select · enter run · esc quit"
	if m.filter != "" {
		count = fmt.Sprintf("%d/%d tasks", len(m.visible()), len(m.tasks))
		keys = "↑/↓ select · enter run · esc clear filter"
	}
	if m.opts.Inline {
		keys = strings.Replace(keys, "enter run", "enter run · tab to prompt", 1)
	}

	leftPlain := count
	left := m.styles.dim.Render(count)
	if len(m.warnings) > 0 {
		wc := warningCount(len(m.warnings))
		leftPlain += " · " + wc
		left += m.styles.dim.Render(" · ") + m.styles.err.Render(wc)

		warnHint := " · ctrl+w warnings"
		if w > 0 && ansi.StringWidth(leftPlain+"   "+keys+warnHint) > w {
			keys = strings.TrimPrefix(keys, "type to filter · ")
			keys = strings.Replace(keys, "esc clear filter", "esc clear", 1)
		}
		if w <= 0 || ansi.StringWidth(leftPlain+"   "+keys+warnHint) <= w {
			keys += warnHint
		}
	}

	if hint := " · ctrl+t theme"; w <= 0 || ansi.StringWidth(leftPlain+"   "+keys+hint) <= w {
		keys += hint
	}
	return left + m.styles.dim.Render("   "+keys)
}

// areaHeight returns the number of rows the inline UI occupies: the space
// between the line below the prompt and the bottom of the screen. When fewer
// than minInlineHeight rows are free, the minimum is used and the terminal
// scrolls up to make room. It returns 0 while the terminal size or the start
// row is still unknown.
func (m model) areaHeight() int {
	if !m.ready || !m.startRowKnown {
		return 0
	}
	if h := m.height - m.startRow; h > minInlineHeight {
		return h
	}
	return minInlineHeight
}

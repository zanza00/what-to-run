// Package tui implements the terminal user interface of what-to-run.
package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// accent is the picker's accent color.
var accent = lipgloss.Color("#7D56F4")

// Styles are stateless and safe to reuse across model values.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	// selectedStyle highlights the row under the cursor.
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(accent)

	filterStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))

	// areaStyle draws the border around the inline area when DebugArea is
	// set: a development aid to check that the UI covers exactly the rows
	// between the prompt and the bottom of the screen.
	areaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

// chromeRows is the number of rows the picker reserves around the task list:
// title, filter line, blank separator and help line.
const chromeRows = 4

// minInlineHeight is the smallest area reserved for the inline UI. Television
// uses the same value: when fewer rows are free below the cursor, the
// terminal is left to scroll up instead of squeezing the UI.
const minInlineHeight = 15

// Options configures the TUI.
type Options struct {
	// Inline renders the UI between the current cursor line and the bottom of
	// the screen instead of taking over the whole screen. It is meant for
	// shell widgets that launch the binary right below the prompt (the zsh
	// integration from 'what-to-run init zsh', see ZshInit) so that
	// the shell history above stays visible.
	Inline bool

	// DebugArea draws a border around the whole inline area so the region
	// managed by the app is visible. It is a development aid and is ignored
	// in fullscreen mode.
	DebugArea bool
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
		fmt.Fprintln(os.Stderr, "what-to-run: --inline requires an interactive terminal, running fullscreen")
		opts.Inline = false
	}

	// A widget captures stdout to receive the result; the UI then renders on
	// the terminal itself, like fzf does inside shell integrations.
	var progOpts []tea.ProgramOption
	out := io.Writer(os.Stdout)
	if opts.Inline && !term.IsTerminal(os.Stdout.Fd()) {
		in, ttyOut, err := tea.OpenTTY()
		if err != nil {
			fmt.Fprintln(os.Stderr, "what-to-run: cannot open the terminal, running fullscreen")
			opts.Inline = false
		} else {
			defer func() { _ = in.Close(); _ = ttyOut.Close() }()
			progOpts = []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(ttyOut)}
			out = ttyOut
		}
	}

	cfg, warnings := loadConfig(defaultConfigFiles("."))
	tasks, discoveryWarnings := discoverTasks(".", cfg)
	warnings = append(warnings, discoveryWarnings...)
	if len(tasks) == 0 {
		if len(warnings) > 0 {
			return fmt.Errorf("no tasks found: %w", errors.Join(warnings...))
		}
		return errors.New("no tasks found: no package.json scripts, Makefile targets, cargo-make tasks or rc file tasks")
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

	// chosenInsert reports that chosen was picked with tab: the command goes
	// to the shell prompt instead of being executed.
	chosenInsert bool
}

func newModel(opts Options, tasks []Task, warnings []error) model {
	return model{opts: opts, tasks: tasks, warnings: warnings}
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	if m.opts.Inline {
		// Find out which row the UI starts on, i.e. which line to anchor the
		// bottom part of the screen to.
		return tea.RequestCursorPosition
	}
	return nil
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

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case taskDoneMsg:
		elapsed := time.Since(m.started).Round(100 * time.Millisecond)
		if msg.err != nil {
			m.status = errStyle.Render(fmt.Sprintf("✗ %s (%s): %v", msg.task.Name, elapsed, msg.err))
		} else {
			m.status = okStyle.Render(fmt.Sprintf("✓ %s (%s)", msg.task.Name, elapsed))
		}
	}

	return m, nil
}

// handleKey reacts to a key press.
func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

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

	m.status = dimStyle.Render("running " + t.Name + "…")
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

// visible returns the tasks matching the current filter, in display order.
func (m model) visible() []Task {
	if m.filter == "" {
		return m.tasks
	}

	tokens := strings.Fields(strings.ToLower(m.filter))
	matches := make([]Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		if t.matches(tokens) {
			matches = append(matches, t)
		}
	}
	return matches
}

// matches reports whether the task matches all tokens, as case-insensitive
// subsequences of its name.
func (t Task) matches(tokens []string) bool {
	name := strings.ToLower(t.Name)
	for _, token := range tokens {
		if !subsequence(name, token) {
			return false
		}
	}
	return true
}

// subsequence reports whether needle is a subsequence of the lowercase
// haystack.
func subsequence(haystack, needle string) bool {
	rest := needle
	for _, r := range haystack {
		if rest == "" {
			return true
		}
		if r == rune(rest[0]) {
			rest = rest[1:]
		}
	}
	return rest == ""
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
		titleStyle.Render("what-to-run"),
		m.filterLine(),
		"",
	)
	lines = append(lines, m.listLines(w)...)
	lines = append(lines, m.helpLine())

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
	return strings.Join(lines, "\n")
}

// filterLine renders the filter input.
func (m model) filterLine() string {
	if m.filter == "" {
		return filterStyle.Render("» ") + dimStyle.Render("type to filter")
	}
	return filterStyle.Render("» ") + m.filter + filterStyle.Render("▏")
}

// listLines renders the visible window of the filtered task list.
func (m model) listLines(w int) []string {
	vis := m.visible()
	if len(vis) == 0 {
		if m.filter != "" {
			return []string{dimStyle.Render("no task matches " + strconv.Quote(m.filter))}
		}
		return []string{dimStyle.Render("no tasks found")}
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

// taskRow renders one list row: cursor marker, task name, description and the
// source file, right-aligned.
func (m model) taskRow(t Task, selected bool, w int) string {
	badge := t.Source
	badgeW := lipgloss.Width(badge)

	marker := "  "
	if selected {
		marker = "› "
	}

	nameW := m.nameWidth(w)
	name := t.Name
	if lipgloss.Width(name) > nameW {
		name = ansi.Truncate(name, nameW, "…")
	}
	name += strings.Repeat(" ", max(nameW-lipgloss.Width(name), 0))

	// marker(2) + name(nameW) + space(1) + description, padded to leave room
	// for a space and the badge at the right edge.
	room := max(w-badgeW-1, 0)
	var desc string
	if descW := room - 2 - nameW - 1; descW > 0 {
		desc = ansi.Truncate(t.Desc, descW, "…")
	}

	if selected {
		left := marker + name + " " + desc
		pad := strings.Repeat(" ", max(room-lipgloss.Width(left), 0))
		return selectedStyle.Render(left + pad + " " + badge)
	}

	left := marker + name + " " + dimStyle.Render(desc)
	pad := strings.Repeat(" ", max(room-lipgloss.Width(left), 0))
	return left + pad + " " + dimStyle.Render(badge)
}

// nameWidth returns the width of the name column: wide enough for the widest
// visible task name, capped to a third of the available width.
func (m model) nameWidth(w int) int {
	nameW := 0
	for _, t := range m.visible() {
		nameW = max(nameW, lipgloss.Width(t.Name))
	}
	return min(nameW, max(w/3, 12))
}

// helpLine renders the bottom line: the outcome of the last in-place run if
// there is one, then the discovery warnings, then the help text.
func (m model) helpLine() string {
	switch {
	case m.status != "":
		return m.status

	case len(m.warnings) > 0:
		s := m.warnings[0].Error()
		if n := len(m.warnings) - 1; n > 0 {
			s += fmt.Sprintf(" (+%d more)", n)
		}
		return errStyle.Render(s)
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
	return dimStyle.Render(count + "   " + keys)
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

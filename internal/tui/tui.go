// Package tui implements the terminal user interface of what-to-run.
package tui

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// Styles are stateless and safe to reuse across model values.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	helloStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 6)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// areaStyle draws the border around the inline area when DebugArea is
	// set: a development aid to check that the UI covers exactly the rows
	// between the prompt and the bottom of the screen.
	areaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

// minInlineHeight is the smallest area reserved for the inline UI. Television
// uses the same value: when fewer rows are free below the cursor, the
// terminal is left to scroll up instead of squeezing the UI.
const minInlineHeight = 15

// Options configures the TUI.
type Options struct {
	// Inline renders the UI between the current cursor line and the bottom of
	// the screen instead of taking over the whole screen. It is meant for
	// shell widgets that launch the binary right below the prompt (see
	// scripts/ctrl-s.zsh) so that the shell history above stays visible.
	Inline bool

	// DebugArea draws a border around the whole inline area so the region
	// managed by the app is visible. It is a development aid and is ignored
	// in fullscreen mode.
	DebugArea bool
}

// Run starts the TUI and blocks until the user quits. It returns an error
// only if the program fails to start or crashes.
func Run(opts Options) error {
	if opts.Inline && !inlineInputAvailable() {
		fmt.Fprintln(os.Stderr, "what-to-run: --inline requires an interactive terminal, running fullscreen")
		opts.Inline = false
	}

	final, err := tea.NewProgram(newModel(opts)).Run()
	if opts.Inline && term.IsTerminal(os.Stdout.Fd()) {
		if m, ok := final.(model); ok {
			restoreCursor(os.Stdout, m)
		}
	}
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
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

// model is the root Bubble Tea model. For now it renders a hello world and
// quits on q/esc/ctrl+c; it will grow into the task list.
type model struct {
	opts Options

	width  int
	height int
	ready  bool // true once the first tea.WindowSizeMsg has arrived

	// startRow is the 0-based row the inline UI starts on: the line right
	// below the shell prompt, as reported by the terminal.
	startRow      int
	startRowKnown bool // true once tea.CursorPositionMsg has arrived
}

func newModel(opts Options) model {
	return model{opts: opts}
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

	case tea.CursorPositionMsg:
		m.startRow = msg.Y
		m.startRowKnown = true

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m model) View() tea.View {
	body := m.body()

	if m.opts.Inline {
		h := m.areaHeight()
		if h <= 0 {
			// The terminal size or the cursor position is still unknown:
			// render nothing rather than flash the UI in the wrong spot.
			return tea.NewView("")
		}
		if m.opts.DebugArea {
			// The border takes 2 columns and 2 rows of the area for itself.
			inner := lipgloss.Place(max(m.width-2, 0), max(h-2, 0), lipgloss.Center, lipgloss.Center, body)
			return tea.NewView(areaStyle.Render(inner))
		}
		body = lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, body)
		return tea.NewView(body)
	}

	if m.ready {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}

	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

// body renders the UI content without padding; View places it in the area
// available in the current mode.
func (m model) body() string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("what-to-run"),
		"",
		helloStyle.Render("Hello, World!"),
		"",
		helpStyle.Render("press q to quit"),
	)
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

// Package tui implements the terminal user interface of what-to-run.
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
)

// Run starts the TUI and blocks until the user quits. It returns an error
// only if the program fails to start or crashes.
func Run() error {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

// model is the root Bubble Tea model. For now it renders a hello world and
// quits on q/esc/ctrl+c; it will grow into the task list.
type model struct {
	width  int
	height int
	ready  bool // true once the first tea.WindowSizeMsg has arrived
}

func newModel() model {
	return model{}
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true

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
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("what-to-run"),
		"",
		helloStyle.Render("Hello, World!"),
		"",
		helpStyle.Render("press q to quit"),
	)

	if m.ready {
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

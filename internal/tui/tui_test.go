package tui

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// update feeds msg to m and returns the resulting concrete model.
func update(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func TestViewRendersHelloWorld(t *testing.T) {
	got := newModel(Options{}).View().Content
	if !strings.Contains(got, "Hello, World!") {
		t.Errorf("View() content = %q, want it to contain %q", got, "Hello, World!")
	}
}

func TestViewFullscreenUsesAltScreen(t *testing.T) {
	if !newModel(Options{}).View().AltScreen {
		t.Error("View() in fullscreen mode should set AltScreen")
	}
}

func TestViewInlineWaitsForCursorPosition(t *testing.T) {
	m := update(newModel(Options{Inline: true}), tea.WindowSizeMsg{Width: 80, Height: 30})
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
			m := update(newModel(Options{Inline: true}), tea.WindowSizeMsg{Width: 80, Height: tt.height})
			m = update(m, tea.CursorPositionMsg{Y: tt.startRow})

			v := m.View()
			if v.AltScreen {
				t.Error("View() in inline mode should not set AltScreen")
			}
			if got := strings.Count(v.Content, "\n") + 1; got != tt.want {
				t.Errorf("inline view height = %d, want %d", got, tt.want)
			}
			if got := strings.Count(v.Content, "╭"); got != 1 {
				t.Errorf("inline view has %d top-left corners, want 1 (the hello box only; the area border is opt-in)", got)
			}
		})
	}
}

func TestViewInlineDebugArea(t *testing.T) {
	m := update(newModel(Options{Inline: true, DebugArea: true}), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5}) // 25 rows of UI

	v := m.View()
	if got := strings.Count(v.Content, "\n") + 1; got != 25 {
		t.Errorf("inline view height with the debug border = %d, want 25", got)
	}
	if got := strings.Count(v.Content, "╭"); got != 2 {
		t.Errorf("inline view has %d top-left corners, want 2 (hello box + area border)", got)
	}
}

func TestRestoreCursor(t *testing.T) {
	m := update(newModel(Options{Inline: true}), tea.WindowSizeMsg{Width: 80, Height: 30})
	m = update(m, tea.CursorPositionMsg{Y: 5}) // 25 rows of UI, cursor at the bottom

	var buf bytes.Buffer
	restoreCursor(&buf, m)
	if got, want := buf.String(), "\x1b[24A\x1b[J"; got != want {
		t.Errorf("restoreCursor wrote %q, want %q", got, want)
	}

	buf.Reset()
	restoreCursor(&buf, newModel(Options{})) // never rendered
	if got := buf.String(); got != "" {
		t.Errorf("restoreCursor on an unrendered model wrote %q, want nothing", got)
	}
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"q", tea.KeyPressMsg{Code: 'q', Text: "q"}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := newModel(Options{}).Update(tt.key)
			if cmd == nil {
				t.Fatalf("Update(%s) returned nil command, want tea.Quit", tt.name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("Update(%s) command returned %T, want tea.QuitMsg", tt.name, cmd())
			}
		})
	}
}

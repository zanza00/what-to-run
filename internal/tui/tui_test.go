package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestViewRendersHelloWorld(t *testing.T) {
	got := newModel().View().Content
	if !strings.Contains(got, "Hello, World!") {
		t.Errorf("View() content = %q, want it to contain %q", got, "Hello, World!")
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
			_, cmd := newModel().Update(tt.key)
			if cmd == nil {
				t.Fatalf("Update(%s) returned nil command, want tea.Quit", tt.name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("Update(%s) command returned %T, want tea.QuitMsg", tt.name, cmd())
			}
		})
	}
}

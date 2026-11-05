package core

import "testing"

// TestHelpOrBody pins the law of the description chain: the help text of a
// task wins, and the body shows otherwise, as display text.
func TestHelpOrBody(t *testing.T) {
	tests := []struct {
		name string
		help string
		body string
		want string
	}{
		{name: "help wins over the body", help: "Build the app", body: "cargo build", want: "Build the app"},
		{name: "empty help shows the body", help: "", body: "cargo build", want: "cargo build"},
		{name: "help wins when the body is empty", help: "Build the app", body: "", want: "Build the app"},
		{name: "both empty show nothing", help: "", body: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helpOrBody(tt.help, tt.body); got != tt.want {
				t.Errorf("helpOrBody(%q, %q) = %q, want %q", tt.help, tt.body, got, tt.want)
			}
		})
	}
}

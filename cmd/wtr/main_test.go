package main

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInitShells pins the init subcommand dispatch: each shell gets its own
// integration, flags written after the shell name are honoured, and the
// unsupported shells are refused.
func TestRunInitShells(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"zsh", []string{"zsh"}, "zle -N wtr-widget"},
		{"bash", []string{"bash"}, "for _wtr_keymap in emacs-standard vi-insert vi-command; do"},
		{"bash, key after the shell", []string{"bash", "--key", "ctrl-g"}, `WTR_KEY='\C-g'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := runInit(tt.args, &out); err != nil {
				t.Fatalf("runInit(%q) error: %v", tt.args, err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Errorf("runInit(%q) output does not contain %q:\n%s", tt.args, tt.want, out.String())
			}
		})
	}

	for _, args := range [][]string{nil, {"fish"}, {"zsh", "bash"}} {
		if err := runInit(args, io.Discard); err == nil {
			t.Errorf("runInit(%q) should fail", args)
		}
	}
}

func TestProgName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/usr/local/bin/wtr", "wtr"},
		{"what-to-run", "what-to-run"},
		{"bin/wtr", "wtr"},
		{"", "wtr"},
	}
	for _, tt := range tests {
		if got := progName(tt.in); got != tt.want {
			t.Errorf("progName(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestVersionLine(t *testing.T) {
	tests := []struct {
		prog    string
		version string
		want    string
	}{
		{"wtr", "0.1.0", "wtr 0.1.0"},
		{"what-to-run", "dev", "what-to-run dev"},
	}
	for _, tt := range tests {
		if got := versionLine(tt.prog, tt.version); got != tt.want {
			t.Errorf("versionLine(%q, %q) = %q; want %q", tt.prog, tt.version, got, tt.want)
		}
	}
}

func TestWidgetBin(t *testing.T) {
	// A bare name is kept: the widget resolves it through PATH when the key
	// is pressed, so it keeps calling the binary by the invoked name.
	if got := widgetBin("what-to-run"); got != "what-to-run" {
		t.Errorf("widgetBin(%q) = %q; want the bare name", "what-to-run", got)
	}

	// A path is made absolute, so the widget runs exactly that copy.
	abs, err := filepath.Abs("bin/wtr")
	if err != nil {
		t.Fatal(err)
	}
	if got := widgetBin("bin/wtr"); got != abs {
		t.Errorf("widgetBin(%q) = %q; want %q", "bin/wtr", got, abs)
	}

	// An absolute path is kept as is.
	if got := widgetBin(abs); got != abs {
		t.Errorf("widgetBin(%q) = %q; want the same path", abs, got)
	}
}

func TestHelpHint(t *testing.T) {
	const wantHint = "The same program is also available as 'wtr', quicker to type."
	tests := []struct {
		name  string
		alias bool
		want  string
	}{
		{"what-to-run", true, wantHint},
		{"what-to-run", false, ""},
		{"wtr", true, ""},
		{"", true, ""},
	}
	for _, tt := range tests {
		if got := helpHint(tt.name, tt.alias); got != tt.want {
			t.Errorf("helpHint(%q, %v) = %q; want %q", tt.name, tt.alias, got, tt.want)
		}
	}
}

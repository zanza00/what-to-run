package main

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInitShells pins the dispatch of the init subcommand. Each shell gets
// its own integration, a flag written after the shell name is honoured, and an
// unsupported shell is refused.
func TestRunInitShells(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"zsh", []string{"zsh"}, "zle -N wtr-widget"},
		{"bash", []string{"bash"}, "for _wtr_keymap in emacs-standard vi-insert vi-command; do"},
		{"bash, key after the shell", []string{"bash", "--key", "ctrl-g"}, `WTR_KEY='\C-g'`},
		{"powershell", []string{"powershell"}, "Set-PSReadLineKeyHandler"},
		{"pwsh alias, key after the shell", []string{"pwsh", "--key", "alt-x"}, "-Chord 'Alt+x'"},
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

	for _, args := range [][]string{nil, {"fish"}, {"zsh", "bash"}, {"powershell", "--key", "^[x"}} {
		if err := runInit(args, io.Discard); err == nil {
			t.Errorf("runInit(%q) should fail", args)
		}
	}
}

// TestRunInitDefaultKey pins the binding that is used when --key is absent, in
// the notation of each shell.
func TestRunInitDefaultKey(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{"zsh", "bindkey '^O' wtr-widget"},
		{"bash", `WTR_KEY='\C-o'`},
		{"powershell", "-Chord 'Ctrl+o'"},
	}
	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			var out bytes.Buffer
			if err := runInit([]string{tt.shell}, &out); err != nil {
				t.Fatalf("runInit(%q) error: %v", tt.shell, err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Errorf("runInit(%q) output does not contain %q:\n%s", tt.shell, tt.want, out.String())
			}
		})
	}
}

func TestProgName(t *testing.T) {
	tests := []struct {
		name string
		goos string
		arg0 string
		want string
	}{
		{"absolute unix path", "linux", "/usr/local/bin/wtr", "wtr"},
		{"bare name", "linux", "task-picker", "task-picker"},
		{"former shipped name", "linux", "what-to-run", "what-to-run"},
		{"relative unix path", "linux", "bin/wtr", "wtr"},
		{"empty argv0", "linux", "", "wtr"},
		{"exe suffix survives off windows", "linux", "wtr.exe", "wtr.exe"},
		{"windows path", "windows", `C:\Tools\WTR.EXE`, "WTR"},
		{"windows forward slashes", "windows", "C:/Tools/task-picker.exe", "task-picker"},
		{"windows bare name with suffix", "windows", "wtr.exe", "wtr"},
		{"windows bare name", "windows", "wtr", "wtr"},
		{"windows suffix only", "windows", ".exe", "wtr"},
		{"windows empty argv0", "windows", "", "wtr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := progNameFor(tt.goos, tt.arg0); got != tt.want {
				t.Errorf("progNameFor(%q, %q) = %q; want %q", tt.goos, tt.arg0, got, tt.want)
			}
		})
	}
}

func TestVersionLine(t *testing.T) {
	tests := []struct {
		prog    string
		version string
		want    string
	}{
		{"wtr", "0.1.0", "wtr 0.1.0"},
		{"task-picker", "dev", "task-picker dev"},
	}
	for _, tt := range tests {
		if got := versionLine(tt.prog, tt.version); got != tt.want {
			t.Errorf("versionLine(%q, %q) = %q; want %q", tt.prog, tt.version, got, tt.want)
		}
	}
}

func TestWidgetBin(t *testing.T) {
	// A bare name is kept, because the widget resolves it through PATH when the
	// key is pressed. The widget then calls the binary by the invoked name.
	if got := widgetBin("task-picker"); got != "task-picker" {
		t.Errorf("widgetBin(%q) = %q; want the bare name", "task-picker", got)
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

func TestHasPathSeparator(t *testing.T) {
	tests := []struct {
		name string
		goos string
		arg0 string
		want bool
	}{
		{"unix slash", "linux", "bin/wtr", true},
		{"unix backslash is a file name", "linux", `C:\Tools\wtr.exe`, false},
		{"unix bare name", "linux", "wtr", false},
		{"windows slash", "windows", "bin/wtr", true},
		{"windows backslash", "windows", `C:\Tools\wtr.exe`, true},
		{"windows bare name with suffix", "windows", "wtr.exe", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPathSeparator(tt.goos, tt.arg0); got != tt.want {
				t.Errorf("hasPathSeparator(%q, %q) = %v; want %v", tt.goos, tt.arg0, got, tt.want)
			}
		})
	}
}

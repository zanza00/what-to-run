package tui

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParseBindKey(t *testing.T) {
	tests := []struct {
		in   string
		seq  string
		name string
	}{
		{"ctrl-s", "^S", "Ctrl-S"},
		{"CTRL-G", "^G", "Ctrl-G"},
		{"ctrl-q", "^Q", "Ctrl-Q"},
		{"alt-x", "^[x", "Alt-x"},
		{"alt-X", "^[X", "Alt-X"},
		{"alt-1", "^[1", "Alt-1"},
		{"^[[15~", "^[[15~", "^[[15~"},
		{"^s", "^s", "^s"},
		{"  ctrl-t  ", "^T", "Ctrl-T"},
	}
	for _, tt := range tests {
		seq, name, err := parseBindKey(tt.in)
		if err != nil {
			t.Errorf("parseBindKey(%q) error: %v", tt.in, err)
			continue
		}
		if seq != tt.seq || name != tt.name {
			t.Errorf("parseBindKey(%q) = %q, %q; want %q, %q", tt.in, seq, name, tt.seq, tt.name)
		}
	}
}

func TestParseBindKeyInvalid(t *testing.T) {
	for _, in := range []string{"", "s", "ctrl-", "ctrl-ab", "ctrl-1", "alt-", "alt-xy", "shift-x", "^"} {
		if _, _, err := parseBindKey(in); err == nil {
			t.Errorf("parseBindKey(%q) should fail", in)
		}
	}
}

func TestZshInitScript(t *testing.T) {
	got, err := ZshInit("/opt/wtr/what-to-run", "ctrl-g")
	if err != nil {
		t.Fatalf("ZshInit() error: %v", err)
	}
	for _, want := range []string{
		"zle -N what-to-run-widget _what_to_run_widget",
		"bindkey '^G' what-to-run-widget",
		`typeset -g WTR_BIN="${WTR_BIN:-/opt/wtr/what-to-run}"`,
		"out=$(\"$WTR_BIN\" --inline ${=WTR_FLAGS} < /dev/tty)",
		"Ctrl-G opens",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ZshInit() script does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "stty -ixon") {
		t.Errorf("binding Ctrl-G must not touch IXON:\n%s", got)
	}
}

func TestZshInitFlowControlKeyDisablesIXON(t *testing.T) {
	for _, key := range []string{"ctrl-s", "ctrl-q", "^s", "^Q"} {
		got, err := ZshInit("/bin/what-to-run", key)
		if err != nil {
			t.Fatalf("ZshInit(%q) error: %v", key, err)
		}
		if !strings.Contains(got, "stty -ixon 2>/dev/null") {
			t.Errorf("ZshInit(%q) should turn IXON off:\n%s", key, got)
		}
	}
}

func TestZshInitErrors(t *testing.T) {
	for _, tt := range []struct{ bin, key string }{
		{"/bin/what-to-run", "bogus"},
		{"/bin/what-to-run", ""},
		{"", "ctrl-s"},
	} {
		if _, err := ZshInit(tt.bin, tt.key); err == nil {
			t.Errorf("ZshInit(%q, %q) should fail", tt.bin, tt.key)
		}
	}
}

// TestZshInitParsesAsZsh runs the generated script through the real zsh, when
// one is installed: eval-ability is the whole point of `init zsh`.
func TestZshInitParsesAsZsh(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not installed")
	}
	for _, key := range []string{"ctrl-s", "alt-x", "^[[15~"} {
		script, err := ZshInit("/bin/what-to-run", key)
		if err != nil {
			t.Fatalf("ZshInit(%q) error: %v", key, err)
		}
		cmd := exec.Command(zsh, "-n")
		cmd.Stdin = strings.NewReader(script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("zsh -n rejects the script for %q: %v\n%s", key, err, out)
		}
	}
}

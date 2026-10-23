package tui

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParseBashBindKey(t *testing.T) {
	tests := []struct {
		in   string
		seq  string
		name string
	}{
		{"ctrl-s", `\C-s`, "Ctrl-S"},
		{"CTRL-G", `\C-g`, "Ctrl-G"},
		{"ctrl-q", `\C-q`, "Ctrl-Q"},
		{"alt-x", `\ex`, "Alt-x"},
		{"alt-X", `\eX`, "Alt-X"},
		{"alt-1", `\e1`, "Alt-1"},
		{`\e[15~`, `\e[15~`, `\e[15~`},
		{`\C-x`, `\C-x`, `\C-x`},
		{"  ctrl-t  ", `\C-t`, "Ctrl-T"},
	}
	for _, tt := range tests {
		seq, name, err := parseBashBindKey(tt.in)
		if err != nil {
			t.Errorf("parseBashBindKey(%q) error: %v", tt.in, err)
			continue
		}
		if seq != tt.seq || name != tt.name {
			t.Errorf("parseBashBindKey(%q) = %q, %q; want %q, %q", tt.in, seq, name, tt.seq, tt.name)
		}
	}
}

func TestParseBashBindKeyInvalid(t *testing.T) {
	for _, in := range []string{"", "s", "ctrl-", "ctrl-ab", "ctrl-1", "alt-", "alt-xy", "shift-x", `\`, "^[x"} {
		if _, _, err := parseBashBindKey(in); err == nil {
			t.Errorf("parseBashBindKey(%q) should fail", in)
		}
	}
}

func TestBashInitScript(t *testing.T) {
	got, err := BashInit("wtr", "/opt/wtr/wtr", "ctrl-g")
	if err != nil {
		t.Fatalf("BashInit() error: %v", err)
	}
	for _, want := range []string{
		"# wtr: bash integration",
		`WTR_BIN="${WTR_BIN:-/opt/wtr/wtr}"`,
		`out=$("$WTR_BIN" --inline --shell posix ${flags[@]+"${flags[@]}"} < /dev/tty)`,
		`WTR_KEY='\C-g'`,
		"tail=accept-line",
		`_wtr_tail "$tail"`,
		"for _wtr_keymap in emacs-standard vi-insert vi-command; do",
		`bind -m "$_wtr_keymap" -x "\"$WTR_RUN\": _wtr_widget"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("BashInit() script does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "stty -ixon") {
		t.Errorf("binding Ctrl-G must not touch IXON:\n%s", got)
	}
}

// TestBashInitNamesTheInvokedProgram pins the mnemonic rule: the script keeps
// the name it was generated with, so the integration is self-explaining in
// the rc file.
func TestBashInitNamesTheInvokedProgram(t *testing.T) {
	tests := []struct {
		prog string
		want []string
	}{
		{
			prog: "what-to-run",
			want: []string{
				"# what-to-run: bash integration, Ctrl-G opens the inline picker.",
				`eval "$(what-to-run init bash --key Ctrl-G)"`,
			},
		},
		{prog: "wtr", want: []string{"# wtr: bash integration"}},
		{prog: "", want: []string{"# wtr: bash integration"}},
	}
	for _, tt := range tests {
		got, err := BashInit(tt.prog, "/bin/wtr", "ctrl-g")
		if err != nil {
			t.Fatalf("BashInit(%q) error: %v", tt.prog, err)
		}
		for _, want := range tt.want {
			if !strings.Contains(got, want) {
				t.Errorf("BashInit(%q) script does not contain %q:\n%s", tt.prog, want, got)
			}
		}
	}
}

// TestBashInitIsQuiet pins that evaluating the script prints nothing where
// there is nothing to bind: the integration lives in the rc file, and a
// non-interactive shell must not even see readline's complaints.
func TestBashInitIsQuiet(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	for _, prog := range []string{"wtr", "what-to-run", ""} {
		for _, key := range []string{"ctrl-s", "ctrl-g", `\e[15~`} {
			script, err := BashInit(prog, "/bin/wtr", key)
			if err != nil {
				t.Fatalf("BashInit(%q, %q) error: %v", prog, key, err)
			}
			cmd := exec.Command(bash, "--norc", "-c", script)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("BashInit(%q, %q) does not evaluate in a non-interactive bash: %v\n%s", prog, key, err, out)
			}
			if len(out) > 0 {
				t.Errorf("BashInit(%q, %q) should not print a message:\n%s", prog, key, out)
			}
		}
	}
}

func TestBashInitFlowControlKeyDisablesIXON(t *testing.T) {
	for _, key := range []string{"ctrl-s", "ctrl-q", "CTRL-S", `\C-s`, `\C-Q`} {
		got, err := BashInit("wtr", "/bin/wtr", key)
		if err != nil {
			t.Fatalf("BashInit(%q) error: %v", key, err)
		}
		if !strings.Contains(got, "stty -ixon 2>/dev/null") {
			t.Errorf("BashInit(%q) should turn IXON off:\n%s", key, got)
		}
	}
}

func TestBashInitErrors(t *testing.T) {
	for _, tt := range []struct{ bin, key string }{
		{"/bin/wtr", "bogus"},
		{"/bin/wtr", ""},
		{"", "ctrl-s"},
	} {
		if _, err := BashInit("wtr", tt.bin, tt.key); err == nil {
			t.Errorf("BashInit(%q, %q) should fail", tt.bin, tt.key)
		}
	}
}

// TestBashInitParsesAsBash runs the generated script through the real bash,
// when one is installed: eval-ability is the whole point of `init bash`.
func TestBashInitParsesAsBash(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	for _, prog := range []string{"wtr", "what-to-run"} {
		for _, key := range []string{"ctrl-s", "alt-x", `\e[15~`} {
			script, err := BashInit(prog, "/bin/wtr", key)
			if err != nil {
				t.Fatalf("BashInit(%q, %q) error: %v", prog, key, err)
			}
			cmd := exec.Command(bash, "--norc", "-n")
			cmd.Stdin = strings.NewReader(script)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("bash -n rejects the script for %s %q: %v\n%s", prog, key, err, out)
			}
		}
	}
}

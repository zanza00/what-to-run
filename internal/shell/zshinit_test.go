package shell

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
	got, err := ZshInit("wtr", "/opt/wtr/wtr", "ctrl-g")
	if err != nil {
		t.Fatalf("ZshInit() error: %v", err)
	}
	for _, want := range []string{
		"# wtr: zsh integration",
		"zle -N wtr-widget _wtr_widget",
		"bindkey '^G' wtr-widget",
		`typeset -g WTR_BIN="${WTR_BIN:-/opt/wtr/wtr}"`,
		"out=$(\"$WTR_BIN\" --inline --shell posix ${=WTR_FLAGS} < /dev/tty)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ZshInit() script does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "stty -ixon") {
		t.Errorf("binding Ctrl-G must not touch IXON:\n%s", got)
	}
}

// TestZshInitNamesTheInvokedProgram pins the mnemonic rule: the script keeps
// the name it was generated with, so the integration is self-explaining in
// the rc file.
func TestZshInitNamesTheInvokedProgram(t *testing.T) {
	tests := []struct {
		prog string
		want []string
	}{
		{
			prog: "task-picker",
			want: []string{
				"# task-picker: zsh integration, Ctrl-G opens the inline picker.",
				`eval "$(task-picker init zsh --key Ctrl-G)"`,
			},
		},
		{prog: "wtr", want: []string{"# wtr: zsh integration"}},
		{prog: "", want: []string{"# wtr: zsh integration"}},
	}
	for _, tt := range tests {
		got, err := ZshInit(tt.prog, "/bin/wtr", "ctrl-g")
		if err != nil {
			t.Fatalf("ZshInit(%q) error: %v", tt.prog, err)
		}
		for _, want := range tt.want {
			if !strings.Contains(got, want) {
				t.Errorf("ZshInit(%q) script does not contain %q:\n%s", tt.prog, want, got)
			}
		}
	}
}

// TestZshInitIsQuiet pins that evaluating the script prints nothing: the
// integration lives in the rc file, and a greeting at every new shell is
// noise.
func TestZshInitIsQuiet(t *testing.T) {
	for _, prog := range []string{"wtr", "task-picker", ""} {
		for _, key := range []string{"ctrl-s", "ctrl-g"} {
			got, err := ZshInit(prog, "/bin/wtr", key)
			if err != nil {
				t.Fatalf("ZshInit(%q, %q) error: %v", prog, key, err)
			}
			if strings.Contains(got, `print "`) {
				t.Errorf("ZshInit(%q, %q) should not print a message:\n%s", prog, key, got)
			}
		}
	}
}

func TestZshInitFlowControlKeyDisablesIXON(t *testing.T) {
	for _, key := range []string{"ctrl-s", "ctrl-q", "^s", "^Q"} {
		got, err := ZshInit("wtr", "/bin/wtr", key)
		if err != nil {
			t.Fatalf("ZshInit(%q) error: %v", key, err)
		}
		if !strings.Contains(got, "stty -ixon 2>/dev/null") {
			t.Errorf("ZshInit(%q) should turn IXON off:\n%s", key, got)
		}
	}
}

// TestDefaultKeyLeavesFlowControlAlone pins why the default is not Ctrl-S or
// Ctrl-Q: those are the terminal's XOFF/XON keys, and binding one makes the
// generated zsh and bash scripts run stty -ixon.
func TestDefaultKeyLeavesFlowControlAlone(t *testing.T) {
	for _, tt := range []struct {
		name string
		init func(prog, bin, key string) (string, error)
	}{
		{"zsh", ZshInit},
		{"bash", BashInit},
	} {
		script, err := tt.init("wtr", "/bin/wtr", DefaultKey)
		if err != nil {
			t.Fatalf("%s Init(%q) error: %v", tt.name, DefaultKey, err)
		}
		if strings.Contains(script, "stty -ixon") {
			t.Errorf("%s script for the default key must not touch IXON:\n%s", tt.name, script)
		}
	}
}

func TestZshInitErrors(t *testing.T) {
	for _, tt := range []struct{ bin, key string }{
		{"/bin/wtr", "bogus"},
		{"/bin/wtr", ""},
		{"", "ctrl-s"},
	} {
		if _, err := ZshInit("wtr", tt.bin, tt.key); err == nil {
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
	for _, prog := range []string{"wtr", "task-picker"} {
		for _, key := range []string{"ctrl-s", "alt-x", "^[[15~"} {
			script, err := ZshInit(prog, "/bin/wtr", key)
			if err != nil {
				t.Fatalf("ZshInit(%q, %q) error: %v", prog, key, err)
			}
			cmd := exec.Command(zsh, "-n")
			cmd.Stdin = strings.NewReader(script)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("zsh -n rejects the script for %s %q: %v\n%s", prog, key, err, out)
			}
		}
	}
}

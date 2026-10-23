package tui

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParsePowerShellBindKey(t *testing.T) {
	tests := []struct {
		in    string
		chord string
		name  string
	}{
		{"ctrl-s", "Ctrl+s", "Ctrl-S"},
		{"CTRL-G", "Ctrl+g", "Ctrl-G"},
		{"ctrl-q", "Ctrl+q", "Ctrl-Q"},
		{"alt-x", "Alt+x", "Alt-x"},
		{"alt-X", "Alt+X", "Alt-X"},
		{"alt-1", "Alt+1", "Alt-1"},
		{"  ctrl-t  ", "Ctrl+t", "Ctrl-T"},
	}
	for _, tt := range tests {
		chord, name, err := parsePowerShellBindKey(tt.in)
		if err != nil {
			t.Errorf("parsePowerShellBindKey(%q) error: %v", tt.in, err)
			continue
		}
		if chord != tt.chord || name != tt.name {
			t.Errorf("parsePowerShellBindKey(%q) = %q, %q; want %q, %q", tt.in, chord, name, tt.chord, tt.name)
		}
	}
}

func TestParsePowerShellBindKeyInvalid(t *testing.T) {
	for _, in := range []string{"", "s", "ctrl-", "ctrl-ab", "ctrl-1", "alt-", "alt-xy", "shift-x", `\`, "^[x", `\e[15~`} {
		if _, _, err := parsePowerShellBindKey(in); err == nil {
			t.Errorf("parsePowerShellBindKey(%q) should fail", in)
		}
	}
}

// TestParsePowerShellBindKeyExplainsRawSequences pins the hint in the error:
// a key copied from another shell's widget is the likely mistake, and
// PowerShell cannot take a raw sequence at all.
func TestParsePowerShellBindKeyExplainsRawSequences(t *testing.T) {
	for _, in := range []string{"", "shift-x", "^[x", `\e[15~`} {
		_, _, err := parsePowerShellBindKey(in)
		if err == nil {
			t.Fatalf("parsePowerShellBindKey(%q) should fail", in)
		}
		if !strings.Contains(err.Error(), "not raw sequences") ||
			!strings.Contains(err.Error(), "ctrl-s or alt-x") {
			t.Errorf("parsePowerShellBindKey(%q) error %q should point at key names", in, err)
		}
	}
}

func TestPowerShellInitScript(t *testing.T) {
	got, err := PowerShellInit("wtr", `C:\Users\O'Brien\wtr.exe`, "ctrl-g")
	if err != nil {
		t.Fatalf("PowerShellInit() error: %v", err)
	}
	for _, want := range []string{
		"# wtr: PowerShell integration, Ctrl-G opens the inline picker.",
		`$script:WtrBin = if ($env:WTR_BIN) { $env:WTR_BIN } else { 'C:\Users\O''Brien\wtr.exe' }`,
		`$result = @(& $script:WtrBin --inline --shell powershell @flags)`,
		"[Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()",
		"[Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)",
		"[Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, $result[1])",
		"[Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine()",
		"[Microsoft.PowerShell.PSConsoleReadLine]::Insert($result[1])",
		"Set-PSReadLineKeyHandler -Chord 'Ctrl+g' -ScriptBlock { Invoke-WtrWidget }",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PowerShellInit() script does not contain %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"/dev/tty", "stty", "export "} {
		if strings.Contains(got, forbidden) {
			t.Errorf("PowerShellInit() script should not contain %q:\n%s", forbidden, got)
		}
	}
}

// TestPowerShellInitNamesTheInvokedProgram pins the mnemonic rule: the script
// keeps the name it was generated with, so the integration is self-explaining
// in the profile.
func TestPowerShellInitNamesTheInvokedProgram(t *testing.T) {
	tests := []struct {
		prog string
		want []string
	}{
		{
			prog: "what-to-run",
			want: []string{
				"# what-to-run: PowerShell integration, Ctrl-G opens the inline picker.",
				"#   what-to-run init powershell | Out-String | Invoke-Expression",
			},
		},
		{prog: "wtr", want: []string{"# wtr: PowerShell integration"}},
		{prog: "", want: []string{"# wtr: PowerShell integration"}},
	}
	for _, tt := range tests {
		got, err := PowerShellInit(tt.prog, `C:\Tools\wtr.exe`, "ctrl-g")
		if err != nil {
			t.Fatalf("PowerShellInit(%q) error: %v", tt.prog, err)
		}
		for _, want := range tt.want {
			if !strings.Contains(got, want) {
				t.Errorf("PowerShellInit(%q) script does not contain %q:\n%s", tt.prog, want, got)
			}
		}
	}
}

func TestPowerShellInitErrors(t *testing.T) {
	for _, tt := range []struct{ bin, key string }{
		{`C:\Tools\wtr.exe`, "bogus"},
		{`C:\Tools\wtr.exe`, ""},
		{"", "ctrl-s"},
	} {
		if _, err := PowerShellInit("wtr", tt.bin, tt.key); err == nil {
			t.Errorf("PowerShellInit(%q, %q) should fail", tt.bin, tt.key)
		}
	}
}

// TestPowerShellInitParsesAsPowerShell runs the generated script through the
// real PowerShell parser, without executing it, when one is installed: an
// eval-able script is the whole point of `init powershell`.
func TestPowerShellInitParsesAsPowerShell(t *testing.T) {
	shell, err := exec.LookPath("pwsh")
	if err != nil {
		shell, err = exec.LookPath("powershell")
	}
	if err != nil {
		t.Skip("neither pwsh nor powershell installed")
	}
	const parser = `$tokens = $null; $errors = $null
[System.Management.Automation.Language.Parser]::ParseInput([Console]::In.ReadToEnd(), [ref]$tokens, [ref]$errors) | Out-Null
if ($errors.Count -gt 0) { $errors | ForEach-Object { [Console]::Error.WriteLine($_.Message) }; exit 1 }
`
	for _, prog := range []string{"wtr", "what-to-run"} {
		for _, key := range []string{"ctrl-s", "alt-x", "alt-1"} {
			script, err := PowerShellInit(prog, `C:\Tools\wtr.exe`, key)
			if err != nil {
				t.Fatalf("PowerShellInit(%q, %q) error: %v", prog, key, err)
			}
			cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", parser)
			cmd.Stdin = strings.NewReader(script)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s rejects the script for %s %q: %v\n%s", shell, prog, key, err, out)
			}
		}
	}
}

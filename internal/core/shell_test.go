package core

import (
	"reflect"
	"testing"
)

func TestParseShellDialect(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want ShellDialect
	}{
		{"empty is the platform default", "", DefaultShellDialect()},
		{"auto is the platform default", "auto", DefaultShellDialect()},
		{"auto is case-insensitive", "AUTO", DefaultShellDialect()},
		{"posix", "posix", ShellPOSIX},
		{"sh", "sh", ShellPOSIX},
		{"bash", "bash", ShellPOSIX},
		{"zsh", "zsh", ShellPOSIX},
		{"powershell", "powershell", ShellPowerShell},
		{"pwsh", "pwsh", ShellPowerShell},
		{"ps", "ps", ShellPowerShell},
		{"cmd", "cmd", ShellCmd},
		{"cmd.exe", "cmd.exe", ShellCmd},
		{"surrounding spaces are ignored", "  cmd  ", ShellCmd},
		{"names are case-insensitive", "PowerShell", ShellPowerShell},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseShellDialect(tt.in)
			if err != nil {
				t.Fatalf("ParseShellDialect(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseShellDialect(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseShellDialectUnknown(t *testing.T) {
	for _, in := range []string{"fish", "cmd.exe /c", "nu", "power"} {
		if _, err := ParseShellDialect(in); err == nil {
			t.Errorf("ParseShellDialect(%q) should fail", in)
		}
	}
}

func TestShellDialectString(t *testing.T) {
	tests := []struct {
		shell ShellDialect
		want  string
	}{
		{ShellPOSIX, "posix"},
		{ShellPowerShell, "powershell"},
		{ShellCmd, "cmd"},
	}
	for _, tt := range tests {
		if got := tt.shell.String(); got != tt.want {
			t.Errorf("ShellDialect(%d).String() = %q, want %q", int(tt.shell), got, tt.want)
		}
	}
}

func TestQuoteArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain word", "make", "make"},
		{"path and flag characters", "./bin/tool", "./bin/tool"},
		{"safe punctuation", "a+b=c,d@e%f:g", "a+b=c,d@e%f:g"},
		{"empty string", "", "''"},
		{"space", "a b", "'a b'"},
		{"single quote", "it's", `'it'\''s'`},
		{"double quote", `say "hi"`, `'say "hi"'`},
		{"shell metacharacters", "$HOME;rm -rf /", "'$HOME;rm -rf /'"},
		{"backslash", `C:\tmp`, `'C:\tmp'`},
		{"leading dash is not special", "--flag", "--flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteArg(tt.in); got != tt.want {
				t.Errorf("quoteArg(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestQuotePowerShellArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain word", "make", "make"},
		{"path with spaces is quoted", `C:\Program Files`, `'C:\Program Files'`},
		{"path without spaces passes through", `C:\tmp\tool.exe`, `C:\tmp\tool.exe`},
		{"empty string", "", "''"},
		{"space", "a b", "'a b'"},
		{"single quote is doubled", "it's", "'it''s'"},
		{"double quote", `say "hi"`, `'say "hi"'`},
		{"variable", "$env:HOME", "'$env:HOME'"},
		{"semicolon", "a;b", "'a;b'"},
		{"pipe", "a|b", "'a|b'"},
		{"comma is an array separator", "a,b", "'a,b'"},
		{"splat", "@args", "'@args'"},
		{"backslash and punctuation pass through", `a=b+c-d:e_f.g/h\i`, `a=b+c-d:e_f.g/h\i`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quotePowerShellArg(tt.in); got != tt.want {
				t.Errorf("quotePowerShellArg(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestQuoteCmdArg pins the best-effort output of the cmd dialect, not a
// round-trip. cmd expands %VAR% and !VAR! even inside double quotes.
func TestQuoteCmdArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain word", "make", "make"},
		{"path without spaces passes through", `C:\tmp\tool.exe`, `C:\tmp\tool.exe`},
		{"empty string", "", `""`},
		{"space", "a b", `"a b"`},
		{"tab", "a\tb", "\"a\tb\""},
		{"single quote is not special", "it's", "it's"},
		{"embedded double quote", `a"b`, `"a\"b"`},
		{"backslash before a quote is doubled", `a\"b`, `"a\\\"b"`},
		{"trailing backslash is doubled before the closing quote", `a b\`, `"a b\\"`},
		{"path with spaces keeps its backslashes", `C:\Program Files\tool.exe`, `"C:\Program Files\tool.exe"`},
		{"ampersand", "a&b", `"a&b"`},
		{"pipe", "a|b", `"a|b"`},
		{"redirection", "a>b", `"a>b"`},
		{"caret", "a^b", `"a^b"`},
		{"parentheses", "a(b)", `"a(b)"`},
		{"percent is expanded by cmd despite quotes", "%PATH%", `"%PATH%"`},
		{"bang", "a!b", `"a!b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteCmdArg(tt.in); got != tt.want {
				t.Errorf("quoteCmdArg(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestShellCommandFor(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		command string
		want    []string
	}{
		{"windows runs through cmd", "windows", "npm run build", []string{"cmd", "/c", "npm run build"}},
		{"linux runs through sh", "linux", "npm run build", []string{"sh", "-c", "npm run build"}},
		{"darwin runs through sh", "darwin", "npm run build", []string{"sh", "-c", "npm run build"}},
		{"empty command", "windows", "", []string{"cmd", "/c", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellCommandFor(tt.goos, tt.command); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shellCommandFor(%q, %q) = %v, want %v", tt.goos, tt.command, got, tt.want)
			}
		})
	}
}

func TestCmdLineFor(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"plain command", "npm run build", `/d /s /c "npm run build"`},
		{"quoted path with spaces", `"C:\Program Files\tool.exe" --flag`, `/d /s /c ""C:\Program Files\tool.exe" --flag"`},
		{"quoted argument", `echo "hi"`, `/d /s /c "echo "hi""`},
		{"trailing backslash", `dir C:\`, `/d /s /c "dir C:\"`},
		{"empty command", "", `/d /s /c ""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmdLineFor(tt.command); got != tt.want {
				t.Errorf("cmdLineFor(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestCommandLineFor(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		shell ShellDialect
		want  string
	}{
		{"posix quotes an argument with a space and a quote", []string{"echo", "it's a test"}, ShellPOSIX, `echo 'it'\''s a test'`},
		{"powershell doubles the quote", []string{"echo", "it's a test"}, ShellPowerShell, `echo 'it''s a test'`},
		{"cmd groups the argument in double quotes", []string{"echo", "it's a test"}, ShellCmd, `echo "it's a test"`},
		{"posix path with spaces", []string{`C:\Program Files\tool.exe`, "--flag"}, ShellPOSIX, `'C:\Program Files\tool.exe' --flag`},
		{"powershell path with spaces takes the call operator", []string{`C:\Program Files\tool.exe`, "--flag"}, ShellPowerShell, `& 'C:\Program Files\tool.exe' --flag`},
		{"cmd path with spaces", []string{`C:\Program Files\tool.exe`, "--flag"}, ShellCmd, `"C:\Program Files\tool.exe" --flag`},
		{"empty argv", nil, ShellPOSIX, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Source: "rc", Name: "t", Args: tt.args}
			if got := task.CommandLineFor(tt.shell); got != tt.want {
				t.Errorf("CommandLineFor(%s) = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}

package tui

import (
	"reflect"
	"testing"
)

func TestParseShell(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want shellDialect
	}{
		{"empty is the platform default", "", defaultShell()},
		{"auto is the platform default", "auto", defaultShell()},
		{"auto is case-insensitive", "AUTO", defaultShell()},
		{"posix", "posix", shellPOSIX},
		{"sh", "sh", shellPOSIX},
		{"bash", "bash", shellPOSIX},
		{"zsh", "zsh", shellPOSIX},
		{"powershell", "powershell", shellPowerShell},
		{"pwsh", "pwsh", shellPowerShell},
		{"ps", "ps", shellPowerShell},
		{"cmd", "cmd", shellCmd},
		{"cmd.exe", "cmd.exe", shellCmd},
		{"surrounding spaces are ignored", "  cmd  ", shellCmd},
		{"names are case-insensitive", "PowerShell", shellPowerShell},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShell(tt.in)
			if err != nil {
				t.Fatalf("parseShell(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("parseShell(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseShellUnknown(t *testing.T) {
	for _, in := range []string{"fish", "cmd.exe /c", "nu", "power"} {
		if _, err := parseShell(in); err == nil {
			t.Errorf("parseShell(%q) should fail", in)
		}
	}
}

func TestShellDialectString(t *testing.T) {
	tests := []struct {
		shell shellDialect
		want  string
	}{
		{shellPOSIX, "posix"},
		{shellPowerShell, "powershell"},
		{shellCmd, "cmd"},
	}
	for _, tt := range tests {
		if got := tt.shell.String(); got != tt.want {
			t.Errorf("shellDialect(%d).String() = %q, want %q", int(tt.shell), got, tt.want)
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
		{"percent variable", "%PATH%", `"%PATH%"`},
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

func TestCommandLineFor(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		shell shellDialect
		want  string
	}{
		{"posix quotes an argument with a space and a quote", []string{"echo", "it's a test"}, shellPOSIX, `echo 'it'\''s a test'`},
		{"powershell doubles the quote", []string{"echo", "it's a test"}, shellPowerShell, `echo 'it''s a test'`},
		{"cmd groups the argument in double quotes", []string{"echo", "it's a test"}, shellCmd, `echo "it's a test"`},
		{"posix path with spaces", []string{`C:\Program Files\tool.exe`, "--flag"}, shellPOSIX, `'C:\Program Files\tool.exe' --flag`},
		{"powershell path with spaces", []string{`C:\Program Files\tool.exe`, "--flag"}, shellPowerShell, `'C:\Program Files\tool.exe' --flag`},
		{"cmd path with spaces", []string{`C:\Program Files\tool.exe`, "--flag"}, shellCmd, `"C:\Program Files\tool.exe" --flag`},
		{"empty argv", nil, shellPOSIX, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Source: "rc", Name: "t", Args: tt.args}
			if got := task.commandLineFor(tt.shell); got != tt.want {
				t.Errorf("commandLineFor(%s) = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}

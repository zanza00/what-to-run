// The shell dialect of the command lines the picker shows.
//
// A task runs as an argv, but the picker also displays one command line per
// task and the inline widget hands that line to the shell prompt. How an
// argument must be quoted depends on the shell that will read it back:
// POSIX sh/bash/zsh, PowerShell, or cmd.exe.
package core

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
)

// ShellDialect is the shell syntax command lines are quoted for.
type ShellDialect int

const (
	// ShellPOSIX is sh, bash and zsh: single quotes make everything literal.
	ShellPOSIX ShellDialect = iota
	// ShellPowerShell is Windows PowerShell and PowerShell 7: single-quoted
	// strings are literal and an embedded quote is doubled.
	ShellPowerShell
	// ShellCmd is cmd.exe: double quotes group an argument.
	ShellCmd
)

// String returns the name used by the --shell flag.
func (s ShellDialect) String() string {
	switch s {
	case ShellPowerShell:
		return "powershell"
	case ShellCmd:
		return "cmd"
	default:
		return "posix"
	}
}

// DefaultShellDialect is the dialect used when --shell is not given:
// PowerShell on Windows, whose prompt is where an inline widget places the
// line, and POSIX everywhere else.
func DefaultShellDialect() ShellDialect {
	if runtime.GOOS == "windows" {
		return ShellPowerShell
	}
	return ShellPOSIX
}

// ParseShellDialect maps a --shell value to a dialect: empty and "auto" mean
// the platform default, and the shell names people actually type are accepted
// alongside the three dialect names.
func ParseShellDialect(name string) (ShellDialect, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "auto":
		return DefaultShellDialect(), nil
	case "posix", "sh", "bash", "zsh":
		return ShellPOSIX, nil
	case "powershell", "pwsh", "ps":
		return ShellPowerShell, nil
	case "cmd", "cmd.exe":
		return ShellCmd, nil
	}
	return 0, fmt.Errorf("unknown shell %q: use auto, posix, powershell or cmd", name)
}

// shellCommand returns the argv that runs command through the platform's
// command interpreter: sh -c on Unix, cmd /c on Windows. Windows has no sh
// unless Git Bash or MSYS is installed, and rc tasks must run on a stock
// install.
func shellCommand(command string) []string {
	return shellCommandFor(runtime.GOOS, command)
}

// shellCommandFor is shellCommand with an explicit platform, so tests can pin
// both behaviours on any machine.
func shellCommandFor(goos, command string) []string {
	if goos == "windows" {
		return []string{"cmd", "/c", command}
	}
	return []string{"sh", "-c", command}
}

// cmdLineFor is the raw command line cmd.exe needs for command: /d skips
// AutoRun, /s strips only the outermost quotes, so quotes inside command
// reach the child unchanged.
func cmdLineFor(command string) string {
	return `/d /s /c "` + command + `"`
}

// quoteArgFor quotes arg the way shell reads it back. POSIX and PowerShell
// round-trip exactly; the cmd dialect is best-effort, because cmd expands
// %VAR% and !VAR! even inside double quotes and a newline cannot be
// represented in one command line, so callers must not rely on a lossless
// round-trip for those inputs.
func quoteArgFor(shell ShellDialect, arg string) string {
	switch shell {
	case ShellPowerShell:
		return quotePowerShellArg(arg)
	case ShellCmd:
		return quoteCmdArg(arg)
	default:
		return quoteArg(arg)
	}
}

// quoteArg quotes arg the way a POSIX shell reads it back unchanged: plain
// words pass through, anything else is single-quoted.
func quoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, r := range arg {
		if r == '_' || r == '.' || r == '/' || r == '-' || r == '@' || r == '%' ||
			r == ':' || r == '=' || r == '+' || r == ',' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
	}
	return arg
}

// quotePowerShellArg quotes arg for PowerShell: plain words pass through,
// anything special is wrapped in single quotes, where only an embedded single
// quote is special and is doubled. Dollar signs, semicolons, pipes, commas,
// parentheses and the splat @ stay literal that way.
func quotePowerShellArg(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, r := range arg {
		if r == '_' || r == '.' || r == '/' || r == '\\' || r == '-' || r == ':' ||
			r == '=' || r == '+' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "'" + strings.ReplaceAll(arg, "'", "''") + "'"
	}
	return arg
}

// quoteCmdArg quotes arg for cmd.exe: a word without whitespace or cmd
// metacharacters passes through, anything else is double-quoted and its
// quotes are escaped the way the C runtime parses them back (backslash
// doubling before a quote and before the closing quote).
//
// The result is best-effort: cmd expands %VAR% (and !VAR! with delayed
// expansion) even inside double quotes, and a newline cannot be represented
// in one command line, so those inputs do not round-trip.
func quoteCmdArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\"&|<>^()%!") {
		return arg
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			slashes++
			b.WriteRune(r)
		case '"':
			b.WriteString(strings.Repeat(`\`, slashes+1))
			b.WriteRune(r)
			slashes = 0
		default:
			slashes = 0
			b.WriteRune(r)
		}
	}
	if slashes > 0 {
		b.WriteString(strings.Repeat(`\`, slashes))
	}
	b.WriteByte('"')
	return b.String()
}

// Command wtr is a TUI for running the tasks declared in a project's files
// (package.json, Makefile, Taskfile.yml, justfile, Makefile.toml,
// .cargo/config, devbox.json, composer.json, deno.json, mise.toml, and the rc
// file). The invoked name is the one shown in usage, messages and the picker
// title.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zanza00/what-to-run/internal/shell"
	"github.com/zanza00/what-to-run/internal/tui"
)

// prog is the name the program was invoked as (wtr, or whatever it was
// renamed to) and identifies itself with.
var prog = progName(os.Args[0])

// version is the release version, injected at build time with
// `-X main.version` (see Taskfile.yml and .goreleaser.yaml); local builds
// report "dev".
var version = "dev"

func main() {
	// A subcommand comes first; everything else is the picker's own flags.
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInit(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
			os.Exit(1)
		}
		return
	}

	inline := flag.Bool("inline", false,
		"render between the current line and the bottom of the screen instead\n"+
			"of taking over the whole screen (for shell widgets that launch the\n"+
			"binary right below the prompt, see '"+prog+" init zsh',\n"+
			"'"+prog+" init bash' and '"+prog+" init powershell')")
	debugArea := flag.Bool("debug-area", false,
		"draw a border around the area managed in inline mode (development aid)")
	theme := flag.String("theme", "",
		"theme to render the picker with (\"classic\", \"nord\", \"dracula\", …);\n"+
			"ctrl+t opens the theme picker")
	shell := flag.String("shell", "",
		"shell dialect the command lines are quoted for: \"posix\", \"powershell\"\n"+
			"or \"cmd\"; empty or \"auto\" means the platform default (PowerShell on\n"+
			"Windows, POSIX elsewhere); cmd expands %VAR% and !VAR! inside quotes")
	listThemes := flag.Bool("list-themes", false,
		"list the available themes and exit")
	showVersion := flag.Bool("version", false,
		"print the version and exit")
	preview := flag.Bool("preview", false,
		"print one static frame of the picker to stdout and exit (theming\n"+
			"aid: compare themes side by side, e.g. with --theme)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags]\n"+
			"       %s init zsh|bash|powershell [--key combo]\n\nFlags:\n", prog, prog)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(versionLine(prog, version))
		return
	}

	if *listThemes {
		tui.ListThemes(os.Stdout)
		return
	}

	if *preview {
		if err := tui.Preview(os.Stdout, tui.Options{Prog: prog, Theme: *theme, Shell: *shell}, 80, 24); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
			os.Exit(1)
		}
		return
	}

	err := tui.Run(tui.Options{Prog: prog, Inline: *inline, DebugArea: *debugArea, Theme: *theme, Shell: *shell})
	if err == nil {
		return
	}

	// A task launched from the inline picker failed: exit with the task's
	// own code and stay quiet, the task already reported its error.
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) {
		code := exit.ExitCode()
		if code < 1 {
			code = 1
		}
		os.Exit(code)
	}

	fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
	os.Exit(1)
}

// runInit implements the init subcommand, `<prog> init zsh|bash|powershell
// [--key combo]`: it prints to w the shell integration for the running binary,
// ready to be evaluated by an interactive shell:
//
//	eval "$(wtr init zsh --key ctrl-s)"
//	eval "$(wtr init bash)"
//	wtr init powershell | Out-String | Invoke-Expression
func runInit(args []string, w io.Writer) error {
	fs := flag.NewFlagSet(prog+" init", flag.ExitOnError)
	key := fs.String("key", shell.DefaultKey,
		"combo that opens the picker: a name like ctrl-s or alt-x, or, for zsh\n"+
			"and bash, a raw sequence as they bind keys (^[x for zsh, \\e[15~\n"+
			"for bash); PowerShell takes key names only")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s init zsh|bash|powershell [--key combo]\n\n"+
			"Print the shell integration to stdout, for eval:\n\n"+
			"  eval \"$(%s init zsh --key ctrl-s)\"\n"+
			"  eval \"$(%s init bash --key ctrl-s)\"\n"+
			"  %s init powershell | Out-String | Invoke-Expression\n\nFlags:\n", prog, prog, prog, prog)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("missing shell argument: use '%s init zsh|bash|powershell [--key combo]'", prog)
	}
	sh, tail := rest[0], rest[1:]
	init := shell.ZshInit
	switch sh {
	case "zsh":
	case "bash":
		init = shell.BashInit
	case "powershell", "pwsh":
		init = shell.PowerShellInit
	default:
		return fmt.Errorf("unsupported shell %q: only zsh, bash and powershell are supported", sh)
	}
	// flag stops at the first positional argument, so a flag written after
	// the shell name ("init zsh --key ctrl-g") needs a second pass.
	if len(tail) > 0 {
		if err := fs.Parse(tail); err != nil {
			return err
		}
		if extra := fs.Args(); len(extra) > 0 {
			return fmt.Errorf("unexpected argument %q", extra[0])
		}
	}

	script, err := init(prog, widgetBin(os.Args[0]), *key)
	if err != nil {
		return err
	}
	fmt.Fprint(w, script)
	return nil
}

// progName is the name the program shows as its own: the basename of the
// command it was invoked as, with a fallback for the odd exec without argv[0].
func progName(arg0 string) string {
	return progNameFor(runtime.GOOS, arg0)
}

// progNameFor is progName for the given platform. On Windows argv[0] may use
// either separator, and the launcher usually appends the .exe suffix: neither
// belongs in the name the program shows, so both are dropped there. Elsewhere
// a file literally named wtr.exe keeps its suffix.
func progNameFor(goos, arg0 string) string {
	if arg0 == "" {
		return "wtr"
	}
	if goos != "windows" {
		return filepath.Base(arg0)
	}
	name := path.Base(strings.ReplaceAll(arg0, `\`, "/"))
	if ext := path.Ext(name); strings.EqualFold(ext, ".exe") {
		name = strings.TrimSuffix(name, ext)
	}
	if name == "" {
		return "wtr"
	}
	return name
}

// versionLine is what --version prints: the invoked name and the release
// version, so a copy-pasted bug report says both.
func versionLine(prog, ver string) string {
	return prog + " " + ver
}

// widgetBin is the command the generated zsh widget launches: the binary as
// the user invoked it, so the widget keeps calling it by the same name (and
// the picker keeps showing that name). A bare name stays bare and is
// resolved through PATH when the key is pressed; a path is made absolute.
func widgetBin(arg0 string) string {
	switch {
	case arg0 == "":
		if exe, err := os.Executable(); err == nil {
			return exe
		}
		return "wtr"
	case !hasPathSeparator(runtime.GOOS, arg0):
		return arg0
	default:
		if abs, err := filepath.Abs(arg0); err == nil {
			return abs
		}
		return arg0
	}
}

// hasPathSeparator reports whether arg0 names a path rather than a bare
// command: '/' separates everywhere, '\' only on Windows, where it cannot be
// part of a file name.
func hasPathSeparator(goos, arg0 string) bool {
	if strings.ContainsRune(arg0, '/') {
		return true
	}
	return goos == "windows" && strings.ContainsRune(arg0, '\\')
}

// Command wtr is a TUI for running the scripts declared in a project's
// package.json, Makefile, or Cargo.toml (cargo-make). The same program is
// also built as what-to-run, the explicit name for shell rc files:
// the invoked name is the one shown in usage, messages and the picker title.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zanza00/what-to-run/internal/tui"
)

// prog is the name the program was invoked as (wtr, what-to-run, or
// whatever it was renamed to) and identifies itself with.
var prog = progName(os.Args[0])

func main() {
	// A subcommand comes first; everything else is the picker's own flags.
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
			os.Exit(1)
		}
		return
	}

	inline := flag.Bool("inline", false,
		"render between the current line and the bottom of the screen instead\n"+
			"of taking over the whole screen (for shell widgets that launch the\n"+
			"binary right below the prompt, see '"+prog+" init zsh')")
	debugArea := flag.Bool("debug-area", false,
		"draw a border around the area managed in inline mode (development aid)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags]\n"+
			"       %s init zsh [--key combo]\n\nFlags:\n", prog, prog)
		flag.PrintDefaults()
	}
	flag.Parse()

	err := tui.Run(tui.Options{Prog: prog, Inline: *inline, DebugArea: *debugArea})
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

// runInit implements the init subcommand, `<prog> init zsh [--key combo]`: it
// prints to stdout the zsh integration for the running binary, ready to be
// evaluated by an interactive shell:
//
//	eval "$(what-to-run init zsh --key ctrl-s)"
func runInit(args []string) error {
	fs := flag.NewFlagSet(prog+" init", flag.ExitOnError)
	key := fs.String("key", tui.DefaultKey,
		"combo that opens the picker: a name like ctrl-s or alt-x, or a raw\n"+
			"zsh key sequence as bindkey takes it, like ^[x")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s init zsh [--key combo]\n\n"+
			"Print the zsh integration to stdout, for eval:\n\n"+
			"  eval \"$(%s init zsh --key ctrl-s)\"\n\nFlags:\n", prog, prog)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("missing shell argument: use '%s init zsh [--key combo]'", prog)
	}
	shell, tail := rest[0], rest[1:]
	if shell != "zsh" {
		return fmt.Errorf("unsupported shell %q: only zsh is supported for now", shell)
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

	script, err := tui.ZshInit(prog, widgetBin(os.Args[0]), *key)
	if err != nil {
		return err
	}
	fmt.Print(script)
	return nil
}

// progName is the name the program shows as its own: the basename of the
// command it was invoked as, with a fallback for the odd exec without argv[0].
func progName(arg0 string) string {
	if arg0 == "" {
		return "wtr"
	}
	return filepath.Base(arg0)
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
	case !strings.ContainsRune(arg0, os.PathSeparator):
		return arg0
	default:
		if abs, err := filepath.Abs(arg0); err == nil {
			return abs
		}
		return arg0
	}
}

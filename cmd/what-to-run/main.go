// Command what-to-run is a TUI for running the scripts declared in a
// project's package.json, Makefile, or Cargo.toml (cargo-make).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/zanza00/what-to-run/internal/tui"
)

func main() {
	// A subcommand comes first; everything else is the picker's own flags.
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "what-to-run: %v\n", err)
			os.Exit(1)
		}
		return
	}

	inline := flag.Bool("inline", false,
		"render between the current line and the bottom of the screen instead\n"+
			"of taking over the whole screen (for shell widgets that launch the\n"+
			"binary right below the prompt, see 'what-to-run init zsh')")
	debugArea := flag.Bool("debug-area", false,
		"draw a border around the area managed in inline mode (development aid)")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "usage: what-to-run [flags]\n"+
			"       what-to-run init zsh [--key combo]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	err := tui.Run(tui.Options{Inline: *inline, DebugArea: *debugArea})
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

	fmt.Fprintf(os.Stderr, "what-to-run: %v\n", err)
	os.Exit(1)
}

// runInit implements the init subcommand, `what-to-run init zsh
// [--key combo]`: it prints to stdout the zsh integration for the running
// binary, ready to be evaluated by an interactive shell:
//
//	eval "$(what-to-run init zsh --key ctrl-s)"
func runInit(args []string) error {
	fs := flag.NewFlagSet("what-to-run init", flag.ExitOnError)
	key := fs.String("key", tui.DefaultKey,
		"combo that opens the picker: a name like ctrl-s or alt-x, or a raw\n"+
			"zsh key sequence as bindkey takes it, like ^[x")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "usage: what-to-run init zsh [--key combo]\n\n"+
			"Print the zsh integration to stdout, for eval:\n\n"+
			"  eval \"$(what-to-run init zsh --key ctrl-s)\"\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return errors.New("missing shell argument: use 'what-to-run init zsh [--key combo]'")
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

	bin, err := os.Executable()
	if err != nil {
		// Fall back to a PATH lookup when the widget launches the binary.
		bin = "what-to-run"
	}
	script, err := tui.ZshInit(bin, *key)
	if err != nil {
		return err
	}
	fmt.Print(script)
	return nil
}

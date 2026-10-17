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
	inline := flag.Bool("inline", false,
		"render between the current line and the bottom of the screen instead\n"+
			"of taking over the whole screen (for shell widgets that launch the\n"+
			"binary right below the prompt, see scripts/ctrl-s.zsh)")
	debugArea := flag.Bool("debug-area", false,
		"draw a border around the area managed in inline mode (development aid)")
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

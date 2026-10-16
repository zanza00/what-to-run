// Command what-to-run is a TUI for running the scripts declared in a
// project's package.json, Makefile, or Cargo.toml (cargo-make).
package main

import (
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

	if err := tui.Run(tui.Options{Inline: *inline, DebugArea: *debugArea}); err != nil {
		fmt.Fprintf(os.Stderr, "what-to-run: %v\n", err)
		os.Exit(1)
	}
}

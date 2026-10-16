// Command what-to-run is a TUI for running the scripts declared in a
// project's package.json, Makefile, or Cargo.toml (cargo-make).
package main

import (
	"fmt"
	"os"

	"github.com/zanza00/what-to-run/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "what-to-run: %v\n", err)
		os.Exit(1)
	}
}

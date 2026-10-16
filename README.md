# what-to-run

A TUI for running the scripts declared in a project's `package.json`,
`Makefile`, or `Cargo.toml` (cargo-make).

Built with [Bubble Tea](https://charm.land/bubbletea/v2) and
[Lip Gloss](https://charm.land/lipgloss/v2).

**Current state:** skeleton that renders a hello world. Press `q` (or `esc`,
`ctrl+c`) to quit.

## Requirements

Go 1.27 or later.

## Usage

```sh
go run ./cmd/what-to-run            # full screen
go run ./cmd/what-to-run --inline   # from the line below the prompt down
```

`--inline` renders only between the current cursor line (the shell prompt) and
the bottom of the screen, like `television --inline`; it is meant for shell
widgets that launch the binary right below the prompt. `--debug-area` draws a
border around that area (development aid).

## Development

```sh
make test    # run tests with the race detector
make build   # build the binary into bin/
make run     # go run, without building a binary
make help    # list all targets
```

### Trying the inline UI from the shell

```sh
make build
source scripts/ctrl-s.zsh   # binds Ctrl-S to bin/what-to-run --inline
```

Ctrl-S then opens the picker just below the prompt. The script disables `stty
ixon` because Ctrl-S is the XOFF flow-control character.

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
go run ./cmd/what-to-run
```

## Development

```sh
make test    # run tests with the race detector
make build   # build the binary into bin/
make run     # go run, without building a binary
make help    # list all targets
```

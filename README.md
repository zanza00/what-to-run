# what-to-run

A terminal UI to discover and run the scripts declared in a project's
`package.json`, `Makefile`, or `Cargo.toml` (cargo-make) — so you do not have to
remember the exact task names.

It is a single Go binary built with [Bubble Tea](https://charm.land/bubbletea/v2)
v2 and [Lip Gloss](https://charm.land/lipgloss/v2) v2. It runs either full
screen, or **inline**: anchored between the shell prompt and the bottom of the
screen, with the scrollback above left intact. The inline mode is the same idea
as `television --inline` and is meant for shell widgets that launch the binary
right below the prompt (see [`scripts/ctrl-s.zsh`](scripts/ctrl-s.zsh)).

```
   shell history + prompt      ← never touched (no alternate screen)
   ────────────────────────
   what-to-run          ← rendered from the line right below the
   ╭───────────────────╮          prompt down to the bottom of the screen
   │    Hello, World!  │
   ╰───────────────────╯
   press q to quit
```

## Status

Early skeleton. It currently renders a hello world, quits on `q`, `esc` or
`ctrl+c`, and implements the two rendering modes (full screen and inline,
including the cursor anchoring and restore). Task discovery and execution are
not implemented yet — see [Roadmap](#roadmap).

## Getting started

Requirements: Go 1.27 or later.

```sh
make build                        # build bin/what-to-run
bin/what-to-run            # full screen
bin/what-to-run --inline   # anchored below the prompt
```

Or without building a binary:

```sh
go run ./cmd/what-to-run
```

### Flags

| Flag | Description |
| --- | --- |
| `--inline` | Render between the current cursor line (the prompt) and the bottom of the screen instead of taking over the terminal. Requires an interactive terminal; falls back to full screen with a warning otherwise. |
| `--debug-area` | Draw a border around the whole area managed in inline mode (development aid; ignored in full screen). |

## Features

### Inline mode

Start the binary with `--inline` from a shell widget and the UI fills only the
space below the prompt:

- it asks the terminal for the cursor position (`tea.RequestCursorPosition`) and
  anchors itself on the line right below the prompt;
- it does not use the alternate screen, so the shell history above stays
  visible;
- the area is at least 15 rows (the same value television uses); when fewer rows
  are free below the prompt, the terminal is left to scroll up instead of
  squeezing the UI;
- on quit, the cursor returns to the first row of the block and the block is
  cleared, so the shell redraws its prompt right below the line the user was
  typing on.

### Full screen mode

The default mode: the UI takes over the terminal through the alternate screen
and the previous shell contents are restored on quit.

## Roadmap

- [ ] Discover tasks from `scripts` in `package.json`, targets in `Makefile`,
      and tasks in `Cargo.toml` (cargo-make)
- [ ] Present them in a navigable, filterable list
- [ ] Run the selected task and return to the list when it finishes

## Development

```sh
make test    # run tests with the race detector and coverage
make build   # build the binary into bin/
make run     # go run, without building a binary
make fmt     # gofmt and go vet
make lint    # golangci-lint (requires golangci-lint installed)
make help    # list all targets
```

### Trying the inline UI from the shell

```sh
make build
source scripts/ctrl-s.zsh   # binds Ctrl-S to bin/what-to-run --inline
```

Ctrl-S then opens the UI just below the prompt. The script disables `stty ixon`
because Ctrl-S is the XOFF flow-control character. Extra flags can be passed to
the widget with `WTR_FLAGS` (e.g. `WTR_FLAGS=--debug-area source
scripts/ctrl-s.zsh`), and `WTR_BIN` overrides the binary path.

## Repository layout

```
cmd/what-to-run/   entry point (flag parsing only)
internal/tui/             Bubble Tea model, views, and styles
scripts/ctrl-s.zsh        local zsh harness for the inline mode
```

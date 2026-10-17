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
   ───────────────────────────
   what-to-run          ← rendered from the line right below the
   » type to filter▏              prompt down to the bottom of the screen
   › build      go build ./...      package.json
     test       go test ./...       Makefile
     fmt        Format the code     Cargo.toml
   3 tasks   ↑/↓ select · enter run · esc quit
```

## Status

Task discovery and execution are implemented: the picker lists what it finds,
filters as you type, and runs the selected task. In full screen it runs in
place and returns to the list; in inline mode it quits first and runs the task
at the prompt, propagating its exit code. See [Roadmap](#roadmap) for what is
still open.

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

### Keys

| Key | Action |
| --- | --- |
| typing | filter the list (case-insensitive subsequence of the task names) |
| `↑`/`↓`, `ctrl+p`/`ctrl+n` | move the selection |
| `pgup`/`pgdown`, `home`/`end` | jump in the list |
| `enter` | run the selected task |
| `esc` | clear the filter, or quit when the filter is empty |
| `ctrl+u` | clear the filter |
| `ctrl+c` | quit |

`q` types into the filter; it does not quit.

## Features

### Task sources

| Source | Discovered | Runs as |
| --- | --- | --- |
| `package.json` | `scripts`, sorted by name; the script body is shown as its description | `<pm> run NAME`, with `pm` from the `packageManager` field, a lock file (`bun.lock`, `pnpm-lock.yaml`, `yarn.lock`, `package-lock.json`), or npm |
| `Makefile` (or `makefile`, `GNUmakefile`) | targets, in file order; the comment directly above a target is its description, with the `## name: text` and `## name - text` conventions (`make help` style) cleaned up | `make NAME` |
| `Cargo.toml` | cargo-make `[tasks]`, sorted by name; `description`, or `command` `args` when there is none | `cargo make NAME` |

A source that fails to parse is skipped and reported in the status line; the
tasks from the other sources still show up. With no tasks at all, the picker
exits with an error.

### Running a task

In **full screen**, `enter` runs the task in place: the picker hands the
terminal over to the task (Bubble Tea `tea.ExecProcess`) and takes it back when
the task ends, showing the outcome in the status line (`✓ build (1.2s)`,
`✗ test (0.4s): exit status 1`). The task's output stays in the terminal
scrollback, so it can be reviewed after quitting the picker.

In **inline** mode, `enter` quits the picker first and the task then runs at
the prompt like a hand-typed command, so its output is ordinary shell output in
the normal scrollback; the task's exit code becomes the exit code of
what-to-run. This is also the only way it can work: the inline renderer
cannot reposition its frame after another process scrolls the screen, so the
task must run after the picker has left it.

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

- [x] Discover tasks from `scripts` in `package.json`, targets in `Makefile`,
      and tasks in `Cargo.toml` (cargo-make)
- [x] Present them in a navigable, filterable list
- [x] Run the selected task and return to the list when it finishes
      (full screen; inline mode runs the task at the prompt instead)
- [ ] More sources: `justfile`, `Makefile.toml`
- [ ] `ctrl+r` to refresh the list after editing a task file

## Development

```sh
make test    # run tests with the race detector and coverage
make build   # build the binary into bin/
make run     # go run, without building a binary
make fmt     # gofmt and go vet
make lint    # golangci-lint (requires golangci-lint installed)
make help    # list all targets
```

There are no PTY tests for the inline mode: verify it end-to-end in tmux (zsh
plus the Ctrl-S harness), and check the row math in
`internal/tui/tui_test.go` after any layout change.

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
internal/tui/tui.go       Bubble Tea model, views, and styles
internal/tui/tasks.go     the Task model
internal/tui/sources.go   discovery framework and the source registry
internal/tui/source_*.go  one parser per source (package.json, Makefile, Cargo.toml)
scripts/ctrl-s.zsh        local zsh harness for the inline mode
```

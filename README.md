# what-to-run

A terminal UI to discover and run the scripts declared in a project's
`package.json`, `Makefile`, or `Cargo.toml` (cargo-make) — so you do not have to
remember the exact task names.

It is a single Go binary built with [Bubble Tea](https://charm.land/bubbletea/v2)
v2 and [Lip Gloss](https://charm.land/lipgloss/v2) v2. The same program is
built under two names: `wtr`, for typing at the command line, and
`what-to-run`, the explicit name meant for shell rc files — the name you
invoke is the one shown in usage, messages and the picker title. It runs either
full screen, or **inline**:
anchored between the shell prompt and the bottom of the screen, with the
scrollback above left intact. The inline mode is the same idea as
`television --inline` and is meant for shell widgets that launch the binary
right below the prompt (see [shell integration](#shell-integration-zsh)).

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

Task discovery and execution are implemented: the picker lists what it finds —
each row shows the command line that will run the task (`npm run build`,
`make build`, `cargo make clean`, …), its description and the file it comes
from — filters as you type, and runs the selected task. In full screen it runs in
place and returns to the list; in inline mode it quits first and the task then
runs at the prompt, propagating its exit code. `tab` inserts the chosen command
in the prompt without running it, and an [`init zsh`](#shell-integration-zsh)
subcommand prints the shell widget to evaluate, bound to the key you choose.
An optional [rc file](#rc-file) can
add custom tasks, force a package manager, or disable sources. See
[Roadmap](#roadmap) for what is still open.

## Getting started

Requirements: Go 1.27 or later.

```sh
make build                        # build bin/wtr and bin/what-to-run
bin/wtr                           # full screen
bin/wtr --inline                  # anchored below the prompt
```

For a permanent setup, `make install` copies both names into `~/.local/bin`
(`PREFIX=...` to install elsewhere), so they work from any directory; re-run
it after rebuilding.

Or without building a binary:

```sh
go run ./cmd/wtr
```

### Flags

| Flag | Description |
| --- | --- |
| `--inline` | Render between the current cursor line (the prompt) and the bottom of the screen instead of taking over the terminal. Requires an interactive terminal; falls back to full screen with a warning otherwise. |
| `--theme NAME` | Render the picker with one of the [themes](#themes). An unknown name is a warning; the default look is used. |
| `--list-themes` | Print the available themes and exit. |
| `--preview` | Print one static 80×24 frame of the picker with the tasks of the current directory and exit (aiding to compare themes: `--preview --theme nord`). |
| `--debug-area` | Draw a border around the whole area managed in inline mode (development aid; ignored in full screen). |

### Keys

| Key | Action |
| --- | --- |
| typing | filter the list (case-insensitive subsequence of the command or description, with matches highlighted) |
| `↑`/`↓`, `ctrl+p`/`ctrl+n` | move the selection |
| `pgup`/`pgdown`, `home`/`end` | jump in the list |
| `enter` | run the selected task |
| `tab` | inline only: place the command in the shell prompt without running it |
| `esc` | clear the filter, or quit when the filter is empty |
| `ctrl+u` | clear the filter |
| `ctrl+w` | open the warning box when warnings are present (`d` inside it dismisses for future runs) |
| `ctrl+t` | open the theme picker: ↑/↓ preview live, enter keep, esc cancel |
| `ctrl+c` | quit |

`q` types into the filter; it does not quit.

### Shell integration (zsh)

`init` prints a small zsh script to stdout; evaluate it to bind the picker to
a key in the current shell (the same pattern as `tv init zsh`):

```sh
eval "$(what-to-run init zsh)"               # default: Ctrl-S
eval "$(what-to-run init zsh --key ctrl-g)"  # bind another combo
```

Generate the integration with either name: the script keeps the name it was
generated with, so bind it as `what-to-run` to see that name in the
rc file and in the picker title, or as `wtr` to keep it short. A bare name is
resolved through `PATH` when the key is pressed; a path, like
`eval "$(./bin/wtr init zsh)"`, is embedded as an absolute path instead. With
the explicit name, `--help` and the message printed at shell startup also
point to `wtr`, when it is on `PATH`.

`--key` takes a combo name (`ctrl-s`, `alt-x`) or a raw zsh key sequence as
`bindkey` accepts it (`'^[[15~'` for F5). The key opens the inline picker
right below the prompt: `enter` runs the chosen task at the prompt, so it
lands in the shell history, and `tab` places the command line there without
running it. The generated script honours `WTR_BIN` (the binary the widget
launches) and `WTR_FLAGS` (extra flags, e.g. `WTR_FLAGS=--debug-area`, or
`WTR_FLAGS="--theme nord"` to pin a [theme](#themes)).
Ctrl-S and Ctrl-Q are the XOFF and XON flow-control characters: when one of
them is bound, the script turns IXON off (`stty -ixon`) so the key reaches zle
instead of freezing the terminal, and `stty ixon` restores the default.

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

### rc file

Two optional TOML files extend what the picker finds:

- `.what-to-run.toml` in the project directory (committed with the
  project: settings that belong to everyone);
- `$XDG_CONFIG_HOME/what-to-run/config.toml`, usually
  `~/.config/what-to-run/config.toml` (your personal settings).

The global file can also define per-project overrides under
`[projects."<path>"]` (matched against the current directory and its
subdirectories, from outermost parent to deepest match, with `~` and symlinks
resolved). Layers apply from general to specific: global top-level, then
`.what-to-run.toml`, then matching `[projects."<path>"]` blocks. The
theme is a user setting, like OpenCode's `cli.json`: only the global file sets
it, and the picker saves the choice there (a `theme` in a project file is a
warning, ignored). Problems in an rc file open the warning modal (`ctrl+w`,
dismissible with `d`) while keeping every valid setting in the file; see
[`docs/config.md`](docs/config.md) for the full specification.

```toml
# ~/.config/what-to-run/config.toml: start with a theme instead of the
# default one (see Themes). Choosing one in the picker (ctrl+t, then enter)
# saves it here; --theme wins for a single run.
theme = "nord"

# Override settings or add tasks for a directory tree without creating a local
# .what-to-run.toml inside the repository.
[projects."~/work/my-app"]
package_manager = "pnpm"
disable = ["Makefile"]
```

```toml
# .what-to-run.toml: force a package manager instead of detecting one.
package_manager = "bun"

# Skip a source, by any of the file names it looks for.
disable = ["Cargo.toml"]

# Custom tasks, shown above the discovered ones with the "rc" badge.
# They run through the shell (sh -c), so pipes and && work.
[[tasks]]
name = "deploy"
command = "scripts/deploy.sh --prod"
description = "Deploy to production" # defaults to the command
cwd = "ops"                          # optional, relative to the project

# An rc task hides the discovered task with the same name.
[[tasks]]
name = "test"
command = "go test -race ./..."
```

### Running a task

In **full screen**, `enter` runs the task in place: the picker hands the
terminal over to the task (Bubble Tea `tea.ExecProcess`) and takes it back when
the task ends, showing the outcome in the status line (`✓ build (1.2s)`,
`✗ test (0.4s): exit status 1`). The task's output stays in the terminal
scrollback, so it can be reviewed after quitting the picker.

In **inline** mode, `enter` quits the picker first and the task then runs at
the prompt like a hand-typed command, so its output is ordinary shell output in
the normal scrollback; the task's exit code becomes the exit code of
the picker. This is also the only way it can work: the inline renderer
cannot reposition its frame after another process scrolls the screen, so the
task must run after the picker has left it.

`tab` does the same up to the point of running: the chosen command line is
placed in the shell prompt, ready to be edited or deleted. Both keys go through
the shell when the picker is launched by a widget (see
[shell integration](#shell-integration-zsh)): the widget captures stdout, the
picker renders on the terminal itself and reports its result on stdout (a mode
line, `run` or `insert`, then the command line), and the widget fills the zle
buffer and accepts the line — so the command is visible at the prompt and lands
in the shell history, and `enter` executes it as if it had been typed. Without
a widget (stdout is the terminal), `enter` executes the task directly and `tab`
prints the command line.

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

### Themes

The look of the picker is one of these named themes — palettes sharing the
same selection style, an accent bar over a lighter tinted row (the colorless
`mono` tints the row with reverse video):

| Theme | Look |
| --- | --- |
| `classic` | the default violet look |
| `nord` | arctic blues and frost |
| `dracula` | high-contrast plum |
| `tokyo` | night blues on ink |
| `gruvbox` | warm retro amber |
| `catppuccin` | soft pastel mocha |
| `one-dark` | the editor classic: cool grays and blue |
| `monokai` | vivid pink, orange and green |
| `solarized` | the low-contrast classic: teal and blue |
| `everforest` | muted forest greens |
| `kanagawa` | ukiyo-e ink blues and sakura |
| `rose-pine` | dusty rose, pine and iris |
| `system` | the terminal's own palette and background |
| `mono` | no colors at all, reverse-video tint |

Every theme uses the bar-and-tinted-row selection; `mono`, having no colors,
tints the row with reverse video.

Pick one with `--theme nord` (for a single run), or open the theme picker with
`ctrl+t`: ↑/↓ scroll the list and preview each theme live, enter (or ctrl+t
again) keeps the choice and saves it, esc restores the previous one. A small
toast in the top-right corner confirms where it was saved and fades after a
couple of seconds; a failed save stays in the status line. The theme is a
personal preference, like OpenCode's `cli.json`: the choice is saved in your
user rc file, never in the project one, and starts every following run. The
box lists five theme names at a time, each in its own accent; the position and
↑/↓ arrows sit on the top border when the list scrolls. To compare them side
by side in a script, use the static preview:

```sh
for t in $(bin/wtr --list-themes | cut -c3- | cut -d' ' -f1); do
  printf '\n== %s ==\n' "$t"
  bin/wtr --preview --theme "$t"
done
```

`--list-themes` marks the default with `*`. The themes are plain data in
`internal/tui/theme.go`: a new variant is one entry in the registry.

## Roadmap

- [x] Discover tasks from `scripts` in `package.json`, targets in `Makefile`,
      and tasks in `Cargo.toml` (cargo-make)
- [x] Present them in a navigable, filterable list
- [x] Run the selected task and return to the list when it finishes
      (full screen; inline mode runs the task at the prompt instead)
- [x] `tab` in inline mode: place the command in the shell prompt without
      running it (zsh widget flow)
- [x] rc file (`.what-to-run.toml`, global `config.toml`): custom tasks,
      package manager override, disabled sources
- [x] zsh integration: `eval "$(what-to-run init zsh)"`, with a
      configurable key combo (`--key ctrl-g`)
- [ ] More sources: `justfile`, `Makefile.toml`
- [ ] `ctrl+r` to refresh the list after editing a task file

## Development

```sh
make test    # run tests with the race detector and coverage
make build   # build both binaries into bin/
make install # install both names into ~/.local/bin (PREFIX=... to change)
make bind    # build + print the zsh integration for eval (KEY=combo)
make run     # go run, without building a binary
make fmt     # gofmt and go vet
make lint    # golangci-lint (requires golangci-lint installed)
make help    # list all targets
```

There are no PTY tests for the inline mode: verify it end-to-end in tmux (zsh
plus the widget), and check the row math in
`internal/tui/tui_test.go` after any layout change.

### Trying the inline UI from the shell

```sh
eval "$(make bind)"             # build + bind Ctrl-S in the current shell
eval "$(make bind KEY=ctrl-g)"  # bind another combo
```

The widget runs `bin/wtr` on every invocation, so the binding
survives a rebuild: after editing Go code, `make build` is enough. After
editing the integration itself (`internal/tui/zshinit.go`), evaluate `make
bind` again to re-bind. The chosen key then opens the UI just below the prompt
(Ctrl-S turns `stty ixon` off because it is the XOFF flow-control character).
Extra flags can be passed to the widget with `WTR_FLAGS` (e.g.
`WTR_FLAGS=--debug-area`), and `WTR_BIN` overrides the binary path.

## Repository layout

```
cmd/wtr/                  entry point; built as bin/wtr and bin/what-to-run
internal/tui/tui.go       Bubble Tea model, views, and styles
internal/tui/tasks.go     the Task model
internal/tui/sources.go   discovery framework and the source registry
internal/tui/source_*.go  one parser per source (package.json, Makefile, Cargo.toml)
internal/tui/zshinit.go   zsh integration generator (init zsh)
```

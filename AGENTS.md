# what-to-run

TUI in Go (Bubble Tea v2 + Lip Gloss v2) that runs the tasks a project declares.
The sources are the `defaultSources` registry in `internal/core`, documented in
[docs/task-sources.md](docs/task-sources.md); the rc file adds `[[tasks]]` on
top.

Each fact has one owner: this file holds the rules for agents, `docs/` holds the
user reference, and the code, tests and workflows hold their own details. Link
to the owner instead of copying it.

## Pull requests

- **Merges are manual.** Only the maintainer merges a pull request, by hand.
  Agents never run `gh pr merge`, never push to `main`, and open PRs as drafts.
  Mark a PR ready only when the user asks in that turn. Stop for review.
- **Complex work uses stacked PRs.** Split a multi-part change into a chain of
  small PRs, each based on the previous, and open them as drafts. After the
  maintainer merges a parent, rebase and retarget the next onto `main`.

## Go development

Load the `golang-how-to` skill before any Go task: it routes to the Go skills
the task needs.

- `task all` is the full check and runs before every commit. CI
  (`.github/workflows/checks.yml`) runs `go build ./...`, `gofmt -l`,
  `dprint check`, `go vet ./...` and `go test -race -shuffle=on ./...` on
  Linux and macOS. Its `cross` job adds the GoReleaser check, the six
  cross-builds, the Windows vet, the Python tests of the release publishers
  and of the setup scripts, and the launcher's Node tests (`node --test`,
  Node 24). The Node tests are not in `task all`, so run `task test-npm`
  when touching `npm/`. If `task` is missing, run the commands in
  `checks.yml` by hand. Prerequisites, the dprint download and the task list
  are in [docs/development.md](docs/development.md).
- One test: `go test -race ./internal/tui -run TestViewInlineHeight`.
- Packages: `internal/core` is the task domain, `internal/config` reads and
  saves the rc file, `internal/tui` is the picker, `internal/shell` holds the
  three widgets, and `cmd/wtr` parses flags and subcommands. `internal/core` has
  no direct filesystem access: it reads through the `Files` port, calls
  `os/exec` for `Task.Command`, and keeps the Windows `SysProcAttr` in
  `command_windows.go`; `shellCommandFor(goos, ...)` keeps the platform shell
  testable.
- Adding a source: a `SourceInfo` literal (the ID from
  [Source IDs](docs/task-sources.md#source-ids), plus `Label`, `Files`,
  `SearchParents`, and `DisplayFileOverride` when the empty screen shows another
  file), its parser in `internal/core/source_<name>.go`, one line in
  `defaultSources`, and table-driven tests. `TestRegistryInvariants` enforces the
  metadata. The docs row and the README bullet are in the docs skill, and
  `cmd/wtr/docs_test.go` fails until both exist. An unknown `package.json` shape
  belongs in the matching `source_*.go`.
- `task build` compiles `cmd/wtr` into `bin/wtr`. The invoked name appears in
  usage, messages and the picker title, and the three `init` commands embed it in
  `WTR_BIN`; `cmd/wtr/main_test.go` pins the helpers.
- Releasing: [docs/development.md](docs/development.md) owns the pipeline,
  secrets and recovery, and the `release` skill walks the tag-to-verification
  flow.
- `.agents/skills/` is committed on purpose (`skills-lock.json`). `release/` and
  `docs-writing/` are this repo's own skills and have no lock entry;
  `bubbletea-tfe` is a manual rename of `ggprompts/tfe`, so reinstall it only
  under that name.

## TUI invariants

Each of these breaks silently, so check the one your change touches:

- The inline row math (`areaHeight`, `restoreCursor`) is pinned in `tui_test.go`
  by counting `\n` and `╭` glyphs. Update those tests with a layout change.
- Inline mode quits and the shell runs the chosen task; `tea.ExecProcess` is for
  fullscreen only, because the renderer cannot reposition the frame after the
  child scrolls the screen.
- `ctrl+f` flips `View().AltScreen`. Keep the main screen's width: the alternate
  screen can report one column more, and one column too many wraps every row and
  leaves the previous frame on screen.
- The selected row is painted segment by segment, never through a nested style
  reset and never in the dim color: the tint and the dim color can be nearly
  identical, and the reset drops the highlight before the row padding.
- The widget protocol is fzf-style: the widgets capture stdout on the terminal
  and the picker writes `run` or `insert` plus the command line. With no tasks,
  an inline run stays on a "no tasks found here" screen and a fullscreen run
  exits with the error.
- The rc file lives in [docs/config.md](docs/config.md). Parsing stays
  best-effort: unknown keys, wrong types and invalid entries warn and drop only
  the offending item. Warnings open the `ctrl+w` modal, and `d` dismisses one
  into `$XDG_CACHE_HOME/what-to-run/dismissed-warnings`.

## Comments

Comments follow the principles of Simplified Technical English
([ASD-STE100](https://www.asd-ste100.org/), Issue 9), but not its controlled
dictionary:

- Inside a function, explain why the code is there, not what it does.
- Write short sentences, with one idea in each sentence.
- Use the active voice.
- Keep the articles and the full sentence: no telegraphic English.
- Use one term for one thing, and prefer the term the codebase already uses.

## Charm v2 gotchas

- Import `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, not the
  `github.com/charmbracelet/...` v1 paths of most examples.
- v2 API: `View() tea.View` (set `AltScreen` on the view), `tea.KeyPressMsg`,
  `tea.RequestCursorPosition` / `tea.CursorPositionMsg`. Load the `bubbletea`
  skill for layout rules before writing UI code.
- `--inline` needs an interactive terminal and falls back to fullscreen with a
  warning. Inside a zle widget or a readline binding stdin is not the tty, so
  both widgets redirect `< /dev/tty` and disable `stty -ixon` when Ctrl-S or
  Ctrl-Q is bound.
- Ctrl-O is `DefaultKey`, not Ctrl-S: readline and ZLE use Ctrl-S for
  incremental forward search.
- No PTY tests: verify inline behavior in tmux (zsh with `eval "$(task bind)"`,
  bash with `eval "$(task bind-bash)"`, `KEY=ctrl-g` for another key).
  `--debug-area` draws a border around the managed area.

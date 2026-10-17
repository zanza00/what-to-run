# what-to-run

TUI in Go (Bubble Tea v2 + Lip Gloss v2) to run the scripts declared in a
project's `package.json`, `Makefile`, or `Cargo.toml` (cargo-make). Discovery
and execution are implemented: a filterable list runs the selected task, in
place in fullscreen (tea.ExecProcess) or at the prompt after quitting with
`--inline` (the Ctrl-S widget flow), propagating the task's exit code. In
inline mode `tab` instead places the command line in the shell prompt without
running it.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `golang-how-to` skill first — it routes to whichever other Go skills the task needs.

- `make all` is the full check (fmt + vet, race tests, build); no CI runs it, so
  run it before committing. A single test:
  `go test -race ./internal/tui -run TestViewInlineHeight`.
- All runtime code lives in `internal/tui` (`tui.go` = model and layout,
  `tasks.go` = the Task model, `sources.go` = the discovery framework and the
  source registry, `source_*.go` = one parser per source); `cmd/what-to-run`
  only parses flags and calls `tui.Run`. Adding a source: a `source` literal and
  its parser in a new `source_<name>.go`, one line in `defaultSources`, and
  table-driven tests next to them. Unknown `package.json` shapes etc. belong in
  the matching `source_*.go`.
- The optional rc file (`config.go`; `.what-to-run.toml` in the project,
  `$XDG_CONFIG_HOME/what-to-run/config.toml` globally) adds `[[tasks]]`,
  overrides `package_manager` and disables sources; its problems are warnings
  (status line), never fatal, like broken sources. Its tasks are assembled in
  `discoverTasks` and shadow same-named source tasks.
- The inline row math (`areaHeight`, `restoreCursor`) is pinned in
  `tui_test.go` by counting `\n` and `╭` glyphs — keep those tests in sync with
  any layout change.
- Do not run tasks in place with `tea.ExecProcess` from the inline picker: the
  renderer cannot reposition the frame after the child scrolls the screen
  (verified in tmux: the child output gets erased and dead frame rows remain).
  That is why the inline picker quits and the task runs afterwards.
- The inline widget protocol (fzf-style): `scripts/ctrl-s.zsh` captures
  stdout, the picker renders on the terminal via `tea.OpenTTY`/`tea.WithOutput`
  and reports its result on stdout (`run`/`insert` line + the command line from
  `Task.commandLine`); the widget fills the zle buffer and accepts the line.
  With stdout on the terminal (no widget) `Run` executes the task itself after
  `restoreCursor`, and `tab` just prints the command line.
- `.agents/skills/` is committed on purpose (see `skills-lock.json`);
  `bubbletea-tfe` is a manual rename of `ggprompts/tfe` to avoid an id collision
  with `bubbletea` — don't reinstall it unrenamed.

## Charm v2 gotchas

- Import `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, not the older
  `github.com/charmbracelet/...` v1 paths used by most examples.
- v2 API: `View() tea.View` (set `AltScreen` on the view), `tea.KeyPressMsg`,
  `tea.RequestCursorPosition` / `tea.CursorPositionMsg`; load the `bubbletea`
  skill for layout rules before writing UI code.
- `--inline` requires an interactive terminal (warns and falls back to
  fullscreen otherwise). Inside a zle widget stdin is not the tty:
  `scripts/ctrl-s.zsh` redirects `< /dev/tty` and disables `stty -ixon` because
  Ctrl-S is the XOFF flow-control character.
- No PTY tests: verify inline behavior end-to-end in tmux (zsh + the Ctrl-S
  harness); `make try` builds and opens a clean interactive zsh with the widget
  bound, `--debug-area` draws a border around the managed area.

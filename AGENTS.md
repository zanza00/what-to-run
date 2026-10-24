# what-to-run

TUI in Go (Bubble Tea v2 + Lip Gloss v2) to run the scripts declared in a
project. Supported task files:

- `package.json`
- `Makefile`, `makefile`, `GNUmakefile`
- `Taskfile.yml` and the other go-task default names (`taskfile.yml`,
  `Taskfile.yaml`, `taskfile.yaml`, `Taskfile.dist.yml`, `taskfile.dist.yml`,
  `Taskfile.dist.yaml`, `taskfile.dist.yaml`)
- `justfile`, `Justfile`, `.justfile`
- `Makefile.toml`
- `.cargo/config`, `.cargo/config.toml`
- `devbox.json`
- `composer.json`
- `deno.json`, `deno.jsonc`
- `mise.toml`, `.mise.toml`
- `[[tasks]]` in `.what-to-run.toml` and the user config

Discovery and execution are implemented: a filterable list runs the selected
task, in place in fullscreen (tea.ExecProcess) or at the prompt after quitting
with `--inline` (the widget flow from `init zsh`, `init bash` and
`init powershell`), propagating the task's exit code. In inline mode `tab`
instead places the command line in the shell prompt without running it.

## Pull requests

- **Merges are manual.** Only the maintainer merges a pull request, by hand.
  Agents never run `gh pr merge`, never push to `main`, and open PRs as drafts;
  do not mark a PR ready unless the user asks in that turn. Stop for review.
- **Complex work uses stacked PRs.** Split a multi-part change into a chain of
  small PRs, each based on the previous, and open them as drafts. After the
  maintainer merges a parent, rebase/retarget the next onto `main`; never merge
  a layer.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `golang-how-to` skill first — it routes to whichever other Go skills the task needs.

- `task all` is the full check (fmt + vet, race tests, build); CI
  (`.github/workflows/checks.yml`, also called by the release workflow) runs
  the same Go commands plus gofmt and dprint gates on linux and macOS, and its
  `cross` job builds and vets for Windows, so run `task all` before committing.
  `task` is a contributor prerequisite, not a runtime dependency of `wtr`:
  install it with
  `go install github.com/go-task/task/v3/cmd/task@v3.54.0`, which writes the
  binary to `$(go env GOPATH)/bin` (or `$GOBIN`), so that directory must be on
  `PATH`; when `task` is missing, run the Go commands in `checks.yml` by hand.
  A single test:
  `go test -race ./internal/tui -run TestViewInlineHeight`.
- Markdown is formatted by dprint, configured in `dprint.json`; `.agents/` is
  excluded because those skills are vendored, and tables keep their compact
  layout. The pinned CLI is downloaded into `bin/tools` by
  `scripts/install-dprint.go`, which `task fmt-md` and `task check-md` run, so
  no global dprint install is needed; `task all` includes `fmt-md`.
- Runtime code is split in three internal packages. `internal/core` is the task
  domain: `task.go` = the Task model and command lines, `shell.go` = the shell
  dialects, their quoting and the platform shell for rc tasks, `discover.go` =
  the discovery framework and the source registry, `source_*.go` = one parser
  per source, `jsonc.go` = the JSON-with-comments decoder, `files.go` = the
  `Files` port (`ReadFile` and `Exists`) that discovery and the rc loader read
  through. `internal/tui` is the Bubble Tea picker: `tui.go` = model and layout,
  `config.go` = the rc file, `files.go` = the `Files` OS adapter, `theme.go` =
  the named looks and the styles derived from them, `theme_modal.go` = the
  ctrl+t theme picker overlay, `warning_modal.go` = the ctrl+w warning modal and
  dismissed-warnings cache, `notice.go` = the transient toast. `internal/shell`
  holds the shell integrations (`zshinit.go`, `bashinit.go` and
  `powershellinit.go` = the shell widgets printed by `init zsh`, `init bash`
  and `init powershell`). `cmd/wtr` only parses flags/subcommands and calls
  `tui.Run` / `tui.Preview` / `shell.ZshInit` / `shell.BashInit` /
  `shell.PowerShellInit`. Adding a source: a `source` literal and its parser in
  a new `internal/core/source_<name>.go`, one line in the `defaultSources`
  registry in `internal/core`, and table-driven tests next to the parser.
  Unknown `package.json` shapes etc. belong in the matching `source_*.go`.
  `internal/core` itself has no direct filesystem access — its only OS access
  is `os/exec`, for `Task.Command`, plus the Windows-only
  `syscall.SysProcAttr` that hands cmd.exe a raw command line
  (`command_windows.go`) — so discovery and config parsing can run against an
  in-memory port in tests. The platform shell choice stays testable off Windows
  through `shellCommandFor(goos, ...)`.
- `task build` compiles `cmd/wtr` twice: `bin/wtr` is the CLI name and
  `bin/what-to-run` the explicit one for shell rc files. The invoked
  name is the one shown in usage, messages and the picker title (on Windows
  without the `.exe` suffix), and
  `init zsh`, `init bash` and `init powershell` embed the invoked command — the
  bare name via PATH, or the absolute path — in `WTR_BIN`; with the explicit
  name, `--help` and the generated script also point at `wtr` when it is on
  PATH.
  `cmd/wtr/main_test.go` pins the helpers.
- Releases are tag-driven: `.github/workflows/release.yml` runs on `vX.Y.Z`
  tags whose commit is in `main`, builds the linux+darwin archives, the
  Windows zips (both holding both binaries) and the linux `.deb` and `.rpm`
  packages through `.goreleaser.yaml`, and
  `scripts/publish-homebrew.py` writes `Formula/what-to-run.rb` in
  `zanza00/homebrew-tap` using the `HOMEBREW_TAP_TOKEN` secret while
  `scripts/publish-aur.py` pushes the `what-to-run-bin` PKGBUILD to the AUR
  using the `AUR_SSH_PRIVATE_KEY` secret, skipped with a warning while that
  secret is unset, and `scripts/publish-repo.py` updates the apt and RPM
  repositories on the `gh-pages` branch (served via GitHub Pages) using the
  `APT_GPG_PRIVATE_KEY` secret, skipped with a warning while that secret is
  unset; the same key signs the RPM packages at build time. The publisher also
  copies `scripts/setup.deb.sh` and `scripts/setup.rpm.sh` to the root of
  `gh-pages`, where the one-liners in `docs/install.md` fetch them from; the
  apt repository stays at the site root and the RPM repository lives under
  `/rpm/`. `--version` prints
  the version
  injected with `-X main.version` (the tag in GoReleaser, `git describe` in
  `Taskfile.yml`). The publishers have Python tests: run
  `python3 -m unittest discover -s scripts -p '*_test.py'` when touching them.
- The optional rc file (`config.go`, documented in `docs/config.md`;
  `.what-to-run.toml` in the project,
  `$XDG_CONFIG_HOME/what-to-run/config.toml` globally,
  `%AppData%\what-to-run\config.toml` on Windows) adds `[[tasks]]`,
  overrides `package_manager`, disables sources, and supports per-project
  overrides (`[projects."<path>"]`) in the user file. It is parsed in
  best-effort mode so unknown keys, wrong types, or invalid entries warn and
  drop only the offending item while keeping the rest of the file. Warnings open
  the `ctrl+w` warning modal (`warning_modal.go`, dismissible with `d` into
  `$XDG_CACHE_HOME/what-to-run/dismissed-warnings`) and show a warning
  count in the bottom help line without hiding the chrome. The theme is a user
  setting like OpenCode's `cli.json`: only the global file sets it (a project
  `theme` warns and is ignored) and `saveTheme` writes it there.
- The look is a named theme (`theme.go`): the registry order is the order of
  the `ctrl+t` picker, the first entry (`classic`) is the default, `--theme`
  pins one and an unknown name is only a warning. `--list-themes` and
  `--preview` (a static 80×24 frame) are the comparison aids. Every theme uses
  the same bar selection: the selected row is painted segment by segment — the
  `selBg` tint with the `selFg` light text, or reverse video for the colorless
  `mono` — never the dim color, which can be nearly identical to the tint,
  because a nested style's reset would drop the highlight before the row
  padding.
- `ctrl+t` opens the theme picker (`theme_modal.go`): a centered box listing
  five theme names at a time, each in its own accent (the top border carries
  the position and ↑/↓ when the list scrolls). ↑/↓ preview live, enter or
  ctrl+t keep and save the choice (`saveTheme`: the user rc file, like
  OpenCode's cli.json, confirmed by a transient toast — `notice.go`, 2s, top
  right; failures stay in the status line), esc restores the theme that was
  active when it opened. The notice and the box are composited cell by cell
  over the rendered rows (the lipgloss canvas, `overlay`), so a tinted
  background crossing their edges is not broken.
- Rows show the command line (`Task.CommandLineFor`, quoted for the
  `--shell` dialect: the platform default unless set), the description and the
  source badge: no name column, the command already contains the name. The
  command column is the widest visible command line, so descriptions start
  together; it is capped to the room the row has, and the description takes
  what is left — dropped before the command shrinks. An rc task without a
  custom description defaults to its own command, which the platform shell
  (`sh -c`, `cmd /c` on Windows) already shows in the command column, so the
  duplicate description is hidden there. The
  filter matches tokens against
  the task name, the command line or the description, highlighting the matched
  characters in the row.
- The inline row math (`areaHeight`, `restoreCursor`) is pinned in
  `tui_test.go` by counting `\n` and `╭` glyphs — keep those tests in sync with
  any layout change. In inline mode `ctrl+f` expands the picker to the whole
  terminal and toggles back: it flips `View().AltScreen`, so the prompt and
  history are restored by the terminal, and enter/tab keep their inline
  semantics (choose and quit, the shell runs the command). Expanding
  remembers the main screen's width: the alternate screen can report one
  column more (a scrollbar disappears), and laying the first inline frame out
  at that width wraps every row and leaves the old frame on screen (Herdr
  reproduces this). The next main-screen `WindowSizeMsg` refreshes it.
- Do not run tasks in place with `tea.ExecProcess` from the inline picker: the
  renderer cannot reposition the frame after the child scrolls the screen
  (verified in tmux: the child output gets erased and dead frame rows remain).
  That is why the inline picker quits and the task runs afterwards.
- The inline widget protocol (fzf-style): the generated widgets (`init zsh`,
  `init bash`, `init powershell`) capture stdout, the picker renders on the
  terminal via `tea.OpenTTY`/`tea.WithOutput` and reports its result on stdout
  (`run`/`insert` line + the command line from `Task.CommandLineFor`). A
  directory with no tasks keeps the inline picker open on a persistent
  "no tasks found here" screen (enter/esc close), instead of printing an
  error the prompt redraw would erase; a direct fullscreen run still fails
  fast. The zsh
  widget fills the zle buffer and submits it with `zle accept-line`; the bash
  widget puts the command in `READLINE_LINE` and points the second half of the
  key's readline macro at `accept-line` — readline has no way to submit a line
  from shell code: the key expands to two invisible sequences and the widget
  rebinds the tail of the chain at press time. The chain needs bash 4.3 or
  later (`bind -x` takes only two-byte sequences before), and the binds sit
  inside an `[[ $- == *i* ]]` guard, so a non-interactive shell sourcing the
  rc file stays quiet. The PowerShell widget (`internal/shell/powershellinit.go`)
  captures stdout natively and hands the result to PSReadLine — `Insert` for
  tab, `Replace` + `AcceptLine` for enter — with the line quoted for
  `--shell powershell`; on legacy conhost the cursor-query timeout makes the
  first inline frame fall back to fullscreen. With stdout on the terminal (no
  widget) `Run` executes the task itself after `restoreCursor`, and `tab` just
  prints the command line.
- `.agents/skills/` is committed on purpose (see `skills-lock.json`);
  `release/` is this repository's own skill (not vendored, so it has no lock
  entry) and walks the tag-to-verification release flow; `bubbletea-tfe` is a
  manual rename of `ggprompts/tfe` to avoid an id collision with `bubbletea` —
  don't reinstall it unrenamed.

## Charm v2 gotchas

- Import `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, not the older
  `github.com/charmbracelet/...` v1 paths used by most examples.
- v2 API: `View() tea.View` (set `AltScreen` on the view), `tea.KeyPressMsg`,
  `tea.RequestCursorPosition` / `tea.CursorPositionMsg`; load the `bubbletea`
  skill for layout rules before writing UI code.
- `--inline` requires an interactive terminal (warns and falls back to
  fullscreen otherwise). Inside a zle widget or a readline keybinding stdin is
  not the tty: both generated widgets redirect `< /dev/tty` and disable
  `stty -ixon` when Ctrl-S or Ctrl-Q is bound, because Ctrl-S is the XOFF
  flow-control character.
- No PTY tests: verify inline behavior end-to-end in tmux (zsh + the widget
  from `eval "$(task bind)"`, bash + the one from `eval "$(task bind-bash)"`,
  with `KEY=ctrl-g` to bind another combo); the widget runs `bin/wtr` on every
  invocation, so plain `task build` is enough for Go-only changes.
  `--debug-area` draws a border around the managed area.

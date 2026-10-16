# what-to-run

TUI in Go (Bubble Tea v2 + Lip Gloss v2) to run the scripts declared in a
project's `package.json`, `Makefile`, or `Cargo.toml` (cargo-make).
Current state: hello-world skeleton; task discovery and execution are not
implemented yet (see `README.md`).

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `golang-how-to` skill first — it routes to whichever other Go skills the task needs.

- `make all` is the full check (fmt + vet, race tests, build); no CI runs it, so
  run it before committing. A single test:
  `go test -race ./internal/tui -run TestViewInlineHeight`.
- All runtime code lives in `internal/tui`; `cmd/what-to-run` only parses
  flags and calls `tui.Run`. The inline row math (`areaHeight`, `restoreCursor`)
  is pinned in `tui_test.go` by counting `\n` and `╭` glyphs — keep those tests
  in sync with any layout change.
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
  harness); `--debug-area` draws a border around the managed area.

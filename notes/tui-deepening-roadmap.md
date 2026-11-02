# TUI deepening roadmap

The architecture review of 2026-11-01 (scope: the graphical part of
`internal/tui`) surfaced seven deepening candidates. Line references below
are true as of `main@80c4175`; the box module (FP4) landed in PR1
(`8674bdb`) and shifts some of them.

Attack order = silent-breakage risk removed ÷ invasiveness. The box module
is done (PR #84); the rest are pending:

1. **Box module (FP4)** — DONE, see [box-module](box-module.md). The two
   modals drew the same box twice; one module owns the border protocol now.
2. **Status as data (FP7)** — the status line is styled at event time
   (`taskDoneMsg` tui.go:465-471, `applyThemeSaved` 727-730,
   `applyWarningDismissed` warning_modal.go:94-96), so the ctrl+t live
   preview restyles everything except the status. Fix: status becomes data,
   `helpLine` paints it. Small; fixes the preview inconsistency outright.
3. **Frame as structure (FP2)** — `render` (tui.go:1248-1275) flattens to a
   string too early; `warningModalY` re-renders the list to measure
   (tui.go:1327-1338, with a magic `3` duplicating `chromeRows`); tests count
   `\n` and `╭` to recover structure. Biggest payoff; the placedBox/overlay
   seam is already halfway there.
4. **Windowing module (FP6)** — three hand-rolled list windows (task, theme,
   warning); `keepCursorVisible` is a calling convention remembered at six
   call sites and re-applied defensively in `listLines` (tui.go:1412-1414).
5. **Row painter (FP3)** — the selected-row protocol (AGENTS.md invariant #4)
   is written twice (`taskRow` tui.go:1454-1518, `themeModalRow`
   theme_modal.go:128-146) and enforced by SGR-grep tests.
6. **List module absorbs the snapshot (FP5)** — the filter snapshot's
   validity leaks into three read paths (tui.go:795-865); `taskMatches`
   (tui.go:895-897) had zero callers. Do after FP6 (same state).
7. **Display-mode module (FP1)** — four real display modes on three
   booleans, re-derived at five sites (tui.go:313-343, 430-452, 548-562,
   1227-1234, 171-178). The widest cut; naturally last, after FP2.

## The invariants map

Each AGENTS.md "breaks silently" invariant implies a seam that did not
exist on 2026-11-01:

| Invariant | Missing seam | Candidate |
| --- | --- | --- |
| Inline row math | module owning anchor + row budget | FP1 |
| Widget protocol | (already the strongest seam) | — |
| ctrl+f width flip | display-mode module | FP1 |
| Segment painting | row painter | FP3 |
| No-tasks screens | frame makes screens first-class | FP2 |

Bonus (not in AGENTS.md): warning identity for dismissal is the English
message text (`warningKey` warning_modal.go:220-227), and
`hasThemeWarning` matches the substring "theme" (tui.go:749-757); a reworded
message silently un-dismisses every dismissal in `$XDG_CACHE_HOME`. A typed
warning kind is a small seam with a large silent-failure radius.

## Verified only in tmux (2026-11-01)

`Run`'s TTY wiring and Windows cancel-reader caveats; the cursor-position
round trip against a real terminal; the alternate-screen width quirk (CI
simulates it with a synthetic 81-column resize); `restoreCursor`'s handoff
with the shell widgets. No PTY tests by repo rule. `truncateMatched`
(tui.go:1520-1607) had no direct unit test on 2026-11-01 — an accidental
gap, not a PTY limit; it is pure and testable now.

# Status as data (piece 2)

The status line becomes data in the model — a paint kind and a plain text —
and the help line paints that pair at render time, so no styled bytes live in
the model and the ctrl+t live preview restyles the status together with every
other surface. Grilled with the maintainer on 2026-11-04; the synthesis lives
in the local tracker (`.scratch/fp7-status-as-data/spec.md`); this page is
the durable record. Candidate: "FP7 status as data" in
[arch-review-2026-11](arch-review-2026-11.md), piece 2 of
the [attack plan](attack-plan.md). Shipped on 2026-11-04 as PR #118 (merge
commit `7e09ed2`).

## What was broken (all at 2026-11-04, pre-change)

- The model stored the status as a pre-rendered styled string, painted at
  event time (tui.go:473, 475, 734, 779; warning_modal.go:143), so
  the ctrl+t live preview restyled every surface except the status line: a
  message kept the colors of the theme that was active when the event
  happened.
- The tests stripped ANSI escape sequences to read the status
  (theme_test.go:268, 513, 608; effects_test.go:64, 99, 240; tui_test.go:
  1096-1110), because the styled bytes were the only form of the fact.
- The comments lied about the lifetime: the notice claimed a problem stays
  "until the next event", and the field claimed "the outcome of the last
  in-place run", but nothing clears the status and two of its five writers
  are save failures.

## The decisions (2026-11-04)

- **The kind is a paint bucket.** The status is a small type, `statusKind`
  plus a text (`internal/tui/status.go`). The kind is a closed set of three
  paints — `running`, `ok`, `err` — plus the zero kind, which means that
  the model holds no status. The kind names the paint, not the event: a
  failed run and a failed save share `err`. The text is the exact message
  of today, icon included, because only run outcomes carry `✓`/`✗` and the
  kind cannot derive the icon.
- The three kinds map one to one onto the three styles that painted them
  before. The mapping lives in `helpLine` (tui.go:1672), the only reader:
  with the zero kind it shows the task count and the hints as before, and
  with a paint kind it renders the text through the mapped style and
  returns it.
- **No setter joins the writers.** All five writers build the pair with a
  struct literal: the run outcomes (tui.go:476, 478), the theme-save
  failure (tui.go:737), the running notice (tui.go:782), the
  dismissal-save failure (warning_modal.go:143). The type itself carries
  the invariant that no styled bytes enter the model; a setter would
  offer a second door.
- **The notice and the status stay two types.** They share one rule, which
  both their comments state: the model holds no styled bytes, and the
  chrome paints at render time through the active theme. The notice
  already complied; no notice behaviour changed.
- **The lifetime lives in prose, not in a mechanism.** A status stays in
  the help line until another status replaces it. No key press clears it,
  no timer runs on it. The field comment covers every writer, and the
  notice comment states the split: a problem is not a notice, so it stays
  in the help line until another status replaces it. This closed the
  notice.go:15 drift.
- The painting stays byte-identical wherever the theme has not changed. The
  unit-level sweep of the [differential-testing](differential-testing.md)
  recipe proved it: the old help line, inlined from
  `git show main:internal/tui/tui.go`, compared against the new one across
  every status text of the five writers and the useful widths, same theme,
  zero tolerated differences. The binary-level `--preview` check stays
  out: a preview frame carries no run and no save, so it cannot exercise
  the status.
- Scope riders: one CHANGELOG entry under `### Fixed` in `[Unreleased]`
  names the user-visible fix, the ctrl+t live preview restyling the status
  line. The status only shows in fullscreen, and inline mode still quits
  at `pick`, before any status exists. No ticket, the piece is small.

## What landed (2026-11-04, PR #118)

- `0820514` holds the status as data and paints it at render time:
  `internal/tui/status.go` (the closed kind set and the pair), the five
  writers, the `helpLine` kind check, and the tests that read the data
  instead of stripping ANSI. Exactly one assertion goes through the painted
  view and pins that the help line the picker draws carries the active
  status.
- `a45dd4d` words the CHANGELOG entry; `5b51d5b` gives the status comments
  one owner and one register, notice comment included.

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

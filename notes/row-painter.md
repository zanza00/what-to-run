# Row painter (piece 6)

The row assembly becomes a painter: one new file, `internal/tui/row.go`,
owns both branches of the selection protocol that the task row and the
theme row share — the per-segment law, the never-dim law and the pad law —
while the two rows shrink to their budgets and hand the painter their
pieces, so a new surface inherits the protocol instead of copying it.
Grilled with the maintainer on 2026-11-05; the synthesis lives in the
local tracker (`.scratch/fp3-row-painter/spec.md`); this page is the
durable record. Candidate: "FP3 row painter" in
[arch-review-2026-11](arch-review-2026-11.md), piece 6 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #126 (merge
commit `d2256f8`, the single implementer commit `c434c76`).

## What was broken (all at 2026-11-05, at `1662dc6`)

- The selection protocol was written twice. `taskRow` (tui.go:1447-1509)
  assembled the marker, the matched pieces, the pad and the badge in its
  selected branch (:1488-1502, through `selectedName`/`selectedText`/
  `selectedMatch`) and in its plain branch (:1504-1509), and
  `themeModalRow` (theme_modal.go:112-133) assembled the same protocol for
  the theme rows a second time.
- Three house laws lived by copy at both sites, and any new surface
  inherited them by copy: paint segment by segment, never through a
  nested style reset; the selected row never takes the dim color, because
  the dim color can be nearly identical to the tint; the pad goes through
  its style by fraction only when it is not empty, because lipgloss
  renders the empty string as a zero-width escape pair.
- Nothing pinned the two sites to each other. The SGR pin,
  `TestSystemThemeAdaptsToTerminalColors` (theme_test.go:59, the
  `wantSelBg`/`wantSelFg`/`wantDim` table at :64-97), asserted the
  selected bytes of the task row only; the selected theme row's escape
  sequences carried no pin.
- The two painting primitives had no direct tests: `truncateMatched`
  (tui.go:1513, the shift-toward-match window with its ellipsis) and
  `renderMatched` (:1602, the per-segment SGR pairs) ran only through the
  full view, while their sibling `matchPositions` (:961) had a table since
  before the review.

## The decisions (2026-11-05)

- **Both branch bodies move into the painter.** The caller hands
  `paintRow` the marker, the pieces with their match flags and styles, the
  badge, the row budget and the pad style; the painter assembles both
  branches, so each of the three laws has exactly one home. The selected
  and the plain branch differ only through the data the caller passes.
- **The rows keep their budgets.** `taskRow` keeps the badge width,
  `room`/`avail`, the command-width cap against the list module's
  `commandWidth`, the rc-task description rule and `flattenDesc`;
  `themeModalRow` keeps `nameCol`/`innerW`. The painter owns one thing:
  the branch assembly, no layout logic.
- **The primitives move, the matcher stays.** `truncateMatched` and
  `renderMatched` move beside the painter verbatim — the bodies frozen,
  free functions with no receivers — and gain their first direct tables;
  `matchPositions` stays in `tui.go`, because it is matching and not
  painting.
- **The measured row cache stays out.** The arch-bench patch B baselines
  (−12.5…−44.5% on steady frames) ride their own piece after FP1 — this
  grilling's Q4, taken together with FP5's Q2, fixed that order. This
  piece freezes behavior, not cost.

## The ratifications (2026-11-05, PR #126)

The implementer declared five deviations from the spec; the review ratified
every one:

- the selected flag: the styles carry the paints of each branch, and the
  flag names the branch whose assembly runs;
- the SGR pin leg through the view, in the same register as the task
  row's pin, not a sibling table;
- the Underline omission in the pin's byte capture — a lipgloss per-rune
  pair quirk;
- the no-badge signal (the theme row passes no badge rather than an empty
  one);
- mutation teeth instead of a red phase for frozen behaviour.

The Standards axis found no hard violations; the persisting rule held
(the words "row", "paint", "pad", "badge" traced to the comments' prose
before the first commit). Its three judgement calls closed in the final
commit: the contradicted comment sentence rewritten honestly (the styles
carry the paints, the flag names the branch), the duplicated piece
literals deduplicated behind one style-pair pick, and the budget port's
correspondence named (the task row's room, the theme row's inner width).
The godoc binding was fixed too: the module overview no longer binds to
`rowPiece`.

## The proofs (2026-11-05, PR #126)

- **Frame byte sweep**: the old `taskRow`/`themeModalRow` inlined from
  `git show 1662dc6:internal/tui/tui.go` and
  `git show 1662dc6:internal/tui/theme_modal.go`, compared raw through
  `buildFrame(h, w).paint()`, never ANSI-stripped — the task-row legs at
  every h 1-40 × w 1-100 under classic/nord/mono/system-dark, the filter
  typed through its states, the rc rule, the CR/LF description, the wide
  badge, the window-shifted command and the cursor at top/middle/bottom
  (48,000 comparisons), the 1000-task four-leg block (16,000), and the
  box legs — the theme box at top/scrolled/bottom, the warning box and
  the notice — at the sparse edges (2,400): **50,400 comparisons, 0
  differences**. Re-run on the amended tree with the same numbers. The
  teeth: breaking the pad law made the sweep report 19,401 differences.
- **Binary check**: both builds invoked as `wtr-preview` with a fresh
  `XDG_CACHE_HOME` per run — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B) `cmp` identical against
  `main` `1662dc6`.
- **Permanent tests** (27 subtests): `TestPaintRow` pins the painter's
  contract by raw SGR bytes captured from the old assembly (the marker
  pair, one pair per piece and per matched run, the pad law's fraction at
  the exact budget in both branches, the badge's separator placement, the
  badge-less theme shape); `TestTruncateMatched` (9 cases, the first
  direct tests) and `TestRenderMatched` (5 cases) pin the two primitives.
  The SGR pin extends to the selected theme row, its teeth proven by
  mutation. `task all` green; 259 internal/tui tests with
  `-race -shuffle=on`; the glyph pins unmodified.
- **The review verdicts.** The Spec axis: conformant, nothing missing,
  wrong or creeping. The Standards axis: no hard violations, the
  judgement calls closed above.

## What landed (2026-11-05, PR #126)

- `c434c76` owns the row assembly in one painter: 5 files, +591/-155 —
  `row.go` (199 lines), `row_test.go` (345), `tui.go` (mostly
  deletions), `theme_modal.go`, `theme_test.go`. `task all` green at
  that commit; 259 internal/tui tests with `-race -shuffle=on`.
- `d2256f8` merged it into `main`. The piece-5 bookkeeping had landed
  first, with PR #125 (merge commit `5de0237`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

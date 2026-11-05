# Display-mode module (piece 7)

Where the picker draws becomes a module: one new file,
`internal/tui/display.go`, owns the three mode facts, the six geometry
facts and the three derivations; the ten model-side re-reads of
`m.opts.Inline` die, the mutations settle behind five named operations,
`Preview` crosses the same door as the messages, and `restoreCursor`
un-nests from the model. Grilled with the maintainer on 2026-11-05
(eight questions, every recommendation accepted); the synthesis lives in
the local tracker (`.scratch/fp1-display-mode/spec.md`); this page is the
durable record. Candidate: "FP1 display-mode module (with Preview at the
seam)" in [arch-review-2026-11](arch-review-2026-11.md), piece 7 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #128 (merge
commit `cbdde4f`, the single implementer commit `b22afa2`).

## What was broken (all at 2026-11-05, at `d2256f8`)

- Three booleans — `opts.Inline`, `m.fullscreen`, `m.inlineUnavailable` —
  combined into four display modes, and the model recombined them at
  fifteen sites instead of reading one fact: `Init`'s cursor query
  (tui.go:437), the `WindowSizeMsg` mainWidth memory (:453), the
  `CursorPositionMsg`/`cursorPosTimeoutMsg` fallback (:468-478), the tab
  (:551) and ctrl+f (:583-589) handlers, `run()`'s quit-or-exec fork
  (:816), `View`'s register fork (:1156), `inlineWidth` (:1190),
  `listHeight` (:1199), the tab-to-prompt hint (:1535), the ctrl+f hints
  (:1551-1557), `themeModalHeight` (theme_modal.go:23-24), and `Run`'s
  empty-screen decision (:157) with its restoreCursor recombination
  (:167-172). `m.opts.Inline` alone was re-read at ten of them.
- The inline geometry was re-derived the same way: `areaHeight`
  (tui.go:1585), `viewHeight` (:1177) and `inlineWidth` (:1189) each
  recombined the six raw facts (`width`, `height`, `ready`, `startRow`,
  `startRowKnown`, `mainWidth`), and `restoreCursor` (tui.go:282-287)
  built on `areaHeight` while `Run` re-derived whether it must run at
  all.
- `Preview` (tui.go:240-256) bypassed the seam: it muted its local
  Options by hand (:246-247), constructed the model, then hand-set the
  geometry — `m.width, m.height, m.ready = width, height, true` (:252) —
  a second construction path free to drift from the live one.
- `m.opts.Theme` was written by `applyThemeSaved` (tui.go:779) and read
  by nothing after construction: `newModel` reads the local `opts`
  parameter, never the field.

## The decisions (2026-11-05)

- **The module owns the mode and the geometry.** `display`
  (`internal/tui/display.go`, field `m.display`) replaced the fields
  `width`, `height`, `ready`, `startRow`, `startRowKnown`, `fullscreen`,
  `inlineUnavailable`, `mainWidth`. The three derivations —
  `areaHeight`, `inlineWidth`, `viewHeight` — keep their names and
  become reads of the module. The model never re-reads `m.opts.Inline`;
  `m.opts` stays a construction fact (`m.opts.DebugArea` and
  `m.opts.prog()` remain direct reads); `Run`'s pre-model fallbacks
  (tui.go:117-131) stay where they are.
- **Two named facts, one screen choice.** `startedInline()` reports how
  the picker was started (inline quits with run/insert, fullscreen runs
  in place): `run()`, `tab`, the tab-to-prompt hint read it — and
  `themeModalHeight` (theme_modal.go) keys on it, preserving
  byte-for-byte the asymmetry below. `anchored()` reports the inline
  screen (the frame drawn in the area, no alternate screen) and
  `expanded()` the alternate screen: `View` forks on `anchored()`, sets
  `AltScreen` from `expanded()`, `listHeight` and `inlineWidth` read the
  screen, the ctrl+f hints read both, and `Run`'s finally clears the
  block only when `anchored()` held at quit.
- **Five operations settle the mutations, in the windowing pattern.**
  `resize(w, h)` writes the size, marks the model ready and remembers
  `mainWidth` only while anchored; `learnPosition(y)` sets the anchor
  row and clears the fallback; `fallBack()` (only when started inline
  and the position never answered) expands once, reports the move, and
  settles nothing on a second fire; `collapse()` returns to the area;
  `expand()` guards the fallback internally (today's ctrl+f guard moved
  inside) and also refuses when already expanded, so the operation stays
  total. The refusals report no move, and the handlers gate their window
  settles on the report, so a refused key never settles a window the old
  code never settled.
- **Preview crosses the door.** The local Options mutes stay (Preview's
  copy is a value, no model exists yet); the geometry write goes through
  the module: `m.display.resize(width, height)` replaced the triple
  assignment. Byte-identical: `inlineWidth` returns `min(width, width)`,
  the printed frame unchanged.
- **`restoreCursor` becomes a pure shell helper** — `restoreCursor(w
  io.Writer, h int)` — and `Run`'s finally reads the module once:
  `if m.display.anchored() { restoreCursor(out, m.display.areaHeight()) }`.
  The name stays (AGENTS.md cites it).
- **The dead theme write dies.** `m.opts.Theme = msg.name` in
  `applyThemeSaved` is deleted; the model already holds the saved theme
  through `themeWin.cursor` and `styles`.
- **Naming per the persisting rule.** `display`, `display.go`,
  `m.display`, and the operation and read names above; the implementer
  checked them against the comments' prose before the first commit. The
  spec's own umbrella word "the register" was dropped in review for the
  screens/area vocabulary, after the Standards axis found it colliding
  with the sources/themes registry and the color register.
- **The proofs and the scope guards.** The selection sweep plus the
  FP3-shape frame sweep and binary check (see The proofs); no CHANGELOG
  nor GLOSSARY, AGENTS.md untouched (the pinned names keep their
  names), and the measured row cache stays its own piece — this grilling
  round fixed it as the next piece after this one.

## The finding, a candidate follow-up (2026-11-05, ratified)

`themeModalHeight` keys on the inline start, not on the screen: from an
inline start expanded with ctrl+f, the theme box height is the inline
area height, not the terminal height. The grilling (question 8) ratified
preserving the bytes byte-for-byte and recording the asymmetry here as
a candidate follow-up. The piece-7 frame sweep exercised it on purpose —
the theme box ran anchored, expanded-from-inline and under the fallback,
and the mutation test proved a register-keyed `themeModalHeight` fails
its leg (the window settles {7, 6} where the old algebra settles
{7, 3}).

Superseded as a candidate on 2026-11-06: the plan-closing grilling of
2026-11-05 weighed the follow-up against the arch-bench numbers (the
modal-open cost sits at the overlay compositor, not at the height math)
and the row cache (the steady-frame derivations), found no
user-visible rider, and the maintainer let it die as accepted. The
asymmetry above stays as the byte-preserving fact of the picked theme
box height; the tracker's closing line records the death.

## The ratifications (2026-11-05, PR #128)

The Spec axis ruled the implementer's seven declared deviations within
latitude, none crossing a grilling-ratified line:

- the five test-member-path adaptations (`m.startRow`, `m.inlineWidth`
  twice, `m.fullscreen` twice): the field replacement made them
  uncompilable; only the member paths moved, asserted values unchanged;
- the removed assertion pinning the dead theme write in
  `effects_test.go` (the failed-save pin stays);
- `anchored()` reads true at construction for an inline start: the
  waiting is pinned through `areaHeight() == 0`, because forking on
  anchored-at-construction would flip the View before the geometry
  arrives;
- the report-gated settles (the operations return a bool the handlers
  gate `listWin.settle` on);
- `expand` also refusing when already expanded (unreachable through the
  handler; keeps the operation total);
- `resize` remembering `mainWidth` only while anchored (the old code's
  dead, never-read write on a plain fullscreen start is not
  reproduced);
- the spec's "three start modes" read as the two constructions plus the
  fallback reached by the timeout leg.

The Standards axis found no hard violations; its five judgement calls
closed in the final amendment `b22afa2`: the "register" prose coinage
dropped for the screens/area vocabulary, the model's field comment
thinned to a one-line pointer, `expand`'s refused reasons split
honestly, `fallBack`'s guard owning the once-law itself (`||
noPosition`, two new table rows), and the report prose unified on "the
frame moved".

## The proofs (2026-11-05, PR #128)

- **Selection sweep**: the old mode algebra inlined from
  `git show d2256f8:internal/tui/tui.go` and `theme_modal.go`, driven
  against the new module over the full grid — widths 1-120, heights
  1-40, start rows 0-30 — in both start modes through 13 transition
  legs (resize before and after the position, the timeout fallback,
  expand, collapse, expand again, the mainWidth leg over a wider
  alternate screen, the fallback collapse that never comes back, the
  late answer, expand-before-anything, the round trip,
  expand-before-size, timeout-after-answer), 50 steps per cell:
  **14,880,000 frame-parameter comparisons (height, width, AltScreen),
  0 differing selections**. A difference would have byte-compared the
  frames at both tuples through the untouched `buildFrame(h, w).paint()`;
  none fired.
- **Frame byte sweep**: real models driven through real messages, the
  old selection painted through the shared `buildFrame`, pinned against
  `View()` bytes and AltScreen at every step, the list and theme windows
  replayed through the old settle algebra — 4 contents (task rows, the
  1000-task block, the warning box, the empty state) × 13 legs × the
  sparse edge grid × 2 start modes: **865,280 driven states, 0 byte
  differences**.
- **Binary check**: both builds invoked as `wtr-preview` with a fresh
  `XDG_CACHE_HOME` per run — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B) `cmp` identical against
  `main` `d2256f8`.
- **Permanent tests**: `display_test.go` — 5 top-level tests, 52 rows
  (construction derives, the named facts and the screen fork, the five
  operations settle, the refusal reports, the reads as facts). The
  referee view tests pass with asserted bytes unmodified. `task all`
  green; 311 internal/tui tests with `-race -shuffle=on` (coverage
  86.7% → 87.6%); the glyph pins unmodified.
- **The review verdicts.** The Spec axis: conformant, nothing missing,
  wrong or creeping. The Standards axis: no hard violations, the
  judgement calls closed above.

## What landed (2026-11-05, PR #128)

- `b22afa2` owns the display modes in one module: 6 files, +678/-133 —
  `display.go` (176 lines), `display_test.go` (445), `tui.go` (mostly
  deletions), `theme_modal.go`, `effects_test.go` (-3), `tui_test.go`
  (16). `task all` green at that commit.
- `cbdde4f` merged it into `main`. The piece-6 bookkeeping had landed
  first, with PR #127 (merge commit `b2eaed7`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

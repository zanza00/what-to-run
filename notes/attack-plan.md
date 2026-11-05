# Attack plan (2026-11-04)

The execution tracker for the twelve deepening candidates of the 2026-11-04
review ([arch-review-2026-11](arch-review-2026-11.md)) plus the follow-up
piece the 2026-11-05 grillings added (row 13). The order, the
cadence per piece and the cross-piece rules live in that page's "The attack
plan" section; this page tracks where the work has arrived. Every status
below carries the date it became true.

| # | Piece | Strength | Status | Notes |
| --- | --- | --- | --- | --- |
| 1 | Warning identity (core ↔ tui) | Strong | done (2026-11-04, PR #116, cc3d6de) | the decisions, the two post-review ratifications and the spec pointer in [warning-identity](warning-identity.md) |
| 2 | FP7 status as data | Strong | done (2026-11-04, PR #118, 7e09ed2) | user-visible fix (ctrl+t preview): CHANGELOG entry; closes the notice.go:15 drift; the decisions live in [status-as-data](status-as-data.md) |
| 3 | FP2 frame as structure (+ empty state) | Strong | done (2026-11-04, PR #120, 32f6564) | byte-identical: the 165,060-comparison sweep found zero differences; patch A landed with it; the decisions live in [frame-as-data](frame-as-data.md) |
| 4 | FP6 windowing module | Strong | done (2026-11-05, PR #122, df6870b) | the modal-open re-checks moved inside the module read; the decisions live in [windowing](windowing.md) |
| 5 | FP5 list module (absorbs the snapshot) | Worth exploring | done (2026-11-05, PR #124, 1662dc6) | the measured row cache rides its own follow-up; the decisions live in [list-module](list-module.md) |
| 6 | FP3 row painter | Worth exploring | done (2026-11-05, PR #126, d2256f8) | the three selection laws own one home in `row.go`; the decisions live in [row-painter](row-painter.md) |
| 7 | FP1 display-mode module (+ Preview) | Strong | done (2026-11-05, PR #128, cbdde4f) | Preview crosses the same door as the messages; the decisions and the themeModalHeight finding live in [display-mode](display-mode.md) |
| 8 | Parser frame shared (core) | Strong | not started (2026-11-04) | stops the eleventh copy; tickets likely |
| 9 | justfile: one quote scanner + split | Strong | not started (2026-11-04) | the 81-case table must pass unchanged |
| 10 | Discovery: one read, one walk | Worth exploring | not started (2026-11-04) | inside Discover; no new interfaces |
| 11 | Core test friction | Worth exploring | not started (2026-11-04) | the name→argv rule asserted once; label-keyed registry mutations |
| 12 | Files `~/` exception | Speculative | not started (2026-11-04) | may close as an accepted exception, no code |
| 13 | Measured row cache (arch-bench patch B) | follow-up (grilled) | done (2026-11-05, PR #130, 35ba523) | the derivations live in the era-keyed cache; the suite landed at the merged seams; the decisions and the two ratifications live in [row-cache](row-cache.md) |

Update discipline: when a piece moves, update its status with the date and
the PR number, and add one dated line to the progress log naming the commit
that made it true. The TUI pieces stack on each other (the order is the
review page's); the core pieces are independent and can interleave.

To resume work: read this table, then the candidate details in
[arch-review-2026-11](arch-review-2026-11.md), then
[agent-process](agent-process.md) for how agents work here.

## Progress log

- 2026-11-04: the plan drafted; all twelve pieces not started. The review
  record and the measurements live in
  [arch-review-2026-11](arch-review-2026-11.md); the benchmark source and
  the baselines in [arch-bench-2026-11](arch-bench-2026-11.md). Piece 1
  (warning identity) is next.
- 2026-11-04: piece 1 grilled — the eight decisions (typed warning, five
  kinds, key grammar, granularity, dedup, typed theme guard, lookup-duale
  migration, scope riders) recorded in
  [warning-identity](warning-identity.md); the spec is published to the local
  tracker (`ready-for-agent`); implementation is next.
- 2026-11-04: piece 1 shipped — PR #116 landed on `main` with merge commit
  `cc3d6de` (typed warnings across the seam, kind + dismissal key, the
  lookup duale, the triple cache read retired). The maintainer ratified the
  two edge cases the code review surfaced, recorded in
  [warning-identity](warning-identity.md); the spec survives at
  `.scratch/warning-identity/spec.md`.
- 2026-11-04: piece 2 shipped — PR #118 landed on `main` with merge commit
  `7e09ed2` (the status held as data, a paint kind and a plain text, painted
  by the help line at render time, so the ctrl+t preview restyles it; the
  notice.go:15 comment drift closed with it). The decisions live in
  [status-as-data](status-as-data.md); the spec survives at
  `.scratch/fp7-status-as-data/spec.md`.
- 2026-11-04: piece 3 shipped — PR #120 landed on `main` with merge commit
  `32f6564` (the frame held as data: the styled rows, the placed boxes and
  the list row count; `buildFrame` the only writer, `paint` at the View
  edge; patch A in-piece, `warningModalY` pure arithmetic over the frame
  facts; the empty screen one fact read by name by its four sites; the
  sweep proved the bytes identical at 165,060 comparisons). The decisions
  live in [frame-as-data](frame-as-data.md); the spec survives at
  `.scratch/fp2-frame-as-structure/spec.md`.
- 2026-11-05: piece 4 shipped — PR #122 landed on `main` with merge commit
  `df6870b` (the three list windows in one module: the window holds the
  cursor and the offset, its operations settle every mutation, the renders
  read `visibleRange` and the warning box reads `row`; the six remembered
  call sites became named settles; the warning list entered as the
  height-one instance with the advance policy staying in
  `applyWarningDismissed`). The decisions live in [windowing](windowing.md);
  the spec survives at `.scratch/fp6-windowing/spec.md`.
- 2026-11-05: piece 5 shipped — PR #124 landed on `main` with merge commit
  `1662dc6` (the task list owns the tasks, the filter text and their derived
  facts in one module: built at construction, rebuilt unconditionally at
  every mutation, so nothing derived can go stale; the valid flag, its two
  mirror fields, the three read fallbacks, `taskMatches` and the recompute
  benchmark died with the stale world they served; the tests assert at the
  module's seam). The decisions live in [list-module](list-module.md);
  the spec survives at `.scratch/fp5-list-module/spec.md`.
- 2026-11-05: piece 6 shipped — PR #126 landed on `main` with merge commit
  `d2256f8` (the row assembly owned by one painter: `row.go` assembles both
  branches of the task and theme rows, so the per-segment law, the
  never-dim law and the pad law have one home; `truncateMatched` and
  `renderMatched` moved beside the painter and gained their first direct
  tables; the SGR pin extends to the selected theme row). The decisions
  live in [row-painter](row-painter.md); the spec survives at
  `.scratch/fp3-row-painter/spec.md`.
- 2026-11-05: piece 7 shipped — PR #128 landed on `main` with merge commit
  `cbdde4f` (the display modes owned by one module: `display.go` holds the
  three mode facts, the six geometry facts and the derivations; five
  operations — resize, learnPosition, fallBack, expand, collapse — settle
  every mutation; two named facts, startedInline with anchored and
  expanded, replace the fifteen recombinations of the raw booleans;
  Preview crosses the same door as the messages; restoreCursor un-nests
  from the model, and the write-only `m.opts.Theme` dies). The decisions
  and the themeModalHeight finding live in [display-mode](display-mode.md);
  the spec survives at `.scratch/fp1-display-mode/spec.md`. The measured
  row cache (row 13) is next.
- 2026-11-05: row 13 shipped — PR #130 landed on `main` with merge commit
  `35ba523` (the row derivations keyed by the listing era: one rowCache
  owns the pre-paint derivations, emptied wholesale at every era change
  — the listing generation and the width of the frame — with the paint
  applying the current styles every frame; the list module counts the
  rebuilds as its generation; the arch-bench suite landed at the merged
  seams with cache legs; steady frames −12.5…−27.7% plain,
  −29.3…−41.9% filtered, bytes −67.8…−81.1%, zero byte differences).
  The decisions, the two maintainer ratifications and the benchstat
  numbers live in [row-cache](row-cache.md); the spec survives at
  `.scratch/row-cache/spec.md`. The TUI pieces are done: the core
  pieces 8-11 are next.

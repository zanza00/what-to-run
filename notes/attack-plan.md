# Attack plan (2026-11-04)

The execution tracker for the twelve deepening candidates of the 2026-11-04
review ([arch-review-2026-11](arch-review-2026-11.md)). The order, the
cadence per piece and the cross-piece rules live in that page's "The attack
plan" section; this page tracks where the work has arrived. Every status
below carries the date it became true.

| # | Piece | Strength | Status | Notes |
| --- | --- | --- | --- | --- |
| 1 | Warning identity (core ↔ tui) | Strong | done (2026-11-04, PR #116, cc3d6de) | the decisions, the two post-review ratifications and the spec pointer in [warning-identity](warning-identity.md) |
| 2 | FP7 status as data | Strong | done (2026-11-04, PR #118, 7e09ed2) | user-visible fix (ctrl+t preview): CHANGELOG entry; closes the notice.go:15 drift; the decisions live in [status-as-data](status-as-data.md) |
| 3 | FP2 frame as structure (+ empty state) | Strong | done (2026-11-04, PR #120, 32f6564) | byte-identical: the 165,060-comparison sweep found zero differences; patch A landed with it; the decisions live in [frame-as-data](frame-as-data.md) |
| 4 | FP6 windowing module | Strong | not started (2026-11-04) | carries the modal-open re-checks |
| 5 | FP5 list module (absorbs the snapshot) | Worth exploring | not started (2026-11-04) | open: the measured row cache in-piece or as a follow-up |
| 6 | FP3 row painter | Worth exploring | not started (2026-11-04) | SGR pin for both rows; first direct tests for truncateMatched/renderMatched |
| 7 | FP1 display-mode module (+ Preview) | Strong | not started (2026-11-04) | the widest cut, after FP2; carries the write-only `m.opts.Theme` |
| 8 | Parser frame shared (core) | Strong | not started (2026-11-04) | stops the eleventh copy; tickets likely |
| 9 | justfile: one quote scanner + split | Strong | not started (2026-11-04) | the 81-case table must pass unchanged |
| 10 | Discovery: one read, one walk | Worth exploring | not started (2026-11-04) | inside Discover; no new interfaces |
| 11 | Core test friction | Worth exploring | not started (2026-11-04) | the name→argv rule asserted once; label-keyed registry mutations |
| 12 | Files `~/` exception | Speculative | not started (2026-11-04) | may close as an accepted exception, no code |

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

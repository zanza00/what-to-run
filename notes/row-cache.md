# Row cache (row 13)

The per-row derivations become a cache beside the painter: one new file,
`internal/tui/row_cache.go`, owns what `taskRow` used to re-derive for
every visible row on every frame, keyed by the row's position in the
ranked slice and the listing era; the paint keeps applying the current
styles every frame, so the selection, the theme and the ctrl+t preview
never invalidate. Grilled with the maintainer on 2026-11-05 (eight
questions, every recommendation accepted); the synthesis lives in the
local tracker (`.scratch/row-cache/spec.md`); this page is the durable
record. Row 13 of the [attack plan](attack-plan.md) — the follow-up the
2026-11-05 grillings ratified (FP5's Q2, FP3's Q4). Shipped on
2026-11-05 as PR #130 (merge commit `35ba523`, the single implementer
commit `03aa6b2`).

## What was broken (all at 2026-11-05, at `cbdde4f`)

- Every frame re-derived, for every visible row, inputs that did not
  change. A steady frame — a cursor move, a timer tick, the warning box
  scrolling, the ctrl+t theme preview — ran `taskRow`
  (tui.go:1386-1435) for each visible row, and `taskRow` re-derives the
  badge width (:1391), the `cmdW` cap (:1403), the command's
  `matchPositions` + `truncateMatched` + pad `Repeat` (:1405-1406), the
  rc-task description rule with `flattenDesc` (:1414-1423) and the
  description's `matchPositions` + `truncateMatched` (:1421). The
  selection and the theme change only the styles, but the derivations
  paid anyway.
- The 2026-11-04 review had measured it
  ([arch-bench-2026-11](arch-bench-2026-11.md)): `taskRow` is 59.3% of
  a steady modal-closed frame; a plain row costs 8.78 µs, a
  selected-filtered one 29.28 µs. Its throwaway patch B — cache the
  per-row derivations — measured steady frames −12.5…−28.6% plain and
  −30…−44.5% filtered, allocated bytes −76…−80%, and the grillings
  (FP5's Q2, FP3's Q4) had reserved the piece for after FP1.

## The decisions (2026-11-05)

- **Steady frames only.** The keystroke path stays out (setFilter's
  measurement legs exist for reference); the modal-open compositor cost
  stays out — patch A landed with piece 3.
- **The cache owns exactly the pre-paint derivations**: the badge
  width, the `cmdW` cap, the truncated masked padded command, the
  rc-rule and `flattenDesc` outcome, the description line with its
  mask. The branch assembly — the marker, the `rowStyles`, the pieces,
  `paintRow` — stays outside and runs fresh every frame.
- **`rowCache` in `internal/tui/row_cache.go`, the model's field
  `m.rowCache`, beside the painter.** The entry keys on the row's
  position in the ranked slice plus the listing era — the pair of the
  listing generation and the width of the frame, bundled as the `era`
  type (a review correction: the pair travelled through four call
  shapes before it got its type). `listLines` is the only consumer;
  `deriveRow` is the single derivation function — the miss and the
  fill run the same code, no fallback lane.
- **The list module owns the invalidation fact**: a generation counter
  bumped inside `rebuild()`, which both door mutations (`setFilter`,
  `setTasks`) call, read through `generation()` at the seam. Every
  task-list change goes through those two (`applyThemeSaved` included
  via `setTasks`), so no other site can forget an invalidation.
- **Wholesale eviction at the era change — the FP5 pattern.** Any
  change of (generation, width) empties the cache as a whole: no
  per-entry validity flags, no partial eviction, no freshness checks.
  The fill is lazy, at the row's first paint of an era.
- **The benchmark suite lands.** The arch-bench source, adapted to the
  merged seams, became `internal/tui/render_bench_test.go` with the
  benchmark names kept where the workload survived, plus
  `BenchmarkRowCacheSteadyFrame` (the hit era) and
  `BenchmarkRowCacheFirstFrame` (the miss) at 10/100/1000 tasks.
  Manual `go test -bench` only, never in `task all` or CI.
- **The acceptance is the paired benchstat** — old build against new on
  the same machine, `-count=10 -benchtime=0.5s -benchmem`, medians —
  coherent with the patch-B ratios, no frame leg degrading beyond
  5% noise.
- **The byte proofs run in full**, with legs that cross the
  invalidations (the typed filter mid-capture, resize walks, a
  saved-theme relisting, a deep scroll, plain cursor moves over a hit
  era), plus the binary check — the standard recipe.

## The maintainer ratifications (2026-11-05, PR #130)

- **The uncached row legs.** `BenchmarkTaskRow` measures
  deriveRow+paint — the miss lane's atomic unit — and its plain leg
  reads +11.1% on the review's Intel N150 (the filtered variants +6.7
  and +8.3%). Proven micro-architectural, not allocation: identical
  704 B and 18 allocations per run, an order-controlled A/B ladder that
  refutes thermal order, and a scalar-sink control showing that any
  materialization of the six derived values costs about a microsecond
  on that machine. No steady frame takes that lane, and the miss lane
  at frame scale holds within the gate (FirstFrame +1.0/+3.4/+2.8%).
  The maintainer accepted the exception and recorded it: the 5% gate
  reads frame-scale; the atomic legs stay as reference instrumentation
  carrying the note.
- **The era bound.** Wholesale eviction plus lazy fill means the
  entries follow the rows the era visited — worst case one entry per
  ranked row (about 0.5 MB at 1000 tasks), emptied by the first
  keystroke or resize — and not the visible window the first draft of
  the spec claimed. The maintainer accepted the honest bound; the
  module header, the commit message and the record state it exactly.

## The proofs (2026-11-05, PR #130)

- **Frame byte sweep**: the old paths inlined verbatim from
  `git show cbdd4f:internal/tui/tui.go` (byte-verified), compared raw,
  never ANSI-stripped — 64,548 frame comparisons, zero differences:
  the closed grid h 1-40 × w 1-100 over five task shapes × three cursor
  walks (60,000), the 1000-task block with a 60-step deep scroll
  (4,060), the boxes at the sparse edges (384), the typed filter at two
  widths (36), resize walks (7), saved-theme relistings including an
  empty-tasks save (3), cursor moves over a full window (45), the
  ctrl+t preview (9), the inline ctrl+f flip (4). Every comparison
  also pins miss-frame == hit-frame bytes. The amended era mechanics
  re-ran the era legs: 106 comparisons, zero differences.
- **Binary check**: both builds invoked as `wtr-preview` with a fresh
  `XDG_CACHE_HOME` per run — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B) `cmp` identical against
  `main` `cbdde4f`.
- **The cache table** (`row_cache_test.go`, 10 legs, mutation teeth
  verified live): the fill pins on hand-worked budgets; the hit serves
  a poisoned entry with no rebuild; the era wholesale-empties through
  the real doors — `setFilter`, `setTasks`, a width change — and the
  era legs prove a stale poison is never served; dropping the
  wholesale empty panics the first fill.
- **Paired benchstat** (old `cbdde4f` build against new, Intel N150,
  medians): steady frames **−12.5…−27.7% plain, −29.3…−41.9%
  filtered** (patch-B reference −12.5…−28.6% / −30…−44.5%);
  allocated bytes of the filtered steady frames **−67.8…−81.1%**
  (reference −76…−80%); ListLines −23.0…−25.2%; the modal-open legs
  −4.5…−11.3% where significant; geomean −13.6%. The keystroke
  reference legs stayed flat. `task all` green; 913 tests with
  `-race`.

## The review verdicts (2026-11-05, PR #130)

The Spec axis: conformant, nothing missing or wrong — the four extra
table legs (each pinning a stated decision) ruled benign; the one
test-line adaptation within the pre-agreed call-shape latitude. The
Standards axis: two hard violations at the first commit — a comment
naming a symbol that exists nowhere (`benchTableSizes` for
`windowLegTasks`), and a benchmark comment still claiming the
pre-piece-3 render-twice world — closed in the amendment, together
with the judgement calls: "budget width" disentangled from row.go's
budget (the width of the frame now), list.go's doc separating the
generation from the derived facts, the era type with `openEra`, the
`gen` fields checked against their prose, and one honest refusal: the
tui_test miss-path duplication must keep the direct `deriveRow` call,
because `listLines` paints the cursor row in the selected branch while
the pin reads the plain one. "Era" stays out of GLOSSARY by decision:
module-internal mechanics vocabulary, never user-facing.

## What landed (2026-11-05, PR #130)

- `03aa6b2` keys the row derivations by the listing era: 6 files,
  +841/-48 — `row_cache.go` (150 lines), `row_cache_test.go` (318),
  `render_bench_test.go` (306, the landed suite), `list.go` (30, the
  generation), `tui.go` (77, the rewiring), `tui_test.go` (8). `task
  all` green at that commit; 913 tests with `-race`.
- `35ba523` merged it into `main`. The piece-7 bookkeeping had landed
  first, with PR #129 (merge commit `7272ac8`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

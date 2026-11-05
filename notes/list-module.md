# List module (piece 5)

The task list becomes a module that owns the tasks, the filter text and the
facts they derive — the tokens, the ranking and the command width — built
once at construction and rebuilt unconditionally at every mutation, so
nothing derived can go stale and every read is a plain return of a fact.
Grilled with the maintainer on 2026-11-05; the synthesis lives in the local
tracker (`.scratch/fp5-list-module/spec.md`); this page is the durable
record. Candidate: "FP5 list module (absorbs the snapshot)" in
[arch-review-2026-11](arch-review-2026-11.md), piece 5 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #124 (merge
commit `1662dc6`, the single implementer commit `98a3e77`).

## What was broken (all at 2026-11-05, at `df6870b`)

- The filter snapshot re-derived its own identity at every read: the
  `filterSnapshot` struct (tui.go:846-853) carried a `valid` flag and two
  mirror fields, `filter` and `taskCount`, whose only job was to check what
  the model already promised — `snapshotValid` (:859-862) ran three
  comparisons per read, and both writers stated the promise outright.
- Three read paths carried that check with production-dead fallbacks:
  `filterTokens` (:897-902, the re-parse), `visible` (:907-912, the
  re-rank) and `commandWidth` (:1701-1708, which re-ranks and re-widths
  through `visible` again). Both writers — `setFilter` (:866-870) and
  `setTasks` (:874-878) — rebuild unconditionally ("the cache cannot go
  stale"), so the fallbacks were unreachable in production.
- The tests kept the dead half alive by crossing the interface:
  `tui_test.go:845` and `:890` read `m.snapshot.valid` and
  `m.snapshot.tasks` directly, and `BenchmarkRenderFilteredRecompute`
  (tui_bench_test.go:43-50) zeroed the `valid` flag on every iteration to
  measure the pre-snapshot world — 66.6 ms against 694 µs at 400 tasks,
  the 96× of the [arch-bench-2026-11](arch-bench-2026-11.md) baselines.
- `taskMatches` (tui.go:946-950) wrapped `taskMatchScore` to drop its
  score and had zero callers.
- The state sat in three model fields — `filter`, `tasks`, `snapshot` —
  with a pairing rule every consumer had to know, and the filter text was
  read at six sites beyond its writer (tui.go:540, 613, 620, 1410-1413,
  1467-1468, 1740).

## The decisions (2026-11-05)

- **The list is a module.** `taskList` in `internal/tui/list.go` holds the
  tasks, the filter text and the shell dialect beside the derived facts
  (the tokens, the ranked slice, the command width). `newTaskList` derives
  once at construction; the two mutations — `setFilter(text)` and
  `setTasks(tasks)` — rebuild unconditionally, each fact from the one
  before it (the tokens, then the ranking over the full task list, then
  the width over the ranked tasks). Staleness is impossible by
  construction: the `valid` flag, both mirror fields and the three
  fallbacks die with nothing left to verify.
- **The reads are plain returns.** `visible` returns the shared ranked
  slice — no defensive copy, today's measured behavior — and
  `filterText`, `filterTokens`, `commandWidth` and `taskCount` return
  facts. The six filter-text read sites (the esc case, the typing and
  backspace paths, `filterLine`, the `no task matches` row, the help
  counts) go through the module's reads.
- **The model stays the composer with one door.** Its `setFilter` and
  `setTasks` keep their names and delegate: `setFilter` writes through the
  list and then resets the task window to its top (`listWin.top()`), the
  window staying the model's per piece 4. The model's fields `filter`,
  `tasks` and `snapshot` die; one module field remains. The invariant has
  one owner (list.go's header); the model's comments point at it.
- **The pure functions stay free functions with their tests untouched.**
  `rankTasks`, `taskMatchScore` and `commandWidthFor` keep their bodies,
  their behavior and their direct tests; the module calls them without
  absorbing them. `taskMatches` dies (zero callers).
- **The measured work stays out, by decision.** The keystroke-path cost
  (`setFilter` 4.73 µs at 100 tasks rising to 4.83-6.71 ms at 1000:
  scoreSubsequence 41%, GC 15%, commandWidthFor 10%) and the per-row
  derivations cache (arch-bench patch B, −12.5…−44.5% on steady frames,
  bytes −76…−80%) ride their own follow-up spec with the baselines of
  [arch-bench-2026-11](arch-bench-2026-11.md); this piece freezes
  behavior, not cost.
- **The tests assert at the module's seam.** The two crossed tests
  (`tui_test.go:838-850` and `:884-895` shapes) now read the module after
  every mutation with `rankTasks`/`commandWidthFor` recomputation as the
  oracle. `list_test.go` pins the module directly: construction derives,
  `setFilter` rebuilds from the new text, `setTasks` rebuilds from the
  new slice with the filter held, and repeated reads share the ranked
  slice.
- **The stale world's tests died with the stale world.**
  `BenchmarkRenderFilteredRecompute` and `TestModelWithoutSnapshotRenders`
  are no longer constructible — the premise they pinned (a model without
  a valid snapshot) cannot exist. Both died; the remaining benchmark legs
  keep their names and workloads, measuring a world that still exists.
  No GLOSSARY entry: the piece adds no new domain concept.

## The proofs (2026-11-05, PR #124)

- **Frame byte sweep**: the old `listLines`+`buildFrame` path inlined
  verbatim from `git show df6870b:internal/tui/tui.go`, compared raw,
  never ANSI-stripped — 279,040 comparisons over the closed variants at
  every h 1-40 × w 1-100, the inline variants, the box variants at the
  sparse edges and the optional 1000-task pass: zero differences. The
  negative control: with `cmdW = 0` injected, the sweep fails on its
  first state — the harness is sensitive, not vacuous.
- **Binary check**: both builds invoked as `wtr-preview` with a fresh
  `XDG_CACHE_HOME` per run — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B, the warning modal open at
  start) `cmp` identical against `main` `df6870b`.
- **The review verdicts.** The Spec axis: matches, nothing missing, wrong
  or creeping; the three declared deviations ratified. The Standards
  axis: no hard violations; the judgement calls closed in the final
  commit — the duplicated comment facts thinned to one owner plus a
  pointer, the test-oracle formula given one owner (`filterTokensOf`),
  the model's `setTasks` door kept as the spec pinned, the dual `shell`
  holder verified as set-once-from-one-source, and the two-theme sweep
  density accepted as the three-piece precedent.

## What landed (2026-11-05, PR #124)

- `98a3e77` owns the task list and its derived facts in one module: 9
  files, +310/-213 — `list.go` (94 lines), `list_test.go`, the model
  rewired, six test files. `task all` green at that commit; 234
  internal/tui tests with `-race -shuffle=on`.
- `1662dc6` merged it into `main`. The piece-4 bookkeeping had landed
  first, with PR #123 (merge commit `9b90b0c`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

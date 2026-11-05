# Core test friction (piece 11)

The two core test tables stop hiding their intent: the run-prefix rule
(a justfile task runs as `just <name>`) is asserted once in a `justTask`
helper, and the registry invariants mutate their sources through an
ID-keyed lookup, so the mutation names what it breaks regardless of the
registry order. Grilled with the maintainer on 2026-11-05 (eight
questions, every recommendation accepted; the counts were verified per
site on the merged `main` before the grilling, applying the
agent-process gotchas of the same day); the synthesis lives in the
local tracker (`.scratch/core-test-friction/spec.md`); this page is the
durable record. Candidate: "Core test friction" in
[arch-review-2026-11](arch-review-2026-11.md), piece 11 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #138 (merge
commit `d5ae222`, the single implementer commit `5a7a489`).

## What was broken (all at 2026-11-05, at `7fe6a3b`)

- The run prefix restated 124 times, verified couple-for-couple: every
  want task in the justfile tests carried the argv pair
  `[]string{"just", <its own name>}` — 120 sites in
  `source_justfile_test.go` (the 77-case `TestParseJustfile` table, the
  import/module/dot tests and the two `want :=` blocks; 90 with an
  inline `Desc`, 28 with it omitted, 2 with a raw-string `Desc`, 6 of
  them carrying a badge `Source`) and 4 badge-carrying sites in
  `sources_test.go` (:29, :514, :523, :538). The rule never deviated —
  and nothing shared it.
- The registry table mutated by index at every case: 19 mutation cases
  in `TestRegistryInvariants` (`sources_test.go:79-176`), each writing
  `srcs[<position>]…`. A registry reordering moves the index under a
  case: the mutation still runs, but it silently breaks a different
  source than its name claims.

## The decisions (2026-11-05)

- **One piece, both frictions**: a test-only refactor of the same
  register, one worktree, one proof, one pull request. The production
  diff stays empty as a hard gate, not an expectation.
- **The `justTask` helper**: `justTask(name, desc string) Task` in
  `testhelpers_test.go` (the shared test helpers file) owns the rule
  once, in its comment; all 124 restatements become calls, and after
  the piece the helper holds the only `[]string{"just", name}` in the
  package.
- **The ID-keyed mutations**: `sourceByID(srcs []source, id string)
  *source` lives in `sources_test.go` beside its only consumer, and the
  per-source cases read their intent through it. The ID is the key, not
  the review's first word "label": the rc disable list keys by ID, the
  invariant engine validates IDs first, and the want-messages name IDs.
- **The positional cases stay positional, named honestly**: the four
  cases whose point is a position — the swapped pair, the
  out-of-registry-order IDs, the missing-source slice, the
  extra-source append — keep their indices, each with a comment
  stating what that position tests, and the law stated once at the
  table's home.
- **The want-messages do not move**: they are `checkRegistry`'s output
  strings, and the production is untouched.
- **The proof**: a two-leg harness over the old literals, plus the
  binary check run anyway, plus `task all`.
- **The names follow the persisting rule**: `justTask` and
  `sourceByID`, checked against the package's words before the first
  commit; no GLOSSARY entry (test-only vocabulary; the word *badge*
  here is the task's Source stamp, not the dismissal-key avoid-term).

## The seam shapes the ratification forced (2026-11-05, endorsed)

- `badgeTask(badge string, task Task) Task` — the badge variant the
  spec licensed as the implementer's seam shape. It became a shared
  helper in `testhelpers_test.go` because the badge sites turned out
  to be ten across the two files (the spec's count of four named only
  `sources_test.go`); it touches only `Source`, so the run prefix
  stays in `justTask` alone.
- `sourceByID` returns a pointer and panics on a missing ID: a typo'd
  ID in a test fails loudly instead of mutating nothing.

## The proofs (2026-11-05, PR #138)

- **The harness, two legs, deleted before the commit** (a
  `go/parser` generator inlined the old want literals from
  `git show 7fe6a3b:…`):
  - leg one: **124/124** sites — the OLD want literal `DeepEqual` the
    helper encoding of the same case (the ten badge sites with their
    `Source` set over the return). The generator also re-asserted that
    every old argv pair matched its own task's name: no deviation.
  - leg two: **19/19** cases — the OLD index-keyed mutation and the
    NEW mutation, each over a fresh clone of the registry, produce the
    same `checkRegistry` error strings, byte for byte.
  - **Zero tolerated differences on both legs.**
- **The honest corrections, reported not absorbed**: the never-fail
  table (`TestParseJustfileNeverFails`) holds inputs only, zero want
  sites — the 124 still parsed; and the badge sites are ten, not the
  four the review counted — six live inside
  `source_justfile_test.go`'s 120. Both numbers above already carry
  the corrections.
- **The hard gate held**: `git diff main --stat` listed only the
  three `_test.go` files; the production diff is empty.
- **The binary check, run anyway** (the standing probe): both builds
  as `wtr-preview` under a fresh `XDG_CACHE_HOME` — `--preview`
  (2,417 B) and `--preview --theme nonexistent` (2,578 B)
  byte-identical against `7fe6a3b`.
- `task all` green; 558 tests pass in `internal/core` under `-race`.
- **The review verdicts.** The Standards axis: clean — one minor STE
  finding (three compound sentences coupling a cause and its effect
  with *and*/*so*) split into short sentences in the amendment, the
  "keeps its indices" law at the table's home; the `badgeTask` shape
  judged honest, its comment defensible as reader-facing prose. The
  Spec axis: nothing missing — `badgeTask` weighed as fidelity to the
  ratified 124-conversion scope, `sourceByID`'s loud miss accepted,
  the index→ID mapping verified faithful at every case against
  `defaultSources`' order.

## What landed (2026-11-05, PR #138)

- `5a7a489` asserts the just run prefix once and keys the registry
  mutations by ID: 3 files, +208/-152 — `testhelpers_test.go`
  (`justTask` + `badgeTask`), `source_justfile_test.go` (the 120
  conversions), `sources_test.go` (the 4 badge conversions,
  `sourceByID`, the 15 keyed mutations, the four commented positional
  cases). `task all` green at that commit.
- `d5ae222` merged it into `main`. The piece-10 bookkeeping had
  landed first, with PR #137 (merge commit `556ebc9`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

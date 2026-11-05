# Parser frame (piece 8)

The parser skeleton stops being copied: the discover seam owns the
conventions every source follows — an empty file reports like a
missing file, once; the tasks of a source arrive sorted by name, once;
the help-else-body description chain lives in one helper; and the
nil-drift deviants return to the sibling convention. Grilled with the
maintainer on 2026-11-05 (eight questions, every recommendation
accepted); the synthesis lives in the local tracker
(`.scratch/parser-frame/spec.md`); this page is the durable record.
Candidate: "Parser frame copied eleven times" in
[arch-review-2026-11](arch-review-2026-11.md), piece 8 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #132 (merge
commit `cfc203a`, the single implementer commit `4ad3e5b`).

## What was broken (all at 2026-11-05, at `35ba523`)

- The empty-file guard — `if len(bytes.TrimSpace(data)) == 0 { return
  nil, nil }` with its law ("an empty file declares no tasks, so
  report it like a missing file") — was verbatim in eleven parsers
  (source_cargo_config.go, source_composer.go, source_deno.go,
  source_devbox.go, source_mise.go, source_pdm.go, source_pipenv.go,
  source_poe.go, source_taskfile.go, source_taskipy.go,
  source_tox.go).
- `slices.Sorted(maps.Keys(...))` appeared eighteen times — one
  convention (tasks arrive sorted by name) re-stated at every map.
- The help-else-body description chain (the help wins; the body is
  display text, never a promise of the executed command) lived in six
  dialects (source_pdm.go:100, source_poe.go:123, source_taskipy.go:84
  and the mise, deno and hatch shapes).
- The drift the review predicted existed:
  source_package.go:47 (`make([]Task, 0, ...)`) and its twin in
  source_cargo_make.go returned an empty slice where every sibling
  returned nil.
- The seam that would own all of it was one call site:
  `src.Parse(filepath.Dir(path), name, data)` in discover.go:268.

## The decisions (2026-11-05)

- **Framework side only**: the conventions move to the discover seam;
  no type hierarchy (the review's law — hatch and tox deform one), no
  `Parse` signature, source registry or metadata change.
- **The empty-file guard moves to the seam** (one place, before
  `Parse` runs); the eleven copies die.
- **The sorted-by-name convention moves to the seam**: `sortSourceTasks`
  orders each source's slice after `Parse`; the parsers deliver their
  document's order.
- **The description chain becomes `helpOrBody(help, body)`** in
  `desc.go`, named by the register's existing word; the dialects call
  it with their own extracted body.
- **The nil drift dies**: the two deviants return `var tasks []Task`,
  nil when nothing; the pins that asserted the empty-non-nil form
  adapt.
- **The proofs**: the fixture-corpus differential with the old parsers
  inlined (zero tolerated differences), the per-source tables
  continuing, the repo-root `wtr-preview` integration check, and
  `task all`. One implementer, no tickets, by the grilling's decision.
- **The naming** follows the persisting rule (`emptyFile`,
  `sortSourceTasks`, `helpOrBody`, `desc.go`), and the spec kept
  "frame" out of core prose: it is the picker render's GLOSSARY term;
  the spec said "the shared skeleton" or "the conventions".

## The discoveries and the ratifications (2026-11-05, PR #132)

Two of the spec's uniform claims were false per site, and the
implementer's differential caught both; the maintainer ratified every
outcome, choosing behavior preservation over mechanism uniformity:

- **npm keeps its empty-file warning.** The spec draft claimed the
  unguarded parsers already produced zero tasks on empty bytes — npm's
  parse has always errored on empty JSON, so an empty package.json
  warned. A uniform guard would have removed a warning; the seam skips
  npm (`src.ID != \"npm\"`), and a new pin asserts "an empty file keeps
  the parse warning".
- **make and hatch are exempt from the seam's sort**, so fifteen of
  the eighteen sorted-keys sites died, not eighteen. The draft claimed
  each parser already sorted its own slice: make delivers the order of
  its file (pinned user-visible — a name sort would reorder the
  picker rows of every Makefile), and a flat name sort flips hatch's
  "default:z" before "default-dev:a" because '-' sorts before ':'.
  taskfile's includes sort stays too: it is first-definition-wins
  determinism, not a delivery convention.
- **The exemption mechanism is three `src.ID` literals at the seam**
  (npm for the guard, make and hatch for the sort), the why stated
  once in the owning comment — honest, local, minimal; the metadata
  alternative would have inflated the registry surface for three
  quirks.
- **`rcTasks` calls `helpOrBody` too** — a seventh site of the same
  law in the seam's own file; and `TestEmptyFile` gained a byte-order
  mark edge. Both benign; both ratified.
- **taskfile's order-dependent cases moved to the seam-driven table**
  within the spec's placement latitude (every case survived
  somewhere, none weakened into set comparisons; its whitespace-only
  case lives at the seam exactly where the tab byte would break YAML).

## The proofs (2026-11-05, PR #132)

- **Corpus differential**: the old parse functions inlined from a
  git-archive of `35ba523`, driven beside the new ones over 106
  entries — every fixture under `internal/core/testdata` (15 drives,
  including the nested taskfile and justfile trees) plus 91 synthetic
  edges (empty and whitespace-only for all 17 sources, the nil-drift
  shapes, declarations without tasks, deformed TOML/YAML/JSON/INI,
  help-else-body edges, the hatch prefix-env edge) — comparing error
  parity and the deep task sets (Name, Desc, Args): **0 differences,
  stable across randomized reruns**. The harness re-ran after the
  review fixes.
- **Integration**: both builds invoked as `wtr-preview` in the repo
  root under a fresh `XDG_CACHE_HOME` — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B) `cmp` identical against
  `35ba523`.
- **Permanent tests**: the seam's tables — `TestEmptyFile` (6 cases,
  the BOM edge included), `TestSortSourceTasks` (5, both exemptions),
  `TestHelpOrBody` (4), `TestDiscoverSourceEmptyFileLikeMissing` (2,
  the parent-search stop) — plus the two deviants' nil pins and the
  npm warning pin. `task all` green; **939 tests with
  `-race -shuffle=on`** across 5 packages (core coverage 97.2%); the
  registry invariants and the cmd/wtr docs tests unmodified.
- **The review verdicts.** The Spec axis: conformant, nothing missing
  or wrong. The Standards axis: no name drift; two hard violations
  closed in the final amendment — the unhedged "every source sorted"
  law (four comments claimed what the exemptions disprove; the law now
  lives once in `sortSourceTasks`'s doc, the rest point at it) and the
  "rules" synonym (the make source says "targets") — plus the
  judgement calls: the hatch prose deduplicated behind its named
  functions, the desc.go header stating its honest scope (the cargo
  alias expansion and the Taskfile desc/summary/command ladder stay
  outside), and the taskfile test prose separating the seam's facts
  from the parser's.

## What landed (2026-11-05, PR #132)

- `4ad3e5b` owns the parser skeleton at the discover seam: 19 files,
  +434/-334 — `desc.go` (13 lines) with its table, `discover.go` (48,
  the two seam functions and their laws), `discover_test.go` (139),
  the eleven guarded parsers minus their guards, the two deviants, the
  six dialects plus rcTasks calling `helpOrBody`, and the taskfile
  test table (180 lines reshaped). `task all` green at that commit.
- `cfc203a` merged it into `main`. The row-13 bookkeeping had landed
  first, with PR #131 (merge commit `ab874ad`).

## The lesson (recorded in agent-process)

Two consecutive pieces caught the same failure at spec time: a spec's
uniform claim about code ("the unguarded parsers produce zero tasks
on empty bytes"; "the entries follow the visible window"; "each
parser sorted its own slice") stated an assumption as a fact, and the
implementer's differential discovered the truth (npm warned; make and
hatch delivered document order). Both times the maintainer ratified
the honest exemptions. From now on, a spec's uniform claims are
verified per site at spec time, or written as the open question the
implementer must settle first.

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

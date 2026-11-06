# Discovery: one read, one walk (piece 10)

Discovery stops multiplying its work per source: one Discover reads one
file at most once, and enumerates the ancestors of the project directory
at most once. A `readCache` wraps the Files port inside Discover — every
ReadFile result, `fs.ErrNotExist` included, is remembered under the
exact path string — and `parentLevels` enumerates the ancestors lazily
(one entry `parentLevel` holds the level directory and the badge prefix),
which each of the eight parent-searching sources consumes in registry
order, stopping at its own find. Grilled with the maintainer on
2026-11-05 (eight questions, every recommendation accepted; the numbers
were re-derived per site on the merged `main` before the grilling,
applying the agent-process gotchas of the same day); the synthesis lives
in the local tracker (`.scratch/discovery-read-walk/spec.md`); this page
is the durable record. Candidate: "Discovery: one read per file, one
ancestor walk" in [arch-review-2026-11](arch-review-2026-11.md), piece 10
of the [attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #136
(merge commit `7fe6a3b`, the single implementer commit `b1c26b7`).

## What was broken (all at 2026-11-05, at `f09fe8a`)

- The five Python sources held `pyproject.toml` among their candidates
  (poe at `source_poe.go:20`, pdm at `source_pdm.go:18`, taskipy at
  `source_taskipy.go:14`, hatch at `source_hatch.go:21`, tox at
  `source_tox.go:21`), so a Python project read and parsed the same
  bytes five times per Discover.
- Eight sources declared `SearchParents` (justfile, taskfile, poe, pdm,
  taskipy, hatch, tox, pipenv), and each ran its own copy of the ancestor
  enumeration inside `discoverSource` (`discover.go:217`: the absolute
  base at `:225`, the level loop at `:232`, the badge prefix inside it),
  re-running the same path math and re-reading the same missing
  candidates at every level. Discovery runs one or two times per run,
  so the review called the growth structural, not a latency problem.

## The decisions (2026-11-05)

- **One piece, both halves**: the read cache and the walk hoist ship
  together, inside Discover, with no new interfaces and no Files
  widening (the review's constraint).
- **The shared parent list**: the levels are enumerated once per
  Discover; each parent-searching source consumes the shared list in
  registry order and stops at its own find, so the order of tasks and
  warnings cannot move. The grouped per-directory walk was rejected —
  it would reorder tasks across depths, a user-visible change.
- **The whole-port wrap**: `Discover` wraps the Files port once, before
  `defaultSources` and the walk, so the parsers' sibling and include
  reads ride the same memo and the read law needs no carve-out. The
  wrapper satisfies the existing Files interface and declares none.
  `Exists` and `ReadDir` delegate untouched.
- **Every result is remembered, misses included**: the memo is keyed by
  the exact path string handed to the port, so the in-memory fakes
  keyed by relative names keep working. The memo lives inside one
  Discover call and dies with it; nothing is cached across runs.
- **The build failure keeps its shape**: when `filepath.Abs` fails,
  every parent-searching source that missed in the project directory
  replays the remembered failure as its own warning — up to eight
  warnings for one broken directory, exactly as the per-source walk
  reported them.
- **The build is lazy and remembered**: the first source that needs the
  parent search enumerates; a source that finds its file in the project
  directory never touches the walk.
- **The names** (the persisting rule): `readCache`, `parentLevels` and
  `parentLevel`, checked against the comments' prose before the first
  commit. "Chain" was rejected on purpose — cargo alias chains and the
  just and task include chains own that word.
- **Two shape choices the ratification forced, blessed in review**: the
  shared state (the cache, the levels, the directory) sits on one
  private type `discovery`, and the old free `discoverSource(files,
  src, dir)` stays as a thin adapter over it — a registry of one —
  because the existing package tests call that exact signature, and the
  spec pinned the existing tables unmodified. The Standards axis judged
  the adapter an acceptable delegation; the Spec axis judged the holder
  as serving the spec.

## The proofs (2026-11-05, PR #136)

- **The corpus differential**: the old `Discover`, `discoverSource` and
  `readSourceFile` inlined verbatim from `f09fe8a`, driven beside the
  new flow over the five `testdata` trees and the spec's synthetic
  edges (a pyproject-only project at depths, sources finding at mixed
  levels, disabled sources, broken files, npm's empty-file warning,
  make's and hatch's document order, relative and absolute
  directories, rc shadowing): **36 comparisons, zero differences**,
  full task sets and full warning sets in order. The harness drove the
  `filepath.Abs` failure too, by deleting the process working
  directory: eight warnings, byte-identical in both flows. Deleted
  before the commit, per the recipe.
- **The committed pins**: a `countFiles` fake over `mapFiles` counts the
  reads that reach the port. `pyproject.toml` is read **once** across
  the five Python sources (was five); a missing candidate of each
  level — the pyproject.toml of the five Python sources plus the
  first candidate of just, task and pipenv, so the eight walkers are
  covered — is read **once** per level (was five or eight); a taskfile
  include stays readable through the wrapper at one port call.
  `parentlevels_test.go` pins the enumeration itself: the build table
  (level directories and badge prefixes, the root-walks-itself edge
  with a dot prefix), the remembered second build (the same backing
  array, so a re-enumeration reddens), the laziness (every walker finds
  in the project directory, so no absolute path reaches the port), and
  the unresolvable-directory replay (eight warnings, `KindSource`,
  keys `source:<id>:.`, registry order, one remembered message). The
  core coverage rose from 97.3% to 97.7%.
- **The platform truth**: the fixed-tree pins resolve drive-less paths
  the way `filepath.Abs` defines, so they skip on Windows with the
  reason in the comment. The suite runs on Linux, and the CI darwin
  legs where `ci.yml` asks for them; Windows gets vet and cross-builds
  only.
- **The shared-slice claim, verified per site**: no parser writes into
  its bytes. Every parser hands `data` to read-only consumers —
  `toml.Unmarshal` (cargo-alias, cargo-make, hatch, mise, pdm, pipenv,
  poe, taskipy, tox), `json.Unmarshal` (package.json),
  `unmarshalJSONC` (which builds its own buffer), `yaml.Unmarshal`
  (taskfile root and includes), `string(data)` line scans (justfile,
  make, tox ini) and `bytes.TrimSpace` — and a mechanical scan for
  writes returned zero hits. The readers of one file share the slice
  of its first read.
- **The integration check**: both builds invoked as `wtr-preview`
  under a fresh `XDG_CACHE_HOME` in the repo root — `--preview`
  (2,417 B) and `--preview --theme nonexistent` (2,578 B) identical
  against `f09fe8a`. `task all` green on the committed tree.
- **The review verdicts.** The Spec axis flagged three gaps, all
  closed in the amendment: the levels rested on the deleted
  differential (now pinned, above), the ancestor pin covered five
  readers instead of the eight walkers (widened), and the hardcoded
  Unix paths were replaced with `filepath.Join`-derived keys. The
  Standards axis flagged the laws stated twice in the `Discover` doc
  (now only the consequence and the naming of the shared state), a
  five-clause sentence in the `parentLevels` doc (split into four
  STE sentences), the read law named once at its home, and a test
  loop duplicated three times (one `taskNames` helper now serves all
  three sites, including the pre-existing `justTaskNames` it deleted).

## What landed (2026-11-05, PR #136)

- `b1c26b7` reads each file once and walks the parents once: 6 files,
  +442/-38 — `discover.go` (the three new types and the rewired flow),
  `readcache_test.go` (98), `parentlevels_test.go` (156),
  `files_test.go` (the `countFiles` fake), `testhelpers_test.go`
  (`taskNames`), `source_justfile_test.go` (the duplicated helper
  dies). `task all` green at that commit.
- `7fe6a3b` merged it into `main`. The piece-9 bookkeeping had landed
  first, with PR #135 (merge commit `f09fe8a`).

## The cost, measured after the plan closed (2026-11-06)

A before/after pair (a portable benchmark compiled unchanged at
`cd1e933` and at the closed `main` `c80c6f4`, the same machine and
method as the bench record, interleaved pairs) measured what the read
cache costs per call:

- The port-call law holds on every leg: the Python project reads
  pyproject.toml once instead of five (92 → 88 calls), and the deep
  all-miss tree drops 24 duplicate misses (139 → 115).
- The memo costs more than the reads it saves at the registry's sixteen
  sources: in-memory +111,2% per Discover on the Python project
  (37,5 → 79,2 µs) and +238,3% on the deep miss tree (40,7 → 137,8 µs);
  on a real temp tree on disk, +43,5% (452,7 → 649,7 µs). The bytes per
  op rise with the memo's entries (+144% and +411%) while the allocs per
  op fall (−5,1% and −6,6%).
- Discovery runs at most twice per run, so the per-run impact stays
  under a millisecond; the win the piece bought is bounded growth — a
  file shared by more sources is still read once, and each new
  shared-candidate reader adds no calls instead of one per level.

The 2026-11-05 decisions stand: the piece closed the row for the
structural law, which the numbers prove true. The TUI halves of the
same pair are recorded in
[arch-bench-2026-11](arch-bench-2026-11.md) (addendum of 2026-11-06).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

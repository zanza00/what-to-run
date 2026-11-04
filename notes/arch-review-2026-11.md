# Architecture review of 2026-11-04

The improve-codebase-architecture run of 2026-11-04 at HEAD `cd1e933` (clean
tree). Three fresh-context subagents: a TUI review, a task-sources review
and a measurement agent (benchmarks and profiles in a throwaway detached
worktree, removed after; `main` untouched). The maintainer asked to
re-validate and incorporate the 2026-11-01 roadmap
([tui-deepening-roadmap](tui-deepening-roadmap.md)), to measure TUI
rendering, and to follow the git hot spots. The HTML report (in Italian, at
the maintainer's request) was served on the LAN and is ephemeral; this page
is the durable record.

## Re-validation of the 2026-11-01 roadmap

All six pending candidates are still true at `cd1e933`. The only
`internal/tui` changes since `main@80c4175` are the box-module stack
(`8674bdb`, `c779816`, `a4c8cc9`, `3a0a419`); none of the six was
accidentally fixed. Fresh line references are in the candidate list below.
The 2026-11-01 "bonus" (the typed warning kind) is now a full candidate: the
core review and the TUI review converged on it from both sides of the seam.

## Candidates (all file:line at `cd1e933`)

### The TUI

- **FP7 status as data** — STRONG. The status is a pre-rendered styled
  string painted at event time at four sites (tui.go:465-471, 729, 772;
  warning_modal.go:90), so the ctrl+t live preview restyles everything
  except the status. Tests strip ANSI to read it (theme_test.go:595-618).
- **FP2 frame as structure** (with the empty state as data) — STRONG.
  `render` joins to a string at tui.go:1273; `warningModalY` re-renders the
  list to measure it (tui.go:1328, the magic `3` duplicating chromeRows);
  the no-tasks screen is re-derived at four sites (tui.go:501, 1348, 1401,
  1667); tests count newline and box glyphs (tui_test.go:495-635).
- **FP6 windowing module** — STRONG. `keepCursorVisible` is remembered at
  six call sites and re-applied defensively at two renders
  (tui.go:1164-1182, 1414-15; theme_modal.go:50-63, 70); the warning list
  has no window at all: `warningIdx` is hand-clamped at five sites
  (tui.go:611-619; warning_modal.go:68, 93, 131).
- **FP1 display-mode module** (with Preview at the seam) — STRONG, the
  widest cut, after FP2. Four display modes are products of three booleans,
  re-derived at thirteen sites (tui.go:319-331 and the read sites at 166,
  411, 428, 445-452, 521, 553-560, 766, 1194, 1228, 1237, 1676, 1692-96);
  `areaHeight` (tui.go:1726-34) and `restoreCursor` (tui.go:273-80) each
  re-derive the inline geometry; `Preview` hand-sets it instead of crossing
  the seam (tui.go:240-43).
- **FP5 list module absorbs the filter snapshot** — WORTH EXPLORING, after
  FP6 (same state). Snapshot validity leaks into three read paths with
  production-dead fallbacks (tui.go:846-53, 856-63, 1641-47);
  `taskMatches` has zero callers (tui.go:895-98); tests cross the interface
  by zeroing `m.snapshot.valid` (tui_test.go:764-82).
- **FP3 row painter** — WORTH EXPLORING. The selected-row segment protocol
  is written twice (tui.go:1489-1518; theme_modal.go:135-141) but the
  SGR-grep pin covers only the task row (theme_test.go:357-393);
  `truncateMatched`/`renderMatched` (tui.go:1520, 1609) still have zero
  direct tests.

### The core ↔ tui seam

- **Warning identity** — STRONG (both reviews converged). `warningKey` is
  the whitespace-normalized message text and doubles as display
  (warning_modal.go:197-202), so a reword silently un-dismisses every
  dismissal; the badge is a relative path (discover.go:258, 262), so the
  same broken file changes identity from another subdirectory;
  `hasThemeWarning` matches the substring "theme" (tui.go:749-57). Side
  finding: the dismissal cache file is read three times per dismiss
  (warning_modal.go:66-73, 233, 234-37).

### The task sources

- **Parser frame copied eleven times** — STRONG. The empty-file guard is
  verbatim in eleven parsers; `slices.Sorted(maps.Keys(...))` appears
  seventeen times; the help-else-body description chain lives in six
  dialects; the drift already exists: source_cargo_make.go:56 and
  source_package.go:46 return an empty slice where every sibling returns
  nil. The conventions belong on the framework side of the parser seam
  (discover.go), not in a type hierarchy: hatch and tox deform one.
- **justfile: five quote scanners and a fused flow** — STRONG (the
  scanners) / WORTH EXPLORING (the split). The same quote-aware byte
  scanner is written five times (source_justfile.go:584, 639, 716, 740,
  763); `parseJustfileFor` fuses pure scanning (justDocumentOf:174),
  Files-port traversal (justCollect:308) and selection policy
  (selectJustTasks:449); the error half of its interface is dead (:145,
  pinned by TestParseJustfileNeverErrors); the real interface is the
  78-line prose comment (:66-143).
- **Discovery: one read per file, one ancestor walk** — WORTH EXPLORING.
  Five sources list pyproject.toml (poe:22, pdm:20, taskipy:20, hatch:20,
  tox:24), so a Python project parses the same bytes five times per
  Discover; eight sources walk the ancestor chain per source
  (discover.go:214-43). Discovery runs 1-2 times per run; the
  per-keystroke filter is in memory (tui.go:812). The growth is
  structural, not a latency problem today.
- **Core test friction** — WORTH EXPLORING. The justfile table restates the
  run prefix 81 times (source_justfile_test.go:13-697); the registry tests
  mutate by index (sources_test.go:82).
- **Files port: the one ~/ reach-past** — SPECULATIVE. `justImportPath`
  resolves `~/` through `os.UserHomeDir` past the seam
  (source_justfile.go:356-66), so the test mutates the process env
  (source_justfile_test.go:1179-84). Do not widen Files for it.

## Measurements (2026-11-04, Intel N150, Go 1.27.3, benchstat medians)

- A modal-closed frame costs 145-500 µs and task count does not matter
  (1000 tasks ≈ 100): windowing and the snapshot absorb it.
- A modal-open frame costs 0.9-3.2 ms, 1.10 MiB/op, 2228 allocs/op; the
  overlay compositor owns it (64% CPU, 83% of the bytes,
  `ultraviolet.NewBuffer`), scaling with width×height.
- Throwaway patch A (count rows instead of re-rendering): modal frames
  −10…−27% time, −23…−41% allocations.
- Throwaway patch B (cache the per-row derivations): steady frames
  −12.5…−28.6% plain, −30…−44.5% filtered (bytes −76…−80%);
  RenderFilteredRecompute 66.6 ms → 4.06 ms, because an invalid snapshot
  re-ranks up to three times per frame.
- The keystroke path is the scalability wall: setFilter costs 4.8-6.7 ms at
  1000 tasks (scoreSubsequence 41%, GC ~15%, commandWidthFor ~10%).
- The filter snapshot works: the recompute it avoids is 96× slower.
- Killed by the numbers: "more tasks slow rendering"; "truncateMatched is
  a hotspot" (321 ns-5 µs; the per-row `lipgloss.Width` passes are the
  cost); "render-twice is a general pattern" (only the modal path).
- Not measured: the end-to-end frame rate through a real TTY (the no-PTY
  rule); the Bubble Tea renderer-side diff. Ratios transfer; nanoseconds
  do not.

## Rejected (2026-11-04, with the load-bearing reason)

- Modal state as one enum — the two flags are exclusive by construction
  (tui.go:494-97); no leverage until a third modal exists.
- Picker-result value type — each field is read once in Run
  (tui.go:166-89).
- A TTY-edge module for Run's wiring (tui.go:117-135) — one caller,
  tmux-verified only: a hypothetical seam.
- Extracting the matcher/scorer (tui.go:865-1162) — already pure and
  directly tested; the friction is the snapshot straddle (FP5).
- Splitting discover.go into a registry package and an engine — it passes
  the deletion test as one module and exports three small functions.
- A shared ini package for `iniSections` (source_tox.go:251) — one
  consumer.
- Widening Files with Stat/Write — the miss-as-probe is the port's
  intended use.
- Splitting cmd/wtr/main.go per subcommand — a linear file with additive
  churn; nothing to hide.
- Caching discovery across runs — no per-keystroke rediscovery exists;
  1-2 discoveries per run.
- A full just grammar — the forgiving subset is the spec; the extra
  fidelity has no caller in the picker.
- Task.Source as a stable source ID — display text by design (task.go:13);
  the identity gap belongs to the warnings, not the tasks.

## The attack plan (2026-11-04)

The maintainer asked to attack all candidates in order, one spec per piece.
Drafted order: 1) warning identity; 2) FP7; 3) FP2 with the empty state;
4) FP6; 5) FP5; 6) FP3; 7) FP1 with Preview; 8) parser frame; 9) justfile
scanners and split; 10) discovery read/walk; 11) core test friction; 12) the
Files `~/` exception (may close as an accepted exception instead of code).

Cadence per piece: settle the open decisions (grilling), write the spec
(to-spec) and publish it with `ready-for-agent`, cut tickets when the piece
is large (to-tickets), implement (implement-spec; the worktree exists
before the implementer spawns; the brief inlines the implement discipline
because children cannot load skills), the implementer runs its own
code-review, then a draft PR stacked on the previous piece; merges stay
manual; after a merge the next piece rebases and retargets onto `main`.

Cross-piece rules: `task all` green before every commit; byte-identical
rendering wherever the frame changes, proved with the
[differential-testing](differential-testing.md) recipe; a CHANGELOG entry
for user-visible pieces (the FP7 preview fix, the warning-identity
persistence), `skip-changelog` otherwise; the small cleanups ride their
piece: the naming drift with the boxes work, the write-only `m.opts.Theme`
with FP1, the modal-open re-checks with FP6, the triple cache read with the
warning identity, the notice.go:15 drift with FP7.

Open decisions to settle at spec time: the dismissal-cache migration for
the warning identity (old text keys → new identity keys); whether the FP5
module ships the measured row cache with it or as a follow-up; the
differential scope of FP2 (which frames the byte-compare must cover).

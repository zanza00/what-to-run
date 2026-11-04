# Frame as data (piece 3)

The frame becomes data in the model — the styled rows, the placed boxes and
the list row count — and `paint` turns that structure into exactly today's
bytes at the View edge, so the placement and the tests read facts instead of
re-rendering the list or counting glyphs. Grilled with the maintainer on
2026-11-04; the synthesis lives in the local tracker
(`.scratch/fp2-frame-as-structure/spec.md`); this page is the durable
record. Candidate: "FP2 frame as structure" in
[arch-review-2026-11](arch-review-2026-11.md), piece 3 of the
[attack plan](attack-plan.md). Shipped on 2026-11-04 as PR #120 (merge
commit `32f6564`).

## What was broken (all at 2026-11-04, at `17bc5d0`)

- `render` joined everything into one string too early (tui.go:1258-1283):
  the title, the filter line, the list rows, the help line, the padding to
  `h` and the per-line truncation to `w` collapsed before any consumer could
  read a frame's facts back.
- The warning box placement could not count the list without drawing it:
  `warningModalY` re-rendered `m.listLines(w)` on every modal-open frame
  (tui.go:1337-1347), and its magic `3` duplicated the knowledge that
  `chromeRows = 4` held (tui.go:35). A modal-open frame cost 0.9-3.2 ms and
  1.10 MiB on the [arch-bench-2026-11](arch-bench-2026-11.md) suite, and
  the throwaway patch A had measured that counting rows instead removes
  10-27% of the time and 23-41% of the allocations.
- The empty screen was not a thing in the code: four sites held their own
  `len(m.tasks) == 0` branch (tui.go:508, 1356, 1410, 1684), so a new hint
  or key had to land in four places, and a miss degraded the empty screen
  silently.
- The tests recovered the frame's structure by counting newline and box
  glyphs (tui_test.go:495-635), because no structure existed to assert on.

## The decisions (2026-11-04)

- **The frame carries read fields only.** The `frame` struct holds the styled
  `rows` already truncated to `w` and padded to `h`, the placed `boxes`, and
  `listRows`, the count the boxes' placement needs. The spec's sketch also
  carried the frame's own geometry `w, h`; the implementation never read them
  back, and the final commit dropped them — the one recorded deviation from
  the spec's shape, caught by the piece's Standards review.
- **`buildFrame` is the only writer; `paint` sits at the View edge.**
  `buildFrame(h, w) frame` assembles the frame in exactly today's work and
  order; the string-returning `render` died. `paint` produces the bytes —
  `strings.Join` when no box is open, the compositor through `overlay` when
  boxes exist — and the View is its only caller, `--debug-area` included
  (tui.go, all at `32f6564`).
- **Patch A landed in the piece.** `warningModalY` stopped calling
  `listLines`: it is pure arithmetic over the frame facts (the height, the
  help row derived from `listRows`, the box height), with the same value at
  every input and the same branch table: centered above the help line when
  the list is tall enough, right below it for a couple of tasks, centered in
  the view on tiny terminals. Measured with benchstat against the
  `arch-bench-2026-11` baselines at `40c8a60`: modal-open frames
  −6.9…−22.5% time and −23…−41% allocations; modal-closed frames keep
  identical allocations, and their small time medians sit inside the measured
  same-code drift band.
- **`chromeTop = 3` joined `chromeRows = 4`** as a named constant, with a
  comment that ties them: the frame rows above the list, plus the help line
  below, are what `chromeRows` counts.
- **The empty screen is one fact.** `m.empty`, computed once in `newModel`
  (the discovery runs once, the filter changes visibility only, so
  `m.tasks` never changes after construction), is read by name by its four
  sites: `handleKey`, `filterLine`, `listLines`, `helpLine`. The
  `no task matches` row stays inside `listLines`.
- **The bytes are identical, proved twice.** The unit sweep of the
  [differential-testing](differential-testing.md) recipe — the closed
  variants at every h 1-40 × w 1-100, the box variants at the sparse edges,
  over 0/3/50 tasks, the default and the colorless theme, the cursor at the
  top, middle and bottom, the filter on and off — ran 165,060 comparisons
  with zero differences, raw bytes, never ANSI-stripped. The binary check
  proved the only two frames `--preview` can produce — the plain frame and
  the warning-modal frame — byte-identical, both binaries built and invoked
  under the same name, with a fresh `XDG_CACHE_HOME` per run.
- **The tests read the structure.** Three permanent tests ship with the
  piece: `TestWarningModalPlacement` pins the three placement branches,
  `TestBuildFrameFacts` pins the clamped window and the box placement, and
  `TestEmptyStateKeys` pins the empty-screen keys (enter and esc quit, every
  other key a no-op). The glyph-count tests stay as the byte-level pin.
  `GLOSSARY.md` gained the **Frame** term.
- **The two-axis review corrected five Standards findings in the final
  commit:** a stale neighbour comment still naming the "canvas", a glossary
  sentence copied into a code comment, a comment narrating a data flow the
  code did not implement, a four-idea sentence, and the write-only `w`/`h`
  fields above. The Spec axis reported no findings.

## What landed (2026-11-04, PR #120)

- `40c8a60` holds the frame as data and paints it at the View edge: the
  `frame` type, `buildFrame`, `paint`, the pure-arithmetic `warningModalY`,
  `chromeTop`, the single `m.empty` fact, and the three permanent tests.
  `task all` is green at that commit.
- `32f6564` merged it into `main`.

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

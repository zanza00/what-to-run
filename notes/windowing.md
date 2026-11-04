# Windowing (piece 4)

The three list windows — the task list, the theme picker, the warning list —
become one module: a `window` that holds the cursor and the offset, whose
operations settle every mutation and whose two reads own the defensive
clamps that two renders and three warning read paths used to repeat inline.
Grilled with the maintainer on 2026-11-04; the synthesis lives in the local
tracker (`.scratch/fp6-windowing/spec.md`); this page is the durable record.
Candidate: "FP6 windowing module" in
[arch-review-2026-11](arch-review-2026-11.md), piece 4 of the
[attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #122 (merge
commit `df6870b`, the single implementer commit `75e2a5b`).

## What was broken (all at 2026-11-04, at `32f6564`)

- The task list kept its cursor and its offset on the model and clamped and
  scrolled them in `keepCursorVisible` (tui.go:1190-1206), remembered at six
  call sites: the resize (:456), the cursor-row reply (:464), the reply
  timeout (:473), the ctrl+f flip (:584), the handleKey tail (:612) and the
  theme-saved relisting (:768). The cursor keys wrote the cursor raw
  (tui.go:551-566) and leaned on the tail as their only clamp.
- The theme picker repeated the same algorithm in `keepThemeVisible`
  (theme_modal.go:50-58) without the cursor clamp; the clamps lived at the
  six key sites of `handleThemeKey` (tui.go:664-675).
- The warning list had no window at all: `warningIdx` was clamped by hand at
  seven sites (tui.go:633, 636, 642, home pinning 0 at :639, the relisting
  at :764-766; warning_modal.go:118, 150, 172).
- No render trusted its own state, and no test read the algorithms: the task
  render re-clamped the offset (listLines, tui.go:1470) and the theme render
  re-clamped its offset (theme_modal.go:68) on every frame — the modal-open
  re-checks the attack plan rides with this piece — while zero direct tests
  existed for any of the three window algorithms.

## The decisions (2026-11-04 grilling; the review corrections of 2026-11-05 inline)

- **The window is a module.** `type window struct { cursor, offset int }`
  in `internal/tui/window.go`:16; it reads nothing — every method takes the
  list length and the height as its arguments, so the type is exhaustively
  testable without the model. The operations own every mutation: up, down,
  pageUp, pageDown, home, end (:22-58) each move the cursor and then settle
  (:69); top (:60) is the filter's reset.
- **Three instances, one type.** The task list at `listHeight()`
  (`listWin`), the theme picker at `themeModalVisibleRows()` (`themeWin`),
  and the warning list at height one (`warnWin`), where the algebra
  collapses onto the hand clamps: the offset always equals the clamped
  cursor, so the box' reads keep their defensive clamp through the module.
- **The six remembered call sites became named settles.** The geometry and
  relist events settle the task window at tui.go:463 (the resize), :471
  (the cursor-row reply), :480 (the reply timeout), :593 (the ctrl+f flip)
  and :793 (the theme-saved relisting, with the warning window settling at
  :792); the ctrl+t opening settles the theme window (:610). The handleKey
  tail and both `keepXxxVisible` functions died; `setFilter` keeps its top
  reset through `window.top()`.
- **The defensive clamps moved inside the module, once.** `visibleRange`
  (:89) owns the render clamp at tui.go:1474 (listLines) and
  theme_modal.go:55 (themeModalLines): a missed settle still renders a sane
  frame. `row` (:97) owns the clamped cursor read at warning_modal.go:118,
  156, 180. The review added `row`: the first cut read the cursor through
  the offset, which kept the clamp but conflated the two facts the glossary
  separates and dropped the negative and drift guard; the moved read keeps
  them apart.
- **The warning list entered as the height-one instance.** Its key
  operations settle through the module; the advance to the next warning
  that is not dismissed stays in `applyWarningDismissed` — the selection
  policy is not window arithmetic — writing its chosen index into the
  cursor and settling (warning_modal.go:163-164). The ratified deviation:
  `warningDismissedMsg` carries the window that held the warning instead of
  a bare index, so the clamp travels inside the module across the
  asynchronous save.
- **The sticky scroll is state.** The offset stays a field; a pure
  cursor-to-offset function was rejected at the grilling, because it would
  recentre the list under the cursor at every keystroke, which is a byte
  change.
- **One term.** The operation is named `settle` — the spec's `reprise`
  died in the review, because the comments already said "settles" and the
  house keeps one word for one concept. The height getter became
  `themeModalVisibleRows()`, because "window" now names the state pair.
  `GLOSSARY.md` gained the **Window** term.
- **The bytes stayed identical, proved at three levels.** The op
  equivalence sweep compared the old raw writes plus the old tail against
  the module operations on (cursor, offset): the task grid over lengths
  {0,1,2,3,5,50} × heights {-1,0,1,2,3,8} × cursors {-2…n+2} × offsets ran
  12,528 comparisons; the theme band 720; the warning band 56; the new
  `row` read against the old inline clamp 44, over the full grid with n = 0
  included. Zero differences everywhere; the out-of-band probes were
  counted, not failed — the old reads panic on their own unholdable
  states, so old and new differ there by construction. The frame sweep
  compared the pasted 32f6564 renderer against the new one, raw bytes: closed variants 56,000
  comparisons, box variants 5,814, and after the review corrections the box
  legs with drifted-cursor scenarios 3,978 — zero differences. The binary
  check proved the two `--preview` frames — plain and warning-modal —
  `cmp` identical, and the md5s did not move across the corrections.
- **Four permanent tests read the module directly** (window_test.go):
  `TestWindowOperations` (:30, 22 cases: the sticky scroll at the edges, the
  page steps, home and end at every boundary, the empty list at (0,0)),
  `TestWindowVisibleRange` (:100), `TestWindowRow` (:133) and
  `TestWindowNotifications` (:166, at the model seam). The glyph, empty and
  no-tasks tests stayed unmodified: they remain the byte-level pin.
- **The review verdicts.** The Spec axis: complete and faithful. The
  Standards axis: five findings, all fixed in the final commit — the
  settle rename, the getter rename, the `row` read, a dead trailing settle
  in `handleThemeKey` deleted, and the height-one fact stated once above
  its `1` literals.

## What landed (2026-11-05, PR #122)

- `75e2a5b` owns the three list windows in one module: window.go, the four
  permanent tests, the GLOSSARY term, and the migration of tui.go,
  theme_modal.go, warning_modal.go plus the four test files that read the
  old fields — 10 files, +499/-157. `task all` green at that commit; 822
  tests with `-race -shuffle=on`.
- `df6870b` merged it into `main`. The piece-3 bookkeeping had landed
  first, with PR #121 (merge commit `cc60157`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.

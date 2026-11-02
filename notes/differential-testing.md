# Differential testing for byte-identical refactors

The recipe for proving that a rendering refactor changes nothing. Used on
2026-11-02 to verify PR #84 (the box module) against `main`; the retro of
the box-implementer session (2026-11-02) promoted it here from prose so the
next byte-identical refactor (PR2) can cite it.

## The unit-level sweep

Before converting a renderer, capture its current behavior across the whole
input grid, then compare the converted one against it:

1. Write a scratch test file (`zz_*_test.go`, never committed) that inlines
   the OLD implementation, copied from `git show HEAD:<file>`.
2. Build both the old and the new output for the same inputs and compare
   with `==` on the rendered strings.
3. Sweep the grid: every height and width that can matter (for the box:
   h 1–32 × w 1–100), every data set, every theme, every cursor index. The
   2026-11-02 run was ~230k combinations, all zero-difference.
4. Delete the file before committing.

The grid matters because the guards and clamps live at the edges: h or w one
row or column around a threshold (chrome+1, minimum width) is where a
"defensive no-op" stops being one.

## The binary-level check

The unit sweep cannot see the program edge (invoked name, cache state,
discovery). Run the real binary:

1. `git worktree add /tmp/<repo>-main-wt main`, build both binaries.
2. Invoke both under the SAME filename: the picker title shows the invoked
   name (AGENTS.md rule), so different names differ on line 1 for a
   legitimate reason.
3. `--preview` renders a fixed 80×24 frame with no terminal, deterministic.
   Trigger the modal under test deterministically (for the warning box:
   `--theme nonexistent` produces an undismissed warning and opens the modal
   at start).
4. Point `XDG_CACHE_HOME` at a fresh empty dir per run, so the
   dismissed-warnings state is deterministic.
5. `cmp` the two frames: identical bytes or investigate.

## Cost

The 2026-11-02 sweep ran in seconds. The binary check needs two builds and a
worktree; clean both up after (`git worktree remove`, delete the binaries).

## Validation

**2026-11-02, PR2 sweep:** the grid caught what the whole test suite could
not — 62,524 combinations where the old `themeModalFooter` emitted a
zero-width escape pair (`dim.Render("")`) and the new box emitted bare `""`.
ANSI-stripped output was identical everywhere; only the raw byte
comparison found it. Compare raw bytes, never stripped output.

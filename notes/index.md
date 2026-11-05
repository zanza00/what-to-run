# Notes

Research notes and the agent wiki for `what-to-run`.

## Research notes

Long-form research, one file per topic:

- [npm-as-install-method-for-binaries](npm-as-install-method-for-binaries.md) — whether npm suits this CLI's install story.
- [npm-js-launcher-and-release-pipeline](npm-js-launcher-and-release-pipeline.md) — how the npm launcher and the release pipeline interact.
- [changelog-cut-with-protected-main](changelog-cut-with-protected-main.md) — cutting a changelog release while main is protected.
- [nixos-install-comparable-clis](nixos-install-comparable-clis.md) — how comparable CLIs install on NixOS.
- [python-task-sources](python-task-sources.md) — which files a Python project
  declares runnable tasks in, and which are static-parseable.
- [arch-bench-2026-11](arch-bench-2026-11.md) — the throwaway TUI benchmark
  suite of the 2026-11-04 review: source, baselines and profile highlights,
  so a future piece can rebuild the same measurements.

## Agent wiki

Tribal knowledge from the agent sessions: decisions, designs, recurring
problems. History, not a snapshot: every claim is dated and cites the commit
it was true at.

- [box-module](box-module.md) — the settled box-module design (FP4): PR1
  shipped, PR2 pending.
- [tui-deepening-roadmap](tui-deepening-roadmap.md) — the seven deepening
  candidates for `internal/tui`, in attack order.
- [agent-process](agent-process.md) — how agents work in this repo: the
  decisions and the skills they must load.
- [differential-testing](differential-testing.md) — the recipe for proving
  a rendering refactor byte-identical (unit sweep + binary check).
- [arch-install-channel](arch-install-channel.md) — the Arch install
  channels and the AUR removal decision (2026-11-03).
- [arch-review-2026-11](arch-review-2026-11.md) — the 2026-11-04 review:
  the re-validated roadmap, twelve candidates, the TUI measurements, the
  rejected list and the attack plan.
- [attack-plan](attack-plan.md) — the execution tracker of the twelve
  candidates plus the row-cache follow-up (row 13): a status table with
  dates, a progress log, and the resume pointer.
- [warning-identity](warning-identity.md) — the settled piece-1 design
  (2026-11-04): a warning carries a kind and a dismissal key instead of
  being its message text, and the dismissed-warnings file accepts both the
  key form and the legacy text form.
- [status-as-data](status-as-data.md) — the settled piece-2 design
  (2026-11-04): the model holds the status as a paint kind and a plain
  text, and the help line paints them at render time, so the ctrl+t live
  preview restyles the status line.
- [frame-as-data](frame-as-data.md) — the settled piece-3 design
  (2026-11-04): the frame is data — the styled rows, the placed boxes, the
  list row count — with `buildFrame` as its only writer and `paint` at the
  View edge.
- [windowing](windowing.md) — the settled piece-4 design (2026-11-05): one
  window module — the cursor, the offset, operations that settle, one render
  read and one cursor read — serves the task list, the theme picker and the
  warning list at height one.
- [list-module](list-module.md) — the settled piece-5 design (2026-11-05):
  the task list owns the tasks, the filter text and their derived facts in
  one module — built at construction, rebuilt at every mutation — so
  staleness is impossible and every read is a plain return.
- [row-painter](row-painter.md) — the settled piece-6 design (2026-11-05):
  one painter in `row.go` owns the row assembly of both the task and theme
  rows, so the per-segment, never-dim and pad laws have one home, and the
  SGR pin covers the selected theme row too.
- [display-mode](display-mode.md) — the settled piece-7 design (2026-11-05):
  one display module owns the mode facts, the geometry facts and the
  derivations — five operations settle, two named facts replace fifteen
  recombinations, Preview crosses the same door as the messages, and the
  themeModalHeight asymmetry is recorded as a candidate follow-up.
- [row-cache](row-cache.md) — the settled row-13 design (2026-11-05): one
  rowCache owns the pre-paint row derivations, keyed by the ranked
  position and the listing era, emptied wholesale at every era —
  steady frames stop re-deriving, the paint stays fresh, and the landed
  benchstat numbers prove the gain.
- [parser-frame](parser-frame.md) — the settled piece-8 design (2026-11-05):
  the discover seam owns the parser skeleton — the empty-file report,
  the sorted delivery, the help-else-body chain in one helper, the nil
  convention — with the npm, make and hatch exemptions ratified as
  facts of their sources.

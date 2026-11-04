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
  candidates: a status table with dates, a progress log, and the resume
  pointer.

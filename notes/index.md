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
- [invisible-runner-repo-map](invisible-runner-repo-map.md) — what the repo
  has, and lacks, for turning `wtr build` into the real command before the
  shell accepts it.
- [invisible-runner-shell-history](invisible-runner-shell-history.md) — the
  verified zsh and bash mechanics for rewriting the accepted line before it
  lands in history.

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

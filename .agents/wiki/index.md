# Wiki index

Working memory of the chief-of-staff sessions. Every claim in a page is dated and
cites its source: a commit SHA, a PR, or a report. Pages record history, so they
say when something was true, never that it is true now. Since 2026-11-01
evening this wiki lives on the `docs/nix-research-notes` branch, not as loose
files in a checkout. That branch merged into `main` through PR #82 on
2026-11-02 and no longer exists: commit `aaf35ab` added the wiki pages to
`main`, so since 2026-11-02 the wiki lives on `main` again.

## Packaging

- [Nix install verification](nix-install-verification.md) - PASS on 2026-11-01:
  the branch-ref install of the unmerged PR #61 works; how we test it.
- [Packaging channel PRs](packaging-channel-prs.md) - which PRs carry the nix
  and winget channel work, how PR #61 was aligned with main on 2026-11-01,
  which branches are stale, and how PR #62 was reviewed after its 2026-11-03
  rebase.

## CI

- [CI cost guardrails](ci-cost-guardrails.md) - 2026-11-02 audit: the CI costs
  $0.00 because the repo is public (billable 0 ms, 0 artifacts); what could
  charge the card, the one manual budgets check, and the runner-usage fixes:
  the trigger wastes in draft PR #100, and the darwin legs moved off the
  per-PR matrix — the nix install to flake paths and dispatch (PR #101), the
  Go test leg to the release cut (PR #103, 2026-11-02).

## Features

- [Invisible runner](invisible-runner.md) - the 2026-11-03 feasibility verdict
  for a typed `wtr build` becoming the real command at Enter: verified zsh and
  bash history-rewrite mechanics, the resolve gap in `cmd/wtr`, design
  direction A, the Enter-takeover risk and the open decisions.

## Research

- [NixOS install landscape](nixos-install-landscape.md) - how eleven
  comparable CLIs install on Nix/NixOS (2026-11-01), and what that says
  about PR #61's flake; cites `notes/nixos-install-comparable-clis.md`.

## Task sources

- [Python task sources](python-task-sources.md) - the scope and design
  decisions behind the poe, pdm, hatch, taskipy, tox and pipenv sources and
  their PR stack (2026-11-02); cites `notes/python-task-sources.md`.

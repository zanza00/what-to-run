# Wiki index

Working memory of the chief-of-staff sessions. Every claim in a page is dated and
cites its source: a commit SHA, a PR, or a report. Pages record history, so they
say when something was true, never that it is true now. Since 2026-11-01
evening this wiki lives on the `docs/nix-research-notes` branch, not as loose
files in a checkout. (Superseded on 2026-11-02: that branch no longer exists
locally or on origin; the wiki files are tracked on `main` again.)

## Packaging

- [Nix install verification](nix-install-verification.md) - PASS on 2026-11-01:
  the branch-ref install of the unmerged PR #61 works; how we test it.
- [Packaging channel PRs](packaging-channel-prs.md) - which PRs carry the nix
  and winget channel work, how PR #61 was aligned with main on 2026-11-01,
  and which branches are stale.

## CI

- [CI cost guardrails](ci-cost-guardrails.md) - 2026-11-02 audit: the CI costs
  $0.00 because the repo is public (billable 0 ms, 0 artifacts); what could
  charge the card, the one manual budgets check, and the two residual trigger
  wastes, fixed in draft PR #100 (2026-11-02).

## Research

- [NixOS install landscape](nixos-install-landscape.md) - how eleven
  comparable CLIs install on Nix/NixOS (2026-11-01), and what that says
  about PR #61's flake; cites `notes/nixos-install-comparable-clis.md`.

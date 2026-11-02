# CI cost guardrails

What-to-run's CI cannot charge zanza00's credit card while the repository is
public, and this page records why, what could break that, and the levers that
lower runner usage. Audit of 2026-11-02, everything read-only. Sources: the
subagent report in `.scratch/ci-usage-review/findings.md` and the billing
research in `.scratch/ci-usage-review/research/billing-rules.md` (scratch is
gitignored; the durable numbers are restated here), and the workflow files at
main@25501a1.

On 2026-11-02 the repository is public and its CI costs $0.00:

- 254 runs since 2026-10-18 is the full CI history: 17.4 runs/day, ~2,438
  runner-minutes per month (~40.6 h), 50.5% of runner time on macOS.
- The `/runs/{id}/timing` API reported `billable: 0 ms` on all 40 sampled
  runs: public repositories do not bill standard GitHub-hosted runners, and
  the docs (consulted 2026-11-02) confirm the included compute is not consumed
  by public-repository runs.
- Zero Actions artifacts: the release publishes GoReleaser assets, which are
  release assets and free without storage limits. The runner cache stays at
  the free default (10 GB/repo, 7-day eviction).

On 2026-11-02, the only paths that could charge the card:

1. A billing budget without "Stop usage when budget limit is reached": since
   2025 spending limits migrated to budgets, and a budget without that flag
   charges overage instead of blocking it. The migrated default is not
   documented explicitly, so the check at github.com/settings/billing →
   "Budgets and alerts" is the one manual action to take.
2. Artifact/Package storage beyond 500 MB (the docs do not state an exemption
   for public repositories, so the invariant to keep is: no `upload-artifact`).
3. Opting into the >10 GB per-repo cache ($0.07/GB-month; the docs are
   ambiguous for Free+card, so do not opt in).
4. Codespaces (never free, but separate from this repo's CI) and Git LFS
   beyond 10 GiB (unused).
5. Making the repository private: with the job mix at main@25501a1 that costs
   ~$84/month at 2026 rates (macOS $0.062/min ≈ 10.3× Linux), of which ~$72
   beyond the $12 included. Dropping the two macOS jobs from push/PR triggers
   brings it back inside the included compute.
6. Larger/GPU runners cannot bill: they are not available to personal accounts.

Trigger hygiene at main@25501a1 is already good (concurrency with
cancel-in-progress kills superseded runs at 0.0-0.1 min/job, per-job timeouts,
push only on main). The two residual wastes found on 2026-11-02: the
`edited`/`labeled`/`unlabeled` PR events re-run the whole 7-job matrix when
only the changelog job is affected (16 duplicate pairs in 14.6 days, ~6% of
runner minutes), and draft PRs run full CI. Both cost $0 today; the fix is a
separate changelog-only workflow plus a draft gate with `ready_for_review`.

Related: [Packaging channel PRs](packaging-channel-prs.md) covers the release
channels that the release workflow publishes.

On 2026-11-02 the two wastes got their fix in draft PR #100 (branch
`chore/ci-trigger-hygiene`, commits e448489, 112f97e, 7bfdac2): the changelog
rule moved to its own `changelog.yml` that owns the no-code events, and
`ci.yml` skips draft pull requests until `ready_for_review`. The draft PR
demonstrated both live on 2026-11-02: the ci run skipped `checks` and the
`CI required` aggregate, and the changelog run honored the skip-changelog
label (job skipped, `Changelog required` green). The review of the diff
surfaced that a superseded changelog run leaves a red aggregate row on
result `cancelled`, which is the same behavior the old ci workflow showed
on its cancelled runs (checked on runs 41427353121 and 41427354412); the
checks panel deduplicates per check name, so the newest run still carries
the verdict.

On 2026-11-02, marking PR #100 ready started the full matrix (the
ready_for_review trigger works). The two macOS legs queued 26 minutes
without a runner and the platform cancelled them, annotating "capacity
constraints, jobs targeting macOS arm64 runners may experience longer queue
times"; the GitHub status page reported no incident, and macos-14 retired
the same day. A rerun of the failed jobs acquired a runner for
`test (macos-latest)` in seconds (64 s, green); `nix (macos-latest)` kept
queueing. On top of the billing picture, macOS arm64 queue capacity is an
availability risk for the two darwin legs.

On 2026-11-02 the maintainer moved the darwin nix verification out of the
per-PR matrix: draft PR #101 (branch `chore/nix-install-at-release`, stacked
on PR #100) makes `checks.yml` run the nix job on Linux only, and adds
`nix-install.yml`, which runs `nix flake check`, `nix build`, then the user
install path no CI job ever covered (`nix profile install` from a GitHub ref
and the installed `wtr --version`), on flake-touching pull requests and on
dispatch. The release skill's step 4 now requires that workflow green on the
cut branch before the tag, through `scripts/verify-nix-install.sh`, which
reruns legs the pool cancelled before assigning a runner. On the PR's own head the paths trigger self-tested the workflow: the ubuntu
leg went green in 40 seconds, the darwin leg queued about nine minutes and
then went green too, and both printed `wtr 0.9.0` from the ref-installed
binary — the first CI proof of the user install path.

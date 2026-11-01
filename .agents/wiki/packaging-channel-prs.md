# Packaging channel PRs

State recorded on 2026-11-01.

- PR #58, "feat: install what-to-run with Nix", was closed **without merge** on
  2026-10-29 (`gh pr view` showed `mergedAt: null`). Its branch
  `feat/nix-channel` was deleted upstream.
- The local branch `feat/nix-channel` (head `0f00b0a` on 2026-11-01) was a
  stale copy of that old work: same commit titles as the live PR, different
  SHAs, and a tree that also mixes in winget scripts. The maintainer had it
  deleted on 2026-11-01 in the evening, after the wiki and the research note
  moved onto the `docs/nix-research-notes` branch.
- The live Nix work is PR #61, "feat: install what-to-run with Nix": draft, base
  `main`, branch `feat/channel-nix`, head `fa47b0e` on 2026-11-01. Relative to
  `main` it adds exactly three commits, only nix work: `flake.nix`,
  `flake.lock`, docs, README and workflow changes.
- PR #62, "feat: publish to WinGet from the release pipeline", is a separate
  draft on `feat/channel-winget`; it is not part of PR #61.
- House rules live in `AGENTS.md` (2026-11-01): PRs open as drafts, merges are
  manual, and agents never merge or push to `main`.

## Drift against main (2026-11-01, later in the day)

PR #61 at `fa47b0e` sat 36 commits behind `main` at `399a138`
(merge-base `5c6d1a8`). A trial merge with `git merge-tree` conflicted in five
files: `.github/workflows/checks.yml`, `.agents/skills/release/SKILL.md`,
`docs/install.md`, `docs/development.md` and `README.md`, because `main`
reworded the docs and the skill and added the changelog gate after the branch
diverged. The PR also carried no `CHANGELOG.md` entry, while `main` gates PRs
on one (`c791225`, `8eca80f`; retroactive changelog in `fee9b3d`). Whoever
rebases must reapply the Nix sections onto the new text and add a changelog
entry, then let the CI re-run, including the `nix` job.

## Alignment with main (2026-11-01, evening)

The drift above is history: on 2026-11-01 at 17:02 UTC `origin/main` was
merged into `feat/channel-nix` as merge commit `9c98c4b`, pushed to the PR
branch (fast-forward, no force). The five conflicts were resolved by keeping
`main`'s newer structure and reapplying the PR's Nix facts onto it:

- `checks.yml`: both new jobs kept, `nix` and `changelog`.
- `README.md`: the Nix install block plus `main`'s new pointer line.
- `docs/install.md`: the whole `## Nix` section plus `main`'s `## apt`
  heading; also added the missing Nix row to the channel table (the PR had
  the section but no row).
- `docs/development.md`: `main`'s version wholesale (its `## Release` section
  had moved into the release skill, which the PR had already patched), plus
  the `nix develop` sentence in Prerequisites.
- `.agents/skills/release/SKILL.md`: `main`'s reworded intro with "The
  version lives in the tag, and `flake.nix` repeats it for Nix builds", and
  the Nix row added to the Facts table.

A changelog entry ("Added Nix as an install channel...") was added under
`## [Unreleased]` / `### Added` because the PR touches
`.github/workflows/release.yml`, a watched path for the changelog gate. Local
gates passed (`task all`, `task check-md`, `scripts/check-changelog.sh
origin/main`), and the whole CI went green on `9c98c4b`: test on both
runners, cross, nix on both runners, the changelog job and `CI required`.
The PR stays a draft and the merge stays the maintainer's manual call.

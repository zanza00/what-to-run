---
name: release
description: Cut and verify a what-to-run release end to end, from picking the semver version to proving every package channel moved. Use it whenever the user wants a release out the door — also in Italian ("fare la release"), even without saying "release" — or asks why a release or a package-manager channel did not update.
---

# Release what-to-run

The version lives in the tag, and `flake.nix` repeats it for Nix builds.
Pushing `vX.Y.Z` on a commit in `main` starts `.github/workflows/release.yml`,
which turns the tag into a public release on every configured channel. The
work is to reach the tag push with evidence and then prove that every channel
moved; step 4 marks the point of no return.

## Facts the release depends on

| Thing | Value |
| --- | --- |
| Trigger | `vX.Y.Z` tag on a commit contained in `origin/main` |
| Branch protection | `main-guard` ruleset: `main` takes changes only through a merged pull request |
| Required secret | `HOMEBREW_TAP_TOKEN` (the workflow fails without it) |
| Optional secrets | `APT_GPG_PRIVATE_KEY`, `SCOOP_BUCKET_TOKEN`, `WINGET_TOKEN` |
| Release assets | `checksums.txt`, 4 tarballs, 2 Windows zips, 2 `.deb` files, 2 `.rpm` files, 2 `.pkg.tar.zst` files (13 total) |
| Version string | `wtr --version` prints `wtr X.Y.Z` in released binaries, `wtr vX.Y.Z-N-g…` from `task build` |
| Nix | the `version` attribute in `flake.nix`, bumped with the tag and checked by the validate job; the flake builds the default branch; the `nix install` workflow proves the user install path on Linux and Apple silicon, on the cut head before the tag (step 4) |
| Darwin test | the darwin leg of the `test` matrix runs only on the cut pull request and on a `ci` dispatch with `darwin=true`; the macOS arm64 pool can queue the leg for tens of minutes and cancel it before a runner takes it (step 4) |
| Homebrew | formula `Formula/what-to-run.rb` in `zanza00/homebrew-tap` |
| Scoop | `bucket/what-to-run.json` in `zanza00/scoop-bucket` |
| WinGet | `manifests/z/zanza00/WhatToRun/<version>` in `microsoft/winget-pkgs`, through a pull request from `zanza00/winget-pkgs` |
| apt | `dists/stable/main/binary-{amd64,arm64}/Packages` on `gh-pages` |
| RPM | `rpm/repodata/primary.xml.gz` and `repomd.xml.asc` on `gh-pages` |
| pacman | `arch/{x86_64,aarch64}/what-to-run.db` on `gh-pages`; the database lists the newest release only, while the directories keep every published package |
| Setup scripts | `setup.deb.sh`, `setup.rpm.sh` and `setup.arch.sh` at the root of `gh-pages` |
| npm | `what-to-run` plus the six platform packages `what-to-run-<os>-<arch>` |
| npm opt-in | the `NPM_PUBLISH` repository variable |

## 0. Read the state

```sh
git fetch origin --tags
git status --short --branch
git rev-list --left-right --count origin/main...HEAD
git tag --sort=-creatordate | head -5
gh auth status
gh api repos/zanza00/what-to-run/actions/secrets --jq '.secrets[].name'
gh api repos/zanza00/what-to-run/actions/variables \
  --jq '.variables[] | select(.name == "NPM_PUBLISH") | .value'
```

The secret list decides what a successful release means: a channel whose
optional secret is unset — and npm while `NPM_PUBLISH` is not `true` — is
skipped. Such a release is *expected*: complete for every channel it
promised, with nothing to repair.

`git rev-list --left-right --count origin/main...HEAD` prints
`<behind> <ahead>`. Stop and explain if the tree is dirty or HEAD is not on
`main`; if the previous release left you on its cut branch, switch back to
`main` first. Bring `main` up to `origin/main` before the cut:

```sh
git switch main
git pull --ff-only
```

If the pull refuses because the local branch diverged, the local commits
belong to this release: open a pull request for them, wait for the user to
merge it, then `git reset --hard origin/main`. Never tag from a side branch —
the workflow checks the tag commit is an ancestor of `origin/main`.

This step is done when the tree is clean, `main` sits at `origin/main`, and
you know which secrets exist and whether `NPM_PUBLISH` is `true`.

## 1. Choose the version

```sh
last=$(git describe --tags --abbrev=0)
git log --oneline "$last"..HEAD
```

If there are no commits, there is nothing to release — say so instead of
re-tagging the same tree.

Propose one version and explain it. The project is pre-1.0 and its history is
the useful rule: features and behaviour changes bump the minor (`0.5.0` →
`0.6.0`), fixes and docs-only releases bump the patch (`0.5.0` → `0.5.1`), and
a breaking change before 1.0 also bumps the minor but should be flagged as
breaking in the proposal. After 1.0, apply normal semver.

Ask the user to confirm or correct the version. Never tag a version they have
not seen.

## 2. Preflight

Confirm the `[Unreleased]` section has at least one entry:

```sh
scripts/check-changelog.sh --unreleased
```

If it exits non-zero, there is nothing to release: stop and ask the user
whether the release is needed.

Run the full gate from the repository root:

```sh
task all
git status --short
```

If `task all` reformats files, commit them on a separate branch and open a
pull request; the cut branch stays changelog-only. A format-only change is
not user-visible, so that pull request can carry the `skip-changelog` label.
The tag must contain what passed, and a dirty tree means the checks ran on
something else.

The Nix package reads its version from `flake.nix`, so set the `version`
attribute to the proposed version before the tag. Commit the bump with the
release changes. The validate job fails when the tag and the attribute differ.

When `goreleaser` is installed, validate the config and build every artifact
without publishing:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

The snapshot catches a broken `.goreleaser.yaml` or ldflags before the tag and
takes seconds; `dist/` is gitignored. If `goreleaser` is missing or resolves to
a shim without an installed version, either run it ad hoc
(`mise x goreleaser@latest -- goreleaser check`) or skip it and say so — CI's
`cross` job still runs `goreleaser check`.

Touch `scripts/publish-*.py`, the setup scripts or the release workflow in
this change? Run the publisher tests:

```sh
python3 -m unittest discover -s scripts -p '*_test.py'
```

Touch `npm/` or `scripts/publish-npm.py` in this change too? Run
`task smoke-npm` before the tag (it stages the packages into a local Verdaccio
registry and checks global `npm`, `pnpm --ignore-scripts` and
`npm --omit=optional` installs).

A release is the last cheap documentation checkpoint: if the change added a
task source, flag, theme or install method, confirm `AGENTS.md`, `README.md`
and `docs/` describe it now.

This step is done when `[Unreleased]` has at least one entry, `task all`
passes on a clean tree, and every gate this change triggers has run.

## 3. Cut the changelog on a pull request

The cut cannot reach `main` directly (see Facts), so open it as a pull request.
Cut it on a branch based on `origin/main`:

```sh
git fetch origin
git switch -c chore/cut-changelog-vX.Y.Z origin/main
```

Rename `## [Unreleased]` in `CHANGELOG.md` into
`## [X.Y.Z] - YYYY-MM-DD` with the chosen version and today's date, and put a
fresh empty `## [Unreleased]` section above it. Keep the entry wording
unchanged: the section is the release's curated record, and step 6 compares the
generated notes with it.

Commit the cut, push the branch, and open the pull request as a draft:

```sh
git add CHANGELOG.md
git commit -m "chore: cut the changelog for vX.Y.Z"
git push -u origin chore/cut-changelog-vX.Y.Z
gh pr create --draft --title "chore: cut the changelog for vX.Y.Z" \
  --body "Cut the changelog for vX.Y.Z."
```

The cut pull request touches only `CHANGELOG.md`, so the `changelog` job passes
without an entry. If the branch touches a watched path, stop: those commits
belong in a separate pull request.

This step is done when the pull request is open as a draft. The next step asks
the user to confirm the release and merge it.

## 4. Tag and push — confirm first

The tag push is the point of no return: once `publish` runs, consumers can see
the release. Prove the tree the tag will name installs through the Nix
channel: the cut pull request bumped `flake.nix`, so the `nix install`
workflow triggered on it, and this script watches the run and reruns the
failed jobs when the pool cancelled them before a darwin runner took them:

```sh
.agents/skills/release/scripts/verify-nix-install.sh chore/cut-changelog-vX.Y.Z
```

A red run from a real build failure stops the release: fix the flake on the
cut branch, push, and rerun the script. The script also accepts a tag, to
re-prove an already published tree.

The cut pull request also runs the darwin leg of the test matrix: the `ci`
workflow turns the leg on for the cut branch, so the `CI required` aggregate
waits for it. The macOS arm64 pool can queue the leg for tens of minutes and
cancel it before a runner takes it; when the aggregate reports the leg
cancelled, rerun the failed jobs of the cut pull request
(`gh run rerun --failed`) and wait for the green aggregate. Do not merge the
cut pull request before its checks are green.

Show the version, the release-notes outline, the cut pull request and the
commit at its head, then ask for confirmation. The user marks the cut pull
request ready and merges it; run these commands after the merge lands on
`origin/main`:

```sh
git fetch origin
cut_sha=$(gh pr list --head chore/cut-changelog-vX.Y.Z --state merged \
  --repo zanza00/what-to-run --limit 1 \
  --json mergeCommit --jq '.[0].mergeCommit.oid')
test -n "$cut_sha"
git merge-base --is-ancestor "$cut_sha" origin/main
git rev-parse -q --verify "refs/tags/vX.Y.Z" && echo "tag already exists"
git ls-remote --tags origin "refs/tags/vX.Y.Z"
git tag -a vX.Y.Z -m vX.Y.Z "$cut_sha"
git push origin vX.Y.Z
```

`cut_sha` is the commit the merged cut pull request put on `origin/main`, not
the tip of `origin/main`: another pull request may merge in between, and those
commits do not belong to this release. If `origin/main` has moved past
`cut_sha`, ask the user whether the later commits must ship now. If they must,
add their entries to the `[X.Y.Z]` section in another pull request, wait for
the merge, then tag the new commit; otherwise tag `cut_sha` and let the next
release carry them.

If the local or remote tag already exists, stop: a tag is never moved or
force-pushed. See `references/recovery.md` if it belongs to a failed or
published release.

## 5. Watch the workflow

The run appears a few seconds after the push:

```sh
gh run list --workflow=release.yml --limit 5
```

If the tag is not listed yet, wait a moment and retry rather than assuming the
trigger failed. Then watch the run that belongs to this tag:

```sh
tag=vX.Y.Z
run_id=$(gh run list --workflow=release.yml --limit 20 \
  --json databaseId,headBranch \
  --jq ".[] | select(.headBranch == \"$tag\") | .databaseId" | head -n1)
gh run watch "$run_id" --exit-status
```

This step is done when the run for the tag is green; `--exit-status` enforces
it. After `publish`, a `repo-arch` job publishes the pacman repository from an
Arch Linux container; it skips with a warning while `APT_GPG_PRIVATE_KEY` is
unset. While `NPM_PUBLISH` is `true`, the run also has a `verify-npm` job, which
installs the wrapper on Linux and Windows. On failure, read `gh run view
"$run_id" --log-failed` and follow `references/recovery.md`.

## 6. Verify every channel

The release notes are the commits between the previous tag and this one,
oldest first, with merge commits dropped; GoReleaser prints their subjects
only, so a `BREAKING CHANGE:` footer or a one-way removal stays invisible.
The generated notes include the cut commit (a `chore:` line); skip it. Compare
the notes with the cut `CHANGELOG.md` section: the file is the curated record,
and the notes are generated from commits. Every entry in the cut section must
appear in the notes; add the missing ones now — the tag stays put, only the
notes change:

```sh
gh release edit vX.Y.Z --notes-file notes.md
```

Run the verifier with the tag:

```sh
.agents/skills/release/scripts/verify-release.sh vX.Y.Z
```

The script prints one line per check, and its header lists what it checks.
Read each line by its state: `ok` is a check that passed; `skip` is a channel
the workflow skipped — expected; `FAIL` is a real problem; and `warn` means
the script could not decide, and its message says why — provide what it asks
for and rerun. WinGet has a review step, so a manifest still in an open pull
request shows as `pending`; a `FAIL` means there is neither a merged manifest
nor an open pull request.

Treat a `FAIL` as a real problem even when the workflow is green: the
verifier, not the workflow, is the source of truth for every channel. For a
failing channel, inspect the `publish` step of the run, or the `repo-arch`
job when the pacman repository is the failing channel, then follow section 5
of `references/recovery.md`. When the first WinGet manifest is merged, add
the WinGet section to `docs/install.md` and the one-liner to `README.md`;
until then, leave WinGet out of both. This step is done when the verifier
prints `N checks, 0 failed` and no `warn` is left.

## 7. Report

Close with a short, concrete summary:

- version and the user-visible changes that justify it;
- the release URL;
- channels updated, and channels skipped because their secret is unset or the
  npm opt-in is off;
- the verifier result (`N checks, 0 failed`).

## When something goes wrong

The full repair paths live in `references/recovery.md`. Triage:

- `validate` or `checks` failed → nothing is published; section 1.
- `publish` failed → whether the release exists decides the repair; sections 2
  and 3.
- `verify-npm` failed → everything else is published; repair npm as in
  section 3.
- published artifacts are wrong → section 4.

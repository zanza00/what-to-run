---
name: release
description: Cut and verify a what-to-run release end to end — pick the semver version from the commits since the last tag, run the preflight gates, push the annotated vX.Y.Z tag that starts the GitHub release workflow, watch the run, and verify every package channel (GitHub assets, Homebrew, Scoop, apt, RPM, AUR, npm). Use it whenever the user wants a release out the door — also in Italian ("fare la release"), even without saying "release" — or asks why a release or a package-manager channel did not update.
---

# Release what-to-run

The version lives only in the tag. Pushing `vX.Y.Z` on a commit in `main`
starts `.github/workflows/release.yml`, which validates the tag, reruns the CI
checks, builds the archives and Linux packages plus `checksums.txt`, publishes
the GitHub release, and updates Homebrew, Scoop, apt, RPM and, when
configured, the AUR and npm. The work is to reach the tag push with evidence
and then prove that every channel moved; step 3 marks the point of no
return.

## Facts the pipeline depends on

| Thing | Value |
| --- | --- |
| Trigger | `vX.Y.Z` tag on a commit contained in `origin/main` |
| Required secret | `HOMEBREW_TAP_TOKEN` (the workflow fails without it) |
| Optional secrets | `APT_GPG_PRIVATE_KEY`, `AUR_SSH_PRIVATE_KEY`, `SCOOP_BUCKET_TOKEN` (missing → channel skipped with a warning) |
| Release assets | `checksums.txt`, 4 tarballs, 2 Windows zips, 2 `.deb` files, 2 `.rpm` files (11 total) |
| Version string | `wtr --version` prints `wtr X.Y.Z` in released binaries, `wtr vX.Y.Z-N-g…` from `task build` |
| Homebrew | formula `Formula/what-to-run.rb` in `zanza00/homebrew-tap` |
| Scoop | `bucket/what-to-run.json` in `zanza00/scoop-bucket` |
| apt | `dists/stable/main/binary-{amd64,arm64}/Packages` on `gh-pages` |
| RPM | `rpm/repodata/primary.xml.gz` and `repomd.xml.asc` on `gh-pages` |
| Setup scripts | `setup.deb.sh` and `setup.rpm.sh` at the root of `gh-pages` |
| AUR | `what-to-run-bin` |
| npm | `what-to-run` plus the six platform packages `what-to-run-<os>-<arch>` |
| npm opt-in | the `NPM_PUBLISH` repository variable (not `true` → channel skipped with a warning) |

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

The secret list decides what a successful release means: Homebrew is
mandatory, the apt and RPM repositories, the AUR and the Scoop bucket may
legitimately be skipped. A missing optional secret means that channel-less
release is expected, not a failure. The `NPM_PUBLISH` variable does the same
for npm: while it is not `true`, a release without the npm packages is
expected, not a failure.

`git rev-list --left-right --count origin/main...HEAD` prints
`<behind> <ahead>`. Stop and explain if the tree is dirty or HEAD is not on
`main`. If HEAD is ahead, those commits belong to this release: review them,
then confirm and push them before tagging. Never tag from a side branch — the
workflow checks the tag commit is an ancestor of `origin/main`.

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

Cut the changelog version section first: rename `## [Unreleased]` in
`CHANGELOG.md` into `## [X.Y.Z] - YYYY-MM-DD` with the chosen version and
today's date, and put a fresh empty `## [Unreleased]` section above it. Keep
the entry wording unchanged, so the section doubles as the release summary.
Commit it now; step 3 pushes it to `main` before tagging, and the tag must
contain the renamed section.

Run the full gate from the repository root:

```sh
task all
git status --short
```

`task all` mirrors what CI runs locally: format, vet, race tests, build. CI
itself calls the Go commands directly in `.github/workflows/checks.yml`, plus a
gofmt gate. If it reformats files, commit that separately and push before
tagging — the tag must contain what passed, and a dirty tree means the checks
ran on something else.

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

## 3. Tag and push — confirm first

The tag push is the point of no return: once `publish` runs, consumers can see
the release. Show the version, the exact commit, and the release-notes outline,
then ask for confirmation before running:

```sh
git rev-parse -q --verify "refs/tags/vX.Y.Z" && echo "tag already exists"
git ls-remote --tags origin "refs/tags/vX.Y.Z"
git push origin main        # the tag must be an ancestor of origin/main
git tag -a vX.Y.Z -m vX.Y.Z
git push origin vX.Y.Z
```

Use an annotated tag, like the existing ones. If the local or remote tag
already exists, stop: a tag is never moved or force-pushed. See
`references/recovery.md` if it belongs to a failed or published release.

## 4. Watch the workflow

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
it. `validate` fails fast on a non-semver tag or a commit outside
`origin/main`; `checks` reuses `checks.yml`; only `publish` builds, uploads and
updates the package channels. While `NPM_PUBLISH` is `true`, the run also has
a `verify-npm` job, which installs the wrapper on Linux and Windows. On
failure, read `gh run view "$run_id" --log-failed` and follow
`references/recovery.md` — where it stopped decides what is safe.

## 5. Verify every channel

The release notes are the commits between the previous tag and this one,
oldest first, with merge commits dropped; GoReleaser prints their subjects
only, so a `BREAKING CHANGE:` footer or a one-way removal stays invisible.
When the range contains one, add that note to the published release now — the
tag stays put, only the notes change:

```sh
gh release edit vX.Y.Z --notes-file notes.md
```

Run the verifier with the tag:

```sh
.agents/skills/release/scripts/verify-release.sh vX.Y.Z
```

It checks the release is public and carries exactly the eleven assets, the
Homebrew formula has the version and download URLs, the Scoop manifest installs
the right zip on amd64 and arm64 with the hashes from `checksums.txt`, the apt
index and the RPM metadata on `gh-pages` have the version for both
architectures, the RPM metadata signature verifies with the repository key, the
setup scripts are published, the AUR package has it when that secret is
configured, the npm registry carries the wrapper and the six platform packages
at the tag version when `NPM_PUBLISH` is `true`, and `npx what-to-run@X.Y.Z`
reports `wtr X.Y.Z`, the downloaded archive matches `checksums.txt`, and a
binary for this platform prints `wtr X.Y.Z`. Missing repository/AUR/Scoop
secrets, and an npm opt-in that is not `true`, show as `skip`, matching the
workflow. Scoop has no review step — the manifest is written during the
release — so a stale or missing one is a `FAIL` while the secret list is
readable, and a `warn` when it is not.

Treat a `FAIL` as a real problem even when the workflow succeeded: the workflow
checks the release, the asset names, the Scoop bucket and the package
repositories, but the verifier is the source of truth for every channel. For a
failing channel, inspect the `publish` step of the run, then follow section 5
of `references/recovery.md`. If this is the first AUR publish, add an Arch
Linux section to `docs/install.md` that advertises `yay -S what-to-run-bin`.

## 6. Report

Close with a short, concrete summary:

- version and the user-visible changes that justify it;
- the release URL;
- channels updated, and channels skipped because their secret is unset or the
  npm opt-in is off;
- the verifier result (`N checks, 0 failed`);
- anything still manual, such as advertising the AUR package in the docs.

## When something goes wrong

The full repair paths live in `references/recovery.md`. Triage:

- `validate` or `checks` failed → nothing is published; section 1.
- `publish` failed → whether the release exists decides the repair; sections 2
  and 3. The Homebrew, AUR and npm publishers are idempotent.
- published artifacts are wrong → section 4.

# Release recovery

Where the release workflow stopped decides what is safe to do. Read it before
changing any tag:

```sh
run_id=…
gh run view "$run_id" --json jobs \
  --jq '.jobs[] | [.name, .conclusion] | @tsv'
gh run view "$run_id" --log-failed
```

The workflow jobs are `validate`, `checks` (the reusable `checks.yml`) and
`publish`, plus `repo-arch` after it, which publishes the pacman repository
and skips while `APT_GPG_PRIVATE_KEY` is unset, and `verify-npm` when
`NPM_PUBLISH` is `true`. Only `publish` creates the GitHub release and
updates the package channels; `repo-arch` and `verify-npm` only check or add
to what `publish` published, so a failure there changes nothing. Repair them
as in their sections below.

First question: does the release exist?

```sh
gh release view vX.Y.Z --json isDraft,isPrerelease,assets
```

## 1. `validate` or `checks` failed — nothing is published

The tag exists, but no release was created and no package channel moved. This
is the cheap case.

- If the failure is flaky infrastructure (network, runner image), rerun the
  failed jobs: `gh run rerun "$run_id" --failed`. The tag still points at the
  same commit, so the rerun is the whole repair: watch it to green.
- If the tag is malformed, or its commit is not in `origin/main`, or the code
  fails the checks, cancel the run if it is still queued (`gh run cancel
  "$run_id"`), then delete the tag, fix `main`, and tag again:

```sh
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z
# fix, task all, commit, push main, then redo the tag
```

Deleting a tag is only safe while `gh release view vX.Y.Z` says "not found".
A published release makes the tag consumer-visible; see section 4.

## 2. `publish` failed but the release does not exist

The build or upload failed before the release was created. Nothing is public
yet, so `gh run rerun "$run_id" --failed` is safe and rebuilds from the same
tag; watch the rerun to green. If the cause is the code or `.goreleaser.yaml`,
fix `main`, delete the tag, and tag the fix instead.

## 3. `publish` failed after the release exists — repair the channel

The GitHub release and its assets are up; one publisher failed. **Do not rerun
the `publish` job**: it starts from `goreleaser release --clean` again and
usually fails with "release already exists" before reaching the publisher.
Repair the channel directly. A repair is done when the verifier passes again:

```sh
.agents/skills/release/scripts/verify-release.sh vX.Y.Z
```

### apt, RPM and pacman repositories

The `republish packages` workflow is the recovery path, also used when
`gh-pages` rejected the push:

```sh
gh workflow run republish-packages.yml -f tag=vX.Y.Z
gh run list --workflow=republish-packages.yml --limit 1
gh run watch <run-id> --exit-status
```

It republishes the existing release to the apt, RPM and pacman repositories
without recreating it, and requires `APT_GPG_PRIVATE_KEY`. The pacman job runs
`repo-add` inside an Arch Linux container, like the release workflow does, so
you cannot repair the pacman repository from a workstation that lacks pacman.

### Homebrew

The publisher is idempotent: it regenerates the file from the released
`checksums.txt` and skips when the content already matches.

```sh
tmp=$(mktemp -d)
gh release download vX.Y.Z --pattern checksums.txt --dir "$tmp" --clobber
python3 scripts/publish-homebrew.py --tag vX.Y.Z --artifact "$tmp/checksums.txt"
```

The Homebrew script uses the authenticated `gh` CLI, so `gh auth status` must
have write access to `zanza00/homebrew-tap`.

### Scoop

The manifest is a file in `zanza00/scoop-bucket` and every release rewrites it,
so a failed push only delays the channel until the next one. To publish the
version immediately, clone the bucket, update `version`, the two URLs and the
two hashes in `bucket/what-to-run.json` from the released `checksums.txt`, and
push:

```sh
tmp=$(mktemp -d)
gh repo clone zanza00/scoop-bucket "$tmp/bucket"
gh release download vX.Y.Z --pattern checksums.txt --dir "$tmp" --clobber
"$EDITOR" "$tmp/bucket/bucket/what-to-run.json"
git -C "$tmp/bucket" commit -am "what-to-run: add vX.Y.Z"
git -C "$tmp/bucket" push
```

### WinGet

The manifest lives in `microsoft/winget-pkgs` and goes in through a pull
request from the `zanza00/winget-pkgs` fork, so a failed push leaves the
channel at the previous version until someone opens that pull request. When
the automation did not, `komac` opens it from the released assets:

```sh
GITHUB_TOKEN=… komac update zanza00.WhatToRun --version X.Y.Z --urls \
  https://github.com/zanza00/what-to-run/releases/download/vX.Y.Z/what-to-run_X.Y.Z_windows_amd64.zip \
  https://github.com/zanza00/what-to-run/releases/download/vX.Y.Z/what-to-run_X.Y.Z_windows_arm64.zip \
  --submit
```

`komac sync` first brings the fork up to date. The pull request still goes
through the usual review, and the verifier reports it as `pending` until the
manifest is merged. To fix a merged manifest, open a follow-up pull request
editing the three files under `manifests/z/zanza00/WhatToRun/<version>`;
never push to `microsoft/winget-pkgs` directly.

### npm

While the `NPM_PUBLISH` repository variable is not `true`, the workflow skips
the channel, and the release is *expected* without it. Turning the channel
on is a one-time setup, and only the maintainer can perform it: registering
trusted publishing for all seven packages on npmjs.com is a human step. The
npm section of `docs/development.md` owns the whole procedure. Never attempt
that registration.

When the variable is `true` and the publish step stopped partway, rerun the
publisher with the commands in the npm section of `docs/development.md`. It is
idempotent: every version already on the registry is skipped, so only the
missing packages publish.

## 4. The published release itself is wrong

If the binaries are broken, or an asset is missing and cannot be fixed by a
channel publisher, prefer a **new patch version**. Replacing a published
version changes hashes for anyone who already downloaded it, and the Homebrew
formula, apt and RPM indexes all key off that version.

If the release is still fresh and nothing has been fetched, the user may
decide to redo it. Confirm explicitly, then:

```sh
gh release delete vX.Y.Z --yes --cleanup-tag
# fix main, task all, commit, push main, tag the fix, push the tag
```

`--cleanup-tag` deletes the remote tag too. The next tag push reruns validate,
checks and publish from scratch. Homebrew regenerates its file from the new
checksums and accepts the new hashes, but **apt does not**: reprepro refuses a
package version already present in the index, so a rebuilt release of the same
version will fail the apt publisher. Anyone who fetched the old artifacts also
holds checksums that no longer match. That is why a new patch version is the
safe path.

Never force-push or move `main`, and never move a tag while its release is
still published.

## 5. The workflow succeeded but verification finds a stale channel

For a channel that is on:

- apt, RPM or pacman stale → run the `republish packages` workflow above;
- Homebrew stale → run `publish-homebrew.py` locally;
- Scoop stale → update `bucket/what-to-run.json` in the bucket as above;
- npm stale → rerun the publisher as above;
- WinGet pending → open the update pull request with `komac` as above.

After any repair, run the verifier again:

```sh
.agents/skills/release/scripts/verify-release.sh vX.Y.Z
```

If a secret was rotated or removed, this shows what the workflow will actually
do next time:

```sh
gh api repos/zanza00/what-to-run/actions/secrets --jq '.secrets[].name'
```

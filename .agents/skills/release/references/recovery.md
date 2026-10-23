# Release recovery

Where the release workflow stopped decides what is safe to do. Read it before
changing any tag:

```sh
run_id=…
gh run view "$run_id" --json jobs \
  --jq '.jobs[] | [.name, .conclusion] | @tsv'
gh run view "$run_id" --log-failed
```

The workflow has three jobs: `validate`, `checks` (the reusable `checks.yml`)
and `publish`. Only `publish` creates the GitHub release and updates the
package channels.

First question: does the release exist?

```sh
gh release view vX.Y.Z --json isDraft,isPrerelease,assets
```

## 1. `validate` or `checks` failed — nothing is published

The tag exists, but no release was created and no package channel moved. This
is the cheap case.

- If the failure is flaky infrastructure (network, runner image), rerun the
  failed jobs: `gh run rerun "$run_id" --failed`. The tag still points at the
  same commit, so nothing else is needed.
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
tag. If the cause is the code or `.goreleaser.yaml`, fix `main`, delete the
tag, and tag the fix instead.

## 3. `publish` failed after the release exists — repair the channel

The GitHub release and its assets are up; one publisher failed. **Do not rerun
the `publish` job**: it starts from `goreleaser release --clean` again and
usually fails with "release already exists" before reaching the publisher.
Repair the channel directly.

### apt

Documented recovery path, also used when `gh-pages` rejected the push:

```sh
gh workflow run republish-apt.yml -f tag=vX.Y.Z
gh run list --workflow=republish-apt.yml --limit 1
gh run watch <run-id> --exit-status
```

It downloads the released `.deb` files and `checksums.txt` and publishes them
to `gh-pages` without recreating the release. It requires
`APT_GPG_PRIVATE_KEY`.

### Homebrew and AUR

Both publishers are idempotent: they regenerate the file from the released
`checksums.txt` and skip when the content already matches.

```sh
tmp=$(mktemp -d)
gh release download vX.Y.Z --pattern checksums.txt --dir "$tmp" --clobber
python3 scripts/publish-homebrew.py --tag vX.Y.Z --artifact "$tmp/checksums.txt"
python3 scripts/publish-aur.py --tag vX.Y.Z --artifact "$tmp/checksums.txt" \
  --ssh-key-file ~/.ssh/aur
```

The Homebrew script uses the authenticated `gh` CLI, so `gh auth status` must
have write access to `zanza00/homebrew-tap`. The AUR script needs the private
key whose public half is on the AUR account. Without the local key, the AUR
path cannot be repaired from a workstation.

After any repair, run the verifier again:

```sh
.agents/skills/release/scripts/verify-release.sh vX.Y.Z
```

## 4. The published release itself is wrong

If the binaries are broken, or an asset is missing and cannot be fixed by a
channel publisher, prefer a **new patch version**. Replacing a published
version changes hashes for anyone who already downloaded it, and the Homebrew
formula, apt index and AUR package all key off that version.

If the release is still fresh and nothing has been fetched, the user may
decide to redo it. Confirm explicitly, then:

```sh
gh release delete vX.Y.Z --yes --cleanup-tag
# fix main, task all, commit, push main, tag the fix, push the tag
```

`--cleanup-tag` deletes the remote tag too. The next tag push reruns validate,
checks and publish from scratch. Homebrew and the AUR regenerate their files
from the new checksums and accept the new hashes, but **apt does not**: reprepro
refuses a package version already present in the index, so a rebuilt release of
the same version will fail the apt publisher. Anyone who fetched the old
artifacts also holds checksums that no longer match. That is why a new patch
version is the safe path.

Never force-push or move `main`, and never move a tag while its release is
still published.

## 5. The workflow succeeded but verification finds a stale channel

The CI "Verify published release" step checks only the asset names, so a
successful run is not proof that Homebrew, apt or the AUR moved. The bundled
verifier is the source of truth. For a channel whose secret is set:

- apt stale → run the `republish apt` workflow above;
- Homebrew stale → run `publish-homebrew.py` locally;
- AUR stale → run `publish-aur.py` locally with the key.

If a secret was rotated or removed, this shows what the workflow will actually
do next time:

```sh
gh api repos/zanza00/what-to-run/actions/secrets --jq '.secrets[].name'
```

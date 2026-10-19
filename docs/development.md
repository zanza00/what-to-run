# Development

Run these commands from the repository root:

```sh
make build # build bin/wtr and bin/what-to-run
make test  # run tests with the race detector and coverage
make all   # format, vet, test, and build
```

To try the widget from a checkout in the current shell:

```sh
eval "$(make bind)"                  # zsh, Ctrl-S
eval "$(make bind-bash)"             # bash, Ctrl-S
eval "$(make bind KEY=ctrl-g)"       # zsh on another key
eval "$(make bind-bash KEY=ctrl-g)"  # bash on another key
```

## Release

Releases are automated. Push a `vX.Y.Z` tag on a commit contained in `main`:
`.github/workflows/release.yml` builds the archives and the `.deb` packages,
creates the GitHub release, updates `Formula/what-to-run.rb` in the
[zanza00/homebrew-tap](https://github.com/zanza00/homebrew-tap) repository,
and, once the AUR secret below is set, pushes to the
[what-to-run-bin](https://aur.archlinux.org/packages/what-to-run-bin) AUR
package.

The Homebrew step needs a `HOMEBREW_TAP_TOKEN` repository secret (a token
with Contents read/write on the tap), the same secret the source-fetch release
uses. It publishes through `scripts/publish-homebrew.py`.

The AUR step needs an `AUR_SSH_PRIVATE_KEY` repository secret holding an
unencrypted ed25519 private key whose public half is registered on your
[AUR account](https://aur.archlinux.org/account). Generate a dedicated key
with `ssh-keygen -t ed25519 -f aur -C what-to-run-release`, add `aur.pub` on
the account page, and store the private `aur` file as the secret. It publishes
through `scripts/publish-aur.py`; the first release creates the AUR package.
While the secret is unset (AUR registration is currently closed), releases skip
AUR publishing with a warning instead of failing. When the package exists, the
Arch Linux section in `docs/install.md` can advertise `yay -S what-to-run-bin`.

The apt step needs an `APT_GPG_PRIVATE_KEY` repository secret holding the
armored private key used to sign the repository (generated with
`gpg --batch --gen-key`). It publishes through `scripts/publish-apt.py`, which
updates the `gh-pages` branch served via GitHub Pages; the first release
populates the repository. While the secret is unset, releases skip apt
publishing with a warning instead of failing. The Debian/Ubuntu section in
`docs/install.md` advertises the apt repository. If the apt step fails after the
GitHub release was published, run the `republish apt` workflow with the tag as
input: it updates `gh-pages` from the released assets without recreating the
release.

To validate the release setup without publishing:

```sh
goreleaser check                                   # validate .goreleaser.yaml
goreleaser release --snapshot --clean              # build dist/ locally
python3 -m unittest discover -s scripts -p '*_test.py'
```

`wtr --version` reports the version injected at build time: the release tag in
released archives, a `git describe` string in local `make build` binaries.

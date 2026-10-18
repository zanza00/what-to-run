# Development

Run these commands from the repository root:

```sh
make build # build bin/wtr and bin/what-to-run
make test  # run tests with the race detector and coverage
make all   # format, vet, test, and build
```

To try the widget from a checkout in the current shell:

```sh
eval "$(make bind)"             # Ctrl-S
eval "$(make bind KEY=ctrl-g)"  # choose another key
```

## Release

Releases are automated. Push a `vX.Y.Z` tag on a commit contained in `main`:
`.github/workflows/release.yml` builds the archives, creates the GitHub
release, and updates `Formula/what-to-run.rb` in the
[zanza00/homebrew-tap](https://github.com/zanza00/homebrew-tap) repository.

The Homebrew step needs a `HOMEBREW_TAP_TOKEN` repository secret (a token
with Contents read/write on the tap), the same secret the source-fetch release
uses. It publishes through `scripts/publish-homebrew.py`.

To validate the release setup without publishing:

```sh
goreleaser check                                   # validate .goreleaser.yaml
goreleaser release --snapshot --clean              # build dist/ locally
python3 -m unittest discover -s scripts -p '*_test.py'
```

`wtr --version` reports the version injected at build time: the release tag in
released archives, a `git describe` string in local `make build` binaries.

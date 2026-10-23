# Development

## Prerequisites

Go and [`task`](https://taskfile.dev) (go-task 3.x):

```sh
go install github.com/go-task/task/v3/cmd/task@v3.54.0
```

`go install` writes the `task` binary to `$(go env GOPATH)/bin`, or to `$GOBIN`
when that is set. That directory must be on `PATH`: if `task` is not found
after installing, add it to the `PATH` in your shell rc file.

`task` is a contributor prerequisite, not a runtime dependency of `wtr`. When
it is not installed, run the Go commands from `.github/workflows/checks.yml`
directly.

## Commands

Run these commands from the repository root:

```sh
task build # build bin/wtr and bin/what-to-run
task test  # run tests with the race detector and coverage
task all   # format, vet, test, and build
```

## Use the local build instead of a release

`wtr` and `what-to-run` are found through `PATH`, and the first match wins. If
you installed a release with Homebrew, apt, mise or any of the official install methods,
a bare `wtr` runs that copy. To make the checkout win, put its `bin/` directory
first in `PATH`, after the rc file lines that prepend an installer's directory
(`brew shellenv`, `mise activate`, and similar add to the front themselves):

```sh
# ~/.zshrc or ~/.bashrc; adjust the path to your checkout
# substitute with where you checkout the folder
if [ -x "$HOME/path/to/checkout/what-to-run/bin/wtr" ]; then
  PATH="$HOME/path/to/checkout/what-to-run/bin:$PATH"
fi
```

The `-x` guard keeps the rc file quiet when the checkout is not there. In a new
shell, check what is resolved:

```sh
command -v wtr
command -v what-to-run
wtr --version # release: 0.4.0, local build: v0.4.0-3-gbe08e69
```

A release prints the bare tag, while a local `task build` appends the commit
distance from `git describe`, so `--version` tells the two apart. From then on
`task build` replaces the binaries the shell runs, and `task install` is not
needed for local development.

Start the widget with the bare command name, not with `bin/wtr`:

```sh
eval "$(wtr init zsh)"  # or: eval "$(what-to-run init bash)"
```

The generated script keeps `WTR_BIN` as that bare name and resolves it through
`PATH` at every keypress, so a later `task build` is picked up immediately,
without evaluating `init` again. Shells opened before the `PATH` change need a
restart (or the snippet above run again) first; if the old command is still
cached, run `hash -r` in bash or `rehash` in zsh. For a one-off,
`WTR_BIN=/path/to/wtr` pins a specific binary for the session.

`task bind` and `task bind-bash` remain available for trying the widget from a
checkout:

```sh
eval "$(task bind)"                  # zsh, Ctrl-S
eval "$(task bind-bash)"             # bash, Ctrl-S
eval "$(task bind KEY=ctrl-g)"       # zsh on another key
eval "$(task bind-bash KEY=ctrl-g)"  # bash on another key
```

They build and then invoke `bin/wtr init ...`, so the generated `WTR_BIN` is
the absolute path of `bin/wtr`, fixed when you evaluate it: convenient for a
one-off test, but later `task build` runs are not picked up until you evaluate
the output again. Use the bare-name `init` above when the shell should always
follow the checkout.

## Windows

Cross-build the Windows binary from any platform:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/wtr.exe ./cmd/wtr
```

To try the PowerShell widget from the checkout, run this in an interactive
PowerShell from the repository root:

```powershell
$env:Path = "$PWD\bin;$env:Path"
wtr.exe init powershell | Out-String | Invoke-Expression
```

`Out-String` keeps the script's newlines and `#` comments, which
`Invoke-Expression` needs. There are no PTY tests for Windows: try the widget by
hand in Windows Terminal and in conhost. Both answer the cursor position query
on Windows 11, so the inline frame works there; consoles that do not answer
fall back to the full-screen picker after 250 ms.

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
released archives, a `git describe` string in local `task build` binaries.

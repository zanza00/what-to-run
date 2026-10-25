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
task build    # build bin/wtr
task test     # run tests with the race detector and coverage
task fmt-md   # format the Markdown files with dprint
task check-md # check the Markdown formatting
task all      # format (Go and Markdown), vet, test, and build
```

`task fmt-md` and `task check-md` download the pinned dprint release into
`bin/tools` on first use, so there is no global dprint install and every
machine uses the same version. The first run needs network access to download
the binary and the Markdown plugin, which dprint caches in its user cache;
later runs are offline. `task all` runs `task fmt-md` as part of its format
step, and `task clean` deletes `bin/tools`, so the next run downloads the
binary again.
The Markdown under `.agents/skills/` is excluded in `dprint.json`: it is
vendored and keeps the upstream formatting. Tables are kept as written too,
because aligning their cells would turn the longer rows into very wide lines.

## README demo

The README demo is generated from `docs/demo/demo.tape`:

```sh
task demo
```

The tape drives `docs/demo/fixture` through the zsh widget: Ctrl-O opens the
inline picker, the `buld` filter narrows the list to `make build` (a
subsequence match, so the GIF shows the fuzzy finder at work), and Enter runs
it.
VHS draws the window chrome (`Set WindowBar Colorful`, so the frame has its
close/minimize/maximize dots) over the wallpaper in `docs/demo/background.webp`,
a crop of [andreas kretschmer's photo on
Unsplash](https://unsplash.com/photos/brown-mountain-during-daytime-4VXCQjiyVyc)
cropped to the recording size: VHS stretches `MarginFill` to the canvas, so the
image has to match its aspect ratio. It is WebP rather than JPEG because ffmpeg
decodes JPEG as full-range while VHS fills `Padding` with a limited-range
color, which shows as a lighter band around the terminal. The keycaps drawn by
`docs/demo/make-keys.sh` are overlaid inside the window near its bottom edge,
and `docs/demo/make-demo.sh` composites them and encodes the GIF with gifski.

`task demo` needs `vhs` (with its `ttyd` and ffmpeg dependencies), ImageMagick 7
(`magick`), gifski and the JetBrainsMono Nerd Font (`DEMO_FONT` overrides the
font file). VHS downloads a headless Chromium on first use. The GIF is committed
and the target is not part of `task all` or CI, so these tools are only needed
when changing the demo.

## Use the local build instead of a release

`wtr` is found through `PATH`, and the first match wins. If you installed a
release with Homebrew, apt, mise or any of the official install methods, a bare
`wtr` runs that copy. To make the checkout win, put its `bin/` directory first
in `PATH`, after the rc file lines that prepend an installer's directory
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
wtr --version # release: 0.4.0, local build: v0.4.0-3-gbe08e69
```

A release prints the bare tag, while a local `task build` appends the commit
distance from `git describe`, so `--version` tells the two apart. From then on
`task build` replaces the binary the shell runs, and `task install` is not
needed for local development.

Start the widget with the bare command name, not with `bin/wtr`:

```sh
eval "$(wtr init zsh)"
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
eval "$(task bind)"                  # zsh, Ctrl-O
eval "$(task bind-bash)"             # bash, Ctrl-O
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
`.github/workflows/release.yml` builds the archives and the `.deb` and `.rpm`
packages, creates the GitHub release, updates `Formula/what-to-run.rb` in the
[zanza00/homebrew-tap](https://github.com/zanza00/homebrew-tap) repository,
writes the Scoop manifest into
[zanza00/scoop-bucket](https://github.com/zanza00/scoop-bucket), and, once the
AUR secret below is set, pushes to the
[what-to-run-bin](https://aur.archlinux.org/packages/what-to-run-bin) AUR
package.

The Homebrew step needs a `HOMEBREW_TAP_TOKEN` repository secret (a token
with Contents read/write on the tap), the same secret the source-fetch release
uses. It publishes through `scripts/publish-homebrew.py`.

The Scoop step needs a `SCOOP_BUCKET_TOKEN` repository secret (a token with
Contents read/write on the bucket). GoReleaser writes
`bucket/what-to-run.json` there, and the bucket must already exist with a
commit on `main`, because the publisher creates the file on that branch.
GoReleaser's scoop pipe has no "do nothing without a token" mode: while the
secret is unset the workflow passes `--skip=scoop` and warns instead of
failing.

The AUR step needs an `AUR_SSH_PRIVATE_KEY` repository secret holding an
unencrypted ed25519 private key whose public half is registered on your
[AUR account](https://aur.archlinux.org/account). Generate a dedicated key
with `ssh-keygen -t ed25519 -f aur -C what-to-run-release`, add `aur.pub` on
the account page, and store the private `aur` file as the secret. It publishes
through `scripts/publish-aur.py`; the first release creates the AUR package.
While the secret is unset (AUR registration is currently closed), releases skip
AUR publishing with a warning instead of failing. When the package exists, the
Arch Linux section in `docs/install.md` can advertise `yay -S what-to-run-bin`.

The package repository step needs an `APT_GPG_PRIVATE_KEY` repository secret
holding the armored private key used to sign the apt repository, the RPM
repository metadata and the RPM packages (generated with
`gpg --batch --gen-key`). It publishes through `scripts/publish-repo.py`,
which updates the apt and RPM repositories on the `gh-pages` branch served via
GitHub Pages; the first release populates them. While the secret is unset,
releases skip publishing with a warning instead of failing. The Debian/Ubuntu
section in `docs/install.md` advertises the apt repository, and the Fedora/RHEL
section documents the `.rpm` package. If the step fails after the GitHub
release was published, run the `republish packages` workflow with the tag as
input: it updates `gh-pages` from the released assets without recreating the
release.

The same publisher copies `scripts/setup.deb.sh` and `scripts/setup.rpm.sh`
to the root of `gh-pages`, where the one-liners in `docs/install.md` fetch
them from. The scripts are not tied to a version, so an edit reaches the site
with the next release (or a `republish packages` run).

To validate the release setup without publishing:

```sh
goreleaser check                                   # validate .goreleaser.yaml
goreleaser release --snapshot --clean              # build dist/ locally
python3 -m unittest discover -s scripts -p '*_test.py'
```

A snapshot writes `dist/scoop/bucket/what-to-run.json` with a `SNAPSHOT`
version: the Scoop pipe runs in the build phase, so the manifest is generated
without being published. The Homebrew formula comes from a custom publisher, so
only a real release writes it. `goreleaser check` is what catches a broken
`scoops` block before the tag.

`wtr --version` reports the version injected at build time: the release tag in
released archives, a `git describe` string in local `task build` binaries.

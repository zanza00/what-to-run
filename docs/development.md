# Development

This page covers the tools you need to work on `wtr`, the development tasks and
the release pipeline.

## Prerequisites

Install Go 1.27.2 or later and [`task`](https://taskfile.dev) (go-task 3.x):

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

`task` with no arguments lists every task.

`task fmt-md` and `task check-md` download the pinned dprint release into
`bin/tools` on first use. There is no global dprint install, and every machine
uses the same version. The first run needs network access for the binary and
the Markdown plugin, which dprint stores in its user cache. Later runs work
offline. `task all` runs `task fmt-md` in its format step, and `task clean`
deletes `bin/tools`, so the next run downloads the binary again.

dprint skips the Markdown under `.agents/skills/`. Those files are vendored and
keep the upstream formatting. Tables stay as written too: aligning the cells
would turn the longer rows into very wide lines.

## README demo

`docs/demo/demo.tape` records the demo in the README. Regenerate the GIF with:

```sh
task demo
```

The tape drives `docs/demo/fixture` through the zsh widget. Ctrl-O opens the
inline picker, and typing `buld` narrows the list to `make build`, because the
filter matches subsequences. Enter runs the command.

VHS draws the window chrome (`Set WindowBar Colorful` adds the
close/minimize/maximize dots) over the wallpaper in
`docs/demo/background.webp`. The wallpaper is a crop of
[andreas kretschmer's photo on
Unsplash](https://unsplash.com/photos/brown-mountain-during-daytime-4VXCQjiyVyc),
cut to the recording size. VHS stretches `MarginFill` to the canvas, so the
image must have the same aspect ratio. It is WebP and not JPEG because ffmpeg
decodes JPEG as full-range, while VHS fills `Padding` with a limited-range
color. A JPEG would show a lighter band around the terminal.

`docs/demo/make-keys.sh` draws the keycaps. `docs/demo/make-demo.sh` overlays
them inside the window, near its bottom edge, and encodes the GIF with gifski.

`task demo` needs `vhs` (with its `ttyd` and ffmpeg dependencies), ImageMagick
7 (`magick`), gifski and the JetBrainsMono Nerd Font (`DEMO_FONT` overrides the
font file). VHS downloads a headless Chromium on first use. The GIF is
committed, and the target is not part of `task all` or CI. You need these tools
only when you change the demo.

## Use the local build instead of a release

`wtr` is found through `PATH`, and the first match wins. If you installed a
release with Homebrew, apt, mise or any of the official install methods, a bare
`wtr` runs that copy. To make the checkout win, put its `bin/` directory first
in `PATH`. Put the line after the rc file lines that prepend an installer's
directory (`brew shellenv`, `mise activate` and similar add to the front
themselves):

```sh
# ~/.zshrc or ~/.bashrc, with the path of your checkout
if [ -x "$HOME/path/to/checkout/what-to-run/bin/wtr" ]; then
  PATH="$HOME/path/to/checkout/what-to-run/bin:$PATH"
fi
```

The `-x` guard keeps the rc file quiet when the checkout is not there. In a new
shell, check which binary is resolved:

```sh
command -v wtr
wtr --version # wtr 0.9.0 (release) or wtr v0.9.0-18-g1ff1695 (local build)
```

A release prints the version after the command name. A local `task build`
appends the commit distance from `git describe`, so `--version` tells the two
apart. From then on `task build` replaces the binary the shell runs, and
`task install` is not needed for local development.

Start the widget with the bare command name, not with `bin/wtr`:

```sh
eval "$(wtr init zsh)"
```

The generated script keeps `WTR_BIN` as that bare name and resolves it through
`PATH` at every keypress. A later `task build` is picked up immediately, without
evaluating `init` again. Shells opened before the `PATH` change need a restart
(or the snippet above run again) first. If the old command is still cached, run
`hash -r` in bash or `rehash` in zsh. For a one-off, `WTR_BIN=/path/to/wtr`
pins a specific binary for the session.

`task bind` and `task bind-bash` remain available for trying the widget from a
checkout:

```sh
eval "$(task bind)"                  # zsh, Ctrl-O
eval "$(task bind-bash)"             # bash, Ctrl-O
eval "$(task bind KEY=ctrl-g)"       # zsh on another key
eval "$(task bind-bash KEY=ctrl-g)"  # bash on another key
```

They build and then invoke `bin/wtr init ...`, so the generated `WTR_BIN` is
the absolute path of `bin/wtr`, fixed when you evaluate it. That is convenient
for a one-off test, but later `task build` runs are not picked up until you
evaluate the output again. Use the bare-name `init` above when the shell should
always follow the checkout.

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
on Windows 11, so the inline frame works there. Consoles that do not answer
fall back to the fullscreen picker after 250 ms.

## Release

Releases are automated. Push a `vX.Y.Z` tag on a commit in `main`.
`.github/workflows/release.yml` runs the checks and builds the archives and the
`.deb` and `.rpm` packages. It creates the GitHub release and updates the
Homebrew, Scoop, apt, RPM and AUR channels. The workflow is the source of truth
for the steps, and its publishers are the scripts in `scripts/`.

### Secrets

Four repository secrets configure the channels. `HOMEBREW_TAP_TOKEN` is the
only required one: the release fails without it.

- `HOMEBREW_TAP_TOKEN` is a token with Contents read/write on
  [zanza00/homebrew-tap](https://github.com/zanza00/homebrew-tap). It publishes
  `Formula/what-to-run.rb`.
- `SCOOP_BUCKET_TOKEN` is a token with Contents read/write on
  [zanza00/scoop-bucket](https://github.com/zanza00/scoop-bucket). The bucket
  must already exist with a commit on `main`, because GoReleaser creates
  `bucket/what-to-run.json` on that branch.
- `AUR_SSH_PRIVATE_KEY` is an unencrypted ed25519 private key whose public half
  is registered on your [AUR account](https://aur.archlinux.org/account).
  Generate one with
  `ssh-keygen -t ed25519 -f aur -C what-to-run-release`, add `aur.pub` to the
  account, and save the private `aur` file as the secret. The first release
  with this key creates the package.
- `APT_GPG_PRIVATE_KEY` is the armored private key that signs the apt
  repository, the RPM repository metadata and the RPM packages. Generate it
  with `gpg --batch --gen-key`. The first release with this key populates both
  repositories.

While an optional secret is unset, the release skips that channel with a
warning instead of failing. The Scoop step in GoReleaser has no "do nothing
without a token" mode, so the workflow passes `--skip=scoop`.

After the first AUR release, update the Arch Linux section in `docs/install.md`
to advertise `yay -S what-to-run-bin`.

### Check the setup

To check the release setup without publishing:

```sh
goreleaser check                      # validate .goreleaser.yaml
goreleaser release --snapshot --clean # build dist/ locally
python3 -m unittest discover -s scripts -p '*_test.py'
```

A snapshot writes `dist/scoop/bucket/what-to-run.json` with a `SNAPSHOT`
version. The Scoop pipe runs in the build phase, so the manifest is generated
without being published. The Homebrew formula comes from a custom publisher, so
only a real release writes it. `goreleaser check` catches a broken `scoops`
block before the tag.

### Recover a failed repository step

If the repository step fails after the GitHub release exists, run the
`republish packages` workflow with the tag as input. It updates `gh-pages`, the
branch GitHub Pages serves, from the released assets without recreating the
release.

The repository step also copies `scripts/setup.deb.sh` and
`scripts/setup.rpm.sh` to the root of `gh-pages`, where the one-liners in
`docs/install.md` fetch them from. The scripts are not tied to a version, so an
edit reaches the site with the next release or a `republish packages` run.

# Development

## Prerequisites

Install Go 1.27.2 or later and [`task`](https://taskfile.dev) (go-task 3.x):

```sh
go install github.com/go-task/task/v3/cmd/task@v3.54.0
```

Make sure `$(go env GOPATH)/bin` (or `$GOBIN`) is on your `PATH`. If `task` is
not installed, run the Go commands from `.github/workflows/checks.yml` directly.

## Commands

Run these commands from the repository root:

```sh
task build    # build bin/wtr
task test     # run tests with the race detector and coverage
task fmt-md   # format the Markdown files with dprint
task check-md # check the Markdown formatting
task all      # format (Go and Markdown), vet, test, and build
```

`task` with no arguments lists every task. `task fmt-md` and `task check-md`
download the pinned `dprint` binary into `bin/tools` on first run.

## Local build

To test your checkout in daily use, prepend its `bin/` directory to `PATH`
after any package manager setup (`brew shellenv`, `mise activate`) in your
shell rc file:

```sh
if [ -x "$HOME/path/to/what-to-run/bin/wtr" ]; then
  PATH="$HOME/path/to/what-to-run/bin:$PATH"
fi
```

Check which binary is active:

```sh
wtr --version # wtr 0.9.0 (release) or wtr v0.9.0-18-g1ff1695 (local build)
```

For a one-off widget test without changing `PATH`:

```sh
eval "$(task bind)"                  # zsh, Ctrl-O
eval "$(task bind-bash)"             # bash, Ctrl-O
eval "$(task bind KEY=ctrl-g)"       # zsh on another key
eval "$(task bind-bash KEY=ctrl-g)"  # bash on another key
```

## Windows

Cross-build the Windows binary from any platform:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/wtr.exe ./cmd/wtr
```

To test the PowerShell widget from the repository root:

```powershell
$env:Path = "$PWD\bin;$env:Path"
wtr.exe init powershell | Out-String | Invoke-Expression
```

## README demo

`docs/demo/demo.tape` records the GIF in the README:

```sh
task demo
```

`task demo` requires `vhs` (with `ttyd` and `ffmpeg`), ImageMagick 7
(`magick`), `gifski` and JetBrainsMono Nerd Font (`DEMO_FONT` overrides the
font path).

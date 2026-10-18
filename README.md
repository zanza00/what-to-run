# what-to-run

A task picker for the commands you can never remember in a repository.

I built `what-to-run` because I kept opening `package.json`, `Makefile`, or
`Cargo.toml` just to check what the project could run. It discovers those tasks
and makes them available from one key. It is first and foremost a **zsh
widget**: selected commands run through your shell and become part of your
normal terminal history.

## Screenshot

<!-- Add a screenshot at docs/images/what-to-run.png, then uncomment this line. -->
<!-- ![what-to-run running in a terminal](docs/images/what-to-run.png) -->

## Get started

### Install

On macOS or Linux with Homebrew (installs both `wtr` and `what-to-run`):

```sh
brew install zanza00/tap/what-to-run
```

Prebuilt tarballs with both binaries (macOS and Linux, Intel and ARM64) are
attached to each [release](https://github.com/zanza00/what-to-run/releases).

With Go installed (installs only `wtr`):

```sh
go install github.com/zanza00/what-to-run/cmd/wtr@latest
```

### Set up the zsh widget

Add this line to your `~/.zshrc` (use `wtr` instead if that is the command name
you installed):

```sh
eval "$(what-to-run init zsh)"
```

The widget is bound to **Ctrl-S** by default. To use another key, for example
Ctrl-G, pass it to `init`:

```sh
eval "$(what-to-run init zsh --key ctrl-g)"
```

Start a new shell, or evaluate the line in your current one, then press the
chosen key at the prompt.

### Use the picker

Type to filter the task list and use the arrow keys to select a task. Press
`enter` to run it: the widget submits the command through zsh, so it appears in
your shell history and its output stays in the terminal. Press `tab` to put the
selected command on the prompt without running it. Press `esc` to clear the
filter or close the picker.

### Choose a theme

Press `ctrl+t` while the picker is open to browse themes. Use the arrow keys to
preview them, then press `enter` (or `ctrl+t` again) to select and save your
choice. Press `esc` to cancel and restore the current theme. Your choice is
saved in your user configuration and applies to future runs.

## Task sources

The picker discovers tasks in the current project directory:

| File | Discovered tasks | Command |
| --- | --- | --- |
| `package.json` | `scripts` | `<package-manager> run <name>` |
| `Makefile`, `makefile`, `GNUmakefile` | Targets | `make <name>` |
| `Cargo.toml` | cargo-make `[tasks]` | `cargo make <name>` |

For `package.json`, the package manager is detected from the `packageManager`
field or a lockfile, with npm as the fallback. The task list shows descriptions
and each task's source file. If one source cannot be parsed, tasks from other
sources can still be shown.

## Running without the widget

You can also launch the picker directly from a terminal. Use the command name
available on your `PATH` (`wtr` is the short alias):

```sh
wtr          # full-screen picker
wtr --inline # inline picker, below the current prompt
```

Full-screen mode is the default. When you run a task there, the picker returns
after the task finishes; its output remains in the terminal scrollback. Inline
mode is intended for shell widgets and requires an interactive terminal.

## Configuration

Optional TOML files let you add custom tasks, override the package manager, or
disable task sources:

- `.what-to-run.toml` in a project, for settings shared with that repository;
- `$XDG_CONFIG_HOME/what-to-run/config.toml` for personal settings and
  per-project overrides (normally `~/.config/what-to-run/config.toml`).

See [Configuration](docs/config.md) for the full format and examples.

## Development

See [Development](docs/development.md) for build, test, and local widget
instructions.

## License

[MIT](LICENSE)


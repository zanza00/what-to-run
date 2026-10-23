<div align="center">

# What to Run (wtr)

A task picker for the commands you can never remember in a repository.

![what-to-run running in a terminal](docs/images/what-to-run.png)
</div>

## Why

I built `what-to-run` because I kept opening `package.json`, `Makefile`,
`Taskfile.yml`, `justfile`, or `Makefile.toml` just to check what a project
could run. It discovers those tasks and makes them available from one key. It
is first and foremost a **shell widget**: selected commands run through your
shell and become part of your normal terminal history.

## Get started

### Install

On macOS or Linux with Homebrew (installs both `wtr` and `what-to-run`):

```sh
brew install zanza00/tap/what-to-run
```

With [mise](https://mise.jdx.dev/) instead:

```sh
mise use -g github:zanza00/what-to-run
```

For the apt repository, `.deb` packages, release tarballs, or `go install`,
see [Installation](docs/install.md).

### Set up the widget

Add the line for your shell to its rc file, then start a new shell (or evaluate
it in the current one). Use `wtr` instead of `what-to-run` if that is the
command name you installed.

```sh
# zsh: ~/.zshrc
eval "$(what-to-run init zsh)"
```

```sh
# bash 4.3 or later: ~/.bashrc
eval "$(what-to-run init bash)"
```

The widget is bound to **Ctrl-S**. Pass `--key` to `init` to choose another
key, for example `--key ctrl-g`.

## Task sources

The picker discovers tasks in the current project directory:

| File | Discovered tasks | Command |
| --- | --- | --- |
| `package.json` | `scripts` | `<package-manager> run <name>` |
| `Makefile`, `makefile`, `GNUmakefile` | Targets | `make <name>` |
| `Taskfile.yml` and the other go-task default names | `tasks` (internal tasks hidden) | `task <name>` |
| `justfile`, `Justfile`, `.justfile` | Recipes (private recipes hidden) | `just <name>` |
| `Makefile.toml` | cargo-make `[tasks]` | `cargo make <name>` |
| `.cargo/config`, `.cargo/config.toml` | `[alias]` (chains resolved for the description) | `cargo <name>` |
| `devbox.json` | `shell.scripts` | `devbox run <name>` |
| `composer.json` | custom `scripts` (lifecycle events are not listed) | `composer run-script <name>` |
| `deno.json`, `deno.jsonc` | `tasks` | `deno task <name>` |
| `mise.toml`, `.mise.toml` | `[tasks]` | `mise run <name>` |

### Caveats

- The task list shows descriptions and each task's source file. If one source
  cannot be parsed, tasks from the other sources are still shown.
- For `package.json`, the package manager is detected from the `packageManager`
  field or a lockfile, falling back to npm.
- Composer lifecycle events are skipped because Composer runs them itself.
- For Taskfiles, only the tasks in the file itself are listed: `includes` and
  remote Taskfiles are not followed, aliases are not listed as extra rows and
  `label` does not rename a task. Internal tasks are hidden; a task described
  only by a template shows the template as written.
- For justfiles, only the recipes in the file itself are listed: `import` and
  `mod` files are not followed and aliases are not listed as extra rows. Private
  recipes are hidden (a `_` prefix or `[private]`); `@` only marks a silent
  recipe, which is still listed. Recipe parameters do not appear in the command
  line: the picker runs `just <name>` and lets just apply the defaults. Groups
  are flattened into one alphabetical list, so the order differs from
  `just --list`, which prints grouped recipes after the ungrouped ones. Because
  the parser works line by line, a column-0 line inside a `'''...'''` string can
  be mistaken for a recipe header.

## Running without the widget

You can also launch the picker directly from a terminal, using the command
name available on your `PATH` (`wtr` is the short alias):

```sh
wtr          # full-screen picker
wtr --inline # inline picker, below the current prompt
```

Full-screen mode is the default. When you run a task there, the picker returns
after the task finishes; its output remains in the terminal scrollback. Inline
mode is intended for shell widgets and requires an interactive terminal. If the
current directory declares no tasks, the inline picker stays open on a message
listing the files it looked for (close it with `enter` or `esc`), while a
full-screen run reports the error and exits.

## Configuration

Two optional TOML files let you add custom tasks, override the package manager,
or disable task sources:

- `.what-to-run.toml` in a project, for settings shared with that repository;
- `$XDG_CONFIG_HOME/what-to-run/config.toml` for personal settings and
  per-project overrides (normally `~/.config/what-to-run/config.toml`).

See [Configuration](docs/config.md) for the full format and examples.

### Themes

While the picker is open, press `ctrl+t` to cycle through the themes. The one
you choose is saved automatically to your user config and used from then on.

## Development

See [Development](docs/development.md) for build, test, and local widget
instructions.

## License

[MIT](LICENSE)

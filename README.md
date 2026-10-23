<div align="center">

# What to Run (wtr)

A task picker for the commands you can never remember.

![what-to-run running in a terminal](docs/images/what-to-run.png)
</div>

## Why

I built `what-to-run` because I kept opening project files just to check what
a project could run. It discovers those tasks and makes them available from one
key. It is first and foremost a **shell widget**: selected commands run through
your shell and become part of your normal terminal history.

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

For the apt repository, `.deb` packages, release tarballs, Windows or
`go install`, see [Installation](docs/install.md).

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

```powershell
# PowerShell 5.1 or 7+, in $PROFILE
what-to-run init powershell | Out-String | Invoke-Expression
```

The widget is bound to **Ctrl-S**. See [Widget setup](docs/widget.md) for other
keys, shell dialects and running the picker without the widget.

## Task sources

The picker reads tasks from:

- `package.json`
- `Makefile`
- `Taskfile`
- `justfile`
- `Makefile.toml`
- Cargo aliases (`.cargo/config`)
- `devbox.json`
- `composer.json`
- `deno.json`
- `mise.toml`

See [Task sources](docs/task-sources.md) for the commands and caveats.

## Configuration

Two optional TOML files add custom tasks, override the package manager, disable
sources and set the theme. See [Configuration](docs/config.md) for the format,
per-project overrides and the `ctrl+t` theme picker.

## Inspiration

This project is heavily inspired by [television](https://github.com/alexpasmantier/television), [fzf](https://github.com/junegunn/fzf) and [lazygit](https://github.com/jesseduffield/lazygit)

## License

[MIT](LICENSE)

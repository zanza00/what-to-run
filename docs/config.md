# Configuration

`wtr` reads two optional TOML files: a global file for your personal defaults
and per-project overrides, and a project file that you can commit to a
repository to share settings with everyone who works on it.

## Global file

The global file is `$XDG_CONFIG_HOME/what-to-run/config.toml`. When
`XDG_CONFIG_HOME` is unset, `wtr` reads:

- `~/.config/what-to-run/config.toml` on Linux
- `~/Library/Application Support/what-to-run/config.toml` on macOS
- `%AppData%\what-to-run\config.toml` on Windows

You can also pick and save `theme` directly from the picker (see
[Themes](#themes)).

```toml
# UI theme (global file only).
theme = "dracula"

# Default package manager for package.json ("npm", "pnpm", "yarn", "bun").
package_manager = "npm"

# Disabled sources, by ID (see Task sources).
disable = ["cargo-make"]

# Custom tasks available in every project.
[[tasks]]
name = "status"
command = "git status -sb"
description = "Short git status"

# Per-project overrides for a directory and its subdirectories.
[projects."~/work"]
package_manager = "pnpm"

[projects."~/work/legacy-service"]
package_manager = "yarn"
disable = ["make"]

[[projects."~/work/legacy-service".tasks]]
name = "dev"
command = "yarn dev --port 3001"
description = "Run local dev server"
cwd = "packages/web"
```

## Project file

The project file is `.what-to-run.toml` in the current directory. It supports
`package_manager`, `disable` and `[[tasks]]`:

```toml
package_manager = "bun"
disable = ["make"]

[[tasks]]
name = "deploy"
command = "scripts/deploy.sh"
description = "Deploy the service"
cwd = "ops"
```

Both files use the same `[[tasks]]` fields:

- `name` and `command` are required.
- `description` is optional and defaults to the command.
- `cwd` is optional, relative to the project directory, and must stay inside it.

`disable` takes source IDs (`npm`, `make`, `cargo-make`), not file names. See
[Task sources](task-sources.md) for the full list of IDs.

## Per-project overrides

A `[projects."<path>"]` section in the global file sets `package_manager`,
`disable` and `[[tasks]]` for a directory and all its subdirectories. Use it to
customize a project without adding `.what-to-run.toml` to the repository.

The path must be absolute or start with `~`. `wtr` resolves symlinks before
matching, and when multiple entries match the current directory, the deepest
path wins.

On Windows, wrap the path in single quotes so backslashes are not parsed as TOML
escapes:

```toml
[projects.'C:\Users\me\src\api']
package_manager = "pnpm"
```

## Precedence

Settings merge from the most general layer to the most specific:

1. Global top-level settings
2. `.what-to-run.toml` in the project
3. Matching `[projects."<path>"]` sections, from parent directory to deepest
   subdirectory

| Field | Rule |
| --- | --- |
| `theme` | Read only from the global top level (`--theme` overrides it for one run). |
| `package_manager` | The most specific value wins. |
| `disable` | All layers combine. |
| `tasks` | More specific layers come first and override same-named tasks from lower layers or discovered sources. |

## Themes

Press `ctrl+t` inside the picker to open the theme picker. `↑` and `↓` preview
each theme live, `enter` saves the choice to the global file, and `esc`
restores the previous theme.

From the command line, `--theme <name>` sets a theme for one run,
`--list-themes` prints the available names, and `--preview` renders a static
preview.

## Warnings

`wtr` skips unknown keys, wrong types and invalid entries with a warning and
keeps the rest of the file.

When there are warnings, the bottom bar shows the count and the `ctrl+w` hint.
The warning modal opens automatically at startup when a new warning appears:

- `enter`, `esc` or `ctrl+w` closes the modal for the session.
- `↑` and `↓` move between warnings.
- `d` dismisses the selected warning for future runs (saved in
  `$XDG_CACHE_HOME/what-to-run/dismissed-warnings`, or
  `%LocalAppData%\what-to-run\dismissed-warnings` on Windows).

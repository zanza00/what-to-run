# Configuration

`wtr` reads two optional TOML files. The global file holds your preferences,
global tasks and per-project overrides. The project file holds settings that a
repository shares with everyone who works on it.

Neither file is required. A problem in a file is a warning, not an error: `wtr`
keeps every valid setting and task. The [Warnings](#warnings) section covers
how the picker shows them.

## Schema

### Global file

The global file is `$XDG_CONFIG_HOME/what-to-run/config.toml`. Without
`XDG_CONFIG_HOME`, the path is `~/.config/what-to-run/config.toml` on Linux,
`~/Library/Application Support/what-to-run/config.toml` on macOS, and
`%AppData%\what-to-run\config.toml` on Windows.

```toml
# User UI theme (only valid in the global file).
theme = "dracula"

# Global default package manager override ("npm", "pnpm", "yarn", "bun").
package_manager = "npm"

# Sources disabled globally, by ID (see Task sources).
disable = ["cargo-make"]

# Global custom tasks, available in every project unless shadowed.
[[tasks]]
name = "status"
command = "git status -sb"
description = "Short git status"

# Per-project overrides: keyed by directory path.
# Applies to the directory itself and any subdirectory beneath it.
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

Both files use the same `[[tasks]]` fields. `name` and `command` are required.
`description` defaults to the command. `cwd` is relative to the project
directory and must stay inside it.

### Project file

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

`wtr` warns about `theme` and `projects` in a project file and ignores them.

### Built-in task sources

[Task sources](task-sources.md) lists the supported task files, the commands,
how each source is discovered and parsed, and the stable ID that `disable`
uses. `disable` takes those IDs (`npm`, `make`, `cargo-make`), not file names.
See [Source IDs](task-sources.md#source-ids). A `disable` entry that is a file
name is ignored with a warning that names the ID to use. A package manager
name is not a source ID either: to disable `package.json` scripts, use `npm`.

### Themes

`theme` selects the look and is read only from the global file. While the
picker is open, `ctrl+t` opens the theme picker. `↑`/`↓` preview a theme live.
`enter` or `ctrl+t` keeps the choice and saves it to the global file. `esc`
restores the theme that was active when the picker opened. `--theme <name>`
pins a theme for one run, `--list-themes` lists the names, and `--preview`
prints a static frame.

## Per-project overrides

A `[projects."<path>"]` table in the global file sets `package_manager`,
`disable` and `[[tasks]]` for one directory and everything below it. Use it to
customize a repository without adding a `.what-to-run.toml` to that repository.

### Path resolution

`wtr` prepares each project key and the current directory before it matches
them:

1. A leading `~` or `~/` expands to the home directory (`os.UserHomeDir()`).
2. A relative path is rejected with a warning. A project key must be absolute
   or start with `~`.
3. Both paths are normalized with `filepath.EvalSymlinks`, which resolves the
   longest part of the path that exists, and the result is cleaned with
   `filepath.Clean`. So `~/dev/app` (`/home/user/dev/app`) matches
   `/var/home/user/dev/app` on ostree/Silverblue systems and symlinked
   workspaces.

On Windows, write the key as a TOML literal string:

```toml
[projects.'C:\Users\me\src\api']
package_manager = "pnpm"
```

A basic string (`[projects."C:\Users\me\src\api"]`) fails to parse, because
`\U` starts a Unicode escape in TOML. Windows paths are not case-sensitive, so
`C:\Users\me\src\api` also matches `c:\users\me\src\api`.

### Subtree matching

A `[projects."<path>"]` entry matches the current directory when the current
directory is `<path>` or a subdirectory inside it. The check uses path-segment
boundaries, so `~/dev/app` matches `~/dev/app` and `~/dev/app/web`, but never
`~/dev/app-2`.

When several entries match, `wtr` orders them by specificity: number of path
segments, then path length, then alphabetical order. It applies them from the
least specific to the most specific.

## Layering and precedence

Settings merge from the most general layer to the most specific:

1. The global top level.
2. The project file.
3. Matching `[projects."<path>"]` blocks, from the outermost directory to the
   deepest.

How each field merges:

| Field | Merge rule |
| --- | --- |
| `theme` | Read only from the global top level. `--theme` overrides it for one run. |
| `package_manager` | The most specific non-empty value wins. |
| `disable` | Every source ID from every layer applies. |
| `tasks` | Tasks from more specific layers come first and shadow same-named tasks from lower layers and discovered sources. Inside a layer, tasks keep their order. |

## Best-effort parsing

`wtr` keeps as much valid information as possible, even when part of a file is
malformed or comes from another version of `wtr`:

1. A TOML syntax error (an unclosed quote, a broken table header) skips that
   file. The other file still loads.
2. An unknown key at the top level, in `[[tasks]]` or in
   `[projects."<path>"]` produces a warning that names the key, for example
   `unknown key "packge_manager"`. Every other valid key and section in the
   file still applies.
3. At the top level, a field with the wrong type drops that field with a
   warning. For example, `disable` must be an array and `theme` must be a
   string. In a task, a wrong type on `name`, `command` or `cwd` drops the
   task, while a wrong `description` drops only the description.
4. Invalid entries:
   - An unknown `package_manager` or `theme` is dropped with a warning.
   - A `disable` entry that is not a source ID is dropped with a warning that
     names the ID to use. The other entries stay.
   - An invalid task is dropped on its own. The causes are an empty `name`, an
     empty `command`, a duplicate `name`, or a `cwd` that escapes the project
     with `..`. The other tasks in the list still load.
   - An invalid `[projects."<path>"]` entry (an empty or relative path, or a
     bad setting or task inside it) drops only the bad entry or field. Other
     projects and global settings are unaffected.

## Warnings

Warnings from config files and task sources never hide the bottom help line:

- When there are warnings, the bottom bar shows the task count and the warning
  count. The warning count uses the theme's error color, for example
  `11 tasks · 1 warning`. The bar also shows the `ctrl+w warnings` hint.
- The warning modal (`ctrl+w`) opens automatically at startup when at least
  one warning has not been dismissed. Long messages wrap across lines,
  breaking on spaces, hyphens and path separators.
  - `enter`, `esc` or `ctrl+w` closes the modal for the current session.
  - `↑`/`↓` moves between warnings when there are several.
  - `d` stops the modal from opening for the warning in future runs. `wtr`
    records it in `$XDG_CACHE_HOME/what-to-run/dismissed-warnings`
    (`%LocalAppData%\what-to-run\dismissed-warnings` on Windows). `ctrl+w`
    still shows the warning on demand.

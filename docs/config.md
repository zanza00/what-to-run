# Configuration (`config.toml` & `.what-to-run.toml`)

`what-to-run` (`wtr`) reads two optional TOML configuration files:

1. **User global config:** `$XDG_CONFIG_HOME/what-to-run/config.toml`
   (normally `~/.config/what-to-run/config.toml`; on macOS with
   `XDG_CONFIG_HOME` unset, `~/Library/Application Support/what-to-run/config.toml`;
   on Windows, `%AppData%\what-to-run\config.toml`)
   — personal preferences, global custom tasks, and per-project overrides
   (`[projects."<path>"]`).
2. **Project local config:** `.what-to-run.toml` in the current working
   directory — project settings shared with the repository.

Neither file is required. Any problem in a configuration file is reported as a
non-fatal **warning** (shown in the warning modal at startup and accessible via
`ctrl+w`), while every valid setting and task in the file still applies.

---

## 1. Schema

### Global file (`~/.config/what-to-run/config.toml`, or `%AppData%\what-to-run\config.toml` on Windows)

```toml
# User UI theme (only valid in the global file).
theme = "dracula"

# Global default package manager override ("npm", "pnpm", "yarn", "bun").
package_manager = "npm"

# Sources disabled globally, by any of their file names (package.json,
# Makefile, Taskfile.yml, justfile, Makefile.toml, .cargo/config, devbox.json,
# composer.json, deno.json, mise.toml).
disable = ["Makefile.toml"]

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
disable = ["Makefile"]

[[projects."~/work/legacy-service".tasks]]
name = "dev"
command = "yarn dev --port 3001"
description = "Run local dev server"
cwd = "packages/web"
```

### Project file (`.what-to-run.toml`)

The project file supports `package_manager`, `disable`, and `[[tasks]]`:

```toml
package_manager = "bun"
disable = ["Makefile"]

[[tasks]]
name = "deploy"
command = "scripts/deploy.sh"
description = "Deploy the service"
cwd = "ops"
```

User-only keys (`theme` and `projects`) are not allowed in a project
`.what-to-run.toml`: if present, they are ignored with a warning.

### Built-in task sources

The supported task files, their commands, and how each source is discovered
and parsed are documented in [Task sources](task-sources.md). The `disable`
setting above accepts any of their file names.

### Themes

The `theme` key selects the look and is read only from the global file. While
the picker is open, `ctrl+t` opens the theme picker: `↑`/`↓` preview live,
`enter` keeps and saves the choice to the global file, and `esc` restores the
theme that was active when the picker opened. `--theme <name>` pins a theme for
one run, `--list-themes` lists the available names, and `--preview` renders a
static frame.

---

## 2. Per-Project Overrides (`[projects."<path>"]`)

The `[projects."<path>"]` table in the user's global `config.toml` lets you
customize `package_manager`, `disable`, and `[[tasks]]` for specific directories
or workspace trees without creating a `.what-to-run.toml` inside those
repositories.

### Path Resolution and Canonicalization

Before matching a project key against the current working directory (`dir`):

1. **Home expansion:** A leading `~` or `~/` is expanded to the user's home
   directory (`os.UserHomeDir()`).
2. **Absolute resolution:** Relative paths in `[projects."..."]` are rejected
   with a warning (project keys in the global config must be absolute or start
   with `~`).
3. **Symlink canonicalization:** Both the configured path and the current
   working directory are normalized with `filepath.EvalSymlinks` (falling back
   to `filepath.Clean` when a path does not exist on disk yet). This guarantees
   that `~/dev/app` (`/home/user/dev/app`) matches `/var/home/user/dev/app` on
   ostree/Silverblue systems or symlinked workspaces.

On Windows, write the key as a TOML **literal string**:

```toml
[projects.'C:\Users\me\src\api']
package_manager = "pnpm"
```

A basic string (`[projects."C:\Users\me\src\api"]`) fails to parse: `\U` starts
a Unicode escape in TOML. Paths are compared case-insensitively on Windows, so
`C:\Users\me\src\api` also matches `c:\users\me\src\api`.

### Hierarchical Subtree Matching

A `[projects."<path>"]` entry matches the current directory if the canonicalized
current directory is `<path>` itself or a subdirectory inside `<path>` (checked
on path-segment boundaries via `filepath.Rel`, so `~/dev/app` matches
`~/dev/app` and `~/dev/app/web`, but **never** `~/dev/app-2`).

When multiple `[projects."..."]` entries match the current directory (for
example `[projects."~/work"]` and `[projects."~/work/api"]`), they are ordered
by **specificity** (number of path segments, then path length, then lexical
order for ties) from **least specific to most specific**.

---

## 3. Layering & Precedence

Settings are merged in order from the most general to the most specificuser
override:

1. **Global top-level** (`~/.config/what-to-run/config.toml`)
2. **Project local file** (`.what-to-run.toml` in the working directory)
3. **Matching `[projects."<path>"]` blocks** in the global file, from outermost
   parent directory to the deepest matching directory.

How each field merges across layers:

| Field | Merge rule |
| --- | --- |
| `theme` | Read only from the global top-level config (overridden for a single run by `--theme`). |
| `package_manager` | Later (more specific) non-empty layer wins. |
| `disable` | Additive union across all active layers. |
| `tasks` | Higher-precedence layers come first and shadow same-named tasks from lower layers (and discovered sources). Within a layer, tasks keep their declaration order. |

---

## 4. Resilient Best-Effort TOML Parsing

Configuration files are parsed to **recover as much valid information as
possible** even when part of a file is malformed or comes from a newer/older
version of `wtr`:

1. **Syntax errors:** Only a grammatical TOML syntax error (e.g. an unclosed
   quote or broken table header) skips that file; the other config file still
   loads.
2. **Unknown keys:** An unrecognized key at the top level, inside `[[tasks]]`,
   or inside `[projects."<path>"]` produces a specific warning naming the key
   (e.g. `unknown key "packge_manager"`), and **all other valid keys and
   sections in the same file are still applied**.
3. **Wrong types:** A field with the wrong TOML type (e.g. `disable = "Makefile"`
   instead of an array, or `theme = 123`) drops only that field with a warning;
   the rest of the file and section is preserved.
4. **Invalid entries:**
   - An unknown `package_manager` or `theme` is dropped with a warning.
   - An invalid task in `[[tasks]]` (empty `name`, empty `command`, duplicate
     `name`, or `cwd` escaping the project with `..`) is dropped individually;
     sibling tasks in the same `[[tasks]]` list still load.
   - An invalid `[projects."<path>"]` entry (empty/relative path, or a bad
     setting/task inside it) drops only the bad entry or field without affecting
     other projects or global settings.

---

## 5. Warning Display & Dismissal

Warnings from config files and task sources never hide the bottom help line:

- **Bottom chrome:** When warnings are present, the bottom bar shows the task
  count alongside the warning count in the theme's error color (`11 tasks · 1
  warning`) and the `ctrl+w warnings` key hint.
- **Warning modal (`ctrl+w`):** Opens automatically at startup when there is at
  least one undismissed warning, wrapping long messages across multiple lines.
  - `enter`, `esc`, or `ctrl+w` closes the modal for the current session.
  - `d` (*don't show again*) records the warning in
    `$XDG_CACHE_HOME/what-to-run/dismissed-warnings`
    (`%LocalAppData%\what-to-run\dismissed-warnings` on Windows) so future runs do
    not pop the modal open automatically for that same warning, while `ctrl+w`
    can still open it on demand.

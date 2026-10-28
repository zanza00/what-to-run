# Task sources

`wtr` discovers tasks in the current project directory. The `justfile` and
`Taskfile` sources also look in parent directories, like `just` and `task`
themselves do. The nearest file wins, and its badge is the path relative to the
current directory (for example `../../justfile`).

In the picker, each row shows the command line, the description and the source
badge. A source that cannot be parsed reports a warning and does not hide the
tasks of the other sources.

| ID | File | Discovered tasks | Command |
| --- | --- | --- | --- |
| `npm` | `package.json` | `scripts` | `<package-manager> run <name>` |
| `make` | `Makefile`, `makefile`, `GNUmakefile` | Targets | `make <name>` |
| `task` | `Taskfile.yml`, `taskfile.yml`, `Taskfile.yaml`, `taskfile.yaml`, `Taskfile.dist.yml`, `taskfile.dist.yml`, `Taskfile.dist.yaml`, `taskfile.dist.yaml` | `tasks` and local `includes` (internal tasks and internal includes hidden) | `task <name>` |
| `just` | `justfile`, `Justfile`, `.justfile` | Recipes, local `import`s and modules (private recipes hidden) | `just <name>` |
| `cargo-make` | `Makefile.toml` | cargo-make `[tasks]` | `cargo make <name>` |
| `cargo-alias` | `.cargo/config`, `.cargo/config.toml` | `[alias]` (chains resolved for the description) | `cargo <name>` |
| `devbox` | `devbox.json` | `shell.scripts` | `devbox run <name>` |
| `composer` | `composer.json` | custom `scripts` (lifecycle events excluded) | `composer run-script <name>` |
| `deno` | `deno.json`, `deno.jsonc` | `tasks` | `deno task <name>` |
| `mise` | `mise.toml`, `.mise.toml` | `[tasks]` | `mise run <name>` |

Four sources accept comments and trailing commas, because `wtr` parses them as
JSONC: `devbox.json`, `deno.json`, `deno.jsonc` and `composer.json`.

## Source IDs

`disable` names a source by its ID. IDs are part of the configuration format,
so a new source chooses one by these rules:

1. The ID is lowercase kebab-case (`npm`, `cargo-make`). Such a name is also a
   valid bare TOML key, so a future `[sources.cargo-make]` table needs no
   quotes.
2. The ID names the tool, not the file (`make`, `just`, `mise`, `deno`). Add a
   kebab-case suffix when one tool has several families (`cargo-make`,
   `cargo-alias`). `npm` is the family name for `package.json` scripts,
   whatever package manager runs them.
3. An ID is never a file name. No ID equals a file that any source reads, so a
   file name in `disable` is never ambiguous with a valid ID.
4. The ID is unique across the registry.
5. The ID is stable. Renaming it breaks existing configuration like renaming a
   key, so a source keeps its ID instead of following a new file name.

A file name in `disable` warns with the ID to use and disables nothing.
`TestRegistryInvariants` in `internal/core` enforces the machine-checkable
rules.

## package.json

The package manager comes from the `package_manager` setting, then the
`packageManager` field, then a lockfile (`bun.lock`, `bun.lockb`,
`pnpm-lock.yaml`, `yarn.lock`, `package-lock.json`). If none of these applies,
`wtr` uses npm.

## Composer

Lifecycle and plugin events are skipped because Composer runs them itself. A
script is a command string or an array of commands. For an array, the
description joins the commands with `&&`. The `scripts-descriptions` field
overrides the description.

## Taskfiles

`wtr` follows local `includes` recursively, with paths relative to the
including file, and searches a directory include for the default Taskfile
names. An included task gets the include key as a namespace (`docs:build`),
unless the include sets `flatten: true`. An `internal` include hides all its
tasks.

`wtr` does not follow remote or templated include paths. It skips a missing,
unreadable or unparsable include (optional or not), without losing the other
tasks. On a flatten collision, the first definition wins (the root file before
the includes).

`wtr` does not list aliases as extra rows, and `label` does not rename a task.
It hides internal tasks. A task described only by a template shows the template
as written.

## justfiles

`wtr` follows local `import`s recursively, relative to the importing file. An
imported recipe keeps the same name and command, so the picker still runs
`just <name>`.

Modules (`mod`) are followed too. The source is `name.just`, `name/mod.just`,
`name/justfile` or `name/.justfile`. The last two match in any capitalization.
An explicit quoted path may name the file or its directory. The module's
recipes are listed as `name::recipe`, chained for nested modules.

`wtr` skips a missing, unreadable or unparsable file, and a cycle ends instead
of recursing. When a name is defined more than once, the shallowest file wins.
At equal depth, the first definition in source order wins.

`wtr` does not list aliases as extra rows. It hides private recipes (a `_`
prefix or `[private]`). `@` only marks a silent recipe, which is still listed.

Recipe parameters do not appear in the command line. The picker runs
`just <name>` and lets just apply the defaults. Groups are flattened into one
alphabetical list, so the order differs from `just --list`, which prints
grouped recipes after the ungrouped ones. `wtr` reads the justfile line by
line, so a line that starts at column 0 inside a `'''...'''` string can look
like a recipe header.

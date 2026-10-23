# Task sources

`wtr` discovers tasks in the current project directory. The `justfile` and
`Taskfile` sources also look in parent directories, like `just` and `task`
themselves do: the nearest file wins, and its badge is the path relative to the
current directory (for example `../../justfile`).

Each row shows the task's description and source file. A source that cannot be
parsed only warns and does not hide the tasks of the other sources.

| File | Discovered tasks | Command |
| --- | --- | --- |
| `package.json` | `scripts` | `<package-manager> run <name>` |
| `Makefile`, `makefile`, `GNUmakefile` | Targets | `make <name>` |
| `Taskfile.yml`, `taskfile.yml`, `Taskfile.yaml`, `taskfile.yaml`, `Taskfile.dist.yml`, `taskfile.dist.yml`, `Taskfile.dist.yaml`, `taskfile.dist.yaml` | `tasks` and local `includes` (internal tasks and internal includes hidden) | `task <name>` |
| `justfile`, `Justfile`, `.justfile` | Recipes, local `import`s and modules (private recipes hidden) | `just <name>` |
| `Makefile.toml` | cargo-make `[tasks]` | `cargo make <name>` |
| `.cargo/config`, `.cargo/config.toml` | `[alias]` (chains resolved for the description) | `cargo <name>` |
| `devbox.json` | `shell.scripts` | `devbox run <name>` |
| `composer.json` | custom `scripts` (lifecycle events excluded) | `composer run-script <name>` |
| `deno.json`, `deno.jsonc` | `tasks` | `deno task <name>` |
| `mise.toml`, `.mise.toml` | `[tasks]` | `mise run <name>` |

`devbox.json`, `deno.jsonc` and the other JSON-with-comments files are parsed
as JSONC, so comments and trailing commas are accepted.

## package.json

The package manager comes from the `package_manager` setting, then the
`packageManager` field, then a lockfile (`bun.lock`, `bun.lockb`,
`pnpm-lock.yaml`, `yarn.lock`, `package-lock.json`), falling back to npm.

## Composer

Lifecycle and plugin events are skipped because Composer runs them itself. A
script is a command string or an array of commands (joined with `&&` for the
description), and `scripts-descriptions` overrides the description.

## Taskfiles

Local `includes` are followed recursively, with paths relative to the including
file (a directory include is searched for the default Taskfile names). An
included task is namespaced with the include key (`docs:build`) unless the
include sets `flatten: true`; an `internal` include hides all its tasks. Remote
and templated include paths are not followed, and a missing include (optional
or not) or an unparsable one is skipped without losing the other tasks. On a
flatten collision the first definition wins (the root file before the
includes).

Aliases are not listed as extra rows and `label` does not rename a task.
Internal tasks are hidden; a task described only by a template shows the
template as written.

## justfiles

Local `import`s are followed recursively, relative to the importing file: an
imported recipe keeps the root file's name and command, so the picker still
runs `just <name>`. Modules (`mod`) are followed too: the source is
`name.just`, `name/mod.just` or `name/justfile`/`name/.justfile` in any
capitalization, an explicit quoted path may name the file or its directory,
and the module's recipes are listed as `name::recipe`, chained for nested
modules. A missing, unreadable or unparsable file is skipped, and a cycle
terminates instead of recursing. When a name is defined more than once, the
shallowest file wins and, at equal depth, the earlier include.

Aliases are not listed as extra rows. Private recipes are hidden (a `_` prefix
or `[private]`); `@` only marks a silent recipe, which is still listed. Recipe
parameters do not appear in the command line: the picker runs `just <name>`
and lets just apply the defaults. Groups are flattened into one alphabetical
list, so the order differs from `just --list`, which prints grouped recipes
after the ungrouped ones. Because the parser works line by line, a column-0
line inside a `'''...'''` string can be mistaken for a recipe header.

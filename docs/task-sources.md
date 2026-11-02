# Task sources

`wtr` discovers tasks in the current directory. For `Taskfile`, `justfile`,
Poe tasks, PDM scripts, taskipy tasks and Hatch scripts, it also searches
parent directories (nearest file wins) and shows the relative path in the
badge.

| ID | File | Discovered tasks | Command |
| --- | --- | --- | --- |
| `npm` | `package.json` | `scripts` | `<package-manager> run <name>` |
| `make` | `Makefile`, `makefile`, `GNUmakefile` | Targets | `make <name>` |
| `task` | `Taskfile.yml`, `taskfile.yml`, `Taskfile.yaml`, `taskfile.yaml`, `Taskfile.dist.yml`, `taskfile.dist.yml`, `Taskfile.dist.yaml`, `taskfile.dist.yaml` | `tasks` and local `includes` | `task <name>` |
| `just` | `justfile`, `Justfile`, `.justfile` | Recipes, local `import`s and modules | `just <name>` |
| `cargo-make` | `Makefile.toml` | `[tasks]` | `cargo make <name>` |
| `cargo-alias` | `.cargo/config`, `.cargo/config.toml` | `[alias]` | `cargo <name>` |
| `devbox` | `devbox.json` | `shell.scripts` | `devbox run <name>` |
| `composer` | `composer.json` | custom `scripts` | `composer run-script <name>` |
| `deno` | `deno.json`, `deno.jsonc` | `tasks` | `deno task <name>` |
| `mise` | `mise.toml`, `.mise.toml` | `[tasks]` | `mise run <name>` |
| `poe` | `pyproject.toml`, `poe_tasks.toml` | `[tool.poe.tasks]` | `poe <name>` |
| `pdm` | `pyproject.toml` | `[tool.pdm.scripts]` | `pdm run <name>` |
| `taskipy` | `pyproject.toml` | `[tool.taskipy.tasks]` | `task <name>` |
| `hatch` | `pyproject.toml` | `[tool.hatch.envs.<env>.scripts]` | `hatch run <env>:<script>` |

Use the `ID` column in the `disable` setting (see [Configuration](config.md)).
`devbox.json`, `deno.json`, `deno.jsonc` and `composer.json` accept comments and
trailing commas (JSONC).

## package.json

`wtr` picks the package manager from the `package_manager` setting, then the
`packageManager` field in `package.json`, then the lockfile (`bun.lock`,
`bun.lockb`, `pnpm-lock.yaml`, `yarn.lock`, `package-lock.json`), and defaults
to `npm`.

## Composer

`wtr` skips lifecycle and plugin events because Composer runs them
automatically. When a script is an array of commands, the description joins
them with `&&` unless `scripts-descriptions` sets a custom description.

## Taskfiles

`wtr` follows local file and directory `includes` recursively. Included tasks
use `<include>:<task>` unless the include sets `flatten: true`. Internal tasks,
internal includes, remote includes and templated include paths are skipped.

## justfiles

`wtr` follows local `import`s and `mod` modules recursively. Module recipes are
listed as `module::recipe`. Private recipes (`_` prefix or `[private]`) and
aliases are hidden, and recipes with parameters run with their default values.

## Poe tasks

Tasks of every poe type are listed: strings, arrays and the typed tables. A
task's `help` is its description. Without one, the body shows as text, such
as the command, the shell line or the Python reference. Tasks with a `_`
prefix stay hidden, as in poe's own listing.

`poe_tasks.toml` works without a `pyproject.toml`: its tasks sit at the top
level or under `[tool.poe.tasks]`.

## PDM scripts

Scripts of every pdm shape are listed: strings and the `cmd`, `shell`, `call`
and `composite` tables. A `cmd` given as an array joins with a space, because
pdm splits one command across it. A script's `help` is its description.
Without one, the body shows as text. The shared-options `_` key and scripts
with a `_` prefix stay hidden, as in pdm's own listing.

## taskipy tasks

Both taskipy shapes are listed: strings and the `cmd`/`help` tables. A task's
`help` is its description. Without one, the command shows as text. The
`[tool.taskipy.variables]` and `[tool.taskipy.settings]` tables declare no
tasks, so they contribute nothing.

## Hatch scripts

Every script is listed as `<env>:<script>`, because two environments can
declare the same script name. The environment's `description` is the
description of its scripts. Without one, the script shows as text, and an
array script joins its commands. Environments named with the `hatch-` prefix,
and environments with a `matrix` or an `overrides` table, stay hidden, because
their names expand or mutate at run time.

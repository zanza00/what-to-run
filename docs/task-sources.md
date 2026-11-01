# Task sources

`wtr` discovers tasks in the current directory. For `Taskfile` and `justfile`,
it also searches parent directories (nearest file wins) and shows the relative
path in the badge.

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

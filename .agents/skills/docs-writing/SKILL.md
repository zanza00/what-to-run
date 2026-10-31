---
name: docs-writing
description: Write or edit README.md, docs/*.md and CHANGELOG.md, or document a new task source.
---

# Write the what-to-run docs

## Ownership

Every fact has a **single source of truth**: one file states it in full, and every other page links to that file by title (`see [Configuration](docs/config.md)`).

| File | Owns |
| --- | --- |
| `README.md` | pitch, install shortcuts, widget one-liner, source bullet list, config summary, inspiration, license |
| `docs/install.md` | install channels, setup scripts, release file naming |
| `docs/widget.md` | zsh, bash and PowerShell widget setup, key binding, fullscreen toggle, direct runs |
| `docs/config.md` | rc schema, per-project overrides, precedence, themes and warnings |
| `docs/task-sources.md` | source registry table and per-source caveats |
| `docs/development.md` | contributor prerequisites, dev tasks, local build, README demo |
| `CHANGELOG.md` | the `[Unreleased]` section, the entry style, the released version sections |

When adding a task source, add one row to `docs/task-sources.md` and one bullet to `README.md` (`## Task sources`) in `defaultSources` registry order. Add a `##` section in `docs/task-sources.md` only when the table cannot hold the caveats.

## Changelog entries

`CHANGELOG.md` follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/): one `[Unreleased]`
section on top, one section per released version, and entries grouped under
`### Added`, `### Changed` and `### Fixed`.

A pull request that changes user-visible behavior adds its entry under
`[Unreleased]` in the same pull request. The `changelog` CI job fails without
it, and the `skip-changelog` label opts a pull request out. The `release`
skill renames `[Unreleased]` into the released version section, so never add
news to a released section.

Write entries like the existing ones:

- One entry per change, in the group the change belongs to.
- Lead with the user-visible outcome, not the implementation: what the user
  writes, presses, installs or sees.
- Follow the voice rules of this skill, and wrap at 80 columns.
- Prefix a breaking change with **Breaking:**, as the `disable` entry under
  `[Unreleased]` does.

## Glossary

Use these exact domain terms across all docs.

| Term | Meaning |
| --- | --- |
| task | a runnable command that a source declares |
| source | one family of project files that the picker reads |
| picker | the TUI list of tasks |
| row | one line of the picker, for one task |
| badge | the source file shown at the right of a row |
| widget | the shell function that opens the picker from a key |
| inline | the picker drawn below the prompt |
| fullscreen | the picker that takes the whole terminal |
| rc file | the TOML files the picker reads: the global file and the project file |
| global file | `$XDG_CONFIG_HOME/what-to-run/config.toml`, or `%AppData%\what-to-run\config.toml` on Windows |
| project file | `.what-to-run.toml` in a project |
| theme | a named look from the registry |

## Structure and voice

Read `docs/install.md`, `docs/widget.md` and `docs/config.md` as the structural and stylistic models:

- **Start directly with what the reader needs**: a reader on `docs/*.md` already knows what `wtr` is. Open with the action, the key shortcuts or a summary table rather than re-introducing the tool.
- **Flat `##` sections**: keep one level of `##` headings ordered by how the reader uses the feature, and use tables to compare options (install methods, merge rules, task sources).
- **User-visible behavior only**: document what the user writes, presses or sees. Leave out Go implementation details, internal parser mechanics, release internals (owned by the `release` skill) and prompts that the TUI already shows on screen.

Write in **CEFR B2** American English:

- **One idea per sentence**, most under 25 words, with varied sentence length. Keep a cause with its effect in one sentence ("On Windows, wrap the path in single quotes so backslashes are not parsed as TOML escapes.").
- **Active voice** with the component as subject or "you" as the reader: "The picker reads the rc file." / "Add this line to your rc file."
- **Condition first**: "When multiple entries match the current directory, the deepest path wins."
- **Action in the verb**: "Press `ctrl-o` to open the inline picker." / "To change the shortcut, pass `--key` to `init`."
- **Plain verbs and full forms**: write "shows", "puts", "add", and full forms ("do not", "it is") with periods instead of semicolons.
- **Command name**: write `wtr` in user docs (`bin/wtr` belongs only in `docs/development.md`).

## Completion criteria

Every doc change is done when all four checks hold:

1. **Single owner**: no fact is stated in two files; cross-references link to `[Title](path#anchor)` and every target heading exists.
2. **Ground truth**: every flag, subcommand and key in the touched sections matches `wtr --help` and `cmd/wtr`.
3. **Formatting**: `task check-md` exits 0 with no diff.
4. **Source parity**: `go test ./cmd/wtr` passes.

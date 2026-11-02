# Python task sources

The decision record for the Python task-source family. The research lives in
notes/python-task-sources.md (2026-11-02); this page holds the decisions and
links to the PRs that carry them. Every claim about a tool's behavior cites the
research note, which cites the upstream primary sources.

## Scope decisions (2026-11-02, the maintainer)

- Static data files only. No regex or heuristic parsing of Python task files
  (noxfile.py, tasks.py, dodo.py, pavement.py, SConstruct): their task list is
  knowable only by executing project code. The research note, "Brief exclusions"
  section, confirms this for each file.
- Development tasks only. Application entry points ([project.scripts],
  [project.gui-scripts]) stay out of the picker: they name commands that exist
  only after the package is installed.
- The maintainer delegated every remaining decision to the chief of staff on
  2026-11-02, with review through the draft PRs.

## Design decisions (2026-11-02, the chief of staff)

- One source per tool, following the registry's one-tool-one-ID convention
  (cargo-make and cargo-alias are the precedent): poe, pdm, hatch, taskipy,
  tox, pipenv. Each parser reads only its own tables, so tools that share one
  pyproject.toml need no precedence logic.
- Every Python source sets SearchParents: every runner locates its file by
  walking up from the working directory (research note, "Patterns across the
  tools", and each tool's "Config file location" section).
- The framework passes the winning candidate's file name to the parser, and
  this change lands first: tox needs it to dispatch INI (tox.ini, setup.cfg)
  from TOML (pyproject.toml, tox.toml).
- poe reads TOML candidates only (pyproject.toml, poe_tasks.toml). The
  poe_tasks.yaml, poe_tasks.yml and poe_tasks.json variants wait for a
  follow-up that uses the file-name parameter; [tool.poe] include files are
  skipped, the way Taskfile remote includes are.
- What the tools hide, the picker hides: poe and pdm tasks with an underscore
  prefix, and the pdm shared-options `_` key. pre_* and post_* hooks stay
  listed, because they run standalone and the tools keep them runnable.
- Every task runs through its tool (`poe`, `pdm run`, `hatch run env:script`,
  `task`, `tox run -e`, `pipenv run`), never as the raw command string: the
  tool owns the environment, the templating and the hooks. The description is
  display text and never a promise of the executed command.
- uv needs no source of its own: on 2026-11-02 it has no task file, and uv
  projects run poe or taskipy through `uv run` (research note, "uv"). When uv
  grows native tasks (astral-sh/uv#5903), revisit.

## The PR stack (opened 2026-11-02)

All nine tickets completed on 2026-11-02. Each code PR carries its CHANGELOG
entry, its docs row and its README bullet; the tickets live under
`.scratch/python-task-sources/` because the repo has no configured issue
tracker on that date.

1. PR #88 `docs/python-research-note`: the research note, the notes index and
   this page (commit `a30c0be`).
2. PR #89 `feat/python-parsers-filename`: the file-name parameter for parsers
   (commit `6e200c3`), with the `skip-changelog` label.
3. PR #90 `feat/source-poe`: the poe source (`7bbba5d`).
4. PR #91 `feat/source-pdm`: the pdm source (`c98d4f3`).
5. PR #92 `feat/source-taskipy`: the taskipy source (`0e43b8a`).
6. PR #93 `feat/source-hatch`: the hatch source (`df68117`).
7. PR #94 `feat/source-tox`: the tox source, TOML half (`8741641`).
8. PR #95 `feat/source-tox-ini`: the tox source, INI half (`3e798d8`).
9. PR #96 `feat/source-pipenv`: the pipenv source (`0454871`).

Merges stay manual, in stack order. Since 2026-11-02 the nine code PRs are
also linked as GitHub stack #98 (created with `gh stack link`, no local
tracking state); merging from the stack view retargets the next PR to the
parent base automatically. PR #88 sits outside the stack, based on `main`.

## Lessons from the implementer wave (2026-11-02)

- Repo-local skills were not loadable through the skill tool by the
  implementer subagents: one read `.agents/skills/implement/SKILL.md` from
  disk and followed it, another substituted the `tdd` and `code-review`
  skills. Future spawn instructions should name the SKILL.md path on disk,
  not a skill load.
- The local ticket tracker (`.scratch/`) is gitignored, so worktrees do not
  see the tickets: the implementers read them by absolute path from the main
  checkout. A configured issue tracker would remove that workaround.
- Every implementer flagged the CI `cross` job as unverifiable locally.
  `task all` plus the CI-style `go test -race -shuffle=on ./...` is the local
  ceiling. No finding: that is the design.

## The final review (2026-11-02)

A two-axis review of the whole stack (`main...feat/source-pipenv`, 8 commits)
found no hard standard breaches and a complete spec, with two real bugs in
the tox INI half and one duplicated loop. Outcomes:

- Fix `7594033` on PR #95: the parser now reads the legacy `envlist` key and
  the `[tox:tox]` core section of a setup.cfg, verified against the tox
  4.67.5 source (`env_list` wins when a file holds both spellings; the
  testenv sections in setup.cfg are unprefixed).
- PR #97 `refactor/string-items-helper` (`4590033`): one shared string-items
  join helper for the hatch, pipenv and pdm parsers; poe keeps its variant,
  because its fallback is the opposite contract.

## Lessons from the CI failures (2026-11-02)

Two of the ten stack PRs failed CI on their first runs; both were pushed
green the same day (PR #88 at `92ddf7b`, PR #95 at `54f6082`).

- Hand-written markdown must pass `bin/tools/dprint check` before the push.
  The implementers ran `dprint fmt` in their worktrees, which do not hold the
  docs branch, so the research note reached CI unformatted (the `.agents/`
  tree is excluded from dprint, `notes/` is not).
- In a stacked chain every PR that touches a watched path (`internal`,
  `cmd`, `npm`, `scripts`, `go.mod`, `go.sum`, `.goreleaser.yaml`, the
  release workflow) needs its own new entry under `[Unreleased]`: the
  changelog job counts the entries the PR adds over its base branch, so
  extending the entry of the parent PR does not count. The tox INI half
  learned this the hard way.

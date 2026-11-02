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

1. docs: the research note, notes/index.md and this page.
2. The file-name parameter for parsers.
3. poe, 4. pdm, 5. taskipy, 6. hatch, 7. tox (TOML), 8. tox (INI), 9. pipenv.

Each code PR carries its CHANGELOG entry, its docs row and its README bullet.
The tickets live under `.scratch/python-task-sources/` because the repo has no
configured issue tracker on 2026-11-02.

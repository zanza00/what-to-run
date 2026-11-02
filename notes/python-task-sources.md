# Python task sources: what a Python project uses to declare runnable tasks

Research note, 2026-11-02, verified against primary sources only: PEPs (peps.python.org), each
tool's own documentation site or in-repo `docs/`, the Python Packaging User Guide
(packaging.python.org), and the tools' own repositories and PyPI pages. Every claim below carries
the URL of the page it came from, read on that date. No blog posts, tutorials or third-party
overviews were used. Where a tool's docs are ambiguous, the ambiguity is recorded rather than
guessed. Where no trustworthy source exists (especially for prevalence), this note says so
plainly.

Purpose: what-to-run wants a Python task source. This note maps the territory before any spec is
written: which files declare runnable tasks, the exact sections and entry shapes, the run command
for each tool, where the human description lives, whether the format is static-parseable, and
the edge cases a static parser must beware.

Scope decisions from the maintainer (2026-11-02): the picker reads static data files only. It
will not parse Python code (noxfile.py, tasks.py, dodo.py, pavement.py, SConstruct), and it will
not show application entry points ([project.scripts] and friends). Both families are covered
below as brief exclusions.

## Summary

- The center of gravity is `pyproject.toml`. Five current tools declare runnable dev tasks
  in static tables inside it: `[tool.poe.tasks]` (poethepoet), `[tool.pdm.scripts]` (PDM),
  `[tool.hatch.envs.<env>.scripts]` (Hatch), `[tool.taskipy.tasks]` (taskipy), and
  `[tool.tox]`/`[tool.tox.env.<name>]` (tox 4). Two more use their own static files:
  `tox.ini`/`tox.toml` (tox) and `Pipfile` `[scripts]` (pipenv). Every one of these is
  TOML or INI data, parseable without executing the project's code.
- All five pyproject-based runners locate `pyproject.toml` by walking up from the current
  directory (poe, pdm, hatch, tox, taskipy — sources cited per section below). pipenv does
  the same for `Pipfile`. A picker should match that expectation, because users invoke the
  runners from subdirectories.
- The run command differs per tool: `poe <task>`, `pdm run <script>`, `hatch run
  <env>:<script>`, `task <task>` (or `poetry run task`), `tox run -e <env>`, `pipenv run
  <script>`. In 2026 most of these are usually invoked through `uv run …` or
  `poetry run …`.
- Where the human description lives: a per-task `help` field (poe, pdm, taskipy), a
  per-environment `description` (tox, and hatch — hatch has no per-script help at all), and
  nothing in `Pipfile [scripts]`.
- Every tool except taskipy and tox can hold a Python callable (`script`/`expr` in poe,
  `call` in pdm and pipenv) instead of a shell command; and every tool except taskipy
  supports templating (`${VAR}`, `{env:…}`, `{args}`, `{posargs}`, `{matrix:…}`) that changes
  the command at run time. A static parser can list names and show raw strings, but must
  never promise that the raw string is the executed command.
- uv has no task-running support (open feature request astral-sh/uv#5903, checked
  2026-11-02); the ecosystem answer is poethepoet or taskipy run through `uv run`. Rye, the
  predecessor with `[tool.rye.scripts]`, is dead. No comparable task picker (fzf-make,
  television) supports Python task files today.
- setup.py and setup.cfg are dead as task declarations: PyPA deprecates `python setup.py`
  as a CLI wholesale, and neither file ever had a task table.
- The Python-code runners (nox, invoke, doit, paver, SCons, and pytask) define tasks as
  Python functions/decorators — excluded by the maintainer's decision, and their static
  config tables (none for nox, `[tool.doit]`, `invoke.yaml`) hold options, not tasks.
  Entry points (`[project.scripts]`, `[project.gui-scripts]`, `[project.entry-points]`,
  `[tool.poetry.scripts]`) are application commands installed with the package — excluded.
- Maintenance status on 2026-11-02: tox 4.67.5 (2026-10-31), pipenv 2026.10.5 (2026-11-01),
  hatch 1.18.2 (2026-10-28), pdm 2.29.5 (2026-10-30), poethepoet 0.49.0 (2026-10-15) — all
  actively released within the three weeks before this note. taskipy 1.14.1 (2024-11-26)
  has been dormant for two years. setuptools 84.2.1 (2026-10-20) is alive but not a task
  runner.
- Prevalence: no trustworthy source measures which of these files or sections a typical
  Python repo has. The Python Developers Survey 2024 measures tool adoption (pyproject.toml
  used by 36% for dependency info; Poetry 20%, uv 12%, Pipenv 8%, PDM 5%) but never asks
  about task runners. Nothing better exists; do not guess.

## [tool.poe.tasks] — poethepoet

Poe the Poet is a task runner that reads tasks from `pyproject.toml` and works with Poetry and
uv. Docs: <https://poethepoet.natn.io/> (source: the `docs/` directory of
<https://github.com/nat-n/poethepoet>, `main` branch, read 2026-11-02). Latest release 0.49.0,
uploaded 2026-10-15 (<https://pypi.org/project/poethepoet/>).

### Section names and task entry shape

Tasks are members of `[tool.poe.tasks]` in `pyproject.toml`
(<https://github.com/nat-n/poethepoet/blob/main/docs/guides/running_guide.rst>). One entry can be
a string, an array, or a table:

- A **string** is a `cmd` task by default: `test = "pytest"`
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/index.rst>).
- An **array** is a `sequence` task whose items are `ref` references to other tasks by default:
  `build = ["test", "_build"]` (same page).
- A **table** is typed by whichever task-type key it contains. The eight types
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/index.rst>):
  - execution: `cmd`, `script` (a Python callable reference), `shell` (a shell script), `expr`
    (a Python expression to evaluate);
  - orchestration: `sequence`, `parallel`, `switch`, `ref` (alias of another task).
- Task names starting with `_` are hidden from the list and cannot be run directly
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/index.rst>).

Standard options on any task (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>):
`help`, `args`, `env`, `envfile`, `cwd`, `deps`, `uses`, `uses_env`, `capture_stdout`,
`executor`, `verbosity`, `use_exec`. Type-specific options exist per type (for example `cmd`
adds `ignore_fail`, `empty_glob`; `shell` adds `interpreter`; `expr` adds `imports`, `assert`;
`sequence` and `parallel` add `default_item_type`; `switch` requires `control`).

Minimal realistic example:

```toml
[tool.poe.tasks.test]
help = "Run the test suite"
cmd  = "pytest --cov=src"

[tool.poe.tasks.serve]
help   = "Run the app in debug mode"
script = "my_app.service:run(debug=True)"
```

### The run command

`poe <task>` is the CLI
(<https://github.com/nat-n/poethepoet/blob/main/docs/guides/running_guide.rst>). Free
arguments pass through to the task: `poe test -v tests/favorite_test.py`. Other invocation
routes: `python -m poethepoet <task>`, `poetry poe <task>` (as a Poetry plugin), `poetry run
poe <task>` (as a Poetry dev dependency) — all documented on the same page. Global `-C,
--directory PATH` tells poe where to find the config
(<https://github.com/nat-n/poethepoet/blob/main/docs/guides/help_guide.rst> shows the full
`--help` output).

### Where the human description lives

The `help` key on the task table: "Help text to be displayed next to the task name in the
documentation when poe is run without specifying a task"
(<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>). Arguments may carry
their own `help` text
(<https://github.com/nat-n/poethepoet/blob/main/docs/guides/args_guide.rst>).

### Static-parseable, and which parts are not

`[tool.poe.tasks]` is data: a static parser can list task names, types, `help` strings, and
`cmd`/`shell` content. The parts that are not plain data:

- `script` and `expr` content is Python code (a callable reference plus a Python argument
  expression), not a shell command
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/script.rst>,
  <https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/expr.rst>).
- `cmd` strings can carry `${VAR}` parameter expansion with `:-`/`:+` operators, glob
  expansion, comments, and multi-line content — the executed command differs from the literal
  string (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/cmd.rst>).
- The type of a string or array entry depends on the global `default_task_type`,
  `default_array_task_type` and `default_array_item_task_type` options under `[tool.poe]`,
  which can each change what a bare string means
  (<https://github.com/nat-n/poethepoet/blob/main/docs/global_options.rst>).

### Config file location and separate task files

Poe looks for `pyproject.toml`, `poe_tasks.toml`, `poe_tasks.yaml`, `poe_tasks.yml`,
`poe_tasks.json` in the current directory, then in parent directories in ascending order —
`PoeConfig._config_filenames` and `PoeConfigFile.find_config_files(..., search_parent=True)`
(<https://github.com/nat-n/poethepoet/blob/main/poethepoet/config/config.py>,
<https://github.com/nat-n/poethepoet/blob/main/poethepoet/config/file.py>, read 2026-11-02;
the docs site does not spell this out, the repository source is the authority here).

Tasks can also live in other files: the `[tool.poe]` `include` option names toml, yaml or json
files with "the same structure for tasks as used in pyproject.toml"; included files are not
required to namespace under `tool.poe`; missing files are ignored; includes recurse by default
and accept a `cwd` option
(<https://github.com/nat-n/poethepoet/blob/main/docs/guides/include_guide.rst>). There is also
`include_script` to generate tasks from a Python function
(<https://github.com/nat-n/poethepoet/blob/main/docs/global_options.rst>) — not static data.

### Edge cases for a static parser

- `args` with `required = true` make a task unrunnable as-is; `positional = true` args are
  filled positionally, so a bare `poe <task>` errors
  (<https://github.com/nat-n/poethepoet/blob/main/docs/guides/args_guide.rst>).
- `sequence`/`parallel` arrays hold strings (references, not tasks), inline tables (real
  sub-tasks that are not listable separately), and nested arrays (a nested `parallel`);
  `default_item_type` changes what strings mean
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/sequence.rst>,
  <https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/parallel.rst>).
- `switch` tasks choose a sub-task at runtime from a control task's output, typically
  `sys.platform` — the platform the picker runs on may not match
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/switch.rst>).
- `ref` tasks alias another task with extra `env` or arguments; running the ref is not the
  same as running the target
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/ref.rst>).
- `cwd` on a task is resolved relative to the config file's parent directory and supports
  `${VAR_NAME}` expansion
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>).
- Environment variables in `env` values, `cmd` content and `args` defaults all use `${VAR}`
  templating with `:-`/`:+` operators
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/task_types/cmd.rst>,
  <https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>).
- The executor (`uv`, `poetry`, `venv`, `simple`) decides which environment runs the command —
  a task's `cmd` cannot be run raw without that context
  (<https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>, executor options).
- `deps` runs other tasks first; `uses` captures other tasks' stdout into variables
  (<https://github.com/nat-n/poethepoet/blob/main/docs/guides/composition_guide.rst>).

## [tool.pdm.scripts] — PDM

PDM is a package manager and "like `npm run`" task runner
(<https://pdm-project.org/latest/usage/scripts/>; source of the docs:
<https://github.com/pdm-project/pdm/blob/main/docs/usage/scripts.md>, `main` branch, read
2026-11-02). Latest release 2.29.5, uploaded 2026-10-30 (<https://pypi.org/project/pdm/>).

### Section names and task entry shape

User scripts live in `[tool.pdm.scripts]` in `pyproject.toml`. PDM supports four script types
(all from docs/usage/scripts.md):

- `cmd`: a plain string, or a table `{cmd = "flask run -p 54321"}`; `cmd` also accepts an
  **array** of strings so comments can sit between arguments.
- `shell`: `{shell = "cat error.log|grep CRITICAL > critical.log"}` — run via
  `subprocess.Popen(shell=True)`.
- `call`: `{call = "foo_package.bar_module:main"}` — a Python function reference, optionally
  with literal Python arguments: `{call = "foo_package.bar_module:main('dev')"}`.
- `composite`: `{composite = ["lint", "test"]}` — runs other named scripts (added in 2.13.0);
  `keep_going = true` continues after a failure, and items may carry arguments
  (`"test -v tests/"`).

Minimal realistic example:

```toml
[tool.pdm.scripts]
start = {cmd = "flask run -p 54321", help = "Start the dev server"}
test  = "pytest"
```

Script options: `env` (map), `env_file` (string, or an `override` key), `working_dir`
(resolved against the project root), `site_packages`. A special `_` key holds options shared
by all scripts (`_.env_file = ".env"`). Placeholders: `{args}` and `{args:--default --value}`,
`{pdm}` (path to the pdm executable), `{run_cwd}`.

### The run command

`pdm run <script_name>`; trailing arguments are appended to the command. As a shortcut a
non-conflicting script is also a root command: `pdm start` works, but `pdm install` runs the
builtin, not a script of that name. `pdm run --list/-l` shows the table of Name, Type,
Description. `pdm run --skip=:all` and friends skip pre/post hooks.

### Where the human description lives

The `help` option on the script table, shown in the `Description` column of `pdm run --list`
(docs/usage/scripts.md, "Show the List of Scripts"). Script names starting with `_` are
internal and hidden from the listing.

### Static-parseable, and which parts are not

`[tool.pdm.scripts]` is static data. Not static: `call` scripts reference Python functions;
`shell` scripts are shell-dependent; `composite` items are names of other scripts, not
commands; `${VAR}` substitution happens in all script types; `{args}` changes how arguments
are appended.

### Config file location and separate task files

PDM locates the project by searching `pyproject.toml` in the current directory and then
recursively upward: `find_project_root` — "Recursively find a `pyproject.toml` at given path or
current working directory"
(<https://github.com/pdm-project/pdm/blob/main/src/pdm/utils.py>, read 2026-11-02). `-p
/--project` gives an explicit root. Tasks live only in `pyproject.toml`; no separate include
file for scripts is documented.

### Edge cases for a static parser

- `pre_<name>` and `post_<name>` scripts run automatically around `<name>`: "a failure will
  cancel the subsequent execution" — a picker that lists them as ordinary tasks shows
  plumbing npm users would recognize.
- Fixed hook names (`post_init`, `pre_install`, `post_install`, `pre_lock`, `post_lock`,
  `pre_build`, `post_build`, `pre_publish`, `post_publish`, `pre_script`, `post_script`,
  `pre_run`, `post_run`) are triggered by pdm commands, not the user; the docs warn to avoid
  reserved names.
- `composite` items with arguments (`"test -v tests/"`) and `{args}` placeholders mean a
  listed task cannot always be run bare.
- `env`/`env_file` values support `${VAR}` and `${VAR:-default}` references, so commands are
  not literal.
- The shared `_` key looks like a script named `_` but is options; a naive parser would list
  it or misread its shape.
- `pdm <script>` as a root command only exists when the name does not collide with a builtin —
  the safe universal prefix is `pdm run <script>`.
- `cmd` given as an array needs joining before display; a parser that assumes strings only
  drops those tasks.

## [tool.hatch.envs.*] scripts — Hatch

Hatch is a project manager whose "environments" are isolated workspaces, and each environment
can define named scripts. Docs: <https://hatch.pypa.io/latest/> (source: the `docs/` directory
of <https://github.com/pypa/hatch>, `master` branch, read 2026-11-02). Latest release 1.18.2,
uploaded 2026-10-28 (<https://pypi.org/project/hatch/>).

### Section names and task entry shape

Environments are tables under `[tool.hatch.envs.<ENV_NAME>]`
(<https://github.com/pypa/hatch/blob/master/docs/config/environment/overview.md>). Scripts are
a table under each environment:

```toml
[tool.hatch.envs.test]
dependencies = ["pytest", "pytest-cov"]

[tool.hatch.envs.test.scripts]
run-coverage = "pytest --cov-config=pyproject.toml --cov=pkg --cov=tests"
run = "run-coverage --no-cov"
```

A script is a **string or an array of strings**. A leading word that names another script of
the same environment expands to that script: `run` above becomes
`pytest --cov-config=pyproject.toml --cov=pkg --cov=tests --no-cov`. Array items run
sequentially; a command starting with `-` has its exit code ignored (make-style). Environment
options that matter to a picker: `template` (inheritance from another env, default `default`),
`detached`, `skip-install`, `dependencies`, `features`, `dependency-groups`, `env-vars`,
`env-include`/`env-exclude`, `platforms` (`linux`, `windows`, `macos`), `python`, `description`,
`type`, `pre-install-commands`/`post-install-commands`.

### The run command

`hatch run <ENV_NAME>:<script>` — e.g. `hatch run docs:serve`; without the `<ENV_NAME>:` prefix
the `default` environment is used, and `-e/--env` or `HATCH_ENV` select it too
(<https://github.com/pypa/hatch/blob/master/docs/environment.md>, "Selection"). Extra CLI
arguments reach the script through the `{args}` context field.

### Where the human description lives

There is **no per-script help field**. The only human description is the environment-level
`description` option, "purely informational and is displayed in the output of the `env show`
command" (<https://github.com/pypa/hatch/blob/master/docs/config/environment/overview.md>).
A picker gets script names and commands from the TOML, but a description only per environment.

### Static-parseable, and which parts are not

The `[tool.hatch.envs.*.scripts]` tables are static data. Not static: context formatting inside
script text (`{env:FOO:DEFAULT}`, `{root}`, `{matrix:version:v1.0.0}`, `{args}`,
`{verbosity:flag:-1}`, and nesting like `{env:FOO:{env:BAR:{home}}}` —
<https://github.com/pypa/hatch/blob/master/docs/config/context.md>); scripts referencing other
scripts by name; matrices and overrides (below), which multiply and mutate environments.

### Config file location and separate task files

"By default, Hatch will look for a `pyproject.toml` file in the current working directory and
any parent directories. The directory storing the first found file will be the project root"
(<https://github.com/pypa/hatch/blob/master/docs/config/hatch.md>). Project task config lives
only in `pyproject.toml`; the user-level `config.toml` stores Hatch's own settings and a
project-mode table, not project tasks. `-p, --project` selects a project by the names
configured in that user-level table, not by path
(<https://github.com/pypa/hatch/blob/master/src/hatch/cli/__init__.py>, root options).

### Edge cases for a static parser

- **Matrices**: `[[tool.hatch.envs.<ENV>.matrix]]` generates the product of each variable
  combination as its own environment, named `<ENV>.<value>-<value>` (a `python`/`py` variable
  ranks first and gets a `py` prefix; `matrix-name-format` can change the join). A static
  parser that lists only the root env shows `test` while `hatch env show` shows
  `test.py3.10-42` … (<https://github.com/pypa/hatch/blob/master/docs/config/environment/advanced.md>).
- **Overrides**: the `overrides` table mutates options by `platform`, `env`, `matrix`, or
  `name` conditions — including `scripts` itself (`platform.windows.scripts = ['run=pytest
  -m "not io_uring"']`). The effective script for a given machine is decided at runtime, not
  in the file.
- **Environment inheritance**: every env inherits from `template` (default `default`), so
  scripts visible in `default` are runnable in children; `extra-scripts` adds without
  overriding. A parser reading one env table misses inherited scripts.
- **Platform gating**: `platforms` limits where an env can run; a picker on the wrong OS would
  list tasks that fail.
- Environments named with the `hatch-` prefix are internal (e.g. hatch's own testing).
- A script that is an array is a pipeline of commands, not one command.
- Every script is scoped to an environment: the picker must run `hatch run <env>:<script>`,
  and the environment is created/installed on first run, so "running the task" may involve
  dependency installation.

## [tool.taskipy.tasks] — taskipy

taskipy is "the complementary task runner for python", "heavily inspired by npm's run script
command". Docs: the README of <https://github.com/taskipy/taskipy> (`master` branch, read
2026-11-02; there is no separate docs site). Latest release 1.14.1, uploaded 2024-11-26
(<https://pypi.org/project/taskipy/>); the last commit on `master` is that same release
(<https://github.com/taskipy/taskipy/commits/master>), so the project has been dormant for
about two years as of 2026-11-02.

### Section names and task entry shape

Tasks live in `[tool.taskipy.tasks]` in `pyproject.toml`. A task is either a plain string
(command) or an inline table with the fields `cmd`, `help`, `cwd`, `use_vars`
(<https://github.com/taskipy/taskipy/blob/master/taskipy/task.py>, `__extract_*` methods):

```toml
[tool.taskipy.tasks]
test = { cmd = "python -m unittest tests/test_*.py", help = "runs all unit tests" }
lint = "pylint tests taskipy"
```

Companion tables: `[tool.taskipy.variables]` (a string, or a table `{ var, recursive }`) and
`[tool.taskipy.settings]` (keys `use_vars`, `cwd`, `runner`) — README "Using Variables" and
"Working directory", and `PyProject.settings`/`PyProject.runner` in
<https://github.com/taskipy/taskipy/blob/master/taskipy/pyproject.py>.

### The run command

`task <name>` without Poetry (from the project directory), or `poetry run task <name>` with
it (README "Running Tasks"). `task --list` prints the task names with their commands or
`help` text. Extra arguments append to the command.

### Where the human description lives

The `help` key of the task table; shown next to the task name by `task --list` (README
"Running Tasks": "you will see the description of the task by the side of the task's name").

### Static-parseable, and which parts are not

`[tool.taskipy.tasks]` is static data — the simplest of the pyproject runners: strings or
four-field tables. Not literal: `{var}` substitution when `use_vars` is on (Python
`string.format` syntax, so `{}`-sequences in commands may or may not be intentional), and
commands run through `shell=True`
(<https://github.com/taskipy/taskipy/blob/master/taskipy/task_runner.py>).

### Config file location and separate task files

taskipy looks for `pyproject.toml` in the current directory and then in every parent
directory (`PyProject.__find_pyproject_path` with `candidate_dirs` yielding the base dir and
all parents, <https://github.com/taskipy/taskipy/blob/master/taskipy/pyproject.py>). Tasks
live only in `pyproject.toml`; no separate task file is documented.

### Edge cases for a static parser

- `pre_<name>` and `post_<name>` tasks are hooks that run automatically around `<name>`
  (README "Composing Tasks"); listing them as ordinary tasks shows plumbing.
- The `runner` setting (e.g. `poetry run`) is prepended to every command at execution
  (`task_runner.py`), so the raw `cmd` is not the full executed command.
- `use_vars` (per task or global in settings) turns `{name}` sequences into variable
  references; with it off, the same text is literal.
- `cwd` on a task or in settings changes the working directory to a path relative to the
  `pyproject.toml` directory.
- Composite tasks are plain shell strings that call `task <other>`, so a `lint` task may
  just dispatch to other tasks.

## [tool.tox.*] in pyproject.toml, and tox.ini — tox 4

tox is "an environment orchestrator. Use it to define how to set up and execute various tools
on your projects" (<https://github.com/tox-dev/tox/blob/main/docs/explanation.rst>). A tox
environment is an isolated workspace — "by default virtualenv is used" — where `deps` are
installed, the project may be built and installed, and `commands` run
(same page, system overview). Latest release 4.67.5, uploaded 2026-10-31
(<https://pypi.org/project/tox/>); actively maintained (changelog entries in the repo are
dated weeks before, <https://github.com/tox-dev/tox/blob/main/docs/changelog.rst>).

### Config files: five locations, INI deprecated

Out of the box tox supports five configuration locations, in this priority order
(<https://github.com/tox-dev/tox/blob/main/docs/reference/config.rst>, "Configuration"):

1. `tox.ini` (INI, **deprecated**),
2. `setup.cfg` (INI, **deprecated**),
3. `pyproject.toml` under the `tool.tox` table (native TOML),
4. `pyproject.toml` with `tool.tox.legacy_tox_ini` (INI text inside TOML, deprecated),
5. `tox.toml` (TOML).

"Historically, the INI format was created first, and TOML was added in 2024. **The INI format
is deprecated** — it keeps working for existing projects, but it is frozen: no new features
will be added to it, and it may be removed in a future major release" (same page). Native
TOML support was added in tox 4.21.0, released 2024-09-30
(<https://github.com/tox-dev/tox/blob/main/docs/changelog.rst>, `release-4-21-0`). The
getting-started guide says: "tox supports two configuration formats: TOML and INI. **Use
TOML**" (<https://github.com/tox-dev/tox/blob/main/docs/tutorial/getting-started.rst>).

Section names:

- INI (`tox.ini`): `[tox]` for core settings, `[testenv]` as the base for run environments,
  `[testenv:{env_name}]` per environment, `[pkgenv]` as base for package environments
  (<https://github.com/tox-dev/tox/blob/main/docs/reference/config.rst>, "tox.ini").
- Native TOML in `pyproject.toml`: `[tool.tox]` core, `[tool.tox.env.{env_name}]` per
  environment, `[tool.tox.env_run_base]` and `[tool.tox.env_pkg_base]` as the two base tables
  (same page, "pyproject.toml - native"). `tox.toml` is the same without the `tool.tox`
  prefix.

Minimal realistic example, INI:

```ini
[tox]
env_list = 3.13, lint

[testenv]
description = run the test suite with pytest
deps = pytest>=8
commands = pytest {posargs:tests}

[testenv:lint]
description = run linters
skip_install = true
deps = ruff
commands = ruff check {posargs:.}
```

and the same natively in `pyproject.toml` (getting-started, TOML tab):

```toml
[tool.tox]
env_list = ["3.13", "lint"]

[tool.tox.env_run_base]
description = "run the test suite with pytest"
deps = ["pytest>=8"]
commands = [["pytest", { replace = "posargs", default = ["tests"], extend = true }]]

[tool.tox.env.lint]
description = "run linters"
skip_install = true
deps = ["ruff"]
commands = [["ruff", "check", { replace = "posargs", default = ["."], extend = true }]]
```

### The run command

`tox run -e <env>` — e.g. `tox run -e lint`, several at once `tox run -e 3.13,lint`, and
extra args after `--`: `tox run -e 3.13 -- -v` (getting-started). Bare `tox` runs `env_list`.
`tox list` prints every environment with its description:

```
default environments:
3.13 -> run the test suite with pytest
lint -> run linters
```

(getting-started, "Listing available environments").

### Where the human description lives

The per-environment `description` key: "A short description of the environment, this will be
used to explain the environment to the user upon listing environments"
(<https://github.com/tox-dev/tox/blob/main/docs/reference/config.rst>, `description` option,
`[testenv]`/`env_run_base` base options).

### Which sections declare commands

Per-environment options that matter to a picker (all in the `tox environment` section of
config.rst): `commands` — "The commands to be called for testing"; `commands_pre` and
`commands_post` around them; `deps` (packages installed into the env); `set_env` ("A
dictionary of environment variables to set when running commands in the tox environment",
plus an env-file reference); `description`; `depends`; `change_dir`; `base_python`;
`skip_install`. In INI each `commands` line is one command (multi-line with `\` continuation);
a leading `-` ignores the exit code, make-style; since tox 4.4.0 lines are split with POSIX
`shlex` rules on all platforms.

### Static-parseable, and which parts are not

Both formats are static data: environment names are the `[testenv:<name>]` section names (INI)
or `tool.tox.env.<name>` tables (TOML), and each carries `commands`, `deps`, `set_env` and
`description` as strings/arrays/tables. Not static:

- **Substitutions** in almost every value: `{...}` references to config keys, `{env:KEY}` host
  environment variables, `{posargs}` CLI arguments, `{glob:PATTERN}`, `{tox_root}`,
  `{env_name}` etc. (config.rst, "Substitutions" and "Environment variable substitutions").
- **Generative environment lists** (INI): `env_list = py3{9-11}-django{41,40}-{sqlite,mysql}`
  is a product matrix; INI also has generative *section* names like
  `[testenv:py3{10,11}-{linux,windows}]`. TOML has no generative sections and uses
  `env_base` templates and `product` ranges instead (config.rst, "Generative environment
  list", `env_base templates`, "Format comparison": "The only INI feature without a TOML
  equivalent is generative section names").
- **Conditional settings**: factor-conditional values (`py311-mysql: PyMySQL` in INI deps;
  `{ replace = "if", condition = "factor.django41", then = [...] }` in TOML) pick different
  values per generated environment (config.rst, "Conditional settings").

### Config file location and search

tox looks in the current directory and then walks parent directories; in each directory it
tries the five sources in the priority order above; a `pyproject.toml` without a loadable
`[tool.tox]` table is skipped (`src/tox/config/source/discover.py`, `_locate_source`:
`for base in chain([folder], folder.parents)`,
<https://github.com/tox-dev/tox/blob/main/src/tox/config/source/discover.py>, read
2026-11-02). `-c/--conf` names a file or folder explicitly, `--root` sets the project root
(<https://github.com/tox-dev/tox/blob/main/docs/man/tox.1.rst>). Ambiguity recorded: the
repo's man page `FILES` section lists the search order as `tox.toml`, `tox.ini`,
`pyproject.toml`, `setup.cfg` — a different order than `config.rst` and the source code
(same URL); the source code's order is the one the released binary implements.

### Edge cases for a static parser

- `[testenv]` (INI) and `env_run_base` (TOML) are base tables, not environments: values fall
  back to them, so an env's effective `commands` may live outside its own section.
- Environment names are often Python versions (`3.13`) or factor combinations, not task
  names: a picker that lists them as "tasks" shows `3.13 -> run the test suite with pytest`
  where a user thinks "run tests".
- Generative sections and `env_list` products expand at load time; the file does not name the
  environments that actually run.
- `TOX_SKIP_ENV` filters environments at run time via a regex (config.rst, `env_list`).
- `posargs` means the task is meant to be run with extra arguments; a bare run uses the
  default after `:`.
- Every run is a full environment lifecycle: create venv, install deps, build/install the
  project, then run commands. "Running the task" from a picker inherits all of that.
- The `-e` values can also be `.pkg`-style internal package environments; `pkgenv`/
  `env_pkg_base` are packaging bases, not user tasks (config.rst, "tox.ini"/"pyproject.toml -
  native").

## Pipfile [scripts] — pipenv

pipenv is a packaging tool that pairs `Pipfile` (TOML, a proposed replacement for
requirements.txt) with `Pipfile.lock`. The Pipfile format spec repo does not define
`[scripts]` at all — its README describes sources, requires, packages and dev-packages only
(<https://github.com/pypa/Pipfile/blob/main/README.rst>, read 2026-11-02): `[scripts]` is a
pipenv feature. pipenv's own docs: <https://pipenv.pypa.io/> (source:
<https://github.com/pypa/pipenv/tree/main/docs>). Latest release 2026.10.5, uploaded
2026-11-01 (<https://pypi.org/project/pipenv/>).

### Section names and task entry shape

Scripts live in the `[scripts]` section of `Pipfile`. The implementation
(`pipenv/cmdparse.py`, `Script.parse`, read 2026-11-02,
<https://github.com/pypa/pipenv/blob/main/pipenv/cmdparse.py>) accepts three entry shapes:

- **string** — one command, shell-split in POSIX mode on every platform:
  `test = "pytest -x"`;
- **array of strings** — a sequence of commands run in order, stopping at the first
  non-zero exit code; extra CLI arguments are appended to the **last** command
  (`lint = ["ruff check .", "ruff format --check ."]`) — added 2026-03-18
  (<https://github.com/pypa/pipenv/commit/886d676e>, "Add sequential script execution via
  TOML array syntax in [scripts]");
- **inline table with a single `call` key** — a Python callable reference:
  `check = {call = "mypackage.checks:run()"}`, converted to
  `python -c "import mypackage.checks as _m; _m.run()"`. `script_types = ["call"]` is the
  only accepted key; a table with any other key raises
  "Not an accepted script callable".

Minimal realistic example:

```toml
[scripts]
start = "python app.py"
test  = "pytest"
lint  = ["ruff check .", "ruff format --check ."]
```

**Docs/code discrepancy, recorded, not resolved**: pipenv's current docs pages
(<https://github.com/pypa/pipenv/blob/main/docs/scripts.md>,
<https://github.com/pypa/pipenv/blob/main/docs/advanced.md>, "Custom Scripts") present an
"extended syntax" `{cmd = "python app.py"}`, which the implementation quoted above rejects.
Those doc pages were rewritten by an AI prompt in 2025 (commit f996fd63, 2025-04-22, "Rewrite
docs w/ Cline + Claude AI prompt"; docs/scripts.md itself landed in 2025-12-10, commit
9e471910), so the `{cmd = ...}` shape may be documentation invention rather than a supported
feature; a static parser should treat `cmd` tables as "unknown validity" and fall back to the
raw value.

### The run command

`pipenv run <script_name> [args...]` — arguments are appended to the command (for arrays, to
the last command) (<https://github.com/pypa/pipenv/blob/main/docs/advanced.md>, "Running
Scripts"). `pipenv scripts` lists the scripts defined in the Pipfile: a two-column
"Command / Script" table built straight from the `[scripts]` section
(<https://github.com/pypa/pipenv/blob/main/docs/cli.md>, `scripts`;
implementation `cmd_scripts` in
<https://github.com/pypa/pipenv/blob/main/pipenv/cli/command.py>).

### Where the human description lives

Nowhere. The `[scripts]` section carries no description field; `pipenv scripts` shows only
name and command. A picker must synthesize the description or show the command.

### Static-parseable, and which parts are not

`[scripts]` is static data: name → string / array / `call` table. Not literal: `call` entries
are Python function references; commands run in the project's virtualenv (the venv's
`bin/`/`Scripts` dir is prepended to `PATH` and `VIRTUAL_ENV` is set —
<https://github.com/pypa/pipenv/blob/main/pipenv/routines/shell.py>, `do_run`); inline
`FOO=bar` prefixes are extracted into environment variables before execution (same file, and
`Script.with_extracted_env_vars`).

### Config file location and separate task files

The file is `Pipfile` (TOML), plus the generated `Pipfile.lock`. pipenv honors
`PIPENV_PIPFILE`, otherwise it walks parent directories from the cwd to find the Pipfile
(`find_pipfile`, "Returns the path of a Pipfile in parent directories",
<https://github.com/pypa/pipenv/blob/main/pipenv/utils/pipfile.py>; also
<https://pipenv.pypa.io/en/stable/>). No separate task file.

### Edge cases for a static parser

- Scripts are venv-scoped: the same name may exist on `PATH` outside the venv, so a picker
  must run `pipenv run <name>`, never the raw command.
- An array entry is a `&&`-chain, not one command.
- `call` entries need the project's package importable; they are not shell commands.
- Pipenv also injects `PIPENV_ACTIVE=1` and `PIPENV_PROJECT_DIR` (project root) into script
  environments (`do_run`), which scripts may rely on.
- The docs' `{cmd = ...}` shape (see the discrepancy above) — handle defensively.

## setup.py and setup.cfg

Answer up front: in 2026 these files are effectively dead as a place to declare runnable dev
tasks. They never had a task table; their "tasks" were the `python setup.py <command>` CLI,
which PyPA has deprecated wholesale.

- The Python Packaging User Guide's dedicated page "Is setup.py deprecated?" (read 2026-11-02,
  <https://packaging.python.org/en/latest/discussions/setup-py-deprecated/>): "No, setup.py
  and Setuptools are not deprecated. Setuptools is perfectly usable as a build backend for
  packaging Python projects. And setup.py is a valid configuration file for Setuptools that
  happens to be written in Python… However, `python setup.py` and the use of setup.py as a
  command line tool are deprecated." It lists the commands that "MUST NOT be run anymore":
  `python setup.py install`, `develop`, `sdist`, `bdist_wheel`, with replacements
  (`python -m pip install .`, `python -m pip install --editable .`, `python -m build`).
  `python setup.py test` gets: "The recommendation is to use a test runner such as pytest."
- Setuptools' own quickstart (read 2026-11-02,
  <https://setuptools.pypa.io/en/latest/userguide/quickstart.html>): "Every python package
  must provide a pyproject.toml and specify the backend"; on setup.py: "running this file as
  a script (e.g. `python setup.py sdist`) is strongly discouraged, and the majority of the
  command line interfaces are (or will be) deprecated… We also recommend users to expose as
  much as possible configuration in a more declarative way via the pyproject.toml or
  setup.cfg, and keep the setup.py minimal with only the dynamic parts (or even omit it
  completely if applicable)."
- setup.cfg is configuration for packaging, not tasks: it "contains option defaults for
  setup.py commands" (<https://packaging.python.org/en/latest/guides/distributing-packages-using-setuptools/>,
  "setup.cfg") and "[metadata]…[options]" tables
  (<https://setuptools.pypa.io/en/latest/userguide/declarative_config.html>). There is no
  section that declares a runnable task.
- Custom setuptools/distuptools *commands* written in setup.py would be Python code —
  excluded by the maintainer's decision anyway, and the list is knowable only by executing
  it.
- setuptools itself remains actively maintained (84.2.1, uploaded 2026-10-20,
  <https://pypi.org/project/setuptools/>); what is dead is the setup.py/setup.cfg *task*
  role, not the tool. Note also that setup.cfg survives as a config location for unrelated
  tools (tox lists it as a legacy location, deprecated — see the tox section above), which
  is configuration, not task declaration.

A Python source for what-to-run should not read setup.py or setup.cfg for tasks.

## uv and other current runners

### uv (Astral)

uv has **no task-running support** as of 2026-11-02:

- `uv run <command>` "runs commands in the project environment" — an arbitrary command
  wrapper, not named tasks
  (<https://docs.astral.sh/uv/concepts/projects/run/>, source:
  <https://github.com/astral-sh/uv/blob/main/docs/concepts/projects/run.md>). uv's "scripts"
  feature is PEP 723 inline metadata for single-file scripts
  (<https://docs.astral.sh/uv/guides/scripts/>), per-script dependencies, not a project task
  list.
- The feature request "Using `uv run` as a task runner"
  (<https://github.com/astral-sh/uv/issues/5903>) is still **open** (1040+ reactions, last
  updated 2026-10-02, checked 2026-11-02). uv's latest release is 0.13.6 (2026-10-26,
  <https://github.com/astral-sh/uv/blob/main/CHANGELOG.md>), and no "task" feature appears
  in the changelog entries to that date.
- The issue itself points to the ecosystem answer: run a pyproject-based runner *through* uv
  — `uv add --dev poethepoet` then `uv run poe <task>`, or `uv run task <task>` with
  taskipy. Poe documents its uv integration first-class: `uv tool install poethepoet`,
  `uv add --dev poethepoet` + `uv run poe`, the alias `alias poe="uv run poe"`, plus a `uv`
  task executor
  (<https://github.com/nat-n/poethepoet/blob/main/docs/installation.rst>,
  <https://github.com/nat-n/poethepoet/blob/main/docs/tasks/options.rst>, executor options).

A picker therefore cannot show "uv tasks" — there is no such file — but a uv project is
very likely to host `[tool.poe.tasks]` or `[tool.taskipy.tasks]`, run through `uv run`.

### Rye — the predecessor with tasks, now dead

Rye had exactly this feature: `[tool.rye.scripts]` in pyproject.toml with `cmd` (string or
array), `env`, `env-file`, `chain`, `call`
(<https://github.com/astral-sh/rye/blob/main/docs/guide/pyproject.md>, "tool.rye.scripts"),
run with `rye run <name>`
(<https://github.com/astral-sh/rye/blob/main/docs/guide/commands/run.md>). But Rye's README
says: "Rye is no longer developed. We encourage all users to use uv… While Rye will continue
to be available, no further updates are planned, including security updates"
(<https://github.com/astral-sh/rye/blob/main/README.md>). Not a 2026 target; noted because
its scripts shape is the model the uv issue #5903 asks for.

### pytask — current, but tasks are Python functions

pytask (0.6.0, uploaded 2026-05-01, <https://pypi.org/project/pytask/>) is an actively
maintained pytest-style runner: "A task is a function that is detected if the module and the
function name are prefixed with `task_`"
(<https://github.com/pytask-dev/pytask/blob/main/README.md>). Configuration can live in
pyproject.toml (`[tool.pytask.ini_options]`,
<https://pytask-dev.readthedocs.io/en/stable/how_to_guides/configuration.html>), but that
table holds options, not tasks; the task list is knowable only by importing the project's
Python files. It belongs with the Python-code exclusions below.

### Comparable task-discovery/picker tools: none found for Python

- **fzf-make** — a TUI fuzzy picker for project tasks — supports make, npm, pnpm, yarn,
  just, and go-task. Python task files are absent from its support list
  (<https://github.com/kyu08/fzf-make/blob/main/README.md>: "Currently supporting make, npm,
  pnpm, yarn, just, task", read 2026-11-02).
- **television** — a fuzzy finder with channels; its default channel is files
  (`DEFAULT_PROTOTYPE_NAME = "files"`,
  <https://github.com/alexpasmantier/television/blob/main/television/channels/prototypes.rs>);
  no built-in project-task channel for any ecosystem, and none for Python.
- No comparable task-discovery tool with first-class Python (pyproject/poe/taskipy) support
  turned up in official docs searched on 2026-11-02; what-to-run's planned Python source
  would be first in this family. I state this as "not found after checking", not as proof
  of absence.

## Practical prevalence in 2026

Straight answer first: **no trustworthy source measures which task sections Python projects
actually use**. Neither the Python Developers Survey nor any PyPA publication asks about
`[tool.poe.tasks]`, `[tool.taskipy.tasks]`, `[tool.pdm.scripts]` or similar; I found no
census of pyproject.toml sections. I will not invent numbers for it. What is verifiable:

- The **Python Developers Survey 2024** (PSF + JetBrains; responses collected
  October–November 2024, 30,000+ respondents;
  <https://lp.jetbrains.com/python-developers-survey-2024/>, read 2026-11-02; the 2025
  edition was not yet published — the URL 404s on that date) gives adjacent facts, not task
  prevalence:
  - environment isolation tools used: venv 62%, virtualenv 25%, Conda 18%, Poetry 18%,
    uv 11%, Pipenv 8%;
  - dependency management tools: pip 74%, Poetry 20%, Conda 18%, uv 12%, Pipenv 8%,
    pip-tools 9%;
  - "What format(s) is your application dependency information stored in?":
    requirements.txt 59%, **pyproject.toml 36%**, setup.py 16%;
  - package-creation tools: Twine 44%, PyPI Publish GitHub Action 31%, Poetry 29%,
    **Hatch 10%, PDM 5%**.
  The survey never asks about task runners. uv's 11–12% is first-year (2024) data and has
  very likely grown — but by how much is not measured by this source.
- **PyPI download counts** (pypistats.org, which serves the official PyPI download dataset;
  "last month" figures read 2026-11-02) are a rough proxy for runner adoption, with the
  standing caveats that downloads count CI runs and dependencies of other tools, not
  distinct projects: tox 14.3M, pipenv 8.0M, hatch 7.2M, poethepoet 5.5M, nox 3.8M,
  pdm 2.5M, taskipy 1.5M, doit 0.5M downloads/month; invoke's 60M/month is inflated by
  fabric's dependency on it. These say nothing about which files exist in a repo.
- **What the file landscape looks like structurally** (from the per-tool findings above,
  not from any census): pyproject.toml is the one file every Python tool ecosystem anchors
  on — PEP 518 (<https://peps.python.org/pep-0518/>) requires it for build-system
  declarations, setuptools' quickstart says "Every python package must provide a
  pyproject.toml" (<https://setuptools.pypa.io/en/latest/userguide/quickstart.html>), and
  all five pyproject-based runners discover it by walking up from the working directory
  (poe, pdm, hatch, tox, taskipy sections above). Whether a given repo *also* has
  `tox.ini`/`tox.toml`, a `Pipfile`, or a Makefile alongside it varies per project and is
  exactly the thing for which no source exists.

## Brief exclusions: Python-code task files

Scope decision: the picker does not parse `.py` files, no regex, no heuristics. Each tool
below confirms the same fact: its task list is defined by Python functions/decorators, so it
is knowable only by executing (or importing) the project's code.

- **nox — `noxfile.py`** (latest 2026.10.20, 2026-10-20, <https://pypi.org/project/nox/>).
  "Nox sessions are configured via standard Python functions that are decorated with
  `@nox.session`" (<https://github.com/wntrblm/nox/blob/main/docs/config.rst>, "Defining
  sessions"; run via `nox -s <session>`). Static config: **there is no `[tool.nox]` table** —
  checked the nox docs (`config.rst`, `usage.rst`) and source (`nox/_options.py`), read
  2026-11-02; pyproject.toml appears only as *helpers* for noxfile authors
  (`nox.project` namespace — <https://github.com/wntrblm/nox/blob/main/docs/config.rst>,
  "The pyproject.toml helpers"), and `nox.options` lives inside the noxfile, which is code.
  Session descriptions are docstrings of the functions (same page, "Session description").
- **invoke — `tasks.py`** (latest 3.0.3, 2026-04-07, <https://pypi.org/project/invoke/>).
  "simply calling `invoke` will look for a single Python module or package named `tasks`"
  (<https://github.com/pyinvoke/invoke/blob/main/sites/docs/concepts/loading.rst>), tasks are
  `@task`-decorated functions
  (<https://github.com/pyinvoke/invoke/blob/main/sites/docs/getting-started.rst>), run with
  `invoke <task>`. Static config: `invoke.yaml` (system, user, or project-level next to
  `tasks.py`) and the `tasks.*`/`run.*` keys hold options, never tasks
  (<https://github.com/pyinvoke/invoke/blob/main/sites/docs/concepts/configuration.rst>).
- **doit — `dodo.py`** (latest 0.37.0, 2026-02-09, <https://pypi.org/project/doit/>).
  "A function that starts with the name `task_` defines a *task-creator* recognized by
  `doit`… a python module/file that defines *tasks* for `doit` is called **dodo** file";
  "When `doit` is executed without any parameters it will look for tasks in a file named
  `dodo.py` in the current folder"
  (<https://github.com/pydoit/doit/blob/master/doc/tasks.rst>). Static config: `[tool.doit]`
  in pyproject.toml is "the preferred configuration source" but holds "command line
  options", not tasks (<https://github.com/pydoit/doit/blob/master/doc/configuration.rst>);
  the legacy `doit.cfg` and the `DOIT_CONFIG` dict (in `dodo.py`) are options too.
- **paver — `pavement.py`** (latest 1.3.4, uploaded 2017-12-31,
  <https://pypi.org/project/paver/> — effectively dead for ~9 years). "the paver command
  will look for a `pavement.py` file in the current directory. `pavement.py` is a standard
  Python module"; "The `@task` decorator just tells Paver that this is a task… The function
  name determines the name of the task. The docstring is what shows up in Paver's help
  output" (<https://github.com/paver/paver/blob/master/docs/source/getting_started.rst>).
  No static config file.
- **SCons — `SConstruct`** (latest 4.11.1, 2026-08-27, <https://pypi.org/project/SCons/>).
  "since SConscript files are Python scripts, you can use more flexible built-in Python
  services to perform these tasks"
  (<https://github.com/SCons/scons/blob/master/doc/user/tasks.xml>, SCons user guide, task
  recipes appendix); the getting-started shows a `SConstruct` whose content is a Python
  call, `Program('hello.c')`
  (<https://github.com/SCons/scons/blob/master/doc/user/simple.xml>). No static config
  file.

Same family, same conclusion: **pytask** (0.6.0, 2026-05-01) — "A task is a function that is
detected if the module and the function name are prefixed with `task_`"
(<https://github.com/pytask-dev/pytask/blob/main/README.md>); its `[tool.pytask.ini_options]`
pyproject table holds options, not tasks (see the uv section). A Django project's
`manage.py` is likewise Python code invoked as `python manage.py <command>`
(<https://docs.djangoproject.com/en/stable/ref/django-admin/>); excluded by the same rule.

## Brief exclusions: entry points and Poetry

The maintainer decided the picker will not show application entry points.

- `[project.scripts]` (PEP 621): console-script entry points — "[project.scripts]
  corresponds to the `console_scripts` group… The key of the table is the name of the entry
  point and the value is the object reference"
  (<https://packaging.python.org/en/latest/specifications/pyproject-toml/#entry-points>,
  read 2026-11-02). They exist only once the package is installed; excluded.
- `[project.gui-scripts]`: the same shape for the `gui_scripts` group (same page). Excluded.
- `[project.entry-points.<group>]`: plugin entry points for other applications (same
  page). Excluded — not even commands.
- `[tool.poetry.scripts]`: Poetry's entry-point section, deprecated in favor of
  `[project.scripts]` except for `type = "file"` scripts
  (<https://github.com/python-poetry/poetry/blob/main/docs/pyproject.md>, "scripts" and
  "tool.poetry.scripts"). Excluded.

**Poetry overall** (2.5.1, uploaded 2026-09-20, <https://pypi.org/project/poetry/>): Poetry
declares runnable dev tasks **nowhere** in a static file. Its pyproject documentation covers
dependencies, extras, scripts/entry points and build configuration
(<https://github.com/python-poetry/poetry/blob/main/docs/pyproject.md>); `poetry run
<command>` executes an arbitrary command or an installed entry point in the project
environment (<https://github.com/python-poetry/poetry/blob/main/docs/pyproject.md>,
"poetry run"), and there is no `[tool.poetry.tasks]`-style table. The community's answer is
poethepoet, which runs as a Poetry plugin (`poetry poe <task>`) or dev dependency
(`poetry run poe <task>`) —
<https://github.com/nat-n/poethepoet/blob/main/docs/poetry_plugin.rst>,
<https://github.com/nat-n/poethepoet/blob/main/docs/guides/running_guide.rst>. A Poetry
project that wants listable tasks, then, hosts `[tool.poe.tasks]` in its pyproject.toml.

## Patterns across the tools

- **One anchor file, discovered by walking up.** poe, pdm, hatch, tox and taskipy all read
  `pyproject.toml` and all search the current directory then parents (citations in each
  section above); pipenv does the same for `Pipfile`. Only poe names extra task files
  (`poe_tasks.toml`/`poe_tasks.yaml`/`poe_tasks.json`, plus `[tool.poe] include`), and only
  tox has same-content alternatives (`tox.ini`, `setup.cfg`, `tox.toml`). Everything else is
  pyproject.toml.
- **The section name is the tool's namespace**: `[tool.<tool>.tasks]`/`[scripts]`/env
  sub-tables. Two tools can coexist in one pyproject.toml (poe and taskipy sections do not
  collide), so the picker's task label needs the tool name, not just the task name.
- **A task entry is one of: string, table, array** — in every tool. Strings are commands
  (except in poe, where `default_task_type` can reinterpret them); tables add options
  (`cmd`, `shell`, `call`, `script`, `expr`, `help`, `env`, `cwd`, `args`); arrays are
  sequences (poe `sequence`, pdm `composite`, pipenv arrays, hatch scripts, tox `commands`).
- **A per-task help field exists in most, not all**: `help` (poe, pdm, taskipy), `description`
  per environment (tox, hatch). Hatch has no per-script description; Pipfile has none at
  all. When absent, the tools' own listings show the raw command — a picker can do the same.
- **Hidden-by-convention names**: poe and pdm hide tasks starting with `_` from their
  listings; pdm and taskipy also give `pre_*`/`post_*` names automatic hook semantics. A
  picker should hide `_`-prefixed names and decide explicitly about `pre_*`/`post_*`.
- **Templating is the norm, not the exception**: `${VAR}` with `:-`/`:+` operators (poe,
  pdm), `{env:VAR:DEFAULT}`/`{root}`/`{args}` (hatch), `{posargs}`/`{env:KEY}`/`{glob:…}`
  plus config-key references (tox), `{args}`/`{args:--default}` (pipenv), `{var}`
  (taskipy `use_vars`). The raw string is display text, never a promise of the executed
  command.
- **Two escape hatches out of shell**: a Python callable (`script`/`expr` in poe, `call` in
  pdm and pipenv) and a reference to another task (`ref` in poe, `composite` items in pdm,
  leading-word script references in hatch, `sequence` items in poe). A picker can show
  them, but running them is meaningful only through the tool's own runner.
- **Environment is part of the task**: every runner executes inside a managed environment
  (poe executors uv/poetry/venv; pdm/pipenv project venv; hatch envs; tox envs) and most
  allow per-task environment variables (`env`/`envfile`/`env-vars`/`set_env`) and a working
  directory (`cwd`/`working_dir`/`change_dir`). This is why the run command must go through
  the tool, not the raw string.
- **Matrix/conditional machinery exists in the heavyweight tools**: hatch matrices and
  `overrides`, tox generative `env_list`/factor conditions. Both make the file *understate*
  the environments that actually exist; the opposite of a static parser's risk.
- **Description of "the task list" varies in granularity**: poe/pdm list tasks; hatch lists
  environments (scripts are inside); tox lists environments (commands are inside). What a
  user thinks of as "tasks" in hatch and tox is one level below what the file names.

## What this suggests for what-to-run

- **Read `pyproject.toml` first, as the primary file.** It is the only file every tool
  anchors on, and per PEP 518 every buildable project has it
  (<https://peps.python.org/pep-0518/>). Then read the same-content alternatives in the
  tools that have them: `tox.ini`/`tox.toml`/`Pipfile`. The existing registry's one-source-
  per-tool shape maps cleanly: a `poe` source (candidate files `pyproject.toml`,
  `poe_tasks.toml`, `poe_tasks.yaml`, `poe_tasks.yml`, `poe_tasks.json`), a `pdm` source
  (`pyproject.toml`), a `hatch` source (`pyproject.toml`), a `taskipy` source
  (`pyproject.toml`), a `tox` source (`tox.ini`, `tox.toml`, `pyproject.toml`), and a
  `pipenv` source (`Pipfile`). Each parser reads only its own `[tool.<id>…]` table(s), so
  coexistence needs no precedence logic.
- **Search parent directories, like every runner does.** Users invoke `poe`, `pdm run`,
  `tox` from subdirectories and expect the parent pyproject.toml to be found; the picker
  should honor `SearchParents` the same way (poe, pdm, hatch, tox, taskipy, pipenv
  citations above).
- **Sections and shapes per tool** (the direct answer to the spec-to-be):

  | tool | file(s) | section(s) | entry shapes | description | run prefix |
  |---|---|---|---|---|---|
  | poethepoet | pyproject.toml (+poe_tasks.*, include files) | `[tool.poe.tasks]` | string=cmd, array=sequence of refs, table typed by `cmd`/`shell`/`script`/`expr`/`sequence`/`parallel`/`switch`/`ref` | task `help` | `poe ` (via `uv run poe`/`poetry run poe`/`poetry poe`) |
  | pdm | pyproject.toml | `[tool.pdm.scripts]` | string=cmd, `{cmd=…}` (string or array), `{shell=…}`, `{call=…}`, `{composite=[…]}` | task `help` | `pdm run ` |
  | hatch | pyproject.toml | `[tool.hatch.envs.<env>]` + `.scripts` | string or array of strings; leading word may reference another script | env `description` only | `hatch run <env>:` |
  | taskipy | pyproject.toml | `[tool.taskipy.tasks]` | string, or `{cmd, help, cwd, use_vars}` | task `help` | `task ` (or `poetry run task `) |
  | tox 4 | tox.toml / pyproject.toml / tox.ini / setup.cfg | `[tool.tox]` + `[tool.tox.env.<name>]`, `env_list` (+ `[testenv:<name>]` in INI) | env table with `commands` (array of string or string-array), `deps`, `set_env` | env `description` | `tox run -e ` |
  | pipenv | Pipfile | `[scripts]` | string, array of strings (sequence), `{call=…}` | none | `pipenv run ` |

- **Beware list, decide on**: poe `args` with `required`/`positional` (task fails bare —
  still showable, the runner errors helpfully); pdm/taskipy `pre_*`/`post_*` hooks and the
  `_` shared-options key in pdm; poe `_`-prefixed hidden tasks; hatch matrices/overrides and
  the fact scripts live under envs (also inherited via `template`); tox generative
  `env_list`/factor conditions and the `[testenv]`/`env_run_base` fallback tables;
  `default_task_type`-style reinterpretation of plain strings in poe; per-tool templating
  (never promise the raw string is the command). The picker should not flatten
  hatch/tox env-scoped tasks into bare commands: the run prefix must carry the env.
- **Do not parse**: `.py` files (noxfile.py, tasks.py, dodo.py, pavement.py, SConstruct,
  task_*.py, manage.py) and entry-point tables — per the maintainer decision, with the
  confirmations above that they are code or install-time commands.
- **A description fallback chain**: task `help` → env `description` (hatch, tox) → the raw
  command string (pipenv, taskipy strings, hatch scripts). All the in-depth tools print a
  listing themselves (`poe`, `pdm run --list`, `task --list`, `tox list`, `pipenv scripts`,
  `hatch env show`), which is the behavior a picker replaces.
- **The uv question is settled by poe**: since uv has no task file, a uv project's tasks
  live in `[tool.poe.tasks]` (or `[tool.taskipy.tasks]`) and run as `uv run poe <task>` —
  reading pyproject.toml covers uv projects with no extra source.

## Unverified

- **Prevalence is open by design**: no source exists that measures which task sections
  Python projects use. The Python Developers Survey 2024 measures tool adoption, not task
  files, and its 2025 edition was not yet published on 2026-11-02 (the URL 404s). The
  pypistats.org download figures are a community service over PyPI's official download
  dataset; I did not audit them, and downloads are not projects.
- **Docs vs. code discrepancies are recorded, not resolved**: pipenv's docs present a
  `{cmd = …}` script shape that its `main`-branch parser rejects (`script_types =
  ["call"]`); tox's man page lists a different config-file search order than its
  `config.rst` and source code. In both cases I cite the implementation as authoritative
  for the released behavior, but I did not execute the released binaries to confirm.
- **I read documentation and source, not behavior**: I did not run any of these runners
  against real projects, so per-tool edge cases (poe arg prompting, hatch matrix expansion,
  tox substitution chains) rest on the docs and code, not on experiments. No PTY-style
  verification was attempted, matching the repo's own no-PTY-tests rule.
- **Branch skew**: poe, pdm, hatch, tox, pipenv, nox, invoke, doit and SCons facts were read
  from their default branches on 2026-11-02, which may be hours-to-weeks ahead of the
  released versions quoted from PyPI (tox `main` in particular documents the not-yet-everywhere
  TOML-first wording; the released 4.67.5 is from 2026-10-31).
- **Not checked**: Windows-only behavior of any tool; conda projects (`environment.yml`
  declares dependencies, not tasks); Bazel/Please (BUILD files are code, out of scope by
  the same rule); tool ecosystems smaller than the ones here (I stopped at the named
  scope); and whether any of the six runners changed release on 2026-11-02 itself.
- The comparable-picker claim ("none found for Python") is after checking fzf-make and
  television official docs, not an exhaustive survey of the TUI ecosystem.

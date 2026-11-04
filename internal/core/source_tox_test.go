package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// toxIniFixture is the INI form of the tox configuration, with the same
// environments as the TOML fixtures of the table below. The [testenv] base
// describes "3.13", because tox falls back to the base for every run
// environment, and the INI half of the source honors that fallback. The
// parity test pins this fixture to the same task names as the TOML form,
// which keeps the two formats from drifting apart.
const toxIniFixture = `[tox]
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
`

// toxSetupCfgFixture is a real setup.cfg: the core section carries the tox:
// prefix, because the file hosts the sections of other tools, while the
// testenv sections keep the plain spelling. The environments match the
// tox.ini fixture, so the parity test holds for both spellings of the core
// section.
const toxSetupCfgFixture = `[metadata]
name = demo

[tox:tox]
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
`

func TestDiscoverToxEnvironments(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "environments from env_list and from the env tables, sorted",
			files: map[string]string{
				"pyproject.toml": `
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
`,
			},
			// "3.13" appears in env_list only, so no table describes it. The
			// description of env_run_base does not reach it, because the base
			// table is not an environment.
			want: []Task{
				{Source: "pyproject.toml", Name: "3.13", Desc: "", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "a tox.toml holds the same tables without the tool.tox prefix",
			files: map[string]string{
				"tox.toml": `
env_list = ["3.13", "lint"]

[env_run_base]
description = "run the test suite with pytest"
deps = ["pytest>=8"]
commands = [["pytest", { replace = "posargs", default = ["tests"], extend = true }]]

[env.lint]
description = "run linters"
skip_install = true
deps = ["ruff"]
commands = [["ruff", "check", { replace = "posargs", default = ["."], extend = true }]]
`,
			},
			want: []Task{
				{Source: "tox.toml", Name: "3.13", Desc: "", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "tox.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "a tox.ini lists the environments of the ini form",
			files: map[string]string{
				"tox.ini": toxIniFixture,
			},
			// "3.13" appears in env_list only, so its description comes
			// from the [testenv] base, which tox applies to every run
			// environment.
			want: []Task{
				{Source: "tox.ini", Name: "3.13", Desc: "run the test suite with pytest", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "tox.ini", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "a setup.cfg holds the core section under the tox prefix",
			files: map[string]string{
				"setup.cfg": toxSetupCfgFixture,
			},
			// Only the core section carries the tox: prefix in a
			// setup.cfg, so the [testenv] and [testenv:lint] sections
			// parse like the ones of a tox.ini.
			want: []Task{
				{Source: "setup.cfg", Name: "3.13", Desc: "run the test suite with pytest", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "setup.cfg", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "the legacy envlist spelling lists the same environments",
			files: map[string]string{
				"tox.ini": `
[tox]
envlist = py312,py311

[testenv]
deps = pytest
commands = pytest tests
`,
			},
			// A classic pre-TOML file names its environments through
			// the legacy spelling of the key. The base section is
			// the only one that holds commands, so its first command
			// describes every environment.
			want: []Task{
				{Source: "tox.ini", Name: "py311", Desc: "pytest tests", Args: []string{"tox", "run", "-e", "py311"}},
				{Source: "tox.ini", Name: "py312", Desc: "pytest tests", Args: []string{"tox", "run", "-e", "py312"}},
			},
		},
		{
			name: "env_list wins when a file holds both spellings",
			files: map[string]string{
				"tox.ini": "[tox]\nenv_list = lint\nenvlist = wrong\n\n[testenv:lint]\ndescription = run linters\n",
			},
			want: []Task{
				{Source: "tox.ini", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "a tox.ini wins over the other candidates",
			files: map[string]string{
				"tox.ini":        toxIniFixture,
				"setup.cfg":      "[tox:tox]\nenv_list = wrong\n",
				"pyproject.toml": "[tool.tox]\nenv_list = [\"wrong\"]\n",
				"tox.toml":       "env_list = [\"wrong\"]\n",
			},
			want: []Task{
				{Source: "tox.ini", Name: "3.13", Desc: "run the test suite with pytest", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "tox.ini", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "a setup.cfg wins over the toml candidates",
			files: map[string]string{
				"setup.cfg":      toxSetupCfgFixture,
				"pyproject.toml": "[tool.tox]\nenv_list = [\"wrong\"]\n",
				"tox.toml":       "env_list = [\"wrong\"]\n",
			},
			want: []Task{
				{Source: "setup.cfg", Name: "3.13", Desc: "run the test suite with pytest", Args: []string{"tox", "run", "-e", "3.13"}},
				{Source: "setup.cfg", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "without descriptions the base describes, then the first command shows",
			files: map[string]string{
				"tox.ini": `
[tox]
env_list = cover

[testenv]
commands =
    pytest --cov {posargs:tests}

[testenv:cover]

[testenv:docs]
commands =
    sphinx-build docs {posargs:build}
`,
			},
			// "cover" inherits the commands of the [testenv] base, so the
			// base's first command describes it. "docs" declares commands
			// of its own, which override the base as a whole.
			want: []Task{
				{Source: "tox.ini", Name: "cover", Desc: "pytest --cov {posargs:tests}", Args: []string{"tox", "run", "-e", "cover"}},
				{Source: "tox.ini", Name: "docs", Desc: "sphinx-build docs {posargs:build}", Args: []string{"tox", "run", "-e", "docs"}},
			},
		},
		{
			name: "comments, blank lines and continuation lines parse",
			files: map[string]string{
				"tox.ini": `# the environments of the project
[tox]
; also a comment
env_list = lint

[testenv:lint]
description = run the linters with ruff
commands =
    ruff check \
        {posargs:.}
    ruff format --check .
    [ -f pyproject.toml ] && echo legacy

[testenv:type]
    description = run mypy
    commands = mypy {posargs:src}
`,
			},
			// The backslash joins the first command with its second
			// physical line, so the description shows one command and
			// not a dangling backslash. The bracket line stays a
			// continuation, because it is indented deeper than its key.
			// The indented keys of [testenv:type] parse like the flat
			// ones.
			want: []Task{
				{Source: "tox.ini", Name: "lint", Desc: "run the linters with ruff", Args: []string{"tox", "run", "-e", "lint"}},
				{Source: "tox.ini", Name: "type", Desc: "run mypy", Args: []string{"tox", "run", "-e", "type"}},
			},
		},
		{
			name: "a colon separates a key from its value too",
			files: map[string]string{
				"tox.ini": "[tox]\nenv_list: lint\n\n[testenv:lint]\ndescription: run linters\n",
			},
			want: []Task{
				{Source: "tox.ini", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "an environment without a description falls back to its first command",
			files: map[string]string{
				"pyproject.toml": `
[tool.tox.env.lint]
commands = ["ruff check .", "ruff format --check ."]

[tool.tox.env.type]
commands = [["mypy", { replace = "posargs", default = ["src"] }]]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "lint", Desc: "ruff check .", Args: []string{"tox", "run", "-e", "lint"}},
				{Source: "pyproject.toml", Name: "type", Desc: "mypy {posargs}", Args: []string{"tox", "run", "-e", "type"}},
			},
		},
		{
			name:  "an environment that only env_list names has no description",
			files: map[string]string{"pyproject.toml": "[tool.tox]\nenv_list = [\"3.13\", \"3.12\"]\n"},
			want: []Task{
				{Source: "pyproject.toml", Name: "3.12", Desc: "", Args: []string{"tox", "run", "-e", "3.12"}},
				{Source: "pyproject.toml", Name: "3.13", Desc: "", Args: []string{"tox", "run", "-e", "3.13"}},
			},
		},
		{
			name: "an environment in both env_list and the env tables is listed once",
			files: map[string]string{
				"pyproject.toml": `
[tool.tox]
env_list = ["lint"]

[tool.tox.env.lint]
description = "run linters"
commands = [["ruff", "check"]]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "names with template braces stay hidden",
			files: map[string]string{
				"pyproject.toml": `
[tool.tox]
env_list = ["py3{9-11}", "lint"]

[tool.tox.env."py{10,11}"]
commands = [["python", "-V"]]

[tool.tox.env.lint]
description = "run linters"
commands = [["ruff", "check"]]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "ini names with template braces stay hidden",
			files: map[string]string{
				"tox.ini": `
[tox]
env_list = py3{9-11}, lint

[testenv:py{10,11}-{linux,windows}]
commands = python -V

[testenv:lint]
description = run linters
commands = ruff check
`,
			},
			want: []Task{
				{Source: "tox.ini", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "pyproject.toml wins over tox.toml",
			files: map[string]string{
				"pyproject.toml": "[tool.tox.env.lint]\ndescription = \"run linters\"\ncommands = [[\"ruff\", \"check\"]]\n",
				"tox.toml":       "[env.type]\ncommands = [[\"mypy\", \"src\"]]\n",
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"pyproject.toml": `
[tool.tox]
env_list = ["lint", 42]

[tool.tox.env]
not-a-table = "a string value"

[tool.tox.env.lint]
description = 42
commands = "not an array"

[tool.tox.env.type]
commands = [["mypy", 42]]

[tool.tox.env.docs]
skip_install = true
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "docs", Desc: "", Args: []string{"tox", "run", "-e", "docs"}},
				{Source: "pyproject.toml", Name: "lint", Desc: "", Args: []string{"tox", "run", "-e", "lint"}},
				{Source: "pyproject.toml", Name: "type", Desc: "", Args: []string{"tox", "run", "-e", "type"}},
			},
		},
		{
			name:    "malformed toml in pyproject.toml",
			files:   map[string]string{"pyproject.toml": "[tool.tox\n"},
			wantErr: true,
		},
		{
			name:    "malformed toml in tox.toml",
			files:   map[string]string{"tox.toml": "[env\n"},
			wantErr: true,
		},
		{
			name:    "malformed ini: a line without a separator",
			files:   map[string]string{"tox.ini": "[tox]\nenv_list\n"},
			wantErr: true,
		},
		{
			name:    "malformed ini: a section header without a closing bracket",
			files:   map[string]string{"tox.ini": "[testenv:lint\n"},
			wantErr: true,
		},
		{
			name:    "malformed ini: a key outside a section",
			files:   map[string]string{"setup.cfg": "env_list = lint\n"},
			wantErr: true,
		},
		{
			name:  "empty file",
			files: map[string]string{"pyproject.toml": ""},
			want:  nil,
		},
		{
			name:  "whitespace only file",
			files: map[string]string{"pyproject.toml": " \n\t "},
			want:  nil,
		},
		{
			name:  "an ini file with comments only",
			files: map[string]string{"tox.ini": "# no environments here\n; still none\n"},
			want:  nil,
		},
		{
			name:  "a setup.cfg without tox sections contributes nothing",
			files: map[string]string{"setup.cfg": "[metadata]\nname = demo\n\n[options]\npackages = find:\n"},
			want:  nil,
		},
		{
			name:  "a [tox] section in a setup.cfg is not the core section",
			files: map[string]string{"setup.cfg": "[tox]\nenv_list = wrong\n"},
			want:  nil,
		},
		{
			name:  "no tox table",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			want:  nil,
		},
		{
			name:  "a tox table without environments contributes nothing",
			files: map[string]string{"pyproject.toml": "[tool.tox]\nrequires = [\"tox>=4\"]\n"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, warn := discoverSource(osFiles{}, toxSource(), dir)
			if tt.wantErr {
				if warn == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				for name := range tt.files {
					if !strings.Contains(warn.Error(), name) {
						t.Errorf("error = %q, want it to mention %s", warn, name)
					}
				}
				return
			}
			if warn != nil {
				t.Fatalf("discoverSource() error = %v", warn)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverSource() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDiscoverToxMissingFile(t *testing.T) {
	got, warn := discoverSource(osFiles{}, toxSource(), t.TempDir())
	if warn != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, warn)
	}
}

// TestDiscoverToxIniMatchesToml: the same configuration in the INI and the
// TOML form lists the same environments with the same run command, whichever
// candidate file holds it. The setup.cfg fixture holds the core section under
// the tox: prefix, the spelling that the file requires. The descriptions are
// not compared, because the [testenv] base describes an environment that
// only env_list names, while the env_run_base table does not.
func TestDiscoverToxIniMatchesToml(t *testing.T) {
	toxTomlFixture := `
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
`
	files := map[string]string{
		"tox.ini":        toxIniFixture,
		"setup.cfg":      toxSetupCfgFixture,
		"pyproject.toml": toxTomlFixture,
	}

	var want []string
	for _, name := range []string{"tox.ini", "setup.cfg", "pyproject.toml"} {
		dir := t.TempDir()
		writeFile(t, dir, name, files[name])

		tasks, warn := discoverSource(osFiles{}, toxSource(), dir)
		if warn != nil {
			t.Fatalf("discoverSource() on %s error = %v", name, warn)
		}
		var got []string
		for _, task := range tasks {
			got = append(got, task.Name+" "+strings.Join(task.Args, " "))
		}
		if want == nil {
			want = got
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("discoverSource() on %s lists %v, want %v", name, got, want)
		}
	}
	if want == nil {
		t.Fatal("discoverSource() listed no environment on any fixture")
	}
}

// TestDiscoverToxSearchesParents: tox walks up to find its configuration, so
// the source lists the environments of a pyproject.toml of a parent
// directory, badged with its relative path.
func TestDiscoverToxSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pyproject.toml", "[tool.tox.env.lint]\ndescription = \"run linters\"\ncommands = [[\"ruff\", \"check\"]]\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, warn := discoverSource(osFiles{}, toxSource(), sub)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

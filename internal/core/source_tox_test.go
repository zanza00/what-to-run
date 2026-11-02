package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// toxIniFixture is the INI form of the tox configuration, with the same
// environments as the pyproject.toml fixture of the table below. The source
// reads the TOML candidates only, so no test runs on this fixture yet.
// Ticket 08 lands tox.ini and setup.cfg as candidates and pins this fixture
// to the same task list as the TOML cases, which keeps the two formats from
// drifting apart.
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

			got, err := discoverSource(osFiles{}, toxSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				for name := range tt.files {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("error = %q, want it to mention %s", err, name)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("discoverSource() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverSource() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDiscoverToxMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, toxSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
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

	tasks, err := discoverSource(osFiles{}, toxSource(), sub)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "lint", Desc: "run linters", Args: []string{"tox", "run", "-e", "lint"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

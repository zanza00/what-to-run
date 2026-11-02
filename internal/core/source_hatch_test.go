package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverHatchScripts(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "scripts are listed with their environment and sorted",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.test]
dependencies = ["pytest", "pytest-cov"]

[tool.hatch.envs.test.scripts]
run-coverage = "pytest --cov-config=pyproject.toml --cov=pkg --cov=tests"
run = "run-coverage --no-cov"

[tool.hatch.envs.docs.scripts]
serve = "mkdocs serve"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "docs:serve", Desc: "mkdocs serve", Args: []string{"hatch", "run", "docs:serve"}},
				{Source: "pyproject.toml", Name: "test:run", Desc: "run-coverage --no-cov", Args: []string{"hatch", "run", "test:run"}},
				{Source: "pyproject.toml", Name: "test:run-coverage", Desc: "pytest --cov-config=pyproject.toml --cov=pkg --cov=tests", Args: []string{"hatch", "run", "test:run-coverage"}},
			},
		},
		{
			name: "the environment description describes every script of the environment",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.default]
description = "Run the linters and the formatters"

[tool.hatch.envs.default.scripts]
fmt = "ruff format ."
lint = "ruff check ."
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "default:fmt", Desc: "Run the linters and the formatters", Args: []string{"hatch", "run", "default:fmt"}},
				{Source: "pyproject.toml", Name: "default:lint", Desc: "Run the linters and the formatters", Args: []string{"hatch", "run", "default:lint"}},
			},
		},
		{
			name: "an empty or non-string description falls back to the script",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.empty-desc]
description = ""
platforms = ["linux"]

[tool.hatch.envs.empty-desc.scripts]
fmt = "ruff format ."

[tool.hatch.envs.wrong-desc]
description = 42

[tool.hatch.envs.wrong-desc.scripts]
lint = "ruff check ."
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "empty-desc:fmt", Desc: "ruff format .", Args: []string{"hatch", "run", "empty-desc:fmt"}},
				{Source: "pyproject.toml", Name: "wrong-desc:lint", Desc: "ruff check .", Args: []string{"hatch", "run", "wrong-desc:lint"}},
			},
		},
		{
			name: "an array script joins its commands",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.docs.scripts]
build = ["mkdocs build", "rsync -a site/ docs-host:."]
serve = "mkdocs serve"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "docs:build", Desc: "mkdocs build && rsync -a site/ docs-host:.", Args: []string{"hatch", "run", "docs:build"}},
				{Source: "pyproject.toml", Name: "docs:serve", Desc: "mkdocs serve", Args: []string{"hatch", "run", "docs:serve"}},
			},
		},
		{
			name: "hatch-, matrix and overrides environments stay hidden",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.hatch-test.scripts]
run = "pytest tests"

[tool.hatch.envs.test]
dependencies = ["pytest"]

[tool.hatch.envs.test.scripts]
run = "pytest"

[[tool.hatch.envs.test.matrix]]
python = ["3.10", "3.11"]

[tool.hatch.envs.lint]
dependencies = ["ruff"]

[tool.hatch.envs.lint.scripts]
run = "ruff check ."

[tool.hatch.envs.lint.overrides]
platform.windows.dependencies = ["ruff"]

[tool.hatch.envs.docs.scripts]
serve = "mkdocs serve"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "docs:serve", Desc: "mkdocs serve", Args: []string{"hatch", "run", "docs:serve"}},
			},
		},
		{
			name: "two environments can declare the same script name",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs.test.scripts]
run = "pytest"

[tool.hatch.envs.lint.scripts]
run = "ruff check ."
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "lint:run", Desc: "ruff check .", Args: []string{"hatch", "run", "lint:run"}},
				{Source: "pyproject.toml", Name: "test:run", Desc: "pytest", Args: []string{"hatch", "run", "test:run"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"pyproject.toml": `
[tool.hatch.envs]
not-a-table = "a string value"

[tool.hatch.envs.good.scripts]
ok = "echo good"

[tool.hatch.envs.bad-scripts]
scripts = "not a table"

[tool.hatch.envs.number.scripts]
run = 42

[tool.hatch.envs.flag.scripts]
run = true

[tool.hatch.envs.table.scripts]
run = { cmd = "echo no" }

[tool.hatch.envs.mixed-array.scripts]
run = ["echo one", 42]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "good:ok", Desc: "echo good", Args: []string{"hatch", "run", "good:ok"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"pyproject.toml": "[tool.hatch.envs\n"},
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
			name:  "no hatch table",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			want:  nil,
		},
		{
			name: "tool.hatch without envs",
			files: map[string]string{
				"pyproject.toml": "[tool.hatch.version]\npath = \"src/pkg/__about__.py\"\n",
			},
			want: nil,
		},
		{
			name: "an environment without scripts contributes nothing",
			files: map[string]string{
				"pyproject.toml": "[tool.hatch.envs.test]\ndependencies = [\"pytest\"]\n",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(osFiles{}, hatchSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "pyproject.toml") {
					t.Errorf("error = %q, want it to mention pyproject.toml", err)
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

func TestDiscoverHatchScriptsMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, hatchSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestDiscoverHatchTasksSearchesParents: Hatch walks up to find the
// pyproject.toml, so the source lists the scripts of a pyproject.toml of a
// parent directory, badged with its relative path.
func TestDiscoverHatchScriptsSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pyproject.toml", "[tool.hatch.envs.docs.scripts]\nserve = \"mkdocs serve\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, err := discoverSource(osFiles{}, hatchSource(), sub)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "docs:serve", Desc: "mkdocs serve", Args: []string{"hatch", "run", "docs:serve"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

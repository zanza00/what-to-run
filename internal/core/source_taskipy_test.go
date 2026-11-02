package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverTaskipyTasks(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string and table tasks are listed and sorted",
			files: map[string]string{
				"pyproject.toml": `
[tool.taskipy.tasks]
test = { cmd = "python -m unittest tests/test_*.py", help = "runs all unit tests" }
lint = "pylint tests taskipy"

[tool.taskipy.tasks.build]
cmd = "python -m build"

[tool.taskipy.tasks.echo-var]
cmd = "echo {greeting}"
use_vars = true
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "build", Desc: "python -m build", Args: []string{"task", "build"}},
				{Source: "pyproject.toml", Name: "echo-var", Desc: "echo {greeting}", Args: []string{"task", "echo-var"}},
				{Source: "pyproject.toml", Name: "lint", Desc: "pylint tests taskipy", Args: []string{"task", "lint"}},
				{Source: "pyproject.toml", Name: "test", Desc: "runs all unit tests", Args: []string{"task", "test"}},
			},
		},
		{
			name: "the companion tables contribute nothing",
			files: map[string]string{
				"pyproject.toml": `
[tool.taskipy.variables]
greeting = "hello"

[tool.taskipy.settings]
use_vars = true
runner = "poetry run"

[tool.taskipy.tasks]
test = "pytest"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"task", "test"}},
			},
		},
		{
			name: "pre and post hooks stay listed, because they run standalone",
			files: map[string]string{
				"pyproject.toml": `
[tool.taskipy.tasks]
pre_test = "rm -rf tests/__pycache__"
test = "pytest"
post_test = "coverage report"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "post_test", Desc: "coverage report", Args: []string{"task", "post_test"}},
				{Source: "pyproject.toml", Name: "pre_test", Desc: "rm -rf tests/__pycache__", Args: []string{"task", "pre_test"}},
				{Source: "pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"task", "test"}},
			},
		},
		{
			name: "an empty or non-string help falls back to the command",
			files: map[string]string{
				"pyproject.toml": `
[tool.taskipy.tasks]

[tool.taskipy.tasks.wrong-help]
cmd = "echo fallback"
help = 42

[tool.taskipy.tasks.empty-help]
cmd = "echo fallback"
help = ""
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "empty-help", Desc: "echo fallback", Args: []string{"task", "empty-help"}},
				{Source: "pyproject.toml", Name: "wrong-help", Desc: "echo fallback", Args: []string{"task", "wrong-help"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"pyproject.toml": `
[tool.taskipy.tasks]
good = "echo good"
number = 42
flag = true
list = ["echo", "array"]

[tool.taskipy.tasks.no-cmd]
help = "A table without a command"

[tool.taskipy.tasks.wrong-cmd]
help = "The command is no string"
cmd = 42

[tool.taskipy.tasks.cmd-array]
cmd = ["echo", "array"]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "good", Desc: "echo good", Args: []string{"task", "good"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"pyproject.toml": "[tool.taskipy.tasks\n"},
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
			name:  "no taskipy table",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			want:  nil,
		},
		{
			name: "tool.taskipy without tasks",
			files: map[string]string{
				"pyproject.toml": "[tool.taskipy.variables]\ngreeting = \"hello\"\n",
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

			got, err := discoverSource(osFiles{}, taskipySource(), dir)
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

func TestDiscoverTaskipyTasksMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, taskipySource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestDiscoverTaskipyTasksSearchesParents: taskipy walks up to find the
// pyproject.toml, so the source lists the tasks of a pyproject.toml of a
// parent directory, badged with its relative path.
func TestDiscoverTaskipyTasksSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pyproject.toml", "[tool.taskipy.tasks]\ntest = \"pytest\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, err := discoverSource(osFiles{}, taskipySource(), sub)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"task", "test"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

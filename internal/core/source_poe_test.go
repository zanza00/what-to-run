package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverPoeTasks(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string, array and table tasks of every type, sorted",
			files: map[string]string{
				"pyproject.toml": `
[tool.poe.tasks]
test = "pytest --cov=src"
greet = "echo Hello ${USER}"
release = ["test", "_build"]
_build = "poetry build"
_publish = "poetry publish"

[tool.poe.tasks.build]
help = "Build the package"
cmd = "poetry build"

[tool.poe.tasks.serve]
help = "Run the app in debug mode"
script = "my_app.service:run(debug=True)"

[tool.poe.tasks.tunnel]
shell = "ssh -N -L 0.0.0.0:8080:$PROD:8080 $PROD &"

[tool.poe.tasks.sum]
expr = "1 + 1"

[tool.poe.tasks.check]
sequence = ["test", "build"]

[tool.poe.tasks.check-all]
parallel = ["lint", "test"]

[tool.poe.tasks.deploy]
script = "my_app.cli:deploy()"

[tool.poe.tasks.build-on]
control.expr = "sys.platform"
switch = [
  { case = "win32", cmd = "windows_build" },
  { cmd = "posix_build" },
]

[tool.poe.tasks.do-specific-things]
ref = "do_things thing1 thing2"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "build", Desc: "Build the package", Args: []string{"poe", "build"}},
				{Source: "pyproject.toml", Name: "build-on", Desc: "", Args: []string{"poe", "build-on"}},
				{Source: "pyproject.toml", Name: "check", Desc: "test && build", Args: []string{"poe", "check"}},
				{Source: "pyproject.toml", Name: "check-all", Desc: "lint && test", Args: []string{"poe", "check-all"}},
				{Source: "pyproject.toml", Name: "deploy", Desc: "my_app.cli:deploy()", Args: []string{"poe", "deploy"}},
				{Source: "pyproject.toml", Name: "do-specific-things", Desc: "do_things thing1 thing2", Args: []string{"poe", "do-specific-things"}},
				{Source: "pyproject.toml", Name: "greet", Desc: "echo Hello ${USER}", Args: []string{"poe", "greet"}},
				{Source: "pyproject.toml", Name: "release", Desc: "test && _build", Args: []string{"poe", "release"}},
				{Source: "pyproject.toml", Name: "serve", Desc: "Run the app in debug mode", Args: []string{"poe", "serve"}},
				{Source: "pyproject.toml", Name: "sum", Desc: "1 + 1", Args: []string{"poe", "sum"}},
				{Source: "pyproject.toml", Name: "test", Desc: "pytest --cov=src", Args: []string{"poe", "test"}},
				{Source: "pyproject.toml", Name: "tunnel", Desc: "ssh -N -L 0.0.0.0:8080:$PROD:8080 $PROD &", Args: []string{"poe", "tunnel"}},
			},
		},
		{
			name: "underscore-prefixed tasks stay hidden",
			files: map[string]string{
				"pyproject.toml": `
[tool.poe.tasks]
_ = "echo hidden"
_setup = "pip install -e ."
serve = "uvicorn app:app"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "serve", Desc: "uvicorn app:app", Args: []string{"poe", "serve"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"pyproject.toml": `
[tool.poe.tasks]
good = "echo good"
number = 42
flag = true

[tool.poe.tasks.no-type]
help = "A table without a task type"

[tool.poe.tasks.wrong-cmd]
help = "The command is no string"
cmd = 42

[tool.poe.tasks.wrong-sequence]
sequence = "test"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "good", Desc: "echo good", Args: []string{"poe", "good"}},
			},
		},
		{
			name: "arrays with non-string items render no description",
			files: map[string]string{
				"pyproject.toml": `
[tool.poe.tasks]
mixed = ["test", { shell = "echo sub" }]

[tool.poe.tasks.inline]
sequence = ["lint", { cmd = "go vet ./..." }]
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "inline", Desc: "", Args: []string{"poe", "inline"}},
				{Source: "pyproject.toml", Name: "mixed", Desc: "", Args: []string{"poe", "mixed"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"pyproject.toml": "[tool.poe.tasks\n"},
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
			name:  "no poe table",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			want:  nil,
		},
		{
			name:  "tool.poe without tasks",
			files: map[string]string{"pyproject.toml": "[tool.poe]\nexecutor.type = \"uv\"\n"},
			want:  nil,
		},
		{
			name: "pyproject.toml wins over poe_tasks.toml",
			files: map[string]string{
				"pyproject.toml": "[tool.poe.tasks]\nfrom-pyproject = \"echo pyproject\"\n",
				"poe_tasks.toml": "[tasks]\nfrom-standalone = \"echo standalone\"\n",
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "from-pyproject", Desc: "echo pyproject", Args: []string{"poe", "from-pyproject"}},
			},
		},
		{
			name: "a standalone poe_tasks.toml lists top-level tasks",
			files: map[string]string{
				"poe_tasks.toml": `
[tasks]
test = "pytest"

[tasks.serve]
help = "Run the app"
script = "my_app.service:run()"
`,
			},
			want: []Task{
				{Source: "poe_tasks.toml", Name: "serve", Desc: "Run the app", Args: []string{"poe", "serve"}},
				{Source: "poe_tasks.toml", Name: "test", Desc: "pytest", Args: []string{"poe", "test"}},
			},
		},
		{
			name: "a standalone poe_tasks.toml lists tasks under tool.poe",
			files: map[string]string{
				"poe_tasks.toml": "[tool.poe.tasks]\ntest = \"pytest\"\n",
			},
			want: []Task{
				{Source: "poe_tasks.toml", Name: "test", Desc: "pytest", Args: []string{"poe", "test"}},
			},
		},
		{
			name: "pyproject.toml ignores a top-level tasks table",
			files: map[string]string{
				"pyproject.toml": `
[tasks]
ignored = "echo ignored"

[tool.poe.tasks]
kept = "echo kept"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "kept", Desc: "echo kept", Args: []string{"poe", "kept"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(osFiles{}, poeSource(), dir)
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

func TestDiscoverPoeTasksMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, poeSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestDiscoverPoeTasksSearchesParents: poe walks up to find its config, so the
// source lists the tasks of a pyproject.toml of a parent directory, badged
// with its relative path.
func TestDiscoverPoeTasksSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pyproject.toml", "[tool.poe.tasks]\ntest = \"pytest\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, err := discoverSource(osFiles{}, poeSource(), sub)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"poe", "test"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

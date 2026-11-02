package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverPdmScripts(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string scripts are listed and sorted",
			files: map[string]string{
				"pyproject.toml": `
[tool.pdm.scripts]
test = "pytest"
start = "flask run -p 54321"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "start", Desc: "flask run -p 54321", Args: []string{"pdm", "run", "start"}},
				{Source: "pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"pdm", "run", "test"}},
			},
		},
		{
			name: "scripts of every pdm shape are listed",
			files: map[string]string{
				"pyproject.toml": `
[tool.pdm.scripts]
start = {cmd = "flask run -p 54321", help = "Start the dev server"}
serve = {cmd = ["python", "-m", "http.server"]}
error-log = {shell = "cat error.log|grep CRITICAL > critical.log"}
main = {call = "foo_package.bar_module:main"}
dev = {call = "foo_package.bar_module:main('dev')"}
check = {composite = ["test -v tests/", "docs"]}
release = {composite = ["test", "build"], keep_going = true, help = "Test then build"}
test = "pytest"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "check", Desc: "test -v tests/ && docs", Args: []string{"pdm", "run", "check"}},
				{Source: "pyproject.toml", Name: "dev", Desc: "foo_package.bar_module:main('dev')", Args: []string{"pdm", "run", "dev"}},
				{Source: "pyproject.toml", Name: "error-log", Desc: "cat error.log|grep CRITICAL > critical.log", Args: []string{"pdm", "run", "error-log"}},
				{Source: "pyproject.toml", Name: "main", Desc: "foo_package.bar_module:main", Args: []string{"pdm", "run", "main"}},
				{Source: "pyproject.toml", Name: "release", Desc: "Test then build", Args: []string{"pdm", "run", "release"}},
				{Source: "pyproject.toml", Name: "serve", Desc: "python -m http.server", Args: []string{"pdm", "run", "serve"}},
				{Source: "pyproject.toml", Name: "start", Desc: "Start the dev server", Args: []string{"pdm", "run", "start"}},
				{Source: "pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"pdm", "run", "test"}},
			},
		},
		{
			name: "the shared-options key and underscore-prefixed scripts stay hidden",
			files: map[string]string{
				"pyproject.toml": `
[tool.pdm.scripts]
_.env_file = ".env"
_setup = "pip install -e ."
serve = "uvicorn app:app"

[tool.pdm.scripts._private]
cmd = "echo hidden"
help = "Never listed"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "serve", Desc: "uvicorn app:app", Args: []string{"pdm", "run", "serve"}},
			},
		},
		{
			name: "pre and post hooks stay listed, because they run standalone",
			files: map[string]string{
				"pyproject.toml": `
[tool.pdm.scripts]
pre_test = "rm -rf tests/__pycache__"
test = "pytest"
post_test = "coverage report"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "post_test", Desc: "coverage report", Args: []string{"pdm", "run", "post_test"}},
				{Source: "pyproject.toml", Name: "pre_test", Desc: "rm -rf tests/__pycache__", Args: []string{"pdm", "run", "pre_test"}},
				{Source: "pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"pdm", "run", "test"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"pyproject.toml": `
[tool.pdm.scripts]
good = "echo good"
number = 42
flag = true

[tool.pdm.scripts.no-type]
help = "A table without a script type"

[tool.pdm.scripts.wrong-cmd]
help = "The command is no string"
cmd = 42

[tool.pdm.scripts.wrong-cmd-array]
cmd = ["echo", 42]

[tool.pdm.scripts.wrong-shell]
shell = ["echo", "array"]

[tool.pdm.scripts.wrong-composite]
composite = "test"
`,
			},
			want: []Task{
				{Source: "pyproject.toml", Name: "good", Desc: "echo good", Args: []string{"pdm", "run", "good"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"pyproject.toml": "[tool.pdm.scripts\n"},
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
			name:  "no pdm table",
			files: map[string]string{"pyproject.toml": "[project]\nname = \"demo\"\n"},
			want:  nil,
		},
		{
			name:  "tool.pdm without scripts",
			files: map[string]string{"pyproject.toml": "[tool.pdm]\nresolution = \"highest\"\n"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(osFiles{}, pdmSource(), dir)
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

func TestDiscoverPdmScriptsMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, pdmSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestDiscoverPdmScriptsSearchesParents: pdm walks up to find the
// pyproject.toml, so the source lists the scripts of a pyproject.toml of a
// parent directory, badged with its relative path.
func TestDiscoverPdmScriptsSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pyproject.toml", "[tool.pdm.scripts]\ntest = \"pytest\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, err := discoverSource(osFiles{}, pdmSource(), sub)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../pyproject.toml", Name: "test", Desc: "pytest", Args: []string{"pdm", "run", "test"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

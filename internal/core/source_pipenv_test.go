package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverPipenvScripts(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string, array, call and cmd scripts are listed and sorted",
			files: map[string]string{
				"Pipfile": `
[scripts]
start = "python app.py"
test  = "pytest -x"
lint  = ["ruff check .", "ruff format --check ."]
check = {call = "mypackage.checks:run()"}
serve = {cmd = "gunicorn app:app"}
`,
			},
			want: []Task{
				{Source: "Pipfile", Name: "check", Desc: "mypackage.checks:run()", Args: []string{"pipenv", "run", "check"}},
				{Source: "Pipfile", Name: "lint", Desc: "ruff check . && ruff format --check .", Args: []string{"pipenv", "run", "lint"}},
				{Source: "Pipfile", Name: "serve", Desc: "gunicorn app:app", Args: []string{"pipenv", "run", "serve"}},
				{Source: "Pipfile", Name: "start", Desc: "python app.py", Args: []string{"pipenv", "run", "start"}},
				{Source: "Pipfile", Name: "test", Desc: "pytest -x", Args: []string{"pipenv", "run", "test"}},
			},
		},
		{
			name: "the other tables of a Pipfile contribute nothing",
			files: map[string]string{
				"Pipfile": `
[[source]]
url = "https://pypi.org/simple"
verify_ssl = true
name = "pypi"

[packages]
requests = "*"

[dev-packages]
pytest = "*"

[requires]
python_version = "3.13"

[scripts]
test = "pytest"
`,
			},
			want: []Task{
				{Source: "Pipfile", Name: "test", Desc: "pytest", Args: []string{"pipenv", "run", "test"}},
			},
		},
		{
			name: "a call table with more keys is skipped, like a table of another shape",
			files: map[string]string{
				"Pipfile": `
[scripts]
good = {call = "mypackage.checks:run()"}
with-cwd = {call = "mypackage.checks:run()", cwd = "."}
env-vars = {env = "FOO=bar"}
shell = {shell = "cat error.log"}
`,
			},
			want: []Task{
				{Source: "Pipfile", Name: "good", Desc: "mypackage.checks:run()", Args: []string{"pipenv", "run", "good"}},
			},
		},
		{
			name: "malformed entries are skipped and the file still lists",
			files: map[string]string{
				"Pipfile": `
[scripts]
good = "echo good"
number = 42
flag = true
mixed = ["echo one", 42]
call-number = {call = 42}
cmd-number = {cmd = 42}
`,
			},
			want: []Task{
				{Source: "Pipfile", Name: "good", Desc: "echo good", Args: []string{"pipenv", "run", "good"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"Pipfile": "[scripts\n"},
			wantErr: true,
		},
		{
			name:  "empty file",
			files: map[string]string{"Pipfile": ""},
			want:  nil,
		},
		{
			name:  "whitespace only file",
			files: map[string]string{"Pipfile": " \n\t "},
			want:  nil,
		},
		{
			name: "no scripts table",
			files: map[string]string{
				"Pipfile": `
[[source]]
url = "https://pypi.org/simple"
verify_ssl = true
name = "pypi"

[packages]
requests = "*"
`,
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

			got, warn := discoverSource(osFiles{}, pipenvSource(), dir)
			if tt.wantErr {
				if warn == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(warn.Error(), "Pipfile") {
					t.Errorf("error = %q, want it to mention Pipfile", warn)
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

func TestDiscoverPipenvScriptsMissingFile(t *testing.T) {
	got, warn := discoverSource(osFiles{}, pipenvSource(), t.TempDir())
	if warn != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, warn)
	}
}

// TestDiscoverPipenvScriptsSearchesParents: pipenv walks up to find the
// Pipfile, so the source lists the scripts of a Pipfile of a parent
// directory, badged with its relative path.
func TestDiscoverPipenvScriptsSearchesParents(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Pipfile", "[scripts]\ntest = \"pytest\"\n")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, warn := discoverSource(osFiles{}, pipenvSource(), sub)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{{Source: "../Pipfile", Name: "test", Desc: "pytest", Args: []string{"pipenv", "run", "test"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from a subdirectory = %+v, want %+v", tasks, want)
	}
}

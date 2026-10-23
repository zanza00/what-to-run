package tui

import (
	"reflect"
	"testing"
)

// TestParseJustfile covers the shapes a justfile can declare: doc comments and
// attributes, private and silent recipes, body lines that look like headers,
// and the statements that are not recipes at all.
func TestParseJustfile(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Task
	}{
		{
			name: "settings, assignments, aliases, exports, imports and modules are not recipes",
			in: `set dotenv-load
set shell := ["bash", "-c"]
alias b := build
version := "1.0"
export FOO := "bar"
import "shared.just"
import? "optional:shared.just"
mod sub
mod? "sub:dir"

build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a doc comment uses only its last contiguous line",
			in: `# First line
# Second line
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "Second line", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a blank line breaks the doc comment of the recipe below",
			in: `# Build the app

build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "the doc attribute wins over the comment",
			in: `# A comment
[doc("Run the tests")]
test:
    echo test
`,
			want: []Task{
				{Name: "test", Desc: "Run the tests", Args: []string{"just", "test"}},
			},
		},
		{
			name: "a single-quoted doc attribute is read too",
			in: `[doc('Single quoted doc')]
test:
    echo test
`,
			want: []Task{
				{Name: "test", Desc: "Single quoted doc", Args: []string{"just", "test"}},
			},
		},
		{
			name: "an attribute between the comment and the recipe keeps the comment",
			in: `# Build the app
[group("build")]
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a private recipe is hidden",
			in: `[private]
secret:
    echo secret
`,
			want: nil,
		},
		{
			name: "a recipe with an underscore prefix is hidden",
			in: `_hidden:
    echo hidden

visible:
    echo visible
`,
			want: []Task{
				{Name: "visible", Desc: "", Args: []string{"just", "visible"}},
			},
		},
		{
			name: "a silent recipe is listed without the at sign",
			in: `@silent-recipe:
    echo silent
`,
			want: []Task{
				{Name: "silent-recipe", Desc: "", Args: []string{"just", "silent-recipe"}},
			},
		},
		{
			name: "a top-of-file shebang is not a description",
			in: `#!/usr/bin/env just --justfile
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "parameters and a dependency do not reach the name",
			in: `build target="debug": other
    echo {{target}}

other:
    echo other
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
				{Name: "other", Desc: "", Args: []string{"just", "other"}},
			},
		},
		{
			name: "a shebang recipe body and an indented fake header are ignored",
			in: `script:
    #!/usr/bin/env bash
    set -euo pipefail
    fake:
`,
			want: []Task{
				{Name: "script", Desc: "", Args: []string{"just", "script"}},
			},
		},
		{
			name: "a hyphenated recipe name is kept whole",
			in: `hyphen-name:
    echo hyphen
`,
			want: []Task{
				{Name: "hyphen-name", Desc: "", Args: []string{"just", "hyphen-name"}},
			},
		},
		{
			name: "a doc attribute does not leak onto the next recipe",
			in: `[doc("Only the first")]
first:
    echo first
second:
    echo second
`,
			want: []Task{
				{Name: "first", Desc: "Only the first", Args: []string{"just", "first"}},
				{Name: "second", Desc: "", Args: []string{"just", "second"}},
			},
		},
		{
			name: "a private recipe does not hide the next recipe",
			in: `[private]
secret:
    echo secret
visible:
    echo visible
`,
			want: []Task{
				{Name: "visible", Desc: "", Args: []string{"just", "visible"}},
			},
		},
		{
			name: "a comment does not leak past the recipe it describes",
			in: `# Build the app
build:
    echo build
test:
    echo test
`,
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
				{Name: "test", Desc: "", Args: []string{"just", "test"}},
			},
		},
		{
			name: "groups are flattened into one alphabetical list",
			in: `[group("zzz")]
zebra:
    echo z

[group("aaa")]
alpha:
    echo a

omega:
    echo o
`,
			want: []Task{
				{Name: "alpha", Desc: "", Args: []string{"just", "alpha"}},
				{Name: "omega", Desc: "", Args: []string{"just", "omega"}},
				{Name: "zebra", Desc: "", Args: []string{"just", "zebra"}},
			},
		},
		{
			name: "duplicate recipe names keep the first",
			in: `build:
    echo one

build:
    echo two
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a line that is not a recipe header is skipped",
			in: `this is not a recipe

build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "CRLF line endings are normalized",
			in:   "# Build the app\r\nbuild:\r\n    echo build\r\n",
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
			},
		},
		{
			name: "the fixture of the picker keeps build and test only",
			in: `# Build the app
build:
    echo build

[doc("Run the tests")]
test:
    echo test

[private]
secret:
    echo secret

@quiet:
    echo quiet

[group("deploy")]
deploy target="prod":
    echo deploy
`,
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
				{Name: "deploy", Desc: "", Args: []string{"just", "deploy"}},
				{Name: "quiet", Desc: "", Args: []string{"just", "quiet"}},
				{Name: "test", Desc: "Run the tests", Args: []string{"just", "test"}},
			},
		},
		{
			name: "empty file declares no recipes",
			in:   "",
			want: nil,
		},
		{
			name: "whitespace-only file declares no recipes",
			in:   " \n\t\n",
			want: nil,
		},
		{
			name: "a settings-only file declares no recipes",
			in: `set dotenv-load
version := "1.0"
export FOO := "bar"
`,
			want: nil,
		},
		{
			name: "a shebang-only file declares no recipes",
			in:   "#!/usr/bin/env just --justfile\n",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJustfile("", []byte(tt.in))
			if err != nil {
				t.Fatalf("parseJustfile() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfile() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseJustfileNeverErrors pins the forgiving contract: an odd or broken
// justfile produces no tasks instead of a warning that would hide the file's
// siblings.
func TestParseJustfileNeverErrors(t *testing.T) {
	for _, in := range []string{
		"this is not a justfile %%%\n    indented?\n",
		"build\n",
		"{{{{ unbalanced\n",
		"set shell := [\n",
	} {
		tasks, err := parseJustfile("", []byte(in))
		if err != nil {
			t.Errorf("parseJustfile(%q) error = %v, want nil", in, err)
		}
		if tasks != nil {
			t.Errorf("parseJustfile(%q) = %+v, want no tasks", in, tasks)
		}
	}
}

// TestJustfileSourceFiles pins the three names just looks for, in preference
// order.
func TestJustfileSourceFiles(t *testing.T) {
	want := []string{"justfile", "Justfile", ".justfile"}
	if got := justfileSource().Files; !reflect.DeepEqual(got, want) {
		t.Errorf("justfileSource().Files = %v, want %v", got, want)
	}
}

// TestDiscoverJustfileFirstFileWins pins the candidate-file preference:
// justfile beats .justfile. The names differ by a dot, not by case, so the
// fixture is safe on the case-insensitive filesystem of macOS.
func TestDiscoverJustfileFirstFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".justfile", "dot-only:\n    echo dot\n")
	writeFile(t, dir, "justfile", "local-only:\n    echo local\n")

	tasks, err := discoverSource(justfileSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "justfile", Name: "local-only", Desc: "", Args: []string{"just", "local-only"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverJustfileDotFallback: .justfile is read when there is no
// higher-priority candidate, and its name is the badge.
func TestDiscoverJustfileDotFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".justfile", "# Build the app\nbuild:\n    echo build\n")

	tasks, err := discoverSource(justfileSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: ".justfile", Name: "build", Desc: "Build the app", Args: []string{"just", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

func TestDiscoverJustfileMissingFile(t *testing.T) {
	got, err := discoverSource(justfileSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestDiscoverTasksJustfileOrder: rc tasks first, then package.json,
// Taskfile.yml and justfile, in the registry order.
func TestDiscoverTasksJustfileOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts": {"dev": "vite"}}`)
	writeFile(t, dir, "Taskfile.yml", "version: '3'\ntasks:\n  build:\n    desc: Build the app\n")
	writeFile(t, dir, "justfile", "test:\n    echo test\n")

	cfg := config{Tasks: []configTask{{Name: "deploy", Command: "./deploy.sh"}}}
	tasks, errs := discoverTasks(dir, cfg)
	if len(errs) != 0 {
		t.Fatalf("discoverTasks() errors = %v, want none", errs)
	}

	var got []string
	for _, task := range tasks {
		got = append(got, task.Source+"/"+task.Name)
	}
	want := []string{"rc/deploy", "package.json/dev", "Taskfile.yml/build", "justfile/test"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
}

// TestDiscoverTasksOddJustfileKeepsSiblings: a justfile the parser cannot make
// sense of contributes no recipes and no warning, and does not hide a
// package.json in the same directory.
func TestDiscoverTasksOddJustfileKeepsSiblings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "justfile", "this is not a justfile %%%\n    indented?\n")
	writeFile(t, dir, "package.json", `{"scripts": {"dev": "vite"}}`)

	tasks, errs := discoverTasks(dir, config{})
	if len(errs) != 0 {
		t.Fatalf("discoverTasks() errors = %v, want none", errs)
	}
	want := []Task{{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"npm", "run", "dev"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverTasks() = %+v, want %+v", tasks, want)
	}
}

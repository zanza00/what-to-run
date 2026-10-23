package core

import (
	"os"
	"path/filepath"
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
			name: "a tab after a keyword is still a setting",
			in: "set\tshell := [\"bash\", \"-c\"]\n" +
				"alias\tb := build\n" +
				"export\tFOO := \"bar\"\n" +
				"import\t\"x:y.just\"\n" +
				"import?\t\"opt:y.just\"\n" +
				"mod\tsub\n" +
				"mod?\t\"sub:dir\"\n" +
				"\nbuild:\n    echo build\n",
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a recipe named after a keyword is still a recipe",
			in: `set:
    echo set

import: build
    echo import
`,
			want: []Task{
				{Name: "import", Desc: "", Args: []string{"just", "import"}},
				{Name: "set", Desc: "", Args: []string{"just", "set"}},
			},
		},
		{
			name: "a recipe named after a keyword may take parameters",
			in: `set x="1":
    echo set

alias y="2":
    echo alias

export z="3":
    echo export

import q="4":
    echo import
`,
			want: []Task{
				{Name: "alias", Desc: "", Args: []string{"just", "alias"}},
				{Name: "export", Desc: "", Args: []string{"just", "export"}},
				{Name: "import", Desc: "", Args: []string{"just", "import"}},
				{Name: "set", Desc: "", Args: []string{"just", "set"}},
			},
		},
		{
			name: "a quoted colon and a trailing comment keep a keyword line a statement",
			in: `set shell := ["bash", "-c"] # see a:b
export PATH := "/a:b/bin"
import "x:y.just" # : not a header
mod? sub
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
			name: "an empty doc attribute means no description, not the comment",
			in: `# A comment
[doc("")]
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a whitespace-only doc attribute means no description",
			in: `# A comment
[doc("   ")]
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "an empty doc attribute does not leak onto the next recipe",
			in: `# A comment
[doc("")]
build:
    echo build
# Test the build
test:
    echo test
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
				{Name: "test", Desc: "Test the build", Args: []string{"just", "test"}},
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
			name: "a doc attribute with a trailing comment keeps its doc",
			in: `[doc("the doc")] # note
r:
    echo r
`,
			want: []Task{
				{Name: "r", Desc: "the doc", Args: []string{"just", "r"}},
			},
		},
		{
			name: "escaped quotes in a doc attribute are unquoted",
			in: `[doc("say \"hi\"")]
say:
    echo say

[doc('it\'s fine')]
apostrophe:
    echo apostrophe
`,
			want: []Task{
				{Name: "apostrophe", Desc: "it's fine", Args: []string{"just", "apostrophe"}},
				{Name: "say", Desc: `say "hi"`, Args: []string{"just", "say"}},
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
			name: "a private attribute with a trailing comment hides the recipe",
			in: `[private] # why
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
			name: "a private attribute in a comma list hides the recipe",
			in: `[private, group("g")]
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
			name: "a private attribute split across two lines hides the recipe",
			in: `[private,
 group("g")]
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
			name: "a multi-line doc attribute hides its recipe and does not leak the doc",
			in: `[doc("multi-line doc"),
 private]
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
			name: "an attribute list split across three lines lists the recipe with its doc",
			in: `[doc("x"),
 group("g"),
 no-cd]
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "x", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a blank line inside an attribute list keeps the list and the comment above it",
			in: `# doc
[group("a"),

 group("b")]
build:
    echo b
`,
			want: []Task{
				{Name: "build", Desc: "doc", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a closing bracket inside a doc string does not close the attribute list",
			in: `[doc("a ] b"),
 private]
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
			name: "a multi-line private recipe does not hide the next recipe without a blank line",
			in: `[private,
 group("g")]
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
			name: "an unclosed attribute at the end of the file is ignored",
			in: `build:
    echo build

[private,
 group("g")
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
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
			name: "a recipe header with a trailing comment containing a colon-equals is a recipe",
			in: `build: # see FOO:=1
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a recipe parameter default containing a colon-equals is a recipe",
			in: `build args="a:=b":
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a top-level assignment is still skipped",
			in: `version := "1.0"

build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
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
			name: "a continued recipe header is listed under its first word",
			in: `foo \
  bar:
    echo foo
`,
			want: []Task{
				{Name: "foo", Desc: "", Args: []string{"just", "foo"}},
			},
		},
		{
			name: "two continuations join into one recipe header",
			in: `foo \
  bar \
  baz:
    echo foo
`,
			want: []Task{
				{Name: "foo", Desc: "", Args: []string{"just", "foo"}},
			},
		},
		{
			name: "a continued header keeps its parameters and dependencies",
			in: `foo \
  bar="1": dep \
  other
    echo foo

dep:
    echo dep

other:
    echo other
`,
			want: []Task{
				{Name: "dep", Desc: "", Args: []string{"just", "dep"}},
				{Name: "foo", Desc: "", Args: []string{"just", "foo"}},
				{Name: "other", Desc: "", Args: []string{"just", "other"}},
			},
		},
		{
			name: "a continued header keeps the doc comment and the attributes above it",
			in: `# Build the app
foo \
  bar:
    echo foo

[private]
secret \
  name:
    echo secret

visible:
    echo visible
`,
			want: []Task{
				{Name: "foo", Desc: "Build the app", Args: []string{"just", "foo"}},
				{Name: "visible", Desc: "", Args: []string{"just", "visible"}},
			},
		},
		{
			name: "a continued comment is not continued and keeps its recipe",
			in: `# Build the app \
foo \
  bar:
    echo foo
`,
			want: []Task{
				{Name: "foo", Desc: `Build the app \`, Args: []string{"just", "foo"}},
			},
		},
		{
			name: "a header inline comment ending with a backslash keeps the next recipe",
			in: `build: # note \
test:
    echo test
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
				{Name: "test", Desc: "", Args: []string{"just", "test"}},
			},
		},
		{
			name: "a trailing backslash inside a quoted parameter default does not continue",
			in: `build args="a \
other:
    echo other
`,
			want: []Task{
				{Name: "other", Desc: "", Args: []string{"just", "other"}},
			},
		},
		{
			name: "a continued setting or assignment is still skipped",
			in: `version := "1.0" + \
  "-dev"
build:
    echo build
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
			},
		},
		{
			name: "a body line ending with a backslash does not consume the next recipe",
			in: `build:
    echo one \
test:
    echo test
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
				{Name: "test", Desc: "", Args: []string{"just", "test"}},
			},
		},
		{
			name: "an unclosed continuation at the end of the file is ignored",
			in: `build:
    echo build

foo \`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"just", "build"}},
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

// TestParseJustfilePlatforms pins the platform attributes: each one lists its
// recipe only on a matching GOOS, several attributes are OR-ed whether they
// share a list or not, and unix covers every GOOS but Windows.
func TestParseJustfilePlatforms(t *testing.T) {
	in := `[windows]
win:
    echo win

[linux]
lin:
    echo lin

[macos]
mac:
    echo mac

[unix]
uni:
    echo uni

[linux, macos]
nix:
    echo nix

[windows, linux]
both:
    echo both

[windows]
[linux]
split:
    echo split
`
	tests := []struct {
		goos string
		want []string
	}{
		{goos: "linux", want: []string{"both", "lin", "nix", "split", "uni"}},
		{goos: "darwin", want: []string{"mac", "nix", "uni"}},
		{goos: "windows", want: []string{"both", "split", "win"}},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			tasks, err := parseJustfileFor(tt.goos, "", []byte(in))
			if err != nil {
				t.Fatalf("parseJustfileFor(%q) error = %v", tt.goos, err)
			}
			if got := justTaskNames(tasks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfileFor(%q) names = %v, want %v", tt.goos, got, tt.want)
			}
		})
	}
}

// TestParseJustfilePlatformVariants reads the platform fixture from disk: a
// recipe defined once per platform must produce one row, carrying the doc and
// description of the definition enabled on the target GOOS.
func TestParseJustfilePlatformVariants(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "justfile-platforms", "justfile"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		goos string
		want []Task
	}{
		{
			goos: "linux",
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
				{Name: "serve", Desc: "Build the server (Unix)", Args: []string{"just", "serve"}},
				{Name: "test", Desc: "Run the tests on Linux", Args: []string{"just", "test"}},
			},
		},
		{
			goos: "darwin",
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
				{Name: "serve", Desc: "Build the server (Unix)", Args: []string{"just", "serve"}},
				{Name: "test", Desc: "Run the tests on macOS", Args: []string{"just", "test"}},
			},
		},
		{
			goos: "windows",
			want: []Task{
				{Name: "build", Desc: "Build the app", Args: []string{"just", "build"}},
				{Name: "serve", Desc: "Build the server (Windows)", Args: []string{"just", "serve"}},
				{Name: "test", Desc: "Run the tests on Windows", Args: []string{"just", "test"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			tasks, err := parseJustfileFor(tt.goos, "", data)
			if err != nil {
				t.Fatalf("parseJustfileFor(%q) error = %v", tt.goos, err)
			}
			if !reflect.DeepEqual(tasks, tt.want) {
				t.Errorf("parseJustfileFor(%q) = %+v, want %+v", tt.goos, tasks, tt.want)
			}
		})
	}
}

// TestParseJustfilePlatformPrivate: privacy belongs to the selected variant.
// A private definition hides the name only on its platforms, and a public
// definition of the same name stays visible elsewhere.
func TestParseJustfilePlatformPrivate(t *testing.T) {
	in := `[linux, private]
[doc("Private Linux build")]
build:
    echo private-linux

[windows]
[doc("Public Windows build")]
build:
    echo public-windows

[unix]
[doc("Public Unix deploy")]
deploy:
    echo public-unix

[windows, private]
[doc("Private Windows deploy")]
deploy:
    echo private-windows
`
	tests := []struct {
		goos string
		want []Task
	}{
		{
			goos: "linux",
			want: []Task{
				{Name: "deploy", Desc: "Public Unix deploy", Args: []string{"just", "deploy"}},
			},
		},
		{
			goos: "darwin",
			want: []Task{
				{Name: "deploy", Desc: "Public Unix deploy", Args: []string{"just", "deploy"}},
			},
		},
		{
			goos: "windows",
			want: []Task{
				{Name: "build", Desc: "Public Windows build", Args: []string{"just", "build"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			tasks, err := parseJustfileFor(tt.goos, "", []byte(in))
			if err != nil {
				t.Fatalf("parseJustfileFor(%q) error = %v", tt.goos, err)
			}
			if !reflect.DeepEqual(tasks, tt.want) {
				t.Errorf("parseJustfileFor(%q) = %+v, want %+v", tt.goos, tasks, tt.want)
			}
		})
	}
}

// justTaskNames returns the names of the parsed tasks in order, for tests that
// only care about which recipes are visible.
func justTaskNames(tasks []Task) []string {
	names := make([]string, 0, len(tasks))
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	return names
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

	tasks, err := discoverSource(osFiles{}, justfileSource(), dir)
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

	tasks, err := discoverSource(osFiles{}, justfileSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: ".justfile", Name: "build", Desc: "Build the app", Args: []string{"just", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

func TestDiscoverJustfileMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, justfileSource(), t.TempDir())
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

	settings := Settings{Tasks: []RCTask{{Name: "deploy", Command: "./deploy.sh"}}}
	tasks, errs := Discover(osFiles{}, dir, settings)
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
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

	tasks, errs := Discover(osFiles{}, dir, Settings{})
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}
	want := []Task{{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"npm", "run", "dev"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("Discover() = %+v, want %+v", tasks, want)
	}
}

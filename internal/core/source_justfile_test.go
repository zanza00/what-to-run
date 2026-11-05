package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseJustfile covers the shapes that a justfile can declare: doc
// comments and attributes, private and silent recipes, body lines that look
// like headers, and the statements that are not recipes at all.
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
				justTask("build", ""),
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
				justTask("build", ""),
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
				justTask("import", ""),
				justTask("set", ""),
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
				justTask("alias", ""),
				justTask("export", ""),
				justTask("import", ""),
				justTask("set", ""),
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
				justTask("build", ""),
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
				justTask("build", "Second line"),
			},
		},
		{
			name: "a blank line breaks the doc comment of the recipe below",
			in: `# Build the app

build:
    echo build
`,
			want: []Task{
				justTask("build", ""),
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
				justTask("test", "Run the tests"),
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
				justTask("build", ""),
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
				justTask("build", ""),
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
				justTask("build", ""),
				justTask("test", "Test the build"),
			},
		},
		{
			name: "a single-quoted doc attribute is read too",
			in: `[doc('Single quoted doc')]
test:
    echo test
`,
			want: []Task{
				justTask("test", "Single quoted doc"),
			},
		},
		{
			name: "a doc attribute with a trailing comment keeps its doc",
			in: `[doc("the doc")] # note
r:
    echo r
`,
			want: []Task{
				justTask("r", "the doc"),
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
				justTask("apostrophe", "it's fine"),
				justTask("say", `say "hi"`),
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
				justTask("build", "Build the app"),
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
				justTask("visible", ""),
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
				justTask("visible", ""),
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
				justTask("visible", ""),
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
				justTask("visible", ""),
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
				justTask("build", "x"),
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
				justTask("build", "doc"),
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
				justTask("visible", ""),
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
				justTask("visible", ""),
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
				justTask("build", ""),
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
				justTask("visible", ""),
			},
		},
		{
			name: "a silent recipe is listed without the at sign",
			in: `@silent-recipe:
    echo silent
`,
			want: []Task{
				justTask("silent-recipe", ""),
			},
		},
		{
			name: "a top-of-file shebang is not a description",
			in: `#!/usr/bin/env just --justfile
build:
    echo build
`,
			want: []Task{
				justTask("build", ""),
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
				justTask("build", ""),
				justTask("other", ""),
			},
		},
		{
			name: "a recipe header with a trailing comment containing a colon-equals is a recipe",
			in: `build: # see FOO:=1
    echo build
`,
			want: []Task{
				justTask("build", ""),
			},
		},
		{
			name: "a recipe parameter default containing a colon-equals is a recipe",
			in: `build args="a:=b":
    echo build
`,
			want: []Task{
				justTask("build", ""),
			},
		},
		{
			name: "a top-level assignment is still skipped",
			in: `version := "1.0"

build:
    echo build
`,
			want: []Task{
				justTask("build", ""),
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
				justTask("script", ""),
			},
		},
		{
			name: "a hyphenated recipe name is kept whole",
			in: `hyphen-name:
    echo hyphen
`,
			want: []Task{
				justTask("hyphen-name", ""),
			},
		},
		{
			name: "a continued recipe header is listed under its first word",
			in: `foo \
  bar:
    echo foo
`,
			want: []Task{
				justTask("foo", ""),
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
				justTask("foo", ""),
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
				justTask("dep", ""),
				justTask("foo", ""),
				justTask("other", ""),
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
				justTask("foo", "Build the app"),
				justTask("visible", ""),
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
				justTask("foo", `Build the app \`),
			},
		},
		{
			name: "a header inline comment ending with a backslash keeps the next recipe",
			in: `build: # note \
test:
    echo test
`,
			want: []Task{
				justTask("build", ""),
				justTask("test", ""),
			},
		},
		{
			name: "a trailing backslash inside a quoted parameter default does not continue",
			in: `build args="a \
other:
    echo other
`,
			want: []Task{
				justTask("other", ""),
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
				justTask("build", ""),
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
				justTask("build", ""),
				justTask("test", ""),
			},
		},
		{
			name: "an unclosed continuation at the end of the file is ignored",
			in: `build:
    echo build

foo \`,
			want: []Task{
				justTask("build", ""),
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
				justTask("first", "Only the first"),
				justTask("second", ""),
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
				justTask("visible", ""),
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
				justTask("build", "Build the app"),
				justTask("test", ""),
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
				justTask("alpha", ""),
				justTask("omega", ""),
				justTask("zebra", ""),
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
				justTask("build", ""),
			},
		},
		{
			name: "a line that is not a recipe header is skipped",
			in: `this is not a recipe

build:
    echo build
`,
			want: []Task{
				justTask("build", ""),
			},
		},
		{
			name: "CRLF line endings are normalized",
			in:   "# Build the app\r\nbuild:\r\n    echo build\r\n",
			want: []Task{
				justTask("build", "Build the app"),
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
				justTask("build", "Build the app"),
				justTask("deploy", ""),
				justTask("quiet", ""),
				justTask("test", "Run the tests"),
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
			got := parseJustfileFor("linux", nil, "", []byte(tt.in))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfileFor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseJustfileNeverFails pins the forgiving contract. An odd or broken
// justfile produces no tasks, instead of a warning that would hide the sibling
// files. The signature carries no error, so the parser can only answer with
// its task list, and the pin holds the parser to the empty one.
func TestParseJustfileNeverFails(t *testing.T) {
	for _, in := range []string{
		"this is not a justfile %%%\n    indented?\n",
		"build\n",
		"{{{{ unbalanced\n",
		"set shell := [\n",
	} {
		tasks := parseJustfileFor("linux", nil, "", []byte(in))
		if tasks != nil {
			t.Errorf("parseJustfileFor(%q) = %+v, want no tasks", in, tasks)
		}
	}
}

// TestParseJustfilePlatforms pins the platform attributes. Each attribute
// lists its recipe only on a matching GOOS. Several attributes are OR-ed,
// whether they share a list or not, and unix covers every GOOS but Windows.
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
			tasks := parseJustfileFor(tt.goos, nil, "", []byte(in))
			if got := taskNames(tasks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfileFor(%q) names = %v, want %v", tt.goos, got, tt.want)
			}
		})
	}
}

// TestParseJustfilePlatformVariants reads the platform fixture from disk. A
// recipe that is defined once for each platform must produce one row, and that
// row carries the doc and the description of the definition that is enabled on
// the target GOOS.
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
				justTask("build", "Build the app"),
				justTask("serve", "Build the server (Unix)"),
				justTask("test", "Run the tests on Linux"),
			},
		},
		{
			goos: "darwin",
			want: []Task{
				justTask("build", "Build the app"),
				justTask("serve", "Build the server (Unix)"),
				justTask("test", "Run the tests on macOS"),
			},
		},
		{
			goos: "windows",
			want: []Task{
				justTask("build", "Build the app"),
				justTask("serve", "Build the server (Windows)"),
				justTask("test", "Run the tests on Windows"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			tasks := parseJustfileFor(tt.goos, nil, "", data)
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
				justTask("deploy", "Public Unix deploy"),
			},
		},
		{
			goos: "darwin",
			want: []Task{
				justTask("deploy", "Public Unix deploy"),
			},
		},
		{
			goos: "windows",
			want: []Task{
				justTask("build", "Public Windows build"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			tasks := parseJustfileFor(tt.goos, nil, "", []byte(in))
			if !reflect.DeepEqual(tasks, tt.want) {
				t.Errorf("parseJustfileFor(%q) = %+v, want %+v", tt.goos, tasks, tt.want)
			}
		})
	}
}

// parseJustfileTree parses main as the root justfile in root. The tree holds
// the justfiles that the root may import, keyed by their path relative to
// root.
func parseJustfileTree(t *testing.T, goos, root, main string, tree map[string]string) []Task {
	t.Helper()
	files := make(mapFiles, len(tree))
	for name, content := range tree {
		files[filepath.Join(root, name)] = content
	}
	tasks := parseJustfileFor(goos, files, root, []byte(main))
	return tasks
}

// TestParseJustfileImports covers how local imports join the list. The recipes
// of an import keep their own descriptions, and a relative path resolves
// against the importing file. Missing imports are skipped. The duplicate cases
// pin the precedence rule: the shallowest file wins, and at equal depth the
// earlier import wins.
func TestParseJustfileImports(t *testing.T) {
	tests := []struct {
		name string
		main string
		tree map[string]string
		want []Task
	}{
		{
			name: "an import adds its recipes with their descriptions",
			main: "import 'lib/shared.just' # shared helpers\n\nroot:\n    echo root\n",
			tree: map[string]string{
				"lib/shared.just": "# Shared helper\nshared:\n    echo shared\n",
			},
			want: []Task{
				justTask("root", ""),
				justTask("shared", "Shared helper"),
			},
		},
		{
			name: "nested imports resolve against the importing file",
			main: "import \"lib/a.just\"\n",
			tree: map[string]string{
				"lib/a.just":        "import \"nested/b.just\"\n\nfrom-a:\n    echo a\n",
				"lib/nested/b.just": "from-b:\n    echo b\n",
			},
			want: []Task{
				justTask("from-a", ""),
				justTask("from-b", ""),
			},
		},
		{
			name: "import? loads the file when it exists",
			main: "import? \"shared.just\"\n",
			tree: map[string]string{
				"shared.just": "shared:\n    echo shared\n",
			},
			want: []Task{justTask("shared", "")},
		},
		{
			name: "missing imports are skipped, optional or not",
			main: "import \"missing.just\"\nimport? \"also-missing.just\"\n\nroot:\n    echo root\n",
			want: []Task{justTask("root", "")},
		},
		{
			name: "the root definition wins over an imported one",
			main: "import \"lib/shared.just\"\n\nbuild:\n    echo root\n",
			tree: map[string]string{
				"lib/shared.just": "# Imported build\nbuild:\n    echo shared\n",
			},
			want: []Task{justTask("build", "")},
		},
		{
			name: "the shallowest import wins even when a deeper one is read first",
			main: "import \"lib/a.just\"\nimport \"deep.just\"\n",
			tree: map[string]string{
				"lib/a.just":           "import \"nested/deep.just\"\n\nfrom-a:\n    echo a\n",
				"lib/nested/deep.just": "# Deep build\nbuild:\n    echo deep\n",
				"deep.just":            "# Direct build\nbuild:\n    echo direct\n",
			},
			want: []Task{
				justTask("build", "Direct build"),
				justTask("from-a", ""),
			},
		},
		{
			name: "equal depth keeps the earlier import",
			main: "import \"a.just\"\nimport \"b.just\"\n",
			tree: map[string]string{
				"a.just": "# From a\ndeploy:\n    echo a\n",
				"b.just": "# From b\ndeploy:\n    echo b\n",
			},
			want: []Task{justTask("deploy", "From a")},
		},
		{
			name: "a diamond is not a cycle",
			main: "import \"a.just\"\nimport \"b.just\"\n",
			tree: map[string]string{
				"a.just":      "import \"shared.just\"\nfrom-a:\n    echo a\n",
				"b.just":      "import \"shared.just\"\nfrom-b:\n    echo b\n",
				"shared.just": "shared:\n    echo shared\n",
			},
			want: []Task{
				justTask("from-a", ""),
				justTask("from-b", ""),
				justTask("shared", ""),
			},
		},
		{
			name: "a cycle terminates",
			main: "import \"a.just\"\n",
			tree: map[string]string{
				"a.just": "import \"b.just\"\nfrom-a:\n    echo a\n",
				"b.just": "import \"a.just\"\nfrom-b:\n    echo b\n",
			},
			want: []Task{
				justTask("from-a", ""),
				justTask("from-b", ""),
			},
		},
		{
			name: "an imported platform variant is filtered",
			main: "import \"lib/platforms.just\"\n",
			tree: map[string]string{
				"lib/platforms.just": "[windows]\nwin:\n    echo win\n\n[linux]\nlin:\n    echo lin\n",
			},
			want: []Task{justTask("lin", "")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJustfileTree(t, "linux", t.TempDir(), tt.main, tt.tree)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfileFor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseJustfileModules covers how modules join the list. The recipes of a
// module are namespaced with "name::", and the source is looked up where just
// looks for it. Nested modules chain the namespace.
func TestParseJustfileModules(t *testing.T) {
	tests := []struct {
		name string
		main string
		tree map[string]string
		want []Task
	}{
		{
			name: "mod name loads name.just",
			main: "mod tools\n\nroot:\n    echo root\n",
			tree: map[string]string{
				"tools.just": "# Format the code\nfmt:\n    echo fmt\n",
			},
			want: []Task{
				justTask("root", ""),
				justTask("tools::fmt", "Format the code"),
			},
		},
		{
			name: "mod name loads name/mod.just",
			main: "mod tools\n",
			tree: map[string]string{
				"tools/mod.just": "fmt:\n    echo fmt\n",
			},
			want: []Task{justTask("tools::fmt", "")},
		},
		{
			name: "mod name loads name/justfile in any capitalization",
			main: "mod tools\n",
			tree: map[string]string{
				"tools/JustFile": "fmt:\n    echo fmt\n",
			},
			want: []Task{justTask("tools::fmt", "")},
		},
		{
			name: "an explicit path reads a file",
			main: "mod tools \"lib/tools.just\"\n",
			tree: map[string]string{
				"lib/tools.just": "fmt:\n    echo fmt\n",
			},
			want: []Task{justTask("tools::fmt", "")},
		},
		{
			name: "an explicit path reads a directory",
			main: "mod tools \"lib/tools\"\n",
			tree: map[string]string{
				"lib/tools/mod.just": "fmt:\n    echo fmt\n",
			},
			want: []Task{justTask("tools::fmt", "")},
		},
		{
			name: "missing modules are skipped, optional or not",
			main: "mod missing\nmod? also-missing\n\nroot:\n    echo root\n",
			want: []Task{justTask("root", "")},
		},
		{
			name: "nested modules chain the namespace",
			main: "mod outer\n",
			tree: map[string]string{
				"outer/mod.just":   "mod inner\n\nouter-task:\n    echo outer\n",
				"outer/inner.just": "inner-task:\n    echo inner\n",
			},
			want: []Task{
				justTask("outer::inner::inner-task", ""),
				justTask("outer::outer-task", ""),
			},
		},
		{
			name: "an import inside a module keeps the module namespace",
			main: "mod tools \"lib/mod.just\"\n",
			tree: map[string]string{
				"lib/mod.just":    "import \"shared.just\"\n",
				"lib/shared.just": "shared:\n    echo shared\n",
			},
			want: []Task{justTask("tools::shared", "")},
		},
		{
			name: "private recipes stay hidden inside a module",
			main: "mod tools\n",
			tree: map[string]string{
				"tools.just": "_hidden:\n    echo hidden\n\nshown:\n    echo shown\n",
			},
			want: []Task{justTask("tools::shown", "")},
		},
		{
			name: "a module cycle terminates",
			main: "mod a\n",
			tree: map[string]string{
				"a.just": "mod b\n\nfrom-a:\n    echo a\n",
				"b.just": "mod a\n\nfrom-b:\n    echo b\n",
			},
			want: []Task{
				justTask("a::b::from-b", ""),
				justTask("a::from-a", ""),
			},
		},
		{
			name: "platform attributes filter module recipes too",
			main: "mod tools\n",
			tree: map[string]string{
				"tools.just": "[windows]\nwin:\n    echo win\n\n[linux]\nlin:\n    echo lin\n",
			},
			want: []Task{justTask("tools::lin", "")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJustfileTree(t, "linux", t.TempDir(), tt.main, tt.tree)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJustfileFor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseJustfileImportAbsolute: the parser reads an absolute import path as
// written.
func TestParseJustfileImportAbsolute(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "abs.just")
	files := mapFiles{abs: "from-abs:\n    echo abs\n"}

	tasks := parseJustfileFor("linux", files, t.TempDir(), []byte("import \""+abs+"\"\n"))
	want := []Task{justTask("from-abs", "")}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("parseJustfileFor() = %+v, want %+v", tasks, want)
	}
}

// TestParseJustfileImportHome: the parser expands a leading "~/" to the home
// directory, as it does in just.
func TestParseJustfileImportHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	files := mapFiles{filepath.Join(home, "home.just"): "from-home:\n    echo home\n"}

	tasks := parseJustfileFor("linux", files, t.TempDir(), []byte("import \"~/home.just\"\n"))
	want := []Task{justTask("from-home", "")}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("parseJustfileFor() = %+v, want %+v", tasks, want)
	}
}

// TestJustfileSourceFiles pins the three names just looks for, in preference
// order.
func TestJustfileSourceFiles(t *testing.T) {
	want := []string{"justfile", "Justfile", ".justfile"}
	if got := justfileSource(osFiles{}).Files; !reflect.DeepEqual(got, want) {
		t.Errorf("justfileSource(osFiles{}).Files = %v, want %v", got, want)
	}
}

// TestDiscoverJustfileFirstFileWins pins the preference between candidate
// files: justfile wins over .justfile. The names differ by a dot, not by case,
// so the fixture is safe on the case-insensitive filesystem of macOS.
func TestDiscoverJustfileFirstFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".justfile", "dot-only:\n    echo dot\n")
	writeFile(t, dir, "justfile", "local-only:\n    echo local\n")

	tasks, warn := discoverSource(osFiles{}, justfileSource(osFiles{}), dir)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{badgeTask("justfile", justTask("local-only", ""))}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverJustfileDotFallback: discovery reads .justfile when no candidate
// with a higher priority exists, and the name of that file is the badge.
func TestDiscoverJustfileDotFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".justfile", "# Build the app\nbuild:\n    echo build\n")

	tasks, warn := discoverSource(osFiles{}, justfileSource(osFiles{}), dir)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{badgeTask(".justfile", justTask("build", "Build the app"))}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverJustfileImports: discovery lists imported recipes under the
// badge of the root file, because the picker runs "just <name>" from the
// project directory.
func TestDiscoverJustfileImports(t *testing.T) {
	dir := "testdata/justfile-imports"

	tasks, warn := discoverSource(osFiles{}, justfileSource(osFiles{}), dir)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{
		badgeTask("justfile", justTask("root", "Root recipe")),
		badgeTask("justfile", justTask("shared", "Shared recipe")),
	}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverJustfileModules: discovery lists module recipes under the badge
// of the root file, because the picker runs "just name::recipe" from the
// project directory.
func TestDiscoverJustfileModules(t *testing.T) {
	dir := "testdata/justfile-modules"

	tasks, warn := discoverSource(osFiles{}, justfileSource(osFiles{}), dir)
	if warn != nil {
		t.Fatalf("discoverSource() error = %v", warn)
	}
	want := []Task{
		badgeTask("justfile", justTask("root", "Root recipe")),
		badgeTask("justfile", justTask("tools::fmt", "Format the code")),
	}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

func TestDiscoverJustfileMissingFile(t *testing.T) {
	got, warn := discoverSource(osFiles{}, justfileSource(osFiles{}), t.TempDir())
	if warn != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, warn)
	}
}

// TestDiscoverTasksJustfileOrder: rc tasks come first, then package.json,
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

// TestDiscoverTasksOddJustfileKeepsSiblings: a justfile that the parser cannot
// make sense of contributes no recipes and no warning, and it does not hide a
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

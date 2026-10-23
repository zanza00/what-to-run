package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestParseTaskfile covers the task shapes a Taskfile can declare: the short
// string and sequence syntaxes, the table with desc/summary/cmd/cmds, the
// internal flag, and the entries the picker cannot turn into a command line.
func TestParseTaskfile(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Task
	}{
		{
			name: "task table with desc, commands, task and cmd entries",
			in: `version: '3'
tasks:
  build:
    cmds:
      - echo build
      - task: lint
      - cmd: echo done
  lint:
    desc: Lint the code
    cmds: [golangci-lint run ./...]
`,
			want: []Task{
				{Name: "build", Desc: "echo build && task lint && echo done", Args: []string{"task", "build"}},
				{Name: "lint", Desc: "Lint the code", Args: []string{"task", "lint"}},
			},
		},
		{
			name: "short string syntax",
			in: `tasks:
  lint: golangci-lint run ./...
`,
			want: []Task{
				{Name: "lint", Desc: "golangci-lint run ./...", Args: []string{"task", "lint"}},
			},
		},
		{
			name: "short sequence syntax",
			in: `tasks:
  short-list: [echo one, echo two]
`,
			want: []Task{
				{Name: "short-list", Desc: "echo one && echo two", Args: []string{"task", "short-list"}},
			},
		},
		{
			name: "summary falls back to its first non-empty line",
			in: `tasks:
  release:
    summary: |

      Cut the release

      More detail here.
    cmds: [echo release]
`,
			want: []Task{
				{Name: "release", Desc: "Cut the release", Args: []string{"task", "release"}},
			},
		},
		{
			name: "desc wins over summary and commands",
			in: `tasks:
  release:
    desc: Cut the release
    summary: A longer explanation
    cmds: [echo release]
`,
			want: []Task{
				{Name: "release", Desc: "Cut the release", Args: []string{"task", "release"}},
			},
		},
		{
			name: "first command fallback without desc or summary",
			in: `tasks:
  serve:
    cmd: go run ./cmd/server
`,
			want: []Task{
				{Name: "serve", Desc: "go run ./cmd/server", Args: []string{"task", "serve"}},
			},
		},
		{
			name: "cmds list fallback without desc or summary",
			in: `tasks:
  serve:
    cmds:
      - go build ./...
      - ./bin/server
`,
			want: []Task{
				{Name: "serve", Desc: "go build ./... && ./bin/server", Args: []string{"task", "serve"}},
			},
		},
		{
			name: "internal tasks are skipped",
			in: `tasks:
  hidden:
    internal: true
    desc: Not for the picker
    cmds: [echo hidden]
  shown:
    desc: Listed
`,
			want: []Task{
				{Name: "shown", Desc: "Listed", Args: []string{"task", "shown"}},
			},
		},
		{
			name: "internal false is listed",
			in: `tasks:
  shown:
    internal: false
    desc: Listed
`,
			want: []Task{
				{Name: "shown", Desc: "Listed", Args: []string{"task", "shown"}},
			},
		},
		{
			name: "deps-only task is kept with an empty description",
			in: `tasks:
  setup:
    deps: [build]
`,
			want: []Task{
				{Name: "setup", Desc: "", Args: []string{"task", "setup"}},
			},
		},
		{
			name: "null task value is kept with an empty description",
			in: `tasks:
  build:
`,
			want: []Task{
				{Name: "build", Desc: "", Args: []string{"task", "build"}},
			},
		},
		{
			name: "other task shapes are kept with an empty description",
			in: `tasks:
  weird: 3
`,
			want: []Task{
				{Name: "weird", Desc: "", Args: []string{"task", "weird"}},
			},
		},
		{
			name: "namespaced task name is kept and run as task docker:build",
			in: `tasks:
  docker:build:
    desc: Build the image
`,
			want: []Task{
				{Name: "docker:build", Desc: "Build the image", Args: []string{"task", "docker:build"}},
			},
		},
		{
			name: "non-command cmds entries are skipped",
			in: `tasks:
  build:
    cmds:
      - defer: echo cleanup
      - silent: true
      - echo build
`,
			want: []Task{
				{Name: "build", Desc: "echo build", Args: []string{"task", "build"}},
			},
		},
		{
			name: "includes produce no tasks and no error",
			in: `version: '3'
includes:
  docs: ./docs/Taskfile.yml
`,
			want: nil,
		},
		{
			name: "templated desc is shown raw",
			in: `tasks:
  build:
    desc: 'Build {{.VERSION}}'
`,
			want: []Task{
				{Name: "build", Desc: "Build {{.VERSION}}", Args: []string{"task", "build"}},
			},
		},
		{
			name: "tasks are sorted by name",
			in: `tasks:
  test:
    desc: Run the tests
  build:
    desc: Build the app
  all:
    desc: Run everything
`,
			want: []Task{
				{Name: "all", Desc: "Run everything", Args: []string{"task", "all"}},
				{Name: "build", Desc: "Build the app", Args: []string{"task", "build"}},
				{Name: "test", Desc: "Run the tests", Args: []string{"task", "test"}},
			},
		},
		{
			name: "empty file declares no tasks",
			in:   "",
			want: nil,
		},
		{
			name: "whitespace-only file declares no tasks",
			in:   " \n\t\n",
			want: nil,
		},
		{
			name: "file without a tasks map declares no tasks",
			in:   "version: '3'\n",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTaskfile(osFiles{}, "", []byte(tt.in))
			if err != nil {
				t.Fatalf("parseTaskfile() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTaskfile() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseTaskfileCmdsScalarEqualsList pins the shorthand: a cmds scalar is
// the same task as a one-element command list.
func TestParseTaskfileCmdsScalarEqualsList(t *testing.T) {
	scalar, err := parseTaskfile(osFiles{}, "", []byte("tasks:\n  x:\n    cmds: echo one\n"))
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}
	list, err := parseTaskfile(osFiles{}, "", []byte("tasks:\n  x:\n    cmds: [echo one]\n"))
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}

	want := []Task{{Name: "x", Desc: "echo one", Args: []string{"task", "x"}}}
	if !reflect.DeepEqual(scalar, want) {
		t.Errorf("scalar cmds = %+v, want %+v", scalar, want)
	}
	if !reflect.DeepEqual(list, want) {
		t.Errorf("list cmds = %+v, want %+v", list, want)
	}
}

// TestParseTaskfileIncludes covers the local includes the parser follows: the
// string file and directory forms, the map form, namespacing and its chain,
// flatten, optional and mandatory missing files, internal includes, remote
// paths, broken included files and duplicate final names.
func TestParseTaskfileIncludes(t *testing.T) {
	// task is the task the parser derives from a name and a description.
	task := func(name, desc string) Task {
		return Task{Name: name, Desc: desc, Args: []string{"task", name}}
	}

	tests := []struct {
		name  string
		root  string
		files map[string]string
		want  []Task
	}{
		{
			name: "string file include is namespaced",
			root: `version: '3'
includes:
  docs: ./docs/Taskfile.yml
tasks:
  build:
    desc: Build the app
`,
			files: map[string]string{
				"docs/Taskfile.yml": "tasks:\n  serve:\n    desc: Serve the docs\n",
			},
			want: []Task{task("build", "Build the app"), task("docs:serve", "Serve the docs")},
		},
		{
			name: "string directory include picks the first default name",
			root: `includes:
  tools: ./tools
tasks:
  root:
    desc: Root task
`,
			files: map[string]string{
				"tools/taskfile.yml":      "tasks:\n  format:\n    desc: Format the tools\n",
				"tools/Taskfile.dist.yml": "tasks:\n  dist-only:\n    desc: From the dist file\n",
			},
			want: []Task{task("root", "Root task"), task("tools:format", "Format the tools")},
		},
		{
			name: "map include with dir keeps the names and leaves Task.Dir empty",
			root: `includes:
  api:
    taskfile: ./api/Taskfile.yml
    dir: ./api
tasks:
  root:
    desc: Root task
`,
			files: map[string]string{
				"api/Taskfile.yml": "tasks:\n  build:\n    desc: Build the API\n",
			},
			want: []Task{task("api:build", "Build the API"), task("root", "Root task")},
		},
		{
			name: "nested includes chain the namespace",
			root: `includes:
  a: ./a/Taskfile.yml
`,
			files: map[string]string{
				"a/Taskfile.yml": `includes:
  b: ./sub/Taskfile.yml
tasks:
  own:
    desc: From a
`,
				"a/sub/Taskfile.yml": "tasks:\n  deep:\n    desc: From b\n",
			},
			want: []Task{task("a:b:deep", "From b"), task("a:own", "From a")},
		},
		{
			name: "same file included on two branches keeps both namespaces",
			root: `includes:
  a: ./a/Taskfile.yml
  b: ./b/Taskfile.yml
`,
			files: map[string]string{
				"a/Taskfile.yml":      "includes:\n  shared: ../shared/Taskfile.yml\n",
				"b/Taskfile.yml":      "includes:\n  shared: ../shared/Taskfile.yml\n",
				"shared/Taskfile.yml": "tasks:\n  build:\n    desc: Shared build\n",
			},
			want: []Task{
				task("a:shared:build", "Shared build"),
				task("b:shared:build", "Shared build"),
			},
		},
		{
			name: "flatten at the root drops only its own key",
			root: `includes:
  lib:
    taskfile: ./lib/Taskfile.yml
    flatten: true
tasks:
  root:
    desc: Root task
`,
			files: map[string]string{
				"lib/Taskfile.yml": "tasks:\n  build:\n    desc: Build the lib\n",
			},
			want: []Task{task("build", "Build the lib"), task("root", "Root task")},
		},
		{
			name: "flatten nested under a namespace keeps the inherited prefix",
			root: `includes:
  a: ./a/Taskfile.yml
`,
			files: map[string]string{
				"a/Taskfile.yml": `includes:
  sub:
    taskfile: ./sub/Taskfile.yml
    flatten: true
tasks:
  own:
    desc: From a
`,
				"a/sub/Taskfile.yml": "tasks:\n  flat:\n    desc: From sub\n",
			},
			want: []Task{task("a:flat", "From sub"), task("a:own", "From a")},
		},
		{
			name: "optional missing include is skipped",
			root: `includes:
  extra:
    taskfile: ./missing/Taskfile.yml
    optional: true
tasks:
  root:
    desc: Root task
`,
			want: []Task{task("root", "Root task")},
		},
		{
			name: "mandatory missing include is skipped best-effort",
			root: `includes:
  extra: ./missing/Taskfile.yml
tasks:
  root:
    desc: Root task
`,
			want: []Task{task("root", "Root task")},
		},
		{
			name: "internal include hides its nested tasks",
			root: `includes:
  hidden:
    taskfile: ./hidden/Taskfile.yml
    internal: true
  visible: ./visible/Taskfile.yml
tasks:
  root:
    desc: Root task
`,
			files: map[string]string{
				"hidden/Taskfile.yml": `includes:
  more: ./more/Taskfile.yml
tasks:
  secret:
    desc: Should stay hidden
`,
				"hidden/more/Taskfile.yml": "tasks:\n  leak:\n    desc: Also hidden\n",
				"visible/Taskfile.yml":     "tasks:\n  shown:\n    desc: Visible\n",
			},
			want: []Task{task("root", "Root task"), task("visible:shown", "Visible")},
		},
		{
			name: "remote includes are skipped",
			root: `includes:
  docs: https://example.com/Taskfile.yml
  repo: git@github.com:owner/repo.git//Taskfile.yml
tasks:
  root:
    desc: Root task
`,
			want: []Task{task("root", "Root task")},
		},
		{
			name: "included parse error is skipped",
			root: `includes:
  broken: ./broken/Taskfile.yml
tasks:
  root:
    desc: Root task
`,
			files: map[string]string{
				"broken/Taskfile.yml": "tasks: [\n",
			},
			want: []Task{task("root", "Root task")},
		},
		{
			name: "duplicate flatten name keeps the first include",
			root: `includes:
  a:
    taskfile: ./a/Taskfile.yml
    flatten: true
  b:
    taskfile: ./b/Taskfile.yml
    flatten: true
`,
			files: map[string]string{
				"a/Taskfile.yml": "tasks:\n  from-a:\n    desc: From a\n  shared:\n    desc: First\n",
				"b/Taskfile.yml": "tasks:\n  from-b:\n    desc: From b\n  shared:\n    desc: Second\n",
			},
			want: []Task{
				task("from-a", "From a"),
				task("from-b", "From b"),
				task("shared", "First"),
			},
		},
		{
			name: "root task beats a flattened include task",
			root: `includes:
  shared:
    taskfile: ./shared/Taskfile.yml
    flatten: true
tasks:
  build:
    desc: Root build
`,
			files: map[string]string{
				"shared/Taskfile.yml": "tasks:\n  build:\n    desc: Included build\n  test:\n    desc: Shared test\n",
			},
			want: []Task{task("build", "Root build"), task("test", "Shared test")},
		},
		{
			name: "sequence taskfile is skipped",
			root: `includes:
  bad:
    taskfile: [./a/Taskfile.yml]
tasks:
  root:
    desc: Root task
`,
			want: []Task{task("root", "Root task")},
		},
		{
			name: "map include without a taskfile is skipped",
			root: `includes:
  bad:
    dir: ./somewhere
tasks:
  root:
    desc: Root task
`,
			want: []Task{task("root", "Root task")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := parseTaskfile(osFiles{}, dir, []byte(tt.root))
			if err != nil {
				t.Fatalf("parseTaskfile() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTaskfile() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestParseTaskfileTemplatedIncludeSkipped: an include path with Task template
// braces is not followed, even when a directory with that literal name exists.
func TestParseTaskfileTemplatedIncludeSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "{{.DOCS}}/Taskfile.yml", "tasks:\n  serve:\n    desc: Should not appear\n")

	root := `includes:
  docs: ./{{.DOCS}}/Taskfile.yml
tasks:
  root:
    desc: Root task
`
	got, err := parseTaskfile(osFiles{}, dir, []byte(root))
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}
	want := []Task{{Name: "root", Desc: "Root task", Args: []string{"task", "root"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTaskfile() = %+v, want %+v", got, want)
	}
}

// TestParseTaskfileIncludeCycle: two Taskfiles including each other terminate,
// with the include read once and the result stable across calls.
func TestParseTaskfileIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Taskfile.yml", "includes:\n  a: ./a/Taskfile.yml\n")
	writeFile(t, dir, "a/Taskfile.yml", "includes:\n  b: ../b/Taskfile.yml\ntasks:\n  one:\n    desc: From a\n")
	writeFile(t, dir, "b/Taskfile.yml", "includes:\n  a: ../a/Taskfile.yml\ntasks:\n  two:\n    desc: From b\n")
	root, err := os.ReadFile(filepath.Join(dir, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}

	first, err := parseTaskfile(osFiles{}, dir, root)
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}
	second, err := parseTaskfile(osFiles{}, dir, root)
	if err != nil {
		t.Fatalf("second parseTaskfile() error = %v", err)
	}
	want := []Task{
		{Name: "a:b:two", Desc: "From b", Args: []string{"task", "a:b:two"}},
		{Name: "a:one", Desc: "From a", Args: []string{"task", "a:one"}},
	}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("parseTaskfile() = %+v, want %+v", first, want)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("parseTaskfile() is not deterministic: %+v vs %+v", first, second)
	}
}

// TestParseTaskfileIncludesFixture parses the permanent fixture: a root
// Taskfile with a map include into .taskfiles/, a string directory include, a
// flatten include, an internal include, an optional missing include and a
// nested include. It pins the final names, the descriptions and the order.
func TestParseTaskfileIncludesFixture(t *testing.T) {
	const dir = "testdata/taskfile-includes"
	root, err := os.ReadFile(filepath.Join(dir, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := parseTaskfile(osFiles{}, dir, root)
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}
	want := []Task{
		{Name: "build", Desc: "Build the project", Args: []string{"task", "build"}},
		{Name: "docs:nested:lint", Desc: "Lint the documentation", Args: []string{"task", "docs:nested:lint"}},
		{Name: "docs:serve", Desc: "Serve the documentation", Args: []string{"task", "docs:serve"}},
		{Name: "test", Desc: "Test the shared code", Args: []string{"task", "test"}},
		{Name: "tools:format", Desc: "Format the tools", Args: []string{"task", "tools:format"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTaskfile() = %+v, want %+v", got, want)
	}
}

// TestDiscoverTaskfileBadYAML: a malformed Taskfile is a warning naming the
// file that broke, not a fatal error and not a silent empty list.
func TestDiscoverTaskfileBadYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Taskfile.yml", "tasks:\n  build: [\n")

	tasks, err := discoverSource(osFiles{}, taskfileSource(osFiles{}), dir)
	if err == nil {
		t.Fatalf("discoverSource() = %+v, nil; want a parse error", tasks)
	}
	if !strings.Contains(err.Error(), "Taskfile.yml") {
		t.Errorf("error = %q, want it to mention Taskfile.yml", err)
	}
}

// TestDiscoverTaskfileFirstFileWins pins the candidate-file preference:
// Taskfile.yml beats Taskfile.dist.yml. The names differ by ".dist", not by
// case, so the fixture is safe on the case-insensitive filesystem of macOS.
func TestDiscoverTaskfileFirstFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Taskfile.dist.yml", "tasks:\n  dist-only:\n    desc: From the dist file\n")
	writeFile(t, dir, "Taskfile.yml", "tasks:\n  local-only:\n    desc: From the local file\n")

	tasks, err := discoverSource(osFiles{}, taskfileSource(osFiles{}), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "Taskfile.yml", Name: "local-only", Desc: "From the local file", Args: []string{"task", "local-only"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverTaskfileDistFallback: Taskfile.dist.yml is read when there is no
// higher-priority candidate, and its name is the badge.
func TestDiscoverTaskfileDistFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Taskfile.dist.yml", "tasks:\n  build:\n    desc: Build from the dist file\n")

	tasks, err := discoverSource(osFiles{}, taskfileSource(osFiles{}), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "Taskfile.dist.yml", Name: "build", Desc: "Build from the dist file", Args: []string{"task", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

func TestDiscoverTaskfileMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, taskfileSource(osFiles{}), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

// TestTaskfileSourceFiles pins the eight default names in go-task's search
// order: the order is the preference order of discoverSource.
func TestTaskfileSourceFiles(t *testing.T) {
	want := []string{
		"Taskfile.yml", "taskfile.yml", "Taskfile.yaml", "taskfile.yaml",
		"Taskfile.dist.yml", "taskfile.dist.yml",
		"Taskfile.dist.yaml", "taskfile.dist.yaml",
	}
	if got := taskfileSource(osFiles{}).Files; !reflect.DeepEqual(got, want) {
		t.Errorf("taskfileSource(osFiles{}).Files = %v, want %v", got, want)
	}
}

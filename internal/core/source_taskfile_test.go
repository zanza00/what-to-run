package core

import (
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
			got, err := parseTaskfile("", []byte(tt.in))
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
	scalar, err := parseTaskfile("", []byte("tasks:\n  x:\n    cmds: echo one\n"))
	if err != nil {
		t.Fatalf("parseTaskfile() error = %v", err)
	}
	list, err := parseTaskfile("", []byte("tasks:\n  x:\n    cmds: [echo one]\n"))
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

// TestDiscoverTaskfileBadYAML: a malformed Taskfile is a warning naming the
// file that broke, not a fatal error and not a silent empty list.
func TestDiscoverTaskfileBadYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Taskfile.yml", "tasks:\n  build: [\n")

	tasks, err := discoverSource(osFiles{}, taskfileSource(), dir)
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

	tasks, err := discoverSource(osFiles{}, taskfileSource(), dir)
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

	tasks, err := discoverSource(osFiles{}, taskfileSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "Taskfile.dist.yml", Name: "build", Desc: "Build from the dist file", Args: []string{"task", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

func TestDiscoverTaskfileMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, taskfileSource(), t.TempDir())
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
	if got := taskfileSource().Files; !reflect.DeepEqual(got, want) {
		t.Errorf("taskfileSource().Files = %v, want %v", got, want)
	}
}

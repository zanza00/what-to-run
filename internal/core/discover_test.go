package core

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverPackageScripts(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		locks   []string
		runner  string
		want    []Task
		wantErr bool
	}{
		{
			name: "scripts sorted, npm by default",
			pkg:  `{"scripts": {"test": "go test ./...", "build": "go build ./..."}}`,
			want: []Task{
				{Source: "package.json", Name: "build", Desc: "go build ./...", Args: []string{"npm", "run", "build"}},
				{Source: "package.json", Name: "test", Desc: "go test ./...", Args: []string{"npm", "run", "test"}},
			},
		},
		{
			name: "packageManager field wins",
			pkg:  `{"scripts": {"dev": "vite"}, "packageManager": "bun@1.2.3"}`,
			want: []Task{
				{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"bun", "run", "dev"}},
			},
		},
		{
			name: "bare packageManager field wins",
			pkg:  `{"scripts": {"dev": "vite"}, "packageManager": "yarn"}`,
			want: []Task{
				{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"yarn", "run", "dev"}},
			},
		},
		{
			name:  "lock file decides",
			pkg:   `{"scripts": {"dev": "vite"}}`,
			locks: []string{"pnpm-lock.yaml"},
			want: []Task{
				{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"pnpm", "run", "dev"}},
			},
		},
		{
			name:  "yarn lock file",
			pkg:   `{"scripts": {"dev": "vite"}}`,
			locks: []string{"yarn.lock"},
			want: []Task{
				{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"yarn", "run", "dev"}},
			},
		},
		{
			name:   "rc package_manager wins over the lock file",
			pkg:    `{"scripts": {"dev": "vite"}}`,
			locks:  []string{"pnpm-lock.yaml"},
			runner: "bun",
			want: []Task{
				{Source: "package.json", Name: "dev", Desc: "vite", Args: []string{"bun", "run", "dev"}},
			},
		},
		{
			name:    "broken json",
			pkg:     `{"scripts": `,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "package.json", tt.pkg)
			for _, lock := range tt.locks {
				writeFile(t, dir, lock, "")
			}

			got, err := discoverSource(packageJSONSource(tt.runner), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "package.json") {
					t.Errorf("error = %q, want it to mention package.json", err)
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

func TestDiscoverPackageScriptsMissingFile(t *testing.T) {
	got, err := discoverSource(packageJSONSource(""), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

const makefileFixture = `# Variables
NAME := app

.PHONY: all build clean

## build: Build the application
build: bin/$(NAME)

## test - Run the tests
test:
	go test ./...

clean:
	rm -rf bin

%.o: %.c
	cc -c $<

# A comment describing run
run: build
	./bin/$(NAME)
`

func TestParseMakeTargets(t *testing.T) {
	want := []Task{
		{Name: "build", Desc: "Build the application", Args: []string{"make", "build"}},
		{Name: "test", Desc: "Run the tests", Args: []string{"make", "test"}},
		{Name: "clean", Desc: "", Args: []string{"make", "clean"}},
		{Name: "run", Desc: "A comment describing run", Args: []string{"make", "run"}},
	}

	got := parseMakeTargets([]byte(makefileFixture))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseMakeTargets() = %+v, want %+v", got, want)
	}
}

func TestParseMakeTargetsDeduplicates(t *testing.T) {
	got := parseMakeTargets([]byte("build:\n\techo one\n\nbuild:\n\techo two\n"))
	if len(got) != 1 || got[0].Name != "build" {
		t.Errorf("parseMakeTargets() = %+v, want a single build task", got)
	}
}

// TestDiscoverTaskfileOfThisRepo runs the source over the project's own
// Taskfile, which is the repository's build entry point: every task a
// contributor types must reach the picker with its description.
func TestDiscoverTaskfileOfThisRepo(t *testing.T) {
	tasks, err := discoverSource(taskfileSource(), "../..")
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}

	byName := map[string]Task{}
	for _, task := range tasks {
		byName[task.Name] = task
	}
	for _, name := range []string{
		"default", "all", "build", "bind", "bind-bash", "install", "run",
		"test", "fmt", "lint", "tidy", "clean", "help",
	} {
		if _, ok := byName[name]; !ok {
			t.Errorf("task %q not found in this repo's Taskfile, got %v", name, tasks)
		}
	}
	if got := byName["build"].Source; got != "Taskfile.yml" {
		t.Errorf("build source = %q, want Taskfile.yml", got)
	}
	const wantDesc = "Build bin/wtr and bin/what-to-run"
	if got := byName["build"].Desc; got != wantDesc {
		t.Errorf("build description = %q, want %q", got, wantDesc)
	}
	if got := byName["build"].Args; !reflect.DeepEqual(got, []string{"task", "build"}) {
		t.Errorf("build args = %v, want [task build]", got)
	}
}

// TestDiscoverSourceFirstFileWins pins the candidate-file preference: the
// first name in source.Files that exists is the one parsed and shown. The
// fixtures avoid names that differ only by case, which collide on the
// case-insensitive filesystem of macOS.
func TestDiscoverSourceFirstFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "GNUmakefile", "from-gnu:\n\ttrue\n")
	writeFile(t, dir, "Makefile", "from-upper:\n\ttrue\n")

	tasks, err := discoverSource(makeSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].Name != "from-gnu" || tasks[0].Source != "GNUmakefile" {
		t.Errorf("discoverSource() = %+v, want the GNUmakefile task with the GNUmakefile badge", tasks)
	}
}

func TestDiscoverCargoMakeTasks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile.toml", `
[tasks.format]
description = "Run rustfmt"
command = "rustfmt"

[tasks.build]
command = "cargo"
args = ["build", "--release"]

[tasks.internal]
private = true
command = "echo internal"

[tasks.disabled]
disabled = true
command = "echo disabled"
`)

	got, err := discoverSource(cargoMakeSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{
		{Source: "Makefile.toml", Name: "build", Desc: "cargo build --release", Args: []string{"cargo", "make", "build"}},
		{Source: "Makefile.toml", Name: "format", Desc: "Run rustfmt", Args: []string{"cargo", "make", "format"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("discoverSource() = %+v, want %+v", got, want)
	}
}

func TestDiscoverCargoMakeTasksBrokenToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile.toml", "[tasks\n")

	_, err := discoverSource(cargoMakeSource(), dir)
	if err == nil || !strings.Contains(err.Error(), "Makefile.toml") {
		t.Errorf("discoverSource() error = %v, want a Makefile.toml parse error", err)
	}
}

func TestDiscoverTasksMergesSources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts": {"build": "go build ./..."}}`)
	// GNUmakefile, not Makefile: case-sensitive filesystems would otherwise
	// resolve the earlier "makefile" candidate and change the badge.
	writeFile(t, dir, "GNUmakefile", "test:\n\tgo test ./...\n")
	writeFile(t, dir, "Makefile.toml", "[tasks.fmt]\ndescription = \"Format\"\n")
	writeFile(t, dir, ".cargo/config.toml", "[alias]\nlint = \"clippy\"\n")

	tasks, errs := Discover(dir, Settings{})
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}

	var sources []string
	for _, task := range tasks {
		sources = append(sources, task.Source)
	}
	if want := []string{"package.json", "GNUmakefile", "Makefile.toml", ".cargo/config.toml"}; !reflect.DeepEqual(sources, want) {
		t.Errorf("task sources = %v, want %v", sources, want)
	}
}

// TestDiscoverTasksTaskfileOrder: a directory with both a package.json and a
// Taskfile lists the package scripts first and the Taskfile tasks second, with
// the rc tasks still on top.
func TestDiscoverTasksTaskfileOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts": {"dev": "vite"}}`)
	writeFile(t, dir, "Taskfile.yml", "version: '3'\ntasks:\n  build:\n    desc: Build the app\n")

	settings := Settings{Tasks: []RCTask{{Name: "deploy", Command: "./deploy.sh"}}}
	tasks, errs := Discover(dir, settings)
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}

	var got []string
	for _, task := range tasks {
		got = append(got, task.Source+"/"+task.Name)
	}
	want := []string{"rc/deploy", "package.json/dev", "Taskfile.yml/build"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tasks = %v, want %v", got, want)
	}
}

func TestDiscoverTasksSkipsBrokenSource(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `not json`)
	writeFile(t, dir, "Makefile", "build:\n\techo hi\n")

	tasks, errs := Discover(dir, Settings{})
	if len(errs) != 1 {
		t.Fatalf("Discover() errors = %v, want exactly one", errs)
	}
	if len(tasks) != 1 || tasks[0].Name != "build" {
		t.Errorf("Discover() tasks = %+v, want the Makefile build task", tasks)
	}
}

// TestDiscoverTasksRCTasksFirst: rc tasks come first, run through the shell,
// default their description to the command and resolve a relative cwd.
func TestDiscoverTasksRCTasksFirst(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts": {"build": "go build ./..."}}`)

	settings := Settings{Tasks: []RCTask{
		{Name: "deploy", Command: "scripts/deploy.sh --prod", Description: "Deploy to production", Cwd: "ops"},
		{Name: "status", Command: "git status"},
	}}

	tasks, errs := Discover(dir, settings)
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}
	want := []Task{
		{Source: "rc", Name: "deploy", Desc: "Deploy to production", Args: []string{"sh", "-c", "scripts/deploy.sh --prod"}, Dir: filepath.Join(dir, "ops")},
		{Source: "rc", Name: "status", Desc: "git status", Args: []string{"sh", "-c", "git status"}},
		{Source: "package.json", Name: "build", Desc: "go build ./...", Args: []string{"npm", "run", "build"}},
	}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("Discover() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverTasksRCShadowsBuiltin: an rc task hides the same-named task of
// the built-in sources.
func TestDiscoverTasksRCShadowsBuiltin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts": {"build": "go build ./..."}}`)

	settings := Settings{Tasks: []RCTask{{Name: "build", Command: "make -C tools build"}}}
	tasks, errs := Discover(dir, settings)
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}
	if len(tasks) != 1 || tasks[0].Source != "rc" || tasks[0].Name != "build" {
		t.Errorf("Discover() = %+v, want just the rc build task", tasks)
	}
}

// TestDiscoverTasksDisabledSourceIsNotRead: a disabled source is not even
// read, so a file it could not parse cannot warn.
func TestDiscoverTasksDisabledSourceIsNotRead(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile.toml", "[tasks\n") // broken, but disabled
	writeFile(t, dir, "Makefile", "build:\n\techo hi\n")

	tasks, errs := Discover(dir, Settings{Disable: []string{"Makefile.toml"}})
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none from the disabled source", errs)
	}
	if len(tasks) != 1 || tasks[0].Name != "build" {
		t.Errorf("Discover() = %+v, want the Makefile build task", tasks)
	}
}

// TestDiscoverTasksUnknownDisableEntry: a disable entry that names no source
// is reported, and no source is disabled.
func TestDiscoverTasksUnknownDisableEntry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", "build:\n\techo hi\n")

	tasks, errs := Discover(dir, Settings{Disable: []string{"cargo"}})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "cargo") {
		t.Errorf("Discover() errors = %v, want one warning about %q", errs, "cargo")
	}
	if len(tasks) != 1 || tasks[0].Name != "build" {
		t.Errorf("Discover() = %+v, want the Makefile build task", tasks)
	}
}

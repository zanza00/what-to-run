// Task discovery: the runnable scripts a project declares.
//
// Three sources are understood, in this fixed order:
//
//	package.json   npm/pnpm/yarn/bun scripts
//	Makefile       make targets (also makefile, GNUmakefile)
//	Cargo.toml     cargo-make tasks
//
// A source that is missing contributes nothing; a source that exists but
// fails to parse is reported in errs and skipped, so one broken file does
// not hide the tasks of the others.

package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Task is a script discovered in the project: it can be run and shown in
// the picker.
type Task struct {
	Source string   // file declaring the task: "package.json", "Makefile" or "Cargo.toml"
	Name   string   // task name as declared in Source
	Desc   string   // human-readable description; may be empty
	Args   []string // argv of the command that runs the task
}

// command returns the process that runs the task.
func (t Task) command() *exec.Cmd {
	return exec.Command(t.Args[0], t.Args[1:]...)
}

// discoverTasks scans dir for the task sources and merges them in a stable
// order: package.json, Makefile, Cargo.toml. Scripts and cargo-make tasks are
// sorted by name; Makefile targets keep the order they appear in the file.
func discoverTasks(dir string) (tasks []Task, errs []error) {
	for _, discover := range []func(string) ([]Task, error){
		discoverPackageScripts,
		discoverMakeTargets,
		discoverCargoMakeTasks,
	} {
		ts, err := discover(dir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tasks = append(tasks, ts...)
	}
	return tasks, errs
}

// discoverPackageScripts returns the scripts declared in package.json.
func discoverPackageScripts(dir string) ([]Task, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Scripts        map[string]string `json:"scripts"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("package.json: %w", err)
	}

	runner := packageRunner(dir, pkg.PackageManager)
	tasks := make([]Task, 0, len(pkg.Scripts))
	for _, name := range slices.Sorted(maps.Keys(pkg.Scripts)) {
		tasks = append(tasks, Task{
			Source: "package.json",
			Name:   name,
			Desc:   pkg.Scripts[name],
			Args:   append(slices.Clone(runner), name),
		})
	}
	return tasks, nil
}

// packageRunner returns the argv prefix for running a package.json script:
// the declared packageManager field wins, then the lock file in use, then npm.
func packageRunner(dir, declared string) []string {
	if i := strings.IndexByte(declared, '@'); i > 0 {
		switch declared[:i] {
		case "bun", "pnpm", "yarn", "npm":
			return []string{declared[:i], "run"}
		}
	}
	for _, c := range []struct{ lock, runner string }{
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	} {
		if _, err := os.Stat(filepath.Join(dir, c.lock)); err == nil {
			return []string{c.runner, "run"}
		}
	}
	return []string{"npm", "run"}
}

// makeTargetRe matches a Makefile target line: a name at the start of the
// line followed by one or two colons, as in "build: ## Build everything".
var makeTargetRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.+-]*)[ \t]*:{1,2}[ \t]*(.*)$`)

// discoverMakeTargets returns the targets of the project's Makefile. The
// file is the first of GNUmakefile, makefile, Makefile that exists, which
// is the order make itself uses.
func discoverMakeTargets(dir string) ([]Task, error) {
	for _, name := range []string{"GNUmakefile", "makefile", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return parseMakeTargets(data, name), nil
	}
	return nil, nil
}

// parseMakeTargets extracts the targets from a Makefile.
//
// The parser is intentionally shallow: it recognizes "name: deps" and
// "name:: deps" lines at the start of a line, skips variable assignments
// ("NAME := value") and special targets (".PHONY", pattern rules), and uses
// the contiguous comment directly above a target as its description. The
// common "## name: text" and "## name - text" conventions (`make help`
// style) are cleaned up to just the text.
func parseMakeTargets(data []byte, file string) []Task {
	var (
		tasks   []Task
		seen    = map[string]bool{}
		comment string
	)

	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")

		if rest, ok := strings.CutPrefix(line, "#"); ok {
			comment = rest
			continue
		}
		if strings.TrimSpace(line) == "" {
			comment = ""
			continue
		}

		m := makeTargetRe.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(m[2], "=") { // variable assignment (NAME := value)
			comment = ""
			continue
		}
		name := m[1]
		if seen[name] {
			comment = ""
			continue
		}
		seen[name] = true
		tasks = append(tasks, Task{
			Source: file,
			Name:   name,
			Desc:   makeDescription(name, comment),
			Args:   []string{"make", name},
		})
		comment = ""
	}
	return tasks
}

// makeDescription derives a task description from the comment above a
// target, stripping the "name: text" and "name - text" help prefixes.
func makeDescription(name, comment string) string {
	c := strings.TrimSpace(strings.TrimLeft(comment, "#"))
	if rest, ok := strings.CutPrefix(c, name+":"); ok {
		return strings.TrimSpace(rest)
	}
	if rest, ok := strings.CutPrefix(c, name+" - "); ok {
		return strings.TrimSpace(rest)
	}
	return c
}

// cargoTask is the subset of a cargo-make task definition the picker uses.
type cargoTask struct {
	Description string   `toml:"description"`
	Command     string   `toml:"command"`
	Args        []string `toml:"args"`
}

// discoverCargoMakeTasks returns the cargo-make tasks declared in the
// project's Cargo.toml [tasks] section.
func discoverCargoMakeTasks(dir string) ([]Task, error) {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var doc struct {
		Tasks map[string]cargoTask `toml:"tasks"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("Cargo.toml: %w", err)
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		t := doc.Tasks[name]
		desc := t.Description
		if desc == "" && t.Command != "" {
			desc = strings.Join(append([]string{t.Command}, t.Args...), " ")
		}
		tasks = append(tasks, Task{
			Source: "Cargo.toml",
			Name:   name,
			Desc:   desc,
			Args:   []string{"cargo", "make", name},
		})
	}
	return tasks, nil
}

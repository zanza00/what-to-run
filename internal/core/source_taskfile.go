// The Taskfile.yml source: go-task tasks.
package core

import (
	"bytes"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// taskfileNames are the eight default Taskfile names, in the order go-task
// searches them: the preference order of the source, and the names a
// directory include is searched for.
var taskfileNames = []string{
	"Taskfile.yml", "taskfile.yml", "Taskfile.yaml", "taskfile.yaml",
	"Taskfile.dist.yml", "taskfile.dist.yml",
	"Taskfile.dist.yaml", "taskfile.dist.yaml",
}

// taskfileSource returns the source for go-task tasks: the first of the eight
// default Taskfile names that exists, in the order go-task itself searches.
// Like task, it also looks in the parent directories of the project directory.
func taskfileSource(files Files) source {
	return source{
		Files:         slices.Clone(taskfileNames),
		SearchParents: true,
		Parse: func(dir string, data []byte) ([]Task, error) {
			return parseTaskfile(files, dir, data)
		},
	}
}

// taskfileDoc is one Taskfile document: its own tasks and the local Taskfiles
// it includes.
type taskfileDoc struct {
	Version  string         `yaml:"version"`
	Includes map[string]any `yaml:"includes"`
	Tasks    map[string]any `yaml:"tasks"`
}

// parseTaskfile returns the tasks a Taskfile declares, its local includes
// followed recursively, sorted by name like "task --list-all", each run as
// "task <name>".
//
// The description is the task's desc, else the first line of its summary,
// else its command line: a Taskfile has no help convention beyond those
// fields, and the picker must still show every runnable task. A task whose
// body is only deps or a null value is kept with an empty description.
// Internal tasks are skipped: "task --list" hides them and they cannot be
// called directly.
//
// Includes bring in another local Taskfile, relative to the including file's
// directory, and their tasks are namespaced with the include key ("docs:build",
// chained for nested includes). An include with flatten: true keeps the
// inherited namespace instead of adding its key, an internal include hides
// all its tasks (nested ones too), and aliases are not listed as extra rows.
// Remote and templated include paths are not followed. An include that is
// missing, unreadable or unparseable is skipped without losing the tasks
// already gathered; only an error in the root file is returned.
//
// desc and summary are shown as written, without evaluating Task templates.
// The include dir field only affects where task runs, not the listed names.
// The first definition of a final name wins: the root tasks come before the
// includes, and includes are read in sorted key order.
func parseTaskfile(files Files, dir string, data []byte) ([]Task, error) {
	// An empty file declares no tasks: report it like a missing one
	// instead of failing on an empty YAML document.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	doc, err := taskfileDocOf(data)
	if err != nil {
		return nil, err
	}

	// Keyed by final name so a duplicate (a flatten collision, or the same
	// name reached twice) keeps the first definition.
	collected := make(map[string]Task)
	taskfileCollect(files, doc, dir, "", nil, collected)

	tasks := make([]Task, 0, len(collected))
	for _, name := range slices.Sorted(maps.Keys(collected)) {
		tasks = append(tasks, collected[name])
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// taskfileDocOf unmarshals one Taskfile document. An empty file is a document
// without tasks, not an error.
func taskfileDocOf(data []byte) (taskfileDoc, error) {
	var doc taskfileDoc
	if len(bytes.TrimSpace(data)) == 0 {
		return doc, nil
	}
	err := yaml.Unmarshal(data, &doc)
	return doc, err
}

// taskfileCollect adds one document's tasks to collected and recurses into its
// includes. prefix is the namespace inherited from the parent include and
// active holds the absolute paths on the current include chain: a file already
// on it is a cycle, while the same file included from a sibling branch is read
// again under that branch's namespace.
func taskfileCollect(files Files, doc taskfileDoc, dir, prefix string, active map[string]bool, collected map[string]Task) {
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		desc, ok := taskfileDesc(doc.Tasks[name])
		if !ok {
			continue
		}
		final := prefix + name
		if _, exists := collected[final]; !exists {
			collected[final] = Task{Name: final, Desc: desc, Args: []string{"task", final}}
		}
	}

	// No directory means no file reads: the tests that parse bytes in
	// memory see the root document only.
	if dir == "" {
		return
	}
	if active == nil {
		active = make(map[string]bool)
	}

	for _, key := range slices.Sorted(maps.Keys(doc.Includes)) {
		inc, ok := taskfileIncludeOf(doc.Includes[key])
		if !ok || inc.internal {
			continue
		}
		path, data, ok := taskfileIncludeFile(files, dir, inc.path)
		if !ok {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil || active[abs] {
			continue
		}
		included, err := taskfileDocOf(data)
		if err != nil {
			continue
		}
		next := prefix
		if !inc.flatten {
			next += key + ":"
		}
		active[abs] = true
		taskfileCollect(files, included, filepath.Dir(path), next, active, collected)
		delete(active, abs)
	}
}

// taskfileInclude is one normalized includes entry: the path to read and the
// flags that affect the tasks it brings in. The dir field only changes where
// task runs, so it is ignored here.
type taskfileInclude struct {
	path     string
	internal bool
	flatten  bool
}

// taskfileIncludeOf normalizes an includes value. A string is the path; a
// table carries it in taskfile, which must be a string. ok is false for a
// table without a taskfile and for a remote or templated path, which the
// parser does not follow.
func taskfileIncludeOf(v any) (taskfileInclude, bool) {
	inc := taskfileInclude{}
	switch entry := v.(type) {
	case string:
		inc.path = entry
	case map[string]any:
		path, isString := entry["taskfile"].(string)
		if !isString || path == "" {
			return inc, false
		}
		inc.path = path
		if internal, isBool := entry["internal"].(bool); isBool {
			inc.internal = internal
		}
		if flatten, isBool := entry["flatten"].(bool); isBool {
			inc.flatten = flatten
		}
	default:
		return inc, false
	}

	if inc.path == "" || strings.Contains(inc.path, "{{") || strings.Contains(inc.path, "}}") {
		return inc, false
	}
	for _, remote := range []string{"http://", "https://", "git@"} {
		if strings.HasPrefix(inc.path, remote) {
			return inc, false
		}
	}
	return inc, true
}

// taskfileIncludeFile resolves an include path against the including file's
// directory and reads it. A directory is searched for the eight default names
// in go-task's order. The Files port has no Stat, so a path that reads as a
// file wins and a directory is the one whose default name reads. ok is false
// when nothing readable exists there.
func taskfileIncludeFile(files Files, dir, path string) (string, []byte, bool) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	if data, err := files.ReadFile(path); err == nil {
		return path, data, true
	}
	for _, name := range taskfileNames {
		candidate := filepath.Join(path, name)
		if data, err := files.ReadFile(candidate); err == nil {
			return candidate, data, true
		}
	}
	return "", nil, false
}

// taskfileDesc turns one tasks value into the description of the task and
// reports whether the task should be listed: a short-syntax command string or
// command list, or a task table whose desc, summary or command line describes
// it. ok is false only for an internal task, which the picker hides.
func taskfileDesc(v any) (string, bool) {
	switch task := v.(type) {
	case string:
		return task, true
	case []any:
		return taskfileCommands(task), true
	case map[string]any:
		if internal, isBool := task["internal"].(bool); isBool && internal {
			return "", false
		}
		if desc, isString := task["desc"].(string); isString && desc != "" {
			return desc, true
		}
		if summary, isString := task["summary"].(string); isString && strings.TrimSpace(summary) != "" {
			if line := firstNonEmptyLine(summary); line != "" {
				return line, true
			}
		}
		if cmd, isString := task["cmd"].(string); isString && cmd != "" {
			return cmd, true
		}
		if cmds, ok := task["cmds"]; ok {
			return taskfileCmdsDesc(cmds), true
		}
		// A table without a description or a command (deps-only, or a
		// null body) is still runnable: keep it with an empty desc.
		return "", true
	}
	// Any other shape (a number, a boolean) is not a command, but the task
	// exists: keep it so it stays reachable where Task can run it.
	return "", true
}

// taskfileCmdsDesc renders a task's cmds field: a list of commands, or a
// single command string, which is the same as a one-element list.
func taskfileCmdsDesc(cmds any) string {
	switch list := cmds.(type) {
	case string:
		return list
	case []any:
		return taskfileCommands(list)
	}
	return ""
}

// taskfileCommands renders a Taskfile command list as one description line:
// a bare command string, a "task:" reference to another task and a "cmd:"
// block are shown; any other entry (defer, silent, vars, platforms) is
// skipped. Only the other tasks' names and the shell lines matter here.
func taskfileCommands(cmds []any) string {
	parts := make([]string, 0, len(cmds))
	for _, entry := range cmds {
		line, ok := taskfileCommand(entry)
		if !ok || line == "" {
			continue
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " && ")
}

// taskfileCommand renders one cmds entry: a command string, a task
// reference ("task: name") or a cmd block ("cmd: text"). ok is false for the
// entries that change how a command runs rather than what it is.
func taskfileCommand(v any) (string, bool) {
	switch entry := v.(type) {
	case string:
		return entry, true
	case map[string]any:
		if name, isString := entry["task"].(string); isString && name != "" {
			return "task " + name, true
		}
		if cmd, isString := entry["cmd"].(string); isString {
			return cmd, true
		}
	}
	return "", false
}

// firstNonEmptyLine returns the first line of a block string that holds
// anything but whitespace, trimmed.
func firstNonEmptyLine(s string) string {
	for line := range strings.Lines(s) {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

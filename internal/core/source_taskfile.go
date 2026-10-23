// The Taskfile.yml source: go-task tasks.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// taskfileSource returns the source for go-task tasks: the first of the eight
// default Taskfile names that exists, in the order go-task itself searches.
func taskfileSource() source {
	return source{
		Files: []string{
			"Taskfile.yml", "taskfile.yml", "Taskfile.yaml", "taskfile.yaml",
			"Taskfile.dist.yml", "taskfile.dist.yml",
			"Taskfile.dist.yaml", "taskfile.dist.yaml",
		},
		Parse: parseTaskfile,
	}
}

// parseTaskfile returns the tasks declared under a Taskfile's top-level tasks
// map, sorted by name like "task --list", each run as "task <name>".
//
// The description is the task's desc, else the first line of its summary,
// else its command line: a Taskfile has no help convention beyond those
// fields, and the picker must still show every runnable task. A task whose
// body is only deps or a null value is kept with an empty description.
// Internal tasks are skipped: "task --list" hides them and they cannot be
// called directly.
//
// Only the file's own tasks are read: includes and remote Taskfiles are not
// followed, aliases are not listed as extra rows, and label does not rename
// the task. desc and summary are shown as written, without evaluating Task
// templates.
func parseTaskfile(_ string, data []byte) ([]Task, error) {
	// An empty file declares no tasks: report it like a missing one
	// instead of failing on an empty YAML document.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Tasks map[string]any `yaml:"tasks"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		desc, ok := taskfileDesc(doc.Tasks[name])
		if !ok {
			continue
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"task", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
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

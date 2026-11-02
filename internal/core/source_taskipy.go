// The taskipy source: tasks of taskipy, from pyproject.toml.
package core

import (
	"bytes"
	"maps"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

// taskipySource returns the source for the pyproject.toml of the project. The
// source searches the parent directories, because taskipy walks up to find the
// pyproject.toml.
func taskipySource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "taskipy",
			Label:         "taskipy tasks",
			Files:         []string{"pyproject.toml"},
			SearchParents: true,
		},
		Parse: parseTaskipy,
	}
}

// parseTaskipy returns the tasks that the [tool.taskipy.tasks] table declares,
// sorted by name. Each task runs through "task". The description comes from
// the task's help text and otherwise from the command, which is display text
// and never a promise of the executed command. The variables and settings
// tables under [tool.taskipy] declare no tasks, so this function does not
// read them. An entry that is not a task shape is skipped.
func parseTaskipy(_, _ string, data []byte) ([]Task, error) {
	// An empty file declares no tasks, so report it like a missing file
	// instead of failing on invalid TOML.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Tool struct {
			Taskipy struct {
				Tasks map[string]any `toml:"tasks"`
			} `toml:"taskipy"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	tasks := doc.Tool.Taskipy.Tasks
	out := make([]Task, 0, len(tasks))
	for _, name := range slices.Sorted(maps.Keys(tasks)) {
		desc, ok := taskipyTaskDesc(tasks[name])
		if !ok {
			continue
		}
		out = append(out, Task{
			Name: name,
			Desc: desc,
			Args: []string{"task", name},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// taskipyTaskDesc turns one task entry into its description, as display text.
// A string is a command, and a table holds the command under "cmd". The
// table's help text overrides the command. ok is false when the entry is not
// a task shape, because taskipy rejects such a task and the caller skips the
// entry.
func taskipyTaskDesc(v any) (string, bool) {
	switch task := v.(type) {
	case string:
		return task, true
	case map[string]any:
		cmd, isString := task["cmd"].(string)
		if !isString {
			return "", false
		}
		if help, isString := task["help"].(string); isString && help != "" {
			return help, true
		}
		return cmd, true
	}
	return "", false
}

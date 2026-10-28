// The mise.toml source: mise tasks.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// miseSource returns the source for the project's mise.toml or .mise.toml,
// in that order: mise looks for mise.toml first, so it wins when both exist.
func miseSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:    "mise",
			Label: "mise.toml",
			Files: []string{"mise.toml", ".mise.toml"},
		},
		Parse: parseMise,
	}
}

// parseMise returns the tasks declared in mise.toml's [tasks] table, sorted
// by name, each run through "mise run". A task is a command string or a table
// with a description, run commands and depends; the description shows the run
// commands when the table does not declare one, and a table without a run is
// kept — a depends-only task is still runnable. An entry whose value is
// neither a string nor a table is skipped.
func parseMise(_ string, data []byte) ([]Task, error) {
	// An empty file declares no tasks: report it like a missing one
	// instead of failing on invalid TOML.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Tasks map[string]any `toml:"tasks"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		desc, ok := miseTaskDesc(doc.Tasks[name])
		if !ok {
			continue
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"mise", "run", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// miseTaskDesc turns one tasks value into the description of the task: a
// command string, or a table whose description overrides its run commands.
// ok is false when the value is neither shape — a number, a boolean, an
// array — and the caller skips that entry.
func miseTaskDesc(v any) (string, bool) {
	switch task := v.(type) {
	case string:
		return task, true
	case map[string]any:
		if desc, isString := task["description"].(string); isString && desc != "" {
			return desc, true
		}
		desc, _ := miseRunDesc(task["run"])
		return desc, true
	}
	return "", false
}

// miseRunDesc renders a task's run body: a command string, or an array of
// commands joined as a shell line. ok is false when the table declares no
// usable run — the task may still be runnable through depends.
func miseRunDesc(run any) (string, bool) {
	switch body := run.(type) {
	case string:
		return body, true
	case []any:
		parts := make([]string, 0, len(body))
		for _, c := range body {
			s, isString := c.(string)
			if !isString {
				return "", false
			}
			if s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " && "), len(parts) > 0
	}
	return "", false
}

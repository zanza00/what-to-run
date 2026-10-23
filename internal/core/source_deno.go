// The deno.json source: Deno tasks.
package core

import (
	"bytes"
	"maps"
	"slices"
)

// denoSource returns the source for the project's deno.json or deno.jsonc,
// in that order: Deno looks for deno.json first, so it wins when both exist.
func denoSource() source {
	return source{
		Files: []string{"deno.json", "deno.jsonc"},
		Parse: parseDeno,
	}
}

// parseDeno returns the tasks declared in deno.json's tasks object, sorted by
// name, each run through "deno task". A task is a command string or an object
// with a command and an optional description; the description shows the
// command when the object does not declare one. An entry without a usable
// command is skipped.
func parseDeno(_ string, data []byte) ([]Task, error) {
	// An empty file declares no tasks: report it like a missing one
	// instead of failing on invalid JSON.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Tasks map[string]any `json:"tasks"`
	}
	if err := unmarshalJSONC(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		desc, ok := denoTaskDesc(doc.Tasks[name])
		if !ok {
			continue
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"deno", "task", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// denoTaskDesc turns one tasks value into the description of the task: a
// command string, or an object whose description overrides its command. ok is
// false when the value declares no usable command — an object command that is
// missing or not a string, an empty string, or neither shape — and the caller
// skips that task.
func denoTaskDesc(v any) (string, bool) {
	switch task := v.(type) {
	case string:
		return task, task != ""
	case map[string]any:
		cmd, isString := task["command"].(string)
		if !isString || cmd == "" {
			return "", false
		}
		if desc, isString := task["description"].(string); isString && desc != "" {
			return desc, true
		}
		return cmd, true
	}
	return "", false
}

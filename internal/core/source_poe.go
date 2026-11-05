// The poe source: tasks of Poe the Poet, from pyproject.toml or
// poe_tasks.toml.
package core

import (
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// poeSource returns the source for the pyproject.toml or poe_tasks.toml of the
// project, in that order. poe looks for pyproject.toml first, so that file
// wins when both exist. The source searches the parent directories, because
// poe walks up to find its config.
func poeSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "poe",
			Label:         "Poe tasks",
			Files:         []string{"pyproject.toml", "poe_tasks.toml"},
			SearchParents: true,
		},
		Parse: parsePoe,
	}
}

// parsePoe returns the tasks that the [tool.poe.tasks] table declares. Each
// task runs through "poe". The description comes from the task's help text
// and otherwise from the raw body, which is display text and never a promise
// of the executed command. Task names that start with an underscore stay
// hidden, because poe hides them from its own listing too. An entry that is
// not a task shape is skipped.
func parsePoe(_, name string, data []byte) ([]Task, error) {
	var doc struct {
		Tool struct {
			Poe struct {
				Tasks map[string]any `toml:"tasks"`
			} `toml:"poe"`
		} `toml:"tool"`
		Tasks map[string]any `toml:"tasks"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// pyproject.toml keeps the poe config only under tool.poe. A standalone
	// poe_tasks.toml makes the tool.poe namespace optional, so its tasks may
	// also sit at the top level.
	tasks := doc.Tool.Poe.Tasks
	if tasks == nil && name != "pyproject.toml" {
		tasks = doc.Tasks
	}

	out := make([]Task, 0, len(tasks))
	for task := range tasks {
		if strings.HasPrefix(task, "_") {
			continue
		}
		desc, ok := poeTaskDesc(tasks[task])
		if !ok {
			continue
		}
		out = append(out, Task{
			Name: task,
			Desc: desc,
			Args: []string{"poe", task},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// poeTypeKeys are the keys that type a poe task table, in the order of poe's
// own documentation. A table is typed by whichever key it holds, and the type
// accepts only one value shape: a line for the execution types and the ref
// type, an array of sub-tasks for the orchestration types.
var poeTypeKeys = [...]struct {
	key        string
	wantsArray bool
}{
	{"cmd", false},
	{"script", false},
	{"shell", false},
	{"expr", false},
	{"sequence", true},
	{"parallel", true},
	{"switch", true},
	{"ref", false},
}

// poeTaskDesc turns one task entry into its description, as display text. A
// string is a command, an array is a sequence of references, and a table is
// typed by its task-type key. The table's help text overrides the body. ok is
// false when the entry is not a task shape, or its type key holds a value of the
// wrong shape, and the caller skips that entry.
func poeTaskDesc(v any) (string, bool) {
	switch task := v.(type) {
	case string:
		return task, true
	case []any:
		return poeItemsDesc(task), true
	case map[string]any:
		for _, typeKey := range poeTypeKeys {
			body, typed := task[typeKey.key]
			if !typed {
				continue
			}
			desc, ok := poeBodyDesc(body, typeKey.wantsArray)
			if !ok {
				return "", false
			}
			help, _ := task["help"].(string)
			return helpOrBody(help, desc), true
		}
	}
	return "", false
}

// poeBodyDesc renders the body of a typed task table as display text. ok is
// false when the body holds the shape that its type does not accept, because
// poe rejects such a task and the caller skips the entry.
func poeBodyDesc(body any, wantsArray bool) (string, bool) {
	switch body := body.(type) {
	case string:
		if wantsArray {
			return "", false
		}
		return body, true
	case []any:
		if !wantsArray {
			return "", false
		}
		return poeItemsDesc(body), true
	}
	return "", false
}

// poeItemsDesc joins the items of an array task into one line of display text.
// The items are usually references to other tasks, and the line shows the
// order of the sequence. An inline sub-task has no one-line form, so an array
// that holds one renders no description instead of a partial list.
func poeItemsDesc(items []any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		s, isString := item.(string)
		if !isString {
			return ""
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " && ")
}

// The devbox.json source: Jetify devbox shell scripts.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"
)

// devboxSource returns the source for the project's devbox.json.
func devboxSource() source {
	return source{
		Files: []string{"devbox.json"},
		Parse: parseDevbox,
	}
}

// parseDevbox returns the scripts declared in devbox.json's shell.scripts
// object, sorted by name, each run through "devbox run". A script is a
// command string or an array of them; the description shows those commands.
// shell.init_hook and the rest of the file are ignored, and a script without
// usable commands is skipped.
func parseDevbox(_ string, data []byte) ([]Task, error) {
	// An empty file declares no scripts: report it like a missing one
	// instead of failing on invalid JSON.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Shell struct {
			Scripts map[string]any `json:"scripts"`
		} `json:"shell"`
	}
	if err := unmarshalJSONC(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Shell.Scripts))
	for _, name := range slices.Sorted(maps.Keys(doc.Shell.Scripts)) {
		desc, ok := devboxScriptDesc(doc.Shell.Scripts[name])
		if !ok {
			continue
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"devbox", "run", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// devboxScriptDesc turns one shell.scripts value into the description of the
// task: a command string, or an array of commands joined as a shell line. ok
// is false when the value declares no usable command — an empty string, an
// empty array, or neither — and the caller skips that script.
func devboxScriptDesc(v any) (string, bool) {
	switch cmds := v.(type) {
	case string:
		return cmds, cmds != ""
	case []any:
		parts := make([]string, 0, len(cmds))
		for _, c := range cmds {
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

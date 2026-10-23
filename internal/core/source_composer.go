// The composer.json source: Composer custom scripts.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"
)

// composerSource returns the source for the project's composer.json.
func composerSource() source {
	return source{
		Files: []string{"composer.json"},
		Parse: parseComposer,
	}
}

// parseComposer returns the custom scripts declared in composer.json, sorted
// by name, each run through "composer run-script". Composer's documented
// lifecycle and plugin events are left out: composer runs them itself at the
// matching moment, so they are not commands to start by hand. A script is a
// command string or an array of them; scripts-descriptions overrides the
// description, which otherwise shows those commands.
func parseComposer(_ string, data []byte) ([]Task, error) {
	// An empty file declares no scripts: report it like a missing one
	// instead of failing on invalid JSON.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Scripts             map[string]any `json:"scripts"`
		ScriptsDescriptions map[string]any `json:"scripts-descriptions"`
	}
	if err := unmarshalJSONC(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Scripts))
	for _, name := range slices.Sorted(maps.Keys(doc.Scripts)) {
		if composerLifecycleEvent(name) {
			continue
		}
		desc, ok := composerScriptDesc(doc.Scripts[name])
		if !ok {
			continue
		}
		if d, isString := doc.ScriptsDescriptions[name].(string); isString && d != "" {
			desc = d
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"composer", "run-script", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// composerScriptDesc turns one scripts value into the description of the
// task: a command string, or an array of commands joined as a shell line. ok
// is false when the value declares no usable command — an empty string, an
// empty array, or neither — and the caller skips that script.
func composerScriptDesc(v any) (string, bool) {
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

// composerLifecycleEvent reports whether name is one of Composer's
// documented lifecycle or plugin event names. A project may declare extra
// scratch scripts under those names, but composer also invokes them itself
// during install, update and the other commands, so the picker does not
// offer them as custom scripts.
func composerLifecycleEvent(name string) bool {
	switch name {
	case "pre-install-cmd",
		"post-install-cmd",
		"pre-update-cmd",
		"post-update-cmd",
		"pre-status-cmd",
		"post-status-cmd",
		"pre-archive-cmd",
		"post-archive-cmd",
		"pre-autoload-dump",
		"post-autoload-dump",
		"post-root-package-install",
		"post-create-project-cmd",
		"pre-operations-exec",
		"pre-package-install",
		"post-package-install",
		"pre-package-update",
		"post-package-update",
		"pre-package-uninstall",
		"post-package-uninstall",
		"init",
		"command",
		"pre-file-download",
		"post-file-download",
		"pre-command-run",
		"pre-pool-create":
		return true
	}
	return false
}

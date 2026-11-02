// The pipenv source: custom scripts of pipenv, from the [scripts] section of
// a Pipfile.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// pipenvSource returns the source for the Pipfile of the project. The source
// searches the parent directories, because pipenv walks up to find the
// Pipfile.
func pipenvSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "pipenv",
			Label:         "Pipenv scripts",
			Files:         []string{"Pipfile"},
			SearchParents: true,
		},
		Parse: parsePipenv,
	}
}

// parsePipenv returns the scripts that the [scripts] table of a Pipfile
// declares, sorted by name. Each script runs through "pipenv run", because
// the script needs the virtual environment of the project. A Pipfile has no
// description field, so the body of a script is its description, as display
// text and never a promise of the executed command. An entry that is not a
// script shape is skipped.
func parsePipenv(_, _ string, data []byte) ([]Task, error) {
	// An empty file declares no scripts, so report it like a missing file
	// instead of failing on invalid TOML.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Scripts map[string]any `toml:"scripts"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	out := make([]Task, 0, len(doc.Scripts))
	for _, name := range slices.Sorted(maps.Keys(doc.Scripts)) {
		desc, ok := pipenvScriptDesc(doc.Scripts[name])
		if !ok {
			continue
		}
		out = append(out, Task{
			Name: name,
			Desc: desc,
			Args: []string{"pipenv", "run", name},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// pipenvScriptDesc turns one script entry into its description, as display
// text. A string is one command, and an array is a sequence of commands that
// stops at the first failure, so the commands join with "&&". A table with
// one "call" key holds a Python reference. The pipenv documentation presents
// a "cmd" table that its parser rejects, so that table is listed too, with its
// raw string, because "pipenv run" runs the script either way. ok is false
// when the entry is not a script shape, or a table holds any other key, and
// the caller skips that entry.
func pipenvScriptDesc(v any) (string, bool) {
	switch script := v.(type) {
	case string:
		return script, true
	case []any:
		parts := make([]string, len(script))
		for i, item := range script {
			s, isString := item.(string)
			if !isString {
				return "", false
			}
			parts[i] = s
		}
		return strings.Join(parts, " && "), true
	case map[string]any:
		if len(script) == 1 {
			if call, isString := script["call"].(string); isString {
				return call, true
			}
		}
		if cmd, isString := script["cmd"].(string); isString {
			return cmd, true
		}
	}
	return "", false
}

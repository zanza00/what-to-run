// The pdm source: user scripts of PDM, from pyproject.toml.
package core

import (
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// pdmSource returns the source for the pyproject.toml of the project. The
// source searches the parent directories, because pdm walks up to find the
// pyproject.toml.
func pdmSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "pdm",
			Label:         "PDM scripts",
			Files:         []string{"pyproject.toml"},
			SearchParents: true,
		},
		Parse: parsePdm,
	}
}

// parsePdm returns the scripts that the [tool.pdm.scripts] table declares.
// Each script runs through "pdm run". The description comes from the script's
// help text and otherwise from the raw body, which is display text and never
// a promise of the executed command. An entry that is not a script shape is
// skipped.
func parsePdm(_, _ string, data []byte) ([]Task, error) {
	var doc struct {
		Tool struct {
			Pdm struct {
				Scripts map[string]any `toml:"scripts"`
			} `toml:"pdm"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	scripts := doc.Tool.Pdm.Scripts
	out := make([]Task, 0, len(scripts))
	for name := range scripts {
		// The "_" key holds the options shared by all scripts, and names with
		// an underscore prefix are internal. One prefix check hides both,
		// because pdm hides such scripts from its own listing too.
		if strings.HasPrefix(name, "_") {
			continue
		}
		desc, ok := pdmScriptDesc(scripts[name])
		if !ok {
			continue
		}
		out = append(out, Task{
			Name: name,
			Desc: desc,
			Args: []string{"pdm", "run", name},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// pdmTypeKeys are the keys that type a pdm script table, in the order of pdm's
// own documentation. A table is typed by whichever key it holds, and the other
// keys of the table are options, such as help, env or keep_going.
var pdmTypeKeys = [...]string{"cmd", "shell", "call", "composite"}

// pdmScriptDesc turns one script entry into its description, as display text.
// A string is a command, and a table is typed by its type key. The table's
// help text overrides the body. ok is false when the entry is not a script
// shape, or its type key holds a value of the wrong shape, and the caller
// skips that entry.
func pdmScriptDesc(v any) (string, bool) {
	switch script := v.(type) {
	case string:
		return script, true
	case map[string]any:
		for _, typeKey := range pdmTypeKeys {
			body, typed := script[typeKey]
			if !typed {
				continue
			}
			desc, ok := pdmBodyDesc(typeKey, body)
			if !ok {
				return "", false
			}
			help, _ := script["help"].(string)
			return helpOrBody(help, desc), true
		}
	}
	return "", false
}

// pdmBodyDesc renders the body of a typed script table as display text. A cmd
// body is one command, so it takes a string, or an array that pdm splits into
// one command. A shell body is one shell line, and a call body is one Python
// reference, so both take a string. A composite body is an array of other
// script names, and it renders like a sequence. ok is false when the body
// holds the shape that its type does not accept, because pdm rejects such a
// script and the caller skips the entry.
func pdmBodyDesc(typeKey string, body any) (string, bool) {
	switch typeKey {
	case "cmd":
		switch body := body.(type) {
		case string:
			return body, true
		case []any:
			return joinStringItems(body, " ")
		}
	case "shell", "call":
		if s, isString := body.(string); isString {
			return s, true
		}
	case "composite":
		items, isArray := body.([]any)
		if !isArray {
			return "", false
		}
		return joinStringItems(items, " && ")
	}
	return "", false
}

// The tox source: environments of tox, from the [tool.tox] tables of
// pyproject.toml or from a standalone tox.toml.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// toxSource returns the source for the pyproject.toml or tox.toml of the
// project, in that order. The source searches the parent directories, because
// tox walks up to find its configuration.
func toxSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "tox",
			Label:         "tox environments",
			Files:         []string{"pyproject.toml", "tox.toml"},
			SearchParents: true,
		},
		Parse: parseTox,
	}
}

// toxConfig is the part of the tox configuration that declares the
// environments: the env_list array and the env tables. The same content sits
// under [tool.tox] in a pyproject.toml and at the root of a tox.toml.
type toxConfig struct {
	EnvList any            `toml:"env_list"`
	Env     map[string]any `toml:"env"`
}

// parseTox returns the environments that the tox configuration declares,
// sorted by name: the tables under env and the names in env_list. Each
// environment runs through "tox run" with -e, because tox owns the
// environment, the substitutions and the lifecycle. The description comes
// from the environment's description and otherwise from its first command,
// which is display text and never a promise of the executed command. A name
// that contains template braces stays hidden, because it expands at run time.
// An entry that is not an environment shape is skipped.
func parseTox(_, name string, data []byte) ([]Task, error) {
	// An empty file declares no environments, so report it like a missing
	// file instead of failing on invalid TOML.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	// pyproject.toml keeps the tox configuration under [tool.tox]. A
	// standalone tox.toml holds the same tables at the root, without the
	// prefix.
	var conf toxConfig
	if name == "pyproject.toml" {
		var doc struct {
			Tool struct {
				Tox toxConfig `toml:"tox"`
			} `toml:"tool"`
		}
		if err := toml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		conf = doc.Tool.Tox
	} else {
		if err := toml.Unmarshal(data, &conf); err != nil {
			return nil, err
		}
	}

	names := make(map[string]bool)
	for envName, env := range conf.Env {
		if _, isTable := env.(map[string]any); isTable {
			names[envName] = true
		}
	}
	if list, isArray := conf.EnvList.([]any); isArray {
		for _, item := range list {
			if envName, isString := item.(string); isString {
				names[envName] = true
			}
		}
	}

	out := make([]Task, 0, len(names))
	for _, envName := range slices.Sorted(maps.Keys(names)) {
		if strings.ContainsAny(envName, "{}") {
			continue
		}
		var desc string
		if env, isTable := conf.Env[envName].(map[string]any); isTable {
			desc = toxEnvDesc(env)
		}
		out = append(out, Task{
			Name: envName,
			Desc: desc,
			Args: []string{"tox", "run", "-e", envName},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// toxEnvDesc returns the description of one environment, as display text. The
// environment's description comes first, and without one its first command
// shows. The base tables env_run_base and env_pkg_base describe no
// environment, so an environment that only env_list names shows no
// description.
func toxEnvDesc(env map[string]any) string {
	if desc, isString := env["description"].(string); isString && desc != "" {
		return desc
	}
	commands, isArray := env["commands"].([]any)
	if !isArray || len(commands) == 0 {
		return ""
	}
	desc, ok := toxCommandDesc(commands[0])
	if !ok {
		return ""
	}
	return desc
}

// toxCommandDesc renders one command of an environment as display text. A
// command is one line as a plain string, or one argv array. A replace table
// inside an argv shows as the substitution it stands for, such as {posargs}.
// ok is false when the command holds a shape that tox rejects, because the
// caller shows no description instead of a partial command.
func toxCommandDesc(cmd any) (string, bool) {
	switch cmd := cmd.(type) {
	case string:
		return cmd, true
	case []any:
		parts := make([]string, 0, len(cmd))
		for _, part := range cmd {
			switch part := part.(type) {
			case string:
				parts = append(parts, part)
			case map[string]any:
				name, isString := part["replace"].(string)
				if !isString {
					return "", false
				}
				parts = append(parts, "{"+name+"}")
			default:
				return "", false
			}
		}
		return strings.Join(parts, " "), true
	}
	return "", false
}

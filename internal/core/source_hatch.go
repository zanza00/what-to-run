// The hatch source: scripts of the environments of Hatch, from
// pyproject.toml.
package core

import (
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// hatchSource returns the source for the pyproject.toml of the project. The
// source searches the parent directories, because Hatch walks up to find the
// pyproject.toml.
func hatchSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "hatch",
			Label:         "Hatch scripts",
			Files:         []string{"pyproject.toml"},
			SearchParents: true,
		},
		Parse: parseHatch,
	}
}

// parseHatch returns the scripts that the environments of [tool.hatch.envs]
// declare, sorted by environment and then by script. Each script runs through
// "hatch run" with its environment, because a script only means something
// inside its environment: the task is named "<env>:<script>", like the run
// command, because two environments can declare the same script name. The
// description comes from the environment's description and otherwise from the
// raw script, which is display text and never a promise of the executed
// command. An entry that is not a script shape is skipped.
func parseHatch(_, _ string, data []byte) ([]Task, error) {
	var doc struct {
		Tool struct {
			Hatch struct {
				Envs map[string]any `toml:"envs"`
			} `toml:"hatch"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	envs := doc.Tool.Hatch.Envs
	var out []Task
	for _, envName := range slices.Sorted(maps.Keys(envs)) {
		env, isTable := envs[envName].(map[string]any)
		if !isTable {
			continue
		}
		if hatchEnvHidden(envName, env) {
			continue
		}
		scripts, isTable := env["scripts"].(map[string]any)
		if !isTable {
			continue
		}
		envDesc, hasEnvDesc := hatchEnvDesc(env)
		for _, scriptName := range slices.Sorted(maps.Keys(scripts)) {
			script, ok := hatchScriptDesc(scripts[scriptName])
			if !ok {
				continue
			}
			// The environment's description overrides the script, because
			// Hatch has no per-script help.
			desc := script
			if hasEnvDesc {
				desc = envDesc
			}
			name := envName + ":" + scriptName
			out = append(out, Task{
				Name: name,
				Desc: desc,
				Args: []string{"hatch", "run", name},
			})
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// hatchEnvHidden reports whether the picker must leave an environment out.
// Hatch reserves the "hatch-" name prefix for its own environments. A matrix
// table multiplies an environment into one environment per combination, and
// an overrides table mutates the options by the platform or the environment,
// so both move the effective scripts from the file to the run time.
func hatchEnvHidden(name string, env map[string]any) bool {
	if strings.HasPrefix(name, "hatch-") {
		return true
	}
	_, hasMatrix := env["matrix"]
	_, hasOverrides := env["overrides"]
	return hasMatrix || hasOverrides
}

// hatchEnvDesc returns the description of an environment, as display text.
// Hatch has no per-script help, so the description of an environment is the
// closest description of its scripts. ok is false when the environment
// declares no description, and the caller falls back to the raw script.
func hatchEnvDesc(env map[string]any) (string, bool) {
	desc, isString := env["description"].(string)
	if !isString || desc == "" {
		return "", false
	}
	return desc, true
}

// hatchScriptDesc turns one script entry into its description, as display
// text. A script is a string or an array of strings, and the commands of an
// array run sequentially, so they join like a chain. ok is false when the
// entry is not a script shape, because Hatch rejects such a script and the
// caller skips the entry.
func hatchScriptDesc(v any) (string, bool) {
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
	}
	return "", false
}

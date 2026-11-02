// The tox source: environments of tox, from the [tox] or [tox:tox] core
// section and the [testenv] sections of a tox.ini or setup.cfg, or from the
// [tool.tox] tables of a pyproject.toml or a standalone tox.toml.
package core

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// toxSource returns the source for the tox.ini, setup.cfg, pyproject.toml or
// tox.toml of the project, in that order. The source searches the parent
// directories, because tox walks up to find its configuration.
func toxSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "tox",
			Label:         "tox environments",
			Files:         []string{"tox.ini", "setup.cfg", "pyproject.toml", "tox.toml"},
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
// sorted by name. Each environment runs through "tox run" with -e, because
// tox owns the environment, the substitutions and the lifecycle. The
// description is display text and never a promise of the executed command. A
// name that contains template braces stays hidden, because it expands at run
// time. An entry that is not an environment shape is skipped. The winning
// file's name picks the format: tox.ini and setup.cfg hold the INI form, a
// pyproject.toml keeps the tables under [tool.tox] and a standalone tox.toml
// holds them at the root.
func parseTox(_, name string, data []byte) ([]Task, error) {
	// An empty file declares no environments, so report it like a missing
	// file instead of failing on the format.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	if name == "tox.ini" || name == "setup.cfg" {
		return parseToxIni(name, data)
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

// parseToxIni returns the environments that the INI form of the tox
// configuration declares: the names of the env_list key of the core section
// and the sections named [testenv:<name>]. The core section is [tox] in a
// tox.ini and [tox:tox] in a setup.cfg, and the key also appears in its
// legacy envlist spelling, which most pre-TOML projects use. The run command
// and the skipping rules are the ones of the TOML form, because both forms
// describe the same environments.
func parseToxIni(name string, data []byte) ([]Task, error) {
	sections, err := iniSections(data)
	if err != nil {
		return nil, err
	}

	// A setup.cfg hosts the sections of other tools too, so tox reads its
	// core section under the tox: prefix. The testenv sections keep the
	// plain spelling in both files.
	core := sections["tox"]
	if name == "setup.cfg" {
		core = sections["tox:tox"]
	}
	// The env_list key wins when a file holds both spellings, because the
	// tox reference documents that spelling first.
	envList := core["env_list"]
	if len(envList) == 0 {
		envList = core["envlist"]
	}

	names := make(map[string]bool)
	for _, line := range envList {
		for _, envName := range strings.Split(line, ",") {
			if envName = strings.TrimSpace(envName); envName != "" {
				names[envName] = true
			}
		}
	}
	for section := range sections {
		if envName, ok := strings.CutPrefix(section, "testenv:"); ok {
			names[envName] = true
		}
	}

	base := sections["testenv"]
	out := make([]Task, 0, len(names))
	for _, envName := range slices.Sorted(maps.Keys(names)) {
		if strings.ContainsAny(envName, "{}") {
			continue
		}
		out = append(out, Task{
			Name: envName,
			Desc: toxIniEnvDesc(sections["testenv:"+envName], base),
			Args: []string{"tox", "run", "-e", envName},
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// toxIniEnvDesc returns the description of one environment of the INI form,
// as display text. A run environment inherits every key that it does not
// declare, so the description falls back to the [testenv] base, and the
// first command falls back with it. The first command of the environment's
// own commands wins over the base one, because the environment overrides the
// base as a whole, key by key.
func toxIniEnvDesc(env, base map[string][]string) string {
	if desc := strings.Join(env["description"], " "); desc != "" {
		return desc
	}
	if desc := strings.Join(base["description"], " "); desc != "" {
		return desc
	}
	for _, commands := range [][]string{env["commands"], base["commands"]} {
		if len(commands) > 0 {
			return commands[0]
		}
	}
	return ""
}

// iniSections scans the INI format that tox.ini and setup.cfg use: section
// headers in brackets, key = value lines, and # or ; comment lines. A value
// continues on the lines that are indented deeper than its key, and a
// trailing backslash joins two physical lines into one logical line. The
// result holds the logical lines of every key, grouped by section. A line
// that fits none of these shapes is malformed structure, and the whole file
// fails, because a half-read file can hide environments.
func iniSections(data []byte) (map[string]map[string][]string, error) {
	sections := make(map[string]map[string][]string)
	var section map[string][]string
	var key string // the key that a deeper line continues
	var keyIndent int

	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		content := strings.TrimLeft(line, " \t")
		switch {
		case content == "", strings.HasPrefix(content, "#"), strings.HasPrefix(content, ";"):
			continue
		case key != "" && indent > keyIndent:
			section[key] = appendIniLine(section[key], content)
		case content[0] == '[':
			if !strings.HasSuffix(content, "]") {
				return nil, fmt.Errorf("line %d: section %q has no closing bracket", i+1, content)
			}
			name := strings.TrimSpace(content[1 : len(content)-1])
			if name == "" {
				return nil, fmt.Errorf("line %d: the section name is empty", i+1)
			}
			section = sections[name]
			if section == nil {
				section = make(map[string][]string)
				sections[name] = section
			}
			key, keyIndent = "", 0
		case section == nil:
			return nil, fmt.Errorf("line %d: %q is outside a section", i+1, content)
		default:
			sep := strings.IndexAny(content, "=:")
			if sep <= 0 {
				return nil, fmt.Errorf("line %d: %q is not a key = value line", i+1, content)
			}
			key = strings.TrimSpace(content[:sep])
			keyIndent = indent
			if value := strings.TrimSpace(content[sep+1:]); value != "" {
				section[key] = appendIniLine(section[key], value)
			}
		}
	}
	return sections, nil
}

// appendIniLine adds one physical line to the logical lines of a key. A
// trailing backslash continues the previous logical line, so the new line
// joins it with a single space.
func appendIniLine(lines []string, line string) []string {
	if n := len(lines); n > 0 && strings.HasSuffix(lines[n-1], `\`) {
		lines[n-1] = strings.TrimSpace(strings.TrimSuffix(lines[n-1], `\`)) + " " + line
		return lines
	}
	return append(lines, line)
}

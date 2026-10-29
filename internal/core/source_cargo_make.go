// The Makefile.toml source: cargo-make tasks.
package core

import (
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// cargoMakeSource returns the source for the Makefile.toml of the project.
func cargoMakeSource() source {
	return source{
		SourceInfo: SourceInfo{
			ID:    "cargo-make",
			Label: "Makefile.toml",
			Files: []string{"Makefile.toml"},
		},
		Parse: parseCargoMakefile,
	}
}

// cargoTask is the subset of a cargo-make task definition that the picker uses.
type cargoTask struct {
	Description string   `toml:"description"`
	Command     string   `toml:"command"`
	Args        []string `toml:"args"`
	Private     bool     `toml:"private"`
	Disabled    bool     `toml:"disabled"`
}

// parseCargoMakefile returns the cargo-make tasks that the [tasks] section of
// the project's Makefile.toml declares, sorted by name. It skips private and
// disabled tasks, because the picker could not run them directly.
func parseCargoMakefile(_ string, data []byte) ([]Task, error) {
	var doc struct {
		Tasks map[string]cargoTask `toml:"tasks"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		t := doc.Tasks[name]
		if t.Private || t.Disabled {
			continue
		}
		desc := t.Description
		if desc == "" && t.Command != "" {
			desc = strings.Join(append([]string{t.Command}, t.Args...), " ")
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"cargo", "make", name},
		})
	}
	return tasks, nil
}

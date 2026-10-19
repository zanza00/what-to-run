// The Makefile.toml source: cargo-make tasks.
package tui

import (
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// cargoMakeSource returns the source for the project's Makefile.toml.
func cargoMakeSource() source {
	return source{
		Files: []string{"Makefile.toml"},
		Parse: parseCargoMakefile,
	}
}

// cargoTask is the subset of a cargo-make task definition the picker uses.
type cargoTask struct {
	Description string   `toml:"description"`
	Command     string   `toml:"command"`
	Args        []string `toml:"args"`
	Private     bool     `toml:"private"`
	Disabled    bool     `toml:"disabled"`
}

// parseCargoMakefile returns the cargo-make tasks declared in the project's
// Makefile.toml [tasks] section, sorted by name. Private and disabled tasks
// are skipped: the picker would not be able to run them directly.
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

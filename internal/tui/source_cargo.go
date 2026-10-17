// The Cargo.toml source: cargo-make tasks.
package tui

import (
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// cargoMakeSource returns the source for the project's Cargo.toml.
func cargoMakeSource() source {
	return source{
		Files: []string{"Cargo.toml"},
		Parse: parseCargoToml,
	}
}

// cargoTask is the subset of a cargo-make task definition the picker uses.
type cargoTask struct {
	Description string   `toml:"description"`
	Command     string   `toml:"command"`
	Args        []string `toml:"args"`
}

// parseCargoToml returns the cargo-make tasks declared in the project's
// Cargo.toml [tasks] section, sorted by name.
func parseCargoToml(_ string, data []byte) ([]Task, error) {
	var doc struct {
		Tasks map[string]cargoTask `toml:"tasks"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(doc.Tasks))
	for _, name := range slices.Sorted(maps.Keys(doc.Tasks)) {
		t := doc.Tasks[name]
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

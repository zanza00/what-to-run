// The .cargo/config source: cargo aliases.
package core

import (
	"bytes"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
)

// cargoAliasSource returns the source for the project's .cargo/config or
// .cargo/config.toml. Cargo prefers the extensionless file when both exist
// (and warns about it), so it is the first candidate.
func cargoAliasSource() source {
	return source{
		SourceInfo: SourceInfo{
			Label:               "Cargo aliases",
			DisplayFileOverride: ".cargo/config.toml",
			Files:               []string{".cargo/config", ".cargo/config.toml"},
		},
		Parse: parseCargoConfig,
	}
}

// parseCargoConfig returns the aliases declared in the project's cargo config
// [alias] table, sorted by name, each run as "cargo <name>". The description
// shows the alias expansion, with alias chains followed the way cargo does
// (zellij's q -> x -> xtask); shell aliases ("!...") are shown verbatim. An
// entry that is not a string, or whose expansion is empty, is skipped: cargo
// would not run it either.
func parseCargoConfig(_ string, data []byte) ([]Task, error) {
	// An empty file declares no aliases: report it like a missing one
	// instead of failing on invalid TOML.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var doc struct {
		Alias map[string]any `toml:"alias"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	aliases := make(map[string]string, len(doc.Alias))
	for name, value := range doc.Alias {
		if expansion, ok := value.(string); ok {
			aliases[name] = expansion
		}
	}

	tasks := make([]Task, 0, len(aliases))
	for _, name := range slices.Sorted(maps.Keys(aliases)) {
		desc := cargoAliasDesc(name, aliases)
		if desc == "" {
			continue
		}
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"cargo", name},
		})
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks, nil
}

// cargoAliasDesc renders what an alias expands to, following alias chains
// (zellij's q -> x -> xtask) the way cargo does: only the first word of the
// expansion is looked up as an alias, and the rest is carried along. The
// chain stops at a shell alias ("!...") or at a cycle, which cargo rejects;
// the bound on the loop is the number of aliases, so a cycle cannot spin.
func cargoAliasDesc(name string, aliases map[string]string) string {
	desc := strings.TrimSpace(aliases[name])
	seen := map[string]bool{name: true}
	for range aliases {
		if desc == "" || strings.HasPrefix(desc, "!") {
			break
		}
		first, rest := cutFirstField(desc)
		next, ok := aliases[first]
		if !ok || seen[first] {
			break
		}
		seen[first] = true
		next = strings.TrimSpace(next)
		if next == "" {
			break
		}
		desc = next
		if rest != "" {
			desc += " " + rest
		}
	}
	return desc
}

// cutFirstField splits s at its first whitespace run: the first field and the
// trimmed remainder, which is empty when s is a single field.
func cutFirstField(s string) (first, rest string) {
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i:])
}

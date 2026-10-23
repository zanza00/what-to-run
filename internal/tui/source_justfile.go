// The justfile source: just recipes.
package tui

import (
	"regexp"
	"slices"
	"strings"
)

// justfileSource returns the source for just recipes: the first of the three
// file names just itself looks for, in that order.
func justfileSource() source {
	return source{
		Files: []string{"justfile", "Justfile", ".justfile"},
		Parse: parseJustfile,
	}
}

// justRecipeRe matches a recipe header at the start of a line: an optional
// "@" marking a silent recipe, the recipe name, then anything that is not a
// colon (parameters and dependencies) up to the colon that opens the recipe.
var justRecipeRe = regexp.MustCompile(`^@?([A-Za-z_][A-Za-z0-9_-]*)[^:]*:`)

// justModuleRe matches the import and mod keywords, including their optional
// "?" forms, which take a path up to a space.
var justModuleRe = regexp.MustCompile(`^(import|mod)\?? `)

// parseJustfile returns the recipes declared by a justfile, sorted by name
// like the ungrouped part of "just --list", each run as "just <name>".
//
// The parser is line-based and forgiving. It recognizes a recipe header at
// column 0, keeps the last contiguous "#" comment line above it or its
// "[doc(...)]" attribute as the description, and hides private recipes: the
// "_" prefix and the "[private]" attribute. A leading "#!" is a shebang, not
// a doc comment. Lines inside recipe bodies are ignored.
//
// Settings, variable assignments, aliases, exports, imports and modules are
// skipped. Imports and modules are not followed, so a justfile whose recipes
// live in another file shows none of them; aliases are not listed as extra
// rows. Groups are ignored: the picker shows one flat alphabetical list,
// which orders grouped recipes differently from "just --list". Triple-quoted
// strings can contain a column-0 line that looks like a recipe header, which
// the line-based parser cannot tell from a real one.
func parseJustfile(_ string, data []byte) ([]Task, error) {
	var (
		tasks   []Task
		seen    = map[string]bool{}
		comment string // last contiguous "#" line
		docAttr string // "[doc(...)]" attribute
		private bool   // "[private]" attribute
	)

	reset := func() {
		comment, docAttr, private = "", "", false
	}

	first := true
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")

		// A shebang describes the interpreter, not the recipe below it.
		if first {
			first = false
			if strings.HasPrefix(line, "#!") {
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			reset()
			continue
		}
		if rest, ok := strings.CutPrefix(line, "#"); ok {
			comment = strings.TrimSpace(rest)
			continue
		}
		if strings.HasPrefix(line, "[") {
			if doc, ok := justDocAttribute(line); ok {
				docAttr = doc
			} else if strings.TrimSpace(line) == "[private]" {
				private = true
			}
			// Other attributes, like [group("...")], leave the pending
			// description and privacy alone.
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			continue // a line inside a recipe body
		}
		// Settings, assignments, aliases, exports, imports and modules are
		// checked before the recipe regex: an import path or an assignment
		// containing a colon must not be read as a recipe header.
		if isJustSetting(line) {
			reset()
			continue
		}

		m := justRecipeRe.FindStringSubmatch(line)
		if m == nil {
			reset()
			continue
		}
		name := m[1]
		desc := docAttr
		if desc == "" {
			desc = comment
		}
		skip := private || strings.HasPrefix(name, "_") || seen[name]
		// just clears the pending attributes after every recipe header, even
		// with no blank line between two recipes: a [doc] or [private] recipe
		// must not leak its description or privacy onto the next one.
		reset()
		if skip {
			continue
		}
		seen[name] = true
		tasks = append(tasks, Task{
			Name: name,
			Desc: desc,
			Args: []string{"just", name},
		})
	}

	if len(tasks) == 0 {
		return nil, nil
	}
	slices.SortFunc(tasks, func(a, b Task) int { return strings.Compare(a.Name, b.Name) })
	return tasks, nil
}

// isJustSetting reports whether the line declares a setting, an assignment, an
// alias, an export, an import or a module rather than a recipe.
func isJustSetting(line string) bool {
	for _, prefix := range []string{"set ", "alias ", "export "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return justModuleRe.MatchString(line) || strings.Contains(line, ":=")
}

// justDocAttribute returns the text of a "[doc(...)]" attribute line.
func justDocAttribute(line string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), "[doc(")
	if !ok {
		return "", false
	}
	rest, ok = strings.CutSuffix(rest, ")]")
	if !ok {
		return "", false
	}

	doc := strings.TrimSpace(rest)
	if len(doc) < 2 || (doc[0] != '"' && doc[0] != '\'') || doc[len(doc)-1] != doc[0] {
		return "", false
	}
	return doc[1 : len(doc)-1], true
}

// The Makefile source: make targets.
package core

import (
	"regexp"
	"strings"
)

// makeSource returns the source for make targets: the first of GNUmakefile,
// makefile, Makefile that exists, which is the order make itself uses.
func makeSource() source {
	return source{
		SourceInfo: SourceInfo{
			Label:               "Makefile",
			DisplayFileOverride: "Makefile",
			Files:               []string{"GNUmakefile", "makefile", "Makefile"},
		},
		Parse: parseMakeFile,
	}
}

// parseMakeFile adapts parseMakeTargets to the source parser signature.
func parseMakeFile(_ string, data []byte) ([]Task, error) {
	return parseMakeTargets(data), nil
}

// makeTargetRe matches a Makefile target line: a name at the start of the
// line followed by one or two colons, as in "build: ## Build everything".
var makeTargetRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.+-]*)[ \t]*:{1,2}[ \t]*(.*)$`)

// parseMakeTargets extracts the targets from a Makefile.
//
// The parser is intentionally shallow: it recognizes "name: deps" and
// "name:: deps" lines at the start of a line, skips variable assignments
// ("NAME := value") and special targets (".PHONY", pattern rules), and uses
// the contiguous comment directly above a target as its description. The
// common "## name: text" and "## name - text" conventions (`make help`
// style) are cleaned up to just the text.
func parseMakeTargets(data []byte) []Task {
	var (
		tasks   []Task
		seen    = map[string]bool{}
		comment string
	)

	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")

		if rest, ok := strings.CutPrefix(line, "#"); ok {
			comment = rest
			continue
		}
		if strings.TrimSpace(line) == "" {
			comment = ""
			continue
		}

		m := makeTargetRe.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(m[2], "=") { // variable assignment (NAME := value)
			comment = ""
			continue
		}
		name := m[1]
		if seen[name] {
			comment = ""
			continue
		}
		seen[name] = true
		tasks = append(tasks, Task{
			Name: name,
			Desc: makeDescription(name, comment),
			Args: []string{"make", name},
		})
		comment = ""
	}
	return tasks
}

// makeDescription derives a task description from the comment above a
// target, stripping the "name: text" and "name - text" help prefixes.
func makeDescription(name, comment string) string {
	c := strings.TrimSpace(strings.TrimLeft(comment, "#"))
	if rest, ok := strings.CutPrefix(c, name+":"); ok {
		return strings.TrimSpace(rest)
	}
	if rest, ok := strings.CutPrefix(c, name+" - "); ok {
		return strings.TrimSpace(rest)
	}
	return c
}

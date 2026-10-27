package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// The docs live at the repository root, two levels above the binary package.
const (
	taskSourcesDocPath = "../../docs/task-sources.md"
	readmePath         = "../../README.md"
)

// backtickedName matches a name wrapped in backticks inside a table cell.
var backtickedName = regexp.MustCompile("`([^`]+)`")

// bulletFileSuffix matches the optional file after a README label, as in
// "Cargo aliases (`.cargo/config`)".
var bulletFileSuffix = regexp.MustCompile(" \\(`[^`]+`\\)$")

// TestTaskSourcesTableMatchesRegistry checks that docs/task-sources.md has one
// row per source, in registry order, listing exactly the source's files.
func TestTaskSourcesTableMatchesRegistry(t *testing.T) {
	const docName = "docs/task-sources.md"

	rows := sourceTableRows(t, readDoc(t, taskSourcesDocPath, docName), docName, "File")
	sources := core.Sources()

	for i, src := range sources {
		if i >= len(rows) {
			t.Errorf("%s: no row for source %q (files %v); add one", docName, src.Label, src.Files)
			continue
		}
		if !sameNames(rows[i], src.Files) {
			t.Errorf("%s: row %d for source %q lists %v, want %v; edit that row", docName, i+1, src.Label, rows[i], src.Files)
		}
	}
	if extra := len(rows) - len(sources); extra > 0 {
		t.Errorf("%s: %d extra row(s) after the %d source rows (first lists %v); remove them",
			docName, extra, len(sources), rows[len(sources)])
	}
}

// TestReadmeTaskSourcesMatchRegistry checks that the README bullet list names
// every source label exactly, with one bullet per source.
func TestReadmeTaskSourcesMatchRegistry(t *testing.T) {
	const docName = "README.md"

	section, ok := markdownSection(readDoc(t, readmePath, docName), "## Task sources")
	if !ok {
		t.Fatalf("%s: no %q section", docName, "## Task sources")
	}

	sources := core.Sources()
	known := make(map[string]bool, len(sources))
	for _, src := range sources {
		known[src.Label] = true
	}

	bullets := 0
	seen := make(map[string]bool)
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		bullets++
		label := bulletLabel(strings.TrimPrefix(line, "- "))
		switch {
		case !known[label]:
			t.Errorf("%s: bullet %q matches no source label", docName, line)
		case seen[label]:
			t.Errorf("%s: duplicate bullet for source %q", docName, label)
		default:
			seen[label] = true
		}
	}

	if bullets != len(sources) {
		t.Errorf("%s: %d task-source bullets, want %d (one per source)", docName, bullets, len(sources))
	}
	for _, src := range sources {
		if !seen[src.Label] {
			t.Errorf("%s: no bullet for source %q (label %q); add one", docName, src.Label, src.Label)
		}
	}
}

// TestBulletLabel pins the two allowances of the README check: whole-label
// backticks are stripped, and the parenthetical file after a label is ignored.
func TestBulletLabel(t *testing.T) {
	tests := []struct {
		name   string
		bullet string
		want   string
	}{
		{name: "backticked file", bullet: "`package.json`", want: "package.json"},
		{name: "backticked longer file", bullet: "`Makefile.toml`", want: "Makefile.toml"},
		{name: "plain label with file", bullet: "Cargo aliases (`.cargo/config`)", want: "Cargo aliases"},
		{name: "prose after the label", bullet: "`Makefile` (see below)", want: "`Makefile` (see below)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bulletLabel(tt.bullet); got != tt.want {
				t.Errorf("bulletLabel(%q) = %q, want %q", tt.bullet, got, tt.want)
			}
		})
	}
}

// readDoc reads a document relative to the package directory, naming it as
// docName in failures.
func readDoc(t *testing.T, path, docName string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		t.Fatalf("%s: %v", docName, err)
	}
	return string(data)
}

// sourceTableRows returns the names in the named column of the first Markdown
// table whose header has that column. It reads data rows until the first line
// that does not start with "|", so a table later in the document is ignored.
func sourceTableRows(t *testing.T, doc, docName, column string) [][]string {
	t.Helper()

	lines := strings.Split(doc, "\n")
	header, columnIndex := -1, -1
	for i, line := range lines {
		if i+1 >= len(lines) || !isTableSeparator(tableCells(lines[i+1])) {
			continue
		}
		if j := slices.Index(tableCells(line), column); j >= 0 {
			header, columnIndex = i, j
			break
		}
	}
	if header < 0 {
		t.Fatalf("%s: no table header with a %q column", docName, column)
	}

	var rows [][]string
	for _, line := range lines[header+1:] {
		if !strings.HasPrefix(line, "|") {
			break
		}
		cells := tableCells(line)
		if isTableSeparator(cells) {
			continue
		}
		if columnIndex >= len(cells) {
			t.Fatalf("%s: row %q has no %q cell", docName, line, column)
		}
		rows = append(rows, backtickedNames(cells[columnIndex]))
	}
	return rows
}

// tableCells splits a Markdown table row into trimmed cells.
func tableCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	cells := strings.Split(line[1:len(line)-1], "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// isTableSeparator reports whether cells form the separator under a Markdown
// table header, such as "| --- | --- |".
func isTableSeparator(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if strings.Contains(cell, "-") && strings.Trim(cell, ":-") == "" {
			continue
		}
		return false
	}
	return true
}

// backtickedNames returns the names wrapped in backticks in cell.
func backtickedNames(cell string) []string {
	matches := backtickedName.FindAllStringSubmatch(cell, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

// sameNames reports whether got and want hold the same names, regardless of
// order.
func sameNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(got))
	for _, name := range got {
		counts[name]++
	}
	for _, name := range want {
		if counts[name] == 0 {
			return false
		}
		counts[name]--
	}
	return true
}

// bulletLabel returns a README bullet's text with the whole-label backticks
// stripped and the optional trailing file dropped, as in "Cargo aliases
// (`.cargo/config`)".
func bulletLabel(bullet string) string {
	bullet = strings.TrimSpace(bullet)
	if len(bullet) >= 2 && strings.HasPrefix(bullet, "`") && strings.HasSuffix(bullet, "`") {
		bullet = bullet[1 : len(bullet)-1]
	}
	return bulletFileSuffix.ReplaceAllString(bullet, "")
}

// markdownSection returns the lines below heading, up to the next level-two
// heading.
func markdownSection(doc, heading string) (string, bool) {
	lines := strings.Split(doc, "\n")
	start := slices.Index(lines, heading)
	if start < 0 {
		return "", false
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start+1:end], "\n"), true
}

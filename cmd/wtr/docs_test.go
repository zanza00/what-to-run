package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// taskSourcesDoc is the user-facing table of the files each source reads.
const taskSourcesDoc = "../../docs/task-sources.md"

// readmeDoc is the short list of task sources in the project README.
const readmeDoc = "../../README.md"

var backtickedName = regexp.MustCompile("`([^`]+)`")

// TestTaskSourcesTableMatchesRegistry: every built-in source has one row, in
// registry order, and the row lists exactly the source's candidate files.
func TestTaskSourcesTableMatchesRegistry(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(taskSourcesDoc))
	if err != nil {
		t.Fatal(err)
	}

	rows := tableRows(t, string(data), "File")
	sources := core.Sources()
	if len(rows) != len(sources) {
		t.Fatalf("docs/task-sources.md: %d rows, want %d (one per source)", len(rows), len(sources))
	}

	for i, src := range sources {
		if i >= len(rows) {
			t.Fatalf("docs/task-sources.md: no row for source %q (files %v)", src.Label, src.Files)
		}
		if !sameSet(rows[i].files, src.Files) {
			t.Errorf("docs/task-sources.md: row for %q lists %v, want %v", src.Label, rows[i].files, src.Files)
		}
	}
}

// TestReadmeTaskSourcesMatchRegistry: the README names every source label and
// has one bullet per source, so a removed source cannot leave a stale bullet.
func TestReadmeTaskSourcesMatchRegistry(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(readmeDoc))
	if err != nil {
		t.Fatal(err)
	}

	section, ok := markdownSection(string(data), "## Task sources")
	if !ok {
		t.Fatal("README.md: no ## Task sources section")
	}

	sources := core.Sources()
	bullets := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "- ") {
			bullets++
		}
	}
	if bullets != len(sources) {
		t.Errorf("README.md: %d task-source bullets, want %d", bullets, len(sources))
	}

	for _, src := range sources {
		if !strings.Contains(section, src.Label) {
			t.Errorf("README.md: no text for source %q (label %q)", src.Label, src.Label)
		}
	}
}

// tableRows returns the data rows of the first Markdown table whose header has
// the named column, extracting the backticked names from that column.
func tableRows(t *testing.T, doc, column string) []tableRow {
	t.Helper()

	lines := strings.Split(doc, "\n")
	header, columnIndex := -1, -1
	for i, line := range lines {
		cells := tableCells(line)
		for j, cell := range cells {
			if cell == column {
				header, columnIndex = i, j
				break
			}
		}
		if header >= 0 {
			break
		}
	}
	if header < 0 {
		t.Fatalf("docs/task-sources.md: no table header with a %q column", column)
	}

	var rows []tableRow
	for _, line := range lines[header+1:] {
		if !strings.HasPrefix(line, "| `") {
			if strings.HasPrefix(line, "|") {
				continue // the header separator
			}
			break
		}
		cells := tableCells(line)
		if columnIndex >= len(cells) {
			t.Fatalf("docs/task-sources.md: row %q has no %q cell", line, column)
		}
		rows = append(rows, tableRow{files: backtickedNames(cells[columnIndex])})
	}
	return rows
}

type tableRow struct {
	files []string
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

// backtickedNames returns the names wrapped in backticks in cell.
func backtickedNames(cell string) []string {
	matches := backtickedName.FindAllStringSubmatch(cell, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

// sameSet reports whether got and want hold the same names, regardless of
// order.
func sameSet(got, want []string) bool {
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

// markdownSection returns the lines below heading, up to the next level-two
// heading.
func markdownSection(doc, heading string) (string, bool) {
	lines := strings.Split(doc, "\n")
	start := -1
	for i, line := range lines {
		if line == heading {
			start = i
			break
		}
	}
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

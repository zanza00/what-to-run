// Task discovery: the runnable scripts a project declares.
//
// A source is one family of project files the picker understands. Each
// source owns the file names it looks for and a parser that turns one
// file's bytes into tasks; the framework below finds the first existing
// candidate file, stamps the tasks with that file name as their badge, and
// turns a failed parse into a warning instead of losing the tasks of the
// other sources.
//
// The built-in sources, in this fixed order:
//
//	package.json   npm/pnpm/yarn/bun scripts
//	Makefile       make targets (also makefile, GNUmakefile)
//	Cargo.toml     cargo-make tasks
//
// A source that is missing contributes nothing; a source that exists but
// fails to parse is reported in errs and skipped, so one broken file does
// not hide the tasks of the others.

package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// source is one family of project files. Adding a source is a literal in
// defaultSources plus a parser function; the framework does the rest.
type source struct {
	// Files are candidate base names in preference order: the first that
	// exists in the project directory is parsed. Its name becomes the
	// badge of the tasks the source returns.
	Files []string

	// Parse turns the winning file's content into tasks: it receives the
	// project directory for sibling lookups (lock files) and the file's
	// bytes. Task.Source is stamped by the framework, not by the parser.
	Parse func(dir string, data []byte) ([]Task, error)
}

// defaultSources returns the built-in sources in display order.
func defaultSources() []source {
	return []source{
		packageJSONSource(),
		makeSource(),
		cargoMakeSource(),
	}
}

// discoverSource runs one source over dir: the first candidate file that
// exists is parsed and its tasks are stamped with the file name.
func discoverSource(src source, dir string) ([]Task, error) {
	for _, name := range src.Files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		tasks, err := src.Parse(dir, data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for i := range tasks {
			tasks[i].Source = name
		}
		return tasks, nil
	}
	return nil, nil
}

// discoverTasks scans dir with the built-in sources, merging them in a
// stable order: package.json, Makefile, Cargo.toml. Scripts and cargo-make
// tasks are sorted by name; Makefile targets keep the order they appear in
// the file.
func discoverTasks(dir string) (tasks []Task, errs []error) {
	for _, src := range defaultSources() {
		ts, err := discoverSource(src, dir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tasks = append(tasks, ts...)
	}
	return tasks, errs
}

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
//	Taskfile.yml   go-task tasks (also taskfile.yml, Taskfile.dist.yml)
//	justfile       just recipes (also Justfile, .justfile)
//	Makefile.toml  cargo-make tasks
//	.cargo/config  cargo aliases (also .cargo/config.toml)
//	devbox.json    devbox shell scripts
//	composer.json  Composer custom scripts
//	deno.json      Deno tasks (also deno.jsonc)
//	mise.toml      mise tasks (also .mise.toml)
//
// The rc file (decoded by the caller) sits on top: its tasks are shown
// first and shadow same-named tasks of the sources, and a source can be
// disabled by the names of the files it looks for. A source that is missing
// contributes nothing; a source that exists but fails to parse is reported in
// errs and skipped, so one broken file does not hide the tasks of the others.

package core

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
)

// source is one family of project files. Adding a source is a literal in
// defaultSources plus a parser function; the framework does the rest.
type source struct {
	// Files are candidate paths in preference order, relative to the
	// project directory: the first that exists is parsed. Its name becomes
	// the badge of the tasks the source returns.
	Files []string

	// Parse turns the winning file's content into tasks: it receives the
	// project directory for sibling lookups (lock files) and the file's
	// bytes. Task.Source is stamped by the framework, not by the parser.
	Parse func(dir string, data []byte) ([]Task, error)
}

// Settings is the part of the rc file that reaches task discovery.
type Settings struct {
	// PackageManager overrides the runner used for package.json scripts
	// (npm, pnpm, yarn, bun); empty means autodetect.
	PackageManager string

	// Disable names the sources to skip, by the file names they look for.
	Disable []string

	// Tasks are the rc file's custom tasks, shown above the discovered ones.
	Tasks []RCTask
}

// RCTask is one custom task declared in the rc file.
type RCTask struct {
	Name        string
	Command     string
	Description string
	Cwd         string
}

// defaultSources returns the built-in sources in display order, wired with
// the rc file settings that reach into them.
func defaultSources(settings Settings, files Files) []source {
	return []source{
		packageJSONSource(settings.PackageManager, files),
		makeSource(),
		taskfileSource(),
		justfileSource(),
		cargoMakeSource(),
		cargoAliasSource(),
		devboxSource(),
		composerSource(),
		denoSource(),
		miseSource(),
	}
}

// discoverSource runs one source over dir: the first candidate file that
// exists is parsed and its tasks are stamped with the file name.
func discoverSource(files Files, src source, dir string) ([]Task, error) {
	for _, name := range src.Files {
		data, err := files.ReadFile(filepath.Join(dir, name))
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

// enabledSources drops the sources named in disable and reports the entries
// that match no source file (a typo in the rc file). An entry disables a
// source when it matches any of the file names the source looks for.
func enabledSources(sources []source, disable []string) ([]source, []error) {
	if len(disable) == 0 {
		return sources, nil
	}

	var errs []error
	known := make(map[string]bool)
	for _, src := range sources {
		for _, name := range src.Files {
			known[name] = true
		}
	}
	for _, name := range disable {
		if !known[name] {
			errs = append(errs, fmt.Errorf("rc file: unknown source %q in the disable list", name))
		}
	}

	kept := make([]source, 0, len(sources))
	for _, src := range sources {
		if slices.ContainsFunc(src.Files, func(name string) bool { return slices.Contains(disable, name) }) {
			continue
		}
		kept = append(kept, src)
	}
	return kept, errs
}

// rcTasks converts the rc file's custom tasks into picker tasks: they show
// the "rc" badge, run through the shell, and a relative cwd is resolved
// against the project directory.
func rcTasks(settings Settings, dir string) []Task {
	var tasks []Task
	for _, t := range settings.Tasks {
		desc := t.Description
		if desc == "" {
			desc = t.Command
		}
		task := Task{
			Source: "rc",
			Name:   t.Name,
			Desc:   desc,
			Args:   []string{"sh", "-c", t.Command},
		}
		if t.Cwd != "" {
			task.Dir = filepath.Join(dir, t.Cwd)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// Discover scans dir for the tasks of the rc file and the built-in
// sources, in display order: rc tasks first, then package.json, Makefile,
// Taskfile.yml, justfile, Makefile.toml, .cargo/config, devbox.json,
// composer.json, deno.json/deno.jsonc and mise.toml/.mise.toml. Package
// scripts, Taskfile tasks, just recipes, cargo-make tasks, cargo aliases,
// devbox scripts, Composer scripts, Deno tasks and mise tasks are sorted by
// name; Makefile targets keep the order they appear in the file. A source that
// fails to parse is reported in errs and skipped (a disabled source is not read
// at all), and rc tasks shadow same-named tasks of the built-in sources.
func Discover(files Files, dir string, settings Settings) ([]Task, []error) {
	sources, errs := enabledSources(defaultSources(settings, files), settings.Disable)

	rc := rcTasks(settings, dir)
	shadowed := make(map[string]bool, len(rc))
	for _, t := range rc {
		shadowed[t.Name] = true
	}

	var tasks []Task
	tasks = append(tasks, rc...)
	for _, src := range sources {
		ts, err := discoverSource(files, src, dir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, t := range ts {
			if !shadowed[t.Name] {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks, errs
}

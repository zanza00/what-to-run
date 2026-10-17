// The rc file: optional settings that extend or patch what the built-in
// sources discover.
//
// Two files are read, if present: the project's .what-to-run.toml and
// the user's $XDG_CONFIG_HOME/what-to-run/config.toml. The project
// file wins on package_manager, its disable entries add to the global ones,
// and its tasks shadow same-named global tasks. Without either file the zero
// config keeps the built-in behaviour.
//
// Problems are warnings, never fatal: a file that cannot be read or parsed
// is skipped, an invalid entry is dropped, and everything else still applies.

package tui

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// config is the merged content of the rc files.
type config struct {
	// PackageManager forces the runner used for package.json scripts:
	// "npm", "pnpm", "yarn" or "bun". Empty means detection as usual.
	PackageManager string `toml:"package_manager"`

	// Disable names the sources to skip, by the file names they look for
	// ("package.json", "Makefile", "Cargo.toml").
	Disable []string `toml:"disable"`

	// Tasks are custom tasks, shown above the discovered ones.
	Tasks []configTask `toml:"tasks"`
}

// configTask is one [[tasks]] entry of an rc file.
type configTask struct {
	Name        string `toml:"name"`        // required
	Command     string `toml:"command"`     // required; run with sh -c
	Description string `toml:"description"` // defaults to Command
	Cwd         string `toml:"cwd"`         // optional, relative to the project
}

// defaultConfigFiles returns the project and user rc file locations for a
// project directory. The user file is empty when the home config directory
// cannot be resolved.
func defaultConfigFiles(dir string) (project, global string) {
	project = filepath.Join(dir, ".what-to-run.toml")
	if configDir, err := os.UserConfigDir(); err == nil {
		global = filepath.Join(configDir, "what-to-run", "config.toml")
	}
	return project, global
}

// loadConfig reads the rc files and returns the merged settings: the project
// file wins on package_manager and its tasks come first, a disable entry
// adds to the list, a task shadows the same-named one from a more general
// file. Missing files are fine; a file that cannot be read or parsed is
// reported as a warning and skipped.
func loadConfig(projectFile, globalFile string) (config, []error) {
	global, errs := readConfig(globalFile)
	project, projectErrs := readConfig(projectFile)
	errs = append(errs, projectErrs...)

	cfg := global
	if project.PackageManager != "" {
		cfg.PackageManager = project.PackageManager
	}
	cfg.Disable = append(cfg.Disable, project.Disable...)
	// Project tasks come first, so they shadow same-named global ones.
	cfg.Tasks = firstTasks(append(project.Tasks, cfg.Tasks...))
	return cfg, errs
}

// firstTasks keeps the first task of each name.
func firstTasks(tasks []configTask) []configTask {
	seen := make(map[string]bool, len(tasks))
	var kept []configTask
	for _, t := range tasks {
		if seen[t.Name] {
			continue
		}
		seen[t.Name] = true
		kept = append(kept, t)
	}
	return kept
}

// readConfig reads one rc file. An empty path and a missing file yield the
// zero config. A file-level problem is returned as a warning with an empty
// config; invalid entries are dropped with a warning each and the rest of
// the file still applies.
func readConfig(path string) (config, []error) {
	if path == "" {
		return config{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return config{}, nil
	}
	if err != nil {
		return config{}, []error{fmt.Errorf("rc file %s: %w", path, err)}
	}

	var cfg config
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return config{}, []error{fmt.Errorf("rc file %s: %w", path, err)}
	}
	return cfg, validateConfig(&cfg, path)
}

// validateConfig drops invalid settings and tasks, reporting each.
func validateConfig(cfg *config, path string) (errs []error) {
	if cfg.PackageManager != "" {
		name, ok := packageManagerName(cfg.PackageManager)
		if !ok {
			errs = append(errs, fmt.Errorf("rc file %s: unknown package_manager %q", path, cfg.PackageManager))
		}
		cfg.PackageManager = name // "" when unknown
	}

	var tasks []configTask
	seen := make(map[string]bool, len(cfg.Tasks))
	for _, t := range cfg.Tasks {
		switch {
		case t.Name == "":
			errs = append(errs, fmt.Errorf("rc file %s: task with an empty name", path))
			continue
		case t.Command == "":
			errs = append(errs, fmt.Errorf("rc file %s: task %q has an empty command", path, t.Name))
			continue
		case seen[t.Name]:
			errs = append(errs, fmt.Errorf("rc file %s: duplicate task %q", path, t.Name))
			continue
		case t.Cwd != "" && !filepath.IsLocal(t.Cwd):
			errs = append(errs, fmt.Errorf("rc file %s: task %q: cwd %q must stay inside the project", path, t.Name, t.Cwd))
			continue
		}
		seen[t.Name] = true
		tasks = append(tasks, t)
	}
	cfg.Tasks = tasks
	return errs
}

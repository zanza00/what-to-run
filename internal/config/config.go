// Package config reads and saves the rc file: optional settings that extend or
// patch what the built-in sources discover.
//
// wtr reads two files, when they are present: the project's
// .what-to-run.toml and the user's $XDG_CONFIG_HOME/what-to-run/config.toml.
// The user file can also declare per-project overrides under
// [projects."<path>"], and the loader matches them against the current
// directory and its subdirectories, from the least specific to the most
// specific. The layers apply from the general to the specific: the global top
// level first, then the project's .what-to-run.toml, then the matching
// [projects."<path>"] blocks of the user file. The theme is a user setting,
// like OpenCode's cli.json, so only the top level of the user file sets it,
// and the loader ignores a project theme with a warning. Without either file,
// the zero config keeps the built-in behaviour.
//
// A problem is a warning, never fatal. The loader skips a file with broken
// TOML syntax, while an unknown key, a wrong field type, or an invalid task or
// project entry drops only that item. Every other valid setting in the file
// still applies.
//
// SaveTheme writes a chosen theme back into the user file, creates the file and
// its directory when they are missing, and preserves the rest of the file.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// Config is the merged content of the rc files. The discovery settings are the
// task domain types, so core.Discover takes them without a conversion.
type Config struct {
	// Settings holds the package manager, the disabled sources and the custom
	// tasks of the rc file.
	core.Settings

	// Theme is the look to start with, one of the names that the picker
	// accepts. It is a user setting, like OpenCode's cli.json, so only the user
	// rc file sets it. An empty value means the default, and --theme overrides
	// it for a single run.
	Theme string

	// Projects maps directory paths (e.g. "~/work/app") to the per-project
	// overrides of the user rc file.
	Projects map[string]core.Settings
}

// DefaultFiles returns the locations of the project rc file and of the user rc
// file for a project directory. The user file is empty when the loader cannot
// resolve the home config directory.
func DefaultFiles(dir string) (project, global string) {
	project = filepath.Join(dir, ".what-to-run.toml")
	if configDir, err := userConfigDir(); err == nil {
		global = filepath.Join(configDir, "what-to-run", "config.toml")
	}
	return project, global
}

// Load reads the rc files and returns the merged settings. The layers apply
// from the general to the specific: the global top-level settings first, then
// the project's .what-to-run.toml, then the matching [projects."<path>"]
// blocks of the global file, ordered from the outermost parent directory to the
// deepest match. Missing files are fine. The function reports a broken key or
// entry as a warning and skips it, while the rest of the file still applies.
//
// themeNames are the theme names that the user rc file accepts. The caller owns
// the registry, and an unknown name is a warning.
func Load(files core.Files, projectFile, globalFile string, themeNames []string) (Config, []error) {
	global, errs := readConfig(files, globalFile, false, themeNames)
	project, projectErrs := readConfig(files, projectFile, true, themeNames)
	errs = append(errs, projectErrs...)

	cfg := Config{
		Settings: core.Settings{
			PackageManager: global.PackageManager,
			Disable:        slices.Clone(global.Disable),
			Tasks:          slices.Clone(global.Tasks),
		},
		Theme: global.Theme,
	}
	cfg = mergeLayer(cfg, project.Settings)

	projectDir := "."
	if projectFile != "" {
		projectDir = filepath.Dir(projectFile)
	}
	for _, ov := range matchingProjects(global.Projects, projectDir) {
		cfg = mergeLayer(cfg, ov)
	}
	return cfg, errs
}

// mergeLayer applies a more specific layer on top of cfg. A non-empty
// package_manager wins, the disable entries add to the list, and the tasks of
// the layer come first so that they shadow the same-named tasks of the lower
// layers.
func mergeLayer(cfg Config, layer core.Settings) Config {
	if layer.PackageManager != "" {
		cfg.PackageManager = layer.PackageManager
	}
	cfg.Disable = append(cfg.Disable, layer.Disable...)
	cfg.Tasks = firstTasks(append(slices.Clone(layer.Tasks), cfg.Tasks...))
	return cfg
}

// firstTasks keeps the first task of each name and drops the later duplicates.
func firstTasks(tasks []core.RCTask) []core.RCTask {
	seen := make(map[string]bool, len(tasks))
	var kept []core.RCTask
	for _, t := range tasks {
		if seen[t.Name] {
			continue
		}
		seen[t.Name] = true
		kept = append(kept, t)
	}
	return kept
}

// readConfig reads one rc file in best-effort mode. An empty path and a
// missing file yield the zero config. A TOML syntax error skips the file with
// a warning. An unknown key, a wrong type and an invalid entry drop only the
// offending item, with one warning each, so the rest of the file still
// applies.
func readConfig(files core.Files, path string, isProject bool, themeNames []string) (Config, []error) {
	if path == "" {
		return Config{}, nil
	}
	data, err := files.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	disp := ShortPath(path)
	if err != nil {
		return Config{}, []error{fmt.Errorf("rc file %s: %w", disp, err)}
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, []error{fmt.Errorf("rc file %s: %w", disp, err)}
	}
	return decodeConfigMap(raw, disp, isProject, themeNames)
}

// decodeConfigMap extracts the valid settings from a parsed top-level TOML
// map. It collects a warning for every unknown key, type mismatch and invalid
// entry.
func decodeConfigMap(raw map[string]any, disp string, isProject bool, themeNames []string) (Config, []error) {
	var cfg Config
	var errs []error

	for _, key := range slices.Sorted(maps.Keys(raw)) {
		val := raw[key]
		switch key {
		case "package_manager":
			pm, ok := decodePackageManager(val, disp, "", &errs)
			if ok {
				cfg.PackageManager = pm
			}

		case "theme":
			if isProject {
				errs = append(errs, fmt.Errorf("rc file %s: theme is a user setting, ignored here", disp))
				continue
			}
			s, ok := val.(string)
			if !ok {
				errs = append(errs, fmt.Errorf("rc file %s: theme must be a string", disp))
				continue
			}
			if s != "" {
				if !slices.Contains(themeNames, s) {
					errs = append(errs, fmt.Errorf("rc file %s: unknown theme %q", disp, s))
					continue
				}
				cfg.Theme = s
			}

		case "disable":
			cfg.Disable = decodeDisable(val, disp, "", &errs)

		case "tasks":
			cfg.Tasks = decodeTasks(val, disp, "", &errs)

		case "projects":
			if isProject {
				errs = append(errs, fmt.Errorf("rc file %s: projects is a user setting, ignored here", disp))
				continue
			}
			cfg.Projects = decodeProjects(val, disp, &errs)

		default:
			errs = append(errs, fmt.Errorf("rc file %s: unknown key %q", disp, key))
		}
	}

	return cfg, errs
}

// decodePackageManager validates a package_manager value. scope prefixes the
// field in the warning messages when the value is inside a
// [projects."<path>"] block.
func decodePackageManager(val any, disp, scope string, errs *[]error) (string, bool) {
	s, ok := val.(string)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: %spackage_manager must be a string", disp, scope))
		return "", false
	}
	if s == "" {
		return "", true
	}
	name, valid := core.PackageManagerName(s)
	if !valid {
		*errs = append(*errs, fmt.Errorf("rc file %s: %sunknown package_manager %q", disp, scope, s))
		return "", false
	}
	return name, true
}

// decodeDisable validates a disable array. It keeps every valid source ID,
// even when another element of the array has the wrong type or names no
// source.
func decodeDisable(val any, disp, scope string, errs *[]error) []string {
	items, ok := val.([]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: %sdisable must be an array of strings", disp, scope))
		return nil
	}
	var out []string
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			*errs = append(*errs, fmt.Errorf("rc file %s: %sdisable entries must be strings", disp, scope))
			continue
		}
		if s == "" {
			continue
		}
		if err := core.CheckSourceID(s); err != nil {
			*errs = append(*errs, fmt.Errorf("rc file %s: %sdisable: %w", disp, scope, err))
			continue
		}
		out = append(out, s)
	}
	return out
}

// decodeTasks validates a [[tasks]] array of tables. It keeps every valid
// task, and it even recovers a task that has an unknown extra key next to
// valid name and command fields.
func decodeTasks(val any, disp, scope string, errs *[]error) []core.RCTask {
	items, ok := val.([]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: %stasks must be an array of tables", disp, scope))
		return nil
	}
	var tasks []core.RCTask
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		table, ok := item.(map[string]any)
		if !ok {
			*errs = append(*errs, fmt.Errorf("rc file %s: %seach task must be a table", disp, scope))
			continue
		}
		t, valid := decodeOneTask(table, disp, scope, errs)
		if !valid {
			continue
		}
		switch {
		case t.Name == "":
			*errs = append(*errs, fmt.Errorf("rc file %s: %stask with an empty name", disp, scope))
			continue
		case t.Command == "":
			*errs = append(*errs, fmt.Errorf("rc file %s: %stask %q has an empty command", disp, scope, t.Name))
			continue
		case seen[t.Name]:
			*errs = append(*errs, fmt.Errorf("rc file %s: %sduplicate task %q", disp, scope, t.Name))
			continue
		case t.Cwd != "" && !filepath.IsLocal(t.Cwd):
			*errs = append(*errs, fmt.Errorf("rc file %s: %stask %q: cwd %q must stay inside the project", disp, scope, t.Name, t.Cwd))
			continue
		}
		seen[t.Name] = true
		tasks = append(tasks, t)
	}
	return tasks
}

// decodeOneTask reads the fields of a single [[tasks]] table. An unknown key
// warns without discarding the task. A wrong type on name, command or cwd drops
// the task, while a wrong description drops only the description.
func decodeOneTask(table map[string]any, disp, scope string, errs *[]error) (core.RCTask, bool) {
	var t core.RCTask
	valid := true
	for _, key := range slices.Sorted(maps.Keys(table)) {
		val := table[key]
		switch key {
		case "name":
			s, ok := val.(string)
			if !ok {
				*errs = append(*errs, fmt.Errorf("rc file %s: %stask field \"name\" must be a string", disp, scope))
				valid = false
				continue
			}
			t.Name = s
		case "command":
			s, ok := val.(string)
			if !ok {
				*errs = append(*errs, fmt.Errorf("rc file %s: %stask field \"command\" must be a string", disp, scope))
				valid = false
				continue
			}
			t.Command = s
		case "description":
			s, ok := val.(string)
			if !ok {
				*errs = append(*errs, fmt.Errorf("rc file %s: %stask field \"description\" must be a string", disp, scope))
				continue
			}
			t.Description = s
		case "cwd":
			s, ok := val.(string)
			if !ok {
				*errs = append(*errs, fmt.Errorf("rc file %s: %stask field \"cwd\" must be a string", disp, scope))
				valid = false
				continue
			}
			t.Cwd = s
		default:
			*errs = append(*errs, fmt.Errorf("rc file %s: %stask has unknown key %q", disp, scope, key))
		}
	}
	return t, valid
}

// decodeProjects validates the [projects."<path>"] table of the user rc file.
func decodeProjects(val any, disp string, errs *[]error) map[string]core.Settings {
	table, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: projects must be a table", disp))
		return nil
	}
	projects := make(map[string]core.Settings, len(table))
	for _, rawPath := range slices.Sorted(maps.Keys(table)) {
		scope := fmt.Sprintf("projects.%q: ", rawPath)
		expanded := expandHome(strings.TrimSpace(rawPath))
		if expanded == "" || !filepath.IsAbs(expanded) {
			*errs = append(*errs, fmt.Errorf("rc file %s: %spath must be absolute or start with ~", disp, scope))
			continue
		}
		entry, ok := table[rawPath].(map[string]any)
		if !ok {
			*errs = append(*errs, fmt.Errorf("rc file %s: %smust be a table", disp, scope))
			continue
		}
		var settings core.Settings
		for _, key := range slices.Sorted(maps.Keys(entry)) {
			v := entry[key]
			switch key {
			case "package_manager":
				if pm, ok := decodePackageManager(v, disp, scope, errs); ok {
					settings.PackageManager = pm
				}
			case "disable":
				settings.Disable = decodeDisable(v, disp, scope, errs)
			case "tasks":
				settings.Tasks = decodeTasks(v, disp, scope, errs)
			case "theme":
				*errs = append(*errs, fmt.Errorf("rc file %s: %stheme is a global setting, ignored here", disp, scope))
			default:
				*errs = append(*errs, fmt.Errorf("rc file %s: %sunknown key %q", disp, scope, key))
			}
		}
		projects[rawPath] = settings
	}
	return projects
}

// matchingProjects returns the [projects."<path>"] blocks whose path is dir
// itself or a parent directory of dir. The list runs from the least specific
// block (the outermost directory) to the most specific one (the deepest
// directory).
func matchingProjects(projects map[string]core.Settings, dir string) []core.Settings {
	if len(projects) == 0 {
		return nil
	}
	target := canonicalPath(dir)
	type match struct {
		rawKey   string
		canon    string
		segments int
		settings core.Settings
	}
	var matches []match
	for rawKey, settings := range projects {
		canon := canonicalPath(strings.TrimSpace(rawKey))
		if segs, ok := pathWithin(canon, target); ok {
			matches = append(matches, match{
				rawKey:   rawKey,
				canon:    canon,
				segments: segs,
				settings: settings,
			})
		}
	}
	slices.SortFunc(matches, func(a, b match) int {
		if c := cmp.Compare(a.segments, b.segments); c != 0 {
			return c
		}
		if c := cmp.Compare(len(a.canon), len(b.canon)); c != 0 {
			return c
		}
		return cmp.Compare(a.rawKey, b.rawKey)
	})
	out := make([]core.Settings, len(matches))
	for i, m := range matches {
		out[i] = m.settings
	}
	return out
}

// pathWithin reports whether target is base itself or a subdirectory inside
// base, on path-segment boundaries. It also returns the segment depth of base.
func pathWithin(base, target string) (int, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return 0, false
	}
	if rel != "." && !filepath.IsLocal(rel) {
		return 0, false
	}
	trimmed := strings.Trim(base, string(filepath.Separator))
	if trimmed == "" {
		return 0, true
	}
	return strings.Count(trimmed, string(filepath.Separator)) + 1, true
}

// canonicalPath expands a leading "~", resolves a relative path against the
// current directory, and resolves the symlinks on the longest prefix of the
// path that exists. In this way a symlinked home (e.g. /home -> /var/home)
// still matches.
func canonicalPath(p string) string {
	p = expandHome(p)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return evalSymlinksPartial(filepath.Clean(p))
}

// evalSymlinksPartial resolves the symlinks on p, or on its longest existing
// ancestor directory when p does not exist on disk yet.
func evalSymlinksPartial(p string) string {
	cur := p
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
}

// expandHome replaces a leading "~" or "~/" with the user's home directory.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// ShortPath replaces the home directory of the user with "~" in messages.
func ShortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	rest, ok := strings.CutPrefix(path, home)
	if !ok || (rest != "" && rest[0] != filepath.Separator) {
		return path
	}
	return "~" + rest
}

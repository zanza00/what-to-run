// The rc file: optional settings that extend or patch what the built-in
// sources discover.
//
// Two files are read, if present: the project's .what-to-run.toml and
// the user's $XDG_CONFIG_HOME/what-to-run/config.toml. The user file
// can also declare per-project overrides under [projects."<path>"], matched
// against the current directory (and its subdirectories, from least to most
// specific). Layers apply from general to specific: global top-level, then the
// project's .what-to-run.toml, then matching [projects."<path>"] blocks
// from the user file. The theme is a user setting, like OpenCode's cli.json:
// only the user file's top level sets it, and a project theme is ignored with
// a warning. Without either file the zero config keeps the built-in behaviour.
//
// Problems are warnings, never fatal: a file with broken TOML syntax is
// skipped, while an unknown key, a wrong field type, or an invalid task or
// project entry drops only that item so every other valid setting in the file
// still applies.
//
// saveTheme writes a chosen theme back into the user file (creating it, with
// its directory), preserving the rest of the file.

package tui

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// config is the merged content of the rc files.
type config struct {
	// PackageManager forces the runner used for package.json scripts:
	// "npm", "pnpm", "yarn" or "bun". Empty means detection as usual.
	PackageManager string `toml:"package_manager"`

	// Theme is the look to start with, one of the names of ListThemes. It is
	// a user setting, like OpenCode's cli.json: only the user rc file sets
	// it. Empty means the default; --theme overrides it for a single run.
	Theme string `toml:"theme"`

	// Disable names the sources to skip, by the file names they look for;
	// see core.Sources for the built-in registry.
	Disable []string `toml:"disable"`

	// Tasks are custom tasks, shown above the discovered ones.
	Tasks []configTask `toml:"tasks"`

	// Projects maps directory paths (e.g. "~/work/app") to per-project
	// overrides in the user rc file.
	Projects map[string]projectConfig `toml:"projects"`
}

// projectConfig is one [projects."<path>"] override block in the user rc file.
type projectConfig struct {
	PackageManager string       `toml:"package_manager"`
	Disable        []string     `toml:"disable"`
	Tasks          []configTask `toml:"tasks"`
}

// configTask is one [[tasks]] entry of an rc file.
type configTask struct {
	Name        string `toml:"name"`        // required
	Command     string `toml:"command"`     // required; run with the platform shell (sh -c, cmd /c on Windows)
	Description string `toml:"description"` // defaults to Command
	Cwd         string `toml:"cwd"`         // optional, relative to the project
}

// defaultConfigFiles returns the project and user rc file locations for a
// project directory. The user file is empty when the home config directory
// cannot be resolved.
func defaultConfigFiles(dir string) (project, global string) {
	project = filepath.Join(dir, ".what-to-run.toml")
	if configDir, err := userConfigDir(); err == nil {
		global = filepath.Join(configDir, "what-to-run", "config.toml")
	}
	return project, global
}

// loadConfig reads the rc files and returns the merged settings. Layers are
// applied from general to specific: global top-level settings first, then the
// project's .what-to-run.toml, then any matching [projects."<path>"]
// blocks from the global file ordered from outermost parent directory to the
// deepest match. Missing files are fine; broken keys or entries are reported
// as warnings and skipped while the rest of the file still applies.
func loadConfig(files core.Files, projectFile, globalFile string) (config, []error) {
	global, errs := readConfig(files, globalFile, false)
	project, projectErrs := readConfig(files, projectFile, true)
	errs = append(errs, projectErrs...)

	cfg := config{
		PackageManager: global.PackageManager,
		Theme:          global.Theme,
		Disable:        slices.Clone(global.Disable),
		Tasks:          slices.Clone(global.Tasks),
	}
	cfg = mergeLayer(cfg, projectConfig{
		PackageManager: project.PackageManager,
		Disable:        project.Disable,
		Tasks:          project.Tasks,
	})

	projectDir := "."
	if projectFile != "" {
		projectDir = filepath.Dir(projectFile)
	}
	for _, ov := range matchingProjects(global.Projects, projectDir) {
		cfg = mergeLayer(cfg, ov)
	}
	return cfg, errs
}

// mergeLayer applies a more specific layer on top of cfg: a non-empty
// package_manager wins, disable entries add to the list, and tasks from the
// layer come first so they shadow same-named ones from lower layers.
func mergeLayer(cfg config, layer projectConfig) config {
	if layer.PackageManager != "" {
		cfg.PackageManager = layer.PackageManager
	}
	cfg.Disable = append(cfg.Disable, layer.Disable...)
	cfg.Tasks = firstTasks(append(slices.Clone(layer.Tasks), cfg.Tasks...))
	return cfg
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

// readConfig reads one rc file in best-effort mode. An empty path and a
// missing file yield the zero config. A TOML syntax error skips the file with
// a warning; unknown keys, wrong types, and invalid entries drop only the
// offending item with a warning each so the rest of the file still applies.
func readConfig(files core.Files, path string, isProject bool) (config, []error) {
	if path == "" {
		return config{}, nil
	}
	data, err := files.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return config{}, nil
	}
	disp := shortPath(path)
	if err != nil {
		return config{}, []error{fmt.Errorf("rc file %s: %w", disp, err)}
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return config{}, []error{fmt.Errorf("rc file %s: %w", disp, err)}
	}
	return decodeConfigMap(raw, disp, isProject)
}

// decodeConfigMap extracts valid settings from a parsed top-level TOML map,
// collecting a warning for every unknown key, type mismatch, or invalid entry.
func decodeConfigMap(raw map[string]any, disp string, isProject bool) (config, []error) {
	var cfg config
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
				if _, valid := themeIndex(s); !valid {
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
// field in warning messages when inside a [projects."<path>"] block.
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

// decodeDisable validates a disable array, keeping every valid string entry
// even if another element in the array has the wrong type.
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
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// decodeTasks validates a [[tasks]] array of tables, keeping every valid task
// and even recovering a task that has an unknown extra key alongside valid
// name and command fields.
func decodeTasks(val any, disp, scope string, errs *[]error) []configTask {
	items, ok := val.([]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: %stasks must be an array of tables", disp, scope))
		return nil
	}
	var tasks []configTask
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

// decodeOneTask reads the fields of a single [[tasks]] table. Unknown keys
// warn without discarding the task; a wrong type on a known field drops the
// task.
func decodeOneTask(table map[string]any, disp, scope string, errs *[]error) (configTask, bool) {
	var t configTask
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

// decodeProjects validates the [projects."<path>"] table in the user rc file.
func decodeProjects(val any, disp string, errs *[]error) map[string]projectConfig {
	table, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("rc file %s: projects must be a table", disp))
		return nil
	}
	projects := make(map[string]projectConfig, len(table))
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
		var pc projectConfig
		for _, key := range slices.Sorted(maps.Keys(entry)) {
			v := entry[key]
			switch key {
			case "package_manager":
				if pm, ok := decodePackageManager(v, disp, scope, errs); ok {
					pc.PackageManager = pm
				}
			case "disable":
				pc.Disable = decodeDisable(v, disp, scope, errs)
			case "tasks":
				pc.Tasks = decodeTasks(v, disp, scope, errs)
			case "theme":
				*errs = append(*errs, fmt.Errorf("rc file %s: %stheme is a global setting, ignored here", disp, scope))
			default:
				*errs = append(*errs, fmt.Errorf("rc file %s: %sunknown key %q", disp, scope, key))
			}
		}
		projects[rawPath] = pc
	}
	return projects
}

// matchingProjects returns the [projects."<path>"] blocks whose path equals or
// contains dir, ordered from least specific (outermost directory) to most
// specific (deepest directory).
func matchingProjects(projects map[string]projectConfig, dir string) []projectConfig {
	if len(projects) == 0 {
		return nil
	}
	target := canonicalPath(dir)
	type match struct {
		rawKey   string
		canon    string
		segments int
		cfg      projectConfig
	}
	var matches []match
	for rawKey, pc := range projects {
		canon := canonicalPath(strings.TrimSpace(rawKey))
		if segs, ok := pathWithin(canon, target); ok {
			matches = append(matches, match{
				rawKey:   rawKey,
				canon:    canon,
				segments: segs,
				cfg:      pc,
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
	out := make([]projectConfig, len(matches))
	for i, m := range matches {
		out[i] = m.cfg
	}
	return out
}

// pathWithin reports whether target is base itself or a subdirectory inside
// base on path-segment boundaries, and returns the segment depth of base.
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

// canonicalPath expands a leading "~", resolves relative paths against the
// current directory, and resolves symlinks on the longest existing prefix of
// the path so symlinked homes (e.g. /home -> /var/home) match reliably.
func canonicalPath(p string) string {
	p = expandHome(p)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return evalSymlinksPartial(filepath.Clean(p))
}

// evalSymlinksPartial resolves symlinks on p, or on its longest existing
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

// saveTheme writes a theme into the user rc file, the only place that holds
// it (like OpenCode's cli.json): the file is created with its directory when
// missing, and the rest of its content is preserved. It returns the path
// written.
func saveTheme(name string) (string, error) {
	_, userFile := defaultConfigFiles(".")
	if userFile == "" {
		return "", errors.New("cannot resolve the user config directory")
	}

	data, err := os.ReadFile(userFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(userFile, setThemeKey(data, name), 0o600); err != nil {
		return "", err
	}
	return userFile, nil
}

// setThemeKey returns the rc file content with the top-level theme key set to
// name: an existing assignment is replaced, a missing one is added at the top
// of the file (after a table header it would belong to that table).
func setThemeKey(src []byte, name string) []byte {
	assignment := "theme = " + strconv.Quote(name)
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break // tables start here: the key must go above them
		}
		if key, _, ok := strings.Cut(trimmed, "="); ok && strings.TrimSpace(key) == "theme" {
			lines[i] = assignment
			return []byte(strings.Join(lines, "\n"))
		}
	}
	if len(bytes.TrimSpace(src)) == 0 {
		return []byte(assignment + "\n")
	}
	return []byte(assignment + "\n\n" + string(src))
}

// shortPath replaces the user's home directory with "~", for messages.
func shortPath(path string) string {
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

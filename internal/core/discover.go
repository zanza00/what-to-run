// Task discovery: the runnable scripts a project declares.
//
// A source is one family of project files the picker understands. Each
// source owns the file names it looks for and a parser that turns one
// file's bytes into tasks; the framework below finds the first existing
// candidate file, stamps the tasks with its path relative to the project
// directory as their badge, and turns a failed parse into a warning instead
// of losing the tasks of the other sources.
//
// Most sources look only in the project directory. The justfile and Taskfile
// sources also search its ancestors, like the just and task tools do: the
// nearest directory holding a candidate wins, and a file found above the
// project directory is badged with the relative path, such as ../justfile.
//
// The built-in sources and their order are the defaultSources registry;
// Sources exposes them and docs/task-sources.md documents them.
//
// The rc file (decoded by the caller) sits on top: its tasks are shown
// first and shadow same-named tasks of the sources, and a source can be
// disabled by its ID ("make", "cargo-make"). A source that is missing
// contributes nothing; a source that exists but fails to parse is reported in
// errs and skipped, so one broken file does not hide the tasks of the others.

package core

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// SourceInfo describes a built-in task source: the read-only view of the
// registry that the TUI and the docs test use.
type SourceInfo struct {
	// ID is the stable name of the source in the rc file, named after the
	// tool it reads rather than after one of its files ("npm", "make",
	// "cargo-make"). It is lowercase kebab-case, unique, and never one of
	// the file names of any source, so it keys the disable list and, later,
	// the [sources.<id>] tables.
	ID string

	// Label is the human name shown in the README and the docs ("Cargo
	// aliases", "Makefile"). It is unique and need not be a file name.
	Label string

	// DisplayFileOverride overrides the file shown in the empty screen;
	// empty means Files[0]. Make sets it to Makefile because it prefers
	// GNUmakefile but displays Makefile; cargo-alias sets it to
	// .cargo/config.toml because Cargo recommends that name while the
	// parser prefers .cargo/config.
	DisplayFileOverride string

	// Files are candidate paths in preference order, relative to the
	// directory being searched: the first that exists is parsed. Its path
	// relative to the project directory becomes the badge of the tasks the
	// source returns.
	Files []string

	// SearchParents makes the source look in the project directory's
	// ancestors after the directory itself, as just and task do. It is
	// false for every other source: their files are directory-local.
	SearchParents bool
}

// DisplayFile is the file the empty screen shows: DisplayFileOverride when
// set, otherwise the first candidate.
func (s SourceInfo) DisplayFile() string {
	if s.DisplayFileOverride != "" {
		return s.DisplayFileOverride
	}
	if len(s.Files) == 0 {
		return ""
	}
	return s.Files[0]
}

// source is one family of project files. Adding a source is a SourceInfo
// literal in defaultSources plus a parser function; the framework does the
// rest. Sources are discovered in registry order, and each parser decides the
// order of the tasks it returns.
type source struct {
	SourceInfo

	// Parse turns the winning file's content into tasks: it receives the
	// directory containing the file for sibling lookups (lock files,
	// referenced files) and the file's bytes. Task.Source is stamped by the
	// framework, not by the parser.
	Parse func(dir string, data []byte) ([]Task, error)
}

// Settings is the part of the rc file that reaches task discovery.
type Settings struct {
	// PackageManager overrides the runner used for package.json scripts
	// (npm, pnpm, yarn, bun); empty means autodetect.
	PackageManager string

	// Disable names the sources to skip, by ID ("make", "cargo-make").
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
		taskfileSource(files),
		justfileSource(files),
		cargoMakeSource(),
		cargoAliasSource(),
		devboxSource(),
		composerSource(),
		denoSource(),
		miseSource(),
	}
}

// Sources returns the built-in sources in display order. The result is a
// copy: changing it does not affect discovery.
func Sources() []SourceInfo {
	srcs := defaultSources(Settings{}, missingFiles{})
	infos := make([]SourceInfo, len(srcs))
	for i, src := range srcs {
		infos[i] = src.SourceInfo
		infos[i].Files = slices.Clone(src.Files)
	}
	return infos
}

// missingFiles is the Files used to build metadata: every path is missing.
type missingFiles struct{}

var _ Files = missingFiles{}

func (missingFiles) ReadFile(string) ([]byte, error)    { return nil, fs.ErrNotExist }
func (missingFiles) Exists(string) bool                 { return false }
func (missingFiles) ReadDir(string) ([]DirEntry, error) { return nil, fs.ErrNotExist }

// CheckSourceID returns nil when name is the ID of a built-in source. The
// error explains why it is not one: a file name a source reads (with the ID
// to use instead), a package manager name (to disable package.json scripts
// use "npm"), a near miss in case (with the ID), or an unknown name (with the
// list of valid IDs).
func CheckSourceID(name string) error {
	return checkSourceID(defaultSources(Settings{}, missingFiles{}), name)
}

// checkSourceID validates name against the given registry. It takes the
// sources as a parameter so tests can check a shared file name with synthetic
// sources.
func checkSourceID(sources []source, name string) error {
	var byFile []string
	for _, src := range sources {
		if src.ID == name {
			return nil
		}
		if slices.Contains(src.Files, name) {
			byFile = append(byFile, strconv.Quote(src.ID))
		}
	}
	switch {
	case len(byFile) == 1:
		return fmt.Errorf("%q is a file name; use %s", name, byFile[0])
	case len(byFile) > 1:
		return fmt.Errorf("%q is a file name; use one of %s", name, strings.Join(byFile, ", "))
	}
	if _, ok := PackageManagerName(name); ok {
		return fmt.Errorf(
			"%q is a package manager, not a source ID; to disable package.json scripts use \"npm\"",
			name,
		)
	}
	// IDs are unique and lowercase, so an EqualFold match is unique.
	for _, src := range sources {
		if strings.EqualFold(src.ID, name) {
			return fmt.Errorf("unknown source %q; did you mean %q?", name, src.ID)
		}
	}
	ids := make([]string, len(sources))
	for i, src := range sources {
		ids[i] = src.ID
	}
	return fmt.Errorf("unknown source %q; known sources: %s", name, strings.Join(ids, ", "))
}

// discoverSource runs one source over dir, and over its ancestors when the
// source searches parents: in each directory the first candidate file that
// exists is parsed, the nearest directory wins, and its path relative to dir
// becomes the badge (a bare file name in dir itself, ../justfile above it).
func discoverSource(files Files, src source, dir string) ([]Task, error) {
	// Read the project directory through the caller's path, so an in-memory
	// Files keyed by relative names still matches. Parent search needs
	// absolute paths: filepath.Dir(".") is ".".
	tasks, found, err := readSourceFile(files, src, dir, "")
	if found || err != nil || !src.SearchParents {
		return tasks, err
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	for current := filepath.Dir(base); ; current = filepath.Dir(current) {
		prefix, relErr := filepath.Rel(base, current)
		if relErr != nil {
			prefix = current
		}
		tasks, found, err := readSourceFile(files, src, current, filepath.ToSlash(prefix))
		if found || err != nil {
			return tasks, err
		}
		if filepath.Dir(current) == current {
			return nil, nil
		}
	}
}

// readSourceFile reads the first candidate file of src that exists in dir and
// runs the parser on it. prefix is the directory's path relative to the project
// directory, empty for the project directory itself: the badge is the file name
// there and prefix/name above it. found tells whether a file was read, whether
// it parsed or not.
func readSourceFile(files Files, src source, dir, prefix string) (tasks []Task, found bool, err error) {
	for _, name := range src.Files {
		path := filepath.Join(dir, name)
		data, err := files.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		badge := name
		if prefix != "" {
			badge = filepath.ToSlash(filepath.Join(prefix, name))
		}
		if err != nil {
			return nil, true, fmt.Errorf("%s: %w", badge, err)
		}
		tasks, err := src.Parse(filepath.Dir(path), data)
		if err != nil {
			return nil, true, fmt.Errorf("%s: %w", badge, err)
		}
		for i := range tasks {
			tasks[i].Source = badge
		}
		return tasks, true, nil
	}
	return nil, false, nil
}

// enabledSources drops the sources named in disable, by ID, and reports the
// entries that are not IDs (a typo in the rc file). The rc decoder validates
// entries before they get here, so this warning is a safety net for callers
// that build Settings by hand; sources must be the full registry, or a valid
// ID missing from it warns as unknown.
func enabledSources(sources []source, disable []string) ([]source, []error) {
	var errs []error
	for _, name := range disable {
		if err := checkSourceID(sources, name); err != nil {
			errs = append(errs, fmt.Errorf("disable: %w", err))
		}
	}

	kept := make([]source, 0, len(sources))
	for _, src := range sources {
		if !slices.Contains(disable, src.ID) {
			kept = append(kept, src)
		}
	}
	return kept, errs
}

// rcTasks converts the rc file's custom tasks into picker tasks: they show
// the "rc" badge, run through the platform's shell (sh on Unix, cmd on
// Windows), and a relative cwd is resolved against the project directory.
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
			Args:   shellCommand(t.Command),
		}
		if t.Cwd != "" {
			task.Dir = filepath.Join(dir, t.Cwd)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// Discover scans dir for the tasks of the rc file and the built-in sources,
// in display order: rc tasks first, then each enabled source in registry
// order; each parser decides the order of its own tasks. A source that fails
// to parse is reported in errs and skipped (a disabled source is not read at
// all), and rc tasks shadow same-named tasks of the built-in sources.
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

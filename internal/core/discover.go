// Task discovery: the runnable scripts a project declares.
//
// A source is one family of project files that the picker understands. Each
// source owns the file names it looks for and a parser that turns the bytes of
// one file into tasks. The framework below finds the first candidate file that
// exists, stamps its path relative to the project directory on the tasks as
// their badge, and turns a failed parse into a warning. In this way one broken
// file does not hide the tasks of the other sources.
//
// Most sources look only in the project directory. The justfile and Taskfile
// sources also search its ancestors, as the just and task tools do. The nearest
// directory that holds a candidate wins, and a file found above the project
// directory gets the relative path as its badge, such as ../justfile.
//
// The defaultSources registry holds the built-in sources and their order.
// Sources exposes them, and docs/task-sources.md documents them.
//
// The rc file (decoded by the caller) sits on top. Its tasks come first and
// shadow same-named tasks of the sources, and the disable list removes a source
// by its ID ("make", "cargo-make"). A source that is missing contributes
// nothing. A source that exists but fails to parse is reported in errs and
// skipped.

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

// SourceInfo describes a built-in task source. It is the read-only view of the
// registry that the TUI and the docs test use.
type SourceInfo struct {
	// ID is the stable name of the source in the rc file. It is named after
	// the tool that the source reads, not after one of its files ("npm",
	// "make", "cargo-make"). It is lowercase kebab-case and unique, and it is
	// never one of the file names of any source. It keys the disable list and,
	// later, the [sources.<id>] tables.
	ID string

	// Label is the human name that the README and the docs show ("Cargo
	// aliases", "Makefile"). It is unique and need not be a file name.
	Label string

	// DisplayFileOverride overrides the file that the empty screen shows. An
	// empty value means Files[0]. Make sets it to Makefile because it prefers
	// GNUmakefile but displays Makefile. cargo-alias sets it to
	// .cargo/config.toml because Cargo recommends that name, while the parser
	// prefers .cargo/config.
	DisplayFileOverride string

	// Files are candidate paths in preference order, relative to the directory
	// being searched. The first one that exists is parsed, and its path
	// relative to the project directory becomes the badge of the tasks that
	// the source returns.
	Files []string

	// SearchParents makes the source look in the ancestors of the project
	// directory after the directory itself, as just and task do. It is false
	// for every other source, because their files are directory-local.
	SearchParents bool
}

// DisplayFile returns the file that the empty screen shows: DisplayFileOverride
// when it is set, otherwise the first candidate.
func (s SourceInfo) DisplayFile() string {
	if s.DisplayFileOverride != "" {
		return s.DisplayFileOverride
	}
	if len(s.Files) == 0 {
		return ""
	}
	return s.Files[0]
}

// source is one family of project files. To add a source, write a SourceInfo
// literal in defaultSources and a parser function. The framework does the rest.
// The registry order is the discovery order, and each parser decides the order
// of the tasks it returns.
type source struct {
	SourceInfo

	// Parse turns the content of the winning file into tasks. It receives the
	// directory that contains the file, for sibling lookups (lock files,
	// referenced files), and the bytes of the file. The framework stamps
	// Task.Source, not the parser.
	Parse func(dir string, data []byte) ([]Task, error)
}

// Settings is the part of the rc file that reaches task discovery.
type Settings struct {
	// PackageManager overrides the runner that runs package.json scripts (npm,
	// pnpm, yarn, bun). An empty value means autodetect.
	PackageManager string

	// Disable names the sources to skip, by ID ("make", "cargo-make").
	Disable []string

	// Tasks are the custom tasks of the rc file. The picker shows them above
	// the discovered ones.
	Tasks []RCTask
}

// RCTask is one custom task that the rc file declares.
type RCTask struct {
	Name        string
	Command     string
	Description string
	Cwd         string
}

// defaultSources returns the built-in sources in display order. It wires in the
// rc file settings that reach the sources.
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

// Sources returns the built-in sources in display order. The result is a copy,
// so a change to it does not affect discovery.
func Sources() []SourceInfo {
	srcs := defaultSources(Settings{}, missingFiles{})
	infos := make([]SourceInfo, len(srcs))
	for i, src := range srcs {
		infos[i] = src.SourceInfo
		infos[i].Files = slices.Clone(src.Files)
	}
	return infos
}

// missingFiles is the Files implementation used to build metadata. Every path
// is missing.
type missingFiles struct{}

var _ Files = missingFiles{}

func (missingFiles) ReadFile(string) ([]byte, error)    { return nil, fs.ErrNotExist }
func (missingFiles) Exists(string) bool                 { return false }
func (missingFiles) ReadDir(string) ([]DirEntry, error) { return nil, fs.ErrNotExist }

// CheckSourceID returns nil when name is the ID of a built-in source. The error
// explains why it is not one. It names a file that a source reads (with the ID
// to use instead), a package manager (to disable package.json scripts, use
// "npm"), a near miss in case (with the ID), or an unknown name (with the list
// of valid IDs).
func CheckSourceID(name string) error {
	return checkSourceID(defaultSources(Settings{}, missingFiles{}), name)
}

// checkSourceID validates name against the given registry. It takes the sources
// as a parameter so that tests can check a shared file name with synthetic
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
	// IDs are unique and lowercase, so an EqualFold match finds only one.
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

// discoverSource runs one source over dir and, when the source searches
// parents, over its ancestors. In each directory it parses the first candidate
// file that exists, and the nearest directory wins. The path of that file
// relative to dir becomes the badge: a bare file name in dir itself, or
// ../justfile above it.
func discoverSource(files Files, src source, dir string) ([]Task, error) {
	// Read the project directory through the path of the caller, so an
	// in-memory Files keyed by relative names still matches. The parent search
	// needs absolute paths, because filepath.Dir(".") is ".".
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
// runs the parser on it. prefix is the path of the directory relative to the
// project directory, and it is empty for the project directory itself: the
// badge is the file name there and prefix/name above it. found reports whether
// the function read a file, whether it parsed or not.
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

// enabledSources drops the sources that disable names, by ID, and reports the
// entries that are not IDs (a typo in the rc file). The rc decoder validates
// entries before they get here, so this warning is a safety net for callers
// that build Settings by hand. sources must be the full registry. Otherwise a
// valid ID that is missing from it warns as unknown.
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

// rcTasks converts the custom tasks of the rc file into picker tasks. They show
// the "rc" badge, run through the shell of the platform (sh on Unix, cmd on
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

// Discover scans dir for the tasks of the rc file and of the built-in sources,
// in display order: rc tasks first, then each enabled source in registry order.
// Each parser decides the order of its own tasks. A source that fails to parse
// is reported in errs and skipped, and the function does not read a disabled
// source at all. rc tasks shadow same-named tasks of the built-in sources.
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

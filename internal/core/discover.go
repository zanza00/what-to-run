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
// nothing. A source that exists but fails to parse is reported as a warning and
// skipped.

package core

import (
	"bytes"
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
// The registry order is the discovery order, and sortSourceTasks at the seam
// owns the order of the tasks of a source.
type source struct {
	SourceInfo

	// Parse turns the content of the winning file into tasks. It receives the
	// directory that contains the file, for sibling lookups (lock files,
	// referenced files), the base name of the winning file, and the bytes of
	// the file. The framework stamps Task.Source, not the parser.
	Parse func(dir, name string, data []byte) ([]Task, error)
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
		poeSource(),
		pdmSource(),
		taskipySource(),
		hatchSource(),
		toxSource(),
		pipenvSource(),
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

// readResult is one remembered result of a read through the Files port. A
// miss is a result like any other: the parent search asks the port for the
// same missing candidate at every level for every source, so the remembered
// misses save most of the calls.
type readResult struct {
	data []byte
	err  error
}

// readCache wraps the Files port of one Discover and owns the read law: one
// Discover reads one file at most once. Every result is remembered under the
// exact path string that the port received. Several sources can list the
// same file, such as the pyproject.toml of the five Python sources. The
// parsers keep their sibling and include reads on the wrapped port too, so
// the read law covers them without a carve-out. No parser writes into the
// bytes it receives, so the readers of one file can share the slice of its
// first read. Exists and ReadDir pass through untouched. The memo lives
// inside one Discover call and dies with it: nothing is cached across runs.
type readCache struct {
	files Files
	memo  map[string]readResult
}

var _ Files = (*readCache)(nil)

// newReadCache wraps files for one Discover.
func newReadCache(files Files) *readCache {
	return &readCache{files: files, memo: make(map[string]readResult)}
}

// ReadFile remembers the result of the first read of name and returns it for
// every later read of the same path string.
func (c *readCache) ReadFile(name string) ([]byte, error) {
	if result, ok := c.memo[name]; ok {
		return result.data, result.err
	}
	data, err := c.files.ReadFile(name)
	c.memo[name] = readResult{data: data, err: err}
	return data, err
}

func (c *readCache) Exists(name string) bool                 { return c.files.Exists(name) }
func (c *readCache) ReadDir(name string) ([]DirEntry, error) { return c.files.ReadDir(name) }

// parentLevel is one ancestor directory of the project directory. The parent
// search reads the directory through dir, and prefix holds the path of that
// directory relative to the project directory, with a slash separator. A
// file found in the level gets prefix plus its name as the badge, such as
// ../justfile.
type parentLevel struct {
	dir    string
	prefix string
}

// parentLevels enumerates the ancestors of the project directory once per
// Discover. The first parent-searching source that misses in the project
// directory triggers the enumeration, and every later source consumes the
// same list and stops at its own find, so the nearest directory still wins.
// The enumeration fails when the project directory has no absolute path.
// The failure is remembered like the list. Every source that needs the
// parent search then replays the failure as its own warning, keyed by its
// ID. One broken directory therefore still reports one warning per source,
// exactly as the per-source walk did.
type parentLevels struct {
	dir    string
	built  bool
	levels []parentLevel
	err    error
}

// newParentLevels prepares the enumeration of the ancestors of dir. Nothing
// is enumerated until a source needs the parent search: a source that finds
// its file in the project directory never touches the ancestors.
func newParentLevels(dir string) *parentLevels {
	return &parentLevels{dir: dir}
}

// build returns the ancestor levels, and enumerates them on the first call.
// The levels must be absolute, because filepath.Dir(".") is ".": a relative
// project directory would never ascend. The list starts at the parent of the
// project directory and ends at the root.
func (p *parentLevels) build() ([]parentLevel, error) {
	if p.built {
		return p.levels, p.err
	}
	p.built = true
	base, err := filepath.Abs(p.dir)
	if err != nil {
		p.err = err
		return nil, err
	}
	for current := filepath.Dir(base); ; current = filepath.Dir(current) {
		prefix := current
		if rel, relErr := filepath.Rel(base, current); relErr == nil {
			prefix = rel
		}
		p.levels = append(p.levels, parentLevel{dir: current, prefix: filepath.ToSlash(prefix)})
		if filepath.Dir(current) == current {
			break
		}
	}
	return p.levels, nil
}

// discovery holds the state that one Discover shares across its sources: the
// read cache over the Files port and the parent levels of the project
// directory. The registry loop is the outermost loop, and every enabled
// source runs through the same state.
type discovery struct {
	cache   *readCache
	parents *parentLevels
	dir     string
}

// newDiscovery builds the state that one Discover shares across its sources.
func newDiscovery(files Files, dir string) *discovery {
	return &discovery{
		cache:   newReadCache(files),
		parents: newParentLevels(dir),
		dir:     dir,
	}
}

// discoverSource runs one source over the project directory and, when the
// source searches parents, over the parent levels, and returns a warning for
// the first candidate file that fails it. In each directory it parses the
// first candidate file that exists, and the nearest directory wins. The path
// of that file relative to the project directory becomes the badge: a bare
// file name in the directory itself, or ../justfile above it. The warning
// keys on the leaf name alone, so the same file keeps one identity from any
// subdirectory of the project.
func (d *discovery) discoverSource(src source) ([]Task, *Warning) {
	// Read the project directory through the path of the caller, so an
	// in-memory Files keyed by relative names still matches.
	tasks, found, warn := readSourceFile(d.cache, src, d.dir, "")
	if found || warn != nil || !src.SearchParents {
		return tasks, warn
	}
	levels, err := d.parents.build()
	if err != nil {
		// The parent search cannot even resolve the project directory. The
		// warning names that directory, because no candidate file was
		// reached yet.
		w := NewSourceWarning(src.ID, d.dir, fmt.Sprintf("%s: %v", d.dir, err))
		return nil, &w
	}
	for _, level := range levels {
		tasks, found, warn := readSourceFile(d.cache, src, level.dir, level.prefix)
		if found || warn != nil {
			return tasks, warn
		}
	}
	return nil, nil
}

// discoverSource runs one source over dir and its ancestors. It builds the
// state of one Discover for this source alone, so the read law and the
// parent levels hold for the single source as they hold for the sources of
// the registry. The package tests run one source at a time through this
// seam.
func discoverSource(files Files, src source, dir string) ([]Task, *Warning) {
	return newDiscovery(files, dir).discoverSource(src)
}

// readSourceFile reads the first candidate file of src that exists in dir and
// runs the parser on it. This function owns the law of the empty file: a file
// that holds only whitespace reports like a missing one, with no parser call
// and no warning. The package.json source is the exception, because JSON
// parses no empty document, so its empty file keeps its parse failure as a
// warning. sortSourceTasks owns the order of the tasks that come back.
// prefix is the path of the directory relative to the project directory, and
// it is empty for the project directory itself: the badge is the file name
// there and prefix/name above it. found reports whether the function read a
// file, whether it parsed or not. The warning keys on the leaf name, never on
// the badge, and a read failure and a parse failure of one file produce the
// same warning.
func readSourceFile(files Files, src source, dir, prefix string) (tasks []Task, found bool, warn *Warning) {
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
			w := NewSourceWarning(src.ID, name, fmt.Sprintf("%s: %v", badge, err))
			return nil, true, &w
		}
		if emptyFile(data) && src.ID != "npm" {
			// JSON parses no empty document, so the npm source still reads
			// its empty file and keeps its warning. For every other source
			// the file was read, and the parent search stops here too.
			return nil, true, nil
		}
		tasks, err := src.Parse(filepath.Dir(path), name, data)
		if err != nil {
			w := NewSourceWarning(src.ID, name, fmt.Sprintf("%s: %v", badge, err))
			return nil, true, &w
		}
		tasks = sortSourceTasks(src, tasks)
		for i := range tasks {
			tasks[i].Source = badge
		}
		return tasks, true, nil
	}
	return nil, false, nil
}

// emptyFile reports whether data holds nothing but whitespace. Such a file
// declares no tasks, so the seam reports it like a missing one, as the law of
// readSourceFile states.
func emptyFile(data []byte) bool {
	return len(bytes.TrimSpace(data)) == 0
}

// sortSourceTasks returns the tasks of src in the order that the picker shows
// them: sorted by name. The make source keeps the order of its file, because
// a Makefile lists its targets in that order. The hatch source also keeps
// the order of its parser: its sorted loops list the environments and then
// the scripts of each one.
func sortSourceTasks(src source, tasks []Task) []Task {
	if src.ID == "make" || src.ID == "hatch" {
		return tasks
	}
	slices.SortStableFunc(tasks, func(a, b Task) int {
		return strings.Compare(a.Name, b.Name)
	})
	return tasks
}

// enabledSources drops the sources that disable names, by ID, and reports the
// entries that are not IDs (a typo in the rc file). The rc decoder validates
// entries before they get here, so this warning is a safety net for callers
// that build Settings by hand. sources must be the full registry. Otherwise a
// valid ID that is missing from it warns as unknown. The warning keys on the
// raw stored entry, so it holds the same identity as the rc loader's warning
// for the same entry, and the picker shows one warning instead of two.
func enabledSources(sources []source, disable []string) ([]source, []Warning) {
	var warnings []Warning
	for _, name := range disable {
		if err := checkSourceID(sources, name); err != nil {
			warnings = append(warnings,
				NewRegistryWarning(name, fmt.Sprintf("disable: %v", err)))
		}
	}

	kept := make([]source, 0, len(sources))
	for _, src := range sources {
		if !slices.Contains(disable, src.ID) {
			kept = append(kept, src)
		}
	}
	return kept, warnings
}

// rcTasks converts the custom tasks of the rc file into picker tasks. They show
// the "rc" badge, run through the shell of the platform (sh on Unix, cmd on
// Windows), and a relative cwd is resolved against the project directory.
func rcTasks(settings Settings, dir string) []Task {
	var tasks []Task
	for _, t := range settings.Tasks {
		task := Task{
			Source: "rc",
			Name:   t.Name,
			Desc:   helpOrBody(t.Description, t.Command),
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
// Each source's tasks arrive in the order that sortSourceTasks gives them. A
// source that fails to parse is reported in warnings and skipped, and the
// function does not read a disabled source at all. rc tasks shadow
// same-named tasks of the built-in sources. The sources of one scan share
// the read cache and the parent levels, so the work does not multiply with
// the number of sources.
func Discover(files Files, dir string, settings Settings) ([]Task, []Warning) {
	d := newDiscovery(files, dir)
	sources, warns := enabledSources(defaultSources(settings, d.cache), settings.Disable)

	rc := rcTasks(settings, dir)
	shadowed := make(map[string]bool, len(rc))
	for _, t := range rc {
		shadowed[t.Name] = true
	}

	var tasks []Task
	tasks = append(tasks, rc...)
	for _, src := range sources {
		ts, warn := d.discoverSource(src)
		if warn != nil {
			warns = append(warns, *warn)
			continue
		}
		for _, t := range ts {
			if !shadowed[t.Name] {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks, warns
}

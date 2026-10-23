package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// osFiles is the test-only OS adapter for the Files port: it reads the real
// filesystem, which the fixture-based tests in this package need.
type osFiles struct{}

var _ Files = osFiles{}

func (osFiles) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFiles) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
func (osFiles) ReadDir(name string) ([]DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	directories := make([]DirEntry, len(entries))
	for i, entry := range entries {
		directories[i] = entry
	}
	return directories, nil
}

// mapFiles is an in-memory Files implementation: a path is present when it is
// a key of the map, empty content included.
type mapFiles map[string]string

var _ Files = mapFiles{}

func (m mapFiles) ReadFile(name string) ([]byte, error) {
	content, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(content), nil
}

func (m mapFiles) Exists(name string) bool {
	_, ok := m[name]
	return ok
}

func (m mapFiles) ReadDir(name string) ([]DirEntry, error) {
	prefix := strings.TrimSuffix(name, string(filepath.Separator)) + string(filepath.Separator)
	children := make(map[string]bool)
	for path := range m {
		rest, ok := strings.CutPrefix(path, prefix)
		if !ok || rest == "" {
			continue
		}
		child, _, nested := strings.Cut(rest, string(filepath.Separator))
		children[child] = children[child] || nested
	}
	if len(children) == 0 {
		return nil, fs.ErrNotExist
	}
	entries := make([]DirEntry, 0, len(children))
	for child, dir := range children {
		entries = append(entries, mapDirEntry{name: child, dir: dir})
	}
	slices.SortFunc(entries, func(a, b DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// mapDirEntry is the DirEntry an in-memory directory reports.
type mapDirEntry struct {
	name string
	dir  bool
}

func (e mapDirEntry) Name() string { return e.name }
func (e mapDirEntry) IsDir() bool  { return e.dir }

// TestDiscoverInMemoryFiles proves discovery needs no disk: a package.json and
// its lock file live only in the map, and the lock file still picks the runner.
func TestDiscoverInMemoryFiles(t *testing.T) {
	files := mapFiles{
		"package.json": `{"scripts": {"build": "go build ./..."}}`,
		"bun.lock":     "",
	}

	tasks, errs := Discover(files, ".", Settings{})
	if len(errs) != 0 {
		t.Fatalf("Discover() errors = %v, want none", errs)
	}
	if len(tasks) != 1 {
		t.Fatalf("Discover() = %+v, want exactly one task", tasks)
	}
	got := tasks[0]
	if got.Name != "build" {
		t.Errorf("task name = %q, want %q", got.Name, "build")
	}
	if want := []string{"bun", "run", "build"}; !slices.Equal(got.Args, want) {
		t.Errorf("task args = %v, want %v", got.Args, want)
	}
}

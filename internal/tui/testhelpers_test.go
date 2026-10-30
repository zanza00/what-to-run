package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zanza00/what-to-run/internal/config"
	"github.com/zanza00/what-to-run/internal/core"
)

// memFiles is an in-memory core.Files and config.Store implementation. A path
// is present when it is a key of the map, empty content included.
type memFiles map[string]string

var (
	_ core.Files   = memFiles{}
	_ config.Store = memFiles{}
)

func (m memFiles) ReadFile(name string) ([]byte, error) {
	content, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(content), nil
}

func (m memFiles) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m[name] = string(data)
	return nil
}

func (m memFiles) MkdirAll(string, fs.FileMode) error { return nil }

func (m memFiles) Remove(name string) error {
	delete(m, name)
	return nil
}

func (m memFiles) Exists(name string) bool {
	_, ok := m[name]
	return ok
}

func (m memFiles) ReadDir(name string) ([]core.DirEntry, error) {
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
	entries := make([]core.DirEntry, 0, len(children))
	for child, dir := range children {
		entries = append(entries, memDirEntry{name: child, dir: dir})
	}
	slices.SortFunc(entries, func(a, b core.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// memDirEntry is the core.DirEntry that an in-memory directory reports.
type memDirEntry struct {
	name string
	dir  bool
}

func (e memDirEntry) Name() string { return e.name }
func (e memDirEntry) IsDir() bool  { return e.dir }

// writeFile writes a fixture file inside dir, and it creates the parent
// directories for a nested path like .cargo/config.toml.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// testEnv returns an explicit environment for one picker run. The files live in
// memory under temporary paths, so a test does not depend on the process
// directory or the process environment. The caller can seed the returned map
// before it builds the model.
func testEnv(t *testing.T) (*Env, memFiles) {
	t.Helper()
	dir := t.TempDir()
	home := t.TempDir()
	cacheFile := filepath.Join(home, ".cache", "what-to-run", "dismissed-warnings")
	files := memFiles{}
	env := &Env{
		Files:       files,
		Store:       files,
		Dir:         dir,
		ProjectFile: filepath.Join(dir, ".what-to-run.toml"),
		UserFile:    filepath.Join(home, ".config", "what-to-run", "config.toml"),
		CacheFile:   cacheFile,
		Home:        home,
		Dismissed:   loadDismissedWarnings(files, cacheFile),
	}
	return env, files
}

// diskEnv returns an explicit environment on the real filesystem. The project
// directory and the user directory are temporary.
func diskEnv(t *testing.T) *Env {
	t.Helper()
	dir := t.TempDir()
	userDir := t.TempDir()
	return &Env{
		Files:       osFiles{},
		Store:       osFiles{},
		Dir:         dir,
		ProjectFile: filepath.Join(dir, ".what-to-run.toml"),
		UserFile:    filepath.Join(userDir, "what-to-run", "config.toml"),
		CacheFile:   filepath.Join(userDir, "cache", "dismissed-warnings"),
		Home:        t.TempDir(),
	}
}

// errStore fails every write with err, so a test can check the error path of a
// persistence effect.
type errStore struct {
	config.Store
	err error
}

func (s errStore) WriteFile(string, []byte, fs.FileMode) error { return s.err }

package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// memFiles is an in-memory core.Files implementation. A path is present when it
// is a key of the map, empty content included.
type memFiles map[string]string

var _ core.Files = memFiles{}

func (m memFiles) ReadFile(name string) ([]byte, error) {
	content, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(content), nil
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

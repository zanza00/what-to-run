package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// testThemes is the theme registry that the loader checks against. The real
// registry belongs to the picker.
var testThemes = []string{"classic", "nord", "dracula", "tokyo"}

// memFiles is an in-memory core.Files implementation. A path is present when it
// is a key of the map, empty content included. The rc loader never lists a
// directory, so ReadDir reports that there is none.
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

func (memFiles) ReadDir(string) ([]core.DirEntry, error) { return nil, fs.ErrNotExist }

// diskFiles is the core.Files adapter for the tests that use the real
// filesystem.
type diskFiles struct{}

var _ core.Files = diskFiles{}

func (diskFiles) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (diskFiles) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
func (diskFiles) ReadDir(name string) ([]core.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	out := make([]core.DirEntry, len(entries))
	for i, entry := range entries {
		out[i] = entry
	}
	return out, nil
}

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

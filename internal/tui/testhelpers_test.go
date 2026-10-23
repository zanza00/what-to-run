package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// memFiles is an in-memory core.Files implementation: a path is present when
// it is a key of the map, empty content included.
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

// writeFile writes a fixture file inside dir, creating parent directories for
// nested paths like .cargo/config.toml.
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

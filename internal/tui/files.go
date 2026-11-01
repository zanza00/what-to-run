package tui

import (
	"io/fs"
	"os"

	"github.com/zanza00/what-to-run/internal/config"
	"github.com/zanza00/what-to-run/internal/core"
)

// osFiles is the production adapter of the read port and of the writable
// store. The picker builds one per run.
type osFiles struct{}

var (
	_ core.Files   = osFiles{}
	_ config.Store = osFiles{}
)

func (osFiles) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFiles) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
func (osFiles) ReadDir(name string) ([]core.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	directories := make([]core.DirEntry, len(entries))
	for i, entry := range entries {
		directories[i] = entry
	}
	return directories, nil
}
func (osFiles) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFiles) MkdirAll(name string, perm fs.FileMode) error { return os.MkdirAll(name, perm) }
func (osFiles) Remove(name string) error                     { return os.Remove(name) }

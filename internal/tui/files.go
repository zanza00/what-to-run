package tui

import (
	"os"

	"github.com/zanza00/what-to-run/internal/core"
)

// osFiles is the core.Files adapter that the picker uses in production.
type osFiles struct{}

var _ core.Files = osFiles{}

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

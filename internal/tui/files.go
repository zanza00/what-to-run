package tui

import (
	"os"

	"github.com/zanza00/what-to-run/internal/core"
)

// osFiles is the production core.Files adapter.
type osFiles struct{}

var _ core.Files = osFiles{}

func (osFiles) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFiles) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

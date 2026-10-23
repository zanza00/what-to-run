package core

import (
	"io/fs"
	"os"
	"slices"
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

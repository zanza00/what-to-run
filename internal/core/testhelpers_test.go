package core

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes a fixture file inside dir. It creates the parent
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

// taskNames returns the names of the tasks, in order. A test that cares only
// about which tasks are visible compares the names against its wanted names.
func taskNames(tasks []Task) []string {
	names := make([]string, 0, len(tasks))
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	return names
}

// justTask builds the want task of one justfile recipe. A justfile task runs
// as "just <name>". The run prefix lives in this helper, so a table row
// states only the recipe name and its doc.
func justTask(name, desc string) Task {
	return Task{Name: name, Desc: desc, Args: []string{"just", name}}
}

// badgeTask sets the badge of a want task. A discovery test compares the
// badge of the found file too. The badge sits in this wrapper over the
// justTask return. The run prefix stays in justTask alone.
func badgeTask(badge string, task Task) Task {
	task.Source = badge
	return task
}

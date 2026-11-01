package tui

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestDiscoverUsesTheEnvDirectoryAndFiles pins that discovery reads the
// project directory, the project rc file and the user rc file of the
// environment. The process directory does not contribute a task.
func TestDiscoverUsesTheEnvDirectoryAndFiles(t *testing.T) {
	dir := t.TempDir()
	userDir := t.TempDir()
	projectFile := filepath.Join(dir, ".what-to-run.toml")
	userFile := filepath.Join(userDir, "what-to-run", "config.toml")
	writeFile(t, dir, "Makefile", "make-task:\n\t@echo ok\n")
	writeFile(t, dir, ".what-to-run.toml",
		"[[tasks]]\nname = \"project-task\"\ncommand = \"echo ok\"\n")
	writeFile(t, filepath.Dir(userFile), "config.toml",
		"[[tasks]]\nname = \"user-task\"\ncommand = \"echo ok\"\n")

	env := Env{
		Files:       osFiles{},
		Store:       osFiles{},
		Dir:         dir,
		ProjectFile: projectFile,
		UserFile:    userFile,
		CacheFile:   filepath.Join(t.TempDir(), "dismissed-warnings"),
		Home:        t.TempDir(),
	}

	tasks, _, warnings := env.discover()
	if len(warnings) != 0 {
		t.Fatalf("discover() warnings = %v, want none", warnings)
	}
	if got, want := taskNames(tasks), []string{"project-task", "user-task", "make-task"}; !slices.Equal(got, want) {
		t.Errorf("discover() tasks = %v, want %v", got, want)
	}
}

// TestDiscoverUsesTheProjectLabelInWarnings pins the warning text of the rc
// files. The project file keeps its short label, and the user file is shortened
// with the home of the environment.
func TestDiscoverUsesTheProjectLabelInWarnings(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	projectFile := filepath.Join(dir, ".what-to-run.toml")
	userFile := filepath.Join(home, ".config", "what-to-run", "config.toml")
	writeFile(t, dir, ".what-to-run.toml", "theme = [\n")
	writeFile(t, filepath.Dir(userFile), "config.toml", "unknown = 1\n")

	env := Env{
		Files:        osFiles{},
		Store:        osFiles{},
		Dir:          dir,
		ProjectFile:  projectFile,
		ProjectLabel: ".what-to-run.toml",
		UserFile:     userFile,
		CacheFile:    filepath.Join(t.TempDir(), "dismissed-warnings"),
		Home:         home,
	}

	_, _, warnings := env.discover()
	if len(warnings) != 2 {
		t.Fatalf("discover() warnings = %v, want the user and project files", warnings)
	}
	userDisplay := filepath.FromSlash("~/.config/what-to-run/config.toml")
	if got := warnings[0].Error(); !strings.HasPrefix(got, "rc file "+userDisplay+": ") {
		t.Errorf("user warning = %q, want rc file %s", got, userDisplay)
	}
	if got := warnings[1].Error(); !strings.HasPrefix(got, "rc file .what-to-run.toml: ") {
		t.Errorf("project warning = %q, want the short project label", got)
	}
}

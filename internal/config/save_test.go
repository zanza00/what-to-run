package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetThemeKey(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{
			"empty file",
			"",
			"theme = \"nord\"\n",
		},
		{
			"adds the key at the top",
			"package_manager = \"bun\"\n",
			"theme = \"nord\"\n\npackage_manager = \"bun\"\n",
		},
		{
			"replaces an existing key",
			"theme = \"classic\"\npackage_manager = \"bun\"\n",
			"theme = \"nord\"\npackage_manager = \"bun\"\n",
		},
		{
			"keeps comments and tables",
			"# my rc\n[[tasks]]\nname = \"x\"\ncommand = \"y\"\n",
			"theme = \"nord\"\n\n# my rc\n[[tasks]]\nname = \"x\"\ncommand = \"y\"\n",
		},
		{
			"leaves a key inside a table alone",
			"[[tasks]]\ntheme = \"classic\"\n",
			"theme = \"nord\"\n\n[[tasks]]\ntheme = \"classic\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(setThemeKey([]byte(tt.src), "nord")); got != tt.want {
				t.Errorf("setThemeKey(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

// TestSaveThemeWritesTheUserFile pins that the theme always goes to the user
// rc file, like OpenCode's cli.json, and that the project rc file stays
// unchanged.
func TestSaveThemeWritesTheUserFile(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "xdg", "what-to-run", "config.toml")
	projectFile := filepath.Join(dir, ".what-to-run.toml")
	writeFile(t, dir, ".what-to-run.toml", "package_manager = \"bun\"\n")

	path, err := SaveTheme(diskFiles{}, userFile, "nord")
	if err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}
	if path != userFile {
		t.Errorf("SaveTheme path = %q, want %q", path, userFile)
	}
	data, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("reading the user rc: %v", err)
	}
	if got := string(data); got != "theme = \"nord\"\n" {
		t.Errorf("user rc = %q, want %q", got, "theme = \"nord\"\n")
	}
	info, err := os.Stat(userFile)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("user rc mode = %o, want 0600", perm)
	}

	project, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("reading the project rc: %v", err)
	}
	if got := string(project); got != "package_manager = \"bun\"\n" {
		t.Errorf("project rc = %q, want it untouched", got)
	}
}

// TestSaveThemeNeedsAUserFile pins that the caller must resolve the user file
// before it saves.
func TestSaveThemeNeedsAUserFile(t *testing.T) {
	if _, err := SaveTheme(diskFiles{}, "", "nord"); err == nil {
		t.Error("SaveTheme without a user file returned nil, want an error")
	}
}

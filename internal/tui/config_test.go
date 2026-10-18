package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultConfigFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")

	project, global := defaultConfigFiles("proj")
	if want := filepath.Join("proj", ".what-to-run.toml"); project != want {
		t.Errorf("project rc = %q, want %q", project, want)
	}
	if want := filepath.Join("/xdg", "what-to-run", "config.toml"); global != want {
		t.Errorf("global rc = %q, want %q", global, want)
	}
}

func TestLoadConfigMissingFiles(t *testing.T) {
	cfg, errs := loadConfig(filepath.Join(t.TempDir(), ".what-to-run.toml"), "")
	if len(errs) != 0 {
		t.Errorf("loadConfig() warnings = %v, want none", errs)
	}
	if !reflect.DeepEqual(cfg, config{}) {
		t.Errorf("loadConfig() = %+v, want the zero config", cfg)
	}
}

// TestLoadConfigProjectOverridesGlobal pins the layering: the project file
// replaces scalars, adds to the disable list, and its tasks (including a
// same-named one) come first.
func TestLoadConfigProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", `
package_manager = "yarn"
disable = ["Cargo.toml"]

[[tasks]]
name = "build"
command = "global build"

[[tasks]]
name = "status"
command = "git status"
`)
	writeFile(t, dir, "project.toml", `
package_manager = "bun"
disable = ["Makefile"]

[[tasks]]
name = "build"
command = "project build"
`)

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	want := config{
		PackageManager: "bun",
		Disable:        []string{"Cargo.toml", "Makefile"},
		Tasks: []configTask{
			{Name: "build", Command: "project build"},
			{Name: "status", Command: "git status"},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigUnknownField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", "packge_manager = \"bun\"\n")

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "project.toml") {
		t.Errorf("loadConfig() warnings = %v, want one about project.toml", errs)
	}
	if !reflect.DeepEqual(cfg, config{}) {
		t.Errorf("loadConfig() = %+v, want the zero config", cfg)
	}
}

// TestLoadConfigBrokenGlobalKeepsProject: a file-level error skips that
// file only.
func TestLoadConfigBrokenGlobalKeepsProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "package_manager = [\n")
	writeFile(t, dir, "project.toml", "package_manager = \"bun\"\n")

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "global.toml") {
		t.Errorf("loadConfig() warnings = %v, want one about global.toml", errs)
	}
	if cfg.PackageManager != "bun" {
		t.Errorf("PackageManager = %q, want %q", cfg.PackageManager, "bun")
	}
}

// TestLoadConfigInvalidEntries: each bad entry is dropped with its own
// warning, and the valid settings and tasks still apply.
func TestLoadConfigInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", `
package_manager = "cargo"
disable = ["Makefile"]

[[tasks]]
name = ""
command = "x"

[[tasks]]
name = "no-command"

[[tasks]]
name = "escape"
command = "ls"
cwd = "../outside"

[[tasks]]
name = "dup"
command = "one"

[[tasks]]
name = "dup"
command = "two"

[[tasks]]
name = "ok"
command = "echo ok"
`)

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), "")
	if len(errs) != 5 {
		t.Errorf("loadConfig() warnings = %v, want 5", errs)
	}
	want := config{
		Disable: []string{"Makefile"},
		Tasks: []configTask{
			{Name: "dup", Command: "one"},
			{Name: "ok", Command: "echo ok"},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

// TestLoadConfigThemeIsAUserSetting: like OpenCode's cli.json, the theme is
// personal. The user file sets it; a project theme is ignored with a warning.
func TestLoadConfigThemeIsAUserSetting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "theme = \"nord\"\n")

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "nord")
	}

	writeFile(t, dir, "project.toml", "theme = \"tokyo\"\n")
	cfg, errs = loadConfig(filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "user setting") {
		t.Errorf("loadConfig() warnings = %v, want one about the user setting", errs)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want the global one %q", cfg.Theme, "nord")
	}
}

func TestLoadConfigUnknownTheme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "theme = \"nope\"\n")

	cfg, errs := loadConfig(filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), `"nope"`) {
		t.Errorf("loadConfig() warnings = %v, want one about %q", errs, "nope")
	}
	if cfg.Theme != "" {
		t.Errorf("Theme = %q, want it dropped", cfg.Theme)
	}
}

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

// TestSaveThemeWritesTheUserFile: the theme always goes to the user rc file,
// like OpenCode's cli.json, and a project rc file is left alone.
func TestSaveThemeWritesTheUserFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	writeFile(t, dir, ".what-to-run.toml", "package_manager = \"bun\"\n")

	path, err := saveTheme("nord")
	if err != nil {
		t.Fatalf("saveTheme: %v", err)
	}
	want := filepath.Join(dir, "xdg", "what-to-run", "config.toml")
	if path != want {
		t.Errorf("saveTheme path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("reading the user rc: %v", err)
	}
	if got := string(data); got != "theme = \"nord\"\n" {
		t.Errorf("user rc = %q, want %q", got, "theme = \"nord\"\n")
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("user rc mode = %o, want 0600", perm)
	}

	project, err := os.ReadFile(filepath.Join(dir, ".what-to-run.toml"))
	if err != nil {
		t.Fatalf("reading the project rc: %v", err)
	}
	if got := string(project); got != "package_manager = \"bun\"\n" {
		t.Errorf("project rc = %q, want it untouched", got)
	}
}

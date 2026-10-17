package tui

import (
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

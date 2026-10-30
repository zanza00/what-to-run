package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

func TestDefaultFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")

	project, global := DefaultFiles("proj")
	if want := filepath.Join("proj", ".what-to-run.toml"); project != want {
		t.Errorf("project rc = %q, want %q", project, want)
	}
	if want := filepath.Join("/xdg", "what-to-run", "config.toml"); global != want {
		t.Errorf("global rc = %q, want %q", global, want)
	}
}

// TestShortPath pins the "~" shortening against the home directory that the
// caller passes.
func TestShortPath(t *testing.T) {
	home := filepath.FromSlash("/home/user")
	tests := []struct {
		name string
		path string
		home string
		want string
	}{
		{"home itself", home, home, "~"},
		{"inside home", filepath.Join(home, "dev", "app"), home, filepath.FromSlash("~/dev/app")},
		{"outside home", filepath.FromSlash("/other/dev"), home, filepath.FromSlash("/other/dev")},
		{"sibling with the same prefix", home + "-2", home, home + "-2"},
		{"no home", filepath.FromSlash("/home/user/dev"), "", filepath.FromSlash("/home/user/dev")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortPath(tt.path, tt.home); got != tt.want {
				t.Errorf("ShortPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestLoadInMemoryFiles proves that the rc loader reads through the Files port.
// The project file exists only in the map.
func TestLoadInMemoryFiles(t *testing.T) {
	files := memFiles{"project.toml": "package_manager = \"bun\"\n"}

	cfg, errs := Load(files, Paths{Project: "project.toml"}, testThemes)
	if len(errs) != 0 {
		t.Fatalf("Load() warnings = %v, want none", errs)
	}
	if cfg.PackageManager != "bun" {
		t.Errorf("Load() package_manager = %q, want %q", cfg.PackageManager, "bun")
	}
}

func TestLoadMissingFiles(t *testing.T) {
	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(t.TempDir(), ".what-to-run.toml")}, testThemes)
	if len(errs) != 0 {
		t.Errorf("Load() warnings = %v, want none", errs)
	}
	if !reflect.DeepEqual(cfg, Config{}) {
		t.Errorf("Load() = %+v, want the zero config", cfg)
	}
}

// TestLoadProjectOverridesGlobal pins the layering. The project file replaces
// the scalars, adds to the disable list, and its tasks (including a same-named
// one) come first.
func TestLoadProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", `
package_manager = "yarn"
disable = ["cargo-make"]

[[tasks]]
name = "build"
command = "global build"

[[tasks]]
name = "status"
command = "git status"
`)
	writeFile(t, dir, "project.toml", `
package_manager = "bun"
disable = ["make"]

[[tasks]]
name = "build"
command = "project build"
`)

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml"), Global: filepath.Join(dir, "global.toml")}, testThemes)
	if len(errs) != 0 {
		t.Fatalf("Load() warnings = %v, want none", errs)
	}
	want := Config{
		Settings: core.Settings{
			PackageManager: "bun",
			Disable:        []string{"cargo-make", "make"},
			Tasks: []core.RCTask{
				{Name: "build", Command: "project build"},
				{Name: "status", Command: "git status"},
			},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadUnknownField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", `
packge_manager = "bun"
package_manager = "pnpm"
disable = ["make", 42]

[[tasks]]
name = "ok"
command = "echo ok"
extra_key = "ignored"

[[tasks]]
name = 123
command = "bad"
`)

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml")}, testThemes)
	if len(errs) != 4 {
		t.Fatalf("Load() warnings = %v, want 4", errs)
	}
	for _, wantErr := range []string{
		"disable entries must be strings",
		`unknown key "packge_manager"`,
		`task has unknown key "extra_key"`,
		`task field "name" must be a string`,
	} {
		if !slices.ContainsFunc(errs, func(err error) bool { return strings.Contains(err.Error(), wantErr) }) {
			t.Errorf("Load() warnings = %v, missing %q", errs, wantErr)
		}
	}
	want := Config{
		Settings: core.Settings{
			PackageManager: "pnpm",
			Disable:        []string{"make"},
			Tasks:          []core.RCTask{{Name: "ok", Command: "echo ok"}},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadProjectsSubtreeAndPrecedence(t *testing.T) {
	root := t.TempDir()
	realHome := filepath.Join(root, "var", "home", "user")
	linkHome := filepath.Join(root, "home", "user")
	if err := os.MkdirAll(filepath.Join(realHome, "work", "api", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(realHome, "work", "api-2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(linkHome), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realHome, linkHome); err != nil {
		t.Fatal(err)
	}

	globalFile := filepath.Join(root, "global.toml")
	writeFile(t, root, "global.toml", `
theme = "dracula"
package_manager = "npm"
disable = ["cargo-make"]

[[tasks]]
name = "shared"
command = "global shared"

[projects."~/work"]
package_manager = "pnpm"
disable = ["make"]

[[projects."~/work".tasks]]
name = "shared"
command = "work shared"

[[projects."~/work".tasks]]
name = "work-only"
command = "echo work"

[projects."~/work/api"]
package_manager = "bun"

[[projects."~/work/api".tasks]]
name = "shared"
command = "api shared"
`)

	apiSub := filepath.Join(realHome, "work", "api", "sub")
	writeFile(t, apiSub, ".what-to-run.toml", `
package_manager = "yarn"

[[tasks]]
name = "shared"
command = "local shared"

[[tasks]]
name = "local-only"
command = "echo local"
`)

	// The Load call reaches ~/work/api/sub through the real /var/home path,
	// while the config uses ~/ through the /home symlink. The precedence is
	// global < local < ~/work < ~/work/api.
	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(apiSub, ".what-to-run.toml"), Global: globalFile, Home: linkHome}, testThemes)
	if len(errs) != 0 {
		t.Fatalf("Load() warnings = %v, want none", errs)
	}
	want := Config{
		Settings: core.Settings{
			PackageManager: "bun",
			Disable:        []string{"cargo-make", "make"},
			Tasks: []core.RCTask{
				{Name: "shared", Command: "api shared"},
				{Name: "work-only", Command: "echo work"},
				{Name: "local-only", Command: "echo local"},
			},
		},
		Theme: "dracula",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load(api/sub) = %+v, want %+v", cfg, want)
	}

	// Inside ~/work/api-2 the block ~/work matches, but ~/work/api must not
	// match, because the prefix is not a directory boundary.
	api2 := filepath.Join(realHome, "work", "api-2")
	cfg2, errs := Load(diskFiles{}, Paths{Project: filepath.Join(api2, ".what-to-run.toml"), Global: globalFile, Home: linkHome}, testThemes)
	if len(errs) != 0 {
		t.Fatalf("Load(api-2) warnings = %v, want none", errs)
	}
	if cfg2.PackageManager != "pnpm" {
		t.Errorf("api-2 PackageManager = %q, want %q (from ~/work, not ~/work/api)", cfg2.PackageManager, "pnpm")
	}
}

func TestLoadProjectsValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", `
[projects."relative/path"]
package_manager = "bun"

[projects."~"]
package_manager = "pnpm"
theme = "nord"
unknown = 1
`)
	writeFile(t, dir, ".what-to-run.toml", `
[projects."~"]
package_manager = "yarn"
`)

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, ".what-to-run.toml"), Global: filepath.Join(dir, "global.toml"), Home: dir}, testThemes)
	if len(errs) != 4 {
		t.Fatalf("Load() warnings = %v, want 4", errs)
	}
	if cfg.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q from valid [projects.\"~\"]", cfg.PackageManager, "pnpm")
	}
}

// TestLoadBrokenGlobalKeepsProject checks that a file-level error skips only
// that file.
func TestLoadBrokenGlobalKeepsProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "package_manager = [\n")
	writeFile(t, dir, "project.toml", "package_manager = \"bun\"\n")

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml"), Global: filepath.Join(dir, "global.toml")}, testThemes)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "global.toml") {
		t.Errorf("Load() warnings = %v, want one about global.toml", errs)
	}
	if cfg.PackageManager != "bun" {
		t.Errorf("PackageManager = %q, want %q", cfg.PackageManager, "bun")
	}
}

// TestLoadInvalidEntries checks that each bad entry is dropped with its own
// warning, and that the valid settings and tasks still apply.
func TestLoadInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", `
package_manager = "cargo"
disable = ["make"]

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

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml")}, testThemes)
	if len(errs) != 5 {
		t.Errorf("Load() warnings = %v, want 5", errs)
	}
	want := Config{
		Settings: core.Settings{
			Disable: []string{"make"},
			Tasks: []core.RCTask{
				{Name: "dup", Command: "one"},
				{Name: "ok", Command: "echo ok"},
			},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

// TestLoadDisableFileName pins that disable takes source IDs. The loader drops
// a file name at the top level or in a [projects."<path>"] block with a warning
// that carries the file, the scope and the ID to use. A valid ID next to it
// stays.
func TestLoadDisableFileName(t *testing.T) {
	dir := t.TempDir()
	globalFile := filepath.Join(dir, "global.toml")
	writeFile(t, dir, "global.toml", `
disable = ["Makefile.toml", "cargo-make"]

[projects."~"]
disable = ["Makefile", "make"]
`)

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, ".what-to-run.toml"), Global: globalFile, Home: dir}, testThemes)
	wantErrs := []string{
		`rc file ~/global.toml: disable: "Makefile.toml" is a file name; use "cargo-make"`,
		`rc file ~/global.toml: projects."~": disable: "Makefile" is a file name; use "make"`,
	}
	if len(errs) != len(wantErrs) {
		t.Fatalf("Load() warnings = %v, want %d", errs, len(wantErrs))
	}
	for i, want := range wantErrs {
		if errs[i].Error() != want {
			t.Errorf("Load() warning %d = %q, want %q", i, errs[i], want)
		}
	}
	want := Config{Settings: core.Settings{Disable: []string{"cargo-make", "make"}}}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

// TestLoadThemeIsAUserSetting: like OpenCode's cli.json, the theme is personal.
// The user file sets it, and the picker ignores a project theme with a warning.
func TestLoadThemeIsAUserSetting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "theme = \"nord\"\n")

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml"), Global: filepath.Join(dir, "global.toml")}, testThemes)
	if len(errs) != 0 {
		t.Fatalf("Load() warnings = %v, want none", errs)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "nord")
	}

	writeFile(t, dir, "project.toml", "theme = \"tokyo\"\n")
	cfg, errs = Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml"), Global: filepath.Join(dir, "global.toml")}, testThemes)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "user setting") {
		t.Errorf("Load() warnings = %v, want one about the user setting", errs)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want the global one %q", cfg.Theme, "nord")
	}
}

func TestLoadUnknownTheme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "theme = \"nope\"\n")

	cfg, errs := Load(diskFiles{}, Paths{Project: filepath.Join(dir, "project.toml"), Global: filepath.Join(dir, "global.toml")}, testThemes)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), `"nope"`) {
		t.Errorf("Load() warnings = %v, want one about %q", errs, "nope")
	}
	if cfg.Theme != "" {
		t.Errorf("Theme = %q, want it dropped", cfg.Theme)
	}
}

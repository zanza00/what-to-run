package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

// TestLoadConfigInMemoryFiles proves the rc loader reads through the Files
// port: the project file exists only in the map.
func TestLoadConfigInMemoryFiles(t *testing.T) {
	files := memFiles{"project.toml": "package_manager = \"bun\"\n"}

	cfg, errs := loadConfig(files, "project.toml", "")
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	if cfg.PackageManager != "bun" {
		t.Errorf("loadConfig() package_manager = %q, want %q", cfg.PackageManager, "bun")
	}
}

func TestLoadConfigMissingFiles(t *testing.T) {
	cfg, errs := loadConfig(osFiles{}, filepath.Join(t.TempDir(), ".what-to-run.toml"), "")
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

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	want := config{
		PackageManager: "bun",
		Disable:        []string{"cargo-make", "make"},
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

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), "")
	if len(errs) != 4 {
		t.Fatalf("loadConfig() warnings = %v, want 4", errs)
	}
	for _, wantErr := range []string{
		"disable entries must be strings",
		`unknown key "packge_manager"`,
		`task has unknown key "extra_key"`,
		`task field "name" must be a string`,
	} {
		if !slices.ContainsFunc(errs, func(err error) bool { return strings.Contains(err.Error(), wantErr) }) {
			t.Errorf("loadConfig() warnings = %v, missing %q", errs, wantErr)
		}
	}
	want := config{
		PackageManager: "pnpm",
		Disable:        []string{"make"},
		Tasks:          []configTask{{Name: "ok", Command: "echo ok"}},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigProjectsSubtreeAndPrecedence(t *testing.T) {
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
	t.Setenv("HOME", linkHome)

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

	// Inside ~/work/api/sub (accessed via the real /var/home path while
	// config uses ~/ through the /home symlink): global < local < ~/work < ~/work/api.
	cfg, errs := loadConfig(osFiles{}, filepath.Join(apiSub, ".what-to-run.toml"), globalFile)
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	want := config{
		Theme:          "dracula",
		PackageManager: "bun",
		Disable:        []string{"cargo-make", "make"},
		Tasks: []configTask{
			{Name: "shared", Command: "api shared"},
			{Name: "work-only", Command: "echo work"},
			{Name: "local-only", Command: "echo local"},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig(api/sub) = %+v, want %+v", cfg, want)
	}

	// Inside ~/work/api-2: ~/work matches, but ~/work/api must NOT match on
	// a prefix that is not a directory boundary.
	api2 := filepath.Join(realHome, "work", "api-2")
	cfg2, errs := loadConfig(osFiles{}, filepath.Join(api2, ".what-to-run.toml"), globalFile)
	if len(errs) != 0 {
		t.Fatalf("loadConfig(api-2) warnings = %v, want none", errs)
	}
	if cfg2.PackageManager != "pnpm" {
		t.Errorf("api-2 PackageManager = %q, want %q (from ~/work, not ~/work/api)", cfg2.PackageManager, "pnpm")
	}
}

func TestLoadConfigProjectsValidation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

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

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, ".what-to-run.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 4 {
		t.Fatalf("loadConfig() warnings = %v, want 4", errs)
	}
	if cfg.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q from valid [projects.\"~\"]", cfg.PackageManager, "pnpm")
	}
}

// TestLoadConfigBrokenGlobalKeepsProject: a file-level error skips that
// file only.
func TestLoadConfigBrokenGlobalKeepsProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "package_manager = [\n")
	writeFile(t, dir, "project.toml", "package_manager = \"bun\"\n")

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
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

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), "")
	if len(errs) != 5 {
		t.Errorf("loadConfig() warnings = %v, want 5", errs)
	}
	want := config{
		Disable: []string{"make"},
		Tasks: []configTask{
			{Name: "dup", Command: "one"},
			{Name: "ok", Command: "echo ok"},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

// TestLoadConfigDisableFileName: disable takes source IDs, so a file name at
// the top level or in a [projects."<path>"] block is dropped with a warning
// carrying the file, the scope and the ID to use; valid IDs next to it stay.
func TestLoadConfigDisableFileName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	globalFile := filepath.Join(dir, "global.toml")
	writeFile(t, dir, "global.toml", `
disable = ["Makefile.toml", "cargo-make"]

[projects."~"]
disable = ["Makefile", "make"]
`)

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, ".what-to-run.toml"), globalFile)
	wantErrs := []string{
		`rc file ~/global.toml: disable: "Makefile.toml" is a file name; use "cargo-make"`,
		`rc file ~/global.toml: projects."~": disable: "Makefile" is a file name; use "make"`,
	}
	if len(errs) != len(wantErrs) {
		t.Fatalf("loadConfig() warnings = %v, want %d", errs, len(wantErrs))
	}
	for i, want := range wantErrs {
		if errs[i].Error() != want {
			t.Errorf("loadConfig() warning %d = %q, want %q", i, errs[i], want)
		}
	}
	want := config{Disable: []string{"cargo-make", "make"}}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

// TestLoadConfigThemeIsAUserSetting: like OpenCode's cli.json, the theme is
// personal. The user file sets it; a project theme is ignored with a warning.
func TestLoadConfigThemeIsAUserSetting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.toml", "theme = \"nord\"\n")

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
	if len(errs) != 0 {
		t.Fatalf("loadConfig() warnings = %v, want none", errs)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "nord")
	}

	writeFile(t, dir, "project.toml", "theme = \"tokyo\"\n")
	cfg, errs = loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
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

	cfg, errs := loadConfig(osFiles{}, filepath.Join(dir, "project.toml"), filepath.Join(dir, "global.toml"))
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

package config

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/zanza00/what-to-run/internal/core"
)

// warningWant is the identity and the message of one warning a test expects.
type warningWant struct {
	kind    core.WarningKind
	key     string
	message string // empty means the test leaves the message alone
}

// TestLoadWarningIdentity pins the identity that the rc loader declares at the
// place it creates each warning: the kind and the dismissal key. The key keeps
// the file origin, the problem and the offending value apart, so one dismissed
// problem does not silence a new one.
func TestLoadWarningIdentity(t *testing.T) {
	tests := []struct {
		name    string
		project string
		global  string
		want    []warningWant
	}{
		{
			name:    "an unknown key of the project file",
			project: "packge_manager = \"bun\"\n",
			want: []warningWant{
				{core.KindRC, "rc:project:unknown-key.packge_manager", `rc file .what-to-run.toml: unknown key "packge_manager"`},
			},
		},
		{
			name:   "a syntax error of the user file",
			global: "theme = [\n",
			want: []warningWant{
				{core.KindRC, "rc:user:syntax", ""},
			},
		},
		{
			name:    "an unknown theme of the user file",
			global:  "theme = \"nope\"\n",
			project: "disable = [\"make\"]\n",
			want: []warningWant{
				{core.KindTheme, "theme:user:nope", `rc file ~/global.toml: unknown theme "nope"`},
			},
		},
		{
			name:   "a theme of the wrong type",
			global: "theme = 3\n",
			want: []warningWant{
				{core.KindTheme, "theme:user", "rc file ~/global.toml: theme must be a string"},
			},
		},
		{
			name:    "a theme in the project file",
			project: "theme = \"tokyo\"\n",
			want: []warningWant{
				{core.KindTheme, "theme:project", "rc file .what-to-run.toml: theme is a user setting, ignored here"},
			},
		},
		{
			name:    "a bad disable entry is a registry warning",
			project: "disable = [\"nmake\"]\n",
			want: []warningWant{
				{core.KindRegistry, "registry:nmake", ""},
			},
		},
		{
			name:    "a duplicate task names the task",
			project: "[[tasks]]\nname = \"dup\"\ncommand = \"one\"\n\n[[tasks]]\nname = \"dup\"\ncommand = \"two\"\n",
			want: []warningWant{
				{core.KindRC, "rc:project:task-duplicate.dup", ""},
			},
		},
		{
			name:   "a problem inside a project block carries the block origin",
			global: "[projects.\"~/work\"]\npackge_manager = \"pnpm\"\n",
			want: []warningWant{
				{core.KindRC, "rc:projects/~/work:unknown-key.packge_manager", ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "project.toml", tt.project)
			writeFile(t, dir, "global.toml", tt.global)

			// Home is dir, so the messages name the user file as
			// ~/global.toml, the way the production run does.
			_, warnings := Load(diskFiles{}, Paths{
				Project:      filepath.Join(dir, "project.toml"),
				Global:       filepath.Join(dir, "global.toml"),
				ProjectLabel: ".what-to-run.toml",
				Home:         dir,
			}, testThemes)

			if len(warnings) != len(tt.want) {
				t.Fatalf("Load() warnings = %v, want %d", warnings, len(tt.want))
			}
			for i, want := range tt.want {
				got := warnings[i]
				if got.Kind() != want.kind || got.Key() != want.key {
					t.Errorf("warning %d identity = %s %q, want %s %q", i,
						got.Kind(), got.Key(), want.kind, want.key)
				}
				if want.message != "" && got.Error() != want.message {
					t.Errorf("warning %d message = %q, want %q", i, got.Error(), want.message)
				}
			}
		})
	}
}

// TestLoadKeyIgnoresDisplayLabel pins that the display label never enters the
// key: the project rc file is "project" in the key, whatever the label says.
func TestLoadKeyIgnoresDisplayLabel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", "packge_manager = \"bun\"\n")
	project := filepath.Join(dir, "project.toml")
	const wantKey = "rc:project:unknown-key.packge_manager"

	var keys []string
	for _, label := range []string{".what-to-run.toml", "my config", ""} {
		_, warnings := Load(diskFiles{}, Paths{Project: project, ProjectLabel: label, Home: dir}, testThemes)
		if len(warnings) != 1 {
			t.Fatalf("Load() with label %q warnings = %v, want one", label, warnings)
		}
		if warnings[0].Kind() != core.KindRC {
			t.Errorf("kind = %s, want rc", warnings[0].Kind())
		}
		keys = append(keys, warnings[0].Key())
	}
	if !slices.Equal(keys, []string{wantKey, wantKey, wantKey}) {
		t.Errorf("keys change with the label: %v, want one stable key %q", keys, wantKey)
	}
}

// TestLoadDisableEntryKeysEqualAcrossFiles pins the collapse of the disable
// twin: the same wrong entry typed in the user file and in the project file
// produces two warnings with one identity, so the picker can show one warning
// instead of two.
func TestLoadDisableEntryKeysEqualAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "project.toml", "disable = [\"nmake\"]\n")
	writeFile(t, dir, "global.toml", "disable = [\"nmake\"]\n")

	_, warnings := Load(diskFiles{}, Paths{
		Project: filepath.Join(dir, "project.toml"),
		Global:  filepath.Join(dir, "global.toml"),
	}, testThemes)
	if len(warnings) != 2 {
		t.Fatalf("Load() warnings = %v, want two", warnings)
	}
	if warnings[0].Key() != warnings[1].Key() {
		t.Errorf("keys = %q and %q, want equal", warnings[0].Key(), warnings[1].Key())
	}
	if warnings[0].Kind() != core.KindRegistry {
		t.Errorf("kind = %s, want registry", warnings[0].Kind())
	}
}

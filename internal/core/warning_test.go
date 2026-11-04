package core

import (
	"testing"
)

// TestWarningKeyGrammar pins the dismissal key of every constructor: the kind
// opens the key, the subject names the offending value, and the optional detail
// tells two warnings apart. The key is the line the dismissed-warnings file
// stores, so the table also documents what a user reads there.
func TestWarningKeyGrammar(t *testing.T) {
	tests := []struct {
		name     string
		warning  Warning
		wantKind WarningKind
		wantKey  string
		wantMsg  string
	}{
		{
			name:     "a broken file of a source keys on the ID and the leaf",
			warning:  NewSourceWarning("just", "justfile", "justfile: bad recipe"),
			wantKind: KindSource,
			wantKey:  "source:just:justfile",
			wantMsg:  "justfile: bad recipe",
		},
		{
			name: "a file found above the project keeps the leaf alone",
			warning: NewSourceWarning("just", "justfile",
				"../justfile: bad recipe"),
			wantKind: KindSource,
			wantKey:  "source:just:justfile",
			wantMsg:  "../justfile: bad recipe",
		},
		{
			name: "a bad disable entry keys on the raw stored value",
			warning: NewRegistryWarning("nmake",
				`disable: unknown source "nmake"`),
			wantKind: KindRegistry,
			wantKey:  "registry:nmake",
			wantMsg:  `disable: unknown source "nmake"`,
		},
		{
			name:     "an unusable shell value keys on the raw stored value",
			warning:  NewShellWarning("fish", `unknown shell "fish"`),
			wantKind: KindShell,
			wantKey:  "shell:fish",
			wantMsg:  `unknown shell "fish"`,
		},
		{
			name: "an unknown theme of the flag names the origin and the theme",
			warning: NewThemeWarning("flag", "nope",
				`unknown theme "nope"`),
			wantKind: KindTheme,
			wantKey:  "theme:flag:nope",
			wantMsg:  `unknown theme "nope"`,
		},
		{
			name: "an unknown theme of the user file names the origin and the theme",
			warning: NewThemeWarning("user", "nope",
				`rc file ~/.config/what-to-run/config.toml: unknown theme "nope"`),
			wantKind: KindTheme,
			wantKey:  "theme:user:nope",
			wantMsg:  `rc file ~/.config/what-to-run/config.toml: unknown theme "nope"`,
		},
		{
			name:     "a theme type mistake has no theme name",
			warning:  NewThemeWarning("user", "", "rc file ~/.config: theme must be a string"),
			wantKind: KindTheme,
			wantKey:  "theme:user",
			wantMsg:  "rc file ~/.config: theme must be a string",
		},
		{
			name:     "a theme in the project file is one subject",
			warning:  NewThemeWarning("project", "", "rc file .what-to-run.toml: theme is a user setting"),
			wantKind: KindTheme,
			wantKey:  "theme:project",
			wantMsg:  "rc file .what-to-run.toml: theme is a user setting",
		},
		{
			name: "an rc problem names the origin, the problem and the value",
			warning: NewRCWarning("project", "unknown-key", "packge_manager",
				`rc file .what-to-run.toml: unknown key "packge_manager"`),
			wantKind: KindRC,
			wantKey:  "rc:project:unknown-key.packge_manager",
			wantMsg:  `rc file .what-to-run.toml: unknown key "packge_manager"`,
		},
		{
			name: "an rc problem without an offending value drops the detail part",
			warning: NewRCWarning("user", "syntax", "",
				"rc file ~/.config/what-to-run/config.toml: toml: line 1: boom"),
			wantKind: KindRC,
			wantKey:  "rc:user:syntax",
			wantMsg:  "rc file ~/.config/what-to-run/config.toml: toml: line 1: boom",
		},
		{
			name: "a project block forms the origin with its path",
			warning: NewRCWarning("projects/~/work", "unknown-key", "packge_manager",
				`rc file ~/.config/what-to-run/config.toml: projects."~/work": unknown key "packge_manager"`),
			wantKind: KindRC,
			wantKey:  "rc:projects/~/work:unknown-key.packge_manager",
			wantMsg:  `rc file ~/.config/what-to-run/config.toml: projects."~/work": unknown key "packge_manager"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.warning.Key(); got != tt.wantKey {
				t.Errorf("Key() = %q, want %q", got, tt.wantKey)
			}
			if got := tt.warning.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := tt.warning.Kind(); got != tt.wantKind {
				t.Errorf("Kind() = %s, want %s", got, tt.wantKind)
			}
		})
	}
}

// TestWarningKeySurvivesAReword is the release guarantee: the message can be
// reworded freely, and a warning keeps its identity, so a dismissal made
// before the release still matches after it.
func TestWarningKeySurvivesAReword(t *testing.T) {
	before := NewRCWarning("user", "unknown-key", "packge_manager",
		`rc file ~/.config/what-to-run/config.toml: unknown key "packge_manager"`)
	after := NewRCWarning("user", "unknown-key", "packge_manager",
		"rc file config.toml: the key \"packge_manager\" is not known")
	if before.Key() != after.Key() {
		t.Errorf("a reword changes the key: before %q, after %q", before.Key(), after.Key())
	}

	sourceBefore := NewSourceWarning("make", "Makefile", "Makefile: boom")
	sourceAfter := NewSourceWarning("make", "Makefile", "Makefile: unknown directive at line 3")
	if sourceBefore.Key() != sourceAfter.Key() {
		t.Errorf("a reword changes the key: before %q, after %q", sourceBefore.Key(), sourceAfter.Key())
	}
}

// TestWarningKeyHoldsOneLineWithoutSpaces pins the sanitizer: an offending
// value with whitespace still yields a key the dismissed-warnings file can
// store as one entry per line.
func TestWarningKeyHoldsOneLineWithoutSpaces(t *testing.T) {
	tests := []struct {
		name    string
		warning Warning
		wantKey string
	}{
		{
			name:    "spaces collapse to one underscore",
			warning: NewRegistryWarning("two words", "disable: unknown source"),
			wantKey: "registry:two_words",
		},
		{
			name:    "newlines and tabs collapse too",
			warning: NewThemeWarning("user", "a\ttheme\nname", "theme"),
			wantKey: "theme:user:a_theme_name",
		},
		{
			name:    "leading and trailing whitespace trims away",
			warning: NewShellWarning("  fish  ", "unknown shell"),
			wantKey: "shell:fish",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.warning.Key(); got != tt.wantKey {
				t.Errorf("Key() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

// TestWarningImplementsError keeps the message the error interface returns:
// the display and the no-tasks error derive from it exactly as before.
func TestWarningImplementsError(t *testing.T) {
	var err error = NewSourceWarning("make", "Makefile", "Makefile: boom")
	if got, want := err.Error(), "Makefile: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

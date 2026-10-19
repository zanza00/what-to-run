package tui

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverComposerTasks(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    []Task
		wantErr bool
	}{
		{
			name: "custom scripts sorted, description and array body",
			file: `{
				"require": {"php": ">=8.1"},
				"scripts": {
					"test": "phpunit",
					"build": ["composer dump-autoload", "php bin/build.php"]
				},
				"scripts-descriptions": {
					"test": "Run the test suite"
				}
			}`,
			want: []Task{
				{Source: "composer.json", Name: "build", Desc: "composer dump-autoload && php bin/build.php", Args: []string{"composer", "run-script", "build"}},
				{Source: "composer.json", Name: "test", Desc: "Run the test suite", Args: []string{"composer", "run-script", "test"}},
			},
		},
		{
			name: "lifecycle event is excluded",
			file: `{"scripts": {
				"post-install-cmd": "php artisan migrate",
				"serve": "php -S localhost:8000"
			}}`,
			want: []Task{
				{Source: "composer.json", Name: "serve", Desc: "php -S localhost:8000", Args: []string{"composer", "run-script", "serve"}},
			},
		},
		{
			name: "lifecycle names are matched case-sensitively",
			file: `{"scripts": {"Post-Install-Cmd": "echo hi"}}`,
			want: []Task{
				{Source: "composer.json", Name: "Post-Install-Cmd", Desc: "echo hi", Args: []string{"composer", "run-script", "Post-Install-Cmd"}},
			},
		},
		{
			name: "scripts-aliases are ignored",
			file: `{
				"scripts": {"test": "phpunit"},
				"scripts-aliases": {"test": ["unit", "check"]}
			}`,
			want: []Task{
				{Source: "composer.json", Name: "test", Desc: "phpunit", Args: []string{"composer", "run-script", "test"}},
			},
		},
		{
			name: "scripts without usable commands are skipped",
			file: `{"scripts": {
				"good": "echo good",
				"also-good": ["echo one", "echo two"],
				"number": 42,
				"object": {"cmd": "echo hi"},
				"mixed": ["echo hi", 7],
				"empty": "",
				"empty-list": []
			}}`,
			want: []Task{
				{Source: "composer.json", Name: "also-good", Desc: "echo one && echo two", Args: []string{"composer", "run-script", "also-good"}},
				{Source: "composer.json", Name: "good", Desc: "echo good", Args: []string{"composer", "run-script", "good"}},
			},
		},
		{
			name:    "malformed json",
			file:    `{"scripts": `,
			wantErr: true,
		},
		{
			name: "empty file",
			file: "",
			want: nil,
		},
		{
			name: "whitespace only file",
			file: " \n\t ",
			want: nil,
		},
		{
			name: "no scripts object",
			file: `{"require": {"php": ">=8.1"}}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "composer.json", tt.file)

			got, err := discoverSource(composerSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "composer.json") {
					t.Errorf("error = %q, want it to mention composer.json", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("discoverSource() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discoverSource() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestDiscoverComposerLifecycleEvents pins the excluded names: every
// documented Composer lifecycle and plugin event is filtered out, while a
// custom script next to them still loads.
func TestDiscoverComposerLifecycleEvents(t *testing.T) {
	events := []string{
		"pre-install-cmd",
		"post-install-cmd",
		"pre-update-cmd",
		"post-update-cmd",
		"pre-status-cmd",
		"post-status-cmd",
		"pre-archive-cmd",
		"post-archive-cmd",
		"pre-autoload-dump",
		"post-autoload-dump",
		"post-root-package-install",
		"post-create-project-cmd",
		"pre-operations-exec",
		"pre-package-install",
		"post-package-install",
		"pre-package-update",
		"post-package-update",
		"pre-package-uninstall",
		"post-package-uninstall",
		"init",
		"command",
		"pre-file-download",
		"post-file-download",
		"pre-command-run",
		"pre-pool-create",
	}

	scripts := make(map[string]string, len(events)+1)
	for _, name := range events {
		scripts[name] = "echo " + name
	}
	scripts["custom"] = "echo custom"
	data, err := json.Marshal(map[string]any{"scripts": scripts})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	writeFile(t, dir, "composer.json", string(data))

	got, err := discoverSource(composerSource(), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{
		{Source: "composer.json", Name: "custom", Desc: "echo custom", Args: []string{"composer", "run-script", "custom"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("discoverSource() = %+v, want only the custom task", got)
	}
}

func TestDiscoverComposerTasksMissingFile(t *testing.T) {
	got, err := discoverSource(composerSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

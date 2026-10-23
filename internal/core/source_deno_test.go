package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverDenoTasks(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string and object tasks sorted",
			files: map[string]string{
				"deno.json": `{
					"tasks": {
						"dev": "deno run -A main.ts",
						"build": {
							"command": "deno run -A build.ts",
							"description": "Build the app"
						},
						"test": {"command": "deno test -A"}
					}
				}`,
			},
			want: []Task{
				{Source: "deno.json", Name: "build", Desc: "Build the app", Args: []string{"deno", "task", "build"}},
				{Source: "deno.json", Name: "dev", Desc: "deno run -A main.ts", Args: []string{"deno", "task", "dev"}},
				{Source: "deno.json", Name: "test", Desc: "deno test -A", Args: []string{"deno", "task", "test"}},
			},
		},
		{
			name: "deno.jsonc comments and trailing commas",
			files: map[string]string{
				"deno.jsonc": `{
					// the tasks of this project
					"tasks": {
						"lint": "deno lint", // inline comment
						/* a task with an object body */
						"fmt": {
							"command": "deno fmt",
							"description": "Format the sources",
						},
					},
				}`,
			},
			want: []Task{
				{Source: "deno.jsonc", Name: "fmt", Desc: "Format the sources", Args: []string{"deno", "task", "fmt"}},
				{Source: "deno.jsonc", Name: "lint", Desc: "deno lint", Args: []string{"deno", "task", "lint"}},
			},
		},
		{
			name: "tasks without a usable command are skipped",
			files: map[string]string{
				"deno.json": `{"tasks": {
					"good": "deno run -A main.ts",
					"no-command": {"description": "Missing the command"},
					"empty": "",
					"number": 42,
					"empty-command": {"command": ""}
				}}`,
			},
			want: []Task{
				{Source: "deno.json", Name: "good", Desc: "deno run -A main.ts", Args: []string{"deno", "task", "good"}},
			},
		},
		{
			name:    "malformed json",
			files:   map[string]string{"deno.json": `{"tasks": `},
			wantErr: true,
		},
		{
			name:  "empty file",
			files: map[string]string{"deno.json": ""},
			want:  nil,
		},
		{
			name:  "whitespace only file",
			files: map[string]string{"deno.json": " \n\t "},
			want:  nil,
		},
		{
			name:  "no tasks object",
			files: map[string]string{"deno.json": `{"imports": {"std/": "https://deno.land/std/"}}`},
			want:  nil,
		},
		{
			name: "deno.json wins over deno.jsonc",
			files: map[string]string{
				"deno.json":  `{"tasks": {"from-json": "deno run -A a.ts"}}`,
				"deno.jsonc": `{"tasks": {"from-jsonc": "deno run -A b.ts"}}`,
			},
			want: []Task{
				{Source: "deno.json", Name: "from-json", Desc: "deno run -A a.ts", Args: []string{"deno", "task", "from-json"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(osFiles{}, denoSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "deno.json") {
					t.Errorf("error = %q, want it to mention deno.json", err)
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

func TestDiscoverDenoTasksMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, denoSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

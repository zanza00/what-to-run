package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverMiseTasks(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "string, table and array tasks sorted",
			files: map[string]string{
				"mise.toml": `
[tasks]
simple = "echo simple"

[tasks.build]
description = "Build the project"
run = "go build ./..."

[tasks.ci]
run = ["go vet ./...", "go test -race ./..."]

[tasks.dev]
run = "npm run dev"
`,
			},
			want: []Task{
				{Source: "mise.toml", Name: "build", Desc: "Build the project", Args: []string{"mise", "run", "build"}},
				{Source: "mise.toml", Name: "ci", Desc: "go vet ./... && go test -race ./...", Args: []string{"mise", "run", "ci"}},
				{Source: "mise.toml", Name: "dev", Desc: "npm run dev", Args: []string{"mise", "run", "dev"}},
				{Source: "mise.toml", Name: "simple", Desc: "echo simple", Args: []string{"mise", "run", "simple"}},
			},
		},
		{
			name: "depends-only tasks are kept with an empty description",
			files: map[string]string{
				"mise.toml": `
[tasks.build]
run = "go build ./..."

[tasks.ci]
depends = ["build", "test"]

[tasks.test]
depends = "build"
`,
			},
			want: []Task{
				{Source: "mise.toml", Name: "build", Desc: "go build ./...", Args: []string{"mise", "run", "build"}},
				{Source: "mise.toml", Name: "ci", Desc: "", Args: []string{"mise", "run", "ci"}},
				{Source: "mise.toml", Name: "test", Desc: "", Args: []string{"mise", "run", "test"}},
			},
		},
		{
			name: "values that are neither strings nor tables are skipped",
			files: map[string]string{
				"mise.toml": `
[tasks]
good = "echo good"
number = 42
flag = true
list = ["echo one", "echo two"]
`,
			},
			want: []Task{
				{Source: "mise.toml", Name: "good", Desc: "echo good", Args: []string{"mise", "run", "good"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{"mise.toml": "[tasks\n"},
			wantErr: true,
		},
		{
			name:  "empty file",
			files: map[string]string{"mise.toml": ""},
			want:  nil,
		},
		{
			name:  "whitespace only file",
			files: map[string]string{"mise.toml": " \n\t "},
			want:  nil,
		},
		{
			name:  "no tasks table",
			files: map[string]string{"mise.toml": "[tools]\ngo = \"1.23\"\n"},
			want:  nil,
		},
		{
			name: "only .mise.toml",
			files: map[string]string{
				".mise.toml": "[tasks.lint]\nrun = \"golangci-lint run\"\n",
			},
			want: []Task{
				{Source: ".mise.toml", Name: "lint", Desc: "golangci-lint run", Args: []string{"mise", "run", "lint"}},
			},
		},
		{
			name: "mise.toml wins over .mise.toml",
			files: map[string]string{
				"mise.toml":  "[tasks.from-toml]\nrun = \"echo toml\"\n",
				".mise.toml": "[tasks.from-dot-toml]\nrun = \"echo dot toml\"\n",
			},
			want: []Task{
				{Source: "mise.toml", Name: "from-toml", Desc: "echo toml", Args: []string{"mise", "run", "from-toml"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(miseSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "mise.toml") {
					t.Errorf("error = %q, want it to mention mise.toml", err)
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

func TestDiscoverMiseTasksMissingFile(t *testing.T) {
	got, err := discoverSource(miseSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

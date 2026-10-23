package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverDevboxTasks(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    []Task
		wantErr bool
	}{
		{
			name: "string and array scripts sorted, init_hook ignored",
			file: `{
				"packages": ["go@1.23"],
				"shell": {
					"init_hook": "echo hello",
					"scripts": {
						"test": "go test ./...",
						"build": ["go build ./...", "echo built"]
					}
				}
			}`,
			want: []Task{
				{Source: "devbox.json", Name: "build", Desc: "go build ./... && echo built", Args: []string{"devbox", "run", "build"}},
				{Source: "devbox.json", Name: "test", Desc: "go test ./...", Args: []string{"devbox", "run", "test"}},
			},
		},
		{
			name: "comments are allowed",
			file: `{
				// the packages to install
				"packages": [],
				"shell": {
					/* the runnable scripts */
					"init_hook": ["echo hello"],
					"scripts": {
						"lint": "golangci-lint run" // inline comment
					}
				}
			}`,
			want: []Task{
				{Source: "devbox.json", Name: "lint", Desc: "golangci-lint run", Args: []string{"devbox", "run", "lint"}},
			},
		},
		{
			name: "scripts without usable commands are skipped",
			file: `{"shell": {"scripts": {
				"good": "echo good",
				"empty": "",
				"empty-list": [],
				"number": 42,
				"object": {"cmd": "echo hi"},
				"mixed": ["echo hi", 7]
			}}}`,
			want: []Task{
				{Source: "devbox.json", Name: "good", Desc: "echo good", Args: []string{"devbox", "run", "good"}},
			},
		},
		{
			name:    "malformed json",
			file:    `{"shell": {"scripts": `,
			wantErr: true,
		},
		{
			name: "empty file",
			file: "",
			want: nil,
		},
		{
			name: "no shell.scripts object",
			file: `{"packages": ["hello"]}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "devbox.json", tt.file)

			got, err := discoverSource(osFiles{}, devboxSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), "devbox.json") {
					t.Errorf("error = %q, want it to mention devbox.json", err)
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

func TestDiscoverDevboxTasksMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, devboxSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverCargoAliases(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Task
		wantErr bool
	}{
		{
			name: "aliases sorted and chains resolved",
			files: map[string]string{
				".cargo/config.toml": `
[alias]
xtask = "run --package xtask --"
x = "xtask"
q = "x run --quick-run"
lint = "clippy --all-targets -- --cap-lints warn"
tq = "test -- -q"
qt = "tq"
`,
			},
			want: []Task{
				{Source: ".cargo/config.toml", Name: "lint", Desc: "clippy --all-targets -- --cap-lints warn", Args: []string{"cargo", "lint"}},
				{Source: ".cargo/config.toml", Name: "q", Desc: "run --package xtask -- run --quick-run", Args: []string{"cargo", "q"}},
				{Source: ".cargo/config.toml", Name: "qt", Desc: "test -- -q", Args: []string{"cargo", "qt"}},
				{Source: ".cargo/config.toml", Name: "tq", Desc: "test -- -q", Args: []string{"cargo", "tq"}},
				{Source: ".cargo/config.toml", Name: "x", Desc: "run --package xtask --", Args: []string{"cargo", "x"}},
				{Source: ".cargo/config.toml", Name: "xtask", Desc: "run --package xtask --", Args: []string{"cargo", "xtask"}},
			},
		},
		{
			name: "shell aliases are shown verbatim",
			files: map[string]string{
				".cargo/config.toml": "[alias]\ndeploy = \"!git push && echo done\"\n",
			},
			want: []Task{
				{Source: ".cargo/config.toml", Name: "deploy", Desc: "!git push && echo done", Args: []string{"cargo", "deploy"}},
			},
		},
		{
			name: "non-string and empty aliases are skipped",
			files: map[string]string{
				".cargo/config.toml": "[alias]\ngood = \"check\"\nnumber = 42\nflag = true\nempty = \"\"\n",
			},
			want: []Task{
				{Source: ".cargo/config.toml", Name: "good", Desc: "check", Args: []string{"cargo", "good"}},
			},
		},
		{
			name: "an alias cycle ends instead of spinning",
			files: map[string]string{
				".cargo/config.toml": "[alias]\na = \"b\"\nb = \"a\"\n",
			},
			want: []Task{
				{Source: ".cargo/config.toml", Name: "a", Desc: "a", Args: []string{"cargo", "a"}},
				{Source: ".cargo/config.toml", Name: "b", Desc: "b", Args: []string{"cargo", "b"}},
			},
		},
		{
			name:  "legacy .cargo/config",
			files: map[string]string{".cargo/config": "[alias]\nlint = \"clippy\"\n"},
			want: []Task{
				{Source: ".cargo/config", Name: "lint", Desc: "clippy", Args: []string{"cargo", "lint"}},
			},
		},
		{
			name: "legacy .cargo/config wins over the toml file, like cargo",
			files: map[string]string{
				".cargo/config":      "[alias]\nfrom-legacy = \"check\"\n",
				".cargo/config.toml": "[alias]\nfrom-toml = \"build\"\n",
			},
			want: []Task{
				{Source: ".cargo/config", Name: "from-legacy", Desc: "check", Args: []string{"cargo", "from-legacy"}},
			},
		},
		{
			name:    "malformed toml",
			files:   map[string]string{".cargo/config.toml": "[alias\n"},
			wantErr: true,
		},
		{
			name:  "empty file",
			files: map[string]string{".cargo/config.toml": ""},
			want:  nil,
		},
		{
			name:  "no alias table",
			files: map[string]string{".cargo/config.toml": "[build]\ntarget-dir = \"target\"\n"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}

			got, err := discoverSource(osFiles{}, cargoAliasSource(), dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("discoverSource() succeeded, want error")
				}
				if !strings.Contains(err.Error(), ".cargo/config.toml") {
					t.Errorf("error = %q, want it to mention .cargo/config.toml", err)
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

func TestDiscoverCargoAliasesMissingFile(t *testing.T) {
	got, err := discoverSource(osFiles{}, cargoAliasSource(), t.TempDir())
	if err != nil || got != nil {
		t.Errorf("discoverSource() on an empty dir = %v, %v; want nil, nil", got, err)
	}
}

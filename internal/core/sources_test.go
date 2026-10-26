package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// upwardSearchDir is the in-repo fixture for the parent-directory search: the
// task files sit in its root and a/b is the subdirectory the tests start from.
const upwardSearchDir = "testdata/upward-search"

// TestDiscoverSourceSearchesParentsJust: from a subdirectory without a local
// justfile, the ancestor's file is found and badged with its relative path.
func TestDiscoverSourceSearchesParentsJust(t *testing.T) {
	dir := filepath.Join(upwardSearchDir, "a", "b")

	tasks, err := discoverSource(osFiles{}, justfileSource(osFiles{}), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../../justfile", Name: "build", Desc: "Build the project", Args: []string{"just", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverSourceSearchesParentsTask: the Taskfile source searches parents
// too, with the same relative badge.
func TestDiscoverSourceSearchesParentsTask(t *testing.T) {
	dir := filepath.Join(upwardSearchDir, "a", "b")

	tasks, err := discoverSource(osFiles{}, taskfileSource(osFiles{}), dir)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../../Taskfile.yml", Name: "build", Desc: "Build the project", Args: []string{"task", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverSourceDoesNotSearchParentsForPackageJSON: the package.json
// source is directory-local, so an ancestor's file is not discovered.
func TestDiscoverSourceDoesNotSearchParentsForPackageJSON(t *testing.T) {
	dir := filepath.Join(upwardSearchDir, "a", "b")

	got, err := discoverSource(osFiles{}, packageJSONSource("", osFiles{}), dir)
	if err != nil || got != nil {
		t.Errorf("discoverSource() from a subdirectory = %v, %v; want nil, nil", got, err)
	}
}

// TestRegistryInvariants pins the metadata every built-in source carries:
// labels, display files, candidate order, parse functions and parent search.
// It also runs the checks against broken registries, so the failure messages
// are part of the test.
func TestRegistryInvariants(t *testing.T) {
	if errs := checkRegistry(defaultSources(Settings{}, osFiles{})); len(errs) != 0 {
		t.Fatalf("defaultSources() invariants = %v, want none", errs)
	}

	tests := []struct {
		name   string
		mutate func([]source)
		want   string
	}{
		{
			name:   "duplicate label",
			mutate: func(srcs []source) { srcs[1].Label = srcs[0].Label },
			want:   "duplicate label",
		},
		{
			name:   "display override outside files",
			mutate: func(srcs []source) { srcs[0].DisplayFileOverride = "nope" },
			want:   "DisplayFileOverride",
		},
		{
			name:   "wrong first file",
			mutate: func(srcs []source) { srcs[2].Files[0] = "nope" },
			want:   "first file",
		},
		{
			name:   "shared file name",
			mutate: func(srcs []source) { srcs[1].Files = append(srcs[1].Files, srcs[0].Files[0]) },
			want:   "belongs to",
		},
		{
			name:   "nil parser",
			mutate: func(srcs []source) { srcs[0].Parse = nil },
			want:   "parser",
		},
		{
			name:   "wrong parent search",
			mutate: func(srcs []source) { srcs[0].SearchParents = true },
			want:   "SearchParents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcs := cloneSources(defaultSources(Settings{}, osFiles{}))
			tt.mutate(srcs)

			errs := checkRegistry(srcs)
			if len(errs) == 0 {
				t.Fatal("checkRegistry() = nil, want an error")
			}
			var got strings.Builder
			for _, err := range errs {
				got.WriteString(err.Error())
				got.WriteByte('\n')
			}
			if !strings.Contains(got.String(), tt.want) {
				t.Errorf("checkRegistry() = %q, want it to mention %q", got.String(), tt.want)
			}
		})
	}
}

// TestSourcesReturnsCopies: the read-only registry view must not expose the
// slices discovery uses.
func TestSourcesReturnsCopies(t *testing.T) {
	first := Sources()
	if len(first) == 0 {
		t.Fatal("Sources() = empty, want the built-in registry")
	}

	first[0].Label = "mutated"
	first[0].Files[0] = "mutated"

	second := Sources()
	if second[0].Label != "package.json" || second[0].Files[0] != "package.json" {
		t.Errorf("Sources() shares state with a previous call: %+v", second[0])
	}
}

// firstFiles is the candidate order each built-in source must keep. The keys
// are labels until plan 2 introduces the stable IDs.
var firstFiles = map[string]string{
	"package.json":  "package.json",
	"Makefile":      "GNUmakefile",
	"Taskfile.yml":  "Taskfile.yml",
	"justfile":      "justfile",
	"Makefile.toml": "Makefile.toml",
	"Cargo aliases": ".cargo/config",
	"devbox.json":   "devbox.json",
	"composer.json": "composer.json",
	"deno.json":     "deno.json",
	"mise.toml":     "mise.toml",
}

// parentSources are the only sources that walk up the directory tree.
var parentSources = map[string]bool{
	"justfile":     true,
	"Taskfile.yml": true,
}

// checkRegistry reports the metadata invariants a registry must satisfy.
func checkRegistry(srcs []source) []error {
	var errs []error
	labels := make(map[string]bool, len(srcs))
	files := make(map[string]string, len(srcs)*2)

	for i, src := range srcs {
		if src.Label == "" {
			errs = append(errs, fmt.Errorf("source %d: empty label", i))
			continue
		}
		if labels[src.Label] {
			errs = append(errs, fmt.Errorf("source %q: duplicate label", src.Label))
		}
		labels[src.Label] = true

		if len(src.Files) == 0 {
			errs = append(errs, fmt.Errorf("source %q: no candidate files", src.Label))
		} else if want, ok := firstFiles[src.Label]; !ok {
			errs = append(errs, fmt.Errorf("source %q: no expected first file", src.Label))
		} else if src.Files[0] != want {
			errs = append(errs, fmt.Errorf("source %q: first file %q, want %q", src.Label, src.Files[0], want))
		}

		if src.DisplayFileOverride != "" && !slices.Contains(src.Files, src.DisplayFileOverride) {
			errs = append(errs, fmt.Errorf("source %q: DisplayFileOverride %q is not a candidate", src.Label, src.DisplayFileOverride))
		}
		if display := src.DisplayFile(); display == "" || !slices.Contains(src.Files, display) {
			errs = append(errs, fmt.Errorf("source %q: display file %q is not a candidate", src.Label, display))
		}

		for _, name := range src.Files {
			if owner, ok := files[name]; ok && owner != src.Label {
				errs = append(errs, fmt.Errorf("file %q belongs to %q and %q", name, owner, src.Label))
			}
			files[name] = src.Label
		}

		if src.Parse == nil {
			errs = append(errs, fmt.Errorf("source %q: nil parser", src.Label))
		}
		if src.SearchParents != parentSources[src.Label] {
			errs = append(errs, fmt.Errorf("source %q: SearchParents = %v", src.Label, src.SearchParents))
		}
	}

	if len(srcs) != len(firstFiles) {
		errs = append(errs, fmt.Errorf("registry has %d sources, want %d", len(srcs), len(firstFiles)))
	}
	return errs
}

// cloneSources deep-copies the registry so a broken fixture cannot mutate the
// slices the real sources share.
func cloneSources(srcs []source) []source {
	out := make([]source, len(srcs))
	for i, src := range srcs {
		out[i] = src
		out[i].Files = slices.Clone(src.Files)
	}
	return out
}

// TestDiscoverSourceNearestDirectoryWins: the file in the nearest ancestor
// beats the one further up, and a file in the directory itself keeps the bare
// name as its badge.
func TestDiscoverSourceNearestDirectoryWins(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	grandchild := filepath.Join(child, "grandchild")
	writeFile(t, root, "justfile", "root-only:\n    echo root\n")
	writeFile(t, child, "justfile", "child-only:\n    echo child\n")
	if err := os.MkdirAll(grandchild, 0o750); err != nil {
		t.Fatal(err)
	}

	tasks, err := discoverSource(osFiles{}, justfileSource(osFiles{}), grandchild)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../justfile", Name: "child-only", Args: []string{"just", "child-only"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from the grandchild = %+v, want %+v", tasks, want)
	}

	tasks, err = discoverSource(osFiles{}, justfileSource(osFiles{}), child)
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want = []Task{{Source: "justfile", Name: "child-only", Args: []string{"just", "child-only"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() from the child = %+v, want %+v", tasks, want)
	}
}

// TestDiscoverSourceSearchesParentsFromDot: the picker starts from ".", and
// filepath.Dir(".") stays ".", so the walk must normalize the directory first.
func TestDiscoverSourceSearchesParentsFromDot(t *testing.T) {
	t.Chdir(filepath.Join(upwardSearchDir, "a", "b"))

	tasks, err := discoverSource(osFiles{}, justfileSource(osFiles{}), ".")
	if err != nil {
		t.Fatalf("discoverSource() error = %v", err)
	}
	want := []Task{{Source: "../../justfile", Name: "build", Desc: "Build the project", Args: []string{"just", "build"}}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("discoverSource() = %+v, want %+v", tasks, want)
	}
}

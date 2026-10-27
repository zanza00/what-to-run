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
// It builds the registry through missingFiles, proving metadata needs no disk,
// and runs the checks against broken fixtures, so the failure messages are
// part of the test.
func TestRegistryInvariants(t *testing.T) {
	if errs := checkRegistry(defaultSources(Settings{}, missingFiles{})); len(errs) != 0 {
		t.Fatalf("defaultSources() invariants = %v, want none", errs)
	}

	tests := []struct {
		name   string
		mutate func([]source) []source
		want   string
	}{
		{
			name:   "empty label",
			mutate: func(srcs []source) []source { srcs[0].Label = ""; return srcs },
			want:   "empty label",
		},
		{
			name:   "duplicate label",
			mutate: func(srcs []source) []source { srcs[1].Label = srcs[0].Label; return srcs },
			want:   "duplicate label",
		},
		{
			name:   "display override outside the files",
			mutate: func(srcs []source) []source { srcs[0].DisplayFileOverride = "nope"; return srcs },
			want:   "DisplayFileOverride",
		},
		{
			name:   "display override on a source that needs none",
			mutate: func(srcs []source) []source { srcs[0].DisplayFileOverride = srcs[0].Files[0]; return srcs },
			want:   "DisplayFileOverride",
		},
		{
			name:   "missing display override",
			mutate: func(srcs []source) []source { srcs[1].DisplayFileOverride = ""; return srcs },
			want:   "display file",
		},
		{
			name:   "resolved display file outside the files",
			mutate: func(srcs []source) []source { srcs[0].Files = nil; return srcs },
			want:   "display file",
		},
		{
			name:   "wrong first file",
			mutate: func(srcs []source) []source { srcs[2].Files[0] = "nope"; return srcs },
			want:   "first file",
		},
		{
			name:   "file name shared by two sources",
			mutate: func(srcs []source) []source { srcs[1].Files = append(srcs[1].Files, srcs[0].Files[0]); return srcs },
			want:   "belongs to",
		},
		{
			name:   "wrong parent search",
			mutate: func(srcs []source) []source { srcs[0].SearchParents = true; return srcs },
			want:   "SearchParents",
		},
		{
			name:   "missing parser",
			mutate: func(srcs []source) []source { srcs[0].Parse = nil; return srcs },
			want:   "nil parser",
		},
		{
			name: "reordered sources",
			mutate: func(srcs []source) []source {
				srcs[0], srcs[1] = srcs[1], srcs[0]
				return srcs
			},
			want: "registry order",
		},
		{
			name:   "missing source",
			mutate: func(srcs []source) []source { return srcs[1:] },
			want:   "9 sources, want 10",
		},
		{
			name:   "extra source",
			mutate: func(srcs []source) []source { return append(srcs, srcs[0]) },
			want:   "11 sources, want 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := checkRegistry(tt.mutate(cloneSources(defaultSources(Settings{}, missingFiles{}))))
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

// TestSourcesMatchesRegistry: the read-only view carries the registry metadata
// in the display order pinned by registryOrder.
func TestSourcesMatchesRegistry(t *testing.T) {
	infos := Sources()
	if len(infos) != len(registryOrder) {
		t.Fatalf("Sources() has %d entries, want %d", len(infos), len(registryOrder))
	}
	for i, info := range infos {
		want := registryOrder[i]
		if info.Label != want.label {
			t.Errorf("Sources()[%d].Label = %q, want %q", i, info.Label, want.label)
		}
		if len(info.Files) == 0 || info.Files[0] != want.firstFile {
			t.Errorf("Sources()[%d].Files = %v, want %q first", i, info.Files, want.firstFile)
		}
		if info.DisplayFileOverride != want.displayOverride {
			t.Errorf(
				"Sources()[%d].DisplayFileOverride = %q, want %q",
				i, info.DisplayFileOverride, want.displayOverride,
			)
		}
		if display := info.DisplayFile(); display != want.displayFile {
			t.Errorf("Sources()[%d].DisplayFile() = %q, want %q", i, display, want.displayFile)
		}
		if info.SearchParents != want.searchParents {
			t.Errorf("Sources()[%d].SearchParents = %v, want %v", i, info.SearchParents, want.searchParents)
		}
	}
}

// TestSourcesReturnsCopies: changing the read-only view, file lists included,
// must not affect a later call.
func TestSourcesReturnsCopies(t *testing.T) {
	first := Sources()
	if len(first) == 0 {
		t.Fatal("Sources() = empty, want the built-in registry")
	}
	want := make([]SourceInfo, len(first))
	for i, info := range first {
		info.Files = slices.Clone(info.Files)
		want[i] = info
	}

	for i := range first {
		first[i].Label = "mutated"
		first[i].DisplayFileOverride = "mutated"
		first[i].SearchParents = !first[i].SearchParents
		for j := range first[i].Files {
			first[i].Files[j] = "mutated"
		}
		first[i].Files = append(first[i].Files, "mutated")
	}

	if got := Sources(); !reflect.DeepEqual(got, want) {
		t.Errorf("Sources() shares state with a previous call: got %+v, want %+v", got, want)
	}
}

// registryOrder is the display order of the built-in sources: each entry has
// the label, the first candidate file the source must prefer, the display
// override it must declare, the file the empty screen must show and whether it
// searches parent directories. Until plan 2 gives the sources stable IDs, the
// label identifies the entry. The table pins the order itself, so a reordered,
// missing or extra source fails checkRegistry.
var registryOrder = []struct {
	label           string
	firstFile       string
	displayOverride string
	displayFile     string
	searchParents   bool
}{
	{label: "package.json", firstFile: "package.json", displayFile: "package.json"},
	{label: "Makefile", firstFile: "GNUmakefile", displayOverride: "Makefile", displayFile: "Makefile"},
	{label: "Taskfile.yml", firstFile: "Taskfile.yml", displayFile: "Taskfile.yml", searchParents: true},
	{label: "justfile", firstFile: "justfile", displayFile: "justfile", searchParents: true},
	{label: "Makefile.toml", firstFile: "Makefile.toml", displayFile: "Makefile.toml"},
	{
		label:           "Cargo aliases",
		firstFile:       ".cargo/config",
		displayOverride: ".cargo/config.toml",
		displayFile:     ".cargo/config.toml",
	},
	{label: "devbox.json", firstFile: "devbox.json", displayFile: "devbox.json"},
	{label: "composer.json", firstFile: "composer.json", displayFile: "composer.json"},
	{label: "deno.json", firstFile: "deno.json", displayFile: "deno.json"},
	{label: "mise.toml", firstFile: "mise.toml", displayFile: "mise.toml"},
}

// checkRegistry reports the invariants a built-in registry must satisfy: the
// order, labels, first files, display files and parent search in registryOrder;
// unique labels and file names; and a parser.
func checkRegistry(srcs []source) []error {
	var errs []error
	if len(srcs) != len(registryOrder) {
		errs = append(errs, fmt.Errorf("registry has %d sources, want %d", len(srcs), len(registryOrder)))
	}

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

		switch {
		case i >= len(registryOrder):
			errs = append(errs, fmt.Errorf("source %q: extra source after the %d expected", src.Label, len(registryOrder)))
		case src.Label != registryOrder[i].label:
			errs = append(errs, fmt.Errorf(
				"source %d: label %q, want %q in registry order",
				i, src.Label, registryOrder[i].label,
			))
		default:
			want := registryOrder[i]
			if len(src.Files) == 0 {
				errs = append(errs, fmt.Errorf("source %q: no candidate files", src.Label))
			} else if src.Files[0] != want.firstFile {
				errs = append(errs, fmt.Errorf("source %q: first file %q, want %q", src.Label, src.Files[0], want.firstFile))
			}
			if src.DisplayFileOverride != want.displayOverride {
				errs = append(errs, fmt.Errorf(
					"source %q: DisplayFileOverride = %q, want %q",
					src.Label, src.DisplayFileOverride, want.displayOverride,
				))
			}
			if display := src.DisplayFile(); display != want.displayFile {
				errs = append(errs, fmt.Errorf(
					"source %q: display file = %q, want %q",
					src.Label, display, want.displayFile,
				))
			}
			if src.SearchParents != want.searchParents {
				errs = append(errs, fmt.Errorf(
					"source %q: SearchParents = %v, want %v",
					src.Label, src.SearchParents, want.searchParents,
				))
			}
		}

		if src.DisplayFileOverride != "" && !slices.Contains(src.Files, src.DisplayFileOverride) {
			errs = append(errs, fmt.Errorf(
				"source %q: DisplayFileOverride %q is not a candidate",
				src.Label, src.DisplayFileOverride,
			))
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

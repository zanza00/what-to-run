package core

import (
	"os"
	"path/filepath"
	"reflect"
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

// TestSourceSearchParents pins the flag per built-in source: only the justfile
// and Taskfile sources walk up, since only just and task do.
func TestSourceSearchParents(t *testing.T) {
	tests := []struct {
		firstFile     string
		searchParents bool
	}{
		{"package.json", false},
		{"GNUmakefile", false},
		{"Taskfile.yml", true},
		{"justfile", true},
		{"Makefile.toml", false},
		{".cargo/config", false},
		{"devbox.json", false},
		{"composer.json", false},
		{"deno.json", false},
		{"mise.toml", false},
	}

	sources := defaultSources(Settings{}, osFiles{})
	if len(sources) != len(tests) {
		t.Fatalf("defaultSources() has %d sources, want %d", len(sources), len(tests))
	}
	for i, tt := range tests {
		src := sources[i]
		if len(src.Files) == 0 || src.Files[0] != tt.firstFile {
			t.Errorf("source %d = %v, want %q first", i, src.Files, tt.firstFile)
			continue
		}
		if src.SearchParents != tt.searchParents {
			t.Errorf("%s SearchParents = %v, want %v", tt.firstFile, src.SearchParents, tt.searchParents)
		}
	}
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

package core

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// TestParentLevelsBuild pins the enumeration of the ancestors: the level
// directories of the walk, the badge prefix of each level, and the end at the
// root. The table holds a fixed tree of the POSIX root; on Windows the build
// resolves its paths against the drive of the cwd, so the skip keeps the pin
// on the platforms the suite runs on.
func TestParentLevelsBuild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("filepath.Abs resolves a drive-less path against the drive of the cwd")
	}
	tests := []struct {
		name string
		dir  string
		want []parentLevel
	}{
		{
			name: "two ancestors below the root",
			dir:  "/proj/sub",
			want: []parentLevel{{dir: "/proj", prefix: ".."}, {dir: "/", prefix: "../.."}},
		},
		{
			name: "three ancestors below the root",
			dir:  "/a/b/c",
			want: []parentLevel{
				{dir: "/a/b", prefix: ".."},
				{dir: "/a", prefix: "../.."},
				{dir: "/", prefix: "../../.."},
			},
		},
		{
			name: "one level up lands on the root",
			dir:  "/proj",
			want: []parentLevel{{dir: "/", prefix: ".."}},
		},
		{
			name: "the root walks itself with a dot prefix",
			dir:  "/",
			want: []parentLevel{{dir: "/", prefix: "."}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levels, err := newParentLevels(tt.dir).build()
			if err != nil {
				t.Fatalf("build() error = %v, want none", err)
			}
			if !reflect.DeepEqual(levels, tt.want) {
				t.Errorf("build() = %+v, want %+v", levels, tt.want)
			}
		})
	}
}

// TestParentLevelsBuildRemembersTheList pins the memory of the enumeration:
// the second build hands out the remembered list and does not enumerate
// again. The remembered list itself comes back, so a second enumeration,
// which would build a new one, reddens here.
func TestParentLevelsBuildRemembersTheList(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("filepath.Abs resolves a drive-less path against the drive of the cwd")
	}
	parents := newParentLevels("/proj/sub")
	first, firstErr := parents.build()
	if firstErr != nil {
		t.Fatalf("build() error = %v, want none", firstErr)
	}
	second, secondErr := parents.build()
	if secondErr != nil {
		t.Fatalf("the second build() error = %v, want none", secondErr)
	}
	if !reflect.DeepEqual(second, first) {
		t.Errorf("the second build() = %+v, want the remembered %+v", second, first)
	}
	if len(first) == 0 || &second[0] != &first[0] {
		t.Error("the second build() enumerated a new list, want the remembered one")
	}
}

// TestDiscoverProjectFindNeverWalks pins the laziness of the parent levels:
// the enumeration starts at the first source that needs the parent search.
// A source that finds its file in the project directory never touches the
// walk, so when every parent-searching source finds its file there, the port
// sees no absolute path: the walk of the levels starts at the absolute base.
func TestDiscoverProjectFindNeverWalks(t *testing.T) {
	files := newCountFiles(mapFiles{
		"pyproject.toml": pyprojectAllTools,
		"justfile":       "no-walk:\n\techo stayed\n",
		"Taskfile.yml":   "tasks:\n  build:\n    desc: Build the app\n",
		"Pipfile":        "[scripts]\ncheck = \"ruff check .\"\n",
	})

	tasks, warns := Discover(files, ".", Settings{})
	if len(warns) != 0 {
		t.Fatalf("Discover() warnings = %v, want none", warns)
	}
	if len(tasks) == 0 {
		t.Fatal("Discover() found no tasks, although every parent-searching source lists a file")
	}
	for path := range files.reads {
		if filepath.IsAbs(path) {
			t.Errorf("the port read the absolute path %s, so a source left the project directory instead of stopping at its found file", path)
		}
	}
}

// TestDiscoverUnresolvableDirWarnsPerSource pins the replay of the failure of
// the enumeration. The pin destroys the working directory of the process, so
// the project directory "." has no absolute path. Every parent-searching
// source that missed in the project directory replays the remembered failure
// as its own warning: keyed by its source ID, naming the project directory,
// all eight with the same message and in registry order, byte-shaped exactly
// as the per-source walk reported them. A platform that refuses to remove
// the live working directory skips the pin, because the failure of
// filepath.Abs does not occur there.
func TestDiscoverUnresolvableDirWarnsPerSource(t *testing.T) {
	gone := t.TempDir()
	t.Chdir(gone)
	if err := os.Remove(gone); err != nil {
		t.Skipf("cannot remove the working directory: %v", err)
	}

	tasks, warns := Discover(mapFiles{}, ".", Settings{})
	if len(tasks) != 0 {
		t.Errorf("Discover() tasks = %v, want none", tasks)
	}
	want := []string{"task", "just", "poe", "pdm", "taskipy", "hatch", "tox", "pipenv"}
	if len(warns) != len(want) {
		t.Fatalf("Discover() gave %d warnings, want %d: %v", len(warns), len(want), warns)
	}
	for i, id := range want {
		if kind := warns[i].Kind(); kind != KindSource {
			t.Errorf("warning %d kind = %s, want %s", i, kind, KindSource)
		}
		if key := warns[i].Key(); key != "source:"+id+":." {
			t.Errorf("warning %d key = %q, want %q", i, key, "source:"+id+":.")
		}
		if !strings.HasPrefix(warns[i].Error(), ".: ") {
			t.Errorf("warning %d message = %q, want it to name the project directory", i, warns[i].Error())
		}
	}
	for i := 1; i < len(warns); i++ {
		if warns[i].Error() != warns[0].Error() {
			t.Errorf("warning %d message = %q, want the replay of %q", i, warns[i].Error(), warns[0].Error())
		}
	}
}

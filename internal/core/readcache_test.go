package core

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// pyprojectAllTools is a pyproject.toml that every Python source parses
// without a warning: poe, pdm, taskipy, hatch and tox each find their own
// table in it. The five sources list the same file, so it is the shared file
// that the read cache law has to hold.
const pyprojectAllTools = `[tool.poe.tasks.go]
cmd = "go test ./..."

[tool.taskipy.tasks]
test = "pytest"

[tool.hatch.envs.default.scripts]
lint = "ruff check ."
`

// TestDiscoverReadsSharedPyprojectOnce pins the read law of the cache in the
// project directory: one Discover reads one file at most once, so the
// pyproject.toml that the five Python sources list reaches the port once,
// whatever the number of sources that list it.
func TestDiscoverReadsSharedPyprojectOnce(t *testing.T) {
	files := newCountFiles(mapFiles{"pyproject.toml": pyprojectAllTools})

	tasks, warns := Discover(files, ".", Settings{})
	if len(warns) != 0 {
		t.Fatalf("Discover() warnings = %v, want none", warns)
	}
	names := taskNames(tasks)
	for _, want := range []string{"go", "test", "default:lint"} {
		if !slices.Contains(names, want) {
			t.Errorf("Discover() tasks = %v, want the %q task", names, want)
		}
	}
	if got := files.reads["pyproject.toml"]; got != 1 {
		t.Errorf("pyproject.toml reached the port %d times, want once", got)
	}
}

// TestDiscoverReadsMissingAncestorOnce pins the read law of the cache in the
// parent search: every parent-searching source misses through the same
// levels, so a missing candidate of an ancestor level reaches the port
// once, however many sources list it. The asserted candidates cover the
// eight walkers: the pyproject.toml of poe, pdm, taskipy, hatch and tox, and
// the first candidate of the just, task and pipenv sources. The project
// directory has the parent /proj and the root / as its ancestors.
func TestDiscoverReadsMissingAncestorOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("filepath.Abs resolves a drive-less path against the drive of the cwd")
	}
	base := filepath.Join(string(filepath.Separator), "proj", "sub")
	parent := filepath.Dir(base)
	root := filepath.Dir(parent)
	files := newCountFiles(mapFiles{})

	tasks, warns := Discover(files, base, Settings{})
	if len(tasks) != 0 || len(warns) != 0 {
		t.Fatalf("Discover() = %v, %v, want nothing", tasks, warns)
	}
	for _, name := range []string{"pyproject.toml", "justfile", "Taskfile.yml", "Pipfile"} {
		for _, level := range []string{base, parent, root} {
			path := filepath.Join(level, name)
			if got := files.reads[path]; got != 1 {
				t.Errorf("%s reached the port %d times, want once", path, got)
			}
		}
	}
}

// TestDiscoverTaskfileIncludeThroughCache pins the parsers' side of the whole
// wrap: the parser keeps its captured port, so the include that a Taskfile
// pulls in stays readable through the read cache and reaches the port once.
func TestDiscoverTaskfileIncludeThroughCache(t *testing.T) {
	files := newCountFiles(mapFiles{
		"Taskfile.yml":            "includes:\n  shared: .taskfiles/Taskfile.yml\n\ntasks:\n  build:\n    desc: Build the app\n",
		".taskfiles/Taskfile.yml": "tasks:\n  asset:\n    desc: Build the assets\n",
	})

	tasks, warns := Discover(files, ".", Settings{})
	if len(warns) != 0 {
		t.Fatalf("Discover() warnings = %v, want none", warns)
	}
	if got := taskNames(tasks); !slices.Equal(got, []string{"build", "shared:asset"}) {
		t.Errorf("Discover() task names = %v, want [build shared:asset]", got)
	}
	if got := files.reads[".taskfiles/Taskfile.yml"]; got != 1 {
		t.Errorf("the include reached the port %d times, want once", got)
	}
	if got := files.reads["Taskfile.yml"]; got != 1 {
		t.Errorf("Taskfile.yml reached the port %d times, want once", got)
	}
}

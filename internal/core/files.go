package core

// DirEntry is one entry of a directory listing: the name inside the listed
// directory and whether the entry is itself a directory. It is the port-side
// subset of os.DirEntry that module resolution needs.
type DirEntry interface {
	Name() string
	IsDir() bool
}

// Files is the filesystem port that discovery and the rc loader use. Paths are
// OS paths (not fs.FS names), because the rc file lives outside the project
// directory. The production adapter is internal/tui/osFiles, and tests can
// provide an in-memory implementation.
type Files interface {
	ReadFile(name string) ([]byte, error)
	Exists(name string) bool
	// ReadDir lists a directory, sorted by name. Module resolution needs the
	// sort so that it can accept justfile and .justfile in any capitalization,
	// as just does.
	ReadDir(name string) ([]DirEntry, error)
}

package core

// Files is the filesystem port used by discovery and by the rc loader.
// Paths are OS paths (not fs.FS names) because the rc file lives outside
// the project directory. The production adapter is internal/tui/osFiles;
// tests can provide an in-memory implementation.
type Files interface {
	ReadFile(name string) ([]byte, error)
	Exists(name string) bool
}

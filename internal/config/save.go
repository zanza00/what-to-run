package config

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// Store is the writable filesystem that the rc saver uses. The picker passes
// its production adapter, and tests pass a temporary or an in-memory one.
type Store interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(name string, perm fs.FileMode) error
	Remove(name string) error
}

// SaveTheme writes a theme into the user rc file, the only place that holds it
// (like OpenCode's cli.json). The function creates the file and its directory
// when they are missing, and it preserves the rest of the content. It returns
// the path that it wrote.
//
// The caller resolves userFile and provides the store, so the function never
// reads the process environment or the current directory.
func SaveTheme(store Store, userFile, name string) (string, error) {
	if userFile == "" {
		return "", errors.New("cannot resolve the user config directory")
	}

	data, err := store.ReadFile(userFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := store.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		return "", err
	}
	if err := store.WriteFile(userFile, setThemeKey(data, name), 0o600); err != nil {
		return "", err
	}
	return userFile, nil
}

// setThemeKey returns the rc file content with the top-level theme key set to
// name. An existing assignment is replaced, and a missing one is added at the
// top of the file, because after a table header the key would belong to that
// table.
func setThemeKey(src []byte, name string) []byte {
	assignment := "theme = " + strconv.Quote(name)
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break // A table starts here, so the key must go above it.
		}
		if key, _, ok := strings.Cut(trimmed, "="); ok && strings.TrimSpace(key) == "theme" {
			lines[i] = assignment
			return []byte(strings.Join(lines, "\n"))
		}
	}
	if len(bytes.TrimSpace(src)) == 0 {
		return []byte(assignment + "\n")
	}
	return []byte(assignment + "\n\n" + string(src))
}

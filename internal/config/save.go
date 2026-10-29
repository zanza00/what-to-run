package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SaveTheme writes a theme into the user rc file, the only place that holds it
// (like OpenCode's cli.json). The function creates the file and its directory
// when they are missing, and it preserves the rest of the content. It returns
// the path that it wrote.
func SaveTheme(name string) (string, error) {
	_, userFile := DefaultFiles(".")
	if userFile == "" {
		return "", errors.New("cannot resolve the user config directory")
	}

	data, err := os.ReadFile(userFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(userFile, setThemeKey(data, name), 0o600); err != nil {
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

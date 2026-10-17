// The package.json source: npm/pnpm/yarn/bun scripts.
package tui

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// packageJSONSource returns the source for the project's package.json.
func packageJSONSource() source {
	return source{
		Files: []string{"package.json"},
		Parse: parsePackageScripts,
	}
}

// parsePackageScripts returns the scripts declared in package.json, sorted
// by name, each run through the package manager in use.
func parsePackageScripts(dir string, data []byte) ([]Task, error) {
	var pkg struct {
		Scripts        map[string]string `json:"scripts"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	runner := packageRunner(dir, pkg.PackageManager)
	tasks := make([]Task, 0, len(pkg.Scripts))
	for _, name := range slices.Sorted(maps.Keys(pkg.Scripts)) {
		tasks = append(tasks, Task{
			Name: name,
			Desc: pkg.Scripts[name],
			Args: append(slices.Clone(runner), name),
		})
	}
	return tasks, nil
}

// packageRunner returns the argv prefix for running a package.json script:
// the declared packageManager field wins, then the lock file in use, then npm.
func packageRunner(dir, declared string) []string {
	if i := strings.IndexByte(declared, '@'); i > 0 {
		switch declared[:i] {
		case "bun", "pnpm", "yarn", "npm":
			return []string{declared[:i], "run"}
		}
	}
	for _, c := range []struct{ lock, runner string }{
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	} {
		if _, err := os.Stat(filepath.Join(dir, c.lock)); err == nil {
			return []string{c.runner, "run"}
		}
	}
	return []string{"npm", "run"}
}

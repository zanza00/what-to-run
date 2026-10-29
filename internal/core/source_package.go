// The package.json source: npm/pnpm/yarn/bun scripts.
package core

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// packageJSONSource returns the source for the package.json of the project.
// runnerOverride is the package_manager setting of the rc file, empty when it
// is unset.
func packageJSONSource(runnerOverride string, files Files) source {
	return source{
		SourceInfo: SourceInfo{
			ID:    "npm",
			Label: "package.json",
			Files: []string{"package.json"},
		},
		Parse: func(dir string, data []byte) ([]Task, error) {
			return parsePackageScripts(files, dir, data, runnerOverride)
		},
	}
}

// parsePackageScripts returns the scripts that package.json declares, sorted by
// name. Each script runs through the package manager in use.
func parsePackageScripts(files Files, dir string, data []byte, runnerOverride string) ([]Task, error) {
	var pkg struct {
		Scripts        map[string]string `json:"scripts"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	runner := packageRunner(files, dir, pkg.PackageManager, runnerOverride)
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

// PackageManagerName returns the bare name of a package manager declaration
// ("bun@1.2.3" or just "bun") when it names one of the supported managers.
func PackageManagerName(declared string) (string, bool) {
	name := declared
	if i := strings.IndexByte(declared, '@'); i > 0 {
		name = declared[:i]
	}
	switch name {
	case "bun", "pnpm", "yarn", "npm":
		return name, true
	}
	return "", false
}

// packageRunner returns the argv prefix that runs a package.json script. The
// precedence is the package_manager of the rc file, then the declared
// packageManager field, then the lock file in use, then npm.
func packageRunner(files Files, dir, declared, override string) []string {
	for _, name := range []string{override, declared} {
		if pm, ok := PackageManagerName(name); ok {
			return []string{pm, "run"}
		}
	}
	for _, c := range []struct{ lock, runner string }{
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	} {
		if files.Exists(filepath.Join(dir, c.lock)) {
			return []string{c.runner, "run"}
		}
	}
	return []string{"npm", "run"}
}

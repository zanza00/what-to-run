package tui

import (
	"os"
	"path/filepath"

	"github.com/zanza00/what-to-run/internal/config"
	"github.com/zanza00/what-to-run/internal/core"
)

// Env is the environment of one picker run: the paths and the filesystems that
// the picker reads and writes. Run and Preview build the production one at the
// edge, and tests build an explicit one. The model and its handlers never read
// the process environment or the current directory.
type Env struct {
	// Files is the read port for the task sources and the rc loader.
	Files core.Files

	// Store is the writable port for the rc file and the dismissed warnings.
	Store config.Store

	// Dir is the project directory that discovery reads. It is absolute in
	// production, so a change of the process directory cannot move it.
	Dir string

	// ProjectFile and UserFile are the project and user rc files.
	ProjectFile, UserFile string

	// ProjectLabel names the project rc file in warnings. The production env
	// keeps the historical ".what-to-run.toml" text. An empty label shortens
	// ProjectFile with Home.
	ProjectLabel string

	// CacheFile is the dismissed-warnings file.
	CacheFile string

	// Home shortens paths in messages and expands "~" in project keys.
	Home string

	// Dismissed is the dismissed-warnings set. The edge loads it before it
	// builds the model.
	Dismissed map[string]bool
}

// discover gathers the tasks of the environment directory and the rc settings.
// It reads the rc file first, and its problems become warnings, then it reads
// every source. The warnings are returned even when tasks were found, because
// the picker shows them in the status line.
func (e Env) discover() ([]core.Task, config.Config, []error) {
	cfg, warnings := config.Load(e.Files, config.Paths{
		Project:      e.ProjectFile,
		Global:       e.UserFile,
		ProjectLabel: e.ProjectLabel,
		Home:         e.Home,
	}, themeNames())
	tasks, sourceWarnings := core.Discover(e.Files, e.Dir, cfg.Settings)
	return tasks, cfg, append(warnings, sourceWarnings...)
}

// resolveEnv builds the production environment when the caller did not set
// one. It stores the result on the options, so the model receives the same
// environment. The edge calls it once per run.
func (o *Options) resolveEnv() *Env {
	if o.Env == nil {
		o.Env = productionEnv()
	}
	return o.Env
}

// productionEnv builds the environment of a run from the process. It is the
// only place that resolves the current directory, the home directory and the
// cache directory.
func productionEnv() *Env {
	dir, err := os.Getwd()
	if err != nil {
		// Discovery with a relative directory keeps the old best-effort
		// behaviour when the process has no current directory.
		dir = "."
	}
	projectFile, userFile := config.DefaultFiles(dir)
	home, _ := os.UserHomeDir()

	cacheFile := ""
	if cacheDir, err := userCacheDir(); err == nil && cacheDir != "" {
		cacheFile = filepath.Join(cacheDir, "what-to-run", "dismissed-warnings")
	}

	files := osFiles{}
	return &Env{
		Files:        files,
		Store:        files,
		Dir:          dir,
		ProjectFile:  projectFile,
		ProjectLabel: ".what-to-run.toml",
		UserFile:     userFile,
		CacheFile:    cacheFile,
		Home:         home,
		Dismissed:    loadDismissedWarnings(files, cacheFile),
	}
}

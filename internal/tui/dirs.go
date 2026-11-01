package tui

import (
	"os"
	"path/filepath"
)

// userCacheDir returns the root of the dismissed-warnings file. XDG_CACHE_HOME
// wins on every platform, and a relative value is ignored, as the XDG
// specification requires. It is the counterpart of the config userConfigDir.
func userCacheDir() (string, error) {
	if dir := os.Getenv("XDG_CACHE_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	return os.UserCacheDir()
}

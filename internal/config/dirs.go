package config

import (
	"os"
	"path/filepath"
)

// userConfigDir returns the root of the user rc file. XDG_CONFIG_HOME wins on
// every platform, so the documented $XDG_CONFIG_HOME/what-to-run/config.toml
// also applies on macOS, where os.UserConfigDir would otherwise return
// ~/Library/Application Support. A relative value is ignored, as the XDG
// specification requires.
func userConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	return os.UserConfigDir()
}

package tui

import (
	"path/filepath"
	"testing"
)

func TestUserDirsHonorAbsoluteXDG(t *testing.T) {
	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	if got, err := userConfigDir(); err != nil || got != configDir {
		t.Errorf("userConfigDir() = %q, %v; want %q", got, err, configDir)
	}
	if got, err := userCacheDir(); err != nil || got != cacheDir {
		t.Errorf("userCacheDir() = %q, %v; want %q", got, err, cacheDir)
	}
}

func TestUserDirsIgnoreRelativeXDG(t *testing.T) {
	// The XDG spec makes a relative path invalid, so the two functions must
	// ignore it and never resolve a config file inside the current directory.
	t.Setenv("XDG_CONFIG_HOME", "relative")
	t.Setenv("XDG_CACHE_HOME", "relative")
	t.Setenv("HOME", t.TempDir())

	for name, dir := range map[string]func() (string, error){
		"userConfigDir": userConfigDir,
		"userCacheDir":  userCacheDir,
	} {
		got, err := dir()
		if err != nil {
			continue // No native directory on this platform, so there is nothing to check.
		}
		if got == "relative" || !filepath.IsAbs(got) {
			t.Errorf("%s() = %q, want an absolute native directory", name, got)
		}
	}
}

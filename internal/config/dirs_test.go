package config

import (
	"path/filepath"
	"testing"
)

func TestUserConfigDirHonorsAbsoluteXDG(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	if got, err := userConfigDir(); err != nil || got != configDir {
		t.Errorf("userConfigDir() = %q, %v; want %q", got, err, configDir)
	}
}

func TestUserConfigDirIgnoresRelativeXDG(t *testing.T) {
	// The XDG spec makes a relative path invalid, so the function must ignore
	// it and never resolve a config file inside the current directory.
	t.Setenv("XDG_CONFIG_HOME", "relative")
	t.Setenv("HOME", t.TempDir())

	got, err := userConfigDir()
	if err != nil {
		return // No native directory on this platform, so there is nothing to check.
	}
	if got == "relative" || !filepath.IsAbs(got) {
		t.Errorf("userConfigDir() = %q, want an absolute native directory", got)
	}
}

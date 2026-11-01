package tui

import (
	"path/filepath"
	"testing"
)

func TestUserCacheDirHonorsAbsoluteXDG(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)

	if got, err := userCacheDir(); err != nil || got != cacheDir {
		t.Errorf("userCacheDir() = %q, %v; want %q", got, err, cacheDir)
	}
}

func TestUserCacheDirIgnoresRelativeXDG(t *testing.T) {
	// The XDG spec makes a relative path invalid, so the function must ignore
	// it and never resolve a cache file inside the current directory.
	t.Setenv("XDG_CACHE_HOME", "relative")
	t.Setenv("HOME", t.TempDir())

	got, err := userCacheDir()
	if err != nil {
		return // No native directory on this platform, so there is nothing to check.
	}
	if got == "relative" || !filepath.IsAbs(got) {
		t.Errorf("userCacheDir() = %q, want an absolute native directory", got)
	}
}

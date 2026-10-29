//go:build ignore

// Command install-dprint downloads the dprint release that is pinned below for
// the host platform into the given path. `task fmt-md` and `task check-md` run
// the command through `go run`, so contributors do not need a global dprint
// install, and every machine, CI included, uses the same version.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// dprintVersion is the release that `task fmt-md` downloads and that CI uses.
const dprintVersion = "0.64.3"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/install-dprint.go DEST")
		os.Exit(2)
	}
	dest := os.Args[1]
	if hasPinnedVersion(dest) {
		return
	}
	target, err := targetFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fail(err)
	}
	if err := download(target, dest); err != nil {
		fail(err)
	}
	fmt.Printf("Installed dprint %s to %s\n", dprintVersion, dest)
}

// hasPinnedVersion reports whether dest is a regular file that runs the release
// pinned in dprintVersion. The check compares the version and not the existence
// of the file, so bin/tools stays in sync when the pin moves. A stale copy is
// reinstalled, and so is a copy that fails to run.
func hasPinnedVersion(dest string) bool {
	info, err := os.Stat(dest)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	out, err := exec.Command(dest, "--version").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "dprint "+dprintVersion
}

func targetFor(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	case "windows/amd64":
		return "x86_64-pc-windows-msvc", nil
	case "windows/arm64":
		return "aarch64-pc-windows-msvc", nil
	}
	return "", fmt.Errorf("unsupported platform %s/%s", goos, goarch)
}

func download(target, dest string) error {
	url := fmt.Sprintf("https://github.com/dprint/dprint/releases/download/%s/dprint-%s.zip", dprintVersion, target)
	client := http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s: %w", url, err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	for _, entry := range archive.File {
		if entry.Name != "dprint" && entry.Name != "dprint.exe" {
			continue
		}
		return extract(entry, dest)
	}
	return fmt.Errorf("%s has no dprint binary", url)
}

func extract(entry *zip.File, dest string) error {
	in, err := entry.Open()
	if err != nil {
		return fmt.Errorf("open %s in archive: %w", entry.Name, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "install-dprint:", err)
	os.Exit(1)
}

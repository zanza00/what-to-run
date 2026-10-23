//go:build !windows

package core

import "os/exec"

// applyPlatformCommand is a no-op outside Windows: the POSIX shells os/exec
// uses follow the same quoting rules as the command's argv.
func applyPlatformCommand(cmd *exec.Cmd, args []string) {}

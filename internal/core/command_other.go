//go:build !windows

package core

import "os/exec"

// applyPlatformCommand is a no-op outside Windows. The POSIX shells that
// os/exec uses follow the same quoting rules as the argv of the command.
func applyPlatformCommand(cmd *exec.Cmd, args []string) {}

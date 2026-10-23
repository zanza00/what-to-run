//go:build windows

package core

import (
	"os/exec"
	"strings"
	"syscall"
)

// applyPlatformCommand builds the raw command line cmd.exe needs from args,
// keeping quotes in the command intact instead of letting os/exec re-quote
// them with the C runtime rules cmd.exe does not follow.
func applyPlatformCommand(cmd *exec.Cmd, args []string) {
	if len(args) == 3 && strings.EqualFold(args[0], "cmd") && args[1] == "/c" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdLineFor(args[2])}
	}
}

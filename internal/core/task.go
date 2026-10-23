// Package core is the task domain: the model, the source registry and the
// parsers. It has no dependency on the terminal UI.
package core

import (
	"os/exec"
	"strings"
)

// Task is a script discovered in the project: it can be run and shown in
// the picker.
type Task struct {
	Source string   // badge: the file declaring the task, or "rc" for rc file tasks
	Name   string   // task name as declared in Source
	Desc   string   // human-readable description; may be empty
	Args   []string // argv of the command that runs the task
	Dir    string   // working directory; "" runs in the picker's directory
}

// Command returns the process that runs the task.
func (t Task) Command() *exec.Cmd {
	cmd := exec.Command(t.Args[0], t.Args[1:]...)
	applyPlatformCommand(cmd, t.Args)
	cmd.Dir = t.Dir
	return cmd
}

// CommandLine returns the task's command as a shell line in the platform's
// default dialect, ready to be placed in the shell prompt by the inline
// widget (see internal/shell).
func (t Task) CommandLine() string {
	return t.CommandLineFor(DefaultShellDialect())
}

// CommandLineFor returns the task's command as a shell line quoted for the
// given dialect, ready to be placed in a prompt by an inline widget.
func (t Task) CommandLineFor(shell ShellDialect) string {
	parts := make([]string, len(t.Args))
	for i, arg := range t.Args {
		parts[i] = quoteArgFor(shell, arg)
	}
	return strings.Join(parts, " ")
}

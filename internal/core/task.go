// Package core is the task domain: the model, the source registry and the
// parsers. The package has no dependency on the terminal UI.
package core

import (
	"os/exec"
	"strings"
)

// Task is a script discovered in the project. The picker shows the script and
// runs it.
type Task struct {
	// Source is the badge: the file that declares the task, or "rc" for a
	// task of the rc file.
	Source string
	// Name is the task name as declared in Source.
	Name string
	// Desc is the human-readable description, and it may be empty.
	Desc string
	// Args is the argv of the command that runs the task.
	Args []string
	// Dir is the working directory. An empty value runs the task in the
	// directory of the picker.
	Dir string
}

// Command returns the process that runs the task.
func (t Task) Command() *exec.Cmd {
	cmd := exec.Command(t.Args[0], t.Args[1:]...)
	applyPlatformCommand(cmd, t.Args)
	cmd.Dir = t.Dir
	return cmd
}

// CommandLineFor returns the command of the task as a shell line, quoted for
// the given dialect. An inline widget can place the line in a prompt. In
// PowerShell, a quoted command word is a string expression, so the line needs
// the call operator (&) to run it.
func (t Task) CommandLineFor(shell ShellDialect) string {
	parts := make([]string, len(t.Args))
	for i, arg := range t.Args {
		parts[i] = quoteArgFor(shell, arg)
	}
	if shell == ShellPowerShell && len(parts) > 0 && parts[0] != t.Args[0] {
		parts[0] = "& " + parts[0]
	}
	return strings.Join(parts, " ")
}

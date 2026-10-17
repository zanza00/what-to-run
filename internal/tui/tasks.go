// The Task model: a script discovered in a project.
package tui

import (
	"os/exec"
	"strings"
	"unicode"
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

// command returns the process that runs the task.
func (t Task) command() *exec.Cmd {
	cmd := exec.Command(t.Args[0], t.Args[1:]...)
	cmd.Dir = t.Dir
	return cmd
}

// commandLine returns the task's command as a shell line, ready to be placed
// in the shell prompt by the inline widget (see scripts/ctrl-s.zsh).
func (t Task) commandLine() string {
	parts := make([]string, len(t.Args))
	for i, arg := range t.Args {
		parts[i] = quoteArg(arg)
	}
	return strings.Join(parts, " ")
}

// quoteArg quotes arg the way a shell reads it back unchanged: plain words
// pass through, anything else is single-quoted.
func quoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, r := range arg {
		if r == '_' || r == '.' || r == '/' || r == '-' || r == '@' || r == '%' ||
			r == ':' || r == '=' || r == '+' || r == ',' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
	}
	return arg
}

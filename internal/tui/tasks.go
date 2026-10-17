// The Task model: a script discovered in a project.
package tui

import "os/exec"

// Task is a script discovered in the project: it can be run and shown in
// the picker.
type Task struct {
	Source string   // badge: the file declaring the task
	Name   string   // task name as declared in Source
	Desc   string   // human-readable description; may be empty
	Args   []string // argv of the command that runs the task
}

// command returns the process that runs the task.
func (t Task) command() *exec.Cmd {
	return exec.Command(t.Args[0], t.Args[1:]...)
}

// Task descriptions: helpOrBody, the help-else-body chain of a description.
// The parsers whose task entries carry an optional help string build their
// descriptions through it, and so do the rc tasks. A parser that describes
// its tasks another way stays outside it, such as the alias expansion of the
// cargo source or the desc, summary and command line ladder of the Taskfile
// source.
package core

// helpOrBody returns the description of one task: its help text when it has
// one, and otherwise its body. The body is display text. It shows what the
// task's entry says, and it never promises the command that the picker runs.
func helpOrBody(help, body string) string {
	if help != "" {
		return help
	}
	return body
}

// The identity of a warning.
//
// A warning carries three facts: the kind, the dismissal key and the message.
// The key is the identity, and the message never enters it. In this way a
// reword of the message cannot un-dismiss a dismissal, and the file a warning
// names keeps its identity when the picker starts from another subdirectory.
// The picker displays the message alone, byte for byte.

package core

import "strings"

// WarningKind names the subject a warning is about. It names the subject,
// not the origin: theme warnings from the --theme flag and from the rc file
// share one kind, which is what the theme-save guard asks.
type WarningKind string

// The five subjects of the closed set. Their names open the dismissal key.
const (
	KindTheme    WarningKind = "theme"
	KindShell    WarningKind = "shell"
	KindRC       WarningKind = "rc"
	KindSource   WarningKind = "source"
	KindRegistry WarningKind = "registry"
)

// Warning is one problem that the picker reports to the user. It implements
// error, so the older error plumbing keeps working while the identity stays
// separate from the message.
type Warning struct {
	kind    WarningKind
	key     string
	message string
}

var _ error = Warning{}

// Error returns the message that the picker displays.
func (w Warning) Error() string { return w.message }

// Kind returns the subject the warning is about.
func (w Warning) Kind() WarningKind { return w.kind }

// Key returns the dismissal key, the line that the dismissed-warnings file
// stores. The key never depends on the message.
func (w Warning) Key() string { return w.key }

// NewSourceWarning names one broken file of a task source. The leaf is the
// file name alone, because a source that searches the ancestors finds the same
// file from every subdirectory of the project. The leaf, not the badge: the
// badge names the directory the picker started from, and it changes with it.
// An unresolvable directory of the parent search appears here too, with the
// directory in place of the leaf.
//
// A read failure and a parse failure of one file share the warning, so one
// broken file holds one identity. Two broken files of one source that share a
// leaf name, one in the project directory and one above it, also share one
// entry; this rarity of monorepos is accepted on purpose.
func NewSourceWarning(id, leaf, message string) Warning {
	return newWarning(KindSource, id, leaf, message)
}

// NewRegistryWarning names a value that the rc disable list stores but the
// source registry does not know. The raw stored value is the identity, so the
// loader's validation and this package's safety net produce the same key for
// one bad entry, whatever file or block it sits in.
func NewRegistryWarning(value, message string) Warning {
	return newWarning(KindRegistry, value, "", message)
}

// NewShellWarning names a stored shell value that the picker cannot use. The
// raw stored value is the identity.
func NewShellWarning(value, message string) Warning {
	return newWarning(KindShell, value, "", message)
}

// NewThemeWarning names a theme problem. The origin is where the theme value
// comes from: "flag", "user", "project" or a "projects/<path>" block. The name
// is the theme name when the problem is about one.
func NewThemeWarning(origin, name, message string) Warning {
	return newWarning(KindTheme, origin, name, message)
}

// NewRCWarning names one problem of an rc file. The origin is "project",
// "user" or a "projects/<path>" block, never the display label of the file.
// The problem is a stable short code, and the offending value completes the
// detail when the problem has one. The three stay in the key, so one
// dismissed problem does not silence a new one.
func NewRCWarning(origin, problem, value, message string) Warning {
	detail := problem
	if value != "" {
		detail += "." + value
	}
	return newWarning(KindRC, origin, detail, message)
}

// newWarning builds the warning and its dismissal key. The grammar is
// kind:subject[:detail] on one line, and sanitizeKeyPart holds that shape
// when a path or a value carries whitespace: the dismissed-warnings file
// stores one entry per line.
func newWarning(kind WarningKind, subject, detail, message string) Warning {
	key := string(kind) + ":" + sanitizeKeyPart(subject)
	if d := sanitizeKeyPart(detail); d != "" {
		key += ":" + d
	}
	return Warning{kind: kind, key: key, message: message}
}

// sanitizeKeyPart replaces every run of whitespace with one underscore, so an
// offending value keeps the key on one line and without spaces.
func sanitizeKeyPart(s string) string {
	return strings.Join(strings.Fields(s), "_")
}

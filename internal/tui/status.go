package tui

// statusKind is the paint that the help line gives the status text. The zero
// kind means that there is no status, so the help line shows the task count
// and the hints.
type statusKind int

const (
	// statusNone means that the model holds no status.
	statusNone statusKind = iota
	// statusRunning means that a task is running in place.
	statusRunning
	// statusOk means that a run finished without an error.
	statusOk
	// statusErr means that a run finished with an error, or that a save
	// failed.
	statusErr
)

// A status is the message that the help line paints in the place of the task
// count and the hints: the outcome of the last run in place, or the failure
// of a theme save or a dismissal save. The model holds no styled bytes, and
// the chrome paints at render time through the styles of the active theme.
// The notice follows the same rule. A status stays until another status
// replaces it.
type status struct {
	kind statusKind
	// text is the message exactly as the help line paints it, icon
	// included. Only a run outcome carries an icon, so the kind cannot
	// derive it.
	text string
}

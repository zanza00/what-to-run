# what-to-run

The TUI picker that runs the tasks a project declares. This glossary is the
shared language of the picker; the user reference lives in `docs/`.

## Language

**Box**:
A bordered drawing that the picker lays over the frame: a title on the top
border, body rows, and a footer, all fitted to one column width. The warning
box and the theme picker are boxes.
*Avoid*: modal (that is the keyboard state), dialog, panel

**Modal**:
The keyboard state in which a box is open and captures the keys, with its own
result behavior. The warning modal owns `d`, the theme modal owns the live
preview.
*Avoid*: box (that is the drawing), dialog

**Placed box**:
A rendered box plus the cell where its first line starts on the frame. The
caller owns the placement rule; the box owns the drawing.
*Avoid*: overlay (that is the compositor that draws placed boxes)

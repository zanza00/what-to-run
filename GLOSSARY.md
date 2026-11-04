# what-to-run

The TUI picker that runs the tasks a project declares. This glossary is the
shared language of the picker; the user reference lives in `docs/`.

## Language

**Box**:
A bordered drawing that the picker lays over the frame: a title on the top
border, body rows, and a footer, all fitted to one column width. The warning
box and the theme picker are boxes.
*Avoid*: modal (that is the keyboard state), dialog, panel

**Dismissal key**:
The stable single-line identity of a warning in the dismissed-warnings
file: its kind, its subject and its offending value. The wording of the
message never enters it.
*Avoid*: warning key (the message text), checksum, badge

**Frame**:
The painted rows at one width and height, with the boxes placed over them.
The frame owns its own geometry facts, and painting turns it into the
string the view shows.
*Avoid*: canvas, the compositor (that draws placed boxes)

**Modal**:
The keyboard state in which a box is open and captures the keys, with its own
result behavior. The warning modal owns `d`, the theme modal owns the live
preview.
*Avoid*: box (that is the drawing), dialog

**Placed box**:
A rendered box plus the cell where its first line starts on the frame. The
caller owns the placement rule; the box owns the drawing.
*Avoid*: overlay (that is the compositor that draws placed boxes)

**Warning**:
A problem the picker reports without failing: a project file that cannot be
read or parsed, a bad rc entry, or an unusable flag value. A warning is
dismissable; the dismissed set lives in the cache file.
*Avoid*: error (the general carrier), notice (the status toast)

**Warning kind**:
The subject family a warning belongs to. A closed set of five: theme,
shell, rc, source, registry.
*Avoid*: type, category, origin

**Window**:
The pair of facts that decides which rows of a list one frame shows: the
cursor, one selected row, and the offset, the first row shown. A window is
clamped to its list and to the rows that the box or the frame gives it.
*Avoid*: viewport (that is a scrollable canvas), pane, scroll

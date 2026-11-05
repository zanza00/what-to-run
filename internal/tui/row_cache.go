package tui

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/zanza00/what-to-run/internal/core"
)

// The row cache owns the derivations of the task rows: the pre-paint algebra
// of row.go, moved out of taskRow so that a steady frame stops repeating it.
//
// Two laws hold:
//
//   - One era holds every entry. An era is the generation of the listing
//     together with the width of the frame, and a change of either one
//     empties the cache wholesale. The cache never checks the freshness of
//     a single entry: no per-entry validity flag, no partial eviction, no
//     fallback lane. A stale entry cannot survive, because the list module
//     bumps the generation at every mutation of its ranking (see list.go),
//     and the width comes from the frame.
//   - The paint stays fresh. The entries carry no style, so the selection,
//     the theme and the ctrl+t preview never invalidate them: the paint
//     applies the styles of the frame to the derivations of the open era on
//     every frame.
//
// The fill is lazy: a row fills at its first paint of an era, never before,
// and the entries follow the rows that the era visited. A long scroll can
// gather one entry per ranked row, so the worst case holds one entry for
// every row of the ranking, and the first keystroke or resize that opens
// the next era empties them all.

// era is the validity span of the derivations of the row cache: one
// generation of the listing together with one width of the frame. The two
// numbers name one fact, and the derivations of one era cannot serve
// another.
type era struct {
	// gen is the generation of the listing that the entries belong to; the
	// read of the fact is taskList.generation().
	gen int
	// width is the width of the frame that the entries were derived at.
	width int
}

// rowDerivations is the derivation of one task row: the facts that the row
// derives from the task, the filter tokens and the width of the frame,
// before any style applies to them. The values are style-free, so the
// styles of a frame paint them without a new derivation.
type rowDerivations struct {
	// badgeW is the width of the source badge.
	badgeW int
	// cmdW is the command column for this row: the widest command line of
	// the ranking, capped to the room that the row has.
	cmdW int
	// cmd is the command line, truncated with the ellipsis of row.go and
	// padded to cmdW columns. The pad carries no match.
	cmd     string
	cmdMask []bool // the match mask of cmd, on the runes before the pad

	// descLine is the description line: a leading space plus the flattened
	// description, truncated with an ellipsis. The rc rule can drop the
	// description, and a narrow row can lose its room, and both cases leave
	// the line empty.
	descLine string
	// descMask marks the runes of descLine that carry a filter match, with
	// the leading space marked as unmatched.
	descMask []bool
}

// rowCache holds the derivations of one era of the task rows, keyed by the
// position of the row in the ranked slice. listLines is the only consumer.
type rowCache struct {
	// open holds the era that the entries belong to.
	open era
	// rows holds the derivations of the open era, keyed by the position in
	// the ranked slice.
	rows map[int]rowDerivations
}

// openEra adopts the era of a frame. A change of the generation or of the
// width ends the previous era, and the entries of an ended era are stale
// wholesale, so the cache empties in one step.
func (c *rowCache) openEra(e era) {
	if c.open != e {
		c.open = e
		c.rows = make(map[int]rowDerivations)
	}
}

// lookup returns the derivation that the era holds for the row at pos in the
// ranked slice. The boolean reports whether the era holds one, so the caller
// of a miss fills through the one derivation function and the caller of a
// hit paints the stored derivation.
func (c *rowCache) lookup(e era, pos int) (rowDerivations, bool) {
	c.openEra(e)
	d, ok := c.rows[pos]
	return d, ok
}

// fill stores the derivation of the row at pos as part of the era e. The
// derivations of an earlier era are stale wholesale, so a changed era
// empties the cache before the store.
func (c *rowCache) fill(e era, pos int, d rowDerivations) {
	c.openEra(e)
	c.rows[pos] = d
}

// deriveRow is the one derivation function of a task row: the caller of a
// miss fills the cache with it, and no other code derives the facts of the
// row. The algebra is the truncation, the match and the pad of row.go under
// the filter tokens of the listing, at the width of the frame.
func deriveRow(t core.Task, shell core.ShellDialect, tokens []string, listCmdW, w int) rowDerivations {
	badgeW := lipgloss.Width(t.Source)
	room := max(w-badgeW-1, 0) // columns left of the badge separator
	avail := max(room-2, 0)    // columns after the marker

	cmdW := min(listCmdW, avail)
	cmdRaw := t.CommandLineFor(shell)
	cmd, cmdMask := truncateMatched(cmdRaw, matchPositions(cmdRaw, tokens), cmdW)
	cmd += strings.Repeat(" ", max(cmdW-lipgloss.Width(cmd), 0))

	desc := t.Desc
	if t.Source == "rc" && slices.Contains(t.Args, desc) {
		// The command already says it. An rc task without a custom
		// description defaults to its own command (sh -c <cmd>).
		desc = ""
	}
	desc = flattenDesc(desc)
	var (
		descLine string
		descMask []bool
	)
	if descW := avail - cmdW - 1; desc != "" && descW > 0 {
		var dm []bool
		desc, dm = truncateMatched(desc, matchPositions(desc, tokens), descW)
		descLine = " " + desc
		if dm != nil {
			descMask = append([]bool{false}, dm...)
		}
	}
	return rowDerivations{
		badgeW:   badgeW,
		cmdW:     cmdW,
		cmd:      cmd,
		cmdMask:  cmdMask,
		descLine: descLine,
		descMask: descMask,
	}
}

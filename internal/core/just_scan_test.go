package core

import "testing"

// TestJustScan covers the scan that every justfile stop question shares. The
// table holds the escape and the quote edges for each kind of stop: a stop
// inside a quoted string never stops the scan, an escaped quote closes nothing,
// and a backslash outside a quoted string escapes nothing. The expectations
// come from the justfile rules, with the bytes of each case counted by hand.
func TestJustScan(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		stops    string
		wantAt   int
		wantOpen bool
	}{
		{
			name:   "no stop and no quote",
			text:   "plain text",
			stops:  "#",
			wantAt: -1,
		},
		{
			name:   "a stop outside a quote",
			text:   "run a # build",
			stops:  "#",
			wantAt: 6,
		},
		{
			name:   "a double-quoted stop does not stop",
			text:   `"quoted # text"`,
			stops:  "#",
			wantAt: -1,
		},
		{
			name:   "a single-quoted stop does not stop",
			text:   `'quoted # text'`,
			stops:  "#",
			wantAt: -1,
		},
		{
			name:     "an open quote stays open at the end",
			text:     `"never closed`,
			stops:    "#",
			wantAt:   -1,
			wantOpen: true,
		},
		{
			name:   "the scan reports the first stop it meets",
			text:   `a # "open`,
			stops:  "#",
			wantAt: 2,
		},
		{
			// The backslash inside a quoted string escapes the quote, so
			// the string stays open at the end.
			name:     "an escaped quote does not close",
			text:     `"a\" still open`,
			stops:    "#",
			wantAt:   -1,
			wantOpen: true,
		},
		{
			name:   "the quote after an escaped quote closes",
			text:   `"escaped quote \" still open"`,
			stops:  "#",
			wantAt: -1,
		},
		{
			// A backslash escapes a byte only inside a quoted string.
			name:   "a backslash outside a quote escapes nothing",
			text:   `a\#b`,
			stops:  "#",
			wantAt: 2,
		},
		{
			name:   "the first byte stops the scan",
			text:   "#first",
			stops:  "#",
			wantAt: 0,
		},
		{
			// The double quote sits inside a single-quoted string, so it
			// neither closes nor opens one.
			name:   "only the same quote closes a quote",
			text:   `a'"b"'c#d`,
			stops:  "#",
			wantAt: 7,
		},
		{
			name:   "a plain closing bracket",
			text:   "[private]",
			stops:  "]",
			wantAt: 8,
		},
		{
			name:   "a bracket inside a doc string does not close",
			text:   `[doc("a ] b")]`,
			stops:  "]",
			wantAt: 13,
		},
		{
			name:     "a doc string that never closes",
			text:     `[doc("never closed]`,
			stops:    "]",
			wantAt:   -1,
			wantOpen: true,
		},
		{
			name:   "the first bracket outside a quote ends the scan",
			text:   `[a]'b]c'`,
			stops:  "]",
			wantAt: 2,
		},
		{
			// The escaped quote lets the bracket pass as string text.
			name:   "a bracket after an escaped quote stays inside",
			text:   `[doc("a \"] b")]`,
			stops:  "]",
			wantAt: 15,
		},
		{
			name:   "no comma in a lone attribute",
			text:   "private",
			stops:  ",",
			wantAt: -1,
		},
		{
			name:   "the comma between two attributes",
			text:   `private, group("g")`,
			stops:  ",",
			wantAt: 7,
		},
		{
			name:   "a comma inside a doc string does not separate",
			text:   `doc("a, b"), linux`,
			stops:  ",",
			wantAt: 11,
		},
		{
			name:   "the first of several commas",
			text:   "windows, unix,",
			stops:  ",",
			wantAt: 7,
		},
		{
			name:   "the comma of a single-quoted doc string does not separate",
			text:   `'a, b', c`,
			stops:  ",",
			wantAt: 6,
		},
		{
			name:     "a doc string that never closes before a comma",
			text:     `doc("endless,`,
			stops:    ",",
			wantAt:   -1,
			wantOpen: true,
		},
		{
			name:   "the colon of a recipe header",
			text:   "build: dep",
			stops:  ":#",
			wantAt: 5,
		},
		{
			// justScan reports the first colon; the caller that asks the
			// header question skips the ":=" of an assignment.
			name:   "the colon of an assignment stops the scan too",
			text:   `version := "1.0"`,
			stops:  ":#",
			wantAt: 8,
		},
		{
			name:   "the colon of an import path stays inside a quote",
			text:   `import "x:y.just" # note`,
			stops:  ":#",
			wantAt: 18,
		},
		{
			name:   "a quoted colon and a quoted stop do not stop",
			text:   `"#" first`,
			stops:  ":#",
			wantAt: -1,
		},
		{
			name:   "the colon at the start of the text",
			text:   ":'a:b'",
			stops:  ":#",
			wantAt: 0,
		},
		{
			name:   "the stop after a closed pair of quotes",
			text:   "a''b:c",
			stops:  ":#",
			wantAt: 4,
		},
		{
			name:   "the colons inside a double-quoted string do not stop",
			text:   `"x:' y"`,
			stops:  ":#",
			wantAt: -1,
		},
		{
			name:   "a comment stops the scan before a colon later in the text",
			text:   "tag#not:colon",
			stops:  ":#",
			wantAt: 3,
		},
		{
			name:     "an open quote before an unclosed colon",
			text:     `"a`,
			stops:    ":#",
			wantAt:   -1,
			wantOpen: true,
		},
		{
			name:   "empty text stops nowhere",
			text:   "",
			stops:  "#",
			wantAt: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at, open := justScan(tt.text, tt.stops)
			if at != tt.wantAt {
				t.Errorf("justScan(%q, %q) at = %d, want %d", tt.text, tt.stops, at, tt.wantAt)
			}
			if open != tt.wantOpen {
				t.Errorf("justScan(%q, %q) open = %v, want %v", tt.text, tt.stops, open, tt.wantOpen)
			}
		})
	}
}

// The just scan: the one quote-aware scanner that the justfile parser asks
// its stop questions on.
package core

import "strings"

// justScan scans text and returns the index of the first stop byte that sits
// outside a quoted string, or -1 when it finds none. Quoted strings are read
// like just reads them: a double or single quote opens one, the same quote
// closes it, and a backslash inside it escapes the next byte. A backslash
// outside a quoted string escapes nothing. open reports whether a quote was
// left unclosed when the scan ran to the end of the text. The scan stops
// outside a quote, so open is false at a stop, and a caller that resumes the
// scan after a stop starts with the quotes closed.
func justScan(text, stops string) (at int, open bool) {
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // the escaped byte cannot close the quote
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case strings.IndexByte(stops, c) >= 0:
			return i, false
		}
	}
	return -1, quote != 0
}

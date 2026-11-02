// String items: joining the array items of a task source into one line of
// display text.
package core

import "strings"

// joinStringItems joins the items of an array with sep, as one line of
// display text. ok is false when one item is not a string, and the caller
// skips the entry.
func joinStringItems(items []any, sep string) (string, bool) {
	parts := make([]string, len(items))
	for i, item := range items {
		s, isString := item.(string)
		if !isString {
			return "", false
		}
		parts[i] = s
	}
	return strings.Join(parts, sep), true
}

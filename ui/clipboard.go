package ui

import "github.com/atotto/clipboard"

// clipboardWriteAll is the real clipboard write function, kept in its own file
// so the atotto import only lives here and can be excluded during testing.
func clipboardWriteAll(s string) error {
	return clipboard.WriteAll(s)
}

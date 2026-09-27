package internal

import (
	"strings"
	"unicode/utf8"
)

// WrapBreak is where to cut text so the first part fits in maxWidth, given the
// whole of it measures width. It estimates by characters, prefers the last
// space before that, and never splits a character: slicing by bytes cut
// multi-byte text such as icons and non-Latin titles into squares.
func WrapBreak(text string, width, maxWidth int32) int {
	count := utf8.RuneCountInString(text)
	chars := 1
	if width > 0 {
		chars = int(float32(count) * float32(maxWidth) / float32(width))
	}
	chars = max(chars, 1)
	if chars >= count {
		return len(text)
	}

	cut := len(text)
	seen := 0
	for i := range text {
		if seen == chars {
			cut = i
			break
		}
		seen++
	}

	// A space is one byte, so looking at the byte at cut is safe.
	if space := strings.LastIndexByte(text[:cut+1], ' '); space > 0 {
		return space
	}
	return cut
}

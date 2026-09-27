package internal

import "strings"

// Ellipsize shortens text to fit maxWidth, ending it with an ellipsis, as
// measured by measure. Text that fits is returned as it is, and characters are
// never cut through.
func Ellipsize(text string, maxWidth int32, measure func(string) int32) string {
	if measure(text) <= maxWidth {
		return text
	}

	const ellipsis = "…"
	runes := []rune(text)
	for n := len(runes) - 1; n > 0; n-- {
		if candidate := strings.TrimRight(string(runes[:n]), " ") + ellipsis; measure(candidate) <= maxWidth {
			return candidate
		}
	}
	if measure(ellipsis) <= maxWidth {
		return ellipsis
	}
	return ""
}

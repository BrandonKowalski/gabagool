package internal

import (
	"testing"
	"unicode/utf8"
)

// Ten pixels a character, whatever the character.
func tenEach(s string) int32 { return int32(utf8.RuneCountInString(s) * 10) }

func TestEllipsize_FitsIsUntouched(t *testing.T) {
	if got := Ellipsize("Game Boy", 100, tenEach); got != "Game Boy" {
		t.Errorf("got %q, want it untouched", got)
	}
}

// Text that runs past the width is cut to fit with an ellipsis, never through
// a character.
func TestEllipsize_CutsToFit(t *testing.T) {
	got := Ellipsize("Sep 27 · Super Nintendo Entertainment System", 200, tenEach)
	if tenEach(got) > 200 {
		t.Errorf("%q is %dpx, wider than 200", got, tenEach(got))
	}
	if got != "Sep 27 · Super Nint…" {
		t.Errorf("got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Errorf("%q is not valid text", got)
	}
}

// Not even the ellipsis fits: nothing is better than overflowing.
func TestEllipsize_NoRoomAtAll(t *testing.T) {
	if got := Ellipsize("Game Boy", 5, tenEach); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

// A cut that lands after a space does not leave the space hanging before the
// ellipsis.
func TestEllipsize_NoSpaceBeforeTheEllipsis(t *testing.T) {
	if got := Ellipsize("Super Nintendo", 70, tenEach); got != "Super…" {
		t.Errorf("got %q, want %q", got, "Super…")
	}
}

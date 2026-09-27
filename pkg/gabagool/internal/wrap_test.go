package internal

import (
	"testing"
	"unicode/utf8"
)

// A break never lands inside a character. Cutting a four byte icon between
// its bytes drew it as four squares in a table cell.
func TestWrapBreak_KeepsCharactersWhole(t *testing.T) {
	for _, text := range []string{
		"\U000F0B7E",                     // one Material Design icon
		"\U000F0B7E\U000F0B7D\U000F0163", // several
		"ポケットモンスター 赤",                    // Japanese title
		"Pokémon Émeraude",
	} {
		for maxWidth := int32(1); maxWidth < 100; maxWidth += 7 {
			cut := WrapBreak(text, 100, maxWidth)
			if cut <= 0 || cut > len(text) {
				t.Fatalf("%q at %d: cut %d is outside the text", text, maxWidth, cut)
			}
			if !utf8.ValidString(text[:cut]) || !utf8.ValidString(text[cut:]) {
				t.Errorf("%q at %d: cut %d splits a character", text, maxWidth, cut)
			}
		}
	}
}

// Text that fits is left whole.
func TestWrapBreak_TextThatFits(t *testing.T) {
	if cut := WrapBreak("Tetris", 50, 80); cut != len("Tetris") {
		t.Errorf("cut = %d, want the whole text", cut)
	}
}

// A break prefers the last space before the estimate, as it always has.
func TestWrapBreak_PrefersASpace(t *testing.T) {
	text := "Super Mario World"
	// Half the width is about eight characters, inside "Mario".
	if cut := WrapBreak(text, 170, 85); text[:cut] != "Super" {
		t.Errorf("first line = %q, want %q", text[:cut], "Super")
	}
}

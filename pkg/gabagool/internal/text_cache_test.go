package internal

import (
	"testing"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// A collision in the cache key draws the wrong text, or the right text in the
// wrong colour, which is the only user-visible risk of caching rendered text.
func TestTextCacheKeyDistinguishesEveryInput(t *testing.T) {
	fontA, fontB := new(ttf.Font), new(ttf.Font)
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	grey := sdl.Color{R: 128, G: 128, B: 128, A: 255}

	base := textCacheKey(fontA, "Settings", white)

	cases := []struct {
		name string
		key  string
	}{
		{"different font", textCacheKey(fontB, "Settings", white)},
		{"different colour", textCacheKey(fontA, "Settings", grey)},
		{"different text", textCacheKey(fontA, "Storage", white)},
		{"different alpha only", textCacheKey(fontA, "Settings", sdl.Color{R: 255, G: 255, B: 255, A: 128})},
	}
	for _, tc := range cases {
		if tc.key == base {
			t.Errorf("%s: key collided with base key %q", tc.name, base)
		}
	}

	if again := textCacheKey(fontA, "Settings", white); again != base {
		t.Errorf("same inputs produced different keys: %q vs %q", base, again)
	}
}

// Text is rendered from user-supplied strings, so a string containing the
// separator must not be able to impersonate another font or colour.
func TestTextCacheKeySeparatorCannotBeForged(t *testing.T) {
	font := new(ttf.Font)
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	black := sdl.Color{R: 0, G: 0, B: 0, A: 255}

	honest := textCacheKey(font, "hello", black)
	forged := textCacheKey(font, "ffffffff|hello", white)

	if honest == forged {
		t.Errorf("a string containing the separator forged another colour's key: %q", honest)
	}
}

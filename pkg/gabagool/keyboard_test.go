package gabagool

import (
	"fmt"
	"testing"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/sdl"
)

// testKeyboard builds a keyboard's keys and rects the way Keyboard and
// URLKeyboard do, without the help overlay, which needs a window.
func testKeyboard(name string, width, height int32) *virtualKeyboard {
	kb := &virtualKeyboard{}
	switch name {
	case "general":
		kb.Layout = KeyboardLayoutGeneral
		kb.Keys, kb.keyLayout = createKeys(), createKeyLayout()
		setupKeyboardRects(kb, width, height)
	case "url":
		kb.Layout = KeyboardLayoutURL
		kb.Keys, kb.keyLayout = createURLKeys(), createURLKeyLayout()
		setupURLKeyboardRects(kb, width, height)
	case "url-10-shortcuts":
		kb.Layout = KeyboardLayoutURL
		kb.Keys, kb.keyLayout = createURLKeysWithShortcuts10(defaultURLShortcuts), createURLKeyLayoutFor10()
		setupURLKeyboardRectsFor10(kb, width, height)
	case "numeric":
		kb.Layout = KeyboardLayoutNumeric
		kb.Keys, kb.keyLayout = createNumericKeys(), createNumericKeyLayout()
		setupNumericKeyboardRects(kb, width, height)
	}
	return kb
}

func keyName(kb *virtualKeyboard, entry interface{}) string {
	if idx, ok := entry.(int); ok {
		return kb.Keys[idx].LowerValue
	}
	return entry.(string)
}

func covers(rect sdl.Rect, x int32) bool {
	return x >= rect.X && x < rect.X+rect.W
}

// Up and down go to the key drawn above or below, not to the key with the same
// position in its row. With rows of different lengths those were different
// keys: down from 2 selected w, which sits under 3.
//
// Going back the other way returns to the key you started from, so passing
// through a wide key such as space does not lose your column.
func TestKeyboardVerticalMovesFollowTheDrawing(t *testing.T) {
	sizes := [][2]int32{{640, 480}, {720, 720}, {1024, 768}, {1280, 720}}
	layouts := []string{"general", "url", "url-10-shortcuts", "numeric"}

	moves := []struct {
		dir, back constants.VirtualButton
		step      int
	}{
		{constants.VirtualButtonDown, constants.VirtualButtonUp, 1},
		{constants.VirtualButtonUp, constants.VirtualButtonDown, -1},
	}

	for _, size := range sizes {
		for _, name := range layouts {
			t.Run(fmt.Sprintf("%s %dx%d", name, size[0], size[1]), func(t *testing.T) {
				kb := testKeyboard(name, size[0], size[1])
				rows := kb.keyLayout.rows

				for r, row := range rows {
					for c, entry := range row {
						for _, move := range moves {
							next := r + move.step
							if next < 0 || next >= len(rows) {
								continue
							}

							// Arrive with a sideways move, as a user would, so no
							// column is remembered from earlier.
							kb.setSelection(kb.keyLayout, r, c)
							kb.navAnchorSet = false

							kb.navigate(move.dir)
							gotRow, gotCol := kb.findCurrentPosition(kb.keyLayout)
							landed := rows[gotRow][gotCol]

							from := kb.keyRect(entry)
							center := from.X + from.W/2
							to := kb.keyRect(landed)

							drawnThere := false
							for _, candidate := range rows[next] {
								if covers(kb.keyRect(candidate), center) {
									drawnThere = true
								}
							}
							if drawnThere && !covers(to, center) {
								t.Errorf("from %q went to %q, which is not drawn under it",
									keyName(kb, entry), keyName(kb, landed))
							}

							kb.navigate(move.back)
							if backRow, backCol := kb.findCurrentPosition(kb.keyLayout); backRow != r || backCol != c {
								t.Errorf("from %q and back again ended on %q",
									keyName(kb, entry), keyName(kb, rows[backRow][backCol]))
							}
						}
					}
				}
			})
		}
	}
}

// Down from a key that has nothing straight below it, such as backspace above
// the empty corner of the QWERTY row, picks the nearest key.
func TestKeyboardMovesToTheNearestKeyWhenNothingIsBelow(t *testing.T) {
	kb := testKeyboard("general", 1024, 768)

	kb.setSelection(kb.keyLayout, 0, 10) // backspace
	kb.navigate(constants.VirtualButtonDown)

	row, col := kb.findCurrentPosition(kb.keyLayout)
	if got := keyName(kb, kb.keyLayout.rows[row][col]); got != "p" {
		t.Errorf("down from backspace went to %q, want p", got)
	}
}

// Keys never overlap one another or leave the window.
func TestKeyboardKeysFitWithoutOverlapping(t *testing.T) {
	for _, name := range []string{"general", "url", "url-10-shortcuts", "numeric"} {
		kb := testKeyboard(name, 640, 480)

		var rects []sdl.Rect
		for _, row := range kb.keyLayout.rows {
			for _, entry := range row {
				rects = append(rects, kb.keyRect(entry))
			}
		}

		for i, a := range rects {
			if a.X < 0 || a.Y < 0 || a.X+a.W > 640 || a.Y+a.H > 480 {
				t.Errorf("%s: key %v is outside the 640x480 window", name, a)
			}
			for _, b := range rects[i+1:] {
				if a.HasIntersection(&b) {
					t.Errorf("%s: keys %v and %v overlap", name, a, b)
				}
			}
		}
	}
}

package internal

import "github.com/veandco/go-sdl2/sdl"

// KeyboardDimensions holds calculated keyboard positioning values.
// This is computed once from window dimensions and reused across layout setup.
type KeyboardDimensions struct {
	WindowWidth     int32
	WindowHeight    int32
	KeyboardWidth   int32
	KeyboardHeight  int32
	StartX          int32
	TextInputY      int32
	KeyboardStartY  int32
	TextInputHeight int32
}

// CalculateKeyboardDimensions computes the standard keyboard dimensions
// based on window size. Uses 85% of window for keyboard area.
func CalculateKeyboardDimensions(windowWidth, windowHeight int32) KeyboardDimensions {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	return KeyboardDimensions{
		WindowWidth:     windowWidth,
		WindowHeight:    windowHeight,
		KeyboardWidth:   keyboardWidth,
		KeyboardHeight:  keyboardHeight,
		StartX:          startX,
		TextInputY:      textInputY,
		KeyboardStartY:  keyboardStartY,
		TextInputHeight: textInputHeight,
	}
}

// KeyboardRect returns the main keyboard area rectangle.
func (d KeyboardDimensions) KeyboardRect() sdl.Rect {
	return sdl.Rect{X: d.StartX, Y: d.KeyboardStartY, W: d.KeyboardWidth, H: d.KeyboardHeight}
}

// TextInputRect returns the text input area rectangle.
func (d KeyboardDimensions) TextInputRect() sdl.Rect {
	return sdl.Rect{X: d.StartX, Y: d.TextInputY, W: d.KeyboardWidth, H: d.TextInputHeight}
}

// KeySizes holds the calculated sizes for different key types.
type KeySizes struct {
	KeyWidth       int32
	KeyHeight      int32
	KeySpacing     int32
	BackspaceWidth int32
	EnterWidth     int32
}

// CalculateKeySizes computes key sizes for QWERTY-style layouts.
// numRows is typically 5 or 6 depending on layout.
func CalculateKeySizes(dims KeyboardDimensions, numRows int) KeySizes {
	keyWidth := dims.KeyboardWidth / 12
	keyHeight := dims.KeyboardHeight / int32(numRows)
	keySpacing := int32(3)

	return KeySizes{
		KeyWidth:       keyWidth,
		KeyHeight:      keyHeight,
		KeySpacing:     keySpacing,
		BackspaceWidth: keyWidth * 2,
		EnterWidth:     keyWidth + keyWidth/2,
	}
}

// CalculateNumericKeySizes computes key sizes for numeric keypad.
func CalculateNumericKeySizes(dims KeyboardDimensions) KeySizes {
	keyWidth := dims.KeyboardWidth / 5
	keyHeight := dims.KeyboardHeight / 5
	keySpacing := int32(5)

	return KeySizes{
		KeyWidth:       keyWidth,
		KeyHeight:      keyHeight,
		KeySpacing:     keySpacing,
		BackspaceWidth: keyWidth,
		EnterWidth:     keyWidth,
	}
}

// RowSpec defines a single row in the keyboard layout.
type RowSpec struct {
	// KeyIndices are the indices into the Keys array for regular keys in this row
	KeyIndices []int
	// KeyWidth override for keys in this row (0 = use default)
	KeyWidth int32
	// LeftKey is the special key on the left (shift, etc.) - empty string if none
	LeftKey string
	// RightKey is the special key on the right (backspace, enter, symbol, etc.) - empty string if none
	RightKey string
}

// LayoutRow positions keys for a single row.
// Returns the next Y position after this row.
func LayoutRow(
	keyRects []sdl.Rect,
	indices []int,
	x, y int32,
	keyWidth, keyHeight, keySpacing int32,
) int32 {
	for _, idx := range indices {
		if idx >= 0 && idx < len(keyRects) {
			keyRects[idx] = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		}
		x += keyWidth + keySpacing
	}
	return x
}

// KeyboardGridColumns is the number of columns in QWERTY-style layouts.
const KeyboardGridColumns = 12

// GridCell returns the rectangle of a key on the QWERTY column grid, spanning
// the given number of columns.
func GridCell(dims KeyboardDimensions, sizes KeySizes, row, col, span int) sdl.Rect {
	pitchX := sizes.KeyWidth + sizes.KeySpacing
	pitchY := sizes.KeyHeight + sizes.KeySpacing
	gridWidth := KeyboardGridColumns*pitchX - sizes.KeySpacing
	return sdl.Rect{
		X: dims.StartX + (dims.KeyboardWidth-gridWidth)/2 + int32(col)*pitchX,
		Y: dims.KeyboardStartY + sizes.KeySpacing + int32(row)*pitchY,
		W: int32(span)*pitchX - sizes.KeySpacing,
		H: sizes.KeyHeight,
	}
}

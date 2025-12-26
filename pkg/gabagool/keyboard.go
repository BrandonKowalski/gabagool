package gabagool

import (
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type key struct {
	Rect        sdl.Rect
	LowerValue  string
	UpperValue  string
	SymbolValue string
	IsPressed   bool
}

type keyboardState int

const (
	lowerCase keyboardState = iota
	upperCase
	symbolsMode
)

// KeyboardLayout specifies the type of keyboard layout to use.
type KeyboardLayout int

const (
	// KeyboardLayoutGeneral is the default QWERTY keyboard layout.
	KeyboardLayoutGeneral KeyboardLayout = iota
	// KeyboardLayoutURL is optimized for entering URLs with shortcuts.
	KeyboardLayoutURL
	// KeyboardLayoutNumeric is a simple numpad for entering numbers.
	KeyboardLayoutNumeric
)

// URLShortcut represents a shortcut key on the URL keyboard.
// Value is shown normally, SymbolValue is shown when symbol mode is active.
type URLShortcut struct {
	Value       string
	SymbolValue string
}

// URLKeyboardConfig holds configuration for the URL keyboard.
type URLKeyboardConfig struct {
	// Shortcuts to display on the URL keyboard (up to 10).
	// 1-5 shortcuts: single row layout
	// 6-10 shortcuts: two row layout
	// If empty, 10 default shortcuts are used (two rows).
	Shortcuts []URLShortcut
}

type virtualKeyboard struct {
	Layout           KeyboardLayout
	keyLayout        *keyLayout
	Keys             []key
	TextBuffer       string
	CurrentState     keyboardState
	ShiftPressed     bool
	SymbolPressed    bool
	BackspaceRect    sdl.Rect
	EnterRect        sdl.Rect
	SpaceRect        sdl.Rect
	ShiftRect        sdl.Rect
	SymbolRect       sdl.Rect
	TextInputRect    sdl.Rect
	KeyboardRect     sdl.Rect
	SelectedKeyIndex int
	SelectedSpecial  int
	CursorPosition   int
	CursorVisible    bool
	LastCursorBlink  time.Time
	CursorBlinkRate  time.Duration
	helpOverlay      *helpOverlay
	helpExitText     string
	ShowingHelp      bool
	EnterPressed     bool
	InputDelay       time.Duration
	lastInputTime    time.Time
	urlShortcuts     []URLShortcut

	heldDirections struct {
		up, down, left, right bool
	}
	lastRepeatTime time.Time
	repeatDelay    time.Duration
	repeatInterval time.Duration
	hasRepeated    bool
}

var defaultKeyboardHelpLines = []string{
	"• D-Pad: Navigate between keys",
	"• A: Type the selected key",
	"• B: Backspace",
	"• X: Space",
	"• L1 / R1: Move cursor within text",
	"• Select: Toggle Shift (uppercase/symbols)",
	"• Y: Exit keyboard without saving",
	"• Start: Enter (confirm input)",
}

var numericKeyboardHelpLines = []string{
	"• D-Pad: Navigate between keys",
	"• A: Type the selected digit",
	"• B: Backspace",
	"• L1 / R1: Move cursor within text",
	"• Y: Exit keyboard without saving",
	"• Start: Enter (confirm input)",
}

var urlKeyboardHelpLines = []string{
	"• D-Pad: Navigate between keys",
	"• A: Type the selected key",
	"• B: Backspace",
	"• X: Toggle symbols (0-9)",
	"• L1 / R1: Move cursor within text",
	"• Select: Toggle Shift (uppercase)",
	"• Y: Exit keyboard without saving",
	"• Start: Enter (confirm input)",
}

var defaultURLShortcuts = []URLShortcut{
	{Value: "https://", SymbolValue: "http://"},
	{Value: "www.", SymbolValue: "ftp://"},
	{Value: ".com", SymbolValue: ".co"},
	{Value: ".org", SymbolValue: ".tv"},
	{Value: ".net", SymbolValue: ".me"},
	{Value: ".io", SymbolValue: ".gg"},
	{Value: ".dev", SymbolValue: ".uk"},
	{Value: ".app", SymbolValue: ".de"},
	{Value: ".edu", SymbolValue: ".ca"},
	{Value: ".gov", SymbolValue: ".au"},
}

type keyLayout struct {
	rows [][]interface{}
}

func createKeyLayout() *keyLayout {
	return &keyLayout{
		rows: [][]interface{}{
			// Row 1: numbers + backspace
			{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, "backspace"},
			// Row 2: qwerty row
			{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			// Row 3: asdf row + enter
			{20, 21, 22, 23, 24, 25, 26, 27, 28, "enter"},
			// Row 4: shift + zxcv row + symbol
			{"shift", 29, 30, 31, 32, 33, 34, 35, "symbol"},
			// Row 5: space only
			{"space"},
		},
	}
}

func createKeyboard(windowWidth, windowHeight int32, helpExitText string, layout KeyboardLayout) *virtualKeyboard {
	kb := &virtualKeyboard{
		Layout:           layout,
		TextBuffer:       "",
		CurrentState:     lowerCase,
		SelectedKeyIndex: 0,
		SelectedSpecial:  0,
		CursorPosition:   0,
		CursorVisible:    true,
		LastCursorBlink:  time.Now(),
		CursorBlinkRate:  500 * time.Millisecond,
		helpExitText:     helpExitText,
		ShowingHelp:      false,
		InputDelay:       100 * time.Millisecond,
		lastInputTime:    time.Now(),
		lastRepeatTime:   time.Now(),
		repeatDelay:      150 * time.Millisecond,
		repeatInterval:   50 * time.Millisecond,
	}

	// Initialize layout-specific keys and rects
	switch layout {
	case KeyboardLayoutURL:
		kb.Keys = createURLKeys()
		kb.keyLayout = createURLKeyLayout()
		kb.helpOverlay = newHelpOverlay("URL Keyboard Help", urlKeyboardHelpLines, helpExitText)
		setupURLKeyboardRects(kb, windowWidth, windowHeight)
	case KeyboardLayoutNumeric:
		kb.Keys = createNumericKeys()
		kb.keyLayout = createNumericKeyLayout()
		kb.helpOverlay = newHelpOverlay("Numeric Keyboard Help", numericKeyboardHelpLines, helpExitText)
		setupNumericKeyboardRects(kb, windowWidth, windowHeight)
	default:
		kb.Keys = createKeys()
		kb.keyLayout = createKeyLayout()
		kb.helpOverlay = newHelpOverlay("Keyboard Help", defaultKeyboardHelpLines, helpExitText)
		setupKeyboardRects(kb, windowWidth, windowHeight)
	}

	return kb
}

func createURLKeyboard(windowWidth, windowHeight int32, helpExitText string, shortcuts []URLShortcut) *virtualKeyboard {
	kb := &virtualKeyboard{
		Layout:           KeyboardLayoutURL,
		TextBuffer:       "",
		CurrentState:     lowerCase,
		SelectedKeyIndex: 0,
		SelectedSpecial:  0,
		CursorPosition:   0,
		CursorVisible:    true,
		LastCursorBlink:  time.Now(),
		CursorBlinkRate:  500 * time.Millisecond,
		helpExitText:     helpExitText,
		ShowingHelp:      false,
		InputDelay:       100 * time.Millisecond,
		lastInputTime:    time.Now(),
		lastRepeatTime:   time.Now(),
		repeatDelay:      150 * time.Millisecond,
		repeatInterval:   50 * time.Millisecond,
		urlShortcuts:     shortcuts,
	}

	// Use 5-row layout if 5 or fewer shortcuts, 6-row layout if more
	if len(shortcuts) <= 5 {
		kb.Keys = createURLKeysWithShortcuts5(shortcuts)
		kb.keyLayout = createURLKeyLayoutFor5()
		setupURLKeyboardRectsFor5(kb, windowWidth, windowHeight)
	} else {
		kb.Keys = createURLKeysWithShortcuts10(shortcuts)
		kb.keyLayout = createURLKeyLayoutFor10()
		setupURLKeyboardRectsFor10(kb, windowWidth, windowHeight)
	}
	kb.helpOverlay = newHelpOverlay("URL Keyboard Help", urlKeyboardHelpLines, helpExitText)

	return kb
}

func createKeys() []key {
	keys := make([]key, 36) // Total number of regular keys

	// Numbers row
	numbers := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	numberSymbols := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")"}

	for i, num := range numbers {
		keys[i] = key{
			LowerValue:  num,
			UpperValue:  num,
			SymbolValue: numberSymbols[i],
		}
	}

	// QWERTY row
	qwerty := "qwertyuiop"
	qwertySymbols := []string{"`", "~", "[", "]", "\\", "|", "{", "}", ";", ":"}
	for i, char := range qwerty {
		keys[10+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: qwertySymbols[i],
		}
	}

	// ASDF row
	asdf := "asdfghjkl"
	asdfSymbols := []string{"'", "\"", "<", ">", "?", "/", "+", "=", "_"}
	for i, char := range asdf {
		keys[20+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: asdfSymbols[i],
		}
	}

	// ZXCV row - avoiding symbols already used
	zxcv := "zxcvbnm"
	zxcvSymbols := []string{",", ".", "-", "€", "£", "¥", "¢"}
	for i, char := range zxcv {
		keys[29+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: zxcvSymbols[i],
		}
	}

	return keys
}

func createURLKeyLayout() *keyLayout {
	return &keyLayout{
		rows: [][]interface{}{
			// Row 1: URL shortcuts + backspace
			{0, 1, 2, 3, 4, "backspace"},
			// Row 2: URL special characters
			{5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
			// Row 3: qwertyuiop (QWERTY row)
			{15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
			// Row 4: asdfghjkl + enter (ASDF row)
			{25, 26, 27, 28, 29, 30, 31, 32, 33, "enter"},
			// Row 5: shift + zxcvbnm + symbol (ZXCV row) - no space for URLs
			{"shift", 34, 35, 36, 37, 38, 39, 40, "symbol"},
		},
	}
}

func createURLKeys() []key {
	keys := make([]key, 41)

	// URL shortcuts (keys 0-4)
	shortcuts := []string{"www.", ".com", ".org", ".net", ".io"}
	shortcutSymbols := []string{".co", ".tv", ".me", ".uk", ".gg"}
	for i, shortcut := range shortcuts {
		keys[i] = key{
			LowerValue:  shortcut,
			UpperValue:  shortcut,
			SymbolValue: shortcutSymbols[i],
		}
	}

	// URL special characters (keys 5-14)
	urlChars := []string{"/", ":", "@", "-", "_", ".", "~", "?", "#", "&"}
	urlSymbols := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for i, char := range urlChars {
		keys[5+i] = key{
			LowerValue:  char,
			UpperValue:  char,
			SymbolValue: urlSymbols[i],
		}
	}

	// QWERTY row (keys 15-24)
	qwerty := "qwertyuiop"
	for i, char := range qwerty {
		keys[15+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ASDF row (keys 25-33)
	asdf := "asdfghjkl"
	for i, char := range asdf {
		keys[25+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ZXCV row (keys 34-40)
	zxcv := "zxcvbnm"
	for i, char := range zxcv {
		keys[34+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	return keys
}

func createURLKeyLayoutFor10() *keyLayout {
	return &keyLayout{
		rows: [][]interface{}{
			// Row 1: URL shortcuts (5) + backspace
			{0, 1, 2, 3, 4, "backspace"},
			// Row 2: URL shortcuts (5)
			{5, 6, 7, 8, 9},
			// Row 3: URL special characters
			{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			// Row 4: qwertyuiop (QWERTY row)
			{20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
			// Row 5: asdfghjkl + enter (ASDF row)
			{30, 31, 32, 33, 34, 35, 36, 37, 38, "enter"},
			// Row 6: shift + zxcvbnm + symbol (ZXCV row)
			{"shift", 39, 40, 41, 42, 43, 44, 45, "symbol"},
		},
	}
}

func createURLKeyLayoutFor5() *keyLayout {
	return &keyLayout{
		rows: [][]interface{}{
			// Row 1: URL shortcuts (5) + backspace
			{0, 1, 2, 3, 4, "backspace"},
			// Row 2: URL special characters
			{5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
			// Row 3: qwertyuiop (QWERTY row)
			{15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
			// Row 4: asdfghjkl + enter (ASDF row)
			{25, 26, 27, 28, 29, 30, 31, 32, 33, "enter"},
			// Row 5: shift + zxcvbnm + symbol (ZXCV row)
			{"shift", 34, 35, 36, 37, 38, 39, 40, "symbol"},
		},
	}
}

func createURLKeysWithShortcuts5(shortcuts []URLShortcut) []key {
	keys := make([]key, 41)

	// URL shortcuts (keys 0-4)
	for i := 0; i < 5; i++ {
		if i < len(shortcuts) {
			keys[i] = key{
				LowerValue:  shortcuts[i].Value,
				UpperValue:  shortcuts[i].Value,
				SymbolValue: shortcuts[i].SymbolValue,
			}
		}
	}

	// URL special characters (keys 5-14)
	urlChars := []string{"/", ":", "@", "-", "_", ".", "~", "?", "#", "&"}
	urlSymbols := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for i, char := range urlChars {
		keys[5+i] = key{
			LowerValue:  char,
			UpperValue:  char,
			SymbolValue: urlSymbols[i],
		}
	}

	// QWERTY row (keys 15-24)
	qwerty := "qwertyuiop"
	for i, char := range qwerty {
		keys[15+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ASDF row (keys 25-33)
	asdf := "asdfghjkl"
	for i, char := range asdf {
		keys[25+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ZXCV row (keys 34-40)
	zxcv := "zxcvbnm"
	for i, char := range zxcv {
		keys[34+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	return keys
}

func createURLKeysWithShortcuts10(shortcuts []URLShortcut) []key {
	keys := make([]key, 46)

	// URL shortcuts (keys 0-9)
	for i, shortcut := range shortcuts {
		if i >= 10 {
			break
		}
		keys[i] = key{
			LowerValue:  shortcut.Value,
			UpperValue:  shortcut.Value,
			SymbolValue: shortcut.SymbolValue,
		}
	}

	// URL special characters (keys 10-19)
	urlChars := []string{"/", ":", "@", "-", "_", ".", "~", "?", "#", "&"}
	urlSymbols := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for i, char := range urlChars {
		keys[10+i] = key{
			LowerValue:  char,
			UpperValue:  char,
			SymbolValue: urlSymbols[i],
		}
	}

	// QWERTY row (keys 20-29)
	qwerty := "qwertyuiop"
	for i, char := range qwerty {
		keys[20+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ASDF row (keys 30-38)
	asdf := "asdfghjkl"
	for i, char := range asdf {
		keys[30+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	// ZXCV row (keys 39-45)
	zxcv := "zxcvbnm"
	for i, char := range zxcv {
		keys[39+i] = key{
			LowerValue:  string(char),
			UpperValue:  string(char - 32),
			SymbolValue: string(char),
		}
	}

	return keys
}

func createNumericKeyLayout() *keyLayout {
	return &keyLayout{
		rows: [][]interface{}{
			// Row 1: 7, 8, 9, backspace
			{6, 7, 8, "backspace"},
			// Row 2: 4, 5, 6, enter
			{3, 4, 5, "enter"},
			// Row 3: 1, 2, 3
			{0, 1, 2},
			// Row 4: 0 (spans full width visually)
			{9},
		},
	}
}

func createNumericKeys() []key {
	keys := make([]key, 10)

	// Keys 0-9 represent digits 1-9, 0
	digits := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	for i, digit := range digits {
		keys[i] = key{
			LowerValue:  digit,
			UpperValue:  digit,
			SymbolValue: digit,
		}
	}

	return keys
}

func setupKeyboardRects(kb *virtualKeyboard, windowWidth, windowHeight int32) {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	kb.KeyboardRect = sdl.Rect{X: startX, Y: keyboardStartY, W: keyboardWidth, H: keyboardHeight}
	kb.TextInputRect = sdl.Rect{X: startX, Y: textInputY, W: keyboardWidth, H: textInputHeight}

	keyWidth := keyboardWidth / 12
	keyHeight := keyboardHeight / 6
	keySpacing := int32(3)

	// Define consistent key widths for special keys
	backspaceWidth := keyWidth * 2
	shiftWidth := keyWidth * 2
	symbolWidth := keyWidth * 2
	enterWidth := keyWidth + keyWidth/2
	spaceWidth := keyWidth * 8

	// Calculate the maximum row width to determine a consistent left margin
	// Row 1: 10 regular keys + backspace
	row1Width := keyWidth*10 + keySpacing*9 + backspaceWidth + keySpacing
	// Row 2: 10 regular keys
	row2Width := keyWidth*10 + keySpacing*9
	// Row 3: 9 regular keys + enter
	row3Width := keyWidth*9 + keySpacing*8 + enterWidth + keySpacing
	// Row 4: shift + 7 regular keys + symbol
	row4Width := shiftWidth + keySpacing + keyWidth*7 + keySpacing*6 + symbolWidth + keySpacing
	// Row 5: space
	row5Width := spaceWidth

	// Find the maximum width to align all rows consistently
	maxRowWidth := row1Width
	if row2Width > maxRowWidth {
		maxRowWidth = row2Width
	}
	if row3Width > maxRowWidth {
		maxRowWidth = row3Width
	}
	if row4Width > maxRowWidth {
		maxRowWidth = row4Width
	}

	// Calculate a consistent left margin for all rows
	leftMargin := startX + (keyboardWidth-maxRowWidth)/2

	y := keyboardStartY + keySpacing

	// Row 1: Numbers + Backspace
	x := leftMargin
	for i := 0; i < 10; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.BackspaceRect = sdl.Rect{X: x, Y: y, W: backspaceWidth, H: keyHeight}

	// Row 2: QWERTY
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row2Width)/2 // Center this row within max width
	for i := 10; i < 20; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 3: ASDF + Enter
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row3Width)/2 // Center this row within max width
	for i := 20; i < 29; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.EnterRect = sdl.Rect{X: x, Y: y, W: enterWidth, H: keyHeight}

	// Row 4: Shift + ZXCV + Symbol
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row4Width)/2 // Center this row within max width

	kb.ShiftRect = sdl.Rect{X: x, Y: y, W: shiftWidth, H: keyHeight}
	x += shiftWidth + keySpacing

	for i := 29; i < 36; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	kb.SymbolRect = sdl.Rect{X: x, Y: y, W: symbolWidth, H: keyHeight}

	// Row 5: Space
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row5Width)/2 // Center space bar within max width
	kb.SpaceRect = sdl.Rect{X: x, Y: y, W: spaceWidth, H: keyHeight}
}

func setupURLKeyboardRects(kb *virtualKeyboard, windowWidth, windowHeight int32) {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	kb.KeyboardRect = sdl.Rect{X: startX, Y: keyboardStartY, W: keyboardWidth, H: keyboardHeight}
	kb.TextInputRect = sdl.Rect{X: startX, Y: textInputY, W: keyboardWidth, H: textInputHeight}

	keyWidth := keyboardWidth / 12
	keyHeight := keyboardHeight / 6
	keySpacing := int32(3)

	// Shortcut keys are wider to fit text like "www." and ".com"
	shortcutWidth := keyWidth * 2
	backspaceWidth := keyWidth * 2
	shiftWidth := keyWidth * 2
	symbolWidth := keyWidth * 2
	enterWidth := keyWidth + keyWidth/2

	// Row widths for URL layout (QWERTY-based)
	// Row 1: 5 shortcuts + backspace
	row1Width := shortcutWidth*5 + keySpacing*4 + backspaceWidth + keySpacing
	// Row 2: 10 URL special chars
	row2Width := keyWidth*10 + keySpacing*9
	// Row 3: 10 letters (qwertyuiop)
	row3Width := keyWidth*10 + keySpacing*9
	// Row 4: 9 letters (asdfghjkl) + enter
	row4Width := keyWidth*9 + keySpacing*8 + enterWidth + keySpacing
	// Row 5: shift + 7 letters (zxcvbnm) + symbol (no space for URLs)
	row5Width := shiftWidth + keySpacing + keyWidth*7 + keySpacing*6 + symbolWidth + keySpacing

	// Find the maximum width
	maxRowWidth := row1Width
	if row2Width > maxRowWidth {
		maxRowWidth = row2Width
	}
	if row3Width > maxRowWidth {
		maxRowWidth = row3Width
	}
	if row4Width > maxRowWidth {
		maxRowWidth = row4Width
	}
	if row5Width > maxRowWidth {
		maxRowWidth = row5Width
	}

	leftMargin := startX + (keyboardWidth-maxRowWidth)/2
	y := keyboardStartY + keySpacing

	// Row 1: URL shortcuts + Backspace
	x := leftMargin + (maxRowWidth-row1Width)/2
	for i := 0; i < 5; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: shortcutWidth, H: keyHeight}
		x += shortcutWidth + keySpacing
	}
	kb.BackspaceRect = sdl.Rect{X: x, Y: y, W: backspaceWidth, H: keyHeight}

	// Row 2: URL special characters
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row2Width)/2
	for i := 5; i < 15; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 3: QWERTY row (qwertyuiop)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row3Width)/2
	for i := 15; i < 25; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 4: ASDF row (asdfghjkl) + Enter
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row4Width)/2
	for i := 25; i < 34; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.EnterRect = sdl.Rect{X: x, Y: y, W: enterWidth, H: keyHeight}

	// Row 5: Shift + ZXCV row (zxcvbnm) + Symbol (no space for URLs)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row5Width)/2
	kb.ShiftRect = sdl.Rect{X: x, Y: y, W: shiftWidth, H: keyHeight}
	x += shiftWidth + keySpacing
	for i := 34; i < 41; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.SymbolRect = sdl.Rect{X: x, Y: y, W: symbolWidth, H: keyHeight}

	// No space key for URL layout
	kb.SpaceRect = sdl.Rect{}
}

func setupURLKeyboardRectsFor5(kb *virtualKeyboard, windowWidth, windowHeight int32) {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	kb.KeyboardRect = sdl.Rect{X: startX, Y: keyboardStartY, W: keyboardWidth, H: keyboardHeight}
	kb.TextInputRect = sdl.Rect{X: startX, Y: textInputY, W: keyboardWidth, H: textInputHeight}

	keyWidth := keyboardWidth / 12
	keyHeight := keyboardHeight / 6
	keySpacing := int32(3)

	// Shortcut keys are wider to fit text like "https://" and ".com"
	shortcutWidth := keyWidth * 2
	backspaceWidth := keyWidth * 2
	shiftWidth := keyWidth * 2
	symbolWidth := keyWidth * 2
	enterWidth := keyWidth + keyWidth/2

	// Row widths for URL layout with 5 shortcuts
	// Row 1: 5 shortcuts + backspace
	row1Width := shortcutWidth*5 + keySpacing*4 + backspaceWidth + keySpacing
	// Row 2: 10 URL special chars
	row2Width := keyWidth*10 + keySpacing*9
	// Row 3: 10 letters (qwertyuiop)
	row3Width := keyWidth*10 + keySpacing*9
	// Row 4: 9 letters (asdfghjkl) + enter
	row4Width := keyWidth*9 + keySpacing*8 + enterWidth + keySpacing
	// Row 5: shift + 7 letters (zxcvbnm) + symbol
	row5Width := shiftWidth + keySpacing + keyWidth*7 + keySpacing*6 + symbolWidth + keySpacing

	// Find the maximum width
	maxRowWidth := row1Width
	if row2Width > maxRowWidth {
		maxRowWidth = row2Width
	}
	if row3Width > maxRowWidth {
		maxRowWidth = row3Width
	}
	if row4Width > maxRowWidth {
		maxRowWidth = row4Width
	}
	if row5Width > maxRowWidth {
		maxRowWidth = row5Width
	}

	leftMargin := startX + (keyboardWidth-maxRowWidth)/2
	y := keyboardStartY + keySpacing

	// Row 1: URL shortcuts (0-4) + Backspace
	x := leftMargin + (maxRowWidth-row1Width)/2
	for i := 0; i < 5; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: shortcutWidth, H: keyHeight}
		x += shortcutWidth + keySpacing
	}
	kb.BackspaceRect = sdl.Rect{X: x, Y: y, W: backspaceWidth, H: keyHeight}

	// Row 2: URL special characters (5-14)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row2Width)/2
	for i := 5; i < 15; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 3: QWERTY row (15-24)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row3Width)/2
	for i := 15; i < 25; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 4: ASDF row (25-33) + Enter
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row4Width)/2
	for i := 25; i < 34; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.EnterRect = sdl.Rect{X: x, Y: y, W: enterWidth, H: keyHeight}

	// Row 5: Shift + ZXCV row (34-40) + Symbol
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row5Width)/2
	kb.ShiftRect = sdl.Rect{X: x, Y: y, W: shiftWidth, H: keyHeight}
	x += shiftWidth + keySpacing
	for i := 34; i < 41; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.SymbolRect = sdl.Rect{X: x, Y: y, W: symbolWidth, H: keyHeight}

	// No space key for URL layout
	kb.SpaceRect = sdl.Rect{}
}

func setupURLKeyboardRectsFor10(kb *virtualKeyboard, windowWidth, windowHeight int32) {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	kb.KeyboardRect = sdl.Rect{X: startX, Y: keyboardStartY, W: keyboardWidth, H: keyboardHeight}
	kb.TextInputRect = sdl.Rect{X: startX, Y: textInputY, W: keyboardWidth, H: textInputHeight}

	keyWidth := keyboardWidth / 12
	keyHeight := keyboardHeight / 7 // 6 rows + some padding
	keySpacing := int32(3)

	// Shortcut keys are wider to fit text like "https://" and ".com"
	shortcutWidth := keyWidth * 2
	backspaceWidth := keyWidth * 2
	shiftWidth := keyWidth * 2
	symbolWidth := keyWidth * 2
	enterWidth := keyWidth + keyWidth/2

	// Row widths for URL layout with 10 shortcuts
	// Row 1: 5 shortcuts + backspace
	row1Width := shortcutWidth*5 + keySpacing*4 + backspaceWidth + keySpacing
	// Row 2: 5 shortcuts (no backspace)
	row2Width := shortcutWidth*5 + keySpacing*4
	// Row 3: 10 URL special chars
	row3Width := keyWidth*10 + keySpacing*9
	// Row 4: 10 letters (qwertyuiop)
	row4Width := keyWidth*10 + keySpacing*9
	// Row 5: 9 letters (asdfghjkl) + enter
	row5Width := keyWidth*9 + keySpacing*8 + enterWidth + keySpacing
	// Row 6: shift + 7 letters (zxcvbnm) + symbol
	row6Width := shiftWidth + keySpacing + keyWidth*7 + keySpacing*6 + symbolWidth + keySpacing

	// Find the maximum width
	maxRowWidth := row1Width
	if row2Width > maxRowWidth {
		maxRowWidth = row2Width
	}
	if row3Width > maxRowWidth {
		maxRowWidth = row3Width
	}
	if row4Width > maxRowWidth {
		maxRowWidth = row4Width
	}
	if row5Width > maxRowWidth {
		maxRowWidth = row5Width
	}
	if row6Width > maxRowWidth {
		maxRowWidth = row6Width
	}

	leftMargin := startX + (keyboardWidth-maxRowWidth)/2
	y := keyboardStartY + keySpacing

	// Row 1: URL shortcuts (0-4) + Backspace
	x := leftMargin + (maxRowWidth-row1Width)/2
	for i := 0; i < 5; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: shortcutWidth, H: keyHeight}
		x += shortcutWidth + keySpacing
	}
	kb.BackspaceRect = sdl.Rect{X: x, Y: y, W: backspaceWidth, H: keyHeight}

	// Row 2: URL shortcuts (5-9)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row2Width)/2
	for i := 5; i < 10; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: shortcutWidth, H: keyHeight}
		x += shortcutWidth + keySpacing
	}

	// Row 3: URL special characters (10-19)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row3Width)/2
	for i := 10; i < 20; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 4: QWERTY row (20-29)
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row4Width)/2
	for i := 20; i < 30; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}

	// Row 5: ASDF row (30-38) + Enter
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row5Width)/2
	for i := 30; i < 39; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.EnterRect = sdl.Rect{X: x, Y: y, W: enterWidth, H: keyHeight}

	// Row 6: Shift + ZXCV row (39-45) + Symbol
	y += keyHeight + keySpacing
	x = leftMargin + (maxRowWidth-row6Width)/2
	kb.ShiftRect = sdl.Rect{X: x, Y: y, W: shiftWidth, H: keyHeight}
	x += shiftWidth + keySpacing
	for i := 39; i < 46; i++ {
		kb.Keys[i].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight}
		x += keyWidth + keySpacing
	}
	kb.SymbolRect = sdl.Rect{X: x, Y: y, W: symbolWidth, H: keyHeight}

	// No space key for URL layout
	kb.SpaceRect = sdl.Rect{}
}

func setupNumericKeyboardRects(kb *virtualKeyboard, windowWidth, windowHeight int32) {
	keyboardWidth := (windowWidth * 85) / 100
	keyboardHeight := (windowHeight * 85) / 100
	textInputHeight := windowHeight / 10
	keyboardHeight = keyboardHeight - textInputHeight - 20
	startX := (windowWidth - keyboardWidth) / 2
	textInputY := (windowHeight - keyboardHeight - textInputHeight - 20) / 2
	keyboardStartY := textInputY + textInputHeight + 20

	kb.KeyboardRect = sdl.Rect{X: startX, Y: keyboardStartY, W: keyboardWidth, H: keyboardHeight}
	kb.TextInputRect = sdl.Rect{X: startX, Y: textInputY, W: keyboardWidth, H: textInputHeight}

	// Numeric pad uses larger keys since there are fewer
	keyWidth := keyboardWidth / 5
	keyHeight := keyboardHeight / 5
	keySpacing := int32(5)

	backspaceWidth := keyWidth
	enterWidth := keyWidth

	// Calculate grid width (3 digit keys + 1 action key per row)
	gridWidth := keyWidth*3 + keySpacing*2 + backspaceWidth + keySpacing

	// Center the grid
	leftMargin := startX + (keyboardWidth-gridWidth)/2
	y := keyboardStartY + keySpacing

	// Row 1: 7, 8, 9, Backspace
	x := leftMargin
	kb.Keys[6].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 7
	x += keyWidth + keySpacing
	kb.Keys[7].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 8
	x += keyWidth + keySpacing
	kb.Keys[8].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 9
	x += keyWidth + keySpacing
	kb.BackspaceRect = sdl.Rect{X: x, Y: y, W: backspaceWidth, H: keyHeight}

	// Row 2: 4, 5, 6, Enter
	y += keyHeight + keySpacing
	x = leftMargin
	kb.Keys[3].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 4
	x += keyWidth + keySpacing
	kb.Keys[4].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 5
	x += keyWidth + keySpacing
	kb.Keys[5].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 6
	x += keyWidth + keySpacing
	kb.EnterRect = sdl.Rect{X: x, Y: y, W: enterWidth, H: keyHeight}

	// Row 3: 1, 2, 3
	y += keyHeight + keySpacing
	x = leftMargin
	kb.Keys[0].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 1
	x += keyWidth + keySpacing
	kb.Keys[1].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 2
	x += keyWidth + keySpacing
	kb.Keys[2].Rect = sdl.Rect{X: x, Y: y, W: keyWidth, H: keyHeight} // 3

	// Row 4: 0 (spans width of 3 keys)
	y += keyHeight + keySpacing
	x = leftMargin
	zeroWidth := keyWidth*3 + keySpacing*2
	kb.Keys[9].Rect = sdl.Rect{X: x, Y: y, W: zeroWidth, H: keyHeight} // 0

	// Initialize unused rects to zero (shift, symbol, space not used in numeric mode)
	kb.ShiftRect = sdl.Rect{}
	kb.SymbolRect = sdl.Rect{}
	kb.SpaceRect = sdl.Rect{}
}

// KeyboardResult represents the result of the Keyboard component.
type KeyboardResult struct {
	Text string
}

// Keyboard displays a virtual keyboard for text input.
// An optional layout parameter can be provided to use a specific keyboard layout.
// If no layout is specified, KeyboardLayoutGeneral is used.
// Returns ErrCancelled if the user exits without pressing Enter.
func Keyboard(initialText string, helpExitText string, layout ...KeyboardLayout) (*KeyboardResult, error) {
	selectedLayout := KeyboardLayoutGeneral
	if len(layout) > 0 {
		selectedLayout = layout[0]
	}

	window := internal.GetWindow()
	renderer := window.Renderer
	font := internal.Fonts.MediumFont

	kb := createKeyboard(window.GetWidth(), window.GetHeight(), helpExitText, selectedLayout)
	if initialText != "" {
		kb.TextBuffer = initialText
		kb.CursorPosition = len(initialText)
	}

	for {
		if kb.handleEvents() {
			break
		}

		kb.handleDirectionalRepeats()

		kb.updateCursorBlink()
		kb.render(renderer, font)
		sdl.Delay(16)
	}

	if kb.EnterPressed {
		return &KeyboardResult{Text: kb.TextBuffer}, nil
	}
	return nil, ErrCancelled
}

// URLKeyboard displays a URL-optimized keyboard with customizable shortcuts.
// If 1-5 shortcuts are provided, a single row of shortcuts is shown.
// If 6-10 shortcuts are provided, two rows of shortcuts are shown.
// If no config is provided, 10 default shortcuts are used (two rows).
// Returns ErrCancelled if the user exits without pressing Enter.
func URLKeyboard(initialText string, helpExitText string, config ...URLKeyboardConfig) (*KeyboardResult, error) {
	// Build shortcuts list - use provided shortcuts or defaults
	var shortcuts []URLShortcut
	if len(config) > 0 && len(config[0].Shortcuts) > 0 {
		// Use only the provided shortcuts (up to 10)
		maxShortcuts := len(config[0].Shortcuts)
		if maxShortcuts > 10 {
			maxShortcuts = 10
		}
		shortcuts = config[0].Shortcuts[:maxShortcuts]
	} else {
		// No config provided, use all 10 defaults
		shortcuts = defaultURLShortcuts
	}

	window := internal.GetWindow()
	renderer := window.Renderer
	font := internal.Fonts.MediumFont

	kb := createURLKeyboard(window.GetWidth(), window.GetHeight(), helpExitText, shortcuts)
	if initialText != "" {
		kb.TextBuffer = initialText
		kb.CursorPosition = len(initialText)
	}

	for {
		if kb.handleEvents() {
			break
		}

		kb.handleDirectionalRepeats()

		kb.updateCursorBlink()
		kb.render(renderer, font)
		sdl.Delay(16)
	}

	if kb.EnterPressed {
		return &KeyboardResult{Text: kb.TextBuffer}, nil
	}
	return nil, ErrCancelled
}

func (kb *virtualKeyboard) handleEvents() bool {
	processor := internal.GetInputProcessor()

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			return true

		case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
			inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
			if inputEvent == nil {
				continue
			}

			if inputEvent.Pressed {
				if kb.handleInputEvent(inputEvent) {
					return true
				}
			} else {
				kb.handleInputEventRelease(inputEvent)
			}
		}
	}
	return false
}

func (kb *virtualKeyboard) handleInputEvent(inputEvent *internal.Event) bool {
	// Rate limit navigation to prevent too-fast input
	if kb.isDirectionalButton(inputEvent.Button) {
		if time.Since(kb.lastInputTime) < kb.InputDelay {
			return false
		}
		kb.lastInputTime = time.Now()
	}

	button := inputEvent.Button

	// Help toggle - always available
	if button == constants.VirtualButtonMenu {
		kb.toggleHelp()
		return false
	}

	// If help is showing, handle help-specific input
	if kb.ShowingHelp {
		return kb.handleHelpInputEvent(button)
	}

	// Handle keyboard input
	switch button {
	case constants.VirtualButtonUp:
		kb.navigate(button)
		kb.heldDirections.up = true
		kb.heldDirections.down = false
		kb.lastRepeatTime = time.Now()
		return false
	case constants.VirtualButtonDown:
		kb.navigate(button)
		kb.heldDirections.down = true
		kb.heldDirections.up = false
		kb.lastRepeatTime = time.Now()
		return false
	case constants.VirtualButtonLeft:
		kb.navigate(button)
		kb.heldDirections.left = true
		kb.heldDirections.right = false
		kb.lastRepeatTime = time.Now()
		return false
	case constants.VirtualButtonRight:
		kb.navigate(button)
		kb.heldDirections.right = true
		kb.heldDirections.left = false
		kb.lastRepeatTime = time.Now()
		return false
	case constants.VirtualButtonA:
		kb.processSelection()
		return kb.EnterPressed
	case constants.VirtualButtonB:
		kb.backspace()
		return false
	case constants.VirtualButtonX:
		if kb.Layout == KeyboardLayoutGeneral {
			kb.insertSpace()
		} else if kb.Layout == KeyboardLayoutURL {
			kb.toggleSymbols()
		}
		return false
	case constants.VirtualButtonSelect:
		// No shift in numeric layout
		if kb.Layout != KeyboardLayoutNumeric {
			kb.toggleShift()
		}
		return false
	case constants.VirtualButtonY:
		return true // Exit without saving
	case constants.VirtualButtonStart:
		kb.EnterPressed = true
		return true // Exit and save
	case constants.VirtualButtonL1:
		kb.moveCursor(-1)
		return false
	case constants.VirtualButtonR1:
		kb.moveCursor(1)
		return false
	}

	return false
}

func (kb *virtualKeyboard) isDirectionalButton(button constants.VirtualButton) bool {
	return button == constants.VirtualButtonUp || button == constants.VirtualButtonDown ||
		button == constants.VirtualButtonLeft || button == constants.VirtualButtonRight
}

func (kb *virtualKeyboard) handleHelpInputEvent(button constants.VirtualButton) bool {
	switch button {
	case constants.VirtualButtonUp:
		kb.scrollHelpOverlay(-1)
		return false
	case constants.VirtualButtonDown:
		kb.scrollHelpOverlay(1)
		return false
	default:
		kb.ShowingHelp = false
		return false
	}
}

func (kb *virtualKeyboard) handleInputEventRelease(inputEvent *internal.Event) {
	switch inputEvent.Button {
	case constants.VirtualButtonUp:
		kb.heldDirections.up = false
		kb.hasRepeated = false
	case constants.VirtualButtonDown:
		kb.heldDirections.down = false
		kb.hasRepeated = false
	case constants.VirtualButtonLeft:
		kb.heldDirections.left = false
		kb.hasRepeated = false
	case constants.VirtualButtonRight:
		kb.heldDirections.right = false
		kb.hasRepeated = false
	}
}

func (kb *virtualKeyboard) handleDirectionalRepeats() {
	if !kb.heldDirections.up && !kb.heldDirections.down && !kb.heldDirections.left && !kb.heldDirections.right {
		kb.lastRepeatTime = time.Now()
		kb.hasRepeated = false
		return
	}

	timeSince := time.Since(kb.lastRepeatTime)

	// Use repeatDelay for first repeat, then repeatInterval for subsequent repeats
	threshold := kb.repeatInterval
	if !kb.hasRepeated {
		threshold = kb.repeatDelay
	}

	if timeSince >= threshold {
		kb.lastRepeatTime = time.Now()
		kb.hasRepeated = true

		if kb.heldDirections.up {
			kb.navigate(constants.VirtualButtonUp)
		} else if kb.heldDirections.down {
			kb.navigate(constants.VirtualButtonDown)
		} else if kb.heldDirections.left {
			kb.navigate(constants.VirtualButtonLeft)
		} else if kb.heldDirections.right {
			kb.navigate(constants.VirtualButtonRight)
		}
	}
}

func (kb *virtualKeyboard) navigate(button constants.VirtualButton) {
	layout := kb.keyLayout
	currentRow, currentCol := kb.findCurrentPosition(layout)

	var newRow, newCol int
	switch button {
	case constants.VirtualButtonUp:
		newRow, newCol = kb.moveUp(layout, currentRow, currentCol)
	case constants.VirtualButtonDown:
		newRow, newCol = kb.moveDown(layout, currentRow, currentCol)
	case constants.VirtualButtonLeft:
		newRow, newCol = kb.moveLeft(layout, currentRow, currentCol)
	case constants.VirtualButtonRight:
		newRow, newCol = kb.moveRight(layout, currentRow, currentCol)
	}

	kb.setSelection(layout, newRow, newCol)
}

func (kb *virtualKeyboard) findCurrentPosition(layout *keyLayout) (int, int) {
	specialKeys := map[int]string{1: "backspace", 2: "enter", 3: "space", 4: "shift", 5: "symbol"}

	if kb.SelectedSpecial > 0 {
		targetKey := specialKeys[kb.SelectedSpecial]
		for r, row := range layout.rows {
			for c, key := range row {
				if str, ok := key.(string); ok && str == targetKey {
					return r, c
				}
			}
		}
	}

	for r, row := range layout.rows {
		for c, key := range row {
			if idx, ok := key.(int); ok && idx == kb.SelectedKeyIndex {
				return r, c
			}
		}
	}

	return 0, 0
}

func (kb *virtualKeyboard) moveUp(layout *keyLayout, row, col int) (int, int) {
	newRow := row - 1
	if newRow < 0 {
		newRow = len(layout.rows) - 1
	}
	if col >= len(layout.rows[newRow]) {
		col = len(layout.rows[newRow]) - 1
	}
	return newRow, col
}

func (kb *virtualKeyboard) moveDown(layout *keyLayout, row, col int) (int, int) {
	newRow := row + 1
	if newRow >= len(layout.rows) {
		newRow = 0
	}
	if col >= len(layout.rows[newRow]) {
		col = len(layout.rows[newRow]) - 1
	}
	return newRow, col
}

func (kb *virtualKeyboard) moveLeft(layout *keyLayout, row, col int) (int, int) {
	newCol := col - 1
	if newCol < 0 {
		newCol = len(layout.rows[row]) - 1
	}
	return row, newCol
}

func (kb *virtualKeyboard) moveRight(layout *keyLayout, row, col int) (int, int) {
	newCol := col + 1
	if newCol >= len(layout.rows[row]) {
		newCol = 0
	}
	return row, newCol
}

func (kb *virtualKeyboard) setSelection(layout *keyLayout, row, col int) {
	kb.resetPressedKeys()

	selectedKey := layout.rows[row][col]
	if idx, ok := selectedKey.(int); ok {
		kb.SelectedKeyIndex = idx
		kb.SelectedSpecial = 0
		kb.Keys[kb.SelectedKeyIndex].IsPressed = true
	} else if str, ok := selectedKey.(string); ok {
		kb.SelectedKeyIndex = -1
		specialMap := map[string]int{"backspace": 1, "enter": 2, "space": 3, "shift": 4, "symbol": 5}
		kb.SelectedSpecial = specialMap[str]
	}
}

func (kb *virtualKeyboard) processSelection() {
	if kb.SelectedKeyIndex >= 0 && kb.SelectedKeyIndex < len(kb.Keys) {
		keyValue := kb.getKeyValue(kb.SelectedKeyIndex)
		kb.insertText(keyValue)
	} else {
		kb.handleSpecialKey()
	}

	kb.CursorVisible = true
	kb.LastCursorBlink = time.Now()
}

func (kb *virtualKeyboard) getKeyValue(index int) string {
	key := kb.Keys[index]
	if kb.CurrentState == symbolsMode {
		return key.SymbolValue
	} else if kb.Layout == KeyboardLayoutGeneral && index < 10 && kb.ShiftPressed {
		return key.SymbolValue
	} else if kb.CurrentState == upperCase {
		return key.UpperValue
	}
	return key.LowerValue
}

func (kb *virtualKeyboard) insertText(text string) {
	if kb.CursorPosition == len(kb.TextBuffer) {
		kb.TextBuffer += text
	} else {
		textRunes := []rune(kb.TextBuffer)
		before := string(textRunes[:kb.CursorPosition])
		after := string(textRunes[kb.CursorPosition:])
		kb.TextBuffer = before + text + after
	}
	kb.CursorPosition += len([]rune(text))
}

func (kb *virtualKeyboard) handleSpecialKey() {
	switch kb.SelectedSpecial {
	case 1: // backspace
		kb.backspace()
	case 2: // enter
		kb.EnterPressed = true
	case 3: // space
		kb.insertSpace()
	case 4: // shift
		kb.toggleShift()
	case 5: // symbol
		kb.toggleSymbols()
	}
}

func (kb *virtualKeyboard) backspace() {
	if kb.CursorPosition > 0 {
		textRunes := []rune(kb.TextBuffer)
		before := string(textRunes[:kb.CursorPosition-1])
		after := string(textRunes[kb.CursorPosition:])
		kb.TextBuffer = before + after
		kb.CursorPosition--
	}
}

func (kb *virtualKeyboard) insertSpace() {
	kb.insertText(" ")
}

func (kb *virtualKeyboard) toggleShift() {
	if kb.CurrentState == symbolsMode {
		// If in symbols mode, shift just toggles the shift flag
		kb.ShiftPressed = !kb.ShiftPressed
	} else {
		// Normal shift behavior for upper/lower case
		kb.ShiftPressed = !kb.ShiftPressed
		if kb.ShiftPressed {
			kb.CurrentState = upperCase
		} else {
			kb.CurrentState = lowerCase
		}
	}
}

func (kb *virtualKeyboard) toggleSymbols() {
	kb.SymbolPressed = !kb.SymbolPressed
	if kb.SymbolPressed {
		kb.CurrentState = symbolsMode
	} else {
		if kb.ShiftPressed {
			kb.CurrentState = upperCase
		} else {
			kb.CurrentState = lowerCase
		}
	}
}

func (kb *virtualKeyboard) moveCursor(direction int) {
	if direction > 0 && kb.CursorPosition < len(kb.TextBuffer) {
		kb.CursorPosition++
	} else if direction < 0 && kb.CursorPosition > 0 {
		kb.CursorPosition--
	}

	kb.CursorVisible = true
	kb.LastCursorBlink = time.Now()
}

func (kb *virtualKeyboard) updateCursorBlink() {
	if time.Since(kb.LastCursorBlink) > kb.CursorBlinkRate {
		kb.CursorVisible = !kb.CursorVisible
		kb.LastCursorBlink = time.Now()
	}
}

func (kb *virtualKeyboard) resetPressedKeys() {
	for i := range kb.Keys {
		kb.Keys[i].IsPressed = false
	}
}

func (kb *virtualKeyboard) toggleHelp() {
	if kb.helpOverlay == nil {
		kb.helpOverlay = newHelpOverlay("Keyboard Help", defaultKeyboardHelpLines, kb.helpExitText)
	}
	kb.helpOverlay.toggle()
	kb.ShowingHelp = kb.helpOverlay.ShowingHelp
}

func (kb *virtualKeyboard) scrollHelpOverlay(direction int) {
	if kb.helpOverlay != nil {
		kb.helpOverlay.scroll(direction)
	}
}

func (kb *virtualKeyboard) render(renderer *sdl.Renderer, font *ttf.Font) {
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	window := internal.GetWindow()

	if window.Background != nil {
		window.RenderBackground()
	} else {
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
	}

	if !kb.ShowingHelp {
		kb.renderTextInput(renderer, font)
		kb.renderKeys(renderer, font)
		kb.renderSpecialKeys(renderer)
		kb.renderFooter(renderer)
	}

	if kb.ShowingHelp && kb.helpOverlay != nil {
		kb.helpOverlay.render(renderer, internal.Fonts.SmallFont)
	}

	renderer.Present()
}

func (kb *virtualKeyboard) renderTextInput(renderer *sdl.Renderer, font *ttf.Font) {
	renderer.SetDrawColor(50, 50, 50, 255)
	renderer.FillRect(&kb.TextInputRect)
	renderer.SetDrawColor(200, 200, 200, 255)
	renderer.DrawRect(&kb.TextInputRect)

	padding := int32(10)
	if kb.TextBuffer != "" {
		kb.renderTextWithCursor(renderer, font, padding)
	} else if kb.CursorVisible {
		kb.renderEmptyCursor(renderer, font, padding)
	}
}

func (kb *virtualKeyboard) renderTextWithCursor(renderer *sdl.Renderer, font *ttf.Font, padding int32) {
	textColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	textSurface, err := font.RenderUTF8Blended(kb.TextBuffer, textColor)
	if err != nil {
		return
	}
	defer textSurface.Free()

	textTexture, err := renderer.CreateTextureFromSurface(textSurface)
	if err != nil {
		return
	}
	defer textTexture.Destroy()

	// Calculate cursor position and scrolling
	cursorX := kb.calculateCursorX(font)
	visibleWidth := kb.TextInputRect.W - (padding * 2)
	offsetX := kb.calculateScrollOffset(cursorX, visibleWidth, textSurface.W, padding)

	// Render text
	srcRect := &sdl.Rect{X: offsetX, Y: 0, W: visibleWidth, H: textSurface.H}
	if textSurface.W < visibleWidth {
		srcRect.W = textSurface.W
	}

	textRect := sdl.Rect{
		X: kb.TextInputRect.X + padding,
		Y: kb.TextInputRect.Y + (kb.TextInputRect.H-textSurface.H)/2,
		W: srcRect.W,
		H: textSurface.H,
	}
	renderer.Copy(textTexture, srcRect, &textRect)

	// Render cursor
	if kb.CursorVisible {
		cursorRect := sdl.Rect{
			X: kb.TextInputRect.X + padding + cursorX - offsetX,
			Y: textRect.Y,
			W: 2,
			H: textSurface.H,
		}
		if cursorRect.X >= kb.TextInputRect.X+padding && cursorRect.X <= kb.TextInputRect.X+padding+visibleWidth {
			renderer.SetDrawColor(255, 255, 255, 255)
			renderer.FillRect(&cursorRect)
		}
	}
}

func (kb *virtualKeyboard) renderEmptyCursor(renderer *sdl.Renderer, font *ttf.Font, padding int32) {
	fontHeight := font.Height()

	cursorRect := sdl.Rect{
		X: kb.TextInputRect.X + padding,
		Y: kb.TextInputRect.Y + (kb.TextInputRect.H - int32(fontHeight)),
		W: 2,
		H: int32(fontHeight),
	}
	renderer.SetDrawColor(255, 255, 255, 255)
	renderer.FillRect(&cursorRect)
}

func (kb *virtualKeyboard) calculateCursorX(font *ttf.Font) int32 {
	if kb.CursorPosition == 0 {
		return 0
	}

	cursorText := kb.TextBuffer[:kb.CursorPosition]
	textColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	cursorSurface, err := font.RenderUTF8Blended(cursorText, textColor)
	if err != nil {
		return 0
	}
	defer cursorSurface.Free()

	return cursorSurface.W
}

func (kb *virtualKeyboard) calculateScrollOffset(cursorX, visibleWidth, textWidth, padding int32) int32 {
	offsetX := int32(0)
	if cursorX > visibleWidth {
		offsetX = cursorX - visibleWidth + padding
	}

	maxOffset := textWidth - visibleWidth
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offsetX > maxOffset {
		offsetX = maxOffset
	}

	return offsetX
}

func (kb *virtualKeyboard) renderKeys(renderer *sdl.Renderer, font *ttf.Font) {
	for i, key := range kb.Keys {
		kb.renderSingleKey(renderer, font, i, key)
	}
}

func (kb *virtualKeyboard) renderSingleKey(renderer *sdl.Renderer, font *ttf.Font, index int, key key) {
	bgColor := sdl.Color{R: 50, G: 50, B: 60, A: 255}
	if index == kb.SelectedKeyIndex {
		bgColor = sdl.Color{R: 100, G: 100, B: 240, A: 255}
	} else if key.IsPressed {
		bgColor = sdl.Color{R: 80, G: 80, B: 120, A: 255}
	}

	renderer.SetDrawColor(bgColor.R, bgColor.G, bgColor.B, bgColor.A)
	renderer.FillRect(&key.Rect)
	renderer.SetDrawColor(70, 70, 80, 255)
	renderer.DrawRect(&key.Rect)

	keyValue := kb.getKeyValue(index)
	kb.renderKeyText(renderer, font, keyValue, key.Rect)
}

func (kb *virtualKeyboard) renderKeyText(renderer *sdl.Renderer, font *ttf.Font, text string, rect sdl.Rect) {
	textColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	textSurface, err := font.RenderUTF8Blended(text, textColor)
	if err != nil {
		return
	}
	defer textSurface.Free()

	textTexture, err := renderer.CreateTextureFromSurface(textSurface)
	if err != nil {
		return
	}
	defer textTexture.Destroy()

	textRect := sdl.Rect{
		X: rect.X + (rect.W-textSurface.W)/2,
		Y: rect.Y + (rect.H-textSurface.H)/2,
		W: textSurface.W,
		H: textSurface.H,
	}
	renderer.Copy(textTexture, nil, &textRect)
}

func (kb *virtualKeyboard) renderSpecialKeys(renderer *sdl.Renderer) {
	kb.renderSpecialKey(renderer, kb.BackspaceRect, "←", kb.SelectedSpecial == 1)
	kb.renderSpecialKey(renderer, kb.EnterRect, "↵", kb.SelectedSpecial == 2)

	// Numeric layout only has backspace and enter
	if kb.Layout == KeyboardLayoutNumeric {
		return
	}

	kb.renderSpecialKey(renderer, kb.ShiftRect, "⇧", kb.SelectedSpecial == 4 || kb.CurrentState == upperCase)
	kb.renderSpecialKey(renderer, kb.SymbolRect, "sym", kb.SelectedSpecial == 5 || kb.CurrentState == symbolsMode)

	// URL layout has no space key
	if kb.Layout != KeyboardLayoutURL {
		kb.renderSpaceKey(renderer)
	}
}

func (kb *virtualKeyboard) renderSpecialKey(renderer *sdl.Renderer, rect sdl.Rect, symbol string, isSelected bool) {
	bgColor := sdl.Color{R: 50, G: 50, B: 60, A: 255}
	if isSelected {
		bgColor = sdl.Color{R: 100, G: 100, B: 240, A: 255}
	}

	renderer.SetDrawColor(bgColor.R, bgColor.G, bgColor.B, bgColor.A)
	renderer.FillRect(&rect)
	renderer.SetDrawColor(70, 70, 80, 255)
	renderer.DrawRect(&rect)

	textColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	textSurface, err := internal.Fonts.LargeSymbolFont.RenderUTF8Blended(symbol, textColor)
	if err != nil {
		return
	}
	defer textSurface.Free()

	textTexture, err := renderer.CreateTextureFromSurface(textSurface)
	if err != nil {
		return
	}
	defer textTexture.Destroy()

	textRect := sdl.Rect{
		X: rect.X + (rect.W-textSurface.W)/2,
		Y: rect.Y + (rect.H-textSurface.H)/2,
		W: textSurface.W,
		H: textSurface.H,
	}
	renderer.Copy(textTexture, nil, &textRect)
}

func (kb *virtualKeyboard) renderSpaceKey(renderer *sdl.Renderer) {
	bgColor := sdl.Color{R: 50, G: 50, B: 60, A: 255}
	if kb.SelectedSpecial == 3 {
		bgColor = sdl.Color{R: 100, G: 100, B: 240, A: 255}
	}

	renderer.SetDrawColor(bgColor.R, bgColor.G, bgColor.B, bgColor.A)
	renderer.FillRect(&kb.SpaceRect)
	renderer.SetDrawColor(70, 70, 80, 255)
	renderer.DrawRect(&kb.SpaceRect)

	// draw space bar indicator
	lineWidth := kb.SpaceRect.W / 3
	lineHeight := int32(4)
	lineRect := sdl.Rect{
		X: kb.SpaceRect.X + (kb.SpaceRect.W-lineWidth)/2,
		Y: kb.SpaceRect.Y + (kb.SpaceRect.H-lineHeight)/2,
		W: lineWidth,
		H: lineHeight,
	}
	renderer.SetDrawColor(255, 255, 255, 255)
	renderer.FillRect(&lineRect)
}

func (kb *virtualKeyboard) renderFooter(renderer *sdl.Renderer) {
	renderFooter(
		renderer,
		internal.Fonts.SmallFont,
		[]FooterHelpItem{
			{ButtonName: "Menu", HelpText: "Help"},
		},
		20,
		true,
		true,
	)
}

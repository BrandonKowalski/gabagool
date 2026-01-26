package internal

import (
	"github.com/veandco/go-sdl2/sdl"
)

// Theme defines the visual appearance of the UI framework.
// Colors are typically loaded from CFW theme files (NextUI, Cannoli).
type Theme struct {
	HighlightColor       sdl.Color // Selected item background, footer button background
	AccentColor          sdl.Color // Pill backgrounds, status bar pill
	ButtonLabelColor     sdl.Color // Button label text (inside pills)
	TextColor            sdl.Color // Default text color
	HighlightedTextColor sdl.Color // Text on highlighted items
	HintColor            sdl.Color // Help text, status bar text
	BackgroundColor      sdl.Color // Screen background color
	FontPath             string    // Path to the primary UI font
	BackgroundImagePath  string    // Path to the background image
}

var currentTheme Theme

// SetTheme sets the active theme for the framework.
func SetTheme(theme Theme) {
	currentTheme = theme
}

// GetTheme returns the currently active theme.
func GetTheme() Theme {
	return currentTheme
}

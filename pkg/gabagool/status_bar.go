package gabagool

import (
	"sync/atomic"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// TimeFormat specifies 12-hour or 24-hour clock display
type TimeFormat int

const (
	TimeFormat24Hour TimeFormat = iota
	TimeFormat12Hour
)

// DynamicStatusBarIcon allows goroutines to safely update icon content.
// Use SetText to update from any goroutine.
type DynamicStatusBarIcon struct {
	text atomic.Value // stores string
}

// NewDynamicStatusBarIcon creates a new dynamic icon with initial text
func NewDynamicStatusBarIcon(initialText string) *DynamicStatusBarIcon {
	d := &DynamicStatusBarIcon{}
	d.text.Store(initialText)
	return d
}

// SetText updates the icon text (goroutine-safe)
func (d *DynamicStatusBarIcon) SetText(s string) {
	d.text.Store(s)
}

// GetText returns the current text
func (d *DynamicStatusBarIcon) GetText() string {
	if v := d.text.Load(); v != nil {
		return v.(string)
	}
	return ""
}

// StatusBarIcon represents a single icon slot in the status bar
type StatusBarIcon struct {
	Text    string                // Icon text (font glyph/symbol)
	Dynamic *DynamicStatusBarIcon // If set, reads from this instead of static Text
}

// StatusBarOptions configures the status bar appearance and behavior
type StatusBarOptions struct {
	Enabled    bool
	ShowTime   bool
	TimeFormat TimeFormat
	Icons      []StatusBarIcon // Max 3 icons
}

// DefaultStatusBarOptions returns sensible defaults with the status bar disabled
func DefaultStatusBarOptions() StatusBarOptions {
	return StatusBarOptions{
		Enabled:    false,
		ShowTime:   true,
		TimeFormat: TimeFormat24Hour,
		Icons:      nil,
	}
}

// calculateStatusBarWidth returns the total width of the status bar including pill and padding
// This is used by components to adjust title max width
func calculateStatusBarWidth(
	font *ttf.Font,
	options StatusBarOptions,
) int32 {
	if !options.Enabled {
		return 0
	}

	scaleFactor := internal.GetScaleFactor()
	outerPadding := int32(float32(20) * scaleFactor)
	innerPaddingX := int32(float32(10) * scaleFactor)
	iconSpacing := int32(float32(8) * scaleFactor)

	var contentWidth int32

	// Add time width
	if options.ShowTime {
		timeText := formatCurrentTime(options.TimeFormat)
		surface, err := font.RenderUTF8Blended(timeText, internal.GetTheme().HighlightColor)
		if err == nil && surface != nil {
			contentWidth += surface.W
			surface.Free()
		}
	}

	// Add icon widths
	maxIcons := 3
	if len(options.Icons) < maxIcons {
		maxIcons = len(options.Icons)
	}

	for i := 0; i < maxIcons; i++ {
		icon := options.Icons[i]
		var text string
		if icon.Dynamic != nil {
			text = icon.Dynamic.GetText()
		} else {
			text = icon.Text
		}

		if text != "" {
			surface, err := font.RenderUTF8Blended(text, internal.GetTheme().HighlightColor)
			if err == nil && surface != nil {
				if contentWidth > 0 {
					contentWidth += iconSpacing
				}
				contentWidth += surface.W
				surface.Free()
			}
		}
	}

	if contentWidth == 0 {
		return 0
	}

	// Total width = outer padding + pill (inner padding + content + inner padding) + some spacing
	return outerPadding + (innerPaddingX * 2) + contentWidth + iconSpacing
}

// calculateStatusBarContentWidth calculates the width of the status bar content (time + icons)
// without including the pill padding
func calculateStatusBarContentWidth(
	font *ttf.Font,
	options StatusBarOptions,
	iconSpacing int32,
) int32 {
	var contentWidth int32

	// Add time width
	if options.ShowTime {
		timeText := formatCurrentTime(options.TimeFormat)
		surface, err := font.RenderUTF8Blended(timeText, internal.GetTheme().HighlightColor)
		if err == nil && surface != nil {
			contentWidth += surface.W
			surface.Free()
		}
	}

	// Add icon widths
	maxIcons := 3
	if len(options.Icons) < maxIcons {
		maxIcons = len(options.Icons)
	}

	for i := 0; i < maxIcons; i++ {
		icon := options.Icons[i]
		var text string
		if icon.Dynamic != nil {
			text = icon.Dynamic.GetText()
		} else {
			text = icon.Text
		}

		if text != "" {
			surface, err := font.RenderUTF8Blended(text, internal.GetTheme().HighlightColor)
			if err == nil && surface != nil {
				if contentWidth > 0 {
					contentWidth += iconSpacing
				}
				contentWidth += surface.W
				surface.Free()
			}
		}
	}

	return contentWidth
}

// renderStatusBar renders the status bar in the top-right corner of the component
func renderStatusBar(
	renderer *sdl.Renderer,
	font *ttf.Font,
	options StatusBarOptions,
	margins internal.Padding,
) {
	if !options.Enabled {
		return
	}

	scaleFactor := internal.GetScaleFactor()
	window := internal.GetWindow()
	windowWidth, _ := window.Window.GetSize()

	outerPadding := int32(float32(20) * scaleFactor)
	innerPaddingX := int32(float32(10) * scaleFactor)
	innerPaddingY := int32(float32(6) * scaleFactor)
	iconSpacing := int32(float32(8) * scaleFactor)

	// Calculate content width (without pill padding)
	contentWidth := calculateStatusBarContentWidth(font, options, iconSpacing)
	if contentWidth <= 0 {
		return
	}

	// Calculate content height from time or icons (whichever is taller)
	var contentHeight int32
	if options.ShowTime {
		timeText := formatCurrentTime(options.TimeFormat)
		surface, err := font.RenderUTF8Blended(timeText, internal.GetTheme().AccentColor)
		if err == nil && surface != nil {
			contentHeight = surface.H
			surface.Free()
		}
	}

	// Check icon heights if no time or icons are taller
	maxIcons := 3
	if len(options.Icons) < maxIcons {
		maxIcons = len(options.Icons)
	}
	for i := 0; i < maxIcons; i++ {
		icon := options.Icons[i]
		var text string
		if icon.Dynamic != nil {
			text = icon.Dynamic.GetText()
		} else {
			text = icon.Text
		}
		if text != "" {
			surface, err := font.RenderUTF8Blended(text, internal.GetTheme().AccentColor)
			if err == nil && surface != nil {
				if surface.H > contentHeight {
					contentHeight = surface.H
				}
				surface.Free()
			}
		}
	}

	pillHeight := contentHeight + (innerPaddingY * 2)
	pillWidth := contentWidth + (innerPaddingX * 2)
	pillX := windowWidth - margins.Right - outerPadding - pillWidth
	pillY := int32(20) // Align with title start position

	// Draw pill background
	pillRect := &sdl.Rect{X: pillX, Y: pillY, W: pillWidth, H: pillHeight}
	cornerRadius := pillHeight / 2
	internal.DrawRoundedRect(renderer, pillRect, cornerRadius, internal.GetTheme().AccentColor)

	// Content starts inside the pill
	currentX := pillX + pillWidth - innerPaddingX
	contentY := pillY + innerPaddingY

	// 1. Render time (rightmost element)
	if options.ShowTime {
		timeText := formatCurrentTime(options.TimeFormat)
		currentX = renderStatusBarTime(renderer, font, timeText, currentX, contentY)
		currentX -= iconSpacing
	}

	// 2. Render icons (up to 3, right to left), vertically centered
	// Icons render right-to-left (last icon closest to time)
	for i := maxIcons - 1; i >= 0; i-- {
		icon := options.Icons[i]
		currentX = renderStatusBarIcon(renderer, font, icon, currentX, contentY, contentHeight)
		if i > 0 {
			currentX -= iconSpacing
		}
	}
}

func formatCurrentTime(format TimeFormat) string {
	now := time.Now()
	switch format {
	case TimeFormat12Hour:
		return now.Format("3:04 PM")
	case TimeFormat24Hour:
		return now.Format("15:04")
	default:
		return now.Format("15:04")
	}
}

func renderStatusBarTime(
	renderer *sdl.Renderer,
	font *ttf.Font,
	timeText string,
	rightX, y int32,
) int32 {
	textColor := internal.GetTheme().HintColor
	surface, err := font.RenderUTF8Blended(timeText, textColor)
	if err != nil || surface == nil {
		return rightX
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return rightX
	}
	defer texture.Destroy()

	textX := rightX - surface.W
	rect := sdl.Rect{X: textX, Y: y, W: surface.W, H: surface.H}
	renderer.Copy(texture, nil, &rect)

	return textX
}

func renderStatusBarIcon(
	renderer *sdl.Renderer,
	font *ttf.Font,
	icon StatusBarIcon,
	rightX, y, lineHeight int32,
) int32 {
	// Resolve text (dynamic takes priority)
	var text string
	if icon.Dynamic != nil {
		text = icon.Dynamic.GetText()
	} else {
		text = icon.Text
	}

	if text == "" {
		return rightX
	}

	textColor := internal.GetTheme().HintColor
	surface, err := font.RenderUTF8Blended(text, textColor)
	if err != nil || surface == nil {
		return rightX
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return rightX
	}
	defer texture.Destroy()

	// Position text at rightX, vertically centered with line height
	textX := rightX - surface.W
	textY := y + (lineHeight-surface.H)/2
	rect := sdl.Rect{X: textX, Y: textY, W: surface.W, H: surface.H}
	renderer.Copy(texture, nil, &rect)
	return textX
}

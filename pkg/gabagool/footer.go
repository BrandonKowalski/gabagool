package gabagool

import (
	"sync/atomic"
	"unicode"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// containsExtendedUnicode checks if a string contains characters that need a symbol font
// (emojis, special symbols, characters outside basic Latin/ASCII)
func containsExtendedUnicode(s string) bool {
	for _, r := range s {
		// Check for symbols, emojis, and other special characters
		if r > 0x2000 || unicode.IsSymbol(r) || unicode.Is(unicode.So, r) {
			return true
		}
	}
	return false
}

// FooterHelpItem represents a button and its help text that should be displayed in the footer.
// ButtonName is the text that will be displayed in the inner pill.
// HelpText is the text that will be displayed in the outer pill to the right of the button.
// IsConfirmButton marks this item as the confirm/start button, which can be hidden in multiselect mode when nothing is selected.
// Show is an optional atomic boolean that controls visibility. When not nil and false, the item is not rendered.
type FooterHelpItem struct {
	HelpText        string
	ButtonName      string
	IsConfirmButton bool
	Show            *atomic.Bool
}

func renderFooter(
	renderer *sdl.Renderer,
	font *ttf.Font,
	footerHelpItems []FooterHelpItem,
	bottomPadding int32,
	transparentBackground bool,
	centerSingleItem bool,
) {
	if len(footerHelpItems) == 0 {
		return
	}

	// Filter out items where Show is not nil and false
	visibleItems := make([]FooterHelpItem, 0, len(footerHelpItems))
	for _, item := range footerHelpItems {
		if item.Show != nil && !item.Show.Load() {
			continue
		}
		visibleItems = append(visibleItems, item)
	}
	footerHelpItems = visibleItems

	if len(footerHelpItems) == 0 {
		return
	}

	symbolFont := internal.Fonts.SmallSymbolFont

	scaleFactor := internal.GetScaleFactor()
	window := internal.GetWindow()
	windowWidth, windowHeight := window.Window.GetSize()
	y := windowHeight - bottomPadding - int32(float32(50)*scaleFactor)
	outerPillHeight := int32(float32(60) * scaleFactor)

	if !transparentBackground {
		// Add a black background for the entire footer area
		footerBackgroundRect := &sdl.Rect{
			X: 0,                                                // Start from left edge
			Y: y - 10,                                           // Same Y as the pills
			W: windowWidth - 15,                                 // Full window.GetWidth()
			H: outerPillHeight + int32(float32(50)*scaleFactor), // Same height as the pills
		}

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.FillRect(footerBackgroundRect)
	}

	innerPillMargin := int32(float32(6) * scaleFactor)
	var leftItems []FooterHelpItem
	var rightItems []FooterHelpItem
	switch len(footerHelpItems) {
	case 1:
		// For a single item, center it
		leftItems = footerHelpItems[0:1]
	case 2:
		leftItems = footerHelpItems[0:1]
		rightItems = footerHelpItems[1:2]
	case 3:
		leftItems = footerHelpItems[0:2]
		rightItems = footerHelpItems[2:3]
	case 4, 5, 6:
		leftItems = footerHelpItems[0:2]
		rightItems = footerHelpItems[2:min(4, len(footerHelpItems))]
	default:
		leftItems = footerHelpItems[0:2]
		rightItems = footerHelpItems[2:4]
	}

	if len(leftItems) > 0 {
		if len(footerHelpItems) == 1 && centerSingleItem {
			pillWidth := calculateContinuousPillWidth(font, symbolFont, leftItems, outerPillHeight, innerPillMargin)
			centerX := (windowWidth - pillWidth) / 2
			renderGroupAsContinuousPill(renderer, font, symbolFont, leftItems, centerX, y, outerPillHeight, innerPillMargin)
		} else {
			renderGroupAsContinuousPill(renderer, font, symbolFont, leftItems, bottomPadding, y, outerPillHeight, innerPillMargin)
		}
	}
	if len(rightItems) > 0 {
		rightGroupWidth := calculateContinuousPillWidth(font, symbolFont, rightItems, outerPillHeight, innerPillMargin)
		rightX := windowWidth - bottomPadding - rightGroupWidth
		renderGroupAsContinuousPill(renderer, font, symbolFont, rightItems, rightX, y, outerPillHeight, innerPillMargin)
	}
}

func calculateContinuousPillWidth(font *ttf.Font, symbolFont *ttf.Font, items []FooterHelpItem, outerPillHeight, innerPillMargin int32) int32 {
	scaleFactor := internal.GetScaleFactor()
	var totalWidth = int32(float32(10) * scaleFactor)

	innerPillHeight := outerPillHeight - (innerPillMargin * 2)

	for i, item := range items {
		// Use symbol font for extended unicode in button name
		buttonFont := font
		if containsExtendedUnicode(item.ButtonName) {
			buttonFont = symbolFont
		}

		buttonSurface, err := buttonFont.RenderUTF8Blended(item.ButtonName, internal.GetTheme().MainColor)
		if err != nil {
			continue
		}

		// Use symbol font for extended unicode in help text
		helpFont := font
		if containsExtendedUnicode(item.HelpText) {
			helpFont = symbolFont
		}

		helpSurface, err := helpFont.RenderUTF8Blended(item.HelpText, internal.GetTheme().PrimaryAccentColor)
		if err != nil || helpSurface == nil {
			buttonSurface.Free()
			continue
		}

		innerPillWidth := calculateInnerPillWidth(buttonSurface, innerPillHeight)

		itemWidth := innerPillWidth + 15 + helpSurface.W
		totalWidth += itemWidth
		if i < len(items)-1 {
			totalWidth += 20
		}
		buttonSurface.Free()
		helpSurface.Free()
	}
	totalWidth += int32(float32(10) * scaleFactor)
	return totalWidth
}

func calculateInnerPillWidth(buttonSurface *sdl.Surface, innerPillHeight int32) int32 {
	if buttonSurface.W <= innerPillHeight-20 {
		return innerPillHeight
	} else {
		return buttonSurface.W + 20
	}
}

func renderGroupAsContinuousPill(
	renderer *sdl.Renderer,
	font *ttf.Font,
	symbolFont *ttf.Font,
	items []FooterHelpItem,
	startX, y,
	outerPillHeight,
	innerPillMargin int32,
) {
	if len(items) == 0 {
		return
	}
	scaleFactor := internal.GetScaleFactor()
	pillWidth := calculateContinuousPillWidth(font, symbolFont, items, outerPillHeight, innerPillMargin)
	outerPillRect := &sdl.Rect{
		X: startX,
		Y: y,
		W: pillWidth,
		H: outerPillHeight,
	}

	cornerRadius := outerPillHeight / 2
	internal.DrawRoundedRect(renderer, outerPillRect, cornerRadius, internal.GetTheme().PrimaryAccentColor)

	currentX := startX + int32(float32(10)*scaleFactor)
	innerPillHeight := outerPillHeight - (innerPillMargin * 2)

	// Apply damping to Padding for smaller screens
	var paddingFactor float32 = 1.0
	if scaleFactor < 1.0 {
		paddingFactor = 0.5 + (scaleFactor * 0.5) // Reduces Padding impact on small screens
	}
	rightPadding := int32(float32(30) * paddingFactor)

	for _, item := range items {
		// Use symbol font for extended unicode in button name
		buttonFont := font
		if containsExtendedUnicode(item.ButtonName) {
			buttonFont = symbolFont
		}

		buttonSurface, err := buttonFont.RenderUTF8Blended(item.ButtonName, internal.GetTheme().SecondaryAccentColor)
		if err != nil || buttonSurface == nil {
			continue
		}

		// Use symbol font for extended unicode in help text
		helpFont := font
		if containsExtendedUnicode(item.HelpText) {
			helpFont = symbolFont
		}

		helpSurface, err := helpFont.RenderUTF8Blended(item.HelpText, internal.GetTheme().HintInfoColor)
		if err != nil || helpSurface == nil {
			buttonSurface.Free()
			continue
		}

		innerPillWidth := calculateInnerPillWidth(buttonSurface, innerPillHeight)
		isCircle := innerPillWidth == innerPillHeight

		if isCircle {
			drawCircleShape(renderer, currentX+innerPillHeight/2, y+innerPillMargin+innerPillHeight/2, innerPillHeight/2, internal.GetTheme().MainColor)
		} else {
			innerPillRect := &sdl.Rect{
				X: currentX,
				Y: y + innerPillMargin,
				W: innerPillWidth,
				H: innerPillHeight,
			}
			cornerRadiusInner := innerPillHeight / 2
			internal.DrawRoundedRect(renderer, innerPillRect, cornerRadiusInner, internal.GetTheme().MainColor)
		}

		buttonTexture, err := renderer.CreateTextureFromSurface(buttonSurface)
		if err == nil {
			buttonTextRect := &sdl.Rect{
				X: currentX + (innerPillWidth-buttonSurface.W)/2,
				Y: y + (outerPillHeight-buttonSurface.H)/2,
				W: buttonSurface.W,
				H: buttonSurface.H,
			}
			renderer.Copy(buttonTexture, nil, buttonTextRect)
			buttonTexture.Destroy()
		}

		currentX += innerPillWidth + int32(float32(10)*scaleFactor)

		helpTexture, err := renderer.CreateTextureFromSurface(helpSurface)
		if err == nil {
			helpTextRect := &sdl.Rect{
				X: currentX,
				Y: y + (outerPillHeight-helpSurface.H)/2,
				W: helpSurface.W,
				H: helpSurface.H,
			}
			renderer.Copy(helpTexture, nil, helpTextRect)
			helpTexture.Destroy()
		}

		currentX += helpSurface.W + rightPadding
		buttonSurface.Free()
		helpSurface.Free()
	}
}

func drawCircleShape(renderer *sdl.Renderer, centerX, centerY, radius int32, color sdl.Color) {
	gfx.FilledCircleColor(
		renderer,
		centerX,
		centerY,
		radius,
		color,
	)

	gfx.AACircleColor(
		renderer,
		centerX,
		centerY,
		radius,
		color,
	)

	if radius > 2 {
		gfx.AACircleColor(
			renderer,
			centerX,
			centerY,
			radius-1,
			color,
		)
	}
}

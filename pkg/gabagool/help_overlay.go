package gabagool

import (
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// helpOverlay renders a full-screen scrollable help text overlay.
// Used by UI components to display contextual help when the menu button is pressed.
type helpOverlay struct {
	Title           string
	Lines           []string
	ExitText        string
	ShowingHelp     bool
	ScrollOffset    int32
	MaxScrollOffset int32
	Padding         int32
	LineHeight      int32
	Width           int32
	Height          int32
	BackgroundColor sdl.Color
	TextColor       sdl.Color
	ScrollbarColor  sdl.Color
	ScrollbarWidth  int32
	ExitTextPadding int32
}

// newHelpOverlay creates a help overlay with the given title, content lines, and exit instruction text.
func newHelpOverlay(title string, lines []string, exitText string) *helpOverlay {
	window := internal.GetWindow()
	width, height := window.Window.GetSize()

	if title == "" {
		title = "Help"
	}

	return &helpOverlay{
		Title:           title,
		Lines:           lines,
		ExitText:        exitText,
		ScrollOffset:    0,
		MaxScrollOffset: 0,
		Padding:         20,
		LineHeight:      50,
		Width:           width,
		Height:          height,
		BackgroundColor: sdl.Color{R: 0, G: 0, B: 0, A: 255},
		TextColor:       sdl.Color{R: 255, G: 255, B: 255, A: 255},
		ScrollbarColor:  sdl.Color{R: 150, G: 150, B: 150, A: 180},
		ScrollbarWidth:  8,
		ExitTextPadding: 10,
	}
}

// render draws the help overlay if visible, including title, scrollable content, and exit text.
func (h *helpOverlay) render(renderer *sdl.Renderer, font *ttf.Font) {
	if !h.ShowingHelp {
		return
	}

	h.calculateMaxScroll()

	bgRect := &sdl.Rect{X: 0, Y: 0, W: h.Width, H: h.Height}
	renderer.SetDrawColor(h.BackgroundColor.R, h.BackgroundColor.G, h.BackgroundColor.B, h.BackgroundColor.A)
	renderer.FillRect(bgRect)

	titleText, err := font.RenderUTF8Blended(h.Title, h.TextColor)
	if err == nil {
		titleTexture, err := renderer.CreateTextureFromSurface(titleText)
		if err == nil {
			titleRect := &sdl.Rect{
				X: h.Padding,
				Y: h.Padding,
				W: titleText.W,
				H: titleText.H,
			}
			renderer.Copy(titleTexture, nil, titleRect)
			titleTexture.Destroy()
		}
		titleText.Free()
	}

	contentY := h.Padding + h.LineHeight*2
	contentHeight := h.Height - contentY - h.Padding - h.LineHeight - h.ExitTextPadding*7

	contentWidth := h.Width - h.Padding*2
	if h.MaxScrollOffset > 0 {
		contentWidth -= h.ScrollbarWidth + 10
	}

	y := contentY - h.ScrollOffset

	for _, line := range h.Lines {

		if y+h.LineHeight < contentY || y > contentY+contentHeight {
			y += h.LineHeight
			continue
		}

		textSurface, err := font.RenderUTF8Blended(line, h.TextColor)
		if err == nil {
			textTexture, err := renderer.CreateTextureFromSurface(textSurface)
			if err == nil {
				textRect := &sdl.Rect{
					X: h.Padding,
					Y: y,
					W: textSurface.W,
					H: textSurface.H,
				}
				renderer.Copy(textTexture, nil, textRect)
				textTexture.Destroy()
			}
			textSurface.Free()
		}

		y += h.LineHeight
	}

	if h.MaxScrollOffset > 0 {
		scrollbarX := h.Width - h.Padding - h.ScrollbarWidth
		scrollbarY := contentY
		scrollbarHeight := contentHeight

		// Clear scrollbar area to prevent anti-aliasing artifacts from alpha blending
		renderer.SetDrawColor(h.BackgroundColor.R, h.BackgroundColor.G, h.BackgroundColor.B, 255)
		renderer.FillRect(&sdl.Rect{
			X: scrollbarX - 2,
			Y: scrollbarY - 2,
			W: h.ScrollbarWidth + 4,
			H: scrollbarHeight + 4,
		})

		scrollbarBgColor := sdl.Color{R: 50, G: 50, B: 50, A: 255}
		internal.DrawSmoothScrollbar(renderer, scrollbarX, scrollbarY, h.ScrollbarWidth, scrollbarHeight, scrollbarBgColor)

		totalContentHeight := int32(len(h.Lines)) * h.LineHeight
		handleRatio := float64(contentHeight) / float64(totalContentHeight)
		if handleRatio > 1.0 {
			handleRatio = 1.0
		}

		handleHeight := int32(float64(scrollbarHeight) * handleRatio)
		if handleHeight < 30 {
			handleHeight = 30
		}

		scrollRatio := 0.0
		if h.MaxScrollOffset > 0 {
			scrollRatio = float64(h.ScrollOffset) / float64(h.MaxScrollOffset)
		}

		handleY := scrollbarY + int32(float64(scrollbarHeight-handleHeight)*scrollRatio)

		internal.DrawSmoothScrollbar(renderer, scrollbarX, handleY, h.ScrollbarWidth, handleHeight, sdl.Color{R: 150, G: 150, B: 150, A: 255})
	}

	exitText := h.ExitText
	exitSurface, err := font.RenderUTF8Blended(exitText, h.TextColor)
	if err == nil && exitSurface != nil {
		exitTexture, err := renderer.CreateTextureFromSurface(exitSurface)
		if err == nil {
			exitRect := &sdl.Rect{
				X: (h.Width - exitSurface.W) / 2,
				Y: h.Height - h.Padding - exitSurface.H - h.ExitTextPadding,
				W: exitSurface.W,
				H: exitSurface.H,
			}
			renderer.Copy(exitTexture, nil, exitRect)
			exitTexture.Destroy()
		}
		exitSurface.Free()
	}
}

// calculateMaxScroll computes the maximum scroll offset based on content height.
func (h *helpOverlay) calculateMaxScroll() {
	contentY := h.Padding + h.LineHeight*2
	contentHeight := h.Height - contentY - h.Padding - h.LineHeight - h.ExitTextPadding*7

	totalContentHeight := int32(len(h.Lines)) * h.LineHeight

	if totalContentHeight > contentHeight {
		h.MaxScrollOffset = totalContentHeight - contentHeight
	} else {
		h.MaxScrollOffset = 0
	}
}

// scroll moves the content up (negative) or down (positive) by one line.
func (h *helpOverlay) scroll(direction int) {
	if !h.ShowingHelp {
		return
	}

	newOffset := h.ScrollOffset + int32(direction)*h.LineHeight

	if newOffset < 0 {
		newOffset = 0
	}

	if newOffset > h.MaxScrollOffset {
		newOffset = h.MaxScrollOffset
	}

	h.ScrollOffset = newOffset
}

// handleInput processes D-pad input for scrolling. Returns true if the event was consumed.
func (h *helpOverlay) handleInput(event interface{}) bool {
	processor := internal.GetInputProcessor()
	inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))

	if inputEvent == nil || !inputEvent.Pressed {
		return false
	}

	button := inputEvent.Button

	switch button {
	case constants.VirtualButtonUp:
		h.scroll(-1)
		return true
	case constants.VirtualButtonDown:
		h.scroll(1)
		return true
	case constants.VirtualButtonMenu, constants.VirtualButtonB:
		return false
	default:
		return false
	}
}

// toggle shows or hides the help overlay, resetting scroll position when shown.
func (h *helpOverlay) toggle() {
	h.ShowingHelp = !h.ShowingHelp
	if h.ShowingHelp {
		h.ScrollOffset = 0
	}
}

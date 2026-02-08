package gabagool

import (
	"strings"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// SelectionMessageSettings configures the selection message component.
type SelectionMessageSettings struct {
	// ConfirmButton is the button used to confirm the selection (default: VirtualButtonA)
	ConfirmButton constants.VirtualButton
	// BackButton is the button used to go back/cancel (default: VirtualButtonB)
	BackButton constants.VirtualButton
	// DisableBackButton hides the back button and disables its functionality
	DisableBackButton bool
	// InitialSelection is the index of the initially selected option (default: 0)
	InitialSelection int
	// StatusBar configures the optional status bar in the top-right corner
	StatusBar StatusBarOptions
}

// SelectionMessageResult represents the result of a selection message.
type SelectionMessageResult struct {
	// SelectedIndex is the index of the selected option
	SelectedIndex int
	// SelectedValue is the value of the selected option
	SelectedValue interface{}
}

// SelectionOption represents a selectable option in the selection message.
type SelectionOption struct {
	// DisplayName is the text shown to the user
	DisplayName string
	// Description is optional text that replaces the main message when this option is selected
	Description string
	// Value is the value returned when this option is selected
	Value interface{}
}

type selectionMessageController struct {
	message           string
	options           []SelectionOption
	selectedIndex     int
	visibleStartIndex int
	confirmButton     constants.VirtualButton
	backButton        constants.VirtualButton
	disableBack       bool
	footerHelpItems   []FooterHelpItem
	statusBar         StatusBarOptions
	inputDelay        time.Duration
	lastInputTime     time.Time
	confirmed         bool
	cancelled         bool
}

const maxVisibleOptions = 3

// SelectionMessage displays a message with horizontally selectable options.
// The user can navigate options with left/right and confirm with the confirm button.
// Returns ErrCancelled if the user presses the back button.
func SelectionMessage(message string, options []SelectionOption, footerHelpItems []FooterHelpItem, settings SelectionMessageSettings) (*SelectionMessageResult, error) {
	if len(options) == 0 {
		return nil, ErrCancelled
	}

	window := internal.GetWindow()
	renderer := window.Renderer

	controller := &selectionMessageController{
		message:         message,
		options:         options,
		selectedIndex:   settings.InitialSelection,
		confirmButton:   settings.ConfirmButton,
		backButton:      settings.BackButton,
		disableBack:     settings.DisableBackButton,
		footerHelpItems: footerHelpItems,
		statusBar:       settings.StatusBar,
		inputDelay:      constants.DefaultInputDelay,
		lastInputTime:   time.Now(),
	}

	if controller.confirmButton == constants.VirtualButtonUnassigned {
		controller.confirmButton = constants.VirtualButtonA
	}
	if controller.backButton == constants.VirtualButtonUnassigned {
		controller.backButton = constants.VirtualButtonB
	}

	if controller.selectedIndex < 0 || controller.selectedIndex >= len(options) {
		controller.selectedIndex = 0
	}

	if len(options) >= maxVisibleOptions {
		controller.visibleStartIndex = controller.selectedIndex - maxVisibleOptions/2
		if controller.visibleStartIndex < 0 {
			controller.visibleStartIndex += len(options)
		}
	}

	for {
		if !controller.handleEvents() {
			break
		}

		controller.render(renderer, window)
	}

	if controller.cancelled {
		return nil, ErrCancelled
	}

	return &SelectionMessageResult{
		SelectedIndex: controller.selectedIndex,
		SelectedValue: controller.options[controller.selectedIndex].Value,
	}, nil
}

func (c *selectionMessageController) handleEvents() bool {
	processor := internal.GetInputProcessor()

	if event := sdl.WaitEventTimeout(16); event != nil {
		switch event.(type) {
		case *sdl.QuitEvent:
			c.cancelled = true
			return false

		case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
			inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
			if inputEvent == nil || !inputEvent.Pressed {
				return true
			}

			if time.Since(c.lastInputTime) < c.inputDelay {
				return true
			}
			c.lastInputTime = time.Now()

			switch inputEvent.Button {
			case constants.VirtualButtonLeft:
				c.navigateLeft()
			case constants.VirtualButtonRight:
				c.navigateRight()
			case c.confirmButton, constants.VirtualButtonStart:
				c.confirmed = true
				return false
			case c.backButton:
				if !c.disableBack {
					c.cancelled = true
					return false
				}
			}
		}
	}
	return true
}

func (c *selectionMessageController) navigateLeft() {
	c.selectedIndex--
	if c.selectedIndex < 0 {
		c.selectedIndex = len(c.options) - 1
	}
	// Smoothly scroll the visible window left
	c.visibleStartIndex--
	if c.visibleStartIndex < 0 {
		c.visibleStartIndex = len(c.options) - 1
	}
}

func (c *selectionMessageController) navigateRight() {
	c.selectedIndex++
	if c.selectedIndex >= len(c.options) {
		c.selectedIndex = 0
	}
	// Smoothly scroll the visible window right
	c.visibleStartIndex++
	if c.visibleStartIndex >= len(c.options) {
		c.visibleStartIndex = 0
	}
}

func (c *selectionMessageController) render(renderer *sdl.Renderer, window *internal.Window) {
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	windowWidth := window.GetWidth()
	windowHeight := window.GetHeight()

	messageFont := internal.Fonts.LargeFont
	optionFont := internal.Fonts.MediumFont

	maxMessageWidth := int32(float64(windowWidth) * 0.75)
	if maxMessageWidth > 800 {
		maxMessageWidth = 800
	}

	// Determine display message (use description if selected option has one)
	displayMessage := c.message
	if desc := c.options[c.selectedIndex].Description; desc != "" {
		displayMessage = desc
	}

	// Calculate max message height across all possible messages to prevent bouncing
	maxMessageHeight := c.calculateTextHeight(c.message, messageFont, maxMessageWidth)
	for _, opt := range c.options {
		if opt.Description != "" {
			h := c.calculateTextHeight(opt.Description, messageFont, maxMessageWidth)
			if h > maxMessageHeight {
				maxMessageHeight = h
			}
		}
	}

	optionHeight := int32(optionFont.Height())
	spacing := int32(30)
	totalHeight := maxMessageHeight + spacing + optionHeight

	startY := (windowHeight - totalHeight) / 2

	// Center the current message within the max height area
	currentMessageHeight := c.calculateTextHeight(displayMessage, messageFont, maxMessageWidth)
	messageY := startY + (maxMessageHeight-currentMessageHeight)/2

	centerX := windowWidth / 2
	internal.RenderMultilineText(
		renderer,
		displayMessage,
		messageFont,
		maxMessageWidth,
		centerX,
		messageY,
		sdl.Color{R: 255, G: 255, B: 255, A: 255},
		constants.TextAlignCenter,
	)

	optionY := startY + maxMessageHeight + spacing
	c.renderOptions(renderer, centerX, optionY, optionFont)

	renderStatusBar(renderer, internal.Fonts.SmallFont, c.statusBar, internal.UniformPadding(20))

	renderFooter(
		renderer,
		internal.Fonts.SmallFont,
		c.footerHelpItems,
		20,
		false,
		true,
	)

	window.Present()
}

func (c *selectionMessageController) calculateTextHeight(text string, font *ttf.Font, maxWidth int32) int32 {
	if text == "" {
		return 0
	}

	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	_, fontHeight, err := font.SizeUTF8("Aj")
	if err != nil {
		fontHeight = 20
	}

	lineSpacing := int32(float64(fontHeight) * 0.2)
	totalLines := int32(0)

	for _, line := range lines {
		if line == "" {
			totalLines++
			continue
		}

		words := strings.Fields(line)
		currentLine := ""

		for _, word := range words {
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			width, _, _ := font.SizeUTF8(testLine)
			if int32(width) > maxWidth && currentLine != "" {
				totalLines++
				currentLine = word
			} else {
				currentLine = testLine
			}
		}
		if currentLine != "" {
			totalLines++
		}
	}

	return totalLines*int32(fontHeight) + (totalLines-1)*lineSpacing
}

func (c *selectionMessageController) renderOptions(renderer *sdl.Renderer, centerX, y int32, font *ttf.Font) {
	// Render format: < Option1 | Option2 | Option3 >
	// Selected option is highlighted
	// Only show up to maxVisibleOptions at a time

	arrowColor := sdl.Color{R: 180, G: 180, B: 180, A: 255}
	selectedColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	unselectedColor := sdl.Color{R: 100, G: 100, B: 100, A: 255}
	separatorColor := sdl.Color{R: 80, G: 80, B: 80, A: 255}

	// Build the options string and calculate positions
	leftArrow := "<  "
	rightArrow := "  >"
	separator := "  |  "

	// Calculate widths
	leftArrowWidth := c.getTextWidth(font, leftArrow)
	rightArrowWidth := c.getTextWidth(font, rightArrow)
	separatorWidth := c.getTextWidth(font, separator)

	numOptions := len(c.options)
	visibleCount := maxVisibleOptions
	if visibleCount > numOptions {
		visibleCount = numOptions
	}

	// Find the max width of any single option for even spacing
	maxOptionWidth := int32(0)
	for _, opt := range c.options {
		w := c.getTextWidth(font, opt.DisplayName)
		if w > maxOptionWidth {
			maxOptionWidth = w
		}
	}

	var visibleOptions []SelectionOption
	var visibleIndices []int
	for j := 0; j < visibleCount; j++ {
		idx := (c.visibleStartIndex + j) % numOptions
		visibleOptions = append(visibleOptions, c.options[idx])
		visibleIndices = append(visibleIndices, idx)
	}

	optionsAreaWidth := int32(visibleCount)*maxOptionWidth + int32(visibleCount-1)*separatorWidth
	totalWidth := leftArrowWidth + optionsAreaWidth + rightArrowWidth
	startX := centerX - totalWidth/2

	// Render left arrow
	x := startX
	c.renderText(renderer, font, leftArrow, x, y, arrowColor)
	x += leftArrowWidth

	// Render visible options with separators, each in a fixed-width slot
	for i, opt := range visibleOptions {
		color := unselectedColor
		if visibleIndices[i] == c.selectedIndex {
			color = selectedColor
		}
		// Center option text within its fixed-width slot
		optWidth := c.getTextWidth(font, opt.DisplayName)
		slotPadding := (maxOptionWidth - optWidth) / 2
		c.renderText(renderer, font, opt.DisplayName, x+slotPadding, y, color)
		x += maxOptionWidth

		if i < len(visibleOptions)-1 {
			c.renderText(renderer, font, separator, x, y, separatorColor)
			x += separatorWidth
		}
	}

	// Render right arrow at fixed position
	rightArrowX := startX + leftArrowWidth + optionsAreaWidth
	c.renderText(renderer, font, rightArrow, rightArrowX, y, arrowColor)
}

func (c *selectionMessageController) getTextWidth(font *ttf.Font, text string) int32 {
	width, _, err := font.SizeUTF8(text)
	if err != nil {
		return 0
	}
	return int32(width)
}

func (c *selectionMessageController) renderText(renderer *sdl.Renderer, font *ttf.Font, text string, x, y int32, color sdl.Color) {
	surface, err := font.RenderUTF8Blended(text, color)
	if err != nil {
		return
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return
	}
	defer texture.Destroy()

	rect := sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}
	renderer.Copy(texture, nil, &rect)
}

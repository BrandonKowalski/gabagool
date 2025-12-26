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
	// Value is the value returned when this option is selected
	Value interface{}
}

type selectionMessageController struct {
	message         string
	options         []SelectionOption
	selectedIndex   int
	confirmButton   constants.VirtualButton
	backButton      constants.VirtualButton
	disableBack     bool
	footerHelpItems []FooterHelpItem
	inputDelay      time.Duration
	lastInputTime   time.Time
	confirmed       bool
	cancelled       bool
}

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
		inputDelay:      constants.DefaultInputDelay,
		lastInputTime:   time.Now(),
	}

	// Set defaults
	if controller.confirmButton == constants.VirtualButtonUnassigned {
		controller.confirmButton = constants.VirtualButtonA
	}
	if controller.backButton == constants.VirtualButtonUnassigned {
		controller.backButton = constants.VirtualButtonB
	}

	// Validate initial selection
	if controller.selectedIndex < 0 || controller.selectedIndex >= len(options) {
		controller.selectedIndex = 0
	}

	for {
		if !controller.handleEvents() {
			break
		}

		controller.render(renderer, window)
		sdl.Delay(16)
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

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			c.cancelled = true
			return false

		case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
			inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
			if inputEvent == nil || !inputEvent.Pressed {
				continue
			}

			if time.Since(c.lastInputTime) < c.inputDelay {
				continue
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
}

func (c *selectionMessageController) navigateRight() {
	c.selectedIndex++
	if c.selectedIndex >= len(c.options) {
		c.selectedIndex = 0
	}
}

func (c *selectionMessageController) render(renderer *sdl.Renderer, window *internal.Window) {
	// Clear screen
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	if window.Background != nil {
		window.RenderBackground()
	}

	windowWidth := window.GetWidth()
	windowHeight := window.GetHeight()

	// Calculate content dimensions
	messageFont := internal.Fonts.SmallFont
	optionFont := internal.Fonts.MediumFont

	maxMessageWidth := int32(float64(windowWidth) * 0.75)
	if maxMessageWidth > 800 {
		maxMessageWidth = 800
	}

	// Calculate total content height
	messageHeight := c.calculateTextHeight(c.message, messageFont, maxMessageWidth)
	optionHeight := int32(optionFont.Height())
	spacing := int32(30)
	totalHeight := messageHeight + spacing + optionHeight

	// Start Y position (centered)
	startY := (windowHeight - totalHeight) / 2

	// Render message
	centerX := windowWidth / 2
	internal.RenderMultilineText(
		renderer,
		c.message,
		messageFont,
		maxMessageWidth,
		centerX,
		startY,
		sdl.Color{R: 255, G: 255, B: 255, A: 255},
		constants.TextAlignCenter,
	)

	// Render options selector
	optionY := startY + messageHeight + spacing
	c.renderOptions(renderer, centerX, optionY, optionFont)

	// Render footer
	renderFooter(
		renderer,
		internal.Fonts.SmallFont,
		c.footerHelpItems,
		20,
		false,
		true,
	)

	renderer.Present()
}

func (c *selectionMessageController) calculateTextHeight(text string, font *ttf.Font, maxWidth int32) int32 {
	if text == "" {
		return 0
	}

	lines := strings.Split(text, "\n")
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

	arrowColor := sdl.Color{R: 180, G: 180, B: 180, A: 255}
	selectedColor := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	unselectedColor := sdl.Color{R: 100, G: 100, B: 100, A: 255}
	separatorColor := sdl.Color{R: 80, G: 80, B: 80, A: 255}

	// Build the options string and calculate positions
	leftArrow := "<  "
	rightArrow := "  >"
	separator := "  |  "

	// Calculate total width
	leftArrowWidth := c.getTextWidth(font, leftArrow)
	rightArrowWidth := c.getTextWidth(font, rightArrow)
	separatorWidth := c.getTextWidth(font, separator)

	var optionWidths []int32
	totalOptionsWidth := int32(0)
	for i, opt := range c.options {
		w := c.getTextWidth(font, opt.DisplayName)
		optionWidths = append(optionWidths, w)
		totalOptionsWidth += w
		if i < len(c.options)-1 {
			totalOptionsWidth += separatorWidth
		}
	}

	totalWidth := leftArrowWidth + totalOptionsWidth + rightArrowWidth
	startX := centerX - totalWidth/2

	// Render left arrow
	x := startX
	c.renderText(renderer, font, leftArrow, x, y, arrowColor)
	x += leftArrowWidth

	// Render options with separators
	for i, opt := range c.options {
		color := unselectedColor
		if i == c.selectedIndex {
			color = selectedColor
		}
		c.renderText(renderer, font, opt.DisplayName, x, y, color)
		x += optionWidths[i]

		if i < len(c.options)-1 {
			c.renderText(renderer, font, separator, x, y, separatorColor)
			x += separatorWidth
		}
	}

	// Render right arrow
	c.renderText(renderer, font, rightArrow, x, y, arrowColor)
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

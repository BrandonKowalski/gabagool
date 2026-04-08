package gabagool

import (
	"fmt"
	"sync"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
)

// InputMapping is the public alias for the input mapping type returned by the logger wizard.
type InputMapping = internal.InputMapping

// InputCaptureButton defines a virtual button to configure and its display name shown to the user.
type InputCaptureButton struct {
	Button      constants.VirtualButton // The virtual button to map
	DisplayName string                  // Label shown on screen (e.g. "A Button", "Jump")
}

// InputCaptureOptions configures the input mapping wizard component.
type InputCaptureOptions struct {
	// Title shown at the top of the wizard screen. Defaults to "Input Configuration".
	Title string
	// InstructionText is shown on an intro screen before mapping begins.
	// The user presses any button to continue. If empty, a default message is used.
	// Set to " " (space) to skip the instruction screen entirely.
	InstructionText string
	// ReleasedEarlyText is shown when the user releases a button before the hold completes.
	// Defaults to "Released too early".
	ReleasedEarlyText string
	// CompleteText is shown when all buttons have been mapped.
	// Defaults to "Configuration Complete!".
	CompleteText string
	// HoldDuration is how long a button must be held to register. Defaults to 1 second.
	HoldDuration time.Duration
	// Buttons to configure. Defaults to all standard buttons when empty.
	Buttons []InputCaptureButton
}

// defaultInputCaptureButtons returns the standard set of buttons to configure.
func defaultInputCaptureButtons() []InputCaptureButton {
	return []InputCaptureButton{
		{constants.VirtualButtonA, "A Button"},
		{constants.VirtualButtonB, "B Button"},
		{constants.VirtualButtonX, "X Button"},
		{constants.VirtualButtonY, "Y Button"},
		{constants.VirtualButtonUp, "D-Pad Up"},
		{constants.VirtualButtonDown, "D-Pad Down"},
		{constants.VirtualButtonLeft, "D-Pad Left"},
		{constants.VirtualButtonRight, "D-Pad Right"},
		{constants.VirtualButtonStart, "Start"},
		{constants.VirtualButtonSelect, "Select"},
		{constants.VirtualButtonL1, "L1"},
		{constants.VirtualButtonL2, "L2"},
		{constants.VirtualButtonR1, "R1"},
		{constants.VirtualButtonR2, "R2"},
		{constants.VirtualButtonMenu, "Menu"},
	}
}

// buttonConfig pairs a virtual button with its display name for the input capture wizard.
type buttonConfig struct {
	internalButton constants.VirtualButton
	displayName    string
}

// mappedInput stores the raw input code and its source type (keyboard, joystick, etc.).
type mappedInput struct {
	code   int
	source internal.Source
}

type inputState int

const (
	stateInstruction inputState = iota
	statePrompting
	stateHolding
	stateRegistered
	stateReleasedEarly
)

// inputCaptureController manages the interactive input mapping wizard that guides
// users through configuring each button on their controller.
type inputCaptureController struct {
	title             string
	instructionText   string
	releasedEarlyText string
	completeText      string
	currentButtonIdx  int
	mappedButtons     map[constants.VirtualButton]mappedInput
	mutex             sync.Mutex
	buttonSequence    []buttonConfig

	// State machine
	state          inputState
	holdStartTime  time.Time
	holdInput      mappedInput
	holdInputLabel string
	holdDuration   time.Duration
	completedTime  time.Time
}

func newInputCapture(options InputCaptureOptions) *inputCaptureController {
	title := options.Title
	if title == "" {
		title = "Input Configuration"
	}

	holdDuration := options.HoldDuration
	if holdDuration == 0 {
		holdDuration = 1000 * time.Millisecond
	}

	instructionText := options.InstructionText
	skipInstruction := false
	if instructionText == " " {
		skipInstruction = true
	} else if instructionText == "" {
		holdSeconds := float64(holdDuration) / float64(time.Second)
		instructionText = fmt.Sprintf("Press and hold each button when prompted.\nEach input must be held for %.1g seconds to register.", holdSeconds)
	}

	releasedEarlyText := options.ReleasedEarlyText
	if releasedEarlyText == "" {
		releasedEarlyText = "Released too early"
	}

	completeText := options.CompleteText
	if completeText == "" {
		completeText = "Configuration Complete!"
	}

	buttons := options.Buttons
	if len(buttons) == 0 {
		buttons = defaultInputCaptureButtons()
	}

	sequence := make([]buttonConfig, len(buttons))
	for i, b := range buttons {
		sequence[i] = buttonConfig{internalButton: b.Button, displayName: b.DisplayName}
	}

	initialState := stateInstruction
	if skipInstruction {
		initialState = statePrompting
	}

	return &inputCaptureController{
		title:             title,
		instructionText:   instructionText,
		releasedEarlyText: releasedEarlyText,
		completeText:      completeText,
		mappedButtons:     make(map[constants.VirtualButton]mappedInput),
		currentButtonIdx:  0,
		state:             initialState,
		holdDuration:      holdDuration,
		buttonSequence:    sequence,
	}
}

// ShowInputCapture runs an interactive wizard that prompts the user to press and hold
// each button on their controller, building a custom input mapping. Each button must
// be held for 1 second to confirm. Returns the completed mapping.
//
// An optional instruction screen is shown first (customizable via InstructionText).
// Use InputCaptureOptions to customize the title and the subset of buttons to configure.
// An empty Options value uses sensible defaults (all standard buttons).
func ShowInputCapture(options InputCaptureOptions) *InputMapping {
	logger := newInputCapture(options)

	internal.GetInternalLogger().Info("Input logger started", "totalButtons", len(logger.buttonSequence))

	running := true

	for running {
		event := sdl.WaitEventTimeout(16)
		for ; event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
			default:
				running = logger.handleEvent(event)
			}
		}

		// Time-based state transitions
		if !logger.checkTimedTransitions() {
			running = false
		}

		logger.render()
	}

	return logger.buildMapping()
}

// checkTimedTransitions handles time-based state changes (hold completion, complete screen).
// Returns false when all buttons are configured and the complete screen has been shown.
func (il *inputCaptureController) checkTimedTransitions() bool {
	il.mutex.Lock()
	defer il.mutex.Unlock()

	// Check if complete screen has been shown long enough
	if il.currentButtonIdx >= len(il.buttonSequence) && !il.completedTime.IsZero() {
		if time.Since(il.completedTime) >= 1*time.Second {
			return false
		}
		return true
	}

	if il.state == stateHolding {
		if time.Since(il.holdStartTime) >= il.holdDuration {
			currentButton := il.buttonSequence[il.currentButtonIdx]
			il.mappedButtons[currentButton.internalButton] = il.holdInput
			internal.GetInternalLogger().Debug("Hold completed, registered input",
				"button", currentButton.displayName,
				"source", il.holdInput.source)
			il.state = stateRegistered
		}
	}

	return true
}

func (il *inputCaptureController) handleEvent(event sdl.Event) bool {
	il.mutex.Lock()
	defer il.mutex.Unlock()

	if il.currentButtonIdx >= len(il.buttonSequence) {
		return true // Ignore events during complete screen
	}

	switch il.state {
	case stateInstruction:
		il.handleInstructionEvent(event)
	case statePrompting, stateReleasedEarly:
		il.handlePromptingEvent(event)
	case stateHolding:
		il.handleHoldingEvent(event)
	case stateRegistered:
		il.handleRegisteredEvent(event)
	}

	return true
}

// handleInstructionEvent waits for any button press to dismiss the instruction screen.
func (il *inputCaptureController) handleInstructionEvent(event sdl.Event) {
	pressed := false
	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		pressed = e.State == sdl.PRESSED
	case *sdl.JoyButtonEvent:
		pressed = e.State == sdl.PRESSED
	case *sdl.ControllerButtonEvent:
		pressed = e.State == sdl.PRESSED
	case *sdl.JoyAxisEvent:
		pressed = internal.Abs(int(e.Value)) > 16000
	case *sdl.JoyHatEvent:
		pressed = e.Value != sdl.HAT_CENTERED
	}
	if pressed {
		il.state = statePrompting
	}
}

func (il *inputCaptureController) handlePromptingEvent(event sdl.Event) {
	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		if e.State == sdl.PRESSED {
			il.startHold(
				mappedInput{code: int(e.Keysym.Sym), source: internal.SourceKeyboard},
				fmt.Sprintf("Keyboard: %d", int(e.Keysym.Sym)),
			)
		}
	case *sdl.JoyButtonEvent:
		if e.State == sdl.PRESSED {
			il.startHold(
				mappedInput{code: int(e.Button), source: internal.SourceJoystick},
				fmt.Sprintf("Joystick Button: %d", int(e.Button)),
			)
		}
	case *sdl.ControllerButtonEvent:
		if e.State == sdl.PRESSED {
			il.startHold(
				mappedInput{code: int(e.Button), source: internal.SourceController},
				fmt.Sprintf("Game Controller: %d", int(e.Button)),
			)
		}
	case *sdl.JoyAxisEvent:
		if internal.Abs(int(e.Value)) > 16000 {
			var source internal.Source
			var label string
			if e.Value > 16000 {
				source = internal.SourceJoystickAxisPositive
				label = fmt.Sprintf("Joystick Axis %d (Positive)", int(e.Axis))
			} else {
				source = internal.SourceJoystickAxisNegative
				label = fmt.Sprintf("Joystick Axis %d (Negative)", int(e.Axis))
			}
			il.startHold(
				mappedInput{code: int(e.Axis), source: source},
				label,
			)
		}
	case *sdl.JoyHatEvent:
		if e.Value != sdl.HAT_CENTERED {
			hatName := ""
			switch e.Value {
			case sdl.HAT_UP:
				hatName = "UP"
			case sdl.HAT_DOWN:
				hatName = "DOWN"
			case sdl.HAT_LEFT:
				hatName = "LEFT"
			case sdl.HAT_RIGHT:
				hatName = "RIGHT"
			}
			il.startHold(
				mappedInput{code: int(e.Value), source: internal.SourceHatSwitch},
				fmt.Sprintf("Hat Switch: %s (%d)", hatName, e.Value),
			)
		}
	}
}

func (il *inputCaptureController) startHold(input mappedInput, label string) {
	il.state = stateHolding
	il.holdInput = input
	il.holdInputLabel = label
	il.holdStartTime = time.Now()
	currentButton := il.buttonSequence[il.currentButtonIdx]
	internal.GetInternalLogger().Debug("Hold started",
		"button", currentButton.displayName,
		"input", label)
}

func (il *inputCaptureController) isHeldInputReleased(event sdl.Event) bool {
	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		if e.State == sdl.RELEASED && il.holdInput.source == internal.SourceKeyboard {
			return true
		}
	case *sdl.JoyButtonEvent:
		if e.State == sdl.RELEASED && il.holdInput.source == internal.SourceJoystick {
			return true
		}
	case *sdl.ControllerButtonEvent:
		if e.State == sdl.RELEASED && il.holdInput.source == internal.SourceController {
			return true
		}
	case *sdl.JoyAxisEvent:
		if (il.holdInput.source == internal.SourceJoystickAxisPositive || il.holdInput.source == internal.SourceJoystickAxisNegative) &&
			internal.Abs(int(e.Value)) < 5000 {
			return true
		}
	case *sdl.JoyHatEvent:
		if il.holdInput.source == internal.SourceHatSwitch && e.Value == sdl.HAT_CENTERED {
			return true
		}
	}
	return false
}

func (il *inputCaptureController) handleHoldingEvent(event sdl.Event) {
	if il.isHeldInputReleased(event) {
		il.state = stateReleasedEarly
		currentButton := il.buttonSequence[il.currentButtonIdx]
		internal.GetInternalLogger().Debug("Hold released early",
			"button", currentButton.displayName,
			"elapsed", time.Since(il.holdStartTime))
	}
}

// handleRegisteredEvent waits for the held input to be released before advancing.
func (il *inputCaptureController) handleRegisteredEvent(event sdl.Event) {
	if il.isHeldInputReleased(event) {
		il.currentButtonIdx++
		if il.currentButtonIdx >= len(il.buttonSequence) {
			internal.GetInternalLogger().Info("All buttons configured successfully",
				"totalConfigured", len(il.mappedButtons))
			il.completedTime = time.Now()
		}
		il.state = statePrompting
	}
}

func (il *inputCaptureController) render() {
	window := internal.GetWindow()
	renderer := window.Renderer

	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	il.mutex.Lock()
	defer il.mutex.Unlock()

	theme := internal.GetTheme()
	scaleFactor := internal.GetScaleFactor()
	windowWidth := window.GetWidth()
	windowHeight := window.GetHeight()
	maxTextWidth := windowWidth * 3 / 4

	titleFont := internal.Fonts.MediumFont
	bodyFont := internal.Fonts.SmallFont
	largeFont := internal.Fonts.LargeFont

	// Title — always shown
	titleY := int32(float32(50) * scaleFactor)
	internal.RenderMultilineText(renderer, il.title, titleFont, maxTextWidth, windowWidth/2, titleY, theme.TextColor)

	centerY := windowHeight / 2

	switch {
	case il.state == stateInstruction:
		// Instruction screen
		internal.RenderMultilineText(renderer, il.instructionText, bodyFont, maxTextWidth, windowWidth/2, centerY, theme.TextColor)
		hintY := windowHeight - int32(float32(80)*scaleFactor)
		internal.RenderMultilineText(renderer, "Press any button to continue", bodyFont, maxTextWidth, windowWidth/2, hintY, theme.HintColor)

	case il.currentButtonIdx >= len(il.buttonSequence):
		// Complete state
		internal.RenderMultilineText(renderer, il.completeText, bodyFont, maxTextWidth, windowWidth/2, centerY-int32(float32(20)*scaleFactor), theme.TextColor)
		completeText := fmt.Sprintf("%d of %d buttons mapped", len(il.buttonSequence), len(il.buttonSequence))
		internal.RenderMultilineText(renderer, completeText, bodyFont, maxTextWidth, windowWidth/2, centerY+int32(float32(20)*scaleFactor), theme.HintColor)

	default:
		currentButton := il.buttonSequence[il.currentButtonIdx]

		// Progress counter
		progressText := fmt.Sprintf("Button %d of %d", il.currentButtonIdx+1, len(il.buttonSequence))
		progressY := titleY + int32(float32(50)*scaleFactor)
		internal.RenderMultilineText(renderer, progressText, bodyFont, maxTextWidth, windowWidth/2, progressY, theme.HintColor)

		// Button name — large, dead center, always visible
		internal.RenderMultilineText(renderer, currentButton.displayName, largeFont, maxTextWidth, windowWidth/2, centerY, theme.TextColor)

		// Progress bar area — below the button name
		barY := centerY + int32(float32(50)*scaleFactor)

		switch il.state {
		case statePrompting:
			// Nothing extra — just the button name

		case stateReleasedEarly:
			releasedColor := sdl.Color{R: 200, G: 100, B: 100, A: 255}
			internal.RenderMultilineText(renderer, il.releasedEarlyText, bodyFont, maxTextWidth, windowWidth/2, barY, releasedColor)

		case stateHolding:
			il.renderProgressBar(renderer, windowWidth, barY, false)

		case stateRegistered:
			il.renderProgressBar(renderer, windowWidth, barY, true)
		}
	}

	window.Present()
}

func (il *inputCaptureController) renderProgressBar(renderer *sdl.Renderer, windowWidth, barY int32, complete bool) {
	scaleFactor := internal.GetScaleFactor()
	theme := internal.GetTheme()

	barWidth := windowWidth * 7 / 10
	if barWidth > 900 {
		barWidth = 900
	}
	barHeight := int32(float32(18) * scaleFactor)
	barX := (windowWidth - barWidth) / 2

	var fillWidth int32
	var fillColor sdl.Color

	if complete {
		fillWidth = barWidth
		fillColor = sdl.Color{R: 100, G: 200, B: 100, A: 255} // Green when registered
	} else {
		elapsed := time.Since(il.holdStartTime)
		progress := float64(elapsed) / float64(il.holdDuration)
		if progress > 1.0 {
			progress = 1.0
		}
		fillWidth = int32(float64(barWidth) * progress)
		fillColor = theme.AccentColor
	}

	bgRect := sdl.Rect{X: barX, Y: barY, W: barWidth, H: barHeight}
	internal.DrawSmoothProgressBar(renderer, &bgRect, fillWidth, sdl.Color{R: 50, G: 50, B: 50, A: 255}, fillColor)
}

// buildMapping converts the collected button mappings into an InputMapping struct
// that can be used by the input processor or saved to JSON.
func (il *inputCaptureController) buildMapping() *internal.InputMapping {
	il.mutex.Lock()
	defer il.mutex.Unlock()

	mapping := &internal.InputMapping{
		KeyboardMap:         make(map[sdl.Keycode]constants.VirtualButton),
		ControllerButtonMap: make(map[sdl.GameControllerButton]constants.VirtualButton),
		ControllerHatMap:    make(map[uint8]constants.VirtualButton),
		JoystickAxisMap:     make(map[uint8]internal.JoystickAxisMapping),
		JoystickButtonMap:   make(map[uint8]constants.VirtualButton),
		JoystickHatMap:      make(map[uint8]constants.VirtualButton),
	}

	for button, input := range il.mappedButtons {
		switch input.source {
		case internal.SourceKeyboard:
			mapping.KeyboardMap[sdl.Keycode(input.code)] = button
		case internal.SourceController:
			mapping.ControllerButtonMap[sdl.GameControllerButton(input.code)] = button
		case internal.SourceJoystick:
			mapping.JoystickButtonMap[uint8(input.code)] = button
		case internal.SourceJoystickAxisPositive:
			axisMapping := mapping.JoystickAxisMap[uint8(input.code)]
			axisMapping.PositiveButton = button
			axisMapping.Threshold = 16000
			mapping.JoystickAxisMap[uint8(input.code)] = axisMapping
		case internal.SourceJoystickAxisNegative:
			axisMapping := mapping.JoystickAxisMap[uint8(input.code)]
			axisMapping.NegativeButton = button
			axisMapping.Threshold = 16000
			mapping.JoystickAxisMap[uint8(input.code)] = axisMapping
		case internal.SourceHatSwitch:
			mapping.JoystickHatMap[uint8(input.code)] = button
		}
	}

	internal.GetInternalLogger().Info("Input mapping complete",
		"totalMapped", len(il.mappedButtons),
		"keyboardMappings", len(mapping.KeyboardMap),
		"controllerButtonMappings", len(mapping.ControllerButtonMap),
		"joystickButtonMappings", len(mapping.JoystickButtonMap),
		"joystickAxisMappings", len(mapping.JoystickAxisMap),
		"hatSwitchMappings", len(mapping.JoystickHatMap))

	return mapping
}

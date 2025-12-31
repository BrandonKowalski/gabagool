package gabagool

import (
	"fmt"
	"sync"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type buttonConfig struct {
	internalButton constants.VirtualButton
	displayName    string
}

type mappedInput struct {
	code   int
	source internal.Source
}

type inputLoggerController struct {
	lastInput         string
	lastButtonName    string
	font              *ttf.Font
	textColor         sdl.Color
	currentButtonIdx  int
	mappedButtons     map[constants.VirtualButton]mappedInput
	mutex             sync.Mutex
	buttonSequence    []buttonConfig
	lastInputTime     time.Time
	debounceDelay     time.Duration
	waitingForRelease bool
}

func newInputLogger() *inputLoggerController {
	return &inputLoggerController{
		lastInput:         "",
		lastButtonName:    "",
		font:              internal.Fonts.LargeFont,
		textColor:         sdl.Color{R: 200, G: 100, B: 255, A: 255},
		mappedButtons:     make(map[constants.VirtualButton]mappedInput),
		currentButtonIdx:  0,
		debounceDelay:     1000 * time.Millisecond,
		waitingForRelease: false,
		buttonSequence: []buttonConfig{
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
			//{InternalButtonF1, "F1"},
			//{InternalButtonF2, "F2"},
			//{InternalButtonVolumeUp, "Volume Up"},
			//{InternalButtonVolumeDown, "Volume Down"},
			//{InternalButtonPower, "Power"},
		},
	}
}

func InputLogger() *internal.InputMapping {
	logger := newInputLogger()

	internal.GetInternalLogger().Info("Input logger started", "totalButtons", len(logger.buttonSequence))

	running := true

	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
			default:
				running = logger.handleEvent(event)
			}
		}

		logger.render()
		sdl.Delay(16)
	}

	return logger.buildMapping()
}

func (il *inputLoggerController) handleEvent(event sdl.Event) bool {
	il.mutex.Lock()
	defer il.mutex.Unlock()

	// Check if we're done with all buttons
	if il.currentButtonIdx >= len(il.buttonSequence) {
		internal.GetInternalLogger().Info("All buttons configured successfully",
			"totalConfigured", len(il.mappedButtons))
		return false
	}

	// Check debounce - if we're waiting for the button to be released or debounce time hasn't passed, ignore
	if il.waitingForRelease {
		// Wait for button release
		switch e := event.(type) {
		case *sdl.KeyboardEvent:
			if e.State == sdl.RELEASED {
				il.waitingForRelease = false
			}
		case *sdl.JoyButtonEvent:
			if e.State == sdl.RELEASED {
				il.waitingForRelease = false
			}
		case *sdl.ControllerButtonEvent:
			if e.State == sdl.RELEASED {
				il.waitingForRelease = false
			}
		case *sdl.JoyAxisEvent:
			// Axis has to go back to center
			if internal.Abs(int(e.Value)) < 5000 {
				il.waitingForRelease = false
			}
		case *sdl.JoyHatEvent:
			if e.Value == sdl.HAT_CENTERED {
				il.waitingForRelease = false
			}
		}
		return true
	}

	currentButton := il.buttonSequence[il.currentButtonIdx]

	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		if e.State == sdl.PRESSED {
			// Exit on ESC key
			if e.Keysym.Scancode == sdl.SCANCODE_ESCAPE {
				internal.GetInternalLogger().Info("Input logger cancelled by user (ESC pressed)")
				return false
			}
			// Store the KEYCODE (Sym), not the scancode
			il.lastInput = fmt.Sprintf("Keyboard: %d", int(e.Keysym.Sym))
			il.lastButtonName = "Registered!"
			il.mappedButtons[currentButton.internalButton] = mappedInput{
				code:   int(e.Keysym.Sym),
				source: internal.SourceKeyboard,
			}
			il.lastInputTime = time.Now()
			il.waitingForRelease = true
			internal.GetInternalLogger().Debug("Registered keyboard button",
				"button", currentButton.displayName,
				"keycode", e.Keysym.Sym,
				"scancode", e.Keysym.Scancode)
			il.advanceToNextButton()
		}
	case *sdl.JoyButtonEvent:
		if e.State == sdl.PRESSED {
			il.lastInput = fmt.Sprintf("Joystick Button: %d", int(e.Button))
			il.lastButtonName = "Registered!"
			il.mappedButtons[currentButton.internalButton] = mappedInput{
				code:   int(e.Button),
				source: internal.SourceJoystick,
			}
			il.lastInputTime = time.Now()
			il.waitingForRelease = true
			internal.GetInternalLogger().Debug("Registered joystick button",
				"button", currentButton.displayName,
				"joystickButton", e.Button)
			il.advanceToNextButton()
		}
	case *sdl.ControllerButtonEvent:
		if e.State == sdl.PRESSED {
			il.lastInput = fmt.Sprintf("Game Controller: %d", int(e.Button))
			il.lastButtonName = "Registered!"
			il.mappedButtons[currentButton.internalButton] = mappedInput{
				code:   int(e.Button),
				source: internal.SourceController,
			}
			il.lastInputTime = time.Now()
			il.waitingForRelease = true
			internal.GetInternalLogger().Debug("Registered controller button",
				"button", currentButton.displayName,
				"controllerButton", e.Button)
			il.advanceToNextButton()
		}
	case *sdl.JoyAxisEvent:
		// Only log significant axis movements
		if internal.Abs(int(e.Value)) > 16000 {
			var source internal.Source
			var direction string
			if e.Value > 16000 {
				il.lastInput = fmt.Sprintf("Joystick Axis %d (Positive)", int(e.Axis))
				source = internal.SourceJoystickAxisPositive
				direction = "Positive"
			} else {
				il.lastInput = fmt.Sprintf("Joystick Axis %d (Negative)", int(e.Axis))
				source = internal.SourceJoystickAxisNegative
				direction = "Negative"
			}
			il.lastButtonName = "Registered!"
			il.mappedButtons[currentButton.internalButton] = mappedInput{
				code:   int(e.Axis),
				source: source,
			}
			il.lastInputTime = time.Now()
			il.waitingForRelease = true
			internal.GetInternalLogger().Debug("Registered joystick axis",
				"button", currentButton.displayName,
				"axis", e.Axis,
				"direction", direction,
				"value", e.Value)
			il.advanceToNextButton()
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
			il.lastInput = fmt.Sprintf("Hat Switch: %s (%d)", hatName, e.Value)
			il.lastButtonName = "Registered!"
			il.mappedButtons[currentButton.internalButton] = mappedInput{
				code:   int(e.Value), // Store the hat VALUE (direction), not the hat index
				source: internal.SourceHatSwitch,
			}
			il.lastInputTime = time.Now()
			il.waitingForRelease = true
			internal.GetInternalLogger().Debug("Registered hat switch",
				"button", currentButton.displayName,
				"hat", e.Hat,
				"direction", hatName,
				"value", e.Value)
			il.advanceToNextButton()
		}
	}
	return true
}

func (il *inputLoggerController) advanceToNextButton() {
	il.currentButtonIdx++
	il.lastInput = ""
	il.lastButtonName = ""
}

func (il *inputLoggerController) render() {
	renderer := internal.GetWindow().Renderer

	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	il.mutex.Lock()
	defer il.mutex.Unlock()

	title := "Gabagool Input Configuration"
	il.renderText(renderer, title, internal.GetWindow().GetWidth()/2, 50, true)

	if il.currentButtonIdx >= len(il.buttonSequence) {
		completionText := "Configuration Complete!"
		il.renderText(renderer, completionText, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2-40, true)
		instructionText := "Press any button to exit"
		il.renderText(renderer, instructionText, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2+40, true)
	} else {
		currentButton := il.buttonSequence[il.currentButtonIdx]
		progressText := fmt.Sprintf("Button %d of %d", il.currentButtonIdx+1, len(il.buttonSequence))
		il.renderText(renderer, progressText, internal.GetWindow().GetWidth()/2, 120, true)

		buttonPrompt := fmt.Sprintf("Press: %s", currentButton.displayName)
		il.renderText(renderer, buttonPrompt, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2-60, true)

		if il.lastInput != "" {
			il.renderText(renderer, il.lastInput, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2-20, true)
			if il.waitingForRelease {
				il.renderText(renderer, "Release button...", internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2+20, true)
			} else {
				il.renderText(renderer, il.lastButtonName, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()/2+20, true)
			}
		}

		escapeHint := "Press ESC to cancel"
		il.renderText(renderer, escapeHint, internal.GetWindow().GetWidth()/2, internal.GetWindow().GetHeight()-80, true)
	}

	renderer.Present()
}

func (il *inputLoggerController) buildMapping() *internal.InputMapping {
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

	// Populate the mapping based on each button's individual source
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

func (il *inputLoggerController) renderText(renderer *sdl.Renderer, text string, x, y int32, centered bool) {
	surface, err := il.font.RenderUTF8Blended(text, il.textColor)
	if err != nil {
		internal.GetInternalLogger().Error("Failed to render text surface", "text", text, "error", err)
		return
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		internal.GetInternalLogger().Error("Failed to create texture from surface", "text", text, "error", err)
		return
	}
	defer texture.Destroy()

	_, _, w, h, err := texture.Query()
	if err != nil {
		internal.GetInternalLogger().Error("Failed to query texture", "text", text, "error", err)
		return
	}

	finalX := x
	if centered {
		finalX = x - w/2
	}

	dstRect := sdl.Rect{X: finalX, Y: y, W: w, H: h}
	renderer.Copy(texture, nil, &dstRect)
}

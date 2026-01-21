package internal

import (
	"fmt"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/sdl"
)

var globalInputProcessor *Processor
var gameControllers []*sdl.GameController
var rawJoysticks []*sdl.Joystick

func InitInputProcessor() {
	globalInputProcessor = NewInputProcessor()

	numJoysticks := sdl.NumJoysticks()
	GetInternalLogger().Debug("Detecting controllers", "joystick_count", numJoysticks)

	for i := 0; i < numJoysticks; i++ {
		if sdl.IsGameController(i) {
			controller := sdl.GameControllerOpen(i)
			if controller != nil {
				name := controller.Name()

				// Register this joystick INDEX (not instance ID) as being handled by a game controller
				globalInputProcessor.RegisterGameControllerJoystickIndex(i)

				GetInternalLogger().Debug("Opened game controller",
					"index", i,
					"name", name,
				)

				gameControllers = append(gameControllers, controller)
			} else {
				GetInternalLogger().Error("Failed to open game controller", "index", i)
			}
		} else {
			joystick := sdl.JoystickOpen(i)
			if joystick != nil {
				name := joystick.Name()
				GetInternalLogger().Debug("Opened raw joystick (not a standard game controller)",
					"index", i,
					"name", name,
				)

				rawJoysticks = append(rawJoysticks, joystick)
			} else {
				GetInternalLogger().Debug("Failed to open raw joystick", "index", i)
			}
		}
	}

	GetInternalLogger().Debug("Controller detection complete",
		"game_controllers", len(gameControllers),
		"raw_joysticks", len(rawJoysticks),
		"total_joysticks", numJoysticks,
	)
}

func GetInputProcessor() *Processor {
	return globalInputProcessor
}

// Processor handles input mapping from physical devices to virtual buttons.
// It processes SDL events and converts them to a unified input format,
// supporting keyboard, game controllers, joysticks, and hat switches.
// Also handles combo detection (chords and sequences).
type Processor struct {
	mapping                       *InputMapping
	gameControllerJoystickIndices map[int]bool
	axisStates                    map[uint8]int8  // tracks which direction each axis is pressed: -1 (negative), 0 (none), 1 (positive)
	hatStates                     map[uint8]uint8 // tracks the current hat position
	eventQueue                    []*Event        // queue for events that need to be processed

	// Combo detection state
	buttonStates     map[constants.VirtualButton]buttonState // tracks press times for each button
	registeredCombos []registeredCombo                       // all registered combos
	comboEventQueue  []*ComboEvent                           // queue for combo events
	sequenceBuffer   []sequenceEntry                         // recent button presses for sequence detection
}

// buttonState tracks when a button was pressed
type buttonState struct {
	Pressed   bool
	PressTime time.Time
}

// registeredCombo holds combo configuration
type registeredCombo struct {
	ID       string
	Type     ComboType
	Buttons  []constants.VirtualButton
	Chord    *ChordOptions
	Sequence *SequenceOptions
	active   bool // For chords: currently held. For sequences: just triggered (reset on next press).
}

// sequenceEntry records a button press for sequence matching
type sequenceEntry struct {
	Button constants.VirtualButton
	Time   time.Time
}

func NewInputProcessor() *Processor {
	return &Processor{
		mapping:                       GetInputMapping(),
		gameControllerJoystickIndices: make(map[int]bool),
		axisStates:                    make(map[uint8]int8),
		hatStates:                     make(map[uint8]uint8),
		buttonStates:                  make(map[constants.VirtualButton]buttonState),
		registeredCombos:              make([]registeredCombo, 0),
	}
}

func (ip *Processor) RegisterGameControllerJoystickIndex(joystickIndex int) {
	ip.gameControllerJoystickIndices[joystickIndex] = true
}

func (ip *Processor) IsGameControllerJoystick(joystickIndex int) bool {
	return ip.gameControllerJoystickIndices[joystickIndex]
}

// RegisterChord registers a chord combination (multiple buttons pressed simultaneously)
func (ip *Processor) RegisterChord(id string, buttons []constants.VirtualButton, opts ChordOptions) error {
	if len(buttons) < 2 {
		return fmt.Errorf("chord requires at least 2 buttons")
	}
	if opts.Window == 0 {
		opts.Window = 100 * time.Millisecond
	}

	ip.registeredCombos = append(ip.registeredCombos, registeredCombo{
		ID:      id,
		Type:    ComboTypeChord,
		Buttons: buttons,
		Chord:   &opts,
	})
	return nil
}

// RegisterSequence registers a sequence combination (buttons pressed in order)
func (ip *Processor) RegisterSequence(id string, buttons []constants.VirtualButton, opts SequenceOptions) error {
	if len(buttons) < 2 {
		return fmt.Errorf("sequence requires at least 2 buttons")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 500 * time.Millisecond
	}

	ip.registeredCombos = append(ip.registeredCombos, registeredCombo{
		ID:       id,
		Type:     ComboTypeSequence,
		Buttons:  buttons,
		Sequence: &opts,
	})
	return nil
}

// UnregisterCombo removes a combo by ID
func (ip *Processor) UnregisterCombo(id string) {
	for i, combo := range ip.registeredCombos {
		if combo.ID == id {
			ip.registeredCombos = append(ip.registeredCombos[:i], ip.registeredCombos[i+1:]...)
			return
		}
	}
}

// ClearCombos removes all registered combos
func (ip *Processor) ClearCombos() {
	ip.registeredCombos = nil
	ip.sequenceBuffer = nil
}

// createEvent creates an Event and updates button state for combo detection
func (ip *Processor) createEvent(button constants.VirtualButton, pressed bool, source Source, rawCode int) *Event {
	ip.updateButtonState(button, pressed)
	return &Event{
		Button:  button,
		Pressed: pressed,
		Source:  source,
		RawCode: rawCode,
	}
}

func (ip *Processor) ProcessSDLEvent(event sdl.Event) *Event {
	// If there are queued events, return those first
	if len(ip.eventQueue) > 0 {
		evt := ip.eventQueue[0]
		ip.eventQueue = ip.eventQueue[1:]
		return evt
	}

	logger := GetInternalLogger()

	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		keyCode := e.Keysym.Sym
		keyName := sdl.GetKeyName(keyCode)
		if button, exists := ip.mapping.KeyboardMap[keyCode]; exists {
			if e.Type == sdl.KEYDOWN {
				logger.Debug("Keyboard input mapped",
					"physical", keyName,
					"keyCode", fmt.Sprintf("%s (%d)", keyName, keyCode),
					"virtualButton", button.GetName())
			}
			return ip.createEvent(button, e.Type == sdl.KEYDOWN, SourceKeyboard, int(keyCode))
		}
		logger.Debug("Keyboard input not mapped",
			"key_code", fmt.Sprintf("%s (%d)", keyName, keyCode),
			"mappingSize", len(ip.mapping.KeyboardMap))
	case *sdl.ControllerButtonEvent:
		buttonName := sdl.GameControllerGetStringForButton(sdl.GameControllerButton(e.Button))
		if button, exists := ip.mapping.ControllerButtonMap[sdl.GameControllerButton(e.Button)]; exists {
			if e.Type == sdl.CONTROLLERBUTTONDOWN {
				logger.Debug("Controller button mapped",
					"physical", buttonName,
					"buttonCode", fmt.Sprintf("%s (%d)", buttonName, e.Button),
					"virtualButton", button.GetName())
			}
			return ip.createEvent(button, e.Type == sdl.CONTROLLERBUTTONDOWN, SourceController, int(e.Button))
		}
		logger.Debug("Controller button not mapped",
			"button_code", fmt.Sprintf("%s (%d)", buttonName, e.Button))
	case *sdl.JoyHatEvent:
		previousValue := ip.hatStates[e.Hat]
		ip.hatStates[e.Hat] = e.Value

		// If previous direction was set and different from new value, generate release event
		if previousValue != sdl.HAT_CENTERED && previousValue != e.Value {
			if button, exists := ip.mapping.JoystickHatMap[previousValue]; exists {
				hatDirection := getHatDirectionName(previousValue)
				logger.Debug("Joy hat released (direction change)",
					"hat_value", fmt.Sprintf("%s (%d)", hatDirection, previousValue),
					"virtual_button", button.GetName())

				// If new direction is also mapped, queue the press event
				if e.Value != sdl.HAT_CENTERED {
					if newButton, exists := ip.mapping.JoystickHatMap[e.Value]; exists {
						newHatDirection := getHatDirectionName(e.Value)
						logger.Debug("Joy hat pressed (queued)",
							"hat_value", fmt.Sprintf("%s (%d)", newHatDirection, e.Value),
							"virtual_button", newButton.GetName())
						// Queue the press event with combo tracking
						ip.updateButtonState(newButton, true)
						ip.eventQueue = append(ip.eventQueue, &Event{
							Button:  newButton,
							Pressed: true,
							Source:  SourceHatSwitch,
							RawCode: int(e.Value),
						})
					}
				}

				return ip.createEvent(button, false, SourceHatSwitch, int(previousValue))
			}
		}

		// If hat moved to a new direction (and previous was centered), generate press event
		if e.Value != sdl.HAT_CENTERED {
			hatDirection := getHatDirectionName(e.Value)
			if button, exists := ip.mapping.JoystickHatMap[e.Value]; exists {
				logger.Debug("Joy hat mapped",
					"hat_value", fmt.Sprintf("%s (%d)", hatDirection, e.Value),
					"virtual_button", button.GetName())
				return ip.createEvent(button, true, SourceHatSwitch, int(e.Value))
			}
			logger.Debug("Joy hat not mapped",
				"hat_value", fmt.Sprintf("%s (%d)", hatDirection, e.Value))
		}

		// If hat returned to center from a direction
		if e.Value == sdl.HAT_CENTERED && previousValue != sdl.HAT_CENTERED {
			if button, exists := ip.mapping.JoystickHatMap[previousValue]; exists {
				hatDirection := getHatDirectionName(previousValue)
				logger.Debug("Joy hat released (centered)",
					"hat_value", fmt.Sprintf("%s (%d)", hatDirection, previousValue),
					"virtual_button", button.GetName())
				return ip.createEvent(button, false, SourceHatSwitch, int(previousValue))
			}
		}
	case *sdl.ControllerAxisEvent:
		axisName := sdl.GameControllerGetStringForAxis(sdl.GameControllerAxis(e.Axis))
		if axisConfig, exists := ip.mapping.JoystickAxisMap[e.Axis]; exists {
			previousState := ip.axisStates[e.Axis]
			var newState int8 = 0

			// Determine new state
			if e.Value > axisConfig.Threshold {
				newState = 1
			} else if e.Value < -axisConfig.Threshold {
				newState = -1
			}

			// If state changed, generate appropriate event
			if newState != previousState {
				ip.axisStates[e.Axis] = newState

				// Generate release event for previous state
				if previousState == 1 {
					logger.Debug("Controller axis positive released",
						"axis_code", fmt.Sprintf("%s+ (%d)", axisName, e.Axis),
						"value", e.Value,
						"virtual_button", axisConfig.PositiveButton.GetName())
					return ip.createEvent(axisConfig.PositiveButton, false, SourceController, int(e.Axis))
				} else if previousState == -1 {
					logger.Debug("Controller axis negative released",
						"axis_code", fmt.Sprintf("%s- (%d)", axisName, e.Axis),
						"value", e.Value,
						"virtual_button", axisConfig.NegativeButton.GetName())
					return ip.createEvent(axisConfig.NegativeButton, false, SourceController, int(e.Axis))
				}

				// Generate press event for new state
				if newState == 1 {
					logger.Debug("Controller axis positive threshold exceeded",
						"axis_code", fmt.Sprintf("%s+ (%d)", axisName, e.Axis),
						"value", e.Value,
						"threshold", axisConfig.Threshold,
						"virtual_button", axisConfig.PositiveButton.GetName())
					return ip.createEvent(axisConfig.PositiveButton, true, SourceController, int(e.Axis))
				} else if newState == -1 {
					logger.Debug("Controller axis negative threshold exceeded",
						"axis_code", fmt.Sprintf("%s- (%d)", axisName, e.Axis),
						"value", e.Value,
						"threshold", axisConfig.Threshold,
						"virtual_button", axisConfig.NegativeButton.GetName())
					return ip.createEvent(axisConfig.NegativeButton, true, SourceController, int(e.Axis))
				}
			}
		}
		logger.Debug("Controller axis not mapped or threshold not exceeded",
			"axis_code", fmt.Sprintf("%s (%d)", axisName, e.Axis),
			"value", e.Value)
	case *sdl.JoyButtonEvent:
		joyButtonName := getJoyButtonName(e.Button)
		if button, exists := ip.mapping.JoystickButtonMap[e.Button]; exists {
			logger.Debug("Joy button mapped",
				"button_code", fmt.Sprintf("%s (%d)", joyButtonName, e.Button),
				"virtual_button", button.GetName())
			return ip.createEvent(button, e.Type == sdl.JOYBUTTONDOWN, SourceJoystick, int(e.Button))
		}
		logger.Debug("Joy button not mapped",
			"button_code", fmt.Sprintf("%s (%d)", joyButtonName, e.Button))
	case *sdl.JoyAxisEvent:
		joyAxisName := getJoyAxisName(e.Axis)
		if axisConfig, exists := ip.mapping.JoystickAxisMap[e.Axis]; exists {
			previousState := ip.axisStates[e.Axis]
			var newState int8 = 0

			// Determine new state
			if e.Value > axisConfig.Threshold {
				newState = 1
			} else if e.Value < -axisConfig.Threshold {
				newState = -1
			}

			// If state changed, generate appropriate event
			if newState != previousState {
				ip.axisStates[e.Axis] = newState

				// Generate release event for previous state
				if previousState == 1 {
					logger.Debug("Joy axis positive released",
						"axis_code", fmt.Sprintf("%s+ (%d)", joyAxisName, e.Axis),
						"value", e.Value,
						"virtual_button", axisConfig.PositiveButton.GetName())
					return ip.createEvent(axisConfig.PositiveButton, false, SourceJoystick, int(e.Axis))
				} else if previousState == -1 {
					logger.Debug("Joy axis negative released",
						"axis_code", fmt.Sprintf("%s- (%d)", joyAxisName, e.Axis),
						"value", e.Value,
						"virtual_button", axisConfig.NegativeButton.GetName())
					return ip.createEvent(axisConfig.NegativeButton, false, SourceJoystick, int(e.Axis))
				}

				// Generate press event for new state
				if newState == 1 {
					logger.Debug("Joy axis positive threshold exceeded",
						"axis_code", fmt.Sprintf("%s+ (%d)", joyAxisName, e.Axis),
						"value", e.Value,
						"threshold", axisConfig.Threshold,
						"virtual_button", axisConfig.PositiveButton.GetName())
					return ip.createEvent(axisConfig.PositiveButton, true, SourceJoystick, int(e.Axis))
				} else if newState == -1 {
					logger.Debug("Joy axis negative threshold exceeded",
						"axis_code", fmt.Sprintf("%s- (%d)", joyAxisName, e.Axis),
						"value", e.Value,
						"threshold", axisConfig.Threshold,
						"virtual_button", axisConfig.NegativeButton.GetName())
					return ip.createEvent(axisConfig.NegativeButton, true, SourceJoystick, int(e.Axis))
				}
			}
		}
		logger.Debug("Joy axis not mapped or threshold not exceeded",
			"axis_code", fmt.Sprintf("%s (%d)", joyAxisName, e.Axis),
			"value", e.Value)
	}
	return nil
}

func getHatDirectionName(value uint8) string {
	switch value {
	case sdl.HAT_UP:
		return "Hat Up"
	case sdl.HAT_DOWN:
		return "Hat Down"
	case sdl.HAT_LEFT:
		return "Hat Left"
	case sdl.HAT_RIGHT:
		return "Hat Right"
	case sdl.HAT_LEFTUP:
		return "Hat Left Up"
	case sdl.HAT_LEFTDOWN:
		return "Hat Left Down"
	case sdl.HAT_RIGHTUP:
		return "Hat Right Up"
	case sdl.HAT_RIGHTDOWN:
		return "Hat Right Down"
	default:
		return "Hat Unknown"
	}
}

func getJoyButtonName(button uint8) string {
	return fmt.Sprintf("JoyButton%d", button)
}

func getJoyAxisName(axis uint8) string {
	return fmt.Sprintf("JoyAxis%d", axis)
}

func CloseAllControllers() {
	for _, controller := range gameControllers {
		if controller != nil {
			controller.Close()
		}
	}
	for _, joystick := range rawJoysticks {
		if joystick != nil {
			joystick.Close()
		}
	}
}

// ProcessComboEvent returns the next queued combo event, if any
func (ip *Processor) ProcessComboEvent() *ComboEvent {
	if len(ip.comboEventQueue) > 0 {
		evt := ip.comboEventQueue[0]
		ip.comboEventQueue = ip.comboEventQueue[1:]
		return evt
	}
	return nil
}

// updateButtonState updates tracking for a button and triggers combo checks
func (ip *Processor) updateButtonState(button constants.VirtualButton, pressed bool) {
	now := time.Now()

	ip.buttonStates[button] = buttonState{
		Pressed:   pressed,
		PressTime: now,
	}

	if pressed {
		ip.addToSequenceBuffer(button, now)
		ip.checkChords(now)
		ip.checkSequences(now)
	} else {
		ip.checkChordReleases()
	}
}

// addToSequenceBuffer adds a button press to the sequence buffer
func (ip *Processor) addToSequenceBuffer(button constants.VirtualButton, t time.Time) {
	ip.sequenceBuffer = append(ip.sequenceBuffer, sequenceEntry{
		Button: button,
		Time:   t,
	})

	// Limit buffer size to prevent unbounded growth
	maxBufferSize := 20
	if len(ip.sequenceBuffer) > maxBufferSize {
		ip.sequenceBuffer = ip.sequenceBuffer[len(ip.sequenceBuffer)-maxBufferSize:]
	}
}

// checkChords checks if any chord combinations are satisfied
func (ip *Processor) checkChords(now time.Time) {
	for i := range ip.registeredCombos {
		combo := &ip.registeredCombos[i]
		if combo.Type != ComboTypeChord || combo.active {
			continue
		}

		allPressed := true
		var earliestPress, latestPress time.Time

		for _, btn := range combo.Buttons {
			state, exists := ip.buttonStates[btn]
			if !exists || !state.Pressed {
				allPressed = false
				break
			}
			if earliestPress.IsZero() || state.PressTime.Before(earliestPress) {
				earliestPress = state.PressTime
			}
			if state.PressTime.After(latestPress) {
				latestPress = state.PressTime
			}
		}

		if allPressed && latestPress.Sub(earliestPress) <= combo.Chord.Window {
			combo.active = true
			ip.comboEventQueue = append(ip.comboEventQueue, &ComboEvent{
				ComboID:   combo.ID,
				ComboType: ComboTypeChord,
				Buttons:   combo.Buttons,
				Triggered: true,
			})
			// Call the callback if provided
			if combo.Chord.OnTrigger != nil {
				combo.Chord.OnTrigger()
			}
		}
	}
}

// checkChordReleases checks if any active chords are released
func (ip *Processor) checkChordReleases() {
	for i := range ip.registeredCombos {
		combo := &ip.registeredCombos[i]
		if combo.Type != ComboTypeChord || !combo.active {
			continue
		}

		for _, btn := range combo.Buttons {
			state, exists := ip.buttonStates[btn]
			if !exists || !state.Pressed {
				combo.active = false
				ip.comboEventQueue = append(ip.comboEventQueue, &ComboEvent{
					ComboID:   combo.ID,
					ComboType: ComboTypeChord,
					Buttons:   combo.Buttons,
					Triggered: false,
				})
				// Call the release callback if provided
				if combo.Chord.OnRelease != nil {
					combo.Chord.OnRelease()
				}
				break
			}
		}
	}
}

// checkSequences checks if any sequence combinations are complete
func (ip *Processor) checkSequences(now time.Time) {
	for i := range ip.registeredCombos {
		combo := &ip.registeredCombos[i]
		if combo.Type != ComboTypeSequence {
			continue
		}

		if ip.matchesSequence(combo, now) {
			ip.comboEventQueue = append(ip.comboEventQueue, &ComboEvent{
				ComboID:   combo.ID,
				ComboType: ComboTypeSequence,
				Buttons:   combo.Buttons,
				Triggered: true,
			})
			// Call the callback if provided
			if combo.Sequence.OnTrigger != nil {
				combo.Sequence.OnTrigger()
			}
			// Clear the matched portion from buffer to prevent re-triggering
			ip.sequenceBuffer = ip.sequenceBuffer[:0]
		}
	}
}

// matchesSequence checks if the sequence buffer matches a sequence combo
func (ip *Processor) matchesSequence(combo *registeredCombo, now time.Time) bool {
	if len(ip.sequenceBuffer) < len(combo.Buttons) {
		return false
	}

	// Check from the end of the buffer
	bufferStart := len(ip.sequenceBuffer) - len(combo.Buttons)

	for i, btn := range combo.Buttons {
		entry := ip.sequenceBuffer[bufferStart+i]

		if entry.Button != btn {
			return false
		}

		// Check timeout between entries
		if i > 0 {
			prevEntry := ip.sequenceBuffer[bufferStart+i-1]
			if entry.Time.Sub(prevEntry.Time) > combo.Sequence.Timeout {
				return false
			}
		}
	}

	// If strict mode, ensure no other buttons were pressed during the sequence window
	if combo.Sequence.Strict && bufferStart > 0 {
		firstEntryTime := ip.sequenceBuffer[bufferStart].Time
		// Check if any earlier entries are within the timeout window of the sequence start
		for j := bufferStart - 1; j >= 0; j-- {
			if now.Sub(ip.sequenceBuffer[j].Time) < combo.Sequence.Timeout*time.Duration(len(combo.Buttons)) {
				return false
			}
		}
		_ = firstEntryTime // used for documentation purposes
	}

	return true
}

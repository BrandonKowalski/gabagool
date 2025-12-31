package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/sdl"
)

const MappingPathEnvVar = "INPUT_MAPPING_PATH"

var inputMappingBytes []byte

func SetInputMappingBytes(data []byte) {
	inputMappingBytes = data
}

type Source int

const (
	SourceKeyboard Source = iota
	SourceController
	SourceJoystick
	SourceJoystickAxisPositive
	SourceJoystickAxisNegative
	SourceHatSwitch
)

type Event struct {
	Button  constants.VirtualButton
	Pressed bool
	Source  Source
	RawCode int
}

// ComboType distinguishes between chord and sequence combinations
type ComboType int

const (
	ComboTypeChord ComboType = iota
	ComboTypeSequence
)

// ComboEvent represents a triggered button combination
type ComboEvent struct {
	ComboID   string                    // Unique identifier for the combo
	ComboType ComboType                 // Chord or Sequence
	Buttons   []constants.VirtualButton // Buttons involved in the combo
	Triggered bool                      // true on activation, false on release (chords only)
}

// ComboCallback is called when a combo is triggered or released
type ComboCallback func()

// ChordOptions configures chord detection behavior
type ChordOptions struct {
	Window    time.Duration // Time window for simultaneous press (default: 100ms)
	OnTrigger ComboCallback // Called when the chord is activated (all buttons pressed)
	OnRelease ComboCallback // Called when the chord is released (any button released)
}

// SequenceOptions configures sequence detection behavior
type SequenceOptions struct {
	Timeout   time.Duration // Max time between sequence inputs (default: 500ms)
	Strict    bool          // If true, sequence breaks on any non-sequence button
	OnTrigger ComboCallback // Called when the sequence is completed
}

type JoystickAxisMapping struct {
	PositiveButton constants.VirtualButton
	NegativeButton constants.VirtualButton
	Threshold      int16
}

type InputMapping struct {
	KeyboardMap map[sdl.Keycode]constants.VirtualButton

	ControllerButtonMap map[sdl.GameControllerButton]constants.VirtualButton

	ControllerHatMap map[uint8]constants.VirtualButton

	JoystickAxisMap map[uint8]JoystickAxisMapping

	JoystickButtonMap map[uint8]constants.VirtualButton

	JoystickHatMap map[uint8]constants.VirtualButton
}

type Mapping struct {
	KeyboardMap map[int]int `json:"keyboard_map"`

	ControllerButtonMap map[int]int `json:"controller_button_map"`

	ControllerHatMap map[int]int `json:"controller_hat_map"`

	JoystickAxisMap map[int]struct {
		PositiveButton int   `json:"positive_button"`
		NegativeButton int   `json:"negative_button"`
		Threshold      int16 `json:"threshold"`
	} `json:"joystick_axis_map"`

	JoystickButtonMap map[int]int `json:"joystick_button_map"`

	JoystickHatMap map[int]int `json:"joystick_hat_map"`
}

func DefaultInputMapping() *InputMapping {
	return &InputMapping{
		KeyboardMap: map[sdl.Keycode]constants.VirtualButton{
			sdl.K_UP:     constants.VirtualButtonUp,
			sdl.K_DOWN:   constants.VirtualButtonDown,
			sdl.K_LEFT:   constants.VirtualButtonLeft,
			sdl.K_RIGHT:  constants.VirtualButtonRight,
			sdl.K_a:      constants.VirtualButtonA,
			sdl.K_b:      constants.VirtualButtonB,
			sdl.K_x:      constants.VirtualButtonX,
			sdl.K_y:      constants.VirtualButtonY,
			sdl.K_l:      constants.VirtualButtonL1,
			sdl.K_r:      constants.VirtualButtonR1,
			sdl.K_RETURN: constants.VirtualButtonStart,
			sdl.K_SPACE:  constants.VirtualButtonSelect,
			sdl.K_h:      constants.VirtualButtonMenu,
		},
		ControllerButtonMap: map[sdl.GameControllerButton]constants.VirtualButton{
			sdl.CONTROLLER_BUTTON_DPAD_UP:       constants.VirtualButtonUp,
			sdl.CONTROLLER_BUTTON_DPAD_DOWN:     constants.VirtualButtonDown,
			sdl.CONTROLLER_BUTTON_DPAD_LEFT:     constants.VirtualButtonLeft,
			sdl.CONTROLLER_BUTTON_DPAD_RIGHT:    constants.VirtualButtonRight,
			sdl.CONTROLLER_BUTTON_A:             constants.VirtualButtonB,
			sdl.CONTROLLER_BUTTON_B:             constants.VirtualButtonA,
			sdl.CONTROLLER_BUTTON_X:             constants.VirtualButtonY,
			sdl.CONTROLLER_BUTTON_Y:             constants.VirtualButtonX,
			sdl.CONTROLLER_BUTTON_LEFTSHOULDER:  constants.VirtualButtonL1,
			sdl.CONTROLLER_BUTTON_RIGHTSHOULDER: constants.VirtualButtonR1,
			sdl.CONTROLLER_BUTTON_START:         constants.VirtualButtonStart,
			sdl.CONTROLLER_BUTTON_BACK:          constants.VirtualButtonSelect,
			sdl.CONTROLLER_BUTTON_GUIDE:         constants.VirtualButtonMenu,
		},
	}
}

// GetInputMapping returns the input mapping from embedded bytes if set,
// from the environment variable if set, otherwise returns the default mapping
func GetInputMapping() *InputMapping {
	logger := GetInternalLogger()

	if len(inputMappingBytes) > 0 {
		mapping, err := LoadInputMappingFromBytes(inputMappingBytes)
		if err == nil {
			logger.Info("Loaded custom input mapping from embedded bytes")
			return mapping
		}
		logger.Warn("Failed to load custom input mapping from bytes, trying file path", "error", err)
	}

	mappingPath := os.Getenv(MappingPathEnvVar)
	if mappingPath != "" {
		mapping, err := LoadInputMappingFromJSON(mappingPath)
		if err == nil {
			logger.Info("Loaded custom input mapping from environment variable", "path", mappingPath)
			return mapping
		}
		logger.Warn("Failed to load custom input mapping, using default", "path", mappingPath, "error", err)
	}
	return DefaultInputMapping()
}

func LoadInputMappingFromJSON(filePath string) (*InputMapping, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}
	return LoadInputMappingFromBytes(data)
}

func LoadInputMappingFromBytes(data []byte) (*InputMapping, error) {
	var serializableMapping Mapping
	err := json.Unmarshal(data, &serializableMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	mapping := &InputMapping{
		KeyboardMap:         make(map[sdl.Keycode]constants.VirtualButton),
		ControllerButtonMap: make(map[sdl.GameControllerButton]constants.VirtualButton),
		ControllerHatMap:    make(map[uint8]constants.VirtualButton),
		JoystickAxisMap:     make(map[uint8]JoystickAxisMapping),
		JoystickButtonMap:   make(map[uint8]constants.VirtualButton),
		JoystickHatMap:      make(map[uint8]constants.VirtualButton),
	}

	if serializableMapping.KeyboardMap != nil {
		for keyCode, button := range serializableMapping.KeyboardMap {
			mapping.KeyboardMap[sdl.Keycode(keyCode)] = constants.VirtualButton(button)
		}
	}

	if serializableMapping.ControllerButtonMap != nil {
		for button, vb := range serializableMapping.ControllerButtonMap {
			mapping.ControllerButtonMap[sdl.GameControllerButton(button)] = constants.VirtualButton(vb)
		}
	}

	if serializableMapping.ControllerHatMap != nil {
		for hat, button := range serializableMapping.ControllerHatMap {
			mapping.ControllerHatMap[uint8(hat)] = constants.VirtualButton(button)
		}
	}

	if serializableMapping.JoystickAxisMap != nil {
		for axis, axisMapping := range serializableMapping.JoystickAxisMap {
			mapping.JoystickAxisMap[uint8(axis)] = JoystickAxisMapping{
				PositiveButton: constants.VirtualButton(axisMapping.PositiveButton),
				NegativeButton: constants.VirtualButton(axisMapping.NegativeButton),
				Threshold:      axisMapping.Threshold,
			}
		}
	}

	if serializableMapping.JoystickButtonMap != nil {
		for button, vb := range serializableMapping.JoystickButtonMap {
			mapping.JoystickButtonMap[uint8(button)] = constants.VirtualButton(vb)
		}
	}

	if serializableMapping.JoystickHatMap != nil {
		for hat, button := range serializableMapping.JoystickHatMap {
			mapping.JoystickHatMap[uint8(hat)] = constants.VirtualButton(button)
		}
	}

	return mapping, nil
}

// ToJSON converts the InputMapping to JSON bytes in the export format.
// Keys are SDL codes, values are VirtualButton iota values.
func (im *InputMapping) ToJSON() ([]byte, error) {
	serializableMapping := &Mapping{
		KeyboardMap:         make(map[int]int),
		ControllerButtonMap: make(map[int]int),
		ControllerHatMap:    make(map[int]int),
		JoystickAxisMap: make(map[int]struct {
			PositiveButton int   `json:"positive_button"`
			NegativeButton int   `json:"negative_button"`
			Threshold      int16 `json:"threshold"`
		}),
		JoystickButtonMap: make(map[int]int),
		JoystickHatMap:    make(map[int]int),
	}

	for keyCode, button := range im.KeyboardMap {
		serializableMapping.KeyboardMap[int(keyCode)] = int(button)
	}

	for button, vb := range im.ControllerButtonMap {
		serializableMapping.ControllerButtonMap[int(button)] = int(vb)
	}

	for hat, button := range im.ControllerHatMap {
		serializableMapping.ControllerHatMap[int(hat)] = int(button)
	}

	for axis, axisMapping := range im.JoystickAxisMap {
		serializableMapping.JoystickAxisMap[int(axis)] = struct {
			PositiveButton int   `json:"positive_button"`
			NegativeButton int   `json:"negative_button"`
			Threshold      int16 `json:"threshold"`
		}{
			PositiveButton: int(axisMapping.PositiveButton),
			NegativeButton: int(axisMapping.NegativeButton),
			Threshold:      axisMapping.Threshold,
		}
	}

	for button, vb := range im.JoystickButtonMap {
		serializableMapping.JoystickButtonMap[int(button)] = int(vb)
	}

	for hat, button := range im.JoystickHatMap {
		serializableMapping.JoystickHatMap[int(hat)] = int(button)
	}

	return json.MarshalIndent(serializableMapping, "", "  ")
}

func (im *InputMapping) SaveToJSON(filePath string) error {
	data, err := im.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal mapping to JSON: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

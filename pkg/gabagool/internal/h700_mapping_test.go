package internal

import (
	"testing"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/sdl"
)

func withNextUI(t *testing.T, enabled bool) {
	t.Helper()
	t.Setenv("PLATFORM", "h700")
	originalMode := isNextUIMode
	isNextUIMode = enabled
	t.Cleanup(func() { isNextUIMode = originalMode })
}

func TestH700NextUIRawMapping(t *testing.T) {
	withNextUI(t, true)

	mapping := platformDefaultInputMapping()
	for button, want := range map[uint8]constants.VirtualButton{
		0: constants.VirtualButtonB, 1: constants.VirtualButtonA,
		2: constants.VirtualButtonY, 3: constants.VirtualButtonX,
		4: constants.VirtualButtonL1, 5: constants.VirtualButtonR1,
		6: constants.VirtualButtonSelect, 7: constants.VirtualButtonStart,
		8: constants.VirtualButtonMenu, 13: constants.VirtualButtonVolumeDown,
		14: constants.VirtualButtonVolumeUp,
	} {
		if got := mapping.JoystickButtonMap[button]; got != want {
			t.Errorf("button %d = %s, want %s", button, got.GetName(), want.GetName())
		}
	}
	for _, button := range []uint8{9, 10, 15, 16} {
		if got, ok := mapping.JoystickButtonMap[button]; ok {
			t.Errorf("button %d = %s, want unmapped", button, got.GetName())
		}
	}
	for axis, want := range map[uint8]constants.VirtualButton{2: constants.VirtualButtonL2, 5: constants.VirtualButtonR2} {
		if got := mapping.JoystickAxisMap[axis]; got.PositiveButton != want || got.NegativeButton != constants.VirtualButtonUnassigned {
			t.Errorf("axis %d = %+v, want positive %s only", axis, got, want.GetName())
		}
	}
	if got := mapping.JoystickHatMap[sdl.HAT_UP]; got != constants.VirtualButtonUp {
		t.Errorf("hat up = %s, want Up", got.GetName())
	}
}

func TestH700NextUIOpensBuiltInPadAsRawJoystick(t *testing.T) {
	withNextUI(t, true)
	if !opensAsRawJoystick("ANBERNIC-keys") {
		t.Error("built-in pad opens as a game controller, want raw joystick")
	}
	if opensAsRawJoystick("Xbox Wireless Controller") {
		t.Error("external pad opens as raw joystick, want game controller")
	}

	withNextUI(t, false)
	if opensAsRawJoystick("ANBERNIC-keys") {
		t.Error("built-in pad without NextUI opens as raw joystick, want game controller")
	}
}

func TestH700WithoutNextUIMapping(t *testing.T) {
	withNextUI(t, false)

	mapping := platformDefaultInputMapping()
	for button, want := range map[uint8]constants.VirtualButton{
		3: constants.VirtualButtonA, 4: constants.VirtualButtonB,
		5: constants.VirtualButtonY, 6: constants.VirtualButtonX,
	} {
		if got := mapping.JoystickButtonMap[button]; got != want {
			t.Errorf("button %d = %s, want %s", button, got.GetName(), want.GetName())
		}
	}
}

func TestPositiveOnlyAxisFromRest(t *testing.T) {
	withNextUI(t, true)
	ip := &Processor{
		mapping:      platformDefaultInputMapping(),
		axisStates:   make(map[uint8]int8),
		hatStates:    make(map[uint8]uint8),
		buttonStates: make(map[constants.VirtualButton]buttonState),
	}

	if evt := ip.ProcessSDLEvent(&sdl.JoyAxisEvent{Axis: 2, Value: -32768}); evt != nil {
		t.Fatalf("resting trigger = %+v, want no event", evt)
	}
	if evt := ip.ProcessSDLEvent(&sdl.JoyAxisEvent{Axis: 2, Value: 32767}); evt == nil || evt.Button != constants.VirtualButtonL2 || !evt.Pressed {
		t.Fatalf("pressed trigger = %+v, want L2 down", evt)
	}
	if evt := ip.ProcessSDLEvent(&sdl.JoyAxisEvent{Axis: 2, Value: -32768}); evt == nil || evt.Button != constants.VirtualButtonL2 || evt.Pressed {
		t.Fatalf("released trigger = %+v, want L2 up", evt)
	}
}

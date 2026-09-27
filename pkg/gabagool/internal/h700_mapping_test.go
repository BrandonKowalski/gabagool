package internal

import (
	"testing"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
)

// On an h700 the pad's buttons are numbered from 0 as the kernel reports them,
// which is how NextUI reads it. A muOS numbering, from 3, put B on A.
func TestH700DefaultMapping(t *testing.T) {
	t.Setenv("PLATFORM", "h700")

	mapping := platformDefaultInputMapping()
	for button, want := range map[uint8]constants.VirtualButton{
		0: constants.VirtualButtonA, 1: constants.VirtualButtonB,
		2: constants.VirtualButtonY, 3: constants.VirtualButtonX,
		4: constants.VirtualButtonL1, 5: constants.VirtualButtonR1,
		6: constants.VirtualButtonSelect, 7: constants.VirtualButtonStart,
		8: constants.VirtualButtonMenu, 9: constants.VirtualButtonL2,
		10: constants.VirtualButtonR2, 13: constants.VirtualButtonUp,
		16: constants.VirtualButtonDown,
	} {
		if got := mapping.JoystickButtonMap[button]; got != want {
			t.Errorf("button %d = %s, want %s", button, got.GetName(), want.GetName())
		}
	}
}

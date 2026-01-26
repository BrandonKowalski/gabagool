// Package constants defines shared constants, types, and configuration values
// used throughout the gabagool UI framework.
package constants

import (
	"os"
	"time"
)

// Development is the environment variable value for development mode.
const Development = "DEV"

// BackgroundPathEnvVar is the environment variable name for custom background image path.
const BackgroundPathEnvVar = "BACKGROUND_PATH"

// IsDevMode returns true if running in development mode (ENVIRONMENT=DEV).
func IsDevMode() bool {
	return os.Getenv("ENVIRONMENT") == Development
}

// VirtualButton represents an abstract input button, mapped from physical hardware.
// This abstraction allows gabagool to work with different controller configurations.
type VirtualButton int

const (
	VirtualButtonUnassigned VirtualButton = iota
	VirtualButtonUp
	VirtualButtonDown
	VirtualButtonLeft
	VirtualButtonRight
	VirtualButtonA
	VirtualButtonB
	VirtualButtonX
	VirtualButtonY
	VirtualButtonL1
	VirtualButtonL2
	VirtualButtonR1
	VirtualButtonR2
	VirtualButtonStart
	VirtualButtonSelect
	VirtualButtonMenu
	VirtualButtonF1
	VirtualButtonF2
	VirtualButtonVolumeUp
	VirtualButtonVolumeDown
	VirtualButtonPower
)

func (vb VirtualButton) GetName() string {
	switch vb {
	case VirtualButtonUnassigned:
		return "Unassigned"
	case VirtualButtonUp:
		return "Up"
	case VirtualButtonDown:
		return "Down"
	case VirtualButtonLeft:
		return "Left"
	case VirtualButtonRight:
		return "Right"
	case VirtualButtonA:
		return "A"
	case VirtualButtonB:
		return "B"
	case VirtualButtonX:
		return "X"
	case VirtualButtonY:
		return "Y"
	case VirtualButtonL1:
		return "L1"
	case VirtualButtonL2:
		return "L2"
	case VirtualButtonR1:
		return "R1"
	case VirtualButtonR2:
		return "R2"
	case VirtualButtonStart:
		return "Start"
	case VirtualButtonSelect:
		return "Select"
	case VirtualButtonMenu:
		return "Menu"
	case VirtualButtonF1:
		return "F1"
	case VirtualButtonF2:
		return "F2"
	case VirtualButtonVolumeUp:
		return "VolumeUp"
	case VirtualButtonVolumeDown:
		return "VolumeDown"
	case VirtualButtonPower:
		return "Power"
	default:
		return "Unknown"
	}
}

// TextAlign specifies horizontal text alignment.
type TextAlign int

const (
	TextAlignLeft   TextAlign = iota // Align text to the left edge
	TextAlignCenter                  // Center text horizontally
	TextAlignRight                   // Align text to the right edge
)

// Default timing and spacing constants.
const (
	DefaultInputDelay         = 20 * time.Millisecond // Debounce delay between input events
	DefaultTitleSpacing int32 = 5                     // Vertical spacing below title text
)

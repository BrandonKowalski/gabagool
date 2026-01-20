package internal

import (
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
)

// Direction represents a cardinal direction for navigation.
type Direction int

const (
	DirectionNone Direction = iota
	DirectionUp
	DirectionDown
	DirectionLeft
	DirectionRight
)

// DirectionalInput tracks held directions and handles repeat timing.
// Embed this in component controllers to get consistent directional
// input handling across all components.
type DirectionalInput struct {
	held struct {
		up, down, left, right bool
	}
	lastRepeatTime time.Time
	repeatDelay    time.Duration
	repeatInterval time.Duration
	hasRepeated    bool
}

// NewDirectionalInput creates a DirectionalInput with default timing.
// Default delay is 300ms before first repeat, then 50ms between repeats.
func NewDirectionalInput() DirectionalInput {
	return DirectionalInput{
		repeatDelay:    300 * time.Millisecond,
		repeatInterval: 50 * time.Millisecond,
		lastRepeatTime: time.Now(),
	}
}

// NewDirectionalInputWithTiming creates a DirectionalInput with custom timing.
func NewDirectionalInputWithTiming(delay, interval time.Duration) DirectionalInput {
	return DirectionalInput{
		repeatDelay:    delay,
		repeatInterval: interval,
		lastRepeatTime: time.Now(),
	}
}

// SetHeld updates the held state for a direction based on a virtual button.
// Returns true if the button was a directional button.
func (d *DirectionalInput) SetHeld(button constants.VirtualButton, held bool) bool {
	switch button {
	case constants.VirtualButtonUp:
		d.held.up = held
		if !held {
			d.hasRepeated = false
		}
		return true
	case constants.VirtualButtonDown:
		d.held.down = held
		if !held {
			d.hasRepeated = false
		}
		return true
	case constants.VirtualButtonLeft:
		d.held.left = held
		if !held {
			d.hasRepeated = false
		}
		return true
	case constants.VirtualButtonRight:
		d.held.right = held
		if !held {
			d.hasRepeated = false
		}
		return true
	}
	return false
}

// IsHeld returns true if any direction is currently held.
func (d *DirectionalInput) IsHeld() bool {
	return d.held.up || d.held.down || d.held.left || d.held.right
}

// HeldDirection returns the currently held direction.
// If multiple directions are held, priority is: up, down, left, right.
// Returns DirectionNone if no direction is held.
func (d *DirectionalInput) HeldDirection() Direction {
	if d.held.up {
		return DirectionUp
	}
	if d.held.down {
		return DirectionDown
	}
	if d.held.left {
		return DirectionLeft
	}
	if d.held.right {
		return DirectionRight
	}
	return DirectionNone
}

// Update checks if a repeat event should fire based on timing.
// Call this every frame. It returns the direction that should be processed,
// or DirectionNone if no repeat should occur.
//
// The first repeat occurs after repeatDelay, subsequent repeats after repeatInterval.
func (d *DirectionalInput) Update() Direction {
	if !d.IsHeld() {
		d.lastRepeatTime = time.Now()
		d.hasRepeated = false
		return DirectionNone
	}

	timeSince := time.Since(d.lastRepeatTime)

	// Use repeatDelay for first repeat, then repeatInterval for subsequent repeats
	threshold := d.repeatInterval
	if !d.hasRepeated {
		threshold = d.repeatDelay
	}

	if timeSince >= threshold {
		d.lastRepeatTime = time.Now()
		d.hasRepeated = true
		return d.HeldDirection()
	}

	return DirectionNone
}

// Reset clears all held directions and timing state.
func (d *DirectionalInput) Reset() {
	d.held.up = false
	d.held.down = false
	d.held.left = false
	d.held.right = false
	d.hasRepeated = false
	d.lastRepeatTime = time.Now()
}

// VirtualButtonFor returns the VirtualButton constant for a Direction.
func (d Direction) VirtualButton() constants.VirtualButton {
	switch d {
	case DirectionUp:
		return constants.VirtualButtonUp
	case DirectionDown:
		return constants.VirtualButtonDown
	case DirectionLeft:
		return constants.VirtualButtonLeft
	case DirectionRight:
		return constants.VirtualButtonRight
	default:
		return constants.VirtualButton(0)
	}
}

// String returns a string representation of the direction.
func (d Direction) String() string {
	switch d {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	case DirectionLeft:
		return "left"
	case DirectionRight:
		return "right"
	default:
		return ""
	}
}

# gabagool

A Go-based UI library for building graphical interfaces on retro gaming handhelds that support SDL2.

![Bring it here](./.github/resources/gabagool.gif)
> 🇮🇹 (Chase, Grey, & HBO Home Entertainment, 1999–2007) 🇮🇹

---

## Features

- **Type-safe FSM navigation** - Context-based finite state machine for multiscreen flows
- **Rich UI components** - Lists, keyboards, dialogs, detail screens, and more
- **Advanced input handling** - Chord/sequence detection, configurable button mapping
- **Thread-safe updates** - Atomic operations for progress bars, status text, visibility
- **Responsive design** - Automatic scaling based on screen resolution
- **Image support** - PNG, JPEG, and SVG rendering with scaling

---

## Installation

### Prerequisites

#### macOS (Homebrew)

```bash
brew install sdl2 sdl2_image sdl2_ttf
```

### Install the package

```bash
go get github.com/BrandonKowalski/gabagool/v2
```

---

## Quick Start

```go
package main

import "github.com/BrandonKowalski/gabagool/v2"

func main() {
	gabagool.Init(gabagool.Options{
		WindowTitle:    "My App",
		ShowBackground: true,
		IsNextUI:       true,
	})
	defer gabagool.Close()

	// Create a simple list
	list := gabagool.NewList(gabagool.ListOptions{
		Title: "Main Menu",
		Items: []gabagool.ListItem{
			{Label: "Option 1"},
			{Label: "Option 2"},
			{Label: "Option 3"},
		},
	})

	// Run the FSM
	fsm := gabagool.NewFSM()
	fsm.AddNode("main", list)
	fsm.Run("main")
}
```

---

## Components

### List

Scrollable list with multi-select, reordering, and customizable appearance.

### Detail Screen

Rich content display with slideshows, metadata sections, descriptions, and images.

### Keyboard

Text input with multiple layouts (QWERTY, URL-optimized, numeric) and symbol modes.

### Option List

Settings menu with toggles, text input, color pickers, and clickable items.

### Confirmation Message

Dialog with customizable confirm/cancel buttons and optional imagery.

### Selection Message

Multiple choice dialog with descriptions for each option.

### Process Message

Loading screen for async operations with optional progress bar.

### Color Picker

Hexagonal grid color selector with 25 distinguishable colors.

### Download Manager

Multi-threaded downloads with progress tracking and speed calculation.

### Status Bar

Top-right display for clock, battery, WiFi, and custom icons.

---

## Input System

Gabagool abstracts physical controller inputs into virtual buttons. 

This can be controlled by mapping files.

### Button Combos

```go
// Chord: buttons pressed simultaneously
processor.AddChord([]constants.VirtualButton{constants.L1, constants.R1}, func () {
// triggered when L1+R1 pressed together
})

// Sequence: buttons pressed in order
processor.AddSequence([]constants.VirtualButton{constants.A, constants.B, constants.A}, func () {
// triggered after A -> B -> A sequence
})
```

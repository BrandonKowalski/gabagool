<!-- trunk-ignore-all(markdownlint/MD033) -->
<!-- trunk-ignore(markdownlint/MD041) -->
<div align="center">

# gabagool

A Go-based UI library for building graphical interfaces on retro gaming handhelds that support SDL2.

![Bring it here](./.github/resources/gabagool.gif)

> 🇮🇹 (Chase, Grey, & HBO Home Entertainment, 1999–2007) 🇮🇹

[![license-badge-img]][license-badge]
[![godoc-badge-img]][godoc-badge]
[![stars-badge-img]][stars-badge]

</div>

---

## Features

- **Router-based navigation** - Type-safe screen transitions with explicit data flow
- **Rich UI components** - Lists, keyboards, dialogs, detail screens, and more
- **Advanced input handling** - Chord/sequence detection, configurable button mapping
- **Thread-safe updates** - Atomic operations for progress bars, status text, visibility
- **Responsive design** - Automatic scaling based on screen resolution
- **Image support** - PNG, JPEG, and SVG rendering with scaling
- **Platform support** - NextUI and Cannoli CFW theming integration

---

## Installation

### Prerequisites

#### macOS (Homebrew)

```bash
brew install sdl2 sdl2_image sdl2_ttf sdl2_gfx
```

#### Linux (Debian/Ubuntu)

```bash
sudo apt-get install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-gfx-dev
```

### Install the package

```bash
go get github.com/BrandonKowalski/gabagool/v2
```

---

## Quick Start

```go
package main

import (
	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

func main() {
	gaba.Init(gaba.Options{
		WindowTitle:    "My App",
		ShowBackground: true,
		IsNextUI:       true, // or IsCannoli: true
	})
	defer gaba.Close()

	// Display a simple list
	result, err := gaba.List(gaba.DefaultListOptions("Main Menu", []gaba.MenuItem{
		{Text: "Option 1"},
		{Text: "Option 2"},
		{Text: "Option 3"},
	}))

	if err == gaba.ErrCancelled {
		// User pressed back
		return
	}

	// result.SelectedIndex contains the selected item
	_ = result
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

Gabagool abstracts physical controller inputs into virtual buttons, allowing the same code to work across different devices.

### Custom Input Mapping

Load a custom mapping from JSON:

```go
// From embedded bytes
gaba.SetInputMappingBytes(mappingJSON)

// Or via environment variable
// INPUT_MAPPING_PATH=/path/to/mapping.json
```

### Button Combos

Register chord (simultaneous) or sequence (ordered) button combinations:

```go
import (
	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
)

// Chord: buttons pressed simultaneously
gaba.RegisterChord("quick_menu", []constants.VirtualButton{
	constants.VirtualButtonL1,
	constants.VirtualButtonR1,
}, gaba.ChordOptions{
	OnTrigger: func() {
		// triggered when L1+R1 pressed together
	},
})

// Sequence: buttons pressed in order (like Konami code)
gaba.RegisterSequence("secret", []constants.VirtualButton{
	constants.VirtualButtonUp,
	constants.VirtualButtonUp,
	constants.VirtualButtonDown,
	constants.VirtualButtonDown,
}, gaba.SequenceOptions{
	OnTrigger: func() {
		// triggered after sequence completes
	},
})
```

---

## Multi-Screen Navigation

For apps with multiple screens, use the `router` package for type-safe navigation with explicit data flow:

```go
import (
	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/router"
)

// Define screen identifiers
const (
	ScreenList router.Screen = iota
	ScreenDetail
)

func main() {
	gaba.Init(gaba.Options{WindowTitle: "My App", IsNextUI: true})
	defer gaba.Close()

	r := router.New()

	// Register screen handlers
	r.Register(ScreenList, func(input any) (any, error) {
		return showList(input.(ListInput)), nil
	})

	r.Register(ScreenDetail, func(input any) (any, error) {
		return showDetail(input.(DetailInput)), nil
	})

	// Define navigation transitions
	r.OnTransition(func(from router.Screen, result any, stack *router.Stack) (router.Screen, any) {
		switch from {
		case ScreenList:
			r := result.(ListResult)
			if r.Action == ActionSelected {
				stack.Push(from, input, r.Resume) // Save for back navigation
				return ScreenDetail, DetailInput{Item: r.Selected}
			}
		case ScreenDetail:
			// Pop returns to previous screen with resume state
			entry := stack.Pop()
			return entry.Screen, entry.Input
		}
		return router.ScreenExit, nil
	})

	r.Run(ScreenList, ListInput{})
}
```

See the `router` package documentation for complete examples.

<!-- Badges - Italian flag colors: Green (#009246), White (#F4F5F0), Red (#CE2B37) -->

[license-badge-img]: https://img.shields.io/github/license/BrandonKowalski/gabagool?style=for-the-badge&color=009246

[license-badge]: LICENSE

[godoc-badge-img]: https://img.shields.io/badge/godoc-reference-F4F5F0?style=for-the-badge

[godoc-badge]: https://pkg.go.dev/github.com/BrandonKowalski/gabagool/v2

[stars-badge-img]: https://img.shields.io/github/stars/BrandonKowalski/gabagool?style=for-the-badge&color=CE2B37

[stars-badge]: https://github.com/BrandonKowalski/gabagool/stargazers

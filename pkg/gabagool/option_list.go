package gabagool

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
)

type OptionType int

const (
	OptionTypeStandard OptionType = iota
	OptionTypeKeyboard
	OptionTypeClickable
	OptionTypeColorPicker // New option type for the color picker
)

// Option represents a single option for a menu item.
// DisplayName is the text that will be displayed to the user.
// Value is the value that will be returned when the option is submitted.
// Type controls the option's behavior. There are four types:
//   - Standard: A standard option that will be displayed to the user.
//   - Keyboard: A keyboard option that will be displayed to the user.
//   - Clickable: A clickable option that will be displayed to the user.
//   - ColorPicker: A hexagonal color picker for selecting colors.
//
// KeyboardPrompt is the text that will be displayed to the user when the option is a keyboard option.
// For ColorPicker type, Value should be an sdl.Color.
type Option struct {
	DisplayName    string
	Value          interface{}
	Type           OptionType
	KeyboardPrompt string
	KeyboardLayout KeyboardLayout // Layout to use for keyboard input (default: KeyboardLayoutGeneral)
	URLShortcuts   []URLShortcut  // Custom shortcuts for URL keyboard (up to 10, only used when KeyboardLayout is KeyboardLayoutURL)
	Masked         bool
	OnUpdate       func(newValue interface{})
}

type OptionListSettings struct {
	InitialSelectedIndex  int
	VisibleStartIndex     int
	DisableBackButton     bool
	UseSmallTitle         bool
	FooterHelpItems       []FooterHelpItem
	HelpExitText          string
	ActionButton          constants.VirtualButton
	SecondaryActionButton constants.VirtualButton
	ConfirmButton         constants.VirtualButton // Default: VirtualButtonStart
	StatusBar             StatusBarOptions
	ListPickerButton      constants.VirtualButton // Button to show a list picker for standard options
}

// ItemWithOptions represents a menu item with multiple choices.
// Item is the menu item itself.
// Options is the list of options for the menu item.
// SelectedOption is the index of the currently selected option.
// Visible is an optional function that determines if the item should be shown.
// If nil, the item is always visible.
// VisibleWhen is an atomic bool that can be toggled dynamically (e.g., by another option's OnUpdate).
// If set, it takes precedence over Visible.
type ItemWithOptions struct {
	Item           MenuItem
	Options        []Option
	SelectedOption int
	Visible        func() bool  // nil = always visible
	VisibleWhen    *atomic.Bool // if set, takes precedence over Visible
	colorPicker    *ColorPicker
}

func (iow *ItemWithOptions) Value() interface{} {
	if iow.Options[iow.SelectedOption].Value == nil {
		return ""
	}

	return fmt.Sprintf("%s", iow.Options[iow.SelectedOption].Value)
}

// IsVisible returns whether the item should be displayed.
// If VisibleWhen is set, it takes precedence.
// Otherwise, returns true if Visible is nil or if Visible() returns true.
func (iow *ItemWithOptions) IsVisible() bool {
	if iow.VisibleWhen != nil {
		return iow.VisibleWhen.Load()
	}
	if iow.Visible == nil {
		return true
	}
	return iow.Visible()
}

// OptionsListResult represents the return value of the OptionsList function.
// Items is the entire list of menu items.
// Selected is the index of the selected item.
// VisibleStartIndex is the index of the first visible item in the list.
// Action is the action taken when exiting (Selected, Triggered, SecondaryTriggered, or Confirmed).
type OptionsListResult struct {
	Items             []ItemWithOptions
	Selected          int
	VisibleStartIndex int
	Action            ListAction
}
type internalOptionsListSettings struct {
	Margins               internal.Padding
	ItemSpacing           int32
	InputDelay            time.Duration
	Title                 string
	TitleAlign            constants.TextAlign
	TitleSpacing          int32
	UseSmallTitle         bool
	ScrollSpeed           float32
	ScrollPauseTime       int
	FooterHelpItems       []FooterHelpItem
	FooterColor           sdl.Color
	DisableBackButton     bool
	HelpExitText          string
	ActionButton          constants.VirtualButton
	SecondaryActionButton constants.VirtualButton
	ConfirmButton         constants.VirtualButton
	StatusBar             StatusBarOptions
	ListPickerButton      constants.VirtualButton
}

type optionsListController struct {
	Items         []ItemWithOptions
	SelectedIndex int
	Settings      internalOptionsListSettings
	StartY        int32
	lastInputTime time.Time
	OnSelect      func(index int, item *ItemWithOptions)

	VisibleStartIndex int
	MaxVisibleItems   int

	HelpEnabled bool
	helpOverlay *helpOverlay
	ShowingHelp bool

	itemScrollData       map[int]*internal.TextScrollData
	showingColorPicker   bool
	activeColorPickerIdx int

	directionalInput internal.DirectionalInput
}

func defaultOptionsListSettings(title string) internalOptionsListSettings {
	return internalOptionsListSettings{
		Margins:         internal.UniformPadding(20),
		ItemSpacing:     60,
		InputDelay:      constants.DefaultInputDelay,
		Title:           title,
		TitleAlign:      constants.TextAlignLeft,
		TitleSpacing:    constants.DefaultTitleSpacing,
		ScrollSpeed:     150.0,
		ScrollPauseTime: 25,
		FooterColor:     sdl.Color{R: 180, G: 180, B: 180, A: 255},
		FooterHelpItems: []FooterHelpItem{},
		ConfirmButton:   constants.VirtualButtonStart,
		StatusBar:       DefaultStatusBarOptions(),
	}
}

func newOptionsListController(title string, items []ItemWithOptions) *optionsListController {
	selectedIndex := 0

	for i, item := range items {
		if item.Item.Selected {
			selectedIndex = i
			break
		}
	}

	// Ensure selected item is visible; if not, find first visible item
	if len(items) > 0 && !items[selectedIndex].IsVisible() {
		for i := range items {
			if items[i].IsVisible() {
				selectedIndex = i
				break
			}
		}
	}

	for i := range items {
		items[i].Item.Selected = i == selectedIndex
	}

	for i := range items {
		for j, opt := range items[i].Options {
			if opt.Type == OptionTypeColorPicker {
				// Initialize with the default color if not already Set
				if opt.Value == nil {
					items[i].Options[j].Value = sdl.Color{R: 255, G: 255, B: 255, A: 255}
				}

				// Create the color picker
				window := internal.GetWindow()
				items[i].colorPicker = NewHexColorPicker(window)

				// Initialize with the current color value if it's a sdl.Color
				if color, ok := opt.Value.(sdl.Color); ok {
					colorFound := false
					for idx, pickerColor := range items[i].colorPicker.Colors {
						if pickerColor.R == color.R && pickerColor.G == color.G && pickerColor.B == color.B {
							items[i].colorPicker.SelectedIndex = idx
							colorFound = true
							break
						}
					}
					// If color not found in the predefined list, we could add it
					if !colorFound {
						// TODO: Add custom color to the list or leave as is
					}
				}

				items[i].colorPicker.setVisible(false)

				items[i].colorPicker.setOnColorSelected(func(color sdl.Color) {
					items[i].Options[j].Value = color
					items[i].Options[j].DisplayName = fmt.Sprintf("#%02X%02X%02X", color.R, color.G, color.B)

					if items[i].Options[j].OnUpdate != nil {
						items[i].Options[j].OnUpdate(color)
					}
				})

				break
			}
		}
	}

	return &optionsListController{
		Items:                items,
		SelectedIndex:        selectedIndex,
		Settings:             defaultOptionsListSettings(title),
		StartY:               20,
		lastInputTime:        time.Now(),
		itemScrollData:       make(map[int]*internal.TextScrollData),
		showingColorPicker:   false,
		activeColorPickerIdx: -1,
		directionalInput:     internal.NewDirectionalInputWithTiming(150*time.Millisecond, 50*time.Millisecond),
	}
}

// OptionsList presents a list of options to the user.
// This blocks until a selection is made or the user cancels.
func OptionsList(title string, listOptions OptionListSettings, items []ItemWithOptions) (*OptionsListResult, error) {
	window := internal.GetWindow()
	renderer := window.Renderer
	processor := internal.GetInputProcessor()

	optionsListController := newOptionsListController(title, items)

	optionsListController.MaxVisibleItems = int(optionsListController.calculateMaxVisibleItems(window))
	optionsListController.Settings.FooterHelpItems = listOptions.FooterHelpItems
	optionsListController.Settings.DisableBackButton = listOptions.DisableBackButton
	optionsListController.Settings.UseSmallTitle = listOptions.UseSmallTitle
	optionsListController.Settings.HelpExitText = listOptions.HelpExitText
	optionsListController.Settings.ActionButton = listOptions.ActionButton
	optionsListController.Settings.SecondaryActionButton = listOptions.SecondaryActionButton
	optionsListController.Settings.StatusBar = listOptions.StatusBar
	optionsListController.Settings.ListPickerButton = listOptions.ListPickerButton

	// Use provided ConfirmButton or default to VirtualButtonStart
	if listOptions.ConfirmButton != constants.VirtualButtonUnassigned {
		optionsListController.Settings.ConfirmButton = listOptions.ConfirmButton
	}

	if listOptions.InitialSelectedIndex > 0 && listOptions.InitialSelectedIndex < len(items) {
		if optionsListController.SelectedIndex >= 0 && optionsListController.SelectedIndex < len(items) {
			optionsListController.Items[optionsListController.SelectedIndex].Item.Selected = false
		}
		optionsListController.SelectedIndex = listOptions.InitialSelectedIndex
		optionsListController.Items[listOptions.InitialSelectedIndex].Item.Selected = true
	}

	if listOptions.VisibleStartIndex >= 0 && listOptions.VisibleStartIndex < len(items) {
		optionsListController.VisibleStartIndex = listOptions.VisibleStartIndex
	}

	running := true
	cancelled := false
	result := OptionsListResult{
		Items:    items,
		Selected: -1,
		Action:   ListActionSelected,
	}

	var err error

	for running {
		if event := sdl.WaitEventTimeout(16); event != nil {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
				err = sdl.GetError()

			case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
				inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
				if inputEvent == nil {
					continue
				}

				if inputEvent.Pressed {
					if optionsListController.showingColorPicker {
						optionsListController.handleColorPickerInput(inputEvent)
					} else {
						optionsListController.handleOptionsInput(inputEvent, &running, &result, &cancelled)
					}
				} else {
					optionsListController.handleInputEventRelease(inputEvent)
				}
			}
		}

		optionsListController.handleDirectionalRepeats()

		if window.Background != nil {
			window.RenderBackground()
		} else {
			renderer.SetDrawColor(0, 0, 0, 255)
			renderer.Clear()
		}

		// If showing the color picker, draw it; otherwise draw just the option list
		if optionsListController.showingColorPicker &&
			optionsListController.activeColorPickerIdx >= 0 &&
			optionsListController.activeColorPickerIdx < len(optionsListController.Items) {
			item := &optionsListController.Items[optionsListController.activeColorPickerIdx]
			if item.colorPicker != nil {
				item.colorPicker.draw(renderer)
			}
		} else {
			optionsListController.render(renderer)
		}

		window.Present()
	}

	if err != nil {
		return nil, err
	}

	if cancelled {
		return nil, ErrCancelled
	}

	result.VisibleStartIndex = optionsListController.VisibleStartIndex
	return &result, nil
}

func (olc *optionsListController) calculateMaxVisibleItems(window *internal.Window) int32 {
	scaleFactor := internal.GetScaleFactor()

	itemSpacing := int32(float32(60) * scaleFactor)

	_, screenHeight, _ := window.Renderer.GetOutputSize()

	var titleHeight int32 = 0
	if olc.Settings.Title != "" {
		if olc.Settings.UseSmallTitle {
			titleHeight = int32(float32(50) * scaleFactor)
		} else {
			titleHeight = int32(float32(60) * scaleFactor)
		}
		titleHeight += olc.Settings.TitleSpacing
	}

	footerHeight := int32(float32(50) * scaleFactor)

	availableHeight := screenHeight - titleHeight - footerHeight - olc.StartY

	maxItems := availableHeight / itemSpacing

	if maxItems < 1 {
		maxItems = 1
	}

	return maxItems
}

func (olc *optionsListController) handleColorPickerInput(inputEvent *internal.Event) {
	if !inputEvent.Pressed {
		return
	}

	if olc.activeColorPickerIdx < 0 || olc.activeColorPickerIdx >= len(olc.Items) {
		return
	}

	item := &olc.Items[olc.activeColorPickerIdx]
	if item.colorPicker == nil {
		return
	}

	switch inputEvent.Button {
	case constants.VirtualButtonB:
		olc.hideColorPicker()
	case constants.VirtualButtonA:
		selectedColor := item.colorPicker.getSelectedColor()
		for j := range item.Options {
			if item.Options[j].Type == OptionTypeColorPicker {
				item.Options[j].Value = selectedColor
				item.Options[j].DisplayName = fmt.Sprintf("#%02X%02X%02X",
					selectedColor.R, selectedColor.G, selectedColor.B)
				if item.Options[j].OnUpdate != nil {
					item.Options[j].OnUpdate(selectedColor)
				}
				break
			}
		}
		olc.hideColorPicker()
	case constants.VirtualButtonLeft, constants.VirtualButtonRight, constants.VirtualButtonUp, constants.VirtualButtonDown:
		var keycode sdl.Keycode
		switch inputEvent.Button {
		case constants.VirtualButtonLeft:
			keycode = sdl.K_LEFT
		case constants.VirtualButtonRight:
			keycode = sdl.K_RIGHT
		case constants.VirtualButtonUp:
			keycode = sdl.K_UP
		case constants.VirtualButtonDown:
			keycode = sdl.K_DOWN
		}
		item.colorPicker.handleKeyPress(keycode)

		selectedColor := item.colorPicker.getSelectedColor()
		for j := range item.Options {
			if item.Options[j].Type == OptionTypeColorPicker && item.Options[j].OnUpdate != nil {
				item.Options[j].OnUpdate(selectedColor)
				break
			}
		}
	}
}

func (olc *optionsListController) handleOptionsInput(inputEvent *internal.Event, running *bool, result *OptionsListResult, cancelled *bool) {
	if !inputEvent.Pressed {
		return
	}

	currentTime := time.Now()
	if currentTime.Sub(olc.lastInputTime) < olc.Settings.InputDelay {
		return
	}

	switch inputEvent.Button {
	case constants.VirtualButtonMenu:
		olc.toggleHelp()
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonB:
		if olc.ShowingHelp {
			olc.ShowingHelp = false
		} else if !olc.Settings.DisableBackButton {
			*running = false
			*cancelled = true
		}
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonA:
		if olc.ShowingHelp {
			olc.ShowingHelp = false
		} else {
			olc.handleAButton(running, result)
		}
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonLeft:
		olc.directionalInput.SetHeld(inputEvent.Button, true)
		if !olc.ShowingHelp {
			olc.cycleOptionLeft()
		}
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonRight:
		olc.directionalInput.SetHeld(inputEvent.Button, true)
		if !olc.ShowingHelp {
			olc.cycleOptionRight()
		}
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonUp:
		olc.directionalInput.SetHeld(inputEvent.Button, true)
		if olc.ShowingHelp {
			olc.scrollHelpOverlay(-1)
		} else {
			olc.moveSelection(-1)
		}
		olc.lastInputTime = time.Now()

	case constants.VirtualButtonDown:
		olc.directionalInput.SetHeld(inputEvent.Button, true)
		if olc.ShowingHelp {
			olc.scrollHelpOverlay(1)
		} else {
			olc.moveSelection(1)
		}
		olc.lastInputTime = time.Now()

	default:
		// Handle configurable action buttons
		if olc.Settings.ConfirmButton != constants.VirtualButtonUnassigned &&
			inputEvent.Button == olc.Settings.ConfirmButton {
			if !olc.ShowingHelp && olc.SelectedIndex >= 0 && olc.SelectedIndex < len(olc.Items) {
				*running = false
				result.Action = ListActionConfirmed
				result.Selected = olc.SelectedIndex
			}
			olc.lastInputTime = time.Now()
		}

		if olc.Settings.ActionButton != constants.VirtualButtonUnassigned &&
			inputEvent.Button == olc.Settings.ActionButton {
			if !olc.ShowingHelp && olc.SelectedIndex >= 0 && olc.SelectedIndex < len(olc.Items) {
				*running = false
				result.Action = ListActionTriggered
				result.Selected = olc.SelectedIndex
			}
			olc.lastInputTime = time.Now()
		}

		if olc.Settings.SecondaryActionButton != constants.VirtualButtonUnassigned &&
			inputEvent.Button == olc.Settings.SecondaryActionButton {
			if !olc.ShowingHelp && olc.SelectedIndex >= 0 && olc.SelectedIndex < len(olc.Items) {
				*running = false
				result.Action = ListActionSecondaryTriggered
				result.Selected = olc.SelectedIndex
			}
			olc.lastInputTime = time.Now()
		}

		if olc.Settings.ListPickerButton != constants.VirtualButtonUnassigned &&
			inputEvent.Button == olc.Settings.ListPickerButton {
			if !olc.ShowingHelp && olc.SelectedIndex >= 0 && olc.SelectedIndex < len(olc.Items) {
				olc.showListPicker()
			}
			olc.lastInputTime = time.Now()
		}
	}
}

func (olc *optionsListController) handleInputEventRelease(inputEvent *internal.Event) {
	olc.directionalInput.SetHeld(inputEvent.Button, false)
}

func (olc *optionsListController) handleDirectionalRepeats() {
	dir := olc.directionalInput.Update()
	if dir == internal.DirectionNone {
		return
	}

	switch dir {
	case internal.DirectionUp:
		if olc.ShowingHelp {
			olc.scrollHelpOverlay(-1)
		} else {
			olc.moveSelection(-1)
		}
	case internal.DirectionDown:
		if olc.ShowingHelp {
			olc.scrollHelpOverlay(1)
		} else {
			olc.moveSelection(1)
		}
	case internal.DirectionLeft:
		if !olc.ShowingHelp {
			olc.cycleOptionLeft()
		}
	case internal.DirectionRight:
		if !olc.ShowingHelp {
			olc.cycleOptionRight()
		}
	}
}

func (olc *optionsListController) handleAButton(running *bool, result *OptionsListResult) {
	if olc.SelectedIndex >= 0 && olc.SelectedIndex < len(olc.Items) {
		item := &olc.Items[olc.SelectedIndex]
		if len(item.Options) > 0 && item.SelectedOption < len(item.Options) {
			o := item.Options[item.SelectedOption]
			switch o.Type {
			case OptionTypeKeyboard:
				prompt := o.KeyboardPrompt
				var keyboardResult *KeyboardResult
				var err error

				// Use URLKeyboard if layout is URL and shortcuts are provided
				if o.KeyboardLayout == KeyboardLayoutURL && len(o.URLShortcuts) > 0 {
					keyboardResult, err = URLKeyboard(prompt, olc.Settings.HelpExitText, URLKeyboardConfig{
						Shortcuts: o.URLShortcuts,
					})
				} else {
					keyboardResult, err = Keyboard(prompt, olc.Settings.HelpExitText, o.KeyboardLayout)
				}

				if err == nil {
					enteredText := keyboardResult.Text
					item.Options[item.SelectedOption] = Option{
						DisplayName:    enteredText,
						Value:          enteredText,
						Type:           OptionTypeKeyboard,
						KeyboardPrompt: enteredText,
						KeyboardLayout: o.KeyboardLayout,
						URLShortcuts:   o.URLShortcuts,
						Masked:         o.Masked,
					}
				}
			case OptionTypeColorPicker:
				olc.showColorPicker(olc.SelectedIndex)
			case OptionTypeClickable:
				*running = false
				result.Action = ListActionSelected
				result.Selected = olc.SelectedIndex
			case OptionTypeStandard:
				// Show list picker if enabled via ListPickerButton set to A
				if olc.Settings.ListPickerButton == constants.VirtualButtonA {
					olc.showListPicker()
				}
			}
		}
	}
}

func (olc *optionsListController) moveSelection(direction int) {
	if len(olc.Items) == 0 {
		return
	}

	startIndex := olc.SelectedIndex
	olc.Items[olc.SelectedIndex].Item.Selected = false

	// Find next visible item in the given direction
	for {
		if direction > 0 {
			olc.SelectedIndex++
			if olc.SelectedIndex >= len(olc.Items) {
				olc.SelectedIndex = 0
				olc.VisibleStartIndex = 0
			}
		} else {
			olc.SelectedIndex--
			if olc.SelectedIndex < 0 {
				olc.SelectedIndex = len(olc.Items) - 1
				if len(olc.Items) > olc.MaxVisibleItems {
					olc.VisibleStartIndex = len(olc.Items) - olc.MaxVisibleItems
				} else {
					olc.VisibleStartIndex = 0
				}
			}
		}

		// If item is visible, we found our target
		if olc.Items[olc.SelectedIndex].IsVisible() {
			break
		}

		// If we've wrapped around to start, no visible items exist
		if olc.SelectedIndex == startIndex {
			break
		}
	}

	olc.Items[olc.SelectedIndex].Item.Selected = true
	olc.scrollTo(olc.SelectedIndex)

	if olc.OnSelect != nil {
		olc.OnSelect(olc.SelectedIndex, &olc.Items[olc.SelectedIndex])
	}
}

func (olc *optionsListController) showColorPicker(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(olc.Items) {
		return
	}

	item := &olc.Items[itemIndex]
	if item.colorPicker != nil {
		item.colorPicker.setVisible(true)
		olc.showingColorPicker = true
		olc.activeColorPickerIdx = itemIndex
	}
}

func (olc *optionsListController) hideColorPicker() {
	if olc.activeColorPickerIdx >= 0 && olc.activeColorPickerIdx < len(olc.Items) {
		item := &olc.Items[olc.activeColorPickerIdx]
		if item.colorPicker != nil {
			item.colorPicker.setVisible(false)
		}
	}
	olc.showingColorPicker = false
	olc.activeColorPickerIdx = -1
}

func (olc *optionsListController) showListPicker() {
	item := &olc.Items[olc.SelectedIndex]

	if len(item.Options) <= 1 {
		return
	}

	// Only show list picker for standard or keyboard options
	if len(item.Options) > 0 && item.SelectedOption < len(item.Options) {
		optType := item.Options[item.SelectedOption].Type
		if optType != OptionTypeStandard && optType != OptionTypeKeyboard {
			return
		}
	}

	menuItems := make([]MenuItem, len(item.Options))
	for i, opt := range item.Options {
		menuItems[i] = MenuItem{
			Text:     opt.DisplayName,
			Metadata: i,
		}
	}

	listOpts := DefaultListOptions(item.Item.Text, menuItems)
	listOpts.SelectedIndex = item.SelectedOption
	listOpts.UseSmallTitle = true
	listOpts.FooterHelpItems = []FooterHelpItem{
		{ButtonName: "B", HelpText: "Back"},
		{ButtonName: "A", HelpText: "Select"},
	}
	listResult, err := List(listOpts)

	if err != nil {
		return
	}

	if len(listResult.Selected) > 0 {
		newIndex := listResult.Selected[0]
		if newIndex >= 0 && newIndex < len(item.Options) {
			selectedOpt := item.Options[newIndex]

			// Handle keyboard option - show keyboard for custom input
			if selectedOpt.Type == OptionTypeKeyboard {
				prompt := selectedOpt.KeyboardPrompt

				var keyboardResult *KeyboardResult
				var kbErr error

				if selectedOpt.KeyboardLayout == KeyboardLayoutURL && len(selectedOpt.URLShortcuts) > 0 {
					keyboardResult, kbErr = URLKeyboard(prompt, olc.Settings.HelpExitText, URLKeyboardConfig{
						Shortcuts: selectedOpt.URLShortcuts,
					})
				} else {
					keyboardResult, kbErr = Keyboard(prompt, olc.Settings.HelpExitText, selectedOpt.KeyboardLayout)
				}

				if kbErr == nil && keyboardResult.Text != "" {
					enteredText := keyboardResult.Text
					item.Options[newIndex] = Option{
						DisplayName:    enteredText,
						Value:          enteredText,
						Type:           OptionTypeKeyboard,
						KeyboardPrompt: selectedOpt.KeyboardPrompt,
						KeyboardLayout: selectedOpt.KeyboardLayout,
						URLShortcuts:   selectedOpt.URLShortcuts,
						Masked:         selectedOpt.Masked,
						OnUpdate:       selectedOpt.OnUpdate,
					}
					item.SelectedOption = newIndex

					if selectedOpt.OnUpdate != nil {
						selectedOpt.OnUpdate(enteredText)
					}
				}
				return
			}

			// Standard option - just update selection
			item.SelectedOption = newIndex
			if selectedOpt.OnUpdate != nil {
				selectedOpt.OnUpdate(selectedOpt.Value)
			}
		}
	}
}

func (olc *optionsListController) cycleOptionLeft() {
	if olc.SelectedIndex < 0 || olc.SelectedIndex >= len(olc.Items) {
		return
	}

	item := &olc.Items[olc.SelectedIndex]
	if len(item.Options) == 0 {
		return
	}

	if item.Options[item.SelectedOption].Type == OptionTypeClickable {
		return
	}

	item.SelectedOption--
	if item.SelectedOption < 0 {
		item.SelectedOption = len(item.Options) - 1
	}

	currentOption := item.Options[item.SelectedOption]
	if currentOption.OnUpdate != nil {
		currentOption.OnUpdate(currentOption.Value)
	}
}

func (olc *optionsListController) cycleOptionRight() {
	if olc.SelectedIndex < 0 || olc.SelectedIndex >= len(olc.Items) {
		return
	}

	item := &olc.Items[olc.SelectedIndex]
	if len(item.Options) == 0 {
		return
	}

	if item.Options[item.SelectedOption].Type == OptionTypeClickable {
		return
	}

	item.SelectedOption++
	if item.SelectedOption >= len(item.Options) {
		item.SelectedOption = 0
	}

	currentOption := item.Options[item.SelectedOption]
	if currentOption.OnUpdate != nil {
		currentOption.OnUpdate(currentOption.Value)
	}
}

func (olc *optionsListController) scrollTo(index int) {
	if index < 0 || index >= len(olc.Items) {
		return
	}

	contextItems := olc.MaxVisibleItems / 4
	if contextItems < 1 {
		contextItems = 1
	}

	newStart := index - contextItems
	if newStart < 0 {
		newStart = 0
	}

	maxStart := len(olc.Items) - olc.MaxVisibleItems
	if maxStart < 0 {
		maxStart = 0
	}
	if newStart > maxStart {
		newStart = maxStart
	}

	olc.VisibleStartIndex = newStart
}

func (olc *optionsListController) toggleHelp() {
	if !olc.HelpEnabled {
		return
	}

	olc.ShowingHelp = !olc.ShowingHelp
	if olc.ShowingHelp && olc.helpOverlay == nil {
		helpLines := []string{
			"Navigation Controls:",
			"• Up / Down: Navigate through items",
			"• Left / Right: Change option for current item",
			"• A: Select or input text for keyboard options",
			"• B: Cancel and exit",
		}
		olc.helpOverlay = newHelpOverlay(fmt.Sprintf("%s Help", olc.Settings.Title), helpLines, olc.Settings.HelpExitText)
	}
}

func (olc *optionsListController) scrollHelpOverlay(direction int) {
	if olc.helpOverlay == nil {
		return
	}
	olc.helpOverlay.scroll(direction)
}

func (olc *optionsListController) render(renderer *sdl.Renderer) {
	if olc.ShowingHelp && olc.helpOverlay != nil {
		olc.helpOverlay.render(renderer, internal.Fonts.SmallFont)
		return
	}

	scaleFactor := internal.GetScaleFactor()
	window := internal.GetWindow()
	titleFont := internal.Fonts.ExtraLargeFont
	if olc.Settings.UseSmallTitle {
		titleFont = internal.Fonts.LargeFont
	}
	font := internal.Fonts.SmallFont

	itemSpacing := int32(float32(60) * scaleFactor)
	selectionRectHeight := int32(float32(60) * scaleFactor)
	cornerRadius := int32(float32(20) * scaleFactor)

	statusBarWidth := calculateStatusBarWidth(internal.Fonts.SmallFont, olc.Settings.StatusBar)

	if olc.Settings.Title != "" {
		titleSurface, _ := titleFont.RenderUTF8Blended(olc.Settings.Title, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if titleSurface != nil {
			defer titleSurface.Free()
			titleTexture, _ := renderer.CreateTextureFromSurface(titleSurface)
			if titleTexture != nil {
				defer titleTexture.Destroy()

				maxTitleWidth := window.GetWidth() - olc.Settings.Margins.Left - olc.Settings.Margins.Right - statusBarWidth
				displayWidth := titleSurface.W
				if displayWidth > maxTitleWidth {
					displayWidth = maxTitleWidth
				}

				var titleX int32
				switch olc.Settings.TitleAlign {
				case constants.TextAlignLeft:
					titleX = olc.Settings.Margins.Left
				case constants.TextAlignCenter:
					titleX = (window.GetWidth() - displayWidth) / 2
				case constants.TextAlignRight:
					titleX = window.GetWidth() - olc.Settings.Margins.Right - statusBarWidth - displayWidth
				}

				// Clip title to available width
				srcRect := &sdl.Rect{X: 0, Y: 0, W: displayWidth, H: titleSurface.H}
				destRect := &sdl.Rect{X: titleX, Y: olc.Settings.Margins.Top, W: displayWidth, H: titleSurface.H}
				renderer.Copy(titleTexture, srcRect, destRect)

				olc.StartY = olc.Settings.Margins.Top + titleSurface.H + olc.Settings.TitleSpacing + 5
			}
		}
	}

	renderStatusBar(renderer, internal.Fonts.SmallFont, olc.Settings.StatusBar, olc.Settings.Margins)

	olc.MaxVisibleItems = int(olc.calculateMaxVisibleItems(window))

	displayPosition := 0
	for itemIndex := olc.VisibleStartIndex; itemIndex < len(olc.Items) && displayPosition < olc.MaxVisibleItems; itemIndex++ {
		item := olc.Items[itemIndex]

		// Skip hidden items
		if !item.IsVisible() {
			continue
		}

		textColor := internal.GetTheme().TextColor
		bgColor := sdl.Color{R: 0, G: 0, B: 0, A: 0}

		if item.Item.Selected {
			textColor = internal.GetTheme().HighlightedTextColor
			bgColor = internal.GetTheme().HighlightColor
		}

		itemY := olc.StartY + (int32(displayPosition) * itemSpacing)

		if item.Item.Selected {
			selectionRect := &sdl.Rect{
				X: olc.Settings.Margins.Left - 10,
				Y: itemY - 5,
				W: window.GetWidth() - olc.Settings.Margins.Left - olc.Settings.Margins.Right + 20,
				H: selectionRectHeight,
			}
			internal.DrawRoundedRect(renderer, selectionRect, cornerRadius, sdl.Color{R: bgColor.R, G: bgColor.G, B: bgColor.B, A: bgColor.A})
		}

		// Calculate vertical center within selection rect
		selectionRectY := itemY - 5

		itemSurface, _ := font.RenderUTF8Blended(item.Item.Text, textColor)
		if itemSurface != nil {
			defer itemSurface.Free()
			itemTexture, _ := renderer.CreateTextureFromSurface(itemSurface)
			if itemTexture != nil {
				defer itemTexture.Destroy()
				renderer.Copy(itemTexture, nil, &sdl.Rect{
					X: olc.Settings.Margins.Left,
					Y: selectionRectY + (selectionRectHeight-itemSurface.H)/2,
					W: itemSurface.W,
					H: itemSurface.H,
				})
			}
		}

		if len(item.Options) > 0 {
			selectedOption := item.Options[item.SelectedOption]

			if selectedOption.Type == OptionTypeKeyboard {
				var indicatorText string

				if selectedOption.Masked {
					indicatorText = strings.Repeat("*", len(selectedOption.DisplayName))
				} else {
					indicatorText = selectedOption.DisplayName
				}

				optionSurface, _ := font.RenderUTF8Blended(indicatorText, textColor)
				if optionSurface != nil {
					defer optionSurface.Free()
					optionTexture, _ := renderer.CreateTextureFromSurface(optionSurface)
					if optionTexture != nil {
						defer optionTexture.Destroy()

						renderer.Copy(optionTexture, nil, &sdl.Rect{
							X: window.GetWidth() - olc.Settings.Margins.Right - optionSurface.W,
							Y: selectionRectY + (selectionRectHeight-optionSurface.H)/2,
							W: optionSurface.W,
							H: optionSurface.H,
						})
					}
				}
			} else if selectedOption.Type == OptionTypeClickable {
				indicatorText := selectedOption.DisplayName

				optionSurface, _ := font.RenderUTF8Blended(indicatorText, textColor)
				if optionSurface != nil {
					defer optionSurface.Free()
					optionTexture, _ := renderer.CreateTextureFromSurface(optionSurface)
					if optionTexture != nil {
						defer optionTexture.Destroy()

						renderer.Copy(optionTexture, nil, &sdl.Rect{
							X: window.GetWidth() - olc.Settings.Margins.Right - optionSurface.W,
							Y: selectionRectY + (selectionRectHeight-optionSurface.H)/2,
							W: optionSurface.W,
							H: optionSurface.H,
						})
					}
				}
			} else if selectedOption.Type == OptionTypeColorPicker {
				// For color picker option, display the color swatch and hex value
				indicatorText := selectedOption.DisplayName
				if indicatorText == "" {
					if color, ok := selectedOption.Value.(sdl.Color); ok {
						indicatorText = fmt.Sprintf("#%02X%02X%02X", color.R, color.G, color.B)
					} else {
						indicatorText = ""
					}
				}

				optionSurface, _ := font.RenderUTF8Blended(indicatorText, textColor)
				if optionSurface != nil {
					defer optionSurface.Free()
					optionTexture, _ := renderer.CreateTextureFromSurface(optionSurface)
					if optionTexture != nil {
						defer optionTexture.Destroy()

						// Make the swatch slightly smaller than text height
						swatchHeight := int32(float32(optionSurface.H) * 0.8) // 80% of text height
						swatchWidth := swatchHeight                           // Keep it square
						swatchSpacing := int32(float32(10) * scaleFactor)     // Scale spacing

						// Position swatch on the right
						swatchX := window.GetWidth() - olc.Settings.Margins.Right - swatchWidth

						// Position the text to the left of the swatch
						textX := swatchX - optionSurface.W - swatchSpacing

						// Center text vertically within selection rect
						textY := selectionRectY + (selectionRectHeight-optionSurface.H)/2

						// Center the swatch vertically within selection rect
						swatchY := selectionRectY + (selectionRectHeight-swatchHeight)/2

						// draw the text on the left
						renderer.Copy(optionTexture, nil, &sdl.Rect{
							X: textX,
							Y: textY,
							W: optionSurface.W,
							H: optionSurface.H,
						})

						// draw color swatch on the right
						if color, ok := selectedOption.Value.(sdl.Color); ok {
							swatchRect := &sdl.Rect{
								X: swatchX,
								Y: swatchY, // Centered vertically
								W: swatchWidth,
								H: swatchHeight,
							}

							// Save current color
							r, g, b, a, _ := renderer.GetDrawColor()

							// draw color swatch
							renderer.SetDrawColor(color.R, color.G, color.B, color.A)
							renderer.FillRect(swatchRect)

							// draw swatch border
							renderer.SetDrawColor(255, 255, 255, 255)
							renderer.DrawRect(swatchRect)

							// Restore previous color
							renderer.SetDrawColor(r, g, b, a)
						}
					}
				}
			} else {
				optionSurface, _ := font.RenderUTF8Blended(selectedOption.DisplayName, textColor)
				if optionSurface != nil {
					defer optionSurface.Free()
					optionTexture, _ := renderer.CreateTextureFromSurface(optionSurface)
					if optionTexture != nil {
						defer optionTexture.Destroy()

						renderer.Copy(optionTexture, nil, &sdl.Rect{
							X: window.GetWidth() - olc.Settings.Margins.Right - optionSurface.W,
							Y: selectionRectY + (selectionRectHeight-optionSurface.H)/2,
							W: optionSurface.W,
							H: optionSurface.H,
						})
					}
				}
			}
		}

		displayPosition++
	}

	renderFooter(
		renderer,
		internal.Fonts.SmallFont,
		olc.Settings.FooterHelpItems,
		olc.Settings.Margins.Bottom,
		true,
		true,
	)
}

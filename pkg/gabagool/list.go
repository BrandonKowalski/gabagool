package gabagool

import (
	"strings"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type ListOptions struct {
	Title             string
	Items             []MenuItem
	SelectedIndex     int
	VisibleStartIndex int
	MaxVisibleItems   int

	EnableImages bool

	StartInMultiSelectMode bool
	DisableBackButton      bool

	HelpTitle    string
	HelpText     []string
	HelpExitText string

	Margins         internal.Padding
	ItemSpacing     int32
	SmallTitle      bool
	TitleAlign      constants.TextAlign
	TitleSpacing    int32
	FooterText      string
	FooterTextColor sdl.Color
	FooterHelpItems []FooterHelpItem
	StatusBar       StatusBarOptions

	ScrollSpeed     float32
	ScrollPauseTime int

	InputDelay            time.Duration
	MultiSelectButton     constants.VirtualButton
	ReorderButton         constants.VirtualButton
	ActionButton          constants.VirtualButton
	SecondaryActionButton constants.VirtualButton
	HelpButton            constants.VirtualButton

	EmptyMessage      string
	EmptyMessageColor sdl.Color

	OnSelect  func(index int, item *MenuItem)
	OnReorder func(from, to int)
}

func DefaultListOptions(title string, items []MenuItem) ListOptions {
	return ListOptions{
		Title:                 title,
		Items:                 items,
		SelectedIndex:         0,
		MaxVisibleItems:       9,
		Margins:               internal.UniformPadding(20),
		TitleAlign:            constants.TextAlignLeft,
		TitleSpacing:          constants.DefaultTitleSpacing,
		FooterTextColor:       sdl.Color{R: 180, G: 180, B: 180, A: 255},
		ScrollSpeed:           4.0,
		ScrollPauseTime:       1250,
		InputDelay:            constants.DefaultInputDelay,
		MultiSelectButton:     constants.VirtualButtonUnassigned,
		ReorderButton:         constants.VirtualButtonUnassigned,
		ActionButton:          constants.VirtualButtonUnassigned,
		SecondaryActionButton: constants.VirtualButtonUnassigned,
		HelpButton:            constants.VirtualButtonUnassigned,
		EmptyMessage:          "No items available",
		EmptyMessageColor:     sdl.Color{R: 255, G: 255, B: 255, A: 255},
		StatusBar:             DefaultStatusBarOptions(),
	}
}

type listController struct {
	Options       ListOptions
	SelectedItems map[int]bool
	MultiSelect   bool
	ReorderMode   bool
	ShowingHelp   bool
	StartY        int32
	lastInputTime time.Time

	helpOverlay     *helpOverlay
	itemScrollData  map[int]*internal.TextScrollData
	titleScrollData *internal.TextScrollData

	heldDirections struct {
		up, down, left, right bool
	}
	lastRepeatTime time.Time
	repeatDelay    time.Duration
	repeatInterval time.Duration
	hasRepeated    bool
}

func newListController(options ListOptions) *listController {
	selectedItems := make(map[int]bool)
	if options.SelectedIndex < 0 || options.SelectedIndex >= len(options.Items) {
		options.SelectedIndex = 0
	}

	for i := range options.Items {
		if options.Items[i].Selected {
			selectedItems[i] = true
		}
	}

	var helpOverlay *helpOverlay
	if options.HelpButton != constants.VirtualButtonUnassigned {
		helpOverlay = newHelpOverlay(options.HelpTitle, options.HelpText, options.HelpExitText)
	}

	return &listController{
		Options:         options,
		SelectedItems:   selectedItems,
		MultiSelect:     options.StartInMultiSelectMode,
		StartY:          20,
		lastInputTime:   time.Now(),
		helpOverlay:     helpOverlay,
		itemScrollData:  make(map[int]*internal.TextScrollData),
		titleScrollData: &internal.TextScrollData{},
		lastRepeatTime:  time.Now(),
		repeatDelay:     150 * time.Millisecond,
		repeatInterval:  50 * time.Millisecond,
	}
}

func List(options ListOptions) (*ListResult, error) {
	window := internal.GetWindow()
	renderer := window.Renderer

	if options.MaxVisibleItems <= 0 {
		options.MaxVisibleItems = 9
	}

	lc := newListController(options)

	lc.Options.MaxVisibleItems = int(lc.calculateMaxVisibleItems(window))

	if options.SelectedIndex > 0 {
		lc.scrollTo(options.SelectedIndex)
	}

	running := true
	cancelled := false
	result := ListResult{
		Items:    lc.Options.Items,
		Selected: []int{},
		Action:   ListActionSelected,
	}

	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
				lc.handleInput(event, &running, &result, &cancelled)
			case *sdl.WindowEvent:
				we := event.(*sdl.WindowEvent)
				if we.Event == sdl.WINDOWEVENT_RESIZED {
					newMaxItems := lc.calculateMaxVisibleItems(window)
					lc.Options.MaxVisibleItems = int(newMaxItems)
					if lc.Options.SelectedIndex >= lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems {
						lc.scrollTo(lc.Options.SelectedIndex)
					}
				}
			}
		}

		lc.handleDirectionalRepeats()

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
		renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

		lc.render(window)
		renderer.Present()
		sdl.Delay(16)
	}

	// Update result with final item order (in case items were reordered)
	result.Items = lc.Options.Items

	if cancelled {
		return &result, ErrCancelled
	}

	return &result, nil
}

func (lc *listController) handleInput(event interface{}, running *bool, result *ListResult, cancelled *bool) {
	processor := internal.GetInputProcessor()

	inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
	if inputEvent == nil {
		return
	}

	if inputEvent.Pressed {
		if lc.ShowingHelp {
			lc.handleHelpInput(inputEvent.Button)
			return
		}

		if lc.ReorderMode && !lc.isDirectionalInput(inputEvent.Button) {
			lc.ReorderMode = false
			return
		}

		if lc.handleNavigation(inputEvent.Button) {
			return
		}

		lc.handleActionButtons(inputEvent.Button, running, result, cancelled)
	} else {
		lc.handleInputEventRelease(inputEvent)
	}
}

func (lc *listController) handleHelpInput(button constants.VirtualButton) {
	switch button {
	case constants.VirtualButtonUp:
		if lc.helpOverlay != nil {
			lc.helpOverlay.scroll(-1)
		}
	case constants.VirtualButtonDown:
		if lc.helpOverlay != nil {
			lc.helpOverlay.scroll(1)
		}
	case constants.VirtualButtonMenu:
		lc.ShowingHelp = false
	default:
		lc.ShowingHelp = false
	}
}

func (lc *listController) handleInputEventRelease(inputEvent *internal.Event) {
	switch inputEvent.Button {
	case constants.VirtualButtonUp:
		lc.heldDirections.up = false
		lc.hasRepeated = false
	case constants.VirtualButtonDown:
		lc.heldDirections.down = false
		lc.hasRepeated = false
	case constants.VirtualButtonLeft:
		lc.heldDirections.left = false
		lc.hasRepeated = false
	case constants.VirtualButtonRight:
		lc.heldDirections.right = false
		lc.hasRepeated = false
	}
}

func (lc *listController) isDirectionalInput(button constants.VirtualButton) bool {
	return button == constants.VirtualButtonUp || button == constants.VirtualButtonDown ||
		button == constants.VirtualButtonLeft || button == constants.VirtualButtonRight
}

func (lc *listController) handleNavigation(button constants.VirtualButton) bool {
	if len(lc.Options.Items) == 0 {
		return false
	}

	direction := ""
	switch button {
	case constants.VirtualButtonUp:
		direction = "up"
		lc.heldDirections.up = true
		lc.heldDirections.down = false
	case constants.VirtualButtonDown:
		direction = "down"
		lc.heldDirections.down = true
		lc.heldDirections.up = false
	case constants.VirtualButtonLeft:
		direction = "left"
		lc.heldDirections.left = true
		lc.heldDirections.right = false
	case constants.VirtualButtonRight:
		direction = "right"
		lc.heldDirections.right = true
		lc.heldDirections.left = false
	default:
	}

	if direction != "" {
		lc.navigate(direction)
		lc.lastRepeatTime = time.Now()
		return true
	}
	return false
}

func (lc *listController) handleActionButtons(button constants.VirtualButton, running *bool, result *ListResult, cancelled *bool) {
	if len(lc.Options.Items) == 0 && button != constants.VirtualButtonB && button != constants.VirtualButtonMenu {
		return
	}

	if button == constants.VirtualButtonA {
		if lc.MultiSelect && len(lc.Options.Items) > 0 {
			lc.toggleSelection(lc.Options.SelectedIndex)
		} else if len(lc.Options.Items) > 0 {
			*running = false
			result.Action = ListActionSelected
			result.Selected = []int{lc.Options.SelectedIndex}
			result.VisiblePosition = lc.Options.SelectedIndex - lc.Options.VisibleStartIndex
		}
	}

	if button == constants.VirtualButtonB {
		if !lc.Options.DisableBackButton {
			*running = false
			*cancelled = true
			// Update result with current item order before cancelling
			result.Items = lc.Options.Items
		}
	}

	// Primary action button handling
	if lc.Options.ActionButton != constants.VirtualButtonUnassigned && button == lc.Options.ActionButton {
		*running = false
		result.Action = ListActionTriggered
		if len(lc.Options.Items) > 0 {
			if lc.MultiSelect {
				if indices := lc.getSelectedItems(); len(indices) > 0 {
					result.Selected = indices
					result.VisiblePosition = indices[0] - lc.Options.VisibleStartIndex
				}
			} else {
				result.Selected = []int{lc.Options.SelectedIndex}
				result.VisiblePosition = lc.Options.SelectedIndex - lc.Options.VisibleStartIndex
			}
		}
	}

	// Secondary action button handling
	if lc.Options.SecondaryActionButton != constants.VirtualButtonUnassigned &&
		button == lc.Options.SecondaryActionButton {
		*running = false
		result.Action = ListActionSecondaryTriggered
		if len(lc.Options.Items) > 0 {
			if lc.MultiSelect {
				if indices := lc.getSelectedItems(); len(indices) > 0 {
					result.Selected = indices
					result.VisiblePosition = indices[0] - lc.Options.VisibleStartIndex
				}
			} else {
				result.Selected = []int{lc.Options.SelectedIndex}
				result.VisiblePosition = lc.Options.SelectedIndex - lc.Options.VisibleStartIndex
			}
		}
	}

	if lc.Options.HelpButton != constants.VirtualButtonUnassigned &&
		button == lc.Options.HelpButton {
		lc.ShowingHelp = !lc.ShowingHelp
	}

	if button == constants.VirtualButtonStart {
		if lc.MultiSelect && len(lc.Options.Items) > 0 {
			// Only allow start button when at least one item is selected
			if indices := lc.getSelectedItems(); len(indices) > 0 {
				*running = false
				result.Action = ListActionSelected
				result.Selected = indices
				result.VisiblePosition = indices[0] - lc.Options.VisibleStartIndex
			}
		}
	}

	if lc.Options.MultiSelectButton != constants.VirtualButtonUnassigned &&
		button == lc.Options.MultiSelectButton && len(lc.Options.Items) > 0 {
		lc.toggleMultiSelect()
	}

	if lc.Options.ReorderButton != constants.VirtualButtonUnassigned &&
		button == lc.Options.ReorderButton && len(lc.Options.Items) > 0 &&
		!lc.Options.Items[lc.Options.SelectedIndex].NotReorderable {
		lc.ReorderMode = !lc.ReorderMode
	}
}

func (lc *listController) navigate(direction string) {
	if time.Since(lc.lastInputTime) < lc.Options.InputDelay {
		return
	}
	lc.lastInputTime = time.Now()

	switch direction {
	case "up":
		if lc.ReorderMode {
			lc.moveItem(-1)
		} else {
			lc.moveSelection(-1)
		}
	case "down":
		if lc.ReorderMode {
			lc.moveItem(1)
		} else {
			lc.moveSelection(1)
		}
	case "left":
		if lc.ReorderMode {
			lc.moveItem(-lc.Options.MaxVisibleItems)
		} else {
			lc.moveSelection(-lc.Options.MaxVisibleItems)
		}
	case "right":
		if lc.ReorderMode {
			lc.moveItem(lc.Options.MaxVisibleItems)
		} else {
			lc.moveSelection(lc.Options.MaxVisibleItems)
		}
	}
}

func (lc *listController) moveSelection(delta int) {
	newIndex := lc.Options.SelectedIndex + delta

	// Handle wrapping and page jumps
	if delta == 1 { // Down
		if newIndex >= len(lc.Options.Items) {
			newIndex = 0
			lc.Options.VisibleStartIndex = 0
		}
	} else if delta == -1 { // Up
		if newIndex < 0 {
			newIndex = len(lc.Options.Items) - 1
			if len(lc.Options.Items) > lc.Options.MaxVisibleItems {
				lc.Options.VisibleStartIndex = len(lc.Options.Items) - lc.Options.MaxVisibleItems
			}
		}
	} else { // Page jumps
		if delta > 0 { // Page right
			if len(lc.Options.Items) <= lc.Options.MaxVisibleItems {
				newIndex = len(lc.Options.Items) - 1
			} else {
				maxStart := len(lc.Options.Items) - lc.Options.MaxVisibleItems
				if lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems >= len(lc.Options.Items) {
					newIndex = len(lc.Options.Items) - 1
					lc.Options.VisibleStartIndex = maxStart
				} else {
					newIndex = min(lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems, len(lc.Options.Items)-1)
					lc.Options.VisibleStartIndex = newIndex
				}
			}
		} else { // Page left
			if lc.Options.VisibleStartIndex == 0 {
				newIndex = 0
			} else {
				newIndex = max(lc.Options.VisibleStartIndex-lc.Options.MaxVisibleItems, 0)
				lc.Options.VisibleStartIndex = newIndex
			}
		}
	}

	lc.Options.SelectedIndex = newIndex
	lc.scrollTo(newIndex)
	lc.updateSelectionState()
}

func (lc *listController) moveItem(delta int) {
	if delta == 1 && lc.Options.SelectedIndex >= len(lc.Options.Items)-1 {
		return
	}
	if delta == -1 && lc.Options.SelectedIndex <= 0 {
		return
	}

	// Handle multistep moves for page jumps
	steps := delta
	if delta > 1 || delta < -1 {
		steps = delta / internal.Abs(delta) // Get direction
		targetIndex := lc.Options.SelectedIndex + delta
		targetIndex = max(0, min(targetIndex, len(lc.Options.Items)-1))

		// Move item step by step to target
		for lc.Options.SelectedIndex != targetIndex {
			if !lc.moveItemOneStep(steps) {
				break
			}
		}
		return
	}

	lc.moveItemOneStep(steps)
}

func (lc *listController) moveItemOneStep(direction int) bool {
	currentIndex := lc.Options.SelectedIndex
	var targetIndex int

	if direction > 0 {
		if currentIndex >= len(lc.Options.Items)-1 {
			return false
		}
		targetIndex = currentIndex + 1
	} else {
		if currentIndex <= 0 {
			return false
		}
		targetIndex = currentIndex - 1
	}

	// Check if either item is marked as not reorderable
	if lc.Options.Items[currentIndex].NotReorderable || lc.Options.Items[targetIndex].NotReorderable {
		return false
	}

	// Swap items
	lc.Options.Items[currentIndex], lc.Options.Items[targetIndex] = lc.Options.Items[targetIndex], lc.Options.Items[currentIndex]

	// Update selection states
	if lc.MultiSelect {
		currentSelected := lc.SelectedItems[currentIndex]
		targetSelected := lc.SelectedItems[targetIndex]

		delete(lc.SelectedItems, currentIndex)
		delete(lc.SelectedItems, targetIndex)

		if currentSelected {
			lc.SelectedItems[targetIndex] = true
		}
		if targetSelected {
			lc.SelectedItems[currentIndex] = true
		}
	}

	lc.Options.SelectedIndex = targetIndex
	lc.scrollTo(targetIndex)

	if lc.Options.OnReorder != nil {
		lc.Options.OnReorder(currentIndex, targetIndex)
	}

	return true
}

func (lc *listController) toggleMultiSelect() {
	lc.MultiSelect = !lc.MultiSelect

	if !lc.MultiSelect {
		for i := range lc.Options.Items {
			lc.Options.Items[i].Selected = false
		}
		lc.SelectedItems = make(map[int]bool)
	}

	lc.updateSelectionState()
}

func (lc *listController) toggleSelection(index int) {
	if index < 0 || index >= len(lc.Options.Items) || lc.Options.Items[index].NotMultiSelectable {
		return
	}

	lc.Options.Items[index].Selected = !lc.Options.Items[index].Selected
	if lc.Options.Items[index].Selected {
		lc.SelectedItems[index] = true
	} else {
		delete(lc.SelectedItems, index)
	}
}

func (lc *listController) updateSelectionState() {
	if !lc.MultiSelect {
		for i := range lc.Options.Items {
			lc.Options.Items[i].Selected = i == lc.Options.SelectedIndex
		}
		lc.SelectedItems = map[int]bool{lc.Options.SelectedIndex: true}
	}
}

func (lc *listController) getSelectedItems() []int {
	var indices []int
	for idx := range lc.SelectedItems {
		indices = append(indices, idx)
	}
	return indices
}

func (lc *listController) scrollTo(index int) {
	if index < lc.Options.VisibleStartIndex {
		lc.Options.VisibleStartIndex = index
	} else if index >= lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems {
		lc.Options.VisibleStartIndex = index - lc.Options.MaxVisibleItems + 1
		if lc.Options.VisibleStartIndex < 0 {
			lc.Options.VisibleStartIndex = 0
		}
	}
}

func (lc *listController) handleDirectionalRepeats() {
	if len(lc.Options.Items) == 0 || (!lc.heldDirections.up && !lc.heldDirections.down && !lc.heldDirections.left && !lc.heldDirections.right) {
		lc.lastRepeatTime = time.Now()
		lc.hasRepeated = false
		return
	}

	timeSince := time.Since(lc.lastRepeatTime)

	// Use repeatDelay for first repeat, then repeatInterval for subsequent repeats
	threshold := lc.repeatInterval
	if !lc.hasRepeated {
		threshold = lc.repeatDelay
	}

	if timeSince >= threshold {
		lc.lastRepeatTime = time.Now()
		lc.hasRepeated = true

		if lc.heldDirections.up {
			lc.navigate("up")
		} else if lc.heldDirections.down {
			lc.navigate("down")
		} else if lc.heldDirections.left {
			lc.navigate("left")
		} else if lc.heldDirections.right {
			lc.navigate("right")
		}
	}
}

func (lc *listController) render(window *internal.Window) {
	lc.updateScrolling()

	for i := range lc.Options.Items {
		lc.Options.Items[i].Focused = i == lc.Options.SelectedIndex
	}

	endIndex := min(lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems, len(lc.Options.Items))
	visibleItems := make([]MenuItem, endIndex-lc.Options.VisibleStartIndex)
	copy(visibleItems, lc.Options.Items[lc.Options.VisibleStartIndex:endIndex])

	if lc.ReorderMode {
		selectedIdx := lc.Options.SelectedIndex - lc.Options.VisibleStartIndex
		if selectedIdx >= 0 && selectedIdx < len(visibleItems) {
			visibleItems[selectedIdx].Text = "↕ " + visibleItems[selectedIdx].Text
		}
	}

	lc.renderContent(window, visibleItems)

	if lc.ShowingHelp && lc.helpOverlay != nil {
		lc.helpOverlay.ShowingHelp = true
		lc.helpOverlay.render(window.Renderer, internal.Fonts.SmallFont)
	}
}

func (lc *listController) renderContent(window *internal.Window, visibleItems []MenuItem) {
	renderer := window.Renderer

	itemStartY := lc.StartY

	if lc.Options.EnableImages && lc.Options.SelectedIndex < len(lc.Options.Items) {
		selectedItem := lc.Options.Items[lc.Options.SelectedIndex]
		if selectedItem.BackgroundFilename != "" {
			lc.renderSelectedItemBackground(window, selectedItem.BackgroundFilename)
		} else {
			window.RenderBackground()
		}
	} else {
		window.RenderBackground()
	}

	statusBarWidth := calculateStatusBarWidth(internal.Fonts.SmallFont, internal.Fonts.SmallSymbolFont, lc.Options.StatusBar)

	if lc.Options.Title != "" {
		titleFont := internal.Fonts.ExtraLargeFont
		if lc.Options.SmallTitle {
			titleFont = internal.Fonts.LargeFont
		}
		itemStartY = lc.renderScrollableTitle(renderer, titleFont, lc.Options.Title, lc.Options.TitleAlign, lc.StartY, lc.Options.Margins.Left+10, statusBarWidth) + lc.Options.TitleSpacing
	}

	renderStatusBar(renderer, internal.Fonts.SmallFont, internal.Fonts.SmallSymbolFont, lc.Options.StatusBar, lc.Options.Margins)

	if len(lc.Options.Items) == 0 {
		lc.renderEmptyMessage(renderer, internal.Fonts.MediumFont, itemStartY)
	} else {
		lc.renderItems(renderer, internal.Fonts.SmallFont, visibleItems, itemStartY)
	}

	if lc.imageIsDisplayed() {
		lc.renderSelectedItemImage(renderer, lc.Options.Items[lc.Options.SelectedIndex].ImageFilename)
	}

	// Filter footer items: hide confirm button when multiselect is active with no selections
	footerItems := lc.Options.FooterHelpItems
	centerSingleItem := len(lc.Options.FooterHelpItems) == 1
	if lc.MultiSelect && len(lc.SelectedItems) == 0 {
		footerItems = lc.filterConfirmButton(lc.Options.FooterHelpItems)
	}

	renderFooter(renderer, internal.Fonts.SmallFont, footerItems, lc.Options.Margins.Bottom, true, centerSingleItem)
}

func (lc *listController) imageIsDisplayed() bool {
	if lc.Options.EnableImages && lc.Options.SelectedIndex < len(lc.Options.Items) {
		selectedItem := lc.Options.Items[lc.Options.SelectedIndex]
		if selectedItem.ImageFilename != "" {
			return true
		}
	}
	return false
}

func (lc *listController) renderItems(renderer *sdl.Renderer, font *ttf.Font, visibleItems []MenuItem, startY int32) {
	scaleFactor := internal.GetScaleFactor()

	pillHeight := int32(float32(60) * scaleFactor)
	pillPadding := int32(float32(40) * scaleFactor)

	screenWidth, _, _ := renderer.GetOutputSize()
	availableWidth := screenWidth - lc.Options.Margins.Left - lc.Options.Margins.Right
	if lc.imageIsDisplayed() {
		availableWidth -= screenWidth / 7
	}

	maxPillWidth := availableWidth
	if lc.imageIsDisplayed() {
		maxPillWidth = availableWidth * 3 / 4
	}
	maxTextWidth := maxPillWidth - pillPadding

	for i, item := range visibleItems {
		itemText := lc.formatItemText(item, lc.MultiSelect)
		itemY := startY + int32(i)*(pillHeight+lc.Options.ItemSpacing)
		globalIndex := lc.Options.VisibleStartIndex + i

		if item.Selected || item.Focused {
			_, bgColor := lc.getItemColors(item)
			pillWidth := internal.Min32(maxPillWidth, lc.measureText(font, itemText)+pillPadding)

			pillRect := sdl.Rect{
				X: lc.Options.Margins.Left,
				Y: itemY,
				W: pillWidth,
				H: pillHeight,
			}
			internal.DrawRoundedRect(renderer, &pillRect, int32(float32(30)*scaleFactor), bgColor)
		}

		lc.renderItemText(renderer, font, itemText, item.Focused, globalIndex, itemY, pillHeight, maxTextWidth)
	}
}

func (lc *listController) renderItemText(renderer *sdl.Renderer, font *ttf.Font, text string, focused bool, globalIndex int, itemY, pillHeight, maxWidth int32) {
	textColor := lc.getTextColor(focused)

	if focused && lc.shouldScroll(font, text, maxWidth) {
		lc.renderScrollingText(renderer, font, text, textColor, globalIndex, itemY, pillHeight, maxWidth)
	} else {
		truncatedText := lc.truncateText(font, text, maxWidth)
		lc.renderStaticText(renderer, font, truncatedText, textColor, itemY, pillHeight)
	}
}

func (lc *listController) renderStaticText(renderer *sdl.Renderer, font *ttf.Font, text string, color sdl.Color, itemY, pillHeight int32) {
	scaleFactor := internal.GetScaleFactor()

	surface, _ := font.RenderUTF8Blended(text, color)
	if surface == nil {
		return
	}
	defer surface.Free()

	texture, _ := renderer.CreateTextureFromSurface(surface)
	if texture == nil {
		return
	}
	defer texture.Destroy()

	textPadding := int32(float32(20) * scaleFactor)
	destRect := sdl.Rect{
		X: lc.Options.Margins.Left + textPadding,
		Y: itemY + (pillHeight-surface.H)/2,
		W: surface.W,
		H: surface.H,
	}

	renderer.Copy(texture, nil, &destRect)
}

func (lc *listController) renderScrollingText(renderer *sdl.Renderer, font *ttf.Font, text string, color sdl.Color, globalIndex int, itemY, pillHeight, maxWidth int32) {
	scaleFactor := internal.GetScaleFactor()
	scrollData := lc.getOrCreateScrollData(globalIndex, text, font, maxWidth)

	surface, _ := font.RenderUTF8Blended(text, color)
	if surface == nil {
		return
	}
	defer surface.Free()

	texture, _ := renderer.CreateTextureFromSurface(surface)
	if texture == nil {
		return
	}
	defer texture.Destroy()

	clipRect := &sdl.Rect{
		X: scrollData.ScrollOffset,
		Y: 0,
		W: internal.Min32(maxWidth, surface.W-scrollData.ScrollOffset),
		H: surface.H,
	}

	textPadding := int32(float32(20) * scaleFactor)
	destRect := sdl.Rect{
		X: lc.Options.Margins.Left + textPadding,
		Y: itemY + (pillHeight-surface.H)/2,
		W: clipRect.W,
		H: surface.H,
	}

	renderer.Copy(texture, clipRect, &destRect)
}

func (lc *listController) renderEmptyMessage(renderer *sdl.Renderer, font *ttf.Font, startY int32) {
	lines := strings.Split(lc.Options.EmptyMessage, "\n")
	screenWidth, screenHeight, _ := renderer.GetOutputSize()

	lineHeight := int32(25)
	totalHeight := int32(len(lines)) * lineHeight
	centerY := startY + (screenHeight-startY-lc.Options.Margins.Bottom-totalHeight)/2

	for i, line := range lines {
		if line == "" {
			continue
		}

		surface, _ := font.RenderUTF8Blended(line, lc.Options.EmptyMessageColor)
		if surface == nil {
			continue
		}

		texture, _ := renderer.CreateTextureFromSurface(surface)
		if texture == nil {
			surface.Free()
			continue
		}

		rect := sdl.Rect{
			X: (screenWidth - surface.W) / 2,
			Y: centerY + int32(i)*lineHeight,
			W: surface.W,
			H: surface.H,
		}

		renderer.Copy(texture, nil, &rect)
		texture.Destroy()
		surface.Free()
	}
}

func (lc *listController) renderSelectedItemBackground(window *internal.Window, imageFilename string) {
	bgTexture, err := img.LoadTexture(window.Renderer, imageFilename)
	if err != nil {
		return
	}
	defer bgTexture.Destroy()
	window.Renderer.Copy(bgTexture, nil, &sdl.Rect{X: 0, Y: 0, W: window.GetWidth(), H: window.GetHeight()})
}

func (lc *listController) renderSelectedItemImage(renderer *sdl.Renderer, imageFilename string) {
	texture, err := img.LoadTexture(renderer, imageFilename)
	if err != nil {
		return
	}
	defer texture.Destroy()

	_, _, textureWidth, textureHeight, _ := texture.Query()
	screenWidth, screenHeight, _ := renderer.GetOutputSize()

	if textureWidth == 0 || textureHeight == 0 {
		return
	}

	maxImageWidth := screenWidth / 3
	maxImageHeight := screenHeight / 2

	scaleX := float32(maxImageWidth) / float32(textureWidth)
	scaleY := float32(maxImageHeight) / float32(textureHeight)

	// Use the smaller scale to maintain the aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	imageWidth := int32(float32(textureWidth) * scale)
	imageHeight := int32(float32(textureHeight) * scale)

	// Ensure we have valid dimensions after scaling
	if imageWidth <= 0 || imageHeight <= 0 {
		return
	}

	destRect := sdl.Rect{
		X: screenWidth - imageWidth - 20,
		Y: (screenHeight - imageHeight) / 2,
		W: imageWidth,
		H: imageHeight,
	}

	renderer.Copy(texture, nil, &destRect)
}

func (lc *listController) renderScrollableTitle(renderer *sdl.Renderer, font *ttf.Font, title string, align constants.TextAlign, startY, marginLeft, statusBarWidth int32) int32 {
	surface, _ := font.RenderUTF8Blended(title, internal.GetTheme().ListTextColor)
	if surface == nil {
		return startY + 40
	}
	defer surface.Free()

	texture, _ := renderer.CreateTextureFromSurface(surface)
	if texture == nil {
		return startY + 40
	}
	defer texture.Destroy()

	screenWidth, _, _ := renderer.GetOutputSize()
	availableWidth := screenWidth - (marginLeft * 2) - statusBarWidth

	if surface.W > availableWidth {
		lc.renderScrollingTitle(renderer, texture, surface.H, availableWidth, marginLeft, startY)
	} else {
		var titleX int32
		switch align {
		case constants.TextAlignCenter:
			titleX = (screenWidth - surface.W) / 2
		case constants.TextAlignRight:
			titleX = screenWidth - surface.W - marginLeft
		default:
			titleX = marginLeft
		}

		rect := sdl.Rect{X: titleX, Y: startY, W: surface.W, H: surface.H}
		renderer.Copy(texture, nil, &rect)
	}

	return startY + surface.H
}

func (lc *listController) renderScrollingTitle(renderer *sdl.Renderer, texture *sdl.Texture, textHeight, maxWidth, titleX, titleY int32) {
	if !lc.titleScrollData.NeedsScrolling {
		_, _, fullWidth, _, _ := texture.Query()
		lc.titleScrollData.NeedsScrolling = true
		lc.titleScrollData.TextWidth = fullWidth
		lc.titleScrollData.ContainerWidth = maxWidth
		lc.titleScrollData.Direction = 1
	}

	clipRect := &sdl.Rect{
		X: internal.Max32(0, lc.titleScrollData.ScrollOffset),
		Y: 0,
		W: internal.Min32(maxWidth, lc.titleScrollData.TextWidth-lc.titleScrollData.ScrollOffset),
		H: textHeight,
	}

	destRect := sdl.Rect{X: titleX, Y: titleY, W: clipRect.W, H: textHeight}
	renderer.Copy(texture, clipRect, &destRect)
}

func (lc *listController) updateScrolling() {
	currentTime := time.Now()

	if lc.titleScrollData.NeedsScrolling {
		lc.updateScrollData(lc.titleScrollData, currentTime)
	}

	for idx := lc.Options.VisibleStartIndex; idx < min(lc.Options.VisibleStartIndex+lc.Options.MaxVisibleItems, len(lc.Options.Items)); idx++ {
		if scrollData, exists := lc.itemScrollData[idx]; exists && scrollData.NeedsScrolling {
			lc.updateScrollData(scrollData, currentTime)
		}
	}
}

func (lc *listController) updateScrollData(data *internal.TextScrollData, currentTime time.Time) {
	if data.LastDirectionChange != nil && currentTime.Sub(*data.LastDirectionChange) < time.Duration(lc.Options.ScrollPauseTime)*time.Millisecond {
		return
	}

	scrollIncrement := int32(lc.Options.ScrollSpeed)
	data.ScrollOffset += int32(data.Direction) * scrollIncrement

	maxOffset := data.TextWidth - data.ContainerWidth
	if data.ScrollOffset <= 0 {
		data.ScrollOffset = 0
		if data.Direction < 0 {
			data.Direction = 1
			now := currentTime
			data.LastDirectionChange = &now
		}
	} else if data.ScrollOffset >= maxOffset {
		data.ScrollOffset = maxOffset
		if data.Direction > 0 {
			data.Direction = -1
			now := currentTime
			data.LastDirectionChange = &now
		}
	}
}

func (lc *listController) getOrCreateScrollData(index int, text string, font *ttf.Font, maxWidth int32) *internal.TextScrollData {
	data, exists := lc.itemScrollData[index]
	if !exists {
		surface, _ := font.RenderUTF8Blended(text, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if surface == nil {
			return &internal.TextScrollData{}
		}
		defer surface.Free()

		data = &internal.TextScrollData{
			NeedsScrolling: surface.W > maxWidth,
			TextWidth:      surface.W,
			ContainerWidth: maxWidth,
			Direction:      1,
		}
		lc.itemScrollData[index] = data
	}
	return data
}

func (lc *listController) shouldScroll(font *ttf.Font, text string, maxWidth int32) bool {
	surface, _ := font.RenderUTF8Blended(text, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if surface == nil {
		return false
	}
	defer surface.Free()
	return surface.W > maxWidth
}

func (lc *listController) calculateMaxVisibleItems(window *internal.Window) int32 {
	scaleFactor := internal.GetScaleFactor()

	pillHeight := int32(float32(60) * scaleFactor)

	_, screenHeight, _ := window.Renderer.GetOutputSize()

	var titleHeight int32 = 0
	if lc.Options.Title != "" {
		if lc.Options.SmallTitle {
			titleHeight = int32(float32(50) * scaleFactor)
		} else {
			titleHeight = int32(float32(60) * scaleFactor)
		}
		titleHeight += lc.Options.TitleSpacing
	}

	footerHeight := int32(float32(50) * scaleFactor)

	availableHeight := screenHeight - titleHeight - footerHeight - (lc.StartY * 2)

	itemHeightWithSpacing := pillHeight + lc.Options.ItemSpacing
	maxItems := availableHeight/itemHeightWithSpacing - 1

	if maxItems < 1 {
		maxItems = 1
	}

	return maxItems
}

func (lc *listController) measureText(font *ttf.Font, text string) int32 {
	surface, _ := font.RenderUTF8Blended(text, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if surface == nil {
		return 0
	}
	defer surface.Free()
	return surface.W
}

func (lc *listController) truncateText(font *ttf.Font, text string, maxWidth int32) string {
	if !lc.shouldScroll(font, text, maxWidth) {
		return text
	}

	ellipsis := "..."
	runes := []rune(text)
	for len(runes) > 5 {
		runes = runes[:len(runes)-1]
		testText := string(runes) + ellipsis
		if !lc.shouldScroll(font, testText, maxWidth) {
			return testText
		}
	}
	return ellipsis
}

func (lc *listController) formatItemText(item MenuItem, multiSelect bool) string {
	if !multiSelect || item.NotMultiSelectable {
		return item.Text
	}
	if item.Selected {
		return "☑ " + item.Text
	}
	return "☐ " + item.Text
}

func (lc *listController) getItemColors(item MenuItem) (textColor, bgColor sdl.Color) {
	if item.Focused && item.Selected {
		return internal.GetTheme().ListTextSelectedColor, internal.GetTheme().MainColor
	} else if item.Focused {
		return internal.GetTheme().ListTextSelectedColor, internal.GetTheme().MainColor
	} else if item.Selected {
		return internal.GetTheme().ListTextColor, sdl.Color{R: 255, G: 0, B: 0, A: 0}
	}
	return internal.GetTheme().ListTextColor, sdl.Color{}
}

func (lc *listController) getTextColor(focused bool) sdl.Color {
	if focused {
		return internal.GetTheme().ListTextSelectedColor
	}
	return internal.GetTheme().ListTextColor
}

func (lc *listController) filterConfirmButton(items []FooterHelpItem) []FooterHelpItem {
	filtered := make([]FooterHelpItem, 0, len(items))
	for _, item := range items {
		if !item.IsConfirmButton {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

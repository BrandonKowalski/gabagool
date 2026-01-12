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

type MetadataItem struct {
	Label string
	Value string
}

const (
	SectionTypeSlideshow = iota
	SectionTypeInfo
	SectionTypeDescription
	SectionTypeImage
	SectionTypeDropdown
)

type DropdownOption struct {
	Label string
	Value string
}

type Section struct {
	Type            int
	Title           string
	ImagePaths      []string
	Metadata        []MetadataItem
	Description     string
	MaxWidth        int32
	MaxHeight       int32
	Alignment       int
	DropdownOptions []DropdownOption
	DropdownID      string
	DefaultIndex    int
}

type DetailScreenOptions struct {
	Sections            []Section
	TitleColor          sdl.Color
	MetadataColor       sdl.Color
	DescriptionColor    sdl.Color
	BackgroundColor     sdl.Color
	EnableAction        bool
	ActionButton        constants.VirtualButton
	ConfirmButton       constants.VirtualButton
	MaxImageHeight      int32
	MaxImageWidth       int32
	ShowScrollbar       bool
	ShowThemeBackground bool
	StatusBar           StatusBarOptions
}

// DropdownSelection represents the selected option from a dropdown.
type DropdownSelection struct {
	ID     string
	Option DropdownOption
	Index  int
}

// DetailScreenResult represents the result of the DetailScreen component.
type DetailScreenResult struct {
	Action             DetailAction
	DropdownSelections []DropdownSelection
}

type detailScreenState struct {
	window                 *internal.Window
	renderer               *sdl.Renderer
	options                DetailScreenOptions
	footerHelpItems        []FooterHelpItem
	scrollY                int32
	targetScrollY          int32
	maxScrollY             int32
	scrollSpeed            int32
	scrollAnimationSpeed   float32
	lastInputTime          time.Time
	inputDelay             time.Duration
	slideshowStates        map[int]slideshowState
	dropdownStates         map[string]*dropdownState
	focusedDropdownID      string
	visibleDropdownID      string
	textureCache           *internal.TextureCache
	titleTexture           *sdl.Texture
	sectionTitleTextures   []*sdl.Texture
	metadataLabelTextures  map[int][]*sdl.Texture
	heldDirections         struct{ up, down bool }
	lastRepeatTime         time.Time
	repeatDelay            time.Duration
	repeatInterval         time.Duration
	result                 DetailScreenResult
	activeSlideshow        int
	lastDirectionPressTime time.Time
	directionTimeout       time.Duration
}

type slideshowState struct {
	currentIndex int
	textures     []*sdl.Texture
	dimensions   []sdl.Rect
}

type dropdownState struct {
	selectedIndex    int
	highlightedIndex int
	expanded         bool
}

func DefaultInfoScreenOptions() DetailScreenOptions {
	return DetailScreenOptions{
		Sections:         []Section{},
		TitleColor:       sdl.Color{R: 255, G: 255, B: 255, A: 255},
		MetadataColor:    sdl.Color{R: 220, G: 220, B: 220, A: 255},
		DescriptionColor: sdl.Color{R: 200, G: 200, B: 200, A: 255},
		BackgroundColor:  sdl.Color{R: 0, G: 0, B: 0, A: 255},
		ActionButton:     constants.VirtualButtonA,
		ConfirmButton:    constants.VirtualButtonA,
		ShowScrollbar:    true,
		EnableAction:     false,
	}
}

func NewSlideshowSection(title string, imagePaths []string, maxWidth, maxHeight int32) Section {
	return Section{
		Type:       SectionTypeSlideshow,
		Title:      title,
		ImagePaths: imagePaths,
		MaxWidth:   maxWidth,
		MaxHeight:  maxHeight,
	}
}

func NewInfoSection(title string, metadata []MetadataItem) Section {
	return Section{
		Type:     SectionTypeInfo,
		Title:    title,
		Metadata: metadata,
	}
}

func NewDescriptionSection(title string, description string) Section {
	return Section{
		Type:        SectionTypeDescription,
		Title:       title,
		Description: description,
	}
}

func NewImageSection(title string, imagePath string, maxWidth, maxHeight int32, alignment constants.TextAlign) Section {
	return Section{
		Type:       SectionTypeImage,
		Title:      title,
		ImagePaths: []string{imagePath},
		MaxWidth:   maxWidth,
		MaxHeight:  maxHeight,
		Alignment:  int(alignment),
	}
}

func NewDropdownSection(title string, id string, options []DropdownOption, defaultIndex int) Section {
	return Section{
		Type:            SectionTypeDropdown,
		Title:           title,
		DropdownID:      id,
		DropdownOptions: options,
		DefaultIndex:    defaultIndex,
	}
}

// DetailScreen displays a scrollable detail screen with sections.
func DetailScreen(title string, options DetailScreenOptions, footerHelpItems []FooterHelpItem) (*DetailScreenResult, error) {
	state := initializeDetailScreenState(title, options, footerHelpItems)
	defer state.cleanup()

	for !state.isFinished() {
		state.handleEvents()
		state.update()
		state.render()
	}

	state.collectDropdownSelections()

	if state.result.Action == DetailActionCancelled {
		return nil, ErrCancelled
	}
	return &state.result, nil
}

func (s *detailScreenState) collectDropdownSelections() {
	for _, section := range s.options.Sections {
		if section.Type == SectionTypeDropdown && section.DropdownID != "" {
			if state, ok := s.dropdownStates[section.DropdownID]; ok {
				if state.selectedIndex >= 0 && state.selectedIndex < len(section.DropdownOptions) {
					s.result.DropdownSelections = append(s.result.DropdownSelections, DropdownSelection{
						ID:     section.DropdownID,
						Option: section.DropdownOptions[state.selectedIndex],
						Index:  state.selectedIndex,
					})
				}
			}
		}
	}
}

func initializeDetailScreenState(title string, options DetailScreenOptions, footerHelpItems []FooterHelpItem) *detailScreenState {
	window := internal.GetWindow()
	state := &detailScreenState{
		window:                window,
		renderer:              window.Renderer,
		options:               options,
		footerHelpItems:       footerHelpItems,
		scrollSpeed:           85,
		scrollAnimationSpeed:  0.15,
		lastInputTime:         time.Now(),
		inputDelay:            constants.DefaultInputDelay,
		slideshowStates:       make(map[int]slideshowState),
		dropdownStates:        make(map[string]*dropdownState),
		textureCache:          internal.NewTextureCache(),
		metadataLabelTextures: make(map[int][]*sdl.Texture),
		repeatDelay:           time.Millisecond * 150,
		repeatInterval:        time.Millisecond * 50,
		result:                DetailScreenResult{Action: DetailActionNone},
		directionTimeout:      time.Millisecond * 200,
	}

	state.initializeImageDefaults()
	state.loadTextures(title)
	state.initializeSlideshows()
	state.initializeDropdowns()

	return state
}

func (s *detailScreenState) initializeImageDefaults() {
	footerHeight := int32(30)
	safeAreaHeight := s.window.GetHeight() - footerHeight

	if s.options.MaxImageHeight == 0 {
		s.options.MaxImageHeight = int32(float64(safeAreaHeight) / 2)
	}
	if s.options.MaxImageWidth == 0 {
		s.options.MaxImageWidth = int32(float64(s.window.GetWidth()) / 2)
	}
}

func (s *detailScreenState) loadTextures(title string) {
	s.titleTexture = renderText(s.renderer, title, internal.Fonts.LargeFont, s.options.TitleColor)
	s.sectionTitleTextures = make([]*sdl.Texture, len(s.options.Sections))

	for i, section := range s.options.Sections {
		if section.Title != "" {
			s.sectionTitleTextures[i] = renderText(s.renderer, section.Title, internal.Fonts.MediumFont, s.options.TitleColor)
		}

		if section.Type == SectionTypeInfo {
			labelTextures := make([]*sdl.Texture, len(section.Metadata))
			for j, item := range section.Metadata {
				labelTextures[j] = renderText(s.renderer, item.Label+":", internal.Fonts.SmallFont, s.options.MetadataColor)
			}
			s.metadataLabelTextures[i] = labelTextures
		}
	}
}

func (s *detailScreenState) initializeSlideshows() {
	for i, section := range s.options.Sections {
		if section.Type == SectionTypeSlideshow || section.Type == SectionTypeImage {
			state := s.createSlideshowState(section)
			if len(state.textures) > 0 {
				s.slideshowStates[i] = state
			}
		}
	}
}

func (s *detailScreenState) initializeDropdowns() {
	for _, section := range s.options.Sections {
		if section.Type == SectionTypeDropdown && section.DropdownID != "" {
			defaultIdx := section.DefaultIndex
			if defaultIdx < 0 || defaultIdx >= len(section.DropdownOptions) {
				defaultIdx = 0
			}
			s.dropdownStates[section.DropdownID] = &dropdownState{
				selectedIndex:    defaultIdx,
				highlightedIndex: defaultIdx,
				expanded:         false,
			}
		}
	}
}

func (s *detailScreenState) createSlideshowState(section Section) slideshowState {
	maxWidth := section.MaxWidth
	maxHeight := section.MaxHeight
	if maxWidth == 0 {
		maxWidth = s.options.MaxImageWidth
	}
	if maxHeight == 0 {
		maxHeight = s.options.MaxImageHeight
	}

	imagesToLoad := section.ImagePaths
	if section.Type == SectionTypeImage && len(imagesToLoad) > 0 {
		imagesToLoad = imagesToLoad[:1]
	}

	var textures []*sdl.Texture
	var dimensions []sdl.Rect

	for _, imagePath := range imagesToLoad {
		texture, rect := s.loadAndScaleImage(imagePath, maxWidth, maxHeight, section)
		if texture != nil {
			textures = append(textures, texture)
			dimensions = append(dimensions, rect)
		}
	}

	return slideshowState{
		currentIndex: 0,
		textures:     textures,
		dimensions:   dimensions,
	}
}

func (s *detailScreenState) loadAndScaleImage(imagePath string, maxWidth, maxHeight int32, section Section) (*sdl.Texture, sdl.Rect) {
	image, err := img.Load(imagePath)
	if err != nil || image == nil {
		return nil, sdl.Rect{}
	}
	defer image.Free()

	imageW, imageH := s.calculateScaledDimensions(image.W, image.H, maxWidth, maxHeight)
	texture, err := s.renderer.CreateTextureFromSurface(image)
	if err != nil {
		return nil, sdl.Rect{}
	}

	imageX := s.calculateImageX(imageW, section)
	return texture, sdl.Rect{X: imageX, Y: 0, W: imageW, H: imageH}
}

func (s *detailScreenState) calculateScaledDimensions(originalW, originalH, maxW, maxH int32) (int32, int32) {
	imageW, imageH := originalW, originalH

	if imageW > maxW {
		ratio := float32(maxW) / float32(imageW)
		imageW = maxW
		imageH = int32(float32(imageH) * ratio)
	}

	if imageH > maxH {
		ratio := float32(maxH) / float32(imageH)
		imageH = maxH
		imageW = int32(float32(imageW) * ratio)
	}

	return imageW, imageH
}

func (s *detailScreenState) calculateImageX(imageW int32, section Section) int32 {
	if section.Type == SectionTypeImage {
		alignment := constants.TextAlign(section.Alignment)
		switch alignment {
		case constants.TextAlignLeft:
			return 20
		case constants.TextAlignRight:
			return s.window.GetWidth() - 20 - imageW
		default:
			return (s.window.GetWidth() - imageW) / 2
		}
	}
	return (s.window.GetWidth() - imageW) / 2
}

func (s *detailScreenState) isFinished() bool {
	return s.result.Action != DetailActionNone
}

func (s *detailScreenState) handleEvents() {
	processor := internal.GetInputProcessor()

	if event := sdl.WaitEventTimeout(16); event != nil {
		switch event.(type) {
		case *sdl.QuitEvent:
			s.result.Action = DetailActionCancelled
			return
		case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
			inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
			if inputEvent == nil {
				return
			}

			if inputEvent.Pressed {
				s.handleInputEvent(inputEvent)
			} else {
				s.handleInputEventRelease(inputEvent)
			}
		}
	}
}

func (s *detailScreenState) handleInputEvent(inputEvent *internal.Event) {
	if !s.isInputAllowed() {
		return
	}
	s.lastInputTime = time.Now()

	// Check if any dropdown is expanded and handle its input
	if s.handleExpandedDropdownInput(inputEvent) {
		return
	}

	switch inputEvent.Button {
	case constants.VirtualButtonUp:
		s.startScrolling(true)
	case constants.VirtualButtonDown:
		s.startScrolling(false)
	case constants.VirtualButtonLeft, constants.VirtualButtonRight:
		s.handleSlideshowNavigation(inputEvent.Button == constants.VirtualButtonLeft)
	case constants.VirtualButtonB:
		s.result.Action = DetailActionCancelled
	case constants.VirtualButtonA:
		// A button is used to expand/interact with dropdowns
		// If no dropdown was activated and A is the ConfirmButton, trigger confirm
		if !s.handleDropdownActivation() && s.options.ConfirmButton == constants.VirtualButtonA {
			s.result.Action = DetailActionConfirmed
		}
	case constants.VirtualButtonStart:
		s.result.Action = DetailActionConfirmed
	case s.options.ConfirmButton:
		// Confirm button triggers confirm action (e.g., download)
		// Skip A button since it's handled above
		if inputEvent.Button != constants.VirtualButtonA {
			s.result.Action = DetailActionConfirmed
		}
	case constants.VirtualButtonY:
		// Y button for action/options
		if s.options.EnableAction && s.options.ActionButton == constants.VirtualButtonY {
			s.result.Action = DetailActionTriggered
		}
	}
}

func (s *detailScreenState) handleExpandedDropdownInput(inputEvent *internal.Event) bool {
	// Find any expanded dropdown
	var expandedID string
	var expandedState *dropdownState
	var section *Section

	for i := range s.options.Sections {
		sec := &s.options.Sections[i]
		if sec.Type == SectionTypeDropdown && sec.DropdownID != "" {
			if state, ok := s.dropdownStates[sec.DropdownID]; ok && state.expanded {
				expandedID = sec.DropdownID
				expandedState = state
				section = sec
				break
			}
		}
	}

	if expandedState == nil {
		return false
	}

	switch inputEvent.Button {
	case constants.VirtualButtonUp:
		if expandedState.highlightedIndex > 0 {
			expandedState.highlightedIndex--
		}
		return true
	case constants.VirtualButtonDown:
		if expandedState.highlightedIndex < len(section.DropdownOptions)-1 {
			expandedState.highlightedIndex++
		}
		return true
	case constants.VirtualButtonA:
		// Select the highlighted option and collapse
		expandedState.selectedIndex = expandedState.highlightedIndex
		expandedState.expanded = false
		return true
	case constants.VirtualButtonB:
		// Just collapse without changing selection
		expandedState.highlightedIndex = expandedState.selectedIndex
		expandedState.expanded = false
		return true
	}

	_ = expandedID // Suppress unused variable warning
	return false
}

func (s *detailScreenState) handleDropdownActivation() bool {
	// Only activate a dropdown if it's currently visible
	if s.visibleDropdownID == "" {
		return false
	}

	// Find the visible dropdown and expand it
	for _, section := range s.options.Sections {
		if section.Type == SectionTypeDropdown && section.DropdownID == s.visibleDropdownID {
			if state, ok := s.dropdownStates[section.DropdownID]; ok {
				// Focus and expand this dropdown
				s.focusedDropdownID = section.DropdownID
				state.expanded = true
				state.highlightedIndex = state.selectedIndex
				return true
			}
		}
	}
	return false
}

func (s *detailScreenState) handleInputEventRelease(inputEvent *internal.Event) {
	switch inputEvent.Button {
	case constants.VirtualButtonUp:
		s.heldDirections.up = false
	case constants.VirtualButtonDown:
		s.heldDirections.down = false
	}
}

func (s *detailScreenState) isInputAllowed() bool {
	return time.Since(s.lastInputTime) >= s.inputDelay
}

func (s *detailScreenState) startScrolling(up bool) {
	if up {
		s.heldDirections.up = true
		s.heldDirections.down = false
		s.targetScrollY = internal.Max32(0, s.targetScrollY-s.scrollSpeed)
	} else {
		s.heldDirections.down = true
		s.heldDirections.up = false
		s.targetScrollY = internal.Min32(s.maxScrollY, s.targetScrollY+s.scrollSpeed)
	}
	s.lastRepeatTime = time.Now()
	s.lastDirectionPressTime = time.Now()
}

func (s *detailScreenState) handleSlideshowNavigation(isLeft bool) {
	activeSlideshow := s.findActiveSlideshow()
	if activeSlideshow >= 0 {
		if state, ok := s.slideshowStates[activeSlideshow]; ok && len(state.textures) > 1 {
			if isLeft {
				state.currentIndex = (state.currentIndex - 1 + len(state.textures)) % len(state.textures)
			} else {
				state.currentIndex = (state.currentIndex + 1) % len(state.textures)
			}
			s.slideshowStates[activeSlideshow] = state
		}
	}
}

func (s *detailScreenState) findActiveSlideshow() int {
	return s.activeSlideshow
}

func (s *detailScreenState) update() {
	s.handleDirectionalRepeats()
	s.scrollY += int32(float32(s.targetScrollY-s.scrollY) * s.scrollAnimationSpeed)
}

func (s *detailScreenState) handleDirectionalRepeats() {
	now := time.Now()

	// Reset held directions if no input received recently (handles missing release events)
	if now.Sub(s.lastDirectionPressTime) > s.directionTimeout {
		s.heldDirections.up = false
		s.heldDirections.down = false
		return
	}

	timeSinceLastRepeat := now.Sub(s.lastRepeatTime)

	if timeSinceLastRepeat < s.repeatDelay {
		return
	}
	if s.repeatInterval > 0 && timeSinceLastRepeat < s.repeatInterval {
		return
	}

	if s.heldDirections.up {
		s.targetScrollY = internal.Max32(0, s.targetScrollY-s.scrollSpeed)
		s.lastRepeatTime = now
		s.lastDirectionPressTime = now
	} else if s.heldDirections.down {
		s.targetScrollY = internal.Min32(s.maxScrollY, s.targetScrollY+s.scrollSpeed)
		s.lastRepeatTime = now
		s.lastDirectionPressTime = now
	}
}

func (s *detailScreenState) render() {
	s.clearScreen()

	margins := internal.UniformPadding(20)
	footerHeight := int32(30)
	safeAreaHeight := s.window.GetHeight() - footerHeight

	statusBarWidth := calculateStatusBarWidth(internal.Fonts.SmallFont, s.options.StatusBar)

	currentY := s.renderTitle(margins, statusBarWidth)
	currentY, totalContentHeight := s.renderSections(margins, currentY, safeAreaHeight)

	renderStatusBar(s.renderer, internal.Fonts.SmallFont, s.options.StatusBar, margins)

	s.updateScrollLimits(totalContentHeight, safeAreaHeight, margins)
	s.renderScrollbar(safeAreaHeight)
	s.renderFooter(margins)

	s.renderer.Present()
}

func (s *detailScreenState) clearScreen() {
	s.renderer.SetDrawColor(
		s.options.BackgroundColor.R,
		s.options.BackgroundColor.G,
		s.options.BackgroundColor.B,
		s.options.BackgroundColor.A)
	s.renderer.Clear()

	if s.options.ShowThemeBackground {
		s.window.RenderBackground()
	}
}

func (s *detailScreenState) renderTitle(margins internal.Padding, statusBarWidth int32) int32 {
	if s.titleTexture == nil {
		return margins.Top + constants.DefaultTitleSpacing - s.scrollY
	}

	_, _, titleW, titleH, err := s.titleTexture.Query()
	if err != nil {
		return margins.Top + constants.DefaultTitleSpacing - s.scrollY
	}

	maxTitleWidth := s.window.GetWidth() - margins.Left - margins.Right - statusBarWidth
	displayWidth := titleW
	if displayWidth > maxTitleWidth {
		displayWidth = maxTitleWidth
	}

	// Center the title horizontally
	titleX := (s.window.GetWidth() - displayWidth) / 2

	titleRect := sdl.Rect{
		X: titleX,
		Y: margins.Top - s.scrollY,
		W: displayWidth,
		H: titleH,
	}

	if isRectVisible(titleRect, s.window.GetHeight()) {
		srcRect := &sdl.Rect{X: 0, Y: 0, W: displayWidth, H: titleH}
		s.renderer.Copy(s.titleTexture, srcRect, &titleRect)
	}

	return margins.Top + titleH + constants.DefaultTitleSpacing - s.scrollY
}

func (s *detailScreenState) renderSections(margins internal.Padding, startY int32, safeAreaHeight int32) (int32, int32) {
	currentY := startY
	contentWidth := s.window.GetWidth() - (margins.Left + margins.Right)

	// Reserve space for scrollbar to prevent text overlap
	if s.options.ShowScrollbar {
		scrollbarWidth := int32(10)
		scrollbarMargin := int32(5)
		scrollbarPadding := int32(10) // Extra padding between content and scrollbar
		contentWidth -= (scrollbarWidth + scrollbarMargin + scrollbarPadding)
	}

	s.activeSlideshow = -1
	s.visibleDropdownID = ""

	for sectionIndex, section := range s.options.Sections {
		if sectionIndex > 0 {
			currentY += 30
		}

		currentY = s.renderSectionTitle(sectionIndex, margins, currentY, safeAreaHeight)
		currentY = s.renderSectionDivider(margins, contentWidth, currentY, safeAreaHeight)
		currentY = s.renderSectionContent(sectionIndex, section, margins, contentWidth, currentY, safeAreaHeight)
	}

	return currentY, currentY + s.scrollY + margins.Bottom
}

func (s *detailScreenState) renderSectionTitle(sectionIndex int, margins internal.Padding, currentY int32, safeAreaHeight int32) int32 {
	if sectionIndex >= len(s.sectionTitleTextures) || s.sectionTitleTextures[sectionIndex] == nil {
		return currentY
	}

	texture := s.sectionTitleTextures[sectionIndex]
	_, _, titleW, titleH, err := texture.Query()
	if err != nil {
		return currentY
	}

	sectionTitleRect := sdl.Rect{
		X: margins.Left,
		Y: currentY,
		W: titleW,
		H: titleH,
	}

	if isRectVisible(sectionTitleRect, safeAreaHeight) {
		s.renderer.Copy(texture, nil, &sectionTitleRect)
	}

	return currentY + titleH + 15
}

func (s *detailScreenState) renderSectionDivider(margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	if isLineVisible(currentY, safeAreaHeight) {
		s.renderer.SetDrawColor(80, 80, 80, 255)
		s.renderer.DrawLine(margins.Left, currentY, margins.Left+contentWidth, currentY)
	}
	return currentY + 15
}

func (s *detailScreenState) renderSectionContent(sectionIndex int, section Section, margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	switch section.Type {
	case SectionTypeSlideshow:
		return s.renderSlideshow(sectionIndex, currentY, safeAreaHeight)
	case SectionTypeImage:
		return s.renderImage(sectionIndex, currentY, safeAreaHeight)
	case SectionTypeInfo:
		return s.renderInfo(sectionIndex, section, margins, contentWidth, currentY, safeAreaHeight)
	case SectionTypeDescription:
		return s.renderDescription(section, margins, contentWidth, currentY, safeAreaHeight)
	case SectionTypeDropdown:
		return s.renderDropdown(section, margins, contentWidth, currentY, safeAreaHeight)
	}
	return currentY
}

func (s *detailScreenState) renderSlideshow(sectionIndex int, currentY int32, safeAreaHeight int32) int32 {
	state, ok := s.slideshowStates[sectionIndex]
	if !ok || len(state.textures) == 0 {
		return currentY
	}

	imageRect := state.dimensions[state.currentIndex]
	imageRect.Y = currentY

	if isRectVisible(imageRect, safeAreaHeight) {
		s.renderer.Copy(state.textures[state.currentIndex], nil, &imageRect)
		// Set this as the active slideshow when it's being rendered and visible
		s.activeSlideshow = sectionIndex
	}

	currentY += imageRect.H + 15

	if len(state.textures) > 1 {
		currentY = s.renderSlideshowIndicators(state, currentY)
	}

	return currentY
}

func (s *detailScreenState) renderSlideshowIndicators(state slideshowState, currentY int32) int32 {
	indicatorSize := int32(10)
	indicatorSpacing := int32(5)
	totalIndicatorsWidth := (indicatorSize * int32(len(state.textures))) + (indicatorSpacing * int32(len(state.textures)-1))

	indicatorX := (s.window.GetWidth() - totalIndicatorsWidth) / 2
	indicatorY := currentY

	for i := 0; i < len(state.textures); i++ {
		if i == state.currentIndex {
			s.renderer.SetDrawColor(255, 255, 255, 255)
		} else {
			s.renderer.SetDrawColor(150, 150, 150, 150)
		}

		indicatorRect := sdl.Rect{
			X: indicatorX,
			Y: indicatorY,
			W: indicatorSize,
			H: indicatorSize,
		}

		s.renderer.FillRect(&indicatorRect)
		indicatorX += indicatorSize + indicatorSpacing
	}

	return currentY + indicatorSize + 15
}

func (s *detailScreenState) renderImage(sectionIndex int, currentY int32, safeAreaHeight int32) int32 {
	state, ok := s.slideshowStates[sectionIndex]
	if !ok || len(state.textures) == 0 {
		return currentY
	}

	imageRect := state.dimensions[0]
	imageRect.Y = currentY

	if isRectVisible(imageRect, safeAreaHeight) {
		s.renderer.Copy(state.textures[0], nil, &imageRect)
	}

	return currentY + imageRect.H + 15
}

func (s *detailScreenState) renderInfo(sectionIndex int, section Section, margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	labelTextures, ok := s.metadataLabelTextures[sectionIndex]
	if !ok {
		return currentY
	}

	for j, item := range section.Metadata {
		if j >= len(labelTextures) || labelTextures[j] == nil {
			continue
		}

		currentY = s.renderMetadataItem(labelTextures[j], item, margins, contentWidth, currentY, safeAreaHeight)
	}

	return currentY + 5
}

func (s *detailScreenState) renderMetadataItem(labelTexture *sdl.Texture, item MetadataItem, margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	_, _, labelWidth, labelHeight, _ := labelTexture.Query()
	labelRect := sdl.Rect{
		X: margins.Left,
		Y: currentY,
		W: labelWidth,
		H: labelHeight,
	}

	if isRectVisible(labelRect, safeAreaHeight) {
		s.renderer.Copy(labelTexture, nil, &labelRect)
	}

	if item.Value != "" {
		valueX := margins.Left + labelWidth + 10
		maxValueWidth := contentWidth - labelWidth - 10
		valueHeight := calculateMultilineTextHeight(item.Value, internal.Fonts.SmallFont, maxValueWidth)

		if valueHeight > 0 && isRectVisible(sdl.Rect{X: valueX, Y: currentY, W: maxValueWidth, H: valueHeight}, safeAreaHeight) {
			internal.RenderMultilineTextWithCache(
				s.renderer,
				item.Value,
				internal.Fonts.SmallFont,
				maxValueWidth,
				valueX,
				currentY,
				s.options.MetadataColor,
				constants.TextAlignLeft,
				s.textureCache)
		}

		return currentY + internal.Max32(labelHeight, valueHeight) + 10
	}

	return currentY + labelHeight + 10
}

func (s *detailScreenState) renderDescription(section Section, margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	if section.Description == "" {
		return currentY
	}

	// Add extra padding for description text to prevent overlap with scrollbar
	descriptionPadding := int32(15)
	descriptionX := margins.Left + descriptionPadding
	descriptionWidth := contentWidth - (descriptionPadding * 2)

	descHeight := calculateMultilineTextHeight(section.Description, internal.Fonts.SmallFont, descriptionWidth)
	if descHeight > 0 && isRectVisible(sdl.Rect{X: descriptionX, Y: currentY, W: descriptionWidth, H: descHeight}, safeAreaHeight) {
		internal.RenderMultilineTextWithCache(
			s.renderer,
			section.Description,
			internal.Fonts.SmallFont,
			descriptionWidth,
			descriptionX,
			currentY,
			s.options.DescriptionColor,
			constants.TextAlignLeft,
			s.textureCache)
	}

	return currentY + descHeight + 15
}

func (s *detailScreenState) renderDropdown(section Section, margins internal.Padding, contentWidth, currentY int32, safeAreaHeight int32) int32 {
	if section.DropdownID == "" || len(section.DropdownOptions) == 0 {
		return currentY
	}

	state, ok := s.dropdownStates[section.DropdownID]
	if !ok {
		return currentY
	}

	dropdownX := margins.Left
	dropdownWidth := contentWidth
	itemHeight := int32(35)
	isFocused := s.focusedDropdownID == section.DropdownID

	if state.expanded {
		return s.renderExpandedDropdown(section, state, dropdownX, dropdownWidth, itemHeight, currentY, safeAreaHeight)
	}
	return s.renderCollapsedDropdown(section, state, dropdownX, dropdownWidth, itemHeight, currentY, safeAreaHeight, isFocused)
}

func (s *detailScreenState) renderCollapsedDropdown(section Section, state *dropdownState, dropdownX, dropdownWidth, itemHeight, currentY int32, safeAreaHeight int32, isFocused bool) int32 {
	// Draw background
	bgColor := sdl.Color{R: 40, G: 40, B: 40, A: 255}
	if isFocused {
		bgColor = sdl.Color{R: 60, G: 60, B: 80, A: 255}
	}

	dropdownRect := sdl.Rect{X: dropdownX, Y: currentY, W: dropdownWidth, H: itemHeight}
	if isRectVisible(dropdownRect, safeAreaHeight) {
		// Mark this dropdown as visible for input handling
		s.visibleDropdownID = section.DropdownID

		s.renderer.SetDrawColor(bgColor.R, bgColor.G, bgColor.B, bgColor.A)
		s.renderer.FillRect(&dropdownRect)

		// Draw border
		borderColor := sdl.Color{R: 80, G: 80, B: 80, A: 255}
		if isFocused {
			borderColor = sdl.Color{R: 100, G: 100, B: 200, A: 255}
		}
		s.renderer.SetDrawColor(borderColor.R, borderColor.G, borderColor.B, borderColor.A)
		s.renderer.DrawRect(&dropdownRect)

		// Draw selected option text
		if state.selectedIndex >= 0 && state.selectedIndex < len(section.DropdownOptions) {
			selectedOption := section.DropdownOptions[state.selectedIndex]
			textX := dropdownX + 10
			textY := currentY + (itemHeight-int32(internal.Fonts.SmallFont.Height()))/2

			internal.RenderMultilineTextWithCache(
				s.renderer,
				selectedOption.Label,
				internal.Fonts.SmallFont,
				dropdownWidth-50,
				textX,
				textY,
				sdl.Color{R: 255, G: 255, B: 255, A: 255},
				constants.TextAlignLeft,
				s.textureCache)
		}

		// Draw dropdown arrow indicator
		arrowX := dropdownX + dropdownWidth - 25
		arrowY := currentY + itemHeight/2
		s.renderDropdownArrow(arrowX, arrowY, false)
	}

	return currentY + itemHeight + 15
}

func (s *detailScreenState) renderExpandedDropdown(section Section, state *dropdownState, dropdownX, dropdownWidth, itemHeight, currentY int32, safeAreaHeight int32) int32 {
	totalHeight := itemHeight * int32(len(section.DropdownOptions))

	// Draw background for all options
	bgRect := sdl.Rect{X: dropdownX, Y: currentY, W: dropdownWidth, H: totalHeight}
	if isRectVisible(bgRect, safeAreaHeight) {
		// Mark this dropdown as visible for input handling
		s.visibleDropdownID = section.DropdownID

		s.renderer.SetDrawColor(40, 40, 40, 255)
		s.renderer.FillRect(&bgRect)

		// Draw border
		s.renderer.SetDrawColor(100, 100, 200, 255)
		s.renderer.DrawRect(&bgRect)
	}

	// Draw each option
	for i, option := range section.DropdownOptions {
		optionY := currentY + int32(i)*itemHeight
		optionRect := sdl.Rect{X: dropdownX, Y: optionY, W: dropdownWidth, H: itemHeight}

		if isRectVisible(optionRect, safeAreaHeight) {
			// Highlight the currently highlighted option
			if i == state.highlightedIndex {
				s.renderer.SetDrawColor(80, 80, 120, 255)
				s.renderer.FillRect(&optionRect)
			}

			// Draw checkmark for selected option
			if i == state.selectedIndex {
				checkX := dropdownX + 8
				checkY := optionY + (itemHeight-int32(internal.Fonts.SmallFont.Height()))/2
				internal.RenderMultilineTextWithCache(
					s.renderer,
					"*",
					internal.Fonts.SmallFont,
					20,
					checkX,
					checkY,
					sdl.Color{R: 100, G: 200, B: 100, A: 255},
					constants.TextAlignLeft,
					s.textureCache)
			}

			// Draw option text
			textX := dropdownX + 25
			textY := optionY + (itemHeight-int32(internal.Fonts.SmallFont.Height()))/2

			textColor := sdl.Color{R: 200, G: 200, B: 200, A: 255}
			if i == state.highlightedIndex {
				textColor = sdl.Color{R: 255, G: 255, B: 255, A: 255}
			}

			internal.RenderMultilineTextWithCache(
				s.renderer,
				option.Label,
				internal.Fonts.SmallFont,
				dropdownWidth-40,
				textX,
				textY,
				textColor,
				constants.TextAlignLeft,
				s.textureCache)
		}
	}

	return currentY + totalHeight + 15
}

func (s *detailScreenState) renderDropdownArrow(x, y int32, expanded bool) {
	// Draw a simple triangle arrow
	s.renderer.SetDrawColor(200, 200, 200, 255)
	if expanded {
		// Up arrow
		s.renderer.DrawLine(x, y+3, x+5, y-3)
		s.renderer.DrawLine(x+5, y-3, x+10, y+3)
	} else {
		// Down arrow
		s.renderer.DrawLine(x, y-3, x+5, y+3)
		s.renderer.DrawLine(x+5, y+3, x+10, y-3)
	}
}

func (s *detailScreenState) updateScrollLimits(totalContentHeight int32, safeAreaHeight int32, margins internal.Padding) {
	s.maxScrollY = internal.Max32(0, totalContentHeight-safeAreaHeight+margins.Bottom)
}

func (s *detailScreenState) renderScrollbar(safeAreaHeight int32) {
	if !s.options.ShowScrollbar || s.maxScrollY <= 0 {
		return
	}

	scrollbarWidth := int32(10)
	trackY := int32(5)
	trackHeight := safeAreaHeight - 10

	// Calculate handle height proportional to visible content
	totalContentHeight := s.maxScrollY + safeAreaHeight
	handleHeight := int32(float64(trackHeight) * float64(safeAreaHeight) / float64(totalContentHeight))

	// Clamp handle height between reasonable bounds
	handleHeight = internal.Max32(handleHeight, 20)
	handleHeight = internal.Min32(handleHeight, trackHeight/3) // Handle should be at most 1/3 of track

	// Calculate handle position within track bounds
	var handleY int32
	if s.scrollY >= s.maxScrollY {
		handleY = trackHeight - handleHeight
	} else if s.scrollY <= 0 {
		handleY = 0
	} else {
		handleY = int32(float64(s.scrollY) * float64(trackHeight-handleHeight) / float64(s.maxScrollY))
	}

	scrollbarX := s.window.GetWidth() - scrollbarWidth - 5

	// Clear the scrollbar area first to prevent anti-aliasing artifacts
	s.renderer.SetDrawColor(
		s.options.BackgroundColor.R,
		s.options.BackgroundColor.G,
		s.options.BackgroundColor.B,
		255)
	s.renderer.FillRect(&sdl.Rect{
		X: scrollbarX - 2,
		Y: trackY - 2,
		W: scrollbarWidth + 4,
		H: trackHeight + 4,
	})

	// Draw scrollbar track
	internal.DrawSmoothScrollbar(s.renderer, scrollbarX, trackY, scrollbarWidth, trackHeight, sdl.Color{R: 50, G: 50, B: 50, A: 255})

	// Draw scrollbar handle
	internal.DrawSmoothScrollbar(s.renderer, scrollbarX, trackY+handleY, scrollbarWidth, handleHeight, sdl.Color{R: 100, G: 100, B: 100, A: 255})
}

func (s *detailScreenState) renderFooter(margins internal.Padding) {
	if len(s.footerHelpItems) > 0 {
		renderFooter(
			s.renderer,
			internal.Fonts.SmallFont,
			s.footerHelpItems,
			margins.Bottom,
			false,
			true,
		)
	}
}

func (s *detailScreenState) cleanup() {
	s.textureCache.Destroy()

	if s.titleTexture != nil {
		s.titleTexture.Destroy()
	}

	for _, texture := range s.sectionTitleTextures {
		if texture != nil {
			texture.Destroy()
		}
	}

	for _, textures := range s.metadataLabelTextures {
		for _, texture := range textures {
			if texture != nil {
				texture.Destroy()
			}
		}
	}

	for _, state := range s.slideshowStates {
		for _, texture := range state.textures {
			texture.Destroy()
		}
	}
}

func renderText(renderer *sdl.Renderer, text string, font *ttf.Font, color sdl.Color) *sdl.Texture {
	if text == "" {
		return nil
	}

	surface, err := font.RenderUTF8Blended(text, color)
	if err != nil {
		return nil
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return nil
	}

	return texture
}

func isRectVisible(rect sdl.Rect, viewportHeight int32) bool {
	if rect.Y+rect.H < 0 || rect.Y > viewportHeight {
		return false
	}
	return true
}

func isLineVisible(y, viewportHeight int32) bool {
	if y < 0 || y > viewportHeight {
		return false
	}
	return true
}

func calculateMultilineTextHeight(text string, font *ttf.Font, maxWidth int32) int32 {
	if text == "" {
		return 0
	}

	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")

	_, fontHeight, err := font.SizeUTF8("Aj")
	if err != nil {
		fontHeight = 20
	}

	lineSpacing := int32(float32(fontHeight) * 0.3)
	totalHeight := int32(0)

	for _, line := range lines {
		if line == "" {
			totalHeight += int32(fontHeight) + lineSpacing
			continue
		}

		remainingText := line
		for len(remainingText) > 0 {
			width, _, err := font.SizeUTF8(remainingText)
			if err != nil || int32(width) <= maxWidth {
				totalHeight += int32(fontHeight) + lineSpacing
				break
			}

			charsPerLine := int(float32(len(remainingText)) * float32(maxWidth) / float32(width))
			if charsPerLine <= 0 {
				charsPerLine = 1
			}

			if charsPerLine < len(remainingText) {
				for i := charsPerLine; i > 0; i-- {
					if i < len(remainingText) && remainingText[i] == ' ' {
						charsPerLine = i
						break
					}
				}
			}

			totalHeight += int32(fontHeight) + lineSpacing
			if charsPerLine >= len(remainingText) {
				break
			}
			remainingText = remainingText[charsPerLine:]
			remainingText = strings.TrimLeft(remainingText, " ")
		}
	}

	if totalHeight > lineSpacing {
		totalHeight -= lineSpacing
	}

	totalHeight += 20

	return totalHeight
}

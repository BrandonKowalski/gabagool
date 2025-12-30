package gabagool

import (
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type MessageOptions struct {
	ImagePath     string
	ConfirmButton constants.VirtualButton
	CancelButton  constants.VirtualButton
	StatusBar     StatusBarOptions
}

// ConfirmationResult represents the result of a confirmation message.
type ConfirmationResult struct {
	Confirmed bool
}

type confirmationMessageSettings struct {
	Margins          internal.Padding
	MessageText      string
	MessageAlign     constants.TextAlign
	ButtonSpacing    int32
	ConfirmButton    constants.VirtualButton
	CancelButton     constants.VirtualButton
	ImagePath        string
	MaxImageHeight   int32
	MaxImageWidth    int32
	BackgroundColor  sdl.Color
	MessageTextColor sdl.Color
	FooterText       string
	FooterHelpItems  []FooterHelpItem
	FooterTextColor  sdl.Color
	InputDelay       time.Duration
	StatusBar        StatusBarOptions
}

func defaultMessageSettings(message string) confirmationMessageSettings {
	return confirmationMessageSettings{
		Margins:          internal.UniformPadding(20),
		MessageText:      message,
		MessageAlign:     constants.TextAlignCenter,
		ButtonSpacing:    20,
		ConfirmButton:    constants.VirtualButtonA,
		CancelButton:     constants.VirtualButtonB,
		BackgroundColor:  sdl.Color{R: 0, G: 0, B: 0, A: 255},
		MessageTextColor: sdl.Color{R: 255, G: 255, B: 255, A: 255},
		FooterTextColor:  sdl.Color{R: 180, G: 180, B: 180, A: 255},
		InputDelay:       constants.DefaultInputDelay,
		FooterHelpItems:  []FooterHelpItem{},
		StatusBar:        DefaultStatusBarOptions(),
	}
}

// ConfirmationMessage displays a confirmation dialog.
// Returns ErrCancelled if the user cancels or presses the cancel button.
func ConfirmationMessage(message string, footerHelpItems []FooterHelpItem, options MessageOptions) (*ConfirmationResult, error) {
	window := internal.GetWindow()
	renderer := window.Renderer

	settings := defaultMessageSettings(message)
	settings.FooterHelpItems = footerHelpItems

	if options.ImagePath != "" {
		settings.ImagePath = options.ImagePath
		settings.MaxImageWidth = int32(float64(window.GetWidth()) / 1.75)
		settings.MaxImageHeight = int32(float64(window.GetHeight()) / 1.75)
	}

	if options.ConfirmButton != constants.VirtualButtonUnassigned {
		settings.ConfirmButton = options.ConfirmButton
	}

	if options.CancelButton != constants.VirtualButtonUnassigned {
		settings.CancelButton = options.CancelButton
	}

	settings.StatusBar = options.StatusBar

	result := ConfirmationResult{Confirmed: false}
	lastInputTime := time.Now()

	imageTexture, imageRect := loadAndPrepareImage(renderer, settings)
	defer func() {
		if imageTexture != nil {
			imageTexture.Destroy()
		}
	}()

	for {
		if !handleEvents(&result, &lastInputTime, settings) {
			break
		}

		renderFrame(renderer, window, settings, imageTexture, imageRect)
	}

	if !result.Confirmed {
		return nil, ErrCancelled
	}
	return &result, nil
}

func loadAndPrepareImage(renderer *sdl.Renderer, settings confirmationMessageSettings) (*sdl.Texture, sdl.Rect) {
	if settings.ImagePath == "" {
		return nil, sdl.Rect{}
	}

	image, err := img.Load(settings.ImagePath)
	if err != nil {
		return nil, sdl.Rect{}
	}
	defer image.Free()

	imageTexture, err := renderer.CreateTextureFromSurface(image)
	if err != nil {
		return nil, sdl.Rect{}
	}

	widthScale := float32(settings.MaxImageWidth) / float32(image.W)
	heightScale := float32(settings.MaxImageHeight) / float32(image.H)
	scale := widthScale
	if heightScale < widthScale {
		scale = heightScale
	}

	imageW := int32(float32(image.W) * scale)
	imageH := int32(float32(image.H) * scale)

	return imageTexture, sdl.Rect{
		W: imageW,
		H: imageH,
	}
}

func handleEvents(result *ConfirmationResult, lastInputTime *time.Time, settings confirmationMessageSettings) bool {
	processor := internal.GetInputProcessor()

	if event := sdl.WaitEventTimeout(16); event != nil {
		switch event.(type) {
		case *sdl.QuitEvent:
			result.Confirmed = false
			return false

		case *sdl.KeyboardEvent, *sdl.ControllerButtonEvent, *sdl.ControllerAxisEvent, *sdl.JoyButtonEvent, *sdl.JoyAxisEvent, *sdl.JoyHatEvent:
			inputEvent := processor.ProcessSDLEvent(event.(sdl.Event))
			if inputEvent == nil || !inputEvent.Pressed {
				return true
			}

			if !isInputAllowed(*lastInputTime, settings.InputDelay) {
				return true
			}

			*lastInputTime = time.Now()

			switch inputEvent.Button {
			case settings.ConfirmButton, constants.VirtualButtonStart:
				result.Confirmed = true
				return false
			case settings.CancelButton:
				result.Confirmed = false
				return false
			}
		}
	}
	return true
}

func isInputAllowed(lastInputTime time.Time, inputDelay time.Duration) bool {
	return time.Since(lastInputTime) >= inputDelay
}

func renderFrame(renderer *sdl.Renderer, window *internal.Window, settings confirmationMessageSettings, imageTexture *sdl.Texture, imageRect sdl.Rect) {
	renderer.SetDrawColor(
		settings.BackgroundColor.R,
		settings.BackgroundColor.G,
		settings.BackgroundColor.B,
		settings.BackgroundColor.A)
	renderer.Clear()

	windowWidth := window.GetWidth()
	windowHeight := window.GetHeight()
	responsiveMaxWidth := int32(float64(windowWidth) * 0.75)
	if responsiveMaxWidth > 800 {
		responsiveMaxWidth = 800
	}

	contentHeight := calculateContentHeight(settings, imageRect)
	startY := (windowHeight - contentHeight) / 2

	if imageTexture != nil {
		imageRect.X = (windowWidth - imageRect.W) / 2
		imageRect.Y = startY
		renderer.Copy(imageTexture, nil, &imageRect)
		startY = imageRect.Y + imageRect.H + 30
	}

	if len(settings.MessageText) > 0 {
		centerX := windowWidth / 2
		internal.RenderMultilineText(
			renderer,
			settings.MessageText,
			internal.Fonts.SmallFont,
			responsiveMaxWidth,
			centerX,
			startY,
			settings.MessageTextColor,
			constants.TextAlignCenter)
	}

	renderStatusBar(renderer, internal.Fonts.SmallFont, settings.StatusBar, settings.Margins)

	renderFooter(
		renderer,
		internal.Fonts.SmallFont,
		settings.FooterHelpItems,
		settings.Margins.Bottom,
		false,
		true,
	)

	renderer.Present()
}

func calculateContentHeight(settings confirmationMessageSettings, imageRect sdl.Rect) int32 {
	var contentHeight int32

	if imageRect.W > 0 {
		contentHeight += imageRect.H + 30
	}

	if len(settings.MessageText) > 0 {
		contentHeight += 30
	}

	return contentHeight
}

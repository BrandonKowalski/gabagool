package internal

import (
	"os"
	"strconv"
	"sync"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

// Window wraps SDL window and renderer with additional state for the UI framework.
type Window struct {
	Window            *sdl.Window
	Renderer          *sdl.Renderer
	Title             string
	FontSize          int
	SmallFontSize     int
	Background        *sdl.Texture
	DisplayBackground bool
	PowerButtonWG     sync.WaitGroup
	PowerButtonConfig PowerButtonConfig
	hasVSync          bool
	lastPresentTime   uint64
}

func initWindow(title string, displayBackground bool, winOpts WindowOptions) *Window {
	displayIndex := 0
	displayMode, err := sdl.GetCurrentDisplayMode(displayIndex)

	if err != nil {
		GetInternalLogger().Error("Failed to Get display mode!", "error", err)
	}

	return initWindowWithSize(title, displayMode.W, displayMode.H, displayBackground, winOpts)
}

func initWindowWithSize(title string, width, height int32, displayBackground bool, winOpts WindowOptions) *Window {
	x, y := int32(0), int32(0)

	if constants.IsDevMode() {
		winOpts.Borderless = false

		x, y = int32(50), int32(50)
		if v := os.Getenv(constants.WindowWidthEnvVar); v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				width = int32(n)
			} else {
				GetInternalLogger().Warn("Invalid WINDOW_WIDTH; using default", "value", v, "error", err)
				width = 1024
			}
		} else {
			width = 1024
		}

		if v := os.Getenv(constants.WindowHeightEnvVar); v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				height = int32(n)
			} else {
				GetInternalLogger().Warn("Invalid WINDOW_HEIGHT; using default", "value", v, "error", err)
				height = 768
			}
		} else {
			height = 768
		}
	}

	windowFlags := winOpts.ToSDLFlags()

	GetInternalLogger().Debug("Initializing SDL Window", "width", width, "height", height)

	window, err := sdl.CreateWindow(title, x, y, width, height, windowFlags)
	if err != nil {
		panic(err)
	}

	var renderer *sdl.Renderer
	var lastErr error

	renderer, lastErr = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC|sdl.RENDERER_TARGETTEXTURE)

	if lastErr != nil {
		GetInternalLogger().Error("Failed to create any renderer!", "final_error", lastErr)
		os.Exit(1)
	}

	renderer.SetLogicalSize(width, height)

	info, err := renderer.GetInfo()
	vsync := err == nil && info.Flags&sdl.RENDERER_PRESENTVSYNC != 0

	win := &Window{
		Window:            window,
		Renderer:          renderer,
		Title:             title,
		DisplayBackground: displayBackground,
		hasVSync:          vsync,
	}

	win.loadBackground()

	return win
}

func (window *Window) initPowerButtonHandling(pbc PowerButtonConfig) {
	window.PowerButtonWG.Add(1)

	go PowerButtonHandler(&window.PowerButtonWG, pbc)
}

func (window *Window) loadBackground() {
	img.Init(img.INIT_PNG)

	theme := GetTheme()

	bgTexture, err := img.LoadTexture(window.Renderer, theme.BackgroundImagePath)
	if err == nil {
		window.Background = bgTexture
	} else {
		window.Background = nil
	}
}

func (window *Window) closeWindow() {
	if !constants.IsDevMode() {
		window.PowerButtonWG.Done()
	}

	if window.Background != nil {
		window.Background.Destroy()
	}
	window.Renderer.Destroy()
	window.Window.Destroy()

	img.Quit()
}

func GetWindow() *Window {
	return window
}

func (window *Window) GetWidth() int32 {
	w, _ := window.Window.GetSize()
	return w
}

func (window *Window) GetHeight() int32 {
	_, h := window.Window.GetSize()
	return h
}

func (window *Window) RenderBackground() {
	if window.Background != nil {
		window.Renderer.Copy(window.Background, nil, &sdl.Rect{X: 0, Y: 0, W: window.GetWidth(), H: window.GetHeight()})
	}
}

// Present swaps the render buffer and enforces ~60fps frame timing
// when VSync is not available. Use this instead of renderer.Present().
func (w *Window) Present() {
	w.Renderer.Present()
	if !w.hasVSync {
		now := sdl.GetTicks64()
		if elapsed := now - w.lastPresentTime; elapsed < 16 {
			sdl.Delay(uint32(16 - elapsed))
		}
		w.lastPresentTime = sdl.GetTicks64()
	}
}

func ResetBackground() {
	window.loadBackground()
}

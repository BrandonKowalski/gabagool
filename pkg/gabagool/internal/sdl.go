package internal

import (
	"os"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var window *Window

func Init(title string, showBackground bool, winOpts WindowOptions, pbc PowerButtonConfig) {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO |
		img.INIT_PNG | img.INIT_JPG | img.INIT_TIF | img.INIT_WEBP |
		sdl.INIT_GAMECONTROLLER | sdl.INIT_JOYSTICK); err != nil {
		os.Exit(1)
	}

	if err := ttf.Init(); err != nil {
		os.Exit(1)
	}

	InitInputProcessor()

	// Apply default window options if none specified
	if winOpts.IsZero() {
		if constants.IsDevMode() {
			winOpts = WindowOptions{Borderless: true, Resizable: true}
		} else {
			winOpts = WindowOptions{Resizable: true}
		}
	}

	window = initWindow(title, showBackground, winOpts)

	initFonts(DefaultFontSizes)

	if !constants.IsDevMode() && pbc.DevicePath != "" {
		window.initPowerButtonHandling(pbc)
	}
}

func SDLCleanup() {
	window.closeWindow()
	CloseAllControllers()
	closeFonts()
	ttf.Quit()
	img.Quit()
	sdl.Quit()
	CloseLogger()
}

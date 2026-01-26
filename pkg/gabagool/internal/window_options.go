package internal

import "github.com/veandco/go-sdl2/sdl"

type WindowOptions struct {
	Borderless        bool // Remove window decorations (SDL_WINDOW_BORDERLESS)
	Resizable         bool // Allow window resizing (SDL_WINDOW_RESIZABLE)
	Fullscreen        bool // Fullscreen mode (SDL_WINDOW_FULLSCREEN)
	FullscreenDesktop bool // Fullscreen at desktop resolution (SDL_WINDOW_FULLSCREEN_DESKTOP)
	AlwaysOnTop       bool // Window stays above others (SDL_WINDOW_ALWAYS_ON_TOP)
	Maximized         bool // Start maximized (SDL_WINDOW_MAXIMIZED)
	Hidden            bool // Start hidden (omits SDL_WINDOW_SHOWN)
}

func (wo WindowOptions) IsZero() bool {
	return wo == WindowOptions{}
}

func (wo WindowOptions) ToSDLFlags() uint32 {
	var flags uint32

	if !wo.Hidden {
		flags |= sdl.WINDOW_SHOWN
	}

	if wo.Resizable {
		flags |= sdl.WINDOW_RESIZABLE
	}

	if wo.Borderless {
		flags |= sdl.WINDOW_BORDERLESS
	}

	if wo.Fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN
	}

	if wo.FullscreenDesktop {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}

	if wo.AlwaysOnTop {
		flags |= sdl.WINDOW_ALWAYS_ON_TOP
	}

	if wo.Maximized {
		flags |= sdl.WINDOW_MAXIMIZED
	}

	return flags
}

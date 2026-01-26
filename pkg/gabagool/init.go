// Package gabagool provides a UI framework for building graphical applications
// on embedded Linux devices, particularly handheld gaming consoles running
// custom firmware like NextUI or Cannoli.
//
// The package handles SDL initialization, input processing, theming, and provides
// various UI components including lists, detail views, keyboards, and dialogs.
package gabagool

import (
	"log/slog"
	"os"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/platform/cannoli"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/platform/nextui"
)

// Options configures the gabagool UI framework initialization.
type Options struct {
	WindowTitle          string                 // Window title displayed in windowed mode
	ShowBackground       bool                   // Whether to render the theme background
	WindowOptions        internal.WindowOptions // SDL window flags (borderless, resizable, etc.)
	PrimaryThemeColorHex uint32                 // Custom accent color (ignored on NextUI which uses system theme)
	IsCannoli            bool                   // Enable Cannoli CFW theming and input handling
	IsNextUI             bool                   // Enable NextUI CFW theming and power button handling
	ControllerConfigFile string                 // Path to custom controller mapping file
	LogFilename          string                 // Log file path (empty for stdout only)
}

// Init initializes the SDL subsystems, theming, and input handling.
// Must be called before any other gabagool functions.
// If INPUT_CAPTURE environment variable is set, runs the input logger wizard instead.
func Init(options Options) {
	if options.LogFilename != "" {
		internal.SetLogFilename(options.LogFilename)
	}

	if os.Getenv("NITRATES") != "" || os.Getenv("INPUT_CAPTURE") != "" {
		internal.SetInternalLogLevel(slog.LevelDebug)
	} else {
		internal.SetInternalLogLevel(slog.LevelError)
	}

	pbc := internal.PowerButtonConfig{}

	if options.IsNextUI {
		theme := nextui.InitNextUITheme()
		pbc = internal.PowerButtonConfig{
			ButtonCode:      116,
			DevicePath:      "/dev/input/event1",
			ShortPressMax:   2 * time.Second,
			CoolDownTime:    1 * time.Second,
			SuspendScript:   "/mnt/SDCARD/.system/tg5040/bin/suspend",
			ShutdownCommand: "/sbin/poweroff",
		}
		internal.SetTheme(theme)
	} else if options.IsCannoli {
		internal.SetTheme(cannoli.InitCannoliTheme("/mnt/SDCARD/System/fonts/Cannoli.ttf"))
	} else {
		internal.SetTheme(cannoli.InitCannoliTheme("/mnt/SDCARD/System/fonts/Cannoli.ttf")) // TODO fix this
	}

	if options.PrimaryThemeColorHex != 0 && !options.IsNextUI {
		theme := internal.GetTheme()
		theme.AccentColor = internal.HexToColor(options.PrimaryThemeColorHex)
		internal.SetTheme(theme)
	}

	internal.Init(options.WindowTitle, options.ShowBackground, options.WindowOptions, pbc)

	if os.Getenv("INPUT_CAPTURE") != "" {
		mapping := InputLogger()
		if mapping != nil {
			err := mapping.SaveToJSON("custom_input_mapping.json")
			if err != nil {
				internal.GetInternalLogger().Error("Failed to save custom input mapping", "error", err)
			}
		}
		os.Exit(0)
	}
}

// Close releases all SDL resources and shuts down the UI framework.
// Must be called before program exit to prevent resource leaks.
func Close() {
	internal.SDLCleanup()
}

// SetLogFilename sets the path for the application log file.
// Call before Init() to take effect during initialization.
func SetLogFilename(filename string) {
	internal.SetLogFilename(filename)
}

// GetLogger returns the application logger for structured logging.
func GetLogger() *slog.Logger {
	return internal.GetLogger()
}

// SetLogLevel sets the minimum log level for the application logger.
func SetLogLevel(level slog.Level) {
	internal.SetLogLevel(level)
}

// SetRawLogLevel parses and sets the log level from a string (e.g., "debug", "info", "error").
func SetRawLogLevel(level string) {
	internal.SetRawLogLevel(level)
}

// SetInputMappingBytes loads a custom input mapping from JSON bytes.
// Use this to override the default controller/keyboard bindings.
func SetInputMappingBytes(data []byte) {
	internal.SetInputMappingBytes(data)
}

// GetWindow returns the underlying SDL window wrapper for advanced use cases.
func GetWindow() *internal.Window {
	return internal.GetWindow()
}

// HideWindow hides the application window.
func HideWindow() {
	internal.GetWindow().Window.Hide()
}

// ShowWindow shows the application window.
func ShowWindow() {
	internal.GetWindow().Window.Show()
}

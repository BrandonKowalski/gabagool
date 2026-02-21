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
	"strings"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
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
	LogPath              string                 // Full path for log file including filename (creates parent directories)
	LogFilename          string                 // Deprecated: Use LogPath instead. Log filename within "logs" directory.
	FlipFaceButtons      bool                   // Use direct face button mapping (A=A, B=B) instead of Nintendo-style swap
}

// Init initializes the SDL subsystems, theming, and input handling.
// Must be called before any other gabagool functions.
// If INPUT_CAPTURE environment variable is set, runs the input logger wizard instead.
func Init(options Options) {
	if options.LogPath != "" {
		internal.SetLogPath(options.LogPath)
	} else if options.LogFilename != "" {
		internal.SetLogFilename(options.LogFilename)
	}

	if os.Getenv(constants.NitratesEnvVar) != "" || os.Getenv(constants.InputCaptureEnvVar) != "" {
		internal.SetInternalLogLevel(slog.LevelDebug)
	} else {
		internal.SetInternalLogLevel(slog.LevelError)
	}

	// Set face button flip preference before input mapping is loaded
	internal.SetFlipFaceButtons(options.FlipFaceButtons)

	pbc := internal.PowerButtonConfig{}

	if options.IsNextUI {
		theme := nextui.InitNextUITheme()

		// Detect power button input device path based on platform.
		// TG5050 uses /dev/input/event2, all others use /dev/input/event1.
		powerDevicePath := "/dev/input/event1"
		platformEnv := strings.ToUpper(os.Getenv("PLATFORM"))
		if strings.Contains(platformEnv, "TG5050") {
			powerDevicePath = "/dev/input/event2"
		}

		pbc = internal.PowerButtonConfig{
			ButtonCode:      116,
			DevicePath:      powerDevicePath,
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

	if os.Getenv(constants.InputCaptureEnvVar) != "" {
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

// SetLogPath sets the full path for the log file, including filename.
// Creates all necessary parent directories.
// Call before Init() to take effect during initialization.
func SetLogPath(path string) {
	internal.SetLogPath(path)
}

// SetLogFilename sets the filename for the log file within the "logs" directory.
// Deprecated: Use SetLogPath instead for full path support.
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

// SetFlipFaceButtons enables or disables direct face button mapping.
// When true, uses A=A, B=B, X=X, Y=Y instead of the default Nintendo-style swap.
// Can also be set via the FLIP_FACE_BUTTONS environment variable.
// Call before Init() to take effect.
func SetFlipFaceButtons(flip bool) {
	internal.SetFlipFaceButtons(flip)
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

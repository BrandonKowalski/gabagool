package gabagool

import (
	"log/slog"
	"os"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/platform/cannoli"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/platform/nextui"
)

type Options struct {
	WindowTitle          string
	ShowBackground       bool
	WindowOptions        internal.WindowOptions
	PrimaryThemeColorHex uint32
	IsCannoli            bool
	IsNextUI             bool
	ControllerConfigFile string
	LogFilename          string
}

// Init initializes SDL and the UI
// Must be called before any other UI functions!
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

// Close Tidies up SDL and the UI
// Must be called after all UI functions!
func Close() {
	internal.SDLCleanup()
}

func SetLogFilename(filename string) {
	internal.SetLogFilename(filename)
}

func GetLogger() *slog.Logger {
	return internal.GetLogger()
}

func SetLogLevel(level slog.Level) {
	internal.SetLogLevel(level)
}

func SetRawLogLevel(level string) {
	internal.SetRawLogLevel(level)
}

func SetInputMappingBytes(data []byte) {
	internal.SetInputMappingBytes(data)
}

func GetWindow() *internal.Window {
	return internal.GetWindow()
}

func HideWindow() {
	internal.GetWindow().Window.Hide()
}

func ShowWindow() {
	internal.GetWindow().Window.Show()
}

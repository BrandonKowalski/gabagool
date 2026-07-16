// Package nextui provides theming and configuration support for the NextUI custom firmware.
// NextUI is a CFW for the Trimui Smart Pro (tg5040) handheld gaming device.
// Theme colors are loaded from the system's nextval configuration utility.
package nextui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/veandco/go-sdl2/sdl"
)

// Default theme used when nextval configuration cannot be loaded.
var defaultTheme = internal.Theme{
	HighlightColor:       internal.HexToColor(0xFFFFFF),
	AccentColor:          internal.HexToColor(0x9B2257),
	ButtonLabelColor:     internal.HexToColor(0x1E2329),
	HintColor:            internal.HexToColor(0xFFFFFF),
	TextColor:            internal.HexToColor(0xFFFFFF),
	HighlightedTextColor: internal.HexToColor(0x000000),
	BackgroundColor:      internal.HexToColor(0x000000),
	FontPath:             "",
	BackgroundImagePath:  "/mnt/SDCARD/bg.png",
}

// InitNextUITheme loads the NextUI theme from the system's nextval configuration.
// Falls back to default theme if configuration cannot be loaded.
func InitNextUITheme() internal.Theme {
	var nv *NextVal
	var err error

	if constants.IsDevMode() {
		nv, err = InitStaticNextVal(os.Getenv(constants.NextvalPathEnvVar))
	} else {
		nv, err = loadNextVal()
	}

	if err != nil {
		// Enable NextUI mode with default font (RoundedMPlus1C)
		internal.SetNextUIMode(true, 1)
		return defaultTheme
	}

	// Set NextUI mode with font choice from nextval
	// Font 1 = RoundedMPlus1C, Font 2 = BPreplay
	internal.SetNextUIMode(true, nv.Font)

	theme := internal.Theme{
		HighlightColor:       parseHexColor(nv.Color1),
		AccentColor:          parseHexColor(nv.Color2),
		ButtonLabelColor:     parseHexColor(nv.Color3),
		TextColor:            parseHexColor(nv.Color4),
		HighlightedTextColor: parseHexColor(nv.Color5),
		HintColor:            parseHexColor(nv.Color6),
		BackgroundColor:      resolveBackgroundColor(nv.BGColor),
	}

	if constants.IsDevMode() {
		theme.BackgroundImagePath = os.Getenv(constants.BackgroundPathEnvVar)
	} else {
		theme.BackgroundImagePath = "/mnt/SDCARD/bg.png"
	}

	return theme
}

func InitStaticNextVal(filePath string) (*NextVal, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var nextval NextVal
	err = json.Unmarshal(data, &nextval)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON from file: %w", err)
	}

	return &nextval, nil
}

func loadNextVal() (*NextVal, error) {
	platformEnv := strings.ToLower(strings.TrimSpace(os.Getenv("PLATFORM")))
	if platformEnv == "" {
		platformEnv = "tg5040"
	}

	execPath := "/mnt/SDCARD/.system/" + platformEnv + "/bin/nextval.elf"

	cmd := exec.Command(execPath)
	output, err := cmd.Output()
	if err != nil {
		internal.GetInternalLogger().Error("Error executing command!", "error", err)
		return nil, err
	}

	jsonStr := strings.TrimSpace(string(output))

	var nextval NextVal
	err = json.Unmarshal([]byte(jsonStr), &nextval)
	if err != nil {
		internal.GetInternalLogger().Error("Error parsing JSON", "error", err)
		return nil, err
	}

	return &nextval, nil
}

// errorColor is returned for a non-background color that cannot be parsed. It
// is deliberately conspicuous so a misconfigured theme is obvious on screen
// while still leaving the rest of the UI usable.
var errorColor = sdl.Color{R: 255, G: 0, B: 0, A: 255}

// parseHexColor parses a color string from nextval, returning errorColor when
// it cannot be parsed. Use resolveBackgroundColor for the background, whose
// error color would fill the whole screen.
func parseHexColor(hexStr string) sdl.Color {
	if color, ok := tryParseHexColor(hexStr); ok {
		return color
	}
	return errorColor
}

// resolveBackgroundColor parses the nextval background color, falling back to
// the default theme's background (and logging a warning) when it is missing or
// unparseable. Unlike other theme colors the background fills the entire
// screen, so the errorColor sentinel would render an unusable solid-red UI;
// the default is a sane, legible fallback instead. A missing value is expected
// on NextUI builds predating the color7 background setting.
func resolveBackgroundColor(hexStr string) sdl.Color {
	if color, ok := tryParseHexColor(hexStr); ok {
		return color
	}
	internal.GetInternalLogger().Warn(
		"NextUI background color missing or unparseable; using default",
		"value", hexStr,
	)
	return defaultTheme.BackgroundColor
}

// tryParseHexColor parses a hex color string from nextval. NextUI emits colors
// as a quoted "0x%08X" RRGGBBAA string; builds predating alpha support emit
// 6-digit RRGGBB. The format is decided by digit count rather than value, so we
// mirror that here. Returns ok=false for anything it cannot parse.
func tryParseHexColor(hexStr string) (sdl.Color, bool) {
	hexStr = strings.TrimSpace(hexStr)
	hexStr = strings.TrimPrefix(hexStr, "#")
	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimPrefix(hexStr, "0X")

	hex, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return sdl.Color{}, false
	}

	switch len(hexStr) {
	case 6:
		// "RRGGBB" - legacy opaque format.
		return internal.HexToColor(uint32(hex)), true
	case 8:
		// "RRGGBBAA" - NextUI's current format.
		return sdl.Color{
			R: uint8((hex >> 24) & 0xFF),
			G: uint8((hex >> 16) & 0xFF),
			B: uint8((hex >> 8) & 0xFF),
			A: uint8(hex & 0xFF),
		}, true
	default:
		return sdl.Color{}, false
	}
}

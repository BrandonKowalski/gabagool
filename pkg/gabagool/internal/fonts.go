package internal

import (
	"os"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// NextUI font configuration
var (
	isNextUIMode   bool
	nextUIFontType int // 1 = RoundedMPlus1C (default), 2 = BPreplay
)

// SetNextUIMode configures the font system for NextUI mode
func SetNextUIMode(enabled bool, fontType int) {
	isNextUIMode = enabled
	nextUIFontType = fontType
}

// getNextUIEmbeddedFont returns the appropriate NextUI font based on configuration
func getNextUIEmbeddedFont() []byte {
	if nextUIFontType == 2 {
		return nextUIFont2
	}
	// Default to font1 (RoundedMPlus1C)
	return nextUIFont1
}

type FontSizes struct {
	XLarge int `json:"xlarge" yaml:"xlarge"`
	Large  int `json:"large" yaml:"large"`
	Medium int `json:"medium" yaml:"medium"`
	Small  int `json:"small" yaml:"small"`
	Tiny   int `json:"tiny" yaml:"tiny"`
	Micro  int `json:"micro" yaml:"micro"`
}

var DefaultFontSizes = FontSizes{
	XLarge: 60,
	Large:  50,
	Medium: 44,
	Small:  34,
	Tiny:   24,
	Micro:  18,
}

var Fonts fontsManager

type fontsManager struct {
	ExtraLargeFont *ttf.Font
	LargeFont      *ttf.Font
	MediumFont     *ttf.Font
	SmallFont      *ttf.Font
	TinyFont       *ttf.Font
	MicroFont      *ttf.Font
}

func CalculateFontSizeForResolution(baseSize int, screenWidth int32) int {
	const referenceWidth int32 = 1024
	scaleFactor := float32(screenWidth) / float32(referenceWidth)

	// Apply damping for larger screens to reduce scaling growth
	if screenWidth > referenceWidth {
		scaleFactor = 1.0 + (scaleFactor-1.0)*0.75 // 75% of the growth above 1x
	}

	return int(float32(baseSize) * scaleFactor)
}

// GetScaleFactor returns the scale factor based on current screen width
func GetScaleFactor() float32 {
	const referenceWidth int32 = 1024
	screenWidth := GetWindow().GetWidth()

	scaleFactor := float32(screenWidth) / float32(referenceWidth)

	// Apply damping for larger screens
	if screenWidth > referenceWidth {
		scaleFactor = 1.0 + (scaleFactor-1.0)*0.75
	}

	return scaleFactor
}

func initFonts(sizes FontSizes) {
	screenWidth := GetWindow().GetWidth()
	fallback := os.Getenv(constants.FallbackFontEnvVar)

	// Calculate all sizes
	calcSize := func(base int) int {
		return CalculateFontSizeForResolution(base, screenWidth)
	}

	Fonts = fontsManager{
		ExtraLargeFont: loadFont(fallback, calcSize(sizes.XLarge)),
		LargeFont:      loadFont(fallback, calcSize(sizes.Large)),
		MediumFont:     loadFont(fallback, calcSize(sizes.Medium)),
		SmallFont:      loadFont(fallback, calcSize(sizes.Small)),
		TinyFont:       loadFont(fallback, calcSize(sizes.Tiny)),
		MicroFont:      loadFont(fallback, calcSize(sizes.Micro)),
	}
}

func loadFont(fallback string, size int) *ttf.Font {
	var font *ttf.Font
	var err error

	if fallback != "" {
		font, err = ttf.OpenFont(fallback, size)
		if err == nil {
			return font
		}
		GetInternalLogger().Debug("Failed to load fallback font, using embedded font", "fallback", fallback, "error", err)
	}

	// Use NextUI embedded font if in NextUI mode, otherwise default
	var embeddedFont []byte
	var fontType string
	if isNextUIMode {
		embeddedFont = getNextUIEmbeddedFont()
		fontType = "NextUI"
	} else {
		embeddedFont = defaultFont
		fontType = "default"
	}

	if embeddedFont == nil {
		GetInternalLogger().Error("No embedded font available and FALLBACK_FONT not set",
			"fontType", fontType,
			"hint", "Set FALLBACK_FONT environment variable or rebuild without the corresponding build tag")
		os.Exit(1)
	}

	return loadEmbeddedFont(embeddedFont, size)
}

func loadEmbeddedFont(bytes []byte, size int) *ttf.Font {
	rw, err := sdl.RWFromMem(bytes)
	if err != nil {
		GetInternalLogger().Error("Failed to create RW from embedded font", "size", size, "error", err)
		os.Exit(1)
	}

	font, err := ttf.OpenFontRW(rw, 1, size)
	if err != nil {
		GetInternalLogger().Error("Failed to load embedded font", "size", size, "error", err)
		os.Exit(1)
	}

	return font
}

func closeFonts() {
	Fonts.LargeFont.Close()
	Fonts.MediumFont.Close()
	Fonts.SmallFont.Close()
	Fonts.TinyFont.Close()
	Fonts.MicroFont.Close()
}

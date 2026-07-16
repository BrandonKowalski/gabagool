package nextui

import (
	"testing"

	"github.com/veandco/go-sdl2/sdl"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sdl.Color
	}{
		{
			name:  "rgba 8-digit",
			input: "1E2329FF",
			want:  sdl.Color{R: 0x1E, G: 0x23, B: 0x29, A: 0xFF},
		},
		{
			name:  "rgba preserves alpha byte",
			input: "1E232980",
			want:  sdl.Color{R: 0x1E, G: 0x23, B: 0x29, A: 0x80},
		},
		{
			name:  "rgba zero alpha",
			input: "12345600",
			want:  sdl.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x00},
		},
		{
			name:  "rgb 6-digit is opaque",
			input: "1E2329",
			want:  sdl.Color{R: 0x1E, G: 0x23, B: 0x29, A: 255},
		},
		{
			name:  "rgb white",
			input: "FFFFFF",
			want:  sdl.Color{R: 0xFF, G: 0xFF, B: 0xFF, A: 255},
		},
		{
			name:  "0x prefix stripped",
			input: "0x9B2257",
			want:  sdl.Color{R: 0x9B, G: 0x22, B: 0x57, A: 255},
		},
		{
			name:  "uppercase 0X prefix stripped",
			input: "0XFFFFFFFF",
			want:  sdl.Color{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		},
		{
			name:  "hash prefix stripped",
			input: "#FF0000",
			want:  sdl.Color{R: 0xFF, G: 0x00, B: 0x00, A: 255},
		},
		{
			name:  "surrounding whitespace trimmed",
			input: "  FFFFFF  ",
			want:  sdl.Color{R: 0xFF, G: 0xFF, B: 0xFF, A: 255},
		},
		{
			name:  "empty string falls back to error color",
			input: "",
			want:  errorColor,
		},
		{
			name:  "non-hex falls back to error color",
			input: "nothex",
			want:  errorColor,
		},
		{
			name:  "odd length falls back to error color",
			input: "1E2329F",
			want:  errorColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHexColor(tt.input)
			if got != tt.want {
				t.Errorf("parseHexColor(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveBackgroundColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sdl.Color
	}{
		{
			name:  "valid rgba is parsed",
			input: "0x102030FF",
			want:  sdl.Color{R: 0x10, G: 0x20, B: 0x30, A: 0xFF},
		},
		{
			name:  "valid rgb is parsed",
			input: "102030",
			want:  sdl.Color{R: 0x10, G: 0x20, B: 0x30, A: 255},
		},
		{
			// Builds predating the color7 setting emit no background key,
			// so the field arrives empty and must not render solid red.
			name:  "empty falls back to default theme, not error color",
			input: "",
			want:  defaultTheme.BackgroundColor,
		},
		{
			name:  "unparseable falls back to default theme",
			input: "nothex",
			want:  defaultTheme.BackgroundColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveBackgroundColor(tt.input)
			if got != tt.want {
				t.Errorf("resolveBackgroundColor(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
			if got == errorColor && tt.want != errorColor {
				t.Errorf("resolveBackgroundColor(%q) returned the error (red) color", tt.input)
			}
		})
	}
}

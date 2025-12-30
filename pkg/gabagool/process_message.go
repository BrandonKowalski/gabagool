package gabagool

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/internal"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"go.uber.org/atomic"
)

type ProcessMessageOptions struct {
	Image               string // Deprecated: Use ImageBytes instead. File path to image (PNG, JPEG, or SVG)
	ImageBytes          []byte // Image data loaded from embedded resources (supports PNG, JPEG, and SVG)
	ImageWidth          int32  // Desired width for rendering (required for SVG, optional for raster images)
	ImageHeight         int32  // Desired height for rendering (required for SVG, optional for raster images)
	ShowThemeBackground bool
	ShowProgressBar     bool
	Progress            *atomic.Float64
}

type processMessage struct {
	window          *internal.Window
	showBG          bool
	message         string
	isProcessing    bool
	completeTime    time.Time
	imageTexture    *sdl.Texture
	imageWidth      int32
	imageHeight     int32
	showProgressBar bool
	progress        *atomic.Float64
}

// ProcessMessage displays a message while executing a function asynchronously.
// The function is generic and returns the typed result of the function.
//
// Supports displaying images in PNG, JPEG, and SVG formats via ImageBytes or Image (legacy).
// For SVG images, ImageWidth and ImageHeight should be specified for optimal rendering quality.
func ProcessMessage[T any](message string, options ProcessMessageOptions, fn func() (T, error)) (T, error) {
	processor := &processMessage{
		window:          internal.GetWindow(),
		showBG:          options.ShowThemeBackground,
		imageWidth:      options.ImageWidth,
		imageHeight:     options.ImageHeight,
		message:         message,
		isProcessing:    true,
		showProgressBar: options.ShowProgressBar,
		progress:        options.Progress,
	}

	// Load image from bytes (preferred) or from file path (legacy)
	if len(options.ImageBytes) > 0 {
		texture, err := loadImageTexture(processor.window.Renderer, options.ImageBytes, options.ImageWidth, options.ImageHeight)
		if err == nil {
			processor.imageTexture = texture
		}
	} else if options.Image != "" {
		// Legacy file path support
		if strings.HasSuffix(strings.ToLower(options.Image), ".svg") {
			// Read SVG file
			svgData, err := os.ReadFile(options.Image)
			if err == nil {
				texture, err := loadImageTexture(processor.window.Renderer, svgData, options.ImageWidth, options.ImageHeight)
				if err == nil {
					processor.imageTexture = texture
				}
			}
		} else {
			// Load raster image
			img.Init(img.INIT_PNG | img.INIT_JPG)
			texture, err := img.LoadTexture(processor.window.Renderer, options.Image)
			if err == nil {
				processor.imageTexture = texture
			}
		}
	}

	var result T
	var fnError error

	window := internal.GetWindow()
	renderer := window.Renderer

	processor.render(renderer)
	renderer.Present()

	resultChan := make(chan struct {
		result T
		err    error
	}, 1)

	go func() {
		res, err := fn()
		resultChan <- struct {
			result T
			err    error
		}{result: res, err: err}
	}()

	running := true
	functionComplete := false
	var quitErr error

	for running {
		if event := sdl.WaitEventTimeout(16); event != nil {
			if _, ok := event.(*sdl.QuitEvent); ok {
				running = false
				quitErr = sdl.GetError()
			}
		}

		if !functionComplete {
			select {
			case processResult := <-resultChan:
				result = processResult.result
				fnError = processResult.err
				functionComplete = true
				processor.isProcessing = false
				processor.completeTime = time.Now()
			default:
			}
		} else {
			if time.Since(processor.completeTime) > 350*time.Millisecond {
				running = false
			}
		}

		processor.render(renderer)
		renderer.Present()
	}

	if processor.imageTexture != nil {
		processor.imageTexture.Destroy()
	}

	// Prioritize function error over quit error
	if fnError != nil {
		return result, fnError
	}

	if quitErr != nil {
		return result, quitErr
	}

	return result, nil
}

func (p *processMessage) render(renderer *sdl.Renderer) {

	if p.showBG && internal.GetWindow().Background != nil {
		internal.GetWindow().RenderBackground()
	} else {
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
	}

	if p.imageTexture != nil {
		width := p.imageWidth
		height := p.imageHeight

		if width == 0 {
			width = p.window.GetWidth()
		}

		if height == 0 {
			height = p.window.GetHeight()
		}

		x := (p.window.GetWidth() - width) / 2
		y := (p.window.GetHeight() - height) / 2

		renderer.Copy(p.imageTexture, nil, &sdl.Rect{X: x, Y: y, W: width, H: height})
	}

	font := internal.Fonts.SmallFont

	maxWidth := p.window.GetWidth() * 3 / 4

	messageY := p.window.GetHeight() / 2
	spacing := int32(5)
	if p.showProgressBar {
		barHeight := int32(40)
		totalHeight := (int32(font.Height()) * 2) + spacing + barHeight
		messageY = (p.window.GetHeight() - totalHeight) / 2
	}

	internal.RenderMultilineText(renderer, p.message, font, maxWidth, p.window.GetWidth()/2, messageY, sdl.Color{R: 255, G: 255, B: 255, A: 255})

	if p.showProgressBar {
		p.renderProgressBar(renderer, messageY, spacing)
	}
}

func (p *processMessage) renderProgressBar(renderer *sdl.Renderer, messageY, spacing int32) {
	windowWidth := p.window.GetWidth()

	barWidth := windowWidth * 3 / 4
	if barWidth > 900 {
		barWidth = 900
	}
	barHeight := int32(40)
	barX := (windowWidth - barWidth) / 2
	// Position progress bar relative to the message with consistent spacing
	barY := messageY + int32(internal.Fonts.SmallFont.Height()) + spacing

	progressBarBg := sdl.Rect{
		X: barX,
		Y: barY,
		W: barWidth,
		H: barHeight,
	}

	progressWidth := int32(float64(barWidth) * p.progress.Load())

	// Use smooth progress bar with anti-aliased rounded edges
	internal.DrawSmoothProgressBar(
		renderer,
		&progressBarBg,
		progressWidth,
		sdl.Color{R: 50, G: 50, B: 50, A: 255},
		sdl.Color{R: 100, G: 150, B: 255, A: 255},
	)

	percentText := fmt.Sprintf("%.0f%%", p.progress.Load()*100)

	percentSurface, err := internal.Fonts.SmallFont.RenderUTF8Blended(percentText, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err == nil {
		percentTexture, err := renderer.CreateTextureFromSurface(percentSurface)
		if err == nil {
			textX := barX + (barWidth-percentSurface.W)/2
			textY := barY + (barHeight-percentSurface.H)/2

			percentRect := &sdl.Rect{
				X: textX,
				Y: textY,
				W: percentSurface.W,
				H: percentSurface.H,
			}
			renderer.Copy(percentTexture, nil, percentRect)
			percentTexture.Destroy()
		}
		percentSurface.Free()
	}
}

// isSVG checks if the data is SVG format
func isSVG(data []byte) bool {
	// Check for SVG header
	return bytes.Contains(data[:min(len(data), 512)], []byte("<svg")) ||
		bytes.Contains(data[:min(len(data), 512)], []byte("<?xml"))
}

// loadImageTexture loads an image (PNG, JPEG, or SVG) from bytes and creates an SDL texture
func loadImageTexture(renderer *sdl.Renderer, imageData []byte, width, height int32) (*sdl.Texture, error) {
	if isSVG(imageData) {
		return loadSVGTexture(renderer, imageData, width, height)
	}
	return loadRasterTexture(renderer, imageData)
}

// loadRasterTexture loads a raster image (PNG, JPEG, etc.) from bytes
func loadRasterTexture(renderer *sdl.Renderer, imageData []byte) (*sdl.Texture, error) {
	img.Init(img.INIT_PNG | img.INIT_JPG)
	rw, err := sdl.RWFromMem(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to create RWops from image data: %w", err)
	}
	texture, err := img.LoadTextureRW(renderer, rw, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load texture from image data: %w", err)
	}
	return texture, nil
}

// loadSVGTexture rasterizes an SVG and creates an SDL texture
func loadSVGTexture(renderer *sdl.Renderer, svgData []byte, width, height int32) (*sdl.Texture, error) {
	// Parse SVG
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Determine dimensions
	svgWidth := int(icon.ViewBox.W)
	svgHeight := int(icon.ViewBox.H)

	// Use provided dimensions, or default to SVG viewBox
	if width == 0 || height == 0 {
		width = int32(svgWidth)
		height = int32(svgHeight)
	}

	// Create image to render SVG into
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	// Create rasterizer
	scanner := rasterx.NewScannerGV(int(width), int(height), img, img.Bounds())
	raster := rasterx.NewDasher(int(width), int(height), scanner)

	// Set the icon to the target size
	icon.SetTarget(0, 0, float64(width), float64(height))

	// Draw SVG
	icon.Draw(raster, 1.0)

	// Convert image.RGBA to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode SVG as PNG: %w", err)
	}

	// Load the PNG as a texture
	return loadRasterTexture(renderer, buf.Bytes())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

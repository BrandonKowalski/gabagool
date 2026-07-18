package internal

import (
	"strings"
	"time"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type TextScrollData struct {
	NeedsScrolling      bool
	ScrollOffset        int32
	TextWidth           int32
	ContainerWidth      int32
	Direction           int
	LastDirectionChange *time.Time
}

// wrapTextToLines splits text into rendered lines, wrapping on word
// boundaries so no line exceeds maxWidth. Explicit newlines are preserved as
// blank lines. It is the shared basis for RenderMultilineText and
// MultilineTextHeight so measurement and rendering always agree.
func wrapTextToLines(text string, font *ttf.Font, maxWidth int32, color sdl.Color) []string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	paragraphs := strings.Split(normalized, "\n")
	var lines []string

	for _, paragraph := range paragraphs {

		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			continue
		}

		currentLine := words[0]

		for _, word := range words[1:] {

			testLine := currentLine + " " + word
			testSurface, err := font.RenderUTF8Blended(testLine, color)
			if err != nil {
				continue
			}

			fits := testSurface.W <= maxWidth
			testSurface.Free()

			if fits {
				currentLine = testLine
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}

		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}

// StackedLayout positions an image above a block of text as a single group
// centered vertically within a window of height winH. gap is the space between
// the image's bottom and the text's top. It returns the image's top Y and the
// text block's vertical center (the startY that RenderMultilineText's
// center alignment expects). If the group is taller than the window the image
// is pinned to the top so it stays on-screen.
func StackedLayout(winH, imgH, textH, gap int32) (imgY, textCenterY int32) {
	blockH := imgH + gap + textH
	top := (winH - blockH) / 2
	if top < 0 {
		top = 0
	}
	return top, top + imgH + gap + textH/2
}

// MultilineTextHeight returns the pixel height RenderMultilineText uses to
// vertically position text with the given font and maxWidth. Callers that
// stack text beneath other content use it to lay the two out without overlap.
func MultilineTextHeight(text string, font *ttf.Font, maxWidth int32) int32 {
	lines := wrapTextToLines(text, font, maxWidth, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if len(lines) == 0 {
		return 0
	}
	return int32(font.Height()) * int32(len(lines))
}

func RenderMultilineText(renderer *sdl.Renderer, text string, font *ttf.Font, maxWidth int32, x, startY int32, color sdl.Color, alignment ...constants.TextAlign) {

	textAlign := constants.TextAlignCenter
	if len(alignment) > 0 {
		textAlign = alignment[0]
	}

	lines := wrapTextToLines(text, font, maxWidth, color)

	if len(lines) == 0 {
		return
	}

	lineHeight := int32(font.Height())
	totalHeight := lineHeight * int32(len(lines))

	var currentY int32
	if textAlign == constants.TextAlignCenter {

		currentY = startY - totalHeight/2
	} else {

		currentY = startY
	}

	for _, line := range lines {

		if line == "" {
			currentY += lineHeight + 5
			continue
		}

		surface, err := font.RenderUTF8Blended(line, color)
		if err != nil {
			continue
		}

		texture, err := renderer.CreateTextureFromSurface(surface)
		if err == nil {
			rect := &sdl.Rect{
				Y: currentY,
				W: surface.W,
				H: surface.H,
			}

			if textAlign == constants.TextAlignCenter {
				rect.X = x - surface.W/2
			} else {
				rect.X = x
			}

			renderer.Copy(texture, nil, rect)
			texture.Destroy()
		}

		surface.Free()
		currentY += lineHeight + 5
	}
}

func RenderMultilineTextWithCache(
	renderer *sdl.Renderer,
	text string,
	font *ttf.Font,
	maxWidth int32,
	x, y int32,
	color sdl.Color,
	align constants.TextAlign,
	cache *TextureCache) {

	if text == "" {
		return
	}

	_, fontHeight, err := font.SizeUTF8("Aj")
	if err != nil {
		fontHeight = 20
	}

	lineSpacing := int32(float32(fontHeight) * 0.3)
	lineY := y

	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		if line == "" {
			lineY += int32(fontHeight) + lineSpacing
			continue
		}

		remainingText := line
		for len(remainingText) > 0 {
			width, _, err := font.SizeUTF8(remainingText)
			if err != nil || int32(width) <= maxWidth {
				cacheKey := "line_" + remainingText + "_" + string(color.R) + string(color.G) + string(color.B)
				lineTexture := cache.Get(cacheKey)

				if lineTexture == nil {
					lineSurface, err := font.RenderUTF8Blended(remainingText, color)
					if err == nil {
						lineTexture, err = renderer.CreateTextureFromSurface(lineSurface)
						lineSurface.Free()

						if err == nil {
							cache.Set(cacheKey, lineTexture)
						}
					}
				}

				if lineTexture != nil {
					_, _, lineW, lineH, _ := lineTexture.Query()

					var lineX int32
					switch align {
					case constants.TextAlignCenter:
						lineX = x + (maxWidth-lineW)/2
					case constants.TextAlignRight:
						lineX = x + maxWidth - lineW
					default:
						lineX = x
					}

					lineRect := &sdl.Rect{
						X: lineX,
						Y: lineY,
						W: lineW,
						H: lineH,
					}

					renderer.Copy(lineTexture, nil, lineRect)
				}

				lineY += int32(fontHeight) + lineSpacing
				break
			}

			charsPerLine := int(float32(len(remainingText)) * float32(maxWidth) / float32(width))
			if charsPerLine <= 0 {
				charsPerLine = 1
			}

			if charsPerLine < len(remainingText) {
				for i := charsPerLine; i > 0; i-- {
					if i < len(remainingText) && remainingText[i] == ' ' {
						charsPerLine = i
						break
					}
				}
			}

			lineText := remainingText[:min(charsPerLine, len(remainingText))]
			cacheKey := "line_" + lineText + "_" + string(color.R) + string(color.G) + string(color.B)
			lineTexture := cache.Get(cacheKey)

			if lineTexture == nil {
				lineSurface, err := font.RenderUTF8Blended(lineText, color)
				if err == nil {
					lineTexture, err = renderer.CreateTextureFromSurface(lineSurface)
					lineSurface.Free()

					if err == nil {
						cache.Set(cacheKey, lineTexture)
					}
				}
			}

			if lineTexture != nil {
				_, _, lineW, lineH, _ := lineTexture.Query()

				var lineX int32
				switch align {
				case constants.TextAlignCenter:
					lineX = x + (maxWidth-lineW)/2
				case constants.TextAlignRight:
					lineX = x + maxWidth - lineW
				default:
					lineX = x
				}

				lineRect := &sdl.Rect{
					X: lineX,
					Y: lineY,
					W: lineW,
					H: lineH,
				}

				renderer.Copy(lineTexture, nil, lineRect)
			}

			lineY += int32(fontHeight) + lineSpacing

			if charsPerLine >= len(remainingText) {
				break
			}

			remainingText = remainingText[charsPerLine:]
			remainingText = strings.TrimLeft(remainingText, " ")
		}
	}
}

func DrawRoundedRect(renderer *sdl.Renderer, rect *sdl.Rect, radius int32, color sdl.Color) {
	if radius <= 0 {
		renderer.SetDrawColor(color.R, color.G, color.B, color.A)
		renderer.FillRect(rect)
		return
	}

	// Clamp radius to half the smaller dimension
	maxRadius := rect.W / 2
	if rect.H/2 < maxRadius {
		maxRadius = rect.H / 2
	}
	if radius > maxRadius {
		radius = maxRadius
	}

	if color.A == 255 {
		drawRoundedRectPrimitives(renderer, rect, radius, color)
		return
	}

	drawTranslucentShape(renderer, rect, color, func(localRect *sdl.Rect, opaqueColor sdl.Color) {
		drawRoundedRectPrimitives(renderer, localRect, radius, opaqueColor)
	})
}

func drawRoundedRectPrimitives(renderer *sdl.Renderer, rect *sdl.Rect, radius int32, color sdl.Color) {
	x1 := rect.X
	y1 := rect.Y
	x2 := rect.X + rect.W - 1
	y2 := rect.Y + rect.H - 1

	// Draw filled rounded rectangle
	gfx.RoundedBoxColor(renderer, x1, y1, x2, y2, radius, color)

	// Add anti-aliased outline for smoother edges
	gfx.RoundedRectangleColor(renderer, x1, y1, x2, y2, radius, color)
}

// DrawFilledCircle draws a filled circle with an anti-aliased edge.
func DrawFilledCircle(renderer *sdl.Renderer, centerX, centerY, radius int32, color sdl.Color) {
	// A circle of the given radius spans radius*2+1 pixels (inclusive of both
	// the center row/column), not radius*2 - undersizing the bounding rect
	// clips the AA edge on the right/bottom.
	diameter := radius*2 + 1
	rect := &sdl.Rect{X: centerX - radius, Y: centerY - radius, W: diameter, H: diameter}

	if color.A == 255 {
		drawFilledCirclePrimitives(renderer, centerX, centerY, radius, color)
		return
	}

	// Derive the center from localRect so both paths are correct: on the
	// scratch texture localRect is at the origin (center radius,radius), and on
	// the fallback path (drawTranslucentShape draws straight to destRect when
	// the scratch texture is unavailable) localRect is destRect and the circle
	// lands back at centerX,centerY instead of the screen corner.
	drawTranslucentShape(renderer, rect, color, func(localRect *sdl.Rect, opaqueColor sdl.Color) {
		drawFilledCirclePrimitives(renderer, localRect.X+radius, localRect.Y+radius, radius, opaqueColor)
	})
}

func drawFilledCirclePrimitives(renderer *sdl.Renderer, centerX, centerY, radius int32, color sdl.Color) {
	gfx.FilledCircleColor(renderer, centerX, centerY, radius, color)

	// Add anti-aliased outline(s) for smoother edges
	gfx.AACircleColor(renderer, centerX, centerY, radius, color)
	if radius > 2 {
		gfx.AACircleColor(renderer, centerX, centerY, radius-1, color)
	}
}

// drawTranslucentShape works around a class of SDL2_gfx artifacts where a
// shape is rasterized as several overlapping filled/outline primitives (e.g.
// a rounded box plus its outline, or a filled circle plus one or two AA
// edges). SDL2_gfx's internal renderColor() helper forces
// SDL_BLENDMODE_BLEND onto the renderer whenever alpha < 255 (only alpha ==
// 255 gets BLENDMODE_NONE), so wherever those primitives overlap - a pill's
// middle scanline, a circle's antialiased rim - the translucent color gets
// composited twice and visibly brightens there, regardless of blend mode set
// beforehand. Work around it by having `draw` render the shape fully opaque
// (safe: opaque draws are idempotent under SDL2_gfx's own NONE-mode fast
// path) onto a transparent scratch texture, then apply the real translucency
// once via the texture's alpha modulation instead of the fill color.
func drawTranslucentShape(renderer *sdl.Renderer, destRect *sdl.Rect, color sdl.Color, draw func(localRect *sdl.Rect, opaqueColor sdl.Color)) {
	tex, err := getScratchTexture(renderer, destRect.W, destRect.H)
	if err != nil {
		draw(destRect, color)
		return
	}

	prevTarget := renderer.GetRenderTarget()
	var prevBlendMode sdl.BlendMode
	renderer.GetDrawBlendMode(&prevBlendMode)

	if err := renderer.SetRenderTarget(tex); err != nil {
		// The target is left unchanged on failure, so it is safe to fall back
		// to drawing straight to the real target rather than blitting garbage.
		draw(destRect, color)
		return
	}
	// Clear() honors the current draw blend mode; with BLENDMODE_BLEND left
	// over from an earlier draw, clearing would blend against (not overwrite)
	// whatever this cached texture held from its previous use and leak it
	// through, so force NONE first. Clear to the shape's own chroma at alpha 0
	// rather than transparent black: the AA rim pixels are rasterized with
	// coverage-weighted alpha and so still pass through BLENDMODE_BLEND when
	// copied out. Blending them against a matching-chroma, zero-alpha
	// background reconstructs the correct non-premultiplied color
	// (C*w + C*(1-w) = C); clearing to black instead would leave a dark fringe
	// (C*w against 0) around the shape.
	renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
	renderer.SetDrawColor(color.R, color.G, color.B, 0)
	renderer.Clear()

	localRect := sdl.Rect{X: 0, Y: 0, W: destRect.W, H: destRect.H}
	opaqueColor := sdl.Color{R: color.R, G: color.G, B: color.B, A: 255}
	draw(&localRect, opaqueColor)

	renderer.SetRenderTarget(prevTarget)
	renderer.SetDrawBlendMode(prevBlendMode)

	tex.SetBlendMode(sdl.BLENDMODE_BLEND)
	tex.SetAlphaMod(color.A)
	renderer.Copy(tex, &localRect, destRect)
}

var (
	translucentScratchTexture *sdl.Texture
	translucentScratchW       int32
	translucentScratchH       int32
)

func getScratchTexture(renderer *sdl.Renderer, w, h int32) (*sdl.Texture, error) {
	if translucentScratchTexture != nil && translucentScratchW >= w && translucentScratchH >= h {
		return translucentScratchTexture, nil
	}

	// Grow to cover both the cached and requested dimensions on each axis so
	// alternating aspect ratios (a tall pill then a wide one) don't destroy and
	// recreate the texture every frame.
	if translucentScratchW > w {
		w = translucentScratchW
	}
	if translucentScratchH > h {
		h = translucentScratchH
	}

	if translucentScratchTexture != nil {
		translucentScratchTexture.Destroy()
	}

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, w, h)
	if err != nil {
		translucentScratchTexture = nil
		translucentScratchW = 0
		translucentScratchH = 0
		return nil, err
	}

	translucentScratchTexture = tex
	translucentScratchW = w
	translucentScratchH = h
	return tex, nil
}

// destroyScratchTexture releases the cached scratch texture and resets the
// cache. It must run before the renderer that owns the texture is destroyed:
// the cache is a package global that outlives any single window, so a later
// reuse would otherwise hand a freed texture back to SDL.
func destroyScratchTexture() {
	if translucentScratchTexture != nil {
		translucentScratchTexture.Destroy()
		translucentScratchTexture = nil
		translucentScratchW = 0
		translucentScratchH = 0
	}
}

// DrawSmoothScrollbar renders a simple square scrollbar
func DrawSmoothScrollbar(renderer *sdl.Renderer, x, y, width, height int32, color sdl.Color) {
	if width <= 0 || height <= 0 {
		return
	}

	renderer.SetDrawColor(color.R, color.G, color.B, color.A)
	renderer.FillRect(&sdl.Rect{X: x, Y: y, W: width, H: height})
}

// DrawSmoothProgressBar renders a rectangular progress bar
func DrawSmoothProgressBar(renderer *sdl.Renderer, bgRect *sdl.Rect, fillWidth int32, bgColor, fillColor sdl.Color) {
	if bgRect == nil {
		return
	}

	// Draw background rectangle
	renderer.SetDrawColor(bgColor.R, bgColor.G, bgColor.B, bgColor.A)
	renderer.FillRect(bgRect)

	// Draw fill rectangle if there's progress
	if fillWidth > 0 && fillWidth <= bgRect.W {
		fillRect := &sdl.Rect{
			X: bgRect.X,
			Y: bgRect.Y,
			W: Min32(fillWidth, bgRect.W),
			H: bgRect.H,
		}
		renderer.SetDrawColor(fillColor.R, fillColor.G, fillColor.B, fillColor.A)
		renderer.FillRect(fillRect)
	}
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func Min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func Max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func HexToColor(hex uint32) sdl.Color {
	r := uint8((hex >> 16) & 0xFF)
	g := uint8((hex >> 8) & 0xFF)
	b := uint8(hex & 0xFF)

	return sdl.Color{R: r, G: g, B: b, A: 255}
}

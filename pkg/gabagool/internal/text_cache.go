package internal

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// Rendered text is cached for the lifetime of the renderer.
//
// Every screen used to build a fresh surface and texture for each piece of
// text on every frame and destroy them again: list rows, the title, the
// footer's button hints, the status bar clock. For a full list that is roughly
// 500 create/destroy cycles a second at 60fps.
//
// On TrimUI hardware the Mali driver does not fully reclaim that churn, so an
// idle screen grows resident memory steadily -- measured at ~150 kB/s in
// isolation. A pak left open long enough is OOM-killed. Rendering each
// distinct string once and reusing the texture removes the growth.
//
// 128 entries comfortably covers a full screen of rows in both focused and
// unfocused colours, the title, the footer and the clock, while staying
// bounded as text changes.
var textCache = NewTextureCacheWithSize(128)

// textCacheKey identifies one rendered string.
//
// The font pointer distinguishes sizes and faces; the colour distinguishes
// focused from unfocused rows and themed variants of the same string. Text goes
// last so that a string containing the separator cannot forge a different
// font or colour prefix.
func textCacheKey(font *ttf.Font, text string, color sdl.Color) string {
	return fmt.Sprintf("%p|%02x%02x%02x%02x|%s", font, color.R, color.G, color.B, color.A, text)
}

// RenderTextCached renders text and returns a texture along with its size.
//
// The cache owns the texture: callers must draw with it and must NOT destroy
// it. Access is not synchronised, which matches SDL's single-threaded
// rendering.
func RenderTextCached(renderer *sdl.Renderer, font *ttf.Font, text string, color sdl.Color) (*sdl.Texture, int32, int32) {
	if font == nil || text == "" {
		return nil, 0, 0
	}

	key := textCacheKey(font, text, color)

	if texture := textCache.Get(key); texture != nil {
		if _, _, w, h, err := texture.Query(); err == nil {
			return texture, w, h
		}
		// Unqueryable means the texture is no longer usable, most likely
		// because the renderer it belonged to is gone. Fall through and
		// rebuild rather than handing back something broken.
	}

	surface, err := font.RenderUTF8Blended(text, color)
	if err != nil || surface == nil {
		return nil, 0, 0
	}
	defer surface.Free()

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil || texture == nil {
		return nil, 0, 0
	}
	textCache.Set(key, texture)
	return texture, surface.W, surface.H
}

// ResetTextCache destroys every cached texture. Call it when the renderer is
// torn down, since the textures belong to it.
func ResetTextCache() { textCache.Destroy() }

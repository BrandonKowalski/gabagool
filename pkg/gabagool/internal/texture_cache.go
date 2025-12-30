package internal

import "github.com/veandco/go-sdl2/sdl"

const defaultMaxCacheSize = 5

type TextureCache struct {
	textures map[string]*sdl.Texture
	order    []string // tracks insertion order for LRU eviction
	maxSize  int
}

func NewTextureCache() *TextureCache {
	return NewTextureCacheWithSize(defaultMaxCacheSize)
}

func NewTextureCacheWithSize(maxSize int) *TextureCache {
	return &TextureCache{
		textures: make(map[string]*sdl.Texture),
		order:    make([]string, 0, maxSize),
		maxSize:  maxSize,
	}
}

func (c *TextureCache) Get(key string) *sdl.Texture {
	if texture, exists := c.textures[key]; exists {
		// Move to end (most recently used)
		c.moveToEnd(key)
		return texture
	}
	return nil
}

func (c *TextureCache) Set(key string, texture *sdl.Texture) {
	// If key already exists, just update and move to end
	if _, exists := c.textures[key]; exists {
		c.textures[key] = texture
		c.moveToEnd(key)
		return
	}

	// Evict oldest if at capacity
	if len(c.order) >= c.maxSize {
		c.evictOldest()
	}

	c.textures[key] = texture
	c.order = append(c.order, key)
}

func (c *TextureCache) moveToEnd(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			c.order = append(c.order, key)
			return
		}
	}
}

func (c *TextureCache) evictOldest() {
	if len(c.order) == 0 {
		return
	}

	oldest := c.order[0]
	c.order = c.order[1:]

	if texture, exists := c.textures[oldest]; exists {
		texture.Destroy()
		delete(c.textures, oldest)
	}
}

func (c *TextureCache) Destroy() {
	for _, texture := range c.textures {
		texture.Destroy()
	}
	c.textures = make(map[string]*sdl.Texture)
	c.order = c.order[:0]
}

package game

import (
	"path/filepath"
	"strings"

	"kaijuengine.com/engine/assets"
)

// contentDB wraps a flat file database with the standard Kaiju extension-based
// path mapping so engine lookups like "swapchain.renderpass" resolve to
// "renderer/passes/swapchain.renderpass" inside the content root.
//
// This mirrors the editor's toEmbedPath mapping so the same asset keys work
// in both editor and standalone builds.
type contentDB struct {
	base assets.Database
}

// mapPaths maps a key to its candidate content-layout paths by extension.
// Mirrors the editor's toEmbedPath mapping: .png is searched across
// textures/, fonts/, meshes/ in order.
func mapPaths(key string) []string {
	key = filepath.ToSlash(key)
	if strings.Contains(key, "/") {
		// Already a subpath — use as-is.
		return []string{key}
	}
	switch filepath.Ext(key) {
	case ".bin":
		return []string{filepath.Join("fonts", key)}
	case ".fbx", ".gltf":
		return []string{filepath.Join("meshes", key)}
	case ".png":
		return []string{
			filepath.Join("textures", key),
			filepath.Join("fonts", key),
			filepath.Join("meshes", key),
		}
	case ".css", ".html":
		return []string{filepath.Join("ui", key)}
	case ".material":
		return []string{filepath.Join("renderer", "materials", key)}
	case ".renderpass":
		return []string{filepath.Join("renderer", "passes", key)}
	case ".shaderpipeline":
		return []string{filepath.Join("renderer", "pipelines", key)}
	case ".shader":
		return []string{filepath.Join("renderer", "shaders", key)}
	case ".spv":
		return []string{filepath.Join("renderer", "spv", key)}
	default:
		return []string{key}
	}
}

func (c *contentDB) PostWindowCreate(h assets.PostWindowCreateHandle) error {
	return c.base.PostWindowCreate(h)
}

func (c *contentDB) Cache(key string, data []byte) { c.base.Cache(key, data) }
func (c *contentDB) CacheRemove(key string)        { c.base.CacheRemove(key) }
func (c *contentDB) CacheClear()                   { c.base.CacheClear() }

func (c *contentDB) Read(key string) ([]byte, error) {
	for _, mapped := range mapPaths(key) {
		if b, err := c.base.Read(mapped); err == nil {
			return b, nil
		}
	}
	// Fall back to a direct read for special keys.
	return c.base.Read(key)
}

func (c *contentDB) ReadText(key string) (string, error) {
	b, err := c.Read(key)
	return string(b), err
}

func (c *contentDB) Exists(key string) bool {
	for _, mapped := range mapPaths(key) {
		if c.base.Exists(mapped) {
			return true
		}
	}
	return c.base.Exists(key)
}

func (c *contentDB) Close() { c.base.Close() }

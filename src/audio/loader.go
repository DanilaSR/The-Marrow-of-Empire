package audio

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"kaijuengine.com/engine/assets"
)

// Loader loads audio clips from the asset database (Story AUDIO-001 AC-3).
// Keys are relative to the assets/audio/ root: SFX live under sfx/ (WAV,
// uncompressed), music under music/ (OGG, compressed) and ambient under
// ambient/ (Story AUDIO-002) — impl note 3. Format gating happens here, in
// the game layer, before the backend sees the bytes.
type Loader struct {
	db      assets.Database
	backend Backend
}

// NewLoader creates a Loader over an asset database and a Backend. The
// backend may be the disabled backend — loads then report ErrAudioDisabled
// without touching the device (AC-5).
func NewLoader(db assets.Database, backend Backend) *Loader {
	return &Loader{db: db, backend: backend}
}

// LoadSFX loads an uncompressed WAV clip. Only .wav is accepted for SFX
// (uncompressed, per impl note 3); a missing file or an unsupported format
// returns nil plus a logged error — never a crash (AC-3 edge cases).
func (l *Loader) LoadSFX(key string) (Clip, error) {
	if err := requireExt(key, ".wav"); err != nil {
		return nil, l.logLoadError(key, err)
	}
	clip, err := l.backend.LoadSound(l.db, key)
	if err != nil {
		return nil, l.logLoadError(key, err)
	}
	return clip, nil
}

// LoadMusic loads a compressed OGG clip; WAV is also accepted (the engine
// decodes both through the same SoLoud path). Missing files and unsupported
// formats return nil plus a logged error — never a crash (AC-3 edge cases).
func (l *Loader) LoadMusic(key string) (Clip, error) {
	if err := requireExt(key, ".ogg", ".wav"); err != nil {
		return nil, l.logLoadError(key, err)
	}
	clip, err := l.backend.LoadMusic(l.db, key)
	if err != nil {
		return nil, l.logLoadError(key, err)
	}
	return clip, nil
}

// logLoadError records a failed load (AC-3: "logged error, returns nil
// buffer") and returns the error to the caller.
func (l *Loader) logLoadError(key string, err error) error {
	slog.Error("audio: failed to load clip", "key", key, "error", err)
	return err
}

// requireExt rejects keys whose extension is not in the accepted set,
// case-insensitively. Blank keys are rejected too.
func requireExt(key string, accepted ...string) error {
	if key == "" {
		return fmt.Errorf("audio: blank asset key")
	}
	ext := strings.ToLower(filepath.Ext(key))
	for _, want := range accepted {
		if ext == want {
			return nil
		}
	}
	return fmt.Errorf("audio: unsupported format %q for key %q (accepted: %s)",
		ext, key, strings.Join(accepted, ", "))
}

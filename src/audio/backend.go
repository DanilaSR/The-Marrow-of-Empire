package audio

import (
	"errors"

	"kaijuengine.com/engine/assets"
	"kaijuengine.com/matrix"
	kaijuaudio "kaijuengine.com/platform/audio"
)

// ErrAudioDisabled is returned by playback-facing APIs when the audio device
// is unavailable (Story AUDIO-001 AC-5). Systems check System.Enabled() and
// skip playback, so this is a safety net, not the primary path.
var ErrAudioDisabled = errors.New("audio: subsystem disabled — no audio device available")

// Backend abstracts the engine audio device behind the minimal surface the
// game layer needs (dependency injection per the project testing standards —
// tests inject fakes, production uses NewKaijuBackend). The interface is a
// subset of kaijuengine.com/platform/audio.Audio.
type Backend interface {
	// SetSoundVolume sets the SFX bus volume, clamped to [0, 1].
	SetSoundVolume(volume float32)
	// SetMusicVolume sets the music bus volume, clamped to [0, 1].
	SetMusicVolume(volume float32)
	// LoadSound loads and decodes an audio clip (WAV/OGG) from the asset
	// database for the SFX bus.
	LoadSound(db assets.Database, key string) (Clip, error)
	// LoadMusic loads and decodes an audio clip (WAV/OGG) from the asset
	// database for the music bus.
	LoadMusic(db assets.Database, key string) (Clip, error)
}

// Clip is a loaded, decoded audio buffer. Playback is out of scope for the
// foundation story (Story AUDIO-002..004); the handle is the AC-3 load
// contract.
type Clip interface {
	// Length returns the clip duration in seconds.
	Length() float64
}

// SpatialSink receives the listener state for engine spatialization
// (distance attenuation and panning). The production Kaiju backend does not
// implement this interface yet: the engine's Go audio bindings
// (kaiju/src/platform/audio/soloud.c.go) do not expose the SoLoud 3D API
// (Soloud_set3dListenerPosition/At/Up, Soloud_update3dAudio, Soloud_play3d)
// even though the C header soloud_c.h declares them — verified against
// engine source 2026-08-11.
//
// When the bindings land (a small mechanical engine patch mirroring the
// existing wrapper style, pending lead-programmer approval), NewKaijuBackend
// implements this interface and the listener state System already tracks
// flows to the engine with zero game-layer changes. Until then the listener
// position is tracked game-side and System.SpatialSupported() reports false.
type SpatialSink interface {
	// SetListenerPosition sets the listener position in world space.
	SetListenerPosition(position matrix.Vec3)
	// SetListenerAt sets the listener forward (look) direction.
	SetListenerAt(at matrix.Vec3)
	// SetListenerUp sets the listener up vector.
	SetListenerUp(up matrix.Vec3)
	// SetSoundSpeed sets the speed of sound in world units per second.
	SetSoundSpeed(speed float32)
	// UpdateSpatialization flushes pending listener/source state into the
	// audio engine (SoLoud_update3dAudio).
	UpdateSpatialization()
}

// kaijuBackend adapts the engine audio device (kaijuengine.com/platform/
// audio) to the game-layer Backend interface.
type kaijuBackend struct {
	device *kaijuaudio.Audio
}

// NewKaijuBackend wraps the engine audio device created by
// host.InitializeAudio. The device must not be nil; pass nil to NewSystem to
// create a disabled system instead.
func NewKaijuBackend(device *kaijuaudio.Audio) Backend {
	return &kaijuBackend{device: device}
}

func (k *kaijuBackend) SetSoundVolume(volume float32) { k.device.SetSoundVolume(volume) }

func (k *kaijuBackend) SetMusicVolume(volume float32) { k.device.SetMusicVolume(volume) }

func (k *kaijuBackend) LoadSound(db assets.Database, key string) (Clip, error) {
	clip, err := k.device.LoadSound(db, key)
	if err != nil {
		return nil, err
	}
	return &kaijuClip{clip: clip}, nil
}

func (k *kaijuBackend) LoadMusic(db assets.Database, key string) (Clip, error) {
	clip, err := k.device.LoadMusic(db, key)
	if err != nil {
		return nil, err
	}
	return &kaijuClip{clip: clip}, nil
}

// kaijuClip adapts the engine's *kaijuaudio.AudioClip to the game-layer Clip
// contract.
type kaijuClip struct {
	clip *kaijuaudio.AudioClip
}

func (c *kaijuClip) Length() float64 { return c.clip.Length() }

// disabledBackend is the no-op Backend for the missing-hardware path (Story
// AUDIO-001 AC-5). Every method is inert so a disabled system can never
// touch the audio device; loads report ErrAudioDisabled.
type disabledBackend struct{}

func (disabledBackend) SetSoundVolume(float32)          {}
func (disabledBackend) SetMusicVolume(float32)          {}
func (disabledBackend) LoadSound(assets.Database, string) (Clip, error) {
	return nil, ErrAudioDisabled
}
func (disabledBackend) LoadMusic(assets.Database, string) (Clip, error) {
	return nil, ErrAudioDisabled
}

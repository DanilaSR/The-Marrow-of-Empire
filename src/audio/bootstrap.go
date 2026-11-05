package audio

import (
	"log/slog"

	"kaijuengine.com/engine/assets"
	kaijuaudio "kaijuengine.com/platform/audio"

	"marrow-of-empire/entity"
)

// AudioHost is the minimal engine host surface Bootstrap needs. The engine's
// *engine.Host satisfies it; tests inject fakes (the same dependency-
// injection pattern as entity.KaijuHost).
type AudioHost interface {
	// InitializeAudio initializes the engine audio device. It returns an
	// error when no audio hardware is available or the driver fails.
	InitializeAudio() error
	// Audio returns the initialized engine audio device. Only valid after
	// InitializeAudio succeeded.
	Audio() *kaijuaudio.Audio
}

// Bootstrap initializes the audio subsystem at game launch (Story AUDIO-001
// AC-1 / AC-5). The game entry calls it after the renderer is up and before
// game logic (impl note 1), then registers System.Update on host.Updater
// behind the camera controller update.
//
// On success the returned system reports StatusReady and wraps the engine
// device. When the device cannot be initialized — no hardware, driver
// failure — a warning is logged, the game continues, and the returned system
// reports StatusDisabled with playback APIs returning ErrAudioDisabled.
// Bootstrap never fails the launch.
func Bootstrap(host AudioHost, cfg Config, db assets.Database, bus entity.EventBus) *System {
	if err := host.InitializeAudio(); err != nil {
		slog.Warn("audio: device initialization failed; audio disabled (Story AUDIO-001 AC-5)",
			"error", err)
		return NewSystem(cfg, nil, db, bus)
	}
	return NewSystem(cfg, NewKaijuBackend(host.Audio()), db, bus)
}

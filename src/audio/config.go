package audio

// Config carries the data-driven audio settings (Story AUDIO-001 impl note
// 4). Gameplay values live here — never hardcoded in system logic — so
// designers can tune without touching code. Zero-value Config fields fall
// back to the defaults below (same pattern as asset.Config and save.Config).
type Config struct {
	// MasterVolume is the initial master volume in [MinVolume, MaxVolume]
	// (0.0 = silence, 1.0 = full). Applied to all audio buses at startup.
	// Default 0.8 (the story QA reference level). Values outside the range
	// are clamped.
	MasterVolume float32
	// Muted starts the subsystem muted (silence regardless of volume).
	// Default false.
	Muted bool
	// SoundSpeed is the spatial sound speed in world units per second,
	// pushed to the engine listener for Doppler-based spatialization
	// (SoLoud default 343.0). Default 343.0.
	SoundSpeed float32
}

// Volume bounds (Story AUDIO-001 AC-4 edge case: >1.0 clamps to 1.0, <0.0
// clamps to 0.0).
const (
	MinVolume float32 = 0
	MaxVolume float32 = 1
)

// Default tuning values for the audio foundation story.
const (
	// DefaultMasterVolume is the QA reference output level from the story
	// ("Audio playing at master volume 0.8").
	DefaultMasterVolume float32 = 0.8
	// DefaultSoundSpeed is the speed of sound in world units per second
	// (1 unit = 1 meter, ADR-0004), matching the SoLoud engine default.
	DefaultSoundSpeed float32 = 343.0
)

// DefaultConfig returns the story-prescribed configuration.
func DefaultConfig() Config {
	return Config{
		MasterVolume: DefaultMasterVolume,
		Muted:        false,
		SoundSpeed:   DefaultSoundSpeed,
	}
}

// configWithDefaults fills zero-value fields with the project defaults so a
// zero-value Config can never produce a silent-by-accident or unsound system.
func configWithDefaults(cfg Config) Config {
	if cfg.MasterVolume == 0 && cfg.Muted == false && cfg.SoundSpeed == 0 {
		return DefaultConfig()
	}
	if cfg.SoundSpeed <= 0 {
		cfg.SoundSpeed = DefaultSoundSpeed
	}
	cfg.MasterVolume = clampVolume(cfg.MasterVolume)
	return cfg
}

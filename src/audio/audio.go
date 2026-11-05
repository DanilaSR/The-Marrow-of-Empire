// Package audio implements the Audio Engine Foundation (Story AUDIO-001,
// requirement TR-audio-001, GDD design/gdd/audio-system.md): subsystem
// bootstrap, the spatial audio listener attached to the isometric camera,
// WAV/OGG asset loading from assets/audio/, master volume + mute controls,
// and graceful handling of missing audio hardware.
//
// Layer: Presentation (control-manifest 2026-08-09). The package depends
// downward on Foundation (entity event bus, engine assets database) and on
// the engine's native audio device (kaijuengine.com/platform/audio, SoLoud
// backend) behind the Backend interface — the game layer never talks to the
// device directly, so tests inject fakes.
//
// Story scope boundary: ambient soundscapes, adaptive music and SFX playback
// are Stories AUDIO-002..004 and are NOT implemented here. This story only
// establishes the infrastructure those stories build on.
package audio

import (
	"log/slog"

	"kaijuengine.com/engine/assets"
	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// Status reports the audio subsystem state (Story AUDIO-001 AC-1 / AC-5).
type Status int

const (
	// StatusDisabled means the audio device could not be initialized (no
	// hardware, driver failure). The game continues normally; playback
	// requests return ErrAudioDisabled (AC-5).
	StatusDisabled Status = iota
	// StatusReady means the audio device is initialized and usable (AC-1).
	StatusReady
)

// String implements fmt.Stringer. The values match the story QA vocabulary:
// "ready" and "disabled".
func (s Status) String() string {
	switch s {
	case StatusReady:
		return "ready"
	default:
		return "disabled"
	}
}

// System is the game-layer audio subsystem. It owns the listener, the master
// volume/mute state, the asset loader and the ADR-0011 event subscriptions.
// All engine access goes through the Backend and SpatialSink interfaces.
//
// Construction is engine-free (NewSystem); Bootstrap wires it to a running
// engine host at game launch.
type System struct {
	cfg        Config
	backend    Backend
	loader     *Loader
	bus        entity.EventBus
	status     Status
	lastError  error
	listener   Listener
	spatial    SpatialSink // nil when the backend cannot spatialize (engine gap, see backend.go)
	camera     Camera      // listener source; nil until AttachCamera
	masterVol  float32     // clamped [MinVolume, MaxVolume]; remembered across mute
	muted      bool
}

// NewSystem creates the audio subsystem over the given Backend. When backend
// is nil the system is created in the disabled state (Story AUDIO-001 AC-5 —
// missing audio hardware): a warning is logged, playback APIs return
// ErrAudioDisabled and the game continues. The bus may be nil to skip event
// subscription.
//
// The configured master volume and mute flag are applied to the backend
// immediately, so a fresh system starts at the intended output level.
func NewSystem(cfg Config, backend Backend, db assets.Database, bus entity.EventBus) *System {
	cfg = configWithDefaults(cfg)
	s := &System{
		cfg:       cfg,
		bus:       bus,
		masterVol: cfg.MasterVolume,
		muted:     cfg.Muted,
	}
	if backend == nil {
		s.status = StatusDisabled
		s.lastError = ErrAudioDisabled
		s.backend = disabledBackend{}
		slog.Warn("audio: no audio device available; audio disabled (Story AUDIO-001 AC-5)",
			"status", s.status)
	} else {
		s.status = StatusReady
		s.backend = backend
		if sink, ok := backend.(SpatialSink); ok {
			s.spatial = sink
		}
	}
	s.loader = NewLoader(db, s.backend)
	s.applyVolume() // pushes cfg.MasterVolume (or silence when muted) to the backend
	if bus != nil {
		s.subscribe(bus)
	}
	return s
}

// Status returns the subsystem state: StatusReady or StatusDisabled.
func (s *System) Status() Status { return s.status }

// Enabled reports whether audio playback is available. Systems that trigger
// sounds must check this flag and skip playback when false (Story AUDIO-001
// impl note 5).
func (s *System) Enabled() bool { return s.status == StatusReady }

// LastError returns the reason the subsystem was disabled, or nil when ready.
func (s *System) LastError() error { return s.lastError }

// Listener returns the current spatial listener state: the world-space
// anchor (camera focus), the look direction and the up vector. Later audio
// stories read this for spatialized playback (Story AUDIO-002..004).
func (s *System) Listener() Listener { return s.listener }

// SpatialSupported reports whether the backend can spatialize audio. The
// Kaiju engine's Go audio bindings do not yet expose the SoLoud 3D API
// (verified against kaiju/src/platform/audio/soloud.c.go), so the production
// backend reports false and listener state is tracked game-side until the
// engine binding exists. See SpatialSink in backend.go.
func (s *System) SpatialSupported() bool { return s.spatial != nil }

// AttachCamera binds the audio listener to a camera. The camera's focus is
// the listener anchor: panning moves the listener with the camera, zoom does
// not (Story AUDIO-001 AC-2 — zoom changes the ortho half-height and the
// camera's standoff position, never the focus, so the listener stays put).
//
// The game entry calls this once after creating the isometric camera
// (ADR-0003: 30° pitch / 45° yaw / 0° roll). The camera is read every frame
// from Update, never cached.
func (s *System) AttachCamera(cam Camera) { s.camera = cam }

// Update advances the subsystem by one frame. It re-syncs the listener from
// the attached camera (AC-2: listener updates each frame, within 1 frame of
// camera movement) and pushes listener state to the SpatialSink when the
// backend supports spatialization.
//
// Call this after the camera controller's own per-frame update — the game
// entry registers it on host.Updater behind the camera update, or on
// host.LateUpdater.
func (s *System) Update(deltaTime float64) {
	if s.camera != nil {
		s.syncListener()
	}
}

// syncListener copies the camera transform into the listener state and, when
// a SpatialSink is available, pushes it to the engine.
func (s *System) syncListener() {
	focus := s.camera.Focus()
	pos := s.camera.Position()
	s.listener.position = focus
	// The listener looks along the camera's view direction: from the camera
	// position toward the focus. A degenerate camera (position == focus) keeps
	// the previous orientation.
	if view := focus.Subtract(pos); view.LengthSquared() > epsilonSquared {
		s.listener.at = view.Normal()
	} else if s.listener.at.LengthSquared() <= epsilonSquared {
		s.listener.at = matrix.Vec3{0, 0, -1}
	}
	s.listener.up = matrix.Vec3Up()
	if s.spatial != nil {
		s.spatial.SetListenerPosition(s.listener.position)
		s.spatial.SetListenerAt(s.listener.at)
		s.spatial.SetListenerUp(s.listener.up)
		s.spatial.SetSoundSpeed(s.cfg.SoundSpeed)
		s.spatial.UpdateSpatialization()
	}
}

// epsilonSquared is the squared-length tolerance for degenerate direction
// vectors (float32 arithmetic).
const epsilonSquared float32 = 1e-12

// ── Asset loading (AC-3) ─────────────────────────────────────────────────────

// LoadSFX loads an uncompressed WAV clip from the asset database, keyed
// relative to the assets/audio/ root (e.g. "sfx/test_tone.wav"). Only .wav
// is accepted for SFX (Story AUDIO-001 impl note 3). A missing or
// undecodable clip returns nil plus a logged error — never a crash.
func (s *System) LoadSFX(key string) (Clip, error) { return s.loader.LoadSFX(key) }

// LoadMusic loads a compressed OGG (or WAV) clip from the asset database,
// keyed relative to the assets/audio/ root (e.g. "music/test_music.ogg").
// Missing or undecodable clips return nil plus a logged error — never a
// crash.
func (s *System) LoadMusic(key string) (Clip, error) { return s.loader.LoadMusic(key) }

// ── Volume and mute (AC-4) ───────────────────────────────────────────────────

// SetMasterVolume sets the master volume in [MinVolume, MaxVolume]; values
// outside the range are clamped (Story AUDIO-001 AC-4 edge case). The volume
// applies to all audio buses (sound and music). While muted the backend stays
// silent and the setting is remembered for unmute.
func (s *System) SetMasterVolume(volume float32) {
	s.masterVol = clampVolume(volume)
	if !s.muted {
		s.pushVolume(s.masterVol)
	}
}

// MasterVolume returns the current master volume setting, clamped to
// [MinVolume, MaxVolume]. The setting is remembered across mute.
func (s *System) MasterVolume() float32 { return s.masterVol }

// SetMuted silences or restores all audio buses. Mute produces silence
// regardless of the volume setting; unmute restores the last volume
// (Story AUDIO-001 AC-4).
func (s *System) SetMuted(muted bool) {
	s.muted = muted
	if muted {
		s.pushVolume(0)
	} else {
		s.pushVolume(s.masterVol)
	}
}

// Muted reports whether the subsystem is muted.
func (s *System) Muted() bool { return s.muted }

// pushVolume applies a volume to every audio bus.
func (s *System) pushVolume(volume float32) {
	s.backend.SetSoundVolume(volume)
	s.backend.SetMusicVolume(volume)
}

// applyVolume pushes the effective volume at construction time: silence when
// the config starts muted, otherwise the configured master volume.
func (s *System) applyVolume() {
	if s.muted {
		s.pushVolume(0)
	} else {
		s.pushVolume(s.masterVol)
	}
}

// clampVolume clamps a volume setting to [MinVolume, MaxVolume].
func clampVolume(volume float32) float32 {
	if volume < MinVolume {
		return MinVolume
	}
	if volume > MaxVolume {
		return MaxVolume
	}
	return volume
}

// ── Event bus integration (ADR-0011, impl note 6) ────────────────────────────

// EventPriority is the ADR-0011 dispatch priority the audio layer registers
// with. Framework foundation subscribers use priority 1; the presentation
// layer registers at 10 so entity lifecycle bookkeeping runs first.
const EventPriority = 10

// subscribe registers the audio subsystem on the entity lifecycle events.
// The foundation story establishes the subscriber pattern — Story AUDIO-004
// hooks SFX triggers into these handlers.
func (s *System) subscribe(bus entity.EventBus) {
	bus.Subscribe(entity.EventEntityCreated, EventPriority, s.OnEntityEvent)
	bus.Subscribe(entity.EventEntityDestroyed, EventPriority, s.OnEntityEvent)
	bus.Subscribe(entity.EventEntitySuspended, EventPriority, s.OnEntityEvent)
	bus.Subscribe(entity.EventEntityReactivated, EventPriority, s.OnEntityEvent)
}

// OnEntityEvent is the ADR-0011 subscriber entry point for entity lifecycle
// events. The audio layer never emits audio-specific events at this layer
// (Story AUDIO-001 impl note 6); the handler is the hook later stories
// extend.
func (s *System) OnEntityEvent(ev entity.Event) {
	slog.Debug("audio: entity lifecycle event", "type", ev.Type, "source", ev.Source)
}

// Integration tests for Story AUDIO-001: Audio Engine Foundation & Spatial
// Listener.
//
// Evidence path: production/epics/audio-system/story-001-audio-engine-foundation.md
// Required evidence: this file must exist and pass.
//
// The tests wire the full game-layer stack with real engine types and fakes
// at every boundary: the audio Backend, the asset Database and the ADR-0011
// EventBus are interfaces, so the device, storage and bus are injectable
// (dependency-injection pattern, same as the entity.KaijuHost mocks in
// tests/integration/camera). The only engine runtime that cannot run
// headless is *engine.Host itself; Bootstrap takes the minimal AudioHost
// interface so both the with-device and without-device launch paths are
// covered here.
//
// QA test case -> test function map:
//
//	AC-1   test_subsystem_initializes_ready                -> TestBootstrap_WithAudioDevice_ReportsReady
//	AC-1   test_bootstrap_never_fails_launch               -> TestBootstrap_WithoutDevice_ReportsDisabled
//	AC-2   test_listener_tracks_camera_within_1_frame      -> TestListener_TracksCameraPan_WithinOneFrame
//	AC-2   test_zoom_does_not_move_listener                -> TestListener_UnchangedByZoomChange
//	AC-2   test_listener_state_pushed_to_engine            -> TestListener_PushesStateToSpatialSink_EachFrame
//	AC-3   test_wav_and_ogg_load_from_assets_audio         -> TestLoader_LoadsWavAndOggFromAssetDirectory
//	AC-3   test_missing_file_returns_nil_no_crash          -> TestLoader_MissingFile_ReturnsNilAndError
//	AC-3   test_unsupported_format_returns_error           -> TestLoader_UnsupportedFormat_ReturnsError
//	AC-4   test_master_volume_clamps_0_1                   -> TestMasterVolume_ClampsAndPushesToBackend
//	AC-4   test_mute_silences_regardless_of_volume         -> TestMute_ProducesSilenceRegardlessOfVolume
//	AC-5   test_missing_hardware_disables_with_warning     -> TestNoAudioHardware_SystemDisabled_LogsWarning_NoCrash
//	AC-5   test_unplugged_mid_game_system_stays_disabled   -> TestNoAudioHardware_SystemDisabled_LogsWarning_NoCrash
//	impl6  test_eventbus_subscriber_pattern                -> TestEventSubscription_RegistersAtPriority_AndDispatches
//	QA     test_status_reports_ready_and_disabled          -> TestStatus_String_ReportsReadyAndDisabled
package audio_test

import (
	"bytes"
	"errors"
	"log/slog"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"kaijuengine.com/engine/assets"
	"kaijuengine.com/matrix"
	kaijuaudio "kaijuengine.com/platform/audio"

	"marrow-of-empire/audio"
	"marrow-of-empire/entity"
)

// eps is the tolerance for unit-length and position assertions.
const eps float32 = 1e-3

// ── Fakes ────────────────────────────────────────────────────────────────────

// fakeCamera is a controllable Camera double: the listener anchor is the
// focus (zoom changes the camera position/standoff, never the focus).
type fakeCamera struct {
	pos   matrix.Vec3
	focus matrix.Vec3
}

func (f *fakeCamera) Position() matrix.Vec3 { return f.pos }
func (f *fakeCamera) Focus() matrix.Vec3    { return f.focus }

// fakeClip is a loaded-clip double.
type fakeClip struct{ length float64 }

func (f *fakeClip) Length() float64 { return f.length }

// recordingBackend records every call the game layer makes on the device.
type recordingBackend struct {
	soundVolumes []float32
	musicVolumes []float32
	soundKeys    []string
	musicKeys    []string
	clips        map[string]audio.Clip
}

func (r *recordingBackend) SetSoundVolume(v float32) { r.soundVolumes = append(r.soundVolumes, v) }
func (r *recordingBackend) SetMusicVolume(v float32) { r.musicVolumes = append(r.musicVolumes, v) }

func (r *recordingBackend) LoadSound(_ assets.Database, key string) (audio.Clip, error) {
	r.soundKeys = append(r.soundKeys, key)
	if c, ok := r.clips[key]; ok {
		return c, nil
	}
	return nil, errors.New("audio test: clip not in backend")
}

func (r *recordingBackend) LoadMusic(_ assets.Database, key string) (audio.Clip, error) {
	r.musicKeys = append(r.musicKeys, key)
	if c, ok := r.clips[key]; ok {
		return c, nil
	}
	return nil, errors.New("audio test: clip not in backend")
}

// spatialBackend is a recordingBackend that also implements SpatialSink —
// the contract the engine bindings will fulfil once they expose the SoLoud
// 3D API.
type spatialBackend struct {
	*recordingBackend
	positions []matrix.Vec3
	ats       []matrix.Vec3
	ups       []matrix.Vec3
	speeds    []float32
	updates   int
}

func (s *spatialBackend) SetListenerPosition(p matrix.Vec3) {
	s.positions = append(s.positions, p)
}
func (s *spatialBackend) SetListenerAt(at matrix.Vec3) { s.ats = append(s.ats, at) }
func (s *spatialBackend) SetListenerUp(up matrix.Vec3) { s.ups = append(s.ups, up) }
func (s *spatialBackend) SetSoundSpeed(speed float32)  { s.speeds = append(s.speeds, speed) }
func (s *spatialBackend) UpdateSpatialization()        { s.updates++ }

// fakeAudioHost is the AudioHost double Bootstrap talks to. The device double
// is a zero-value engine Audio: SetSoundVolume/SetMusicVolume only touch the
// bus maps, which are empty — safe without an initialized SoLoud instance.
type fakeAudioHost struct {
	initErr error
	dev     *kaijuaudio.Audio
}

func (f *fakeAudioHost) InitializeAudio() error { return f.initErr }
func (f *fakeAudioHost) Audio() *kaijuaudio.Audio { return f.dev }

// recordingBus is an EventBus double that records subscriptions and
// dispatches synchronously like the real EventBusImpl (ADR-0011).
type recordingBus struct {
	subs       map[entity.EventType][]entity.EventHandler
	types      []entity.EventType
	priorities []int
}

func (r *recordingBus) Subscribe(eventType entity.EventType, priority int, handler entity.EventHandler) {
	r.subs[eventType] = append(r.subs[eventType], handler)
	r.types = append(r.types, eventType)
	r.priorities = append(r.priorities, priority)
}

func (r *recordingBus) Emit(ev entity.Event) {
	for _, h := range r.subs[ev.Type] {
		h(ev)
	}
}

func (r *recordingBus) Unsubscribe(eventType entity.EventType, handler entity.EventHandler) {
	subs := r.subs[eventType]
	for i, h := range subs {
		if reflect.ValueOf(h).Pointer() == reflect.ValueOf(handler).Pointer() {
			r.subs[eventType] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

// captureLogs redirects slog into a buffer for the test's duration and
// returns the buffer. Tests that capture logs must not run in parallel.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func last(v []float32) float32 { return v[len(v)-1] }

func newReadySystem(bk audio.Backend) *audio.System {
	return audio.NewSystem(audio.DefaultConfig(), bk, nil, nil)
}

// ── AC-1: initializes on game launch without errors ──────────────────────────

// TestBootstrap_WithAudioDevice_ReportsReady covers AC-1: on a system with
// audio hardware, bootstrap completes with status "ready", the device handle
// is non-nil and no error is logged.
func TestBootstrap_WithAudioDevice_ReportsReady(t *testing.T) {
	buf := captureLogs(t)
	host := &fakeAudioHost{dev: &kaijuaudio.Audio{}}
	sys := audio.Bootstrap(host, audio.DefaultConfig(), nil, nil)

	if sys.Status() != audio.StatusReady {
		t.Errorf("status = %v, want ready", sys.Status())
	}
	if !sys.Enabled() {
		t.Error("Enabled() = false, want true")
	}
	out := buf.String()
	if strings.Contains(out, "level=ERROR") {
		t.Errorf("error logged on successful bootstrap: %s", out)
	}
	if strings.Contains(out, "audio disabled") {
		t.Errorf("warning logged on successful bootstrap: %s", out)
	}
}

// TestBootstrap_WithoutDevice_ReportsDisabled covers AC-1's edge case and
// AC-5: when initialization fails, bootstrap still succeeds from the game's
// perspective — status "disabled", warning logged, game continues.
func TestBootstrap_WithoutDevice_ReportsDisabled(t *testing.T) {
	buf := captureLogs(t)
	host := &fakeAudioHost{initErr: errors.New("no audio device")}
	sys := audio.Bootstrap(host, audio.DefaultConfig(), nil, nil)

	if sys.Status() != audio.StatusDisabled {
		t.Errorf("status = %v, want disabled", sys.Status())
	}
	if sys.Enabled() {
		t.Error("Enabled() = true, want false")
	}
	if !errors.Is(sys.LastError(), audio.ErrAudioDisabled) {
		t.Errorf("LastError() = %v, want ErrAudioDisabled", sys.LastError())
	}
	if !strings.Contains(buf.String(), "audio disabled") {
		t.Errorf("missing warning log: %s", buf.String())
	}
	// The game continues: every API is inert.
	sys.SetMasterVolume(0.5)
	sys.SetMuted(true)
	sys.AttachCamera(&fakeCamera{pos: matrix.NewVec3(100, 5, 50), focus: matrix.NewVec3(100, 0, 50)})
	sys.Update(0.016)
	clip, err := sys.LoadSFX("sfx/test_tone.wav")
	if clip != nil {
		t.Errorf("LoadSFX returned a clip while disabled: %v", clip)
	}
	if !errors.Is(err, audio.ErrAudioDisabled) {
		t.Errorf("LoadSFX err = %v, want ErrAudioDisabled", err)
	}
}

// ── AC-2: spatial listener attached to the camera, updated each frame ────────

// TestListener_TracksCameraPan_WithinOneFrame covers AC-2: camera focus pans
// from (100, 0, 50) to (200, 0, 50); after one Update the listener already
// sits at the new focus.
func TestListener_TracksCameraPan_WithinOneFrame(t *testing.T) {
	sys := newReadySystem(&recordingBackend{})
	cam := &fakeCamera{pos: matrix.NewVec3(100, 0, 50), focus: matrix.NewVec3(100, 0, 50)}
	sys.AttachCamera(cam)

	sys.Update(0.016)
	if got := sys.Listener().Position(); got != matrix.NewVec3(100, 0, 50) {
		t.Fatalf("listener position = %v, want (100, 0, 50)", got)
	}

	cam.pos = matrix.NewVec3(200, 0, 50)
	cam.focus = matrix.NewVec3(200, 0, 50)
	sys.Update(0.016)
	if got := sys.Listener().Position(); got != matrix.NewVec3(200, 0, 50) {
		t.Errorf("listener did not track camera within 1 frame: position = %v, want (200, 0, 50)", got)
	}
}

// TestListener_UnchangedByZoomChange covers AC-2's edge case: zoom registers
// 4u/10u/20u move the camera's standoff position but never the focus, so the
// listener position is unchanged.
func TestListener_UnchangedByZoomChange(t *testing.T) {
	sys := newReadySystem(&recordingBackend{})
	cam := &fakeCamera{pos: matrix.NewVec3(99.5, 2, 49.5), focus: matrix.NewVec3(100, 0, 50)}
	sys.AttachCamera(cam)

	sys.Update(0.016)
	want := sys.Listener().Position()

	for _, standoff := range []float32{4, 10, 20} {
		cam.pos = matrix.NewVec3(100-0.612*standoff, 0.5*standoff, 50-0.612*standoff)
		sys.Update(0.016)
		if got := sys.Listener().Position(); got != want {
			t.Errorf("zoom %vu moved the listener: position = %v, want %v", standoff, got, want)
		}
	}
	// The look direction stays unit length and points from the camera toward
	// the focus (down into the scene).
	at := sys.Listener().At()
	if l := at.Length(); math.Abs(float64(l)-1) > float64(eps) {
		t.Errorf("listener At is not unit length: %v (len %v)", at, l)
	}
	if at.Y() >= 0 {
		t.Errorf("listener At should point downward in isometric view, got %v", at)
	}
}

// TestListener_PushesStateToSpatialSink_EachFrame covers AC-2's engine
// contract: when the backend spatializes, listener state is pushed every
// frame. The production Kaiju backend does not implement SpatialSink yet
// (engine Go bindings gap, see src/audio/backend.go); this test pins the
// contract the engine patch must fulfil.
func TestListener_PushesStateToSpatialSink_EachFrame(t *testing.T) {
	bk := &spatialBackend{recordingBackend: &recordingBackend{}}
	sys := audio.NewSystem(audio.DefaultConfig(), bk, nil, nil)
	if !sys.SpatialSupported() {
		t.Fatal("SpatialSupported() = false, want true for a SpatialSink backend")
	}
	cam := &fakeCamera{pos: matrix.NewVec3(100, 5, 50), focus: matrix.NewVec3(100, 0, 50)}
	sys.AttachCamera(cam)

	sys.Update(0.016)
	if len(bk.positions) != 1 || bk.positions[0] != matrix.NewVec3(100, 0, 50) {
		t.Errorf("listener position pushed = %v, want [(100, 0, 50)]", bk.positions)
	}
	if bk.updates != 1 {
		t.Errorf("UpdateSpatialization calls = %d, want 1", bk.updates)
	}
	if len(bk.speeds) != 1 || bk.speeds[0] != audio.DefaultSoundSpeed {
		t.Errorf("sound speed pushed = %v, want %v", bk.speeds, audio.DefaultSoundSpeed)
	}
	if len(bk.ups) != 1 || bk.ups[0] != matrix.Vec3Up() {
		t.Errorf("up vector pushed = %v, want world up", bk.ups)
	}

	cam.focus = matrix.NewVec3(150, 0, 50)
	sys.Update(0.016)
	if len(bk.positions) != 2 || bk.positions[1] != matrix.NewVec3(150, 0, 50) {
		t.Errorf("listener not pushed every frame: positions = %v", bk.positions)
	}
}

// TestKaijuBackend_DoesNotReportSpatialSupport guards the documented engine
// gap: the Kaiju Go audio bindings expose no SoLoud 3D API today. When the
// engine patch lands, this test fails and the SpatialSink wiring in
// src/audio/backend.go must be completed.
func TestKaijuBackend_DoesNotReportSpatialSupport(t *testing.T) {
	sys := audio.NewSystem(audio.DefaultConfig(), audio.NewKaijuBackend(&kaijuaudio.Audio{}), nil, nil)
	if sys.SpatialSupported() {
		t.Error("SpatialSupported() = true: engine 3D audio bindings now exist — implement SpatialSink on kaijuBackend and remove the gap note")
	}
}

// ── AC-3: audio assets loadable from assets/audio/ ───────────────────────────

// TestLoader_LoadsWavAndOggFromAssetDirectory covers AC-3: the QA fixture
// files exist at the documented paths and both load through the asset
// pipeline, returning valid clips with no errors.
func TestLoader_LoadsWavAndOggFromAssetDirectory(t *testing.T) {
	// The test runs with cwd = tests/integration/audio; the repo assets/audio
	// root is three levels up. os.OpenRoot refuses relative paths containing
	// ".." (Go os.Root semantics), so resolve to an absolute path first.
	root, err := filepath.Abs("../../../assets/audio")
	if err != nil {
		t.Fatalf("resolve assets/audio path: %v", err)
	}
	db, err := assets.NewFileDatabase(root)
	if err != nil {
		t.Fatalf("open assets/audio database: %v", err)
	}
	defer db.Close()
	if !db.Exists("sfx/test_tone.wav") {
		t.Fatal("QA fixture missing: assets/audio/sfx/test_tone.wav")
	}
	if !db.Exists("music/test_music.ogg") {
		t.Fatal("QA fixture missing: assets/audio/music/test_music.ogg")
	}

	bk := &recordingBackend{clips: map[string]audio.Clip{
		"sfx/test_tone.wav":  &fakeClip{length: 1},
		"music/test_music.ogg": &fakeClip{length: 1},
	}}
	sys := audio.NewSystem(audio.DefaultConfig(), bk, db, nil)

	wav, err := sys.LoadSFX("sfx/test_tone.wav")
	if err != nil {
		t.Errorf("LoadSFX(sfx/test_tone.wav): %v", err)
	}
	if wav == nil {
		t.Error("LoadSFX returned nil clip")
	} else if wav.Length() != 1 {
		t.Errorf("clip length = %v, want 1s", wav.Length())
	}
	if len(bk.soundKeys) != 1 || bk.soundKeys[0] != "sfx/test_tone.wav" {
		t.Errorf("backend LoadSound keys = %v, want [sfx/test_tone.wav]", bk.soundKeys)
	}

	ogg, err := sys.LoadMusic("music/test_music.ogg")
	if err != nil {
		t.Errorf("LoadMusic(music/test_music.ogg): %v", err)
	}
	if ogg == nil {
		t.Error("LoadMusic returned nil clip")
	}
	if len(bk.musicKeys) != 1 || bk.musicKeys[0] != "music/test_music.ogg" {
		t.Errorf("backend LoadMusic keys = %v, want [music/test_music.ogg]", bk.musicKeys)
	}
}

// TestLoader_MissingFile_ReturnsNilAndError covers AC-3's edge case: a
// missing file returns a nil buffer with a logged error — no crash.
func TestLoader_MissingFile_ReturnsNilAndError(t *testing.T) {
	buf := captureLogs(t)
	sys := newReadySystem(&recordingBackend{}) // no clips registered

	clip, err := sys.LoadSFX("sfx/missing.wav")
	if clip != nil {
		t.Errorf("clip = %v, want nil for missing file", clip)
	}
	if err == nil {
		t.Error("err = nil, want error for missing file")
	}
	if !strings.Contains(buf.String(), "failed to load clip") {
		t.Errorf("missing file must be logged (AC-3): %s", buf.String())
	}
}

// TestLoader_UnsupportedFormat_ReturnsError covers AC-3's edge case: an
// unsupported format is rejected in the game layer before the backend is
// touched, with a logged error.
func TestLoader_UnsupportedFormat_ReturnsError(t *testing.T) {
	bk := &recordingBackend{}
	sys := newReadySystem(bk)

	if clip, err := sys.LoadSFX("sfx/test_tone.ogg"); clip != nil || err == nil {
		t.Errorf("LoadSFX(.ogg) = (%v, %v), want (nil, error) — SFX is WAV only", clip, err)
	}
	if clip, err := sys.LoadMusic("music/test_music.mp3"); clip != nil || err == nil {
		t.Errorf("LoadMusic(.mp3) = (%v, %v), want (nil, error)", clip, err)
	}
	if len(bk.soundKeys) != 0 || len(bk.musicKeys) != 0 {
		t.Errorf("backend called with unsupported formats: sounds %v, music %v", bk.soundKeys, bk.musicKeys)
	}
}

// ── AC-4: master volume and mute controls ────────────────────────────────────

// TestMasterVolume_ClampsAndPushesToBackend covers AC-4 and its edge cases:
// the default volume applies at construction, every change reaches both
// buses, and out-of-range values clamp to [0, 1].
func TestMasterVolume_ClampsAndPushesToBackend(t *testing.T) {
	bk := &recordingBackend{}
	sys := newReadySystem(bk)

	if got := last(bk.soundVolumes); got != audio.DefaultMasterVolume {
		t.Errorf("initial sound bus volume = %v, want %v", got, audio.DefaultMasterVolume)
	}
	if got := last(bk.musicVolumes); got != audio.DefaultMasterVolume {
		t.Errorf("initial music bus volume = %v, want %v", got, audio.DefaultMasterVolume)
	}

	sys.SetMasterVolume(0.5)
	if got := last(bk.soundVolumes); got != 0.5 {
		t.Errorf("sound bus after SetMasterVolume(0.5) = %v", got)
	}
	if got := last(bk.musicVolumes); got != 0.5 {
		t.Errorf("music bus after SetMasterVolume(0.5) = %v", got)
	}

	sys.SetMasterVolume(2.0) // > 1.0 clamps to 1.0
	if got := last(bk.soundVolumes); got != 1.0 {
		t.Errorf("volume after SetMasterVolume(2.0) = %v, want clamped 1.0", got)
	}
	if got := sys.MasterVolume(); got != 1.0 {
		t.Errorf("MasterVolume() = %v, want clamped 1.0", got)
	}

	sys.SetMasterVolume(-1.0) // < 0.0 clamps to 0.0
	if got := last(bk.soundVolumes); got != 0.0 {
		t.Errorf("volume after SetMasterVolume(-1.0) = %v, want clamped 0.0", got)
	}
	if got := sys.MasterVolume(); got != 0.0 {
		t.Errorf("MasterVolume() = %v, want clamped 0.0", got)
	}
}

// TestMute_ProducesSilenceRegardlessOfVolume covers AC-4: mute silences both
// buses immediately, volume changes while muted stay silent and are
// remembered, and unmute restores the setting.
func TestMute_ProducesSilenceRegardlessOfVolume(t *testing.T) {
	bk := &recordingBackend{}
	sys := newReadySystem(bk)

	sys.SetMasterVolume(0.5)
	sys.SetMuted(true)
	if !sys.Muted() {
		t.Error("Muted() = false after SetMuted(true)")
	}
	if got := last(bk.soundVolumes); got != 0.0 {
		t.Errorf("sound bus while muted = %v, want 0.0 (silence)", got)
	}
	if got := last(bk.musicVolumes); got != 0.0 {
		t.Errorf("music bus while muted = %v, want 0.0 (silence)", got)
	}

	sys.SetMasterVolume(0.9) // volume change while muted
	if got := last(bk.soundVolumes); got != 0.0 {
		t.Errorf("sound bus = %v while muted after volume change, want 0.0", got)
	}
	if got := sys.MasterVolume(); got != 0.9 {
		t.Errorf("MasterVolume() = %v, want remembered 0.9", got)
	}

	sys.SetMuted(false)
	if sys.Muted() {
		t.Error("Muted() = true after SetMuted(false)")
	}
	if got := last(bk.soundVolumes); got != 0.9 {
		t.Errorf("sound bus after unmute = %v, want restored 0.9", got)
	}
	if got := last(bk.musicVolumes); got != 0.9 {
		t.Errorf("music bus after unmute = %v, want restored 0.9", got)
	}
}

// ── AC-5: missing audio hardware handled gracefully ──────────────────────────

// TestNoAudioHardware_SystemDisabled_LogsWarning_NoCrash covers AC-5: with
// no output device the subsystem reports "disabled", logs a warning, and
// every API stays inert — the game never crashes. This also covers the
// "device unplugged mid-game" edge case at the game layer: a disabled
// system keeps reporting disabled and skipping playback.
func TestNoAudioHardware_SystemDisabled_LogsWarning_NoCrash(t *testing.T) {
	buf := captureLogs(t)
	sys := audio.NewSystem(audio.DefaultConfig(), nil, nil, nil)

	if sys.Status() != audio.StatusDisabled {
		t.Errorf("status = %v, want disabled", sys.Status())
	}
	if sys.Enabled() {
		t.Error("Enabled() = true, want false")
	}
	if !errors.Is(sys.LastError(), audio.ErrAudioDisabled) {
		t.Errorf("LastError() = %v, want ErrAudioDisabled", sys.LastError())
	}
	if !strings.Contains(buf.String(), "audio disabled") {
		t.Errorf("missing warning log (AC-5): %s", buf.String())
	}

	// Every API is inert — no panics, game continues.
	sys.SetMasterVolume(0.5)
	sys.SetMasterVolume(1.5)
	sys.SetMuted(true)
	sys.SetMuted(false)
	sys.AttachCamera(&fakeCamera{pos: matrix.NewVec3(100, 5, 50), focus: matrix.NewVec3(100, 0, 50)})
	sys.Update(0.016)
	if got := sys.MasterVolume(); got != 1.0 {
		t.Errorf("MasterVolume() = %v, want clamped 1.0 even while disabled", got)
	}
	if clip, err := sys.LoadMusic("music/test_music.ogg"); clip != nil || !errors.Is(err, audio.ErrAudioDisabled) {
		t.Errorf("LoadMusic = (%v, %v), want (nil, ErrAudioDisabled)", clip, err)
	}
}

// ── impl note 6: EventBus subscriber pattern (ADR-0011) ──────────────────────

// TestEventSubscription_RegistersAtPriority_AndDispatches covers impl note
// 6: the audio layer subscribes to the entity lifecycle events at its
// priority and its handler is dispatched synchronously. The handler is a
// placeholder — later stories hook SFX triggers into it.
func TestEventSubscription_RegistersAtPriority_AndDispatches(t *testing.T) {
	bus := &recordingBus{subs: map[entity.EventType][]entity.EventHandler{}}
	buf := captureLogs(t)
	newReadySystemWithBus(bus)

	wantTypes := []entity.EventType{
		entity.EventEntityCreated,
		entity.EventEntityDestroyed,
		entity.EventEntitySuspended,
		entity.EventEntityReactivated,
	}
	for _, want := range wantTypes {
		if len(bus.subs[want]) == 0 {
			t.Errorf("no subscription for event %q", want)
		}
	}
	for _, p := range bus.priorities {
		if p != audio.EventPriority {
			t.Errorf("subscription priority = %d, want %d", p, audio.EventPriority)
		}
	}

	bus.Emit(entity.Event{Type: entity.EventEntityDestroyed, Source: "test"})
	if !strings.Contains(buf.String(), "audio: entity lifecycle event") {
		t.Errorf("handler was not dispatched synchronously on Emit: %s", buf.String())
	}
}

func newReadySystemWithBus(bus *recordingBus) *audio.System {
	return audio.NewSystem(audio.DefaultConfig(), &recordingBackend{}, nil, bus)
}

// ── QA vocabulary: status strings ────────────────────────────────────────────

// TestStatus_String_ReportsReadyAndDisabled covers the QA vocabulary: the
// subsystem reports exactly "ready" and "disabled".
func TestStatus_String_ReportsReadyAndDisabled(t *testing.T) {
	if got := audio.StatusReady.String(); got != "ready" {
		t.Errorf("StatusReady.String() = %q, want \"ready\"", got)
	}
	if got := audio.StatusDisabled.String(); got != "disabled" {
		t.Errorf("StatusDisabled.String() = %q, want \"disabled\"", got)
	}
}

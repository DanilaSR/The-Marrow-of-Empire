# ADR-0014: Audio System Architecture

## Status
Accepted

## Date
2026-08-12
**Accepted**: 2026-08-12 — after /architecture-review verification

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Audio |
| **Knowledge Risk** | HIGH — custom engine, no LLM training data |
| **References Consulted** | `docs/engine-reference/kaiju/VERSION.md`; `kaiju/src/platform/audio/` (SoLoud backend — `soloud.go`, `soloud.c.go`, `soloud_c.h`); `kaiju/src/engine/host.go` (audio initialization); `design/gdd/audio-system.md`; `design/gdd/living-environment.md`; `design/gdd/tactical-combat-engine.md` |
| **Post-Cutoff APIs Used** | Kaiju SoLoud wrapper (`kaiju/src/platform/audio/` — `audio.New()`, `host.InitializeAudio()`, `host.Audio()`; exported: PlaySound, PlayMusic, SetSoundVolume, SetMusicVolume, MuteSounds, MuteMusic, UnmuteSounds, UnmuteMusic, StopAll); C-layer SoLoud API (`soloud_c.h` — 3D spatial, per-voice volume, fades, filters, buses — fully declared but **unwrapped in Go**) |
| **Verification Required** | Per-voice `setVolume` (implemented but unexported in `soloud.c.go:71` — verify export); `Soloud_fadeVolume` / `Soloud_play3d` / `Soloud_set3dListenerParameters` / Bus CGo wrappers (not yet written — feasibility spike); ALSA dev headers on Linux build hosts; `Wav` decode memory (PCM, not compressed — re-estimate budgets); `WavStream` availability (true streaming, unwrapped in Go) |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Framework — Audio component on entities); ADR-0003 (Camera System — camera position = spatial audio listener); ADR-0007 (Game Clock — time/light state drives ambient layer); ADR-0011 (Event Bus — SFX and music events) |
| **Enables** | Combat UI (combat SFX), Dialogue UI (UI SFX), Investigation Journal (journal SFX), Main Menu (menu audio) |
| **Blocks** | All Presentation-layer systems that emit audio events |
| **Ordering Note** | Must be Accepted before any system that triggers audio events (Combat UI, Dialogue UI, Investigation Journal, Main Menu). Ambient layer can be implemented independently after ADR-0007. |

## Context

### Problem Statement
The Audio System has no architectural specification. Eight technical requirements (TR-audio-001 through TR-audio-008) are uncovered by any ADR. The system must provide a 4-layer audio mixer (ambient, music, SFX, voice), spatial audio tied to the camera position, cross-system audio event contracts with 6 other systems, and integration with Kaiju Engine's audio subsystem — which has a known gap: the Go wrapper does not expose the C-layer 3D audio API.

### Constraints
- Kaiju Engine audio backend is **SoLoud** (`kaiju/src/platform/audio/`) — confirmed via prebuilt `libsoloud_*.a` and `soloud_c.h`
- Go wrapper exposes only 2 buses (SFX/BGM), basic play/stop, and mute/unmute — **no per-voice volume, no fades, no 3D spatial, no filters, no buses** in Go
- Full SoLoud C API is declared in `soloud_c.h` but unwrapped — CGo extension is required for the 4-layer mixer, crossfades, and spatial audio (following existing `soloud.c.go` patterns)
- C-layer 3D audio exists in the header (`Soloud_play3d`, `Soloud_set3dListenerParameters`) and is not exposed to Go
- Voice layer is deferred (text-only at MVP/VS)
- Audio must not block the render loop — SoLoud runs its own mixing thread, all play/stop/volume calls are thread-safe and non-blocking. **However**, `LoadMusic`/`LoadSound` are synchronous (decode full file on calling thread) — game must wrap loads in a worker goroutine
- Must function without Steam (offline/GOG builds)
- Audio files ship with the game (no streaming/download)
- Linux builds require ALSA dev headers (`-lasound` in CGo flags)

### Requirements
- 4-layer mixer: ambient, music, SFX, voice (voice deferred) with per-layer volume control
- Ambient layer driven by Living Environment time-of-day + light state + region; 5-second crossfades between states
- Adaptive music: exploration / combat / revelation cues; 1-second transition on combat start/end
- Spatial audio: camera position = listener; positional attenuation; off-screen directional cues
- SFX event contracts: combat impacts (weapon-type-specific), UI feedback (parchment, quill, clicks), footstep surface types, environmental sounds
- Ducking: ambient mutes on overlay open (journal); pause applies low-pass filter
- Audio device selection (deferred engine work)
- Mute-when-unfocused setting

## Decision

**Hybrid event-driven architecture with a 4-layer mixer. Ambient is clock-driven via Living Environment subscription. SFX is event-driven via ADR-0011 EventBus. Music is state-machine-driven, subscribing to combat state and narrative/revelation events. Spatial audio is deferred behind a Go interface (`Spatializer`) that wraps the C-layer when available, with a stereo-only fallback for MVP.**

**Engine reality note**: Kaiju's Go audio wrapper (`kaiju/src/platform/audio/`) exposes only 2 volume buses (SFX/BGM) and basic play/stop/mute. The C-layer (`soloud_c.h`) has full SoLoud capabilities — per-voice volume, fades, 3D spatial, filters, buses — but these are unwrapped in Go. The 4-layer mixer, crossfades, and spatial audio require a small CGo wrapper extension (~2-6 lines per capability, following the existing `soloud.c.go` pattern). The feasibility spike (migration step 6) validates and implements these wrappers. Per-voice `setVolume` is already implemented but unexported in `soloud.c.go:71` — export is a 1-line change.

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     AUDIO SYSTEM                             │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │ AMBIENT  │  │  MUSIC   │  │   SFX    │  │   VOICE    │  │
│  │  Layer   │  │  Layer   │  │  Layer   │  │ (deferred) │  │
│  │          │  │          │  │          │  │            │  │
│  │ Clock-   │  │ State-   │  │ Event-   │  │ Future:    │  │
│  │ driven   │  │ machine  │  │ driven   │  │ TTS/VA     │  │
│  │ (LE sub) │  │ (CBT+Rev)│  │ (EvtBus) │  │            │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘  │
│       │             │             │               │          │
│  ┌────┴─────────────┴─────────────┴───────────────┴────┐     │
│  │                   MIXER BUS                          │     │
│  │  master_volume × per_layer_volume × instance_volume  │     │
│  └──────────────────────┬──────────────────────────────┘     │
│                         │                                    │
│  ┌──────────────────────┴──────────────────────────────┐     │
│  │              OUTPUT PIPELINE                         │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │     │
│  │  │Spatializer│  │ Ducking  │  │ Device Selection │   │     │
│  │  │(interface)│  │  Filter  │  │   (deferred)     │   │     │
│  │  └──────────┘  └──────────┘  └──────────────────┘   │     │
│  └──────────────────────┬──────────────────────────────┘     │
│                         │                                    │
│                  Kaiju Engine Audio                          │
│                  (kaiju/src/engine/)                         │
└─────────────────────────────────────────────────────────────┘

DRIVERS (external to Audio System):
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│Living Environ│   │ Tactical     │   │ All Systems  │
│              │   │ Combat Eng.  │   │ (via EventBus)│
│ time_of_day  │   │              │   │              │
│ light_state  │   │combat.started│   │combat hit/dmg│
│ region_id    │   │combat.ended  │   │ui interaction│
│              │   │turn_changed  │   │thread reveal │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       ▼                  ▼                  ▼
  Audio.Ambient     Audio.Music        Audio.SFX
  .Update(time,     .OnCombatStart()   .OnEvent(event)
   light, region)   .OnCombatEnd()     (EventBus subscriber)
                    .OnRevelation()
```

### Key Interfaces

```go
// AudioSystem is the top-level audio manager. One instance, owned by bootstrap.
type AudioSystem interface {
    // Layer volume controls (0.0–1.0, persisted to settings.cfg)
    SetMasterVolume(v float32)
    SetLayerVolume(layer Layer, v float32)

    // Spatial audio listener position — updated each frame from Camera System
    SetListenerPosition(x, y, z float32)
    SetListenerOrientation(forward, up [3]float32)

    // Ducking control (called by UI overlay systems)
    DuckAmbient()
    RestoreAmbient()
    SetLowPassFilter(enabled bool) // pause low-pass

    // Mute-when-unfocused
    SetMuted(muted bool) // called when game window loses/gains focus

    // Lifecycle
    Update(dt float32) // called each frame — non-blocking
    Shutdown()
}

// Layer identifies the four mixer layers.
type Layer int
const (
    LayerAmbient Layer = iota
    LayerMusic
    LayerSFX
    LayerVoice // deferred
)

// AmbientController drives the ambient layer from Living Environment state.
type AmbientController interface {
    // Update is called each frame with current game state.
    // Returns immediately; crossfade happens asynchronously.
    Update(gameTime float64, lightState LightState, regionID string)
}

// MusicController manages the adaptive music state machine.
type MusicController interface {
    OnCombatStarted()   // 1s transition to combat music
    OnCombatEnded()     // 1s transition back to exploration
    OnRevelationReady() // transition to revelation cue
    Update(dt float32)  // per-frame fade progress
}

// SFXController receives events from the EventBus and plays one-shot sounds.
type SFXController interface {
    // OnEvent is the EventBus handler (registered for all SFX-relevant events).
    OnEvent(event events.Event)
    // PlayFootstep handles footstep sounds (too frequent for EventBus — direct call).
    PlayFootstep(surfaceType string, position [3]float32)
}

// Spatializer wraps spatial audio computation.
// Implementation is a deferred engine integration point.
type Spatializer interface {
    // ComputeGain returns the gain factor for a sound at worldPosition
    // relative to the current listener position.
    // Range: 0.0 (silent) to 1.0 (at listener).
    ComputeGain(worldPosition [3]float32) float32

    // ComputePan returns left/right pan (-1.0 = full left, 1.0 = full right).
    ComputePan(worldPosition [3]float32) float32
}

// SoundID is a hashed identifier for a sound asset.
// Sounds are pre-registered at init; runtime references use SoundID.
type SoundID uint32
```

**Spatializer implementations:**

```go
// StereoSpatializer is the MVP fallback — simple distance-based attenuation,
// no HRTF or 3D positioning. Uses inverse-square attenuation.
type StereoSpatializer struct {
    listenerPos [3]float32
    minDistance float32 // below this = full volume
    maxDistance float32 // beyond this = silent
}

// KaijuSpatializer wraps Kaiju's C-layer 3D audio API when available.
// Deferred to engine feasibility spike. The StereoSpatializer is used
// until KaijuSpatializer passes integration tests.
type KaijuSpatializer struct {
    // Calls kaiju/src/engine/ C audio functions via CGo.
    // Implementation deferred — blocked on engine verification.
}
```

### EventBus Integration

The SFXController subscribes to these existing ADR-0011 event types:

| Event Type | Audio Response |
|------------|---------------|
| `combat.started` | MusicController.OnCombatStarted() + combat-start fanfare SFX |
| `combat.ended` | MusicController.OnCombatEnded() + victory/defeat music |
| `combat.turn_changed` | Turn-change tick SFX |
| `trust.changed` | Soft chime SFX on trust increase |
| `thread.revelation_ready` | MusicController.OnRevelationReady() + revelation sting |
| `entity.destroyed` | Death sound for combatant entities |
| `world.flag_set` | Environmental SFX triggers (door, gate, etc.) |

New event types proposed for ADR-0011 extension (or used as internal dispatch within the SFX controller from higher-level events):

| Proposed Event | Purpose | Emitted By |
|---------------|---------|------------|
| `audio.sfx_oneshot` | Generic one-shot SFX with SoundID + world position | Any system (via EventBus) |

Combat-specific audio (weapon impacts, critical hits) is driven by the Combat UI's rendering of damage events — the TCE emits `combat.turn_changed` and the Combat UI translates the resolved action into the appropriate SFX event. This keeps TCE audio-agnostic.

UI feedback audio (parchment unfurl, quill scratch, clicks) is triggered directly by the UI systems via `audio.sfx_oneshot` events, not through a separate contract per sound.

### Ambient Layer Detail

The AmbientController maintains a crossfade state machine:

```
State: current_ambient (sound_id, volume)
Target: next_ambient (sound_id, volume)
Transition: lerp(current_gain, target_gain, t) over crossfade_duration (5s default)
```

Transitions are triggered when:
- `lightState` changes (Dawn→Day, Day→Dusk, etc.)
- `regionID` changes (region transition)
- Overlay open/close (DuckAmbient/RestoreAmbient — 500ms, not 5s)

The living environment's TR-env-013 specifies ambient audio driven by clock/light state. The ambient controller subscribes to the Living Environment's exported `GetGameTime()` and `GetLightState()` read-only queries — no new EventBus event type is needed for ambient transitions because they are continuous (driven by frame delta, not discrete events).

### Music Layer Detail

The MusicController state machine:

```
States:
  EXPLORATION — calm, melancholic exploration music (per-region)
  COMBAT       — tense, driving, percussive
  REVELATION   — sparse, luminous (State 4 moments)
  SILENT       — no music (main menu, loading)

Transitions:
  EXPLORATION → COMBAT       on combat.started (1s crossfade)
  COMBAT       → EXPLORATION on combat.ended (1s crossfade)
  EXPLORATION → REVELATION   on thread.revelation_ready (1s crossfade)
  REVELATION  → EXPLORATION  on player acknowledgment (1s crossfade)
  ANY          → SILENT       on pause / main menu
  SILENT       → EXPLORATION  on resume / game start

Combat-at-night rule: music stays in COMBAT state while combat is active
regardless of time-of-day ambient changes. Post-combat, music returns to
EXPLORATION with the region-appropriate night variant.
```

## Alternatives Considered

### Alternative 1: Pure Event-Bus for Everything (including ambient)
- **Description**: All audio triggers — including ambient transitions, footstep sounds, and UI clicks — go through ADR-0011 EventBus events.
- **Pros**: Maximum consistency with ADR-0011; all audio sources are observable in the event stream.
- **Cons**: Ambient transitions are continuous (frame-by-frame crossfade), not discrete — shoehorning them into events creates a firehose of `ambient.crossfade_tick` events. Footstep events would generate ~4-8 events per second for a moving party — excessive heap allocation in the hot path, violating ADR-0011's performance constraint.
- **Rejection Reason**: Over-use of EventBus for continuous/high-frequency audio degrades into an event firehose. Direct subscription to Living Environment state for ambient, and direct calls for footsteps, are the appropriate patterns for these high-frequency or continuous audio needs.

### Alternative 2: Direct-Call Only (no EventBus)
- **Description**: Every system calls `AudioSystem.Play(sound_id, position)` directly. No EventBus integration.
- **Pros**: Simplest to understand; no event subscription boilerplate; lowest latency path.
- **Cons**: Every audio-triggering system must import and depend on the Audio System package. Combat UI, Dialogue UI, Investigation Journal, and Main Menu would all have direct Audio dependencies — adding ~4 cross-system couplings. Adding a new audio trigger requires modifying the triggering system.
- **Rejection Reason**: Violates the downward-only dependency rule (Presentation→Audio is acceptable since both are Presentation layer, but audio triggers from Feature-layer systems like Thread Investigation would invert the dependency). The EventBus already exists (ADR-0011) and is the established cross-system communication pattern.

### Alternative 3: Separate Audio Event Bus
- **Description**: Audio has its own event bus, separate from ADR-0011's general-purpose EventBus.
- **Pros**: Audio-specific event types don't pollute the general event namespace; audio events can be batched/coalesced independently.
- **Cons**: Two event buses = two dispatch mechanisms = two registrations for systems that emit both game events and audio events. Architectural fragmentation without clear benefit — the existing EventBus (ADR-0011) already supports priority ordering and has performance headroom.
- **Rejection Reason**: ADR-0011's EventBus is designed for exactly this use case. A second bus adds complexity without solving a problem the existing bus cannot handle.

## Consequences

### Positive
- Cross-system audio triggers use the established EventBus pattern (ADR-0011) — consistent with other cross-system communication
- Ambient and music layers are decoupled from the systems they listen to (Living Environment, Combat, Thread Investigation) — they read state or subscribe to events rather than being called directly
- Spatializer interface allows the C-layer integration to be deferred without blocking MVP audio
- Footstep sounds use a direct-call optimization (too frequent for EventBus) with explicit justification
- Each layer (ambient, music, SFX) can be implemented and tested independently

### Negative
- Spatial audio at MVP is stereo-only (no HRTF, no true 3D positioning) — off-screen directional cues are pan-based, not spatialized
- Audio device selection is deferred engine work (greyed out at MVP)
- Pause low-pass filter is deferred engine work (SoLoud filter wrapper)
- The `Spatializer` interface adds an abstraction layer that will need to be validated against the actual Kaiju C API when it becomes available
- New `audio.sfx_oneshot` event type must be added to ADR-0011's event catalog

### Risks
- **Kaiju Go wrapper gap**: Only 2 buses and basic play/stop are exposed. Per-voice volume, fades, 3D spatial, filters, and buses require CGo wrappers. Mitigation: per-voice `setVolume` is already implemented (unexported, 1-line export); fade, 3D, and filter bindings follow the same pattern (~2-6 lines each); the feasibility spike validates all needed wrappers before any audio layer is built.
- **Memory budget for decoded audio**: `Wav` decodes fully to PCM (~134MB at MVP for uncompressed audio). Mitigation: evaluate `WavStream` (true streaming, declared in C header but unwrapped) in the feasibility spike. If viable, music+ambient memory drops to near-zero (streamed from disk). Fallback: reduce music track length to 2 min, mono ambient, or cap at 1 region at MVP.
- **Spatializer interface shape**: When the C-layer API is exposed, the `Spatializer` interface may not map cleanly. Mitigation: the interface is intentionally minimal (ComputeGain, ComputePan). The spike should also evaluate using SoLoud's native `play3d` + listener bindings as an alternative to the hand-rolled `StereoSpatializer` — native SoLoud 3D may be simpler and more correct.
- **Sync audio loading**: `LoadMusic`/`LoadSound` decode synchronously on the calling thread. Mitigation: game wraps all audio loading in worker goroutines; loading screen masks load time.
- **SFX handle leak**: `AudioClip.handles` grows unbounded in the engine wrapper (`soloud.go:146-158` appends every play, only `StopSource` clears). Mitigation: the game's SFXController tracks handles and periodically prunes completed voices (SoLoud returns `InvalidVoiceHandle` for finished sounds).
- **Audio device selection**: The C API exposes backend getters but not full per-device enumeration. Mitigation: device selection deferred to VS; the C++-level SoLoud enum API may need new C glue beyond the vendored `soloud_c.h`.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| audio-system.md | TR-audio-001: 4-layer mixer with per-layer volume control | Mixer Bus with master_volume × per_layer_volume × instance_volume; Layer enum for ambient/music/SFX/voice |
| audio-system.md | TR-audio-002: Ambient driven by Living Environment time-of-day + light state + region, 5s crossfades | AmbientController.Update(gameTime, lightState, regionID) with crossfade state machine (5s default) |
| audio-system.md | TR-audio-003: Adaptive music state machine (exploration/combat/revelation), 1s transition on combat start/end | MusicController with EXPLORATION/COMBAT/REVELATION/SILENT states; 1s crossfade transitions |
| audio-system.md | TR-audio-004: Spatial audio with camera = listener, positional attenuation, off-screen directional cues | Spatializer interface; StereoSpatializer (MVP) with inverse-square attenuation; KaijuSpatializer (deferred) |
| audio-system.md | TR-audio-005: SFX event contracts — combat impacts, UI feedback, footstep types, environmental | SFXController subscribes to EventBus; PlayFootstep direct-call for high-frequency sounds; audio.sfx_oneshot event type |
| audio-system.md | TR-audio-006: Ducking — ambient mutes on overlay open, pause low-pass filter | DuckAmbient/RestoreAmbient methods; SetLowPassFilter (deferred SoLoud wrapper) |
| audio-system.md | TR-audio-007: Audio output device selection | Deferred — StereoSpatializer uses default device; KaijuSpatializer will expose device enumeration |
| audio-system.md | TR-audio-008: Mute-when-unfocused | SetMuted(bool) — called when game window loses/gains focus |

## Performance Implications

- **CPU**: Ambient crossfade update < 0.1ms/frame (2 lerp calculations). SFX event dispatch < 0.1ms per event (1-3 events/frame typical, ~10 events/frame during combat). Music state machine < 0.01ms/frame. Total audio CPU budget: < 0.5ms/frame.
- **Memory**: SoLoud `Wav` decodes fully to PCM in RAM (not streamed — `WavStream` exists in C header but is unwrapped). Ambient loops: ~42MB (2 min × 2 files × 2 regions at 44.1kHz 16-bit stereo). Music tracks: ~84MB (3 states × 4 min × 2 regions). SFX banks: ~8MB (100 sounds × ~80KB each, 44.1kHz 16-bit mono). **Total audio memory: ~134MB at MVP — must revisit if `WavStream` wrapping is feasible.** Compressed OGG sizes (~25MB) do NOT reflect runtime memory; `Wav` decodes on load. Mitigation: the feasibility spike evaluates `WavStream` wrapping for music/ambient (true streaming, no decode-to-RAM); if viable, music+ambient drop to near-zero additional memory.
- **Load Time**: Audio assets loaded asynchronously at scene load; music streams, SFX pre-loaded. Audio load < 2s for a region's full audio set.
- **Network**: None — all audio is local.

## Migration Plan
1. Implement `StereoSpatializer` (MVP — stereo-only, inverse-square attenuation + pan; no engine dependency beyond basic play)
2. Implement 4-layer mixer with volume controls (game-side handle tracking with manual multiply chain until per-voice volume CGo wrapper is exported)
3. Implement AmbientController (integrates with Living Environment read-only queries; game-side lerp for crossfade until `Soloud_fadeVolume` wrapper is added)
4. Implement MusicController state machine (subscribes to EventBus combat/revelation events)
5. Implement SFXController (subscribes to EventBus; add `audio.sfx_oneshot` event type to ADR-0011; reuse engine's existing `MuteSounds`/`UnmuteSounds` for ducking)
6. **Audio feasibility spike** — validate and implement CGo wrappers:
   - Export per-voice `setVolume` (already implemented, unexported) — 1 line
   - Wrap `Soloud_fadeVolume` for native crossfades — ~3 lines
   - Evaluate `Soloud_play3d` + `Soloud_set3dListenerParameters` vs hand-rolled StereoSpatializer
   - Wrap `BiquadResonantFilter` + `Soloud_setFilterParameter` for pause low-pass — ~5 lines
   - Evaluate `WavStream` wrapping for music/ambient streaming (memory budget decision)
   - Verify HRTF init flag (`Soloud_initEx` vs current `Soloud_init`)
   - Measure `Wav` decode memory for representative music/ambient tracks
7. When spike confirms viability: implement `KaijuSpatializer`, swap in for `StereoSpatializer` with no game-code changes
8. Deferred engine work (VS): pause low-pass filter (if spike didn't complete it), device enumeration (requires new C glue beyond vendored header), `WavStream` if spike deferred it

## Validation Criteria
- Mixer: 4 independent layers, each volume-controllable, output to Kaiju audio
- Ambient: Dawn→Day transition crossfades birdsong→day ambience over 5s ± 0.5s
- Music: Combat start triggers 1s transition to combat music; combat end returns within 1s
- SFX: Critical hit → heavy impact + chime plays within 50ms of event dispatch
- Spatial: Sound at 20u from listener is quieter than sound at 5u (StereoSpatializer)
- Ducking: Journal open → ambient muted within 500ms; journal close → restored
- Performance: Audio.Update() < 0.5ms/frame at p95 with combat active (10 SFX events)
- Non-blocking: Audio.Update() returns in < 0.5ms; no audio path blocks the render loop

## Related Decisions
- ADR-0001: Entity Framework — Audio component on entities
- ADR-0003: Camera System — camera position = spatial audio listener
- ADR-0007: Game Clock & Environment — time/light states drive ambient audio
- ADR-0011: Event Bus — SFX and music events
- ADR-0012: UI Framework — UI SFX event contracts

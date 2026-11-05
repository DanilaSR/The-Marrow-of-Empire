# Story 001: Audio Engine Foundation & Spatial Listener

> **Epic**: Audio System
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11 (Complete)

## Context

**GDD**: `design/gdd/audio-system.md`
**Requirement**: `TR-audio-001`
*(No entry yet in `docs/architecture/tr-registry.yaml` — run `/architecture-review` to register)*

**ADR Governing Implementation**: N/A — pure Kaiju engine API integration, no architectural decision required. Audio subsystem bootstrap follows the engine's native audio APIs as documented in `kaiju/AGENTS.md`.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju audio subsystem APIs are not in LLM training data. Verify against `kaiju/src/engine/` for: audio device initialization, audio source components, listener attachment, supported formats (WAV/OGG), and volume control APIs. Spatial audio behavior (distance attenuation, panning in isometric view) must be verified against engine source.

**Control Manifest Rules (Presentation Layer)**:
- Required: Kaiju HTML/CSS-like UI — all UI rendered in screen space (ADR-0012 — audio UI controls)
- Forbidden: N/A for audio subsystem
- Guardrail: UI render (static): <1ms/frame (ADR-0012 — applies to any audio UI overlay)

---

## Acceptance Criteria

*Derived from GDD `design/gdd/audio-system.md` Overview — infrastructure prerequisite for all audio layers:*

- [ ] Audio subsystem initializes on game launch without errors
- [ ] Spatial audio listener is attached to the isometric camera and updates position each frame
- [ ] Audio assets (WAV/OGG) can be loaded from `assets/audio/` via the engine's asset pipeline
- [ ] Master volume and mute controls function correctly
- [ ] Audio device gracefully handles missing hardware (no crash, audio disabled with log warning)

---

## Implementation Notes

*Derived from Kaiju Engine reference and architecture:*

1. **Audio bootstrap**: Initialize the Kaiju audio device during game startup (after renderer, before game logic). Follow the pattern in `kaiju/src/engine/` for audio subsystem initialization.
2. **Listener attachment**: The isometric camera (ADR-0003: 30° pitch, 45° yaw, 0° roll) is the audio listener. Update listener transform each frame from the camera's matrix Transform. The listener's position in world space determines spatialization.
3. **Asset pipeline**: Audio assets follow the same import pipeline pattern as visual assets (ADR-0004). Store in `assets/audio/` with subdirectories: `sfx/`, `music/`, `ambient/`. Supported formats: WAV (uncompressed SFX), OGG (compressed music/ambient).
4. **Volume control**: Master volume as float 0.0–1.0, stored in game settings. Mute is a boolean flag. Volume changes apply to all audio buses.
5. **Error handling**: If audio device fails to initialize (no hardware, driver issue), log a warning and continue with audio disabled. The game must not crash. Systems that trigger sounds check the disabled flag and skip playback.
6. **Event bus integration**: Audio system subscribes to relevant events via EventBus (ADR-0011) but does not emit audio-specific events at this layer. Foundation story establishes the subscriber pattern.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Ambient soundscapes (time-of-day driven, region-based)
- Story 003: Adaptive music system (exploration/combat/revelation states)
- Story 004: Combat SFX, UI feedback sounds, footstep types

---

## QA Test Cases

*Written at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1**: Audio subsystem initializes on game launch without errors
  - Given: Fresh game launch on a system with audio hardware
  - When: Game bootstrap completes
  - Then: Audio device handle is non-nil, no error logged, audio subsystem reports status "ready"
  - Edge cases: Launch on system without audio hardware → audio disabled, game continues, warning logged

- **AC-2**: Spatial audio listener attached to camera and updates each frame
  - Given: Game running, camera at world position (100, 0, 50)
  - When: Camera pans to (200, 0, 50) over 10 frames
  - Then: Audio listener position tracks camera transform within 1 frame of movement
  - Edge cases: Camera zoom changes (4u/10u/20u) — listener position unchanged (zoom is ortho height, not position)

- **AC-3**: Audio assets loadable from `assets/audio/`
  - Given: A test WAV file at `assets/audio/sfx/test_tone.wav` and test OGG at `assets/audio/music/test_music.ogg`
  - When: Audio system loads each file
  - Then: Both return valid audio buffers, no errors
  - Edge cases: Missing file → logged error, returns nil buffer (no crash); unsupported format → logged error

- **AC-4**: Master volume and mute controls function
  - Given: Audio playing at master volume 0.8
  - When: Set master volume to 0.0, then 0.5, then mute=true
  - Then: Output level reflects each change; mute=true produces silence regardless of volume setting
  - Edge cases: Volume >1.0 → clamped to 1.0; volume <0.0 → clamped to 0.0

- **AC-5**: Missing audio hardware handled gracefully
  - Given: System with no audio output device
  - When: Game launches
  - Then: Game starts normally, audio subsystem status is "disabled", warning logged, no crash
  - Edge cases: Audio device unplugged mid-game → DEFERRED: engine provides no device-loss callback (Kaiju SoLoud wrapper has no hot-unplug hook). System status is immutable after construction. Revisit when engine audio bindings expose device-loss events.

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/audio/audio_foundation_test.go` — must exist and pass

**Status**: [x] Created — `tests/integration/audio/audio_foundation_test.go`, 14 tests, all PASS (2026-08-11)

---

## Dependencies

- Depends on: None (foundation story — no prior audio stories)
- Unlocks: Story 002 (Ambient Soundscapes), Story 003 (Adaptive Music), Story 004 (Combat & UI SFX)

---

## Completion Notes

**Completed**: 2026-08-11
**Criteria**: 5/5 passing (AC-5 "device unplugged mid-game" edge case deferred — engine limitation)
**Deviations**: ADVISORY — AC-5 hot-unplug edge case deferred (Kaiju SoLoud bindings have no device-loss callback). Revisit when engine audio bindings expose device-loss events.
**Test Evidence**: `tests/integration/audio/audio_foundation_test.go` — 14 tests, all PASS (0.009s)
**Code Review**: Complete — APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester, 2026-08-11)

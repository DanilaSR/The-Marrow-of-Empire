# Story 003: Adaptive Music System

> **Epic**: Audio System
> **Status**: Ready
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/audio-system.md`
**Requirement**: `TR-audio-003`
*(No entry yet in `docs/architecture/tr-registry.yaml` — run `/architecture-review` to register)*

**ADR Governing Implementation**: ADR-0009: Tactical Combat Engine (primary), ADR-0011: Event Bus Architecture (secondary)
**ADR Decision Summary**: Combat is turn-based on a square grid with AP system and d20 resolution. Combat start/end is signaled via EventBus events. The music system subscribes to combat state changes and crossfades between exploration, combat, and revelation music layers.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Music playback (long-form streaming OGG) and crossfade between music tracks must be verified against Kaiju audio APIs. Verify: streaming playback support (vs. fully-buffered), track looping with seamless gapless restart, and whether multiple music sources can play simultaneously for crossfading.

**Control Manifest Rules (Presentation Layer)**:
- Required: N/A (audio-specific)
- Forbidden: Never advance time during combat (ADR-0007 — but music transition is triggered by combat state, not clock)
- Guardrail: Combat AI decision: 10-50ms (ADR-0009 — music transition must not add latency to combat start)

---

## Acceptance Criteria

*From GDD `design/gdd/audio-system.md`:*

- [ ] **AC-1**: GIVEN combat start, WHEN the encounter begins, THEN music transitions from exploration to combat over 1 second.
- [ ] **AC-2**: GIVEN combat end, WHEN the encounter concludes (all enemies defeated or party flees), THEN music transitions from combat back to exploration over 2 seconds.
- [ ] **AC-3**: Three music states exist — Exploration (calm, melancholic, warm), Combat (tense, driving, percussive), and Revelation (sparse, luminous) — each with at least one playable track.
- [ ] **AC-4**: Revelation music triggers on State 4 revelation moments (3+ connected facts → `revelation_ready` event from Thread Investigation). Revelation music crossfades in over 3 seconds, plays once (no loop), then returns to exploration.
- [ ] **AC-5**: Exploration music loops seamlessly. Combat music loops while combat is active. If combat music track ends before combat concludes, it restarts from beginning without gap.

---

## Implementation Notes

*Derived from ADR-0009 and ADR-0011 Implementation Guidelines:*

1. **Music state machine**: Three primary states — Exploration, Combat, Revelation. Transitions triggered by EventBus events:
   - `CombatStarted` → Exploration → Combat (1s crossfade)
   - `CombatEnded` → Combat → Exploration (2s crossfade)
   - `RevelationReady` (ADR-0010) → Exploration → Revelation (3s crossfade)
   - `RevelationComplete` → Revelation → Exploration (2s crossfade)
2. **Crossfade implementation**: Similar to ambient crossfade pattern (Story 002). Ramp down current track gain while ramping up next track. Combat transition is faster (1s) for urgency; return to exploration is slower (2s) for relief; revelation is slowest (3s) for gravitas.
3. **Track management**: Music tracks are OGG files in `assets/audio/music/`. Each state can have multiple tracks; one is selected at random (or sequentially) on state entry. Track list per state is defined in `assets/audio/music/music_config.yaml`.
4. **Looping**: Exploration and Combat tracks loop. When a track nears its end (< 1s remaining), restart from beginning with gapless transition. Revelation track plays once and stops (no loop) — silence after revelation until next state change.
5. **Combat edge cases**: If combat starts during a revelation moment, combat music takes priority (revelation is interrupted). If a revelation triggers during combat, it is queued until combat ends.
6. **Event subscription**: Subscribe to EventBus with appropriate priority. Music system priority should be lower than combat system (ADR-0011: priority 1 = highest). Music responds to state changes; it never initiates them.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Audio engine bootstrap, spatial listener (prerequisite)
- Story 002: Ambient soundscapes — runs simultaneously with music, independent layer
- Story 004: Individual combat SFX (impact sounds, critical hit chimes) — SFX layer, not music

---

## QA Test Cases

*Written at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1**: Combat start music transition
  - Given: Exploration music playing at gain 1.0, player initiates combat with hostile entity
  - When: `CombatStarted` event fires on EventBus
  - Then: Exploration music gain ramps 1.0→0.0 over 1 second; combat music gain ramps 0.0→1.0 over 1 second; at t=0.5s both tracks at ~0.5 gain
  - Edge cases: Combat starts during ambient crossfade — music transition is independent of ambient; combat starts before exploration track loaded → skip crossfade, start combat track immediately

- **AC-2**: Combat end music transition
  - Given: Combat music playing, last enemy defeated
  - When: `CombatEnded` event fires
  - Then: Combat music gain ramps 1.0→0.0 over 2 seconds; exploration music gain ramps 0.0→1.0 over 2 seconds
  - Edge cases: Combat ends within 1s of starting → transition still takes full 2s; new combat starts during return transition → abort return, restart combat transition from current gains

- **AC-3**: Three music states with playable tracks
  - Given: Music system initialized with config at `assets/audio/music/music_config.yaml`
  - When: Each state (Exploration, Combat, Revelation) is activated
  - Then: A track from that state's playlist begins playing; if state has multiple tracks, selection varies (random or sequential)
  - Edge cases: State has zero configured tracks → silence, warning logged; track file missing → skip track, try next in playlist, log error

- **AC-4**: Revelation music trigger
  - Given: Exploration music playing, player has 3+ connected facts pointing to same conclusion
  - When: `RevelationReady` event fires (from Thread Investigation, ADR-0010)
  - Then: Exploration music crossfades to revelation track over 3 seconds; revelation track plays once to completion; after track ends, silence until `RevelationComplete` or next state change
  - Edge cases: Revelation triggers during combat → queued, plays after `CombatEnded`; multiple rapid `RevelationReady` events → only first triggers, subsequent ignored until current revelation completes

- **AC-5**: Seamless looping
  - Given: Exploration or Combat track playing, track duration is 180s
  - When: Track reaches 179s (1s remaining)
  - Then: Track restarts from beginning without audible gap or pop; gain remains at 1.0 throughout loop point
  - Edge cases: Very short track (< 5s) → still loops; track fails to loop (decode error) → log error, attempt next track in playlist

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/audio/adaptive_music_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Audio Engine Foundation & Spatial Listener) must be DONE
- Unlocks: None

# Story 004: Combat & UI SFX

> **Epic**: Audio System
> **Status**: Ready
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/audio-system.md`
**Requirement**: `TR-audio-004`
*(No entry yet in `docs/architecture/tr-registry.yaml` — run `/architecture-review` to register)*

**ADR Governing Implementation**: ADR-0009: Tactical Combat Engine (primary), ADR-0012: UI Framework (secondary)
**ADR Decision Summary**: Combat uses d20 attack resolution with critical hits on natural 20. UI uses three registers (Desk/Field/World) with paper-slide transitions and pen-and-ink icons. SFX are triggered by combat events (hit/miss/crit/kill) and UI interactions (open/close/click/typewriter). All SFX are spatialized based on source position relative to the camera listener, except UI sounds which play in screen space.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: One-shot SFX playback, spatialization parameters (distance attenuation, panning), and simultaneous voice count limits must be verified against Kaiju audio APIs. Verify: max concurrent SFX voices, priority/stealing behavior when voice limit reached, and whether screen-space (non-spatialized) audio is supported for UI sounds.

**Control Manifest Rules (Presentation Layer)**:
- Required: Desk never trembles — UI immune to camera shake (ADR-0012 — UI SFX must not spatialize, always play at center)
- Required: Paper-slide motion: enter 200-300ms ease-out, exit 200ms (ADR-0012 — UI SFX timing should align with transitions)
- Forbidden: Never encode game-critical information by hue alone (ADR-0012 — applies to audio too: critical info must have audio + visual backup, never audio-only)
- Guardrail: UI render (static): <1ms/frame (ADR-0012 — SFX playback must not block UI thread)

---

## Acceptance Criteria

*From GDD `design/gdd/audio-system.md`:*

- [ ] **AC-1**: GIVEN a critical hit, WHEN damage is dealt, THEN heavy impact + rising chime plays spatialized at the target's position.
- [ ] **AC-2**: Combat SFX vary by weapon type — at minimum, distinct sounds for: bladed (slash/ring), blunt (thud/crack), and unarmed (impact/breath).
- [ ] **AC-3**: UI interactions produce appropriate feedback sounds: parchment unfurl on journal open, quill scratch on text advance, soft click on button press.
- [ ] **AC-4**: Footstep SFX vary by surface type — at minimum, distinct sounds for: stone, wood, grass, and water. Footstep type is determined by the entity's current ground surface.
- [ ] **AC-5**: All SFX are spatialized (3D positioned) relative to the camera listener, except UI sounds which play in screen space (center, non-spatialized).
- [ ] **AC-6**: SFX voice limit is enforced — when more than N sounds would play simultaneously, lowest-priority sounds are culled (priority: combat > UI > footsteps > ambient one-shots).

---

## Implementation Notes

*Derived from ADR-0009 and ADR-0012 Implementation Guidelines:*

1. **SFX trigger system**: Subscribe to EventBus events and map them to SFX assets via a data-driven lookup table (`assets/audio/sfx/sfx_config.yaml`):
   - `AttackHit(attacker, target, weaponType, isCrit)` → weapon-type-specific impact + crit chime if isCrit
   - `AttackMiss(attacker, target)` → whoosh/deflect based on weapon
   - `EntityDeath(entity)` → death sound
   - `UIOpen(screen)` → parchment unfurl
   - `UIClose(screen)` → soft close
   - `UIButtonClick(button)` → soft click
   - `DialogueAdvance()` → quill scratch
   - `EntityMove(entity, fromTile, toTile)` → footstep based on ground surface
2. **Weapon type → SFX mapping**: Defined in config, not hardcoded. Categories: bladed (slash on hit, ring on parry), blunt (thud on hit, crack on crit), unarmed (impact, breath), ranged (release twang, impact thud). Each weapon type has: hit sound, crit sound, miss sound.
3. **Surface type → footstep mapping**: Ground surface is queried from the spatial index or tile data. Surfaces: stone (sharp click), wood (hollow thud), grass (soft crunch), water (splash). Entity's current tile determines surface.
4. **Spatialization**: SFX play at the source entity's world position. Kaiju audio engine handles distance attenuation and stereo panning based on listener (camera) position. UI sounds are exceptional — they play with spatialization disabled (screen space, full center).
5. **Voice management**: Configurable max concurrent voices (default 32). When exceeded, cull by priority: Combat (prio 1) > UI (prio 2) > Footsteps (prio 3) > Ambient one-shots (prio 4). Within same priority, cull oldest first. Track active voice count in SFX system.
6. **Performance**: SFX trigger → playback path must be <0.5ms (non-blocking). Audio decoding happens on load, not on play. All SFX assets are pre-loaded as short WAV buffers (no streaming for SFX).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Audio engine bootstrap, spatial listener (prerequisite)
- Story 002: Ambient soundscapes (continuous, not one-shot SFX)
- Story 003: Adaptive music system (music, not SFX)
- Environmental one-shots (door creak, lantern light, fire crackle) — these are ambient one-shots, part of Story 002's region layer or a future "Environmental SFX" story
- Voice acting / TTS — explicitly deferred per GDD

---

## QA Test Cases

*Written at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1**: Critical hit SFX
  - Given: Combat active, attacker rolls natural 20 against target entity at position (50, 0, 30)
  - When: Damage is dealt (crit confirmed)
  - Then: Heavy impact SFX plays spatialized at target position; rising chime SFX plays spatialized at target position; both audible with correct stereo panning relative to camera
  - Edge cases: Camera far from target (zoom 20u) → sounds attenuated but still audible; multiple simultaneous crits → all play, voice limit enforced if needed

- **AC-2**: Weapon-type-specific combat SFX
  - Given: Three attackers with bladed, blunt, and unarmed weapons
  - When: Each lands a hit on their target
  - Then: Bladed hit produces slash/ring sound; blunt hit produces thud/crack; unarmed produces body impact; sounds are audibly distinct
  - Edge cases: Unknown weapon type → fallback to default "generic hit" sound; weapon type not in config → log warning, use fallback

- **AC-3**: UI feedback sounds
  - Given: Game running, player in exploration mode
  - When: Player opens journal (parchment unfurl), advances dialogue (quill scratch), clicks a button (soft click), closes journal (soft close)
  - Then: Each UI interaction plays its corresponding SFX in screen space (center, non-spatialized)
  - Edge cases: Rapid UI actions (button spam) → each click plays, voice limit may cull oldest UI sounds; UI open during combat → UI SFX still screen-space, combat SFX still spatialized

- **AC-4**: Footstep surface types
  - Given: Entity moving across tiles with surfaces: stone → wood → grass → water
  - When: Entity steps on each surface type
  - Then: Stone produces sharp click; wood produces hollow thud; grass produces soft crunch; water produces splash; each footstep spatialized at entity position
  - Edge cases: Entity teleports (no intermediate steps) → no footsteps for skipped tiles; entity moves diagonally → one footstep per tile entered, not per animation frame

- **AC-5**: Spatialization of SFX vs UI sounds
  - Given: Combat SFX playing at world position (100, 0, 50), UI open with parchment sound
  - When: Camera is at (0, 0, 0) — SFX source is 100 units right of camera
  - Then: Combat SFX is louder in right speaker; UI parchment sound is centered (equal L/R); moving camera changes combat SFX panning but never affects UI panning
  - Edge cases: SFX source at exact camera position → centered; SFX source behind camera → audible but attenuated (off-screen directional cue)

- **AC-6**: Voice limit enforcement
  - Given: Voice limit set to 8, 12 simultaneous SFX requested (5 combat, 3 UI, 4 footsteps)
  - When: All 12 attempt to play
  - Then: Only 8 play; culled sounds are lowest priority (footsteps culled first, then UI if needed); combat sounds never culled while footsteps remain
  - Edge cases: All sounds same priority → oldest culled first; voice limit set to 0 → all SFX disabled; voice limit > hardware capability → hardware handles, engine reports actual max

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/audio/sfx_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Audio Engine Foundation & Spatial Listener) must be DONE
- Unlocks: None

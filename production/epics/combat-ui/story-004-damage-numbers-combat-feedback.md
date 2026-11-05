# Story 004: Damage Numbers & Combat Feedback

> **Epic**: Combat UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: Visual/Feel
> **Estimate**: 3h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/combat-ui.md`
**Requirement**: `TR-ui-002`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: World register elements (damage numbers) are anchored in world space. Motion language: paper slides, ink appears — but damage numbers float upward (a necessary exception approved by creative-director for combat readability). The desk never trembles — but the world shakes on critical hits (screen shake affects world-space rendering, not UI panels). Amber is accent only. All colors have shape/icon/value/sound backup.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju world-space UI anchoring for floating text, CSS animation for float-up + fade, screen shake API (verify against `kaiju/src/` — may need camera offset manipulation). Audio event routing to Audio System. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Combat face (grotesque sans, tabular figures) for damage numbers; world register for anchored elements; amber accent only; desk never trembles (UI panels immune to camera shake)
- Forbidden: Never encode game-critical information by hue alone — damage numbers pair color with size/position/text; never use bounce for damage numbers (float-up is a translate, not a bounce arc)
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations; screen shake must not affect Field register elements

---

## Acceptance Criteria

*From GDD `design/gdd/combat-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a combatant takes damage, WHEN the hit lands, THEN an oxblood damage number floats upward from the hit location (ascending ~30px over 800ms, fading out) showing the damage amount in combat face tabular figures.
- [ ] **AC-2**: GIVEN an attack misses, WHEN the miss is resolved, THEN a grey "Miss" text floats up from the target with the same float-up/fade animation as damage numbers.
- [ ] **AC-3**: GIVEN a critical hit, WHEN the attack roll is a natural 20, THEN the damage number appears larger (1.5× scale), an amber flash overlays the target for 150ms, the screen shakes (2 units displacement, 100ms), and a heavy impact + chime sound plays.
- [ ] **AC-4**: GIVEN combat audio events, WHEN turn changes, AP is spent, attacks are selected, hits land, or enemies die, THEN the corresponding combat audio plays: soft tick (turn change), subtle click (per AP spent), soft select (attack chosen), impact sound type-specific (hit), death sound (enemy death).
- [ ] **AC-5**: GIVEN the Field register combat HUD elements (turn order bar, AP meter, ability menu, End Turn), WHEN screen shake occurs (crit), THEN the Field register UI panels do not shake — only the world-space elements (health bars, damage numbers) and world rendering are affected.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Damage number element**: World-space anchored text element, positioned at the hit location (or slightly above — the target's "chest" or center point). On creation: float upward via CSS `transform: translateY(0)` → `translateY(-30px)` over 800ms, with `opacity: 1` → `opacity: 0` over the last 400ms (fade out during ascent). Use combat face (grotesque sans, tabular figures) for the number. Font size: 18px at reference for normal hits, 27px (1.5×) for crits.
- **Damage number color**: 
  - Normal hit: oxblood (#6B2020 or art-bible-defined oxblood).
  - Critical hit: oxblood but 1.5× larger + amber flash on target (see below).
  - Miss: grey (#888888), text "Miss" instead of a number.
  - Healing (if applicable): soft green — but combat GDD doesn't mention healing numbers; implement as configurable for future use.
- **Float-up lifecycle**: Create element on `DamageDealt`/`AttackMissed` event → append to world overlay container → CSS animation runs → on `animationend` event, remove the element from DOM. Multiple simultaneous hits: each creates its own damage number element, slightly offset horizontally (stagger by ±5-10px) so overlapping numbers are readable.
- **Critical hit effects** (natural 20 per TR-combat-003):
  1. Damage number: 1.5× font size (27px vs 18px at reference).
  2. Amber flash: overlay a semi-transparent amber (#D4A040, opacity 0.3) circle/glow on the target's position for 150ms, then fade. Use CSS `@keyframes` — appear (opacity 0→0.3 over 50ms), hold (50ms), fade (opacity 0.3→0 over 50ms).
  3. Screen shake: displace the camera by 2 units (2 meters in world space — per the GDD "2 units") in a rapid damped oscillation over 100ms. Use the camera system's shake API if available; otherwise, apply a sine-wave offset to the camera position that decays to 0 over 100ms. **Critical**: Field register UI (panels, bars, buttons) must NOT shake — only the world-space rendering and world-anchored elements are affected. This means the camera shake offset applies to the world render transform but NOT to the screen-space UI transform.
  4. Audio: Emit `CritLanded` event → Audio System plays heavy impact sound + chime. These are two layered sounds; the Audio System handles mixing.
- **Combat audio events**: Map combat actions to audio events via EventBus:
  | Game Event | Audio Event | Description |
  |------------|-------------|-------------|
  | `TurnChanged` | `SFX_Combat_Tick` | Soft tick |
  | `APChanged` (decrease) | `SFX_Combat_APClick` | Subtle click per AP |
  | `AbilitySelected` | `SFX_Combat_Select` | Soft select sound |
  | `DamageDealt` (normal) | `SFX_Combat_Impact_{type}` | Type-specific impact (slash/blunt/pierce/magic) |
  | `DamageDealt` (crit) | `SFX_Combat_CritImpact` + `SFX_Combat_Chime` | Heavy impact + chime |
  | `AttackMissed` | `SFX_Combat_Miss` | Whoosh or soft thud |
  | `EntityDied` | `SFX_Combat_Death` | Death sound |
  Audio events are consumed by the Audio System — this story only emits the events. Sound file assignment and mixing are the Audio System's responsibility.
- **Screen shake immunity**: Field register elements (turn order bar, AP meter, ability menu, End Turn button) must be immune to screen shake. In Kaiju's rendering: screen-space UI elements are rendered in a separate pass or layer that does not receive the camera shake transform. Verify this separation in `kaiju/src/` — if the engine doesn't natively separate layers, apply the inverse shake transform to Field register elements. Desk register elements (menus, journal — not part of combat UI) are also immune per ADR-0012 ("the desk never trembles").

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Turn order bar, combat start/end animations, Field register layout
- Story 002: AP meter, ability menu, End Turn button
- Story 003: World-anchored health bars (damage numbers appear near but are independent of health bars)
- Story 005: Targeting reticle, range line, flanking/cover indicators
- Story 006: Victory fanfare (different from combat hit sounds — victory audio is part of resolution screens)
- Audio file creation: Audio System handles sound file authoring, mixing, and playback — this story only emits the events

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (damage number floats up on hit):
  - Setup: In combat, land a normal hit dealing 12 damage to an enemy
  - Verify: "12" appears in oxblood at the hit location, floats upward ~30px over ~800ms, fades out during last 400ms. Text is in combat face tabular figures. Element is removed after animation completes.
  - Pass condition: Number correct, color oxblood, float-up + fade animation smooth, element cleaned up

- **AC-2** (miss text floats up):
  - Setup: In combat, make an attack that misses (roll < evasion)
  - Verify: Grey "Miss" text appears at the target, floats up and fades with the same animation timing as damage numbers. No damage dealt (health bar unchanged).
  - Pass condition: "Miss" text grey, correct animation, no HP change

- **AC-3** (critical hit — larger number, amber flash, screen shake, audio):
  - Setup: In combat, land a critical hit (natural 20)
  - Verify: Damage number 1.5× larger than normal. Amber flash on target (150ms). Screen shakes (world-space only, 100ms, 2-unit displacement). Heavy impact + chime audio plays.
  - Pass condition: All 4 crit effects present (large number, amber flash, shake, audio)

- **AC-4** (combat audio events fire correctly):
  - Setup: Play through a full turn: select attack → hear soft select. Spend AP on move → hear click per AP. Land hit → hear type-specific impact. End turn → hear soft tick. Kill enemy → hear death sound.
  - Verify: Each audio event fires at the correct moment. No missing sounds. No doubled sounds.
  - Pass condition: All combat audio events mapped and firing correctly

- **AC-5** (Field register immune to screen shake):
  - Setup: Trigger a critical hit (screen shake active)
  - Verify: World rendering visibly shakes. Health bars and damage numbers (world-anchored) shake with the world. Turn order bar, AP meter, ability menu, End Turn button (Field register) remain perfectly still — no jitter, no displacement.
  - Pass condition: Field register UI stationary during screen shake, world elements shake

---

## Test Evidence

**Story Type**: Visual/Feel
**Required evidence**:
- Visual/Feel: `production/qa/evidence/combat-ui-damage-feedback-evidence.md` + sign-off

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Combat HUD Framework — combat lifecycle events, Field register established), Story 003 (World Health Bars — damage numbers appear near health bars and share world-space anchoring)
- Unlocks: None directly, but combat feel is incomplete without this story

# Story 003: Camera Follow & Framing

> **Epic**: Camera System
> **Status**: Complete
> **Last Updated**: 2026-08-11
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001, Story 002

## Context

**GDD**: `design/gdd/camera-system.md`
**Requirement**: `TR-cam-001` (partial — follow behavior)

**ADR**: ADR-0003 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [x] GIVEN exploration mode, WHEN the party moves, THEN camera smoothly follows party centroid (lerp 5 units/sec)
- [x] GIVEN conversation starts, WHEN dialogue initiates, THEN camera frames speaking NPC and player at Close zoom
- [x] GIVEN combat starts, WHEN enemies are at varying distances, THEN camera frames all combatants at Default zoom minimum
- [x] GIVEN the player drags middle mouse button, WHEN drag completes, THEN camera holds at manual position (no auto-return in reviewed GDD)
- [x] GIVEN the player presses Home key, WHEN pressed, THEN camera instantly returns to party centroid

## Implementation Notes

From ADR-0003: Camera focus point lerps toward the current target. Follow speed: 5 units/sec exploration, 8 units/sec combat. No auto-return after manual pan per the reviewed GDD — Home key for snap-back.

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10. Source: GDD CR-3, CR-4, CR-7, States, Off-Screen Markers acceptance criteria.

**Follow (CR-3)**:
- **AC-1 — Exploration follow**:
  - Given: Party moves during exploration
  - When: Each frame
  - Then: Camera follows centroid at `follow_speed_exploration` (5 u/s) using constant-velocity move-toward with distance clamping. Camera reaches target exactly, not exponential ease.

- **AC-2 — Single character selected**:
  - Given: Single character selected
  - When: Character moves
  - Then: Camera follows selected character position.

- **AC-3 — Party spread**:
  - Given: Party spread > one viewport half-extent
  - When: Exploration follow active
  - Then: Camera follows selected character, not centroid.

**Conversation framing**:
- **AC-4 — Conversation start**:
  - Given: Dialogue with NPC at ≤ 10u
  - When: Conversation starts
  - Then: Camera frames PC+NPC within inner 90% of viewport (inset = 5% of viewport height per side). Uses last Conversation zoom if both fit; minimum zoom that fits otherwise.

- **AC-5 — Speaker change, inside 70%**:
  - Given: Conversation active, speaker changes
  - When: New speaker within inner 70% of viewport
  - Then: No re-frame occurs.

- **AC-6 — Speaker change, outside 70%**:
  - Given: Conversation active, speaker changes
  - When: New speaker outside inner 70%, player NOT manually adjusted within `conversation_override_window` (5s)
  - Then: Camera re-frames.

- **AC-7 — Conversation at Far zoom**:
  - Given: Conversation starts at Far zoom
  - When: Conversation triggers
  - Then: Camera zooms to last Conversation zoom + pans to PC+NPC midpoint over `conversation_snap_duration × 2` (400ms).

- **AC-8 — Conversation ends**:
  - Given: Conversation ends
  - When: Dialogue closes
  - Then: Camera → Exploration at last Exploration zoom within `zoom_transition_duration`.

**Combat framing (CR-7)**:
- **AC-9 — Combat start, fits at Default**:
  - Given: Combat starts with 4 party + 6 enemies in 18×18u box
  - When: Combat activates
  - Then: Camera frames all combatants within inner 90% at player's Combat zoom (minimum Default) within `combat_framing_speed` (40 u/s).

- **AC-10 — Combat start, only fits at Far**:
  - Given: Combat starts with 20 combatants in 60×60u box
  - When: Combat activates
  - Then: Camera frames at Far zoom. Party + nearest 75% of enemies visible; remainder get edge markers.

- **AC-11 — Single combatant**:
  - Given: Combat with single combatant
  - When: Combat activates
  - Then: Pad framing box to minimum 8×8u in XZ plane.

- **AC-12 — Manual pan suppresses re-frame**:
  - Given: Player manually pans/zooms during combat
  - When: Turn boundary occurs
  - Then: Camera does NOT re-frame. Manual input suppresses turn-boundary re-frames for the encounter.

- **AC-13 — New combatant re-frame**:
  - Given: Player has NOT manually adjusted
  - When: New combatants join mid-encounter
  - Then: Re-frame on next turn boundary (or immediately if new count > current framed count). Overrides manual flag; player's zoom preserved.

- **AC-14 — Combat ends**:
  - Given: Combat ends
  - When: Combat exits
  - Then: Camera → Exploration at current zoom/position within `zoom_transition_duration`.

**Manual pan (CR-4)**:
- **AC-15 — Drag pan hold**:
  - Given: Player drags with pan button ≥ 100ms, ≥ 8 logical px, releases
  - When: Release occurs
  - Then: Camera holds position (drift < 0.1u over 500ms). No auto-return. World-space point under cursor at drag start stays under cursor within 2 logical pixels.

- **AC-16 — Home key**:
  - Given: Player presses Home
  - When: Key processed
  - Then: Party centroid projects to within 0.5u of camera focus on first rendered frame after input. No intermediate positions. Zoom unchanged.

- **AC-17 — End key**:
  - Given: Player presses End with character selected
  - When: Key processed
  - Then: Camera centers on selected character within one frame. No selection → no-op.

**State machine transitions**:
- **AC-18 — Combat preempts Conversation**:
  - Given: Aggro'd enemy approaching, player initiates dialogue
  - When: Combat starts mid-conversation
  - Then: Combat preempts Conversation within one frame. Combat ends → return to Conversation (one-deep stack).

- **AC-19 — Journal open during combat**:
  - Given: Journal open during combat
  - When: Journal opens
  - Then: Camera stays in Combat framing, combat UI visible behind journal. Journal close → combat framing resumes.

- **AC-20 — Journal close**:
  - Given: Journal open
  - When: Closes
  - Then: Camera returns to base state active before journal opened — NOT unconditionally Exploration.

- **AC-21 — Region transition**:
  - Given: Player crosses region boundary
  - When: Trigger fires
  - Then: Fade to black (300ms) → reposition → fade in (300ms). Movement/camera/interaction inputs locked; menu navigation functional. Fade overlay in screen space.

- **AC-22 — Pan during conversation, dialogue ends**:
  - Given: Player pans during conversation
  - When: Dialogue ends
  - Then: Camera → Exploration (not stranded in Manual).

**Off-screen markers (data contract)**:
- **AC-23 — Party marker**:
  - Given: Party member outside viewport
  - When: Any camera state active
  - Then: Edge marker at screen edge nearest entity position — solid chevron, amber `#D9A441`, display name.

- **AC-24 — Hostile action flash**:
  - Given: Hostile combatant outside viewport takes action
  - When: Action fires
  - Then: Marker flashes (200ms pulse scale 1.0→1.3→1.0) + action label adjacent for 2s.

- **AC-25 — Marker clustering**:
  - Given: ≥3 markers same edge within 40px
  - When: Rendered
  - Then: Merge into cluster (chevron + count badge). Expand on hover/focus.

- **AC-26 — Reduce Camera Motion disables animations**:
  - Given: Reduce Camera Motion = Disabled
  - When: Markers rendered
  - Then: Markers static (no pulse). Action flash suppressed.

**Edge cases**:
- `follow_speed_exploration` ≥ 1.2× party walk speed (TBD — movement GDD)
- Combat start during manual pan away from party: frame at 40 u/s; >100u distance → fade-to-black + hold first turn until framing completes
- Camera reads `Transform.Position()` directly each frame — no spatial-index range queries
- Membership via `on_component_added`/`on_component_removed` for PartyMember and Combatant

**Estimated test count**: 16 integration tests
**Test file**: `tests/integration/camera/follow_framing_test.go`

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 5/5 passing
**Deviations**: None
**Test Evidence**: `tests/integration/camera/follow_framing_test.go` (16 tests, all PASS). Covers AC-1 through AC-26 including party-spread follow, speaker-change re-frame, oversized combat at Far, drag-point-under-cursor, End key, combat-preempts-conversation one-deep stack, journal base-state restore, off-screen marker contract, edge-scroll hysteresis, keyboard pan.
**Code Review**: Skipped (Lean mode)

# Story 005: Map Centering & Camera Integration

> **Epic**: Investigation Journal
> **Status**: Ready
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 2-3 hours
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/investigation-journal.md`
**Requirement**: `TR-invj-005`
*(Map centering: clicking a thread that references a known location centers the map on that location. Unknown location fallback opens entity journal entry. Camera integration for Journal state. TR-ID pending formal registration via /architecture-review.)*

**ADR Governing Implementation**: ADR-0012: UI Framework, ADR-0003: Camera System
**ADR Decision Summary**: Journal operates in screen space (ADR-0012). Camera enters frozen Journal state when journal is open (ADR-0003). Thread click with known location triggers map centering — a cross-system integration between journal UI, entity data, and map system.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Cross-system event routing between UI layer and Camera/Map systems. Post-cutoff APIs: camera state transitions, map focus/center operations. Verify against `kaiju/src/`.

**Control Manifest Rules (Presentation layer)**:
- Required: Kaiju HTML/CSS-like UI, screen space only; Desk never trembles (journal UI immune to camera changes)
- Forbidden: Never use direct cross-system calls for state mutations — all mutations go through owning system's interface, with events for notification (ADR-0011)
- Guardrail: Map centering response <100ms from click

---

## Acceptance Criteria

*From GDD `design/gdd/investigation-journal.md`, scoped to this story:*

- [ ] **AC-1 (GDD AC-4)**: GIVEN the journal open with a thread that references a known location entity, WHEN the player clicks that thread, THEN the map centers on that location
- [ ] **AC-2**: GIVEN a thread that references an entity without a known location (e.g., a person, a concept), WHEN the player clicks that thread, THEN the journal opens the entity's journal entry (EntitySelected state) instead of attempting map centering
- [ ] **AC-3**: Camera enters Journal state (frozen) when journal opens, and returns to previous state (Exploration or Combat) when journal closes. The transition is coordinated through the Camera System's state machine
- [ ] **AC-4**: Map centering is a "request" — the journal emits a `MapCenterRequested` event with the target entity ID. The Map System handles the actual centering. If the Map System is not available, the journal shows a subtle inline notification: "Map unavailable"
- [ ] **AC-5**: Thread click behavior is consistent — location threads show a "View on map" affordance (small map pin icon next to the thread). Non-location threads show an entity chip affordance

---

## Implementation Notes

*Derived from ADR-0012, ADR-0003, and ADR-0011 Implementation Guidelines:*

**Event-driven integration (ADR-0011)**: The journal never directly calls MapSystem.CenterOn(entityID) or CameraSystem.SetState(Journal). Instead:
- Journal emits `MapCenterRequested{EntityID, Requester: "investigation_journal"}` via EventBus
- Map System subscribes to `MapCenterRequested` and handles centering
- Journal emits `JournalStateChanged{Open: true/false}` via EventBus
- Camera System subscribes to `JournalStateChanged` and transitions camera state

This keeps the Presentation layer from depending directly on other Presentation-layer systems. The EventBus mediates all cross-system communication.

**Location detection**: When a thread is clicked, check the thread's target entity. Resolve the entity's GameEntityID against Entity Framework to determine if it has a known position (has a Transform component with valid coordinates). Entities with positions: locations (buildings, areas, landmarks), some NPCs (if their current location is known). Entities without positions: abstract concepts, items in inventory, dead NPCs, unresolved references.

**Map centering flow**:
1. Player clicks thread in journal detail view
2. Journal resolves target entity → checks for position data
3. If position exists: emit `MapCenterRequested{EntityID: target, Position: (x,y), Requester: "investigation_journal"}`
4. Map System receives event → animates map to center on position
5. Journal closes (or stays open — design choice: close journal to reveal map, or keep journal open with map centering in background? Per GDD CR-2: "Clicking a thread centers the map on the relevant location" — implies journal stays open and map centers behind/alongside. Implement as: journal stays open, map centers behind the dimmed world. Player sees the location highlighted through the dimmed overlay.)

**Unknown location fallback**: If target entity has no position data, instead of map centering, transition to EntitySelected state for that entity (Story 003 implemented this state). This provides a consistent "learn more about this" behavior regardless of entity type.

**Click affordances**: Each thread entry in the detail view shows a small icon indicating click behavior:
- Thread targeting a location entity: map pin icon (pen-and-ink style, 16px, umber) to the right of the thread description
- Thread targeting a non-location entity: entity chip icon (small circle/dot, 12px)
- Thread with no target entity (manual/abstract thread): no icon

**Camera state coordination**: Journal open → Camera enters Journal state (frozen: no pan, no zoom, no follow). Journal close → Camera returns to Exploration state (or Combat, if combat is active — check current game state). This was partially implemented in Story 001; this story ensures the Camera System properly consumes the `JournalStateChanged` event and validates the integration end-to-end.

**Map System dependency note**: As of 2026-08-11, the Map System epic has no stories yet. The journal's `MapCenterRequested` event emission is implemented and testable (verify the event is emitted with correct payload), but the Map System consumer does not exist. Test by subscribing a mock/test consumer to `MapCenterRequested` and verifying event content. When the Map System is implemented, it subscribes to this event and integration is complete without journal changes.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Journal open/close lifecycle (the `JournalStateChanged` event emission is wired here as part of the lifecycle)
- Story 002: Thread display and click handling (this story adds the location-aware click behavior on top)
- Story 003: EntitySelected state and entity detail display
- Map System implementation: the journal only emits `MapCenterRequested` events; map rendering and centering animation are owned by the Map System epic
- Camera state machine: camera state transitions are owned by Camera System (ADR-0003); journal only emits `JournalStateChanged`

---

## QA Test Cases

**Automated tests (Integration story — must have test file):**

- **AC-1**: Map centering on location thread click
  - Given: Journal open, chain selected, thread with target entity that has known position (e.g., "Old Mine" at grid (45, 30))
  - When: Player clicks the thread
  - Then: `MapCenterRequested` event is emitted via EventBus with EntityID of "Old Mine", position (45, 30), and Requester "investigation_journal". Thread shows map pin icon
  - Edge cases: Thread with position (0,0) (valid — emit event); thread with negative coordinates (valid — emit event); multiple rapid clicks (debounce — emit one event per click, map system handles dedup)

- **AC-2**: Entity entry fallback for non-location thread
  - Given: Thread targeting an NPC entity ("Captain Veren") that has no position data
  - When: Player clicks the thread
  - Then: No `MapCenterRequested` event emitted. Journal transitions to EntitySelected state for "Captain Veren". Thread shows entity chip icon (not map pin)
  - Edge cases: Entity with position that becomes unknown between journal open and click (stale position — emit event with last known position); entity deleted between journal open and click (show "Entity no longer exists" notification)

- **AC-3**: Camera state on journal open/close
  - Given: Camera is in Exploration state
  - When: Journal opens
  - Then: `JournalStateChanged{Open: true}` event emitted. Camera receives event and enters Journal state (frozen). When journal closes: `JournalStateChanged{Open: false}` emitted. Camera returns to Exploration state
  - Edge cases: Journal opens during combat (camera was in Combat state — journal opens, camera freezes; on close, camera returns to Combat, not Exploration); rapid open/close (each transition handled cleanly, no stuck states)

- **AC-4**: Map unavailable graceful degradation
  - Given: No subscriber is registered for `MapCenterRequested` events (Map System not implemented/not running)
  - When: Player clicks location thread
  - Then: Event is emitted (no error). Journal shows subtle inline notification: "Map unavailable" (12px, muted umber, appears for 2s then fades). No crash, no hung state
  - Edge cases: Map System subscribes mid-session (journal doesn't track availability — always emits event); Map System crashes (event bus handles dead subscriber, journal unaffected)

- **AC-5**: Thread click affordances
  - Given: Chain detail view with 3 threads: one targeting location, one targeting NPC, one with no target
  - When: View the thread list
  - Then: Location thread shows map pin icon (16px, umber). NPC thread shows entity chip icon (12px). No-target thread shows no icon. All icons are pen-and-ink style, distinguishable by shape alone
  - Edge cases: Thread has both location AND entity target (prefer location — show map pin); target type changes between renders (re-render affordance icon)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/investigation_journal/map_camera_integration_test.go` — must exist and pass
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 002 (Chain List & Thread Detail), Camera System stories 001-004 (Complete ✅), Entity Framework (Complete ✅)
- Soft dependency: Map System epic (stories not yet created) — journal emits `MapCenterRequested` events that the map system will consume. Test with mock subscriber until Map System is implemented.
- Unlocks: None (final story in epic)

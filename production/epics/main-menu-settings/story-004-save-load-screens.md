# Story 004: Save/Load Screens

> **Epic**: Main Menu & Settings
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: L (5-7 hours) — 22-slot browser, metadata display, overwrite confirmation, corrupted save error dialog, Steam Cloud conflict modal. Integration type: requires integration tests with stubbed Save/Load System data contract.
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/main-menu-and-settings.md`
**Requirement**: `TR-ui-001`, `TR-ui-005`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework; also references ADR-0002 (Save/Load — Steam Cloud conflict, checksum validation)
**ADR Decision Summary**: Desk register parchment panels for save/load UI. Save/Load System provides slot metadata; Main Menu renders it. Steam Cloud conflict resolution dialog with Keep Local/Cloud/Decide Later. Corrupted saves detected via CRC32 checksum (ADR-0002).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju UI scroll area for 22-slot list. Thumbnail image rendering (PNG, 320×180). Button states (occupied/empty/corrupted). Confirmation dialog modal. Steam Cloud API integration path (Go Steamworks bindings or game-layer concern — open question).

**Control Manifest Rules (Presentation layer)**:
- Required: Desk register — parchment panels, world dims to 30%
- Required: Paper-slide motion for screen transitions
- Required: Desk serif (Garamond class) for save slot metadata
- Forbidden: Never use amber-on-parchment text (1.5:1 fails WCAG AA)
- Guardrail: UI render (static): <1ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/main-menu-and-settings.md` CR-2:*

- [ ] **AC-1**: The Load Game screen displays all 22 save slots (20 manual + 1 auto-save + 1 quicksave), sorted by last-modified timestamp descending. Each occupied slot shows: slot name, character name, playtime (hours:minutes), region name, thumbnail (320×180 PNG), and timestamp (date + time). Empty slots show "Empty Slot" with dimmed styling.

- [ ] **AC-2**: Occupied slots are clickable in the Load screen; empty slots are non-interactive. Clicking an occupied slot begins the load operation: crossfade to black (`save_load_fade_ms` = 300ms), load the save, crossfade in to the game world.

- [ ] **AC-3**: In the Save screen, clicking an occupied slot shows an overwrite confirmation dialog: "Overwrite [slot name]?" with details of what will be overwritten. Options: "Overwrite" / "Cancel". Clicking an empty slot opens the name-entry prompt (default: character name + timestamp), then saves.

- [ ] **AC-4**: Clicking a corrupted save slot (checksum mismatch) in the Load screen shows an error dialog: "This save file is corrupted." Options: "Restore from backup" / "Delete save" / "Cancel". The game does not attempt to load the corrupted file.

- [ ] **AC-5**: On game launch, if Steam Cloud detects local and cloud saves differ, a conflict modal appears before the title screen showing both versions (timestamp, playtime, region for each). Options: "Keep Local" / "Keep Cloud" / "Decide Later."

- [ ] **AC-6**: If "Decide Later" is chosen: a persistent warning banner displays on the title screen and pause menu: "Auto-save disabled — resolve Steam Cloud conflict in Settings → Save/Load." Auto-saves are blocked until resolution. The unchosen version is kept as `.conflict_backup` for 30 days.

- [ ] **AC-7**: The Save screen is accessed from the Pause Menu only. The Load screen is accessed from the Title Screen or Pause Menu. Both share the same slot-list layout but are distinct screens with different interaction rules.

---

## Implementation Notes

*Derived from ADR-0012 and ADR-0002:*

- **Slot card layout**: Dark stone card background (NOT parchment — metadata is illegible on parchment at 1.17:1 contrast). Thumbnail on the left, metadata stacked on the right. Selected slot: amber gold border + ink underline (two cues — never color alone). Cards use the desk serif for metadata text (pale ivory, ≥12:1 contrast on dark stone).
- **Data contract**: Save/Load System provides slot metadata via a Go interface: slot ID, name, character name, playtime (time.Duration), region name, thumbnail (image.Image or path to PNG), timestamp (time.Time), checksum valid (bool). Main Menu renders this data — never queries entities directly. Empty slots return nil/zero values.
- **Save/Load commands**: Menu sends commands to Save/Load System: `SaveGame(slotID, name)` → returns error or success; `LoadGame(slotID)` → begins async load; `DeleteSave(slotID)` → confirmation then delete. Menu blocks input during save/load operations.
- **Corrupted save detection**: Save/Load System returns `checksum_valid: false` in slot metadata. Menu renders the slot with a warning icon and different click behavior (error dialog instead of load). The actual CRC32 check is Save/Load's responsibility; menu trusts the metadata flag.
- **Steam Cloud conflict**: On game launch, Save/Load System checks cloud state and returns conflict data if mismatch. Menu renders the conflict dialog before the title screen. "Keep Local" / "Keep Cloud" overwrites the other. "Decide Later" sets a flag — auto-saves are blocked, warning banner shown. The unchosen version is renamed to `.conflict_backup` with a 30-day retention timestamp.
- **Save slot count**: 20 manual + 1 auto-save + 1 quicksave = 22 total. Slot list uses a Kaiju scroll area. Sort: last-modified descending (most recent first).
- **Overwrite confirmation**: Destructive action — uses the confirmation dialog pattern. "Overwrite" label is clear about what's being lost (existing save name, character, playtime). Cancel returns to slot list.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Navigation (Tab/arrows/gamepad in slot list), focus rings, crossfade transitions, input blocking during load
- Story 002: "Load Game" button on title screen (triggers this screen)
- Story 005: Settings → Save/Load tab for resolving Steam Cloud conflict from settings
- Story 006: Pause menu "Save" and "Load" options (trigger these screens)
- Save/Load System: Actual save/load logic, CRC32 validation, Steam RemoteStorage API, .sav file format, backup/restore, migration — menu is UI only
- Steam Cloud sync: The engine-side Steamworks integration is a deferred engine feature (MVP: greyed out if unavailable)

---

## QA Test Cases

*Derived from GDD acceptance criteria. The developer implements against these — do not invent new test cases during implementation.*

### Automated (integration tests)

- **AC-1** (Slot list display):
  - Given: Save/Load System provides metadata for 5 saves
  - When: the Load screen renders
  - Then: 5 occupied slots display character name, playtime, region, thumbnail, timestamp; 17 empty slots show "Empty Slot" with dimmed styling; slots sorted by timestamp descending
  - Edge cases: 0 saves → all 22 slots empty, "Load Game" on title greyed out (Story 002 AC-6); 22 saves → all slots occupied, no empty slots visible

- **AC-2** (Load from occupied slot):
  - Given: Load screen with 5 occupied slots
  - When: an occupied slot is clicked
  - Then: crossfade to black begins (300ms), `LoadGame(slotID)` is called; on success, crossfade in to game world; on failure, Title Screen with error (Story 001 AC-9)
  - Edge cases: empty slot click → no action; input blocked during load operation

- **AC-3** (Save overwrite confirmation):
  - Given: Save screen, slot 3 is occupied (character "Alric", 12:30 playtime)
  - When: slot 3 is clicked
  - Then: confirmation dialog appears: "Overwrite Alric — 12:30?" with details of existing save; "Overwrite" calls `SaveGame(slotID, name)`; "Cancel" returns to slot list
  - Edge cases: save write fails → error toast (per GDD edge case: "Save failed — unable to write to [path]"); empty slot click → name-entry prompt with default "character_name + timestamp"

- **AC-4** (Corrupted save dialog):
  - Given: Load screen, slot 7 has `checksum_valid: false`
  - When: slot 7 is clicked
  - Then: error dialog appears: "This save file is corrupted." with options "Restore from backup", "Delete save", "Cancel"; game does NOT call `LoadGame`; "Restore from backup" calls Save/Load restore; "Delete save" calls `DeleteSave` with confirmation
  - Edge cases: backup also corrupted → "No valid backup found" message; Cancel returns to slot list

- **AC-5** (Steam Cloud conflict dialog):
  - Given: game launch, local save has playtime 8:30, cloud save has playtime 7:45
  - When: conflict is detected
  - Then: modal appears before title screen showing both versions with timestamp, playtime, region; three options present: "Keep Local", "Keep Cloud", "Decide Later"
  - Edge cases: network unavailable → deferred, game launches normally; "Decide Later" → warning banner, auto-save blocked

- **AC-6** (Decide Later consequences):
  - Given: Steam Cloud conflict, "Decide Later" was chosen
  - When: title screen renders
  - Then: warning banner visible: "Auto-save disabled — resolve Steam Cloud conflict in Settings → Save/Load"; auto-save calls are blocked
  - Edge cases: banner also visible on pause menu; resolution via Settings clears banner and re-enables auto-save

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/main_menu/save_load_screens_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Navigation & Transition Framework), Save/Load System (ADR-0002 — provides slot metadata and save/load/delete commands)
- Unlocks: None directly (pause menu triggers these screens but doesn't depend on them)

---

## Completion Notes

**Completed**: 2026-08-12
**Criteria**: 7/7 passing (29 integration tests)
**Deviations**: None
**Scope note**: `screen_flow.go` transition table extended — `ScreenLoadGame` now includes `ScreenPauseMenu` (back-nav from PauseMenu entry point)
**Test Evidence**: `tests/integration/main_menu/save_load_screens_test.go` — 29 tests, all PASS (99 total across Stories 001-004)
**Code Review**: Complete — APPROVED, 6/6 standards, ADR-0012 + ADR-0002 compliant
**External dependency**: Save/Load System data contract stubbed — real integration blocked until save system stories complete

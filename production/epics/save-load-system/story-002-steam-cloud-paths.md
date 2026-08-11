# Story 002: Steam Cloud & Platform Paths

> **Epic**: Save/Load System
> **Status**: Complete
> **Last Updated**: 2026-08-11
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/save-load-system.md`
**Requirement**: `TR-save-003`, `TR-save-004`

**ADR**: ADR-0002 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN Steam is running, WHEN the game saves, THEN the save file syncs to Steam Cloud via Auto-Cloud
- [ ] GIVEN local and cloud saves with different timestamps, WHEN the game launches, THEN the player is shown both versions and asked to choose
- [ ] GIVEN Steam is not running (offline/GOG), WHEN the player saves and loads, THEN all operations function normally without Steam
- [ ] GIVEN Linux, WHEN save files are written, THEN they go to `$XDG_DATA_HOME/marrow-of-empire/saves/`
- [ ] GIVEN Windows, WHEN save files are written, THEN they go to `%APPDATA%/marrow-of-empire/saves/`

## Implementation Notes

From ADR-0002: Steamworks API for Auto-Cloud. Save files registered with Steam Cloud on creation. Conflict resolution dialog shows timestamp + playtime for both versions. Non-Steam builds compile without Steamworks dependency — save paths work identically.

**Control manifest rules** (Foundation Layer, ADR-0002):
- **Required**: Platform-standard save paths (Linux `$XDG_DATA_HOME`, Windows `%APPDATA%`, macOS `~/Library/Application Support`), Steam Auto-Cloud at launch, conflict resolution dialog on timestamp mismatch
- **Required**: Save files registered with Steam Cloud on creation; cloud sync is additive — save system functions fully without Steam runtime
- **Forbidden**: Never hardcode save paths — always resolve from OS environment
- **Performance**: Steam Cloud sync non-blocking — game continues during sync. Platform path resolution is a one-time lookup at startup via `os.UserConfigDir` — no per-frame cost. No frame-budget impact.

## Out of Scope

- **Steamworks CGo bindings**: The actual Steam API integration (`steam_cloud.go` with CGo) is deferred — engine remote-storage bindings are unavailable in the current Kaiju nightly. A pure-Go `Cloud` interface is provided as the integration point for when bindings become available.
- **Steam Auto-Cloud enable/disable at runtime**: The `Cloud` interface declares `Sync(path)` and `ResolveConflict(local, remote)` — the runtime toggle (enable/disable in settings) is a UI concern (Main Menu & Settings).
- **macOS path testing**: macOS `~/Library/Application Support` path is implemented but tested only on Linux — macOS CI is not configured.
- **`.bak` backup/restore**: Covered by Story 003 (Corruption Recovery).
- **Save file I/O primitives** (CRC32, header, atomic writes): Implemented in Story 001 — this story adds the path resolution and cloud sync layer on top.

## Test Evidence

| Test File | Type | Covers |
|-----------|------|--------|
| `tests/unit/save/platform_test.go` | Unit | AC-3 (offline no-op), AC-4 (Linux XDG path), AC-5 (Windows APPDATA path) — 8 tests |
| `tests/unit/save/steam_test.go` | Unit | Cloud interface contract, no-op backend — 2 tests |

**Deferred tests** (AC-1, AC-2): Require Steam runtime or Steamworks mock — follow-up story when engine bindings available.

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 3/5 implemented, 2 deferred (AC-1 Steam sync, AC-2 conflict dialog — blocked on engine remote-storage bindings; Cloud interface + integration point documented)
**Implemented**: AC-3 offline/GOG no-op, AC-4 Linux XDG path, AC-5 Windows APPDATA path
**Deviations**: No steam_cloud.go (bindings unavailable per story fallback); Cloud interface uses pure Go contract; DefaultSaveDir changed from const to function
**Test Evidence**: `tests/unit/save/platform_test.go` (8 tests) + `tests/unit/save/steam_test.go` (2 tests), all passing
**Code Review**: Skipped (Lean mode)

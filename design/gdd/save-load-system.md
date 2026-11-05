# Save/Load System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 2 (The World Remembers — saves are the persistence mechanism)

## Summary

The Save/Load System provides persistence for The Marrow of Empire by
orchestrating Kaiju Engine's `pod` serialization across all game component
stores. It supports manual saves (20 slots), quick-save/load (F5/F9), and
auto-save on region transitions. Save files include versioning for forward
compatibility, checksums for corruption detection, and Steam Cloud sync.
Linux saves live at `~/.local/share/marrow-of-empire/saves/`.

> **Quick reference** — Layer: `Foundation` · Priority: `MVP` · Key deps: `Entity Framework`

## Overview

The Save/Load System provides persistence for The Marrow of Empire. It captures
the complete game state — entities, components, world state, investigation
progress, and settings — and restores it on demand. This is the system that
makes "continue playing" possible.

The system orchestrates Kaiju Engine's native serialization (`pod` package,
`EntityData` interface, reflection-based encode/decode) rather than implementing
its own serialization layer. Each game component registers with `pod.Register`;
the Save/Load system triggers the serialization pass, writes the resulting data
to disk, and orchestrates the reverse on load.

The system provides:

- **Save slots**: Multiple named save files. The player can create, overwrite,
  and delete saves. Slots display character name, playtime, region name, and
  a thumbnail screenshot for identification.
- **Auto-save**: Automatic saves on region transitions and after significant
  story events. One auto-save slot, overwritten each time. Prevents progress
  loss from crashes or forgetting to save.
- **Quick-save/load**: Single-key quicksave (F5) and quickload (F9) for rapid
  save/reload during exploration. One quicksave slot.
- **Save file integrity**: Each save file includes a version header, checksum,
  and metadata block. Corrupted saves are detected on load and reported to the
  player rather than silently crashing. Save versioning supports forward
  compatibility — newer game versions can load older saves with data migration.
- **Save orchestration**: The Save/Load system doesn't know what data each
  component contains. It triggers the serialization pass on all registered
  component stores, collects the encoded data, and writes the save file. On
  load, it clears current state, reads the save file, and distributes decoded
  data back to component stores.

Without the Save/Load System: the player's investigation, party, and world
state exist only in RAM. Every game session starts from zero. The World
cannot Remember anything.

## Player Fantasy

The Save/Load System serves Pillar 2 (The World Remembers) directly. Its player
fantasy is simple but load-bearing:

**Your progress is safe.** The player can close the game knowing that every NPC
they questioned, every thread they followed, every ruin they cleared, and every
connection they made will be there when they return. The save system is the
silent guarantor of persistence — it turns "this session" into "this story."

A CRPG investigation that spans dozens of hours across multiple regions cannot
exist without trust in the save system. If a player worries about losing progress,
they stop investigating and start save-scumming. If they trust the auto-save and
quick-save, they stay immersed in the world.

The Save/Load System's success metric: the player never thinks about it. They
quit when they're done, not when they've reached a save point. They return to
exactly where they left off — same party, same threads, same world state. The
system is invisible, and the persistence feels natural.

## Detailed Design

### Core Rules

**CR-1: Save File Structure**

Each save file is a single binary blob produced by Kaiju's `pod` encoder:

| Section | Content | Writer |
|---------|---------|--------|
| Header | Save version (uint32), checksum (uint32), timestamp | Save/Load System |
| Metadata | Character name, playtime, region ID, thumbnail PNG | Save/Load System |
| Entity Table | GameEntityID → Kaiju EntityId mapping | Entity Framework |
| Component Stores | All registered component data, per component type | Each component store via `pod` |
| World State | Active/suspended entity lists, persistent flags | Persistent World State |
| Investigation | PlayerKnowledge component, thread state | Thread Investigation |
| Settings | Audio levels, keybindings, accessibility flags | Main Menu & Settings |

**CR-2: Save Operations**

| Operation | Trigger | Behavior |
|-----------|---------|----------|
| Manual Save | Player clicks "Save" in menu | Opens slot selection; player names the save or overwrites existing; snapshot thumbnail captured; save written synchronously |
| Quick-save (F5) | Player presses F5 | Writes to the quicksave slot silently; brief "Game Saved" indicator flashes (1.5s) |
| Auto-save | Region transition, major story event | Writes to the auto-save slot silently; no UI indicator |
| Load | Player selects save from "Load Game" menu | Current state cleared; save file read; entities recreated; component data restored; player placed at saved position |
| Quick-load (F9) | Player presses F9 | Loads the quicksave slot; confirmation dialog if combat is active |
| Delete | Player selects save and confirms deletion | Save file deleted from disk; slot freed |

**CR-3: Save Versioning**

Each save file carries a version number. The game checks this on load:

- **Same version**: Direct load.
- **Older version, supported migration**: Run migration function(s) to transform
  old data to current format. Log migration applied. Load migrated data.
- **Older version, unsupported migration**: Warn player: "This save was created
  with an older version of the game and may not load correctly." Offer to attempt
  load or cancel.
- **Newer version**: Reject load: "This save was created with a newer version
  of the game. Please update your game."

Migration functions are registered per version transition (e.g., `v1 → v2`
migration adds a new component field with a default value).

**CR-4: What Is NOT Saved**

Not everything persists. The following are explicitly excluded from save files:

- **Transient visual state**: Current animation frame, particle positions,
  temporary VFX. These reset on load — the entity appears in its idle state.
- **Audio state**: Current playback position of ambient loops. Audio fades in
  on load.
- **Camera position**: ~~Camera resets to party centroid on load.~~ **Camera state IS saved** per ADR-0002 amendment (2026-08-12): camera position, current zoom, per-context zoom preferences, base state, Manual flag, and preemption stack are fully restored on load. This overrides the previous exclusion — camera persistence is required by Camera System GDD CR-13.
- **UI state**: Open panels, scroll positions. UI resets to default on load.
- **Combat turn state** (if saved mid-combat): Combat restarts at the beginning
  of the current round on load.

**CR-5: Save During Restricted States**

Some states restrict saving:

| State | Manual Save | Quick-save | Auto-save |
|-------|-------------|------------|-----------|
| Exploration | Allowed | Allowed | Allowed |
| Conversation | Allowed | Allowed | Blocked |
| Combat | Allowed | Allowed | Blocked |
| Journal | Allowed | Blocked | Blocked |
| Region Transition | Blocked | Blocked | Blocked (already saving) |
| Cutscene | Blocked | Blocked | Blocked |

**CR-6: Save File Location**

Save files are stored in the OS-standard user data directory, not in the game
install directory:

| Platform | Path |
|----------|------|
| Linux | `$XDG_DATA_HOME/marrow-of-empire/saves/` (default: `~/.local/share/marrow-of-empire/saves/`) |
| Windows | `%APPDATA%/marrow-of-empire/saves/` |
| macOS | `~/Library/Application Support/marrow-of-empire/saves/` |

- **Naming**: `save_[slot_id].sav` — slot IDs are sequential integers.
- **Maximum slots**: 20 manual saves + 1 auto-save + 1 quicksave = 22 total.
  Exceeding the limit prompts the player to delete an existing save.
- **Backup**: Before overwriting a save, the previous version is kept as
  `save_[slot_id].bak`. If the new save fails (disk full, crash mid-write), the
  backup is restored on next launch.

**CR-7: Steam Cloud**

Save files are registered with Steam Cloud for automatic synchronization:

- All `.sav` and `.bak` files in the saves directory are included in Steam
  Cloud sync.
- Steam Auto-Cloud is enabled — saves sync on game launch and exit.
- Conflict resolution: if local and cloud saves differ (e.g., player played on
  two machines offline), the game prompts the player to choose which version to
  keep, displaying timestamp and playtime for each.
- Steam Cloud is a launch requirement. The save system must function correctly
  without Steam (offline/GOG release) — cloud sync is additive, not required.

### States and Transitions

The Save/Load System is stateless at runtime — it performs discrete operations.

**Save operation flow**:
`trigger_save` → collect entity table → serialize component stores → capture thumbnail → compute checksum → write file → verify → sync to Steam Cloud → `save_complete`

**Load operation flow**:
`trigger_load` → read header → verify checksum → check version → run migrations if needed → clear current state → deserialize component stores → rebuild entity mappings → restore world state → place player → `load_complete`

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Save serializes entity table; Load rebuilds it | Entity Framework provides `GameEntityID` ↔ `EntityId` mapping and component store access |
| Persistent World State | Save includes persistent flags; Load restores them | World state data included in save file |
| Thread Investigation | Save includes `PlayerKnowledge`; Load restores it | Investigation progress included in save file |
| Main Menu & Settings | Save includes settings; Load restores them | Settings are per-user, not per-save — saved separately in `settings.cfg` |
| Asset & Scene Pipeline | Load triggers scene load for the saved region | Save stores region ID; Load requests scene load from pipeline |
| Camera System | Load restores full camera state (position, zoom, per-context prefs, base state, preemption stack) | Camera state serialized by Save/Load, restored by Camera System per ADR-0002 amendment |
| Steamworks API | Save triggers Steam Cloud sync | Auto-Cloud enabled; conflict resolution on mismatch |

---

## Formulas

No gameplay formulas. Save file size estimate: 500 entities × average 2KB per
entity component data ≈ 1MB + asset references + metadata ≈ 2-3MB per save.
Quick-save and auto-save sizes are identical to manual saves.

## Edge Cases

- **If the disk is full during save**: The write fails. The system retains the
  previous `.bak` file. Player sees "Save failed — insufficient disk space."
  Game state is unaffected.
- **If the game crashes mid-save**: The `.sav` file is incomplete (checksum
  mismatch on next launch). The `.bak` file is intact. On next launch, the
  system detects the corrupted `.sav`, restores from `.bak`, and logs the
  incident.
- **If the player loads a save from a different game version where a referenced
  asset has been retired**: The asset still exists (Retired assets are never
  deleted per Asset Pipeline CR-5). The entity loads normally. Editor warning
  is logged.
- **If a component's `EntityData` registration changed between versions**: The
  migration function for that version handles the data transformation. If no
  migration exists, new fields get zero values and removed fields are silently
  dropped.
- **If the player deletes the auto-save**: The next auto-save event creates a
  new auto-save. No error.
- **If Steam Cloud sync fails (network issue, Steam offline)**: The local save
  completes normally. Cloud sync retries on next launch. No error shown to
  player unless conflict is detected on next sync.
- **If local and cloud saves conflict**: Player is shown both versions
  (timestamp, playtime, region) and asked to choose. The unchosen version is
  kept as a `.conflict_backup` file for 30 days.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | Save/Load depends on Entity Framework | Receives entity table, component store access. Hard — cannot save without entity data. |
| Kaiju Engine (`pod`) | Save/Load depends on Kaiju `pod` | Uses `pod.Register`, `EntityData` interface for serialization. Hard — engine-level dependency. |
| Persistent World State | Persistent World State depends on Save/Load | World state data included in save. Hard — world memory requires persistence. |
| Thread Investigation | Thread Investigation depends on Save/Load | Investigation state included in save. Hard — threads must survive sessions. |
| Main Menu & Settings | Main Menu depends on Save/Load | Provides save/load UI; settings saved separately. Hard — menu needs save slots to display. |
| Asset & Scene Pipeline | Save/Load triggers scene load | Save stores region ID; load requests the scene. Soft — save can store the ID without the pipeline being ready. |
| Steamworks API | Save/Load syncs via Steam Cloud | Auto-Cloud enabled at launch. Soft — game functions fully without Steam. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect |
|-----------|---------|------------|--------|
| `max_save_slots` | 20 | 5–50 | More slots, more disk usage. Each slot ~3MB. |
| `auto_save_slots` | 1 | 1–3 | More auto-save history, more disk usage |
| `thumbnail_width` | 320px | 160–640 | Larger thumbnails, larger save files (~50KB each) |
| `thumbnail_quality` | 70 (JPEG %) | 30–95 | Higher quality, larger thumbnails |

## Visual/Audio Requirements

The Save/Load System produces no visual or audio output of its own. Its UI
requirements are covered by Main Menu & Settings (save/load screens). The
only visual feedback owned by this system: the brief "Game Saved" indicator
(1.5s fade-in/out text overlay) on quick-save.

## UI Requirements

The Save/Load System provides data to Main Menu for its save/load screens:
save slot list (name, thumbnail, playtime, region, timestamp), confirmation
dialogs (overwrite, delete), and the conflict resolution dialog for Steam
Cloud mismatches. All UI rendering is handled by Main Menu & Settings.

## Acceptance Criteria

- **GIVEN** an active game session with 100 entities and investigation state, **WHEN** the player performs a manual save, **THEN** a save file is written to disk within 2 seconds and reloading produces identical entity IDs, positions, component values, and investigation threads.

- **GIVEN** a save file from game version N, **WHEN** game version N+1 loads it with a registered migration, **THEN** the migration runs successfully and the save loads with migrated data intact.

- **GIVEN** a corrupted save file (checksum mismatch), **WHEN** the player attempts to load it, **THEN** the system reports the corruption and offers to restore from backup.

- **GIVEN** the player presses F5 during exploration, **WHEN** the quicksave completes, **THEN** a brief "Game Saved" indicator appears for 1.5 seconds and the save syncs to Steam Cloud.

- **GIVEN** the player presses F9 with a valid quicksave, **WHEN** the quickload completes, **THEN** the game state is identical to the moment of the quicksave.

- **GIVEN** a region transition, **WHEN** the player crosses the boundary, **THEN** an auto-save triggers and completes before the new region loads.

- **GIVEN** local and Steam Cloud saves with different timestamps, **WHEN** the game launches, **THEN** the player is shown both versions and asked to choose.

- **GIVEN** the game running without Steam (offline/GOG), **WHEN** the player saves and loads, **THEN** all save operations function normally without Steam Cloud.

- **Performance**: Save completes within 2 seconds for 500 entities. Load completes within 3 seconds. No frame drops below 30fps during save or load. Save file size ≤ 5MB for 500 entities.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Does Kaiju's `pod` package support incremental/delta serialization, or is it always full-state? Affects save performance for large worlds. | Engine Programmer | Before implementation (ADR) | — |
| What is the Kaiju build tag or platform constant for detecting Steam vs. non-Steam builds? Needed for conditional Steamworks API compilation. | Engine Programmer | Before implementation | — |

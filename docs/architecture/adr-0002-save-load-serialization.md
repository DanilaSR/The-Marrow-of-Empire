# ADR-0002: Save/Load Serialization

## Status
Accepted

## Date
2026-08-09
**Amended**: 2026-08-12 — Camera state persistence (resolves ADR-0002/ADR-0003 conflict: camera position, zoom, per-context prefs, and preemption stack are now included in save scope)

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan, custom) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | `kaiju/AGENTS.md`, `kaiju/src/pod/`, Save/Load GDD |
| **Post-Cutoff APIs Used** | `pod.Register`, `EntityData` interface |
| **Verification Required** | `pod` delta serialization support, `EntityData` interface requirements, Steamworks API Go bindings |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — component stores must register with `pod`); ADR-0003 (Camera System — camera state serialization contract) |
| **Enables** | ADR-0010 (Thread Investigation — investigation state persistence); ADR-0014 (Audio System — volume/mute settings persistence) |
| **Blocks** | Persistent World State implementation |
| **Ordering Note** | Must be Accepted before any system that persists state is implemented |

## Context

### Problem Statement
The Marrow of Empire must persist all game state across sessions — entities, components, world state, investigation progress, and settings. Kaiju Engine provides `pod` (reflection-based binary encoding) and the `EntityData` interface. How should the game orchestrate save/load using these primitives?

### Constraints
- Save must complete within 2 seconds for 500 entities
- Load must complete within 3 seconds
- Save files must be forward-compatible (newer game versions load older saves)
- Must support Steam Cloud sync at launch
- Must function without Steam (offline/GOG)
- 20 manual save slots + 1 auto-save + 1 quicksave = 22 max

### Requirements
- All game state lives in component stores (ADR-0001) — save captures all stores
- Corrupted saves must be detected (checksum) and recovered (backup)
- Save versioning with migration functions per version transition
- Platform-standard save locations (Linux: `$XDG_DATA_HOME/marrow-of-empire/saves/`)

## Decision

**Full-state serialization via Kaiju `pod` with versioned save files.** Each save operation captures the complete state of all registered component stores, the entity table, world state flags, investigation data, and camera state — encoded as a single binary blob via `pod.Encode`. Save files carry a version header, checksum, and metadata block.

**Camera state persistence (amended 2026-08-12)**: Camera position, current zoom level, per-context zoom preferences (Exploration/Conversation/Combat), base state, Manual flag, and preemption stack are included in save scope. On load, the camera is restored to its saved state — not reset to party centroid. This resolves the conflict between Save/Load GDD CR-4 (which excluded camera) and Camera GDD CR-13 (which requires full camera state persistence). The Camera GDD is authoritative on this point.

### Architecture Diagram

```
Save:  Save/Load trigger → collect entity table → serialize all component stores
       → encode world state + investigation + camera state → compute checksum →
       write .sav → write .bak → sync Steam Cloud

Load:  Read .sav → verify checksum → read version → run migrations (if needed)
       → clear current state → pod.Decode component stores → rebuild entity
       mappings → restore world state → restore camera state (position, zoom,
       per-context prefs, base state, preemption stack) → restore settings
```

### Key Interfaces

```go
type SaveManager interface {
    Save(slotID int, metadata SaveMetadata) error
    Load(slotID int) (*SaveMetadata, error)
    ListSaves() []SaveSlot
    DeleteSave(slotID int) error
}

type SaveMetadata struct {
    CharacterName string
    PlayTime      time.Duration
    RegionID      string
    Thumbnail     []byte // 320px JPEG
    Timestamp     time.Time
    Version       uint32
}

// Component stores implement this for pod serialization.
// Registered via pod.Register at init.
type EntityData interface {
    Encode() ([]byte, error)
    Decode([]byte) error
}
```

**Save file format:**
```
[Header: 16 bytes]
  Version    uint32 (4 bytes)
  Checksum   uint32 (4 bytes — CRC32 of remaining data)
  Timestamp  int64  (8 bytes)

[Metadata: variable — pod-encoded SaveMetadata]
[Entity Table: variable — map[GameEntityID]EntityId]
[Component Stores: variable — per-component-type encoded data]
[World State: variable — Persistent World State flags + memory log]
[Investigation: variable — PlayerKnowledge component data]
[Camera State: variable — position (vec3), zoom (float32), per-context zoom prefs,
  base state (enum), manual flag (bool), preemption stack]
[Settings: variable — per-user settings snapshot (optional, separate settings.cfg)]
```

**Explicitly NOT saved** (transient state, per Save/Load GDD CR-4):
- Animation frames, particle systems, VFX — reset to idle on load
- Audio playback positions — fade in on load
- UI open/close state, scroll positions, selected tabs — reset to default
- Combat turn state (if saved mid-combat) — combat restarts at beginning of current round

**Camera state IS saved** (amended 2026-08-12 — overrides GDD CR-4 exclusion). See Camera State section above.

**Save location**: `$XDG_DATA_HOME/marrow-of-empire/saves/save_{slotID}.sav`

## Alternatives Considered

### Alternative 1: Full-state pod serialization (CHOSEN)
- **Pros**: Simple — one encode/decode pass. Kaiju native — no custom serialization. Reflection-based — new components auto-register. Consistent save state — no partial saves.
- **Cons**: Full-state means save time scales with entity count. No incremental/delta saves. Large save files at high entity counts.
- **Rejection Reason**: N/A — chosen.

### Alternative 2: Incremental/delta serialization
- **Description**: Only save changed components since last save.
- **Pros**: Faster saves, smaller files.
- **Cons**: Requires change tracking per component. Load requires replaying delta chain. Complex — bugs in delta logic produce unrecoverable saves. Kaiju `pod` may not support delta encoding natively.
- **Rejection Reason**: Added complexity not justified for MVP (500 entities × 2KB = 1MB save files). Revisit if save times exceed 2-second budget.

### Alternative 3: JSON/text-based saves
- **Description**: Encode game state as JSON for human readability/debugging.
- **Pros**: Debuggable, diffable, human-readable.
- **Cons**: 5-10× larger files. Slower encode/decode. Reflection overhead in Go's encoding/json. Not Kaiju-native.
- **Rejection Reason**: Performance cost and file size overhead not justified. Binary saves are smaller, faster, and Kaiju-native.

## Consequences

### Positive
- Single save/load code path — easy to test, easy to debug
- Kaiju-native serialization — no custom encoding to maintain
- Versioned saves enable forward compatibility
- Checksum + backup prevents save corruption data loss
- Steam Cloud integration at launch

### Negative
- Full-state saves scale linearly with entity count — 5000 entities = ~10MB saves
- No partial/delta saves — every save writes everything
- pod reflection may have runtime cost for very large component counts

### Risks
- **pod API changes**: Kaiju nightly may change pod interface. Mitigation: abstraction layer — if pod changes, only the encode/decode bridge changes.
- **Save file size growth**: As content expands (VS: 4 regions, 2000+ entities), save files may exceed 10MB. Mitigation: monitor save size; add compression (gzip) if needed.
- **Steam Cloud sync failures**: Network issues, quota limits. Mitigation: local save always succeeds; cloud sync is best-effort with retry.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| Save/Load | CR-1: Save File Structure — header, metadata, entity table, components, world state, investigation | Binary sections encoded via pod per the format defined above |
| Save/Load | CR-2: Save Operations — manual, quick-save (F5), auto-save, load (F9), delete | `SaveManager` interface with slot-based operations |
| Save/Load | CR-3: Save Versioning — forward compatibility, migration functions | Version header in save file; registered migration functions per version transition |
| Save/Load | CR-6: Save File Location — platform-standard paths | Linux: `$XDG_DATA_HOME`, Windows: `%APPDATA%`, macOS: `~/Library/Application Support` |
| Save/Load | CR-7: Steam Cloud — Auto-Cloud, conflict resolution | Steamworks API integration; conflict dialog when local/cloud mismatch |
| Save/Load | CR-4: What Is NOT Saved (amended) | Camera state IS saved (overrides GDD exclusion). Transient visual state, audio positions, UI state, and combat turn state remain excluded per original CR-4. |
| Camera System | CR-13 / TR-cam-013: Camera save/load contract | Camera state (position, zoom, per-context prefs, base state, Manual flag, preemption stack) included in save file; restored on load per Camera GDD |

## Performance Implications
- **CPU**: Save: ~2 seconds for 500 entities (pod.Encode + disk write). Load: ~3 seconds (disk read + pod.Decode + entity reconstruction).
- **Memory**: Save buffers ~5MB peak during encode. No persistent memory overhead.
- **Load Time**: Scene load ~2 seconds (dominated by pod.Decode and entity creation). Save file read ~100ms (disk I/O).
- **Network**: Steam Cloud sync ~1-5 seconds per save (dependent on network conditions). Non-blocking — game continues during sync.

## Migration Plan
First implementation. No existing saves to migrate. Version 1 save format as specified. Migration functions will be added for v1→v2 when the format changes.

## Validation Criteria
- Save/load round-trip: 500 entities with all components → save → load → identical GameEntityIDs, positions, component values
- Checksum validation: flip one byte in save file → load detects corruption, offers backup restore
- Version migration: save with version N, load with version N+1 with registered migration → migrated data intact
- Steam Cloud offline: save/load functions normally without Steam runtime

## Related Decisions
- ADR-0001: Entity Architecture — component stores register with pod
- ADR-0003: Camera System — camera state serialization contract (amended 2026-08-12)
- ADR-0010: Thread Investigation — investigation state included in save
- ADR-0013: PWS Memory & Evolution — world state flags and memory log in save scope
- ADR-0014: Audio System — volume/mute settings persistence

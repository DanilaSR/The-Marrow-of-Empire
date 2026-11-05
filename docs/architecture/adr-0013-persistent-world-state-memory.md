# ADR-0013: Persistent World State — Memory Log & Evolution Rules

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Persistent World State GDD, ADR-0001, ADR-0011 |
| **Post-Cutoff APIs Used** | None — pure data structures |
| **Verification Required** | None — data model and rules are engine-agnostic |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — Persistent component), ADR-0011 (Event Bus — world state change events) |
| **Enables** | Thread Investigation (world state queries), NPC dialogue (memory log references) |
| **Blocks** | Persistent World State full implementation |
| **Ordering Note** | Must be Accepted before PWS implementation |

## Context

The Persistent World State system needs two capabilities not covered by existing ADRs: a structured memory log per entity (append-only, capped, with protected entries) and autonomous world evolution rules (condition → event, evaluated on region load). This ADR closes the two coverage gaps identified by architecture review (TR-pws-002, TR-pws-003).

## Decision

**Append-only memory log with soft cap (50 entries) and protected-entry mechanism. World evolution rules as YAML-authored condition→event pairs evaluated once on region load.**

### Memory Log

```go
type MemoryEntry struct {
    EventID       uint64
    EventType     string
    GameTimestamp float64
    ActorID       GameEntityID
    TargetID      GameEntityID
    RegionID      string
    Details       string // JSON blob
    Protected     bool   // never evicted by cap
}

type MemoryLog struct {
    Entries []MemoryEntry // oldest first
    MaxSize int           // default 50
}
```

- Append: `log.Entries = append(log.Entries, entry)`
- Eviction: if `len(log.Entries) > MaxSize`, remove oldest entry where `Protected == false`
- If all entries are protected, append anyway (soft cap, warning logged)
- Protected events: `npc_state_changed`, `discovery_event`, `quest_relevant` — marked by the submitting system
- Query: `GetMemoryLog(entityID, since, limit)` — returns entries newer than timestamp, newest first

### World Evolution Rules

```yaml
# Per-region YAML: assets/world_rules/{region_id}.yaml
rules:
  - name: "ruin_repopulation"
    condition: "world.ruin_cleared == true AND days_since(world.ruin_cleared_at) > 7"
    event:
      type: "location_repopulated"
      target: "{region_id}"
      details:
        occupant: "bandit_camp"
        difficulty: "medium"
```

Evaluated on region load with current world flags and game time. Rules fire once per region load — no re-evaluation loop. If a rule modifies a flag read by an earlier rule in the same pass, a warning is logged (circular dependency detection).

## GDD Requirements Addressed

| GDD | Requirement | How Addressed |
|-----|-------------|---------------|
| Persistent World State | CR-3: Memory Log — append-only, capped, protected entries | MemoryLog struct with eviction policy |
| Persistent World State | CR-5: World Evolution — condition → event, region load evaluation | YAML rules, single-pass evaluation |

## Performance Implications
Memory log: <0.1ms append, <1ms query (50 entries). Evolution rules: <1ms per region (20 rules × simple condition evaluation). Memory: ~2KB per entity log (50 entries × ~40 bytes).

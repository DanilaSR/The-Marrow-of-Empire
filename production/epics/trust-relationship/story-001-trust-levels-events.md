# Story 001: Trust Levels & Trust Events

> **Epic**: Trust & Relationship
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 5/5 passing
**Deviations**: ADVISORY — ADR-0011 priority table lists Trust at priority 3; story/manifest specify priority 1. Implementation follows story.
**Test Evidence**: Logic — `tests/unit/trust_relationship/trust_events_test.go` (18 tests, all PASS)
**Code Review**: Complete — APPROVED

## Context

**GDD**: `design/gdd/trust-relationship.md`
**Requirements**:
- `TR-trust-001` — Trust levels 0-3 per NPC: stranger/acquaintance/trusted/confidant — persisted in Persistent component
- `TR-trust-002` — Trust events: demonstrated_investigation, small_favor, thread_resolved, significant_help, vouched_for, secret_betrayed — each with defined trust gain/loss

**ADR Governing Implementation (primary)**: ADR-0005: NPC Knowledge Graph Data Model
**ADR (secondary)**: ADR-0011: Event Bus Architecture — trust events are emitted via EventBus for downstream systems to consume
**ADR Decision Summary**: Trust levels gate fact access in the Knowledge Graph (ADR-0005). Trust events are gameplay events emitted through the synchronous EventBus with system priority ordering (ADR-0011) — trust changes must be recorded BEFORE downstream systems (AI Dialogue) read the new trust level.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Trust system is pure Go data model + event processing. No Kaiju-specific APIs required. Persistence uses Entity Framework Persistent component (ADR-0001) — verify Persistent component field access API.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Required: Event subscribers register by event type + priority (1 = highest) — trust system subscribes at priority 1
- Forbidden: Never use direct cross-system calls for state mutations — all mutations go through the owning system's interface
- Guardrail: Event dispatch (5 subscribers) <1µs

---

## Acceptance Criteria

*From GDD `design/gdd/trust-relationship.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN an NPC at trust level 0, WHEN the player asks about a topic in the NPC's domain using knowledge from another NPC (demonstrated_investigation event), THEN trust increases to level 1.

- [ ] **AC-2**: GIVEN an NPC at trust level 2, WHEN the player betrays the NPC's secret to someone they wouldn't trust (secret_betrayed event), THEN trust drops to level 0 AND the trust freeze activates for 72 game hours.

- [ ] **AC-3**: GIVEN multiple trust events in one interaction, WHEN events would push trust from 0 to 2 (e.g., demonstrated_investigation + significant_help in the same interaction), THEN only the highest-value event applies — trust reaches level 1, not level 2.

- [ ] **Edge case: trust clamped at 3**: GIVEN an NPC at trust level 3, WHEN a trust-increasing event fires, THEN trust stays at 3 and the event is recorded but no level change occurs.

- [ ] **Edge case: trust clamped at 0**: GIVEN an NPC at trust level 0, WHEN a trust-decreasing event fires, THEN trust stays at 0.

---

## Implementation Notes

*Derived from GDD CR-1 (Trust Levels), CR-2 (Earning Trust), CR-3 (Losing Trust), CR-5 (Initial Trust), and ADR-0011:*

### Trust Level Model

```go
type TrustLevel int
const (
    TrustStranger     TrustLevel = 0
    TrustAcquaintance TrustLevel = 1
    TrustTrusted      TrustLevel = 2
    TrustConfidant    TrustLevel = 3
)

type TrustState struct {
    Level      TrustLevel
    FreezeUntil time.Time     // game time; zero when no freeze
    Events     []TrustEvent   // history of trust events
}
```

### Trust Events (CR-2, CR-3)

```go
type TrustEventType string
const (
    EventDemonstratedInvestigation TrustEventType = "demonstrated_investigation"
    EventSmallFavorCompleted       TrustEventType = "small_favor_completed"
    EventThreadResolvedInvolvingNPC TrustEventType = "thread_resolved_involving_npc"
    EventSignificantHelp           TrustEventType = "significant_help"
    EventVouchedFor                TrustEventType = "vouched_for"
    EventSecretDiscoveredAndKept   TrustEventType = "secret_discovered_and_kept"
    EventSecretBetrayed            TrustEventType = "secret_betrayed"
    EventCaughtLying               TrustEventType = "caught_lying"
    EventHostileAction             TrustEventType = "hostile_action"
    EventBrokenPromise             TrustEventType = "broken_promise"
    EventAssociationPenalty        TrustEventType = "association_penalty"
)

type TrustEvent struct {
    Type      TrustEventType
    Timestamp time.Time  // game time
    ThreadRef uint64     // optional: investigation thread that triggered this
}
```

### Trust Change Rules

**Gain events** — sequential, cannot skip levels. A +1 event at level 0 → level 1. A `significant_help` (+1 to level 2) at level 0 → level 1 only.

| Event | Max Level Unlocked | Effect |
|-------|-------------------|--------|
| `demonstrated_investigation` | 1 | +1 trust |
| `small_favor_completed` | 1 | +1 trust |
| `thread_resolved_involving_npc` | any | +1 trust (one-time per thread) |
| `significant_help` | 2 | +1 trust |
| `vouched_for` | 2 | +1 trust |
| `secret_discovered_and_kept` | 3 | +1 trust |

**Loss events** — sharper than gains, may trigger freeze:

| Event | Trust Loss | Freeze |
|-------|-----------|--------|
| `secret_betrayed` | −2 (min 0) | 72 game hours |
| `hostile_action` | −2 (min 0) | 72 game hours |
| `caught_lying` | −1 | — |
| `broken_promise` | −1 | — |
| `association_penalty` | −1 | — |

**Multiple events in one interaction**: Apply only the highest-value event (by absolute level change). Events don't stack in the same interaction.

**Trust freeze**: During the freeze period, all trust-increasing events are silently ignored (still recorded in `Events[]` but no level change). Trust-decreasing events still apply during freeze.

### Initial Trust (CR-5)

Default: level 0. Overrides configurable per NPC:
- Story-critical NPCs: configurable starting level (default 1)
- Party members: start at level 1
- Antagonists: cap at level 2 (cannot reach confidant)
- Faction enemies: cap at level 1 without faction shift

### EventBus Integration (ADR-0011)

The trust system subscribes to gameplay events (thread resolved, favor completed, secret betrayed) at priority 1. On receiving an event, it evaluates trust changes and emits `TrustChanged` events for downstream consumers (AI Dialogue Engine, NPC Knowledge Graph).

```go
// TrustSystem subscribes to game events and emits TrustChanged.
type TrustSystem struct {
    states   map[entity.GameEntityID]*TrustState
    bus      EventBus
    gameClock GameClock  // for freeze timestamps
}

// HandleEvent processes incoming gameplay events and applies trust rules.
func (ts *TrustSystem) HandleEvent(event GameEvent) []TrustChanged
```

### Persistence

Trust state stored per NPC via the Persistent component:
```go
// Save/Load round-trip: trust state serialized with NPC entity.
// MVP: store TrustLevel + FreezeUntil + recent Events (last 5) in Persistent flags.
```

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Relationship metadata (vouched_by[], betrayal_history[], knows_secret[], owes_favor), vouching mechanics, reputation bleed, betrayal bleed
- AI Dialogue Engine: Reading trust level for fact gating — this story provides the trust data
- Persistent World State: Full flag persistence API — this story uses the Entity Framework Persistent component directly
- UI: Portrait border glow indicators (Presentation layer)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1**: demonstrated_investigation → trust 0→1
- **AC-2**: secret_betrayed → trust 2→0 + 72h freeze
- **AC-3**: multiple events → only highest-value applies
- **Edge cases**: clamp at 3, clamp at 0, freeze blocks gains but not losses

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/trust_relationship/trust_events_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Entity Framework epic (Complete), NPC Knowledge Graph epic (Complete)
- Also depends on: ADR-0011 EventBus — the EventBus interface must exist (event_bus.go in src/entity/)
- Unlocks: Story 002 (Relationship Metadata & Social Propagation needs trust state machine)

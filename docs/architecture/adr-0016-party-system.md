# ADR-0016: Party System Architecture

## Status
Proposed

## Date
2026-08-12

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core |
| **Knowledge Risk** | LOW — pure game logic, no engine-specific APIs |
| **References Consulted** | `design/gdd/party-system.md`, `design/gdd/tactical-combat-engine.md`, `design/gdd/ai-dialogue-engine.md`, ADR-0001, ADR-0006, ADR-0009, ADR-0011 |
| **Post-Cutoff APIs Used** | None — pure game logic |
| **Verification Required** | None — data model and formulas are engine-agnostic |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — PartyState/PartyStatus components); ADR-0006 (Character Stats — reads attributes for formation/scout/AI); ADR-0011 (Event Bus — recruitment/departure/interjection events); ADR-0002 (Save/Load — party state persistence); ADR-0009 (Combat — reads active party roster, provides behavior-mode contract); ADR-0008 (AI Dialogue — interjection interface) |
| **Enables** | Character & Inventory UI (party management screen), Combat UI (party portraits), Question System (companion knowledge gating) |
| **Blocks** | Party System implementation — 6 TRs uncovered before this ADR |
| **Ordering Note** | Must be Accepted before Party System stories. Requires ADR-0009 amendment for companion knockout state (design gap T0-3) and ADR-0008 amendment for the interjection interface (design gap T0-2). |

## Context

### Problem Statement

The Party System is a Vertical Slice-tier feature system with 6 technical requirements (TR-party-002 through TR-party-007) that have no architectural coverage. It sits at the intersection of four systems — Entity Framework (companion entities), Trust & Relationship (recruitment/departure thresholds), Tactical Combat Engine (roster + AI behavior modes), and AI Dialogue Engine (interjection interface) — and needs clear data ownership, state machine, and cross-system contracts before any of those systems can implement party-related code. Two known cross-document conflicts (consistency-failures.md T0-2 interjection, T0-3 knockout state) block implementation and must be resolved by this ADR.

### Current State

No party code exists. Companion entities are NPC entities with Stats, Knowledge, and Dialogue components (ADR-0001 component catalog). The Party System GDD defines the full design: 4-state companion lifecycle, 3 AI behavior modes, scout designation with opposed detection checks, formation rules, and knowledge contribution. The GDD's contracts require: TCE must implement the mode→target-selection translation; ADE must implement the interjection interface; Question System must gate domain access on companion presence.

### Constraints

- Companion entities are NPC entities — party state must not fork the entity model (ADR-0001: game layer wraps Kaiju entities, typed component stores)
- Trust values are owned by Trust & Relationship (ADR-0011) — the Party System reads, never writes, trust
- Combat roster and AllyTurn AI decisions are owned by TCE (ADR-0009) — the Party System provides configuration, not combat logic
- Conversation flow is owned by ADE (ADR-0008) — interjections must be a request/response contract, not Party System code running inside dialogue
- Party management is gated behind `rest_point` entities (Living Environment) — the UI must never be available mid-combat/conversation/transition
- VS tier ships knockout, not permadeath, for companions

### Requirements

- TR-party-002: Companion state machine (Unrecruited/InParty/InPool/Departed with persisted cause)
- TR-party-003: AI behavior modes (Aggressive/Defensive/Passive) — set during exploration, persist across save/load
- TR-party-004: Scout system — single designation, WIS-based detection range, opposed d20 checks, staged alert
- TR-party-005: Formation — default loose diamond, trailing arc, front/back rank config, combat deployment positions
- TR-party-006: Companion knowledge contribution — domain availability gating + interjection interface (Hard ADE contract)
- TR-party-007: Rest-point gating — party management only at rest_point entities
- All party state must round-trip through Save/Load (ADR-0002 pod serialization)
- Performance: formation updates allocation-free, scout checks ticked (not per-frame), interjection scoring ≤5ms

## Decision

**The Party System is a state-management system with two components on existing entities: `PartyState` (on the player entity — the party's authoritative roster) and `PartyStatus` (on each companion entity — that companion's lifecycle state). All behavior modes, formation, and scout configuration live in `PartyState`; all per-companion lifecycle data lives in `PartyStatus`. The Party System is the sole writer of both components; TCE, ADE, Question System, and UI read them. Cross-system interactions use Event Bus events and request/response contracts — the Party System never runs code inside another system's update loop.**

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ PARTY SYSTEM (sole writer of PartyState + PartyStatus)       │
│                                                              │
│  PartyState (on player entity)          PartyStatus (per     │
│  ├─ active_roster[] GameEntityID        companion entity)    │
│  ├─ pool[] GameEntityID                 ├─ state: enum       │
│  ├─ formation: front_rank[]/back_rank[] ├─ cause: enum       │
│  ├─ behavior_modes: map[id]Mode         └─ joined_at: gametime│
│  ├─ scout_id GameEntityID (or None)                          │
│  └─ home_region_stash: map[id]stash    ── serialized via pod  │
│                                                              │
│  Events (ADR-0011):                                          │
│    party_member_recruited / party_member_departed{cause}     │
│    party_formation_changed / party_behavior_mode_changed     │
│    scout_alerted{entity, position, confidence}               │
│    companion_interjected{companion, domain}                  │
└──────────┬───────────────────────────────────────────────────┘
           │ reads (never writes)
     ┌─────┴──────────────────────────────┐
     │ TCE: roster → turn order; mode →   │
     │   AllyTurn target/ability rules;   │
     │   formation → deployment positions │
     │ ADE: CompanionContext →            │
     │   interjection eligibility + async │
     │   LLM fallback lines               │
     │ Question System: domain gating on  │
     │   active presence (exclusive:true) │
     │ Trust & Relationship: writes trust │
     │   → Party reads thresholds         │
     │ C&I UI: management screen reads    │
     └────────────────────────────────────┘
```

### Key Interfaces

```go
// package party

// Component stores registered with EF pod (ADR-0001 component catalog).
type PartyState struct {
    ActiveRoster  []GameEntityID `pod:"active_roster"`   // player always [0]
    Pool          []GameEntityID `pod:"pool"`
    FrontRank     []GameEntityID `pod:"front_rank"`
    BackRank      []GameEntityID `pod:"back_rank"`
    BehaviorModes map[GameEntityID]BehaviorMode `pod:"behavior_modes"`
    ScoutID       GameEntityID   `pod:"scout_id"`        // NoneID = no scout
}

type PartyStatus struct {
    State  PartyStateEnum `pod:"state"`   // Unrecruited|InParty|InPool|Departed
    Cause  DepartureCause `pod:"cause"`   // trust_betrayal|story|death|dismissed|none
    JoinedAt GameTime     `pod:"joined_at"`
}

type BehaviorMode uint8
const (ModeDefensive BehaviorMode = iota; ModeAggressive; ModePassive)

// Request/response contracts (implemented by consuming systems):
type PartyReader interface {          // implemented by Party System
    ActiveRoster() []GameEntityID
    IsActive(id GameEntityID) bool
    CompanionContext() []CompanionContext   // for ADE interjection scoring
    FormationPositions() []FormationSlot    // for TCE deployment
}

// ADE contract (resolves design gap T0-2):
type InterjectionRequest struct {
    ConversationID uint64
    Topic          TopicScore  // domain match scores from Question System
    // ADE calls: party.EvaluateInterjections(req) → []CompanionCandidate
    // then issues companion_interjected event with the chosen companion
}

// TCE contract (resolves design gap T0-3):
// TCE reads behavior_modes and applies mode→target rules in AllyTurn.
// Companion at 0 HP with party.KnockoutEnabled() == true (VS default)
// → KO state: unconscious until combat end, revive at 1 HP.
```

### Implementation Guidelines

1. **Component ownership**: PartyState lives on the player entity; PartyStatus on companions. Both register with `pod` (ADR-0001 serialization). The Party System is the sole writer — all mutations go through `party.Recruit()`, `party.Swap()`, `party.Dismiss()`, `party.SetMode()`, `party.SetScout()`, `party.SetRank()`.
2. **Recruitment flow**: ADE raises `party_recruit_request` event on dialogue trigger → Party System validates (trust ≥ threshold via Trust & Relationship read, discovery_trigger satisfied, slot available) → transitions state → emits `party_member_recruited`. Validation failures return a coded reason the ADE maps to in-character dialogue.
3. **Departure flow**: Trust & Relationship writes trust; Party System subscribes to `trust_changed` events and evaluates `trust_level <= departure_threshold` → emits `party_member_departed{cause}` with combat/conversation deferral (deferral queue processed on `combat_ended` / `conversation_ended` events). Death transition consumed from TCE `entity_died` lifecycle event.
4. **Behavior modes**: `SetMode()` allowed only outside combat. TCE reads modes at AllyTurn start. Scout designation independent of mode; scout+Aggressive precedence per GDD CR-5.
5. **Scout**: detection runs on a 500ms tick (not per frame) — `scout_detection_range` computed from WIS_mod, opposed d20 checks, 180° cone, disadvantage flags from Living Environment light state. Alert is staged: emit `scout_alerted`, hold scout, wait for player choice.
6. **Formation**: procedural slot computation from party size + ranks (diamond at 4, trailing arc otherwise) at `formation_spacing` 3u. Positions are pure functions of roster + formation config — deterministic and unit-testable. TCE queries `FormationPositions()` at encounter start.
7. **Knowledge gating**: Question System queries `IsActive(id)` before offering companion domains; `exclusive: true` fragments gated on active presence per GDD CR-6. ADE pulls `CompanionContext()` (active companions + domains + trust) when building the prompt.
8. **Interjection**: ADE calls `EvaluateInterjections(req)` at response-choice points only (never mid-sentence). Party System returns candidates ranked by domain confidence ≥0.7, then trust, then WIS. Caps: 1/companion/conversation, 3/conversation total. Thread-resolution witness reactions are exempt from caps (guaranteed, triggered by `thread_resolved` event).
9. **Rest-point gating**: Party management UI availability is a query `party.ManagementAvailable()` — true only when player entity is near a `rest_point` entity AND no combat/conversation/transition is active. The UI polls this; the Party System rejects mutation calls when false (defense in depth).
10. **Save/Load**: both components serialize via pod; departure `cause` persists; formation/modes/scout config restore exactly (ADR-0002).

## Alternatives Considered

### Alternative 1: Party roster as a plain Go singleton service (no components)

- **Description**: Party state held in a module-level struct, not in EF components.
- **Pros**: Simplest implementation; no pod registration; fast access.
- **Cons**: Violates ADR-0001 ("no game state lives outside component stores"); breaks save/load round-trip unless bespoke serialization is written; invisible to EF queries (targeting, UI feeds); write-arbitration bypass.
- **Estimated Effort**: 0.5× chosen approach initially, 2× when save/load and UI feeds land.
- **Rejection Reason**: ADR-0001's save-everything principle is a Foundation-layer invariant; the Party System must not be the exception.

### Alternative 2: Party state distributed entirely across companion entities (no player-side PartyState)

- **Description**: Each companion's PartyStatus carries everything; roster = EF query over `pc_controlled` tag.
- **Pros**: No aggregate component; per-companion writes are localized.
- **Cons**: Roster ordering, formation ranks, scout designation, and behavior modes are party-level facts — deriving them from per-entity tags is a query every consumer must repeat; race conditions on roster size validation (two companions recruited same frame); no single serialization point for party-level config.
- **Estimated Effort**: 1.5× chosen approach.
- **Rejection Reason**: Party-level configuration (formation, modes, scout) needs a single authoritative writer and a single source of truth; per-entity state only covers lifecycle, which is exactly what PartyStatus handles.

### Alternative 3: Full AI squad controller (per-companion behavior trees)

- **Description**: Each companion runs a behavior tree or utility AI continuously in exploration.
- **Pros**: Emergent behavior; richer simulation.
- **Cons**: Massive scope beyond VS; 3-mode design is explicit in the GDD; behavior trees would fight the TCE turn structure; unpredictable for QA.
- **Estimated Effort**: 5× chosen approach.
- **Rejection Reason**: The GDD's 3-mode model is a deliberate scope decision; the Party System's job is configuration and state, not autonomous behavior authoring.

## Consequences

### Positive

- Clear ownership: Party System = party state; TCE = combat behavior; ADE = conversation; no cross-writes
- All party state survives save/load through existing pod serialization (ADR-0002) with zero bespoke code
- Resolves both tracked cross-document conflicts (T0-2 interjection interface, T0-3 knockout state) with concrete contracts
- Deterministic formation/scout formulas are directly unit-testable per acceptance criteria

### Negative

- ADE must implement the interjection interface (scope added to ADR-0008); TCE must implement knockout + mode translation (scope added to ADR-0009)
- Two components add serialization surface (~1KB per save) and two more event types to the Event Bus
- Departure deferral queue adds a small state machine (combat/conversation deferral) that must be tested for all 4 causes

### Neutral

- Pool companions "wait at rest point" — their world presence is abstracted (not simulated); only `InParty` companions are physically present. This matches GDD CR-1 but means pool companions cannot be killed by world events at VS (edge case deferred to Full Vision).

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| TCE knockout amendment scope creeps into Full Vision permadeath | MEDIUM | MEDIUM | Knockout only (revive at 1 HP) at VS; `companion_death_enabled` flag defaults false, permadeath path deferred |
| Interjection LLM fallback latency disrupts conversation pacing | MEDIUM | LOW | Async generation with pause indicator per GDD CR-6; authored-line-first path avoids LLM in most cases |
| Roster size validation races (recruit while full) | LOW | LOW | All mutations funnel through Party System methods (single-threaded main loop per ADR-0001) |
| Formation positions collide with scene geometry | MEDIUM | LOW | ±1u tolerance in acceptance criteria; TCE deployment clamps to nearest valid tile |

## Performance Implications

| Metric | Before | Expected After | Budget |
|--------|--------|---------------|--------|
| CPU (frame time) | 0ms | Formation recompute ≤0.1ms on roster change only (cached) | 16.6ms frame |
| Scout tick | 0ms | ≤0.5ms per 500ms tick (1 scout, opposed check) | ticked, not per-frame |
| Interjection evaluation | 0ms | ≤5ms at response-choice points (≤4 companions × domain scoring) | conversation-time, not frame-critical |
| Memory | 0KB | ~1KB PartyState + ~64B per companion PartyStatus | 1MB camera-class budget irrelevant |

## Migration Plan

New system — no existing code to migrate.

1. Add PartyState/PartyStatus to EF component catalog (ADR-0001) — pod registration
2. Implement Party System core (state machine, recruitment/departure validation, events)
3. Amend ADR-0009: companion knockout state + mode→AllyTurn translation
4. Amend ADR-0008: interjection interface (EvaluateInterjections + async fallback)
5. Wire Question System domain gating + C&I UI management screen

**Rollback plan**: Party System is additive; removing it returns companions to plain NPC behavior. No data migration needed — components are new.

## Validation Criteria

- [ ] Recruitment/departure/dismiss state transitions match GDD state table for all 4 causes (unit tests)
- [ ] Scout detection range formula verified at WIS 16 (+3 → 16u) and WIS 10 (→ 10u) with debug readout
- [ ] Formation positions deterministic for party sizes 2-6; round-trip through save/load exactly
- [ ] Interjection caps enforced (1/companion, 3/conversation); thread-resolution witness reactions exempt
- [ ] Party management rejected during combat/conversation/transition (AC-PS-12)
- [ ] Save/load round-trip preserves roster, pool, modes, scout, formation, Departed cause
- [ ] Behavior modes persist across save/load and translate correctly in AllyTurn

## GDD Requirements Addressed

| GDD Document | System | Requirement | How This ADR Satisfies It |
|-------------|--------|-------------|--------------------------|
| `design/gdd/party-system.md` | Party System | TR-party-002: Companion state machine (Unrecruited/InParty/InPool/Departed, persisted cause) | PartyStatus component with state + cause enums; transitions via Party System methods; deferral queue for combat/conversation |
| `design/gdd/party-system.md` | Party System | TR-party-003: AI behavior modes (Aggressive/Defensive/Passive) | BehaviorModes map in PartyState; SetMode gated to exploration; TCE reads modes in AllyTurn |
| `design/gdd/party-system.md` | Party System | TR-party-004: Scout system (WIS detection, opposed checks, staged alert) | ScoutID in PartyState; 500ms tick; d20 opposed checks; scout_alerted event |
| `design/gdd/party-system.md` | Party System | TR-party-005: Formation (diamond, trailing arc, ranks) | FrontRank/BackRank in PartyState; procedural slot computation; FormationPositions() consumed by TCE |
| `design/gdd/party-system.md` | Party System | TR-party-006: Companion knowledge contribution + interjection interface | CompanionContext() contract; EvaluateInterjections() with caps; exclusive-fragment gating via Question System |
| `design/gdd/party-system.md` | Party System | TR-party-007: Rest-point gating | ManagementAvailable() query + Party System mutation guard |
| `design/gdd/party-system.md` | Party System | TR-party-001: Roster (max 6), recruitment via trust/dialogue, dismissal, combat participation | ActiveRoster/Pool in PartyState; recruitment validation flow; roster consumed by TCE |

## Related Decisions

- ADR-0001 (Entity Architecture) — components live in typed stores, pod-registered
- ADR-0006 (Character Stats) — attributes read for formation, scout, AI
- ADR-0009 (Combat Engine) — roster + behavior-mode contract; knockout amendment required
- ADR-0008 (LLM Inference) — interjection interface amendment required
- ADR-0011 (Event Bus) — recruitment/departure/alert/interjection events
- ADR-0002 (Save/Load) — PartyState/PartyStatus pod serialization

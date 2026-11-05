# Story 002: Relationship Metadata & Social Propagation

> **Epic**: Trust & Relationship
> **Status**: Ready
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (5–7 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/trust-relationship.md`
**Requirements**:
- `TR-trust-003` — Vouching: NPC at trust 2+ can vouch for player to another NPC; trust propagation via social networks
- `TR-trust-001` — Trust levels persisted with relationship metadata (vouched_by[], betrayal_history[], knows_secret[])

**ADR Governing Implementation (primary)**: ADR-0011: Event Bus Architecture
**ADR (secondary)**: ADR-0005: NPC Knowledge Graph Data Model — relationship metadata flows into KG queries for AI Dialogue prompt context
**ADR Decision Summary**: Cross-system communication via synchronous EventBus with priority ordering (ADR-0011). Vouching and reputation propagation are multi-NPC operations that must complete atomically before downstream systems read the updated state. Relationship metadata is persisted per NPC and fed to the AI Dialogue Engine via the Knowledge Graph query pipeline (ADR-0005).

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: MEDIUM
**Engine Notes**: Pure Go data model. No Kaiju-specific APIs. Social graph traversal uses entity registry for NPC lookups. Metadata persistence via Persistent component (ADR-0001).

**Control Manifest Rules (Feature Layer)**:
- Required: Event-driven thread graph — auto-chained by entity overlap (ADR-0010 — trust events may create investigation threads)
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: All event handlers combined <2ms/frame

---

## Acceptance Criteria

*From GDD `design/gdd/trust-relationship.md`, scoped to this story:*

- [ ] **AC-4**: GIVEN an NPC at trust level 2+, WHEN the player asks them to vouch for them to another NPC, THEN the target NPC gains +1 trust (to level 1 if at 0).

- [ ] **AC-5**: GIVEN an NPC with trust level 0 and `betrayal_history` non-empty, WHEN the player first meets them, THEN the AI Dialogue Engine receives the betrayal metadata and the NPC is guarded in conversation.

- [ ] **AC-6**: GIVEN the player at trust 2+ with 3 NPCs in the same faction, WHEN the player meets a new NPC in that faction, THEN the new NPC starts at trust level 1 (reputation bleed).

- [ ] **Edge case: vouching validity after trust drop**: GIVEN NPC A vouched for the player to NPC B, WHEN NPC A's trust later drops below 2, THEN the vouching remains valid (it was genuine at the time), but NPC B receives a `voucher_trust_dropped` flag for dialogue context.

- [ ] **Edge case: betrayal bleed**: GIVEN the player betrays an NPC's secret and that NPC has vouched for the player to 2 other NPCs, WHEN the betrayal occurs, THEN those 2 NPCs also lose 1 trust level.

---

## Implementation Notes

*Derived from GDD CR-4 (Relationship Metadata), CR-6 (Trust Propagation), and ADR-0011:*

### Relationship Metadata (CR-4)

```go
type Relationship struct {
    NPCID           entity.GameEntityID
    TrustLevel      TrustLevel  // from Story 001
    VouchedBy       []entity.GameEntityID  // who vouched for player to this NPC
    VouchedFor      []entity.GameEntityID  // who this NPC vouched for player to
    KnowsSecret     []uint64               // fact IDs player has discovered about this NPC
    BetrayalHistory []BetrayalRecord       // secrets player betrayed
    OwesFavor       bool
    PlayerOwesFavor bool
}

type BetrayalRecord struct {
    FactID    uint64
    Timestamp time.Time  // game time
}
```

### Vouching (CR-6)

The player can request vouching from an NPC at trust 2+. The vouching NPC must be at trust level 2+ with the player. The target NPC gains +1 trust (sequential — cannot skip levels). Vouching is recorded in both NPCs' relationship metadata.

```go
// Vouch requests an NPC to vouch for the player to a target NPC.
// Returns error if voucher trust < 2 or target is already at max.
func (rs *RelationshipSystem) Vouch(voucherID, targetID entity.GameEntityID) error
```

### Reputation Bleed (CR-6)

When the player reaches trust 2+ with N NPCs in the same faction (configurable threshold, default 3), new NPCs in that faction initialize at trust level 1 instead of 0. Recalculated when trust changes — triggered by `TrustChanged` events from Story 001.

### Betrayal Bleed (CR-6)

When a `secret_betrayed` event fires against an NPC who has vouched for the player, all NPCs in that NPC's `vouched_for[]` list receive −1 trust. This is a cascading check triggered by the trust event from Story 001.

### Metadata Flow to AI Dialogue

Relationship metadata is exposed via the NPC Knowledge Graph query pipeline. When the AI Dialogue Engine queries facts for prompt construction (CR-3 step 9), it also receives:
- `vouched_by` → "Orin spoke well of them"
- `betrayal_history` → "They once shared your secret with the wrong person"
- `knows_secret[]` → which secrets the player already knows (for "as you know..." framing)

This is a read path — the relationship system writes metadata; the KG query reads it. No direct coupling.

### Event Flow

```
Game Event (e.g., thread_resolved)
  → TrustSystem.HandleEvent (Story 001) — computes trust change
    → Emits TrustChanged event
      → RelationshipSystem — updates metadata, checks reputation/betrayal bleed
      → Knowledge Graph — updates trust_level for fact gating
      → AI Dialogue Engine — receives updated trust + metadata for next prompt
```

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Trust level state machine, trust events (gain/loss/freeze), sequential level enforcement, initial trust rules
- AI Dialogue Engine: Reading metadata for prompt construction — this story provides the data
- Persistent World State: Full persistence API — uses Entity Framework Persistent component directly (same as Story 001)
- UI: Portrait border glow indicators, trust level display

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-4**: Vouching: trust 2+ NPC vouches → target +1 trust
- **AC-5**: Betrayal metadata → guarded NPC conversation
- **AC-6**: Reputation bleed: 3+ faction NPCs at trust 2+ → new NPC starts at level 1
- **Edge cases**: vouching validity after trust drop, betrayal bleed propagation

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/trust_relationship/social_propagation_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Trust Levels & Trust Events — the trust state machine must be DONE)
- Also depends on: Entity Framework epic (Complete), NPC Knowledge Graph epic (Complete), EventBus (ADR-0011)
- Unlocks: AI Dialogue Engine Story 002 (Prompt Builder needs relationship metadata for prompt context)

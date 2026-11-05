# Sprint 4 — AI & World Foundation

**Dates**: 2026-08-11 to 2026-08-17 (5 days)
**Milestone**: Production — Core Layer

## Sprint Goal
Establish the NPC Knowledge Graph — the data model powering AI-driven dialogue — and the Living Environment's game clock. These are the two Core systems that everything in the AI & World layers depends on.

## Capacity
- Total: 5 days (solo developer, ~30 hours)
- Buffer (20%): 6 hours reserved for YAML format iteration and game clock verification
- Available: ~24 hours

> **Estimation note**: Sprint 3 velocity was ~6 stories/day with subagent parallelism. Sprint 4 estimates use the same calibrated sizes. KG stories carry MEDIUM engine risk (pure Go data structures, no Kaiju APIs). ENV stories carry LOW engine risk (pure math clock, no rendering).

## Tasks

### Must Have (Critical Path)

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| KG-001 | Knowledge Domain & Fact Model | ai-programmer | 4 | EF-001, EF-002 | Fact types (FACT/RUMOR/OPINION/SECRET) defined as Go types; knowledge domains as named string categories; fact struct with trust_required (0-3), entity references, and content fields; domain exclusion (knows_not) per NPC |
| KG-002 | YAML .kg File Parser | ai-programmer | 4 | KG-001 | Parse YAML .kg files from `assets/npc_knowledge/{region}/{npc_id}.kg`; validate fact types and trust levels; error on malformed files with line numbers; parse time <10ms per NPC (ADR-0005) |
| KG-003 | Trust Gating & Query Interface | ai-programmer | 4 | KG-001, KG-002 | Query facts by domain + trust_level; facts with trust_required > current_trust excluded; query by entity reference (GameEntityID); query performance <0.5ms (ADR-0005) |
| ENV-001 | Game Clock & Light States | gameplay-programmer | 4 | EF-001 | 24-hour clock, 10:1 time ratio (2.4 real hours = 1 game day), starts 8:00 AM Day; five light states (Dawn 4-7, Day 7-17, Dusk 17-20, Night 20-2, Deep Night 2-4); clock update <0.01ms/frame (ADR-0007) |

### Should Have

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| KG-004 | Entity References & Knowledge Boundaries | ai-programmer | 2 | KG-003 | Facts reference entities by GameEntityID; cross-referencing queries: "what does NPC X know about entity Y?"; knowledge boundaries enforced: knows_not domains and knows_not_entities per NPC |
| KG-005 | Knowledge Component Integration | ai-programmer | 4 | KG-003, EF-002 | Knowledge component (from EF-002) populated from parsed .kg files; component queries return domain + fact data; trust_level persisted via Persistent component |
| ENV-002 | Wait / Pass-Time Mechanic | gameplay-programmer | 4 | ENV-001 | Interact with rest_point → select wake time → 100× fast-forward with fade transition; clock pauses during combat, conversation, menus, journal |
| SAVE-003 | Save File Corruption Recovery | gameplay-programmer | 3 | SAVE-001 | CRC32 checksum verification on load; corrupted save detected → attempt .bak restore; restore success → game loads from backup; restore failure → error with slot info; corrupted save never overwrites .bak |

### Nice to Have

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| ASSET-001 | FBX Import & Normalization | technical-artist | 4 | — | KitBash3D Dark Fantasy FBX imports via staging pipeline; scale normalized (1u = 1m); rotation corrected (Z-up → Y-up); reference object validation; poly count ≤5k static / ≤20k hero |
| STAT-004 | Equipment Stat Modification | gameplay-programmer | 1.5 | STAT-001 | Items modify attributes via equipment slots; stat bonuses applied on equip, removed on unequip; derived stats recalculated after modification |

## Carryover from Previous Sprint

| Task | Reason | New Estimate |
|------|--------|-------------|
| STAT-004 Equipment Stats | Should-have Sprint 3, not started | 1.5 hrs (unchanged) |
| ASSET-001 FBX Import | Nice-to-have Sprint 3, not started | 4 hrs (unchanged) |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| .kg YAML format requires iteration | Medium | Low | ADR-0005 defines the format; parser is straightforward YAML unmarshaling. Budget buffer time for format tweaks |
| Game clock drift over long sessions | Low | Medium | 10:1 ratio is simple integer math; verify with 24h simulation test (2.4 real hours → 24 game hours) |
| KG dependency chain is fully serial | High | Low | KG-001→002→003 must be sequential by design. ENV and SAVE stories run in parallel on separate tracks |
| Trust gating design needs playtesting | Low | Medium | Trust levels (0-3) are simple thresholds; query filtering is a single comparison. Real tuning happens during playtesting, not implementation |
| KitBash3D asset import complexity | High | Medium | Marked Nice to Have for risk management; engine API verification required before ASSET-001 |

## Dependencies on External Factors
- Kaiju Engine source in `kaiju/` must be readable (KG stories are pure Go, no engine APIs)
- YAML parsing: Go standard library `gopkg.in/yaml.v3` (no engine dependency)
- KitBash3D Dark Fantasy assets at `/home/danila/Downloads/KitBash3D - Dark Fantasy/` (ASSET-001)
- KitBash3D Enchanted assets at `/home/danila/Downloads/KitBash3D - Enchanted/` (ASSET-004)

## Definition of Done for this Sprint
- [ ] All 4 Must Have stories completed
- [ ] All acceptance criteria verified
- [ ] `go test ./...` passes from repo root (all Sprint 1–4 tests)
- [ ] NPC Knowledge Graph: .kg files parseable, facts queryable by domain + trust
- [ ] Game clock: 24-hour cycle completes correctly (2.4 real hours = 1 game day)
- [ ] QA plan exists (`production/qa/qa-plan-sprint-4.md`)
- [ ] Code reviewed and committed per-story

> ⚠️ **No QA Plan**: Run `/qa-plan sprint` before the last story is implemented.
> The Production → Polish gate requires a QA sign-off report, which requires a QA plan.

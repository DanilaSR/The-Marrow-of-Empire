# Epic: Trust & Relationship

> **Layer**: Feature | **GDD**: trust-relationship.md | **Status**: Ready | **Stories**: 2 stories
> **Governing ADRs**: ADR-0005, ADR-0011 | **Engine Risk**: HIGH

## Overview
Implement NPC trust system: 0-3 trust levels per NPC, trust events (demonstrated_investigation, small_favor, significant_help, vouched_for, secret_betrayed), trust freeze on betrayal, vouching and trust propagation through social networks, and relationship metadata (trust_events[], vouched_by[], betrayal_history[]). Persisted via Entity Framework Persistent component.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-trust-001 | Trust levels 0-3 | ADR-0005 ✅ |
| TR-trust-002 | Trust events | ADR-0011 ✅ |
| TR-trust-003 | Vouching & propagation | ADR-0011 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Trust Levels & Trust Events | Logic | Complete | ADR-0005, ADR-0011 |
| 002 | Relationship Metadata & Social Propagation | Integration | Ready | ADR-0011, ADR-0005 |

## Next Step
Run `/story-readiness production/epics/trust-relationship/story-001-trust-levels-events.md`

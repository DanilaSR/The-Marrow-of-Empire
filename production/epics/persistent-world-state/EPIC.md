# Epic: Persistent World State

> **Layer**: Feature | **GDD**: persistent-world-state.md | **Status**: Ready | **Stories**: 6 stories
> **Governing ADRs**: ADR-0001, ADR-0013 | **Engine Risk**: HIGH

## Overview
Implement the world memory system: typed flag store per entity (world.* namespace), append-only memory log with soft cap (50 entries) and protected events, region state tracking, world evolution rules (YAML-authored condition→event, evaluated on region load), and clue-path protection for quest-relevant entities. VS tier.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-pws-001 | World flag store | ADR-0001 ✅ |
| TR-pws-002 | Memory log | ADR-0013 ✅ |
| TR-pws-003 | World evolution rules | ADR-0013 ✅ |
| TR-pws-004 | Clue-path protection | ADR-0001 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | World Flag Store & Query API | Logic | Ready | ADR-0001 |
| 002 | Event Submission Pipeline | Integration | Ready | ADR-0001, ADR-0011 |
| 003 | Memory Log | Logic | Ready | ADR-0013 |
| 004 | Clue-Path Protection | Integration | Ready | ADR-0001 |
| 005 | World Evolution Rules | Logic | Ready | ADR-0013 |
| 006 | Save/Load & Performance | Integration | Ready | ADR-0001, ADR-0013 |

## Next Step
Run `/story-readiness production/epics/persistent-world-state/story-001-world-flag-store-query-api.md` to begin implementation.

# Epic: Thread Investigation

> **Layer**: Feature | **GDD**: thread-investigation.md | **Status**: Ready | **Stories**: 5 stories
> **Governing ADRs**: ADR-0010 | **Engine Risk**: HIGH

## Overview
Implement the quest-log replacement: event-driven thread graph with auto-creation from NPC fact references, auto-chaining by entity overlap, contradiction detection (fact A contradicts fact B → create thread + emit event), revelation threshold (3+ connected facts → chain revelation_ready), and PlayerKnowledge component persistence. Pillar 4 delivery mechanism.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-thread-001 | Thread creation from facts | ADR-0010 ✅ |
| TR-thread-002 | Auto-chaining | ADR-0010 ✅ |
| TR-thread-003 | Contradiction detection | ADR-0010 ✅ |
| TR-thread-004 | Revelation threshold | ADR-0010 ✅ |
| TR-thread-005 | PlayerKnowledge persistence | ADR-0010 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Thread Data Model & Auto-Creation | Logic | Complete | ADR-0010 |
| 002 | Auto-Chaining & Entity Overlap | Logic | Complete | ADR-0010 |
| 003 | Contradiction Detection | Logic | Complete | ADR-0010, ADR-0011 |
| 004 | Revelation Threshold & Resolution | Logic | Complete | ADR-0010, ADR-0011 |
| 005 | PlayerKnowledge Persistence & Integration | Integration | Complete | ADR-0010, ADR-0001, ADR-0002 |

## Next Step
Run `/story-readiness production/epics/thread-investigation/story-003-contradiction-detection.md`

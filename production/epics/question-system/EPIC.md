# Epic: Question System

> **Layer**: Feature | **GDD**: question-system.md | **Status**: Ready | **Stories**: 2 stories
> **Governing ADRs**: ADR-0005 | **Engine Risk**: HIGH

## Overview
Implement the player input system for NPC conversation: free-text input with topic matching (keyword overlap + entity reference detection → domain relevance scoring), topic suggestion chips (generated from NPC domains + active threads + recent entities), conversation history tracking, and question routing to the AI Dialogue Engine.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-q-001 | Free-text topic matching | ADR-0005 ✅ |
| TR-q-002 | Domain keyword scoring | ADR-0005 ✅ |
| TR-q-003 | Topic suggestion generation | ADR-0005 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Topic Matcher & Domain Scoring | Logic | Complete | ADR-0005 |
| 002 | Question Input, Suggestions & State Machine | Integration | Ready | ADR-0005 |

## Next Step
Run `/story-readiness production/epics/question-system/story-002-question-input-suggestions-state-machine.md`

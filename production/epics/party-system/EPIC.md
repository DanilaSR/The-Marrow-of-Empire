# Epic: Party System

> **Layer**: Feature | **GDD**: party-system.md | **Status**: Ready | **Stories**: 14 (5 Ready, 9 Blocked — Party ADR GAP)
> **Governing ADRs**: ADR-0006, ADR-0011 | **Engine Risk**: HIGH | **Tier**: VS

## Overview
Implement party management: roster (max 6 — 1 player + 5 companions), recruitment via trust/dialogue, dismissal with confirmation, companion combat participation, and party member stats display. VS tier.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-party-001 | Party roster, recruitment, dismissal | ADR-0006, ADR-0011 ✅ |
| TR-party-002 | Companion state machine (4 states, persisted cause, deferral, death-block) | GAP — needs Party ADR ⛔ |
| TR-party-003 | Companion AI behavior modes + TCE translation contract | GAP — needs Party ADR or ADR-0009 extension ⛔ |
| TR-party-004 | Scout system (designation, detection formula, opposed checks, staged alert) | GAP — needs Party ADR ⛔ |
| TR-party-005 | Formation (loose diamond, arcs, ranks, combat deployment) | GAP — needs Party ADR ⛔ |
| TR-party-006 | Companion knowledge contribution + interjection contract | GAP — needs Party ADR + ADR-0008 revision ⛔ |
| TR-party-007 | Rest-point gating + party management screen | GAP — needs Party ADR ⛔ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Party Roster & Active Slots | Logic | Ready | ADR-0006, ADR-0011 |
| 002 | Companion State Machine | Logic | Blocked | GAP — Party ADR |
| 003 | Recruitment via Trust & Dialogue | Integration | Ready | ADR-0011, ADR-0006 |
| 004 | Swap & Dismissal at Rest Points | Integration | Ready | ADR-0011, ADR-0006 |
| 005 | Re-recruitment Rules | Logic | Ready | ADR-0006 |
| 006 | Departure & Death Handling | Integration | Blocked | GAP — Party ADR |
| 007 | Combat Roster Integration | Integration | Ready | ADR-0011, ADR-0009 |
| 008 | Companion AI Behavior Modes (Exploration) | Logic | Blocked | GAP — Party ADR / ADR-0009 ext |
| 009 | Combat Mode Translation (TCE Contract) | Integration | Blocked | GAP — Party ADR / ADR-0009 ext |
| 010 | Scout System | Logic | Blocked | GAP — Party ADR |
| 011 | Party Formation & Marching Order | Logic | Blocked | GAP — Party ADR |
| 012 | Companion Knowledge Contribution | Integration | Blocked | GAP — Party ADR + ADR-0008 rev |
| 013 | Rest-Point Gating | Logic | Blocked | GAP — Party ADR |
| 014 | Party Management Screen | UI | Blocked | GAP — Party ADR (+ ADR-0012) |

## Next Step
Unblock the epic: run `/architecture-decision` to create the Party System ADR (covers TR-party-002..007 GAPs), then flip stories 002, 006, 008–014 to Ready. Stories 001, 003–005, 007 can be implemented now (`/story-readiness` → `/dev-story`), though 003–005 depend on Story 002's state machine.

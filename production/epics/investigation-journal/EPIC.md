# Epic: Investigation Journal

> **Layer**: Presentation | **GDD**: investigation-journal.md | **Status**: Ready | **Stories**: 5 stories
> **Governing ADRs**: ADR-0012 | **Engine Risk**: HIGH | **Tier**: MVP

## Overview
Implement the player's investigation interface: three-panel parchment overlay (chain list sidebar, detail view, entity quick-ref bar), thread display with status indicators, chain view with auto-summary, fact log organized by source NPC, contradiction visualization (thread-red connector), and revelation prompts (amber glow on chains with 3+ connected facts). Desk register — world dims behind parchment.

## GDD Requirements
All TR-ui-* via ADR-0012 (UI Framework) ✅

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Journal Shell & Three-Panel Layout | UI | Complete | ADR-0012 |
| 002 | Chain List Sidebar & Thread Detail View | UI | Complete | ADR-0012 |
| 003 | Fact Log & Entity Quick-Ref Bar | UI | Complete | ADR-0012 |
| 004 | Contradiction Display & Revelation Prompts | Integration | Complete | ADR-0012, ADR-0010 |
| 005 | Map Centering & Camera Integration | Integration | Ready | ADR-0012, ADR-0003 |

## Next Step
Run `/story-readiness production/epics/investigation-journal/story-001-journal-shell-layout.md`

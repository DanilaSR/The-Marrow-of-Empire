# Epic: Character & Inventory UI

> **Layer**: Presentation | **GDD**: character-inventory-ui.md | **Status**: Ready | **Stories**: 9 stories
> **Governing ADRs**: ADR-0012 | **Engine Risk**: HIGH | **Tier**: VS

## Overview
Implement character sheet and inventory: tabbed parchment overlay (P24) with party sidebar, attribute bars (P13) with modifiers, derived stats panel, equipment slots with drag-to-equip (P14), inventory grid (P15) sorted by category, item detail tooltip, weight bar (amber→oxblood), and gold display. C key for character, I for inventory.

## GDD Requirements
All TR-ui-* via ADR-0012 ✅

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Character Sheet Stat Display | UI | Complete | ADR-0012 |
| 002 | XP Progress Bar & Level-Up Modal | Logic | Complete | ADR-0012 |
| 003 | Equipment Slots & Drag-and-Drop | Integration | Complete | ADR-0012 |
| 004 | Equipment Stat Comparison | UI | Complete | ADR-0012 |
| 005 | Inventory Grid & Weight Bar | UI | Complete | ADR-0012 |
| 006 | Inventory Toast Notifications | UI | Complete | ADR-0012 |
| 007 | Screen State Machine & Combat Integration | Integration | Complete | ADR-0012 |
| 008 | Party Sidebar & Companion Inspect | Integration | Complete | ADR-0012 |
| 009 | Accessibility & Input | UI | Complete | ADR-0012 |

## Next Step
Run `/story-readiness production/epics/character-inventory-ui/story-001-character-sheet-display.md`

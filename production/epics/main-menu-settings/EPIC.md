# Epic: Main Menu & Settings

> **Layer**: Presentation | **GDD**: main-menu-and-settings.md | **Status**: Ready | **Stories**: 7 stories
> **Governing ADRs**: ADR-0012 | **Engine Risk**: HIGH | **Tier**: MVP

## Overview
Implement title screen (storybook dusk vista per art bible State 6), New Game → Character Creation flow, Load Game with save slot cards (22 slots), Settings (audio/video/gameplay/controls/accessibility tabs), pause menu (Esc overlay), Steam Cloud conflict dialog, and AI model status indicator. Desk register — world dims behind parchment.

## GDD Requirements
All TR-ui-* via ADR-0012 ✅

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Navigation Accessibility & Screen Transition Framework | UI | Ready | ADR-0012 |
| 002 | Title Screen & New Game Flow | UI | Ready | ADR-0012 |
| 003 | First-Launch Accessibility Prompt | UI | Ready | ADR-0012 |
| 004 | Save/Load Screens | Integration | Ready | ADR-0012 |
| 005 | Settings Screen & Persistence | Integration | Ready | ADR-0012 |
| 006 | Pause Menu | Integration | Ready | ADR-0012 |
| 007 | Keybindings & Input Remapping | Integration | Ready | ADR-0012 |

## Next Step
Run `/story-readiness production/epics/main-menu-settings/story-001-navigation-transition-framework.md`

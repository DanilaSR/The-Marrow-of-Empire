# Epic: Map System

> **Layer**: Presentation | **GDD**: map-system.md | **Status**: Ready | **Stories**: 3 stories
> **Governing ADRs**: ADR-0012 | **Engine Risk**: HIGH | **Tier**: Alpha

## Overview
Implement spatial navigation: ink-on-parchment local map with discovered locations, NPC positions, thread markers (amber dot/star, P07), and player position. World map at Alpha tier with region summaries and travel routes. M key to open.

## GDD Requirements
All TR-ui-* via ADR-0012 ✅

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Local Map Display & Navigation | UI | Ready | ADR-0012 |
| 002 | Map Markers & Pins | Integration | Ready | ADR-0012, ADR-0005, ADR-0010 |
| 003 | World Map & Region Travel | Integration | Ready | ADR-0012, ADR-0011 |

## Next Step
Run `/story-readiness production/epics/map-system/story-001-local-map-display.md` to validate readiness, then `/dev-story` to implement.

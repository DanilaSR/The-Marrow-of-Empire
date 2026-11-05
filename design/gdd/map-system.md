# Map System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 1 (Every Voice — the map shows where witnesses are)

## Summary

The Map System provides spatial navigation across regions. Local map for the
current region (discovered areas, NPC locations, points of interest), and a
world map for travel between regions (Alpha tier). Displays world state —
cleared ruins marked, discovered locations noted, active threads pinned.
Parchment aesthetic with ink-drawn geography.

> **Quick reference** — Layer: `Presentation` · Priority: `Alpha` · Key deps: `Living Environment`, `Persistent World State`

## Overview

The Map System helps the player navigate the borderland. At MVP (basic map
within one region), the local map shows the current region's layout, discovered
locations, known NPC positions, and active investigation thread markers. At
Alpha, the world map enables travel between regions.

**Local map (MVP foundation, Alpha polish):**
- Ink-on-parchment linework of the current region
- Discovered locations marked with named pins
- Known NPC positions (last seen location)
- Active thread markers (amber dot) and prioritized threads (amber star)
- Cleared/destroyed locations marked (ink X)
- Player position and facing

**World map (Alpha):**
- Overview of all discovered regions
- Region state summaries (cleared, visited, unexplored)
- Travel routes between regions
- Faction territory overlays

## Visual/Audio Requirements

Map uses the desk register — parchment overlay with ink-drawn geography.
Thread markers per Section 6.3 iconography. Map opens with M key.

## Acceptance Criteria

- **GIVEN** a discovered NPC location, **WHEN** the map is opened, **THEN** the NPC is shown at their last known position with their name.
- **GIVEN** an active thread referencing a location, **WHEN** the map is opened, **THEN** an amber dot marks that location.

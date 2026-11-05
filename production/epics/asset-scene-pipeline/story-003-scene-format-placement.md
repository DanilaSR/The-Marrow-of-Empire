# Story 003: Scene Format & Entity Placement

> **Epic**: Asset & Scene Pipeline
> **Status**: Ready
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/asset-scene-pipeline.md`
**Requirement**: `TR-asset-005`

**ADR**: ADR-0004 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN a scene file with 50 entity placements, WHEN LoadScene runs, THEN 50 entities are created with correct positions, rotations, and initial components
- [ ] GIVEN a scene file, WHEN loaded, THEN camera bounds from the scene are applied to the Camera System
- [ ] GIVEN trigger volumes in the scene, WHEN an entity enters/exits, THEN the trigger event fires
- [ ] GIVEN a nav mesh baked into the scene, WHEN AI pathfinding queries it, THEN navigable areas are correctly identified

## Implementation Notes

From ADR-0004: Scene file format: metadata (name, region ID, ambient lighting preset, audio ref) + entity placements (asset_ref, position, rotation, scale, initial_components) + lighting entities + trigger volumes + nav mesh + camera bounds. Scenes are hand-authored in the Kaiju editor.

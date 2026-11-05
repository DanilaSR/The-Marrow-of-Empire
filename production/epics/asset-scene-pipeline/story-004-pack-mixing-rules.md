# Story 004: Two-Pack Integration Rules

> **Epic**: Asset & Scene Pipeline
> **Status**: Ready
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001, Story 003

## Context

**GDD**: `design/gdd/asset-scene-pipeline.md`
**Requirement**: TR-asset-005 (pack mixing rules per CR-6)

**ADR**: ADR-0004 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN a scene using only Dark Fantasy assets (>80% surface area), WHEN the validator runs, THEN a dominance warning is flagged
- [ ] GIVEN a scene mixing both packs, WHEN validated, THEN no warning if each pack is between 20-80% of visible surface area
- [ ] GIVEN assets from both packs at the same nominal size (e.g., "door"), WHEN imported, THEN both are within 10% scale of each other after normalization
- [ ] GIVEN a scene's area archetype (e.g., Warm Borderland), WHEN the dominant material palette is >80% Dark Fantasy grey stone, THEN a color temperature warning is flagged

## Implementation Notes

From ADR-0004: The "no 50/50 frames" rule has a mechanical ally — the validator's single-pack-dominance warning. Art bible Section 5.5: balance reads as collision, reserved for State 4 revelation. Pack-to-stratum mapping: Dark Fantasy = bones, Enchanted = skin. Both pass through same painterly grade.

# Story 002: Material Mapping & Validation

> **Epic**: Asset & Scene Pipeline
> **Status**: Ready
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/asset-scene-pipeline.md`
**Requirement**: `TR-asset-003`, `TR-asset-004`

**ADR**: ADR-0004 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN a YAML material mapping table, WHEN an asset is imported, THEN source material names map to project material definitions
- [ ] GIVEN a material not in the mapping table, WHEN validation runs, THEN the material is flagged for human review and loads as default grey
- [ ] GIVEN an asset with >5000 tris (static prop), WHEN validated, THEN it is rejected with decimation flag
- [ ] GIVEN an asset with >8 material slots, WHEN validated, THEN it is rejected with consolidation flag
- [ ] GIVEN a texture at 3000×3000, WHEN validated, THEN it is auto-downscaled to 2048×2048 with warning logged

## Implementation Notes

From ADR-0004: Validation is blocking — rejected assets cannot enter scenes. Hero assets have higher thresholds (≤20k tris, warn not reject). YAML mapping tables are hand-authored per pack. Color palette compliance checked against art bible families — warns, doesn't reject.

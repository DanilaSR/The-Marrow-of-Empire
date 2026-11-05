# Epic: Asset & Scene Pipeline

> **Layer**: Foundation
> **GDD**: design/gdd/asset-scene-pipeline.md
> **Architecture Module**: Asset & Scene Pipeline
> **Status**: Ready
> **Stories**: 4 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | FBX Import & Normalization | Integration | Ready | ADR-0004 |
| 002 | Material Mapping & Validation | Logic | Ready | ADR-0004 |
| 003 | Scene Format & Entity Placement | Integration | Ready | ADR-0004 |
| 004 | Two-Pack Integration Rules | Integration | Ready | ADR-0004 |

## Overview

Implement the asset import pipeline for KitBash3D Dark Fantasy and Enchanted
packs: FBX/OBJ → Kaiju native conversion, scale/rotation normalization
(1 unit = 1 meter, Z-up → Y-up), YAML material mapping tables, validation
(poly count, texture resolution, naming, palette), scene file format with
entity placements, and the two-pack mixing rules (no 50/50 frames).

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0004: Asset Pipeline | FBX interchange, YAML mappings, validation rules, two-pack integration | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-asset-001 | FBX → Kaiju native conversion | ADR-0004 ✅ |
| TR-asset-002 | KitBash3D scale/rotation normalization | ADR-0004 ✅ |
| TR-asset-003 | YAML material mapping tables | ADR-0004 ✅ |
| TR-asset-004 | Asset validation rules | ADR-0004 ✅ |
| TR-asset-005 | Scene format and entity placement | ADR-0004 ✅ |

## Definition of Done

- All stories implemented, reviewed, and closed
- KitBash3D assets import without error
- Validation catches and reports poly/texture/naming violations
- Scene file loads with correct entity placements
- Material mapping correctly routes source materials to project palette

## Next Step

Run `/create-stories asset-scene-pipeline`

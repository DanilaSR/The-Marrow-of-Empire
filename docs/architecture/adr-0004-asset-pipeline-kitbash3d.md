# ADR-0004: Asset Pipeline & KitBash3D Integration

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Rendering / Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Asset & Scene Pipeline GDD, Art Bible Sections 5, 8 |
| **Post-Cutoff APIs Used** | Kaiju native mesh/texture formats |
| **Verification Required** | Kaiju mesh import API, texture format requirements, material system integration |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — scene entities) |
| **Enables** | All visual content — without asset import, nothing renders |
| **Blocks** | Scene authoring, environment art, character placement |
| **Ordering Note** | Must be Accepted before KitBash3D assets can be imported |

## Context

The Marrow of Empire uses two KitBash3D asset packs: Dark Fantasy (Gothic ruins) and Enchanted (peaceful nature). Assets arrive as FBX/OBJ with PNG/TGA textures. They must be converted to Kaiju native formats, normalized to project scale (1 unit = 1 meter), material-remapped to the art bible's palette, and validated against project standards.

## Decision

**FBX as interchange format, PNG textures, YAML material mapping tables.** Assets are normalized in a staging directory before entering the engine. Validation enforces poly count, texture resolution, material slot count, naming conventions, and color palette compliance per the Asset & Scene Pipeline GDD's validation table.

### Pipeline Stages
Source (KitBash3D originals, read-only) → Staging (normalization, conversion) → Imported (Kaiju native, validated) → Placed (in scenes)

### Validation Thresholds
Static props ≤5000 tris. Hero assets ≤20000 tris. Textures ≤2048×2048 (hero), ≤1024×1024 (standard). Material slots ≤8 per mesh. Naming: kebab-case. Color palette: within art bible families.

### Material Mapping
Hand-authored YAML mapping tables per pack. Source material names → project material definitions. Unmapped materials flagged for human review; load as default grey.

### Two-Pack Integration
Dark Fantasy = bones (Gothic construction, empire remnants). Enchanted = skin (green reclamation, settled warmth). Both pass through the same painterly grade — no asset remembers which pack it came from. Validator enforces no single-pack dominance (>80% of visible surface area = warning).

## GDD Requirements Addressed
Asset & Scene Pipeline GDD: CR-1 through CR-6 (all core rules including validation table, scene format, asset lifecycle, pack mixing).

## Performance Implications
Asset import: ~30s for 50MB FBX. Validation: <100ms per asset. Scene load: ~2s for 200 entity placements. Material mapping: <1ms per material lookup.

## Related Decisions
ADR-0001: Entity Architecture — scene placements create entities.

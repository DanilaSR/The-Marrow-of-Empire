# Asset & Scene Pipeline

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 1 (Every Voice — environmental objects are witnesses), Pillar 2 (The World Remembers — scene state persists)

## Summary

The Asset & Scene Pipeline imports KitBash3D Dark Fantasy and Enchanted packs
into Kaiju Engine, normalizes scale and materials between them, validates every
asset against project standards, and composes them into hand-authored scenes.
It is the bridge from source art to playable world — the reason the player sees
Gothic ruins swallowed by enchanted forests at golden hour instead of grey
placeholder geometry.

> **Quick reference** — Layer: `Foundation` · Priority: `MVP` · Key deps: `None`

## Overview

The Asset & Scene Pipeline is the bridge between source art and the playable
world. It imports external assets into Kaiju Engine's native formats, validates
them against project standards, and composes them into hand-authored scenes —
the regions, interiors, and environmental setpieces the player explores.

The pipeline draws from two complementary asset foundations:

- **KitBash3D Dark Fantasy**: Gothic architecture, ruined structures, dark stone,
  crumbling empire remnants. The "dangerous" half of the fairy-tale balance.
  Provides the empire stratum — bone ivory ruins, machine-halls, corrupted ground.
- **KitBash3D Enchanted**: Peaceful natural fantasy — living forests, gentle
  architecture, verdant landscapes. The "beautiful" half of the fairy-tale
  balance. Provides the present stratum — the warm-gold borderland, settled
  hearths, the world the player is fighting to protect.

The pipeline provides:

- **Asset import**: Converts source assets (FBX/OBJ models, texture sets,
  material definitions) into Kaiju Engine's native format. Handles format
  conversion, scale normalization between the two packs, material remapping to
  the art bible's color palette, and collision mesh generation.
- **Asset validation**: Every imported asset is checked against project standards:
  poly count within budget, texture resolution within limits, material palette
  within the art bible's color families, naming conventions followed. Validation
  failures block the asset from entering a scene.
- **Scene composition**: A scene is a hand-authored arrangement of entities,
  props, lighting, terrain, and triggers placed in world space. Scenes mix assets
  from both packs — a ruined archway (Dark Fantasy) swallowed by an enchanted
  forest (Enchanted) is the visual signature of the borderland. The pipeline
  provides the placement tools and the scene file format.
- **Asset lifecycle**: Assets have states: Source (original KitBash3D file),
  Imported (converted, validated, in the asset library), Placed (instantiated
  in a scene), and Retired (removed from active use, kept for reference).
- **Scene serialization**: Scenes save as data files listing entity placements
  and asset references. The Save/Load system serializes runtime scene state
  (dynamic changes); the pipeline serializes the authored scene (the starting state).

Without the pipeline: KitBash3D assets are files on disk with no path into the
engine. There are no forests, no ruins, no borderland to explore.

## Player Fantasy

The pipeline has no direct player fantasy, but its output IS the player's entire
visual and spatial experience:

- **The world is hand-crafted.** Every ruin, forest clearing, and border-town
  street is composed with intention. An artist placed that lantern on that
  windowsill, chose that Dark Fantasy archway to frame that Enchanted oak. The
  pipeline makes those placements possible and preserves them.
- **The fairy-tale balance is delivered.** Dark Fantasy and Enchanted assets
  woven together create the art bible's central tension: beautiful enough to
  mourn, dangerous enough to fear. A sunlit Enchanted grove with a Dark Fantasy
  spire on the horizon — that's the borderland in one frame.
- **The environment is a witness.** Per Pillar 1, every placed object knows
  something. The pipeline ensures every prop, mural, and relic is a distinct
  entity with its own identity — not a cloned decoration with no memory.

The pipeline's success: a player stops at a ruined archway at dusk, watches
amber light filter through enchanted leaves, and thinks "this world is real."

## Detailed Design

### Core Rules

**CR-1: Asset Source Management**

The pipeline tracks assets from their original source files through to engine-native format:

| Stage | Location | Format | Purpose |
|-------|----------|--------|---------|
| Source | `KitBash3D - Dark Fantasy/`, `KitBash3D - Enchanted/` | FBX, OBJ, BLEND, PNG/TGA textures | Original purchased assets — never modified |
| Staging | `assets/staging/` | FBX/OBJ (normalized) + PNG (converted textures) | Pre-import normalization: scale, rotation, texture format conversion |
| Imported | `assets/library/` | Kaiju Engine native format | Validated, engine-ready assets referenced by scenes |
| Placed | Referenced in scene files | Scene data (entity placements) | Assets instantiated in the world |

The source directory is read-only to the pipeline. All modifications happen in staging.

**CR-2: Format Conversion and Normalization**

Assets undergo conversion before entering the engine:

- **Model format**: FBX and OBJ → Kaiju Engine native mesh format. BLEND files
  are exported to FBX first (Blender headless or manual export).
- **Scale normalization**: Dark Fantasy and Enchanted packs may use different
  base units. All assets are normalized to the project unit scale (1 unit = 1
  meter in world space) during staging. A reference object (human-height door
  at 2 units tall) validates scale consistency between packs.
- **Rotation normalization**: Z-up to Y-up conversion (KitBash3D uses Z-up;
  Kaiju uses Y-up). Applied during staging.
- **Texture format**: PNG/TGA → Kaiju Engine native texture format. Power-of-two
  dimensions enforced (non-PoT textures are padded or downscaled with a warning).
- **Material remapping**: Source material names are mapped to project material
  definitions. A material mapping table (hand-authored per asset pack) specifies
  which source material maps to which engine material. Materials that don't match
  the art bible's color palette are flagged for manual review.

**CR-3: Asset Validation**

Every imported asset must pass validation before it can be placed in a scene.
Validation failures block the asset — no broken content in the game world.

| Check | Threshold | Action on Failure |
|-------|-----------|-------------------|
| Polygon count (static prop) | ≤ 5000 tris | Reject — flag for decimation or LOD generation |
| Polygon count (hero asset) | ≤ 20000 tris | Warn — flag for manual review |
| Texture resolution (single) | ≤ 2048×2048 | Downscale automatically; log warning |
| Texture resolution (tileset) | ≤ 4096×4096 | Warn — flag for manual review |
| Material slot count | ≤ 8 per mesh | Reject — flag for material consolidation |
| Naming convention | kebab-case, no spaces, no special chars | Reject — rename suggestion provided |
| Missing texture reference | None | Reject — missing texture flagged |
| Non-power-of-two texture | — | Auto-pad to next PoT; log warning |
| Color palette compliance | Within art bible families (amber, moss, stone, indigo, ivory, oxblood) | Warn — flag for manual review |
| Scale within range | 0.5× to 2× of reference object | Warn — flag for manual review |

**CR-4: Scene Format**

A scene file defines an authored region or interior. It is a data file (not code)
containing:

- **Metadata**: Scene name, region ID, ambient lighting preset (from art bible
  Section 2.3 — the area archetype), background audio reference
- **Entity placements**: List of (asset_ref, position, rotation, scale,
  initial_components) tuples. Each placement creates an entity at scene load.
- **Lighting entities**: Point lights, directional lights, ambient probes placed
  as entities with Light components
- **Trigger volumes**: Named regions that fire events on entry/exit (dialogue
  triggers, combat boundaries, region transitions)
- **Nav mesh data**: Pre-computed navigation mesh for AI pathfinding (generated
  from placed geometry at scene bake time)
- **Camera bounds**: The playable area rectangle — camera boundary enforcement
  reads this

Scenes are hand-authored in the Kaiju editor, saved as scene files, and loaded
by the engine at runtime.

**CR-5: Asset Lifecycle**

| State | Meaning | Allowed Actions |
|-------|---------|-----------------|
| `Source` | Original KitBash3D file, unmodified | Read metadata only |
| `Staging` | Being processed for import | Normalize, convert, validate |
| `Imported` | In the asset library, validated | Reference in scenes, query metadata |
| `Placed` | Instantiated in at least one scene | Modify placement, runtime state changes |
| `Retired` | Removed from all active scenes, kept for reference | No new placements, retained for existing saves |
| `Deprecated` | Known-broken or superseded | No new placements, log warning if referenced by any scene |

Retired assets are not deleted — existing save files may reference them (a save
from before a region was redesigned should still load). Deprecated assets are
flagged for cleanup in the next major version.

**CR-6: Mixing Asset Packs in Scenes**

The two KitBash3D packs are designed to be interleaved within scenes to create
the fairy-tale balance. The pipeline enforces:

- **No pack-exclusive scenes**: Every outdoor scene should contain at least one
  asset from each pack. (Interior scenes may be single-pack — a deep ruin
  interior may be entirely Dark Fantasy.)
- **Color temperature consistency**: Placed assets' dominant material colors
  should align with the scene's area archetype (art bible Section 2.3). A Warm
  Borderland scene that is 80% Dark Fantasy grey stone fails the archetype —
  the validator warns.
- **Scale coherence**: Assets from both packs at the same nominal size (e.g.,
  "door") must be within 10% of each other in world units after normalization.

### States and Transitions

The pipeline has no runtime states — it operates at authoring time and scene load time.

**Authoring-time states** (asset processing):
`Source` → (export/conversion) → `Staging` → (validation pass) → `Imported` → (placement in scene) → `Placed` → (removal from all scenes) → `Retired` → (marked broken) → `Deprecated`

**Runtime** (scene load):
Scene file read → Entities instantiated → Components attached → Scene active

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Pipeline creates entities at scene load | Scene placements become entities with position + components. Pipeline provides `load_scene(scene_id) → entity_ids[]` |
| Living Environment | Pipeline provides terrain and foliage assets | Weather effects interact with placed geometry; wildlife spawns on nav mesh |
| Camera System | Pipeline provides camera bounds per scene | Scene file defines the playable area rectangle that Camera uses for boundary enforcement |
| Save/Load System | Pipeline provides the initial scene state | Save/Load serializes the delta between authored scene and runtime state |
| Rendering System | Pipeline provides all visible assets | Renderer reads imported meshes, textures, and materials from the asset library |
| Audio System | Pipeline places audio emitters | Scenes contain ambient audio entities referencing sound assets |

## Formulas

No gameplay formulas. The pipeline's calculations are scale normalization
(source unit to project unit ratio) and texture PoT padding — both standard
asset pipeline math, not game design formulas.

## Edge Cases

- **If an asset from one pack references a material that doesn't exist in the mapping table**: The asset is flagged with a warning and assigned a default grey material. A human must author the mapping — the pipeline never guesses material intent.
- **If a scene references a Retired asset**: The scene still loads — the asset is still in the library. The editor shows a warning on that placement. The asset cannot be Deprecated until all scene references are updated.
- **If two assets from different packs have conflicting scale after normalization**: Both are flagged. The human author decides which is the reference scale. The other is re-normalized.
- **If a texture exceeds the resolution limit**: Auto-downscale to the limit. The original is preserved in staging. If downscaling causes visible quality loss (checked manually), the asset gets a "hero asset" exemption from the resolution cap.
- **If the nav mesh fails to generate for a section of the scene**: The affected area is marked as non-navigable. AI pathfinding routes around it. The scene still loads — a gap in the nav mesh is better than a scene that won't load.
- **If a scene file references an asset ID that doesn't exist in the library**: The scene load fails for that placement with an error logged. Other placements load normally. The missing asset is replaced with a visible error marker (red box) in the editor.

## Dependencies

The Asset & Scene Pipeline depends on nothing — it is a Foundation-layer system.

| System | Direction | Nature of Dependency |
|--------|-----------|---------------------|
| Entity Framework | Entity Framework receives entities from pipeline | Pipeline calls `create_entity()` for each scene placement at scene load. Hard dependency — no pipeline, no entities in the world. |
| Living Environment | Living Environment depends on pipeline | Receives terrain meshes, foliage assets, and weather-interactive geometry. Soft — environment can function with placeholder terrain. |
| Camera System | Camera depends on pipeline | Scene file provides camera bounds (playable area rectangle). Hard — camera needs bounds to enforce. |
| Save/Load System | Save/Load depends on pipeline | Pipeline provides the authored scene state; Save/Load serializes the runtime delta. Hard — save/load needs an initial state to diff against. |
| Rendering System | Renderer depends on pipeline | All visible meshes, textures, and materials come through the pipeline. Hard — nothing to render without imported assets. |
| Audio System | Audio depends on pipeline | Scene-defined ambient audio entities. Soft — audio can function with default ambience. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `max_tris_static_prop` | 5000 | 1000–15000 | More detailed props, higher GPU load | Lower detail, better performance |
| `max_tris_hero_asset` | 20000 | 5000–50000 | More detailed hero assets | Simpler hero assets, better performance |
| `max_texture_resolution` | 2048 | 512–4096 | Sharper textures, more VRAM usage | Blurrier textures, less VRAM |
| `scale_reference_height` | 2.0 units | 1.5–3.0 | Larger world relative to assets | Smaller world relative to assets |
| `nav_mesh_cell_size` | 0.5 units | 0.25–1.0 | Finer pathfinding, larger nav mesh bake time | Coarser pathfinding, faster bake |
| `color_temp_warn_threshold` | 80% | 50–100 | Less strict about single-pack dominance | More strict — fewer single-pack scenes allowed |

## Visual/Audio Requirements

The pipeline produces no visual or audio output itself. Its requirements:

- The asset library must store visual assets at the quality defined by the
  tuning knobs — imported meshes and textures should be visually identical to
  their source files at Default zoom level.
- Scene loading must produce a visually complete frame within the engine's
  scene load budget (target: 2 seconds for an average-sized scene of 200
  entity placements). A loading screen or fade transition masks the load.
- Audio emitters placed in scenes must reference valid audio assets. Missing
  audio references log a warning and produce silence — no crash.

## UI Requirements

The pipeline has no player-facing UI. Editor UI requirements (out of scope
for this GDD — tooling backlog):

- Asset browser: browse the asset library, preview meshes/textures, inspect
  validation status
- Scene hierarchy: view and select entity placements in a scene
- Material mapping editor: author and edit material mapping tables
- Validation report: view validation failures and warnings per asset

## Acceptance Criteria

- **GIVEN** a KitBash3D FBX asset in the source directory, **WHEN** the import pipeline runs, **THEN** the asset is converted to Kaiju native format, normalized to project scale (1 unit = 1 meter), and placed in the asset library with an auto-generated asset ID.

- **GIVEN** an asset with 8000 tris, **WHEN** validated as a static prop (max 5000 tris), **THEN** validation rejects it with a decimation flag and the asset is not placeable in scenes until the flag is resolved.

- **GIVEN** a non-power-of-two texture at 1500×900, **WHEN** imported, **THEN** it is padded to 2048×1024 and a warning is logged.

- **GIVEN** a scene file with 50 entity placements, **WHEN** the scene loads, **THEN** 50 entities are created by the Entity Framework with correct positions, rotations, and initial components matching the scene data.

- **GIVEN** a scene mixing both KitBash3D packs, **WHEN** the color temperature validator runs, **THEN** scenes where >80% of visible surface area comes from one pack are flagged with a warning.

- **GIVEN** a Retired asset referenced by an existing scene, **WHEN** the scene loads, **THEN** the asset loads normally and the editor shows a warning on that placement.

- **Performance**: Asset import processes a 50 MB FBX file within 30 seconds. Scene load creates 200 entities within 2 seconds at 60fps.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| What is Kaiju Engine's native mesh and texture format? The import pipeline must target this — need engine specifics. | Engine Programmer | Before implementation (ADR) | — |
| Does Kaiju Editor support custom scene file formats, or does the pipeline need to output Kaiju's native scene format? | Engine Programmer | Before implementation (ADR) | — |
| Are there existing Blender export scripts for Kaiju, or does the pipeline need to build FBX→Kaiju conversion from scratch? | Tools Programmer | Before first asset import | — |
| Should the pipeline support hot-reloading (artist saves asset → scene updates in running editor)? | Technical Director | Before vertical slice | — |

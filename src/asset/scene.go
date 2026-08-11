// Scene file format for the Asset & Scene Pipeline (ASSET-003, TR-asset-005,
// ADR-0004, GDD CR-4): hand-authored YAML defining an authored region or
// interior — metadata, entity placements, lighting entities, trigger
// volumes, a baked nav mesh, and camera bounds. Parsing is a pure function
// of the input bytes (no file I/O, no engine state), mirroring
// material.go's ParseMaterialMap.
//
// Format contract (GDD CR-4 / ADR-0004):
//   - Metadata: scene name, region ID, ambient lighting preset (art bible
//     Section 2.3 area archetype), background audio reference.
//   - Placements: (id, asset_ref, position, rotation, scale,
//     initial_components) tuples. Each placement creates an entity at scene
//     load. Asset refs must exist in the asset library — existence is
//     checked by ValidatePrefabRefs, since the parser cannot know the
//     library.
//   - Lighting entities: point, directional, and ambient-probe lights with
//     Light components. Lights share the scene entity ID namespace with
//     placements.
//   - Trigger volumes: named AABB regions that fire events on entity
//     enter/exit (dialogue triggers, combat boundaries, region transitions).
//   - Nav mesh: pre-baked navigation polygons for AI pathfinding (generated
//     from placed geometry at scene bake time), with the GDD
//     nav_mesh_cell_size knob.
//   - Camera bounds: the playable area rectangle (AABB) the Camera System
//     enforces. Required — the camera hard-depends on it (GDD).
//
// Rotation is Euler degrees on XYZ, matching the matrix.Transform rotation
// convention (QuaternionFromEuler converts degrees). Positions and scales
// are world units (1 unit = 1 meter, ADR-0004).
//
// Performance: ADR-0004 budgets scene load at ~2s for 200 entity
// placements. This layer covers the parse step only; entity instantiation
// is the runtime story that consumes Scene.
package asset

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"kaijuengine.com/matrix"
)

// Scene file errors. These are sentinel errors, use errors.Is to match them.
var (
	// ErrMissingSceneField is returned when a scene is missing a required
	// field (scene name, region ID, placement id or asset_ref, camera
	// bounds or its corners). Hand-authored scenes must be complete.
	ErrMissingSceneField = errors.New("asset: scene missing required field")
	// ErrDuplicateEntityID is returned when two placements, two lights, or
	// a placement and a light share an entity ID. Placements and lights
	// both create entities, so they share the scene entity ID namespace.
	ErrDuplicateEntityID = errors.New("asset: duplicate entity ID in scene")
	// ErrUnknownPrefabRef is returned by ValidatePrefabRefs when a
	// placement references an asset that is not in the library. Every
	// unknown reference is named in the error, in placement order.
	ErrUnknownPrefabRef = errors.New("asset: placement references unknown prefab")
	// ErrInvalidBounds is returned when an axis-aligned volume's min corner
	// exceeds its max corner on any axis (camera bounds, trigger volumes).
	ErrInvalidBounds = errors.New("asset: invalid scene bounds")
	// ErrInvalidLightType is returned when a light's type is not one of
	// LightTypePoint, LightTypeDirectional, or LightTypeAmbientProbe.
	ErrInvalidLightType = errors.New("asset: invalid light type")
	// ErrInvalidNavMesh is returned when a nav mesh polygon has fewer than
	// three vertices — a degenerate polygon cannot be navigable.
	ErrInvalidNavMesh = errors.New("asset: invalid nav mesh")
)

// Light types for Light.Type (GDD CR-4 lighting entities).
const (
	// LightTypePoint is a point light: positioned, radial falloff over
	// Range (lanterns, braziers).
	LightTypePoint = "point"
	// LightTypeDirectional is a directional light: parallel rays along
	// Rotation, no positional falloff (sun, moon).
	LightTypeDirectional = "directional"
	// LightTypeAmbientProbe is an ambient probe: indirect fill light for an
	// area, no position or direction.
	LightTypeAmbientProbe = "ambient-probe"
)

// DefaultNavMeshCellSize is the GDD nav_mesh_cell_size tuning knob default
// (meters). Scenes that omit cell_size fall back to it.
const DefaultNavMeshCellSize = matrix.Float(0.5)

// Scene is one parsed scene file: everything a scene loader needs to
// instantiate the authored region (GDD CR-4).
type Scene struct {
	// Name is the scene name (required, kebab-case).
	Name string
	// RegionID identifies the region the scene belongs to (required,
	// kebab-case).
	RegionID string
	// AmbientPreset is the ambient lighting preset from the art bible area
	// archetype (Section 2.3), e.g. "golden-hour". Optional.
	AmbientPreset string
	// AudioRef references the background audio asset. Optional.
	AudioRef string
	// Placements list the entity placements in authored order.
	Placements []Placement
	// Lights list the lighting entities (they create entities with Light
	// components at scene load).
	Lights []Light
	// Triggers list the named trigger volumes.
	Triggers []TriggerVolume
	// NavMesh carries the pre-baked navigation mesh. Empty when the scene
	// has none baked yet.
	NavMesh NavMesh
	// CameraBounds is the playable area rectangle the Camera System
	// enforces (required).
	CameraBounds CameraBounds
}

// Placement is one entity placement tuple (asset_ref, position, rotation,
// scale, initial_components) — each placement creates an entity at scene
// load (GDD CR-4).
type Placement struct {
	// ID is the scene-unique entity ID (required, kebab-case). Must not
	// collide with any other placement or light ID.
	ID string
	// AssetRef references an imported, validated asset in the library
	// (required, kebab-case). Existence is checked by ValidatePrefabRefs.
	AssetRef string
	// Position is the world position in meters (1 unit = 1 meter).
	Position matrix.Vec3
	// Rotation is the Euler rotation in degrees (XYZ), matching the
	// matrix.Transform rotation convention.
	Rotation matrix.Vec3
	// Scale is the per-axis scale. It defaults to zero when omitted —
	// always write it explicitly.
	Scale matrix.Vec3
	// InitialComponents lists the component types attached to the entity
	// at load (e.g. "visual", "audio", "interactable"). Resolved against
	// the component registry by the runtime scene loader — this layer
	// treats them as opaque kebab-case identifiers.
	InitialComponents []string
}

// Light is one lighting entity (GDD CR-4): point, directional, or
// ambient-probe. Lights create entities with Light components at scene load
// and share the entity ID namespace with placements.
type Light struct {
	// ID is the scene-unique entity ID (required, kebab-case).
	ID string
	// Type is one of LightTypePoint, LightTypeDirectional,
	// LightTypeAmbientProbe (required).
	Type string
	// Position is the world position in meters; meaningful for point
	// lights, ignored for directional lights.
	Position matrix.Vec3
	// Rotation is the Euler rotation in degrees (XYZ); points directional
	// lights, ignored for point and ambient-probe lights.
	Rotation matrix.Vec3
	// Color is the RGBA light color. It defaults to black when omitted —
	// always write it explicitly.
	Color matrix.Color
	// Range bounds point light falloff in meters; zero for directional
	// lights.
	Range matrix.Float
	// Intensity scales the light output.
	Intensity matrix.Float
}

// TriggerVolume is a named axis-aligned region that fires events when
// entities enter or exit (GDD CR-4: dialogue triggers, combat boundaries,
// region transitions).
type TriggerVolume struct {
	// Name is the scene-unique trigger name (required, kebab-case).
	Name string
	// Min and Max are the axis-aligned bounds in world meters; Min <= Max
	// on every axis (required).
	Min matrix.Vec3
	Max matrix.Vec3
	// EnterEvent fires when an entity enters the volume.
	EnterEvent string
	// ExitEvent fires when an entity exits the volume.
	ExitEvent string
	// Enabled defaults to true; set false to keep the volume inert without
	// removing it from the file.
	Enabled bool
}

// NavMesh is the pre-baked navigation mesh for AI pathfinding, generated
// from placed geometry at scene bake time (GDD CR-4).
type NavMesh struct {
	// CellSize is the nav mesh cell size in meters (GDD tuning knob
	// nav_mesh_cell_size). Zero falls back to DefaultNavMeshCellSize.
	CellSize matrix.Float
	// Polygons are the navigable convex polygons in world space; each has
	// at least 3 vertices. Empty when no mesh is baked.
	Polygons [][]matrix.Vec3
}

// CameraBounds is the playable area rectangle (world meters, axis-aligned)
// that Camera System boundary enforcement reads (GDD CR-4). Required in
// every scene file.
type CameraBounds struct {
	// Min is the minimum corner; Min <= Max on every axis.
	Min matrix.Vec3
	// Max is the maximum corner.
	Max matrix.Vec3
}

// sceneYAML mirrors the hand-authored scene schema (GDD CR-4 / ADR-0004):
//
//	name: borderland-east-grove
//	region_id: borderland-east
//	ambient_preset: golden-hour
//	audio_ref: ambience-forest-birds
//	placements:
//	  - id: grove-oak-01
//	    asset_ref: enchanted-oak
//	    position: [10, 0, -20]
//	    rotation: [0, 45, 0]       # Euler degrees (matrix.Transform convention)
//	    scale: [1, 1, 1]
//	    initial_components: [visual, audio]
//	lights:
//	  - id: lantern-01
//	    type: point
//	    position: [12, 2, -18]
//	    color: [1.0, 0.7, 0.4, 1.0]
//	    range: 8.0
//	    intensity: 1.0
//	triggers:
//	  - name: grove-dialogue
//	    min: [5, 0, -30]
//	    max: [15, 4, -10]
//	    enter_event: dialogue-grove
//	    exit_event: dialogue-grove-end
//	navmesh:
//	  cell_size: 0.5
//	  polygons:
//	    - [[0, 0, 0], [20, 0, 0], [20, 0, 20]]
//	camera_bounds:
//	  min: [-50, 0, -50]
//	  max: [50, 20, 50]
//
// Pointer fields distinguish "absent" from "zero" so required sections and
// corners can be enforced, and the trigger enabled flag can default to true
// (a silently disabled trigger would be a footgun).
type sceneYAML struct {
	Name          string            `yaml:"name"`
	RegionID      string            `yaml:"region_id"`
	AmbientPreset string            `yaml:"ambient_preset"`
	AudioRef      string            `yaml:"audio_ref"`
	Placements    []placementYAML   `yaml:"placements"`
	Lights        []lightYAML       `yaml:"lights"`
	Triggers      []triggerYAML     `yaml:"triggers"`
	NavMesh       navMeshYAML       `yaml:"navmesh"`
	CameraBounds  *cameraBoundsYAML `yaml:"camera_bounds"`
}

type placementYAML struct {
	ID                string      `yaml:"id"`
	AssetRef          string      `yaml:"asset_ref"`
	Position          matrix.Vec3 `yaml:"position"`
	Rotation          matrix.Vec3 `yaml:"rotation"`
	Scale             matrix.Vec3 `yaml:"scale"`
	InitialComponents []string    `yaml:"initial_components"`
}

type lightYAML struct {
	ID        string       `yaml:"id"`
	Type      string       `yaml:"type"`
	Position  matrix.Vec3  `yaml:"position"`
	Rotation  matrix.Vec3  `yaml:"rotation"`
	Color     matrix.Color `yaml:"color"`
	Range     matrix.Float `yaml:"range"`
	Intensity matrix.Float `yaml:"intensity"`
}

type triggerYAML struct {
	Name       string       `yaml:"name"`
	Min        *matrix.Vec3 `yaml:"min"`
	Max        *matrix.Vec3 `yaml:"max"`
	EnterEvent string       `yaml:"enter_event"`
	ExitEvent  string       `yaml:"exit_event"`
	Enabled    *bool        `yaml:"enabled"`
}

type navMeshYAML struct {
	CellSize matrix.Float    `yaml:"cell_size"`
	Polygons [][]matrix.Vec3 `yaml:"polygons"`
}

type cameraBoundsYAML struct {
	Min *matrix.Vec3 `yaml:"min"`
	Max *matrix.Vec3 `yaml:"max"`
}

// ParseScene parses and validates one scene file (GDD CR-4, ADR-0004).
// Required fields must be present (scene name, region ID, per-placement id
// and asset_ref, camera bounds with both corners), identifiers must be
// kebab-case, entity IDs must be unique across placements and lights,
// trigger names unique, light types known, volume corners ordered
// Min <= Max per axis, and nav mesh polygons closed with at least 3
// vertices. Prefab existence is NOT checked here — the parser has no
// library access; call ValidatePrefabRefs with the library's prefab set.
//
//	scene, err := asset.ParseScene(data)
//	if err != nil {
//		return err // malformed YAML, missing fields, duplicates, bad bounds
//	}
func ParseScene(data []byte) (Scene, error) {
	var raw sceneYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Scene{}, fmt.Errorf("parse scene: %w", err)
	}
	if strings.TrimSpace(raw.Name) == "" {
		return Scene{}, fmt.Errorf("%w: name", ErrMissingSceneField)
	}
	if err := ValidateKebabCase(raw.Name); err != nil {
		return Scene{}, fmt.Errorf("scene name: %w", err)
	}
	if strings.TrimSpace(raw.RegionID) == "" {
		return Scene{}, fmt.Errorf("%w: region_id", ErrMissingSceneField)
	}
	if err := ValidateKebabCase(raw.RegionID); err != nil {
		return Scene{}, fmt.Errorf("region id: %w", err)
	}
	if raw.CameraBounds == nil {
		return Scene{}, fmt.Errorf("%w: camera_bounds (the Camera System hard-depends on it)", ErrMissingSceneField)
	}
	if raw.CameraBounds.Min == nil || raw.CameraBounds.Max == nil {
		return Scene{}, fmt.Errorf("%w: camera_bounds.min/max", ErrMissingSceneField)
	}
	if err := validateBounds(*raw.CameraBounds.Min, *raw.CameraBounds.Max, "camera_bounds"); err != nil {
		return Scene{}, err
	}

	scene := Scene{
		Name:          raw.Name,
		RegionID:      raw.RegionID,
		AmbientPreset: raw.AmbientPreset,
		AudioRef:      raw.AudioRef,
		Placements:    make([]Placement, 0, len(raw.Placements)),
		Lights:        make([]Light, 0, len(raw.Lights)),
		Triggers:      make([]TriggerVolume, 0, len(raw.Triggers)),
		CameraBounds:  CameraBounds{Min: *raw.CameraBounds.Min, Max: *raw.CameraBounds.Max},
	}
	if err := validateOptionalIdentifier("ambient_preset", scene.AmbientPreset); err != nil {
		return Scene{}, err
	}
	if err := validateOptionalIdentifier("audio_ref", scene.AudioRef); err != nil {
		return Scene{}, err
	}

	// Placements and lights both create entities at scene load, so they
	// share one entity ID namespace (GDD CR-4 "lighting entities").
	entityIDs := make(map[string]bool, len(raw.Placements)+len(raw.Lights))
	for i, p := range raw.Placements {
		if strings.TrimSpace(p.ID) == "" {
			return Scene{}, fmt.Errorf("%w: placements[%d].id", ErrMissingSceneField, i)
		}
		if err := ValidateKebabCase(p.ID); err != nil {
			return Scene{}, fmt.Errorf("placements[%d].id: %w", i, err)
		}
		if strings.TrimSpace(p.AssetRef) == "" {
			return Scene{}, fmt.Errorf("%w: placements[%d].asset_ref", ErrMissingSceneField, i)
		}
		if err := ValidateKebabCase(p.AssetRef); err != nil {
			return Scene{}, fmt.Errorf("placements[%d].asset_ref: %w", i, err)
		}
		if entityIDs[p.ID] {
			return Scene{}, fmt.Errorf("%w: %q", ErrDuplicateEntityID, p.ID)
		}
		entityIDs[p.ID] = true
		components := make([]string, 0, len(p.InitialComponents))
		for j, c := range p.InitialComponents {
			if err := ValidateKebabCase(c); err != nil {
				return Scene{}, fmt.Errorf("placements[%d].initial_components[%d]: %w", i, j, err)
			}
			components = append(components, c)
		}
		scene.Placements = append(scene.Placements, Placement{
			ID:                p.ID,
			AssetRef:          p.AssetRef,
			Position:          p.Position,
			Rotation:          p.Rotation,
			Scale:             p.Scale,
			InitialComponents: components,
		})
	}

	for i, l := range raw.Lights {
		if strings.TrimSpace(l.ID) == "" {
			return Scene{}, fmt.Errorf("%w: lights[%d].id", ErrMissingSceneField, i)
		}
		if err := ValidateKebabCase(l.ID); err != nil {
			return Scene{}, fmt.Errorf("lights[%d].id: %w", i, err)
		}
		if entityIDs[l.ID] {
			return Scene{}, fmt.Errorf("%w: %q (placements and lights share the scene entity namespace)", ErrDuplicateEntityID, l.ID)
		}
		entityIDs[l.ID] = true
		switch l.Type {
		case LightTypePoint, LightTypeDirectional, LightTypeAmbientProbe:
		default:
			return Scene{}, fmt.Errorf("%w: lights[%d].type %q (want %q, %q, or %q)",
				ErrInvalidLightType, i, l.Type,
				LightTypePoint, LightTypeDirectional, LightTypeAmbientProbe)
		}
		scene.Lights = append(scene.Lights, Light{
			ID:        l.ID,
			Type:      l.Type,
			Position:  l.Position,
			Rotation:  l.Rotation,
			Color:     l.Color,
			Range:     l.Range,
			Intensity: l.Intensity,
		})
	}

	triggerNames := make(map[string]bool, len(raw.Triggers))
	for i, t := range raw.Triggers {
		if strings.TrimSpace(t.Name) == "" {
			return Scene{}, fmt.Errorf("%w: triggers[%d].name", ErrMissingSceneField, i)
		}
		if err := ValidateKebabCase(t.Name); err != nil {
			return Scene{}, fmt.Errorf("triggers[%d].name: %w", i, err)
		}
		if triggerNames[t.Name] {
			return Scene{}, fmt.Errorf("%w: trigger name %q", ErrDuplicateEntityID, t.Name)
		}
		triggerNames[t.Name] = true
		if t.Min == nil || t.Max == nil {
			return Scene{}, fmt.Errorf("%w: triggers[%d].min/max", ErrMissingSceneField, i)
		}
		if err := validateBounds(*t.Min, *t.Max, fmt.Sprintf("triggers[%d]", i)); err != nil {
			return Scene{}, err
		}
		if err := validateOptionalIdentifier(fmt.Sprintf("triggers[%d].enter_event", i), t.EnterEvent); err != nil {
			return Scene{}, err
		}
		if err := validateOptionalIdentifier(fmt.Sprintf("triggers[%d].exit_event", i), t.ExitEvent); err != nil {
			return Scene{}, err
		}
		enabled := true // a trigger without an explicit flag is on (GDD CR-4)
		if t.Enabled != nil {
			enabled = *t.Enabled
		}
		scene.Triggers = append(scene.Triggers, TriggerVolume{
			Name:       t.Name,
			Min:        *t.Min,
			Max:        *t.Max,
			EnterEvent: t.EnterEvent,
			ExitEvent:  t.ExitEvent,
			Enabled:    enabled,
		})
	}

	cellSize := raw.NavMesh.CellSize
	if cellSize <= 0 {
		cellSize = DefaultNavMeshCellSize
	}
	scene.NavMesh.CellSize = cellSize
	scene.NavMesh.Polygons = make([][]matrix.Vec3, 0, len(raw.NavMesh.Polygons))
	for i, poly := range raw.NavMesh.Polygons {
		if len(poly) < 3 {
			return Scene{}, fmt.Errorf("%w: polygons[%d] has %d vertices, want at least 3",
				ErrInvalidNavMesh, i, len(poly))
		}
		scene.NavMesh.Polygons = append(scene.NavMesh.Polygons, poly)
	}
	return scene, nil
}

// ValidatePrefabRefs checks every placement's asset_ref against the
// available prefab set (the imported asset library, ADR-0004 "Placed"
// state). Every missing reference is reported once, in placement order, in
// a single error; a scene whose refs all exist passes with nil. This is
// the "all prefab references exist" gate — the parser cannot know the
// library, so the library is injected here. The caller may still load a
// scene with missing refs (GDD edge case: that placement fails, others
// load normally); the error names exactly which refs are missing.
//
//	library, err := loadLibrary() // asset registry, not yet implemented
//	if err := asset.ValidatePrefabRefs(scene, library.PrefabIDs()); err != nil {
//		return err // names every unknown prefab ref, in placement order
//	}
func ValidatePrefabRefs(scene Scene, availablePrefabs []string) error {
	available := make(map[string]bool, len(availablePrefabs))
	for _, p := range availablePrefabs {
		available[p] = true
	}
	var missing []string
	reported := make(map[string]bool)
	for _, pl := range scene.Placements {
		if !available[pl.AssetRef] && !reported[pl.AssetRef] {
			reported[pl.AssetRef] = true
			missing = append(missing, pl.AssetRef)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnknownPrefabRef, strings.Join(missing, ", "))
}

// validateBounds returns ErrInvalidBounds when min exceeds max on any axis.
func validateBounds(min, max matrix.Vec3, what string) error {
	if min.X() > max.X() || min.Y() > max.Y() || min.Z() > max.Z() {
		return fmt.Errorf("%w: %s min %v exceeds max %v", ErrInvalidBounds, what, min, max)
	}
	return nil
}

// validateOptionalIdentifier enforces kebab-case on optional identifier
// fields; empty values pass (the field is simply unused).
func validateOptionalIdentifier(field, value string) error {
	if value == "" {
		return nil
	}
	if err := ValidateKebabCase(value); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

// Unit tests for Story ASSET-003: Scene format & entity placement
// (TR-asset-005, ADR-0004, GDD CR-4). ParseScene and ValidatePrefabRefs are
// pure functions of their inputs — no file I/O, no engine state — so every
// rule is exercised with inline YAML, deterministically. Test naming:
// Test[Feature]_[Scenario]_[Expected].
package asset_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"kaijuengine.com/matrix"

	"marrow-of-empire/asset"
)

// sceneExtraSections is the lighting/triggers/navmesh/camera_bounds tail of
// a valid scene document. Concatenated after sceneYAMLWithPlacements to
// build full-scene fixtures.
const sceneExtraSections = `
lights:
  - id: lantern-01
    type: point
    position: [12, 2, -18]
    color: [1.0, 0.7, 0.4, 1.0]
    range: 8.0
    intensity: 1.0
  - id: sun-moon
    type: directional
    rotation: [45, -30, 0]
    intensity: 1.4
triggers:
  - name: grove-dialogue
    min: [5, 0, -30]
    max: [15, 4, -10]
    enter_event: dialogue-grove
    exit_event: dialogue-grove-end
  - name: vault-guard
    min: [-5, 0, 5]
    max: [5, 3, 15]
    enabled: false
navmesh:
  cell_size: 0.5
  polygons:
    - [[0, 0, 0], [20, 0, 0], [20, 0, 20]]
    - [[0, 0, 0], [20, 0, 20], [0, 0, 20]]
camera_bounds:
  min: [-50, 0, -50]
  max: [50, 20, 50]
`

// sceneYAMLWithPlacements builds a valid scene header plus n placements of
// asset_ref "enchanted-oak": entity "grove-oak-NN" at [N, 0, -N], rotated
// 45 degrees around Y, with visual + audio initial components.
func sceneYAMLWithPlacements(n int) string {
	var sb strings.Builder
	sb.WriteString("name: borderland-east-grove\nregion_id: borderland-east\n" +
		"ambient_preset: golden-hour\naudio_ref: ambience-forest-birds\nplacements:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb,
			"  - id: grove-oak-%02d\n    asset_ref: enchanted-oak\n"+
				"    position: [%d, 0, %d]\n    rotation: [0, 45, 0]\n"+
				"    scale: [1, 1, 1]\n    initial_components: [visual, audio]\n",
			i, i, -i)
	}
	return sb.String()
}

// TestSceneParse_ValidScene_AllSectionsParsed: a complete scene with 50
// entity placements (the acceptance-criteria scale) parses with every
// section intact — metadata, placements with correct transforms and initial
// components, lights, triggers with their enable defaults, nav mesh, and
// camera bounds.
func TestSceneParse_ValidScene_AllSectionsParsed(t *testing.T) {
	data := []byte(sceneYAMLWithPlacements(50) + sceneExtraSections)
	scene, err := asset.ParseScene(data)
	if err != nil {
		t.Fatalf("ParseScene: %v", err)
	}
	if scene.Name != "borderland-east-grove" {
		t.Errorf("Name = %q, want %q", scene.Name, "borderland-east-grove")
	}
	if scene.RegionID != "borderland-east" {
		t.Errorf("RegionID = %q, want %q", scene.RegionID, "borderland-east")
	}
	if scene.AmbientPreset != "golden-hour" || scene.AudioRef != "ambience-forest-birds" {
		t.Errorf("metadata = %q/%q, want golden-hour/ambience-forest-birds",
			scene.AmbientPreset, scene.AudioRef)
	}
	if len(scene.Placements) != 50 {
		t.Fatalf("len(Placements) = %d, want 50", len(scene.Placements))
	}
	first := scene.Placements[0]
	if first.ID != "grove-oak-00" || first.AssetRef != "enchanted-oak" {
		t.Errorf("first placement = %q/%q, want grove-oak-00/enchanted-oak", first.ID, first.AssetRef)
	}
	if first.Position != matrix.NewVec3(0, 0, 0) {
		t.Errorf("first position = %v, want [0 0 0]", first.Position)
	}
	if first.Rotation != matrix.NewVec3(0, 45, 0) {
		t.Errorf("first rotation = %v, want [0 45 0]", first.Rotation)
	}
	if first.Scale != matrix.NewVec3(1, 1, 1) {
		t.Errorf("first scale = %v, want [1 1 1]", first.Scale)
	}
	if len(first.InitialComponents) != 2 || first.InitialComponents[0] != "visual" ||
		first.InitialComponents[1] != "audio" {
		t.Errorf("first initial components = %v, want [visual audio]", first.InitialComponents)
	}
	last := scene.Placements[49]
	if last.Position != matrix.NewVec3(49, 0, -49) {
		t.Errorf("last position = %v, want [49 0 -49]", last.Position)
	}

	if len(scene.Lights) != 2 {
		t.Fatalf("len(Lights) = %d, want 2", len(scene.Lights))
	}
	lantern := scene.Lights[0]
	if lantern.ID != "lantern-01" || lantern.Type != asset.LightTypePoint {
		t.Errorf("lantern = %q/%q, want lantern-01/point", lantern.ID, lantern.Type)
	}
	if lantern.Position != matrix.NewVec3(12, 2, -18) {
		t.Errorf("lantern position = %v, want [12 2 -18]", lantern.Position)
	}
	if lantern.Color != matrix.NewColor(1.0, 0.7, 0.4, 1.0) {
		t.Errorf("lantern color = %v, want [1 0.7 0.4 1]", lantern.Color)
	}
	if lantern.Range != 8.0 || lantern.Intensity != 1.0 {
		t.Errorf("lantern range/intensity = %v/%v, want 8/1", lantern.Range, lantern.Intensity)
	}
	if sun := scene.Lights[1]; sun.Type != asset.LightTypeDirectional || sun.Intensity != 1.4 {
		t.Errorf("directional light = %q/%v, want directional/1.4", sun.Type, sun.Intensity)
	}

	if len(scene.Triggers) != 2 {
		t.Fatalf("len(Triggers) = %d, want 2", len(scene.Triggers))
	}
	dialogue := scene.Triggers[0]
	if dialogue.Name != "grove-dialogue" || !dialogue.Enabled {
		t.Errorf("dialogue trigger = %q enabled=%v, want grove-dialogue enabled", dialogue.Name, dialogue.Enabled)
	}
	if dialogue.Min != matrix.NewVec3(5, 0, -30) || dialogue.Max != matrix.NewVec3(15, 4, -10) {
		t.Errorf("dialogue bounds = %v..%v, want [5 0 -30]..[15 4 -10]", dialogue.Min, dialogue.Max)
	}
	if dialogue.EnterEvent != "dialogue-grove" || dialogue.ExitEvent != "dialogue-grove-end" {
		t.Errorf("dialogue events = %q/%q, want dialogue-grove/dialogue-grove-end",
			dialogue.EnterEvent, dialogue.ExitEvent)
	}
	if vault := scene.Triggers[1]; vault.Enabled {
		t.Error("vault-guard trigger has enabled: false, but Enabled = true")
	}

	if scene.NavMesh.CellSize != 0.5 {
		t.Errorf("NavMesh.CellSize = %v, want 0.5", scene.NavMesh.CellSize)
	}
	if len(scene.NavMesh.Polygons) != 2 || len(scene.NavMesh.Polygons[0]) != 3 {
		t.Fatalf("NavMesh.Polygons = %v, want 2 polygons of 3 vertices", scene.NavMesh.Polygons)
	}
	if scene.NavMesh.Polygons[0][0] != matrix.NewVec3(0, 0, 0) {
		t.Errorf("nav polygon first vertex = %v, want [0 0 0]", scene.NavMesh.Polygons[0][0])
	}

	if scene.CameraBounds.Min != matrix.NewVec3(-50, 0, -50) ||
		scene.CameraBounds.Max != matrix.NewVec3(50, 20, 50) {
		t.Errorf("CameraBounds = %v..%v, want [-50 0 -50]..[50 20 50]",
			scene.CameraBounds.Min, scene.CameraBounds.Max)
	}
}

// TestSceneParse_DuplicateEntityID_Rejected: entity IDs must be unique
// across the whole scene — placements and lights share the entity ID
// namespace (both create entities at load), and trigger names are a
// distinct unique namespace.
func TestSceneParse_DuplicateEntityID_Rejected(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"two placements with the same id", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
  - id: prop-01
    asset_ref: stone-wall
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"placement id collides with light id", `
name: test-scene
region_id: test-region
placements:
  - id: lantern-01
    asset_ref: stone-pillar
lights:
  - id: lantern-01
    type: point
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"duplicate trigger name", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
triggers:
  - name: gate
    min: [0, 0, 0]
    max: [2, 2, 2]
  - name: gate
    min: [5, 0, 5]
    max: [7, 2, 7]
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := asset.ParseScene([]byte(tc.yaml))
			if !errors.Is(err, asset.ErrDuplicateEntityID) {
				t.Fatalf("ParseScene = %v, want ErrDuplicateEntityID", err)
			}
		})
	}
}

// TestSceneParse_MissingRequiredField_Error: hand-authored scenes must be
// complete — missing scene name, region ID, placement id or asset_ref, or
// the camera bounds section all reject at parse.
func TestSceneParse_MissingRequiredField_Error(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"missing name", `
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"missing region_id", `
name: test-scene
placements:
  - id: prop-01
    asset_ref: stone-pillar
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"placement missing id", `
name: test-scene
region_id: test-region
placements:
  - asset_ref: stone-pillar
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"placement missing asset_ref", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
		{"missing camera_bounds section", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
`},
		{"camera_bounds missing corners", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
camera_bounds: {}
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := asset.ParseScene([]byte(tc.yaml))
			if !errors.Is(err, asset.ErrMissingSceneField) {
				t.Fatalf("ParseScene = %v, want ErrMissingSceneField", err)
			}
		})
	}
}

// TestSceneParse_InvalidBounds_Rejected: axis-aligned volumes (camera
// bounds, trigger volumes) must have Min <= Max on every axis — an inverted
// rectangle cannot be a playable area or a trigger region.
func TestSceneParse_InvalidBounds_Rejected(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"camera bounds min > max on X", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
camera_bounds:
  min: [10, 0, 0]
  max: [0, 10, 10]
`},
		{"camera bounds min > max on Z", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
camera_bounds:
  min: [0, 0, 10]
  max: [10, 10, 0]
`},
		{"trigger bounds min > max on Y", `
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
triggers:
  - name: gate
    min: [0, 4, 0]
    max: [2, 0, 2]
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := asset.ParseScene([]byte(tc.yaml))
			if !errors.Is(err, asset.ErrInvalidBounds) {
				t.Fatalf("ParseScene = %v, want ErrInvalidBounds", err)
			}
		})
	}
}

// TestSceneParse_InvalidContent_Rejected: content-level rules reject —
// nav mesh polygons need at least 3 vertices, and light types must be one
// of the three supported kinds.
func TestSceneParse_InvalidContent_Rejected(t *testing.T) {
	t.Run("nav polygon with 2 vertices", func(t *testing.T) {
		_, err := asset.ParseScene([]byte(`
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
navmesh:
  polygons:
    - [[0, 0, 0], [5, 0, 5]]
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`))
		if !errors.Is(err, asset.ErrInvalidNavMesh) {
			t.Fatalf("ParseScene = %v, want ErrInvalidNavMesh", err)
		}
	})
	t.Run("unknown light type", func(t *testing.T) {
		_, err := asset.ParseScene([]byte(`
name: test-scene
region_id: test-region
placements:
  - id: prop-01
    asset_ref: stone-pillar
lights:
  - id: laser-01
    type: laser
camera_bounds:
  min: [0, 0, 0]
  max: [10, 10, 10]
`))
		if !errors.Is(err, asset.ErrInvalidLightType) {
			t.Fatalf("ParseScene = %v, want ErrInvalidLightType", err)
		}
	})
}

// TestSceneParse_MalformedYAML_Error: non-YAML input is a parse error, not
// a silent empty scene.
func TestSceneParse_MalformedYAML_Error(t *testing.T) {
	_, err := asset.ParseScene([]byte("placements: [unclosed"))
	if err == nil {
		t.Fatal("ParseScene on malformed YAML = nil, want an error")
	}
}

// TestSceneValidatePrefabs_MissingRef_Rejected: ValidatePrefabRefs rejects
// placements whose asset_ref is not in the library, naming every unknown
// ref exactly once in placement order; a scene whose refs all exist passes.
func TestSceneValidatePrefabs_MissingRef_Rejected(t *testing.T) {
	scene := asset.Scene{
		Placements: []asset.Placement{
			{ID: "p1", AssetRef: "enchanted-oak"},
			{ID: "p2", AssetRef: "stone-pillar"},
			{ID: "p3", AssetRef: "stone-pillar"}, // duplicate ref reported once
		},
	}
	err := asset.ValidatePrefabRefs(scene, []string{"enchanted-oak"})
	if !errors.Is(err, asset.ErrUnknownPrefabRef) {
		t.Fatalf("ValidatePrefabRefs = %v, want ErrUnknownPrefabRef", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "stone-pillar") {
		t.Errorf("error %q does not name the missing ref stone-pillar", msg)
	}
	if strings.Count(msg, "stone-pillar") != 1 {
		t.Errorf("error %q reports stone-pillar %d times, want exactly once",
			msg, strings.Count(msg, "stone-pillar"))
	}

	if err := asset.ValidatePrefabRefs(scene, []string{"enchanted-oak", "stone-pillar"}); err != nil {
		t.Errorf("all refs available: ValidatePrefabRefs = %v, want nil", err)
	}
	if err := asset.ValidatePrefabRefs(asset.Scene{}, nil); err != nil {
		t.Errorf("empty scene: ValidatePrefabRefs = %v, want nil", err)
	}
}

// TestSceneParse_200Placements_WithinLoadBudget: ADR-0004 budgets scene
// load at ~2s for 200 entity placements. Parsing is this layer's share of
// that budget — 200 placements must parse well within it.
func TestSceneParse_200Placements_WithinLoadBudget(t *testing.T) {
	data := []byte(sceneYAMLWithPlacements(200) + sceneExtraSections)
	start := time.Now()
	scene, err := asset.ParseScene(data)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ParseScene: %v", err)
	}
	if len(scene.Placements) != 200 {
		t.Errorf("len(Placements) = %d, want 200", len(scene.Placements))
	}
	if elapsed >= 2*time.Second {
		t.Errorf("parse of 200 placements took %v, exceeding the 2s scene load budget", elapsed)
	}
}

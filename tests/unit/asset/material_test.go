// Unit tests for Story ASSET-002: Material mapping & validation
// (TR-asset-003/004, ADR-0004). ParseMaterialMap and ValidateAsset are pure
// functions of their inputs — no file I/O, no engine state — so every rule
// is exercised with inline YAML and struct literals, deterministically.
// Test naming: Test[Feature]_[Scenario]_[Expected].
package asset_test

import (
	"errors"
	"strings"
	"testing"

	"marrow-of-empire/asset"
)

// validMappingYAML is a hand-authored table in the ADR-0004 schema: pack +
// source/project/palette entries. Palette is optional; the families here are
// all in the art bible (amber, moss, stone, indigo, ivory, oxblood).
const validMappingYAML = `
pack: dark-fantasy
materials:
  - source: Door_DarkWood
    project: wood-warm
    palette: stone
  - source: Iron_Gate
    project: metal-dark
    palette: stone
  - source: Mossy_Stone_Wall
    project: stone-moss
    palette: moss
`

// findIssue returns the first validation issue with the given code, or nil.
func findIssue(issues asset.Issues, code asset.IssueCode) *asset.ValidationIssue {
	for i := range issues {
		if issues[i].Code == code {
			return &issues[i]
		}
	}
	return nil
}

// TestMaterialMap_ValidTable_LoadsAndResolves: a well-formed table parses,
// exposes its entries, and resolves mapped source materials to their project
// definitions with no review warning.
func TestMaterialMap_ValidTable_LoadsAndResolves(t *testing.T) {
	m, err := asset.ParseMaterialMap([]byte(validMappingYAML))
	if err != nil {
		t.Fatalf("ParseMaterialMap: %v", err)
	}
	if m.Pack != "dark-fantasy" {
		t.Errorf("Pack = %q, want %q", m.Pack, "dark-fantasy")
	}
	if len(m.Materials) != 3 {
		t.Fatalf("len(Materials) = %d, want 3", len(m.Materials))
	}
	e, ok := m.Lookup("Door_DarkWood")
	if !ok {
		t.Fatal("Lookup(Door_DarkWood): not found")
	}
	if e.Project != "wood-warm" || e.Palette != "stone" {
		t.Errorf("entry = %+v, want project %q palette %q", e, "wood-warm", "stone")
	}
	projectID, warn := m.ResolveMaterial("Door_DarkWood")
	if projectID != "wood-warm" {
		t.Errorf("ResolveMaterial = %q, want %q", projectID, "wood-warm")
	}
	if warn != "" {
		t.Errorf("ResolveMaterial warning = %q, want none for a mapped material", warn)
	}
}

// TestMaterialMap_UnmappedMaterial_DefaultGreyWithWarning: a source material
// absent from the table resolves to DefaultGreyMaterial with a human-review
// warning naming the material — the pipeline never guesses (ADR-0004).
func TestMaterialMap_UnmappedMaterial_DefaultGreyWithWarning(t *testing.T) {
	m, err := asset.ParseMaterialMap([]byte(validMappingYAML))
	if err != nil {
		t.Fatalf("ParseMaterialMap: %v", err)
	}
	projectID, warn := m.ResolveMaterial("Ghost_Material")
	if projectID != asset.DefaultGreyMaterial {
		t.Errorf("ResolveMaterial = %q, want default grey %q",
			projectID, asset.DefaultGreyMaterial)
	}
	if warn == "" {
		t.Fatal("expected a human-review warning for an unmapped material, got none")
	}
	if !strings.Contains(warn, "Ghost_Material") {
		t.Errorf("warning %q does not name the unmapped material", warn)
	}
}

// TestAssetValidation_StaticPropOverBudget_Rejected: an 8000-tri static prop
// exceeds the 5000 budget and is rejected with a blocking decimation flag
// (GDD CR-3).
func TestAssetValidation_StaticPropOverBudget_Rejected(t *testing.T) {
	issues := asset.ValidateAsset(asset.AssetSpec{
		Name:          "stone-pillar",
		TriangleCount: 8000,
		MaterialSlots: 2,
	}, asset.ValidationLimits{})
	dec := findIssue(issues, asset.IssueDecimation)
	if dec == nil {
		t.Fatal("expected a decimation issue")
	}
	if dec.Severity != asset.SeverityError {
		t.Errorf("decimation severity = %v, want error", dec.Severity)
	}
	if !issues.Blocking() {
		t.Error("8000-tri static prop must be blocking, got no error severity")
	}
}

// TestAssetValidation_HeroAssetOverBudget_Warns: a 25000-tri hero asset
// exceeds the 20000 budget but warns for manual review instead of blocking
// (ADR-0004 hero threshold). A hero under budget passes clean.
func TestAssetValidation_HeroAssetOverBudget_Warns(t *testing.T) {
	over := asset.ValidateAsset(asset.AssetSpec{
		Name:          "emperor-throne",
		TriangleCount: 25000,
		IsHero:        true,
		MaterialSlots: 4,
	}, asset.ValidationLimits{})
	dec := findIssue(over, asset.IssueDecimation)
	if dec == nil {
		t.Fatal("expected a decimation issue")
	}
	if dec.Severity != asset.SeverityWarn {
		t.Errorf("hero decimation severity = %v, want warn", dec.Severity)
	}
	if over.Blocking() {
		t.Error("hero budget breach must warn, not block")
	}

	under := asset.ValidateAsset(asset.AssetSpec{
		Name:          "emperor-throne",
		TriangleCount: 12000,
		IsHero:        true,
		MaterialSlots: 4,
	}, asset.ValidationLimits{})
	if dec := findIssue(under, asset.IssueDecimation); dec != nil {
		t.Errorf("hero at 12000 tris got a decimation issue: %+v", dec)
	}
	if under.Blocking() {
		t.Error("valid hero asset must not block")
	}
}

// TestAssetValidation_InvalidName_KebabCaseError: a non-kebab-case asset
// name is rejected with a naming issue that carries a rename suggestion.
func TestAssetValidation_InvalidName_KebabCaseError(t *testing.T) {
	issues := asset.ValidateAsset(asset.AssetSpec{
		Name:          "Door_Wood",
		TriangleCount: 100,
		MaterialSlots: 1,
	}, asset.ValidationLimits{})
	naming := findIssue(issues, asset.IssueNaming)
	if naming == nil {
		t.Fatal("expected a naming issue")
	}
	if naming.Severity != asset.SeverityError {
		t.Errorf("naming severity = %v, want error", naming.Severity)
	}
	if !strings.Contains(naming.Message, "door-wood") {
		t.Errorf("naming message %q lacks the rename suggestion %q", naming.Message, "door-wood")
	}
	if !issues.Blocking() {
		t.Error("non-kebab-case name must be blocking")
	}
}

// TestMaterialMap_MissingRequiredField_ParseError: an entry missing pack,
// source, or project is a parse error — an incomplete table cannot be mapped.
func TestMaterialMap_MissingRequiredField_ParseError(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"missing project", "pack: dark-fantasy\nmaterials:\n  - source: Door_DarkWood\n"},
		{"missing source", "pack: dark-fantasy\nmaterials:\n  - project: wood-warm\n"},
		{"missing pack", "materials:\n  - source: Door_DarkWood\n    project: wood-warm\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := asset.ParseMaterialMap([]byte(tc.yaml))
			if !errors.Is(err, asset.ErrMissingField) {
				t.Fatalf("ParseMaterialMap = %v, want ErrMissingField", err)
			}
		})
	}
}

// TestAssetValidation_MaterialSlotsOverLimit_Rejected: 9 material slots on
// one mesh exceed the 8-slot budget and reject with a consolidation flag
// (GDD CR-3).
func TestAssetValidation_MaterialSlotsOverLimit_Rejected(t *testing.T) {
	issues := asset.ValidateAsset(asset.AssetSpec{
		Name:          "barracks",
		TriangleCount: 100,
		MaterialSlots: 9,
	}, asset.ValidationLimits{})
	cons := findIssue(issues, asset.IssueConsolidation)
	if cons == nil {
		t.Fatal("expected a consolidation issue")
	}
	if cons.Severity != asset.SeverityError {
		t.Errorf("consolidation severity = %v, want error", cons.Severity)
	}
	if !issues.Blocking() {
		t.Error("9 material slots must be blocking")
	}
}

// TestAssetValidation_TextureOverLimit_DownscaleWarning: a 3000x3000 texture
// exceeds 2048x2048 and warns with the downscale target; the warning does not
// block the asset (GDD CR-3; resizing itself is ASSET-003).
func TestAssetValidation_TextureOverLimit_DownscaleWarning(t *testing.T) {
	issues := asset.ValidateAsset(asset.AssetSpec{
		Name:          "chapel-door",
		TriangleCount: 100,
		MaterialSlots: 1,
		TextureWidth:  3000,
		TextureHeight: 3000,
	}, asset.ValidationLimits{})
	td := findIssue(issues, asset.IssueTextureDownscale)
	if td == nil {
		t.Fatal("expected a texture-downscale issue")
	}
	if td.Severity != asset.SeverityWarn {
		t.Errorf("texture-downscale severity = %v, want warn", td.Severity)
	}
	if !strings.Contains(td.Message, "2048") {
		t.Errorf("message %q must name the 2048 downscale target", td.Message)
	}
	if issues.Blocking() {
		t.Error("texture over-limit must warn, not block")
	}
}

// TestMaterialMap_DuplicateSource_ParseError: two entries mapping the same
// source material are ambiguous and rejected at parse.
func TestMaterialMap_DuplicateSource_ParseError(t *testing.T) {
	_, err := asset.ParseMaterialMap([]byte(`
pack: dark-fantasy
materials:
  - source: Door_DarkWood
    project: wood-warm
  - source: Door_DarkWood
    project: wood-cold
`))
	if !errors.Is(err, asset.ErrDuplicateSource) {
		t.Fatalf("ParseMaterialMap = %v, want ErrDuplicateSource", err)
	}
}

// TestAssetValidation_PaletteOutsideFamilies_Warns: a palette family outside
// the art bible's six families warns per material and never blocks (GDD
// CR-3); in-family palettes pass silently.
func TestAssetValidation_PaletteOutsideFamilies_Warns(t *testing.T) {
	issues := asset.ValidateAsset(asset.AssetSpec{
		Name:          "vault-door",
		TriangleCount: 100,
		MaterialSlots: 2,
		Materials: []asset.MaterialEntry{
			{Source: "Door_DarkWood", Project: "wood-warm", Palette: "stone"},
			{Source: "Door_Neon", Project: "neon-sign", Palette: "neon"},
		},
	}, asset.ValidationLimits{})
	pal := findIssue(issues, asset.IssuePalette)
	if pal == nil {
		t.Fatal("expected a palette issue")
	}
	if pal.Severity != asset.SeverityWarn {
		t.Errorf("palette severity = %v, want warn", pal.Severity)
	}
	count := 0
	for _, i := range issues {
		if i.Code == asset.IssuePalette {
			count++
		}
	}
	if count != 1 {
		t.Errorf("palette issues = %d, want 1 (only the out-of-family material)", count)
	}
	if issues.Blocking() {
		t.Error("palette breach must warn, not block")
	}
}

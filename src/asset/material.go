// Material mapping and asset validation for the Asset & Scene Pipeline
// (ASSET-002, TR-asset-003/004, ADR-0004): per-pack YAML mapping tables that
// translate KitBash3D source material names into project material
// definitions, plus the CR-3 validation rules that gate assets before they
// enter scenes.
//
// Rules enforced here (ADR-0004 / GDD CR-3):
//   - Unmapped materials never guess: they resolve to DefaultGreyMaterial
//     with a human-review warning.
//   - Static props <=5000 tris (reject), hero assets <=20000 tris (warn).
//   - Textures <=2048x2048 (downscale warning; resizing itself is ASSET-003).
//   - Material slots <=8 per mesh (reject).
//   - Names are kebab-case (reject with rename suggestion).
//   - Material palettes stay within the six art bible families (warn).
package asset

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Material mapping errors. These are sentinel errors, use errors.Is to match
// them.
var (
	// ErrMissingField is returned when a mapping table entry is missing a
	// required field (pack, source, or project). Hand-authored tables must
	// be complete — an incomplete entry cannot be mapped.
	ErrMissingField = errors.New("asset: material mapping entry missing required field")
	// ErrDuplicateSource is returned when two entries map the same source
	// material name. A source name must map to exactly one project material;
	// ambiguity is rejected, never guessed.
	ErrDuplicateSource = errors.New("asset: duplicate source material in mapping table")
	// ErrInvalidName is returned when a pack or project material name is not
	// kebab-case (CR-3 naming rule).
	ErrInvalidName = errors.New("asset: name is not kebab-case")
)

// Defaults for material mapping and validation. Override via ValidationLimits
// — zero values fall back to these constants (same pattern as Config).
const (
	// DefaultGreyMaterial is the project material definition assigned to
	// unmapped source materials. The pipeline never guesses material intent
	// (ADR-0004); a human must author the mapping, and until then the asset
	// renders grey.
	DefaultGreyMaterial = "material-default-grey"
	// DefaultMaxTrisStatic bounds static prop geometry (GDD tuning knob
	// max_tris_static_prop). Exceeding it rejects the asset with a
	// decimation flag.
	DefaultMaxTrisStatic = 5000
	// DefaultMaxTrisHero bounds hero asset geometry (GDD tuning knob
	// max_tris_hero_asset). Exceeding it warns for manual review, never
	// rejects (ADR-0004).
	DefaultMaxTrisHero = 20000
	// DefaultMaxTextureResolution bounds single textures per side (GDD
	// tuning knob max_texture_resolution). Exceeding it warns and requires
	// downscale to this limit (resizing is ASSET-003).
	DefaultMaxTextureResolution = 2048
	// DefaultMaxMaterialSlots bounds material slots per mesh (GDD CR-3).
	// Exceeding it rejects the asset with a consolidation flag.
	DefaultMaxMaterialSlots = 8
)

// artBiblePaletteFamilies is the art bible's six color families
// (design/art/art-bible.md section 2.1). Palette compliance warns, never
// rejects (GDD CR-3).
var artBiblePaletteFamilies = map[string]bool{
	"amber":   true,
	"moss":    true,
	"stone":   true,
	"indigo":  true,
	"ivory":   true,
	"oxblood": true,
}

// IsPaletteFamily reports whether family is one of the six art bible color
// families: amber, moss, stone, indigo, ivory, oxblood.
//
//	asset.IsPaletteFamily("stone")  // true
//	asset.IsPaletteFamily("neon")   // false
func IsPaletteFamily(family string) bool {
	return artBiblePaletteFamilies[family]
}

// MaterialEntry maps one source material name to a project material
// definition. Tables are hand-authored per KitBash3D pack (ADR-0004).
type MaterialEntry struct {
	// Source is the material name as shipped in the pack (e.g.
	// "Door_DarkWood"). Matched exactly.
	Source string
	// Project is the project material definition ID (kebab-case).
	Project string
	// Palette is the art bible color family of the project material
	// (amber, moss, stone, indigo, ivory, oxblood). Optional — an empty
	// family warns during validation for manual review.
	Palette string
}

// MaterialMap is one pack's parsed YAML mapping table.
type MaterialMap struct {
	// Pack is the pack identifier (kebab-case, e.g. "dark-fantasy").
	Pack string
	// Materials lists the pack's mappings in table order.
	Materials []MaterialEntry
}

// materialMapYAML mirrors the hand-authored YAML schema:
//
//	pack: dark-fantasy
//	materials:
//	  - source: Door_DarkWood
//	    project: wood-warm
//	    palette: stone
type materialMapYAML struct {
	Pack      string `yaml:"pack"`
	Materials []struct {
		Source  string `yaml:"source"`
		Project string `yaml:"project"`
		Palette string `yaml:"palette"`
	} `yaml:"materials"`
}

// ParseMaterialMap parses and validates one pack's YAML mapping table.
// Required fields (pack, source, project) must be present, project names
// must be kebab-case, and source names must be unique. Palette is optional —
// its compliance is checked at asset validation, warn-only.
//
//	table, err := asset.ParseMaterialMap([]byte(yamlData))
//	if err != nil {
//		return err // missing fields, duplicates, or non-kebab-case IDs
//	}
func ParseMaterialMap(data []byte) (MaterialMap, error) {
	var raw materialMapYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return MaterialMap{}, fmt.Errorf("parse material mapping: %w", err)
	}
	if strings.TrimSpace(raw.Pack) == "" {
		return MaterialMap{}, fmt.Errorf("%w: pack", ErrMissingField)
	}
	if err := ValidateKebabCase(raw.Pack); err != nil {
		return MaterialMap{}, fmt.Errorf("pack %q: %w", raw.Pack, err)
	}
	m := MaterialMap{
		Pack:      raw.Pack,
		Materials: make([]MaterialEntry, 0, len(raw.Materials)),
	}
	seen := make(map[string]bool, len(raw.Materials))
	for i, e := range raw.Materials {
		if strings.TrimSpace(e.Source) == "" {
			return MaterialMap{}, fmt.Errorf("%w: materials[%d].source", ErrMissingField, i)
		}
		if strings.TrimSpace(e.Project) == "" {
			return MaterialMap{}, fmt.Errorf("%w: materials[%d].project", ErrMissingField, i)
		}
		if err := ValidateKebabCase(e.Project); err != nil {
			return MaterialMap{}, fmt.Errorf("project %q: %w", e.Project, err)
		}
		if seen[e.Source] {
			return MaterialMap{}, fmt.Errorf("%w: %q", ErrDuplicateSource, e.Source)
		}
		seen[e.Source] = true
		m.Materials = append(m.Materials, MaterialEntry{
			Source:  e.Source,
			Project: e.Project,
			Palette: e.Palette,
		})
	}
	return m, nil
}

// LoadMaterialMap reads and parses the mapping table at path. Thin wrapper
// over ParseMaterialMap for pipeline use.
//
//	table, err := asset.LoadMaterialMap("assets/material-maps/dark-fantasy.yaml")
func LoadMaterialMap(path string) (MaterialMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MaterialMap{}, err
	}
	return ParseMaterialMap(data)
}

// Lookup returns the entry for a source material name (exact match).
// Tables are small (<100 entries per pack) and lookup stays a linear scan —
// well under the ADR-0004 <1ms budget.
func (m MaterialMap) Lookup(sourceName string) (MaterialEntry, bool) {
	for _, e := range m.Materials {
		if e.Source == sourceName {
			return e, true
		}
	}
	return MaterialEntry{}, false
}

// ResolveMaterial maps a source material name to a project material
// definition ID. Mapped names return their project ID with no warning.
// Unmapped names resolve to DefaultGreyMaterial with a human-review warning —
// the pipeline never guesses material intent (ADR-0004), and the asset
// renders grey until an author adds the mapping.
//
//	projectID, warn := table.ResolveMaterial("Door_DarkWood")
//	if warn != "" {
//		slog.Warn(warn) // material is grey until reviewed
//	}
func (m MaterialMap) ResolveMaterial(sourceName string) (projectID, reviewWarning string) {
	if e, ok := m.Lookup(sourceName); ok {
		return e.Project, ""
	}
	return DefaultGreyMaterial, fmt.Sprintf(
		"material %q has no mapping entry in pack %q; loaded as %s — "+
			"author a mapping entry (ADR-0004: the pipeline never guesses material intent)",
		sourceName, m.Pack, DefaultGreyMaterial)
}

// IsKebabCase reports whether name is kebab-case: lowercase ASCII letters
// and digits separated by single hyphens, no leading or trailing hyphen, no
// other characters (GDD CR-3 naming rule).
func IsKebabCase(name string) bool {
	if name == "" {
		return false
	}
	lastSep := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			lastSep = false
		case c == '-':
			if lastSep || i == 0 {
				return false
			}
			lastSep = true
		default:
			return false
		}
	}
	return !lastSep
}

// KebabCaseSuggestion derives a kebab-case form of name for rename
// suggestions: lowercased, camelCase boundaries and runs of non-alphanumeric
// characters become single hyphens, leading/trailing hyphens trimmed.
//
//	asset.KebabCaseSuggestion("Door_DarkWood") // "door-dark-wood"
//	asset.KebabCaseSuggestion("door--wood")    // "door-wood"
func KebabCaseSuggestion(name string) string {
	var sb strings.Builder
	prevLower := false
	lastSep := false
	for _, r := range name {
		upper := r >= 'A' && r <= 'Z'
		lc := r
		if upper {
			lc = r + ('a' - 'A')
		}
		isAlnum := (lc >= 'a' && lc <= 'z') || (lc >= '0' && lc <= '9')
		if isAlnum {
			if upper && prevLower && !lastSep && sb.Len() > 0 {
				sb.WriteByte('-')
			}
			sb.WriteRune(lc)
			prevLower = lc >= 'a' && lc <= 'z'
			lastSep = false
			continue
		}
		if !lastSep && sb.Len() > 0 {
			sb.WriteByte('-')
		}
		lastSep = true
		prevLower = false
	}
	return strings.Trim(sb.String(), "-")
}

// ValidateKebabCase returns ErrInvalidName (wrapping a rename suggestion)
// when name is not kebab-case.
func ValidateKebabCase(name string) error {
	if IsKebabCase(name) {
		return nil
	}
	return fmt.Errorf("%w: %q (suggestion: %q)",
		ErrInvalidName, name, KebabCaseSuggestion(name))
}

// Severity grades a validation finding. Warnings never fail an asset;
// errors block it from entering scenes (GDD CR-3).
type Severity int

const (
	// SeverityWarn reports a non-blocking finding that needs manual
	// review (hero poly budget, texture downscale, palette compliance).
	SeverityWarn Severity = iota
	// SeverityError reports a blocking finding (static poly budget,
	// material slots, naming).
	SeverityError
)

// IssueCode identifies the CR-3 rule that produced an issue. Codes are
// stable strings — downstream tooling (validation reports, editor flags)
// keys off them.
type IssueCode string

const (
	// IssueDecimation flags poly budget breach: reject static props, warn
	// hero assets.
	IssueDecimation IssueCode = "decimation"
	// IssueConsolidation flags material slot budget breach (reject).
	IssueConsolidation IssueCode = "consolidation"
	// IssueTextureDownscale flags texture resolution breach; the asset must
	// downscale to the limit (warn; resizing is ASSET-003).
	IssueTextureDownscale IssueCode = "texture-downscale"
	// IssueNaming flags non-kebab-case names (reject).
	IssueNaming IssueCode = "naming"
	// IssuePalette flags palettes outside the art bible families (warn).
	IssuePalette IssueCode = "palette"
)

// ValidationIssue is one CR-3 finding about an asset.
type ValidationIssue struct {
	// Severity grades the finding (warn or error).
	Severity Severity
	// Code identifies the rule that fired.
	Code IssueCode
	// Message is the human-readable finding, including any flag or rename
	// suggestion.
	Message string
}

// Issues is the ordered result of ValidateAsset.
type Issues []ValidationIssue

// Blocking reports whether any issue has error severity. Blocking issues
// reject the asset — it cannot enter scenes until resolved (GDD CR-3).
func (is Issues) Blocking() bool {
	for _, i := range is {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// AssetSpec is the input to ValidateAsset: what is known about one asset at
// validation time.
type AssetSpec struct {
	// Name is the asset's project name. Must be kebab-case.
	Name string
	// TriangleCount is the normalized geometry's triangle count.
	TriangleCount int
	// TextureWidth and TextureHeight are the largest single texture's
	// dimensions in pixels. Zero values skip the resolution check.
	TextureWidth  int
	TextureHeight int
	// MaterialSlots is the number of material slots on the mesh. Zero
	// values skip the slot check.
	MaterialSlots int
	// IsHero marks a hero asset: it gets the 20k poly budget and budget
	// breaches warn instead of rejecting (ADR-0004).
	IsHero bool
	// Materials lists the mapped project material entries for the asset's
	// slots; their palette families are checked against the art bible.
	Materials []MaterialEntry
}

// ValidationLimits overrides the CR-3 thresholds. Zero values fall back to
// the Default* constants (GDD tuning knobs).
type ValidationLimits struct {
	// MaxTrisStatic bounds static prop triangles (default 5000).
	MaxTrisStatic int
	// MaxTrisHero bounds hero asset triangles (default 20000).
	MaxTrisHero int
	// MaxTextureResolution bounds texture pixels per side (default 2048).
	MaxTextureResolution int
	// MaxMaterialSlots bounds material slots per mesh (default 8).
	MaxMaterialSlots int
}

// withDefaults fills zero-value limits with the project defaults.
func (l ValidationLimits) withDefaults() ValidationLimits {
	if l.MaxTrisStatic <= 0 {
		l.MaxTrisStatic = DefaultMaxTrisStatic
	}
	if l.MaxTrisHero <= 0 {
		l.MaxTrisHero = DefaultMaxTrisHero
	}
	if l.MaxTextureResolution <= 0 {
		l.MaxTextureResolution = DefaultMaxTextureResolution
	}
	if l.MaxMaterialSlots <= 0 {
		l.MaxMaterialSlots = DefaultMaxMaterialSlots
	}
	return l
}

// ValidateAsset runs the CR-3 validation rules over one asset and returns
// every finding, in deterministic order: naming, poly budget (static reject /
// hero warn), texture resolution (downscale warn), material slots (reject),
// and palette compliance (warn) per mapped material. Errors block the asset;
// warnings flag it for manual review.
//
//	issues := asset.ValidateAsset(asset.AssetSpec{
//	    Name:          "stone-pillar",
//	    TriangleCount: 8000,
//	    MaterialSlots: 2,
//	}, asset.ValidationLimits{})
//	if issues.Blocking() {
//	    return fmt.Errorf("asset rejected: %+v", issues)
//	}
func ValidateAsset(spec AssetSpec, limits ValidationLimits) Issues {
	lim := limits.withDefaults()
	var issues Issues

	if err := ValidateKebabCase(spec.Name); err != nil {
		issues = append(issues, ValidationIssue{
			Severity: SeverityError,
			Code:     IssueNaming,
			Message:  err.Error(),
		})
	}

	if spec.TriangleCount > 0 {
		switch {
		case spec.IsHero && spec.TriangleCount > lim.MaxTrisHero:
			issues = append(issues, ValidationIssue{
				Severity: SeverityWarn,
				Code:     IssueDecimation,
				Message: fmt.Sprintf("hero asset %q has %d tris, exceeding the %d budget; "+
					"flag for manual review", spec.Name, spec.TriangleCount, lim.MaxTrisHero),
			})
		case !spec.IsHero && spec.TriangleCount > lim.MaxTrisStatic:
			issues = append(issues, ValidationIssue{
				Severity: SeverityError,
				Code:     IssueDecimation,
				Message: fmt.Sprintf("static prop %q has %d tris, exceeding the %d budget; "+
					"flag for decimation or LOD generation",
					spec.Name, spec.TriangleCount, lim.MaxTrisStatic),
			})
		}
	}

	if spec.TextureWidth > 0 && spec.TextureHeight > 0 &&
		(spec.TextureWidth > lim.MaxTextureResolution || spec.TextureHeight > lim.MaxTextureResolution) {
		issues = append(issues, ValidationIssue{
			Severity: SeverityWarn,
			Code:     IssueTextureDownscale,
			Message: fmt.Sprintf("texture %dx%d exceeds %dx%d; downscale to %dx%d "+
				"(original preserved in staging)",
				spec.TextureWidth, spec.TextureHeight,
				lim.MaxTextureResolution, lim.MaxTextureResolution,
				lim.MaxTextureResolution, lim.MaxTextureResolution),
		})
	}

	if spec.MaterialSlots > 0 && spec.MaterialSlots > lim.MaxMaterialSlots {
		issues = append(issues, ValidationIssue{
			Severity: SeverityError,
			Code:     IssueConsolidation,
			Message: fmt.Sprintf("mesh %q has %d material slots, exceeding the %d limit; "+
				"flag for material consolidation",
				spec.Name, spec.MaterialSlots, lim.MaxMaterialSlots),
		})
	}

	for _, m := range spec.Materials {
		if m.Palette == "" {
			issues = append(issues, ValidationIssue{
				Severity: SeverityWarn,
				Code:     IssuePalette,
				Message: fmt.Sprintf("material %q has no palette family declared; "+
					"flag for manual review", m.Project),
			})
			continue
		}
		if !IsPaletteFamily(m.Palette) {
			issues = append(issues, ValidationIssue{
				Severity: SeverityWarn,
				Code:     IssuePalette,
				Message: fmt.Sprintf("material %q palette family %q is not in the art bible "+
					"families (amber, moss, stone, indigo, ivory, oxblood); flag for manual review",
					m.Project, m.Palette),
			})
		}
	}
	return issues
}

// Pack mixing validation for the Asset & Scene Pipeline (ASSET-004,
// TR-asset-005, ADR-0004, GDD CR-6): the two-pack integration rules that
// keep Dark Fantasy and Enchanted assets interleaving as the art bible's
// borderland signature (Section 5.5) instead of colliding.
//
// Rules enforced here:
//   - No single-pack dominance: a pack owning more than 80% of a scene's
//     visible surface area flags a dominance warning (GDD CR-6, art bible
//     5.5). This is the mechanical ally of the no-50/50-frames rule: a
//     frame that ties the packs reads as a collision, and the collision is
//     reserved for State 4 (Revelation) — the validator leans authorship
//     toward a dominant stratum without replacing the human rule.
//   - Color temperature consistency: a scene's dominant material palette
//     must match its area archetype register (art bible 2.3). A Warm
//     Borderland scene that is 80% Dark Fantasy grey stone fails the
//     archetype and warns.
//   - Scale coherence: both packs normalize the same nominal size (the
//     door reference, ASSET-001) within 10% of each other (GDD CR-6).
//   - Legend filter at import: assets reading modern-engineering are
//     treated (redesigned as ornament: overgrowth, re-paint, modification)
//     or rejected at import; the human pass is final (art bible 1.1, 5.5,
//     5.8).
//   - Per-pack treatment: both packs pass through the same import pipeline
//     (ASSET-001) and the same painterly grade, differing only in their
//     material mapping tables (ASSET-002) — no asset remembers which pack
//     it came from (ADR-0004).
//
// Every function here is a pure function of its inputs — no file I/O, no
// engine state — mirroring scene.go and material.go. The asset library
// (pack membership, per-asset materials) does not exist yet, so it is
// injected; this layer never guesses.
package asset

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"kaijuengine.com/matrix"
)

// Pack mixing errors. These are sentinel errors, use errors.Is to match
// them. All findings here are warning-grade: they flag scenes and assets
// for manual review, they never reject a scene outright (GDD CR-6 calls
// them warnings; the editor surfaces them at authoring time).
var (
	// ErrPackDominance is returned by ValidatePackMixing when one pack
	// owns more than the dominance threshold of a scene's visible surface
	// area (GDD CR-6, art bible 5.5 no-50/50-frames rule).
	ErrPackDominance = errors.New("asset: single-pack dominance exceeds threshold")
	// ErrColorTemperature is returned by ValidateColorTemperature when a
	// scene's dominant material palette opposes its area archetype's
	// temperature register (art bible 2.3).
	ErrColorTemperature = errors.New("asset: material palette opposes the area archetype")
	// ErrScaleIncoherent is returned by ValidateScaleCoherence when two
	// packs normalize the same nominal size (the door reference) to
	// heights that deviate beyond the limit (GDD CR-6 scale coherence).
	ErrScaleIncoherent = errors.New("asset: pack scale normalization deviates beyond the limit")
)

// Pack stratums (ADR-0004 two-pack integration): the pack-to-stratum
// mapping that makes the interleaving legible — Dark Fantasy is the
// ruin's bones, Enchanted is the world's skin. Both stratums pass through
// the same painterly grade.
const (
	// PackStratumBones marks the Dark Fantasy pack: Gothic construction,
	// empire remnants.
	PackStratumBones = "bones"
	// PackStratumSkin marks the Enchanted pack: green reclamation,
	// settled warmth.
	PackStratumSkin = "skin"
)

// Defaults for pack mixing validation. Override via PackMixingLimits —
// zero values fall back to these constants (same pattern as
// ValidationLimits).
const (
	// DefaultDominanceThreshold bounds a single pack's share of a scene's
	// visible surface area (GDD tuning knob color_temp_warn_threshold).
	// Above it the scene reads as single-pack, which the art bible's
	// ratio rule forbids for outdoor scenes (CR-6, art bible 5.5).
	DefaultDominanceThreshold = matrix.Float(0.8)
	// DefaultMaxScaleDeviation bounds how far two packs' normalization of
	// the same nominal size may drift (GDD CR-6: "within 10% of each
	// other after normalization").
	DefaultMaxScaleDeviation = matrix.Float(0.10)
)

// Pack is one KitBash3D pack definition in the two-pack registry
// (ADR-0004). The registry is the single source of truth for what a pack
// is: which stratum it plays, which material mapping table it feeds, and
// which imported assets belong to it.
type Pack struct {
	// ID is the pack identifier (kebab-case): "dark-fantasy" or
	// "enchanted".
	ID string
	// Stratum is the pack's stratum — PackStratumBones (Dark Fantasy) or
	// PackStratumSkin (Enchanted). The interleaving rule is that the two
	// stratums touch: the skin grows on the bones (art bible 5.5).
	Stratum string
	// MaterialMap is the pack's ASSET-002 material mapping table ID (the
	// table's Pack field). Both packs flow through the same import
	// pipeline and the same painterly grade; the mapping table is the
	// only pack-specific step (ADR-0004 per-pack treatment).
	MaterialMap string
	// Members lists the imported asset IDs (kebab-case, as placed in
	// scenes) that belong to this pack. The asset library populates this
	// from import manifests (ASSET-001) — until then it is empty and
	// pack mixing cannot classify placements.
	Members []string
}

// DefaultPacks returns the two canonical pack definitions (ADR-0004):
// dark-fantasy (bones) and enchanted (skin), each referencing its
// material mapping table. Members are empty — the asset library fills
// them from import manifests at runtime; tests and callers inject their
// own member sets.
//
//	packs := asset.DefaultPacks()
//	library.AttachMembers(packs) // future asset library populates Members
func DefaultPacks() []Pack {
	return []Pack{
		{ID: "dark-fantasy", Stratum: PackStratumBones, MaterialMap: "dark-fantasy"},
		{ID: "enchanted", Stratum: PackStratumSkin, MaterialMap: "enchanted"},
	}
}

// PackMixingLimits overrides the CR-6 thresholds. Zero values fall back
// to the Default* constants (GDD tuning knobs).
type PackMixingLimits struct {
	// DominanceThreshold bounds a single pack's share of a scene's
	// visible surface area, and the opposing palette's share of an
	// archetype's scene (default 0.8 — GDD knob color_temp_warn_threshold).
	DominanceThreshold matrix.Float
	// MaxScaleDeviation bounds cross-pack scale drift on the same
	// nominal size (default 0.10 — GDD CR-6 "within 10%").
	MaxScaleDeviation matrix.Float
}

// withDefaults fills zero-value limits with the project defaults.
func (l PackMixingLimits) withDefaults() PackMixingLimits {
	if l.DominanceThreshold <= 0 {
		l.DominanceThreshold = DefaultDominanceThreshold
	}
	if l.MaxScaleDeviation <= 0 {
		l.MaxScaleDeviation = DefaultMaxScaleDeviation
	}
	return l
}

// ValidatePackMixing checks a scene's placement mix against the CR-6
// single-pack dominance rule (art bible 5.5 no-50/50-frames). Each
// placement contributes its ground footprint (|scale.X * scale.Z|, world
// meters — the scene layer has no mesh geometry, so footprint is the
// visible-surface-area proxy) to its pack's share of the scene total.
// When a pack's share exceeds DominanceThreshold the validator returns
// ErrPackDominance naming the pack; a mix where every pack sits between
// 20% and 80% passes. A 50/50 tie passes the mechanical check — the art
// rule itself (balance reads as a collision, reserved for State 4)
// remains a human authorship rule the warning leans toward.
//
// Pack membership is injected because it lives in the asset library
// (which does not exist yet): each pack's Members list the imported
// asset IDs it owns. Placements whose asset_ref matches no pack count
// toward the scene total but toward no pack — unclassified area can
// dilute a warning but is never credited to one side.
//
//	packs := asset.DefaultPacks()
//	packs[0].Members = darkFantasyAssetIDs
//	packs[1].Members = enchantedAssetIDs
//	if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); err != nil {
//		slog.Warn(err.Error()) // warning-grade: flag for review, never block the scene
//	}
func ValidatePackMixing(scene Scene, packs []Pack, limits PackMixingLimits) error {
	lim := limits.withDefaults()

	packOf := make(map[string]*Pack, len(packs))
	for i := range packs {
		for _, member := range packs[i].Members {
			packOf[member] = &packs[i]
		}
	}

	area := make(map[string]matrix.Float, len(packs))
	var total matrix.Float
	for _, pl := range scene.Placements {
		footprint := footprintOf(pl)
		if footprint <= 0 {
			continue
		}
		total += footprint
		if p := packOf[pl.AssetRef]; p != nil {
			area[p.ID] += footprint
		}
	}
	if total <= 0 {
		return nil // nothing visible to judge
	}

	// At most one pack can exceed the threshold (shares sum to <= 1 with
	// unclassified area), so the first hit in registry order is the only
	// hit.
	for i := range packs {
		share := area[packs[i].ID] / total
		if share > lim.DominanceThreshold {
			return fmt.Errorf(
				"%w: pack %q owns %.0f%% of the scene's visible surface area, exceeding the %.0f%% "+
					"threshold (GDD CR-6, art bible 5.5: every frame needs a dominant stratum — "+
					"lean the mix one way; a tie reads as a collision, reserved for State 4)",
				ErrPackDominance, packs[i].ID, share*100, lim.DominanceThreshold*100)
		}
	}
	return nil
}

// footprintOf is the visible-surface-area proxy for one placement: the
// absolute ground footprint in world meters (|scale.X * scale.Z|, the
// isometric ground plane). The scene layer carries no mesh geometry, so
// footprint stands in for surface area (GDD CR-6 "visible surface area").
func footprintOf(pl Placement) matrix.Float {
	return matrix.Float(math.Abs(float64(pl.Scale.X() * pl.Scale.Z())))
}

// TemperatureRegister is an area archetype's position on the art bible
// 2.3 warm-to-cold gradient, and a material palette family's register.
type TemperatureRegister int

const (
	// TempWarm marks warm registers: amber, moss, oxblood families, and
	// the warm areas (borderland settlements, river towns).
	TempWarm TemperatureRegister = iota
	// TempCold marks cold registers: stone, indigo, ivory families, and
	// the cold areas (deep ruins, high places).
	TempCold
)

// String names the register for validation messages.
func (r TemperatureRegister) String() string {
	switch r {
	case TempWarm:
		return "warm"
	case TempCold:
		return "cold"
	default:
		return "unknown"
	}
}

// AreaArchetype is one art bible 2.3 area archetype (the temperature
// family of a region, e.g. Warm Borderland). Scenes declare their
// archetype via AmbientPreset; the register is what palette validation
// checks against.
type AreaArchetype struct {
	// ID is the archetype identifier (kebab-case, art bible 2.3), e.g.
	// "warm-borderland" or "deep-ruins".
	ID string
	// Register is the archetype's temperature register: TempWarm (amber
	// owns the frame) or TempCold (pale fraction rises).
	Register TemperatureRegister
}

// ArchetypeWarmBorderland is the borderland settlements archetype (art
// bible 2.3): the warmest register — golden day, amber dusk, ochre stone,
// garden moss.
//
//	archetype := asset.ArchetypeWarmBorderland()
func ArchetypeWarmBorderland() AreaArchetype {
	return AreaArchetype{ID: "warm-borderland", Register: TempWarm}
}

// ArchetypeDeepRuins is the empire strata archetype (art bible 2.3): the
// marrow zone — pale fraction rises with depth, saturation drains toward
// the achromatic.
//
//	archetype := asset.ArchetypeDeepRuins()
func ArchetypeDeepRuins() AreaArchetype {
	return AreaArchetype{ID: "deep-ruins", Register: TempCold}
}

// paletteFamilyRegisters maps the six art bible color families to their
// temperature registers (art bible 2.1): amber, moss, and oxblood are
// warm life-color; stone, indigo, and ivory are cold — the empire's
// marble is bleached of life-color.
var paletteFamilyRegisters = map[string]TemperatureRegister{
	"amber":   TempWarm,
	"moss":    TempWarm,
	"oxblood": TempWarm,
	"stone":   TempCold,
	"indigo":  TempCold,
	"ivory":   TempCold,
}

// familyRegister returns the temperature register of an art bible color
// family. Unknown families (including empty) are not judged.
func familyRegister(family string) (TemperatureRegister, bool) {
	r, ok := paletteFamilyRegisters[family]
	return r, ok
}

// ValidateColorTemperature checks a scene's dominant material palette
// against its area archetype (GDD CR-6 color temperature consistency, art
// bible 2.3). Each placement's register is the majority register of its
// mapped materials (the art bible families), weighted by ground footprint
// (the same visible-surface-area proxy as ValidatePackMixing). When the
// opposing register's share of the scene total exceeds
// DominanceThreshold, the validator returns ErrColorTemperature — a Warm
// Borderland scene that is 80% Dark Fantasy grey stone fails the
// archetype. Placements with no material data and tied placements count
// toward the total but toward no register; unknown families are not
// judged.
//
// The per-asset material data is injected because it lives in the asset
// library (mapped through the pack's ASSET-002 table); until then the
// caller supplies what it knows.
//
//	colors := map[string][]asset.MaterialEntry{
//	    "stone-pillar": {{Project: "stone-grey", Palette: "stone"}},
//	}
//	if err := asset.ValidateColorTemperature(scene, archetype, colors, asset.PackMixingLimits{}); err != nil {
//		slog.Warn(err.Error()) // warning-grade: palette fights the archetype
//	}
func ValidateColorTemperature(scene Scene, archetype AreaArchetype,
	materialsByAsset map[string][]MaterialEntry, limits PackMixingLimits) error {
	lim := limits.withDefaults()

	var warmArea, coldArea, total matrix.Float
	for _, pl := range scene.Placements {
		footprint := footprintOf(pl)
		if footprint <= 0 {
			continue
		}
		total += footprint
		mats := materialsByAsset[pl.AssetRef]
		var warm, cold int
		for _, m := range mats {
			if r, ok := familyRegister(m.Palette); ok {
				if r == TempWarm {
					warm++
				} else {
					cold++
				}
			}
		}
		switch {
		case warm > cold:
			warmArea += footprint
		case cold > warm:
			coldArea += footprint
		}
	}
	if total <= 0 {
		return nil
	}

	var opposing string
	var opposingArea matrix.Float
	switch archetype.Register {
	case TempWarm:
		opposing, opposingArea = TempCold.String(), coldArea
	case TempCold:
		opposing, opposingArea = TempWarm.String(), warmArea
	default:
		return nil // unknown archetype register: nothing to check against
	}

	share := opposingArea / total
	if share > lim.DominanceThreshold {
		return fmt.Errorf(
			"%w: %s scene has %.0f%% of visible surface area in opposing %s-family materials, "+
				"exceeding the %.0f%% threshold (GDD CR-6 color temperature consistency, art bible 2.3: "+
				"the dominant palette must match the area register — a Warm Borderland is not grey stone)",
			ErrColorTemperature, archetype.ID, share*100, opposing, lim.DominanceThreshold*100)
	}
	return nil
}

// ValidateScaleCoherence checks that packs normalize the same nominal
// size to within MaxScaleDeviation of each other (GDD CR-6 scale
// coherence: assets at the same nominal size, e.g. "door", must be within
// 10% of each other after normalization). Inputs are each pack's
// normalized door height in world meters — the ASSET-001 import output
// (DoorHeightSource * ScaleFactor). Deviation is |a - b| / min(a, b);
// packs with no measurement (zero or negative heights) are skipped.
//
//	heights := map[string]matrix.Float{
//	    "dark-fantasy": resultDF.DoorHeightSource * resultDF.ScaleFactor,
//	    "enchanted":    resultEN.DoorHeightSource * resultEN.ScaleFactor,
//	}
//	if err := asset.ValidateScaleCoherence(heights, asset.PackMixingLimits{}); err != nil {
//		return err // one pack's scale reference is off; re-baseline it
//	}
func ValidateScaleCoherence(doorHeights map[string]matrix.Float, limits PackMixingLimits) error {
	lim := limits.withDefaults()

	ids := make([]string, 0, len(doorHeights))
	for id := range doorHeights {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic report order
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := doorHeights[ids[i]], doorHeights[ids[j]]
			if a <= 0 || b <= 0 {
				continue // no usable measurement for this pair
			}
			minH := a
			if b < minH {
				minH = b
			}
			deviation := matrix.Float(math.Abs(float64(a-b))) / minH
			if deviation > lim.MaxScaleDeviation {
				return fmt.Errorf(
					"%w: packs %q and %q normalize the same nominal size (the %q door reference) "+
						"to %.2fm and %.2fm, a %.0f%% deviation exceeding the %.0f%% limit; "+
						"re-baseline one pack's scale reference (GDD CR-6 scale coherence)",
					ErrScaleIncoherent, ids[i], ids[j], DefaultDoorReferenceName,
					a, b, deviation*100, lim.MaxScaleDeviation*100)
			}
		}
	}
	return nil
}

// LegendVerdict is the legend filter's assessment of one asset (art
// bible 1.1, 5.5, 5.8): does it read as myth, or as the machine the myth
// is supposed to hide?
type LegendVerdict int

const (
	// LegendAccept imports the asset as-is: it passes the legend filter.
	LegendAccept LegendVerdict = iota
	// LegendTreat imports the asset with a treatment flag: it reads
	// generic-fantasy or modern-engineering and must be redesigned as
	// ornament — overgrowth, re-paint, modification (art bible 5.5).
	LegendTreat
	// LegendReject blocks the asset at import: its vocabulary cannot be
	// treated into myth, so the source must be replaced.
	LegendReject
)

// String names the verdict for reports.
func (v LegendVerdict) String() string {
	switch v {
	case LegendAccept:
		return "accept"
	case LegendTreat:
		return "treat"
	case LegendReject:
		return "reject"
	default:
		return "unknown"
	}
}

// legendTreatTokens is the modern-engineering vocabulary that can be
// redesigned as ornament: machinery the borderland can overgrow,
// repaint, or hide (art bible 5.2: no standardized panels, no cables, no
// safety geometry — but the craftsman can make them myth).
var legendTreatTokens = map[string]bool{
	"cable": true, "panel": true, "vent": true, "duct": true,
	"conduit": true, "rack": true, "grille": true, "gauge": true,
}

// legendRejectTokens is the vocabulary that cannot read as myth in any
// treatment: the borderland has no word that turns a plasma bolt into
// "the breath of the old world" (art bible 7.2: the old god waking,
// never a boot sequence). These assets are replaced, not treated.
var legendRejectTokens = map[string]bool{
	"plasma": true, "hologram": true, "laser": true, "monitor": true,
	"screen": true, "drone": true, "circuit": true, "antenna": true,
	"servo": true, "reactor": true,
}

// AssessLegendFilter runs the legend filter's mechanical first pass over
// one asset (art bible 5.8: the palette validator's out-of-family warning
// is the filter's first pass; the human pass is final). Verdicts, in
// priority order:
//
//   - LegendReject — the source name contains irredeemably modern
//     vocabulary (plasma, hologram, laser, monitor, screen, drone,
//     circuit, antenna, servo, reactor): it cannot be treated into myth.
//   - LegendTreat — the name contains treatable engineering vocabulary
//     (cable, panel, vent, duct, conduit, rack, grille, gauge), or any
//     mapped material's palette is outside the art bible families
//     (re-paint): import with a treatment flag.
//   - LegendAccept — otherwise: passes as-is.
//
// The verdict names the outcome only; the authoring team applies the
// treatment (overgrowth, re-paint, modification) or replaces the asset.
//
//	verdict := asset.AssessLegendFilter("Cable_Rack_01", nil) // LegendTreat
//	verdict := asset.AssessLegendFilter("Door_DarkWood", nil) // LegendAccept
func AssessLegendFilter(name string, materials []MaterialEntry) LegendVerdict {
	lower := strings.ToLower(name)

	// Reject beats treat: an irredeemable token overrides any treatable
	// one in the same name.
	for token := range legendRejectTokens {
		if strings.Contains(lower, token) {
			return LegendReject
		}
	}
	for token := range legendTreatTokens {
		if strings.Contains(lower, token) {
			return LegendTreat
		}
	}
	for _, m := range materials {
		if m.Palette != "" && !IsPaletteFamily(m.Palette) {
			return LegendTreat // re-paint: the filter's mechanical first pass (art bible 5.8)
		}
	}
	return LegendAccept
}

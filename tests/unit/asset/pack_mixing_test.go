// Unit tests for Story ASSET-004: Two-pack integration rules
// (TR-asset-005, ADR-0004, GDD CR-6). ValidatePackMixing,
// ValidateColorTemperature, ValidateScaleCoherence, and
// AssessLegendFilter are pure functions of their inputs — no file I/O,
// no engine state — so every rule is exercised with struct literals,
// deterministically. Test naming: Test[Feature]_[Scenario]_[Expected].
package asset_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/asset"
)

// assetRefs builds n kebab-case asset IDs under one prefix, e.g.
// assetRefs("df-pillar", 3) -> ["df-pillar-00", "df-pillar-01",
// "df-pillar-02"]. Used to populate pack Members and scene placements.
func assetRefs(prefix string, n int) []string {
	refs := make([]string, n)
	for i := range refs {
		refs[i] = fmt.Sprintf("%s-%02d", prefix, i)
	}
	return refs
}

// placementAt builds one placement with a ground footprint of f world
// meters (scale.X = f, scale.Z = 1; the footprint proxy is
// |scale.X * scale.Z|, so the area is exactly f).
func placementAt(id, assetRef string, footprint matrix.Float) asset.Placement {
	return asset.Placement{ID: id, AssetRef: assetRef, Scale: matrix.NewVec3(footprint, 1, 1)}
}

// placementsWithRefs builds one unit-footprint placement per ref, in
// order.
func placementsWithRefs(refs ...string) []asset.Placement {
	pls := make([]asset.Placement, 0, len(refs))
	for i, ref := range refs {
		pls = append(pls, placementAt(fmt.Sprintf("p-%02d", i), ref, 1))
	}
	return pls
}

// mixingFixture builds the canonical two packs (dark-fantasy, enchanted)
// with the given members and a scene placing every ref once with
// footprint 1.
func mixingFixture(dfRefs, enRefs []string) ([]asset.Pack, asset.Scene) {
	packs := []asset.Pack{
		{ID: "dark-fantasy", Stratum: asset.PackStratumBones, MaterialMap: "dark-fantasy", Members: dfRefs},
		{ID: "enchanted", Stratum: asset.PackStratumSkin, MaterialMap: "enchanted", Members: enRefs},
	}
	refs := append(append([]string{}, dfRefs...), enRefs...)
	return packs, asset.Scene{Placements: placementsWithRefs(refs...)}
}

// TestPackMixing_SinglePackDominant_DominanceWarning (acceptance
// criterion 1): a scene whose visible surface area is >80% one pack is
// flagged with a dominance warning naming the pack — the mechanical ally
// of the art bible's no-50/50-frames rule (art bible 5.5, GDD CR-6).
func TestPackMixing_SinglePackDominant_DominanceWarning(t *testing.T) {
	t.Run("90% dark fantasy warns", func(t *testing.T) {
		packs, scene := mixingFixture(assetRefs("df-pillar", 9), assetRefs("en-oak", 1))
		err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{})
		if !errors.Is(err, asset.ErrPackDominance) {
			t.Fatalf("ValidatePackMixing = %v, want ErrPackDominance", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, "dark-fantasy") {
			t.Errorf("error %q does not name the dominant pack", msg)
		}
		if !strings.Contains(msg, "90%") {
			t.Errorf("error %q does not state the 90%% share", msg)
		}
	})

	t.Run("100% dark fantasy warns", func(t *testing.T) {
		packs, scene := mixingFixture(assetRefs("df-pillar", 10), nil)
		if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); !errors.Is(err, asset.ErrPackDominance) {
			t.Fatalf("ValidatePackMixing = %v, want ErrPackDominance", err)
		}
	})

	t.Run("unclassified area dilutes but never credits", func(t *testing.T) {
		// 9 df + 1 en is a warning on its own; adding 10 unclassified
		// footprints dilutes the share to 45% (never credited to a pack).
		packs, scene := mixingFixture(assetRefs("df-pillar", 9), assetRefs("en-oak", 1))
		scene.Placements = append(scene.Placements, placementsWithRefs(assetRefs("unknown-prop", 10)...)...)
		if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("diluted scene = %v, want nil (unclassified area is not credited)", err)
		}
	})
}

// TestPackMixing_BalancedMixing_Passes (acceptance criterion 2): a scene
// where each pack holds 20-80% of the visible surface area passes
// without a warning — including a 50/50 tie, which the mechanical check
// deliberately leaves to the human authorship rule (a tie reads as a
// collision, reserved for State 4; the dominance warning leans authors
// away from it, art bible 5.5).
func TestPackMixing_BalancedMixing_Passes(t *testing.T) {
	t.Run("50/50 split passes", func(t *testing.T) {
		packs, scene := mixingFixture(assetRefs("df-pillar", 5), assetRefs("en-oak", 5))
		if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("50/50 scene = %v, want nil", err)
		}
	})

	t.Run("70/30 lean passes", func(t *testing.T) {
		packs, scene := mixingFixture(assetRefs("df-pillar", 7), assetRefs("en-oak", 3))
		if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("70/30 scene = %v, want nil", err)
		}
	})

	t.Run("exactly 80% boundary passes", func(t *testing.T) {
		packs, scene := mixingFixture(assetRefs("df-pillar", 8), assetRefs("en-oak", 2))
		if err := asset.ValidatePackMixing(scene, packs, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("80/20 boundary scene = %v, want nil (warning is strictly greater than 80%%)", err)
		}
	})

	t.Run("empty scene passes", func(t *testing.T) {
		packs, _ := mixingFixture(nil, nil)
		if err := asset.ValidatePackMixing(asset.Scene{}, packs, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("empty scene = %v, want nil", err)
		}
	})
}

// TestScaleCoherence_OverDeviation_Flagged (acceptance criterion 3):
// both packs' normalization of the same nominal size (the door
// reference) must land within 10% of each other (GDD CR-6). The inputs
// are the ASSET-001 import outputs: measured door height in world
// meters.
func TestScaleCoherence_OverDeviation_Flagged(t *testing.T) {
	t.Run("15% deviation flags", func(t *testing.T) {
		heights := map[string]matrix.Float{
			"dark-fantasy": 2.0,
			"enchanted":    2.3,
		}
		err := asset.ValidateScaleCoherence(heights, asset.PackMixingLimits{})
		if !errors.Is(err, asset.ErrScaleIncoherent) {
			t.Fatalf("ValidateScaleCoherence = %v, want ErrScaleIncoherent", err)
		}
		msg := err.Error()
		for _, want := range []string{"dark-fantasy", "enchanted", "15%"} {
			if !strings.Contains(msg, want) {
				t.Errorf("error %q does not mention %q", msg, want)
			}
		}
	})

	t.Run("5% deviation passes", func(t *testing.T) {
		heights := map[string]matrix.Float{
			"dark-fantasy": 2.0,
			"enchanted":    2.1,
		}
		if err := asset.ValidateScaleCoherence(heights, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("5%% deviation = %v, want nil", err)
		}
	})

	t.Run("one pack has no usable measurement", func(t *testing.T) {
		heights := map[string]matrix.Float{
			"dark-fantasy": 2.0,
			"enchanted":    0, // no door reference measured: cannot judge
		}
		if err := asset.ValidateScaleCoherence(heights, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("unmeasured pack = %v, want nil", err)
		}
		if err := asset.ValidateScaleCoherence(map[string]matrix.Float{"dark-fantasy": 2.0}, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("single pack = %v, want nil", err)
		}
		if err := asset.ValidateScaleCoherence(nil, asset.PackMixingLimits{}); err != nil {
			t.Fatalf("no packs = %v, want nil", err)
		}
	})
}

// materialsFor maps every warm-ref to a warm-family material entry and
// every cold-ref to a cold-family entry, as the asset library would
// supply after ASSET-002 material mapping.
func materialsFor(warmRefs, coldRefs []string) map[string][]asset.MaterialEntry {
	m := make(map[string][]asset.MaterialEntry, len(warmRefs)+len(coldRefs))
	for _, r := range warmRefs {
		m[r] = []asset.MaterialEntry{{Source: r, Project: "moss-cover", Palette: "moss"}}
	}
	for _, r := range coldRefs {
		m[r] = []asset.MaterialEntry{{Source: r, Project: "stone-grey", Palette: "stone"}}
	}
	return m
}

// TestColorTemperature_WarmArchetypeStoneDominant_Warns (acceptance
// criterion 4): a Warm Borderland scene whose dominant material palette
// is >80% cold family (Dark Fantasy grey stone) fails the archetype with
// a color temperature warning (GDD CR-6, art bible 2.3). The inverse —
// warm-dominant in a cold archetype — warns symmetrically.
func TestColorTemperature_WarmArchetypeStoneDominant_Warns(t *testing.T) {
	t.Run("90% grey stone in warm borderland warns", func(t *testing.T) {
		stoneRefs, mossRefs := assetRefs("stone-pillar", 9), assetRefs("enchanted-oak", 1)
		_, scene := mixingFixture(stoneRefs, mossRefs)
		err := asset.ValidateColorTemperature(scene, asset.ArchetypeWarmBorderland(),
			materialsFor(mossRefs, stoneRefs), asset.PackMixingLimits{})
		if !errors.Is(err, asset.ErrColorTemperature) {
			t.Fatalf("ValidateColorTemperature = %v, want ErrColorTemperature", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, "warm-borderland") {
			t.Errorf("error %q does not name the archetype", msg)
		}
		if !strings.Contains(msg, "90%") {
			t.Errorf("error %q does not state the 90%% opposing share", msg)
		}
	})

	t.Run("30% grey stone in warm borderland passes", func(t *testing.T) {
		stoneRefs, mossRefs := assetRefs("stone-pillar", 3), assetRefs("enchanted-oak", 7)
		_, scene := mixingFixture(stoneRefs, mossRefs)
		if err := asset.ValidateColorTemperature(scene, asset.ArchetypeWarmBorderland(),
			materialsFor(mossRefs, stoneRefs), asset.PackMixingLimits{}); err != nil {
			t.Fatalf("balanced palette = %v, want nil", err)
		}
	})

	t.Run("warm-dominant in deep ruins warns symmetrically", func(t *testing.T) {
		stoneRefs, mossRefs := assetRefs("stone-pillar", 1), assetRefs("enchanted-oak", 9)
		_, scene := mixingFixture(stoneRefs, mossRefs)
		err := asset.ValidateColorTemperature(scene, asset.ArchetypeDeepRuins(),
			materialsFor(mossRefs, stoneRefs), asset.PackMixingLimits{})
		if !errors.Is(err, asset.ErrColorTemperature) {
			t.Fatalf("ValidateColorTemperature = %v, want ErrColorTemperature", err)
		}
		if !strings.Contains(err.Error(), "deep-ruins") {
			t.Errorf("error %q does not name the archetype", err.Error())
		}
	})
}

// TestLegendFilter_EngineeringVocabulary_TreatOrReject: the legend
// filter's mechanical first pass (art bible 5.8) — modern-engineering
// vocabulary that can be redesigned as ornament is treated, vocabulary
// that cannot read as myth is rejected, and an out-of-family material
// palette is treated (re-paint). Assets that pass are accepted as-is.
func TestLegendFilter_EngineeringVocabulary_TreatOrReject(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		materials []asset.MaterialEntry
		want      asset.LegendVerdict
	}{
		{"treatable engineering vocabulary is treated", "Cable_Rack_01", nil, asset.LegendTreat},
		{"irredeemable vocabulary is rejected", "Plasma_Cannon", nil, asset.LegendReject},
		{"reject beats treat in one name", "Plasma_Cable_Housing", nil, asset.LegendReject},
		{"clean dark fantasy asset passes as-is", "Door_DarkWood", nil, asset.LegendAccept},
		{"out-of-family palette is treated (re-paint)", "Door_DarkWood",
			[]asset.MaterialEntry{{Source: "Door_DarkWood", Project: "neon-door", Palette: "neon"}}, asset.LegendTreat},
		{"in-family palette does not trigger a treat", "Stone_Wall",
			[]asset.MaterialEntry{{Source: "Stone_Wall", Project: "stone-grey", Palette: "stone"}}, asset.LegendAccept},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := asset.AssessLegendFilter(tc.source, tc.materials)
			if got != tc.want {
				t.Errorf("AssessLegendFilter(%q) = %v, want %v", tc.source, got, tc.want)
			}
		})
	}
}

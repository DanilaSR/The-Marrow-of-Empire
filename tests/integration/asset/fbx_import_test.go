// Integration tests for ASSET-001: FBX Import & Normalization
// (ADR-0004, TR-asset-001).
//
// These tests run the real pipeline (src/asset) against small fixture files
// in testdata/asset/: two OBJ packs at different source scales, one minimal
// binary FBX with an embedded non-power-of-two TGA texture, and standalone
// PNG/TGA textures. The engine loaders parse the fixtures for real, so the
// tests cover load -> normalize -> serialize -> round-trip.
//
// Assertions focus on the ADR-0004 normalization contract:
//   - 1 source unit = 1 meter, validated against the door reference (2 m)
//   - Z-up sources are rotated to Kaiju's Y-up
//   - source stays read-only; everything is written under staging
//   - non-power-of-two textures are padded to PoT with a warning
package asset_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"marrow-of-empire/asset"

	"kaijuengine.com/rendering/loaders/kaiju_mesh"
)

// importPack runs the pipeline over one fixture pack directory. opts may
// override the config. Staging is always a fresh temp dir.
func importPack(t *testing.T, packDir string, opts ...func(*asset.Config)) (asset.Result, string) {
	t.Helper()
	generateFixtures(t)
	cfg := asset.Config{
		SourceDir:  filepath.Join(fixtureDir, packDir),
		StagingDir: filepath.Join(t.TempDir(), "staging"),
	}
	for _, o := range opts {
		o(&cfg)
	}
	res, err := asset.ImportAsset(cfg)
	if err != nil {
		t.Fatalf("ImportAsset(%s): %v", packDir, err)
	}
	return res, cfg.StagingDir
}

func meshByKey(t *testing.T, res asset.Result, key string) asset.MeshResult {
	t.Helper()
	for _, m := range res.Meshes {
		if m.Key == key {
			return m
		}
	}
	keys := make([]string, 0, len(res.Meshes))
	for _, m := range res.Meshes {
		keys = append(keys, m.Key)
	}
	t.Fatalf("mesh %q not found in result (keys: %v)", key, keys)
	return asset.MeshResult{}
}

// glbBounds reads a serialized GLB mesh back through the engine deserializer
// and returns its axis-aligned bounds and vertex count.
func glbBounds(t *testing.T, outFile string) (minX, minY, minZ, maxX, maxY, maxZ float64, verts int) {
	t.Helper()
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", outFile, err)
	}
	km, err := kaiju_mesh.Deserialize(data)
	if err != nil {
		t.Fatalf("kaiju_mesh.Deserialize(%s): %v", outFile, err)
	}
	if len(km.Verts) == 0 {
		t.Fatalf("mesh %s round-tripped with no vertices", outFile)
	}
	p := km.Verts[0].Position
	minX, minY, minZ = float64(p.X()), float64(p.Y()), float64(p.Z())
	maxX, maxY, maxZ = minX, minY, minZ
	for _, v := range km.Verts {
		x, y, z := float64(v.Position.X()), float64(v.Position.Y()), float64(v.Position.Z())
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		minZ, maxZ = math.Min(minZ, z), math.Max(maxZ, z)
	}
	return minX, minY, minZ, maxX, maxY, maxZ, len(km.Verts)
}

// TestImportScaleNormalizationAcrossPacks is the QA acceptance case: two
// packs authored at wildly different source scales (door 200 units vs 1
// unit) must both normalize to a 2-meter door, and the scale factor must
// differ accordingly. The 10% tolerance is the story's acceptance bound.
func TestImportScaleNormalizationAcrossPacks(t *testing.T) {
	packs := []struct {
		name          string
		pack          string
		wantFactor    float64
		wantDoorUnits float64
	}{
		{name: "door 200u", pack: "obj_door_200u", wantFactor: 0.01, wantDoorUnits: 200},
		{name: "door 1u", pack: "obj_door_1u", wantFactor: 2.0, wantDoorUnits: 1},
	}
	for _, p := range packs {
		t.Run(p.name, func(t *testing.T) {
			res, _ := importPack(t, p.pack)
			if got := float64(res.ScaleFactor); math.Abs(got-p.wantFactor) > p.wantFactor*0.001 {
				t.Errorf("ScaleFactor = %v, want ~%v", got, p.wantFactor)
			}
			if got := float64(res.DoorHeightSource); math.Abs(got-p.wantDoorUnits) > 0.001 {
				t.Errorf("DoorHeightSource = %v, want ~%v", got, p.wantDoorUnits)
			}
			door := meshByKey(t, res, "door")
			_, _, _, _, maxY, minY, _ := glbBounds(t, door.OutputFile)
			height := maxY - minY
			if math.Abs(height-2.0) > 2.0*0.10 {
				t.Errorf("door height = %v m, want 2.0 m within 10%%", height)
			}
			// Uniform scale: the crate must land at exactly 50% of the door.
			crate := meshByKey(t, res, "crate")
			_, _, _, _, cMaxY, cMinY, _ := glbBounds(t, crate.OutputFile)
			if got := cMaxY - cMinY; math.Abs(got-height/2) > 0.001 {
				t.Errorf("crate height = %v, want half the door (%v)", got, height/2)
			}
		})
	}
}

// TestImportRotatesZUpToYUp verifies the Z-up -> Y-up rotation for OBJ
// sources: the door's height (source Z) must end up on Kaiju's Y, and the
// top-face normal must point along +Y. Mirrors the engine FBX basis
// converter: (x, y, z) -> (-x, z, -y).
func TestImportRotatesZUpToYUp(t *testing.T) {
	res, _ := importPack(t, "obj_door_200u")
	door := meshByKey(t, res, "door")

	minX, minY, minZ, maxX, maxY, maxZ, _ := glbBounds(t, door.OutputFile)
	if math.Abs(minY) > 0.001 {
		t.Errorf("door base minY = %v, want 0", minY)
	}
	if height := maxY - minY; math.Abs(height-2.0) > 0.001 {
		t.Errorf("door height on Y = %v, want 2.0", height)
	}
	// The old Z extent (door height) must not remain on Z; only the old
	// width (1 source unit) is on Z after rotation.
	if zExtent := maxZ - minZ; zExtent > 0.02 {
		t.Errorf("door Z extent = %v, want <= 0.02 (old width only)", zExtent)
	}
	// Old width on X, negative after (-x) rotation.
	if xExtent := maxX - minX; math.Abs(xExtent-0.01) > 0.001 {
		t.Errorf("door X extent = %v, want ~0.01", xExtent)
	}

	// The top face normal (0,0,1) in source space must be (0,1,0) in Kaiju
	// space, attached to the top vertices at y = 2.
	data, err := os.ReadFile(door.OutputFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", door.OutputFile, err)
	}
	km, err := kaiju_mesh.Deserialize(data)
	if err != nil {
		t.Fatalf("kaiju_mesh.Deserialize: %v", err)
	}
	found := false
	for _, v := range km.Verts {
		if math.Abs(float64(v.Position.Y())-2.0) < 0.001 {
			nx, ny := float64(v.Normal.X()), float64(v.Normal.Y())
			if math.Abs(nx) < 0.001 && math.Abs(ny-1.0) < 0.001 {
				found = true
			}
		}
	}
	if !found {
		t.Error("no top vertex at y=2 with normal (0,1,0); rotation did not convert normals")
	}
}

// TestImportFbxDoorWithEmbeddedTga runs the pipeline over a minimal binary
// FBX: Z-up door (200 units) with an embedded 300x128 TGA base-color
// texture. It must normalize to a 2 m door, record the texture key in the
// manifest, and pad the embedded TGA to 512x128 PNG in staging.
func TestImportFbxDoorWithEmbeddedTga(t *testing.T) {
	res, _ := importPack(t, "fbx")

	if got := float64(res.ScaleFactor); math.Abs(got-0.01) > 0.0001 {
		t.Errorf("ScaleFactor = %v, want ~0.01", got)
	}
	door := meshByKey(t, res, "door")
	_, _, _, _, maxY, minY, _ := glbBounds(t, door.OutputFile)
	if height := maxY - minY; math.Abs(height-2.0) > 0.001 {
		t.Errorf("door height = %v, want 2.0", height)
	}

	// The texture key (recorded for ASSET-002 material mapping) must be the
	// embedded video slot, not a source path.
	if got := door.Textures["baseColor"]; got != "embedded_10004_baseColor" {
		t.Errorf("Textures[baseColor] = %q, want embedded_10004_baseColor", got)
	}

	// Padding must have produced a 512x128 PNG in staging.
	if len(door.PaddedTextures) != 1 {
		t.Fatalf("PaddedTextures = %d entries, want 1", len(door.PaddedTextures))
	}
	pad := door.PaddedTextures[0]
	if pad.OriginalWidth != 300 || pad.OriginalHeight != 128 {
		t.Errorf("padded original = %dx%d, want 300x128", pad.OriginalWidth, pad.OriginalHeight)
	}
	if pad.Width != 512 || pad.Height != 128 {
		t.Errorf("padded size = %dx%d, want 512x128", pad.Width, pad.Height)
	}
	data, err := os.ReadFile(pad.OutputFile)
	if err != nil {
		t.Fatalf("padded texture %s not written: %v", pad.OutputFile, err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("padded texture is not a decodable PNG: %v", err)
	}
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != 512 || h != 128 {
		t.Errorf("padded PNG size = %dx%d, want 512x128", w, h)
	}

	// The import warns about the padding.
	warned := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "padded") && strings.Contains(w, "300x128 -> 512x128") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("expected a padding warning, got warnings: %v", res.Warnings)
	}

	// The manifest records source, scale factor and the texture key.
	manifestPath := filepath.Join(filepath.Dir(door.OutputFile), "manifest.json")
	mdata, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
	var man asset.Manifest
	if err := json.Unmarshal(mdata, &man); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if man.Source != "door.fbx" {
		t.Errorf("manifest source = %q, want door.fbx", man.Source)
	}
	if got := float64(man.ScaleFactor); math.Abs(got-0.01) > 0.0001 {
		t.Errorf("manifest scaleFactor = %v, want ~0.01", got)
	}
	if len(man.Meshes) != 1 || man.Meshes[0].Textures["baseColor"] != "embedded_10004_baseColor" {
		t.Errorf("manifest meshes = %+v, want door with baseColor texture", man.Meshes)
	}
}

// TestImportRejectsMissingDoorReference: without a door reference the
// pipeline must refuse to guess scale.
func TestImportRejectsMissingDoorReference(t *testing.T) {
	src := t.TempDir()
	writeOBJ(t, src, "crate.obj", `o Crate
v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
v 0 0 1
v 1 0 1
v 1 1 1
v 0 1 1
vt 0 0
vn 0 0 1
f 1/1/1 2/1/1 3/1/1
f 1/1/1 3/1/1 4/1/1
f 5/1/1 6/1/1 7/1/1
f 5/1/1 7/1/1 8/1/1
f 1/1/1 2/1/1 6/1/1
f 1/1/1 6/1/1 5/1/1
f 4/1/1 3/1/1 7/1/1
f 4/1/1 7/1/1 8/1/1
f 1/1/1 5/1/1 8/1/1
f 1/1/1 8/1/1 4/1/1
f 2/1/1 3/1/1 7/1/1
f 2/1/1 7/1/1 6/1/1
`)
	_, err := asset.ImportAsset(asset.Config{
		SourceDir:  src,
		StagingDir: filepath.Join(t.TempDir(), "staging"),
	})
	if !errors.Is(err, asset.ErrDoorReferenceNotFound) {
		t.Fatalf("err = %v, want ErrDoorReferenceNotFound", err)
	}
}

// TestImportRejectsNotYUp: a source that is not Y-up after normalization
// must be rejected, not silently re-rotated (ADR-0004: never guess).
func TestImportRejectsNotYUp(t *testing.T) {
	// Treat the Z-up fixture as if it were already Y-up: the door spans
	// 200 units on Z and only 0.5 on Y.
	_, err := func() (asset.Result, error) {
		generateFixtures(t)
		return asset.ImportAsset(asset.Config{
			SourceDir:   filepath.Join(fixtureDir, "obj_door_200u"),
			StagingDir:  filepath.Join(t.TempDir(), "staging"),
			SourceUpAxis: "Y",
		})
	}()
	if !errors.Is(err, asset.ErrNotYUp) {
		t.Fatalf("err = %v, want ErrNotYUp", err)
	}
}

// TestImportRejectsScaleOutOfRange: a door-derived scale factor outside the
// sanity range must reject the import (ADR-0004: never guess scale).
func TestImportRejectsScaleOutOfRange(t *testing.T) {
	src := t.TempDir()
	// Door 5000 units tall: factor would be 2/5000 = 0.0004, below the
	// 0.001 minimum.
	writeOBJ(t, src, "huge_door.obj", zUpDoorOBJ(5000, 1, 0))
	_, err := asset.ImportAsset(asset.Config{
		SourceDir:  src,
		StagingDir: filepath.Join(t.TempDir(), "staging"),
	})
	if !errors.Is(err, asset.ErrScaleOutOfRange) {
		t.Fatalf("err = %v, want ErrScaleOutOfRange", err)
	}
}

// TestImportSourceReadOnlyAndBlendWarning: BLEND files warn (export to FBX
// first) and the source directory is never written to.
func TestImportSourceReadOnlyAndBlendWarning(t *testing.T) {
	src := filepath.Join(fixtureDir, "obj_blend")
	generateFixtures(t)

	before, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", src, err)
	}
	res, staging := importPack(t, "obj_blend")

	warned := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "note.blend") && strings.Contains(w, "must be exported to FBX") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("expected BLEND export-first warning, got warnings: %v", res.Warnings)
	}

	after, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", src, err)
	}
	if len(before) != len(after) {
		t.Errorf("source dir changed: %d files before, %d after", len(before), len(after))
	}
	// Staging must exist and hold the outputs.
	if entries, err := os.ReadDir(staging); err != nil || len(entries) == 0 {
		t.Errorf("staging %s missing outputs: %v", staging, err)
	}
}

// TestPadTextureToPoTPng covers the padding math on a real PNG fixture:
// 300x128 -> 512x128 with edge extension, and no-op for power-of-two input.
func TestPadTextureToPoTPng(t *testing.T) {
	generateFixtures(t)
	data, err := os.ReadFile(filepath.Join(fixtureDir, "textures", "door_d.png"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	padded, err := asset.PadTextureToPoT(data)
	if err != nil {
		t.Fatalf("PadTextureToPoT: %v", err)
	}
	if !padded.Changed {
		t.Fatal("Changed = false, want true for 300x128")
	}
	if padded.OriginalWidth != 300 || padded.OriginalHeight != 128 {
		t.Errorf("original = %dx%d, want 300x128", padded.OriginalWidth, padded.OriginalHeight)
	}
	if padded.Width != 512 || padded.Height != 128 {
		t.Errorf("padded = %dx%d, want 512x128", padded.Width, padded.Height)
	}
	img, err := png.Decode(bytes.NewReader(padded.Data))
	if err != nil {
		t.Fatalf("padded output is not PNG: %v", err)
	}
	// Edge extension: padded columns/rows replicate the source border.
	srcImg, _ := png.Decode(bytes.NewReader(data))
	sr, sg, sb, _ := srcImg.At(299, 127).RGBA()
	if pr, pg, pb, _ := img.At(511, 127).RGBA(); pr != sr || pg != sg || pb != sb {
		t.Errorf("padded (511,127) = %d,%d,%d, want source border (299,127) %d,%d,%d", pr, pg, pb, sr, sg, sb)
	}
	sr2, sg2, sb2, _ := srcImg.At(299, 50).RGBA()
	if pr, pg, pb, _ := img.At(300, 50).RGBA(); pr != sr2 || pg != sg2 || pb != sb2 {
		t.Errorf("padded (300,50) = %d,%d,%d, want source edge (299,50) %d,%d,%d", pr, pg, pb, sr2, sg2, sb2)
	}

	// Power-of-two input is a no-op.
	pot, err := asset.PadTextureToPoT(pngBytes(256, 128))
	if err != nil {
		t.Fatalf("PadTextureToPoT(pot): %v", err)
	}
	if pot.Changed || pot.Data != nil {
		t.Errorf("PoT input: Changed = %v, Data = %d bytes, want no-op", pot.Changed, len(pot.Data))
	}
}

// TestPadTextureToPoTTga covers TGA -> PNG conversion: KitBash3D ships TGA
// textures and the engine loads only PNG, so padding a TGA must produce a
// padded PNG.
func TestPadTextureToPoTTga(t *testing.T) {
	generateFixtures(t)
	data, err := os.ReadFile(filepath.Join(fixtureDir, "textures", "door_d.tga"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	padded, err := asset.PadTextureToPoT(data)
	if err != nil {
		t.Fatalf("PadTextureToPoT(tga): %v", err)
	}
	if !padded.Changed {
		t.Fatal("Changed = false, want true for 300x128 TGA")
	}
	if padded.Width != 512 || padded.Height != 128 {
		t.Errorf("padded = %dx%d, want 512x128", padded.Width, padded.Height)
	}
	img, err := png.Decode(bytes.NewReader(padded.Data))
	if err != nil {
		t.Fatalf("TGA padding did not produce PNG: %v", err)
	}
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != 512 || h != 128 {
		t.Errorf("padded PNG = %dx%d, want 512x128", w, h)
	}
	// Edge extension across the converted format: every padded column
	// replicates the source edge column (x = 299).
	br, bg, bb, _ := img.At(300, 127).RGBA()
	er, eg, eb, _ := img.At(511, 127).RGBA()
	if br != er || bg != eg || bb != eb {
		t.Errorf("edge extension broke: (300,127) = %d,%d,%d vs (511,127) = %d,%d,%d", br, bg, bb, er, eg, eb)
	}
}

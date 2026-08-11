// Package asset implements the Asset & Scene Pipeline import stage
// (ADR-0004, TR-asset-001, TR-asset-002): conversion of source art
// (KitBash3D FBX/OBJ) into Kaiju native mesh archives with scale and
// rotation normalization.
//
// Pipeline: Source (read-only) -> Staging (normalized output) -> Imported.
//
//   - Source directory is read-only. All output is written under StagingDir.
//   - Scale normalization: 1 unit = 1 meter, validated by a door reference
//     object known to be 2 meters tall. If the reference cannot be measured
//     the import rejects with an error — the pipeline never guesses scale.
//   - Rotation normalization: Z-up sources are converted to Kaiju's Y-up.
//     FBX carries its own axis metadata which the engine basis converter
//     applies during load (verified against kaiju/src/rendering/loaders/fbx);
//     OBJ carries none, so the pipeline applies the explicit Z-up -> Y-up
//     rotation for OBJ sources.
//   - BLEND files are never imported; they must be exported to FBX first
//     (manual or headless Blender). The import warns when it finds them.
//   - Non-power-of-two textures are padded to the next PoT dimension with
//     edge extension and a warning. Material mapping is ASSET-002; this
//     stage only records source texture keys in the manifest.
package asset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaijuengine.com/engine/assets"
	"kaijuengine.com/matrix"
	"kaijuengine.com/rendering/loaders"
	"kaijuengine.com/rendering/loaders/kaiju_mesh"
	"kaijuengine.com/rendering/loaders/load_result"
)

// Import errors. These are sentinel errors, use errors.Is to match them.
var (
	// ErrSourceDirMissing is returned when the configured source directory
	// does not exist or is not a directory.
	ErrSourceDirMissing = errors.New("asset: source directory does not exist")
	// ErrStagingInsideSource is returned when the staging directory is the
	// source directory or inside it. Staging must never touch the read-only
	// source tree (ADR-0004).
	ErrStagingInsideSource = errors.New("asset: staging directory must not be inside the read-only source directory")
	// ErrNoImportableFiles is returned when the source directory contains no
	// .fbx or .obj files.
	ErrNoImportableFiles = errors.New("asset: no importable files (.fbx/.obj) found in source directory")
	// ErrDoorReferenceNotFound is returned when no mesh matches the door
	// reference name. The pipeline refuses to guess scale (ADR-0004).
	ErrDoorReferenceNotFound = errors.New("asset: scale reference not found, refusing to guess scale")
	// ErrNotYUp is returned when the door reference has no usable height on
	// the Y axis, i.e. rotation normalization did not produce a Y-up asset.
	// The pipeline refuses to guess rotation (ADR-0004).
	ErrNotYUp = errors.New("asset: reference object is not Y-up, refusing to guess rotation")
	// ErrScaleOutOfRange is returned when the door-derived scale factor is
	// outside the configured sanity range. The pipeline refuses to guess
	// scale (ADR-0004).
	ErrScaleOutOfRange = errors.New("asset: door-derived scale factor is outside the sane range, refusing to guess scale")
)

// Project defaults for import configuration. Override via Config — zero
// values fall back to these constants (same pattern as save.Config).
const (
	// DefaultDoorReferenceName is the substring matched against mesh and
	// node names to locate the scale reference object. KitBash3D door
	// assets are named like "Door_DarkWood" / "Door_EnchantedOak".
	DefaultDoorReferenceName = "door"
	// DefaultDoorHeightMeters is the known height of the reference object:
	// a human-height door at 2 world units (1 unit = 1 meter).
	DefaultDoorHeightMeters = matrix.Float(2.0)
	// DefaultSourceUpAxis is the up axis of OBJ sources ("Z" for Blender/
	// KitBash3D exports). FBX sources carry their own axis metadata and are
	// converted by the engine loader, so this only applies to OBJ.
	DefaultSourceUpAxis = "Z"
	// DefaultMinScaleFactor and DefaultMaxScaleFactor bound the door-derived
	// scale factor. A factor outside this range means the reference measured
	// nonsense and the import rejects rather than guesses.
	DefaultMinScaleFactor = matrix.Float(0.001)
	DefaultMaxScaleFactor = matrix.Float(1000)
)

// embeddedTexturePrefix marks texture keys whose bytes are stored in the
// load result (FBX video Content nodes) instead of on disk.
const embeddedTexturePrefix = "embedded_"

// Config drives an import. Zero values fall back to the project defaults.
type Config struct {
	// SourceDir is the read-only KitBash3D source directory (e.g.
	// "assets/source/Dark Fantasy"). Required.
	SourceDir string
	// StagingDir receives the normalized output (e.g. "assets/staging").
	// Required. Must not be inside SourceDir.
	StagingDir string
	// DoorReferenceName is a case-insensitive substring matched against
	// mesh and node names to find the scale reference. Default "door".
	DoorReferenceName string
	// DoorHeightMeters is the known height of the reference object in
	// meters (world units). Default 2.0.
	DoorHeightMeters matrix.Float
	// SourceUpAxis is the up axis of OBJ sources: "Z" (KitBash3D default)
	// or "Y". FBX sources are converted by the engine and are unaffected.
	// Default "Z".
	SourceUpAxis string
	// MinScaleFactor and MaxScaleFactor bound the door-derived scale
	// factor. Defaults 0.001 and 1000.
	MinScaleFactor matrix.Float
	MaxScaleFactor matrix.Float
}

// Result reports what an import produced. Warnings never fail the import;
// errors do.
type Result struct {
	// ScaleFactor maps 1 source unit to this many meters. Uniform across
	// the whole import (one pack).
	ScaleFactor matrix.Float
	// DoorHeightSource is the door reference height measured in source
	// units after rotation normalization (before scaling).
	DoorHeightSource matrix.Float
	// Meshes lists every mesh written to staging, in deterministic order.
	Meshes []MeshResult
	// Warnings collects non-fatal findings (BLEND files, texture padding,
	// undecodable textures).
	Warnings []string
}

// MeshResult describes one converted mesh in the staging directory.
type MeshResult struct {
	// SourceFile is the source path relative to SourceDir.
	SourceFile string
	// Key is the stable Kaiju mesh key (kebab-case, unique per source file).
	Key string
	// Name is the source mesh name.
	Name string
	// VertexCount and TriangleCount describe the normalized geometry.
	VertexCount   int
	TriangleCount int
	// Textures maps material slot -> source texture key. Material mapping
	// is ASSET-002; this records the inputs.
	Textures map[string]string
	// OutputFile is the staging path of the Kaiju native mesh archive.
	OutputFile string
	// PaddedTextures lists textures that were padded to power-of-two.
	PaddedTextures []PaddedTexture
}

// PaddedTexture describes a texture that was padded to the next
// power-of-two dimension during import.
type PaddedTexture struct {
	// Key is the source texture key.
	Key string
	// OutputFile is the staging path of the padded PNG.
	OutputFile string
	// OriginalWidth/OriginalHeight are the source dimensions.
	OriginalWidth  int
	OriginalHeight int
	// Width/Height are the padded dimensions (power of two).
	Width  int
	Height int
}

// PaddedImage is the result of PadTextureToPoT.
type PaddedImage struct {
	// Data is the PNG-encoded padded image. Nil when Changed is false.
	Data []byte
	// Changed reports whether either dimension was padded.
	Changed bool
	// OriginalWidth/OriginalHeight are the source dimensions.
	OriginalWidth  int
	OriginalHeight int
	// Width/Height are the output dimensions (power of two when Changed).
	Width  int
	Height int
}

// Manifest is the JSON file written next to a source file's meshes in
// staging. It records everything downstream stages need: the normalized
// geometry stats, the source texture keys per material slot (the input to
// ASSET-002 material mapping), and the padded texture outputs.
type Manifest struct {
	// Source is the source file path relative to SourceDir.
	Source string `json:"source"`
	// ScaleFactor maps 1 source unit to this many meters.
	ScaleFactor matrix.Float `json:"scaleFactor"`
	// Meshes lists the converted meshes and their metadata.
	Meshes []MeshResult `json:"meshes"`
}

// sourceEntry is one loaded source file plus its parsed result.
type sourceEntry struct {
	rel    string // path relative to the source root, slash-separated
	isOBJ  bool
	result load_result.Result
}

// doorMeasurement is the reference object geometry that drives scale.
type doorMeasurement struct {
	heightY matrix.Float // height along Kaiju Y (up), source units
	extentZ matrix.Float // extent along Kaiju Z, source units
}

// ImportAsset runs the full import pipeline over one source directory (one
// KitBash3D pack): load every FBX/OBJ, normalize rotation and scale against
// the door reference, serialize meshes to Kaiju native archives in staging,
// pad non-power-of-two textures, and write a manifest per source file.
//
// The source directory is only ever read. All writes go to StagingDir.
//
//	cfg := asset.Config{
//	    SourceDir:  "assets/source/Dark Fantasy",
//	    StagingDir: "assets/staging",
//	}
//	result, err := asset.ImportAsset(cfg)
func ImportAsset(config Config) (Result, error) {
	cfg := configWithDefaults(config)
	if err := validateConfig(cfg); err != nil {
		return Result{}, err
	}
	files, warnings, err := collectSourceFiles(cfg.SourceDir)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{}, ErrNoImportableFiles
	}
	db, err := assets.NewFileDatabase(cfg.SourceDir)
	if err != nil {
		return Result{}, err
	}
	entries := make([]sourceEntry, 0, len(files))
	for _, rel := range files {
		res, err := loadSourceFile(db, rel)
		if err != nil {
			return Result{}, fmt.Errorf("import %s: %w", rel, err)
		}
		if isOBJPath(rel) && cfg.SourceUpAxis == "Z" {
			rotateResultZUpToYUp(&res)
		}
		entries = append(entries, sourceEntry{rel: rel, isOBJ: isOBJPath(rel), result: res})
	}
	door, err := findDoorReference(entries, cfg.DoorReferenceName)
	if err != nil {
		return Result{}, err
	}
	if door.extentZ > door.heightY {
		return Result{}, fmt.Errorf(
			"%w: reference spans %.3f units on Z but only %.3f on Y",
			ErrNotYUp, door.extentZ, door.heightY)
	}
	factor, err := computeScaleFactor(door.heightY, cfg)
	if err != nil {
		return Result{}, err
	}
	for i := range entries {
		for j := range entries[i].result.Meshes {
			applyUniformScale(&entries[i].result.Meshes[j], factor)
		}
	}

	result := Result{
		ScaleFactor:      factor,
		DoorHeightSource: door.heightY,
		Warnings:         warnings,
	}
	for i := range entries {
		entry := &entries[i]
		outDir := stagingDirFor(cfg.StagingDir, entry.rel)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return Result{}, fmt.Errorf("create staging dir %s: %w", outDir, err)
		}
		meshes := kaiju_mesh.LoadedResultToKaijuMesh(entry.result)
		manifest := Manifest{Source: entry.rel, ScaleFactor: factor}
		for _, km := range meshes {
			// Texture URIs in the archive would reference transient source
			// paths. The pipeline clears them for serialization but keeps the
			// keys for the manifest and the PoT padding pass.
			texSlots := km.Textures
			km.Textures = nil
			data, err := km.Serialize()
			if err != nil {
				return Result{}, fmt.Errorf("serialize %s/%s: %w", entry.rel, km.Key, err)
			}
			outFile := filepath.Join(outDir, km.Key+".glb")
			if err := os.WriteFile(outFile, data, 0o644); err != nil {
				return Result{}, fmt.Errorf("write %s: %w", outFile, err)
			}
			mres := MeshResult{
				SourceFile:    entry.rel,
				Key:           km.Key,
				Name:          km.Name,
				VertexCount:   len(km.Verts),
				TriangleCount: len(km.Indexes) / 3,
				Textures:      texSlots,
				OutputFile:    outFile,
			}
			for slot, texKey := range texSlots {
				padded, warn, err := padSourceTexture(texKey, entry, db, outDir)
				if err != nil {
					return Result{}, fmt.Errorf("%s: %w", entry.rel, err)
				}
				if warn != "" {
					result.Warnings = append(result.Warnings, warn)
					slog.Warn(warn, "source", entry.rel, "mesh", km.Key, "slot", slot)
				}
				if padded != nil {
					mres.PaddedTextures = append(mres.PaddedTextures, *padded)
				}
			}
			manifest.Meshes = append(manifest.Meshes, mres)
			result.Meshes = append(result.Meshes, mres)
		}
		manifestData, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return Result{}, fmt.Errorf("encode manifest for %s: %w", entry.rel, err)
		}
		manifestFile := filepath.Join(outDir, "manifest.json")
		if err := os.WriteFile(manifestFile, append(manifestData, '\n'), 0o644); err != nil {
			return Result{}, fmt.Errorf("write manifest %s: %w", manifestFile, err)
		}
	}
	return result, nil
}

// configWithDefaults fills zero-value config fields with project defaults.
func configWithDefaults(cfg Config) Config {
	if cfg.DoorReferenceName == "" {
		cfg.DoorReferenceName = DefaultDoorReferenceName
	}
	if cfg.DoorHeightMeters <= 0 {
		cfg.DoorHeightMeters = DefaultDoorHeightMeters
	}
	if strings.ToUpper(cfg.SourceUpAxis) != "Y" {
		cfg.SourceUpAxis = DefaultSourceUpAxis
	}
	if cfg.MinScaleFactor <= 0 {
		cfg.MinScaleFactor = DefaultMinScaleFactor
	}
	if cfg.MaxScaleFactor <= 0 {
		cfg.MaxScaleFactor = DefaultMaxScaleFactor
	}
	return cfg
}

// validateConfig enforces the ADR-0004 source/staging separation: the source
// directory must exist and staging must not be inside it.
func validateConfig(cfg Config) error {
	info, err := os.Stat(cfg.SourceDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrSourceDirMissing, cfg.SourceDir)
	}
	src, err := filepath.Abs(cfg.SourceDir)
	if err != nil {
		return err
	}
	stg, err := filepath.Abs(cfg.StagingDir)
	if err != nil {
		return err
	}
	if stg == src || strings.HasPrefix(stg, src+string(filepath.Separator)) {
		return fmt.Errorf("%w: staging %s is inside source %s",
			ErrStagingInsideSource, cfg.StagingDir, cfg.SourceDir)
	}
	return nil
}

// collectSourceFiles walks the source directory and returns the sorted
// importable files (.fbx/.obj) relative to the source root. BLEND files are
// not importable — the pipeline requires an FBX export first (manual or
// headless Blender) — and produce a warning each. Hidden entries are
// skipped.
func collectSourceFiles(sourceDir string) ([]string, []string, error) {
	var files, blendWarnings []string
	err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch strings.ToLower(filepath.Ext(d.Name())) {
		case ".fbx", ".obj":
			files = append(files, rel)
		case ".blend":
			blendWarnings = append(blendWarnings,
				fmt.Sprintf("BLEND file %q must be exported to FBX first "+
					"(manual or headless Blender); skipped", rel))
		}
		return nil
	})
	sort.Strings(files)
	return files, blendWarnings, err
}

// loadSourceFile loads one source file through the engine loaders. The
// engine FBX loader applies the file's axis metadata (Z-up -> Y-up) and
// unit scale during parse; the OBJ loader passes geometry through raw.
func loadSourceFile(db assets.Database, rel string) (load_result.Result, error) {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".fbx":
		return loaders.FBX(rel, db)
	case ".obj":
		return loaders.OBJ(rel, db)
	default:
		return load_result.Result{}, fmt.Errorf("unsupported source format %q", rel)
	}
}

func isOBJPath(rel string) bool {
	return strings.EqualFold(filepath.Ext(rel), ".obj")
}

// rotateResultZUpToYUp converts every vertex of every mesh from Z-up to
// Kaiju's Y-up basis: (x, y, z) -> (-x, z, -y). This mirrors the engine's
// FBX basis converter for Blender/KitBash3D Z-up exports (kaiju/src/
// rendering/loaders/fbx/transform.go — ConvertPosition with UpAxis=2,
// FrontAxis=1), so OBJ imports land in the same orientation as the
// equivalent FBX. Positions, normals and tangent XYZ are rotated; UVs,
// colors and tangent W are unchanged. Only OBJ sources need this — FBX
// carries its own axis metadata and the engine loader already converted it.
func rotateResultZUpToYUp(res *load_result.Result) {
	for i := range res.Meshes {
		for j := range res.Meshes[i].Verts {
			v := &res.Meshes[i].Verts[j]
			p, n, t := v.Position, v.Normal, v.Tangent
			v.Position = matrix.NewVec3(-p.X(), p.Z(), -p.Y())
			v.Normal = matrix.NewVec3(-n.X(), n.Z(), -n.Y())
			v.Tangent = matrix.NewVec4(-t.X(), t.Z(), -t.Y(), t.W())
		}
	}
}

// meshBounds returns the axis-aligned bounds of a mesh's positions.
func meshBounds(m *load_result.Mesh) (min, max matrix.Vec3, ok bool) {
	if len(m.Verts) == 0 {
		return matrix.Vec3Zero(), matrix.Vec3Zero(), false
	}
	min = m.Verts[0].Position
	max = min
	for i := 1; i < len(m.Verts); i++ {
		pos := m.Verts[i].Position
		min = matrix.Vec3Min(min, pos)
		max = matrix.Vec3Max(max, pos)
	}
	return min, max, true
}

// findDoorReference locates the scale reference: the first mesh whose name
// or node name contains the reference substring, in deterministic document
// order. Its Y height (source units, after rotation normalization) drives
// the pack scale factor. A matched door with no Y height is a hard error —
// the pipeline never guesses scale.
func findDoorReference(entries []sourceEntry, refName string) (doorMeasurement, error) {
	ref := strings.ToLower(refName)
	for i := range entries {
		for j := range entries[i].result.Meshes {
			m := &entries[i].result.Meshes[j]
			name := strings.ToLower(m.Name + " " + m.MeshName)
			if !strings.Contains(name, ref) {
				continue
			}
			min, max, ok := meshBounds(m)
			if !ok {
				continue
			}
			heightY := max.Y() - min.Y()
			if heightY <= 0 {
				return doorMeasurement{}, fmt.Errorf(
					"%w: reference mesh %q has no height on Y (%.3f)",
					ErrNotYUp, m.Name, heightY)
			}
			return doorMeasurement{heightY: heightY, extentZ: max.Z() - min.Z()}, nil
		}
	}
	return doorMeasurement{}, fmt.Errorf("%w (reference name %q)",
		ErrDoorReferenceNotFound, refName)
}

// computeScaleFactor derives the uniform pack scale from the door
// reference: known height in meters divided by measured height in source
// units. Out-of-range or non-finite results reject the import.
func computeScaleFactor(doorHeight matrix.Float, cfg Config) (matrix.Float, error) {
	if doorHeight <= 0 || math.IsNaN(float64(doorHeight)) || math.IsInf(float64(doorHeight), 0) {
		return 0, fmt.Errorf("%w: door height %v is not a positive finite number",
			ErrScaleOutOfRange, doorHeight)
	}
	factor := cfg.DoorHeightMeters / doorHeight
	if factor < cfg.MinScaleFactor || factor > cfg.MaxScaleFactor {
		return 0, fmt.Errorf("%w: door height %v implies scale %v outside [%v, %v]",
			ErrScaleOutOfRange, doorHeight, factor, cfg.MinScaleFactor, cfg.MaxScaleFactor)
	}
	return factor, nil
}

// applyUniformScale scales every vertex position by factor. Normals and
// tangents are direction vectors and are unchanged by uniform scale.
func applyUniformScale(m *load_result.Mesh, factor matrix.Float) {
	if factor == 1 {
		return
	}
	for i := range m.Verts {
		m.Verts[i].Position = m.Verts[i].Position.Scale(factor)
	}
}

// stagingDirFor returns the staging directory for one source file: the
// source-relative path without extension, preserving the source layout
// (e.g. "Props/Door.fbx" -> "<staging>/Props/Door").
func stagingDirFor(stagingDir, rel string) string {
	ext := filepath.Ext(rel)
	return filepath.Join(stagingDir, filepath.Dir(rel), strings.TrimSuffix(rel, ext))
}

// padSourceTexture resolves a mesh texture key to bytes, pads it to
// power-of-two dimensions when needed, and writes the padded PNG to staging.
// It returns a non-nil *PaddedTexture only when padding changed something.
// Missing textures are errors (the GDD rejects missing texture references);
// undecodable textures warn and pass through (format conversion is
// ASSET-002).
func padSourceTexture(texKey string, entry *sourceEntry, db assets.Database,
	outDir string) (*PaddedTexture, string, error) {
	data, err := resolveTextureBytes(texKey, entry, db)
	if err != nil {
		return nil, "", err
	}
	padded, err := PadTextureToPoT(data)
	if err != nil {
		warn := fmt.Sprintf("texture %q could not be decoded (%v); "+
			"PoT check skipped (ASSET-002 handles texture conversion)", texKey, err)
		return nil, warn, nil
	}
	if !padded.Changed {
		return nil, "", nil
	}
	texDir := filepath.Join(outDir, "textures")
	if err := os.MkdirAll(texDir, 0o755); err != nil {
		return nil, "", err
	}
	outFile := filepath.Join(texDir, textureFileName(texKey))
	if err := os.WriteFile(outFile, padded.Data, 0o644); err != nil {
		return nil, "", err
	}
	return &PaddedTexture{
		Key:            texKey,
		OutputFile:     outFile,
		OriginalWidth:  padded.OriginalWidth,
		OriginalHeight: padded.OriginalHeight,
		Width:          padded.Width,
		Height:         padded.Height,
	}, fmt.Sprintf("texture %q padded %dx%d -> %dx%d",
		texKey, padded.OriginalWidth, padded.OriginalHeight, padded.Width, padded.Height), nil
}

// resolveTextureBytes fetches a texture's bytes. Keys with the embedded_
// prefix come from the load result (FBX video content); other keys are
// paths relative to the source root. Absolute paths escape the file
// database root and fall back to a direct read (source stays read-only).
func resolveTextureBytes(texKey string, entry *sourceEntry,
	db assets.Database) ([]byte, error) {
	if strings.HasPrefix(texKey, embeddedTexturePrefix) {
		data, ok := entry.result.TextureBytes[texKey]
		if !ok {
			return nil, fmt.Errorf("embedded texture %q not found in import result", texKey)
		}
		return data, nil
	}
	data, err := db.Read(texKey)
	if err != nil && filepath.IsAbs(texKey) {
		data, err = os.ReadFile(texKey)
	}
	if err != nil {
		return nil, fmt.Errorf("texture %q: %w", texKey, err)
	}
	return data, nil
}

// textureFileName derives a stable kebab-case file name for a padded
// texture (e.g. "Textures/Door_D.png" -> "door_d.png",
// "embedded_10003_baseColor" -> "embedded_10003_basecolor.png").
func textureFileName(texKey string) string {
	base := strings.ToLower(filepath.Base(texKey))
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	var sb strings.Builder
	lastSep := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			lastSep = false
		} else if !lastSep && sb.Len() > 0 {
			sb.WriteByte('_')
			lastSep = true
		}
	}
	name := strings.Trim(sb.String(), "_")
	if name == "" {
		name = "texture"
	}
	return name + ".png"
}

// PadTextureToPoT decodes a PNG or TGA texture and, when either dimension
// is not a power of two, returns a PNG padded to the next power of two
// using edge extension (the border rows/columns replicate). Output is
// always PNG — the engine's texture loader reads only PNG, so TGA textures
// are converted here when padding is required. When neither dimension
// changes, Changed is false and Data is nil.
//
//	data, _ := os.ReadFile("door_d.png")
//	padded, err := asset.PadTextureToPoT(data)
//	if err == nil && padded.Changed {
//	    os.WriteFile("door_d_padded.png", padded.Data, 0o644)
//	}
func PadTextureToPoT(data []byte) (PaddedImage, error) {
	img, _, err := decodeTexture(data)
	if err != nil {
		return PaddedImage{}, fmt.Errorf("decode texture: %w", err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return PaddedImage{}, errors.New("texture has empty bounds")
	}
	nw, nh := nextPoT(w), nextPoT(h)
	if nw == w && nh == h {
		return PaddedImage{
			Changed:         false,
			OriginalWidth:   w,
			OriginalHeight:  h,
			Width:           w,
			Height:          h,
		}, nil
	}
	padded := padWithEdgeExtension(img, nw, nh)
	var buf bytes.Buffer
	if err := png.Encode(&buf, padded); err != nil {
		return PaddedImage{}, fmt.Errorf("encode padded texture: %w", err)
	}
	return PaddedImage{
		Data:            buf.Bytes(),
		Changed:         true,
		OriginalWidth:   w,
		OriginalHeight:  h,
		Width:           nw,
		Height:          nh,
	}, nil
}

// decodeTexture decodes a PNG or TGA texture. PNG is detected by its magic
// bytes and decoded with image/png; anything else is treated as TGA, which
// KitBash3D ships alongside PNG textures.
func decodeTexture(data []byte) (image.Image, string, error) {
	if len(data) >= 8 && bytes.Equal(data[:8], pngMagic) {
		img, err := png.Decode(bytes.NewReader(data))
		return img, "png", err
	}
	img, err := decodeTGA(data)
	return img, "tga", err
}

var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// decodeTGA decodes uncompressed (type 2) and RLE (type 10) true-color TGA
// images with 24 or 32 bits per pixel. TGA stores BGR(A) pixels from the
// bottom-left by default; the top-left origin bit flips the rows.
// Color-mapped and grayscale TGA variants are not produced by KitBash3D
// exports and are rejected. The engine's texture loader only reads PNG, so
// TGA textures must be converted to PNG at import time — the padding step
// writes PNG output.
func decodeTGA(data []byte) (image.Image, error) {
	if len(data) < 18 {
		return nil, errors.New("tga: header too short")
	}
	if data[1] != 0 {
		return nil, errors.New("tga: color-mapped images are not supported")
	}
	imgType := data[2]
	if imgType != 2 && imgType != 10 {
		return nil, fmt.Errorf("tga: unsupported image type %d (want 2 or 10)", imgType)
	}
	w := int(data[12]) | int(data[13])<<8
	h := int(data[14]) | int(data[15])<<8
	depth := int(data[16])
	if depth != 24 && depth != 32 {
		return nil, fmt.Errorf("tga: unsupported pixel depth %d (want 24 or 32)", depth)
	}
	if w <= 0 || h <= 0 {
		return nil, errors.New("tga: empty image")
	}
	topDown := data[17]&0x20 != 0
	bpp := depth / 8
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	pos := int(data[0]) + 18 // skip the image ID block
	set := func(i int, c color.RGBA) {
		y := i / w
		x := i % w
		if !topDown {
			y = h - 1 - y
		}
		img.SetRGBA(x, y, c)
	}
	readPixel := func() (color.RGBA, error) {
		if pos+bpp > len(data) {
			return color.RGBA{}, errors.New("tga: truncated pixel data")
		}
		c := color.RGBA{R: data[pos+2], G: data[pos+1], B: data[pos], A: 0xFF}
		if bpp == 4 {
			c.A = data[pos+3]
		}
		pos += bpp
		return c, nil
	}
	total := w * h
	if imgType == 10 {
		for i := 0; i < total; {
			if pos >= len(data) {
				return nil, errors.New("tga: truncated RLE packet data")
			}
			packet := data[pos]
			pos++
			count := int(packet&0x7F) + 1
			if packet&0x80 != 0 {
				// RLE run: one pixel repeated count times.
				c, err := readPixel()
				if err != nil {
					return nil, err
				}
				for j := 0; j < count && i < total; j++ {
					set(i, c)
					i++
				}
				continue
			}
			// Raw packet: count literal pixels.
			for j := 0; j < count && i < total; j++ {
				c, err := readPixel()
				if err != nil {
					return nil, err
				}
				set(i, c)
				i++
			}
		}
		return img, nil
	}
	for i := 0; i < total; i++ {
		c, err := readPixel()
		if err != nil {
			return nil, err
		}
		set(i, c)
	}
	return img, nil
}

// nextPoT returns n when n is a power of two, otherwise the next power of
// two greater than n.
func nextPoT(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// padWithEdgeExtension stretches an image to w x h by replicating the edge
// rows and columns.
func padWithEdgeExtension(src image.Image, w, h int) *image.RGBA {
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := min(y, sh-1)
		for x := 0; x < w; x++ {
			out.Set(x, y, src.At(min(x, sw-1), sy))
		}
	}
	return out
}

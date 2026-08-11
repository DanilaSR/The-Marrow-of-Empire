// Test fixture builders for the ASSET-001 integration tests.
//
// The engine FBX parser accepts binary FBX only (ASCII is rejected), so the
// FBX fixture is a minimal hand-crafted binary FBX 7400 document: one
// Z-up door mesh, one material, one embedded TGA texture. The record layout
// mirrors what the engine parser expects (kaiju/src/rendering/loaders/fbx/
// document.go): 23-byte magic header, uint32 version, then records with
// absolute end offsets and 13-byte null sentinels.
//
// All fixtures are generated into testdata/asset/ at test runtime
// (deterministic, idempotent) so the tests exercise real files on disk.
package asset_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fixtureDir is the testdata root relative to the package directory
// (go test sets the working directory to the package dir).
const fixtureDir = "testdata/asset"

var fixtureOnce sync.Once

// generateFixtures writes all test fixtures to testdata/asset/. It is
// idempotent and safe to call from any test.
func generateFixtures(t *testing.T) {
	t.Helper()
	fixtureOnce.Do(func() {
		if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", fixtureDir, err)
		}
		// FBX pack: Z-up door, 200 source units tall, embedded non-PoT TGA.
		dir := filepath.Join(fixtureDir, "fbx")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		fbxDoorFixture(t, filepath.Join(dir, "door.fbx"), fbxFixtureOptions{
			doorHeight:  200,
			upAxis:      2,
			embeddedTGA: tgaBytes(300, 128),
		})
		// OBJ pack A: door 200 units + crate 100 units (Z-up).
		dir = filepath.Join(fixtureDir, "obj_door_200u")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		writeOBJ(t, dir, "door_zup.obj", zUpDoorOBJ(200, 1, 100))
		// OBJ pack B: door 1 unit + crate 0.5 units — different scale, same
		// real-world meters after normalization.
		dir = filepath.Join(fixtureDir, "obj_door_1u")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		writeOBJ(t, dir, "door_1u.obj", zUpDoorOBJ(1, 1, 0.5))
		// Pack with a BLEND file next to the OBJ (export-first warning).
		dir = filepath.Join(fixtureDir, "obj_blend")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		writeOBJ(t, dir, "door_zup.obj", zUpDoorOBJ(200, 1, 100))
		if err := os.WriteFile(filepath.Join(dir, "note.blend"), []byte("not a real blend"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", filepath.Join(dir, "note.blend"), err)
		}
		// Standalone KitBash3D-style textures for the pure padding tests.
		dir = filepath.Join(fixtureDir, "textures")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		writeTGA(t, dir, "door_d.tga", 300, 128)
		writePNG(t, dir, "door_d.png", 300, 128)
	})
}

// zUpDoorOBJ builds a Z-up OBJ: a door (height x width, depth 0.5) and,
// when crateHeight > 0, a second object "Crate" of that height. KitBash3D
// OBJ exports are Z-up, so the import must rotate them to Y-up.
func zUpDoorOBJ(doorHeight, doorWidth, crateHeight float64) string {
	var sb strings.Builder
	sb.WriteString("# KitBash3D-style Z-up door fixture\n")
	sb.WriteString("o Door\n")
	w := doorWidth
	for _, v := range [][3]float64{
		{0, 0, 0}, {w, 0, 0}, {w, 0.5, 0}, {0, 0.5, 0},
		{0, 0, doorHeight}, {w, 0, doorHeight}, {w, 0.5, doorHeight}, {0, 0.5, doorHeight},
	} {
		fmt.Fprintf(&sb, "v %g %g %g\n", v[0], v[1], v[2])
	}
	sb.WriteString("vt 0 0\n")
	for _, n := range [][3]float64{
		{0, 0, -1}, {0, 0, 1}, {0, -1, 0}, {0, 1, 0}, {-1, 0, 0}, {1, 0, 0},
	} {
		fmt.Fprintf(&sb, "vn %g %g %g\n", n[0], n[1], n[2])
	}
	// 6 quads, fanned by the loader. Face order: bottom, top, -Y, +Y, -X, +X.
	sb.WriteString("f 1/1/1 3/1/1 4/1/1\nf 1/1/1 2/1/1 3/1/1\n")
	sb.WriteString("f 5/1/2 6/1/2 7/1/2\nf 5/1/2 7/1/2 8/1/2\n")
	sb.WriteString("f 1/1/3 2/1/3 6/1/3\nf 1/1/3 6/1/3 5/1/3\n")
	sb.WriteString("f 4/1/4 3/1/4 7/1/4\nf 4/1/4 7/1/4 8/1/4\n")
	sb.WriteString("f 1/1/5 5/1/5 8/1/5\nf 1/1/5 8/1/5 4/1/5\n")
	sb.WriteString("f 2/1/6 3/1/6 7/1/6\nf 2/1/6 7/1/6 6/1/6\n")
	if crateHeight > 0 {
		sb.WriteString("o Crate\n")
		for _, v := range [][3]float64{
			{0, 0, 0}, {2, 0, 0}, {2, 2, 0}, {0, 2, 0},
			{0, 0, crateHeight}, {2, 0, crateHeight}, {2, 2, crateHeight}, {0, 2, crateHeight},
		} {
			fmt.Fprintf(&sb, "v %g %g %g\n", v[0], v[1], v[2])
		}
		sb.WriteString("vt 0 0\n")
		for _, n := range [][3]float64{
			{0, 0, -1}, {0, 0, 1}, {0, -1, 0}, {0, 1, 0}, {-1, 0, 0}, {1, 0, 0},
		} {
			fmt.Fprintf(&sb, "vn %g %g %g\n", n[0], n[1], n[2])
		}
		// Crate vertices are library indices 9-16.
		sb.WriteString("f 9/1/1 11/1/1 12/1/1\nf 9/1/1 10/1/1 11/1/1\n")
		sb.WriteString("f 13/1/2 14/1/2 15/1/2\nf 13/1/2 15/1/2 16/1/2\n")
		sb.WriteString("f 9/1/3 10/1/3 14/1/3\nf 9/1/3 14/1/3 13/1/3\n")
		sb.WriteString("f 12/1/4 11/1/4 15/1/4\nf 12/1/4 15/1/4 16/1/4\n")
		sb.WriteString("f 9/1/5 13/1/5 16/1/5\nf 9/1/5 16/1/5 12/1/5\n")
		sb.WriteString("f 10/1/6 11/1/6 15/1/6\nf 10/1/6 15/1/6 14/1/6\n")
	}
	return sb.String()
}

// writeOBJ writes an OBJ fixture into dir and returns its absolute path.
func writeOBJ(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

// pngBytes encodes an RGBA PNG of w x h and returns its bytes.
func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 0x80, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// writePNG writes an RGBA PNG of w x h and returns its bytes.
func writePNG(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	data := pngBytes(w, h)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return data
}

// tgaBytes builds an uncompressed true-color (type 2) 24-bit TGA of w x h
// with the top-left origin bit set. KitBash3D ships TGA textures, and the
// engine loads only PNG, so the import converts TGA to PNG when padding.
func tgaBytes(w, h int) []byte {
	hdr := make([]byte, 18)
	hdr[2] = 2 // uncompressed true-color
	hdr[12] = byte(w & 0xFF)
	hdr[13] = byte(w >> 8)
	hdr[14] = byte(h & 0xFF)
	hdr[15] = byte(h >> 8)
	hdr[16] = 24
	hdr[17] = 0x20 // top-left origin
	var buf bytes.Buffer
	buf.Write(hdr)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// BGR order, per the TGA spec.
			buf.Write([]byte{byte(x % 256), byte(y % 256), 0xFF})
		}
	}
	return buf.Bytes()
}

// writeTGA writes an uncompressed true-color (type 2) 24-bit TGA of w x h.
func writeTGA(t *testing.T, dir, name string, w, h int) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, tgaBytes(w, h), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// --- Minimal binary FBX 7400 writer --------------------------------------

// fbxProp is one encoded FBX record property.
type fbxProp struct {
	typ byte
	buf []byte
}

func pInt32(v int32) fbxProp { return fbxProp{'I', binary.LittleEndian.AppendUint32(nil, uint32(v))} }
func pInt64(v int64) fbxProp { return fbxProp{'L', binary.LittleEndian.AppendUint64(nil, uint64(v))} }
func pFloat64(v float64) fbxProp {
	return fbxProp{'D', binary.LittleEndian.AppendUint64(nil, math.Float64bits(v))}
}
func pString(s string) fbxProp {
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(s)))
	return fbxProp{'S', append(buf, s...)}
}
func pRaw(b []byte) fbxProp {
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(b)))
	return fbxProp{'R', append(buf, b...)}
}
func pFloat64Array(v []float64) fbxProp {
	payload := make([]byte, 0, 8*len(v))
	for _, f := range v {
		payload = binary.LittleEndian.AppendUint64(payload, math.Float64bits(f))
	}
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(v)))
	buf = binary.LittleEndian.AppendUint32(buf, 0) // raw encoding
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(payload)))
	return fbxProp{'d', append(buf, payload...)}
}
func pInt32Array(v []int32) fbxProp {
	payload := make([]byte, 0, 4*len(v))
	for _, i := range v {
		payload = binary.LittleEndian.AppendUint32(payload, uint32(i))
	}
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(v)))
	buf = binary.LittleEndian.AppendUint32(buf, 0)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(payload)))
	return fbxProp{'i', append(buf, payload...)}
}

// fbxNode is one FBX record: name, properties, children.
type fbxNode struct {
	name     string
	props    []fbxProp
	children []fbxNode
}

// build serializes the record with absolute end offsets (FBX 7400 layout:
// 32-bit offsets, 13-byte null sentinel). The record header is
// endOffset(4) + numProps(4) + propListLen(4) + nameLen(1) + name.
func (n fbxNode) build(offset int) []byte {
	props := encodeProps(n.props)
	var children []byte
	childOff := offset + 13 + len(n.name) + len(props)
	for _, c := range n.children {
		children = append(children, c.build(childOff+len(children))...)
	}
	end := childOff + len(children) + 13
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint32(end))
	binary.Write(&out, binary.LittleEndian, uint32(len(n.props)))
	binary.Write(&out, binary.LittleEndian, uint32(len(props)))
	out.WriteByte(byte(len(n.name)))
	out.WriteString(n.name)
	out.Write(props)
	out.Write(children)
	out.Write(make([]byte, 13)) // null sentinel
	return out.Bytes()
}

func encodeProps(props []fbxProp) []byte {
	var out []byte
	for _, p := range props {
		out = append(out, p.typ)
		out = append(out, p.buf...)
	}
	return out
}

// buildFBX assembles a full binary FBX 7400 document.
func buildFBX(rootNodes []fbxNode) []byte {
	var out []byte
	out = append(out, []byte("Kaydara FBX Binary  \x00\x1a\x00")...)
	out = binary.LittleEndian.AppendUint32(out, 7400)
	off := len(out)
	for _, n := range rootNodes {
		b := n.build(off)
		out = append(out, b...)
		off += len(b)
	}
	return out
}

// fbxFixtureOptions configures the synthesized door FBX.
type fbxFixtureOptions struct {
	doorHeight float64 // door height in source units (Z-up: along Z)
	upAxis     int     // 2 = Z-up (KitBash3D/Blender), 1 = Y-up
	// embeddedTGA is embedded as video content when non-nil; its width must
	// be non-power-of-two to exercise the padding path.
	embeddedTGA []byte
}

// fbxDoorFixture writes a minimal but complete binary FBX containing one
// door mesh (width 1, depth 0.5, height doorHeight in source units), one
// material, and optionally a texture with embedded video content. The
// layout mirrors a Blender 4.5 export.
func fbxDoorFixture(t *testing.T, path string, opts fbxFixtureOptions) {
	t.Helper()
	const (
		modelID    = 10000
		geometryID = 10001
		materialID = 10002
		textureID  = 10003
		videoID    = 10004
	)
	h := opts.doorHeight
	verts := []float64{
		0, 0, 0, 1, 0, 0, 1, 0.5, 0, 0, 0.5, 0,
		0, 0, h, 1, 0, h, 1, 0.5, h, 0, 0.5, h,
	}
	// 6 quads; each polygon's last corner index is negated minus one.
	polys := []int32{0, 1, 2, -4, 4, 5, 6, -8, 0, 3, 7, -5, 1, 2, 6, -7, 0, 1, 5, -4, 3, 2, 6, -8}
	normals := []float64{
		0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1,
		0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1,
		0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0,
		0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0,
		-1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0,
		1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0,
	}
	uvs := make([]float64, 0, 48)
	for range 6 { // one UV per polygon corner, 4 corners x 6 quads
		uvs = append(uvs, 0, 0, 1, 0, 1, 1, 0, 1)
	}
	upAxis := opts.upAxis
	if upAxis == 0 {
		upAxis = 2
	}
	property := func(name, typ, label string, values ...fbxProp) fbxNode {
		p := []fbxProp{pString(name), pString(typ), pString(label), pString("A")}
		return fbxNode{name: "P", props: append(p, values...)}
	}
	globalSettings := fbxNode{name: "GlobalSettings", children: []fbxNode{
		{name: "Version", props: []fbxProp{pInt32(1000)}},
		{name: "Properties70", children: []fbxNode{
			property("UpAxis", "int", "Integer", pInt32(int32(upAxis))),
			property("UpAxisSign", "int", "Integer", pInt32(1)),
			property("FrontAxis", "int", "Integer", pInt32(1)), // Blender Z-up export: forward = Y
			property("FrontAxisSign", "int", "Integer", pInt32(1)),
			property("CoordAxis", "int", "Integer", pInt32(0)),
			property("CoordAxisSign", "int", "Integer", pInt32(1)),
			property("UnitScaleFactor", "double", "Number", pFloat64(1)),
		}},
	}}
	geometry := fbxNode{name: "Geometry", props: []fbxProp{
		pInt64(geometryID), pString("DoorGeo\x00\x01Geometry"), pString("Mesh"),
	}, children: []fbxNode{
		{name: "Vertices", props: []fbxProp{pFloat64Array(verts)}},
		{name: "PolygonVertexIndex", props: []fbxProp{pInt32Array(polys)}},
		{name: "LayerElementNormal", children: []fbxNode{
			{name: "Version", props: []fbxProp{pInt32(101)}},
			{name: "Name", props: []fbxProp{pString("")}},
			{name: "MappingInformationType", props: []fbxProp{pString("ByPolygonVertex")}},
			{name: "ReferenceInformationType", props: []fbxProp{pString("Direct")}},
			{name: "Normals", props: []fbxProp{pFloat64Array(normals)}},
		}},
		{name: "LayerElementUV", children: []fbxNode{
			{name: "Version", props: []fbxProp{pInt32(101)}},
			{name: "Name", props: []fbxProp{pString("UVMap")}},
			{name: "MappingInformationType", props: []fbxProp{pString("ByPolygonVertex")}},
			{name: "ReferenceInformationType", props: []fbxProp{pString("Direct")}},
			{name: "UV", props: []fbxProp{pFloat64Array(uvs)}},
		}},
		{name: "LayerElementMaterial", children: []fbxNode{
			{name: "Version", props: []fbxProp{pInt32(101)}},
			{name: "Name", props: []fbxProp{pString("")}},
			{name: "MappingInformationType", props: []fbxProp{pString("AllSame")}},
			{name: "ReferenceInformationType", props: []fbxProp{pString("IndexToDirect")}},
			{name: "Materials", props: []fbxProp{pInt32Array([]int32{0})}},
		}},
	}}
	model := fbxNode{name: "Model", props: []fbxProp{
		pInt64(modelID), pString("Door\x00\x01Model"), pString("Mesh"),
	}, children: []fbxNode{
		{name: "Version", props: []fbxProp{pInt32(232)}},
		{name: "Properties70", children: []fbxNode{
			property("Lcl Translation", "Lcl Translation", "", pFloat64(0), pFloat64(0), pFloat64(0)),
			property("Lcl Rotation", "Lcl Rotation", "", pFloat64(0), pFloat64(0), pFloat64(0)),
			property("Lcl Scaling", "Lcl Scaling", "", pFloat64(1), pFloat64(1), pFloat64(1)),
		}},
		{name: "Shading", props: []fbxProp{{typ: 'C', buf: []byte{1}}}},
		{name: "Culling", props: []fbxProp{pString("CullingOff")}},
	}}
	material := fbxNode{name: "Material", props: []fbxProp{
		pInt64(materialID), pString("Stone\x00\x01Material"), pString(""),
	}, children: []fbxNode{
		{name: "Version", props: []fbxProp{pInt32(102)}},
		{name: "ShadingModel", props: []fbxProp{pString("phong")}},
		{name: "MultiLayer", props: []fbxProp{pInt32(0)}},
		{name: "Properties70", children: []fbxNode{
			property("DiffuseColor", "Color", "", pFloat64(0.8), pFloat64(0.8), pFloat64(0.8)),
		}},
	}}
	objects := []fbxNode{model, geometry, material}
	var connections []fbxNode
	connections = append(connections,
		fbxNode{name: "C", props: []fbxProp{pString("OO"), pInt64(modelID), pInt64(0)}},
		fbxNode{name: "C", props: []fbxProp{pString("OO"), pInt64(geometryID), pInt64(modelID)}},
		fbxNode{name: "C", props: []fbxProp{pString("OO"), pInt64(materialID), pInt64(modelID)}},
	)
	if opts.embeddedTGA != nil {
		texture := fbxNode{name: "Texture", props: []fbxProp{
			pInt64(textureID), pString("DoorTexture\x00\x01Texture"), pString(""),
		}, children: []fbxNode{
			{name: "Type", props: []fbxProp{pString("TextureVideoClip")}},
			{name: "Version", props: []fbxProp{pInt32(202)}},
			{name: "TextureName", props: []fbxProp{pString("DoorTexture")}},
			{name: "Media", props: []fbxProp{pString("Video::DoorVideo")}},
			{name: "FileName", props: []fbxProp{pString("Textures/door_d.tga")}},
			{name: "RelativeFilename", props: []fbxProp{pString("Textures/door_d.tga")}},
		}}
		video := fbxNode{name: "Video", props: []fbxProp{
			pInt64(videoID), pString("DoorVideo\x00\x01Video"), pString("Clip"),
		}, children: []fbxNode{
			{name: "Type", props: []fbxProp{pString("Clip")}},
			{name: "Version", props: []fbxProp{pInt32(202)}},
			{name: "Filename", props: []fbxProp{pString("Textures/door_d.tga")}},
			{name: "RelativeFilename", props: []fbxProp{pString("Textures/door_d.tga")}},
			{name: "Content", props: []fbxProp{pRaw(opts.embeddedTGA)}},
		}}
		objects = append(objects, texture, video)
		connections = append(connections,
			fbxNode{name: "C", props: []fbxProp{pString("OP"), pInt64(textureID), pInt64(materialID), pString("DiffuseColor")}},
			fbxNode{name: "C", props: []fbxProp{pString("OO"), pInt64(videoID), pInt64(textureID)}},
		)
	}
	doc := buildFBX([]fbxNode{
		{name: "Objects", children: objects},
		{name: "Connections", children: connections},
		globalSettings,
	})
	if err := os.WriteFile(path, doc, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// Region boundary enforcement for the isometric camera (Story CAM-004, GDD
// CR-6): the camera target is clamped so the viewport never shows void space
// beyond the region edge.
//
// The clamp distance is the GDD padding formula:
//
//	padding >= (ground_width_max + ground_depth) / (2*sqrt(2))
//
// with ground_width = 2*O*aspect (current camera width) and ground_depth =
// 4*O. This is exactly the far-side extent of the viewport's ground footprint
// — the AABB of the 45°-rotated viewport rect (verified symbolically and in
// tests: the far corner projection equals the formula for every aspect), so
// clamping the target to [region_min + padding, region_max - padding] keeps
// the entire visible ground inside the region. The near side of the footprint
// is smaller than the padding, so the formula is conservative there.
package camera

import (
	"math"

	"kaijuengine.com/matrix"
)

// Region is the playable XZ bounds of a game region (from the scene file per
// ADR-0003). Y is unbounded — the camera focus elevation is preserved.
type Region struct {
	MinX, MaxX float32
	MinZ, MaxZ float32
}

// Boundary clamps the camera target against a Region. The clamp distance
// derives from the camera's current width and half-height, so it adapts to
// zoom transitions and aspect-ratio resizes automatically (AC-8: 16:9 + 21:9).
type Boundary struct {
	cam    *Isometric
	region Region
}

// NewBoundary creates a boundary clamp for the given camera and region.
func NewBoundary(cam *Isometric, region Region) *Boundary {
	return &Boundary{cam: cam, region: region}
}

// SetRegion replaces the playable bounds (called on region transition).
func (b *Boundary) SetRegion(region Region) { b.region = region }

// Region returns the current playable bounds.
func (b *Boundary) Region() Region { return b.region }

// Padding returns the clamp distance for the camera's current configuration,
// the GDD CR-6 formula (ground_width_max + ground_depth) / (2*sqrt(2)). At
// Default zoom, 16:9 this is 26.71 world units (half of the QA's 53.45
// rotated-rect AABB); at Far zoom, 21:9 it is 61.28.
func (b *Boundary) Padding() float32 {
	groundWidth := b.cam.Camera().Width() // 2*O*aspect
	groundDepth := 4 * b.cam.HalfHeight() // 4*O
	return (groundWidth + groundDepth) / (2 * float32(math.Sqrt2))
}

// ClampTarget returns target with its XZ clamped so the full viewport ground
// footprint stays inside the region — the camera stops at the boundary and no
// void space is visible (AC-8). A region narrower than twice the padding (a
// degenerate region) centers the camera rather than leaving it outside.
func (b *Boundary) ClampTarget(target matrix.Vec3) matrix.Vec3 {
	pad := b.Padding()
	x, z := target.X(), target.Z()
	if b.region.MaxX-b.region.MinX < 2*pad {
		x = (b.region.MinX + b.region.MaxX) / 2
	} else {
		x = clampf(x, b.region.MinX+pad, b.region.MaxX-pad)
	}
	if b.region.MaxZ-b.region.MinZ < 2*pad {
		z = (b.region.MinZ + b.region.MaxZ) / 2
	} else {
		z = clampf(z, b.region.MinZ+pad, b.region.MaxZ-pad)
	}
	return matrix.NewVec3(x, target.Y(), z)
}

// Footprint returns the exact world-space AABB of the ground visible through
// the camera: the four viewport corners projected along the view direction
// onto the horizontal plane at the focus elevation (the AABB of the
// 45°-rotated viewport rect per story QA AC-8). Used by tests to prove the
// clamped viewport shows no void, and as a debug readout.
func (b *Boundary) Footprint() AABB {
	cam := b.cam.Camera()
	planeY := cam.LookAt().Y()
	pos := cam.Position()
	fwd, right, up := cam.Forward(), cam.Right(), cam.Up()
	hw := cam.Width() / 2
	hh := cam.Height() / 2

	out := AABB{
		Min: matrix.NewVec3(matrix.FloatMax, 0, matrix.FloatMax),
		Max: matrix.NewVec3(-matrix.FloatMax, 0, -matrix.FloatMax),
	}
	for _, sx := range [2]float32{-1, 1} {
		for _, sy := range [2]float32{-1, 1} {
			corner := pos.
				Add(right.Scale(sx * hw)).
				Add(up.Scale(sy * hh))
			// fwd.Y() is -sin(30°) — never zero for the locked 30° pitch.
			t := (planeY - corner.Y()) / fwd.Y()
			hit := corner.Add(fwd.Scale(t))
			if hit.X() < out.Min.X() {
				out.Min.SetX(hit.X())
			}
			if hit.X() > out.Max.X() {
				out.Max.SetX(hit.X())
			}
			if hit.Z() < out.Min.Z() {
				out.Min.SetZ(hit.Z())
			}
			if hit.Z() > out.Max.Z() {
				out.Max.SetZ(hit.Z())
			}
		}
	}
	return out
}

// ── Edge scroll (AC-9) ──────────────────────────────────────────────────────

// DefaultEdgeScrollMargin is the screen-edge band (in pixels) that triggers
// edge scrolling. Story CAM-004 AC-9 requires edge scroll to never fire while
// the cursor is over UI; the band width itself is a UX tuning knob.
const DefaultEdgeScrollMargin float32 = 32

// EdgeScrollAllowed reports whether edge scroll may fire for a cursor
// position: the cursor must be inside the screen-edge band AND not over any
// UI rect (AC-9). uiRects are screen-space rects in Kaiju's (left, top,
// right, bottom) convention (the same convention as the UI system's scissor
// rects); a cursor inside any of them suppresses edge scroll while
// keyboard/drag/gamepad pan remains functional.
func EdgeScrollAllowed(cursor matrix.Vec2, viewport matrix.Vec4, uiRects []matrix.Vec4, marginPx float32) bool {
	if marginPx <= 0 {
		marginPx = DefaultEdgeScrollMargin
	}
	x, y := cursor.X(), cursor.Y()
	if x < 0 || x > viewport.Z() || y < 0 || y > viewport.W() {
		return false // cursor outside the viewport entirely
	}
	inEdgeBand := x <= marginPx || x >= viewport.Z()-marginPx ||
		y <= marginPx || y >= viewport.W()-marginPx
	if !inEdgeBand {
		return false
	}
	for _, r := range uiRects {
		if r.ScreenAreaContains(x, y) {
			return false
		}
	}
	return true
}

// clampf clamps v into [lo, hi].
func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

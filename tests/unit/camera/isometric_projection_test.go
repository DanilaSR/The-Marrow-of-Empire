// Unit tests for Story CAM-001: Isometric Projection Setup.
//
// Evidence path: production/epics/camera-system/story-001-isometric-projection.md
// Each test function maps to acceptance criteria and QA test cases from the story.
//
// These tests verify the camera wrapper against the Kaiju StandardCameraOrthographic
// — no mock needed; the Kaiju camera is pure math (no window/Vulkan dependency).
// Matrix dump verification (story QA AC-1) compares the camera's actual view and
// projection matrices against matrices built from the documented formulas.
package camera_test

import (
	"math"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/camera"
)

const (
	angleTolerance = 0.1 // ±0.1° per story QA AC-1
	floatTolerance = 0.001
)

// assertMat4Equal fails the test if any element of got differs from want.
func assertMat4Equal(t *testing.T, got, want matrix.Mat4) {
	t.Helper()
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > floatTolerance {
			t.Errorf("matrix element %d = %.6f, expected %.6f", i, got[i], want[i])
		}
	}
}

// ── AC-1: Projection matrix (30°/45°/0°, orthographic) ─────────────────────

func TestIsometric_New_IsOrthographic(t *testing.T) {
	// GIVEN the game starts, WHEN the first frame renders, THEN the camera
	// uses orthographic projection.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	if !cam.Camera().IsOrthographic() {
		t.Error("expected orthographic projection, got perspective")
	}
}

func TestIsometric_ForwardVector_CorrectPitch(t *testing.T) {
	// AC-1: pitch = 30° ± 0.1°. Pitch is the angle between the forward vector
	// and the horizontal (XZ) plane.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	fwd := cam.Camera().Forward()
	horizLen := math.Sqrt(float64(fwd.X()*fwd.X() + fwd.Z()*fwd.Z()))
	pitch := float32(math.Atan2(float64(-fwd.Y()), float64(horizLen)))
	pitchDeg := float64(matrix.Rad2Deg(pitch))

	expected := float64(cfg.Pitch)
	if math.Abs(pitchDeg-expected) > angleTolerance {
		t.Errorf("pitch = %.4f°, expected %.1f° ± %.1f°", pitchDeg, expected, angleTolerance)
	}
}

func TestIsometric_ForwardVector_CorrectYaw(t *testing.T) {
	// AC-1: yaw = 45° ± 0.1°. Yaw is the angle of the forward vector's XZ
	// projection relative to the cardinal axes.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	fwd := cam.Camera().Forward()
	yaw := float32(math.Atan2(float64(math.Abs(float64(fwd.X()))), float64(math.Abs(float64(fwd.Z())))))
	yawDeg := float64(matrix.Rad2Deg(yaw))

	expected := float64(cfg.Yaw)
	if math.Abs(yawDeg-expected) > angleTolerance {
		t.Errorf("yaw = %.4f°, expected %.1f° ± %.1f°", yawDeg, expected, angleTolerance)
	}
}

func TestIsometric_RightVector_Horizontal_NoRoll(t *testing.T) {
	// AC-1: roll = 0° ± 0.1°. A zero-roll camera has a horizontal right
	// vector (no Y component). This is how Mat4LookAt constructs the basis:
	// right = normalize(cross(worldUp, forward)), so right.Y() ≡ 0.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	right := cam.Camera().Right()
	if math.Abs(float64(right.Y())) > floatTolerance {
		t.Errorf("right.Y() = %.4f, expected 0 (camera rolled)", right.Y())
	}
}

func TestIsometric_ProjectionMatrix_MatchesOrthographicFormula(t *testing.T) {
	// AC-1 matrix dump: the projection matrix must be exactly the documented
	// orthographic matrix:
	//   Orthographic(-w/2, w/2, -h/2, h/2, -0.75*O, 1000)
	// with w = 2*O*AR, h = 2*O (16:9 viewport).
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	O := camera.ZoomDefault.HalfHeight // 10
	halfW := O * (1920.0 / 1080.0)     // O * aspect ratio
	want := matrix.Mat4Identity()
	want.Orthographic(-halfW, halfW, -O, O, -cfg.NearClipFactor*O, cfg.FarClip)

	assertMat4Equal(t, cam.Camera().Projection(), want)
}

func TestIsometric_ViewMatrix_MatchesLookAt(t *testing.T) {
	// AC-1 matrix dump: the view matrix must equal Mat4LookAt at the GDD
	// position, looking at the target with world up (zero roll).
	target := matrix.Vec3{10, 0, -5}
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(target, cfg, camera.ZoomDefault, 1920, 1080)

	want := matrix.Mat4LookAt(cam.Position(), target, matrix.Vec3Up())
	assertMat4Equal(t, cam.Camera().View(), want)
}

func TestIsometric_LookAt_IsTarget(t *testing.T) {
	target := matrix.Vec3{10, 0, -5}
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(target, cfg, camera.ZoomDefault, 1920, 1080)

	lookAt := cam.Camera().LookAt()
	if math.Abs(float64(lookAt.X()-target.X())) > floatTolerance ||
		math.Abs(float64(lookAt.Y()-target.Y())) > floatTolerance ||
		math.Abs(float64(lookAt.Z()-target.Z())) > floatTolerance {
		t.Errorf("lookAt = %v, expected %v", lookAt, target)
	}
}

// ── AC-2: Ground plane visible to viewport bottom edge (no void strip) ──────

func TestIsometric_GroundPlaneVisibleToViewportBottomEdge(t *testing.T) {
	// GIVEN the camera renders at any zoom register, THEN the ground plane
	// (world y = 0) crosses every viewport column within [near, far] — the
	// bottom edge of the viewport contains ground geometry, not void.
	//
	// Derived from GDD CR-1: near = -0.75 * O places the near plane just
	// behind the bottom-edge ground crossing (t = -0.732 * O), so the near
	// plane clips the underground region instead of the ground.
	cfg := camera.DefaultConfig()
	registers := []camera.ZoomLevel{
		camera.ZoomExtraClose, camera.ZoomClose, camera.ZoomDefault, camera.ZoomFar,
	}
	for _, zoom := range registers {
		t.Run(zoom.Name, func(t *testing.T) {
			cam := camera.NewIsometric(matrix.Vec3{}, cfg, zoom, 1920, 1080)

			pos := cam.Position()
			fwd := cam.Camera().Forward()
			up := cam.Camera().Up()
			O := cam.Camera().Height() / 2
			near := cam.Camera().NearPlane()
			far := cam.Camera().FarPlane()

			// Depth t along the view direction where the ground plane crosses
			// the viewport column at view-space height v (v = -O is the bottom
			// edge). Derived from: worldY(P + t*fwd + v*up) = 0.
			groundCrossing := func(v float32) float32 {
				return -(pos.Y() + v*up.Y()) / fwd.Y()
			}

			// 1. Every viewport column contains the ground surface: crossings
			//    at the bottom edge, center, and top edge are all within
			//    [near, far].
			for _, v := range []float32{-O, 0, O} {
				tc := groundCrossing(v)
				if tc < near-floatTolerance || tc > far+floatTolerance {
					t.Errorf("ground crossing at view height %.1f = %.4f, outside [%.4f, %.4f] — void strip",
						v, tc, near, far)
				}
			}

			// 2. The near plane intersects the ground at or below the viewport
			//    bottom edge — the underground region is clipped, not the
			//    ground surface.
			vGroundAtNear := -(pos.Y() + near*fwd.Y()) / up.Y()
			if vGroundAtNear > -O+floatTolerance {
				t.Errorf("ground at near plane = view y %.4f, above bottom edge %.4f — ground clipped",
					vGroundAtNear, -O)
			}
		})
	}
}

// ── AC-3: Aspect ratio resize preserves half-height ─────────────────────────

func TestIsometric_Resize_PreservesHalfHeight(t *testing.T) {
	// GIVEN 16:9 viewport, WHEN resized to 21:9, THEN ortho half-height
	// unchanged and horizontal extent increases proportionally.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	oldHeight := cam.Camera().Height()
	oldWidth := cam.Camera().Width()

	cam.Resize(2560, 1080) // 21:9

	newHeight := cam.Camera().Height()
	newWidth := cam.Camera().Width()

	if math.Abs(float64(newHeight-oldHeight)) > floatTolerance {
		t.Errorf("height changed: %.4f → %.4f, expected preserved", oldHeight, newHeight)
	}
	if newWidth <= oldWidth {
		t.Errorf("width did not increase: %.4f → %.4f", oldWidth, newWidth)
	}
	expectedRatio := (2560.0 / 1080.0) / (1920.0 / 1080.0)
	actualRatio := float64(newWidth / oldWidth)
	if math.Abs(actualRatio-expectedRatio) > 0.01 {
		t.Errorf("width ratio = %.4f, expected ~%.4f", actualRatio, expectedRatio)
	}
}

func TestIsometric_Resize_SquarePreservesHeight(t *testing.T) {
	// Resize to 1:1 (square) viewport — height still preserved.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	oldHeight := cam.Camera().Height()
	cam.Resize(1080, 1080)

	if math.Abs(float64(cam.Camera().Height()-oldHeight)) > floatTolerance {
		t.Errorf("height changed: %.4f → %.4f, expected preserved", oldHeight, cam.Camera().Height())
	}
}

// ── AC-4: Resize mid-zoom-transition re-applies from interpolated O ─────────

func TestIsometric_Resize_MidTransition_ExtentsFromInterpolatedO(t *testing.T) {
	// GIVEN a window resize during a zoom transition (O interpolated between
	// registers, e.g. 7.3), WHEN the resize completes, THEN ortho extents are
	// immediately re-applied from the current interpolated O — not from either
	// endpoint register.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	cam.SetHalfHeight(7.3) // current interpolated O mid-transition
	cam.Resize(2560, 1080) // 21:9 resize arrives

	O := float32(7.3)
	wantHeight := 2 * O
	if math.Abs(float64(cam.Camera().Height()-wantHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f (from interpolated O)", cam.Camera().Height(), wantHeight)
	}
	wantWidth := wantHeight * (2560.0 / 1080.0)
	if math.Abs(float64(cam.Camera().Width()-wantWidth)) > 0.01 {
		t.Errorf("width = %.4f, expected ~%.4f (from interpolated O × 21:9)", cam.Camera().Width(), wantWidth)
	}
	// The near clip is zoom-derived, so it re-applies from the interpolated O too.
	wantNear := -cfg.NearClipFactor * O
	if math.Abs(float64(cam.Camera().NearPlane()-wantNear)) > floatTolerance {
		t.Errorf("near = %.4f, expected %.4f (from interpolated O)", cam.Camera().NearPlane(), wantNear)
	}
}

// ── AC-5: Minimum aspect ratio 16:10 ────────────────────────────────────────

func TestIsometric_Resize_ClampsBelowMinAspect(t *testing.T) {
	// GIVEN 16:9 viewport, WHEN resized to narrower than 16:10,
	// THEN ortho extents are clamped at 16:10 minimum.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	oldHeight := cam.Camera().Height()
	cam.Resize(400, 300) // 4:3 = 1.33, below 16:10 = 1.6

	actualAspect := float64(cam.Camera().Width() / cam.Camera().Height())
	if math.Abs(float64(cam.Camera().Height()-oldHeight)) > floatTolerance {
		t.Errorf("height changed: %.4f → %.4f, expected preserved", oldHeight, cam.Camera().Height())
	}
	if actualAspect < float64(cfg.MinAspect)-0.001 {
		t.Errorf("aspect ratio %.4f below minimum %.4f", actualAspect, cfg.MinAspect)
	}
}

func TestIsometric_Resize_AtExactMinAspect(t *testing.T) {
	// At exactly 16:10, aspect is accepted without clamping.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	oldHeight := cam.Camera().Height()
	cam.Resize(1600, 1000) // exactly 16:10

	actualAspect := float64(cam.Camera().Width() / cam.Camera().Height())
	if math.Abs(actualAspect-1.6) > 0.001 {
		t.Errorf("aspect ratio = %.4f, expected 1.6", actualAspect)
	}
	if math.Abs(float64(cam.Camera().Height()-oldHeight)) > floatTolerance {
		t.Errorf("height changed at exact min aspect")
	}
}

func TestIsometric_Resize_AboveMinAspect_UsesActualAspect(t *testing.T) {
	// At 21:9, aspect is the actual ratio (not clamped to min).
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 2560, 1080)

	actualAspect := float64(cam.Camera().Width() / cam.Camera().Height())
	expectedAspect := 2560.0 / 1080.0

	if math.Abs(actualAspect-expectedAspect) > 0.01 {
		t.Errorf("aspect ratio = %.4f, expected ~%.4f (not clamped)", actualAspect, expectedAspect)
	}
}

// ── AC-6: Rotation is architecturally locked ────────────────────────────────

func TestIsometric_NoLookAtMutationExposed(t *testing.T) {
	// The Isometric type does not expose SetLookAt, SetPosition or
	// SetLookAtWithUp on its wrapped camera — enforced at compile time. The
	// test verifies the view direction stays stable under every operation the
	// Isometric type DOES expose (Resize, SetZoom).
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{5, 0, 5}, cfg, camera.ZoomDefault, 1920, 1080)

	fwdBefore := cam.Camera().Forward()

	cam.Resize(2560, 1440)
	cam.SetZoom(camera.ZoomClose)
	cam.Resize(800, 600)

	fwdAfter := cam.Camera().Forward()
	if math.Abs(float64(fwdBefore.X()-fwdAfter.X())) > floatTolerance ||
		math.Abs(float64(fwdBefore.Y()-fwdAfter.Y())) > floatTolerance ||
		math.Abs(float64(fwdBefore.Z()-fwdAfter.Z())) > floatTolerance {
		t.Errorf("forward changed after resize/zoom: %v → %v (rotation must be locked)", fwdBefore, fwdAfter)
	}
}

func TestIsometric_Resize_DoesNotMoveCamera(t *testing.T) {
	// Resize changes extents only — the camera position must not move.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{5, 0, 5}, cfg, camera.ZoomDefault, 1920, 1080)

	posBefore := cam.Position()
	cam.Resize(2560, 1440)
	cam.Resize(800, 600)
	posAfter := cam.Position()

	if math.Abs(float64(posBefore.X()-posAfter.X())) > floatTolerance ||
		math.Abs(float64(posBefore.Y()-posAfter.Y())) > floatTolerance ||
		math.Abs(float64(posBefore.Z()-posAfter.Z())) > floatTolerance {
		t.Errorf("camera moved on resize: %v → %v", posBefore, posAfter)
	}
}

// ── Clip planes (GDD CR-1: near = -0.75 × O, far = 1000) ────────────────────

func TestIsometric_NearClip_ZoomDerived(t *testing.T) {
	// Story QA edge cases: Extra Close -1.875, Close -3, Default -7.5, Far -15.
	cfg := camera.DefaultConfig()
	cases := []struct {
		zoom camera.ZoomLevel
		want float32
	}{
		{camera.ZoomExtraClose, -1.875},
		{camera.ZoomClose, -3},
		{camera.ZoomDefault, -7.5},
		{camera.ZoomFar, -15},
	}
	for _, tc := range cases {
		t.Run(tc.zoom.Name, func(t *testing.T) {
			cam := camera.NewIsometric(matrix.Vec3{}, cfg, tc.zoom, 1920, 1080)
			if math.Abs(float64(cam.Camera().NearPlane()-tc.want)) > floatTolerance {
				t.Errorf("near clip = %.4f, expected %.4f (-0.75 × O)", cam.Camera().NearPlane(), tc.want)
			}
		})
	}
}

func TestIsometric_FarClip_Always1000(t *testing.T) {
	cfg := camera.DefaultConfig()
	for _, zoom := range []camera.ZoomLevel{
		camera.ZoomExtraClose, camera.ZoomClose, camera.ZoomDefault, camera.ZoomFar,
	} {
		t.Run(zoom.Name, func(t *testing.T) {
			cam := camera.NewIsometric(matrix.Vec3{}, cfg, zoom, 1920, 1080)
			if math.Abs(float64(cam.Camera().FarPlane()-cfg.FarClip)) > floatTolerance {
				t.Errorf("far clip = %.4f, expected %.4f", cam.Camera().FarPlane(), cfg.FarClip)
			}
		})
	}
}

// ── Camera position (GDD formula) ───────────────────────────────────────────

func TestIsometric_Position_FollowsGDDFormula(t *testing.T) {
	// GDD Formulas — Camera Position from Focus:
	//   camera_x = focus_x - O*cos(30°)*sin(45°)
	//   camera_y = focus_y + O*sin(30°)          (= 0.5 × O above focus)
	//   camera_z = focus_z - O*cos(30°)*cos(45°)
	target := matrix.Vec3{10, 2, -5}
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(target, cfg, camera.ZoomDefault, 1920, 1080)

	O := float64(camera.ZoomDefault.HalfHeight)
	cos30 := math.Cos(math.Pi / 6)
	sin45 := math.Sin(math.Pi / 4)
	pos := cam.Position()

	wantX := float64(target.X()) - O*cos30*sin45
	wantY := float64(target.Y()) + O*math.Sin(math.Pi/6)
	wantZ := float64(target.Z()) - O*cos30*sin45

	if math.Abs(float64(pos.X())-wantX) > floatTolerance {
		t.Errorf("position.X = %.4f, expected %.4f", pos.X(), wantX)
	}
	if math.Abs(float64(pos.Y())-wantY) > floatTolerance {
		t.Errorf("position.Y = %.4f, expected %.4f (0.5 × O above focus)", pos.Y(), wantY)
	}
	if math.Abs(float64(pos.Z())-wantZ) > floatTolerance {
		t.Errorf("position.Z = %.4f, expected %.4f", pos.Z(), wantZ)
	}
}

// ── Zoom registers ──────────────────────────────────────────────────────────

func TestIsometric_ZoomDefault_CorrectExtents(t *testing.T) {
	// At Default zoom (O=10u) with 16:9 viewport, height = 20u, width = 20 × 16/9.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	expectedHeight := float32(20.0) // 2 × half-height
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
	expectedWidth := expectedHeight * (1920.0 / 1080.0)
	if math.Abs(float64(cam.Camera().Width()-expectedWidth)) > 0.01 {
		t.Errorf("width = %.4f, expected ~%.4f", cam.Camera().Width(), expectedWidth)
	}
}

func TestIsometric_ZoomClose_CorrectExtents(t *testing.T) {
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomClose, 1920, 1080)

	expectedHeight := float32(8.0) // 2 × 4
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
}

func TestIsometric_ZoomFar_CorrectExtents(t *testing.T) {
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomFar, 1920, 1080)

	expectedHeight := float32(40.0) // 2 × 20
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
}

func TestIsometric_ZoomExtraClose_CorrectExtents(t *testing.T) {
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomExtraClose, 1920, 1080)

	expectedHeight := float32(5.0) // 2 × 2.5
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
}

func TestIsometric_ZoomSet_UpdatesExtentsAndRegister(t *testing.T) {
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	cam.SetZoom(camera.ZoomClose)

	expectedHeight := float32(8.0) // 2 × 4
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("after SetZoom: height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %v, expected %v", cam.Zoom(), camera.ZoomClose)
	}
	if cam.HalfHeight() != camera.ZoomClose.HalfHeight {
		t.Errorf("HalfHeight() = %.2f, expected %.2f", cam.HalfHeight(), camera.ZoomClose.HalfHeight)
	}
}

func TestIsometric_SetHalfHeight_UpdatesExtentsAndPosition(t *testing.T) {
	// SetHalfHeight is the transition hook (CAM-002): O, position and extents
	// all re-apply from the given half-height.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	cam.SetHalfHeight(4) // equivalent to Close

	expectedHeight := float32(8.0) // 2 × 4
	if math.Abs(float64(cam.Camera().Height()-expectedHeight)) > floatTolerance {
		t.Errorf("height = %.4f, expected %.4f", cam.Camera().Height(), expectedHeight)
	}
	wantY := float32(2) // 0.5 × O above focus
	if math.Abs(float64(cam.Position().Y()-wantY)) > floatTolerance {
		t.Errorf("position.Y = %.4f, expected %.4f (0.5 × O)", cam.Position().Y(), wantY)
	}
	wantNear := float32(-3) // -0.75 × 4
	if math.Abs(float64(cam.Camera().NearPlane()-wantNear)) > floatTolerance {
		t.Errorf("near = %.4f, expected %.4f", cam.Camera().NearPlane(), wantNear)
	}
}

func TestIsometric_SetHalfHeight_NonPositive_Ignored(t *testing.T) {
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	before := cam.Camera().Height()
	cam.SetHalfHeight(0)
	cam.SetHalfHeight(-3)

	if cam.Camera().Height() != before {
		t.Errorf("height changed to %.4f after non-positive SetHalfHeight", cam.Camera().Height())
	}
	if cam.HalfHeight() != camera.ZoomDefault.HalfHeight {
		t.Errorf("HalfHeight() = %.2f, expected %.2f", cam.HalfHeight(), camera.ZoomDefault.HalfHeight)
	}
}

// ── Config ──────────────────────────────────────────────────────────────────

func TestDefaultConfig_MatchesADR0003(t *testing.T) {
	cfg := camera.DefaultConfig()
	if cfg.Pitch != 30 || cfg.Yaw != 45 {
		t.Errorf("angles = %v°/%v°, expected 30/45", cfg.Pitch, cfg.Yaw)
	}
	if cfg.NearClipFactor != 0.75 {
		t.Errorf("NearClipFactor = %v, expected 0.75", cfg.NearClipFactor)
	}
	if cfg.FarClip != 1000 {
		t.Errorf("FarClip = %v, expected 1000", cfg.FarClip)
	}
	if math.Abs(float64(cfg.MinAspect-1.6)) > floatTolerance {
		t.Errorf("MinAspect = %v, expected 1.6 (16:10)", cfg.MinAspect)
	}
}

func TestIsometric_Config_ReturnsProvidedConfig(t *testing.T) {
	cfg := camera.Config{
		Pitch:          30,
		Yaw:            45,
		NearClipFactor: 0.75,
		FarClip:        1000,
		MinAspect:      1.6,
	}
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	got := cam.Config()
	if got != cfg {
		t.Errorf("Config() = %v, expected %v", got, cfg)
	}
}

// ── Edge cases ──────────────────────────────────────────────────────────────

func TestIsometric_TargetOffset_ViewFollowsTarget(t *testing.T) {
	// A camera at a non-origin target still has correct isometric angles.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{100, 0, -200}, cfg, camera.ZoomDefault, 1920, 1080)

	fwd := cam.Camera().Forward()
	horizLen := math.Sqrt(float64(fwd.X()*fwd.X() + fwd.Z()*fwd.Z()))
	pitch := float32(math.Atan2(float64(-fwd.Y()), float64(horizLen)))
	pitchDeg := float64(matrix.Rad2Deg(pitch))

	if math.Abs(pitchDeg-float64(cfg.Pitch)) > angleTolerance {
		t.Errorf("pitch at offset target = %.4f°, expected %.1f°", pitchDeg, cfg.Pitch)
	}
}

func TestIsometric_Resize_ZeroViewport_KeepsLastExtents(t *testing.T) {
	// GDD edge case: minimized window (0-size viewport) keeps last extents;
	// ortho extents re-applied on restore.
	cfg := camera.DefaultConfig()
	cam := camera.NewIsometric(matrix.Vec3{}, cfg, camera.ZoomDefault, 1920, 1080)

	oldWidth := cam.Camera().Width()
	oldHeight := cam.Camera().Height()

	cam.Resize(0, 0) // must not panic or corrupt extents

	if cam.Camera().Width() != oldWidth || cam.Camera().Height() != oldHeight {
		t.Errorf("extents changed on zero viewport: %.2f×%.2f → %.2f×%.2f",
			oldWidth, oldHeight, cam.Camera().Width(), cam.Camera().Height())
	}
}

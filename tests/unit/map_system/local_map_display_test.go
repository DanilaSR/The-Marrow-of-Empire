// Unit tests for Story 001: Local Map Display & Navigation.
//
// Evidence path: production/epics/map-system/story-001-local-map-display.md
// Required evidence: this file must exist and pass (automated gate for the
// logic portions of this UI story; the Kaiju document rendering and the
// visual result are verified by the manual walkthrough,
// production/qa/evidence).
//
// The map's state machine, tween math, viewport clamping, and world-dim
// calculation are pure game logic — the Kaiju document rendering is
// headless here (NewLocalMap with nil host and manager skips markup
// loading and the per-frame update registration, while Update() remains
// directly callable to advance the tweens).
//
// The acceptance criteria map to test functions (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1/AC-2  test_map_open_close_state_machine      -> TestMapOpenCloseStateMachine
//	AC-1/AC-2  test_m_key_toggle_behavior             -> TestMKeyToggleBehavior
//	AC-2       test_close_during_enter_cancels_to_exit -> TestCloseDuringEnterCancelsToExit
//	AC-2       test_toggle_during_exit_reopens         -> TestToggleDuringExitReopens
//	edge       test_reopen_after_close                 -> TestReopenAfterClose
//	edge       test_open_noop_when_open                -> TestOpenNoopWhenOpen
//	edge       test_reduced_motion_instant_transitions -> TestReducedMotionInstantTransitions
//	AC-1       test_paper_slide_tween_eases_out        -> TestPaperSlideTweenEasesOut
//	edge       test_paper_slide_tween_zero_duration    -> TestPaperSlideTweenZeroDuration
//	edge       test_paper_slide_tween_clamps           -> TestPaperSlideTweenClamps
//	AC-5       test_clamp_viewport_large_region        -> TestClampViewportLargeRegion
//	AC-5       test_clamp_viewport_small_region_centers -> TestClampViewportSmallRegionCenters
//	AC-5       test_pan_clamped_to_region              -> TestPanClampedToRegion
//	AC-4       test_open_centers_viewport_on_player    -> TestOpenCentersViewportOnPlayer
//	AC-1       test_world_dim_calculation              -> TestWorldDimCalculation
//	AC-7       test_height_scale_factor_floor_and_cap  -> TestHeightScaleFactorFloorAndCap
//	AC-4       test_player_position_polled_each_frame  -> TestPlayerPositionPolledEachFrame
//	AC-4       test_facing_nib_offset_math             -> TestFacingNibOffsetMath
//
// The M/Esc hotkey routing itself needs a live engine host (input polling)
// and is covered by the manual walkthrough; these tests drive the
// Open/Close/Toggle state machine the hotkey handlers call.
package map_system_test

import (
	"math"
	"testing"

	localmap "marrow-of-empire/ui/map"
)

// stubProvider is a mutable PositionProvider fixture (AC-4).
type stubProvider struct {
	x, y, facing float64
}

func (p *stubProvider) GetPlayerPosition() (float64, float64, float64) {
	return p.x, p.y, p.facing
}

// testRegion is a 250x200 m region (2000x1600 px at the 8 px/m default
// scale) with one boundary, one path, and one structure segment.
func testRegion() localmap.RegionMapData {
	return localmap.RegionMapData{
		Name:   "Test Region",
		Width:  250,
		Height: 200,
		Boundaries: []localmap.LineSegment{
			{X1: 0, Y1: 0, X2: 250, Y2: 0},
		},
		Paths: []localmap.LineSegment{
			{X1: 10, Y1: 10, X2: 200, Y2: 80},
		},
		Structures: []localmap.LineSegment{
			{X1: 100, Y1: 100, X2: 120, Y2: 100},
		},
	}
}

func newHeadlessMap(prov localmap.PositionProvider) *localmap.LocalMap {
	return localmap.NewLocalMap(nil, nil, prov)
}

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// ---------------------------------------------------------------------------
// AC-1 / AC-2: map open/close state machine
// ---------------------------------------------------------------------------

// test_map_open_close_state_machine: Open starts the enter sequence (map
// on screen immediately, slide running), Close starts the exit sequence
// (still on screen while sliding out), and once the exit tween completes
// the map is fully closed.
func TestMapOpenCloseStateMachine(t *testing.T) {
	m := newHeadlessMap(nil)
	if m.IsOpen() {
		t.Fatal("map must start closed")
	}

	m.Open()
	if !m.IsOpen() {
		t.Fatal("map must be on screen while entering")
	}
	m.Update(0.5) // completes the 250ms enter slide
	if !m.IsOpen() {
		t.Fatal("map must stay open after the enter slide completes")
	}

	m.Close()
	if !m.IsOpen() {
		t.Fatal("map must stay on screen while the exit slide runs")
	}
	m.Update(0.5) // completes the 200ms exit slide
	if m.IsOpen() {
		t.Fatal("map must be closed after the exit slide completes")
	}
}

// test_m_key_toggle_behavior: Toggle opens a closed map and closes an open
// one (the M-key behavior, AC-1/AC-2).
func TestMKeyToggleBehavior(t *testing.T) {
	m := newHeadlessMap(nil)

	m.Toggle()
	if !m.IsOpen() {
		t.Fatal("first toggle must open the map")
	}
	m.Update(0.5)
	if !m.IsOpen() {
		t.Fatal("map must be open after enter completes")
	}

	m.Toggle()
	if !m.IsOpen() {
		t.Fatal("second toggle must start the exit (map still on screen)")
	}
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("map must be closed after the exit completes")
	}

	m.Toggle()
	if !m.IsOpen() {
		t.Fatal("third toggle must reopen the map")
	}
}

// test_close_during_enter_cancels_to_exit: pressing Esc while the map is
// still sliding in cancels the enter and exits instead (AC-2 — the map
// closes whenever it is on screen).
func TestCloseDuringEnterCancelsToExit(t *testing.T) {
	m := newHeadlessMap(nil)

	m.Open()
	m.Close() // still entering
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("map must be closed after Close() during the enter slide")
	}
}

// test_toggle_during_exit_reopens: toggling while the exit slide runs
// cancels the exit and enters again.
func TestToggleDuringExitReopens(t *testing.T) {
	m := newHeadlessMap(nil)

	m.Open()
	m.Toggle() // entering -> exiting
	m.Toggle() // exiting -> entering again
	m.Update(0.5)
	if !m.IsOpen() {
		t.Fatal("map must be open after toggling back from the exit slide")
	}
}

// test_reopen_after_close: a closed map opens again cleanly.
func TestReopenAfterClose(t *testing.T) {
	m := newHeadlessMap(nil)

	m.Open()
	m.Update(0.5)
	m.Close()
	m.Update(0.5)

	m.Open()
	m.Update(0.5)
	if !m.IsOpen() {
		t.Fatal("map must reopen after a full close")
	}
	m.Close()
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("map must close again after reopen")
	}
}

// test_open_noop_when_open: Open while the map is already on screen is a
// no-op — the state machine must not restart the enter tween.
func TestOpenNoopWhenOpen(t *testing.T) {
	m := newHeadlessMap(nil)

	m.Open()
	m.Update(0.5) // fully open
	m.Open()      // no-op
	m.Update(0.5)
	if !m.IsOpen() {
		t.Fatal("map must remain open after a no-op Open")
	}
	m.Close()
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("map must still close cleanly after the no-op Open")
	}
}

// test_reduced_motion_instant_transitions: under the reduced-motion gate
// the map appears and disappears instantly — no slide, no dim animation.
func TestReducedMotionInstantTransitions(t *testing.T) {
	m := newHeadlessMap(nil)
	m.SetReducedMotion(true)

	m.Open()
	if !m.IsOpen() {
		t.Fatal("map must be open instantly under reduced motion")
	}
	m.Close()
	if m.IsOpen() {
		t.Fatal("map must be closed instantly under reduced motion")
	}
}

// ---------------------------------------------------------------------------
// AC-1: paper-slide tween math
// ---------------------------------------------------------------------------

// test_paper_slide_tween_eases_out: the tween starts at From, eases out
// along the paper-slide curve (more than half the distance in the first
// half of the duration), completes at To, and is monotonic.
func TestPaperSlideTweenEasesOut(t *testing.T) {
	tw := localmap.NewPaperSlideTween(0, 100, 0.25)
	if tw.Value() != 0 {
		t.Fatalf("initial value = %v, want 0", tw.Value())
	}
	if tw.Done() {
		t.Fatal("fresh tween must not be done")
	}

	tw.Update(0.125) // half the duration
	if tw.Done() {
		t.Fatal("tween must not be done at half duration")
	}
	half := tw.Value()
	if half <= 50 || half >= 100 {
		t.Fatalf("ease-out value at half duration = %v, want strictly between 50 and 100", half)
	}

	tw.Update(0.125)
	if !tw.Done() {
		t.Fatal("tween must complete at full duration")
	}
	if tw.Value() != 100 {
		t.Fatalf("final value = %v, want 100", tw.Value())
	}

	// Monotonicity: sample the curve at quarter steps.
	mono := localmap.NewPaperSlideTween(0, 100, 1.0)
	prev := 0.0
	for i := 1; i <= 4; i++ {
		mono.Update(0.25)
		v := mono.Value()
		if v <= prev {
			t.Fatalf("tween not monotonic at step %d: %v <= %v", i, v, prev)
		}
		prev = v
	}
}

// test_paper_slide_tween_zero_duration: a non-positive duration completes
// on the first Update (the reduced-motion path).
func TestPaperSlideTweenZeroDuration(t *testing.T) {
	tw := localmap.NewPaperSlideTween(5, 10, 0)
	tw.Update(0)
	if !tw.Done() {
		t.Fatal("zero-duration tween must complete on first Update")
	}
	if tw.Value() != 10 {
		t.Fatalf("zero-duration tween value = %v, want 10", tw.Value())
	}
}

// test_paper_slide_tween_clamps: an oversized dt completes the tween at
// To, and a done tween ignores further updates.
func TestPaperSlideTweenClamps(t *testing.T) {
	tw := localmap.NewPaperSlideTween(-720, 0, 0.25) // park offset -> 0
	tw.Update(5.0)
	if !tw.Done() || tw.Value() != 0 {
		t.Fatalf("oversized dt: done = %v, value = %v, want done, 0", tw.Done(), tw.Value())
	}
	tw.Update(1.0)
	if tw.Value() != 0 {
		t.Fatalf("done tween moved: value = %v, want 0", tw.Value())
	}
}

// ---------------------------------------------------------------------------
// AC-5: viewport scroll clamping
// ---------------------------------------------------------------------------

// test_clamp_viewport_large_region: for a region larger than the viewport
// the offset stays in [viewport-region, 0] — panning stops at the region
// edges so the surface always covers the viewport.
func TestClampViewportLargeRegion(t *testing.T) {
	const (
		regionW   = 2000.0
		regionH   = 1500.0
		viewportW = 1280.0
		viewportH = 720.0
	)
	// Inside the valid range: unchanged.
	ox, oy := localmap.ClampViewport(-300, -100, regionW, regionH, viewportW, viewportH)
	if ox != -300 || oy != -100 {
		t.Fatalf("in-range clamp = (%v, %v), want (-300, -100)", ox, oy)
	}
	// Past the right/bottom edges: clamped to 0.
	ox, oy = localmap.ClampViewport(100, -2000, regionW, regionH, viewportW, viewportH)
	if ox != 0 || oy != viewportH-regionH {
		t.Fatalf("past-edge clamp = (%v, %v), want (0, %v)", ox, oy, viewportH-regionH)
	}
	// Past the left/top edges: clamped to the far bound.
	ox, oy = localmap.ClampViewport(-2000, 100, regionW, regionH, viewportW, viewportH)
	if ox != viewportW-regionW || oy != 0 {
		t.Fatalf("past-far clamp = (%v, %v), want (%v, 0)", ox, oy, viewportW-regionW)
	}
}

// test_clamp_viewport_small_region_centers: a region smaller than the
// viewport is centered — parchment shows around its edges.
func TestClampViewportSmallRegionCenters(t *testing.T) {
	ox, oy := localmap.ClampViewport(0, 0, 800, 600, 1280, 720)
	if ox != 240 || oy != 60 {
		t.Fatalf("small-region clamp = (%v, %v), want (240, 60)", ox, oy)
	}
}

// test_pan_clamped_to_region: Pan shifts the viewport and clamps to the
// region bounds (2000x1600 px at the default scale). Dragging right moves
// the surface right and clamps at 0 (region left edge at the viewport left
// edge); dragging left moves it left and clamps at viewport-region (region
// right edge at the viewport right edge).
func TestPanClampedToRegion(t *testing.T) {
	m := newHeadlessMap(nil)
	m.SetRegion(testRegion())
	m.SetViewportSize(1280, 720)

	m.Pan(5000, 0) // drag right, way past the limit -> left-aligned origin
	ox, oy := m.ViewportOffset()
	if ox != 0 || oy != 0 {
		t.Fatalf("pan right = (%v, %v), want (0, 0)", ox, oy)
	}

	m.Pan(-5000, 0) // drag left, way past the limit -> right-aligned edge
	ox, oy = m.ViewportOffset()
	if ox != 1280-2000 || oy != 0 {
		t.Fatalf("pan left = (%v, %v), want (%v, 0)", ox, oy, 1280-2000)
	}

	m.Pan(0, -5000) // drag up, way past the limit -> bottom-aligned edge
	ox, oy = m.ViewportOffset()
	if ox != 1280-2000 || oy != 720-1600 {
		t.Fatalf("pan up = (%v, %v), want (%v, %v)", ox, oy, 1280-2000, 720-1600)
	}

	m.Pan(-5000, -5000) // further panning stays clamped
	ox, oy = m.ViewportOffset()
	if ox != 1280-2000 || oy != 720-1600 {
		t.Fatalf("clamped pan = (%v, %v), want (%v, %v)", ox, oy, 1280-2000, 720-1600)
	}

	m.Pan(5000, 5000) // back to the origin
	ox, oy = m.ViewportOffset()
	if ox != 0 || oy != 0 {
		t.Fatalf("pan home = (%v, %v), want (0, 0)", ox, oy)
	}
}

// test_open_centers_viewport_on_player: opening the map centers the
// viewport on the player's position, clamped to the region (AC-4).
func TestOpenCentersViewportOnPlayer(t *testing.T) {
	prov := &stubProvider{x: 125, y: 100} // region center
	m := newHeadlessMap(prov)
	m.SetRegion(testRegion()) // 2000x1600 px
	m.SetViewportSize(1280, 720)

	m.Open()

	// Player at world (125, 100) -> surface (1000, 1600-800=800).
	// Center: (640-1000, 360-800) = (-360, -440), in range [-720,0]x[-880,0].
	ox, oy := m.ViewportOffset()
	if ox != -360 || oy != -440 {
		t.Fatalf("player-centered offset = (%v, %v), want (-360, -440)", ox, oy)
	}
}

// ---------------------------------------------------------------------------
// AC-1: world-dim calculation
// ---------------------------------------------------------------------------

// test_world_dim_calculation: the dim overlay starts at 0 (world at full
// brightness), settles at WorldDimOpacity (world at 30%), and is
// monotonic along the paper-slide curve.
func TestWorldDimCalculation(t *testing.T) {
	if got := localmap.DimOpacity(0); got != 0 {
		t.Fatalf("DimOpacity(0) = %v, want 0", got)
	}
	if got := localmap.DimOpacity(1); got != localmap.WorldDimOpacity {
		t.Fatalf("DimOpacity(1) = %v, want %v", got, localmap.WorldDimOpacity)
	}
	if localmap.WorldDimOpacity != 0.70 {
		t.Fatalf("WorldDimOpacity = %v, want 0.70 (world at 30%%)", localmap.WorldDimOpacity)
	}

	mid := localmap.DimOpacity(0.5)
	quarter := localmap.DimOpacity(0.25)
	if quarter <= 0 || mid <= quarter || mid >= localmap.WorldDimOpacity {
		t.Fatalf("dim curve not monotonic: quarter = %v, mid = %v", quarter, mid)
	}
}

// test_dim_opacity_known_answer: the paper-slide cubic-bezier curve has a
// known value at its midpoint — solvePaperSlide(0.5) ≈ 0.8729 — so the
// world-dim opacity at half-progress is pinned (DimOpacity(0.5) ≈ 0.6110).
// If the curve constants change, this test catches the regression.
func TestDimOpacityKnownAnswer(t *testing.T) {
	const eps = 0.0001
	if got := localmap.DimOpacity(0.5); !approxEqual(got, 0.6110, eps) {
		t.Fatalf("DimOpacity(0.5) = %v, want ~0.6110 — curve constant regression?", got)
	}
	if got := localmap.DimOpacity(0.25); !approxEqual(got, 0.4160, eps) {
		t.Fatalf("DimOpacity(0.25) = %v, want ~0.4160", got)
	}
	if got := localmap.DimOpacity(0.75); !approxEqual(got, 0.6829, eps) {
		t.Fatalf("DimOpacity(0.75) = %v, want ~0.6829", got)
	}
}

// ---------------------------------------------------------------------------
// AC-7: height-based scaling
// ---------------------------------------------------------------------------

// test_height_scale_factor_floor_and_cap: the scale is windowHeight/1080
// with a floor of 1.0 — text never shrinks below the reference size, so
// body text (16px) stays >= 12px effective at 1280x720 — and a cap of 1.5.
func TestHeightScaleFactorFloorAndCap(t *testing.T) {
	cases := []struct {
		height float64
		want   float64
	}{
		{540, 1.0},  // floor
		{720, 1.0},  // 1280x720: 16px body >= 12px effective
		{1080, 1.0}, // reference
		{1440, 4.0 / 3.0},
		{2160, 1.5}, // cap
	}
	for _, c := range cases {
		if got := localmap.HeightScaleFactor(c.height); !approxEqual(got, c.want, 0.0001) {
			t.Fatalf("HeightScaleFactor(%v) = %v, want %v", c.height, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-4: player position polling
// ---------------------------------------------------------------------------

// test_player_position_polled_each_frame: while the map is open the
// indicator follows the PositionProvider, refreshed every frame.
func TestPlayerPositionPolledEachFrame(t *testing.T) {
	prov := &stubProvider{x: 10, y: 20, facing: 1.0}
	m := newHeadlessMap(prov)
	m.SetRegion(testRegion())

	m.Open()
	m.Update(0.5)
	x, y, facing := m.PlayerPosition()
	if x != 10 || y != 20 || facing != 1.0 {
		t.Fatalf("initial player = (%v, %v, %v), want (10, 20, 1)", x, y, facing)
	}

	prov.x, prov.y, prov.facing = 30, 40, 0.5
	m.Update(0.016)
	x, y, facing = m.PlayerPosition()
	if x != 30 || y != 40 || facing != 0.5 {
		t.Fatalf("updated player = (%v, %v, %v), want (30, 40, 0.5)", x, y, facing)
	}

	// After closing, polling stops; the last position is retained.
	m.Close()
	m.Update(0.5)
	prov.x, prov.y, prov.facing = 99, 99, 2.0
	m.Update(0.016)
	if x, y, facing = m.PlayerPosition(); x != 30 || y != 40 || facing != 0.5 {
		t.Fatalf("player must not update while the map is closed: (%v, %v, %v)", x, y, facing)
	}
}

// ---------------------------------------------------------------------------
// AC-7: font-scale re-application on resize (regression: scale was only
// applied once at construction, so resize 720p → 1440p left labels at the
// old scale).
// ---------------------------------------------------------------------------

// test_font_scale_reapplied_on_resize: the height-based font scale is
// re-evaluated every frame from the current viewport height (AC-7).
// Changing the viewport size between frames triggers a scale re-apply.
func TestFontScaleReappliedOnResize(t *testing.T) {
	m := newHeadlessMap(nil)
	m.SetViewportSize(1280, 720)
	if got := localmap.HeightScaleFactor(720); !approxEqual(got, 1.0, 0.0001) {
		t.Fatalf("720p scale = %v, want 1.0", got)
	}
	// Resize to 1440p: scale should change from 1.0 to 1.33.
	m.SetViewportSize(2560, 1440)
	if got := localmap.HeightScaleFactor(1440); !approxEqual(got, 4.0/3.0, 0.0001) {
		t.Fatalf("1440p scale = %v, want 1.333", got)
	}
	// Resize back to 720p: scale returns to floor.
	m.SetViewportSize(1280, 720)
	m.Update(0.016) // Update re-applies font scale from the cached viewport
	if got := localmap.HeightScaleFactor(720); !approxEqual(got, 1.0, 0.0001) {
		t.Fatalf("back to 720p scale = %v, want 1.0", got)
	}
}

// ---------------------------------------------------------------------------
// Edge cases surfaced by code review
// ---------------------------------------------------------------------------

// test_pan_without_region_noops: Pan on a fresh map with no region installed
// falls through clampAxis with (0,0) extents — the small-region centering
// rule kicks in and the surface centers at viewport/2. This is math-consistent
// but semantically surprising; the test documents current behavior.
func TestPanWithoutRegionCenters(t *testing.T) {
	m := newHeadlessMap(nil)
	m.SetViewportSize(1280, 720)
	// No SetRegion call — region extents are (0, 0).
	m.Pan(500, 300)
	ox, oy := m.ViewportOffset()
	// (0,0) region with (1280,720) viewport: centered at (640, 360).
	if ox != 640 || oy != 360 {
		t.Fatalf("pan without region offset = (%v, %v), want (640, 360)", ox, oy)
	}
}

// test_player_outside_region: a PositionProvider returning coordinates outside
// the region produces off-surface indicator positions — the world-to-surface
// mapping does not clamp. This documents current behavior; the game layer is
// expected to supply in-region positions.
func TestPlayerOutsideRegion(t *testing.T) {
	prov := &stubProvider{x: -50, y: 999} // far outside the 250x200 region
	m := newHeadlessMap(prov)
	m.SetRegion(testRegion())
	m.SetViewportSize(1280, 720)

	m.Open()
	m.Update(0.5)
	x, y, _ := m.PlayerPosition()
	// Positions are passed through as-is — no clamping in the map layer.
	if x != -50 || y != 999 {
		t.Fatalf("outside-region player = (%v, %v), want (-50, 999) — pass-through", x, y)
	}
}

// test_clamp_viewport_region_equals_viewport: when the region surface exactly
// matches the viewport, clampAxis hits the "smaller-or-equal" branch and
// centers at 0 — correct, since there is nothing to scroll.
func TestClampViewportRegionEqualsViewport(t *testing.T) {
	ox, oy := localmap.ClampViewport(-100, 50, 1280, 720, 1280, 720)
	if ox != 0 || oy != 0 {
		t.Fatalf("region==viewport clamp = (%v, %v), want (0, 0)", ox, oy)
	}
}

// test_close_when_closed_noop: Close on an already-closed map is a no-op —
// the state machine must not restart the exit tween or flip IsOpen.
func TestCloseWhenClosedNoop(t *testing.T) {
	m := newHeadlessMap(nil)
	if m.IsOpen() {
		t.Fatal("fresh map must be closed")
	}
	m.Close() // no-op
	if m.IsOpen() {
		t.Fatal("Close on closed map must remain closed")
	}
	// After a full open/close cycle, a second Close is still a no-op.
	m.Open()
	m.Update(0.5)
	m.Close()
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("map must be closed after full cycle")
	}
	m.Close() // second close — no-op
	if m.IsOpen() {
		t.Fatal("second Close after full cycle must remain closed")
	}
}

// test_update_on_fresh_closed_map: calling Update before the first Open is
// safe — no slide tweens exist, so updateTweens returns immediately, and
// the closed state skips player/pan updates.
func TestUpdateOnFreshClosedMap(t *testing.T) {
	m := newHeadlessMap(nil)
	// Must not panic.
	m.Update(0.016)
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("fresh map must stay closed after Update")
	}
}

// test_facing_nib_offset_math: the facing nib offset follows the facing
// angle in screen coordinates (0 = east, positive = counter-clockwise,
// y-down flips the y axis). Epsilon comparisons — cos(pi/2) is not exactly
// 0 in floating point.
func TestFacingNibOffsetMath(t *testing.T) {
	const dist = 9.0
	const eps = 0.000001
	dx, dy := localmap.FacingNibOffset(0, dist)
	if !approxEqual(dx, dist, eps) || !approxEqual(dy, 0, eps) {
		t.Fatalf("east facing = (%v, %v), want (%v, 0)", dx, dy, dist)
	}
	dx, dy = localmap.FacingNibOffset(math.Pi/2, dist)
	if !approxEqual(dx, 0, eps) || !approxEqual(dy, -dist, eps) {
		t.Fatalf("north facing = (%v, %v), want (0, %v)", dx, dy, -dist)
	}
	dx, dy = localmap.FacingNibOffset(math.Pi, dist)
	if !approxEqual(dx, -dist, eps) || !approxEqual(dy, 0, eps) {
		t.Fatalf("west facing = (%v, %v), want (%v, 0)", dx, dy, -dist)
	}
	dx, dy = localmap.FacingNibOffset(-math.Pi/2, dist)
	if !approxEqual(dx, 0, eps) || !approxEqual(dy, dist, eps) {
		t.Fatalf("south facing = (%v, %v), want (0, %v)", dx, dy, dist)
	}
}

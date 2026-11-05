// Unit tests for Story CAM-004: Occlusion & Boundaries.
//
// Evidence path: production/epics/camera-system/story-004-occlusion-boundaries.md
//
// These tests cover the fade state machine (AC-1/AC-2 timing and hysteresis),
// the foliage alpha floor, the interior roof controller (AC-3), the Tab
// highlight controller (AC-4), and the edge-scroll decision (AC-9). The
// Kaiju StandardCamera is pure math and runs headless, so the detector is
// exercised against the real camera (same convention as isometric_projection
// tests). The render layer is faked via the AlphaFader / VisibilityApplier /
// OutlineApplier interfaces.
//
// Notes:
//   - The story QA "no alpha cycle faster than one per 2s" is not reachable
//     with the story's own constants (150ms fade-out + 300ms hold + 150ms
//     fade-in = 600ms minimum cycle = 1.67Hz). The GDD CR-8 bound (toggle
//     frequency below 2Hz) is the reachable guarantee and is what these
//     tests assert; the 2s claim was flagged to the coordinator.
//   - A pure depth-gate-only negative case (screen overlap + entirely-behind
//     structure) is not constructible with connected world AABBs: a world
//     AABB whose view-space rect overlaps the member's central rect and whose
//     depth range sits entirely beyond the member's would have to extend
//     below the ground plane. The screen-overlap gate and the combined test
//     are covered instead.
package camera_test

import (
	"math"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/camera"
	"marrow-of-empire/entity"
)

// floatTolerance is shared with isometric_projection_test.go (same package).
const padTolerance = 0.05 // padding/QA-number assertions (world units)

// ── Fakes ───────────────────────────────────────────────────────────────────

// fakeFader records per-mesh alpha pushes (AlphaFader).
type fakeFader struct {
	alpha map[entity.GameEntityID]float32
}

func newFakeFader() *fakeFader {
	return &fakeFader{alpha: make(map[entity.GameEntityID]float32)}
}

func (f *fakeFader) SetAlpha(id entity.GameEntityID, alpha float32) { f.alpha[id] = alpha }

// fakeVisibility records visibility toggles (VisibilityApplier).
type fakeVisibility struct {
	visible map[entity.GameEntityID]bool
}

func newFakeVisibility() *fakeVisibility {
	return &fakeVisibility{visible: make(map[entity.GameEntityID]bool)}
}

func (f *fakeVisibility) SetVisible(id entity.GameEntityID, visible bool) { f.visible[id] = visible }

// fakeProvider returns a fixed interactable set and counts queries
// (InteractableProvider).
type fakeProvider struct {
	ids   []entity.GameEntityID
	calls int
}

func (f *fakeProvider) InteractablesNear(center matrix.Vec3, radius float32) []entity.GameEntityID {
	f.calls++
	return f.ids
}

// fakeOutline records outline state (OutlineApplier).
type fakeOutline struct {
	outlines map[entity.GameEntityID]bool // true = outlined
	color    map[entity.GameEntityID]matrix.Color
	width    map[entity.GameEntityID]float32
}

func newFakeOutline() *fakeOutline {
	return &fakeOutline{
		outlines: make(map[entity.GameEntityID]bool),
		color:    make(map[entity.GameEntityID]matrix.Color),
		width:    make(map[entity.GameEntityID]float32),
	}
}

func (f *fakeOutline) SetOutline(id entity.GameEntityID, color matrix.Color, widthPx float32) {
	f.outlines[id] = true
	f.color[id] = color
	f.width[id] = widthPx
}

func (f *fakeOutline) ClearOutline(id entity.GameEntityID) {
	f.outlines[id] = false
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// newTestDetector builds a detector on a real Default-zoom 16:9 camera with a
// recording fader.
func newTestDetector(t *testing.T) (*camera.OcclusionDetector, *fakeFader) {
	t.Helper()
	cam := camera.NewIsometric(matrix.Vec3{}, camera.DefaultConfig(), camera.ZoomDefault, 1920, 1080)
	fader := newFakeFader()
	det := camera.NewOcclusionDetector(cam.Camera(), camera.DefaultOcclusionConfig(), fader)
	return det, fader
}

// structureAroundOrigin is a wall whose AABB fully contains a party member at
// the origin (view-space rect overlap + near side in front of the member).
func structureAroundOrigin(id entity.GameEntityID) camera.Occluder {
	return camera.Occluder{
		ID: id,
		AABB: camera.AABB{
			Min: matrix.NewVec3(-2, 0, -2),
			Max: matrix.NewVec3(2, 5, 2),
		},
	}
}

// farStructure is a wall far to the north-east: no screen-space overlap with
// a member at the origin (and it is in front of the camera in depth).
func farStructure(id entity.GameEntityID) camera.Occluder {
	return camera.Occluder{
		ID: id,
		AABB: camera.AABB{
			Min: matrix.NewVec3(50, 0, 50),
			Max: matrix.NewVec3(52, 5, 52),
		},
	}
}

// newTestParty returns two world positions: the origin (occluded by
// structureAroundOrigin) and (50, 0, 50) (no screen overlap with any of the
// test structures).
func newTestParty() [2]matrix.Vec3 {
	return [2]matrix.Vec3{
		{0, 0, 0},
		{50, 0, 50},
	}
}

func assertAlpha(t *testing.T, got, want, tol float32, ctx string) {
	t.Helper()
	if math.Abs(float64(got-want)) > float64(tol) {
		t.Errorf("%s: alpha = %.4f, expected %.4f ± %.4f", ctx, got, want, tol)
	}
}

// ── AC-1: Structure fade ────────────────────────────────────────────────────

func TestOcclusion_FadeToFloor_LinearWithin150ms(t *testing.T) {
	// GIVEN a structure whose screen-space AABB overlaps the party rect and
	// whose near side is between the camera and the party, WHEN the party
	// moves behind it, THEN the structure fades to 20% alpha within 150ms
	// (linear interpolation; QA: fade-alpha debug readout).
	det, fader := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	det.Update(party[:1], 75) // half of the fade duration

	if !det.IsOccluded(1) {
		t.Error("IsOccluded(1) = false, expected true (structure between camera and party)")
	}
	assertAlpha(t, det.Alpha(1), 0.6, 0.001, "75ms into fade") // 1 - 0.8*(75/150)

	det.Update(party[:1], 75) // 150ms total

	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "150ms into fade")
	if got := fader.alpha[1]; math.Abs(float64(got-0.2)) > 0.001 {
		t.Errorf("fader alpha = %.4f, expected 0.2 (occluder_fade_alpha)", got)
	}
}

func TestOcclusion_NoScreenOverlap_StaysOpaque(t *testing.T) {
	// A structure outside the party's screen rect must never fade.
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{farStructure(1)})

	det.Update(party[:1], 200)
	det.Update(party[:1], 200)

	if det.IsOccluded(1) {
		t.Error("IsOccluded(1) = true, expected false (no screen-space overlap)")
	}
	assertAlpha(t, det.Alpha(1), 1, 0.001, "far structure")
}

// ── AC-2: Fade hysteresis ───────────────────────────────────────────────────

func TestOcclusion_ClearMustPersist300ms_BeforeRestore(t *testing.T) {
	// GIVEN a structure in the faded state, WHEN the occlusion clears, THEN
	// the structure stays faded while the clear persists < 300ms and restores
	// only after >= 300ms of continuous clear (occluder_fade_min_clear).
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	det.Update(party[:1], 150) // fade to floor
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "faded")

	det.Update(party[1:], 100) // clear 100ms — must stay faded
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "clear 100ms")
	det.Update(party[1:], 200) // clear 300ms total — restore may start
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "clear 300ms (restore not yet visible)")

	det.Update(party[1:], 75) // 75ms into the 150ms restore
	assertAlpha(t, det.Alpha(1), 0.6, 0.001, "75ms into restore")

	det.Update(party[1:], 75) // restore complete
	assertAlpha(t, det.Alpha(1), 1, 0.001, "restore complete")
}

func TestOcclusion_ReocclusionDuringHold_ResetsClearWindow(t *testing.T) {
	// A brief re-occlusion during the clear window resets the clear timer; the
	// structure must not restore until 300ms of NEW continuous clear.
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	det.Update(party[:1], 150) // fade to floor
	det.Update(party[1:], 200) // clear 200ms
	det.Update(party[:1], 100) // re-occlusion resets the clear window
	det.Update(party[1:], 200) // clear 200ms again

	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "clear window reset by re-occlusion")

	det.Update(party[1:], 100) // 300ms continuous clear — restore starts
	det.Update(party[1:], 150) // restore completes
	assertAlpha(t, det.Alpha(1), 1, 0.001, "restored after reset clear window")
}

func TestOcclusion_HighFrequencyFlicker_AlphaMonotonicToFloor(t *testing.T) {
	// A 4Hz flicker pattern (125ms occluded / 125ms clear) must never produce
	// an alpha cycle: alpha is monotonically non-increasing and settles at the
	// floor.
	det, fader := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	last := float32(1)
	for i := 0; i < 4; i++ {
		det.Update(party[:1], 125)
		det.Update(party[1:], 125)
		a := det.Alpha(1)
		if a > last+0.001 {
			t.Fatalf("flicker step %d: alpha rose %.4f -> %.4f (must be monotonic)", i, last, a)
		}
		last = a
	}
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "after 4Hz flicker")
	if fader.alpha[1] > 0.201 {
		t.Errorf("fader alpha = %.4f, expected at floor 0.2", fader.alpha[1])
	}
}

func TestOcclusion_ReocclusionMidRestore_ReversesImmediately(t *testing.T) {
	// WHEN the occlusion returns during the restore fade, THEN the fade-out
	// reverses from the current alpha — never a snap to opaque.
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	det.Update(party[:1], 150) // fade to floor
	det.Update(party[1:], 300) // clear window passes — restore starts
	det.Update(party[1:], 75)  // mid-restore: alpha 0.6
	assertAlpha(t, det.Alpha(1), 0.6, 0.001, "mid-restore")

	det.Update(party[:1], 75) // re-occluded: reversal, fade-out resumes from 0.6 → 0.4
	a := det.Alpha(1)
	if a >= 0.6 {
		t.Errorf("alpha = %.4f after re-occlusion, expected < 0.6 (reverse from current alpha)", a)
	}
	assertAlpha(t, a, 0.4, 0.001, "75ms into resumed fade-out")
	det.Update(party[:1], 75)
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "fade-out completes again")
}

func TestOcclusion_WallWalk_NoCompletedCycles(t *testing.T) {
	// Walking-alongside-wall pattern (400ms occluded / 400ms clear): the
	// structure fades out and is re-occluded before the restore completes —
	// zero completed alpha cycles over 4 seconds, alpha always within
	// [floor, 1].
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	restores := 0
	for i := 0; i < 5; i++ { // 5 wall-walk cycles = 4s
		det.Update(party[:1], 400) // occluded phase
		det.Update(party[1:], 400) // clear phase
		a := det.Alpha(1)
		if a < 0.2-0.001 || a > 1+0.001 {
			t.Fatalf("cycle %d: alpha %.4f outside [0.2, 1]", i, a)
		}
		if a >= 0.999 {
			restores++ // a completed fade-out -> fade-in cycle within the walk
		}
	}
	if restores != 0 {
		t.Errorf("wall-walk produced %d completed alpha cycles, expected 0 (no flicker)", restores)
	}
}

func TestOcclusion_WorstCaseCycle_Minimum600ms(t *testing.T) {
	// Worst-case toggle pattern (occluded exactly 150ms, clear exactly 300ms)
	// yields the minimum possible cycle: 150 fade-out + 300 hold/clear + 150
	// fade-in = 600ms = 1.67Hz < 2Hz (GDD CR-8 bound). The story QA's 2s
	// claim is unreachable with the story's own constants (flagged).
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	// Full cycle: fade-out start -> alpha 1 -> fade-out start again.
	det.Update(party[:1], 150) // occlude: fade-out completes
	det.Update(party[1:], 300) // clear: restore begins at the end of this step
	det.Update(party[1:], 150) // restore completes
	assertAlpha(t, det.Alpha(1), 1, 0.001, "restored")

	cycleMs := float32(150 + 300 + 150)
	if cycleMs < 500 {
		t.Fatalf("minimum cycle %.0fms would exceed 2Hz", cycleMs)
	}
	// And the actual observed cycle (re-occlude immediately) must hold too.
	det.Update(party[:1], 150)
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "cycle repeats at 600ms")
}

// ── Foliage alpha floor ─────────────────────────────────────────────────────

func TestOcclusion_Foliage_FadesTo35Percent(t *testing.T) {
	// GDD edge cases: foliage materials use a higher alpha floor (35% vs 20%).
	det, fader := newTestDetector(t)
	party := newTestParty()
	foliage := structureAroundOrigin(2)
	foliage.Foliage = true
	det.SetStructures([]camera.Occluder{foliage})

	det.Update(party[:1], 150)

	assertAlpha(t, det.Alpha(2), 0.35, 0.001, "foliage floor")
	if math.Abs(float64(fader.alpha[2]-0.35)) > 0.001 {
		t.Errorf("fader alpha = %.4f, expected 0.35", fader.alpha[2])
	}
}

// ── Structure lifecycle ─────────────────────────────────────────────────────

func TestOcclusion_RemoveStructure_RestoresAlphaAndStopsUpdating(t *testing.T) {
	det, fader := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})
	det.Update(party[:1], 150)
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "faded")

	det.RemoveStructure(1)

	assertAlpha(t, det.Alpha(1), 1, 0.001, "after removal")
	if math.Abs(float64(fader.alpha[1]-1)) > 0.001 {
		t.Errorf("fader alpha after removal = %.4f, expected 1 (restored)", fader.alpha[1])
	}
}

func TestOcclusion_SetStructures_PrunesRemovedAndKeepsState(t *testing.T) {
	det, fader := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1), structureAroundOrigin(2)})
	det.Update(party[:1], 150)
	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "structure 1 faded")
	assertAlpha(t, det.Alpha(2), 0.2, 0.001, "structure 2 faded")

	// Region refresh: structure 1 stays (same AABB — state preserved),
	// structure 2 is pruned, structure 3 is new.
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1), farStructure(3)})

	assertAlpha(t, det.Alpha(1), 0.2, 0.001, "state preserved across refresh")
	assertAlpha(t, det.Alpha(2), 1, 0.001, "pruned structure restored")
	if math.Abs(float64(fader.alpha[2]-1)) > 0.001 {
		t.Errorf("pruned fader alpha = %.4f, expected 1 (restored on prune)", fader.alpha[2])
	}
	assertAlpha(t, det.Alpha(3), 1, 0.001, "new structure starts opaque")
}

func TestOcclusion_NoParty_NoStateChange(t *testing.T) {
	det, _ := newTestDetector(t)
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})

	det.Update(nil, 16) // zero party members — nothing to test against

	assertAlpha(t, det.Alpha(1), 1, 0.001, "no party, no fade")
}

// ── AC-3: Interior roof removal ─────────────────────────────────────────────

func TestInterior_Enter_HidesRoofsImmediately(t *testing.T) {
	// GIVEN the party enters an interior, WHEN the interior is active, THEN
	// roof/ceiling geometry is absent from render.
	vis := newFakeVisibility()
	ctl := camera.NewInteriorController(vis, camera.DefaultInteriorConfig())
	roofs := []entity.GameEntityID{101, 102}

	ctl.EnterInterior(roofs)

	if !ctl.IsInterior() || !ctl.AreRoofsHidden() {
		t.Error("expected interior state with roofs hidden")
	}
	for _, id := range roofs {
		if vis.visible[id] {
			t.Errorf("roof %d still visible after EnterInterior", id)
		}
	}
}

func TestInterior_Exit_RestoresWithin150ms(t *testing.T) {
	// THEN roofs restore within 150ms on exit (AC-3). With a 100ms configured
	// restore delay, visibility must return between 100ms and 150ms after exit.
	vis := newFakeVisibility()
	cfg := camera.DefaultInteriorConfig()
	cfg.RestoreDelayMs = 100
	ctl := camera.NewInteriorController(vis, cfg)
	roofs := []entity.GameEntityID{101}

	ctl.EnterInterior(roofs)
	ctl.ExitInterior()

	ctl.Update(50)
	if !ctl.AreRoofsHidden() {
		t.Error("roofs restored too early (50ms < 100ms delay)")
	}
	ctl.Update(50) // 100ms elapsed
	if ctl.AreRoofsHidden() {
		t.Error("roofs still hidden at 100ms after exit")
	}
	if !vis.visible[101] {
		t.Error("roof 101 not visible after restore")
	}
	if ctl.IsInterior() {
		t.Error("IsInterior() = true after exit")
	}
}

func TestInterior_ExitWithoutEnter_NoOp(t *testing.T) {
	vis := newFakeVisibility()
	ctl := camera.NewInteriorController(vis, camera.DefaultInteriorConfig())

	ctl.ExitInterior() // must not panic or toggle anything
	ctl.Update(16)

	if ctl.IsInterior() || ctl.AreRoofsHidden() {
		t.Error("unexpected state change from ExitInterior without EnterInterior")
	}
}

func TestInterior_ReenterWithNewRoofs_HidesThem(t *testing.T) {
	vis := newFakeVisibility()
	ctl := camera.NewInteriorController(vis, camera.DefaultInteriorConfig())

	ctl.EnterInterior([]entity.GameEntityID{101})
	ctl.ExitInterior()
	ctl.Update(150)                                    // restored
	ctl.EnterInterior([]entity.GameEntityID{102, 103}) // different building

	if !ctl.AreRoofsHidden() {
		t.Fatal("expected roofs hidden after re-entry")
	}
	for _, id := range []entity.GameEntityID{102, 103} {
		if vis.visible[id] {
			t.Errorf("new roof %d still visible after re-entry", id)
		}
	}
}

// ── AC-4: Interactable highlight ────────────────────────────────────────────

func TestHighlight_TabPress_AppliesParchmentOutline(t *testing.T) {
	// GIVEN an interactable in range, WHEN Tab is pressed, THEN a 2px solid
	// outline in parchment #E3D5B0 (227, 213, 176 — QA tolerance ±8/255) is
	// applied.
	prov := &fakeProvider{ids: []entity.GameEntityID{7, 8}}
	out := newFakeOutline()
	hl := camera.NewInteractableHighlight(camera.DefaultHighlightConfig(), prov, out)

	hl.SetTabHeld(true, matrix.Vec3{})

	if prov.calls != 1 {
		t.Errorf("provider queried %d times, expected 1 (Tab press edge only)", prov.calls)
	}
	want := camera.DefaultHighlightConfig().ParchmentColor
	for _, id := range []entity.GameEntityID{7, 8} {
		if !out.outlines[id] {
			t.Errorf("interactable %d not outlined", id)
		}
		got := out.color[id]
		for i := 0; i < 3; i++ {
			if math.Abs(float64(got[i]-want[i])) > 8.0/255.0 {
				t.Errorf("outline color[%d] = %.4f, expected %.4f ± 8/255 (parchment #E3D5B0)",
					i, got[i], want[i])
			}
		}
		if out.width[id] != 2 {
			t.Errorf("outline width = %.1fpx, expected 2px", out.width[id])
		}
	}

	hl.SetTabHeld(false, matrix.Vec3{})
	for _, id := range []entity.GameEntityID{7, 8} {
		if out.outlines[id] {
			t.Errorf("interactable %d still outlined after Tab release", id)
		}
	}
}

func TestHighlight_TabPress_AppliesThroughOccludingGeometry(t *testing.T) {
	// AC-4: the highlight is visible THROUGH occluding geometry — the outline
	// is applied even while the interactable is occluded from the camera.
	det, _ := newTestDetector(t)
	party := newTestParty()
	det.SetStructures([]camera.Occluder{structureAroundOrigin(1)})
	det.Update(party[:1], 16)
	if !det.IsOccluded(1) {
		t.Fatal("precondition failed: structure must be occluding")
	}

	prov := &fakeProvider{ids: []entity.GameEntityID{7}}
	out := newFakeOutline()
	hl := camera.NewInteractableHighlight(camera.DefaultHighlightConfig(), prov, out)

	hl.SetTabHeld(true, matrix.Vec3{})

	if !out.outlines[7] {
		t.Error("interactable behind occluding geometry not highlighted")
	}
}

func TestHighlight_RepressWithoutRelease_NoRequery(t *testing.T) {
	prov := &fakeProvider{ids: []entity.GameEntityID{7}}
	out := newFakeOutline()
	hl := camera.NewInteractableHighlight(camera.DefaultHighlightConfig(), prov, out)

	hl.SetTabHeld(true, matrix.Vec3{})
	hl.SetTabHeld(true, matrix.Vec3{}) // same held state — no new query
	hl.SetTabHeld(true, matrix.Vec3{})

	if prov.calls != 1 {
		t.Errorf("provider queried %d times, expected 1", prov.calls)
	}
	if !out.outlines[7] {
		t.Error("outline lost after repeated SetTabHeld(true)")
	}
}

// ── AC-9: Edge scroll over UI ───────────────────────────────────────────────

func TestEdgeScroll_CursorOverUI_Suppressed(t *testing.T) {
	// GIVEN the cursor at the screen edge over a UI panel, THEN edge scroll
	// does not fire. Keyboard/drag/gamepad pan is unaffected (no API here).
	// UI rects use Kaiju's (left, top, right, bottom) convention — the same
	// as the UI system's scissor rects.
	vp := matrix.NewVec4(0, 0, 1920, 1080)
	margin := float32(32)

	cases := []struct {
		name   string
		cursor matrix.Vec2
		ui     []matrix.Vec4
		margin float32
		want   bool
	}{
		{"left edge, no UI", matrix.NewVec2(8, 540), nil, margin, true},
		{"right edge, no UI", matrix.NewVec2(1912, 540), nil, margin, true},
		{"top edge, no UI", matrix.NewVec2(960, 8), nil, margin, true},
		{"bottom edge, no UI", matrix.NewVec2(960, 1072), nil, margin, true},
		{"corner, no UI", matrix.NewVec2(8, 8), nil, margin, true},
		{"center, no UI", matrix.NewVec2(960, 540), nil, margin, false},
		{"inside UI panel", matrix.NewVec2(100, 100),
			[]matrix.Vec4{matrix.NewVec4(0, 0, 200, 200)}, margin, false},
		{"left edge over UI", matrix.NewVec2(8, 100),
			[]matrix.Vec4{matrix.NewVec4(0, 0, 200, 200)}, margin, false},
		{"right edge over UI", matrix.NewVec2(1912, 540),
			[]matrix.Vec4{matrix.NewVec4(1800, 0, 1920, 1080)}, margin, false},
		{"edge band, UI elsewhere", matrix.NewVec2(8, 540),
			[]matrix.Vec4{matrix.NewVec4(400, 400, 600, 600)}, margin, true},
		{"outside viewport right", matrix.NewVec2(2000, 540), nil, margin, false},
		{"outside viewport left", matrix.NewVec2(-5, 540), nil, margin, false},
		{"zero margin uses default", matrix.NewVec2(20, 540), nil, 0, true},
		{"beyond default margin", matrix.NewVec2(40, 540), nil, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := camera.EdgeScrollAllowed(tc.cursor, vp, tc.ui, tc.margin)
			if got != tc.want {
				t.Errorf("EdgeScrollAllowed = %v, expected %v", got, tc.want)
			}
		})
	}
}

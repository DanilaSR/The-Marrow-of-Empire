// Unit tests for Story CAM-002: Zoom System.
//
// Evidence path: production/epics/camera-system/story-002-zoom-system.md
// Each test function maps to acceptance criteria (AC-2..AC-9) and QA test
// cases from the story.
package camera_test

import (
	"math"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/camera"
)

// newTestCamera returns a Default-config camera at Default zoom on a 16:9
// viewport. ExtraCloseEnabled defaults to false (accessibility opt-in).
func newTestCamera() *camera.Isometric {
	return camera.NewIsometric(matrix.Vec3{}, camera.DefaultConfig(), camera.ZoomDefault, 1920, 1080)
}

// ── AC-2 / AC-3: discrete step in and out ───────────────────────────────────

func TestZoomStep_WheelUp_DefaultToClose(t *testing.T) {
	// AC-2: wheel up at Default → transition to Close (4u) over 300ms.
	cam := newTestCamera()

	cam.ZoomStep(1)
	if cam.Zoom() != camera.ZoomDefault {
		t.Fatal("Zoom() changed before transition completed")
	}
	cam.Update(300)

	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomClose.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-camera.ZoomClose.HalfHeight)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected %.4f", cam.CurrentHalfHeight(), camera.ZoomClose.HalfHeight)
	}
}

func TestZoomStep_WheelDown_DefaultToFar(t *testing.T) {
	// AC-3: wheel down at Default → transition to Far (20u) over 300ms.
	cam := newTestCamera()

	cam.ZoomStep(-1)
	cam.Update(300)

	if cam.Zoom() != camera.ZoomFar {
		t.Errorf("Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomFar.Name)
	}
}

// ── QA AC-5 / edge: clamping at the ends of the ordered list ────────────────

func TestZoomStep_ClampAtMax_WheelDownAtFar(t *testing.T) {
	// AC-5: wheel down at Far → no effect.
	cam := newTestCamera()
	cam.ZoomStep(-1)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomFar {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	cam.ZoomStep(-1)
	cam.Update(300)

	if cam.Zoom() != camera.ZoomFar {
		t.Errorf("Zoom() = %s, expected %s (clamped)", cam.Zoom().Name, camera.ZoomFar.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-20)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected 20 (no-op)", cam.CurrentHalfHeight())
	}
}

func TestZoomStep_ClampAtMin_WheelUpAtExtraClose(t *testing.T) {
	// Clamp at the near end: wheel up at Extra Close → no effect.
	cam := newTestCamera()
	cam.ExtraCloseEnabled = true
	cam.ZoomStep(1) // Default → Close
	cam.Update(300)
	cam.ZoomStep(1) // Close → Extra Close
	cam.Update(300)
	if cam.Zoom() != camera.ZoomExtraClose {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	cam.ZoomStep(1)
	cam.Update(300)

	if cam.Zoom() != camera.ZoomExtraClose {
		t.Errorf("Zoom() = %s, expected %s (clamped)", cam.Zoom().Name, camera.ZoomExtraClose.Name)
	}
}

// ── AC-2: transition lerp ───────────────────────────────────────────────────

func TestZoomTransition_Monotonic_ReachesTargetWithin300ms(t *testing.T) {
	// AC-2: the transition lerps monotonically from 10u to 4u with no
	// overshoot, and completes within the 300ms duration — still in flight at
	// 250ms, settled exactly at 300ms.
	cam := newTestCamera()
	cam.ZoomTo(camera.ZoomClose)

	prev := cam.CurrentHalfHeight() // 10
	for i := 0; i < 5; i++ {        // 250ms: still in flight
		cam.Update(50)
		cur := cam.CurrentHalfHeight()
		if cur >= prev {
			t.Errorf("step %d: half-height %f not strictly decreasing (prev %f)", i, cur, prev)
		}
		if cur < camera.ZoomClose.HalfHeight-floatTolerance || cur > 10+floatTolerance {
			t.Errorf("step %d: half-height %f outside [4, 10] (overshoot)", i, cur)
		}
		prev = cur
	}
	if cam.Zoom() == camera.ZoomClose {
		t.Fatal("transition completed before 300ms")
	}
	if got := cam.CurrentHalfHeight(); math.Abs(float64(got-5)) > 0.01 {
		t.Errorf("half-height at 250ms = %.4f, expected ~5 (linear lerp)", got)
	}

	cam.Update(50) // 300ms total: completes exactly
	last := cam.CurrentHalfHeight()
	if math.Abs(float64(last-camera.ZoomClose.HalfHeight)) > floatTolerance {
		t.Errorf("final half-height = %.4f, expected %.4f ± %.3f", last, camera.ZoomClose.HalfHeight, floatTolerance)
	}
	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s after 300ms", cam.Zoom().Name, camera.ZoomClose.Name)
	}
}

// ── AC-7: rapid input during a transition ───────────────────────────────────

func TestZoomStep_RapidInput_ThreeDetents_SkipsIntermediate(t *testing.T) {
	// AC-7: 3 detents during a transition → at most one queued level,
	// intermediate levels skipped, lands on the last requested target.
	cam := newTestCamera()
	cam.ExtraCloseEnabled = true

	cam.ZoomStep(1) // start Default → Close
	cam.ZoomStep(1) // queue Extra Close
	cam.ZoomStep(1) // last input wins: still Extra Close
	cam.Update(300) // settle at Close, continue toward Extra Close immediately

	// The follow-up must already be in flight — Close never rests between frames.
	if got := cam.ZoomState(); got != camera.ZoomExtraClose {
		t.Fatalf("follow-up not in flight: ZoomState() = %s, expected %s", got.Name, camera.ZoomExtraClose.Name)
	}
	cam.Update(300)

	if cam.Zoom() != camera.ZoomExtraClose {
		t.Errorf("Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomExtraClose.Name)
	}
}

func TestZoomStep_RapidInput_LastInputWins(t *testing.T) {
	// AC-7: a reversing detent during a transition replaces the queued level.
	cam := newTestCamera()
	cam.ExtraCloseEnabled = true

	cam.ZoomStep(1)  // start Default → Close
	cam.ZoomStep(1)  // queue Extra Close
	cam.ZoomStep(-1) // reverse: queue Default instead (last input wins)
	cam.Update(300)  // settle at Close, continue toward Default
	cam.Update(300)

	if cam.Zoom() != camera.ZoomDefault {
		t.Errorf("Zoom() = %s, expected %s (returned to Default)", cam.Zoom().Name, camera.ZoomDefault.Name)
	}
}

// ── AC-6: Extra Close gating ────────────────────────────────────────────────

func TestExtraClose_Disabled_WheelUpAtClose_NoOp(t *testing.T) {
	// AC-6: Extra Close disabled (the default) → wheel up at Close is a no-op.
	cam := newTestCamera()
	cam.ZoomStep(1)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomClose {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	cam.ZoomStep(1)
	cam.Update(300)

	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s (Extra Close gated off)", cam.Zoom().Name, camera.ZoomClose.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-4)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected 4 (no-op)", cam.CurrentHalfHeight())
	}
}

func TestExtraClose_Enabled_WheelUpAtClose_ExtraClose(t *testing.T) {
	// AC-6: Extra Close enabled → wheel up at Close transitions to 2.5u.
	cam := newTestCamera()
	cam.ZoomStep(1)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomClose {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	cam.ExtraCloseEnabled = true
	cam.ZoomStep(1)
	cam.Update(300)

	if cam.Zoom() != camera.ZoomExtraClose {
		t.Errorf("Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomExtraClose.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-2.5)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected 2.5", cam.CurrentHalfHeight())
	}
}

// ── Edge case: zero-size viewport ───────────────────────────────────────────

func TestZoomTransition_ZeroSizeViewport_PausedAndResumed(t *testing.T) {
	// Edge case: a minimized window (the engine sends a 0x0 viewport directly
	// to the Kaiju camera) pauses the zoom transition; it resumes on the first
	// Update after the viewport is restored. Must not panic.
	cam := newTestCamera()
	cam.Camera().ViewportChanged(0, 0) // simulate engine minimization

	cam.ZoomTo(camera.ZoomClose)
	cam.Update(300) // paused: no advance while minimized

	if cam.Zoom() == camera.ZoomClose {
		t.Fatal("transition completed while viewport was zero-sized")
	}
	if got := cam.CurrentHalfHeight(); math.Abs(float64(got-10)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected 10 (paused at start)", got)
	}

	cam.Resize(1920, 1080) // window restored
	cam.Update(300)

	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s after restore", cam.Zoom().Name, camera.ZoomClose.Name)
	}
}

// ── AC-8: per-context zoom persistence ──────────────────────────────────────

func TestContextPersistence_RestoresLastZoomPerContext(t *testing.T) {
	// AC-8: leaving a context remembers its zoom; re-entry restores that zoom
	// (not Default).
	cam := newTestCamera()

	cam.ZoomStep(1) // exploration: Default → Close
	cam.Update(300)
	if cam.Zoom() != camera.ZoomClose {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	// Enter conversation: restores its construction zoom (Default), then zoom out.
	cam.SetContext(camera.ConversationContext)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomDefault {
		t.Fatalf("conversation entry: Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomDefault.Name)
	}
	cam.ZoomStep(-1)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomFar {
		t.Fatalf("conversation setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	// Exit to exploration: restores Close (last exploration zoom), not Default.
	cam.SetContext(camera.ExplorationContext)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("exploration re-entry: Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomClose.Name)
	}

	// Re-enter conversation: restores Far.
	cam.SetContext(camera.ConversationContext)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomFar {
		t.Errorf("conversation re-entry: Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomFar.Name)
	}
}

// ── AC-9: save/load ─────────────────────────────────────────────────────────

func TestSaveLoad_ZoomState_RoundTripInstant(t *testing.T) {
	// AC-9: save/load zoom is discrete and instant — no transition.
	cam := newTestCamera()
	cam.ZoomStep(-1)
	cam.Update(300)
	if cam.Zoom() != camera.ZoomFar {
		t.Fatalf("setup failed: Zoom() = %s", cam.Zoom().Name)
	}

	saved := cam.ZoomState()
	if saved != camera.ZoomFar {
		t.Errorf("ZoomState() = %s, expected %s", saved.Name, camera.ZoomFar.Name)
	}

	// ZoomState during an in-flight transition reports the target (intent).
	cam.ZoomTo(camera.ZoomClose)
	if got := cam.ZoomState(); got != camera.ZoomClose {
		t.Errorf("ZoomState() during transition = %s, expected %s (target)", got.Name, camera.ZoomClose.Name)
	}

	// Load while a transition would otherwise be in flight: instant, no lerp.
	cam.SetZoomState(saved)
	if cam.Zoom() != camera.ZoomFar {
		t.Errorf("after load: Zoom() = %s, expected %s (instant)", cam.Zoom().Name, camera.ZoomFar.Name)
	}
	if got := cam.CurrentHalfHeight(); math.Abs(float64(got-20)) > floatTolerance {
		t.Errorf("after load: half-height = %.4f, expected 20 (no lerp)", got)
	}
	cam.Update(300)
	if cam.Zoom() != camera.ZoomFar {
		t.Errorf("after update: Zoom() = %s, expected %s (transition cancelled)", cam.Zoom().Name, camera.ZoomFar.Name)
	}
}

// ── Edge case: zero/negative dt must not advance or produce NaN ──────────────

func TestZoomTransition_ZeroOrNegativeDt_NoAdvance_NoNaN(t *testing.T) {
	// A zero or negative frame time (e.g. a paused or clamped frame timer)
	// must not advance the transition, corrupt state, or produce a NaN
	// half-height — and the transition must still complete exactly afterwards.
	cam := newTestCamera()
	cam.ZoomStep(1) // Default → Close in flight
	if !cam.Transitioning() {
		t.Fatal("setup failed: transition not in flight")
	}

	cam.Update(0) // zero dt: no division by zero, no advance
	if got := cam.CurrentHalfHeight(); got != camera.ZoomDefault.HalfHeight {
		t.Fatalf("Update(0) advanced half-height to %.4f, expected %.4f (no advance)",
			got, camera.ZoomDefault.HalfHeight)
	}
	if math.IsNaN(float64(cam.CurrentHalfHeight())) {
		t.Fatal("Update(0) produced a NaN half-height")
	}
	if cam.Zoom() != camera.ZoomDefault || !cam.Transitioning() {
		t.Fatalf("Update(0) changed state: Zoom() = %s, transitioning = %v",
			cam.Zoom().Name, cam.Transitioning())
	}

	cam.Update(-1) // negative dt: no backwards advance either
	if got := cam.CurrentHalfHeight(); got != camera.ZoomDefault.HalfHeight {
		t.Fatalf("Update(-1) advanced half-height to %.4f, expected %.4f (no advance)",
			got, camera.ZoomDefault.HalfHeight)
	}
	if math.IsNaN(float64(cam.CurrentHalfHeight())) {
		t.Fatal("Update(-1) produced a NaN half-height")
	}

	cam.Update(float32(math.NaN())) // NaN dt: guarded — must not advance or spread NaN
	if got := cam.CurrentHalfHeight(); got != camera.ZoomDefault.HalfHeight {
		t.Fatalf("Update(NaN) advanced half-height to %.4f, expected %.4f (no advance)",
			got, camera.ZoomDefault.HalfHeight)
	}
	if math.IsNaN(float64(cam.CurrentHalfHeight())) {
		t.Fatal("Update(NaN) produced a NaN half-height")
	}
	if cam.Zoom() != camera.ZoomDefault || !cam.Transitioning() {
		t.Fatalf("Update(NaN) changed state: Zoom() = %s, transitioning = %v",
			cam.Zoom().Name, cam.Transitioning())
	}

	cam.Update(300) // the transition still completes exactly on time
	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s after zero/negative dt frames", cam.Zoom().Name, camera.ZoomClose.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-camera.ZoomClose.HalfHeight)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected %.4f", cam.CurrentHalfHeight(), camera.ZoomClose.HalfHeight)
	}
	if cam.Transitioning() {
		t.Error("Transitioning() = true after completion")
	}
}

// ── AC-5: frame-granularity — completes at 300ms ± 1 frame at 60fps ──────────

func TestZoomTransition_CompletesAt300ms_60FpsFrames(t *testing.T) {
	// AC-5 at 60fps frame granularity: 17 frames of 1/60s (≈283ms) leave the
	// transition in flight; the 18th frame (exactly 300ms) settles it at the
	// exact target half-height.
	cam := newTestCamera()
	cam.ZoomStep(1) // Default → Close

	frame := float32(1000.0 / 60.0) // ms per frame at 60fps
	for i := 0; i < 17; i++ {
		cam.Update(frame)
		if cam.Zoom() == camera.ZoomClose {
			t.Fatalf("frame %d (%.0fms): transition completed early", i+1, float64(frame)*float64(i+1))
		}
	}
	if !cam.Transitioning() {
		t.Fatal("transition not in flight after 17 frames (283ms)")
	}

	cam.Update(frame) // 18th frame: 300ms total — settles exactly
	if cam.Zoom() != camera.ZoomClose {
		t.Errorf("Zoom() = %s, expected %s after 18 frames (300ms)", cam.Zoom().Name, camera.ZoomClose.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-camera.ZoomClose.HalfHeight)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected %.4f (exact target)", cam.CurrentHalfHeight(), camera.ZoomClose.HalfHeight)
	}
	if cam.Transitioning() {
		t.Error("Transitioning() = true after 18 frames")
	}
}

// ── AC-7: input during transition replaces intent without snapping ───────────

func TestZoomStep_InterruptMidTransition_NewTargetWins(t *testing.T) {
	// AC-7: an input arriving mid-transition replaces the in-flight intent
	// (last input wins). The half-height must not snap; the in-flight level
	// settles first, then the follow-up continues to the new target.
	cam := newTestCamera()

	cam.ZoomStep(1) // Default → Close
	cam.Update(150) // halfway: half-height = 7 (linear lerp)
	if got := cam.CurrentHalfHeight(); math.Abs(float64(got-7)) > floatTolerance {
		t.Fatalf("setup failed: half-height = %.4f, expected 7 at 150ms", got)
	}

	cam.ZoomStep(-1) // interrupt: retarget to Default (one step out from Close)
	if got := cam.CurrentHalfHeight(); math.Abs(float64(got-7)) > floatTolerance {
		t.Fatalf("interrupt snapped half-height to %.4f, expected 7 (no snap)", got)
	}

	cam.Update(150) // settle at Close; follow-up toward Default starts immediately
	if cam.Zoom() != camera.ZoomClose {
		t.Fatalf("after settle: Zoom() = %s, expected %s", cam.Zoom().Name, camera.ZoomClose.Name)
	}
	if !cam.Transitioning() {
		t.Fatal("follow-up transition not in flight after settle")
	}

	cam.Update(300) // land on the last requested target
	if cam.Zoom() != camera.ZoomDefault {
		t.Errorf("Zoom() = %s, expected %s (last input wins)", cam.Zoom().Name, camera.ZoomDefault.Name)
	}
	if math.Abs(float64(cam.CurrentHalfHeight()-camera.ZoomDefault.HalfHeight)) > floatTolerance {
		t.Errorf("half-height = %.4f, expected %.4f", cam.CurrentHalfHeight(), camera.ZoomDefault.HalfHeight)
	}
	if cam.Transitioning() {
		t.Error("Transitioning() = true after final landing")
	}
}

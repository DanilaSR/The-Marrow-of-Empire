// Integration tests for Story CAM-003: Camera Follow & Framing.
//
// Evidence path: production/epics/camera-system/story-003-follow-framing.md
// Required evidence: this file must exist and pass.
//
// The tests wire the full stack: a real Kaiji orthographic camera, a real
// entity store over a mock Kaiju host, and the camera Controller — entities
// are moved via their Kaiju transforms (Transform.Position() is the
// authoritative source, ADR-0001) and the camera reads them directly, never
// through the spatial index.
//
// QA test case -> test function map (Go only executes Test-prefixed
// functions, so the story's test_ names are exported with the Go prefix):
//
//	AC-1   test_follow_moves_at_5_units_per_second_clamped     -> TestFollowMovesAtFiveUnitsPerSecondClamped
//	AC-3   test_party_spread_follows_selected_not_centroid     -> TestPartySpreadFollowsSelectedNotCentroid
//	AC-4   test_conversation_framing_fits_inner_90_percent     -> TestConversationFramingFitsInner90Percent
//	AC-4   test_conversation_uses_minimum_zoom_that_fits       -> TestConversationUsesMinimumZoomThatFits
//	AC-5/6 test_speaker_change_reframes_only_outside_70        -> TestSpeakerChangeReframesOnlyOutsideInner70
//	AC-9   test_combat_framing_min_default_zoom_at_40ups       -> TestCombatFramingMinimumDefaultZoom
//	AC-10  test_combat_framing_oversized_frames_at_far_75      -> TestCombatFramingOversizedFramesAtFar
//	AC-15  test_manual_pan_holds_no_auto_return                -> TestManualPanHoldsNoAutoReturn
//	AC-15  test_drag_pan_point_under_cursor_tracking           -> TestDragPanPointUnderCursorTracking
//	AC-16  test_home_instantly_centers_on_party                -> TestHomeInstantlyCentersOnParty
//	AC-17  test_end_centers_on_selected_or_noop                -> TestEndCentersOnSelectedOrNoop
//	AC-18  test_combat_preempts_conversation_one_deep_stack    -> TestCombatPreemptsConversationOneDeepStack
//	AC-19/20 test_journal_close_returns_to_prior_base_state    -> TestJournalCloseReturnsToPriorBaseState
//	AC-23  test_off_screen_marker_nearest_edge_contract        -> TestOffScreenMarkerNearestEdgeContract
//	CR-4   test_edge_scroll_hysteresis                         -> TestEdgeScrollHysteresis
//	CR-4   test_keyboard_pan_constant_speed                    -> TestKeyboardPanConstantSpeed
package camera_test

import (
	"fmt"
	"math"
	"testing"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/camera"
	"marrow-of-empire/entity"
)

// eps is the position tolerance in world units for settled camera assertions.
const eps float32 = 1e-3

// ── Test rig (mirrored from tests/integration/entity — Go cannot import
// test helpers across packages) ──────────────────────────────────────────────

// mockKaijuHost is a test double for entity.KaijuHost: it registers engine
// entities by EntityId so Transform.Position() reads work end to end.
type mockKaijuHost struct {
	entities  map[engine.EntityId]*engine.Entity
	destroyed []engine.EntityId
	workGroup concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(e *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = e
	return true
}

func (m *mockKaijuHost) DestroyEntity(e *engine.Entity) {
	for id, ent := range m.entities {
		if ent == e {
			delete(m.entities, id)
			m.destroyed = append(m.destroyed, id)
			return
		}
	}
}

func (m *mockKaijuHost) WorkGroup() *concurrent.WorkGroup { return &m.workGroup }

func newMockHost() *mockKaijuHost {
	m := &mockKaijuHost{entities: make(map[engine.EntityId]*engine.Entity)}
	m.workGroup.Init()
	return m
}

// rig wires a real entity store over the mock host plus a real isometric
// camera and the CAM-003 controller at Default zoom on a 1920×1080 viewport.
type rig struct {
	store *entity.EntityStoreImpl
	host  *mockKaijuHost
	cam   *camera.Isometric
	ctl   *camera.Controller
}

func newRig() *rig {
	host := newMockHost()
	store := entity.NewEntityStore(host, entity.NewEventBus(), entity.EntityStoreConfig{})
	cam := camera.NewIsometric(matrix.Vec3{}, camera.DefaultConfig(), camera.ZoomDefault, 1920, 1080)
	ctl := camera.NewController(cam, camera.NewHostPositionSource(host),
		camera.DefaultFollowConfig(), camera.DefaultPanConfig())
	return &rig{store: store, host: host, cam: cam, ctl: ctl}
}

// char creates a Character entity at the given position.
func (r *rig) char(name string, pos matrix.Vec3) entity.GameEntityID {
	id, err := r.store.CreateEntity(entity.EntityTypeCharacter, name, pos)
	if err != nil {
		panic(fmt.Sprintf("CreateEntity %s: %v", name, err))
	}
	return id
}

// npc creates an NPC entity at the given position.
func (r *rig) npc(name string, pos matrix.Vec3) entity.GameEntityID {
	id, err := r.store.CreateEntity(entity.EntityTypeNPC, name, pos)
	if err != nil {
		panic(fmt.Sprintf("CreateEntity %s: %v", name, err))
	}
	return id
}

// move sets an entity's world position through its Kaiju transform — the
// authoritative position source the camera reads directly (ADR-0001).
func (r *rig) move(id entity.GameEntityID, pos matrix.Vec3) {
	kaiju := r.host.entities[engine.EntityId(fmt.Sprintf("game_%d", id))]
	if kaiju == nil {
		panic(fmt.Sprintf("no engine entity for game_%d", id))
	}
	kaiju.Transform.SetPosition(pos)
}

func (r *rig) pos(id entity.GameEntityID) matrix.Vec3 {
	kaiju := r.host.entities[engine.EntityId(fmt.Sprintf("game_%d", id))]
	if kaiju == nil {
		panic(fmt.Sprintf("no engine entity for game_%d", id))
	}
	return kaiju.Transform.Position()
}

func (r *rig) focus() matrix.Vec3 { return r.cam.Focus() }

func approx(a, b float32) bool { return float32(math.Abs(float64(a-b))) <= eps }

// ── AC-1: exploration follow at 5 u/s, distance-clamped, exact arrival ──────

func TestFollowMovesAtFiveUnitsPerSecondClamped(t *testing.T) {
	// Arrange: two party members at the camera origin; move them 20u away.
	r := newRig()
	a := r.char("A", matrix.NewVec3(0, 0, 0))
	b := r.char("B", matrix.NewVec3(0, 0, 0))
	r.ctl.TrackPartyMember(a)
	r.ctl.TrackPartyMember(b)
	r.move(a, matrix.NewVec3(20, 0, 0))
	r.move(b, matrix.NewVec3(20, 0, 0))

	// Act: 40 frames of 0.1s (constant velocity: 0.5u per frame at 5 u/s).
	for i := 0; i < 10; i++ {
		r.ctl.Update(0.1)
	}
	// Assert: after 1s exactly 5u moved — linear, not exponential ease.
	if !approx(r.focus().X(), 5) {
		t.Fatalf("after 1s focus.X() = %.4f, expected 5.0 (constant velocity)", r.focus().X())
	}

	for i := 0; i < 30; i++ {
		r.ctl.Update(0.1)
	}
	// Assert: camera reaches the centroid exactly, no overshoot.
	if !approx(r.focus().X(), 20) || !approx(r.focus().Z(), 0) {
		t.Fatalf("after 4s focus = %v, expected (20, 0, 0) exactly", r.focus())
	}

	// Assert: distance clamping — the party jogs 0.3u; one frame's step (0.5u)
	// exceeds the distance, so the camera reaches the target exactly with no
	// overshoot or drift past it.
	r.move(a, matrix.NewVec3(20.3, 0, 0))
	r.move(b, matrix.NewVec3(20.3, 0, 0))
	r.ctl.Update(0.1) // step (0.5) exceeds distance (0.3): clamps to target
	if !approx(r.focus().X(), 20.3) {
		t.Errorf("clamped follow focus.X() = %.4f, expected 20.3 exactly", r.focus().X())
	}
}

// ── AC-3: party spread > viewport half-extent -> follow selected ─────────────

func TestPartySpreadFollowsSelectedNotCentroid(t *testing.T) {
	// Arrange: two members 40u apart at Default zoom (half-extent = 10 × 16/9
	// ≈ 17.8u); the selected member is at (40, 0, 0).
	r := newRig()
	a := r.char("A", matrix.NewVec3(0, 0, 0))
	b := r.char("B", matrix.NewVec3(40, 0, 0))
	r.ctl.TrackPartyMember(a)
	r.ctl.TrackPartyMember(b)
	r.ctl.SetSelected(b)

	// Act: let the camera follow at 5 u/s for 9s (40u at 5 u/s = 8s).
	for i := 0; i < 90; i++ {
		r.ctl.Update(0.1)
	}

	// Assert: focus is the selected character (40,0,0), not the centroid (20,0,0).
	if !approx(r.focus().X(), 40) || !approx(r.focus().Z(), 0) {
		t.Fatalf("focus = %v, expected selected character (40, 0, 0)", r.focus())
	}
	if approx(r.focus().X(), 20) {
		t.Error("camera followed the party centroid (20,0,0) instead of the selected character")
	}
}

// ── AC-4: conversation framing — PC+NPC within inner 90%, last zoom ─────────

func TestConversationFramingFitsInner90Percent(t *testing.T) {
	// Arrange: PC at origin, NPC 8u away (side 8: fits at Default zoom:
	// 8×1.414 = 11.3 ≤ 2×10×(16/9)×0.9 = 32).
	r := newRig()
	pc := r.char("PC", matrix.NewVec3(0, 0, 0))
	npc := r.npc("Guard", matrix.NewVec3(8, 0, 0))
	r.ctl.TrackPartyMember(pc)
	r.ctl.SetPlayerCharacter(pc)
	r.ctl.SetSpeaker(npc)

	// Act: conversation starts; pan to midpoint takes snap×2 = 400ms.
	r.ctl.RequestConversation()
	r.ctl.Update(0.4)

	// Assert: camera at the PC+NPC midpoint (4, 0, 0).
	if !approx(r.focus().X(), 4) || !approx(r.focus().Z(), 0) {
		t.Fatalf("focus = %v, expected midpoint (4, 0, 0)", r.focus())
	}
	// Assert: pair fits within the inner 90% at the settled zoom (the pair fits
	// at the Conversation register, Default 10 — no zoom change needed).
	o := r.cam.CurrentHalfHeight()
	if !approx(o, 10) {
		t.Errorf("CurrentHalfHeight = %.2f, expected 10 (last Conversation zoom)", o)
	}
	if side := float32(8); side*1.41421356 > 2*o*(1920.0/1080.0)*0.9 {
		t.Errorf("pair side 8 does not fit the inner 90%% fit test at O=%.1f", o)
	}
}

// ── AC-4: conversation at a zoom that does not fit -> minimum zoom that fits ─

func TestConversationUsesMinimumZoomThatFits(t *testing.T) {
	// Arrange: conversation 1 with a close pair settles at Close zoom, so the
	// Conversation register remembers Close; conversation 2 has a pair 10u
	// apart, which does not fit at Close (14.1 > 12.8) and must use the
	// minimum zoom that fits (Default).
	r := newRig()
	pc := r.char("PC", matrix.NewVec3(0, 0, 0))
	close := r.npc("Close", matrix.NewVec3(2, 0, 0))
	far := r.npc("Far", matrix.NewVec3(10, 0, 0))
	r.ctl.TrackPartyMember(pc)
	r.ctl.SetPlayerCharacter(pc)

	// Conversation 1 (pair side 2): settle, then zoom in to Close.
	r.ctl.SetSpeaker(close)
	r.ctl.RequestConversation()
	r.ctl.Update(0.4)
	r.cam.ZoomStep(1) // Default -> Close
	r.ctl.Update(0.3)
	if !approx(r.cam.CurrentHalfHeight(), 4) {
		t.Fatalf("setup failed: half-height = %.2f, expected 4 (Close)", r.cam.CurrentHalfHeight())
	}
	r.ctl.EndConversation()
	r.ctl.Update(0.3) // zoom returns to last Exploration zoom

	// Conversation 2 (pair side 10): Close does not fit -> Default chosen.
	r.ctl.SetSpeaker(far)
	r.ctl.RequestConversation()
	r.ctl.Update(0.3) // zoom transition settles
	r.ctl.Update(0.4) // pan to midpoint (5, 0, 0) completes

	if !approx(r.cam.CurrentHalfHeight(), 10) {
		t.Errorf("half-height = %.2f, expected 10 (minimum zoom that fits)", r.cam.CurrentHalfHeight())
	}
	if !approx(r.focus().X(), 5) {
		t.Errorf("focus.X() = %.4f, expected 5 (PC+NPC midpoint)", r.focus().X())
	}
}

// ── AC-5/AC-6: speaker change re-frames only outside the inner 70% ───────────

func TestSpeakerChangeReframesOnlyOutsideInner70(t *testing.T) {
	// Arrange: conversation framing PC (0,0,0) and NPC1 (4,0,0); midpoint (2,0,0).
	r := newRig()
	pc := r.char("PC", matrix.NewVec3(0, 0, 0))
	npc1 := r.npc("Near", matrix.NewVec3(4, 0, 0))
	npc2 := r.npc("Mid", matrix.NewVec3(6, 0, 0))
	npc3 := r.npc("Far", matrix.NewVec3(40, 0, 0))
	npc4 := r.npc("Farer", matrix.NewVec3(50, 0, 0))
	r.ctl.TrackPartyMember(pc)
	r.ctl.SetPlayerCharacter(pc)
	r.ctl.SetSpeaker(npc1)
	r.ctl.RequestConversation()
	r.ctl.Update(0.4)
	if !approx(r.focus().X(), 2) {
		t.Fatalf("setup failed: focus.X() = %.4f, expected 2", r.focus().X())
	}

	// Speaker change inside the inner 70% (6,0,0 relative to focus (2,0,0)):
	// no re-frame.
	r.ctl.SetSpeaker(npc2)
	r.ctl.Update(0.5)
	if !approx(r.focus().X(), 2) {
		t.Fatalf("speaker inside inner 70%% re-framed: focus.X() = %.4f, expected 2", r.focus().X())
	}

	// Speaker change outside the inner 70% (40,0,0), no manual adjustment:
	// re-frame to the new midpoint (20,0,0) over 400ms.
	r.ctl.SetSpeaker(npc3)
	r.ctl.Update(0.4)
	if !approx(r.focus().X(), 20) {
		t.Fatalf("speaker outside inner 70%% did not re-frame: focus.X() = %.4f, expected 20", r.focus().X())
	}

	// Manual adjustment within conversation_override_window (5s) suppresses
	// the next speaker-change re-frame (AC-6).
	r.ctl.KeyboardPan(1, 0, 0.01) // raises Manual + starts the override window
	before := r.focus()
	r.ctl.SetSpeaker(npc4) // far outside: would re-frame if allowed
	r.ctl.Update(0.5)
	if !approx(r.focus().X(), before.X()) || !approx(r.focus().Z(), before.Z()) {
		t.Errorf("re-frame fired within the override window: focus moved from %v to %v",
			before, r.focus())
	}
}

// ── AC-9: combat framing at minimum Default zoom, 40 u/s ─────────────────────

func TestCombatFramingMinimumDefaultZoom(t *testing.T) {
	// Arrange: 4 party + 6 enemies in an 18×18u XZ box (ground plane XZ,
	// elevation Y = 0); camera at origin.
	r := newRig()
	partyPos := []matrix.Vec3{
		matrix.NewVec3(0, 0, 0), matrix.NewVec3(6, 0, 0),
		matrix.NewVec3(0, 0, 6), matrix.NewVec3(6, 0, 6),
	}
	enemyPos := []matrix.Vec3{
		matrix.NewVec3(12, 0, 0), matrix.NewVec3(18, 0, 0),
		matrix.NewVec3(12, 0, 6), matrix.NewVec3(18, 0, 6),
		matrix.NewVec3(12, 0, 12), matrix.NewVec3(18, 0, 12),
	}
	var combatants []entity.GameEntityID
	for i, p := range partyPos {
		id := r.char(fmt.Sprintf("P%d", i), p)
		r.ctl.TrackCombatant(id)
		r.ctl.TrackPartyMember(id)
		combatants = append(combatants, id)
	}
	for i, p := range enemyPos {
		id := r.char(fmt.Sprintf("E%d", i), p)
		r.ctl.TrackCombatant(id)
		combatants = append(combatants, id)
	}

	// Act: combat starts; the box (side 18, center (9,0,6)) fits at Default;
	// distance from origin ≈ 10.8u at 40 u/s ≈ 0.27s.
	r.ctl.RequestCombat()
	r.ctl.Update(0.3)

	// Assert: camera at the box center; framing zoom Default (10).
	if !approx(r.focus().X(), 9) || !approx(r.focus().Z(), 6) {
		t.Fatalf("focus = %v, expected box center (9, 0, 6)", r.focus())
	}
	if !approx(r.cam.CurrentHalfHeight(), 10) {
		t.Errorf("half-height = %.2f, expected 10 (Default)", r.cam.CurrentHalfHeight())
	}
	if !approx(r.focus().Y(), 0) {
		t.Errorf("focus.Y() = %.4f, expected 0 (box mid-elevation)", r.focus().Y())
	}

	// Act: player zooms to Close during combat, combat ends, combat starts
	// again — framing must use the player's Combat zoom at minimum Default.
	r.cam.ZoomStep(1) // Default -> Close
	r.ctl.Update(0.3)
	if !approx(r.cam.CurrentHalfHeight(), 4) {
		t.Fatalf("setup failed: half-height = %.2f, expected 4 (Close)", r.cam.CurrentHalfHeight())
	}
	r.ctl.EndCombat()
	r.ctl.RequestCombat() // re-enter: Combat register holds Close
	r.ctl.Update(0.3)     // zoom transition settles at Default (min)

	if !approx(r.cam.CurrentHalfHeight(), 10) {
		t.Errorf("after re-entry half-height = %.2f, expected 10 (minimum Default)", r.cam.CurrentHalfHeight())
	}
}

// ── AC-10: oversized encounter frames at Far with party + nearest 75% ────────

func TestCombatFramingOversizedFramesAtFar(t *testing.T) {
	// Arrange: 4 party + 16 enemies in a 60×60u box (side 60 does not fit at
	// Far: 60×1.414 = 84.9 > 2×20×(16/9)×0.9 = 64). Party at the origin
	// corner; enemies on a 4×4 grid at x,z ∈ {10, 26, 42, 58}.
	r := newRig()
	var partyIDs []entity.GameEntityID
	partyPos := []matrix.Vec3{
		matrix.NewVec3(0, 0, 0), matrix.NewVec3(2, 0, 0),
		matrix.NewVec3(0, 0, 2), matrix.NewVec3(2, 0, 2),
	}
	for i, p := range partyPos {
		id := r.char(fmt.Sprintf("P%d", i), p)
		r.ctl.TrackCombatant(id)
		r.ctl.TrackPartyMember(id)
		partyIDs = append(partyIDs, id)
	}
	var enemyIDs []entity.GameEntityID
	for i, x := range []float32{10, 26, 42, 58} {
		for j, z := range []float32{10, 26, 42, 58} {
			id := r.char(fmt.Sprintf("E%d", i*4+j), matrix.NewVec3(x, 0, z))
			r.ctl.TrackCombatant(id)
			enemyIDs = append(enemyIDs, id)
		}
	}

	// Expected frame (mirrors GDD CR-7 rule 1): party + the 12 nearest enemies
	// (75% of 16) by distance to the party centroid (1,0,1).
	pc := r.pos(partyIDs[0]).Add(r.pos(partyIDs[3])).Scale(0.5) // (1,0,1)
	type dist struct {
		id   entity.GameEntityID
		dist float32
	}
	enemies := make([]dist, 0, len(enemyIDs))
	for _, id := range enemyIDs {
		enemies = append(enemies, dist{id, r.pos(id).Distance(pc)})
	}
	for i := 0; i < len(enemies); i++ {
		best := i
		for j := i + 1; j < len(enemies); j++ {
			if enemies[j].dist < enemies[best].dist {
				best = j
			}
		}
		enemies[i], enemies[best] = enemies[best], enemies[i]
	}
	keep := int(math.Ceil(0.75 * float64(len(enemies))))
	kept := make([]entity.GameEntityID, 0, keep)
	for i := 0; i < keep; i++ {
		kept = append(kept, enemies[i].id) // nearest keep enemies (sorted order)
	}
	minX, maxX, minZ, maxZ := float32(math.MaxFloat32), float32(-math.MaxFloat32), float32(math.MaxFloat32), float32(-math.MaxFloat32)
	boxIDs := append(append([]entity.GameEntityID{}, partyIDs...), kept...)
	for _, id := range boxIDs {
		p := r.pos(id)
		minX, maxX = minF(minX, p.X()), maxF(maxX, p.X())
		minZ, maxZ = minF(minZ, p.Z()), maxF(maxZ, p.Z())
	}
	wantCenter := matrix.NewVec3((minX+maxX)/2, 0, (minZ+maxZ)/2)

	// Act: combat starts; zoom transitions to Far (300ms), then the 40 u/s
	// pan to the trimmed box center completes (≈ 30u / 40 = 0.75s).
	r.ctl.RequestCombat()
	r.ctl.Update(0.3)
	r.ctl.Update(0.8)

	// Assert: framing zoom Far; focus at the trimmed box center (party +
	// nearest 75% of enemies — the far-corner remainder excluded).
	if !approx(r.cam.CurrentHalfHeight(), 20) {
		t.Errorf("half-height = %.2f, expected 20 (Far)", r.cam.CurrentHalfHeight())
	}
	if !approx(r.focus().X(), wantCenter.X()) || !approx(r.focus().Z(), wantCenter.Z()) {
		t.Errorf("focus = %v, expected trimmed box center %v", r.focus(), wantCenter)
	}

	// Assert: every excluded enemy that projects beyond the Far viewport
	// (XZ ground plane, Y = 0) has a marker on its nearest edge (AC-10
	// "remainder get edge markers"); the on-screen remainder does not. The
	// projection mirrors the GDD fit-test math with the Far zoom registers.
	markers := r.ctl.OffScreenMarkers()
	got := make(map[entity.GameEntityID]camera.ScreenEdge)
	for _, m := range markers {
		got[m.EntityID] = m.Edge
	}
	hw := float32(20) * (1920.0 / 1080.0) // Far half-width = O × AR
	wantOff := map[entity.GameEntityID]camera.ScreenEdge{}
	for _, id := range enemyIDs {
		p := r.pos(id)
		dx, dz := p.X()-r.focus().X(), p.Z()-r.focus().Z()
		sxWorld := (dx - dz) * 0.70710678
		syWorld := (dx + dz) * 0.35355339
		if absF(sxWorld) > hw || absF(syWorld) > 20 {
			wantOff[id] = camera.EdgeBottom
		}
	}
	for id, edge := range wantOff {
		if got[id] != edge {
			t.Errorf("enemy %d: marker edge = %v, expected %v (nearest edge)", id, got[id], edge)
		}
	}
}

// ── AC-15: manual pan holds position — no auto-return ────────────────────────

func TestManualPanHoldsNoAutoReturn(t *testing.T) {
	// Arrange: party at origin, camera centered on them.
	r := newRig()
	a := r.char("A", matrix.NewVec3(0, 0, 0))
	r.ctl.TrackPartyMember(a)

	// Act: drag-pan right by 50px (engages past the 8px threshold), release.
	r.ctl.PanStart(960, 540)
	r.ctl.PanDrag(1010, 540)
	if !r.ctl.Pan().Active() {
		t.Fatal("drag did not engage past the 8px threshold")
	}
	r.ctl.PanEnd()
	held := r.focus()
	if !r.ctl.State().Manual() {
		t.Fatal("Manual overlay not raised by drag pan")
	}

	// Assert: with the party moving and time passing, the camera holds the
	// manual position (drift < 0.1u over 500ms, AC-15) — no auto-return.
	r.move(a, matrix.NewVec3(30, 0, 0))
	r.ctl.Update(0.5)
	dx := absF(r.focus().X() - held.X())
	dz := absF(r.focus().Z() - held.Z())
	if dx > 0.1 || dz > 0.1 {
		t.Fatalf("camera drifted after manual pan: focus %v -> %v (drift %.2f, %.2f)",
			held, r.focus(), dx, dz)
	}

	// Assert: Home recovers — instant return to the party centroid and the
	// Manual overlay is cleared so base follow resumes (GDD CR-3).
	r.ctl.Home()
	if r.ctl.State().Manual() {
		t.Error("Manual overlay not cleared by Home")
	}
	if !approx(r.focus().X(), 30) || !approx(r.focus().Z(), 0) {
		t.Errorf("after Home focus = %v, expected party centroid (30, 0, 0)", r.focus())
	}
}

// ── AC-15: drag pan keeps the world point under the cursor ───────────────────

func TestDragPanPointUnderCursorTracking(t *testing.T) {
	// Arrange: Default zoom 1920×1080, focus (0,0,0). The world point under
	// the cursor (860, 540) is recovered by inverting the isometric
	// projection: screen_x = (dx−dz)×cos45, screen_y = (dx+dz)×cos45/2.
	r := newRig()
	hw := float32(10) * (1920.0 / 1080.0)
	sxWorld := (860-960) * hw / 960.0
	_ = (540-540) * 10 / 540.0 // syWorld = 0 for center row
	// Invert: dx−dz = sxWorld/cos45; dx+dz = syWorld/(cos45/2) = 0.
	dx := sxWorld / 0.70710678 / 2
	dz := -dx
	worldPoint := matrix.NewVec3(dx, 0, dz)

	// Act: drag from (860,540) to (1060,540) — 200px right.
	r.ctl.PanStart(860, 540)
	r.ctl.PanDrag(1060, 540)
	r.ctl.PanEnd()

	// Assert: the world point under the start cursor projects to within 2
	// logical pixels of the end cursor (point-under-cursor tracking).
	fx, fz := r.focus().X(), r.focus().Z()
	rdx, rdz := worldPoint.X()-fx, worldPoint.Z()-fz
	sx := (rdx - rdz) * 0.70710678
	sy := (rdx + rdz) * 0.35355339
	px := sx/hw*960 + 960
	py := sy/10*540 + 540
	if absF(px-1060) > 2 || absF(py-540) > 2 {
		t.Errorf("world point under cursor projected to (%.1f, %.1f), expected (1060, 540) ± 2px",
			px, py)
	}
}

// ── AC-16: Home instantly centers on the party centroid ──────────────────────

func TestHomeInstantlyCentersOnParty(t *testing.T) {
	// Arrange: party at (15, 0, 25); camera focus elsewhere.
	r := newRig()
	a := r.char("A", matrix.NewVec3(15, 0, 25))
	b := r.char("B", matrix.NewVec3(15, 0, 25))
	r.ctl.TrackPartyMember(a)
	r.ctl.TrackPartyMember(b)
	r.ctl.KeyboardPan(-1, 0, 2) // pan away (manual)

	// Act: Home — instant, single call, no frames elapsed.
	zoomBefore := r.cam.CurrentHalfHeight()
	r.ctl.Home()

	// Assert: centroid within 0.5u of focus on the first frame after input;
	// no intermediate positions; zoom unchanged; Manual cleared.
	if d := r.focus().Distance(matrix.NewVec3(15, 0, 25)); d > 0.5 {
		t.Errorf("after Home focus distance to centroid = %.3f, expected ≤ 0.5", d)
	}
	if !approx(r.cam.CurrentHalfHeight(), zoomBefore) {
		t.Errorf("zoom changed by Home: %.2f -> %.2f", zoomBefore, r.cam.CurrentHalfHeight())
	}
	if r.ctl.State().Manual() {
		t.Error("Manual overlay not cleared by Home")
	}
}

// ── AC-17: End centers on the selected character; no selection -> no-op ──────

func TestEndCentersOnSelectedOrNoop(t *testing.T) {
	// Arrange: selected character at (40, 0, 10); camera at origin.
	r := newRig()
	sel := r.char("Sel", matrix.NewVec3(40, 0, 10))
	r.ctl.TrackPartyMember(sel)
	r.ctl.SetSelected(sel)

	// Act: End key.
	r.ctl.CenterOnSelected()

	// Assert: centered within one frame (no Update needed).
	if !approx(r.focus().X(), 40) || !approx(r.focus().Z(), 10) {
		t.Errorf("after End focus = %v, expected (40, 0, 10)", r.focus())
	}

	// Act: no selection -> no-op.
	r.ctl.PanStart(960, 540)
	r.ctl.PanDrag(1100, 540) // manual pan away
	r.ctl.PanEnd()
	held := r.focus()
	r.ctl.SetSelected(0)
	r.ctl.CenterOnSelected()
	if !approx(r.focus().X(), held.X()) || !approx(r.focus().Z(), held.Z()) {
		t.Errorf("End with no selection moved the camera: %v -> %v", held, r.focus())
	}
}

// ── AC-18: combat preempts conversation (one-deep stack) ─────────────────────

func TestCombatPreemptsConversationOneDeepStack(t *testing.T) {
	// Arrange: conversation active.
	r := newRig()
	pc := r.char("PC", matrix.NewVec3(0, 0, 0))
	npc := r.npc("Guard", matrix.NewVec3(4, 0, 0))
	r.ctl.TrackPartyMember(pc)
	r.ctl.SetPlayerCharacter(pc)
	r.ctl.SetSpeaker(npc)
	r.ctl.RequestConversation()
	if r.ctl.State().Base() != camera.ConversationBase {
		t.Fatal("setup failed: not in Conversation")
	}

	// Act: combat starts mid-conversation.
	r.ctl.RequestCombat()

	// Assert: Combat preempts within one frame (immediately on the call).
	if r.ctl.State().Base() != camera.CombatBase {
		t.Fatalf("base = %v, expected Combat (preemption within one frame)", r.ctl.State().Base())
	}

	// Act: combat ends.
	r.ctl.EndCombat()

	// Assert: returns to Conversation (one-deep stack), not Exploration.
	if r.ctl.State().Base() != camera.ConversationBase {
		t.Errorf("base = %v, expected Conversation (one-deep stack pop)", r.ctl.State().Base())
	}
}

// ── AC-19/AC-20: journal overlay — close returns to the prior base state ─────

func TestJournalCloseReturnsToPriorBaseState(t *testing.T) {
	// Arrange: conversation active.
	r := newRig()
	pc := r.char("PC", matrix.NewVec3(0, 0, 0))
	npc := r.npc("Guard", matrix.NewVec3(4, 0, 0))
	r.ctl.TrackPartyMember(pc)
	r.ctl.SetPlayerCharacter(pc)
	r.ctl.SetSpeaker(npc)
	r.ctl.RequestConversation()

	// Act: journal opens and closes during conversation.
	r.ctl.OpenJournal()
	if r.ctl.State().Base() != camera.ConversationBase {
		t.Fatalf("journal open changed base state to %v", r.ctl.State().Base())
	}
	r.ctl.CloseJournal()

	// Assert: returns to Conversation — NOT unconditionally Exploration (AC-20).
	if r.ctl.State().Base() != camera.ConversationBase {
		t.Errorf("journal close: base = %v, expected Conversation (prior base state)", r.ctl.State().Base())
	}

	// Act: journal during combat (AC-19) — camera stays in Combat framing.
	r.ctl.RequestCombat()
	r.ctl.OpenJournal()
	if r.ctl.State().Base() != camera.CombatBase {
		t.Fatalf("journal open during combat changed base to %v", r.ctl.State().Base())
	}
	if !r.ctl.State().Journal() {
		t.Fatal("Journal overlay not set")
	}
	before := r.focus()
	r.ctl.Update(0.5) // combat framing holds while journal open
	if !approx(r.focus().X(), before.X()) || !approx(r.focus().Z(), before.Z()) {
		t.Errorf("camera moved while journal open: %v -> %v", before, r.focus())
	}
	r.ctl.CloseJournal()

	// Assert: combat framing resumes (still Combat, not Exploration).
	if r.ctl.State().Base() != camera.CombatBase {
		t.Errorf("journal close during combat: base = %v, expected Combat", r.ctl.State().Base())
	}
}

// ── AC-23: off-screen marker data contract — nearest edge + anchor ───────────

func TestOffScreenMarkerNearestEdgeContract(t *testing.T) {
	// Arrange: party member far to the right, hostile far top-left, one
	// party member on screen; camera at origin at Default zoom.
	r := newRig()
	onScreen := r.char("Near", matrix.NewVec3(2, 0, 0))
	right := r.char("Right", matrix.NewVec3(50, 0, 0))
	hostile := r.npc("Rogue", matrix.NewVec3(-40, 0, -40))
	r.ctl.TrackPartyMember(onScreen)
	r.ctl.TrackPartyMember(right)
	r.ctl.TrackCombatant(hostile)

	// Act: markers computed for the current frame, journal open included
	// (markers are active regardless of camera state).
	r.ctl.OpenJournal()
	markers := r.ctl.OffScreenMarkers()

	// Assert: exactly the two off-screen entities, each with the nearest edge.
	if len(markers) != 2 {
		t.Fatalf("marker count = %d, expected 2", len(markers))
	}
	var rightMarker, hostileMarker *camera.EdgeMarker
	for i := range markers {
		switch markers[i].EntityID {
		case right:
			rightMarker = &markers[i]
		case hostile:
			hostileMarker = &markers[i]
		default:
			t.Errorf("unexpected marker for entity %d (on-screen entity must not produce a marker)", markers[i].EntityID)
		}
	}
	if rightMarker == nil || rightMarker.Edge != camera.EdgeRight {
		t.Errorf("party marker edge = %v, expected EdgeRight (nearest edge)", markerEdge(rightMarker))
	}
	if !approx(rightMarker.AnchorX, 1920-32) {
		t.Errorf("party marker AnchorX = %.1f, expected 1888 (right edge inset 32px)", rightMarker.AnchorX)
	}
	if rightMarker.Kind != camera.TrackedPartyMember {
		t.Errorf("party marker kind = %v, expected TrackedPartyMember", rightMarker.Kind)
	}
	if rightMarker.DisplayName != "Right" {
		t.Errorf("party marker display name = %q, expected %q", rightMarker.DisplayName, "Right")
	}
	if hostileMarker == nil || hostileMarker.Edge != camera.EdgeTop {
		t.Errorf("hostile marker edge = %v, expected EdgeTop (nearest edge)", markerEdge(hostileMarker))
	}
	if hostileMarker.Kind != camera.TrackedHostileCombatant {
		t.Errorf("hostile marker kind = %v, expected TrackedHostileCombatant", hostileMarker.Kind)
	}
}

// markerEdge formats a nil marker for error messages.
func markerEdge(m *camera.EdgeMarker) camera.ScreenEdge {
	if m == nil {
		return camera.ScreenEdge(-1)
	}
	return m.Edge
}

// ── CR-4: edge scroll with hysteresis + UI suppression ───────────────────────

func TestEdgeScrollHysteresis(t *testing.T) {
	// Arrange: camera at origin, cursor 8px inside the right edge (within the
	// 16px deadzone).
	r := newRig()
	r.ctl.SetCursor(1912, 540)

	// Act: 1s of edge scrolling at 15 u/s.
	r.ctl.Update(1.0)

	// Assert: panned right by 15u along the screen-right axis.
	want := float32(15 * 0.70710678)
	if !approx(r.focus().X(), want) || !approx(r.focus().Z(), -want) {
		t.Fatalf("focus = %v, expected (%.2f, 0, %.2f) after 1s edge scroll", r.focus(), want, -want)
	}
	if !r.ctl.Pan().EdgeScrolling() {
		t.Fatal("edge scroll not active inside the deadzone")
	}

	// Act: cursor retreats to 20px from the edge — inside deadzone+hysteresis
	// (16 × 1.5 = 24px): still scrolling.
	r.ctl.SetCursor(1900, 540)
	r.ctl.Update(0.1)
	if !r.ctl.Pan().EdgeScrolling() {
		t.Fatal("edge scroll stopped within the hysteresis margin (20px < 24px)")
	}

	// Act: cursor retreats to 50px — past the exit threshold: scrolling stops.
	r.ctl.SetCursor(1870, 540)
	r.ctl.Update(0.1)
	if r.ctl.Pan().EdgeScrolling() {
		t.Fatal("edge scroll did not stop past deadzone + hysteresis")
	}
	held := r.focus()
	r.ctl.Update(1.0)
	if !approx(r.focus().X(), held.X()) || !approx(r.focus().Z(), held.Z()) {
		t.Error("camera kept moving after edge scroll deactivated")
	}

	// Act: cursor over a UI panel at the edge — edge scroll disabled (CR-4).
	r.ctl.SetCursor(1912, 540)
	r.ctl.SetCursorOverUI(true)
	r.ctl.Update(1.0)
	if r.ctl.Pan().EdgeScrolling() {
		t.Fatal("edge scroll fired while cursor over a UI panel")
	}
	if !approx(r.focus().X(), held.X()) {
		t.Error("camera moved while cursor over a UI panel")
	}
}

// ── CR-4: keyboard pan at 20 u/s ─────────────────────────────────────────────

func TestKeyboardPanConstantSpeed(t *testing.T) {
	// Arrange: camera at origin.
	r := newRig()

	// Act: hold D for 1s (20 u/s along screen-right).
	r.ctl.KeyboardPan(1, 0, 1.0)

	// Assert: moved 20u; Manual overlay raised.
	want := float32(20 * 0.70710678)
	if !approx(r.focus().X(), want) || !approx(r.focus().Z(), -want) {
		t.Fatalf("focus = %v, expected (%.2f, 0, %.2f)", r.focus(), want, -want)
	}
	if !r.ctl.State().Manual() {
		t.Error("Manual overlay not raised by keyboard pan")
	}

	// Act: two half-second frames — constant speed, no acceleration. Each step
	// is measured from the position before that step.
	held := r.focus()
	r.ctl.KeyboardPan(0, -1, 0.5) // W for 0.5s
	step1 := r.focus().Distance(held)
	held = r.focus()
	r.ctl.KeyboardPan(0, -1, 0.5)
	step2 := r.focus().Distance(held)
	if absF(step1-10) > eps || absF(step2-10) > eps {
		t.Errorf("keyboard pan steps = %.3f, %.3f, expected 10u each (constant speed)", step1, step2)
	}
}

// ── Small helpers ────────────────────────────────────────────────────────────

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

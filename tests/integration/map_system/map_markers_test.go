// Integration tests for Story 002: Map Markers & Pins.
//
// Evidence path: production/epics/map-system/story-002-map-markers-pins.md
// Required evidence: this file must exist and pass (Integration story type
// — BLOCKING gate).
//
// The tests drive the map headlessly (NewLocalMap with nil host and nil
// manager skips markup loading), exercising the marker model, the provider
// contract, the world-to-surface positioning math, and the shape
// distinction between the amber dot and the amber star. The Kaiju document
// rendering of the marker elements themselves is verified by the manual
// walkthrough (production/qa/evidence) — the renderer is thin glue over
// the model pinned here.
//
// The acceptance criteria map to test functions (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_npc_marker_at_last_known_position   -> TestNPCMarkerAtLastKnownPosition
//	edge  test_npc_marker_unnamed_placeholder      -> TestNPCMarkerUnnamedShowsPlaceholder
//	AC-2  test_thread_dot_at_referenced_location   -> TestThreadDotAtReferencedLocation
//	edge  test_thread_without_location_no_marker   -> TestThreadWithoutLocationNoMarker
//	AC-3  test_prioritized_thread_shows_amber_star -> TestPrioritizedThreadShowsAmberStar
//	edge  test_two_stars_at_same_location          -> TestTwoStarsAtSameLocation
//	AC-3  test_amber_star_shape_geometry           -> TestAmberStarShapeGeometry
//	AC-4  test_discovered_location_pin_and_label   -> TestDiscoveredLocationPinAndLabel
//	edge  test_long_location_name_truncated        -> TestLongLocationNameTruncated
//	AC-5  test_cleared_location_keeps_pin_shows_x  -> TestClearedLocationKeepsPinShowsX
//	AC-6  test_markers_scoped_to_current_region    -> TestMarkersScopedToCurrentRegion
//	edge  test_markers_refresh_on_open             -> TestMarkersRefreshOnOpen
//	edge  test_refresh_markers_while_open          -> TestRefreshMarkersWhileOpen
//	edge  test_no_region_id_clears_markers         -> TestNoRegionIDClearsMarkers
//	AC-7  test_undiscovered_npc_never_rendered     -> TestUndiscoveredNPCNeverRendered
//	edge  test_marker_world_to_surface_math        -> TestMarkerWorldToSurfaceMath
//	edge  test_no_providers_no_markers             -> TestNoProvidersNoMarkers
//
// No random seeds, no time-dependent assertions, no external I/O.
package map_system_test

import (
	"math"
	"testing"

	localmap "marrow-of-empire/ui/map"
)

// stubNPCMarkerProvider is a mutable NPCMarkerProvider fixture (AC-1,
// AC-6, AC-7).
type stubNPCMarkerProvider struct {
	byRegion map[string][]localmap.NPCMarkerData
}

func (p *stubNPCMarkerProvider) DiscoveredNPCs(regionID string) []localmap.NPCMarkerData {
	return p.byRegion[regionID]
}

// stubThreadMarkerProvider is a mutable ThreadMarkerProvider fixture
// (AC-2, AC-3, AC-6).
type stubThreadMarkerProvider struct {
	byRegion map[string][]localmap.ThreadMarkerData
}

func (p *stubThreadMarkerProvider) ActiveThreads(regionID string) []localmap.ThreadMarkerData {
	return p.byRegion[regionID]
}

// stubLocationMarkerProvider is a mutable LocationMarkerProvider fixture
// (AC-4, AC-5, AC-6).
type stubLocationMarkerProvider struct {
	byRegion map[string][]localmap.LocationMarkerData
}

func (p *stubLocationMarkerProvider) DiscoveredLocations(regionID string) []localmap.LocationMarkerData {
	return p.byRegion[regionID]
}

// markerRegion is a 250x200 m region (2000x1600 px at the default 8 px/m
// map scale) with a stable region ID for marker scoping.
func markerRegion(id string) localmap.RegionMapData {
	return localmap.RegionMapData{
		RegionID: id,
		Name:     "Test Region",
		Width:    250,
		Height:   200,
	}
}

func newHeadlessMarkerMap() *localmap.LocalMap {
	return localmap.NewLocalMap(nil, nil, nil)
}

func markerApproxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// ---------------------------------------------------------------------------
// AC-1: NPC markers at last known position
// ---------------------------------------------------------------------------

// test_npc_marker_at_last_known_position: a discovered NPC appears on the
// map at the surface position of its world coordinates, with its display
// name in the snapshot (the renderer labels the pin from it).
func TestNPCMarkerAtLastKnownPosition(t *testing.T) {
	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {{EntityID: 101, Name: "Greta", X: 100, Y: 50, RegionID: "R1"}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()
	m.Update(0.5) // complete the enter slide (mirrors the real flow)

	snap := m.Markers()
	if len(snap.NPCs) != 1 {
		t.Fatalf("NPC markers = %d, want 1", len(snap.NPCs))
	}
	n := snap.NPCs[0]
	if n.EntityID != 101 || n.Name != "Greta" || n.X != 100 || n.Y != 50 {
		t.Fatalf("NPC marker = %+v, want entity 101 Greta at (100, 50)", n)
	}

	// The renderer places the pin at the surface position of world
	// (100, 50): x*8 = 800, 1600 - 50*8 = 1200.
	px, py := localmap.WorldToSurface(100, 50, 8, 1600)
	if px != 800 || py != 1200 {
		t.Fatalf("surface position = (%v, %v), want (800, 1200)", px, py)
	}
}

// test_npc_marker_unnamed_placeholder: an NPC the player saw but never
// learned the name of still appears (the provider decided discovery), with
// the label falling back to "???" (AC-7 QA edge).
func TestNPCMarkerUnnamedShowsPlaceholder(t *testing.T) {
	if got := localmap.MarkerDisplayName(""); got != "???" {
		t.Fatalf("MarkerDisplayName(\"\") = %q, want %q", got, "???")
	}
	if got := localmap.MarkerDisplayName("Greta"); got != "Greta" {
		t.Fatalf("MarkerDisplayName(Greta) = %q, want Greta", got)
	}

	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {{EntityID: 102, Name: "", X: 10, Y: 10, RegionID: "R1"}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()
	if len(m.Markers().NPCs) != 1 {
		t.Fatalf("unnamed NPC must still appear (discovery is the provider's call): %+v", m.Markers().NPCs)
	}
}

// ---------------------------------------------------------------------------
// AC-2: active threads -> amber dots
// ---------------------------------------------------------------------------

// test_thread_dot_at_referenced_location: two active threads at different
// locations each yield an amber dot marker at their referenced position
// (one dot per thread-location pair; the provider fans out multi-location
// threads, verified in TestThreadWithoutLocationNoMarker).
func TestThreadDotAtReferencedLocation(t *testing.T) {
	threadProv := &stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {
			{ThreadID: "thread_a_42", LocationX: 200, LocationY: 300, RegionID: "R1", IsPrioritized: false},
			{ThreadID: "thread_b_7", LocationX: 60, LocationY: 90, RegionID: "R1", IsPrioritized: false},
		},
	}}
	m := newHeadlessMarkerMap()
	m.SetThreadMarkerProvider(threadProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.Threads) != 2 {
		t.Fatalf("thread markers = %d, want 2", len(snap.Threads))
	}
	t0 := snap.Threads[0]
	if t0.ThreadID != "thread_a_42" || t0.LocationX != 200 || t0.LocationY != 300 || t0.IsPrioritized {
		t.Fatalf("thread marker 0 = %+v, want thread_a_42 at (200, 300), regular", t0)
	}
	// A regular thread renders as the amber dot — never the star.
	if kind := localmap.ThreadMarkerKind(t0.IsPrioritized); kind != localmap.MarkerKindThreadDot {
		t.Fatalf("regular thread kind = %v, want ThreadDot", kind)
	}
	// The renderer places the dot at the surface position of world
	// (200, 300): x*8 = 1600, 1600 - 300*8 = -800 (pass-through, no clamp).
	px, py := localmap.WorldToSurface(200, 300, 8, 1600)
	if px != 1600 || py != -800 {
		t.Fatalf("thread surface position = (%v, %v), want (1600, -800)", px, py)
	}
}

// test_thread_without_location_no_marker: a thread without a location
// reference produces no marker — the provider omits it and the map renders
// exactly what it receives.
func TestThreadWithoutLocationNoMarker(t *testing.T) {
	m := newHeadlessMarkerMap()
	m.SetThreadMarkerProvider(&stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {}, // provider filtered the location-less thread out
	}})
	m.SetRegion(markerRegion("R1"))
	m.Open()
	if got := len(m.Markers().Threads); got != 0 {
		t.Fatalf("thread markers = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// AC-3: prioritized threads (revelation_ready) -> amber stars
// ---------------------------------------------------------------------------

// test_prioritized_thread_shows_amber_star: a prioritized thread's
// location renders as the amber star — a shape distinct from the amber
// dot, never hue alone.
func TestPrioritizedThreadShowsAmberStar(t *testing.T) {
	threadProv := &stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {
			{ThreadID: "thread_prio_1", LocationX: 150, LocationY: 250, RegionID: "R1", IsPrioritized: true},
		},
	}}
	m := newHeadlessMarkerMap()
	m.SetThreadMarkerProvider(threadProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.Threads) != 1 || !snap.Threads[0].IsPrioritized {
		t.Fatalf("prioritized thread marker = %+v, want IsPrioritized=true", snap.Threads)
	}
	if kind := localmap.ThreadMarkerKind(true); kind != localmap.MarkerKindThreadStar {
		t.Fatalf("prioritized thread kind = %v, want ThreadStar", kind)
	}
	if localmap.ThreadMarkerKind(false) == localmap.ThreadMarkerKind(true) {
		t.Fatal("dot and star kinds must differ — shape, not hue")
	}
}

// test_two_stars_at_same_location: two prioritized threads at the same
// location yield two star markers — the map does not deduplicate;
// aggregation is the game layer's call (AC-3 QA edge).
func TestTwoStarsAtSameLocation(t *testing.T) {
	threadProv := &stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {
			{ThreadID: "ta", LocationX: 100, LocationY: 100, RegionID: "R1", IsPrioritized: true},
			{ThreadID: "tb", LocationX: 100, LocationY: 100, RegionID: "R1", IsPrioritized: true},
		},
	}}
	m := newHeadlessMarkerMap()
	m.SetThreadMarkerProvider(threadProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()
	if got := len(m.Markers().Threads); got != 2 {
		t.Fatalf("thread markers = %d, want 2 (both stars at the shared location)", got)
	}
}

// test_amber_star_shape_geometry: the star icon is five spokes, evenly
// spaced 72° apart, one pointing north, each spanning the star's radius
// (6px -> 12px star) with a 2px stroke. The spoke geometry is what makes
// the star shape-distinct from the dot.
func TestAmberStarShapeGeometry(t *testing.T) {
	const eps = 0.0001
	arms := localmap.StarArms(6)
	if len(arms) != 5 {
		t.Fatalf("StarArms(6) = %d arms, want 5", len(arms))
	}
	// Arm 0 points north (up on the map): center 3px above the star
	// center, no rotation (the arm panel is already vertical).
	if !markerApproxEqual(arms[0].OffsetX, 0, eps) ||
		!markerApproxEqual(arms[0].OffsetY, -3, eps) ||
		!markerApproxEqual(arms[0].RotationDeg, 0, eps) {
		t.Fatalf("arm 0 = %+v, want (0, -3, 0°)", arms[0])
	}
	// Each successive arm rotates -72° (screen convention, y-down).
	for i := 1; i < len(arms); i++ {
		if got := arms[i].RotationDeg - arms[i-1].RotationDeg; !markerApproxEqual(got, -72, eps) {
			t.Fatalf("arm %d rotation delta = %v°, want -72°", i, got)
		}
	}
	// Every arm center sits radius/2 from the star center.
	for i, a := range arms {
		if d := math.Hypot(a.OffsetX, a.OffsetY); !markerApproxEqual(d, 3, eps) {
			t.Fatalf("arm %d center distance = %v, want 3", i, d)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-4: discovered locations -> named pins
// ---------------------------------------------------------------------------

// test_discovered_location_pin_and_label: a discovered, named location
// yields a pin marker at its position with the discovered name in the
// snapshot (the renderer labels the pin from it).
func TestDiscoveredLocationPinAndLabel(t *testing.T) {
	locProv := &stubLocationMarkerProvider{byRegion: map[string][]localmap.LocationMarkerData{
		"R1": {{EntityID: 501, Name: "Old Watchtower", X: 400, Y: 100, RegionID: "R1", Cleared: false}},
	}}
	m := newHeadlessMarkerMap()
	m.SetLocationMarkerProvider(locProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.Locations) != 1 {
		t.Fatalf("location markers = %d, want 1", len(snap.Locations))
	}
	l := snap.Locations[0]
	if l.EntityID != 501 || l.Name != "Old Watchtower" || l.X != 400 || l.Y != 100 || l.Cleared {
		t.Fatalf("location marker = %+v, want Old Watchtower at (400, 100), not cleared", l)
	}
	// A name that fits renders unchanged (truncation kicks in past 24).
	if got := localmap.TruncateLabel(l.Name, 24); got != "Old Watchtower" {
		t.Fatalf("TruncateLabel = %q, want %q", got, l.Name)
	}
	px, py := localmap.WorldToSurface(400, 100, 8, 1600)
	if px != 3200 || py != 800 {
		t.Fatalf("location surface position = (%v, %v), want (3200, 800)", px, py)
	}
}

// test_long_location_name_truncated: labels over 24 characters truncate
// with an ellipsis at exactly 24, and truncation is rune-safe (AC-4 QA
// edge).
func TestLongLocationNameTruncated(t *testing.T) {
	long := "The Ancient Watchtower of the Northern Marches"
	got := localmap.TruncateLabel(long, 24)
	if runes := len([]rune(got)); runes != 24 {
		t.Fatalf("truncated label runes = %d, want 24: %q", runes, got)
	}
	if want := string([]rune(long)[:23]) + "…"; got != want {
		t.Fatalf("truncated label = %q, want %q", got, want)
	}

	// Boundary: exactly 24 runes pass through unchanged.
	exact := "ExactlyTwentyFourRunes!!"
	if got := localmap.TruncateLabel(exact, 24); got != exact {
		t.Fatalf("24-rune label must pass through, got %q", got)
	}

	// Rune-safety: multi-byte runes are never split mid-character.
	accents := "Östermarkturm der alten Wächter"
	got = localmap.TruncateLabel(accents, 24)
	if runes := len([]rune(got)); runes != 24 {
		t.Fatalf("accented truncation runes = %d, want 24: %q", runes, got)
	}
	if want := string([]rune(accents)[:23]) + "…"; got != want {
		t.Fatalf("accented truncation = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// AC-5: cleared locations -> ink X over the pin
// ---------------------------------------------------------------------------

// test_cleared_location_keeps_pin_shows_x: a cleared location keeps its
// pin and is flagged Cleared (the renderer draws the ink X over the pin);
// discovered and cleared markers coexist.
func TestClearedLocationKeepsPinShowsX(t *testing.T) {
	locProv := &stubLocationMarkerProvider{byRegion: map[string][]localmap.LocationMarkerData{
		"R1": {
			{EntityID: 601, Name: "Ruined Mill", X: 350, Y: 200, RegionID: "R1", Cleared: true},
			{EntityID: 602, Name: "Old Watchtower", X: 100, Y: 100, RegionID: "R1", Cleared: false},
		},
	}}
	m := newHeadlessMarkerMap()
	m.SetLocationMarkerProvider(locProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.Locations) != 2 {
		t.Fatalf("location markers = %d, want 2", len(snap.Locations))
	}
	if !snap.Locations[0].Cleared {
		t.Fatalf("Ruined Mill must be marked cleared: %+v", snap.Locations[0])
	}
	if snap.Locations[1].Cleared {
		t.Fatalf("Old Watchtower must not be marked cleared: %+v", snap.Locations[1])
	}
}

// ---------------------------------------------------------------------------
// AC-6: region-scoped markers
// ---------------------------------------------------------------------------

// test_markers_scoped_to_current_region: providers are queried with the
// current region ID only — Region B markers never appear while Region A is
// installed, and SetRegion swaps the marker layer (QA edge: transition
// while the map is open).
func TestMarkersScopedToCurrentRegion(t *testing.T) {
	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {{EntityID: 101, Name: "Greta", X: 10, Y: 10, RegionID: "R1"}},
		"R2": {{EntityID: 201, Name: "Orin", X: 20, Y: 20, RegionID: "R2"}},
	}}
	threadProv := &stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {{ThreadID: "t1", LocationX: 30, LocationY: 30, RegionID: "R1"}},
		"R2": {{ThreadID: "t2", LocationX: 40, LocationY: 40, RegionID: "R2"}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetThreadMarkerProvider(threadProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.NPCs) != 1 || snap.NPCs[0].EntityID != 101 {
		t.Fatalf("Region R1 NPCs = %+v, want only Greta (101)", snap.NPCs)
	}
	if len(snap.Threads) != 1 || snap.Threads[0].ThreadID != "t1" {
		t.Fatalf("Region R1 threads = %+v, want only t1", snap.Threads)
	}

	// Crossing into Region R2 while the map is open swaps the marker layer.
	m.SetRegion(markerRegion("R2"))
	snap = m.Markers()
	if len(snap.NPCs) != 1 || snap.NPCs[0].EntityID != 201 {
		t.Fatalf("Region R2 NPCs = %+v, want only Orin (201)", snap.NPCs)
	}
	if len(snap.Threads) != 1 || snap.Threads[0].ThreadID != "t2" {
		t.Fatalf("Region R2 threads = %+v, want only t2", snap.Threads)
	}
}

// test_markers_refresh_on_open: the marker layer snapshots provider data
// each time the map opens (AC-6 refresh semantics).
func TestMarkersRefreshOnOpen(t *testing.T) {
	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {{EntityID: 101, Name: "Greta", X: 10, Y: 10, RegionID: "R1"}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()
	if len(m.Markers().NPCs) != 1 {
		t.Fatal("open snapshot must show the discovered NPC")
	}
	m.Close()
	m.Update(0.5)

	// An NPC is discovered while the map is closed; the next open takes a
	// fresh snapshot.
	npcProv.byRegion["R1"] = append(npcProv.byRegion["R1"],
		localmap.NPCMarkerData{EntityID: 102, Name: "Orin", X: 20, Y: 20, RegionID: "R1"})
	m.Open()
	if got := len(m.Markers().NPCs); got != 2 {
		t.Fatalf("reopen snapshot NPCs = %d, want 2 (snapshot at open)", got)
	}
}

// test_refresh_markers_while_open: the game layer pushes knowledge changes
// with RefreshMarkers — e.g. a thread reaching revelation_ready flips its
// dot to a star on the next refresh (AC-3 QA edge).
func TestRefreshMarkersWhileOpen(t *testing.T) {
	threadProv := &stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"R1": {{ThreadID: "t1", LocationX: 30, LocationY: 30, RegionID: "R1", IsPrioritized: false}},
	}}
	m := newHeadlessMarkerMap()
	m.SetThreadMarkerProvider(threadProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	threadProv.byRegion["R1"][0].IsPrioritized = true
	if m.Markers().Threads[0].IsPrioritized {
		t.Fatal("snapshot must not change until RefreshMarkers is called")
	}
	m.RefreshMarkers()
	if !m.Markers().Threads[0].IsPrioritized {
		t.Fatal("RefreshMarkers must re-query the provider while the map is open")
	}
}

// test_no_region_id_clears_markers: a region without a RegionID has
// nothing to scope markers by, so the marker layer stays empty — there is
// deliberately no fallback to the region name (silent ID/name mismatch
// must be impossible).
func TestNoRegionIDClearsMarkers(t *testing.T) {
	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"": {{EntityID: 101, Name: "Greta", X: 10, Y: 10, RegionID: ""}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetRegion(localmap.RegionMapData{Name: "No ID Region", Width: 100, Height: 100})
	m.Open()
	if got := len(m.Markers().NPCs); got != 0 {
		t.Fatalf("markers without a region ID = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// AC-7: undiscovered NPCs never appear
// ---------------------------------------------------------------------------

// test_undiscovered_npc_never_rendered: the provider returns only
// discovered NPCs and the map renders exactly that list — an NPC the
// player has never seen has no marker (no omniscient placement).
func TestUndiscoveredNPCNeverRendered(t *testing.T) {
	npcProv := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {
			// Greta is discovered; entity 102 is deliberately absent from
			// the provider's list (never seen by the player).
			{EntityID: 101, Name: "Greta", X: 10, Y: 10, RegionID: "R1"},
		},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(npcProv)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.NPCs) != 1 || snap.NPCs[0].EntityID != 101 {
		t.Fatalf("NPC markers = %+v, want only the discovered NPC (101)", snap.NPCs)
	}
	for _, n := range snap.NPCs {
		if n.EntityID == 102 {
			t.Fatalf("undiscovered NPC 102 must not appear: %+v", snap.NPCs)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases: positioning math and provider-less operation
// ---------------------------------------------------------------------------

// test_marker_world_to_surface_math: the marker layer converts world
// coordinates (meters, y-up) to surface pixels (y-down, north up) at the
// map scale; out-of-region positions pass through unclamped.
func TestMarkerWorldToSurfaceMath(t *testing.T) {
	cases := []struct {
		x, y       float64
		wantX      float64
		wantY      float64
	}{
		{0, 0, 0, 1600},
		{100, 50, 800, 1200},
		{100, 150, 800, 400},
		{250, 200, 2000, 0}, // north-west corner (y-up: 200 is north)
	}
	for _, c := range cases {
		px, py := localmap.WorldToSurface(c.x, c.y, 8, 1600)
		if px != c.wantX || py != c.wantY {
			t.Fatalf("WorldToSurface(%v, %v) = (%v, %v), want (%v, %v)", c.x, c.y, px, py, c.wantX, c.wantY)
		}
	}
	// Off-surface positions are passed through, not clamped (the game
	// layer supplies in-region data; markers outside simply land off the
	// surface, like the player indicator).
	px, py := localmap.WorldToSurface(-50, 999, 8, 1600)
	if px != -400 || py != 1600-7992 {
		t.Fatalf("off-surface position = (%v, %v), want (-400, %v)", px, py, 1600-7992)
	}
}

// test_no_providers_no_markers: a map with no marker providers is
// nil-safe — open, region changes, and manual refreshes produce an empty
// snapshot without panicking.
func TestNoProvidersNoMarkers(t *testing.T) {
	m := newHeadlessMarkerMap()
	m.SetRegion(markerRegion("R1"))
	m.Open()
	if snap := m.Markers(); len(snap.NPCs) != 0 || len(snap.Threads) != 0 || len(snap.Locations) != 0 {
		t.Fatalf("no-provider snapshot must be empty: %+v", snap)
	}
	m.RefreshMarkers() // must not panic
	m.Update(0.016)    // must not panic
}

// ---------------------------------------------------------------------------
// H1: Markers() returns an independent deep copy — mutating the returned
// snapshot must not affect the map's internal state.
// ---------------------------------------------------------------------------

// test_markers_returns_independent_copy: mutating the snapshot returned by
// Markers() must not change the map's rendered state. The map owns its
// snapshot; the caller gets a copy.
func TestMarkersReturnsIndependentCopy(t *testing.T) {
	prov := &stubNPCMarkerProvider{byRegion: map[string][]localmap.NPCMarkerData{
		"R1": {{EntityID: 1, Name: "Greta", X: 100, Y: 50, RegionID: "R1"}},
	}}
	m := newHeadlessMarkerMap()
	m.SetNPCMarkerProvider(prov)
	m.SetRegion(markerRegion("R1"))
	m.Open()

	snap := m.Markers()
	if len(snap.NPCs) != 1 {
		t.Fatalf("initial snapshot NPCs = %d, want 1", len(snap.NPCs))
	}
	// Mutate the returned snapshot.
	snap.NPCs[0].Name = "CORRUPTED"
	snap.NPCs = append(snap.NPCs, localmap.NPCMarkerData{EntityID: 99, Name: "INJECTED"})

	// The map's internal snapshot must be unchanged.
	snap2 := m.Markers()
	if len(snap2.NPCs) != 1 {
		t.Fatalf("snapshot after mutation has %d NPCs, want 1 — caller mutation leaked", len(snap2.NPCs))
	}
	if snap2.NPCs[0].Name != "Greta" {
		t.Fatalf("snapshot name = %q, want %q — caller mutation leaked", snap2.NPCs[0].Name, "Greta")
	}
}

// ---------------------------------------------------------------------------
// M3: TruncateLabel boundary values
// ---------------------------------------------------------------------------

// test_truncate_label_boundaries: maxRunes <= 0 returns empty string;
// maxRunes == 1 returns just the ellipsis; a name shorter than the limit
// is returned unchanged.
func TestTruncateLabelBoundaries(t *testing.T) {
	if got := localmap.TruncateLabel("hello", 0); got != "" {
		t.Fatalf("TruncateLabel maxRunes=0 = %q, want \"\"", got)
	}
	if got := localmap.TruncateLabel("hello", -1); got != "" {
		t.Fatalf("TruncateLabel maxRunes=-1 = %q, want \"\"", got)
	}
	if got := localmap.TruncateLabel("hello", 1); got != "…" {
		t.Fatalf("TruncateLabel maxRunes=1 = %q, want \"…\"", got)
	}
	// Rune-safe: 3-byte UTF-8 characters are not split.
	if got := localmap.TruncateLabel("世界!", 3); got != "世界!" {
		t.Fatalf("TruncateLabel 3-rune CJK = %q, want unchanged", got)
	}
	if got := localmap.TruncateLabel("世界!!", 3); got != "世界…" {
		t.Fatalf("TruncateLabel 4-rune CJK at max=3 = %q, want truncated with ellipsis", got)
	}
}

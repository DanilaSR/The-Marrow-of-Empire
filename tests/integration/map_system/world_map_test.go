// Integration tests for Story 003: World Map & Region Travel.
//
// Evidence path: production/epics/map-system/story-003-world-map-region-travel.md
// Required evidence: this file must exist and pass (Integration story type
// — BLOCKING gate).
//
// The tests drive the world map headlessly (NewWorldMap with nil host and
// nil manager skips markup loading), exercising the region/route/faction
// models, the provider contracts, the canvas geometry (bounds, centroid,
// hit-testing, dash/dot/hatch math), the detail panel, the navigation
// return target, and the toggle debounce. The Kaiju document rendering of
// the shapes themselves is verified by the manual walkthrough
// (production/qa/evidence) — the renderer is thin glue over the model
// pinned here.
//
// The acceptance criteria map to test functions:
//
//	AC-1  test_discovered_regions_shown_on_world_map -> TestDiscoveredRegionsShownOnWorldMap
//	edge  test_zero_regions_shows_empty_parchment    -> TestZeroRegionsShowsEmptyParchment
//	edge  test_single_region_centered_no_scroll      -> TestSingleRegionCenteredNoScroll
//	edge  test_twenty_plus_regions_scrollable        -> TestTwentyPlusRegionsScrollable
//	AC-2  test_region_state_ink_treatments           -> TestRegionStateInkTreatments
//	edge  test_region_state_transition_refreshes     -> TestRegionStateTransitionRefreshesWhileOpen
//	AC-3  test_travel_route_connects_discovered      -> TestTravelRouteConnectsDiscoveredRegions
//	edge  test_route_hidden_if_endpoint_undiscovered -> TestTravelRouteHiddenWhenEndpointUndiscovered
//	edge  test_route_dot_segments_math               -> TestRouteDotSegmentsMath
//	AC-4  test_faction_wash_opacity_clamped          -> TestFactionWashOpacityClamped
//	edge  test_faction_overlays_snapshotted          -> TestFactionOverlaysSnapshottedPerRegion
//	edge  test_two_factions_blend_distinct_hatches   -> TestTwoFactionsBlendWithDistinctHatches
//	edge  test_unexplored_region_never_washes        -> TestUnexploredRegionNeverWashes
//	AC-5  test_region_click_shows_detail_panel       -> TestRegionClickShowsDetailPanel
//	edge  test_unexplored_detail_limited_info        -> TestUnexploredDetailShowsLimitedInfo
//	edge  test_click_outside_dismisses_panel         -> TestClickOutsideDismissesPanel
//	edge  test_detail_panel_fits_min_resolution      -> TestDetailPanelFitsMinimumResolution
//	AC-6  test_close_returns_to_local_map            -> TestCloseReturnsToLocalMap
//	edge  test_close_standalone_returns_to_game      -> TestCloseStandaloneReturnsToGame
//	edge  test_rapid_toggle_debounced                -> TestRapidToggleDebounced
//	edge  test_reduced_motion_opens_instantly        -> TestReducedMotionOpensInstantly
//	edge  test_world_canvas_geometry_math            -> TestWorldCanvasGeometryMath
//
// No random seeds, no time-dependent assertions, no external I/O.
package map_system_test

import (
	"math"
	"testing"

	localmap "marrow-of-empire/ui/map"
)

// stubWorldRegionProvider is a mutable WorldRegionProvider fixture (AC-1,
// AC-3, AC-2 refresh edge).
type stubWorldRegionProvider struct {
	regions []localmap.RegionSummary
	routes  []localmap.TravelRoute
}

func (p *stubWorldRegionProvider) DiscoveredRegions() []localmap.RegionSummary {
	return p.regions
}

func (p *stubWorldRegionProvider) TravelRoutes() []localmap.TravelRoute {
	return p.routes
}

// stubFactionProvider is a mutable FactionDataProvider fixture (AC-4).
type stubFactionProvider struct {
	byRegion map[string][]localmap.FactionOverlay
}

func (p *stubFactionProvider) FactionOverlays(regionID string) []localmap.FactionOverlay {
	return p.byRegion[regionID]
}

// rectBoundary is a closed rectangular boundary polygon (four segments)
// in world-canvas units — the authored shape of every test region.
func rectBoundary(x, y, w, h float64) []localmap.LineSegment {
	return []localmap.LineSegment{
		{X1: x, Y1: y, X2: x + w, Y2: y},
		{X1: x + w, Y1: y, X2: x + w, Y2: y + h},
		{X1: x + w, Y1: y + h, X2: x, Y2: y + h},
		{X1: x, Y1: y + h, X2: x, Y2: y},
	}
}

// region builds a 40x30 unit region at (x, y) in world-canvas units.
func region(id, name string, state localmap.RegionState, x, y float64) localmap.RegionSummary {
	return localmap.RegionSummary{
		RegionID: id,
		Name:     name,
		State:    state,
		Boundary: rectBoundary(x, y, 40, 30),
	}
}

func newHeadlessWorldMap() *localmap.WorldMap {
	return localmap.NewWorldMap(nil, nil)
}

// worldMapWith builds a headless world map with the region provider
// installed and the viewport sized to the reference 1920x1080.
func worldMapWith(regions []localmap.RegionSummary, routes []localmap.TravelRoute) *localmap.WorldMap {
	m := newHeadlessWorldMap()
	m.SetWorldRegionProvider(&stubWorldRegionProvider{regions: regions, routes: routes})
	m.SetViewportSize(1920, 1080)
	return m
}

func approxEqual(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

// ---------------------------------------------------------------------------
// AC-1: discovered regions as ink-drawn shapes on parchment
// ---------------------------------------------------------------------------

// test_discovered_regions_shown_on_world_map: three discovered regions in
// three different states all appear in the world map snapshot, each with
// its authored boundary and state.
func TestDiscoveredRegionsShownOnWorldMap(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
		region("oldroad", "Old Road", localmap.RegionStateCleared, 60, 0),
		region("thornwood", "Thornwood", localmap.RegionStateUnexplored, 120, 0),
	}, nil)
	m.OpenStandalone()
	m.Update(0.5) // complete the enter slide

	data := m.Data()
	if len(data.Regions) != 3 {
		t.Fatalf("discovered regions = %d, want 3", len(data.Regions))
	}
	got := map[string]localmap.RegionState{}
	for _, r := range data.Regions {
		got[r.RegionID] = r.State
		if len(r.Boundary) == 0 {
			t.Fatalf("region %s has no boundary linework", r.RegionID)
		}
	}
	want := map[string]localmap.RegionState{
		"march":     localmap.RegionStateVisited,
		"oldroad":   localmap.RegionStateCleared,
		"thornwood": localmap.RegionStateUnexplored,
	}
	for id, state := range want {
		if got[id] != state {
			t.Fatalf("region %s state = %v, want %v", id, got[id], state)
		}
	}
	if m.IsEmpty() {
		t.Fatal("map with three regions must not report empty")
	}
}

// test_zero_regions_shows_empty_parchment: no discovered regions -> the
// map reports empty (the renderer shows the hint text) and the canvas is
// degenerate.
func TestZeroRegionsShowsEmptyParchment(t *testing.T) {
	m := newHeadlessWorldMap()
	m.SetViewportSize(1920, 1080)
	m.OpenStandalone()

	if !m.IsEmpty() {
		t.Fatal("map without a provider must report empty")
	}
	if data := m.Data(); len(data.Regions) != 0 || len(data.Routes) != 0 || len(data.Factions) != 0 {
		t.Fatalf("empty map data = %+v, want all empty", data)
	}
}

// test_single_region_centered_no_scroll: one small region produces a
// canvas smaller than the viewport, so the viewport clamp centers it —
// no scrolling needed (AC-1 QA edge).
func TestSingleRegionCenteredNoScroll(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("solo", "Solo", localmap.RegionStateVisited, 0, 0),
	}, nil)
	m.OpenStandalone()
	m.Update(0.5)

	// Canvas: 40x30 units at 4px/unit plus 2x80px padding = 320x280px.
	ox, oy := m.ViewportOffset()
	if !approxEqual(ox, (1920-320)/2, 0.01) || !approxEqual(oy, (1080-280)/2, 0.01) {
		t.Fatalf("single region offset = (%v, %v), want centered (%v, %v)",
			ox, oy, (1920-320)/2, (1080-280)/2)
	}
}

// test_twenty_plus_regions_scrollable: twenty regions spanning a large
// world canvas pan freely and clamp at the canvas edges (AC-1 QA edge:
// 20+ regions scroll/pan to view all).
func TestTwentyPlusRegionsScrollable(t *testing.T) {
	regions := make([]localmap.RegionSummary, 0, 20)
	for i := 0; i < 20; i++ {
		regions = append(regions, region(
			"r"+string(rune('a'+i)), "Region "+string(rune('A'+i)),
			localmap.RegionStateVisited, float64(i)*60, 0))
	}
	m := worldMapWith(regions, nil)
	m.OpenStandalone()
	m.Update(0.5)

	// World canvas: 1180 units wide -> 4720 + 160 padding = 4880px > 1920.
	m.Pan(-500, 0)
	ox, _ := m.ViewportOffset()
	if !approxEqual(ox, -500, 0.01) {
		t.Fatalf("pan offset = %v, want -500", ox)
	}
	// Panning past the canvas clamps at the far edge (viewport - canvas).
	m.Pan(-5000, 0)
	ox, _ = m.ViewportOffset()
	if !approxEqual(ox, 1920-4880, 0.01) {
		t.Fatalf("clamped offset = %v, want %v", ox, 1920-4880)
	}
}

// ---------------------------------------------------------------------------
// AC-2: region state summaries — Cleared, Visited, Unexplored
// ---------------------------------------------------------------------------

// test_region_state_ink_treatments: the state->ink contract — Cleared is
// faded with an X, Visited is solid ink, Unexplored is dashed with no
// fill (AC-2).
func TestRegionStateInkTreatments(t *testing.T) {
	if got := localmap.RegionStateInk(localmap.RegionStateVisited); got != "solid" {
		t.Fatalf("visited ink = %q, want solid", got)
	}
	if got := localmap.RegionStateInk(localmap.RegionStateCleared); got != "faint" {
		t.Fatalf("cleared ink = %q, want faint", got)
	}
	if got := localmap.RegionStateInk(localmap.RegionStateUnexplored); got != "dashed" {
		t.Fatalf("unexplored ink = %q, want dashed", got)
	}
	// Cleared: faded ink; the renderer adds the X at the region centroid.
	_, dashed := localmap.RegionBoundaryInk(localmap.RegionStateCleared)
	if dashed {
		t.Fatal("cleared boundary must be solid, not dashed")
	}
	_, dashed = localmap.RegionBoundaryInk(localmap.RegionStateUnexplored)
	if !dashed {
		t.Fatal("unexplored boundary must be dashed")
	}
	if localmap.RegionWashColor(localmap.RegionStateVisited) == "" {
		t.Fatal("visited region must have an interior wash")
	}
	if localmap.RegionWashColor(localmap.RegionStateCleared) == "" {
		t.Fatal("cleared region must have a faded interior wash")
	}
	if localmap.RegionWashColor(localmap.RegionStateUnexplored) != "" {
		t.Fatal("unexplored region must have no fill")
	}
	if localmap.RegionStateName(localmap.RegionStateCleared) != "Cleared" {
		t.Fatal("state display name must be Cleared")
	}
	// The X anchor is the region centroid (20, 15) of the 40x30 rect.
	cx, cy := localmap.RegionCentroid(region("r", "R", localmap.RegionStateCleared, 0, 0))
	if cx != 20 || cy != 15 {
		t.Fatalf("centroid = (%v, %v), want (20, 15)", cx, cy)
	}
}

// test_region_state_transition_refreshes: a region transitioning from
// Visited to Cleared while the map is open is picked up by Refresh
// (AC-2 QA edge).
func TestRegionStateTransitionRefreshesWhileOpen(t *testing.T) {
	prov := &stubWorldRegionProvider{regions: []localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}}
	m := newHeadlessWorldMap()
	m.SetWorldRegionProvider(prov)
	m.SetViewportSize(1920, 1080)
	m.OpenStandalone()

	if got := m.Data().Regions[0].State; got != localmap.RegionStateVisited {
		t.Fatalf("state before refresh = %v, want Visited", got)
	}
	prov.regions[0].State = localmap.RegionStateCleared
	m.Refresh()
	if got := m.Data().Regions[0].State; got != localmap.RegionStateCleared {
		t.Fatalf("state after refresh = %v, want Cleared", got)
	}
}

// ---------------------------------------------------------------------------
// AC-3: dotted travel routes between discovered regions
// ---------------------------------------------------------------------------

// test_travel_route_connects_discovered: a route between two discovered
// regions is visible (AC-3).
func TestTravelRouteConnectsDiscoveredRegions(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
		region("oldroad", "Old Road", localmap.RegionStateVisited, 60, 0),
	}, []localmap.TravelRoute{
		{FromRegion: "march", ToRegion: "oldroad"},
	})
	m.OpenStandalone()

	routes := m.VisibleRoutes()
	if len(routes) != 1 || routes[0].FromRegion != "march" || routes[0].ToRegion != "oldroad" {
		t.Fatalf("visible routes = %+v, want the march->oldroad route", routes)
	}
}

// test_route_hidden_if_endpoint_undiscovered: a route touching an
// undiscovered region is hidden — both endpoints must be discovered
// (AC-3 QA edge).
func TestTravelRouteHiddenWhenEndpointUndiscovered(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}, []localmap.TravelRoute{
		{FromRegion: "march", ToRegion: "oldroad"}, // oldroad undiscovered
		{FromRegion: "nowhere", ToRegion: "march"}, // nowhere undiscovered
	})
	m.OpenStandalone()

	if routes := m.VisibleRoutes(); len(routes) != 0 {
		t.Fatalf("visible routes = %+v, want none (both endpoints must be discovered)", routes)
	}
}

// test_route_dot_segments_math: a dotted route between two points yields
// evenly spaced short dots along the line (the dotted ink path the
// renderer draws).
func TestRouteDotSegmentsMath(t *testing.T) {
	dots := localmap.RouteDotSegments(0, 0, 100, 0, 20, 6)
	if len(dots) != 5 {
		t.Fatalf("dots = %d, want 5", len(dots))
	}
	first := dots[0]
	if first.X1 != 0 || first.X2 != 6 || first.Y1 != 0 || first.Y2 != 0 {
		t.Fatalf("first dot = %+v, want (0,0)->(6,0)", first)
	}
	last := dots[len(dots)-1]
	if last.X1 != 80 || last.X2 != 86 {
		t.Fatalf("last dot = %+v, want (80,0)->(86,0)", last)
	}
	// Every dot is dotLen long, on the line, spacing apart.
	for i, d := range dots {
		if math.Hypot(d.X2-d.X1, d.Y2-d.Y1) != 6 {
			t.Fatalf("dot %d length = %v, want 6", i, math.Hypot(d.X2-d.X1, d.Y2-d.Y1))
		}
	}
}

// ---------------------------------------------------------------------------
// AC-4: faction territory washes with hatch patterns
// ---------------------------------------------------------------------------

// test_faction_wash_opacity_clamped: the settled wash opacity stays in
// the 10-15% band whatever the requested value (AC-4).
func TestFactionWashOpacityClamped(t *testing.T) {
	if got := localmap.FactionWashOpacity(0.05); got != 0.10 {
		t.Fatalf("wash opacity 0.05 = %v, want floor 0.10", got)
	}
	if got := localmap.FactionWashOpacity(0.12); got != 0.12 {
		t.Fatalf("wash opacity 0.12 = %v, want 0.12", got)
	}
	if got := localmap.FactionWashOpacity(1.0); got != 0.15 {
		t.Fatalf("wash opacity 1.0 = %v, want cap 0.15", got)
	}
}

// test_faction_overlays_snapshotted: faction overlays are snapshotted per
// region — a region with standing below the presence threshold is absent
// from the provider's output and renders nothing (AC-4 + QA edge).
func TestFactionOverlaysSnapshottedPerRegion(t *testing.T) {
	prov := &stubFactionProvider{byRegion: map[string][]localmap.FactionOverlay{
		"march": {
			{FactionName: "Iron Banner", Color: "#7A2E28", Hatch: localmap.HatchDiagonalA, Opacity: 0.12},
			// Merchant Guild standing below threshold — omitted by the provider.
		},
	}}
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
		region("oldroad", "Old Road", localmap.RegionStateVisited, 60, 0),
	}, nil)
	m.SetFactionDataProvider(prov)
	m.OpenStandalone()

	data := m.Data()
	if got := len(data.Factions["march"]); got != 1 {
		t.Fatalf("march overlays = %d, want 1 (provider already filtered)", got)
	}
	if data.Factions["march"][0].FactionName != "Iron Banner" {
		t.Fatalf("march overlay = %+v, want Iron Banner", data.Factions["march"][0])
	}
	if _, ok := data.Factions["oldroad"]; ok {
		t.Fatal("oldroad must have no overlay (no faction above threshold)")
	}
}

// test_two_factions_blend_distinct_hatches: two factions above threshold
// in one region both render, with shape-distinct hatch directions — never
// hue alone (AC-4 QA edge, control manifest).
func TestTwoFactionsBlendWithDistinctHatches(t *testing.T) {
	prov := &stubFactionProvider{byRegion: map[string][]localmap.FactionOverlay{
		"march": {
			{FactionName: "Iron Banner", Color: "#7A2E28", Hatch: localmap.HatchDiagonalA, Opacity: 0.12},
			{FactionName: "Merchant Guild", Color: "#2E4A62", Hatch: localmap.HatchDiagonalB, Opacity: 0.12},
		},
	}}
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}, nil)
	m.SetFactionDataProvider(prov)
	m.OpenStandalone()

	data := m.Data()
	if got := len(data.Factions["march"]); got != 2 {
		t.Fatalf("march overlays = %d, want 2 (blended wash)", got)
	}
	a := data.Factions["march"][0]
	b := data.Factions["march"][1]
	if a.Hatch == b.Hatch {
		t.Fatal("blended factions must use distinct hatch patterns")
	}
	angA := localmap.HatchAngles(a.Hatch)
	angB := localmap.HatchAngles(b.Hatch)
	if len(angA) == 0 || len(angB) == 0 {
		t.Fatal("both factions must render hatch lines")
	}
	// Each hatch direction produces clipped lines inside the region bounds.
	segsA := localmap.HatchSegments(0, 0, 40, 30, angA[0], 16)
	segsB := localmap.HatchSegments(0, 0, 40, 30, angB[0], 16)
	if len(segsA) == 0 || len(segsB) == 0 {
		t.Fatalf("hatch lines: A=%d B=%d, want both > 0", len(segsA), len(segsB))
	}
	for _, s := range append(segsA, segsB...) {
		if s.X1 < 0 || s.X1 > 40 || s.X2 < 0 || s.X2 > 40 || s.Y1 < 0 || s.Y1 > 30 || s.Y2 < 0 || s.Y2 > 30 {
			t.Fatalf("hatch segment escapes region bounds: %+v", s)
		}
	}
}

// test_unexplored_region_never_washes: Unexplored regions render no fill
// and no faction territory (AC-2: dashed outline, no fill).
func TestUnexploredRegionNeverWashes(t *testing.T) {
	if !localmap.RegionHasWash(localmap.RegionStateVisited) {
		t.Fatal("visited regions must wash")
	}
	if localmap.RegionHasWash(localmap.RegionStateUnexplored) {
		t.Fatal("unexplored regions must never wash")
	}
}

// ---------------------------------------------------------------------------
// AC-5: region detail panel
// ---------------------------------------------------------------------------

// test_region_click_shows_detail_panel: selecting a region opens the
// detail panel with name, state, faction presence, discovered locations
// count, and active threads count (AC-5).
func TestRegionClickShowsDetailPanel(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}, nil)
	m.SetFactionDataProvider(&stubFactionProvider{byRegion: map[string][]localmap.FactionOverlay{
		"march": {{FactionName: "Iron Banner", Color: "#7A2E28", Hatch: localmap.HatchDiagonalA, Opacity: 0.12}},
	}})
	m.SetLocationMarkerProvider(&stubLocationMarkerProvider{byRegion: map[string][]localmap.LocationMarkerData{
		"march": make([]localmap.LocationMarkerData, 12),
	}})
	m.SetThreadMarkerProvider(&stubThreadMarkerProvider{byRegion: map[string][]localmap.ThreadMarkerData{
		"march": make([]localmap.ThreadMarkerData, 3),
	}})
	m.OpenStandalone()

	if !m.SelectRegion("march") {
		t.Fatal("SelectRegion must succeed for a discovered region")
	}
	d, ok := m.SelectedDetail()
	if !ok {
		t.Fatal("detail panel must be open after selection")
	}
	if d.Name != "The March" || d.State != localmap.RegionStateVisited {
		t.Fatalf("detail = %+v, want The March / Visited", d)
	}
	if len(d.Factions) != 1 || d.Factions[0] != "Iron Banner" {
		t.Fatalf("faction presence = %v, want [Iron Banner]", d.Factions)
	}
	if d.Locations != 12 || d.Threads != 3 || !d.HasCounts {
		t.Fatalf("counts = %d locations / %d threads (HasCounts=%v), want 12/3/true",
			d.Locations, d.Threads, d.HasCounts)
	}
}

// test_unexplored_detail_limited_info: clicking an unexplored region shows
// name, state, and faction presence only — no location/thread counts
// (AC-5 QA edge).
func TestUnexploredDetailShowsLimitedInfo(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("thornwood", "Thornwood", localmap.RegionStateUnexplored, 0, 0),
	}, nil)
	m.SetLocationMarkerProvider(&stubLocationMarkerProvider{byRegion: map[string][]localmap.LocationMarkerData{
		"thornwood": make([]localmap.LocationMarkerData, 9),
	}})
	m.OpenStandalone()

	if !m.SelectRegion("thornwood") {
		t.Fatal("SelectRegion must succeed for an unexplored region")
	}
	d, ok := m.SelectedDetail()
	if !ok {
		t.Fatal("detail panel must be open after selection")
	}
	if d.Name != "Thornwood" || d.State != localmap.RegionStateUnexplored {
		t.Fatalf("detail = %+v, want Thornwood / Unexplored", d)
	}
	if d.HasCounts {
		t.Fatal("unexplored detail must not show counts")
	}
	if d.Locations != 0 || d.Threads != 0 {
		t.Fatalf("unexplored counts = %d/%d, want 0/0", d.Locations, d.Threads)
	}
}

// test_click_outside_dismisses_panel: dismissing the panel closes it and
// clears the selection (AC-5 QA edge).
func TestClickOutsideDismissesPanel(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}, nil)
	m.OpenStandalone()

	if !m.SelectRegion("march") {
		t.Fatal("SelectRegion must succeed")
	}
	m.DismissDetail()
	if _, ok := m.SelectedDetail(); ok {
		t.Fatal("detail panel must be dismissed")
	}
	// Selecting an unknown region is a no-op and must not reopen it.
	if m.SelectRegion("nowhere") {
		t.Fatal("SelectRegion must fail for an unknown region")
	}
	if _, ok := m.SelectedDetail(); ok {
		t.Fatal("failed selection must not open the panel")
	}
}

// test_detail_panel_fits_min_resolution: the 240px panel with 20px margins
// fits the 1280x720 minimum resolution without clipping (AC-5 QA edge).
func TestDetailPanelFitsMinimumResolution(t *testing.T) {
	const minWidth = 1280.0
	if localmap.DetailPanelWidth > minWidth-40 {
		t.Fatalf("detail panel %vpx exceeds minimum width budget %vpx", localmap.DetailPanelWidth, minWidth-40)
	}
}

// ---------------------------------------------------------------------------
// AC-6: world map close and navigation
// ---------------------------------------------------------------------------

// test_close_returns_to_local_map: closing a world map opened from the
// local map re-enables the local map's input and reopens it (AC-6).
func TestCloseReturnsToLocalMap(t *testing.T) {
	local := localmap.NewLocalMap(nil, nil, nil)
	local.SetRegion(localmap.RegionMapData{RegionID: "march", Name: "The March", Width: 250, Height: 200})
	local.Open()

	m := newHeadlessWorldMap()
	m.SetViewportSize(1920, 1080)
	m.OpenFromLocal(local)

	if !m.IsOpen() {
		t.Fatal("world map must be open after OpenFromLocal")
	}
	if local.InputEnabled() {
		t.Fatal("local map input must be suspended while the world map is up")
	}

	m.Close()
	m.Update(0.5) // complete the exit slide -> return to local
	if m.IsOpen() {
		t.Fatal("world map must be closed after the exit slide")
	}
	if !local.IsOpen() {
		t.Fatal("closing the world map must return to the local map")
	}
	if !local.InputEnabled() {
		t.Fatal("closing the world map must re-enable the local map's input")
	}
}

// test_close_standalone_returns_to_game: a world map opened standalone
// (from the main menu) closes to the game — no local map is touched
// (AC-6 QA edge).
func TestCloseStandaloneReturnsToGame(t *testing.T) {
	m := worldMapWith([]localmap.RegionSummary{
		region("march", "The March", localmap.RegionStateVisited, 0, 0),
	}, nil)
	m.OpenStandalone()
	if !m.IsOpen() {
		t.Fatal("standalone world map must be open")
	}
	m.Close()
	m.Update(0.5)
	if m.IsOpen() {
		t.Fatal("standalone world map must close")
	}
}

// test_rapid_toggle_debounced: a second M toggle within 300ms of the
// first is ignored; after the window the toggle works (AC-6 QA edge).
func TestRapidToggleDebounced(t *testing.T) {
	if localmap.ToggleDebounce != 0.3 {
		t.Fatalf("ToggleDebounce = %v, want 0.3s", localmap.ToggleDebounce)
	}
	m := worldMapWith(nil, nil)

	// Open via Toggle from closed state — Toggle arms the debounce window.
	m.Toggle()
	if !m.IsOpen() {
		t.Fatal("first toggle must open the map")
	}
	// Second toggle within the debounce window — must be ignored.
	m.Toggle()
	if !m.IsOpen() {
		t.Fatal("second toggle inside the 300ms window must be ignored — map must stay open")
	}
	// Advance past the debounce window (also completes the enter slide).
	m.Update(0.35)
	if !m.IsOpen() {
		t.Fatal("map must still be open after debounce expires")
	}
	// Now toggle should succeed in closing.
	m.Toggle()
	m.Update(0.3) // complete the exit slide
	if m.IsOpen() {
		t.Fatal("toggle after debounce window must close the map")
	}
}

// test_reduced_motion_opens_instantly: with the reduced-motion gate on,
// the world map appears instantly — no slide to advance (accessibility).
func TestReducedMotionOpensInstantly(t *testing.T) {
	m := newHeadlessWorldMap()
	m.SetReducedMotion(true)
	m.OpenStandalone()
	if !m.IsOpen() {
		t.Fatal("reduced motion must open the map instantly")
	}
	m.Close()
	if m.IsOpen() {
		t.Fatal("reduced motion must close the map instantly")
	}
}

// ---------------------------------------------------------------------------
// Geometry math shared by the renderer and hit-testing
// ---------------------------------------------------------------------------

// test_world_canvas_geometry_math: bounds, hit-testing, dashes, and hatch
// clipping — the pure geometry the renderer and the click handler build on.
func TestWorldCanvasGeometryMath(t *testing.T) {
	// WorldCanvasBounds unions the region bounds: regions at x=0 and x=60
	// (each 40 wide) span 100 units; height 30.
	r1 := region("a", "A", localmap.RegionStateVisited, 0, 0)
	r2 := region("b", "B", localmap.RegionStateVisited, 60, 0)
	minX, maxY, w, h := localmap.WorldCanvasBounds([]localmap.RegionSummary{r1, r2})
	if minX != 0 || maxY != 30 || w != 100 || h != 30 {
		t.Fatalf("canvas bounds = (%v, %v, %v, %v), want (0, 30, 100, 30)", minX, maxY, w, h)
	}
	// No regions -> degenerate bounds.
	minX, maxY, w, h = localmap.WorldCanvasBounds(nil)
	if minX != 0 || maxY != 0 || w != 0 || h != 0 {
		t.Fatalf("empty canvas bounds = (%v, %v, %v, %v), want all zero", minX, maxY, w, h)
	}

	// PointInRegion: inside, outside, and on-boundary points of the rect.
	if !localmap.PointInRegion(20, 15, r1.Boundary) {
		t.Fatal("(20, 15) must be inside the region")
	}
	if localmap.PointInRegion(50, 15, r1.Boundary) {
		t.Fatal("(50, 15) must be outside the region")
	}
	if !localmap.PointInRegion(40, 15, r1.Boundary) {
		t.Fatal("a point on the boundary must count as inside")
	}

	// DashSegments splits a 40-unit segment into 10/6 dashes: 3 dashes,
	// the last one truncated at the endpoint.
	dashes := localmap.DashSegments(localmap.LineSegment{X1: 0, Y1: 0, X2: 40, Y2: 0}, 10, 6)
	if len(dashes) != 3 {
		t.Fatalf("dashes = %d, want 3", len(dashes))
	}
	if dashes[0].X1 != 0 || dashes[0].X2 != 10 || dashes[2].X2 != 40 {
		t.Fatalf("dash run = %+v, want 0-10 ... ending exactly at 40", dashes)
	}

	// ClipSegmentToRect clips a crossing line to the rect (Liang-Barsky).
	clipped, ok := localmap.ClipSegmentToRect(
		localmap.LineSegment{X1: -10, Y1: 15, X2: 50, Y2: 15}, 0, 0, 40, 30)
	if !ok {
		t.Fatal("crossing segment must clip successfully")
	}
	if clipped.X1 != 0 || clipped.X2 != 40 {
		t.Fatalf("clipped segment = %+v, want x in [0, 40]", clipped)
	}
	if _, ok := localmap.ClipSegmentToRect(
		localmap.LineSegment{X1: 100, Y1: 100, X2: 110, Y2: 110}, 0, 0, 40, 30); ok {
		t.Fatal("segment fully outside the rect must not clip")
	}
}

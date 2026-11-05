// World map half of the map package (Story 003: World Map & Region Travel,
// TR-ui-001/003, TR-faction-001). The world map is the desk-register
// overview above the local map: every discovered region rendered as an
// ink-drawn boundary shape on parchment, state summaries (Visited/Cleared/
// Unexplored), dotted ink travel routes between discovered regions, faction
// territory washes with shape-distinct hatch patterns, and a region detail
// panel that paper-slides in from the side (AC-1 through AC-6).
//
// The design mirrors the local map exactly: engine-free models own all
// state (WorldMap controller + WorldMapData snapshot), the worldMapRenderer
// translates them into Kaiju document operations, and the paper-slide/dim
// motion reuses the package Tween. Cross-system data arrives through
// provider interfaces the game layer implements (WorldRegionProvider,
// FactionDataProvider, plus the Story 002 Location/ThreadMarkerProviders for
// the detail panel counts) — the map never reads entity stores and never
// mutates game state. Headless mode (nil host and ui.Manager) keeps the
// state machine, navigation, geometry math, and debounce fully active for
// the integration tests.
//
// Engine constraint: Kaiju CSS implements no clip-path and gradient
// functions error out as "not implemented", so region fills, faction
// washes, and hatch lines are clipped to each region's bounding box in Go
// (ClipSegmentToRect, HatchSegments) rather than to the authored polygon.
// Boundary linework is the authored shape; the fill/wash sits behind it
// within the bounds rect. The Unexplored dashed outline is produced
// Go-side by splitting boundary segments (DashSegments) — no reliance on
// shader border dashing.
package local_map

import (
	"fmt"
	"log/slog"
	"math"
	"slices"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/platform/hid"
)

// WorldMapScale maps world-canvas units to surface pixels (4 px per unit).
// World-map units are abstract canvas units authored per region, distinct
// from the local map's meter scale.
const WorldMapScale = 4.0

// canvasPadding is the parchment margin (px) around the world canvas — the
// world always sits inside a frame of parchment, even for a single small
// region.
const canvasPadding = 80.0

// ToggleDebounce guards against rapid M/Esc double-taps (AC-6 QA edge): the
// world map accepts only one toggle per 300ms.
const ToggleDebounce = 0.3 // seconds

// Detail panel geometry (AC-5): a 240px parchment sheet paper-sliding in
// from the right edge, with 20px margins — it fits the 1280x720 minimum
// resolution without clipping.
const (
	DetailPanelWidth  = 240.0
	detailPanelMargin = 20.0
)

// Faction wash opacity bounds (AC-4): the settled wash is 10-15% — subtle,
// never hiding the ink linework. Values outside the band are clamped.
const (
	FactionWashMinOpacity = 0.10
	FactionWashMaxOpacity = 0.15
)

// Ink linework geometry for the world map. Dash and dot lengths are in
// surface px.
const (
	dashLength    = 10.0 // px — Unexplored boundary dashes
	dashGap       = 6.0  // px — gap between dashes
	routeSpacing  = 18.0 // px — dot spacing along travel routes
	routeDotLen   = 6.0  // px — one route dot's length
	hatchSpacing  = 16.0 // px — spacing between faction hatch lines
	hatchThickness = 1.5 // px — hatch stroke weight
)

// RegionState is a discovered region's state summary (AC-2): the player
// visited it, cleared every major location, or knows it exists but has
// never entered it.
type RegionState int

const (
	// RegionStateVisited: normal ink, solid outline, faint interior wash.
	RegionStateVisited RegionState = iota
	// RegionStateCleared: faded ink, faint wash, ink X over the region.
	RegionStateCleared
	// RegionStateUnexplored: dashed outline, no fill, no interior detail.
	RegionStateUnexplored
)

// RegionStateName returns the display name of a region state ("Visited",
// "Cleared", "Unexplored") — the detail panel's state row (AC-5).
func RegionStateName(s RegionState) string {
	switch s {
	case RegionStateCleared:
		return "Cleared"
	case RegionStateUnexplored:
		return "Unexplored"
	default:
		return "Visited"
	}
}

// RegionStateInk returns the outline treatment token for a region state
// (AC-2): "solid" for Visited, "faint" for Cleared, "dashed" for
// Unexplored. The renderer maps tokens to ink colors and dash
// segmentation; the token keeps the state->style contract testable.
func RegionStateInk(s RegionState) string {
	switch s {
	case RegionStateCleared:
		return "faint"
	case RegionStateUnexplored:
		return "dashed"
	default:
		return "solid"
	}
}

// RegionBoundaryInk returns the boundary stroke color and whether the
// outline is dashed for a region state. Cleared boundaries fade to 45%
// ink; Unexplored boundaries are full ink but dashed (DashSegments);
// Visited is full ink and solid (AC-2).
func RegionBoundaryInk(s RegionState) (color string, dashed bool) {
	switch s {
	case RegionStateCleared:
		return "rgba(28, 24, 19, 0.45)", false
	case RegionStateUnexplored:
		return "#1C1813", true
	default:
		return "#1C1813", false
	}
}

// RegionWashColor returns the faint interior ink wash for a region state,
// or "" when the state has no fill. Unexplored regions never wash (AC-2:
// dashed outline, no fill).
func RegionWashColor(s RegionState) string {
	switch s {
	case RegionStateCleared:
		return "rgba(28, 24, 19, 0.04)"
	case RegionStateVisited:
		return "rgba(28, 24, 19, 0.06)"
	default:
		return ""
	}
}

// RegionHasWash reports whether a region state renders an interior fill
// and faction territory (AC-2: Unexplored has no fill; AC-4 washes sit
// within region boundaries).
func RegionHasWash(s RegionState) bool { return RegionWashColor(s) != "" }

// RegionSummary is one discovered region on the world map: its stable ID,
// display name, state, and authored boundary polygon in world-canvas units
// (y-up). Boundaries are closed polygons of LineSegments; the map renders
// them as ink linework (Story 001 convention).
type RegionSummary struct {
	RegionID string
	Name     string
	State    RegionState
	Boundary []LineSegment
}

// TravelRoute is an authored route between two regions (AC-3). Routes are
// world data supplied by the region provider; the map shows a route only
// when both endpoint regions are discovered (VisibleRoutes).
type TravelRoute struct {
	FromRegion string
	ToRegion   string
}

// HatchPattern is the shape distinction of a faction's territory wash —
// never hue alone (AC-4, control manifest: never encode game-critical
// information by hue alone). Each faction is authored a pattern; two
// factions sharing a region render different directions so both remain
// visible to colorblind players.
type HatchPattern int

const (
	// HatchNone renders no hatch lines.
	HatchNone HatchPattern = iota
	// HatchDiagonalA renders 45° lines.
	HatchDiagonalA
	// HatchDiagonalB renders -45° lines.
	HatchDiagonalB
	// HatchCross renders both directions.
	HatchCross
)

// HatchAngles returns the line angles (degrees, screen convention) for a
// hatch pattern: one direction for the diagonal patterns, both for cross,
// none for HatchNone.
func HatchAngles(p HatchPattern) []float64 {
	switch p {
	case HatchDiagonalA:
		return []float64{45}
	case HatchDiagonalB:
		return []float64{-45}
	case HatchCross:
		return []float64{45, -45}
	default:
		return nil
	}
}

// FactionOverlay is one faction's territory in one region (AC-4): a
// colored wash at low opacity with a shape-distinct hatch pattern. Color
// comes from the faction palette (game layer); the map clamps the settled
// opacity into the 10-15% band (FactionWashOpacity).
type FactionOverlay struct {
	RegionID    string
	FactionName string
	Color       string // CSS color from the faction palette
	Hatch       HatchPattern
	Opacity     float64 // requested wash opacity, clamped to 0.10-0.15
}

// WorldRegionProvider supplies the world map's region data (AC-1): all
// discovered regions with their state summaries and boundaries, plus the
// authored travel routes. The game layer implements it from persistent
// world state (discovery flags) and world data; the map renders exactly
// what it returns — an undiscovered region is simply absent.
type WorldRegionProvider interface {
	DiscoveredRegions() []RegionSummary
	TravelRoutes() []TravelRoute
}

// FactionDataProvider supplies faction territory overlays per region
// (AC-4). The game layer implements it from faction standings (threshold
// filtering is its call — a faction below the presence threshold is
// omitted); the map depends on the interface and renders exactly what it
// returns.
type FactionDataProvider interface {
	FactionOverlays(regionID string) []FactionOverlay
}

// RegionLoader supplies the local-map geography of a region for the detail
// panel's zoom-in (Navigation: click region -> detail panel -> "Zoom to
// map" opens that region's local map). The game layer implements it from
// the scene pipeline; the ok result is false when the region has no local
// data, in which case zoom is a no-op.
type RegionLoader func(regionID string) (RegionMapData, bool)

// WorldMapData is the world map's current state: the discovered regions,
// the authored travel routes, and the faction overlays per region,
// exactly as rendered (the snapshot the map holds after querying its
// providers). Factions is keyed by region ID; a region with no faction
// presence has no entry (AC-4: zero factions above threshold -> no
// overlay).
type WorldMapData struct {
	Regions  []RegionSummary
	Routes   []TravelRoute
	Factions map[string][]FactionOverlay
}

// RegionDetail is the detail panel's content for a selected region
// (AC-5): name, state, faction presence names, discovered locations
// count, and active threads count. HasCounts is false for Unexplored
// regions — their panel shows name, state, and faction presence only (QA
// edge: no location/thread counts for regions the player never entered).
type RegionDetail struct {
	RegionID  string
	Name      string
	State     RegionState
	Factions  []string
	Locations int
	Threads   int
	HasCounts bool
}

// WorldMap is the world map entry point: it owns the region data
// snapshot, the canvas viewport, the paper-slide/dim tweens, the detail
// panel state, the navigation return target (the local map it zoomed out
// from), and the renderer, and polls the M/Esc hotkeys, region clicks,
// and pan input per frame.
type WorldMap struct {
	host     *engine.Host
	renderer *worldMapRenderer

	regionProvider   WorldRegionProvider
	factionProvider  FactionDataProvider
	locationProvider LocationMarkerProvider // Story 002 providers, reused for AC-5 counts
	threadProvider   ThreadMarkerProvider
	loader           RegionLoader

	// data is the snapshot of all providers at the last Open/Refresh —
	// the world map's WorldMapData, exposed via Data.
	data WorldMapData

	scale float64 // px per world-canvas unit

	// World-canvas origin and surface extent (px). canvasMinX/canvasMaxY
	// anchor the snapshot's bounds in world-canvas units; canvasW/canvasH
	// are the surface px extent including canvasPadding.
	canvasMinX, canvasMaxY float64
	canvasW, canvasH       float64

	// Viewport offset of the surface inside the parchment (px), clamped by
	// ClampViewport so the canvas always covers the viewport (or is
	// centered when smaller — AC-1 single-region case).
	offsetX, offsetY float64

	viewportW, viewportH float64 // cached window size (px)

	state         mapState
	slide         *Tween // paper slide (translateY on the root)
	dim           *Tween // world dim (opacity on the dim layer)
	reducedMotion bool

	// Detail panel state (AC-5): detail is the selected region's content,
	// detailOpen whether the panel is on screen, detailSlide the paper
	// slide-in/out from the right edge.
	detailOpen  bool
	detail      RegionDetail
	detailSlide *Tween

	// Pan state: left-button drag follows the cursor (same as LocalMap).
	dragging bool
	lastMX   float64
	lastMY   float64

	// returnTo is the local map the world map zoomed out from (AC-6): on
	// close, its input is re-enabled and it is reopened. Nil when the
	// world map opened standalone (from the main menu).
	returnTo *LocalMap

	// debounceRemaining counts down the ToggleDebounce window after any
	// toggle; hotkey toggles within the window are ignored (AC-6 QA edge).
	debounceRemaining float64

	updateID engine.UpdateId
}

// NewWorldMap loads the map markup (the world overlay shares
// assets/ui/map.html with the local map, under its own root), binds the
// world elements, parks the overlay off-screen, and registers the
// per-frame update.
//
// Headless mode: pass nil host and nil ui.Manager (as the integration
// tests do) to skip markup loading and the per-frame update while keeping
// the state machine, navigation, geometry math, and debounce fully
// active. Providers may all be nil — the map then renders an empty
// parchment with the hint text.
func NewWorldMap(host *engine.Host, man *ui.Manager) *WorldMap {
	m := &WorldMap{
		host:  host,
		scale: WorldMapScale,
	}
	m.renderer = newWorldMapRenderer(nil)

	if host != nil && man != nil {
		doc, err := markup.DocumentFromHTMLAsset(man, "ui/map.html", nil, nil)
		if err != nil {
			slog.Warn("world map: failed to load markup", "err", err)
			return m
		}
		m.renderer = newWorldMapRenderer(doc)
		m.renderer.bind()
		m.viewportW = float64(host.Window.Width())
		m.viewportH = float64(host.Window.Height())
		m.renderer.applyFontScale(HeightScaleFactor(m.viewportH))
		m.updateID = host.Updater.AddUpdate(m.Update)
	}
	return m
}

// Providers --------------------------------------------------------------

// SetWorldRegionProvider installs the discovered-region and travel-route
// source (AC-1, AC-3). nil clears the map. The provider is snapshotted on
// Open and Refresh.
func (m *WorldMap) SetWorldRegionProvider(p WorldRegionProvider) { m.regionProvider = p }

// SetFactionDataProvider installs the faction territory source (AC-4).
// nil disables faction washes.
func (m *WorldMap) SetFactionDataProvider(p FactionDataProvider) { m.factionProvider = p }

// SetLocationMarkerProvider installs the discovered-location source for
// the detail panel's location count (AC-5) — the same Story 002 provider
// the local map uses. nil shows a zero count.
func (m *WorldMap) SetLocationMarkerProvider(p LocationMarkerProvider) { m.locationProvider = p }

// SetThreadMarkerProvider installs the active-thread source for the
// detail panel's thread count (AC-5) — the same Story 002 provider the
// local map uses. nil shows a zero count.
func (m *WorldMap) SetThreadMarkerProvider(p ThreadMarkerProvider) { m.threadProvider = p }

// SetRegionLoader installs the local-geography source for the detail
// panel's "Zoom to map" navigation. nil disables zoom.
func (m *WorldMap) SetRegionLoader(l RegionLoader) { m.loader = l }

// State machine ----------------------------------------------------------

// OpenFromLocal opens the world map on top of an open local map (AC-6
// navigation: zoom out from the local map). The local map's input is
// suspended while the world map is up so M cannot toggle both overlays in
// one frame; closing the world map re-enables it and reopens the local
// map (already-open is a no-op).
func (m *WorldMap) OpenFromLocal(local *LocalMap) {
	if local == nil {
		m.OpenStandalone()
		return
	}
	m.returnTo = local
	local.SetInputEnabled(false)
	m.open()
}

// OpenStandalone opens the world map directly (from the main menu, AC-6):
// closing returns to the game, not to a local map. Any previous local-map
// return target is released and its input re-enabled.
func (m *WorldMap) OpenStandalone() {
	if m.returnTo != nil {
		m.returnTo.SetInputEnabled(true)
		m.returnTo = nil
	}
	m.open()
}

// open begins the enter sequence: refresh the provider snapshot, then
// paper-slide the parchment up (250ms ease-out) while the world dims to
// 30% (desk register, AC-1). No-op when already entering or open.
func (m *WorldMap) open() {
	if m.state == stateEntering || m.state == stateOpen {
		return
	}
	m.refresh()
	m.state = stateEntering
	m.renderer.show()
	m.renderer.applySlide(-m.viewportH)
	m.renderer.applyDim(0)
	if m.reducedMotion {
		m.state = stateOpen
		m.renderer.applySlide(0)
		m.renderer.applyDim(worldDimOpacity)
		return
	}
	m.slide = NewPaperSlideTween(-m.viewportH, 0, slideEnterDuration)
	m.dim = NewPaperSlideTween(0, worldDimOpacity, slideEnterDuration)
}

// Close begins the exit sequence (AC-6): the parchment slides out below
// the screen edge and the world brightens (both 200ms). The detail panel
// is dropped immediately. On completion the return target — the local
// map the world map zoomed out from — has its input re-enabled and is
// reopened; standalone mode returns to the game.
func (m *WorldMap) Close() {
	if m.state == stateExiting || m.state == stateClosed {
		return
	}
	m.state = stateExiting
	m.detailOpen = false
	m.detailSlide = nil
	m.renderer.hideDetail()
	if m.reducedMotion {
		m.finishClose()
		return
	}
	m.slide = NewPaperSlideTween(0, -m.viewportH, slideExitDuration)
	m.dim = NewPaperSlideTween(worldDimOpacity, 0, slideExitDuration)
}

// Toggle opens a closed (or closing) world map and closes an open (or
// entering) one — the M-key behavior (AC-1/AC-6). Toggles are debounced:
// a second toggle within ToggleDebounce of the first is ignored (AC-6 QA
// edge: rapid M double-tap).
func (m *WorldMap) Toggle() {
	if m.debounceRemaining > 0 {
		return
	}
	m.debounceRemaining = ToggleDebounce
	if m.state == stateClosed || m.state == stateExiting {
		m.open()
		return
	}
	m.Close()
}

// finishClose completes the exit: park the overlay, then restore the
// return target (AC-6).
func (m *WorldMap) finishClose() {
	m.state = stateClosed
	m.renderer.applySlide(0)
	m.renderer.applyDim(0)
	m.renderer.hide()
	if m.returnTo != nil {
		m.returnTo.SetInputEnabled(true)
		m.returnTo.Open()
	}
}

// IsOpen reports whether the world map is on screen (entering, open, or
// exiting). The map is interactive only in the open state.
func (m *WorldMap) IsOpen() bool { return m.state != stateClosed }

// ViewportOffset returns the current surface offset (px).
func (m *WorldMap) ViewportOffset() (float64, float64) { return m.offsetX, m.offsetY }

// IsEmpty reports whether the world map has no discovered regions (AC-1
// edge: empty parchment with hint text).
func (m *WorldMap) IsEmpty() bool { return len(m.data.Regions) == 0 }

// Data returns the world map's current snapshot: the discovered regions,
// travel routes, and faction overlays, exactly as rendered. The returned
// data is a deep copy — the caller can mutate it without affecting the
// map's rendered layer.
func (m *WorldMap) Data() WorldMapData {
	factions := make(map[string][]FactionOverlay, len(m.data.Factions))
	for id, overlays := range m.data.Factions {
		factions[id] = slices.Clone(overlays)
	}
	return WorldMapData{
		Regions:  slices.Clone(m.data.Regions),
		Routes:   slices.Clone(m.data.Routes),
		Factions: factions,
	}
}

// VisibleRoutes returns the travel routes that are shown (AC-3): routes
// whose endpoints are both discovered regions. Routes touching an
// undiscovered region are hidden — the map filters; the provider authors
// all routes.
func (m *WorldMap) VisibleRoutes() []TravelRoute {
	return VisibleRoutes(m.data.Routes, m.data.Regions)
}

// SelectedDetail returns the detail panel's current content and whether
// the panel is open (AC-5). The returned slice is a copy.
func (m *WorldMap) SelectedDetail() (RegionDetail, bool) {
	if !m.detailOpen {
		return RegionDetail{}, false
	}
	return RegionDetail{
		RegionID:  m.detail.RegionID,
		Name:      m.detail.Name,
		State:     m.detail.State,
		Factions:  slices.Clone(m.detail.Factions),
		Locations: m.detail.Locations,
		Threads:   m.detail.Threads,
		HasCounts: m.detail.HasCounts,
	}, true
}

// SelectRegion opens the detail panel for a discovered region (AC-5):
// name, state, faction presence (from the snapshot), discovered locations
// and active threads counts (from the Story 002 providers — Unexplored
// regions show no counts). Returns false when the region is not in the
// snapshot.
func (m *WorldMap) SelectRegion(regionID string) bool {
	region, ok := m.findRegion(regionID)
	if !ok {
		return false
	}
	d := RegionDetail{RegionID: region.RegionID, Name: region.Name, State: region.State}
	for _, o := range m.data.Factions[regionID] {
		d.Factions = append(d.Factions, o.FactionName)
	}
	if region.State != RegionStateUnexplored {
		d.HasCounts = true
		if m.locationProvider != nil {
			d.Locations = len(m.locationProvider.DiscoveredLocations(regionID))
		}
		if m.threadProvider != nil {
			d.Threads = len(m.threadProvider.ActiveThreads(regionID))
		}
	}
	m.detail = d
	m.detailOpen = true
	m.renderer.applyDetail(d)
	m.renderer.showDetail()
	m.renderer.applyDetailSlide(DetailPanelWidth)
	if m.reducedMotion {
		m.renderer.applyDetailSlide(0)
		return true
	}
	m.detailSlide = NewPaperSlideTween(DetailPanelWidth, 0, slideEnterDuration)
	return true
}

// DismissDetail slides the detail panel out (AC-5 QA edge: click outside
// the panel dismisses). No-op when the panel is not open.
func (m *WorldMap) DismissDetail() {
	if !m.detailOpen {
		return
	}
	m.detailOpen = false
	if m.reducedMotion {
		m.renderer.hideDetail()
		return
	}
	from := DetailPanelWidth
	if m.detailSlide != nil {
		from = m.detailSlide.Value()
		m.detailSlide = nil
	}
	m.detailSlide = NewPaperSlideTween(from, DetailPanelWidth, slideExitDuration)
}

// Refresh re-queries every provider and rebuilds the world map — the
// game layer calls it when knowledge changes while the map is open (AC-2
// QA edge: a region transitions Visited -> Cleared; AC-4 edge: faction
// presence changes). Also called automatically on Open.
func (m *WorldMap) Refresh() { m.refresh() }

// SetViewportSize updates the cached viewport size and re-clamps the
// offset (window resize support; the per-frame Update also refreshes the
// cache from the host window).
func (m *WorldMap) SetViewportSize(w, h float64) {
	m.viewportW, m.viewportH = w, h
	m.reclamp()
}

// SetReducedMotion enables or disables the reduced-motion gate: while on,
// paper-slide and dim transitions complete instantly (accessibility
// requirement; the game layer wires it from the accessibility settings).
func (m *WorldMap) SetReducedMotion(on bool) { m.reducedMotion = on }

// Pan shifts the viewport by (dx, dy) surface px and re-clamps — the
// shared path for drag and edge-scroll (AC-1: 20+ regions scroll).
func (m *WorldMap) Pan(dx, dy float64) {
	m.offsetX, m.offsetY = ClampViewport(m.offsetX+dx, m.offsetY+dy, m.canvasW, m.canvasH, m.viewportW, m.viewportH)
	m.renderer.applySurface(m.offsetX, m.offsetY)
}

// Destroy tears down the world map: removes the per-frame update and
// destroys the markup document. The instance must not be used afterwards.
func (m *WorldMap) Destroy() {
	if m.host != nil {
		m.host.Updater.RemoveUpdate(&m.updateID)
	}
	if m.renderer != nil {
		m.renderer.destroy()
	}
	m.host = nil
	m.regionProvider = nil
	m.factionProvider = nil
	m.locationProvider = nil
	m.threadProvider = nil
	m.loader = nil
	m.returnTo = nil
}

// Per-frame update --------------------------------------------------------

// Update advances the slides and the dim, routes the M/Esc hotkeys, and —
// while the map is fully open — polls region clicks, the detail panel,
// and the pan input (drag + edge-scroll). Registered on the host updater
// by NewWorldMap; callable directly in headless tests.
func (m *WorldMap) Update(dt float64) {
	if m.host != nil {
		m.viewportW = float64(m.host.Window.Width())
		m.viewportH = float64(m.host.Window.Height())
	}
	m.renderer.applyFontScale(HeightScaleFactor(m.viewportH))
	if m.debounceRemaining > 0 {
		m.debounceRemaining -= dt
	}
	m.updateTweens(dt)
	m.pollHotkeys()
	if m.state == stateOpen {
		m.pollClicks()
		m.updatePan(dt)
	}
}

// updateTweens advances the paper slide, the world dim, and the detail
// panel slide together, completing the state transitions: entering ->
// open when the slide-in finishes, exiting -> closed (return target
// restored) when the slide-out finishes, detail slide-out -> panel hidden.
func (m *WorldMap) updateTweens(dt float64) {
	if m.slide != nil {
		m.slide.Update(dt)
		m.dim.Update(dt)
		m.renderer.applySlide(m.slide.Value())
		m.renderer.applyDim(m.dim.Value())
		if m.slide.Done() {
			m.slide, m.dim = nil, nil
			switch m.state {
			case stateEntering:
				m.state = stateOpen
			case stateExiting:
				m.finishClose()
			}
		}
	}
	if m.detailSlide != nil {
		m.detailSlide.Update(dt)
		m.renderer.applyDetailSlide(m.detailSlide.Value())
		if m.detailSlide.Done() {
			m.detailSlide = nil
			if !m.detailOpen {
				m.renderer.hideDetail()
			}
		}
	}
}

// pollHotkeys routes keyboard input (AC-1/AC-6): M toggles the world map
// (debounced — Toggle enforces ToggleDebounce), Esc closes it while it is
// on screen. KeyDown is edge-triggered in the engine, so a held key
// cannot toggle repeatedly. No-op without a host (headless mode — tests
// drive Open/Close/Toggle directly).
func (m *WorldMap) pollHotkeys() {
	if m.host == nil {
		return
	}
	kb := &m.host.Window.Keyboard
	if kb.KeyDown(hid.KeyboardKeyM) {
		m.Toggle()
	}
	if kb.KeyDown(hid.KeyboardKeyEscape) && m.IsOpen() {
		m.Close()
	}
}

// pollClicks routes mouse input while the map is open (AC-5): with the
// detail panel up, a click inside the panel hits the "Zoom to map"
// button, a click outside dismisses the panel; with no panel, a click
// selects the topmost region under the cursor. No-op without a host.
func (m *WorldMap) pollClicks() {
	if m.host == nil {
		return
	}
	mouse := &m.host.Window.Mouse
	if !mouse.Pressed(hid.MouseButtonLeft) {
		return
	}
	pos := mouse.ScreenPosition()
	mx, my := float64(pos.X()), float64(pos.Y())
	if m.detailOpen {
		if m.inDetailPanel(mx, my) {
			if m.inZoomButton(mx, my) {
				m.zoomToSelected()
			}
			return
		}
		m.DismissDetail()
		return
	}
	m.selectRegionAt(mx, my)
}

// selectRegionAt hit-tests the region under a window-space click (in
// surface coordinates, the same y-down convention the renderer uses) and
// opens its detail panel. Regions are tested topmost-first (reverse
// render order) so an overlapping region drawn later wins.
func (m *WorldMap) selectRegionAt(mx, my float64) {
	sx, sy := mx-m.offsetX, my-m.offsetY
	cx, cy := m.surfaceToCanvas(sx, sy)
	for i := len(m.data.Regions) - 1; i >= 0; i-- {
		if PointInRegion(cx, cy, m.data.Regions[i].Boundary) {
			m.SelectRegion(m.data.Regions[i].RegionID)
			return
		}
	}
}

// zoomToSelected closes the world map back to the selected region's local
// map (Navigation: click region -> detail panel -> "Zoom to map"). The
// region loader supplies the local geography; without a loader, a return
// target, or local data for the region, zoom is a no-op (panel
// dismissed).
func (m *WorldMap) zoomToSelected() {
	if !m.detailOpen {
		return
	}
	if m.loader == nil || m.returnTo == nil {
		m.DismissDetail()
		return
	}
	data, ok := m.loader(m.detail.RegionID)
	if !ok {
		slog.Warn("world map: no local data for zoom target", "region", m.detail.RegionID)
		m.DismissDetail()
		return
	}
	m.returnTo.SetRegion(data)
	m.Close()
}

// inDetailPanel reports whether a window-space point falls inside the
// open detail panel's rect (click-outside dismissal, AC-5 QA edge).
func (m *WorldMap) inDetailPanel(mx, my float64) bool {
	x := m.viewportW - DetailPanelWidth - detailPanelMargin
	return mx >= x && mx <= x+DetailPanelWidth && my >= detailPanelMargin && my <= m.viewportH-detailPanelMargin
}

// inZoomButton reports whether a window-space point falls inside the
// detail panel's "Zoom to map" button (the bottom row of the panel).
func (m *WorldMap) inZoomButton(mx, my float64) bool {
	x := m.viewportW - DetailPanelWidth - detailPanelMargin + 12
	y := m.viewportH - detailPanelMargin - zoomButtonHeight - 10
	return mx >= x && mx <= x+DetailPanelWidth-24 && my >= y && my <= y+zoomButtonHeight
}

// updatePan drives the pan input while the map is open (AC-1: scrolling
// for 20+ regions): mouse drag (left button held, viewport follows the
// cursor delta) and edge-scroll (cursor within 20px of a viewport edge
// scrolls that way at 300px/s) — the same conventions as the local map.
func (m *WorldMap) updatePan(dt float64) {
	if m.host == nil {
		return
	}
	mouse := &m.host.Window.Mouse
	pos := mouse.ScreenPosition()
	mx, my := float64(pos.X()), float64(pos.Y())

	if mouse.Pressed(hid.MouseButtonLeft) {
		m.dragging = true
	}
	if mouse.Released(hid.MouseButtonLeft) {
		m.dragging = false
	}
	if m.dragging && mouse.Held(hid.MouseButtonLeft) {
		m.Pan(mx-m.lastMX, my-m.lastMY)
	}
	m.lastMX, m.lastMY = mx, my

	m.edgeScroll(dt)
}

// edgeScroll pans the viewport while the cursor sits in the 20px edge
// zones, at 300px/s — identical conventions to the local map (AC-1
// scrollable world canvas).
func (m *WorldMap) edgeScroll(dt float64) {
	if m.host == nil {
		return
	}
	mouse := &m.host.Window.Mouse
	pos := mouse.ScreenPosition()
	mx, my := float64(pos.X()), float64(pos.Y())

	dx, dy := 0.0, 0.0
	if mx < edgeScrollMargin {
		dx = -edgeScrollSpeed * dt
	}
	if mx > m.viewportW-edgeScrollMargin {
		dx = edgeScrollSpeed * dt
	}
	if my < edgeScrollMargin {
		dy = edgeScrollSpeed * dt
	}
	if my > m.viewportH-edgeScrollMargin {
		dy = -edgeScrollSpeed * dt
	}
	if dx != 0 || dy != 0 {
		m.Pan(dx, dy)
	}
}

// Data refresh ------------------------------------------------------------

// refresh snapshots every provider into m.data and rebuilds the rendered
// layer: regions, visible routes, faction washes, and the world canvas
// extent. The viewport re-clamps to the new canvas.
func (m *WorldMap) refresh() {
	m.data = WorldMapData{Factions: map[string][]FactionOverlay{}}
	if m.regionProvider != nil {
		m.data.Regions = slices.Clone(m.regionProvider.DiscoveredRegions())
		m.data.Routes = slices.Clone(m.regionProvider.TravelRoutes())
	}
	if m.factionProvider != nil {
		for _, r := range m.data.Regions {
			overlays := slices.Clone(m.factionProvider.FactionOverlays(r.RegionID))
				if len(overlays) > 0 {
					m.data.Factions[r.RegionID] = overlays
				}
		}
	}
	minX, maxY, w, h := WorldCanvasBounds(m.data.Regions)
	m.canvasMinX, m.canvasMaxY = minX, maxY
	m.canvasW = w*m.scale + 2*canvasPadding
	m.canvasH = h*m.scale + 2*canvasPadding
	m.renderer.rebuild(m.data, m.scale, m.canvasW, m.canvasH, minX, maxY)
	m.reclamp()
}

// findRegion looks a region up in the snapshot by ID.
func (m *WorldMap) findRegion(regionID string) (RegionSummary, bool) {
	for _, r := range m.data.Regions {
		if r.RegionID == regionID {
			return r, true
		}
	}
	return RegionSummary{}, false
}

// Geometry ----------------------------------------------------------------

// canvasToSurface maps world-canvas coordinates (y-up) to surface pixels
// (y-down, north up): the snapshot bounds anchor the canvas inside the
// parchment padding.
func (m *WorldMap) canvasToSurface(x, y float64) (float64, float64) {
	return canvasPadding + (x-m.canvasMinX)*m.scale,
		canvasPadding + (m.canvasMaxY-y)*m.scale
}

// surfaceToCanvas maps surface pixels (y-down) back to world-canvas
// coordinates (y-up) — the inverse of canvasToSurface, used for click
// hit-testing.
func (m *WorldMap) surfaceToCanvas(sx, sy float64) (float64, float64) {
	return m.canvasMinX + (sx-canvasPadding)/m.scale,
		m.canvasMaxY - (sy-canvasPadding)/m.scale
}

// reclamp re-applies the viewport clamp after a resize or refresh.
func (m *WorldMap) reclamp() {
	m.offsetX, m.offsetY = ClampViewport(m.offsetX, m.offsetY, m.canvasW, m.canvasH, m.viewportW, m.viewportH)
	m.renderer.applySurface(m.offsetX, m.offsetY)
}

// World geometry math -----------------------------------------------------

// RegionBounds returns the axis-aligned bounds of a region's boundary
// polygon in world-canvas units. A region with no boundary segments has
// zero bounds.
func RegionBounds(r RegionSummary) (minX, minY, w, h float64) {
	if len(r.Boundary) == 0 {
		return 0, 0, 0, 0
	}
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, s := range r.Boundary {
		minX = math.Min(minX, math.Min(s.X1, s.X2))
		maxX = math.Max(maxX, math.Max(s.X1, s.X2))
		minY = math.Min(minY, math.Min(s.Y1, s.Y2))
		maxY = math.Max(maxY, math.Max(s.Y1, s.Y2))
	}
	return minX, minY, maxX - minX, maxY - minY
}

// WorldCanvasBounds returns the union bounds of all discovered regions in
// world-canvas units (AC-1): the world canvas extent, with maxY (the
// canvas top, y-up) returned directly so the surface conversion can flip
// once. With no regions the bounds are (0, 0, 0, 0).
func WorldCanvasBounds(regions []RegionSummary) (minX, maxY, w, h float64) {
	if len(regions) == 0 {
		return 0, 0, 0, 0
	}
	first := true
	for _, r := range regions {
		rx, ry, rw, rh := RegionBounds(r)
		if rw == 0 || rh == 0 {
			continue
		}
		if first {
			minX, maxY, w, h = rx, ry+rh, rw, rh
			first = false
			continue
		}
		loX := math.Min(minX, rx)
		hiX := math.Max(minX+w, rx+rw)
		loY := math.Min(maxY-h, ry)
		hiY := math.Max(maxY, ry+rh)
		minX, maxY, w, h = loX, hiY, hiX-loX, hiY-loY
	}
	if first {
		return 0, 0, 0, 0
	}
	return minX, maxY, w, h
}

// RegionCentroid returns the center of a region's bounds — the anchor for
// the ink X over Cleared regions (AC-2) and the travel-route endpoints
// (AC-3).
func RegionCentroid(r RegionSummary) (float64, float64) {
	minX, minY, w, h := RegionBounds(r)
	return minX + w/2, minY + h/2
}

// PointInRegion reports whether the point (x, y) in world-canvas units
// lies inside the region's boundary polygon (AC-5 hit-testing): standard
// ray-casting across the boundary segments, counting crossings of a ray
// cast in +x. Points exactly on a boundary segment are considered inside.
func PointInRegion(x, y float64, boundary []LineSegment) bool {
	inside := false
	for _, s := range boundary {
		if (s.Y1 > y) != (s.Y2 > y) {
			xInt := s.X1 + (y-s.Y1)/(s.Y2-s.Y1)*(s.X2-s.X1)
			if xInt >= x {
				inside = !inside
			}
		}
	}
	return inside
}

// DashSegments splits one line segment into a dashed run (AC-2: the
// Unexplored outline): dashes of length dash alternating with gaps of
// length gap, starting with a dash at the segment start. All lengths in
// the same units as the segment.
func DashSegments(s LineSegment, dash, gap float64) []LineSegment {
	dx, dy := s.X2-s.X1, s.Y2-s.Y1
	length := math.Hypot(dx, dy)
	if length <= 0 || dash <= 0 {
		return nil
	}
	ux, uy := dx/length, dy/length
	step := dash + gap
	var out []LineSegment
	for d := 0.0; d < length; d += step {
		from := math.Min(d, length)
		to := math.Min(d+dash, length)
		if to-from < 0.5 {
			break
		}
		out = append(out, LineSegment{
			X1: s.X1 + ux*from, Y1: s.Y1 + uy*from,
			X2: s.X1 + ux*to, Y2: s.Y1 + uy*to,
		})
	}
	return out
}

// RouteDotSegments returns the dots of a dotted travel route (AC-3): one
// short segment per spacing interval along the line from (x1, y1) to
// (x2, y2), each dot centered on its interval point and aligned with the
// route direction. All values in the same units.
func RouteDotSegments(x1, y1, x2, y2, spacing, dotLen float64) []LineSegment {
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	if length <= 0 || spacing <= 0 || dotLen <= 0 {
		return nil
	}
	ux, uy := dx/length, dy/length
	var out []LineSegment
	for d := dotLen / 2; d < length; d += spacing {
		from := math.Max(0, d-dotLen/2)
		to := math.Min(length, d+dotLen/2)
		if to-from < 0.5 {
			break
		}
		out = append(out, LineSegment{
			X1: x1 + ux*from, Y1: y1 + uy*from,
			X2: x1 + ux*to, Y2: y1 + uy*to,
		})
	}
	return out
}

// ClipSegmentToRect clips a line segment to an axis-aligned rectangle
// (Liang-Barsky) — the engine has no clip-path, so hatch lines and any
// wash geometry are confined to the region's bounds rect Go-side. Returns
// the clipped segment and whether it is visible.
// clampTo snaps v into [lo, hi]. Used after parametric clipping to prevent
// floating-point overshoot at rect boundaries.
func clampTo(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ClipSegmentToRect(s LineSegment, minX, minY, w, h float64) (LineSegment, bool) {
	dx, dy := s.X2-s.X1, s.Y2-s.Y1
	t0, t1 := 0.0, 1.0
	clip := func(p, q float64) bool {
		if p == 0 {
			return q >= 0 // parallel and inside
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return false
			}
			if r < t1 {
				t1 = r
			}
		}
		return true
	}
	if !clip(-dx, s.X1-minX) || !clip(dx, minX+w-s.X1) ||
		!clip(-dy, s.Y1-minY) || !clip(dy, minY+h-s.Y1) {
		return LineSegment{}, false
	}
	return LineSegment{
		// Clamp output to rect bounds — the Liang-Barsky parametric clip can
		// produce values just outside the rect due to floating-point rounding
		// (e.g. 30.000000000000004 for a rect right edge at 30).
		X1: clampTo(s.X1+dx*t0, minX, minX+w), Y1: clampTo(s.Y1+dy*t0, minY, minY+h),
		X2: clampTo(s.X1+dx*t1, minX, minX+w), Y2: clampTo(s.Y1+dy*t1, minY, minY+h),
	}, true
}

// HatchSegments generates the hatch lines of a faction wash (AC-4): lines
// at angleDeg (degrees, screen convention) spaced spacing apart across the
// rectangle (minX, minY, w, h), each clipped to the rectangle. All values
// in the same units (surface px for the renderer).
func HatchSegments(minX, minY, w, h, angleDeg, spacing float64) []LineSegment {
	if w <= 0 || h <= 0 || spacing <= 0 {
		return nil
	}
	rad := angleDeg * math.Pi / 180
	// Direction d along the hatch lines; normal n across them. The lines
	// are n.P = c for c stepping by spacing across the rect's projection.
	dx, dy := math.Cos(rad), math.Sin(rad)
	nx, ny := -dy, dx
	corners := [4][2]float64{{minX, minY}, {minX + w, minY}, {minX, minY + h}, {minX + w, minY + h}}
	cMin, cMax := math.Inf(1), math.Inf(-1)
	for _, c := range corners {
		v := c[0]*nx + c[1]*ny
		cMin = math.Min(cMin, v)
		cMax = math.Max(cMax, v)
	}
	// A line spanning the rect diagonal, long enough to cover any clip.
	diag := math.Hypot(w, h)
	cx, cy := minX+w/2, minY+h/2
	var out []LineSegment
	for c := cMin; c <= cMax; c += spacing {
		// Point on the line nearest the rect center: center + (c - n.center)*n.
		nc := cx*nx + cy*ny
		px, py := cx+(c-nc)*nx, cy+(c-nc)*ny
		s := LineSegment{
			X1: px - dx*diag, Y1: py - dy*diag,
			X2: px + dx*diag, Y2: py + dy*diag,
		}
		if clipped, ok := ClipSegmentToRect(s, minX, minY, w, h); ok {
			out = append(out, clipped)
		}
	}
	return out
}

// VisibleRoutes filters authored travel routes to those shown on the map
// (AC-3): both endpoints must be discovered regions. A route touching an
// undiscovered region is hidden.
func VisibleRoutes(routes []TravelRoute, regions []RegionSummary) []TravelRoute {
	discovered := make(map[string]bool, len(regions))
	for _, r := range regions {
		discovered[r.RegionID] = true
	}
	var out []TravelRoute
	for _, rt := range routes {
		if discovered[rt.FromRegion] && discovered[rt.ToRegion] {
			out = append(out, rt)
		}
	}
	return out
}

// FactionWashOpacity clamps a faction's requested wash opacity into the
// settled 10-15% band (AC-4) — subtle, never hiding the ink linework.
func FactionWashOpacity(opacity float64) float64 {
	return math.Max(FactionWashMinOpacity, math.Min(FactionWashMaxOpacity, opacity))
}

// ---------------------------------------------------------------------------
// Renderer
// ---------------------------------------------------------------------------

// zoomButtonHeight is the detail panel's "Zoom to map" button row height
// (px), mirrored in the markup CSS.
const zoomButtonHeight = 24.0

// worldMapRenderer owns the bound world-map markup elements and the
// cloned region geometry. Like the local map's renderer, all methods are
// no-ops when the document is nil (headless mode).
type worldMapRenderer struct {
	doc *document.Document

	root    *document.Element // world-root: full-screen overlay (translateY slide)
	dim     *document.Element // world-dim: soot world-dim layer (opacity)
	surface *document.Element // world-surface: pan/scroll viewport content (left/top)
	regions *document.Element // world-regions: region group container
	routes  *document.Element // world-routes: travel route dot container

	hintElm *document.Element // world-hint-label (font-size scaling)
	hintLbl *ui.Label         // world-hint-label text

	// Clone sources (hidden templates): duplicated per element at rebuild.
	regionTpl    *document.Element // world-region-template: one region group
	washTpl      *document.Element // world-wash-template: interior/faction wash panel
	boundaryTpl  *document.Element // world-boundary-template: ink line segment
	hatchTpl     *document.Element // world-hatch-template: faction hatch line
	xTpl         *document.Element // world-x-template: ink X arms (cleared regions)
	labelTpl     *document.Element // world-region-label-template: region name

	// Detail panel (AC-5): the paper-sliding sheet and its rows.
	detailPanel   *document.Element // world-detail-panel (translateX slide)
	detailNameLbl *ui.Label         // world-detail-name text
	stateIcon     *document.Element // world-detail-state-icon: state shape
	detailState   *document.Element // world-detail-state (font-size scaling)
	detailStateLbl *ui.Label
	detailFactions *document.Element // world-detail-factions (font-size scaling)
	detailFactionsLbl *ui.Label
	detailLocations *document.Element // world-detail-locations (font-size scaling)
	detailLocationsLbl *ui.Label
	detailThreads *document.Element // world-detail-threads (font-size scaling)
	detailThreadsLbl *ui.Label
	zoomBtn   *document.Element // world-detail-zoom (font-size scaling)
	detailArm []*document.Element // cloned X arms inside the state icon

	clones []*document.Element // every cloned element, removed at rebuild
}

// newWorldMapRenderer creates the renderer. A nil document puts the
// renderer in headless mode (all operations no-op).
func newWorldMapRenderer(doc *document.Document) *worldMapRenderer {
	return &worldMapRenderer{doc: doc}
}

// online reports whether the renderer has a document to operate on.
func (r *worldMapRenderer) online() bool { return r.doc != nil }

// element resolves a markup id, logging a warning when missing.
func (r *worldMapRenderer) element(id string) *document.Element {
	elm, ok := r.doc.GetElementById(id)
	if !ok {
		slog.Warn("world map: markup element not found", "id", id)
		return nil
	}
	return elm
}

// bind resolves the markup ids and parks the overlay: root and detail
// panel hidden, dim at full brightness, templates hidden. Open() slides
// it in.
func (r *worldMapRenderer) bind() {
	if !r.online() {
		return
	}
	r.root = r.element("world-root")
	r.dim = r.element("world-dim")
	r.surface = r.element("world-surface")
	r.regions = r.element("world-regions")
	r.routes = r.element("world-routes")
	r.hintElm = r.element("world-hint-label")
	if r.hintElm != nil {
		r.hintLbl = r.hintElm.FindElementLabelById("world-hint-label")
	}
	r.regionTpl = r.element("world-region-template")
	r.washTpl = r.element("world-wash-template")
	r.boundaryTpl = r.element("world-boundary-template")
	r.hatchTpl = r.element("world-hatch-template")
	r.xTpl = r.element("world-x-template")
	r.labelTpl = r.element("world-region-label-template")
	r.detailPanel = r.element("world-detail-panel")
	if r.detailPanel != nil {
		if name := r.element("world-detail-name"); name != nil {
			r.detailNameLbl = name.FindElementLabelById("world-detail-name")
		}
		r.stateIcon = r.element("world-detail-state-icon")
		r.detailState = r.element("world-detail-state")
		if r.detailState != nil {
			r.detailStateLbl = r.detailState.FindElementLabelById("world-detail-state")
		}
		r.detailFactions = r.element("world-detail-factions")
		if r.detailFactions != nil {
			r.detailFactionsLbl = r.detailFactions.FindElementLabelById("world-detail-factions")
		}
		r.detailLocations = r.element("world-detail-locations")
		if r.detailLocations != nil {
			r.detailLocationsLbl = r.detailLocations.FindElementLabelById("world-detail-locations")
		}
		r.detailThreads = r.element("world-detail-threads")
		if r.detailThreads != nil {
			r.detailThreadsLbl = r.detailThreads.FindElementLabelById("world-detail-threads")
		}
		r.zoomBtn = r.element("world-detail-zoom")
	}
	if r.root != nil {
		r.doc.SetElementStyleProperty(r.root, "display", "none")
	}
	// Clone sources park off-screen; clones are shown Go-side.
	for _, tpl := range []*document.Element{r.regionTpl, r.washTpl, r.boundaryTpl, r.hatchTpl, r.xTpl, r.labelTpl} {
		if tpl != nil {
			r.doc.SetElementStyleProperty(tpl, "display", "none")
		}
	}
	if r.detailPanel != nil {
		r.doc.SetElementStyleProperty(r.detailPanel, "display", "none")
	}
	r.applyDim(0)
}

// Overlay visibility ------------------------------------------------------

func (r *worldMapRenderer) show() {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "display", "flex")
}

func (r *worldMapRenderer) hide() {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "display", "none")
}

// applySlide positions the overlay via translateY (paper-slide motion; the
// values come from the map's Go-driven tweens) — same convention as the
// local map: negative parks below the screen edge.
func (r *worldMapRenderer) applySlide(offsetY float64) {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "translateY", fmt.Sprintf("%.0fpx", offsetY))
}

// applyDim sets the world-dim layer opacity (0 = world at full brightness,
// worldDimOpacity = world at 30%, desk register). Driven per frame by the
// dim tween.
func (r *worldMapRenderer) applyDim(opacity float64) {
	if !r.online() || r.dim == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.dim, "opacity", fmt.Sprintf("%.3f", opacity))
}

// applySurface positions the world canvas inside the parchment — the
// pan/scroll viewport offset (AC-1).
func (r *worldMapRenderer) applySurface(offsetX, offsetY float64) {
	if !r.online() || r.surface == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.surface, "left", fmt.Sprintf("%.1fpx", offsetX))
	r.doc.SetElementStyleProperty(r.surface, "top", fmt.Sprintf("%.1fpx", offsetY))
}

// World rebuild (AC-1 through AC-4) --------------------------------------

// rebuild replaces the world map's rendered layer with the snapshot: one
// region group per discovered region (wash, boundary linework, cleared X,
// name label), faction washes with hatch lines, and the visible dotted
// travel routes. The world canvas is sized to the snapshot bounds plus
// the parchment padding.
func (r *worldMapRenderer) rebuild(data WorldMapData, scale, canvasW, canvasH, canvasMinX, canvasMaxY float64) {
	if !r.online() {
		return
	}
	for _, elm := range r.clones {
		r.doc.RemoveElement(elm)
	}
	r.clones = nil

	if r.surface != nil {
		r.doc.SetElementStyleProperty(r.surface, "width", fmt.Sprintf("%.0fpx", canvasW))
		r.doc.SetElementStyleProperty(r.surface, "height", fmt.Sprintf("%.0fpx", canvasH))
	}
	if r.hintElm != nil {
		display := "none"
		if len(data.Regions) == 0 {
			display = "block"
		}
		r.doc.SetElementStyleProperty(r.hintElm, "display", display)
	}
	if r.regionTpl == nil {
		return
	}

	// Region groups: one cloned container per discovered region.
	groups := r.doc.DuplicateElementRepeatWithoutApplyStyles(r.regionTpl, len(data.Regions))
	byID := make(map[string]RegionSummary, len(data.Regions))
	for i, region := range data.Regions {
		byID[region.RegionID] = region
		g := groups[i]
		r.clones = append(r.clones, g)
		r.doc.SetElementIdWithoutApplyStyles(g, fmt.Sprintf("world-region-%d", i))
		r.doc.SetElementClassesWithoutApply(g, "world-region")
		r.doc.ChangeElementParentWithoutApply(g, r.regions)
		r.doc.SetElementStylePropertyWithoutApply(g, "display", "block")

		minX, minY, w, h := RegionBounds(region)
		if w <= 0 || h <= 0 {
			continue
		}
		wPx, hPx := w*scale, h*scale
		// Group origin: the region bounds' top-left in surface px (y-down).
		gpx := canvasPadding + (minX-canvasMinX)*scale
		gpy := canvasPadding + (canvasMaxY-(minY+h))*scale
		r.doc.SetElementStylePropertyWithoutApply(g, "left", fmt.Sprintf("%.1fpx", gpx))
		r.doc.SetElementStylePropertyWithoutApply(g, "top", fmt.Sprintf("%.1fpx", gpy))
		r.doc.SetElementStylePropertyWithoutApply(g, "width", fmt.Sprintf("%.1fpx", wPx))
		r.doc.SetElementStylePropertyWithoutApply(g, "height", fmt.Sprintf("%.1fpx", hPx))

		// Interior wash (AC-2): faint ink fill, none for Unexplored.
		if color := RegionWashColor(region.State); color != "" {
			r.clonePanel(r.washTpl, g, "world-wash", wPx, hPx, color, 1.0)
		}
		// Faction territory (AC-4): colored wash at 10-15% with hatch.
		if RegionHasWash(region.State) {
			for _, o := range data.Factions[region.RegionID] {
				op := FactionWashOpacity(o.Opacity)
				r.clonePanel(r.washTpl, g, "world-faction-wash", wPx, hPx, o.Color, op)
				for _, angle := range HatchAngles(o.Hatch) {
					for _, hs := range HatchSegments(0, 0, wPx, hPx, angle, hatchSpacing) {
						elm := r.cloneLine(r.hatchTpl, g, "world-hatch", hs, hatchThickness, o.Color, 0.5)
						r.clones = append(r.clones, elm)
					}
				}
			}
		}
		// Boundary linework (AC-2): solid ink for Visited, faded for
		// Cleared, dashed for Unexplored (DashSegments).
		ink, dashed := RegionBoundaryInk(region.State)
		for _, s := range region.Boundary {
			seg := r.localSegment(s, minX, minY, scale, hPx)
			segs := []LineSegment{seg}
			if dashed {
				segs = DashSegments(seg, dashLength, dashGap)
			}
			for _, ds := range segs {
				elm := r.cloneLine(r.boundaryTpl, g, "world-boundary", ds, boundaryThickness, ink, 1.0)
				r.clones = append(r.clones, elm)
			}
		}
		// Ink X over Cleared regions (AC-2).
		if region.State == RegionStateCleared {
			elm := r.cloneX(r.xTpl, g, wPx/2-6, hPx/2-6)
			r.clones = append(r.clones, elm)
		}
		// Region name label (AC-1): ink text at the bounds center.
		lbl := r.cloneLabel(r.labelTpl, g, wPx/2, hPx/2, TruncateLabel(region.Name, markerLabelMaxRunes))
		if lbl != nil {
			r.clones = append(r.clones, lbl)
		}
	}

	// Travel routes (AC-3): dotted ink lines between discovered region
	// centers.
	for _, rt := range VisibleRoutes(data.Routes, data.Regions) {
		from, okF := byID[rt.FromRegion]
		to, okT := byID[rt.ToRegion]
		if !okF || !okT {
			continue
		}
		fx, fy := RegionCentroid(from)
		tx, ty := RegionCentroid(to)
		sx1, sy1 := canvasPadding+(fx-canvasMinX)*scale, canvasPadding+(canvasMaxY-fy)*scale
		sx2, sy2 := canvasPadding+(tx-canvasMinX)*scale, canvasPadding+(canvasMaxY-ty)*scale
		for _, dot := range RouteDotSegments(sx1, sy1, sx2, sy2, routeSpacing, routeDotLen) {
			elm := r.cloneLine(r.boundaryTpl, r.routes, "world-route-dot", dot, 2.0, "rgba(28, 24, 19, 0.60)", 1.0)
			r.clones = append(r.clones, elm)
		}
	}
	r.doc.ApplyStyles()
}

// clonePanel clones the wash template into a parent, sized and colored
// inline (background-color + element opacity).
func (r *worldMapRenderer) clonePanel(tpl, parent *document.Element, class string, w, h float64, color string, opacity float64) *document.Element {
	if tpl == nil || parent == nil {
		return nil
	}
	elm := tpl.Clone(tpl.Parent.Value())
	r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("world-wash-%d", len(r.clones)))
	r.doc.SetElementClassesWithoutApply(elm, class)
	r.doc.ChangeElementParentWithoutApply(elm, parent)
	r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
	r.doc.SetElementStylePropertyWithoutApply(elm, "left", "0px")
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", "0px")
	r.doc.SetElementStylePropertyWithoutApply(elm, "width", fmt.Sprintf("%.1fpx", w))
	r.doc.SetElementStylePropertyWithoutApply(elm, "height", fmt.Sprintf("%.1fpx", h))
	r.doc.SetElementStylePropertyWithoutApply(elm, "background-color", color)
	r.doc.SetElementStylePropertyWithoutApply(elm, "opacity", fmt.Sprintf("%.2f", opacity))
	r.clones = append(r.clones, elm)
	return elm
}

// cloneLine clones a line template into a parent and styles one segment
// (left/top/width/height/rotate in parent-local px, y-down).
func (r *worldMapRenderer) cloneLine(tpl, parent *document.Element, class string, s LineSegment, thickness float64, color string, opacity float64) *document.Element {
	if tpl == nil || parent == nil {
		return nil
	}
	elm := tpl.Clone(tpl.Parent.Value())
	r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("world-line-%d", len(r.clones)))
	r.doc.SetElementClassesWithoutApply(elm, "line-segment", class)
	r.doc.ChangeElementParentWithoutApply(elm, parent)
	r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
	r.styleSegment(elm, s, thickness, color, opacity)
	return elm
}

// cloneX clones the ink-X template into a parent at (left, top) px,
// showing both arms.
func (r *worldMapRenderer) cloneX(tpl, parent *document.Element, left, top float64) *document.Element {
	if tpl == nil || parent == nil {
		return nil
	}
	elm := tpl.Clone(tpl.Parent.Value())
	r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("world-x-%d", len(r.clones)))
	r.doc.SetElementClassesWithoutApply(elm, "world-x")
	r.doc.ChangeElementParentWithoutApply(elm, parent)
	r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
	r.doc.SetElementStylePropertyWithoutApply(elm, "left", fmt.Sprintf("%.1fpx", left))
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", fmt.Sprintf("%.1fpx", top))
	for _, c := range elm.Children {
		if c.HasClass("world-x-arm") {
			r.doc.SetElementStylePropertyWithoutApply(c, "display", "block")
		}
	}
	return elm
}

// cloneLabel clones the region-name label template into a parent and
// anchors it near the region bounds center (right of the X, slightly
// above the center line).
func (r *worldMapRenderer) cloneLabel(tpl, parent *document.Element, cx, cy float64, text string) *document.Element {
	if tpl == nil || parent == nil {
		return nil
	}
	elm := tpl.Clone(tpl.Parent.Value())
	r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("world-label-%d", len(r.clones)))
	r.doc.SetElementClassesWithoutApply(elm, "world-region-label")
	r.doc.ChangeElementParentWithoutApply(elm, parent)
	r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
	r.doc.SetElementStylePropertyWithoutApply(elm, "left", fmt.Sprintf("%.1fpx", cx+8))
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", fmt.Sprintf("%.1fpx", cy-9))
	for _, c := range elm.Children {
		if c.HasClass("marker-label") {
			c.InnerLabel().SetText(text)
		}
	}
	return elm
}

// styleSegment places one segment as a thin rotated panel (the local
// map's linework convention): from the midpoint, sized to the length,
// rotated to the angle, colored inline.
func (r *worldMapRenderer) styleSegment(elm *document.Element, s LineSegment, thickness float64, color string, opacity float64) {
	dx, dy := s.X2-s.X1, s.Y2-s.Y1
	length := math.Hypot(dx, dy)
	if length < 1.0 {
		return // degenerate segment — leave the clone empty
	}
	angleDeg := math.Atan2(dy, dx) * 180 / math.Pi
	midX, midY := (s.X1+s.X2)/2, (s.Y1+s.Y2)/2
	r.doc.SetElementStylePropertyWithoutApply(elm, "left", fmt.Sprintf("%.1fpx", midX-length/2))
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", fmt.Sprintf("%.1fpx", midY-thickness/2))
	r.doc.SetElementStylePropertyWithoutApply(elm, "width", fmt.Sprintf("%.1fpx", length))
	r.doc.SetElementStylePropertyWithoutApply(elm, "height", fmt.Sprintf("%.1fpx", thickness))
	r.doc.SetElementStylePropertyWithoutApply(elm, "rotate", fmt.Sprintf("%.1fdeg", angleDeg))
	r.doc.SetElementStylePropertyWithoutApply(elm, "background-color", color)
	r.doc.SetElementStylePropertyWithoutApply(elm, "opacity", fmt.Sprintf("%.2f", opacity))
}

// localSegment converts a world-canvas segment (y-up) to group-local px
// (y-down): the group origin is the region bounds' top-left, so the y
// axis flips within the group height.
func (r *worldMapRenderer) localSegment(s LineSegment, minX, minY, scale, hPx float64) LineSegment {
	return LineSegment{
		X1: (s.X1 - minX) * scale, Y1: hPx - (s.Y1-minY)*scale,
		X2: (s.X2 - minX) * scale, Y2: hPx - (s.Y2-minY)*scale,
	}
}

// Detail panel (AC-5) ------------------------------------------------------

// showDetail displays the detail panel (parked off the right edge; the
// slide tween brings it in).
func (r *worldMapRenderer) showDetail() {
	if !r.online() || r.detailPanel == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.detailPanel, "display", "block")
}

// hideDetail removes the panel after its slide-out completes (or on map
// close).
func (r *worldMapRenderer) hideDetail() {
	if !r.online() || r.detailPanel == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.detailPanel, "display", "none")
}

// applyDetailSlide positions the detail panel via translateX — the
// paper-slide from the right edge (parked off-screen at +DetailPanelWidth,
// sliding in to 0).
func (r *worldMapRenderer) applyDetailSlide(offsetX float64) {
	if !r.online() || r.detailPanel == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.detailPanel, "translateX", fmt.Sprintf("%.0fpx", offsetX))
}

// applyDetail writes the selected region's content into the panel rows
// (AC-5): name, state shape + name, faction presence, discovered
// locations and active threads counts (rows hidden for Unexplored
// regions).
func (r *worldMapRenderer) applyDetail(d RegionDetail) {
	if !r.online() {
		return
	}
	if r.detailNameLbl != nil {
		r.detailNameLbl.SetText(d.Name)
	}
	// State shape: an ink X for Cleared (cloned arms), a solid dot for
	// Visited, a dashed ring for Unexplored — never text alone.
	if r.stateIcon != nil {
		r.doc.SetElementClassesWithoutApply(r.stateIcon, "world-state-icon", "world-state-"+RegionStateInk(d.State))
		for _, arm := range r.detailArm {
			r.doc.RemoveElement(arm)
		}
		r.detailArm = nil
		if d.State == RegionStateCleared && r.xTpl != nil {
			arm := r.cloneX(r.xTpl, r.stateIcon, 0, 0)
			if arm != nil {
				r.detailArm = append(r.detailArm, arm)
			}
		}
	}
	if r.detailStateLbl != nil {
		r.detailStateLbl.SetText(RegionStateName(d.State))
	}
	if r.detailFactionsLbl != nil {
		if len(d.Factions) == 0 {
			r.detailFactionsLbl.SetText("Factions: none")
		} else {
			r.detailFactionsLbl.SetText("Factions: " + joinFactions(d.Factions))
		}
	}
	showCounts := d.HasCounts
	if r.detailLocationsLbl != nil {
		if showCounts {
			r.detailLocationsLbl.SetText(fmt.Sprintf("Locations discovered: %d", d.Locations))
		}
		r.doc.SetElementStyleProperty(r.detailLocations, "display", boolDisplay(showCounts))
	}
	if r.detailThreadsLbl != nil {
		if showCounts {
			r.detailThreadsLbl.SetText(fmt.Sprintf("Active threads: %d", d.Threads))
		}
		r.doc.SetElementStyleProperty(r.detailThreads, "display", boolDisplay(showCounts))
	}
}

// joinFactions joins the faction presence names with a comma separator —
// "Faction A, Faction B". Localized punctuation lands with the
// localization system (ui-code rules; see the markup TODO).
func joinFactions(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// boolDisplay converts a visibility flag to the CSS display value.
func boolDisplay(on bool) string {
	if on {
		return "block"
	}
	return "none"
}

// Furniture ---------------------------------------------------------------

// applyFontScale applies the height-based UI scale (AC-7: body 16px at
// reference height, hint 12px, never below reference size) to the world
// map's fixed labels.
func (r *worldMapRenderer) applyFontScale(scale float64) {
	if !r.online() {
		return
	}
	if r.hintElm != nil {
		r.doc.SetElementStyleProperty(r.hintElm, "font-size", fmt.Sprintf("%.0fpx", hintFontSize*scale))
	}
	for _, elm := range []*document.Element{r.detailState, r.detailFactions, r.detailLocations, r.detailThreads, r.zoomBtn} {
		if elm != nil {
			r.doc.SetElementStyleProperty(elm, "font-size", fmt.Sprintf("%.0fpx", hintFontSize*scale))
		}
	}
}

// destroy tears the markup down. The renderer must not be used afterwards.
func (r *worldMapRenderer) destroy() {
	if r.online() {
		r.doc.Destroy()
	}
	r.doc = nil
	r.clones = nil
	r.detailArm = nil
}

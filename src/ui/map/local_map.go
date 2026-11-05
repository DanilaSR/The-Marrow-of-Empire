// Package local_map implements the Desk-register local map (ADR-0012,
// Story 001: Local Map Display & Navigation, Story 002: Map Markers &
// Pins): the full-screen parchment overlay, its paper-slide enter/exit
// transitions, the world dim to 30%, ink-on-parchment region linework, the
// player position indicator, the pan/scroll viewport (mouse drag +
// edge-scroll), and the marker layer — NPC pins, amber thread dots/stars,
// location pins, and ink X marks over cleared locations (markers.go).
//
// The map is a pure consumer (ui-code rules): it reads player position
// through the PositionProvider interface and marker data through the
// NPCMarkerProvider/ThreadMarkerProvider/LocationMarkerProvider interfaces
// the game layer implements, and never mutates game state. It renders in
// screen space on the UI orthographic camera — immune to world camera
// shake, the desk never trembles (TR-ui-004, AC-6). Desk register styling
// (parchment, world dims to 30%, dark ink palette, no amber text) is
// defined in assets/ui/map.html.
//
// Design: the engine-free models own all state — the map state machine and
// viewport (local_map.go), RegionMapData/LineSegment region geometry — the
// mapRenderer (render.go) translates them into Kaiju document operations,
// and tween.go provides the paper-slide motion (Kaiju CSS has no
// transitions, so motion is Go-driven per frame).
//
// Note on package naming: the directory is src/ui/map/ but the Go keyword
// `map` cannot be a package clause name, so the package is declared
// `local_map` (matching the project's underscore package convention, e.g.
// combat_hud).
package local_map

import (
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/platform/hid"
)

// Paper-slide timings per ADR-0012 / TR-ui-004: enter 200-300ms ease-out,
// exit 200ms, no bounce/scale/spin/flip. The map enters at 250ms — within
// the required window.
const (
	slideEnterDuration = 0.25 // seconds
	slideExitDuration  = 0.20 // seconds
)

// worldDimOpacity is the settled opacity of the soot dim layer behind the
// parchment — the world shows at 30% brightness (ADR-0012 desk register,
// AC-1).
const worldDimOpacity = 0.70

// WorldDimOpacity is the dim layer's settled opacity, exported as the
// single source of truth for the world-dim calculation and its tests.
const WorldDimOpacity = worldDimOpacity

// defaultMapScale maps region world units to surface pixels. Region data
// is authored in world units (1 unit = 1 meter per TR-combat-001); the
// surface is rendered at 8px per meter.
const defaultMapScale = 8.0

// Pan input parameters (Story 001 implementation notes): the cursor is
// within edgeScrollMargin px of the viewport edge, the view scrolls in
// that direction at edgeScrollSpeed px/s.
const (
	edgeScrollMargin = 20.0  // px
	edgeScrollSpeed  = 300.0 // px/s
)

// Height-based scaling (TR-ui-006, AC-7): reference height 1080px with a
// floor of 1.0 — text never shrinks below the reference size, so body text
// (16px) stays >= the 12px effective minimum at 1280x720. Scale cap 1.5
// prevents oversized text on very tall windows.
const (
	referenceHeight = 1080.0
	minHeightScale  = 1.0
	maxHeightScale  = 1.5
)

// PositionProvider supplies the player's current world position and facing
// for the map indicator (AC-4). The game layer implements it from the
// entity transform; the map depends on the interface, not the source, and
// polls it once per frame while the map is open.
//
// Coordinates are region world units (meters) with y-up; facing is radians
// in the math convention (0 = +x / east, positive = counter-clockwise).
type PositionProvider interface {
	GetPlayerPosition() (x, y, facing float64)
}

// LineSegment is a straight ink line in region world units (meters, y-up):
// from (X1, Y1) to (X2, Y2). Boundaries, paths, and structures are all
// rendered as collections of these (AC-3).
type LineSegment struct {
	X1, Y1, X2, Y2 float64
}

// RegionMapData describes the current region's geography for the map
// linework. It is a data structure only — population from scene/region
// data lands with the scene pipeline; the game layer supplies it via
// SetRegion. Width/Height are the region extent in world units.
//
// RegionID is the region's stable identifier, passed to the marker
// providers (Story 002, AC-6: markers are region-scoped). Markers render
// only when RegionID is non-empty — there is deliberately no fallback to
// Name, so a provider keyed by region ID can never silently receive a
// display name.
type RegionMapData struct {
	RegionID   string
	Name       string
	Width      float64
	Height     float64
	Boundaries []LineSegment // region edge boundaries (3px ink)
	Paths      []LineSegment // roads and paths (faint ink)
	Structures []LineSegment // building footprints and walls (2px ink)
}

// mapState is the local map lifecycle state machine (AC-1/AC-2).
type mapState int

const (
	stateClosed   mapState = iota // hidden, parked off-screen
	stateEntering                 // paper-slide in / dim in running
	stateOpen                     // fully open, pan + player live
	stateExiting                  // paper-slide out / dim out running
)

// LocalMap is the local map entry point: it owns the region data, the
// viewport offset, the paper-slide and dim tweens, the drag/edge-scroll
// pan state, and the renderer, and polls the M/Esc hotkeys per frame.
type LocalMap struct {
	host     *engine.Host
	renderer *mapRenderer
	provider PositionProvider

	// Marker providers (Story 002): nil disables the corresponding marker
	// type. Snapshotted on Open and SetRegion, re-queryable via
	// RefreshMarkers.
	npcProvider      NPCMarkerProvider
	threadProvider   ThreadMarkerProvider
	locationProvider LocationMarkerProvider

	// snapshot is the marker layer state for the current region (AC-6),
	// exposed via Markers.
	snapshot MarkerSnapshot

	region RegionMapData
	scale  float64 // px per world unit

	// Viewport offset: the surface's left/top position inside the
	// parchment (px). Clamped by ClampViewport so the region always
	// covers the viewport (or is centered when smaller).
	offsetX, offsetY float64

	viewportW, viewportH float64 // cached window size (px)

	state         mapState
	slide         *Tween // paper slide (translateY on the root)
	dim           *Tween // world dim (opacity on the dim layer)
	reducedMotion bool

	// Drag state: left-button press starts a grab; the viewport follows
	// the cursor delta. lastMX/lastMY are the previous frame's cursor
	// screen position.
	dragging bool
	lastMX   float64
	lastMY   float64

	// worldHandler is invoked when the player clicks the "World" button
	// (Story 003 navigation: zoom out to the world map). nil disables the
	// button; the game layer wires it to WorldMap.OpenFromLocal.
	worldHandler func()

	// inputEnabled gates the local map's input (hotkeys + pan). The world
	// map suspends it while it is open on top, so M cannot toggle both
	// overlays in one frame (Story 003, AC-6).
	inputEnabled bool

	// Last polled player position (AC-4), exposed for tests.
	playerX, playerY, playerFacing float64

	updateID engine.UpdateId
}

// NewLocalMap loads the map markup, binds elements, parks the overlay
// off-screen, and registers the per-frame update.
//
// Headless mode: pass nil host and nil ui.Manager (as the unit tests do)
// to skip markup loading and the per-frame update while keeping the state
// machine, viewport math, and tweens fully active. provider may be nil —
// the player indicator then stays hidden.
func NewLocalMap(host *engine.Host, man *ui.Manager, provider PositionProvider) *LocalMap {
	m := &LocalMap{
		host:     host,
		provider: provider,
		scale:    defaultMapScale,
	}
	m.renderer = newMapRenderer(nil)

	if host != nil && man != nil {
		doc, err := markup.DocumentFromHTMLAsset(man, "ui/map.html", nil, nil)
		if err != nil {
			slog.Warn("local map: failed to load markup", "err", err)
			return m
		}
		m.renderer = newMapRenderer(doc)
		m.renderer.bind()
		m.viewportW = float64(host.Window.Width())
		m.viewportH = float64(host.Window.Height())
		m.renderer.applyFontScale(HeightScaleFactor(m.viewportH))
		m.updateID = host.Updater.AddUpdate(m.Update)
	}
	return m
}

// SetRegion installs the region geography: stores the data, rebuilds the
// linework, writes the region name label, and refreshes the marker layer
// for the new region (AC-6). The viewport re-centers on the player at the
// next Open.
func (m *LocalMap) SetRegion(data RegionMapData) {
	m.region = data
	m.renderer.rebuildRegion(data, m.scale)
	m.renderer.applyRegionName(data.Name)
	m.refreshMarkers()
}

// SetViewportSize updates the cached viewport size and re-clamps the
// offset (window resize support; the per-frame Update also refreshes the
// cache from the host window).
func (m *LocalMap) SetViewportSize(w, h float64) {
	m.viewportW, m.viewportH = w, h
	m.reclamp()
}

// State machine -----------------------------------------------------------

// Open begins the enter sequence: the parchment slides up from below the
// screen edge (250ms ease-out) while the world dims to 30% (AC-1). No-op
// when already entering or open. Under reduced motion the map appears
// instantly.
func (m *LocalMap) Open() {
	if m.state == stateEntering || m.state == stateOpen {
		return
	}
	m.state = stateEntering
	m.renderer.show()
	m.renderer.applySlide(-m.viewportH) // park below the screen edge
	m.renderer.applyDim(0)
	m.centerOnPlayer()
	m.updatePlayer()
	m.refreshMarkers() // snapshot the marker layer at open (Story 002)
	if m.reducedMotion {
		m.state = stateOpen
		m.renderer.applySlide(0)
		m.renderer.applyDim(worldDimOpacity)
		return
	}
	m.slide = NewPaperSlideTween(-m.viewportH, 0, slideEnterDuration)
	m.dim = NewPaperSlideTween(0, worldDimOpacity, slideEnterDuration)
}

// Close begins the exit sequence: the parchment slides out below the
// screen edge and the world brightens back to full (both 200ms, AC-2).
// No-op when already exiting or closed.
func (m *LocalMap) Close() {
	if m.state == stateExiting || m.state == stateClosed {
		return
	}
	m.state = stateExiting
	if m.reducedMotion {
		m.state = stateClosed
		m.renderer.applySlide(0)
		m.renderer.applyDim(0)
		m.renderer.hide()
		return
	}
	m.slide = NewPaperSlideTween(0, -m.viewportH, slideExitDuration)
	m.dim = NewPaperSlideTween(worldDimOpacity, 0, slideExitDuration)
}

// Toggle opens a closed (or closing) map and closes an open (or entering)
// one — the M-key behavior (AC-1/AC-2).
func (m *LocalMap) Toggle() {
	if m.state == stateClosed || m.state == stateExiting {
		m.Open()
		return
	}
	m.Close()
}

// IsOpen reports whether the map is on screen (entering, open, or
// exiting). The map is interactive only in the open state.
func (m *LocalMap) IsOpen() bool { return m.state != stateClosed }

// ViewportOffset returns the current surface offset (px).
func (m *LocalMap) ViewportOffset() (float64, float64) { return m.offsetX, m.offsetY }

// PlayerPosition returns the last polled player position and facing
// (AC-4).
func (m *LocalMap) PlayerPosition() (float64, float64, float64) {
	return m.playerX, m.playerY, m.playerFacing
}

// Pan shifts the viewport by (dx, dy) surface px and re-clamps — the
// shared path for drag and edge-scroll (AC-5). Dragging right/down moves
// the surface with the cursor; edge-scroll applies the same convention.
func (m *LocalMap) Pan(dx, dy float64) {
	rw, rh := m.regionPxSize()
	m.offsetX, m.offsetY = ClampViewport(m.offsetX+dx, m.offsetY+dy, rw, rh, m.viewportW, m.viewportH)
	m.renderer.applySurface(m.offsetX, m.offsetY)
}

// SetReducedMotion enables or disables the reduced-motion gate: while on,
// paper-slide and dim transitions complete instantly (accessibility
// requirement; the game layer wires it from the accessibility settings).
func (m *LocalMap) SetReducedMotion(on bool) { m.reducedMotion = on }

// SetWorldRequestHandler installs the "World" button handler (Story 003
// navigation: zoom out from the local map to the world map). The button
// is clickable only while the map is open; the game layer wires it to
// WorldMap.OpenFromLocal. nil disables the button.
func (m *LocalMap) SetWorldRequestHandler(fn func()) { m.worldHandler = fn }

// SetInputEnabled suspends or restores the local map's input (hotkeys and
// pan). The world map suspends it while it is open on top, so pressing M
// cannot toggle both overlays in one frame (Story 003, AC-6).
func (m *LocalMap) SetInputEnabled(on bool) { m.inputEnabled = on }

// InputEnabled reports whether the local map's input is active.
func (m *LocalMap) InputEnabled() bool { return m.inputEnabled }

// Destroy tears down the map: removes the per-frame update and destroys
// the markup document. The instance must not be used afterwards.
func (m *LocalMap) Destroy() {
	if m.host != nil {
		m.host.Updater.RemoveUpdate(&m.updateID)
	}
	if m.renderer != nil {
		m.renderer.destroy()
	}
	m.host = nil
	m.provider = nil
	m.npcProvider = nil
	m.threadProvider = nil
	m.locationProvider = nil
	m.snapshot = MarkerSnapshot{}
}

// Per-frame update --------------------------------------------------------

// Update advances the slides and the dim, routes the M/Esc hotkeys, and —
// while the map is fully open — polls the player position and drives the
// pan input (drag + edge-scroll). Registered on the host updater by
// NewLocalMap; callable directly in headless tests.
func (m *LocalMap) Update(dt float64) {
	if m.host != nil {
		m.viewportW = float64(m.host.Window.Width())
		m.viewportH = float64(m.host.Window.Height())
	}
	m.refreshFontScale()
	m.updateTweens(dt)
	m.pollHotkeys()
	if m.state == stateOpen {
		m.updatePlayer()
		m.updatePan(dt)
	}
}

// refreshFontScale re-applies the height-based font scale from the current
// viewport height (AC-7). Called every frame so that window resize keeps
// the body text >= 12px effective at any resolution.
func (m *LocalMap) refreshFontScale() {
	m.renderer.applyFontScale(HeightScaleFactor(m.viewportH))
}

// updateTweens advances the paper slide and the world dim together and
// completes the state transitions: entering -> open when the slide-in
// finishes, exiting -> closed (hidden) when the slide-out finishes.
func (m *LocalMap) updateTweens(dt float64) {
	if m.slide == nil {
		return
	}
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
			m.state = stateClosed
			m.renderer.hide()
		}
	}
}

// pollHotkeys routes keyboard input (AC-1/AC-2): M toggles the map, Esc
// closes it while it is on screen. KeyDown is edge-triggered in the engine
// (KeyStateDown, distinct from KeyHeld), so a held key cannot toggle
// repeatedly. While input is suspended (the world map is open on top,
// Story 003 AC-6) no hotkey is routed. No-op without a host (headless
// mode — tests drive Open/Close/Toggle directly).
func (m *LocalMap) pollHotkeys() {
	if m.host == nil {
		return
	}
	if !m.inputEnabled {
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

// updatePlayer polls the PositionProvider and places the player indicator
// (AC-4). No-op without a provider.
func (m *LocalMap) updatePlayer() {
	if m.provider == nil {
		return
	}
	x, y, facing := m.provider.GetPlayerPosition()
	m.playerX, m.playerY, m.playerFacing = x, y, facing
	px, py := m.worldToSurface(x, y)
	m.renderer.applyPlayer(px, py, facing)
}

// updatePan drives the pan input while the map is open (AC-5): mouse drag
// (left button held, viewport follows the cursor delta) and edge-scroll
// (cursor within 20px of a viewport edge scrolls that way at 300px/s). A
// press on the "World" button (Story 003 navigation) fires the world-map
// request handler instead of starting a drag. Suspended input (the world
// map is open on top) routes nothing.
func (m *LocalMap) updatePan(dt float64) {
	if m.host == nil {
		return
	}
	if !m.inputEnabled {
		return
	}
	mouse := &m.host.Window.Mouse
	pos := mouse.ScreenPosition()
	mx, my := float64(pos.X()), float64(pos.Y())

	if mouse.Pressed(hid.MouseButtonLeft) {
		if m.worldHandler != nil && m.worldButtonHit(mx, my) {
			m.worldHandler()
			m.lastMX, m.lastMY = mx, my
			return
		}
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

// worldButtonHit reports whether a window-space point falls inside the
// "World" button's rect — the top-right tab, positioned Go-side from the
// same constants as the renderer (render.go worldButton*).
func (m *LocalMap) worldButtonHit(mx, my float64) bool {
	x := m.viewportW - worldButtonMargin - worldButtonWidth
	return mx >= x && mx <= x+worldButtonWidth &&
		my >= worldButtonMargin && my <= worldButtonMargin+worldButtonHeight
}

// edgeScroll pans the viewport while the cursor sits in the 20px edge
// zones, at 300px/s (AC-5). Screen y grows downward: the top edge pans the
// view up (surface down, offsetY+), the bottom edge pans down, the left
// edge pans left (offsetX-), the right edge pans right.
func (m *LocalMap) edgeScroll(dt float64) {
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

// Geometry ----------------------------------------------------------------

// regionPxSize returns the region surface extent in px (world units times
// the map scale).
func (m *LocalMap) regionPxSize() (float64, float64) {
	return m.region.Width * m.scale, m.region.Height * m.scale
}

// worldToSurface maps region world coordinates (meters, y-up) to surface
// pixels (y-down, north up) — the surface's y axis flips so north is up.
// Delegates to the exported WorldToSurface (markers.go) so the marker
// layer and the player indicator share one conversion.
func (m *LocalMap) worldToSurface(x, y float64) (float64, float64) {
	_, rh := m.regionPxSize()
	return WorldToSurface(x, y, m.scale, rh)
}

// centerOnPlayer positions the viewport so the player sits at the screen
// center (AC-4), clamped to the region bounds. No-op without a provider or
// a region.
func (m *LocalMap) centerOnPlayer() {
	if m.provider == nil || m.region.Width <= 0 {
		return
	}
	x, y, _ := m.provider.GetPlayerPosition()
	px, py := m.worldToSurface(x, y)
	rw, rh := m.regionPxSize()
	m.offsetX, m.offsetY = ClampViewport(m.viewportW/2-px, m.viewportH/2-py, rw, rh, m.viewportW, m.viewportH)
	m.renderer.applySurface(m.offsetX, m.offsetY)
}

// reclamp re-applies the viewport clamp after a resize.
func (m *LocalMap) reclamp() {
	rw, rh := m.regionPxSize()
	m.offsetX, m.offsetY = ClampViewport(m.offsetX, m.offsetY, rw, rh, m.viewportW, m.viewportH)
	m.renderer.applySurface(m.offsetX, m.offsetY)
}

// ClampViewport clamps a surface offset (px) so the region always covers
// the viewport; a region smaller than the viewport is centered instead
// (the parchment shows around its edges). Returns the clamped (offsetX,
// offsetY).
//
// Region extents are in surface px (world units x map scale); viewport
// extents in window px.
func ClampViewport(offsetX, offsetY, regionW, regionH, viewportW, viewportH float64) (float64, float64) {
	return clampAxis(offsetX, regionW, viewportW), clampAxis(offsetY, regionH, viewportH)
}

// clampAxis clamps one viewport axis: for a region larger than the
// viewport the offset stays in [viewport-region, 0] (the surface covers
// the viewport; panning stops at the region edges); for a smaller region
// the offset centers it.
func clampAxis(offset, region, viewport float64) float64 {
	if region <= viewport {
		return (viewport - region) / 2
	}
	if offset < viewport-region {
		return viewport - region
	}
	if offset > 0 {
		return 0
	}
	return offset
}

// HeightScaleFactor returns the height-based UI scale (TR-ui-006, AC-7):
// windowHeight / 1080 with a floor of 1.0 — text never shrinks below the
// reference size, keeping body text (16px) >= 12px effective at 1280x720 —
// and a cap of 1.5.
func HeightScaleFactor(windowHeight float64) float64 {
	s := windowHeight / referenceHeight
	if s < minHeightScale {
		s = minHeightScale
	}
	if s > maxHeightScale {
		s = maxHeightScale
	}
	return s
}

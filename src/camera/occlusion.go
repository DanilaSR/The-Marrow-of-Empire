// Occlusion, interior roof removal, and interactable highlighting for the
// isometric camera (Story CAM-004, GDD CR-8).
//
// The occlusion policy is screen-space: a structure occludes the party when
// its world AABB, projected into camera view space, overlaps a party member's
// view-space rect and its near side sits between the camera and that member.
// Faded structures interpolate their shader alpha to a configurable floor
// (20% solid, 35% foliage) over 150ms, with hysteresis (300ms hold + 300ms
// continuous clear) so walls never flicker.
//
// Per-frame cost contract (GDD CR-8): the structure list is provided once per
// region via SetStructures — the detector performs zero spatial-index range
// queries per frame. The only per-frame work is the view-space projection of
// the already-known structure list.
package camera

import (
	"kaijuengine.com/engine/cameras"
	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// AABB is an axis-aligned world-space bounding box.
type AABB struct {
	Min matrix.Vec3
	Max matrix.Vec3
}

// OverlapsXY reports whether two AABBs overlap when projected onto the XZ
// plane (screen-space overlap test, GDD CR-8).
func (a AABB) OverlapsXY(b AABB) bool {
	return a.Min.X() < b.Max.X() && a.Max.X() > b.Min.X() &&
		a.Min.Z() < b.Max.Z() && a.Max.Z() > b.Min.Z()
}

// Occluder describes one structure mesh (or mesh group) eligible for fading.
// AABB is the mesh's world-space bounds; Foliage selects the higher alpha
// floor (35% vs 20% solid, GDD edge cases).
type Occluder struct {
	ID      entity.GameEntityID
	AABB    AABB
	Foliage bool
}

// OcclusionConfig holds the GDD CR-8 tuning knobs. All values are the GDD
// defaults; the game's config system may override them (data-driven values).
type OcclusionConfig struct {
	FadeAlpha           float32 // solid structure fade floor (default 0.20)
	FoliageAlpha        float32 // foliage fade floor (default 0.35)
	FadeDurationMs      float32 // fade-out / fade-in duration (default 150)
	MinHoldMs           float32 // minimum time held at the fade floor (default 300)
	MinClearMs          float32 // minimum continuous clear before restore (default 300)
	PartyRectHalfExtent float32 // world half-extent of a party member's rect (default 0.9)
}

// DefaultOcclusionConfig returns the GDD CR-8 prescribed tuning knobs.
func DefaultOcclusionConfig() OcclusionConfig {
	return OcclusionConfig{
		FadeAlpha:           0.20,
		FoliageAlpha:        0.35,
		FadeDurationMs:      150,
		MinHoldMs:           300,
		MinClearMs:          300,
		PartyRectHalfExtent: 0.9,
	}
}

// AlphaFader applies per-mesh alpha to the render layer. The concrete
// implementation (render framework story) writes the value into the drawing's
// ShaderDataStandard.Color.A channel; Kaiju materials have no direct alpha.
type AlphaFader interface {
	// SetAlpha sets the mesh's rendered alpha. 1.0 is fully opaque, values
	// below are the fade floors from OcclusionConfig.
	SetAlpha(id entity.GameEntityID, alpha float32)
}

// fadePhase is the per-occluder hysteresis state machine (GDD CR-8):
//
//	Opaque --occluded--> FadingOut --alpha floor--> Faded --clear>=max(hold,clear)--> FadingIn --alpha 1--> Opaque
//
// FadingOut completes monotonically even if the occlusion clears mid-fade
// (the structure finishes fading, preventing near-threshold flicker).
// FadingIn reverses immediately on re-occlusion.
type fadePhase int

const (
	phaseOpaque fadePhase = iota
	phaseFadingOut
	phaseFaded
	phaseFadingIn
)

// occluderState is the per-structure fade state.
type occluderState struct {
	occluder    Occluder
	phase       fadePhase
	alpha       float32 // current alpha, 1.0 = opaque
	lastApplied float32 // last alpha pushed to the AlphaFader
	fadeFrom    float32 // alpha at the start of the current fade (no snaps on reversal)
	outMs       float32 // elapsed in FadingOut
	inMs        float32 // elapsed in FadingIn
	holdMs      float32 // elapsed in Faded (accumulates, occlusion or not)
	clearMs     float32 // continuous clear time while Faded; resets on re-occlusion
	occluded    bool    // last Update's occlusion test result (debug readout)
	targetAlpha float32 // fade floor for this occluder (foliage or solid)
}

// viewAABB is an axis-aligned box in camera view space (x/y are screen-space,
// z is depth along the view direction, increasing away from the camera).
type viewAABB struct {
	Min, Max matrix.Vec3
}

// OcclusionDetector tests structures against the party each frame and drives
// per-mesh alpha fades through an AlphaFader. The structure list is set at
// region load; there are no per-frame spatial queries (GDD CR-8).
type OcclusionDetector struct {
	config OcclusionConfig
	cam    *cameras.StandardCamera
	fader  AlphaFader

	occluders map[entity.GameEntityID]*occluderState
	order     []entity.GameEntityID // stable iteration order

	partyBuf []viewAABB // reused per frame; zero allocation in the hot path
}

// NewOcclusionDetector creates a detector bound to a camera. cfg is typically
// DefaultOcclusionConfig; fader receives every alpha transition.
func NewOcclusionDetector(cam *cameras.StandardCamera, cfg OcclusionConfig, fader AlphaFader) *OcclusionDetector {
	return &OcclusionDetector{
		config:    cfg,
		cam:       cam,
		fader:     fader,
		occluders: make(map[entity.GameEntityID]*occluderState),
	}
}

// SetStructures replaces the occluder list (called once per region load, GDD
// CR-8). Structures removed from the list are restored to full alpha and
// dropped; new structures start opaque.
func (d *OcclusionDetector) SetStructures(structures []Occluder) {
	seen := make(map[entity.GameEntityID]struct{}, len(structures))
	for _, o := range structures {
		seen[o.ID] = struct{}{}
		if _, ok := d.occluders[o.ID]; ok {
			continue // keep existing state (alpha position) across refreshes
		}
		target := d.config.FadeAlpha
		if o.Foliage {
			target = d.config.FoliageAlpha
		}
		d.occluders[o.ID] = &occluderState{
			occluder:    o,
			phase:       phaseOpaque,
			alpha:       1,
			lastApplied: 1,
			targetAlpha: target,
		}
		d.order = append(d.order, o.ID)
	}
	for _, id := range d.order {
		if _, ok := seen[id]; !ok {
			d.RemoveStructure(id)
		}
	}
}

// RemoveStructure removes an occluder, restoring it to full alpha.
func (d *OcclusionDetector) RemoveStructure(id entity.GameEntityID) {
	st, ok := d.occluders[id]
	if !ok {
		return
	}
	if st.alpha != 1 {
		d.apply(id, 1)
	}
	delete(d.occluders, id)
	for i, oid := range d.order {
		if oid == id {
			d.order = append(d.order[:i], d.order[i+1:]...)
			break
		}
	}
}

// Update runs the occlusion test for every structure against every party
// member and advances the fade state machines. party is the current world
// position of each party member (typically 4). dtMs is the frame delta.
//
// A structure is occluding when its view-space rect overlaps a member's
// view-space rect (screen-space AABB test) and its near side is closer to the
// camera than the member (depth test) — GDD CR-8.
func (d *OcclusionDetector) Update(party []matrix.Vec3, dtMs float32) {
	if dtMs <= 0 {
		return
	}
	d.partyBuf = d.partyBuf[:0]
	half := d.config.PartyRectHalfExtent
	for _, p := range party {
		member := AABB{
			Min: matrix.NewVec3(p.X()-half, p.Y()-half, p.Z()-half),
			Max: matrix.NewVec3(p.X()+half, p.Y()+half, p.Z()+half),
		}
		d.partyBuf = append(d.partyBuf, d.toView(member))
	}
	for _, id := range d.order {
		st := d.occluders[id]
		st.occluded = false
		sv := d.toView(st.occluder.AABB)
		for i := range d.partyBuf {
			mv := &d.partyBuf[i]
			if sv.Min.X() < mv.Max.X() && sv.Max.X() > mv.Min.X() &&
				sv.Min.Y() < mv.Max.Y() && sv.Max.Y() > mv.Min.Y() &&
				sv.Min.Z() < mv.Min.Z() {
				st.occluded = true
				break
			}
		}
		d.advance(st, dtMs)
	}
}

// IsOccluded reports the last frame's occlusion test result for a structure
// (debug/QA readout, AC-1).
func (d *OcclusionDetector) IsOccluded(id entity.GameEntityID) bool {
	st, ok := d.occluders[id]
	return ok && st.occluded
}

// Alpha returns the current rendered alpha of a structure (fade-alpha debug
// readout per story QA AC-1; 1.0 when opaque).
func (d *OcclusionDetector) Alpha(id entity.GameEntityID) float32 {
	st, ok := d.occluders[id]
	if !ok {
		return 1
	}
	return st.alpha
}

// advance runs one step of the fade state machine for a structure.
func (d *OcclusionDetector) advance(st *occluderState, dtMs float32) {
	cfg := d.config
	switch st.phase {
	case phaseOpaque:
		if st.occluded {
			// Enter FadingOut and progress the fade in the same frame: the
			// detection frame already counts toward the 150ms fade-out.
			st.phase = phaseFadingOut
			st.outMs = 0
			st.fadeFrom = st.alpha // 1.0 in Opaque
			if d.fadeOutStep(st, dtMs) {
				st.enterFaded()
			}
		}

	case phaseFadingOut:
		// Monotonic completion: fade reaches the floor even if the occlusion
		// cleared mid-fade, so near-threshold toggling cannot flicker.
		if d.fadeOutStep(st, dtMs) {
			st.enterFaded()
		}

	case phaseFaded:
		st.holdMs += dtMs
		if st.occluded {
			st.clearMs = 0 // re-occlusion resets the clear window
		} else {
			st.clearMs += dtMs
		}
		// Restore needs BOTH the minimum hold at the floor and a continuous
		// clear (GDD: min_hold 300ms, min_clear 300ms) — both gates must pass
		// before the fade-in starts.
		if st.clearMs >= cfg.MinClearMs && st.holdMs >= cfg.MinHoldMs {
			st.phase = phaseFadingIn
			st.inMs = 0
			st.fadeFrom = st.alpha // the floor
		}

	case phaseFadingIn:
		if st.occluded {
			// Reverse immediately from the current alpha; never snap to 1 or
			// back to the start of the fade.
			st.phase = phaseFadingOut
			st.outMs = 0
			st.fadeFrom = st.alpha
			if d.fadeOutStep(st, dtMs) {
				st.enterFaded()
			}
			break
		}
		st.inMs += dtMs
		t := st.inMs / cfg.FadeDurationMs
		if t >= 1 {
			st.alpha = 1
			st.phase = phaseOpaque
		} else {
			st.alpha = st.fadeFrom + (1-st.fadeFrom)*t
		}
		d.apply(st.occluder.ID, st.alpha)
	}
}

// fadeOutStep advances a FadingOut by dtMs, interpolating alpha from the
// fadeFrom anchor toward the target floor, and pushes it to the fader. It
// returns true when the fade has reached the floor (caller transitions to
// Faded).
func (d *OcclusionDetector) fadeOutStep(st *occluderState, dtMs float32) bool {
	st.outMs += dtMs
	t := st.outMs / d.config.FadeDurationMs
	if t >= 1 {
		st.alpha = st.targetAlpha
		d.apply(st.occluder.ID, st.alpha)
		return true
	}
	st.alpha = st.targetAlpha + (st.fadeFrom-st.targetAlpha)*(1-t)
	d.apply(st.occluder.ID, st.alpha)
	return false
}

// enterFaded resets the hold/clear timers when a fade-out reaches the floor.
func (st *occluderState) enterFaded() {
	st.phase = phaseFaded
	st.holdMs = 0
	st.clearMs = 0
}

// apply pushes an alpha to the fader, but only when it actually changed.
func (d *OcclusionDetector) apply(id entity.GameEntityID, alpha float32) {
	st := d.occluders[id]
	if alpha == st.lastApplied {
		return
	}
	st.lastApplied = alpha
	if d.fader != nil {
		d.fader.SetAlpha(id, alpha)
	}
}

// ── Interior roof removal (AC-3) ────────────────────────────────────────────

// VisibilityApplier toggles geometry visibility on the render layer. The
// concrete implementation (render framework story) removes roof/ceiling
// drawings from the render queue, or culls them via the view culler.
type VisibilityApplier interface {
	// SetVisible shows (true) or hides (false) the given entity's geometry.
	SetVisible(id entity.GameEntityID, visible bool)
}

// InteriorController hides roof/ceiling geometry while the party is inside a
// building and restores it on exit (GDD CR-8 / story AC-3: roofs absent from
// render while inside, restored within 150ms on exit). The roof entity IDs
// (geometry tagged "roof"/"ceiling" at region load) are supplied by the
// interior trigger system via EnterInterior.
//
// The render-layer applier may animate the restore (fade-in) over up to
// 150ms; the controller itself restores visibility within RestoreDelayMs of
// exit, which defaults to 0 (next frame) — comfortably inside the AC bound.
type InteriorController struct {
	applier VisibilityApplier
	config  InteriorConfig

	interior       bool
	roofs          []entity.GameEntityID
	hidden         bool    // whether roofs are currently hidden
	restorePending bool    // exit applied, restore timer running
	restoreMs      float32 // elapsed since exit
}

// InteriorConfig holds the roof restore tuning knob (GDD CR-8).
type InteriorConfig struct {
	// RestoreDelayMs delays roof restoration after ExitInterior (default 0 —
	// restored on the next Update, within the AC-3 150ms bound).
	RestoreDelayMs float32
}

// DefaultInteriorConfig returns the GDD restore timing (immediate).
func DefaultInteriorConfig() InteriorConfig {
	return InteriorConfig{RestoreDelayMs: 0}
}

// NewInteriorController creates a controller that hides/restores roofs via
// applier.
func NewInteriorController(applier VisibilityApplier, cfg InteriorConfig) *InteriorController {
	return &InteriorController{applier: applier, config: cfg}
}

// EnterInterior marks the party as inside a building and hides its roof
// geometry immediately (AC-3: "roof is not rendered" while inside). Re-entry
// with a changed roof set re-hides (idempotent) so the new roofs are removed
// too.
func (c *InteriorController) EnterInterior(roofs []entity.GameEntityID) {
	if c.interior && c.hidden {
		c.roofs = roofs
		if c.applier != nil {
			for _, id := range roofs {
				c.applier.SetVisible(id, false)
			}
		}
		return
	}
	c.interior = true
	c.roofs = roofs
	c.hidden = true
	c.restorePending = false
	c.restoreMs = 0
	if c.applier != nil {
		for _, id := range roofs {
			c.applier.SetVisible(id, false)
		}
	}
}

// ExitInterior starts roof restoration. Visibility is applied when the
// restore timer reaches RestoreDelayMs (Update-driven).
func (c *InteriorController) ExitInterior() {
	if !c.interior {
		return
	}
	c.interior = false
	c.restorePending = true
	c.restoreMs = 0
}

// Update advances the restore timer. Roofs become visible once the exit has
// been pending for RestoreDelayMs — within 150ms per story AC-3.
func (c *InteriorController) Update(dtMs float32) {
	if !c.restorePending {
		return
	}
	c.restoreMs += dtMs
	if c.restoreMs < c.config.RestoreDelayMs {
		return
	}
	c.restorePending = false
	c.hidden = false
	if c.applier != nil {
		for _, id := range c.roofs {
			c.applier.SetVisible(id, true)
		}
	}
}

// IsInterior reports whether the party is currently inside a building.
func (c *InteriorController) IsInterior() bool { return c.interior }

// AreRoofsHidden reports whether roof geometry is currently hidden (debug
// readout for AC-3 tests).
func (c *InteriorController) AreRoofsHidden() bool { return c.hidden }

// ── Interactable highlight (AC-4) ───────────────────────────────────────────

// InteractableProvider resolves the interactables near a world position. The
// concrete implementation (entity framework story) runs a single spatial
// query against the interactable component store; the camera package calls it
// only on Tab press, never per frame (GDD CR-8 contract).
type InteractableProvider interface {
	// InteractablesNear returns the GameEntityIDs of interactable entities
	// within radius world units of center.
	InteractablesNear(center matrix.Vec3, radius float32) []entity.GameEntityID
}

// OutlineApplier draws or removes an outline on an interactable's rendering.
// The concrete implementation (render framework story) sets a per-draw
// outline (2px, parchment #E3D5B0) that renders unlit and on top of
// occluding geometry — the highlight is visible through walls (AC-4).
type OutlineApplier interface {
	SetOutline(id entity.GameEntityID, color matrix.Color, widthPx float32)
	ClearOutline(id entity.GameEntityID)
}

// HighlightConfig holds the AC-4 highlight parameters.
type HighlightConfig struct {
	ParchmentColor matrix.Color // outline color #E3D5B0 (±8/255 per QA)
	OutlineWidthPx float32      // outline width, 2px
	QueryRadius    float32      // world radius for the Tab query
}

// DefaultHighlightConfig returns the AC-4 prescribed parchment highlight.
func DefaultHighlightConfig() HighlightConfig {
	return HighlightConfig{
		ParchmentColor: matrix.ColorRGBInt(227, 213, 176), // #E3D5B0
		OutlineWidthPx: 2,
		QueryRadius:    6,
	}
}

// InteractableHighlight applies the parchment outline to interactables while
// Tab is held. The provider query runs on the Tab press edge only — there are
// no per-frame spatial queries (GDD CR-8).
type InteractableHighlight struct {
	config   HighlightConfig
	provider InteractableProvider
	applier  OutlineApplier

	held        bool // Tab currently held
	highlighted []entity.GameEntityID
}

// NewInteractableHighlight creates the Tab-highlight controller.
func NewInteractableHighlight(cfg HighlightConfig, provider InteractableProvider, applier OutlineApplier) *InteractableHighlight {
	return &InteractableHighlight{config: cfg, provider: provider, applier: applier}
}

// SetTabHeld updates the Tab state. On the press edge (false→true) it queries
// interactables near center and outlines them; on release it clears all
// outlines. The highlight is not gated by occlusion — the outline renders
// through geometry (AC-4).
func (h *InteractableHighlight) SetTabHeld(held bool, center matrix.Vec3) {
	if held == h.held {
		return
	}
	h.held = held
	if held {
		if h.provider != nil {
			h.highlighted = append(h.highlighted[:0], h.provider.InteractablesNear(center, h.config.QueryRadius)...)
		}
		if h.applier != nil {
			for _, id := range h.highlighted {
				h.applier.SetOutline(id, h.config.ParchmentColor, h.config.OutlineWidthPx)
			}
		}
		return
	}
	if h.applier != nil {
		for _, id := range h.highlighted {
			h.applier.ClearOutline(id)
		}
	}
	h.highlighted = h.highlighted[:0]
}

// Update re-applies the outline to the currently highlighted set while Tab is
// held. It is idempotent and cheap — provided for render-layer state recovery
// (e.g. outline state lost on material swap) and future animated effects.
func (h *InteractableHighlight) Update(dtMs float32) {
	if !h.held || h.applier == nil {
		return
	}
	for _, id := range h.highlighted {
		h.applier.SetOutline(id, h.config.ParchmentColor, h.config.OutlineWidthPx)
	}
}

// Highlighted returns the currently highlighted interactables (debug readout).
func (h *InteractableHighlight) Highlighted() []entity.GameEntityID {
	return h.highlighted
}

// toView projects a world AABB's 8 corners into camera view space. View x/y
// are screen-space coordinates (orthographic projection drops z); view z is
// depth along the view direction, increasing away from the camera.
func (d *OcclusionDetector) toView(a AABB) viewAABB {
	v := d.cam.View()
	out := viewAABB{
		Min: matrix.NewVec3(matrix.FloatMax, matrix.FloatMax, matrix.FloatMax),
		Max: matrix.NewVec3(-matrix.FloatMax, -matrix.FloatMax, -matrix.FloatMax),
	}
	for xi := 0; xi < 2; xi++ {
		for yi := 0; yi < 2; yi++ {
			for zi := 0; zi < 2; zi++ {
				x := a.Min.X()
				if xi == 1 {
					x = a.Max.X()
				}
				y := a.Min.Y()
				if yi == 1 {
					y = a.Max.Y()
				}
				z := a.Min.Z()
				if zi == 1 {
					z = a.Max.Z()
				}
				c := matrix.Vec4MultiplyMat4(matrix.NewVec3(x, y, z).AsVec4WithW(1), v)
				if c.X() < out.Min.X() {
					out.Min.SetX(c.X())
				}
				if c.X() > out.Max.X() {
					out.Max.SetX(c.X())
				}
				if c.Y() < out.Min.Y() {
					out.Min.SetY(c.Y())
				}
				if c.Y() > out.Max.Y() {
					out.Max.SetY(c.Y())
				}
				if c.Z() < out.Min.Z() {
					out.Min.SetZ(c.Z())
				}
				if c.Z() > out.Max.Z() {
					out.Max.SetZ(c.Z())
				}
			}
		}
	}
	return out
}

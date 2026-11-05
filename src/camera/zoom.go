// Zoom system for the isometric camera: ordered discrete zoom registers,
// delta-time transitions, rapid-input queueing, per-context zoom persistence,
// and save/load state accessors. Implements Story CAM-002 (TR-cam-002) per
// ADR-0003: no free scroll wheel — three discrete levels, 300ms lerp.
package camera

import "math"

// CameraContext identifies the gameplay context the camera operates in. Each
// context remembers its own zoom level; re-entering a context restores the
// last zoom used there (Story CAM-002 AC-8).
type CameraContext int

const (
	// ExplorationContext is the default context: free camera with wheel zoom.
	ExplorationContext CameraContext = iota
	// ConversationContext is active during NPC dialogue.
	ConversationContext
	// CombatContext is active during combat encounters.
	CombatContext
)

// defaultZoomTransitionMs is the ADR-0003 transition duration used when the
// config value is missing or invalid.
const defaultZoomTransitionMs float32 = 300

// zoomLevels is the ordered set of discrete zoom registers, from closest
// (smallest half-height) to farthest. ZoomStep walks this list one level per
// detent; stepping clamps at both ends.
var zoomLevels = []ZoomLevel{ZoomExtraClose, ZoomClose, ZoomDefault, ZoomFar}

// zoomDuration returns the transition duration in milliseconds. Config values
// of zero or less fall back to the ADR-0003 default so a zero-value Config
// can never produce an instant snap.
func (c *Isometric) zoomDuration() float32 {
	if c.config.ZoomTransitionMs <= 0 {
		return defaultZoomTransitionMs
	}
	return c.config.ZoomTransitionMs
}

// ZoomTo starts a delta-time lerp from the current half-height to the target
// level over zoomDuration milliseconds (300 by default). If a transition is
// already in flight it is retargeted from its current interpolated half-height
// — no snap, no restart. The lerp is linear: every half-height between the
// endpoints is passed through monotonically, with no overshoot. For instant
// changes use SetZoom (internal use) or SetZoomState (save/load path).
func (c *Isometric) ZoomTo(target ZoomLevel) {
	c.transitionFrom = c.halfHeight
	c.transitionTo = target
	c.transitionT = 0
	c.transitioning = true
	c.pending = nil
}

// ZoomStep handles a mouse-wheel detent: a positive direction zooms in
// (toward Extra Close), a negative direction zooms out (toward Far). Each
// detent moves exactly one discrete level in zoomLevels and clamps at the
// ends of the list. While a transition is in flight the step is computed
// relative to the in-flight target level — intermediate levels are skipped
// and at most one level is queued, with the last input winning. ZoomExtraClose
// is only reachable when ExtraCloseEnabled is true. This method only exposes
// the intent; input binding lives in the input system.
func (c *Isometric) ZoomStep(direction int) {
	if direction == 0 {
		return
	}
	base := c.zoom
	if c.transitioning {
		base = c.transitionTo
	}
	idx := zoomIndex(base)
	if idx < 0 {
		return
	}
	next := idx - 1 // positive direction → zoom in → toward Extra Close
	if direction < 0 {
		next = idx + 1 // negative direction → zoom out → toward Far
	}
	if next < 0 || next >= len(zoomLevels) {
		return // clamped at the end of the ordered list
	}
	level := zoomLevels[next]
	if next == 0 && !c.ExtraCloseEnabled {
		return // Extra Close gated off (accessibility)
	}
	if !c.transitioning {
		c.ZoomTo(level)
		return
	}
	// Transition in flight: queue at most one level; the last input wins.
	c.pending = &level
}

// Update advances the in-flight zoom transition by dt milliseconds. Call once
// per frame. A non-positive or NaN dt is ignored — the transition neither
// advances nor corrupts state. The lerp is delta-time based and frame-rate
// independent: a long frame advances the transition proportionally, and
// leftover time carries into a queued follow-up transition, so chained inputs
// never stall mid-frame. Each frame feeds the interpolated half-height to
// SetHalfHeight (the CAM-002 transition hook); the target register is applied
// exactly via SetZoom on completion. While the viewport is zero-sized
// (minimized window) the transition is paused and resumes on the first Update
// after a restore.
func (c *Isometric) Update(dt float32) {
	if dt <= 0 || math.IsNaN(float64(dt)) || !c.transitioning {
		return
	}
	vp := c.cam.Viewport()
	if vp.Z() <= 0 || vp.W() <= 0 {
		return // minimized: pause, resume on restore
	}
	dur := c.zoomDuration()
	for c.transitioning {
		c.transitionT += dt / dur
		if c.transitionT < 1 {
			c.SetHalfHeight(c.transitionHalfHeight())
			return
		}
		leftover := (c.transitionT - 1) * dur
		c.settle()
		dt = leftover
	}
}

// settle completes the in-flight transition: applies the target zoom exactly
// via SetZoom, then — if a level is queued — starts the follow-up transition
// from the settled half-height without waiting for another frame.
func (c *Isometric) settle() {
	c.SetZoom(c.transitionTo)
	c.transitioning = false
	if c.pending != nil {
		pending := *c.pending
		c.pending = nil
		c.ZoomTo(pending)
	}
}

// transitionHalfHeight returns the linearly interpolated half-height at the
// current transition progress.
func (c *Isometric) transitionHalfHeight() float32 {
	return c.transitionFrom + (c.transitionTo.HalfHeight-c.transitionFrom)*c.transitionT
}

// CurrentHalfHeight returns the canonical ortho half-height O currently on
// screen: the interpolated value during a transition, or the settled zoom
// level's half-height when idle. This is the canonical zoom parameter per
// ADR-0003 — framing and occlusion code should read this rather than Zoom().
func (c *Isometric) CurrentHalfHeight() float32 {
	return c.halfHeight
}

// Transitioning reports whether a zoom transition is currently in flight.
// While true, ZoomState returns the in-flight target level (the player's
// intent); input and UI systems can use this to debounce detents or reflect
// the transition in HUD state.
func (c *Isometric) Transitioning() bool {
	return c.transitioning
}

// ZoomState returns the discrete zoom level for save/load serialization.
// During an in-flight transition the target level is returned (the player's
// intent); use CurrentHalfHeight for the on-screen value.
func (c *Isometric) ZoomState() ZoomLevel {
	if c.transitioning {
		return c.transitionTo
	}
	return c.zoom
}

// SetZoomState restores a zoom level instantly with no transition — the
// save/load path (Story CAM-002 AC-9). Any in-flight transition and queued
// level are discarded. Context registers are left untouched.
func (c *Isometric) SetZoomState(z ZoomLevel) {
	c.transitioning = false
	c.pending = nil
	c.SetZoom(z)
}

// SetContext switches the camera between gameplay contexts. The zoom of the
// context being left is remembered (the in-flight target if one is set); the
// last zoom of the context being entered is restored with a normal
// transition. Contexts never entered before restore the construction zoom.
func (c *Isometric) SetContext(ctx CameraContext) {
	if ctx == c.context {
		return
	}
	c.setContextZoom(c.context, c.ZoomState())
	c.transitioning = false
	c.pending = nil
	c.context = ctx
	if restored := c.contextZoom(ctx); restored != c.zoom {
		c.ZoomTo(restored)
	}
}

// setContextZoom records the given zoom in the context's persistence register.
func (c *Isometric) setContextZoom(ctx CameraContext, z ZoomLevel) {
	switch ctx {
	case ConversationContext:
		c.lastConversationZoom = z
	case CombatContext:
		c.lastCombatZoom = z
	default:
		c.lastExplorationZoom = z
	}
}

// contextZoom returns the last zoom used in the given context.
func (c *Isometric) contextZoom(ctx CameraContext) ZoomLevel {
	switch ctx {
	case ConversationContext:
		return c.lastConversationZoom
	case CombatContext:
		return c.lastCombatZoom
	default:
		return c.lastExplorationZoom
	}
}

// zoomIndex returns the index of the ordered zoom level matching z's
// half-height, or the index of the nearest level when z is not one of the
// registers (for example a custom level from save data). Returns -1 if the
// level list is empty.
func zoomIndex(z ZoomLevel) int {
	if len(zoomLevels) == 0 {
		return -1
	}
	best, bestDist := 0, float32(math.Inf(1))
	for i, lvl := range zoomLevels {
		dist := lvl.HalfHeight - z.HalfHeight
		if dist < 0 {
			dist = -dist
		}
		if dist < bestDist {
			best, bestDist = i, dist
		}
	}
	return best
}

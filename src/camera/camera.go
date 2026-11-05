// Package camera provides the isometric orthographic camera prescribed by
// ADR-0003 and the Camera System GDD. It wraps Kaiju's StandardCameraOrthographic
// with a fixed 30°/45° isometric angle, discrete zoom levels, aspect-ratio-safe
// resize, and no rotation path (architecturally locked — the player cannot
// rotate the view).
//
// The ortho half-height (O) is the canonical zoom parameter: the vertical
// extent is always 2*O world units and the horizontal extent scales with the
// viewport aspect ratio (clamped to a 16:10 minimum). The near clip plane is
// zoom-derived (near = -0.75 * O, per GDD CR-1) so the ground plane reaches
// the viewport bottom edge at every zoom — no void strip.
package camera

import (
	"math"

	"kaijuengine.com/engine/cameras"
	"kaijuengine.com/matrix"
)

// ── Configuration ───────────────────────────────────────────────────────────

// Config holds the fixed isometric camera parameters from ADR-0003 and the
// Camera System GDD (CR-1).
type Config struct {
	Pitch          float32 // degrees above horizontal, fixed at 30
	Yaw            float32 // degrees around the vertical axis, fixed at 45
	NearClipFactor float32 // near clip = -NearClipFactor * O (GDD CR-1), default 0.75
	FarClip        float32 // far plane in world units, fixed at 1000
	MinAspect      float32 // minimum aspect ratio (width/height), fixed at 16:10

	// ZoomTransitionMs is the duration of a zoom level transition in
	// milliseconds (Story CAM-002 / ADR-0003: 300). Values <= 0 fall back
	// to 300ms so a zero-value Config can never produce an instant snap.
	ZoomTransitionMs float32
}

// DefaultConfig returns the ADR-0003 / GDD CR-1 prescribed configuration.
func DefaultConfig() Config {
	return Config{
		Pitch:            30,
		Yaw:              45,
		NearClipFactor:   0.75,
		FarClip:          1000,
		MinAspect:        16.0 / 10.0,
		ZoomTransitionMs: 300,
	}
}

// ── Zoom ────────────────────────────────────────────────────────────────────

// ZoomLevel is a discrete zoom preset. The HalfHeight is the orthographic
// half-height O in world units — the canonical zoom parameter per ADR-0003.
type ZoomLevel struct {
	Name       string
	HalfHeight float32
}

// Canonical zoom registers from ADR-0003 and the Camera System GDD.
// Extra Close is the accessibility level (opt-in via settings, GDD CR-2).
var (
	ZoomExtraClose = ZoomLevel{"Extra Close", 2.5}
	ZoomClose      = ZoomLevel{"Close", 4}
	ZoomDefault    = ZoomLevel{"Default", 10}
	ZoomFar        = ZoomLevel{"Far", 20}
)

// ── Camera ──────────────────────────────────────────────────────────────────

// Isometric wraps a Kaiju StandardCameraOrthographic in a fixed isometric
// projection. The view angle (30° pitch / 45° yaw / 0° roll) is baked in at
// construction and cannot be changed — there is no rotation escape hatch.
type Isometric struct {
	cam        *cameras.StandardCamera
	config     Config
	target     matrix.Vec3
	zoom       ZoomLevel // most recently set zoom register (lags during transitions)
	halfHeight float32   // canonical ortho half-height O (world units)

	// Zoom transition state (Story CAM-002). transitioning is false when the
	// camera is idle; when true, a ZoomTo target is in flight and Update
	// advances it.
	transitioning  bool
	transitionT    float32    // lerp progress in [0, 1]
	transitionFrom float32    // half-height at transition start
	transitionTo   ZoomLevel  // target level of the in-flight transition
	pending        *ZoomLevel // at most one queued level; last input wins

	// ExtraCloseEnabled gates ZoomExtraClose: when false, wheel-up at Close
	// is a no-op. Accessibility-controlled (Story CAM-002 AC-6).
	ExtraCloseEnabled bool

	// Per-context zoom persistence (Story CAM-002 AC-8). Each register holds
	// the last zoom used in its context; SetContext restores it on re-entry.
	context              CameraContext
	lastExplorationZoom  ZoomLevel
	lastConversationZoom ZoomLevel
	lastCombatZoom       ZoomLevel
}

// NewIsometric creates a new orthographic camera at the fixed isometric angle,
// positioned per the GDD formula and looking at target. viewWidth and
// viewHeight are the initial viewport dimensions in pixels; the orthographic
// extents are world units derived from the zoom level's half-height and the
// clamped aspect ratio.
func NewIsometric(target matrix.Vec3, cfg Config, zoom ZoomLevel, viewWidth, viewHeight float32) *Isometric {
	c := &Isometric{
		config:     cfg,
		target:     target,
		zoom:       zoom,
		halfHeight: zoom.HalfHeight,
		// Context registers start at the construction zoom; contexts never
		// entered before restore this value (Story CAM-002 AC-8).
		lastExplorationZoom:  zoom,
		lastConversationZoom: zoom,
		lastCombatZoom:       zoom,
	}
	pos := isometricPosition(target, cfg, zoom.HalfHeight)
	worldWidth, worldHeight := orthoExtents(zoom.HalfHeight, viewWidth, viewHeight, cfg.MinAspect)

	cam := cameras.NewStandardCameraOrthographic(worldWidth, worldHeight, viewWidth, viewHeight, pos)
	// FOV is unused by orthographic cameras; near/far and extents set in one pass.
	cam.SetProperties(0, c.nearClip(), cfg.FarClip, worldWidth, worldHeight)
	cam.SetPositionAndLookAtWithUp(pos, target, matrix.Vec3Up())

	c.cam = cam
	return c
}

// Resize recalculates orthographic extents for a new viewport size. The
// ortho half-height (current O — the interpolated value mid-transition) is
// preserved; the horizontal extent adjusts to the new aspect ratio, clamped
// to the minimum aspect ratio per ADR-0003. A zero-size viewport (window
// minimized) keeps the last extents per the GDD edge case.
func (c *Isometric) Resize(viewWidth, viewHeight float32) {
	if viewWidth <= 0 || viewHeight <= 0 {
		return
	}
	worldWidth, worldHeight := orthoExtents(c.halfHeight, viewWidth, viewHeight, c.config.MinAspect)
	c.cam.SetProperties(0, c.nearClip(), c.config.FarClip, worldWidth, worldHeight)
	c.cam.ViewportChanged(viewWidth, viewHeight)
}

// SetZoom changes the zoom level to the register's canonical half-height and
// re-applies position and orthographic extents.
func (c *Isometric) SetZoom(zoom ZoomLevel) {
	c.zoom = zoom
	c.SetHalfHeight(zoom.HalfHeight)
}

// SetHalfHeight sets the canonical ortho half-height O and immediately
// re-applies the camera position and orthographic extents from it. This is
// the hook the zoom transition system (CAM-002) calls each frame with the
// interpolated O — a resize arriving mid-transition therefore re-applies
// extents from the current interpolated value (story AC-4). Non-positive
// values are ignored.
func (c *Isometric) SetHalfHeight(halfHeight float32) {
	if halfHeight <= 0 {
		return
	}
	c.halfHeight = halfHeight
	c.applyPosition()
}

// SetFocus moves the camera focus (the world-space look-at target) to pos and
// immediately re-applies the camera position at the current half-height. This
// is the follow/framing hook of Story CAM-003 and the GDD's map-click-to-center
// API: the camera always looks at exactly the given point at the current zoom.
func (c *Isometric) SetFocus(pos matrix.Vec3) {
	c.target = pos
	c.applyPosition()
}

// applyPosition re-applies the camera position and orthographic extents from
// the current focus target and half-height. Shared by SetHalfHeight (zoom
// transitions) and SetFocus (follow/framing). A zero-sized viewport (window
// minimized) skips the extent update; extents are re-applied on restore.
func (c *Isometric) applyPosition() {
	pos := isometricPosition(c.target, c.config, c.halfHeight)
	c.cam.SetPositionAndLookAtWithUp(pos, c.target, matrix.Vec3Up())
	vp := c.cam.Viewport()
	if vp.Z() <= 0 || vp.W() <= 0 {
		return // minimized window: extents re-applied on restore (GDD edge case)
	}
	worldWidth, worldHeight := orthoExtents(c.halfHeight, vp.Z(), vp.W(), c.config.MinAspect)
	c.cam.SetProperties(0, c.nearClip(), c.config.FarClip, worldWidth, worldHeight)
}

// SetTarget retargets the camera at a new world-space focus point and
// re-applies position from the current half-height, preserving zoom and
// extents. This is the integration hook for region boundary clamping (Story
// CAM-004 AC-8): the boundary clamps the requested target and feeds the
// clamped result back here.
func (c *Isometric) SetTarget(target matrix.Vec3) {
	c.target = target
	pos := isometricPosition(target, c.config, c.halfHeight)
	c.cam.SetPositionAndLookAtWithUp(pos, target, matrix.Vec3Up())
}

// ── Accessors (read-only — no rotation or position mutation exposed) ────────

// Camera returns the underlying Kaiju camera. Use only for read operations
// (adding to host, reading matrices) — never for SetLookAt, SetPosition or
// SetLookAtWithUp, which would break the isometric contract.
func (c *Isometric) Camera() *cameras.StandardCamera { return c.cam }

// Position returns the camera position in world space.
func (c *Isometric) Position() matrix.Vec3 { return c.cam.Position() }

// Focus returns the current focus point — the world-space look-at target the
// camera is centered on (GDD CR-3). Follow/framing systems read this as the
// camera's current center and move it via SetFocus.
func (c *Isometric) Focus() matrix.Vec3 { return c.target }

// ViewportSize returns the current viewport dimensions in logical pixels.
// The returned height is the screen height used for world-per-pixel scaling;
// a zero-sized viewport means the window is minimized.
func (c *Isometric) ViewportSize() (width, height float32) {
	vp := c.cam.Viewport()
	return vp.Z(), vp.W()
}

// Zoom returns the zoom register most recently set via NewIsometric or
// SetZoom. During a transition (SetHalfHeight), the register lags the
// interpolated value — use HalfHeight for the current canonical O.
func (c *Isometric) Zoom() ZoomLevel { return c.zoom }

// HalfHeight returns the current canonical ortho half-height O in world units.
func (c *Isometric) HalfHeight() float32 { return c.halfHeight }

// Config returns the camera configuration.
func (c *Isometric) Config() Config { return c.config }

// ── Internal helpers ────────────────────────────────────────────────────────

// nearClip returns the zoom-derived near clip plane distance per GDD CR-1:
// near = -0.75 * O. The negative near plane prevents ground-plane clipping at
// the viewport bottom edge — a fixed +1 near plane created a 22–33% void strip
// at every zoom level. Kaiju SetNearPlane accepts negative values (its
// orthographic default is -1).
func (c *Isometric) nearClip() float32 {
	return -c.config.NearClipFactor * c.halfHeight
}

// isometricPosition computes the camera position for a target at the given
// ortho half-height O, per the GDD formula:
//
//	camera_x = focus_x - O * cos(pitch) * sin(yaw)
//	camera_y = focus_y + O * sin(pitch)   (= 0.5 * O above the focus)
//	camera_z = focus_z - O * cos(pitch) * cos(yaw)
//
// The standoff equals O, so the camera height is 0.5 * O (1.25u Extra Close
// to 10u Far). The no-void near-clip guarantee (near = -0.75 * O) is derived
// from this position.
func isometricPosition(target matrix.Vec3, cfg Config, halfHeight float32) matrix.Vec3 {
	pitchRad := float64(matrix.Deg2Rad(cfg.Pitch))
	yawRad := float64(matrix.Deg2Rad(cfg.Yaw))
	dx := float32(math.Cos(pitchRad) * math.Sin(yawRad))
	dy := float32(math.Sin(pitchRad))
	dz := float32(math.Cos(pitchRad) * math.Cos(yawRad))
	return matrix.Vec3{
		target.X() - dx*halfHeight,
		target.Y() + dy*halfHeight,
		target.Z() - dz*halfHeight,
	}
}

// orthoExtents returns the world-space (width, height) extents for the given
// ortho half-height O and viewport dimensions. The vertical extent is 2*O —
// zoom independent of window size — and the horizontal extent scales with the
// aspect ratio, clamped to the minimum aspect ratio.
func orthoExtents(halfHeight, viewWidth, viewHeight, minAspect float32) (float32, float32) {
	aspect := viewWidth / viewHeight
	if aspect < minAspect {
		aspect = minAspect
	}
	worldHeight := halfHeight * 2 // full height = 2 × half-height
	worldWidth := worldHeight * aspect
	return worldWidth, worldHeight
}

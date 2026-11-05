// Manual camera panning for Story CAM-003 (GDD CR-4): drag pan with
// point-under-cursor tracking, edge scroll with hysteresis, keyboard pan, and
// the manual-position hold contract (no auto-return — the player recovers via
// Home / End / selection, handled by the Controller and Follower).
//
// All pan paths move the camera focus in the XZ plane at a constant elevation
// and raise the Manual overlay through the Controller; the camera then holds
// the manual position until the player re-centers. Pan vectors follow the
// isometric screen axes: screen-right = (cos45°, 0, −sin45°) and screen-up =
// normalize(sin45°·sin30°, 0, cos45°·sin30°) — the XZ projection of the
// screen-up direction.
package camera

import (
	"math"

	"kaijuengine.com/matrix"
)

// PanConfig carries the pan tuning knobs (GDD CR-4 / tuning knob table).
// DefaultPanConfig returns the GDD values.
type PanConfig struct {
	// EdgeScrollSpeed is the edge-scroll pan speed in world units per second
	// (GDD pan_speed_edge, default 15).
	EdgeScrollSpeed float32
	// KeyboardPanSpeed is the keyboard pan speed in world units per second
	// (GDD pan_speed_keyboard, default 20).
	KeyboardPanSpeed float32
	// EdgeScrollDeadzonePx is the cursor distance from a viewport edge in
	// logical pixels that activates edge scrolling (GDD edge_scroll_deadzone,
	// default 16).
	EdgeScrollDeadzonePx float32
	// EdgeScrollHysteresis is the fraction of the deadzone added to the exit
	// threshold: edge scrolling stops only when the cursor retreats past
	// deadzone × (1 + hysteresis) (GDD edge_scroll_hysteresis, default 0.5).
	EdgeScrollHysteresis float32
	// DragThresholdPx is the pixel distance a drag must exceed before panning
	// engages (GDD drag_pan_threshold, default 8). Prevents micro-drift.
	DragThresholdPx float32
	// DragSensitivity scales the drag pan world-per-pixel delta (GDD
	// drag_pan_sensitivity, default 1.0, range 0.25–2.0).
	DragSensitivity float32
}

// DefaultPanConfig returns the GDD-prescribed pan tuning values.
func DefaultPanConfig() PanConfig {
	return PanConfig{
		EdgeScrollSpeed:       15,
		KeyboardPanSpeed:      20,
		EdgeScrollDeadzonePx:  16,
		EdgeScrollHysteresis:  0.5,
		DragThresholdPx:       8,
		DragSensitivity:       1.0,
	}
}

// panRight is the world direction of screen-right for the 45° yaw: moving the
// focus along it pans the view right.
var panRight = matrix.NewVec3(isoCos45, 0, -isoCos45)

// panUp is the world direction of screen-up (XZ projection): moving the focus
// along it pans the view up-screen.
var panUp = matrix.NewVec3(isoCos45, 0, isoCos45)

// Pan implements the GDD CR-4 manual pan controls. It owns input state only;
// the Controller decides whether a pan activation raises the Manual overlay.
type Pan struct {
	cam *Isometric
	cfg PanConfig

	// Drag state.
	dragDown     bool // pan button held
	dragEngaged  bool // pixel threshold crossed; panning active
	dragLastX    float32
	dragLastY    float32
	dragTotalX   float32 // accumulated distance for the threshold check
	dragTotalY   float32

	// Edge-scroll state (with hysteresis, GDD CR-4).
	edgeActive   bool
	edgeDirX     float32 // pan direction while active (screen space, ±1/0)
	edgeDirY     float32
	cursorX      float32 // last cursor position in logical pixels
	cursorY      float32
	cursorKnown  bool
	cursorOverUI bool // edge scroll disabled over UI panels (GDD CR-4)
}

// NewPan creates a Pan controller for the given camera and config.
func NewPan(cam *Isometric, cfg PanConfig) *Pan {
	return &Pan{cam: cam, cfg: cfg}
}

// Active reports whether a pan is currently moving the camera: an engaged
// drag or an active edge scroll. The Controller watches the rising edge of
// this flag to raise the Manual overlay.
func (p *Pan) Active() bool {
	return p.dragEngaged || p.edgeActive
}

// ── Drag pan (GDD CR-4) ──────────────────────────────────────────────────────

// PanStart begins a drag-pan gesture at the given cursor position in logical
// pixels (the pan button just went down). Panning engages once the cursor has
// moved at least drag_pan_threshold pixels; until then the gesture can be a
// click and nothing moves (no micro-drift).
func (p *Pan) PanStart(x, y float32) {
	p.dragDown = true
	p.dragEngaged = false
	p.dragLastX = x
	p.dragLastY = y
	p.dragTotalX = 0
	p.dragTotalY = 0
}

// PanDrag continues a drag-pan gesture with the current cursor position.
// While engaged, the focus moves so the world-space point under the cursor
// stays under the cursor (point-under-cursor tracking, AC-15): the focus
// delta is the negative cursor delta scaled by the world-per-pixel rate of
// the current zoom, along the isometric screen axes.
func (p *Pan) PanDrag(x, y float32) {
	if !p.dragDown {
		return
	}
	dx := x - p.dragLastX
	dy := y - p.dragLastY
	p.dragLastX = x
	p.dragLastY = y
	if !p.dragEngaged {
		p.dragTotalX += dx
		p.dragTotalY += dy
		if abs32(p.dragTotalX) < p.cfg.DragThresholdPx && abs32(p.dragTotalY) < p.cfg.DragThresholdPx {
			return // threshold not crossed: still a candidate click
		}
		p.dragEngaged = true
	}
	// Grab semantics: the world point under the cursor follows the cursor.
	// Sensitivity scales the world-per-pixel rate (drag pan only, GDD CR-4).
	p.panByScreenDelta(-dx*p.cfg.DragSensitivity, -dy*p.cfg.DragSensitivity)
}

// PanEnd ends a drag-pan gesture (the pan button released). The camera holds
// the manual position — no auto-return (AC-15).
func (p *Pan) PanEnd() {
	p.dragDown = false
	p.dragEngaged = false
}

// ── Edge scroll (GDD CR-4) ───────────────────────────────────────────────────

// SetCursor updates the cursor position in logical pixels each frame. Edge
// scrolling reads it during Update.
func (p *Pan) SetCursor(x, y float32) {
	p.cursorX = x
	p.cursorY = y
	p.cursorKnown = true
}

// SetCursorOverUI reports whether the cursor is over a UI panel. When true,
// edge scrolling is disabled (GDD CR-4: "Edge scroll DISABLED when cursor is
// over a UI panel"); drag and keyboard pan remain functional.
func (p *Pan) SetCursorOverUI(over bool) {
	p.cursorOverUI = over
}

// Update advances the edge-scroll state by dt seconds: the cursor inside the
// edge deadzone activates scrolling toward the nearest edge at
// EdgeScrollSpeed; once active, scrolling continues until the cursor retreats
// past deadzone × (1 + hysteresis) — the hysteresis margin prevents jitter at
// the boundary (GDD CR-4). Edge scrolling never starts over a UI panel.
func (p *Pan) Update(dt float32) {
	if dt <= 0 || math.IsNaN(float64(dt)) {
		return
	}
	deadzone := p.cfg.EdgeScrollDeadzonePx
	exit := deadzone * (1 + p.cfg.EdgeScrollHysteresis)
	if !p.cursorKnown || p.cursorOverUI {
		p.edgeActive = false
		p.edgeDirX, p.edgeDirY = 0, 0
		return
	}
	w, h := p.cam.ViewportSize()
	if w <= 0 || h <= 0 {
		p.edgeActive = false
		p.edgeDirX, p.edgeDirY = 0, 0
		return
	}
	// Distance from the cursor to each edge (y down-from-top window space).
	dLeft := p.cursorX
	dRight := w - p.cursorX
	dTop := p.cursorY
	dBottom := h - p.cursorY
	dirX, dirY, nearest := p.edgeDirection(dLeft, dRight, dTop, dBottom)
	threshold := deadzone
	if p.edgeActive {
		threshold = exit // hysteresis: active scroll requires deeper retreat
	}
	if nearest > threshold {
		p.edgeActive = false
		p.edgeDirX, p.edgeDirY = 0, 0
		return
	}
	p.edgeActive = true
	p.edgeDirX, p.edgeDirY = dirX, dirY
	p.panByWorldDelta(p.edgeDirX*p.cfg.EdgeScrollSpeed*dt, p.edgeDirY*p.cfg.EdgeScrollSpeed*dt)
}

// edgeDirection returns the screen-space pan direction for the nearest edge
// (pans right when the cursor hugs the right edge, up at the top edge, ...)
// and the cursor's distance to that edge.
func (p *Pan) edgeDirection(dLeft, dRight, dTop, dBottom float32) (dirX, dirY, nearest float32) {
	dirX, dirY, nearest = 0, 0, dLeft
	if dRight < nearest {
		dirX, dirY, nearest = 1, 0, dRight
	}
	if dTop < nearest {
		dirX, dirY, nearest = 0, -1, dTop
	}
	if dBottom < nearest {
		dirX, dirY, nearest = 0, 1, dBottom
	}
	return dirX, dirY, nearest
}

// EdgeScrolling reports whether edge scrolling is currently active (used by
// the Controller's Manual-overlay wiring).
func (p *Pan) EdgeScrolling() bool { return p.edgeActive }

// ── Keyboard pan (GDD CR-4) ──────────────────────────────────────────────────

// KeyboardPan pans the camera at KeyboardPanSpeed (world units per second)
// for dt seconds along the given screen-space direction (dirX/dirY in [−1, 1];
// e.g. W = (0, −1), D = (1, 0), diagonal normalized by the caller). The input
// system calls this each frame while a pan key is held; the Controller raises
// the Manual overlay on nonzero input.
func (p *Pan) KeyboardPan(dirX, dirY, dt float32) {
	if dt <= 0 || math.IsNaN(float64(dt)) || (dirX == 0 && dirY == 0) {
		return
	}
	p.panByWorldDelta(dirX*p.cfg.KeyboardPanSpeed*dt, dirY*p.cfg.KeyboardPanSpeed*dt)
}

// ── Motion ───────────────────────────────────────────────────────────────────

// panByWorldDelta moves the camera focus by a world-space delta along the
// screen-horizontal (panRight) and screen-vertical (panUp) directions.
// The focus elevation (look-at Y) is preserved — panning never changes the
// camera height.
func (p *Pan) panByWorldDelta(dxWorld, dyWorld float32) {
	focus := p.cam.Focus()
	delta := panRight.Scale(dxWorld)
	delta = delta.Add(panUp.Scale(dyWorld))
	p.cam.SetFocus(matrix.NewVec3(focus.X()+delta.X(), focus.Y(), focus.Z()+delta.Z()))
}

// panByScreenDelta moves the camera focus by a screen-space delta: the focus
// moves along panRight for the horizontal component and panUp for the
// vertical component, scaled by the world-per-pixel rate of the current zoom.
// The focus elevation (look-at Y) is preserved — panning never changes the
// camera height.
func (p *Pan) panByScreenDelta(dxScreen, dyScreen float32) {
	w, h := p.cam.ViewportSize()
	if w <= 0 || h <= 0 {
		return // minimized window: no panning
	}
	o := p.cam.CurrentHalfHeight()
	// World units per logical pixel: the full viewport width spans 2×O×AR
	// world units; the full height spans 2×O.
	worldPerPixelX := (2 * o * p.aspect()) / w
	worldPerPixelY := (2 * o) / h
	focus := p.cam.Focus()
	delta := panRight.Scale(dxScreen * worldPerPixelX)
	delta = delta.Add(panUp.Scale(dyScreen * worldPerPixelY))
	p.cam.SetFocus(matrix.NewVec3(focus.X()+delta.X(), focus.Y(), focus.Z()+delta.Z()))
}

// aspect returns the viewport aspect ratio clamped to the camera minimum
// (mirrors the Follower's aspect; the pan world-per-pixel scale must match
// the camera's orthographic extents).
func (p *Pan) aspect() float32 {
	w, h := p.cam.ViewportSize()
	if w <= 0 || h <= 0 {
		return 16.0 / 9.0
	}
	a := w / h
	if a < p.cam.Config().MinAspect {
		a = p.cam.Config().MinAspect
	}
	return a
}

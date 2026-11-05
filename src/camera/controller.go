// Controller is the composition root of the Story CAM-003 camera system: it
// owns the State machine, the Follower, and the Pan controller, and drives
// them from one Update per frame. The game layer drives the Controller from
// gameplay events (combat/dialogue/journal starts and ends) and input (pan,
// Home, End); the Controller translates state transitions into framing
// lifecycles and overlay flags.
//
// Update ordering per frame: zoom transitions (CAM-002) first, then edge
// scroll, then follow/framing motion. Manual-overlay arbitration follows the
// GDD: Manual suspends exploration/conversation follow; combat framing still
// runs under Manual for the joiner re-frame override (AC-13) but holds under
// Journal and Transition.
package camera

import (
	"math"

	"marrow-of-empire/entity"
)

// Controller owns the camera follow/framing system state.
type Controller struct {
	cam    *Isometric
	state  State
	follow *Follower
	pan    *Pan

	// panWasActive tracks the rising edge of pan activity to raise the
	// Manual overlay exactly when a pan engages.
	panWasActive bool
}

// NewController creates a Controller for the given camera, position source,
// and follow/pan configs. The camera starts in the Exploration base state
// with no overlays — the game-start configuration.
//
// Example usage:
//
//	cam := camera.NewIsometric(matrix.Vec3{}, camera.DefaultConfig(),
//	    camera.ZoomDefault, 1920, 1080)
//	ctl := camera.NewController(cam,
//	    camera.NewHostPositionSource(host),
//	    camera.DefaultFollowConfig(), camera.DefaultPanConfig())
//	ctl.TrackPartyMember(partyLeaderID) // wired to component.added
//	// per frame: ctl.Update(dt)
func NewController(cam *Isometric, pos PositionSource, followCfg FollowConfig, panCfg PanConfig) *Controller {
	return &Controller{
		cam:    cam,
		state:  NewState(),
		follow: NewFollower(cam, pos, followCfg),
		pan:    NewPan(cam, panCfg),
	}
}

// ── Component access (wiring surface for the game layer) ─────────────────────

// Camera returns the underlying isometric camera.
func (c *Controller) Camera() *Isometric { return c.cam }

// State returns the state machine. The game layer may read base state and
// overlays for HUD and save/load; transitions go through the Controller
// methods so framing lifecycles stay in sync.
func (c *Controller) State() *State { return &c.state }

// Follow returns the Follower. The game layer uses it to wire membership and
// context (party, combatants, player character, speaker, selection).
func (c *Controller) Follow() *Follower { return c.follow }

// Pan returns the Pan controller. The input system drives drag, edge scroll,
// and keyboard pan through it.
func (c *Controller) Pan() *Pan { return c.pan }

// ── Membership & context (wired to EF component lifecycle events) ────────────

// TrackPartyMember adds an entity to the tracked party set (game layer: wire
// to component.added for PartyMember). See Follower.TrackPartyMember.
func (c *Controller) TrackPartyMember(id entity.GameEntityID) { c.follow.TrackPartyMember(id) }

// UntrackPartyMember removes an entity from the tracked party set (game
// layer: wire to component.removed). See Follower.UntrackPartyMember.
func (c *Controller) UntrackPartyMember(id entity.GameEntityID) { c.follow.UntrackPartyMember(id) }

// TrackCombatant adds an entity to the tracked combatant set (game layer:
// wire to component.added for Combatant). Joiner re-frame events are
// signalled separately via NoteCombatantJoined so the combat engine controls
// encounter boundaries.
func (c *Controller) TrackCombatant(id entity.GameEntityID) { c.follow.TrackCombatant(id) }

// UntrackCombatant removes an entity from the tracked combatant set (game
// layer: wire to component.removed). See Follower.UntrackCombatant.
func (c *Controller) UntrackCombatant(id entity.GameEntityID) { c.follow.UntrackCombatant(id) }

// SetPlayerCharacter sets the player character for conversation framing.
func (c *Controller) SetPlayerCharacter(id entity.GameEntityID) { c.follow.SetPlayerCharacter(id) }

// SetSpeaker sets the speaking NPC; the Follower applies the inner-70% /
// override-window re-frame rules (AC-5, AC-6).
func (c *Controller) SetSpeaker(id entity.GameEntityID) { c.follow.SetSpeaker(id) }

// SetSelected sets the selected character and clears the Manual overlay —
// selecting a party member re-engages the base state's follow (GDD CR-3).
func (c *Controller) SetSelected(id entity.GameEntityID) {
	c.follow.SetSelected(id)
	c.state.SetManual(false)
}

// NoteTurnBoundary signals a combat turn boundary; the Follower applies the
// turn-boundary re-frame rule (AC-12).
func (c *Controller) NoteTurnBoundary() { c.follow.NoteTurnBoundary() }

// NoteCombatantJoined signals new combatants joining mid-encounter; the
// Follower applies the immediate-or-next-turn re-frame rule (AC-13).
func (c *Controller) NoteCombatantJoined() { c.follow.NoteCombatantJoined() }

// ── Base-state transitions (gameplay lifecycle) ──────────────────────────────

// RequestConversation enters the Conversation base state (player initiates
// dialogue): the camera restores the last Conversation zoom and frames the
// PC+NPC midpoint (AC-4). Ignored while Combat is active (priority).
func (c *Controller) RequestConversation() {
	before := c.state.Base()
	c.state.RequestConversation()
	if c.state.Base() == ConversationBase && before != ConversationBase {
		c.cam.SetContext(ConversationContext)
		c.follow.StartConversation()
		c.state.SetManual(false)
	}
}

// EndConversation leaves the Conversation base state (dialogue ends): the
// camera returns to Exploration at the last Exploration zoom (AC-8) and the
// Manual overlay is cleared so the player is never stranded (AC-22).
func (c *Controller) EndConversation() {
	if c.state.Base() != ConversationBase {
		return
	}
	c.state.EndConversation()
	c.follow.EndConversation()
	c.state.SetManual(false)
	c.cam.SetContext(ExplorationContext)
}

// RequestCombat enters the Combat base state (encounter starts): the camera
// restores the Combat zoom context and frames the combatant bounding box
// (AC-9..AC-11). Combat preempts Conversation; EndCombat returns to it.
func (c *Controller) RequestCombat() {
	before := c.state.Base()
	c.state.RequestCombat()
	if c.state.Base() == CombatBase && before != CombatBase {
		c.cam.SetContext(CombatContext)
		c.follow.StartCombat()
		c.state.SetManual(false)
	}
}

// EndCombat leaves the Combat base state (encounter ends): the camera returns
// to the preempted Conversation (AC-18) or to Exploration (AC-14) at the
// appropriate zoom context, and follow resumes.
func (c *Controller) EndCombat() {
	if c.state.Base() != CombatBase {
		return
	}
	c.state.EndCombat()
	c.follow.EndCombat()
	c.state.SetManual(false)
	switch c.state.Base() {
	case ConversationBase:
		// Combat preempted a conversation: re-apply its framing.
		c.cam.SetContext(ConversationContext)
		c.follow.StartConversation()
	default:
		c.cam.SetContext(ExplorationContext)
	}
}

// BeginTransition enters the exclusive Transition base state (region
// boundary crossed, GDD CR-6): follow and framing hold while the fade plays.
// Requests during the transition are queued by the State and applied on
// EndTransition.
func (c *Controller) BeginTransition() {
	c.state.BeginTransition()
	c.follow.CancelFraming()
}

// EndTransition leaves the Transition base state after the fade-in: the
// pre-transition base state is restored and any queued request applied (GDD
// CR-6: combat/conversation events queued during the fade are processed
// after fade-in).
func (c *Controller) EndTransition() {
	before := c.state.Base()
	c.state.EndTransition()
	after := c.state.Base()
	if before == after {
		return
	}
	c.state.SetManual(false)
	switch after {
	case CombatBase:
		c.cam.SetContext(CombatContext)
		c.follow.StartCombat()
	case ConversationBase:
		c.cam.SetContext(ConversationContext)
		c.follow.StartConversation()
	default:
		c.cam.SetContext(ExplorationContext)
	}
}

// ── Overlay flags ────────────────────────────────────────────────────────────

// OpenJournal sets the Journal overlay: the camera holds position while the
// investigation journal is open; the base state is unchanged (AC-19, AC-20).
func (c *Controller) OpenJournal() { c.state.OpenJournal() }

// CloseJournal clears the Journal overlay: the camera returns to the base
// state framing active before the journal opened — never unconditionally
// Exploration (AC-20).
func (c *Controller) CloseJournal() { c.state.CloseJournal() }

// ── Instant centering (Home / End keys, GDD CR-3) ────────────────────────────

// Home instantly centers on the party centroid (AC-16) and clears the Manual
// overlay so base follow resumes. Zoom unchanged.
func (c *Controller) Home() {
	c.follow.Home()
	c.state.SetManual(false)
}

// CenterOnSelected instantly centers on the selected character (AC-17) and
// clears the Manual overlay. With no selection it is a no-op.
func (c *Controller) CenterOnSelected() {
	c.follow.CenterOnSelected()
	c.state.SetManual(false)
}

// ── Pan input (exposed APIs; input binding lives in the input system) ────────

// PanStart begins a drag-pan gesture (pan button down). See Pan.PanStart.
func (c *Controller) PanStart(x, y float32) { c.pan.PanStart(x, y) }

// PanDrag continues a drag-pan gesture (pan button held, cursor moved).
// Engaging the drag raises the Manual overlay and notes a manual adjustment
// (override window, combat re-frame suppression).
func (c *Controller) PanDrag(x, y float32) {
	c.pan.PanDrag(x, y)
	if c.pan.Active() {
		c.raiseManual()
	}
}

// PanEnd ends a drag-pan gesture (pan button released). The camera holds the
// manual position — no auto-return (AC-15).
func (c *Controller) PanEnd() { c.pan.PanEnd() }

// SetCursor updates the cursor position for edge scrolling.
func (c *Controller) SetCursor(x, y float32) { c.pan.SetCursor(x, y) }

// SetCursorOverUI disables edge scrolling while the cursor is over a UI panel
// (GDD CR-4).
func (c *Controller) SetCursorOverUI(over bool) { c.pan.SetCursorOverUI(over) }

// KeyboardPan pans the camera along the given screen-space direction at the
// keyboard pan speed. Any nonzero input raises the Manual overlay.
func (c *Controller) KeyboardPan(dirX, dirY, dt float32) {
	if dirX == 0 && dirY == 0 {
		return
	}
	c.pan.KeyboardPan(dirX, dirY, dt)
	c.raiseManual()
}

// raiseManual sets the Manual overlay and notes a manual adjustment on the
// Follower (override window + combat re-frame suppression + framing cancel).
func (c *Controller) raiseManual() {
	c.state.SetManual(true)
	c.follow.NoteManualAdjust()
}

// ── Per-frame update ─────────────────────────────────────────────────────────

// Update advances the camera system by dt seconds: zoom transitions first
// (CAM-002 — the zoom transition API advances in milliseconds, so dt is
// converted), then edge scroll, then follow/framing motion. Follow is
// suspended by the Manual, Journal, and Transition overlays; combat framing
// continues under Manual (joiner re-frame override, AC-13) but holds under
// Journal and Transition. A non-positive or NaN dt is ignored. This path is
// allocation-free (see Follower.Update).
func (c *Controller) Update(dt float32) {
	if dt <= 0 || math.IsNaN(float64(dt)) {
		return
	}
	c.cam.Update(dt * 1000) // zoom transitions (CAM-002 API takes milliseconds)
	c.pan.Update(dt)        // edge scroll
	if c.pan.EdgeScrolling() {
		c.raiseManual()
	}
	switch c.state.Base() {
	case CombatBase:
		if !c.state.Journal() {
			c.follow.Update(dt, CombatBase)
		}
	case ExplorationBase, ConversationBase:
		if !c.state.Manual() && !c.state.Journal() {
			c.follow.Update(dt, c.state.Base())
		}
	}
}

// ── Marker data contract (camera → HUD) ──────────────────────────────────────

// OffScreenMarkers returns the edge markers for tracked entities outside the
// viewport (GDD Off-Screen Markers, AC-23). See Follower.OffScreenMarkers for
// the zero-allocation scratch-buffer contract.
func (c *Controller) OffScreenMarkers() []EdgeMarker {
	return c.follow.OffScreenMarkers()
}

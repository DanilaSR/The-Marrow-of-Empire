// Camera state machine for Story CAM-003: base states + overlay flags, base
// state priority, combat preemption of conversation (one-deep stack), and
// journal handling (GDD CR-3 "States and Transitions").
//
// The state model is deliberately a plain value type with no camera or entity
// dependencies: it only answers "what is the camera supposed to be doing" —
// which base state is active, whether the Manual or Journal overlays are set,
// and which state combat will return to when it ends. The Controller
// (controller.go) owns this State and translates its answers into camera
// motion via the Follower and Pan.
package camera

// BaseState is the active camera context (GDD CR-3). Exactly one base state
// is active at a time; overlay flags (Manual, Journal) modify behavior on top
// of it. Base-state priority, highest first: Transition, Combat, Conversation,
// Exploration.
type BaseState int

const (
	// ExplorationBase is the default state: smooth follow of the party
	// centroid, all pan modes enabled, the player's last Exploration zoom.
	ExplorationBase BaseState = iota
	// ConversationBase is active during NPC dialogue: frames PC+NPC midpoint
	// on entry, player retains full pan and zoom.
	ConversationBase
	// CombatBase is active during combat encounters: frames the combatant
	// bounding box once at start, no centroid chase afterwards.
	CombatBase
	// TransitionBase is exclusive: region-boundary fades. No other state is
	// active during a transition; requests made while transitioning are
	// queued and applied after the fade-in (GDD CR-6).
	TransitionBase
)

// String returns the base state name for logging and test output.
func (s BaseState) String() string {
	switch s {
	case ExplorationBase:
		return "Exploration"
	case ConversationBase:
		return "Conversation"
	case CombatBase:
		return "Combat"
	case TransitionBase:
		return "Transition"
	default:
		return "Unknown"
	}
}

// State is the camera state machine (GDD CR-3 "States and Transitions").
//
// Rules enforced:
//   - Base-state priority: Transition > Combat > Conversation > Exploration.
//     A request for a lower-priority state while a higher-priority base state
//     is active is ignored (Conversation cannot interrupt Combat).
//   - Combat preempts Conversation: entering Combat while in Conversation
//     pushes Conversation onto a one-deep stack; ending Combat pops the stack
//     and returns to Conversation. Combat also dismisses the Journal overlay
//     (GDD: "Combat preempts Journal").
//   - Journal is an overlay flag, not a base state: opening the journal leaves
//     the base state untouched, so closing it returns to the base state active
//     before it opened — never unconditionally Exploration (AC-20).
//   - Transition is exclusive: requests arriving during a transition are
//     queued (single slot, last request wins) and applied on EndTransition
//     after the pre-transition base state is restored (GDD CR-6: combat/
//     conversation events are queued and processed after fade-in).
type State struct {
	base BaseState

	// manual is the Manual overlay flag: camera at a player-chosen position,
	// follow suspended, no auto-return (GDD CR-4).
	manual bool
	// journal is the Journal overlay flag: camera holds position while the
	// investigation journal is open.
	journal bool

	// stack is the preemption stack, depth at most one. It holds the base
	// state preempted by Combat when Combat interrupted an active
	// Conversation. Empty for every other transition.
	stack []BaseState

	// pending is a base state request queued while TransitionBase is
	// exclusive. At most one request is held; the last request wins. nil when
	// nothing is queued.
	pending *BaseState

	// preTransition is the base state active before BeginTransition; the
	// transition restores it on EndTransition (GDD CR-6: "camera re-enters
	// the pre-transition base state (not unconditionally Exploration)").
	preTransition BaseState
}

// NewState returns a State machine starting in the Exploration base state
// with no overlays — the game-start configuration (GDD: Exploration entry at
// game start).
func NewState() State {
	return State{base: ExplorationBase}
}

// Base returns the currently active base state.
func (s *State) Base() BaseState { return s.base }

// Manual reports whether the Manual overlay flag is set.
func (s *State) Manual() bool { return s.manual }

// Journal reports whether the Journal overlay flag is set.
func (s *State) Journal() bool { return s.journal }

// SetManual sets or clears the Manual overlay flag. Setting Manual suspends
// base-state follow; clearing it resumes the base state's framing. Manual is
// cleared by the controller on Home/End/select and on base-state entries that
// frame (conversation start, combat start); the state machine itself never
// auto-clears it — there is no auto-return from a manual pan (GDD CR-4).
func (s *State) SetManual(manual bool) { s.manual = manual }

// OpenJournal sets the Journal overlay flag. The base state is left untouched,
// so closing the journal returns to the base state active before it opened
// (AC-20). During a transition the journal may still open — menu navigation
// stays functional during fades (GDD CR-6).
func (s *State) OpenJournal() { s.journal = true }

// CloseJournal clears the Journal overlay flag. The base state is unchanged;
// the controller resumes the base state's framing (journal close returns to
// the prior base state, never unconditionally Exploration — AC-20).
func (s *State) CloseJournal() { s.journal = false }

// RequestConversation requests the Conversation base state. Ignored while
// Combat is active (Combat outranks Conversation) and while a transition is
// in flight (queued instead). Entering Conversation clears any stale
// preemption stack — a fresh conversation never returns to a preempted state.
func (s *State) RequestConversation() {
	switch s.base {
	case TransitionBase:
		s.pending = statePtr(ConversationBase)
		return
	case CombatBase:
		return // priority: Combat > Conversation
	case ConversationBase:
		return // already there
	}
	s.base = ConversationBase
	s.stack = nil
}

// EndConversation leaves the Conversation base state. Dialogue ends always
// return to Exploration (GDD: "Dialogue ends → Exploration"). The controller
// clears the Manual overlay on this transition so the player is never stranded
// in Manual after dialogue (AC-22).
func (s *State) EndConversation() {
	if s.base == ConversationBase {
		s.base = ExplorationBase
		s.stack = nil
	}
}

// RequestCombat requests the Combat base state — the highest-priority
// gameplay state. If Conversation is active, it is pushed onto the one-deep
// preemption stack (Combat preempts Conversation); ending combat pops back to
// it. The Journal overlay is dismissed (GDD: "Combat preempts Journal"). A
// request while TransitionBase is active is queued and applied after the
// fade-in.
func (s *State) RequestCombat() {
	switch s.base {
	case TransitionBase:
		s.pending = statePtr(CombatBase)
		return
	case CombatBase:
		return // already there
	case ConversationBase:
		s.stack = append(s.stack, s.base)
	}
	s.base = CombatBase
	s.journal = false // combat dismisses the journal
}

// EndCombat leaves the Combat base state. If Combat preempted an active
// Conversation (one-deep stack), the camera returns to Conversation; otherwise
// it returns to Exploration (GDD CR-7 rule 4).
func (s *State) EndCombat() {
	if s.base != CombatBase {
		return
	}
	if n := len(s.stack); n > 0 {
		s.base = s.stack[n-1]
		s.stack = s.stack[:n-1]
		return
	}
	s.base = ExplorationBase
}

// BeginTransition enters the exclusive Transition base state, remembering the
// active base state so the fade-in can restore it (GDD CR-6). Overlay flags
// are preserved — follow is suspended during the fade anyway, and the player's
// pre-transition context survives. A call while already transitioning is a
// no-op.
func (s *State) BeginTransition() {
	if s.base == TransitionBase {
		return
	}
	s.preTransition = s.base
	s.base = TransitionBase
}

// EndTransition leaves the Transition base state: the pre-transition base
// state is restored, then any request queued during the fade is applied (last
// request wins — GDD CR-6: combat/conversation events queued during the fade
// are processed after fade-in). The controller inspects Base() afterwards to
// apply the corresponding framing lifecycle.
func (s *State) EndTransition() {
	if s.base != TransitionBase {
		return
	}
	s.base = s.preTransition
	if s.pending != nil {
		req := *s.pending
		s.pending = nil
		switch req {
		case CombatBase:
			// Re-apply the preemption logic against the restored base state
			// (which may itself be a Conversation to preempt).
			s.RequestCombat()
		case ConversationBase:
			s.RequestConversation()
		case ExplorationBase:
			s.base = ExplorationBase
			s.stack = nil
		}
	}
}

// statePtr boxes a BaseState for the single-slot pending request. Allocates
// once per queued request (transitions are rare, event-driven).
func statePtr(b BaseState) *BaseState { return &b }

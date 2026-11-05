// Package trust implements the Trust & Relationship core state machine
// (Story TRUST-001: production/epics/trust-relationship/
// story-001-trust-levels-events.md): per-NPC trust levels 0-3 (GDD CR-1,
// TR-trust-001), trust event gain/loss rules (GDD CR-2/CR-3, TR-trust-002),
// sequential level progression, betrayal freezes, and TrustChanged emission
// through the synchronous EventBus (ADR-0011).
//
// Trust levels gate Knowledge Graph fact access (ADR-0005): level N unlocks
// facts with trust_required <= N. The package is pure Go with no Kaiju
// engine dependency — the clock is an injected interface (GameClock) so the
// freeze logic is fully unit-testable (tests/unit/trust_relationship).
//
// Architecture: ADR-0005 (primary), ADR-0011 (secondary). Story 002
// (relationship metadata, vouching, reputation bleed) builds on this
// package; AI Dialogue integration and UI rendering are out of scope here.
package trust

// TrustLevel is the 0-3 trust scale per NPC (GDD CR-1, TR-trust-001).
// Trust gates Knowledge Graph fact access (ADR-0005): level N unlocks
// facts with trust_required <= N.
//
//	0 Stranger     — surface facts only (trust_required: 0)
//	1 Acquaintance — basic trust (trust_required: 0-1)
//	2 Trusted      — significant knowledge (trust_required: 0-2)
//	3 Confidant    — all facts including SECRETS
type TrustLevel int

const (
	// TrustStranger is level 0: surface facts only. The CR-5 default for
	// most NPCs.
	TrustStranger TrustLevel = iota
	// TrustAcquaintance is level 1: basic trust.
	TrustAcquaintance
	// TrustTrusted is level 2: significant knowledge; the NPC greets the
	// player by name.
	TrustTrusted
	// TrustConfidant is level 3: all facts including secrets.
	TrustConfidant
)

// String returns the trust level name for logging and test output.
func (l TrustLevel) String() string {
	switch l {
	case TrustStranger:
		return "Stranger"
	case TrustAcquaintance:
		return "Acquaintance"
	case TrustTrusted:
		return "Trusted"
	case TrustConfidant:
		return "Confidant"
	default:
		return "Unknown"
	}
}

// TrustEventType identifies a trust-relevant happening (GDD CR-2/CR-3,
// TR-trust-002). The 11 event types are the vocabulary the trust system
// records and evaluates; each is mapped to a gain or loss rule in Config.
type TrustEventType string

// Gain events (CR-2) — trust increases are sequential: an event can only
// raise trust up to its max unlocked level, never past it and never by
// more than one level at a time.
const (
	// EventDemonstratedInvestigation: the player asked about a topic in
	// the NPC's domain using knowledge gained from another NPC (proves
	// investigation, not random asking). +1 to level 1.
	EventDemonstratedInvestigation TrustEventType = "demonstrated_investigation"
	// EventSmallFavorCompleted: the player completed a minor task for the
	// NPC (fetch item, deliver message, share relevant information).
	// +1 to level 1.
	EventSmallFavorCompleted TrustEventType = "small_favor_completed"
	// EventThreadResolvedInvolvingNPC: the player resolved an
	// investigation thread that directly involves or affects this NPC.
	// +1, one-time per thread (CR-2).
	EventThreadResolvedInvolvingNPC TrustEventType = "thread_resolved_involving_npc"
	// EventSignificantHelp: the player helped the NPC in a meaningful way
	// (saved them from danger, resolved a major problem, protected someone
	// they care about). +1 to level 2.
	EventSignificantHelp TrustEventType = "significant_help"
	// EventVouchedFor: a character the NPC already trusts at level 2+
	// vouched for the player; the vouching character's trust must be
	// genuine. +1 to level 2.
	EventVouchedFor TrustEventType = "vouched_for"
	// EventSecretDiscoveredAndKept: the player discovered the NPC's secret
	// and did NOT share it with anyone the NPC wouldn't want to know.
	// +1 to level 3.
	EventSecretDiscoveredAndKept TrustEventType = "secret_discovered_and_kept"
)

// Loss events (CR-3) — losses are sharper than gains; secret_betrayed and
// hostile_action additionally trigger the trust freeze (GDD: 72 game
// hours). Losses always apply, even during a freeze.
const (
	// EventSecretBetrayed: the player shared the NPC's secret with someone
	// the NPC explicitly wouldn't trust with it. -2 (min 0), 72h freeze.
	EventSecretBetrayed TrustEventType = "secret_betrayed"
	// EventCaughtLying: the NPC catches the player in a direct lie
	// (contradicts a fact the NPC knows). -1.
	EventCaughtLying TrustEventType = "caught_lying"
	// EventHostileAction: the player attacks, steals from, or deliberately
	// harms the NPC or someone they care about. -2 (min 0), 72h freeze.
	EventHostileAction TrustEventType = "hostile_action"
	// EventBrokenPromise: the player promised to help and didn't (timeout
	// on a favor). -1.
	EventBrokenPromise TrustEventType = "broken_promise"
	// EventAssociationPenalty: the player is seen working closely with an
	// NPC's known enemy (faction-based). -1.
	EventAssociationPenalty TrustEventType = "association_penalty"
)

// TrustEvent is one recorded trust-relevant happening (GDD CR-4
// trust_events[] entry: {event_type, timestamp, thread_ref}).
type TrustEvent struct {
	// Type is which trust event occurred.
	Type TrustEventType
	// Timestamp is the game time in minutes when the event occurred — the
	// codebase's canonical game-time unit (entity.Event.Timestamp
	// convention). Producers set it to the clock time; a zero value is
	// stamped with the current clock time at processing.
	Timestamp float64
	// ThreadRef optionally references the investigation thread that
	// triggered this event (0 = none). Used by the one-time-per-thread
	// rule of thread_resolved_involving_npc.
	ThreadRef uint64
}

// TrustState is the per-NPC trust state (GDD CR-1, TR-trust-001). It is
// the unit the trust system mutates and GetTrust returns; persistence
// (Persistent component, story 002) serializes it with the NPC entity.
type TrustState struct {
	// Level is the current trust level, 0-3.
	Level TrustLevel
	// FreezeUntil is the game time in minutes at which an active trust
	// freeze expires; zero means no freeze. The time axis is monotonic
	// game minutes (see GameClock) — a 72h freeze spans three game days,
	// so this must not be a wrapping daily clock value.
	FreezeUntil float64
	// Events is the full history of trust events for this NPC in
	// processing order (GDD CR-4 trust_events[]). Persistence decides how
	// much to keep (story note: last 5 for MVP flags); memory keeps all.
	Events []TrustEvent
}

// frozen reports whether a trust freeze is active at the given game time:
// gains are blocked until FreezeUntil passes, and a zero FreezeUntil means
// no freeze ever started. At exactly FreezeUntil the freeze has expired
// (now < FreezeUntil is false).
func (s *TrustState) frozen(now float64) bool {
	return s.FreezeUntil > 0 && now < s.FreezeUntil
}

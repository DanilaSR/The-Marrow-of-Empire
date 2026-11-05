package trust

import "marrow-of-empire/entity"

// Outbound event (ADR-0011: "trust.changed"). The trust system is the sole
// producer; downstream consumers — AI Dialogue Engine (trust gates fact
// inclusion, ADR-0005), NPC Knowledge Graph, UI — subscribe without
// coupling. Emission is synchronous during HandleEvent, so subscribers read
// the NEW trust level (story 001: trust changes are recorded BEFORE
// downstream systems read them).
const (
	// EventTrustChanged fires whenever an applied trust event changes an
	// NPC's trust level. Payload: TrustChanged.
	EventTrustChanged entity.EventType = "trust.changed"
)

// TrustChanged is the payload of EventTrustChanged. It is a value type so
// the event avoids heap allocation beyond the Event.Data interface boxing
// (ADR-0011).
type TrustChanged struct {
	// NPCID is the NPC whose trust changed.
	NPCID entity.GameEntityID
	// OldLevel is the trust level before the change.
	OldLevel TrustLevel
	// NewLevel is the trust level after the change.
	NewLevel TrustLevel
	// EventType is the trust event that caused the change.
	EventType TrustEventType
}

// Inbound gameplay event types the trust system subscribes to at priority 1
// (story 001 control-manifest rule: trust subscribes at priority 1 so trust
// changes are recorded before downstream systems read the new level;
// ADR-0011 synchronous dispatch with priority ordering).
//
// These constants are the subscription contract with the producer systems:
//
//   - EventGameplayThreadResolved — Thread Investigation (thread resolved)
//   - EventGameplayFavorCompleted — favor/quest code (favor completed)
//   - EventGameplaySecretBetrayed — dialogue/player-action code (secret
//     betrayed)
//   - EventGameplayInteraction — generic carrier for any other
//     trust-relevant interaction (demonstrated_investigation,
//     significant_help, vouched_for, secret_discovered_and_kept,
//     caught_lying, hostile_action, broken_promise, association_penalty)
//
// Producers are not implemented yet; each must emit its event with a
// TrustEventData payload describing the interaction. The event type selects
// the dispatch ordering context; the payload drives the trust rules.
const (
	// EventGameplayThreadResolved is emitted when an investigation thread
	// involving the NPC resolves.
	EventGameplayThreadResolved entity.EventType = "thread.resolved"
	// EventGameplayFavorCompleted is emitted when the player completes a
	// favor for the NPC.
	EventGameplayFavorCompleted entity.EventType = "favor.completed"
	// EventGameplaySecretBetrayed is emitted when the player shares the
	// NPC's secret with someone the NPC wouldn't trust.
	EventGameplaySecretBetrayed entity.EventType = "secret.betrayed"
	// EventGameplayInteraction is the generic carrier for any other
	// trust-relevant interaction.
	EventGameplayInteraction entity.EventType = "trust.interaction"
)

// TrustEventData is the payload of the inbound gameplay events above. One
// payload = one interaction with one NPC. A single interaction may carry
// multiple trust events, of which only the highest-value one applies (GDD
// edge case: trust events don't stack in the same interaction) while all
// are recorded in the NPC's history.
type TrustEventData struct {
	// NPCID is the NPC the interaction concerned.
	NPCID entity.GameEntityID
	// Events lists the trust-relevant happenings of the interaction, in
	// producer order.
	Events []TrustEvent
}

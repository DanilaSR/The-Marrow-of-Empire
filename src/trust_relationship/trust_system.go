package trust

import (
	"log/slog"

	"marrow-of-empire/entity"
)

// GameClock supplies the current game time in minutes for freeze expiry
// (story 001: freeze uses game clock time — injectable interface so tests
// do not depend on the real clock).
//
// The time axis must be monotonic (never decrease) and MUST NOT wrap
// daily: a 72-hour freeze spans three game days, so expiry comparisons
// need absolute game minutes since game start. env.GameClock.TimeMinutes
// wraps into [0, 1440) per game day — a production adapter is required
// (integration work, out of scope); the mock clock in tests is absolute.
type GameClock interface {
	// TimeMinutes returns the current game time in minutes since game
	// start.
	TimeMinutes() float64
}

// Config carries the story 001 tuning knobs (GDD CR-2/CR-3/CR-5 and the
// tuning table: max_trust_level, trust_freeze_duration_hours). Zero-valued
// and nil fields fall back to the GDD defaults in DefaultConfig (pattern:
// combat.Config — gameplay values stay data-driven; a later story wires
// configs to external config files).
type Config struct {
	// MaxTrustLevel is the global trust ceiling (GDD max_trust_level,
	// default 3). Gains never raise trust past it.
	MaxTrustLevel TrustLevel
	// FreezeDurationMinutes is the trust freeze length (GDD
	// trust_freeze_duration_hours, default 72 game hours = 4320 minutes).
	FreezeDurationMinutes float64
	// InitialLevels overrides the CR-5 default start (level 0) per NPC:
	// story-critical NPCs and party members start at level 1. Missing
	// NPCs default to 0. Levels are clamped to MaxLevels/MaxTrustLevel.
	InitialLevels map[entity.GameEntityID]TrustLevel
	// MaxLevels caps trust per NPC (CR-5: antagonists cap at level 2,
	// faction enemies at level 1 without a faction shift). Missing NPCs
	// cap at MaxTrustLevel.
	MaxLevels map[entity.GameEntityID]TrustLevel
	// GainEvents maps each gain event to the highest level it can unlock
	// (CR-2 table). Sequential rule: the event raises trust by at most
	// one level and never past its max unlocked level. Defaults:
	// demonstrated_investigation 1, small_favor_completed 1,
	// thread_resolved_involving_npc 3 ("any", one-time per thread),
	// significant_help 2, vouched_for 2, secret_discovered_and_kept 3.
	GainEvents map[TrustEventType]TrustLevel
	// LossEvents maps each loss event to its rule (CR-3 table). Defaults:
	// secret_betrayed -2 + freeze, hostile_action -2 + freeze,
	// caught_lying -1, broken_promise -1, association_penalty -1.
	LossEvents map[TrustEventType]LossRule
}

// LossRule is the effect of one loss event (CR-3).
type LossRule struct {
	// Loss is the number of levels lost (result clamped at 0).
	Loss TrustLevel
	// Freeze reports whether the event activates the trust freeze for
	// FreezeDurationMinutes.
	Freeze bool
}

// GDD default tuning values (CR-2, CR-3, tuning table).
const (
	// DefaultMaxTrustLevel is 3: Confidant (GDD max_trust_level default).
	DefaultMaxTrustLevel TrustLevel = TrustConfidant
	// DefaultFreezeDurationMinutes is 72 game hours = 3 in-game days
	// (GDD trust_freeze_duration_hours default 72; ~7 real hours).
	DefaultFreezeDurationMinutes float64 = 72 * 60
)

// DefaultConfig returns the GDD-default tuning (CR-2 gain table, CR-3 loss
// table, CR-5 default start). The returned maps are shared per call —
// callers must not mutate them (or copy first).
func DefaultConfig() Config {
	return Config{
		MaxTrustLevel:         DefaultMaxTrustLevel,
		FreezeDurationMinutes: DefaultFreezeDurationMinutes,
		GainEvents: map[TrustEventType]TrustLevel{
			EventDemonstratedInvestigation:  TrustAcquaintance,
			EventSmallFavorCompleted:        TrustAcquaintance,
			EventThreadResolvedInvolvingNPC: TrustConfidant, // "any" — one-time per thread
			EventSignificantHelp:            TrustTrusted,
			EventVouchedFor:                 TrustTrusted,
			EventSecretDiscoveredAndKept:    TrustConfidant,
		},
		LossEvents: map[TrustEventType]LossRule{
			EventSecretBetrayed:     {Loss: 2, Freeze: true},
			EventHostileAction:      {Loss: 2, Freeze: true},
			EventCaughtLying:        {Loss: 1},
			EventBrokenPromise:      {Loss: 1},
			EventAssociationPenalty: {Loss: 1},
		},
	}
}

// trustBusPriority is the trust system's subscription priority (story 001
// control-manifest rule: trust subscribes at priority 1, the highest, so
// trust changes are recorded BEFORE downstream systems — AI Dialogue, NPC
// Knowledge Graph — read the new trust level).
const trustBusPriority = 1

// inboundGameplayEvents lists the gameplay event types the trust system
// reacts to (subscription contract; see events.go).
var inboundGameplayEvents = []entity.EventType{
	EventGameplayThreadResolved,
	EventGameplayFavorCompleted,
	EventGameplaySecretBetrayed,
	EventGameplayInteraction,
}

// TrustSystem is the per-NPC trust state machine (story 001).
//
// It subscribes to gameplay events (thread resolved, favor completed,
// secret betrayed, generic interaction) at priority 1, applies the CR-2/CR-3
// gain and loss rules, and emits EventTrustChanged for downstream consumers
// (ADR-0011, ADR-0005). All trust mutations go through this system — it
// never mutates other systems' state directly (control manifest: no direct
// cross-system calls for state mutations).
type TrustSystem struct {
	states map[entity.GameEntityID]*TrustState
	bus    entity.EventBus
	clock  GameClock
	cfg    Config
}

// NewTrustSystem builds a TrustSystem with the given bus, clock, and
// configuration, and subscribes it to the inbound gameplay event types at
// priority 1 (thread.resolved, favor.completed, secret.betrayed,
// trust.interaction).
//
// bus may be nil (emission is skipped — useful for tests that only assert
// return values); clock must be non-nil. Nil map fields and zero-valued
// scalars in cfg fall back to the DefaultConfig values (combat.Config
// pattern).
func NewTrustSystem(bus entity.EventBus, clock GameClock, cfg Config) *TrustSystem {
	ts := &TrustSystem{
		states: make(map[entity.GameEntityID]*TrustState),
		bus:    bus,
		clock:  clock,
		cfg:    cfg,
	}
	// Zero-value fallbacks (pattern: combat.Config).
	if ts.cfg.MaxTrustLevel == 0 {
		ts.cfg.MaxTrustLevel = DefaultMaxTrustLevel
	}
	if ts.cfg.FreezeDurationMinutes == 0 {
		ts.cfg.FreezeDurationMinutes = DefaultFreezeDurationMinutes
	}
	if ts.cfg.GainEvents == nil {
		ts.cfg.GainEvents = DefaultConfig().GainEvents
	}
	if ts.cfg.LossEvents == nil {
		ts.cfg.LossEvents = DefaultConfig().LossEvents
	}
	if bus != nil {
		for _, eventType := range inboundGameplayEvents {
			bus.Subscribe(eventType, trustBusPriority, ts.handleGameplayEvent)
		}
	}
	return ts
}

// handleGameplayEvent is the priority-1 EventBus handler (ADR-0011). The
// inbound event type selects the dispatch ordering; the payload carries the
// trust events, so all inbound types share this handler.
func (ts *TrustSystem) handleGameplayEvent(ev entity.Event) {
	ts.HandleEvent(ev)
}

// HandleEvent processes one inbound gameplay event: it extracts the
// interaction payload, applies the trust rules (highest-value event only,
// sequential gains, freeze, clamps), records the history, and emits
// EventTrustChanged for every applied level change. Returns the emitted
// changes for direct consumers and tests.
//
// Events carrying a payload other than TrustEventData are ignored with a
// warning (defensive — the bus is shared and payloads are `any`, ADR-0011).
func (ts *TrustSystem) HandleEvent(ev entity.Event) []TrustChanged {
	data, ok := ev.Data.(TrustEventData)
	if !ok {
		slog.Warn("trust: ignoring event with unexpected payload", "type", ev.Type, "source", ev.Source)
		return nil
	}
	return ts.applyInteraction(data)
}

// applyInteraction applies one interaction (a batch of trust events for one
// NPC): it selects and applies the highest-value event (AC-3 — events don't
// stack in the same interaction) and records every event of the batch in
// the NPC's history (full fidelity for dialogue context, GDD CR-4).
func (ts *TrustSystem) applyInteraction(data TrustEventData) []TrustChanged {
	if len(data.Events) == 0 {
		return nil
	}
	state := ts.state(data.NPCID)
	now := ts.clock.TimeMinutes()

	// Stamp missing timestamps with the current game time (codebase
	// convention: float64 game minutes). Copy so caller-owned payload data
	// is never mutated.
	recorded := make([]TrustEvent, len(data.Events))
	copy(recorded, data.Events)
	for i := range recorded {
		if recorded[i].Timestamp == 0 {
			recorded[i].Timestamp = now
		}
	}

	// Apply only the highest-value event of the interaction...
	best := ts.highestValue(state, now, recorded)
	changes := ts.applyOne(state, data.NPCID, best, now)

	// ...but record every event, applied or not.
	state.Events = append(state.Events, recorded...)
	return changes
}

// highestValue selects the single highest-value event of an interaction
// (AC-3: only the highest-value event applies). Gain events score their max
// unlocked level, loss events their loss amount; ties resolve to the first
// event in the batch (deterministic). While a freeze is active, gain events
// are ineligible (score 0) — losses still apply during a freeze (CR-3), so
// a loss in a mixed batch wins by default. Events with no gain/loss rule
// score 0 and never win.
func (ts *TrustSystem) highestValue(state *TrustState, now float64, events []TrustEvent) TrustEvent {
	best := events[0]
	bestVal := ts.value(state, now, best)
	for _, e := range events[1:] {
		if v := ts.value(state, now, e); v > bestVal {
			best, bestVal = e, v
		}
	}
	return best
}

// value returns the event's ranking value: its max unlocked level for gain
// events (0 while frozen — gains are blocked), its loss amount for loss
// events, 0 for events with no rule.
func (ts *TrustSystem) value(state *TrustState, now float64, e TrustEvent) TrustLevel {
	if maxLevel, ok := ts.cfg.GainEvents[e.Type]; ok {
		if state.frozen(now) {
			return 0
		}
		return maxLevel
	}
	if rule, ok := ts.cfg.LossEvents[e.Type]; ok {
		return rule.Loss
	}
	return 0
}

// applyOne applies a single trust event to the NPC state and emits
// TrustChanged if the level changed. The event has already been recorded by
// the caller (applyInteraction), which is also what makes the
// one-time-per-thread check below correct: the current event is not yet in
// the history.
func (ts *TrustSystem) applyOne(state *TrustState, npcID entity.GameEntityID, ev TrustEvent, now float64) []TrustChanged {
	old := state.Level
	if maxLevel, ok := ts.cfg.GainEvents[ev.Type]; ok {
		// Gain rules (CR-2): sequential progression — an event raises
		// trust by at most one level and never past its max unlocked
		// level; the freeze blocks gains (recorded, no change); thread
		// rewards are one-time per thread.
		if state.frozen(now) {
			return nil
		}
		if ev.Type == EventThreadResolvedInvolvingNPC && alreadyRewarded(state, ev) {
			return nil
		}
		candidate := old + 1
		if candidate > maxLevel {
			candidate = maxLevel
		}
		if cap := ts.cfg.MaxLevels[npcID]; cap > 0 && candidate > cap {
			candidate = cap
		}
		if candidate > ts.cfg.MaxTrustLevel {
			candidate = ts.cfg.MaxTrustLevel
		}
		if candidate <= old {
			// Sequential/cap no-op (e.g. significant_help at level 2, or
			// an event whose max level is below the current level):
			// recorded, no level change. Never downgrades trust.
			return nil
		}
		state.Level = candidate
	} else if rule, ok := ts.cfg.LossEvents[ev.Type]; ok {
		// Loss rules (CR-3): losses always apply — even during a freeze —
		// and clamp at 0 (no level below Stranger). Freeze-triggering
		// losses (secret_betrayed, hostile_action) set the freeze from
		// now; a repeat betrayal during an active freeze re-arms
		// (extends) it.
		if state.Level > rule.Loss {
			state.Level -= rule.Loss
		} else {
			state.Level = TrustStranger
		}
		if rule.Freeze {
			state.FreezeUntil = now + ts.cfg.FreezeDurationMinutes
		}
	}
	if state.Level == old {
		return nil // recorded, but no level change → no TrustChanged
	}
	changed := TrustChanged{
		NPCID:     npcID,
		OldLevel:  old,
		NewLevel:  state.Level,
		EventType: ev.Type,
	}
	ts.emit(changed)
	return []TrustChanged{changed}
}

// alreadyRewarded reports whether the one-time-per-thread reward for the
// event's thread was already granted (CR-2: thread_resolved_involving_npc is
// +1 one-time per thread). Events with ThreadRef 0 never count as rewarded
// (no thread to key on).
func alreadyRewarded(state *TrustState, ev TrustEvent) bool {
	if ev.ThreadRef == 0 {
		return false
	}
	for _, prev := range state.Events {
		if prev.Type == EventThreadResolvedInvolvingNPC && prev.ThreadRef == ev.ThreadRef {
			return true
		}
	}
	return false
}

// emit publishes a TrustChanged event on the bus (ADR-0011 synchronous
// dispatch; subscribers see the new level immediately). No-op when the bus
// is nil.
func (ts *TrustSystem) emit(changed TrustChanged) {
	if ts.bus == nil {
		return
	}
	ts.bus.Emit(entity.Event{
		Type:      EventTrustChanged,
		Timestamp: ts.clock.TimeMinutes(),
		Source:    "trust",
		Data:      changed,
	})
}

// state returns the NPC's trust state, creating it lazily with the
// configured initial level (CR-5: default 0) on first touch.
func (ts *TrustSystem) state(npcID entity.GameEntityID) *TrustState {
	s, ok := ts.states[npcID]
	if !ok {
		s = &TrustState{Level: ts.initialLevel(npcID)}
		ts.states[npcID] = s
	}
	return s
}

// GetTrust returns the NPC's current trust state. NPCs that have never
// interacted report their configured initial level (CR-5 default 0) with an
// empty history. The returned value is a copy — mutating it never affects
// the system's state.
func (ts *TrustSystem) GetTrust(npcID entity.GameEntityID) TrustState {
	if s, ok := ts.states[npcID]; ok {
		snap := *s
		snap.Events = append([]TrustEvent(nil), s.Events...)
		return snap
	}
	return TrustState{Level: ts.initialLevel(npcID)}
}

// SetInitialTrust seeds an NPC's starting trust level (CR-5 overrides:
// story-critical NPCs and party members start at level 1). It is a no-op
// once the NPC already has state — events own the level from then on. The
// level is clamped to the NPC's cap and MaxTrustLevel.
func (ts *TrustSystem) SetInitialTrust(npcID entity.GameEntityID, level TrustLevel) {
	if _, ok := ts.states[npcID]; ok {
		return
	}
	ts.states[npcID] = &TrustState{Level: ts.clampLevel(npcID, level)}
}

// initialLevel returns the configured starting level for an NPC (CR-5:
// default 0; story-critical NPCs and party members configured at 1),
// clamped to the NPC's cap and MaxTrustLevel.
func (ts *TrustSystem) initialLevel(npcID entity.GameEntityID) TrustLevel {
	// Lookup on a nil map returns 0 (TrustStranger) — the CR-5 default.
	return ts.clampLevel(npcID, ts.cfg.InitialLevels[npcID])
}

// clampLevel clamps a trust level to the NPC's per-NPC cap (CR-5:
// antagonists 2, faction enemies 1) and the global MaxTrustLevel.
func (ts *TrustSystem) clampLevel(npcID entity.GameEntityID, level TrustLevel) TrustLevel {
	if cap := ts.cfg.MaxLevels[npcID]; cap > 0 && level > cap {
		level = cap
	}
	if level > ts.cfg.MaxTrustLevel {
		level = ts.cfg.MaxTrustLevel
	}
	return level
}

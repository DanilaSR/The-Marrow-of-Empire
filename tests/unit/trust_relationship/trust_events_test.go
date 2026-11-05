// Unit tests for Story TRUST-001: Trust Levels & Trust Events.
//
// Evidence path: production/epics/trust-relationship/
// story-001-trust-levels-events.md
//
// Tests map to the acceptance criteria (AC-1 demonstrated_investigation
// 0->1, AC-2 secret_betrayed 2->0 + 72h freeze, AC-3 highest-value event
// only) and the edge cases (clamp at 3, clamp at 0), plus sequential level
// enforcement, freeze blocking gains, losses applying during freeze, the
// one-time-per-thread rule, CR-5 initial levels, per-NPC caps, and the
// EventBus integration (ADR-0011 priority subscription + trust.changed
// emission). Requirements TR-trust-001 and TR-trust-002; trust gates
// Knowledge Graph fact access (ADR-0005).
package trust_test

import (
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/trust_relationship"
)

// mockClock is an absolute game-time clock for tests (monotonic minutes;
// the trust.GameClock contract — no daily wrap, unlike env.GameClock).
type mockClock struct {
	minutes float64
}

func (c *mockClock) TimeMinutes() float64 { return c.minutes }

// newTestSystem returns a TrustSystem with a real EventBus (entity package,
// pure Go), a mock clock at 1000 game minutes, and GDD-default tuning.
func newTestSystem() (*trust.TrustSystem, *mockClock, *entity.EventBusImpl) {
	bus := entity.NewEventBus()
	clock := &mockClock{minutes: 1000}
	return trust.NewTrustSystem(bus, clock, trust.DefaultConfig()), clock, bus
}

// interaction builds an inbound gameplay event carrying the given trust
// events for one NPC (one bus event = one interaction).
func interaction(npc entity.GameEntityID, events ...trust.TrustEvent) entity.Event {
	return entity.Event{
		Type:   trust.EventGameplayInteraction,
		Source: "test",
		Data:   trust.TrustEventData{NPCID: npc, Events: events},
	}
}

// AC-1: GIVEN an NPC at trust level 0, WHEN the player demonstrates
// investigation using knowledge from another NPC (demonstrated_investigation
// event), THEN trust increases to level 1.
func TestTrustSystem_AC1_DemonstratedInvestigation_Level0To1(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(1)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventDemonstratedInvestigation}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level = %v, want Acquaintance", got)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d events, want 1", len(changed))
	}
	if changed[0].NPCID != npc {
		t.Errorf("NPCID = %v, want %v", changed[0].NPCID, npc)
	}
	if changed[0].OldLevel != trust.TrustStranger || changed[0].NewLevel != trust.TrustAcquaintance {
		t.Errorf("change = %v -> %v, want Stranger -> Acquaintance", changed[0].OldLevel, changed[0].NewLevel)
	}
	if changed[0].EventType != trust.EventDemonstratedInvestigation {
		t.Errorf("EventType = %q, want demonstrated_investigation", changed[0].EventType)
	}
}

// AC-2: GIVEN an NPC at trust level 2, WHEN the player betrays the NPC's
// secret (secret_betrayed event), THEN trust drops to level 0 AND the trust
// freeze activates for 72 game hours.
func TestTrustSystem_AC2_SecretBetrayed_DropsTo0With72hFreeze(t *testing.T) {
	ts, clock, _ := newTestSystem()
	const npc = entity.GameEntityID(2)
	ts.SetInitialTrust(npc, trust.TrustTrusted)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretBetrayed}))

	state := ts.GetTrust(npc)
	if state.Level != trust.TrustStranger {
		t.Fatalf("Level = %v, want Stranger", state.Level)
	}
	if want := clock.minutes + 72*60; state.FreezeUntil != want {
		t.Errorf("FreezeUntil = %v, want %v (72 game hours from now)", state.FreezeUntil, want)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d events, want 1", len(changed))
	}
	if changed[0].OldLevel != trust.TrustTrusted || changed[0].NewLevel != trust.TrustStranger {
		t.Errorf("change = %v -> %v, want Trusted -> Stranger", changed[0].OldLevel, changed[0].NewLevel)
	}
	if changed[0].EventType != trust.EventSecretBetrayed {
		t.Errorf("EventType = %q, want secret_betrayed", changed[0].EventType)
	}
}

// AC-3: GIVEN multiple trust events in one interaction, WHEN events would
// push trust from 0 to 2 (demonstrated_investigation + significant_help in
// the same interaction), THEN only the highest-value event applies — trust
// reaches level 1, not level 2.
func TestTrustSystem_AC3_MultipleEvents_OnlyHighestValueApplies(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(3)

	changed := ts.HandleEvent(interaction(npc,
		trust.TrustEvent{Type: trust.EventDemonstratedInvestigation},
		trust.TrustEvent{Type: trust.EventSignificantHelp},
	))

	// significant_help (unlocks level 2) is the higher-value event, but the
	// sequential rule caps it at level 1 from level 0 — never level 2.
	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level = %v, want Acquaintance (not Trusted)", got)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d events, want 1 (events don't stack)", len(changed))
	}
	if changed[0].EventType != trust.EventSignificantHelp {
		t.Errorf("applied EventType = %q, want significant_help (highest value)", changed[0].EventType)
	}
	// Both events are recorded in the history (dialogue context, CR-4).
	if n := len(ts.GetTrust(npc).Events); n != 2 {
		t.Errorf("history = %d events, want 2 (both recorded)", n)
	}
}

// Edge: GIVEN an NPC at trust level 3, WHEN a trust-increasing event fires,
// THEN trust stays at 3 and the event is recorded but no level change
// occurs.
func TestTrustSystem_Edge_ClampAtConfidant(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(4)
	ts.SetInitialTrust(npc, trust.TrustConfidant)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretDiscoveredAndKept}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustConfidant {
		t.Fatalf("Level = %v, want Confidant", got)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 (no level change)", len(changed))
	}
	if n := len(ts.GetTrust(npc).Events); n != 1 {
		t.Errorf("history = %d events, want 1 (recorded)", n)
	}
}

// Edge: GIVEN an NPC at trust level 0, WHEN a trust-decreasing event fires,
// THEN trust stays at 0 (NPC cannot go below stranger status).
func TestTrustSystem_Edge_ClampAtStranger(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(5)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventCaughtLying}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Fatalf("Level = %v, want Stranger", got)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 (no level change)", len(changed))
	}
	if n := len(ts.GetTrust(npc).Events); n != 1 {
		t.Errorf("history = %d events, want 1 (recorded)", n)
	}
}

// Sequential enforcement: GIVEN an NPC at level 0, WHEN significant_help (a
// level-2 event) fires, THEN trust rises to level 1, not level 2 — levels
// cannot be skipped (CR-2).
func TestTrustSystem_Sequential_SignificantHelp_NoLevelSkipping(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(6)

	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSignificantHelp}))
	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level after first event = %v, want Acquaintance", got)
	}

	// At level 1 the same event reaches level 2.
	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSignificantHelp}))
	if got := ts.GetTrust(npc).Level; got != trust.TrustTrusted {
		t.Errorf("Level after second event = %v, want Trusted", got)
	}

	// At level 2 it can go no further: min(3, max 2) = 2 — no change, no
	// downgrade, no emission.
	if changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSignificantHelp})); len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 at event's max level", len(changed))
	}
	if got := ts.GetTrust(npc).Level; got != trust.TrustTrusted {
		t.Errorf("Level after third event = %v, want Trusted (capped by event)", got)
	}
}

// Freeze: GIVEN a frozen NPC, WHEN a trust-increasing event fires, THEN the
// event is recorded but the level does not change and no TrustChanged is
// emitted. Gains resume once the freeze expires (exactly at FreezeUntil).
func TestTrustSystem_Freeze_BlocksGains(t *testing.T) {
	ts, clock, _ := newTestSystem()
	const npc = entity.GameEntityID(7)
	ts.SetInitialTrust(npc, trust.TrustAcquaintance)
	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretBetrayed})) // 1 -> 0, freeze at t=1000

	// Gain during the freeze: recorded, no change.
	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventDemonstratedInvestigation}))
	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Fatalf("Level during freeze = %v, want Stranger", got)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 during freeze", len(changed))
	}
	if n := len(ts.GetTrust(npc).Events); n != 2 {
		t.Errorf("history = %d events, want 2 (gain recorded even when frozen)", n)
	}

	// Exactly at expiry the gain applies.
	clock.minutes = ts.GetTrust(npc).FreezeUntil
	changed = ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventDemonstratedInvestigation}))
	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level after freeze = %v, want Acquaintance", got)
	}
	if len(changed) != 1 {
		t.Errorf("changed = %d events, want 1 after freeze", len(changed))
	}
}

// Freeze: GIVEN a frozen NPC, WHEN a trust-decreasing event fires, THEN the
// loss still applies — the freeze blocks gains, not losses (CR-3).
func TestTrustSystem_Freeze_LossesStillApply(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(8)
	ts.SetInitialTrust(npc, trust.TrustConfidant)
	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretBetrayed})) // 3 -> 1, freeze

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventCaughtLying}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Fatalf("Level = %v, want Stranger (loss applied during freeze)", got)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d events, want 1", len(changed))
	}
	if changed[0].EventType != trust.EventCaughtLying {
		t.Errorf("EventType = %q, want caught_lying", changed[0].EventType)
	}
}

// Freeze: GIVEN a mixed batch (gain + loss) during an active freeze, WHEN
// the gain is the nominally higher-value event, THEN the loss still applies
// — gain events are ineligible while frozen (CR-3).
func TestTrustSystem_Freeze_MixedBatch_LossApplies(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(9)
	ts.SetInitialTrust(npc, trust.TrustConfidant)
	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretBetrayed})) // 3 -> 1, freeze

	changed := ts.HandleEvent(interaction(npc,
		trust.TrustEvent{Type: trust.EventSecretDiscoveredAndKept}, // gain value 3, ineligible while frozen
		trust.TrustEvent{Type: trust.EventCaughtLying},             // loss value 1, applies
	))

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Fatalf("Level = %v, want Stranger", got)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %d events, want 1", len(changed))
	}
	if changed[0].EventType != trust.EventCaughtLying {
		t.Errorf("applied EventType = %q, want caught_lying", changed[0].EventType)
	}
}

// Freeze: GIVEN a betrayal during an active freeze, WHEN a freeze-triggering
// loss fires again, THEN the loss applies and the freeze re-arms from the
// current time (extends).
func TestTrustSystem_Freeze_RepeatBetrayal_ExtendsFreeze(t *testing.T) {
	ts, clock, _ := newTestSystem()
	const npc = entity.GameEntityID(10)
	ts.SetInitialTrust(npc, trust.TrustTrusted)
	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretBetrayed})) // 2 -> 0, freeze until 5320

	clock.minutes = 2000
	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventHostileAction}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Errorf("Level = %v, want Stranger", got)
	}
	if want := clock.minutes + 72*60; ts.GetTrust(npc).FreezeUntil != want {
		t.Errorf("FreezeUntil = %v, want %v (re-armed from second betrayal)", ts.GetTrust(npc).FreezeUntil, want)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 (already at 0, clamped)", len(changed))
	}
}

// One-time per thread: GIVEN a resolved thread, WHEN the same thread
// resolves again, THEN only the first resolution grants trust; a different
// thread still does.
func TestTrustSystem_ThreadResolved_OneTimePerThread(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(11)

	first := interaction(npc, trust.TrustEvent{Type: trust.EventThreadResolvedInvolvingNPC, ThreadRef: 42})
	second := interaction(npc, trust.TrustEvent{Type: trust.EventThreadResolvedInvolvingNPC, ThreadRef: 42})
	other := interaction(npc, trust.TrustEvent{Type: trust.EventThreadResolvedInvolvingNPC, ThreadRef: 43})

	ts.HandleEvent(first) // 0 -> 1
	if changed := ts.HandleEvent(second); len(changed) != 0 {
		t.Error("second resolution of thread 42 must not grant trust again")
	}
	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level = %v, want Acquaintance", got)
	}
	if changed := ts.HandleEvent(other); len(changed) != 1 {
		t.Errorf("a different thread must still grant trust: changed = %d", len(changed))
	}
	if got := ts.GetTrust(npc).Level; got != trust.TrustTrusted {
		t.Errorf("Level = %v, want Trusted", got)
	}
}

// CR-5: GIVEN an NPC with a configured initial level, WHEN the system is
// created, THEN the NPC reports that level without any events processed.
func TestTrustSystem_InitialLevel_ConfigOverride(t *testing.T) {
	bus := entity.NewEventBus()
	clock := &mockClock{minutes: 1000}
	cfg := trust.DefaultConfig()
	const npc = entity.GameEntityID(12)
	cfg.InitialLevels = map[entity.GameEntityID]trust.TrustLevel{npc: trust.TrustAcquaintance}
	ts := trust.NewTrustSystem(bus, clock, cfg)

	if got := ts.GetTrust(npc).Level; got != trust.TrustAcquaintance {
		t.Fatalf("Level = %v, want Acquaintance (CR-5 story-critical start)", got)
	}
	if n := len(ts.GetTrust(npc).Events); n != 0 {
		t.Errorf("history = %d events, want 0", n)
	}
}

// CR-5 caps: GIVEN an NPC capped at level 2 (antagonist), WHEN a level-3
// gain event fires at level 2, THEN trust stays at 2 — confidant is
// unreachable.
func TestTrustSystem_Cap_AntagonistCannotReachConfidant(t *testing.T) {
	bus := entity.NewEventBus()
	clock := &mockClock{minutes: 1000}
	cfg := trust.DefaultConfig()
	const npc = entity.GameEntityID(13)
	cfg.MaxLevels = map[entity.GameEntityID]trust.TrustLevel{npc: trust.TrustTrusted}
	ts := trust.NewTrustSystem(bus, clock, cfg)
	ts.SetInitialTrust(npc, trust.TrustTrusted)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSecretDiscoveredAndKept}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustTrusted {
		t.Fatalf("Level = %v, want Trusted (antagonist cap at 2)", got)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0 (capped)", len(changed))
	}
}

// EventBus integration (ADR-0011): GIVEN a real EventBus, WHEN a gameplay
// event is emitted, THEN the priority-1 subscription reacts synchronously
// and emits trust.changed with the new level.
func TestTrustSystem_EventBus_EmissionAndPrioritySubscription(t *testing.T) {
	bus := entity.NewEventBus()
	clock := &mockClock{minutes: 1000}
	ts := trust.NewTrustSystem(bus, clock, trust.DefaultConfig())
	const npc = entity.GameEntityID(14)

	var got trust.TrustChanged
	bus.Subscribe(trust.EventTrustChanged, 10, func(ev entity.Event) {
		got = ev.Data.(trust.TrustChanged)
	})

	bus.Emit(entity.Event{
		Type:   trust.EventGameplayThreadResolved,
		Source: "thread",
		Data: trust.TrustEventData{
			NPCID:  npc,
			Events: []trust.TrustEvent{{Type: trust.EventThreadResolvedInvolvingNPC, ThreadRef: 7}},
		},
	})

	if got.NPCID != npc || got.OldLevel != trust.TrustStranger || got.NewLevel != trust.TrustAcquaintance {
		t.Errorf("trust.changed = %+v, want NPC %v Stranger -> Acquaintance", got, npc)
	}
	if ev := ts.GetTrust(npc).Events[0]; ev.ThreadRef != 7 {
		t.Errorf("recorded ThreadRef = %v, want 7", ev.ThreadRef)
	}
}

// Defensive cast (ADR-0011 payloads are `any`): GIVEN a gameplay event with
// a non-TrustEventData payload, WHEN it is emitted, THEN the trust system
// ignores it.
func TestTrustSystem_WrongPayload_Ignored(t *testing.T) {
	bus := entity.NewEventBus()
	clock := &mockClock{minutes: 1000}
	ts := trust.NewTrustSystem(bus, clock, trust.DefaultConfig())
	const npc = entity.GameEntityID(15)

	bus.Emit(entity.Event{
		Type:   trust.EventGameplayInteraction,
		Source: "other",
		Data:   entity.EntityEventData{EntityID: npc},
	})

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Errorf("Level = %v, want Stranger (payload ignored)", got)
	}
}

// GIVEN a trust event type with no gain/loss rule, WHEN it is processed,
// THEN it is recorded but causes no level change and no emission.
func TestTrustSystem_UnknownEventType_RecordedNoChange(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(16)

	changed := ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.TrustEventType("mystery_event")}))

	if got := ts.GetTrust(npc).Level; got != trust.TrustStranger {
		t.Errorf("Level = %v, want Stranger", got)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %d events, want 0", len(changed))
	}
	if n := len(ts.GetTrust(npc).Events); n != 1 {
		t.Errorf("history = %d events, want 1 (recorded)", n)
	}
}

// GIVEN a trust event without a timestamp, WHEN it is processed, THEN it is
// recorded with the current game time (codebase convention: float64
// minutes).
func TestTrustSystem_EventTimestamp_StampedWithClockTime(t *testing.T) {
	ts, clock, _ := newTestSystem()
	const npc = entity.GameEntityID(17)

	ts.HandleEvent(interaction(npc, trust.TrustEvent{Type: trust.EventSmallFavorCompleted}))

	if got := ts.GetTrust(npc).Events[0].Timestamp; got != clock.minutes {
		t.Errorf("Timestamp = %v, want %v (clock time)", got, clock.minutes)
	}
}

// GIVEN an interaction with no trust events, WHEN it is processed, THEN
// nothing happens — no state mutation, no history.
func TestTrustSystem_EmptyInteraction_NoOp(t *testing.T) {
	ts, _, _ := newTestSystem()
	const npc = entity.GameEntityID(18)

	if changed := ts.HandleEvent(interaction(npc)); len(changed) != 0 {
		t.Errorf("changed = %d events, want 0", len(changed))
	}
	if n := len(ts.GetTrust(npc).Events); n != 0 {
		t.Errorf("history = %d events, want 0", n)
	}
}

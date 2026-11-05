// Integration tests for Story 001: Combat HUD Framework & Turn Order.
//
// Evidence path: production/epics/combat-ui/story-001-combat-hud-framework-turn-order.md
// Required evidence: this file must exist and pass (integration gate).
//
// The HUD's event wiring and turn order model are pure game logic — the
// Kaiju document rendering is verified by the manual walkthrough
// (production/qa/evidence). These tests drive the HUD headlessly:
// NewCombatHUD with nil host and manager skips markup loading and the
// per-frame update, while the real synchronous EventBus (ADR-0011) and
// the real combat payload types exercise the subscription layer end to
// end — an event emitted on the bus reaches the model exactly as it would
// in a running game.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_combat_started_populates_turn_order    -> TestCombatStartedPopulatesTurnOrder
//	AC-1  test_combatant_removed_drops_from_order     -> TestCombatantRemovedDropsFromOrder
//	AC-3  test_highlight_follows_turn_started         -> TestHighlightFollowsTurnStarted
//	AC-3  test_turn_ended_marks_acted                 -> TestTurnEndedMarksActed
//	AC-3  test_round_wrap_resets_acted                -> TestRoundWrapResetsActed
//	AC-4  test_combat_ended_resets_state              -> TestCombatEndedResetsState
//	edge  test_restart_combat_clears_stale_state      -> TestRestartCombatClearsStaleState
//	edge  test_removed_current_keeps_turn_slot        -> TestRemovedCurrentKeepsTurnSlot
//	edge  test_wrong_payload_types_ignored            -> TestWrongPayloadTypesIgnored
//	edge  test_turn_events_ignored_outside_combat     -> TestTurnEventsIgnoredOutsideCombat
//	AC-5  test_resolve_combatant_name_fallback        -> TestResolveCombatantNameFallback
package combat_ui_test

import (
	"reflect"
	"testing"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/ui/combat_hud"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// ac1TurnOrder mirrors the QA setup for AC-1: 4 player characters with
// initiative 8, 5, 3, 1 and 3 enemies with initiative 6, 4, 2. The combat
// system (combat.rollInitiative) delivers this already sorted in
// descending order; the HUD displays the order exactly as provided.
func ac1TurnOrder() []combat.TurnEntry {
	return []combat.TurnEntry{
		{EntityID: 101, Side: combat.SidePlayer, Initiative: 8},
		{EntityID: 201, Side: combat.SideEnemy, Initiative: 6},
		{EntityID: 102, Side: combat.SidePlayer, Initiative: 5},
		{EntityID: 202, Side: combat.SideEnemy, Initiative: 4},
		{EntityID: 103, Side: combat.SidePlayer, Initiative: 3},
		{EntityID: 203, Side: combat.SideEnemy, Initiative: 2},
		{EntityID: 104, Side: combat.SidePlayer, Initiative: 1},
	}
}

// newHeadlessHUD constructs the HUD without a rendering context and
// returns it with its real EventBus. All assertions drive events through
// the bus — a passing test proves the subscription wiring, not just the
// model. The stub combat manager (ability_menu_test.go) supplies the
// Story 002 AP readouts and the End Turn action.
func newHeadlessHUD(t *testing.T) (*combat_hud.CombatHUD, *entity.EventBusImpl) {
	t.Helper()
	bus := entity.NewEventBus()
	mgr := newStubManager(bus, 8)
	hud := combat_hud.NewCombatHUD(nil, nil, bus, nil, nil, mgr)
	if hud == nil {
		t.Fatal("NewCombatHUD returned nil")
	}
	return hud, bus
}

func emit(t *testing.T, bus *entity.EventBusImpl, ev entity.Event) {
	t.Helper()
	bus.Emit(ev)
}

func emitCombatStarted(t *testing.T, bus *entity.EventBusImpl, order []combat.TurnEntry) {
	t.Helper()
	emit(t, bus, entity.Event{
		Type:   combat.EventCombatStarted,
		Source: "test",
		Data:   combat.CombatStartedData{TurnOrder: order},
	})
}

func emitTurnStarted(t *testing.T, bus *entity.EventBusImpl, id entity.GameEntityID, round int) {
	t.Helper()
	emit(t, bus, entity.Event{
		Type:   combat.EventTurnStarted,
		Source: "test",
		Data:   combat.TurnStartedData{Combatant: id, Side: combat.SidePlayer, Round: round},
	})
}

func emitTurnEnded(t *testing.T, bus *entity.EventBusImpl, id entity.GameEntityID, round int) {
	t.Helper()
	emit(t, bus, entity.Event{
		Type:   combat.EventTurnEnded,
		Source: "test",
		Data:   combat.TurnEndedData{Combatant: id, Round: round},
	})
}

func emitCombatantRemoved(t *testing.T, bus *entity.EventBusImpl, id entity.GameEntityID, reason combat.RemovalReason) {
	t.Helper()
	emit(t, bus, entity.Event{
		Type:   combat.EventCombatantRemoved,
		Source: "test",
		Data:   combat.CombatantRemovedData{EntityID: id, Reason: reason},
	})
}

func emitCombatEnded(t *testing.T, bus *entity.EventBusImpl, outcome combat.Outcome) {
	t.Helper()
	emit(t, bus, entity.Event{
		Type:   combat.EventCombatEnded,
		Source: "test",
		Data:   combat.CombatEndedData{Outcome: outcome},
	})
}

// stubNameReader supplies fixed display names for the name-resolution
// test.
type stubNameReader struct {
	names map[entity.GameEntityID]string
}

func (s stubNameReader) Name(id entity.GameEntityID) string { return s.names[id] }

// ---------------------------------------------------------------------------
// AC-1: turn order display
// ---------------------------------------------------------------------------

// test_combat_started_populates_turn_order: on EventCombatStarted the HUD
// activates and the turn order matches the payload exactly (7 portraits in
// descending initiative order: PC1(8), Enemy1(6), PC2(5), Enemy2(4),
// PC3(3), Enemy3(2), PC4(1)).
func TestCombatStartedPopulatesTurnOrder(t *testing.T) {
	hud, bus := newHeadlessHUD(t)

	if hud.Active() {
		t.Fatal("HUD must be inactive before combat starts")
	}
	order := ac1TurnOrder()
	emitCombatStarted(t, bus, order)

	if !hud.Active() {
		t.Fatal("HUD must be active after EventCombatStarted")
	}
	if got := hud.TurnOrder(); !reflect.DeepEqual(got, order) {
		t.Fatalf("turn order = %v, want %v", got, order)
	}
	if got := hud.Round(); got != 1 {
		t.Fatalf("round = %d, want 1", got)
	}
	if _, ok := hud.CurrentCombatant(); ok {
		t.Fatal("no combatant may be current until EventTurnStarted (combat contract)")
	}
}

// test_combatant_removed_drops_from_order: on EventCombatantRemoved the
// portrait leaves the row and the remaining portraits keep their order.
func TestCombatantRemovedDropsFromOrder(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	emitCombatantRemoved(t, bus, 202, combat.RemovalDefeated)

	want := []combat.TurnEntry{
		{EntityID: 101, Side: combat.SidePlayer, Initiative: 8},
		{EntityID: 201, Side: combat.SideEnemy, Initiative: 6},
		{EntityID: 102, Side: combat.SidePlayer, Initiative: 5},
		{EntityID: 103, Side: combat.SidePlayer, Initiative: 3},
		{EntityID: 203, Side: combat.SideEnemy, Initiative: 2},
		{EntityID: 104, Side: combat.SidePlayer, Initiative: 1},
	}
	if got := hud.TurnOrder(); !reflect.DeepEqual(got, want) {
		t.Fatalf("turn order after removal = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// AC-3: turn change highlight
// ---------------------------------------------------------------------------

// test_highlight_follows_turn_started: the amber highlight advances to the
// combatant named by EventTurnStarted.
func TestHighlightFollowsTurnStarted(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())

	emitTurnStarted(t, bus, 101, 1)
	if cur, ok := hud.CurrentCombatant(); !ok || cur != 101 {
		t.Fatalf("current after first TurnStarted = (%d, %v), want (101, true)", cur, ok)
	}

	emitTurnEnded(t, bus, 101, 1)
	emitTurnStarted(t, bus, 201, 1)
	if cur, ok := hud.CurrentCombatant(); !ok || cur != 201 {
		t.Fatalf("current after second TurnStarted = (%d, %v), want (201, true)", cur, ok)
	}
}

// test_turn_ended_marks_acted: a combatant whose turn ended is marked
// acted this round (further dimmed portrait).
func TestTurnEndedMarksActed(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	if hud.IsActed(101) {
		t.Fatal("combatant must not be acted while its turn runs")
	}
	emitTurnEnded(t, bus, 101, 1)
	if !hud.IsActed(101) {
		t.Fatal("combatant must be marked acted after EventTurnEnded")
	}
}

// test_round_wrap_resets_acted: when the turn order wraps, the acted
// marks reset for the new round.
func TestRoundWrapResetsActed(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())

	// Round 1: every combatant acts in order.
	for _, e := range ac1TurnOrder() {
		emitTurnStarted(t, bus, e.EntityID, 1)
		emitTurnEnded(t, bus, e.EntityID, 1)
	}
	// Round 2 begins with the first combatant again.
	emitTurnStarted(t, bus, 101, 2)

	if got := hud.Round(); got != 2 {
		t.Fatalf("round = %d, want 2", got)
	}
	for _, e := range ac1TurnOrder() {
		if hud.IsActed(e.EntityID) {
			t.Fatalf("combatant %d must not be acted in round 2", e.EntityID)
		}
	}
	if cur, ok := hud.CurrentCombatant(); !ok || cur != 101 {
		t.Fatalf("current at round 2 = (%d, %v), want (101, true)", cur, ok)
	}
}

// ---------------------------------------------------------------------------
// AC-4: combat end
// ---------------------------------------------------------------------------

// test_combat_ended_resets_state: on EventCombatEnded the HUD clears all
// encounter state and ignores late turn events.
func TestCombatEndedResetsState(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	emitCombatEnded(t, bus, combat.OutcomeVictory)

	if hud.Active() {
		t.Fatal("HUD must be inactive after EventCombatEnded")
	}
	if got := hud.TurnOrder(); len(got) != 0 {
		t.Fatalf("turn order after combat end = %v, want empty", got)
	}
	if _, ok := hud.CurrentCombatant(); ok {
		t.Fatal("no combatant may be current after combat end")
	}
	if got := hud.Round(); got != 0 {
		t.Fatalf("round after combat end = %d, want 0", got)
	}

	// A straggler TurnStarted after the end must not resurrect state.
	emitTurnStarted(t, bus, 101, 1)
	if _, ok := hud.CurrentCombatant(); ok {
		t.Fatal("late TurnStarted must be ignored after combat end")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// test_restart_combat_clears_stale_state: a second encounter starts from a
// clean slate (no stale acted marks, removals, or current combatant).
func TestRestartCombatClearsStaleState(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	order := ac1TurnOrder()

	// First encounter: advance, remove, end.
	emitCombatStarted(t, bus, order)
	emitTurnStarted(t, bus, 101, 1)
	emitTurnEnded(t, bus, 101, 1)
	emitCombatantRemoved(t, bus, 202, combat.RemovalFled)
	emitCombatEnded(t, bus, combat.OutcomeVictory)

	// Second encounter.
	emitCombatStarted(t, bus, order)
	if !hud.Active() {
		t.Fatal("HUD must be active for the second encounter")
	}
	if got := hud.TurnOrder(); !reflect.DeepEqual(got, order) {
		t.Fatalf("turn order on restart = %v, want full order %v", got, order)
	}
	if hud.IsActed(101) {
		t.Fatal("acted marks must not survive a combat restart")
	}
	if got := hud.Round(); got != 1 {
		t.Fatalf("round on restart = %d, want 1", got)
	}
}

// test_removed_current_keeps_turn_slot: the combat system completes the
// active turn before removal takes effect, so the removed combatant keeps
// the current slot until the next EventTurnStarted advances it (AC-3
// edge: "complete their turn, then remove from order").
func TestRemovedCurrentKeepsTurnSlot(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	emitCombatantRemoved(t, bus, 101, combat.RemovalDefeated)

	if cur, ok := hud.CurrentCombatant(); !ok || cur != 101 {
		t.Fatalf("current after removing the active combatant = (%d, %v), want (101, true)", cur, ok)
	}
	order := hud.TurnOrder()
	if len(order) == 0 {
		t.Fatal("turn order must be non-empty")
	}
	for _, e := range order {
		if e.EntityID == 101 {
			t.Fatal("removed combatant must not remain in the order")
		}
	}

	// The next TurnStarted advances the highlight normally.
	emitTurnStarted(t, bus, 201, 1)
	if cur, ok := hud.CurrentCombatant(); !ok || cur != 201 {
		t.Fatalf("current after advancing = (%d, %v), want (201, true)", cur, ok)
	}
}

// test_wrong_payload_types_ignored: a malformed payload must not panic or
// corrupt state.
func TestWrongPayloadTypesIgnored(t *testing.T) {
	hud, bus := newHeadlessHUD(t)

	bus.Emit(entity.Event{Type: combat.EventCombatStarted, Source: "test", Data: "not a payload"})
	if hud.Active() {
		t.Fatal("malformed CombatStarted payload must be ignored")
	}
	bus.Emit(entity.Event{Type: combat.EventCombatantRemoved, Source: "test", Data: 42})
	bus.Emit(entity.Event{Type: combat.EventTurnStarted, Source: "test", Data: nil})
	bus.Emit(entity.Event{Type: combat.EventTurnEnded, Source: "test", Data: struct{}{}})
	// Still inactive and unpanicked.
	if hud.Active() {
		t.Fatal("HUD must remain inactive after malformed events")
	}
}

// test_turn_events_ignored_outside_combat: turn and removal events without
// an active encounter are ignored (the combat system never emits them, but
// the HUD must be defensive).
func TestTurnEventsIgnoredOutsideCombat(t *testing.T) {
	hud, bus := newHeadlessHUD(t)

	emitTurnStarted(t, bus, 101, 1)
	emitTurnEnded(t, bus, 101, 1)
	emitCombatantRemoved(t, bus, 202, combat.RemovalDefeated)
	emitCombatEnded(t, bus, combat.OutcomeDefeat)

	if hud.Active() {
		t.Fatal("HUD must remain inactive outside combat")
	}
	if _, ok := hud.CurrentCombatant(); ok {
		t.Fatal("no combatant may be current outside combat")
	}
	if got := hud.TurnOrder(); len(got) != 0 {
		t.Fatalf("turn order outside combat = %v, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// AC-5: tooltip name resolution
// ---------------------------------------------------------------------------

// test_resolve_combatant_name_fallback: the tooltip name comes from the
// injected NameReader, falling back to "Combatant N" when unavailable.
func TestResolveCombatantNameFallback(t *testing.T) {
	if got := combat_hud.ResolveCombatantName(nil, 101); got != "Combatant 101" {
		t.Fatalf("nil reader name = %q, want %q", got, "Combatant 101")
	}
	if got := combat_hud.ResolveCombatantName(stubNameReader{names: map[entity.GameEntityID]string{102: ""}}, 102); got != "Combatant 102" {
		t.Fatalf("empty reader name = %q, want %q", got, "Combatant 102")
	}
	reader := stubNameReader{names: map[entity.GameEntityID]string{103: "Verena"}}
	if got := combat_hud.ResolveCombatantName(reader, 103); got != "Verena" {
		t.Fatalf("reader name = %q, want %q", got, "Verena")
	}
}

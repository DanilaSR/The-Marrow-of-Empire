// Integration tests for Story 002: AP Meter, Ability Menu & End Turn.
//
// Evidence path: production/epics/combat-ui/story-002-ap-meter-ability-menu-end-turn.md
// Required evidence: this file must exist and pass (integration gate).
//
// These tests drive the HUD headlessly (NewCombatHUD with nil host and
// nil ui.Manager): the real synchronous EventBus (ADR-0011), the real
// combat payload types, and a stub combat.Manager exercise the
// subscription layer and the input paths end to end — an event emitted on
// the bus reaches the models exactly as it would in a running game. The
// stub manager (stubCombatManager) serves the AP readouts, records End
// Turn calls, and — when started via StartCombat — replays the real
// manager's TurnEnded/TurnStarted advance contract on EndTurn (whose
// behavior the combat integration tests verify for the real
// CombatManager). Rendering (bar width, classes, pulse glow) is covered
// by the manual walkthrough (production/qa/evidence).
//
// The acceptance criteria map to test functions 1:1:
//
//	AC-1  test_ap_meter_depletes_on_spend            -> TestAPMeterDepletesOnSpend
//	AC-2  test_ability_menu_shows_six_actions        -> TestAbilityMenuShowsSixActionsOnPlayerTurn
//	AC-2  test_enemy_turn_hides_action_bar           -> TestEnemyTurnHidesActionBarKeepsMeter
//	AC-3  test_ability_greyout_follows_ap_cost       -> TestAbilityGreyoutFollowsAPCost
//	AC-4  test_ability_select_toggle_deselect_event  -> TestAbilitySelectToggleDeselectAndEvent
//	AC-4  test_end_turn_calls_manager_with_guard     -> TestEndTurnCallsManagerWithDoubleTriggerGuard
//	AC-4  test_end_turn_advances_to_next_combatant   -> TestEndTurnAdvancesToNextCombatant
//	AC-5  test_ap_meter_refill_animates_and_pulses   -> TestAPMeterRefillAnimatesAndPulses
//	edge  test_turn_ended_clears_selection           -> TestTurnEndedClearsSelectionAndMenu
//	edge  test_combat_end_resets_action_bar          -> TestCombatEndResetsActionBarAndMeter
//	edge  test_reduced_motion_skips_refill           -> TestReducedMotionSkipsRefillAnimation
//	edge  test_nil_manager_tolerated                 -> TestNilManagerTolerated
package combat_ui_test

import (
	"testing"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/ui/combat_hud"
)

// ---------------------------------------------------------------------------
// Test double: stub combat manager
// ---------------------------------------------------------------------------

// stubCombatManager implements combat.Manager for the HUD tests. It
// serves the AP readouts (curAP/maxAP), records End Turn calls, and —
// when started via StartCombat — simulates the real manager's turn
// advance on EndTurn by emitting TurnEnded/TurnStarted over the bus with
// the same payloads the real CombatManager emits (turn start resets the
// pool, so curAP = maxAP at every TurnStarted). Tests that only need AP
// readouts drive events manually with the story-001 helpers instead.
type stubCombatManager struct {
	bus   *entity.EventBusImpl
	curAP int
	maxAP int

	endTurns int
	order    []combat.TurnEntry
	idx      int
	current  entity.GameEntityID
	active   bool
	round    int
}

// Compile-time assertion: the stub satisfies combat.Manager.
var _ combat.Manager = (*stubCombatManager)(nil)

// newStubManager creates a stub wired to the bus with the given per-turn
// AP budget. curAP starts at the budget — the "fresh turn" contract the
// real manager guarantees at every TurnStarted.
func newStubManager(bus *entity.EventBusImpl, maxAP int) *stubCombatManager {
	return &stubCombatManager{bus: bus, maxAP: maxAP, curAP: maxAP}
}

func (s *stubCombatManager) StartCombat(entries []combat.CombatantEntry) error {
	if len(entries) == 0 {
		return combat.ErrNoCombatants
	}
	s.active = true
	s.round = 1
	s.order = make([]combat.TurnEntry, len(entries))
	for i, e := range entries {
		s.order[i] = combat.TurnEntry{EntityID: e.EntityID, Side: e.Side}
	}
	s.idx = 0
	s.bus.Emit(entity.Event{
		Type:   combat.EventCombatStarted,
		Source: "test-stub",
		Data:   combat.CombatStartedData{TurnOrder: append([]combat.TurnEntry(nil), s.order...)},
	})
	s.beginTurn()
	return nil
}

func (s *stubCombatManager) EndTurn() error {
	if !s.active {
		return combat.ErrNoActiveCombat
	}
	s.endTurns++
	s.bus.Emit(entity.Event{
		Type:   combat.EventTurnEnded,
		Source: "test-stub",
		Data:   combat.TurnEndedData{Combatant: s.current, Round: s.round},
	})
	s.idx = (s.idx + 1) % len(s.order)
	if s.idx == 0 {
		s.round++
	}
	s.beginTurn()
	return nil
}

func (s *stubCombatManager) RemoveCombatant(entity.GameEntityID, combat.RemovalReason) error {
	return nil
}

func (s *stubCombatManager) EndCombat(combat.Outcome) error {
	s.active = false
	return nil
}

func (s *stubCombatManager) IsCombatActive() bool            { return s.active }
func (s *stubCombatManager) CurrentState() combat.State       { return combat.StatePlayerTurn }
func (s *stubCombatManager) CurrentCombatant() (entity.GameEntityID, bool) {
	return s.current, s.active
}
func (s *stubCombatManager) TurnOrder() []combat.TurnEntry {
	return append([]combat.TurnEntry(nil), s.order...)
}
func (s *stubCombatManager) Round() int      { return s.round }
func (s *stubCombatManager) CurrentAP() int  { return s.curAP }
func (s *stubCombatManager) MaxAP() int      { return s.maxAP }

// beginTurn simulates the real manager's turn start: the pool resets to
// the combatant's full budget, then TurnStarted fires.
func (s *stubCombatManager) beginTurn() {
	e := s.order[s.idx]
	s.current = e.EntityID
	s.curAP = s.maxAP
	s.bus.Emit(entity.Event{
		Type:   combat.EventTurnStarted,
		Source: "test-stub",
		Data:   combat.TurnStartedData{Combatant: e.EntityID, Side: e.Side, Round: s.round},
	})
}

// emitTurnStartedSide emits TurnStarted for the given side (the story-001
// helper hardcodes SidePlayer).
func emitTurnStartedSide(t *testing.T, bus *entity.EventBusImpl, id entity.GameEntityID, side combat.Side, round int) {
	t.Helper()
	bus.Emit(entity.Event{
		Type:   combat.EventTurnStarted,
		Source: "test",
		Data:   combat.TurnStartedData{Combatant: id, Side: side, Round: round},
	})
}

// newHeadlessHUDWithStub constructs the HUD headlessly over the given
// stub manager (for tests that assert on the stub's state).
func newHeadlessHUDWithStub(t *testing.T, stub *stubCombatManager) (*combat_hud.CombatHUD, *entity.EventBusImpl) {
	t.Helper()
	hud := combat_hud.NewCombatHUD(nil, nil, stub.bus, nil, nil, stub)
	if hud == nil {
		t.Fatal("NewCombatHUD returned nil")
	}
	return hud, stub.bus
}

// ---------------------------------------------------------------------------
// AC-1: AP bar depletes in real-time
// ---------------------------------------------------------------------------

// test_ap_meter_depletes_on_spend: spending AP drives the fill down to
// current/max over 150ms with the readout counting down (AC-1). The
// combat system does not emit an AP change event yet, so the test drives
// CombatHUD.SetAP — the hook the future EventAPChanged wiring calls.
func TestAPMeterDepletesOnSpend(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)
	hud.Update(0.21) // refill settles at 8 / 8
	if hud.APText() != "8 / 8" {
		t.Fatalf("readout before spend = %q, want %q", hud.APText(), "8 / 8")
	}

	hud.SetAP(5)
	hud.Update(0.08) // mid-depletion: fill must be between 5/8 and full
	if f := hud.APFill(); f <= 0.625 || f >= 1.0 {
		t.Fatalf("mid-depletion fill = %v, want in (0.625, 1.0)", f)
	}
	hud.Update(0.08) // 0.16s total > 150ms — depletion completes
	if hud.APFill() != 0.625 {
		t.Fatalf("fill after depletion = %v, want 0.625", hud.APFill())
	}
	if got := hud.APText(); got != "5 / 8" {
		t.Fatalf("readout after depletion = %q, want %q", got, "5 / 8")
	}
	if s := hud.APPulseStrength(); s != 0 {
		t.Fatalf("depletion must not pulse, got %v", s)
	}

	hud.SetAP(3)
	hud.Update(0.16)
	if got := hud.APText(); got != "3 / 8" {
		t.Fatalf("readout after second spend = %q, want %q", got, "3 / 8")
	}
}

// ---------------------------------------------------------------------------
// AC-2: ability menu displays all six actions
// ---------------------------------------------------------------------------

// test_ability_menu_shows_six_actions: on a player turn the action bar
// shows the six Story 002 actions with their hotkeys and AP costs (the
// Abilities slot shows the 1-3 range), and every slot is affordable at
// full AP.
func TestAbilityMenuShowsSixActionsOnPlayerTurn(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	if !hud.AbilityActive() {
		t.Fatal("ability menu must be active on a player turn")
	}
	slots := hud.AbilitySlots()
	if len(slots) != 6 {
		t.Fatalf("ability slots = %d, want 6", len(slots))
	}
	want := []struct {
		hotkey, cost int
		name, label  string
	}{
		{1, 1, "Move", "1"},
		{2, 2, "Attack", "2"},
		{3, 1, "Abilities", "1-3"},
		{4, 1, "Items", "1"},
		{5, 1, "Defend", "1"},
		{6, 0, "Wait", "0"},
	}
	for i, w := range want {
		s := slots[i]
		if s.Hotkey != w.hotkey || s.Cost != w.cost || s.Name != w.name || s.CostLabel() != w.label {
			t.Fatalf("slot %d = {hotkey %d, %q, cost %d, label %q}, want {hotkey %d, %q, cost %d, label %q}",
				i, s.Hotkey, s.Name, s.Cost, s.CostLabel(), w.hotkey, w.name, w.cost, w.label)
		}
	}
	// Full AP: cost <= AP keeps every ability active (AC-3 boundary).
	for i := 1; i <= 6; i++ {
		if !hud.CanAffordAbility(i) {
			t.Fatalf("ability %d must be affordable at full AP", i)
		}
	}
}

// test_enemy_turn_hides_action_bar: the menu and End Turn belong to
// player-controlled turns only (AC-2); the AP meter still tracks the
// active combatant during enemy turns.
func TestEnemyTurnHidesActionBarKeepsMeter(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStartedSide(t, bus, 201, combat.SideEnemy, 1)

	if hud.AbilityActive() {
		t.Fatal("ability menu must be hidden on an enemy turn")
	}
	if next, ok := hud.SelectAbility(2); ok || next != 0 {
		t.Fatalf("ability select on enemy turn = (%d, %v), want (0, false)", next, ok)
	}
	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("end turn on enemy turn error = %v, want nil", err)
	}
	if hud.EndTurnLocked() {
		t.Fatal("enemy turns must not arm the End Turn guard")
	}
	if hud.APMax() != 8 {
		t.Fatalf("meter max on enemy turn = %d, want 8", hud.APMax())
	}
	hud.Update(0.21)
	if got := hud.APText(); got != "8 / 8" {
		t.Fatalf("meter readout on enemy turn = %q, want %q", got, "8 / 8")
	}
}

// ---------------------------------------------------------------------------
// AC-3: abilities grey out when AP is insufficient
// ---------------------------------------------------------------------------

// test_ability_greyout_follows_ap_cost: grey state matches the
// currentAP < cost rule (AC-3): at 2 AP, Attack (2) stays active because
// cost == AP; at 1 AP it greys out while the 1 AP actions and Wait remain
// active; disabled abilities do not respond to selection.
func TestAbilityGreyoutFollowsAPCost(t *testing.T) {
	bus := entity.NewEventBus()
	stub := newStubManager(bus, 8)
	hud, _ := newHeadlessHUDWithStub(t, stub)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	// 2 AP remaining (QA setup after spending): Attack (2) active —
	// cost == AP stays active.
	hud.SetAP(2)
	if !hud.CanAffordAbility(2) {
		t.Fatal("Attack must be active at exactly 2 AP (cost == AP)")
	}
	for _, hk := range []int{1, 3, 4, 5, 6} {
		if !hud.CanAffordAbility(hk) {
			t.Fatalf("ability %d must stay active at 2 AP", hk)
		}
	}

	// 1 AP: Attack greys out; only cost <= 1 remains.
	hud.SetAP(1)
	if hud.CanAffordAbility(2) {
		t.Fatal("Attack must grey out at 1 AP")
	}
	for _, hk := range []int{1, 3, 4, 5, 6} {
		if !hud.CanAffordAbility(hk) {
			t.Fatalf("ability %d must stay active at 1 AP", hk)
		}
	}

	// 0 AP: only Wait (0) remains.
	hud.SetAP(0)
	for _, hk := range []int{1, 2, 3, 4, 5} {
		if hud.CanAffordAbility(hk) {
			t.Fatalf("ability %d must grey out at 0 AP", hk)
		}
	}
	if !hud.CanAffordAbility(6) {
		t.Fatal("Wait (0 AP) must remain active at 0 AP")
	}

	// Disabled abilities do not respond to click or hotkey (AC-3).
	if next, ok := hud.SelectAbility(2); ok || next != 0 {
		t.Fatalf("unaffordable Attack select = (%d, %v), want (0, false)", next, ok)
	}
	if next, ok := hud.SelectAbility(6); !ok || next != 6 {
		t.Fatalf("affordable Wait select = (%d, %v), want (6, true)", next, ok)
	}
}

// ---------------------------------------------------------------------------
// AC-4: ability selection and End Turn
// ---------------------------------------------------------------------------

// test_ability_select_toggle_deselect_event: selecting an ability
// highlights it (model selection) and emits EventAbilitySelected with the
// active combatant and slot; re-selecting deselects without an event; one
// ability is selected at a time; unknown hotkeys are ignored (AC-4).
func TestAbilitySelectToggleDeselectAndEvent(t *testing.T) {
	bus := entity.NewEventBus()
	stub := newStubManager(bus, 8)
	hud, _ := newHeadlessHUDWithStub(t, stub)

	var got []combat.AbilitySelectedData
	bus.Subscribe(combat.EventAbilitySelected, 1, func(ev entity.Event) {
		if d, ok := ev.Data.(combat.AbilitySelectedData); ok {
			got = append(got, d)
		}
	})

	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	if next, ok := hud.SelectAbility(2); !ok || next != 2 {
		t.Fatalf("select Attack = (%d, %v), want (2, true)", next, ok)
	}
	if hud.SelectedAbility() != 2 {
		t.Fatalf("selected = %d, want 2", hud.SelectedAbility())
	}
	if len(got) != 1 || got[0].Combatant != 101 || got[0].Slot != 2 {
		t.Fatalf("ability events = %+v, want one {Combatant: 101, Slot: 2}", got)
	}

	// Re-selecting deselects (AC-4); a deselection emits no event.
	if next, ok := hud.SelectAbility(2); !ok || next != 0 {
		t.Fatalf("deselect Attack = (%d, %v), want (0, true)", next, ok)
	}
	if hud.SelectedAbility() != 0 {
		t.Fatal("selection must clear on deselect")
	}
	if len(got) != 1 {
		t.Fatalf("deselection must not emit an event, got %d events", len(got))
	}

	// Switching selection moves the highlight and emits the new ability.
	if _, ok := hud.SelectAbility(4); !ok {
		t.Fatal("select Items must succeed")
	}
	if hud.SelectedAbility() != 4 {
		t.Fatalf("selected = %d, want 4", hud.SelectedAbility())
	}
	if len(got) != 2 || got[1].Slot != 4 {
		t.Fatalf("ability events = %+v, want second event with Slot 4", got)
	}

	// Unknown hotkeys are ignored.
	if _, ok := hud.SelectAbility(0); ok {
		t.Fatal("hotkey 0 must be ignored")
	}
	if _, ok := hud.SelectAbility(7); ok {
		t.Fatal("hotkey 7 must be ignored")
	}
}

// test_end_turn_calls_manager_with_guard: an End Turn press reaches
// combat.Manager.EndTurn exactly once per 300ms guard window (AC-4
// double-trigger protection).
func TestEndTurnCallsManagerWithDoubleTriggerGuard(t *testing.T) {
	bus := entity.NewEventBus()
	stub := newStubManager(bus, 8)
	hud, _ := newHeadlessHUDWithStub(t, stub)
	// Start the stub through its manager contract (it emits CombatStarted
	// and TurnStarted itself); a stub that was never started correctly
	// refuses EndTurn with ErrNoActiveCombat, mirroring the real manager.
	if err := stub.StartCombat([]combat.CombatantEntry{
		{EntityID: 101, Side: combat.SidePlayer},
	}); err != nil {
		t.Fatalf("start combat error = %v", err)
	}

	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("end turn error = %v, want nil", err)
	}
	if stub.endTurns != 1 {
		t.Fatalf("EndTurn calls = %d, want 1", stub.endTurns)
	}
	if !hud.EndTurnLocked() {
		t.Fatal("double-trigger guard must be active after a press")
	}

	// A second press inside the guard window must not reach the manager.
	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("guarded end turn error = %v, want nil", err)
	}
	if stub.endTurns != 1 {
		t.Fatalf("EndTurn calls inside guard = %d, want 1", stub.endTurns)
	}

	// After 300ms the guard expires.
	hud.Update(0.31)
	if hud.EndTurnLocked() {
		t.Fatal("guard must expire after 300ms")
	}
	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("end turn after expiry error = %v, want nil", err)
	}
	if stub.endTurns != 2 {
		t.Fatalf("EndTurn calls after expiry = %d, want 2", stub.endTurns)
	}
}

// test_end_turn_advances_to_next_combatant: pressing End Turn ends the
// current turn and transitions to the next combatant (AC-4) — the stub
// manager replays the real manager's TurnEnded/TurnStarted contract over
// the bus, so the HUD's own state advances with it.
func TestEndTurnAdvancesToNextCombatant(t *testing.T) {
	bus := entity.NewEventBus()
	stub := newStubManager(bus, 8)
	hud, _ := newHeadlessHUDWithStub(t, stub)
	if err := stub.StartCombat([]combat.CombatantEntry{
		{EntityID: 101, Side: combat.SidePlayer},
		{EntityID: 102, Side: combat.SidePlayer},
	}); err != nil {
		t.Fatalf("start combat error = %v", err)
	}

	if cur, ok := hud.CurrentCombatant(); !ok || cur != 101 {
		t.Fatalf("current = (%d, %v), want (101, true)", cur, ok)
	}
	hud.Update(0.21) // refill settles
	if !hud.AbilityActive() {
		t.Fatal("action bar must be active on the first player turn")
	}
	if _, ok := hud.SelectAbility(2); !ok {
		t.Fatal("select Attack must succeed")
	}

	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("end turn error = %v, want nil", err)
	}
	if stub.endTurns != 1 {
		t.Fatalf("EndTurn calls = %d, want 1", stub.endTurns)
	}
	if cur, ok := hud.CurrentCombatant(); !ok || cur != 102 {
		t.Fatalf("current after end turn = (%d, %v), want (102, true)", cur, ok)
	}
	if hud.SelectedAbility() != 0 {
		t.Fatalf("selection must clear on turn change, got %d", hud.SelectedAbility())
	}
	if !hud.AbilityActive() {
		t.Fatal("action bar must stay active for the next player turn")
	}
	if hud.APCurrent() != 8 || hud.APMax() != 8 {
		t.Fatalf("AP after turn change = (%d/%d), want (8/8)", hud.APCurrent(), hud.APMax())
	}
}

// ---------------------------------------------------------------------------
// AC-5: AP refill on a new turn
// ---------------------------------------------------------------------------

// test_ap_meter_refill_animates_and_pulses: the fill animates from its
// previous level to full over 200ms with the readout counting up, and a
// brief amber pulse fires at completion (AC-5). The reduced-motion gate
// skips the animation and the pulse.
func TestAPMeterRefillAnimatesAndPulses(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	// The meter starts empty and animates to full.
	if got := hud.APText(); got != "0 / 8" {
		t.Fatalf("readout at turn start = %q, want %q", got, "0 / 8")
	}
	if hud.APFill() != 0 {
		t.Fatalf("fill at turn start = %v, want 0", hud.APFill())
	}

	hud.Update(0.1) // mid-refill (200ms total)
	if f := hud.APFill(); f <= 0 || f >= 1 {
		t.Fatalf("mid-refill fill = %v, want in (0, 1)", f)
	}

	hud.Update(0.11) // 0.21s total — refill completes
	if hud.APFill() != 1.0 {
		t.Fatalf("fill after refill = %v, want 1.0", hud.APFill())
	}
	if got := hud.APText(); got != "8 / 8" {
		t.Fatalf("readout after refill = %q, want %q", got, "8 / 8")
	}
	if s := hud.APPulseStrength(); s <= 0 {
		t.Fatalf("amber pulse must fire at refill completion, got %v", s)
	}

	hud.Update(0.16) // pulse window (150ms) passes
	if s := hud.APPulseStrength(); s != 0 {
		t.Fatalf("pulse must end after 150ms, got %v", s)
	}
}

// test_reduced_motion_skips_refill: under the reduced-motion gate the
// refill is instant and no pulse runs (accessibility: "0ms").
func TestReducedMotionSkipsRefillAnimation(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	hud.SetReducedMotion(true)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	if hud.APFill() != 1.0 {
		t.Fatalf("fill under reduced motion = %v, want 1.0 (instant)", hud.APFill())
	}
	if got := hud.APText(); got != "8 / 8" {
		t.Fatalf("readout under reduced motion = %q, want %q", got, "8 / 8")
	}
	hud.Update(0.2)
	if s := hud.APPulseStrength(); s != 0 {
		t.Fatalf("pulse under reduced motion = %v, want 0", s)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// test_turn_ended_clears_selection: ending the turn hides the menu and
// clears the selection (it belonged to the previous combatant).
func TestTurnEndedClearsSelectionAndMenu(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)
	if _, ok := hud.SelectAbility(2); !ok {
		t.Fatal("select Attack must succeed")
	}

	emitTurnEnded(t, bus, 101, 1)

	if hud.AbilityActive() {
		t.Fatal("menu must hide when the turn ends")
	}
	if hud.SelectedAbility() != 0 {
		t.Fatalf("selection must clear on turn end, got %d", hud.SelectedAbility())
	}
}

// test_combat_end_resets_action_bar: combat end clears the meter (0 / 0),
// the menu, the selection, and the End Turn guard.
func TestCombatEndResetsActionBarAndMeter(t *testing.T) {
	hud, bus := newHeadlessHUD(t)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)
	if _, ok := hud.SelectAbility(2); !ok {
		t.Fatal("select Attack must succeed")
	}
	hud.SetAP(3)
	hud.Update(0.21)
	if hud.APText() != "3 / 8" {
		t.Fatalf("readout before combat end = %q, want %q", hud.APText(), "3 / 8")
	}

	emitCombatEnded(t, bus, combat.OutcomeVictory)

	if hud.AbilityActive() {
		t.Fatal("menu must hide after combat end")
	}
	if hud.SelectedAbility() != 0 {
		t.Fatalf("selection must clear after combat end, got %d", hud.SelectedAbility())
	}
	if hud.APCurrent() != 0 || hud.APMax() != 0 {
		t.Fatalf("meter after combat end = (%d/%d), want (0/0)", hud.APCurrent(), hud.APMax())
	}
	if got := hud.APText(); got != "0 / 0" {
		t.Fatalf("readout after combat end = %q, want %q", got, "0 / 0")
	}
}

// test_nil_manager_tolerated: a nil manager (headless context without a
// combat system) leaves the meter empty and makes End Turn a no-op
// without panicking.
func TestNilManagerTolerated(t *testing.T) {
	bus := entity.NewEventBus()
	hud := combat_hud.NewCombatHUD(nil, nil, bus, nil, nil, nil)
	emitCombatStarted(t, bus, ac1TurnOrder())
	emitTurnStarted(t, bus, 101, 1)

	if got := hud.APText(); got != "0 / 0" {
		t.Fatalf("readout with nil manager = %q, want %q", got, "0 / 0")
	}
	if next, ok := hud.SelectAbility(6); !ok || next != 6 {
		t.Fatalf("Wait select with nil manager = (%d, %v), want (6, true)", next, ok)
	}
	if err := hud.PressEndTurn(); err != nil {
		t.Fatalf("end turn with nil manager error = %v, want nil", err)
	}
}

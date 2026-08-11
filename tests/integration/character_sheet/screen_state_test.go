// Integration tests for Story 007: Screen State Machine & Combat
// Integration.
//
// Evidence path: production/epics/character-inventory-ui/
// story-007-screen-state-machine-combat.md
// Required evidence: this file must exist and pass (Integration story type —
// BLOCKING gate).
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-15  test_combat_blocks_open_keys                        -> TestCombatBlocksOpenKeys
//	AC-15  test_combat_end_unblocks_open_keys                  -> TestCombatEndUnblocksOpenKeys
//	AC-16  test_combat_forces_close_on_character_tab           -> TestCombatForcesCloseOnCharacterTab
//	AC-16  test_combat_forces_close_on_inventory_tab           -> TestCombatForcesCloseOnInventoryTab
//	AC-16  test_combat_forces_close_defers_level_up_modal      -> TestCombatForcesCloseDefersLevelUpModal
//	AC-17  test_switch_tab_preserves_member_and_crossfades     -> TestSwitchTabPreservesMemberAndCrossfades
//	AC-17  test_tab_crossfade_duration_is_200ms                -> TestTabCrossfadeDurationIs200ms
//	edge   test_rapid_tab_switching_latest_target_wins         -> TestRapidTabSwitchingLatestTargetWins
//	edge   test_double_open_is_noop                            -> TestDoubleOpenIsNoOp
//	edge   test_toggle_key_closes_on_same_tab                  -> TestToggleKeyClosesOnSameTab
//	edge   test_toggle_switches_from_other_tab                 -> TestToggleSwitchesFromOtherTab
//	edge   test_esc_closes_with_animated_exit                  -> TestEscClosesWithAnimatedExit
//	edge   test_close_button_closes                            -> TestCloseButtonCloses
//	edge   test_forced_close_vs_user_close                     -> TestForcedCloseVsUserClose
//	edge   test_input_blocked_while_level_up_modal_active      -> TestInputBlockedWhileLevelUpModalActive
//	edge   test_level_up_event_shows_modal                     -> TestLevelUpEventShowsModal
//	edge   test_level_up_while_closed_is_deferred              -> TestLevelUpWhileClosedIsDeferred
//	edge   test_open_uses_last_viewed_member                   -> TestOpenUsesLastViewedMember
//	edge   test_switch_tab_noop_cases                          -> TestSwitchTabNoOpCases
//	edge   test_open_failure_leaves_ui_closed                  -> TestOpenFailureLeavesUIClosed
//	edge   test_level_up_hp_gain_derivation                    -> TestLevelUpHpGainDerivation
//	edge   test_level_up_milestone_crossing                    -> TestLevelUpMilestoneCrossing
//	edge   test_level_up_malformed_payload_ignored             -> TestLevelUpMalformedPayloadIgnored
//	edge   test_combat_started_ignores_payload                 -> TestCombatStartedIgnoresPayload
//	edge   test_teardown_unsubscribes                          -> TestTeardownUnsubscribes
//
// The tests drive the state machine headlessly: NewCharacterSheet with nil
// host and nil UI manager skips markup loading (degraded sheet) and the
// level-up modal is constructed as a pure state machine (nil document,
// Story 002 test pattern), while the real synchronous EventBus (ADR-0011)
// and the real combat/stats payload types exercise the subscription layer
// end to end — an event emitted on the bus reaches the state machine
// exactly as it would in a running game. Key polling (host) is not
// exercised: the C/I/Esc handlers are called directly, and Update without
// a host only advances the crossfade and close timers.
package character_sheet_test

import (
	"errors"
	"testing"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------
//
// newTestStore and createCharacter live in equipment_slots_test.go (same
// package — Go cannot import test helpers across packages, so the shared
// helpers are per-package). A real CharacterSheet cannot be constructed
// headlessly (NewCharacterSheet requires a live UI manager to load the
// markup), so the state machine's Sheet interface is stubbed here.

// stubSheet is a test double for the state machine's Sheet interface: it
// records open/close calls and tracks visibility, so tests can assert
// exactly which close path ran — animated Close (user request) vs
// immediate ForceClose (combat start).
type stubSheet struct {
	openErr         error
	visible         bool
	openedFor       entity.GameEntityID
	closeCalls      int
	forceCloseCalls int
}

func (s *stubSheet) Open(id entity.GameEntityID) error {
	s.openedFor = id
	s.visible = true
	return s.openErr
}

func (s *stubSheet) Close() {
	s.closeCalls++
	s.visible = false
}

func (s *stubSheet) ForceClose() {
	s.forceCloseCalls++
	s.visible = false
}

func (s *stubSheet) Visible() bool { return s.visible }

// newScreenMachine wires a state machine over a stub sheet and a nil-doc
// modal (pure state machine, Story 002 test pattern), with the real
// synchronous EventBus (ADR-0011) and a real store. playerID is the
// session's default last-viewed member.
func newScreenMachine(t *testing.T, playerID entity.GameEntityID) (*character_sheet.ScreenStateMachine, *entity.EventBusImpl, *entity.EntityStoreImpl, *stubSheet) {
	t.Helper()
	store, _ := newTestStore()
	bus := entity.NewEventBus()
	sheet := &stubSheet{}
	modal := character_sheet.NewLevelUpModal(nil, store, nil)
	sm := character_sheet.NewScreenStateMachine(sheet, modal, store, bus, playerID, nil)
	if sm == nil {
		t.Fatal("NewScreenStateMachine returned nil")
	}
	return sm, bus, store, sheet
}

// levelThreeCharacter creates a character at level 3 with CON 14 (+2 mod):
// a 3→4 level-up grants 2 HP (max(1, +2) × 1 level) and crosses the
// level-4 milestone (attribute point) — the story-002 QA setup.
func levelThreeCharacter(t *testing.T, store *entity.EntityStoreImpl) entity.GameEntityID {
	t.Helper()
	attrs := [stats.NumAttributes]int{10, 10, 14, 10, 10, 10}
	id := createCharacter(t, store, attrs)
	if err := entity.SetComponent(store, id, stats.StatsComponent{Attributes: attrs, Level: 3}); err != nil {
		t.Fatalf("SetComponent(Stats level 3) error: %v", err)
	}
	return id
}

func emitCombatStarted(t *testing.T, bus *entity.EventBusImpl) {
	t.Helper()
	bus.Emit(entity.Event{Type: combat.EventCombatStarted, Source: "test"})
}

func emitCombatEnded(t *testing.T, bus *entity.EventBusImpl) {
	t.Helper()
	bus.Emit(entity.Event{Type: combat.EventCombatEnded, Source: "test"})
}

func emitLevelUp(t *testing.T, bus *entity.EventBusImpl, id entity.GameEntityID, newLevel int) {
	t.Helper()
	bus.Emit(entity.Event{
		Type:   stats.EventStatLevelUp,
		Source: "test",
		Data:   stats.LevelUpEventData{EntityID: id, NewLevel: newLevel},
	})
}

// ---------------------------------------------------------------------------
// AC-15: combat blocks UI
// ---------------------------------------------------------------------------

// test_combat_blocks_open_keys: GIVEN the C&I UI is closed and combat is
// active, WHEN the player presses C or I, THEN the UI does not open — every
// open command (keyboard, gamepad menu button, menu command) routes through
// the same methods and is silently ignored; no partial render, no flash.
func TestCombatBlocksOpenKeys(t *testing.T) {
	sm, bus, _, _ := newScreenMachine(t, 1)
	emitCombatStarted(t, bus)

	sm.OpenCharacter()
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("OpenCharacter during combat: State() = %v, want Closed", got)
	}
	sm.ToggleCharacter()
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("ToggleCharacter during combat: State() = %v, want Closed", got)
	}
	sm.OpenInventory()
	sm.ToggleInventory()
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("I key during combat: State() = %v, want Closed", got)
	}
	if sm.Visible() {
		t.Fatal("UI must not be visible during combat")
	}
}

// test_combat_end_unblocks_open_keys: GIVEN combat ends, WHEN the player
// presses C, THEN the UI opens — the combat-end event clears the flag
// synchronously, before the next input is processed (QA event-ordering
// edge).
func TestCombatEndUnblocksOpenKeys(t *testing.T) {
	sm, bus, _, _ := newScreenMachine(t, 1)
	emitCombatStarted(t, bus)
	emitCombatEnded(t, bus)

	sm.ToggleCharacter()
	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("State() after combat end + C = %v, want CharacterTab", got)
	}
}

// ---------------------------------------------------------------------------
// AC-16: combat forces close
// ---------------------------------------------------------------------------

// test_combat_forces_close_on_character_tab: GIVEN the UI is open on the
// Character tab, WHEN combat starts, THEN the UI closes immediately — no
// animation, no delay, no prompt; the parchment overlay is gone in the
// same frame.
func TestCombatForcesCloseOnCharacterTab(t *testing.T) {
	sm, bus, _, sheet := newScreenMachine(t, 1)
	sm.OpenCharacter()
	if !sm.Visible() {
		t.Fatal("UI must be open before combat starts")
	}

	emitCombatStarted(t, bus)

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() after combat start = %v, want Closed", got)
	}
	if sm.Visible() {
		t.Fatal("overlay must be gone immediately after combat start (no exit animation)")
	}
	if sm.Closing() {
		t.Fatal("a forced close must not run the animated exit")
	}
	if sheet.forceCloseCalls != 1 || sheet.closeCalls != 0 {
		t.Fatalf("close calls = (close %d, forceClose %d), want (0, 1) — immediate path only",
			sheet.closeCalls, sheet.forceCloseCalls)
	}
}

// test_combat_forces_close_on_inventory_tab: the same immediate forced
// close applies from the Inventory tab.
func TestCombatForcesCloseOnInventoryTab(t *testing.T) {
	sm, bus, _, sheet := newScreenMachine(t, 1)
	sm.OpenInventory()

	emitCombatStarted(t, bus)

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() after combat start on Inventory tab = %v, want Closed", got)
	}
	if sm.Visible() {
		t.Fatal("overlay must be gone immediately from the Inventory tab")
	}
	if sheet.forceCloseCalls != 1 || sheet.closeCalls != 0 {
		t.Fatalf("close calls = (close %d, forceClose %d), want (0, 1) — immediate path only",
			sheet.closeCalls, sheet.forceCloseCalls)
	}
}

// test_combat_forces_close_defers_level_up_modal: GIVEN the level-up modal
// is open when combat starts, THEN the modal closes with the UI and the
// level-up is deferred — it reappears when the UI opens again after combat
// (QA AC-16 edge).
func TestCombatForcesCloseDefersLevelUpModal(t *testing.T) {
	sm, bus, store, _ := newScreenMachine(t, 1)
	player := levelThreeCharacter(t, store)
	sm.OpenCharacter()
	emitLevelUp(t, bus, player, 4)
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("State() after level-up = %v, want LevelUp", got)
	}

	emitCombatStarted(t, bus)

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() after combat start with modal open = %v, want Closed", got)
	}

	// Combat ends; the UI reopens; the deferred level-up is still pending.
	emitCombatEnded(t, bus)
	sm.ToggleCharacter()
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("State() after reopen = %v, want LevelUp (level-up deferred)", got)
	}
}

// ---------------------------------------------------------------------------
// AC-17: tab switching
// ---------------------------------------------------------------------------

// test_switch_tab_preserves_member_and_crossfades: GIVEN the UI is open on
// the Character tab viewing party member X, WHEN the player switches to the
// Inventory tab, THEN the UI switches with a 200ms crossfade and party
// member X stays selected.
func TestSwitchTabPreservesMemberAndCrossfades(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.SetMember(2)

	sm.SwitchTab(character_sheet.InventoryTab)

	if got := sm.State(); got != character_sheet.InventoryTab {
		t.Fatalf("State() after switch = %v, want InventoryTab", got)
	}
	if got := sm.LastMember(); got != 2 {
		t.Fatalf("LastMember() after switch = %d, want 2 (member preserved)", got)
	}
	if !sm.Transitioning() {
		t.Fatal("tab switch must start the 200ms crossfade")
	}

	sm.Update(0.2)
	if sm.Transitioning() {
		t.Fatal("crossfade must complete after 200ms")
	}
	if got := sm.State(); got != character_sheet.InventoryTab {
		t.Fatalf("State() after crossfade = %v, want InventoryTab", got)
	}
}

// test_tab_crossfade_duration_is_200ms: the crossfade duration is pinned —
// still transitioning at 199ms, settled by 201ms (TR-ui-004: 200-300ms; the
// GDD pins 200ms). The settle step advances 1ms past the nominal end to
// absorb floating-point accumulation in the exact-sum frame times (the
// production timer also snaps sub-nanosecond residuals to zero).
func TestTabCrossfadeDurationIs200ms(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.SwitchTab(character_sheet.InventoryTab)

	sm.Update(0.19)
	if !sm.Transitioning() {
		t.Fatal("crossfade must still be in flight at 199ms")
	}
	sm.Update(0.011) // 0.19 + 0.011 = 0.201 — beyond 200ms with float tolerance
	if sm.Transitioning() {
		t.Fatal("crossfade must settle by 201ms")
	}
}

// test_rapid_tab_switching_latest_target_wins: rapid tab switching must not
// stack or queue transitions — the latest target wins (QA edge).
func TestRapidTabSwitchingLatestTargetWins(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()

	sm.SwitchTab(character_sheet.InventoryTab)
	sm.Update(0.1)
	sm.SwitchTab(character_sheet.CharacterTab)
	sm.Update(0.1)
	sm.SwitchTab(character_sheet.InventoryTab)

	if got := sm.State(); got != character_sheet.InventoryTab {
		t.Fatalf("State() after rapid switches = %v, want InventoryTab (latest target)", got)
	}
	if !sm.Transitioning() {
		t.Fatal("the latest switch must restart the crossfade")
	}
	sm.Update(0.2)
	if sm.Transitioning() {
		t.Fatal("crossfade must settle after the final switch")
	}
}

// ---------------------------------------------------------------------------
// Edge cases: open/close/toggle
// ---------------------------------------------------------------------------

// test_double_open_is_noop: opening an already-open UI is a no-op —
// pressing C twice opens once and stays open.
func TestDoubleOpenIsNoOp(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.OpenCharacter()
	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("State() after double open = %v, want CharacterTab", got)
	}
	if !sm.Visible() {
		t.Fatal("UI must remain visible after double open")
	}

	sm.OpenInventory()
	sm.OpenInventory()
	if got := sm.State(); got != character_sheet.InventoryTab {
		t.Fatalf("State() after double inventory open = %v, want InventoryTab", got)
	}
}

// test_toggle_key_closes_on_same_tab: pressing the open-state key again
// closes the UI (toggle behavior) — I on the Inventory tab, C on the
// Character tab.
func TestToggleKeyClosesOnSameTab(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.ToggleCharacter()
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("C on Character tab: State() = %v, want Closed (toggle)", got)
	}

	sm.OpenInventory()
	sm.ToggleInventory()
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("I on Inventory tab: State() = %v, want Closed (toggle)", got)
	}
}

// test_toggle_switches_from_other_tab: pressing C on the Inventory tab
// switches back with the same crossfade and the same member — it does not
// close (QA AC-17 edge).
func TestToggleSwitchesFromOtherTab(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenInventory()
	sm.SetMember(7)

	sm.ToggleCharacter()

	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("C on Inventory tab: State() = %v, want CharacterTab", got)
	}
	if got := sm.LastMember(); got != 7 {
		t.Fatalf("member after C switch = %d, want 7", got)
	}
	if !sm.Transitioning() {
		t.Fatal("C on Inventory tab must crossfade, not animate a close")
	}

	sm.ToggleInventory()
	if got := sm.State(); got != character_sheet.InventoryTab {
		t.Fatalf("I on Character tab: State() = %v, want InventoryTab", got)
	}
	if got := sm.LastMember(); got != 7 {
		t.Fatalf("member after I switch = %d, want 7", got)
	}
}

// test_esc_closes_with_animated_exit: Esc (CloseReasonUserRequest) closes
// with the animated path — the state is Closed immediately (input routing)
// and the exit flag stays set until the overlay hides.
func TestEscClosesWithAnimatedExit(t *testing.T) {
	sm, _, _, sheet := newScreenMachine(t, 1)
	sm.OpenCharacter()

	sm.Close(character_sheet.CloseReasonUserRequest)

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() after Esc = %v, want Closed", got)
	}
	if !sm.Closing() {
		t.Fatal("user close must start the animated exit")
	}
	if sheet.closeCalls != 1 || sheet.forceCloseCalls != 0 {
		t.Fatalf("close calls = (close %d, forceClose %d), want (1, 0) — animated path only",
			sheet.closeCalls, sheet.forceCloseCalls)
	}
	// The stub sheet hides instantly; the closing flag clears on the next
	// update once the sheet is hidden.
	sm.Update(0.25)
	if sm.Closing() {
		t.Fatal("closing must clear once the exit completed")
	}
	if sm.Visible() {
		t.Fatal("UI must not be visible after the exit completed")
	}
}

// test_close_button_closes: the close button (Story 008 markup) routes
// through Close(UserRequest) — the same animated path as Esc, from either
// tab.
func TestCloseButtonCloses(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.Close(character_sheet.CloseReasonUserRequest)
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("close button on Character tab: State() = %v, want Closed", got)
	}

	sm.OpenInventory()
	sm.Close(character_sheet.CloseReasonUserRequest)
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("close button on Inventory tab: State() = %v, want Closed", got)
	}
}

// test_forced_close_vs_user_close: the user path runs the animated exit;
// the forced path hides the overlay in the same frame with no animation and
// no prompt — even when it interrupts a user close in flight.
func TestForcedCloseVsUserClose(t *testing.T) {
	sm, bus, _, sheet := newScreenMachine(t, 1)
	sm.OpenCharacter()
	sm.Close(character_sheet.CloseReasonUserRequest)
	if !sm.Closing() {
		t.Fatal("user close must set the animated-exit flag")
	}
	if sheet.closeCalls != 1 {
		t.Fatalf("user close: close calls = %d, want 1", sheet.closeCalls)
	}

	// Reopen, then force close via combat while the UI is open.
	sm.ToggleCharacter()
	if !sm.Visible() {
		t.Fatal("UI must be open again before the forced close")
	}
	emitCombatStarted(t, bus)
	if sm.Closing() {
		t.Fatal("forced close must clear the animated-exit flag")
	}
	if sm.Visible() {
		t.Fatal("forced close must hide the overlay immediately")
	}
	if sheet.closeCalls != 1 || sheet.forceCloseCalls != 1 {
		t.Fatalf("close calls = (close %d, forceClose %d), want (1, 1)",
			sheet.closeCalls, sheet.forceCloseCalls)
	}
}

// test_input_blocked_while_level_up_modal_active: Esc and C/I are ignored
// while the level-up modal is active — only Confirm dismisses it (AC-18).
func TestInputBlockedWhileLevelUpModalActive(t *testing.T) {
	sm, bus, store, _ := newScreenMachine(t, 1)
	player := levelThreeCharacter(t, store)
	sm.OpenCharacter()
	emitLevelUp(t, bus, player, 4)
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("State() = %v, want LevelUp", got)
	}

	sm.Close(character_sheet.CloseReasonUserRequest)
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("Esc with modal active: State() = %v, want LevelUp (AC-18)", got)
	}
	sm.ToggleCharacter()
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("C with modal active: State() = %v, want LevelUp (AC-18)", got)
	}
}

// test_level_up_event_shows_modal: EventStatLevelUp shows the modal and the
// UI reports the LevelUp screen state (Story 002 forward contract: the
// EventStatLevelUp subscriber lives in Story 007).
func TestLevelUpEventShowsModal(t *testing.T) {
	sm, bus, store, _ := newScreenMachine(t, 1)
	player := levelThreeCharacter(t, store)
	sm.OpenCharacter()

	emitLevelUp(t, bus, player, 4)

	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("State() after level-up = %v, want LevelUp", got)
	}
}

// test_level_up_while_closed_is_deferred: a level-up that arrives while the
// UI is closed stays queued and the UI still reports Closed; the modal
// reappears on the next open.
func TestLevelUpWhileClosedIsDeferred(t *testing.T) {
	sm, bus, store, _ := newScreenMachine(t, 1)
	player := levelThreeCharacter(t, store)

	emitLevelUp(t, bus, player, 4)

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() with UI closed = %v, want Closed (level-up deferred)", got)
	}
	sm.OpenCharacter()
	if got := sm.State(); got != character_sheet.LevelUp {
		t.Fatalf("State() after open = %v, want LevelUp (deferred modal reappears)", got)
	}
}

// test_open_uses_last_viewed_member: a new session (fresh machine) opens to
// the player character; after the player views another member, close/open
// cycles return to that member (session-only state, GDD).
func TestOpenUsesLastViewedMember(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	if got := sm.LastMember(); got != 1 {
		t.Fatalf("LastMember() on fresh session = %d, want player 1", got)
	}
	sm.SetMember(5)
	sm.Close(character_sheet.CloseReasonUserRequest)
	sm.OpenInventory()
	if got := sm.LastMember(); got != 5 {
		t.Fatalf("LastMember() after reopen = %d, want 5", got)
	}
}

// test_switch_tab_noop_cases: switching on a closed UI, on the same tab, or
// to a non-tab target is a no-op.
func TestSwitchTabNoOpCases(t *testing.T) {
	sm, _, _, _ := newScreenMachine(t, 1)
	sm.SwitchTab(character_sheet.InventoryTab) // closed: no-op
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("SwitchTab while closed: State() = %v, want Closed", got)
	}

	sm.OpenCharacter()
	sm.SwitchTab(character_sheet.CharacterTab) // same tab: no-op
	if sm.Transitioning() {
		t.Fatal("same-tab switch must not start a crossfade")
	}
	sm.SwitchTab(character_sheet.LevelUp) // non-tab target: no-op
	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("SwitchTab(LevelUp): State() = %v, want CharacterTab", got)
	}
}

// A failed open (the member no longer has a Stats component in the real
// game) must leave the UI closed — the machine does not claim a state the
// sheet cannot render.
func TestOpenFailureLeavesUIClosed(t *testing.T) {
	store, _ := newTestStore()
	bus := entity.NewEventBus()
	sheet := &stubSheet{openErr: errors.New("character_sheet: entity has no Stats component")}
	sm := character_sheet.NewScreenStateMachine(sheet, character_sheet.NewLevelUpModal(nil, store, nil), store, bus, 1, nil)

	sm.ToggleCharacter()

	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() after failed open = %v, want Closed", got)
	}
	if sm.Visible() {
		t.Fatal("UI must not be visible after a failed open")
	}
}

// ---------------------------------------------------------------------------
// Level-up derivation helpers
// ---------------------------------------------------------------------------

// test_level_up_hp_gain_derivation: max(1, CON_mod) per level gained from
// the pre-award component (ADR-0006 minimum +1 rule) — CON 14 (+2) gains
// 2/level, CON 6 (−2) gains the minimum 1/level.
func TestLevelUpHpGainDerivation(t *testing.T) {
	con14 := stats.StatsComponent{Attributes: [stats.NumAttributes]int{10, 10, 14, 10, 10, 10}, Level: 3}
	if got := character_sheet.LevelUpHpGain(con14, 4); got != 2 {
		t.Errorf("LevelUpHpGain(CON 14, 3→4) = %d, want 2", got)
	}
	if got := character_sheet.LevelUpHpGain(con14, 5); got != 4 {
		t.Errorf("LevelUpHpGain(CON 14, 3→5) = %d, want 4 (two levels)", got)
	}
	con6 := stats.StatsComponent{Attributes: [stats.NumAttributes]int{10, 10, 6, 10, 10, 10}, Level: 3}
	if got := character_sheet.LevelUpHpGain(con6, 4); got != 1 {
		t.Errorf("LevelUpHpGain(CON 6, 3→4) = %d, want 1 (minimum +1)", got)
	}
}

// test_level_up_milestone_crossing: the level range (prev, new] grants a
// point only when it crosses 4, 8, 12, or 16 — 3→4 yes, 4→5 no (the
// level-4 milestone was the previous crossing, story-002 QA), 3→5 yes
// (multi-level award crosses 4), 19→20 no (cap excluded).
func TestLevelUpMilestoneCrossing(t *testing.T) {
	tests := []struct {
		prev, new int
		want      bool
	}{
		{3, 4, true},
		{4, 5, false},  // not a milestone: 4→5 doesn't grant (story-002 QA)
		{3, 5, true},   // multi-level award crosses 4
		{11, 12, true},
		{16, 17, false},
		{19, 20, false}, // cap level excluded
	}
	for _, tt := range tests {
		if got := character_sheet.LevelUpGrantsAttributePoint(tt.prev, tt.new); got != tt.want {
			t.Errorf("LevelUpGrantsAttributePoint(%d, %d) = %v, want %v", tt.prev, tt.new, got, tt.want)
		}
	}
}

// test_level_up_malformed_payload_ignored: a malformed level-up payload
// must not panic or corrupt state.
func TestLevelUpMalformedPayloadIgnored(t *testing.T) {
	sm, bus, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	bus.Emit(entity.Event{Type: stats.EventStatLevelUp, Source: "test", Data: "nope"})
	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("State() after malformed level-up = %v, want CharacterTab", got)
	}
}

// test_combat_started_ignores_payload: the state machine does not consume
// the combat payload — even a malformed EventCombatStarted must set the
// flag and force close (the flag semantics are load-bearing, AC-15/16).
func TestCombatStartedIgnoresPayload(t *testing.T) {
	sm, bus, _, _ := newScreenMachine(t, 1)
	sm.OpenCharacter()
	bus.Emit(entity.Event{Type: combat.EventCombatStarted, Source: "test", Data: 42})
	if !sm.CombatActive() {
		t.Fatal("combat flag must be set regardless of payload")
	}
	if got := sm.State(); got != character_sheet.Closed {
		t.Fatalf("State() = %v, want Closed", got)
	}
}

// test_teardown_unsubscribes: events emitted after Teardown must not reach
// the machine (subscriptions never outlive their owner); Teardown is
// idempotent.
func TestTeardownUnsubscribes(t *testing.T) {
	sm, bus, _, _ := newScreenMachine(t, 1)
	sm.Teardown()

	sm.OpenCharacter()
	emitCombatStarted(t, bus)
	if sm.CombatActive() {
		t.Fatal("combat event must not reach the machine after Teardown")
	}
	if got := sm.State(); got != character_sheet.CharacterTab {
		t.Fatalf("State() = %v, want CharacterTab (UI untouched by stale events)", got)
	}

	sm.Teardown() // idempotent
}

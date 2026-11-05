// Integration tests for Story 007: Keybindings & Input Remapping.
//
// Evidence path: production/epics/main-menu-settings/
// story-007-keybindings-input-remapping.md
// Required evidence: this file must exist and pass (Integration story gate —
// BLOCKING for /story-done).
//
// These tests drive the pure-Go core of src/ui/mainmenu headlessly with an
// injectable clock (fakeClock) and stubbed settings seams — no file I/O, no
// engine host, no external APIs. The persistence contract (GetSetting /
// WriteSettings / settings.cfg existence) is injected as functional options,
// exactly as the game layer wires it. Timing assertions follow the GDD
// measurement protocol: exact and tolerance-free at the unit level.
//
// Acceptance criteria map to test functions 1:1:
//
//	AC-1  test_actions_displayed                  -> TestKeybindingsScreenDisplaysAllActionsWithPrimaryAndSecondary
//	AC-1  test_click_slot_captures_next_key       -> TestClickBindingSlotEntersCaptureAndBindsNextKey
//	AC-1  test_non_printable_key_binds            -> TestCaptureBindsNonPrintableFunctionKey
//	AC-1  test_escape_cancels_capture             -> TestEscapeDuringCaptureCancelsBindingUnchanged
//	AC-1  test_modifier_plus_key                  -> TestModifierPlusKeyStoredAsChord
//	AC-1  test_modifier_alone_waits               -> TestModifierAloneKeepsCaptureActive
//	AC-3  test_multiple_modifiers                 -> TestMultipleModifiersStoredInBitmask
//	AC-2  test_conflict_auto_unbind               -> TestConflictUnbindsOldActionWithWarning
//	AC-2  test_conflict_victim_secondary          -> TestConflictUnbindsVictimSecondarySlot
//	AC-2  test_same_action_no_conflict            -> TestSameActionDifferentSlotNoConflict
//	AC-2  test_cascade_stops_at_one_hop           -> TestCascadeStopsAtOneHop
//	AC-2  test_capture_clears_victim_both_slots   -> TestCaptureClearsVictimBothSlots
//	AC-2  test_restore_conflict_resolves          -> TestRestoreResolvesConflictWithWarningAndChangeNotification
//	AC-2  test_restore_clears_victim_both_slots   -> TestRestoreClearsVictimBothSlots
//	AC-6  test_restore_last_reserved_blocked      -> TestRestoreBlockedWhenDefaultHeldByLastReservedBinding
//	AC-4  test_restore_per_row                    -> TestRestoreDefaultsPerRowNoConfirmation
//	AC-4  test_restore_all_confirm                -> TestRestoreAllDefaultsRequiresConfirmation
//	AC-4  test_restore_all_fixpoint               -> TestRestoreAllFixpointRestoresBlockedDefaults
//	AC-4  test_restore_all_dialog_focus           -> TestRestoreAllDialogOpensOnConfirmFocus
//	AC-5  test_reserved_key_warning_rebind        -> TestReservedKeyWarningOnRebindBindAnyway
//	AC-5  test_reserved_key_warning_cancel        -> TestF9ReservedWarningCancelLeavesBindingUnchanged
//	AC-5  test_reserved_dialog_keyboard           -> TestReservedDialogKeyboardNavigation
//	AC-5  test_non_reserved_no_warning            -> TestNonReservedKeyCompletesWithoutWarning
//	AC-6  test_last_reserved_binding_blocked      -> TestLastReservedBindingBlockedWithExactWarning
//	AC-6  test_two_bindings_unbind_ok             -> TestReservedFunctionWithTwoBindingsAllowsUnbind
//	AC-7  test_preset_list_and_select             -> TestGamepadPresetsListedAndSelectable
//	AC-7/8 test_preset_persists                   -> TestGamepadPresetPersistsAcrossReload
//	AC-8  test_round_trip                         -> TestBindingsPersistAndReloadRoundTrip
//	AC-8  test_missing_settings_defaults          -> TestMissingSettingsLoadsDefaults
//	AC-8  test_invalid_binding_fallback           -> TestInvalidBindingDataFallsBackToDefaults
//	AC-8  test_cleared_slot_round_trip            -> TestClearedSlotStaysUnboundAcrossReload
//	inte test_settings_nav_to_keybindings         -> TestSettingsControlsKeybindingsRowOpensKeybindingScreen
package main_menu_test

import (
	"fmt"
	"testing"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
//
// fakeClock, stubScreen, navigate, click and stubSettings are shared with
// save_load_screens_test.go and settings_screen_test.go (same package
// main_menu_test) — see those files.
// ---------------------------------------------------------------------------

// kbChange is one recorded binding-change notification.
type kbChange struct {
	action string
	slot   mainmenu.BindingSlot
	chord  mainmenu.KeyChord
}

// kbFixture bundles the pieces the keybinding tests drive.
type kbFixture struct {
	flow   *mainmenu.ScreenFlow
	clk    *fakeClock
	screen *mainmenu.KeybindingsScreen
	stub   *stubSettings

	changes []kbChange
	presets []string
}

// newKBFixture builds a flow in the title state with stub screens, a real
// KeybindingsScreen wired to the stub settings contract, and recording
// change/preset seams.
func newKBFixture(t *testing.T) *kbFixture {
	t.Helper()
	fx, err := buildKBFixture()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	return fx
}

func buildKBFixture() (*kbFixture, error) {
	clk := &fakeClock{}
	stub := &stubSettings{hasFile: true, store: map[string]string{}}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	for _, sc := range []mainmenu.Screen{mainmenu.ScreenTitleScreen, mainmenu.ScreenInGame, mainmenu.ScreenPauseMenu, mainmenu.ScreenSettings} {
		if err := flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
			return nil, fmt.Errorf("register %s: %w", sc, err)
		}
	}
	fx := &kbFixture{flow: flow, clk: clk, stub: stub}
	screen := mainmenu.NewKeybindingsScreen(flow,
		mainmenu.WithKeybindingsClock(clk.now),
		mainmenu.WithKeybindingsReader(stub.get),
		mainmenu.WithKeybindingsWriter(stub.write),
		mainmenu.WithKeybindingsHasSettingsFile(func() bool { return stub.hasFile }),
		mainmenu.WithKeybindingsLogger(stub.log),
		mainmenu.WithBindingChange(func(action string, slot mainmenu.BindingSlot, chord mainmenu.KeyChord) {
			fx.changes = append(fx.changes, kbChange{action: action, slot: slot, chord: chord})
		}),
		mainmenu.WithGamepadPresetChange(func(key string) {
			fx.presets = append(fx.presets, key)
		}),
	)
	if err := flow.RegisterController(mainmenu.ScreenKeybindings, screen); err != nil {
		return nil, fmt.Errorf("register keybindings screen: %w", err)
	}
	fx.screen = screen
	return fx, nil
}

// navigateToKeybindings reaches the Keybindings screen the way the game does
// (the ONLY path per the GDD transition table): Settings -> Keybindings.
func navigateToKeybindings(t *testing.T, fx *kbFixture) {
	t.Helper()
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenKeybindings)
}

// pressKey sends a raw key event to the screen (the capture input path the
// game layer uses while CapturesInput is true).
func pressKey(t *testing.T, fx *kbFixture, e mainmenu.KeyEvent) {
	t.Helper()
	if !fx.screen.HandleKey(e) {
		t.Fatalf("key %+v not consumed (capture inactive?)", e)
	}
}

// startCapture clicks a slot id and asserts capture mode for the target.
func startCapture(t *testing.T, fx *kbFixture, slotID string, action mainmenu.ActionBinding, slot mainmenu.BindingSlot) {
	t.Helper()
	click(t, fx.flow, slotID)
	if fx.screen.State() != mainmenu.KeybindingsStateCapture {
		t.Fatalf("state = %s, want Capture", fx.screen.State())
	}
	if fx.screen.CaptureAction() != action || fx.screen.CaptureSlot() != slot {
		t.Fatalf("capture target = %s/%s, want %s/%s", fx.screen.CaptureAction(), fx.screen.CaptureSlot(), action, slot)
	}
}

// flushBindings advances the clock past the debounce and runs the screen
// update so the pending settings.cfg write fires (AC-8).
func flushBindings(t *testing.T, fx *kbFixture) {
	t.Helper()
	fx.clk.advance(mainmenu.KeybindingsDebounceMs)
	fx.screen.Update()
	if fx.screen.Dirty() {
		t.Fatalf("bindings still dirty after flush: %v", fx.screen.DirtyKeys())
	}
}

// renderRow returns the render command of the action's row.
func renderRow(t *testing.T, fx *kbFixture, name mainmenu.ActionBinding) mainmenu.KeybindingsRenderCommand {
	t.Helper()
	var cmds []mainmenu.KeybindingsRenderCommand
	cmds = fx.screen.RenderCommands(cmds)
	for _, c := range cmds {
		if c.Type == mainmenu.RenderBindingRow && c.Action == name {
			return c
		}
	}
	t.Fatalf("no render row for %s", name)
	return mainmenu.KeybindingsRenderCommand{}
}

// assertNoSharedKeys verifies AC-2's global invariant: no key maps to more
// than one action at any time.
func assertNoSharedKeys(t *testing.T, fx *kbFixture) {
	t.Helper()
	seen := map[mainmenu.KeyChord]mainmenu.ActionBinding{}
	for _, a := range fx.screen.Actions() {
		name := a.RegistryName()
		for _, c := range fx.screen.Bindings(name) {
			if c.Key == "" {
				continue
			}
			if prev, ok := seen[c]; ok && prev != name {
				t.Fatalf("key %q maps to both %s and %s", mainmenu.SerializeChord(c), prev, name)
			}
			seen[c] = name
		}
	}
}

// mustChord asserts a slot's binding.
func mustChord(t *testing.T, fx *kbFixture, action mainmenu.ActionBinding, slot mainmenu.BindingSlot, want mainmenu.KeyChord) {
	t.Helper()
	list := fx.screen.Bindings(action)
	if int(slot) >= len(list) || list[slot] != want {
		t.Fatalf("%s slot %s = %v, want %v", action, slot, list, want)
	}
}

// ---------------------------------------------------------------------------
// AC-1: Action display and key capture
// ---------------------------------------------------------------------------

func TestKeybindingsScreenDisplaysAllActionsWithPrimaryAndSecondary(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	actions := fx.screen.Actions()
	if len(actions) != 14 {
		t.Fatalf("action count = %d, want 14", len(actions))
	}
	// Spot-check the display labels and defaults. The map is keyed by the
	// player-facing label: registry names are the stable snake_case IDs
	// ("move_forward"), except the reserved functions whose names ARE their
	// player-facing labels ("Pause Menu", "Quick Load" — AC-6 warnings).
	labels := map[string]string{}
	for _, a := range actions {
		labels[a.Label] = string(a.RegistryName())
	}
	for _, want := range []string{"Move Forward", "Move Backward", "Interact", "Journal", "Pause Menu", "Quick Load"} {
		if _, ok := labels[want]; !ok {
			t.Fatalf("action table missing %q (got %v)", want, labels)
		}
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotSecondary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionQuickLoad, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "F9"})

	// Rows render the chord labels.
	row := renderRow(t, fx, mainmenu.ActionBinding("move_forward"))
	if row.Primary != "W" || row.Secondary != mainmenu.LabelUnbound {
		t.Fatalf("move_forward row = %q / %q, want W / Unbound", row.Primary, row.Secondary)
	}
}

func TestClickBindingSlotEntersCaptureAndBindsNextKey(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)

	// Capture mode is visually distinct in the render stream: the slot shows
	// the prompt and the row carries the pulsing-capture flag.
	row := renderRow(t, fx, mainmenu.ActionBinding("move_forward"))
	if !row.Capturing || row.Primary != mainmenu.LabelPressKey {
		t.Fatalf("capture row = capturing:%v primary:%q, want capturing + %q", row.Capturing, row.Primary, mainmenu.LabelPressKey)
	}
	if !fx.screen.CapturesInput() {
		t.Fatal("screen must consume all raw input during capture")
	}
	if p := fx.screen.CapturePulse(); p < 0 || p >= 1 {
		t.Fatalf("CapturePulse = %v, want [0,1)", p)
	}

	// Press G: the binding completes and capture exits.
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("state = %s, want Panel after capture", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "G"})
	if fx.screen.CapturesInput() {
		t.Fatal("screen must release input after capture")
	}
	if len(fx.changes) != 1 || fx.changes[0].action != "move_forward" || fx.changes[0].chord != (mainmenu.KeyChord{Key: "G"}) {
		t.Fatalf("binding change notifications = %+v, want one move_forward/G", fx.changes)
	}

	// AC-8: the binding is written to settings.cfg under the dotted key.
	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "G" {
		t.Fatalf("persisted primary = %q, want %q", got, "G")
	}
}

func TestCaptureBindsNonPrintableFunctionKey(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "F5"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "F5"})
}

func TestEscapeDuringCaptureCancelsBindingUnchanged(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	// Escape is the reserved pause key (AC-5), so capture routes it through
	// the warning dialog; its Cancel path is the AC-1 capture cancel.
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	if fx.screen.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn for the reserved pause key", fx.screen.State())
	}
	click(t, fx.flow, mainmenu.FocusReservedCancel)

	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("state = %s, want Panel after cancel", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	if fx.screen.Dirty() {
		t.Fatalf("cancel must not mark anything dirty, got %v", fx.screen.DirtyKeys())
	}
	if len(fx.changes) != 0 {
		t.Fatalf("cancel must not notify binding changes, got %+v", fx.changes)
	}
}

// ---------------------------------------------------------------------------
// AC-3: Modifier keys
// ---------------------------------------------------------------------------

func TestModifierPlusKeyStoredAsChord(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	// Left Shift held, then W: the chord stores the modifier FLAG, not the
	// side (AC-3 edge case: right/left modifiers are identical).
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Shift", Shift: true})
	if fx.screen.State() != mainmenu.KeybindingsStateCapture {
		t.Fatalf("modifier alone must keep capture active, state = %s", fx.screen.State())
	}
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W", Shift: true})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W", Shift: true})

	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "Shift+W" {
		t.Fatalf("persisted chord = %q, want %q", got, "Shift+W")
	}
}

func TestModifierAloneKeepsCaptureActive(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	// Left Shift pressed and released without another key: capture stays
	// active waiting for the base key (AC-3).
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Shift", Shift: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Shift", Shift: false})
	if fx.screen.State() != mainmenu.KeybindingsStateCapture {
		t.Fatalf("modifier alone must keep capture active, state = %s", fx.screen.State())
	}
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "G", Shift: true})
}

func TestMultipleModifiersStoredInBitmask(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Control", Ctrl: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Shift", Shift: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Alt", Alt: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "S"})

	want := mainmenu.KeyChord{Key: "S", Shift: true, Ctrl: true, Alt: true}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, want)

	// The persisted value round-trips through ParseChord.
	serialized := mainmenu.SerializeChord(want)
	if serialized != "Ctrl+Shift+Alt+S" {
		t.Fatalf("serialized = %q, want %q", serialized, "Ctrl+Shift+Alt+S")
	}
	if got, ok := mainmenu.ParseChord(serialized); !ok || got != want {
		t.Fatalf("ParseChord(%q) = %+v, %v; want %+v", serialized, got, ok, want)
	}
	// Order-lenient parse: a hand-edited settings.cfg still round-trips.
	if got, ok := mainmenu.ParseChord("Shift+Ctrl+S"); !ok || got != (mainmenu.KeyChord{Key: "S", Shift: true, Ctrl: true}) {
		t.Fatalf("ParseChord(Shift+Ctrl+S) = %+v, %v", got, ok)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Conflict resolution
// ---------------------------------------------------------------------------

func TestConflictUnbindsOldActionWithWarning(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Move Forward primary = W, Interact primary = E. Bind E to Move
	// Forward: Interact is unbound with the warning (AC-2).
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "E"})

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "E"})
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("interact")) {
		t.Fatal("victim action must carry the Unbound warning")
	}
	if fx.screen.IsWarned(mainmenu.ActionBinding("move_forward")) {
		t.Fatal("winning action must not be warned")
	}
	assertNoSharedKeys(t, fx)

	// Two cues on the victim row: the "Unbound" text AND the Warned flag
	// (the render pass draws the ⚠ icon + amber — never hue alone).
	row := renderRow(t, fx, mainmenu.ActionBinding("interact"))
	if !row.Warned || row.Primary != mainmenu.LabelUnbound {
		t.Fatalf("interact row = warned:%v primary:%q, want warned + %q", row.Warned, row.Primary, mainmenu.LabelUnbound)
	}

	// Both sides persist: the victim's slot clears (as the AC-8 "Unbound"
	// token), the winner takes the key.
	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.interact.primary"]; got != mainmenu.SettingUnboundValue {
		t.Fatalf("victim persisted = %q, want %q", got, mainmenu.SettingUnboundValue)
	}
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "E" {
		t.Fatalf("winner persisted = %q, want E", got)
	}
}

func TestConflictUnbindsVictimSecondarySlot(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Give Interact a secondary binding first.
	startCapture(t, fx, "kb-row-interact-secondary", mainmenu.ActionBinding("interact"), mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "R"})
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "R"})

	// Binding R to Move Forward unbounds the VICTIM'S SECONDARY slot only.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "R"})

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "R"})
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotSecondary, mainmenu.KeyChord{})
	// The victim's primary stays bound.
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "E"})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("interact")) {
		t.Fatal("victim must carry the Unbound warning")
	}
	assertNoSharedKeys(t, fx)
}

func TestSameActionDifferentSlotNoConflict(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Binding the action's own primary key to its secondary slot is NOT a
	// conflict (AC-2 edge case: same action, different slot).
	startCapture(t, fx, "kb-row-move-forward-secondary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "W"})
	if fx.screen.IsWarned(mainmenu.ActionBinding("move_forward")) {
		t.Fatal("same-action bind must not warn")
	}
	assertNoSharedKeys(t, fx)
}

func TestCascadeStopsAtOneHop(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// A steals E from B (Interact); C (Journal) is untouched — the cascade
	// stops at one hop (GDD edge case).
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "E"})

	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionBinding("journal"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "J"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "E"})

	warned := fx.screen.WarnedActions()
	if len(warned) != 1 || warned[0] != mainmenu.ActionBinding("interact") {
		t.Fatalf("warned actions = %v, want [interact] only", warned)
	}
	assertNoSharedKeys(t, fx)
}

// TestCaptureClearsVictimBothSlots pins the review regression: when the
// victim holds the captured chord in BOTH slots (a legal same-action
// double-bind), the capture must clear EVERY matching slot — clearing only
// the first would leave the key shared (AC-2 violated). The cascade still
// stops at one hop (only the one victim action is touched).
func TestCaptureClearsVictimBothSlots(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Attack double-binds W (primary + secondary, same action = legal).
	startCapture(t, fx, "kb-row-attack-primary", mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	startCapture(t, fx, "kb-row-attack-secondary", mainmenu.ActionBinding("attack"), mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "W"})

	// Capturing W for Move Forward must release BOTH of Attack's slots.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotSecondary, mainmenu.KeyChord{})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("attack")) {
		t.Fatal("victim must carry the Unbound warning")
	}
	// Both cleared slots are broadcast through changeFn.
	cleared := 0
	for _, c := range fx.changes {
		if c.action == "attack" && c.chord == (mainmenu.KeyChord{}) {
			cleared++
		}
	}
	if cleared != 2 {
		t.Fatalf("attack clear notifications = %d, want 2, got %+v", cleared, fx.changes)
	}
	assertNoSharedKeys(t, fx)
}

// ---------------------------------------------------------------------------
// AC-4: Default restoration
// ---------------------------------------------------------------------------

func TestRestoreDefaultsPerRowNoConfirmation(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Move Forward -> E (steals from Interact): Interact is warned-unbound.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "E"})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("interact")) {
		t.Fatal("precondition: interact warned")
	}

	// Per-row restore: no confirmation, the row's warning clears, and other
	// rows (Interact) are NOT auto-restored (AC-4 edge case).
	click(t, fx.flow, "kb-row-move-forward-restore")
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("per-row restore must not open a dialog, state = %s", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	if fx.screen.IsWarned(mainmenu.ActionBinding("move_forward")) {
		t.Fatal("restored row's warning must clear")
	}
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("interact")) {
		t.Fatal("unrelated row must keep its warning (not auto-restored)")
	}

	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "W" {
		t.Fatalf("persisted primary = %q, want W", got)
	}
}

func TestRestoreAllDefaultsRequiresConfirmation(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Scramble: Move Forward -> G, Journal -> X, preset -> Southpaw.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})
	startCapture(t, fx, "kb-row-journal-primary", mainmenu.ActionBinding("journal"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "X"})
	click(t, fx.flow, mainmenu.FocusGamepadPreset) // Default -> Southpaw
	if fx.screen.SelectedGamepadPreset().Key != "Southpaw" {
		t.Fatalf("preset = %s, want Southpaw", fx.screen.SelectedGamepadPreset().Key)
	}

	// Restore-all opens the confirmation dialog; Cancel changes nothing.
	click(t, fx.flow, mainmenu.FocusRestoreAll)
	if fx.screen.State() != mainmenu.KeybindingsStateRestoreAll {
		t.Fatalf("state = %s, want RestoreAll", fx.screen.State())
	}
	click(t, fx.flow, mainmenu.FocusRestoreAllCancel)
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "G"})
	if fx.screen.SelectedGamepadPreset().Key != "Southpaw" {
		t.Fatal("cancel must keep the preset")
	}

	// Confirmed: everything returns to defaults, warnings clear, preset
	// resets to Default (AC-4 QA).
	click(t, fx.flow, mainmenu.FocusRestoreAll)
	click(t, fx.flow, mainmenu.FocusRestoreAllConfirm)
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("journal"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "J"})
	if len(fx.screen.WarnedActions()) != 0 {
		t.Fatalf("warnings must clear, got %v", fx.screen.WarnedActions())
	}
	if fx.screen.SelectedGamepadPreset().Key != "Default" {
		t.Fatalf("preset = %s, want Default after restore-all", fx.screen.SelectedGamepadPreset().Key)
	}

	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "W" {
		t.Fatalf("persisted primary = %q, want W", got)
	}
	if got := fx.stub.store["controls.gamepad_preset"]; got != "Default" {
		t.Fatalf("persisted preset = %q, want Default", got)
	}
}

// TestRestoreResolvesConflictWithWarningAndChangeNotification pins the
// review regression: per-row restore must run the same conflict resolution
// as a capture (AC-2). When Move Forward's default W was stolen by Attack,
// restoring Move Forward reclaims W from Attack — Attack's slot clears, it
// carries the warning, and the clear is broadcast through changeFn (the game
// layer must see the unbind, exactly like finishCapture).
func TestRestoreResolvesConflictWithWarningAndChangeNotification(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Steal W from Move Forward onto Attack.
	startCapture(t, fx, "kb-row-attack-primary", mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{})

	click(t, fx.flow, "kb-row-move-forward-restore")

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("attack")) {
		t.Fatal("restore conflict victim must carry the Unbound warning")
	}
	assertNoSharedKeys(t, fx)
	// The victim's clear is broadcast through changeFn (capture parity).
	cleared := false
	for _, c := range fx.changes {
		if c.action == "attack" && c.slot == mainmenu.SlotPrimary && c.chord == (mainmenu.KeyChord{}) {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("no change notification for the cleared attack slot, got %+v", fx.changes)
	}
}

// TestRestoreClearsVictimBothSlots pins the review regression on the RESTORE
// path: attack=[W,W] (legal same-action double-bind) + per-row restore of
// Move Forward — the restore must clear BOTH of Attack's W slots. Clearing
// only the first (old slotOf behaviour) would leave W shared: move_forward
// =[W,zero] and attack=[zero,W] (AC-2 violated).
func TestRestoreClearsVictimBothSlots(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Steal W from Move Forward onto both of Attack's slots.
	startCapture(t, fx, "kb-row-attack-primary", mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	startCapture(t, fx, "kb-row-attack-secondary", mainmenu.ActionBinding("attack"), mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{})

	click(t, fx.flow, "kb-row-move-forward-restore")

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionBinding("attack"), mainmenu.SlotSecondary, mainmenu.KeyChord{})
	if !fx.screen.IsWarned(mainmenu.ActionBinding("attack")) {
		t.Fatal("restore conflict victim must carry the Unbound warning")
	}
	assertNoSharedKeys(t, fx)
}

// TestRestoreBlockedWhenDefaultHeldByLastReservedBinding pins the review
// regression: the restore path runs the SAME AC-6 gate as finishCapture.
// When the default key is held by a reserved function's ONLY remaining
// binding, the restore of that slot is blocked with the AC-6 banner and the
// slot keeps its current binding.
func TestRestoreBlockedWhenDefaultHeldByLastReservedBinding(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Pause Menu gets W as a secondary so Move Forward can take Escape
	// (AC-6 allows the steal while two bindings remain)...
	startCapture(t, fx, "kb-row-pause-menu-secondary", mainmenu.ActionPauseMenu, mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	click(t, fx.flow, mainmenu.FocusReservedBind)
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})

	// ...leaving Pause Menu with W as its ONLY binding.
	bindingCount := 0
	for _, c := range fx.screen.Bindings(mainmenu.ActionPauseMenu) {
		if c.Key != "" {
			bindingCount++
		}
	}
	if bindingCount != 1 {
		t.Fatalf("precondition: Pause Menu must hold exactly one binding, got %d", bindingCount)
	}

	// Restoring Move Forward wants W — held by Pause Menu's last binding.
	// The restore is blocked: AC-6 banner, Move Forward keeps Escape.
	// The blocked restore must add NO new dirty keys (earlier capture steps
	// in this test legitimately dirtied their slots).
	dirtyBefore := make([]string, len(fx.screen.DirtyKeys()))
	copy(dirtyBefore, fx.screen.DirtyKeys())
	click(t, fx.flow, "kb-row-move-forward-restore")
	if got := fx.screen.BlockedMessage(); got != "This is the only binding for Pause Menu. It cannot be removed." {
		t.Fatalf("blocked message = %q, want the AC-6 warning", got)
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "W"})
	if len(fx.screen.DirtyKeys()) != len(dirtyBefore) {
		t.Fatalf("blocked restore must not add dirty keys: %v -> %v", dirtyBefore, fx.screen.DirtyKeys())
	}
	assertNoSharedKeys(t, fx)
}

// TestRestoreAllFixpointRestoresBlockedDefaults is the discriminating
// restore-all regression (review residual 2): pause_menu=[zero,W] (its ONLY
// binding) + move_forward=[Escape,zero]. A single restore-all pass gates
// move_forward's restore (W is held by Pause Menu's last binding), then
// pause_menu's restore reclaims Escape — leaving move_forward=[zero,zero]
// with its default W never restored (AC-4 violated) and the blocked banner
// swallowed. Restore-all must iterate to a FIXPOINT: the example converges
// on the third pass to the full default table, zero warnings, and no stale
// blocked banner.
func TestRestoreAllFixpointRestoresBlockedDefaults(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Pause Menu gets W as a secondary (so the Escape steal is legal)...
	startCapture(t, fx, "kb-row-pause-menu-secondary", mainmenu.ActionPauseMenu, mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "W"})
	// ...then Move Forward takes Escape, leaving Pause Menu with W only.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	click(t, fx.flow, mainmenu.FocusReservedBind)
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "W"})

	click(t, fx.flow, mainmenu.FocusRestoreAll)
	click(t, fx.flow, mainmenu.FocusRestoreAllConfirm)

	// Fixpoint result: EVERY default restored, no shared keys, no warnings,
	// and the transient AC-6 block banner is gone.
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{})
	if got := fx.screen.WarnedActions(); len(got) != 0 {
		t.Fatalf("restore-all must clear every warning, got %v", got)
	}
	if got := fx.screen.BlockedMessage(); got != "" {
		t.Fatalf("blocked banner must clear once the fixpoint completes, got %q", got)
	}
	assertNoSharedKeys(t, fx)
}

// TestRestoreAllDialogOpensOnConfirmFocus pins review residual 3: the
// restore-all dialog must open focused on "Restore All", not "Cancel" —
// otherwise Enter cancels immediately. The focus is explicit, and Enter
// through the flow confirms the restore.
func TestRestoreAllDialogOpensOnConfirmFocus(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Scramble a binding so a confirm is observable.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})

	click(t, fx.flow, mainmenu.FocusRestoreAll)
	if fx.screen.State() != mainmenu.KeybindingsStateRestoreAll {
		t.Fatalf("state = %s, want RestoreAll", fx.screen.State())
	}
	if got := fx.screen.Focus().FocusedID(); got != mainmenu.FocusRestoreAllConfirm {
		t.Fatalf("dialog focus = %q, want %q (Enter must confirm, not cancel)", got, mainmenu.FocusRestoreAllConfirm)
	}

	// Enter through the flow activates the focused Confirm button.
	if !fx.flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Enter"})) {
		t.Fatal("flow must dispatch Enter to the dialog (Restore All)")
	}
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("state = %s, want Panel after confirm", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
}

// ---------------------------------------------------------------------------
// AC-5: Reserved key warnings
// ---------------------------------------------------------------------------

const reservedPauseWarning = "This key is reserved for Pause Menu. Rebinding may conflict with system behaviour."

func TestReservedKeyWarningOnRebindBindAnyway(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Pause Menu gets a secondary binding so the rebind leaves it functional
	// (AC-6: only the LAST binding is protected).
	startCapture(t, fx, "kb-row-pause-menu-secondary", mainmenu.ActionPauseMenu, mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Backspace"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "Backspace"})

	// Capturing Escape opens the AC-5 warning dialog.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	if fx.screen.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn", fx.screen.State())
	}
	if got := fx.screen.ReservedWarning(); got != reservedPauseWarning {
		t.Fatalf("warning = %q, want %q", got, reservedPauseWarning)
	}

	// Bind Anyway completes the binding; Escape no longer triggers pause.
	click(t, fx.flow, mainmenu.FocusReservedBind)
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "Backspace"})
	for _, c := range fx.screen.Bindings(mainmenu.ActionPauseMenu) {
		if c.Key == "Escape" {
			t.Fatal("Escape must no longer be bound to Pause Menu")
		}
	}
	if !fx.screen.IsWarned(mainmenu.ActionPauseMenu) {
		t.Fatal("Pause Menu's lost primary must carry the warning")
	}
	assertNoSharedKeys(t, fx)

	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "Escape" {
		t.Fatalf("persisted primary = %q, want Escape", got)
	}
}

func TestF9ReservedWarningCancelLeavesBindingUnchanged(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "F9"})
	if fx.screen.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn for F9", fx.screen.State())
	}
	if got := fx.screen.ReservedWarning(); got != "This key is reserved for Quick Load. Rebinding may conflict with system behaviour." {
		t.Fatalf("warning = %q, want the Quick Load warning", got)
	}
	click(t, fx.flow, mainmenu.FocusReservedCancel)

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionQuickLoad, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "F9"})
	if fx.screen.Dirty() {
		t.Fatalf("cancel must not mark anything dirty, got %v", fx.screen.DirtyKeys())
	}
}

// TestReservedDialogKeyboardNavigation pins the review regression
// (accessibility, BLOCKING): the AC-5 dialog must not be a keyboard
// dead-end. Escape cancels via HandleKey (consumed — it is the capture
// escape hatch); every other key returns false from HandleKey so the game
// layer dispatches it to the flow, where Enter activates the focused "Bind
// Anyway" button and Tab moves focus — the dialog is reachable with the
// keyboard alone.
func TestReservedDialogKeyboardNavigation(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Pause Menu needs a second binding so "Bind Anyway" passes the AC-6 gate.
	startCapture(t, fx, "kb-row-pause-menu-secondary", mainmenu.ActionPauseMenu, mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Backspace"})
	flushBindings(t, fx) // settle it so the cancel below can assert clean state

	// Capture Escape -> AC-5 warning dialog.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	if fx.screen.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn", fx.screen.State())
	}

	// Escape through HandleKey cancels the dialog (AC-1/AC-5): consumed by
	// the screen, binding unchanged, nothing dirty.
	if !fx.screen.HandleKey(mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatal("Escape during the dialog must be consumed by HandleKey")
	}
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("state = %s, want Panel after Escape cancel", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	if fx.screen.Dirty() {
		t.Fatalf("dialog cancel must not mark anything dirty, got %v", fx.screen.DirtyKeys())
	}

	// Re-open the dialog. Enter/Space/Tab are NOT consumed by HandleKey (no
	// keyboard dead-end) — the game layer dispatches them to the flow.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	if fx.screen.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn", fx.screen.State())
	}
	for _, k := range []string{"Enter", "Space", "Tab"} {
		if fx.screen.HandleKey(mainmenu.KeyEvent{Key: k}) {
			t.Fatalf("%s must NOT be consumed by HandleKey during the dialog", k)
		}
	}

	// Enter through the flow activates the focused dialog button ("Bind
	// Anyway" — setState refocuses the first enabled dialog element).
	if !fx.flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Enter"})) {
		t.Fatal("flow must dispatch Enter to the dialog (Bind Anyway)")
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "Backspace"})
	if !fx.screen.IsWarned(mainmenu.ActionPauseMenu) {
		t.Fatal("Pause Menu's lost primary must carry the warning")
	}
	assertNoSharedKeys(t, fx)
}

func TestNonReservedKeyCompletesWithoutWarning(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("non-reserved key must bind immediately, state = %s", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "G"})
}

// ---------------------------------------------------------------------------
// AC-6: Last reserved binding protection
// ---------------------------------------------------------------------------

func TestLastReservedBindingBlockedWithExactWarning(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Pause Menu holds ONLY Escape. Rebinding Escape to Move Forward would
	// strip the last pause binding — the operation is blocked (AC-6).
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	click(t, fx.flow, mainmenu.FocusReservedBind)

	if got := fx.screen.BlockedMessage(); got != "This is the only binding for Pause Menu. It cannot be removed." {
		t.Fatalf("blocked message = %q, want the AC-6 warning", got)
	}
	if fx.screen.State() != mainmenu.KeybindingsStatePanel {
		t.Fatalf("state = %s, want Panel after the block", fx.screen.State())
	}
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	if fx.screen.Dirty() {
		t.Fatalf("blocked operation must not mark anything dirty, got %v", fx.screen.DirtyKeys())
	}
	assertNoSharedKeys(t, fx)
}

func TestReservedFunctionWithTwoBindingsAllowsUnbind(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Two bindings: unbinding one succeeds — the warning only fires when the
	// last binding would go (AC-6 edge case).
	startCapture(t, fx, "kb-row-pause-menu-secondary", mainmenu.ActionPauseMenu, mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Backspace"})
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Escape"})
	click(t, fx.flow, mainmenu.FocusReservedBind)

	if got := fx.screen.BlockedMessage(); got != "" {
		t.Fatalf("unbind of a non-last reserved binding must succeed, blocked = %q", got)
	}
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "Backspace"})
	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	assertNoSharedKeys(t, fx)
}

// ---------------------------------------------------------------------------
// AC-7: Gamepad presets
// ---------------------------------------------------------------------------

func TestGamepadPresetsListedAndSelectable(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	presets := fx.screen.GamepadPresets()
	if len(presets) != 3 {
		t.Fatalf("preset count = %d, want 3", len(presets))
	}
	wantLabels := []string{"Default", "Southpaw", "Left-Handed"}
	for i, want := range wantLabels {
		if presets[i].Label != want {
			t.Fatalf("preset %d label = %q, want %q", i, presets[i].Label, want)
		}
	}
	// Presets carry per-action gamepad data for the Input Binding System.
	if presets[0].Bindings["move_forward"] == "" || presets[1].Bindings["move_forward"] == presets[0].Bindings["move_forward"] {
		t.Fatal("presets must carry distinct per-action button maps")
	}

	// The dropdown cycles and applies immediately (AC-7).
	click(t, fx.flow, mainmenu.FocusGamepadPreset)
	if fx.screen.SelectedGamepadPreset().Key != "Southpaw" {
		t.Fatalf("preset = %s, want Southpaw", fx.screen.SelectedGamepadPreset().Key)
	}
	if len(fx.presets) != 1 || fx.presets[0] != "Southpaw" {
		t.Fatalf("preset change notifications = %v, want [Southpaw]", fx.presets)
	}
	click(t, fx.flow, mainmenu.FocusGamepadPreset)
	if fx.screen.SelectedGamepadPreset().Key != "Left-Handed" {
		t.Fatalf("preset = %s, want Left-Handed", fx.screen.SelectedGamepadPreset().Key)
	}

	// Selection persists under controls.gamepad_preset.
	flushBindings(t, fx)
	if got := fx.stub.store["controls.gamepad_preset"]; got != "Left-Handed" {
		t.Fatalf("persisted preset = %q, want Left-Handed", got)
	}
}

func TestGamepadPresetPersistsAcrossReload(t *testing.T) {
	fx := newKBFixture(t)
	// The click routes through the flow to the ACTIVE screen's focus manager,
	// so the Keybindings screen must be current (the title stub has no
	// focusables).
	navigateToKeybindings(t, fx)
	click(t, fx.flow, mainmenu.FocusGamepadPreset) // -> Southpaw
	flushBindings(t, fx)

	// A fresh boot with the written settings.cfg restores the selection.
	reloaded := newKBFixture(t)
	reloaded.stub.store = fx.stub.store
	reloaded.screen.Load()
	if got := reloaded.screen.SelectedGamepadPreset().Key; got != "Southpaw" {
		t.Fatalf("reloaded preset = %s, want Southpaw", got)
	}
	if reloaded.screen.Dirty() {
		t.Fatal("pre-population must not dirty the screen")
	}
}

// ---------------------------------------------------------------------------
// AC-8: Persistence round-trip
// ---------------------------------------------------------------------------

func TestBindingsPersistAndReloadRoundTrip(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Modify 3 actions + 1 gamepad preset.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "G"})
	startCapture(t, fx, "kb-row-zoom-in-primary", mainmenu.ActionBinding("zoom_in"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Control", Ctrl: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Shift", Shift: true})
	pressKey(t, fx, mainmenu.KeyEvent{Key: "Z"})
	startCapture(t, fx, "kb-row-journal-secondary", mainmenu.ActionBinding("journal"), mainmenu.SlotSecondary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "X"})
	click(t, fx.flow, mainmenu.FocusGamepadPreset) // -> Southpaw
	flushBindings(t, fx)

	// The exact dotted-key contract.
	wantStore := map[string]string{
		"controls.bindings.move_forward.primary": "G",
		"controls.bindings.zoom_in.primary":      "Ctrl+Shift+Z",
		"controls.bindings.journal.secondary":    "X",
		"controls.gamepad_preset":                "Southpaw",
	}
	for k, v := range wantStore {
		if got := fx.stub.store[k]; got != v {
			t.Fatalf("persisted %s = %q, want %q", k, got, v)
		}
	}

	// A fresh boot reads the file back: all modified bindings + the preset
	// match the values saved before restart (AC-8).
	reloaded := newKBFixture(t)
	reloaded.stub.store = fx.stub.store
	reloaded.screen.Load()
	mustChord(t, reloaded, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "G"})
	mustChord(t, reloaded, mainmenu.ActionBinding("zoom_in"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Z", Shift: true, Ctrl: true})
	mustChord(t, reloaded, mainmenu.ActionBinding("journal"), mainmenu.SlotSecondary, mainmenu.KeyChord{Key: "X"})
	// Unmodified actions keep their defaults.
	mustChord(t, reloaded, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "E"})
	if got := reloaded.screen.SelectedGamepadPreset().Key; got != "Southpaw" {
		t.Fatalf("reloaded preset = %s, want Southpaw", got)
	}
}

// TestClearedSlotStaysUnboundAcrossReload pins the review regression
// (AC-8): a conflict-cleared slot persists as the SettingUnboundValue token
// ("Unbound") and a fresh boot with that settings.cfg must keep the slot
// unbound — the token must NOT resurrect the action's default binding.
func TestClearedSlotStaysUnboundAcrossReload(t *testing.T) {
	fx := newKBFixture(t)
	navigateToKeybindings(t, fx)

	// Conflict-unbind Interact (AC-2), then flush: the cleared slot persists
	// as the "Unbound" token, not an empty value.
	startCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	pressKey(t, fx, mainmenu.KeyEvent{Key: "E"})
	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.interact.primary"]; got != mainmenu.SettingUnboundValue {
		t.Fatalf("cleared slot persisted = %q, want %q", got, mainmenu.SettingUnboundValue)
	}

	// A fresh boot with that file must keep Interact unbound: ParseChord
	// maps the token onto the zero chord, so no default resurrection.
	reloaded := newKBFixture(t)
	reloaded.stub.store = fx.stub.store
	reloaded.screen.Load()
	mustChord(t, reloaded, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{})
	mustChord(t, reloaded, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "E"})
	if reloaded.screen.Dirty() {
		t.Fatalf("pre-population must not dirty the screen, got %v", reloaded.screen.DirtyKeys())
	}
	assertNoSharedKeys(t, reloaded)
}

func TestMissingSettingsLoadsDefaults(t *testing.T) {
	fx := newKBFixture(t)
	fx.stub.hasFile = false
	fx.screen.Load()

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	if !fx.screen.Dirty() {
		t.Fatal("first launch must mark every binding dirty so the flush creates the file")
	}
	// Every binding key is dirty in action-table order.
	keys := fx.screen.DirtyKeys()
	if len(keys) != len(fx.screen.Actions())*2+1 {
		t.Fatalf("dirty = %d keys, want %d", len(keys), len(fx.screen.Actions())*2+1)
	}
	if keys[0] != "controls.bindings.move_forward.primary" {
		t.Fatalf("first dirty key = %q, want move_forward.primary", keys[0])
	}
	if keys[len(keys)-1] != "controls.gamepad_preset" {
		t.Fatalf("last dirty key = %q, want controls.gamepad_preset", keys[len(keys)-1])
	}

	// The flush writes the file with the defaults.
	flushBindings(t, fx)
	if got := fx.stub.store["controls.bindings.move_forward.primary"]; got != "W" {
		t.Fatalf("persisted primary = %q, want W", got)
	}
}

func TestInvalidBindingDataFallsBackToDefaults(t *testing.T) {
	fx := newKBFixture(t)
	// settings.cfg has invalid binding data for two keys and a valid override
	// for a third (AC-8 edge case: invalid data -> defaults for the affected
	// keys; other categories/keys are unaffected).
	fx.stub.store["controls.bindings.move_forward.primary"] = "???"
	fx.stub.store["controls.bindings.journal.secondary"] = "Ctrl+"
	fx.stub.store["controls.bindings.interact.primary"] = "R"
	fx.screen.Load()

	mustChord(t, fx, mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "W"})
	mustChord(t, fx, mainmenu.ActionBinding("journal"), mainmenu.SlotSecondary, mainmenu.KeyChord{})
	mustChord(t, fx, mainmenu.ActionBinding("interact"), mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "R"})
	mustChord(t, fx, mainmenu.ActionPauseMenu, mainmenu.SlotPrimary, mainmenu.KeyChord{Key: "Escape"})
	if fx.screen.Dirty() {
		t.Fatalf("pre-population must not dirty the screen, got %v", fx.screen.DirtyKeys())
	}
}

// ---------------------------------------------------------------------------
// Integration: Settings Controls -> Keybindings navigation (CR-6 entry)
// ---------------------------------------------------------------------------

func TestSettingsControlsKeybindingsRowOpensKeybindingScreen(t *testing.T) {
	clk := &fakeClock{}
	stub := &stubSettings{hasFile: true, store: map[string]string{}}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	for _, sc := range []mainmenu.Screen{mainmenu.ScreenTitleScreen, mainmenu.ScreenInGame, mainmenu.ScreenPauseMenu} {
		if err := flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
			t.Fatalf("register %s: %v", sc, err)
		}
	}
	settings := mainmenu.NewSettingsScreen(flow,
		mainmenu.WithSettingsClock(clk.now),
		mainmenu.WithSettingsReader(stub.get),
		mainmenu.WithSettingsWriter(stub.write),
		mainmenu.WithHasSettingsFile(func() bool { return stub.hasFile }),
	)
	if err := flow.RegisterController(mainmenu.ScreenSettings, settings); err != nil {
		t.Fatalf("register settings: %v", err)
	}
	if err := flow.RegisterController(mainmenu.ScreenKeybindings, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatalf("register keybindings stub: %v", err)
	}

	navigate(t, flow, clk, mainmenu.ScreenSettings)
	settings.SetActiveCategory(mainmenu.CategoryControls)
	click(t, flow, "setting-controls-keybindings")

	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenKeybindings {
		t.Fatalf("current = %s, want Keybindings", flow.Current())
	}

	// Back returns to the Settings screen (entry point via the back stack).
	if err := flow.Back(); err != nil {
		t.Fatalf("back: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenSettings {
		t.Fatalf("current = %s, want Settings after back", flow.Current())
	}
}

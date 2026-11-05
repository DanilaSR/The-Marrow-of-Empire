// Unit tests for Story 001: Navigation Accessibility & Screen Transition
// Framework (Main Menu & Settings epic).
//
// Evidence path: production/epics/main-menu-settings/
// story-001-navigation-transition-framework.md
// Required evidence: this file must exist and pass (UI story gate).
//
// These tests drive the pure-Go core of src/ui/mainmenu headlessly with an
// injectable clock (fakeClock) — no file I/O, no engine host, no external
// APIs. Timing assertions follow the GDD measurement protocol:
// progress = clamp(t_game / duration, 0, 1), exact and tolerance-free at
// the unit level.
//
// Acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_tab_visits_all_five_elements_in_order        -> TestTabVisitsAllFiveElementsInOrder
//	AC-1  test_tab_wraps_from_last_to_first                 -> TestTabWrapsFromLastToFirst
//	AC-1  test_shift_tab_reverses_order                     -> TestShiftTabReversesOrder
//	AC-1  test_tab_skips_disabled_elements                  -> TestTabSkipsDisabledElements
//	AC-1  test_enter_activates_focused_element              -> TestEnterActivatesFocusedElement
//	AC-1  test_space_activates_focused_element              -> TestSpaceActivatesFocusedElement
//	AC-1  test_arrows_navigate_within_groups                -> TestArrowsNavigateWithinGroups
//	AC-1  test_escape_returns_to_previous_screen            -> TestEscapeReturnsToPreviousScreen
//	AC-2  test_gamepad_down_moves_focus_same_as_keyboard    -> TestGamepadDownMovesFocusSameAsKeyboardArrows
//	AC-2  test_gamepad_start_toggles_pause                  -> TestGamepadStartTogglesPause
//	AC-3  test_mouse_click_focuses_and_activates            -> TestMouseClickFocusesAndActivates
//	AC-3  test_mouse_wheel_scrolls_focused_element          -> TestMouseWheelScrollsFocusedElement
//	AC-4  test_focus_ring_visible_during_keyboard_navigation -> TestFocusRingVisibleDuringKeyboardNavigation
//	AC-4  test_focus_ring_hidden_during_mouse_navigation    -> TestFocusRingHiddenDuringMouseNavigation
//	AC-4  test_focus_ring_shown_on_first_keyboard_input     -> TestFocusRingShownOnFirstKeyboardInput
//	AC-4  test_focus_ring_updates_within_one_frame          -> TestFocusRingUpdatesWithinOneFrameOfFocusChange
//	AC-4  test_focus_ring_contrast_at_least_3_to_1          -> TestFocusRingContrastAtLeastThreeToOneOnAllBackgrounds
//	AC-4  test_focus_ring_width_2px_dpi_scaled              -> TestFocusRingWidthDpiScaledFrom2px
//	AC-5  test_focus_indicator_carries_non_color_cue        -> TestFocusIndicatorAlwaysCarriesNonColorCue
//	AC-6  test_unbind_last_reserved_binding_blocked         -> TestUnbindLastReservedBindingBlocked
//	AC-6  test_unbind_non_last_reserved_binding_succeeds    -> TestUnbindNonLastReservedBindingSucceeds
//	AC-6  test_f9_quick_load_last_binding_blocked           -> TestF9QuickLoadLastBindingBlocked
//	AC-6  test_unbind_non_reserved_action_succeeds          -> TestUnbindNonReservedActionSucceeds
//	AC-7  test_crossfade_progress_tracks_clock_exactly      -> TestCrossfadeProgressTracksClockExactly
//	AC-7  test_crossfade_completes_at_duration_boundary     -> TestCrossfadeCompletesAtDurationBoundary
//	AC-7  test_input_discarded_during_crossfade             -> TestInputDiscardedDuringCrossfade
//	AC-7  test_rapid_double_transition_ignored              -> TestRapidDoubleTransitionIgnored
//	AC-7  test_transition_completes_after_frame_spike       -> TestTransitionCompletesAfterFrameTimeSpike
//	AC-7  test_loading_uses_hard_cut_and_locks_input        -> TestLoadingUsesHardCutAndLocksInput
//	AC-8  test_settings_back_returns_to_title_entry_point   -> TestSettingsBackReturnsToTitleEntryPoint
//	AC-8  test_settings_back_returns_to_pause_entry_point   -> TestSettingsBackReturnsToPauseEntryPoint
//	AC-9  test_corrupted_save_transitions_to_title_with_error -> TestCorruptedSaveTransitionsToTitleWithError
//	AC-9  test_version_mismatch_load_fails_to_title         -> TestVersionMismatchLoadTransitionsToTitleWithError
//	AC-9  test_successful_load_transitions_to_in_game       -> TestSuccessfulLoadTransitionsToInGame
//	AC-10 test_in_game_to_title_requires_confirmation       -> TestInGameToTitleRequiresConfirmationWithUnsavedProgress
//	AC-10 test_in_game_to_title_direct_without_unsaved      -> TestInGameToTitleTransitionsDirectlyWithoutUnsavedProgress
//	AC-10 test_quit_confirmation_resets_after_returning     -> TestQuitConfirmationResetsAfterReturningToInGame
//	edge  test_screen_controllers_receive_enter_exit        -> TestScreenControllersReceiveEnterExitCallbacks
//	edge  test_cancel_quit_abandons_pending_quit            -> TestCancelQuitAbandonsPendingQuit
package main_menu_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// fakeClock is the injectable game-time clock (milliseconds). All timing
// assertions read from it, so they are deterministic and tolerance-free.
type fakeClock struct{ ms float64 }

func (c *fakeClock) now() float64 { return c.ms }

func (c *fakeClock) advance(ms float64) { c.ms += ms }

// stubScreen is a minimal ScreenController for headless flow tests.
type stubScreen struct {
	focus  *mainmenu.FocusManager
	enters []mainmenu.Screen
	exits  int
}

func (s *stubScreen) OnEnter(prev mainmenu.Screen) { s.enters = append(s.enters, prev) }
func (s *stubScreen) OnExit()                      { s.exits++ }
func (s *stubScreen) Focus() *mainmenu.FocusManager { return s.focus }

// stubLoader is the injected SaveLoader seam (AC-9): the flow never
// touches the filesystem.
type stubLoader struct{ err error }

func (s stubLoader) Load(string) error { return s.err }

// newTestFlow builds a flow with a fake clock and optional options.
func newTestFlow(t *testing.T, opts ...mainmenu.FlowOption) (*mainmenu.ScreenFlow, *fakeClock) {
	t.Helper()
	clk := &fakeClock{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts, opts...)
	return flow, clk
}

// registerTitleScreen plugs a stub title-screen controller with a fresh
// focus manager into the flow and returns the focus manager.
func registerTitleScreen(t *testing.T, flow *mainmenu.ScreenFlow) *mainmenu.FocusManager {
	t.Helper()
	fm := mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, &stubScreen{focus: fm}); err != nil {
		t.Fatalf("register title controller: %v", err)
	}
	return fm
}

// registerController plugs a stub controller for any screen.
func registerController(t *testing.T, flow *mainmenu.ScreenFlow, s mainmenu.Screen) *mainmenu.FocusManager {
	t.Helper()
	fm := mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))
	if err := flow.RegisterController(s, &stubScreen{focus: fm}); err != nil {
		t.Fatalf("register controller for %s: %v", s, err)
	}
	return fm
}

// registerFive registers the five AC-1 elements in order, with optional
// per-id activation handlers.
func registerFive(t *testing.T, fm *mainmenu.FocusManager, onActivate map[string]func()) {
	t.Helper()
	for _, id := range []string{"one", "two", "three", "four", "five"} {
		var opts []mainmenu.FocusOption
		if fn, ok := onActivate[id]; ok {
			opts = append(opts, mainmenu.WithActivate(fn))
		}
		if err := fm.Register(id, opts...); err != nil {
			t.Fatalf("register %q: %v", id, err)
		}
	}
}

// navigate transitions to target and completes the crossfade.
func navigate(t *testing.T, flow *mainmenu.ScreenFlow, clk *fakeClock, target mainmenu.Screen) {
	t.Helper()
	if err := flow.RequestTransition(target); err != nil {
		t.Fatalf("transition to %s: %v", target, err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != target {
		t.Fatalf("current = %s, want %s", flow.Current(), target)
	}
}

// ---------------------------------------------------------------------------
// AC-1: Keyboard navigation
// ---------------------------------------------------------------------------

func TestTabVisitsAllFiveElementsInOrder(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	for i, id := range []string{"one", "two", "three", "four", "five"} {
		if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"})) {
			t.Fatalf("tab %d not handled", i)
		}
		if got := fm.FocusedID(); got != id {
			t.Fatalf("tab %d: focus = %q, want %q", i, got, id)
		}
	}
}

func TestTabWrapsFromLastToFirst(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	for i := 0; i < 5; i++ {
		flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	}
	if got := fm.FocusedID(); got != "five" {
		t.Fatalf("after 5 tabs: focus = %q, want %q", got, "five")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.FocusedID(); got != "one" {
		t.Fatalf("6th tab must wrap to first: focus = %q, want %q", got, "one")
	}
}

func TestShiftTabReversesOrder(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	for i, id := range []string{"five", "four", "three", "two", "one"} {
		flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab", Shift: true}))
		if got := fm.FocusedID(); got != id {
			t.Fatalf("shift+tab %d: focus = %q, want %q", i, got, id)
		}
	}
	// Shift+Tab from the first wraps to the last.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab", Shift: true}))
	if got := fm.FocusedID(); got != "five" {
		t.Fatalf("shift+tab wrap: focus = %q, want %q", got, "five")
	}
}

func TestTabSkipsDisabledElements(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	if err := fm.SetEnabled("three", false); err != nil {
		t.Fatalf("disable three: %v", err)
	}
	for i, id := range []string{"one", "two", "four", "five"} {
		flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
		if got := fm.FocusedID(); got != id {
			t.Fatalf("tab %d: focus = %q, want %q", i, got, id)
		}
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.FocusedID(); got != "one" {
		t.Fatalf("tab past disabled wraps: focus = %q, want %q", got, "one")
	}
}

func TestEnterActivatesFocusedElement(t *testing.T) {
	activated := 0
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, map[string]func(){"two": func() { activated++ }})
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.FocusedID(); got != "two" {
		t.Fatalf("focus = %q, want %q", got, "two")
	}
	if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Enter"})) {
		t.Fatal("Enter not handled")
	}
	if activated != 1 {
		t.Fatalf("activation count = %d, want 1", activated)
	}
}

func TestSpaceActivatesFocusedElement(t *testing.T) {
	activated := 0
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, map[string]func(){"one": func() { activated++ }})
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Space"})) {
		t.Fatal("Space not handled")
	}
	if activated != 1 {
		t.Fatalf("activation count = %d, want 1", activated)
	}
}

func TestArrowsNavigateWithinGroups(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	for _, id := range []string{"a1", "a2", "b1", "b2"} {
		group := "cats"
		if strings.HasPrefix(id, "b") {
			group = "dogs"
		}
		if err := fm.Register(id, mainmenu.WithGroup(group)); err != nil {
			t.Fatalf("register %q: %v", id, err)
		}
	}
	// Tab walks the flattened order across groups.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.FocusedID(); got != "a1" {
		t.Fatalf("focus = %q, want %q", got, "a1")
	}
	// Down moves within the cats group and wraps inside it.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := fm.FocusedID(); got != "a2" {
		t.Fatalf("down: focus = %q, want %q", got, "a2")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := fm.FocusedID(); got != "a1" {
		t.Fatalf("down wrap: focus = %q, want %q", got, "a1")
	}
	// Up moves backward within the group.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowUp"}))
	if got := fm.FocusedID(); got != "a2" {
		t.Fatalf("up: focus = %q, want %q", got, "a2")
	}
	// Tab crosses into the dogs group; arrows stay within it.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.FocusedID(); got != "b1" {
		t.Fatalf("tab across groups: focus = %q, want %q", got, "b1")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := fm.FocusedID(); got != "b2" {
		t.Fatalf("down in dogs: focus = %q, want %q", got, "b2")
	}
}

func TestEscapeReturnsToPreviousScreen(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	registerController(t, flow, mainmenu.ScreenLoadGame)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"})) {
		t.Fatal("Escape not handled")
	}
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatal("screen must not change until the crossfade completes")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Gamepad navigation (same actions as keyboard)
// ---------------------------------------------------------------------------

func TestGamepadDownMovesFocusSameAsKeyboardArrows(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if !flow.Dispatch(mainmenu.TranslateGamepad(mainmenu.GamepadEvent{Button: "down"})) {
		t.Fatal("D-pad down not handled")
	}
	if got := fm.FocusedID(); got != "two" {
		t.Fatalf("d-pad down: focus = %q, want %q", got, "two")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := fm.FocusedID(); got != "three" {
		t.Fatalf("arrow down: focus = %q, want %q", got, "three")
	}
}

func TestGamepadStartTogglesPause(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	if !flow.Dispatch(mainmenu.TranslateGamepad(mainmenu.GamepadEvent{Button: "start"})) {
		t.Fatal("Start not handled")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenPauseMenu)
	}
	if !flow.Dispatch(mainmenu.TranslateGamepad(mainmenu.GamepadEvent{Button: "start"})) {
		t.Fatal("second Start not handled")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenInGame)
	}
}

// ---------------------------------------------------------------------------
// AC-3: Mouse navigation
// ---------------------------------------------------------------------------

func TestMouseClickFocusesAndActivates(t *testing.T) {
	activated := 0
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, map[string]func(){"four": func() { activated++ }})
	if !flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: "four"})) {
		t.Fatal("mouse click not handled")
	}
	if got := fm.FocusedID(); got != "four" {
		t.Fatalf("click: focus = %q, want %q", got, "four")
	}
	if activated != 1 {
		t.Fatalf("click activation count = %d, want 1", activated)
	}
	if fm.RingVisible() {
		t.Fatal("ring must be hidden during mouse navigation (AC-4)")
	}
}

func TestMouseWheelScrollsFocusedElement(t *testing.T) {
	var scrolled []int
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	if err := fm.Register("list", mainmenu.WithScroll(func(dir int) { scrolled = append(scrolled, dir) })); err != nil {
		t.Fatalf("register list: %v", err)
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if !flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{Scroll: -1})) {
		t.Fatal("wheel down not handled")
	}
	flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{Scroll: 1}))
	if len(scrolled) != 2 || scrolled[0] != -1 || scrolled[1] != 1 {
		t.Fatalf("scroll deltas = %v, want [-1 1]", scrolled)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Focus ring visibility, contrast, width
// ---------------------------------------------------------------------------

func TestFocusRingVisibleDuringKeyboardNavigation(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	if fm.RingVisible() {
		t.Fatal("ring must not be visible before any input")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if !fm.RingVisible() {
		t.Fatal("ring must be visible during keyboard navigation")
	}
	if got := fm.RingID(); got != "one" {
		t.Fatalf("ring on %q, want %q", got, "one")
	}
}

func TestFocusRingHiddenDuringMouseNavigation(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: "two"}))
	if fm.RingVisible() {
		t.Fatal("ring may be hidden during mouse navigation")
	}
	if got := fm.RingID(); got != "" {
		t.Fatalf("ring id = %q, want empty while hidden", got)
	}
}

func TestFocusRingShownOnFirstKeyboardInput(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: "two"}))
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if !fm.RingVisible() {
		t.Fatal("ring must reappear on first keyboard input after mouse navigation")
	}
	if got := fm.FocusedID(); got != "three" {
		t.Fatalf("focus = %q, want %q (tab after click on two)", got, "three")
	}
}

func TestFocusRingUpdatesWithinOneFrameOfFocusChange(t *testing.T) {
	flow, _ := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	// Each single Dispatch is one update tick: the ring must track the
	// new focus by the end of the same tick.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.RingID(); got != "one" {
		t.Fatalf("ring = %q, want %q", got, "one")
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := fm.RingID(); got != "two" {
		t.Fatalf("ring = %q, want %q", got, "two")
	}
}

func TestFocusRingContrastAtLeastThreeToOneOnAllBackgrounds(t *testing.T) {
	style := mainmenu.NewFocusStyle(1.0)
	for _, bg := range []mainmenu.Color{
		mainmenu.ColorParchment,
		mainmenu.ColorDarkStone,
	} {
		ring := style.FocusRingFor(bg)
		if ratio := mainmenu.ContrastRatio(ring, bg); ratio < mainmenu.MinFocusContrast {
			t.Errorf("ring %#v on background %#v: ratio %.2f, want >= %g",
				ring, bg, ratio, mainmenu.MinFocusContrast)
		}
	}
	// Explicit palette expectations (data-driven sanity checks).
	if got := style.FocusRingFor(mainmenu.ColorParchment); got != mainmenu.ColorInk {
		t.Errorf("ring on parchment = %#v, want ink %#v", got, mainmenu.ColorInk)
	}
	if got := style.FocusRingFor(mainmenu.ColorDarkStone); got != mainmenu.ColorParchment {
		t.Errorf("ring on dark stone = %#v, want parchment %#v", got, mainmenu.ColorParchment)
	}
}

func TestFocusRingWidthDpiScaledFrom2px(t *testing.T) {
	for _, tc := range []struct {
		height float64
		want   float64
	}{
		{720, 2.0},                    // minimum resolution: scaled width floors at 2px
		{1080, 2.0},                   // reference: exactly 2px
		{1440, 2.0 * 1440.0 / 1080.0}, // 1.33x DPI (exact float expression)
		{2160, 4.0},                   // 4K: 2x DPI
	} {
		style := mainmenu.NewFocusStyle(mainmenu.DPIScaleForHeight(tc.height))
		if got := style.RingWidth(); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("height %.0f: ring width = %g, want %g", tc.height, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-5: Color is never the sole focus indicator
// ---------------------------------------------------------------------------

func TestFocusIndicatorAlwaysCarriesNonColorCue(t *testing.T) {
	style := mainmenu.NewFocusStyle(1.0)
	fm := mainmenu.NewFocusManager(style)
	if err := fm.Register("light", mainmenu.WithBackground(mainmenu.ColorParchment)); err != nil {
		t.Fatalf("register light: %v", err)
	}
	if err := fm.Register("dark", mainmenu.WithBackground(mainmenu.ColorDarkStone)); err != nil {
		t.Fatalf("register dark: %v", err)
	}
	indLight, ok := fm.IndicatorFor("light")
	if !ok {
		t.Fatal("indicator for light not found")
	}
	indDark, ok := fm.IndicatorFor("dark")
	if !ok {
		t.Fatal("indicator for dark not found")
	}
	// The ring color must adapt per background, but the non-color shape
	// cue (outline width) must always be present and constant — focus
	// state is never communicated by color alone.
	if indLight.WidthPx < mainmenu.MinRingWidthPx || indDark.WidthPx < mainmenu.MinRingWidthPx {
		t.Errorf("non-color cue (width) missing: light %g, dark %g",
			indLight.WidthPx, indDark.WidthPx)
	}
	if indLight.WidthPx != indDark.WidthPx {
		t.Errorf("width must not vary with color: light %g, dark %g",
			indLight.WidthPx, indDark.WidthPx)
	}
	if indLight.Color == indDark.Color {
		t.Errorf("ring colors must adapt to background: light %#v, dark %#v",
			indLight.Color, indDark.Color)
	}
}

// ---------------------------------------------------------------------------
// AC-6: Reserved key protection
// ---------------------------------------------------------------------------

func TestUnbindLastReservedBindingBlocked(t *testing.T) {
	reg := mainmenu.NewDefaultBindingRegistry()
	err := reg.Unbind(mainmenu.ActionPauseMenu, mainmenu.KeyChord{Key: "Escape"})
	if err == nil {
		t.Fatal("unbinding the last Escape binding must be blocked")
	}
	want := "This is the only binding for Pause Menu. It cannot be removed."
	if err.Error() != want {
		t.Fatalf("warning = %q, want %q", err.Error(), want)
	}
	var lrbe mainmenu.LastReservedBindingError
	if !errors.As(err, &lrbe) {
		t.Fatalf("error type = %T, want LastReservedBindingError", err)
	}
	if lrbe.Action != mainmenu.ActionPauseMenu {
		t.Fatalf("error action = %q, want %q", lrbe.Action, mainmenu.ActionPauseMenu)
	}
	remaining := reg.BindingsFor(mainmenu.ActionPauseMenu)
	if len(remaining) != 1 || remaining[0].Key != "Escape" {
		t.Fatalf("binding must survive the blocked unbind, got %v", remaining)
	}
}

func TestUnbindNonLastReservedBindingSucceeds(t *testing.T) {
	reg := mainmenu.NewDefaultBindingRegistry()
	reg.Bind(mainmenu.ActionPauseMenu, mainmenu.KeyChord{Key: "F1"})
	if err := reg.Unbind(mainmenu.ActionPauseMenu, mainmenu.KeyChord{Key: "Escape"}); err != nil {
		t.Fatalf("unbinding a non-last binding must succeed, got %v", err)
	}
	remaining := reg.BindingsFor(mainmenu.ActionPauseMenu)
	if len(remaining) != 1 || remaining[0].Key != "F1" {
		t.Fatalf("remaining bindings = %v, want [F1]", remaining)
	}
	// The second (last) binding is still protected.
	if err := reg.Unbind(mainmenu.ActionPauseMenu, mainmenu.KeyChord{Key: "F1"}); err == nil {
		t.Fatal("the last remaining binding must still be blocked")
	}
}

func TestF9QuickLoadLastBindingBlocked(t *testing.T) {
	reg := mainmenu.NewDefaultBindingRegistry()
	err := reg.Unbind(mainmenu.ActionQuickLoad, mainmenu.KeyChord{Key: "F9"})
	if err == nil {
		t.Fatal("unbinding the last F9 binding must be blocked")
	}
	want := "This is the only binding for Quick Load. It cannot be removed."
	if err.Error() != want {
		t.Fatalf("warning = %q, want %q", err.Error(), want)
	}
}

func TestUnbindNonReservedActionSucceeds(t *testing.T) {
	reg := mainmenu.NewDefaultBindingRegistry()
	reg.Bind("Move", mainmenu.KeyChord{Key: "W"})
	if err := reg.Unbind("Move", mainmenu.KeyChord{Key: "W"}); err != nil {
		t.Fatalf("non-reserved actions unbind freely, got %v", err)
	}
	if got := reg.BindingsFor("Move"); len(got) != 0 {
		t.Fatalf("remaining bindings = %v, want none", got)
	}
}

// ---------------------------------------------------------------------------
// AC-7: Crossfade transitions
// ---------------------------------------------------------------------------

func TestCrossfadeProgressTracksClockExactly(t *testing.T) {
	clk := &fakeClock{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	ts.SetCrossfadeMs(250)
	if !ts.Begin(mainmenu.TransitionCrossfade) {
		t.Fatal("Begin failed")
	}
	if p := ts.Progress(); p != 0.0 {
		t.Fatalf("progress at t=0 = %g, want 0", p)
	}
	clk.advance(125)
	if p := ts.Progress(); p != 0.5 {
		t.Fatalf("progress at t=125 = %g, want 0.5", p)
	}
	clk.advance(125)
	if p := ts.Progress(); p != 1.0 {
		t.Fatalf("progress at t=250 = %g, want 1", p)
	}
	if !ts.Update() {
		t.Fatal("transition must complete exactly at 250ms")
	}
	if ts.Active() {
		t.Fatal("transition must be inactive after completion")
	}
}

func TestCrossfadeCompletesAtDurationBoundary(t *testing.T) {
	clk := &fakeClock{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	ts.SetCrossfadeMs(250)
	if !ts.Begin(mainmenu.TransitionCrossfade) {
		t.Fatal("Begin failed")
	}
	clk.advance(249)
	if ts.Update() {
		t.Fatal("transition must not complete at 249ms (inside 250ms ±10%)")
	}
	if !ts.Active() {
		t.Fatal("transition must still be active at 249ms")
	}
	clk.advance(1)
	if !ts.Update() {
		t.Fatal("transition must complete at exactly 250ms")
	}
}

func TestInputDiscardedDuringCrossfade(t *testing.T) {
	flow, clk := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	if err := flow.RequestTransition(mainmenu.ScreenLoadGame); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !flow.InputBlocked() {
		t.Fatal("input must be blocked during the crossfade")
	}
	if flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"})) {
		t.Fatal("input during crossfade must be discarded (prevents double-clicks across screens)")
	}
	if got := fm.FocusedID(); got != "" {
		t.Fatalf("focus must not move during crossfade, got %q", got)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenLoadGame)
	}
	if flow.InputBlocked() {
		t.Fatal("input must be unblocked after the crossfade")
	}
}

func TestRapidDoubleTransitionIgnored(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	if err := flow.RequestTransition(mainmenu.ScreenLoadGame); err != nil {
		t.Fatalf("first transition: %v", err)
	}
	err := flow.RequestTransition(mainmenu.ScreenSettings)
	if !errors.Is(err, mainmenu.ErrTransitionInProgress) {
		t.Fatalf("second transition error = %v, want ErrTransitionInProgress", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("double-click must not redirect the transition: current = %s, want %s",
			flow.Current(), mainmenu.ScreenLoadGame)
	}
}

func TestTransitionCompletesAfterFrameTimeSpike(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	if err := flow.RequestTransition(mainmenu.ScreenLoadGame); err != nil {
		t.Fatalf("transition: %v", err)
	}
	clk.advance(1000) // single giant frame — completion is clock-based
	flow.Update()
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenLoadGame)
	}
	if flow.InputBlocked() {
		t.Fatal("transition must not linger after a frame-time spike")
	}
}

func TestLoadingUsesHardCutAndLocksInput(t *testing.T) {
	flow, clk := newTestFlow(t)
	fm := registerTitleScreen(t, flow)
	registerFive(t, fm, nil)
	if err := flow.RequestTransition(mainmenu.ScreenLoading); err != nil {
		t.Fatalf("transition to Loading: %v", err)
	}
	if flow.Current() != mainmenu.ScreenLoading {
		t.Fatalf("current = %s, want %s (hard cut resolves immediately)",
			flow.Current(), mainmenu.ScreenLoading)
	}
	if clk.ms != 0 {
		t.Fatalf("hard cut must not advance the clock, clock = %g", clk.ms)
	}
	if !flow.InputBlocked() {
		t.Fatal("input must be locked while Loading")
	}
	if flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"})) {
		t.Fatal("input must be discarded while Loading")
	}
	flow.CompleteLoad(nil)
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenInGame)
	}
}

// ---------------------------------------------------------------------------
// AC-8: Settings remembers its entry point
// ---------------------------------------------------------------------------

func TestSettingsBackReturnsToTitleEntryPoint(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenSettings)
	if err := flow.Back(); err != nil {
		t.Fatalf("Back from Settings: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s (entered from Title)", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

func TestSettingsBackReturnsToPauseEntryPoint(t *testing.T) {
	flow, clk := newTestFlow(t)
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	navigate(t, flow, clk, mainmenu.ScreenPauseMenu)
	navigate(t, flow, clk, mainmenu.ScreenSettings)
	if err := flow.Back(); err != nil {
		t.Fatalf("Back from Settings: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want %s (entered from Pause)", flow.Current(), mainmenu.ScreenPauseMenu)
	}
}

// ---------------------------------------------------------------------------
// AC-9: Loading state error handling
// ---------------------------------------------------------------------------

func TestCorruptedSaveTransitionsToTitleWithError(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithSaveLoader(stubLoader{
		err: errors.New("save checksum mismatch: this save file is corrupted"),
	}))
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if err := flow.BeginLoad("slot1"); err != nil {
		t.Fatalf("BeginLoad: %v", err)
	}
	if flow.Current() != mainmenu.ScreenLoading {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenLoading)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
	if got := flow.LastError(); !strings.Contains(got, "corrupted") {
		t.Fatalf("error message = %q, want it to describe the corruption", got)
	}
}

func TestVersionMismatchLoadTransitionsToTitleWithError(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithSaveLoader(stubLoader{
		err: errors.New("save version 3 is not supported by this build (expected 2)"),
	}))
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if err := flow.BeginLoad("slot1"); err != nil {
		t.Fatalf("BeginLoad: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
	if got := flow.LastError(); !strings.Contains(got, "version") {
		t.Fatalf("error message = %q, want it to describe the version mismatch", got)
	}
}

func TestSuccessfulLoadTransitionsToInGame(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithSaveLoader(stubLoader{err: nil}))
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if err := flow.BeginLoad("slot1"); err != nil {
		t.Fatalf("BeginLoad: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenInGame)
	}
	if got := flow.LastError(); got != "" {
		t.Fatalf("error message = %q, want empty after a successful load", got)
	}
}

// ---------------------------------------------------------------------------
// AC-10: InGame → TitleScreen confirmation gate
// ---------------------------------------------------------------------------

func TestInGameToTitleRequiresConfirmationWithUnsavedProgress(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithUnsavedProgressCheck(func() bool { return true }))
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	err := flow.RequestTransition(mainmenu.ScreenTitleScreen)
	if !errors.Is(err, mainmenu.ErrUnsavedProgress) {
		t.Fatalf("error = %v, want ErrUnsavedProgress", err)
	}
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s before confirmation", flow.Current(), mainmenu.ScreenInGame)
	}
	if err := flow.ConfirmQuit(); err != nil {
		t.Fatalf("ConfirmQuit: %v", err)
	}
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatal("screen must not swap until the crossfade completes")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s after confirmed quit", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

func TestInGameToTitleTransitionsDirectlyWithoutUnsavedProgress(t *testing.T) {
	flow, clk := newTestFlow(t) // no probe installed: no unsaved progress
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	if err := flow.RequestTransition(mainmenu.ScreenTitleScreen); err != nil {
		t.Fatalf("transition without unsaved progress must be direct, got %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

func TestQuitConfirmationResetsAfterReturningToInGame(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithUnsavedProgressCheck(func() bool { return true }))
	registerTitleScreen(t, flow)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	if err := flow.RequestTransition(mainmenu.ScreenTitleScreen); !errors.Is(err, mainmenu.ErrUnsavedProgress) {
		t.Fatalf("first quit error = %v, want ErrUnsavedProgress", err)
	}
	if err := flow.ConfirmQuit(); err != nil {
		t.Fatalf("ConfirmQuit: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
	// Back in gameplay, the gate must demand confirmation again.
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	err := flow.RequestTransition(mainmenu.ScreenTitleScreen)
	if !errors.Is(err, mainmenu.ErrUnsavedProgress) {
		t.Fatalf("second quit error = %v, want ErrUnsavedProgress (confirmation must be re-required)", err)
	}
}

// ---------------------------------------------------------------------------
// Edge cases: lifecycle wiring and cancel-quit
// ---------------------------------------------------------------------------

func TestScreenControllersReceiveEnterExitCallbacks(t *testing.T) {
	flow, clk := newTestFlow(t)
	title := &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}
	load := &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, title); err != nil {
		t.Fatalf("register title: %v", err)
	}
	if err := flow.RegisterController(mainmenu.ScreenLoadGame, load); err != nil {
		t.Fatalf("register load: %v", err)
	}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if title.exits != 1 {
		t.Fatalf("title exits = %d, want 1", title.exits)
	}
	if len(load.enters) != 1 || load.enters[0] != mainmenu.ScreenTitleScreen {
		t.Fatalf("load enters = %v, want [TitleScreen]", load.enters)
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"}))
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if load.exits != 1 {
		t.Fatalf("load exits = %d, want 1", load.exits)
	}
	if len(title.enters) != 1 || title.enters[0] != mainmenu.ScreenLoadGame {
		t.Fatalf("title re-enters = %v, want [LoadGame]", title.enters)
	}
}

func TestCancelQuitAbandonsPendingQuit(t *testing.T) {
	flow, clk := newTestFlow(t, mainmenu.WithUnsavedProgressCheck(func() bool { return true }))
	registerTitleScreen(t, flow)
	ingame := registerController(t, flow, mainmenu.ScreenInGame)
	if err := ingame.Register("unit"); err != nil {
		t.Fatalf("register unit: %v", err)
	}
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	if err := flow.RequestTransition(mainmenu.ScreenTitleScreen); !errors.Is(err, mainmenu.ErrUnsavedProgress) {
		t.Fatalf("quit error = %v, want ErrUnsavedProgress", err)
	}
	flow.CancelQuit()
	if err := flow.ConfirmQuit(); !errors.Is(err, mainmenu.ErrNoPendingQuit) {
		t.Fatalf("ConfirmQuit after CancelQuit = %v, want ErrNoPendingQuit", err)
	}
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenInGame)
	}
	// The flow remains fully navigable after the abandoned quit.
	if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"})) {
		t.Fatal("navigation must keep working after cancel-quit")
	}
}

// Integration tests for Story 005: Settings Screen & Persistence.
//
// Evidence path: production/epics/main-menu-settings/
// story-005-settings-screen-persistence.md
// Required evidence: this file must exist and pass (Integration story gate).
//
// These tests drive the pure-Go core of src/ui/mainmenu headlessly with an
// injectable clock (fakeClock) and stubbed settings seams — no file I/O, no
// engine host, no external APIs. The persistence/application contract
// (ApplySetting/WriteSettings/HasSettings/GetSetting) is injected as
// functional options, exactly as the game layer wires it. Timing assertions
// follow the GDD measurement protocol: exact and tolerance-free at the unit
// level.
//
// Acceptance criteria map to test functions 1:1:
//
//	AC-1  test_six_categories_registered        -> TestSixCategoriesRegisteredWithExpectedSettings
//	AC-2  test_change_applies_instantly         -> TestChangeAppliesInstantlyAndDebouncedFlushWrites
//	AC-2  test_rapid_changes_debounce           -> TestRapidChangesDebounceToFinalValue
//	AC-2  test_clean_shutdown_flush             -> TestCleanShutdownFlushWritesLatestValues
//	AC-2  test_slider_clamps_to_step_grid       -> TestSliderValuesClampToStepGrid
//	AC-3  test_resolution_preview_overlay       -> TestResolutionChangeStartsPreviewAndKeepsScreenInteractive
//	AC-3  test_preview_keep                     -> TestPreviewKeepPersistsChanges
//	AC-3  test_preview_revert                   -> TestPreviewRevertRestoresPreviousValue
//	AC-3  test_preview_auto_revert              -> TestPreviewAutoRevertsAfter15Seconds
//	AC-3  test_preview_countdown                -> TestPreviewCountdownCountsDownAndClamps
//	AC-4  test_reset_category_confirmation      -> TestResetCategoryRequiresConfirmationAndResetsOnlyThatCategory
//	AC-4  test_reset_all_confirmation           -> TestResetAllRequiresConfirmationAndResetsEverything
//	AC-5  test_missing_settings_first_launch    -> TestMissingSettingsCreatesDefaultsAndFlagsFirstLaunch
//	AC-5  test_existing_settings_prepopulated   -> TestExistingSettingsPopulateValuesWithoutDirty
//	AC-5  test_corrupted_values_fallback        -> TestCorruptedStoreValuesFallBackToDefaults
//	AC-6  test_write_failure_logs_keeps_dirty   -> TestWriteFailureKeepsDirtyAndLogsNoToastOnFirstFailure
//	AC-6  test_second_failure_toast             -> TestSecondConsecutiveFailureShowsAutoDismissingToast
//	AC-6  test_success_clears_error_state       -> TestSuccessfulWriteClearsErrorStateAndDismissesToast
//	AC-6  test_retry_after_failures             -> TestFlushRetriesOnNextChangeAfterFailures
//	AC-7  test_deferred_greyed_not_focusable    -> TestDeferredControlsGreyedWithComingSoonAndRejectInteraction
//	AC-7  test_fps_cap_interactive              -> TestFPSCapIsInteractive
//	AC-8  test_pause_entry_restricts_graphics   -> TestPauseMenuEntryRestrictsGraphicsAbsentNotGreyed
//	AC-8  test_title_entry_full_graphics        -> TestTitleMenuEntryShowsFullGraphics
//	AC-9  test_ui_update_p95_under_2ms          -> TestSettingsScreenUpdateAndRenderP95Under2ms
//	AC-9  benchmark_render_commands             -> BenchmarkSettingsScreenUpdateAndRenderCommands
package main_menu_test

import (
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
//
// fakeClock, stubScreen, navigate, click and clickIgnored are shared with
// save_load_screens_test.go (same package main_menu_test) — see there.
// ---------------------------------------------------------------------------

// keyValue is one recorded apply/write call.
type keyValue struct{ key, value string }

// stubSettings is the stubbed settings contract: the store holds the
// "settings.cfg" contents the game layer adapter would read/write. Tests
// toggle hasFile and writeErr to simulate the AC-5/AC-6 scenarios.
type stubSettings struct {
	hasFile  bool
	store    map[string]string
	writeErr error

	applied []keyValue
	writes  []keyValue
	logs    []string
	reruns  int
}

func (st *stubSettings) apply(key, value string) {
	st.applied = append(st.applied, keyValue{key, value})
}

func (st *stubSettings) write(key, value string) error {
	if st.writeErr != nil {
		return st.writeErr
	}
	st.writes = append(st.writes, keyValue{key, value})
	st.store[key] = value
	return nil
}

func (st *stubSettings) get(key string) string { return st.store[key] }

func (st *stubSettings) log(format string, args ...any) {
	st.logs = append(st.logs, fmt.Sprintf(format, args...))
}

// settingsFixture bundles the pieces the tests drive.
type settingsFixture struct {
	flow   *mainmenu.ScreenFlow
	clk    *fakeClock
	screen *mainmenu.SettingsScreen
	stub   *stubSettings
}

// newSettingsFixture builds a flow in the title state with stub screens and
// a real SettingsScreen wired to the stub settings contract.
func newSettingsFixture(t *testing.T) *settingsFixture {
	t.Helper()
	fx, err := buildSettingsFixture()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	return fx
}

// buildSettingsFixture wires the flow and screen without a testing.T so the
// benchmark can reuse the same construction.
func buildSettingsFixture() (*settingsFixture, error) {
	clk := &fakeClock{}
	stub := &stubSettings{hasFile: true, store: map[string]string{}}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	for _, sc := range []mainmenu.Screen{mainmenu.ScreenTitleScreen, mainmenu.ScreenInGame, mainmenu.ScreenPauseMenu} {
		if err := flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
			return nil, fmt.Errorf("register %s: %w", sc, err)
		}
	}
	screen := mainmenu.NewSettingsScreen(flow,
		mainmenu.WithSettingsClock(clk.now),
		mainmenu.WithApplySetting(stub.apply),
		mainmenu.WithSettingsWriter(stub.write),
		mainmenu.WithHasSettingsFile(func() bool { return stub.hasFile }),
		mainmenu.WithSettingsReader(stub.get),
		mainmenu.WithSettingsLogger(stub.log),
		mainmenu.WithReRunAccessibility(func() { stub.reruns++ }),
	)
	if err := flow.RegisterController(mainmenu.ScreenSettings, screen); err != nil {
		return nil, fmt.Errorf("register settings screen: %w", err)
	}
	return &settingsFixture{flow: flow, clk: clk, screen: screen, stub: stub}, nil
}

// navigateToSettings reaches the Settings screen from the title the way the
// game does (entry point: title screen).
func navigateToSettings(t *testing.T, fx *settingsFixture) {
	t.Helper()
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)
}

// navigateToSettingsFromPause reaches Settings from the in-game pause menu
// (entry point: pause menu, AC-8).
func navigateToSettingsFromPause(t *testing.T, fx *settingsFixture) {
	t.Helper()
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenInGame)
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenPauseMenu)
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)
}

// advanceAndUpdate advances the clock and runs one Update frame — the game
// loop's per-frame cadence the debounce and preview timers live on.
func advanceAndUpdate(t *testing.T, fx *settingsFixture, ms float64) {
	t.Helper()
	fx.clk.advance(ms)
	fx.screen.Update()
}

// recordedWrites returns the stub's writes for one key.
func recordedWrites(stub *stubSettings, key string) []string {
	var out []string
	for _, w := range stub.writes {
		if w.key == key {
			out = append(out, w.value)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// AC-1: Six categories with the CR-3 settings
// ---------------------------------------------------------------------------

func TestSixCategoriesRegisteredWithExpectedSettings(t *testing.T) {
	fx := newSettingsFixture(t)
	settings := fx.screen.Settings()

	// Category count matches the GDD CR-3 tab list.
	seen := map[mainmenu.Category]bool{}
	for _, st := range settings {
		seen[st.Category] = true
	}
	if len(seen) != mainmenu.CategoryCount {
		t.Fatalf("category count = %d, want %d", len(seen), mainmenu.CategoryCount)
	}

	// Spot-check the CR-3 rows per category (key-level contract).
	want := map[mainmenu.Category][]string{
		mainmenu.CategoryGameplay:      {"gameplay.difficulty", "gameplay.auto_pause_on_conversation", "gameplay.tooltip_verbosity", "gameplay.tutorial_hints"},
		mainmenu.CategoryCamera:        {"camera.edge_scroll", "camera.edge_scroll_deadzone", "camera.edge_scroll_speed", "camera.keyboard_pan_speed", "camera.gamepad_pan_speed", "camera.zoom_transition_duration"},
		mainmenu.CategoryAudio:         {"audio.master_volume", "audio.music_volume", "audio.sfx_volume", "audio.ambient_volume", "audio.dialogue_volume", "audio.mute_when_unfocused"},
		mainmenu.CategoryControls:      {"controls.mouse_sensitivity", "controls.gamepad_sensitivity"},
		mainmenu.CategoryAccessibility: {"camera_motion", "subtitles", "accessibility.colorblind_mode", "accessibility.text_size"},
		mainmenu.CategoryGraphics:      {"graphics.resolution", "graphics.display_mode", "graphics.fps_cap", "graphics.vsync", "graphics.texture_quality", "graphics.anti_aliasing"},
	}
	for cat, keys := range want {
		for _, key := range keys {
			st := fx.screen.Setting(key)
			if st == nil {
				t.Fatalf("category %s: missing setting %q", cat, key)
			}
			if st.Category != cat {
				t.Fatalf("setting %q category = %s, want %s", key, st.Category, cat)
			}
		}
	}

	// Camera knobs match the Camera System GDD tuning table.
	for _, tc := range []struct {
		key                 string
		min, max, step, def float64
	}{
		{"camera.edge_scroll_deadzone", 4, 64, 1, 16},
		{"camera.edge_scroll_speed", 5, 30, 1, 15},
		{"camera.keyboard_pan_speed", 10, 40, 1, 20},
		{"camera.gamepad_pan_speed", 5, 30, 1, 15},
		{"camera.zoom_transition_duration", 150, 600, 10, 300},
	} {
		st := fx.screen.Setting(tc.key)
		if st.Min != tc.min || st.Max != tc.max || st.Step != tc.step || st.Value.Float != tc.def {
			t.Fatalf("%s = [%g,%g]/%g default %g, want [%g,%g]/%g default %g",
				tc.key, st.Min, st.Max, st.Step, st.Value.Float, tc.min, tc.max, tc.step, tc.def)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-2: Instant apply + debounced persistence
// ---------------------------------------------------------------------------

func TestChangeAppliesInstantlyAndDebouncedFlushWrites(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	// A slider change applies instantly — before any flush.
	if !fx.screen.SetSlider("camera.edge_scroll_speed", 20) {
		t.Fatal("SetSlider must accept a valid value")
	}
	wantApplied := []keyValue{{"camera.edge_scroll_speed", "20"}}
	if len(fx.stub.applied) != 1 || fx.stub.applied[0] != wantApplied[0] {
		t.Fatalf("applied = %v, want %v (instant apply, AC-2)", fx.stub.applied, wantApplied)
	}
	if !fx.screen.Dirty() {
		t.Fatal("change must mark the screen dirty")
	}
	if len(fx.stub.writes) != 0 {
		t.Fatalf("no write before the debounce window, got %v", fx.stub.writes)
	}

	// 499ms: still quiet, no flush.
	advanceAndUpdate(t, fx, mainmenu.DebounceMs-1)
	if len(fx.stub.writes) != 0 {
		t.Fatalf("write before DebounceMs: %v", fx.stub.writes)
	}
	// The last millisecond trips the debounce.
	advanceAndUpdate(t, fx, 1)
	if len(fx.stub.writes) != 1 || fx.stub.writes[0] != wantApplied[0] {
		t.Fatalf("writes = %v, want [%v] after DebounceMs", fx.stub.writes, wantApplied[0])
	}
	if fx.screen.Dirty() {
		t.Fatal("dirty must clear after a successful flush")
	}
	if !fx.stub.hasFile || fx.stub.store["camera.edge_scroll_speed"] != "20" {
		t.Fatalf("store = %v, want the flushed value persisted", fx.stub.store)
	}
}

func TestRapidChangesDebounceToFinalValue(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	// Five changes in 200ms (GDD QA case): only the final value is written.
	for i := 0; i < 5; i++ {
		if !fx.screen.SetSlider("camera.edge_scroll_speed", float64(10+i)) {
			t.Fatal("SetSlider failed")
		}
		advanceAndUpdate(t, fx, 40)
	}
	if len(fx.stub.writes) != 0 {
		t.Fatalf("intermediate values must not be written, got %v", fx.stub.writes)
	}
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	writes := recordedWrites(fx.stub, "camera.edge_scroll_speed")
	if len(writes) != 1 || writes[0] != "14" {
		t.Fatalf("writes = %v, want exactly [14] (only the final value)", writes)
	}
	if fx.screen.ValueText("camera.edge_scroll_speed") != "14" {
		t.Fatalf("value = %q, want %q", fx.screen.ValueText("camera.edge_scroll_speed"), "14")
	}
}

func TestCleanShutdownFlushWritesLatestValues(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.screen.SetToggle("camera.edge_scroll", false)
	advanceAndUpdate(t, fx, 100) // mid-debounce

	fx.screen.Flush() // clean shutdown path
	if len(fx.stub.writes) != 2 {
		t.Fatalf("writes = %v, want both dirty keys flushed immediately", fx.stub.writes)
	}
	if fx.screen.Dirty() {
		t.Fatal("dirty must clear after Flush")
	}
	if fx.stub.store["camera.edge_scroll_speed"] != "22" || fx.stub.store["camera.edge_scroll"] != "off" {
		t.Fatalf("store = %v, want the latest values", fx.stub.store)
	}
	// Formatting: values are the exact settings.cfg strings (AC-2 contract).
	if fx.screen.ValueText("camera.edge_scroll") != "off" {
		t.Fatalf("toggle value text = %q, want %q", fx.screen.ValueText("camera.edge_scroll"), "off")
	}
}

func TestSliderValuesClampToStepGrid(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	// Below min clamps to min; above max clamps to max; off-grid snaps.
	fx.screen.SetSlider("camera.edge_scroll_speed", 2)
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "5" {
		t.Fatalf("below-min = %q, want %q", got, "5")
	}
	fx.screen.SetSlider("camera.edge_scroll_speed", 99)
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "30" {
		t.Fatalf("above-max = %q, want %q", got, "30")
	}
	fx.screen.SetSlider("camera.edge_scroll_speed", 15.7)
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "16" {
		t.Fatalf("off-grid = %q, want %q (snapped to the step grid)", got, "16")
	}
	// Fractional steps keep exact decimals (0.8, not 0.8000000000000001).
	fx.screen.SetSlider("audio.master_volume", 0.83)
	if got := fx.screen.ValueText("audio.master_volume"); got != "0.85" {
		t.Fatalf("fractional = %q, want %q", got, "0.85")
	}
	fx.screen.SetSlider("audio.master_volume", 0.8)
	if got := fx.screen.ValueText("audio.master_volume"); got != "0.8" {
		t.Fatalf("exact step = %q, want %q", got, "0.8")
	}
}

// ---------------------------------------------------------------------------
// AC-3: Display resolution preview
// ---------------------------------------------------------------------------

func TestResolutionChangeStartsPreviewAndKeepsScreenInteractive(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.resolution", "1280x720")
	if fx.screen.ValueText("graphics.resolution") != "1280x720" {
		t.Fatalf("resolution = %q, want %q (applied instantly)", fx.screen.ValueText("graphics.resolution"), "1280x720")
	}
	if !fx.screen.PreviewActive() {
		t.Fatal("resolution change must start the preview overlay")
	}
	// The countdown is live and clamped to the 3s window (GDD CR-3).
	if r := fx.screen.PreviewCountdownRemaining(); r != mainmenu.PreviewCountdownMs {
		t.Fatalf("countdown at t0 = %g, want %g", r, mainmenu.PreviewCountdownMs)
	}
	// The Settings screen stays fully accessible behind the overlay (AC-3).
	fx.screen.SetSlider("audio.master_volume", 0.5)
	if got := fx.screen.ValueText("audio.master_volume"); got != "0.5" {
		t.Fatalf("audio change while previewing = %q, want %q (screen stays interactive)", got, "0.5")
	}
	if !fx.screen.PreviewActive() {
		t.Fatal("unrelated changes must not close the preview")
	}
	// The preview overlay buttons are focusable (mouse/gamepad path).
	if !fx.screen.Focus().FocusByID(mainmenu.FocusPreviewKeep) {
		t.Fatal("Keep button must be focusable while the preview is active")
	}
	if !fx.screen.Focus().FocusByID(mainmenu.FocusPreviewRevert) {
		t.Fatal("Revert button must be focusable while the preview is active")
	}
}

func TestPreviewKeepPersistsChanges(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.resolution", "1280x720")
	fx.clk.advance(100)
	if !fx.screen.HandleKey("y") {
		t.Fatal("HandleKey(y) must be consumed while the preview is active")
	}
	if fx.screen.PreviewActive() {
		t.Fatal("preview must close after Keep")
	}
	if fx.screen.ValueText("graphics.resolution") != "1280x720" {
		t.Fatalf("resolution = %q, want %q after Keep", fx.screen.ValueText("graphics.resolution"), "1280x720")
	}
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	if writes := recordedWrites(fx.stub, "graphics.resolution"); len(writes) != 1 || writes[0] != "1280x720" {
		t.Fatalf("resolution writes = %v, want [1280x720] (kept change persists)", writes)
	}
	// Letter keys other than y/n are not consumed.
	if fx.screen.HandleKey("x") {
		t.Fatal("HandleKey(x) must not be consumed")
	}
}

func TestPreviewRevertRestoresPreviousValue(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.resolution", "1280x720")
	if fx.screen.ValueText("graphics.resolution") != "1280x720" {
		t.Fatal("precondition: new resolution applied")
	}
	fx.screen.HandleKey("n")
	if fx.screen.PreviewActive() {
		t.Fatal("preview must close after Revert")
	}
	if got := fx.screen.ValueText("graphics.resolution"); got != "1920x1080" {
		t.Fatalf("resolution = %q, want %q (previous working value restored)", got, "1920x1080")
	}
	// The revert was applied through the normal path: the persisted value
	// stays the default, and the key is dirty only for the revert+keep.
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	writes := recordedWrites(fx.stub, "graphics.resolution")
	if len(writes) != 1 || writes[0] != "1920x1080" {
		t.Fatalf("resolution writes = %v, want [1920x1080] (reverted value persisted)", writes)
	}
}

func TestPreviewRevertViaOverlayButton(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.display_mode", "windowed")
	click(t, fx.flow, mainmenu.FocusPreviewRevert)
	if fx.screen.PreviewActive() {
		t.Fatal("preview must close after the Revert button")
	}
	if got := fx.screen.ValueText("graphics.display_mode"); got != "fullscreen" {
		t.Fatalf("display mode = %q, want %q", got, "fullscreen")
	}
	// The Keep button path via click.
	fx.screen.SelectOptionKey("graphics.display_mode", "windowed")
	click(t, fx.flow, mainmenu.FocusPreviewKeep)
	if fx.screen.PreviewActive() || fx.screen.ValueText("graphics.display_mode") != "windowed" {
		t.Fatalf("after Keep: active=%v mode=%q, want inactive windowed", fx.screen.PreviewActive(), fx.screen.ValueText("graphics.display_mode"))
	}
}

func TestPreviewAutoRevertsAfter15Seconds(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.resolution", "2560x1440")
	advanceAndUpdate(t, fx, mainmenu.PreviewAutoRevertMs-1)
	if !fx.screen.PreviewActive() {
		t.Fatal("preview must survive until the auto-revert deadline")
	}
	advanceAndUpdate(t, fx, 1)
	if fx.screen.PreviewActive() {
		t.Fatal("preview must auto-revert at 15s")
	}
	if got := fx.screen.ValueText("graphics.resolution"); got != "1920x1080" {
		t.Fatalf("resolution = %q, want %q after auto-revert", got, "1920x1080")
	}
}

func TestPreviewCountdownCountsDownAndClamps(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SelectOptionKey("graphics.resolution", "1280x720")
	advanceAndUpdate(t, fx, 1000)
	if r := fx.screen.PreviewCountdownRemaining(); r != mainmenu.PreviewCountdownMs-1000 {
		t.Fatalf("countdown after 1s = %g, want %g", r, mainmenu.PreviewCountdownMs-1000)
	}
	advanceAndUpdate(t, fx, 2500)
	if r := fx.screen.PreviewCountdownRemaining(); r != 0 {
		t.Fatalf("countdown past 3s = %g, want 0 (clamped)", r)
	}
	// The auto-revert deadline is independent (15s) and still counting.
	if r := fx.screen.PreviewAutoRevertRemaining(); r <= 0 || r > mainmenu.PreviewAutoRevertMs-3500 {
		t.Fatalf("auto-revert remaining = %g, want ~%g", r, mainmenu.PreviewAutoRevertMs-3500)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Reset to defaults (with confirmation)
// ---------------------------------------------------------------------------

func TestResetCategoryRequiresConfirmationAndResetsOnlyThatCategory(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetActiveCategory(mainmenu.CategoryCamera)
	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.screen.SetToggle("camera.edge_scroll", false)
	fx.screen.SetActiveCategory(mainmenu.CategoryAudio)
	fx.screen.SetSlider("audio.master_volume", 0.3)

	// Open the per-category dialog from the Camera tab.
	fx.screen.SetActiveCategory(mainmenu.CategoryCamera)
	click(t, fx.flow, mainmenu.FocusResetCategory)
	if fx.screen.State() != mainmenu.SettingsStateResetCategory {
		t.Fatalf("state = %s, want ResetCategory (confirmation required)", fx.screen.State())
	}
	wantMsg := "Reset Camera to default settings?"
	if got := fx.screen.ResetCategoryConfirmMessage(); got != wantMsg {
		t.Fatalf("dialog = %q, want %q", got, wantMsg)
	}

	// Cancel aborts: nothing resets.
	click(t, fx.flow, mainmenu.FocusResetCancel)
	if fx.screen.State() != mainmenu.SettingsStatePanel {
		t.Fatalf("state = %s, want Panel after cancel", fx.screen.State())
	}
	if fx.screen.ValueText("camera.edge_scroll_speed") != "22" {
		t.Fatal("cancel must not reset anything")
	}

	// Confirm resets ONLY the Camera category.
	click(t, fx.flow, mainmenu.FocusResetCategory)
	click(t, fx.flow, mainmenu.FocusResetConfirm)
	if fx.screen.State() != mainmenu.SettingsStatePanel {
		t.Fatalf("state = %s, want Panel after confirm", fx.screen.State())
	}
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "15" {
		t.Fatalf("camera speed = %q, want %q (default restored)", got, "15")
	}
	if got := fx.screen.ValueText("camera.edge_scroll"); got != "on" {
		t.Fatalf("camera edge scroll = %q, want %q (default restored)", got, "on")
	}
	// Other categories untouched.
	if got := fx.screen.ValueText("audio.master_volume"); got != "0.3" {
		t.Fatalf("audio volume = %q, want %q (other categories untouched)", got, "0.3")
	}
	// The reset was applied instantly (dirty -> flushed).
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	if len(recordedWrites(fx.stub, "camera.edge_scroll_speed")) != 1 {
		t.Fatalf("camera writes = %v, want the reset value flushed", recordedWrites(fx.stub, "camera.edge_scroll_speed"))
	}
}

func TestResetAllRequiresConfirmationAndResetsEverything(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.screen.SetSlider("audio.master_volume", 0.3)
	fx.screen.SetToggle("gameplay.tutorial_hints", false)
	fx.screen.SelectOptionKey("accessibility.camera_motion", "off")

	click(t, fx.flow, mainmenu.FocusResetAll)
	if fx.screen.State() != mainmenu.SettingsStateResetAll {
		t.Fatalf("state = %s, want ResetAll (confirmation required)", fx.screen.State())
	}
	if got := fx.screen.ResetAllConfirmMessage(); got != "Reset all settings to defaults?" {
		t.Fatalf("dialog = %q", got)
	}
	click(t, fx.flow, mainmenu.FocusResetConfirm)

	for key, want := range map[string]string{
		"camera.edge_scroll_speed": "15",
		"audio.master_volume":      "0.8",
		"gameplay.tutorial_hints":  "on",
		// Flat CR-7 key (declared in accessibility_prompt.go), not the
		// dotted scheme — the accessibility category's canonical key.
		"camera_motion": "reduced",
	} {
		if got := fx.screen.ValueText(key); got != want {
			t.Fatalf("%s = %q, want %q after reset-all", key, got, want)
		}
	}
	// Everything dirty flushed: the store reflects the defaults.
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	if !fx.stub.hasFile || fx.stub.store["camera.edge_scroll_speed"] != "15" {
		t.Fatalf("store = %v, want reset values persisted", fx.stub.store)
	}
}

// ---------------------------------------------------------------------------
// AC-5: Missing/corrupted settings recovery
// ---------------------------------------------------------------------------

func TestMissingSettingsCreatesDefaultsAndFlagsFirstLaunch(t *testing.T) {
	fx := newSettingsFixture(t)
	fx.stub.hasFile = false

	fx.screen.CheckFirstLaunch()
	if !fx.screen.FirstLaunchPending() {
		t.Fatal("missing settings.cfg must flag the first-launch prompt")
	}
	// Every value is a default, and every key is dirty so the next flush
	// creates the file with all defaults (AC-5: "a new settings.cfg is
	// created with all defaults").
	for _, st := range fx.screen.Settings() {
		if !st.IsDefault() {
			t.Fatalf("%s = %+v, want the default on first launch", st.Key, st.Value)
		}
	}
	if len(fx.screen.DirtyKeys()) != len(fx.screen.Settings()) {
		t.Fatalf("dirty = %d keys, want %d (all defaults must be written)", len(fx.screen.DirtyKeys()), len(fx.screen.Settings()))
	}
	// No apply calls: systems already boot with defaults; the flush only
	// creates the file.
	if len(fx.stub.applied) != 0 {
		t.Fatalf("no apply calls expected on first launch, got %v", fx.stub.applied)
	}
	// The debounced flush writes the fresh file.
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	if fx.screen.Dirty() || fx.stub.store["camera.edge_scroll_speed"] != "15" {
		t.Fatalf("store = %v, dirty = %v; want the fresh defaults written", fx.stub.store, fx.screen.DirtyKeys())
	}
}

func TestExistingSettingsPopulateValuesWithoutDirty(t *testing.T) {
	fx := newSettingsFixture(t)
	fx.stub.store["camera.edge_scroll_speed"] = "22"
	fx.stub.store["graphics.resolution"] = "1600x900"
	fx.stub.store["camera_motion"] = "full"
	fx.stub.store["gameplay.difficulty"] = "tactical"
	fx.stub.store["audio.master_volume"] = "0.45"

	fx.screen.CheckFirstLaunch()
	if fx.screen.FirstLaunchPending() {
		t.Fatal("an existing settings.cfg must not trigger the first-launch prompt")
	}
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "22" {
		t.Fatalf("camera speed = %q, want %q (pre-populated)", got, "22")
	}
	if got := fx.screen.ValueText("graphics.resolution"); got != "1600x900" {
		t.Fatalf("resolution = %q, want %q", got, "1600x900")
	}
	if got := fx.screen.ValueText("camera_motion"); got != "full" {
		t.Fatalf("camera motion = %q, want %q (flat CR-7 key)", got, "full")
	}
	if got := fx.screen.ValueText("gameplay.difficulty"); got != "tactical" {
		t.Fatalf("difficulty = %q, want %q", got, "tactical")
	}
	if got := fx.screen.ValueText("audio.master_volume"); got != "0.45" {
		t.Fatalf("volume = %q, want %q", got, "0.45")
	}
	// Pre-population never dirties: an unmodified boot writes nothing.
	if fx.screen.Dirty() {
		t.Fatalf("pre-population must not dirty the screen, got %v", fx.screen.DirtyKeys())
	}
}

func TestCorruptedStoreValuesFallBackToDefaults(t *testing.T) {
	fx := newSettingsFixture(t)
	// Corrupted entries: unparsable slider, unknown dropdown key, stale
	// toggle spelling. A partial settings.cfg stays valid (AC-5).
	fx.stub.store["camera.edge_scroll_speed"] = "not-a-number"
	fx.stub.store["gameplay.difficulty"] = "legendary"
	fx.stub.store["camera.edge_scroll"] = "yes"
	// A valid entry elsewhere survives.
	fx.stub.store["audio.music_volume"] = "0.25"

	fx.screen.CheckFirstLaunch()
	if fx.screen.FirstLaunchPending() {
		t.Fatal("a partially corrupt settings.cfg must not trigger first-launch")
	}
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "15" {
		t.Fatalf("unparsable slider = %q, want %q (default kept)", got, "15")
	}
	if got := fx.screen.ValueText("gameplay.difficulty"); got != "normal" {
		t.Fatalf("unknown option = %q, want %q (default kept)", got, "normal")
	}
	if got := fx.screen.ValueText("camera.edge_scroll"); got != "on" {
		t.Fatalf("stale toggle = %q, want %q (default kept)", got, "on")
	}
	if got := fx.screen.ValueText("audio.music_volume"); got != "0.25" {
		t.Fatalf("valid entry = %q, want %q (valid values survive)", got, "0.25")
	}
	if fx.screen.Dirty() {
		t.Fatalf("fallback to defaults must not dirty the screen, got %v", fx.screen.DirtyKeys())
	}
}

// ---------------------------------------------------------------------------
// AC-6: Write failure handling
// ---------------------------------------------------------------------------

func TestWriteFailureKeepsDirtyAndLogsNoToastOnFirstFailure(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.stub.writeErr = errors.New("disk full")
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)

	if !fx.screen.Dirty() {
		t.Fatal("failed write must keep the key dirty for the next retry")
	}
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "22" {
		t.Fatalf("in-memory value = %q, want %q (settings keep working in memory)", got, "22")
	}
	if len(fx.stub.logs) == 0 {
		t.Fatal("the first failure must be logged (and only logged)")
	}
	if fx.screen.ToastVisible() {
		t.Fatal("no toast on the first failure — only after two consecutive failures (AC-6)")
	}
	if fx.screen.WriteFailures() != 1 {
		t.Fatalf("failures = %d, want 1", fx.screen.WriteFailures())
	}
}

func TestSecondConsecutiveFailureShowsAutoDismissingToast(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.stub.writeErr = errors.New("disk full")
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #1: logged only
	if fx.screen.ToastVisible() {
		t.Fatal("precondition: no toast after failure #1")
	}
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #2: toast
	if !fx.screen.ToastVisible() {
		t.Fatal("toast must show after two consecutive failures")
	}
	if got := fx.screen.ToastMessage(); got != mainmenu.SettingsSaveFailedMessage {
		t.Fatalf("toast = %q, want %q", got, mainmenu.SettingsSaveFailedMessage)
	}
	// The toast is non-blocking: settings remain fully interactive.
	fx.screen.SetSlider("camera.edge_scroll_speed", 24)
	if got := fx.screen.ValueText("camera.edge_scroll_speed"); got != "24" {
		t.Fatalf("value = %q, want %q (screen stays interactive)", got, "24")
	}
	// Auto-dismiss after SettingsToastMs via the clock. The clock advances
	// WITHOUT Update: the toast lifetime is a pure clock function, and no
	// flush re-arms it mid-window.
	fx.clk.advance(mainmenu.SettingsToastMs - 1)
	if !fx.screen.ToastVisible() {
		t.Fatal("toast must still be visible just before the timeout")
	}
	fx.clk.advance(1)
	if fx.screen.ToastVisible() {
		t.Fatal("toast must auto-dismiss after SettingsToastMs")
	}
	// Manual dismissal also works (a fresh failure re-shows it).
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #3: toast again
	if !fx.screen.ToastVisible() {
		t.Fatal("precondition: third failure shows the toast again")
	}
	fx.screen.DismissToast()
	if fx.screen.ToastVisible() {
		t.Fatal("toast must dismiss on DismissToast")
	}
}

func TestSuccessfulWriteClearsErrorStateAndDismissesToast(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.stub.writeErr = errors.New("disk full")
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #1
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #2 -> toast
	if !fx.screen.ToastVisible() {
		t.Fatal("precondition: toast visible after two failures")
	}

	// The disk recovers: the next flush succeeds.
	fx.stub.writeErr = nil
	advanceAndUpdate(t, fx, mainmenu.DebounceMs)
	if fx.screen.Dirty() {
		t.Fatalf("successful write must clear the dirty set, got %v", fx.screen.DirtyKeys())
	}
	if fx.screen.ToastVisible() {
		t.Fatal("a successful write must clear the error state silently (toast dismissed)")
	}
	if fx.screen.WriteFailures() != 0 {
		t.Fatalf("failures = %d, want 0 after success", fx.screen.WriteFailures())
	}
	if fx.stub.store["camera.edge_scroll_speed"] != "22" {
		t.Fatalf("store = %v, want the retried value persisted", fx.stub.store)
	}
}

func TestFlushRetriesOnNextChangeAfterFailures(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	fx.screen.SetSlider("camera.edge_scroll_speed", 22)
	fx.stub.writeErr = errors.New("disk full")
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #1
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #2 -> toast
	toastLogs := len(fx.stub.logs)

	// A new change re-arms the debounce and retries the whole dirty set
	// (the write path never gives up permanently).
	fx.screen.SetSlider("audio.master_volume", 0.4)
	advanceAndUpdate(t, fx, mainmenu.DebounceMs) // failure #3
	if len(fx.stub.logs) <= toastLogs {
		t.Fatal("the retry must be attempted (logged again) after a new change")
	}
	if fx.screen.WriteFailures() != 3 {
		t.Fatalf("failures = %d, want 3 (retried, never stuck)", fx.screen.WriteFailures())
	}
	if !fx.screen.Dirty() {
		t.Fatal("dirty keys must survive the failed retry")
	}
}

// ---------------------------------------------------------------------------
// AC-7: Deferred engine features
// ---------------------------------------------------------------------------

func TestDeferredControlsGreyedWithComingSoonAndRejectInteraction(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	// The deferred rows are flagged in the registry (data contract).
	for _, key := range []string{"graphics.vsync", "graphics.texture_quality", "graphics.anti_aliasing", "audio.output_device"} {
		st := fx.screen.Setting(key)
		if st == nil || !st.Deferred {
			t.Fatalf("%s: Deferred = %v, want true", key, st)
		}
		// Programmatic changes are rejected.
		if fx.screen.SetToggle("graphics.vsync", true) {
			t.Fatalf("SetToggle(%s) must be rejected for a deferred setting", key)
		}
		if fx.screen.SelectOptionKey("graphics.texture_quality", "low") {
			t.Fatalf("SelectOptionKey(%s) must be rejected", key)
		}
		// The value stays at its default.
		if !st.IsDefault() {
			t.Fatalf("%s value changed despite deferred: %+v", key, st.Value)
		}
	}

	// Render commands carry the greyed treatment + tooltip (AC-7).
	fx.screen.SetActiveCategory(mainmenu.CategoryGraphics)
	cmds := fx.screen.RenderCommands(nil)
	seen := map[string]bool{}
	for _, c := range cmds {
		if c.Type == mainmenu.RenderSettingRow {
			seen[c.Setting.Key] = c.Dimmed
		}
	}
	if !seen["graphics.vsync"] || !seen["graphics.texture_quality"] || !seen["graphics.anti_aliasing"] {
		t.Fatalf("deferred rows must render (dimmed): %v", seen)
	}
	// Interactive rows are not dimmed.
	if seen["graphics.fps_cap"] {
		t.Fatal("fps_cap must not be dimmed (it is NOT deferred)")
	}
	for _, key := range []string{"graphics.vsync", "graphics.texture_quality", "graphics.anti_aliasing"} {
		var tooltip string
		for _, c := range cmds {
			if c.Type == mainmenu.RenderSettingRow && c.Setting.Key == key {
				tooltip = c.Tooltip
			}
		}
		if tooltip != mainmenu.TooltipComingSoon {
			t.Fatalf("%s tooltip = %q, want %q", key, tooltip, mainmenu.TooltipComingSoon)
		}
	}

	// Greyed rows are not focusable: clicks are ignored.
	clickIgnored(t, fx.flow, fx.screen.SettingFocusID("graphics.vsync"))
	clickIgnored(t, fx.flow, fx.screen.SettingFocusID("audio.output_device"))
	if len(fx.stub.applied) != 0 {
		t.Fatalf("deferred rows must never apply, got %v", fx.stub.applied)
	}
}

func TestFPSCapIsInteractive(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	// Not deferred: cycling via click applies instantly (AC-2/AC-7 edge).
	fx.screen.SetActiveCategory(mainmenu.CategoryGraphics)
	click(t, fx.flow, fx.screen.SettingFocusID("graphics.fps_cap")) // 60 -> 120
	if got := fx.screen.ValueText("graphics.fps_cap"); got != "120" {
		t.Fatalf("fps cap = %q, want %q after click", got, "120")
	}
	last := fx.stub.applied[len(fx.stub.applied)-1]
	if last != (keyValue{"graphics.fps_cap", "120"}) {
		t.Fatalf("applied = %+v, want fps_cap=120", last)
	}
}

// ---------------------------------------------------------------------------
// AC-8: In-game Graphics restrictions
// ---------------------------------------------------------------------------

func TestPauseMenuEntryRestrictsGraphicsAbsentNotGreyed(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettingsFromPause(t, fx)

	if fx.screen.EntryPoint() != mainmenu.ScreenPauseMenu {
		t.Fatalf("entry point = %s, want %s (fixed at screen open)", fx.screen.EntryPoint(), mainmenu.ScreenPauseMenu)
	}
	if !fx.screen.GraphicsRestricted() {
		t.Fatal("settings entered from the pause menu must restrict Graphics")
	}

	visible := fx.screen.VisibleSettings(mainmenu.CategoryGraphics)
	for _, st := range visible {
		if st.Key == "graphics.resolution" || st.Key == "graphics.display_mode" {
			t.Fatalf("%s must be ABSENT (not greyed) in-game", st.Key)
		}
	}
	hasRes, hasMode := false, false
	for _, st := range visible {
		hasRes = hasRes || st.Key == "graphics.resolution"
		hasMode = hasMode || st.Key == "graphics.display_mode"
	}
	if hasRes || hasMode {
		t.Fatal("resolution/display mode must not be listed in-game")
	}
	// Render commands for the Graphics tab mirror the absence.
	cmds := fx.screen.RenderCommands(nil)
	for _, c := range cmds {
		if c.Type == mainmenu.RenderSettingRow && (c.Setting.Key == "graphics.resolution" || c.Setting.Key == "graphics.display_mode") {
			t.Fatalf("render command for %s in restricted mode", c.Setting.Key)
		}
	}
	// Not focusable either.
	clickIgnored(t, fx.flow, fx.screen.SettingFocusID("graphics.resolution"))
	// Non-restricted settings in the tab stay usable (switch to Graphics).
	fx.screen.SetActiveCategory(mainmenu.CategoryGraphics)
	click(t, fx.flow, fx.screen.SettingFocusID("graphics.fps_cap"))
	if got := fx.screen.ValueText("graphics.fps_cap"); got != "120" {
		t.Fatalf("fps cap = %q, want %q in-game (non-restricted rows stay live)", got, "120")
	}
}

func TestTitleMenuEntryShowsFullGraphics(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	if fx.screen.EntryPoint() != mainmenu.ScreenTitleScreen {
		t.Fatalf("entry point = %s, want %s", fx.screen.EntryPoint(), mainmenu.ScreenTitleScreen)
	}
	if fx.screen.GraphicsRestricted() {
		t.Fatal("settings from the title screen must show full Graphics")
	}
	for _, key := range []string{"graphics.resolution", "graphics.display_mode"} {
		st := fx.screen.Setting(key)
		if st == nil || !st.Visible(fx.screen.GraphicsRestricted()) {
			t.Fatalf("%s must be visible from the title screen", key)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-9: Performance (UI update + render-command generation ≤ 2ms p95)
// ---------------------------------------------------------------------------

func TestSettingsScreenUpdateAndRenderP95Under2ms(t *testing.T) {
	fx := newSettingsFixture(t)
	navigateToSettings(t, fx)

	const frames = 3000
	durations := make([]float64, 0, frames)
	scratch := make([]mainmenu.RenderCommand, 0, 64)
	var total float64
	for i := 0; i < frames; i++ {
		fx.clk.advance(16.6)
		start := time.Now()
		fx.screen.Update()
		_ = fx.screen.RenderCommands(scratch[:0])
		elapsed := time.Since(start).Seconds() * 1000
		durations = append(durations, elapsed)
		total += elapsed
	}
	sort.Float64s(durations)
	// p95 index (rounded up, 1-based percentile).
	idx := int(float64(len(durations))*0.95) - 1
	if idx < 0 {
		idx = 0
	}
	p95 := durations[idx]
	if p95 > 2.0 {
		t.Fatalf("p95 = %.3fms over %d frames, want <= 2ms (TR-menu-014)", p95, frames)
	}
	t.Logf("settings screen p95 = %.3fms, mean = %.3fms over %d frames", p95, total/float64(frames), frames)
}

func BenchmarkSettingsScreenUpdateAndRenderCommands(b *testing.B) {
	fx, err := buildSettingsFixture()
	if err != nil {
		b.Fatalf("build fixture: %v", err)
	}
	b.ReportAllocs()
	scratch := make([]mainmenu.RenderCommand, 0, 64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fx.clk.advance(16.6)
		fx.screen.Update()
		_ = fx.screen.RenderCommands(scratch[:0])
	}
}

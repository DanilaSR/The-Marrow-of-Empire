// Unit tests for Story 003: First-Launch Accessibility Prompt (Main Menu &
// Settings epic).
//
// Evidence path: production/epics/main-menu-settings/
// story-003-first-launch-accessibility-prompt.md
// Required evidence: this file must exist and pass (UI story gate).
//
// These tests drive src/ui/mainmenu/accessibility_prompt.go headlessly with
// the same injectable fakeClock and click helper as the Story 001/002
// suites (shared package main_menu_test) — no file I/O, no engine host, no
// external APIs. settings.cfg persistence is asserted through stub
// HasSettings/GetSetting/WriteSettings callbacks.
//
// Acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_prompt_appears_with_defaults_on_first_launch     -> TestAccessibilityPromptAppearsWithDefaultsOnFirstLaunch
//	AC-2  test_continue_dismisses_and_persists                  -> TestAccessibilityPromptContinueDismissesAndPersists
//	AC-2  test_escape_dismisses_and_applies_changed_values      -> TestAccessibilityPromptEscapeDismissesAndAppliesChangedValues
//	AC-2  test_remember_unchecked_applies_session_only          -> TestAccessibilityPromptRememberUncheckedAppliesSessionOnly
//	AC-2  test_no_timer_or_click_away_dismissal                 -> TestAccessibilityPromptDoesNotAutoDismiss
//	AC-3  test_prompt_skipped_when_settings_exist               -> TestAccessibilityPromptSkippedWhenSettingsExist
//	AC-4  test_rerun_prepopulates_and_labels_apply              -> TestAccessibilityPromptReRunPrepoulatesAndLabelsApply
//	AC-4  test_rerun_unknown_values_fall_back_to_defaults       -> TestAccessibilityPromptReRunUnknownValuesFallBackToDefaults
//	AC-5  test_keyboard_tab_order_and_arrow_value_cycling       -> TestAccessibilityPromptKeyboardTabOrderAndArrowValueCycling
//	AC-5  test_gamepad_navigation                               -> TestAccessibilityPromptGamepadNavigation
//	AC-6  test_prompt_shown_when_settings_missing               -> TestAccessibilityPromptShownWhenSettingsMissing
//	edge  test_on_exit_restores_title_interactivity             -> TestAccessibilityPromptExitRestoresTitleInteractivity
//	edge  test_appear_progress_tracks_clock                     -> TestAccessibilityPromptAppearProgressTracksClock
package main_menu_test

import (
	"reflect"
	"testing"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// promptProbes collects the injected prompt probe/callback state so tests
// can flip probes and assert callback invocations.
type promptProbes struct {
	hasSettings  bool
	settings     map[string]string
	writes       []string // "key=value" pairs in write order
	dismissCalls int
}

// newPrompt builds a flow, a TitleScreen (with saves present so all four
// menu buttons enable when interactive), and an AccessibilityPrompt wired
// to the same fake clock with stub probes/callbacks. The title controller
// is registered for ScreenTitleScreen; the prompt is the overlay the owner
// drives directly.
func newPrompt(t *testing.T) (*promptProbes, *mainmenu.ScreenFlow, *mainmenu.TitleScreen, *mainmenu.AccessibilityPrompt, *fakeClock) {
	t.Helper()
	clk := &fakeClock{}
	flow := mainmenu.NewScreenFlow(mainmenu.NewTransitionSystem(clk.now))
	p := &promptProbes{settings: map[string]string{}}
	title := mainmenu.NewTitleScreen(flow,
		mainmenu.WithClock(clk.now),
		mainmenu.WithHasSaves(func() bool { return true }),
	)
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, title); err != nil {
		t.Fatalf("register title controller: %v", err)
	}
	prompt := mainmenu.NewAccessibilityPrompt(title,
		mainmenu.WithPromptClock(clk.now),
		mainmenu.WithHasSettings(func() bool { return p.hasSettings }),
		mainmenu.WithGetSetting(func(key string) string { return p.settings[key] }),
		mainmenu.WithWriteSettings(func(key, value string) { p.writes = append(p.writes, key+"="+value) }),
		mainmenu.WithOnDismiss(func() { p.dismissCalls++ }),
	)
	return p, flow, title, prompt, clk
}

// key dispatches a keyboard event through the prompt's own input path (the
// owner routes input to the prompt while it is visible).
func key(prompt *mainmenu.AccessibilityPrompt, k string, shift bool) bool {
	return prompt.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: k, Shift: shift}))
}

// pad dispatches a gamepad event through the prompt's own input path.
func pad(prompt *mainmenu.AccessibilityPrompt, b string) bool {
	return prompt.Dispatch(mainmenu.TranslateGamepad(mainmenu.GamepadEvent{Button: b}))
}

// ---------------------------------------------------------------------------
// AC-1: First-launch modal display
// ---------------------------------------------------------------------------

func TestAccessibilityPromptAppearsWithDefaultsOnFirstLaunch(t *testing.T) {
	p, flow, title, prompt, _ := newPrompt(t)
	p.hasSettings = false
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// Modal is visible with safe defaults (CR-7): Reduced / On / checked.
	if !prompt.Visible() {
		t.Fatal("modal must be visible on first launch")
	}
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("default camera motion = %v, want %v", got, mainmenu.CameraMotionReduced)
	}
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOn {
		t.Fatalf("default subtitles = %v, want %v", got, mainmenu.SubtitlesOn)
	}
	if !prompt.RememberChoice() {
		t.Fatal("'Remember my choice' must default to checked")
	}
	if got := prompt.ConfirmLabel(); got != mainmenu.LabelContinue {
		t.Fatalf("first-launch button label = %q, want %q", got, mainmenu.LabelContinue)
	}
	if got := mainmenu.LabelContinue; got != "Continue" {
		t.Fatalf("Continue label text = %q, want the CR-7 text", got)
	}

	// Initial focus is the first control (asserted before the focusability
	// loop below: FocusByID moves focus).
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusCameraMotion {
		t.Fatalf("initial focus = %q, want %q", got, mainmenu.FocusCameraMotion)
	}

	// Exactly these four controls, in Tab order, all focusable.
	order := []string{
		mainmenu.FocusCameraMotion,
		mainmenu.FocusSubtitles,
		mainmenu.FocusRemember,
		mainmenu.FocusContinue,
	}
	if n := prompt.Focus().Count(); n != len(order) {
		t.Fatalf("focusable controls = %d, want %d", n, len(order))
	}
	for _, id := range order {
		if !prompt.Focus().FocusByID(id) {
			t.Fatalf("control %q must be focusable", id)
		}
	}

	// Dropdown option lists and settings.cfg keys.
	wantMotion := []mainmenu.CameraMotion{mainmenu.CameraMotionFull, mainmenu.CameraMotionReduced, mainmenu.CameraMotionOff}
	if got := mainmenu.CameraMotionOptions(); !reflect.DeepEqual(got, wantMotion) {
		t.Fatalf("camera motion options = %v, want %v", got, wantMotion)
	}
	wantSubs := []mainmenu.Subtitles{mainmenu.SubtitlesOn, mainmenu.SubtitlesOff}
	if got := mainmenu.SubtitlesOptions(); !reflect.DeepEqual(got, wantSubs) {
		t.Fatalf("subtitles options = %v, want %v", got, wantSubs)
	}
	if got := mainmenu.CameraMotionReduced.Key(); got != "reduced" {
		t.Fatalf("reduced key = %q, want %q", got, "reduced")
	}
	if got := mainmenu.CameraMotionOff.Key(); got != "off" {
		t.Fatalf("off key = %q, want %q", got, "off")
	}
	if got := mainmenu.SubtitlesOn.Key(); got != "on" {
		t.Fatalf("subtitles on key = %q, want %q", got, "on")
	}
	if got := mainmenu.CameraMotionReduced.String(); got != "Reduced" {
		t.Fatalf("reduced label = %q, want %q", got, "Reduced")
	}
	if got := mainmenu.SubtitlesOn.String(); got != "On" {
		t.Fatalf("subtitles on label = %q, want %q", got, "On")
	}
	if v, ok := mainmenu.ParseCameraMotion("reduced"); !ok || v != mainmenu.CameraMotionReduced {
		t.Fatalf("ParseCameraMotion(reduced) = %v, %v; want Reduced, true", v, ok)
	}
	if v, ok := mainmenu.ParseSubtitles("bogus"); ok || v != mainmenu.SubtitlesOn {
		t.Fatalf("ParseSubtitles(bogus) = %v, %v; want default, false", v, ok)
	}

	// The title screen menu stays visible behind (buttons still registered)
	// but is not interactive (AC-1).
	if n := title.Focus().Count(); n != 11 {
		t.Fatalf("title registered elements = %d, want 11 (buttons stay visible)", n)
	}
	if n := title.Focus().EnabledCount(); n != 0 {
		t.Fatalf("title enabled elements while modal active = %d, want 0", n)
	}
	if click(flow, mainmenu.FocusNewGame) {
		t.Fatal("title menu click must be ignored while the modal is active")
	}
	if title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("title buttons must not be focusable while the modal is active")
	}
}

// ---------------------------------------------------------------------------
// AC-2: Dismissal via Continue
// ---------------------------------------------------------------------------

func TestAccessibilityPromptContinueDismissesAndPersists(t *testing.T) {
	p, _, title, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// Click Continue (mouse path: focus + activate).
	if !prompt.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: mainmenu.FocusContinue})) {
		t.Fatal("Continue click not handled")
	}
	if prompt.Visible() {
		t.Fatal("modal must be removed after Continue")
	}
	// Both settings persist in order, with the currently-selected values.
	want := []string{"camera_motion=reduced", "subtitles=on"}
	if !reflect.DeepEqual(p.writes, want) {
		t.Fatalf("settings written = %v, want %v", p.writes, want)
	}
	if p.dismissCalls != 1 {
		t.Fatalf("dismiss callback calls = %d, want 1", p.dismissCalls)
	}
	// The title screen menu is interactive again.
	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("title enabled elements after dismissal = %d, want 4", n)
	}
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must be focusable after dismissal")
	}
}

func TestAccessibilityPromptEscapeDismissesAndAppliesChangedValues(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// Change Camera Motion to Off with the arrow keys (Reduced -> Off).
	if !key(prompt, "ArrowRight", false) {
		t.Fatal("arrow key not handled")
	}
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionOff {
		t.Fatalf("camera motion after arrow = %v, want %v", got, mainmenu.CameraMotionOff)
	}

	// Escape produces the identical result to Continue: the CURRENT value
	// is applied, not the default.
	if !key(prompt, "Escape", false) {
		t.Fatal("Escape not handled")
	}
	if prompt.Visible() {
		t.Fatal("modal must be removed after Escape")
	}
	want := []string{"camera_motion=off", "subtitles=on"}
	if !reflect.DeepEqual(p.writes, want) {
		t.Fatalf("settings written = %v, want %v (changed values applied, not defaults)", p.writes, want)
	}
	if p.dismissCalls != 1 {
		t.Fatalf("dismiss callback calls = %d, want 1", p.dismissCalls)
	}
}

func TestAccessibilityPromptRememberUncheckedAppliesSessionOnly(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// Change a value, then uncheck "Remember my choice".
	key(prompt, "ArrowRight", false) // Reduced -> Off
	key(prompt, "Tab", false)        // subtitles
	key(prompt, "Tab", false)        // remember
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusRemember {
		t.Fatalf("focus = %q, want %q", got, mainmenu.FocusRemember)
	}
	if !key(prompt, "Enter", false) {
		t.Fatal("Enter on the checkbox not handled")
	}
	if prompt.RememberChoice() {
		t.Fatal("checkbox must be unchecked after activation")
	}

	// Escape dismisses without persisting; the session values stay applied.
	if !key(prompt, "Escape", false) {
		t.Fatal("Escape not handled")
	}
	if prompt.Visible() {
		t.Fatal("modal must be removed after Escape")
	}
	if len(p.writes) != 0 {
		t.Fatalf("settings written = %v, want none (session-only)", p.writes)
	}
	if p.dismissCalls != 1 {
		t.Fatalf("dismiss callback calls = %d, want 1", p.dismissCalls)
	}
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionOff {
		t.Fatalf("camera motion after session-only dismissal = %v, want %v (applied for session)", got, mainmenu.CameraMotionOff)
	}
}

func TestAccessibilityPromptDoesNotAutoDismiss(t *testing.T) {
	p, _, _, prompt, clk := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// No timer: a long clock advance changes nothing.
	clk.advance(60000)
	if !prompt.Visible() {
		t.Fatal("modal must not dismiss on a timer")
	}
	// No click-away: a click on empty space (no element) is ignored.
	if prompt.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{})) {
		t.Fatal("click-away must be ignored")
	}
	if !prompt.Visible() {
		t.Fatal("modal must not dismiss on click-away")
	}
	// A click on an unregistered background element is ignored as well.
	if prompt.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: "prompt-backdrop"})) {
		t.Fatal("background click must be ignored")
	}
	if !prompt.Visible() {
		t.Fatal("modal must not dismiss on background click")
	}
	if p.dismissCalls != 0 {
		t.Fatalf("dismiss callback calls = %d, want 0", p.dismissCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-3: Prompt suppressed when settings.cfg exists
// ---------------------------------------------------------------------------

func TestAccessibilityPromptSkippedWhenSettingsExist(t *testing.T) {
	p, _, title, prompt, _ := newPrompt(t)
	p.hasSettings = true
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	if prompt.Visible() {
		t.Fatal("prompt must never appear when a valid settings.cfg exists")
	}
	// The title screen menu is immediately interactive.
	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("title enabled elements = %d, want 4 (immediately interactive)", n)
	}
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must be focusable")
	}
	if p.dismissCalls != 0 {
		t.Fatalf("dismiss callback calls = %d, want 0", p.dismissCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Re-run from Settings -> Accessibility
// ---------------------------------------------------------------------------

func TestAccessibilityPromptReRunPrepoulatesAndLabelsApply(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	p.hasSettings = true
	p.settings["camera_motion"] = "off"
	p.settings["subtitles"] = "off"

	prompt.SetReRun(true)
	if got := prompt.ConfirmLabel(); got != mainmenu.LabelApply {
		t.Fatalf("re-run button label = %q, want %q", got, mainmenu.LabelApply)
	}
	if got := mainmenu.LabelApply; got != "Apply" {
		t.Fatalf("Apply label text = %q, want the AC-4 text", got)
	}
	// Current values pre-populate the controls.
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionOff {
		t.Fatalf("pre-populated camera motion = %v, want %v", got, mainmenu.CameraMotionOff)
	}
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOff {
		t.Fatalf("pre-populated subtitles = %v, want %v", got, mainmenu.SubtitlesOff)
	}
	// Re-run shows the modal even though settings.cfg exists (AC-3 only
	// gates the first-launch path).
	prompt.OnEnter(mainmenu.ScreenTitleScreen)
	if !prompt.Visible() {
		t.Fatal("re-run must always show the modal")
	}
}

func TestAccessibilityPromptReRunUnknownValuesFallBackToDefaults(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	p.hasSettings = true
	// Partial config: one of the two settings present (AC-3 edge case).
	p.settings["subtitles"] = "off"

	prompt.SetReRun(true)
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("camera motion with missing key = %v, want default %v", got, mainmenu.CameraMotionReduced)
	}
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOff {
		t.Fatalf("subtitles with present key = %v, want %v", got, mainmenu.SubtitlesOff)
	}
	if got := prompt.ConfirmLabel(); got != mainmenu.LabelApply {
		t.Fatalf("re-run button label = %q, want %q", got, mainmenu.LabelApply)
	}

	// An unknown key value also falls back to the default.
	p.settings["camera_motion"] = "turbo"
	prompt.SetReRun(true)
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("camera motion with unknown key = %v, want default %v", got, mainmenu.CameraMotionReduced)
	}
}

// ---------------------------------------------------------------------------
// AC-5: Keyboard and gamepad navigation
// ---------------------------------------------------------------------------

func TestAccessibilityPromptKeyboardTabOrderAndArrowValueCycling(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusCameraMotion {
		t.Fatalf("initial focus = %q, want %q", got, mainmenu.FocusCameraMotion)
	}
	// Tab visits the remaining controls in order and wraps from Continue
	// back to Camera Motion.
	order := []string{
		mainmenu.FocusSubtitles,
		mainmenu.FocusRemember,
		mainmenu.FocusContinue,
		mainmenu.FocusCameraMotion,
	}
	for i, id := range order {
		if !key(prompt, "Tab", false) {
			t.Fatalf("tab %d not handled", i)
		}
		if got := prompt.Focus().FocusedID(); got != id {
			t.Fatalf("tab %d: focus = %q, want %q", i, got, id)
		}
	}
	// Shift+Tab reverses.
	if !key(prompt, "Tab", true) {
		t.Fatal("shift+tab not handled")
	}
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusContinue {
		t.Fatalf("shift+tab: focus = %q, want %q", got, mainmenu.FocusContinue)
	}

	// Arrow keys cycle the focused dropdown's value; focus stays put.
	// Navigate back to Camera Motion and cycle through all four directions.
	key(prompt, "Tab", true) // remember
	key(prompt, "Tab", true) // subtitles
	key(prompt, "Tab", true) // camera motion
	key(prompt, "ArrowRight", false) // Reduced -> Off
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionOff {
		t.Fatalf("right arrow: camera motion = %v, want %v", got, mainmenu.CameraMotionOff)
	}
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusCameraMotion {
		t.Fatalf("focus after arrow cycle = %q, want %q (focus stays on the dropdown)", got, mainmenu.FocusCameraMotion)
	}
	key(prompt, "ArrowDown", false) // Off -> Full
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionFull {
		t.Fatalf("down arrow: camera motion = %v, want %v", got, mainmenu.CameraMotionFull)
	}
	key(prompt, "ArrowUp", false) // Full -> Off (backward wraps in cycle order)
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionOff {
		t.Fatalf("up arrow: camera motion = %v, want %v", got, mainmenu.CameraMotionOff)
	}
	key(prompt, "ArrowLeft", false) // Off -> Reduced
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("left arrow: camera motion = %v, want %v", got, mainmenu.CameraMotionReduced)
	}

	// Subtitles dropdown cycles the same way.
	key(prompt, "Tab", false) // subtitles
	key(prompt, "ArrowRight", false) // On -> Off
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOff {
		t.Fatalf("right arrow: subtitles = %v, want %v", got, mainmenu.SubtitlesOff)
	}
	key(prompt, "ArrowLeft", false) // Off -> On
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOn {
		t.Fatalf("left arrow: subtitles = %v, want %v", got, mainmenu.SubtitlesOn)
	}

	// Enter/Space toggles the checkbox.
	key(prompt, "Tab", false) // remember
	key(prompt, "Enter", false)
	if prompt.RememberChoice() {
		t.Fatal("Enter must toggle the checkbox off")
	}
	key(prompt, "Space", false)
	if !prompt.RememberChoice() {
		t.Fatal("Space must toggle the checkbox back on")
	}

	// Enter on Continue dismisses with the cycled values.
	key(prompt, "Tab", false) // continue
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusContinue {
		t.Fatalf("focus before final enter = %q, want %q", got, mainmenu.FocusContinue)
	}
	if !key(prompt, "Enter", false) {
		t.Fatal("Enter on Continue not handled")
	}
	if prompt.Visible() {
		t.Fatal("Enter on Continue must dismiss the modal")
	}
	want := []string{"camera_motion=reduced", "subtitles=on"}
	if !reflect.DeepEqual(p.writes, want) {
		t.Fatalf("settings written = %v, want %v", p.writes, want)
	}
	if p.dismissCalls != 1 {
		t.Fatalf("dismiss callback calls = %d, want 1", p.dismissCalls)
	}
}

func TestAccessibilityPromptGamepadNavigation(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// D-pad moves focus between controls; it does not change values.
	if !pad(prompt, "down") {
		t.Fatal("D-pad down not handled")
	}
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusSubtitles {
		t.Fatalf("D-pad down: focus = %q, want %q", got, mainmenu.FocusSubtitles)
	}
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("camera motion after D-pad = %v, want unchanged %v", got, mainmenu.CameraMotionReduced)
	}
	// A activates the focused control: on a dropdown it cycles the value.
	if !pad(prompt, "a") {
		t.Fatal("A not handled")
	}
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOff {
		t.Fatalf("A on subtitles: subtitles = %v, want %v", got, mainmenu.SubtitlesOff)
	}
	// D-pad continues; A toggles the checkbox twice (off, then back on).
	pad(prompt, "down") // remember
	pad(prompt, "a")
	if prompt.RememberChoice() {
		t.Fatal("A must toggle the checkbox off")
	}
	pad(prompt, "a")
	if !prompt.RememberChoice() {
		t.Fatal("A must toggle the checkbox back on")
	}
	// A on Continue dismisses with the current values.
	pad(prompt, "down") // continue
	if got := prompt.Focus().FocusedID(); got != mainmenu.FocusContinue {
		t.Fatalf("D-pad down: focus = %q, want %q", got, mainmenu.FocusContinue)
	}
	if !pad(prompt, "a") {
		t.Fatal("A on Continue not handled")
	}
	if prompt.Visible() {
		t.Fatal("A on Continue must dismiss the modal")
	}
	want := []string{"camera_motion=reduced", "subtitles=off"}
	if !reflect.DeepEqual(p.writes, want) {
		t.Fatalf("settings written = %v, want %v", p.writes, want)
	}

	// B is the gamepad Escape: re-show and dismiss with B.
	prompt.OnEnter(mainmenu.ScreenTitleScreen)
	if !prompt.Visible() {
		t.Fatal("modal must re-show")
	}
	if !pad(prompt, "b") {
		t.Fatal("B not handled")
	}
	if prompt.Visible() {
		t.Fatal("B must dismiss the modal")
	}
	if p.dismissCalls != 2 {
		t.Fatalf("dismiss callback calls = %d, want 2", p.dismissCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-6: Missing settings.cfg -> first-launch path
// ---------------------------------------------------------------------------

func TestAccessibilityPromptShownWhenSettingsMissing(t *testing.T) {
	p, _, _, prompt, _ := newPrompt(t)
	p.hasSettings = false
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	// The missing-file path behaves exactly like first launch: the prompt
	// is shown (Story 005 creates the defaults file; no error is displayed
	// to the player — the prompt is the only surface).
	if !prompt.Visible() {
		t.Fatal("prompt must appear when settings.cfg is missing")
	}
	if got := prompt.ConfirmLabel(); got != mainmenu.LabelContinue {
		t.Fatalf("button label = %q, want %q (first-launch path)", got, mainmenu.LabelContinue)
	}
	// Safe defaults per CR-7: Escape/Skip applies accessible defaults.
	if got := prompt.CameraMotion(); got != mainmenu.CameraMotionReduced {
		t.Fatalf("camera motion = %v, want safe default %v", got, mainmenu.CameraMotionReduced)
	}
	if got := prompt.Subtitles(); got != mainmenu.SubtitlesOn {
		t.Fatalf("subtitles = %v, want safe default %v", got, mainmenu.SubtitlesOn)
	}
}

// ---------------------------------------------------------------------------
// Edge cases: OnExit contract and slide-in timing
// ---------------------------------------------------------------------------

func TestAccessibilityPromptExitRestoresTitleInteractivity(t *testing.T) {
	p, _, title, prompt, _ := newPrompt(t)
	prompt.OnEnter(mainmenu.ScreenTitleScreen)
	if n := title.Focus().EnabledCount(); n != 0 {
		t.Fatalf("title enabled elements while modal active = %d, want 0", n)
	}

	// The owner removes the overlay without dismissal (e.g. leaving the
	// title screen): interactivity is restored, nothing is written, and no
	// dismiss callback fires.
	prompt.OnExit()
	if prompt.Visible() {
		t.Fatal("modal must be hidden after OnExit")
	}
	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("title enabled elements after OnExit = %d, want 4", n)
	}
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must be focusable after OnExit")
	}
	if len(p.writes) != 0 {
		t.Fatalf("settings written = %v, want none (OnExit is not a dismissal)", p.writes)
	}
	if p.dismissCalls != 0 {
		t.Fatalf("dismiss callback calls = %d, want 0", p.dismissCalls)
	}
	// OnExit is idempotent.
	prompt.OnExit()
	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("title enabled elements after double OnExit = %d, want 4", n)
	}
}

func TestAccessibilityPromptAppearProgressTracksClock(t *testing.T) {
	p, _, _, prompt, clk := newPrompt(t)
	if got := prompt.AppearProgress(); got != 0 {
		t.Fatalf("hidden modal progress = %g, want 0", got)
	}
	p.hasSettings = false
	prompt.OnEnter(mainmenu.ScreenTitleScreen)

	if got := prompt.AppearProgress(); got != 0 {
		t.Fatalf("progress at t=0 = %g, want 0", got)
	}
	clk.advance(mainmenu.ModalSlideInMs / 2)
	if got := prompt.AppearProgress(); got != 0.5 {
		t.Fatalf("progress at half slide = %g, want 0.5", got)
	}
	clk.advance(mainmenu.ModalSlideInMs / 2)
	if got := prompt.AppearProgress(); got != 1.0 {
		t.Fatalf("progress at full slide = %g, want 1.0", got)
	}
	clk.advance(1000)
	if got := prompt.AppearProgress(); got != 1.0 {
		t.Fatalf("progress after slide = %g, want 1.0 (clamped)", got)
	}
	prompt.OnExit()
	if got := prompt.AppearProgress(); got != 0 {
		t.Fatalf("hidden modal progress after OnExit = %g, want 0", got)
	}
}

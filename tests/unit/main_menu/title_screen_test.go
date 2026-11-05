// Unit tests for Story 002: Title Screen & New Game Flow (Main Menu &
// Settings epic).
//
// Evidence path: production/epics/main-menu-settings/
// story-002-title-screen-new-game-flow.md
// Required evidence: this file must exist and pass (UI story gate).
//
// These tests drive src/ui/mainmenu/title_screen.go headlessly with the
// same injectable fakeClock as the Story 001 suite (shared package) — no
// file I/O, no engine host, no external APIs. Timing assertions follow the
// GDD measurement protocol: opacity = clamp(t_game / duration, 0, 1),
// exact and tolerance-free at the unit level.
//
// Acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_fade_in_background_at_timeline_points        -> TestTitleScreenFadeInBackgroundAtTimelinePoints
//	AC-1  test_menu_appears_after_300ms_delay               -> TestTitleScreenMenuAppearsAfter300msDelay
//	AC-1  test_buttons_non_interactive_while_prompt_active  -> TestTitleScreenButtonsNonInteractiveWhilePromptActive
//	AC-2  test_menu_buttons_registered_in_tab_order         -> TestTitleScreenMenuButtonsRegisteredInTabOrder
//	AC-2  test_version_label_present                        -> TestTitleScreenVersionLabelPresent
//	AC-2  test_menu_button_style_contract                   -> TestTitleScreenMenuButtonStyleContract
//	AC-3  test_new_game_without_saves_flows_to_begin        -> TestTitleScreenNewGameWithoutSavesFlowsToDifficultyAndBegin
//	AC-3  test_empty_name_falls_back_to_template            -> TestTitleScreenNewGameEmptyNameFallsBackToTemplate
//	AC-3  test_name_entry_cancel_returns_to_menu            -> TestTitleScreenNameEntryCancelReturnsToMenu
//	AC-4  test_new_game_with_saves_shows_confirmation       -> TestTitleScreenNewGameWithSavesShowsConfirmationDialog
//	AC-4  test_confirm_cancel_returns_to_menu               -> TestTitleScreenConfirmCancelReturnsToMenu
//	AC-4  test_confirm_begin_proceeds_to_difficulty         -> TestTitleScreenConfirmBeginProceedsToDifficulty
//	AC-5  test_new_game_disabled_while_saving               -> TestTitleScreenNewGameDisabledWhileSaving
//	AC-6  test_load_game_greyed_out_without_saves           -> TestTitleScreenLoadGameGreyedOutWithoutSaves
//	AC-6  test_load_game_transitions_to_load_screen         -> TestTitleScreenLoadGameTransitionsToLoadScreen
//	AC-6  test_load_game_enables_after_saves_created        -> TestTitleScreenLoadGameEnablesAfterSavesCreated
//	edge  test_settings_transitions_to_settings_screen      -> TestTitleScreenSettingsTransitionsToSettingsScreen
//	edge  test_quit_calls_quit_to_desktop                   -> TestTitleScreenQuitCallsQuitToDesktop
package main_menu_test

import (
	"testing"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// titleProbes collects the injected title screen probe/callback state so
// tests can flip probes and assert callback invocations.
type titleProbes struct {
	hasSaves   bool
	isSaving   bool
	beginCalls int
	beginDiff  mainmenu.Difficulty
	beginName  string
	quitCalls  int
}

// newTitleScreen builds a flow and a TitleScreen wired to the same fake
// clock with stub probes/callbacks. The title controller is registered for
// ScreenTitleScreen; other controllers register per test as needed.
func newTitleScreen(t *testing.T) (*titleProbes, *mainmenu.ScreenFlow, *mainmenu.TitleScreen, *fakeClock) {
	t.Helper()
	clk := &fakeClock{}
	flow := mainmenu.NewScreenFlow(mainmenu.NewTransitionSystem(clk.now))
	p := &titleProbes{}
	title := mainmenu.NewTitleScreen(flow,
		mainmenu.WithClock(clk.now),
		mainmenu.WithHasSaves(func() bool { return p.hasSaves }),
		mainmenu.WithIsSaving(func() bool { return p.isSaving }),
		mainmenu.WithBeginNewGame(func(d mainmenu.Difficulty, name string) {
			p.beginCalls++
			p.beginDiff = d
			p.beginName = name
		}),
		mainmenu.WithQuitToDesktop(func() { p.quitCalls++ }),
	)
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, title); err != nil {
		t.Fatalf("register title controller: %v", err)
	}
	return p, flow, title, clk
}

// click dispatches a mouse click on the element with the given id through
// the full flow path (translate → dispatch → focus + activate) and reports
// whether it was handled. Unhandled clicks (disabled elements) return
// false.
func click(flow *mainmenu.ScreenFlow, id string) bool {
	return flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id}))
}

// ---------------------------------------------------------------------------
// AC-1: First-launch fade-in sequence
// ---------------------------------------------------------------------------

func TestTitleScreenFadeInBackgroundAtTimelinePoints(t *testing.T) {
	_, _, title, clk := newTitleScreen(t)
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if op := title.BackgroundOpacity(); op != 0 {
		t.Fatalf("background opacity at t=0 = %g, want 0", op)
	}
	clk.advance(250)
	if op := title.BackgroundOpacity(); op != 0.5 {
		t.Fatalf("background opacity at t=250 = %g, want 0.5", op)
	}
	clk.advance(250)
	if op := title.BackgroundOpacity(); op != 1.0 {
		t.Fatalf("background opacity at t=500 = %g, want 1.0", op)
	}
}

func TestTitleScreenMenuAppearsAfter300msDelay(t *testing.T) {
	_, _, title, clk := newTitleScreen(t)
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if title.MenuVisible() {
		t.Fatal("menu must not be visible at t=0")
	}
	clk.advance(299)
	if title.MenuVisible() {
		t.Fatal("menu must not be visible at 299ms (delay is 300ms)")
	}
	clk.advance(1)
	if !title.MenuVisible() {
		t.Fatal("menu must appear at exactly 300ms")
	}
	if p := title.MenuRevealProgress(); p != 0 {
		t.Fatalf("menu reveal progress at t=300 = %g, want 0", p)
	}
	clk.advance(100)
	if p := title.MenuRevealProgress(); p != 0.5 {
		t.Fatalf("menu reveal progress at t=400 = %g, want 0.5", p)
	}
	clk.advance(100)
	if p := title.MenuRevealProgress(); p != 1.0 {
		t.Fatalf("menu reveal progress at t=500 = %g, want 1.0", p)
	}
}

func TestTitleScreenButtonsNonInteractiveWhilePromptActive(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true // otherwise probes would gate load-game as well
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("menu enabled count before prompt = %d, want 4", n)
	}

	// Story 003 hook: prompt appears → buttons visible (still registered)
	// but non-interactive.
	title.SetInteractive(false)
	if n := title.Focus().Count(); n != 11 {
		t.Fatalf("registered elements = %d, want 11 (buttons stay visible)", n)
	}
	if n := title.Focus().EnabledCount(); n != 0 {
		t.Fatalf("enabled elements while prompt active = %d, want 0", n)
	}
	if title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must not be focusable while the prompt is active")
	}
	if click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click must be ignored while the prompt is active")
	}
	if title.State() != mainmenu.TitleStateMenu {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateMenu)
	}

	// Prompt dismissed → all menu buttons interactive again.
	title.SetInteractive(true)
	if n := title.Focus().EnabledCount(); n != 4 {
		t.Fatalf("menu enabled count after dismissal = %d, want 4", n)
	}
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must be focusable after the prompt is dismissed")
	}
}

// ---------------------------------------------------------------------------
// AC-2: Menu buttons, version label, styling contract
// ---------------------------------------------------------------------------

func TestTitleScreenMenuButtonsRegisteredInTabOrder(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true
	title.OnEnter(mainmenu.ScreenTitleScreen)

	// OnEnter focuses the first menu button.
	if got := title.Focus().FocusedID(); got != mainmenu.FocusNewGame {
		t.Fatalf("initial focus = %q, want %q", got, mainmenu.FocusNewGame)
	}
	// Tab visits the remaining buttons in order and wraps from Quit back
	// to New Game.
	order := []string{
		mainmenu.FocusLoadGame,
		mainmenu.FocusSettings,
		mainmenu.FocusQuit,
		mainmenu.FocusNewGame,
	}
	for i, id := range order {
		if !flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"})) {
			t.Fatalf("tab %d not handled", i)
		}
		if got := title.Focus().FocusedID(); got != id {
			t.Fatalf("tab %d: focus = %q, want %q", i, got, id)
		}
	}
	// Arrows move within the menu group: Down from New Game → Load Game.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := title.Focus().FocusedID(); got != mainmenu.FocusLoadGame {
		t.Fatalf("down-arrow: focus = %q, want %q", got, mainmenu.FocusLoadGame)
	}
	// From Quit, Down wraps to New Game.
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Tab"}))
	if got := title.Focus().FocusedID(); got != mainmenu.FocusQuit {
		t.Fatalf("before down-arrow wrap: focus = %q, want %q", got, mainmenu.FocusQuit)
	}
	flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "ArrowDown"}))
	if got := title.Focus().FocusedID(); got != mainmenu.FocusNewGame {
		t.Fatalf("down-arrow wrap: focus = %q, want %q", got, mainmenu.FocusNewGame)
	}
}

func TestTitleScreenVersionLabelPresent(t *testing.T) {
	_, _, title, _ := newTitleScreen(t)
	want := "v" + mainmenu.GameVersion
	if got := title.Version(); got != want {
		t.Fatalf("version label = %q, want %q", got, want)
	}
}

func TestTitleScreenMenuButtonStyleContract(t *testing.T) {
	_, _, title, _ := newTitleScreen(t)
	s := title.MenuButtonStyle()
	if s.Background != mainmenu.ColorParchment {
		t.Errorf("button background = %#v, want parchment %#v", s.Background, mainmenu.ColorParchment)
	}
	if s.Text != mainmenu.ColorInk {
		t.Errorf("button text = %#v, want ink %#v", s.Text, mainmenu.ColorInk)
	}
	if s.HoverEdge != mainmenu.ColorAmber {
		t.Errorf("hover edge = %#v, want amber %#v", s.HoverEdge, mainmenu.ColorAmber)
	}
	if s.ActiveUnderline != mainmenu.ColorOxblood {
		t.Errorf("active underline = %#v, want oxblood %#v", s.ActiveUnderline, mainmenu.ColorOxblood)
	}
	if s.DisabledOpacity != mainmenu.DisabledButtonOpacity {
		t.Errorf("disabled opacity = %g, want %g", s.DisabledOpacity, mainmenu.DisabledButtonOpacity)
	}
}

// ---------------------------------------------------------------------------
// AC-3: New Game without saves → difficulty → name → begin
// ---------------------------------------------------------------------------

func TestTitleScreenNewGameWithoutSavesFlowsToDifficultyAndBegin(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = false
	title.OnEnter(mainmenu.ScreenTitleScreen)

	// New Game → difficulty selection (no saves: no confirmation dialog).
	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if title.State() != mainmenu.TitleStateDifficulty {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateDifficulty)
	}
	if title.SelectedDifficulty() != mainmenu.DifficultyNormal {
		t.Fatalf("default difficulty = %v, want %v", title.SelectedDifficulty(), mainmenu.DifficultyNormal)
	}
	if title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("menu buttons must not be focusable during difficulty selection")
	}
	if !title.Focus().FocusByID(mainmenu.FocusDifficultyTactical) {
		t.Fatal("difficulty options must be focusable during selection")
	}

	// Tactical → character name entry.
	if !click(flow, mainmenu.FocusDifficultyTactical) {
		t.Fatal("difficulty click not handled")
	}
	if title.State() != mainmenu.TitleStateNameEntry {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateNameEntry)
	}
	if title.SelectedDifficulty() != mainmenu.DifficultyTactical {
		t.Fatalf("selected difficulty = %v, want %v", title.SelectedDifficulty(), mainmenu.DifficultyTactical)
	}

	// Name + Begin → Loading state, callback with correct arguments.
	title.SetCharacterName("  Elias  ")
	if got := title.CharacterName(); got != "Elias" {
		t.Fatalf("character name = %q, want %q (trimmed)", got, "Elias")
	}
	if !click(flow, mainmenu.FocusNameBegin) {
		t.Fatal("Begin click not handled")
	}
	if flow.Current() != mainmenu.ScreenLoading {
		t.Fatalf("current screen = %s, want %s", flow.Current(), mainmenu.ScreenLoading)
	}
	if p.beginCalls != 1 {
		t.Fatalf("BeginNewGame calls = %d, want 1", p.beginCalls)
	}
	if p.beginDiff != mainmenu.DifficultyTactical {
		t.Fatalf("BeginNewGame difficulty = %v, want %v", p.beginDiff, mainmenu.DifficultyTactical)
	}
	if p.beginName != "Elias" {
		t.Fatalf("BeginNewGame name = %q, want %q", p.beginName, "Elias")
	}
}

func TestTitleScreenNewGameEmptyNameFallsBackToTemplate(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if !click(flow, mainmenu.FocusDifficultyNormal) {
		t.Fatal("Normal difficulty click not handled")
	}
	if title.State() != mainmenu.TitleStateNameEntry {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateNameEntry)
	}
	// No name entered: Begin uses the pre-made template name.
	if !click(flow, mainmenu.FocusNameBegin) {
		t.Fatal("Begin click not handled")
	}
	if p.beginCalls != 1 {
		t.Fatalf("BeginNewGame calls = %d, want 1", p.beginCalls)
	}
	if p.beginDiff != mainmenu.DifficultyNormal {
		t.Fatalf("BeginNewGame difficulty = %v, want %v", p.beginDiff, mainmenu.DifficultyNormal)
	}
	if p.beginName != mainmenu.DefaultCharacterName {
		t.Fatalf("BeginNewGame name = %q, want template %q", p.beginName, mainmenu.DefaultCharacterName)
	}
}

func TestTitleScreenNameEntryCancelReturnsToMenu(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if !click(flow, mainmenu.FocusDifficultyTactical) {
		t.Fatal("difficulty click not handled")
	}
	if !click(flow, mainmenu.FocusNameCancel) {
		t.Fatal("name cancel click not handled")
	}
	if title.State() != mainmenu.TitleStateMenu {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateMenu)
	}
	if p.beginCalls != 0 {
		t.Fatalf("BeginNewGame calls = %d, want 0", p.beginCalls)
	}
	// The menu is interactive again: New Game re-opens difficulty.
	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click after cancel not handled")
	}
	if title.State() != mainmenu.TitleStateDifficulty {
		t.Fatalf("state after re-click = %v, want %v", title.State(), mainmenu.TitleStateDifficulty)
	}
}

// ---------------------------------------------------------------------------
// AC-4: New Game with existing saves → confirmation dialog
// ---------------------------------------------------------------------------

func TestTitleScreenNewGameWithSavesShowsConfirmationDialog(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if title.State() != mainmenu.TitleStateConfirmNew {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateConfirmNew)
	}
	if got := mainmenu.NewGameConfirmMessage; got != "Starting a new game will not overwrite your existing saves." {
		t.Fatalf("confirmation message = %q, want the AC-4 text", got)
	}
	if title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("menu buttons must not be focusable while the dialog is open")
	}
	if !title.Focus().FocusByID(mainmenu.FocusConfirmBegin) {
		t.Fatal("Begin New Game must be focusable")
	}
	if !title.Focus().FocusByID(mainmenu.FocusConfirmCancel) {
		t.Fatal("Cancel must be focusable")
	}
	if p.beginCalls != 0 {
		t.Fatalf("BeginNewGame calls = %d, want 0 (dialog blocks the flow)", p.beginCalls)
	}
}

func TestTitleScreenConfirmCancelReturnsToMenu(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if !click(flow, mainmenu.FocusConfirmCancel) {
		t.Fatal("Cancel click not handled")
	}
	if title.State() != mainmenu.TitleStateMenu {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateMenu)
	}
	if p.beginCalls != 0 {
		t.Fatalf("BeginNewGame calls = %d, want 0", p.beginCalls)
	}
	// The dialog re-appears on the next New Game click.
	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click after cancel not handled")
	}
	if title.State() != mainmenu.TitleStateConfirmNew {
		t.Fatalf("state after re-click = %v, want %v", title.State(), mainmenu.TitleStateConfirmNew)
	}
}

func TestTitleScreenConfirmBeginProceedsToDifficulty(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if !click(flow, mainmenu.FocusConfirmBegin) {
		t.Fatal("Begin New Game click not handled")
	}
	if title.State() != mainmenu.TitleStateDifficulty {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateDifficulty)
	}
	if p.beginCalls != 0 {
		t.Fatalf("BeginNewGame calls = %d, want 0 (difficulty selection comes first)", p.beginCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-5: New Game disabled while a save write is in progress
// ---------------------------------------------------------------------------

func TestTitleScreenNewGameDisabledWhileSaving(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = true
	p.isSaving = true
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !title.NewGameDisabled() {
		t.Fatal("NewGameDisabled must report true while a save write is in progress")
	}
	if title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must not be focusable while a save write is in progress")
	}
	if n := title.Focus().EnabledCount(); n != 3 {
		t.Fatalf("enabled elements = %d, want 3 (load, settings, quit)", n)
	}
	if click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click must be ignored while a save write is in progress")
	}
	if title.State() != mainmenu.TitleStateMenu {
		t.Fatalf("state = %v, want %v", title.State(), mainmenu.TitleStateMenu)
	}

	// Save completes → probe flips; Refresh re-enables the button.
	p.isSaving = false
	title.Refresh()
	if title.NewGameDisabled() {
		t.Fatal("NewGameDisabled must clear after the save completes")
	}
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must be focusable after the save completes")
	}
	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click after save completion not handled")
	}
	if title.State() != mainmenu.TitleStateConfirmNew {
		t.Fatalf("state = %v, want %v (saves exist → confirmation dialog)", title.State(), mainmenu.TitleStateConfirmNew)
	}
}

// ---------------------------------------------------------------------------
// AC-6: Load Game greyed out when no saves exist
// ---------------------------------------------------------------------------

func TestTitleScreenLoadGameGreyedOutWithoutSaves(t *testing.T) {
	p, flow, title, _ := newTitleScreen(t)
	p.hasSaves = false
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if !title.LoadGameDisabled() {
		t.Fatal("LoadGameDisabled must report true when no saves exist")
	}
	if got := mainmenu.NoSavesTooltip; got != "No saved games found." {
		t.Fatalf("tooltip = %q, want the AC-6 text", got)
	}
	if op := title.MenuButtonStyle().DisabledOpacity; op != 0.4 {
		t.Fatalf("disabled opacity = %g, want 0.4 (AC-6 greyed-out)", op)
	}
	if title.Focus().FocusByID(mainmenu.FocusLoadGame) {
		t.Fatal("Load Game must not be focusable when no saves exist")
	}
	if click(flow, mainmenu.FocusLoadGame) {
		t.Fatal("Load Game click must be ignored when no saves exist")
	}
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current screen = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
	// New Game remains fully available.
	if !title.Focus().FocusByID(mainmenu.FocusNewGame) {
		t.Fatal("New Game must remain focusable when no saves exist")
	}
	if !click(flow, mainmenu.FocusNewGame) {
		t.Fatal("New Game click not handled")
	}
	if title.State() != mainmenu.TitleStateDifficulty {
		t.Fatalf("state = %v, want %v (New Game still works without saves)", title.State(), mainmenu.TitleStateDifficulty)
	}
}

func TestTitleScreenLoadGameTransitionsToLoadScreen(t *testing.T) {
	p, flow, title, clk := newTitleScreen(t)
	p.hasSaves = true
	// Entering the screen re-applies the probe state: the button was
	// disabled at construction, before the probe flipped.
	title.OnEnter(mainmenu.ScreenTitleScreen)
	if !click(flow, mainmenu.FocusLoadGame) {
		t.Fatal("Load Game click not handled")
	}
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatal("screen must not change until the crossfade completes")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("current screen = %s, want %s", flow.Current(), mainmenu.ScreenLoadGame)
	}
}

func TestTitleScreenLoadGameEnablesAfterSavesCreated(t *testing.T) {
	p, _, title, _ := newTitleScreen(t)
	p.hasSaves = false
	title.OnEnter(mainmenu.ScreenTitleScreen)

	if title.Focus().FocusByID(mainmenu.FocusLoadGame) {
		t.Fatal("Load Game must start disabled without saves")
	}
	// Saves are created while the screen is displayed (AC-6 edge case):
	// the per-frame Refresh re-polls the probe.
	p.hasSaves = true
	title.Refresh()
	if title.LoadGameDisabled() {
		t.Fatal("LoadGameDisabled must clear after saves are created")
	}
	if !title.Focus().FocusByID(mainmenu.FocusLoadGame) {
		t.Fatal("Load Game must be focusable after Refresh picks up the new saves")
	}
}

// ---------------------------------------------------------------------------
// Edge cases: Settings and Quit navigation
// ---------------------------------------------------------------------------

func TestTitleScreenSettingsTransitionsToSettingsScreen(t *testing.T) {
	_, flow, _, clk := newTitleScreen(t)
	if !click(flow, mainmenu.FocusSettings) {
		t.Fatal("Settings click not handled")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenSettings {
		t.Fatalf("current screen = %s, want %s", flow.Current(), mainmenu.ScreenSettings)
	}
}

func TestTitleScreenQuitCallsQuitToDesktop(t *testing.T) {
	p, flow, _, _ := newTitleScreen(t)
	if !click(flow, mainmenu.FocusQuit) {
		t.Fatal("Quit click not handled")
	}
	if p.quitCalls != 1 {
		t.Fatalf("QuitToDesktop calls = %d, want 1", p.quitCalls)
	}
}

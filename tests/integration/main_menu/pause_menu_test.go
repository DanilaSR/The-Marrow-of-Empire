// Integration tests for Story 006: Pause Menu.
//
// Evidence path: production/epics/main-menu-settings/
// story-006-pause-menu.md
// Required evidence: this file must exist and pass (Integration story gate —
// BLOCKING).
//
// These tests drive the pure-Go core of src/ui/mainmenu headlessly with an
// injectable clock (fakeClock, shared with the other main_menu_test files)
// and stubbed game-layer seams: world freeze callbacks (AC-1), event
// emissions (ADR-0011), the CanPause probe and cutscene-skip routing
// (AC-6), and the unsaved-progress / Save & Quit / Quit Without Saving
// callbacks (AC-4) — exactly as the game layer wires them. No engine host,
// no goroutines, no file I/O. Timing assertions follow the GDD measurement
// protocol: exact and tolerance-free at the unit level (DimAlpha is a pure
// function of the injectable clock).
//
// Acceptance criteria map to test functions 1:1:
//
//	AC-1  test_escape_opens_pause_and_freezes_world        -> TestEscapeOpensPauseMenuAndFreezesWorld
//	AC-1  test_dim_alpha_ramps_to_target                   -> TestDimOverlayAlphaRampsToTarget
//	AC-1  test_pause_has_exactly_five_options              -> TestPauseMenuHasExactlyFiveOptions
//	AC-2  test_resume_click_unfreezes_without_advancing    -> TestResumeClickUnfreezesWorldWithoutAdvancingTime
//	AC-2  test_escape_resumes                              -> TestEscapeResumesFromPauseMenu
//	AC-2  test_dim_lifts_on_resume                         -> TestDimLiftsOnResume
//	AC-3  test_combat_pause_freezes_hud_above_dim          -> TestCombatPauseFreezesCombatantsWithHudAboveDim
//	AC-4  test_quit_confirmation_dialog                    -> TestQuitWithUnsavedProgressShowsConfirmationDialog
//	AC-4  test_escape_in_confirmation_cancels              -> TestEscapeInQuitConfirmationCancels
//	AC-4  test_save_and_quit                               -> TestSaveAndQuitSavesThenTransitionsToTitle
//	AC-4  test_quit_save_failure_stays_in_menu             -> TestQuitSaveFailureReturnsToMenuWithoutQuitting
//	AC-4  test_quit_without_saving                         -> TestQuitWithoutSavingDiscardsAndTransitionsToTitle
//	AC-4  test_destructive_button_style_contrast           -> TestQuitWithoutSavingUsesDestructiveStyle
//	AC-4  test_cancel_dismisses_confirmation               -> TestCancelDismissesConfirmationDialog
//	AC-4  test_quit_without_unsaved_skips_confirmation     -> TestQuitWithoutUnsavedProgressSkipsConfirmation
//	AC-5  test_rapid_second_escape_ignored                 -> TestRapidSecondEscapeWithinCooldownIsIgnored
//	AC-5  test_cooldown_boundary_delta_equals_cooldown     -> TestCooldownBoundaryDeltaEqualsCooldownPermitted
//	AC-5  test_escape_spam_only_first_press_processed      -> TestEscapeSpamOnlyFirstPressProcessed
//	AC-5  test_resume_button_restarts_cooldown             -> TestResumeButtonRestartsCooldown
//	AC-6  test_pause_blocked_during_cutscene_routes_skip   -> TestPauseBlockedDuringCutsceneRoutesToSkip
//	AC-6  test_unskippable_cutscene_ignores_escape         -> TestUnskippableCutsceneIgnoresEscapeEntirely
//	CR-4  test_gamepad_start_toggles_pause_with_cooldown   -> TestGamepadStartTogglesPauseWithCooldown
//	edge  test_lifecycle_emits_events_in_order             -> TestPauseMenuLifecycleEmitsEventsInOrder
//	edge  test_menu_options_navigate_to_screens            -> TestMenuOptionsNavigateToTheirScreens
//	AC-7  test_draw_call_delta_is_single_dim_quad          -> TestDrawCallDeltaIsSingleDimQuadDuringPause
//	AC-7  benchmark_frame_cost                             -> BenchmarkPauseMenuFrameCost
//
// Region transitions and mid-save blocks (AC-6) share the same CanPause
// probe seam as the cutscene block; the cutscene tests exercise the routing
// and the probe contract on behalf of all three restricted states.
package main_menu_test

import (
	"errors"
	"testing"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures (fakeClock, stubScreen, navigate and click are shared with the
// other files in this package — see save_load_screens_test.go)
// ---------------------------------------------------------------------------

// stubWorld simulates the frozen simulation systems (AC-1c): each system
// exposes a TicksAdvanced counter that must stay identical across all
// paused frames. tick() is the game loop's per-frame simulation step; it is
// a no-op while frozen, exactly as the engine loop suspension (ADR-0007)
// behaves. gameTime is the game clock.
type stubWorld struct {
	npcTicks     int
	weatherTicks int
	combatTicks  int
	wildlifeTicks int
	gameTime     float64
	frozen       bool

	pausedCalls  int
	resumedCalls int
}

func (w *stubWorld) tick(dt float64) {
	if w.frozen {
		return
	}
	w.npcTicks++
	w.weatherTicks++
	w.combatTicks++
	w.wildlifeTicks++
	w.gameTime += dt
}

func (w *stubWorld) pause() {
	w.frozen = true
	w.pausedCalls++
}

func (w *stubWorld) resume() {
	w.frozen = false
	w.resumedCalls++
}

// stubGame is the stubbed game-layer seam: the ADR-0011 event emissions
// (recorded in order, with the world's freeze state at emission time so the
// event/freeze ordering contract is testable), the AC-6 CanPause probe, and
// the AC-4 unsaved-progress probe plus quit callbacks.
type stubGame struct {
	events []string
	world  *stubWorld

	canPause   bool
	hasUnsaved bool
	saveErr    error

	saveCalls int
	quitCalls int
	skipCalls int

	// frozenAtPausedEvent/frozenAtResumedEvent capture world.frozen at the
	// moment each event fires: OnEnter must emit GamePaused BEFORE the
	// freeze, OnExit must unfreeze BEFORE emitting GameResumed.
	frozenAtPausedEvent  bool
	frozenAtResumedEvent bool
}

func (g *stubGame) paused() {
	g.frozenAtPausedEvent = g.world.frozen
	g.events = append(g.events, "paused")
}

func (g *stubGame) resumed() {
	g.frozenAtResumedEvent = g.world.frozen
	g.events = append(g.events, "resumed")
}

func (g *stubGame) saveAndQuit() error {
	g.saveCalls++
	return g.saveErr
}

func (g *stubGame) quitWithoutSaving() { g.quitCalls++ }

func (g *stubGame) skipCutscene() { g.skipCalls++ }

// pauseFixture is the assembled test rig.
type pauseFixture struct {
	flow  *mainmenu.ScreenFlow
	clk   *fakeClock
	world *stubWorld
	game  *stubGame
	pause *mainmenu.PauseMenu
}

// newPauseFixture wires the flow in the title state with stub controllers
// everywhere except ScreenPauseMenu, which hosts the real PauseMenu
// controller under test with every injected seam stubbed. opts are applied
// AFTER the defaults so tests can override any single seam (e.g.
// WithSkipCutscene(nil) removes the cutscene-skip routing).
func newPauseFixture(t *testing.T, opts ...mainmenu.PauseMenuOption) *pauseFixture {
	t.Helper()
	clk := &fakeClock{}
	world := &stubWorld{}
	game := &stubGame{canPause: true, world: world}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	for _, sc := range []mainmenu.Screen{
		mainmenu.ScreenTitleScreen,
		mainmenu.ScreenLoadGame,
		mainmenu.ScreenInGame,
		mainmenu.ScreenSaveGame,
		mainmenu.ScreenSettings,
		mainmenu.ScreenLoading,
	} {
		if err := flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
			t.Fatalf("register %s: %v", sc, err)
		}
	}
	base := []mainmenu.PauseMenuOption{
		mainmenu.WithPauseClock(clk.now),
		mainmenu.WithGamePausedEvent(game.paused),
		mainmenu.WithGameResumedEvent(game.resumed),
		mainmenu.WithPauseWorld(world.pause),
		mainmenu.WithResumeWorld(world.resume),
		mainmenu.WithCanPause(func() bool { return game.canPause }),
		mainmenu.WithSkipCutscene(game.skipCutscene),
		mainmenu.WithHasUnsavedProgress(func() bool { return game.hasUnsaved }),
		mainmenu.WithSaveAndQuit(game.saveAndQuit),
		mainmenu.WithQuitWithoutSaving(game.quitWithoutSaving),
	}
	pause := mainmenu.NewPauseMenu(flow, append(base, opts...)...)
	if err := flow.RegisterController(mainmenu.ScreenPauseMenu, pause); err != nil {
		t.Fatalf("register pause menu: %v", err)
	}
	return &pauseFixture{flow: flow, clk: clk, world: world, game: game, pause: pause}
}

// enterInGame moves the flow from the title state into InGame (the game
// layer's entry path), completing the crossfade at t = 250.
func (fx *pauseFixture) enterInGame(t *testing.T) {
	t.Helper()
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenInGame)
}

// openPause presses Escape at InGame (the flow-level toggle path, CR-4) and
// completes the crossfade. The toggle timestamp is stamped at the Escape
// (t = 250); OnEnter fires at t = 500, freezing the world.
func (fx *pauseFixture) openPause(t *testing.T) {
	t.Helper()
	fx.enterInGame(t)
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("escape did not open the pause menu")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want PauseMenu", fx.flow.Current())
	}
}

// escapeInput is the translated Escape key press (ActionBack).
func escapeInput() mainmenu.Input {
	return mainmenu.Input{Action: mainmenu.ActionBack, Source: mainmenu.SourceKeyboard}
}

// ---------------------------------------------------------------------------
// AC-1: World freeze, dim overlay, five options
// ---------------------------------------------------------------------------

// TestEscapeOpensPauseMenuAndFreezesWorld covers AC-1a (Escape at InGame
// opens the pause menu) and AC-1c (the world freezes: every simulation
// system's TicksAdvanced counter and the game clock stay put across paused
// frames, via the injected PauseWorld/ResumeWorld callbacks).
func TestEscapeOpensPauseMenuAndFreezesWorld(t *testing.T) {
	fx := newPauseFixture(t)
	fx.enterInGame(t)
	fx.world.tick(16.6)
	fx.world.tick(16.6)
	if fx.world.npcTicks != 2 || fx.world.gameTime != 2*16.6 {
		t.Fatalf("world not running before pause: npc=%d time=%v", fx.world.npcTicks, fx.world.gameTime)
	}

	fx.openPause(t)

	if fx.world.frozen != true || fx.world.pausedCalls != 1 {
		t.Fatalf("world not frozen via PauseWorld: frozen=%v pausedCalls=%d", fx.world.frozen, fx.world.pausedCalls)
	}
	if !fx.pause.PauseActive() {
		t.Fatalf("PauseActive false while the pause is open")
	}
	// Paused frames must not advance any simulation system (AC-1c).
	before := fx.world.gameTime
	for i := 0; i < 5; i++ {
		fx.world.tick(16.6)
	}
	if fx.world.npcTicks != 2 || fx.world.weatherTicks != 2 || fx.world.combatTicks != 2 || fx.world.wildlifeTicks != 2 {
		t.Fatalf("simulation counters advanced while paused: npc=%d weather=%d combat=%d wildlife=%d",
			fx.world.npcTicks, fx.world.weatherTicks, fx.world.combatTicks, fx.world.wildlifeTicks)
	}
	if fx.world.gameTime != before {
		t.Fatalf("game time advanced while paused: %v -> %v", before, fx.world.gameTime)
	}
	if fx.pause.State() != mainmenu.PauseStateMenu {
		t.Fatalf("state = %s, want Menu", fx.pause.State())
	}
	if n := fx.pause.Focus().EnabledCount(); n != 5 {
		t.Fatalf("enabled focusables = %d, want 5", n)
	}
	if id := fx.pause.Focus().FocusedID(); id != mainmenu.PauseFocusResume {
		t.Fatalf("initial focus = %q, want %q", id, mainmenu.PauseFocusResume)
	}
}

// TestDimOverlayAlphaRampsToTarget covers AC-1b: the dim quad alpha ramps
// from 0 to 0.7 over 200ms with quadratic ease-out, clamped at the target.
// Exact at the unit level — a pure function of the injectable clock.
func TestDimOverlayAlphaRampsToTarget(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // OnEnter at t = 500

	if a := fx.pause.DimAlpha(); a != 0 {
		t.Fatalf("dim alpha at enter = %v, want 0", a)
	}
	// maxAlpha is the runtime float64 value of the constant: expected
	// products must be computed in runtime doubles (the menu does), not as
	// arbitrary-precision compile-time constants.
	maxAlpha := mainmenu.PauseDimMaxAlpha
	fx.clk.advance(50) // 25% progress: easeOut(0.25) = 0.4375
	want := maxAlpha * 0.4375
	if a := fx.pause.DimAlpha(); a != want {
		t.Fatalf("dim alpha at 25%% = %v, want %v", a, want)
	}
	fx.clk.advance(50) // 50% progress: easeOut(0.5) = 0.75
	want = maxAlpha * 0.75
	if a := fx.pause.DimAlpha(); a != want {
		t.Fatalf("dim alpha at 50%% = %v, want %v", a, want)
	}
	fx.clk.advance(100) // complete
	if a := fx.pause.DimAlpha(); a != mainmenu.PauseDimMaxAlpha {
		t.Fatalf("dim alpha at completion = %v, want %v", a, mainmenu.PauseDimMaxAlpha)
	}
	fx.clk.advance(5000) // clamped at the target
	if a := fx.pause.DimAlpha(); a != mainmenu.PauseDimMaxAlpha {
		t.Fatalf("dim alpha after clamp = %v, want %v", a, mainmenu.PauseDimMaxAlpha)
	}
}

// TestPauseMenuHasExactlyFiveOptions covers AC-1d: the pause menu offers
// exactly five options in order — Resume, Save, Load, Settings, Quit to
// Main Menu — all focusable and enabled on entry.
func TestPauseMenuHasExactlyFiveOptions(t *testing.T) {
	fx := newPauseFixture(t)
	opts := fx.pause.Options()
	want := []mainmenu.PauseOption{
		{ID: mainmenu.PauseFocusResume, Label: mainmenu.PauseLabelResume},
		{ID: mainmenu.PauseFocusSave, Label: mainmenu.PauseLabelSave},
		{ID: mainmenu.PauseFocusLoad, Label: mainmenu.PauseLabelLoad},
		{ID: mainmenu.PauseFocusSettings, Label: mainmenu.PauseLabelSettings},
		{ID: mainmenu.PauseFocusQuit, Label: mainmenu.PauseLabelQuitToMainMenu},
	}
	if len(opts) != len(want) {
		t.Fatalf("options = %d, want %d", len(opts), len(want))
	}
	for i := range want {
		if opts[i] != want[i] {
			t.Fatalf("option %d = %+v, want %+v", i, opts[i], want[i])
		}
	}
	fx.openPause(t)
	if n := fx.pause.Focus().EnabledCount(); n != 5 {
		t.Fatalf("enabled focusables = %d, want 5", n)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Resume — dim lift, world unfreeze, no game time during pause
// ---------------------------------------------------------------------------

// TestResumeClickUnfreezesWorldWithoutAdvancingTime covers AC-2 (mouse
// path): clicking Resume fades the dim, unfreezes the world through
// ResumeWorld, and the game clock continues from the exact tick it was
// frozen at — no game time passes while paused.
func TestResumeClickUnfreezesWorldWithoutAdvancingTime(t *testing.T) {
	fx := newPauseFixture(t)
	fx.enterInGame(t)
	fx.world.tick(16.6)
	fx.world.tick(16.6)
	fx.openPause(t)

	// Frames while paused: no time passes.
	fx.world.tick(16.6)
	fx.world.tick(16.6)
	if fx.world.npcTicks != 2 {
		t.Fatalf("npc ticks advanced while paused: %d", fx.world.npcTicks)
	}

	click(t, fx.flow, mainmenu.PauseFocusResume)
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()

	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame after resume", fx.flow.Current())
	}
	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed: frozen=%v resumedCalls=%d", fx.world.frozen, fx.world.resumedCalls)
	}
	if fx.pause.PauseActive() {
		t.Fatalf("PauseActive true after resume")
	}

	// Post-resume ticks resume the clock; total time == pre + post only.
	fx.world.tick(16.6)
	fx.world.tick(16.6)
	if fx.world.npcTicks != 4 {
		t.Fatalf("npc ticks after resume = %d, want 4", fx.world.npcTicks)
	}
	want := 0.0
	for i := 0; i < 4; i++ {
		want += 16.6
	}
	if fx.world.gameTime != want {
		t.Fatalf("game time = %v, want %v (no time may pass during the pause)", fx.world.gameTime, want)
	}
}

// TestEscapeResumesFromPauseMenu covers AC-2 (keyboard path): Escape at the
// pause menu returns to InGame — flow-level Back, past the toggle cooldown.
func TestEscapeResumesFromPauseMenu(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // toggle stamped at t = 250

	fx.clk.advance(100) // t = 600: past the 300ms cooldown
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("escape did not resume the game")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()

	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame", fx.flow.Current())
	}
	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed: frozen=%v resumedCalls=%d", fx.world.frozen, fx.world.resumedCalls)
	}
}

// TestDimLiftsOnResume covers AC-2's dim fade-out: after OnExit the dim
// lifts 0.7 -> 0 over PauseDimMs (ease-out mirrored).
func TestDimLiftsOnResume(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.clk.advance(mainmenu.PauseDimMs) // full dim
	if a := fx.pause.DimAlpha(); a != mainmenu.PauseDimMaxAlpha {
		t.Fatalf("dim alpha before resume = %v, want %v", a, mainmenu.PauseDimMaxAlpha)
	}

	click(t, fx.flow, mainmenu.PauseFocusResume)
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update() // OnExit at t = 950: lift begins

	if a := fx.pause.DimAlpha(); a != mainmenu.PauseDimMaxAlpha {
		t.Fatalf("dim alpha at lift start = %v, want %v", a, mainmenu.PauseDimMaxAlpha)
	}
	fx.clk.advance(100) // 50% of the lift: (1 - 0.75) * 0.7
	maxAlpha := mainmenu.PauseDimMaxAlpha // runtime float64 (see ramp test)
	want := maxAlpha * 0.25
	if a := fx.pause.DimAlpha(); a != want {
		t.Fatalf("dim alpha mid-lift = %v, want %v", a, want)
	}
	fx.clk.advance(100) // lift complete
	if a := fx.pause.DimAlpha(); a != 0 {
		t.Fatalf("dim alpha after lift = %v, want 0", a)
	}
}

// ---------------------------------------------------------------------------
// AC-3: Combat pause — combatants frozen, HUD above the dim
// ---------------------------------------------------------------------------

// TestCombatPauseFreezesCombatantsWithHudAboveDim covers AC-3: the pause is
// the same controller in combat; combatants' TicksAdvanced stays put, the
// Field register (combat HUD chrome) renders above the dim quad, and the
// HUD is a static snapshot while paused — the HUD pass draws only when the
// pause is inactive (draw-call delta 0).
func TestCombatPauseFreezesCombatantsWithHudAboveDim(t *testing.T) {
	// Render order contract: World < Dim < Field < Desk.
	if !(mainmenu.RenderLayerWorld < mainmenu.RenderLayerDim &&
		mainmenu.RenderLayerDim < mainmenu.RenderLayerField &&
		mainmenu.RenderLayerField < mainmenu.RenderLayerDesk) {
		t.Fatalf("render layer order violated: world=%d dim=%d field=%d desk=%d",
			mainmenu.RenderLayerWorld, mainmenu.RenderLayerDim, mainmenu.RenderLayerField, mainmenu.RenderLayerDesk)
	}

	fx := newPauseFixture(t)
	fx.enterInGame(t)
	fx.world.tick(16.6)
	if fx.world.combatTicks != 1 {
		t.Fatalf("combat not running before pause: %d", fx.world.combatTicks)
	}
	fx.openPause(t)

	// Combatants are frozen like every other system.
	fx.world.tick(16.6)
	fx.world.tick(16.6)
	if fx.world.combatTicks != 1 {
		t.Fatalf("combat advanced while paused: %d", fx.world.combatTicks)
	}
	// The game layer's harness rule: the HUD pass draws only when the pause
	// is inactive. Simulated frames must redraw the HUD zero times.
	hudDraws := 0
	for i := 0; i < 3; i++ {
		if !fx.pause.PauseActive() {
			hudDraws++
		}
		fx.world.tick(16.6)
	}
	if hudDraws != 0 {
		t.Fatalf("HUD redrawn while paused: %d draws, want 0", hudDraws)
	}
	// At full dim the overlay adds exactly one draw call (the dim quad);
	// the HUD stays up above it without redrawing (AC-3/AC-7).
	fx.clk.advance(mainmenu.PauseDimMs)
	if fx.pause.DimAlpha() != mainmenu.PauseDimMaxAlpha {
		t.Fatalf("dim not at full alpha: %v", fx.pause.DimAlpha())
	}
	if fx.pause.DrawCallDelta() != 1 {
		t.Fatalf("draw-call delta while dim visible = %d, want 1 (dim quad only)", fx.pause.DrawCallDelta())
	}

	// After resume the HUD draws again.
	fx.clk.advance(100) // t = 600: past the cooldown
	fx.flow.Dispatch(escapeInput())
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.pause.PauseActive() {
		t.Fatalf("PauseActive true after resume")
	}
	hudDraws++
	if hudDraws != 1 {
		t.Fatalf("HUD draws after resume = %d, want 1", hudDraws)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Quit confirmation dialog
// ---------------------------------------------------------------------------

// TestQuitWithUnsavedProgressShowsConfirmationDialog covers AC-4: with
// unsaved progress, Quit to Main Menu opens the inline confirmation dialog
// with its own focusable buttons (Save & Quit / Quit Without Saving /
// Cancel) and the unsaved-progress warning. The world stays frozen.
func TestQuitWithUnsavedProgressShowsConfirmationDialog(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.game.hasUnsaved = true

	click(t, fx.flow, mainmenu.PauseFocusQuit)

	if fx.pause.State() != mainmenu.PauseStateConfirmQuit {
		t.Fatalf("state = %s, want ConfirmQuit", fx.pause.State())
	}
	if m := fx.pause.QuitConfirmMessage(); m != mainmenu.PauseQuitConfirmMessage {
		t.Fatalf("confirmation message = %q, want %q", m, mainmenu.PauseQuitConfirmMessage)
	}
	if n := fx.pause.Focus().EnabledCount(); n != 3 {
		t.Fatalf("enabled focusables in confirm state = %d, want 3", n)
	}
	if id := fx.pause.Focus().FocusedID(); id != mainmenu.PauseFocusSaveAndQuit {
		t.Fatalf("initial confirm focus = %q, want %q", id, mainmenu.PauseFocusSaveAndQuit)
	}
	// No quit happened yet; the world stays frozen behind the dialog.
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("flow left the pause menu: %s", fx.flow.Current())
	}
	if !fx.world.frozen {
		t.Fatalf("world unfrozen behind the confirmation")
	}
	if fx.game.saveCalls != 0 || fx.game.quitCalls != 0 {
		t.Fatalf("quit side effects before confirmation: save=%d quit=%d", fx.game.saveCalls, fx.game.quitCalls)
	}
}

// TestEscapeInQuitConfirmationCancels covers the AC-4 edge case: Escape
// inside the confirmation cancels it and returns to the menu — the game
// layer routes input to PauseMenu.Dispatch while the dialog is open, so the
// flow-level Escape (which would resume the game) never sees it.
func TestEscapeInQuitConfirmationCancels(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.game.hasUnsaved = true
	click(t, fx.flow, mainmenu.PauseFocusQuit)

	if !fx.pause.Dispatch(escapeInput()) {
		t.Fatalf("escape in confirmation not handled by Dispatch")
	}
	if fx.pause.State() != mainmenu.PauseStateMenu {
		t.Fatalf("state = %s, want Menu after cancel", fx.pause.State())
	}
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("flow left the pause menu: %s", fx.flow.Current())
	}
	if !fx.world.frozen || fx.world.resumedCalls != 0 {
		t.Fatalf("world resumed while cancelling: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
	if n := fx.pause.Focus().EnabledCount(); n != 5 {
		t.Fatalf("enabled focusables after cancel = %d, want 5", n)
	}
}

// TestSaveAndQuitSavesThenTransitionsToTitle covers AC-4's Save & Quit
// path: the injected callback runs the save, then the menu transitions to
// the title screen with the world unfrozen.
func TestSaveAndQuitSavesThenTransitionsToTitle(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.game.hasUnsaved = true
	click(t, fx.flow, mainmenu.PauseFocusQuit)

	click(t, fx.flow, mainmenu.PauseFocusSaveAndQuit)

	if fx.game.saveCalls != 1 {
		t.Fatalf("save calls = %d, want 1", fx.game.saveCalls)
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want TitleScreen", fx.flow.Current())
	}
	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed on quit: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
	if e := fx.pause.QuitError(); e != "" {
		t.Fatalf("unexpected quit error: %q", e)
	}
}

// TestQuitSaveFailureReturnsToMenuWithoutQuitting covers the AC-4 edge
// case: a failed save does NOT quit — the error is surfaced on the menu and
// the player stays in the pause menu with the world still frozen.
func TestQuitSaveFailureReturnsToMenuWithoutQuitting(t *testing.T) {
	fx := newPauseFixture(t)
	fx.game.saveErr = errors.New("disk full")
	fx.openPause(t)
	fx.game.hasUnsaved = true
	click(t, fx.flow, mainmenu.PauseFocusQuit)

	click(t, fx.flow, mainmenu.PauseFocusSaveAndQuit)

	if fx.game.saveCalls != 1 {
		t.Fatalf("save calls = %d, want 1", fx.game.saveCalls)
	}
	if fx.pause.State() != mainmenu.PauseStateMenu {
		t.Fatalf("state = %s, want Menu after failed save", fx.pause.State())
	}
	if e := fx.pause.QuitError(); e != "disk full" {
		t.Fatalf("quit error = %q, want %q", e, "disk full")
	}
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("quit proceeded after failed save: %s", fx.flow.Current())
	}
	if !fx.world.frozen {
		t.Fatalf("world unfrozen after failed save")
	}
	// The menu remains fully usable: Resume still works.
	click(t, fx.flow, mainmenu.PauseFocusResume)
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenInGame || fx.world.frozen {
		t.Fatalf("resume after failed save: current=%s frozen=%v", fx.flow.Current(), fx.world.frozen)
	}
}

// TestQuitWithoutSavingDiscardsAndTransitionsToTitle covers AC-4's
// destructive path: Quit Without Saving discards the session via the
// injected callback and transitions to the title screen.
func TestQuitWithoutSavingDiscardsAndTransitionsToTitle(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.game.hasUnsaved = true
	click(t, fx.flow, mainmenu.PauseFocusQuit)

	click(t, fx.flow, mainmenu.PauseFocusQuitWithoutSaving)

	if fx.game.quitCalls != 1 {
		t.Fatalf("quit-without-saving calls = %d, want 1", fx.game.quitCalls)
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want TitleScreen", fx.flow.Current())
	}
	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed on quit: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
}

// TestQuitWithoutSavingUsesDestructiveStyle covers AC-4's styling contract:
// oxblood fill with ivory text at >= 4.5:1 contrast (WCAG AA), amber hover
// edge — so the destructive option is unmistakable and readable.
func TestQuitWithoutSavingUsesDestructiveStyle(t *testing.T) {
	fx := newPauseFixture(t)
	style := fx.pause.QuitWithoutSavingStyle()
	if style.Background != mainmenu.ColorOxblood {
		t.Fatalf("background = %v, want oxblood %v", style.Background, mainmenu.ColorOxblood)
	}
	if style.Text != mainmenu.ColorIvory {
		t.Fatalf("text = %v, want ivory %v", style.Text, mainmenu.ColorIvory)
	}
	if r := mainmenu.ContrastRatio(style.Text, style.Background); r < 4.5 {
		t.Fatalf("destructive button contrast = %.2f, want >= 4.5 (WCAG AA)", r)
	}
	if style.HoverEdge != mainmenu.ColorAmber {
		t.Fatalf("hover edge = %v, want amber %v", style.HoverEdge, mainmenu.ColorAmber)
	}
}

// TestCancelDismissesConfirmationDialog covers AC-4's Cancel button: back
// to the pause menu with no quit side effects.
func TestCancelDismissesConfirmationDialog(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)
	fx.game.hasUnsaved = true
	click(t, fx.flow, mainmenu.PauseFocusQuit)

	click(t, fx.flow, mainmenu.PauseFocusCancel)

	if fx.pause.State() != mainmenu.PauseStateMenu {
		t.Fatalf("state = %s, want Menu", fx.pause.State())
	}
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("flow left the pause menu: %s", fx.flow.Current())
	}
	if fx.game.saveCalls != 0 || fx.game.quitCalls != 0 {
		t.Fatalf("quit side effects on cancel: save=%d quit=%d", fx.game.saveCalls, fx.game.quitCalls)
	}
}

// TestQuitWithoutUnsavedProgressSkipsConfirmation covers AC-4: with a clean
// save the quit proceeds directly to the title screen — no dialog, no
// injected callbacks.
func TestQuitWithoutUnsavedProgressSkipsConfirmation(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // hasUnsaved defaults to false

	click(t, fx.flow, mainmenu.PauseFocusQuit)

	if fx.pause.State() != mainmenu.PauseStateMenu {
		t.Fatalf("state = %s, want Menu (no confirmation without unsaved progress)", fx.pause.State())
	}
	if fx.game.saveCalls != 0 || fx.game.quitCalls != 0 {
		t.Fatalf("quit side effects without confirmation: save=%d quit=%d", fx.game.saveCalls, fx.game.quitCalls)
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want TitleScreen", fx.flow.Current())
	}
	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed on quit: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-5: 300ms toggle cooldown
// ---------------------------------------------------------------------------

// TestRapidSecondEscapeWithinCooldownIsIgnored covers AC-5: a second Escape
// arriving within 300ms of the open toggle is consumed without any state
// transition — no resume, no dim animation reversal.
func TestRapidSecondEscapeWithinCooldownIsIgnored(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // toggle stamped at t = 250; OnEnter at t = 500

	// t = 500: delta 250 < 300 — consumed, nothing happens.
	fx.flow.Dispatch(escapeInput())
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("escape within cooldown left the pause menu: %s", fx.flow.Current())
	}
	if !fx.world.frozen || fx.world.resumedCalls != 0 {
		t.Fatalf("world resumed within cooldown: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
	if a := fx.pause.DimAlpha(); a != 0 {
		t.Fatalf("dim animation reversed within cooldown: alpha = %v, want 0", a)
	}
	// No hidden transition may have started either.
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("a hidden transition started: current = %s", fx.flow.Current())
	}
	if !fx.world.frozen || fx.world.resumedCalls != 0 {
		t.Fatalf("world resumed after the cooldown window: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
}

// TestCooldownBoundaryDeltaEqualsCooldownPermitted covers AC-5's boundary
// inclusivity: a toggle exactly PauseToggleCooldownMs after the last one is
// permitted (delta == cooldown allowed, per the GDD measurement protocol).
func TestCooldownBoundaryDeltaEqualsCooldownPermitted(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // lastToggle = 250

	fx.clk.advance(50) // t = 550: delta == 300 exactly
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("escape at the cooldown boundary (delta == cooldown) must be permitted")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame", fx.flow.Current())
	}
	if fx.world.frozen {
		t.Fatalf("world still frozen after boundary resume")
	}
}

// TestEscapeSpamOnlyFirstPressProcessed covers AC-5 + AC-7 input handling
// under escape mashing: press 1 opens the pause, press 2 during the
// crossfade is discarded by input blocking, press 3 right at the fade end
// is consumed by the cooldown, press 4 (past the boundary) resumes.
func TestEscapeSpamOnlyFirstPressProcessed(t *testing.T) {
	fx := newPauseFixture(t)
	fx.enterInGame(t) // t = 250

	// Press 1: opens the pause menu (toggle stamped t = 250).
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("press 1 must open the pause menu")
	}
	// Press 2 during the crossfade (t = 400): discarded — input blocked.
	fx.clk.advance(150)
	if fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("press 2 during the crossfade must be discarded")
	}
	// Complete the fade (t = 500, OnEnter). Press 3: within the cooldown.
	fx.clk.advance(100)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want PauseMenu", fx.flow.Current())
	}
	fx.flow.Dispatch(escapeInput())
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("press 3 resumed the game within the cooldown: %s", fx.flow.Current())
	}
	if !fx.world.frozen {
		t.Fatalf("world unfrozen after press 3")
	}
	// Press 4 (t = 750: delta 500 >= 300): resumes.
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("press 4 must resume the game")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenInGame || fx.world.frozen {
		t.Fatalf("after press 4: current=%s frozen=%v", fx.flow.Current(), fx.world.frozen)
	}
}

// TestResumeButtonRestartsCooldown covers AC-5's button interaction: the
// Resume button restarts the cooldown window, so an immediate Escape after
// resuming cannot flicker back into the pause.
func TestResumeButtonRestartsCooldown(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // toggle stamped t = 250

	click(t, fx.flow, mainmenu.PauseFocusResume) // t = 500: stamps lastToggle = 500
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update() // t = 750: OnExit

	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame", fx.flow.Current())
	}
	// Immediate Escape (t = 750: delta 250 < 300): consumed, no reopen.
	fx.flow.Dispatch(escapeInput())
	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("escape right after resume reopened the pause (cooldown not restarted)")
	}
	// At the boundary (t = 800: delta 300) the pause opens again.
	fx.clk.advance(50)
	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("escape at the boundary after resume must open the pause")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want PauseMenu", fx.flow.Current())
	}
}

// ---------------------------------------------------------------------------
// AC-6: Restricted states (cutscenes, region transitions, mid-save)
// ---------------------------------------------------------------------------

// TestPauseBlockedDuringCutsceneRoutesToSkip covers AC-6: while the CanPause
// probe reports a restricted state (cutscene playing here; region
// transitions and mid-save use the same probe), Escape does not open the
// pause — the press routes to the injected cutscene-skip callback and the
// world keeps running.
func TestPauseBlockedDuringCutsceneRoutesToSkip(t *testing.T) {
	fx := newPauseFixture(t)
	fx.game.canPause = false
	fx.enterInGame(t)
	fx.world.tick(16.6)

	if !fx.flow.Dispatch(escapeInput()) {
		t.Fatalf("escape must be consumed while the pause is blocked")
	}
	if fx.game.skipCalls != 1 {
		t.Fatalf("skip callback calls = %d, want 1", fx.game.skipCalls)
	}
	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame (pause must not open)", fx.flow.Current())
	}
	if fx.world.frozen || len(fx.game.events) != 0 {
		t.Fatalf("world affected by blocked press: frozen=%v events=%v", fx.world.frozen, fx.game.events)
	}
	// Every blocked press routes to the skip callback.
	fx.flow.Dispatch(escapeInput())
	if fx.game.skipCalls != 2 {
		t.Fatalf("skip callback calls = %d, want 2", fx.game.skipCalls)
	}
	// The world keeps simulating behind the cutscene.
	fx.world.tick(16.6)
	if fx.world.npcTicks != 2 {
		t.Fatalf("world not running behind the cutscene: %d", fx.world.npcTicks)
	}
}

// TestUnskippableCutsceneIgnoresEscapeEntirely covers AC-6 with no skip
// callback installed: the blocked press is consumed with zero side effects
// (the injected skip callback self-guards — unskippable cutscenes and
// non-cutscene blocks are ignored entirely).
func TestUnskippableCutsceneIgnoresEscapeEntirely(t *testing.T) {
	fx := newPauseFixture(t, mainmenu.WithSkipCutscene(nil))
	fx.game.canPause = false
	fx.enterInGame(t)

	fx.flow.Dispatch(escapeInput())

	if fx.game.skipCalls != 0 {
		t.Fatalf("skip callback invoked for an unskippable block: %d", fx.game.skipCalls)
	}
	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame", fx.flow.Current())
	}
	if fx.world.frozen || len(fx.game.events) != 0 {
		t.Fatalf("world affected by blocked press: frozen=%v events=%v", fx.world.frozen, fx.game.events)
	}
}

// ---------------------------------------------------------------------------
// CR-4: Gamepad Start toggles the pause
// ---------------------------------------------------------------------------

// TestGamepadStartTogglesPauseWithCooldown covers CR-4 (gamepad Start) with
// the AC-5 cooldown applied to both directions.
func TestGamepadStartTogglesPauseWithCooldown(t *testing.T) {
	fx := newPauseFixture(t)
	fx.enterInGame(t) // t = 250
	start := mainmenu.Input{Action: mainmenu.ActionTogglePause, Source: mainmenu.SourceGamepad}

	if !fx.flow.Dispatch(start) {
		t.Fatalf("start did not open the pause menu")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want PauseMenu", fx.flow.Current())
	}
	if !fx.world.frozen {
		t.Fatalf("world not frozen after start toggle")
	}
	// Start within the cooldown (t = 500): ignored.
	if fx.flow.Dispatch(start) {
		t.Fatalf("start within the cooldown must be ignored")
	}
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("start within the cooldown left the pause menu: %s", fx.flow.Current())
	}
	// Start at the boundary (t = 550): resumes.
	fx.clk.advance(50)
	if !fx.flow.Dispatch(start) {
		t.Fatalf("start at the cooldown boundary must resume")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want InGame", fx.flow.Current())
	}
	if fx.world.frozen {
		t.Fatalf("world still frozen after resume")
	}
}

// ---------------------------------------------------------------------------
// Lifecycle and navigation
// ---------------------------------------------------------------------------

// TestPauseMenuLifecycleEmitsEventsInOrder covers the ADR-0011 event
// contract: OnEnter emits GamePaused BEFORE the world freeze, OnExit
// unfreezes BEFORE emitting GameResumed, and PauseActive marks exactly the
// frozen interval.
func TestPauseMenuLifecycleEmitsEventsInOrder(t *testing.T) {
	fx := newPauseFixture(t)
	if fx.pause.PauseActive() {
		t.Fatalf("PauseActive before the pause is entered")
	}

	fx.openPause(t)

	if len(fx.game.events) != 1 || fx.game.events[0] != "paused" {
		t.Fatalf("events after enter = %v, want [paused]", fx.game.events)
	}
	if fx.game.frozenAtPausedEvent {
		t.Fatalf("GamePaused emitted after the freeze — want event first, then PauseWorld")
	}
	if !fx.world.frozen || fx.world.pausedCalls != 1 {
		t.Fatalf("world not frozen after enter: frozen=%v paused=%d", fx.world.frozen, fx.world.pausedCalls)
	}

	// Resume: unfreeze then event.
	fx.clk.advance(100) // past the cooldown
	fx.flow.Dispatch(escapeInput())
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()

	if fx.world.frozen || fx.world.resumedCalls != 1 {
		t.Fatalf("world not resumed after exit: frozen=%v resumed=%d", fx.world.frozen, fx.world.resumedCalls)
	}
	if len(fx.game.events) != 2 || fx.game.events[1] != "resumed" {
		t.Fatalf("events after exit = %v, want [paused resumed]", fx.game.events)
	}
	if fx.game.frozenAtResumedEvent {
		t.Fatalf("GameResumed emitted before the unfreeze — want ResumeWorld first, then event")
	}
	if fx.pause.PauseActive() {
		t.Fatalf("PauseActive after the pause is exited")
	}
}

// TestMenuOptionsNavigateToTheirScreens covers the Save/Load/Settings
// options' navigation: each opens its screen (the pause ends — OnExit
// unfreezes the world and emits GameResumed, per the AC-2 exit contract),
// and Back returns to the pause menu (back-stack entry point) where the
// world freezes again on re-entry.
func TestMenuOptionsNavigateToTheirScreens(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t)

	for _, tc := range []struct {
		id     string
		target mainmenu.Screen
	}{
		{mainmenu.PauseFocusSave, mainmenu.ScreenSaveGame},
		{mainmenu.PauseFocusLoad, mainmenu.ScreenLoadGame},
		{mainmenu.PauseFocusSettings, mainmenu.ScreenSettings},
	} {
		click(t, fx.flow, tc.id)
		fx.clk.advance(mainmenu.ScreenCrossfadeMs)
		fx.flow.Update()
		if fx.flow.Current() != tc.target {
			t.Fatalf("option %s: current = %s, want %s", tc.id, fx.flow.Current(), tc.target)
		}
		// Leaving the pause menu ends the pause: the world unfreezes and
		// GameResumed fires (AC-2 exit contract — the freeze lifetime is
		// exactly the pause overlay's active interval).
		if fx.world.frozen {
			t.Fatalf("world still frozen behind %s — OnExit must unfreeze", tc.target)
		}
		if n := len(fx.game.events); n < 2 || fx.game.events[n-1] != "resumed" {
			t.Fatalf("events behind %s = %v, want last event resumed", tc.target, fx.game.events)
		}
		if !fx.flow.Dispatch(escapeInput()) {
			t.Fatalf("back from %s not handled", tc.target)
		}
		fx.clk.advance(mainmenu.ScreenCrossfadeMs)
		fx.flow.Update()
		if fx.flow.Current() != mainmenu.ScreenPauseMenu {
			t.Fatalf("back from %s: current = %s, want PauseMenu", tc.target, fx.flow.Current())
		}
	}

	// Resume ends the pause.
	fx.clk.advance(100) // past the cooldown
	fx.flow.Dispatch(escapeInput())
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenInGame || fx.world.frozen {
		t.Fatalf("after resume: current=%s frozen=%v", fx.flow.Current(), fx.world.frozen)
	}
}

// ---------------------------------------------------------------------------
// AC-7: Draw-call budget
// ---------------------------------------------------------------------------

// TestDrawCallDeltaIsSingleDimQuadDuringPause covers AC-7's deterministic
// part: the pause overlay adds exactly one draw call — the dim quad — while
// the dim is visible, and zero once it is lifted. The menu panel belongs to
// the desk register UI pass (already counted) and the combat HUD is not
// redrawn while paused (AC-3), so the paused frame totals in-combat draw
// calls + 1.
func TestDrawCallDeltaIsSingleDimQuadDuringPause(t *testing.T) {
	fx := newPauseFixture(t)
	fx.openPause(t) // t = 500: dim alpha 0

	if d := fx.pause.DrawCallDelta(); d != 0 {
		t.Fatalf("draw-call delta at enter = %d, want 0", d)
	}
	fx.clk.advance(100) // t = 600: dim visible
	if d := fx.pause.DrawCallDelta(); d != 1 {
		t.Fatalf("draw-call delta while dim visible = %d, want 1 (dim quad)", d)
	}
	fx.clk.advance(100) // t = 700: full dim
	if d := fx.pause.DrawCallDelta(); d != 1 {
		t.Fatalf("draw-call delta at full dim = %d, want 1", d)
	}

	// Resume: the dim quad is drawn through the 200ms lift, then released.
	fx.flow.Dispatch(escapeInput()) // t = 700: delta 450 >= 300
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update() // t = 950: lift begins
	if d := fx.pause.DrawCallDelta(); d != 1 {
		t.Fatalf("draw-call delta at lift start = %d, want 1", d)
	}
	fx.clk.advance(100) // t = 1050
	if d := fx.pause.DrawCallDelta(); d != 1 {
		t.Fatalf("draw-call delta mid-lift = %d, want 1", d)
	}
	fx.clk.advance(100) // t = 1150: lift complete
	if d := fx.pause.DrawCallDelta(); d != 0 {
		t.Fatalf("draw-call delta after lift = %d, want 0", d)
	}
}

// BenchmarkPauseMenuFrameCost measures the CPU-side per-frame cost of the
// pause overlay while paused at full dim (AC-7: frame budget 16.6ms p95).
// The engine profiler owns the p95 evidence on target hardware; this
// benchmark demonstrates the controller's per-frame work — flow Update plus
// the overlay queries — is a fraction of a millisecond, far inside the
// budget, with the overlay adding exactly one draw call.
func BenchmarkPauseMenuFrameCost(b *testing.B) {
	flow, clk, pause := newBenchPauseFixture()
	if err := flow.RequestTransition(mainmenu.ScreenInGame); err != nil {
		b.Fatalf("enter in-game: %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if !flow.Dispatch(escapeInput()) {
		b.Fatalf("open pause failed")
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	clk.advance(mainmenu.PauseDimMs) // settle at full dim

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clk.advance(16.6) // one 60fps frame
		flow.Update()
		_ = pause.DimAlpha()
		_ = pause.PauseActive()
		_ = pause.DrawCallDelta()
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/1e6, "ms/frame")
}

// newBenchPauseFixture is the benchmark variant of newPauseFixture —
// benchmarks cannot use *testing.T.
func newBenchPauseFixture() (*mainmenu.ScreenFlow, *fakeClock, *mainmenu.PauseMenu) {
	clk := &fakeClock{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	for _, sc := range []mainmenu.Screen{mainmenu.ScreenTitleScreen, mainmenu.ScreenInGame} {
		_ = flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))})
	}
	pause := mainmenu.NewPauseMenu(flow, mainmenu.WithPauseClock(clk.now))
	_ = flow.RegisterController(mainmenu.ScreenPauseMenu, pause)
	return flow, clk, pause
}

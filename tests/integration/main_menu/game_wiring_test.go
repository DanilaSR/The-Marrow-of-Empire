// Integration tests for Story 008: Game-Layer Wiring — Title, Settings &
// Keybindings Live.
//
// Evidence path: production/epics/main-menu-settings/
// story-008-game-layer-wiring.md
// Required evidence: this file must exist and pass (Integration story gate —
// BLOCKING for /story-done).
//
// These tests drive the pure-Go core headlessly with the injectable clock
// (fakeClock, shared with save_load_screens_test.go) and the REAL
// settings.cfg adapter (src/game/settings_cfg.go) against temp directories —
// the same adapter the game layer wires into the screens. No engine host, no
// external APIs; file I/O is confined to t.TempDir().
//
// Acceptance criteria map to test functions 1:1:
//
//	AC-1  boot -> production title                    -> TestBootStartsAtProductionTitleScreen
//	AC-1  New Game continues to the slice world path  -> TestNewGameFlowReachesLoadingAndInGame
//	AC-2  screen change -> settings.cfg round-trip    -> TestSettingsChangePersistsToSettingsCfgFile
//	AC-2  missing file -> defaults, no error          -> TestMissingSettingsCfgFallsBackToDefaults
//	AC-2  corrupt file -> defaults, no error dialog   -> TestCorruptSettingsCfgFallsBackToDefaults
//	AC-2  rebind persists across adapter reload       -> TestKeybindingRebindPersistsAcrossAdapterReload
//	AC-3  capture state -> key consumed               -> TestRouteRawKeyCaptureConsumesAndBinds
//	AC-3  panel state -> false passthrough to flow    -> TestRouteRawKeyPanelStatePassesThroughToFlow
//	AC-3  reserved-warn: Escape consumed, rest pass   -> TestRouteRawKeyReservedWarnOnlyEscapeConsumed
//	AC-4  capture consumes nav keys (no leakage)      -> TestCaptureConsumesNavKeyWithoutFocusLeak
//	AC-5  Settings -> Keybindings with real adapter   -> TestSettingsToKeybindingsNavigationWithRealAdapter
//	AC-5  in-game -> pause -> settings -> back chain  -> TestInGamePauseSettingsKeybindingsBackChain
package main_menu_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"marrow-of-empire/game"
	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures
//
// fakeClock, stubScreen, navigate, click and clickIgnored are shared with
// save_load_screens_test.go / settings_screen_test.go / keybindings_test.go
// (same package main_menu_test).
// ---------------------------------------------------------------------------

// wiringFixture bundles the Story 008 pieces: the flow, the real settings.cfg
// adapter, and the real production screens wired to it exactly as the game
// layer wires them (WithSettingsReader/Writer/HasSettingsFile and
// WithKeybindingsReader/Writer/HasSettingsFile over one shared adapter).
type wiringFixture struct {
	flow   *mainmenu.ScreenFlow
	clk    *fakeClock
	adapter *game.SettingsCfg

	title      *mainmenu.TitleScreen
	settings   *mainmenu.SettingsScreen
	keybind    *mainmenu.KeybindingsScreen
	pause      *mainmenu.PauseMenu

	begun      []beginRecord
	quitCalls  int
	applied    []keyValue
	logs       []string
	changes    []kbChange
}

type beginRecord struct {
	difficulty mainmenu.Difficulty
	name       string
}

// newWiringFixture builds the complete production flow in the boot state:
// ScreenTitleScreen with the real TitleScreen controller, plus real
// Settings/Keybindings/PauseMenu controllers all sharing one real adapter at
// a temp settings.cfg path.
func newWiringFixture(t *testing.T) *wiringFixture {
	t.Helper()
	clk := &fakeClock{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts)
	adapter := game.NewSettingsCfg(filepath.Join(t.TempDir(), "settings.cfg"))

	fx := &wiringFixture{flow: flow, clk: clk, adapter: adapter}

	title := mainmenu.NewTitleScreen(flow,
		mainmenu.WithClock(clk.now),
		mainmenu.WithHasSaves(func() bool { return false }),
		mainmenu.WithBeginNewGame(func(d mainmenu.Difficulty, name string) {
			fx.begun = append(fx.begun, beginRecord{d, name})
		}),
		mainmenu.WithQuitToDesktop(func() { fx.quitCalls++ }),
	)
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, title); err != nil {
		t.Fatalf("register title: %v", err)
	}
	fx.title = title

	settings := mainmenu.NewSettingsScreen(flow,
		mainmenu.WithSettingsClock(clk.now),
		mainmenu.WithSettingsReader(adapter.Get),
		mainmenu.WithSettingsWriter(adapter.SetAndFlush),
		mainmenu.WithHasSettingsFile(adapter.HasFile),
		mainmenu.WithSettingsLogger(func(format string, args ...any) {
			fx.logs = append(fx.logs, fmt.Sprintf(format, args...))
		}),
		mainmenu.WithApplySetting(func(key, value string) {
			fx.applied = append(fx.applied, keyValue{key, value})
		}),
	)
	settings.CheckFirstLaunch()
	if err := flow.RegisterController(mainmenu.ScreenSettings, settings); err != nil {
		t.Fatalf("register settings: %v", err)
	}
	fx.settings = settings

	keybind := mainmenu.NewKeybindingsScreen(flow,
		mainmenu.WithKeybindingsClock(clk.now),
		mainmenu.WithKeybindingsReader(adapter.Get),
		mainmenu.WithKeybindingsWriter(adapter.SetAndFlush),
		mainmenu.WithKeybindingsHasSettingsFile(adapter.HasFile),
		mainmenu.WithKeybindingsLogger(func(format string, args ...any) {
			fx.logs = append(fx.logs, fmt.Sprintf(format, args...))
		}),
		mainmenu.WithBindingChange(func(action string, slot mainmenu.BindingSlot, chord mainmenu.KeyChord) {
			fx.changes = append(fx.changes, kbChange{action: action, slot: slot, chord: chord})
		}),
	)
	if err := flow.RegisterController(mainmenu.ScreenKeybindings, keybind); err != nil {
		t.Fatalf("register keybindings: %v", err)
	}
	// The game layer runs Load at boot (the AC-8 boot read); with no
	// settings.cfg yet this marks the default keys dirty so the first flush
	// creates the file.
	keybind.Load()
	fx.keybind = keybind

	pause := mainmenu.NewPauseMenu(flow, mainmenu.WithPauseClock(clk.now))
	if err := flow.RegisterController(mainmenu.ScreenPauseMenu, pause); err != nil {
		t.Fatalf("register pause: %v", err)
	}
	fx.pause = pause

	// InGame and LoadGame/SaveGame are stub screens (the slice world and the
	// INT-6 file operations are not part of this story's headless scope).
	for _, sc := range []mainmenu.Screen{mainmenu.ScreenInGame, mainmenu.ScreenLoadGame, mainmenu.ScreenSaveGame} {
		if err := flow.RegisterController(sc, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
			t.Fatalf("register %s: %v", sc, err)
		}
	}
	return fx
}

// navigateToKeybindings reaches the Keybindings screen the way the game does:
// Title -> Settings -> Keybindings (the ONLY path per the GDD transition
// table).
func wiringNavigateToKeybindings(t *testing.T, fx *wiringFixture) {
	t.Helper()
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenKeybindings)
}

// startCapture clicks an action slot and asserts capture mode for it.
func wiringStartCapture(t *testing.T, fx *wiringFixture, slotID string, action mainmenu.ActionBinding, slot mainmenu.BindingSlot) {
	t.Helper()
	click(t, fx.flow, slotID)
	if fx.keybind.State() != mainmenu.KeybindingsStateCapture {
		t.Fatalf("state = %s, want Capture", fx.keybind.State())
	}
	if !fx.keybind.CapturesInput() {
		t.Fatalf("CapturesInput = false while capturing")
	}
}

// ---------------------------------------------------------------------------
// AC-1 — the standalone build boots into the production title screen
// ---------------------------------------------------------------------------

// TestBootStartsAtProductionTitleScreen verifies the flow boots in the GDD
// initial state with the real TitleScreen controller registered and its menu
// interactive.
func TestBootStartsAtProductionTitleScreen(t *testing.T) {
	fx := newWiringFixture(t)
	if fx.flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("boot screen = %s, want TitleScreen", fx.flow.Current())
	}
	if fx.title.State() != mainmenu.TitleStateMenu {
		t.Fatalf("title state = %d, want Menu", fx.title.State())
	}
	if fx.title.Focus() == nil || fx.title.Focus().Count() == 0 {
		t.Fatalf("title screen has no focusables registered")
	}
	// No valid settings.cfg exists yet: the settings screen must flag the
	// first-launch prompt path (AC-2: defaults, no error).
	if !fx.settings.FirstLaunchPending() {
		t.Fatalf("first-launch flag not set with no settings.cfg")
	}
}

// TestNewGameFlowReachesLoadingAndInGame verifies the production title's New
// Game path: menu -> difficulty -> name -> Loading (hard cut) -> the slice
// world's begin callback -> InGame.
func TestNewGameFlowReachesLoadingAndInGame(t *testing.T) {
	fx := newWiringFixture(t)

	// Menu -> difficulty selection (no save files exist, so no confirm).
	click(t, fx.flow, mainmenu.FocusNewGame)
	if fx.title.State() != mainmenu.TitleStateDifficulty {
		t.Fatalf("state = %d, want Difficulty", fx.title.State())
	}
	click(t, fx.flow, mainmenu.FocusDifficultyTactical)
	if fx.title.State() != mainmenu.TitleStateNameEntry {
		t.Fatalf("state = %d, want NameEntry", fx.title.State())
	}
	click(t, fx.flow, mainmenu.FocusNameBegin)

	// onNameBegin hard-cuts into Loading and invokes the game-layer callback.
	if fx.flow.Current() != mainmenu.ScreenLoading {
		t.Fatalf("current = %s, want Loading", fx.flow.Current())
	}
	if len(fx.begun) != 1 {
		t.Fatalf("beginNewGame calls = %d, want 1", len(fx.begun))
	}
	if fx.begun[0].difficulty != mainmenu.DifficultyTactical {
		t.Fatalf("difficulty = %s, want Tactical", fx.begun[0].difficulty)
	}
	if fx.begun[0].name != mainmenu.DefaultCharacterName {
		t.Fatalf("name = %q, want %q", fx.begun[0].name, mainmenu.DefaultCharacterName)
	}

	// The game layer resolves Loading -> InGame after the world is built
	// (slice world construction is engine-coupled; the flow contract is
	// what is asserted here).
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenInGame)
}

// ---------------------------------------------------------------------------
// AC-2 — settings.cfg file adapter
// ---------------------------------------------------------------------------

// TestSettingsChangePersistsToSettingsCfgFile drives the real screens over
// the real adapter: a change cycles a value, the debounce flush writes the
// file, and a fresh adapter at the same path reads the same value.
func TestSettingsChangePersistsToSettingsCfgFile(t *testing.T) {
	fx := newWiringFixture(t)
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)

	const key = mainmenu.SettingDifficulty
	before := fx.settings.Setting(key).FormatValue()
	click(t, fx.flow, "setting-gameplay-difficulty")
	after := fx.settings.Setting(key).FormatValue()
	if after == before {
		t.Fatalf("click did not cycle the difficulty value (%s)", before)
	}
	if !fx.settings.Dirty() {
		t.Fatalf("change did not mark the store dirty")
	}

	// Debounce: the flush fires DebounceMs after the last change.
	fx.clk.advance(mainmenu.DebounceMs)
	fx.settings.Update()

	fresh := game.NewSettingsCfg(fx.adapter.Path())
	if !fresh.HasFile() {
		t.Fatalf("settings.cfg not written at %s", fresh.Path())
	}
	if got := fresh.Get(key); got != after {
		t.Fatalf("reloaded %s = %q, want %q", key, got, after)
	}
}

// TestMissingSettingsCfgFallsBackToDefaults verifies AC-2's missing-file
// branch: defaults apply, no error surfaces.
func TestMissingSettingsCfgFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	adapter := game.NewSettingsCfg(filepath.Join(dir, "does-not-exist", "settings.cfg"))
	if err := adapter.Load(); err != nil {
		t.Fatalf("Load of missing file returned error: %v", err)
	}
	if adapter.HasFile() {
		t.Fatalf("HasFile = true for a missing file")
	}
	if got := adapter.Get(mainmenu.SettingDifficulty); got != "" {
		t.Fatalf("Get(missing key) = %q, want \"\"", got)
	}
}

// TestCorruptSettingsCfgFallsBackToDefaults verifies AC-2's corrupt-file
// branch: unparsable JSON yields defaults with no error dialog, and a
// partially invalid file (a non-scalar value under one key) leaves sibling
// categories unaffected.
func TestCorruptSettingsCfgFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.cfg")
	if err := os.WriteFile(path, []byte("this is not json {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := game.NewSettingsCfg(path)
	if err := adapter.Load(); err != nil {
		t.Fatalf("Load of corrupt file returned error: %v", err)
	}
	if adapter.HasFile() {
		t.Fatalf("HasFile = true for a corrupt file")
	}
	if got := adapter.Get(mainmenu.SettingDifficulty); got != "" {
		t.Fatalf("Get after corrupt file = %q, want \"\"", got)
	}

	// Partial invalidity: an object under one key reads as absent; numeric
	// and string siblings survive.
	partial := `{"gameplay": {"difficulty": {"nested": true}}, "camera": {"edge_scroll_speed": 20}, "subtitles": "on"}`
	path2 := filepath.Join(t.TempDir(), "settings.cfg")
	if err := os.WriteFile(path2, []byte(partial), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter2 := game.NewSettingsCfg(path2)
	if !adapter2.HasFile() {
		t.Fatalf("HasFile = false for a valid partial file")
	}
	if got := adapter2.Get(mainmenu.SettingDifficulty); got != "" {
		t.Fatalf("Get(object value) = %q, want \"\" (default)", got)
	}
	if got := adapter2.Get(mainmenu.SettingEdgeScrollSpeed); got != "20" {
		t.Fatalf("Get(numeric sibling) = %q, want \"20\"", got)
	}
	if got := adapter2.Get(mainmenu.SettingSubtitles); got != "on" {
		t.Fatalf("Get(string sibling) = %q, want \"on\"", got)
	}
}

// TestKeybindingRebindPersistsAcrossAdapterReload verifies AC-2/AC-3's
// persistence contract end to end: capture a rebind (W -> G) through the
// real screen over the real adapter, flush, then a fresh screen instance
// over the same file shows the rebound chord (the reboot path).
func TestKeybindingRebindPersistsAcrossAdapterReload(t *testing.T) {
	fx := newWiringFixture(t)
	wiringNavigateToKeybindings(t, fx)

	wiringStartCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "G"}) {
		t.Fatalf("capture key G not consumed")
	}
	if got := fx.keybind.Bindings(mainmenu.ActionBinding("move_forward")); len(got) == 0 || got[0].Key != "G" {
		t.Fatalf("primary binding = %+v, want G", got)
	}

	// Debounce flush writes controls.bindings.move_forward.primary = "G".
	fx.clk.advance(mainmenu.DebounceMs)
	fx.keybind.Update()

	fresh := game.NewSettingsCfg(fx.adapter.Path())
	if got := fresh.Get("controls.bindings.move_forward.primary"); got != "G" {
		t.Fatalf("persisted binding = %q, want \"G\"", got)
	}

	// Reboot path: a brand-new keybindings screen over the same file must
	// show the rebound chord after OnEnter (Load re-reads the store).
	clk2 := &fakeClock{}
	flow2 := mainmenu.NewScreenFlow(mainmenu.NewTransitionSystem(clk2.now))
	if err := flow2.RegisterController(mainmenu.ScreenTitleScreen, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatal(err)
	}
	if err := flow2.RegisterController(mainmenu.ScreenSettings, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatal(err)
	}
	kb2 := mainmenu.NewKeybindingsScreen(flow2,
		mainmenu.WithKeybindingsClock(clk2.now),
		mainmenu.WithKeybindingsReader(fresh.Get),
		mainmenu.WithKeybindingsWriter(fresh.SetAndFlush),
		mainmenu.WithKeybindingsHasSettingsFile(fresh.HasFile),
	)
	// The game layer runs Load at boot (the AC-8 boot read) — this is the
	// reboot path that re-reads the persisted store.
	kb2.Load()
	if err := flow2.RegisterController(mainmenu.ScreenKeybindings, kb2); err != nil {
		t.Fatal(err)
	}
	navigate(t, flow2, clk2, mainmenu.ScreenSettings)
	navigate(t, flow2, clk2, mainmenu.ScreenKeybindings)
	if got := kb2.Bindings(mainmenu.ActionBinding("move_forward")); len(got) == 0 || got[0].Key != "G" {
		t.Fatalf("after reload primary binding = %+v, want G", got)
	}
}

// ---------------------------------------------------------------------------
// AC-3 — HandleKey routing contract
// ---------------------------------------------------------------------------

// TestRouteRawKeyCaptureConsumesAndBinds verifies the host routing contract:
// while CapturesInput is true a raw key is routed to the screen first and
// consumed (true), completing the capture.
func TestRouteRawKeyCaptureConsumesAndBinds(t *testing.T) {
	fx := newWiringFixture(t)
	wiringNavigateToKeybindings(t, fx)
	wiringStartCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)

	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "G"}) {
		t.Fatalf("capture key not consumed")
	}
	if fx.keybind.CapturesInput() {
		t.Fatalf("capture still active after a base key")
	}
	if fx.flow.Current() != mainmenu.ScreenKeybindings {
		t.Fatalf("flow left the keybindings screen: %s", fx.flow.Current())
	}
}

// TestRouteRawKeyPanelStatePassesThroughToFlow verifies the false-return
// passthrough: with no capture active the key is dispatched to the flow —
// Escape backs from Keybindings to Settings, exactly as the game layer
// routes it.
func TestRouteRawKeyPanelStatePassesThroughToFlow(t *testing.T) {
	fx := newWiringFixture(t)
	wiringNavigateToKeybindings(t, fx)

	if fx.keybind.CapturesInput() {
		t.Fatalf("CapturesInput = true in the panel state")
	}
	if mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("panel-state key must not be consumed by capture")
	}
	// The dispatch started the back transition; the swap completes on Update.
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenSettings {
		t.Fatalf("Escape passthrough did not back to Settings (current = %s)", fx.flow.Current())
	}
}

// TestRouteRawKeyReservedWarnOnlyEscapeConsumed verifies the reserved-key
// dialog's routing rule: in ReservedWarn only Escape is consumed (it cancels
// the capture); every other key returns false so the dialog's buttons stay
// keyboard-reachable via the flow.
func TestRouteRawKeyReservedWarnOnlyEscapeConsumed(t *testing.T) {
	fx := newWiringFixture(t)
	wiringNavigateToKeybindings(t, fx)

	// Capturing Escape opens the AC-5 reserved-key warning dialog.
	wiringStartCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("Escape not consumed into the reserved dialog")
	}
	if fx.keybind.State() != mainmenu.KeybindingsStateReservedWarn {
		t.Fatalf("state = %s, want ReservedWarn", fx.keybind.State())
	}
	// The dialog's Escape again cancels the capture (AC-1 cancel path).
	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("dialog Escape not consumed")
	}
	if fx.keybind.CapturesInput() {
		t.Fatalf("capture still active after dialog cancel")
	}

	// A non-Escape key in the dialog passes through to the flow.
	wiringStartCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)
	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("Escape not consumed into the reserved dialog")
	}
	if mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "G"}) {
		t.Fatalf("non-Escape dialog key must pass through to the flow")
	}
	if fx.flow.Current() != mainmenu.ScreenKeybindings {
		t.Fatalf("passthrough moved the flow: %s", fx.flow.Current())
	}
}

// ---------------------------------------------------------------------------
// AC-4 — no key leakage during capture
// ---------------------------------------------------------------------------

// TestCaptureConsumesNavKeyWithoutFocusLeak verifies that a key which would
// otherwise navigate the menu (ArrowDown maps to ActionMoveDown -> focus
// moves) is fully consumed by capture: the key ends up in the binding, never
// in the flow. If it leaked, the focus manager would MoveGroup and the
// binding would remain unbound.
func TestCaptureConsumesNavKeyWithoutFocusLeak(t *testing.T) {
	fx := newWiringFixture(t)
	wiringNavigateToKeybindings(t, fx)
	wiringStartCapture(t, fx, "kb-row-move-forward-primary", mainmenu.ActionBinding("move_forward"), mainmenu.SlotPrimary)

	if !mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "ArrowDown"}) {
		t.Fatalf("ArrowDown leaked past capture")
	}
	// The nav key was captured as the binding — proof it never dispatched.
	if got := fx.keybind.Bindings(mainmenu.ActionBinding("move_forward")); len(got) == 0 || got[0].Key != "ArrowDown" {
		t.Fatalf("primary binding = %+v, want ArrowDown", got)
	}
	if fx.flow.Current() != mainmenu.ScreenKeybindings {
		t.Fatalf("flow left the keybindings screen: %s", fx.flow.Current())
	}
	if fx.keybind.CapturesInput() {
		t.Fatalf("capture still active after the captured key")
	}
}

// ---------------------------------------------------------------------------
// AC-5 — menu <-> game transitions
// ---------------------------------------------------------------------------

// TestSettingsToKeybindingsNavigationWithRealAdapter verifies the Settings ->
// Keybindings chain with the real adapter injected, and that Back/Escape
// returns through the same chain to the title screen.
func TestSettingsToKeybindingsNavigationWithRealAdapter(t *testing.T) {
	fx := newWiringFixture(t)

	navigate(t, fx.flow, fx.clk, mainmenu.ScreenSettings)
	if fx.settings.EntryPoint() != mainmenu.ScreenTitleScreen {
		t.Fatalf("settings entry point = %s, want TitleScreen", fx.settings.EntryPoint())
	}

	// Change a setting before leaving (real adapter path), then navigate on.
	click(t, fx.flow, "setting-gameplay-difficulty")
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenKeybindings)

	// Back (Escape passthrough) returns to Settings, then to Title.
	if mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("panel-state Escape consumed by capture")
	}
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenSettings {
		t.Fatalf("back from keybindings = %s, want Settings", fx.flow.Current())
	}
	if mainmenu.RouteRawKey(fx.keybind, fx.flow, mainmenu.KeyEvent{Key: "Escape"}) {
		t.Fatalf("settings Escape consumed by keybindings capture")
	}
	// The settings screen's own Escape routes through the flow's Back.
	fx.flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"}))
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("back from settings = %s, want TitleScreen", fx.flow.Current())
	}

	// The changed setting survived the trip (still dirty, file flushed later).
	if !fx.settings.Dirty() {
		t.Fatalf("setting change lost during navigation")
	}
}

// TestInGamePauseSettingsKeybindingsBackChain verifies AC-5's in-game path:
// Escape opens the pause menu from InGame, Settings opens from the pause
// menu, Controls -> Keybindings opens from Settings, and Back/Escape walks
// the whole chain back to InGame. The world stays live behind the parchment
// (the flow never leaves InGame's controller — the pause freeze contract is
// Story 006's out-of-scope follow-up).
func TestInGamePauseSettingsKeybindingsBackChain(t *testing.T) {
	fx := newWiringFixture(t)

	// InGame -> Escape -> PauseMenu.
	navigate(t, fx.flow, fx.clk, mainmenu.ScreenInGame)
	fx.flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"}))
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("escape in game = %s, want PauseMenu", fx.flow.Current())
	}

	// PauseMenu -> Settings (the pause menu's Settings button).
	click(t, fx.flow, mainmenu.PauseFocusSettings)
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenSettings {
		t.Fatalf("pause settings = %s, want Settings", fx.flow.Current())
	}
	if fx.settings.EntryPoint() != mainmenu.ScreenPauseMenu {
		t.Fatalf("settings entry point = %s, want PauseMenu", fx.settings.EntryPoint())
	}

	// Settings -> Controls tab -> Keybindings row.
	click(t, fx.flow, "tab-controls")
	click(t, fx.flow, "setting-controls-keybindings")
	fx.clk.advance(mainmenu.ScreenCrossfadeMs)
	fx.flow.Update()
	if fx.flow.Current() != mainmenu.ScreenKeybindings {
		t.Fatalf("controls row = %s, want Keybindings", fx.flow.Current())
	}

	// Escape walks back: Keybindings -> Settings -> PauseMenu -> InGame.
	for _, want := range []mainmenu.Screen{mainmenu.ScreenSettings, mainmenu.ScreenPauseMenu, mainmenu.ScreenInGame} {
		fx.flow.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"}))
		fx.clk.advance(mainmenu.ScreenCrossfadeMs)
		fx.flow.Update()
		if fx.flow.Current() != want {
			t.Fatalf("back chain = %s, want %s", fx.flow.Current(), want)
		}
	}
}

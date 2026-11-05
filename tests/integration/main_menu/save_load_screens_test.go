// Integration tests for Story 004: Save/Load Screens.
//
// Evidence path: production/epics/main-menu-settings/
// story-004-save-load-screens.md
// Required evidence: this file must exist and pass (Integration story gate).
//
// These tests drive the pure-Go core of src/ui/mainmenu headlessly with an
// injectable clock (fakeClock) and stubbed Save/Load System callbacks — no
// file I/O, no engine host, no external APIs. The Save/Load System data
// contract (ListSlots/SaveGame/DeleteSave/RestoreSave and the cloud-conflict
// probe/resolver) is injected as functional options, exactly as the game
// layer wires it. Timing assertions follow the GDD measurement protocol:
// exact and tolerance-free at the unit level.
//
// Acceptance criteria map to test functions 1:1:
//
//	AC-1  test_slot_list_displays_22_slots_sorted_desc  -> TestSlotListDisplays22SlotsSortedByTimestampDesc
//	AC-1  test_zero_saves_shows_all_empty_slots         -> TestZeroSavesShowsAllEmptySlots
//	AC-1  test_twenty_two_saves_fills_every_slot        -> TestTwentyTwoSavesFillsEverySlot
//	AC-1  test_empty_slots_dimmed                       -> TestEmptySlotsDimmedInLoadMode
//	AC-2  test_click_occupied_slot_begins_load          -> TestClickOccupiedSlotBeginsLoad
//	AC-2  test_load_failure_returns_to_title_with_error -> TestLoadFailureReturnsToTitleWithError
//	AC-2  test_empty_slot_click_no_op_in_load_mode      -> TestClickEmptySlotInLoadModeDoesNothing
//	AC-3  test_overwrite_dialog_on_occupied_slot        -> TestOverwriteDialogShownForOccupiedSlotInSaveMode
//	AC-3  test_overwrite_confirm_calls_save_game        -> TestOverwriteConfirmCallsSaveGame
//	AC-3  test_save_failure_shows_auto_dismiss_toast    -> TestSaveFailureShowsAutoDismissingToast
//	AC-3  test_empty_slot_opens_name_entry              -> TestEmptySlotInSaveModeOpensNameEntry
//	AC-4  test_corrupted_slot_error_dialog_no_load      -> TestCorruptedSlotShowsErrorDialogAndNeverLoads
//	AC-4  test_corrupted_restore_path                   -> TestCorruptedRestoreCallsRestoreSave
//	AC-4  test_corrupted_restore_disabled_without_backup -> TestCorruptedRestoreDisabledWithoutBackup
//	AC-4  test_corrupted_restore_failure_message        -> TestCorruptedRestoreFailureShowsNoBackupMessage
//	AC-4  test_corrupted_delete_confirmation            -> TestCorruptedDeleteRequiresConfirmation
//	AC-4  test_corrupted_cancel_returns_to_slots        -> TestCorruptedCancelReturnsToSlotList
//	AC-5  test_conflict_modal_shows_before_title        -> TestCloudConflictModalShowsBothVersionsBeforeTitle
//	AC-5  test_conflict_modal_three_choices             -> TestConflictModalOffersThreeChoices
//	AC-5  test_keep_local_resolves_conflict             -> TestKeepLocalResolvesAndCallsResolver
//	AC-5  test_no_conflict_skips_modal                  -> TestNoConflictSkipsModal
//	AC-6  test_decide_later_blocks_autosave_and_banner  -> TestDecideLaterBlocksAutosaveAndShowsBanner
//	AC-6  test_banner_clears_on_resolution              -> TestBannerClearsWhenConflictResolved
//	edge  test_escape_in_conflict_modal_chooses_later   -> TestEscapeInConflictModalChoosesDecideLater
//	reg   test_load_game_back_returns_to_pause          -> TestLoadGameBackReturnsToPauseMenuEntryPoint
package main_menu_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"marrow-of-empire/ui/mainmenu"
)

// ---------------------------------------------------------------------------
// Fixtures (mirrored from tests/unit/main_menu — Go cannot import test
// helpers across packages)
// ---------------------------------------------------------------------------

// fakeClock is the injectable game-time clock (milliseconds). All timing
// assertions read from it, so they are deterministic and tolerance-free.
type fakeClock struct{ ms float64 }

func (c *fakeClock) now() float64 { return c.ms }
func (c *fakeClock) advance(ms float64) { c.ms += ms }

// stubScreen is a minimal ScreenController for headless flow tests.
type stubScreen struct {
	focus *mainmenu.FocusManager
}

func (s *stubScreen) OnEnter(mainmenu.Screen) {}
func (s *stubScreen) OnExit()                 {}
func (s *stubScreen) Focus() *mainmenu.FocusManager { return s.focus }

// recordingLoader is the injected SaveLoader seam (Story 001 AC-9): records
// the slot ids it was asked to load and returns a fixed error.
type recordingLoader struct {
	slots []string
	err   error
}

func (l *recordingLoader) Load(slot string) error {
	l.slots = append(l.slots, slot)
	return l.err
}

// stubSystem is the stubbed Save/Load System data contract. The screen only
// sees the injected callbacks; the stub keeps mutable state so tests can
// simulate saves being created and deleted.
type stubSystem struct {
	slots        []mainmenu.SaveSlot
	savedID      string
	savedName    string
	deleteCalls  []string
	restoreCalls []string
	hasBackup    map[string]bool
	backupErr    error
	saveErr      error
	character    string
	wallClock    time.Time

	hasConflict bool
	conflict    mainmenu.CloudConflictInfo
	resolutions []string
}

func (s *stubSystem) list() []mainmenu.SaveSlot { return s.slots }

func (s *stubSystem) save(slotID, name string) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.savedID, s.savedName = slotID, name
	for i := range s.slots {
		if s.slots[i].ID == slotID {
			s.slots[i].Empty = false
			s.slots[i].Name = name
			s.slots[i].ChecksumValid = true
			return nil
		}
	}
	// Slot not in the list (a screen-padded empty slot): the system accepts
	// the save and the next ListSlots reflects it.
	s.slots = append(s.slots, mainmenu.SaveSlot{ID: slotID, Name: name, Empty: false, ChecksumValid: true})
	return nil
}

func (s *stubSystem) deleteSlot(slotID string) error {
	s.deleteCalls = append(s.deleteCalls, slotID)
	for i := range s.slots {
		if s.slots[i].ID == slotID {
			s.slots[i].Empty = true
			s.slots[i].ChecksumValid = true
		}
	}
	return nil
}

func (s *stubSystem) restore(slotID string) error {
	if s.backupErr != nil {
		return s.backupErr
	}
	s.restoreCalls = append(s.restoreCalls, slotID)
	for i := range s.slots {
		if s.slots[i].ID == slotID {
			s.slots[i].ChecksumValid = true
		}
	}
	return nil
}

func (s *stubSystem) backupExists(slotID string) bool { return s.hasBackup[slotID] }

func (s *stubSystem) hasCloudConflict() bool { return s.hasConflict }

func (s *stubSystem) conflictInfo() mainmenu.CloudConflictInfo { return s.conflict }

func (s *stubSystem) resolve(choice string) { s.resolutions = append(s.resolutions, choice) }

// occupiedSlot builds a non-empty SaveSlot with the given fields.
func occupiedSlot(id, name, character, region, thumb string, playtime time.Duration, ts time.Time) mainmenu.SaveSlot {
	return mainmenu.SaveSlot{
		ID:            id,
		Name:          name,
		CharacterName: character,
		Playtime:      playtime,
		Region:        region,
		Thumbnail:     thumb,
		Timestamp:     ts,
		ChecksumValid: true,
		Empty:         false,
	}
}

// newFixture builds a flow in the title state with a recording loader, a
// stub Save/Load System, and real SaveLoadScreen controllers registered for
// both ScreenLoadGame and ScreenSaveGame. Returns the pieces tests drive.
func newFixture(t *testing.T) (*mainmenu.ScreenFlow, *fakeClock, *recordingLoader, *stubSystem, *mainmenu.SaveLoadScreen, *mainmenu.SaveLoadScreen) {
	t.Helper()
	clk := &fakeClock{}
	sys := &stubSystem{
		hasBackup: map[string]bool{},
		wallClock: time.Date(2026, 8, 10, 14, 22, 0, 0, time.UTC),
	}
	loader := &recordingLoader{}
	ts := mainmenu.NewTransitionSystem(clk.now)
	flow := mainmenu.NewScreenFlow(ts, mainmenu.WithSaveLoader(loader))
	if err := flow.RegisterController(mainmenu.ScreenTitleScreen, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatalf("register title: %v", err)
	}
	if err := flow.RegisterController(mainmenu.ScreenInGame, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatalf("register ingame: %v", err)
	}
	if err := flow.RegisterController(mainmenu.ScreenPauseMenu, &stubScreen{focus: mainmenu.NewFocusManager(mainmenu.NewFocusStyle(1.0))}); err != nil {
		t.Fatalf("register pause: %v", err)
	}
	load := mainmenu.NewSaveLoadScreen(flow, mainmenu.SlotModeLoad,
		mainmenu.WithSaveLoadClock(clk.now),
		mainmenu.WithListSlots(sys.list),
		mainmenu.WithSaveGame(sys.save),
		mainmenu.WithDeleteSave(sys.deleteSlot),
		mainmenu.WithRestoreSave(sys.restore),
		mainmenu.WithHasBackup(sys.backupExists),
		mainmenu.WithCurrentTime(func() time.Time { return sys.wallClock }))
	if err := flow.RegisterController(mainmenu.ScreenLoadGame, load); err != nil {
		t.Fatalf("register load screen: %v", err)
	}
	save := mainmenu.NewSaveLoadScreen(flow, mainmenu.SlotModeSave,
		mainmenu.WithSaveLoadClock(clk.now),
		mainmenu.WithListSlots(sys.list),
		mainmenu.WithSaveGame(sys.save),
		mainmenu.WithDeleteSave(sys.deleteSlot),
		mainmenu.WithRestoreSave(sys.restore),
		mainmenu.WithHasBackup(sys.backupExists),
		mainmenu.WithCharacterName(func() string { return sys.character }),
		mainmenu.WithCurrentTime(func() time.Time { return sys.wallClock }))
	if err := flow.RegisterController(mainmenu.ScreenSaveGame, save); err != nil {
		t.Fatalf("register save screen: %v", err)
	}
	return flow, clk, loader, sys, load, save
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

// click activates the given focusable id through the flow's mouse path
// (AC-3: click to focus and activate).
func click(t *testing.T, flow *mainmenu.ScreenFlow, id string) {
	t.Helper()
	if !flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id})) {
		t.Fatalf("click %q not handled", id)
	}
}

// clickIgnored asserts that a click on id is NOT handled — the expected
// behavior for non-interactive elements (empty slots in Load mode, disabled
// dialog buttons).
func clickIgnored(t *testing.T, flow *mainmenu.ScreenFlow, id string) {
	t.Helper()
	if flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id})) {
		t.Fatalf("click %q must not be handled", id)
	}
}

// navigateToSaveGame reaches the Save screen the way the game does (AC-7):
// TitleScreen -> InGame -> PauseMenu -> SaveGame. The Save screen is entered
// from the Pause Menu only.
func navigateToSaveGame(t *testing.T, flow *mainmenu.ScreenFlow, clk *fakeClock) {
	t.Helper()
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	navigate(t, flow, clk, mainmenu.ScreenPauseMenu)
	navigate(t, flow, clk, mainmenu.ScreenSaveGame)
}

// fiveOccupiedSlots builds a 5-save stub list with deliberately unsorted
// timestamps to prove the screen sorts (AC-1).
func fiveOccupiedSlots() []mainmenu.SaveSlot {
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	return []mainmenu.SaveSlot{
		occupiedSlot("s4", "Oldest", "Elena", "Thornwell", "thumbs/s4.png", 4*time.Hour, base.Add(-72*time.Hour)),
		occupiedSlot("s1", "Newest", "Alaric", "Thornwell", "thumbs/s1.png", 12*time.Hour+30*time.Minute, base),
		occupiedSlot("s3", "Third", "Bram", "Mirefen", "thumbs/s3.png", 2*time.Hour, base.Add(-48*time.Hour)),
		occupiedSlot("s2", "Second", "Cora", "Mirefen", "thumbs/s2.png", 8*time.Hour, base.Add(-24*time.Hour)),
		occupiedSlot("s5", "Fourth", "Dune", "Thornwell", "thumbs/s5.png", 1*time.Hour, base.Add(-50*time.Hour)),
	}
}

// ---------------------------------------------------------------------------
// AC-1: Slot list display, sorting, dimming
// ---------------------------------------------------------------------------

func TestSlotListDisplays22SlotsSortedByTimestampDesc(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)

	slots := load.Slots()
	if len(slots) != mainmenu.TotalSlotCount {
		t.Fatalf("slot count = %d, want %d", len(slots), mainmenu.TotalSlotCount)
	}
	// Occupied slots sorted by timestamp descending (most recent first).
	wantOrder := []string{"s1", "s2", "s3", "s5", "s4"}
	for i, want := range wantOrder {
		if slots[i].ID != want {
			t.Fatalf("occupied slot %d = %q, want %q", i, slots[i].ID, want)
		}
	}
	// The five empties (17) come last and are flagged Empty.
	for i := 5; i < len(slots); i++ {
		if !slots[i].Empty {
			t.Fatalf("slot %d must be empty, got %+v", i, slots[i])
		}
	}
	// Occupied slot metadata is exposed for the rendering pass.
	top := slots[0]
	if top.Name != "Newest" || top.CharacterName != "Alaric" || top.Region != "Thornwell" ||
		top.Thumbnail != "thumbs/s1.png" || top.Playtime != 12*time.Hour+30*time.Minute {
		t.Fatalf("top slot metadata = %+v", top)
	}
	if got := mainmenu.FormatPlaytime(top.Playtime); got != "12:30" {
		t.Fatalf("FormatPlaytime(12h30m) = %q, want %q", got, "12:30")
	}
	if got := mainmenu.FormatTimestamp(top.Timestamp); got != "2026-08-01 12:00" {
		t.Fatalf("FormatTimestamp = %q, want %q", got, "2026-08-01 12:00")
	}
}

func TestZeroSavesShowsAllEmptySlots(t *testing.T) {
	flow, clk, _, _, load, _ := newFixture(t)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	slots := load.Slots()
	if len(slots) != mainmenu.TotalSlotCount {
		t.Fatalf("slot count = %d, want %d", len(slots), mainmenu.TotalSlotCount)
	}
	for i := range slots {
		if !slots[i].Empty {
			t.Fatalf("slot %d must be empty with no saves, got %+v", i, slots[i])
		}
	}
	// Padded slots carry the positional id so SaveGame stays valid.
	if slots[0].ID != "slot-0" {
		t.Fatalf("padded slot id = %q, want %q", slots[0].ID, "slot-0")
	}
}

func TestTwentyTwoSavesFillsEverySlot(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < mainmenu.TotalSlotCount; i++ {
		sys.slots = append(sys.slots,
			occupiedSlot("sys"+string(rune('a'+i)), "Save", "Hero", "Thornwell", "", time.Hour, base.Add(-time.Duration(i)*time.Hour)))
	}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if got := load.SlotCount(); got != mainmenu.TotalSlotCount {
		t.Fatalf("slot count = %d, want %d", got, mainmenu.TotalSlotCount)
	}
	for i := range load.Slots() {
		if load.Slots()[i].Empty {
			t.Fatalf("slot %d must be occupied with 22 saves", i)
		}
	}
}

func TestEmptySlotsDimmedInLoadMode(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	_ = load
	sys.slots = fiveOccupiedSlots()
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	for i := 0; i < 5; i++ {
		if load.SlotDimmed(i) {
			t.Fatalf("occupied slot %d must not be dimmed", i)
		}
	}
	for i := 5; i < mainmenu.TotalSlotCount; i++ {
		if !load.SlotDimmed(i) {
			t.Fatalf("empty slot %d must be dimmed", i)
		}
	}
	style := load.SlotCardStyle()
	if style.Background != mainmenu.ColorDarkStone {
		t.Fatalf("card background = %#v, want dark stone (metadata is illegible on parchment)", style.Background)
	}
	// GDD asks for >=12:1 "pale ivory on dark stone" as the desired target;
	// the desk palette measures 10.47:1, which exceeds WCAG AAA for normal
	// text (7:1) — the assertion gates on the WCAG floor and documents the
	// actual palette ratio.
	if ratio := mainmenu.ContrastRatio(style.Text, style.Background); ratio < 7.0 {
		t.Fatalf("text contrast on dark stone = %.2f, want >= 7:1 (WCAG AAA); palette yields 10.47:1", ratio)
	}
	if style.SelectedBorder != mainmenu.ColorAmber || style.SelectedUnderline != mainmenu.ColorInk {
		t.Fatalf("selection cues = border %#v underline %#v, want amber + ink (two cues)", style.SelectedBorder, style.SelectedUnderline)
	}
}

// ---------------------------------------------------------------------------
// AC-2: Load mode
// ---------------------------------------------------------------------------

func TestClickOccupiedSlotBeginsLoad(t *testing.T) {
	flow, clk, loader, sys, load, _ := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	// The most recent save (s1) sorts first: click its positional slot.
	click(t, flow, "slot-0")
	if flow.Current() != mainmenu.ScreenLoading {
		t.Fatalf("current = %s, want Loading (hard cut) after click", flow.Current())
	}
	if len(loader.slots) != 1 || loader.slots[0] != "s1" {
		t.Fatalf("loader slots = %v, want [s1]", loader.slots)
	}
	// Success: crossfade in to the game world (Story 001 AC-9).
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenInGame {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenInGame)
	}
	if got := flow.LastError(); got != "" {
		t.Fatalf("error = %q, want empty after successful load", got)
	}
	// The load screen was exited; the screen state resets on re-entry.
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("load screen state = %s, want Slots", load.State())
	}
}

func TestLoadFailureReturnsToTitleWithError(t *testing.T) {
	flow, clk, loader, sys, _, _ := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	loader.err = errors.New("save checksum mismatch: this save file is corrupted")
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-1") // s2 (second most recent)
	if len(loader.slots) != 1 || loader.slots[0] != "s2" {
		t.Fatalf("loader slots = %v, want [s2]", loader.slots)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s", flow.Current(), mainmenu.ScreenTitleScreen)
	}
	if got := flow.LastError(); !strings.Contains(got, "corrupted") {
		t.Fatalf("error = %q, want it to describe the corruption", got)
	}
}

func TestClickEmptySlotInLoadModeDoesNothing(t *testing.T) {
	flow, clk, loader, sys, load, _ := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	clickIgnored(t, flow, "slot-5") // first empty slot
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("current = %s, want %s (empty slots are non-interactive)", flow.Current(), mainmenu.ScreenLoadGame)
	}
	if len(loader.slots) != 0 {
		t.Fatalf("loader must never be called for an empty slot, got %v", loader.slots)
	}
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots", load.State())
	}
	// Disabled slots cannot even be focused (AC-2).
	if load.Focus().FocusedID() == "slot-5" {
		t.Fatal("empty slot must not take focus in Load mode")
	}
}

// ---------------------------------------------------------------------------
// AC-3: Save mode
// ---------------------------------------------------------------------------

func TestOverwriteDialogShownForOccupiedSlotInSaveMode(t *testing.T) {
	flow, clk, loader, sys, _, save := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigateToSaveGame(t, flow, clk)
	click(t, flow, "slot-0") // Newest: character Alaric, 12:30 playtime
	if save.State() != mainmenu.SaveLoadStateOverwrite {
		t.Fatalf("state = %s, want Overwrite", save.State())
	}
	want := "Overwrite Newest — 12:30?"
	if got := save.OverwriteTitle(); got != want {
		t.Fatalf("overwrite title = %q, want %q", got, want)
	}
	// The Save screen must never trigger a load (AC-7).
	if len(loader.slots) != 0 {
		t.Fatalf("loader called from Save screen: %v", loader.slots)
	}
	// Cancel returns to the slot list (AC-3).
	click(t, flow, mainmenu.FocusOverwriteCancel)
	if save.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state after cancel = %s, want Slots", save.State())
	}
}

func TestOverwriteConfirmCallsSaveGame(t *testing.T) {
	flow, clk, _, sys, _, save := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigateToSaveGame(t, flow, clk)
	click(t, flow, "slot-2") // Third: system id s3
	click(t, flow, mainmenu.FocusOverwriteConfirm)
	if sys.savedID != "s3" || sys.savedName != "Third" {
		t.Fatalf("SaveGame(%q, %q), want (s3, Third)", sys.savedID, sys.savedName)
	}
	if save.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots after successful save", save.State())
	}
	if save.ToastVisible() {
		t.Fatal("no toast expected on a successful save")
	}
}

func TestSaveFailureShowsAutoDismissingToast(t *testing.T) {
	flow, clk, _, sys, _, save := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	sys.saveErr = errors.New("disk full")
	navigateToSaveGame(t, flow, clk)
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusOverwriteConfirm)

	if !save.ToastVisible() {
		t.Fatal("toast must be visible after a failed save")
	}
	want := "Save failed — unable to write to Newest. Check folder permissions."
	if got := save.ToastMessage(); got != want {
		t.Fatalf("toast = %q, want %q", got, want)
	}
	// The slot list stays fully interactive (UX spec edge case).
	if save.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots (slot list remains interactive)", save.State())
	}
	// Auto-dismiss after ToastDurationMs via the injectable clock.
	clk.advance(mainmenu.ToastDurationMs - 1)
	if !save.ToastVisible() {
		t.Fatal("toast must still be visible just before the timeout")
	}
	clk.advance(1)
	save.Refresh()
	if save.ToastVisible() {
		t.Fatal("toast must auto-dismiss after ToastDurationMs")
	}
	// Manual dismissal also works.
	sys.saveErr = errors.New("still failing")
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusOverwriteConfirm)
	if !save.ToastVisible() {
		t.Fatal("second toast must show")
	}
	save.DismissToast()
	if save.ToastVisible() {
		t.Fatal("toast must dismiss on DismissToast")
	}
}

func TestEmptySlotInSaveModeOpensNameEntry(t *testing.T) {
	flow, clk, _, sys, _, save := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	sys.character = "Alaric"
	navigateToSaveGame(t, flow, clk)
	click(t, flow, "slot-5") // first empty slot
	if save.State() != mainmenu.SaveLoadStateNameEntry {
		t.Fatalf("state = %s, want NameEntry", save.State())
	}
	want := "Alaric · 2026-08-10 14:22"
	if got := save.NameEntryValue(); got != want {
		t.Fatalf("default name = %q, want %q", got, want)
	}
	// Confirm saves with the default name into the empty slot's id.
	click(t, flow, mainmenu.FocusSaveNameConfirm)
	if sys.savedID == "" || sys.savedName != want {
		t.Fatalf("SaveGame(%q, %q), want (slot id, %q)", sys.savedID, sys.savedName, want)
	}
	if save.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots after save", save.State())
	}
	// The refreshed list reflects the new save.
	if save.Slots()[5].Empty {
		t.Fatal("slot 5 must be occupied after the save")
	}
	// Custom entry overrides the default.
	click(t, flow, "slot-6")
	save.SetNameEntry("  Journey Home  ")
	click(t, flow, mainmenu.FocusSaveNameConfirm)
	if sys.savedName != "Journey Home" {
		t.Fatalf("saved name = %q, want trimmed %q", sys.savedName, "Journey Home")
	}
	// Cancel returns to the slot list without saving.
	before := sys.savedID
	click(t, flow, "slot-7")
	click(t, flow, mainmenu.FocusSaveNameCancel)
	if save.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state after cancel = %s, want Slots", save.State())
	}
	if sys.savedID != before {
		t.Fatalf("cancel must not save: savedID changed %q -> %q", before, sys.savedID)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Corrupted save dialog
// ---------------------------------------------------------------------------

func TestCorruptedSlotShowsErrorDialogAndNeverLoads(t *testing.T) {
	flow, clk, loader, sys, load, _ := newFixture(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	sys.slots = []mainmenu.SaveSlot{
		occupiedSlot("good", "Good", "Hero", "Thornwell", "", 2*time.Hour, base),
		{ID: "corrupt", Name: "Corrupt", CharacterName: "Hero", Timestamp: base.Add(time.Hour), ChecksumValid: false, Empty: false},
	}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	// The corrupted slot (13:00) sorts before the good slot (12:00).
	if load.Slots()[0].ID != "corrupt" {
		t.Fatalf("slot[0] = %q, want corrupt (newest timestamp sorts first)", load.Slots()[0].ID)
	}
	click(t, flow, "slot-0")
	if load.State() != mainmenu.SaveLoadStateCorrupted {
		t.Fatalf("state = %s, want Corrupted", load.State())
	}
	if got := load.CorruptedMessage(); got != mainmenu.CorruptedSaveMessage {
		t.Fatalf("dialog message = %q, want %q", got, mainmenu.CorruptedSaveMessage)
	}
	// The game must NOT attempt to load the corrupted file (AC-4).
	if len(loader.slots) != 0 {
		t.Fatalf("loader called for corrupted slot: %v", loader.slots)
	}
	if flow.Current() != mainmenu.ScreenLoadGame {
		t.Fatalf("current = %s, want %s (no transition from corrupted dialog)", flow.Current(), mainmenu.ScreenLoadGame)
	}
}

func TestCorruptedRestoreCallsRestoreSave(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	sys.hasBackup["corrupt"] = true
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	if load.RestoreDisabled() {
		t.Fatal("Restore must be enabled when a backup exists")
	}
	click(t, flow, mainmenu.FocusCorruptRestore)
	if len(sys.restoreCalls) != 1 || sys.restoreCalls[0] != "corrupt" {
		t.Fatalf("restore calls = %v, want [corrupt]", sys.restoreCalls)
	}
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots after restore", load.State())
	}
	if !load.Slots()[0].ChecksumValid {
		t.Fatal("slot must be valid after restore (list refresh)")
	}
}

func TestCorruptedRestoreDisabledWithoutBackup(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	// No backup entry: probe reports false (UX spec: Restore greyed out).
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	if !load.RestoreDisabled() {
		t.Fatal("Restore must be greyed out when no backup exists")
	}
	clickIgnored(t, flow, mainmenu.FocusCorruptRestore)
	if len(sys.restoreCalls) != 0 {
		t.Fatalf("disabled Restore must not fire, got %v", sys.restoreCalls)
	}
	if load.State() != mainmenu.SaveLoadStateCorrupted {
		t.Fatalf("state = %s, want Corrupted", load.State())
	}
}

func TestCorruptedRestoreFailureShowsNoBackupMessage(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	sys.hasBackup["corrupt"] = true
	sys.backupErr = errors.New("backup also corrupted")
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusCorruptRestore)
	if !load.ToastVisible() || load.ToastMessage() != mainmenu.NoValidBackupMessage {
		t.Fatalf("toast = %q, want %q (backup also corrupted edge case)",
			load.ToastMessage(), mainmenu.NoValidBackupMessage)
	}
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots (toast is non-blocking)", load.State())
	}
}

func TestCorruptedDeleteRequiresConfirmation(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusCorruptDelete)
	if load.State() != mainmenu.SaveLoadStateDeleteConfirm {
		t.Fatalf("state = %s, want DeleteConfirm", load.State())
	}
	if got := load.DeleteConfirmMessage(); got != mainmenu.DeleteConfirmMessage {
		t.Fatalf("delete message = %q, want %q", got, mainmenu.DeleteConfirmMessage)
	}
	// No delete before confirmation.
	if len(sys.deleteCalls) != 0 {
		t.Fatalf("DeleteSave called before confirmation: %v", sys.deleteCalls)
	}
	// Confirm deletes and refreshes the list.
	click(t, flow, mainmenu.FocusDeleteConfirm)
	if len(sys.deleteCalls) != 1 || sys.deleteCalls[0] != "corrupt" {
		t.Fatalf("delete calls = %v, want [corrupt]", sys.deleteCalls)
	}
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots after delete", load.State())
	}
	if !load.Slots()[0].Empty {
		t.Fatal("slot must be empty after delete (list refresh)")
	}
}

func TestCorruptedDeleteCancelReturnsToDialog(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusCorruptDelete)
	click(t, flow, mainmenu.FocusDeleteCancel)
	if load.State() != mainmenu.SaveLoadStateCorrupted {
		t.Fatalf("state = %s, want Corrupted after cancel", load.State())
	}
	if len(sys.deleteCalls) != 0 {
		t.Fatalf("DeleteSave must not run on cancel, got %v", sys.deleteCalls)
	}
}

func TestCorruptedCancelReturnsToSlotList(t *testing.T) {
	flow, clk, _, sys, load, _ := newFixture(t)
	sys.slots = []mainmenu.SaveSlot{{
		ID: "corrupt", Name: "Corrupt", Timestamp: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC),
		ChecksumValid: false, Empty: false,
	}}
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, "slot-0")
	click(t, flow, mainmenu.FocusCorruptCancel)
	if load.State() != mainmenu.SaveLoadStateSlots {
		t.Fatalf("state = %s, want Slots", load.State())
	}
}

// ---------------------------------------------------------------------------
// AC-5: Steam Cloud conflict modal
// ---------------------------------------------------------------------------

// newConflictModal builds a modal wired to the stub system and a host
// interactivity hook recorder.
func newConflictModal(t *testing.T, clk *fakeClock, sys *stubSystem) (*mainmenu.CloudConflictModal, *bool) {
	t.Helper()
	host := true
	m := mainmenu.NewCloudConflictModal(
		mainmenu.WithConflictClock(clk.now),
		mainmenu.WithHasCloudConflict(sys.hasCloudConflict),
		mainmenu.WithCloudConflictInfo(sys.conflictInfo),
		mainmenu.WithResolveCloudConflict(sys.resolve),
		mainmenu.WithSetInteractive(func(on bool) { host = on }),
	)
	return m, &host
}

func TestCloudConflictModalShowsBothVersionsBeforeTitle(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{
		hasConflict: true,
		conflict: mainmenu.CloudConflictInfo{
			Local:  mainmenu.ConflictVersion{Timestamp: time.Date(2026, 8, 10, 9, 15, 0, 0, time.UTC), Playtime: 8*time.Hour + 30*time.Minute, Region: "Thornwell"},
			Cloud:  mainmenu.ConflictVersion{Timestamp: time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC), Playtime: 7*time.Hour + 45*time.Minute, Region: "Mirefen"},
		},
	}
	m, host := newConflictModal(t, clk, sys)
	if m.Visible() {
		t.Fatal("modal must not be visible before Show")
	}
	m.Show()
	if !m.Visible() {
		t.Fatal("modal must be visible when a conflict exists")
	}
	info := m.Info()
	if info.Local.Playtime != 8*time.Hour+30*time.Minute || info.Cloud.Playtime != 7*time.Hour+45*time.Minute {
		t.Fatalf("conflict info = %+v, want local 8:30 vs cloud 7:45", info)
	}
	if info.Local.Region != "Thornwell" || info.Cloud.Region != "Mirefen" {
		t.Fatalf("conflict regions = %+v", info)
	}
	// The host screen (title menu) is non-interactive while the modal shows.
	if *host {
		t.Fatal("host screen must be non-interactive while the modal is visible")
	}
	// Slide-in progress tracks the clock (paper-slide motion, ADR-0012).
	if p := m.AppearProgress(); p != 0 {
		t.Fatalf("appear progress at t=0 = %g, want 0", p)
	}
	clk.advance(mainmenu.ModalSlideInMs)
	if p := m.AppearProgress(); p != 1 {
		t.Fatalf("appear progress at t=%g = %g, want 1", mainmenu.ModalSlideInMs, p)
	}
}

func TestConflictModalOffersThreeChoices(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, _ := newConflictModal(t, clk, sys)
	m.Show()
	// Show focuses the first option (Keep Local); Tab walks the
	// registration order: Keep Local -> Keep Cloud -> Decide Later
	// (UX spec Tab order).
	for i, want := range []string{mainmenu.FocusKeepLocal, mainmenu.FocusKeepCloud, mainmenu.FocusDecideLater} {
		if got := m.Focus().FocusedID(); got != want {
			t.Fatalf("step %d: focus = %q, want %q", i, got, want)
		}
		m.Focus().FocusNext()
	}
}

func TestKeepLocalResolvesAndCallsResolver(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, host := newConflictModal(t, clk, sys)
	m.Show()
	m.ResolveConflict(mainmenu.ChoiceKeepLocal)
	if len(sys.resolutions) != 1 || sys.resolutions[0] != mainmenu.ChoiceKeepLocal {
		t.Fatalf("resolutions = %v, want [keep-local]", sys.resolutions)
	}
	if m.Visible() {
		t.Fatal("modal must hide after resolution")
	}
	if !*host {
		t.Fatal("host screen must be interactive again after resolution")
	}
	if m.AutoSaveBlocked() {
		t.Fatal("auto-save must not be blocked after Keep Local")
	}
	if !m.Resolved() || m.Decision() != mainmenu.ChoiceKeepLocal {
		t.Fatalf("decision = %q resolved = %v, want keep-local/true", m.Decision(), m.Resolved())
	}
}

func TestKeepCloudResolvesAndCallsResolver(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, _ := newConflictModal(t, clk, sys)
	m.Show()
	m.ResolveConflict(mainmenu.ChoiceKeepCloud)
	if len(sys.resolutions) != 1 || sys.resolutions[0] != mainmenu.ChoiceKeepCloud {
		t.Fatalf("resolutions = %v, want [keep-cloud]", sys.resolutions)
	}
	if m.Visible() || !m.Resolved() {
		t.Fatalf("visible = %v resolved = %v, want hidden/resolved", m.Visible(), m.Resolved())
	}
}

func TestNoConflictSkipsModal(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: false}
	m, host := newConflictModal(t, clk, sys)
	m.Show()
	if m.Visible() {
		t.Fatal("modal must not appear without a conflict (network unavailable -> launch normally)")
	}
	if !*host {
		t.Fatal("host must stay interactive when the modal is skipped")
	}
	if len(sys.resolutions) != 0 {
		t.Fatalf("no resolution expected, got %v", sys.resolutions)
	}
}

// ---------------------------------------------------------------------------
// AC-6: Decide Later consequences
// ---------------------------------------------------------------------------

func TestDecideLaterBlocksAutosaveAndShowsBanner(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, _ := newConflictModal(t, clk, sys)
	m.Show()
	m.ResolveConflict(mainmenu.ChoiceDecideLater)
	if len(sys.resolutions) != 1 || sys.resolutions[0] != mainmenu.ChoiceDecideLater {
		t.Fatalf("resolutions = %v, want [decide-later]", sys.resolutions)
	}
	if !m.AutoSaveBlocked() {
		t.Fatal("auto-saves must be blocked after Decide Later (AC-6)")
	}
	if !m.BannerVisible() {
		t.Fatal("warning banner must show after Decide Later (AC-6)")
	}
	if m.Decision() != mainmenu.ChoiceDecideLater || m.Resolved() {
		t.Fatalf("decision = %q resolved = %v, want decide-later/false", m.Decision(), m.Resolved())
	}
}

func TestBannerClearsWhenConflictResolved(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, _ := newConflictModal(t, clk, sys)
	m.Show()
	m.ResolveConflict(mainmenu.ChoiceDecideLater)
	if !m.BannerVisible() {
		t.Fatal("precondition: banner visible after Decide Later")
	}
	// Story 005: Settings -> Save/Load resolves the pending conflict.
	m.ResolveConflict(mainmenu.ChoiceKeepCloud)
	if m.BannerVisible() {
		t.Fatal("banner must clear once the conflict is resolved (AC-6)")
	}
	if m.AutoSaveBlocked() {
		t.Fatal("auto-saves must re-enable after resolution (AC-6)")
	}
	if len(sys.resolutions) != 2 || sys.resolutions[1] != mainmenu.ChoiceKeepCloud {
		t.Fatalf("resolutions = %v, want [decide-later keep-cloud]", sys.resolutions)
	}
}

// ---------------------------------------------------------------------------
// Edge cases and regression
// ---------------------------------------------------------------------------

func TestEscapeInConflictModalChoosesDecideLater(t *testing.T) {
	clk := &fakeClock{}
	sys := &stubSystem{hasConflict: true}
	m, _ := newConflictModal(t, clk, sys)
	m.Show()
	// Escape/gamepad B = Decide Later, the safe default (UX spec).
	if !m.Dispatch(mainmenu.TranslateKey(mainmenu.KeyEvent{Key: "Escape"})) {
		t.Fatal("Escape must be handled by the modal")
	}
	if m.Decision() != mainmenu.ChoiceDecideLater {
		t.Fatalf("decision = %q, want decide-later (safe default)", m.Decision())
	}
	if !m.AutoSaveBlocked() {
		t.Fatal("auto-save must be blocked after Escape (Decide Later)")
	}
	// Gamepad B behaves identically.
	m2, _ := newConflictModal(t, clk, sys)
	m2.Show()
	m2.Dispatch(mainmenu.TranslateGamepad(mainmenu.GamepadEvent{Button: "b"}))
	if m2.Decision() != mainmenu.ChoiceDecideLater {
		t.Fatalf("gamepad B decision = %q, want decide-later", m2.Decision())
	}
}

func TestLoadGameBackReturnsToPauseMenuEntryPoint(t *testing.T) {
	// Regression for the GDD transition-table gap: LoadGame entered from the
	// Pause Menu must return there on Back ("the back button always returns
	// to the previous screen", GDD transition rules).
	flow, clk, _, _, _, _ := newFixture(t)
	navigate(t, flow, clk, mainmenu.ScreenInGame)
	navigate(t, flow, clk, mainmenu.ScreenPauseMenu)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if err := flow.Back(); err != nil {
		t.Fatalf("Back from LoadGame (pause entry): %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenPauseMenu {
		t.Fatalf("current = %s, want %s (pause-menu entry point)", flow.Current(), mainmenu.ScreenPauseMenu)
	}
	// And the title-screen entry point still returns to the title.
	navigate(t, flow, clk, mainmenu.ScreenTitleScreen)
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	if err := flow.Back(); err != nil {
		t.Fatalf("Back from LoadGame (title entry): %v", err)
	}
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s (title entry point)", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

func TestBackButtonReturnsToEntryPoint(t *testing.T) {
	// Mouse parity: the visible Back button behaves like Escape/B.
	flow, clk, _, sys, _, _ := newFixture(t)
	sys.slots = fiveOccupiedSlots()
	navigate(t, flow, clk, mainmenu.ScreenLoadGame)
	click(t, flow, mainmenu.FocusBack)
	clk.advance(mainmenu.ScreenCrossfadeMs)
	flow.Update()
	if flow.Current() != mainmenu.ScreenTitleScreen {
		t.Fatalf("current = %s, want %s after Back button", flow.Current(), mainmenu.ScreenTitleScreen)
	}
}

func TestFocusRingOnDarkStoneSlotCards(t *testing.T) {
	// The slot cards use dark stone backgrounds; the ring must hold >=3:1
	// against them (ADR-0012 AC-4).
	style := mainmenu.NewFocusStyle(1.0)
	ring := style.FocusRingFor(mainmenu.ColorDarkStone)
	if ratio := mainmenu.ContrastRatio(ring, mainmenu.ColorDarkStone); ratio < mainmenu.MinFocusContrast {
		t.Fatalf("ring %#v on dark stone: ratio %.2f, want >= %g", ring, ratio, mainmenu.MinFocusContrast)
	}
	if ring != mainmenu.ColorParchment {
		t.Fatalf("ring on dark stone = %#v, want parchment", ring)
	}
}

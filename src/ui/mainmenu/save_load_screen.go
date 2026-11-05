// save_load_screen.go implements the SaveLoadScreen controller (Story 004):
// the 22-slot save/load browser shared by the Load Game screen (entered from
// the Title Screen or the Pause Menu) and the Save screen (entered from the
// Pause Menu only, GDD CR-2 AC-7). Both screens share the slot-list layout
// but differ in interaction rules: Load activates occupied slots, Save
// offers overwrite confirmation on occupied slots and name entry on empty
// slots. Also implements the Steam Cloud conflict modal (AC-5/AC-6): a
// boot-time overlay shown before the title screen when local and cloud saves
// differ.
//
// The controller is pure Go: it owns state, focus, and timing only. All
// timing derives from the injectable clock (the same function passed to
// NewTransitionSystem). Kaiju HTML/CSS rendering is intentionally absent —
// this file exposes the data model (sorted slots, dialog state, toast, style
// contract) that a Kaiju-aware rendering pass consumes.
//
// The Save/Load System is injected, never owned: ListSlots/SaveGame/
// DeleteSave/RestoreSave and the cloud-conflict probe/resolver are
// functional options, so the screen is fully headless-testable and never
// touches the filesystem. The game layer adapts the concrete save system
// (marrow-of-empire/save) onto this contract. Slot IDs are opaque strings;
// slots padded by the screen (when the system returns fewer than 22) use the
// positional id "slot-N", which the adapter maps onto the system's slot
// layout (20 manual + auto + quick).
//
// Known limitation (Story 001 architecture note): ActionBack is handled at
// the flow level, so Escape/gamepad B inside an inline dialog exits the
// screen instead of cancelling the dialog. All dialogs carry explicit
// Cancel buttons for mouse and gamepad users.

package mainmenu

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Slot layout tuning knobs (GDD main-menu-and-settings CR-2 Formulas).
const (
	// TotalSlotCount is the number of save slots: 20 manual + 1 auto-save +
	// 1 quicksave (GDD CR-2).
	TotalSlotCount = 22
	// SaveLoadFadeMs is save_load_fade_ms: the crossfade-to-black duration
	// when a load begins (GDD: 300ms, safe range 200-600ms). Story 001's
	// flow crossfades screen-to-screen at ScreenCrossfadeMs; this knob is
	// exposed for the load-in/load-out fade owned by the Loading screen.
	SaveLoadFadeMs = 300.0
	// ToastDurationMs is the save-failure toast display duration (UX spec:
	// 5s auto-dismiss; the slot list stays fully interactive).
	ToastDurationMs = 5000.0
	// SlotDimmedOpacity is the dimmed opacity of empty slots (GDD: 40%,
	// matching DisabledButtonOpacity — "a dead control is a bleached
	// control", art bible 6.5).
	SlotDimmedOpacity = 0.4
)

// SlotMode selects the interaction rules of the shared slot browser (AC-7):
// Load mode (occupied slots load, empty slots non-interactive) or Save mode
// (occupied slots ask before overwriting, empty slots open name entry).
type SlotMode int

const (
	// SlotModeLoad is the Load Game screen: click an occupied slot to begin
	// the load; empty slots are non-interactive (AC-2).
	SlotModeLoad SlotMode = iota
	// SlotModeSave is the Save screen: overwrite confirmation on occupied
	// slots, name entry on empty slots (AC-3).
	SlotModeSave
)

// String returns the mode name for diagnostics.
func (m SlotMode) String() string {
	switch m {
	case SlotModeLoad:
		return "Load"
	case SlotModeSave:
		return "Save"
	}
	return fmt.Sprintf("SlotMode(%d)", int(m))
}

// SaveLoadState is the inline UI state of the slot browser. Dialogs are
// inline states of the screen — not ScreenFlow states (matching the title
// screen's inline dialog pattern, GDD CR-5).
type SaveLoadState int

const (
	// SaveLoadStateSlots is the slot list.
	SaveLoadStateSlots SaveLoadState = iota
	// SaveLoadStateOverwrite is the overwrite confirmation dialog (AC-3).
	SaveLoadStateOverwrite
	// SaveLoadStateNameEntry is the name-entry prompt for empty slots
	// (AC-3).
	SaveLoadStateNameEntry
	// SaveLoadStateCorrupted is the corrupted-save error dialog (AC-4).
	SaveLoadStateCorrupted
	// SaveLoadStateDeleteConfirm is the delete-save confirmation dialog
	// (AC-4, UX spec "Delete this save permanently?").
	SaveLoadStateDeleteConfirm
)

// String returns the state name for diagnostics.
func (s SaveLoadState) String() string {
	switch s {
	case SaveLoadStateSlots:
		return "Slots"
	case SaveLoadStateOverwrite:
		return "Overwrite"
	case SaveLoadStateNameEntry:
		return "NameEntry"
	case SaveLoadStateCorrupted:
		return "Corrupted"
	case SaveLoadStateDeleteConfirm:
		return "DeleteConfirm"
	}
	return fmt.Sprintf("SaveLoadState(%d)", int(s))
}

// SaveSlot is one row of the slot browser — the UI's data contract with the
// Save/Load System (GDD CR-2 / ADR-0002). The game layer adapts the concrete
// save system's slot metadata onto this type; the screen only renders what
// it is given and never queries entities directly.
type SaveSlot struct {
	// ID is the opaque system slot id passed back to SaveGame/DeleteSave/
	// RestoreSave. Empty slots padded by the screen use "slot-N".
	ID string
	// Name is the slot label ("Save 1", "Auto-save", "Quick save").
	Name string
	// CharacterName is the player character's name at save time.
	CharacterName string
	// Playtime is the total playtime at save time.
	Playtime time.Duration
	// Region is the player-facing region name at save time.
	Region string
	// Thumbnail is the path of the 320x180 PNG screenshot.
	Thumbnail string
	// Timestamp is the wall-clock save time (sort key, AC-1).
	Timestamp time.Time
	// ChecksumValid reports whether the CRC32 checksum verified (AC-4).
	// False marks the slot corrupted; the actual checksum is the Save/Load
	// System's responsibility.
	ChecksumValid bool
	// Empty marks an unoccupied slot: zero metadata, dimmed styling,
	// non-interactive in Load mode (AC-1/AC-2).
	Empty bool
	// FilePath is the .sav path, shown in the save-failure toast when the
	// system provides it (UX spec: "unable to write to [path]").
	FilePath string
}

// CloudConflictInfo is one side of a Steam Cloud conflict (AC-5): the local
// and cloud versions of the save, each with timestamp, playtime, and region.
type CloudConflictInfo struct {
	// Local is the local save version.
	Local ConflictVersion
	// Cloud is the cloud save version.
	Cloud ConflictVersion
}

// ConflictVersion is one save version inside a CloudConflictInfo.
type ConflictVersion struct {
	// Timestamp is the wall-clock save time of this version.
	Timestamp time.Time
	// Playtime is the total playtime of this version.
	Playtime time.Duration
	// Region is the player-facing region name of this version.
	Region string
}

// Steam Cloud conflict resolution choices (AC-5/AC-6). These strings are the
// contract with the Save/Load System's resolver.
const (
	// ChoiceKeepLocal keeps the local save and overwrites the cloud version.
	ChoiceKeepLocal = "keep-local"
	// ChoiceKeepCloud keeps the cloud save and overwrites the local version.
	ChoiceKeepCloud = "keep-cloud"
	// ChoiceDecideLater defers resolution: auto-saves are blocked and a
	// warning banner shows until resolved (AC-6). The unchosen version is
	// kept as .conflict_backup with a 30-day retention by the Save/Load
	// System.
	ChoiceDecideLater = "decide-later"
)

// User-facing strings.
//
// LOCALIZATION: user-facing strings. Exposed as data here because the
// project has no string table yet; route through the localization system
// when one exists (UI rule: no hardcoded strings in logic).
const (
	// LabelEmptySlot is shown on unoccupied slots with dimmed styling
	// (AC-1).
	LabelEmptySlot = "Empty Slot"
	// LabelBack is the visible Back button (mouse parity with Escape/B).
	LabelBack = "Back"
	// LabelOverwrite is the confirm button of the overwrite dialog (AC-3).
	LabelOverwrite = "Overwrite"
	// LabelSave is the confirm button of the name-entry prompt (AC-3).
	LabelSave = "Save"
	// LabelCancel is the cancel button of every dialog (AC-3/AC-4).
	LabelCancel = "Cancel"
	// LabelRestoreFromBackup is the restore action of the corrupted-save
	// dialog (AC-4).
	LabelRestoreFromBackup = "Restore from backup"
	// LabelDeleteSave is the delete action of the corrupted-save dialog
	// (AC-4).
	LabelDeleteSave = "Delete save"
	// LabelKeepLocal is the conflict modal's "Keep Local" choice (AC-5).
	LabelKeepLocal = "Keep Local"
	// LabelKeepCloud is the conflict modal's "Keep Cloud" choice (AC-5).
	LabelKeepCloud = "Keep Cloud"
	// LabelDecideLater is the conflict modal's "Decide Later" choice
	// (AC-5/AC-6).
	LabelDecideLater = "Decide Later"
	// CorruptedSaveMessage is the corrupted-save error dialog body (AC-4).
	CorruptedSaveMessage = "This save file is corrupted."
	// CloudConflictMessage is the conflict modal body (AC-5, GDD CR-2).
	CloudConflictMessage = "Your local and cloud saves differ. Which would you like to keep?"
	// AutoSaveBannerMessage is the persistent warning banner shown on the
	// title screen and pause menu after "Decide Later" (AC-6, GDD CR-2).
	AutoSaveBannerMessage = "Auto-save disabled — resolve Steam Cloud conflict in Settings → Save/Load."
	// NoValidBackupMessage is shown when restoring fails or no backup exists
	// (AC-4 edge case).
	NoValidBackupMessage = "No valid backup found."
	// DeleteConfirmMessage is the delete-save confirmation dialog body
	// (UX spec).
	DeleteConfirmMessage = "Delete this save permanently?"
	// SaveFailedToastFormat formats the save-failure toast (UX spec). The
	// first %s is the .sav path when the system provides it, else the slot
	// name.
	SaveFailedToastFormat = "Save failed — unable to write to %s. Check folder permissions."
	// OverwriteConfirmFormat formats the overwrite dialog title (AC-3):
	// "Overwrite Alric — 12:30?" with the slot name and playtime.
	OverwriteConfirmFormat = "Overwrite %s — %s?"
	// NameEntryDefaultFormat formats the default save name for empty slots
	// (AC-3, UX spec): "[Character Name] · [Timestamp]".
	NameEntryDefaultFormat = "%s · %s"
)

// FormatPlaytime renders a playtime as H:MM ("12:30" for 12h30m, AC-3 QA).
func FormatPlaytime(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Hours()), int(d.Minutes())%60)
}

// FormatTimestamp renders a save timestamp as "2006-01-02 15:04" (AC-1).
func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// SortSlots orders slots by last-modified timestamp descending (AC-1: most
// recent first). Empty slots — which carry no timestamp — sort last, keeping
// the occupied slots at the top of the list. The sort is stable, so equal
// timestamps keep their system-provided order.
func SortSlots(slots []SaveSlot) {
	sort.SliceStable(slots, func(i, j int) bool {
		ei, ej := slots[i].Empty, slots[j].Empty
		if ei != ej {
			return !ei
		}
		if ei {
			return false
		}
		return slots[i].Timestamp.After(slots[j].Timestamp)
	})
}

// SlotCardStyle is the slot-card styling contract for the Kaiju CSS pass
// (GDD CR-2 implementation notes): dark stone card background (NOT
// parchment — metadata is illegible on parchment at 1.17:1), pale ivory
// text at >=12:1 contrast, dimmed opacity for empty slots, and the selected
// slot's amber gold border + ink underline (two cues — never color alone).
type SlotCardStyle struct {
	// Background is the card background: dark stone.
	Background Color
	// Text is the metadata text color: pale ivory parchment.
	Text Color
	// DimmedOpacity is the opacity of empty slots (40%).
	DimmedOpacity float64
	// SelectedBorder is the selected slot's border color: amber gold.
	SelectedBorder Color
	// SelectedUnderline is the selected slot's underline color: ink.
	SelectedUnderline Color
}

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements. Slot ids are
// positional ("slot-N" for N in 0..TotalSlotCount-1) so they stay stable
// across list re-sorts; the activation handler reads the slot at the index
// at activation time. The "save-load-" prefix keeps these distinct from the
// title screen's ids, registered in the same document while screens share
// the flow.
const (
	FocusSlotID = "slot-%d"
	// FocusBack is the visible Back button (Escape/B parity for mouse).
	FocusBack = "save-load-back"
	// FocusOverwriteConfirm is the overwrite dialog's confirm button.
	FocusOverwriteConfirm = "overwrite-confirm"
	// FocusOverwriteCancel is the overwrite dialog's cancel button.
	FocusOverwriteCancel = "overwrite-cancel"
	// FocusSaveNameConfirm is the name-entry prompt's save button.
	FocusSaveNameConfirm = "name-confirm"
	// FocusSaveNameCancel is the name-entry prompt's cancel button.
	FocusSaveNameCancel = "name-cancel"
	// FocusCorruptRestore is the corrupted dialog's restore button.
	FocusCorruptRestore = "corrupt-restore"
	// FocusCorruptDelete is the corrupted dialog's delete button.
	FocusCorruptDelete = "corrupt-delete"
	// FocusCorruptCancel is the corrupted dialog's cancel button.
	FocusCorruptCancel = "corrupt-cancel"
	// FocusDeleteConfirm is the delete confirmation's confirm button.
	FocusDeleteConfirm = "delete-confirm"
	// FocusDeleteCancel is the delete confirmation's cancel button.
	FocusDeleteCancel = "delete-cancel"
	// FocusToastDismiss is the toast's dismiss button (UX spec: toast is
	// dismissible and auto-dismisses after ToastDurationMs).
	FocusToastDismiss = "toast-dismiss"
)

// SaveLoadScreenOption configures a SaveLoadScreen at construction time.
type SaveLoadScreenOption func(*SaveLoadScreen)

// WithSaveLoadClock installs the game-time clock (milliseconds) that drives
// the toast auto-dismiss timer. Pass the same function given to
// NewTransitionSystem so the screen and screen transitions share one time
// source.
func WithSaveLoadClock(now func() float64) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.now = now }
}

// WithListSlots installs the slot-list probe (AC-1): returns every save
// slot the system knows about. Empty slots may be omitted — the screen pads
// the list to TotalSlotCount. Without it the screen shows 22 empty slots.
func WithListSlots(fn func() []SaveSlot) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.listSlotsFn = fn }
}

// WithSaveGame installs the save command (AC-3): writes the save to the
// slot with the given name. A non-nil error surfaces as the save-failure
// toast (UX spec edge case).
func WithSaveGame(fn func(slotID, name string) error) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.saveGameFn = fn }
}

// WithDeleteSave installs the delete command (AC-4): removes the slot's
// save file after the player confirms.
func WithDeleteSave(fn func(slotID string) error) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.deleteSaveFn = fn }
}

// WithRestoreSave installs the restore command (AC-4): restores the slot
// from its backup. A non-nil error surfaces as the "No valid backup found"
// message.
func WithRestoreSave(fn func(slotID string) error) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.restoreSaveFn = fn }
}

// WithHasBackup installs the backup-existence probe (AC-4 edge case): true
// when the slot has a restore-able backup. Without it the screen assumes a
// backup exists (the Restore button is enabled; a failed restore still
// surfaces the error message).
func WithHasBackup(fn func(slotID string) bool) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.hasBackupFn = fn }
}

// WithCharacterName installs the current-character-name probe (AC-3): the
// player character's name, used in the empty-slot name-entry default
// ("[character] · [timestamp]"). Without it the default falls back to
// DefaultCharacterName.
func WithCharacterName(fn func() string) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.characterNameFn = fn }
}

// WithCurrentTime installs the wall-clock probe used by the name-entry
// default timestamp (AC-3). Without it the screen uses time.Now.
func WithCurrentTime(fn func() time.Time) SaveLoadScreenOption {
	return func(s *SaveLoadScreen) { s.currentTimeFn = fn }
}

// SaveLoadScreen is the shared save/load slot browser controller (Story
// 004). It implements ScreenController and registers for ScreenLoadGame or
// ScreenSaveGame (AC-7): the game layer creates one instance per screen with
// the matching SlotMode, so the mode is fixed per registration and the two
// screens can never drift apart in interaction rules.
//
// All probes and commands are injected (functional options), so the
// controller is fully headless-testable: no filesystem, no engine, no
// goroutines. Timing is a pure function of the injectable clock; the load
// operation itself runs through the flow's BeginLoad (Story 001 AC-9
// machinery, hard cut + SaveLoader seam).
type SaveLoadScreen struct {
	flow  *ScreenFlow
	focus *FocusManager
	now   func() float64

	listSlotsFn     func() []SaveSlot
	saveGameFn      func(slotID, name string) error
	deleteSaveFn    func(slotID string) error
	restoreSaveFn   func(slotID string) error
	hasBackupFn     func(slotID string) bool
	characterNameFn func() string
	currentTimeFn   func() time.Time

	mode        SlotMode
	state       SaveLoadState
	interactive bool
	slots       []SaveSlot

	// Dialog targets (indices into slots at the time the dialog opened).
	overwriteIdx int
	corruptIdx   int
	deleteIdx    int

	// Name-entry value (defaulted at dialog open, editable via SetNameEntry).
	nameEntry string

	// Toast state (UX spec: non-blocking overlay, slot list stays
	// interactive; auto-dismiss after ToastDurationMs via the clock).
	toastMsg    string
	toastAt     float64
	toastActive bool
}

// NewSaveLoadScreen creates the slot browser controller for the given flow
// and mode. The flow is required for navigation (BeginLoad, Back); the
// caller registers the screen with flow.RegisterController(ScreenLoadGame,
// screen) or flow.RegisterController(ScreenSaveGame, screen) — one instance
// per screen (AC-7).
func NewSaveLoadScreen(flow *ScreenFlow, mode SlotMode, opts ...SaveLoadScreenOption) *SaveLoadScreen {
	s := &SaveLoadScreen{
		flow:        flow,
		focus:       NewFocusManager(NewFocusStyle(1.0)),
		now:         func() float64 { return 0 },
		mode:        mode,
		state:       SaveLoadStateSlots,
		interactive: true,
		slots:       make([]SaveSlot, 0, TotalSlotCount),
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	if s.now == nil {
		s.now = func() float64 { return 0 }
	}
	if s.currentTimeFn == nil {
		s.currentTimeFn = time.Now
	}
	s.registerFocusables()
	s.refreshSlots()
	return s
}

// registerFocusables registers every focusable element in Tab order: the
// 22 slot cards, the back button, and the dialog/toast controls. Elements
// are registered once and enabled/disabled per state, so the Tab order is
// stable across inline state changes.
func (s *SaveLoadScreen) registerFocusables() {
	for i := 0; i < TotalSlotCount; i++ {
		idx := i
		_ = s.focus.Register(fmt.Sprintf(FocusSlotID, idx),
			WithGroup("slots"), WithBackground(ColorDarkStone),
			WithActivate(func() { s.onSlotActivate(idx) }))
	}
	_ = s.focus.Register(FocusBack, WithGroup("slots"), WithBackground(ColorParchment), WithActivate(func() { s.onBack() }))
	_ = s.focus.Register(FocusOverwriteConfirm, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onOverwriteConfirm))
	_ = s.focus.Register(FocusOverwriteCancel, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onOverwriteCancel))
	_ = s.focus.Register(FocusSaveNameConfirm, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onNameConfirm))
	_ = s.focus.Register(FocusSaveNameCancel, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onNameCancel))
	_ = s.focus.Register(FocusCorruptRestore, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onCorruptRestore))
	_ = s.focus.Register(FocusCorruptDelete, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onCorruptDelete))
	_ = s.focus.Register(FocusCorruptCancel, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onCorruptCancel))
	_ = s.focus.Register(FocusDeleteConfirm, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onDeleteConfirm))
	_ = s.focus.Register(FocusDeleteCancel, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onDeleteCancel))
	_ = s.focus.Register(FocusToastDismiss, WithBackground(ColorParchment), WithActivate(s.DismissToast))
}

// OnEnter re-polls the slot list (so the screen always renders fresh
// metadata, AC-1), resets to the slot-list state, and focuses the first
// enabled slot.
func (s *SaveLoadScreen) OnEnter(prev Screen) {
	s.state = SaveLoadStateSlots
	s.refreshSlots()
	s.refreshEnabled()
	s.focus.FocusNext()
}

// OnExit clears focus and the toast.
func (s *SaveLoadScreen) OnExit() {
	s.focus.ClearFocus()
	s.toastActive = false
	s.toastMsg = ""
}

// Focus returns the screen's focus manager for input routing.
func (s *SaveLoadScreen) Focus() *FocusManager { return s.focus }

// SetInteractive toggles screen interactivity (mirroring the title screen's
// Story 003 hook): elements stay registered but are skipped by focus
// traversal and reject activation while non-interactive.
func (s *SaveLoadScreen) SetInteractive(on bool) {
	s.interactive = on
	s.refreshEnabled()
}

// Mode returns the interaction mode (Load or Save).
func (s *SaveLoadScreen) Mode() SlotMode { return s.mode }

// State returns the current inline UI state.
func (s *SaveLoadScreen) State() SaveLoadState { return s.state }

// Slots returns the current slot list, sorted by timestamp descending with
// empty slots last (AC-1). Treat the returned slice as read-only.
func (s *SaveLoadScreen) Slots() []SaveSlot { return s.slots }

// SlotCount returns the number of displayed slots (TotalSlotCount).
func (s *SaveLoadScreen) SlotCount() int { return len(s.slots) }

// SlotFocusID returns the positional focusable id of slot i ("slot-N").
func (s *SaveLoadScreen) SlotFocusID(i int) string {
	return fmt.Sprintf(FocusSlotID, i)
}

// SlotDimmed reports whether slot i must render dimmed (AC-1): empty slots
// at 40% opacity. In Load mode empty slots are also non-interactive.
func (s *SaveLoadScreen) SlotDimmed(i int) bool {
	return i >= 0 && i < len(s.slots) && s.slots[i].Empty
}

// SelectedSlotID returns the focusable id of the currently selected slot,
// or "" when no slot is selected (the focus is on a dialog control). The
// selected slot renders with the amber border + ink underline (two cues).
func (s *SaveLoadScreen) SelectedSlotID() string {
	id := s.focus.FocusedID()
	if id == "" || !strings.HasPrefix(id, "slot-") {
		return ""
	}
	return id
}

// SelectedSlotIndex returns the index of the selected slot, or -1.
func (s *SaveLoadScreen) SelectedSlotIndex() int {
	id := s.SelectedSlotID()
	if id == "" {
		return -1
	}
	var idx int
	if _, err := fmt.Sscanf(id, FocusSlotID, &idx); err != nil {
		return -1
	}
	return idx
}

// OverwriteTitle returns the overwrite dialog title (AC-3):
// "Overwrite Alric — 12:30?". Empty when no overwrite dialog is open.
func (s *SaveLoadScreen) OverwriteTitle() string {
	if s.state != SaveLoadStateOverwrite {
		return ""
	}
	slot := s.slotAt(s.overwriteIdx)
	if slot == nil {
		return ""
	}
	return fmt.Sprintf(OverwriteConfirmFormat, slot.Name, FormatPlaytime(slot.Playtime))
}

// CorruptedMessage returns the corrupted-save dialog body (AC-4). Empty
// when no corrupted dialog is open.
func (s *SaveLoadScreen) CorruptedMessage() string {
	if s.state != SaveLoadStateCorrupted {
		return ""
	}
	return CorruptedSaveMessage
}

// DeleteConfirmMessage returns the delete-confirmation dialog body. Empty
// when no delete confirmation is open.
func (s *SaveLoadScreen) DeleteConfirmMessage() string {
	if s.state != SaveLoadStateDeleteConfirm {
		return ""
	}
	return DeleteConfirmMessage
}

// RestoreDisabled reports whether the corrupted dialog's Restore button
// must be greyed out (AC-4 edge case, UX spec): no backup exists for the
// corrupted slot.
func (s *SaveLoadScreen) RestoreDisabled() bool {
	if s.state != SaveLoadStateCorrupted {
		return false
	}
	slot := s.slotAt(s.corruptIdx)
	if slot == nil {
		return true
	}
	return s.hasBackupFn != nil && !s.hasBackupFn(slot.ID)
}

// NameEntryValue returns the current name-entry text (defaulted at dialog
// open; the rendering pass binds the input field to this).
func (s *SaveLoadScreen) NameEntryValue() string { return s.nameEntry }

// SetNameEntry sets the name-entry text, trimmed of surrounding whitespace.
func (s *SaveLoadScreen) SetNameEntry(name string) {
	s.nameEntry = strings.TrimSpace(name)
}

// ToastMessage returns the current toast text, or "" when no toast is
// visible.
func (s *SaveLoadScreen) ToastMessage() string { return s.toastMsg }

// ToastVisible reports whether the toast is currently displayed (UX spec:
// non-blocking overlay; the slot list stays fully interactive).
func (s *SaveLoadScreen) ToastVisible() bool {
	if !s.toastActive {
		return false
	}
	return s.toastRemaining() > 0
}

// ToastRemaining returns the toast's remaining display time in
// milliseconds (0 when no toast is visible or it has expired).
func (s *SaveLoadScreen) ToastRemaining() float64 {
	if !s.toastActive {
		return 0
	}
	return s.toastRemaining()
}

// toastRemaining computes the remaining toast time from the clock.
func (s *SaveLoadScreen) toastRemaining() float64 {
	remaining := ToastDurationMs - (s.now() - s.toastAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// DismissToast hides the toast immediately (UX spec: dismissible).
func (s *SaveLoadScreen) DismissToast() {
	s.toastActive = false
	s.toastMsg = ""
	s.refreshEnabled()
}

// Refresh re-applies the focusable enabled states (toast auto-dismiss
// expiry, backup-probe changes). Call once per frame from the game layer so
// clock- and probe-driven state updates the buttons reactively.
func (s *SaveLoadScreen) Refresh() {
	s.refreshEnabled()
}

// SlotCardStyle returns the slot-card styling contract for the Kaiju CSS
// pass (dark stone card, ivory text, amber+ink selection, 40% dimming).
func (s *SaveLoadScreen) SlotCardStyle() SlotCardStyle {
	return SlotCardStyle{
		Background:       ColorDarkStone,
		Text:             ColorParchment,
		DimmedOpacity:    SlotDimmedOpacity,
		SelectedBorder:   ColorAmber,
		SelectedUnderline: ColorInk,
	}
}

// refreshSlots re-polls the slot list and sorts it (AC-1). The list is
// padded to TotalSlotCount with empty slots; padded slots use the
// positional id "slot-N" so the Save/Load adapter can map them onto the
// system slot layout. Call after any mutation (save, delete, restore).
func (s *SaveLoadScreen) refreshSlots() {
	slots := make([]SaveSlot, 0, TotalSlotCount)
	if s.listSlotsFn != nil {
		slots = append(slots, s.listSlotsFn()...)
	}
	SortSlots(slots)
	if len(slots) > TotalSlotCount {
		slots = slots[:TotalSlotCount]
	}
	for i := len(slots); i < TotalSlotCount; i++ {
		slots = append(slots, SaveSlot{
			ID:    fmt.Sprintf(FocusSlotID, i),
			Name:  "",
			Empty: true,
		})
	}
	s.slots = slots
}

// slotAt returns the slot at index i, or nil when out of range.
func (s *SaveLoadScreen) slotAt(i int) *SaveSlot {
	if i < 0 || i >= len(s.slots) {
		return nil
	}
	return &s.slots[i]
}

// refreshEnabled re-applies the enabled state of every focusable element
// from the current state, the interactivity flag, and the mode (AC-2/AC-3/
// AC-7). Call after any of those change.
func (s *SaveLoadScreen) refreshEnabled() {
	slots := s.interactive && s.state == SaveLoadStateSlots
	for i := range s.slots {
		enabled := slots
		if enabled {
			slot := s.slots[i]
			// Load mode: occupied slots only (empty slots are
			// non-interactive, AC-2). Corrupted slots stay clickable —
			// they open the error dialog instead of loading (AC-4).
			if s.mode == SlotModeLoad && slot.Empty {
				enabled = false
			}
		}
		_ = s.focus.SetEnabled(fmt.Sprintf(FocusSlotID, i), enabled)
	}
	_ = s.focus.SetEnabled(FocusBack, s.interactive && s.state == SaveLoadStateSlots)

	overwrite := s.interactive && s.state == SaveLoadStateOverwrite
	_ = s.focus.SetEnabled(FocusOverwriteConfirm, overwrite)
	_ = s.focus.SetEnabled(FocusOverwriteCancel, overwrite)

	name := s.interactive && s.state == SaveLoadStateNameEntry
	_ = s.focus.SetEnabled(FocusSaveNameConfirm, name)
	_ = s.focus.SetEnabled(FocusSaveNameCancel, name)

	corrupted := s.interactive && s.state == SaveLoadStateCorrupted
	_ = s.focus.SetEnabled(FocusCorruptRestore, corrupted && !s.RestoreDisabled())
	_ = s.focus.SetEnabled(FocusCorruptDelete, corrupted)
	_ = s.focus.SetEnabled(FocusCorruptCancel, corrupted)

	del := s.interactive && s.state == SaveLoadStateDeleteConfirm
	_ = s.focus.SetEnabled(FocusDeleteConfirm, del)
	_ = s.focus.SetEnabled(FocusDeleteCancel, del)

	_ = s.focus.SetEnabled(FocusToastDismiss, s.interactive && s.toastActive)
}

// setState switches the inline UI state and refocuses the first enabled
// element of the new state.
func (s *SaveLoadScreen) setState(st SaveLoadState) {
	s.state = st
	s.refreshEnabled()
	s.focus.FocusNext()
}

// backToSlots returns to the slot list, re-polling the slots first so
// mutations (save/delete/restore) are reflected immediately.
func (s *SaveLoadScreen) backToSlots() {
	s.refreshSlots()
	s.setState(SaveLoadStateSlots)
}

// onBack is the visible Back button (mouse parity with Escape/B): the flow
// owns the back destination (entry point, AC-8).
func (s *SaveLoadScreen) onBack() {
	if s.state != SaveLoadStateSlots || !s.interactive {
		return
	}
	if s.flow != nil {
		_ = s.flow.Back()
	}
}

// onSlotActivate is the click/Enter activation of slot i (AC-2/AC-3/AC-4).
// The slot is read at activation time, so list re-sorts never desync the
// positional ids.
func (s *SaveLoadScreen) onSlotActivate(i int) {
	if s.state != SaveLoadStateSlots || !s.interactive {
		return
	}
	slot := s.slotAt(i)
	if slot == nil {
		return
	}
	switch s.mode {
	case SlotModeLoad:
		if slot.Empty {
			return // empty slots are non-interactive in Load mode (AC-2)
		}
		if !slot.ChecksumValid {
			s.corruptIdx = i
			s.setState(SaveLoadStateCorrupted) // never attempt to load (AC-4)
			return
		}
		s.beginLoad(slot.ID)
	case SlotModeSave:
		if slot.Empty {
			s.overwriteIdx = i
			s.nameEntry = s.defaultSaveName()
			s.setState(SaveLoadStateNameEntry) // name entry on empty (AC-3)
			return
		}
		s.overwriteIdx = i
		s.setState(SaveLoadStateOverwrite) // confirm before overwriting (AC-3)
	}
}

// beginLoad starts the load operation (AC-2): the flow hard-cuts to Loading,
// runs the injected SaveLoader, and resolves to InGame (success) or
// TitleScreen with the error (Story 001 AC-9). The screen never touches the
// filesystem.
func (s *SaveLoadScreen) beginLoad(slotID string) {
	if s.flow == nil {
		return
	}
	_ = s.flow.BeginLoad(slotID)
}

// defaultSaveName builds the empty-slot name-entry default (AC-3, UX spec):
// "[Character Name] · [Timestamp]".
func (s *SaveLoadScreen) defaultSaveName() string {
	character := DefaultCharacterName
	if s.characterNameFn != nil {
		if name := s.characterNameFn(); name != "" {
			character = name
		}
	}
	return fmt.Sprintf(NameEntryDefaultFormat, character, FormatTimestamp(s.currentTime()))
}

// currentTime returns the wall-clock time (injectable for determinism).
func (s *SaveLoadScreen) currentTime() time.Time {
	if s.currentTimeFn != nil {
		return s.currentTimeFn()
	}
	return time.Now()
}

// onOverwriteConfirm confirms the overwrite (AC-3): SaveGame with the
// existing slot name; a failure surfaces as the toast and the player stays
// on the slot list.
func (s *SaveLoadScreen) onOverwriteConfirm() {
	if s.state != SaveLoadStateOverwrite || !s.interactive {
		return
	}
	slot := s.slotAt(s.overwriteIdx)
	if slot == nil {
		s.backToSlots()
		return
	}
	s.runSave(slot.ID, slot.Name)
}

// onOverwriteCancel cancels the overwrite dialog and returns to the slot
// list (AC-3).
func (s *SaveLoadScreen) onOverwriteCancel() {
	if s.state != SaveLoadStateOverwrite || !s.interactive {
		return
	}
	s.backToSlots()
}

// onNameConfirm saves with the entered name (AC-3): SaveGame with the
// name-entry value, falling back to the default on empty input.
func (s *SaveLoadScreen) onNameConfirm() {
	if s.state != SaveLoadStateNameEntry || !s.interactive {
		return
	}
	slot := s.slotAt(s.overwriteIdx)
	if slot == nil {
		s.backToSlots()
		return
	}
	name := s.nameEntry
	if name == "" {
		name = s.defaultSaveName()
	}
	s.runSave(slot.ID, name)
}

// onNameCancel cancels the name-entry prompt and returns to the slot list.
func (s *SaveLoadScreen) onNameCancel() {
	if s.state != SaveLoadStateNameEntry || !s.interactive {
		return
	}
	s.backToSlots()
}

// runSave executes the injected SaveGame command and handles the outcome
// (AC-3): success returns to the refreshed slot list; failure shows the
// save-failure toast (UX spec: slot list remains fully interactive, toast
// auto-dismisses after ToastDurationMs).
func (s *SaveLoadScreen) runSave(slotID, name string) {
	if s.saveGameFn != nil {
		if err := s.saveGameFn(slotID, name); err != nil {
			s.backToSlots()
			s.showToast(s.saveFailedMessage(slotID))
			return
		}
	}
	s.backToSlots()
}

// saveFailedMessage formats the save-failure toast, preferring the slot's
// .sav path when the system provided one, then the player-facing slot name
// (UX spec: "unable to write to [path]").
func (s *SaveLoadScreen) saveFailedMessage(slotID string) string {
	where := slotID
	if slot := s.findSlot(slotID); slot != nil {
		if slot.FilePath != "" {
			where = slot.FilePath
		} else if slot.Name != "" {
			where = slot.Name
		}
	}
	return fmt.Sprintf(SaveFailedToastFormat, where)
}

// findSlot returns the slot with the given id, or nil.
func (s *SaveLoadScreen) findSlot(slotID string) *SaveSlot {
	for i := range s.slots {
		if s.slots[i].ID == slotID {
			return &s.slots[i]
		}
	}
	return nil
}

// onCorruptRestore restores the corrupted slot from backup (AC-4). Without
// a backup (or when the restore fails) the dialog closes, the slot list
// becomes interactive again, and the "No valid backup found" toast shows
// (QA edge case: backup also corrupted; UX spec: toast is non-blocking).
func (s *SaveLoadScreen) onCorruptRestore() {
	if s.state != SaveLoadStateCorrupted || !s.interactive {
		return
	}
	slot := s.slotAt(s.corruptIdx)
	if slot == nil {
		s.backToSlots()
		return
	}
	if s.hasBackupFn != nil && !s.hasBackupFn(slot.ID) {
		s.backToSlots()
		s.showToast(NoValidBackupMessage)
		return
	}
	if s.restoreSaveFn != nil {
		if err := s.restoreSaveFn(slot.ID); err != nil {
			s.backToSlots()
			s.showToast(NoValidBackupMessage)
			return
		}
	}
	s.backToSlots()
}

// onCorruptDelete opens the delete confirmation (AC-4: "Delete save" asks
// for confirmation before calling DeleteSave).
func (s *SaveLoadScreen) onCorruptDelete() {
	if s.state != SaveLoadStateCorrupted || !s.interactive {
		return
	}
	s.deleteIdx = s.corruptIdx
	s.setState(SaveLoadStateDeleteConfirm)
}

// onCorruptCancel closes the corrupted dialog and returns to the slot list
// (AC-4).
func (s *SaveLoadScreen) onCorruptCancel() {
	if s.state != SaveLoadStateCorrupted || !s.interactive {
		return
	}
	s.backToSlots()
}

// onDeleteConfirm confirms the delete (AC-4): DeleteSave removes the slot,
// then the list refreshes. Failures are silent at this layer — the delete
// command's error belongs to the Save/Load System's error surface.
func (s *SaveLoadScreen) onDeleteConfirm() {
	if s.state != SaveLoadStateDeleteConfirm || !s.interactive {
		return
	}
	slot := s.slotAt(s.deleteIdx)
	if slot != nil && s.deleteSaveFn != nil {
		_ = s.deleteSaveFn(slot.ID)
	}
	s.backToSlots()
}

// onDeleteCancel abandons the delete and returns to the corrupted dialog.
func (s *SaveLoadScreen) onDeleteCancel() {
	if s.state != SaveLoadStateDeleteConfirm || !s.interactive {
		return
	}
	s.setState(SaveLoadStateCorrupted)
}

// showToast displays the non-blocking toast for ToastDurationMs (UX spec:
// the slot list remains fully interactive).
func (s *SaveLoadScreen) showToast(msg string) {
	s.toastMsg = msg
	s.toastAt = s.now()
	s.toastActive = true
	s.refreshEnabled()
}

// ---------------------------------------------------------------------------
// Cloud conflict modal (AC-5/AC-6)
// ---------------------------------------------------------------------------

// CloudConflictModalOption configures a CloudConflictModal at construction
// time.
type CloudConflictModalOption func(*CloudConflictModal)

// WithConflictClock installs the game-time clock (milliseconds) that drives
// the modal slide-in. Pass the same function given to NewTransitionSystem
// so the modal and screen transitions share one time source.
func WithConflictClock(now func() float64) CloudConflictModalOption {
	return func(m *CloudConflictModal) { m.now = now }
}

// WithHasCloudConflict installs the conflict probe (AC-5): true when Steam
// Cloud detects that local and cloud saves differ. Show() is a no-op when
// it reports false.
func WithHasCloudConflict(fn func() bool) CloudConflictModalOption {
	return func(m *CloudConflictModal) { m.hasConflictFn = fn }
}

// WithCloudConflictInfo installs the conflict-data probe (AC-5): the local
// and cloud versions (timestamp, playtime, region for each). Read when the
// modal shows.
func WithCloudConflictInfo(fn func() CloudConflictInfo) CloudConflictModalOption {
	return func(m *CloudConflictModal) { m.infoFn = fn }
}

// WithResolveCloudConflict installs the resolution command (AC-5/AC-6):
// invoked with ChoiceKeepLocal, ChoiceKeepCloud, or ChoiceDecideLater. The
// Save/Load System owns the .conflict_backup rename and 30-day retention.
func WithResolveCloudConflict(fn func(choice string)) CloudConflictModalOption {
	return func(m *CloudConflictModal) { m.resolveFn = fn }
}

// WithSetInteractive installs the host-screen interactivity hook (mirroring
// the accessibility prompt's title hook): the modal disables the title menu
// while visible and re-enables it on dismissal. The game layer passes
// title.SetInteractive.
func WithSetInteractive(fn func(on bool)) CloudConflictModalOption {
	return func(m *CloudConflictModal) { m.setInteractiveFn = fn }
}

// Focusable element ids of the conflict modal. The "conflict-" prefix keeps
// them distinct from the title screen's ids, registered in the same
// document while the modal overlays it.
const (
	FocusKeepLocal   = "conflict-keep-local"
	FocusKeepCloud   = "conflict-keep-cloud"
	FocusDecideLater = "conflict-decide-later"
)

// CloudConflictModal is the Steam Cloud conflict overlay (AC-5/AC-6): shown
// on game launch before the title screen when local and cloud saves differ,
// presenting both versions (timestamp, playtime, region) with three choices.
// It is NOT a flow screen — it overlays the title screen and the owner
// (game layer) drives it: call Show at boot, route translated input to
// Dispatch while Visible(), and read AutoSaveBlocked/BannerVisible for the
// AC-6 consequences. Story 005 (Settings → Save/Load) resolves a pending
// conflict through the public ResolveConflict.
type CloudConflictModal struct {
	focus *FocusManager
	now   func() float64

	hasConflictFn    func() bool
	infoFn           func() CloudConflictInfo
	resolveFn        func(choice string)
	setInteractiveFn func(on bool)

	info      CloudConflictInfo
	visible   bool
	enteredAt float64
	decision  string // ChoiceKeepLocal / ChoiceKeepCloud / ChoiceDecideLater / ""
	resolved  bool   // false while a Decide Later choice is still pending
}

// NewCloudConflictModal creates the conflict modal. Options inject the
// clock, the conflict probe/data, the resolver, and the host interactivity
// hook.
func NewCloudConflictModal(opts ...CloudConflictModalOption) *CloudConflictModal {
	m := &CloudConflictModal{
		focus: NewFocusManager(NewFocusStyle(1.0)),
		now:   func() float64 { return 0 },
	}
	for _, o := range opts {
		if o != nil {
			o(m)
		}
	}
	if m.now == nil {
		m.now = func() float64 { return 0 }
	}
	_ = m.focus.Register(FocusKeepLocal, WithGroup("conflict"), WithBackground(ColorParchment), WithActivate(func() { m.ResolveConflict(ChoiceKeepLocal) }))
	_ = m.focus.Register(FocusKeepCloud, WithGroup("conflict"), WithBackground(ColorParchment), WithActivate(func() { m.ResolveConflict(ChoiceKeepCloud) }))
	_ = m.focus.Register(FocusDecideLater, WithGroup("conflict"), WithBackground(ColorParchment), WithActivate(func() { m.ResolveConflict(ChoiceDecideLater) }))
	return m
}

// Show presents the modal (AC-5). With no conflict (probe false or absent)
// it is a no-op: the game launches normally (edge case: network unavailable
// → deferred, game launches normally). The host screen becomes
// non-interactive while the modal is visible.
func (m *CloudConflictModal) Show() {
	if m.visible {
		return
	}
	if m.hasConflictFn != nil && !m.hasConflictFn() {
		return
	}
	m.visible = true
	m.enteredAt = m.now()
	if m.infoFn != nil {
		m.info = m.infoFn()
	}
	if m.setInteractiveFn != nil {
		m.setInteractiveFn(false)
	}
	m.focus.FocusNext()
}

// Hide dismisses the modal and restores host interactivity. Idempotent.
func (m *CloudConflictModal) Hide() {
	if !m.visible {
		return
	}
	m.visible = false
	m.focus.ClearFocus()
	if m.setInteractiveFn != nil {
		m.setInteractiveFn(true)
	}
}

// Focus returns the modal's own focus manager for input routing.
func (m *CloudConflictModal) Focus() *FocusManager { return m.focus }

// Dispatch routes a translated input to the modal (the owner calls this
// while Visible() instead of the flow). Escape and gamepad B choose
// "Decide Later" — the safe default that never destroys either version
// (UX spec: B = Decide Later). Tab/arrows move focus; Enter/Space/A/click
// activate.
func (m *CloudConflictModal) Dispatch(in Input) bool {
	if !m.visible {
		return false
	}
	if in.Action == ActionBack {
		m.ResolveConflict(ChoiceDecideLater)
		return true
	}
	return dispatchFocusAction(m.focus, in)
}

// Visible reports whether the modal is currently displayed.
func (m *CloudConflictModal) Visible() bool { return m.visible }

// Info returns the conflict data currently shown (local vs cloud version).
func (m *CloudConflictModal) Info() CloudConflictInfo { return m.info }

// Decision returns the last resolution choice, or "" before any choice
// (Story 005 reads this).
func (m *CloudConflictModal) Decision() string { return m.decision }

// Resolved reports whether the conflict has been resolved (a Keep Local /
// Keep Cloud choice, or a Decide Later superseded by a later resolution).
func (m *CloudConflictModal) Resolved() bool { return m.resolved }

// AutoSaveBlocked reports whether auto-saves must be blocked (AC-6): true
// while the last resolution was "Decide Later". The game layer's auto-save
// trigger must check this before writing.
func (m *CloudConflictModal) AutoSaveBlocked() bool { return m.decision == ChoiceDecideLater }

// BannerVisible reports whether the warning banner must render on the title
// screen and pause menu (AC-6, GDD CR-2): true while a Decide Later choice
// is still pending. Resolution (Story 005) clears it.
func (m *CloudConflictModal) BannerVisible() bool { return m.AutoSaveBlocked() }

// AppearProgress returns the modal slide-in progress in [0,1]:
// clamp((now - enteredAt) / ModalSlideInMs). The rendering pass animates
// the panel from this (paper-slide motion, ADR-0012). Reports 0 before the
// modal is shown and after it hides.
func (m *CloudConflictModal) AppearProgress() float64 {
	if !m.visible {
		return 0
	}
	e := m.now() - m.enteredAt
	if e <= 0 {
		return 0
	}
	if e >= ModalSlideInMs {
		return 1
	}
	return e / ModalSlideInMs
}

// ResolveConflict applies a resolution choice (AC-5/AC-6): records the
// decision, invokes the injected resolver (the Save/Load System owns the
// overwrite and the .conflict_backup handling), and hides the modal.
// Public so Story 005 (Settings → Save/Load) can resolve a pending conflict
// from the settings screen. Unknown choices are ignored.
func (m *CloudConflictModal) ResolveConflict(choice string) {
	switch choice {
	case ChoiceKeepLocal, ChoiceKeepCloud:
		m.decision = choice
		m.resolved = true
	case ChoiceDecideLater:
		m.decision = choice
		m.resolved = false // auto-saves blocked until a later resolution
	default:
		return
	}
	if m.resolveFn != nil {
		m.resolveFn(choice)
	}
	m.Hide()
}

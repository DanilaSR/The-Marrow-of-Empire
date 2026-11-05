// keybindings_screen.go implements the KeybindingsScreen controller (Story
// 007): the Controls → Keybindings sub-screen with per-action primary and
// secondary key slots, click-to-capture binding, conflict auto-unbind,
// modifier key support, reserved-key warnings (AC-5/AC-6), default
// restoration (AC-4), gamepad preset selection (AC-7), and debounced
// settings.cfg persistence (AC-8).
//
// Like the rest of the package, the controller is pure Go: it owns state,
// focus, and timing only. All timing derives from the injectable clock. The
// Kaiju HTML/CSS rendering pass consumes the per-frame render commands; raw
// key events during capture reach the screen through HandleKey (the game
// layer routes every raw key to the screen while CapturesInput() is true).
// Routing contract: HandleKey returning false means the key is not part of
// the capture — the game layer must then dispatch it to the flow (dialog
// buttons stay keyboard-reachable, ADR-0012 "key-capture mode" + tab order).
//
// The action table is injected data (WithActions) — the Input Binding System
// owns the authoritative action registry (its GDD is a separate Foundation
// system; the screen reads and writes the BindingRegistry it constructs from
// the table). Persistence uses Story 005's dotted-key settings.cfg contract:
// "controls.bindings.move_forward.primary" = "W",
// "controls.gamepad_preset" = "Default".
//
// Escape-during-capture reconciliation (AC-1 vs AC-5): Escape is a reserved
// key (pause menu) and AC-5 requires it to be rebindable, so a captured
// Escape cannot silently cancel — it opens the AC-5 warning dialog, whose
// Cancel path is the AC-1 cancel ("binding unchanged, capture mode exits").

package mainmenu

import (
	"fmt"
	"math"
	"strings"
)

// BindingSlot identifies one of an action's two key slots (CR-6 binding
// model: each action has a primary and a secondary key slot).
type BindingSlot int

const (
	// SlotPrimary is the action's primary binding slot.
	SlotPrimary BindingSlot = iota
	// SlotSecondary is the action's secondary binding slot.
	SlotSecondary
)

// String returns the slot name ("primary"/"secondary") — the settings.cfg
// key segment and the diagnostic name.
func (s BindingSlot) String() string {
	if s == SlotSecondary {
		return "secondary"
	}
	return "primary"
}

// BindingCategory groups the action table rows (story implementation note:
// "Rows are grouped by category (Movement, Camera, Combat, UI, System)").
type BindingCategory int

const (
	// BindingCategoryMovement holds movement actions.
	BindingCategoryMovement BindingCategory = iota
	// BindingCategoryCamera holds camera actions.
	BindingCategoryCamera
	// BindingCategoryCombat holds combat actions.
	BindingCategoryCombat
	// BindingCategoryUI holds interface actions (journal, map, pause,
	// quick-load).
	BindingCategoryUI
	// BindingCategorySystem holds system actions.
	BindingCategorySystem
)

// String returns the category name for diagnostics.
func (c BindingCategory) String() string {
	switch c {
	case BindingCategoryMovement:
		return "Movement"
	case BindingCategoryCamera:
		return "Camera"
	case BindingCategoryCombat:
		return "Combat"
	case BindingCategoryUI:
		return "UI"
	case BindingCategorySystem:
		return "System"
	}
	return fmt.Sprintf("BindingCategory(%d)", int(c))
}

// BindingAction is one row of the keybinding table: a stable action ID (the
// settings.cfg key segment), the player-facing label, the display category,
// the default primary/secondary chords, and the AC-6 reserved flag. The
// table is data-driven and injected (the Input Binding System owns the
// authoritative action registry).
type BindingAction struct {
	// ID is the stable settings.cfg key segment ("move_forward").
	ID string
	// Name is the registry key and the player-facing function name used in
	// warnings. The reserved functions use the ActionBinding constants
	// (ActionPauseMenu = "Pause Menu", ActionQuickLoad = "Quick Load") so
	// AC-6 warnings read naturally; empty means the ID is the registry key.
	Name ActionBinding
	// Label is the player-facing row label.
	//
	// LOCALIZATION: user-facing string. Exposed as data because the project
	// has no string table yet; route through the localization system when
	// one exists (UI rule: no hardcoded strings in logic).
	Label string
	// Category groups the row in the table display.
	Category BindingCategory
	// DefaultPrimary / DefaultSecondary are the AC-4 restore values. The
	// zero chord means "no default binding".
	DefaultPrimary   KeyChord
	DefaultSecondary KeyChord
	// Reserved marks a function whose last binding can never be removed
	// (AC-6: Escape = pause menu, F9 = quick-load).
	Reserved bool
}

// RegistryName returns the action's registry key: the explicit Name
// (reserved functions use their player-facing ActionBinding constants so
// AC-6 warnings read naturally) or the stable ID.
func (a BindingAction) RegistryName() ActionBinding {
	if a.Name != "" {
		return a.Name
	}
	return ActionBinding(a.ID)
}

// GamepadPreset is one selectable gamepad layout (AC-7). MVP presets are
// static data — not per-button remappable — and the Input Binding System
// applies the selected preset at runtime (out of scope here; the screen
// only selects and persists).
type GamepadPreset struct {
	// Key is the stable settings.cfg value ("Default", "Southpaw",
	// "Left-Handed" — the story's persistence contract).
	Key string
	// Label is the player-facing dropdown text.
	Label string
	// Bindings maps each action ID to its gamepad button in this layout.
	// The data is exposed for the game layer; the screen never edits it.
	Bindings map[string]string
}

// KeybindingsState is the inline UI state of the keybinding screen. The
// dialogs are inline — not ScreenFlow states (matching the settings
// screen's dialog pattern).
type KeybindingsState int

const (
	// KeybindingsStatePanel is the action table.
	KeybindingsStatePanel KeybindingsState = iota
	// KeybindingsStateCapture is capture mode: a slot is highlighted (amber
	// gold pulsing border) with a "Press a key..." prompt, and the next key
	// (or key + modifier combination) is bound on key-down. All keys are
	// consumed by the screen — not dispatched to the game.
	KeybindingsStateCapture
	// KeybindingsStateReservedWarn is the AC-5 dialog: the captured key is
	// reserved (Escape/F9). "Bind Anyway" completes the binding (subject to
	// the AC-6 last-binding gate); "Cancel" restores the previous binding
	// and exits capture mode.
	KeybindingsStateReservedWarn
	// KeybindingsStateRestoreAll is the AC-4 restore-all confirmation
	// dialog ("Restore all keybindings to defaults?").
	KeybindingsStateRestoreAll
)

// String returns the state name for diagnostics.
func (s KeybindingsState) String() string {
	switch s {
	case KeybindingsStatePanel:
		return "Panel"
	case KeybindingsStateCapture:
		return "Capture"
	case KeybindingsStateReservedWarn:
		return "ReservedWarn"
	case KeybindingsStateRestoreAll:
		return "RestoreAll"
	}
	return fmt.Sprintf("KeybindingsState(%d)", int(s))
}

// Timing and capture tuning knobs (GDD main-menu-and-settings CR-6;
// data-driven).
const (
	// KeybindingsDebounceMs is the settings flush debounce (same value as
	// Story 005's DebounceMs): how long the flush waits after the last
	// binding change before writing settings.cfg (AC-8).
	KeybindingsDebounceMs = 500.0
	// CapturePulseMs is the capture highlight pulse period: the amber gold
	// border "breathes" gently at this rate (CapturePulse returns the
	// phase). Purely cosmetic; never a state cue (ADR-0012: color changes
	// are never the sole indicator).
	CapturePulseMs = 800.0
)

// Settings keys (AC-8 dotted-key contract).
const (
	// SettingBindingPrimaryFmt / SettingBindingSecondaryFmt format a
	// binding key: "controls.bindings.move_forward.primary" = "W".
	SettingBindingPrimaryFmt   = "controls.bindings.%s.primary"
	SettingBindingSecondaryFmt = "controls.bindings.%s.secondary"
	// SettingGamepadPreset is the selected gamepad preset key
	// ("controls.gamepad_preset" = "Default").
	SettingGamepadPreset = "controls.gamepad_preset"
	// SettingUnboundValue is the persisted value of a cleared slot
	// ("controls.bindings.interact.primary" = "Unbound"). It is the
	// settings.cfg TOKEN, distinct from the display label (LabelUnbound):
	// ParseChord maps it back onto the zero chord so a conflict-cleared slot
	// stays unbound across reboots (AC-8) instead of resurrecting its
	// default. An empty value would be indistinguishable from a missing key.
	SettingUnboundValue = "Unbound"
)

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements. Row ids embed the
// action ID's kebab-case form (bindingFocusID) so they are stable across
// rebuilds and match the settings screen's id convention (settingFocusID:
// schema keys with . and _ replaced by -).
const (
	// FocusPrimaryIDFmt / FocusSecondaryIDFmt / FocusRestoreIDFmt format a
	// row's slot and restore ids ("kb-row-move-forward-primary"). The %s is
	// the kebab-case action ID (bindingFocusID).
	FocusPrimaryIDFmt   = "kb-row-%s-primary"
	FocusSecondaryIDFmt = "kb-row-%s-secondary"
	FocusRestoreIDFmt   = "kb-row-%s-restore"
	// FocusGamepadPreset is the gamepad preset dropdown (AC-7).
	FocusGamepadPreset = "kb-gamepad-preset"
	// FocusRestoreAll is the footer "Restore All Defaults" button (AC-4).
	FocusRestoreAll = "kb-restore-all"
	// FocusKeybindingsBack is the visible Back button (distinct id from the
	// save/load screens' FocusBack).
	FocusKeybindingsBack = "kb-back"
	// FocusReservedBind / FocusReservedCancel are the AC-5 dialog buttons.
	FocusReservedBind   = "kb-reserved-bind"
	FocusReservedCancel = "kb-reserved-cancel"
	// FocusRestoreAllConfirm / FocusRestoreAllCancel are the AC-4 dialog
	// buttons.
	FocusRestoreAllConfirm = "kb-restoreall-confirm"
	FocusRestoreAllCancel  = "kb-restoreall-cancel"
)

// User-facing strings.
//
// LOCALIZATION: user-facing strings. Exposed as data because the project has
// no string table yet; route through the localization system when one exists
// (UI rule: no hardcoded strings in logic).
const (
	LabelUnbound            = "Unbound"
	LabelPressKey           = "Press a key..."
	LabelRestoreDefaults    = "Restore Defaults"
	LabelRestoreAllDefaults = "Restore All Defaults"
	LabelGamepadPreset      = "Gamepad Preset"
	LabelBindAnyway         = "Bind Anyway"
	LabelConfirm            = "Confirm"
	// RestoreAllConfirmMessage is the AC-4 restore-all dialog prompt.
	RestoreAllConfirmMessage = "Restore all keybindings to defaults?"
	// LabelBack (visible Back button) and LabelCancel (dialog cancel
	// buttons) are declared in save_load_screen.go and reused here.
)

// ReservedWarningMessage returns the AC-5 warning:
// "This key is reserved for [function]. Rebinding may conflict with system
// behaviour."
func ReservedWarningMessage(function ActionBinding) string {
	return fmt.Sprintf("This key is reserved for %s. Rebinding may conflict with system behaviour.", function)
}

// reservedKeys maps the system-critical keys to their reserved functions
// (AC-5). The map is static: the KEY is reserved for the function
// regardless of current bindings (Escape is pause-menu's key even after the
// player rebinds pause to something else).
var reservedKeys = map[string]ActionBinding{
	"Escape": ActionPauseMenu,
	"F9":     ActionQuickLoad,
}

// modifierKeyNames are the key names the game layer reports for modifier
// key-down events. Left/right variants collapse onto the modifier flag
// (AC-3: "right/left modifier treated identically" — KeyEvent carries
// booleans, not sides); both "Control" and "Ctrl" are accepted.
var modifierKeyNames = map[string]bool{
	"Shift":   true,
	"Control": true,
	"Ctrl":    true,
	"Alt":     true,
}

// SerializeChord renders a chord as its settings.cfg value and display
// text: "W", "Ctrl+S", "Ctrl+Shift+Alt+K". Modifiers are emitted in the
// fixed order Ctrl, Shift, Alt; the zero chord serializes to "" (unbound
// slot).
func SerializeChord(c KeyChord) string {
	if c.Key == "" {
		return ""
	}
	parts := make([]string, 0, 4)
	if c.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if c.Shift {
		parts = append(parts, "Shift")
	}
	if c.Alt {
		parts = append(parts, "Alt")
	}
	parts = append(parts, c.Key)
	return strings.Join(parts, "+")
}

// ParseChord converts a settings.cfg value back onto a chord. Modifier
// tokens are order-lenient ("Shift+Ctrl+S" parses like "Ctrl+Shift+S") so a
// hand-edited settings.cfg round-trips; the SettingUnboundValue token maps
// onto the zero chord (a cleared slot stays cleared, AC-8); unknown tokens,
// an empty value, a missing key, or an implausible key name fail (callers
// keep their default — AC-8 edge case: invalid binding data falls back to
// defaults).
func ParseChord(s string) (KeyChord, bool) {
	if s == "" {
		return KeyChord{}, false
	}
	// The persisted cleared-slot token (AC-8): zero chord, valid value.
	if s == SettingUnboundValue {
		return KeyChord{}, true
	}
	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return KeyChord{}, false
	}
	chord := KeyChord{Key: parts[len(parts)-1]}
	for _, p := range parts[:len(parts)-1] {
		switch p {
		case "Ctrl":
			chord.Ctrl = true
		case "Shift":
			chord.Shift = true
		case "Alt":
			chord.Alt = true
		default:
			return KeyChord{}, false
		}
	}
	if !validKeyName(chord.Key) {
		return KeyChord{}, false
	}
	return chord, true
}

// validKeyName reports whether s is a plausible key name: one or more ASCII
// letters, digits, or spaces ("W", "F5", "PageUp", "Left Bracket") and not
// itself a modifier name (a binding of "Shift" alone is not a base key —
// modifiers only combine with a base key, AC-3). Hand-edited garbage such as
// "???" is rejected so corrupt settings.cfg values fall back to the action
// defaults.
func validKeyName(s string) bool {
	if s == "" || modifierKeyNames[s] {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ' ' {
			return false
		}
	}
	return true
}

// defaultBindingActions is the CR-6 action table (Story 007's data-driven
// default; the game layer may inject its own via WithActions). Pause Menu
// and Quick Load are the AC-6 reserved functions; their registry names are
// the player-facing ActionBinding constants so the AC-6 warning reads
// naturally.
var defaultBindingActions = []BindingAction{
	// ── Movement ──
	{ID: "move_forward", Label: "Move Forward", Category: BindingCategoryMovement, DefaultPrimary: KeyChord{Key: "W"}},
	{ID: "move_backward", Label: "Move Backward", Category: BindingCategoryMovement, DefaultPrimary: KeyChord{Key: "S"}},
	{ID: "strafe_left", Label: "Strafe Left", Category: BindingCategoryMovement, DefaultPrimary: KeyChord{Key: "A"}},
	{ID: "strafe_right", Label: "Strafe Right", Category: BindingCategoryMovement, DefaultPrimary: KeyChord{Key: "D"}},
	// ── Camera ──
	{ID: "zoom_in", Label: "Zoom In", Category: BindingCategoryCamera, DefaultPrimary: KeyChord{Key: "PageUp"}},
	{ID: "zoom_out", Label: "Zoom Out", Category: BindingCategoryCamera, DefaultPrimary: KeyChord{Key: "PageDown"}},
	{ID: "center_party", Label: "Center on Party", Category: BindingCategoryCamera, DefaultPrimary: KeyChord{Key: "Home"}},
	{ID: "center_selected", Label: "Center on Selected", Category: BindingCategoryCamera, DefaultPrimary: KeyChord{Key: "End"}},
	// ── Combat ──
	{ID: "attack", Label: "Attack", Category: BindingCategoryCombat, DefaultPrimary: KeyChord{Key: "F"}},
	{ID: "interact", Label: "Interact", Category: BindingCategoryCombat, DefaultPrimary: KeyChord{Key: "E"}},
	// ── UI ──
	{ID: "journal", Label: "Journal", Category: BindingCategoryUI, DefaultPrimary: KeyChord{Key: "J"}},
	{ID: "map", Label: "Map", Category: BindingCategoryUI, DefaultPrimary: KeyChord{Key: "M"}},
	{ID: "pause_menu", Name: ActionPauseMenu, Label: string(ActionPauseMenu), Category: BindingCategoryUI, Reserved: true,
		DefaultPrimary: KeyChord{Key: "Escape"}},
	{ID: "quick_load", Name: ActionQuickLoad, Label: string(ActionQuickLoad), Category: BindingCategoryUI, Reserved: true,
		DefaultPrimary: KeyChord{Key: "F9"}},
}

// defaultGamepadPresets is the AC-7 MVP preset table. Presets are static
// data (not per-button remappable); the Input Binding System applies the
// selected preset's button map at runtime (out of scope here — the screen
// selects and persists only).
var defaultGamepadPresets = []GamepadPreset{
	{Key: "Default", Label: "Default", Bindings: map[string]string{
		"move_forward": "left_stick_up", "move_backward": "left_stick_down",
		"strafe_left": "left_stick_left", "strafe_right": "left_stick_right",
		"interact": "a", "attack": "x", "journal": "y", "map": "b",
		"center_party": "left_bumper", "center_selected": "right_bumper",
		"zoom_in": "dpad_up", "zoom_out": "dpad_down",
	}},
	{Key: "Southpaw", Label: "Southpaw", Bindings: map[string]string{
		"move_forward": "right_stick_up", "move_backward": "right_stick_down",
		"strafe_left": "right_stick_left", "strafe_right": "right_stick_right",
		"interact": "a", "attack": "x", "journal": "y", "map": "b",
		"center_party": "left_bumper", "center_selected": "right_bumper",
		"zoom_in": "dpad_up", "zoom_out": "dpad_down",
	}},
	{Key: "Left-Handed", Label: "Left-Handed", Bindings: map[string]string{
		"move_forward": "left_stick_up", "move_backward": "left_stick_down",
		"strafe_left": "left_stick_left", "strafe_right": "left_stick_right",
		"interact": "x", "attack": "a", "journal": "b", "map": "y",
		"center_party": "right_bumper", "center_selected": "left_bumper",
		"zoom_in": "dpad_up", "zoom_out": "dpad_down",
	}},
}

// KeybindingsScreenOption configures a KeybindingsScreen at construction
// time.
type KeybindingsScreenOption func(*KeybindingsScreen)

// WithKeybindingsClock installs the game-time clock (milliseconds) that
// drives the debounce flush and the capture pulse. Pass the same function
// given to NewTransitionSystem so the screen and screen transitions share
// one time source.
func WithKeybindingsClock(now func() float64) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.now = now }
}

// WithActions installs the action table (the Input Binding System's action
// registry contract; injected because that system owns the authoritative
// registry). Defaults to the built-in CR-6 table.
func WithActions(actions []BindingAction) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) {
		if actions != nil {
			s.actions = append([]BindingAction{}, actions...)
		}
	}
}

// WithKeybindingsReader installs the settings.cfg read callback (AC-8):
// returns the stored value for a key, or "" when absent (a partial
// settings.cfg is valid — missing or unparsable binding values keep the
// defaults). Named distinctly from the settings screen's WithSettingsReader
// (same package).
func WithKeybindingsReader(fn func(key string) string) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.getSettingFn = fn }
}

// WithKeybindingsWriter installs the persistence seam (AC-8, Story 005's
// pattern): invoked with each dirty key and its serialized value when the
// debounce flush fires (or on Flush). A non-nil error keeps the key dirty
// for the next retry. The game layer adapter owns the settings.cfg file I/O.
func WithKeybindingsWriter(fn func(key, value string) error) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.writeFn = fn }
}

// WithKeybindingsHasSettingsFile installs the settings.cfg existence probe
// (AC-8 edge case): true when a valid settings.cfg exists. Load uses it to
// decide between store pre-population and first-launch defaults.
func WithKeybindingsHasSettingsFile(fn func() bool) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.hasSettingsFn = fn }
}

// WithKeybindingsLogger installs the error-log seam: the pure core never
// touches a logging package; the engine-coupled game layer passes
// slog.Warn-style logging. Defaults to a no-op.
func WithKeybindingsLogger(fn func(format string, args ...any)) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.logFn = fn }
}

// WithGamepadPresets overrides the AC-7 preset table (the game layer may
// ship a different layout set; tests inject small tables).
func WithGamepadPresets(presets []GamepadPreset) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) {
		if presets != nil {
			s.presets = append([]GamepadPreset{}, presets...)
		}
	}
}

// WithBindingChange installs the binding-change notification seam: invoked
// with the action's registry name, the slot, and the new chord (the zero
// chord = slot cleared) after every successful capture, restore, and
// conflict unbind. The game layer forwards it to the Input Binding System,
// the runtime owner of the binding map.
func WithBindingChange(fn func(action string, slot BindingSlot, chord KeyChord)) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.changeFn = fn }
}

// WithGamepadPresetChange installs the preset-change notification seam:
// invoked with the newly selected preset key. The game layer forwards it to
// the Input Binding System so gameplay reflects the new layout immediately
// (AC-7).
func WithGamepadPresetChange(fn func(presetKey string)) KeybindingsScreenOption {
	return func(s *KeybindingsScreen) { s.presetFn = fn }
}

// KeybindingsScreen is the Controls → Keybindings screen controller (Story
// 007). It implements ScreenController and registers for ScreenKeybindings,
// entered from the Settings screen's Controls category. It owns the
// BindingRegistry built from the injected action table, the capture state
// machine, the conflict/reserved logic, the gamepad preset selection, and
// the debounced settings.cfg persistence.
//
// All probes and callbacks are injected (functional options), so the
// controller is fully headless-testable: no filesystem, no engine, no
// goroutines. Timing is a pure function of the injectable clock.
type KeybindingsScreen struct {
	flow  *ScreenFlow
	focus *FocusManager
	now   func() float64

	getSettingFn  func(key string) string
	writeFn       func(key, value string) error
	hasSettingsFn func() bool
	logFn         func(format string, args ...any)
	changeFn      func(action string, slot BindingSlot, chord KeyChord)
	presetFn      func(presetKey string)

	actions  []BindingAction
	presets  []GamepadPreset
	registry *BindingRegistry
	warned   map[ActionBinding]bool

	state      KeybindingsState
	entryPoint Screen

	// Capture state.
	captureAction ActionBinding // target action while capturing
	captureSlot   BindingSlot   // target slot while capturing
	pendingChord  KeyChord      // accumulated modifiers (Capture) or the full captured chord (ReservedWarn)
	reservedFn    ActionBinding // reserved function of the captured key (ReservedWarn)

	presetIndex int

	// Persistence state (AC-8, Story 005 pattern).
	dirtyKeys     map[string]string // key -> serialized value
	lastChangeMs  float64
	writeFailures int

	// blockedMsg is the AC-6 block warning banner ("This is the only binding
	// for Pause Menu. It cannot be removed."), shown until the next capture
	// or screen re-entry.
	blockedMsg string
}

// NewKeybindingsScreen creates the keybinding screen controller for the
// given flow. The flow is required for navigation (Back); the caller
// registers the screen with flow.RegisterController(ScreenKeybindings,
// screen). Options inject the clock, the action table, the persistence
// seams, the preset table, and the change notifications.
func NewKeybindingsScreen(flow *ScreenFlow, opts ...KeybindingsScreenOption) *KeybindingsScreen {
	s := &KeybindingsScreen{
		flow:        flow,
		focus:       NewFocusManager(NewFocusStyle(1.0)),
		now:         func() float64 { return 0 },
		logFn:       func(string, ...any) {},
		actions:     append([]BindingAction{}, defaultBindingActions...),
		presets:     append([]GamepadPreset{}, defaultGamepadPresets...),
		warned:      map[ActionBinding]bool{},
		state:       KeybindingsStatePanel,
		dirtyKeys:   map[string]string{},
		presetIndex: 0,
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	if s.now == nil {
		s.now = func() float64 { return 0 }
	}
	s.buildRegistry()
	s.registerFocusables()
	s.refreshEnabled()
	return s
}

// buildRegistry constructs the screen's BindingRegistry from the action
// table: every action gets its default primary and secondary chords (the
// zero chord = unbound slot), and reserved actions are marked so their last
// binding can never be removed (AC-6).
func (s *KeybindingsScreen) buildRegistry() {
	s.registry = NewBindingRegistry()
	for _, a := range s.actions {
		name := a.RegistryName()
		if a.Reserved {
			s.registry.MarkReserved(name)
		}
		s.registry.BindSlot(name, int(SlotPrimary), a.DefaultPrimary)
		s.registry.BindSlot(name, int(SlotSecondary), a.DefaultSecondary)
	}
}

// actionDef returns the table entry for a registry name, or nil.
func (s *KeybindingsScreen) actionDef(name ActionBinding) *BindingAction {
	for i := range s.actions {
		if s.actions[i].RegistryName() == name {
			return &s.actions[i]
		}
	}
	return nil
}

// bindingKey returns the settings.cfg dotted key for an action's slot
// (AC-8): "controls.bindings.move_forward.primary". The key segment is the
// table entry's stable ID — not the registry name, which for the reserved
// functions is the player-facing warning name ("Pause Menu").
func (s *KeybindingsScreen) bindingKey(action ActionBinding, slot BindingSlot) string {
	id := string(action)
	if def := s.actionDef(action); def != nil {
		id = def.ID
	}
	format := SettingBindingPrimaryFmt
	if slot == SlotSecondary {
		format = SettingBindingSecondaryFmt
	}
	return fmt.Sprintf(format, id)
}

// bindingFocusID converts a settings-key action ID ("move_forward") to its
// kebab-case focusable id segment ("move-forward"). The focusable ids are
// the click-target contract: the Kaiju rendering pass binds markup onclick
// handlers to these ids, and the flow's mouse path activates them by id
// (TranslateMouse -> FocusManager.ActivateAt), so the ids registered here,
// emitted in RenderCommands, and bound in markup must agree. Kebab-case
// matches the settings screen's settingFocusID convention.
func bindingFocusID(id string) string {
	return strings.ReplaceAll(id, "_", "-")
}

// registerFocusables registers every focusable element in Tab order: each
// action row's primary slot, secondary slot, and restore button; the
// gamepad preset dropdown; the footer (restore all, back); and the dialog
// buttons. Elements are registered once and enabled/disabled per state, so
// the Tab order is stable across inline state changes.
func (s *KeybindingsScreen) registerFocusables() {
	for _, a := range s.actions {
		act := a
		name := act.RegistryName()
		_ = s.focus.Register(fmt.Sprintf(FocusPrimaryIDFmt, bindingFocusID(act.ID)),
			WithGroup("table"), WithBackground(ColorParchment),
			WithActivate(func() { s.beginCapture(name, SlotPrimary) }))
		_ = s.focus.Register(fmt.Sprintf(FocusSecondaryIDFmt, bindingFocusID(act.ID)),
			WithGroup("table"), WithBackground(ColorParchment),
			WithActivate(func() { s.beginCapture(name, SlotSecondary) }))
		_ = s.focus.Register(fmt.Sprintf(FocusRestoreIDFmt, bindingFocusID(act.ID)),
			WithGroup("table"), WithBackground(ColorParchment),
			WithActivate(func() { s.RestoreActionDefaults(name) }))
	}
	_ = s.focus.Register(FocusGamepadPreset, WithGroup("footer"), WithBackground(ColorParchment),
		WithActivate(s.cycleGamepadPreset))
	_ = s.focus.Register(FocusRestoreAll, WithGroup("footer"), WithBackground(ColorParchment),
		WithActivate(s.onRestoreAll))
	_ = s.focus.Register(FocusKeybindingsBack, WithGroup("footer"), WithBackground(ColorParchment),
		WithActivate(s.onBack))
	_ = s.focus.Register(FocusReservedBind, WithGroup("dialog"), WithBackground(ColorParchment),
		WithActivate(s.onReservedBind))
	_ = s.focus.Register(FocusReservedCancel, WithGroup("dialog"), WithBackground(ColorParchment),
		WithActivate(s.onReservedCancel))
	_ = s.focus.Register(FocusRestoreAllConfirm, WithGroup("dialog"), WithBackground(ColorParchment),
		WithActivate(s.onRestoreAllConfirm))
	_ = s.focus.Register(FocusRestoreAllCancel, WithGroup("dialog"), WithBackground(ColorParchment),
		WithActivate(s.onRestoreAllCancel))
}

// Load runs the AC-8 boot read: with a valid settings.cfg the registry is
// pre-populated from the store (invalid or unparsable values keep the
// action defaults — a partial settings.cfg is valid); without one, all
// defaults remain and every key is marked dirty so the next flush creates
// the file (mirroring Story 005's AC-5). Also restores the stored gamepad
// preset. Call once at boot, before the title screen becomes interactive.
func (s *KeybindingsScreen) Load() {
	if s.hasSettingsFn == nil || !s.hasSettingsFn() {
		for _, a := range s.actions {
			name := a.RegistryName()
			s.dirtyKeys[s.bindingKey(name, SlotPrimary)] = persistChordValue(a.DefaultPrimary)
			s.dirtyKeys[s.bindingKey(name, SlotSecondary)] = persistChordValue(a.DefaultSecondary)
		}
		s.dirtyKeys[SettingGamepadPreset] = s.SelectedGamepadPreset().Key
		s.lastChangeMs = s.now()
		return
	}
	if s.getSettingFn == nil {
		return
	}
	for _, a := range s.actions {
		name := a.RegistryName()
		if v, ok := ParseChord(s.getSettingFn(s.bindingKey(name, SlotPrimary))); ok {
			s.registry.BindSlot(name, int(SlotPrimary), v)
		}
		if v, ok := ParseChord(s.getSettingFn(s.bindingKey(name, SlotSecondary))); ok {
			s.registry.BindSlot(name, int(SlotSecondary), v)
		}
	}
	if key := s.getSettingFn(SettingGamepadPreset); key != "" {
		if idx := s.presetIndexByKey(key); idx >= 0 {
			s.presetIndex = idx
		}
	}
}

// OnEnter resets to the panel state and focuses the first row. The binding
// registry is already loaded (Load runs at boot); the screen only mirrors
// it.
func (s *KeybindingsScreen) OnEnter(prev Screen) {
	s.entryPoint = prev
	s.state = KeybindingsStatePanel
	s.blockedMsg = ""
	s.refreshEnabled()
	s.focus.FocusNext()
}

// OnExit clears focus and cancels any active capture (no binding change).
func (s *KeybindingsScreen) OnExit() {
	s.focus.ClearFocus()
	s.state = KeybindingsStatePanel
	s.captureAction = ""
	s.pendingChord = KeyChord{}
	s.reservedFn = ""
}

// Focus returns the screen's focus manager for input routing.
func (s *KeybindingsScreen) Focus() *FocusManager { return s.focus }

// ---------------------------------------------------------------------------
// Capture mode (AC-1/AC-3/AC-5)
// ---------------------------------------------------------------------------

// beginCapture enters capture mode for the given slot (AC-1): the slot
// highlights (amber gold pulsing border — CapturePulse), a "Press a key..."
// prompt appears, and every subsequent key is consumed by the screen until a
// base key completes the capture or the capture is cancelled.
func (s *KeybindingsScreen) beginCapture(action ActionBinding, slot BindingSlot) {
	if s.state != KeybindingsStatePanel {
		return
	}
	s.blockedMsg = ""
	s.captureAction = action
	s.captureSlot = slot
	s.pendingChord = KeyChord{}
	s.reservedFn = ""
	s.setState(KeybindingsStateCapture)
}

// CapturesInput reports whether the screen consumes every raw key: capture
// mode and the AC-5 reserved-key dialog. While true, the game layer must
// route every raw key to HandleKey FIRST; a HandleKey return of false means
// the key is not part of the capture and must be dispatched to the flow
// (dialog buttons keep working — accessibility, tab order is preserved).
func (s *KeybindingsScreen) CapturesInput() bool {
	return s.state == KeybindingsStateCapture || s.state == KeybindingsStateReservedWarn
}

// HandleKey routes a raw key event during capture (AC-1/AC-3/AC-5):
//
//   - a modifier key alone (Shift/Ctrl/Alt) accumulates and capture stays
//     active waiting for the base key (AC-3);
//   - a reserved base key (Escape/F9) opens the AC-5 warning dialog — its
//     Cancel path is the AC-1 capture cancel;
//   - any other base key completes the capture immediately.
//
// In the ReservedWarn state only Escape is consumed (it cancels the dialog);
// every other key returns false — the game layer must dispatch it to the
// flow so the dialog's buttons (Bind Anyway / Cancel) stay reachable with
// Enter/Tab/Space, not just the mouse (accessibility, keyboard dead-end fix).
//
// Returns false when the screen is not capturing or the key is not part of
// the capture — the game layer must dispatch the key to the flow.
func (s *KeybindingsScreen) HandleKey(e KeyEvent) bool {
	switch s.state {
	case KeybindingsStateCapture:
		if modifierKeyNames[e.Key] {
			s.accumulateModifier(e)
			return true
		}
		chord := KeyChord{
			Key:   e.Key,
			Shift: s.pendingChord.Shift || e.Shift,
			Ctrl:  s.pendingChord.Ctrl || e.Ctrl,
			Alt:   s.pendingChord.Alt || e.Alt,
		}
		if fn, ok := reservedKeys[e.Key]; ok {
			s.pendingChord = chord
			s.reservedFn = fn
			s.setState(KeybindingsStateReservedWarn)
			return true
		}
		s.finishCapture(chord)
		return true
	case KeybindingsStateReservedWarn:
		// The dialog is not a key trap: only Escape is consumed (it cancels
		// the capture, AC-1/AC-5). Every other key returns false so the game
		// layer dispatches it to the flow — the dialog buttons (focused via
		// the standard tab order) activate with Enter/Space like any other
		// focusable, and the dialog's Cancel stays the mouse-independent
		// escape hatch (see onReservedCancel).
		if e.Key == "Escape" {
			s.cancelCapture()
			return true
		}
		return false
	}
	return false
}

// accumulateModifier records a modifier key for the in-progress capture
// (AC-3): a modifier alone never completes the binding — capture stays
// active waiting for the base key. Left/right variants collapse onto the
// flag (KeyEvent carries booleans, not sides).
func (s *KeybindingsScreen) accumulateModifier(e KeyEvent) {
	switch e.Key {
	case "Shift":
		s.pendingChord.Shift = true
	case "Control", "Ctrl":
		s.pendingChord.Ctrl = true
	case "Alt":
		s.pendingChord.Alt = true
	}
	// A modifier event may also carry its own pressed flags; fold them in.
	s.pendingChord.Shift = s.pendingChord.Shift || e.Shift
	s.pendingChord.Ctrl = s.pendingChord.Ctrl || e.Ctrl
	s.pendingChord.Alt = s.pendingChord.Alt || e.Alt
}

// finishCapture completes a capture: conflict resolution (AC-2), the AC-6
// last-reserved-binding gate, the slot bind, warning bookkeeping, and
// persistence. The cascade stops at one hop — only the directly conflicting
// action is unbound (GDD edge case), and the final state always maps every
// key to at most one action.
func (s *KeybindingsScreen) finishCapture(chord KeyChord) {
	target := s.captureAction
	victim := s.findConflict(target, chord)
	if victim != "" {
		if s.registry.IsReserved(victim) && s.registry.BindingCount(victim) == 1 {
			// AC-6: the cascade would strip the reserved function's last
			// binding — the operation is blocked, binding unchanged.
			s.blockedMsg = (LastReservedBindingError{Action: victim}).Error()
			s.exitCapture()
			return
		}
		// The victim may hold the chord in BOTH slots (a legal same-action
		// double-bind); clear every matching slot — the cascade still stops
		// at one hop (one victim action), but no key can remain shared.
		for _, slot := range []BindingSlot{SlotPrimary, SlotSecondary} {
			if s.chordAt(victim, slot) != chord {
				continue
			}
			_ = s.registry.ClearSlot(victim, int(slot)) // cannot fail: checked above
			s.warned[victim] = true
			s.dirtyKey(victim, slot, KeyChord{})
			if s.changeFn != nil {
				s.changeFn(string(victim), slot, KeyChord{})
			}
		}
	}
	s.blockedMsg = "" // a successful bind clears any stale AC-6 banner
	s.registry.BindSlot(target, int(s.captureSlot), chord)
	delete(s.warned, target)
	s.dirtyKey(target, s.captureSlot, chord)
	if s.changeFn != nil {
		s.changeFn(string(target), s.captureSlot, chord)
	}
	s.exitCapture()
}

// exitCapture leaves capture mode and refocuses the panel.
func (s *KeybindingsScreen) exitCapture() {
	s.captureAction = ""
	s.pendingChord = KeyChord{}
	s.reservedFn = ""
	s.setState(KeybindingsStatePanel)
}

// cancelCapture exits capture mode without changing the binding (AC-1:
// "Pressing Escape during capture cancels without changing the binding" —
// for the reserved pause key the Cancel path of the AC-5 dialog is the
// cancel; see HandleKey).
func (s *KeybindingsScreen) cancelCapture() {
	s.exitCapture()
}

// findConflict returns the action (other than target) whose bindings
// contain the chord, or "". Only the directly conflicting action is found —
// the cascade stops at one hop (AC-2).
func (s *KeybindingsScreen) findConflict(target ActionBinding, chord KeyChord) ActionBinding {
	for _, a := range s.actions {
		name := a.RegistryName()
		if name == target {
			continue
		}
		for _, c := range s.registry.BindingsFor(name) {
			if c == chord {
				return name
			}
		}
	}
	return ""
}

// chordAt returns the chord currently bound in the action's slot (the zero
// chord = unbound).
func (s *KeybindingsScreen) chordAt(action ActionBinding, slot BindingSlot) KeyChord {
	list := s.registry.BindingsFor(action)
	if int(slot) >= len(list) {
		return KeyChord{}
	}
	return list[slot]
}

// onReservedBind completes the binding after the AC-5 acknowledgement
// ("Bind Anyway"), subject to the AC-6 last-binding gate inside
// finishCapture.
func (s *KeybindingsScreen) onReservedBind() {
	if s.state != KeybindingsStateReservedWarn {
		return
	}
	chord := s.pendingChord
	s.finishCapture(chord)
}

// onReservedCancel is the AC-5 dialog's Cancel: binding unchanged, capture
// mode exits.
func (s *KeybindingsScreen) onReservedCancel() {
	if s.state != KeybindingsStateReservedWarn {
		return
	}
	s.cancelCapture()
}

// ---------------------------------------------------------------------------
// Conflict warnings (AC-2)
// ---------------------------------------------------------------------------

// IsWarned reports whether the action's row shows the "Unbound" conflict
// warning (AC-2: ⚠ icon + amber text — two cues, never hue alone).
func (s *KeybindingsScreen) IsWarned(action ActionBinding) bool { return s.warned[action] }

// WarnedActions returns the names of all currently warned actions in table
// order. Treat the slice as read-only.
func (s *KeybindingsScreen) WarnedActions() []ActionBinding {
	out := make([]ActionBinding, 0, len(s.warned))
	for _, a := range s.actions {
		name := a.RegistryName()
		if s.warned[name] {
			out = append(out, name)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Default restoration (AC-4)
// ---------------------------------------------------------------------------

// RestoreActionDefaults restores one action's default primary and secondary
// bindings (AC-4) and clears its conflict warning. No confirmation — the
// per-row restore is instant (AC-4 edge case). Other rows are untouched.
func (s *KeybindingsScreen) RestoreActionDefaults(action ActionBinding) {
	if s.state != KeybindingsStatePanel {
		return
	}
	s.restoreAction(action)
}

// restoreAction restores one action to its defaults without state guards
// (shared by the per-row restore and restore-all). A restored non-zero chord
// runs the same conflict resolution as finishCapture (AC-2): the action
// holding the chord is cleared from EVERY matching slot (a same-action
// double-bind is fully released), marked warned, and notified — the cascade
// stops at one hop. If the holder is a reserved function with only that
// binding, the AC-6 gate blocks the restore of that slot (blocked banner,
// slot keeps its current binding). It reports whether any slot changed, so
// restore-all can iterate to a fixpoint.
func (s *KeybindingsScreen) restoreAction(action ActionBinding) bool {
	changed := false
	def := s.actionDef(action)
	if def == nil {
		return false
	}
	defaults := [2]KeyChord{def.DefaultPrimary, def.DefaultSecondary}
	for _, slot := range []BindingSlot{SlotPrimary, SlotSecondary} {
		want := defaults[slot]
		if s.chordAt(action, slot) == want {
			continue
		}
		if want.Key != "" {
			if victim := s.findConflict(action, want); victim != "" {
				if s.registry.IsReserved(victim) && s.registry.BindingCount(victim) == 1 {
					// AC-6: the restore would strip the reserved function's
					// last binding — blocked, this slot keeps its binding.
					s.blockedMsg = (LastReservedBindingError{Action: victim}).Error()
					continue
				}
				for _, vslot := range []BindingSlot{SlotPrimary, SlotSecondary} {
					if s.chordAt(victim, vslot) != want {
						continue
					}
					_ = s.registry.ClearSlot(victim, int(vslot)) // cannot fail: checked above
					s.warned[victim] = true
					s.dirtyKey(victim, vslot, KeyChord{})
					if s.changeFn != nil {
						s.changeFn(string(victim), vslot, KeyChord{})
					}
				}
			}
		}
		s.registry.BindSlot(action, int(slot), want)
		s.dirtyKey(action, slot, want)
		if s.changeFn != nil {
			s.changeFn(string(action), slot, want)
		}
		changed = true
	}
	delete(s.warned, action)
	return changed
}

// onRestoreAll opens the AC-4 restore-all confirmation dialog.
func (s *KeybindingsScreen) onRestoreAll() {
	if s.state != KeybindingsStatePanel {
		return
	}
	s.setState(KeybindingsStateRestoreAll)
}

// onRestoreAllConfirm restores every action's default bindings and resets
// the gamepad preset to "Default" (AC-4), clearing all conflict warnings.
//
// The restore runs to a FIXPOINT: an AC-6-gated slot (its default held by a
// reserved function's last binding) cannot restore until that holder's own
// restore frees the key, which may happen later in the same pass or in a
// following pass. Passes repeat until one changes nothing — the example
// (move_forward=[Escape,zero], pause_menu=[zero,W]) converges on the third
// pass. The blocked banner is only reset once the fixpoint completes.
func (s *KeybindingsScreen) onRestoreAllConfirm() {
	if s.state != KeybindingsStateRestoreAll {
		return
	}
	for {
		changed := false
		for _, a := range s.actions {
			if s.restoreAction(a.RegistryName()) {
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	s.warned = map[ActionBinding]bool{}
	s.blockedMsg = "" // restore-all settles every warning; no stale AC-6 banner
	if idx := s.presetIndexByKey("Default"); idx >= 0 && s.presetIndex != idx {
		s.presetIndex = idx
		s.dirtyKeys[SettingGamepadPreset] = s.presets[idx].Key
		s.lastChangeMs = s.now()
		if s.presetFn != nil {
			s.presetFn(s.presets[idx].Key)
		}
	}
	s.setState(KeybindingsStatePanel)
}

// onRestoreAllCancel abandons the pending restore-all.
func (s *KeybindingsScreen) onRestoreAllCancel() {
	if s.state != KeybindingsStateRestoreAll {
		return
	}
	s.setState(KeybindingsStatePanel)
}

// ---------------------------------------------------------------------------
// Gamepad presets (AC-7)
// ---------------------------------------------------------------------------

// GamepadPresets returns the selectable preset table (AC-7). Treat as
// read-only.
func (s *KeybindingsScreen) GamepadPresets() []GamepadPreset { return s.presets }

// GamepadPresetIndex returns the index of the selected preset.
func (s *KeybindingsScreen) GamepadPresetIndex() int { return s.presetIndex }

// SelectedGamepadPreset returns the selected preset, or the zero preset
// when the table is empty.
func (s *KeybindingsScreen) SelectedGamepadPreset() GamepadPreset {
	if s.presetIndex < 0 || s.presetIndex >= len(s.presets) {
		return GamepadPreset{}
	}
	return s.presets[s.presetIndex]
}

// SelectGamepadPreset selects the preset at the given index (AC-7): the
// selection applies immediately (presetFn) and persists under
// controls.gamepad_preset. Returns false for out-of-range indices or while
// capturing.
func (s *KeybindingsScreen) SelectGamepadPreset(index int) bool {
	if index < 0 || index >= len(s.presets) {
		return false
	}
	if s.state != KeybindingsStatePanel {
		return false
	}
	s.presetIndex = index
	s.dirtyKeys[SettingGamepadPreset] = s.presets[index].Key
	s.lastChangeMs = s.now()
	if s.presetFn != nil {
		s.presetFn(s.presets[index].Key)
	}
	return true
}

// cycleGamepadPreset steps the preset dropdown one slot forward — the
// dropdown's activate handler (the settings dropdowns cycle the same way).
func (s *KeybindingsScreen) cycleGamepadPreset() {
	if len(s.presets) == 0 || s.state != KeybindingsStatePanel {
		return
	}
	s.SelectGamepadPreset((s.presetIndex + 1) % len(s.presets))
}

func (s *KeybindingsScreen) presetIndexByKey(key string) int {
	for i, p := range s.presets {
		if p.Key == key {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// Persistence (AC-8, Story 005 pattern)
// ---------------------------------------------------------------------------

// persistChordValue serializes a chord for settings.cfg (AC-8): the zero
// chord (unbound slot) persists as SettingUnboundValue so a cleared slot
// stays cleared across reboots — an empty value would be indistinguishable
// from a missing key on Load and resurrect the action's default.
func persistChordValue(c KeyChord) string {
	if c.Key == "" {
		return SettingUnboundValue
	}
	return SerializeChord(c)
}

// dirtyKey records a persisted-key change and arms the debounce flush
// (AC-8). The value is serialized at mark time; a later change to the same
// key overwrites it before the flush.
func (s *KeybindingsScreen) dirtyKey(action ActionBinding, slot BindingSlot, chord KeyChord) {
	s.dirtyKeys[s.bindingKey(action, slot)] = persistChordValue(chord)
	s.lastChangeMs = s.now()
}

// Dirty reports whether any unsaved binding change is pending.
func (s *KeybindingsScreen) Dirty() bool { return len(s.dirtyKeys) > 0 }

// DirtyKeys returns the pending keys in action-table order (deterministic
// for tests and for the adapter's schema building). Treat the slice as
// read-only.
func (s *KeybindingsScreen) DirtyKeys() []string {
	out := make([]string, 0, len(s.dirtyKeys))
	for _, a := range s.actions {
		name := a.RegistryName()
		for _, slot := range []BindingSlot{SlotPrimary, SlotSecondary} {
			key := s.bindingKey(name, slot)
			if _, ok := s.dirtyKeys[key]; ok {
				out = append(out, key)
			}
		}
	}
	if _, ok := s.dirtyKeys[SettingGamepadPreset]; ok {
		out = append(out, SettingGamepadPreset)
	}
	return out
}

// Update advances the clock-driven state (call once per frame from the game
// layer): the debounce flush fires when the dirty flag has been quiet for
// KeybindingsDebounceMs (AC-8).
func (s *KeybindingsScreen) Update() {
	if s.Dirty() && s.now()-s.lastChangeMs >= KeybindingsDebounceMs {
		s.flush()
	}
}

// Flush writes every dirty key immediately (AC-8: clean shutdown flushes
// mid-debounce with the latest values). Safe to call from a shutdown path;
// failures keep the keys dirty for the next retry.
func (s *KeybindingsScreen) Flush() {
	if s.Dirty() {
		s.flush()
	}
}

// flush writes each dirty key with its serialized value through the injected
// seam (Story 005's retry state machine: a failed write keeps the key dirty
// and logs; the next change retries).
func (s *KeybindingsScreen) flush() {
	failed := false
	for _, key := range s.DirtyKeys() {
		if s.writeFn == nil {
			delete(s.dirtyKeys, key)
			continue
		}
		if err := s.writeFn(key, s.dirtyKeys[key]); err != nil {
			failed = true
			s.logFn("keybindings: failed to write %s: %v", key, err)
			continue
		}
		delete(s.dirtyKeys, key)
	}
	if failed {
		s.writeFailures++
		return
	}
	s.writeFailures = 0
}

// WriteFailures returns the consecutive failed flush count (the retry state
// machine never permanently fails — a later change retries).
func (s *KeybindingsScreen) WriteFailures() int { return s.writeFailures }

// ---------------------------------------------------------------------------
// State / focus / rendering
// ---------------------------------------------------------------------------

// State returns the current inline UI state.
func (s *KeybindingsScreen) State() KeybindingsState { return s.state }

// CaptureAction returns the capture target action ("" when not capturing).
func (s *KeybindingsScreen) CaptureAction() ActionBinding { return s.captureAction }

// CaptureSlot returns the capture target slot.
func (s *KeybindingsScreen) CaptureSlot() BindingSlot { return s.captureSlot }

// ReservedWarning returns the AC-5 dialog text, or "" when the dialog is
// not open.
func (s *KeybindingsScreen) ReservedWarning() string {
	if s.state != KeybindingsStateReservedWarn {
		return ""
	}
	return ReservedWarningMessage(s.reservedFn)
}

// BlockedMessage returns the AC-6 block warning ("This is the only binding
// for Pause Menu. It cannot be removed."), or "" when none is shown. The
// banner clears on the next capture, a successful capture, restore-all, or
// screen re-entry.
func (s *KeybindingsScreen) BlockedMessage() string { return s.blockedMsg }

// CapturePulse returns the capture highlight pulse phase in [0, 1) driven
// by the injectable clock: the amber gold border "breathes" at
// CapturePulseMs. Purely cosmetic — never a state cue (two cues rule).
func (s *KeybindingsScreen) CapturePulse() float64 {
	if !s.CapturesInput() {
		return 0
	}
	return math.Mod(s.now(), CapturePulseMs) / CapturePulseMs
}

// Actions returns the action table in display order. Treat as read-only.
func (s *KeybindingsScreen) Actions() []BindingAction { return s.actions }

// Bindings returns the action's current bindings (primary, secondary; the
// zero chord = unbound). Treat the slice as read-only.
func (s *KeybindingsScreen) Bindings(action ActionBinding) []KeyChord {
	return s.registry.BindingsFor(action)
}

// setState switches the inline UI state and refocuses the first enabled
// element of the new state. The restore-all dialog opens on the Confirm
// button explicitly: FocusNext can advance past the first enabled dialog
// element onto Cancel, which would make Enter cancel instead of restore.
func (s *KeybindingsScreen) setState(st KeybindingsState) {
	s.state = st
	s.refreshEnabled()
	s.focus.FocusNext()
	if st == KeybindingsStateRestoreAll {
		_ = s.focus.FocusByID(FocusRestoreAllConfirm)
	}
}

// refreshEnabled re-applies the enabled state of every focusable element
// from the inline state: the table is interactive only in the panel state,
// the dialogs only when open.
func (s *KeybindingsScreen) refreshEnabled() {
	table := s.state == KeybindingsStatePanel
	for _, a := range s.actions {
		_ = s.focus.SetEnabled(fmt.Sprintf(FocusPrimaryIDFmt, bindingFocusID(a.ID)), table)
		_ = s.focus.SetEnabled(fmt.Sprintf(FocusSecondaryIDFmt, bindingFocusID(a.ID)), table)
		_ = s.focus.SetEnabled(fmt.Sprintf(FocusRestoreIDFmt, bindingFocusID(a.ID)), table)
	}
	_ = s.focus.SetEnabled(FocusGamepadPreset, table)
	_ = s.focus.SetEnabled(FocusRestoreAll, table)
	_ = s.focus.SetEnabled(FocusKeybindingsBack, table)

	reserved := s.state == KeybindingsStateReservedWarn
	_ = s.focus.SetEnabled(FocusReservedBind, reserved)
	_ = s.focus.SetEnabled(FocusReservedCancel, reserved)

	restoreAll := s.state == KeybindingsStateRestoreAll
	_ = s.focus.SetEnabled(FocusRestoreAllConfirm, restoreAll)
	_ = s.focus.SetEnabled(FocusRestoreAllCancel, restoreAll)
}

// onBack is the visible Back button (mouse parity with Escape/B): the flow
// owns the back destination — the entry point (the Settings screen).
func (s *KeybindingsScreen) onBack() {
	if s.state != KeybindingsStatePanel {
		return
	}
	if s.flow != nil {
		_ = s.flow.Back()
	}
}

// KeybindingsRenderCommandType is the kind of one render command.
type KeybindingsRenderCommandType int

const (
	// RenderBindingRow is one action row (label + primary slot + secondary
	// slot + restore button).
	RenderBindingRow KeybindingsRenderCommandType = iota
	// RenderGamepadPresetRow is the preset dropdown row (AC-7).
	RenderGamepadPresetRow
	// RenderRestoreAll is the footer restore-all button (AC-4).
	RenderRestoreAll
	// RenderKeybindingsBack is the footer back button.
	RenderKeybindingsBack
	// RenderReservedDialog is the AC-5 warning dialog.
	RenderReservedDialog
	// RenderRestoreAllDialog is the AC-4 confirmation dialog.
	RenderRestoreAllDialog
	// RenderBlockedBanner is the AC-6 block warning banner.
	RenderBlockedBanner
)

// String returns the command type name for diagnostics.
func (t KeybindingsRenderCommandType) String() string {
	switch t {
	case RenderBindingRow:
		return "BindingRow"
	case RenderGamepadPresetRow:
		return "GamepadPresetRow"
	case RenderRestoreAll:
		return "RestoreAll"
	case RenderKeybindingsBack:
		return "Back"
	case RenderReservedDialog:
		return "ReservedDialog"
	case RenderRestoreAllDialog:
		return "RestoreAllDialog"
	case RenderBlockedBanner:
		return "BlockedBanner"
	}
	return fmt.Sprintf("KeybindingsRenderCommandType(%d)", int(t))
}

// KeybindingsRenderCommand is one element of the per-frame render command
// stream the Kaiju pass consumes (same contract as the settings screen's
// RenderCommand). Commands append to a caller-provided buffer so a steady
// frame allocates nothing.
type KeybindingsRenderCommand struct {
	Type   KeybindingsRenderCommandType
	Action ActionBinding
	Label  string
	// Primary / Secondary are the slot display labels: the serialized key
	// name ("Ctrl+S"), "Unbound", or "Press a key..." during capture.
	Primary   string
	Secondary string
	// Warned marks an AC-2 conflict row: the render pass draws the ⚠ icon
	// plus amber text (two cues — never hue alone).
	Warned bool
	// Capturing marks the row whose slot is in capture mode (amber gold
	// pulsing border, CapturePulse phase).
	Capturing bool
	FocusID   string
	// Value carries the preset dropdown's selected label and the dialog
	// bodies.
	Value    string
	Selected bool
}

// RenderCommands appends the current frame's commands to dst and returns
// the extended slice: the category-grouped action rows, the preset row, the
// footer, and any active dialog or block banner. Allocation-free when dst
// has spare capacity (pass a per-frame scratch buffer).
func (s *KeybindingsScreen) RenderCommands(dst []KeybindingsRenderCommand) []KeybindingsRenderCommand {
	capturing := s.state == KeybindingsStateCapture
	for _, a := range s.actions {
		name := a.RegistryName()
		row := KeybindingsRenderCommand{
			Type:      RenderBindingRow,
			Action:    name,
			Label:     a.Label,
			Primary:   chordLabel(s.chordAt(name, SlotPrimary)),
			Secondary: chordLabel(s.chordAt(name, SlotSecondary)),
			Warned:    s.warned[name],
			FocusID:   fmt.Sprintf(FocusPrimaryIDFmt, bindingFocusID(a.ID)),
		}
		if capturing && s.captureAction == name {
			row.Capturing = true
			if s.captureSlot == SlotPrimary {
				row.Primary = LabelPressKey
			} else {
				row.Secondary = LabelPressKey
			}
		}
		dst = append(dst, row)
	}
	dst = append(dst, KeybindingsRenderCommand{
		Type: RenderGamepadPresetRow, Label: LabelGamepadPreset,
		Value: s.SelectedGamepadPreset().Label, FocusID: FocusGamepadPreset,
	})
	dst = append(dst, KeybindingsRenderCommand{Type: RenderRestoreAll, Label: LabelRestoreAllDefaults, FocusID: FocusRestoreAll})
	dst = append(dst, KeybindingsRenderCommand{Type: RenderKeybindingsBack, Label: LabelBack, FocusID: FocusKeybindingsBack})
	switch s.state {
	case KeybindingsStateReservedWarn:
		dst = append(dst, KeybindingsRenderCommand{Type: RenderReservedDialog, Label: s.ReservedWarning(), FocusID: FocusReservedBind})
	case KeybindingsStateRestoreAll:
		dst = append(dst, KeybindingsRenderCommand{Type: RenderRestoreAllDialog, Label: RestoreAllConfirmMessage, FocusID: FocusRestoreAllConfirm})
	}
	if s.blockedMsg != "" {
		dst = append(dst, KeybindingsRenderCommand{Type: RenderBlockedBanner, Label: s.blockedMsg})
	}
	return dst
}

// chordLabel renders a chord for display: the serialized key name
// ("Ctrl+S") or "Unbound" for the zero chord.
func chordLabel(c KeyChord) string {
	if c.Key == "" {
		return LabelUnbound
	}
	return SerializeChord(c)
}

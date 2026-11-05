// settings_screen.go implements the SettingsScreen controller (Story 005):
// the six-category tabbed settings screen (Gameplay, Camera, Audio, Controls,
// Accessibility, Graphics — GDD CR-3), instant-apply on every change, and
// debounced JSON persistence to settings.cfg.
//
// The controller is pure Go: it owns state, focus, and timing only. All
// timing derives from the injectable clock (the same function passed to
// NewTransitionSystem). Kaiju HTML/CSS rendering is intentionally absent —
// this file exposes the data model (settings registry, per-frame render
// commands, dialog/toast/preview state) that a Kaiju-aware rendering pass
// consumes.
//
// Persistence and system application are injected, never owned:
//
//   - ApplySetting(key, value) — instant-apply seam to the target system
//     (camera, audio, gameplay, renderer). Called synchronously on change.
//   - WriteSettings(key, value) error — the debounced flush seam. Keys are
//     fully-qualified dotted keys ("camera.edge_scroll_speed") with string
//     values ("20"); the game layer adapter maps them onto the nested JSON
//     schema ({"camera": {"edge_scroll_speed": 20}}) and owns the file I/O
//     (out of scope here). CR-7 keys persisted by Story 003 (camera_motion,
//     subtitles) keep their flat historical keys.
//   - HasSettings() / GetSetting(key) — the AC-5 boot probe and value
//     pre-population for an existing settings.cfg.
//
// Debounce is clock-driven, matching the package's goroutine-free core: the
// game layer calls Update() once per frame; a flush fires when the dirty
// flag has been set for >= DebounceMs without a new change, or immediately
// via Flush() on clean shutdown. Never synchronous disk I/O in the render
// loop — the render pass only reads; writes happen inside Update/Flush.
//
// Two settings are Preview-flagged (resolution, display mode): changing them
// applies immediately and starts the AC-3 keep/revert countdown. The overlay
// is non-blocking; Y keeps, N or Escape reverts, and no response within
// PreviewAutoRevertMs auto-reverts. The game layer routes raw Y/N/Escape
// keys to HandleKey (the flow's Dispatch translates no letter keys and Story
// 001's flow is out of scope); the overlay buttons provide the mouse/gamepad
// path.
//
// Deferred engine features (VSync, texture quality, AA, audio output device)
// render greyed with a "Coming Soon" tooltip and reject interaction (AC-7).
// The FPS cap is NOT deferred — host.SetFrameRateLimit exists.
//
// Entry point is fixed at OnEnter (AC-8): entered from the Pause Menu, the
// Graphics category hides resolution and display mode — absent, not greyed.

package mainmenu

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Timing and persistence tuning knobs (GDD main-menu-and-settings CR-3
// Formulas; data-driven, safe ranges from the Tuning Knobs table).
const (
	// DebounceMs is settings_write_debounce_ms: how long the flush waits
	// after the last change before writing settings.cfg (GDD: 500ms).
	DebounceMs = 500.0
	// PreviewCountdownMs is the display-mode preview countdown shown on the
	// keep/revert overlay (GDD CR-3: 3-second preview).
	PreviewCountdownMs = 3000.0
	// PreviewAutoRevertMs is the auto-revert timeout: no response within
	// 15 seconds restores the previous working settings (GDD CR-3).
	PreviewAutoRevertMs = 15000.0
	// SettingsToastMs is the write-failure toast display duration (AC-6:
	// 3-second auto-dismiss).
	SettingsToastMs = 3000.0
)

// Category is one of the six settings categories (GDD CR-3). Tabs are
// rendered in declaration order; the active tab shows an amber gold
// indicator plus bold weight (two cues — never color alone).
type Category int

const (
	// CategoryGameplay holds difficulty and helper toggles.
	CategoryGameplay Category = iota
	// CategoryCamera holds the Camera System GDD settings.
	CategoryCamera
	// CategoryAudio holds mixer volumes and output.
	CategoryAudio
	// CategoryControls holds sensitivity and the Story 007 keybinding rows.
	CategoryControls
	// CategoryAccessibility holds the CR-7 accessibility settings.
	CategoryAccessibility
	// CategoryGraphics holds resolution, display mode, FPS cap, and the
	// deferred engine features.
	CategoryGraphics
)

// String returns the category name for diagnostics.
func (c Category) String() string {
	switch c {
	case CategoryGameplay:
		return "Gameplay"
	case CategoryCamera:
		return "Camera"
	case CategoryAudio:
		return "Audio"
	case CategoryControls:
		return "Controls"
	case CategoryAccessibility:
		return "Accessibility"
	case CategoryGraphics:
		return "Graphics"
	}
	return fmt.Sprintf("Category(%d)", int(c))
}

// CategoryCount is the number of settings categories (AC-1).
const CategoryCount = 6

// ControlType is the rendered control kind of a setting (GDD CR-3
// implementation notes: sliders, dropdowns, toggles, keybinding rows).
type ControlType int

const (
	// ControlSlider is a numeric slider (volume, sensitivity, speed).
	ControlSlider ControlType = iota
	// ControlDropdown is a selection list (difficulty, resolution, mode).
	ControlDropdown
	// ControlToggle is an on/off switch (auto-pause, hints, VSync).
	ControlToggle
	// ControlKeybinding is a keybinding row (Story 007 placeholder type:
	// superseded by the dedicated Keybindings screen, retained for API
	// compatibility).
	ControlKeybinding
	// ControlNavigate is a row that opens a sub-screen (Story 007: the
	// Controls category's "Keybindings" row opens ScreenKeybindings).
	ControlNavigate
)

// String returns the control type name for diagnostics.
func (t ControlType) String() string {
	switch t {
	case ControlSlider:
		return "Slider"
	case ControlDropdown:
		return "Dropdown"
	case ControlToggle:
		return "Toggle"
	case ControlKeybinding:
		return "Keybinding"
	case ControlNavigate:
		return "Navigate"
	}
	return fmt.Sprintf("ControlType(%d)", int(t))
}

// DropdownOption is one choice of a dropdown setting: Key is the stable
// settings.cfg value ("fullscreen"), Label is the player-facing,
// LOCALIZATION-marked text ("Fullscreen").
type DropdownOption struct {
	Key   string
	Label string
}

// SettingValue is the typed value of a setting — a discriminated union.
// Exactly one field is meaningful per ControlType: Float for sliders, Index
// for dropdowns (index into Options), Bool for toggles, Chord for
// keybinding rows.
type SettingValue struct {
	Float float64
	Index int
	Bool  bool
	Chord KeyChord
}

// Setting is one row of the settings registry: the screen's data contract
// with the target systems and the rendering pass. Keys are fully-qualified
// dotted strings ("camera.edge_scroll_speed") — the settings.cfg schema key.
// The registry is built from the GDD CR-3 table; each system declares its
// own settings schema (TR-menu-002).
type Setting struct {
	// Key is the settings.cfg key ("camera.edge_scroll_speed").
	Key string
	// Label is the player-facing control label.
	//
	// LOCALIZATION: user-facing string. Exposed as data here because the
	// project has no string table yet; route through the localization
	// system when one exists (UI rule: no hardcoded strings in logic).
	Label string
	// Category is the owning tab.
	Category Category
	// Type selects the rendered control.
	Type ControlType
	// Min/Max/Step bound sliders (inclusive, Step > 0).
	Min, Max, Step float64
	// Options are the dropdown choices in display order.
	Options []DropdownOption
	// Value is the current value.
	Value SettingValue
	// Default is the reset value (AC-4) and the value on first launch
	// (AC-5).
	Default SettingValue
	// Deferred marks an engine feature not yet available (AC-7): the row
	// renders greyed with a "Coming Soon" tooltip and rejects interaction.
	Deferred bool
	// Preview marks a display-affecting setting that starts the keep/revert
	// countdown on change (AC-3: resolution, display mode).
	Preview bool
	// Target is the sub-screen a ControlNavigate row opens (Story 007: the
	// Controls category's "Keybindings" row opens ScreenKeybindings).
	Target Screen
}

// IsDefault reports whether the setting currently holds its default value.
func (st *Setting) IsDefault() bool {
	return st.Value == st.Default
}

// Visible reports whether the setting is shown in its category panel:
// deferred features stay visible (greyed, AC-7) but Graphics settings
// restricted in-game are absent, not greyed (AC-8).
func (st *Setting) Visible(restricted bool) bool {
	if !restricted {
		return true
	}
	switch st.Key {
	case SettingResolution, SettingDisplayMode:
		return false
	}
	return true
}

// FormatValue renders the current value for the value label and for the
// WriteSettings seam: sliders as plain numbers ("20", "0.8"), dropdowns as
// the option key ("fullscreen"), toggles as "on"/"off" (matching the Story
// 003 CR-7 contract), keybindings as the chord's key name.
func (st *Setting) FormatValue() string {
	switch st.Type {
	case ControlSlider:
		// Fixed 2-decimal formatting with trailing zeros trimmed keeps
		// settings.cfg human-readable and parse-stable: 0.8 + 0.05 steps
		// accumulate float error ("0.8500000000000001"), and every slider
		// step is >= 0.05, so 2 decimals never lose information ("0.8",
		// "0.85", "300").
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(st.Value.Float, 'f', 2, 64), "0"), ".")
	case ControlDropdown:
		if st.Value.Index >= 0 && st.Value.Index < len(st.Options) {
			return st.Options[st.Value.Index].Key
		}
		return ""
	case ControlToggle:
		if st.Value.Bool {
			return SubtitlesKeyOn
		}
		return SubtitlesKeyOff
	case ControlKeybinding:
		return st.Value.Chord.Key
	}
	return ""
}

// ValueLabel renders the player-facing value text for the rendering pass
// (slider numbers, option labels, On/Off, binding keys).
func (st *Setting) ValueLabel() string {
	switch st.Type {
	case ControlSlider:
		return st.FormatValue()
	case ControlDropdown:
		if st.Value.Index >= 0 && st.Value.Index < len(st.Options) {
			return st.Options[st.Value.Index].Label
		}
		return ""
	case ControlToggle:
		if st.Value.Bool {
			return LabelSubtitlesOn
		}
		return LabelSubtitlesOff
	case ControlKeybinding:
		return st.Value.Chord.Key
	}
	return ""
}

// ParseValue converts a settings.cfg string back onto the value. ok is false
// for unparsable or out-of-range input; callers keep their default (a
// partial or stale settings.cfg is valid).
func (st *Setting) ParseValue(s string) (SettingValue, bool) {
	switch st.Type {
	case ControlSlider:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return SettingValue{}, false
		}
		return SettingValue{Float: clampStep(v, st.Min, st.Max, st.Step)}, true
	case ControlDropdown:
		for i, o := range st.Options {
			if o.Key == s {
				return SettingValue{Index: i}, true
			}
		}
		return SettingValue{}, false
	case ControlToggle:
		switch s {
		case SubtitlesKeyOn:
			return SettingValue{Bool: true}, true
		case SubtitlesKeyOff:
			return SettingValue{Bool: false}, true
		}
		return SettingValue{}, false
	case ControlKeybinding:
		if s == "" {
			return SettingValue{}, false
		}
		return SettingValue{Chord: KeyChord{Key: s}}, true
	}
	return SettingValue{}, false
}

// Setting keys — the settings.cfg schema contract. Camera keys follow the
// Camera System GDD tuning knobs (pan_speed_edge, pan_speed_keyboard,
// pan_speed_gamepad, edge_scroll_deadzone, zoom_transition_duration);
// camera_motion and subtitles keep their flat CR-7 keys from Story 003.
const (
	// Gameplay
	SettingDifficulty              = "gameplay.difficulty"
	SettingAutoPauseOnConversation = "gameplay.auto_pause_on_conversation"
	SettingTooltipVerbosity        = "gameplay.tooltip_verbosity"
	SettingTutorialHints           = "gameplay.tutorial_hints"
	// Camera
	SettingEdgeScroll          = "camera.edge_scroll"
	SettingDragPanMode         = "camera.drag_pan_mode"
	SettingEdgeScrollDeadzone  = "camera.edge_scroll_deadzone"
	SettingEdgeScrollSpeed     = "camera.edge_scroll_speed"
	SettingKeyboardPanSpeed    = "camera.keyboard_pan_speed"
	SettingGamepadPanSpeed     = "camera.gamepad_pan_speed"
	SettingZoomTransitionSpeed = "camera.zoom_transition_duration"
	SettingInvertZoomDirection = "camera.invert_zoom_direction"
	// Audio
	SettingMasterVolume      = "audio.master_volume"
	SettingMusicVolume       = "audio.music_volume"
	SettingSFXVolume         = "audio.sfx_volume"
	SettingAmbientVolume     = "audio.ambient_volume"
	SettingDialogueVolume    = "audio.dialogue_volume"
	SettingMuteWhenUnfocused = "audio.mute_when_unfocused"
	SettingAudioOutputDevice = "audio.output_device"
	// Controls
	SettingMouseSensitivity     = "controls.mouse_sensitivity"
	SettingGamepadSensitivity   = "controls.gamepad_sensitivity"
	SettingInvertMouseWheelZoom = "controls.invert_mouse_wheel_zoom"
	// SettingKeybindings is the Controls category's "Keybindings" navigation
	// row (Story 007): it opens the Keybindings sub-screen.
	SettingKeybindings = "controls.keybindings"
	// Accessibility (camera_motion/subtitles reuse the flat CR-7 keys
	// declared in accessibility_prompt.go)
	SettingExtraCloseZoom            = "accessibility.extra_close_zoom"
	SettingFollowActiveUnit          = "accessibility.follow_active_unit"
	SettingOutlineThickness          = "accessibility.outline_thickness"
	SettingMarkerSize                = "accessibility.marker_size"
	SettingMarkerOpacity             = "accessibility.marker_opacity"
	SettingMarkerDistanceLabels      = "accessibility.marker_distance_labels"
	SettingOccluderTransparency      = "accessibility.occluder_transparency"
	SettingSubtitleSize              = "accessibility.subtitle_size"
	SettingSubtitleBackgroundOpacity = "accessibility.subtitle_background_opacity"
	SettingSpeakerName               = "accessibility.speaker_name"
	SettingColorblindMode            = "accessibility.colorblind_mode"
	SettingTextSize                  = "accessibility.text_size"
	// Graphics
	SettingResolution  = "graphics.resolution"
	SettingDisplayMode = "graphics.display_mode"
	SettingFPSCap      = "graphics.fps_cap"
	SettingVSync       = "graphics.vsync"
	SettingTextureQual = "graphics.texture_quality"
	SettingAntiAlias   = "graphics.anti_aliasing"
)

// Dropdown option keys (settings.cfg values).
const (
	OptionDifficultyNarrative = "narrative"
	OptionDifficultyNormal    = "normal"
	OptionDifficultyTactical  = "tactical"

	OptionVerbosityMinimal = "minimal"
	OptionVerbosityNormal  = "normal"
	OptionVerbosityVerbose = "verbose"

	OptionDragPanHold   = "hold"
	OptionDragPanToggle = "toggle"
	OptionDragPanOff    = "off"

	OptionDisplayFullscreen = "fullscreen"
	OptionDisplayBorderless = "borderless"
	OptionDisplayWindowed   = "windowed"

	OptionFPSCap30        = "30"
	OptionFPSCap60        = "60"
	OptionFPSCap120       = "120"
	OptionFPSCapUnlimited = "unlimited"

	OptionTextureLow    = "low"
	OptionTextureMedium = "medium"
	OptionTextureHigh   = "high"

	OptionAAOff  = "off"
	OptionAAFXAA = "fxaa"
	OptionAASMAA = "smaa"

	OptionColorblindOff          = "off"
	OptionColorblindProtanopia   = "protanopia"
	OptionColorblindDeuteranopia = "deuteranopia"
	OptionColorblindTritanopia   = "tritanopia"

	OptionTextSizeDefault    = "default"
	OptionTextSizeLarge      = "large"
	OptionTextSizeExtraLarge = "extra_large"

	OptionSubtitleSmall  = "small"
	OptionSubtitleMedium = "medium"
	OptionSubtitleLarge  = "large"

	OptionAudioDeviceDefault = "default"
	OptionAudioDeviceSystem  = "system"
	OptionAudioDeviceHeadset = "headset"
)

// User-facing strings.
//
// LOCALIZATION: user-facing strings. Exposed as data here because the
// project has no string table yet; route through the localization system
// when one exists (UI rule: no hardcoded strings in logic).
const (
	LabelCategoryGameplay      = "Gameplay"
	LabelCategoryCamera        = "Camera"
	LabelCategoryAudio         = "Audio"
	LabelCategoryControls      = "Controls"
	LabelCategoryAccessibility = "Accessibility"
	LabelCategoryGraphics      = "Graphics"

	LabelDifficultyNarrative = "Narrative"
	LabelDifficultyNormal    = "Normal"
	LabelDifficultyTactical  = "Tactical"

	LabelAutoPauseOnConversation = "Auto-pause on conversation"
	LabelTooltipVerbosity        = "Tooltip verbosity"
	LabelVerbosityMinimal        = "Minimal"
	LabelVerbosityNormal         = "Normal"
	LabelVerbosityVerbose        = "Verbose"
	LabelTutorialHints           = "Tutorial hints"

	LabelEdgeScroll          = "Edge Scroll"
	LabelDragPanMode         = "Drag Pan Mode"
	LabelDragPanHold         = "Hold"
	LabelDragPanToggle       = "Toggle"
	LabelDragPanOff          = "Off"
	LabelEdgeScrollDeadzone  = "Edge Scroll Dead Zone"
	LabelEdgeScrollSpeed     = "Edge Scroll Speed"
	LabelKeyboardPanSpeed    = "Keyboard Pan Speed"
	LabelGamepadPanSpeed     = "Gamepad Pan Speed"
	LabelZoomTransitionSpeed = "Zoom Transition Speed"
	LabelInvertZoomDirection = "Invert Zoom Direction"

	LabelMasterVolume      = "Master Volume"
	LabelMusicVolume       = "Music Volume"
	LabelSFXVolume         = "SFX Volume"
	LabelAmbientVolume     = "Ambient Volume"
	LabelDialogueVolume    = "Dialogue Volume"
	LabelMuteWhenUnfocused = "Mute when unfocused"
	LabelAudioOutputDevice = "Audio Output Device"

	LabelMouseSensitivity     = "Mouse Sensitivity"
	LabelGamepadSensitivity   = "Gamepad Sensitivity"
	LabelInvertMouseWheelZoom = "Invert Mouse Wheel Zoom"
	LabelKeybindings          = "Keybindings"

	LabelExtraCloseZoom         = "Extra Close Zoom"
	LabelFollowActiveUnit       = "Follow Active Unit"
	LabelOutlineThickness       = "Outline Thickness"
	LabelMarkerSize             = "Marker Size"
	LabelMarkerOpacity          = "Marker Opacity"
	LabelMarkerDistanceLabels   = "Marker Distance Labels"
	LabelOccluderTransparency   = "Occluder Transparency"
	LabelSubtitleSize           = "Subtitle Size"
	LabelSubtitleBackground     = "Subtitle Background Opacity"
	LabelSpeakerName            = "Speaker Name"
	LabelColorblindMode         = "Colorblind Mode"
	LabelTextSize               = "Text Size"
	LabelTextSizeDefault        = "Default"
	LabelTextSizeLarge          = "Large"
	LabelTextSizeExtraLarge     = "Extra Large"
	LabelSubtitleSmall          = "Small"
	LabelSubtitleMedium         = "Medium"
	LabelSubtitleLarge          = "Large"
	LabelColorblindOff          = "Off"
	LabelColorblindProtanopia   = "Protanopia"
	LabelColorblindDeuteranopia = "Deuteranopia"
	LabelColorblindTritanopia   = "Tritanopia"

	LabelResolution      = "Resolution"
	LabelDisplayMode     = "Display Mode"
	LabelFPSCap          = "FPS Cap"
	LabelVSync           = "VSync"
	LabelTextureQual     = "Texture Quality"
	LabelAntiAlias       = "Anti-Aliasing"
	LabelFPSCapUnlimited = "Unlimited"

	LabelResetCategory      = "Reset to Defaults"
	LabelResetAll           = "Reset All Settings"
	LabelKeepChanges        = "Keep Changes"
	LabelRevert             = "Revert"
	LabelReRunAccessibility = "Re-run Accessibility Setup"
	// LabelBack (visible Back button) is declared in save_load_screen.go and
	// reused here.

	// PreviewMessage is the AC-3 overlay prompt.
	PreviewMessage = "Keep changes? (Y) / Revert (N)"
	// SettingsSaveFailedMessage is the AC-6 write-failure toast.
	SettingsSaveFailedMessage = "Settings could not be saved. Check available disk space."
	// TooltipComingSoon is the AC-7 tooltip for deferred engine features.
	TooltipComingSoon = "Coming Soon"
)

// CategoryLabel returns the LOCALIZATION-marked tab label of a category.
func CategoryLabel(c Category) string {
	switch c {
	case CategoryGameplay:
		return LabelCategoryGameplay
	case CategoryCamera:
		return LabelCategoryCamera
	case CategoryAudio:
		return LabelCategoryAudio
	case CategoryControls:
		return LabelCategoryControls
	case CategoryAccessibility:
		return LabelCategoryAccessibility
	case CategoryGraphics:
		return LabelCategoryGraphics
	}
	return c.String()
}

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements. Setting rows use
// positional-per-key ids ("setting-" + key with . and _ replaced by -), so
// they are stable across tab switches; the "settings-" prefix keeps them
// distinct from the title screen's ids, registered in the same document
// while screens share the flow.
const (
	// FocusTabID formats the tab id for a category ("tab-gameplay").
	FocusTabID = "tab-%s"
	// FocusSettingIDPrefix prefixes the per-key setting row ids.
	FocusSettingIDPrefix = "setting-"
	// FocusResetCategory is the per-category "Reset to Defaults" button
	// (bottom of the active category page, AC-4).
	FocusResetCategory = "settings-reset-category"
	// FocusResetAll is the footer "Reset All Settings" button (AC-4).
	FocusResetAll = "settings-reset-all"
	// FocusSettingsBack is the visible Back button.
	FocusSettingsBack = "settings-back"
	// FocusResetConfirm is the reset confirmation dialog's confirm button.
	FocusResetConfirm = "settings-reset-confirm"
	// FocusResetCancel is the reset confirmation dialog's cancel button.
	FocusResetCancel = "settings-reset-cancel"
	// FocusPreviewKeep is the preview overlay's "Keep Changes" button.
	FocusPreviewKeep = "settings-preview-keep"
	// FocusPreviewRevert is the preview overlay's "Revert" button.
	FocusPreviewRevert = "settings-preview-revert"
	// FocusToastDismiss (toast dismiss button) is declared in
	// save_load_screen.go and reused here.
	// FocusReRunAccessibility is the Accessibility category's re-run button.
	FocusReRunAccessibility = "settings-rerun-accessibility"
)

// SettingsState is the inline UI state of the settings screen. Reset
// confirmations are inline dialogs — not ScreenFlow states (matching the
// title screen's inline dialog pattern). The preview overlay and the toast
// are orthogonal: they can be active in any state and never block the panel
// behind them (AC-3: "the Settings screen remains accessible behind it").
type SettingsState int

const (
	// SettingsStatePanel is the category panel.
	SettingsStatePanel SettingsState = iota
	// SettingsStateResetCategory is the per-category reset confirmation.
	SettingsStateResetCategory
	// SettingsStateResetAll is the reset-all confirmation.
	SettingsStateResetAll
)

// String returns the state name for diagnostics.
func (s SettingsState) String() string {
	switch s {
	case SettingsStatePanel:
		return "Panel"
	case SettingsStateResetCategory:
		return "ResetCategory"
	case SettingsStateResetAll:
		return "ResetAll"
	}
	return fmt.Sprintf("SettingsState(%d)", int(s))
}

// SettingsScreenOption configures a SettingsScreen at construction time.
type SettingsScreenOption func(*SettingsScreen)

// WithSettingsClock installs the game-time clock (milliseconds) that drives
// the debounce flush, the preview countdown/auto-revert, and the toast
// auto-dismiss. Pass the same function given to NewTransitionSystem so the
// screen and screen transitions share one time source.
func WithSettingsClock(now func() float64) SettingsScreenOption {
	return func(s *SettingsScreen) { s.now = now }
}

// WithApplySetting installs the instant-apply seam (AC-2): invoked with the
// setting's key and formatted value on every change, before the dirty flag
// is set. The target system reads the new value on its next frame/tick.
// Deferred settings and keybinding placeholders never reach it.
func WithApplySetting(fn func(key, value string)) SettingsScreenOption {
	return func(s *SettingsScreen) { s.applyFn = fn }
}

// WithSettingsWriter installs the persistence seam (AC-2/AC-6): invoked with
// each dirty key and its formatted value when the debounce flush fires (or
// on Flush). A non-nil error keeps the key dirty for the next retry; two
// consecutive failures show the AC-6 toast. The game layer adapter owns the
// settings.cfg file I/O (out of scope here). The Settings prefix
// distinguishes this seam from the accessibility prompt's write callback.
func WithSettingsWriter(fn func(key, value string) error) SettingsScreenOption {
	return func(s *SettingsScreen) { s.writeFn = fn }
}

// WithHasSettingsFile installs the settings.cfg existence probe (AC-5):
// true when a valid settings.cfg exists. CheckFirstLaunch uses it to decide
// between pre-population and the first-launch defaults + prompt path.
func WithHasSettingsFile(fn func() bool) SettingsScreenOption {
	return func(s *SettingsScreen) { s.hasSettingsFn = fn }
}

// WithSettingsReader installs the settings.cfg read callback (AC-5): returns
// the stored value for a key, or "" when absent (a partial settings.cfg is
// valid — missing or unparsable keys keep their defaults).
func WithSettingsReader(fn func(key string) string) SettingsScreenOption {
	return func(s *SettingsScreen) { s.getSettingFn = fn }
}

// WithSettingsLogger installs the error-log seam (AC-6): the pure core never
// touches a logging package; the engine-coupled game layer passes
// slog.Warn-style logging. Defaults to a no-op.
func WithSettingsLogger(fn func(format string, args ...any)) SettingsScreenOption {
	return func(s *SettingsScreen) { s.logFn = fn }
}

// WithReRunAccessibility installs the re-run hook for the Accessibility
// category's "Re-run Accessibility Setup" button (GDD CR-7): the game layer
// opens the Story 003 prompt in re-run mode.
func WithReRunAccessibility(fn func()) SettingsScreenOption {
	return func(s *SettingsScreen) { s.reRunFn = fn }
}

// SettingsScreen is the settings screen controller (Story 005). It
// implements ScreenController and registers for ScreenSettings. It
// remembers its entry point (TitleScreen vs PauseMenu, AC-8) for the back
// destination — which the flow owns — and for the Graphics restrictions.
//
// All probes and callbacks are injected (functional options), so the
// controller is fully headless-testable: no filesystem, no engine, no
// goroutines. Timing is a pure function of the injectable clock; the flush
// is driven from Update() (per-frame) and Flush() (clean shutdown).
type SettingsScreen struct {
	flow  *ScreenFlow
	focus *FocusManager
	now   func() float64

	applyFn       func(key, value string)
	writeFn       func(key, value string) error
	hasSettingsFn func() bool
	getSettingFn  func(key string) string
	logFn         func(format string, args ...any)
	reRunFn       func()

	state      SettingsState
	entryPoint Screen
	activeCat  Category
	settings   []*Setting

	// Persistence state (AC-2).
	dirtyKeys     map[string]bool
	lastChangeMs  float64
	writeFailures int

	// First-launch state (AC-5).
	firstLaunchPending bool

	// Preview state (AC-3).
	previewActive bool
	previewStart  float64
	previewPrev   map[string]SettingValue

	// Toast state (AC-6).
	toastMsg    string
	toastAt     float64
	toastActive bool
}

// defaultSettingsTable is the CR-3 settings registry template. Values are
// copied per screen instance so screens never share mutable state; Options
// slices are immutable (read-only contract).
var defaultSettingsTable = []Setting{
	// ── Gameplay ──
	{Key: SettingDifficulty, Label: "Difficulty", Category: CategoryGameplay, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionDifficultyNarrative, LabelDifficultyNarrative},
			{OptionDifficultyNormal, LabelDifficultyNormal},
			{OptionDifficultyTactical, LabelDifficultyTactical},
		}, Default: SettingValue{Index: 1}},
	{Key: SettingAutoPauseOnConversation, Label: LabelAutoPauseOnConversation, Category: CategoryGameplay, Type: ControlToggle,
		Default: SettingValue{Bool: true}},
	{Key: SettingTooltipVerbosity, Label: LabelTooltipVerbosity, Category: CategoryGameplay, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionVerbosityMinimal, LabelVerbosityMinimal},
			{OptionVerbosityNormal, LabelVerbosityNormal},
			{OptionVerbosityVerbose, LabelVerbosityVerbose},
		}, Default: SettingValue{Index: 1}},
	{Key: SettingTutorialHints, Label: LabelTutorialHints, Category: CategoryGameplay, Type: ControlToggle,
		Default: SettingValue{Bool: true}},

	// ── Camera (Camera System GDD tuning knobs) ──
	{Key: SettingEdgeScroll, Label: LabelEdgeScroll, Category: CategoryCamera, Type: ControlToggle,
		Default: SettingValue{Bool: true}},
	{Key: SettingDragPanMode, Label: LabelDragPanMode, Category: CategoryCamera, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionDragPanHold, LabelDragPanHold},
			{OptionDragPanToggle, LabelDragPanToggle},
			{OptionDragPanOff, LabelDragPanOff},
		}, Default: SettingValue{Index: 0}},
	{Key: SettingEdgeScrollDeadzone, Label: LabelEdgeScrollDeadzone, Category: CategoryCamera, Type: ControlSlider,
		Min: 4, Max: 64, Step: 1, Default: SettingValue{Float: 16}},
	{Key: SettingEdgeScrollSpeed, Label: LabelEdgeScrollSpeed, Category: CategoryCamera, Type: ControlSlider,
		Min: 5, Max: 30, Step: 1, Default: SettingValue{Float: 15}},
	{Key: SettingKeyboardPanSpeed, Label: LabelKeyboardPanSpeed, Category: CategoryCamera, Type: ControlSlider,
		Min: 10, Max: 40, Step: 1, Default: SettingValue{Float: 20}},
	{Key: SettingGamepadPanSpeed, Label: LabelGamepadPanSpeed, Category: CategoryCamera, Type: ControlSlider,
		Min: 5, Max: 30, Step: 1, Default: SettingValue{Float: 15}},
	{Key: SettingZoomTransitionSpeed, Label: LabelZoomTransitionSpeed, Category: CategoryCamera, Type: ControlSlider,
		Min: 150, Max: 600, Step: 10, Default: SettingValue{Float: 300}},
	{Key: SettingInvertZoomDirection, Label: LabelInvertZoomDirection, Category: CategoryCamera, Type: ControlToggle,
		Default: SettingValue{Bool: false}},

	// ── Audio ──
	{Key: SettingMasterVolume, Label: LabelMasterVolume, Category: CategoryAudio, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.8}},
	{Key: SettingMusicVolume, Label: LabelMusicVolume, Category: CategoryAudio, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.7}},
	{Key: SettingSFXVolume, Label: LabelSFXVolume, Category: CategoryAudio, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.8}},
	{Key: SettingAmbientVolume, Label: LabelAmbientVolume, Category: CategoryAudio, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.6}},
	{Key: SettingDialogueVolume, Label: LabelDialogueVolume, Category: CategoryAudio, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.8}},
	{Key: SettingMuteWhenUnfocused, Label: LabelMuteWhenUnfocused, Category: CategoryAudio, Type: ControlToggle,
		Default: SettingValue{Bool: false}},
	{Key: SettingAudioOutputDevice, Label: LabelAudioOutputDevice, Category: CategoryAudio, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionAudioDeviceDefault, "Default"},
			{OptionAudioDeviceSystem, "System"},
			{OptionAudioDeviceHeadset, "Headset"},
		}, Default: SettingValue{Index: 0}, Deferred: true},

	// ── Controls ──
	{Key: SettingMouseSensitivity, Label: LabelMouseSensitivity, Category: CategoryControls, Type: ControlSlider,
		Min: 0.1, Max: 2.0, Step: 0.05, Default: SettingValue{Float: 1.0}},
	{Key: SettingGamepadSensitivity, Label: LabelGamepadSensitivity, Category: CategoryControls, Type: ControlSlider,
		Min: 0.1, Max: 2.0, Step: 0.05, Default: SettingValue{Float: 1.0}},
	{Key: SettingInvertMouseWheelZoom, Label: LabelInvertMouseWheelZoom, Category: CategoryControls, Type: ControlToggle,
		Default: SettingValue{Bool: false}},
	// Keybindings (Story 007): the per-action keybinding interface lives on
	// the dedicated Keybindings sub-screen (ScreenKeybindings); this row is
	// the Controls category's entry point.
	{Key: SettingKeybindings, Label: LabelKeybindings, Category: CategoryControls, Type: ControlNavigate,
		Target: ScreenKeybindings},

	// ── Accessibility (CR-7) ──
	{Key: SettingCameraMotion, Label: LabelCameraMotion, Category: CategoryAccessibility, Type: ControlDropdown,
		Options: []DropdownOption{
			{CameraMotionKeyFull, LabelMotionFull},
			{CameraMotionKeyReduced, LabelMotionReduced},
			{CameraMotionKeyOff, LabelMotionOff},
		}, Default: SettingValue{Index: 1}},
	{Key: SettingExtraCloseZoom, Label: LabelExtraCloseZoom, Category: CategoryAccessibility, Type: ControlToggle,
		Default: SettingValue{Bool: false}},
	{Key: SettingFollowActiveUnit, Label: LabelFollowActiveUnit, Category: CategoryAccessibility, Type: ControlToggle,
		Default: SettingValue{Bool: true}},
	{Key: SettingOutlineThickness, Label: LabelOutlineThickness, Category: CategoryAccessibility, Type: ControlSlider,
		Min: 1, Max: 4, Step: 1, Default: SettingValue{Float: 2}},
	{Key: SettingMarkerSize, Label: LabelMarkerSize, Category: CategoryAccessibility, Type: ControlSlider,
		Min: 50, Max: 150, Step: 5, Default: SettingValue{Float: 100}},
	{Key: SettingMarkerOpacity, Label: LabelMarkerOpacity, Category: CategoryAccessibility, Type: ControlSlider,
		Min: 0.1, Max: 1.0, Step: 0.05, Default: SettingValue{Float: 0.8}},
	{Key: SettingMarkerDistanceLabels, Label: LabelMarkerDistanceLabels, Category: CategoryAccessibility, Type: ControlToggle,
		Default: SettingValue{Bool: true}},
	{Key: SettingOccluderTransparency, Label: LabelOccluderTransparency, Category: CategoryAccessibility, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.5}},
	{Key: SettingSubtitles, Label: LabelSubtitles, Category: CategoryAccessibility, Type: ControlDropdown,
		Options: []DropdownOption{
			{SubtitlesKeyOn, LabelSubtitlesOn},
			{SubtitlesKeyOff, LabelSubtitlesOff},
		}, Default: SettingValue{Index: 0}},
	{Key: SettingSubtitleSize, Label: LabelSubtitleSize, Category: CategoryAccessibility, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionSubtitleSmall, LabelSubtitleSmall},
			{OptionSubtitleMedium, LabelSubtitleMedium},
			{OptionSubtitleLarge, LabelSubtitleLarge},
		}, Default: SettingValue{Index: 1}},
	{Key: SettingSubtitleBackgroundOpacity, Label: LabelSubtitleBackground, Category: CategoryAccessibility, Type: ControlSlider,
		Min: 0, Max: 1, Step: 0.05, Default: SettingValue{Float: 0.7}},
	{Key: SettingSpeakerName, Label: LabelSpeakerName, Category: CategoryAccessibility, Type: ControlToggle,
		Default: SettingValue{Bool: true}},
	{Key: SettingColorblindMode, Label: LabelColorblindMode, Category: CategoryAccessibility, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionColorblindOff, LabelColorblindOff},
			{OptionColorblindProtanopia, LabelColorblindProtanopia},
			{OptionColorblindDeuteranopia, LabelColorblindDeuteranopia},
			{OptionColorblindTritanopia, LabelColorblindTritanopia},
		}, Default: SettingValue{Index: 0}},
	{Key: SettingTextSize, Label: LabelTextSize, Category: CategoryAccessibility, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionTextSizeDefault, LabelTextSizeDefault},
			{OptionTextSizeLarge, LabelTextSizeLarge},
			{OptionTextSizeExtraLarge, LabelTextSizeExtraLarge},
		}, Default: SettingValue{Index: 0}},

	// ── Graphics ──
	{Key: SettingResolution, Label: LabelResolution, Category: CategoryGraphics, Type: ControlDropdown,
		Options: []DropdownOption{
			{"1280x720", "1280x720"},
			{"1600x900", "1600x900"},
			{"1920x1080", "1920x1080"},
			{"2560x1440", "2560x1440"},
		}, Default: SettingValue{Index: 2}, Preview: true},
	{Key: SettingDisplayMode, Label: LabelDisplayMode, Category: CategoryGraphics, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionDisplayFullscreen, "Fullscreen"},
			{OptionDisplayBorderless, "Borderless Window"},
			{OptionDisplayWindowed, "Windowed"},
		}, Default: SettingValue{Index: 0}, Preview: true},
	{Key: SettingFPSCap, Label: LabelFPSCap, Category: CategoryGraphics, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionFPSCap30, "30 FPS"},
			{OptionFPSCap60, "60 FPS"},
			{OptionFPSCap120, "120 FPS"},
			{OptionFPSCapUnlimited, LabelFPSCapUnlimited},
		}, Default: SettingValue{Index: 1}},
	{Key: SettingVSync, Label: LabelVSync, Category: CategoryGraphics, Type: ControlToggle,
		Default: SettingValue{Bool: false}, Deferred: true},
	{Key: SettingTextureQual, Label: LabelTextureQual, Category: CategoryGraphics, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionTextureLow, "Low"},
			{OptionTextureMedium, "Medium"},
			{OptionTextureHigh, "High"},
		}, Default: SettingValue{Index: 2}, Deferred: true},
	{Key: SettingAntiAlias, Label: LabelAntiAlias, Category: CategoryGraphics, Type: ControlDropdown,
		Options: []DropdownOption{
			{OptionAAOff, "Off"},
			{OptionAAFXAA, "FXAA"},
			{OptionAASMAA, "SMAA"},
		}, Default: SettingValue{Index: 0}, Deferred: true},
}

// NewSettingsScreen creates the settings screen controller for the given
// flow. The flow is required for navigation (Back); the caller registers the
// screen with flow.RegisterController(ScreenSettings, screen). Options
// inject the clock, the apply/write seams, the AC-5 probes, the logger, and
// the re-run hook.
func NewSettingsScreen(flow *ScreenFlow, opts ...SettingsScreenOption) *SettingsScreen {
	s := &SettingsScreen{
		flow:        flow,
		focus:       NewFocusManager(NewFocusStyle(1.0)),
		now:         func() float64 { return 0 },
		logFn:       func(string, ...any) {},
		state:       SettingsStatePanel,
		entryPoint:  ScreenTitleScreen,
		activeCat:   CategoryGameplay,
		dirtyKeys:   map[string]bool{},
		previewPrev: map[string]SettingValue{},
		settings:    make([]*Setting, 0, len(defaultSettingsTable)),
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	if s.now == nil {
		s.now = func() float64 { return 0 }
	}
	for i := range defaultSettingsTable {
		st := defaultSettingsTable[i]
		st.Value = st.Default
		s.settings = append(s.settings, &st)
	}
	s.registerFocusables()
	s.refreshEnabled()
	return s
}

// settingFocusID builds the stable focusable id of a setting row from its
// schema key ("camera.edge_scroll_speed" -> "setting-camera-edge-scroll-speed").
func settingFocusID(key string) string {
	return FocusSettingIDPrefix + strings.NewReplacer(".", "-", "_", "-").Replace(key)
}

// registerFocusables registers every focusable element in Tab order: the six
// tabs, every setting row of every category, the per-category reset button,
// the footer (reset all, back), the dialog controls, the preview overlay
// buttons, the re-run button, and the toast dismiss. Elements are registered
// once and enabled/disabled per state, so the Tab order is stable across tab
// switches and inline state changes.
func (s *SettingsScreen) registerFocusables() {
	for c := CategoryGameplay; c < CategoryCount; c++ {
		cat := c
		_ = s.focus.Register(fmt.Sprintf(FocusTabID, strings.ToLower(cat.String())),
			WithGroup("tabs"), WithBackground(ColorParchment),
			WithActivate(func() { s.SetActiveCategory(cat) }))
	}
	for i := range s.settings {
		st := s.settings[i]
		_ = s.focus.Register(settingFocusID(st.Key),
			WithGroup("panel"), WithBackground(ColorParchment),
			WithActivate(func() { s.cycleSetting(st.Key) }),
			WithScroll(func(dir int) { s.stepSlider(st.Key, dir) }))
	}
	_ = s.focus.Register(FocusReRunAccessibility, WithGroup("panel"), WithBackground(ColorParchment),
		WithActivate(func() {
			if s.reRunFn != nil {
				s.reRunFn()
			}
		}))
	_ = s.focus.Register(FocusResetCategory, WithGroup("panel"), WithBackground(ColorParchment), WithActivate(s.onResetCategory))
	_ = s.focus.Register(FocusResetAll, WithGroup("footer"), WithBackground(ColorParchment), WithActivate(s.onResetAll))
	_ = s.focus.Register(FocusSettingsBack, WithGroup("footer"), WithBackground(ColorParchment), WithActivate(s.onBack))
	_ = s.focus.Register(FocusResetConfirm, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onResetConfirm))
	_ = s.focus.Register(FocusResetCancel, WithGroup("dialog"), WithBackground(ColorParchment), WithActivate(s.onResetCancel))
	_ = s.focus.Register(FocusPreviewKeep, WithGroup("preview"), WithBackground(ColorParchment), WithActivate(s.KeepPreviewChanges))
	_ = s.focus.Register(FocusPreviewRevert, WithGroup("preview"), WithBackground(ColorParchment), WithActivate(s.RevertPreviewChanges))
	_ = s.focus.Register(FocusToastDismiss, WithBackground(ColorParchment), WithActivate(s.DismissToast))
}

// CheckFirstLaunch runs the AC-5 boot probe: with a valid settings.cfg the
// screen pre-populates its values from the store; without one (missing or
// corrupted — the probe returns false), the screen keeps all defaults in
// memory, marks every key dirty so the next flush creates the file, and
// reports that the first-launch accessibility prompt (Story 003) must be
// shown. No error is ever displayed to the player. Call once at boot, before
// the title screen becomes interactive.
func (s *SettingsScreen) CheckFirstLaunch() {
	if s.hasSettingsFn != nil && s.hasSettingsFn() {
		s.firstLaunchPending = false
		s.populateFromStore()
		return
	}
	s.firstLaunchPending = true
	for _, st := range s.settings {
		s.dirtyKeys[st.Key] = true
	}
	s.lastChangeMs = s.now()
}

// FirstLaunchPending reports whether CheckFirstLaunch found no valid
// settings.cfg (AC-5): the game layer shows the Story 003 prompt while true.
func (s *SettingsScreen) FirstLaunchPending() bool { return s.firstLaunchPending }

// populateFromStore mirrors an existing settings.cfg into the registry
// (AC-5): each key is read through the injected GetSetting seam; missing or
// unparsable values keep the defaults (a partial settings.cfg is valid). No
// ApplySetting calls — the target systems already booted with these values;
// the screen only mirrors them for display.
func (s *SettingsScreen) populateFromStore() {
	if s.getSettingFn == nil {
		return
	}
	for _, st := range s.settings {
		if v, ok := st.ParseValue(s.getSettingFn(st.Key)); ok {
			st.Value = v
		}
	}
}

// OnEnter fixes the entry point (AC-8) and resets to the panel state on the
// first tab (Gameplay). The entry point never changes mid-session — it is
// set at screen open time. The first tab is focused.
func (s *SettingsScreen) OnEnter(prev Screen) {
	s.entryPoint = prev
	s.state = SettingsStatePanel
	s.activeCat = CategoryGameplay
	s.refreshEnabled()
	s.focus.FocusNext()
}

// OnExit clears focus and the toast. The preview overlay intentionally
// survives: it is non-blocking (AC-3) and its auto-revert timer keeps
// running on the game clock while the player navigates elsewhere.
func (s *SettingsScreen) OnExit() {
	s.focus.ClearFocus()
	s.toastActive = false
	s.toastMsg = ""
}

// Focus returns the screen's focus manager for input routing.
func (s *SettingsScreen) Focus() *FocusManager { return s.focus }

// EntryPoint returns the screen the settings screen was entered from
// (ScreenTitleScreen or ScreenPauseMenu, AC-8). The flow owns the back
// destination; the screen reads the entry point for Graphics restrictions.
func (s *SettingsScreen) EntryPoint() Screen { return s.entryPoint }

// GraphicsRestricted reports whether the Graphics category must hide its
// display-affecting controls (AC-8): true when entered from the Pause Menu.
func (s *SettingsScreen) GraphicsRestricted() bool { return s.entryPoint == ScreenPauseMenu }

// State returns the current inline UI state.
func (s *SettingsScreen) State() SettingsState { return s.state }

// ActiveCategory returns the active tab's category.
func (s *SettingsScreen) ActiveCategory() Category { return s.activeCat }

// SetActiveCategory switches the active tab and focuses the first enabled
// panel control. Switching shows/hides panels — no page reload (AC-9 edge
// case: tab switching must not spike frame time).
func (s *SettingsScreen) SetActiveCategory(c Category) {
	if c < CategoryGameplay || c >= CategoryCount {
		return
	}
	s.activeCat = c
	s.refreshEnabled()
	// Focus the first interactive row of the new panel.
	if id := s.firstPanelFocusID(); id != "" {
		s.focus.FocusByID(id)
	} else {
		s.focus.FocusNext()
	}
}

// firstPanelFocusID returns the focusable id of the first enabled control in
// the active panel, or "" when none.
func (s *SettingsScreen) firstPanelFocusID() string {
	for _, st := range s.settings {
		if st.Category != s.activeCat || !s.settingInteractive(st) {
			continue
		}
		return settingFocusID(st.Key)
	}
	return ""
}

// Settings returns the full registry. Treat the slice and its Settings as
// read-only.
func (s *SettingsScreen) Settings() []*Setting { return s.settings }

// Setting returns the registry entry for a key, or nil.
func (s *SettingsScreen) Setting(key string) *Setting {
	for _, st := range s.settings {
		if st.Key == key {
			return st
		}
	}
	return nil
}

// VisibleSettings returns the settings shown in a category's panel: all of
// the category's rows except the ones hidden by the in-game Graphics
// restriction (AC-8: resolution and display mode are absent, not greyed).
// Deferred rows stay visible (greyed, AC-7). The slice is freshly built per
// call — treat it as read-only.
func (s *SettingsScreen) VisibleSettings(c Category) []*Setting {
	out := make([]*Setting, 0, 8)
	for _, st := range s.settings {
		if st.Category == c && st.Visible(s.GraphicsRestricted()) {
			out = append(out, st)
		}
	}
	return out
}

// SettingFocusID returns the focusable id of a setting row.
func (s *SettingsScreen) SettingFocusID(key string) string { return settingFocusID(key) }

// Value returns the current value of a setting.
func (s *SettingsScreen) Value(key string) (SettingValue, bool) {
	st := s.Setting(key)
	if st == nil {
		return SettingValue{}, false
	}
	return st.Value, true
}

// ValueText returns the persisted-form value of a setting ("20",
// "fullscreen", "on") — the same string the WriteSettings seam receives.
func (s *SettingsScreen) ValueText(key string) string {
	if st := s.Setting(key); st != nil {
		return st.FormatValue()
	}
	return ""
}

// ValueLabel returns the player-facing value label of a setting.
func (s *SettingsScreen) ValueLabel(key string) string {
	if st := s.Setting(key); st != nil {
		return st.ValueLabel()
	}
	return ""
}

// SetSlider sets a slider setting to v (clamped and stepped), applies it
// instantly (AC-2), and marks it dirty. Returns false for unknown keys,
// non-slider settings, and deferred settings.
func (s *SettingsScreen) SetSlider(key string, v float64) bool {
	st := s.Setting(key)
	if st == nil || st.Type != ControlSlider || st.Deferred {
		return false
	}
	return s.changeSetting(st, SettingValue{Float: clampStep(v, st.Min, st.Max, st.Step)})
}

// SetToggle sets a toggle setting (AC-2). Returns false for unknown keys,
// non-toggle settings, and deferred settings.
func (s *SettingsScreen) SetToggle(key string, on bool) bool {
	st := s.Setting(key)
	if st == nil || st.Type != ControlToggle || st.Deferred {
		return false
	}
	return s.changeSetting(st, SettingValue{Bool: on})
}

// SelectOption selects a dropdown setting by option index (AC-2). Returns
// false for unknown keys, non-dropdown settings, deferred settings, and
// out-of-range indices.
func (s *SettingsScreen) SelectOption(key string, index int) bool {
	st := s.Setting(key)
	if st == nil || st.Type != ControlDropdown || st.Deferred {
		return false
	}
	if index < 0 || index >= len(st.Options) {
		return false
	}
	return s.changeSetting(st, SettingValue{Index: index})
}

// SelectOptionKey selects a dropdown setting by option key ("fullscreen").
func (s *SettingsScreen) SelectOptionKey(key, optionKey string) bool {
	st := s.Setting(key)
	if st == nil || st.Type != ControlDropdown || st.Deferred {
		return false
	}
	for i, o := range st.Options {
		if o.Key == optionKey {
			return s.changeSetting(st, SettingValue{Index: i})
		}
	}
	return false
}

// cycleSetting steps the setting one slot forward: dropdowns cycle their
// options, toggles flip, sliders step up by Step, keybindings are inert
// placeholders (Story 007). This is the activate handler of every row
// (Enter/Space/gamepad A/click).
func (s *SettingsScreen) cycleSetting(key string) {
	st := s.Setting(key)
	if st == nil || st.Deferred || st.Type == ControlKeybinding {
		return
	}
	switch st.Type {
	case ControlDropdown:
		next := (st.Value.Index + 1) % len(st.Options)
		s.changeSetting(st, SettingValue{Index: next})
	case ControlToggle:
		s.changeSetting(st, SettingValue{Bool: !st.Value.Bool})
	case ControlSlider:
		s.changeSetting(st, SettingValue{Float: clampStep(st.Value.Float+st.Step, st.Min, st.Max, st.Step)})
	case ControlNavigate:
		// Story 007: open the sub-screen (Keybindings). The flow owns the
		// transition; Back returns here via the back stack.
		if s.flow != nil {
			_ = s.flow.RequestTransition(st.Target)
		}
	}
}

// stepSlider nudges a slider by one step in the given wheel direction
// (+1 up, -1 down). This is the scroll handler of every row.
func (s *SettingsScreen) stepSlider(key string, dir int) {
	st := s.Setting(key)
	if st == nil || st.Type != ControlSlider || st.Deferred {
		return
	}
	delta := st.Step
	if dir < 0 {
		delta = -delta
	}
	s.changeSetting(st, SettingValue{Float: clampStep(st.Value.Float+delta, st.Min, st.Max, st.Step)})
}

// changeSetting is the single mutation path (AC-2): update the in-memory
// value, apply instantly through the injected seam, mark the key dirty, and
// arm the debounce. Preview-flagged settings start the keep/revert countdown
// (AC-3). A change never touches the filesystem.
func (s *SettingsScreen) changeSetting(st *Setting, v SettingValue) bool {
	if st == nil {
		return false
	}
	// Snapshot the pre-change preview values BEFORE mutating: Revert must
	// restore what the player had before this preview began. The snapshot is
	// taken once per preview session; re-arming (a second preview change
	// while active) keeps the ORIGINAL snapshot.
	if st.Preview && !s.previewActive {
		s.previewPrev = map[string]SettingValue{}
		for _, ps := range s.settings {
			if ps.Preview {
				s.previewPrev[ps.Key] = ps.Value
			}
		}
	}
	st.Value = v
	if s.applyFn != nil {
		s.applyFn(st.Key, st.FormatValue())
	}
	s.dirtyKeys[st.Key] = true
	s.lastChangeMs = s.now()
	if st.Preview {
		s.previewActive = true
		s.previewStart = s.now()
		s.refreshEnabled()
	}
	return true
}

// ---------------------------------------------------------------------------
// Persistence (AC-2/AC-6)
// ---------------------------------------------------------------------------

// Dirty reports whether any unsaved change is pending.
func (s *SettingsScreen) Dirty() bool { return len(s.dirtyKeys) > 0 }

// DirtyKeys returns the pending keys in registry order (deterministic for
// tests and for the adapter's schema building). Treat the slice as read-only.
func (s *SettingsScreen) DirtyKeys() []string {
	out := make([]string, 0, len(s.dirtyKeys))
	for _, st := range s.settings {
		if s.dirtyKeys[st.Key] {
			out = append(out, st.Key)
		}
	}
	return out
}

// Update advances the clock-driven state (call once per frame from the game
// layer): the debounce flush fires when the dirty flag has been quiet for
// DebounceMs (AC-2), and an unanswered preview auto-reverts after
// PreviewAutoRevertMs (AC-3). Also re-applies focusable enabled states for
// the clock-driven toast expiry. No disk I/O happens anywhere else.
func (s *SettingsScreen) Update() {
	if s.Dirty() && s.now()-s.lastChangeMs >= DebounceMs {
		s.flush()
	}
	if s.previewActive && s.now()-s.previewStart >= PreviewAutoRevertMs {
		s.RevertPreviewChanges()
	}
	s.refreshEnabled()
}

// Flush writes every dirty key immediately (AC-2: clean shutdown flushes
// mid-debounce with the latest values). Safe to call from a shutdown path;
// failures follow the AC-6 retry state machine.
func (s *SettingsScreen) Flush() {
	if s.Dirty() {
		s.flush()
	}
}

// flush writes each dirty key with its current formatted value through the
// injected seam. On success the key leaves the dirty set and the error state
// clears silently. On failure the key stays dirty, the error is logged, and
// the next change retries; two consecutive failures raise the non-blocking
// AC-6 toast (3-second auto-dismiss).
func (s *SettingsScreen) flush() {
	failed := false
	for _, key := range s.DirtyKeys() {
		st := s.Setting(key)
		if st == nil {
			delete(s.dirtyKeys, key)
			continue
		}
		if s.writeFn == nil {
			delete(s.dirtyKeys, key)
			continue
		}
		if err := s.writeFn(key, st.FormatValue()); err != nil {
			failed = true
			s.logFn("settings: failed to write %s: %v", key, err)
			continue
		}
		delete(s.dirtyKeys, key)
	}
	if failed {
		s.writeFailures++
		if s.writeFailures >= 2 {
			s.showToast(SettingsSaveFailedMessage)
		}
		return
	}
	s.writeFailures = 0
	s.DismissToast()
}

// WriteFailures returns the consecutive failed flush count (AC-6 edge case:
// a 3rd change after 2 failures retries again — never permanently failed).
func (s *SettingsScreen) WriteFailures() int { return s.writeFailures }

// ---------------------------------------------------------------------------
// Resolution preview (AC-3)
// ---------------------------------------------------------------------------

// PreviewActive reports whether the keep/revert overlay is showing.
func (s *SettingsScreen) PreviewActive() bool { return s.previewActive }

// PreviewCountdownRemaining returns the overlay countdown in milliseconds
// (3-second preview per GDD CR-3): PreviewCountdownMs minus the elapsed game
// time since the change, clamped at 0.
func (s *SettingsScreen) PreviewCountdownRemaining() float64 {
	if !s.previewActive {
		return 0
	}
	remaining := PreviewCountdownMs - (s.now() - s.previewStart)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PreviewAutoRevertRemaining returns the time until the 15-second auto-revert
// in milliseconds, clamped at 0.
func (s *SettingsScreen) PreviewAutoRevertRemaining() float64 {
	if !s.previewActive {
		return 0
	}
	remaining := PreviewAutoRevertMs - (s.now() - s.previewStart)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PreviewMessage returns the overlay prompt (AC-3).
func (s *SettingsScreen) PreviewMessage() string { return PreviewMessage }

// KeepPreviewChanges makes the previewed values permanent (Y): the values
// are already applied and dirty; the overlay just closes. Idempotent.
func (s *SettingsScreen) KeepPreviewChanges() {
	if !s.previewActive {
		return
	}
	s.endPreview()
}

// RevertPreviewChanges restores the previous working values (N, Escape, or
// the 15-second timeout): each preview-flagged setting is changed back
// through the normal path, so the revert is applied instantly and persisted
// by the next flush (AC-3).
func (s *SettingsScreen) RevertPreviewChanges() {
	if !s.previewActive {
		return
	}
	prev := s.previewPrev
	for _, st := range s.settings {
		if !st.Preview {
			continue
		}
		if v, ok := prev[st.Key]; ok && v != st.Value {
			s.changeSetting(st, v)
		}
	}
	s.endPreview()
}

// endPreview closes the overlay and re-enables the panel controls.
func (s *SettingsScreen) endPreview() {
	s.previewActive = false
	s.previewPrev = map[string]SettingValue{}
	s.refreshEnabled()
}

// HandleKey routes raw keyboard keys that the flow's Dispatch does not
// translate (Story 001 out of scope): during the preview overlay, "y"/"Y"
// keeps the changes and "n"/"N" (or Escape) reverts them (AC-3). Returns
// true when the key was consumed. The game layer polls the keyboard and
// calls this while PreviewActive; the overlay buttons cover mouse/gamepad.
func (s *SettingsScreen) HandleKey(name string) bool {
	if !s.previewActive {
		return false
	}
	switch name {
	case "y", "Y":
		s.KeepPreviewChanges()
		return true
	case "n", "N", "Escape":
		s.RevertPreviewChanges()
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Reset to defaults (AC-4)
// ---------------------------------------------------------------------------

// onResetCategory opens the per-category reset confirmation. Confirmed, only
// the active category is reset; every other category is untouched.
func (s *SettingsScreen) onResetCategory() {
	if s.state != SettingsStatePanel {
		return
	}
	s.setState(SettingsStateResetCategory)
}

// onResetAll opens the reset-all confirmation.
func (s *SettingsScreen) onResetAll() {
	if s.state != SettingsStatePanel {
		return
	}
	s.setState(SettingsStateResetAll)
}

// onResetConfirm executes the pending reset (AC-4): the active category only
// for the per-category dialog, every category for reset-all. Each reset
// setting is applied instantly and marked dirty; any active display preview
// is cancelled (the reset values are what the player chose).
func (s *SettingsScreen) onResetConfirm() {
	switch s.state {
	case SettingsStateResetCategory:
		s.resetCategory(s.activeCat)
	case SettingsStateResetAll:
		for c := CategoryGameplay; c < CategoryCount; c++ {
			s.resetCategory(c)
		}
	default:
		return
	}
	s.cancelPreview()
	s.setState(SettingsStatePanel)
}

// onResetCancel abandons the pending reset.
func (s *SettingsScreen) onResetCancel() {
	if s.state != SettingsStateResetCategory && s.state != SettingsStateResetAll {
		return
	}
	s.setState(SettingsStatePanel)
}

// resetCategory restores every setting of one category to its default,
// applying each change instantly (AC-4). Other categories are untouched.
func (s *SettingsScreen) resetCategory(c Category) {
	for _, st := range s.settings {
		if st.Category != c || st.Type == ControlKeybinding || st.Type == ControlNavigate {
			continue
		}
		if st.Value != st.Default {
			s.changeSetting(st, st.Default)
		}
	}
}

// cancelPreview ends an active preview without reverting (the current values
// — e.g. freshly reset defaults — stay applied).
func (s *SettingsScreen) cancelPreview() {
	s.previewActive = false
	s.previewPrev = map[string]SettingValue{}
	s.refreshEnabled()
}

// ResetCategoryConfirmMessage returns the per-category confirmation body
// ("Reset Camera to default settings?"), or "" when no dialog is open.
func (s *SettingsScreen) ResetCategoryConfirmMessage() string {
	if s.state != SettingsStateResetCategory {
		return ""
	}
	return fmt.Sprintf("Reset %s to default settings?", CategoryLabel(s.activeCat))
}

// ResetAllConfirmMessage returns the reset-all confirmation body, or "" when
// no reset-all dialog is open.
func (s *SettingsScreen) ResetAllConfirmMessage() string {
	if s.state != SettingsStateResetAll {
		return ""
	}
	return "Reset all settings to defaults?"
}

// ---------------------------------------------------------------------------
// Toast (AC-6)
// ---------------------------------------------------------------------------

// ToastMessage returns the current toast text, or "" when no toast is shown.
func (s *SettingsScreen) ToastMessage() string { return s.toastMsg }

// ToastVisible reports whether the write-failure toast is displayed
// (non-blocking, auto-dismisses after SettingsToastMs via the clock).
func (s *SettingsScreen) ToastVisible() bool {
	if !s.toastActive {
		return false
	}
	return s.toastRemaining() > 0
}

// ToastRemaining returns the toast's remaining display time in milliseconds.
func (s *SettingsScreen) ToastRemaining() float64 {
	if !s.toastActive {
		return 0
	}
	return s.toastRemaining()
}

func (s *SettingsScreen) toastRemaining() float64 {
	remaining := SettingsToastMs - (s.now() - s.toastAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// DismissToast hides the toast immediately (also called silently when a
// later flush succeeds — "successful write clears error state silently").
func (s *SettingsScreen) DismissToast() {
	if !s.toastActive {
		return
	}
	s.toastActive = false
	s.toastMsg = ""
	s.refreshEnabled()
}

func (s *SettingsScreen) showToast(msg string) {
	s.toastMsg = msg
	s.toastAt = s.now()
	s.toastActive = true
	s.refreshEnabled()
}

// ---------------------------------------------------------------------------
// Focus / state plumbing
// ---------------------------------------------------------------------------

// settingInteractive reports whether a row can take focus and react to
// input: visible in the active panel, not deferred (AC-7), not a keybinding
// placeholder (Story 007), and the screen is in the panel state.
func (s *SettingsScreen) settingInteractive(st *Setting) bool {
	if s.state != SettingsStatePanel {
		return false
	}
	if st.Category != s.activeCat || st.Deferred || st.Type == ControlKeybinding {
		return false
	}
	return st.Visible(s.GraphicsRestricted())
}

// refreshEnabled re-applies the enabled state of every focusable element
// from the state, the active category, the preview overlay, and the toast.
func (s *SettingsScreen) refreshEnabled() {
	panel := s.state == SettingsStatePanel
	for _, st := range s.settings {
		_ = s.focus.SetEnabled(settingFocusID(st.Key), s.settingInteractive(st))
	}
	for c := CategoryGameplay; c < CategoryCount; c++ {
		_ = s.focus.SetEnabled(fmt.Sprintf(FocusTabID, strings.ToLower(c.String())), panel)
	}
	_ = s.focus.SetEnabled(FocusReRunAccessibility, panel && s.activeCat == CategoryAccessibility)
	_ = s.focus.SetEnabled(FocusResetCategory, panel)
	_ = s.focus.SetEnabled(FocusResetAll, panel)
	_ = s.focus.SetEnabled(FocusSettingsBack, panel)

	dialog := s.state == SettingsStateResetCategory || s.state == SettingsStateResetAll
	_ = s.focus.SetEnabled(FocusResetConfirm, dialog)
	_ = s.focus.SetEnabled(FocusResetCancel, dialog)

	_ = s.focus.SetEnabled(FocusPreviewKeep, s.previewActive)
	_ = s.focus.SetEnabled(FocusPreviewRevert, s.previewActive)
	_ = s.focus.SetEnabled(FocusToastDismiss, s.toastActive)
}

// setState switches the inline UI state and refocuses the first enabled
// element of the new state.
func (s *SettingsScreen) setState(st SettingsState) {
	s.state = st
	s.refreshEnabled()
	s.focus.FocusNext()
}

// onBack is the visible Back button (mouse parity with Escape/B): the flow
// owns the back destination — the entry point (AC-8).
func (s *SettingsScreen) onBack() {
	if s.state != SettingsStatePanel {
		return
	}
	if s.flow != nil {
		_ = s.flow.Back()
	}
}

// ---------------------------------------------------------------------------
// Render-command generation (AC-9)
// ---------------------------------------------------------------------------

// RenderCommandType is the kind of one render command.
type RenderCommandType int

const (
	// RenderTab is a category tab row.
	RenderTab RenderCommandType = iota
	// RenderSettingRow is one setting control row.
	RenderSettingRow
	// RenderReRunAccessibility is the Accessibility category's re-run button.
	RenderReRunAccessibility
	// RenderResetCategory is the per-category reset button.
	RenderResetCategory
	// RenderFooterResetAll is the footer reset-all button.
	RenderFooterResetAll
	// RenderBack is the footer back button.
	RenderBack
	// RenderResetDialog is a reset confirmation dialog.
	RenderResetDialog
	// RenderPreviewOverlay is the keep/revert overlay.
	RenderPreviewOverlay
	// RenderToast is the write-failure toast.
	RenderToast
)

// RenderCommand is one element of the per-frame render command stream the
// Kaiju pass consumes (AC-9: UI update + render-command generation ≤ 2ms
// p95). Commands append to a caller-provided buffer so a steady frame
// allocates nothing.
type RenderCommand struct {
	Type     RenderCommandType
	Category Category
	Setting  *Setting
	FocusID  string
	Label    string
	Value    string
	Tooltip  string
	Selected bool
	Dimmed   bool
}

// RenderCommands appends the current frame's commands to dst and returns the
// extended slice: the six tabs, the active panel's rows, the re-run and
// reset buttons, the footer, and any active dialog/preview/toast overlay.
// Allocation-free when dst has spare capacity (pass a per-frame scratch
// buffer).
func (s *SettingsScreen) RenderCommands(dst []RenderCommand) []RenderCommand {
	// Tabs (AC-1): the active tab carries both the amber gold indicator and
	// the bold weight (two cues).
	for c := CategoryGameplay; c < CategoryCount; c++ {
		dst = append(dst, RenderCommand{
			Type:     RenderTab,
			Category: c,
			FocusID:  fmt.Sprintf(FocusTabID, strings.ToLower(c.String())),
			Label:    CategoryLabel(c),
			Selected: c == s.activeCat && s.state == SettingsStatePanel,
		})
	}
	if s.state == SettingsStatePanel {
		for _, st := range s.VisibleSettings(s.activeCat) {
			row := RenderCommand{
				Type:    RenderSettingRow,
				Setting: st,
				FocusID: settingFocusID(st.Key),
				Label:   st.Label,
				Value:   st.ValueLabel(),
				Dimmed:  st.Deferred,
			}
			if st.Deferred {
				row.Tooltip = TooltipComingSoon
			}
			dst = append(dst, row)
		}
		if s.activeCat == CategoryAccessibility {
			dst = append(dst, RenderCommand{Type: RenderReRunAccessibility, FocusID: FocusReRunAccessibility, Label: LabelReRunAccessibility})
		}
		dst = append(dst, RenderCommand{Type: RenderResetCategory, FocusID: FocusResetCategory, Label: LabelResetCategory})
		dst = append(dst, RenderCommand{Type: RenderFooterResetAll, FocusID: FocusResetAll, Label: LabelResetAll})
		dst = append(dst, RenderCommand{Type: RenderBack, FocusID: FocusSettingsBack, Label: LabelBack})
	}
	switch s.state {
	case SettingsStateResetCategory:
		dst = append(dst, RenderCommand{Type: RenderResetDialog, FocusID: FocusResetConfirm, Label: s.ResetCategoryConfirmMessage()})
	case SettingsStateResetAll:
		dst = append(dst, RenderCommand{Type: RenderResetDialog, FocusID: FocusResetConfirm, Label: s.ResetAllConfirmMessage()})
	}
	if s.previewActive {
		dst = append(dst, RenderCommand{Type: RenderPreviewOverlay, FocusID: FocusPreviewKeep, Label: PreviewMessage})
	}
	if s.ToastVisible() {
		dst = append(dst, RenderCommand{Type: RenderToast, FocusID: FocusToastDismiss, Label: s.toastMsg})
	}
	return dst
}

// clampStep clamps v into [min, max] and snaps it to the step grid
// (min + k*step). Used for slider values so stored values are always exact
// multiples of the step (0.8 with step 0.05 stays 0.8, not 0.8000000000000001).
func clampStep(v, min, max, step float64) float64 {
	if step <= 0 {
		if v < min {
			return min
		}
		if v > max {
			return max
		}
		return v
	}
	steps := math.Round((v - min) / step)
	v = min + steps*step
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

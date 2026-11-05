// accessibility_prompt.go implements the AccessibilityPrompt controller
// (Story 003): the first-launch accessibility modal that overlays the title
// screen. It offers Reduce Camera Motion (Full / Reduced / Off) and
// Subtitles (On / Off) dropdowns, a "Remember my choice" checkbox, and a
// confirm button ("Continue" on first launch, "Apply" on re-run). Escape is
// equivalent to the confirm button: the currently-selected values apply and
// the modal dismisses (GDD CR-7). The modal does NOT auto-dismiss on a
// timer or click-away.
//
// The controller is pure Go: it owns state, focus, and timing only. All
// timing derives from the injectable clock (the same function passed to
// NewTransitionSystem), so slide-in assertions are exact and
// tolerance-free at the unit level. Kaiju HTML/CSS rendering (document
// creation, element styling) is intentionally absent — this file exposes
// the data model (values, options, label, visibility) and per-frame getters
// (AppearProgress) that a Kaiju-aware rendering pass consumes.
//
// settings.cfg persistence is injected, not owned: WriteSettings writes a
// single key, HasSettings reports whether a valid settings.cfg exists
// (AC-3/AC-6), and GetSetting reads a key for re-run pre-population
// (AC-4). Story 005 implements the real file; tests inject stubs.

package mainmenu

import "fmt"

// ModalSlideInMs is the modal slide-in duration (paper-slide motion per
// ADR-0012 / control manifest: 200-300ms ease-out; the reveal uses the
// lower bound, matching MenuElementFadeMs).
const ModalSlideInMs = 200.0

// Setting keys persisted to settings.cfg (Story 005 owns the file schema;
// these keys are the CR-7 contract between the prompt and the settings
// system).
const (
	// SettingCameraMotion is the settings.cfg key for Reduce Camera Motion.
	SettingCameraMotion = "camera_motion"
	// SettingSubtitles is the settings.cfg key for Subtitles.
	SettingSubtitles = "subtitles"
)

// Camera motion setting values (settings.cfg string keys, CR-7).
const (
	CameraMotionKeyFull    = "full"
	CameraMotionKeyReduced = "reduced"
	CameraMotionKeyOff     = "off"
)

// Subtitles setting values (settings.cfg string keys, CR-7).
const (
	SubtitlesKeyOn  = "on"
	SubtitlesKeyOff = "off"
)

// User-facing strings.
//
// LOCALIZATION: user-facing strings. Exposed as data here because the
// project has no string table yet; route through the localization system
// when one exists (UI rule: no hardcoded strings in logic).
const (
	// LabelCameraMotion is the Reduce Camera Motion control label (CR-7).
	LabelCameraMotion = "Reduce Camera Motion"
	// LabelSubtitles is the Subtitles control label (CR-7).
	LabelSubtitles = "Subtitles"
	// LabelRememberChoice is the "Remember my choice" checkbox label
	// (CR-7).
	LabelRememberChoice = "Remember my choice"
	// LabelContinue is the first-launch confirm button label (CR-7).
	LabelContinue = "Continue"
	// LabelApply is the re-run confirm button label (AC-4).
	LabelApply = "Apply"
	// Dropdown value labels.
	LabelMotionFull    = "Full"
	LabelMotionReduced = "Reduced"
	LabelMotionOff     = "Off"
	LabelSubtitlesOn   = "On"
	LabelSubtitlesOff  = "Off"
)

// Focusable element ids. Registration order IS the Tab order; the Kaiju
// rendering pass binds these ids to markup elements. The "accessibility-"
// prefix keeps them distinct from the title screen's ids, which are
// registered in the same document while the modal overlays the title.
const (
	FocusCameraMotion = "accessibility-camera-motion"
	FocusSubtitles    = "accessibility-subtitles"
	FocusRemember     = "accessibility-remember-choice"
	FocusContinue     = "accessibility-continue"
)

// CameraMotion is the Reduce Camera Motion setting value (CR-7). String is
// the player-facing label; Key is the settings.cfg value key.
type CameraMotion int

const (
	// CameraMotionFull is full camera motion (no reduction).
	CameraMotionFull CameraMotion = iota
	// CameraMotionReduced is the default: reduced camera motion.
	CameraMotionReduced
	// CameraMotionOff disables camera motion entirely.
	CameraMotionOff
)

// String returns the player-facing dropdown label.
func (m CameraMotion) String() string {
	switch m {
	case CameraMotionFull:
		return LabelMotionFull
	case CameraMotionReduced:
		return LabelMotionReduced
	case CameraMotionOff:
		return LabelMotionOff
	}
	return fmt.Sprintf("CameraMotion(%d)", int(m))
}

// Key returns the settings.cfg value key ("full", "reduced", "off").
func (m CameraMotion) Key() string {
	switch m {
	case CameraMotionFull:
		return CameraMotionKeyFull
	case CameraMotionReduced:
		return CameraMotionKeyReduced
	case CameraMotionOff:
		return CameraMotionKeyOff
	}
	return ""
}

// cameraMotionOrder is the dropdown cycle order (Full → Reduced → Off).
var cameraMotionOrder = []CameraMotion{CameraMotionFull, CameraMotionReduced, CameraMotionOff}

// CameraMotionOptions returns the selectable values in cycle order. The
// rendering pass enumerates the dropdown from this.
func CameraMotionOptions() []CameraMotion { return cameraMotionOrder }

// ParseCameraMotion converts a settings.cfg key to the enum value; ok is
// false for unknown keys (callers keep their default — a partial
// settings.cfg is valid, AC-3 edge case).
func ParseCameraMotion(key string) (CameraMotion, bool) {
	switch key {
	case CameraMotionKeyFull:
		return CameraMotionFull, true
	case CameraMotionKeyReduced:
		return CameraMotionReduced, true
	case CameraMotionKeyOff:
		return CameraMotionOff, true
	}
	return CameraMotionReduced, false
}

// Subtitles is the Subtitles setting value (CR-7). String is the
// player-facing label; Key is the settings.cfg value key.
type Subtitles int

const (
	// SubtitlesOn is the default: subtitles shown.
	SubtitlesOn Subtitles = iota
	// SubtitlesOff hides subtitles.
	SubtitlesOff
)

// String returns the player-facing dropdown label.
func (s Subtitles) String() string {
	switch s {
	case SubtitlesOn:
		return LabelSubtitlesOn
	case SubtitlesOff:
		return LabelSubtitlesOff
	}
	return fmt.Sprintf("Subtitles(%d)", int(s))
}

// Key returns the settings.cfg value key ("on", "off").
func (s Subtitles) Key() string {
	switch s {
	case SubtitlesOn:
		return SubtitlesKeyOn
	case SubtitlesOff:
		return SubtitlesKeyOff
	}
	return ""
}

// subtitlesOrder is the dropdown cycle order (On → Off).
var subtitlesOrder = []Subtitles{SubtitlesOn, SubtitlesOff}

// SubtitlesOptions returns the selectable values in cycle order. The
// rendering pass enumerates the dropdown from this.
func SubtitlesOptions() []Subtitles { return subtitlesOrder }

// ParseSubtitles converts a settings.cfg key to the enum value; ok is
// false for unknown keys (callers keep their default).
func ParseSubtitles(key string) (Subtitles, bool) {
	switch key {
	case SubtitlesKeyOn:
		return SubtitlesOn, true
	case SubtitlesKeyOff:
		return SubtitlesOff, true
	}
	return SubtitlesOn, false
}

// AccessibilityPromptOption configures an AccessibilityPrompt at
// construction time.
type AccessibilityPromptOption func(*AccessibilityPrompt)

// WithPromptClock installs the game-time clock (milliseconds) that drives
// the modal slide-in. Pass the same function given to NewTransitionSystem
// so the prompt and screen transitions share one time source.
func WithPromptClock(now func() float64) AccessibilityPromptOption {
	return func(p *AccessibilityPrompt) { p.now = now }
}

// WithHasSettings installs the settings.cfg existence probe (AC-3/AC-6):
// true when a valid settings.cfg exists. On the first-launch path the
// prompt does not appear when it returns true. Without it the prompt always
// appears on first launch.
func WithHasSettings(fn func() bool) AccessibilityPromptOption {
	return func(p *AccessibilityPrompt) { p.hasSettingsFn = fn }
}

// WithGetSetting installs the settings.cfg read callback used by re-run
// pre-population (AC-4). It returns the value for a key, or "" when the
// key is absent (a partial settings.cfg is valid).
func WithGetSetting(fn func(key string) string) AccessibilityPromptOption {
	return func(p *AccessibilityPrompt) { p.getSettingFn = fn }
}

// WithWriteSettings installs the settings.cfg write callback (AC-2). On
// dismissal with "Remember my choice" checked, the prompt writes
// SettingCameraMotion and SettingSubtitles through it.
func WithWriteSettings(fn func(key, value string)) AccessibilityPromptOption {
	return func(p *AccessibilityPrompt) { p.writeSettingsFn = fn }
}

// WithOnDismiss installs the dismissal notification callback (AC-2),
// invoked after the modal dismisses and the title screen is interactive
// again. The owner uses it to advance the boot flow or re-route input.
func WithOnDismiss(fn func()) AccessibilityPromptOption {
	return func(p *AccessibilityPrompt) { p.onDismissFn = fn }
}

// AccessibilityPrompt is the first-launch accessibility modal controller
// (Story 003). It implements ScreenController but is NOT a flow screen: it
// overlays the title screen (AC-1), and the owner (game layer) drives it —
// call OnEnter after the title background fade-in completes (or SetReRun
// then OnEnter from Settings → Accessibility), route translated input to
// Dispatch while Visible() is true, and call OnExit when the overlay is
// removed.
//
// Dismissal (Continue, Escape, or gamepad B) applies the current values
// for the session, persists them to settings.cfg when "Remember my choice"
// is checked, re-enables the title screen menu via the SetInteractive hook
// (Story 002), and notifies the owner. There is no timer and no click-away
// dismissal (AC-2).
type AccessibilityPrompt struct {
	title *TitleScreen
	focus *FocusManager
	now   func() float64

	hasSettingsFn   func() bool
	getSettingFn    func(key string) string
	writeSettingsFn func(key, value string)
	onDismissFn     func()

	cameraMotion   CameraMotion
	subtitles      Subtitles
	rememberChoice bool
	reRun          bool

	visible   bool
	enteredAt float64
}

// NewAccessibilityPrompt creates the accessibility prompt overlay for the
// given title screen. The title reference is required for the
// SetInteractive hook (Story 002): OnEnter disables the title menu buttons,
// dismissal re-enables them. Options inject the clock and the settings.cfg
// probes/callbacks.
//
// Defaults (CR-7): Reduce Camera Motion = Reduced, Subtitles = On,
// "Remember my choice" checked, confirm button labeled "Continue".
func NewAccessibilityPrompt(title *TitleScreen, opts ...AccessibilityPromptOption) *AccessibilityPrompt {
	p := &AccessibilityPrompt{
		title:          title,
		focus:          NewFocusManager(NewFocusStyle(1.0)),
		now:            func() float64 { return 0 },
		cameraMotion:   CameraMotionReduced,
		subtitles:      SubtitlesOn,
		rememberChoice: true,
	}
	for _, o := range opts {
		if o != nil {
			o(p)
		}
	}
	if p.now == nil {
		p.now = func() float64 { return 0 }
	}
	p.registerFocusables()
	return p
}

// registerFocusables registers the modal's controls in Tab order (AC-1):
// camera motion dropdown, subtitles dropdown, remember checkbox, confirm
// button. All share the implicit top-level group, so arrows/D-pad wrap
// across the whole modal.
func (p *AccessibilityPrompt) registerFocusables() {
	_ = p.focus.Register(FocusCameraMotion, WithBackground(ColorParchment), WithActivate(func() { p.cycleCameraMotion(true) }))
	_ = p.focus.Register(FocusSubtitles, WithBackground(ColorParchment), WithActivate(func() { p.cycleSubtitles(true) }))
	_ = p.focus.Register(FocusRemember, WithBackground(ColorParchment), WithActivate(func() { p.rememberChoice = !p.rememberChoice }))
	_ = p.focus.Register(FocusContinue, WithBackground(ColorParchment), WithActivate(p.dismiss))
}

// OnEnter shows the modal (AC-1): the title menu is visible behind but
// non-interactive (SetInteractive(false)) until dismissal. On the
// first-launch path with a valid settings.cfg present the prompt never
// appears (AC-3); the re-run path (AC-4) always shows, so the probe only
// gates the first-launch path. The modal starts focused on the camera
// motion dropdown.
func (p *AccessibilityPrompt) OnEnter(prev Screen) {
	if p.visible {
		return
	}
	if !p.reRun && p.hasSettingsFn != nil && p.hasSettingsFn() {
		return
	}
	p.visible = true
	p.enteredAt = p.now()
	if p.title != nil {
		p.title.SetInteractive(false)
	}
	p.focus.FocusNext()
}

// OnExit hides the modal and restores title screen interactivity (AC-2).
// Idempotent: a second call is a no-op. The owner may call it either after
// a dismissal or to remove the overlay without dismissal.
func (p *AccessibilityPrompt) OnExit() {
	if !p.visible {
		return
	}
	p.visible = false
	p.focus.ClearFocus()
	if p.title != nil {
		p.title.SetInteractive(true)
	}
}

// Focus returns the modal's own focus manager for input routing.
func (p *AccessibilityPrompt) Focus() *FocusManager { return p.focus }

// Dispatch routes a translated input to the modal (the owner calls this
// while Visible() instead of the flow). Escape and gamepad B dismiss with
// the current values applied (AC-2); keyboard arrows cycle the focused
// dropdown's value (CR-7: "arrow keys change dropdown/radio values");
// Tab/Shift+Tab and gamepad D-pad move focus; Enter/Space/A/click activate.
// Click-away and unmapped input are ignored — the modal never
// auto-dismisses.
func (p *AccessibilityPrompt) Dispatch(in Input) bool {
	if !p.visible {
		return false
	}
	if in.Action == ActionBack {
		p.dismiss()
		return true
	}
	p.focus.SetInputMethod(inputMethodFor(in.Source))
	if in.Source == SourceKeyboard {
		switch in.Action {
		case ActionMoveUp, ActionMoveDown, ActionMoveLeft, ActionMoveRight:
			switch p.focus.FocusedID() {
			case FocusCameraMotion:
				p.cycleCameraMotion(in.Action == ActionMoveRight || in.Action == ActionMoveDown)
				return true
			case FocusSubtitles:
				p.cycleSubtitles(in.Action == ActionMoveRight || in.Action == ActionMoveDown)
				return true
			}
		}
	}
	return dispatchFocusAction(p.focus, in)
}

// Visible reports whether the modal is currently displayed.
func (p *AccessibilityPrompt) Visible() bool { return p.visible }

// SetReRun switches the prompt to re-run mode (AC-4): the confirm button is
// labeled "Apply" instead of "Continue", and the controls pre-populate from
// the injected GetSetting probe. Unknown or missing keys keep the current
// values (a partial settings.cfg is valid). Call before OnEnter; re-run
// mode always shows the modal, bypassing the AC-3 suppression.
func (p *AccessibilityPrompt) SetReRun(on bool) {
	p.reRun = on
	if !on || p.getSettingFn == nil {
		return
	}
	if v, ok := ParseCameraMotion(p.getSettingFn(SettingCameraMotion)); ok {
		p.cameraMotion = v
	}
	if v, ok := ParseSubtitles(p.getSettingFn(SettingSubtitles)); ok {
		p.subtitles = v
	}
}

// CameraMotion returns the current Reduce Camera Motion selection.
func (p *AccessibilityPrompt) CameraMotion() CameraMotion { return p.cameraMotion }

// Subtitles returns the current Subtitles selection.
func (p *AccessibilityPrompt) Subtitles() Subtitles { return p.subtitles }

// RememberChoice reports whether "Remember my choice" is checked (default:
// checked).
func (p *AccessibilityPrompt) RememberChoice() bool { return p.rememberChoice }

// ConfirmLabel returns the confirm button label: "Apply" in re-run mode
// (AC-4), "Continue" on first launch.
func (p *AccessibilityPrompt) ConfirmLabel() string {
	if p.reRun {
		return LabelApply
	}
	return LabelContinue
}

// AppearProgress returns the modal slide-in progress in [0,1]:
// clamp((now - enteredAt) / ModalSlideInMs). The rendering pass animates
// the panel from this (paper-slide motion, ADR-0012). Reports 0 before the
// modal is shown and after it hides.
func (p *AccessibilityPrompt) AppearProgress() float64 {
	if !p.visible {
		return 0
	}
	e := p.now() - p.enteredAt
	if e <= 0 {
		return 0
	}
	if e >= ModalSlideInMs {
		return 1
	}
	return e / ModalSlideInMs
}

// cycleCameraMotion steps the camera motion selection one slot forward
// (true) or backward, wrapping in cycle order (Full → Reduced → Off).
func (p *AccessibilityPrompt) cycleCameraMotion(forward bool) {
	idx := indexOfCameraMotion(p.cameraMotion)
	if idx < 0 {
		idx = 0
	}
	if forward {
		idx = (idx + 1) % len(cameraMotionOrder)
	} else {
		idx = (idx + len(cameraMotionOrder) - 1) % len(cameraMotionOrder)
	}
	p.cameraMotion = cameraMotionOrder[idx]
}

// cycleSubtitles steps the subtitles selection one slot forward (true) or
// backward, wrapping in cycle order (On → Off).
func (p *AccessibilityPrompt) cycleSubtitles(forward bool) {
	idx := indexOfSubtitles(p.subtitles)
	if idx < 0 {
		idx = 0
	}
	if forward {
		idx = (idx + 1) % len(subtitlesOrder)
	} else {
		idx = (idx + len(subtitlesOrder) - 1) % len(subtitlesOrder)
	}
	p.subtitles = subtitlesOrder[idx]
}

func indexOfCameraMotion(m CameraMotion) int {
	for i, v := range cameraMotionOrder {
		if v == m {
			return i
		}
	}
	return -1
}

func indexOfSubtitles(s Subtitles) int {
	for i, v := range subtitlesOrder {
		if v == s {
			return i
		}
	}
	return -1
}

// dismiss applies the current values and closes the modal (AC-2). With
// "Remember my choice" checked, the two settings are persisted through the
// injected WriteSettings callback (camera_motion first, then subtitles);
// unchecked, they apply for the session only — the getters stay current,
// nothing is written, and the prompt reappears on the next launch. The
// title screen menu becomes interactive and the owner's dismiss callback
// fires. Idempotent: dismissing an already-closed modal is a no-op.
func (p *AccessibilityPrompt) dismiss() {
	if !p.visible {
		return
	}
	if p.rememberChoice && p.writeSettingsFn != nil {
		p.writeSettingsFn(SettingCameraMotion, p.cameraMotion.Key())
		p.writeSettingsFn(SettingSubtitles, p.subtitles.Key())
	}
	p.OnExit()
	if p.onDismissFn != nil {
		p.onDismissFn()
	}
}

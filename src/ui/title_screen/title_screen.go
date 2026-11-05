package title_screen

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/platform/filesystem"
)

// ScreenState describes which arrangement of the title screen is active.
type ScreenState int

const (
	StateDefault     ScreenState = iota // returning player, menu interactive
	StateFirstLaunch                     // accessibility modal showing
	StateNoSaves                         // Load Game greyed out
	StateConfirmNew                      // new-game confirmation dialog
	StateLoading                         // hard cut to soot, input locked
	StateError                           // error toast showing
)

// screenName identifies a screen navigation target.
type screenName string

const (
	ScreenLoadGame screenName = "load_game"
	ScreenSettings screenName = "settings"
)

// GameVersion is the version string shown in the menu footer.
const GameVersion = "0.1.0"

// settingsPath is the first-launch settings file. The title screen is its only
// writer; the game layer reads it for camera-motion and subtitle defaults.
const settingsPath = "settings/title_screen.json"

// titleSettings is the persisted first-launch configuration. ReducedMotion is
// derived from CameraMotion at write time and consumed by TweenRunner.
type titleSettings struct {
	FirstLaunchComplete bool   `json:"firstLaunchComplete"`
	CameraMotion        string `json:"cameraMotion"`
	Subtitles           bool   `json:"subtitles"`
	RememberChoice      bool   `json:"rememberChoice"`
	ReducedMotion       bool   `json:"reducedMotion"`
}

// saveSlotPaths must match the save system's slot naming convention.
var saveSlotPaths = []string{
	"saves/slot1.json",
	"saves/slot2.json",
	"saves/slot3.json",
}

// Focus traversal order per state, over the ids in the markup.
var (
	menuFocusItems    = []string{"btn-new-game", "btn-load-game", "btn-settings", "btn-quit"}
	noSavesFocusItems = []string{"btn-new-game", "btn-settings", "btn-quit"}
	modalFocusItems   = []string{"select-camera-motion", "toggle-subtitles", "checkbox-remember", "btn-start-game"}
	confirmFocusItems = []string{"btn-begin-new-game", "btn-cancel-confirm"}
)

// TitleScreen is the title screen entry point: it owns the markup document,
// the focus controller, the tween runner, and every button handler. All
// motion is driven from Go by polling tween values in Update — the Kaiju CSS
// engine has no transitions.
type TitleScreen struct {
	host    *engine.Host
	doc     *document.Document
	manager *ui.Manager
	tweens  *TweenRunner
	focus   *FocusController

	state   ScreenState
	exiting bool

	exitTarget screenName

	// Elements bound from the markup.
	root         *document.Element // title-root
	menuPanel    *document.Element // menu-panel
	dimOverlay   *document.Element // dim-overlay
	modal        *document.Element // modal-accessibility
	confirmModal *document.Element // modal-confirm
	toast        *document.Element // toast-error
	tooltip      *document.Element // tooltip
	btnLoad      *document.Element // btn-load-game
	selectMotion *document.Element // select-camera-motion
	toggleTrack  *document.Element // toggle-subtitles
	chkRemember  *document.Element // checkbox-remember
	version      *ui.Label         // label-version
	toastMsg     *ui.Label         // label-toast-message

	// subtitles is the accessibility-modal toggle state, persisted on Start
	// Game. Defaults to On per the UX spec.
	subtitles bool

	// Active animations, polled in Update.
	sceneFade  *Tween
	menuSlide  *Tween
	exitFade   *Tween
	toastTimer *Tween
}

// NewTitleScreen loads the title screen markup, binds elements, creates the
// focus controller and tween runner, registers onclick handlers, and selects
// the initial state from first-launch detection and save count.
func NewTitleScreen(host *engine.Host, man *ui.Manager) *TitleScreen {
	t := &TitleScreen{
		host:      host,
		manager:   man,
		tweens:    NewTweenRunner(),
		state:     StateDefault,
		subtitles: true, // spec: Subtitles toggle default On
	}

	s, settingsErr := loadTitleSettings()
	if settingsErr == nil {
		t.tweens.SetReducedMotion(s.ReducedMotion)
		t.subtitles = s.Subtitles
	}

	funcMap := map[string]func(*document.Element){
		"clickNewGame":         t.clickNewGame,
		"clickLoadGame":        t.clickLoadGame,
		"clickSettings":        t.clickSettings,
		"clickQuit":            t.clickQuit,
		"clickStartGame":       t.clickStartGame,
		"clickBeginNewGame":    t.clickBeginNewGame,
		"clickCancelConfirm":   t.clickCancelConfirm,
		"clickDismissToast":    t.clickDismissToast,
		"clickToggleSubtitles": t.clickToggleSubtitles,
	}

	doc, err := markup.DocumentFromHTMLAsset(man, "ui/title_screen.html", nil, funcMap)
	if err != nil {
		slog.Warn("title screen: failed to load markup", "err", err)
		return t
	}
	t.doc = doc
	t.bindElements()

	t.focus = NewFocusController(doc)
	t.focus.SetCancelHandler(t.handleCancel)
	t.focus.SetActivate("btn-new-game", func() { t.clickNewGame(nil) })
	t.focus.SetActivate("btn-load-game", func() { t.clickLoadGame(nil) })
	t.focus.SetActivate("btn-settings", func() { t.clickSettings(nil) })
	t.focus.SetActivate("btn-quit", func() { t.clickQuit(nil) })
	t.focus.SetActivate("btn-start-game", func() { t.clickStartGame(nil) })
	t.focus.SetActivate("btn-begin-new-game", func() { t.clickBeginNewGame(nil) })
	t.focus.SetActivate("btn-cancel-confirm", func() { t.clickCancelConfirm(nil) })

	t.applyToggleVisual()

	if t.version != nil {
		t.version.SetText("v" + GameVersion)
	}

	switch {
	case settingsErr != nil || !s.FirstLaunchComplete:
		t.setState(StateFirstLaunch)
	case !t.hasSaves():
		t.setState(StateNoSaves)
	default:
		t.setState(StateDefault)
	}

	host.Updater.AddUpdate(t.Update)
	return t
}

// Enter begins the entrance sequence: a 500ms atmospheric fade of the whole
// scene, followed after a 300ms delay by a 300ms slide-in of the menu panel.
func (t *TitleScreen) Enter() {
	if t.root == nil {
		return
	}
	t.exiting = false

	t.setRootOpacity(0)
	if t.menuPanel != nil && t.doc != nil {
		t.doc.SetElementStyleProperty(t.menuPanel, "translateY", "40px")
	}

	t.sceneFade = NewTween(0, 1, 0.5, EaseOut).SetAtmospheric()
	t.tweens.Play(t.sceneFade)

	t.menuSlide = NewTween(40, 0, 0.3, EaseOut).WithDelay(0.3)
	t.tweens.Play(t.menuSlide)
}

// Exit begins the 250ms exit crossfade and returns the navigation target.
func (t *TitleScreen) Exit() screenName {
	if t.exiting || t.root == nil {
		return t.exitTarget
	}
	t.exiting = true
	t.exitFade = NewTween(1, 0, 0.25, EaseInOut)
	t.tweens.Play(t.exitFade)
	return t.exitTarget
}

// Update advances the tweens and the focus controller, then applies the
// animated values of any active entrance/exit tweens.
func (t *TitleScreen) Update(dt float64) {
	t.tweens.Update(dt)
	if t.focus != nil {
		t.focus.Update(t.host, dt)
	}

	if t.sceneFade != nil {
		t.setRootOpacity(float64(t.sceneFade.Value()))
		if t.sceneFade.Done() {
			t.sceneFade = nil
		}
	}
	if t.menuSlide != nil {
		if t.menuPanel != nil && t.doc != nil {
			t.doc.SetElementStyleProperty(t.menuPanel, "translateY",
				fmt.Sprintf("%.0fpx", t.menuSlide.Value()))
		}
		if t.menuSlide.Done() {
			t.menuSlide = nil
		}
	}
	if t.exitFade != nil {
		t.setRootOpacity(float64(t.exitFade.Value()))
		if t.exitFade.Done() {
			t.exitFade = nil
			t.exiting = false
		}
	}
	if t.toastTimer != nil && t.toastTimer.Done() {
		t.toastTimer = nil
		if t.state == StateError {
			t.setState(t.postLaunchState())
		}
	}
}

// setState transitions between screen states, toggling element visibility,
// interactivity, and the focus traversal order.
func (t *TitleScreen) setState(s ScreenState) {
	t.state = s
	switch s {
	case StateDefault:
		t.setVisible(t.menuPanel, true)
		t.setVisible(t.modal, false)
		t.setVisible(t.confirmModal, false)
		t.setVisible(t.dimOverlay, false)
		t.setVisible(t.toast, false)
		t.setVisible(t.tooltip, false)
		t.setLoadButtonEnabled(t.hasSaves())
		t.setFocusItems(menuFocusItems)

	case StateFirstLaunch:
		t.setVisible(t.menuPanel, true)
		t.setVisible(t.modal, true)
		t.setVisible(t.confirmModal, false)
		t.setVisible(t.dimOverlay, true)
		t.setVisible(t.toast, false)
		t.setVisible(t.tooltip, false)
		t.setLoadButtonEnabled(t.hasSaves())
		t.setFocusItems(modalFocusItems)

	case StateNoSaves:
		t.setVisible(t.menuPanel, true)
		t.setVisible(t.modal, false)
		t.setVisible(t.confirmModal, false)
		t.setVisible(t.dimOverlay, false)
		t.setVisible(t.toast, false)
		// Tooltip is shown unconditionally: hover gating is impossible because
		// Kaiju blocks all events on disabled widgets (UI.disabledBlocksEvent),
		// so the disabled Load button cannot fire onmouseover. Deliberate
		// simplification — revisit with an enabled wrapper panel if the
		// art director requires hover-only.
		t.setVisible(t.tooltip, true)
		t.setLoadButtonEnabled(false)
		t.setFocusItems(noSavesFocusItems)

	case StateConfirmNew:
		t.setVisible(t.menuPanel, true)
		t.setVisible(t.modal, false)
		t.setVisible(t.confirmModal, true)
		t.setVisible(t.dimOverlay, true)
		t.setVisible(t.toast, false)
		t.setVisible(t.tooltip, false)
		t.setFocusItems(confirmFocusItems)

	case StateLoading:
		// Hard cut to soot: the root's own background (#141210) shows through.
		// Root opacity is left untouched so the background is visible.
		t.setVisible(t.menuPanel, false)
		t.setVisible(t.modal, false)
		t.setVisible(t.confirmModal, false)
		t.setVisible(t.dimOverlay, false)
		t.setVisible(t.toast, false)
		t.setVisible(t.tooltip, false)
		t.clearFocus()

	case StateError:
		// UX spec: "Menu fully interactive" during the toast. No dim overlay,
		// and the first-launch modal is dismissed (a write failure must not
		// strand the player behind a full-screen modal — next launch re-prompts
		// because the settings file was never written).
		t.setVisible(t.modal, false)
		t.setVisible(t.confirmModal, false)
		t.setVisible(t.dimOverlay, false)
		t.setVisible(t.toast, true)
		t.setVisible(t.tooltip, false)
		t.setFocusItems(menuFocusItems)
	}
}

// bindElements resolves the markup ids into element references.
func (t *TitleScreen) bindElements() {
	if t.doc == nil {
		return
	}
	t.root = t.element("title-root")
	if t.root == nil {
		return
	}
	t.menuPanel = t.element("menu-panel")
	t.dimOverlay = t.element("dim-overlay")
	t.modal = t.element("modal-accessibility")
	t.confirmModal = t.element("modal-confirm")
	t.toast = t.element("toast-error")
	t.tooltip = t.element("tooltip")
	t.btnLoad = t.element("btn-load-game")
	t.selectMotion = t.element("select-camera-motion")
	t.toggleTrack = t.element("toggle-subtitles")
	t.chkRemember = t.element("checkbox-remember")
	t.version = t.root.FindElementLabelById("label-version")
	t.toastMsg = t.root.FindElementLabelById("label-toast-message")
}

func (t *TitleScreen) element(id string) *document.Element {
	elm, ok := t.doc.GetElementById(id)
	if !ok {
		slog.Warn("title screen: markup element not found", "id", id)
		return nil
	}
	return elm
}

// setVisible shows or hides a panel via inline display property.
func (t *TitleScreen) setVisible(elm *document.Element, visible bool) {
	if elm == nil || t.doc == nil {
		return
	}
	if visible {
		t.doc.SetElementStyleProperty(elm, "display", "flex")
	} else {
		t.doc.SetElementStyleProperty(elm, "display", "none")
	}
}

func (t *TitleScreen) setRootOpacity(v float64) {
	if t.doc == nil || t.root == nil {
		return
	}
	t.doc.SetElementStyleProperty(t.root, "opacity", fmt.Sprintf("%.3f", v))
}

// setLoadButtonEnabled toggles the Load Game button enabled/disabled state.
func (t *TitleScreen) setLoadButtonEnabled(enabled bool) {
	if t.doc == nil || t.btnLoad == nil {
		return
	}
	if enabled {
		classes := t.btnLoad.ClassList()
		filtered := make([]string, 0, len(classes))
		for _, c := range classes {
			if c != "menu-button-disabled" {
				filtered = append(filtered, c)
			}
		}
		t.doc.SetElementClasses(t.btnLoad, filtered...)
	} else if !t.btnLoad.HasClass("menu-button-disabled") {
		t.doc.SetElementClasses(t.btnLoad, append(t.btnLoad.ClassList(), "menu-button-disabled")...)
	}
	t.doc.SetElementDisabled(t.btnLoad, !enabled)
}

func (t *TitleScreen) setFocusItems(ids []string) {
	if t.focus == nil {
		return
	}
	if err := t.focus.SetItems(ids...); err != nil {
		slog.Warn("title screen: focus setup", "err", err)
	}
}

func (t *TitleScreen) clearFocus() {
	if t.focus != nil {
		t.focus.ClearFocus()
	}
}

func (t *TitleScreen) hasSaves() bool {
	for _, p := range saveSlotPaths {
		if filesystem.FileExists(p) {
			return true
		}
	}
	return false
}

// postLaunchState returns the state to enter after first launch or a dialog
// completes, based on whether save files exist.
func (t *TitleScreen) postLaunchState() ScreenState {
	if t.hasSaves() {
		return StateDefault
	}
	return StateNoSaves
}

// saveSettings persists the first-launch configuration to settingsPath.
func (t *TitleScreen) saveSettings(s titleSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return filesystem.WriteTextFile(settingsPath, string(data))
}

// loadTitleSettings reads the persisted first-launch configuration. A missing
// or unparseable file is reported as an error (treated as first launch).
func loadTitleSettings() (titleSettings, error) {
	var s titleSettings
	data, err := filesystem.ReadTextFile(settingsPath)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return s, err
	}
	return s, nil
}

// Button handlers -----------------------------------------------------------

func (t *TitleScreen) clickNewGame(_ *document.Element) {
	switch t.state {
	case StateDefault, StateNoSaves:
	default:
		return
	}
	if t.exiting {
		return
	}
	if t.hasSaves() {
		t.setState(StateConfirmNew)
		return
	}
	t.beginNewGame()
}

func (t *TitleScreen) clickLoadGame(_ *document.Element) {
	if t.exiting {
		return
	}
	// Disabled-state guard: Kaiju blocks UI events on disabled widgets, but
	// focus activation bypasses the event system, so guard here too. The
	// lingering amber hover line on the disabled button is a known CSS
	// limitation (no :not() support) — visual only.
	if t.btnLoad != nil && t.btnLoad.HasClass("menu-button-disabled") {
		return
	}
	if t.state != StateDefault {
		return
	}
	t.exitTarget = ScreenLoadGame
	t.Exit()
}

func (t *TitleScreen) clickSettings(_ *document.Element) {
	if (t.state != StateDefault && t.state != StateNoSaves) || t.exiting {
		return
	}
	t.exitTarget = ScreenSettings
	t.Exit()
}

func (t *TitleScreen) clickQuit(_ *document.Element) {
	t.host.Close()
}

func (t *TitleScreen) clickStartGame(_ *document.Element) {
	if t.state != StateFirstLaunch {
		return
	}
	s := titleSettings{
		FirstLaunchComplete: true,
		CameraMotion:        "Reduced", // spec default; select parser default guard
		Subtitles:           t.subtitles,
		RememberChoice:      true, // spec default; checkbox is checked by default
	}
	if t.selectMotion != nil && t.selectMotion.UI != nil {
		if v := t.selectMotion.UI.ToSelect().Value(); v != "" {
			s.CameraMotion = v
		}
	}
	if t.chkRemember != nil && t.chkRemember.UI != nil {
		s.RememberChoice = t.chkRemember.UI.ToCheckbox().IsChecked()
	}
	s.ReducedMotion = s.CameraMotion != "Off"
	if err := t.saveSettings(s); err != nil {
		slog.Warn("title screen: failed to write settings", "err", err)
		t.showError("Could not save settings.")
		return
	}
	t.tweens.SetReducedMotion(s.ReducedMotion)
	t.setState(t.postLaunchState())
}

func (t *TitleScreen) clickBeginNewGame(_ *document.Element) {
	if t.state != StateConfirmNew {
		return
	}
	t.beginNewGame()
}

func (t *TitleScreen) clickCancelConfirm(_ *document.Element) {
	if t.state != StateConfirmNew {
		return
	}
	t.setState(t.postLaunchState())
}

func (t *TitleScreen) clickToggleSubtitles(_ *document.Element) {
	if t.state != StateFirstLaunch {
		return
	}
	t.subtitles = !t.subtitles
	t.applyToggleVisual()
}

func (t *TitleScreen) clickDismissToast(_ *document.Element) {
	t.dismissToast()
}

// handleCancel dismisses whatever cancelable layer is showing: the new-game
// confirmation dialog or the error toast. Escape and gamepad B route here.
func (t *TitleScreen) handleCancel() {
	switch t.state {
	case StateConfirmNew:
		t.clickCancelConfirm(nil)
	case StateError:
		t.dismissToast()
	}
}

// dismissToast hides the error toast immediately; the pending auto-dismiss
// tween is dropped (it would otherwise fire a no-op state transition).
func (t *TitleScreen) dismissToast() {
	if t.state != StateError {
		return
	}
	t.toastTimer = nil
	t.setState(t.postLaunchState())
}

// applyToggleVisual reflects the subtitles toggle state on the track: knob
// slides to the oxblood side when on. Inline styles win the cascade over the
// .toggle-track rules.
func (t *TitleScreen) applyToggleVisual() {
	if t.doc == nil || t.toggleTrack == nil {
		return
	}
	if t.subtitles {
		t.doc.SetElementStyleProperty(t.toggleTrack, "justify-content", "flex-end")
		t.doc.SetElementStyleProperty(t.toggleTrack, "background-color", "#7A2E28")
	} else {
		t.doc.SetElementStyleProperty(t.toggleTrack, "justify-content", "flex-start")
		t.doc.SetElementStyleProperty(t.toggleTrack, "background-color", "#2E2820")
	}
}

func (t *TitleScreen) beginNewGame() {
	t.setState(StateLoading)
	// TODO(game layer): navigate to difficulty selection here.
}

func (t *TitleScreen) showError(msg string) {
	if t.toastMsg != nil {
		t.toastMsg.SetText(msg)
	}
	t.setState(StateError)
	t.toastTimer = NewTween(0, 0, 5, EaseOut)
	t.tweens.Play(t.toastTimer)
}

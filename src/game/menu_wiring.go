// menu_wiring.go wires the production main-menu screen flow (Story 008:
// Game-Layer Wiring — Title, Settings & Keybindings Live) into the running
// game. It owns:
//
//   - the ScreenFlow state machine with the real Title/Settings/Keybindings/
//     PauseMenu controllers (src/ui/mainmenu), starting in the GDD boot state
//     ScreenTitleScreen (AC-1);
//   - the real settings.cfg file adapter (settings_cfg.go, AC-2), shared by
//     the settings and keybindings screens via their reader/writer/has-file
//     seams;
//   - the Kaiju render adapter: four static markup documents (title,
//     settings, keybindings, pause) built once at boot, label pools bound
//     per frame from the screens' render-command streams, and entity
//     Activate/Deactivate visibility — no runtime UI tree mutations
//     (ADR-0012 UI recipe);
//   - the host input routing contract (AC-3/AC-4): raw keys route to the
//     keybindings screen first while CapturesInput() is true; otherwise they
//     dispatch through the flow; settings preview overlay keys and the pause
//     quit-confirm dialog are consumed before the generic route.
//
// The world (world.go) is untouched: New Game builds the slice world through
// the same startNewGame path (AC-1), and the world stays live behind the
// pause/settings parchment (AC-5; the freeze contract is Story 006's
// out-of-scope follow-up).
package game

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/platform/hid"

	"marrow-of-empire/ui/mainmenu"
)

// menuWiring is the game layer's main-menu runtime: the screen flow, the
// real settings.cfg adapter, the four screen controllers, and the render
// adapter state (documents, per-frame command scratch buffers, per-slot
// focus ids).
type menuWiring struct {
	g *Game

	// flow is the ScreenFlow state machine; boot state is TitleScreen.
	flow *mainmenu.ScreenFlow
	// cfg is the real settings.cfg file adapter shared by all screens.
	cfg *SettingsCfg

	title    *mainmenu.TitleScreen
	settings *mainmenu.SettingsScreen
	keybind  *mainmenu.KeybindingsScreen
	pause    *mainmenu.PauseMenu

	titleDoc    *document.Document
	settingsDoc *document.Document
	keybindDoc  *document.Document
	pauseDoc    *document.Document

	// lastScreen is the screen whose documents were last shown; visibility
	// switches only on change (entity activation is applied once, not per
	// frame). The initial sentinel forces the first render to hide the three
	// inactive documents (markup starts every element active).
	lastScreen mainmenu.Screen

	// settingsSlots maps settings row slot index -> the row's current focus
	// id (the render command's FocusID), read by the slot click handlers.
	settingsSlots [settingsRowSlots]string

	// kbFocusIDs maps keybinding row slot index -> the action's three focus
	// ids (primary/secondary/restore), derived per frame from the render
	// command's FocusID and read by the slot click handlers.
	kbFocusIDs [keybindRowSlots]keybindSlotIDs

	settingsCmds []mainmenu.RenderCommand
	keybindCmds  []mainmenu.KeybindingsRenderCommand

	// frame is a TEMP-DEBUG frame counter (removed after live verification).
	frame int
}

// keybindSlotIDs holds one keybinding row's clickable focus ids.
type keybindSlotIDs struct {
	primary   string
	secondary string
	restore   string
}

// newMenuWiring builds the production flow, registers the real controllers,
// constructs the four static markup documents, and binds every interactive
// element's click to a flow mouse-activation dispatch. Called from Launch
// after the shared ui.Manager is initialized.
func newMenuWiring(g *Game) *menuWiring {
	// The screens' clock is milliseconds (DebounceMs etc.); host.Runtime is
	// seconds. One time source drives the flow, the screens, and the
	// transition system (AC-7 timing contract).
	now := func() float64 { return g.host.Runtime() * 1000 }

	m := &menuWiring{g: g}
	m.flow = mainmenu.NewScreenFlow(mainmenu.NewTransitionSystem(now))
	m.cfg = newSettingsAdapter()

	m.title = mainmenu.NewTitleScreen(m.flow,
		mainmenu.WithClock(now),
		mainmenu.WithHasSaves(func() bool { return false }), // INT-6 wires real saves
		mainmenu.WithIsSaving(func() bool { return false }),
		mainmenu.WithBeginNewGame(func(d mainmenu.Difficulty, name string) {
			slog.Info("New Game — entering the world", "difficulty", d.String(), "name", name)
			g.startNewGame()
			// Loading resolves into InGame once the world is built (the
			// crossfade completes over the next frames).
			_ = m.flow.RequestTransition(mainmenu.ScreenInGame)
		}),
		mainmenu.WithQuitToDesktop(func() { g.host.Close() }),
	)
	_ = m.flow.RegisterController(mainmenu.ScreenTitleScreen, m.title)

	m.settings = mainmenu.NewSettingsScreen(m.flow,
		mainmenu.WithSettingsClock(now),
		mainmenu.WithSettingsReader(m.cfg.Get),
		mainmenu.WithSettingsWriter(m.cfg.SetAndFlush),
		mainmenu.WithHasSettingsFile(m.cfg.HasFile),
		mainmenu.WithSettingsLogger(slog.Warn),
		mainmenu.WithApplySetting(func(key, value string) {
			// Instant-apply seam: target systems apply on their next frame.
			// No gameplay system consumes settings yet — logged only.
			slog.Info("setting applied", "key", key, "value", value)
		}),
	)
	// AC-2 boot probe: a missing/corrupt settings.cfg keeps defaults and
	// flags the (Story 003) first-launch prompt path; a valid file
	// pre-populates the store.
	m.settings.CheckFirstLaunch()
	_ = m.flow.RegisterController(mainmenu.ScreenSettings, m.settings)

	m.keybind = mainmenu.NewKeybindingsScreen(m.flow,
		mainmenu.WithKeybindingsClock(now),
		mainmenu.WithKeybindingsReader(m.cfg.Get),
		mainmenu.WithKeybindingsWriter(m.cfg.SetAndFlush),
		mainmenu.WithKeybindingsHasSettingsFile(m.cfg.HasFile),
		mainmenu.WithKeybindingsLogger(slog.Warn),
		mainmenu.WithBindingChange(func(action string, slot mainmenu.BindingSlot, chord mainmenu.KeyChord) {
			// Forward to the Input Binding System at runtime (its wiring is
			// the story's unlocked follow-up); logged here.
			slog.Info("binding changed", "action", action, "slot", slot.String(), "key", mainmenu.SerializeChord(chord))
		}),
	)
	// AC-8 boot read: re-reads a persisted settings.cfg so rebound keys
	// survive a restart; with no file the defaults stay.
	m.keybind.Load()
	_ = m.flow.RegisterController(mainmenu.ScreenKeybindings, m.keybind)

	m.pause = mainmenu.NewPauseMenu(m.flow, mainmenu.WithPauseClock(now))
	_ = m.flow.RegisterController(mainmenu.ScreenPauseMenu, m.pause)

	m.buildDocuments()

	// Boot state: the title screen is current; the initial visibility switch
	// (hiding the other three documents) happens on the first render.
	m.lastScreen = mainmenu.Screen(-1)
	return m
}

// newSettingsAdapter locates the per-user settings.cfg (ADR-0002). A
// missing user-config dir falls back to the working directory so the game
// still persists settings; the adapter itself never surfaces errors to the
// player (AC-2).
func newSettingsAdapter() *SettingsCfg {
	if path, err := DefaultSettingsPath(); err == nil {
		return NewSettingsCfg(path)
	}
	slog.Warn("no user config dir — settings.cfg falls back to working directory")
	return NewSettingsCfg(filepath.Join("settings.cfg"))
}

// buildDocuments creates the four static markup documents and binds every
// interactive element's click to a flow mouse-activation dispatch. The
// screens' focus managers own the ids; ActivateAt(id) focuses and activates
// (AC-3: click to focus and activate).
func (m *menuWiring) buildDocuments() {
	m.titleDoc = markup.DocumentFromHTMLString(&m.g.uiMan, titleScreenFlowHTML, "", nil, nil, nil)
	m.settingsDoc = markup.DocumentFromHTMLString(&m.g.uiMan, settingsScreenFlowHTML, "", nil, nil, nil)
	m.keybindDoc = markup.DocumentFromHTMLString(&m.g.uiMan, keybindingsScreenFlowHTML, "", nil, nil, nil)
	m.pauseDoc = markup.DocumentFromHTMLString(&m.g.uiMan, pauseMenuFlowHTML, "", nil, nil, nil)

	// Title screen: fixed-id buttons.
	m.bindClick(m.titleDoc, "new-game")
	m.bindClick(m.titleDoc, "load-game")
	m.bindClick(m.titleDoc, "settings")
	m.bindClick(m.titleDoc, "quit")
	m.bindClick(m.titleDoc, "difficulty-narrative")
	m.bindClick(m.titleDoc, "difficulty-normal")
	m.bindClick(m.titleDoc, "difficulty-tactical")
	m.bindClick(m.titleDoc, "name-begin")
	m.bindClick(m.titleDoc, "name-cancel")

	// Settings screen: tabs, footer, dialogs, toast (fixed ids) and the
	// row slots (dynamic focus id per frame).
	for _, id := range []string{
		"tab-gameplay", "tab-camera", "tab-audio", "tab-controls",
		"tab-accessibility", "tab-graphics",
		"settings-reset-category", "settings-reset-all", "settings-back",
		"settings-reset-confirm", "settings-reset-cancel",
		"settings-preview-keep", "settings-preview-revert",
		"toast-dismiss",
	} {
		m.bindClick(m.settingsDoc, id)
	}
	for i := 0; i < settingsRowSlots; i++ {
		m.bindSlotClick(m.settingsDoc, i)
	}

	// Keybindings screen: fixed-id footer/dialog/preset buttons plus the
	// per-action slot and restore buttons (dynamic focus ids per frame).
	for _, id := range []string{
		"kb-gamepad-preset", "kb-restore-all", "kb-back",
		"kb-reserved-bind", "kb-reserved-cancel",
		"kb-restoreall-confirm", "kb-restoreall-cancel",
	} {
		m.bindClick(m.keybindDoc, id)
	}
	for i := 0; i < keybindRowSlots; i++ {
		m.bindKeybindSlotClick(m.keybindDoc, i)
	}

	// Pause menu: options and confirm dialog.
	for _, id := range []string{
		"resume", "save", "load", "settings", "quit",
		"save-and-quit", "quit-without-saving", "cancel",
	} {
		m.bindClick(m.pauseDoc, id)
	}
}

// bindClick wires a document element's click to a flow mouse activation of
// the element's own id.
func (m *menuWiring) bindClick(doc *document.Document, id string) {
	elm, ok := doc.GetElementById(id)
	if !ok {
		slog.Warn("menu markup missing element", "id", id)
		return
	}
	elm.UI.AddEvent(ui.EventTypeClick, func() {
		m.flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id}))
	})
}

// bindSlotClick wires a settings row slot's click to a flow mouse activation
// of the slot's current focus id (set per frame from the render commands).
func (m *menuWiring) bindSlotClick(doc *document.Document, slot int) {
	elm, ok := doc.GetElementById(fmt.Sprintf("settings-row-%d", slot))
	if !ok {
		slog.Warn("menu markup missing settings row slot", "slot", slot)
		return
	}
	elm.UI.AddEvent(ui.EventTypeClick, func() {
		id := m.settingsSlots[slot]
		if id == "" {
			return
		}
		m.flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id}))
	})
}

// bindKeybindSlotClick wires a keybinding row's three slot buttons to flow
// mouse activations of the action's focus ids (set per frame from the render
// commands; the static row ids are positional, not focus-manager ids).
func (m *menuWiring) bindKeybindSlotClick(doc *document.Document, slot int) {
	for _, btn := range []struct {
		id    string
		getID func(*keybindSlotIDs) string
	}{
		{fmt.Sprintf("kb-row-%d-primary", slot), func(k *keybindSlotIDs) string { return k.primary }},
		{fmt.Sprintf("kb-row-%d-secondary", slot), func(k *keybindSlotIDs) string { return k.secondary }},
		{fmt.Sprintf("kb-row-%d-restore", slot), func(k *keybindSlotIDs) string { return k.restore }},
	} {
		elm, ok := doc.GetElementById(btn.id)
		if !ok {
			slog.Warn("menu markup missing keybind slot", "id", btn.id)
			continue
		}
		elm.UI.AddEvent(ui.EventTypeClick, func() {
			id := btn.getID(&m.kbFocusIDs[slot])
			if id == "" {
				return
			}
			m.flow.Dispatch(mainmenu.TranslateMouse(mainmenu.MouseEvent{ClickedID: id}))
		})
	}
}

// update runs once per frame from the game update: advance the flow, drive
// the screens' clocked state (debounce flush, preview countdown, toasts),
// render the current screen, and route raw keys. Runs before the world
// update, so capture-mode keys are consumed before any world input (AC-4).
func (m *menuWiring) update() {
	m.flow.Update()
	m.title.Refresh()
	m.settings.Update()
	m.keybind.Update()
	m.render()
	m.pollRawKeys()
	// TEMP-DEBUG: periodic state dump for live verification.
	m.frame++
	if m.frame%180 == 0 {
		active := func(d *document.Document) int {
			n := 0
			for i := range d.Elements {
				if d.Elements[i].UI.Entity().IsActive() {
					n++
				}
			}
			return n
		}
		slog.Info("TEMP-DEBUG state",
			"screen", m.flow.Current(),
			"titleActive", active(m.titleDoc),
			"settingsActive", active(m.settingsDoc),
			"keybindActive", active(m.keybindDoc),
			"pauseActive", active(m.pauseDoc),
			"settingsState", m.settings.State(),
			"keybindState", m.keybind.State())
	}
}

// Flush persists any pending debounced settings on clean shutdown (AC-2:
// the game layer flushes on host.OnClose).
func (m *menuWiring) Flush() {
	m.settings.Flush()
	m.keybind.Flush()
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// render switches document visibility when the active screen changes and
// binds the current screen's label pools from its render commands.
func (m *menuWiring) render() {
	s := m.flow.Current()
	if s != m.lastScreen {
		m.showDoc(m.titleDoc, s == mainmenu.ScreenTitleScreen)
		m.showDoc(m.settingsDoc, s == mainmenu.ScreenSettings)
		m.showDoc(m.keybindDoc, s == mainmenu.ScreenKeybindings)
		m.showDoc(m.pauseDoc, s == mainmenu.ScreenPauseMenu)
		m.lastScreen = s
	}
	switch s {
	case mainmenu.ScreenTitleScreen:
		m.renderTitle()
	case mainmenu.ScreenSettings:
		m.renderSettings()
	case mainmenu.ScreenKeybindings:
		m.renderKeybindings()
	case mainmenu.ScreenPauseMenu:
		m.renderPause()
	}
}

// showDoc shows or hides every element of a document via entity activation —
// the engine's proven visibility mechanism (CSS display toggles do not
// re-apply layout in this engine version).
func (m *menuWiring) showDoc(doc *document.Document, on bool) {
	if doc == nil {
		return
	}
	for i := range doc.Elements {
		if on {
			doc.Elements[i].UI.Entity().Activate()
		} else {
			doc.Elements[i].UI.Entity().Deactivate()
		}
	}
}

// renderTitle binds the title screen's labels and switches its inline views
// (menu / difficulty / name entry) from the controller state.
func (m *menuWiring) renderTitle() {
	setText(m.titleDoc, "ts-version", m.title.Version())
	setText(m.titleDoc, "new-game", mainmenu.LabelNewGame)
	setText(m.titleDoc, "load-game", mainmenu.LabelLoadGame)
	setText(m.titleDoc, "settings", mainmenu.LabelSettings)
	setText(m.titleDoc, "quit", mainmenu.LabelQuitToDesktop)
	setText(m.titleDoc, "difficulty-narrative", mainmenu.DifficultyNarrative.String())
	setText(m.titleDoc, "difficulty-normal", mainmenu.DifficultyNormal.String())
	setText(m.titleDoc, "difficulty-tactical", mainmenu.DifficultyTactical.String())
	setText(m.titleDoc, "name-begin", "Begin")
	setText(m.titleDoc, "name-cancel", "Cancel")
	name := m.title.CharacterName()
	if name == "" {
		name = mainmenu.DefaultCharacterName
	}
	setText(m.titleDoc, "ts-name-label", name)

	st := m.title.State()
	setElementVisible(m.titleDoc, "ts-menu-view", st == mainmenu.TitleStateMenu)
	setElementVisible(m.titleDoc, "ts-diff-view", st == mainmenu.TitleStateDifficulty)
	setElementVisible(m.titleDoc, "ts-name-view", st == mainmenu.TitleStateNameEntry)
}

// renderSettings consumes the settings screen's render-command stream into
// the static row pool and the fixed overlay elements.
func (m *menuWiring) renderSettings() {
	m.settingsCmds = m.settingsCmds[:0]
	m.settingsCmds = m.settings.RenderCommands(m.settingsCmds)

	// Row pool: tabs first (fixed ids), then the active panel's stream
	// (rows, re-run, reset-category) into the slots, then footer/dialogs.
	rowSlot := 0
	hasDialog := false
	for _, c := range m.settingsCmds {
		switch c.Type {
		case mainmenu.RenderTab:
			tab := fmt.Sprintf("tab-%s", strings.ToLower(c.Category.String()))
			setText(m.settingsDoc, tab, c.Label)
			setSelectedClass(m.settingsDoc, tab, c.Selected)
		case mainmenu.RenderSettingRow, mainmenu.RenderReRunAccessibility, mainmenu.RenderResetCategory:
			if rowSlot >= settingsRowSlots {
				continue
			}
			row := fmt.Sprintf("settings-row-%d", rowSlot)
			m.settingsSlots[rowSlot] = c.FocusID
			setElementVisible(m.settingsDoc, row, true)
			setText(m.settingsDoc, row, c.Label)
			setText(m.settingsDoc, row+"-value", c.Value)
			rowSlot++
		case mainmenu.RenderFooterResetAll:
			setText(m.settingsDoc, "settings-reset-all", c.Label)
		case mainmenu.RenderBack:
			setText(m.settingsDoc, "settings-back", c.Label)
		case mainmenu.RenderResetDialog:
			hasDialog = true
			setText(m.settingsDoc, "st-dialog-msg", c.Label)
			setText(m.settingsDoc, "settings-reset-confirm", mainmenu.LabelConfirm)
			setText(m.settingsDoc, "settings-reset-cancel", mainmenu.LabelCancel)
		case mainmenu.RenderPreviewOverlay:
			setElementVisible(m.settingsDoc, "st-preview", true)
			setText(m.settingsDoc, "st-preview-msg", c.Label)
			setText(m.settingsDoc, "settings-preview-keep", mainmenu.LabelKeepChanges)
			setText(m.settingsDoc, "settings-preview-revert", mainmenu.LabelRevert)
		case mainmenu.RenderToast:
			setElementVisible(m.settingsDoc, "st-toast", true)
			setText(m.settingsDoc, "st-toast-msg", c.Label)
			setText(m.settingsDoc, "toast-dismiss", "Dismiss")
		}
	}
	// Hide unused row slots and inactive overlays.
	for i := rowSlot; i < settingsRowSlots; i++ {
		setElementVisible(m.settingsDoc, fmt.Sprintf("settings-row-%d", i), false)
	}
	setElementVisible(m.settingsDoc, "st-dialog", hasDialog)
	if !m.settings.PreviewActive() {
		setElementVisible(m.settingsDoc, "st-preview", false)
	}
	if !m.settings.ToastVisible() {
		setElementVisible(m.settingsDoc, "st-toast", false)
	}
}

// renderKeybindings consumes the keybindings screen's render-command stream
// into the per-action row pool and the fixed overlays. Each row's clickable
// focus ids are derived from the command's FocusID (the action's primary
// slot id) and stored for the slot click handlers.
func (m *menuWiring) renderKeybindings() {
	m.keybindCmds = m.keybindCmds[:0]
	m.keybindCmds = m.keybind.RenderCommands(m.keybindCmds)

	row := 0
	hasReserved := false
	hasRestoreAll := false
	hasBlocked := false
	for _, c := range m.keybindCmds {
		switch c.Type {
		case mainmenu.RenderBindingRow:
			if row >= keybindRowSlots {
				continue
			}
			rowID := fmt.Sprintf("kb-row-%d", row)
			labelID := rowID + "-label"
			m.kbFocusIDs[row] = keybindSlotIDs{
				primary:   c.FocusID,
				secondary: strings.Replace(c.FocusID, "-primary", "-secondary", 1),
				restore:   strings.Replace(c.FocusID, "-primary", "-restore", 1),
			}
			setElementVisible(m.keybindDoc, rowID, true)
			setText(m.keybindDoc, labelID, c.Label)
			setText(m.keybindDoc, rowID+"-primary", c.Primary)
			setText(m.keybindDoc, rowID+"-secondary", c.Secondary)
			setText(m.keybindDoc, rowID+"-restore", mainmenu.LabelRestoreDefaults)
			row++
		case mainmenu.RenderGamepadPresetRow:
			setText(m.keybindDoc, "kb-preset-label", c.Label)
			setText(m.keybindDoc, "kb-gamepad-preset", c.Value)
		case mainmenu.RenderRestoreAll:
			setText(m.keybindDoc, "kb-restore-all", c.Label)
		case mainmenu.RenderKeybindingsBack:
			setText(m.keybindDoc, "kb-back", c.Label)
		case mainmenu.RenderReservedDialog:
			hasReserved = true
			setText(m.keybindDoc, "kb-dialog-msg", c.Label)
			setText(m.keybindDoc, "kb-reserved-bind", mainmenu.LabelBindAnyway)
			setText(m.keybindDoc, "kb-reserved-cancel", mainmenu.LabelCancel)
		case mainmenu.RenderRestoreAllDialog:
			hasRestoreAll = true
			setText(m.keybindDoc, "kb-dialog-msg", c.Label)
			setText(m.keybindDoc, "kb-restoreall-confirm", mainmenu.LabelConfirm)
			setText(m.keybindDoc, "kb-restoreall-cancel", mainmenu.LabelCancel)
		case mainmenu.RenderBlockedBanner:
			hasBlocked = true
			setText(m.keybindDoc, "kb-blocked", c.Label)
		}
	}
	for i := row; i < keybindRowSlots; i++ {
		setElementVisible(m.keybindDoc, fmt.Sprintf("kb-row-%d", i), false)
	}
	setElementVisible(m.keybindDoc, "kb-dialog", hasReserved || hasRestoreAll)
	if !hasBlocked {
		setText(m.keybindDoc, "kb-blocked", "")
	}
}

// renderPause binds the pause menu options and switches the quit-confirm
// dialog from the controller state.
func (m *menuWiring) renderPause() {
	setText(m.pauseDoc, "pm-title", "Paused")
	for _, o := range m.pause.Options() {
		setText(m.pauseDoc, o.ID, o.Label)
	}
	setText(m.pauseDoc, "save-and-quit", mainmenu.PauseLabelSaveAndQuit)
	setText(m.pauseDoc, "quit-without-saving", mainmenu.PauseLabelQuitWithoutSaving)
	setText(m.pauseDoc, "cancel", mainmenu.LabelCancel)
	setText(m.pauseDoc, "pm-confirm-msg", mainmenu.PauseQuitConfirmMessage)
	setElementVisible(m.pauseDoc, "pm-confirm", m.pause.State() == mainmenu.PauseStateConfirmQuit)
}

// setText binds a label's text through InnerLabel (the label pool recipe:
// markup carries a zero-width-space child so the label exists at bind time).
func setText(doc *document.Document, id, text string) {
	if doc == nil {
		return
	}
	elm, ok := doc.GetElementById(id)
	if !ok {
		return
	}
	lbl := elm.InnerLabel()
	if lbl == nil {
		return
	}
	lbl.SetText(text)
}

// setElementVisible shows or hides one element (and its subtree) via entity
// activation.
func setElementVisible(doc *document.Document, id string, on bool) {
	if doc == nil {
		return
	}
	elm, ok := doc.GetElementById(id)
	if !ok {
		return
	}
	if on {
		elm.UI.Entity().Activate()
	} else {
		elm.UI.Entity().Deactivate()
	}
}

// setSelectedClass toggles the "selected" CSS class (amber indicator + bold
// weight — two cues) on a tab or row.
func setSelectedClass(doc *document.Document, id string, selected bool) {
	if doc == nil {
		return
	}
	elm, ok := doc.GetElementById(id)
	if !ok {
		return
	}
	if selected {
		doc.SetElementClasses(elm, "tab-btn", "selected")
	} else {
		doc.SetElementClasses(elm, "tab-btn")
	}
}

// ---------------------------------------------------------------------------
// Input routing (AC-3/AC-4)
// ---------------------------------------------------------------------------

// rawKeyTable maps hid key codes onto the device-neutral key names the
// mainmenu package uses: letters/digits as-is, nav keys as TranslateKey
// expects ("Escape", "Tab", "Enter", "Space", "ArrowUp"...), modifiers as
// HandleKey accumulates them ("Shift", "Ctrl", "Alt"), and the function keys
// for the reserved F9. Left/right modifier variants collapse onto one name
// (KeyEvent carries booleans, not sides — AC-3).
var rawKeyTable = []struct {
	key  hid.KeyboardKey
	name string
}{
	{hid.KeyboardKeyEscape, "Escape"},
	{hid.KeyboardKeyTab, "Tab"},
	{hid.KeyboardKeySpace, "Space"},
	{hid.KeyboardKeyReturn, "Enter"},
	{hid.KeyboardKeyEnter, "Enter"},
	{hid.KeyboardKeyUp, "ArrowUp"},
	{hid.KeyboardKeyDown, "ArrowDown"},
	{hid.KeyboardKeyLeft, "ArrowLeft"},
	{hid.KeyboardKeyRight, "ArrowRight"},
	{hid.KeyboardKeyLeftShift, "Shift"},
	{hid.KeyboardKeyRightShift, "Shift"},
	{hid.KeyboardKeyLeftCtrl, "Ctrl"},
	{hid.KeyboardKeyRightCtrl, "Ctrl"},
	{hid.KeyboardKeyLeftAlt, "Alt"},
	{hid.KeyboardKeyRightAlt, "Alt"},
	{hid.KeyboardKeyBackspace, "Backspace"},
	{hid.KeyboardKeyDelete, "Delete"},
	{hid.KeyboardKeyInsert, "Insert"},
	{hid.KeyboardKeyHome, "Home"},
	{hid.KeyboardKeyEnd, "End"},
	{hid.KeyboardKeyPageUp, "PageUp"},
	{hid.KeyboardKeyPageDown, "PageDown"},
	{hid.KeyboardKeyBackQuote, "BackQuote"},
	{hid.KeyboardKeyComma, "Comma"},
	{hid.KeyboardKeyPeriod, "Period"},
	{hid.KeyboardKeyForwardSlash, "Slash"},
	{hid.KeyboardKeyBackSlash, "BackSlash"},
	{hid.KeyboardKeyOpenBracket, "OpenBracket"},
	{hid.KeyboardKeyCloseBracket, "CloseBracket"},
	{hid.KeyboardKeySemicolon, "Semicolon"},
	{hid.KeyboardKeyQuote, "Quote"},
	{hid.KeyboardKeyEqual, "Equal"},
	{hid.KeyboardKeyMinus, "Minus"},
	{hid.KeyboardKeyF1, "F1"},
	{hid.KeyboardKeyF2, "F2"},
	{hid.KeyboardKeyF3, "F3"},
	{hid.KeyboardKeyF4, "F4"},
	{hid.KeyboardKeyF5, "F5"},
	{hid.KeyboardKeyF6, "F6"},
	{hid.KeyboardKeyF7, "F7"},
	{hid.KeyboardKeyF8, "F8"},
	{hid.KeyboardKeyF9, "F9"},
	{hid.KeyboardKeyF10, "F10"},
	{hid.KeyboardKeyF11, "F11"},
	{hid.KeyboardKeyF12, "F12"},
}

func init() {
	// Letters A-Z and digits 0-9 extend the static table (hid constants are
	// contiguous from KeyboardKeyA and KeyboardKey0).
	for i := 0; i < 26; i++ {
		rawKeyTable = append(rawKeyTable, struct {
			key  hid.KeyboardKey
			name string
		}{hid.KeyboardKey(hid.KeyboardKeyA + i), string(rune('A' + i))})
	}
	for i := 0; i < 10; i++ {
		rawKeyTable = append(rawKeyTable, struct {
			key  hid.KeyboardKey
			name string
		}{hid.KeyboardKey(hid.KeyboardKey0 + i), string(rune('0' + i))})
	}
}

// pollRawKeys polls the host keyboard for newly-pressed keys (KeyDown =
// pressed this frame) and routes each through the Story 008 host contract:
// capture consumes first, then flow dispatch, with the settings preview
// overlay and the pause quit-confirm dialog ahead of the generic route.
func (m *menuWiring) pollRawKeys() {
	kb := &m.g.host.Window.Keyboard
	for _, k := range rawKeyTable {
		if !kb.KeyDown(k.key) {
			continue
		}
		ev := mainmenu.KeyEvent{Key: k.name, Shift: kb.HasShift(), Ctrl: kb.HasCtrl(), Alt: kb.HasAlt()}
		m.routeKey(ev)
	}
}

// routeKey is the host input routing order (AC-3/AC-4):
//
//  1. The settings preview overlay consumes Y/N/Escape while active (AC-3
//     keep/revert).
//  2. The pause quit-confirm dialog consumes input while open (the pause
//     menu's Dispatch contract — flow-level Escape would resume instead of
//     cancelling).
//  3. RouteRawKey: the keybindings screen first while CapturesInput() is
//     true (a consumed key never reaches the flow or the world — AC-4),
//     then the flow.
func (m *menuWiring) routeKey(ev mainmenu.KeyEvent) {
	before := m.flow.Current()
	if m.flow.Current() == mainmenu.ScreenSettings && m.settings.HandleKey(ev.Key) {
		slog.Info("TEMP-DEBUG key", "key", ev.Key, "consumed", "settings-handler", "screen", before)
		return
	}
	if m.flow.Current() == mainmenu.ScreenPauseMenu && m.pause.Dispatch(mainmenu.TranslateKey(ev)) {
		slog.Info("TEMP-DEBUG key", "key", ev.Key, "consumed", "pause-dispatch", "screen", before)
		return
	}
	consumed := mainmenu.RouteRawKey(m.keybind, m.flow, ev)
	slog.Info("TEMP-DEBUG key", "key", ev.Key, "consumed", consumed, "screen", before, "after", m.flow.Current())
}

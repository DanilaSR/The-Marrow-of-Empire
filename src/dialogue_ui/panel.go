// Package dialogue_ui implements the dialogue panel framework for Marrow of Empire.
// It provides the panel lifecycle, state machine, and input routing for NPC conversations.
//
// Story: EPIC-DialogueUI-001 — Dialogue Panel Framework & Lifecycle
// ADR: ADR-0012 (UI Framework — Desk edge register)
//
// INT-2 note: the panel is built with the markup system
// (markup.DocumentFromHTMLString), the engine's proven UI path. The original
// imperative construction (Layout().Scale + absolute offsets) did not size
// rendered geometry in the engine and was replaced during the vertical slice's
// live verification (first-launch skill failure catalog: "Uniform gray /
// misrendered UI — imperative Scale misuse").
package dialogue_ui

import (
	"fmt"
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/hid"
)

// Layout constants (data-driven — extract to config file when tuning begins).
const (
	topBarHeight    float32 = 56
	bottomBarHeight float32 = 60
	portraitSize    float32 = 48
	portraitPad     float32 = 4
	closeBtnSize    float32 = 32
	closeBtnPad     float32 = 10
	npcNameFontSize float32 = 20
	npcNamePadLeft  float32 = 60
	npcNamePadTop   float32 = 14

	animShowDurationSec float64 = 0.300 // ADR-0012: paper-slide enter 300ms
	animHideDurationSec float64 = 0.200 // ADR-0012: paper-slide exit 200ms
)

// Desk-register colors (art bible §6): parchment panels, ink-brown text.
var (
	colorParchment       = matrix.ColorRGBAInt(242, 230, 209, 250) // #F2E6D1
	colorParchmentDark   = matrix.ColorRGBAInt(230, 217, 196, 255) // #E6D9C4
	colorParchmentBottom = matrix.ColorRGBAInt(224, 212, 191, 255) // #E0D4BF
	colorTextDarkBrown   = matrix.ColorRGBAInt(38, 20, 10, 255)    // #26140A — NOT amber
)

// dialogueHTML is the panel markup: parchment root pinned to the bottom
// third, top bar (name + close), scrollable log, and the question input.
// The log line template is hidden by ID (duplicated lines keep the visible
// class only). No JS — events are wired from Go.
const dialogueHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; }
#dlg-root {
    position: absolute;
    top: 480px;
    left: 0px;
    width: 1280px;
    height: 240px;
    display: flex;
    flex-direction: column;
    background-color: #F2E6D1;
    z-index: 100;
}
#dlg-topbar {
    width: 100%;
    height: 56px;
    display: flex;
    flex-direction: row;
    align-items: center;
    background-color: #E6D9C4;
}
#dlg-name { color: #26140A; font-size: 20px; margin-left: 16px; }
#dlg-close { margin-left: auto; margin-right: 10px; width: 32px; height: 32px; }
#dlg-log {
    width: 100%;
    height: 128px;
    display: flex;
    flex-direction: column;
    padding: 10px 16px;
    overflow: hidden;
}
.dlg-line { color: #26140A; font-size: 16px; }
#dlg-inputrow {
    width: 100%;
    height: 56px;
    display: flex;
    flex-direction: row;
    align-items: center;
    background-color: #E0D4BF;
}
#dlg-input { flex: 1; height: 36px; margin: 0 16px; }
</style>
</head>
<body>
<div id="dlg-root">
    <div id="dlg-topbar">
        <label id="dlg-name">NPC</label>
        <button id="dlg-close">✕</button>
    </div>
    <div id="dlg-log">
        <label id="dlg-line-0" class="dlg-line">&#8203;</label>
        <label id="dlg-line-1" class="dlg-line">&#8203;</label>
        <label id="dlg-line-2" class="dlg-line">&#8203;</label>
        <label id="dlg-line-3" class="dlg-line">&#8203;</label>
        <label id="dlg-line-4" class="dlg-line">&#8203;</label>
        <label id="dlg-line-5" class="dlg-line">&#8203;</label>
        <label id="dlg-line-6" class="dlg-line">&#8203;</label>
        <label id="dlg-line-7" class="dlg-line">&#8203;</label>
    </div>
    <div id="dlg-inputrow">
        <input id="dlg-input" type="text" placeholder="Ask your question..." />
    </div>
</div>
</html>`

// DialoguePanel presents the NPC conversation interface: a parchment panel
// with an NPC name bar, a message log, and a question input.
type DialoguePanel struct {
	host *engine.Host
	man  *ui.Manager

	doc      *document.Document
	rootElm  *document.Element
	nameLbl  *ui.Label
	input    *ui.Input
	closeBtn *ui.Button

	state PanelState

	// Slide animation state
	animating     bool
	animTimer     float64
	animDuration  float64
	animDir       int     // +1 = showing (slide up), -1 = hiding (slide down)
	currentOffset float32 // current Y offset of the root panel
	panelHeight   float32 // computed panel height (screen height / 3)

	updateId     engine.UpdateId
	keyCallbackId hid.KeyCallbackId

	// NPC context
	npcEntityID uint64

	// INT-2 conversation state (messages.go): question input, log lines,
	// and typewriter reveal.
	questionHandler QuestionHandler
	logLabels       []*ui.Label // static pool (logLabelCount), text-only updates
	logLines        []string
	typewriterFull     string
	typewriterRevealed int
	typewriterTimer    float64
}

// NewDialoguePanel creates and initializes the dialogue panel UI hierarchy.
// The panel starts in StateHidden (offscreen). Call Show() to present it.
// man is the game's shared ui.Manager (character_sheet pattern): elements
// created under a second manager do not render (INT-2 live-run fix).
func NewDialoguePanel(host *engine.Host, man *ui.Manager) *DialoguePanel {
	screenH := float32(host.Window.Height())

	dp := &DialoguePanel{
		host:          host,
		man:           man,
		state:         StateHidden,
		panelHeight:   screenH / 3.0,
		currentOffset: screenH / 3.0, // start off-screen (below viewport)
	}

	// Markup construction — the engine's proven sizing path.
	dp.doc = markup.DocumentFromHTMLString(dp.man, dialogueHTML, "", nil, nil, nil)
	if elm, ok := dp.doc.GetElementById("dlg-root"); ok {
		dp.rootElm = elm
	}
	// Static log-label pool: lines update by text assignment only — runtime
	// tree mutations race the UI update pass in this engine version
	// (INT-4 live-run panic: anyChildDirty index out of range).
	dp.logLabels = make([]*ui.Label, 0, logLabelCount)
	for i := 0; i < logLabelCount; i++ {
		if elm, ok := dp.doc.GetElementById(fmt.Sprintf("dlg-line-%d", i)); ok {
			if lbl := elm.InnerLabel(); lbl != nil {
				dp.logLabels = append(dp.logLabels, lbl)
			}
		}
	}
	slog.Debug("dialogue_ui: log label pool bound", "count", len(dp.logLabels))
	if elm, ok := dp.doc.GetElementById("dlg-name"); ok {
		// Markup text elements are panels wrapping an inner label.
		dp.nameLbl = elm.InnerLabel()
	}
	if elm, ok := dp.doc.GetElementById("dlg-input"); ok {
		dp.input = elm.UI.ToInput()
		dp.input.Base().AddEvent(ui.EventTypeEnter, dp.onQuestionEnter)
	}
	if elm, ok := dp.doc.GetElementById("dlg-close"); ok {
		dp.closeBtn = elm.UI.ToButton()
		dp.closeBtn.Base().AddEvent(ui.EventTypeClick, func() { dp.Hide() })
	}

	// Hidden until first Show.
	dp.setVisible(false)

	// Per-frame animation update (UIUpdater runs BEFORE UI render pass;
	// using Updater would cause a 1-frame visual lag).
	dp.updateId = host.UIUpdater.AddUpdate(dp.updateAnimation)

	// Keyboard: Esc → end conversation (in dialogue context only).
	dp.keyCallbackId = host.Window.Keyboard.AddKeyCallback(dp.handleKeyEvent)

	return dp
}

// setVisible toggles the document's rendering via entity activation — the
// engine console's proven show/hide mechanism. The CSS display toggle
// (SetElementStyleProperty "display") does not re-apply layout in this
// engine version (INT-4 live-run finding).
func (dp *DialoguePanel) setVisible(visible bool) {
	for i := range dp.doc.Elements {
		if visible {
			dp.doc.Elements[i].UI.Entity().Activate()
		} else {
			dp.doc.Elements[i].UI.Entity().Deactivate()
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Public API
// ──────────────────────────────────────────────────────────────────────────────

// Show presents the dialogue panel for the given NPC. The panel slides up from
// the bottom with a 300ms ease-out animation (paper-slide motion per ADR-0012).
// The game world remains fully visible above the panel (Desk edge register).
func (dp *DialoguePanel) Show(npcEntityID uint64, npcName string, portraitPath string) {
	dp.npcEntityID = npcEntityID
	dp.SetNPCInfo(npcName, portraitPath)
	dp.SetState(StateActive)
	dp.ClearLog()
	dp.setVisible(true)
	if dp.input != nil {
		dp.input.Focus()
	}
	dp.startAnimation(animShowDurationSec, 1) // 300ms slide-up, ease-out
	slog.Debug("dialogue_ui: panel shown, audio cue: ui_parchment_unfurl",
		"npcEntityID", npcEntityID, "npcName", npcName)
}

// Hide dismisses the dialogue panel. The panel slides down with a 200ms ease-out
// animation (paper-slide motion per ADR-0012). On completion the state transitions
// to StateHidden and the game returns to exploration.
func (dp *DialoguePanel) Hide() {
	if dp.state == StateHidden {
		return
	}
	dp.startAnimation(animHideDurationSec, -1) // 200ms slide-down, ease-out
	slog.Debug("dialogue_ui: panel hidden, audio cue: ui_parchment_roll")
}

// SetNPCInfo updates the top bar with the conversing NPC's name and portrait.
// portraitPath is the asset path to the NPC portrait texture; the slice has no
// portrait art yet, so it is accepted and ignored until art lands.
func (dp *DialoguePanel) SetNPCInfo(name string, portraitPath string) {
	if dp.nameLbl != nil {
		dp.nameLbl.SetText(name)
	}
}

// SetState transitions the panel to a new state.
func (dp *DialoguePanel) SetState(s PanelState) {
	dp.state = s
}

// State returns the current panel state.
func (dp *DialoguePanel) State() PanelState {
	return dp.state
}

// IsVisible reports whether the panel is currently visible (not Hidden).
func (dp *DialoguePanel) IsVisible() bool {
	return dp.state.IsVisible()
}

// CurrentOffset returns the panel's current Y offset from its visible position.
// 0 = fully visible, panelHeight = fully offscreen. Useful for tests.
func (dp *DialoguePanel) CurrentOffset() float32 {
	return dp.currentOffset
}

// Animating reports whether a slide animation is currently in progress.
func (dp *DialoguePanel) Animating() bool {
	return dp.animating
}

// Destroy cleans up the dialogue panel resources: removes update callbacks,
// keyboard handlers, and tears down the document.
func (dp *DialoguePanel) Destroy() {
	dp.host.UIUpdater.RemoveUpdate(&dp.updateId)
	dp.host.Window.Keyboard.RemoveKeyCallback(dp.keyCallbackId)
	dp.doc.Destroy()
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: slide animation
// ──────────────────────────────────────────────────────────────────────────────

// startAnimation begins a show/hide slide: dir +1 slides up (offset shrinks),
// dir -1 slides down (offset grows).
func (dp *DialoguePanel) startAnimation(durationSec float64, dir int) {
	dp.animating = true
	dp.animTimer = 0
	dp.animDuration = durationSec
	dp.animDir = dir
}

// updateAnimation advances the slide and the typewriter reveal.
func (dp *DialoguePanel) updateAnimation(deltaTime float64) {
	// Typewriter reveal advances every frame regardless of slide state.
	dp.updateTypewriter(deltaTime)

	if !dp.animating {
		return
	}

	dp.animTimer += deltaTime
	t := dp.animTimer / dp.animDuration
	if t >= 1.0 {
		t = 1.0
		dp.animating = false
	}

	et := EaseOutCubic(t)

	if dp.animDir == 1 {
		// Showing: offset goes from panelHeight down to 0
		dp.currentOffset = dp.panelHeight * (1.0 - float32(et))
	} else {
		// Hiding: offset goes from 0 up to panelHeight
		dp.currentOffset = dp.panelHeight * float32(et)
	}

	// INT-2 note: the offset is not applied to the element — writing layout
	// offsets to markup elements fights the CSS stylizer in this engine
	// version. Show/hide is a display toggle; the ADR-0012 paper-slide is
	// deferred to a polish story.

	// On hide completion, transition to Hidden
	if !dp.animating && dp.animDir == -1 {
		dp.SetState(StateHidden)
		dp.setVisible(false)
	}
}

// EaseOutCubic implements the cubic ease-out curve: t → 1-(1-t)³.
// Per ADR-0012 paper-slide motion: fast start, gentle stop, no bounce.
// Exported so tests can verify the implementation rather than re-deriving.
func EaseOutCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return 1.0 - (1.0-t)*(1.0-t)*(1.0-t)
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: keyboard input
// ──────────────────────────────────────────────────────────────────────────────

// handleKeyEvent processes raw keyboard events. In dialogue context, Esc ends
// the conversation. This must not conflict with Esc = pause (which is handled
// outside dialogue context by the game layer).
func (dp *DialoguePanel) handleKeyEvent(keyId int, keyState hid.KeyState) {
	if keyId != hid.KeyboardKeyEscape {
		return
	}
	if keyState != hid.KeyStateDown {
		return
	}
	if dp.state == StateHidden {
		return // not in dialogue context — let pause handler take it
	}
	dp.Hide()
}

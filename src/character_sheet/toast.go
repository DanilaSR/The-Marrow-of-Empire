// Toast notifications (Story 006, Epic: Character & Inventory UI): a
// reusable timed-notification overlay per ADR-0012 — brief non-blocking
// toasts for inventory feedback (e.g. "Inventory full", AC-13), designed
// as a general-purpose mechanism for future notification types ("Quest
// item added", "Equipment broken").
//
// The file has two halves, matching the sheet's split between pure display
// logic and engine-bound glue (inventory.go):
//
//   - Toast, ToastMessage, the color token, and PickupBlocked are pure Go
//     with no engine dependency — the queue, countdown, fade, reduced
//     motion, and trigger predicate are unit-tested without the engine
//     running (tests/unit/character_sheet/toast_test.go).
//   - ToastView is the engine-bound presenter: it binds the toast markup
//     elements and applies the Toast state to the DOM each frame.
//
// The toast is a World/Field notification, not a Desk element (ADR-0012
// Decision: "Brief non-blocking toast notifications appear for inventory
// feedback in the Field or World register, not disrupting the Desk
// overlay"). Its markup is a sibling of the sheet overlay in the
// character sheet document, so it stays visible while the sheet is closed
// (QA edge case: pickup attempts during world exploration). Text is amber
// on a dark panel-umber background — never amber on parchment (control
// manifest forbidden pattern). The toast is non-interactive: it carries no
// markup handlers and takes no focus. Kaiju has no working pointer-events
// CSS ("PointerEvents not implemented" in
// kaiju/src/engine/ui/markup/css/properties/css_pointer_events.go) and
// background panels block clicks in their bounds, so the small toast pill
// swallows clicks inside its own bounds while visible — acceptable for a
// ~2s transient, and kept minimal by sizing the pill to its text.
//
// Story: production/epics/character-inventory-ui/
// story-006-inventory-toast-notifications.md
package character_sheet

import (
	"fmt"

	"kaijuengine.com/engine/ui/markup/document"
)

// Toast color tokens, mapped to art-bible palette hex by
// ToastTextColorHex. Every toast is amber today; the token keeps the seam
// for future notification types (quest, equipment, combat — Story 006
// Implementation Notes) that may carry a different text color.
const (
	// ToastColorAmber is the toast text color token (art bible amber
	// #D9A441) — amber text on a dark transient background, never on
	// parchment (control manifest forbidden pattern).
	ToastColorAmber = "amber"
)

// Toast tuning knobs (GDD: toast_duration_ms = 2000, range 1000-4000).
const (
	// inventoryFullToastText is the AC-13 notification line.
	inventoryFullToastText = "Inventory full"
	// inventoryFullToastDurationSec is the AC-13 hold duration: 2000ms
	// (the toast_duration_ms tuning knob).
	inventoryFullToastDurationSec = 2.0
	// toastFadeDurationSec is the fade-out tail after the hold: 300ms.
	// Reduced motion skips it entirely.
	toastFadeDurationSec = 0.300
)

// ToastTextColorHex maps a toast color token to the art-bible palette hex
// value (ADR-0012 desk register palette). Unknown tokens fall back to
// amber. The presenter applies this to the toast text label each frame,
// so the markup CSS carries layout only (matching the weight-bar fill,
// whose palette color comes from Go).
func ToastTextColorHex(token string) string {
	switch token {
	case ToastColorAmber:
		return "#D9A441"
	}
	return "#D9A441"
}

// ToastMessage is one toast notification: the display text and the hold
// duration in seconds. Queued toasts wait until the current toast has
// fully dismissed, then show one at a time (Story 006 Implementation
// Notes).
type ToastMessage struct {
	// Text is the notification line, e.g. "Inventory full" (AC-13).
	Text string
	// DurationSec is how long the toast holds at full opacity before the
	// fade-out, in seconds.
	DurationSec float64
}

// Toast is the toast notification state machine (pure Go, no engine
// dependency): a FIFO queue of ToastMessages of which at most one is
// visible at a time. A toast holds at full opacity for its DurationSec,
// then fades out over toastFadeDurationSec and is dismissed; the next
// queued toast starts. Re-showing the message that is currently displayed
// re-triggers its timer instead of queueing a duplicate (QA AC-13 edge:
// rapid pickup attempts must not spam toasts). The zero value is ready to
// use.
type Toast struct {
	// queue holds toasts waiting for the current one to dismiss.
	queue []ToastMessage
	// current is the toast on screen (or fading out); zero when idle.
	current ToastMessage
	// remaining is the hold time left on current in seconds; 0 while
	// idle or fading.
	remaining float64
	// fading is true during the fade-out tail; visible stays true so the
	// presenter can animate Opacity toward 0.
	fading bool
	// fadeElapsed is the time spent in the fade-out, in seconds.
	fadeElapsed float64
	// visible reports whether a toast is on screen or in its fade-out.
	visible bool
	// reducedMotion skips the fade: on expiry the toast disappears
	// instantly and the queue advances instantly (Story 009 wires the
	// accessibility setting here).
	reducedMotion bool
}

// Show queues a toast with the given text and hold duration in seconds.
// An idle toast displays immediately; otherwise it waits behind the
// current one (one at a time). Re-showing the text of the currently
// displayed toast re-triggers its timer instead of queueing (no toast
// spam, QA AC-13 edge); re-showing the text of the last queued toast is
// ignored for the same reason.
func (t *Toast) Show(message string, durationSec float64) {
	msg := ToastMessage{Text: message, DurationSec: durationSec}
	if !t.visible && !t.fading {
		t.start(msg)
		return
	}
	if t.current.Text == message {
		t.remaining = durationSec
		t.fading = false
		t.fadeElapsed = 0
		return
	}
	if n := len(t.queue); n > 0 && t.queue[n-1].Text == message {
		return
	}
	t.queue = append(t.queue, msg)
}

// InventoryFullToast shows the AC-13 notification: a pickup attempt was
// refused because the pack cannot hold the item, and the item stays in
// the world. Convenience wrapper over Show with the story's tuning-knob
// values (toast_duration_ms = 2000, text "Inventory full").
func (t *Toast) InventoryFullToast() {
	t.Show(inventoryFullToastText, inventoryFullToastDurationSec)
}

// Update advances the toast lifecycle by dt seconds: the hold countdown
// of the visible toast, then the fade-out, then the next queued toast.
// With reduced motion the dismiss and the queue advance are instant.
func (t *Toast) Update(dt float64) {
	if t.fading {
		t.fadeElapsed += dt
		if t.fadeElapsed >= toastFadeDurationSec {
			t.dismiss()
		}
		return
	}
	if !t.visible {
		return
	}
	t.remaining -= dt
	if t.remaining <= 0 {
		if t.reducedMotion {
			t.dismiss()
		} else {
			t.fading = true
			t.fadeElapsed = 0
		}
	}
}

// SetReducedMotion gates the fade: when on, a toast disappears instantly
// on expiry and the queue advances instantly; turning it on mid-fade
// dismisses the fading toast immediately. Story 009 wires the
// accessibility setting here.
func (t *Toast) SetReducedMotion(on bool) {
	t.reducedMotion = on
	if on && t.fading {
		t.dismiss()
	}
}

// Visible reports whether a toast is on screen or in its fade-out.
func (t *Toast) Visible() bool { return t.visible }

// Text returns the currently displayed toast's text; "" while idle.
func (t *Toast) Text() string { return t.current.Text }

// Remaining returns the hold time left on the current toast in seconds; 0
// while idle or fading. The AC-13 duration check (2s ±200ms) reads this
// right after Show.
func (t *Toast) Remaining() float64 { return t.remaining }

// Opacity returns the toast opacity the presenter should render: 1 during
// the hold, falling linearly to 0 across the fade-out, 0 when idle.
func (t *Toast) Opacity() float64 {
	if !t.visible {
		return 0
	}
	if t.fading {
		return 1 - t.fadeElapsed/toastFadeDurationSec
	}
	return 1
}

// dismiss ends the current toast — it disappears (already faded, or
// instantly under reduced motion) — and starts the next queued toast.
func (t *Toast) dismiss() {
	t.fading = false
	t.visible = false
	t.current = ToastMessage{}
	t.remaining = 0
	t.fadeElapsed = 0
	t.advance()
}

// advance starts the next queued toast, if any.
func (t *Toast) advance() {
	if len(t.queue) == 0 {
		return
	}
	msg := t.queue[0]
	t.queue = t.queue[1:]
	t.start(msg)
}

// start shows a toast immediately at full opacity with its full hold
// time.
func (t *Toast) start(msg ToastMessage) {
	t.current = msg
	t.remaining = msg.DurationSec
	t.fading = false
	t.fadeElapsed = 0
	t.visible = true
}

// PickupBlocked reports whether a pickup attempt must be refused because
// the pack cannot hold the item (AC-13): the current carried weight is
// already at or over max capacity, or adding the item's weight would push
// it over. The game layer calls this when the player attempts to pick up
// a world item; on true it cancels the pickup (the item stays in the
// world) and fires the toast via InventoryFullToast. QA edge cases: at
// exactly 100% capacity a pickup still fires the toast (the pack cannot
// exceed capacity), and a pickup that would take 99% over 100% fires too;
// a pickup that lands exactly on 100% is allowed.
func PickupBlocked(currentWeight, itemWeight, maxCapacity int) bool {
	return currentWeight >= maxCapacity || currentWeight+itemWeight > maxCapacity
}

// ToastView is the engine-bound toast presenter: it binds the toast
// markup elements and applies the Toast state to the DOM every frame —
// the root's display (flex/none) and opacity (fade-out), the label's
// text, and the amber text color from the color token. Missing elements
// are logged and left nil; every apply path skips nil elements, so a
// markup mismatch degrades to a no-op toast instead of a crash (matching
// the rest of the sheet).
type ToastView struct {
	doc *document.Document

	// root is the screen-space toast overlay ("toast-root"), a sibling of
	// the sheet overlay so it stays visible while the sheet is closed
	// (world-interaction toasts, QA AC-13 edge).
	root *document.Element
	// text is the amber message label ("toast-text").
	text *document.Element

	// toast is the pure state machine this presenter renders.
	toast *Toast
}

// NewToastView binds the toast markup elements for the sheet's document
// and creates the underlying Toast state machine. The sheet owns the
// lifecycle: Update from the host updater, SetReducedMotion from the
// accessibility setting (Story 009), and the game layer fires
// notifications through Show / InventoryFullToast.
func NewToastView(doc *document.Document) *ToastView {
	return &ToastView{
		doc:   doc,
		root:  sheetElement(doc, "toast-root"),
		text:  sheetElement(doc, "toast-text"),
		toast: &Toast{},
	}
}

// Show queues a toast (Toast.Show); the DOM updates in the same call.
func (v *ToastView) Show(message string, durationSec float64) {
	v.toast.Show(message, durationSec)
	v.apply()
}

// InventoryFullToast shows the AC-13 "Inventory full" notification
// (Toast.InventoryFullToast): the pickup attempt was refused and the item
// stays in the world. The DOM updates in the same call.
func (v *ToastView) InventoryFullToast() {
	v.toast.InventoryFullToast()
	v.apply()
}

// Update advances the toast lifecycle and applies the resulting state to
// the DOM. Called from the sheet's Update, which the host updater runs
// every frame regardless of sheet visibility — so a toast keeps its
// countdown while the sheet is closed.
func (v *ToastView) Update(dt float64) {
	v.toast.Update(dt)
	v.apply()
}

// SetReducedMotion gates the toast fade (Toast.SetReducedMotion): instant
// appear/disappear. Story 009 wires the accessibility setting here.
func (v *ToastView) SetReducedMotion(on bool) {
	v.toast.SetReducedMotion(on)
	v.apply()
}

// apply writes the toast state into the markup: the root's display and
// opacity, the label's text, and the amber text color from the color
// token. Skipped entirely when the markup never loaded or the elements
// are missing.
func (v *ToastView) apply() {
	if v.doc == nil || v.root == nil {
		return
	}
	display := "none"
	if v.toast.Visible() {
		display = "flex"
	}
	v.doc.SetElementStyleProperty(v.root, "display", display)
	v.doc.SetElementStyleProperty(v.root, "opacity", fmt.Sprintf("%.3f", v.toast.Opacity()))
	if v.text != nil {
		v.doc.SetElementStyleProperty(v.text, "color", ToastTextColorHex(ToastColorAmber))
		if lbl := v.text.InnerLabel(); lbl != nil {
			lbl.SetText(v.toast.Text())
		}
	}
}

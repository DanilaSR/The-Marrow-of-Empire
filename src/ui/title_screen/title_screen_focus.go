package title_screen

import (
	"fmt"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/platform/hid"
)

// focusedClass is the CSS class applied to the focused element. The
// stylesheet must draw the oxblood ring with the outline shorthand
// (outline: 2px solid #7A2E28) — the outline longhand properties are not
// implemented in the Kaiju CSS engine.
const focusedClass = "focused"

// FocusController manages keyboard and gamepad focus traversal over an
// ordered list of markup elements, identified by their HTML id attribute.
// Navigation wraps around (Quit to Desktop -> New Game and vice versa),
// activation dispatches to a per-element handler, and the focused element
// carries the oxblood ring class at all times — focus is never gated behind
// animation (art bible 6.5/6.7).
type FocusController struct {
	doc        *document.Document
	order      []string
	base       map[string][]string // base classes per id, restored on blur
	index      int
	focused    bool
	onActivate map[string]func()
	onCancel   func()
}

// NewFocusController creates a focus controller over the given document.
func NewFocusController(doc *document.Document) *FocusController {
	return &FocusController{
		doc:        doc,
		base:       map[string][]string{},
		onActivate: map[string]func(){},
	}
}

// SetItems replaces the focus order and focuses the first element. The
// current class list of each element is captured so blur can restore it.
// Missing element ids are skipped and reported in the returned error.
func (f *FocusController) SetItems(ids ...string) error {
	f.blur()
	f.order = nil
	var missing []string
	for _, id := range ids {
		elm, ok := f.doc.GetElementById(id)
		if !ok {
			missing = append(missing, id)
			continue
		}
		f.base[id] = elm.ClassList()
		f.order = append(f.order, id)
	}
	f.index = 0
	f.focused = false
	if len(f.order) > 0 {
		f.focused = true
		f.applyRing()
	}
	if len(missing) > 0 {
		return fmt.Errorf("focus items not found in document: %v", missing)
	}
	return nil
}

// HandleTab moves focus forward (forward=true) or backward, wrapping from
// the last element to the first and vice versa.
func (f *FocusController) HandleTab(forward bool) {
	if len(f.order) == 0 {
		return
	}
	next := f.index
	if !f.focused {
		next = 0
		if !forward {
			next = len(f.order) - 1
		}
	} else if forward {
		next = (f.index + 1) % len(f.order)
	} else {
		next = f.index - 1
		if next < 0 {
			next = len(f.order) - 1
		}
	}
	f.moveTo(next)
}

// HandleGamepad moves focus by direction: -1 moves up the list, +1 moves down.
func (f *FocusController) HandleGamepad(direction int) {
	if direction >= 0 {
		f.HandleTab(true)
	} else {
		f.HandleTab(false)
	}
}

// Activate triggers the activation handler of the focused element, if any.
func (f *FocusController) Activate() {
	if !f.focused || len(f.order) == 0 {
		return
	}
	if fn, ok := f.onActivate[f.order[f.index]]; ok && fn != nil {
		fn()
	}
}

// FocusIndex focuses the element at index. Indexes beyond the list are
// clamped; negative indexes clear focus.
func (f *FocusController) FocusIndex(index int) {
	if len(f.order) == 0 {
		return
	}
	if index < 0 {
		f.ClearFocus()
		return
	}
	if index >= len(f.order) {
		index = len(f.order) - 1
	}
	f.moveTo(index)
}

// FocusedIndex returns the focused position, or -1 when nothing is focused.
func (f *FocusController) FocusedIndex() int {
	if !f.focused {
		return -1
	}
	return f.index
}

// FocusedID returns the id of the focused element, or "" when nothing is focused.
func (f *FocusController) FocusedID() string {
	if !f.focused {
		return ""
	}
	return f.order[f.index]
}

// SetActivate registers the activation handler for the element with the given id.
func (f *FocusController) SetActivate(id string, fn func()) {
	f.onActivate[id] = fn
}

// SetCancelHandler registers the handler invoked when the player presses
// Escape or gamepad B. Used to dismiss the confirmation dialog and the
// error toast (UX spec: interaction map).
func (f *FocusController) SetCancelHandler(fn func()) {
	f.onCancel = fn
}

// ClearFocus removes the ring from every element and drops focus.
func (f *FocusController) ClearFocus() {
	f.blur()
	f.focused = false
}

// Update polls keyboard and gamepad state and drives traversal.
func (f *FocusController) Update(host *engine.Host, _ float64) {
	kb := &host.Window.Keyboard
	if kb.KeyDown(hid.KeyboardKeyTab) {
		f.HandleTab(!kb.HasShift())
	}
	if kb.KeyDown(hid.KeyboardKeyEnter) || kb.KeyDown(hid.KeyboardKeySpace) {
		f.Activate()
	}
	if kb.KeyDown(hid.KeyboardKeyEscape) {
		if f.onCancel != nil {
			f.onCancel()
		}
	}
	ctl := &host.Window.Controller
	for i := 0; i < hid.ControllerMaxDevices; i++ {
		if !ctl.Available(i) {
			continue
		}
		dev := hid.ControllerButton(i) // device index is typed ControllerButton
		if ctl.IsButtonDown(dev, hid.ControllerButtonUp) {
			f.HandleGamepad(-1)
		}
		if ctl.IsButtonDown(dev, hid.ControllerButtonDown) {
			f.HandleGamepad(1)
		}
		if ctl.IsButtonDown(dev, hid.ControllerButtonA) {
			f.Activate()
		}
		if ctl.IsButtonDown(dev, hid.ControllerButtonB) {
			if f.onCancel != nil {
				f.onCancel()
			}
		}
	}
}

// moveTo blurs the current element, applies focus to the element at index
// and shows its ring.
func (f *FocusController) moveTo(index int) {
	f.blur()
	f.index = index
	f.focused = true
	f.applyRing()
}

// blur removes the ring class from the currently focused element.
func (f *FocusController) blur() {
	if !f.focused || len(f.order) == 0 {
		return
	}
	f.applyClasses(f.order[f.index], f.base[f.order[f.index]])
}

// applyRing adds the ring class to the currently focused element.
func (f *FocusController) applyRing() {
	if !f.focused || len(f.order) == 0 {
		return
	}
	id := f.order[f.index]
	// TODO(outline-offset): the .focused CSS draws the ring, but the 2px
	// offset from the design brief must be applied Go-side. The Kaiju CSS
	// engine errors on the outline-offset longhand, and there is no
	// offset-only setter — use elm.UIPanel.SetOutline(width, offset, color)
	// here (and restore the base offset in blur). CSS outline processing
	// preserves the panel's current offset, so setting it once also works.
	classes := append(append([]string{}, f.base[id]...), focusedClass)
	f.applyClasses(id, classes)
}

// applyClasses sets the full class list of the element with the given id.
func (f *FocusController) applyClasses(id string, classes []string) {
	if elm, ok := f.doc.GetElementById(id); ok {
		f.doc.SetElementClasses(elm, classes...)
	}
}

package mainmenu

import "errors"

var (
	errDuplicateFocusable = errors.New("focusable id already registered")
	errUnknownFocusable   = errors.New("unknown focusable id")
)

// InputMethod identifies the device family that last drove navigation.
// The focus ring is always present during keyboard/gamepad navigation and
// may be hidden during mouse navigation (AC-4), so the method determines
// ring visibility.
type InputMethod int

const (
	// InputMethodNone is the initial state before any input: no ring.
	InputMethodNone InputMethod = iota
	// InputMethodKeyboard shows the ring.
	InputMethodKeyboard
	// InputMethodGamepad shows the ring.
	InputMethodGamepad
	// InputMethodMouse hides the ring (may be shown again on first
	// keyboard/gamepad input).
	InputMethodMouse
)

// Direction is a directional navigation intent from arrows or D-pad.
type Direction int

const (
	DirectionUp Direction = iota
	DirectionDown
	DirectionLeft
	DirectionRight
)

// FocusOption configures a focusable element at registration time.
type FocusOption func(*Focusable)

// WithGroup assigns the element to a navigation group. Arrow keys move
// within the group (wrapping); Tab moves across the whole screen. Elements
// without a group form the implicit top-level group "".
func WithGroup(group string) FocusOption {
	return func(f *Focusable) { f.group = group }
}

// WithActivate registers the activation handler (Enter/Space, gamepad A,
// or click).
func WithActivate(fn func()) FocusOption {
	return func(f *Focusable) { f.activate = fn }
}

// WithScroll registers the scroll handler for lists/sliders (mouse wheel).
func WithScroll(fn func(dir int)) FocusOption {
	return func(f *Focusable) { f.scroll = fn }
}

// WithBackground declares the element's background color so the focus ring
// can be selected with ≥3:1 contrast against it (AC-4). Defaults to
// ColorParchment.
func WithBackground(bg Color) FocusOption {
	return func(f *Focusable) { f.background = bg }
}

// Focusable is a single interactive element in the focus registry.
type Focusable struct {
	id         string
	group      string
	enabled    bool
	background Color
	activate   func()
	scroll     func(dir int)
}

// FocusIndicator is the rendered focus treatment of an element: a color
// plus a non-color shape cue (the ring width). AC-5: color changes are
// never the sole indicator of focus — the 2px outline always accompanies
// the color.
type FocusIndicator struct {
	// Color is the ring color, selected for ≥3:1 against the element
	// background.
	Color Color
	// WidthPx is the ring width (≥ MinRingWidthPx, DPI-scaled). This
	// non-color component is the secondary focus cue.
	WidthPx float64
}

// FocusManager owns the focusable registry of one screen. Registration
// order IS the Tab order (screens register in markup order). Tab/Shift+Tab
// move across the whole registry (wrapping, skipping disabled elements);
// arrows/D-pad move within the current element's group; Enter/Space/A
// activate; Escape/B go back; the mouse focuses+activates by id.
type FocusManager struct {
	style       *FocusStyle
	order       []string
	items       map[string]*Focusable
	index       int
	focused     bool
	inputMethod InputMethod
}

// NewFocusManager creates an empty focus manager using the given style.
func NewFocusManager(style *FocusStyle) *FocusManager {
	return &FocusManager{
		style: style,
		items: map[string]*Focusable{},
		index: -1,
	}
}

// Register adds a focusable element, appending it to the Tab order.
// Duplicate ids are rejected.
func (f *FocusManager) Register(id string, opts ...FocusOption) error {
	if _, ok := f.items[id]; ok {
		return errDuplicateFocusable
	}
	it := &Focusable{id: id, enabled: true, background: ColorParchment}
	for _, o := range opts {
		if o != nil {
			o(it)
		}
	}
	f.items[id] = it
	f.order = append(f.order, id)
	return nil
}

// SetEnabled enables or disables an element. Disabled elements are skipped
// by traversal. Disabling the focused element moves focus to the next
// enabled element (wrapping); with no enabled elements left, focus clears.
func (f *FocusManager) SetEnabled(id string, on bool) error {
	it, ok := f.items[id]
	if !ok {
		return errUnknownFocusable
	}
	it.enabled = on
	if !on && f.focused && f.order[f.index] == id {
		if next := f.nextEnabled(f.index, true); next >= 0 {
			f.moveTo(next)
		} else {
			f.ClearFocus()
		}
	}
	return nil
}

// Remove drops an element from the registry. Focus is cleared if the
// removed element was focused.
func (f *FocusManager) Remove(id string) error {
	if _, ok := f.items[id]; !ok {
		return errUnknownFocusable
	}
	delete(f.items, id)
	wasFocused := f.focused && f.order[f.index] == id
	kept := f.order[:0]
	for _, oid := range f.order {
		if oid != id {
			kept = append(kept, oid)
		}
	}
	f.order = kept
	if wasFocused {
		f.ClearFocus()
	}
	return nil
}

// Count returns the number of registered elements.
func (f *FocusManager) Count() int { return len(f.order) }

// EnabledCount returns the number of enabled elements.
func (f *FocusManager) EnabledCount() int {
	n := 0
	for _, id := range f.order {
		if f.items[id].enabled {
			n++
		}
	}
	return n
}

// FocusNext moves focus to the next enabled element in Tab order,
// wrapping from the last to the first. When nothing is focused it focuses
// the first enabled element. Returns false if there is nothing to focus.
func (f *FocusManager) FocusNext() bool {
	if !f.focused || f.index < 0 {
		idx := f.nextEnabled(-1, true)
		if idx < 0 {
			return false
		}
		f.moveTo(idx)
		return true
	}
	idx := f.nextEnabled(f.index, true)
	if idx < 0 {
		return false
	}
	f.moveTo(idx)
	return true
}

// FocusPrev is the Shift+Tab mirror of FocusNext: previous enabled
// element, wrapping to the last when at the first.
func (f *FocusManager) FocusPrev() bool {
	if !f.focused || f.index < 0 {
		idx := f.nextEnabled(-1, false)
		if idx < 0 {
			return false
		}
		f.moveTo(idx)
		return true
	}
	idx := f.nextEnabled(f.index, false)
	if idx < 0 {
		return false
	}
	f.moveTo(idx)
	return true
}

// MoveGroup moves focus within the current element's group:
// Down/Right step forward, Up/Left step backward, wrapping within the
// group. The implicit group "" covers ungrouped top-level elements.
func (f *FocusManager) MoveGroup(dir Direction) bool {
	if !f.focused || f.index < 0 {
		return f.FocusNext()
	}
	forward := dir == DirectionDown || dir == DirectionRight
	idx := f.nextInGroup(f.index, f.items[f.order[f.index]].group, forward)
	if idx < 0 {
		return false
	}
	f.moveTo(idx)
	return true
}

// FocusByID focuses the element with the given id (mouse click focus).
// Disabled elements cannot be focused.
func (f *FocusManager) FocusByID(id string) bool {
	it, ok := f.items[id]
	if !ok || !it.enabled {
		return false
	}
	for i, oid := range f.order {
		if oid == id {
			f.moveTo(i)
			return true
		}
	}
	return false
}

// Activate fires the activation handler of the focused element. Returns
// false when nothing is focused, the focused element is disabled, or the
// focused element has no activation handler.
func (f *FocusManager) Activate() bool {
	if !f.focused || f.index < 0 {
		return false
	}
	it := f.items[f.order[f.index]]
	if !it.enabled {
		return false
	}
	if it.activate != nil {
		it.activate()
		return true
	}
	return false
}

// ActivateAt focuses the element with the given id and activates it —
// the complete mouse-click path (AC-3: click to focus and activate).
func (f *FocusManager) ActivateAt(id string) bool {
	if !f.FocusByID(id) {
		return false
	}
	return f.Activate()
}

// Scroll sends a wheel delta (1 = up, -1 = down) to the focused element's
// scroll handler. Geometry-aware screens may use ScrollAt instead (the
// hovered element is the correct target per AC-3; the focused fallback is
// used until screens own hit-testing).
func (f *FocusManager) Scroll(dir int) bool {
	if !f.focused || f.index < 0 {
		return false
	}
	return f.dispatchScroll(f.order[f.index], dir)
}

// ScrollAt sends a wheel delta to the element with the given id.
func (f *FocusManager) ScrollAt(id string, dir int) bool {
	return f.dispatchScroll(id, dir)
}

func (f *FocusManager) dispatchScroll(id string, dir int) bool {
	it, ok := f.items[id]
	if !ok || it.scroll == nil {
		return false
	}
	it.scroll(dir)
	return true
}

// SetInputMethod records which device family drove the latest input and
// toggles ring visibility accordingly (AC-4: ring always present during
// keyboard/gamepad navigation, may be hidden during mouse navigation).
func (f *FocusManager) SetInputMethod(m InputMethod) { f.inputMethod = m }

// InputMethod returns the current input method.
func (f *FocusManager) InputMethod() InputMethod { return f.inputMethod }

// RingVisible reports whether the focus ring is currently rendered.
func (f *FocusManager) RingVisible() bool {
	return f.inputMethod == InputMethodKeyboard || f.inputMethod == InputMethodGamepad
}

// RingID returns the id of the element currently carrying the ring, or ""
// when the ring is not visible.
func (f *FocusManager) RingID() string {
	if !f.RingVisible() || !f.focused || f.index < 0 {
		return ""
	}
	return f.order[f.index]
}

// FocusedID returns the focused element id, or "" when nothing is focused.
func (f *FocusManager) FocusedID() string {
	if !f.focused || f.index < 0 {
		return ""
	}
	return f.order[f.index]
}

// FocusedIndex returns the focused position in the Tab order, or -1.
func (f *FocusManager) FocusedIndex() int {
	if !f.focused || f.index < 0 {
		return -1
	}
	return f.index
}

// ClearFocus drops focus from every element.
func (f *FocusManager) ClearFocus() {
	f.focused = false
	f.index = -1
}

// IndicatorFor returns the focus indicator (ring color + width) for the
// element with the given id. The color is selected per background so the
// ≥3:1 contrast holds; the width is the DPI-scaled outline (AC-4/AC-5).
func (f *FocusManager) IndicatorFor(id string) (FocusIndicator, bool) {
	it, ok := f.items[id]
	if !ok {
		return FocusIndicator{}, false
	}
	return FocusIndicator{
		Color:   f.style.FocusRingFor(it.background),
		WidthPx: f.style.RingWidth(),
	}, true
}

// moveTo focuses the element at the given registry index.
func (f *FocusManager) moveTo(index int) {
	f.index = index
	f.focused = true
}

// nextEnabled returns the index of the next enabled element scanning from
// fromIdx (exclusive) in the given direction, wrapping; -1 when none.
func (f *FocusManager) nextEnabled(fromIdx int, forward bool) int {
	n := len(f.order)
	if n == 0 {
		return -1
	}
	step := 1
	if !forward {
		step = -1
	}
	for i := 0; i < n; i++ {
		idx := (fromIdx + step*(i+1)) % n
		if idx < 0 {
			idx += n
		}
		if f.items[f.order[idx]].enabled {
			return idx
		}
	}
	return -1
}

// nextInGroup is nextEnabled restricted to elements of the given group.
func (f *FocusManager) nextInGroup(fromIdx int, group string, forward bool) int {
	n := len(f.order)
	if n == 0 {
		return -1
	}
	step := 1
	if !forward {
		step = -1
	}
	for i := 0; i < n; i++ {
		idx := (fromIdx + step*(i+1)) % n
		if idx < 0 {
			idx += n
		}
		it := f.items[f.order[idx]]
		if it.group == group && it.enabled {
			return idx
		}
	}
	return -1
}

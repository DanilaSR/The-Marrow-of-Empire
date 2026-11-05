package mainmenu

import "fmt"

// KeyChord is a keybinding: a key plus optional modifier state
// (GDD CR-6 binding model; consumed by Story 007's keybinding interface).
type KeyChord struct {
	Key         string
	Shift, Ctrl bool
	Alt         bool
}

// ActionBinding names a game action that keys bind to. The reserved
// functions required by AC-6 are PauseMenu (Escape) and QuickLoad (F9).
// The string value is the player-facing function name used in warnings.
type ActionBinding string

const (
	// ActionPauseMenu is the pause menu toggle (reserved: Escape).
	ActionPauseMenu ActionBinding = "Pause Menu"
	// ActionQuickLoad is quick-load (reserved: F9).
	ActionQuickLoad ActionBinding = "Quick Load"
)

// LastReservedBindingError is returned when unbinding the last remaining
// binding of a reserved function. Its message is the AC-6 warning text.
type LastReservedBindingError struct {
	Action ActionBinding
}

// Error returns the exact AC-6 warning:
// "This is the only binding for [function]. It cannot be removed."
func (e LastReservedBindingError) Error() string {
	return fmt.Sprintf("This is the only binding for %s. It cannot be removed.", e.Action)
}

// BindingRegistry owns the action→keybinding map with reserved-key
// protection (AC-6): the Escape key and F9 cannot be completely unbound —
// the last remaining binding of any reserved function cannot be removed.
type BindingRegistry struct {
	bindings map[ActionBinding][]KeyChord
	reserved map[ActionBinding]bool
}

// NewBindingRegistry creates an empty registry with no reserved functions.
func NewBindingRegistry() *BindingRegistry {
	return &BindingRegistry{
		bindings: map[ActionBinding][]KeyChord{},
		reserved: map[ActionBinding]bool{},
	}
}

// NewDefaultBindingRegistry creates the registry with the system defaults:
// Escape → Pause Menu, F9 → Quick Load, both marked reserved (AC-6).
func NewDefaultBindingRegistry() *BindingRegistry {
	r := NewBindingRegistry()
	r.MarkReserved(ActionPauseMenu)
	r.MarkReserved(ActionQuickLoad)
	r.Bind(ActionPauseMenu, KeyChord{Key: "Escape"})
	r.Bind(ActionQuickLoad, KeyChord{Key: "F9"})
	return r
}

// MarkReserved marks an action as reserved: its last binding can never be
// unbound (AC-6).
func (r *BindingRegistry) MarkReserved(action ActionBinding) {
	r.reserved[action] = true
}

// IsReserved reports whether the action is reserved.
func (r *BindingRegistry) IsReserved(action ActionBinding) bool {
	return r.reserved[action]
}

// Bind adds a keybinding to the action. Binding an already-present chord
// is a no-op. Keys are not de-duplicated across actions here — conflict
// resolution is Story 007's keybinding interface concern (GDD CR-6).
func (r *BindingRegistry) Bind(action ActionBinding, chord KeyChord) {
	for _, c := range r.bindings[action] {
		if c == chord {
			return
		}
	}
	r.bindings[action] = append(r.bindings[action], chord)
}

// BindingsFor returns a copy of the action's current bindings.
func (r *BindingRegistry) BindingsFor(action ActionBinding) []KeyChord {
	return append([]KeyChord{}, r.bindings[action]...)
}

// Unbind removes a keybinding from the action. Unbinding the last
// remaining binding of a reserved function is blocked with the AC-6
// warning; unbinding a chord that is not bound is a no-op.
func (r *BindingRegistry) Unbind(action ActionBinding, chord KeyChord) error {
	list := r.bindings[action]
	idx := -1
	for i, c := range list {
		if c == chord {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	if r.reserved[action] && r.BindingCount(action) == 1 {
		return LastReservedBindingError{Action: action}
	}
	r.bindings[action] = append(list[:idx], list[idx+1:]...)
	return nil
}

// BindingCount returns the number of non-zero bindings of the action.
// Zero chords represent unbound slots (Story 007's [primary, secondary]
// slot model), so they are not counted — the AC-6 last-binding gate
// protects the action's actual bindings, not its slot padding.
func (r *BindingRegistry) BindingCount(action ActionBinding) int {
	n := 0
	for _, c := range r.bindings[action] {
		if c != (KeyChord{}) {
			n++
		}
	}
	return n
}

// BindSlot sets the chord at the given slot (0 = primary, 1 = secondary),
// replacing any chord currently in that slot (Story 007 slot model). Slots
// beyond the current list grow the list, padding with zero chords, so an
// action's list is always [primary, secondary]. Setting a chord never
// violates AC-6 (only removal can strip a reserved function's last
// binding), so BindSlot cannot fail.
func (r *BindingRegistry) BindSlot(action ActionBinding, slot int, chord KeyChord) {
	list := r.bindings[action]
	if slot < 0 {
		slot = 0
	}
	if slot >= len(list) {
		grown := make([]KeyChord, slot+1)
		copy(grown, list)
		grown[slot] = chord
		r.bindings[action] = grown
		return
	}
	list[slot] = chord
}

// ClearSlot clears the chord at the given slot, leaving the zero chord in
// place so slot identity survives (a cleared primary does not promote the
// secondary). Clearing the last remaining binding of a reserved function is
// blocked with the AC-6 warning; clearing an already-empty slot is a no-op.
func (r *BindingRegistry) ClearSlot(action ActionBinding, slot int) error {
	list := r.bindings[action]
	if slot < 0 || slot >= len(list) || list[slot] == (KeyChord{}) {
		return nil
	}
	if r.reserved[action] && r.BindingCount(action) == 1 {
		return LastReservedBindingError{Action: action}
	}
	list[slot] = KeyChord{}
	return nil
}

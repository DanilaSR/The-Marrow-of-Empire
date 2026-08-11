// Story 009: Accessibility & Input (Epic: Character & Inventory UI) —
// keyboard navigation, focus management, reduced motion, and the
// resolution floor, per ADR-0012 and TR-ui-006/TR-ui-007.
//
// This file holds the pure navigation logic (KeyboardNav, FocusLayout,
// ResolutionGuard) that runs headless in the unit tests, plus the sheet's
// engine-bound wiring: the per-frame keyboard dispatch, the focus ring
// rendering, the resolution lock, and the close-button/Esc user-close
// path.
//
// Tab order (AC-23): sidebar portraits → attribute bars → equipment slots
// → tabs (Character/Inventory) → close button → wrap back to the sidebar.
// Shift+Tab walks the same order in reverse. Within a group the arrow
// keys move the cursor: ↑/↓ cycle the party portraits (GDD: "↑/↓ to
// cycle members") and step the attribute bars; ←/→ step the equipment
// slots and the tabs. Enter/Space activates the focused element (select
// party member, switch tab, close). Esc closes the UI through the same
// user-close path as the close button (CloseRequested).
//
// The focus ring is a 3px oxblood (#7A2E28) ring against parchment
// (#E3D5B0) — a 6.4:1 contrast, comfortably ≥3:1 (ADR-0012). The ring is
// a visible shape around the element, so focus is never encoded by color
// alone (control manifest forbidden pattern). The sidebar renders its own
// ring (party-portrait-focused); every other group uses the shared
// focus-ring class.
//
// Reduced motion (AC-24): CharacterSheet.SetReducedMotion gates the
// paper-slide (sheet), the level-up modal's hover pulse (Story 002), the
// inventory weight-bar pulse (Story 005), the toast fade (Story 006), and
// the sidebar's 300ms tooltip hover delay — everything lands instantly at
// its final state. The tab content switch is already instant at the DOM
// level (Story 005's showTab toggles display; the 200ms crossfade timer
// is Story 007's session concern).
//
// Resolution floor (AC-25, TR-ui-006): below 1280×720 the sheet shows the
// "Minimum resolution: 1280×720" lock overlay, which covers the parchment
// and swallows every click in its bounds (Kaiju has no working
// pointer-events; background panels block clicks in their bounds —
// toast.go note) while the sheet's keyboard dispatch is gated from Go.
// Exactly 1280×720 IS the minimum and is not locked. The state machine's
// own C/I/Esc polling (Story 007) continues to run alongside — Esc can
// still close a locked UI, matching the QA edge case "close UI (via Esc
// if possible)". Tab order and arrow keys are gated per the AC-25
// interaction block.
//
// Screen-reader labels and the AccessKit integration are declared in the
// markup (aria-label on the close button) pending the engine's screen
// reader hooks; the sheet's keyboard contract is this file.
//
// Story: production/epics/character-inventory-ui/story-009-accessibility-input.md
package character_sheet

import (
	"kaijuengine.com/engine/ui/markup/document"
	"kaijuengine.com/platform/hid"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// FocusGroup identifies one keyboard-focus group of the C&I UI tab order
// (AC-23, ADR-0012).
type FocusGroup int

const (
	// FocusNone: nothing is focused; the next Tab enters the first group.
	FocusNone FocusGroup = iota
	// FocusSidebar: the party portraits (active members, player first).
	FocusSidebar
	// FocusAttributes: the six attribute bars (STR..CHA).
	FocusAttributes
	// FocusEquipment: the eight equipment slots.
	FocusEquipment
	// FocusTabs: the Character/Inventory tab bar.
	FocusTabs
	// FocusClose: the title-bar close button.
	FocusClose
)

// tabOrder is the AC-23 focus sequence: sidebar → attributes → equipment
// slots → tabs → close button, wrapping at both ends.
var tabOrder = [...]FocusGroup{
	FocusSidebar, FocusAttributes, FocusEquipment, FocusTabs, FocusClose,
}

// FocusLayout describes the tab-order groups' sizes at a point in time.
// Zero-size groups are skipped by Tab; HasClose=false removes the close
// stop (degraded markup). The sheet refreshes the layout on every
// keyboard dispatch so roster changes are always reflected.
type FocusLayout struct {
	// SidebarMembers is the number of ACTIVE party members (pool members
	// are unselectable and never receive focus).
	SidebarMembers int
	// Attributes is the number of attribute bars (six).
	Attributes int
	// EquipmentSlots is the number of equipment slots (eight).
	EquipmentSlots int
	// Tabs is the number of tabs (two: Character, Inventory).
	Tabs int
	// HasClose marks the close button present in the markup.
	HasClose bool
}

// size returns the number of focusable elements in a group; 0 marks an
// empty group that Tab skips.
func (l FocusLayout) size(g FocusGroup) int {
	switch g {
	case FocusSidebar:
		return l.SidebarMembers
	case FocusAttributes:
		return l.Attributes
	case FocusEquipment:
		return l.EquipmentSlots
	case FocusTabs:
		return l.Tabs
	case FocusClose:
		if l.HasClose {
			return 1
		}
		return 0
	}
	return 0
}

// firstGroup returns the first non-empty group in the tab order; FocusNone
// when the layout has no focusable groups at all.
func (l FocusLayout) firstGroup() FocusGroup {
	for _, g := range tabOrder {
		if l.size(g) > 0 {
			return g
		}
	}
	return FocusNone
}

// lastGroup returns the last non-empty group in the tab order; FocusNone
// when the layout has no focusable groups at all.
func (l FocusLayout) lastGroup() FocusGroup {
	for i := len(tabOrder) - 1; i >= 0; i-- {
		if l.size(tabOrder[i]) > 0 {
			return tabOrder[i]
		}
	}
	return FocusNone
}

// nextGroup returns the non-empty group after g in the tab order, wrapping
// to the first; FocusNone when the layout has no other focusable group (a
// lone group wraps to itself).
func (l FocusLayout) nextGroup(g FocusGroup) FocusGroup {
	start := -1
	for i, gg := range tabOrder {
		if gg == g {
			start = i
			break
		}
	}
	if start < 0 {
		return l.firstGroup()
	}
	for i := 1; i <= len(tabOrder); i++ {
		gg := tabOrder[(start+i)%len(tabOrder)]
		if l.size(gg) > 0 {
			return gg
		}
	}
	return FocusNone
}

// prevGroup returns the non-empty group before g in the tab order,
// wrapping to the last; FocusNone when the layout has no other focusable
// group (a lone group wraps to itself).
func (l FocusLayout) prevGroup(g FocusGroup) FocusGroup {
	start := -1
	for i, gg := range tabOrder {
		if gg == g {
			start = i
			break
		}
	}
	if start < 0 {
		return l.lastGroup()
	}
	for i := 1; i <= len(tabOrder); i++ {
		gg := tabOrder[(start-i+len(tabOrder))%len(tabOrder)]
		if l.size(gg) > 0 {
			return gg
		}
	}
	return FocusNone
}

// ArrowDir is one arrow-key direction for in-group navigation (AC-23).
type ArrowDir int

const (
	// ArrowUp: ↑ — moves the cursor up in the sidebar and attribute groups.
	ArrowUp ArrowDir = iota
	// ArrowDown: ↓ — moves the cursor down in the sidebar and attribute
	// groups.
	ArrowDown
	// ArrowLeft: ← — moves the cursor left in the equipment and tab groups.
	ArrowLeft
	// ArrowRight: → — moves the cursor right in the equipment and tab
	// groups.
	ArrowRight
)

// KeyboardNav is the C&I UI's keyboard focus cursor (pure Go, no engine
// dependency): the focused group and the index within it. Tab/Shift+Tab
// move between groups along the AC-23 sequence with wrap-around; the
// arrow keys move within the focused group — ↑/↓ cycling the sidebar
// portraits (GDD "cycle members", wrapping), ↑/↓ stepping the attribute
// bars (clamped), ←/→ stepping the equipment slots and the tabs
// (clamped). The zero value is ready to use (nothing focused, empty
// layout).
type KeyboardNav struct {
	layout FocusLayout
	focus  FocusState
}

// FocusState is one cursor position: the focused group and the index
// within it.
type FocusState struct {
	group FocusGroup
	index int
}

// Reset clears the cursor: nothing is focused and the next Tab enters the
// first group.
func (n *KeyboardNav) Reset() {
	n.focus.group = FocusNone
	n.focus.index = 0
}

// SetLayout refreshes the tab-order layout and clamps the cursor into it:
// an index past a shrunken group's end moves to its last element, and a
// group that became empty clears the cursor to FocusNone. The sheet calls
// it before every dispatch so roster changes are always reflected.
func (n *KeyboardNav) SetLayout(l FocusLayout) {
	n.layout = l
	if n.focus.group == FocusNone {
		return
	}
	if n.layout.size(n.focus.group) == 0 {
		n.focus.group = FocusNone
		n.focus.index = 0
		return
	}
	if size := n.layout.size(n.focus.group); n.focus.index >= size {
		n.focus.index = size - 1
	}
}

// Group returns the focused group; FocusNone when nothing is focused.
func (n *KeyboardNav) Group() FocusGroup { return n.focus.group }

// Index returns the cursor's index within the focused group; 0 when
// nothing is focused.
func (n *KeyboardNav) Index() int { return n.focus.index }

// Tab moves the cursor to the next group in the tab order (AC-23); with
// shift it moves to the previous group. The order wraps at both ends:
// Tab on the close button returns to the sidebar and Shift+Tab on the
// sidebar returns to the close button. Entering a group places the cursor
// on its first element (its last element with shift). Empty groups are
// skipped. From FocusNone, Tab enters the first group (last with shift).
func (n *KeyboardNav) Tab(shift bool) {
	var g FocusGroup
	if n.focus.group == FocusNone || n.layout.size(n.focus.group) == 0 {
		if shift {
			g = n.layout.lastGroup()
		} else {
			g = n.layout.firstGroup()
		}
	} else if shift {
		g = n.layout.prevGroup(n.focus.group)
	} else {
		g = n.layout.nextGroup(n.focus.group)
	}
	n.focus.group = g
	if g == FocusNone {
		n.focus.index = 0
		return
	}
	if shift {
		n.focus.index = n.layout.size(g) - 1
	} else {
		n.focus.index = 0
	}
}

// Arrow moves the cursor within the focused group (AC-23): ↑/↓ in the
// sidebar cycle the portraits with wrap-around (GDD: "↑/↓ to cycle
// members") and step the attribute bars with clamping; ←/→ step the
// equipment slots and the tabs with clamping. Arrows that do not match
// the focused group's axis are ignored, as are arrows with nothing (or
// only one element) focused.
func (n *KeyboardNav) Arrow(dir ArrowDir) {
	size := n.layout.size(n.focus.group)
	if size <= 1 {
		return
	}
	switch n.focus.group {
	case FocusSidebar:
		switch dir {
		case ArrowUp:
			n.focus.index = wrapIndex(n.focus.index-1, size)
		case ArrowDown:
			n.focus.index = wrapIndex(n.focus.index+1, size)
		}
	case FocusAttributes:
		switch dir {
		case ArrowUp:
			n.focus.index = clampIndex(n.focus.index-1, size)
		case ArrowDown:
			n.focus.index = clampIndex(n.focus.index+1, size)
		}
	case FocusEquipment, FocusTabs:
		switch dir {
		case ArrowLeft:
			n.focus.index = clampIndex(n.focus.index-1, size)
		case ArrowRight:
			n.focus.index = clampIndex(n.focus.index+1, size)
		}
	}
}

// wrapIndex cycles i within [0, size): with size 2, -1 wraps to 1 and 2
// wraps to 0.
func wrapIndex(i, size int) int {
	i %= size
	if i < 0 {
		i += size
	}
	return i
}

// clampIndex pins i to [0, size-1].
func clampIndex(i, size int) int {
	if i < 0 {
		return 0
	}
	if i >= size {
		return size - 1
	}
	return i
}

// Minimum resolution (TR-ui-006, ADR-0012): the C&I UI supports a
// 1280×720 minimum and a 1920×1080 reference with height-based scaling.
const (
	// MinResolutionWidth is the minimum supported window width.
	MinResolutionWidth = 1280
	// MinResolutionHeight is the minimum supported window height.
	MinResolutionHeight = 720
	// MinResolutionLockMessage is the AC-25 lock line shown while the
	// window is below the minimum.
	MinResolutionLockMessage = "Minimum resolution: 1280×720"
)

// ResolutionGuard checks window dimensions against the 1280×720 minimum
// (AC-25, TR-ui-006): blocked is true only BELOW the minimum — exactly
// 1280×720 is the minimum and is NOT blocked. message is the lock line
// shown while blocked (MinResolutionLockMessage); "" when not blocked.
// The sheet polls it each frame; callers (e.g. the settings screen) can
// use it to pre-check window sizes.
func ResolutionGuard(width, height int) (blocked bool, message string) {
	if width < MinResolutionWidth || height < MinResolutionHeight {
		return true, MinResolutionLockMessage
	}
	return false, ""
}

// FocusRingClass is the CSS class of the keyboard focus ring (ADR-0012):
// a 3px oxblood ring on parchment — 6.4:1 contrast, comfortably ≥3:1 —
// applied to the focused element. The ring is a visible outline shape
// around the element, so focus is never encoded by color alone (control
// manifest forbidden pattern).
const FocusRingClass = "focus-ring"

// setFocusRing toggles the focus ring class on an element, preserving its
// existing classes (SetClasses replaces the whole class list, so the ring
// is merged with the element's base classes; a class list refresh by
// another owner simply drops the ring until the next dispatch re-applies
// it).
func setFocusRing(elm *document.Element, on bool) {
	if elm == nil {
		return
	}
	classes := elm.ClassList()
	has := false
	kept := make([]string, 0, len(classes)+1)
	for _, c := range classes {
		if c == FocusRingClass {
			has = true
			continue
		}
		if c != "" {
			kept = append(kept, c)
		}
	}
	if on && has || !on && !has {
		return // already in the requested state
	}
	if on {
		kept = append(kept, FocusRingClass)
	}
	elm.SetClasses(kept...)
}

// Keyboard dispatch (AC-23, AC-25) -------------------------------------------

// updateKeyboardNav polls the keyboard while the sheet is open and
// dispatches to the focused element (AC-23): Tab/Shift+Tab move the focus
// cursor along the tab order, the arrow keys move within the focused
// group, Enter/Space activate the focused element, and Esc requests the
// user close — the same path as the close button (CloseRequested). The
// dispatch is edge-triggered (KeyDown, no held-key repeat) like the state
// machine's C/I/Esc polling, and is gated while the level-up modal is
// active (only Confirm dismisses it, AC-18) and while the resolution lock
// is shown (AC-25: interaction blocked below the minimum). The state
// machine's own C/I/Esc polling (Story 007) continues to run alongside —
// its Esc close and this Esc close land on the same user-close path, so
// the second call in a frame is a no-op.
//
// Like the state machine's C/I polling, the dispatch is global to the
// sheet: text typed into the inventory search input coexists with it
// (pre-existing pattern — pressing C or I in the search box already
// toggles the sheet today).
func (s *CharacterSheet) updateKeyboardNav() {
	if !s.visible || s.doc == nil || s.host == nil || s.host.Window == nil {
		return
	}
	if s.resolutionLocked() {
		return // AC-25: interaction blocked below the minimum
	}
	if s.modal != nil && s.modal.Active() {
		return // the level-up modal is mandatory; only Confirm dismisses it (AC-18)
	}
	kb := &s.host.Window.Keyboard
	s.nav.SetLayout(s.focusLayout())
	shift := kb.KeyHeld(hid.KeyboardKeyLeftShift) || kb.KeyHeld(hid.KeyboardKeyRightShift)
	switch {
	case kb.KeyDown(hid.KeyboardKeyTab):
		s.nav.Tab(shift)
		s.applyFocus()
	case kb.KeyDown(hid.KeyboardKeyUp):
		s.nav.Arrow(ArrowUp)
		s.applyFocus()
	case kb.KeyDown(hid.KeyboardKeyDown):
		s.nav.Arrow(ArrowDown)
		s.applyFocus()
	case kb.KeyDown(hid.KeyboardKeyLeft):
		s.nav.Arrow(ArrowLeft)
		s.applyFocus()
	case kb.KeyDown(hid.KeyboardKeyRight):
		s.nav.Arrow(ArrowRight)
		s.applyFocus()
	case kb.KeyDown(hid.KeyboardKeyReturn) || kb.KeyDown(hid.KeyboardKeyEnter) ||
		kb.KeyDown(hid.KeyboardKeySpace):
		s.activateFocused()
	case kb.KeyDown(hid.KeyboardKeyEscape):
		s.CloseRequested()
	}
}

// focusLayout assembles the tab-order layout from the current UI state:
// the active roster size (pool members are unselectable), the six
// attribute bars, the eight equipment slots, the two tabs, and the close
// button when the markup carries one.
func (s *CharacterSheet) focusLayout() FocusLayout {
	members := 0
	if s.sidebar != nil {
		members = len(s.sidebar.Order())
	}
	return FocusLayout{
		SidebarMembers: members,
		Attributes:     stats.NumAttributes,
		EquipmentSlots: int(NumSlots),
		Tabs:           2,
		HasClose:       s.closeButton != nil,
	}
}

// sidebarMemberAt returns the entity ID of the active party member at the
// given roster index (party order, player first); 0 when the index is out
// of range or the sidebar is missing.
func (s *CharacterSheet) sidebarMemberAt(i int) entity.GameEntityID {
	if s.sidebar == nil {
		return 0
	}
	order := s.sidebar.Order()
	if i < 0 || i >= len(order) {
		return 0
	}
	return order[i]
}

// applyFocus renders the focus cursor after a navigation key: the oxblood
// ring lands on the focused element, and the previous ring (element or
// sidebar) is cleared first. The sidebar renders its own ring through
// party-portrait-focused (SetFocus/FocusTo); every other group uses the
// shared focus-ring class. FocusNone clears everything.
func (s *CharacterSheet) applyFocus() {
	if s.doc == nil {
		return
	}
	if s.focusElm != nil {
		setFocusRing(s.focusElm, false)
		s.focusElm = nil
	}
	g := s.nav.Group()
	if g == FocusNone {
		if s.sidebar != nil {
			s.sidebar.SetFocus(false)
		}
		return
	}
	if g == FocusSidebar {
		// The sidebar owns its ring; hand focus over and drop any element
		// ring (already cleared above).
		if s.sidebar != nil {
			s.sidebar.SetFocus(true)
			s.sidebar.FocusTo(s.sidebarMemberAt(s.nav.Index()))
		}
		return
	}
	// Any other group: the sidebar ring clears with focus.
	if s.sidebar != nil {
		s.sidebar.SetFocus(false)
	}
	switch g {
	case FocusAttributes:
		if i := s.nav.Index(); i >= 0 && i < stats.NumAttributes && s.attrRows[i] != nil {
			s.focusElm = s.attrRows[i]
			setFocusRing(s.focusElm, true)
		}
	case FocusEquipment:
		if i := s.nav.Index(); i >= 0 && i < int(NumSlots) && s.equipCells[i] != nil {
			s.focusElm = s.equipCells[i]
			setFocusRing(s.focusElm, true)
		}
	case FocusTabs:
		elm := s.tabCharacter
		if s.nav.Index() == 1 {
			elm = s.tabInventory
		}
		if elm != nil {
			s.focusElm = elm
			setFocusRing(s.focusElm, true)
		}
	case FocusClose:
		if s.closeButton != nil {
			s.focusElm = s.closeButton
			setFocusRing(s.focusElm, true)
		}
	}
}

// resetFocus clears the focus cursor and every focus ring. Called on Open
// (the tab order always starts from the sidebar) and on Close/ForceClose
// (a stale ring must never survive across opens).
func (s *CharacterSheet) resetFocus() {
	s.nav.Reset()
	s.applyFocus()
}

// activateFocused runs the Enter/Space activation of the focused element
// (AC-23): selecting the focused party member (sidebar), switching to the
// focused tab, or closing via the close button. The attribute breakdown
// (Story 004) and keyboard equip/unequip are not implemented — the
// attributes and equipment groups have nothing to activate yet. Tab
// activation uses the same tab chrome as the click handler (showTab,
// Story 005 pattern); the C/I keys route through the state machine
// (Story 007).
func (s *CharacterSheet) activateFocused() {
	switch s.nav.Group() {
	case FocusSidebar:
		if s.sidebar != nil {
			if id := s.sidebarMemberAt(s.nav.Index()); id != 0 {
				s.sidebar.SelectMember(id)
			}
		}
	case FocusTabs:
		tab := "character"
		if s.nav.Index() == 1 {
			tab = "inventory"
		}
		s.showTab(tab)
	case FocusClose:
		s.CloseRequested()
	}
}

// SetCloseHandler wires the sheet's user-close path — the title-bar close
// button and the keyboard Esc dispatch — to the owner of the screen
// state. The game layer passes the state machine's user close (so the
// level-up modal refusal, AC-18, and the animated exit hold). Without a
// handler the sheet closes directly (headless tests, degraded wiring).
func (s *CharacterSheet) SetCloseHandler(fn func()) { s.closeHandler = fn }

// CloseRequested is the close button / Esc activation (AC-23): it
// delegates to the close handler wired by SetCloseHandler, falling back
// to a direct Close when none is wired.
func (s *CharacterSheet) CloseRequested() {
	if s.closeHandler != nil {
		s.closeHandler()
		return
	}
	s.Close()
}

// Resolution floor (AC-25) ----------------------------------------------------

// resolutionLocked reports whether the window is below the 1280×720
// minimum (AC-25). A sheet without a window (headless) is never locked.
func (s *CharacterSheet) resolutionLocked() bool {
	if s.host == nil || s.host.Window == nil {
		return false
	}
	blocked, _ := ResolutionGuard(s.host.Window.Width(), s.host.Window.Height())
	return blocked
}

// updateResolutionLock enforces the resolution floor each frame (AC-25):
// while the sheet is open and the window is below 1280×720, the lock
// overlay is shown — it covers the parchment and swallows every click in
// its bounds (Kaiju has no pointer-events; panels block clicks in their
// bounds, toast.go note) — and the keyboard dispatch is gated by
// resolutionLocked. At or above the minimum the overlay hides and
// interaction resumes in the same frame. The UI never auto-closes while
// locked; it stays open and locked until the window is resized back up.
func (s *CharacterSheet) updateResolutionLock() {
	if s.doc == nil || s.lockOverlay == nil {
		return
	}
	blocked := s.visible && s.resolutionLocked()
	if blocked == s.lockShown {
		return
	}
	s.lockShown = blocked
	display := "none"
	if blocked {
		display = "flex"
	}
	s.doc.SetElementStyleProperty(s.lockOverlay, "display", display)
	if blocked {
		s.resetFocus() // interaction is blocked; drop the focus cursor
	}
}

// accessibilityFuncMap returns the Story 009 markup handlers, dispatched
// through the markup loader's funcMap (title screen pattern). Markup
// contract:
//
//	onclick="sheetClose" — the title-bar close button (AC-23's final tab stop)
func accessibilityFuncMap(s *CharacterSheet) map[string]func(*document.Element) {
	return map[string]func(*document.Element){
		"sheetClose": func(*document.Element) {
			s.CloseRequested()
		},
	}
}

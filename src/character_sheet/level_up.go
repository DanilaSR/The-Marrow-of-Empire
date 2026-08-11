// Level-up modal (Story 002, Epic: Character & Inventory UI): the mandatory
// level-up overlay on the character sheet (GDD screen state LevelUp,
// design/gdd/character-inventory-ui.md).
//
// The modal is a Desk-register overlay (ADR-0012) rendered inside the
// character sheet's markup document and driven entirely from Go: Show
// displays a level-up, AllocatePoint spends the milestone attribute point,
// and Confirm commits the allocation and closes the modal. It cannot be
// dismissed any other way — there is no cancel button, and Esc or clicking
// outside never closes it (AC-18). Level-ups that arrive while a modal is
// active queue in party order (player character first, then companions by
// join order — GDD Edge Cases).
//
// State machine (explicit transition table):
//
//	LevelUpHidden ──Show()──────────────▶ LevelUpShowing ──AllocatePoint()──▶ LevelUpWaitingConfirm ──Confirm()──▶ LevelUpHidden
//	      │                                    │                                  │
//	      │                                    └────── no attribute point ────────┘
//	      └── Show() while active: level-up appended to the queue (party order)
//	      └── Confirm() on a hidden modal: error
//	      └── Confirm() while Showing (point not yet allocated): error
//
// The Stats component is read from the entity store lazily: the display
// values at Show time, and a fresh copy at AllocatePoint time. The re-read
// matters because the XP awarder writes the post-award component back to
// the store only after the synchronous EventStatLevelUp dispatch returns
// (stats.AwardXPAndNotify usage pattern) — an interaction-time read is
// always post-award, so Confirm can never overwrite an un-persisted award.
//
// The modal's Update (pulse) is driven per frame from Go: Kaiju's CSS
// engine supports :hover selectors but has no transitions or animations, so
// the amber "pulse on hover" of allocatable attribute bars is a sine
// opacity oscillation applied to the hovered row's fill (story
// 002 Implementation Notes). Keyboard navigation inside the modal is Story
// 009 (out of scope here); the sheet's forced-close policy for an active
// modal is owned by the screen state machine (Story 007).
//
// Story: production/epics/character-inventory-ui/
// story-002-xp-progress-level-up.md
package character_sheet

import (
	"fmt"
	"log/slog"
	"math"
	"sort"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// LevelUpState is the state of the level-up modal (GDD screen state
// LevelUp). The screen state machine (Story 007) reads it via State.
type LevelUpState int

const (
	// LevelUpHidden: no level-up modal displayed.
	LevelUpHidden LevelUpState = iota
	// LevelUpShowing: the modal is displayed and, for milestone level-ups
	// (4/8/12/16), waits for the attribute point to be allocated — Confirm
	// is disabled until AllocatePoint succeeds.
	LevelUpShowing
	// LevelUpWaitingConfirm: all required input is given; Confirm is
	// enabled and is the only way to close the modal.
	LevelUpWaitingConfirm
)

// attrRowIDs are the lowercase data-attr values of the six modal attribute
// rows in canonical order [STR, DEX, CON, INT, WIS, CHA], matching the
// stats.Attributes indices. Each row's fill, value label, and + button are
// "{id}-fill", "{id}-value", and "{id}-add" with ids "lu-attr-{row}".
var attrRowIDs = [stats.NumAttributes]string{
	"str", "dex", "con", "int", "wis", "cha",
}

// PartyRoster supplies the party member order used to queue simultaneous
// level-up modals (GDD Edge Cases: player character first, then companions
// by join order). The character sheet's party sidebar (Story 008) implements
// this interface; without one the queue preserves arrival order.
type PartyRoster interface {
	// Order returns the party member entity IDs in party order.
	Order() []entity.GameEntityID
}

// pendingLevelUp is one queued level-up display: the character and the
// level-up facts needed to render the modal. component is the working Stats
// component copy: read at Show time for the display values and replaced
// with a fresh post-award copy at AllocatePoint time.
type pendingLevelUp struct {
	entityID          entity.GameEntityID
	prevLevel         int
	newLevel          int
	hpGain            int
	hasAttributePoint bool
	component         stats.StatsComponent
	allocated         bool
}

// LevelUpModal is the level-up overlay on the character sheet. It owns the
// modal's markup elements, the queue of pending level-ups, and the state
// machine above. It reads and writes Stats components through the entity
// store (ADR-0001: UI reads entity data; the modal's Confirm writes the
// attribute point allocation back per the GDD).
//
// The modal degrades to a pure state machine when constructed with a nil
// document (unit tests): every markup access is nil-guarded.
type LevelUpModal struct {
	doc    *document.Document
	store  entity.EntityStore
	roster PartyRoster

	state   LevelUpState
	current *pendingLevelUp
	queue   []*pendingLevelUp

	// Elements bound from the markup (character_sheet.html).
	root        *document.Element              // level-up-modal
	headerLabel *ui.Label                      // lu-header
	hpLabel     *ui.Label                      // lu-hp
	attrValues  [stats.NumAttributes]*ui.Label // lu-attr-{row}-value
	attrFills   [stats.NumAttributes]*document.Element
	attrAdds    [stats.NumAttributes]*document.Element
	confirm     *document.Element // lu-confirm

	// hovered is the attribute row under the pointer, -1 when none; the
	// row's amber fill pulses while hovered and a point is pending.
	hovered   int
	pulseTime float64
	visible   bool

	// reducedMotion disables the hover pulse (Story 009 wires the
	// accessibility setting here, mirroring CharacterSheet).
	reducedMotion bool
}

// NewLevelUpModal creates the level-up modal bound to the character sheet's
// markup document. The modal is hidden until Show is called. roster may be
// nil — the queue then preserves arrival order (party order arrives with
// the Party System, Story 008). A nil doc is valid for unit tests: the
// modal then runs as a pure state machine with no markup access.
func NewLevelUpModal(doc *document.Document, store entity.EntityStore, roster PartyRoster) *LevelUpModal {
	m := &LevelUpModal{
		doc:     doc,
		store:   store,
		roster:  roster,
		state:   LevelUpHidden,
		hovered: -1,
	}
	m.bindElements()
	return m
}

// Show displays a level-up for the given character: the header "Level Up —
// Level [N] → [N+1]" (actual levels — multi-level awards render their true
// range), the health increase "+N HP", and — for milestone level-ups
// (every 4th level: 4, 8, 12, 16 — hasAttributePoint) — the six attribute
// options with a + button each, Confirm disabled until one is allocated.
//
// When another level-up modal is already active the level-up is queued and
// displayed after the active modal is confirmed, in party order when a
// PartyRoster is injected (the pending queue is stable-sorted by roster
// position; characters outside the roster keep arrival order after party
// members).
//
// prevLevel/newLevel/hpGain/hasAttributePoint come from the caller (the
// EventStatLevelUp subscriber, Story 007): prevLevel and hpGain are derived
// from the pre-award component the store still holds during the synchronous
// dispatch, and hasAttributePoint from the milestone levels the award
// crossed. Show may be called from within the dispatch — the modal reads
// the store only at interaction time, after the awarder has written the
// post-award component back.
func (m *LevelUpModal) Show(entityID entity.GameEntityID, prevLevel, newLevel, hpGain int, hasAttributePoint bool) {
	rec := &pendingLevelUp{
		entityID:          entityID,
		prevLevel:         prevLevel,
		newLevel:          newLevel,
		hpGain:            hpGain,
		hasAttributePoint: hasAttributePoint,
	}
	if m.state != LevelUpHidden {
		m.queue = append(m.queue, rec)
		m.sortQueueByPartyOrder()
		return
	}
	m.display(rec)
}

// AllocatePoint spends the level-up's attribute point on the given
// attribute (GDD: six attribute options, one point per milestone level-up)
// and enables Confirm. It reads a fresh Stats component from the entity
// store — by interaction time the XP awarder has written the post-award
// component back, so the allocation targets the character's real state.
// The underlying stats.AllocateAttributePoint rules apply: it errors when
// the modal is not awaiting an allocation, no points remain, the attribute
// index is out of range, or the attribute is already at the mortal maximum
// of 18. A successful allocation is final — a second call errors.
func (m *LevelUpModal) AllocatePoint(attr int) error {
	if m.state != LevelUpShowing {
		return fmt.Errorf("character_sheet: no level-up awaiting allocation")
	}
	rec := m.current
	comp, ok := entity.GetComponent[stats.StatsComponent](m.store, rec.entityID)
	if !ok {
		return fmt.Errorf("character_sheet: entity %d has no Stats component", rec.entityID)
	}
	if err := comp.AllocateAttributePoint(attr); err != nil {
		return err
	}
	rec.component = comp
	rec.allocated = true
	m.state = LevelUpWaitingConfirm
	m.render()
	return nil
}

// Confirm commits the level-up and closes the modal: it writes the
// attribute point allocation (if any) back to the entity's Stats component
// via the entity store, hides the overlay, and displays the next queued
// level-up (if any). This is the only way to dismiss the modal (AC-18) —
// there is no cancel button and the overlay never dismisses itself.
//
// The level-up itself (Level/XP changes) was already committed by the XP
// awarder (stats.AwardXP); Confirm only persists the allocation, so a
// level-up without an attribute point writes nothing. It errors when no
// modal is active or when a milestone point has not been allocated yet.
func (m *LevelUpModal) Confirm() error {
	switch m.state {
	case LevelUpHidden:
		return fmt.Errorf("character_sheet: no active level-up modal")
	case LevelUpShowing:
		return fmt.Errorf("character_sheet: attribute point must be allocated before confirming")
	}
	rec := m.current
	if rec.allocated {
		if err := entity.SetComponent(m.store, rec.entityID, rec.component); err != nil {
			return fmt.Errorf("character_sheet: failed to write allocation for entity %d: %w", rec.entityID, err)
		}
	}
	m.setVisible(false)
	m.current = nil
	m.state = LevelUpHidden
	m.hovered = -1
	if len(m.queue) > 0 {
		next := m.queue[0]
		m.queue = m.queue[1:]
		m.display(next)
	}
	return nil
}

// State returns the modal's current state (LevelUpHidden, LevelUpShowing,
// or LevelUpWaitingConfirm). The screen state machine (Story 007) uses it
// to track the LevelUp screen state.
func (m *LevelUpModal) State() LevelUpState { return m.state }

// Active reports whether a level-up modal is currently displayed. A modal
// once shown stays active until Confirm — it cannot be dismissed any other
// way (AC-18).
func (m *LevelUpModal) Active() bool { return m.state != LevelUpHidden }

// CurrentEntityID returns the entity whose level-up is currently displayed,
// or zero when no modal is active. The party sidebar (Story 008) uses it to
// decide whether a departing companion is showing a level-up modal.
func (m *LevelUpModal) CurrentEntityID() entity.GameEntityID {
	if m.current == nil {
		return 0
	}
	return m.current.entityID
}

// Abort discards the active level-up modal and the pending queue without
// writing anything to the entity store — the milestone attribute point (if
// any) is forfeited and the XP award already committed stands. This is the
// QA edge of Story 008: when the viewed companion departs the party (AC-19),
// the sheet closes the modal for that companion via Abort, because
// Confirm's allocation write would target a character that is no longer in
// the party. The player's own level-up modal is never aborted — the player
// character cannot depart.
func (m *LevelUpModal) Abort() {
	m.current = nil
	m.queue = nil
	m.state = LevelUpHidden
	m.hovered = -1
	m.setVisible(false)
}

// SetReducedMotion disables the hover pulse: when on, hovered attribute
// bars hold a steady amber fill instead of pulsing. Story 009 wires the
// accessibility setting here, mirroring CharacterSheet.
func (m *LevelUpModal) SetReducedMotion(on bool) { m.reducedMotion = on }

// display shows the given level-up: it reads the entity's Stats component
// for the attribute row values (attributes are unchanged by the award, so
// a pre-award read during event dispatch is correct for display), renders
// the modal, and enters LevelUpShowing — or LevelUpWaitingConfirm directly
// when no attribute point is required.
func (m *LevelUpModal) display(rec *pendingLevelUp) {
	comp, ok := entity.GetComponent[stats.StatsComponent](m.store, rec.entityID)
	if !ok {
		slog.Warn("level up modal: entity has no Stats component; level-up skipped",
			"entity", rec.entityID)
		return
	}
	rec.component = comp
	m.current = rec
	m.state = LevelUpShowing
	if !rec.hasAttributePoint {
		m.state = LevelUpWaitingConfirm
	}
	m.hovered = -1
	m.render()
	m.setVisible(true)
}

// render writes the current level-up into the bound markup elements:
// header, HP gain, six attribute rows (value with modifier, proportional
// amber fill, + button enabled while a point is still to be allocated and
// the attribute is below 18), and the Confirm button state.
func (m *LevelUpModal) render() {
	if m.doc == nil || m.current == nil {
		return
	}
	rec := m.current
	if m.headerLabel != nil {
		m.headerLabel.SetText(LevelUpHeader(rec.prevLevel, rec.newLevel))
	}
	if m.hpLabel != nil {
		m.hpLabel.SetText(fmt.Sprintf("+%d HP", rec.hpGain))
	}
	for i := 0; i < stats.NumAttributes; i++ {
		if l := m.attrValues[i]; l != nil {
			l.SetText(AttributeDisplay(rec.component.Attributes[i]))
		}
		if fill := m.attrFills[i]; fill != nil {
			m.doc.SetElementStyleProperty(fill, "width",
				fmt.Sprintf("%.1f%%", BarFillFraction(rec.component.Attributes[i])*100))
		}
		allocatable := rec.hasAttributePoint && !rec.allocated &&
			rec.component.Attributes[i] < maxAttributeValue
		m.setAddEnabled(i, allocatable)
	}
	m.applyConfirmState()
}

// setAddEnabled toggles the + button's disabled styling. Clicks on a
// disabled button are still rejected by AllocatePoint's state guard.
func (m *LevelUpModal) setAddEnabled(attr int, enabled bool) {
	elm := m.attrAdds[attr]
	if elm == nil {
		return
	}
	if enabled {
		elm.SetClasses("lu-add")
	} else {
		elm.SetClasses("lu-add", "lu-add-disabled")
	}
}

// applyConfirmState styles the Confirm button: enabled (ink border) in
// LevelUpWaitingConfirm, disabled (subdued) otherwise. Clicks are always
// dispatched; Confirm's state guard rejects premature confirms.
func (m *LevelUpModal) applyConfirmState() {
	if m.confirm == nil {
		return
	}
	if m.state == LevelUpWaitingConfirm {
		m.confirm.SetClasses("lu-confirm")
	} else {
		m.confirm.SetClasses("lu-confirm", "lu-confirm-disabled")
	}
}

// setVisible shows or hides the modal overlay via the root's inline display
// property (hidden by default in the markup).
func (m *LevelUpModal) setVisible(visible bool) {
	if m.doc == nil || m.root == nil {
		return
	}
	if visible {
		m.doc.SetElementStyleProperty(m.root, "display", "flex")
	} else {
		m.doc.SetElementStyleProperty(m.root, "display", "none")
	}
	m.visible = visible
}

// setHover marks the attribute row under the pointer (attr index, or -1 for
// none) and settles the other rows' fills to full opacity.
func (m *LevelUpModal) setHover(attr int) {
	if attr < -1 || attr >= stats.NumAttributes || attr == m.hovered {
		return
	}
	m.hovered = attr
	m.pulseTime = 0
	if m.doc == nil {
		return
	}
	for i := 0; i < stats.NumAttributes; i++ {
		if i == attr {
			continue
		}
		if fill := m.attrFills[i]; fill != nil {
			m.doc.SetElementStyleProperty(fill, "opacity", "1")
		}
	}
}

// update advances the hover pulse by dt seconds: while the modal is shown
// and an allocatable attribute row is hovered, the row's amber fill opacity
// oscillates on a 1 Hz sine (range 0.4-1.0) — Kaiju CSS has no
// transitions/animations, so the pulse is driven from Go. Skipped under
// reduced motion and once the point is allocated (WaitingForConfirm).
func (m *LevelUpModal) update(dt float64) {
	if m.state != LevelUpShowing || m.hovered < 0 || m.reducedMotion || m.doc == nil {
		return
	}
	m.pulseTime += dt
	fill := m.attrFills[m.hovered]
	if fill == nil {
		return
	}
	op := 0.7 + 0.3*math.Sin(m.pulseTime*2*math.Pi)
	m.doc.SetElementStyleProperty(fill, "opacity", fmt.Sprintf("%.3f", op))
}

// sortQueueByPartyOrder stable-sorts the pending queue by roster position
// (player first, then companions by join order). Characters outside the
// roster keep their arrival order after party members. Without a roster the
// queue preserves arrival order.
func (m *LevelUpModal) sortQueueByPartyOrder() {
	if m.roster == nil || len(m.queue) < 2 {
		return
	}
	idx := make(map[entity.GameEntityID]int, len(m.queue))
	for i, id := range m.roster.Order() {
		idx[id] = i
	}
	sort.SliceStable(m.queue, func(i, j int) bool {
		pi, oki := idx[m.queue[i].entityID]
		pj, okj := idx[m.queue[j].entityID]
		if oki && okj {
			return pi < pj
		}
		return oki && !okj
	})
}

// bindElements resolves the modal markup ids into element references.
// Missing elements are logged and left nil; the update path skips nil
// elements. A nil doc (unit tests) binds nothing.
func (m *LevelUpModal) bindElements() {
	if m.doc == nil {
		return
	}
	m.root = sheetElement(m.doc, "level-up-modal")
	m.headerLabel = sheetLabel(m.doc, "lu-header")
	m.hpLabel = sheetLabel(m.doc, "lu-hp")
	for i, row := range attrRowIDs {
		prefix := "lu-attr-" + row
		m.attrValues[i] = sheetLabel(m.doc, prefix+"-value")
		m.attrFills[i] = sheetElement(m.doc, prefix+"-fill")
		m.attrAdds[i] = sheetElement(m.doc, prefix+"-add")
	}
	m.confirm = sheetElement(m.doc, "lu-confirm")
}

// levelUpFuncMap returns the onclick/onmouseenter/onmouseleave handler map
// for the level-up modal markup (character_sheet.html), dispatched through
// the markup loader's funcMap (title screen pattern). The handlers dispatch
// to the sheet's modal, which is created after the document loads — the
// closures read s.modal lazily and no-op while it is nil. Markup contract:
//
//	onclick="levelUpConfirm"          — Confirm button
//	onclick="levelUpAlloc"            — + button, data-attr="{row}"
//	onmouseenter="levelUpHover"       — attribute row, data-attr="{row}"
//	onmouseleave="levelUpHoverNone"   — attribute row
func levelUpFuncMap(s *CharacterSheet) map[string]func(*document.Element) {
	return map[string]func(*document.Element){
		"levelUpConfirm": func(*document.Element) {
			if m := s.modal; m != nil {
				if err := m.Confirm(); err != nil {
					slog.Warn("level up modal: confirm rejected", "err", err)
				}
			}
		},
		"levelUpAlloc": func(elm *document.Element) {
			if m := s.modal; m != nil {
				attr, ok := levelUpAttrFromData(elm)
				if !ok {
					return
				}
				if err := m.AllocatePoint(attr); err != nil {
					slog.Warn("level up modal: allocation rejected", "attr", attr, "err", err)
				}
			}
		},
		"levelUpHover": func(elm *document.Element) {
			if m := s.modal; m != nil {
				if attr, ok := levelUpAttrFromData(elm); ok {
					m.setHover(attr)
				}
			}
		},
		"levelUpHoverNone": func(*document.Element) {
			if m := s.modal; m != nil {
				m.setHover(-1)
			}
		},
	}
}

// levelUpAttrFromData maps a row's data-attr value to a stats attribute
// index. Unknown values are logged and rejected.
func levelUpAttrFromData(elm *document.Element) (int, bool) {
	name := elm.Attribute("data-attr")
	for i, row := range attrRowIDs {
		if row == name {
			return i, true
		}
	}
	slog.Warn("level up modal: unknown data-attr", "attr", name)
	return 0, false
}

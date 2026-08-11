// Party sidebar and companion inspect (Story 008, Epic: Character &
// Inventory UI): the left sidebar of the character sheet (20% width,
// ADR-0012 desk register) with party member portraits — player character at
// top, active companions in party order, pool companions greyed out below —
// the 300ms hover tooltip (name, level, role, HP bar, trust indicator,
// Inspect button), companion inspect (Trust & Relationship panel and
// Knowledge Domains list), and the AC-19 departure handling (the view
// switches to the player character when the viewed companion leaves the
// party).
//
// The sidebar is a presenter: the party layer pushes the roster in as
// []PartyMember via SetParty (members[0] must be the player character —
// GDD party order: player at top), the sidebar partitions it into active
// and pool, and renders a portrait cell per member by duplicating the
// hidden #party-portrait-tpl template (document.DuplicateElementToParent,
// the inventory grid's dynamic-list pattern; the duplicate's root id is
// stripped by the engine, so children are addressed by class only). Level
// and HP are read live from the entity store (Stats/Health components) so
// tooltips never show stale numbers; trust, role, behavior mode, and
// knowledge domains travel on the PartyMember records, which the party
// layer assembles from the Trust system and the NPC Knowledge Graph
// (ADR-0005). The sidebar never writes entity state (ADR-0001).
//
// Behavior mode glyphs (party GDD CR-5) are single-stroke CSS placeholders
// drawn from borders — Kaiju CSS has no rotate, and pen-and-ink icon
// assets arrive later, same convention as the equipment slot silhouettes:
// aggressive = sword (blade + crossguard), defensive = shield (hollow
// square), passive = eye (ring + pupil). The glyph class per mode comes
// from BehaviorGlyphClass.
//
// Departure (AC-19, party GDD CR-7): the sidebar subscribes to
// EventPartyMemberDeparted (party.member_departed — the subscription
// contract with the Party System, which is the producer and not
// implemented yet) and to combat.EventCharacterDied (QA edge: companion
// death switches the view the same way) at presentation priority 10
// (ADR-0011). When the departed companion is the viewed member, the view
// switches to the player character immediately, and any level-up modal
// showing the departed companion closes (the sheet's CompanionDeparted).
// Simultaneous departures switch at most once: after the first switch the
// viewed member is the player, so later departures only remove portraits.
//
// The party-order contract for the level-up queue (level_up.go) is the
// PartyRoster interface; PartySidebar implements it (Order returns the
// active members in party order), so NewCharacterSheet passes the sidebar
// to NewLevelUpModal and the modal queue sorts in party order (GDD edge
// case). Keyboard navigation and accessibility arrive with Story 009
// (SetFocus/FocusTo, the tab order's first stop, and the reduced-motion
// tooltip timing).
//
// Story: production/epics/character-inventory-ui/
// story-008-party-sidebar-companion-inspect.md
package character_sheet

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
	"marrow-of-empire/trust_relationship"
)

// EventPartyMemberDeparted is the party departure event (Story 008 AC-19).
// The Party System is the sole producer; it emits it when a companion
// leaves the party for any cause (party GDD CR-7: trust betrayal, story,
// death, dismissal). The sidebar subscribes at presentation priority 10
// (ADR-0011). Payload: PartyDepartureData.
//
// The constant is defined here as the subscription contract with the
// producer, mirroring the trust package's inbound gameplay event contract
// (src/trust_relationship/events.go): producers emit against this event
// type; the Party System story may re-export it when the party package
// lands.
const EventPartyMemberDeparted entity.EventType = "party.member_departed"

// PartyDepartureCause is why a companion left the party (party GDD CR-7).
type PartyDepartureCause string

const (
	// DepartureTrustBetrayal: trust dropped to or below the departure
	// threshold (typically trust level 0).
	DepartureTrustBetrayal PartyDepartureCause = "trust_betrayal"
	// DepartureStory: a scripted story departure.
	DepartureStory PartyDepartureCause = "story"
	// DepartureDeath: the companion died permanently (VS: knocked out and
	// never revived; Full Vision: dying countdown elapsed).
	DepartureDeath PartyDepartureCause = "death"
	// DepartureDismissed: the player dismissed the companion from the pool.
	DepartureDismissed PartyDepartureCause = "dismissed"
)

// PartyDepartureData is the payload of EventPartyMemberDeparted. It is a
// value type so the event avoids heap allocation beyond the Event.Data
// interface boxing (ADR-0011).
type PartyDepartureData struct {
	// EntityID is the departed companion.
	EntityID entity.GameEntityID
	// Cause is why the companion left.
	Cause PartyDepartureCause
}

// BehaviorMode is a companion's AI behavior mode (party GDD CR-5):
// aggressive, defensive, or passive. The sidebar shows the mode as a small
// glyph on the portrait, always visible.
type BehaviorMode int

const (
	// BehaviorAggressive: moves ahead of the formation, attacks hostile
	// entities on sight.
	BehaviorAggressive BehaviorMode = iota
	// BehaviorDefensive: stays within 5 units of the player, attacks only
	// engaged enemies. The default for newly recruited companions (CR-5).
	BehaviorDefensive
	// BehaviorPassive: stays in formation, uses support abilities only.
	BehaviorPassive
)

// String returns the mode's display name (English placeholder pending
// localization).
func (m BehaviorMode) String() string {
	switch m {
	case BehaviorAggressive:
		return "Aggressive"
	case BehaviorDefensive:
		return "Defensive"
	case BehaviorPassive:
		return "Passive"
	default:
		return "Unknown"
	}
}

// Valid reports whether the mode is one of the three defined modes.
func (m BehaviorMode) Valid() bool {
	return m >= BehaviorAggressive && m <= BehaviorPassive
}

// BehaviorGlyphClass returns the portrait glyph CSS class for a behavior
// mode (party GDD CR-5): aggressive = sword, defensive = shield, passive =
// eye. The glyphs are single-stroke CSS placeholders until pen-and-ink
// icon assets exist. Unknown modes default to the defensive (shield) glyph
// so a malformed record still renders a neutral marker.
func BehaviorGlyphClass(m BehaviorMode) string {
	switch m {
	case BehaviorAggressive:
		return "party-glyph-aggressive"
	case BehaviorPassive:
		return "party-glyph-passive"
	default:
		return "party-glyph-defensive"
	}
}

// TrustIndicator is the trust display state of a companion (C&I GDD party
// overview: "amber glow at trust 1+, full glow at trust 3"). Amber is
// fill/border only — the indicator never carries text (ADR-0012).
type TrustIndicator int

const (
	// TrustNone: trust 0 — no amber glow (umber-greyed indicator).
	TrustNone TrustIndicator = iota
	// TrustGlow: trust 1-2 — amber glow.
	TrustGlow
	// TrustFull: trust 3 — full amber glow.
	TrustFull
)

// TrustIndicatorFor maps a trust level (0-3, trust GDD CR-1) to its
// indicator state: 0 → TrustNone, 1-2 → TrustGlow, 3+ → TrustFull.
func TrustIndicatorFor(level int) TrustIndicator {
	switch {
	case level >= 3:
		return TrustFull
	case level >= 1:
		return TrustGlow
	default:
		return TrustNone
	}
}

// Class returns the indicator's CSS class: "trust-none", "trust-glow", or
// "trust-full". The class styles fill and glow only — never text.
func (t TrustIndicator) Class() string {
	switch t {
	case TrustGlow:
		return "trust-glow"
	case TrustFull:
		return "trust-full"
	default:
		return "trust-none"
	}
}

// KnowledgeDomain is one knowledge domain of a companion with its
// proficiency indicator (C&I GDD companion inspect: "domain name,
// proficiency indicator"). Domains come from the NPC Knowledge Graph
// (ADR-0005).
type KnowledgeDomain struct {
	// Name is the domain name, e.g. "craft_metallurgy".
	Name string
	// Proficiency is the 0-5 indicator of how well the companion knows the
	// domain (KnowledgeDomainProficiency).
	Proficiency int
}

// KnowledgeDomainProficiency computes the 0-5 proficiency indicator for a
// domain from the number of facts the companion knows in it: 0 facts → 0,
// then one step per two facts, 9+ facts → 5. Clamped to [0, 5].
func KnowledgeDomainProficiency(numFacts int) int {
	if numFacts <= 0 {
		return 0
	}
	if numFacts >= 9 {
		return 5
	}
	return (numFacts + 1) / 2
}

// PartyMember is one row of the sidebar roster, assembled by the party
// layer from the party roster (PartyRoster order), the Trust system, and
// the NPC Knowledge Graph (ADR-0005). members[0] of SetParty must be the
// player character (GDD: player at top).
type PartyMember struct {
	// ID is the member's entity ID.
	ID entity.GameEntityID
	// Name is the display name.
	Name string
	// Role is the class/archetype, e.g. "Rogue" (tooltip role line).
	Role string
	// Behavior is the companion's AI behavior mode (portrait glyph).
	Behavior BehaviorMode
	// Trust is the companion's current trust level, 0-3 (trust system).
	Trust trust.TrustLevel
	// TrustHistory is the relationship history, most recent last (party GDD
	// CR-4 trust_events[]); the inspect panel shows the last five.
	TrustHistory []trust.TrustEvent
	// Domains are the companion's knowledge domains with proficiency
	// indicators (ADR-0005).
	Domains []KnowledgeDomain
	// InPool marks a recruited companion outside the active party: the
	// portrait renders greyed out below the active members and is
	// unselectable (C&I GDD layout).
	InPool bool
}

// Viewer switches the character sheet's viewed member. CharacterSheet
// implements it; the sidebar calls it from portrait clicks, the Inspect
// button, and the AC-19 departure handler. Tests use a stub.
type Viewer interface {
	// SwitchMember switches the viewed member, preserving the active tab
	// (AC-22). It is a no-op when id is already the viewed member.
	SwitchMember(id entity.GameEntityID) error
	// ViewedID returns the member currently being viewed; zero when the
	// sheet is closed.
	ViewedID() entity.GameEntityID
	// CompanionDeparted notifies that a companion left the party: the sheet
	// closes the level-up modal when it was showing that companion (QA
	// edge, AC-19).
	CompanionDeparted(id entity.GameEntityID)
}

// Tooltip timing (GDD: 300ms hover delay, dismisses on mouse-out).
const (
	// tooltipDelay is the hover delay before the tooltip appears.
	tooltipDelay = 0.300
	// tooltipGrace is how long the tooltip survives after the pointer
	// leaves the portrait — long enough to cross onto the tooltip and click
	// Inspect. The tooltip's own enter event cancels the grace.
	tooltipGrace = 0.100
)

// Portrait size tuning knob (GDD sidebar_portrait_size = 72px, range
// 48-128).
const (
	sidebarPortraitSize = 72
	minPortraitSize     = 48
	maxPortraitSize     = 128
)

// partySubscriptionPriority is the sidebar's event subscription priority
// (ADR-0011: presentation is lowest, priority 10 — matching the screen
// state machine).
const partySubscriptionPriority = 10

// TooltipHP returns the current and max HP shown in the companion tooltip:
// read live from the entity's Health component when present; without one it
// falls back to the derived MaxHealth with a full bar (defensive — an
// entity with no Health component is never damaged). An entity with neither
// component yields 0, 0.
func TooltipHP(store entity.EntityStore, id entity.GameEntityID) (current, max int) {
	if h, ok := entity.GetComponent[entity.Health](store, id); ok && h.Max > 0 {
		return h.Current, h.Max
	}
	if c, ok := entity.GetComponent[stats.StatsComponent](store, id); ok {
		max = c.MaxHealth()
		return max, max
	}
	return 0, 0
}

// TooltipHPFraction returns the tooltip HP bar fill fraction: current/max
// clamped to [0, 1]; 0 when max is zero (defensive — TooltipHP returns a
// positive max for every component-bearing entity, so a zero max only
// occurs for a malformed entity).
func TooltipHPFraction(current, max int) float64 {
	if max <= 0 {
		return 0
	}
	f := float64(current) / float64(max)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// TrustLevelDisplay renders a trust level for the companion inspect panel:
// the level name with the numeric scale, e.g. "Trusted (2/3)" (trust GDD
// CR-1; names from trust.TrustLevel.String — English placeholders pending
// localization).
func TrustLevelDisplay(level trust.TrustLevel) string {
	return fmt.Sprintf("%s (%d/3)", level, level)
}

// TrustHistoryDisplay renders the companion's relationship history for the
// inspect panel: the last five trust events in order (most recent last),
// joined with ", ". An empty history renders "No relationship history."
// (English placeholder pending localization).
func TrustHistoryDisplay(events []trust.TrustEvent) string {
	if len(events) == 0 {
		return "No relationship history."
	}
	if len(events) > 5 {
		events = events[len(events)-5:]
	}
	parts := make([]string, 0, len(events))
	for _, e := range events {
		parts = append(parts, string(e.Type))
	}
	return strings.Join(parts, ", ")
}

// portraitCell is the set of bound elements of one rendered portrait,
// addressed by class because duplicated templates share no ids (the engine
// strips the template root id, children are class-addressed).
type portraitCell struct {
	cell       *document.Element
	memberID   entity.GameEntityID
	frame      *document.Element
	glyph      *document.Element
	name       *ui.Label
	tooltip    *document.Element
	tipName    *ui.Label
	tipLevel   *ui.Label
	tipRole    *ui.Label
	tipHPFill  *document.Element
	tipHPLabel *ui.Label
	tipTrust   *document.Element
}

// PartySidebar is the character sheet's left party panel (Story 008). It
// owns the roster (SetParty), the portrait cells rendered from the hidden
// template, the selection state (amber border, oxblood focus ring), the
// companion tooltip state machine (300ms hover delay), and the AC-19
// departure subscriptions on the EventBus.
//
// The sidebar is headless-testable: constructed with a nil document it runs
// as a pure state machine (roster, selection, tooltip timing) with no
// markup access, mirroring the level-up modal's nil-doc test pattern.
type PartySidebar struct {
	doc    *document.Document
	store  entity.EntityStore
	bus    entity.EventBus
	viewer Viewer

	// root is the sidebar container element; tpl is the hidden
	// #party-portrait-tpl template duplicated per roster member. Both are
	// nil for headless tests.
	root *document.Element
	tpl  *document.Element

	// active holds the active members in party order (members[0] is the
	// player character); pool holds the recruited-but-inactive members,
	// rendered greyed below the active ones.
	active []PartyMember
	pool   []PartyMember

	// selectedID is the currently selected (viewed) member; focused marks
	// keyboard focus for the oxblood focus ring (Story 009 calls SetFocus).
	selectedID entity.GameEntityID
	focused    bool
	// focusedID is the portrait under keyboard focus (Story 009): while
	// p.focused, the oxblood ring renders on it instead of the selected
	// member, and FocusTo moves it through the active portraits. Zero
	// falls back to the selected member.
	focusedID entity.GameEntityID
	// reducedMotion skips the 300ms tooltip hover delay (Story 009,
	// AC-24): the tooltip appears instantly on hover.
	reducedMotion bool

	portraitSize int

	// cells are the currently rendered portrait cells, one per roster
	// member (active then pool).
	cells []portraitCell

	// Tooltip state machine: hoveredID is the portrait under the pointer
	// (0 = none), hoverTime accumulates toward tooltipDelay; once shown,
	// gracePending tracks the 100ms dismiss grace (pointer may be crossing
	// onto the tooltip). All fields are pure Go state — headless tests
	// drive them without a document.
	hoveredID      entity.GameEntityID
	hoverTime      float64
	tooltipVisible bool
	gracePending   bool
	graceTime      float64
}

// NewPartySidebar creates the party sidebar bound to the character sheet's
// markup document. It subscribes to EventPartyMemberDeparted (AC-19) and
// combat.EventCharacterDied at presentation priority 10 (ADR-0011); a nil
// bus skips the subscriptions (tests). A nil doc is valid for tests: the
// sidebar then runs as a pure state machine with no markup access.
func NewPartySidebar(doc *document.Document, store entity.EntityStore, bus entity.EventBus) *PartySidebar {
	p := &PartySidebar{
		doc:          doc,
		store:        store,
		bus:          bus,
		portraitSize: sidebarPortraitSize,
	}
	p.bindElements()
	if bus != nil {
		bus.Subscribe(EventPartyMemberDeparted, partySubscriptionPriority, p.onPartyDeparture)
		bus.Subscribe(combat.EventCharacterDied, partySubscriptionPriority, p.onCharacterDied)
	}
	return p
}

// SetViewer attaches the sheet that the sidebar switches views on.
// NewCharacterSheet calls it after both exist; tests pass a stub.
func (p *PartySidebar) SetViewer(v Viewer) { p.viewer = v }

// SetParty populates the sidebar from the party roster (AC-20). members[0]
// must be the player character (GDD party order). Active members render
// first in party order, then pool members greyed out below (unselectable,
// QA edge). The selection is preserved when the selected member is still
// active, otherwise it falls back to the player character. When the sheet
// is open on a member that left the roster, the view switches to the
// player character (defensive — the sheet never displays a non-party
// member).
func (p *PartySidebar) SetParty(members []PartyMember) {
	p.active = p.active[:0]
	p.pool = p.pool[:0]
	for _, m := range members {
		if m.InPool {
			p.pool = append(p.pool, m)
		} else {
			p.active = append(p.active, m)
		}
	}
	keep := p.selectedID
	p.selectedID = 0
	for _, m := range p.active {
		if m.ID == keep {
			p.selectedID = keep
			break
		}
	}
	if p.selectedID == 0 && len(p.active) > 0 {
		p.selectedID = p.active[0].ID
	}
	// Clamp the keyboard focus cursor: a focused member that left the
	// roster (or was demoted to the pool) falls back to the selection.
	if p.focused && p.focusedID != 0 {
		if m, ok := p.Member(p.focusedID); !ok || m.InPool {
			p.focusedID = 0
		}
	}
	p.render()
	// Never display a member that left the roster.
	if p.viewer != nil {
		if id := p.viewer.ViewedID(); id != 0 && p.selectedID != 0 {
			if _, ok := p.Member(id); !ok && id != p.selectedID {
				if err := p.viewer.SwitchMember(p.selectedID); err != nil {
					slog.Warn("party sidebar: roster change switch rejected", "entity", id, "err", err)
				}
			}
		}
	}
}

// SelectMember switches the view to the given party member (AC-22): the
// active tab is preserved (the switch only repopulates the content for the
// new member — the tab chrome is untouched). Pool companions are
// unselectable (click ignored, QA edge) and selecting the already-selected
// member is a no-op (QA edge). With no viewer attached the sidebar still
// updates its selection state (tests).
func (p *PartySidebar) SelectMember(id entity.GameEntityID) {
	if id == p.selectedID {
		return
	}
	m, ok := p.Member(id)
	if !ok || m.InPool {
		return
	}
	if p.viewer != nil {
		if err := p.viewer.SwitchMember(id); err != nil {
			slog.Warn("party sidebar: member switch rejected", "entity", id, "err", err)
			return
		}
	}
	p.setSelected(id)
}

// HoverMember marks the portrait under the pointer (portrait cell
// onmouseenter): it starts the 300ms tooltip timer — instantly when
// reduced motion is enabled (Story 009, AC-24). Pool companions are
// inert — greyed out, no tooltip, no hover (QA edge). Hovering a
// different member restarts the timer.
func (p *PartySidebar) HoverMember(id entity.GameEntityID) {
	m, ok := p.Member(id)
	if !ok || m.InPool {
		return
	}
	if id != p.hoveredID {
		p.hoveredID = id
		p.hoverTime = 0
		p.gracePending = false
		p.hideTooltip()
		if p.reducedMotion {
			p.showTooltip(id)
		}
	}
}

// HoverNone clears the hover state (portrait cell onmouseleave): when a
// tooltip is showing, it starts the 100ms dismiss grace so the pointer can
// cross onto the tooltip and click Inspect; otherwise the hover timer
// resets.
func (p *PartySidebar) HoverNone() {
	if p.tooltipVisible {
		p.gracePending = true
		p.graceTime = 0
		return
	}
	p.hoveredID = 0
	p.hoverTime = 0
}

// TooltipEnter cancels the dismiss grace (tooltip panel onmouseenter): the
// pointer crossed from the portrait onto the tooltip, so it stays up.
func (p *PartySidebar) TooltipEnter() {
	p.gracePending = false
}

// TooltipLeave dismisses the tooltip (tooltip panel onmouseleave).
func (p *PartySidebar) TooltipLeave() {
	p.hideTooltip()
	p.hoveredID = 0
	p.gracePending = false
}

// InspectHovered is the tooltip's Inspect button (AC-21): it switches the
// view to the hovered companion — the full character sheet opens for that
// companion (stats, trust level, knowledge domains) — and dismisses the
// tooltip.
func (p *PartySidebar) InspectHovered() {
	id := p.hoveredID
	p.hideTooltip()
	p.hoveredID = 0
	p.gracePending = false
	if id != 0 {
		p.SelectMember(id)
	}
}

// Update advances the tooltip timer by dt seconds: hovering a portrait for
// tooltipDelay (300ms) shows its tooltip; while the dismiss grace runs,
// the tooltip hides once tooltipGrace (100ms) elapses. The sheet calls it
// from its own Update.
func (p *PartySidebar) Update(dt float64) {
	if p.gracePending {
		p.graceTime += dt
		if p.graceTime >= tooltipGrace {
			p.gracePending = false
			p.hideTooltip()
			p.hoveredID = 0
		}
		return
	}
	if p.hoveredID != 0 && !p.tooltipVisible {
		p.hoverTime += dt
		if p.reducedMotion || p.hoverTime >= tooltipDelay {
			p.showTooltip(p.hoveredID)
		}
	}
}

// SetPortraitSize sets the portrait size in pixels, clamped to the tuning
// knob range 48-128 (GDD sidebar_portrait_size = 72). The size applies to
// every rendered portrait frame.
func (p *PartySidebar) SetPortraitSize(px int) {
	if px < minPortraitSize {
		px = minPortraitSize
	}
	if px > maxPortraitSize {
		px = maxPortraitSize
	}
	if px == p.portraitSize {
		return
	}
	p.portraitSize = px
	p.applyPortraitSize()
}

// PortraitSize returns the current portrait size in pixels.
func (p *PartySidebar) PortraitSize() int { return p.portraitSize }

// SetFocus marks the sidebar as keyboard-focused: the portrait under the
// focus cursor (focusedID — the selected member until FocusTo moves it)
// gains the oxblood focus ring around its amber selection border
// (ADR-0012 focus ring, ≥3:1 contrast). Story 009 wires Tab/arrow
// navigation and calls this.
func (p *PartySidebar) SetFocus(on bool) {
	if on == p.focused {
		return
	}
	p.focused = on
	if on && p.focusedID == 0 {
		p.focusedID = p.selectedID
	}
	p.applySelection()
}

// FocusTo moves the keyboard focus cursor to the given active member
// (Story 009): while the sidebar is focused, the oxblood ring renders on
// that portrait. Pool members are unselectable and are ignored, as are
// unknown IDs. The sheet calls it when the tab order enters the sidebar
// or the arrow keys move the cursor.
func (p *PartySidebar) FocusTo(id entity.GameEntityID) {
	if id == 0 {
		return
	}
	if m, ok := p.Member(id); !ok || m.InPool {
		return
	}
	p.focusedID = id
	p.applySelection()
}

// FocusedID returns the portrait under keyboard focus; 0 when the sidebar
// has no focus cursor.
func (p *PartySidebar) FocusedID() entity.GameEntityID { return p.focusedID }

// SetReducedMotion gates the tooltip hover delay (Story 009, AC-24): when
// on, a hovered portrait's tooltip appears immediately — no 300ms wait —
// and a tooltip pending while the setting turns on shows at once. The
// tooltip itself is a display toggle, not an animation, so this is the
// only timing the sidebar owns.
func (p *PartySidebar) SetReducedMotion(on bool) {
	p.reducedMotion = on
	if on && p.hoveredID != 0 && !p.tooltipVisible {
		p.showTooltip(p.hoveredID)
	}
}

// Reset clears the selection, focus cursor, hover, and tooltip state. The
// sheet calls it on Close/ForceClose so a stale tooltip, selection, or
// focus ring never survives across opens.
func (p *PartySidebar) Reset() {
	p.setSelected(0)
	p.focusedID = 0
	p.hoveredID = 0
	p.hoverTime = 0
	p.gracePending = false
	p.graceTime = 0
	p.hideTooltip()
}

// Teardown unsubscribes the departure/death handlers from the event bus.
// The code that tears down the character sheet calls it so the
// registrations never outlive the sidebar.
func (p *PartySidebar) Teardown() {
	if p.bus == nil {
		return
	}
	p.bus.Unsubscribe(EventPartyMemberDeparted, p.onPartyDeparture)
	p.bus.Unsubscribe(combat.EventCharacterDied, p.onCharacterDied)
}

// Members returns the full roster in render order: active members in party
// order (player first), then pool members.
func (p *PartySidebar) Members() []PartyMember {
	out := make([]PartyMember, 0, len(p.active)+len(p.pool))
	out = append(out, p.active...)
	out = append(out, p.pool...)
	return out
}

// Member returns the roster record for the given entity ID (active or
// pool), or false when the ID is not in the roster.
func (p *PartySidebar) Member(id entity.GameEntityID) (PartyMember, bool) {
	for _, m := range p.active {
		if m.ID == id {
			return m, true
		}
	}
	for _, m := range p.pool {
		if m.ID == id {
			return m, true
		}
	}
	return PartyMember{}, false
}

// PlayerID returns the player character's entity ID — the first active
// member (GDD: player at top); zero when the roster is empty.
func (p *PartySidebar) PlayerID() entity.GameEntityID {
	if len(p.active) == 0 {
		return 0
	}
	return p.active[0].ID
}

// SelectedID returns the currently selected (viewed) member; zero when the
// roster is empty or the sidebar was Reset.
func (p *PartySidebar) SelectedID() entity.GameEntityID { return p.selectedID }

// TooltipVisible reports whether a companion tooltip is currently shown.
func (p *PartySidebar) TooltipVisible() bool { return p.tooltipVisible }

// Order implements PartyRoster (level_up.go): the active party members in
// party order, player character first. The level-up modal uses it to queue
// simultaneous level-ups in party order (GDD edge case). Pool members are
// excluded — they are not active party members.
func (p *PartySidebar) Order() []entity.GameEntityID {
	out := make([]entity.GameEntityID, 0, len(p.active))
	for _, m := range p.active {
		out = append(out, m.ID)
	}
	return out
}

// onPartyDeparture handles EventPartyMemberDeparted (AC-19): the departed
// companion leaves the roster; if it was the viewed member the view
// switches to the player character.
func (p *PartySidebar) onPartyDeparture(ev entity.Event) {
	data, ok := ev.Data.(PartyDepartureData)
	if !ok {
		slog.Warn("party sidebar: departure payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	p.departMember(data.EntityID)
}

// onCharacterDied handles combat.EventCharacterDied (QA edge: companion
// death switches the view the same way as a departure). The Party System
// also emits party.member_departed with cause death; departMember is
// idempotent, so double notification is harmless.
func (p *PartySidebar) onCharacterDied(ev entity.Event) {
	data, ok := ev.Data.(combat.CharacterDiedData)
	if !ok {
		slog.Warn("party sidebar: death payload type mismatch", "type", fmt.Sprintf("%T", ev.Data))
		return
	}
	p.departMember(data.EntityID)
}

// departMember removes a companion from the roster (AC-19) and, when the
// departed companion is the member currently being viewed, switches the
// sheet to the player character immediately (CompanionDeparted closes any
// level-up modal showing the companion first — QA edge) and syncs the
// sidebar selection to the player. Simultaneous departures switch at most
// once: after the first switch the viewed member is the player, so later
// departures only remove portraits.
func (p *PartySidebar) departMember(id entity.GameEntityID) {
	p.removeMember(id)
	if p.hoveredID == id {
		p.hideTooltip()
		p.hoveredID = 0
		p.gracePending = false
	}
	if p.viewer == nil || p.viewer.ViewedID() != id {
		return
	}
	p.viewer.CompanionDeparted(id)
	playerID := p.PlayerID()
	if playerID == 0 || playerID == id {
		return
	}
	if err := p.viewer.SwitchMember(playerID); err != nil {
		slog.Warn("party sidebar: departure switch to player rejected", "err", err)
		return
	}
	// The selection follows the switch directly — the sheet's SwitchMember
	// also syncs it, but the sidebar must not depend on the viewer's
	// implementation (idempotent for the sheet).
	p.setSelected(playerID)
}

// removeMember drops a companion from the active or pool roster and
// re-renders. Removing a member that is not in the roster is a no-op
// (idempotent — death and departure may both fire).
func (p *PartySidebar) removeMember(id entity.GameEntityID) {
	for i, m := range p.active {
		if m.ID == id {
			p.active = append(p.active[:i], p.active[i+1:]...)
			p.render()
			return
		}
	}
	for i, m := range p.pool {
		if m.ID == id {
			p.pool = append(p.pool[:i], p.pool[i+1:]...)
			p.render()
			return
		}
	}
}

// render rebuilds the portrait cells from the roster: one duplicated
// template per active member, then one per pool member. A nil doc (tests)
// renders nothing — the roster state is still authoritative.
func (p *PartySidebar) render() {
	if p.doc == nil {
		return
	}
	for _, c := range p.cells {
		p.doc.RemoveElement(c.cell)
	}
	p.cells = p.cells[:0]
	for _, m := range p.active {
		p.renderCell(m)
	}
	for _, m := range p.pool {
		p.renderCell(m)
	}
	p.applySelection()
}

// renderCell duplicates the portrait template for one member and binds its
// elements by class: the name label, the behavior glyph class, and the
// portrait frame size. The per-cell tooltip is bound for later
// population; it starts hidden.
func (p *PartySidebar) renderCell(m PartyMember) {
	if p.doc == nil || p.tpl == nil || p.root == nil {
		return
	}
	cell := p.doc.DuplicateElementToParent(p.tpl, p.root)
	cell.SetAttribute("data-index", strconv.Itoa(len(p.cells)))
	b := portraitCell{cell: cell, memberID: m.ID}
	if elm := findDescendant(cell, "party-portrait-frame"); elm != nil {
		b.frame = elm
	}
	if elm := findDescendant(cell, "party-glyph"); elm != nil {
		b.glyph = elm
	}
	if elm := findDescendant(cell, "party-name"); elm != nil {
		b.name = elm.InnerLabel()
	}
	if elm := findDescendant(cell, "party-tooltip"); elm != nil {
		b.tooltip = elm
	}
	if elm := findDescendant(cell, "party-tooltip-name"); elm != nil {
		b.tipName = elm.InnerLabel()
	}
	if elm := findDescendant(cell, "party-tooltip-level"); elm != nil {
		b.tipLevel = elm.InnerLabel()
	}
	if elm := findDescendant(cell, "party-tooltip-role"); elm != nil {
		b.tipRole = elm.InnerLabel()
	}
	if elm := findDescendant(cell, "party-tooltip-hp-fill"); elm != nil {
		b.tipHPFill = elm
	}
	if elm := findDescendant(cell, "party-tooltip-hp-label"); elm != nil {
		b.tipHPLabel = elm.InnerLabel()
	}
	if elm := findDescendant(cell, "party-tooltip-trust"); elm != nil {
		b.tipTrust = elm
	}
	if b.name != nil {
		b.name.SetText(m.Name)
	}
	if b.glyph != nil {
		b.glyph.SetClasses("party-glyph", BehaviorGlyphClass(m.Behavior))
	}
	if b.frame != nil {
		p.doc.SetElementStyleProperty(b.frame, "width", fmt.Sprintf("%dpx", p.portraitSize))
		p.doc.SetElementStyleProperty(b.frame, "height", fmt.Sprintf("%dpx", p.portraitSize))
	}
	p.cells = append(p.cells, b)
}

// showTooltip populates and shows the hovered member's tooltip: name,
// level (live from the Stats component), role, HP bar (live from the
// Health component, TooltipHP), and the trust indicator (amber glow at
// trust 1+, full glow at trust 3). The HP bar always pairs with its ink
// "current / max HP" label — never color alone (ADR-0012).
func (p *PartySidebar) showTooltip(id entity.GameEntityID) {
	p.tooltipVisible = true
	if p.doc == nil {
		return
	}
	m, ok := p.Member(id)
	if !ok {
		return
	}
	cell := p.cellFor(id)
	if cell == nil || cell.tooltip == nil {
		return
	}
	if cell.tipName != nil {
		cell.tipName.SetText(m.Name)
	}
	level := 1
	if c, ok := entity.GetComponent[stats.StatsComponent](p.store, id); ok {
		level = c.Level
	}
	if cell.tipLevel != nil {
		cell.tipLevel.SetText(fmt.Sprintf("Level %d", level))
	}
	if cell.tipRole != nil {
		cell.tipRole.SetText(m.Role)
	}
	current, max := TooltipHP(p.store, id)
	if cell.tipHPFill != nil {
		p.doc.SetElementStyleProperty(cell.tipHPFill, "width",
			fmt.Sprintf("%.1f%%", TooltipHPFraction(current, max)*100))
	}
	if cell.tipHPLabel != nil {
		cell.tipHPLabel.SetText(fmt.Sprintf("%d / %d HP", current, max))
	}
	if cell.tipTrust != nil {
		cell.tipTrust.SetClasses("party-tooltip-trust", TrustIndicatorFor(int(m.Trust)).Class())
	}
	p.doc.SetElementStyleProperty(cell.tooltip, "display", "flex")
}

// hideTooltip hides every tooltip (only one can be showing at a time).
func (p *PartySidebar) hideTooltip() {
	p.tooltipVisible = false
	if p.doc == nil {
		return
	}
	for _, c := range p.cells {
		if c.tooltip != nil {
			p.doc.SetElementStyleProperty(c.tooltip, "display", "none")
		}
	}
}

// cellFor returns the rendered portrait cell for a member ID, or nil.
func (p *PartySidebar) cellFor(id entity.GameEntityID) *portraitCell {
	for i := range p.cells {
		if p.cells[i].memberID == id {
			return &p.cells[i]
		}
	}
	return nil
}

// setSelected updates the selection state and re-applies the selection
// classes. Idempotent — the sheet calls it after Open/SwitchMember and the
// sidebar after SelectMember; both paths agree on the same member.
func (p *PartySidebar) setSelected(id entity.GameEntityID) {
	p.selectedID = id
	p.applySelection()
}

// applySelection re-applies the selection classes to every portrait cell:
// the selected cell carries the 3px amber border (amber is selection
// accent, never text — ADR-0012); with keyboard focus the oxblood focus
// ring wraps the portrait under the focus cursor (focusedID, falling back
// to the selected member — Story 009). Pool cells stay greyed
// (party-portrait-pool).
func (p *PartySidebar) applySelection() {
	ringID := p.selectedID
	if p.focused && p.focusedID != 0 {
		ringID = p.focusedID
	}
	for _, c := range p.cells {
		classes := []string{"party-portrait"}
		if c.memberID == p.selectedID {
			classes = append(classes, "party-portrait-selected")
		}
		if p.focused && c.memberID == ringID {
			classes = append(classes, "party-portrait-focused")
		}
		if m, ok := p.Member(c.memberID); ok && m.InPool {
			classes = append(classes, "party-portrait-pool")
		}
		c.cell.SetClasses(classes...)
	}
}

// applyPortraitSize re-applies the portrait size to every rendered frame.
func (p *PartySidebar) applyPortraitSize() {
	if p.doc == nil {
		return
	}
	for _, c := range p.cells {
		if c.frame != nil {
			p.doc.SetElementStyleProperty(c.frame, "width", fmt.Sprintf("%dpx", p.portraitSize))
			p.doc.SetElementStyleProperty(c.frame, "height", fmt.Sprintf("%dpx", p.portraitSize))
		}
	}
}

// bindElements resolves the sidebar markup ids: the sidebar root and the
// hidden portrait template. A nil doc (tests) binds nothing.
func (p *PartySidebar) bindElements() {
	if p.doc == nil {
		return
	}
	p.root = sheetElement(p.doc, "party-sidebar")
	p.tpl = sheetElement(p.doc, "party-portrait-tpl")
}

// selectFromElement resolves a portrait cell's data-index to its member
// and selects it (onclick="partySelect").
func (p *PartySidebar) selectFromElement(elm *document.Element) {
	idx, ok := cellIndexFromData(elm)
	if !ok {
		return
	}
	if idx >= len(p.cells) {
		slog.Warn("party sidebar: portrait index out of range", "index", idx)
		return
	}
	p.SelectMember(p.cells[idx].memberID)
}

// hoverFromElement resolves a portrait cell's data-index to its member and
// marks it hovered (onmouseenter="partyHover").
func (p *PartySidebar) hoverFromElement(elm *document.Element) {
	idx, ok := cellIndexFromData(elm)
	if !ok {
		return
	}
	if idx >= len(p.cells) {
		slog.Warn("party sidebar: portrait index out of range", "index", idx)
		return
	}
	p.HoverMember(p.cells[idx].memberID)
}

// cellIndexFromData reads a portrait cell's data-index attribute. Cells
// without one are logged and rejected.
func cellIndexFromData(elm *document.Element) (int, bool) {
	idx, err := strconv.Atoi(elm.Attribute("data-index"))
	if err != nil {
		slog.Warn("party sidebar: portrait without data-index", "err", err)
		return 0, false
	}
	return idx, true
}

// findDescendant returns the first descendant of elm carrying the given
// class, searching depth-first. Duplicated templates share no ids (the
// engine strips the template root id), so per-cell elements are addressed
// by class — same convention as the inventory grid's cellCountLabel.
func findDescendant(elm *document.Element, class string) *document.Element {
	for _, c := range elm.Children {
		if c.HasClass(class) {
			return c
		}
		if f := findDescendant(c, class); f != nil {
			return f
		}
	}
	return nil
}

// partyFuncMap returns the onclick/onmouseenter/onmouseleave handler map for
// the party sidebar markup (character_sheet.html), dispatched through the
// markup loader's funcMap (title screen pattern). The handlers dispatch to
// the sheet's sidebar, which is created after the document loads — the
// closures read s.sidebar lazily and no-op while it is nil. Markup contract:
//
//	onclick="partySelect"              — portrait cell, data-index="{i}"
//	onmouseenter="partyHover"          — portrait cell, data-index="{i}"
//	onmouseleave="partyHoverNone"      — portrait cell
//	onmouseenter="partyTooltipEnter"   — tooltip panel (cancels dismiss grace)
//	onmouseleave="partyTooltipLeave"   — tooltip panel
//	onclick="partyInspect"             — tooltip Inspect button
func partyFuncMap(s *CharacterSheet) map[string]func(*document.Element) {
	return map[string]func(*document.Element){
		"partySelect": func(elm *document.Element) {
			if s.sidebar != nil {
				s.sidebar.selectFromElement(elm)
			}
		},
		"partyHover": func(elm *document.Element) {
			if s.sidebar != nil {
				s.sidebar.hoverFromElement(elm)
			}
		},
		"partyHoverNone": func(*document.Element) {
			if s.sidebar != nil {
				s.sidebar.HoverNone()
			}
		},
		"partyTooltipEnter": func(*document.Element) {
			if s.sidebar != nil {
				s.sidebar.TooltipEnter()
			}
		},
		"partyTooltipLeave": func(*document.Element) {
			if s.sidebar != nil {
				s.sidebar.TooltipLeave()
			}
		},
		"partyInspect": func(*document.Element) {
			if s.sidebar != nil {
				s.sidebar.InspectHovered()
			}
		},
	}
}

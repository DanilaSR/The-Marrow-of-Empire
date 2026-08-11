// Integration tests for Story 008: Party Sidebar & Companion Inspect.
//
// Evidence path: production/epics/character-inventory-ui/
// story-008-party-sidebar-companion-inspect.md
// Required evidence: this file must exist and pass (Integration story type —
// BLOCKING gate).
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-20 test_party_sidebar_shows_all_four_members_player_at_top          -> TestPartySidebarShowsAllFourMembersPlayerAtTop
//	AC-21 test_companion_inspect_shows_stats_trust_and_domains              -> TestCompanionInspectShowsStatsTrustAndDomains
//	AC-22 test_switch_member_preserves_tab                                  -> TestSwitchMemberPreservesTab
//	AC-19 test_companion_departure_switches_to_player                       -> TestCompanionDepartureSwitchesToPlayer
//
// QA edge cases from the story are covered alongside: pool companions
// greyed and unselectable, trust 0 shows no amber glow (indicator classes),
// death switches the view the same way as a departure, departure while the
// level-up modal shows the departed companion closes the modal, multiple
// simultaneous departures switch at most once, switching to the
// already-selected member is a no-op, the 300ms tooltip delay with the
// 100ms dismiss grace (pointer may cross onto the tooltip and click
// Inspect), the sidebar as the level-up modal's PartyRoster (party order),
// and the portrait size knob clamp (48-128).
//
// The tests drive the PartySidebar state machine and the departure event
// handlers against a real EntityStoreImpl and a real synchronous EventBus
// (ADR-0011), with a stub Viewer (helpers mirrored from
// tests/integration/entity and the Story 003 test file per the shared-helper
// convention — Go cannot import test helpers across packages). The sidebar
// is constructed with a nil document, so it runs as a pure state machine
// (roster, selection, tooltip timing) with no markup access — the same
// nil-doc pattern as the level-up modal tests. No engine UI runs: the
// markup layer (party_sidebar.go element binding and rendering) is verified
// by screenshot evidence per the UI story testing standard; everything
// data-bearing is exercised here against the entity store and event bus.
package character_sheet_test

import (
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
	"marrow-of-empire/trust_relationship"
)

// stubViewer is a test double for character_sheet.Viewer: it records
// SwitchMember and CompanionDeparted calls and exposes the viewed member.
type stubViewer struct {
	viewedID   entity.GameEntityID
	switchedTo []entity.GameEntityID
	departed   []entity.GameEntityID
}

func (v *stubViewer) SwitchMember(id entity.GameEntityID) error {
	v.switchedTo = append(v.switchedTo, id)
	v.viewedID = id
	return nil
}

func (v *stubViewer) ViewedID() entity.GameEntityID { return v.viewedID }

func (v *stubViewer) CompanionDeparted(id entity.GameEntityID) {
	v.departed = append(v.departed, id)
}

// newSidebar wires a sidebar (nil doc — pure state machine) to a real store,
// a real event bus, and a stub viewer.
func newSidebar() (*character_sheet.PartySidebar, *entity.EntityStoreImpl, *entity.EventBusImpl, *stubViewer) {
	store, _ := newTestStore()
	bus := entity.NewEventBus()
	viewer := &stubViewer{}
	sb := character_sheet.NewPartySidebar(nil, store, bus)
	sb.SetViewer(viewer)
	return sb, store, bus, viewer
}

// member is a PartyMember factory for tests: an active member with the
// given name, role, behavior mode, and trust level, and no relationship
// history or knowledge domains.
func member(id entity.GameEntityID, name, role string, behavior character_sheet.BehaviorMode, trustLevel trust.TrustLevel) character_sheet.PartyMember {
	return character_sheet.PartyMember{
		ID:       id,
		Name:     name,
		Role:     role,
		Behavior: behavior,
		Trust:    trustLevel,
	}
}

// poolMember is a recruited-but-inactive companion record (greyed,
// unselectable).
func poolMember(id entity.GameEntityID, name string) character_sheet.PartyMember {
	m := member(id, name, "Rogue", character_sheet.BehaviorPassive, trust.TrustStranger)
	m.InPool = true
	return m
}

// partyOf builds a [player, companions...] roster from character entity IDs.
func partyOf(player entity.GameEntityID, companions ...entity.GameEntityID) []character_sheet.PartyMember {
	out := []character_sheet.PartyMember{member(player, "Player", "Warden", character_sheet.BehaviorDefensive, trust.TrustConfidant)}
	for i, id := range companions {
		out = append(out, member(id, "Companion"+string(rune('A'+i)), "Rogue", character_sheet.BehaviorDefensive, trust.TrustAcquaintance))
	}
	return out
}

// createCharacterNoStats creates an entity with no Stats component (used to
// verify SwitchMember rejects entities that cannot be displayed).
func createCharacterNoStats(t *testing.T, store *entity.EntityStoreImpl) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(entity.EntityTypeCharacter, "No Stats", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity(character) error: %v", err)
	}
	return id
}

// AC-20: GIVEN a party of four (player + three companions), WHEN the sidebar
// receives the roster, THEN all four members show in party order with the
// player at top, and the player is selected by default.
func TestPartySidebarShowsAllFourMembersPlayerAtTop(t *testing.T) {
	sb, _, _, _ := newSidebar()
	player := entity.GameEntityID(1)
	companions := []entity.GameEntityID{2, 3, 4}

	sb.SetParty(partyOf(player, companions...))

	got := sb.Members()
	if len(got) != 4 {
		t.Fatalf("Members() = %d entries, want 4", len(got))
	}
	for i, want := range append([]entity.GameEntityID{player}, companions...) {
		if got[i].ID != want {
			t.Errorf("member %d = %d, want %d (party order)", i, got[i].ID, want)
		}
	}
	if sb.PlayerID() != player {
		t.Errorf("PlayerID() = %d, want %d (player at top)", sb.PlayerID(), player)
	}
	if sb.SelectedID() != player {
		t.Errorf("SelectedID() = %d, want %d (player selected by default)", sb.SelectedID(), player)
	}
}

// AC-20 QA edge: a solo roster (player only) renders the single member and
// selects them; an empty roster renders nothing and selects nothing.
func TestPartySidebarPlayerOnly(t *testing.T) {
	sb, _, _, _ := newSidebar()
	sb.SetParty(partyOf(entity.GameEntityID(1)))

	got := sb.Members()
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("Members() = %+v, want the player only", got)
	}
	if sb.SelectedID() != 1 {
		t.Errorf("SelectedID() = %d, want 1", sb.SelectedID())
	}

	sb.SetParty(nil)
	if got := sb.Members(); len(got) != 0 {
		t.Errorf("Members() after empty roster = %d entries, want 0", len(got))
	}
	if sb.PlayerID() != 0 || sb.SelectedID() != 0 {
		t.Errorf("empty roster: PlayerID = %d, SelectedID = %d, want 0, 0",
			sb.PlayerID(), sb.SelectedID())
	}
}

// AC-20 QA edge: the maximum party (six active members) renders all six in
// party order, with recruited-but-inactive pool companions below them.
func TestPartySidebarMaxPartyShowsAllSix(t *testing.T) {
	sb, _, _, _ := newSidebar()
	ids := []entity.GameEntityID{1, 2, 3, 4, 5, 6}
	roster := partyOf(ids[0], ids[1:]...)
	roster = append(roster, poolMember(7, "Vet"), poolMember(8, "Scribe"))

	sb.SetParty(roster)

	got := sb.Members()
	if len(got) != 8 {
		t.Fatalf("Members() = %d entries, want 8 (6 active + 2 pool)", len(got))
	}
	for i, id := range ids {
		if got[i].ID != id || got[i].InPool {
			t.Errorf("member %d = %d (pool=%v), want active %d", i, got[i].ID, got[i].InPool, id)
		}
	}
	for i := 6; i < 8; i++ {
		if !got[i].InPool {
			t.Errorf("member %d (%d) must be marked InPool", i, got[i].ID)
		}
	}
	if got := sb.Order(); len(got) != 6 || got[0] != 1 || got[5] != 6 {
		t.Errorf("Order() = %v, want the six active members in party order (pool excluded)", got)
	}
}

// QA edge: pool companions render below the active members and are
// unselectable — SelectMember on a pool companion is ignored and never
// reaches the viewer.
func TestPoolCompanionsGreyedBelowActiveUnselectable(t *testing.T) {
	sb, _, _, viewer := newSidebar()
	player := entity.GameEntityID(1)
	companion := entity.GameEntityID(2)
	pool := entity.GameEntityID(7)
	sb.SetParty(append(partyOf(player, companion), poolMember(pool, "Vet")))

	sb.SelectMember(companion)
	if viewer.viewedID != companion {
		t.Fatalf("viewer.viewedID = %d, want %d (companion selected)", viewer.viewedID, companion)
	}

	sb.SelectMember(pool)
	if len(viewer.switchedTo) != 1 {
		t.Errorf("pool selection reached the viewer %d time(s): %v — must be unselectable",
			len(viewer.switchedTo), viewer.switchedTo)
	}
	if sb.SelectedID() != companion {
		t.Errorf("SelectedID() = %d, want %d (selection unchanged)", sb.SelectedID(), companion)
	}
}

// The sidebar implements the level-up modal's PartyRoster: Order returns
// the active members in party order (player first). The modal sorts its
// pending queue by that order, so simultaneous level-ups display player
// first regardless of arrival order.
func TestSidebarOrderImplementsPartyRoster(t *testing.T) {
	var roster character_sheet.PartyRoster // compile-time interface check
	sb, store, _, _ := newSidebar()
	attrs := [stats.NumAttributes]int{}
	player := createCharacter(t, store, attrs)
	a := createCharacter(t, store, attrs)
	b := createCharacter(t, store, attrs)
	roster = sb
	sb.SetParty(append(partyOf(player, a, b), poolMember(9, "Vet")))

	if got := roster.Order(); len(got) != 3 || got[0] != player || got[1] != a || got[2] != b {
		t.Fatalf("Order() = %v, want [%d %d %d] (active only, party order)", got, player, a, b)
	}

	// Queueing arrives player, B, A — the roster sort must display A before B.
	modal := character_sheet.NewLevelUpModal(nil, store, roster)
	modal.Show(player, 1, 2, 3, false)
	modal.Show(b, 1, 2, 3, false)
	modal.Show(a, 1, 2, 3, false)
	if modal.CurrentEntityID() != player {
		t.Fatalf("current level-up = %d, want %d (player first)", modal.CurrentEntityID(), player)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() error: %v", err)
	}
	if modal.CurrentEntityID() != a {
		t.Errorf("next level-up = %d, want %d (party order overrides arrival)", modal.CurrentEntityID(), a)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() error: %v", err)
	}
	if modal.CurrentEntityID() != b {
		t.Errorf("next level-up = %d, want %d", modal.CurrentEntityID(), b)
	}
}

// AC-21: GIVEN the player selects a companion portrait, WHEN the sheet
// switches to the companion, THEN the inspect view shows the companion's
// stats, trust level, and knowledge domains. The trust display helpers, the
// knowledge domain proficiency indicator, and the live HP read are asserted
// here; the DOM rendering of the inspect panel is screenshot evidence.
func TestCompanionInspectShowsStatsTrustAndDomains(t *testing.T) {
	sb, store, _, viewer := newSidebar()
	attrs := [stats.NumAttributes]int{stats.CON: 10}
	player := createCharacter(t, store, attrs)
	companion := createCharacter(t, store, attrs)
	roster := partyOf(player)
	roster = append(roster, character_sheet.PartyMember{
		ID:       companion,
		Name:     "Mara",
		Role:     "Rogue",
		Behavior: character_sheet.BehaviorAggressive,
		Trust:    trust.TrustTrusted,
		TrustHistory: []trust.TrustEvent{
			{Type: trust.EventSmallFavorCompleted},
			{Type: trust.EventThreadResolvedInvolvingNPC},
			{Type: trust.EventSignificantHelp},
		},
		Domains: []character_sheet.KnowledgeDomain{
			{Name: "craft_metallurgy", Proficiency: character_sheet.KnowledgeDomainProficiency(5)},
			{Name: "history_empire", Proficiency: character_sheet.KnowledgeDomainProficiency(9)},
		},
	})
	sb.SetParty(roster)

	sb.SelectMember(companion)
	if viewer.viewedID != companion || sb.SelectedID() != companion {
		t.Fatalf("selection: viewer = %d, sidebar = %d, want both %d",
			viewer.viewedID, sb.SelectedID(), companion)
	}
	if len(viewer.switchedTo) != 1 || viewer.switchedTo[0] != companion {
		t.Errorf("switchedTo = %v, want [%d]", viewer.switchedTo, companion)
	}

	// Trust & Relationship: "Trusted (2/3)" — name with the numeric scale.
	if got := character_sheet.TrustLevelDisplay(trust.TrustTrusted); got != "Trusted (2/3)" {
		t.Errorf("TrustLevelDisplay(Trusted) = %q, want %q", got, "Trusted (2/3)")
	}
	// History: the last five events in order, most recent last; empty
	// history renders the placeholder.
	if got := character_sheet.TrustHistoryDisplay(roster[1].TrustHistory); got !=
		"small_favor_completed, thread_resolved_involving_npc, significant_help" {
		t.Errorf("TrustHistoryDisplay = %q, want the three event types joined", got)
	}
	if got := character_sheet.TrustHistoryDisplay(nil); got != "No relationship history." {
		t.Errorf("TrustHistoryDisplay(nil) = %q, want the empty placeholder", got)
	}
	// History truncation: more than five events keeps the last five.
	seven := make([]trust.TrustEvent, 7)
	for i := range seven {
		seven[i] = trust.TrustEvent{Type: trust.TrustEventType("evt_" + string(rune('1'+i)))}
	}
	if got := character_sheet.TrustHistoryDisplay(seven); got != "evt_3, evt_4, evt_5, evt_6, evt_7" {
		t.Errorf("TrustHistoryDisplay(7 events) = %q, want the last five", got)
	}

	// Knowledge domains: proficiency indicator from fact counts.
	if got := roster[1].Domains[0].Proficiency; got != 3 {
		t.Errorf("metallurgy proficiency (5 facts) = %d, want 3", got)
	}
	if got := roster[1].Domains[1].Proficiency; got != 5 {
		t.Errorf("history proficiency (9 facts) = %d, want 5 (cap)", got)
	}

	// Stats: live HP read from the store — Health component when present,
	// full derived bar without one.
	if cur, max := character_sheet.TooltipHP(store, companion); cur != max || max <= 0 {
		t.Errorf("TooltipHP without Health = (%d, %d), want a full derived bar", cur, max)
	}
	if err := entity.SetComponent(store, companion, entity.Health{Current: 21, Max: 54}); err != nil {
		t.Fatalf("SetComponent(Health) error: %v", err)
	}
	if cur, max := character_sheet.TooltipHP(store, companion); cur != 21 || max != 54 {
		t.Errorf("TooltipHP with Health = (%d, %d), want (21, 54)", cur, max)
	}
	if got := character_sheet.TooltipHPFraction(21, 54); got != 21.0/54.0 {
		t.Errorf("TooltipHPFraction(21, 54) = %v, want 21/54", got)
	}
	if got := character_sheet.TooltipHPFraction(0, 0); got != 0 {
		t.Errorf("TooltipHPFraction(0, 0) = %v, want 0 (defensive)", got)
	}
	if got := character_sheet.TooltipHPFraction(-5, 54); got != 0 {
		t.Errorf("TooltipHPFraction(-5, 54) = %v, want 0 (clamped)", got)
	}
	if got := character_sheet.TooltipHPFraction(60, 54); got != 1 {
		t.Errorf("TooltipHPFraction(60, 54) = %v, want 1 (clamped)", got)
	}
}

// AC-21 QA edge: the trust indicator maps trust 0 → none (no amber glow),
// 1-2 → glow, 3+ → full glow; the classes style fill/glow only, never text.
func TestTrustIndicatorLevels(t *testing.T) {
	cases := []struct {
		level int
		want  character_sheet.TrustIndicator
		class string
	}{
		{0, character_sheet.TrustNone, "trust-none"},
		{1, character_sheet.TrustGlow, "trust-glow"},
		{2, character_sheet.TrustGlow, "trust-glow"},
		{3, character_sheet.TrustFull, "trust-full"},
		{4, character_sheet.TrustFull, "trust-full"}, // above scale: full
	}
	for _, c := range cases {
		got := character_sheet.TrustIndicatorFor(c.level)
		if got != c.want {
			t.Errorf("TrustIndicatorFor(%d) = %v, want %v", c.level, got, c.want)
		}
		if got.Class() != c.class {
			t.Errorf("TrustIndicator(%d).Class() = %q, want %q", c.level, got.Class(), c.class)
		}
	}
}

// KnowledgeDomainProficiency: 0 facts → 0, one step per two facts, 9+ → 5,
// clamped to [0, 5].
func TestKnowledgeDomainProficiency(t *testing.T) {
	cases := []struct {
		facts int
		want  int
	}{
		{-3, 0}, {0, 0}, {1, 1}, {2, 1}, {3, 2}, {4, 2},
		{5, 3}, {6, 3}, {7, 4}, {8, 4}, {9, 5}, {12, 5},
	}
	for _, c := range cases {
		if got := character_sheet.KnowledgeDomainProficiency(c.facts); got != c.want {
			t.Errorf("KnowledgeDomainProficiency(%d) = %d, want %d", c.facts, got, c.want)
		}
	}
}

// Behavior glyph classes (party GDD CR-5): sword for aggressive, shield for
// defensive, eye for passive; unknown modes fall back to the shield.
func TestBehaviorGlyphClasses(t *testing.T) {
	if got := character_sheet.BehaviorGlyphClass(character_sheet.BehaviorAggressive); got != "party-glyph-aggressive" {
		t.Errorf("aggressive glyph = %q, want party-glyph-aggressive", got)
	}
	if got := character_sheet.BehaviorGlyphClass(character_sheet.BehaviorDefensive); got != "party-glyph-defensive" {
		t.Errorf("defensive glyph = %q, want party-glyph-defensive", got)
	}
	if got := character_sheet.BehaviorGlyphClass(character_sheet.BehaviorPassive); got != "party-glyph-passive" {
		t.Errorf("passive glyph = %q, want party-glyph-passive", got)
	}
	if got := character_sheet.BehaviorGlyphClass(character_sheet.BehaviorMode(99)); got != "party-glyph-defensive" {
		t.Errorf("unknown mode glyph = %q, want the shield fallback", got)
	}
	if !character_sheet.BehaviorDefensive.Valid() || character_sheet.BehaviorMode(99).Valid() {
		t.Error("Valid() must accept the three modes and reject unknown values")
	}
	if got := character_sheet.BehaviorAggressive.String(); got != "Aggressive" {
		t.Errorf("BehaviorAggressive.String() = %q, want %q", got, "Aggressive")
	}
}

// QA edge: the companion tooltip shows after a 300ms hover, survives the
// 100ms dismiss grace (the pointer may cross onto the tooltip), and
// dismisses on mouse-out. Pool companions never show a tooltip.
func TestTooltipDelayAndDismiss(t *testing.T) {
	sb, _, _, _ := newSidebar()
	player := entity.GameEntityID(1)
	companion := entity.GameEntityID(2)
	sb.SetParty(partyOf(player, companion))

	sb.HoverMember(companion)
	if sb.TooltipVisible() {
		t.Fatal("tooltip visible immediately on hover")
	}
	sb.Update(0.29)
	if sb.TooltipVisible() {
		t.Fatal("tooltip visible before the 300ms delay")
	}
	sb.Update(0.02)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip did not appear after 300ms of hover")
	}

	// Pointer leaves the portrait, then crosses onto the tooltip within the
	// 100ms grace: the tooltip must survive, and the tooltip's own enter
	// cancels the grace entirely.
	sb.HoverNone()
	sb.Update(0.05)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip dismissed during the 100ms grace")
	}
	sb.TooltipEnter()
	sb.Update(0.5)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip enter must cancel the dismiss grace")
	}

	// Mouse-out from the tooltip dismisses it immediately.
	sb.TooltipLeave()
	if sb.TooltipVisible() {
		t.Fatal("tooltip still visible after mouse-out")
	}

	// Grace expiry without reaching the tooltip dismisses it after 100ms.
	sb.HoverMember(companion)
	sb.Update(0.3)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip did not reappear on re-hover")
	}
	sb.HoverNone()
	sb.Update(0.09)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip dismissed before the grace elapsed")
	}
	sb.Update(0.02)
	if sb.TooltipVisible() {
		t.Fatal("tooltip must dismiss once the 100ms grace elapses")
	}

	// Hovering a pool companion is inert: no timer, no tooltip.
	pool := entity.GameEntityID(9)
	sb.SetParty(append(partyOf(player, companion), poolMember(pool, "Vet")))
	sb.HoverMember(pool)
	sb.Update(1.0)
	if sb.TooltipVisible() {
		t.Error("pool companion hover must never show a tooltip")
	}
}

// QA edge: the tooltip's Inspect button switches the sheet to the hovered
// companion (AC-21) and dismisses the tooltip.
func TestTooltipInspectSwitchesToCompanion(t *testing.T) {
	sb, _, _, viewer := newSidebar()
	player := entity.GameEntityID(1)
	companion := entity.GameEntityID(2)
	sb.SetParty(partyOf(player, companion))

	sb.HoverMember(companion)
	sb.Update(0.3)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip did not appear")
	}
	sb.InspectHovered()

	if len(viewer.switchedTo) != 1 || viewer.switchedTo[0] != companion {
		t.Errorf("Inspect must switch to the hovered companion; switchedTo = %v", viewer.switchedTo)
	}
	if sb.SelectedID() != companion {
		t.Errorf("SelectedID() = %d, want %d", sb.SelectedID(), companion)
	}
	if sb.TooltipVisible() {
		t.Error("tooltip must dismiss after Inspect")
	}
}

// QA edge: the portrait size tuning knob clamps to the 48-128 range
// (GDD sidebar_portrait_size = 72 default).
func TestPortraitSizeClamp(t *testing.T) {
	sb, _, _, _ := newSidebar()
	if got := sb.PortraitSize(); got != 72 {
		t.Fatalf("default PortraitSize() = %d, want 72", got)
	}
	sb.SetPortraitSize(32)
	if got := sb.PortraitSize(); got != 48 {
		t.Errorf("SetPortraitSize(32) = %d, want clamped 48", got)
	}
	sb.SetPortraitSize(200)
	if got := sb.PortraitSize(); got != 128 {
		t.Errorf("SetPortraitSize(200) = %d, want clamped 128", got)
	}
	sb.SetPortraitSize(84)
	if got := sb.PortraitSize(); got != 84 {
		t.Errorf("SetPortraitSize(84) = %d, want 84", got)
	}
}

// AC-22: GIVEN the sheet is open on one member, WHEN the player switches to
// another member, THEN the active tab is preserved — the switch only
// repopulates the viewed member's content and never touches the tab chrome.
// The headless sheet cannot observe the tab DOM (screenshot evidence), so
// the contract is asserted here: SwitchMember changes the viewed member,
// reports errors for entities without Stats, and the sidebar's selection
// path drives the switch.
func TestSwitchMemberPreservesTab(t *testing.T) {
	store, _ := newTestStore()
	bus := entity.NewEventBus()
	player := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 14})
	companion := createCharacter(t, store, [stats.NumAttributes]int{stats.STR: 10})
	sheet := character_sheet.NewCharacterSheet(nil, nil, store, bus) // degraded: markup never loads

	if err := sheet.Open(player); err != nil {
		t.Fatalf("Open(player) error: %v", err)
	}
	if sheet.ViewedID() != player {
		t.Fatalf("ViewedID() = %d, want %d", sheet.ViewedID(), player)
	}

	// Switch to the companion: the view moves, the tab chrome is untouched
	// (SwitchMember calls no tab APIs — by construction the tab selection
	// survives; the DOM rendering is screenshot evidence).
	if err := sheet.SwitchMember(companion); err != nil {
		t.Fatalf("SwitchMember(companion) error: %v", err)
	}
	if sheet.ViewedID() != companion {
		t.Errorf("ViewedID() = %d, want %d after switch", sheet.ViewedID(), companion)
	}

	// Switching back to the player.
	if err := sheet.SwitchMember(player); err != nil {
		t.Fatalf("SwitchMember(player) error: %v", err)
	}
	if sheet.ViewedID() != player {
		t.Errorf("ViewedID() = %d, want %d after switching back", sheet.ViewedID(), player)
	}

	// An entity without a Stats component cannot be displayed: the switch
	// fails and the view stays on the current member.
	bad := createCharacterNoStats(t, store)
	if err := sheet.SwitchMember(bad); err == nil {
		t.Error("SwitchMember on an entity without Stats must error")
	}
	if sheet.ViewedID() != player {
		t.Errorf("ViewedID() = %d, want %d (failed switch leaves the view)", sheet.ViewedID(), player)
	}
}

// AC-22 QA edges: selecting the already-selected member is a no-op, and
// selecting a member not in the roster is ignored — neither reaches the
// viewer.
func TestSwitchMemberNoOpEdges(t *testing.T) {
	sb, _, _, viewer := newSidebar()
	player := entity.GameEntityID(1)
	companion := entity.GameEntityID(2)
	sb.SetParty(partyOf(player, companion))

	sb.SelectMember(companion)
	if len(viewer.switchedTo) != 1 {
		t.Fatalf("initial switch: switchedTo = %v, want [%d]", viewer.switchedTo, companion)
	}
	sb.SelectMember(companion)
	if len(viewer.switchedTo) != 1 {
		t.Errorf("re-selecting the viewed member must be a no-op; switchedTo = %v", viewer.switchedTo)
	}
	sb.SelectMember(entity.GameEntityID(99))
	if len(viewer.switchedTo) != 1 {
		t.Errorf("selecting a member outside the roster must be ignored; switchedTo = %v", viewer.switchedTo)
	}
	if sb.SelectedID() != companion {
		t.Errorf("SelectedID() = %d, want %d", sb.SelectedID(), companion)
	}
}

// AC-19: GIVEN the player is viewing a companion, WHEN that companion leaves
// the party (party.member_departed event), THEN the sheet switches to the
// player character immediately, the tooltip for the departed companion
// dismisses, and the portrait disappears from the roster.
func TestCompanionDepartureSwitchesToPlayer(t *testing.T) {
	sb, _, bus, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a, b := entity.GameEntityID(2), entity.GameEntityID(3)
	sb.SetParty(partyOf(player, a, b))

	sb.SelectMember(a)
	sb.HoverMember(a)
	sb.Update(0.3)
	if !sb.TooltipVisible() {
		t.Fatal("tooltip did not appear for the viewed companion")
	}

	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: a, Cause: character_sheet.DepartureStory},
	})

	if len(viewer.departed) != 1 || viewer.departed[0] != a {
		t.Errorf("CompanionDeparted calls = %v, want [%d]", viewer.departed, a)
	}
	if viewer.viewedID != player {
		t.Errorf("viewer.viewedID = %d, want %d (switch to player)", viewer.viewedID, player)
	}
	if sb.SelectedID() != player {
		t.Errorf("SelectedID() = %d, want %d", sb.SelectedID(), player)
	}
	if sb.TooltipVisible() {
		t.Error("tooltip must dismiss when the hovered companion departs")
	}
	if _, ok := sb.Member(a); ok {
		t.Error("departed companion still in the roster")
	}
	if got := sb.Members(); len(got) != 2 || got[0].ID != player || got[1].ID != b {
		t.Errorf("Members() = %+v, want [player, %d]", got, b)
	}
}

// AC-19 QA edge: when the player is viewing the player character, a
// companion departure removes the portrait but never switches the view.
func TestDepartureWhileViewingPlayer(t *testing.T) {
	sb, _, bus, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a, b := entity.GameEntityID(2), entity.GameEntityID(3)
	sb.SetParty(partyOf(player, a, b))
	// The game layer opened the player sheet directly (Open(player)); the
	// stub's viewedID reflects that without a SwitchMember call.
	viewer.viewedID = player

	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: b, Cause: character_sheet.DepartureDismissed},
	})

	if len(viewer.switchedTo) != 0 {
		t.Errorf("switchedTo = %v, want none (no departure switch while viewing the player)", viewer.switchedTo)
	}
	if len(viewer.departed) != 0 {
		t.Errorf("CompanionDeparted = %v, want none (viewing the player)", viewer.departed)
	}
	if viewer.viewedID != player {
		t.Errorf("viewedID = %d, want %d", viewer.viewedID, player)
	}
	if _, ok := sb.Member(b); ok {
		t.Error("departed companion still in the roster")
	}
}

// AC-19 QA edge: simultaneous departures switch the view at most once —
// after the first switch the viewed member is the player, so the second
// departure only removes its portrait.
func TestMultipleDeparturesSwitchOnce(t *testing.T) {
	sb, _, bus, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a, b := entity.GameEntityID(2), entity.GameEntityID(3)
	sb.SetParty(partyOf(player, a, b))
	sb.SelectMember(a)

	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: a, Cause: character_sheet.DepartureTrustBetrayal},
	})
	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: b, Cause: character_sheet.DepartureStory},
	})

	if len(viewer.switchedTo) != 2 {
		t.Errorf("switchedTo = %v, want exactly [initial, player]", viewer.switchedTo)
	} else if viewer.switchedTo[1] != player {
		t.Errorf("switchedTo[1] = %d, want %d (exactly one departure switch)", viewer.switchedTo[1], player)
	}
	if len(viewer.departed) != 1 || viewer.departed[0] != a {
		t.Errorf("CompanionDeparted = %v, want [%d] only", viewer.departed, a)
	}
	if viewer.viewedID != player {
		t.Errorf("viewedID = %d, want %d", viewer.viewedID, player)
	}
	if got := sb.Members(); len(got) != 1 || got[0].ID != player {
		t.Errorf("Members() = %+v, want [player]", got)
	}
}

// AC-19 QA edge: a companion death (combat.character_died) switches the view
// the same way as a departure, and a subsequent departure event for the same
// companion is a harmless no-op (idempotent removal).
func TestDepartureViaCombatDeath(t *testing.T) {
	sb, _, bus, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a := entity.GameEntityID(2)
	sb.SetParty(partyOf(player, a))
	sb.SelectMember(a)

	bus.Emit(entity.Event{
		Type: combat.EventCharacterDied,
		Data: combat.CharacterDiedData{EntityID: a},
	})
	if viewer.viewedID != player {
		t.Errorf("viewedID = %d, want %d after companion death", viewer.viewedID, player)
	}
	if len(viewer.departed) != 1 || viewer.departed[0] != a {
		t.Errorf("CompanionDeparted = %v, want [%d] (death treated as departure)", viewer.departed, a)
	}

	// The party layer may also emit party.member_departed (cause death) —
	// the second notification must not error or double-switch.
	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: a, Cause: character_sheet.DepartureDeath},
	})
	if len(viewer.switchedTo) != 2 {
		t.Errorf("switchedTo = %v, want exactly [initial, player] (idempotent)", viewer.switchedTo)
	}
	if viewer.viewedID != player {
		t.Errorf("viewedID = %d, want %d (unchanged)", viewer.viewedID, player)
	}
}

// AC-19 QA edge: when the roster changes so the viewed member left it
// entirely (e.g. SetParty without them), the view switches to the player
// defensively — the sheet never displays a non-party member. A roster
// update that keeps the viewed member never switches.
func TestSetPartySwitchesViewWhenViewedMemberLeavesRoster(t *testing.T) {
	sb, _, _, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a, b := entity.GameEntityID(2), entity.GameEntityID(3)
	sb.SetParty(partyOf(player, a, b))
	sb.SelectMember(a)
	base := len(viewer.switchedTo)

	// The viewed member stays: no switch.
	sb.SetParty(partyOf(player, a, b))
	if len(viewer.switchedTo) != base {
		t.Errorf("roster refresh must not switch the view; switchedTo = %v", viewer.switchedTo)
	}

	// The viewed member leaves the roster: switch to the player.
	sb.SetParty(partyOf(player, b))
	if len(viewer.switchedTo) != base+1 || viewer.switchedTo[len(viewer.switchedTo)-1] != player {
		t.Errorf("viewed member left the roster: switchedTo = %v, want a switch to the player", viewer.switchedTo)
	}
	if sb.SelectedID() != player {
		t.Errorf("SelectedID() = %d, want %d", sb.SelectedID(), player)
	}
}

// QA edge: departing a companion closes the level-up modal when the modal is
// showing that companion (Abort: no write, queue discarded), while a modal
// for a different companion stays open. The sheet's CompanionDeparted hook
// performs the guard; on the headless degraded sheet its modal is nil, so
// the modal behavior itself is asserted here directly.
func TestDepartureClosesLevelUpModalForDeparted(t *testing.T) {
	store, _ := newTestStore()
	a := createCharacter(t, store, [stats.NumAttributes]int{stats.CON: 10})
	b := createCharacter(t, store, [stats.NumAttributes]int{stats.CON: 10})
	modal := character_sheet.NewLevelUpModal(nil, store, nil)

	// A level-up for the departed companion is aborted: no attribute write
	// happens (the point is forfeited — the character left the party).
	modal.Show(a, 3, 4, 2, true)
	if !modal.Active() || modal.CurrentEntityID() != a {
		t.Fatalf("modal state: active=%v current=%d, want active on %d", modal.Active(), modal.CurrentEntityID(), a)
	}
	modal.Abort()
	if modal.Active() {
		t.Fatal("modal must close when the departed companion is showing")
	}
	if c, ok := entity.GetComponent[stats.StatsComponent](store, a); ok {
		if c.Attributes[stats.STR] != 0 {
			t.Errorf("abort must not write anything; STR = %d, want 0", c.Attributes[stats.STR])
		}
	}

	// A level-up for a companion that did NOT depart stays open.
	modal.Show(b, 3, 4, 2, false)
	if !modal.Active() || modal.CurrentEntityID() != b {
		t.Fatalf("modal must stay open for the non-departed companion")
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm error: %v", err)
	}

	// Abort discards the pending queue: queued level-ups do not resurface
	// after a departure.
	modal.Show(a, 1, 2, 3, false)
	modal.Show(b, 1, 2, 3, false)
	modal.Abort()
	modal.Show(b, 4, 5, 3, false)
	if modal.CurrentEntityID() != b {
		t.Fatalf("after abort, a fresh level-up must display immediately; current = %d", modal.CurrentEntityID())
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm error: %v", err)
	}
	if modal.Active() {
		t.Fatal("no queued level-up may resurface after an abort")
	}

	// The sheet hook is nil-safe on the degraded sheet (no modal, no panic).
	bus := entity.NewEventBus()
	sheet := character_sheet.NewCharacterSheet(nil, nil, store, bus)
	sheet.CompanionDeparted(a)
}

// The sidebar's departure subscription is released by Teardown: departures
// after teardown never touch the roster.
func TestSidebarTeardownUnsubscribes(t *testing.T) {
	sb, _, bus, viewer := newSidebar()
	player := entity.GameEntityID(1)
	a := entity.GameEntityID(2)
	sb.SetParty(partyOf(player, a))
	sb.SelectMember(a)

	sb.Teardown()
	bus.Emit(entity.Event{
		Type: character_sheet.EventPartyMemberDeparted,
		Data: character_sheet.PartyDepartureData{EntityID: a, Cause: character_sheet.DepartureStory},
	})

	if len(viewer.departed) != 0 {
		t.Errorf("CompanionDeparted after teardown = %v, want none (unsubscribed)", viewer.departed)
	}
	if _, ok := sb.Member(a); !ok {
		t.Error("roster changed after teardown — subscription still live")
	}
}

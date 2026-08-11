// Unit tests for Story 002: XP Progress Bar & Level-Up Modal.
//
// Evidence path: production/epics/character-inventory-ui/
// story-002-xp-progress-level-up.md
// Each test maps to one acceptance criterion or QA edge case: AC-4 (XP bar
// fill at 40% for the GDD example), AC-18 (mandatory level-up modal), the
// GDD formulas (ADR-0006 / TR-stats-005: XP_to_next = Level × 300,
// Total_XP = 150 × N × (N−1), cap 20), the milestone rule (attribute point
// every 4th level), and the party-order modal queue. The tests cover the
// pure display logic and the modal's state machine only — rendering itself
// is verified by screenshot evidence per the UI story testing standard.
// No engine code runs here; the modal is constructed with a nil document,
// which degrades it to a pure state machine, and a minimal fake entity
// store stands in for the Entity Framework (dependency injection, ADR-0001).
package character_sheet_test

import (
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/character_sheet"
	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// --- XP progress bar (AC-4 and QA edge cases) ------------------------------

// AC-4: GIVEN a level 4 character (Lxp = 150 × 4 × 3 = 1800) with 2400
// total XP, WHEN the character sheet is opened, THEN the XP bar fill is at
// 50% (±2% tolerance).
//
// Note on the story's expected value: the GDD example states Xnext = 1500
// for level 4, which yields 40%. That conflicts with the governing formula
// TR-stats-005 / ADR-0006 — XP_to_next = Level × 300, so at level 4 Xnext
// = 1200 (1500 is the level-5 threshold, 5 × 300). The story's own QA case
// flags this ("Wait — GDD says Xnext = 1500 for this example"). The
// implementation composes stats.XPToNext() rather than re-implementing the
// XP math (story rule), so the fill is (2400 − 1800) / 1200 × 100 = 50%.
func TestXPProgressPct_AC4(t *testing.T) {
	c := stats.StatsComponent{Level: 4, XP: 2400}

	got := character_sheet.XPProgressPct(c)

	if got < 48 || got > 52 {
		t.Errorf("XPProgressPct(level 4, 2400 XP) = %.2f%%, want 50%% (±2%%)", got)
	}
}

// QA edge case: GIVEN a level 1 character with 0 XP, THEN the bar shows 0%
// (0 XP toward the 300 XP threshold).
func TestXPProgressPct_Level1_ZeroXP(t *testing.T) {
	c := stats.StatsComponent{Level: 1}

	if got := character_sheet.XPProgressPct(c); got != 0 {
		t.Errorf("XPProgressPct(level 1, 0 XP) = %.2f%%, want 0%%", got)
	}
}

// QA edge case: GIVEN a level 1 character with exactly 300 XP (the level 2
// threshold), THEN the bar shows 100% — ready to level.
func TestXPProgressPct_Level1_ReadyToLevel(t *testing.T) {
	c := stats.StatsComponent{Level: 1, XP: 300}

	if got := character_sheet.XPProgressPct(c); got != 100 {
		t.Errorf("XPProgressPct(level 1, 300 XP) = %.2f%%, want 100%%", got)
	}
}

// QA edge case: GIVEN a level 20 character (level cap), THEN the bar is
// full — XP accumulates past the cap but no further level-ups occur.
func TestXPProgressPct_LevelCap(t *testing.T) {
	c := stats.StatsComponent{Level: 20, XP: 57000}

	if got := character_sheet.XPProgressPct(c); got != 100 {
		t.Errorf("XPProgressPct(level 20) = %.2f%%, want 100%% at cap", got)
	}
}

// QA edge case: GIVEN XP beyond the next level threshold, THEN the fill
// clamps to 100% (overflow is preserved in the component by AwardXP; the
// bar never overflows visually).
func TestXPProgressPct_OverflowClamped(t *testing.T) {
	// Level 4: threshold for level 5 is 3000; 4000 XP is 1200 past it.
	c := stats.StatsComponent{Level: 4, XP: 4000}

	if got := character_sheet.XPProgressPct(c); got != 100 {
		t.Errorf("XPProgressPct(level 4, 4000 XP) = %.2f%%, want clamped 100%%", got)
	}
}

// QA edge case: GIVEN total_xp == xp_for_current_level (just leveled),
// THEN the bar shows 0%.
func TestXPProgressPct_JustLeveled(t *testing.T) {
	c := stats.StatsComponent{Level: 5, XP: 3000} // TotalXPForLevel(5) == 3000

	if got := character_sheet.XPProgressPct(c); got != 0 {
		t.Errorf("XPProgressPct(level 5, 3000 XP) = %.2f%%, want 0%% (just leveled)", got)
	}
}

// Defensive: GIVEN a component with no level (Level 0, outside the
// 1-20 contract), THEN the percentage is 0 and the formula does not divide
// by zero (XPToNext() == 0 at level 0).
func TestXPProgressPct_LevelZero_NoThreshold(t *testing.T) {
	c := stats.StatsComponent{Level: 0, XP: 100}

	if got := character_sheet.XPProgressPct(c); got != 0 {
		t.Errorf("XPProgressPct(level 0) = %.2f%%, want 0%% (no threshold)", got)
	}
}

// XPProgressDisplay: GIVEN a character with accumulated XP, THEN the label
// reads "current / needed XP" — comma-grouped totals, per the GDD display
// format. The needed value is stats.XPToNext() (Level × 300, TR-stats-005):
// 1200 at level 4, 1500 at level 5 — the GDD example's "1500" only holds
// at level 5 (see TestXPProgressPct_AC4 for the example discrepancy).
func TestXPProgressDisplay_Format(t *testing.T) {
	tests := []struct {
		name string
		c    stats.StatsComponent
		want string
	}{
		{
			name: "level 4, 2400 XP",
			c:    stats.StatsComponent{Level: 4, XP: 2400},
			want: "2,400 / 1,200 XP",
		},
		{
			name: "level 5, 3000 XP",
			c:    stats.StatsComponent{Level: 5, XP: 3000},
			want: "3,000 / 1,500 XP",
		},
		{
			name: "level 1, 0 XP",
			c:    stats.StatsComponent{Level: 1},
			want: "0 / 300 XP",
		},
		{
			name: "level 19, 51000 XP",
			c:    stats.StatsComponent{Level: 19, XP: 51000},
			want: "51,000 / 5,700 XP",
		},
		{
			name: "no comma below 1000",
			c:    stats.StatsComponent{Level: 2, XP: 500},
			want: "500 / 600 XP",
		},
	}
	for _, tt := range tests {
		if got := character_sheet.XPProgressDisplay(tt.c); got != tt.want {
			t.Errorf("%s: XPProgressDisplay = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// QA edge case: GIVEN a level 20 character (cap), THEN the XP label reads
// "Max Level" — no threshold is shown because no further level-ups occur.
func TestXPProgressDisplay_LevelCap(t *testing.T) {
	c := stats.StatsComponent{Level: 20, XP: 60000}

	if got := character_sheet.XPProgressDisplay(c); got != "Max Level" {
		t.Errorf("XPProgressDisplay(level 20) = %q, want %q", got, "Max Level")
	}
}

// LevelUpHeader: GIVEN a level-up from 4 to 5, THEN the modal header reads
// "Level Up — Level 4 → 5" (em dash, actual levels).
func TestLevelUpHeader_SingleLevel(t *testing.T) {
	if got := character_sheet.LevelUpHeader(4, 5); got != "Level Up — Level 4 → 5" {
		t.Errorf("LevelUpHeader(4, 5) = %q, want %q", got, "Level Up — Level 4 → 5")
	}
}

// QA edge case (multi-level): GIVEN a level-up from 3 to 5 in one award,
// THEN the header renders the true range "Level Up — Level 3 → 5" — not
// derived from the new level plus one.
func TestLevelUpHeader_MultiLevel(t *testing.T) {
	if got := character_sheet.LevelUpHeader(3, 5); got != "Level Up — Level 3 → 5" {
		t.Errorf("LevelUpHeader(3, 5) = %q, want %q", got, "Level Up — Level 3 → 5")
	}
}

// --- Level-up modal state machine (AC-18) ----------------------------------

// fakeStore is a minimal EntityStore for tests: it stores components by
// type name and entity ID and no-ops the rest of the interface. The modal
// only reads and writes Stats components through GetComponentData /
// SetComponentData (the generic entity.GetComponent/SetComponent helpers
// fall back to these on non-EntityStoreImpl stores).
type fakeStore struct {
	components map[string]map[entity.GameEntityID]any
}

func newFakeStore() *fakeStore {
	return &fakeStore{components: map[string]map[entity.GameEntityID]any{}}
}

func (f *fakeStore) GetComponentData(id entity.GameEntityID, componentType string) (any, bool) {
	v, ok := f.components[componentType][id]
	return v, ok
}

func (f *fakeStore) SetComponentData(id entity.GameEntityID, componentType string, component any) error {
	if f.components[componentType] == nil {
		f.components[componentType] = map[entity.GameEntityID]any{}
	}
	f.components[componentType][id] = component
	return nil
}

func (f *fakeStore) CreateEntity(entityType entity.EntityType, displayName string, position matrix.Vec3) (entity.GameEntityID, error) {
	return 0, nil
}

func (f *fakeStore) DestroyEntity(id entity.GameEntityID) error { return nil }

func (f *fakeStore) RemoveComponentData(id entity.GameEntityID, componentType string) error {
	return nil
}

func (f *fakeStore) QueryEntities(q entity.Query) []entity.GameEntityID { return nil }

func (f *fakeStore) UpdatePosition(id entity.GameEntityID, newPosition matrix.Vec3) error {
	return nil
}

func (f *fakeStore) EntityState(id entity.GameEntityID) (entity.EntityState, bool) {
	return 0, false
}

func (f *fakeStore) Serialize() ([]byte, error) { return nil, nil }

func (f *fakeStore) Deserialize(data []byte) error { return nil }

// putStats stores a Stats component on the fake store under the type name
// the generic entity helpers resolve for stats.StatsComponent.
func putStats(t *testing.T, store *fakeStore, id entity.GameEntityID, c stats.StatsComponent) {
	t.Helper()
	if err := store.SetComponentData(id, "StatsComponent", c); err != nil {
		t.Fatalf("SetComponentData: %v", err)
	}
}

// getStats reads a Stats component back from the fake store.
func getStats(t *testing.T, store *fakeStore, id entity.GameEntityID) stats.StatsComponent {
	t.Helper()
	v, ok := store.GetComponentData(id, "StatsComponent")
	if !ok {
		t.Fatalf("entity %d has no Stats component in fake store", id)
	}
	c, ok := v.(stats.StatsComponent)
	if !ok {
		t.Fatalf("entity %d component is %T, want stats.StatsComponent", id, v)
	}
	return c
}

// postAwardComponent builds the post-award Stats component a level-up
// leaves behind: level 4 with 1800 total XP and one unallocated milestone
// point (the state AllocatePoint reads).
func postAwardComponent() stats.StatsComponent {
	return stats.StatsComponent{
		Attributes:    [6]int{10, 10, 10, 10, 10, 10},
		Level:         4,
		XP:            1800,
		UnspentPoints: 1,
	}
}

// newModal builds the modal as a pure state machine (nil document) over a
// fresh fake store, with an optional party roster.
func newModal(roster character_sheet.PartyRoster) (*character_sheet.LevelUpModal, *fakeStore) {
	store := newFakeStore()
	return character_sheet.NewLevelUpModal(nil, store, roster), store
}

// QA test case (AC-18): GIVEN a 4 → 5 level-up — not a 4th-level milestone
// (4/8/12/16 grant points; 5 does not) — WHEN the modal is shown, THEN
// Confirm is enabled immediately and closes the modal.
func TestLevelUpModal_NoPoint_ConfirmImmediatelyEnabled(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	putStats(t, store, id, stats.StatsComponent{Level: 5, XP: 3000})

	modal.Show(id, 4, 5, 5, false)

	if modal.State() != character_sheet.LevelUpWaitingConfirm {
		t.Fatalf("State() = %v, want LevelUpWaitingConfirm (no point required)", modal.State())
	}
	if !modal.Active() {
		t.Fatal("Active() = false, want true after Show")
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() = %v, want nil (no allocation to commit)", err)
	}
	if modal.Active() {
		t.Error("Active() = true, want false after Confirm")
	}
}

// QA test case (AC-18): GIVEN a 3 → 4 level-up (4th-level milestone), WHEN
// the modal is shown, THEN Confirm is disabled until the attribute point is
// allocated; the allocation is written back to the entity's Stats component
// on Confirm, preserving the post-award level/XP.
func TestLevelUpModal_PointRequired_ConfirmDisabledUntilAllocated(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	putStats(t, store, id, postAwardComponent())

	modal.Show(id, 3, 4, 4, true)

	if modal.State() != character_sheet.LevelUpShowing {
		t.Fatalf("State() = %v, want LevelUpShowing (point pending)", modal.State())
	}

	// Confirm before allocation is rejected; the modal stays up.
	if err := modal.Confirm(); err == nil {
		t.Fatal("Confirm() before allocation: want error, got nil")
	}
	if !modal.Active() {
		t.Fatal("Active() = false, want true (modal is mandatory)")
	}

	if err := modal.AllocatePoint(stats.STR); err != nil {
		t.Fatalf("AllocatePoint(STR) = %v, want nil", err)
	}
	if modal.State() != character_sheet.LevelUpWaitingConfirm {
		t.Fatalf("State() = %v, want LevelUpWaitingConfirm after allocation", modal.State())
	}

	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}

	// The allocation reached the store; the award's level/XP survived.
	c := getStats(t, store, id)
	if c.Attributes[stats.STR] != 11 {
		t.Errorf("STR = %d, want 11 (10 + allocated point)", c.Attributes[stats.STR])
	}
	if c.UnspentPoints != 0 {
		t.Errorf("UnspentPoints = %d, want 0 (point spent)", c.UnspentPoints)
	}
	if c.Level != 4 || c.XP != 1800 {
		t.Errorf("Level/XP = %d/%d, want 4/1800 (post-award state preserved)", c.Level, c.XP)
	}
}

// QA edge case: GIVEN a level-up modal with no attribute point, THEN the
// allocation is rejected (there is nothing to allocate) and Confirm stays
// available.
func TestLevelUpModal_AllocatePoint_NoPointRequired_Rejected(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	putStats(t, store, id, stats.StatsComponent{Level: 5, XP: 3000})

	modal.Show(id, 4, 5, 5, false)

	if err := modal.AllocatePoint(stats.STR); err == nil {
		t.Fatal("AllocatePoint on a no-point level-up: want error, got nil")
	}
	if modal.State() != character_sheet.LevelUpWaitingConfirm {
		t.Errorf("State() = %v, want LevelUpWaitingConfirm (allocation rejected)", modal.State())
	}
}

// QA edge case: GIVEN a milestone level-up whose character has no
// unallocated points (the point was banked and spent elsewhere), THEN the
// allocation fails with the stats package's error and the modal stays in
// LevelUpShowing — the player cannot confirm.
func TestLevelUpModal_AllocatePoint_NoPoints_Rejected(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	c := postAwardComponent()
	c.UnspentPoints = 0
	putStats(t, store, id, c)

	modal.Show(id, 3, 4, 4, true)

	if err := modal.AllocatePoint(stats.STR); err == nil {
		t.Fatal("AllocatePoint with 0 points: want error, got nil")
	}
	if modal.State() != character_sheet.LevelUpShowing {
		t.Errorf("State() = %v, want LevelUpShowing (allocation failed)", modal.State())
	}
	if modal.Confirm() == nil {
		t.Fatal("Confirm() with unallocated point: want error, got nil")
	}
}

// QA edge case: GIVEN a milestone level-up whose attribute is already at
// the mortal maximum (18, CR-1), THEN the allocation fails and the modal
// stays awaiting allocation.
func TestLevelUpModal_AllocatePoint_AtMax18_Rejected(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	c := postAwardComponent()
	c.Attributes[stats.CON] = 18
	putStats(t, store, id, c)

	modal.Show(id, 3, 4, 4, true)

	if err := modal.AllocatePoint(stats.CON); err == nil {
		t.Fatal("AllocatePoint on 18 CON: want error, got nil")
	}
	if modal.State() != character_sheet.LevelUpShowing {
		t.Errorf("State() = %v, want LevelUpShowing (allocation failed)", modal.State())
	}
}

// GIVEN a milestone level-up whose point is already allocated, WHEN the
// player clicks another + button, THEN the second allocation is rejected —
// one point per level-up.
func TestLevelUpModal_AllocatePoint_Twice_Rejected(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	putStats(t, store, id, postAwardComponent())

	modal.Show(id, 3, 4, 4, true)
	if err := modal.AllocatePoint(stats.STR); err != nil {
		t.Fatalf("first AllocatePoint = %v, want nil", err)
	}
	if err := modal.AllocatePoint(stats.DEX); err == nil {
		t.Fatal("second AllocatePoint: want error, got nil")
	}
	if modal.State() != character_sheet.LevelUpWaitingConfirm {
		t.Errorf("State() = %v, want LevelUpWaitingConfirm", modal.State())
	}
}

// GIVEN no active modal, WHEN Confirm or AllocatePoint is called, THEN an
// error is returned (no spurious state changes).
func TestLevelUpModal_NoActiveModal_Rejects(t *testing.T) {
	modal, _ := newModal(nil)

	if modal.Active() {
		t.Fatal("Active() = true on a fresh modal, want false")
	}
	if modal.Confirm() == nil {
		t.Fatal("Confirm() on hidden modal: want error, got nil")
	}
	if err := modal.AllocatePoint(stats.STR); err == nil {
		t.Fatal("AllocatePoint on hidden modal: want error, got nil")
	}
}

// GIVEN a level-up without an attribute point, WHEN Confirm commits it,
// THEN the component is not written (nothing was allocated — the level-up
// itself was committed by the awarder, not the modal).
func TestLevelUpModal_Confirm_NoAllocation_DoesNotWrite(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	before := stats.StatsComponent{Level: 5, XP: 3000, Attributes: [6]int{10, 12, 10, 10, 10, 10}}
	putStats(t, store, id, before)

	modal.Show(id, 4, 5, 5, false)
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}

	after := getStats(t, store, id)
	if after != before {
		t.Errorf("component changed without allocation: got %+v, want %+v", after, before)
	}
}

// GIVEN a level-up for an entity without a Stats component, WHEN the modal
// is shown, THEN the level-up is skipped (nothing to display or allocate).
func TestLevelUpModal_Show_NoStatsComponent_Skipped(t *testing.T) {
	modal, _ := newModal(nil)
	const id = entity.GameEntityID(99) // no component in the store

	modal.Show(id, 3, 4, 4, true)

	if modal.Active() {
		t.Error("Active() = true, want false (no Stats component to level up)")
	}
}

// testRoster is a PartyRoster over a fixed member order (player first,
// companions by join order — GDD Edge Cases).
type testRoster []entity.GameEntityID

func (r testRoster) Order() []entity.GameEntityID { return r }

// GDD Edge Cases: GIVEN multiple simultaneous level-ups (shared XP event),
// WHEN modals queue, THEN they display in party order — player first, then
// companions by join order — regardless of event arrival order.
func TestLevelUpModal_Queue_PartyOrder(t *testing.T) {
	const (
		player     = entity.GameEntityID(1)
		companion1 = entity.GameEntityID(2)
		companion2 = entity.GameEntityID(3)
	)
	roster := testRoster{player, companion1, companion2}
	modal, store := newModal(roster)
	putStats(t, store, player, postAwardComponent())
	putStats(t, store, companion1, postAwardComponent())
	putStats(t, store, companion2, postAwardComponent())

	// The player's modal shows immediately; the companions arrive while it
	// is active — companion2 before companion1 — and must be queued in
	// party order [companion1, companion2].
	modal.Show(player, 3, 4, 4, true)
	modal.Show(companion2, 3, 4, 4, true)
	modal.Show(companion1, 3, 4, 4, true)

	// Player confirms; the next modal must be companion1 (party order).
	if err := modal.AllocatePoint(stats.STR); err != nil {
		t.Fatalf("player AllocatePoint = %v, want nil", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("player Confirm = %v, want nil", err)
	}
	if !modal.Active() {
		t.Fatal("modal inactive, want companion1's modal displayed")
	}

	// The displayed modal allocates on companion1's component: confirm it,
	// then companion2's must follow.
	if err := modal.AllocatePoint(stats.DEX); err != nil {
		t.Fatalf("companion1 AllocatePoint = %v, want nil", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("companion1 Confirm = %v, want nil", err)
	}
	if !modal.Active() {
		t.Fatal("modal inactive, want companion2's modal displayed")
	}
	if err := modal.AllocatePoint(stats.CON); err != nil {
		t.Fatalf("companion2 AllocatePoint = %v, want nil", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("companion2 Confirm = %v, want nil", err)
	}
	if modal.Active() {
		t.Fatal("modal still active after the last queued level-up")
	}

	// Party order proof: companion1 allocated before companion2, and the
	// player allocated first.
	c1 := getStats(t, store, companion1)
	if c1.Attributes[stats.DEX] != 11 || c1.Attributes[stats.STR] != 10 {
		t.Errorf("companion1 attributes = %+v, want DEX 11, STR 10 (allocated second)", c1.Attributes)
	}
	c2 := getStats(t, store, companion2)
	if c2.Attributes[stats.CON] != 11 || c2.Attributes[stats.STR] != 10 {
		t.Errorf("companion2 attributes = %+v, want CON 11, STR 10 (allocated third)", c2.Attributes)
	}
	cp := getStats(t, store, player)
	if cp.Attributes[stats.STR] != 11 {
		t.Errorf("player STR = %d, want 11 (allocated first)", cp.Attributes[stats.STR])
	}
}

// GIVEN multiple simultaneous level-ups without a party roster, WHEN modals
// queue, THEN they display in arrival order (the roster arrives with the
// Party System, Story 008).
func TestLevelUpModal_Queue_ArrivalOrder_WithoutRoster(t *testing.T) {
	const (
		a = entity.GameEntityID(1)
		b = entity.GameEntityID(2)
		c = entity.GameEntityID(3)
	)
	modal, store := newModal(nil)
	putStats(t, store, a, postAwardComponent())
	putStats(t, store, b, postAwardComponent())
	putStats(t, store, c, postAwardComponent())

	modal.Show(a, 3, 4, 4, true)
	modal.Show(c, 3, 4, 4, true)
	modal.Show(b, 3, 4, 4, true)

	// Confirm a; the next modal must be c (arrival order, no roster).
	if err := modal.AllocatePoint(stats.STR); err != nil {
		t.Fatalf("a AllocatePoint = %v, want nil", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("a Confirm = %v, want nil", err)
	}
	if err := modal.AllocatePoint(stats.DEX); err != nil {
		t.Fatalf("c AllocatePoint = %v, want nil (c displayed before b)", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("c Confirm = %v, want nil", err)
	}
	if err := modal.AllocatePoint(stats.CON); err != nil {
		t.Fatalf("b AllocatePoint = %v, want nil", err)
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("b Confirm = %v, want nil", err)
	}

	if got := getStats(t, store, c).Attributes[stats.DEX]; got != 11 {
		t.Errorf("c DEX = %d, want 11 (c was second in arrival order)", got)
	}
	if got := getStats(t, store, b).Attributes[stats.CON]; got != 11 {
		t.Errorf("b CON = %d, want 11 (b was last in arrival order)", got)
	}
}

// QA edge case: GIVEN a level-up to the level cap (19 → 20), THEN no
// attribute point is granted (the cap level 20 is excluded from the
// milestone rule) and Confirm is immediately enabled.
func TestLevelUpModal_CapLevel_NoPoint(t *testing.T) {
	modal, store := newModal(nil)
	const id = entity.GameEntityID(1)
	putStats(t, store, id, stats.StatsComponent{Level: 20, XP: 57000})

	modal.Show(id, 19, 20, 4, false)

	if modal.State() != character_sheet.LevelUpWaitingConfirm {
		t.Fatalf("State() = %v, want LevelUpWaitingConfirm (level 20 grants no point)", modal.State())
	}
	if err := modal.Confirm(); err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}
}

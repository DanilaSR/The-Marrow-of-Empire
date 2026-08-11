// Unit tests for Story STAT-004: Character Creation (Point-Buy).
//
// Evidence path: production/epics/character-stats-progression/story-003-character-creation.md
// Tests map to the acceptance criteria (AC-1 .. AC-4) and verify TR-stats-004:
// 30 points, all attributes start at 8, max starting 16, cost N-8 — per
// ADR-0006. Extra tests cover initial state, insufficient points, invalid
// indices, and SpentPoints tracking.
package stats_test

import (
	"testing"

	"marrow-of-empire/stats"
)

// raiseN raises attr times times, failing the test on the first error.
func raiseN(t *testing.T, ps *stats.PointBuyState, attr, times int) {
	t.Helper()
	for i := 0; i < times; i++ {
		if err := ps.Raise(attr); err != nil {
			t.Fatalf("Raise(%d) attempt %d: %v", attr, i+1, err)
		}
	}
}

// AC-1: GIVEN 30 starting points, WHEN the player distributes STR 16, DEX 12,
// CON 14, INT 10, WIS 10, CHA 8, THEN 22 points are spent (8+4+6+2+2+0) and
// 8 remain.
func TestPointBuyDistribution(t *testing.T) {
	ps := stats.NewPointBuyState()
	raiseN(t, ps, stats.STR, 8) // 8 → 16
	raiseN(t, ps, stats.DEX, 4) // 8 → 12
	raiseN(t, ps, stats.CON, 6) // 8 → 14
	raiseN(t, ps, stats.INT, 2) // 8 → 10
	raiseN(t, ps, stats.WIS, 2) // 8 → 10

	want := [...]int{16, 12, 14, 10, 10, 8}
	if ps.Attributes != want {
		t.Errorf("attributes = %v, want %v", ps.Attributes, want)
	}
	if got := ps.SpentPoints(); got != 22 {
		t.Errorf("SpentPoints() = %d, want 22", got)
	}
	if got := ps.PointsRemaining; got != 8 {
		t.Errorf("PointsRemaining = %d, want 8", got)
	}
}

// AC-2: GIVEN all attributes start at 8, WHEN the player raises an attribute
// to N, THEN cost is N-8 points. Also verifies each raise spends exactly 1
// point (SpentPoints tracking).
func TestCostFormula(t *testing.T) {
	ps := stats.NewPointBuyState()
	raiseN(t, ps, stats.STR, 8) // 8 → 16 costs 8

	if got := ps.SpentPoints(); got != 8 {
		t.Errorf("SpentPoints() = %d, want 8 (cost of STR 16 from 8)", got)
	}

	ps = stats.NewPointBuyState()
	for n := 1; n <= 4; n++ {
		if err := ps.Raise(stats.INT); err != nil {
			t.Fatalf("Raise(INT) attempt %d: %v", n, err)
		}
		if got := ps.SpentPoints(); got != n {
			t.Errorf("SpentPoints() after %d raises = %d, want %d", n, got, n)
		}
	}
	if got := ps.Attributes[stats.INT]; got != 12 {
		t.Errorf("INT = %d, want 12 (8 + 4 raises)", got)
	}
}

// AC-3: GIVEN the player attempts to raise an attribute above 16 at creation,
// THEN it is rejected (max starting attribute) and the state is unchanged.
func TestMaxStartingAttributeRejected(t *testing.T) {
	ps := stats.NewPointBuyState()
	raiseN(t, ps, stats.CON, 8) // 8 → 16

	if err := ps.Raise(stats.CON); err == nil {
		t.Fatal("Raise(CON) above 16 succeeded, want rejection")
	}
	if got := ps.Attributes[stats.CON]; got != 16 {
		t.Errorf("CON = %d, want unchanged 16", got)
	}
	if got := ps.PointsRemaining; got != 22 {
		t.Errorf("PointsRemaining = %d, want unchanged 22", got)
	}
	if ps.CanRaise(stats.CON) {
		t.Error("CanRaise(CON) = true at 16, want false")
	}
}

// AC-4: GIVEN the player confirms creation, WHEN the Stats component is
// created, THEN all six attributes match the allocation and derived stats
// are computed.
func TestCommit(t *testing.T) {
	ps := stats.NewPointBuyState()
	raiseN(t, ps, stats.STR, 8) // 16
	raiseN(t, ps, stats.DEX, 4) // 12
	raiseN(t, ps, stats.CON, 6) // 14

	c := ps.Commit()

	want := [...]int{16, 12, 14, 8, 8, 8}
	if c.Attributes != want {
		t.Errorf("committed attributes = %v, want %v", c.Attributes, want)
	}
	if c.Level != 1 || c.XP != 0 {
		t.Errorf("committed Level/XP = %d/%d, want 1/0", c.Level, c.XP)
	}
	// Derived stats from committed attributes: MaxHealth 20 + (14×2) + (1×2) = 50,
	// CarryCapacity (16×10)+30 = 190.
	if got := c.MaxHealth(); got != 50 {
		t.Errorf("MaxHealth() = %d, want 50", got)
	}
	if got := c.CarryCapacity(); got != 190 {
		t.Errorf("CarryCapacity() = %d, want 190", got)
	}
}

// Initial state: GIVEN a fresh point-buy state, THEN 30 points remain, all
// attributes are at 8, and nothing has been spent.
func TestInitialState(t *testing.T) {
	ps := stats.NewPointBuyState()
	want := [...]int{8, 8, 8, 8, 8, 8}
	if ps.Attributes != want {
		t.Errorf("attributes = %v, want all 8", ps.Attributes)
	}
	if got := ps.PointsRemaining; got != stats.StartingPoints {
		t.Errorf("PointsRemaining = %d, want %d", got, stats.StartingPoints)
	}
	if got := ps.SpentPoints(); got != 0 {
		t.Errorf("SpentPoints() = %d, want 0", got)
	}
}

// Insufficient points: GIVEN 0 points remaining, WHEN Raise is called, THEN
// it is rejected and the state is unchanged.
func TestRaiseInsufficientPoints(t *testing.T) {
	ps := stats.NewPointBuyState()
	raiseN(t, ps, stats.STR, 8) // 16
	raiseN(t, ps, stats.DEX, 8) // 16
	raiseN(t, ps, stats.CON, 8) // 16
	raiseN(t, ps, stats.INT, 6) // 14 — exactly 30 spent

	if got := ps.PointsRemaining; got != 0 {
		t.Fatalf("PointsRemaining = %d, want 0", got)
	}
	if err := ps.Raise(stats.WIS); err == nil {
		t.Fatal("Raise(WIS) with 0 points succeeded, want rejection")
	}
	if got := ps.Attributes[stats.WIS]; got != 8 {
		t.Errorf("WIS = %d, want unchanged 8", got)
	}
	if ps.CanRaise(stats.WIS) {
		t.Error("CanRaise(WIS) = true with 0 points, want false")
	}
}

// Invalid index: GIVEN an out-of-range attribute index, WHEN Raise or
// CanRaise is called, THEN it is rejected without spending points.
func TestRaiseInvalidIndex(t *testing.T) {
	ps := stats.NewPointBuyState()
	for _, attr := range []int{-1, stats.NumAttributes} {
		if err := ps.Raise(attr); err == nil {
			t.Errorf("Raise(%d) succeeded, want out-of-range error", attr)
		}
		if ps.CanRaise(attr) {
			t.Errorf("CanRaise(%d) = true, want false", attr)
		}
	}
	if got := ps.PointsRemaining; got != stats.StartingPoints {
		t.Errorf("PointsRemaining = %d, want unchanged %d", got, stats.StartingPoints)
	}
}

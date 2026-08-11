// Point-buy character creation per ADR-0006 and TR-stats-004: 30 points,
// all attributes start at 8, maximum starting attribute 16, linear cost —
// raising an attribute to N costs N-8 points.
//
// PointBuyState tracks an in-progress allocation; Commit finalizes it into
// a StatsComponent (level 1, 0 XP) for the character creation flow.
//
// Story: STAT-004 (production/epics/character-stats-progression/
// story-003-character-creation.md).
package stats

import (
	"errors"
	"fmt"
)

// StartingPoints is the total point-buy budget per ADR-0006 and TR-stats-004
// (story-003 AC-1).
const StartingPoints = 30

// BaseAttribute is the value every attribute starts at during point-buy
// creation per ADR-0006 and TR-stats-004 (story-003 AC-2).
const BaseAttribute = 8

// MaxStartingAttribute is the highest value an attribute may be raised to
// during point-buy creation per ADR-0006 and TR-stats-004 (story-003 AC-3).
const MaxStartingAttribute = 16

// PointBuyState tracks an in-progress character creation allocation: the
// points remaining to spend and the six attributes in canonical order
// [STR, DEX, CON, INT, WIS, CHA] per ADR-0006.
type PointBuyState struct {
	PointsRemaining int
	Attributes      [NumAttributes]int
}

// NewPointBuyState returns a fresh point-buy allocation with StartingPoints
// remaining and every attribute at BaseAttribute per ADR-0006.
func NewPointBuyState() *PointBuyState {
	ps := &PointBuyState{PointsRemaining: StartingPoints}
	for i := range ps.Attributes {
		ps.Attributes[i] = BaseAttribute
	}
	return ps
}

// CanRaise reports whether the given attribute can be raised by one point:
// the index must be valid, the attribute must be below MaxStartingAttribute,
// and at least one point must remain (story-003 AC-3).
func (ps *PointBuyState) CanRaise(attr int) bool {
	if attr < 0 || attr >= NumAttributes {
		return false
	}
	return ps.Attributes[attr] < MaxStartingAttribute && ps.PointsRemaining >= 1
}

// Raise spends one point to raise the given attribute by 1 per the linear
// cost rule (N-8 points to reach N). It fails when the attribute index is
// invalid, no points remain, or the attribute is already at
// MaxStartingAttribute; on failure the state is unchanged.
//
// Example:
//
//	ps := stats.NewPointBuyState()
//	err := ps.Raise(stats.STR) // STR 9, 29 points remain
func (ps *PointBuyState) Raise(attr int) error {
	if attr < 0 || attr >= NumAttributes {
		return fmt.Errorf("stats: attribute index %d out of range", attr)
	}
	if ps.PointsRemaining < 1 {
		return errors.New("stats: no point-buy points remaining")
	}
	if ps.Attributes[attr] >= MaxStartingAttribute {
		return fmt.Errorf("stats: attribute %d already at creation maximum %d", attr, MaxStartingAttribute)
	}
	ps.PointsRemaining--
	ps.Attributes[attr]++
	return nil
}

// SpentPoints returns the number of points spent so far:
// StartingPoints - PointsRemaining (story-003 AC-1).
func (ps *PointBuyState) SpentPoints() int {
	return StartingPoints - ps.PointsRemaining
}

// Commit finalizes the allocation into a StatsComponent at level 1 with
// 0 XP and the allocated attributes (story-003 AC-4). Derived stats are
// computed on demand by the StatsComponent methods (ADR-0006), so the
// returned component is immediately usable.
func (ps *PointBuyState) Commit() StatsComponent {
	return StatsComponent{
		Attributes: ps.Attributes,
		Level:      1,
		XP:         0,
	}
}

// Unit tests for INT-3 combat wiring's pure parts: the encounter trigger
// distance and the slice stat blocks.
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-3)
package game_test

import (
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
	"marrow-of-empire/game"
)

func TestDistanceXZ_MineTriggerBoundary(t *testing.T) {
	// The mine sits at worldMinePos (12, 18); the trigger is 4.0u.
	mine := matrix.NewVec2(12, 18)

	inside := matrix.NewVec3(13, 0.5, 19) // ~2.2u away
	if d := game.DistanceXZ(inside, mine); d >= 4.0 {
		t.Fatalf("inside-trigger distance must be < 4.0, got %v", d)
	}

	outside := matrix.NewVec3(8, 0.5, 14) // ~5.7u away
	if d := game.DistanceXZ(outside, mine); d <= 4.0 {
		t.Fatalf("outside-trigger distance must be > 4.0, got %v", d)
	}
}

func TestDistanceXZ_ExactPositionZero(t *testing.T) {
	if d := game.DistanceXZ(matrix.NewVec3(12, 99, 18), matrix.NewVec2(12, 18)); d != 0 {
		t.Fatalf("distance at the exact point must be 0 (Y ignored), got %v", d)
	}
}

func TestSlicePlayerStats_PointBuy13s(t *testing.T) {
	c := game.SlicePlayerStats()
	want := entity.Stats{
		Strength: 13, Dexterity: 13, Constitution: 13,
		Intelligence: 13, Wisdom: 13, Charisma: 13,
		MaxHealth: 16, ActionPoints: 5,
	}
	if c != want {
		t.Fatalf("player stats = %+v, want %+v", c, want)
	}
}

func TestSliceWolfStats_ExpectedBlock(t *testing.T) {
	c := game.SliceWolfStats()
	// STR 12, DEX 14, CON 10, INT 8, WIS 10, CHA 8 (authoring order).
	want := entity.Stats{
		Strength: 12, Dexterity: 14, Constitution: 10,
		Intelligence: 8, Wisdom: 10, Charisma: 8,
		MaxHealth: 8, ActionPoints: 6,
	}
	if c != want {
		t.Fatalf("wolf stats = %+v, want %+v", c, want)
	}
}

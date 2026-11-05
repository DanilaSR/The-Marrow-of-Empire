// Unit tests for INT-1 game world bootstrap: MoveToward movement math.
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-1)
// MoveToward is exported from the game package specifically so this math is
// testable without the engine (investigation_journal pattern).
package game_test

import (
	"math"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/game"
)

const (
	floatTolerance = 0.0001
)

func vec3ApproxEqual(t *testing.T, got, want matrix.Vec3, msg string) {
	t.Helper()
	if math.Abs(float64(got.X()-want.X())) > floatTolerance ||
		math.Abs(float64(got.Y()-want.Y())) > floatTolerance ||
		math.Abs(float64(got.Z()-want.Z())) > floatTolerance {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

func TestMoveToward_ZeroDt_StaysPut(t *testing.T) {
	current := matrix.Vec3Zero()
	target := matrix.NewVec3(10, 0, 0)

	next, arrived := game.MoveToward(current, target, 5, 0, 0.25)

	vec3ApproxEqual(t, next, current, "zero dt must not move")
	if arrived {
		t.Fatal("zero dt with a distant target must not report arrival")
	}
}

func TestMoveToward_PartialStep_MovesProportionally(t *testing.T) {
	current := matrix.Vec3Zero()
	target := matrix.NewVec3(10, 0, 0)

	// 1 second at speed 4 → 4 world units toward the target.
	next, arrived := game.MoveToward(current, target, 4, 1, 0.25)

	vec3ApproxEqual(t, next, matrix.NewVec3(4, 0, 0), "partial step must move speed*dt along the direction")
	if arrived {
		t.Fatal("partial step with a distant target must not report arrival")
	}
}

func TestMoveToward_LargeStep_SnapsToTarget_NoOvershoot(t *testing.T) {
	current := matrix.Vec3Zero()
	target := matrix.NewVec3(10, 0, 0)

	// A step far larger than the distance must snap exactly to the target.
	next, arrived := game.MoveToward(current, target, 100, 1, 0.25)

	vec3ApproxEqual(t, next, target, "large step must snap to the target, never overshoot")
	if !arrived {
		t.Fatal("snap-to-target must report arrival")
	}
}

func TestMoveToward_AlreadyWithinEpsilon_ArrivesImmediately(t *testing.T) {
	current := matrix.NewVec3(10, 0, 0)
	target := matrix.NewVec3(10.2, 0, 0) // 0.2 away, inside epsilon 0.25

	next, arrived := game.MoveToward(current, target, 5, 1, 0.25)

	vec3ApproxEqual(t, next, current, "within-epsilon target must not move the mover")
	if !arrived {
		t.Fatal("within-epsilon target must report arrival")
	}
}

func TestMoveToward_PreservesY(t *testing.T) {
	current := matrix.NewVec3(0, 0.5, 0)
	target := matrix.NewVec3(10, 0, 0)

	next, _ := game.MoveToward(current, target, 4, 1, 0.25)

	if math.Abs(float64(next.Y()-0.5)) > floatTolerance {
		t.Fatalf("movement must preserve Y (ground plane): got Y %v, want 0.5", next.Y())
	}
}

func TestMoveToward_ExactStepLandsOnTarget(t *testing.T) {
	current := matrix.Vec3Zero()
	target := matrix.NewVec3(10, 0, 0)

	// A step landing exactly on the target reports arrival at the target.
	next, arrived := game.MoveToward(current, target, 5, 2, 0)

	vec3ApproxEqual(t, next, target, "exact-distance step must land on the target")
	if !arrived {
		t.Fatal("exact-distance step must report arrival")
	}
}

func TestMoveToward_DiagonalMovement_NormalizedDirection(t *testing.T) {
	current := matrix.Vec3Zero()
	target := matrix.NewVec3(10, 0, 10)

	// Distance is ~14.14; 1s at speed 5 moves 5 units along the diagonal.
	next, arrived := game.MoveToward(current, target, 5, 1, 0.25)
	wantStep := 5.0 / math.Sqrt(2.0)

	vec3ApproxEqual(t, next, matrix.NewVec3(float32(wantStep), 0, float32(wantStep)), "diagonal movement must follow the normalized direction")
	if arrived {
		t.Fatal("partial diagonal step must not report arrival")
	}
}

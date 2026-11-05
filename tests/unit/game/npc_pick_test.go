// Unit tests for INT-2 NPC click picking (game.PickNPC).
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-2)
package game_test

import (
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/game"
)

func TestPickNPC_NearestWithinRadius(t *testing.T) {
	positions := []matrix.Vec2{
		matrix.NewVec2(10, 0),
		matrix.NewVec2(1, 0),
		matrix.NewVec2(5, 5),
	}
	idx := game.PickNPC(matrix.NewVec2(1.2, 0), positions, 1.5)
	if idx != 1 {
		t.Fatalf("nearest NPC (index 1) must win, got %d", idx)
	}
}

func TestPickNPC_OutsideRadius(t *testing.T) {
	positions := []matrix.Vec2{matrix.NewVec2(10, 0)}
	if idx := game.PickNPC(matrix.NewVec2(0, 0), positions, 1.5); idx != -1 {
		t.Fatalf("click outside radius must return -1, got %d", idx)
	}
}

func TestPickNPC_ExactBoundary(t *testing.T) {
	positions := []matrix.Vec2{matrix.NewVec2(1.5, 0)}
	if idx := game.PickNPC(matrix.NewVec2(0, 0), positions, 1.5); idx != 0 {
		t.Fatalf("click exactly at radius boundary must match, got %d", idx)
	}
}

func TestPickNPC_EmptyPositions(t *testing.T) {
	if idx := game.PickNPC(matrix.NewVec2(0, 0), nil, 1.5); idx != -1 {
		t.Fatalf("no positions must return -1, got %d", idx)
	}
}

func TestPickNPC_TieKeepsFirst(t *testing.T) {
	positions := []matrix.Vec2{
		matrix.NewVec2(1, 0),
		matrix.NewVec2(0, 1),
	}
	// Both are exactly 1.0 away; strict less-than keeps the first.
	if idx := game.PickNPC(matrix.NewVec2(0, 0), positions, 1.5); idx != 0 {
		t.Fatalf("exact tie must keep the first NPC, got %d", idx)
	}
}

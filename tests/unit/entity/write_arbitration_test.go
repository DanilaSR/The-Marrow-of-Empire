// Unit tests for Story EF-004: Write Arbitration.
//
// Evidence path: production/epics/entity-framework/story-004-write-arbitration.md
// Each test function maps to one acceptance criterion (AC-1 .. AC-3) or one
// edge case from the story's QA section. The story's evidence table names the
// AC tests test_priority_resolution_lower_wins, test_equal_priority_last_writer_wins,
// and test_uncontested_write_passthrough; Go only discovers exported Test*
// functions (tests/README.md naming convention: Test[Feature]_[Scenario]_[Expected]),
// so the mapping is:
//
//	AC-1 -> TestPriorityResolution_LowerWins
//	AC-2 -> TestEqualPriority_LastWriterWins
//	AC-3 -> TestUncontestedWrite_Passthrough
package entity_test

import (
	"errors"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// AC-1: Combat (priority 5) writes Health.Current; VFX (priority 15) writes
// the same field in the same frame. The priority-5 value is applied and the
// priority-15 system receives an error wrapping ErrWriteRejected.
func TestPriorityResolution_LowerWins(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	// Combat (priority 5) writes Health.Current = 0 first.
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 0, Max: 100}, "Current"); err != nil {
		t.Fatalf("priority-5 write returned error: %v", err)
	}

	// VFX (priority 15) writes Health.Current = 50 in the same frame: the
	// write is discarded and the calling system receives an error.
	err = entity.SetComponentArbitrated(arbiter, "vfx", 15, id, entity.Health{Current: 50, Max: 100}, "Current")
	if err == nil {
		t.Fatal("priority-15 write must be rejected when priority-5 already wrote this frame")
	}
	if !errors.Is(err, entity.ErrWriteRejected) {
		t.Fatalf("rejection error %v must wrap entity.ErrWriteRejected", err)
	}

	// When the frame resolves, the priority-5 value is applied.
	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok {
		t.Fatal("Health component missing after arbitrated write")
	}
	if hp.Current != 0 {
		t.Fatalf("Health.Current = %d, want 0 (priority-5 write must win)", hp.Current)
	}
}

// AC-2: Two systems with equal priority write Health.Current, System B last.
// The last writer's value is applied and neither write errors (equal priority
// is not a conflict — ADR-0001 deterministic tie-breaking, no warning).
func TestEqualPriority_LastWriterWins(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	// System A (priority 10) writes first, System B (priority 10) writes last.
	if err := entity.SetComponentArbitrated(arbiter, "system_a", 10, id, entity.Health{Current: 10, Max: 100}, "Current"); err != nil {
		t.Fatalf("first equal-priority write returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "system_b", 10, id, entity.Health{Current: 20, Max: 100}, "Current"); err != nil {
		t.Fatalf("second equal-priority write returned error: %v", err)
	}

	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok || hp.Current != 20 {
		t.Fatalf("Health.Current = %+v, want 20 (last writer must win)", hp)
	}
}

// AC-3: A single system writes Health.Current; no other system touches the
// slot this frame, so the write passes straight through — applied
// immediately, readable without any frame-end commit call.
func TestUncontestedWrite_Passthrough(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 7, Max: 100}, "Current"); err != nil {
		t.Fatalf("uncontested write returned error: %v", err)
	}

	// The value is applied directly — no commit or frame-end call needed.
	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok || hp.Current != 7 {
		t.Fatalf("Health.Current = %+v, want 7 (uncontested write must apply directly)", hp)
	}

	// A second uncontested slot (a different field) in the same frame also
	// passes through untouched.
	if err := entity.SetComponentArbitrated(arbiter, "stats", 10, id, entity.Health{Current: 7, Max: 120}, "Max"); err != nil {
		t.Fatalf("uncontested write to another field returned error: %v", err)
	}
	hp, _ = entity.GetComponent[entity.Health](store, id)
	if hp.Max != 120 {
		t.Fatalf("Health.Max = %d, want 120", hp.Max)
	}
}

// A system may rewrite its own field any number of times in a frame: the
// writes apply and the record follows the latest write, without conflict.
func TestSameSystem_RewritesWithoutConflict(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 10, Max: 100}, "Current"); err != nil {
		t.Fatalf("first write returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 3, Max: 100}, "Current"); err != nil {
		t.Fatalf("same-system rewrite returned error: %v", err)
	}

	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok || hp.Current != 3 {
		t.Fatalf("Health.Current = %+v, want 3 (last same-system write applies)", hp)
	}
}

// Three systems with priorities 15, 5, and 3 write Health.Current, plus a
// fourth at 10. Only the lowest priority number (3) wins, regardless of write
// order; every other conflicting write is rejected.
func TestThreeSystemConflict_OnlyLowestPriorityWins(t *testing.T) {
	for _, order := range []string{"loser_first", "winner_first"} {
		t.Run(order, func(t *testing.T) {
			store, _, _ := newTestStore()
			arbiter := entity.NewWriteArbiter(store)
			arbiter.BeginFrame()

			id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
			if err != nil {
				t.Fatalf("CreateEntity returned error: %v", err)
			}

			attempt := func(system string, priority, current int) error {
				return entity.SetComponentArbitrated(arbiter, system, priority, id, entity.Health{Current: current, Max: 100}, "Current")
			}

			if order == "loser_first" {
				// Lowest-priority writes first; the highest-priority system
				// writes last and supersedes everyone.
				if err := attempt("vfx", 15, 1); err != nil {
					t.Fatalf("vfx write returned error: %v", err)
				}
				if err := attempt("combat", 5, 2); err != nil {
					t.Fatalf("combat write returned error: %v", err)
				}
				if err := attempt("stats", 3, 3); err != nil {
					t.Fatalf("stats write returned error: %v", err)
				}
				if err := attempt("ui", 10, 4); !errors.Is(err, entity.ErrWriteRejected) {
					t.Fatalf("ui write: want ErrWriteRejected, got %v", err)
				}
			} else {
				// Same systems, highest-priority first: every later write is
				// rejected outright.
				if err := attempt("stats", 3, 3); err != nil {
					t.Fatalf("stats write returned error: %v", err)
				}
				if err := attempt("combat", 5, 2); !errors.Is(err, entity.ErrWriteRejected) {
					t.Fatalf("combat write: want ErrWriteRejected, got %v", err)
				}
				if err := attempt("vfx", 15, 1); !errors.Is(err, entity.ErrWriteRejected) {
					t.Fatalf("vfx write: want ErrWriteRejected, got %v", err)
				}
				if err := attempt("ui", 10, 4); !errors.Is(err, entity.ErrWriteRejected) {
					t.Fatalf("ui write: want ErrWriteRejected, got %v", err)
				}
			}

			hp, ok := entity.GetComponent[entity.Health](store, id)
			if !ok || hp.Current != 3 {
				t.Fatalf("Health.Current = %+v, want 3 (only the lowest priority number may win, regardless of order)", hp)
			}
		})
	}
}

// The last-writer log is per frame: after BeginFrame, a write that lost a
// conflict in the previous frame is uncontested again.
func TestBeginFrame_ClearsLastWriterState(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	// Frame 1: combat writes, vfx is rejected.
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 10, Max: 100}, "Current"); err != nil {
		t.Fatalf("combat write returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "vfx", 15, id, entity.Health{Current: 50, Max: 100}, "Current"); !errors.Is(err, entity.ErrWriteRejected) {
		t.Fatalf("vfx write in frame 1: want ErrWriteRejected, got %v", err)
	}

	// Frame 2: the log is empty, so the same write now succeeds.
	arbiter.BeginFrame()
	if err := entity.SetComponentArbitrated(arbiter, "vfx", 15, id, entity.Health{Current: 50, Max: 100}, "Current"); err != nil {
		t.Fatalf("write in the next frame must be uncontested, got error: %v", err)
	}
	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok || hp.Current != 50 {
		t.Fatalf("Health.Current = %+v, want 50", hp)
	}
}

// Writes to destroyed and unknown entities are rejected by the wrapped store
// (a store error, not an arbitration rejection), and nothing is recorded;
// live entities still arbitrate normally in the same frame.
func TestWrite_DestroyedOrUnknownEntityRejected(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 5, Max: 100}, "Current"); err != nil {
		t.Fatalf("combat write returned error: %v", err)
	}

	if err := store.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity returned error: %v", err)
	}

	err = entity.SetComponentArbitrated(arbiter, "vfx", 15, id, entity.Health{Current: 1, Max: 100}, "Current")
	if err == nil {
		t.Fatal("write to a destroyed entity must error")
	}
	if errors.Is(err, entity.ErrWriteRejected) {
		t.Fatalf("destroyed-entity rejection is a store error, not arbitration: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, 9999, entity.Health{Current: 1, Max: 100}, "Current"); err == nil {
		t.Fatal("write to an unknown entity must error")
	}

	// A live entity still arbitrates normally in the same frame.
	other, err := store.CreateEntity(entity.EntityTypeNPC, "Other", matrix.NewVec3(2, 2, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, other, entity.Health{Current: 9, Max: 100}, "Current"); err != nil {
		t.Fatalf("write to a live entity returned error: %v", err)
	}
}

// GDD CR-7 multi-owner scenario: Health.Current belongs to Combat and
// Health.Max to Stats. Both write the same component in the same frame but to
// different fields; slots are per-field, so neither write conflicts. Writes
// replace the whole component (store semantics), so each system
// read-modify-writes: the later write carries the earlier system's field.
func TestDifferentFields_SameFrameNoConflict(t *testing.T) {
	store, _, _ := newTestStore()
	arbiter := entity.NewWriteArbiter(store)
	arbiter.BeginFrame()

	id, err := store.CreateEntity(entity.EntityTypeNPC, "Guard", matrix.NewVec3(1, 1, 0))
	if err != nil {
		t.Fatalf("CreateEntity returned error: %v", err)
	}

	if err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 42, Max: 100}, "Current"); err != nil {
		t.Fatalf("combat write returned error: %v", err)
	}
	if err := entity.SetComponentArbitrated(arbiter, "stats", 10, id, entity.Health{Current: 42, Max: 150}, "Max"); err != nil {
		t.Fatalf("stats write returned error: %v", err)
	}

	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok {
		t.Fatal("Health component missing")
	}
	if hp.Current != 42 || hp.Max != 150 {
		t.Fatalf("Health = %+v, want Current=42 Max=150 (different fields must not conflict)", hp)
	}
}

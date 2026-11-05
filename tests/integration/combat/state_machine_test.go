// Integration tests for Story 001: Combat State Machine & Initiative.
//
// Evidence path: production/epics/tactical-combat-engine/
// story-001-combat-state-machine-initiative.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_start_combat_pauses_clock_and_begins_first_turn       -> TestStartCombatPausesClockAndBeginsFirstTurn
//	AC-1  test_start_combat_multiple_triggers_start_once             -> TestStartCombatMultipleTriggersStartOnce
//	AC-1  test_start_combat_validation                               -> TestStartCombatValidation
//	AC-2  test_initiative_calculation                                -> TestInitiativeCalculation
//	AC-2  test_initiative_tiebreak_by_dex_then_entity_id             -> TestInitiativeTiebreakByDexThenEntityID
//	AC-2  test_initiative_negative_modifiers                         -> TestInitiativeNegativeModifiers
//	AC-2  test_turn_order_fixed_for_encounter                        -> TestTurnOrderFixedForEncounter
//	AC-3  test_turn_sequence_player_enemy_ally                       -> TestTurnSequencePlayerEnemyAlly
//	AC-3  test_combatant_removed_before_turn_is_skipped              -> TestCombatantRemovedBeforeTurnIsSkipped
//	AC-3  test_combatant_removed_during_own_turn_completes_turn      -> TestCombatantRemovedDuringOwnTurnCompletesTurn
//	AC-3  test_last_enemy_removed_during_player_turn_resolves        -> TestLastEnemyRemovedDuringPlayerTurnResolves
//	AC-4  test_combat_ends_when_last_enemy_removed                   -> TestCombatEndsWhenLastEnemyRemoved
//	AC-4  test_combat_ends_when_last_enemy_at_zero_hp                -> TestCombatEndsWhenLastEnemyAtZeroHP
//	AC-5  test_is_combat_active_for_save_gating                      -> TestIsCombatActiveForSaveGating
//	AC-6  test_enemy_turn_attacks_nearest_target_in_range            -> TestEnemyTurnAttacksNearestTargetInRange
//	AC-6  test_enemy_turn_moves_toward_nearest_target                -> TestEnemyTurnMovesTowardNearestTarget
//	AC-6  test_enemy_turn_no_path_ends_turn                          -> TestEnemyTurnNoPathEndsTurn
//	AC-6  test_enemy_turn_skipped_without_valid_targets              -> TestEnemyTurnSkippedWithoutValidTargets
//	AC-6  test_combat_resolves_when_all_players_defeated             -> TestCombatResolvesWhenAllPlayersDefeated
//	AC-7  test_enemy_flee_removes_from_turn_order                    -> TestEnemyFleeRemovesFromTurnOrder
//	AC-7  test_all_enemies_flee_end_combat                           -> TestAllEnemiesFleeEndCombat
//	Perf  test_turn_processing_16_combatants_under_50ms              -> TestTurnProcessing16CombatantsUnder50ms
//
// The Kaiju host is the same mock used by the integration tests
// (tests/integration/entity); combat itself is pure game logic, so all
// engine-bound dependencies (positions, attacks, movement) are test stubs
// injected through the combat package interfaces.
package combat_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/integration/entity). It registers engine entities by EntityId and
// records native destroy calls.
type mockKaijuHost struct {
	entities        map[engine.EntityId]*engine.Entity
	entityByIdCalls []engine.EntityId
	destroyed       []engine.EntityId
	workGroup       concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	m.entityByIdCalls = append(m.entityByIdCalls, id)
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(entity *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = entity
	return true
}

func (m *mockKaijuHost) DestroyEntity(entity *engine.Entity) {
	for id, e := range m.entities {
		if e == entity {
			delete(m.entities, id)
			m.destroyed = append(m.destroyed, id)
			return
		}
	}
}

func (m *mockKaijuHost) WorkGroup() *concurrent.WorkGroup { return &m.workGroup }

func newMockHost() *mockKaijuHost {
	m := &mockKaijuHost{entities: make(map[engine.EntityId]*engine.Entity)}
	m.workGroup.Init()
	return m
}

// mockClock is a stub for combat.Clock: it records pause/resume calls and
// exposes the paused state (the AC-1/AC-4 "game clock pauses/resumes" checks).
type mockClock struct {
	paused      bool
	pauseCalls  int
	resumeCalls int
}

func (c *mockClock) Pause() {
	c.paused = true
	c.pauseCalls++
}

func (c *mockClock) Resume() {
	c.paused = false
	c.resumeCalls++
}

// stubPositioner implements combat.Positioner with a fixed position map.
type stubPositioner struct {
	positions map[entity.GameEntityID]matrix.Vec3
}

func (p *stubPositioner) Position(id entity.GameEntityID) (matrix.Vec3, bool) {
	v, ok := p.positions[id]
	return v, ok
}

// attackCall records one ResolveAttack invocation.
type attackCall struct {
	attacker, target entity.GameEntityID
}

// recordingAttacker implements combat.Attacker and records every attack.
// onAttack, when set, runs after recording — tests use it to simulate the
// Story 003/005 flow (attack reduces HP to 0 → RemoveCombatant).
type recordingAttacker struct {
	attacks  []attackCall
	onAttack func(attacker, target entity.GameEntityID)
}

func (a *recordingAttacker) ResolveAttack(attacker, target entity.GameEntityID) combat.AttackResult {
	a.attacks = append(a.attacks, attackCall{attacker: attacker, target: target})
	if a.onAttack != nil {
		a.onAttack(attacker, target)
	}
	return combat.AttackResult{Hit: true, Damage: 1}
}

// moveCall records one MoveToward invocation.
type moveCall struct {
	combatant, target entity.GameEntityID
}

// recordingMover implements combat.Mover and records every movement.
type recordingMover struct {
	moves []moveCall
}

func (m *recordingMover) MoveToward(combatant, target entity.GameEntityID) error {
	m.moves = append(m.moves, moveCall{combatant: combatant, target: target})
	return nil
}

// failingMover implements combat.Mover and always fails — the "enemy has no
// path to any target" edge case (AC-6).
type failingMover struct{}

func (m *failingMover) MoveToward(combatant, target entity.GameEntityID) error {
	return errors.New("combat_test: no path")
}

// eventRecorder collects every event it is subscribed to.
type eventRecorder struct {
	received []entity.Event
}

func (r *eventRecorder) handler(ev entity.Event) {
	r.received = append(r.received, ev)
}

func (r *eventRecorder) ofType(t entity.EventType) []entity.Event {
	var out []entity.Event
	for _, ev := range r.received {
		if ev.Type == t {
			out = append(out, ev)
		}
	}
	return out
}

func (r *eventRecorder) count(t entity.EventType) int {
	return len(r.ofType(t))
}

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

// fixture wires a combat manager with stub dependencies. The manager uses the
// real Story 001 basic enemy AI (combat.NewBasicEnemyAI) over the stubs, so
// the enemy-turn behavior under test is the shipped behavior. pool is the
// shared Story 002 AP pool wired into both the manager and (in mover tests)
// the mover.
type fixture struct {
	store      *entity.EntityStoreImpl
	clock      *mockClock
	attacker   *recordingAttacker
	mover      *recordingMover
	positioner *stubPositioner
	pool       *combat.APPool
	manager    *combat.CombatManager
	recorder   *eventRecorder
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	clock := &mockClock{}
	attacker := &recordingAttacker{}
	mover := &recordingMover{}
	positioner := &stubPositioner{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	pool := combat.NewAPPool()
	ai := combat.NewBasicEnemyAI(positioner, attacker, mover, combat.DefaultAttackRange)
	manager := combat.NewCombatManager(bus, store, clock, ai, pool, combat.Config{})
	recorder := &eventRecorder{}
	for _, eventType := range []entity.EventType{
		combat.EventCombatStarted,
		combat.EventCombatEnded,
		combat.EventTurnStarted,
		combat.EventTurnEnded,
		combat.EventCombatantRemoved,
	} {
		bus.Subscribe(eventType, 5, recorder.handler)
	}
	return &fixture{
		store:      store,
		clock:      clock,
		attacker:   attacker,
		mover:      mover,
		positioner: positioner,
		pool:       pool,
		manager:    manager,
		recorder:   recorder,
	}
}

// addCombatant creates an entity with Stats and Health components, records
// its position in the stub positioner, and returns its GameEntityID.
func (f *fixture) addCombatant(t *testing.T, entityType entity.EntityType, name string, dex, wis, hp int, pos matrix.Vec3) entity.GameEntityID {
	t.Helper()
	id, err := f.store.CreateEntity(entityType, name, pos)
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	if err := entity.SetComponent(f.store, id, entity.Stats{Dexterity: dex, Wisdom: wis}); err != nil {
		t.Fatalf("SetComponent(Stats, %s): %v", name, err)
	}
	if err := entity.SetComponent(f.store, id, entity.Health{Current: hp, Max: hp}); err != nil {
		t.Fatalf("SetComponent(Health, %s): %v", name, err)
	}
	f.positioner.positions[id] = pos
	return id
}

// start begins an encounter with the given (id, side) pairs.
func (f *fixture) start(t *testing.T, entries ...combat.CombatantEntry) {
	t.Helper()
	if err := f.manager.StartCombat(entries); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
}

func (f *fixture) endTurn(t *testing.T) {
	t.Helper()
	if err := f.manager.EndTurn(); err != nil {
		t.Fatalf("EndTurn: %v", err)
	}
}

// turnStartedIDs returns the combatant IDs in TurnStarted event order.
func (f *fixture) turnStartedIDs() []entity.GameEntityID {
	var ids []entity.GameEntityID
	for _, ev := range f.recorder.ofType(combat.EventTurnStarted) {
		ids = append(ids, ev.Data.(combat.TurnStartedData).Combatant)
	}
	return ids
}

// turnEndedIDs returns the combatant IDs in TurnEnded event order.
func (f *fixture) turnEndedIDs() []entity.GameEntityID {
	var ids []entity.GameEntityID
	for _, ev := range f.recorder.ofType(combat.EventTurnEnded) {
		ids = append(ids, ev.Data.(combat.TurnEndedData).Combatant)
	}
	return ids
}

// combatEndedOutcome returns the Outcome of the (single) CombatEnded event,
// or panics if none fired.
func (f *fixture) combatEndedOutcome(t *testing.T) combat.Outcome {
	t.Helper()
	evs := f.recorder.ofType(combat.EventCombatEnded)
	if len(evs) != 1 {
		t.Fatalf("expected exactly one CombatEnded event, got %d", len(evs))
	}
	return evs[0].Data.(combat.CombatEndedData).Outcome
}

// assertIDsEqual compares two ID sequences exactly.
func assertIDsEqual(t *testing.T, label string, got, want []entity.GameEntityID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", label, got, want)
		}
	}
}

// assertNoTurnEventsAfter ensures no TurnStarted/TurnEnded fired beyond the
// given turn-ended count — "remaining turns are skipped" (AC-3 edge).
func (f *fixture) assertNoTurnEventsAfter(t *testing.T, turnEndedCount int) {
	t.Helper()
	if got := len(f.turnEndedIDs()); got != turnEndedCount {
		t.Errorf("turn ended count: got %d, want %d (remaining turns must be skipped)", got, turnEndedCount)
	}
	if got := len(f.turnStartedIDs()); got != turnEndedCount {
		t.Errorf("turn started count: got %d, want %d (event stream must stay balanced)", got, turnEndedCount)
	}
}

// ---------------------------------------------------------------------------
// AC-1: Combat Initiation Flow
// ---------------------------------------------------------------------------

func TestStartCombatPausesClockAndBeginsFirstTurn(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 16, 12, 20, matrix.NewVec3(0, 0, 0))
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 10, 10, 10, matrix.NewVec3(3, 0, 0))

	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
	)

	// (1) Game clock pauses (AC-1, ADR-0007).
	if !f.clock.paused {
		t.Error("clock not paused after StartCombat")
	}
	if f.clock.pauseCalls != 1 {
		t.Errorf("clock pause calls = %d, want 1", f.clock.pauseCalls)
	}

	// (2) CombatStarted fires once with the populated turn order (AC-1(3,4)).
	if got := f.recorder.count(combat.EventCombatStarted); got != 1 {
		t.Errorf("CombatStarted count = %d, want 1", got)
	}
	started := f.recorder.ofType(combat.EventCombatStarted)[0].Data.(combat.CombatStartedData)
	if len(started.TurnOrder) != 2 {
		t.Fatalf("CombatStarted turn order length = %d, want 2", len(started.TurnOrder))
	}
	if started.TurnOrder[0].EntityID != pc || started.TurnOrder[1].EntityID != enemy {
		t.Errorf("CombatStarted turn order = %v, want [pc, enemy]", started.TurnOrder)
	}

	// (3) Initiative assigned; DEX16(+3)+WIS12(+1) = 4 for the PC.
	if started.TurnOrder[0].Initiative != 4 {
		t.Errorf("pc initiative = %d, want 4", started.TurnOrder[0].Initiative)
	}

	// (4) The first combatant's turn begins (AC-1(5)).
	if got := f.manager.CurrentState(); got != combat.StatePlayerTurn {
		t.Errorf("state = %s, want PlayerTurn", got)
	}
	cur, ok := f.manager.CurrentCombatant()
	if !ok || cur != pc {
		t.Errorf("current combatant = (%d, %v), want (%d, true)", cur, ok, pc)
	}
	assertIDsEqual(t, "turn started sequence", f.turnStartedIDs(), []entity.GameEntityID{pc})
	if f.manager.Round() != 1 {
		t.Errorf("round = %d, want 1", f.manager.Round())
	}
}

// Multiple simultaneous triggers (two hostiles detect at once) start combat
// exactly once with all detected enemies (AC-1 edge case).
func TestStartCombatMultipleTriggersStartOnce(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 14, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 10, 10, 10, matrix.NewVec3(3, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 10, 10, 10, matrix.NewVec3(4, 0, 0))

	entries := []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: e1, Side: combat.SideEnemy},
		{EntityID: e2, Side: combat.SideEnemy},
	}
	// First trigger: the player attacks a hostile — combat starts with all
	// detected enemies.
	if err := f.manager.StartCombat(entries); err != nil {
		t.Fatalf("StartCombat (first trigger): %v", err)
	}
	// Second simultaneous trigger: another hostile detects the player.
	if err := f.manager.StartCombat(entries); !errors.Is(err, combat.ErrCombatActive) {
		t.Errorf("StartCombat (second trigger) error = %v, want ErrCombatActive", err)
	}
	if got := f.recorder.count(combat.EventCombatStarted); got != 1 {
		t.Errorf("CombatStarted count = %d, want 1", got)
	}
	// All detected enemies are in the turn order.
	if got := len(f.manager.TurnOrder()); got != 3 {
		t.Errorf("turn order length = %d, want 3", got)
	}
}

func TestStartCombatValidation(t *testing.T) {
	t.Run("EmptyCombatants", func(t *testing.T) {
		f := newFixture(t)
		err := f.manager.StartCombat(nil)
		if !errors.Is(err, combat.ErrNoCombatants) {
			t.Errorf("error = %v, want ErrNoCombatants", err)
		}
		if f.clock.pauseCalls != 0 {
			t.Error("clock must not pause on an invalid start")
		}
	})
	t.Run("DuplicateCombatant", func(t *testing.T) {
		f := newFixture(t)
		pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 14, 10, 20, matrix.NewVec3(0, 0, 0))
		err := f.manager.StartCombat([]combat.CombatantEntry{
			{EntityID: pc, Side: combat.SidePlayer},
			{EntityID: pc, Side: combat.SideEnemy},
		})
		if !errors.Is(err, combat.ErrDuplicateCombatant) {
			t.Errorf("error = %v, want ErrDuplicateCombatant", err)
		}
		if f.clock.pauseCalls != 0 {
			t.Error("clock must not pause on an invalid start")
		}
	})
	t.Run("MissingStats", func(t *testing.T) {
		f := newFixture(t)
		id, err := f.store.CreateEntity(entity.EntityTypeNPC, "NoStats", matrix.NewVec3(0, 0, 0))
		if err != nil {
			t.Fatalf("CreateEntity: %v", err)
		}
		if err := entity.SetComponent(f.store, id, entity.Health{Current: 10, Max: 10}); err != nil {
			t.Fatalf("SetComponent: %v", err)
		}
		err = f.manager.StartCombat([]combat.CombatantEntry{{EntityID: id, Side: combat.SideEnemy}})
		if !errors.Is(err, combat.ErrMissingStats) {
			t.Errorf("error = %v, want ErrMissingStats", err)
		}
	})
	t.Run("MissingHealth", func(t *testing.T) {
		f := newFixture(t)
		id, err := f.store.CreateEntity(entity.EntityTypeCharacter, "NoHealth", matrix.NewVec3(0, 0, 0))
		if err != nil {
			t.Fatalf("CreateEntity: %v", err)
		}
		if err := entity.SetComponent(f.store, id, entity.Stats{Dexterity: 14, Wisdom: 10}); err != nil {
			t.Fatalf("SetComponent: %v", err)
		}
		err = f.manager.StartCombat([]combat.CombatantEntry{{EntityID: id, Side: combat.SidePlayer}})
		if !errors.Is(err, combat.ErrMissingHealth) {
			t.Errorf("error = %v, want ErrMissingHealth", err)
		}
	})
	t.Run("TooManyCombatants", func(t *testing.T) {
		f := newFixture(t)
		entries := make([]combat.CombatantEntry, 0, 17)
		for i := 0; i < 17; i++ {
			id := f.addCombatant(t, entity.EntityTypeNPC, fmt.Sprintf("Mob%d", i), 10, 10, 10, matrix.NewVec3(float32(i), 0, 0))
			entries = append(entries, combat.CombatantEntry{EntityID: id, Side: combat.SideEnemy})
		}
		err := f.manager.StartCombat(entries)
		if !errors.Is(err, combat.ErrTooManyCombatants) {
			t.Errorf("error = %v, want ErrTooManyCombatants", err)
		}
		if f.clock.pauseCalls != 0 {
			t.Error("clock must not pause on an invalid start")
		}
	})
}

// ---------------------------------------------------------------------------
// AC-2: Initiative Calculation
// ---------------------------------------------------------------------------

func TestInitiativeCalculation(t *testing.T) {
	f := newFixture(t)
	// QA AC-2 combatants: (DEX 16/WIS 12), (DEX 14/WIS 14), (DEX 14/WIS 10).
	// Note: the story QA text writes combatant 2's WIS modifier as +1, but
	// WIS 14 has modifier +2 per ADR-0006 (floor((A-10)/2)) — the corrected
	// values are [4, 4, 2]. The expected order [1, 2, 3] still holds: the
	// 4-4 tie is broken by higher DEX (16 > 14).
	c1 := f.addCombatant(t, entity.EntityTypeCharacter, "C1", 16, 12, 20, matrix.NewVec3(0, 0, 0))
	c2 := f.addCombatant(t, entity.EntityTypeCharacter, "C2", 14, 14, 20, matrix.NewVec3(1, 0, 0))
	c3 := f.addCombatant(t, entity.EntityTypeNPC, "C3", 14, 10, 10, matrix.NewVec3(2, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: c1, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: c2, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: c3, Side: combat.SideEnemy},
	)

	order := f.manager.TurnOrder()
	want := []struct {
		id         entity.GameEntityID
		initiative int
	}{
		{c1, 4}, // 3 + 1
		{c2, 4}, // 2 + 2 (DEX tiebreak over c3's 2+0=2)
		{c3, 2}, // 2 + 0
	}
	if len(order) != len(want) {
		t.Fatalf("turn order length = %d, want %d", len(order), len(want))
	}
	for i, w := range want {
		if order[i].EntityID != w.id || order[i].Initiative != w.initiative {
			t.Errorf("order[%d] = (id %d, init %d), want (id %d, init %d)",
				i, order[i].EntityID, order[i].Initiative, w.id, w.initiative)
		}
	}
}

func TestInitiativeTiebreakByDexThenEntityID(t *testing.T) {
	f := newFixture(t)
	// All three have initiative 2 (DEX 14/WIS 11 and DEX 13/WIS 12 both give
	// 2+0 / 1+1). Higher DEX first: A (DEX 14) beats B (DEX 13). A and C are
	// identical (DEX 14/WIS 11): higher entity ID first — C was created
	// after A, so C leads (AC-2 edge case).
	a := f.addCombatant(t, entity.EntityTypeCharacter, "A", 14, 11, 20, matrix.NewVec3(0, 0, 0))
	b := f.addCombatant(t, entity.EntityTypeCharacter, "B", 13, 12, 20, matrix.NewVec3(1, 0, 0))
	c := f.addCombatant(t, entity.EntityTypeNPC, "C", 14, 11, 10, matrix.NewVec3(2, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: a, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: b, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: c, Side: combat.SideEnemy},
	)

	order := f.manager.TurnOrder()
	ids := make([]entity.GameEntityID, len(order))
	for i, e := range order {
		ids[i] = e.EntityID
	}
	assertIDsEqual(t, "turn order", ids, []entity.GameEntityID{c, a, b})
}

func TestInitiativeNegativeModifiers(t *testing.T) {
	f := newFixture(t)
	// DEX 6 = -2, WIS 8 = -1 → initiative -3 (AC-2 edge: negative modifiers).
	low := f.addCombatant(t, entity.EntityTypeNPC, "Low", 6, 8, 10, matrix.NewVec3(0, 0, 0))
	// DEX 18 = +4, WIS 6 = -2 → initiative +2.
	high := f.addCombatant(t, entity.EntityTypeCharacter, "High", 18, 6, 20, matrix.NewVec3(1, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: high, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: low, Side: combat.SideEnemy},
	)

	order := f.manager.TurnOrder()
	if len(order) != 2 {
		t.Fatalf("turn order length = %d, want 2", len(order))
	}
	if order[0].EntityID != high || order[0].Initiative != 2 {
		t.Errorf("order[0] = (id %d, init %d), want (id %d, init 2)", order[0].EntityID, order[0].Initiative, high)
	}
	if order[1].EntityID != low || order[1].Initiative != -3 {
		t.Errorf("order[1] = (id %d, init %d), want (id %d, init -3)", order[1].EntityID, order[1].Initiative, low)
	}
}

// The turn order is fixed for the encounter duration (AC-2): a full cycle
// leaves the order unchanged and increments the round.
func TestTurnOrderFixedForEncounter(t *testing.T) {
	f := newFixture(t)
	pc1 := f.addCombatant(t, entity.EntityTypeCharacter, "PC1", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	pc2 := f.addCombatant(t, entity.EntityTypeCharacter, "PC2", 15, 10, 20, matrix.NewVec3(0, 2, 0))
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc1, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: pc2, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
	)

	before := f.manager.TurnOrder()

	// PC1 → PC2 → (enemy turn auto-processes) → back to PC1, round 2.
	f.endTurn(t)
	f.endTurn(t)

	if got := f.manager.Round(); got != 2 {
		t.Errorf("round = %d, want 2", got)
	}
	after := f.manager.TurnOrder()
	if len(before) != len(after) {
		t.Fatalf("turn order length changed: %d → %d", len(before), len(after))
	}
	for i := range before {
		if before[i].EntityID != after[i].EntityID || before[i].Initiative != after[i].Initiative {
			t.Errorf("turn order changed at %d: %+v → %+v", i, before[i], after[i])
		}
	}
}

// ---------------------------------------------------------------------------
// AC-3: Turn State Transitions
// ---------------------------------------------------------------------------

// QA AC-3: 2 player characters, 3 enemies, 1 ally → sequence
// PlayerTurn(PC1) → PlayerTurn(PC2) → EnemyTurn(E1..E3) → AllyTurn(A1) →
// PlayerTurn(PC1).
func TestTurnSequencePlayerEnemyAlly(t *testing.T) {
	f := newFixture(t)
	// Distinct initiatives keep the expected order independent of tiebreaks.
	pc1 := f.addCombatant(t, entity.EntityTypeCharacter, "PC1", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	pc2 := f.addCombatant(t, entity.EntityTypeCharacter, "PC2", 16, 10, 20, matrix.NewVec3(0, 2, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	e3 := f.addCombatant(t, entity.EntityTypeNPC, "E3", 10, 10, 10, matrix.NewVec3(5, 4, 0))
	a1 := f.addCombatant(t, entity.EntityTypeCharacter, "A1", 8, 10, 15, matrix.NewVec3(0, 4, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc1, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: pc2, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e3, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: a1, Side: combat.SideAlly},
	)

	// First combatant begins their turn (AC-1).
	if got := f.manager.CurrentState(); got != combat.StatePlayerTurn {
		t.Errorf("state after start = %s, want PlayerTurn", got)
	}

	// PC1's turn ends → PC2's turn.
	f.endTurn(t)
	if got := f.manager.CurrentState(); got != combat.StatePlayerTurn {
		t.Errorf("state after 1st EndTurn = %s, want PlayerTurn", got)
	}

	// PC2's turn ends → the three enemy turns process automatically → A1.
	f.endTurn(t)
	if got := f.manager.CurrentState(); got != combat.StateAllyTurn {
		t.Errorf("state after 2nd EndTurn = %s, want AllyTurn", got)
	}
	cur, ok := f.manager.CurrentCombatant()
	if !ok || cur != a1 {
		t.Errorf("current combatant = (%d, %v), want (%d, true)", cur, ok, a1)
	}

	// A1's turn ends → wrap to PC1, round 2.
	f.endTurn(t)
	if got := f.manager.CurrentState(); got != combat.StatePlayerTurn {
		t.Errorf("state after 3rd EndTurn = %s, want PlayerTurn", got)
	}
	if got := f.manager.Round(); got != 2 {
		t.Errorf("round = %d, want 2", got)
	}

	// Full TurnStarted sequence matches the QA case.
	assertIDsEqual(t, "turn sequence",
		f.turnStartedIDs(),
		[]entity.GameEntityID{pc1, pc2, e1, e2, e3, a1, pc1})
	// Every turn that started also ended (balanced event stream).
	assertIDsEqual(t, "turn ended sequence",
		f.turnEndedIDs(),
		[]entity.GameEntityID{pc1, pc2, e1, e2, e3, a1})
}

// A combatant removed before their turn ever began is skipped (AC-3 edge:
// "complete their turn, then remove from order" — removal during another
// combatant's turn). The encounter continues with the remaining combatants.
func TestCombatantRemovedBeforeTurnIsSkipped(t *testing.T) {
	f := newFixture(t)
	pc1 := f.addCombatant(t, entity.EntityTypeCharacter, "PC1", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	pc2 := f.addCombatant(t, entity.EntityTypeCharacter, "PC2", 10, 10, 20, matrix.NewVec3(0, 2, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc1, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc2, Side: combat.SidePlayer},
	)

	// Remove the mid-order E1 during PC1's turn: combat continues (E2 is
	// still alive) and E1's turn never begins.
	if err := f.manager.RemoveCombatant(e1, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant: %v", err)
	}
	if got := f.recorder.count(combat.EventCombatantRemoved); got != 1 {
		t.Errorf("CombatantRemoved count = %d, want 1", got)
	}
	if f.manager.IsCombatActive() != true {
		t.Error("combat must stay active while an enemy remains")
	}

	// PC1's turn completes normally; E2's turn auto-processes (moves toward
	// the nearest target — out of range, no attack); PC2's turn begins. E1
	// is skipped entirely.
	f.endTurn(t)
	cur, ok := f.manager.CurrentCombatant()
	if !ok || cur != pc2 {
		t.Errorf("current combatant = (%d, %v), want (%d, true)", cur, ok, pc2)
	}
	assertIDsEqual(t, "turn sequence", f.turnStartedIDs(), []entity.GameEntityID{pc1, e2, pc2})
	assertIDsEqual(t, "turn ended sequence", f.turnEndedIDs(), []entity.GameEntityID{pc1, e2})
}

// A combatant removed during their own turn completes the turn, then is out
// of the order (AC-3 edge). The onAttack hook simulates the Story 003/005
// flow: the attack reduces the enemy to 0 HP, and the death system removes
// it mid-turn.
func TestCombatantRemovedDuringOwnTurnCompletesTurn(t *testing.T) {
	f := newFixture(t)
	// Enemy first so its turn runs during StartCombat.
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 18, 10, 10, matrix.NewVec3(0, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 16, 10, 10, matrix.NewVec3(0, 2, 0))
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 14, 10, 20, matrix.NewVec3(1, 0, 0))
	f.attacker.onAttack = func(attacker, target entity.GameEntityID) {
		if err := f.manager.RemoveCombatant(attacker, combat.RemovalDefeated); err != nil {
			t.Errorf("RemoveCombatant inside attack: %v", err)
		}
	}

	f.start(t,
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
	)

	// E1 attacked (adjacent to PC), was removed during its own turn, and its
	// turn still completed with a TurnEnded; E2's turn then began (auto).
	if len(f.attacker.attacks) != 1 || f.attacker.attacks[0].attacker != e1 {
		t.Errorf("attacks = %v, want exactly one attack by e1", f.attacker.attacks)
	}
	order := f.manager.TurnOrder()
	if len(order) != 2 || order[0].EntityID != e2 || order[1].EntityID != pc {
		t.Errorf("turn order after removal = %v, want [e2, pc]", order)
	}
	// E1's turn completed (TurnEnded), E2's turn ran and ended too, and the
	// player's turn is now active: started [e1, e2, pc], ended [e1, e2].
	assertIDsEqual(t, "turn sequence", f.turnStartedIDs(), []entity.GameEntityID{e1, e2, pc})
	assertIDsEqual(t, "turn ended sequence", f.turnEndedIDs(), []entity.GameEntityID{e1, e2})
	if f.manager.IsCombatActive() != true {
		t.Error("combat must stay active while e2 remains")
	}
}

// The last enemy dies during PlayerTurn → skip remaining player turns,
// transition to Resolution (AC-3 edge, AC-4).
func TestLastEnemyRemovedDuringPlayerTurnResolves(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
	)
	if got := f.manager.CurrentState(); got != combat.StatePlayerTurn {
		t.Fatalf("state = %s, want PlayerTurn", got)
	}

	// First enemy removed during PC's turn — combat continues.
	if err := f.manager.RemoveCombatant(e1, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant(e1): %v", err)
	}
	if !f.manager.IsCombatActive() {
		t.Fatal("combat must stay active after the first removal")
	}

	// Last enemy removed → immediate resolution, remaining player turns
	// skipped (AC-3 edge).
	if err := f.manager.RemoveCombatant(e2, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant(e2): %v", err)
	}
	if f.manager.IsCombatActive() {
		t.Error("combat must be inactive after the last enemy is removed")
	}
	if got := f.manager.CurrentState(); got != combat.StateInactive {
		t.Errorf("state = %s, want Inactive", got)
	}
	if got := f.combatEndedOutcome(t); got != combat.OutcomeVictory {
		t.Errorf("outcome = %s, want Victory", got)
	}
	// The interrupted player turn ended; no further turns began.
	f.assertNoTurnEventsAfter(t, 1)
	if f.clock.resumeCalls != 1 {
		t.Errorf("clock resume calls = %d, want 1", f.clock.resumeCalls)
	}
	if f.clock.paused {
		t.Error("clock must be resumed after combat ends")
	}
	// A subsequent EndTurn is rejected — the encounter is over.
	if err := f.manager.EndTurn(); !errors.Is(err, combat.ErrNoActiveCombat) {
		t.Errorf("EndTurn after resolution error = %v, want ErrNoActiveCombat", err)
	}
}

// ---------------------------------------------------------------------------
// AC-4: Combat End on Enemy Defeat
// ---------------------------------------------------------------------------

// QA AC-4: a player attack reduces the last enemy to 0 HP → CombatEnded
// fires, the clock resumes, and the encounter finalizes. The death system
// (Story 005) calls RemoveCombatant; here the onAttack hook simulates it.
func TestCombatEndsWhenLastEnemyRemoved(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 14, 10, 10, matrix.NewVec3(1, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
	)
	if !f.clock.paused {
		t.Fatal("clock must be paused during combat")
	}

	if err := f.manager.RemoveCombatant(enemy, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant: %v", err)
	}

	if got := f.recorder.count(combat.EventCombatEnded); got != 1 {
		t.Errorf("CombatEnded count = %d, want 1", got)
	}
	if got := f.combatEndedOutcome(t); got != combat.OutcomeVictory {
		t.Errorf("outcome = %s, want Victory", got)
	}
	if f.clock.paused || f.clock.resumeCalls != 1 {
		t.Errorf("clock after end: paused=%v resumeCalls=%d, want resumed once", f.clock.paused, f.clock.resumeCalls)
	}
	if f.manager.IsCombatActive() {
		t.Error("combat must be inactive after resolution")
	}
	if _, ok := f.manager.CurrentCombatant(); ok {
		t.Error("no combatant must be current after resolution")
	}
	// The event stream stays balanced: TurnStarted(PC) … TurnEnded(PC).
	f.assertNoTurnEventsAfter(t, 1)
}

// The last enemy reaches 0 HP (e.g. environmental damage) without any removal
// call — the alive check still resolves the encounter (AC-4 edge case).
func TestCombatEndsWhenLastEnemyAtZeroHP(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
	)

	// Environmental damage reduces the last enemy to 0 HP — no removal call.
	if err := entity.SetComponent(f.store, enemy, entity.Health{Current: 0, Max: 10}); err != nil {
		t.Fatalf("SetComponent(Health): %v", err)
	}

	// The player's turn ends; the end condition is evaluated and resolves the
	// encounter without starting further turns.
	f.endTurn(t)

	if got := f.recorder.count(combat.EventCombatEnded); got != 1 {
		t.Errorf("CombatEnded count = %d, want 1", got)
	}
	if got := f.combatEndedOutcome(t); got != combat.OutcomeVictory {
		t.Errorf("outcome = %s, want Victory", got)
	}
	if f.clock.paused {
		t.Error("clock must be resumed after combat ends")
	}
	f.assertNoTurnEventsAfter(t, 1)
}

// ---------------------------------------------------------------------------
// AC-5: Save Restrictions During Combat
// ---------------------------------------------------------------------------

// The save system gates quick-save/auto-save on IsCombatActive: true while an
// encounter runs (including mid-turn), false before and after (AC-5). The
// save-manager wiring itself belongs to the save epic; this test pins the
// contract the save system checks.
func TestIsCombatActiveForSaveGating(t *testing.T) {
	f := newFixture(t)
	if f.manager.IsCombatActive() {
		t.Fatal("combat must be inactive before StartCombat")
	}

	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
	)

	if !f.manager.IsCombatActive() {
		t.Fatal("combat must be active after StartCombat")
	}
	// Mid-combat (during a turn) — still active.
	f.endTurn(t)
	if !f.manager.IsCombatActive() {
		t.Fatal("combat must be active during a turn")
	}

	// Combat ends — the save gate opens (a manual save during Resolution
	// succeeds after the state is cleaned up; resolution is synchronous here).
	if err := f.manager.RemoveCombatant(e1, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant(e1): %v", err)
	}
	if err := f.manager.RemoveCombatant(e2, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant(e2): %v", err)
	}
	if f.manager.IsCombatActive() {
		t.Fatal("combat must be inactive after resolution")
	}
}

// ---------------------------------------------------------------------------
// AC-6: Basic Enemy Turn
// ---------------------------------------------------------------------------

// QA AC-6: an enemy with a player character adjacent attacks the nearest
// target instead of moving.
func TestEnemyTurnAttacksNearestTargetInRange(t *testing.T) {
	f := newFixture(t)
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 18, 10, 10, matrix.NewVec3(0, 0, 0))
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 12, 10, 20, matrix.NewVec3(1, 0, 0))   // 1 tile away — in range
	ally := f.addCombatant(t, entity.EntityTypeCharacter, "Ally", 10, 10, 15, matrix.NewVec3(5, 5, 0)) // far away, lower initiative
	f.start(t,
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: ally, Side: combat.SideAlly},
	)

	// The enemy's turn ran during StartCombat: one attack against the nearest
	// valid target (PC at distance 1, not the distant ally).
	if len(f.attacker.attacks) != 1 {
		t.Fatalf("attacks = %v, want exactly one", f.attacker.attacks)
	}
	if a := f.attacker.attacks[0]; a.attacker != enemy || a.target != pc {
		t.Errorf("attack = (attacker %d, target %d), want (attacker %d, target %d)", a.attacker, a.target, enemy, pc)
	}
	if len(f.mover.moves) != 0 {
		t.Errorf("moves = %v, want none (target in range)", f.mover.moves)
	}
	// The enemy's turn ended and the player's turn began.
	cur, ok := f.manager.CurrentCombatant()
	if !ok || cur != pc {
		t.Errorf("current combatant = (%d, %v), want (%d, true)", cur, ok, pc)
	}
}

// QA AC-6: an enemy with a player character 3 tiles away moves toward the
// nearest player character.
func TestEnemyTurnMovesTowardNearestTarget(t *testing.T) {
	f := newFixture(t)
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 18, 10, 10, matrix.NewVec3(0, 0, 0))
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 10, 10, 20, matrix.NewVec3(3, 0, 0))    // 3 tiles away
	ally := f.addCombatant(t, entity.EntityTypeCharacter, "Ally", 10, 10, 15, matrix.NewVec3(10, 10, 0)) // farther
	f.start(t,
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: ally, Side: combat.SideAlly},
	)

	if len(f.mover.moves) != 1 {
		t.Fatalf("moves = %v, want exactly one", f.mover.moves)
	}
	if m := f.mover.moves[0]; m.combatant != enemy || m.target != pc {
		t.Errorf("move = (combatant %d, target %d), want (combatant %d, target %d)", m.combatant, m.target, enemy, pc)
	}
	if len(f.attacker.attacks) != 0 {
		t.Errorf("attacks = %v, want none (target out of range)", f.attacker.attacks)
	}
}

// An enemy with no path to any target ends its turn without acting (AC-6
// edge case).
func TestEnemyTurnNoPathEndsTurn(t *testing.T) {
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	clock := &mockClock{}
	attacker := &recordingAttacker{}
	positioner := &stubPositioner{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	ai := combat.NewBasicEnemyAI(positioner, attacker, &failingMover{}, combat.DefaultAttackRange)
	manager := combat.NewCombatManager(bus, store, clock, ai, combat.NewAPPool(), combat.Config{})
	recorder := &eventRecorder{}
	for _, eventType := range []entity.EventType{
		combat.EventCombatStarted,
		combat.EventCombatEnded,
		combat.EventTurnStarted,
		combat.EventTurnEnded,
	} {
		bus.Subscribe(eventType, 5, recorder.handler)
	}

	enemy, err := store.CreateEntity(entity.EntityTypeNPC, "Enemy", matrix.NewVec3(0, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	pc, err := store.CreateEntity(entity.EntityTypeCharacter, "PC", matrix.NewVec3(3, 0, 0))
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	// Distinct DEX values keep the enemy first in the order (DEX 14 beats
	// DEX 10), so the enemy's turn runs during StartCombat.
	if err := entity.SetComponent(store, enemy, entity.Stats{Dexterity: 14, Wisdom: 10}); err != nil {
		t.Fatalf("SetComponent(Stats, enemy): %v", err)
	}
	if err := entity.SetComponent(store, pc, entity.Stats{Dexterity: 10, Wisdom: 10}); err != nil {
		t.Fatalf("SetComponent(Stats, pc): %v", err)
	}
	for id, hp := range map[entity.GameEntityID]int{enemy: 10, pc: 20} {
		if err := entity.SetComponent(store, id, entity.Health{Current: hp, Max: hp}); err != nil {
			t.Fatalf("SetComponent(Health): %v", err)
		}
	}
	positioner.positions[enemy] = matrix.NewVec3(0, 0, 0)
	positioner.positions[pc] = matrix.NewVec3(3, 0, 0)

	if err := manager.StartCombat([]combat.CombatantEntry{
		{EntityID: enemy, Side: combat.SideEnemy},
		{EntityID: pc, Side: combat.SidePlayer},
	}); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	// The enemy attempted to move, failed, and its turn ended without acting;
	// the player's turn began.
	if len(attacker.attacks) != 0 {
		t.Errorf("attacks = %v, want none (no path)", attacker.attacks)
	}
	if got := recorder.count(combat.EventTurnEnded); got != 1 {
		t.Errorf("TurnEnded count = %d, want 1 (enemy turn ends)", got)
	}
	cur, ok := manager.CurrentCombatant()
	if !ok || cur != pc {
		t.Errorf("current combatant = (%d, %v), want (%d, true)", cur, ok, pc)
	}
}

// All player characters dead → the enemy turn is skipped (no valid targets),
// and combat resolves (AC-6 edge case).
func TestEnemyTurnSkippedWithoutValidTargets(t *testing.T) {
	f := newFixture(t)
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 18, 10, 10, matrix.NewVec3(0, 0, 0))
	// The player is at 0 HP (not yet removed — Story 005 territory), so no
	// valid target exists.
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 10, 10, 0, matrix.NewVec3(3, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
	)

	if len(f.attacker.attacks) != 0 {
		t.Errorf("attacks = %v, want none (no valid targets)", f.attacker.attacks)
	}
	if len(f.mover.moves) != 0 {
		t.Errorf("moves = %v, want none (no valid targets)", f.mover.moves)
	}
	// The enemy's turn ended without acting.
	assertIDsEqual(t, "turn sequence", f.turnStartedIDs(), []entity.GameEntityID{enemy})
	assertIDsEqual(t, "turn ended sequence", f.turnEndedIDs(), []entity.GameEntityID{enemy})
}

// All player characters removed → Defeat resolution (CR-6, AC-6 edge).
func TestCombatResolvesWhenAllPlayersDefeated(t *testing.T) {
	f := newFixture(t)
	enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 18, 10, 10, matrix.NewVec3(0, 0, 0))
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 10, 10, 20, matrix.NewVec3(3, 0, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
	)

	// The enemy's turn ran during StartCombat (moved toward PC, no attack).
	// The player is removed — defeat.
	if err := f.manager.RemoveCombatant(pc, combat.RemovalDefeated); err != nil {
		t.Fatalf("RemoveCombatant(pc): %v", err)
	}
	if f.manager.IsCombatActive() {
		t.Error("combat must be inactive after the last player is removed")
	}
	if got := f.combatEndedOutcome(t); got != combat.OutcomeDefeat {
		t.Errorf("outcome = %s, want Defeat", got)
	}
	if f.clock.paused {
		t.Error("clock must be resumed after combat ends")
	}
}

// ---------------------------------------------------------------------------
// AC-7: Enemy Flee
// ---------------------------------------------------------------------------

// QA AC-7: an enemy that exits the combat area is removed from the turn order
// (flee detection is the movement system's job — Story 002 — the manager
// handles the removal contract); no XP accounting exists yet (Story 006).
func TestEnemyFleeRemovesFromTurnOrder(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
	)

	if err := f.manager.RemoveCombatant(e1, combat.RemovalFled); err != nil {
		t.Fatalf("RemoveCombatant(e1, Fled): %v", err)
	}

	// The removal event carries the flee reason (the world reacts by making
	// the enemy neutral in exploration mode, AC-7).
	removed := f.recorder.ofType(combat.EventCombatantRemoved)
	if len(removed) != 1 {
		t.Fatalf("CombatantRemoved count = %d, want 1", len(removed))
	}
	data := removed[0].Data.(combat.CombatantRemovedData)
	if data.EntityID != e1 || data.Reason != combat.RemovalFled {
		t.Errorf("removal = (id %d, reason %s), want (id %d, reason Fled)", data.EntityID, data.Reason, e1)
	}

	// E1 is out of the turn order; combat continues with the remaining
	// combatants, and E1's turn never comes.
	order := f.manager.TurnOrder()
	for _, e := range order {
		if e.EntityID == e1 {
			t.Error("fled enemy must not remain in the turn order")
		}
	}
	if !f.manager.IsCombatActive() {
		t.Error("combat must stay active while other enemies remain")
	}
	// Two EndTurns: each advances the player's turn, then E2's turn
	// auto-processes (enemy turns run automatically), then the player's turn
	// begins again. E1 (id 2) never appears — its turn was skipped.
	f.endTurn(t)
	f.endTurn(t)
	assertIDsEqual(t, "turn sequence", f.turnStartedIDs(), []entity.GameEntityID{pc, e2, pc, e2, pc})
	for _, id := range f.turnStartedIDs() {
		if id == e1 {
			t.Error("fled enemy must never begin a turn")
		}
	}
}

// QA AC-7 edge: all enemies flee → combat ends (Victory, no XP for the
// fleeing enemies — XP accounting is Story 006).
func TestAllEnemiesFleeEndCombat(t *testing.T) {
	f := newFixture(t)
	pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
	e1 := f.addCombatant(t, entity.EntityTypeNPC, "E1", 14, 10, 10, matrix.NewVec3(5, 0, 0))
	e2 := f.addCombatant(t, entity.EntityTypeNPC, "E2", 12, 10, 10, matrix.NewVec3(5, 2, 0))
	f.start(t,
		combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
		combat.CombatantEntry{EntityID: e1, Side: combat.SideEnemy},
		combat.CombatantEntry{EntityID: e2, Side: combat.SideEnemy},
	)

	if err := f.manager.RemoveCombatant(e1, combat.RemovalFled); err != nil {
		t.Fatalf("RemoveCombatant(e1, Fled): %v", err)
	}
	if err := f.manager.RemoveCombatant(e2, combat.RemovalFled); err != nil {
		t.Fatalf("RemoveCombatant(e2, Fled): %v", err)
	}

	if f.manager.IsCombatActive() {
		t.Error("combat must be inactive when all enemies fled")
	}
	if got := f.combatEndedOutcome(t); got != combat.OutcomeVictory {
		t.Errorf("outcome = %s, want Victory", got)
	}
	if f.clock.paused {
		t.Error("clock must be resumed after combat ends")
	}
}

// ---------------------------------------------------------------------------
// Guarded transitions (invalid calls are rejected)
// ---------------------------------------------------------------------------

func TestGuardedTransitions(t *testing.T) {
	t.Run("EndTurnWithoutCombat", func(t *testing.T) {
		f := newFixture(t)
		if err := f.manager.EndTurn(); !errors.Is(err, combat.ErrNoActiveCombat) {
			t.Errorf("error = %v, want ErrNoActiveCombat", err)
		}
		if err := f.manager.RemoveCombatant(1, combat.RemovalDefeated); !errors.Is(err, combat.ErrNoActiveCombat) {
			t.Errorf("RemoveCombatant error = %v, want ErrNoActiveCombat", err)
		}
		if err := f.manager.EndCombat(combat.OutcomeScripted); !errors.Is(err, combat.ErrNoActiveCombat) {
			t.Errorf("EndCombat error = %v, want ErrNoActiveCombat", err)
		}
	})
	t.Run("RemoveNonCombatant", func(t *testing.T) {
		f := newFixture(t)
		pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
		enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 14, 10, 10, matrix.NewVec3(5, 0, 0))
		bystander := f.addCombatant(t, entity.EntityTypeNPC, "Bystander", 10, 10, 5, matrix.NewVec3(9, 9, 0))
		f.start(t,
			combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
			combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		)
		err := f.manager.RemoveCombatant(bystander, combat.RemovalDefeated)
		if !errors.Is(err, combat.ErrNotCombatant) {
			t.Errorf("error = %v, want ErrNotCombatant", err)
		}
	})
	t.Run("EndTurnWhileResolutionActive", func(t *testing.T) {
		// EndCombat finalizes synchronously, so a turn cannot be ended during
		// Resolution — after it, EndTurn is rejected (covered above). This
		// pins the scripted end flow itself.
		f := newFixture(t)
		pc := f.addCombatant(t, entity.EntityTypeCharacter, "PC", 18, 10, 20, matrix.NewVec3(0, 0, 0))
		enemy := f.addCombatant(t, entity.EntityTypeNPC, "Enemy", 14, 10, 10, matrix.NewVec3(5, 0, 0))
		f.start(t,
			combat.CombatantEntry{EntityID: pc, Side: combat.SidePlayer},
			combat.CombatantEntry{EntityID: enemy, Side: combat.SideEnemy},
		)
		if err := f.manager.EndCombat(combat.OutcomeScripted); err != nil {
			t.Fatalf("EndCombat: %v", err)
		}
		if got := f.combatEndedOutcome(t); got != combat.OutcomeScripted {
			t.Errorf("outcome = %s, want Scripted", got)
		}
		if f.clock.paused {
			t.Error("clock must be resumed after scripted end")
		}
		if f.manager.IsCombatActive() {
			t.Error("combat must be inactive after scripted end")
		}
	})
}

// ---------------------------------------------------------------------------
// Performance: turn processing for up to 16 combatants
// ---------------------------------------------------------------------------

// QA performance case: with 16 combatants (8 friendly + 8 enemy), a full
// turn cycle (state transition + basic AI evaluation) completes well under
// the 50ms budget (ADR-0009: AI decision 10-50ms; transition <1ms). The
// budget check is generous to stay deterministic in CI.
func TestTurnProcessing16CombatantsUnder50ms(t *testing.T) {
	f := newFixture(t)
	entries := make([]combat.CombatantEntry, 0, 16)
	for i := 0; i < 8; i++ {
		id := f.addCombatant(t, entity.EntityTypeCharacter, fmt.Sprintf("PC%d", i),
			18-i, 10, 20, matrix.NewVec3(float32(i), 0, 0))
		entries = append(entries, combat.CombatantEntry{EntityID: id, Side: combat.SidePlayer})
		id = f.addCombatant(t, entity.EntityTypeNPC, fmt.Sprintf("E%d", i),
			10-i, 10, 10, matrix.NewVec3(float32(i), 10, 0))
		entries = append(entries, combat.CombatantEntry{EntityID: id, Side: combat.SideEnemy})
	}

	start := time.Now()
	if err := f.manager.StartCombat(entries); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	// Complete ten full cycles of all 16 combatants. Each cycle has exactly
	// 8 player turns (advanced via EndTurn); the 8 enemy turns process
	// automatically inside EndTurn. After every EndTurn the state is a
	// player turn again, so the loop simply counts them.
	const cycles = 10
	playerTurns := 0
	for playerTurns < cycles*8 {
		if f.manager.CurrentState() != combat.StatePlayerTurn {
			t.Fatalf("expected a player turn, state = %s", f.manager.CurrentState())
		}
		f.endTurn(t)
		playerTurns++
	}
	elapsed := time.Since(start)

	// QA budget: "full turn processing (including basic AI evaluation)
	// completes in <50ms" per turn. 160 turns in 500ms is a 50x margin on
	// the per-turn budget — a deterministic CI-safe gate.
	const budget = 500 * time.Millisecond
	if elapsed > budget {
		t.Errorf("16-combatant turn cycle (10 rounds, 160 turns) took %v, budget %v (per-turn <50ms)", elapsed, budget)
	} else {
		t.Logf("160 turns processed in %v (per-turn %v — budget 50ms)", elapsed, elapsed/160)
	}
	if got := f.manager.Round(); got != 11 {
		t.Errorf("round = %d, want 11 (10 full cycles after round 1)", got)
	}
}

// Unit tests for Story 002: Grid Movement & AP Economy.
//
// Evidence path: production/epics/tactical-combat-engine/
// story-002-grid-movement-ap-economy.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria and QA test cases map to test functions 1:1 (Go
// only executes Test-prefixed functions, so each story name is exported with
// the Go prefix):
//
//	AC-1  test_diagonal_move_costs_1_ap        -> TestDiagonalMoveCostsOneAP
//	AC-2  test_move_5_tiles_consumes_5_ap      -> TestMoveConsumesOneAPPerTile
//	AC-3  test_range_all_chebyshev_le_ap       -> TestReachableTilesWithinChebyshevBudget
//	AC-4  test_pathfinding_excludes_occupied   -> TestPathExcludesBlockedTiles
//	AC-5  test_attack_spends_2_ap              -> TestAttackCostSpendsTwoAP
//	(TR)  test_ap_derivation                   -> TestAPFor
//	(edge) test_mover_insufficient_ap          -> TestMoveTowardInsufficientAP
//	(edge) test_mover_no_path                  -> TestMoveTowardNoPath
//	(edge) test_mover_blocked_route            -> TestMoveTowardBlockedRoute
//	(EF)  test_entity_tile_blocker             -> TestEntityTileBlocker
//	(int) test_turn_start_resets_pool          -> TestTurnStartResetsAPPool
//	(int) test_turn_ends_when_ap_reaches_0     -> TestEndTurnAfterAPExhausted
//	(int) test_enemy_ai_moves_with_grid_mover  -> TestEnemyAIMovesWithGridMover
//
// The Kaiju host is the same mock used by the unit/integration entity tests;
// combat itself is pure game logic, so all engine-bound dependencies
// (positions, moves, obstacles) are test stubs injected through the combat
// package interfaces.
package combat_test

import (
	"fmt"
	"reflect"
	"testing"

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

// moveRecord is one SetPosition call.
type moveRecord struct {
	id  entity.GameEntityID
	pos matrix.Vec3
}

// stubPositions implements both combat.Positioner and combat.PositionSetter
// over one shared position map, so a moved entity is immediately visible to
// position reads (the game wiring behaves the same: Kaiju transform plus
// EntityStore.UpdatePosition). It also records every move for assertions.
type stubPositions struct {
	positions map[entity.GameEntityID]matrix.Vec3
	moves     []moveRecord
}

func (s *stubPositions) Position(id entity.GameEntityID) (matrix.Vec3, bool) {
	v, ok := s.positions[id]
	return v, ok
}

func (s *stubPositions) SetPosition(id entity.GameEntityID, pos matrix.Vec3) error {
	s.positions[id] = pos
	s.moves = append(s.moves, moveRecord{id: id, pos: pos})
	return nil
}

var (
	_ combat.Positioner   = (*stubPositions)(nil)
	_ combat.PositionSetter = (*stubPositions)(nil)
)

// stubBlocker implements combat.TileBlocker with a fixed blocked-tile set.
type stubBlocker struct {
	blocked map[combat.Tile]bool
}

func (b *stubBlocker) Blocked(tile combat.Tile, _ entity.GameEntityID) bool {
	return b.blocked[tile]
}

var _ combat.TileBlocker = (*stubBlocker)(nil)

// mockClock is a stub for combat.Clock: it records pause/resume calls and
// exposes the paused state.
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

// recordingAttacker is a stub for combat.Attacker that always hits. Story 002
// never resolves attacks (Story 003), so it only needs to exist for the
// basic enemy AI wiring.
type recordingAttacker struct{}

func (a *recordingAttacker) ResolveAttack(_, _ entity.GameEntityID) combat.AttackResult {
	return combat.AttackResult{Hit: true, Damage: 1}
}

// recordingMover is a stub for combat.Mover that records every call and
// never fails — used where the tests must keep the enemy AI from actually
// moving or spending AP.
type recordingMover struct {
	moves []struct{ combatant, target entity.GameEntityID }
}

func (m *recordingMover) MoveToward(combatant, target entity.GameEntityID) error {
	m.moves = append(m.moves, struct{ combatant, target entity.GameEntityID }{combatant, target})
	return nil
}

// apSnapshot is the AP pool state observed at one turn boundary.
type apSnapshot struct {
	turnStart bool // true: TurnStarted; false: TurnEnded
	combatant entity.GameEntityID
	ap        int
}

func (s apSnapshot) String() string {
	kind := "ended"
	if s.turnStart {
		kind = "started"
	}
	return fmt.Sprintf("%s combatant=%d ap=%d", kind, s.combatant, s.ap)
}

// apRecorder snapshots the shared pool at every turn boundary. The pool is
// reset before TurnStarted dispatches and reflects the ending combatant's
// remaining AP at TurnEnded, so the snapshots prove the Story 002 turn
// lifecycle: reset at turn start, spent during the turn, no carryover.
type apRecorder struct {
	snapshots []apSnapshot
}

// assertSnapshots compares the recorded turn-boundary pool states with the
// expected sequence.
func (r *apRecorder) assertSnapshots(t *testing.T, want ...apSnapshot) {
	t.Helper()
	if !reflect.DeepEqual(r.snapshots, want) {
		t.Errorf("AP snapshots = %v, want %v", r.snapshots, want)
	}
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// moverWorld is a grid mover wired over stubs with a fresh pool reset to ap.
type moverWorld struct {
	positions *stubPositions
	blocked   *stubBlocker
	pool      *combat.APPool
	mover     *combat.GridMover
}

func newMoverWorld(t *testing.T, ap int) *moverWorld {
	t.Helper()
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	blocked := &stubBlocker{blocked: make(map[combat.Tile]bool)}
	pool := combat.NewAPPool()
	pool.Reset(ap)
	return &moverWorld{
		positions: positions,
		blocked:   blocked,
		pool:      pool,
		// Nil grid and opportunity: Story 002 tests move on level, open
		// ground without opportunity attacks.
		mover:     combat.NewGridMover(positions, positions, blocked, pool, nil, nil),
	}
}

// managerDeps exposes the shared wiring pieces to the AI builder so the AI
// (and the GridMover it uses) share the same store, positions, and AP pool
// as the manager.
type managerDeps struct {
	store     entity.EntityStore
	positions *stubPositions
	pool      *combat.APPool
}

// managerWorld is a combat manager wired over stubs with a shared AP pool
// and a turn-boundary AP recorder.
type managerWorld struct {
	bus       entity.EventBus
	store     entity.EntityStore
	clock     *mockClock
	positions *stubPositions
	pool      *combat.APPool
	manager   *combat.CombatManager
	recorder  *apRecorder
}

// newManagerWorld wires a combat manager with a shared pool; buildAI
// constructs the enemy AI (and its mover) from the shared pieces.
func newManagerWorld(t *testing.T, buildAI func(deps managerDeps) combat.EnemyAI) *managerWorld {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	clock := &mockClock{}
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	pool := combat.NewAPPool()
	ai := buildAI(managerDeps{store: store, positions: positions, pool: pool})
	manager := combat.NewCombatManager(bus, store, clock, ai, pool, combat.Config{})
	recorder := &apRecorder{}
	bus.Subscribe(combat.EventTurnStarted, 5, func(ev entity.Event) {
		data := ev.Data.(combat.TurnStartedData)
		recorder.snapshots = append(recorder.snapshots, apSnapshot{
			turnStart: true,
			combatant: data.Combatant,
			ap:        manager.CurrentAP(),
		})
	})
	bus.Subscribe(combat.EventTurnEnded, 5, func(ev entity.Event) {
		data := ev.Data.(combat.TurnEndedData)
		recorder.snapshots = append(recorder.snapshots, apSnapshot{
			turnStart: false,
			combatant: data.Combatant,
			ap:        manager.CurrentAP(),
		})
	})
	return &managerWorld{
		bus:       bus,
		store:     store,
		clock:     clock,
		positions: positions,
		pool:      pool,
		manager:   manager,
		recorder:  recorder,
	}
}

// addCombatant creates an entity with Stats and Health components, records
// its position in the stub positioner, and returns its GameEntityID.
func addCombatant(t *testing.T, store entity.EntityStore, positions *stubPositions, entityType entity.EntityType, name string, dex, hp int, pos matrix.Vec3) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(entityType, name, pos)
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	if err := entity.SetComponent(store, id, entity.Stats{Dexterity: dex}); err != nil {
		t.Fatalf("SetComponent(Stats, %s): %v", name, err)
	}
	if err := entity.SetComponent(store, id, entity.Health{Current: hp, Max: hp}); err != nil {
		t.Fatalf("SetComponent(Health, %s): %v", name, err)
	}
	positions.positions[id] = pos
	return id
}

// ---------------------------------------------------------------------------
// AC-1: diagonal movement costs 1 AP (same as orthogonal)
// ---------------------------------------------------------------------------

// QA-1: Given (0,0) with 1 AP, When moving to (1,1), Then 1 AP consumed,
// position (1,1). The diagonal step costs MoveCost (1) — the same as an
// orthogonal step (ADR-0009, TR-combat-002).
func TestDiagonalMoveCostsOneAP(t *testing.T) {
	// Arrange
	w := newMoverWorld(t, 1)
	combatant, target := entity.GameEntityID(1), entity.GameEntityID(2)
	w.positions.positions[combatant] = combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 1, Z: 1})

	// Act
	err := w.mover.MoveToward(combatant, target)

	// Assert
	if err != nil {
		t.Fatalf("MoveToward() error = %v, want nil", err)
	}
	if w.pool.Current() != 0 {
		t.Errorf("AP after diagonal move = %d, want 0 (1 AP consumed)", w.pool.Current())
	}
	got := w.positions.positions[combatant]
	if want := combat.TileToWorld(combat.Tile{X: 1, Z: 1}); got != want {
		t.Errorf("position after diagonal move = %v, want %v (tile (1,1))", got, want)
	}
}

// ---------------------------------------------------------------------------
// AC-2: move 5 tiles consumes 5 AP, 2 AP remain from 7
// ---------------------------------------------------------------------------

// QA-2: Given (0,0) with 7 AP, When moving 5 tiles, Then 5 AP consumed,
// AP = 2. The target sits 10 tiles away so all five MoveToward calls take a
// fresh step toward it (in-game the enemy AI attacks at adjacency instead of
// entering the target's tile).
func TestMoveConsumesOneAPPerTile(t *testing.T) {
	// Arrange
	w := newMoverWorld(t, 7)
	combatant, target := entity.GameEntityID(1), entity.GameEntityID(2)
	w.positions.positions[combatant] = combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 10, Z: 0})

	// Act — five steps of one tile each
	for i := 0; i < 5; i++ {
		if err := w.mover.MoveToward(combatant, target); err != nil {
			t.Fatalf("MoveToward step %d: %v", i+1, err)
		}
	}

	// Assert
	if w.pool.Current() != 2 {
		t.Errorf("AP after 5 moves = %d, want 2 (7 - 5)", w.pool.Current())
	}
	got := w.positions.positions[combatant]
	if want := combat.TileToWorld(combat.Tile{X: 5, Z: 0}); got != want {
		t.Errorf("position after 5 moves = %v, want %v (tile (5,0))", got, want)
	}
	if len(w.positions.moves) != 5 {
		t.Errorf("move count = %d, want 5", len(w.positions.moves))
	}
}

// ---------------------------------------------------------------------------
// AC-3: movement range = all tiles within Chebyshev distance <= AP
// ---------------------------------------------------------------------------

// QA-3: Given (0,0) with 2 AP, When calculating range, Then all Chebyshev
// <= 2 included. On an unobstructed 8-connected grid, the reachable set is
// exactly the square of tiles with Chebyshev(from, t) <= AP, from itself
// included (AC-3: combatant at (3,5) — covered by the second table row).
func TestReachableTilesWithinChebyshevBudget(t *testing.T) {
	open := &stubBlocker{} // nothing blocked

	for _, tc := range []struct {
		from combat.Tile
		ap   int
	}{
		{from: combat.Tile{X: 0, Z: 0}, ap: 2}, // QA-3
		{from: combat.Tile{X: 3, Z: 5}, ap: 3}, // AC-3 combatant at (3,5)
	} {
		// Arrange
		expected := make(map[combat.Tile]bool)
		for x := tc.from.X - tc.ap; x <= tc.from.X+tc.ap; x++ {
			for z := tc.from.Z - tc.ap; z <= tc.from.Z+tc.ap; z++ {
				expected[combat.Tile{X: x, Z: z}] = true
			}
		}

		// Act
		got := combat.ReachableTiles(tc.from, tc.ap, open, 0)

		// Assert
		if len(got) != len(expected) {
			t.Fatalf("ReachableTiles(%v, %d) = %d tiles, want %d", tc.from, tc.ap, len(got), len(expected))
		}
		for _, tile := range got {
			if !expected[tile] {
				t.Errorf("ReachableTiles(%v, %d) includes %v outside Chebyshev <= %d", tc.from, tc.ap, tile, tc.ap)
			}
			if d := combat.Chebyshev(tc.from, tile); d > tc.ap {
				t.Errorf("ReachableTiles(%v, %d) includes %v at Chebyshev %d", tc.from, tc.ap, tile, d)
			}
			delete(expected, tile)
		}
		if len(expected) != 0 {
			t.Errorf("ReachableTiles(%v, %d) misses %d tiles within Chebyshev <= %d", tc.from, tc.ap, len(expected), tc.ap)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-4: pathfinding excludes occupied/blocked tiles
// ---------------------------------------------------------------------------

// QA-4: Given obstacle at (1,0), When pathfinding, Then (1,0) excluded. The
// shortest path from (0,0) to (2,0) routes around the obstacle; with a
// budget too small to detour, no path exists.
func TestPathExcludesBlockedTiles(t *testing.T) {
	// Arrange
	blocked := &stubBlocker{blocked: map[combat.Tile]bool{
		combat.Tile{X: 1, Z: 0}: true, // the obstacle from QA-4
	}}

	// Act — budget 2: exactly enough to detour around the obstacle
	path, err := combat.Path(combat.Tile{X: 0, Z: 0}, combat.Tile{X: 2, Z: 0}, 2, blocked, 0)

	// Assert
	if err != nil {
		t.Fatalf("Path() error = %v, want nil (detour exists within budget)", err)
	}
	if len(path) != 2 {
		t.Fatalf("Path() = %v, want 2 steps", path)
	}
	if path[0] == (combat.Tile{X: 1, Z: 0}) {
		t.Errorf("Path() enters the blocked tile %v", path[0])
	}
	// Deterministic BFS: the detour passes north of the obstacle via (1,-1).
	if want := (combat.Tile{X: 1, Z: -1}); path[0] != want {
		t.Errorf("Path() first step = %v, want %v", path[0], want)
	}
	if want := (combat.Tile{X: 2, Z: 0}); path[1] != want {
		t.Errorf("Path() last step = %v, want %v", path[1], want)
	}

	// Budget 1 cannot detour — the direct neighbor is blocked.
	if _, err := combat.Path(combat.Tile{X: 0, Z: 0}, combat.Tile{X: 2, Z: 0}, 1, blocked, 0); err != combat.ErrNoPath {
		t.Errorf("Path(budget 1) error = %v, want ErrNoPath (detour needs 2 AP)", err)
	}

	// A path to the starting tile is not a path.
	if _, err := combat.Path(combat.Tile{X: 0, Z: 0}, combat.Tile{X: 0, Z: 0}, 4, blocked, 0); err != combat.ErrNoPath {
		t.Errorf("Path(same tile) error = %v, want ErrNoPath", err)
	}
}

// A combatant completely walled in can reach only its own tile.
func TestReachableTilesBlockedAllAround(t *testing.T) {
	// Arrange
	blocked := &stubBlocker{blocked: make(map[combat.Tile]bool)}
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			if dx == 0 && dz == 0 {
				continue
			}
			blocked.blocked[combat.Tile{X: dx, Z: dz}] = true
		}
	}

	// Act
	got := combat.ReachableTiles(combat.Tile{X: 0, Z: 0}, 4, blocked, 0)

	// Assert
	if len(got) != 1 || got[0] != (combat.Tile{X: 0, Z: 0}) {
		t.Errorf("ReachableTiles(walled in) = %v, want [(0,0)] only", got)
	}
}

// ---------------------------------------------------------------------------
// AC-5: attack costs 2 AP
// ---------------------------------------------------------------------------

// QA-5: Given 2 AP, When attacking, Then AP = 0. The attack itself is Story
// 003 (Out of Scope); this story defines the cost and the spend path.
func TestAttackCostSpendsTwoAP(t *testing.T) {
	// Arrange
	pool := combat.NewAPPool()

	// Act — full-budget attack
	pool.Reset(2)
	err := pool.Spend(combat.AttackCost)

	// Assert
	if err != nil {
		t.Fatalf("Spend(AttackCost) error = %v, want nil", err)
	}
	if pool.Current() != 0 {
		t.Errorf("AP after attack = %d, want 0", pool.Current())
	}
	if !pool.Empty() {
		t.Error("Empty() = false after spending the last AP, want true")
	}

	// Insufficient AP leaves the pool unchanged.
	pool.Reset(1)
	if err := pool.Spend(combat.AttackCost); err != combat.ErrInsufficientAP {
		t.Errorf("Spend(2) with 1 AP error = %v, want ErrInsufficientAP", err)
	}
	if pool.Current() != 1 {
		t.Errorf("AP after rejected spend = %d, want 1 (unchanged)", pool.Current())
	}

	// Non-positive costs are rejected.
	if err := pool.Spend(0); err != combat.ErrInvalidAPCost {
		t.Errorf("Spend(0) error = %v, want ErrInvalidAPCost", err)
	}
}

// ---------------------------------------------------------------------------
// TR-combat-002: AP = 4 + DEX_mod, clamped to [2, 8]
// ---------------------------------------------------------------------------

// DEX 16 → 7 AP (ADR-0009 validation criterion), DEX 10 → 4 AP, DEX 6 → 2
// AP (minimum), DEX 18 → 8 AP (maximum), DEX 3 → 2 AP (floor underflows).
func TestAPFor(t *testing.T) {
	for _, tc := range []struct {
		dex  int
		want int
	}{
		{dex: 16, want: 7},
		{dex: 10, want: 4},
		{dex: 6, want: 2},
		{dex: 18, want: 8},
		{dex: 3, want: 2},
	} {
		got := combat.APFor(entity.Stats{Dexterity: tc.dex})
		if got != tc.want {
			t.Errorf("APFor(DEX %d) = %d, want %d", tc.dex, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GridMover edge cases
// ---------------------------------------------------------------------------

// A combatant with no AP cannot move; the position is unchanged.
func TestMoveTowardInsufficientAP(t *testing.T) {
	// Arrange
	w := newMoverWorld(t, 0)
	combatant, target := entity.GameEntityID(1), entity.GameEntityID(2)
	start := combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	w.positions.positions[combatant] = start
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 3, Z: 0})

	// Act
	err := w.mover.MoveToward(combatant, target)

	// Assert
	if err != combat.ErrInsufficientAP {
		t.Fatalf("MoveToward() error = %v, want ErrInsufficientAP", err)
	}
	if w.positions.positions[combatant] != start {
		t.Errorf("position changed without AP: %v", w.positions.positions[combatant])
	}
	if len(w.positions.moves) != 0 {
		t.Errorf("moves = %v, want none", w.positions.moves)
	}
}

// MoveToward fails with ErrNoPath when combatant and target share a tile,
// or when either position is unavailable.
func TestMoveTowardNoPath(t *testing.T) {
	// Arrange
	w := newMoverWorld(t, 7)
	combatant, target := entity.GameEntityID(1), entity.GameEntityID(2)
	w.positions.positions[combatant] = combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 0, Z: 0}) // same tile

	// Same tile
	if err := w.mover.MoveToward(combatant, target); err != combat.ErrNoPath {
		t.Errorf("MoveToward(same tile) error = %v, want ErrNoPath", err)
	}

	// Missing target position
	delete(w.positions.positions, target)
	if err := w.mover.MoveToward(combatant, target); err != combat.ErrNoPath {
		t.Errorf("MoveToward(unknown target) error = %v, want ErrNoPath", err)
	}

	// Missing combatant position
	delete(w.positions.positions, combatant)
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 3, Z: 0})
	if err := w.mover.MoveToward(combatant, target); err != combat.ErrNoPath {
		t.Errorf("MoveToward(unknown combatant) error = %v, want ErrNoPath", err)
	}

	// A failed step never spends AP.
	if w.pool.Current() != 7 {
		t.Errorf("AP after failed moves = %d, want 7 (nothing spent)", w.pool.Current())
	}
}

// A fully blocked route fails without spending AP.
func TestMoveTowardBlockedRoute(t *testing.T) {
	// Arrange — wall the combatant in
	w := newMoverWorld(t, 7)
	combatant, target := entity.GameEntityID(1), entity.GameEntityID(2)
	w.positions.positions[combatant] = combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	w.positions.positions[target] = combat.TileToWorld(combat.Tile{X: 5, Z: 0})
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			if dx == 0 && dz == 0 {
				continue
			}
			w.blocked.blocked[combat.Tile{X: dx, Z: dz}] = true
		}
	}

	// Act
	err := w.mover.MoveToward(combatant, target)

	// Assert
	if err != combat.ErrNoPath {
		t.Fatalf("MoveToward() error = %v, want ErrNoPath", err)
	}
	if w.pool.Current() != 7 {
		t.Errorf("AP after blocked move = %d, want 7 (nothing spent)", w.pool.Current())
	}
	if len(w.positions.moves) != 0 {
		t.Errorf("moves = %v, want none", w.positions.moves)
	}
}

// ---------------------------------------------------------------------------
// Entity-backed TileBlocker (occupied tiles block; own tile never blocks)
// ---------------------------------------------------------------------------

func TestEntityTileBlocker(t *testing.T) {
	// Arrange
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	a := addCombatant(t, store, positions, entity.EntityTypeCharacter, "A", 10, 10, combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	b := addCombatant(t, store, positions, entity.EntityTypeNPC, "B", 10, 10, combat.TileToWorld(combat.Tile{X: 1, Z: 0}))
	prop, err := store.CreateEntity(entity.EntityTypeProp, "Crate", combat.TileToWorld(combat.Tile{X: 2, Z: 0}))
	if err != nil {
		t.Fatalf("CreateEntity(Crate): %v", err)
	}
	positions.positions[prop] = combat.TileToWorld(combat.Tile{X: 2, Z: 0})
	blocker := combat.NewEntityTileBlocker(store, positions)

	// Assert
	if !blocker.Blocked(combat.Tile{X: 1, Z: 0}, a) {
		t.Error("Blocked((1,0), A) = false, want true (occupied by B)")
	}
	if !blocker.Blocked(combat.Tile{X: 2, Z: 0}, a) {
		t.Error("Blocked((2,0), A) = false, want true (occupied by prop)")
	}
	if blocker.Blocked(combat.Tile{X: 0, Z: 0}, a) {
		t.Error("Blocked((0,0), A) = true, want false (own tile)")
	}
	if !blocker.Blocked(combat.Tile{X: 0, Z: 0}, b) {
		t.Error("Blocked((0,0), B) = false, want true (occupied by A)")
	}
	if blocker.Blocked(combat.Tile{X: 9, Z: 9}, a) {
		t.Error("Blocked((9,9), A) = true, want false (empty tile)")
	}
	_ = prop // the prop blocks by occupancy; no components required
}

// ---------------------------------------------------------------------------
// CombatManager integration: the shared pool resets at every turn start and
// never carries over (Story 002 implementation note)
// ---------------------------------------------------------------------------

// DEX 16 combatant (7 AP) and DEX 10 combatant (4 AP): each turn starts with
// its own budget; the player's second turn is 7 again, not the enemy's 4.
func TestTurnStartResetsAPPool(t *testing.T) {
	// Arrange — the AI uses a no-op mover so the enemy turn never spends AP
	w := newManagerWorld(t, func(deps managerDeps) combat.EnemyAI {
		return combat.NewBasicEnemyAI(deps.positions, &recordingAttacker{}, &recordingMover{}, combat.DefaultAttackRange)
	})
	pc := addCombatant(t, w.store, w.positions, entity.EntityTypeCharacter, "PC", 16, 20, combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	enemy := addCombatant(t, w.store, w.positions, entity.EntityTypeNPC, "Enemy", 10, 20, combat.TileToWorld(combat.Tile{X: 10, Z: 0}))

	// Act — player (DEX 16, initiative 3) acts first, then the enemy turn
	if err := w.manager.StartCombat([]combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: enemy, Side: combat.SideEnemy},
	}); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	if w.manager.CurrentAP() != 7 {
		t.Errorf("CurrentAP on player turn = %d, want 7 (DEX 16)", w.manager.CurrentAP())
	}
	if err := w.manager.EndTurn(); err != nil {
		t.Fatalf("EndTurn: %v", err)
	}

	// Assert — after the enemy's turn (and its own reset), the player is
	// back at 7: the enemy's 4 never carried over.
	w.recorder.assertSnapshots(t,
		apSnapshot{turnStart: true, combatant: pc, ap: 7},
		apSnapshot{turnStart: false, combatant: pc, ap: 7},
		apSnapshot{turnStart: true, combatant: enemy, ap: 4},
		apSnapshot{turnStart: false, combatant: enemy, ap: 4},
		apSnapshot{turnStart: true, combatant: pc, ap: 7},
	)
	if w.manager.CurrentAP() != 7 {
		t.Errorf("CurrentAP after full round = %d, want 7 (fresh pool, no carryover)", w.manager.CurrentAP())
	}
}

// When a turn's pool reaches 0 the turn ends (EndTurn), and the next turn
// starts with a fresh pool.
func TestEndTurnAfterAPExhausted(t *testing.T) {
	// Arrange — the AI uses a no-op mover so the enemy turn never spends AP
	w := newManagerWorld(t, func(deps managerDeps) combat.EnemyAI {
		return combat.NewBasicEnemyAI(deps.positions, &recordingAttacker{}, &recordingMover{}, combat.DefaultAttackRange)
	})
	pc := addCombatant(t, w.store, w.positions, entity.EntityTypeCharacter, "PC", 16, 20, combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	enemy := addCombatant(t, w.store, w.positions, entity.EntityTypeNPC, "Enemy", 10, 20, combat.TileToWorld(combat.Tile{X: 10, Z: 0}))
	if err := w.manager.StartCombat([]combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: enemy, Side: combat.SideEnemy},
	}); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	// Act — the player spends the whole pool (e.g. a 2 AP attack plus 5
	// moves), then ends the turn with 0 AP.
	if err := w.pool.Spend(combat.AttackCost); err != nil {
		t.Fatalf("Spend(AttackCost): %v", err)
	}
	for i := 0; i < 5; i++ {
		if err := w.pool.Spend(combat.MoveCost); err != nil {
			t.Fatalf("Spend(MoveCost): %v", err)
		}
	}
	if w.manager.CurrentAP() != 0 {
		t.Fatalf("CurrentAP = %d, want 0 (pool exhausted)", w.manager.CurrentAP())
	}
	if err := w.manager.EndTurn(); err != nil {
		t.Fatalf("EndTurn with 0 AP: %v", err)
	}

	// Assert — the exhausted turn ended and the next player turn refills.
	w.recorder.assertSnapshots(t,
		apSnapshot{turnStart: true, combatant: pc, ap: 7},
		apSnapshot{turnStart: false, combatant: pc, ap: 0},
		apSnapshot{turnStart: true, combatant: enemy, ap: 4},
		apSnapshot{turnStart: false, combatant: enemy, ap: 4},
		apSnapshot{turnStart: true, combatant: pc, ap: 7},
	)
	if w.manager.CurrentAP() != 7 {
		t.Errorf("CurrentAP after EndTurn = %d, want 7 (fresh pool)", w.manager.CurrentAP())
	}
}

// ---------------------------------------------------------------------------
// End-to-end: the enemy AI drives the real GridMover over the real
// entity-backed blocker through the CombatManager; the shared pool is spent
// by the move and reset at the next turn start.
// ---------------------------------------------------------------------------

func TestEnemyAIMovesWithGridMover(t *testing.T) {
	// Arrange — the AI drives the real GridMover over the real
	// entity-backed blocker, sharing the manager's store, positions, and
	// AP pool through the builder.
	w := newManagerWorld(t, func(deps managerDeps) combat.EnemyAI {
		blocker := combat.NewEntityTileBlocker(deps.store, deps.positions)
		mover := combat.NewGridMover(deps.positions, deps.positions, blocker, deps.pool, nil, nil)
		return combat.NewBasicEnemyAI(deps.positions, &recordingAttacker{}, mover, combat.DefaultAttackRange)
	})
	enemy := addCombatant(t, w.store, w.positions, entity.EntityTypeNPC, "Enemy", 16, 20, combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	pc := addCombatant(t, w.store, w.positions, entity.EntityTypeCharacter, "PC", 10, 20, combat.TileToWorld(combat.Tile{X: 3, Z: 0}))

	// Act — the enemy (DEX 16, initiative 3) acts first: 3 tiles from the
	// player, so it moves one tile toward it instead of attacking.
	if err := w.manager.StartCombat([]combat.CombatantEntry{
		{EntityID: enemy, Side: combat.SideEnemy},
		{EntityID: pc, Side: combat.SidePlayer},
	}); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	// Assert
	w.recorder.assertSnapshots(t,
		apSnapshot{turnStart: true, combatant: enemy, ap: 7},  // DEX 16 → 7 AP
		apSnapshot{turnStart: false, combatant: enemy, ap: 6}, // one tile moved, 1 AP spent
		apSnapshot{turnStart: true, combatant: pc, ap: 4},     // DEX 10 → 4 AP, fresh pool
	)
	got := w.positions.positions[enemy]
	// Direction-aware BFS: the first step of the shortest path is the
	// orthogonal (1,0) — neighbors are ordered by progress toward the
	// destination (Chebyshev, then Manhattan), so the path stays straight
	// east instead of staircasing diagonally.
	if want := combat.TileToWorld(combat.Tile{X: 1, Z: 0}); got != want {
		t.Errorf("enemy position = %v, want %v (one tile toward the player)", got, want)
	}
	if len(w.positions.moves) != 1 {
		t.Errorf("moves = %v, want exactly one step", w.positions.moves)
	}
	if w.manager.CurrentAP() != 4 {
		t.Errorf("CurrentAP after enemy turn = %d, want 4 (player's fresh pool)", w.manager.CurrentAP())
	}
	if !w.clock.paused || w.clock.resumeCalls != 0 {
		t.Errorf("clock paused = %v (want true), resume calls = %d (want 0)", w.clock.paused, w.clock.resumeCalls)
	}
}

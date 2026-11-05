// Unit tests for Story 004: Positioning & Cover System.
//
// Evidence path: production/epics/tactical-combat-engine/
// story-004-positioning-cover-system.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria and QA test cases map to test functions 1:1 (Go
// only executes Test-prefixed functions, so each story name is exported with
// the Go prefix):
//
//	AC-1  test_flanking_grants_+2_accuracy          -> TestFlankingAddsTwoAccuracy
//	AC-2  test_half_cover_grants_+2_evasion         -> TestHalfCoverAddsEvasion
//	AC-3  test_full_cover_blocks_line_of_sight      -> TestFullCoverBlocksLineOfSight
//	AC-4  test_elevation_grants_+2_acc_+2_range     -> TestElevationGrantsAccuracyAndRange
//	AC-5  test_adjacent_open_ground_no_modifiers    -> TestAdjacentOpenGroundNoModifiers
//	(edge) test_flanking_requires_opposite_side     -> TestFlankingRequiresOppositeSide
//	(edge) test_flanking_requires_adjacent_ally     -> TestFlankingRequiresAdjacentAlly
//	(edge) test_flanking_requires_ally_relationship -> TestFlankingRequiresAllyRelationship
//	(edge) test_flanking_skips_positionless_ally    -> TestFlankingSkipsPositionlessAlly
//	(edge) test_cover_requires_obstacle_between     -> TestCoverRequiresObstacleBetween
//	(edge) test_own_tile_cover_ignored              -> TestCoverIgnoresOwnTileCover
//	(edge) test_full_cover_grants_no_evasion        -> TestFullCoverGrantsNoEvasion
//	(edge) test_los_excludes_endpoint_tiles         -> TestLineOfSightExcludesEndpointTiles
//	(edge) test_elevation_no_bonus_when_lower       -> TestElevationNoBonusWhenLowerOrLevel
//	(edge) test_modifiers_additive                  -> TestModifiersAdditiveFlankingPlusElevation
//	(int)  test_resolve_applies_flanking            -> TestResolveAttackAppliesFlankingBonus
//	(int)  test_resolve_applies_cover_evasion       -> TestResolveAttackAppliesCoverEvasion
//	(int)  test_resolve_blocked_by_full_cover       -> TestResolveAttackBlockedByFullCover
//	(OA)   test_attack_when_leaving_melee_range     -> TestOpportunityAttackWhenLeavingMeleeRange
//	(OA)   test_no_attack_while_still_adjacent      -> TestOpportunityAttackNotWhileStillAdjacent
//	(OA)   test_no_attack_when_approaching          -> TestOpportunityAttackNotWhenApproaching
//	(OA)   test_once_per_turn_per_enemy             -> TestOpportunityAttackOncePerTurnPerEnemy
//	(OA)   test_reset_on_turn_event                 -> TestOpportunityAttackResetOnTurnEvent
//	(OA)   test_only_enemies_attack                 -> TestOpportunityAttackIgnoresNonEnemies
//	(OA)   test_skips_positionless_enemies          -> TestOpportunityAttackSkipsUnknownEnemyPosition
//	(OA)   test_multiple_enemies_attack_once_each   -> TestOpportunityAttackMultipleEnemies
//	(OA)   test_mover_integration                   -> TestGridMoverTriggersOpportunityAttack
//	(OA)   test_mover_reports_step                  -> TestGridMoverReportsStepToWatcher
//	(GRID) test_brezenham_line                      -> TestLineTilesBresenham
//	(GRID) test_line_last_tile_adjacent_to_target   -> TestLineTilesLastTileAdjacentToDestination
//	(CLIMB) test_climb_costs_2_ap                   -> TestClimbingStepCostsTwoAP
//	(CLIMB) test_level_step_costs_1_ap              -> TestLevelStepCostsOneAP
//	(CLIMB) test_descending_step_costs_1_ap         -> TestDescendingStepCostsOneAP
//	(CLIMB) test_insufficient_ap                    -> TestClimbingStepInsufficientAP
//	(FR)   test_faction_camp_model                  -> TestManagerFactionsCampModel
//
// Test doubles (stubPositions, mockKaijuHost, stubBlocker, fixedRoller,
// stubEquipment, addCombatant) come from grid_movement_test.go and
// attack_resolution_test.go (same package).
package combat_test

import (
	"errors"
	"reflect"
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// stubFactions implements combat.FactionReader over preset ally and enemy
// lists keyed by entity ID. An entity without an entry has no faction
// relationships.
type stubFactions struct {
	allies  map[entity.GameEntityID][]entity.GameEntityID
	enemies map[entity.GameEntityID][]entity.GameEntityID
}

func (f *stubFactions) AlliesOf(id entity.GameEntityID) []entity.GameEntityID {
	return f.allies[id]
}

func (f *stubFactions) EnemiesOf(id entity.GameEntityID) []entity.GameEntityID {
	return f.enemies[id]
}

var _ combat.FactionReader = (*stubFactions)(nil)

// countingAttacker is a stub for combat.Attacker that records every
// resolution call and always hits.
type countingAttacker struct {
	attacks []struct{ attacker, target entity.GameEntityID }
}

func (a *countingAttacker) ResolveAttack(attacker, target entity.GameEntityID) combat.AttackResult {
	a.attacks = append(a.attacks, struct{ attacker, target entity.GameEntityID }{attacker, target})
	return combat.AttackResult{Hit: true, Damage: 1}
}

var _ combat.Attacker = (*countingAttacker)(nil)

// stubSides implements combat.SideSource over a preset turn order.
type stubSides struct {
	order []combat.TurnEntry
}

func (s *stubSides) TurnOrder() []combat.TurnEntry {
	return append([]combat.TurnEntry(nil), s.order...)
}

// watchCall is one recorded CheckStep call.
type watchCall struct {
	mover entity.GameEntityID
	from  combat.Tile
	to    combat.Tile
}

// stubWatcher is a stub for combat.OpportunityWatcher that records every
// step report.
type stubWatcher struct {
	calls []watchCall
}

func (w *stubWatcher) CheckStep(mover entity.GameEntityID, from, to combat.Tile) int {
	w.calls = append(w.calls, watchCall{mover: mover, from: from, to: to})
	return 0
}

var _ combat.OpportunityWatcher = (*stubWatcher)(nil)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// positioningWorld wires the grid positioning resolver and the d20 attacker
// over stubs with a fixed roll.
type positioningWorld struct {
	store       entity.EntityStore
	positions   *stubPositions
	factions    *stubFactions
	grid        *combat.TileGrid
	equipment   *stubEquipment
	resolve     combat.Attacker
	positioning combat.PositioningResolver
}

// newPositioningWorld creates the resolver (NewGridPositioning over the
// shared stub positions/factions/grid) and the d20 attacker consuming it.
func newPositioningWorld(t *testing.T, roll int) *positioningWorld {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	factions := &stubFactions{
		allies:  make(map[entity.GameEntityID][]entity.GameEntityID),
		enemies: make(map[entity.GameEntityID][]entity.GameEntityID),
	}
	grid := combat.NewTileGrid()
	positioning := combat.NewGridPositioning(positions, grid, factions)
	equipment := &stubEquipment{
		weapons: make(map[entity.GameEntityID]combat.WeaponStats),
		armor:   make(map[entity.GameEntityID]int),
	}
	resolve := combat.NewD20Attacker(store, equipment, &fixedRoller{roll: roll}, positioning)
	return &positioningWorld{
		store:       store,
		positions:   positions,
		factions:    factions,
		grid:        grid,
		equipment:   equipment,
		resolve:     resolve,
		positioning: positioning,
	}
}

// addPlaced creates a combatant at the tile's center with the given Stats
// (and full Health) and records its position in the stub positioner.
func addPlaced(t *testing.T, w *positioningWorld, name string, tile combat.Tile, stats entity.Stats) entity.GameEntityID {
	t.Helper()
	id, err := w.store.CreateEntity(entity.EntityTypeCharacter, name, combat.TileToWorld(tile))
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	if err := entity.SetComponent(w.store, id, stats); err != nil {
		t.Fatalf("SetComponent(Stats, %s): %v", name, err)
	}
	if err := entity.SetComponent(w.store, id, entity.Health{Current: 10, Max: 10}); err != nil {
		t.Fatalf("SetComponent(Health, %s): %v", name, err)
	}
	w.positions.positions[id] = combat.TileToWorld(tile)
	return id
}

// opportunityWorld wires the opportunity attacker over stubs with a
// counting attacker.
type opportunityWorld struct {
	bus       entity.EventBus
	store     entity.EntityStore
	positions *stubPositions
	factions  *stubFactions
	attacker  *countingAttacker
	opp       *combat.OpportunityAttacker
}

func newOpportunityWorld(t *testing.T) *opportunityWorld {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	factions := &stubFactions{
		allies:  make(map[entity.GameEntityID][]entity.GameEntityID),
		enemies: make(map[entity.GameEntityID][]entity.GameEntityID),
	}
	attacker := &countingAttacker{}
	opp := combat.NewOpportunityAttacker(bus, positions, attacker, factions)
	return &opportunityWorld{
		bus:       bus,
		store:     store,
		positions: positions,
		factions:  factions,
		attacker:  attacker,
		opp:       opp,
	}
}

// addOpponent creates an entity at the tile's center and records its
// position; opportunity checks need positions only.
func addOpponent(t *testing.T, w *opportunityWorld, name string, tile combat.Tile) entity.GameEntityID {
	t.Helper()
	id, err := w.store.CreateEntity(entity.EntityTypeCharacter, name, combat.TileToWorld(tile))
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	w.positions.positions[id] = combat.TileToWorld(tile)
	return id
}

// moverGridWorld wires the grid mover over a TileGrid for the climb-cost
// tests.
type moverGridWorld struct {
	store     entity.EntityStore
	positions *stubPositions
	grid      *combat.TileGrid
	pool      *combat.APPool
	mover     *combat.GridMover
}

func newMoverGridWorld(t *testing.T, ap int) *moverGridWorld {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	grid := combat.NewTileGrid()
	pool := combat.NewAPPool()
	pool.Reset(ap)
	return &moverGridWorld{
		store:     store,
		positions: positions,
		grid:      grid,
		pool:      pool,
		mover:     combat.NewGridMover(positions, positions, &stubBlocker{blocked: make(map[combat.Tile]bool)}, pool, grid, nil),
	}
}

// addMoverEntity creates an entity at the tile's center and records its
// position.
func addMoverEntity(t *testing.T, w *moverGridWorld, name string, tile combat.Tile) entity.GameEntityID {
	t.Helper()
	id, err := w.store.CreateEntity(entity.EntityTypeCharacter, name, combat.TileToWorld(tile))
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	w.positions.positions[id] = combat.TileToWorld(tile)
	return id
}

// ---------------------------------------------------------------------------
// AC-1: attacker(2,3), defender(3,3), ally(4,3) → flanking +2 accuracy
// ---------------------------------------------------------------------------

// QA-1: Given an attacker and an adjacent ally on opposite sides of the
// defender (all three collinear on X), When the attacker attacks, Then the
// flanking bonus is +2 accuracy (ADR-0009) and no other modifiers apply.
func TestFlankingAddsTwoAccuracy(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{Evasion: 14})
	ally := addPlaced(t, w, "Ally", combat.Tile{X: 4, Z: 3}, entity.Stats{})
	w.factions.allies[attacker] = []entity.GameEntityID{ally}

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != combat.FlankingBonus {
		t.Fatalf("Modifiers accuracy: want %d (flanking), got %+v", combat.FlankingBonus, mods)
	}
	if mods.Evasion != 0 || mods.Range != 0 {
		t.Fatalf("Modifiers: want evasion/range 0, got %+v", mods)
	}
}

// Edge: an attacker perpendicular to the ally is NOT on the opposite side
// (the 180° arc check is strict): attacker (3,4) above the defender while
// the ally stands at (4,3) — no flanking bonus.
func TestFlankingRequiresOppositeSide(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 3, Z: 4}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	ally := addPlaced(t, w, "Ally", combat.Tile{X: 4, Z: 3}, entity.Stats{})
	w.factions.allies[attacker] = []entity.GameEntityID{ally}

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 0 {
		t.Fatalf("Modifiers accuracy: want 0 (perpendicular, not opposite), got %+v", mods)
	}
}

// Edge: the ally must be adjacent to the target (GDD "Flanking requires the
// ally to be adjacent to the target"): ally at (5,3) is Chebyshev 2 away.
func TestFlankingRequiresAdjacentAlly(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	ally := addPlaced(t, w, "Ally", combat.Tile{X: 5, Z: 3}, entity.Stats{})
	w.factions.allies[attacker] = []entity.GameEntityID{ally}

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 0 {
		t.Fatalf("Modifiers accuracy: want 0 (ally not adjacent), got %+v", mods)
	}
}

// Edge: a positioned entity on the opposite side is not an ally — the
// FactionReader's ally list is authoritative.
func TestFlankingRequiresAllyRelationship(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	addPlaced(t, w, "Stranger", combat.Tile{X: 4, Z: 3}, entity.Stats{}) // not in the ally list
	// w.factions.allies[attacker] stays empty

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 0 {
		t.Fatalf("Modifiers accuracy: want 0 (stranger is not an ally), got %+v", mods)
	}
}

// Edge: an ally without a recorded position cannot hold the flank.
func TestFlankingSkipsPositionlessAlly(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	ally, err := w.store.CreateEntity(entity.EntityTypeCharacter, "GhostAlly", matrix.Vec3{})
	if err != nil {
		t.Fatalf("CreateEntity(GhostAlly): %v", err)
	}
	w.factions.allies[attacker] = []entity.GameEntityID{ally} // no position recorded

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 0 {
		t.Fatalf("Modifiers accuracy: want 0 (ally has no position), got %+v", mods)
	}
}

// ---------------------------------------------------------------------------
// AC-2: defender adjacent to half cover → +2 evasion
// ---------------------------------------------------------------------------

// QA-2: Given a defender behind a low wall (half cover on the tile between
// the attacker and the defender, adjacent to the defender), When attacked,
// Then the defender gains +2 evasion (ADR-0009) and no other modifiers.
func TestHalfCoverAddsEvasion(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	w.grid.SetCover(combat.Tile{X: 2, Z: 3}, combat.CoverHalf)

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Evasion != combat.HalfCoverBonus {
		t.Fatalf("Modifiers evasion: want %d (half cover), got %+v", combat.HalfCoverBonus, mods)
	}
	if mods.Accuracy != 0 || mods.Range != 0 {
		t.Fatalf("Modifiers: want accuracy/range 0, got %+v", mods)
	}
}

// Edge: the obstacle must lie between the attacker and the defender — a wall
// behind the defender (off the attack line) grants nothing.
func TestCoverRequiresObstacleBetween(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 4, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	w.grid.SetCover(combat.Tile{X: 2, Z: 3}, combat.CoverHalf) // behind the defender

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Evasion != 0 {
		t.Fatalf("Modifiers evasion: want 0 (wall behind defender), got %+v", mods)
	}
}

// Edge: cover on the combatants' own tiles is ignored — the defender's tile
// and the attacker's tile never grant the defender cover (the story specs
// obstacles adjacent to the defender on the attack line).
func TestCoverIgnoresOwnTileCover(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	w.grid.SetCover(combat.Tile{X: 1, Z: 3}, combat.CoverHalf) // attacker's own tile
	w.grid.SetCover(combat.Tile{X: 3, Z: 3}, combat.CoverHalf) // defender's own tile

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Evasion != 0 {
		t.Fatalf("Modifiers evasion: want 0 (own-tile cover ignored), got %+v", mods)
	}
}

// Edge: full cover blocks line of sight but grants no evasion bonus.
func TestFullCoverGrantsNoEvasion(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	w.grid.SetCover(combat.Tile{X: 2, Z: 3}, combat.CoverFull)

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Evasion != 0 {
		t.Fatalf("Modifiers evasion: want 0 (full cover grants no evasion), got %+v", mods)
	}
}

// ---------------------------------------------------------------------------
// AC-3: solid wall between attacker and defender → line of sight blocked
// ---------------------------------------------------------------------------

// QA-3: Given a solid wall (full cover) on the tile between the combatants,
// When the attacker targets the defender, Then line of sight is blocked —
// and half cover does not block it.
func TestFullCoverBlocksLineOfSight(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	between := combat.Tile{X: 2, Z: 3}

	// Act & Assert: full cover blocks.
	w.grid.SetCover(between, combat.CoverFull)
	if w.positioning.HasLineOfSight(attacker, defender) {
		t.Fatalf("HasLineOfSight: want blocked through full cover, got clear")
	}

	// Act & Assert: half cover does not block.
	w.grid.SetCover(between, combat.CoverHalf)
	if !w.positioning.HasLineOfSight(attacker, defender) {
		t.Fatalf("HasLineOfSight: want clear through half cover, got blocked")
	}
}

// Edge: the combatants' own tiles are endpoints of the line — full cover on
// them never blocks their own line.
func TestLineOfSightExcludesEndpointTiles(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	w.grid.SetCover(combat.Tile{X: 1, Z: 3}, combat.CoverFull) // attacker's own tile
	w.grid.SetCover(combat.Tile{X: 3, Z: 3}, combat.CoverFull) // defender's own tile

	// Act
	clear := w.positioning.HasLineOfSight(attacker, defender)

	// Assert
	if !clear {
		t.Fatalf("HasLineOfSight: want clear (endpoint tiles excluded), got blocked")
	}
}

// ---------------------------------------------------------------------------
// AC-4: attacker 2 levels above defender → +2 accuracy, +2 range
// ---------------------------------------------------------------------------

// QA-4: Given an attacker at elevation 3 and a defender at elevation 1,
// When the attacker attacks, Then accuracy +2 and range +2 (ADR-0009: +1 per
// level above).
func TestElevationGrantsAccuracyAndRange(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 0, Z: 0}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 0}, entity.Stats{})
	w.grid.SetElevation(combat.Tile{X: 0, Z: 0}, 3)
	w.grid.SetElevation(combat.Tile{X: 3, Z: 0}, 1)

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 2 || mods.Range != 2 {
		t.Fatalf("Modifiers: want accuracy 2 and range 2 (2 levels above), got %+v", mods)
	}
	if mods.Evasion != 0 {
		t.Fatalf("Modifiers evasion: want 0, got %+v", mods)
	}
}

// Edge: an attacker below or level with the defender gains nothing — the
// bonus applies to higher ground only.
func TestElevationNoBonusWhenLowerOrLevel(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 0, Z: 0}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 0}, entity.Stats{})

	// Act & Assert: attacker below (1 vs 3).
	w.grid.SetElevation(combat.Tile{X: 0, Z: 0}, 1)
	w.grid.SetElevation(combat.Tile{X: 3, Z: 0}, 3)
	if mods := w.positioning.Modifiers(attacker, defender); mods.Accuracy != 0 || mods.Range != 0 {
		t.Fatalf("Modifiers below: want 0/0, got %+v", mods)
	}

	// Act & Assert: level ground (2 vs 2).
	w.grid.SetElevation(combat.Tile{X: 0, Z: 0}, 2)
	w.grid.SetElevation(combat.Tile{X: 3, Z: 0}, 2)
	if mods := w.positioning.Modifiers(attacker, defender); mods.Accuracy != 0 || mods.Range != 0 {
		t.Fatalf("Modifiers level: want 0/0, got %+v", mods)
	}
}

// ---------------------------------------------------------------------------
// AC-5: attacker adjacent to defender, open ground → no modifiers
// ---------------------------------------------------------------------------

// QA-5: Given adjacent combatants on open, level ground with no allies,
// When the attacker attacks, Then no modifiers apply.
func TestAdjacentOpenGroundNoModifiers(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods != (combat.PositioningModifiers{}) {
		t.Fatalf("Modifiers: want zero value on open ground, got %+v", mods)
	}
}

// ---------------------------------------------------------------------------
// Modifier additivity (implementation notes: flanking + elevation = +3)
// ---------------------------------------------------------------------------

// Edge: positional bonuses are additive (ADR-0009 "Modifiers additive") —
// flanking +2 and one elevation level +1 sum to +3 accuracy, with the
// elevation range bonus applied alongside.
func TestModifiersAdditiveFlankingPlusElevation(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 10)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{})
	ally := addPlaced(t, w, "Ally", combat.Tile{X: 4, Z: 3}, entity.Stats{})
	w.factions.allies[attacker] = []entity.GameEntityID{ally}
	w.grid.SetElevation(combat.Tile{X: 2, Z: 3}, 1) // one level above the defender

	// Act
	mods := w.positioning.Modifiers(attacker, defender)

	// Assert
	if mods.Accuracy != 3 || mods.Range != 1 {
		t.Fatalf("Modifiers: want accuracy 3 (2 flanking + 1 elevation) and range 1, got %+v", mods)
	}
}

// ---------------------------------------------------------------------------
// Line tiles (grid-level Bresenham)
// ---------------------------------------------------------------------------

// GRID: LineTiles walks the deterministic Bresenham line between the tiles,
// endpoints excluded: (0,0)→(3,1) crosses (1,0),(2,1); (0,0)→(2,2) crosses
// (1,1); adjacent and co-located tiles have no intermediate tiles; negative
// directions mirror the positive walk.
func TestLineTilesBresenham(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		from combat.Tile
		to   combat.Tile
		want []combat.Tile
	}{
		{name: "shallow", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 3, Z: 1}, want: []combat.Tile{{X: 1, Z: 0}, {X: 2, Z: 1}}},
		{name: "diagonal", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 2, Z: 2}, want: []combat.Tile{{X: 1, Z: 1}}},
		{name: "steep", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 1, Z: 3}, want: []combat.Tile{{X: 0, Z: 1}, {X: 1, Z: 2}}},
		{name: "negative", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: -2, Z: -1}, want: []combat.Tile{{X: -1, Z: 0}}},
		{name: "adjacent", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 1, Z: 0}, want: nil},
		{name: "same tile", from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 0, Z: 0}, want: nil},
	}

	// Act & Assert
	for _, tc := range cases {
		got := combat.LineTiles(tc.from, tc.to)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: LineTiles(%+v, %+v) = %v, want %v", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

// GRID: the last intermediate tile of the line is the tile adjacent to the
// destination — the "obstacle adjacent to the defender" position the
// half-cover check relies on.
func TestLineTilesLastTileAdjacentToDestination(t *testing.T) {
	// Arrange
	line := combat.LineTiles(combat.Tile{X: 0, Z: 0}, combat.Tile{X: 4, Z: 2})

	// Assert
	if len(line) == 0 {
		t.Fatalf("LineTiles: want intermediate tiles for a Chebyshev-4 line, got none")
	}
	last := line[len(line)-1]
	if got := combat.Chebyshev(last, combat.Tile{X: 4, Z: 2}); got != 1 {
		t.Fatalf("last tile %+v: want Chebyshev 1 to the destination, got %d", last, got)
	}
}

// ---------------------------------------------------------------------------
// Attack resolution integration (positioning feeds the d20 roll)
// ---------------------------------------------------------------------------

// INT: flanking +2 accuracy turns a miss into a hit — accuracy 10 + roll 3
// = 13 < evasion 14 without the ally, 15 >= 14 with it.
func TestResolveAttackAppliesFlankingBonus(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 3)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 2, Z: 3}, entity.Stats{Strength: 10})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{Evasion: 14})
	ally := addPlaced(t, w, "Ally", combat.Tile{X: 4, Z: 3}, entity.Stats{})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 4}

	// Act & Assert: without the flanking ally, the attack misses.
	if got := w.resolve.ResolveAttack(attacker, defender); got.Hit {
		t.Fatalf("ResolveAttack: want miss without flanking (10+3=13 < 14), got hit")
	}

	// Act & Assert: the ally joins the flank — the attack hits.
	w.factions.allies[attacker] = []entity.GameEntityID{ally}
	got := w.resolve.ResolveAttack(attacker, defender)
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit with flanking (10+2+3=15 >= 14), got miss")
	}
}

// INT: half-cover +2 evasion turns a hit into a miss — accuracy 12 + roll 3
// = 15 >= evasion 14 without the wall, 15 < 16 with it.
func TestResolveAttackAppliesCoverEvasion(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 3)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{Strength: 10})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{Evasion: 14})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 12, BaseDamage: 4}

	// Act & Assert: open ground — the attack hits.
	if got := w.resolve.ResolveAttack(attacker, defender); !got.Hit {
		t.Fatalf("ResolveAttack: want hit on open ground (12+3=15 >= 14), got miss")
	}

	// Act & Assert: the low wall goes up — the attack misses.
	w.grid.SetCover(combat.Tile{X: 2, Z: 3}, combat.CoverHalf)
	if got := w.resolve.ResolveAttack(attacker, defender); got.Hit {
		t.Fatalf("ResolveAttack: want miss behind half cover (15 < 14+2=16), got hit")
	}
}

// INT: full cover blocks the attack entirely — even a natural 20 resolves as
// a miss (ADR-0009: "Full cover: blocks line of sight").
func TestResolveAttackBlockedByFullCover(t *testing.T) {
	// Arrange
	w := newPositioningWorld(t, 20)
	attacker := addPlaced(t, w, "Attacker", combat.Tile{X: 1, Z: 3}, entity.Stats{Strength: 18})
	defender := addPlaced(t, w, "Defender", combat.Tile{X: 3, Z: 3}, entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 20, BaseDamage: 10}
	w.grid.SetCover(combat.Tile{X: 2, Z: 3}, combat.CoverFull)

	// Act
	got := w.resolve.ResolveAttack(attacker, defender)

	// Assert
	if got.Hit || got.Crit || got.Damage != 0 {
		t.Fatalf("ResolveAttack: want blocked miss (no hit/crit/damage) through full cover, got %+v", got)
	}
	if got.Roll != 20 {
		t.Fatalf("ResolveAttack roll: want 20 reported, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// Opportunity attacks (ADR-0009: leaving melee range triggers a free attack)
// ---------------------------------------------------------------------------

// OA-1: a combatant stepping out of an enemy's melee range (Chebyshev 1 →
// 2) gives that enemy a free attack against the mover.
func TestOpportunityAttackWhenLeavingMeleeRange(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy}

	// Act
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})

	// Assert
	if attacks != 1 {
		t.Fatalf("CheckStep: want 1 attack, got %d", attacks)
	}
	if len(w.attacker.attacks) != 1 || w.attacker.attacks[0].attacker != enemy || w.attacker.attacks[0].target != mover {
		t.Fatalf("CheckStep attacks: want [enemy → mover], got %+v", w.attacker.attacks)
	}
}

// OA: a step that stays within melee range (Chebyshev 1 → 1) triggers
// nothing.
func TestOpportunityAttackNotWhileStillAdjacent(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy}

	// Act — (1,0) → (1,1) circles the enemy while staying adjacent.
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 1, Z: 1})

	// Assert
	if attacks != 0 {
		t.Fatalf("CheckStep: want 0 attacks while still adjacent, got %d", attacks)
	}
}

// OA: a step toward the enemy (Chebyshev 2 → 1) is not leaving melee range.
func TestOpportunityAttackNotWhenApproaching(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 2, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy}

	// Act — (2,0) → (1,0) moves into range, not out of it.
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 2, Z: 0}, combat.Tile{X: 1, Z: 0})

	// Assert
	if attacks != 0 {
		t.Fatalf("CheckStep: want 0 attacks when approaching, got %d", attacks)
	}
}

// OA: once per turn per enemy (ADR-0009) — leaving the same enemy's range a
// second time in the same turn triggers nothing until ResetTurn.
func TestOpportunityAttackOncePerTurnPerEnemy(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy}

	// Act — leave (1,0)→(2,0), re-enter (2,0)→(1,0), leave again.
	w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})
	w.opp.CheckStep(mover, combat.Tile{X: 2, Z: 0}, combat.Tile{X: 1, Z: 0})
	second := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})

	// Assert
	if second != 0 {
		t.Fatalf("CheckStep: want 0 attacks (already used this turn), got %d", second)
	}
	if len(w.attacker.attacks) != 1 {
		t.Fatalf("CheckStep attacks: want exactly 1 this turn, got %d", len(w.attacker.attacks))
	}

	// Act & Assert — the next turn (ResetTurn) allows the enemy to attack again.
	w.opp.ResetTurn()
	third := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})
	if third != 1 {
		t.Fatalf("CheckStep after reset: want 1 attack, got %d", third)
	}
	if len(w.attacker.attacks) != 2 {
		t.Fatalf("CheckStep attacks after reset: want 2 total, got %d", len(w.attacker.attacks))
	}
}

// OA: the once-per-turn budget resets automatically when EventTurnStarted
// fires — the constructor subscription aligns the budget with the combat
// manager's turn boundaries.
func TestOpportunityAttackResetOnTurnEvent(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy}
	leave := func() int {
		return w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})
	}

	// Act — leave, leave again (blocked by the once-per-turn rule), then a
	// turn starts.
	if got := leave(); got != 1 {
		t.Fatalf("first leave: want 1 attack, got %d", got)
	}
	if got := leave(); got != 0 {
		t.Fatalf("second leave: want 0 attacks (once per turn), got %d", got)
	}
	w.bus.Emit(entity.Event{
		Type:   combat.EventTurnStarted,
		Source: "combat",
		Data:   combat.TurnStartedData{},
	})

	// Assert — the new turn restored the enemy's opportunity attack.
	if got := leave(); got != 1 {
		t.Fatalf("leave after turn event: want 1 attack, got %d", got)
	}
}

// OA: only enemies of the mover attack — a friendly entity stepping out of
// another friendly's range triggers nothing.
func TestOpportunityAttackIgnoresNonEnemies(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	addOpponent(t, w, "Friend", combat.Tile{X: 0, Z: 0}) // positioned, but not an enemy
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	// w.factions.enemies[mover] stays empty — the friend is not an enemy.

	// Act
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})

	// Assert
	if attacks != 0 {
		t.Fatalf("CheckStep: want 0 attacks (friend is not an enemy), got %d", attacks)
	}
	if len(w.attacker.attacks) != 0 {
		t.Fatalf("CheckStep attacks: want none, got %+v", w.attacker.attacks)
	}
}

// OA: an enemy without a recorded position cannot be judged in range —
// skipped defensively.
func TestOpportunityAttackSkipsUnknownEnemyPosition(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemy, err := w.store.CreateEntity(entity.EntityTypeCharacter, "GhostEnemy", matrix.Vec3{})
	if err != nil {
		t.Fatalf("CreateEntity(GhostEnemy): %v", err)
	}
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	w.factions.enemies[mover] = []entity.GameEntityID{enemy} // no position recorded

	// Act
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 0}, combat.Tile{X: 2, Z: 0})

	// Assert
	if attacks != 0 {
		t.Fatalf("CheckStep: want 0 attacks (enemy position unknown), got %d", attacks)
	}
}

// OA: several enemies in range each take their free attack — but only once.
func TestOpportunityAttackMultipleEnemies(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	enemyA := addOpponent(t, w, "EnemyA", combat.Tile{X: 0, Z: 0})
	enemyB := addOpponent(t, w, "EnemyB", combat.Tile{X: 0, Z: 2})
	mover := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 1})
	w.factions.enemies[mover] = []entity.GameEntityID{enemyA, enemyB}

	// Act — (1,1) → (2,1) leaves both enemies' melee range.
	attacks := w.opp.CheckStep(mover, combat.Tile{X: 1, Z: 1}, combat.Tile{X: 2, Z: 1})

	// Assert
	if attacks != 2 {
		t.Fatalf("CheckStep: want 2 attacks (one per enemy), got %d", attacks)
	}
}

// ---------------------------------------------------------------------------
// GridMover integration (movement flows through the opportunity hook)
// ---------------------------------------------------------------------------

// OA: the real GridMover reports its step to the real OpportunityAttacker —
// moving one tile out of the enemy's melee range triggers the enemy's free
// attack through the shared attacker.
func TestGridMoverTriggersOpportunityAttack(t *testing.T) {
	// Arrange
	w := newOpportunityWorld(t)
	pool := combat.NewAPPool()
	pool.Reset(4)
	mover := combat.NewGridMover(w.positions, w.positions, &stubBlocker{blocked: make(map[combat.Tile]bool)}, pool, nil, w.opp)
	enemy := addOpponent(t, w, "Enemy", combat.Tile{X: 0, Z: 0})
	moving := addOpponent(t, w, "Mover", combat.Tile{X: 1, Z: 0})
	target := addOpponent(t, w, "Target", combat.Tile{X: 3, Z: 0})
	w.factions.enemies[moving] = []entity.GameEntityID{enemy}

	// Act — the first step of the path (1,0) → (2,0) leaves the enemy's range.
	if err := mover.MoveToward(moving, target); err != nil {
		t.Fatalf("MoveToward: %v", err)
	}

	// Assert
	if len(w.attacker.attacks) != 1 || w.attacker.attacks[0].attacker != enemy || w.attacker.attacks[0].target != moving {
		t.Fatalf("MoveToward attacks: want [enemy → mover], got %+v", w.attacker.attacks)
	}
	if got := pool.Current(); got != 3 {
		t.Fatalf("pool: want 3 AP after one step, got %d", got)
	}
}

// OA: the mover reports every successful step to the OpportunityWatcher with
// the correct tiles — the interface contract the opportunity logic builds on.
func TestGridMoverReportsStepToWatcher(t *testing.T) {
	// Arrange
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	watcher := &stubWatcher{}
	pool := combat.NewAPPool()
	pool.Reset(4)
	mover := combat.NewGridMover(positions, positions, &stubBlocker{blocked: make(map[combat.Tile]bool)}, pool, nil, watcher)
	moving, err := store.CreateEntity(entity.EntityTypeCharacter, "Mover", combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	if err != nil {
		t.Fatalf("CreateEntity(Mover): %v", err)
	}
	positions.positions[moving] = combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	target, err := store.CreateEntity(entity.EntityTypeCharacter, "Target", combat.TileToWorld(combat.Tile{X: 2, Z: 0}))
	if err != nil {
		t.Fatalf("CreateEntity(Target): %v", err)
	}
	positions.positions[target] = combat.TileToWorld(combat.Tile{X: 2, Z: 0})

	// Act
	if err := mover.MoveToward(moving, target); err != nil {
		t.Fatalf("MoveToward: %v", err)
	}

	// Assert
	want := []watchCall{{mover: moving, from: combat.Tile{X: 0, Z: 0}, to: combat.Tile{X: 1, Z: 0}}}
	if !reflect.DeepEqual(watcher.calls, want) {
		t.Fatalf("CheckStep calls = %+v, want %+v", watcher.calls, want)
	}
}

// ---------------------------------------------------------------------------
// Elevation movement cost (ADR-0009: climbing 1 level costs 2 AP)
// ---------------------------------------------------------------------------

// CLIMB: a step that ascends one elevation level costs ClimbCost (2) AP
// total — MoveCost + (ClimbCost − MoveCost) per level climbed.
func TestClimbingStepCostsTwoAP(t *testing.T) {
	// Arrange
	w := newMoverGridWorld(t, 4)
	w.grid.SetElevation(combat.Tile{X: 1, Z: 0}, 1)
	moving := addMoverEntity(t, w, "Mover", combat.Tile{X: 0, Z: 0})
	target := addMoverEntity(t, w, "Target", combat.Tile{X: 1, Z: 0})

	// Act
	if err := w.mover.MoveToward(moving, target); err != nil {
		t.Fatalf("MoveToward: %v", err)
	}

	// Assert
	if got := w.pool.Current(); got != 2 {
		t.Fatalf("pool: want 2 AP left after a 2-AP climb, got %d", got)
	}
	want := combat.TileToWorld(combat.Tile{X: 1, Z: 0})
	got := w.positions.positions[moving]
	if got.X() != want.X() || got.Z() != want.Z() {
		t.Fatalf("position: want %v, got %v", want, got)
	}
}

// CLIMB: level movement keeps the Story 002 cost — 1 AP per tile.
func TestLevelStepCostsOneAP(t *testing.T) {
	// Arrange
	w := newMoverGridWorld(t, 4)
	moving := addMoverEntity(t, w, "Mover", combat.Tile{X: 0, Z: 0})
	target := addMoverEntity(t, w, "Target", combat.Tile{X: 1, Z: 0})

	// Act
	if err := w.mover.MoveToward(moving, target); err != nil {
		t.Fatalf("MoveToward: %v", err)
	}

	// Assert
	if got := w.pool.Current(); got != 3 {
		t.Fatalf("pool: want 3 AP left after a 1-AP step, got %d", got)
	}
}

// CLIMB: descending costs the normal 1 AP — the climb cost applies to
// ascending only.
func TestDescendingStepCostsOneAP(t *testing.T) {
	// Arrange
	w := newMoverGridWorld(t, 4)
	w.grid.SetElevation(combat.Tile{X: 0, Z: 0}, 1)
	moving := addMoverEntity(t, w, "Mover", combat.Tile{X: 0, Z: 0})
	target := addMoverEntity(t, w, "Target", combat.Tile{X: 1, Z: 0})

	// Act
	if err := w.mover.MoveToward(moving, target); err != nil {
		t.Fatalf("MoveToward: %v", err)
	}

	// Assert
	if got := w.pool.Current(); got != 3 {
		t.Fatalf("pool: want 3 AP left after a descending step, got %d", got)
	}
}

// CLIMB: a climb the pool cannot cover fails with ErrInsufficientAP — no
// position change, no AP spent.
func TestClimbingStepInsufficientAP(t *testing.T) {
	// Arrange
	w := newMoverGridWorld(t, 1)
	w.grid.SetElevation(combat.Tile{X: 1, Z: 0}, 1)
	moving := addMoverEntity(t, w, "Mover", combat.Tile{X: 0, Z: 0})
	target := addMoverEntity(t, w, "Target", combat.Tile{X: 1, Z: 0})

	// Act
	err := w.mover.MoveToward(moving, target)

	// Assert
	if !errors.Is(err, combat.ErrInsufficientAP) {
		t.Fatalf("MoveToward: want ErrInsufficientAP, got %v", err)
	}
	if got := w.pool.Current(); got != 1 {
		t.Fatalf("pool: want untouched 1 AP, got %d", got)
	}
	want := combat.TileToWorld(combat.Tile{X: 0, Z: 0})
	got := w.positions.positions[moving]
	if got.X() != want.X() || got.Z() != want.Z() {
		t.Fatalf("position: want unchanged %v, got %v", want, got)
	}
}

// ---------------------------------------------------------------------------
// FactionReader from the combat manager (NewManagerFactions)
// ---------------------------------------------------------------------------

// FR: faction relationships derive from the turn order's sides — the camp
// model (players and allies are friendly, enemies hostile to both) with
// dead combatants excluded from both lists.
func TestManagerFactionsCampModel(t *testing.T) {
	// Arrange
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	positions := &stubPositions{positions: make(map[entity.GameEntityID]matrix.Vec3)}
	pc := addCombatant(t, store, positions, entity.EntityTypeCharacter, "PC", 10, 10, combat.TileToWorld(combat.Tile{X: 0, Z: 0}))
	ally := addCombatant(t, store, positions, entity.EntityTypeCharacter, "Ally", 10, 10, combat.TileToWorld(combat.Tile{X: 1, Z: 0}))
	enemy := addCombatant(t, store, positions, entity.EntityTypeNPC, "Enemy", 10, 10, combat.TileToWorld(combat.Tile{X: 2, Z: 0}))
	dead := addCombatant(t, store, positions, entity.EntityTypeNPC, "DeadEnemy", 10, 0, combat.TileToWorld(combat.Tile{X: 3, Z: 0}))
	factions := combat.NewManagerFactions(store, &stubSides{order: []combat.TurnEntry{
		{EntityID: pc, Side: combat.SidePlayer, Initiative: 3, Dexterity: 10},
		{EntityID: ally, Side: combat.SideAlly, Initiative: 2, Dexterity: 10},
		{EntityID: enemy, Side: combat.SideEnemy, Initiative: 1, Dexterity: 10},
		{EntityID: dead, Side: combat.SideEnemy, Initiative: 0, Dexterity: 10},
	}})

	// Assert
	if got := factions.AlliesOf(pc); !reflect.DeepEqual(got, []entity.GameEntityID{ally}) {
		t.Fatalf("AlliesOf(pc): want [ally], got %v", got)
	}
	if got := factions.EnemiesOf(pc); !reflect.DeepEqual(got, []entity.GameEntityID{enemy}) {
		t.Fatalf("EnemiesOf(pc): want [enemy] (dead excluded), got %v", got)
	}
	if got := factions.EnemiesOf(ally); !reflect.DeepEqual(got, []entity.GameEntityID{enemy}) {
		t.Fatalf("EnemiesOf(ally): want [enemy], got %v", got)
	}
	if got := factions.AlliesOf(enemy); len(got) != 0 {
		t.Fatalf("AlliesOf(enemy): want none (only dead enemies remain), got %v", got)
	}
	if got := factions.AlliesOf(999); len(got) != 0 {
		t.Fatalf("AlliesOf(unknown): want none (fail-closed), got %v", got)
	}
	if got := factions.EnemiesOf(999); len(got) != 0 {
		t.Fatalf("EnemiesOf(unknown): want none (fail-closed), got %v", got)
	}
}

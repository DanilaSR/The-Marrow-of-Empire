// Unit tests for Story 003: Attack Resolution (d20 System).
//
// Evidence path: production/epics/tactical-combat-engine/
// story-003-attack-resolution-d20.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria and QA test cases map to test functions 1:1 (Go
// only executes Test-prefixed functions, so each story name is exported with
// the Go prefix):
//
//	AC-1  test_attack_hit_when_roll_exceeds_evasion -> TestAttackHitWhenRollExceedsEvasion
//	AC-2  test_attack_miss_when_roll_below_evasion  -> TestAttackMissWhenRollBelowEvasion
//	AC-3  test_natural_20_crits                     -> TestNatural20CritsRegardlessOfEvasion
//	AC-4  test_natural_1_auto_misses                -> TestNatural1AutoMissesRegardlessOfAccuracy
//	AC-5  test_melee_base_damage                    -> TestMeleeBaseDamageFromStrength
//	AC-6  test_ranged_damage_after_armor            -> TestRangedDamageSubtractsArmor
//	(edge) test_min_damage_floor                    -> TestDamageFloorsAtOne
//	(edge) test_crit_damage_floor                   -> TestCriticalDamageFloorsAtOne
//	(edge) test_unarmed_defaults                    -> TestWeaponlessAttackerUsesUnarmedDefaults
//	(edge) test_missing_components_fail_closed      -> TestMissingStatsFailsClosed
//	(edge) test_roll_clamped                        -> TestRollClampedToD20Range
//	(TR)  test_deterministic_seed                   -> TestRandomD20RollerDeterministicSeed
//
// The Kaiju host is the same mock used by the unit/integration entity tests;
// combat is pure game logic, so the resolver's dependencies (d20 roll,
// weapon/armor data) are stubs injected through the combat package
// interfaces — the acceptance criteria fix the roll, and the seeded
// production Roller covers the "deterministic seed" requirement.
package combat_test

import (
	"testing"

	"kaijuengine.com/matrix"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// mockKaijuHost and newMockHost come from grid_movement_test.go (same
// package): the entity test store needs the Kaiju host double to create
// entities.

// fixedRoller is a stub for combat.Roller returning one preset roll. The
// acceptance criteria fix the d20 value, so determinism comes from the stub;
// the production RandomD20Roller's seeded determinism is covered by
// TestRandomD20RollerDeterministicSeed.
type fixedRoller struct {
	roll int
}

func (r *fixedRoller) RollD20() int { return r.roll }

// stubEquipment implements combat.EquipmentReader over preset weapon and
// armor values keyed by entity ID. An entity without a weapon entry is
// unarmed (the resolver falls back to the unarmed defaults); armor defaults
// to 0.
type stubEquipment struct {
	weapons map[entity.GameEntityID]combat.WeaponStats
	armor   map[entity.GameEntityID]int
}

func (e *stubEquipment) Weapon(id entity.GameEntityID) (combat.WeaponStats, bool) {
	w, ok := e.weapons[id]
	return w, ok
}

func (e *stubEquipment) Armor(id entity.GameEntityID) int { return e.armor[id] }

// attackWorld wires the d20 attacker over the entity test store with stubbed
// equipment and a fixed roll.
type attackWorld struct {
	store     entity.EntityStore
	equipment *stubEquipment
	resolve   combat.Attacker
}

// newAttackWorld creates the resolver with the given fixed roll. Tests
// populate w.equipment and add combatants to w.store.
func newAttackWorld(t *testing.T, roll int) *attackWorld {
	t.Helper()
	bus := entity.NewEventBus()
	store := entity.NewEntityStore(newMockHost(), bus, entity.EntityStoreConfig{})
	equipment := &stubEquipment{
		weapons: make(map[entity.GameEntityID]combat.WeaponStats),
		armor:   make(map[entity.GameEntityID]int),
	}
	// Nil positioning: Story 003 tests exercise the resolver without
	// positioning bonuses — nil disables them exactly.
	resolve := combat.NewD20Attacker(store, equipment, &fixedRoller{roll: roll}, nil)
	return &attackWorld{store: store, equipment: equipment, resolve: resolve}
}

// addStatCombatant creates a character entity carrying the given Stats
// component and returns its GameEntityID. (Named distinctly from
// grid_movement_test.go's addCombatant, which also wires Health and a
// position.)
func addStatCombatant(t *testing.T, store entity.EntityStore, name string, stats entity.Stats) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(entity.EntityTypeCharacter, name, matrix.Vec3{})
	if err != nil {
		t.Fatalf("CreateEntity(%s): %v", name, err)
	}
	if err := entity.SetComponent(store, id, stats); err != nil {
		t.Fatalf("SetComponent(Stats, %s): %v", name, err)
	}
	return id
}

// ---------------------------------------------------------------------------
// AC-1: acc=12 vs eva=14, roll=3 → HIT (12+3=15 ≥ 14)
// ---------------------------------------------------------------------------

// QA-1: Given accuracy 12 against evasion 14, When rolling 3, Then the attack
// hits (15 >= 14) without a crit. The resolver reads the target's derived
// evasion from the Stats component (TR-stats-003) and adds the attacker's
// STR_mod (0 for STR 10) to the weapon accuracy.
func TestAttackHitWhenRollExceedsEvasion(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 3)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 14})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 12, BaseDamage: 4}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit (12+3=15 >= 14), got miss")
	}
	if got.Crit {
		t.Fatalf("ResolveAttack: want no crit on roll 3, got crit")
	}
	if got.Roll != 3 {
		t.Fatalf("ResolveAttack roll: want 3, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// AC-2: acc=10 vs eva=20, roll=8 → MISS (10+8=18 < 20)
// ---------------------------------------------------------------------------

// QA-2: Given accuracy 10 against evasion 20, When rolling 8, Then the attack
// misses (18 < 20) with no damage.
func TestAttackMissWhenRollBelowEvasion(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 8)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Ghost", entity.Stats{Evasion: 20})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 4}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if got.Hit {
		t.Fatalf("ResolveAttack: want miss (10+8=18 < 20), got hit")
	}
	if got.Damage != 0 {
		t.Fatalf("ResolveAttack damage: want 0 on miss, got %d", got.Damage)
	}
	if got.Roll != 8 {
		t.Fatalf("ResolveAttack roll: want 8, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// AC-3: roll=20 → crit, damage = weapon_base × 2 + mod (regardless of
// evasion)
// ---------------------------------------------------------------------------

// QA-3: Given a natural 20 against evasion 30 (beyond any reachable derived
// value), When the attack resolves, Then it is a critical hit regardless of
// evasion, and damage = weapon 4 x2 + STR 16 mod 3 = 11 (ADR-0009: crit
// doubles the weapon base only).
func TestNatural20CritsRegardlessOfEvasion(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 20)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 16})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 30})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 0, BaseDamage: 4}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit on natural 20, got miss")
	}
	if !got.Crit {
		t.Fatalf("ResolveAttack: want crit on natural 20, got no crit")
	}
	if want := 4*2 + 3; got.Damage != want {
		t.Fatalf("ResolveAttack damage: want %d (weapon 4 x2 + STR 16 mod 3), got %d", want, got.Damage)
	}
	if got.Roll != 20 {
		t.Fatalf("ResolveAttack roll: want 20, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// AC-4: roll=1 → auto-miss regardless of accuracy
// ---------------------------------------------------------------------------

// QA-4: Given a natural 1, When the attack resolves against evasion 6 with
// accuracy 20, Then it misses automatically with no damage (ADR-0009:
// "Natural 1: automatic miss").
func TestNatural1AutoMissesRegardlessOfAccuracy(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 1)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 18})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 20, BaseDamage: 10}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if got.Hit {
		t.Fatalf("ResolveAttack: want auto-miss on natural 1, got hit")
	}
	if got.Crit {
		t.Fatalf("ResolveAttack: want no crit on natural 1, got crit")
	}
	if got.Damage != 0 {
		t.Fatalf("ResolveAttack damage: want 0 on miss, got %d", got.Damage)
	}
	if got.Roll != 1 {
		t.Fatalf("ResolveAttack roll: want 1, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// AC-5: melee with STR 16 (+3) and weapon 4 → base damage = 7
// ---------------------------------------------------------------------------

// QA-5: Given a melee attack with STR 16 (mod +3) and weapon base 4, When it
// hits, Then damage = 4 + 3 = 7 (armor 0; GDD damage formula).
func TestMeleeBaseDamageFromStrength(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 10)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 16})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 4}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit, got miss")
	}
	if want := 4 + 3; got.Damage != want {
		t.Fatalf("ResolveAttack damage: want %d (weapon 4 + STR 16 mod 3), got %d", want, got.Damage)
	}
}

// ---------------------------------------------------------------------------
// AC-6: ranged with DEX 14 (+2), weapon 5, armor 2 → final = 5
// ---------------------------------------------------------------------------

// QA-6: Given a ranged attack with DEX 14 (mod +2), weapon base 5, and target
// armor 2, When it hits, Then damage = 5 + 2 - 2 = 5. The ranged weapon
// selects DEX_mod over STR_mod (ADR-0009), and armor reduces the total.
func TestRangedDamageSubtractsArmor(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 10)
	attacker := addStatCombatant(t, w.store, "Archer", entity.Stats{Dexterity: 14})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 5, Ranged: true}
	w.equipment.armor[target] = 2

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit, got miss")
	}
	if want := 5 + 2 - 2; got.Damage != want {
		t.Fatalf("ResolveAttack damage: want %d (weapon 5 + DEX 14 mod 2 - armor 2), got %d", want, got.Damage)
	}
}

// ---------------------------------------------------------------------------
// Edges
// ---------------------------------------------------------------------------

// Edge: armor exceeding the damage contribution floors the result at 1
// (GDD: damage = max(1, weapon_base + attribute_mod - target_armor)).
func TestDamageFloorsAtOne(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 10)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Knight", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 2}
	w.equipment.armor[target] = 5

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want hit, got miss")
	}
	if got.Damage != 1 {
		t.Fatalf("ResolveAttack damage: want 1 (floor: 2 + 0 - 5), got %d", got.Damage)
	}
}

// Edge: the min-1 floor also applies to crit damage (weapon 1 x2 + 0 - 5).
func TestCriticalDamageFloorsAtOne(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 20)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Knight", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 1}
	w.equipment.armor[target] = 5

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Crit {
		t.Fatalf("ResolveAttack: want crit, got no crit")
	}
	if got.Damage != 1 {
		t.Fatalf("ResolveAttack damage: want 1 (floor: 1 x2 + 0 - 5), got %d", got.Damage)
	}
}

// Edge: an attacker with no weapon falls back to the unarmed defaults
// (UnarmedAccuracy 10, UnarmedDamage 1, melee) and still resolves normally.
func TestWeaponlessAttackerUsesUnarmedDefaults(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 10)
	attacker := addStatCombatant(t, w.store, "Brawler", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Hit {
		t.Fatalf("ResolveAttack: want unarmed hit (acc 10 + roll 10 >= eva 6), got miss")
	}
	if got.Damage != combat.UnarmedDamage {
		t.Fatalf("ResolveAttack damage: want %d (unarmed base), got %d", combat.UnarmedDamage, got.Damage)
	}
}

// Edge: fail-closed — an attacker without the Stats component cannot attack,
// and a target without Stats is never hit (StartCombat validates Stats for
// every combatant, so this guards wiring bugs only).
func TestMissingStatsFailsClosed(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 20)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 16})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 4}

	ghost, err := w.store.CreateEntity(entity.EntityTypeCharacter, "Ghost", matrix.Vec3{})
	if err != nil {
		t.Fatalf("CreateEntity(Ghost): %v", err)
	}
	phantom, err := w.store.CreateEntity(entity.EntityTypeCharacter, "Phantom", matrix.Vec3{})
	if err != nil {
		t.Fatalf("CreateEntity(Phantom): %v", err)
	}

	// Act & Assert: attacker without Stats → miss.
	got := w.resolve.ResolveAttack(ghost, target)
	if got.Hit {
		t.Fatalf("ResolveAttack: want miss for attacker without Stats, got hit")
	}

	// Act & Assert: target without Stats → miss.
	got = w.resolve.ResolveAttack(attacker, phantom)
	if got.Hit {
		t.Fatalf("ResolveAttack: want miss for target without Stats, got hit")
	}
}

// Edge: a Roller returning an out-of-range value is clamped to the d20 range
// (defensive): 99 resolves as a natural 20 crit, and the result reports the
// clamped roll.
func TestRollClampedToD20Range(t *testing.T) {
	// Arrange
	w := newAttackWorld(t, 99)
	attacker := addStatCombatant(t, w.store, "Fighter", entity.Stats{Strength: 10})
	target := addStatCombatant(t, w.store, "Goblin", entity.Stats{Evasion: 6})
	w.equipment.weapons[attacker] = combat.WeaponStats{Accuracy: 10, BaseDamage: 4}

	// Act
	got := w.resolve.ResolveAttack(attacker, target)

	// Assert
	if !got.Crit {
		t.Fatalf("ResolveAttack: want crit for roll 99 clamped to 20, got %+v", got)
	}
	if got.Roll != 20 {
		t.Fatalf("ResolveAttack roll: want clamped 20, got %d", got.Roll)
	}
}

// ---------------------------------------------------------------------------
// TR: deterministic seed
// ---------------------------------------------------------------------------

// TR: two RandomD20Rollers seeded identically produce identical sequences,
// and every roll stays in [1, 20] (Story 003: "Deterministic seed").
func TestRandomD20RollerDeterministicSeed(t *testing.T) {
	// Arrange
	a := combat.NewRandomD20Roller(42)
	b := combat.NewRandomD20Roller(42)

	// Act & Assert
	for i := 0; i < 20; i++ {
		ra, rb := a.RollD20(), b.RollD20()
		if ra != rb {
			t.Fatalf("roll %d: seeded rollers diverged (%d vs %d)", i, ra, rb)
		}
		if ra < 1 || ra > 20 {
			t.Fatalf("roll %d: %d outside d20 range [1, 20]", i, ra)
		}
	}
}

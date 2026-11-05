package combat

import (
	"math/rand"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// Roller supplies the d20 roll for attack resolution (Story 003,
// TR-combat-003, ADR-0009). The game wiring provides a RandomD20Roller
// backed by the game's random source; tests inject a fixed-roll stub so the
// acceptance criteria are fully deterministic. The resolver clamps the roll
// to [1, 20] defensively, so a misbehaving Roller cannot produce a result
// outside the d20 range.
type Roller interface {
	// RollD20 returns the d20 roll (1-20). Return values outside [1, 20]
	// are clamped by the resolver.
	RollD20() int
}

// RandomD20Roller is the production Roller: a math/rand source over a
// caller-provided seed. The game wiring seeds it from system entropy at
// boot; a fixed seed reproduces an identical roll sequence, which keeps
// replays and tests deterministic (Story 003: "Deterministic seed").
type RandomD20Roller struct {
	rng *rand.Rand
}

// NewRandomD20Roller creates a Roller that produces a reproducible sequence
// for the given seed: two rollers with the same seed roll identically.
func NewRandomD20Roller(seed int64) Roller {
	return &RandomD20Roller{rng: rand.New(rand.NewSource(seed))}
}

// RollD20 implements Roller: a uniform value in [1, 20].
func (r *RandomD20Roller) RollD20() int {
	return r.rng.Intn(20) + 1
}

// WeaponStats is the attacker's weapon data used in attack resolution
// (ADR-0009): the base accuracy added to the attacker's attribute modifier
// for the hit roll, the base damage added to the modifier for the damage
// roll, and the weapon kind that selects which attribute applies — STR for
// melee, DEX for ranged.
type WeaponStats struct {
	// Accuracy is the weapon's base accuracy (GDD: "Base weapon accuracy +
	// attacker modifiers").
	Accuracy int
	// BaseDamage is the weapon's base damage (GDD range 2-20; e.g. dagger
	// 2-4, greatsword 8-20).
	BaseDamage int
	// Ranged selects the attribute modifier: true → DEX_mod, false →
	// STR_mod (ADR-0009: "STR_mod (melee) or DEX_mod (ranged)").
	Ranged bool
}

// EquipmentReader supplies the combat-relevant equipment of combatants: the
// attacker's weapon and the target's armor. The game wiring implements it
// from the Inventory component and item entities once the Loot & Economy
// system lands (the item data model is a known gap — cross-review F10);
// Story 003 tests inject exact values. This mirrors the Positioner pattern:
// the interface lives in combat, the wiring owns the data source.
type EquipmentReader interface {
	// Weapon returns the attacker's equipped weapon. Returns (zero, false)
	// when the attacker has no weapon — the resolver falls back to the
	// unarmed defaults (UnarmedAccuracy, UnarmedDamage, melee).
	Weapon(id entity.GameEntityID) (WeaponStats, bool)
	// Armor returns the target's damage reduction from armor (GDD range
	// 0-15). 0 means no armor.
	Armor(id entity.GameEntityID) int
}

// Unarmed tuning values (Story 003): the default weapon of an attacker with
// no weapon equipped. The GDD does not define unarmed combat; these defaults
// keep weaponless attackers (and enemies whose stat-block data arrives via
// the EquipmentReader wiring) able to act. A future balance story may move
// them into combat.Config (data-driven pattern: config.go).
const (
	// UnarmedAccuracy is the base accuracy of a weaponless attack.
	UnarmedAccuracy = 10
	// UnarmedDamage is the base damage of a weaponless attack (melee, so
	// STR_mod applies).
	UnarmedDamage = 1
)

// d20Attacker implements Attacker with the Story 003 d20 resolution
// (TR-combat-003, ADR-0009):
//
//	Hit roll: d20 + accuracy >= target evasion → hit
//	Natural 20: critical hit — damage doubles the weapon base
//	Natural 1: automatic miss, regardless of accuracy
//	Damage: max(1, weapon_base + attribute_mod − target_armor)
//
// Accuracy = weapon accuracy + STR_mod (melee) or DEX_mod (ranged) +
// positioning bonuses (Story 004: flanking, elevation).
// Evasion is read from the target's Stats component derived field
// (TR-stats-003: 10 + DEX_mod); Story 004 cover adds on top of it. Ability
// bonuses (abilities) are out of scope and add later.
type d20Attacker struct {
	store     entity.EntityStore
	equipment EquipmentReader
	roller    Roller
	// positioning supplies the Story 004 positional bonuses (flanking,
	// cover, elevation) and the line-of-sight gate. Nil disables both —
	// the resolver behaves exactly as in Story 003.
	positioning PositioningResolver
}

// Compile-time assertion: d20Attacker satisfies Attacker.
var _ Attacker = (*d20Attacker)(nil)

// NewD20Attacker creates the Story 003 d20 attack resolver. store provides
// the combatants' Stats components (attributes, evasion); equipment provides
// the attacker's weapon and the target's armor; roller provides the d20
// roll; positioning supplies the Story 004 positional bonuses — flanking,
// cover, elevation, and the line-of-sight gate. A nil positioning disables
// positioning (Story 003 behavior). A nil equipment or roller panics — the
// game wiring always supplies both (the tests inject stubs).
//
// Example usage:
//
//	positioning := combat.NewGridPositioning(positioner, grid, combat.NewManagerFactions(store, manager))
//	attacker := combat.NewD20Attacker(store, equipment, combat.NewRandomD20Roller(seed), positioning)
//	ai := combat.NewBasicEnemyAI(positioner, attacker, mover, combat.DefaultAttackRange)
func NewD20Attacker(store entity.EntityStore, equipment EquipmentReader, roller Roller, positioning PositioningResolver) Attacker {
	return &d20Attacker{store: store, equipment: equipment, roller: roller, positioning: positioning}
}

// ResolveAttack implements Attacker: rolls the d20 (clamped to [1, 20]) and
// resolves the attack per TR-combat-003. The result always reports the
// clamped Roll. Fail-closed defensive behavior: an attacker or target
// missing the Stats component resolves as a miss (StartCombat validates
// Stats for every combatant, so this path only guards wiring bugs); a
// blocked line of sight (full cover between the combatants, Story 004)
// resolves as a miss even on a natural 20 — full cover blocks the attack
// entirely (ADR-0009).
func (a *d20Attacker) ResolveAttack(attacker, target entity.GameEntityID) AttackResult {
	roll := clampRoll(a.roller.RollD20())

	attackerStats, ok := entity.GetComponent[entity.Stats](a.store, attacker)
	if !ok {
		return AttackResult{Roll: roll} // no stats — cannot attack (fail-closed)
	}
	targetStats, ok := entity.GetComponent[entity.Stats](a.store, target)
	if !ok {
		return AttackResult{Roll: roll} // no stats — nothing to hit (fail-closed)
	}
	var mods PositioningModifiers
	if a.positioning != nil {
		if !a.positioning.HasLineOfSight(attacker, target) {
			return AttackResult{Roll: roll} // full cover blocks the attack (ADR-0009)
		}
		mods = a.positioning.Modifiers(attacker, target)
	}
	if roll == 1 {
		return AttackResult{Roll: roll} // natural 1: auto-miss (ADR-0009)
	}

	weapon, ok := a.equipment.Weapon(attacker)
	if !ok {
		weapon = WeaponStats{Accuracy: UnarmedAccuracy, BaseDamage: UnarmedDamage}
	}
	attrMod := stats.Modifier(attackerStats.Strength)
	if weapon.Ranged {
		attrMod = stats.Modifier(attackerStats.Dexterity)
	}

	armor := a.equipment.Armor(target)
	if roll == 20 {
		// Natural 20: critical hit — double the weapon base (AC-3: damage =
		// weapon_base x2 + mod; ADR-0009: "double damage").
		return AttackResult{
			Hit:    true,
			Damage: max(1, weapon.BaseDamage*2+attrMod-armor),
			Crit:   true,
			Roll:   roll,
		}
	}

	accuracy := weapon.Accuracy + attrMod + mods.Accuracy
	if roll+accuracy < targetStats.Evasion+mods.Evasion {
		return AttackResult{Roll: roll} // d20 + accuracy < evasion → miss
	}
	return AttackResult{
		Hit:    true,
		Damage: max(1, weapon.BaseDamage+attrMod-armor),
		Roll:   roll,
	}
}

// clampRoll bounds a roller's output to the d20 range [1, 20] (defensive:
// the resolver never resolves a roll outside the die).
func clampRoll(roll int) int {
	if roll < 1 {
		return 1
	}
	if roll > 20 {
		return 20
	}
	return roll
}

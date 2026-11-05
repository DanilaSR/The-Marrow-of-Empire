// Unit tests for Story 005: Death, Dying & Stabilization.
//
// Evidence path: production/epics/tactical-combat-engine/
// story-005-death-dying-stabilization.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria and QA test cases map to test functions 1:1 (Go
// only executes Test-prefixed functions, so each story name is exported with
// the Go prefix):
//
//	AC-1  test_party_member_fatal_hit             -> TestCompanionFatalHitEntersDyingWithThreeRounds
//	AC-2  test_stabilize_2_ap_adjacent            -> TestStabilizeSpendsTwoAPAndPausesCountdown
//	AC-3  test_dying_expiry_permanent_death       -> TestDyingExpiresIntoPermanentDeath
//	AC-4  test_enemy_fatal_hit_instant_death      -> TestEnemyFatalHitIsInstantDeath
//	AC-5  test_dead_member_permadeath_at_end      -> TestDeadCompanionStaysDeadAtCombatEnd
//	(5)   test_knocked_out_vs_revive              -> TestKnockedOutCompanionRevivesAtCombatEnd
//	(6)   test_player_character_instant_game_over -> TestPlayerCharacterFatalHitResolvesDefeat
//	(5)   test_countdown_pauses_at_combat_end     -> TestDyingCountdownPausesAtCombatEnd
//	(5)   test_post_combat_free_stabilization     -> TestPostCombatStabilizationRevivesOneHP
//	(edge) test_stabilize_validations             -> TestStabilizeRequiresAdjacency / RequiresAlly /
//	                                                     RequiresActiveCombatant / RequiresSufficientAP /
//	                                                     RequiresDyingTarget / RequiresCapableHealer
//	(edge) test_damage_on_incapacitated_ignored   -> TestDamageToIncapacitatedTargetIgnored
//	(TR)  test_dying_rounds_configurable          -> TestDyingRoundsConfigurable
//	(int) test_damage_application_wiring          -> TestDamageApplyingAttackerAppliesHits /
//	                                                     TestEnemyAIAppliesDamageThroughDecorator
//
// The world wiring (mock host, stub positions/clock, manager with a no-op
// recording AI) comes from grid_movement_test.go (same package); the party
// configuration and the death system are injected per test. Combat is pure
// game logic, so every engine-bound dependency is a stub injected through the
// combat package interfaces.
package combat_test

import (
	"errors"
	"testing"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// stubParty implements combat.PartyReader with fixed values: which combatant
// is the player character and the companion_death_enabled knob (owned by the
// Party System in the game wiring, GDD CR-5).
type stubParty struct {
	playerCharacter entity.GameEntityID
	companionDeath  bool
}

func (p *stubParty) IsPlayerCharacter(id entity.GameEntityID) bool { return id == p.playerCharacter }
func (p *stubParty) CompanionDeathEnabled() bool                   { return p.companionDeath }

// fatalAttacker is a stub for combat.Attacker that always hits for a fixed
// damage amount — used to drive the death system through the damage-application
// decorator and the enemy AI.
type fatalAttacker struct {
	damage int
}

func (a *fatalAttacker) ResolveAttack(_, _ entity.GameEntityID) combat.AttackResult {
	return combat.AttackResult{Hit: true, Damage: a.damage}
}

// missAttacker is a stub for combat.Attacker that always misses.
type missAttacker struct{}

func (a *missAttacker) ResolveAttack(_, _ entity.GameEntityID) combat.AttackResult {
	return combat.AttackResult{Hit: false}
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// deathWorld wires a combat manager and a death system over the shared bus,
// store, positions, and AP pool, with an event recorder for the Story 005
// events plus CombatEnded and CombatantRemoved.
type deathWorld struct {
	bus       entity.EventBus
	store     entity.EntityStore
	clock     *mockClock
	positions *stubPositions
	pool      *combat.APPool
	manager   *combat.CombatManager
	party     *stubParty
	death     *combat.DeathSystem
	events    []entity.Event
}

// newDeathWorld creates the world with the given companion-death mode and
// death-system config. The manager's AI is the no-op recording pair from
// grid_movement_test.go, so enemy turns never deal damage — damage only
// arrives through explicit ApplyAttack calls (or the decorator tests).
func newDeathWorld(t *testing.T, companionDeath bool, config combat.Config) *deathWorld {
	t.Helper()
	mw := newManagerWorld(t, func(deps managerDeps) combat.EnemyAI {
		return combat.NewBasicEnemyAI(deps.positions, &recordingAttacker{}, &recordingMover{}, combat.DefaultAttackRange)
	})
	party := &stubParty{companionDeath: companionDeath}
	d := &deathWorld{
		bus:       mw.bus,
		store:     mw.store,
		clock:     mw.clock,
		positions: mw.positions,
		pool:      mw.pool,
		manager:   mw.manager,
		party:     party,
	}
	d.death = combat.NewDeathSystem(mw.bus, mw.store, mw.manager, mw.positions, party, mw.pool, config)
	for _, eventType := range []entity.EventType{
		combat.EventCharacterDying,
		combat.EventCharacterKnockedOut,
		combat.EventCharacterStabilized,
		combat.EventCharacterDied,
		combat.EventEnemyDefeated,
		combat.EventCombatEnded,
		combat.EventCombatantRemoved,
	} {
		mw.bus.Subscribe(eventType, 5, func(ev entity.Event) {
			d.events = append(d.events, ev)
		})
	}
	return d
}

// addPC creates the player character (DEX 16 — initiative 3, first in the
// order used by these tests) and registers it with the party stub.
func (d *deathWorld) addPC(t *testing.T, name string, hp int, tile combat.Tile) entity.GameEntityID {
	t.Helper()
	id := addCombatant(t, d.store, d.positions, entity.EntityTypeCharacter, name, 16, hp, combat.TileToWorld(tile))
	d.party.playerCharacter = id
	return id
}

// addCompanion creates a companion (DEX 14 — initiative 2, second in the
// order used by these tests).
func (d *deathWorld) addCompanion(t *testing.T, name string, hp int, tile combat.Tile) entity.GameEntityID {
	t.Helper()
	return addCombatant(t, d.store, d.positions, entity.EntityTypeCharacter, name, 14, hp, combat.TileToWorld(tile))
}

// addEnemy creates an enemy (DEX 12 — initiative 1, last in the order used
// by these tests; its turn processes automatically without dealing damage).
func (d *deathWorld) addEnemy(t *testing.T, name string, hp int, tile combat.Tile) entity.GameEntityID {
	t.Helper()
	return addCombatant(t, d.store, d.positions, entity.EntityTypeNPC, name, 12, hp, combat.TileToWorld(tile))
}

// start begins an encounter with the given entries (PC, companion, enemy in
// that order in these tests).
func (d *deathWorld) start(t *testing.T, entries []combat.CombatantEntry) {
	t.Helper()
	if err := d.manager.StartCombat(entries); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
}

// endPlayerCycle ends the active player-side turns in initiative order (the
// PC's turn, then the companion's); the enemy turn processes automatically
// between them. Ending the dying companion's turn ticks its countdown once.
func (d *deathWorld) endPlayerCycle(t *testing.T) {
	t.Helper()
	if err := d.manager.EndTurn(); err != nil {
		t.Fatalf("EndTurn (player): %v", err)
	}
	if err := d.manager.EndTurn(); err != nil {
		t.Fatalf("EndTurn (companion): %v", err)
	}
}

// hit applies a resolved hit of the given damage to the target through the
// death system (the same path the DamageDealer wiring uses).
func (d *deathWorld) hit(t *testing.T, target entity.GameEntityID, damage int) {
	t.Helper()
	d.death.ApplyAttack(target, combat.AttackResult{Hit: true, Damage: damage})
}

// eventsOf returns the recorded events of one type, in emission order.
func (d *deathWorld) eventsOf(eventType entity.EventType) []entity.Event {
	var out []entity.Event
	for _, ev := range d.events {
		if ev.Type == eventType {
			out = append(out, ev)
		}
	}
	return out
}

// inTurnOrder reports whether the combatant is still in the turn order.
func (d *deathWorld) inTurnOrder(id entity.GameEntityID) bool {
	for _, e := range d.manager.TurnOrder() {
		if e.EntityID == id {
			return true
		}
	}
	return false
}

// currentHP reads the combatant's Health.Current.
func currentHP(t *testing.T, store entity.EntityStore, id entity.GameEntityID) int {
	t.Helper()
	hp, ok := entity.GetComponent[entity.Health](store, id)
	if !ok {
		t.Fatalf("combatant %d has no Health component", id)
	}
	return hp.Current
}

// ---------------------------------------------------------------------------
// AC-1: party member at 5 HP taking 8 damage → HP=0, DYING, death_rounds=3
// ---------------------------------------------------------------------------

// QA-1: Given a companion at 5 HP with companion_death_enabled (Full Vision),
// When a resolved hit deals 8 damage, Then HP=0, status=DYING, death_rounds=3,
// and the CharacterDying event fires. The character stays in the turn order
// (its turns tick the countdown) and is not removed.
func TestCompanionFatalHitEntersDyingWithThreeRounds(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{}) // FV: default DyingRounds 3
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})

	// Act — 5 HP taking 8 damage (AC-1)
	w.hit(t, companion, 8)

	// Assert — HP=0, status=DYING, death_rounds=3
	if got := currentHP(t, w.store, companion); got != 0 {
		t.Fatalf("companion HP = %d, want 0", got)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want Dying", got)
	}
	dying := w.eventsOf(combat.EventCharacterDying)
	if len(dying) != 1 {
		t.Fatalf("CharacterDying events = %d, want 1", len(dying))
	}
	if data, ok := dying[0].Data.(combat.CharacterDyingData); !ok {
		t.Fatalf("CharacterDying payload type = %T, want CharacterDyingData", dying[0].Data)
	} else if data.EntityID != companion || data.RoundsRemaining != 3 {
		t.Fatalf("CharacterDyingData = %+v, want entity %d with 3 rounds", data, companion)
	}
	if !w.inTurnOrder(companion) {
		t.Fatalf("dying companion removed from turn order — it must stay to tick its countdown")
	}
	if got := len(w.eventsOf(combat.EventCharacterDied)); got != 0 {
		t.Fatalf("CharacterDied events = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// AC-2: dying member at round 2, adjacent ally spends 2 AP → STABLE
// ---------------------------------------------------------------------------

// QA-2: Given a dying companion at countdown 2 (one of its own turns ended),
// When the adjacent player character spends 2 AP to stabilize, Then status=
// STABLE, HP stays 0, the countdown pauses (later turn ends do not tick), and
// the character survives — reviving with 1 HP when combat ends.
func TestStabilizeSpendsTwoAPAndPausesCountdown(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8)          // dying, countdown 3
	w.endPlayerCycle(t)             // companion's first turn ended → countdown 2
	if got := w.manager.CurrentAP(); got != 7 {
		t.Fatalf("PC AP before stabilize = %d, want 7 (DEX 16)", got)
	}

	// Act — the active PC (adjacent) stabilizes: 2 AP
	err := w.death.Stabilize(pc, companion)

	// Assert — STABLE, HP stays 0, 2 AP spent
	if err != nil {
		t.Fatalf("Stabilize: %v", err)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthStable {
		t.Fatalf("StatusOf(companion) = %v, want Stable", got)
	}
	if got := currentHP(t, w.store, companion); got != 0 {
		t.Fatalf("companion HP after stabilize = %d, want 0 (countdown paused, HP stays 0)", got)
	}
	if got := w.manager.CurrentAP(); got != 5 {
		t.Fatalf("PC AP after stabilize = %d, want 5 (7 - StabilizeCost 2)", got)
	}
	stabilized := w.eventsOf(combat.EventCharacterStabilized)
	if len(stabilized) != 1 {
		t.Fatalf("CharacterStabilized events = %d, want 1", len(stabilized))
	}
	if data, ok := stabilized[0].Data.(combat.CharacterStabilizedData); !ok {
		t.Fatalf("CharacterStabilized payload type = %T, want CharacterStabilizedData", stabilized[0].Data)
	} else if data.EntityID != companion || data.HealerID != pc || data.RoundsRemaining != 2 {
		t.Fatalf("CharacterStabilizedData = %+v, want entity %d healed by %d with 2 rounds", data, companion, pc)
	}

	// The countdown pauses: two more full cycles (the stable companion's turns
	// end twice) must not kill it.
	w.endPlayerCycle(t)
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthStable {
		t.Fatalf("StatusOf(companion) after paused turns = %v, want Stable (no countdown ticks)", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDied)); got != 0 {
		t.Fatalf("CharacterDied events = %d, want 0 (stabilized character survives)", got)
	}

	// Combat ends (victory): the stabilized character revives with 1 HP.
	w.hit(t, enemy, 25)
	if got := w.death.StatusOf(companion); got != combat.HealthAlive {
		t.Fatalf("StatusOf(companion) after combat end = %v, want Alive (revived)", got)
	}
	if got := currentHP(t, w.store, companion); got != 1 {
		t.Fatalf("companion HP after combat end = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// AC-3: dying member at round 3 with no stabilization → permanent death
// ---------------------------------------------------------------------------

// QA-3: Given a dying companion (countdown 3) and no stabilization, When
// three of its own turns end, Then the character dies permanently:
// CharacterDied fires, the companion is removed from the turn order with
// RemovalDefeated, and its status is Dead.
func TestDyingExpiresIntoPermanentDeath(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8) // dying, countdown 3

	// Act — three full cycles: the companion's turns end three times (rounds
	// 1→2→3 of the countdown); no stabilization happens.
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf after round 1 = %v, want Dying (countdown 3→2)", got)
	}
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf after round 2 = %v, want Dying (countdown 2→1)", got)
	}
	w.endPlayerCycle(t) // the third turn end: countdown 1→0 → permanent death

	// Assert — DEAD, removed from the turn order, permadeath status
	if got := w.death.StatusOf(companion); got != combat.HealthDead {
		t.Fatalf("StatusOf(companion) = %v, want Dead (countdown elapsed)", got)
	}
	died := w.eventsOf(combat.EventCharacterDied)
	if len(died) != 1 {
		t.Fatalf("CharacterDied events = %d, want 1", len(died))
	}
	removed := w.eventsOf(combat.EventCombatantRemoved)
	if len(removed) != 1 {
		t.Fatalf("CombatantRemoved events = %d, want 1", len(removed))
	}
	if data, ok := removed[0].Data.(combat.CombatantRemovedData); !ok {
		t.Fatalf("CombatantRemoved payload type = %T, want CombatantRemovedData", removed[0].Data)
	} else if data.EntityID != companion || data.Reason != combat.RemovalDefeated {
		t.Fatalf("CombatantRemovedData = %+v, want entity %d removed as Defeated", data, companion)
	}
	if w.inTurnOrder(companion) {
		t.Fatalf("dead companion still in turn order")
	}
	if got := currentHP(t, w.store, companion); got != 0 {
		t.Fatalf("dead companion HP = %d, want 0", got)
	}
	// The encounter continues without it.
	if got := w.manager.IsCombatActive(); !got {
		t.Fatalf("combat ended with the PC and enemy alive — want active")
	}
}

// ---------------------------------------------------------------------------
// AC-4: enemy at 3 HP taking 5 damage → DEAD (no dying state)
// ---------------------------------------------------------------------------

// QA-4: Given an enemy at 3 HP, When a resolved hit deals 5 damage, Then the
// enemy is DEAD instantly: EnemyDefeated fires, the enemy is removed with
// RemovalDefeated, and no dying/knockout state is ever entered. Defeating the
// last enemy resolves the encounter with Victory.
func TestEnemyFatalHitIsInstantDeath(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 3, combat.Tile{X: 1, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: enemy, Side: combat.SideEnemy},
	})

	// Act — 3 HP taking 5 damage (AC-4)
	w.hit(t, enemy, 5)

	// Assert — DEAD, no dying state
	if got := w.death.StatusOf(enemy); got != combat.HealthDead {
		t.Fatalf("StatusOf(enemy) = %v, want Dead (instant, no dying state)", got)
	}
	defeated := w.eventsOf(combat.EventEnemyDefeated)
	if len(defeated) != 1 {
		t.Fatalf("EnemyDefeated events = %d, want 1", len(defeated))
	}
	if data, ok := defeated[0].Data.(combat.EnemyDefeatedData); !ok || data.EntityID != enemy {
		t.Fatalf("EnemyDefeated payload = %+v, want entity %d", defeated[0].Data, enemy)
	}
	if got := len(w.eventsOf(combat.EventCharacterDying)) + len(w.eventsOf(combat.EventCharacterKnockedOut)); got != 0 {
		t.Fatalf("enemy entered a party death state (dying/KO events = %d)", got)
	}
	if w.inTurnOrder(enemy) {
		t.Fatalf("dead enemy still in turn order")
	}
	if got := w.manager.IsCombatActive(); got {
		t.Fatalf("combat still active after the last enemy died — want Victory resolution")
	}
	ended := w.eventsOf(combat.EventCombatEnded)
	if len(ended) != 1 {
		t.Fatalf("CombatEnded events = %d, want 1", len(ended))
	}
	if data, ok := ended[0].Data.(combat.CombatEndedData); !ok || data.Outcome != combat.OutcomeVictory {
		t.Fatalf("CombatEnded payload = %+v, want Victory", ended[0].Data)
	}
}

// ---------------------------------------------------------------------------
// AC-5: dead party member, combat ends → permadeath applied
// ---------------------------------------------------------------------------

// QA-5: Given a companion that died during combat (Full Vision), When combat
// ends, Then permadeath is applied: the companion stays Dead at 0 HP — the
// combat-end revival rules do not touch it (unlike Knocked Out or Stable
// characters).
func TestDeadCompanionStaysDeadAtCombatEnd(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8) // dying, countdown 3
	w.endPlayerCycle(t)    // 3→2
	w.endPlayerCycle(t)    // 2→1
	w.endPlayerCycle(t)    // 1→0 → dead

	// Act — combat ends in victory
	w.hit(t, enemy, 25)

	// Assert — permadeath: the dead companion is not revived
	if got := w.death.StatusOf(companion); got != combat.HealthDead {
		t.Fatalf("StatusOf(companion) after combat end = %v, want Dead (permadeath)", got)
	}
	if got := currentHP(t, w.store, companion); got != 0 {
		t.Fatalf("dead companion HP after combat end = %d, want 0", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDied)); got != 1 {
		t.Fatalf("CharacterDied events = %d, want exactly 1", got)
	}
}

// ---------------------------------------------------------------------------
// VS mode (companion_death_enabled = false): Knocked Out, no countdown,
// auto-revive 1 HP at combat end
// ---------------------------------------------------------------------------

// QA: Given a companion at 0 HP with companion_death_enabled false (VS),
// When the companion reaches 0 HP, Then it enters Knocked Out (no countdown),
// and when combat ends it revives with 1 HP. No dying state is ever entered.
func TestKnockedOutCompanionRevivesAtCombatEnd(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, false, combat.Config{}) // VS
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})

	// Act — the companion reaches 0 HP in VS mode
	w.hit(t, companion, 8)

	// Assert — Knocked Out, no countdown, no dying state
	if got := w.death.StatusOf(companion); got != combat.HealthKnockedOut {
		t.Fatalf("StatusOf(companion) = %v, want KnockedOut", got)
	}
	ko := w.eventsOf(combat.EventCharacterKnockedOut)
	if len(ko) != 1 {
		t.Fatalf("CharacterKnockedOut events = %d, want 1", len(ko))
	}
	if data, ok := ko[0].Data.(combat.CharacterKnockedOutData); !ok || data.EntityID != companion {
		t.Fatalf("CharacterKnockedOut payload = %+v, want entity %d", ko[0].Data, companion)
	}
	if got := len(w.eventsOf(combat.EventCharacterDying)); got != 0 {
		t.Fatalf("CharacterDying events = %d, want 0 (VS has no dying state)", got)
	}

	// The countdown never runs: two full cycles (the KO'd companion's turns
	// end twice) change nothing.
	w.endPlayerCycle(t)
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthKnockedOut {
		t.Fatalf("StatusOf(companion) after turns = %v, want KnockedOut (no countdown)", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDied)); got != 0 {
		t.Fatalf("CharacterDied events = %d, want 0", got)
	}

	// Combat ends: the KO'd companion revives with 1 HP immediately.
	w.hit(t, enemy, 25)
	if got := w.death.StatusOf(companion); got != combat.HealthAlive {
		t.Fatalf("StatusOf(companion) after combat end = %v, want Alive (revived)", got)
	}
	if got := currentHP(t, w.store, companion); got != 1 {
		t.Fatalf("companion HP after combat end = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// Player character death: always game over (not Dying, not KO)
// ---------------------------------------------------------------------------

// QA: Given the player character reaches 0 HP (regardless of the companion
// death setting and of living companions), When the fatal hit lands, Then the
// encounter resolves with Defeat (game over): CharacterDied fires, the
// encounter ends, and the player character never entered Dying or KnockedOut.
func TestPlayerCharacterFatalHitResolvesDefeat(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 20, combat.Tile{X: 1, Z: 0}) // alive!
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})

	// Act — the player character reaches 0 HP
	w.hit(t, pc, 25)

	// Assert — instant game over: Defeat, never Dying/KO
	if got := w.death.StatusOf(pc); got != combat.HealthDead {
		t.Fatalf("StatusOf(pc) = %v, want Dead", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDying)) + len(w.eventsOf(combat.EventCharacterKnockedOut)); got != 0 {
		t.Fatalf("PC entered a dying/KO state — player character death is always game over")
	}
	died := w.eventsOf(combat.EventCharacterDied)
	if len(died) != 1 {
		t.Fatalf("CharacterDied events = %d, want 1", len(died))
	}
	if w.manager.IsCombatActive() {
		t.Fatalf("combat still active after PC death — want Defeat resolution")
	}
	ended := w.eventsOf(combat.EventCombatEnded)
	if len(ended) != 1 {
		t.Fatalf("CombatEnded events = %d, want 1", len(ended))
	}
	if data, ok := ended[0].Data.(combat.CombatEndedData); !ok || data.Outcome != combat.OutcomeDefeat {
		t.Fatalf("CombatEnded payload = %+v, want Defeat (game over)", ended[0].Data)
	}
}

// ---------------------------------------------------------------------------
// Post-combat (FV): the countdown pauses at combat end; free stabilization
// ---------------------------------------------------------------------------

// QA: Given a dying companion at 0 HP (Full Vision) and combat ends with
// countdown time remaining, Then the countdown pauses — the companion remains
// Dying at 0 HP — and a post-combat stabilization (adjacent, no AP cost)
// revives it with 1 HP.
func TestDyingCountdownPausesAtCombatEnd(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8) // dying, countdown 3
	w.endPlayerCycle(t)    // countdown 3→2

	// Act — combat ends (victory) while the companion is still dying
	w.hit(t, enemy, 25)

	// Assert — the countdown pauses: still Dying at 0 HP, not dead
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) after combat end = %v, want Dying (countdown paused)", got)
	}
	if got := currentHP(t, w.store, companion); got != 0 {
		t.Fatalf("companion HP after combat end = %d, want 0 (paused, not revived)", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDied)); got != 0 {
		t.Fatalf("CharacterDied events = %d, want 0 (countdown paused at combat end)", got)
	}
}

// QA: Given a dying companion whose countdown paused at combat end (Full
// Vision), When the player performs a post-combat stabilization (adjacency
// required, no AP cost), Then the companion revives with 1 HP and the paused
// countdown value is reported in the stabilization event.
func TestPostCombatStabilizationRevivesOneHP(t *testing.T) {
	// Arrange — carry the dying companion over the combat end
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8) // dying, countdown 3
	w.endPlayerCycle(t)    // countdown 3→2
	w.hit(t, enemy, 25)    // combat ends, countdown paused at 2
	if w.manager.IsCombatActive() {
		t.Fatalf("combat should have ended")
	}

	// Act — post-combat stabilization: free (no AP), adjacency required
	err := w.death.Stabilize(pc, companion)

	// Assert — revived with 1 HP
	if err != nil {
		t.Fatalf("post-combat Stabilize: %v", err)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthAlive {
		t.Fatalf("StatusOf(companion) after post-combat stabilize = %v, want Alive", got)
	}
	if got := currentHP(t, w.store, companion); got != 1 {
		t.Fatalf("companion HP after post-combat stabilize = %d, want 1", got)
	}
	stabilized := w.eventsOf(combat.EventCharacterStabilized)
	if len(stabilized) != 1 {
		t.Fatalf("CharacterStabilized events = %d, want 1", len(stabilized))
	}
	if data, ok := stabilized[0].Data.(combat.CharacterStabilizedData); !ok {
		t.Fatalf("CharacterStabilized payload type = %T, want CharacterStabilizedData", stabilized[0].Data)
	} else if data.RoundsRemaining != 2 {
		t.Fatalf("paused countdown = %d, want 2 (frozen at combat end)", data.RoundsRemaining)
	}
	// The free action spent no AP (the pool sits at 0 after combat finalizes).
	if got := w.pool.Current(); got != 0 {
		t.Fatalf("AP pool after post-combat stabilize = %d, want 0 (free action)", got)
	}
}

// ---------------------------------------------------------------------------
// Stabilize validations
// ---------------------------------------------------------------------------

// Edge: the stabilizer must stand adjacent to the dying character (Chebyshev
// distance 1 on the grid).
func TestStabilizeRequiresAdjacency(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 3, Z: 0}) // not adjacent
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 4, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8)

	// Act & Assert
	err := w.death.Stabilize(pc, companion)
	if !errors.Is(err, combat.ErrNotAdjacent) {
		t.Fatalf("Stabilize: want ErrNotAdjacent, got %v", err)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want unchanged Dying", got)
	}
	if got := w.pool.Current(); got != 7 {
		t.Fatalf("AP spent on a rejected stabilize = %d, want 7 (untouched)", got)
	}
}

// Edge: an enemy cannot stabilize a dying party member.
func TestStabilizeRequiresAlly(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8)

	// Act & Assert — the enemy is adjacent to the dying companion but hostile
	err := w.death.Stabilize(enemy, companion)
	if !errors.Is(err, combat.ErrNotAlly) {
		t.Fatalf("Stabilize: want ErrNotAlly, got %v", err)
	}
}

// Edge: during combat only the active combatant can spend AP, so a
// non-active ally cannot stabilize.
func TestStabilizeRequiresActiveCombatant(t *testing.T) {
	// Arrange — the PC's turn is active; the second companion acts as the
	// would-be stabilizer
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	healer := w.addCompanion(t, "Healer", 20, combat.Tile{X: 0, Z: 1})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: healer, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8)
	if current, ok := w.manager.CurrentCombatant(); !ok || current != pc {
		t.Fatalf("PC should be the active combatant, got (%d, %v)", current, ok)
	}

	// Act & Assert — healer is adjacent, an ally, and capable, but not active
	err := w.death.Stabilize(healer, companion)
	if !errors.Is(err, combat.ErrNotActiveCombatant) {
		t.Fatalf("Stabilize: want ErrNotActiveCombatant, got %v", err)
	}
	if got := w.pool.Current(); got != 7 {
		t.Fatalf("AP spent on a rejected stabilize = %d, want 7 (untouched)", got)
	}
}

// Edge: stabilization costs 2 AP; an active stabilizer whose pool cannot
// cover it fails with ErrInsufficientAP and the pool is left unchanged.
func TestStabilizeRequiresSufficientAP(t *testing.T) {
	// Arrange — DEX 16 PC (7 AP, first in initiative) spends 6 AP on moves,
	// leaving 1 — below the 2 AP stabilization cost.
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8)
	for i := 0; i < 6; i++ {
		if err := w.pool.Spend(combat.MoveCost); err != nil {
			t.Fatalf("Spend(MoveCost) %d: %v", i+1, err)
		}
	}
	if got := w.manager.CurrentAP(); got != 1 {
		t.Fatalf("PC AP = %d, want 1 (7 - 6 moves)", got)
	}

	// Act & Assert
	err := w.death.Stabilize(pc, companion)
	if !errors.Is(err, combat.ErrInsufficientAP) {
		t.Fatalf("Stabilize: want ErrInsufficientAP, got %v", err)
	}
	if got := w.manager.CurrentAP(); got != 1 {
		t.Fatalf("AP after rejected stabilize = %d, want 1 (pool unchanged)", got)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want unchanged Dying", got)
	}
}

// Edge: only a Dying character can be stabilized — healthy, knocked out, and
// dead targets are rejected with ErrNotDying.
func TestStabilizeRequiresDyingTarget(t *testing.T) {
	// Sub-case 1: a healthy companion is not dying.
	t.Run("healthy", func(t *testing.T) {
		w := newDeathWorld(t, true, combat.Config{})
		pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
		companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
		enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
		w.start(t, []combat.CombatantEntry{
			{EntityID: pc, Side: combat.SidePlayer},
			{EntityID: companion, Side: combat.SideAlly},
			{EntityID: enemy, Side: combat.SideEnemy},
		})
		err := w.death.Stabilize(pc, companion)
		if !errors.Is(err, combat.ErrNotDying) {
			t.Fatalf("Stabilize healthy: want ErrNotDying, got %v", err)
		}
	})

	// Sub-case 2: a Knocked Out companion (VS) is not dying — no stabilization
	// required (GDD CR-5).
	t.Run("knocked out", func(t *testing.T) {
		w := newDeathWorld(t, false, combat.Config{}) // VS
		pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
		companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
		enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
		w.start(t, []combat.CombatantEntry{
			{EntityID: pc, Side: combat.SidePlayer},
			{EntityID: companion, Side: combat.SideAlly},
			{EntityID: enemy, Side: combat.SideEnemy},
		})
		w.hit(t, companion, 8) // KnockedOut
		err := w.death.Stabilize(pc, companion)
		if !errors.Is(err, combat.ErrNotDying) {
			t.Fatalf("Stabilize KO'd: want ErrNotDying, got %v", err)
		}
	})

	// Sub-case 3: a dead companion is not dying.
	t.Run("dead", func(t *testing.T) {
		w := newDeathWorld(t, true, combat.Config{})
		pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
		companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
		enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
		w.start(t, []combat.CombatantEntry{
			{EntityID: pc, Side: combat.SidePlayer},
			{EntityID: companion, Side: combat.SideAlly},
			{EntityID: enemy, Side: combat.SideEnemy},
		})
		w.hit(t, companion, 8)
		w.endPlayerCycle(t)
		w.endPlayerCycle(t)
		w.endPlayerCycle(t) // countdown elapsed → dead
		err := w.death.Stabilize(pc, companion)
		if !errors.Is(err, combat.ErrNotDying) {
			t.Fatalf("Stabilize dead: want ErrNotDying, got %v", err)
		}
	})
}

// Edge: an incapacitated character (0 HP) cannot stabilize — a dying
// companion cannot stabilize itself or another dying companion.
func TestStabilizeRequiresCapableHealer(t *testing.T) {
	// Arrange — both companions are dying; each is a valid ally and adjacent
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companionA := w.addCompanion(t, "CompanionA", 5, combat.Tile{X: 1, Z: 0})
	companionB := w.addCompanion(t, "CompanionB", 5, combat.Tile{X: 0, Z: 1})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companionA, Side: combat.SideAlly},
		{EntityID: companionB, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companionA, 8) // dying
	w.hit(t, companionB, 8) // dying

	// Act & Assert — a dying character cannot act (self-stabilize included)
	err := w.death.Stabilize(companionA, companionB)
	if !errors.Is(err, combat.ErrHealerIncapacitated) {
		t.Fatalf("Stabilize by dying healer: want ErrHealerIncapacitated, got %v", err)
	}
	err = w.death.Stabilize(companionA, companionA)
	if !errors.Is(err, combat.ErrHealerIncapacitated) {
		t.Fatalf("Self-stabilize: want ErrHealerIncapacitated, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edges
// ---------------------------------------------------------------------------

// Edge: damage to an already-incapacitated target is ignored — a dying
// companion cannot be overkilled into a different state, and no duplicate
// events fire.
func TestDamageToIncapacitatedTargetIgnored(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	w.hit(t, companion, 8) // dying
	before := len(w.events)

	// Act — another hit lands on the dying companion
	w.hit(t, companion, 99)

	// Assert — state unchanged, no new events
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want unchanged Dying", got)
	}
	if got := len(w.events); got != before {
		t.Fatalf("events grew from %d to %d after damage to an incapacitated target", before, got)
	}
}

// TR: the dying countdown length is data-driven (GDD tuning knob
// `dying_rounds`, range 2-5): Config.DyingRounds = 2 shortens the window, and
// the CharacterDying event reports it.
func TestDyingRoundsConfigurable(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{DyingRounds: 2})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})

	// Act
	w.hit(t, companion, 8)

	// Assert — entered Dying with the configured countdown
	dying := w.eventsOf(combat.EventCharacterDying)
	if len(dying) != 1 {
		t.Fatalf("CharacterDying events = %d, want 1", len(dying))
	}
	if data, ok := dying[0].Data.(combat.CharacterDyingData); !ok || data.RoundsRemaining != 2 {
		t.Fatalf("CharacterDyingData = %+v, want 2 rounds", dying[0].Data)
	}

	// One turn end: still dying. Second turn end: dead.
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf after 1 turn end = %v, want Dying (2→1)", got)
	}
	w.endPlayerCycle(t)
	if got := w.death.StatusOf(companion); got != combat.HealthDead {
		t.Fatalf("StatusOf after 2 turn ends = %v, want Dead", got)
	}
}

// ---------------------------------------------------------------------------
// Damage application wiring (NewDamageApplyingAttacker)
// ---------------------------------------------------------------------------

// Edge: the wiring decorator applies every resolved hit through the death
// system and passes misses through untouched — the enemy AI and opportunity
// attacks land damage through it without signature changes.
func TestDamageApplyingAttackerAppliesHits(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	applying := combat.NewDamageApplyingAttacker(&fatalAttacker{damage: 8}, w.death)

	// Act — a hit through the decorator
	got := applying.ResolveAttack(enemy, companion)

	// Assert — the result passes through and the damage lands: 5 HP - 8 → 0,
	// companion dying
	if !got.Hit || got.Damage != 8 {
		t.Fatalf("ResolveAttack = %+v, want passthrough hit for 8", got)
	}
	if gotHP := currentHP(t, w.store, companion); gotHP != 0 {
		t.Fatalf("companion HP = %d, want 0 (damage applied through the decorator)", gotHP)
	}
	if gotState := w.death.StatusOf(companion); gotState != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want Dying", gotState)
	}

	// Act — a miss through the decorator is a no-op (fresh healthy target)
	healthier := w.addCompanion(t, "Healthier", 5, combat.Tile{X: 0, Z: 2})
	miss := combat.NewDamageApplyingAttacker(&missAttacker{}, w.death)
	miss.ResolveAttack(enemy, healthier)

	// Assert — the target is untouched
	if gotHP := currentHP(t, w.store, healthier); gotHP != 5 {
		t.Fatalf("healthier companion HP after miss = %d, want 5 (miss deals nothing)", gotHP)
	}
	if gotState := w.death.StatusOf(healthier); gotState != combat.HealthAlive {
		t.Fatalf("StatusOf(healthier) = %v, want Alive", gotState)
	}
}

// Integration edge: the enemy AI resolves attacks through the decorated
// attacker, so a real enemy turn kills the companion — the full loop the game
// wiring composes (NewDamageApplyingAttacker around the d20 attacker).
func TestEnemyAIAppliesDamageThroughDecorator(t *testing.T) {
	// Arrange
	w := newDeathWorld(t, true, combat.Config{})
	pc := w.addPC(t, "PC", 20, combat.Tile{X: 0, Z: 0})
	companion := w.addCompanion(t, "Companion", 5, combat.Tile{X: 1, Z: 0})
	enemy := w.addEnemy(t, "Enemy", 20, combat.Tile{X: 2, Z: 0})
	w.start(t, []combat.CombatantEntry{
		{EntityID: pc, Side: combat.SidePlayer},
		{EntityID: companion, Side: combat.SideAlly},
		{EntityID: enemy, Side: combat.SideEnemy},
	})
	applying := combat.NewDamageApplyingAttacker(&fatalAttacker{damage: 8}, w.death)
	ai := combat.NewBasicEnemyAI(w.positions, applying, &recordingMover{}, combat.DefaultAttackRange)

	// Act — the enemy's turn: it attacks the nearest valid target (both PC and
	// companion are in range; the companion at (1,0) is closest to (2,0))
	ai.TakeTurn(enemy, []entity.GameEntityID{pc, companion})

	// Assert — the attack landed through the decorator: companion dying
	if gotHP := currentHP(t, w.store, companion); gotHP != 0 {
		t.Fatalf("companion HP after enemy turn = %d, want 0", gotHP)
	}
	if got := w.death.StatusOf(companion); got != combat.HealthDying {
		t.Fatalf("StatusOf(companion) = %v, want Dying (enemy attack applied)", got)
	}
	if got := len(w.eventsOf(combat.EventCharacterDying)); got != 1 {
		t.Fatalf("CharacterDying events = %d, want 1", got)
	}
}
// Compile-time assertion: the death system satisfies the DamageDealer the
// decorator consumes (guards the wiring contract).
var _ combat.DamageDealer = (*combat.DeathSystem)(nil)

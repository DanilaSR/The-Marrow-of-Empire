package combat

import (
	"errors"

	"marrow-of-empire/entity"
)

// Story 005 (Death, Dying & Stabilization, TR-combat-004, ADR-0009) owns the
// 0 HP transition (GDD CR-5):
//
//	Alive  --fatal hit (party, FV)-->       Dying (countdown DyingRounds)
//	Dying  --own turn ends-->               countdown - 1; 0 → Dead
//	Dying  --adjacent ally, StabilizeCost--> Stable (countdown paused)
//	Alive  --fatal hit (companion, VS)-->   KnockedOut (no countdown)
//	Alive  --fatal hit (player character)--> Dead (instant game over, Defeat)
//	Alive  --fatal hit (enemy)-->           Dead (instant, EnemyDefeated)
//	KnockedOut | Stable --combat end-->     revived with 1 HP
//	Dying  --combat end (FV)-->             countdown paused; free post-combat
//	                                        stabilization revives with 1 HP
//	Dead   --combat end-->                  stays dead (permadeath, AC-5)
//
// The player character never enters Dying or KnockedOut: reaching 0 HP is an
// immediate Defeat (game over). Enemies never enter Dying or KnockedOut:
// 0 HP is instant death. The companion route is selected by the injected
// PartyReader's CompanionDeathEnabled knob (owned by the Party System, GDD
// CR-5: false = Knocked Out at VS, true = Dying/Dead at Full Vision).
//
// The death system is a separate, injectable system — it consumes the
// CombatManager (removal, resolution, active-turn checks) rather than being
// part of it, matching the OpportunityAttacker pattern. Combat-scoped state
// (dying countdowns, sides snapshot) lives inside the system, never in a
// component: save is manual-only during combat and a loaded save must never
// restore mid-combat death state.

// HealthState is a combatant's death/dying status (GDD CR-5 state table).
type HealthState int

const (
	// HealthAlive: HP > 0, normal combat participation. Also returned for
	// entities the death system does not track (healthy combatants and
	// unknown entities).
	HealthAlive HealthState = iota
	// HealthDying: HP = 0, party only (Full Vision, companion_death_enabled
	// true). Cannot act; must be stabilized by an adjacent ally within
	// DyingRounds of the character's own turn ends, or dies permanently.
	HealthDying
	// HealthKnockedOut: HP = 0, companion only (VS, companion_death_enabled
	// false). Cannot act; no countdown; revives with 1 HP when combat ends.
	HealthKnockedOut
	// HealthStable: HP = 0, party only (Full Vision). The dying countdown was
	// paused by a successful stabilization; the character survives and revives
	// with 1 HP when combat ends.
	HealthStable
	// HealthDead: permanently dead — a party member whose countdown elapsed
	// (or the player character at 0 HP, instant game over) or a defeated
	// enemy. Save/Load is the only recovery mechanism (GDD CR-5).
	HealthDead
)

// String returns the state name for logging and test output.
func (h HealthState) String() string {
	switch h {
	case HealthAlive:
		return "Alive"
	case HealthDying:
		return "Dying"
	case HealthKnockedOut:
		return "KnockedOut"
	case HealthStable:
		return "Stable"
	case HealthDead:
		return "Dead"
	default:
		return "Unknown"
	}
}

// DamageDealer applies resolved attack damage to combatants and owns the 0 HP
// transition (Story 005). The death system implements it; the game wiring
// hands it to the enemy AI and opportunity attacks via
// NewDamageApplyingAttacker so resolved hits land. Story 003/004 only resolve
// attacks (AttackResult) — applying the result is this story's territory.
type DamageDealer interface {
	// ApplyAttack applies a resolved attack's damage to the target: a miss or
	// non-positive damage is a no-op; a hit reduces the target's Health
	// (clamped at 0) and routes the 0 HP transition by side and party
	// configuration (GDD CR-5). Damage to a target that is already dying,
	// knocked out, stable, or dead is ignored. Targets outside the encounter
	// are ignored (fail-closed).
	ApplyAttack(target entity.GameEntityID, result AttackResult)
}

// PartyReader supplies the party configuration the death system reads (GDD
// CR-5): which combatant is the player character and whether companion
// permadeath is enabled. The game wiring implements it from the Party System
// (TR-party-001); tests inject exact values. This mirrors the
// EquipmentReader pattern: the interface lives in combat, the wiring owns the
// data source.
type PartyReader interface {
	// IsPlayerCharacter reports whether the entity is the player character
	// (the protagonist). The player character never enters Dying or
	// KnockedOut: 0 HP is an immediate game over (Defeat).
	IsPlayerCharacter(id entity.GameEntityID) bool
	// CompanionDeathEnabled returns the companion_death_enabled knob
	// (GDD CR-5): false → companions at 0 HP are Knocked Out (VS default,
	// revive with 1 HP at combat end); true → companions at 0 HP enter Dying
	// (Full Vision, 3-round countdown, permadeath on expiry).
	CompanionDeathEnabled() bool
}

// Sentinel errors returned by DeathSystem.Stabilize.
var (
	// ErrNotDying is returned when the patient is not in the Dying state
	// (healthy, knocked out, stable, dead, or untracked).
	ErrNotDying = errors.New("combat: target is not dying")
	// ErrNotAdjacent is returned when the stabilizer is not adjacent to the
	// dying target (Chebyshev distance > 1 on the combat grid).
	ErrNotAdjacent = errors.New("combat: stabilizer is not adjacent")
	// ErrNotAlly is returned when the stabilizer is an enemy combatant — a
	// dying character is always party-side (players and allies share the camp,
	// factions.go), so any non-enemy combatant of the encounter may stabilize.
	ErrNotAlly = errors.New("combat: stabilizer is not an ally")
	// ErrHealerIncapacitated is returned when the stabilizer cannot act — at
	// 0 HP (dying, knocked out, stable, dead) or missing the Health
	// component.
	ErrHealerIncapacitated = errors.New("combat: stabilizer cannot act")
	// ErrNotActiveCombatant is returned when a stabilization is attempted
	// during combat by a combatant whose turn is not active — only the active
	// combatant can spend AP.
	ErrNotActiveCombatant = errors.New("combat: stabilizer is not the active combatant")
)

// deathState is the death system's per-combatant tracking. Only combatants in
// a death state are tracked; healthy combatants are absent (StatusOf reports
// HealthAlive for them).
type deathState struct {
	// side is the combatant's side, captured at the state transition.
	side Side
	// status is the tracked state: Dying, KnockedOut, Stable, or Dead.
	status HealthState
	// roundsRemaining is the dying countdown: DyingRounds on entry,
	// decremented at the end of each of the dying character's own turns;
	// permanent death follows when it reaches 0.
	roundsRemaining int
}

// DeathSystem owns the 0 HP transition, the dying countdown, stabilization,
// and the combat-end revival rules (Story 005, TR-combat-004, ADR-0009).
//
// It reads combatant sides from the CombatStarted turn order (snapshotted at
// encounter start — the manager clears its order on finalize, but post-combat
// stabilization still needs sides), writes Health components through the
// Entity Framework, and resolves deaths through the injected Manager
// (RemoveCombatant — enemy death and countdown expiry remove the combatant
// from the turn order immediately; the player character's removal resolves
// the encounter with Defeat through the manager's existing end conditions).
// All notifications go through EventBus events (ADR-0011) — the death event
// emits before the manager's CombatantRemoved, so subscribers observe the
// death before the removal.
//
// Combat-end processing (EventCombatEnded) never calls RemoveCombatant: the
// manager has already finalized (turn order cleared, clock resumed) by the
// time the event dispatches; it only writes Health and emits events. In Full
// Vision mode a dying character's countdown pauses and the player may
// stabilize them as a free post-combat action (adjacent, no AP cost) — the
// stabilization revives them with 1 HP. In VS mode Knocked Out companions
// revive with 1 HP automatically. The dead stay dead (permadeath, AC-5).
//
// Not safe for concurrent use — the game drives combat from a single thread
// (same contract as CombatManager).
type DeathSystem struct {
	bus     entity.EventBus
	store   entity.EntityStore
	manager Manager
	// position supplies the combatants' world positions for the adjacency
	// check (the same Positioner the enemy AI and positioning use).
	position Positioner
	// party supplies the player-character identity and the
	// companion_death_enabled knob (owned by the Party System, GDD CR-5).
	party PartyReader
	// ap is the shared per-turn AP pool the Stabilize action spends from
	// (StabilizeCost during combat; the post-combat free action spends
	// nothing).
	ap *APPool
	// dyingRounds is the countdown length (Config.DyingRounds,
	// DefaultDyingRounds).
	dyingRounds int

	// sides snapshots every combatant's side from CombatStarted for the
	// encounter — the death system still needs sides after the manager
	// finalizes (post-combat stabilization).
	sides map[entity.GameEntityID]Side
	// states tracks the combatants in a death state (Dying, KnockedOut,
	// Stable, Dead) for the current encounter.
	states map[entity.GameEntityID]deathState
}

// Compile-time assertion: DeathSystem satisfies DamageDealer.
var _ DamageDealer = (*DeathSystem)(nil)

// NewDeathSystem creates the Story 005 death system (dependency injection —
// no singletons). bus must be the same EventBus wired into the combat
// manager; store provides the Health components; manager provides removal
// and the active-turn/combat state; position provides adjacency; party
// provides the player-character identity and the companion death knob; ap is
// the shared AP pool the Stabilize action spends from (a nil ap falls back to
// a fresh empty pool — in-combat stabilization then fails with
// ErrInsufficientAP, fail-safe). Config zero values fall back to the
// defaults (DefaultDyingRounds).
//
// The constructor subscribes the system to EventCombatStarted (state reset +
// side snapshot), EventTurnEnded (dying countdown), and EventCombatEnded
// (revival / countdown pause) — the constructor-subscription pattern
// opportunity.go uses.
//
// Example usage:
//
//	manager := combat.NewCombatManager(bus, store, clock, ai, pool, combat.Config{})
//	death := combat.NewDeathSystem(bus, store, manager, positioner, party, pool, combat.Config{})
//	attacker := combat.NewDamageApplyingAttacker(
//	    combat.NewD20Attacker(store, equipment, roller, positioning), death)
func NewDeathSystem(bus entity.EventBus, store entity.EntityStore, manager Manager, position Positioner, party PartyReader, ap *APPool, config Config) *DeathSystem {
	if ap == nil {
		ap = NewAPPool()
	}
	if config.DyingRounds <= 0 {
		config.DyingRounds = DefaultDyingRounds
	}
	d := &DeathSystem{
		bus:         bus,
		store:       store,
		manager:     manager,
		position:    position,
		party:       party,
		ap:          ap,
		dyingRounds: config.DyingRounds,
		sides:       make(map[entity.GameEntityID]Side),
		states:      make(map[entity.GameEntityID]deathState),
	}
	bus.Subscribe(EventCombatStarted, 10, d.handleCombatStarted)
	bus.Subscribe(EventTurnEnded, 10, d.handleTurnEnded)
	bus.Subscribe(EventCombatEnded, 10, d.handleCombatEnded)
	return d
}

// ApplyAttack implements DamageDealer. A miss or non-positive damage is a
// no-op. A hit reduces the target's Health (clamped at 0); reaching 0 routes
// the transition (GDD CR-5):
//
//   - Enemy → instant death: EnemyDefeated + RemoveCombatant(RemovalDefeated).
//   - Player character → instant game over: CharacterDied +
//     RemoveCombatant(RemovalDefeated) — the manager's end conditions resolve
//     the encounter with Defeat (the player character is the sole SidePlayer).
//   - Companion with CompanionDeathEnabled → Dying, countdown DyingRounds,
//     CharacterDying.
//   - Companion with death disabled (VS) → KnockedOut, CharacterKnockedOut.
//
// Damage to an already-incapacitated target (dying, knocked out, stable, or
// dead) and to entities outside the encounter is ignored (fail-closed).
func (d *DeathSystem) ApplyAttack(target entity.GameEntityID, result AttackResult) {
	if !result.Hit || result.Damage <= 0 {
		return
	}
	if _, ok := d.states[target]; ok {
		return // already dying, knocked out, stable, or dead
	}
	if _, ok := d.sides[target]; !ok {
		return // not a combatant of this encounter (fail-closed)
	}
	hp, ok := entity.GetComponent[entity.Health](d.store, target)
	if !ok || hp.Current <= 0 {
		return // no Health or already at 0 (defensive — see above)
	}
	hp.Current -= result.Damage
	if hp.Current < 0 {
		hp.Current = 0
	}
	if err := entity.SetComponent(d.store, target, hp); err != nil {
		return // write failed (destroyed mid-dispatch) — defensive
	}
	if hp.Current > 0 {
		return // damaged but alive — no death transition
	}
	d.enterZeroHP(target)
}

// enterZeroHP routes the 0 HP transition by side and party configuration
// (GDD CR-5). The caller guarantees the target is a tracked combatant at 0 HP
// that is not already in a death state.
func (d *DeathSystem) enterZeroHP(id entity.GameEntityID) {
	side := d.sides[id]
	if side == SideEnemy {
		// Enemies die instantly (AC-4): no dying state, no countdown.
		d.states[id] = deathState{side: side, status: HealthDead}
		d.emit(EventEnemyDefeated, EnemyDefeatedData{EntityID: id})
		d.remove(id)
		return
	}
	if d.party.IsPlayerCharacter(id) {
		// The player character never enters Dying or KnockedOut (Story 005
		// item 6): 0 HP is an immediate game over — the removal resolves the
		// encounter with Defeat through the manager's end conditions.
		d.states[id] = deathState{side: side, status: HealthDead}
		d.emit(EventCharacterDied, CharacterDiedData{EntityID: id})
		d.remove(id)
		return
	}
	if d.party.CompanionDeathEnabled() {
		// Full Vision: Dying with a 3-round countdown (AC-1).
		d.states[id] = deathState{side: side, status: HealthDying, roundsRemaining: d.dyingRounds}
		d.emit(EventCharacterDying, CharacterDyingData{EntityID: id, RoundsRemaining: d.dyingRounds})
		return
	}
	// VS: Knocked Out — no countdown, revives with 1 HP at combat end.
	d.states[id] = deathState{side: side, status: HealthKnockedOut}
	d.emit(EventCharacterKnockedOut, CharacterKnockedOutData{EntityID: id})
}

// Stabilize stabilizes a dying character: an adjacent, capable ally spends
// StabilizeCost AP during combat — the countdown pauses, HP stays 0, and the
// character survives (AC-2; revive with 1 HP at combat end). After combat
// (Full Vision) the stabilization is a free action — no AP, no active-turn
// requirement — and revives the character with 1 HP immediately (GDD CR-6
// step 6, edge case: post-combat stabilization).
//
// Validations, in order: the patient is Dying (ErrNotDying); the stabilizer
// is adjacent on the grid (ErrNotAdjacent); the stabilizer is a non-enemy
// combatant of the encounter (ErrNotAlly — a dying character is always
// party-side, so players and allies both qualify); the stabilizer has
// Health.Current > 0 (ErrHealerIncapacitated — a dying character cannot
// stabilize itself or anyone else); during combat the stabilizer's turn is
// active and the pool covers StabilizeCost (ErrNotActiveCombatant,
// ErrInsufficientAP).
func (d *DeathSystem) Stabilize(healer, patient entity.GameEntityID) error {
	st, ok := d.states[patient]
	if !ok || st.status != HealthDying {
		return ErrNotDying
	}
	healerPos, ok := d.position.Position(healer)
	if !ok {
		return ErrNotAdjacent
	}
	patientPos, ok := d.position.Position(patient)
	if !ok {
		return ErrNotAdjacent
	}
	if Chebyshev(WorldToTile(healerPos), WorldToTile(patientPos)) > 1 {
		return ErrNotAdjacent
	}
	healerSide, ok := d.sides[healer]
	if !ok || healerSide == SideEnemy {
		return ErrNotAlly
	}
	hp, ok := entity.GetComponent[entity.Health](d.store, healer)
	if !ok || hp.Current <= 0 {
		return ErrHealerIncapacitated
	}
	if d.manager.IsCombatActive() {
		current, ok := d.manager.CurrentCombatant()
		if !ok || current != healer {
			return ErrNotActiveCombatant
		}
		if err := d.ap.Spend(StabilizeCost); err != nil {
			return err // ErrInsufficientAP — the pool is left unchanged
		}
	}
	st.status = HealthStable
	d.states[patient] = st
	d.emit(EventCharacterStabilized, CharacterStabilizedData{
		EntityID:       patient,
		HealerID:       healer,
		RoundsRemaining: st.roundsRemaining,
	})
	if !d.manager.IsCombatActive() {
		// Post-combat (FV): the free stabilization revives immediately (GDD
		// CR-6: "If stabilized, they revive with 1 HP").
		d.revive(patient)
	}
	return nil
}

// StatusOf returns the combatant's tracked health state (GDD CR-5). Entities
// the death system does not track — healthy combatants and unknown entities —
// report HealthAlive.
func (d *DeathSystem) StatusOf(id entity.GameEntityID) HealthState {
	if st, ok := d.states[id]; ok {
		return st.status
	}
	return HealthAlive
}

// remove removes a dead combatant from the encounter (enemy death, player
// character death, countdown expiry). The death event has already fired —
// removal emits CombatantRemoved afterwards, so subscribers observe the death
// before the removal (ADR-0011 synchronous ordering). Errors are ignored
// defensively: the caller has already recorded the Dead state, and removal
// can only fail if the encounter finalized concurrently, which is unreachable
// within the synchronous call path (ApplyAttack only runs during active
// combat).
func (d *DeathSystem) remove(id entity.GameEntityID) {
	_ = d.manager.RemoveCombatant(id, RemovalDefeated)
}

// revive restores a character to 1 HP and clears its tracked state (combat
// end: Knocked Out at VS, Stable at FV; post-combat stabilization). The
// character is alive again — StatusOf reports HealthAlive afterwards.
func (d *DeathSystem) revive(id entity.GameEntityID) {
	hp, ok := entity.GetComponent[entity.Health](d.store, id)
	if !ok {
		return // no Health component — nothing to revive (defensive)
	}
	hp.Current = 1
	if err := entity.SetComponent(d.store, id, hp); err != nil {
		return // write failed (destroyed mid-dispatch) — defensive
	}
	delete(d.states, id)
}

// handleCombatStarted resets the per-encounter state and snapshots the
// combatants' sides from the CombatStarted turn order. The snapshot outlives
// the encounter's turn order (the manager clears it on finalize) so
// post-combat stabilization can still judge sides.
func (d *DeathSystem) handleCombatStarted(ev entity.Event) {
	data, ok := ev.Data.(CombatStartedData)
	if !ok {
		return
	}
	clear(d.states)
	clear(d.sides)
	for _, e := range data.TurnOrder {
		d.sides[e.EntityID] = e.Side
	}
}

// handleTurnEnded decrements the dying countdown at the end of the dying
// character's own turn (Story 005 item 2): reaching 0 is permanent death —
// CharacterDied fires and the combatant is removed from the turn order
// (RemovalDefeated). Stabilized (Stable) characters no longer tick.
func (d *DeathSystem) handleTurnEnded(ev entity.Event) {
	data, ok := ev.Data.(TurnEndedData)
	if !ok {
		return
	}
	st, ok := d.states[data.Combatant]
	if !ok || st.status != HealthDying {
		return
	}
	st.roundsRemaining--
	if st.roundsRemaining > 0 {
		d.states[data.Combatant] = st
		return
	}
	// The countdown elapsed (AC-3): permanent death.
	st.status = HealthDead
	d.states[data.Combatant] = st
	d.emit(EventCharacterDied, CharacterDiedData{EntityID: data.Combatant})
	d.remove(data.Combatant)
}

// handleCombatEnded applies the combat-end rules (GDD CR-6, edge cases): VS —
// Knocked Out companions revive with 1 HP; FV — Stable characters revive with
// 1 HP, Dying countdowns pause (the player may stabilize as a free post-combat
// action), and the dead stay dead (permadeath, AC-5). The manager has already
// finalized by the time this dispatches, so only Health writes and events
// happen here — never RemoveCombatant. A dying character with 0 rounds
// remaining is a defensive case (unreachable in normal flow — death fires at
// the turn end that reaches 0): it dies at combat resolution.
func (d *DeathSystem) handleCombatEnded(entity.Event) {
	for id, st := range d.states {
		if st.side == SideEnemy {
			continue // defeated enemies stay dead
		}
		switch st.status {
		case HealthKnockedOut, HealthStable:
			d.revive(id)
		case HealthDying:
			if st.roundsRemaining <= 0 {
				// Defensive: the countdown had elapsed — die at resolution.
				st.status = HealthDead
				d.states[id] = st
				d.emit(EventCharacterDied, CharacterDiedData{EntityID: id})
			}
			// Otherwise the countdown pauses: HP stays 0, the character
			// remains Dying and can be stabilized as a free post-combat
			// action.
		}
	}
}

// emit fires a Story 005 death/dying event through the EventBus.
func (d *DeathSystem) emit(eventType entity.EventType, data any) {
	d.bus.Emit(entity.Event{Type: eventType, Source: "combat", Data: data})
}

// damageApplyingAttacker is the Story 005 wiring decorator: every attack
// resolved through it also applies its damage through the DamageDealer (the
// death system), so the enemy AI and OpportunityAttacker — which only know
// the Attacker interface — land damage without signature changes.
type damageApplyingAttacker struct {
	attacker Attacker
	damage   DamageDealer
}

// Compile-time assertion: damageApplyingAttacker satisfies Attacker.
var _ Attacker = (*damageApplyingAttacker)(nil)

// NewDamageApplyingAttacker wraps an Attacker so every resolved hit is
// applied through the DamageDealer — the single choke point that makes Story
// 003/004 attack resolution actually deal damage (the enemy AI and
// OpportunityAttacker resolve through their Attacker and would otherwise drop
// the result). Misses are passed through untouched (ApplyAttack no-ops on
// them). The wrapped result is returned unchanged.
//
// Example usage:
//
//	attacker := combat.NewDamageApplyingAttacker(
//	    combat.NewD20Attacker(store, equipment, roller, positioning), death)
//	ai := combat.NewBasicEnemyAI(positioner, attacker, mover, combat.DefaultAttackRange)
//	opportunity := combat.NewOpportunityAttacker(bus, positioner, attacker, factions)
func NewDamageApplyingAttacker(attacker Attacker, damage DamageDealer) Attacker {
	return &damageApplyingAttacker{attacker: attacker, damage: damage}
}

// ResolveAttack implements Attacker: resolves the attack through the wrapped
// attacker and applies the result through the DamageDealer.
func (a *damageApplyingAttacker) ResolveAttack(attacker, target entity.GameEntityID) AttackResult {
	result := a.attacker.ResolveAttack(attacker, target)
	a.damage.ApplyAttack(target, result)
	return result
}

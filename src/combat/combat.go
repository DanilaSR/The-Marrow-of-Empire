// Package combat implements the tactical combat state machine and initiative
// system (Story 001, TR-combat-006, ADR-0009): six states
// (Inactive → Initiating → PlayerTurn → EnemyTurn → AllyTurn → Resolution),
// initiative = DEX_mod + WIS_mod with deterministic tiebreaking, a fixed turn
// order per encounter, combat lifecycle events over the synchronous EventBus
// (ADR-0011), and the basic enemy turn (attack nearest target or move toward
// it).
//
// Story 002 (Grid Movement & AP Economy, TR-combat-001/002) adds the square
// tile grid (grid.go), the per-turn AP economy (ap.go: AP = 4 + DEX_mod,
// clamped [2, 8]; costs Move 1, Attack 2, Ability 1-3, Interact 1), and the
// tile-based GridMover behind the Mover interface (grid_mover.go). The
// CombatManager owns the AP pool lifecycle: it resets the pool to the active
// combatant's AP at every turn start — AP never carries over — and clears it
// when combat finalizes; actions spend from the same injected pool.
//
// Story 003 (Attack Resolution, TR-combat-003) adds the d20 resolver
// (attack_resolution.go): d20 + accuracy >= evasion → hit, natural 20 =
// critical hit (double weapon base damage), natural 1 = auto-miss, and
// damage = max(1, weapon base + STR_mod (melee) or DEX_mod (ranged) -
// target armor). It implements the Story 001 Attacker interface; the d20
// roll arrives through the injected Roller (RNG injection for deterministic
// tests) and weapon/armor data through the injected EquipmentReader (the
// item data model belongs to the Loot & Economy system).
//
// Story 004 (Positioning & Cover, TR-combat-005) adds the positional
// bonuses (positioning.go): flanking +2 accuracy when an adjacent ally
// attacks from the opposite side, half cover +2 evasion from an obstacle
// adjacent to the defender, full cover blocks line of sight (grid-level
// Bresenham), and elevation +1 accuracy/+1 range per level above the target
// (per-tile terrain in TileGrid; climbing a level costs ClimbCost AP). The
// d20 resolver consumes the bonuses through the injected PositioningResolver
// and honors the line-of-sight gate (attack_resolution.go). Opportunity
// attacks (opportunity.go): a combatant leaving an enemy's melee range
// triggers a free attack from that enemy, once per turn per enemy — GridMover
// reports every step through the OpportunityWatcher, and the attacker resets
// its per-turn budget at every EventTurnStarted. Faction relationships
// (factions.go) derive from the manager's turn order via
// NewManagerFactions.
//
// Story 005 (Death, Dying & Stabilization, TR-combat-004) adds the death
// system (death.go): resolved attack damage lands through the DamageDealer
// interface (ApplyAttack), and the 0 HP transition is routed by side and the
// injected PartyReader — enemies die instantly (EnemyDefeated + removal),
// the player character's death is an immediate game over (Defeat), and
// companions either enter Dying with a DyingRounds countdown (Full Vision,
// companion_death_enabled) or are Knocked Out (VS). Stabilization costs
// StabilizeCost AP from an adjacent ally; the countdown decrements at the end
// of the dying character's own turn and permanent death removes the
// combatant via RemoveCombatant. At combat end Knocked Out companions revive
// with 1 HP, Dying countdowns pause for free post-combat stabilization, and
// the dead stay dead (permadeath). The game wiring makes the enemy AI and
// opportunity attacks apply damage through NewDamageApplyingAttacker.
//
// This package is pure Go game logic with no Kaiju engine dependency: it
// reads entity components through the Entity Framework (entity.GetComponent),
// drives the game clock through the Clock interface (ADR-0007: combat pauses
// the clock), and notifies other systems — camera, UI, Persistent World
// State — exclusively through EventBus events (ADR-0011: no direct
// cross-system calls for state mutations).
//
// Out of Scope (neighbouring stories): full combat resolution (Story 006 —
// XP, loot, KO revive; CombatEndedData.Outcome is the hook; it also consumes
// the PositioningModifiers.Range bonus for range-aware targeting), revive
// abilities, and sophisticated enemy AI (future).
//
// Stories: production/epics/tactical-combat-engine/
// story-001-combat-state-machine-initiative.md,
// story-002-grid-movement-ap-economy.md,
// story-003-attack-resolution-d20.md,
// story-004-positioning-cover-system.md
package combat

import (
	"errors"
	"fmt"

	"marrow-of-empire/entity"
)

// Clock is the subset of the game clock the combat system drives (ADR-0007:
// "Combat pauses clock — time frozen during combat"). *env.GameClock
// satisfies it.
type Clock interface {
	// Pause freezes time advancement.
	Pause()
	// Resume unfreezes time advancement.
	Resume()
}

// State is the combat state machine (TR-combat-006, ADR-0009): six states
// sequenced Inactive → Initiating → PlayerTurn → EnemyTurn → AllyTurn →
// Resolution. Explicit transition table:
//
//	Inactive   --StartCombat (validated)-->                       Initiating
//	Initiating --first combatant's side-->                        PlayerTurn | EnemyTurn | AllyTurn
//	PlayerTurn | EnemyTurn | AllyTurn --EndTurn-->                next combatant's state | Resolution
//	PlayerTurn | EnemyTurn | AllyTurn --last living enemy gone--> Resolution (Victory)
//	PlayerTurn | EnemyTurn | AllyTurn --last living player gone--> Resolution (Defeat)
//	PlayerTurn | EnemyTurn | AllyTurn --EndCombat-->              Resolution (Scripted)
//	Resolution  --finalize (CombatEnded + clock resume)-->        Inactive
//
// Invalid transitions are rejected with errors: StartCombat while active,
// EndTurn/RemoveCombatant with no active encounter, EndTurn with no active
// turn.
type State int

const (
	// StateInactive is the zero state: no encounter running.
	StateInactive State = iota
	// StateInitiating is entered by StartCombat while initiative is rolled.
	// Transient: the first combatant's turn begins immediately after
	// CombatStarted fires (AC-1).
	StateInitiating
	// StatePlayerTurn is active while a player-side combatant's turn runs.
	// The player acts; the turn ends via an external EndTurn call.
	StatePlayerTurn
	// StateEnemyTurn is active while an enemy-side combatant's turn runs.
	// The basic enemy AI acts automatically on entry (AC-6).
	StateEnemyTurn
	// StateAllyTurn is active while an ally-side combatant's turn runs.
	// Ally behavior is deferred; the turn ends via an external EndTurn call.
	StateAllyTurn
	// StateResolution is entered when the encounter's end condition is met.
	// Transient: finalize (CombatEnded, clock resume) follows immediately;
	// Story 006 extends this phase.
	StateResolution
)

// String returns the state name for logging and test output.
func (s State) String() string {
	switch s {
	case StateInactive:
		return "Inactive"
	case StateInitiating:
		return "Initiating"
	case StatePlayerTurn:
		return "PlayerTurn"
	case StateEnemyTurn:
		return "EnemyTurn"
	case StateAllyTurn:
		return "AllyTurn"
	case StateResolution:
		return "Resolution"
	default:
		return "Unknown"
	}
}

// Side is a combatant's allegiance within an encounter. Sides are assigned by
// the combat trigger at StartCombat (player attacks hostile, hostile detects
// player, or conversation turns violent — CR-1).
type Side int

const (
	// SidePlayer is the player character (and player-controlled party).
	SidePlayer Side = iota
	// SideEnemy is hostile to players and allies.
	SideEnemy
	// SideAlly fights alongside the player but is not player-controlled.
	SideAlly
)

// String returns the side name for logging and test output.
func (s Side) String() string {
	switch s {
	case SidePlayer:
		return "Player"
	case SideEnemy:
		return "Enemy"
	case SideAlly:
		return "Ally"
	default:
		return "Unknown"
	}
}

// Outcome is how an encounter resolved (AC-4, CR-6). Story 006 consumes it
// for XP calculation, loot drops, and post-combat stabilization.
type Outcome int

const (
	// OutcomeVictory: no enemies remain alive or in combat — all dead, fled,
	// or surrendered (AC-4).
	OutcomeVictory Outcome = iota
	// OutcomeDefeat: no player characters remain alive (CR-6; the game-over
	// reload flow is Story 005/006 territory).
	OutcomeDefeat
	// OutcomeScripted: an external system ended combat (CR-6 scripted events:
	// reinforcements, environmental collapse).
	OutcomeScripted
)

// String returns the outcome name for logging and test output.
func (o Outcome) String() string {
	switch o {
	case OutcomeVictory:
		return "Victory"
	case OutcomeDefeat:
		return "Defeat"
	case OutcomeScripted:
		return "Scripted"
	default:
		return "Unknown"
	}
}

// RemovalReason is why a combatant left the encounter (AC-4: dead, fled, or
// surrendered; AC-7: flee).
type RemovalReason int

const (
	// RemovalDefeated: the combatant died. Story 005 (death and dying) calls
	// RemoveCombatant with this reason when HP reaches 0.
	RemovalDefeated RemovalReason = iota
	// RemovalFled: the combatant left the combat area (AC-7). No XP is
	// awarded for fleeing enemies (Story 006 uses the removal reasons).
	RemovalFled
	// RemovalSurrendered: the combatant yielded.
	RemovalSurrendered
)

// String returns the removal reason name for logging and test output.
func (r RemovalReason) String() string {
	switch r {
	case RemovalDefeated:
		return "Defeated"
	case RemovalFled:
		return "Fled"
	case RemovalSurrendered:
		return "Surrendered"
	default:
		return "Unknown"
	}
}

// CombatantEntry registers one combatant at StartCombat.
type CombatantEntry struct {
	// EntityID is the combatant's game entity. It must carry Stats (for
	// initiative) and Health (for the alive check) components.
	EntityID entity.GameEntityID
	// Side is the combatant's allegiance for the encounter.
	Side Side
}

// TurnEntry is one position in the fixed turn order (AC-2). The order is
// rolled once at combat start and never recalculated during the encounter;
// removed combatants leave the order (AC-4, AC-7).
type TurnEntry struct {
	// EntityID is the combatant.
	EntityID entity.GameEntityID
	// Side is the combatant's side; it selects the turn state.
	Side Side
	// Initiative is the rolled value DEX_mod + WIS_mod (ADR-0009). May be
	// negative or zero (AC-2 edge case).
	Initiative int
	// Dexterity is the raw DEX attribute — the tiebreak key after
	// initiative (AC-2).
	Dexterity int
}

// Sentinel errors returned by the combat manager for invalid transitions and
// inputs.
var (
	// ErrNoCombatants is returned by StartCombat with an empty combatant
	// list.
	ErrNoCombatants = errors.New("combat: no combatants")
	// ErrTooManyCombatants is returned when StartCombat exceeds
	// Config.MaxCombatants.
	ErrTooManyCombatants = errors.New("combat: too many combatants")
	// ErrDuplicateCombatant is returned when StartCombat registers the same
	// entity twice.
	ErrDuplicateCombatant = errors.New("combat: duplicate combatant")
	// ErrCombatActive is returned when StartCombat is called while an
	// encounter is already running. Multiple simultaneous triggers start
	// combat exactly once with all detected combatants (AC-1 edge case).
	ErrCombatActive = errors.New("combat: encounter already active")
	// ErrNoActiveCombat is returned by turn/removal calls with no encounter
	// running.
	ErrNoActiveCombat = errors.New("combat: no active encounter")
	// ErrNotCombatant is returned by RemoveCombatant for an entity not in
	// the encounter.
	ErrNotCombatant = errors.New("combat: entity is not a combatant")
	// ErrNoActiveTurn is returned by EndTurn while no combatant's turn is
	// active (Inactive, Initiating, or Resolution).
	ErrNoActiveTurn = errors.New("combat: no active turn")
	// ErrMissingStats is returned by StartCombat when a combatant lacks the
	// Stats component (initiative requires DEX_mod + WIS_mod).
	ErrMissingStats = errors.New("combat: combatant missing Stats component")
	// ErrMissingHealth is returned by StartCombat when a combatant lacks the
	// Health component (the alive check reads it).
	ErrMissingHealth = errors.New("combat: combatant missing Health component")
)

// Manager is the public interface of the combat system (dependency-injection
// contract). Trigger systems (player attack, hostile detection, violent
// dialogue) call StartCombat; the player input/UI layer calls EndTurn; the
// death system (Story 005) and flee detection call RemoveCombatant; the save
// system checks IsCombatActive (AC-5); Story 006 extends the Resolution
// phase via the CombatEnded event.
type Manager interface {
	// StartCombat begins an encounter with the given combatants (AC-1):
	// validates the combatants, pauses the clock (ADR-0007), rolls
	// initiative, emits CombatStarted (carrying the turn order for display),
	// and begins the first combatant's turn.
	StartCombat(combatants []CombatantEntry) error

	// EndTurn ends the current combatant's turn and advances the turn order
	// (AC-3). Enemy turns that follow process automatically. Returns
	// ErrNoActiveTurn when no combatant turn is active and
	// ErrNoActiveCombat when no encounter is running.
	EndTurn() error

	// RemoveCombatant removes a combatant from the encounter (AC-4, AC-7:
	// defeated, fled, or surrendered). Removing the last living enemy
	// resolves with Victory; losing the last living player character
	// resolves with Defeat. Returns ErrNotCombatant for entities not in the
	// encounter.
	RemoveCombatant(id entity.GameEntityID, reason RemovalReason) error

	// EndCombat ends the encounter with a scripted outcome (CR-6 scripted
	// events: reinforcements, environmental collapse). The active turn, if
	// any, ends; CombatEnded fires; the clock resumes.
	EndCombat(outcome Outcome) error

	// IsCombatActive reports whether an encounter is running (AC-5: the save
	// system blocks quick-save and auto-save while true; manual save always
	// works). True from StartCombat until the encounter finalizes.
	IsCombatActive() bool

	// CurrentState returns the active state machine state.
	CurrentState() State

	// CurrentCombatant returns the entity whose turn is active, or
	// (0, false) when no turn is active.
	CurrentCombatant() (entity.GameEntityID, bool)

	// TurnOrder returns a copy of the fixed turn order (AC-2) — the
	// combatants still in the encounter, in initiative order.
	TurnOrder() []TurnEntry

	// Round returns the current round number, 1-based; it increments when
	// the turn order wraps.
	Round() int

	// CurrentAP returns the active combatant's remaining action points
	// (Story 002, TR-combat-002). The pool resets to the combatant's AP
	// (4 + DEX_mod, clamped [2, 8]) at every turn start and never carries
	// over; the turn ends when it reaches 0 — enemy turns end after the AI
	// acts, and the input layer ends player/ally turns via EndTurn. Returns
	// 0 when no turn is active or combat has finalized.
	CurrentAP() int

	// MaxAP returns the active combatant's per-turn action point budget:
	// the value CurrentAP was reset to at turn start (4 + DEX_mod, clamped
	// [2, 8]). Returns 0 when no turn is active or combat has finalized.
	MaxAP() int
}

// CombatManager is the concrete combat state machine (Story 001). It owns the
// encounter's turn order, state, and lifecycle events; all external effects
// (clock pause/resume, camera, UI) happen through injected dependencies and
// EventBus emissions — never direct cross-system calls (ADR-0011). It is not
// safe for concurrent use — the game drives combat from a single thread
// (same contract as save.SaveManagerImpl).
type CombatManager struct {
	bus    entity.EventBus
	store  entity.EntityStore
	clock  Clock
	ai     EnemyAI
	config Config

	// ap is the shared per-turn AP pool (Story 002): reset to the active
	// combatant's AP at every turn start, spent by actions (movement via
	// GridMover, attacks/abilities in later stories), cleared on finalize.
	ap *APPool

	state   State
	outcome Outcome
	round   int

	// turnOrder is the fixed encounter order (AC-2) minus removed
	// combatants. current is the combatant whose turn is active (0 when no
	// turn is active); currentIndex is the position in turnOrder where the
	// next turn begins — during an active turn it is the successor of
	// current. The two stay consistent when the active combatant is removed
	// mid-turn: current keeps naming the active combatant while the pointer
	// already rests on the next one (AC-3 edge: "complete their turn, then
	// remove from order").
	turnOrder    []TurnEntry
	current      entity.GameEntityID
	currentIndex int
}

// Compile-time assertion: CombatManager satisfies Manager.
var _ Manager = (*CombatManager)(nil)

// NewCombatManager creates the combat manager (dependency injection — no
// singletons). ap is the shared per-turn AP pool (Story 002): the manager
// resets it to the active combatant's AP at every turn start and the action
// systems (GridMover, later attack/ability systems) spend from the same
// pool. A nil ap falls back to a fresh unshared pool. Config zero values
// fall back to the defaults (DefaultAttackRange, DefaultMaxCombatants).
//
// Example usage:
//
//	pool := combat.NewAPPool()
//	mover := combat.NewGridMover(positioner, setter, blocker, pool)
//	ai := combat.NewBasicEnemyAI(positioner, attacker, mover, combat.DefaultAttackRange)
//	manager := combat.NewCombatManager(bus, store, clock, ai, pool, combat.Config{})
func NewCombatManager(bus entity.EventBus, store entity.EntityStore, clock Clock, ai EnemyAI, ap *APPool, config Config) *CombatManager {
	if ap == nil {
		ap = NewAPPool()
	}
	if config.AttackRange <= 0 {
		config.AttackRange = DefaultAttackRange
	}
	if config.MaxCombatants <= 0 {
		config.MaxCombatants = DefaultMaxCombatants
	}
	return &CombatManager{
		bus:    bus,
		store:  store,
		clock:  clock,
		ai:     ai,
		ap:     ap,
		config: config,
	}
}

// StartCombat implements Manager. Sequence (AC-1): validate the combatants →
// pause the game clock (ADR-0007) → roll initiative (Initiating) → emit
// CombatStarted with the turn order (camera and UI react) → begin the first
// combatant's turn. Enemies first in order act immediately — their turns
// process automatically until a player or ally turn (or resolution) is
// reached.
//
// Fails — without pausing the clock or mutating state — when the combatant
// list is empty, exceeds MaxCombatants, contains duplicates, an encounter is
// already active (multiple simultaneous triggers start combat exactly once),
// or a combatant lacks the Stats or Health component.
func (m *CombatManager) StartCombat(combatants []CombatantEntry) error {
	if m.IsCombatActive() {
		return ErrCombatActive
	}
	if len(combatants) == 0 {
		return ErrNoCombatants
	}
	if len(combatants) > m.config.MaxCombatants {
		return fmt.Errorf("%w: %d combatants, max %d", ErrTooManyCombatants, len(combatants), m.config.MaxCombatants)
	}
	seen := make(map[entity.GameEntityID]bool, len(combatants))
	for _, c := range combatants {
		if seen[c.EntityID] {
			return fmt.Errorf("%w: %d", ErrDuplicateCombatant, c.EntityID)
		}
		seen[c.EntityID] = true
		if _, ok := entity.GetComponent[entity.Stats](m.store, c.EntityID); !ok {
			return fmt.Errorf("%w: %d", ErrMissingStats, c.EntityID)
		}
		if _, ok := entity.GetComponent[entity.Health](m.store, c.EntityID); !ok {
			return fmt.Errorf("%w: %d", ErrMissingHealth, c.EntityID)
		}
	}

	m.state = StateInitiating
	m.round = 1
	m.currentIndex = 0
	m.current = 0
	m.outcome = OutcomeVictory // overwritten by resolution if needed

	order, err := rollInitiative(m.store, combatants)
	if err != nil {
		m.state = StateInactive
		return err
	}
	m.turnOrder = order

	m.clock.Pause() // ADR-0007: combat freezes the game clock
	m.emitCombatStarted()
	m.startTurn()
	return nil
}

// EndTurn implements Manager: ends the active turn (TurnEnded), advances the
// turn pointer (the round increments when the order wraps), and begins the
// next turn — enemy turns process automatically (AC-3).
func (m *CombatManager) EndTurn() error {
	if !m.IsCombatActive() {
		return ErrNoActiveCombat
	}
	if !m.turnActive() {
		return ErrNoActiveTurn
	}
	if !m.endCurrentTurn() {
		m.startTurn()
	}
	return nil
}

// RemoveCombatant implements Manager: removes the combatant from the fixed
// turn order and emits CombatantRemoved (AC-4, AC-7). The turn pointer is
// adjusted so the encounter continues in the correct order; a combatant
// removed before its turn ever began is skipped. Removing the last living
// enemy — or losing the last living player character — resolves the
// encounter immediately: the active turn (if any) is ended so the event
// stream stays balanced (every TurnStarted has a matching TurnEnded), then
// CombatEnded fires and the clock resumes.
func (m *CombatManager) RemoveCombatant(id entity.GameEntityID, reason RemovalReason) error {
	if !m.IsCombatActive() {
		return ErrNoActiveCombat
	}
	idx := -1
	for i, e := range m.turnOrder {
		if e.EntityID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNotCombatant
	}
	m.turnOrder = append(m.turnOrder[:idx], m.turnOrder[idx+1:]...)
	// The active combatant's successor (the next-turn slot) shifted down when
	// the removal was before it; the slot itself is unchanged when the next
	// combatant was removed (its successor slid in) or when the removal was
	// after it.
	if idx < m.currentIndex {
		m.currentIndex--
	}
	m.emitCombatantRemoved(id, reason)
	// The turn activity must be captured before the resolution check —
	// checkResolution transitions to Resolution, after which turnActive
	// reports false. The interrupted turn still ends, so every TurnStarted
	// keeps a matching TurnEnded (AC-3 edge: skip remaining turns).
	turnActive := m.turnActive()
	if m.checkResolution() {
		if turnActive {
			m.emitTurnEnded()
		}
		m.finalizeCombat()
	}
	return nil
}

// EndCombat implements Manager: ends the encounter with a scripted outcome
// (CR-6). The active turn, if any, ends; CombatEnded fires with the given
// outcome; the clock resumes.
func (m *CombatManager) EndCombat(outcome Outcome) error {
	if !m.IsCombatActive() {
		return ErrNoActiveCombat
	}
	if m.turnActive() {
		m.emitTurnEnded()
	}
	m.state = StateResolution
	m.outcome = outcome
	m.finalizeCombat()
	return nil
}

// IsCombatActive implements Manager: true from StartCombat until the
// encounter finalizes (AC-5).
func (m *CombatManager) IsCombatActive() bool {
	return m.state != StateInactive
}

// CurrentState implements Manager.
func (m *CombatManager) CurrentState() State {
	return m.state
}

// CurrentCombatant implements Manager.
func (m *CombatManager) CurrentCombatant() (entity.GameEntityID, bool) {
	if !m.turnActive() {
		return 0, false
	}
	return m.current, true
}

// TurnOrder implements Manager: returns a copy of the fixed turn order.
func (m *CombatManager) TurnOrder() []TurnEntry {
	return append([]TurnEntry(nil), m.turnOrder...)
}

// Round implements Manager.
func (m *CombatManager) Round() int {
	return m.round
}

// CurrentAP implements Manager: the active combatant's remaining action
// points, or 0 when no turn is active.
func (m *CombatManager) CurrentAP() int {
	return m.ap.Current()
}

// MaxAP implements Manager: the active combatant's per-turn AP budget, or
// 0 when no turn is active (the pool is cleared on finalize).
func (m *CombatManager) MaxAP() int {
	return m.ap.Max()
}

// resetAP refills the AP pool at turn start (Story 002, TR-combat-002): AP =
// 4 + DEX_mod, clamped to [2, 8], read from the combatant's Stats component.
// AP never carries over — every turn start resets the pool. A combatant
// missing the Stats component (StartCombat validates it, so this path is
// defensive only) gets 0 AP and cannot act.
func (m *CombatManager) resetAP(id entity.GameEntityID) {
	s, ok := entity.GetComponent[entity.Stats](m.store, id)
	if !ok {
		m.ap.Reset(0)
		return
	}
	m.ap.Reset(APFor(s))
}

// startTurn begins the combatant at the next-turn slot: the state becomes the
// combatant's side state, TurnStarted fires, and enemy turns process
// automatically (the basic AI acts, then the turn advances) until a player or
// ally turn — or resolution — is reached. Subscribers may remove combatants
// during CombatStarted dispatch, so the loop re-checks the encounter is still
// running.
func (m *CombatManager) startTurn() {
	for m.IsCombatActive() && len(m.turnOrder) > 0 {
		entry := m.turnOrder[m.currentIndex]
		m.current = entry.EntityID
		m.state = stateForSide(entry.Side)
		m.resetAP(entry.EntityID) // Story 002: the AP pool refills at every turn start
		m.emitTurnStarted(entry)
		if entry.Side != SideEnemy {
			return // player and ally turns wait for an external EndTurn
		}
		if targets := m.validTargets(entry.EntityID); len(targets) > 0 {
			m.ai.TakeTurn(entry.EntityID, targets)
		}
		if m.endCurrentTurn() {
			return
		}
	}
}

// endCurrentTurn emits TurnEnded for the active turn, moves the pointer to
// the next turn (the round increments when the ending combatant was last in
// the order), and checks the end conditions. Returns true when the encounter
// resolved — the caller must not continue the turn. A turn that already ended
// (resolution fired during the action, e.g. the enemy AI's attack removed the
// last combatant and finalized the encounter) counts as resolved so the
// caller stops.
func (m *CombatManager) endCurrentTurn() bool {
	if !m.turnActive() {
		return true // already resolved mid-action
	}
	m.emitTurnEnded()
	if len(m.turnOrder) == 0 {
		// Defensive: no combatants remain. Removal resolves the encounter as
		// soon as the order empties, so this is unreachable in normal flow —
		// resolve rather than stall if it happens.
		m.state = StateResolution
		m.outcome = OutcomeVictory
		m.finalizeCombat()
		return true
	}
	idx := indexOf(m.turnOrder, m.current)
	if idx == len(m.turnOrder)-1 || (idx < 0 && m.currentIndex == 0) {
		// The ending turn was last in the order (either still present and at
		// the last position, or removed mid-turn). A new round begins.
		m.round++
	}
	m.currentIndex = m.nextTurnIndex()
	if m.checkResolution() {
		m.finalizeCombat()
		return true
	}
	return false
}

// nextTurnIndex returns the position of the combatant whose turn begins after
// m.current's turn ends: the successor of m.current in the current order
// (with wrap), or — when the active combatant was removed mid-turn — the
// currentIndex, which removal already adjusted to the combatant that slid
// into its slot (AC-3 edge: "complete their turn, then remove from order").
func (m *CombatManager) nextTurnIndex() int {
	for i, e := range m.turnOrder {
		if e.EntityID == m.current {
			return (i + 1) % len(m.turnOrder)
		}
	}
	return m.currentIndex
}

// indexOf returns the position of the combatant in the order, or -1.
func indexOf(order []TurnEntry, id entity.GameEntityID) int {
	for i, e := range order {
		if e.EntityID == id {
			return i
		}
	}
	return -1
}

// checkResolution evaluates the end conditions (CR-6): no living enemies →
// Victory (AC-4: all enemies dead, fled, or surrendered); no living player
// characters → Defeat (AC-6 edge). The alive check reads the Health component
// (0 HP counts as not alive even before Story 005 removes the combatant).
// Returns true and transitions to Resolution when an end condition is met.
func (m *CombatManager) checkResolution() bool {
	if m.state == StateInactive || m.state == StateResolution {
		return false
	}
	if m.aliveCount(SideEnemy) == 0 {
		m.state = StateResolution
		m.outcome = OutcomeVictory
		return true
	}
	if m.aliveCount(SidePlayer) == 0 {
		m.state = StateResolution
		m.outcome = OutcomeDefeat
		return true
	}
	return false
}

// finalizeCombat transitions Resolution → Inactive: the clock resumes
// (ADR-0007), CombatEnded fires, and the encounter state is cleared.
// Idempotent — the state is Inactive before CombatEnded dispatches, so a
// CombatEnded subscriber (e.g. the save system) observes a fully cleaned-up
// encounter (AC-5 edge case: "save during Resolution succeeds after combat
// state is cleaned up").
func (m *CombatManager) finalizeCombat() {
	if m.state == StateInactive {
		return
	}
	m.state = StateInactive
	m.clock.Resume()
	m.emitCombatEnded()
	m.turnOrder = nil
	m.current = 0
	m.currentIndex = 0
	m.round = 0
	m.ap.Reset(0) // Story 002: no AP outside a turn
}

// turnActive reports whether a combatant's turn is running.
func (m *CombatManager) turnActive() bool {
	switch m.state {
	case StatePlayerTurn, StateEnemyTurn, StateAllyTurn:
		return true
	}
	return false
}

// stateForSide selects the turn state for a combatant's side (AC-3).
func stateForSide(s Side) State {
	switch s {
	case SidePlayer:
		return StatePlayerTurn
	case SideEnemy:
		return StateEnemyTurn
	default:
		return StateAllyTurn
	}
}

// validTargets returns the alive combatants hostile to the given enemy —
// player and ally combatants with Health.Current > 0 (AC-6). Combatants
// missing Health are treated as invalid targets.
func (m *CombatManager) validTargets(enemy entity.GameEntityID) []entity.GameEntityID {
	var targets []entity.GameEntityID
	for _, e := range m.turnOrder {
		if e.EntityID == enemy || e.Side == SideEnemy {
			continue
		}
		hp, ok := entity.GetComponent[entity.Health](m.store, e.EntityID)
		if !ok || hp.Current <= 0 {
			continue
		}
		targets = append(targets, e.EntityID)
	}
	return targets
}

// aliveCount returns how many combatants of the side are alive — present in
// the turn order with Health.Current > 0 (the "alive check" per the Story 001
// implementation notes).
func (m *CombatManager) aliveCount(side Side) int {
	n := 0
	for _, e := range m.turnOrder {
		if e.Side != side {
			continue
		}
		hp, ok := entity.GetComponent[entity.Health](m.store, e.EntityID)
		if !ok || hp.Current <= 0 {
			continue
		}
		n++
	}
	return n
}

func (m *CombatManager) emitCombatStarted() {
	m.bus.Emit(entity.Event{
		Type:   EventCombatStarted,
		Source: "combat",
		Data:   CombatStartedData{TurnOrder: append([]TurnEntry(nil), m.turnOrder...)},
	})
}

func (m *CombatManager) emitCombatEnded() {
	m.bus.Emit(entity.Event{
		Type:   EventCombatEnded,
		Source: "combat",
		Data:   CombatEndedData{Outcome: m.outcome},
	})
}

func (m *CombatManager) emitTurnStarted(entry TurnEntry) {
	m.bus.Emit(entity.Event{
		Type:   EventTurnStarted,
		Source: "combat",
		Data: TurnStartedData{
			Combatant: entry.EntityID,
			Side:      entry.Side,
			Round:     m.round,
		},
	})
}

// emitTurnEnded emits TurnEnded for the active turn's combatant (m.current —
// the combatant whose turn is active, even when it was removed mid-turn).
// Callers must ensure a turn is active (turnActive).
func (m *CombatManager) emitTurnEnded() {
	m.bus.Emit(entity.Event{
		Type:   EventTurnEnded,
		Source: "combat",
		Data: TurnEndedData{
			Combatant: m.current,
			Round:     m.round,
		},
	})
}

func (m *CombatManager) emitCombatantRemoved(id entity.GameEntityID, reason RemovalReason) {
	m.bus.Emit(entity.Event{
		Type:   EventCombatantRemoved,
		Source: "combat",
		Data: CombatantRemovedData{
			EntityID: id,
			Reason:   reason,
		},
	})
}

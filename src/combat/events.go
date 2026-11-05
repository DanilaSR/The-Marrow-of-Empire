package combat

import "marrow-of-empire/entity"

// Combat event types (ADR-0011 combat events: started, ended, turn changed,
// entity died). The combat system is the sole producer of the lifecycle
// events; the input/UI layer produces EventAbilitySelected — the player's
// action-bar choice, consumed by targeting and audio. Subscribers — camera
// (Combat/Exploration states), game clock (pause/resume), combat UI (turn
// order display), Persistent World State — react without coupling. Events
// carry Timestamp 0: the game clock is frozen during combat (ADR-0007),
// matching the entity framework's convention.
const (
	// EventCombatStarted fires when an encounter begins (AC-1). Payload:
	// CombatStartedData — carries the fixed turn order for display.
	EventCombatStarted entity.EventType = "combat.started"
	// EventCombatEnded fires when the encounter resolves (AC-4). Payload:
	// CombatEndedData — carries the outcome (Victory/Defeat/Scripted);
	// Story 006 consumes it for XP, loot, and post-combat stabilization.
	EventCombatEnded entity.EventType = "combat.ended"
	// EventTurnStarted fires when a combatant's turn begins (AC-1). Payload:
	// TurnStartedData.
	EventTurnStarted entity.EventType = "combat.turn_started"
	// EventTurnEnded fires when a combatant's turn ends (AC-3). Payload:
	// TurnEndedData.
	EventTurnEnded entity.EventType = "combat.turn_ended"
	// EventCombatantRemoved fires when a combatant leaves the encounter —
	// defeated, fled, or surrendered (AC-4, AC-7; ADR-0011's "entity died"
	// combat event). Payload: CombatantRemovedData. Turn-order consumers
	// (combat UI) and the world state (fleeing enemy becomes neutral in
	// exploration, AC-7) react to it.
	EventCombatantRemoved entity.EventType = "combat.combatant_removed"
	// EventCharacterDying fires when a party member enters the Dying state at
	// 0 HP (Story 005, TR-combat-004, GDD CR-5) — Full Vision permadeath mode
	// only. Payload: CharacterDyingData — carries the rounds remaining for the
	// countdown UI. The character cannot act and must be stabilized by an
	// adjacent ally within the countdown or dies permanently.
	EventCharacterDying entity.EventType = "combat.character_dying"
	// EventCharacterKnockedOut fires when a companion enters the Knocked Out
	// state at 0 HP (Story 005, GDD CR-5) — VS mode (companion_death_enabled
	// false) only. Payload: CharacterKnockedOutData. No countdown runs; the
	// companion revives with 1 HP when combat ends.
	EventCharacterKnockedOut entity.EventType = "combat.character_knocked_out"
	// EventCharacterStabilized fires when an adjacent ally stabilizes a dying
	// character (Story 005, GDD CR-5): the countdown pauses and the character
	// survives. Payload: CharacterStabilizedData.
	EventCharacterStabilized entity.EventType = "combat.character_stabilized"
	// EventCharacterDied fires when a party member dies permanently — the
	// dying countdown elapsed without stabilization (Story 005, GDD CR-5) or
	// the player character reached 0 HP (instant game over, Story 005 item 6).
	// Payload: CharacterDiedData. The Party System applies permadeath on this
	// event; the player character's death resolves the encounter with Defeat.
	EventCharacterDied entity.EventType = "combat.character_died"
	// EventEnemyDefeated fires when an enemy reaches 0 HP (Story 005, GDD
	// CR-5: "Defeated (enemy) — enemy is dead"). Payload: EnemyDefeatedData.
	// Enemies die instantly — no dying state, no countdown.
	EventEnemyDefeated entity.EventType = "combat.enemy_defeated"
	// EventAbilitySelected fires when the player selects an ability from
	// the combat HUD's action bar (Combat UI epic Story 002, AC-4). The
	// HUD — the input/UI layer, not the combat system — is the producer;
	// consumers are the targeting system (Combat UI Story 005) and the
	// audio system (AUDIO-004: soft select sound). Selecting the same
	// ability again deselects it; no event fires for a deselection.
	EventAbilitySelected entity.EventType = "combat.ability_selected"
)

// CombatStartedData is the payload of EventCombatStarted. It is a value type
// so the event avoids heap allocation beyond the Event.Data interface boxing
// (ADR-0011).
type CombatStartedData struct {
	// TurnOrder is the fixed initiative order for the encounter (AC-2).
	TurnOrder []TurnEntry
}

// CombatEndedData is the payload of EventCombatEnded.
type CombatEndedData struct {
	// Outcome is how the encounter resolved: OutcomeVictory (all enemies
	// removed), OutcomeDefeat (all player characters removed), or
	// OutcomeScripted (external EndCombat).
	Outcome Outcome
}

// TurnStartedData is the payload of EventTurnStarted.
type TurnStartedData struct {
	// Combatant is the entity whose turn begins.
	Combatant entity.GameEntityID
	// Side is the combatant's side; it selects the turn state (PlayerTurn,
	// EnemyTurn, AllyTurn).
	Side Side
	// Round is the round number, 1-based, incremented when the turn order
	// wraps.
	Round int
}

// TurnEndedData is the payload of EventTurnEnded.
type TurnEndedData struct {
	// Combatant is the entity whose turn ended.
	Combatant entity.GameEntityID
	// Round is the round the turn ended in.
	Round int
}

// CombatantRemovedData is the payload of EventCombatantRemoved.
type CombatantRemovedData struct {
	// EntityID is the combatant that left the encounter.
	EntityID entity.GameEntityID
	// Reason is why the combatant left: defeated, fled, or surrendered.
	Reason RemovalReason
}

// CharacterDyingData is the payload of EventCharacterDying (Story 005).
type CharacterDyingData struct {
	// EntityID is the character entering the Dying state.
	EntityID entity.GameEntityID
	// RoundsRemaining is the countdown at entry: DyingRounds (3 by default).
	// It decrements at the end of each of the dying character's turns and
	// permanent death follows when it reaches 0.
	RoundsRemaining int
}

// CharacterKnockedOutData is the payload of EventCharacterKnockedOut
// (Story 005).
type CharacterKnockedOutData struct {
	// EntityID is the companion entering the Knocked Out state.
	EntityID entity.GameEntityID
}

// CharacterStabilizedData is the payload of EventCharacterStabilized
// (Story 005).
type CharacterStabilizedData struct {
	// EntityID is the stabilized character.
	EntityID entity.GameEntityID
	// HealerID is the adjacent ally that performed the stabilization.
	HealerID entity.GameEntityID
	// RoundsRemaining is the paused countdown value (>= 1): the countdown
	// stops; the character survives.
	RoundsRemaining int
}

// CharacterDiedData is the payload of EventCharacterDied (Story 005).
type CharacterDiedData struct {
	// EntityID is the character that died permanently.
	EntityID entity.GameEntityID
}

// EnemyDefeatedData is the payload of EventEnemyDefeated (Story 005).
type EnemyDefeatedData struct {
	// EntityID is the defeated enemy.
	EntityID entity.GameEntityID
}

// AbilitySelectedData is the payload of EventAbilitySelected (Combat UI
// epic Story 002, AC-4).
type AbilitySelectedData struct {
	// Combatant is the entity whose turn is active — the player-side
	// combatant that owns the selection.
	Combatant entity.GameEntityID
	// Slot is the action bar position, 1-6: Move, Attack, Abilities,
	// Items, Defend, Wait.
	Slot int
}

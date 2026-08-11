package stats

import "marrow-of-empire/entity"

// EventStatLevelUp fires when an XP award causes at least one level-up
// (STAT-005 AC-4). The Character Stats system is the sole producer (via
// AwardXPAndNotify); the payload is LevelUpEventData carrying the entity ID
// and the new derived values so subscribers update without re-querying the
// component.
const EventStatLevelUp entity.EventType = "stats.level_up"

// LevelUpEventData is the payload of EventStatLevelUp. It is a value type so
// the event avoids heap allocation beyond the Event.Data interface boxing
// (ADR-0011). The derived values are computed after the award, so they are
// final: MaxHealth and AP follow the level change.
type LevelUpEventData struct {
	// EntityID is the character entity that leveled up.
	EntityID entity.GameEntityID
	// NewLevel is the character's level after the award.
	NewLevel int
	// NewMaxHealth is the character's MaxHealth after the award:
	// 20 + CON×2 + Level×CON_mod (TR-stats-003).
	NewMaxHealth int
	// NewAP is the character's action points after the award:
	// 4 + DEX_mod, minimum 2 (TR-stats-003).
	NewAP int
}

// AwardXPAndNotify applies an XP award via AwardXP and, when the award causes
// at least one level-up, emits EventStatLevelUp with the entity ID and the
// new derived stat values (STAT-005 AC-4, ADR-0011: synchronous dispatch,
// priority ordering).
//
// The wrapper exists because AwardXP is pure math on the component value: it
// has no entity ID and no EventBus, and keeping it engine-free preserves the
// package's unit-testability without the engine running. Components are
// stored by value in the Entity Framework, so c must be the caller's
// component copy: read the component, mutate the copy through this function,
// and write it back with entity.SetComponent:
//
//	c, ok := entity.GetComponent[stats.StatsComponent](store, id)
//	if !ok { /* entity has no stats */ }
//	if stats.AwardXPAndNotify(bus, id, &c, 300) > 0 {
//	    entity.SetComponent(store, id, c)
//	}
//
// A multi-level award emits a single aggregate event carrying the final
// values (AwardXP returns the total levels gained; the payload holds the
// resulting state, not per-level increments). Returns the number of levels
// gained — AwardXP's return value; 0 when xp ≤ 0 or the character is at
// LevelCap, in which case no event is emitted. c must be non-nil.
func AwardXPAndNotify(bus entity.EventBus, id entity.GameEntityID, c *StatsComponent, xp int) int {
	gained := c.AwardXP(xp)
	if gained > 0 {
		bus.Emit(entity.Event{
			Type:   EventStatLevelUp,
			Source: "stats",
			Data: LevelUpEventData{
				EntityID:     id,
				NewLevel:     c.Level,
				NewMaxHealth: c.MaxHealth(),
				NewAP:        c.AP(),
			},
		})
	}
	return gained
}

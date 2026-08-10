// Package entity implements the game-layer entity framework (Story EF-001):
// GameEntityID mapping to Kaiju engine entities, the lifecycle state machine
// (Pending -> Active -> Suspended -> Destroyed), and lifecycle event broadcast
// through a synchronous EventBus with priority ordering.
//
// Architecture: ADR-0001 (Entity Architecture) and ADR-0011 (Event Bus).
// The game layer wraps Kaiju entities — every game entity has both a
// GameEntityID (uint64, monotonic, never reused) and a Kaiju EntityId
// (string, format "game_{GameEntityID}"). This package never defines a
// parallel entity system and never reuses a retired GameEntityID.
package entity

import "kaijuengine.com/matrix"

// GameEntityID is the universal game-layer entity reference. It is generated
// from a monotonic counter, is stable across save/load, and is never reused
// after an entity is destroyed (ADR-0001).
//
// Example usage:
//
//	id, err := store.CreateEntity(EntityTypeNPC, "Guard", matrix.NewVec3(10, 20, 0))
//	if err != nil { /* handle */ }
//	// id maps to the Kaiju EntityId "game_1"
type GameEntityID uint64

// EntityState is the lifecycle state machine of a game entity (ADR-0001).
//
//	Pending   — pre-activation assembly (entity created, not yet in world)
//	Active    — in world, updating
//	Suspended — off-screen (beyond suspension_distance), position preserved
//	Destroyed — removed, ID retired, never reused
type EntityState int

const (
	// EntityStatePending is the pre-activation assembly state. It is not
	// externally observable: CreateEntity transitions to Active atomically.
	EntityStatePending EntityState = iota
	// EntityStateActive means the entity is in the world and updating.
	EntityStateActive
	// EntityStateSuspended means the entity is beyond suspension_distance:
	// hidden from update loops, component data preserved, still visible to
	// spatial queries.
	EntityStateSuspended
	// EntityStateDestroyed means the entity is removed. The GameEntityID is
	// retired and never reused. Destroyed -> any transition is rejected.
	EntityStateDestroyed
)

// String returns the state name for logging and test output.
func (s EntityState) String() string {
	switch s {
	case EntityStatePending:
		return "Pending"
	case EntityStateActive:
		return "Active"
	case EntityStateSuspended:
		return "Suspended"
	case EntityStateDestroyed:
		return "Destroyed"
	default:
		return "Unknown"
	}
}

// EntityType categorizes game entities (Story EF-001).
type EntityType int

const (
	// EntityTypeUnknown is the zero value; a valid type is always passed to
	// CreateEntity.
	EntityTypeUnknown EntityType = iota
	// EntityTypeCharacter is a playable or companion character.
	EntityTypeCharacter
	// EntityTypeNPC is a non-player character.
	EntityTypeNPC
	// EntityTypeItem is a portable item.
	EntityTypeItem
	// EntityTypeProp is a static world object.
	EntityTypeProp
	// EntityTypeWildlife is a non-sapient creature.
	EntityTypeWildlife
)

// String returns the type name for logging and test output.
func (t EntityType) String() string {
	switch t {
	case EntityTypeCharacter:
		return "Character"
	case EntityTypeNPC:
		return "NPC"
	case EntityTypeItem:
		return "Item"
	case EntityTypeProp:
		return "Prop"
	case EntityTypeWildlife:
		return "Wildlife"
	default:
		return "Unknown"
	}
}

// EntityEventData is the payload attached to entity lifecycle events
// (entity.created, entity.destroyed, entity.suspended, entity.reactivated).
// It is a value type so hot-path lifecycle events avoid heap allocation
// beyond the interface boxing in Event.Data (ADR-0011).
type EntityEventData struct {
	// EntityID is the GameEntityID the event concerns.
	EntityID GameEntityID
	// EntityType is the category of the entity.
	EntityType EntityType
	// DisplayName is the human-readable name supplied at creation.
	DisplayName string
	// Position is the entity position at event time.
	Position matrix.Vec3
}

// ComponentEventData is the payload attached to component lifecycle events
// (component.added, component.removed, component.changed, Story EF-002).
// It is a value type so component mutation events avoid heap allocation
// beyond the interface boxing in Event.Data (ADR-0011).
type ComponentEventData struct {
	// EntityID is the GameEntityID the component belongs to.
	EntityID GameEntityID
	// ComponentType is the component type name (e.g. "Health").
	ComponentType string
	// OldValue is the previous component value; nil for component.added.
	OldValue any
	// NewValue is the new component value; nil for component.removed.
	NewValue any
}

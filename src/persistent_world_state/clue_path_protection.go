// Clue-path protection implementation for Story 004 of the Persistent World
// State epic: when a quest-relevant entity is destroyed, PWS records a
// clue_path_protection event and flags the entity as
// world.<id>.destroyed_with_clues = true so the Investigation Journal can
// surface the closed clue path. No clue path permanently closes without an
// authored discovery path.
//
// Story: production/epics/persistent-world-state/
//
//	story-004-clue-path-protection.md
//
// GDD:  design/gdd/persistent-world-state.md (Clue-path protection — EF CR-4
//
//	requires PWS to receive a clue_path_protection event on destruction
//	of any quest-relevant entity; the flag embeds the entity's numeric
//	ID and stays queryable after destruction)
//
// TR:   TR-pws-004 (clue-path protection: quest-relevant entity destruction →
//
//	PWS logs event, flags entity as destroyed_with_clues)
//
// ADR:  ADR-0001 (Entity Architecture — GameEntityIDs are never reused, so
//
//	the destroyed entity's log and flags remain queryable by ID),
//	ADR-0011 (Event Bus — synchronous dispatch, priority table: PWS
//	subscribes to entity lifecycle at priority 2, before Trust/Thread/
//	UI consumers), ADR-0013 (Memory Log — protected entries are never
//	evicted; quest_relevant is a protected category)
//
// Two implementation facts shape this story (deviations from the story's
// Implementation Notes, decided at implementation time):
//
//  1. Quest-relevance cannot be read from the destroyed entity's Persistent
//     component: EntityStoreImpl.retire() purges every component store
//     BEFORE entity.destroyed is emitted. PWS therefore owns a
//     quest-relevance registry (MarkQuestRelevant/UnmarkQuestRelevant/
//     IsQuestRelevant) that content authoring populates at scene/region
//     load.
//  2. WorldFlagStore writes flags into the Persistent component, which is
//     purged at destruction. The store therefore keeps PWS-owned tombstone
//     storage for destroyed entities' world.* flags (NoteEntityDestroyed),
//     and RecordDestructionEvent is the pipeline path that permits a
//     destroyed target (SubmitWorldEvent would reject it with
//     ErrInvalidTarget).
//
// Engine-agnostic: this package imports only the game-layer entity package
// (GameEntityID, EventBus, Event, EntityEventData, EventEntityDestroyed) and
// never touches Kaiju engine APIs. The event bus and pipeline are injected.
//
// Concurrency: not safe for concurrent use — the game loop is
// single-threaded and drives PWS from the main thread.
package persistent_world_state

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"marrow-of-empire/entity"
)

// CluePathProtectorPriority is the EventBus priority the protector subscribes
// with for entity.destroyed (ADR-0011 system priority table: 1 = Entity
// Framework, 2 = Persistent World State, 3 = Trust & Relationship, 4 = Thread
// Investigation, 5 = Combat/Dialogue UI). PWS records the destruction before
// downstream systems react to it.
const CluePathProtectorPriority = 2

// EventTypeCluePathProtection is the memory log event type for clue-path
// protection records (GDD: the EF CR-4 `clue_path_protection` event).
const EventTypeCluePathProtection = "clue_path_protection"

// DestroyedWithCluesFlag returns the world.* flag name that marks a destroyed
// quest-relevant entity (GDD: `world.[entity_id].destroyed_with_clues = true`,
// the Investigation Journal's discovery hook for a closed clue path). The
// entity's numeric GameEntityID is embedded in the name — IDs are never
// reused (ADR-0001), so the flag is unambiguous and stable.
//
// Example usage:
//
//	flag := persistent_world_state.DestroyedWithCluesFlag(id) // "world.12.destroyed_with_clues"
func DestroyedWithCluesFlag(id entity.GameEntityID) string {
	return fmt.Sprintf("world.%d.destroyed_with_clues", id)
}

// CluePathProtector implements Story 004: it watches entity.destroyed events
// and, for entities marked quest-relevant, records the destruction through
// the event pipeline (EventType clue_path_protection, flag
// world.<id>.destroyed_with_clues = true, protected memory entry) so the
// Investigation Journal can surface the closed clue path (AC-1..AC-4).
//
// Quest relevance is PWS-owned: content authoring (scene/region load) marks
// entities via MarkQuestRelevant — the destroyed entity's Persistent
// component is purged before entity.destroyed fires, so it cannot be read at
// handler time. Region authoring pairs with the mark: SetEntityRegion on the
// WorldFlagStore records the region the destruction record needs (the
// destroyed-event payload carries no region, only a position).
//
// The questRelevant and protected registries are in-memory only:
// quest-relevance is re-established at scene load, and the Entity Framework
// emits entity.destroyed exactly once, so neither needs serialization
// (Story 006 serializes the tombstone flags instead).
//
// Start subscribes on the EventBus; Stop unsubscribes. Not safe for
// concurrent use — drive it from the game loop thread.
type CluePathProtector struct {
	// pipeline is the event pipeline the protector records destruction
	// through; it carries the flag store, memory log registry, and clock.
	pipeline *EventPipeline
	// questRelevant is the PWS-owned quest-relevance registry: content
	// authoring marks entities at scene/region load.
	questRelevant map[entity.GameEntityID]bool
	// protected guards idempotency: entities whose clue-path protection has
	// already been recorded (a second destruction event is rejected).
	protected map[entity.GameEntityID]bool
	// bus and handler are the live subscription bookkeeping (Start/Stop).
	bus     entity.EventBus
	handler entity.EventHandler
	started bool
}

// NewCluePathProtector creates a CluePathProtector wired to the given event
// pipeline (dependency injection — no singletons). pipeline must be non-nil;
// it carries the flag store, memory log registry, and game clock the
// protector records through. The protector is inert until Start subscribes
// it on an event bus.
//
// Example usage:
//
//	protector := persistent_world_state.NewCluePathProtector(pipe)
//	protector.Start(bus)
//	defer protector.Stop()
func NewCluePathProtector(pipeline *EventPipeline) *CluePathProtector {
	return &CluePathProtector{
		pipeline:      pipeline,
		questRelevant: make(map[entity.GameEntityID]bool),
		protected:     make(map[entity.GameEntityID]bool),
	}
}

// MarkQuestRelevant marks the entity as quest-relevant: when it is destroyed,
// PWS records clue-path protection for it (AC-1). Content authoring calls
// this at scene/region load, paired with WorldFlagStore.SetEntityRegion so
// the destruction record carries the region. Idempotent. Entities never
// marked (or unmarked) are observed but not acted upon at destruction
// (AC-2).
func (c *CluePathProtector) MarkQuestRelevant(id entity.GameEntityID) {
	c.questRelevant[id] = true
}

// UnmarkQuestRelevant removes the quest-relevant mark (QA AC-2 edge case:
// "quest_relevant flag removed before destruction"). An unmarked entity's
// destruction produces no clue-path protection.
func (c *CluePathProtector) UnmarkQuestRelevant(id entity.GameEntityID) {
	delete(c.questRelevant, id)
}

// IsQuestRelevant reports whether the entity is currently marked
// quest-relevant.
func (c *CluePathProtector) IsQuestRelevant(id entity.GameEntityID) bool {
	return c.questRelevant[id]
}

// Start subscribes the protector to entity.destroyed at
// CluePathProtectorPriority (2 — the ADR-0011 priority table places PWS
// before Trust, Thread Investigation, and UI consumers of the lifecycle
// event). Calling Start twice without an intervening Stop is a no-op with a
// warning; a nil bus is warned and not started. In normal game wiring the
// bus is the same instance the pipeline broadcasts OnWorldStateChanged on,
// so the destruction record reaches the same subscribers.
func (c *CluePathProtector) Start(bus entity.EventBus) {
	if c.started {
		slog.Warn("clue path protector: already started; Start ignored")
		return
	}
	if bus == nil {
		slog.Warn("clue path protector: nil event bus; not started")
		return
	}
	c.bus = bus
	c.handler = c.onEntityDestroyed
	bus.Subscribe(entity.EventEntityDestroyed, CluePathProtectorPriority, c.handler)
	c.started = true
}

// Stop unsubscribes the protector from entity.destroyed so its registration
// never outlives its owner. No-op when not started. The protector may be
// restarted with Start afterwards.
func (c *CluePathProtector) Stop() {
	if !c.started {
		return
	}
	c.bus.Unsubscribe(entity.EventEntityDestroyed, c.handler)
	c.bus = nil
	c.handler = nil
	c.started = false
}

// onEntityDestroyed is the entity.destroyed handler (AC-1, AC-2). It runs
// synchronously at priority 2, before Trust/Thread/UI consumers (ADR-0011).
//
//   - Non-quest-relevant entities (never marked, or unmarked before
//     destruction) are observed but not acted upon: no flag, no log entry,
//     no broadcast (AC-2).
//   - Quest-relevant entities are recorded via RecordDestructionEvent: the
//     memory log gains a protected clue_path_protection entry (which entity,
//     the region from the flag store's registry, the frozen game timestamp,
//     ActorID 0 — the destroyed-event payload carries no destroyer, QA:
//     "or 0 for environmental") and the flag
//     world.<id>.destroyed_with_clues = true lands in tombstone storage so
//     GetWorldFlag keeps working after the component purge (AC-1, AC-3,
//     AC-4).
//   - A duplicate destruction event for an already-protected entity is
//     rejected with a warning (no double log): the Entity Framework emits
//     entity.destroyed exactly once per entity, but the guard keeps the
//     record idempotent (QA AC-4 edge case).
func (c *CluePathProtector) onEntityDestroyed(ev entity.Event) {
	data, ok := ev.Data.(entity.EntityEventData)
	if !ok {
		slog.Warn("clue path protection: entity.destroyed payload is not EntityEventData; ignored",
			"source", ev.Source)
		return
	}
	if !c.questRelevant[data.EntityID] {
		return // AC-2: observed, not acted upon.
	}
	if c.protected[data.EntityID] {
		slog.Warn("clue path protection: entity already protected; duplicate destruction event rejected",
			"entity", data.EntityID)
		return
	}
	// RegionID is left empty: RecordDestructionEvent resolves it from the
	// flag store's entity->region registry (the destroyed-event payload
	// carries no region). The flag mutation marks the entity's record
	// destroyed_with_clues — routed through the flag store's tombstone
	// storage, since the component is already purged (AC-1, AC-3).
	event := WorldEvent{
		EventType:      EventTypeCluePathProtection,
		TargetEntityID: data.EntityID,
		FlagMutations:  map[string]any{DestroyedWithCluesFlag(data.EntityID): true},
		Details:        destructionDetails(data),
	}
	if err := c.pipeline.RecordDestructionEvent(event); err != nil {
		slog.Warn("clue path protection: failed to record destruction event",
			"entity", data.EntityID, "error", err)
		return
	}
	c.protected[data.EntityID] = true
}

// destructionDetails builds the clue_path_protection Details JSON blob: the
// destroyed entity's display name and type. The Entity Framework deletes
// display names at destruction, so the memory entry is the only place the
// name survives for the Investigation Journal to display.
func destructionDetails(data entity.EntityEventData) string {
	raw, err := json.Marshal(struct {
		DisplayName string `json:"display_name"`
		EntityType  string `json:"entity_type"`
	}{
		DisplayName: data.DisplayName,
		EntityType:  data.EntityType.String(),
	})
	if err != nil {
		return ""
	}
	return string(raw)
}

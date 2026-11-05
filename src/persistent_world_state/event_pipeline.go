// Event submission pipeline for Story 002 of the Persistent World State
// epic: the validate -> mutate -> log -> broadcast pipeline every system
// uses to change world state. Systems submit WorldEvents here instead of
// writing world.* flags directly — PWS is the authoritative writer of the
// namespace (ADR-0001 write-ownership table).
//
// Story 004 (clue-path protection) adds RecordDestructionEvent, the
// destruction-time path for entities whose Persistent component is already
// purged: the target's flags route through the flag store's tombstone
// storage and the memory entry is marked protected. SubmitWorldEvent is
// unchanged for live entities.
//
// Story: production/epics/persistent-world-state/
//
//	story-002-event-submission-pipeline.md
//
// GDD:  design/gdd/persistent-world-state.md (Event Processing Pipeline,
//
//	requirement TR-pws-005)
//
// ADR:  ADR-0001 (Entity Architecture — PWS owns the world.* namespace;
//
//	within PWS, submission order resolves same-flag conflicts, last
//	writer wins — no per-event priority), ADR-0011 (Event Bus —
//	synchronous broadcast of OnWorldStateChanged), ADR-0013 (Memory
//	Log — append target)
//
// Pipeline stages, executed synchronously in one call:
//
//  1. Validate: event type and region non-empty, target exists, actor
//     exists (a zero actor is valid — system-generated events). Any
//     validation failure returns immediately with an error: no flags are
//     mutated, no log entry is appended, nothing is broadcast (AC-4).
//  2. Mutate: each FlagMutations entry is applied to the target's world.*
//     flags through WorldFlagStore.SetWorldFlag. Flags outside the world.*
//     namespace and flags whose value fails the store's validation
//     (schema mismatch, unsupported type) are skipped with a warning — the
//     filter is per-flag, not per-event (AC-2). Flag names are applied in
//     sorted order so mutation order is deterministic regardless of Go map
//     iteration order (QA: "10+ mutations all applied in order").
//  3. Log: a MemoryEntry is appended to the target's MemoryLog (ADR-0013).
//     Entity scoping of logs is this story's job (memory_log.go); the
//     pipeline owns the per-entity log registry.
//  4. Broadcast: OnWorldStateChanged is emitted synchronously on the
//     EventBus with the submitted event as payload (AC-5, ADR-0011).
//
// Same-frame ordering (AC-3): events are processed in SubmitWorldEvent call
// order. Conflicting writes to the same flag resolve last-writer-wins by
// submission order — PWS is the sole writer of world.* flags, so the Entity
// Framework's cross-system write arbitration does not apply inside the
// pipeline.
//
// Pure data pipeline, engine-agnostic: this package imports only the
// game-layer entity package (GameEntityID, Persistent, EventBus) and never
// touches Kaiju engine APIs. Component store access, the flag store, the
// event bus, and the game clock are injected (dependency injection, no
// singletons).
//
// Concurrency: the pipeline is not safe for concurrent use. The game loop
// is single-threaded and systems drive PWS from the main thread (GDD:
// events are processed in submission order, same-frame).
package persistent_world_state

import (
	"errors"
	"log/slog"
	"sort"

	"marrow-of-empire/entity"
)

// OnWorldStateChanged is the EventBus event type emitted after every
// successfully processed world event (ADR-0011, AC-5). Consumers subscribe
// with the priority their reaction order requires per ADR-0011's system
// priority table (UI consumers register last). The payload is the submitted
// WorldEvent.
const OnWorldStateChanged entity.EventType = "world.state_changed"

// Sentinel errors returned by EventPipeline.SubmitWorldEvent. Callers may
// test with errors.Is. Validation is stage 1 of the pipeline: on any of
// these errors nothing is mutated, appended, or broadcast (AC-4).
var (
	// ErrInvalidEventType is returned when the event's EventType is empty.
	ErrInvalidEventType = errors.New("persistent_world_state: event type must be non-empty")

	// ErrInvalidRegionID is returned when the event's RegionID is empty —
	// including when RecordDestructionEvent could not resolve an empty
	// RegionID from the flag store's entity->region registry.
	ErrInvalidRegionID = errors.New("persistent_world_state: event region must be non-empty")

	// ErrInvalidTarget is returned when the event's TargetEntityID is zero
	// or names an entity that does not exist — destroyed or never created.
	// The Entity Framework removes components on destruction, so an absent
	// Persistent component means the entity is gone (AC-4).
	ErrInvalidTarget = errors.New("persistent_world_state: event target does not exist")

	// ErrInvalidActor is returned when the event's ActorEntityID is non-zero
	// and names an entity that does not exist. A zero actor is valid —
	// system-generated events (MemoryEntry.ActorID documents this).
	ErrInvalidActor = errors.New("persistent_world_state: event actor does not exist")
)

// WorldEvent is one world-state change submitted by any system (TR-pws-005,
// story implementation notes). It names the event kind, the acting and
// affected entities, the region, an opaque JSON Details blob, and the
// world.* flag mutations to apply to the target.
type WorldEvent struct {
	// EventType identifies the structured event (e.g. "location_cleared",
	// "kill"). Stored verbatim in the memory log entry. Must be non-empty.
	EventType string
	// ActorEntityID is the entity that caused the event; zero is valid for
	// system-generated events.
	ActorEntityID entity.GameEntityID
	// TargetEntityID is the entity the event affects: its world.* flags are
	// mutated and its memory log receives the entry. Must exist.
	TargetEntityID entity.GameEntityID
	// RegionID identifies the region where the event occurred. Must be
	// non-empty.
	RegionID string
	// Details is an event-specific JSON blob, opaque to the pipeline, parsed
	// by the consuming system (Thread Investigation, NPC dialogue).
	Details string
	// FlagMutations are the world.* flag writes to apply to the target.
	// Flags outside the world.* namespace are skipped with a warning (AC-2);
	// filtering is per-flag, never per-event.
	FlagMutations map[string]any
}

// EventPipelineConfig carries the event pipeline tuning knobs. Zero values
// fall back to the GDD/ADR defaults (mirrors WorldFlagStoreConfig; a later
// story wires these to external config files — gameplay values stay
// data-driven).
type EventPipelineConfig struct {
	// Clock returns the current game time in minutes (ADR-0007). When nil,
	// entries are stamped with 0 — matching the EventBus behavior until the
	// game clock ships (event_bus.go: "always 0 for now").
	Clock func() float64
	// LogMaxSize is the per-entity memory log soft cap (ADR-0013, default
	// DefaultMaxSize = 50).
	LogMaxSize int
}

// EventPipeline is the Persistent World State event submission pipeline
// (TR-pws-005): the entry point all systems use to change world state. It
// validates the event, applies world.* flag mutations to the target via the
// WorldFlagStore, appends a MemoryEntry to the target's MemoryLog, and
// broadcasts OnWorldStateChanged on the EventBus — synchronously, in
// submission order, same frame (AC-1..AC-5).
//
// The pipeline owns entity scoping of memory logs (memory_log.go delegates
// that to this story): logs are created lazily per entity on first event.
//
// Not safe for concurrent use — drive it from the game loop thread.
type EventPipeline struct {
	accessor   PersistentAccessor
	flags      *WorldFlagStore
	bus        entity.EventBus
	clock      func() float64
	logMaxSize int
	logs       map[entity.GameEntityID]*MemoryLog
}

// NewEventPipeline creates an event pipeline backed by the given component
// store accessor, world flag store, and event bus (dependency injection —
// no singletons). accessor, flags, and bus must be non-nil. Zero-valued
// config fields fall back to defaults: Clock stamps 0 (game clock not
// shipped yet, ADR-0007), LogMaxSize defaults to 50 (ADR-0013).
//
// Example usage:
//
//	pipe := persistent_world_state.NewEventPipeline(accessor, store, bus,
//	    persistent_world_state.EventPipelineConfig{})
func NewEventPipeline(accessor PersistentAccessor, flags *WorldFlagStore, bus entity.EventBus, config EventPipelineConfig) *EventPipeline {
	clock := config.Clock
	if clock == nil {
		clock = func() float64 { return 0 }
	}
	maxSize := config.LogMaxSize
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &EventPipeline{
		accessor:   accessor,
		flags:      flags,
		bus:        bus,
		clock:      clock,
		logMaxSize: maxSize,
		logs:       make(map[entity.GameEntityID]*MemoryLog),
	}
}

// SubmitWorldEvent is the entry point for all systems that change world
// state (AC-1..AC-5). It runs the pipeline synchronously:
//
//  1. Validate: EventType non-empty (ErrInvalidEventType), RegionID
//     non-empty (ErrInvalidRegionID), target exists (ErrInvalidTarget), and
//     actor exists when non-zero (ErrInvalidActor — zero is a
//     system-generated event). A validation failure returns immediately:
//     no flag is mutated, no log entry appended, no broadcast (AC-4).
//  2. Mutate: each FlagMutations entry is applied to the target's world.*
//     flags via WorldFlagStore.SetWorldFlag, in sorted flag-name order so
//     application is deterministic. Flags outside the world.* namespace and
//     flags whose value fails the store's validation (schema mismatch,
//     unsupported type) are skipped with a warning and never fail the event
//     (AC-2 — per-flag, not per-event).
//  3. Log: a MemoryEntry (NextEventID, current game timestamp via Clock,
//     the event fields) is appended to the target's MemoryLog (ADR-0013).
//  4. Broadcast: OnWorldStateChanged is emitted synchronously on the
//     EventBus with the submitted event as payload (AC-5, ADR-0011).
//
// Events are processed in call order within a frame (AC-3): the pipeline is
// synchronous, so submission order is processing order, and conflicting
// writes to the same flag resolve last-writer-wins.
//
// Returns nil on success; the sentinel validation errors above on failure.
func (p *EventPipeline) SubmitWorldEvent(event WorldEvent) error {
	// Stage 1 — validation. Fail fast: no partial processing (AC-4).
	if event.EventType == "" {
		return ErrInvalidEventType
	}
	if event.RegionID == "" {
		return ErrInvalidRegionID
	}
	if !p.entityExists(event.TargetEntityID) {
		return ErrInvalidTarget
	}
	if event.ActorEntityID != 0 && !p.entityExists(event.ActorEntityID) {
		return ErrInvalidActor
	}

	// Stage 2 — flag mutations, per-flag filtering (AC-2). Sorted names keep
	// application order deterministic.
	for _, flag := range sortedFlagNames(event.FlagMutations) {
		if !isWorldFlagName(flag) {
			slog.Warn("world event: flag outside the world.* namespace skipped",
				"event_type", event.EventType, "target", event.TargetEntityID, "flag", flag)
			continue
		}
		if err := p.flags.SetWorldFlag(event.TargetEntityID, flag, event.FlagMutations[flag]); err != nil {
			slog.Warn("world event: flag mutation skipped",
				"event_type", event.EventType, "target", event.TargetEntityID,
				"flag", flag, "error", err)
			continue
		}
	}

	// Stage 3 — memory log append (ADR-0013). The event ID comes from the
	// package-level monotonic counter; the timestamp from the injected clock.
	timestamp := p.clock()
	p.logFor(event.TargetEntityID).Append(MemoryEntry{
		EventID:       NextEventID(),
		EventType:     event.EventType,
		GameTimestamp: timestamp,
		ActorID:       event.ActorEntityID,
		TargetID:      event.TargetEntityID,
		RegionID:      event.RegionID,
		Details:       event.Details,
	})

	// Stage 4 — synchronous broadcast, same frame (ADR-0011, AC-5).
	p.bus.Emit(entity.Event{
		Type:      OnWorldStateChanged,
		Timestamp: timestamp,
		Source:    "persistent_world_state",
		Data:      event,
	})
	return nil
}

// RecordDestructionEvent is the destruction-time pipeline path (Story 004,
// clue-path protection). It records the destruction of an entity whose
// Persistent component is already gone: the Entity Framework purges component
// stores before entity.destroyed fires, so SubmitWorldEvent's existence
// validation (ErrInvalidTarget) can never pass for a destroyed target.
//
// The stages mirror SubmitWorldEvent with three differences:
//
//  1. Validation: EventType non-empty (ErrInvalidEventType), region known
//     (ErrInvalidRegionID — an empty RegionID is resolved from the flag
//     store's entity->region registry, SetEntityRegion/EntityRegion, before
//     the check), and a non-zero target (ErrInvalidTarget). There is NO
//     target-existence check: the target is destroyed by definition. The
//     actor is recorded as given without an existence check — the current
//     entity.destroyed payload carries no destroyer (EntityEventData), so
//     callers pass 0 (system-generated); a future payload extension may fill
//     it, and at destruction time the destroyer itself may already be
//     destroyed (mutual destruction), so the check would be wrong.
//  2. Mutate: the target is recorded as destroyed in the flag store
//     (NoteEntityDestroyed), which routes its world.* flag writes into
//     PWS-owned tombstone storage instead of the purged component (AC-3).
//  3. Log: the memory entry is marked Protected: true (ADR-0013 protected
//     category "quest_relevant") — a clue-path protection record is the
//     player's only discovery path; eviction would silently close it.
//
// Broadcasting is unchanged: OnWorldStateChanged is emitted synchronously
// with the event as payload (ADR-0011).
//
// Idempotency: a duplicate record for the same entity is NOT detected here —
// the CluePathProtector guards at the event level (the Entity Framework
// emits entity.destroyed exactly once per entity; the protector ignores
// repeats).
func (p *EventPipeline) RecordDestructionEvent(event WorldEvent) error {
	// Stage 1 — validation. Fail fast: no partial processing (AC-4).
	if event.EventType == "" {
		return ErrInvalidEventType
	}
	if event.RegionID == "" {
		// The destroyed-event payload carries no region; resolve from the
		// flag store's entity->region registry (SetEntityRegion).
		event.RegionID = p.flags.EntityRegion(event.TargetEntityID)
	}
	if event.RegionID == "" {
		return ErrInvalidRegionID
	}
	if event.TargetEntityID == 0 {
		return ErrInvalidTarget
	}

	// Stage 2 — tombstone routing: the destroyed entity's flags land in
	// PWS-owned tombstone storage (Story 004, AC-3). Per-flag filtering
	// mirrors SubmitWorldEvent (AC-2): non-world.* flags and flags that fail
	// the store's validation are skipped with a warning, never failing the
	// event.
	p.flags.NoteEntityDestroyed(event.TargetEntityID)
	for _, flag := range sortedFlagNames(event.FlagMutations) {
		if !isWorldFlagName(flag) {
			slog.Warn("world event: flag outside the world.* namespace skipped",
				"event_type", event.EventType, "target", event.TargetEntityID, "flag", flag)
			continue
		}
		if err := p.flags.SetWorldFlag(event.TargetEntityID, flag, event.FlagMutations[flag]); err != nil {
			slog.Warn("world event: flag mutation skipped",
				"event_type", event.EventType, "target", event.TargetEntityID,
				"flag", flag, "error", err)
			continue
		}
	}

	// Stage 3 — memory log append (ADR-0013), protected: the record is the
	// only discovery path for the closed clue path and must never be evicted
	// by the soft cap.
	timestamp := p.clock()
	p.logFor(event.TargetEntityID).Append(MemoryEntry{
		EventID:       NextEventID(),
		EventType:     event.EventType,
		GameTimestamp: timestamp,
		ActorID:       event.ActorEntityID,
		TargetID:      event.TargetEntityID,
		RegionID:      event.RegionID,
		Details:       event.Details,
		Protected:     true,
	})

	// Stage 4 — synchronous broadcast, same frame (ADR-0011, AC-5).
	p.bus.Emit(entity.Event{
		Type:      OnWorldStateChanged,
		Timestamp: timestamp,
		Source:    "persistent_world_state",
		Data:      event,
	})
	return nil
}

// MemoryLogFor returns the append-only memory log of an entity, creating it
// lazily on first access (the pipeline owns entity scoping of logs,
// memory_log.go). Callers read entries via MemoryLog.GetEntries or
// MemoryLog.Entries; they must not append — SubmitWorldEvent and
// RecordDestructionEvent are the only append paths.
func (p *EventPipeline) MemoryLogFor(id entity.GameEntityID) *MemoryLog {
	return p.logFor(id)
}

// entityExists reports whether the entity exists, through the component
// store: an entity exists iff it has a Persistent component. The Entity
// Framework removes components on destruction, so an absent component means
// the entity is nonexistent or destroyed (AC-4).
func (p *EventPipeline) entityExists(id entity.GameEntityID) bool {
	_, ok := p.accessor.GetPersistent(id)
	return ok
}

// logFor returns the entity's memory log, creating it with the configured
// soft cap on first access.
func (p *EventPipeline) logFor(id entity.GameEntityID) *MemoryLog {
	log, ok := p.logs[id]
	if !ok {
		log = NewMemoryLog(p.logMaxSize)
		p.logs[id] = log
	}
	return log
}

// sortedFlagNames returns the event's flag names in ascending order so
// mutations apply deterministically regardless of Go map iteration order
// (QA: "10+ mutations all applied in order").
func sortedFlagNames(mutations map[string]any) []string {
	names := make([]string, 0, len(mutations))
	for name := range mutations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

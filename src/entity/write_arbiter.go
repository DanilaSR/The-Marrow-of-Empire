// Write arbitration for Story EF-004 (TR-ef-005, ADR-0001).
//
// WriteArbiter resolves same-frame modifications of the same component field
// by different systems. Each system identifies itself by name and a priority
// value (1 = highest). On conflict the lower priority number wins; equal
// priority falls back to last writer wins (deterministic, no warning).
// Uncontested writes apply with zero arbitration overhead: one map lookup
// miss, a direct store write, no staging and no commit phase.
package entity

import (
	"errors"
	"fmt"
	"log/slog"
)

// ErrWriteRejected is returned (wrapped) by SetComponentArbitrated when a
// write loses priority arbitration to a different system that already wrote
// the same field this frame. The discarded write is not applied. Callers can
// use errors.Is to distinguish arbitration rejection from store-level errors
// (destroyed or unknown entity).
//
// Example usage:
//
//	err := entity.SetComponentArbitrated(arbiter, "vfx", 15, id, hp, "Current")
//	if errors.Is(err, entity.ErrWriteRejected) { /* another system owns this field this frame */ }
var ErrWriteRejected = errors.New("entity: write rejected by write arbitration")

// WriteArbiter wraps an EntityStore and arbitrates component writes between
// systems (TR-ef-005, ADR-0001 "Write Arbitration" box: per-frame delta
// accumulation, system priority resolution).
//
// Systems call SetComponentArbitrated instead of SetComponent directly; the
// component store layer stays clean — it never learns about systems or
// priorities. Reads are unaffected: systems read components via
// GetComponent as usual (reads are unowned). The arbiter is not safe for
// concurrent use; call it from the game loop thread.
//
// Arbitration is per (entity, component type, field) slot and lasts one
// frame: BeginFrame clears the last-writer log, so a conflict only counts
// between systems writing the same field within one frame. Arbitration state
// is never persisted (EF-005: serialization is out of scope).
//
// Example usage:
//
//	arbiter := entity.NewWriteArbiter(store)
//	arbiter.BeginFrame() // frame start
//	err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 30, Max: 100}, "Current")
type WriteArbiter struct {
	// store is the wrapped EntityStore; all applied writes go through the
	// typed SetComponent helper, so the arbiter works with any EntityStore
	// implementation (ADR-0001: game code depends on the interface).
	store EntityStore
	// records tracks, per frame, the last writer of each arbitrated slot.
	// BeginFrame clears it; nothing survives across frames.
	records map[arbitrationKey]writeRecord
}

// arbitrationKey identifies one arbitrated write slot: an entity, a component
// type, and a field of that component. componentType is the compile-time type
// descriptor (*T)(nil) — the same zero-reflection keying used by the typed
// component stores (component_store.go) — so slot lookups never use
// reflection. A field of "" arbitrates the whole component as a single slot.
type arbitrationKey struct {
	entityID      GameEntityID
	componentType any
	field         string
}

// writeRecord is the per-frame last writer of one slot.
type writeRecord struct {
	systemName string
	priority   int
}

// NewWriteArbiter wraps store with write arbitration. Systems then call
// SetComponentArbitrated with this arbiter instead of SetComponent on the
// store.
func NewWriteArbiter(store EntityStore) *WriteArbiter {
	return &WriteArbiter{
		store:   store,
		records: make(map[arbitrationKey]writeRecord),
	}
}

// BeginFrame starts a new arbitration frame: the per-frame last-writer log is
// cleared, so a write that lost a conflict in the previous frame is
// uncontested again. It is a no-op if no writes happened since the last call
// (clearing an empty map).
func (a *WriteArbiter) BeginFrame() {
	clear(a.records)
}

// SetComponentArbitrated writes a component through arbitration (TR-ef-005).
// It is the system-facing replacement for SetComponent: pass the system name,
// its priority (1 = highest), and the name of the field this write mutates.
//
// Go does not allow generic methods, so — like GetComponent and SetComponent
// — this is a package-level generic function taking the arbiter
// (component_store.go documents the same constraint).
//
// Resolution per slot, on each call:
//
//   - No other system wrote this slot this frame: the write is applied
//     immediately to the wrapped store (AC-3, zero arbitration overhead) and
//     the slot records this system as last writer.
//   - The same system wrote the slot earlier this frame: the write is applied
//     and the record follows the latest write — a system may rewrite its own
//     field freely.
//   - A different system wrote the slot with a strictly higher priority
//     (lower number): the write is applied and replaces the record; the
//     superseded writer is logged with a warning.
//   - A different system wrote the slot with equal priority: the write is
//     applied and replaces the record. Last writer wins, no warning — equal
//     priority is not a conflict (AC-2, ADR-0001 deterministic tie-breaking).
//   - A different system wrote the slot with a strictly lower priority
//     (higher number): the write is discarded — a warning is logged and the
//     calling system receives an error wrapping ErrWriteRejected (AC-1).
//
// Because the wrapped store stores whole components, a write replaces the
// entire component value. When two systems write different fields of the same
// component in one frame, each must read-modify-write (include the other
// field's value in the component it passes) or the later write clobbers the
// earlier system's field; field-level merging is out of scope for EF-004.
//
// Priority values below 1 are rejected: 1 is the documented highest priority,
// so a lower value would silently outrank every system.
//
// Errors from the wrapped store (destroyed or unknown entity, unknown
// component type) propagate unchanged and nothing is recorded.
//
// Example usage:
//
//	arbiter.BeginFrame()
//	err := entity.SetComponentArbitrated(arbiter, "combat", 5, id, entity.Health{Current: 30, Max: 100}, "Current")
//	if errors.Is(err, entity.ErrWriteRejected) { /* combat lost the slot this frame */ }
func SetComponentArbitrated[T any](a *WriteArbiter, systemName string, priority int, id GameEntityID, component T, field string) error {
	if priority < 1 {
		return fmt.Errorf("entity: write arbitration: system %q has invalid priority %d (minimum 1)", systemName, priority)
	}
	key := arbitrationKey{entityID: id, componentType: (*T)(nil), field: field}
	rec, exists := a.records[key]
	if !exists {
		// Uncontested slot (AC-3): apply immediately. No comparison, no
		// staging, no commit phase, no reflection — just record the last
		// writer in case another system writes this slot later in the frame.
		if err := SetComponent(a.store, id, component); err != nil {
			return err
		}
		a.records[key] = writeRecord{systemName: systemName, priority: priority}
		return nil
	}
	if rec.systemName == systemName {
		// A system may rewrite its own field any number of times per frame.
		if err := SetComponent(a.store, id, component); err != nil {
			return err
		}
		a.records[key] = writeRecord{systemName: systemName, priority: priority}
		return nil
	}
	// A different system holds the slot. Before resolving the conflict,
	// reject destroyed and unknown entities with the store's own error: a
	// write to a dead entity is a store error, never an arbitration
	// rejection. The uncontested path needs no check — it reaches the store
	// directly and propagates the store error (keeping AC-3 free of extra
	// lookups).
	if st, ok := a.store.EntityState(id); !ok || st == EntityStateDestroyed {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	if priority > rec.priority {
		// The caller loses: higher priority numbers rank lower. Discarded
		// writes log a warning and the calling system receives an error
		// (QA AC-1: "VFX system receives warning").
		componentName := typeName[T]()
		slog.Warn("write arbitration: lower-priority write discarded",
			"loser", systemName, "loser_priority", priority,
			"winner", rec.systemName, "winner_priority", rec.priority,
			"entity", id, "component", componentName, "field", field)
		return fmt.Errorf("%w: %s (priority %d) wrote %s after %s (priority %d) in the same frame",
			ErrWriteRejected, systemName, priority, slotName(componentName, field), rec.systemName, rec.priority)
	}
	// The caller wins and becomes the last writer. Equal priority is
	// last-writer-wins with no warning (AC-2); only a strictly better
	// priority discards the previous writer with a warning.
	if err := SetComponent(a.store, id, component); err != nil {
		return err
	}
	if priority < rec.priority {
		componentName := typeName[T]()
		slog.Warn("write arbitration: lower-priority write discarded",
			"loser", rec.systemName, "loser_priority", rec.priority,
			"winner", systemName, "winner_priority", priority,
			"entity", id, "component", componentName, "field", field)
	}
	a.records[key] = writeRecord{systemName: systemName, priority: priority}
	return nil
}

// slotName renders a field for diagnostics; a whole-component slot (field "")
// renders as the component name alone.
func slotName(component, field string) string {
	if field == "" {
		return component
	}
	return component + "." + field
}

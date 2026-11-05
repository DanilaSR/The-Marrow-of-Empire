// Memory log implementation for Story 003 of the Persistent World State
// epic: the append-only structured event log with soft cap and protected
// entries.
//
// Story: production/epics/persistent-world-state/
//
//	story-003-memory-log.md
//
// GDD:  design/gdd/persistent-world-state.md (CR-3 Memory Log — append-only,
//
//	capped, protected entries)
//
// TR:   TR-pws-002 (memory log: append-only per entity, soft cap 50,
//
//	protected entries never evicted)
//
// ADR:  ADR-0013 (Persistent World State — Memory Log & Evolution Rules):
//
//	append-only structured memory log per entity; soft cap at 50 entries;
//	oldest unprotected entry evicted when the cap is exceeded; protected
//	entries (npc_state_changed, discovery_event, quest_relevant — marked
//	by the submitting system) never evicted; if all entries are protected
//	the cap is temporarily exceeded with a warning.
//
// Pure data structure, engine-agnostic: this package imports only the
// game-layer entity package for the GameEntityID type and never touches
// Kaiju engine APIs. Entity scoping (which entity a log belongs to) is the
// Persistent World State system's job (Story 002 event submission pipeline);
// this type manages a single entity's log only.
//
// Performance (ADR-0013): append <0.1ms, query <1ms at 50 entries.
//
// Concurrency: the log is not safe for concurrent use. The game loop is
// single-threaded and systems drive PWS from the main thread. The global
// event ID counter is atomic because it is shared across all entity logs.
package persistent_world_state

import (
	"log/slog"
	"sort"
	"sync/atomic"

	"marrow-of-empire/entity"
)

// DefaultMaxSize is the soft cap applied when a MemoryLog is created with a
// non-positive MaxSize (ADR-0013: soft cap 50 entries per entity).
const DefaultMaxSize = 50

// MemoryEntry is one structured memory record in a MemoryLog (GDD CR-3,
// ADR-0013). The log is append-only: entries are never modified after
// submission. Protected entries are never evicted by the cap; the submitting
// system marks protected events (npc_state_changed, discovery_event,
// quest_relevant).
//
// Details carries event-specific fields as a JSON blob — opaque to the log,
// parsed by the consuming system (Thread Investigation, NPC dialogue).
type MemoryEntry struct {
	// EventID is the globally unique, monotonic event identifier, assigned
	// from the package-level atomic counter (NextEventID). Never reused.
	EventID uint64
	// EventType identifies the structured event (e.g. "kill",
	// "trust_change", "discovery_event").
	EventType string
	// GameTimestamp is the game time in minutes when the event occurred.
	GameTimestamp float64
	// ActorID is the entity that caused the event; zero is valid for
	// system-generated events.
	ActorID entity.GameEntityID
	// TargetID is the entity the event affected; zero is valid for
	// system-generated events.
	TargetID entity.GameEntityID
	// RegionID identifies the region where the event occurred.
	RegionID string
	// Details is an event-specific JSON blob.
	Details string
	// Protected marks entries that must never be evicted by the cap
	// (npc_state_changed, discovery_event, quest_relevant).
	Protected bool
}

// MemoryLog is the append-only structured event log for one entity
// (ADR-0013). Entries are stored oldest first; Append always adds to the
// end. When the entry count exceeds MaxSize the oldest unprotected entry is
// evicted; if every entry is protected the cap is temporarily exceeded with
// a warning.
type MemoryLog struct {
	// Entries holds the log in oldest-first order.
	Entries []MemoryEntry
	// MaxSize is the soft cap. Non-positive values behave as
	// DefaultMaxSize (50).
	MaxSize int
}

// nextEventID is the package-level monotonic counter for globally unique
// event IDs (ADR-0013: "use an atomic counter nextEventID uint64 at the PWS
// system level"). Atomic because the counter is shared across all entity
// logs even though each log itself is single-threaded.
var nextEventID atomic.Uint64

// NextEventID returns the next globally unique, monotonic event ID and
// advances the counter. IDs are never reused and are stable across
// save/load (assigned at submission time, persisted with the entry).
//
// The Story 002 event submission pipeline assigns the ID via this function
// before building a MemoryEntry; callers may also supply their own EventID,
// in which case uniqueness is the caller's responsibility.
func NextEventID() uint64 {
	return nextEventID.Add(1)
}

// NewMemoryLog returns an empty MemoryLog with the given soft cap. A
// non-positive maxSize is replaced by DefaultMaxSize (50, per ADR-0013).
func NewMemoryLog(maxSize int) *MemoryLog {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &MemoryLog{MaxSize: maxSize}
}

// Append adds entry to the end of the log (oldest-first ordering preserved)
// and enforces the soft cap: if the entry count now exceeds MaxSize, the
// oldest entry with Protected == false is evicted and evicted is true. If
// every entry is protected, no eviction happens — the cap is temporarily
// exceeded and a warning is logged (evicted is false).
//
// The entry is stored as given: the caller is responsible for assigning a
// globally unique EventID (see NextEventID) and for marking Protected on
// events that must never be evicted (npc_state_changed, discovery_event,
// quest_relevant).
func (l *MemoryLog) Append(entry MemoryEntry) (evicted bool) {
	max := l.MaxSize
	if max <= 0 {
		max = DefaultMaxSize
	}
	l.Entries = append(l.Entries, entry)
	if len(l.Entries) <= max {
		return false
	}
	for i := range l.Entries {
		if l.Entries[i].Protected {
			continue
		}
		// Order-preserving removal of the oldest unprotected entry.
		copy(l.Entries[i:], l.Entries[i+1:])
		l.Entries = l.Entries[:len(l.Entries)-1]
		return true
	}
	slog.Warn("memory log: soft cap exceeded; all entries protected, none evicted",
		"count", len(l.Entries), "max", max)
	return false
}

// GetEntries returns the entries with GameTimestamp > since, sorted newest
// first (descending timestamp; ties broken by EventID descending, since
// EventID is monotonic), up to limit entries. A limit of 0 returns nil
// (validated before any filtering or allocation); a negative limit is
// treated as unlimited. since=0 returns all entries.
//
// The returned slice is a fresh copy: mutating it does not affect the log.
func (l *MemoryLog) GetEntries(since float64, limit int) []MemoryEntry {
	if limit == 0 {
		return nil
	}
	count := 0
	for _, e := range l.Entries {
		if e.GameTimestamp > since {
			count++
		}
	}
	if limit > 0 && count > limit {
		count = limit
	}
	if count == 0 {
		return nil
	}
	out := make([]MemoryEntry, 0, count)
	for _, e := range l.Entries {
		if e.GameTimestamp > since {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].GameTimestamp != out[j].GameTimestamp {
			return out[i].GameTimestamp > out[j].GameTimestamp
		}
		return out[i].EventID > out[j].EventID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Len returns the current number of entries in the log.
func (l *MemoryLog) Len() int {
	return len(l.Entries)
}

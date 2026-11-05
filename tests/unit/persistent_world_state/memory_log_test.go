// Unit tests for Story 003 of the Persistent World State epic: Memory Log.
//
// Evidence path: production/epics/persistent-world-state/
//
//	story-003-memory-log.md
//
// Tests map to the five acceptance criteria (AC-1 cap eviction of the
// oldest unprotected entry, AC-2 all-protected soft-cap exceed with
// warning, AC-3 mixed eviction skipping protected entries, AC-4
// GetEntries with since + limit, AC-5 entry field completeness) and the QA
// edge cases: small caps (MaxSize=10, MaxSize=1), append to empty log,
// cap already exceeded (52->53), single unprotected entry at cap,
// protected entries at the tail position, since=0 / since beyond newest /
// limit=0 / limit>len, and monotonic globally unique event IDs.
// Requirement TR-pws-002; governing ADR: ADR-0013.
package persistent_world_state_test

import (
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/persistent_world_state"
)

// newEntry builds a MemoryEntry with the given timestamp and protection
// flag. EventID is assigned from the global counter so IDs are unique
// within a test run; tests that assert specific EventID values construct
// entries directly.
func newEntry(ts float64, protected bool) persistent_world_state.MemoryEntry {
	return persistent_world_state.MemoryEntry{
		EventID:       persistent_world_state.NextEventID(),
		EventType:     "test_event",
		GameTimestamp: ts,
		RegionID:      "test_region",
		Protected:     protected,
	}
}

// newLog returns a MemoryLog with the given cap, pre-filled with n entries
// whose timestamps run 1.0, 2.0, ..., n.0 in oldest-first order. The
// protect function selects which positions are protected (by index).
func newLog(cap, n int, protect func(i int) bool) *persistent_world_state.MemoryLog {
	log := persistent_world_state.NewMemoryLog(cap)
	for i := 0; i < n; i++ {
		log.Entries = append(log.Entries, newEntry(float64(i+1), protect(i)))
	}
	return log
}

// entryAt returns the log entry with the given timestamp, or nil.
func entryAt(log *persistent_world_state.MemoryLog, ts float64) *persistent_world_state.MemoryEntry {
	for i := range log.Entries {
		if log.Entries[i].GameTimestamp == ts {
			return &log.Entries[i]
		}
	}
	return nil
}

// tsOf extracts the timestamps of entries for readable test failures.
func tsOf(entries []persistent_world_state.MemoryEntry) []float64 {
	ts := make([]float64, len(entries))
	for i, e := range entries {
		ts[i] = e.GameTimestamp
	}
	return ts
}

// TestAppendCapEviction covers AC-1: at cap with no protected entries, a
// new append evicts the oldest entry and total count remains at the cap.
// QA edge cases: small caps (MaxSize=10, MaxSize=1) and append to an
// empty log (no eviction).
func TestAppendCapEviction(t *testing.T) {
	tests := []struct {
		name        string
		cap         int
		prefill     int
		wantEvicted bool
		wantCount   int
	}{
		{"cap50_default", 50, 50, true, 50},
		{"small_cap10", 10, 10, true, 10},
		{"cap1_evicts_immediately", 1, 1, true, 1},
		{"append_to_empty", 3, 0, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newLog(tt.cap, tt.prefill, func(int) bool { return false })

			entry := newEntry(float64(tt.prefill+1), false)
			evicted := log.Append(entry)

			if evicted != tt.wantEvicted {
				t.Fatalf("evicted = %v, want %v", evicted, tt.wantEvicted)
			}
			if got := log.Len(); got != tt.wantCount {
				t.Fatalf("total count = %d, want %d", got, tt.wantCount)
			}
			// Oldest entry must be gone when an eviction happened.
			if tt.wantEvicted {
				if e := entryAt(log, 1.0); e != nil {
					t.Fatalf("oldest entry (ts=1.0) still present after eviction: %+v", *e)
				}
			}
			// New entry must be at the end.
			if got := log.Entries[log.Len()-1]; got.GameTimestamp != entry.GameTimestamp {
				t.Fatalf("new entry at end has ts=%v, want %v", got.GameTimestamp, entry.GameTimestamp)
			}
		})
	}
}

// TestAppendCapEvictionExact verifies the AC-1 full sequence on the default
// cap: 50 unprotected entries, append a 51st, oldest evicted, 50 remain in
// oldest-first order.
func TestAppendCapEvictionExact(t *testing.T) {
	log := newLog(persistent_world_state.DefaultMaxSize, 50, func(int) bool { return false })

	evicted := log.Append(newEntry(51.0, false))

	if !evicted {
		t.Fatal("expected eviction, got evicted=false")
	}
	if got := log.Len(); got != 50 {
		t.Fatalf("total count = %d, want 50", got)
	}
	if e := entryAt(log, 1.0); e != nil {
		t.Fatalf("oldest entry (ts=1.0) not evicted: %+v", *e)
	}
	if e := entryAt(log, 51.0); e == nil {
		t.Fatal("new entry (ts=51.0) missing after append")
	}
	// Remaining entries keep oldest-first order 2..51.
	for i := range log.Entries {
		if want := float64(i + 2); log.Entries[i].GameTimestamp != want {
			t.Fatalf("entries out of order: index %d has ts=%v, want %v", i, log.Entries[i].GameTimestamp, want)
		}
	}
}

// TestAppendAllProtected covers AC-2: all entries protected, the cap is
// temporarily exceeded, nothing is evicted, and a warning is logged.
func TestAppendAllProtected(t *testing.T) {
	tests := []struct {
		name      string
		cap       int
		prefill   int
		wantTotal int
	}{
		{"cap50_all_protected", 50, 50, 51},
		{"cap1_all_protected", 1, 1, 2},
		{"already_over_cap", 50, 52, 53}, // QA edge: 52->53 still appends
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := newLog(tt.cap, tt.prefill, func(int) bool { return true })

			evicted := log.Append(newEntry(99.0, true))

			if evicted {
				t.Fatal("expected no eviction when all entries protected, got evicted=true")
			}
			if got := log.Len(); got != tt.wantTotal {
				t.Fatalf("total count = %d, want %d", got, tt.wantTotal)
			}
			if e := entryAt(log, 1.0); e == nil {
				t.Fatal("oldest protected entry was evicted")
			}
		})
	}
}

// TestAppendMixedEviction covers AC-3: protected entries are skipped and the
// oldest unprotected entry is evicted.
func TestAppendMixedEviction(t *testing.T) {
	// Protected at positions 0, 2, 4, ..., 18 (QA AC-3 fixture); the rest
	// unprotected. 50 total at cap.
	protect := func(i int) bool { return i%2 == 0 && i < 19 }
	log := newLog(50, 50, protect)

	evicted := log.Append(newEntry(51.0, false))

	if !evicted {
		t.Fatal("expected eviction, got evicted=false")
	}
	if got := log.Len(); got != 50 {
		t.Fatalf("total count = %d, want 50", got)
	}
	// Oldest unprotected entry sat at position 1 (ts=2.0).
	if e := entryAt(log, 2.0); e != nil {
		t.Fatalf("oldest unprotected entry (ts=2.0) not evicted: %+v", *e)
	}
	// All protected entries (ts=1,3,5,...,19) must survive.
	for i := 0; i <= 18; i += 2 {
		if e := entryAt(log, float64(i+1)); e == nil {
			t.Fatalf("protected entry ts=%v was evicted", float64(i+1))
		}
	}
	// New entry appended at end.
	if e := entryAt(log, 51.0); e == nil {
		t.Fatal("new entry (ts=51.0) missing after append")
	}
}

// TestAppendMixedEvictionEdges covers the AC-3 QA edge cases: a single
// unprotected entry at cap, and protected entries only at the tail position.
func TestAppendMixedEvictionEdges(t *testing.T) {
	t.Run("single_unprotected_evicted", func(t *testing.T) {
		// Position 0 unprotected, 1..49 protected.
		log := newLog(50, 50, func(i int) bool { return i != 0 })

		evicted := log.Append(newEntry(51.0, false))

		if !evicted {
			t.Fatal("expected eviction, got evicted=false")
		}
		if got := log.Len(); got != 50 {
			t.Fatalf("total count = %d, want 50", got)
		}
		if e := entryAt(log, 1.0); e != nil {
			t.Fatalf("only unprotected entry (ts=1.0) not evicted: %+v", *e)
		}
	})

	t.Run("protected_at_tail_evicts_front", func(t *testing.T) {
		// Only position 49 protected; position 0 unprotected.
		log := newLog(50, 50, func(i int) bool { return i == 49 })

		evicted := log.Append(newEntry(51.0, false))

		if !evicted {
			t.Fatal("expected eviction, got evicted=false")
		}
		if e := entryAt(log, 1.0); e != nil {
			t.Fatalf("oldest unprotected entry (ts=1.0) not evicted: %+v", *e)
		}
		if e := entryAt(log, 50.0); e == nil {
			t.Fatal("protected tail entry (ts=50.0) was evicted")
		}
	})
}

// TestAppendNoEvictionBelowCap covers the baseline: appends below the cap
// never evict.
func TestAppendNoEvictionBelowCap(t *testing.T) {
	log := newLog(50, 10, func(int) bool { return false })

	if evicted := log.Append(newEntry(11.0, false)); evicted {
		t.Fatal("expected no eviction below cap, got evicted=true")
	}
	if got := log.Len(); got != 11 {
		t.Fatalf("total count = %d, want 11", got)
	}
	if e := entryAt(log, 1.0); e == nil {
		t.Fatal("oldest entry evicted below cap")
	}
}

// TestGetEntriesSinceLimit covers AC-4: entries newer than since, newest
// first, up to limit. limit=0 returns empty (validated before allocation),
// so "all entries" cases use limit=-1 (unlimited).
func TestGetEntriesSinceLimit(t *testing.T) {
	log := newLog(50, 5, func(int) bool { return false }) // ts 1..5 oldest->newest

	tests := []struct {
		name  string
		since float64
		limit int
		want  []float64 // timestamps, newest first
	}{
		{"since_mid_limit2", 2.5, 2, []float64{5.0, 4.0}}, // QA AC-4 fixture
		{"since_zero_all", 0, -1, []float64{5.0, 4.0, 3.0, 2.0, 1.0}},
		{"since_beyond_all", 10, -1, nil}, // QA edge: empty result
		{"limit_zero_empty", 0, 0, nil},   // QA edge: limit=0 validated first
		{"limit_gt_len_all", 0, 10, []float64{5.0, 4.0, 3.0, 2.0, 1.0}},
		{"limit_negative_unlimited", 0, -1, []float64{5.0, 4.0, 3.0, 2.0, 1.0}},
		{"exclusive_boundary", 2.0, -1, []float64{5.0, 4.0, 3.0}}, // since is exclusive
		{"limit1_newest", 0, 1, []float64{5.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := log.GetEntries(tt.since, tt.limit)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries %v, want %d %v", len(got), tsOf(got), len(tt.want), tt.want)
			}
			for i := range got {
				if got[i].GameTimestamp != tt.want[i] {
					t.Fatalf("entry %d ts=%v, want %v (got %v)", i, got[i].GameTimestamp, tt.want[i], tsOf(got))
				}
			}
		})
	}
}

// TestGetEntriesEqualTimestampTieBreak verifies the newest-first order is
// deterministic when timestamps collide: the higher EventID (later event)
// comes first.
func TestGetEntriesEqualTimestampTieBreak(t *testing.T) {
	log := persistent_world_state.NewMemoryLog(50)
	log.Append(persistent_world_state.MemoryEntry{EventID: 1, EventType: "a", GameTimestamp: 3.0})
	log.Append(persistent_world_state.MemoryEntry{EventID: 2, EventType: "b", GameTimestamp: 3.0})

	got := log.GetEntries(0, -1)

	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].EventID != 2 || got[1].EventID != 1 {
		t.Fatalf("tie order wrong: got IDs %d,%d want 2,1", got[0].EventID, got[1].EventID)
	}
}

// TestGetEntriesReturnsCopy verifies callers cannot mutate the log through
// the returned slice.
func TestGetEntriesReturnsCopy(t *testing.T) {
	log := newLog(50, 3, func(int) bool { return false })

	got := log.GetEntries(0, -1)
	got[0].Protected = true
	got[0].GameTimestamp = 999.0

	if log.Entries[2].Protected {
		t.Fatal("mutation leaked into the log")
	}
	if log.Entries[2].GameTimestamp == 999.0 {
		t.Fatal("mutation leaked into the log")
	}
}

// TestEntryFieldCompleteness covers AC-5: all eight fields round-trip
// through GetEntries unchanged.
func TestEntryFieldCompleteness(t *testing.T) {
	log := persistent_world_state.NewMemoryLog(50)
	want := persistent_world_state.MemoryEntry{
		EventID:       42,
		EventType:     "discovery_event",
		GameTimestamp: 137.5,
		ActorID:       entity.GameEntityID(7),
		TargetID:      entity.GameEntityID(9),
		RegionID:      "ruin_vault",
		Details:       `{"clue_id":"candle_stub","quality":"good"}`,
		Protected:     true,
	}
	log.Append(want)

	got := log.GetEntries(0, -1)

	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	e := got[0]
	if e.EventID != want.EventID ||
		e.EventType != want.EventType ||
		e.GameTimestamp != want.GameTimestamp ||
		e.ActorID != want.ActorID ||
		e.TargetID != want.TargetID ||
		e.RegionID != want.RegionID ||
		e.Details != want.Details ||
		e.Protected != want.Protected {
		t.Fatalf("fields mismatch:\n got %+v\nwant %+v", e, want)
	}
}

// TestEntryFieldCompletenessEdges covers the AC-5 QA edge cases: empty
// Details, zero GameTimestamp, and zero entity IDs (valid for
// system-generated events).
func TestEntryFieldCompletenessEdges(t *testing.T) {
	tests := []struct {
		name  string
		entry persistent_world_state.MemoryEntry
	}{
		{"empty_details", persistent_world_state.MemoryEntry{EventID: 1, EventType: "kill", GameTimestamp: 5.0, Details: ""}},
		{"zero_timestamp", persistent_world_state.MemoryEntry{EventID: 2, EventType: "world_loaded", GameTimestamp: 0}},
		{"zero_entity_ids", persistent_world_state.MemoryEntry{EventID: 3, EventType: "system_event", GameTimestamp: 5.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := persistent_world_state.NewMemoryLog(50)
			log.Append(tt.entry)

			// since=-1: the filter is strictly GameTimestamp > since, so a
			// stored zero timestamp is retrievable only below 0.
			got := log.GetEntries(-1, -1)

			if len(got) != 1 {
				t.Fatalf("got %d entries, want 1", len(got))
			}
			if got[0] != tt.entry {
				t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got[0], tt.entry)
			}
		})
	}
}

// TestNextEventIDMonotonicUnique covers the "monotonic, globally unique"
// property of event IDs (AC-5 requirement text, ADR-0013).
func TestNextEventIDMonotonicUnique(t *testing.T) {
	seen := make(map[uint64]bool)
	var prev uint64
	for i := 0; i < 1000; i++ {
		id := persistent_world_state.NextEventID()
		if id <= prev {
			t.Fatalf("event ID %d not strictly greater than previous %d", id, prev)
		}
		if seen[id] {
			t.Fatalf("event ID %d reused", id)
		}
		seen[id] = true
		prev = id
	}
}

// TestNewMemoryLogDefaultCap covers the constructor contract: non-positive
// maxSize falls back to DefaultMaxSize (50).
func TestNewMemoryLogDefaultCap(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
		want    int
	}{
		{"zero_defaults", 0, 50},
		{"negative_defaults", -5, 50},
		{"explicit_cap", 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := persistent_world_state.NewMemoryLog(tt.maxSize).MaxSize; got != tt.want {
				t.Fatalf("MaxSize = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestZeroValueMemoryLogUsable covers the zero-value struct: a MemoryLog{}
// behaves as DefaultMaxSize so it is usable without NewMemoryLog.
func TestZeroValueMemoryLogUsable(t *testing.T) {
	var log persistent_world_state.MemoryLog
	for i := 0; i < 51; i++ {
		log.Append(newEntry(float64(i+1), false))
	}
	if got := log.Len(); got != 50 {
		t.Fatalf("zero-value log count = %d, want 50 (default cap)", got)
	}
	if e := entryAt(&log, 1.0); e != nil {
		t.Fatalf("oldest entry (ts=1.0) not evicted in zero-value log")
	}
	if e := entryAt(&log, 51.0); e == nil {
		t.Fatal("new entry (ts=51.0) missing in zero-value log")
	}
}

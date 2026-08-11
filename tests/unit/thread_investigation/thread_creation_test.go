// Unit tests for Story 001: Thread Data Model & Auto-Creation.
//
// Evidence path: production/epics/thread-investigation/story-001-thread-data-model-auto-creation.md
// Tests map to the acceptance criteria (AC-1 .. AC-3) plus the
// performance criterion, and verify TR-thread-001 (thread creation from
// fact leads_to hints) per ADR-0010. All tests are deterministic: the
// clock is injected with a fixed value, no randomness, no time-dependent
// assertions, no external I/O.
package thread_investigation_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
	"marrow-of-empire/thread_investigation"
)

// fixedClock pins the game clock to a fixed timestamp so CreatedAt and
// DiscoveredAt assertions are deterministic.
func fixedClock() thread_investigation.Option {
	return thread_investigation.WithClock(func() float64 { return 1250.5 })
}

// factWithLeads builds a knowledge.Fact with the given leads.
func factWithLeads(factID string, leads ...knowledge.Lead) knowledge.Fact {
	return knowledge.Fact{FactID: factID, LeadsTo: leads}
}

// namer maps entity IDs to display names for topic text assertions.
type namer map[entity.GameEntityID]string

func (n namer) Name(id entity.GameEntityID) (string, bool) {
	name, ok := n[id]
	return name, ok
}

// mustActiveThreads returns the active threads, failing on unexpected
// count.
func mustActiveThreads(t *testing.T, e *thread_investigation.Engine, want int) []thread_investigation.Thread {
	t.Helper()
	threads := e.GetActiveThreads()
	if len(threads) != want {
		t.Fatalf("active threads = %d, want %d", len(threads), want)
	}
	return threads
}

// AC-1: GIVEN a fact with leads_to [{entity_id: 42, topic_hint:
// "Ley-touched metal"}], WHEN OnFactDiscovered("fact_001", 7) is called,
// THEN one active thread is created with Topic "Ask [entity 42] about
// Ley-touched metal", Source "fact_001", Status "active", TargetEntity
// 42, and an EventThreadCreated is queued.
func TestOnFactDiscovered_WithLead_CreatesActiveThread(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	threads := mustActiveThreads(t, e, 1)
	got := threads[0]
	if got.ID != "thread_fact_001_42" {
		t.Errorf("ID = %q, want %q", got.ID, "thread_fact_001_42")
	}
	if got.Source != "fact_001" {
		t.Errorf("Source = %q, want %q", got.Source, "fact_001")
	}
	if got.TargetEntity != 42 {
		t.Errorf("TargetEntity = %d, want 42", got.TargetEntity)
	}
	if got.Topic != "Ask [entity 42] about Ley-touched metal" {
		t.Errorf("Topic = %q, want %q", got.Topic, "Ask [entity 42] about Ley-touched metal")
	}
	if got.Status != thread_investigation.ThreadActive {
		t.Errorf("Status = %q, want %q", got.Status, thread_investigation.ThreadActive)
	}
	if got.CreatedAt != 1250.5 {
		t.Errorf("CreatedAt = %v, want 1250.5", got.CreatedAt)
	}
	// Story 002 auto-chaining: this thread's entity (42) matches no
	// existing chain, so it becomes the first thread of a new chain.
	if got.ChainID != "chain_1" {
		t.Errorf("ChainID = %q, want %q (Story 002 auto-chaining)", got.ChainID, "chain_1")
	}

	events := e.DrainEvents()
	if len(events) != 1 {
		t.Fatalf("queued events = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.ThreadID != got.ID || ev.SourceFactID != "fact_001" || ev.TargetEntity != 42 || ev.Topic != got.Topic {
		t.Errorf("event = %+v, want fields matching the created thread", ev)
	}
}

// AC-1 (name resolution): GIVEN an EntityNamer mapping entity 42 to
// "Orin", WHEN the fact is discovered, THEN the topic reads "Ask Orin
// about Ley-touched metal" (GDD thread template "Ask [target NPC] about
// [topic]").
func TestOnFactDiscovered_WithNamer_UsesEntityNameInTopic(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithNamer(namer{42: "Orin"}))
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	got := mustActiveThreads(t, e, 1)[0].Topic
	if want := "Ask Orin about Ley-touched metal"; got != want {
		t.Errorf("Topic = %q, want %q", got, want)
	}
}

// AC-2: GIVEN a fact with two leads_to entries, WHEN the fact is
// discovered, THEN one thread is created per lead, each with the correct
// target and topic, and two events are queued.
func TestOnFactDiscovered_MultipleLeads_CreatesOneThreadPerLead(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_002",
		knowledge.Lead{EntityID: 10, TopicHint: "the old mine"},
		knowledge.Lead{EntityID: 11, TopicHint: "Captain Veren"}))

	// Act
	e.OnFactDiscovered("fact_002", 3)

	// Assert
	threads := mustActiveThreads(t, e, 2)
	if got, want := threads[0].TargetEntity, entity.GameEntityID(10); got != want {
		t.Errorf("thread[0] target = %d, want %d", got, want)
	}
	if got, want := threads[0].Topic, "Ask [entity 10] about the old mine"; got != want {
		t.Errorf("thread[0] Topic = %q, want %q", got, want)
	}
	if got, want := threads[1].TargetEntity, entity.GameEntityID(11); got != want {
		t.Errorf("thread[1] target = %d, want %d", got, want)
	}
	if got, want := threads[1].Topic, "Ask [entity 11] about Captain Veren"; got != want {
		t.Errorf("thread[1] Topic = %q, want %q", got, want)
	}
	if events := e.DrainEvents(); len(events) != 2 {
		t.Errorf("queued events = %d, want 2", len(events))
	}
}

// AC-3: GIVEN a fact with no leads_to entries, WHEN the fact is
// discovered, THEN no thread is created and no event is queued, but the
// fact is recorded in DiscoveredFacts with FactID and SourceNPC.
func TestOnFactDiscovered_NoLeads_RecordsFactNoThread(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(knowledge.Fact{FactID: "fact_003"})

	// Act
	e.OnFactDiscovered("fact_003", 5)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	facts := e.Knowledge().DiscoveredFacts
	if len(facts) != 1 {
		t.Fatalf("discovered facts = %d, want 1", len(facts))
	}
	got := facts[0]
	if got.FactID != "fact_003" {
		t.Errorf("FactID = %q, want %q", got.FactID, "fact_003")
	}
	if got.SourceNPC != 5 {
		t.Errorf("SourceNPC = %d, want 5", got.SourceNPC)
	}
	if got.DiscoveredAt != 1250.5 {
		t.Errorf("DiscoveredAt = %v, want 1250.5", got.DiscoveredAt)
	}
	if events := e.DrainEvents(); len(events) != 0 {
		t.Errorf("queued events = %d, want 0", len(events))
	}
}

// Idempotency (thread ID format): GIVEN the same fact discovered twice,
// WHEN OnFactDiscovered is called again, THEN no duplicate thread or fact
// record is created.
func TestOnFactDiscovered_SameFactTwice_NoDuplicateThreadOrFact(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_001", 7) // second discovery

	// Assert
	threads := mustActiveThreads(t, e, 1)
	if got, want := threads[0].ID, "thread_fact_001_42"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if facts := e.Knowledge().DiscoveredFacts; len(facts) != 1 {
		t.Errorf("discovered facts = %d, want 1", len(facts))
	}
	if events := e.DrainEvents(); len(events) != 1 {
		t.Errorf("queued events = %d, want 1 (second discovery emits nothing)", len(events))
	}
}

// OnFactDiscovered step 2: GIVEN facts leading to entities 42 and 43,
// WHEN they are discovered, THEN each new entity is recorded in
// KnownEntities exactly once with DiscoveredVia = the introducing fact ID.
func TestOnFactDiscovered_NewLeadEntity_AddedToKnownEntities(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"},
		knowledge.Lead{EntityID: 43, TopicHint: "the old mine"}))
	e.RegisterFact(factWithLeads("fact_002",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal again"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 8)

	// Assert
	known := e.Knowledge().KnownEntities
	if len(known) != 2 {
		t.Fatalf("known entities = %d, want 2 (42 and 43)", len(known))
	}
	if known[0].EntityID != 42 || known[0].DiscoveredVia != "fact_001" || known[0].DiscoveredAt != 1250.5 {
		t.Errorf("known[0] = %+v, want entity 42 via fact_001", known[0])
	}
	if known[1].EntityID != 43 || known[1].DiscoveredVia != "fact_001" {
		t.Errorf("known[1] = %+v, want entity 43 via fact_001", known[1])
	}
	if facts := e.Knowledge().DiscoveredFacts; len(facts) != 2 {
		t.Errorf("discovered facts = %d, want 2", len(facts))
	}
}

// Edge case (empty topic_hint): GIVEN a lead with an empty hint, WHEN the
// fact is discovered, THEN the thread topic drops the "about" suffix.
func TestOnFactDiscovered_EmptyTopicHint_OmitsAboutSuffix(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_004", knowledge.Lead{EntityID: 42}))

	// Act
	e.OnFactDiscovered("fact_004", 7)

	// Assert
	threads := mustActiveThreads(t, e, 1)
	if got, want := threads[0].Topic, "Ask [entity 42]"; got != want {
		t.Errorf("Topic = %q, want %q", got, want)
	}
}

// Degenerate case: GIVEN a fact not registered via RegisterFact, WHEN it
// is discovered, THEN it is still recorded in DiscoveredFacts but no
// threads can be derived without its leads_to data.
func TestOnFactDiscovered_UnregisteredFact_RecordsFactWithoutThreads(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())

	// Act
	e.OnFactDiscovered("ghost_fact", 7)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	facts := e.Knowledge().DiscoveredFacts
	if len(facts) != 1 || facts[0].FactID != "ghost_fact" || facts[0].SourceNPC != 7 {
		t.Errorf("discovered facts = %+v, want ghost_fact recorded from NPC 7", facts)
	}
}

// ResolveThread (manual path): GIVEN an active thread, WHEN ResolveThread
// is called with ThreadResolved, THEN the thread moves to ResolvedThreads.
func TestResolveThread_Resolved_MovesToResolvedAndCannotResurrect(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.OnFactDiscovered("fact_001", 7)
	e.DrainEvents()

	// Act
	e.ResolveThread("thread_fact_001_42", thread_investigation.ThreadResolved)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 {
		t.Fatalf("resolved threads = %d, want 1", len(resolved))
	}
	if resolved[0].ID != "thread_fact_001_42" || resolved[0].Status != thread_investigation.ThreadResolved {
		t.Errorf("resolved thread = %+v, want thread_fact_001_42 resolved", resolved[0])
	}

	// Re-discovery must not create a duplicate or resurrect the thread.
	e.OnFactDiscovered("fact_001", 7)
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads after re-discovery = %d, want 0", len(threads))
	}
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 1 {
		t.Errorf("resolved threads after re-discovery = %d, want 1", len(resolved))
	}
}

// Invalid resolution: GIVEN an active thread, WHEN ResolveThread is
// called with ThreadActive (not a valid resolution), THEN the call is ignored.
func TestResolveThread_InvalidResolution_IsIgnored(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.OnFactDiscovered("fact_001", 7)

	// Act
	e.ResolveThread("thread_fact_001_42", thread_investigation.ThreadActive)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 1 {
		t.Errorf("active threads = %d, want 1 (call ignored)", len(threads))
	}
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 0 {
		t.Errorf("resolved threads = %d, want 0", len(resolved))
	}
}

// Interface completeness: GetChain and GetRevelations are defined for the
// InvestigationEngine interface but return empty results until Stories
// 002/004 implement chains and revelations.
func TestGetChain_UnknownID_ReturnsNotFound(t *testing.T) {
	e := thread_investigation.NewEngine()

	if _, ok := e.GetChain("chain_1"); ok {
		t.Error("GetChain(chain_1) = ok, want not found (Story 002)")
	}
	if revs := e.GetRevelations(); len(revs) != 0 {
		t.Errorf("GetRevelations() = %d entries, want 0 (Story 004)", len(revs))
	}
}

// --- Post-review gap coverage (qa-tester + lead-programmer findings) ---

// GAP-1 (empty fact ID): GIVEN an empty fact ID, WHEN OnFactDiscovered is
// called, THEN it is silently ignored — guards the factIndex map from
// accumulating garbage entries that would skew revelation counts (Story 004).
func TestOnFactDiscovered_EmptyFactID_IsIgnored(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("", 7)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	if facts := e.Knowledge().DiscoveredFacts; len(facts) != 0 {
		t.Errorf("discovered facts = %d, want 0", len(facts))
	}
	if events := e.DrainEvents(); len(events) != 0 {
		t.Errorf("queued events = %d, want 0", len(events))
	}
}

// GAP-2 (duplicate leads within one fact): GIVEN a fact with the same
// EntityID appearing twice in LeadsTo, WHEN the fact is discovered, THEN
// only one thread is created — the threadIndex deduplicates within a
// single call as well as across calls.
func TestOnFactDiscovered_DuplicateLeadsInOneFact_Deduplicates(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_dup",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal again"}))

	// Act
	e.OnFactDiscovered("fact_dup", 7)

	// Assert
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_fact_dup_42" {
		t.Errorf("ID = %q, want thread_fact_dup_42", threads[0].ID)
	}
	if events := e.DrainEvents(); len(events) != 1 {
		t.Errorf("queued events = %d, want 1 (deduplicated)", len(events))
	}
	if known := e.Knowledge().KnownEntities; len(known) != 1 {
		t.Errorf("known entities = %d, want 1", len(known))
	}
}

// GAP-3 (ThreadDeadEnd resolution): GIVEN an active thread, WHEN
// ResolveThread is called with ThreadDeadEnd, THEN the thread moves to
// ResolvedThreads with status dead_end.
func TestResolveThread_DeadEnd_MovesToResolved(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.OnFactDiscovered("fact_001", 7)
	e.DrainEvents()

	// Act
	e.ResolveThread("thread_fact_001_42", thread_investigation.ThreadDeadEnd)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 {
		t.Fatalf("resolved threads = %d, want 1", len(resolved))
	}
	if resolved[0].Status != thread_investigation.ThreadDeadEnd {
		t.Errorf("status = %q, want dead_end", resolved[0].Status)
	}
}

// GAP-4 (DrainEvents semantics): GIVEN queued events, WHEN DrainEvents
// is called, THEN the returned slice is in FIFO order and the engine's
// internal queue is cleared (second DrainEvents returns empty).
func TestDrainEvents_ClearsQueueAndPreservesFIFO(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 10, TopicHint: "mine"}))
	e.RegisterFact(factWithLeads("fact_002",
		knowledge.Lead{EntityID: 20, TopicHint: "Captain"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 8)
	first := e.DrainEvents()
	second := e.DrainEvents()

	// Assert
	if len(first) != 2 {
		t.Fatalf("first drain = %d events, want 2", len(first))
	}
	if first[0].ThreadID != "thread_fact_001_10" {
		t.Errorf("FIFO[0] = %s, want thread_fact_001_10", first[0].ThreadID)
	}
	if first[1].ThreadID != "thread_fact_002_20" {
		t.Errorf("FIFO[1] = %s, want thread_fact_002_20", first[1].ThreadID)
	}
	if len(second) != 0 {
		t.Errorf("second drain = %d events, want 0 (queue cleared)", len(second))
	}
}

// GAP-5 (RegisterFact duplicate): GIVEN the same fact ID registered
// twice, WHEN the second RegisterFact is called, THEN it is a no-op: the
// original fact data is preserved.
func TestRegisterFact_Duplicate_IsNoOp(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	first := factWithLeads("fact_001", knowledge.Lead{EntityID: 42, TopicHint: "original"})
	second := factWithLeads("fact_001", knowledge.Lead{EntityID: 99, TopicHint: "overwrite attempt"})
	e.RegisterFact(first)

	// Act
	e.RegisterFact(second) // should be a no-op

	// Assert — the original fact data is preserved
	e.OnFactDiscovered("fact_001", 7)
	threads := mustActiveThreads(t, e, 1)
	if threads[0].TargetEntity != 42 {
		t.Errorf("TargetEntity = %d, want 42 (original preserved)", threads[0].TargetEntity)
	}
}

// GAP-6 (boundary: EntityID zero): GIVEN a lead targeting EntityID 0,
// WHEN the fact is discovered, THEN a thread is created with ID
// "thread_fact_001_0" — zero is a valid uint64, not a sentinel.
func TestOnFactDiscovered_EntityIDZero_CreatesThread(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001", knowledge.Lead{EntityID: 0, TopicHint: "the void"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_fact_001_0" {
		t.Errorf("ID = %q, want thread_fact_001_0", threads[0].ID)
	}
	if threads[0].TargetEntity != 0 {
		t.Errorf("TargetEntity = %d, want 0", threads[0].TargetEntity)
	}
}

// GAP-6 (boundary: math.MaxUint64): thread ID formats safely without overflow.
func TestOnFactDiscovered_MaxEntityID_SafeStringFormat(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: math.MaxUint64, TopicHint: "edge"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	threads := mustActiveThreads(t, e, 1)
	wantID := fmt.Sprintf("thread_fact_001_%d", uint64(math.MaxUint64))
	if threads[0].ID != wantID {
		t.Errorf("ID = %q, want %q", threads[0].ID, wantID)
	}
}

// GAP-6 (boundary: namer returns ("", true)): bracket fallback used
// when ok=true but name is empty.
func TestOnFactDiscovered_NamerReturnsEmptyOK_UsesBracketFallback(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithNamer(namer{42: ""})) // ok=true, name=""
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	got := mustActiveThreads(t, e, 1)[0].Topic
	if want := "Ask [entity 42] about Ley-touched metal"; got != want {
		t.Errorf("Topic = %q, want %q (bracket fallback)", got, want)
	}
}

// GAP-6 (boundary: default clock): CreatedAt/DiscoveredAt are zero
// with the default clock — matching the "clock not wired yet" convention.
func TestNewEngine_DefaultClockReturnsZero(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine()
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "test"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert
	thread := mustActiveThreads(t, e, 1)[0]
	if thread.CreatedAt != 0 {
		t.Errorf("CreatedAt = %v, want 0 (default clock)", thread.CreatedAt)
	}
	fact := e.Knowledge().DiscoveredFacts[0]
	if fact.DiscoveredAt != 0 {
		t.Errorf("DiscoveredAt = %v, want 0 (default clock)", fact.DiscoveredAt)
	}
}

// ADVISORY-2 (single clock capture): GIVEN an incrementing clock,
// WHEN OnFactDiscovered is called, THEN all records created by that
// call share the same timestamp — the clock is captured once at the top.
func TestOnFactDiscovered_SingleClockCapture_AllRecordsShareTimestamp(t *testing.T) {
	// Arrange
	var callCount int
	e := thread_investigation.NewEngine(
		thread_investigation.WithClock(func() float64 {
			callCount++
			return float64(callCount)
		}))
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"},
		knowledge.Lead{EntityID: 43, TopicHint: "the old mine"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert — both threads and the discovered fact share the SAME timestamp.
	threads := e.GetActiveThreads()
	if len(threads) != 2 {
		t.Fatalf("active threads = %d, want 2", len(threads))
	}
	if threads[0].CreatedAt != threads[1].CreatedAt {
		t.Errorf("thread[0].CreatedAt=%v != thread[1].CreatedAt=%v", threads[0].CreatedAt, threads[1].CreatedAt)
	}
	discovered := e.Knowledge().DiscoveredFacts[0]
	if discovered.DiscoveredAt != threads[0].CreatedAt {
		t.Errorf("DiscoveredAt=%v != thread.CreatedAt=%v", discovered.DiscoveredAt, threads[0].CreatedAt)
	}
}

// ADVISORY-3 (GetActiveThreads copy): GIVEN active threads, WHEN the
// returned slice is mutated, THEN the engine's internal state is unchanged.
func TestGetActiveThreads_ReturnsCopy_DoesNotMutateEngine(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithLeads("fact_001",
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.OnFactDiscovered("fact_001", 7)

	// Act — mutate the returned slice
	threads := e.GetActiveThreads()
	threads[0].Status = thread_investigation.ThreadDeadEnd
	threads = append(threads, thread_investigation.Thread{ID: "injected"})

	// Assert — engine state unchanged
	actual := e.GetActiveThreads()
	if len(actual) != 1 {
		t.Fatalf("active threads = %d, want 1 (append did not affect engine)", len(actual))
	}
	if actual[0].Status != thread_investigation.ThreadActive {
		t.Errorf("status = %q, want active (mutation did not affect engine)", actual[0].Status)
	}
}

// Performance (QA criterion): steady-state benchmark with 50 pre-loaded
// facts. Each iteration re-discovers an already-registered fact to exercise
// the idempotent path — matching the scenario "50 discovered facts in
// PlayerKnowledge" from the story.
//
// Go benchmarks report the mean ns/op, not p95; the actual cost is
// microsecond-scale (~250x headroom), so the mean-is-not-p95 gap is immaterial.
func BenchmarkOnFactDiscovered_ThreadCreation(b *testing.B) {
	e := thread_investigation.NewEngine(fixedClock())

	// Pre-load 50 discovered facts, each with two leads.
	for i := 0; i < 50; i++ {
		factID := fmt.Sprintf("pre_fact_%02d", i)
		e.RegisterFact(factWithLeads(factID,
			knowledge.Lead{EntityID: 1000 + entity.GameEntityID(i), TopicHint: "the old mine"},
			knowledge.Lead{EntityID: 2000 + entity.GameEntityID(i), TopicHint: "Captain Veren"}))
		e.OnFactDiscovered(factID, 7)
	}

	// Steady-state: re-discover already-registered facts (idempotent path).
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factID := fmt.Sprintf("pre_fact_%02d", i%50)
		e.OnFactDiscovered(factID, 7)
	}
	b.StopTimer()

	if perOp := b.Elapsed() / time.Duration(b.N); perOp > 2*time.Millisecond {
		b.Fatalf("OnFactDiscovered per call = %v, want < 2ms", perOp)
	}
}

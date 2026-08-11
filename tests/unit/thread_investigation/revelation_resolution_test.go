// Unit tests for Story 004: Revelation Threshold & Thread Resolution.
//
// Evidence path: production/epics/thread-investigation/story-004-revelation-threshold-resolution.md
// Tests map to the acceptance criteria (AC-1 .. AC-5) plus the QA edge
// cases (spam prevention, prioritized status, threshold config, cross-chain
// isolation, NPC-death auto-resolution, thread reactivation) and the
// performance criterion, verifying TR-thread-004 (revelation threshold per
// ADR-0010 Step 5) and the resolution state machine (ADR-0010 CR-4). All
// tests are deterministic: the clock is injected with a fixed value, no
// randomness, no time-dependent assertions, no external I/O.
//
// Shared helpers (fixedClock, namer, mustActiveThreads, factWithLeads,
// factWithRefs, factWithContradicts) live in thread_creation_test.go and
// auto_chaining_test.go — same package.
package thread_investigation_test

import (
	"fmt"
	"testing"
	"time"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
	"marrow-of-empire/thread_investigation"
)

// factLeadingTo builds a knowledge.Fact with a single lead to the target
// entity (thread topics carry the "the mine" hint, which also becomes the
// chain summary).
func factLeadingTo(factID string, target entity.GameEntityID) knowledge.Fact {
	return factWithRefs(factID, nil, knowledge.Lead{EntityID: target, TopicHint: "the mine"})
}

// revelationEvents drains the chain event queue and returns only the
// revelation.ready events, in FIFO order.
func revelationEvents(t *testing.T, e *thread_investigation.Engine) []thread_investigation.EventRevelationReady {
	t.Helper()
	events := e.DrainChainEvents()
	var out []thread_investigation.EventRevelationReady
	for _, ev := range events {
		if ev.Type != thread_investigation.EventRevelationReadyType {
			continue
		}
		out = append(out, ev.Data.(thread_investigation.EventRevelationReady))
	}
	return out
}

// threadResolvedEvents drains the chain event queue and returns only the
// thread.resolved events, in FIFO order.
func threadResolvedEvents(t *testing.T, e *thread_investigation.Engine) []thread_investigation.EventThreadResolved {
	t.Helper()
	events := e.DrainChainEvents()
	var out []thread_investigation.EventThreadResolved
	for _, ev := range events {
		if ev.Type != thread_investigation.EventThreadResolvedType {
			continue
		}
		out = append(out, ev.Data.(thread_investigation.EventThreadResolved))
	}
	return out
}

// --- AC-1: revelation threshold met (3 facts) ---

// AC-1: GIVEN a chain with two facts leading to entity 42, WHEN a third
// fact leading to entity 42 is discovered, THEN the chain flips to
// revelation_ready, EventRevelationReady is queued via the DrainChainEvents
// pipeline as a ChainEvent envelope with FactCount 3 and TargetEntity 42,
// and the revelation is recorded in PlayerKnowledge.Revelations (readable
// via GetRevelations).
func TestOnFactDiscovered_ThreeFactsSameTarget_RevelationReady(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_001", 42))
	e.RegisterFact(factLeadingTo("fact_002", 42))
	e.RegisterFact(factLeadingTo("fact_003", 42))

	// Act — the first two discoveries keep the chain active (AC-2 check
	// folded in: no event before the threshold).
	e.OnFactDiscovered("fact_001", 5)
	e.OnFactDiscovered("fact_002", 8)
	if evs := revelationEvents(t, e); len(evs) != 0 {
		t.Fatalf("revelation events after 2 facts = %d, want 0", len(evs))
	}
	if chain, _ := e.GetChain("chain_1"); chain.Status != "active" {
		t.Fatalf("chain status after 2 facts = %q, want active", chain.Status)
	}
	e.OnFactDiscovered("fact_003", 11)

	// Assert — chain flips to revelation_ready with all three facts.
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the chain")
	}
	if chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status = %q, want %q", chain.Status, thread_investigation.ChainStatusRevelationReady)
	}
	if len(chain.Facts) != 3 {
		t.Errorf("chain facts = %v, want 3 facts", chain.Facts)
	}

	// Assert — the event travels the ChainEvent envelope with the clock
	// capture and all payload fields.
	events := e.DrainChainEvents()
	var found *thread_investigation.ChainEvent
	for i := range events {
		if events[i].Type == thread_investigation.EventRevelationReadyType {
			found = &events[i]
		}
	}
	if found == nil {
		t.Fatalf("no revelation.ready in chain events %+v", events)
	}
	if found.Timestamp != 1250.5 {
		t.Errorf("Timestamp = %v, want 1250.5 (clock capture)", found.Timestamp)
	}
	ev, ok := found.Data.(thread_investigation.EventRevelationReady)
	if !ok {
		t.Fatalf("Data = %T, want EventRevelationReady", found.Data)
	}
	if ev.ChainID != "chain_1" || ev.FactCount != 3 || ev.TargetEntity != 42 {
		t.Errorf("event = %+v, want ChainID chain_1, FactCount 3, TargetEntity 42", ev)
	}
	if ev.Summary != "the mine" {
		t.Errorf("Summary = %q, want %q (first topic hint)", ev.Summary, "the mine")
	}

	// Assert — revelation recorded in PlayerKnowledge, readable via
	// GetRevelations.
	revs := e.GetRevelations()
	if len(revs) != 1 {
		t.Fatalf("GetRevelations = %d revelations, want 1", len(revs))
	}
	if revs[0].ID != "revelation_chain_1" || revs[0].ChainID != "chain_1" ||
		revs[0].DiscoveredAt != 1250.5 || revs[0].ConfrontedAt != 0 {
		t.Errorf("revelation = %+v, want ID revelation_chain_1, ChainID chain_1, DiscoveredAt 1250.5, not confronted", revs[0])
	}
	if got := e.Knowledge().Revelations; len(got) != 1 {
		t.Errorf("PlayerKnowledge.Revelations = %d, want 1", len(got))
	}
}

// QA edge case (spam prevention, ADR-0010 risk mitigation): GIVEN a chain
// that already reached revelation_ready, WHEN a fourth and fifth fact
// leading to the same entity are discovered, THEN no further
// EventRevelationReady fires and no second revelation is recorded — the
// signal is one-time; extra facts only deepen the chain.
func TestOnFactDiscovered_RevelationReady_AdditionalFacts_NoDuplicateEvents(t *testing.T) {
	// Arrange — five facts all leading to entity 42.
	e := thread_investigation.NewEngine(fixedClock())
	for i, id := range []string{"fact_201", "fact_202", "fact_203", "fact_204", "fact_205"} {
		e.RegisterFact(factLeadingTo(id, 42))
		e.OnFactDiscovered(id, entity.GameEntityID(5+i))
	}
	if evs := revelationEvents(t, e); len(evs) != 1 {
		t.Fatalf("revelation events after 5 facts = %d, want exactly 1", len(evs))
	}

	// Act — a sixth fact, same target.
	e.RegisterFact(factLeadingTo("fact_206", 42))
	e.OnFactDiscovered("fact_206", 30)

	// Assert — no re-fire, single recorded revelation, status retained.
	if evs := revelationEvents(t, e); len(evs) != 0 {
		t.Errorf("revelation events after 6th fact = %d, want 0 (spam prevention)", len(evs))
	}
	if revs := e.GetRevelations(); len(revs) != 1 {
		t.Errorf("GetRevelations = %d, want 1 (recorded once)", len(revs))
	}
	if chain, _ := e.GetChain("chain_1"); chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status = %q, want revelation_ready", chain.Status)
	}
}

// --- AC-2: revelation threshold NOT met (2 facts) ---

// AC-2: GIVEN a chain with only two facts leading to entity 55, WHEN the
// second fact is discovered, THEN the chain stays active — no
// revelation_ready status, no event, no revelation recorded.
func TestOnFactDiscovered_TwoFactsSameTarget_ChainStaysActive(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_101", 55))
	e.RegisterFact(factLeadingTo("fact_102", 55))

	// Act
	e.OnFactDiscovered("fact_101", 5)
	e.OnFactDiscovered("fact_102", 8)

	// Assert
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the chain")
	}
	if chain.Status != "active" {
		t.Errorf("chain status = %q, want active (threshold not met)", chain.Status)
	}
	if evs := revelationEvents(t, e); len(evs) != 0 {
		t.Errorf("revelation events = %d, want 0", len(evs))
	}
	if revs := e.GetRevelations(); len(revs) != 0 {
		t.Errorf("GetRevelations = %d, want 0", len(revs))
	}
}

// --- Revelation counting: contradiction group (Step 5b) ---

// Step 5b (ADR-0010): GIVEN three facts that all contradict the same
// already-discovered fact_Z, WHEN the third contradicting fact is
// discovered, THEN the chain flips to revelation_ready — the contradiction
// group counts toward the threshold — and EventRevelationReady carries
// TargetEntity 0 (the conclusion is a fact, not an entity).
func TestOnFactDiscovered_ThreeFactsContradictSameFact_RevelationReady(t *testing.T) {
	// Arrange — fact_Z is discovered first (no leads, no chain), then three
	// facts referencing entity 42 all contradict it.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_Z", []entity.GameEntityID{42}))
	e.RegisterFact(factWithContradicts("fact_W1", []string{"fact_Z"}, []entity.GameEntityID{42}, "W1", knowledge.FactType_Fact))
	e.RegisterFact(factWithContradicts("fact_W2", []string{"fact_Z"}, []entity.GameEntityID{42}, "W2", knowledge.FactType_Fact))
	e.RegisterFact(factWithContradicts("fact_W3", []string{"fact_Z"}, []entity.GameEntityID{42}, "W3", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_Z", 4)
	e.OnFactDiscovered("fact_W1", 5)
	e.OnFactDiscovered("fact_W2", 8)
	e.OnFactDiscovered("fact_W3", 11)

	// Assert
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the contradiction chain")
	}
	if chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status = %q, want revelation_ready", chain.Status)
	}
	evs := revelationEvents(t, e)
	if len(evs) != 1 {
		t.Fatalf("revelation events = %d, want 1", len(evs))
	}
	if evs[0].FactCount != 3 || evs[0].TargetEntity != 0 {
		t.Errorf("event = %+v, want FactCount 3, TargetEntity 0", evs[0])
	}
}

// --- QA edge case: cross-chain facts do not count ---

// QA edge case (cross-chain isolation): GIVEN two separate chains — facts
// leading to entity 42 (chain_1) and facts leading to entity 99 (chain_2) —
// WHEN chain_1 reaches three facts, THEN only chain_1 flips to
// revelation_ready; chain_2's facts never count toward chain_1's threshold.
func TestOnFactDiscovered_CrossChainFacts_DoNotCount(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	for _, id := range []string{"fact_401", "fact_402"} {
		e.RegisterFact(factLeadingTo(id, 42))
		e.OnFactDiscovered(id, 5)
	}
	for _, id := range []string{"fact_411", "fact_412"} {
		e.RegisterFact(factLeadingTo(id, 99))
		e.OnFactDiscovered(id, 8)
	}
	if evs := revelationEvents(t, e); len(evs) != 0 {
		t.Fatalf("revelation events before threshold = %d, want 0", len(evs))
	}

	// Act — the third 42-fact.
	e.RegisterFact(factLeadingTo("fact_403", 42))
	e.OnFactDiscovered("fact_403", 11)

	// Assert — only chain_1 is ready, with exactly one event.
	if chain, _ := e.GetChain("chain_1"); chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain_1 status = %q, want revelation_ready", chain.Status)
	}
	if chain, _ := e.GetChain("chain_2"); chain.Status != "active" {
		t.Errorf("chain_2 status = %q, want active (2 facts, separate chain)", chain.Status)
	}
	evs := revelationEvents(t, e)
	if len(evs) != 1 || evs[0].ChainID != "chain_1" {
		t.Errorf("revelation events = %+v, want exactly one for chain_1", evs)
	}
}

// --- QA edge case: revelation threshold tuning knob ---

// Tuning knob: GIVEN the revelation_threshold lowered to 2
// (WithRevelationThreshold(2), GDD range 2-5), WHEN the second fact leading
// to the same entity is discovered, THEN the chain flips to
// revelation_ready — lower thresholds fire faster.
func TestOnFactDiscovered_ThresholdTwo_TwoFactsTrigger(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithRevelationThreshold(2))
	e.RegisterFact(factLeadingTo("fact_301", 42))
	e.RegisterFact(factLeadingTo("fact_302", 42))

	// Act
	e.OnFactDiscovered("fact_301", 5)
	e.OnFactDiscovered("fact_302", 8)

	// Assert
	chain, _ := e.GetChain("chain_1")
	if chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status = %q, want revelation_ready (threshold 2)", chain.Status)
	}
	evs := revelationEvents(t, e)
	if len(evs) != 1 || evs[0].FactCount != 2 {
		t.Errorf("revelation events = %+v, want one with FactCount 2", evs)
	}
}

// Tuning knob edge (lower clamp): GIVEN a threshold requested below the GDD
// range (1), WHEN the engine is constructed, THEN the knob clamps to 2 — a
// single fact never triggers a revelation.
func TestWithRevelationThreshold_BelowRange_ClampedToTwo(t *testing.T) {
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithRevelationThreshold(1))
	e.RegisterFact(factLeadingTo("fact_311", 42))
	e.OnFactDiscovered("fact_311", 5)
	if chain, _ := e.GetChain("chain_1"); chain.Status != "active" {
		t.Errorf("chain status with 1 fact = %q, want active (threshold clamped to 2)", chain.Status)
	}
	e.RegisterFact(factLeadingTo("fact_312", 42))
	e.OnFactDiscovered("fact_312", 8)
	if chain, _ := e.GetChain("chain_1"); chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status with 2 facts = %q, want revelation_ready (clamped threshold 2)", chain.Status)
	}
}

// Tuning knob edge (upper clamp): GIVEN a threshold requested above the GDD
// range (9), WHEN the engine is constructed, THEN the knob clamps to 5 —
// four facts sharing a target keep the chain active, the fifth flips it.
func TestWithRevelationThreshold_AboveRange_ClampedToFive(t *testing.T) {
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithRevelationThreshold(9))
	for i := 0; i < 4; i++ {
		factID := fmt.Sprintf("fact_clamp_%d", i)
		e.RegisterFact(factLeadingTo(factID, 42))
		e.OnFactDiscovered(factID, entity.GameEntityID(5+i))
	}
	if chain, _ := e.GetChain("chain_1"); chain.Status != "active" {
		t.Fatalf("chain status after 4 facts = %q, want active (threshold clamped to 5)", chain.Status)
	}
	e.RegisterFact(factLeadingTo("fact_clamp_4", 42))
	e.OnFactDiscovered("fact_clamp_4", 30)
	if chain, _ := e.GetChain("chain_1"); chain.Status != thread_investigation.ChainStatusRevelationReady {
		t.Errorf("chain status after 5 facts = %q, want revelation_ready (clamped threshold 5)", chain.Status)
	}
}

// --- AC-3: NPC death auto-resolution ---

// AC-3: GIVEN an active thread targeting entity 42, WHEN
// OnEntityDestroyed(42, "combat_death") fires, THEN the thread auto-resolves
// as dead_end, moves to ResolvedThreads with the dead-end marker, and
// EventThreadResolved is emitted with Resolution dead_end and Reason
// "combat_death".
func TestOnEntityDestroyed_TargetDies_ThreadAutoResolvesDeadEnd(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_501", 42))
	e.OnFactDiscovered("fact_501", 5)
	mustActiveThreads(t, e, 1)

	// Act
	e.OnEntityDestroyed(42, "combat_death")

	// Assert — thread moved to history with the dead-end marker.
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 {
		t.Fatalf("resolved threads = %d, want 1", len(resolved))
	}
	got := resolved[0]
	if got.ID != "thread_fact_501_42" || got.Status != thread_investigation.ThreadDeadEnd {
		t.Errorf("resolved thread = %+v, want thread_fact_501_42 with Status dead_end", got)
	}

	// Assert — event carries the death reason.
	evs := threadResolvedEvents(t, e)
	if len(evs) != 1 {
		t.Fatalf("thread.resolved events = %d, want 1", len(evs))
	}
	ev := evs[0]
	if ev.ThreadID != "thread_fact_501_42" || ev.Resolution != thread_investigation.ThreadDeadEnd || ev.Reason != "combat_death" {
		t.Errorf("event = %+v, want thread_fact_501_42 dead_end with Reason combat_death", ev)
	}
}

// QA edge case: GIVEN several active threads targeting the same NPC, WHEN
// the NPC is destroyed, THEN every one of them resolves as dead_end, while
// threads targeting other entities stay active.
func TestOnEntityDestroyed_MultipleThreadsSameNPC_AllResolve(t *testing.T) {
	// Arrange — two facts leading to 42, one to 99.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_511", 42))
	e.RegisterFact(factLeadingTo("fact_512", 42))
	e.RegisterFact(factLeadingTo("fact_513", 99))
	e.OnFactDiscovered("fact_511", 5)
	e.OnFactDiscovered("fact_512", 8)
	e.OnFactDiscovered("fact_513", 9)
	mustActiveThreads(t, e, 3)

	// Act
	e.OnEntityDestroyed(42, "combat_death")

	// Assert — the two 42-threads resolved, the 99-thread untouched.
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_fact_513_99" {
		t.Errorf("remaining thread = %+v, want thread_fact_513_99", threads[0])
	}
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 2 {
		t.Fatalf("resolved threads = %d, want 2", len(resolved))
	}
	evs := threadResolvedEvents(t, e)
	if len(evs) != 2 {
		t.Fatalf("thread.resolved events = %d, want 2", len(evs))
	}
	for _, ev := range evs {
		if ev.Resolution != thread_investigation.ThreadDeadEnd || ev.Reason != "combat_death" {
			t.Errorf("event = %+v, want dead_end with Reason combat_death", ev)
		}
	}
}

// QA edge case: GIVEN a thread already resolved (in history), WHEN the
// target NPC is destroyed, THEN nothing happens — no duplicate resolution,
// no event.
func TestOnEntityDestroyed_ThreadAlreadyResolved_NoOp(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_521", 42))
	e.OnFactDiscovered("fact_521", 5)
	e.ResolveThread("thread_fact_521_42", thread_investigation.ThreadResolved)
	if evs := threadResolvedEvents(t, e); len(evs) != 1 {
		t.Fatalf("thread.resolved events after manual resolve = %d, want 1", len(evs))
	}

	// Act
	e.OnEntityDestroyed(42, "combat_death")

	// Assert — unchanged.
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 1 {
		t.Errorf("resolved threads = %d, want 1 (no duplicate)", len(resolved))
	}
	if evs := threadResolvedEvents(t, e); len(evs) != 0 {
		t.Errorf("thread.resolved events = %d, want 0 (already resolved)", len(evs))
	}
}

// --- AC-4: thread resolution — discovered ---

// AC-4: GIVEN an active thread, WHEN ResolveThread is called with
// ThreadResolved (the player followed the lead to its end), THEN the thread
// resolves to history with Status resolved, and EventThreadResolved is
// emitted with Resolution resolved.
func TestResolveThread_Discovered_MovesToHistory(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_601", 42))
	e.OnFactDiscovered("fact_601", 5)
	mustActiveThreads(t, e, 1)

	// Act
	e.ResolveThread("thread_fact_601_42", thread_investigation.ThreadResolved)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 || resolved[0].Status != thread_investigation.ThreadResolved {
		t.Fatalf("resolved threads = %+v, want one thread with Status resolved", resolved)
	}
	evs := threadResolvedEvents(t, e)
	if len(evs) != 1 || evs[0].ThreadID != "thread_fact_601_42" || evs[0].Resolution != thread_investigation.ThreadResolved {
		t.Errorf("events = %+v, want one thread.resolved with Resolution resolved", evs)
	}
}

// QA edge cases: GIVEN a thread that is already resolved, or a thread ID
// that does not exist, WHEN ResolveThread is called, THEN it is a no-op —
// no duplicate history entry and no event.
func TestResolveThread_AlreadyResolvedOrUnknown_NoOp(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_611", 42))
	e.OnFactDiscovered("fact_611", 5)
	e.ResolveThread("thread_fact_611_42", thread_investigation.ThreadResolved)
	if evs := threadResolvedEvents(t, e); len(evs) != 1 {
		t.Fatalf("events after first resolve = %d, want 1", len(evs))
	}

	// Act — resolve again, and resolve a thread that never existed.
	e.ResolveThread("thread_fact_611_42", thread_investigation.ThreadDeadEnd)
	e.ResolveThread("thread_nonexistent_99", thread_investigation.ThreadResolved)

	// Assert — unchanged.
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 1 {
		t.Errorf("resolved threads = %d, want 1", len(resolved))
	}
	if evs := threadResolvedEvents(t, e); len(evs) != 0 {
		t.Errorf("events = %d, want 0 (no-ops)", len(evs))
	}
}

// --- AC-5: thread resolution — dead_end (manual) ---

// AC-5: GIVEN an active thread, WHEN ResolveThread is called with
// ThreadDeadEnd (manual — the player marks the lead unresolvable), THEN the
// thread moves to history with the dead-end marker and the event carries no
// reason (manual resolutions do not supply one).
func TestResolveThread_DeadEndManual_MovesToHistoryWithMarker(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_621", 42))
	e.OnFactDiscovered("fact_621", 5)

	// Act
	e.ResolveThread("thread_fact_621_42", thread_investigation.ThreadDeadEnd)

	// Assert
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 || resolved[0].Status != thread_investigation.ThreadDeadEnd {
		t.Fatalf("resolved threads = %+v, want one dead_end thread", resolved)
	}
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	evs := threadResolvedEvents(t, e)
	if len(evs) != 1 || evs[0].Resolution != thread_investigation.ThreadDeadEnd || evs[0].Reason != "" {
		t.Errorf("events = %+v, want one dead_end event with empty Reason (manual)", evs)
	}
}

// QA edge case (CR-4 reactivation): GIVEN a thread resolved as dead_end,
// WHEN new evidence emerges — a NEW fact with a lead to the same entity —
// THEN a fresh thread to that entity appears in ActiveThreads while the
// dead_end thread stays in history (deterministic IDs tie threads to their
// source facts; the old thread cannot be resurrected).
func TestResolveThread_DeadEnd_NewEvidence_ReactivatesAsNewThread(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_631", 42))
	e.OnFactDiscovered("fact_631", 5)
	e.ResolveThread("thread_fact_631_42", thread_investigation.ThreadDeadEnd)
	_ = threadResolvedEvents(t, e)

	// Act — a different fact leading to the same entity.
	e.RegisterFact(factLeadingTo("fact_632", 42))
	e.OnFactDiscovered("fact_632", 10)

	// Assert — a new active thread; the old one still in history.
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_fact_632_42" {
		t.Errorf("new thread = %+v, want thread_fact_632_42", threads[0])
	}
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 1 {
		t.Errorf("resolved threads = %d, want the original dead_end still in history", len(resolved))
	}
}

// --- QA edge case: prioritized status ---

// ThreadPrioritized (Story 004 resolution state machine): GIVEN an active
// thread, WHEN ResolveThread is called with ThreadPrioritized, THEN the
// thread stays in ActiveThreads with Status prioritized and EventThreadResolved
// is emitted — pinning is not a resolution, the lead remains open. WHEN the
// pinned thread's target NPC dies, THEN it still auto-resolves as dead_end.
func TestResolveThread_Prioritized_StaysActive(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_641", 42))
	e.OnFactDiscovered("fact_641", 5)

	// Act
	e.ResolveThread("thread_fact_641_42", thread_investigation.ThreadPrioritized)

	// Assert — pinned, still active, event emitted.
	threads := mustActiveThreads(t, e, 1)
	if threads[0].Status != thread_investigation.ThreadPrioritized {
		t.Errorf("thread status = %q, want prioritized", threads[0].Status)
	}
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 0 {
		t.Errorf("resolved threads = %d, want 0 (pin is not a resolution)", len(resolved))
	}
	evs := threadResolvedEvents(t, e)
	if len(evs) != 1 || evs[0].Resolution != thread_investigation.ThreadPrioritized {
		t.Errorf("events = %+v, want one prioritized event", evs)
	}

	// Act — the pinned thread's target NPC dies.
	e.OnEntityDestroyed(42, "combat_death")

	// Assert — the pinned thread resolves as dead_end (it was still active).
	threads = mustActiveThreads(t, e, 0)
	if resolved := e.Knowledge().ResolvedThreads; len(resolved) != 1 {
		t.Errorf("resolved threads = %d, want 1 (pinned thread resolved on death)", len(resolved))
	}
	evs = threadResolvedEvents(t, e)
	if len(evs) != 1 || evs[0].Resolution != thread_investigation.ThreadDeadEnd || evs[0].Reason != "combat_death" {
		t.Errorf("death events = %+v, want one dead_end with Reason combat_death", evs)
	}
}

// QA edge case: GIVEN an invalid resolution value, WHEN ResolveThread is
// called, THEN it is a no-op — the thread keeps its status and no event is
// emitted.
func TestResolveThread_InvalidResolution_NoOp(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factLeadingTo("fact_651", 42))
	e.OnFactDiscovered("fact_651", 5)

	// Act
	e.ResolveThread("thread_fact_651_42", thread_investigation.ThreadStatus("banana"))

	// Assert
	threads := mustActiveThreads(t, e, 1)
	if threads[0].Status != thread_investigation.ThreadActive {
		t.Errorf("thread status = %q, want active", threads[0].Status)
	}
	if evs := threadResolvedEvents(t, e); len(evs) != 0 {
		t.Errorf("events = %d, want 0", len(evs))
	}
}

// Performance (QA criterion): Step 5 adds minimal overhead — the revelation
// check runs only over chains touched by the discovery and early-returns on
// the status guard once a chain is revelation_ready. With 100 facts sharing
// one chain, steady-state idempotent re-discovery stays well under 1ms per
// OnFactDiscovered call (ADR-0010 guardrail: full OnFactDiscovered <5ms).
//
// Go benchmarks report the mean ns/op, not p95; the actual cost is
// microsecond-scale, so the mean-is-not-p95 gap is immaterial (same note as
// the Story 001-003 benchmarks).
func BenchmarkOnFactDiscovered_RevelationCheck(b *testing.B) {
	e := thread_investigation.NewEngine(fixedClock())

	// Pre-load 100 facts, each creating a thread and touching chain_1.
	for i := 0; i < 100; i++ {
		factID := fmt.Sprintf("pre_fact_%03d", i)
		e.RegisterFact(factLeadingTo(factID, 42))
		e.OnFactDiscovered(factID, 3000+entity.GameEntityID(i))
	}
	_ = e.DrainChainEvents()

	// Steady state: the chain is revelation_ready, so the Step 5 check
	// early-returns on the spam-prevention guard on every call.
	e.RegisterFact(factLeadingTo("new_fact", 42))
	e.OnFactDiscovered("new_fact", 9999)
	_ = e.DrainChainEvents()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.OnFactDiscovered("new_fact", 9999)
	}
	b.StopTimer()

	if perOp := b.Elapsed() / time.Duration(b.N); perOp > time.Millisecond {
		b.Fatalf("OnFactDiscovered per call = %v, want < 1ms", perOp)
	}
}

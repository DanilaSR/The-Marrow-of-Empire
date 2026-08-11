// Unit tests for Story 003: Contradiction Detection.
//
// Evidence path: production/epics/thread-investigation/story-003-contradiction-detection.md
// Tests map to the acceptance criteria (AC-1 .. AC-4) plus the QA edge
// cases (self-contradiction, circular contradictions, multiple contradicts
// entries, RUMOR confidence) and the performance criterion, verifying
// TR-thread-003 (contradiction detection per ADR-0010 Step 4). All tests
// are deterministic: the clock is injected with a fixed value, no
// randomness, no time-dependent assertions, no external I/O.
//
// Shared helpers (fixedClock, namer, mustActiveThreads, factWithRefs)
// live in thread_creation_test.go and auto_chaining_test.go — same package.
package thread_investigation_test

import (
	"fmt"
	"testing"
	"time"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
	"marrow-of-empire/thread_investigation"
)

// factWithContradicts builds a knowledge.Fact with contradicts entries,
// optional references, content, and fact type.
func factWithContradicts(factID string, contradicts []string, refs []entity.GameEntityID, content string, typ knowledge.FactType) knowledge.Fact {
	return knowledge.Fact{FactID: factID, Contradicts: contradicts, References: refs, Content: content, Type: typ}
}

// contradictionEvents drains the chain event queue and returns only the
// contradiction.detected events, in FIFO order.
func contradictionEvents(t *testing.T, e *thread_investigation.Engine) []thread_investigation.EventContradictionDetected {
	t.Helper()
	events := e.DrainChainEvents()
	var out []thread_investigation.EventContradictionDetected
	for _, ev := range events {
		if ev.Type != thread_investigation.EventContradictionDetectedType {
			continue
		}
		out = append(out, ev.Data.(thread_investigation.EventContradictionDetected))
	}
	return out
}

// --- AC-1: contradiction detected when both facts discovered ---

// AC-1: GIVEN a discovered fact fact_A (from NPC 5) and a new fact fact_B
// (from NPC 8) with contradicts: ["fact_A"], WHEN fact_B is discovered,
// THEN a contradiction thread is created with the contradiction topic, both
// facts are flagged, and EventContradictionDetected is emitted with
// Confidence "high".
func TestOnFactDiscovered_ContradictsDiscoveredFact_CreatesThreadFlagsAndEmits(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithNamer(namer{5: "Orin", 8: "Greta"}))
	e.RegisterFact(knowledge.Fact{FactID: "fact_A", Content: "Aldric died in the mine collapse"})
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, nil, "Aldric left the city alive", knowledge.FactType_Fact))

	// Act — first discovery creates nothing (no leads, no contradiction yet).
	e.OnFactDiscovered("fact_A", 5)
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Fatalf("threads after first fact = %d, want 0", len(threads))
	}
	e.OnFactDiscovered("fact_B", 8)

	// Assert — contradiction thread with the high-confidence topic.
	threads := mustActiveThreads(t, e, 1)
	got := threads[0]
	if got.ID != "thread_contradiction_fact_B_fact_A" {
		t.Errorf("ID = %q, want %q", got.ID, "thread_contradiction_fact_B_fact_A")
	}
	if got.Source != "fact_B↔fact_A" {
		t.Errorf("Source = %q, want %q", got.Source, "fact_B↔fact_A")
	}
	if got.TargetEntity != 0 {
		t.Errorf("TargetEntity = %d, want 0 (open-ended contradiction)", got.TargetEntity)
	}
	wantTopic := "Resolve contradiction: Greta claims Aldric left the city alive. Orin claims Aldric died in the mine collapse."
	if got.Topic != wantTopic {
		t.Errorf("Topic = %q, want %q", got.Topic, wantTopic)
	}
	if got.Status != thread_investigation.ThreadActive || got.CreatedAt != 1250.5 {
		t.Errorf("Status/CreatedAt = %q/%v, want active/1250.5", got.Status, got.CreatedAt)
	}

	// Assert — both facts flagged, in detection order (new fact first).
	flagged := e.Knowledge().ContradictedFacts
	if len(flagged) != 2 || flagged[0] != "fact_B" || flagged[1] != "fact_A" {
		t.Errorf("ContradictedFacts = %v, want [fact_B fact_A]", flagged)
	}

	// Assert — event queued via DrainChainEvents with all fields.
	evs := contradictionEvents(t, e)
	if len(evs) != 1 {
		t.Fatalf("contradiction events = %d, want 1", len(evs))
	}
	ev := evs[0]
	if ev.FactA.FactID != "fact_B" || ev.FactB.FactID != "fact_A" {
		t.Errorf("event facts = %q ↔ %q, want fact_B ↔ fact_A", ev.FactA.FactID, ev.FactB.FactID)
	}
	if ev.SourceNPC_A != 8 || ev.SourceNPC_B != 5 {
		t.Errorf("event NPCs = (%d, %d), want (8, 5)", ev.SourceNPC_A, ev.SourceNPC_B)
	}
	if ev.Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", ev.Confidence, "high")
	}
	if ev.ThreadID != got.ID {
		t.Errorf("ThreadID = %q, want %q", ev.ThreadID, got.ID)
	}
}

// --- AC-2: latent contradiction (other fact not discovered) ---

// AC-2: GIVEN fact_D with contradicts: ["fact_C"] where fact_C has NOT been
// discovered, WHEN fact_D is discovered, THEN no contradiction thread and no
// event are created — the contradiction is latent — and fact_D is recorded
// normally (not flagged). WHEN fact_C is discovered later, THEN the latent
// contradiction fires.
func TestOnFactDiscovered_ContradictsUndiscoveredFact_IsLatent(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithContradicts("fact_D", []string{"fact_C"}, nil, "The courier passed at noon", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_D", 4)

	// Assert — recorded normally, no contradiction artifacts.
	facts := e.Knowledge().DiscoveredFacts
	if len(facts) != 1 || facts[0].FactID != "fact_D" || facts[0].SourceNPC != 4 {
		t.Errorf("discovered facts = %+v, want fact_D from NPC 4", facts)
	}
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0", len(threads))
	}
	if flagged := e.Knowledge().ContradictedFacts; len(flagged) != 0 {
		t.Errorf("ContradictedFacts = %v, want none", flagged)
	}
	if evs := contradictionEvents(t, e); len(evs) != 0 {
		t.Errorf("contradiction events = %d, want 0", len(evs))
	}

	// Act — the other fact is discovered later.
	e.RegisterFact(factWithContradicts("fact_C", []string{"fact_D"}, nil, "The courier came at dawn", knowledge.FactType_Fact))
	e.OnFactDiscovered("fact_C", 6)

	// Assert — the latent contradiction fires on the second discovery.
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_contradiction_fact_C_fact_D" {
		t.Errorf("thread ID = %q, want %q", threads[0].ID, "thread_contradiction_fact_C_fact_D")
	}
	if evs := contradictionEvents(t, e); len(evs) != 1 {
		t.Errorf("contradiction events = %d, want 1 (latent fired)", len(evs))
	}
}

// --- AC-3: event emission on contradiction ---

// AC-3: GIVEN both contradicting facts are discovered, WHEN the
// contradiction is detected, THEN EventContradictionDetected is queued on
// the DrainChainEvents pipeline as a ChainEvent envelope with the
// "contradiction.detected" type, the captured game timestamp, and the typed
// payload carrying both fact IDs, both NPC IDs, and the thread ID.
func TestOnFactDiscovered_ContradictionEvent_QueuedViaChainEventsPipeline(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(knowledge.Fact{FactID: "fact_A"})
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, nil, "", knowledge.FactType_Fact))
	e.OnFactDiscovered("fact_A", 5)
	e.OnFactDiscovered("fact_B", 8)

	// Act
	events := e.DrainChainEvents()

	// Assert — the contradiction event is queued in the chain pipeline.
	var found *thread_investigation.ChainEvent
	for i := range events {
		if events[i].Type == thread_investigation.EventContradictionDetectedType {
			found = &events[i]
		}
	}
	if found == nil {
		t.Fatalf("no contradiction.detected in chain events %+v", events)
	}
	if found.Timestamp != 1250.5 {
		t.Errorf("Timestamp = %v, want 1250.5 (clock capture)", found.Timestamp)
	}
	ev, ok := found.Data.(thread_investigation.EventContradictionDetected)
	if !ok {
		t.Fatalf("Data = %T, want EventContradictionDetected", found.Data)
	}
	if ev.FactA.FactID != "fact_B" || ev.FactB.FactID != "fact_A" {
		t.Errorf("fact IDs = (%q, %q), want (fact_B, fact_A)", ev.FactA.FactID, ev.FactB.FactID)
	}
	if ev.SourceNPC_A != 8 || ev.SourceNPC_B != 5 {
		t.Errorf("NPC IDs = (%d, %d), want (8, 5)", ev.SourceNPC_A, ev.SourceNPC_B)
	}
	if ev.ThreadID != "thread_contradiction_fact_B_fact_A" {
		t.Errorf("ThreadID = %q, want %q", ev.ThreadID, "thread_contradiction_fact_B_fact_A")
	}
}

// --- AC-4: RUMOR lowers confidence ---

// AC-4: GIVEN fact_E is a RUMOR (ADR-0005) and fact_F contradicts it, WHEN
// fact_F is discovered, THEN the contradiction is flagged low confidence,
// the event carries Confidence "low", and the thread topic notes "one
// account may be a rumor".
func TestOnFactDiscovered_RumorContradiction_LowConfidence(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithNamer(namer{2: "Mira", 9: "Ivo"}))
	e.RegisterFact(factWithContradicts("fact_E", nil, nil, "The vault key is under the altar", knowledge.FactType_Rumor))
	e.RegisterFact(factWithContradicts("fact_F", []string{"fact_E"}, nil, "The vault key was melted down", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_E", 2)
	e.OnFactDiscovered("fact_F", 9)

	// Assert — event carries low confidence; topic flags the rumor.
	evs := contradictionEvents(t, e)
	if len(evs) != 1 {
		t.Fatalf("contradiction events = %d, want 1", len(evs))
	}
	if evs[0].Confidence != "low" {
		t.Errorf("Confidence = %q, want %q", evs[0].Confidence, "low")
	}
	thread := mustActiveThreads(t, e, 1)[0]
	wantTopic := "Possible contradiction: Ivo claims The vault key was melted down, but Mira says The vault key is under the altar — one account may be a rumor."
	if thread.Topic != wantTopic {
		t.Errorf("Topic = %q, want %q", thread.Topic, wantTopic)
	}
}

// AC-4 edge: GIVEN BOTH contradicting facts are RUMORs, WHEN the second is
// discovered, THEN confidence is still "low".
func TestOnFactDiscovered_BothFactsRumors_StillLowConfidence(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithContradicts("fact_E", nil, nil, "rumor one", knowledge.FactType_Rumor))
	e.RegisterFact(factWithContradicts("fact_F", []string{"fact_E"}, nil, "rumor two", knowledge.FactType_Rumor))

	// Act
	e.OnFactDiscovered("fact_E", 2)
	e.OnFactDiscovered("fact_F", 9)

	// Assert
	evs := contradictionEvents(t, e)
	if len(evs) != 1 || evs[0].Confidence != "low" {
		t.Errorf("events = %+v, want one with Confidence low", evs)
	}
}

// --- QA edge cases ---

// QA edge case: GIVEN a fact whose contradicts entry lists itself, WHEN it
// is discovered, THEN the self-contradiction is rejected — no thread, no
// event, no flag.
func TestOnFactDiscovered_SelfContradiction_Rejected(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithContradicts("fact_S", []string{"fact_S"}, nil, "I contradict myself", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_S", 1)

	// Assert
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Errorf("active threads = %d, want 0 (self-contradiction rejected)", len(threads))
	}
	if flagged := e.Knowledge().ContradictedFacts; len(flagged) != 0 {
		t.Errorf("ContradictedFacts = %v, want none", flagged)
	}
	if evs := contradictionEvents(t, e); len(evs) != 0 {
		t.Errorf("contradiction events = %d, want 0", len(evs))
	}
}

// QA edge case (forward-only per ADR-0010, confirmed): GIVEN fact_A
// contradicts fact_B AND fact_B contradicts fact_A, WHEN both are
// discovered, THEN the second discovery fires its direction (B↔A) — the
// first direction was latent. WHEN the first fact is re-discovered, THEN
// the first direction (A↔B) fires too: both directions fire, each exactly
// once, with distinct threads and events, and both facts are flagged once.
func TestOnFactDiscovered_CircularContradictions_BothFire(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithContradicts("fact_A", []string{"fact_B"}, nil, "A", knowledge.FactType_Fact))
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, nil, "B", knowledge.FactType_Fact))

	// Act — first discovery is latent (fact_B unknown).
	e.OnFactDiscovered("fact_A", 1)
	if threads := e.GetActiveThreads(); len(threads) != 0 {
		t.Fatalf("threads after fact_A = %d, want 0 (latent)", len(threads))
	}

	// Act — second discovery fires the B↔A direction.
	e.OnFactDiscovered("fact_B", 2)
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_contradiction_fact_B_fact_A" {
		t.Errorf("thread ID = %q, want %q (B↔A fires on second discovery)", threads[0].ID, "thread_contradiction_fact_B_fact_A")
	}
	evs := contradictionEvents(t, e)
	if len(evs) != 1 || evs[0].ThreadID != "thread_contradiction_fact_B_fact_A" {
		t.Fatalf("contradiction events = %+v, want one B↔A event", evs)
	}

	// Act — re-discovering the first fact fires the A↔B direction.
	e.OnFactDiscovered("fact_A", 1)
	threads = mustActiveThreads(t, e, 2)
	if threads[1].ID != "thread_contradiction_fact_A_fact_B" {
		t.Errorf("thread[1] ID = %q, want %q (A↔B fires on re-discovery)", threads[1].ID, "thread_contradiction_fact_A_fact_B")
	}
	evs = contradictionEvents(t, e)
	if len(evs) != 1 || evs[0].ThreadID != "thread_contradiction_fact_A_fact_B" {
		t.Errorf("contradiction events = %+v, want one A↔B event", evs)
	}

	// Assert — both facts flagged once each.
	flagged := e.Knowledge().ContradictedFacts
	if len(flagged) != 2 {
		t.Errorf("ContradictedFacts = %v, want both facts flagged once", flagged)
	}

	// Act — further re-discovery fires nothing new (idempotent).
	e.OnFactDiscovered("fact_B", 2)
	if threads := e.GetActiveThreads(); len(threads) != 2 {
		t.Errorf("threads after re-discovery = %d, want 2", len(threads))
	}
	if evs := contradictionEvents(t, e); len(evs) != 0 {
		t.Errorf("contradiction events after re-discovery = %d, want 0", len(evs))
	}
}

// QA edge case: GIVEN a fact with several contradicts entries, some of
// whose targets are discovered and some not, WHEN it is discovered, THEN
// one thread and one event fire per discovered target; undiscovered targets
// stay latent.
func TestOnFactDiscovered_MultipleContradicts_MixedDiscovery(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(knowledge.Fact{FactID: "fact_known1", Content: "known one"})
	e.RegisterFact(knowledge.Fact{FactID: "fact_unknown"})
	e.RegisterFact(knowledge.Fact{FactID: "fact_known2", Content: "known two"})
	e.RegisterFact(factWithContradicts("fact_multi",
		[]string{"fact_known1", "fact_unknown", "fact_known2"},
		nil, "multi", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_known1", 3)
	e.OnFactDiscovered("fact_known2", 5)
	e.OnFactDiscovered("fact_multi", 7)

	// Assert — one contradiction thread per discovered target, none for the
	// undiscovered fact_unknown, in contradicts-entry order.
	threads := mustActiveThreads(t, e, 2)
	wantIDs := []string{"thread_contradiction_fact_multi_fact_known1", "thread_contradiction_fact_multi_fact_known2"}
	for i, want := range wantIDs {
		if threads[i].ID != want {
			t.Errorf("thread[%d] ID = %q, want %q", i, threads[i].ID, want)
		}
	}
	evs := contradictionEvents(t, e)
	if len(evs) != 2 {
		t.Fatalf("contradiction events = %d, want 2", len(evs))
	}
	if evs[0].FactB.FactID != "fact_known1" || evs[1].FactB.FactID != "fact_known2" {
		t.Errorf("event FactB order = %q, %q, want fact_known1, fact_known2", evs[0].FactB.FactID, evs[1].FactB.FactID)
	}
	flagged := e.Knowledge().ContradictedFacts
	if len(flagged) != 3 {
		t.Errorf("ContradictedFacts = %v, want 3 (fact_multi + both known facts)", flagged)
	}
}

// Idempotency: GIVEN a detected contradiction, WHEN the triggering fact is
// discovered again, THEN no duplicate thread, event, or flag is created.
func TestOnFactDiscovered_ContradictionReDiscovery_NoDuplicate(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(knowledge.Fact{FactID: "fact_A"})
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, nil, "B", knowledge.FactType_Fact))
	e.OnFactDiscovered("fact_A", 5)
	e.OnFactDiscovered("fact_B", 8)
	if evs := contradictionEvents(t, e); len(evs) != 1 {
		t.Fatalf("contradiction events after first detection = %d, want 1", len(evs))
	}

	// Act — re-discover the triggering fact.
	e.OnFactDiscovered("fact_B", 8)

	// Assert — unchanged.
	threads := mustActiveThreads(t, e, 1)
	if threads[0].ID != "thread_contradiction_fact_B_fact_A" {
		t.Errorf("thread ID = %q, want the original contradiction thread", threads[0].ID)
	}
	if flagged := e.Knowledge().ContradictedFacts; len(flagged) != 2 {
		t.Errorf("ContradictedFacts = %v, want the original 2", flagged)
	}
	if evs := contradictionEvents(t, e); len(evs) != 0 {
		t.Errorf("contradiction events = %d, want 0 (no re-fire)", len(evs))
	}
}

// Contradiction threads go through the same chain-assignment path as lead
// threads (Story 002 Step 3): the thread's entity set is the union of both
// facts' references, so a contradiction between facts referencing entity 42
// joins the chain that already covers 42, and both facts are absorbed into
// that chain.
func TestOnFactDiscovered_ContradictionThread_IsChainAssigned(t *testing.T) {
	// Arrange — fact_A creates chain_1 around entity 42.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_A", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "the mine"}))
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, []entity.GameEntityID{42}, "B", knowledge.FactType_Fact))

	// Act
	e.OnFactDiscovered("fact_A", 5)
	e.OnFactDiscovered("fact_B", 8)

	// Assert — the contradiction thread is assigned to chain_1.
	threads := mustActiveThreads(t, e, 2)
	if threads[1].ID != "thread_contradiction_fact_B_fact_A" || threads[1].ChainID != "chain_1" {
		t.Errorf("contradiction thread = %+v, want thread_contradiction_fact_B_fact_A in chain_1", threads[1])
	}
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the chain")
	}
	if len(chain.Threads) != 2 || chain.Threads[1] != "thread_contradiction_fact_B_fact_A" {
		t.Errorf("chain threads = %v, want the contradiction thread absorbed", chain.Threads)
	}
	if len(chain.Facts) != 2 || chain.Facts[1] != "fact_B" {
		t.Errorf("chain facts = %v, want fact_B absorbed", chain.Facts)
	}

	// Assert — event ordering: thread.chained, then contradiction.detected.
	events := e.DrainChainEvents()
	if len(events) != 3 {
		t.Fatalf("chain events = %d, want 3 (created, chained, contradiction)", len(events))
	}
	if events[1].Type != thread_investigation.EventThreadChainedType {
		t.Errorf("events[1].Type = %q, want thread.chained (contradiction thread chain-assigned)", events[1].Type)
	}
	if events[2].Type != thread_investigation.EventContradictionDetectedType {
		t.Errorf("events[2].Type = %q, want contradiction.detected", events[2].Type)
	}
}

// Edge: GIVEN contradicting facts with empty Content, WHEN the contradiction
// is detected, THEN the thread topic falls back to the fact IDs as the
// summaries (same fallback pattern as chain summaries, Story 002).
func TestOnFactDiscovered_ContradictionTopic_EmptyContent_FallsBackToFactIDs(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(knowledge.Fact{FactID: "fact_A"})
	e.RegisterFact(factWithContradicts("fact_B", []string{"fact_A"}, nil, "", knowledge.FactType_Fact))
	e.OnFactDiscovered("fact_A", 5)
	e.OnFactDiscovered("fact_B", 8)

	// Assert
	thread := mustActiveThreads(t, e, 1)[0]
	wantTopic := "Resolve contradiction: [entity 8] claims fact_B. [entity 5] claims fact_A."
	if thread.Topic != wantTopic {
		t.Errorf("Topic = %q, want %q", thread.Topic, wantTopic)
	}
}

// Performance (QA criterion): Step 4 adds minimal overhead — the
// contradiction check scans a fact with several contradicts entries against
// 100 discovered facts and completes well under 1ms per OnFactDiscovered
// call (ADR-0010 guardrail: full OnFactDiscovered <5ms; Story requirement:
// Step 4 <1ms for typical input).
//
// Go benchmarks report the mean ns/op, not p95; the actual cost is
// microsecond-scale, so the mean-is-not-p95 gap is immaterial (same note as
// the Story 001 and 002 benchmarks).
func BenchmarkOnFactDiscovered_ContradictionCheck(b *testing.B) {
	e := thread_investigation.NewEngine(fixedClock())

	// Pre-load 100 discovered facts, each contradicting its predecessor.
	for i := 0; i < 100; i++ {
		factID := fmt.Sprintf("pre_fact_%03d", i)
		var contradicts []string
		if i > 0 {
			contradicts = []string{fmt.Sprintf("pre_fact_%03d", i-1)}
		}
		e.RegisterFact(knowledge.Fact{FactID: factID, Contradicts: contradicts})
		e.OnFactDiscovered(factID, 3000+entity.GameEntityID(i))
	}

	// The measured fact contradicts three discovered facts.
	e.RegisterFact(knowledge.Fact{FactID: "new_fact", Contradicts: []string{"pre_fact_000", "pre_fact_050", "pre_fact_099"}})
	e.OnFactDiscovered("new_fact", 9999)

	// Steady-state: idempotent re-discovery exercises the Step 4 scan over
	// the contradicts entries on every call.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.OnFactDiscovered("new_fact", 9999)
	}
	b.StopTimer()

	if perOp := b.Elapsed() / time.Duration(b.N); perOp > time.Millisecond {
		b.Fatalf("OnFactDiscovered per call = %v, want < 1ms", perOp)
	}
}

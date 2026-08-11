// Unit tests for Story 002: Auto-Chaining & Entity Overlap.
//
// Evidence path: production/epics/thread-investigation/story-002-auto-chaining-entity-overlap.md
// Tests map to the acceptance criteria (AC-1 .. AC-4) plus the performance
// criterion and the QA edge cases, verifying TR-thread-002 (threads sharing
// entity references grouped into investigation chains; chains auto-merge on
// entity overlap) per ADR-0010 Step 3/3b. All tests are deterministic: the
// clock is injected with a fixed value, no randomness, no time-dependent
// assertions, no external I/O.
//
// Shared helpers (fixedClock, namer, mustActiveThreads, factWithLeads)
// live in thread_creation_test.go — same package.
package thread_investigation_test

import (
	"fmt"
	"testing"
	"time"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
	"marrow-of-empire/thread_investigation"
)

// factWithRefs builds a knowledge.Fact with entity references and optional
// leads_to hints. References are what cross-referencing and auto-chaining
// match on (Story 002).
func factWithRefs(factID string, refs []entity.GameEntityID, leads ...knowledge.Lead) knowledge.Fact {
	return knowledge.Fact{FactID: factID, References: refs, LeadsTo: leads}
}

// --- AC-1: entity overlap flagging ---

// AC-1: GIVEN two discovered facts from different NPCs that both reference
// entity 42, WHEN the second fact is discovered, THEN an overlap flag is
// raised naming both NPCs and the entity.
func TestOnFactDiscovered_EntityOverlap_DifferentNPCs_FlagsOverlap(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{42}))

	// Act — first discovery raises nothing; the second one overlaps.
	e.OnFactDiscovered("fact_001", 7)
	if flags := e.GetOverlapFlags(); len(flags) != 0 {
		t.Fatalf("overlap flags after first fact = %d, want 0", len(flags))
	}
	e.OnFactDiscovered("fact_002", 3)

	// Assert
	flags := e.GetOverlapFlags()
	if len(flags) != 1 {
		t.Fatalf("overlap flags = %d, want 1", len(flags))
	}
	got := flags[0]
	if got.Entity != 42 {
		t.Errorf("Entity = %d, want 42", got.Entity)
	}
	if got.NPC_A != 7 || got.NPC_B != 3 {
		t.Errorf("NPC pair = (%d, %d), want (7, 3) — older NPC first", got.NPC_A, got.NPC_B)
	}
	if want := "Both [entity 7] and [entity 3] mentioned [entity 42]."; got.Message != want {
		t.Errorf("Message = %q, want %q", got.Message, want)
	}
}

// AC-1 (name resolution): with an EntityNamer the flag message uses the
// NPC and entity display names instead of bracketed IDs.
func TestOnFactDiscovered_EntityOverlap_WithNamer_UsesDisplayNames(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithNamer(namer{7: "Orin", 3: "Greta", 42: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{42}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 3)

	// Assert
	flags := e.GetOverlapFlags()
	if len(flags) != 1 {
		t.Fatalf("overlap flags = %d, want 1", len(flags))
	}
	if want := "Both Orin and Greta mentioned the old mine."; flags[0].Message != want {
		t.Errorf("Message = %q, want %q", flags[0].Message, want)
	}
}

// Edge (AC-1): facts from the SAME NPC referencing the same entity never
// flag — one NPC knowing several facts about an entity is not a
// cross-reference. Flags are also deduplicated: two facts from NPC 3
// overlapping the same NPC 7 fact raise one flag, not two.
func TestOnFactDiscovered_EntityOverlap_SameNPCSkippedAndDeduplicated(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	for _, id := range []string{"fact_001", "fact_002", "fact_003", "fact_004"} {
		e.RegisterFact(factWithRefs(id, []entity.GameEntityID{42}))
	}

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 7) // same NPC as fact_001 — skipped

	// Assert
	if flags := e.GetOverlapFlags(); len(flags) != 0 {
		t.Fatalf("overlap flags after two same-NPC facts = %d, want 0", len(flags))
	}

	// Act — NPC 3's fact overlaps both NPC 7 facts.
	e.OnFactDiscovered("fact_003", 3)

	// Assert — one deduplicated flag for the (entity 42, NPC 7, NPC 3) pair.
	if flags := e.GetOverlapFlags(); len(flags) != 1 {
		t.Fatalf("overlap flags after NPC 3 fact = %d, want 1 (deduplicated)", len(flags))
	}

	// Act — another NPC 3 fact: same new NPC, already-flagged pair.
	e.OnFactDiscovered("fact_004", 3)

	// Assert — still exactly one flag; re-overlap adds nothing.
	if flags := e.GetOverlapFlags(); len(flags) != 1 {
		t.Fatalf("overlap flags after second NPC 3 fact = %d, want 1", len(flags))
	}
}

// Edge (AC-1): a new fact referencing several entities flags only the
// first overlapping entity per existing fact (Story 002 pseudo-code uses
// overlap[0], in the new fact's reference order).
func TestOnFactDiscovered_EntityOverlap_MultipleEntities_FlagsFirstOverlap(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{43, 42, 44}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 5)

	// Assert — only the 42 overlap is flagged, as the first overlapping
	// reference of the new fact.
	flags := e.GetOverlapFlags()
	if len(flags) != 1 {
		t.Fatalf("overlap flags = %d, want 1", len(flags))
	}
	if flags[0].Entity != 42 {
		t.Errorf("Entity = %d, want 42 (first overlap in reference order)", flags[0].Entity)
	}
}


	// GAP-1 regression (reversed NPC-pair dedup): GIVEN NPC A mentions entity
	// X, THEN NPC B mentions X (flag created), THEN NPC A mentions X again,
	// WHEN the third fact is discovered, THEN no duplicate flag is created.
	func TestOnFactDiscovered_EntityOverlap_ReversedPair_IsDeduplicated(t *testing.T) {
		e := thread_investigation.NewEngine(fixedClock(),
			thread_investigation.WithNamer(namer{3: "Greta", 7: "Orin"}))
		e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
			knowledge.Lead{EntityID: 42, TopicHint: "the old mine"}))
		e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{42},
			knowledge.Lead{EntityID: 42, TopicHint: "the mine"}))
		e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{42},
			knowledge.Lead{EntityID: 42, TopicHint: "the ore"}))
		e.OnFactDiscovered("fact_001", 3)
		e.OnFactDiscovered("fact_002", 7)
		e.OnFactDiscovered("fact_003", 3)
		flags := e.GetOverlapFlags()
		if len(flags) != 1 {
			t.Fatalf("overlap flags = %d, want 1 (reversed pair was deduplicated)", len(flags))
		}
		if flags[0].Entity != 42 || flags[0].NPC_A != 3 || flags[0].NPC_B != 7 {
			t.Errorf("flag = {Entity:%d A:%d B:%d}, want {42 3 7}", flags[0].Entity, flags[0].NPC_A, flags[0].NPC_B)
		}
	}

// --- AC-2: thread auto-assigned to existing chain ---

// AC-2: GIVEN an existing chain containing facts that reference entity 42,
// WHEN a new thread referencing entity 42 is created, THEN the thread is
// auto-assigned to that chain and EventThreadChained is emitted.
func TestOnFactDiscovered_ThreadAutoAssignedToExistingChain(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 43, TopicHint: "the old mine"}))

	// Act
	e.OnFactDiscovered("fact_001", 7) // no chain yet — creates chain_1
	e.OnFactDiscovered("fact_002", 3) // references 42 — assigned to chain_1

	// Assert — the new thread points at the existing chain.
	threads := mustActiveThreads(t, e, 2)
	if got, want := threads[1].ChainID, "chain_1"; got != want {
		t.Errorf("thread_fact_002_43 ChainID = %q, want %q", got, want)
	}

	// Assert — the chain absorbed the new fact and thread (deduplicated).
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the chain")
	}
	if len(chain.Facts) != 2 || chain.Facts[0] != "fact_001" || chain.Facts[1] != "fact_002" {
		t.Errorf("chain facts = %v, want [fact_001 fact_002]", chain.Facts)
	}
	if len(chain.Threads) != 2 || chain.Threads[1] != "thread_fact_002_43" {
		t.Errorf("chain threads = %v, want [thread_fact_001_42 thread_fact_002_43]", chain.Threads)
	}

	// Assert — event order and payloads: chain.created for the first
	// thread's chain, then thread.chained for the assignment.
	events := e.DrainChainEvents()
	if len(events) != 2 {
		t.Fatalf("chain events = %d, want 2", len(events))
	}
	if events[0].Type != thread_investigation.EventChainCreatedType {
		t.Errorf("events[0].Type = %q, want %q", events[0].Type, thread_investigation.EventChainCreatedType)
	}
	if events[0].Timestamp != 1250.5 {
		t.Errorf("events[0].Timestamp = %v, want 1250.5 (clock capture)", events[0].Timestamp)
	}
	created := events[0].Data.(thread_investigation.EventChainCreated)
	if created.ChainID != "chain_1" || created.FirstThreadID != "thread_fact_001_42" || created.Summary != "Ley-touched metal" {
		t.Errorf("EventChainCreated = %+v, want chain_1 / thread_fact_001_42 / summary", created)
	}
	if events[1].Type != thread_investigation.EventThreadChainedType {
		t.Errorf("events[1].Type = %q, want %q", events[1].Type, thread_investigation.EventThreadChainedType)
	}
	chained := events[1].Data.(thread_investigation.EventThreadChained)
	if chained.ThreadID != "thread_fact_002_43" || chained.ChainID != "chain_1" {
		t.Errorf("EventThreadChained = %+v, want thread_fact_002_43 in chain_1", chained)
	}

	// Assert — thread creation events drain separately (Story 001 API).
	if evs := e.DrainEvents(); len(evs) != 2 {
		t.Errorf("thread events = %d, want 2 (unchanged DrainEvents)", len(evs))
	}
}

// Edge (AC-2): a thread referencing several entities, only some of which
// are in the chain, is still assigned to the matching chain.
func TestOnFactDiscovered_ThreadPartialEntityMatch_StillAssigned(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{99, 42},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 3)

	// Assert — matched on entity 42 despite 99 and 43 being unmatched.
	threads := mustActiveThreads(t, e, 2)
	if threads[1].ChainID != "chain_1" {
		t.Errorf("ChainID = %q, want chain_1", threads[1].ChainID)
	}
	if chain, ok := e.GetChain("chain_1"); !ok || len(chain.Facts) != 2 {
		t.Errorf("chain_1 facts = %v, want both facts", chain.Facts)
	}
}

// Edge (AC-2): several threads created from one fact share the fact's
// references, so they all chain into the same chain.
func TestOnFactDiscovered_TwoLeadsFromOneFact_BothAssignedToSameChain(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 43, TopicHint: "the old mine"},
		knowledge.Lead{EntityID: 44, TopicHint: "Captain Veren"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 3)

	// Assert
	threads := mustActiveThreads(t, e, 3)
	for _, th := range threads {
		if th.ChainID != "chain_1" {
			t.Errorf("thread %s ChainID = %q, want chain_1", th.ID, th.ChainID)
		}
	}
	events := e.DrainChainEvents()
	if len(events) != 3 {
		t.Fatalf("chain events = %d, want 3 (created + 2 chained)", len(events))
	}
	if events[1].Type != thread_investigation.EventThreadChainedType || events[2].Type != thread_investigation.EventThreadChainedType {
		t.Errorf("events[1:3] = %q, %q, want two thread.chained", events[1].Type, events[2].Type)
	}
}

// --- AC-3: new chain for unmatched entity ---

// AC-3: GIVEN a newly created thread referencing an entity not in any
// existing chain, WHEN the thread is created, THEN a new chain is created
// containing that thread as its first thread and EventChainCreated is
// emitted with the auto-generated summary.
func TestOnFactDiscovered_UnmatchedEntity_CreatesNewChain(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{99},
		knowledge.Lead{EntityID: 99, TopicHint: "the hermit"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 3)

	// Assert — a second chain was created for the unmatched entity.
	chain, ok := e.GetChain("chain_2")
	if !ok {
		t.Fatal("GetChain(chain_2) = not found, want the new chain")
	}
	if chain.Summary != "the hermit" {
		t.Errorf("Summary = %q, want %q (first topic hint)", chain.Summary, "the hermit")
	}
	if len(chain.Facts) != 1 || chain.Facts[0] != "fact_002" {
		t.Errorf("facts = %v, want [fact_002]", chain.Facts)
	}
	if len(chain.Threads) != 1 || chain.Threads[0] != "thread_fact_002_99" {
		t.Errorf("threads = %v, want [thread_fact_002_99] as first thread", chain.Threads)
	}
	threads := mustActiveThreads(t, e, 2)
	if threads[1].ChainID != "chain_2" {
		t.Errorf("thread_fact_002_99 ChainID = %q, want chain_2", threads[1].ChainID)
	}

	// Assert — event payload names the new chain and its first thread.
	events := e.DrainChainEvents()
	if len(events) != 2 {
		t.Fatalf("chain events = %d, want 2", len(events))
	}
	created := events[1].Data.(thread_investigation.EventChainCreated)
	if created.ChainID != "chain_2" || created.FirstThreadID != "thread_fact_002_99" || created.Summary != "the hermit" {
		t.Errorf("EventChainCreated = %+v, want chain_2 / thread_fact_002_99 / the hermit", created)
	}
}

// Edge (AC-3): the chain summary falls back to the source fact ID when the
// fact's first lead carries no topic hint.
func TestOnFactDiscovered_ChainSummary_EmptyTopicHint_FallsBackToFactID(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{99},
		knowledge.Lead{EntityID: 99}))

	// Act
	e.OnFactDiscovered("fact_001", 3)

	// Assert
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the chain")
	}
	if chain.Summary != "fact_001" {
		t.Errorf("Summary = %q, want %q (fact ID fallback)", chain.Summary, "fact_001")
	}
}

// --- AC-4: chain auto-merge ---

// AC-4: GIVEN two chains that a connecting fact links (it references one
// entity of each), WHEN the overlap is detected, THEN the chains merge
// into one chain containing all facts and threads from both, all threads'
// ChainID point at the surviving chain, the old chain is archived (kept
// for investigation history), and EventChainsMerged is emitted.
func TestOnFactDiscovered_ChainsSharingEntity_AutoMerge(t *testing.T) {
	// Arrange — chain_1 about entity 42, chain_2 about entity 44.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Aldric's death"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{42, 44},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))

	// Act — the third fact references both entities, connecting the chains.
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 9)
	e.OnFactDiscovered("fact_003", 3)

	// Assert — the survivor (oldest, chain_1) holds every fact and thread.
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the surviving chain")
	}
	wantFacts := []string{"fact_001", "fact_003", "fact_002"}
	if len(chain.Facts) != 3 {
		t.Fatalf("survivor facts = %v, want %v", chain.Facts, wantFacts)
	}
	for i, fid := range wantFacts {
		if chain.Facts[i] != fid {
			t.Errorf("survivor facts[%d] = %q, want %q", i, chain.Facts[i], fid)
		}
	}
	wantThreads := []string{"thread_fact_001_42", "thread_fact_003_43", "thread_fact_002_44"}
	if len(chain.Threads) != 3 {
		t.Fatalf("survivor threads = %v, want %v", chain.Threads, wantThreads)
	}
	for i, tid := range wantThreads {
		if chain.Threads[i] != tid {
			t.Errorf("survivor threads[%d] = %q, want %q", i, chain.Threads[i], tid)
		}
	}
	if chain.Summary != "Aldric's death & the old mine" {
		t.Errorf("Summary = %q, want merged summary %q", chain.Summary, "Aldric's death & the old mine")
	}

	// Assert — every thread, including the merged chain's, points at the
	// survivor.
	for _, th := range e.GetActiveThreads() {
		if th.ChainID != "chain_1" {
			t.Errorf("thread %s ChainID = %q, want chain_1 after merge", th.ID, th.ChainID)
		}
	}

	// Assert — the old chain is archived, not deleted (investigation
	// history, ADR-0010).
	archived, ok := e.GetChain("chain_2")
	if !ok {
		t.Fatal("GetChain(chain_2) = not found, want the archived chain kept for history")
	}
	if archived.Status != thread_investigation.ChainStatusArchived {
		t.Errorf("chain_2 Status = %q, want %q", archived.Status, thread_investigation.ChainStatusArchived)
	}
	if len(archived.Facts) != 1 || archived.Facts[0] != "fact_002" {
		t.Errorf("archived facts = %v, want [fact_002] preserved", archived.Facts)
	}

	// Assert — event sequence: created, created, chained, merged.
	events := e.DrainChainEvents()
	if len(events) != 4 {
		t.Fatalf("chain events = %d, want 4", len(events))
	}
	chained := events[2].Data.(thread_investigation.EventThreadChained)
	if chained.ThreadID != "thread_fact_003_43" || chained.ChainID != "chain_1" {
		t.Errorf("EventThreadChained = %+v, want connecting thread in chain_1", chained)
	}
	if events[3].Type != thread_investigation.EventChainsMergedType {
		t.Fatalf("events[3].Type = %q, want %q", events[3].Type, thread_investigation.EventChainsMergedType)
	}
	merged := events[3].Data.(thread_investigation.EventChainsMerged)
	if merged.SurvivingChainID != "chain_1" {
		t.Errorf("SurvivingChainID = %q, want chain_1", merged.SurvivingChainID)
	}
	if len(merged.ArchivedChainIDs) != 1 || merged.ArchivedChainIDs[0] != "chain_2" {
		t.Errorf("ArchivedChainIDs = %v, want [chain_2]", merged.ArchivedChainIDs)
	}
	if merged.MergedSummary != "Aldric's death & the old mine" {
		t.Errorf("MergedSummary = %q, want %q", merged.MergedSummary, "Aldric's death & the old mine")
	}
}

// Edge (AC-4): more than two chains sharing entities through the
// connecting fact merge into a single chain in one step.
func TestOnFactDiscovered_ThreeChainsSharingEntity_MergeIntoOne(t *testing.T) {
	// Arrange — three chains about entities 42, 44, 46.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Aldric's death"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{46},
		knowledge.Lead{EntityID: 46, TopicHint: "Captain Veren"}))
	e.RegisterFact(factWithRefs("fact_004", []entity.GameEntityID{42, 44, 46},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 9)
	e.OnFactDiscovered("fact_003", 11)
	e.OnFactDiscovered("fact_004", 3)

	// Assert — one survivor, two archived, in creation order.
	chain, ok := e.GetChain("chain_1")
	if !ok {
		t.Fatal("GetChain(chain_1) = not found, want the survivor")
	}
	if len(chain.Facts) != 4 || len(chain.Threads) != 4 {
		t.Errorf("survivor facts/threads = %d/%d, want 4/4", len(chain.Facts), len(chain.Threads))
	}
	if got, _ := e.GetChain("chain_2"); got.Status != thread_investigation.ChainStatusArchived {
		t.Errorf("chain_2 Status = %q, want archived", got.Status)
	}
	if got, _ := e.GetChain("chain_3"); got.Status != thread_investigation.ChainStatusArchived {
		t.Errorf("chain_3 Status = %q, want archived", got.Status)
	}
	events := e.DrainChainEvents()
	if len(events) != 5 {
		t.Fatalf("chain events = %d, want 5", len(events))
	}
	merged := events[4].Data.(thread_investigation.EventChainsMerged)
	if merged.SurvivingChainID != "chain_1" {
		t.Errorf("SurvivingChainID = %q, want chain_1", merged.SurvivingChainID)
	}
	if len(merged.ArchivedChainIDs) != 2 || merged.ArchivedChainIDs[0] != "chain_2" || merged.ArchivedChainIDs[1] != "chain_3" {
		t.Errorf("ArchivedChainIDs = %v, want [chain_2 chain_3]", merged.ArchivedChainIDs)
	}
	if want := "Aldric's death & the old mine & Captain Veren"; merged.MergedSummary != want {
		t.Errorf("MergedSummary = %q, want %q", merged.MergedSummary, want)
	}
}

// Edge (AC-4): a merge is idempotent — an entity overlap that already
// merged the chains never re-merges or re-archives; later facts are simply
// absorbed by the survivor, and fresh unmatched entities keep creating new
// chains with incrementing IDs.
func TestOnFactDiscovered_MergeIdempotent_NoRepeatMerge(t *testing.T) {
	// Arrange — build and merge chain_1 (42) + chain_2 (44).
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Aldric's death"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{42, 44},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 9)
	e.OnFactDiscovered("fact_003", 3)
	e.DrainChainEvents() // discard the merge events

	// Act — a new fact referencing entity 44 of the merged chain.
	e.RegisterFact(factWithRefs("fact_004", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "again"}))
	e.OnFactDiscovered("fact_004", 5)

	// Assert — assigned to the survivor, no new merge event.
	events := e.DrainChainEvents()
	if len(events) != 1 {
		t.Fatalf("chain events = %d, want 1 (thread.chained only)", len(events))
	}
	if events[0].Type != thread_investigation.EventThreadChainedType {
		t.Errorf("event type = %q, want %q", events[0].Type, thread_investigation.EventThreadChainedType)
	}
	if chain, _ := e.GetChain("chain_1"); len(chain.Facts) != 4 {
		t.Errorf("survivor facts = %v, want 4 facts (fact_004 absorbed)", chain.Facts)
	}
	if chain, _ := e.GetChain("chain_2"); chain.Status != thread_investigation.ChainStatusArchived {
		t.Errorf("chain_2 Status = %q, want still archived (no re-merge)", chain.Status)
	}

	// Act — a brand-new unmatched entity keeps the chain counter going.
	e.RegisterFact(factWithRefs("fact_005", []entity.GameEntityID{70},
		knowledge.Lead{EntityID: 70, TopicHint: "fresh"}))
	e.OnFactDiscovered("fact_005", 6)

	// Assert
	if _, ok := e.GetChain("chain_3"); !ok {
		t.Error("GetChain(chain_3) = not found, want a fresh chain after the merge")
	}
}

// Edge (AC-4): resolved threads are re-pointed to the surviving chain too
// — the merge updates every thread wherever it lives.
func TestOnFactDiscovered_Merge_UpdatesResolvedThreadChainID(t *testing.T) {
	// Arrange — chain_1 (42) with thread t1; chain_2 (44) with thread t2.
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Aldric's death"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{42, 44},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 9)
	e.DrainChainEvents()

	// Act — resolve chain_2's thread, then connect the chains.
	e.ResolveThread("thread_fact_002_44", thread_investigation.ThreadResolved)
	e.OnFactDiscovered("fact_003", 3)

	// Assert — the resolved thread now points at the survivor.
	resolved := e.Knowledge().ResolvedThreads
	if len(resolved) != 1 {
		t.Fatalf("resolved threads = %d, want 1", len(resolved))
	}
	if resolved[0].ChainID != "chain_1" {
		t.Errorf("resolved thread ChainID = %q, want chain_1 after merge", resolved[0].ChainID)
	}
}

// Tuning knob (auto_chain_merge, default true): with merging disabled, the
// connecting thread is still assigned to the matching chain (Step 3c runs),
// but the chains stay separate — no EventChainsMerged is emitted.
func TestOnFactDiscovered_AutoChainMergeDisabled_ChainsStaySeparate(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock(),
		thread_investigation.WithAutoChainMerge(false))
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Aldric's death"}))
	e.RegisterFact(factWithRefs("fact_002", []entity.GameEntityID{44},
		knowledge.Lead{EntityID: 44, TopicHint: "the old mine"}))
	e.RegisterFact(factWithRefs("fact_003", []entity.GameEntityID{42, 44},
		knowledge.Lead{EntityID: 43, TopicHint: "the courier"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)
	e.OnFactDiscovered("fact_002", 9)
	e.OnFactDiscovered("fact_003", 3)

	// Assert — the connecting thread is chained, but no merge happened.
	events := e.DrainChainEvents()
	if len(events) != 3 {
		t.Fatalf("chain events = %d, want 3 (created, created, chained)", len(events))
	}
	for i, ev := range events {
		if ev.Type == thread_investigation.EventChainsMergedType {
			t.Errorf("events[%d] = %q, want no chain.merged with knob disabled", i, ev.Type)
		}
	}
	if chain, _ := e.GetChain("chain_2"); chain.Status == thread_investigation.ChainStatusArchived {
		t.Error("chain_2 archived despite auto_chain_merge=false")
	}
	threads := mustActiveThreads(t, e, 3)
	if threads[2].ChainID != "chain_1" {
		t.Errorf("connecting thread ChainID = %q, want chain_1 (assignment still runs)", threads[2].ChainID)
	}
}

// Event ordering (queues): thread events and chain events drain through
// independent FIFO queues — DrainEvents keeps its Story 001 behavior while
// DrainChainEvents carries the chain lifecycle events.
func TestDrainChainEvents_SeparateFIFOQueue_IndependentOfDrainEvents(t *testing.T) {
	// Arrange
	e := thread_investigation.NewEngine(fixedClock())
	e.RegisterFact(factWithRefs("fact_001", []entity.GameEntityID{42},
		knowledge.Lead{EntityID: 42, TopicHint: "Ley-touched metal"}))

	// Act
	e.OnFactDiscovered("fact_001", 7)

	// Assert — each drain sees exactly its own events, FIFO, one call.
	chainEvents := e.DrainChainEvents()
	if len(chainEvents) != 1 || chainEvents[0].Type != thread_investigation.EventChainCreatedType {
		t.Fatalf("chain events = %+v, want one chain.created", chainEvents)
	}
	threadEvents := e.DrainEvents()
	if len(threadEvents) != 1 || threadEvents[0].ThreadID != "thread_fact_001_42" {
		t.Fatalf("thread events = %+v, want one thread.created", threadEvents)
	}
	if evs := e.DrainChainEvents(); len(evs) != 0 {
		t.Errorf("second chain drain = %d events, want 0 (queue cleared)", len(evs))
	}
}

// Performance (QA criterion): entity overlap check across 100 discovered
// facts distributed across 5 chains completes within 1ms. Each iteration
// re-discovers a registered fact, exercising the idempotent path with the
// full overlap scan over 100 facts — the story's "overlap checked for a
// new fact" scenario (1000 iterations under benchmark).
//
// Go benchmarks report the mean ns/op, not p95; the actual cost is
// microsecond-scale (large headroom), so the mean-is-not-p95 gap is
// immaterial (same note as the Story 001 benchmark).
func BenchmarkOnFactDiscovered_EntityOverlapCheck(b *testing.B) {
	e := thread_investigation.NewEngine(fixedClock())

	// Pre-load 100 facts across 5 chains: each fact references one of 5
	// shared entities (chains) plus a unique lead target, from a distinct
	// NPC so the overlap scan has real work to do.
	for i := 0; i < 100; i++ {
		chainEntity := 1000 + entity.GameEntityID(i%5)
		factID := fmt.Sprintf("fact_%03d", i)
		e.RegisterFact(factWithRefs(factID,
			[]entity.GameEntityID{chainEntity},
			knowledge.Lead{EntityID: 2000 + entity.GameEntityID(i), TopicHint: "mine"}))
		e.OnFactDiscovered(factID, 3000+entity.GameEntityID(i))
	}
	if chains := e.Knowledge().Chains; len(chains) != 5 {
		b.Fatalf("preload chains = %d, want 5", len(chains))
	}

	// The measured fact: its overlap scan covers all 100 existing facts.
	e.RegisterFact(factWithRefs("new_fact", []entity.GameEntityID{5000},
		knowledge.Lead{EntityID: 5000, TopicHint: "fresh"}))
	e.OnFactDiscovered("new_fact", 9999)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.OnFactDiscovered("new_fact", 9999) // idempotent re-discovery
	}
	b.StopTimer()

	if perOp := b.Elapsed() / time.Duration(b.N); perOp > time.Millisecond {
		b.Fatalf("OnFactDiscovered per call = %v, want < 1ms", perOp)
	}
}

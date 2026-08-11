package thread_investigation

import (
	"fmt"
	"strconv"
	"strings"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// InvestigationEngine is the Thread Investigation system interface
// (ADR-0010, Story 001). The Question System calls OnFactDiscovered when
// the player learns a new fact; the Investigation Journal reads threads
// and chains for display. Thread creation is event-driven: threads,
// chains, and discovered facts are maintained in a graph structure at
// runtime (ADR-0010 Decision).
type InvestigationEngine interface {
	// OnFactDiscovered is called when the player learns a new fact.
	OnFactDiscovered(factID string, sourceNPC entity.GameEntityID)

	// GetActiveThreads returns all active and prioritized threads.
	GetActiveThreads() []Thread

	// GetChain returns a specific investigation chain.
	GetChain(chainID string) (Chain, bool)

	// ResolveThread resolves a thread via the Story 004 state machine:
	// ThreadResolved and ThreadDeadEnd move the thread to ResolvedThreads;
	// ThreadPrioritized pins it in ActiveThreads. Every resolution emits
	// EventThreadResolved.
	ResolveThread(threadID string, resolution ThreadStatus)

	// GetRevelations returns all recorded revelations, in discovery order
	// (Story 004) — one per chain that reached the revelation threshold.
	GetRevelations() []Revelation

	// OnEntityDestroyed auto-resolves every active thread targeting the
	// destroyed entity as a dead end (Story 004 AC-3). The entity framework
	// lifecycle bridge calls this on entity destruction (ADR-0001,
	// ADR-0011); the reason (e.g. "combat_death") is carried on each
	// EventThreadResolved.
	OnEntityDestroyed(entityID entity.GameEntityID, reason string)
}

// EntityNamer resolves display names for GameEntityIDs used in thread
// topic text ("Ask Orin about Ley-touched metal"). The Entity Framework
// supplies names in a later story; the default namer falls back to the
// bracketed "[entity {id}]" placeholder.
type EntityNamer interface {
	// Name returns the display name of the entity, and whether the entity
	// has a known name.
	Name(id entity.GameEntityID) (string, bool)
}

// Option configures an Engine at construction.
type Option func(*Engine)

// WithClock sets the game clock source used for CreatedAt and
// DiscoveredAt timestamps (ADR-0007). Until the game clock is wired to
// the Engine (later story), NewEngine defaults to a clock that returns 0,
// matching the Entity Framework's "clock not wired yet" pattern.
func WithClock(clock func() float64) Option {
	return func(e *Engine) {
		if clock != nil {
			e.clock = clock
		}
	}
}

// WithNamer sets the entity name resolver used in thread topic text.
func WithNamer(namer EntityNamer) Option {
	return func(e *Engine) {
		if namer != nil {
			e.namer = namer
		}
	}
}

// WithAutoChainMerge sets the auto_chain_merge tuning knob (default true,
// GDD tuning knob, ADR-0010 risk mitigation for false chain merges). When
// disabled, threads are still auto-assigned to existing chains (Step 3c)
// and new chains are still created (Step 3d), but chains never auto-merge
// on entity overlap (Step 3b). The player can split false merges via the
// Investigation Journal — that is Presentation layer, out of Story 002.
func WithAutoChainMerge(enabled bool) Option {
	return func(e *Engine) {
		e.autoChainMerge = enabled
	}
}

// WithRevelationThreshold sets the revelation_threshold tuning knob (Story
// 004, GDD tuning table): how many connected facts must share the same
// conclusion before a chain flips to revelation_ready. Default 3; the GDD
// tuning range is 2-5 — lower fires revelations faster. Values outside the
// range are clamped into it: a threshold of 1 would make every fact a
// revelation, and anything above 5 exceeds the designed envelope.
func WithRevelationThreshold(n int) Option {
	return func(e *Engine) {
		if n < 2 {
			n = 2
		}
		if n > 5 {
			n = 5
		}
		e.revelationThreshold = n
	}
}

// Engine implements InvestigationEngine (Story 001: Thread Data Model &
// Auto-Creation, TR-thread-001, ADR-0010).
//
// OnFactDiscovered implements the ADR-0010 algorithm:
//
//  1. Record fact in PlayerKnowledge.DiscoveredFacts
//  2. Record lead target entities in KnownEntities (if new)
//  3. For each lead_to entry: CreateThread + queue EventThreadCreated,
//     then auto-chain (Story 002): assign to a matching chain or create
//     one (Step 3c/3d), merge chains that now share an entity reference
//     (Step 3b), and flag entity overlaps between different NPCs (AC-1)
//  4. Contradiction check (Story 003): for each contradicts entry whose
//     other fact is already discovered, create a contradiction thread,
//     flag both facts, and queue EventContradictionDetected
//  5. Revelation check (Story 004): for each chain this discovery touched,
//     count facts sharing the same leads_to target entity or contradicting
//     the same fact; at the revelation threshold the chain flips to
//     revelation_ready, a Revelation is recorded, and EventRevelationReady
//     is queued — exactly once per chain (spam prevention)
//
// Engine is not safe for concurrent use — it is owned by the single-threaded
// game loop. All mutations happen synchronously on the main thread (ADR-0011
// synchronous event dispatch). If concurrent access is ever needed, wrap
// calls with a mutex or move mutation to a serial queue.
type Engine struct {
	knowledge PlayerKnowledge

	// facts is the fact registry populated via RegisterFact by the NPC
	// Knowledge Graph loader (ADR-0005). OnFactDiscovered resolves fact
	// data — the leads_to hints and contradicts entries — from here by
	// fact ID.
	facts map[string]knowledge.Fact

	// Runtime indexes keep OnFactDiscovered O(n) over leads_to entries
	// (performance requirement: thread creation <2ms with 50 discovered
	// facts). They are engine-owned, not part of PlayerKnowledge; Story
	// 005 rebuilds them on deserialization.
	factIndex   map[string]struct{}
	threadIndex map[string]struct{}
	knownIndex  map[entity.GameEntityID]struct{}
	// contradictedIndex tracks the facts flagged in a detected
	// contradiction (Story 003 Step 4b) for O(1) deduplicated flagging. It
	// mirrors the persisted ContradictedFacts slice; Story 005 rebuilds it
	// on deserialization like the other indexes.
	contradictedIndex map[string]struct{}

	// entityChainIndex maps an entity to the set of chain IDs whose facts
	// or threads reference it (Story 002 Step 3). It gives O(1) entity to
	// chain lookups so chain assignment and the overlap merge check never
	// scan every chain's facts. Maintained on every chain mutation
	// (assignment, creation, merge); Story 005 rebuilds it on
	// deserialization like the other indexes.
	entityChainIndex map[entity.GameEntityID]map[string]struct{}

	// chainCounter is the next chain number for "chain_{N}" IDs — a
	// monotonically incrementing counter (Story 002). Archived chains keep
	// their numbers; the counter never wraps.
	chainCounter int

	// pendingThreadEvents are queued EventThreadCreated values, drained
	// by DrainEvents for EventBus dispatch (ADR-0011, later story).
	pendingThreadEvents []EventThreadCreated

	// pendingChainEvents are queued chain events (EventThreadChained,
	// EventChainCreated, EventChainsMerged, EventContradictionDetected,
	// EventRevelationReady, EventThreadResolved) as ChainEvent envelopes,
	// drained by DrainChainEvents. Kept separate from pendingThreadEvents
	// so DrainEvents' Story 001 signature stays compatible.
	pendingChainEvents []ChainEvent

	// overlapFlags are the deduplicated entity overlap flags raised by
	// AC-1 checks, in discovery order. Read via GetOverlapFlags.
	overlapFlags []OverlapFlag

	// autoChainMerge is the auto_chain_merge tuning knob (default true):
	// whether chains auto-merge when discovered to share an entity
	// reference. Chain assignment still runs when disabled; only the merge
	// (Step 3b) is skipped.
	autoChainMerge bool

	// revelationThreshold is the revelation_threshold tuning knob (Story
	// 004, TR-thread-004): how many facts in a chain must share a conclusion
	// (same leads_to target entity, or all contradicting the same fact)
	// before the chain flips to revelation_ready. Default 3 (GDD tuning
	// table range 2-5).
	revelationThreshold int

	// clock is the game time source for CreatedAt/DiscoveredAt.
	clock func() float64
	// namer resolves entity display names for thread topics.
	namer EntityNamer
}

// defaultNamer is the fallback entity name resolver: no names are known
// until the Entity Framework supplies them, so topics render as
// "Ask [entity 42] about ...". The bracket form marks an unresolved name.
type defaultNamer struct{}

func (defaultNamer) Name(entity.GameEntityID) (string, bool) { return "", false }

// NewEngine returns an Engine with no facts registered, the default clock
// (returns 0), and the default namer. Configure with WithClock and
// WithNamer.
func NewEngine(opts ...Option) *Engine {
	e := &Engine{
		facts:               make(map[string]knowledge.Fact),
		factIndex:           make(map[string]struct{}),
		threadIndex:         make(map[string]struct{}),
		knownIndex:          make(map[entity.GameEntityID]struct{}),
		contradictedIndex:   make(map[string]struct{}),
		entityChainIndex:    make(map[entity.GameEntityID]map[string]struct{}),
		chainCounter:        1, // first chain ID is "chain_1"
		autoChainMerge:      true,
		revelationThreshold: 3,
		clock:               func() float64 { return 0 },
		namer:               defaultNamer{},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Compile-time check that Engine implements InvestigationEngine.
var _ InvestigationEngine = (*Engine)(nil)

// RegisterFact makes a fact's data (its leads_to hints) available to
// thread creation. The NPC Knowledge Graph loader calls this for every
// fact of every loaded NPC graph before any discovery can occur (KG-001,
// ADR-0005). Registering the same fact ID twice is a no-op; empty fact
// IDs are ignored.
func (e *Engine) RegisterFact(fact knowledge.Fact) {
	if fact.FactID == "" {
		return
	}
	if _, ok := e.facts[fact.FactID]; ok {
		return
	}
	e.facts[fact.FactID] = fact
}

// OnFactDiscovered records the fact, auto-creates threads from its
// leads_to hints (TR-thread-001, ADR-0010 steps 1-2), auto-chains them by
// entity overlap (TR-thread-002, ADR-0010 step 3), flags entity overlaps
// between different NPCs (Story 002 AC-1), detects contradictions against
// already-discovered facts (TR-thread-003, ADR-0010 step 4), and checks
// revelation thresholds on the touched chains (TR-thread-004, ADR-0010
// step 5).
//
// Step 3 (Story 002) runs per newly created thread: collect the entity
// references of the thread's source fact plus the thread's own target,
// scan the entity→chain index for a matching chain, assign or create, and
// queue EventThreadChained or EventChainCreated. Step 3b then merges any
// chains that now share an entity reference (auto_chain_merge knob, on by
// default) and queues EventChainsMerged. Finally the AC-1 overlap check
// flags entities referenced by facts from different NPCs.
//
// Idempotency: discovering the same fact twice records it once, does not
// create duplicate threads (thread IDs are deterministic — see
// createThread), does not re-run chain assignment, merge, or overlap
// flagging for already-processed threads (deduplicated flags), and does
// not re-fire a contradiction already detected (contradiction thread IDs
// are deterministic too). A fact not registered via RegisterFact is still
// recorded in DiscoveredFacts, but no threads can be derived without its
// data.
//
// Step 5 (Story 004) runs after Step 4: every chain whose fact set this
// discovery grew is checked for the revelation threshold (3+ facts sharing a
// leads_to target entity, or all contradicting the same fact). A chain at
// the threshold flips to revelation_ready, records a Revelation, and queues
// EventRevelationReady — exactly once per chain (spam prevention, ADR-0010).
//
// All records and events created by one call share a single clock capture.
func (e *Engine) OnFactDiscovered(factID string, sourceNPC entity.GameEntityID) {
	if factID == "" {
		return
	}
	now := e.clock()

	// Step 1: record the fact in DiscoveredFacts (once). Duplicate
	// discovery is not re-recorded so later counts (revelation threshold,
	// Story 004) stay honest.
	if _, ok := e.factIndex[factID]; !ok {
		e.knowledge.DiscoveredFacts = append(e.knowledge.DiscoveredFacts, DiscoveredFact{
			FactID:       factID,
			SourceNPC:    sourceNPC,
			DiscoveredAt: now,
		})
		e.factIndex[factID] = struct{}{}
	}

	// Resolve fact data for leads_to hints.
	fact, ok := e.facts[factID]
	if !ok {
		return
	}

	// touched tracks the chains whose fact sets this discovery grew (Step 5
	// revelation check runs only over them — O(touched_chains), ADR-0010).
	touched := make(map[string]struct{})

	// Step 2: record lead target entities in KnownEntities (if new).
	// Step 3 (Story 002): create one thread per lead, then auto-chain it.
	for _, lead := range fact.LeadsTo {
		e.recordKnownEntity(lead.EntityID, factID, now)
		if !e.createThread(lead, factID, now) {
			continue // thread already existed — chain state unchanged
		}
		// Step 3a: collect the entities this thread references (the
		// source fact's references plus the lead's target entity).
		entities := threadEntitySet(fact, lead)
		// Step 3c/3d: assign to a matching chain or create a new one.
		e.assignOrCreateChain(threadIDFor(factID, lead.EntityID), factID, entities, now, touched)
		// Step 3b: merge chains that now share an entity reference.
		e.checkAndMergeChains(entities, now)
	}

	// Step 4 (Story 003): contradiction detection — if this fact
	// contradicts facts the player already knows, create a contradiction
	// thread, flag both facts, and queue EventContradictionDetected.
	e.checkContradictions(fact, factID, sourceNPC, now, touched)

	// Step 5 (Story 004): revelation check for every chain this discovery
	// touched.
	for chainID := range touched {
		e.checkRevelation(chainID, now)
	}

	// Story 002 AC-1: flag entities referenced by facts from a different
	// NPC than this fact's source.
	e.checkEntityOverlap(fact, factID, sourceNPC)
}

// recordKnownEntity adds the entity to KnownEntities on first discovery.
func (e *Engine) recordKnownEntity(id entity.GameEntityID, viaFactID string, now float64) {
	if _, ok := e.knownIndex[id]; ok {
		return
	}
	e.knowledge.KnownEntities = append(e.knowledge.KnownEntities, KnownEntity{
		EntityID:      id,
		DiscoveredAt:  now,
		DiscoveredVia: viaFactID,
	})
	e.knownIndex[id] = struct{}{}
}

// threadIDFor returns the deterministic thread ID for a lead:
// "thread_{source_fact_id}_{entity_id}" (Story 001). Deterministic IDs make
// re-discovering a source fact idempotent.
func threadIDFor(sourceFactID string, target entity.GameEntityID) string {
	return fmt.Sprintf("thread_%s_%d", sourceFactID, target)
}

// createThread creates a thread for a lead and queues EventThreadCreated.
// Thread IDs are deterministic — "thread_{source_fact_id}_{entity_id}" —
// so re-discovering the source fact cannot create duplicates (idempotent
// thread creation, Story 001). Threads start in ThreadActive; ChainID is
// populated by Story 002 (auto-chaining).
//
// It reports whether a new thread was actually created. On a duplicate
// discovery the call is a no-op and returns false, telling the caller to
// skip Step 3 (the thread is already chained from its first discovery).
func (e *Engine) createThread(lead knowledge.Lead, sourceFactID string, now float64) bool {
	id := threadIDFor(sourceFactID, lead.EntityID)
	thread := Thread{
		ID:           id,
		Source:       sourceFactID,
		TargetEntity: lead.EntityID,
		Topic:        e.formatTopic(lead.EntityID, lead.TopicHint),
		Status:       ThreadActive,
		CreatedAt:    now,
	}
	if !e.addActiveThread(thread) {
		return false // already created — idempotent
	}
	e.pendingThreadEvents = append(e.pendingThreadEvents, EventThreadCreated{
		ThreadID:     id,
		SourceFactID: sourceFactID,
		TargetEntity: lead.EntityID,
		Topic:        thread.Topic,
		CreatedAt:    now,
	})
	return true
}

// addActiveThread appends the thread to ActiveThreads unless its ID is
// already reserved in the threadIndex, reporting whether it was added.
// Deterministic IDs make creation idempotent for both lead threads (Story
// 001) and contradiction threads (Story 003): re-discovering the source
// fact never duplicates either kind.
func (e *Engine) addActiveThread(thread Thread) bool {
	if _, ok := e.threadIndex[thread.ID]; ok {
		return false
	}
	e.knowledge.ActiveThreads = append(e.knowledge.ActiveThreads, thread)
	e.threadIndex[thread.ID] = struct{}{}
	return true
}

// --- Story 003: contradiction detection (ADR-0010 Step 4, TR-thread-003) ---

// checkContradictions implements ADR-0010 Step 4 (Story 003, TR-thread-003):
// for each fact_id in fact.Contradicts that the player has already
// discovered, create a contradiction thread, flag both facts as
// contradictory, and queue EventContradictionDetected. Contradictions are
// clues, not bugs — they create investigation tension.
//
// Detection is forward-only per the ADR algorithm: only the newly
// discovered fact's contradicts entries are scanned. Latent contradictions
// (the other fact undiscovered) create nothing now and fire when the other
// fact is discovered (AC-2). Self-contradictions are rejected. A
// contradicted fact with no registered data is skipped — like leads, no
// thread is derivable without the fact's data. Circular pairs (A
// contradicts B, B contradicts A) fire each direction when that direction's
// contradicting fact is (re)discovered against an already-known other fact.
func (e *Engine) checkContradictions(fact knowledge.Fact, factID string, sourceNPC entity.GameEntityID, now float64, touched map[string]struct{}) {
	for _, contradictedID := range fact.Contradicts {
		if contradictedID == factID {
			continue // self-contradiction — rejected (QA edge case)
		}
		if _, ok := e.factIndex[contradictedID]; !ok {
			continue // latent — the other fact is not discovered yet (AC-2)
		}
		contradicted, ok := e.facts[contradictedID]
		if !ok {
			continue // no data for the contradicted fact — no thread derivable
		}
		contradictedNPC := e.sourceNPCFor(contradictedID)
		confidence := contradictionConfidence(fact, contradicted)
		threadID, created := e.createContradictionThread(factID, contradictedID, sourceNPC, contradictedNPC, fact, contradicted, confidence, now, touched)
		if !created {
			continue // already detected — idempotent re-discovery
		}
		e.flagContradicted(factID)
		e.flagContradicted(contradictedID)
		e.queueChainEvent(EventContradictionDetectedType, EventContradictionDetected{
			FactA:       fact,
			FactB:       contradicted,
			SourceNPC_A: sourceNPC,
			SourceNPC_B: contradictedNPC,
			Confidence:  confidence,
			ThreadID:    threadID,
		}, now)
	}
}

// createContradictionThread creates the thread for a detected contradiction
// and chain-assigns it like any other thread (Story 002 Step 3c/3d): the
// thread's entity set is the union of both facts' references, matched
// against existing chains, then chains sharing those entities are merged
// (Step 3b). The thread ID is deterministic —
// "thread_contradiction_{new_fact}_{contradicted_fact}" — so re-discovery
// never duplicates it. TargetEntity is unset: the contradiction is
// open-ended, the player decides who to believe (ADR-0010). No
// EventThreadCreated is queued — the contradiction's event is
// EventContradictionDetected (Story 003 Step 4d). It reports whether the
// thread was actually created.
func (e *Engine) createContradictionThread(factID, contradictedID string, sourceNPC, contradictedNPC entity.GameEntityID, fact, contradicted knowledge.Fact, confidence string, now float64, touched map[string]struct{}) (string, bool) {
	id := fmt.Sprintf("thread_contradiction_%s_%s", factID, contradictedID)
	thread := Thread{
		ID:           id,
		Source:       factID + "↔" + contradictedID,
		TargetEntity: 0, // open-ended — the player decides who to believe
		Topic:        e.contradictionTopic(sourceNPC, contradictedNPC, fact, contradicted, confidence),
		Status:       ThreadActive,
		CreatedAt:    now,
	}
	if !e.addActiveThread(thread) {
		return id, false
	}
	entities := factReferencesUnion(fact, contradicted)
	e.assignOrCreateChain(id, factID, entities, now, touched)
	e.checkAndMergeChains(entities, now)
	return id, true
}

// contradictionTopic renders the player-facing contradiction thread text
// (Story 003, GDD format): high-confidence contradictions read "Resolve
// contradiction: [NPC A] claims [fact A]. [NPC B] claims [fact B].";
// low confidence reads "Possible contradiction: [NPC A] claims [fact A],
// but [NPC B] says [fact B] — one account may be a rumor." Fact summaries
// are the fact's Content, falling back to the fact ID when empty.
func (e *Engine) contradictionTopic(sourceNPC, contradictedNPC entity.GameEntityID, fact, contradicted knowledge.Fact, confidence string) string {
	aName := e.displayName(sourceNPC)
	bName := e.displayName(contradictedNPC)
	aSummary := factSummary(fact)
	bSummary := factSummary(contradicted)
	if confidence == "low" {
		return fmt.Sprintf("Possible contradiction: %s claims %s, but %s says %s — one account may be a rumor.", aName, aSummary, bName, bSummary)
	}
	return fmt.Sprintf("Resolve contradiction: %s claims %s. %s claims %s.", aName, aSummary, bName, bSummary)
}

// factSummary is the human-readable summary of a fact used in contradiction
// topics: the fact's natural-language Content, or the fact ID when Content
// is empty (same fallback pattern as chain summaries, Story 002).
func factSummary(fact knowledge.Fact) string {
	if fact.Content != "" {
		return fact.Content
	}
	return fact.FactID
}

// contradictionConfidence returns the confidence of a detected
// contradiction (Story 003 CR-3): "low" when either fact is a RUMOR
// (ADR-0005 fact type — the rumor may be false), else "high".
func contradictionConfidence(factA, factB knowledge.Fact) string {
	if factA.Type == knowledge.FactType_Rumor || factB.Type == knowledge.FactType_Rumor {
		return "low"
	}
	return "high"
}

// factReferencesUnion returns the deduplicated union of both facts'
// References — the entity set a contradiction thread is chain-assigned by
// (both facts' subjects connect the thread to their chains).
func factReferencesUnion(factA, factB knowledge.Fact) []entity.GameEntityID {
	seen := make(map[entity.GameEntityID]struct{}, len(factA.References)+len(factB.References))
	out := make([]entity.GameEntityID, 0, len(factA.References)+len(factB.References))
	for _, ref := range factA.References {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	for _, ref := range factB.References {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

// flagContradicted records a fact as flagged in a detected contradiction
// (Story 003 AC-1, ADR-0010 Step 4b), deduplicated per fact in detection
// order.
func (e *Engine) flagContradicted(factID string) {
	if _, ok := e.contradictedIndex[factID]; ok {
		return
	}
	e.contradictedIndex[factID] = struct{}{}
	e.knowledge.ContradictedFacts = append(e.knowledge.ContradictedFacts, factID)
}

// sourceNPCFor returns the GameEntityID of the NPC who revealed a
// discovered fact, or 0 when the fact is not discovered.
func (e *Engine) sourceNPCFor(factID string) entity.GameEntityID {
	for _, d := range e.knowledge.DiscoveredFacts {
		if d.FactID == factID {
			return d.SourceNPC
		}
	}
	return 0
}

// formatTopic renders the player-facing lead text: "Ask {name} about
// {hint}". The name comes from the EntityNamer; unresolvable names fall
// back to "[entity {id}]" (QA test format). An empty hint drops the
// "about" suffix.
func (e *Engine) formatTopic(target entity.GameEntityID, hint string) string {
	name := e.displayName(target)
	if hint == "" {
		return "Ask " + name
	}
	return "Ask " + name + " about " + hint
}

// displayName resolves an entity's display name via the EntityNamer,
// falling back to the bracketed "[entity {id}]" placeholder (same format
// as thread topics, QA test format).
func (e *Engine) displayName(id entity.GameEntityID) string {
	if name, ok := e.namer.Name(id); ok && name != "" {
		return name
	}
	return "[entity " + strconv.FormatUint(uint64(id), 10) + "]"
}

// --- Story 002: auto-chaining (ADR-0010 Step 3, TR-thread-002) ---

// threadEntitySet returns the entity references used for chain matching of
// a newly created thread (Step 3a): the source fact's References plus the
// lead's target entity, deduplicated. Both matter: a fact "references" its
// subjects (AC-1/AC-4), and a thread "references" its target entity
// (AC-2).
func threadEntitySet(fact knowledge.Fact, lead knowledge.Lead) []entity.GameEntityID {
	out := make([]entity.GameEntityID, 0, len(fact.References)+1)
	seen := make(map[entity.GameEntityID]struct{}, len(fact.References)+1)
	for _, ref := range fact.References {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	if _, ok := seen[lead.EntityID]; !ok {
		out = append(out, lead.EntityID)
	}
	return out
}

// autoGenerateSummary derives a chain's summary from its first fact (Step
// 3d): the fact's first lead's topic hint, or the fact ID when there is no
// hint (Story 002: "use the fact's first topic hint or fact ID"). The
// knowledge.Fact type has no TopicHint field — hints live on leads.
func autoGenerateSummary(fact knowledge.Fact) string {
	if len(fact.LeadsTo) > 0 && fact.LeadsTo[0].TopicHint != "" {
		return fact.LeadsTo[0].TopicHint
	}
	return fact.FactID
}

// intersectEntities returns the entity IDs present in both slices, in the
// order of a, deduplicated.
func intersectEntities(a, b []entity.GameEntityID) []entity.GameEntityID {
	inB := make(map[entity.GameEntityID]struct{}, len(b))
	for _, id := range b {
		inB[id] = struct{}{}
	}
	var out []entity.GameEntityID
	seen := make(map[entity.GameEntityID]struct{})
	for _, id := range a {
		if _, ok := inB[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// stringInSlice reports whether v is present in s.
func stringInSlice(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// indexAdd records that the given chain references the given entities in
// the entity→chain index (O(1) entity→chain lookup).
func (e *Engine) indexAdd(chainID string, entities []entity.GameEntityID) {
	for _, ent := range entities {
		set := e.entityChainIndex[ent]
		if set == nil {
			set = make(map[string]struct{})
			e.entityChainIndex[ent] = set
		}
		set[chainID] = struct{}{}
	}
}

// indexRemoveChainEntities removes the chain from the entity→chain index
// for the given entities (called when a chain is archived on merge, so no
// future thread is routed to an archived chain).
func (e *Engine) indexRemoveChainEntities(chainID string, entities []entity.GameEntityID) {
	for _, ent := range entities {
		if set := e.entityChainIndex[ent]; set != nil {
			delete(set, chainID)
			if len(set) == 0 {
				delete(e.entityChainIndex, ent)
			}
		}
	}
}

// collectChainEntities returns the full entity reference set of a chain —
// the References of its facts plus the TargetEntity of its threads,
// deduplicated. This is the inverse of the entity→chain index, used to
// clean and union index entries during merges.
func (e *Engine) collectChainEntities(chainID string) []entity.GameEntityID {
	var chain *Chain
	for i := range e.knowledge.Chains {
		if e.knowledge.Chains[i].ID == chainID {
			chain = &e.knowledge.Chains[i]
			break
		}
	}
	if chain == nil {
		return nil
	}
	seen := make(map[entity.GameEntityID]struct{})
	var out []entity.GameEntityID
	add := func(id entity.GameEntityID) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, fid := range chain.Facts {
		if f, ok := e.facts[fid]; ok {
			for _, ref := range f.References {
				add(ref)
			}
		}
	}
	for _, tid := range chain.Threads {
		for i := range e.knowledge.ActiveThreads {
			if e.knowledge.ActiveThreads[i].ID == tid {
				if id := e.knowledge.ActiveThreads[i].TargetEntity; id != 0 {
					add(id)
				}
			}
		}
		for i := range e.knowledge.ResolvedThreads {
			if e.knowledge.ResolvedThreads[i].ID == tid {
				if id := e.knowledge.ResolvedThreads[i].TargetEntity; id != 0 {
					add(id)
				}
			}
		}
	}
	return out
}

// setThreadChainID updates the ChainID of a thread wherever it lives
// (ActiveThreads or ResolvedThreads).
func (e *Engine) setThreadChainID(threadID, chainID string) {
	for i := range e.knowledge.ActiveThreads {
		if e.knowledge.ActiveThreads[i].ID == threadID {
			e.knowledge.ActiveThreads[i].ChainID = chainID
		}
	}
	for i := range e.knowledge.ResolvedThreads {
		if e.knowledge.ResolvedThreads[i].ID == threadID {
			e.knowledge.ResolvedThreads[i].ChainID = chainID
		}
	}
}

// queueChainEvent appends a chain event to the pending queue, captured at
// the given game time.
func (e *Engine) queueChainEvent(typ entity.EventType, data any, now float64) {
	e.pendingChainEvents = append(e.pendingChainEvents, ChainEvent{
		Type:      typ,
		Timestamp: now,
		Data:      data,
	})
}

// assignOrCreateChain implements ADR-0010 Step 3c/3d for one newly created
// thread: scan the entity→chain index for chains referencing any of the
// thread's entities; if found, assign the thread to the oldest matching
// chain (Step 3c, EventThreadChained); otherwise create a new chain (Step
// 3d, EventChainCreated). Chains whose fact sets this discovery grew are
// recorded in touched for the Step 5 revelation check (Story 004) — a chain
// already holding the fact is not touched.
func (e *Engine) assignOrCreateChain(threadID, factID string, entities []entity.GameEntityID, now float64, touched map[string]struct{}) {
	// Step 3b: collect candidate chains via the O(1) index.
	matched := make(map[string]struct{})
	for _, ent := range entities {
		for chainID := range e.entityChainIndex[ent] {
			matched[chainID] = struct{}{}
		}
	}
	// Oldest first: knowledge.Chains is append-only in creation order.
	for i := range e.knowledge.Chains {
		chain := &e.knowledge.Chains[i]
		if chain.Status == ChainStatusArchived {
			continue // archived chains are never in the index; defensive
		}
		if _, ok := matched[chain.ID]; !ok {
			continue
		}
		if e.addThreadToChain(chain.ID, threadID, factID, entities) {
			touched[chain.ID] = struct{}{}
		}
		e.queueChainEvent(EventThreadChainedType, EventThreadChained{
			ThreadID: threadID,
			ChainID:  chain.ID,
		}, now)
		return
	}
	// Step 3d: no matching chain — create a new one.
	e.createChain(threadID, factID, entities, now, touched)
}

// addThreadToChain records the thread and its source fact in the chain
// (facts deduplicated by fact_id), points the thread at the chain, and
// extends the entity→chain index (Step 3c). It reports whether the fact was
// newly added to the chain — false when the chain already contained it
// (re-discovery), which tells Step 5 the chain was not touched by this
// discovery.
func (e *Engine) addThreadToChain(chainID, threadID, factID string, entities []entity.GameEntityID) bool {
	for i := range e.knowledge.Chains {
		chain := &e.knowledge.Chains[i]
		if chain.ID != chainID {
			continue
		}
		newlyAdded := !stringInSlice(chain.Facts, factID)
		if newlyAdded {
			chain.Facts = append(chain.Facts, factID)
		}
		if !stringInSlice(chain.Threads, threadID) {
			chain.Threads = append(chain.Threads, threadID)
		}
		e.setThreadChainID(threadID, chainID)
		e.indexAdd(chainID, entities)
		return newlyAdded
	}
	return false
}

// createChain creates a new chain containing the thread as its first
// thread (Step 3d, AC-3). Chain IDs are "chain_{N}" with a monotonically
// incrementing counter; the summary comes from autoGenerateSummary.
func (e *Engine) createChain(threadID, factID string, entities []entity.GameEntityID, now float64, touched map[string]struct{}) {
	id := fmt.Sprintf("chain_%d", e.chainCounter)
	e.chainCounter++
	summary := autoGenerateSummary(e.facts[factID])
	e.knowledge.Chains = append(e.knowledge.Chains, Chain{
		ID:      id,
		Summary: summary,
		Facts:   []string{factID},
		Threads: []string{threadID},
		Status:  "active",
	})
	e.setThreadChainID(threadID, id)
	e.indexAdd(id, entities)
	touched[id] = struct{}{} // new chain — the fact is always newly added
	e.queueChainEvent(EventChainCreatedType, EventChainCreated{
		ChainID:       id,
		Summary:       summary,
		FirstThreadID: threadID,
	}, now)
}

// checkAndMergeChains implements ADR-0010 Step 3b: after a thread's chain
// assignment, any chains that share at least one entity reference with the
// thread's entity set are merged into the oldest of them. Because the
// merge runs on every discovery, no two chains share an entity when this
// runs, so the touched set is exactly the merge group — no transitive
// expansion is needed. No-op when the auto_chain_merge tuning knob is
// disabled (WithAutoChainMerge(false)).
func (e *Engine) checkAndMergeChains(entities []entity.GameEntityID, now float64) {
	if !e.autoChainMerge {
		return
	}
	// Collect every chain sharing at least one entity with the new set.
	touched := make(map[string]struct{})
	for _, ent := range entities {
		for chainID := range e.entityChainIndex[ent] {
			touched[chainID] = struct{}{}
		}
	}
	if len(touched) < 2 {
		return
	}
	// Survivor = oldest chain (first in creation order).
	var survivor string
	for i := range e.knowledge.Chains {
		if _, ok := touched[e.knowledge.Chains[i].ID]; ok {
			survivor = e.knowledge.Chains[i].ID
			break
		}
	}
	if survivor == "" {
		return // defensive — touched set built from the index
	}
	// Archived = the rest, in creation order.
	archived := make([]string, 0, len(touched)-1)
	for i := range e.knowledge.Chains {
		if id := e.knowledge.Chains[i].ID; id != survivor {
			if _, ok := touched[id]; ok {
				archived = append(archived, id)
			}
		}
	}
	e.mergeChains(survivor, archived, now)
}

// mergeChains absorbs the archived chains into the survivor (AC-4): union
// facts and threads (deduplicated), re-point every merged thread's ChainID
// at the survivor, archive the old chains (kept for investigation history,
// ADR-0010), update the entity→chain index, recalculate the summary to
// reflect the merged scope, and queue EventChainsMerged.
func (e *Engine) mergeChains(survivorID string, archivedIDs []string, now float64) {
	if len(archivedIDs) == 0 {
		return
	}
	var survivor *Chain
	for i := range e.knowledge.Chains {
		if e.knowledge.Chains[i].ID == survivorID {
			survivor = &e.knowledge.Chains[i]
			break
		}
	}
	if survivor == nil {
		return // defensive — survivor must exist
	}
	summaries := []string{survivor.Summary}
	for _, archivedID := range archivedIDs {
		e.absorbChain(survivor, archivedID, &summaries)
	}
	survivor.Summary = strings.Join(summaries, " & ")
	e.queueChainEvent(EventChainsMergedType, EventChainsMerged{
		SurvivingChainID: survivorID,
		ArchivedChainIDs: archivedIDs,
		MergedSummary:    survivor.Summary,
	}, now)
}

// absorbChain moves all facts and threads from an archived chain into the
// survivor, updates thread ChainIDs, collects summaries, and transfers the
// entity-chain index entries. The archived chain stays in knowledge.Chains
// with Status = ChainStatusArchived for investigation history.
func (e *Engine) absorbChain(survivor *Chain, archivedID string, summaries *[]string) {
	for i := range e.knowledge.Chains {
		archived := &e.knowledge.Chains[i]
		if archived.ID != archivedID {
			continue
		}
		entities := e.collectChainEntities(archivedID)
		for _, fid := range archived.Facts {
			if !stringInSlice(survivor.Facts, fid) {
				survivor.Facts = append(survivor.Facts, fid)
			}
		}
		for _, tid := range archived.Threads {
			if !stringInSlice(survivor.Threads, tid) {
				survivor.Threads = append(survivor.Threads, tid)
			}
			e.setThreadChainID(tid, survivor.ID)
		}
		if archived.Summary != "" && !stringInSlice(*summaries, archived.Summary) {
			*summaries = append(*summaries, archived.Summary)
		}
		archived.Status = ChainStatusArchived
		e.indexRemoveChainEntities(archivedID, entities)
		// Repair: also scan entityChainIndex for any remaining entries
		// pointing at the archived chain (contradiction threads with
		// TargetEntity 0 can leave stale entries that the recomputed set
		// misses; Story 003). Clean them all up so no archived chain is
		// ever reachable through the index.
		for ent, chainSet := range e.entityChainIndex {
			delete(chainSet, archivedID)
			if len(chainSet) == 0 {
				delete(e.entityChainIndex, ent)
			}
		}
		e.indexAdd(survivor.ID, entities)
	}
}

// --- Story 002: entity overlap flagging (AC-1) ---

// checkEntityOverlap implements AC-1: when a newly discovered fact
// references an entity that a previously discovered fact from a DIFFERENT
// NPC also references, an OverlapFlag is raised — "Both [NPC A] and [NPC B]
// mentioned [entity]." Facts from the same NPC never flag (an NPC knowing
// two facts about the same entity is not a cross-reference). Flags are
// deduplicated per (entity, NPC pair), so re-discovery and repeated facts
// surface each connection once.
func (e *Engine) checkEntityOverlap(fact knowledge.Fact, factID string, sourceNPC entity.GameEntityID) {
	if len(fact.References) == 0 {
		return
	}
	for _, existing := range e.knowledge.DiscoveredFacts {
		if existing.FactID == factID || existing.SourceNPC == sourceNPC {
			continue
		}
		existingFact, ok := e.facts[existing.FactID]
		if !ok || len(existingFact.References) == 0 {
			continue
		}
		overlap := intersectEntities(fact.References, existingFact.References)
		if len(overlap) == 0 {
			continue
		}
		e.recordOverlapFlag(OverlapFlag{
			Entity: overlap[0],
			NPC_A:  existing.SourceNPC,
			NPC_B:  sourceNPC,
			Message: fmt.Sprintf("Both %s and %s mentioned %s.",
				e.displayName(existing.SourceNPC),
				e.displayName(sourceNPC),
				e.displayName(overlap[0])),
		})
	}
}

// recordOverlapFlag appends a flag unless one for the same connection was
// already recorded. The NPC pair is unordered — "Both A and B mentioned X"
// is symmetric, so (Entity, A, B) and (Entity, B, A) are the same flag.
func (e *Engine) recordOverlapFlag(flag OverlapFlag) {
	for _, f := range e.overlapFlags {
		if f.Entity == flag.Entity &&
			((f.NPC_A == flag.NPC_A && f.NPC_B == flag.NPC_B) ||
				(f.NPC_A == flag.NPC_B && f.NPC_B == flag.NPC_A)) {
			return
		}
	}
	e.overlapFlags = append(e.overlapFlags, flag)
}

// --- Story 004: revelation threshold (ADR-0010 Step 5, TR-thread-004) ---

// checkRevelation implements ADR-0010 Step 5 (Story 004, TR-thread-004) for
// one chain touched by a fact discovery: if the chain's facts contain a
// group of >= revelationThreshold facts sharing the same conclusion — the
// same leads_to target entity, or all contradicting the same fact_id — the
// chain flips to revelation_ready, the revelation is recorded in
// PlayerKnowledge.Revelations, and EventRevelationReady is queued via the
// DrainChainEvents pipeline.
//
// The signal fires exactly once per chain: a chain already in
// revelation_ready is skipped (ADR-0010 spam mitigation) — additional facts
// deepen the revelation but never re-emit. Archived chains never trigger
// (defensive — merged chains are unreachable through the entity index).
func (e *Engine) checkRevelation(chainID string, now float64) {
	for i := range e.knowledge.Chains {
		chain := &e.knowledge.Chains[i]
		if chain.ID != chainID {
			continue
		}
		if chain.Status == ChainStatusRevelationReady || chain.Status == ChainStatusArchived {
			return
		}
		target, count := e.revelationGroup(chain)
		if count < e.revelationThreshold {
			return
		}
		chain.Status = ChainStatusRevelationReady
		e.knowledge.Revelations = append(e.knowledge.Revelations, Revelation{
			ID:           "revelation_" + chain.ID,
			ChainID:      chain.ID,
			DiscoveredAt: now,
		})
		e.queueChainEvent(EventRevelationReadyType, EventRevelationReady{
			ChainID:      chain.ID,
			FactCount:    count,
			TargetEntity: target,
			Summary:      chain.Summary,
		}, now)
		return
	}
}

// revelationGroup returns the chain's strongest revelation signal: the
// leads_to target entity shared by the most chain facts and that group's
// size, or the largest group of chain facts contradicting the same fact_id
// (reported with entity 0 — the conclusion is a fact, not an entity). Facts
// count once per group target (a fact with two leads to the same entity is
// one fact), self-contradictions are excluded, and entity 0 leads are
// ignored. Ties prefer the leads_to group, which carries a concrete target
// entity for EventRevelationReady; both group types iterate chain facts in
// order, so ties break deterministically to the first-encountered target.
	// buildGroupCounts tallies leads_to target entities and contradicted
	// fact_ids across the chain's facts (deduplicated per fact, self-
	// contradictions excluded, entity 0 ignored). Used by revelationGroup.
	func (e *Engine) buildGroupCounts(chain *Chain) (leadGroups map[entity.GameEntityID]int, contradictGroups map[string]int) {
		leadGroups = make(map[entity.GameEntityID]int, len(chain.Facts))
		contradictGroups = make(map[string]int, len(chain.Facts))
		for _, fid := range chain.Facts {
			fact, ok := e.facts[fid]
			if !ok {
				continue
			}
			seenLead := make(map[entity.GameEntityID]struct{}, len(fact.LeadsTo))
			for _, lead := range fact.LeadsTo {
				if lead.EntityID == 0 {
					continue
				}
				if _, dup := seenLead[lead.EntityID]; dup {
					continue
				}
				seenLead[lead.EntityID] = struct{}{}
				leadGroups[lead.EntityID]++
			}
			seenContra := make(map[string]struct{}, len(fact.Contradicts))
			for _, contradicted := range fact.Contradicts {
				if contradicted == fid {
					continue
				}
				if _, dup := seenContra[contradicted]; dup {
					continue
				}
				seenContra[contradicted] = struct{}{}
				contradictGroups[contradicted]++
			}
		}
		return leadGroups, contradictGroups
	}

	// bestLeadGroup returns the most-referenced leads_to target entity and its
	// count across the chain. Ties are broken by first-encountered in chain
	// fact order (strict >).
	func (e *Engine) bestLeadGroup(chain *Chain, leadGroups map[entity.GameEntityID]int) (entity.GameEntityID, int) {
		var bestTarget entity.GameEntityID
		bestCount := 0
		for _, fid := range chain.Facts {
			fact, ok := e.facts[fid]
			if !ok {
				continue
			}
			for _, lead := range fact.LeadsTo {
				if count := leadGroups[lead.EntityID]; count > bestCount {
					bestCount = count
					bestTarget = lead.EntityID
				}
			}
		}
		return bestTarget, bestCount
	}

	// bestContradictGroup returns the highest count of facts contradicting the
	// same fact_id across the chain.
	func (e *Engine) bestContradictGroup(chain *Chain, contradictGroups map[string]int) int {
		best := 0
		for _, fid := range chain.Facts {
			fact, ok := e.facts[fid]
			if !ok {
				continue
			}
			for _, contradicted := range fact.Contradicts {
				if count := contradictGroups[contradicted]; count > best {
					best = count
				}
			}
		}
		return best
	}

	// revelationGroup returns the dominant entity target and fact count for
	// the chain's revelation check. It compares the best leads_to group
	// against the best contradict group; contradiction must be strictly larger
	// to win (ties prefer the concrete-entity signal from leads_to).
	func (e *Engine) revelationGroup(chain *Chain) (entity.GameEntityID, int) {
		leadGroups, contradictGroups := e.buildGroupCounts(chain)
		bestTarget, bestCount := e.bestLeadGroup(chain, leadGroups)
		if bestContra := e.bestContradictGroup(chain, contradictGroups); bestContra > bestCount {
			return 0, bestContra
		}
		return bestTarget, bestCount
	}

	// GetActiveThreads returns a copy of the open threads (active and
	// prioritized) in creation order.
	func (e *Engine) GetActiveThreads() []Thread {
	return append([]Thread(nil), e.knowledge.ActiveThreads...)
}

// GetChain returns the chain with the given ID. Chains are created by
// Story 002 (auto-chaining); until then every lookup reports not found.
func (e *Engine) GetChain(chainID string) (Chain, bool) {
	for i := range e.knowledge.Chains {
		if e.knowledge.Chains[i].ID == chainID {
			return e.knowledge.Chains[i], true
		}
	}
	return Chain{}, false
}

// ResolveThread implements the manual path of the Story 004 resolution state
// machine (ADR-0010 CR-4): ThreadResolved ("discovered") and ThreadDeadEnd
// move the thread to ResolvedThreads; ThreadPrioritized pins it in
// ActiveThreads. Every resolution emits EventThreadResolved via the
// DrainChainEvents pipeline.
//
// The thread ID stays reserved in every case, so re-discovering the source
// fact cannot resurrect a resolved thread. A dead-end thread is effectively
// reactivated when new evidence emerges: a new fact with a lead to the same
// entity creates a fresh thread (deterministic IDs, CR-4). Invalid
// resolutions and unknown thread IDs are no-ops (QA edge cases). Manual
// resolutions carry no reason ("") on the event; the NPC-death
// auto-resolution path supplies one via OnEntityDestroyed.
func (e *Engine) ResolveThread(threadID string, resolution ThreadStatus) {
	e.resolveThread(threadID, resolution, "")
}

// resolveThread is the shared resolution path behind ResolveThread and
// OnEntityDestroyed. reason is carried on EventThreadResolved ("" for manual
// resolutions, the entity destruction reason for NPC-death auto-resolution).
// Each resolution captures the clock once for the event timestamp.
func (e *Engine) resolveThread(threadID string, resolution ThreadStatus, reason string) {
	now := e.clock()
	emit := func() {
		e.queueChainEvent(EventThreadResolvedType, EventThreadResolved{
			ThreadID:   threadID,
			Resolution: resolution,
			Reason:     reason,
		}, now)
	}
	switch resolution {
	case ThreadResolved, ThreadDeadEnd:
		for i := range e.knowledge.ActiveThreads {
			if e.knowledge.ActiveThreads[i].ID != threadID {
				continue
			}
			thread := e.knowledge.ActiveThreads[i]
			thread.Status = resolution
			e.knowledge.ActiveThreads = append(e.knowledge.ActiveThreads[:i], e.knowledge.ActiveThreads[i+1:]...)
			e.knowledge.ResolvedThreads = append(e.knowledge.ResolvedThreads, thread)
			emit()
			return
		}
	case ThreadPrioritized:
		for i := range e.knowledge.ActiveThreads {
			if e.knowledge.ActiveThreads[i].ID == threadID {
				e.knowledge.ActiveThreads[i].Status = ThreadPrioritized
				emit()
				return
			}
		}
	}
	// Unknown thread ID or invalid resolution: no-op (QA edge case).
}

// OnEntityDestroyed auto-resolves every active thread targeting the
// destroyed entity as a dead end (Story 004 AC-3, ADR-0010 CR-4): the
// lead's subject is gone, so the thread can no longer be followed. The
// entity framework lifecycle bridge calls this on destruction events
// (ADR-0001, ADR-0011 — EventBus wiring is a later story); the reason
// (e.g. "combat_death") is recorded on each EventThreadResolved payload.
//
// Threads targeting other entities are untouched. Contradiction threads
// (TargetEntity 0) never match a real death — entity IDs start at 1 — and
// the entityID 0 guard keeps a bogus call from resolving them all. Multiple
// threads targeting the same NPC all resolve; already-resolved threads are
// not in ActiveThreads, so they are unaffected.
func (e *Engine) OnEntityDestroyed(entityID entity.GameEntityID, reason string) {
	if entityID == 0 {
		return // entity 0 is never a real GameEntityID — defensive
	}
	var targets []string
	for _, thread := range e.knowledge.ActiveThreads {
		if thread.TargetEntity == entityID {
			targets = append(targets, thread.ID)
		}
	}
	for _, id := range targets {
		e.resolveThread(id, ThreadDeadEnd, reason)
	}
}

// GetRevelations returns the recorded revelations in discovery order (Story
// 004, TR-thread-004): one per chain that reached the revelation threshold.
// The returned slice is a copy — treat it as read-only; revelations are
// recorded by OnFactDiscovered via checkRevelation.
func (e *Engine) GetRevelations() []Revelation {
	return append([]Revelation(nil), e.knowledge.Revelations...)
}

// DrainEvents returns and clears the queued EventThreadCreated events.
// The Question System integration story drains these and emits them on
// the EventBus (ADR-0011) as Event{Type: EventThreadCreatedType,
// Source: "thread_investigation"}.
func (e *Engine) DrainEvents() []EventThreadCreated {
	events := e.pendingThreadEvents
	e.pendingThreadEvents = nil
	return events
}

// DrainChainEvents returns and clears the queued chain events —
// EventThreadChained, EventChainCreated, EventChainsMerged,
// EventContradictionDetected, EventRevelationReady, and
// EventThreadResolved — as ChainEvent envelopes, in FIFO order (Story 002).
// The Question System
// integration story drains these and emits them on the EventBus (ADR-0011)
// as Event{Type: ev.Type, Source: "thread_investigation", Data: ev.Data}.
// Thread creation events are drained separately via DrainEvents, keeping
// the Story 001 signature intact.
func (e *Engine) DrainChainEvents() []ChainEvent {
	events := e.pendingChainEvents
	e.pendingChainEvents = nil
	return events
}

// GetOverlapFlags returns the entity overlap flags raised by Story 002
// AC-1 checks, in discovery order, deduplicated per (entity, NPC pair).
// The returned slice is a copy — treat it as read-only; flags are
// generated by OnFactDiscovered and read by the journal UI.
func (e *Engine) GetOverlapFlags() []OverlapFlag {
	return append([]OverlapFlag(nil), e.overlapFlags...)
}

// Knowledge returns a shallow copy of the player's investigation state
// for reading (journal display, tests). The returned slices alias the
// engine's state — treat them as read-only; all mutations must go through
// the Engine so its runtime indexes stay consistent.
func (e *Engine) Knowledge() PlayerKnowledge {
	return e.knowledge
}

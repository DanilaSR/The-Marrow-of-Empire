# ADR-0008: LLM Inference Integration

## Status
Accepted

## Date
2026-08-09
**Revised**: 2026-08-11 — Dual-model architecture (CGo tiny model + IPC primary model)
**Revised**: 2026-08-12 — Prompt caching, context window 4096, per-NPC state machine, post-processing pipeline, input sanitization, capability probe, conversation memory (GDD MAJOR REVISION sync)

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Scripting / AI |
| **Knowledge Risk** | HIGH — custom engine + external LLM inference |
| **References Consulted** | AI Dialogue Engine GDD, NPC Knowledge Graph GDD, GGUF specification |
| **Post-Cutoff APIs Used** | llama.cpp GGUF inference (external to Kaiju) |
| **Verification Required** | GGUF inference performance on target hardware, IPC mechanism latency, memory footprint with game running, CGo tiny-model build integration with Kaiju's existing CGO requirement, tiny model character fidelity for simple responses |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture), ADR-0005 (NPC Knowledge Graph) |
| **Enables** | AI Dialogue Engine implementation, Question System, Dialogue UI |
| **Blocks** | AI Dialogue Engine — cannot implement without this decision |
| **Ordering Note** | Must be Accepted before AI Dialogue Engine development begins |

## Context

### Problem Statement
The Marrow of Empire uses local LLMs for NPC dialogue. The primary model (Fiendish_LLAMA_3B, Llama 3.2 3B RP fine-tune, Q4_K_M, ~2.1 GB) generates knowledge-rich responses but requires ~2 seconds first-token latency on desktop CPU. A significant fraction of conversational turns are simple — greetings, deflections, empty-fact-set queries, "I don't know" responses — where 2-second latency breaks the illusion of natural conversation. How should the LLM inference backend handle both fast simple queries and complex knowledge-rich generation?

### Constraints
- Primary model: Fiendish_LLAMA_3B.Q4_K_M.gguf (~2.1 GB) — authoritative per GDD 2026-08-09 review
- Tiny model: ~200-500M parameters, Q4_K_M, ~200-400 MB on disk, ~400-600 MB runtime
- Primary runtime memory budget: ~5GB total (model ~2.1 GB + KV cache ~448 MiB/slot × 2-4 slots + compute buffers)
- Tiny model runtime memory budget: ~600MB within the game process heap
- Target response time: <100ms for simple queries (tiny model), <2s for complex queries (primary, cached follow-up)
- Effective context window: **4096 tokens default** (2048–8192 configurable; updated from 2048 per GDD 2026-08-09)
- Must run on player hardware — no cloud dependency; minimum 16 GB RAM, quad-core 3.0 GHz+ AVX2, SSD required
- Game must remain responsive during primary-model inference (non-blocking IPC)
- Tiny model must not crash the game on failure (acceptable risk for <500M model)
- Must degrade gracefully if either model fails to load or times out
- Prompt caching required: static core cached per (NPC, trust_level, exemplar_variant); multi-slot LRU KV cache
- Input must be sanitized before reaching the model (chat-format tokens stripped, instruction injection detected)
- Hardware capability probe at startup gates LLM mode vs text-fallback for the session

### Requirements
- Separate process for primary model (crash isolation for the 2.1 GB model)
- In-process CGo for tiny model (latency-critical, acceptable crash risk at small scale)
- Prompt construction from NPC Knowledge Graph data (both models share the same prompt pipeline)
- Response post-processing (entity name validation, truncation, repetition detection)
- Query routing: simple queries → tiny model; complex queries → primary model
- Fallback to plain-text fact display if LLM unavailable (both tiers)

## Decision

**Dual-model architecture: tiny CGo model for fast simple queries + separate IPC process for the primary 3B model.** This revision (2026-08-12) expands the ADR with prompt caching architecture, per-NPC state machine, post-processing pipeline, input sanitization, capability probe, and conversation memory — all derived from the GDD's MAJOR REVISION (2026-08-09).

### Architecture Diagram

```
┌──────────────────────────────────────────────────────┐
│  GAME PROCESS (Kaiju Engine / Go)                    │
│                                                      │
│  AI Dialogue Engine                                  │
│  ┌──────────────────────────────────────────────┐   │
│  │ Prompt Constructor (shared)                  │   │
│  │ NPC facts + personality + question + history │   │
│  └──────────────────┬───────────────────────────┘   │
│                     │ prompt                         │
│  ┌──────────────────▼───────────────────────────┐   │
│  │ TieredBackend (LLMBackend)                   │   │
│  │                                              │   │
│  │  isSimpleQuery(prompt)?                      │   │
│  │   ├── YES → TinyCGoBackend.Generate()        │   │
│  │   │         in-process, ~50-100ms             │   │
│  │   │         model: ~200-500M, Q4_K_M          │   │
│  │   │                                          │   │
│  │   └── NO  → IPCLlamaBackend.Generate()       │   │
│  │             subprocess IPC, ~2s (cached)      │   │
│  │             model: Fiendish 3B, Q4_K_M        │   │
│  └──────────────────┬───────────────────────────┘   │
│                     │ IPC (complex queries only)     │
└─────────────────────┼──────────────────────────────┘
                      │
┌─────────────────────▼──────────────────────────────┐
│  LLM PROCESS (llama.cpp) — primary model only      │
│                                                     │
│  Fiendish_LLAMA_3B.Q4_K_M (~2.1 GB)                │
│  Memory: ~3.5-5GB (model + KV cache + buffers)      │
│  Context window: 4096 tokens (default)               │
│  KV Cache: multi-slot LRU (2-4 slots, ~448 MiB/slot)│
│  Lifecycle: spawn/monitor/restart                    │
│  Prompt caching: static core cached per              │
│    (npc_id, trust_level, exemplar_variant)            │
│  Cancellation: slot abort (preserves KV cache)       │
└─────────────────────────────────────────────────────┘
```

### Query Routing

Simple queries (routed to tiny CGo model):
- Empty fact set (player asked a topic outside all NPC domains) → deflection
- Greeting or conversational opener (no domain keywords matched)
- Follow-up with pronoun resolution only (topic carried from context, no new entities)
- Short questions (<5 tokens, no entity references)

Complex queries (routed to IPC primary model):
- Domain-matched substantive questions
- Questions with entity references
- Questions requiring specific fact knowledge to answer
- Any query where the tiny model's response would be indistinguishable from text-fallback

### Key Interfaces (revised)

The `LLMBackend` interface is unchanged — the TieredBackend is a composition, not a new interface:

```go
// LLMBackend remains the single swappable contract (ADR-0008 original).
type LLMBackend interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Status() ModelStatus
    Shutdown() error
}

// TieredBackend composes two LLMBackend implementations.
// It is itself an LLMBackend — transparent to the AI Dialogue Engine.
type TieredBackend struct {
    fast  LLMBackend  // TinyCGoBackend (in-process, <500M params)
    full  LLMBackend  // IPCLlamaBackend (separate process, 3B params)
}

func (tb *TieredBackend) Generate(ctx context.Context, prompt string) (string, error) {
    if isSimpleQuery(prompt) {
        return tb.fast.Generate(ctx, prompt)
    }
    return tb.full.Generate(ctx, prompt)
}

// isSimpleQuery classifies the prompt for routing (Story 005 — Tiered Routing).
// Pure function: prompt text + MatchResult → backend selection.
// Simple queries: empty facts, no entity refs, greetings, pronoun-only follow-ups.
```

### Tiny Model Constraints

| Parameter | Value |
|-----------|-------|
| Max parameters | 500M |
| Quantization | Q4_K_M minimum |
| Max disk size | ~400 MB |
| Max runtime memory | ~600 MB (within game heap) |
| CGo safety | Model failure must not crash the game. Wrap CGo calls in `recover()`; on panic, degrade to text-fallback for this query only. The tiny model is a fast-path; its failure is not session-stable. |
| Build | Kaiju Engine already requires CGO — no new build complexity |

### Prompt Caching Architecture (revised 2026-08-12)

The prompt is split into a **static core** (~985-1185 tokens, cacheable) and a **dynamic suffix** (~280-375 tokens, per-query). The static core — Character, Exemplars, all trust-visible facts (deterministic order), Referral Roster, Relationship context, and Rules — is invariant for a given `(npc_id, trust_level, exemplar_variant)`. The dynamic suffix — Conversation So Far (rolling summary), Disclosed Facts, and the player's question — is appended after the static core.

**Cache key**: `(npc_id, trust_level, exemplar_variant)`. Invalidated on trust change or exemplar swap.

**Multi-slot LRU**: 2-4 slots preserve static cores for recently-talked NPCs. An NPC switch uses a different slot; returning to a cached NPC avoids a full re-prefill. Trust-level change or exemplar-variant swap (trust ≥2 warm exemplar) invalidates the cache for that NPC — the first query at the new trust tier pays a full static-core prefill.

**KV cache memory**: ~448 MiB per slot at 4096 context (Llama 3.2 3B, GQA 24Q/8KV). 2 slots ≈ 0.9 GiB.

**Cancellation**: In-flight generation is cancelled via slot abort (llama-server), not process kill — this preserves the KV cache for the current NPC and does not consume the restart budget. If the backend does not support slot abort, cancellation may require process restart (consumes restart budget — the Open Question on IPC architecture must resolve this).

### Per-NPC State Machine (revised 2026-08-12)

Each NPC conversation has its own state. The global model subprocess is a shared resource.

| State | Meaning | Behavior |
|-------|---------|----------|
| `Unloaded` | Model subprocess not started | Engine not yet initialized (global) |
| `Loading` | Model loading + capability probe | Progress indicator; text-fallback if player starts early |
| `Ready` | Model loaded, waiting for queries | Idle — no inference running |
| `Generating` | LLM inference in progress | Streaming tokens to UI; new questions queued (depth 1, most recent wins) |
| `TextMode` | This NPC using text-fallback | Full designed text-mode experience |

**Key transitions:**

| Transition | Trigger | Scope |
|------------|---------|-------|
| `Unloaded → Loading` | Game launch | Global |
| `Loading → Ready` | Model loaded + capability probe passes | Global |
| `Loading → TextMode` | Model load failed OR probe fails | Global — session-stable |
| `Ready → Generating` | Player submits question | Per-NPC |
| `Generating → Ready` | Response complete or cancelled | Per-NPC |
| `Generating → TextMode` | Timeout (>25s) — text fallback for this exchange; 3 timeouts in one conversation → session-stable TextMode for this NPC | Per-NPC |
| `TextMode → Ready` | Post-timeout retry succeeds OR warm-up completes mid-game | Per-NPC |
| `Ready → TextMode` | Subprocess crash while idle → restart fails | Global |

**TextMode scope rules:**
- **Global** (probe fail, crash-restart fail): all NPCs, session-stable
- **Per-NPC timeout**: 3 timeouts within one conversation → session-stable TextMode for that NPC only
- **Transitional** (warm-up in progress): resolves to Ready when warm-up completes

**Concurrency**: Single shared subprocess serializes all generation — one in-flight request globally, queue depth 1, most-recent-question-wins. Generation cancellable on: Esc, combat start, walk-away, NPC death, scene unload, NPC switch.

### Response Post-Processing Pipeline (revised 2026-08-12)

The LLM generates tokens which are streamed to the game. Post-processing runs incrementally as tokens arrive and is finalized on generation complete. The pipeline is:

1. **Streaming with sentence buffering**: Tokens accumulated into whole sentences (delimited by `.`, `!`, `?`). Entity correction runs on each completed sentence. The typewriter reveal begins displaying a sentence only after it clears post-processing — adding ~1 sentence of display latency hidden by the typewriter effect. Word-chunks buffered for entity correction (<10ms per chunk).

2. **Stop-string detection**: Generation stops at `<|eot_id|>`, `<|end_of_text|>`, or last complete sentence ≤ `max_response_tokens` (default 150, hard cap 200).

3. **Fuzzy entity resolution**: Generated text scanned against entity registry. Exact match → accept canonical form. Levenshtein ≤2 → correct (case-sensitive, proper nouns only; multi-word names use n-gram matching). No match → attempt pronoun/role substitution; if fails, strip sentence. `knows_not_entities` names: positive knowledge claims stripped; negations/deflections preserved.

4. **Repetition detection**: Token-level Jaccard similarity vs immediate predecessor (same tokenizer as prompt builder). >85% → re-query once with `repetition_penalty = 1.1` (never temperature). 80-85% deadband hysteresis. Responses <10 tokens skip check. Always-on baseline `repetition_penalty = 1.05`.

5. **Empty/garbage response**: Fall back to highest-relevance matching fact displayed as NPC quote text. Log fallback.

6. **Hallucination stripping**: Pronouns/roles substituted where possible; otherwise sentence stripped. Player-introduced entity names (detected during sanitization) are whitelisted — exempt from stripping (NPC must be able to echo/deflect player-introduced names).

**Post-processing budget**: Entity resolution <10ms, repetition check <10ms, total <30ms. Prompt builder <5ms p95 (pure function, no I/O).

### Input Sanitization (revised 2026-08-12)

Applied before prompt insertion:
- **Chat-format tokens stripped**: `<|begin_of_text|>`, `<|eot_id|>`, `<|start_header_id|>`, `<|end_header_id|>`, and attribute variants → replaced with plain-text equivalents or removed
- **Instruction injection detected**: Regex for "ignore your (previous )?instructions", "you are (actually|now)", "forget (everything|your)", "act as", "pretend you are", "your (new |real )?(rules|system prompt)" → route to authored in-character deflection ("I'm not sure what you mean by that. What are you actually asking?")
- **Player-introduced entity names whitelisted**: Names not in entity registry that the player typed → exempt from hallucination stripping in post-processing
- **Question length cap**: 500 characters (Unicode-safe), enforced by Question System before routing

This is a quality-of-illusion guard, not a security boundary. A determined player can still jailbreak a 3B model; the design accepts this for a single-player game.

### Capability Probe (revised 2026-08-12)

At startup, the engine runs a ~150-token benchmark prompt measuring **both prefill and decode speed**. The gate is prefill-inclusive:

```
Probe passes if ALL of:
  (a) decode_speed ≥ 10 tok/s
  (b) (1200/prefill_speed + 150/decode_speed) ≤ 25s × 0.85  [cold-start, 15% margin]
  (c) (280/prefill_speed + 150/decode_speed) ≤ 12s            [cached follow-up]
```

Failure → TextMode for session with one-time notice. The gate has 1.7× margin over timeout-derived minimum (6 tok/s). On laptops, sustained inference may thermal-throttle below probe reading; if repeated timeouts occur mid-session, the player receives a one-time plain-language notification offering manual switch to text mode (no automatic flapping).

### Conversation Memory (revised 2026-08-12)

**Rolling summary**: After each NPC response, the engine generates a brief summary (2-3 bullets, ≤150 tokens) via a separate low-priority idle inference. Older bullets dropped, newest retained. Cancellable if new question arrives before summary completes.

**Disclosed facts**: Set of fact IDs already disclosed in this conversation tracked per-NPC. Injected into prompt as brief labels to prevent repetition.

**Cross-session persistence**: Rolling summary and disclosed-fact set persisted via Persistent World State into save file. On reload, restored with gap acknowledgment ("It's been a while since we last spoke...").

**Exhausted-topic interception**: When all facts at current trust level have been disclosed, further questions on those topics are intercepted before LLM call — NPC delivers authored exhaustion line. The LLM is not called.

### Graceful Degradation (revised 2026-08-12)

| Failure | Behavior |
|---------|----------|
| Primary model fails to load | Game starts normally. Tiny model handles all queries. Text fallback when tiny model can't answer. "AI Dialogue — limited mode" indicator. |
| Tiny model fails to load | All queries route to primary model. No indicator — invisible to player. Latency increases to ~2s for simple queries. |
| Primary model timeout (>25s) | Current response falls back to text. Next query retries normally. 3 timeouts → restart. |
| Primary model crash | Auto-restart once. Current response: try tiny model first, then text. |
| Tiny model CGo panic | `recover()` catches it. Fall back to primary model for this query. Tiny model reloaded on next simple query. |
| Both models unavailable | Text-fallback mode — session-stable (GDD CR-5). |

### Prompt Template

Both models share the same prompt construction pipeline. The prompt template is unchanged from the original decision. The tiny model receives the same structured prompt — its smaller capacity means fewer facts and shorter history fit in its effective context window (~512 tokens), but the structure is identical.

## Alternatives Considered

### Alternative 1: Separate process with IPC (CHOSEN)
- **Pros**: Process isolation — model crash doesn't affect game. Model restart independent. Can use pre-built llama.cpp binary. Memory separation — game and model have separate heaps. Swappable — replace process with cloud API later by swapping LLMBackend implementation.
- **Cons**: IPC overhead — JSON serialize/deserialize per request (~1-5ms). Process management complexity — spawn, monitor, restart. Two processes to ship and update.
- **Rejection Reason**: N/A — chosen.

### Alternative 2: In-process via CGo bindings — ACCEPTED WITH CONSTRAINTS (revised 2026-08-11)
- **Description**: Link llama.cpp as a C library via CGo. Call inference functions directly from Go. Accepted for tiny models only (<500M parameters, <400MB disk).
- **Pros**: No IPC overhead. Single process — simpler deployment. Direct memory access — no serialize/deserialize. ~50-100ms latency for simple responses. Kaiju Engine already requires CGO — no new build complexity.
- **Cons**: CGo overhead per call. Model crash could take down the game. Memory shared with game. CGo complicates cross-compilation.
- **Acceptance Constraints**: Max 500M parameters. Wrap CGo calls in recover(). Model crash during Generate → panic caught, degrade to other backend for this query. Tiny model failure is not session-stable — it's a fast-path only. The cross-compilation concern is mitigated by Kaiju's existing CGO requirement.
- **Rejection Reason for large models (unchanged)**: A 3B+ model crash should never take down the player's game. Process isolation is worth the IPC overhead for models at that scale. The size threshold (~500M) keeps crash risk proportional — a 300MB model has orders of magnitude less code surface than a 2.1GB model with GQA attention and KV cache management.

### Alternative 3: Tiered/Dual Backend (CHOSEN — added 2026-08-11)
- **Description**: Compose two LLMBackend implementations behind a TieredBackend that routes simple queries to the tiny CGo model and complex queries to the IPC primary model. Both backends implement the same LLMBackend interface — the AI Dialogue Engine sees one backend.
- **Pros**: Instant responses for ~60-70% of conversational turns. Primary model can stay unloaded during exploration. Text-fallback mode richer (tiny model produces in-character small talk even when primary is down). Clean composition — no interface changes.
- **Cons**: Two model files to ship. Two inference paths to test. Tiny model still needs prompt construction and response processing (shares pipeline, doesn't replace it). Query classification adds routing logic.
- **Rejection Reason**: N/A — chosen.
- **Description**: Run llama.cpp as a local HTTP server (e.g., llama.cpp server mode). Game sends HTTP POST requests.
- **Pros**: Standard protocol — easy to debug, easy to swap. Can use any HTTP client library. Well-understood failure modes.
- **Cons**: HTTP overhead per request (~10-50ms). Port management — find free port, handle conflicts. Server management — start/stop lifecycle. Extra dependency (HTTP server in llama.cpp or wrapper).
- **Rejection Reason**: HTTP adds latency and complexity without benefit over stdin/stdout IPC for a local-only service.

## Consequences

### Positive
- Model isolation for primary model: crash, memory leak, or hang in the 3B LLM never affects the game
- Swappable backend: `LLMBackend` interface enables future cloud/hybrid/upgraded-model without dialogue logic changes
- Independent scaling: primary model can use different CPU cores than the game
- Text fallback ensures the game is always playable, even without the LLM
- Fast-path latency: simple queries (greetings, deflections, empty-fact-set) respond in ~50-100ms via in-process tiny model — natural conversation feel for the most common turns
- Graceful tier fallback: tiny model failure → primary model; primary failure → tiny model → text. Three-tier degradation instead of two.
- Tiny model keeps text-fallback feeling alive: even when the primary model is down, the tiny model produces in-character small talk

### Negative
- IPC overhead for complex queries: ~1-5ms per request for JSON serialization (primary model only)
- Process lifecycle for primary model: spawn, health-check, restart logic adds complexity
- Three binaries to distribute: game + llama.cpp (primary) + tiny model GGUF
- Two inference paths to test, debug, and tune
- CGo entanglement returns for the tiny model (mitigated by <500M size constraint and recover() guard)
- Query classification adds routing logic and a decision boundary that must be correct

### Risks
- **Primary model startup time**: ~15 seconds. Mitigation: pre-load during main menu; text fallback until ready; tiny model handles all queries during warm-up.
- **Memory pressure**: 3-4GB for primary model + 400-600MB for tiny model + 2-4GB game = 6-8GB total. Mitigation: primary model unloaded when not in conversation; tiny model always resident (small enough to keep).
- **IPC pipe blocking (primary)**: If llama.cpp hangs, stdin pipe blocks. Mitigation: context-based timeout on all Generate calls; force-restart process on timeout.
- **Tiny model CGo crash**: Mitigation: `recover()` wrapper catches panics; fall back to primary model for that query. Tiny model reloads on next simple query. One panic per session is acceptable; repeated panics disable the fast path for the session.
- **Query misclassification**: A complex query routed to the tiny model produces a shallow response; a simple query routed to the primary model adds unnecessary latency. Mitigation: conservative routing — when uncertain, route to primary. The cost of false-positive "simple" is a thin answer; the cost of false-positive "complex" is 2s latency. Err toward primary.
- **Tiny model character fidelity**: A ~300M model may not hold voice as well as the 3B RP fine-tune. Mitigation: evaluate against AC-23 (in-character rubric) before committing to a specific tiny model; if <80% pass rate, the fast path is disabled and all queries route to primary.
- **Cache invalidation storms**: Trust-level changes and exemplar swaps invalidate the KV cache for that NPC — the first query at the new trust tier pays a full static-core prefill (~8s desktop, ~16s laptop). Mitigation: multi-slot LRU preserves caches for other NPCs; only the NPC whose trust changed pays the cost. The trust-tier change is a narrative beat — the latency spike is acceptable as a pacing effect.
- **KV cache memory pressure**: Each slot is ~448 MiB at 4096 context. 4 slots ≈ 1.8 GiB within the model subprocess. Mitigation: default to 2 slots (0.9 GiB); increase only if profiling shows frequent NPC-switch cold starts. Configurable via `kv_cache_slots` tuning knob.
- **Fact selection determinism**: If fact selection is not a deterministic pure function of `(npc_id, trust_level, matched_domain, detected_entity_refs)`, the static core varies per query and the KV cache is invalidated every time. Mitigation: fact selection uses fixed priority ordering (entity-ref-both → entity-ref-only → primary-domain → secondary-domain, then `priority` field, then fact ID). No runtime randomness. Go map iteration avoided in the selection path.
- **Slot abort availability**: If the chosen llama.cpp backend (llama-server vs custom wrapper) does not support slot abort, cancellation requires process restart — consuming the restart budget. Mitigation: the IPC architecture open question (llama-server vs custom wrapper) must be resolved before implementation; slot abort is a requirement for the chosen backend.
- **Thermal throttling on laptops**: Sustained inference may throttle below probe reading mid-session. Mitigation: one-time plain-language notification offers manual switch to text mode (no automatic flapping). Probe has 1.7× margin over minimum; a machine that passes at cold start should sustain full responses.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| AI Dialogue Engine | CR-1: LLM Model and Inference — Fiendish_LLAMA_3B (3.2B, Q4_K_M, ~2.1GB) + tiny model (<500M) | Dual-model: separate IPC process for primary, in-process CGo for tiny fast-path |
| AI Dialogue Engine | CR-2: Prompt Template — two-tier structure with static core + dynamic suffix, Llama 3.2 Instruct chat format | Prompt caching: static core (Character through Rules, ~1000 tok) cached per (NPC, trust, exemplar); dynamic suffix (summary + disclosed + question, ~300 tok) per-query. Both models share the same prompt pipeline. |
| AI Dialogue Engine | CR-3: Prompt Assembly — 10-step assembly pipeline from KG, Trust, Question System, PWS | Prompt builder as pure function (no I/O) with deterministic fact selection priority: entity-ref-both → entity-ref-only → primary-domain → secondary-domain. Budget-constrained (600 tok), not count-constrained. Empty-fact → deflection with referral roster. |
| AI Dialogue Engine | CR-4: Response Processing — streaming, entity resolution, repetition detection, hallucination stripping | Sentence-buffered post-processing pipeline: fuzzy entity resolution (Levenshtein ≤2), token Jaccard repetition detection (>85% → re-query), hallucination stripping, knows_not enforcement, stop-string truncation. Total <30ms. |
| AI Dialogue Engine | CR-5: Graceful Degradation — text fallback, auto-restart, timeout handling | Three-tier degradation: tiny model → primary model → text. LLMBackend interface with Status() and fallback states |
| AI Dialogue Engine | CR-6: Model Warm-Up — load during main menu, text fallback until ready | StatusLoading → StatusReady transition; tiny model handles all queries during primary warm-up |

## Performance Implications
- **CPU**: Primary model inference on separate CPU cores — game loop unaffected. Tiny model runs in-process on game thread (blocking, but <100ms for simple queries — within frame budget slack). Inference latency: ~50-100ms (tiny, simple queries), ~2s (primary, cached follow-up via prompt caching), ~8s (primary, first exchange / trust-tier change — full static core prefill). Prompt builder <5ms p95. Post-processing <30ms total. Capability probe ~150-token benchmark at startup.
- **Memory**: Primary model process: ~3.5-5GB (model ~2.1 GB + KV cache ~448 MiB/slot × 2-4 slots + compute buffers). Tiny model: ~400-600MB within game heap. Game process: ~2-4GB. Total system: ~6-10GB. Minimum recommended: 16GB RAM.
- **Load Time**: Primary model GGUF load <8s (NVMe) / <20s (SATA). Capability probe + warm-up prefill: ~4s. Target total warm-up: <12s NVMe / <20s SATA (during main menu, non-blocking). Tiny model: ~2-5s (loads first — handles queries during primary warm-up). Prompt builder <5ms per query.
- **Network**: N/A — fully local.

## Migration Plan
AI-DIALOGUE-001 (LLM Backend & IPC) is implemented with the single-model IPCLlamaBackend. The dual-model revision requires:
1. Story 005 (Tiered Routing & Tiny Model CGo Backend) — new story in the AI Dialogue Engine epic
2. Tiny model GGUF acquisition and evaluation against AC-23 in-character rubric
3. Revise control manifest forbidden pattern: "Never CGo" → "Never CGo for models >500M"
4. Existing IPCLlamaBackend and LLMBackend interface unchanged — TieredBackend is additive

## Validation Criteria
- Primary model loads within 15 seconds on reference hardware
- Tiny model loads within 5 seconds on reference hardware
- Simple query (empty facts, deflections) returns within 100ms via tiny model for 90% of requests
- Complex query (domain-matched, entity refs) returns within 2s via primary model (cached) for 90% of requests
- Primary model crash → auto-restart within 5 seconds → next complex query succeeds
- Tiny model CGo panic → caught by recover(), query falls back to primary, tiny model reloads
- Text fallback displays facts correctly when both models unavailable
- Game runs at 60fps during primary model inference (non-blocking IPC)
- Game maintains 60fps during tiny model inference (<100ms blocking, within frame budget)

## Related Decisions
- ADR-0005: NPC Knowledge Graph — provides the facts for prompt construction
- AI Dialogue Engine GDD — full prompt template and response pipeline specification

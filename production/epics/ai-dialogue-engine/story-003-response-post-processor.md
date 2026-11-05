# Story 003: Response Post-Processor

> **Epic**: AI Dialogue Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Logic
> **Estimate**: [TBD — fill before sprint planning]
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Context

**GDD**: `design/gdd/ai-dialogue-engine.md`
**Requirement**: `TR-ai-004` — Response target: <2 seconds for 50-200 tokens, non-blocking (separate goroutine), 2048-token context window. Post-processing budget: <30ms total.

**ADR Governing Implementation**: ADR-0008: LLM Inference Integration
**ADR Decision Summary**: Response post-processing happens in the game process after LLM response is received via IPC. The post-processor handles entity name validation, truncation, repetition detection, and hallucination stripping — all within a <30ms budget to maintain responsive feel.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Entity registry lookup uses hash-set indexing for <10ms resolution. Token Jaccard similarity uses the pinned tokenizer fixture. No Kaiju-specific rendering APIs required — the post-processor manipulates text, not UI.

**Control Manifest Rules (Feature Layer)**:
- Required: Graceful degradation — text fallback on model failure, auto-restart, timeout→retry
- Guardrail: LLM response <2s target for 50-200 tokens

---

## Acceptance Criteria

*From GDD `design/gdd/ai-dialogue-engine.md`, scoped to this story:*

### Response Processing (unit tests — deterministic, stub generator)

- [x] **AC-10**: GIVEN a stub generator returns "The ore comes from the Shadowforge" where `Shadowforge` is not in the entity registry and Levenshtein distance >2 to any registry entry, WHEN the response is post-processed, THEN the sentence containing "Shadowforge" is removed AND a hallucination is logged with the unknown term.

- [x] **AC-11**: GIVEN a stub returns "Gretta showed me the forge" where `Gretta` has Levenshtein distance 1 to registered entity `Greta`, WHEN post-processed, THEN "Gretta" is corrected to "Greta" AND a near-miss log entry is written.

- [x] **AC-12**: GIVEN a stub returns an empty string, WHEN post-processed, THEN the highest-relevance matching fact is displayed as quote-styled fallback text AND a fallback event is logged.

- [x] **AC-13**: GIVEN two consecutive stub responses with >85% token Jaccard similarity, WHEN the second response completes post-processing, THEN the engine re-queries exactly once with `repetition_penalty = 1.1` AND does not re-query a third time regardless of result AND the second response (whether original or retry) is accepted. Token Jaccard uses the pinned tokenizer fixture.

- [x] **AC-14**: GIVEN a stub returns a 300-token response, WHEN post-processed, THEN the response is truncated at the last complete sentence boundary ≤ 150 tokens (default `max_response_tokens`).

### Display Buffer (unit test)

- [x] **AC-30**: GIVEN a stub generator returns "The ore comes from Shadowforge" where "Shadowforge" is not in the registry and not whitelisted, WHEN the response is buffered for post-processing, THEN the sentence is stripped from the buffer BEFORE display AND the player never sees "Shadowforge" on screen AND the hallucination is logged.

---

## Implementation Notes

*Derived from GDD CR-4 (Response Processing) and ADR-0008:*

### Post-Processing Pipeline

Applied incrementally as tokens arrive, then finalized on generation complete:

1. **Stop-string detection**: Generation stops at `<|eot_id|>`, `<|end_of_text|>`, or end of complete sentence after `max_response_tokens` (default 150). Hard cap: 200 tokens.

2. **Entity name resolution** (fuzzy, Levenshtein ≤2):
   - Exact match in registry → accept, use canonical form
   - Fuzzy match (Levenshtein ≤2) → correct to canonical form. Log near-miss hallucination.
   - No match → flag as hallucination. **Exception**: player-whitelisted names (from input sanitization, Story 002) accepted verbatim.
   - For non-whitelisted unknown terms: attempt pronoun/role substitution. If substitution fails, strip the sentence.
   - `knows_not_entities` names: strip positive knowledge claims; preserve negations/deflections/ignorance statements.
   - **False positive guard**: case-sensitive (proper nouns only, starts with uppercase). Multi-word names use n-gram matching. Ambiguous matches log and leave unchanged.

3. **Response empty/garbage**: Fall back to highest-relevance matching fact displayed as NPC quote text. Log the fallback.

4. **Repetition detection** (token-level Jaccard vs immediate predecessor):
   - >85% similarity: re-query once with `repetition_penalty = 1.1`. Cap: 1 retry.
   - 80–85%: deadband — no action (hysteresis prevents boundary oscillation)
   - ≤80%: normal — accept
   - Minimum response length: 10 tokens (shorter responses skip detection)

5. **Stripping**: Remove residual stop tokens, chat-format tokens, markdown.

### Sentence-Buffered Display Pipeline (CR-4)

Tokens accumulate into whole sentences (delimited by `.`, `!`, `?` followed by whitespace + capital letter, or end-of-generation). Entity correction runs on each completed sentence. Sentence stripping runs BEFORE the sentence's first word is displayed. The typewriter reveal begins displaying a sentence only after it clears post-processing.

### Performance Budget

- Entity resolution: <10ms (hash-set indexed against ~200-name registry)
- Repetition check: <10ms
- Total post-processing: <30ms

### Important Design Boundaries

- **LLM prose is voice, never evidence** (Creative Director's ruling): The Investigation Journal records only KG-verified facts. The LLM's text is ephemeral performance.
- **False positive guard at Levenshtein ≤2**: Common English words routinely fuzzy-match entity names ("great"→"Greta", "there"→"Theras"). Case-sensitive, proper-noun-only matching required.
- **Player-whitelisted names**: Names the player introduced (from Story 002 input sanitization) are exempted from hallucination stripping — the NPC must be able to echo/deflect player-introduced names.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Input sanitization (chat-format token stripping, instruction detection) — the post-processor receives already-sanitized input context
- Story 002: Prompt construction — the post-processor processes the LLM's output, not the input
- Story 004: State machine orchestration (when to call the post-processor, timeout handling, text-fallback mode)
- Story 004: Conversation memory updates (rolling summary generation — this happens after post-processing, in the state machine)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-10**: Unknown entity hallucination → sentence stripped + logged
- **AC-11**: Fuzzy entity match → corrected to canonical form
- **AC-12**: Empty/garbage response → fact fallback
- **AC-13**: Repetition detection → re-query with repetition_penalty
- **AC-14**: Response truncation at sentence boundary
- **AC-30**: Display buffer — hallucination stripped before player sees it

---

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/ai_dialogue/response_processor_test.go` — must exist and pass

**Status**: [x] Created — 25 tests, all PASS (2026-08-11). Coverage: AC-10 through AC-14, AC-30, plus boundaries (Levenshtein ±2/±3, Jaccard 0.85 exact, 10-token minimum), knows_not gating, player whitelist, ambiguous fuzzy match, deadband hysteresis, nil registry, under-limit pass-through, default MaxResponseTokens, empty fallback placeholder.

---

## Dependencies

- Depends on: Story 002 (input sanitization produces player-whitelisted names; prompt structure defines fact tags used in fallback)
- Unlocks: Story 004 (state machine orchestrates generate → post-process → display → memory update flow)

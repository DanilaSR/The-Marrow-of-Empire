# Story 012: Companion Knowledge Contribution

> **Epic**: Party System
> **Status**: Blocked
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: L (8–12 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: —

> ⛔ **BLOCKED**: `TR-party-006` has no governing ADR — registry GAP: "needs Party ADR + ADR-0008 revision". Run `/architecture-decision` to create the Party System ADR and revise ADR-0008 (interjection interface) before implementation. Once both are Accepted, flip this story to Ready.

## Context

**GDD**: `design/gdd/party-system.md`
**Requirement**:
- `TR-party-006` — Companion knowledge contribution: active companions' Knowledge domains available to Question System; exclusive fragments inaccessible while absent; interjection interface (Hard ADE contract): domain-match ≥0.7, caps 1/companion 3/conversation, thread-resolution witness reactions guaranteed

**ADR Governing Implementation**: N/A — GAP (registry: "needs Party ADR + ADR-0008 revision")
**ADR Decision Summary**: This is Pillar 1's primary mechanical carrier (witness mechanic). The GDD declares a **Hard** cross-document contract: the AI Dialogue Engine must implement a `companion_interject` trigger with domain-match scoring, frequency caps, and async LLM fallback. ADR-0008 (AI Dialogue Engine) needs a revision to include this interface; the Party ADR must fix the domain-availability gating on the Question System side.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Async LLM fallback for un-authored interjections rides ADR-0008's separate-process llama.cpp IPC — same degradation rules (fallback text on failure, auto-restart). No new Kaiju APIs beyond ADR-0008's verified IPC path.

**Control Manifest Rules (Feature Layer)**:
- Required: Synchronous event dispatch via EventBus with system priority ordering (ADR-0011)
- Required: LLM inference via separate llama.cpp process — stdin/stdout JSON IPC, crash isolation (ADR-0008)
- Forbidden: Never use in-process CGo for LLM models exceeding 500M parameters (ADR-0008)
- Forbidden: Never use direct cross-system calls for state mutations (ADR-0011)
- Guardrail: LLM response: <2s target for 50–200 tokens (ADR-0008)

---

## Acceptance Criteria

*From GDD `design/gdd/party-system.md` CR-6 (Companion Knowledge Contribution), scoped to this story:*

- [ ] **AC-1** (AC-PS-05a, knowledge available): GIVEN an active companion with knowledge domain `old_empire_lore` and an NPC with a question gated behind that domain, WHEN the companion is in the active party and the player opens dialogue, THEN the question option is present.

- [ ] **AC-2** (AC-PS-05b, knowledge removed): GIVEN the same companion NOT in the active party, WHEN the player opens dialogue with the same NPC, THEN the domain-gated question option is absent (unless another present companion or the NPC themselves covers the domain).

- [ ] **AC-3** (exclusivity): GIVEN a knowledge fragment tagged `exclusive: true` held by exactly one companion, WHEN that companion leaves the party, THEN the fragment becomes inaccessible until re-recruitment; non-exclusive domain knowledge may still be covered by other companions or NPCs.

- [ ] **AC-4** (interjection trigger): GIVEN a dialogue topic whose domain match confidence for an active companion exceeds `interjection_confidence_threshold` (0.7), THEN the companion may interject — evaluated at response-choice points in the conversation flow, never mid-sentence, and spoken after the NPC finishes.

- [ ] **AC-5** (frequency caps): GIVEN any conversation, THEN at most `interjections_per_companion_cap` (1) interjection per companion and `total_interjections_cap` (3) total across all companions.

- [ ] **AC-6** (interjection priority): GIVEN multiple qualifying companions, THEN the companion with the highest domain match confidence speaks; ties broken by highest trust level, then by highest WIS.

- [ ] **AC-7** (interjection content): GIVEN an interjection, THEN the line comes from the companion's authored interjection lines (keyed by domain); if none exists, the AI Dialogue Engine generates one via async LLM with the companion's portrait shown alongside the NPC portrait and a brief pause indicator.

- [ ] **AC-8** (witness reactions, guaranteed): GIVEN an investigation thread resolution, THEN each present companion whose knowledge domains overlap the thread's topic contributes one guaranteed reaction (dialogue line + optional journal entry) — NOT subject to the frequency caps.

- [ ] **Edge case — already-discovered facts**: GIVEN a companion with unique knowledge leaves the party, THEN facts already discovered from that companion remain in `PlayerKnowledge` (only access to the domain is lost, not discovered content).

---

## Implementation Notes

*Derived from GDD CR-6, Interjection Rules, Knowledge exclusivity — to be confirmed against the governing ADRs once they exist:*

### Domain Availability (Question System side)

- The Question System's fact-routing (TR-q-004) must gate domain access on active-party presence: Party exposes `ActiveDomains()` (union of active companions' Knowledge domains) as a read-only query.
- `exclusive: true` fragments: query gating keyed on that companion's presence; non-exclusive domains remain available via other companions/NPCs (AC-2, AC-3).
- Already-discovered facts live in `PlayerKnowledge` (ADR-0010) and are never removed by departure.

### Interjection Contract (ADE side — Hard)

Per the GDD's cross-document contract, ADE implements `companion_interject` with:
- domain match scoring vs the current topic, threshold 0.7 (`interjection_confidence_threshold`)
- frequency budget: 1/companion, 3/conversation
- priority: confidence → trust → WIS
- authored lines keyed by domain from the companion's Dialogue component; async LLM fallback per ADR-0008 with portrait + pause indicator
- evaluation only at response-choice points, after the NPC finishes

### Witness Reactions (Thread side)

- On thread resolution (ADR-0010 resolution model, TR-thread-006), present companions with domain overlap contribute one guaranteed reaction each — dialogue line + optional journal entry. Caps do not apply (AC-8).
- Journal entries follow the Investigation Journal epic's canon rules (ADR-0008 journal-canon boundary: journal records KG-verified facts only).

### Tuning Knobs (external config)

`interjection_confidence_threshold` 0.7 (0.5–0.95), `interjections_per_companion_cap` 1 (0–3), `total_interjections_cap` 3 (1–6).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 003: Recruitment itself (this story covers what active companions contribute)
- AI Dialogue Engine epic: The interjection state machine and LLM fallback internals (this story defines the contract; ADR-0008 revision owns the mechanism)
- Question System epic: Topic matching internals (this story supplies the domain gate)

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-1/AC-2**: Domain-gated question present/absent by companion presence (and alternative coverage)
- **AC-3**: `exclusive: true` inaccessible while companion absent; non-exclusive still covered by others
- **AC-4**: Interjection at response-choice points only; never mid-sentence; after NPC finishes
- **AC-5**: Caps enforced — 1/companion, 3/conversation
- **AC-6**: Priority chain confidence → trust → WIS
- **AC-7**: Authored line preferred; LLM fallback async with portrait + pause indicator
- **AC-8**: Thread resolution → guaranteed reaction per present domain-overlapping companion, uncapped
- **Edge cases**: departure keeps already-discovered facts; degraded LLM mode still allows authored interjections

---

## Test Evidence

**Story Type**: Integration
**Required evidence**:
- Integration: `tests/integration/party/knowledge_contribution_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 003 (recruitment — active membership), AI Dialogue Engine epic (complete), Question System epic (complete), Thread Investigation epic (complete)
- Unlocks: None

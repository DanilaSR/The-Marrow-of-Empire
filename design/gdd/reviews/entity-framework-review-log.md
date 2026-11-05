# Entity Framework — Review Log

## Review — 2026-08-09 — Verdict: MAJOR REVISION NEEDED (revised same session)
Scope signal: XL
Specialists: game-designer, systems-designer, qa-lead, lead-programmer, creative-director (senior)
Blocking items: 8 | Recommended: 6 | Nice-to-have: 3

**Summary**: Four independent adversarial reviews converged on the same conclusion from game design, systems engineering, QA, and engine architecture perspectives. The GDD was structurally sound in ambition but required major revision on two fronts: (1) it designed a parallel entity system where Kaiju Engine already ships one — every visible entity must be an `engine.Entity` with `Transform: &entity.Transform` on its Drawing, making a parallel Transform a permanent dual-sync bug source; (2) it had no defined memory model, meaning the write semantics (staged vs. immediate, delta vs. assignment) that every consumer system depends on were undefined and self-contradictory. Additional blocking findings: Persistent component was boolean-only with four concurrent writers and no namespacing; Suspended state froze entities (contradicting living schedules and "no permanently missed clues"); five gameplay-essential components were missing; lifecycle events could be silently dropped; and 0 of 8 acceptance criteria were independently testable. Six ADRs were flagged as pre-implementation gates.

**Revision (2026-08-09, same session)**: All 8 blocking items resolved. GDD rewritten as game-component layer over Kaiju entities (CR-1 through CR-8). Memory model defined (CR-7: staged writes, delta accumulation, absolute assignment priority resolution). Persistent redesigned with namespaced flags and memory_log. Suspended redefined as reduced-fidelity tick with guards. Component schema completed (15 components, InteractionType enumerated, PlayerKnowledge added). Lifecycle events guaranteed synchronous. 20 testable ACs with full performance methodology. 6 ADRs flagged in Open Questions. Pending re-review.

Prior verdict resolved: First review

## Review — 2026-08-09 — Verdict: APPROVED (re-review)
Scope signal: XL
Specialists: none (lean mode — re-review of prior MAJOR REVISION)
Blocking items: 2 (cross-GDD coordination) | Recommended: 3
Summary: All 8 prior blocking items confirmed resolved. The EF now correctly wraps Kaiju entities, the memory model (CR-7) is fully specified with staged writes/delta accumulation/absolute assignment, 15 components catalogued, lifecycle events guaranteed synchronous, 20 testable ACs. Two new cross-GDD items flagged: Save/Load serialization ownership is ambiguous between EF CR-8 and the Save/Load GDD, and "Inventory / Loot System" is referenced in InteractionType but doesn't exist in the systems index. Neither blocks EF implementation — both are coordination items for downstream GDD reviews.
Prior verdict resolved: Yes (MAJOR REVISION NEEDED — all 8 blockers resolved)

> **Note (2026-08-09)**: The above lean-mode APPROVED entry is superseded by the full adversarial re-review below. The serialization items the lean review listed as "coordination niceties" were found to be blocking budget contradictions (100× per-entity discrepancy with save-load-system.md) under adversarial analysis.

---

## Review — 2026-08-09 — Verdict: NEEDS REVISION (second full review, revised same session)
Scope signal: XL
Specialists: systems-designer, performance-analyst, qa-lead, lead-programmer (truncated — concurrent-updater verification pending), creative-director (senior synthesizer)
Blocking items: 11 → 0 (all resolved in revision) | Recommended: 8

**Summary**: Second full adversarial review confirmed the architecture held — no specialist attacked the wrap-Kaiju premise, the CR-7 memory-model rules, or the 15-component schema. AC testability went from the prior review's 0/8 → 13/20 — the structural fix worked. But precision failures remained: (1) the serialization budget contradicted save-load-system.md (100ms/2500 vs 2s/500 — 100× per-entity discrepancy); (2) CR-8 save pass omitted Transform, hierarchy, Tags, and the ID counter — loaded worlds would have broken camera tracking and ID collisions; (3) the write-ownership table covered 9 rows for 15 components, leaving Combatant.movement_points with 3 legitimate writers and no owner; (4) the event model contradicted itself (synchronous dispatch claimed but event_queue_max_size implied bounded queue with drops); (5) read-your-writes mechanism was unspecified, making CR-2's O(1) claim unverifiable; (6) AC-7 float tolerance 1e-5 was finer than float32 precision at world-position magnitudes (~3e-5 ULP at 320u); (7) AC-3 fixture math double-counted a boundary entity; (8) AC-P5 zero-alloc was impossible for a runtime-length []GameEntityID without a caller-scratch-buffer contract; (9) AC-15 hysteresis had no initial state; (10) the camera dependency row was still stale from the prior revision; (11) lead-programmer's verification of CR-7's sequential-write claim against Kaiju's concurrent updater mode was truncated. Additional: 6 ACs missing for described behavior (duplicate-ID rejection, re-entrancy guard, dialogue guard, quest-critical destruction logging, event volume diagnostic, write-semantics bundle).

**Revision (2026-08-09, same session)**: All 11 blockers resolved. Three design decisions made: synchronous save (100ms pause acceptable on region transitions), synchronous event dispatch (queue knob removed, payload construction gated on subscribers), and write-ownership table completed for all 15 components. CR-8 save set extended to Transform, hierarchy, Tags, and ID counter; whole-world single-blob pod topology specified. Read-your-writes mechanism: per-entity staged-field bitmask (~50KB at 2500 entities), persistent maps cleared per frame. AC fixes: AC-3 fixture corrected (49+1 boundary), AC-7 tolerance made magnitude-relative, AC-15 initial states + oscillation sequence specified, AC-P5 caller-scratch-buffer contract added to CR-3, AC-P2 radius pinned (100u) and entity count scaled to 2500, AC-P3 scoped to synchronous dispatch. Camera dependency row updated to match camera GDD's revised contract (lifecycle events for membership + per-frame Transform reads). Six new ACs added (AC-16 through AC-21) covering all uncovered edge cases. CR-7 sequential-write verification deferred to write-semantics ADR. Edge case updated for synchronous event model. Invariants cleaned up. Tuning knob: event_queue_max_size → max_change_events_per_frame (diagnostic cap, not drop threshold).

Prior verdict resolved: Yes — NEEDS REVISION → all 11 blockers resolved in revision → pending re-review

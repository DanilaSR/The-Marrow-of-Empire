---
name: gdd-review-context
description: How GDD adversarial reviews work in this project, cross-GDD budget facts, and review log conventions
metadata:
  type: project
---

Adversarial GDD reviews: reviews are appended to `design/gdd/reviews/[system]-review-log.md` with verdicts (APPROVED / NEEDS REVISION / MAJOR REVISION NEEDED), blocking/recommended/nice-to-have counts, and specialist list. The performance-analyst participates as a specialist; findings must be quantitative (measure, don't estimate).

Cross-GDD budget facts as of 2026-08-09:
- EF GDD (post two same-day revisions): serialization budget = full save of 2500 entities in 100ms, deserialize in 200ms (AC-P4); explicit decision "synchronous save (100ms pause acceptable on region transitions)"; whole-world single-blob pod topology; floats float32 with magnitude-relative tolerance; 15-component schema; world scale 2500 entities across 5 regions.
- Save/Load GDD: 2s save / 3s load / no frame drops below 30fps / 5MB cap, framed at 500 entities. EF review log already flagged the 100x per-entity discrepancy (100ms/2500 vs 2s/500); save-load-system.md was NOT revised. The 30fps claim also contradicts EF's accepted 100ms sync pause (100ms > 33.3ms frame).
- `technical-preferences.md` memory ceiling and draw-call budgets are still [TO BE CONFIGURED]. Authoritative model is now Fiendish_LLAMA_3B.Q4_K_M.gguf (2.24 GB in repo root): LLM budget ~3.2-4.2 GiB @ 4096 ctx, 16 GiB minimum spec (see [[llm-model-measurements]]). The earlier 4.5-5.5 GiB figure was for the superseded NPC-LLM-3_8B phi3 model.
- AI Dialogue GDD (reviewed 2026-08-09, verdict MAJOR REVISION NEEDED): CR-1 claims 2048 ctx vs model native 4096; "<4GB total" fails (LLM alone 4.1-5.3 GiB, see [[llm-model-measurements]]); 2 s/90% AC impossible on CPU (prefill 5-16 s alone); 5 s timeout on CPU-tier = permanent 3-timeout restart loop; crash-recovery requires separate-process IPC (currently an open question); model file untracked in repo root — must ship via Steam depot. Blocking items: hardware tiers, TTFT vs decode split, prompt token budget, concurrency/cancellation, storage tier, VRAM budget, thread pinning.

**Why:** Design docs get written with hand-waved budgets; adversarial review with measurements is the mechanism that catches them before implementation.

**How to apply:** When reviewing save/load, entity framework, or any persistence design, anchor to these facts and the [[pod-serialization-measurements]] numbers. Flag cross-GDD contradictions explicitly — the EF review log pattern (list each contradiction with quantities) is the established style.

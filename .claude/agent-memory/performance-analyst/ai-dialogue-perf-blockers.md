---
name: ai-dialogue-perf-blockers
description: Performance-critical findings from 2026-08-09 AI Dialogue GDD review — cacheability contradiction, timeout-gate prefill bug, bandwidth saturation; reported, awaiting GDD revision
metadata:
  type: project
---

Reported to parent review 2026-08-09 (2 Critical / 5 High / 5 Medium / 4 Low). Items likely to resurface in re-review:

- **C1 cacheability contradiction**: CR-3 fact selection is topic-matched to the player's question, but the KV-cache static prefix claims S_facts/S_referrals are invariant (CR-1/CR-2 formula). As written, cache hits ≈ never → TTFT 9.4 s desktop / 18.8 s min-spec; the 2 s target collapses. Fix options: question-independent fact ranking, per-topic cache slots (key = NPC × trust-tier × topic), or cache prefix ending before Knowledge (~385 tokens → TTFT 6.7-13 s).
- **C2 timeout-gate math**: gate 10 tok/s derived decode-only (150/25 × 1.7 margin). Total at gate without caching = 33.7 s > 25 s timeout → max-length responses time out on hardware that PASSED the probe. GDD's "~19 s maximum wait" omits decode. Also at desktop+gate: 24.3 s, margin 0.7 s (2.8%).
- **H1 bandwidth**: decode reads 2.09 GiB/token → 20 tok/s ≈ 42 GiB/s ≈ DDR4-3200 dual-channel saturation; 60 fps during inference unverified (ADR-0008 "game loop unaffected" is false at this bandwidth); min-spec 4c/8t + 8 inference threads = 2× oversubscription; no thread pinning/priority anywhere.
- H2: AC-21 ≤12 s generation-complete needs ≥~14 tok/s decode (fails at probe gate); typewriter reveal (45-60 cps ≈ 11-15 tok/s) visibly stutters below that.
- H5: rolling-summary inference = 4.5-9 s full-contention grind after each exchange; "low-priority" is unspecified and unenforceable (single subprocess, single slot).
- M2: warm-up step 3 redundant — probe prefill already pages all 2.09 GiB of weights.
- M3: probe decode length unspecified → <12 s warm-up budget unverifiable at gate speed (17 s probe alone at 10 tok/s).
- M4: "HDD not supported" has no detection mechanism — probe measures compute, not storage; HDD first-touch 13-21 s.
- M5: thermal detection reactive only; passive decode-speed EMA is free (tokens already streamed).
- Also: VRAM/UMA unaccounted (Steam Deck = the min spec: 4c/8t, 16 GB unified); memory ceiling in technical-preferences.md still [TO BE CONFIGURED].

**Why:** These are the performance blockers the GDD must resolve before implementation; prompt caching is an Open Question but the entire latency contract (2 s/12 s) depends on it.

**How to apply:** When re-reviewing the AI Dialogue GDD or reviewing the ADR-0008 update / llama.cpp IPC design, verify each item is addressed. Numbers in [[llm-model-measurements]].

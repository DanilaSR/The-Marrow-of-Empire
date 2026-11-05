---
name: llm-model-measurements
description: Measured GGUF header facts for both model files (Fiendish 3B authoritative, NPC-LLM phi3 superseded) and derived KV/memory/bandwidth math for AI Dialogue reviews
metadata:
  type: reference
---

Measured 2026-08-09 by parsing GGUF v3 headers of both files in repo root.

**Authoritative model — Fiendish_LLAMA_3B.Q4_K_M.gguf** (2,241,001,056 B = 2.09 GiB; 256 tensors; file_type 15 = Q4_K_M; size_label 3.6B):
- llama arch, 28 layers, 24 Q heads / 8 KV heads (GQA), head_dim 128, embedding 3072, vocab 128256, native ctx 131072 (rope freq_base 5e5).
- KV cache (FP16): 2×28×8×128×ctx×2 B = 448 MiB @ 4096, 896 MiB @ 8192. Static-prefix KV @ ~1100 tok ≈ 120 MiB.
- Decode is bandwidth-bound: 2.09 GiB read per token → 20 tok/s ≈ 42 GiB/s = DDR4-3200 dual-channel saturation (~40 GB/s real); single-channel laptop caps ~11-12 tok/s; GDD's 28 tok/s top-end implies ~58 GB/s (DDR5/quad-channel). Use 18-22 tok/s on DDR4 in budget math.
- Runtime memory ≈ 2.09 (weights, mmap'd) + 0.44 (KV @4096) + 0.5-1.5 (compute buffers, batch-dependent) + ~0.2 overhead ≈ 3.2-4.2 GiB @4096; up to ~4.8-5.3 GiB @8192.

**Superseded model — NPC-LLM-3_8B-Q5_K_M.gguf** (2.82 GB; phi3, 32 layers, MHA 32 KV heads, head_dim 96, native ctx 4096): KV 0.75 GiB @2048 / 1.5 GiB @4096; runtime ~4.1-5.3 GiB. No longer the GDD's model; ADR-0008 still references it (needs update).

Latency math (pessimistic llama.cpp estimates): CPU prefill 50-150 t/s (desktop, contended), decode 6-20 t/s; 1400-token prompt TTFT 9-19 s. At probe gate 10 tok/s, no-caching generation total = 18.7 (prefill) + 15 (150 tok decode) = 33.7 s > 25 s timeout — gate math must include prefill. 2 s TTFT target requires prompt caching; in-memory slot KV save/restore adds ~0 memory (KV preallocated), disk-backed adds ~120 MiB per saved conversation.

How to apply: cite these numbers in AI Dialogue GDD reviews and ADR-0008 updates. Link [[gdd-review-context]] and [[ai-dialogue-perf-blockers]].

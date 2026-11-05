# Performance Analyst — Memory Index

- [pod-serialization-measurements](pod-serialization-measurements.md) — Measured pod encode/decode throughput, allocs, 2x []byte bloat, 10x unbuffered-write penalty, non-determinism (2026-08-09)
- [gdd-review-context](gdd-review-context.md) — Adversarial review workflow, EF vs Save/Load budget contradictions, config gaps
- [llm-model-measurements](llm-model-measurements.md) — GGUF facts for both models (Fiendish 3B: 28L/8KV/128d, KV 448 MiB@4096, 2.09 GiB/token decode = DDR4 saturation; phi3 superseded), runtime 3.2-4.2 GiB
- [ai-dialogue-perf-blockers](ai-dialogue-perf-blockers.md) — 2026-08-09 review: static-prefix/caching contradiction (C1), timeout-gate ignores prefill (C2), bandwidth saturation, warm-up redundancy, VRAM/HDD/thermal gaps

---
name: camera-gdd-perf-review
description: 2026-08-09 performance review of camera-system.md — headline findings, budget state, and the EF contract contradictions that must be resolved before implementation
metadata:
  type: project
---

Performance review of `design/gdd/camera-system.md` completed 2026-08-09 (design phase; no builds to measure against yet).

**Budget state as of review**: camera update 1.0ms p95 CPU + zoom transition 0.5ms + whole-frame 16.6ms p95 ACs exist in the camera GDD. Reference hardware still TBD by technical-director. Draw-call budget and global memory ceiling still "TO BE CONFIGURED" in `.claude/docs/technical-preferences.md`.

**Headline findings** (details returned in the review itself; these are the load-bearing facts for future perf work):
1. Camera math is ~10-50µs/frame — the 1.0ms budget is generous. The real camera-adjacent cost is renderer-side: Kaiju re-runs frustum culling for ALL draw instances every frame the camera moves (`ViewChanged()` in `rendering/draw_instance.go` is true whenever `frameDirty` is set, which is every `SetPosition`). Budgets misattribute this to nothing.
2. The camera's EF contract ("no per-frame range queries, incremental centroid from cached list") is unbuildable as written: EF `on_component_changed` explicitly does NOT fire for Transform changes (EF CR-5), so positions cannot be cached from events — only membership. Positions must be read directly from `entity.Transform` per frame.
3. CR-8 occlusion pre-filter ("pre-filtered by camera frustum") contradicts the same contract: it requires a per-frame spatial query, and the EF uniform grid only supports circular XZ range queries, not frustum queries. Needs an explicit query shape/radius/cadence.
4. Cross-system hitch risks vs the 33ms GC cap: EF save round-trip ≤100ms in-frame (EF AC-P4), LLM result marshalling, region-transition asset loads. LLM native memory (2.7GB GGUF) is outside the Go heap — GC pressure comes from allocation rate, not the model footprint.
5. AC statistics/fixture problems: perf fixture = 60 entities (10 combatants + 50 environment) but static — cannot exercise the EF event processing the 1.0ms budget was raised for. p95 over 600 frames excludes the worst 30 frames by construction (contradicts "no frame drops" requirement, line 562); 33ms GC cap = 2 visible dropped frames. Reference hardware TBD voids all four perf ACs. Zero-allocation hot-path AC (mirror EF AC-P5) is the highest-value addition.
6. Occlusion owner gap: the deps table references an "Occlusion System" (HARD downstream) that has no GDD and no systems-index entry. CR-8's fade mechanism (per-instance ShaderData alpha vs material swap) unspecified — batch-splitting impact unbudgeted; max concurrent fades undefined.
7. Engine-API implementation contracts to mandate when camera is built: update in the Updater stage (before Render; Windows has a dedicated render thread); use `SetPositionAndLookAt` for follow/shake (single view rebuild); use `Resize`/`SetProperties` for zoom/resize — never `SetWidth`+`SetHeight` (two projection rebuilds); coalesce window-resize events to one apply per frame. `sizeIsViewSize` clobbering is mostly impossible by construction (standard_camera.go:341) except by coincidence.

**Why:** the camera GDD will be revised after this review; these contradictions (2 and 3) are subtle cross-document issues that could easily survive revision and resurface at implementation time.

**How to apply:** when reviewing later camera GDD revisions or the occlusion system, first check whether the position-cache mechanism and occlusion query cadence are now specified. When the technical-director configures draw-call/memory budgets, this review's recommendations are the reference. Related: [[entity-framework-contract]] (not yet written).

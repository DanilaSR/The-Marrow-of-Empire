# Agent Memory Index

- [Camera System GDD review verdict](creative-director/camera-system-gdd-review.md) — 2026-08-09 (3rd review): STILL MAJOR REVISION NEEDED; direction sound, but EF position-cache unbuildable, CR-6 padding math ~50% wrong, state model false, shake ACs unsatisfiable
- [Entity Framework GDD review verdict](creative-director/entity-framework-gdd-review.md) — 2026-08-09: MAJOR REVISION NEEDED; wrap Kaiju entities not parallel; 6 ADRs; position-cache unbuildable — camera/EF corrections must be coordinated
- [Camera GDD performance review](performance-analyst/camera-gdd-perf-review.md) — 2026-08-09: p95/33ms ACs hide hitches; fixture static (60 entities); occlusion owner + fade mechanism unspecified; EF position-cache unbuildable; engine API contracts for camera implementation
- [pod serialization measurements](performance-analyst/pod-serialization-measurements.md) — 2026-08-09: encode 14.9MB/s/681k allocs, 2x []byte bloat, 10x unbuffered-write penalty, non-deterministic output, no Steam Cloud API in engine wrapper
- [GDD review context](performance-analyst/gdd-review-context.md) — 2026-08-09: EF 100ms/2500 vs Save/Load 2s/500 = 100x discrepancy; 30fps claim contradicts EF sync-100ms-pause; memory ceiling still unconfigured
- [UX docs state](ux-designer/ux-docs-state.md) — design/ux/ empty; 2026-08-09: 2nd camera review (wheel zoom, markers, prompt) + 3rd Main Menu & Settings review (prompt contradiction, 7 orphaned camera-access settings, keybinding undo, display-mode safe mode)
- [Title screen visual brief](art-director/project_title-screen-visual.md) — 2026-08-09: riverside camp scene overrides State 6 ruin; panel left-third; 4 conflicts flagged; 4 art bible amendments pending user approval

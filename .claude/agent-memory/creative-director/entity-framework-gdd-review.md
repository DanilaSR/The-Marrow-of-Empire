---
name: entity-framework-gdd-review
description: Entity Framework GDD review verdict — 2026-08-09 MAJOR REVISION NEEDED; wrap Kaiju entities, 6 ADRs before coding, position-cache unbuildable
metadata:
  type: project
---

Entity Framework GDD (`design/gdd/entity-framework.md`) reviewed 2026-08-09. Verdict: MAJOR REVISION NEEDED.

**Why:** Two structural failures. (1) Parallel-entity-architecture premise rejected — EF must WRAP Kaiju entities, not duplicate them (a parallel entity layer breaks engine integration and doubles lifecycle complexity). (2) The position-cache mechanism the EF GDD advertises to consumers is unbuildable: Kaiju has no position-change event, and `on_component_changed` does not fire for Transform changes. Camera System (a listed EF consumer) inherited this broken contract — its cached-party-position-list design depends on events that do not exist.

**How to apply:** When camera/combat/AI GDDs declare EF event contracts, require: membership lifecycle events (created/destroyed/tag-changed) + per-frame Transform reads for tracked sets; never position-change events. ~6 ADRs required before EF implementation (entity wrapping, event catalog, tag queries, spatial index, lifecycle ownership, component change semantics). The camera GDD revision and the EF GDD revision must be coordinated in the same pass. Related: [[camera-system-gdd-review]].

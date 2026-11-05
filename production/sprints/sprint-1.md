# Sprint 1 — Entity Architecture Foundation

**Dates**: 2026-08-09 to 2026-08-15 (5 days)
**Milestone**: Pre-Production — Foundation Layer

## Sprint Goal
Establish the entity architecture — the root of the dependency graph. Typed component stores, spatial index, lifecycle events, write arbitration, and pod serialization. Nothing else can be built until entities exist.

## Capacity
- Total: 5 days (solo developer, ~30 hours)
- Buffer (20%): 6 hours reserved for unplanned work / engine verification spikes
- Available: ~24 hours

## Tasks

### Must Have (Critical Path)

| ID | Task | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|----------|-------------|-------------------|
| EF-001 | Entity Lifecycle | 4-6 | — | GameEntityID mapping, Pending→Active→Suspended→Destroyed, lifecycle events broadcast |
| EF-002 | Component Model | 4-6 | EF-001 | Typed component stores, GetComponent/SetComponent, component lifecycle events |
| EF-003 | Spatial Index | 4-6 | EF-001 | Uniform grid, range queries, bucket overflow, component-filtered queries |
| EF-004 | Write Arbitration | 2-3 | EF-002 | System priority resolution, same-frame conflict detection, uncontested write passthrough |
| EF-005 | Serialization | 4-6 | EF-002, EF-003 | pod.Register, EntityData interface, save/load round-trip with identical entity state |
| EF-006 | Performance Validation | 2-3 | EF-001–005 | Benchmarks: create <0.5ms, query <1ms, destroy <2ms, lookup ~50ns |

### Should Have

| ID | Task | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|----------|-------------|-------------------|
| CAM-001 | Isometric Projection | 4-6 | — (parallel) | Orthographic 30°/45°, Kaiju StandardCameraOrthographic, aspect ratio handling |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Kaiju engine API mismatch | Medium | High | Every ADR has "Verification Required" — verify entity/transform APIs against `kaiju/src/` before each story |
| pod serialization limitations | Low | Medium | ADR-0002 notes pod may not support delta encoding — full-state is sufficient for MVP |
| Entity count performance | Low | Medium | Spatial index bucket overflow degrades to linear scan — tune bucket_size if needed |

## Dependencies on External Factors
- Kaiju Engine source in `kaiju/` must be readable for API verification
- CGO build environment must be functional (Vulkan SDK, Go 1.26)

## Definition of Done for this Sprint
- [ ] All 6 Must Have stories completed
- [ ] All acceptance criteria verified
- [ ] Entity creation benchmark passes (<0.5ms)
- [ ] Spatial query benchmark passes (<1ms for 500 entities)
- [ ] Save/load round-trip produces identical entity state
- [ ] Code committed to git

> ⚠️ **No QA Plan**: This sprint was started without a QA plan. Run `/qa-plan sprint` before the last story is implemented.

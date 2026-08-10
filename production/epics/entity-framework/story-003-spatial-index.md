# Story 003: Spatial Index

> **Epic**: Entity Framework
> **Status**: Complete
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Integration
> **Estimate**: M (4-6 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001 (Entity Lifecycle), Story 002 (Component Model — `matchesComponents` for AC-4)

## Context

**GDD**: `design/gdd/entity-framework.md`
**Requirement**: `TR-ef-003`

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: Uniform grid spatial index: `map[[2]int32][]GameEntityID`. Bucket size configurable (default 32 world units). Entities update their bucket on position change.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH

**Control Manifest Rules**: Required: Uniform grid spatial index for range queries. Guardrail: Spatial query <1ms for 500 entities.

## Acceptance Criteria

- [ ] GIVEN 100 entities at various positions, WHEN QueryEntities is called with a position and range, THEN only entities within range are returned
- [ ] GIVEN an entity that moves from one bucket to another, WHEN its position is updated, THEN the spatial index reflects the new bucket
- [ ] GIVEN a query spanning multiple buckets, WHEN the query range crosses bucket boundaries, THEN entities from all overlapping buckets are aggregated correctly
- [ ] GIVEN a query with component filter (e.g., "Health AND Combatant"), WHEN QueryEntities runs, THEN only entities with all specified components in range are returned
- [ ] GIVEN bucket capacity exceeded (>500 entities in one bucket), WHEN a query runs, THEN it degrades to linear scan with warning logged

## Implementation Notes

From ADR-0001: Bucket key = `[2]int32{int32(x / bucketSize), int32(y / bucketSize)}`. Query collects all buckets overlapping the search circle, then filters by exact distance and component requirements. Bucket size default 32 world units (configurable via tuning knob).

## Out of Scope

- Story 004: Write arbitration — spatial index does not resolve multi-system position conflicts
- Story 005: Serialization — index state (bucket assignments) is not persisted to save files by this story
- Story 006: Performance validation — formal benchmarks are measured separately; this story only ensures the degradation path works

## QA Test Cases

- **AC-1 — Range query**:
  - Given: 100 entities, one at (10, 20), another at (50, 80), rest scattered
  - When: QueryEntities(position=(10,20), range=15, components=["Health"])
  - Then: Entity at (10,20) returned. Entity at (50,80) not returned.
  - Edge cases: Range=0 returns only entities at exact position.

- **AC-4 — Component filter**:
  - Given: Entity A has Health+Combatant at (10,20). Entity B has Health only at (10,20).
  - When: QueryEntities(position=(10,20), range=10, components=["Health","Combatant"])
  - Then: Only Entity A returned.

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/entity/spatial_index_test.go` — must exist and pass

| AC | Test function | What it verifies |
|----|--------------|------------------|
| AC-1 | `test_range_query_entities_within_radius` | QueryEntities returns only entities within range, excludes those outside |
| AC-2 | `test_position_update_migrates_bucket` | Entity moves to new bucket, subsequent queries reflect new position |
| AC-3 | `test_multi_bucket_query_aggregates_correctly` | Query spanning bucket boundaries returns entities from all overlapping buckets |
| AC-4 | `test_component_filter_within_range` | Only entities with all specified components returned; entity at same position missing one excluded |
| AC-5 | `test_bucket_overflow_degrades_to_linear_scan` | >500 entities in one bucket triggers linear scan with warning logged

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 5/5 passing
**Deviations**: Test names use `Test` prefix (Go requirement) vs story's `test_*`; Range=0 keeps existing "no spatial filter" contract (existing tests depend on it)
**Test Evidence**: `src/tests/integration/entity/spatial_index_test.go` — 5 test functions, all passing; 17 unit tests also pass (zero regressions)
**Code Review**: Skipped (Lean mode — user confirmed review before sprint close-out)

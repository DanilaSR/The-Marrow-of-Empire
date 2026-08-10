# Story 004: Write Arbitration

> **Epic**: Entity Framework
> **Status**: Ready
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 002 (Component Model)

## Context

**GDD**: `design/gdd/entity-framework.md`
**Requirement**: `TR-ef-005`

**ADR Governing Implementation**: ADR-0001: Entity Architecture
**ADR Decision Summary**: Write arbitration resolves same-frame modifications to the same component field. Systems register a priority value (1 = highest). Lower-priority modifications that conflict are discarded with warning. Equal priority = last write wins.

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH

**Control Manifest Rules**: Required: Write arbitration by system priority. Forbidden: Never use direct cross-system calls for state mutations.

## Acceptance Criteria

- [ ] GIVEN two systems with priorities 5 and 15 attempting to modify the same component field in the same frame, WHEN the frame resolves, THEN the priority-5 system's value is applied and the priority-15 system receives a warning
- [ ] GIVEN two systems with equal priority modifying the same field, WHEN the frame resolves, THEN the last writer's value is applied
- [ ] GIVEN a system modifying a field that no other system touches, WHEN the frame resolves, THEN the modification is applied without arbitration overhead

## Implementation Notes

From ADR-0001: Each component field tracks its last writer (system name + priority) per frame. On SetComponent, if a different system already wrote this field this frame, compare priorities. Lower number wins. Discarded writes log a warning and the calling system receives an error. Delta accumulation: changes are staged per frame and committed at frame end.

## Out of Scope

- Story 005: Serialization — arbitration state (last-writer per field) is not persisted to save files
- Story 006: Performance validation — arbitration overhead is measured by benchmarks, not this story
- GDD CR-7 ownership table authoring — the field-to-system ownership mapping is a design artifact, not runtime code
- Conflict debugging UI — Presentation layer concern; this story provides the logging/error return, not a visual debugger

## QA Test Cases

- **AC-1 — Priority resolution**:
  - Given: Combat system (priority 5) calls SetComponent for Health.Current = 0. VFX system (priority 15) calls SetComponent for Health.Current = 50 in same frame
  - When: Frame resolves
  - Then: Health.Current = 0 (priority 5 wins). VFX system receives warning.

- **AC-2 — Equal priority**:
  - Given: System A (priority 10) and System B (priority 10) both write Health.Current in the same frame, System B writes last
  - When: Frame resolves
  - Then: System B's value is applied (last writer wins). No warning emitted (equal priority is not a conflict — ADR-0001 specifies deterministic tie-breaking).

- **AC-3 — Uncontested write**:
  - Given: Only one system writes Health.Current this frame
  - When: Frame resolves
  - Then: Modification applied with zero arbitration overhead.

## Test Evidence

**Story Type**: Logic
**Required evidence**: `tests/unit/entity/write_arbitration_test.go` — must exist and pass

| AC | Test function | What it verifies |
|----|--------------|------------------|
| AC-1 | `test_priority_resolution_lower_wins` | Priority-5 write wins over priority-15 write; loser receives error |
| AC-2 | `test_equal_priority_last_writer_wins` | Same priority, last write applied, no warning |
| AC-3 | `test_uncontested_write_passthrough` | Single writer: no arbitration overhead, value applied directly |

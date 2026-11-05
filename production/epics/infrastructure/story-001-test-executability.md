# Story INFRA-001: Fix Test Executability (go.work)

> **Epic**: Infrastructure
> **Status**: Complete
> **Last Updated**: 2026-08-10
> **Layer**: Foundation
> **Type**: Config/Data
> **Estimate**: XS (0.5 hours)
> **Sprint**: Sprint 2
> **Retro Action**: BLOCKING from Sprint 1 retrospective (2026-08-10)
> **Dependencies**: None — standalone build configuration task. CI-001 depends on this story.

## Context

Sprint 1 delivered 6 stories with ~1,800 lines of test code, but tests cannot be executed: the `tests/` directory lives outside the Go module root (`src/`). There is no `go.work` file. `go test` fails with "directory prefix does not contain main module."

**GDD**: N/A — infrastructure task
**ADR**: N/A — no architectural decision involved; this is a Go build toolchain configuration task
**Engine Notes**: N/A — no Kaiju engine APIs are involved; this is a Go workspace configuration task
**Control Manifest**: N/A — `go.work` configuration is a build toolchain concern, not covered by Foundation Layer rules

## Acceptance Criteria

- [x] `go.work` file exists at repo root referencing `src` and `tests`
- [x] `go test marrow-of-empire-tests/...` from repo root passes all Sprint 1 tests (EF-001 through EF-006)
- [x] No test assertions modified — tests pass as-is or any fixes are documented in commit message

## Out of Scope

- Modifying test assertions or test logic — tests should pass as-is; if they don't, document any fixes in the commit message but don't rewrite tests
- Restructuring the `tests/` directory layout
- Creating new test files for uncovered code
- Setting up CI/CD (covered by CI-001)

## Performance

No runtime impact — `go.work` is a build configuration file read by the Go toolchain at compile time; it does not affect game performance, memory usage, or frame time.

## Implementation Notes

One-line fix: `go work init && go work use src tests` at repo root. May require adding `go.work` to `.gitignore` per team convention, or committing it (Go workspace files are typically committed).

## QA Test Cases

> Back-filled by /qa-plan 2026-08-10.

**Manual verification only**:
- [x] `go.work` file exists at repo root referencing `src` and `tests`
- [x] `go test marrow-of-empire-tests/...` from repo root passes all Sprint 1 tests
- [x] No test file modifications needed (tests pass as-is) OR test fixes documented in commit message

## Test Evidence

- **Type**: Config/Data
- **Evidence**: `go test marrow-of-empire-tests/...` output showing PASS for all Sprint 1 test packages
- **Location**: Commit message body or CI log

## Completion Notes
**Completed**: 2026-08-10
**Criteria**: 3/3 passing
**Deviations**: None
**Test Evidence**: Config/Data — go.work + tests/go.mod created; `go test marrow-of-empire-tests/...` passes (4/4 packages, 41 tests)
**Code Review**: Skipped (Config/Data story — no Go code to review)
**Commits**: d42fca8

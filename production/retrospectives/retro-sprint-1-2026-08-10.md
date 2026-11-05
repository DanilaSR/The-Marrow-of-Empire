## Retrospective: Sprint 1 — Entity Architecture Foundation

Period: 2026-08-09 -- 2026-08-15
Generated: 2026-08-10

### Metrics

| Metric | Planned | Actual | Delta |
|--------|---------|--------|-------|
| Tasks (Must Have) | 6 | 6 | 0 |
| Tasks (Should Have) | 1 | 0 | −1 |
| Completion Rate (Must Have) | -- | 100% | -- |
| Overall Completion Rate | -- | 86% | -- |
| Story Points / Effort Hours | 24 hrs | ~8–10 hrs | −14 hrs |
| Commits | -- | 2 | -- |
| Unplanned Tasks Added | -- | 0 | -- |
| Bugs Found | -- | 0 | -- |

### Velocity Trend

| Sprint | Planned | Completed | Rate |
|--------|---------|-----------|------|
| Sprint 1 (current) | 7 (6 must, 1 should) | 6 (all must) | 86% |

**Trend**: Baseline. First sprint establishes the reference point for future velocity comparisons.

### What Went Well
- **All 6 critical-path stories completed** — the entity architecture foundation (lifecycle, components, spatial index, write arbitration, serialization, performance benchmarks) shipped functional and complete
- **Significant test coverage written** — 4 test files totaling ~1,800 lines covering unit tests for lifecycle (EF-001), component model (EF-002), write arbitration (EF-004), spatial index (EF-003, integration), and serialization (EF-005, integration), plus benchmarks (EF-006)
- **Clean codebase** — zero TODO/FIXME/HACK markers in the entity source; code was completed, not shipped with known gaps
- **Substantial implementation** — 8 source files (~2,300 lines) plus test infrastructure, representing the root of the entire dependency graph

### What Went Poorly
- **Tests are not executable** — the `tests/` directory lives outside the Go module root (`src/`). There is no `go.work` file, so `go test` fails with "directory prefix tests/unit/entity does not contain main module." This means the 1,800 lines of test code cannot be run, making the test evidence required by the sprint's Definition of Done unverifiable
- **Massive estimation error** — all 6 must-have stories were estimated at 20–30 hours total (4–6 hrs each). All were completed on the same day (2026-08-10). This is a ~60–70% overestimate, which undermines sprint planning accuracy for future iterations
- **QA plan generated after completion** — the QA plan (`qa-plan-sprint-1-2026-08-10.md`) was generated on the same date the sprint completed, not before implementation began
- **CAM-001 (Isometric Projection) left in backlog** — the one should-have story was not started, despite being flagged as parallel (no dependencies on EF stories)
- **Only 2 commits for the entire sprint** — both with identical messages, squashing 6 stories' worth of implementation into a single diff. This loses granular history and makes bisecting bugs harder
- **No CI/CD pipeline configured** — without a `go.work` or tests inside the module, no automated test suite can run on push

### Blockers Encountered

| Blocker | Duration | Resolution | Prevention |
|---------|----------|------------|------------|
| None reported | -- | -- | -- |

No formal blockers were logged. The sprint-status.yaml shows no `blocker` field values set for any story.

### Estimation Accuracy

| Task | Estimated | Actual | Variance | Likely Cause |
|------|-----------|--------|----------|--------------|
| EF-001 Entity Lifecycle | 4–6 hrs | ~2 hrs | −50% | Solo dev velocity higher than scoped; no engine API surprises |
| EF-002 Component Model | 4–6 hrs | ~1.5 hrs | −63% | Generic typing pattern is straightforward Go |
| EF-003 Spatial Index | 4–6 hrs | ~2 hrs | −50% | Uniform grid is well-understood algorithm |
| EF-004 Write Arbitration | 2–3 hrs | ~1 hr | −50% | Priority-queue pattern is simple |
| EF-005 Serialization | 4–6 hrs | ~2 hrs | −50% | pod library straightforward |
| EF-006 Performance Validation | 2–3 hrs | ~1 hr | −50% | Benchmarks scaffolded from test patterns |

**Overall estimation accuracy**: 0% of tasks within ±20% of estimate. All tasks were overestimated by 50–63%.

**Analysis**: First-sprint anchoring bias. The initial estimates included significant "unknown engine" risk buffers that didn't materialize. The Kaiju Engine entity APIs proved straightforward to work with. Future sprints should calibrate estimates down by ~40–50% for similar Logic-type stories, or maintain the buffer but explicitly label it as an engine-risk line item rather than distributing it across individual tasks.

### Carryover Analysis

| Task | Original Sprint | Times Carried | Reason | Action |
|------|----------------|---------------|--------|--------|
| CAM-001 Isometric Projection | Sprint 1 | 1 | Should-have, not critical path; all effort went to must-have stories | Carry to Sprint 2 as must-have (camera is blocking all visual work) |

### Technical Debt Status
- Current TODO count: 2 (in `src/` overall, 0 in `src/entity/`)
- Current FIXME count: 0
- Current HACK count: 0
- Trend: Baseline (first sprint)
- **Critical structural debt**: Tests are not executable — this is not a TODO/FIXME marker but a project structure defect that must be resolved before Sprint 2

### Previous Action Items Follow-Up

No previous sprint — this is the first retrospective.

### Action Items for Next Iteration

| # | Action | Owner | Priority | Deadline |
|---|--------|-------|----------|----------|
| 1 | Fix test executability — create `go.work` or restructure `tests/` into `src/` so all tests can run | lead-programmer | **BLOCKING** | Before Sprint 2 starts |
| 2 | Set up CI/CD workflow — `.github/workflows/test.yml` running `go test ./...` on push/PR | devops-engineer | High | During Sprint 2 |
| 3 | Calibrate estimates — reduce Logic-type story estimates by 40–50% for Sprint 2; separate "engine risk" as an explicit line item rather than distributing it | producer | High | Sprint 2 planning |
| 4 | Generate QA plan BEFORE sprint start — run `/qa-plan sprint` as part of sprint kickoff, not mid-sprint | producer | Medium | Sprint 2 kickoff |
| 5 | Commit per-story or per-logical-group — avoid single squashed commits for multi-story sprints | lead-programmer | Medium | Sprint 2 |

### Process Improvements
- **Use `go.work` for multi-module workspaces** — the current `src/` module + `tests/` directory pattern needs a `go.work` file at the repo root referencing both. This is a one-line fix (`go work init && go work use src tests`) but critical for testability
- **Run `/qa-plan sprint` during sprint kickoff, not mid-sprint** — the QA plan is a planning artifact, not a retrospective artifact. It defines what tests must exist before stories can be marked done
- **Adopt story-level commits** — the coding standards already specify Conventional Commits with story references. Enforce this during `/story-done` — each story completion should produce its own commit

### Summary

Sprint 1 delivered the entity architecture foundation — all 6 critical-path stories complete with substantial test coverage in a single day. This is the root of the dependency graph and nothing else can be built without it, so completing it is a genuine milestone. However, the sprint revealed systemic process issues: tests cannot be executed due to a Go module structure gap, estimates were 2–3× too high, and the QA plan came after implementation. The single most important action for Sprint 2 is fixing test executability — unverifiable tests are equivalent to no tests, and the project's testing standards require passing tests as a blocking gate.

## Retrospective: Sprint 2 — Camera System & Infrastructure Remediation

Period: 2026-08-11 -- 2026-08-17
Generated: 2026-08-11

### Metrics

| Metric | Planned | Actual | Delta |
|--------|---------|--------|-------|
| Tasks (Must Have) | 5 | 5 | 0 |
| Tasks (Should Have) | 2 | 0 | −2 |
| Completion Rate (Must Have) | -- | 100% | -- |
| Overall Completion Rate | -- | 71% | -- |
| Story Points / Effort Days | 3.3 days | ~1 day | −2.3 days |
| Commits | -- | 2 | -- |
| Unplanned Tasks Added | -- | 0 | -- |
| Bugs Found | -- | 0 | -- |

### Velocity Trend

| Sprint | Planned (Must) | Completed (Must) | Rate |
|--------|----------------|-----------------|------|
| Sprint 1 | 6 | 6 | 100% |
| Sprint 2 (current) | 5 | 5 | 100% |

**Trend**: Stable at 100% must-have completion. Actual effort continues at ~1 day per sprint against 5-day plans.

### What Went Well
- **Sprint 1's BLOCKING retro item resolved** — test executability fixed via `go.work` + `tests/go.mod`. All 80+ tests now run and pass across 5 packages. This was the single most important retro action item and it was the first story completed.
- **QA plan generated before implementation** — unlike Sprint 1 where the QA plan came after completion, Sprint 2's QA plan (`qa-plan-sprint-2-2026-08-10.md`) was written during sprint kickoff. Directly addressed Sprint 1 retro action item #4.
- **Substantial integration test coverage** — CAM-003 shipped with 17 integration tests covering the full stack (real Kaiju camera, mock host, entity store, controller). CAM-004 added 9 more. Combined with 40 unit tests, the camera system has ~66 tests across all layers.
- **Agent-driven implementation scaled well** — the engine-programmer agent handled CAM-002 (zoom, 12 tests) and CAM-003 (follow/framing/state machine, 17 tests) autonomously, including self-diagnosing and fixing 6 test failures across both implementation and test code in a single fix cycle.
- **Zero technical debt introduced** — 0 TODOs, 0 FIXMEs, 0 HACKs in all camera source files. Code shipped complete, not with known gaps. 2 pre-existing TODOs from Sprint 1 were resolved.

### What Went Poorly
- **Sprint plan didn't detect already-completed work** — CAM-001 (isometric projection) and CAM-004 (occlusion/boundaries) were already implemented before Sprint 2 began — source and test files existed, tests passed. The sprint plan treated them as new work, inflating planned effort by ~7 hours and making the "5/5 completion" partly a discovery artifact rather than a throughput measure.
- **CI/CD still not set up** — CI-001 (should-have) was not started. Sprint 1 retro item #2 remains unresolved after two sprints. Without CI, tests only run when manually invoked.
- **Still only 2 commits for the sprint** — Sprint 1 retro item #5 (per-story commits) was not addressed. The 2 commits (INFRA-001 + go.work fix) don't capture CAM-002 and CAM-003 agent work separately.
- **Should-have stories untouched** — both CI-001 and SAVE-001 remained in backlog. SAVE-001 (Save/Load Orchestration) is a Foundation-layer dependency that will block Feature-layer epics.
- **Sprint dates are fiction** — planned for 5 days (2026-08-11 to 2026-08-17), all work completed on 2026-08-10/11. Two sprints in a row completed in ~1 day.

### Blockers Encountered

| Blocker | Duration | Resolution | Prevention |
|---------|----------|------------|------------|
| CAM-003 integration test failures (6 tests) | ~1 agent fix cycle | Agent self-diagnosed: controller.Update passed seconds to zoom API (expects ms), pan speeds routed through pixel conversion, 5 test bugs in fixture coordinates and assertions | Add unit-conversion documentation to cross-module APIs; run integration tests as part of agent pre-completion checklist |
| go.work not picking up tests directory | ~5 min | Created `tests/go.mod` with replace directives; `tests/` wasn't a Go module so `go work use` ignored it silently | Document the multi-module workspace pattern in project README; sprint planner should check for this |

### Estimation Accuracy

| Task | Estimated (calibrated) | Actual | Variance | Likely Cause |
|------|----------------------|--------|----------|--------------|
| INFRA-001 | 0.5 hrs | ~0.5 hrs | 0% | Trivial one-command fix + tests/go.mod creation — estimate accurate |
| CAM-001 | 3 hrs | 0 hrs (already done) | −100% | Implemented alongside EF stories in Sprint 1; sprint plan didn't check file existence |
| CAM-002 | 2 hrs | ~6 min (agent) | −95% | Agent implemented 230 lines + 12 tests autonomously; human time would be ~1.5 hrs |
| CAM-003 | 4 hrs | ~15 min (agent) | −94% | Agent implemented 4 files (~800 lines) + 17 tests; human time would be ~3-4 hrs |
| CAM-004 | 4 hrs | 0 hrs (already scaffolded) | −100% | Scaffolded by CAM-003 agent alongside its own work; sprint plan didn't check |

**Overall estimation accuracy**: 20% (1 of 5 tasks within ±20%). Excluding the two already-completed stories: 33%.

**Analysis**: Two root causes. First, the sprint plan doesn't verify whether story implementation files already exist on disk before scoping them — a process gap, not an estimation skill gap. Second, agent-driven implementation is dramatically faster than human-driven for well-specified Logic/Integration stories. For future sprints: (a) pre-flight file existence check per story before estimating, and (b) use agent-calibrated estimates (~0.25 days for Logic, ~0.5 days for Integration) when stories will be delegated.

### Carryover Analysis

| Task | Original Sprint | Times Carried | Reason | Action |
|------|----------------|---------------|--------|--------|
| CI-001 (CI/CD Workflow) | Sprint 1 (retro action #2) | 2 | Should-have, not critical path | Escalate to must-have for Sprint 3 — CI blocks the Production → Polish gate |
| SAVE-001 (Save/Load Orchestration) | Sprint 2 | 1 | Should-have, not critical path | Carry to Sprint 3 as must-have (Foundation-layer dependency for all Feature-layer epics) |

### Technical Debt Status
- Current TODO count: 0 (previous: 2)
- Current FIXME count: 0 (previous: 0)
- Current HACK count: 0 (previous: 0)
- Trend: **Shrinking** (2 TODOs from Sprint 1 resolved)
- No areas of concern — all camera and entity code is clean

### Previous Action Items Follow-Up

| Action Item (from Sprint 1) | Status | Notes |
|-------------------------------|--------|-------|
| Fix test executability (go.work) | ✅ Done | INFRA-001 completed. 80+ tests passing across 5 packages. |
| Set up CI/CD workflow | ❌ Not Started | CI-001 in Sprint 2 backlog, still not started. Carried twice — escalate to must-have. |
| Calibrate estimates down 40-50% | ⚠️ Partial | Estimates were calibrated in sprint plan, but 2/5 stories were already done, making the calibration unrepresentative. Root cause: no pre-flight file check. |
| Generate QA plan before sprint start | ✅ Done | QA plan written during sprint kickoff, before implementation began. |
| Commit per-story | ❌ Not Done | Only 2 commits for Sprint 2. CAM-002 and CAM-003 agent work not separately committed. |

### Action Items for Next Iteration

| # | Action | Owner | Priority | Deadline |
|---|--------|-------|----------|----------|
| 1 | Set up CI/CD workflow — CI-001 carried twice, now BLOCKING | devops-engineer | **BLOCKING** | Sprint 3 start |
| 2 | Pre-flight file existence check in sprint planning — verify source + test files per story before estimating | producer | High | Sprint 3 planning |
| 3 | Adopt per-story commits — each `/story-done` produces its own commit | lead-programmer | High | Sprint 3 |
| 4 | Carry SAVE-001 as must-have — Foundation-layer save/load blocks Feature-layer epics | producer | High | Sprint 3 planning |
| 5 | Use agent-calibrated estimates for delegated stories — ~0.25 days Logic, ~0.5 days Integration | producer | Medium | Sprint 3 planning |

### Process Improvements
- **Pre-flight file existence check in sprint planning** — before estimating a story, check whether `src/[system]/` and `tests/**/[system]/` already contain implementation files. A story with existing source + passing tests is already done and should be marked Complete, not re-estimated. This would have caught CAM-001 and CAM-004.
- **Agent-driven story estimates need a separate scale** — Logic stories take an agent ~5-10 minutes; Integration stories ~15-30 minutes. The sprint plan should use agent-calibrated estimates when stories will be delegated, reserving human-calibrated estimates for stories requiring manual design decisions.
- **Two-carry = must-have escalation rule** — any story or action item carried across two sprints should automatically escalate to must-have priority. This prevents zombie backlog items (CI-001 is the canonical example).

### Summary

Sprint 2 delivered the full camera system — isometric projection, zoom, follow/framing, and occlusion/boundaries — with ~66 tests and zero technical debt. The sprint's BLOCKING retro item from Sprint 1 (test executability) was resolved on day one. However, the "5/5 completion" overstates actual throughput: two stories were already implemented before the sprint began, and two more were agent-driven in minutes. The real wins were fixing the test infrastructure gap, proving the agent-driven implementation pipeline works at scale, and generating the QA plan before implementation. The single most important action for Sprint 3 is setting up CI/CD — carried twice and now BLOCKING — followed by carrying SAVE-001 as must-have to unblock the Feature layer.

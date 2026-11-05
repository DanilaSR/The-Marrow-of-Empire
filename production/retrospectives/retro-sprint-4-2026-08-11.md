## Retrospective: Sprint 4 — NPC Knowledge Graph & Living Environment

Period: 2026-08-11 -- 2026-08-17
Generated: 2026-08-11

### Metrics

| Metric | Planned | Actual | Delta |
|--------|---------|--------|-------|
| Tasks (Must Have) | 4 | 4 | 0 |
| Tasks (Should Have) | 4 | 0 | −4 |
| Completion Rate (Must Have) | -- | 100% | -- |
| Overall Completion Rate | -- | 40% | -- |
| Commits | -- | 3 | -- |
| TODOs/FIXMEs/HACKs | -- | 0 | -- |

### Velocity Trend

| Sprint | Planned (Must) | Completed (Must) | Rate |
|--------|----------------|-----------------|------|
| Sprint 1 | 6 | 6 | 100% |
| Sprint 2 | 5 | 5 | 100% |
| Sprint 3 | 6 | 3 (2 in-progress) | ~50% |
| Sprint 4 (current) | 4 | 4 | 100% |

**Trend**: Stable must-have completion. Sprint 3 was the outlier (SAVE/CI carried forward). Actual effort per sprint remains ~1 day against 5-day plans.

### What Went Well
- **Parallel agent dispatch worked** — KG-002 and ENV-001 were dispatched simultaneously to independent packages, both completed within minutes. First use of parallel agents in a sprint — the pattern scales.
- **Per-story commits adopted** — 3 commits for 4 stories (KG-001 merged into the foundation commit). Sprint 2 retro action item #5 finally addressed after two sprints.
- **Pre-existing implementation detected by agents** — ENV-001 and KG-001 were already implemented in the working tree. Unlike Sprint 2 where this wasn't caught until story-done, the agents detected the existing code during their research phase and integrated with it cleanly rather than duplicating.
- **Zero technical debt** — 0 TODOs/FIXMEs/HACKs across all new code. 39 new tests (20 env + 7 parser + 12 trust query) all passing, plus performance benchmarks with threshold assertions.
- **Performance benchmarks added proactively** — ENV-001 shipped with threshold-asserting benchmarks (GameClock.Update: 17.5ns vs 10,000ns budget, ~570x headroom). First story to include benchmarks without being asked.

### What Went Poorly
- **2 of 4 stories already existed** — KG-001 and ENV-001 were implemented before the sprint began. The pre-flight file existence check (Sprint 2 process improvement) was not applied. Sprint 4's "4/4 completion" is partly a discovery artifact — same failure mode as Sprint 2.
- **CI/CD still not set up** — now carried across 3 sprints (retro action from Sprint 1). The two-carry escalation rule was proposed in Sprint 2 but the story was deferred rather than escalated to must-have.
- **Should-have stories untouched** — 4 stories in backlog, zero started. KG-004 (entity references), KG-005 (component integration), ENV-002 (wait/pass-time), SAVE-003 (corruption recovery) all deferred.
- **Sprint 3 was never formally closed** — no retrospective was written for Sprint 3. SAVE-001 and CI-001 remain in-progress in the yaml, making velocity comparison imprecise and leaving action items untracked.

### Estimation Accuracy

| Task | Estimated | Actual | Variance | Cause |
|------|-----------|--------|----------|-------|
| KG-001 | 0.8 days | 0 (already done) | −100% | Pre-existing implementation — sprint plan didn't check |
| KG-002 | 0.8 days | ~15 min (agent) | −97% | Agent-driven: parser, validation, 12 tests |
| KG-003 | 0.8 days | ~10 min (agent) | −98% | Agent-driven: query methods, 7 tests |
| ENV-001 | 0.8 days | 0 (already done) | −100% | Pre-existing implementation — sprint plan didn't check |

**Overall estimation accuracy**: 0% within ±20%. 50% overestimated due to undiscovered implementations, 50% due to agent-driven speed. This is now the third sprint with this pattern.

### Previous Action Items Follow-Up

| Action Item (from Sprint 2) | Status | Notes |
|------------------------------|--------|-------|
| CI/CD workflow (carried 2×) | ❌ Still not done | Now carried 3 sprints. Escalation rule proposed but not applied. BLOCKING for release gate. |
| Pre-flight file existence check | ⚠️ Not applied | 2/4 Sprint 4 stories already existed. Same failure as Sprint 2. |
| Per-story commits | ✅ Done | 3 commits for Sprint 4 |
| Carry SAVE-001 as must-have | ✅ Done | Carried to Sprint 3 |
| Agent-calibrated estimates | ⚠️ Partial | Estimates still at 0.8 days; actual agent time ~10-15 min for Logic stories |

### Action Items for Next Iteration

| # | Action | Owner | Priority |
|---|--------|-------|----------|
| 1 | CI/CD workflow — carried 3 sprints, now BLOCKING the Production → Polish gate | devops-engineer | **BLOCKING** |
| 2 | Apply pre-flight file existence check in sprint planning — `find src/ tests/ -name "*.go"` per story system before estimating | producer | High |
| 3 | Reduce agent-driven story estimates — 0.2 days for Logic, 0.3 days for Integration, 0.1 days for Config/Data | producer | Medium |
| 4 | Write Sprint 3 retrospective to close the velocity data gap | producer | Low |
| 5 | Enforce two-carry escalation — CI-001 is the proof case; any story carried twice must become must-have | producer | Medium |

### Summary

Sprint 4 delivered the NPC Knowledge Graph data model, YAML parser, trust-gated query interface, and game clock with light states — 39 tests, zero debt, 3 commits. Parallel agent dispatch proved that independent stories can be implemented simultaneously without conflicts. However, two persistent patterns are now undeniable: the sprint plan consistently fails to detect already-implemented stories (3 of 4 sprints), and CI/CD remains unresolved after 3 sprints despite being flagged as BLOCKING in every retrospective. The single most important action for Sprint 5 is shipping CI/CD before it blocks the Production → Polish gate. Without CI, the project cannot advance phases.

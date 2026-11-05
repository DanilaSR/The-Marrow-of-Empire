# Sprint 5 — CI/CD Remediation & Feature Layer Kickoff

**Dates**: 2026-08-12 to 2026-08-18 (5 days)
**Milestone**: Production — Feature Layer

## Sprint Goal
Ship the CI/CD pipeline (carried 3 sprints, now BLOCKING the gate check), then begin the Feature layer with Tactical Combat Engine fundamentals — the grid, AP system, and initiative turn order.

## Capacity
- Total: 5 days (solo developer + agents, ~30 hours)
- Buffer (20%): 6 hours
- Available: ~24 hours

## Tasks

### Must Have (Critical Path)

| ID | Task | Agent | Est. Days | Deps | Acceptance Criteria |
|----|------|-------|-----------|------|-------------------|
| CI-001 | CI/CD Workflow Setup | devops-engineer | 0.3 | — | `.github/workflows/test.yml` committed; `go test marrow-of-empire-tests/...` runs on push/PR; first CI run green |
| COMBAT-001 | Square Grid & Movement | gameplay-programmer | 0.5 | EF-001 | 1u=1m grid, diagonal 1 AP, movement range calculation, pathfinding stub |
| COMBAT-002 | AP System & Initiative | gameplay-programmer | 0.4 | COMBAT-001, STAT-002 | 4 + DEX_mod AP (min 2), initiative = DEX_mod + WIS_mod, turn order sorting |

### Should Have

| ID | Task | Agent | Est. Days | Deps | Acceptance Criteria |
|----|------|-------|-----------|------|-------------------|
| COMBAT-003 | d20 Attack Resolution | gameplay-programmer | 0.4 | COMBAT-002 | d20 + accuracy ≥ evasion → hit; nat20 = 2× crit; nat1 = auto-miss |
| COMBAT-004 | Positioning Bonuses | gameplay-programmer | 0.3 | COMBAT-001 | Flanking +2 acc, half cover +2 evasion, full cover blocks LoS, elevation +1 acc/range |

### Nice to Have

| ID | Task | Agent | Est. Days | Deps | Acceptance Criteria |
|----|------|-------|-----------|------|-------------------|
| COMBAT-005 | Death & Dying System | gameplay-programmer | 0.3 | COMBAT-003 | 0 HP → Dying; 3 rounds to stabilize (2 AP, adjacent); permadeath on timeout |

## Carryover from Previous Sprints

| Task | Reason | New Estimate |
|------|--------|-------------|
| CI-001 | Carried 3 sprints — BLOCKING for gate check. Must ship first. | 0.3 days |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Combat engine complexity | Medium | High | Start with grid + AP only; defer AI and full combat loop to Sprint 6 |
| No story files for Feature epics | High | Medium | Run `/create-stories tactical-combat-engine` before COMBAT-001 |
| CI build environment | Low | Medium | GitHub Actions with Go 1.26 — standard setup |

## Dependencies on External Factors
- GitHub repository must exist for CI-001
- Kaiju Engine source readable for combat grid rendering (future stories)

## Definition of Done for this Sprint
- [ ] CI-001 committed and passing in GitHub Actions
- [ ] All Must Have stories completed
- [ ] QA plan exists (`production/qa/qa-plan-sprint-5.md`)
- [ ] All Logic stories have passing unit tests
- [ ] Smoke check passes
- [ ] Code reviewed and committed per-story

> ⚠️ **Prerequisite**: `tactical-combat-engine` has no story files. Run `/create-stories tactical-combat-engine` before starting COMBAT stories. AI Dialogue and Thread Investigation epics are deferred to Sprint 6.

# Sprint 2 — Camera System & Infrastructure Remediation

**Dates**: 2026-08-11 to 2026-08-17 (5 days)
**Milestone**: Pre-Production — Foundation Layer

## Sprint Goal
Fix the test infrastructure gap discovered in Sprint 1 retro, then implement the full isometric camera system — projection, zoom, follow/framing, and occlusion/boundaries. The camera is blocking all visual work and must ship before any scene rendering or art validation can begin.

## Capacity
- Total: 5 days (solo developer, ~30 hours)
- Buffer (20%): 6 hours reserved for engine API verification spikes (all camera stories carry HIGH engine risk)
- Available: ~24 hours

> **Estimation note**: Sprint 1 was overestimated by ~60%. Sprint 2 estimates are calibrated down accordingly. Engine risk is called out per-story rather than distributed.

## Tasks

### Must Have (Critical Path)

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| INFRA-001 | Fix Test Executability (go.work) | lead-programmer | 0.5 | — | `go.work` at repo root; `go test ./...` passes all Sprint 1 tests (EF-001 through EF-006) |
| CAM-001 | Isometric Projection | gameplay-programmer | 3 | INFRA-001 | Orthographic 30°/45°/0°, rotation disabled, aspect ratio handling. ADR-0003 verified against Kaiju `cameras.NewStandardCameraOrthographic` API |
| CAM-002 | Zoom System | gameplay-programmer | 2 | CAM-001 | Three discrete zoom registers (4u/10u/20u), mouse wheel input, 300ms lerp transitions, clamped at min/max |
| CAM-003 | Camera Follow & Framing | gameplay-programmer | 4 | CAM-001, CAM-002 | Party centroid follow (5u/s exploration, 8u/s combat), dialogue frames speaker+player at Close, combat frames all combatants, manual pan (middle mouse), Home snap-back |
| CAM-004 | Occlusion & Boundaries | gameplay-programmer | 4 | CAM-001 | Structure alpha fade on occlusion, roof removal for interiors, boundary enforcement (no void), 21:9 support, interactable highlight through geometry |

### Should Have

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| CI-001 | CI/CD Workflow Setup | devops-engineer | 1.5 | INFRA-001 | `.github/workflows/test.yml` runs `go test ./...` on push/PR; tests pass in CI; workflow file committed |
| SAVE-001 | Save/Load Orchestration | gameplay-programmer | 3 | EF-002, EF-005 (done) | `Save()` serializes all entity component stores + entity table; `Load()` reconstructs all entities with identical state; ADR-0002 verified against `pod` serialization |

## Carryover from Previous Sprint

| Task | Reason | New Estimate |
|------|--------|-------------|
| CAM-001 Isometric Projection | Should-have in Sprint 1, not started — all effort went to must-have EF stories. Now must-have (camera blocks all visual work) | 3 hrs (was 4-6 hrs, calibrated) |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Kaiju camera API mismatch | Medium | High | ADR-0003 carries "Verification Required" — verify `cameras.NewStandardCameraOrthographic` against `kaiju/src/engine/cameras/` before CAM-001 |
| go.work test failures | Low | Medium | Sprint 1 tests may have latent failures that were never caught (tests never executed). Budget 30 min for fixes |
| Occlusion implementation complexity | Medium | Medium | CAM-004 is the riskiest story — structure fade + roof removal + boundary enforcement in one story. Consider splitting if estimate proves low |
| CI/CD GitHub Actions runner availability | Low | Low | Public repo — GitHub-hosted runners are free and reliable |

## Dependencies on External Factors
- Kaiju Engine source in `kaiju/` must be readable for camera API verification (ADR-0003)
- CGO build environment must be functional (Vulkan SDK, Go 1.26)
- GitHub repository must exist for CI/CD workflow (CI-001)

## Definition of Done for this Sprint
- [ ] All 5 Must Have stories completed
- [ ] All acceptance criteria verified
- [ ] `go test ./...` passes from repo root (all Sprint 1 + Sprint 2 tests)
- [ ] Camera renders isometric view at all three zoom levels (visual sign-off per QA plan)
- [ ] QA plan exists (`production/qa/qa-plan-sprint-2.md`)
- [ ] Code reviewed and committed per-story (not squashed)

> ⚠️ **No QA Plan**: This sprint was started without a QA plan. Run `/qa-plan sprint`
> before the last story is implemented. The Production → Polish gate requires a QA
> sign-off report, which requires a QA plan.

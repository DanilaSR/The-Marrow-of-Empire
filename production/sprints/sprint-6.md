# Sprint 6 — In-Flight Closeout & Party System Foundation

**Dates**: 2026-08-13 to 2026-08-19 (5 days)
**Milestone**: Production — Feature Layer

## Sprint Goal
Close out the three in-flight stories (keybindings, dialogue panel, PWS-005), accept the Party ADR to unblock the Party epic, and land the party foundation (roster, state machine, combat roster) alongside the carried FBX import pipeline.

## Capacity
- Total days: 5 (solo developer + agents, ~30 hours)
- Buffer (20%): 6 hours
- Available: ~24 hours

## Tasks

### Must Have (Critical Path)

| ID | Task | Agent/Owner | Est. Days | Dependencies | Acceptance Criteria |
|----|------|-------------|-----------|-------------|-------------------|
| MMS-007 | Keybindings Remapping — finish (in-flight, tests failing) | ui-programmer | 0.5 | — | All 24 integration tests pass (click routing + persistence round-trip); `/code-review`; `/story-done` Complete |
| DIALOGUE-UI-001 | Dialogue Panel Framework — finish (in-progress since sprint-5; panel exists from vertical slice INT-2) | ui-programmer | 0.4 | — | Reconcile slice panel with story ACs (markup recipe, static label pool); evidence; `/story-done` Complete |
| PWS-005 | World Evolution Rules | gameplay-programmer | 0.5 | PWS-002 (done) | YAML condition→event rules eval once per region load; <1ms/region (20 rules); circular-dependency warning; unit tests (ADR-0013) |
| PARTY-001 | Party Roster & Active Slots | gameplay-programmer | 0.3 | — | Roster model per TR-party-001; add/remove/active slots; unit tests. Unlocks 002/003/007 |
| PRC-001 | Accept ADR-0016 + amend ADR-0008/0009 (enabler task, no story file) | technical-director | 0.4 | — | ADR-0016 Proposed→Accepted; ADR-0008 interjection interface + ADR-0009 knockout amendments logged; flip party stories 002/006/008/009 Blocked→Ready |

### Should Have

| ID | Task | Agent/Owner | Est. Days | Dependencies | Acceptance Criteria |
|----|------|-------------|-----------|-------------|-------------------|
| ASSET-001 | FBX Import & Normalization (carryover from sprint-5) | technical-artist + kaijuengine-game-dev skill | 1.0 | — | Single KitBash3D FBX imports via Kaiju pipeline (ADR-0004); normalization rules validated |
| PARTY-002 | Companion State Machine | gameplay-programmer | 0.5 | PARTY-001, PRC-001 | Companion states per TR-party-002; transition rules; unit tests |
| PARTY-007 | Combat Roster Integration | gameplay-programmer | 0.3 | PARTY-001, combat epic (complete) | Active roster feeds combat initiative; party members controllable; integration tests |
| VS-REPORT | Vertical slice playtest + report (human task) | user (human) | 0.5 | INT-4 (done) | Playtest Session 1 run; REPORT.md + prototypes/index.md verdict (vertical-slice Phases 6–8) |

### Nice to Have

| ID | Task | Agent/Owner | Est. Days | Dependencies | Acceptance Criteria |
|----|------|-------------|-----------|-------------|-------------------|
| PWS-006 | Save/Load Performance | gameplay-programmer | 0.5 | PWS-003 (done) | Completes PWS epic 6/6; benchmark within ADR-0013 budget |
| AUDIO-002 | Ambient Soundscapes | sound-designer + audio-programmer | 0.4 | AUDIO-001 (done), ADR-0014 (accepted) | Region-based ambient loops; first audible audio in game |
| COMBAT-006 | Combat Resolution Integration | gameplay-programmer | 0.5 | COMBAT-003 (done) | Wire attack resolution into game loop (check INT-3 slice overlap at `/story-readiness`) |
| PARTY-003 | Recruitment via Trust & Dialogue | gameplay-programmer | 0.6 | PARTY-002 | Recruitment unlocks at trust threshold; stretch goal |

## Carryover from Previous Sprint

| Task | Reason | New Estimate |
|------|--------|-------------|
| ASSET-001..004 | Sprint-5 Should Have, never started | ASSET-001 into sprint-6 (1.0d); 002–004 return to backlog |
| DIALOGUE-UI-001 | Sprint-5 in-progress; panel built during vertical slice INT-2 | 0.4d to reconcile + close |
| MMS-007 | Mid-flight outside sprint tracking; 20+ integration tests failing | 0.5d to fix + close |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Keybindings click-routing regression deeper than expected | Medium | Medium | Isolate click handling vs binding registry; bisect from last green state |
| ADR-0008/0009 amendments touch shipped systems (interjections, knockdown) | Medium | High | Keep amendments additive — new interface, not rewrite |
| Kaiju FBX import unknowns (custom engine, no LLM training data) | High | Medium | Spike single-FBX import first; lean on kaijuengine-game-dev skill + ADR-0004 |
| ADR-0016 acceptance stalls → PARTY-002 blocked mid-sprint | Low | High | PARTY-007 is ADR-independent; 007 is the fallback should-have |
| Vertical-slice playtest findings demand rework | Medium | Medium | 20% buffer reserved for slice fallout |

## Dependencies on External Factors
- KitBash3D Dark Fantasy pack files accessible for ASSET-001
- Human playtest Session 1 for vertical-slice Phases 6–8
- ADR-0016 acceptance requires a `/architecture-decision` review pass

## Definition of Done for this Sprint
- [ ] All Must Have tasks completed
- [ ] All tasks pass acceptance criteria
- [ ] QA plan exists (`production/qa/qa-plan-sprint-6.md`)
- [ ] All Logic/Integration stories have passing unit/integration tests
- [ ] Smoke check passed (`/smoke-check sprint`)
- [ ] QA sign-off report: APPROVED or APPROVED WITH CONDITIONS (`/team-qa sprint`)
- [ ] No S1 or S2 bugs in delivered features
- [ ] Design documents updated for any deviations
- [ ] Code reviewed and merged

## Integration Pass (added 2026-08-13, user-approved)

Goal: wire the built headless systems into the playable standalone build, live-verified via ai_driver. Displaces the sprint's Should/Nice capacity. Runs in order after PARTY-001 closes:

| Unit | Story | Est |
|------|-------|-----|
| INT-5 | main-menu-settings/story-008 — title flow, settings.cfg adapter, keybindings live | 1.0d |
| INT-6 | save-load-system/story-004 — save/load orchestrator live | 0.5d |
| INT-7 | party-system/story-015 — party store, rest points, combat feed | 0.6d |
| INT-8 | persistent-world-state/story-007 — flags, events, region rules live | 0.5d |
| INT-9 | COMBAT-006 (existing) — full combat loop end-to-end | 0.5d |
| INT-10 | AUDIO-002 (existing) — ambience + clock wiring | 0.4d |

Deferred out of the pass: pause-menu freeze contract wiring (Story 006 ACs — next pass), Steam Cloud live sync (platform task).

## Notes
- QA Plan: `production/qa/qa-plan-sprint-6-2026-08-13.md` — written and back-filled to story files before implementation.
- Review mode: `lean` — director gates (PR-SPRINT etc.) skipped this sprint.

> **Scope check:** If this sprint includes stories added beyond the original epic scope, run `/scope-check [epic]` to detect scope creep before implementation begins.

# Project Stage Analysis Report

**Generated**: 2026-11-05
**Stage**: Production
**Stage Confidence**: CONCERNS
**Analysis Scope**: Full project (general)

---

## Executive Summary

The Marrow of Empire is unambiguously in Production. It has 159 Go source files
(~48.6k LOC) spanning 21 of 23 designed systems, 117 story files across 22 epics,
16 ADRs, six sprint plans, and a vertical slice with three of four integration
milestones complete (INT-1 world bootstrap, INT-2 talk loop, INT-3 combat).

The codebase is substantially ahead of its tracking artifacts. Sprint 6 ended on
2026-08-19 but was last updated 2026-08-13 and is now ~11 weeks stale. There have
been **zero commits since 2026-08-12** while **570 files / +95,131 lines** sit
uncommitted in the working tree — including all source, tests, the VO design pack,
and production records. The last Production → Polish gate check (2026-08-12)
returned **FAIL**, and its prerequisites (playtests, QA sign-off, smoke checks,
green test suite) have not been re-established.

**Current Focus**: Voice-over recording and assembly for the investigation chain
(S01+S02, 48 lines across Vek/Tomas/Mira), plus the carried vertical-slice work.
This work is complete on disk but uncommitted.

**Blocking Issues**: Uncommitted 95k-line changeset; stale/overdue sprint tracking;
one failing performance test; no playtest sessions; Production → Polish gate FAIL.

**Estimated Time to Next Stage**: Not estimable until the gate blockers close.
Realistically 2–3 closeout sprints (commit, green tests, smoke, playtest, QA sign-off).

---

## Completeness Overview

### Design Documentation — ~52% mature (23/23 authored)

- **Status**: 100% coverage (every system has a GDD), ~52% in an approved/stable state.
- **Files Found**: 26 Markdown files in `design/gdd/` — 23 system GDDs plus
  `game-concept.md`, `systems-index.md`, and a cross-review log.
- **GDD status distribution** (from `systems-index.md`):
  - ✅ Approved (2): Entity Framework, Camera System
  - ✅ Designed (9): Asset & Scene Pipeline, NPC Knowledge Graph, Persistent World
    State, Question System, Trust & Relationship, Thread Investigation,
    Character & Inventory UI, Loot & Economy, Main Menu & Settings
  - ⚠️ In Review (1): Party System
  - ⚠️ Needs Revision (11): Save/Load, Character Stats & Progression, Living
    Environment, AI Dialogue Engine, Tactical Combat Engine, Faction System,
    Investigation Journal, Dialogue UI, Combat UI, Map System, Audio System
- **Section completeness** (heuristic heading scan): 18/23 GDDs contain all eight
  required sections. Five are partial: Audio (2), Map (2), Combat UI (3),
  Dialogue UI (3), Investigation Journal (4). Some may use non-standard headings —
  verification needed before treating these as true gaps.
- **Other design docs**: 8 UX docs in `design/ux/`, 5 narrative docs (the VO pack
  and recording checklist), accessibility requirements.
- **Key Gaps**:
  - [ ] `design/game-pillars.md` missing — no single source of truth for design pillars
  - [ ] `design/difficulty-curve.md` missing (also flagged by the 2026-08-12 gate check)
  - [ ] `design/levels/` is empty — no region/encounter/level design docs
  - [ ] 11 GDDs marked Needs Revision, several for systems already implemented

### Source Code — ~90% system coverage

- **Status**: 21/23 systems have implementation code.
- **Files Found**: 159 `.go` files in `src/`, ~48,623 lines.
- **Major Systems Identified**:
  - ✅ Presentation/UI (`src/ui/`, 29 files) — title screen, combat HUD, maps
  - ✅ AI Dialogue (`src/ai_dialogue/`, 18) — state machine, tiered routing, companion
  - ✅ Tactical Combat (`src/combat/`, 13) — combat engine and resolution
  - ✅ Character Sheet (`src/character_sheet/`, 12) and Stats (`src/stats/`, 5)
  - ✅ Game shell (`src/game/`, 11) — world, talk, menu, settings, journal wiring
  - ✅ Investigation (`src/investigation_journal/` 9, `src/thread_investigation/` 5)
  - ✅ Entity (`src/entity/`, 8), Camera (`src/camera/`, 8), Audio (`src/audio/`, 6)
  - ✅ NPC Knowledge Graph (5), Party (5), Save (4), Persistent World State (4),
    Dialogue UI (4), Asset (4), Trust (3), Question System (3), Environment (3)
  - ⚠️ Party System — roster foundation only; 9 of 14 stories blocked on ADR-0016
  - ⚠️ Audio — system exists; ambient soundscapes (AUDIO-002) not implemented
- **Key Gaps**:
  - [ ] **Loot & Economy** — MVP-priority system, ADR-0015 Accepted, no code and
    no epic or stories. Highest-priority missing implementation.
  - [ ] **Faction System** (Alpha priority) — no `src/` package, epic exists but
    0 story files.
  - [ ] 12 `TODO` markers in `src/` (0 FIXMEs) — untriaged.

### Architecture Documentation — 84.4% TR coverage

- **Status**: Strong foundation, with stale registry data.
- **ADRs Found**: 16 in `docs/architecture/`
  - ✅ Accepted (15): ADR-0001 through ADR-0015
  - ⚠️ Proposed (1): **ADR-0016 Party System Architecture** — while PARTY-001 is
    Complete and party stories are marked Ready
- **Coverage** (latest review, `architecture-review-2026-08-12b.md`):
  - 84.4% of TRs covered by Accepted ADRs (222/263), 40 gaps
  - Reviews on 2026-08-09, 2026-08-12, and 2026-08-12b all returned **CONCERNS**
  - Known stale markers: TR-save-009 conflict resolved by ADR-0002 amendment but
    registry still flags it; ADR-0008/0014/0015 coverage not reflected in registry
- **Key Gaps**:
  - [ ] TR registry needs regeneration after ADR-0016 acceptance and recent amendments
  - [ ] ADR-0016 acceptance (or documented reason to keep Proposed)
  - [ ] 40 remaining TR gaps, concentrated in AI Dialogue, Audio, Loot & Economy,
    Party, Faction, Map, Persistent World State

### Production Management — tracking present but stale

- **Status**: Sprint cadence exists; control artifacts lag reality.
- **Found**:
  - Sprint plans: 6 (`production/sprints/sprint-1.md` … `sprint-6.md`)
  - **Sprint 6: 2026-08-13 → 2026-08-19; status file last updated 2026-08-13**
    (~11 weeks stale as of this report)
  - Sprint-6 YAML: 15 stories — 2 done, 2 in-progress, 1 ready-for-dev,
    8 backlog, 2 blocked
  - Story files: 117 — **70 Complete, 37 Ready, 9 Blocked, 1 In Progress**
  - Milestones: 0 (`production/milestones/` empty)
  - Roadmap: missing
  - Retrospectives: sprints 1, 2, 4 (3, 5, 6 missing)
  - QA: plans for sprints 1–6; sign-off only sprints 1–2; smoke checks only
    2026-08-10 and 2026-08-11; 15 evidence files
  - Playtests: plan + template only — **0 executed sessions**
  - Gate check 2026-08-12: **FAIL** (5/11 required artifacts, 2/10 quality checks)
- **Git state**:
  - Last commit: 2026-08-12 (`feat: Investigation Journal`)
  - 570 uncommitted files: 541 added, 24 modified, 4 add/modify, 1 untracked
  - Diff: **+95,131 / −158 lines**
  - Breakdown: `game_content` 145, `production` 143, `src` 105, `tests` 49,
    `design` 43, `.claude` 32, `docs` 25, `prototypes` 10, `assets` 10, `models` 3
- **Key Gaps**:
  - [ ] Uncommitted changeset is the largest single project risk
  - [ ] Sprint 6 not closed; no sprint 7 plan; recent VO work untracked
  - [ ] No milestone definitions or roadmap
  - [ ] QA sign-off and smoke checks 4 sprints behind
  - [ ] Zero playtest sessions (gate requires ≥3)

### Testing — 97% packages green

- **Status**: 36 of 37 tested packages pass.
- **Test Files**: 91 (56 unit, 35 integration), ~1,284 test functions.
- **Command run**: `go test ./tests/unit/... ./tests/integration/...` on 2026-11-05
- **Result**: 36 packages `ok`, 1 failure:
  - ❌ `tests/unit/entity` — `TestPerformanceTargets/AC-2_SpatialQuery`:
    p95 **9.415 ms** vs 5 ms "load-immune" budget. Re-run in isolation before
    classifying as a regression; the suite ran under concurrent load.
- **Notes**:
  - No coverage report exists; per-system coverage cannot be stated.
  - Previous gate-check failures in `map_system` and `main_menu/save_load` now pass.
- **Key Gaps**:
  - [ ] Resolve or re-baseline the spatial-query performance budget
  - [ ] Produce a coverage report to drive test-gap decisions

### Prototypes

- **Active Prototypes**: 2 in `prototypes/`
  - ✅ **Concept prototype** (`marrow-of-empire-concept/`) — `REPORT.md` (PROCEED),
    `play-log.md`, `rules.md`. Documented. The session hook reports this as
    undocumented because it looks for README/CONCEPT — **false positive**.
  - ✅ **Vertical slice** (`marrow-of-empire-vertical-slice/`) — README, PLAN,
    INT-001…INT-004 records. **INT-1, INT-2, INT-3 Complete; INT-4 Planned.**
- **Key Gaps**:
  - [ ] INT-4 (journal/revelation + panel fix) not started
  - [ ] VS-REPORT / playtest session still pending from sprint 6
  - [ ] Optionally add a README in the concept folder or adjust the hook

---

## Stage Classification Rationale

**Why Production?**

`production/stage.txt` explicitly sets Production, and the auto-detection heuristics
agree: 159 source files across 21 systems, 117 story files, six sprints of history,
16 ADRs, and a vertical slice in the production tree.

**Indicators for this stage**:
- `src/` well beyond the 10-file Pre-Production threshold (159 files, ~48.6k LOC)
- Epic/story structure exists for 22 systems; 70 stories Complete
- Vertical slice INT-1–INT-3 shipped in production code
- Active sprint cadence (though currently stale)
- Gate check for Production → Polish attempted (verdict FAIL, not blocking classification)

**Next stage requirements (Production → Polish)**:
- [ ] Test suite fully green (currently 1 failing package)
- [ ] ≥3 playtest sessions with reports (currently 0)
- [ ] Smoke check PASS and QA sign-off for the current sprint
- [ ] Production → Polish gate re-run and passed
- [ ] All MVP systems implemented (Loot & Economy missing)
- [ ] No stale tracking artifacts contradicting the code state

---

## Gaps Identified (with Clarifying Questions)

### Critical Gaps (block progress)

1. **570 uncommitted files / +95,131 lines / zero commits in ~12 weeks**
   - **Impact**: A single bad checkout, disk failure, or tooling error could erase
     months of source, tests, VO design, and production history.
   - **Question**: Should the changeset be split into logical commits now, or is
     something deliberately holding it back?
   - **Suggested Action**: Review the staged set and commit in logical batches
     (game_content/engine, src, tests, design/VO, production, .claude).

2. **Sprint 6 stale and overdue**
   - **Impact**: Status reporting is unusable; recent VO work is invisible to
     planning; 37 stories sit Ready with no active sprint.
   - **Question**: Close sprint 6 and plan sprint 7, or is sprint tracking
     intentionally paused during audio production?
   - **Suggested Action**: Reconcile actual statuses, write the close-out, then
     run `/sprint-plan`.

3. **Production → Polish gate FAIL (2026-08-12), never re-evaluated**
   - **Impact**: The project cannot credibly advance to Polish; some gate findings
     may be stale in both directions.
   - **Question**: Re-run `/gate-check` after the suite is green and the changeset
     committed?
   - **Suggested Action**: Run `/smoke-check`, then `/gate-check` for a current verdict.

4. **Failing performance test** (`unit/entity`, AC-2 SpatialQuery p95 9.4 ms)
   - **Impact**: Test suite is not green; CI gate cannot pass.
   - **Question**: Is this a real regression, or a load-sensitive benchmark?
   - **Suggested Action**: Re-run `go test ./tests/unit/entity -run TestPerformanceTargets`
     in isolation; if it persists, profile and fix or re-baseline with justification.

### Important Gaps (affect quality/velocity)

5. **11 GDDs need revision while their systems are implemented**
   - **Impact**: Design docs no longer describe shipped behavior; onboarding and
     future balancing will rely on code archaeology.
   - **Question**: Run a reconciliation/reverse-document pass now, or defer until
     the vertical slice closes?
   - **Suggested Action**: Prioritize implemented systems; use `/reverse-document`
     or targeted GDD revisions, then update `systems-index.md`.

6. **ADR-0016 Proposed while party stories are Ready and PARTY-001 is Complete**
   - **Impact**: Violates the project rule that stories referencing Proposed ADRs
     are blocked; 9 party stories remain Blocked.
   - **Question**: Accept ADR-0016 now, or are ADR-0008/0009 amendments still required first?
   - **Suggested Action**: `/architecture-decision` update or direct acceptance,
     then unblock party stories.

7. **Loot & Economy (MVP) has no code, epic, or stories; Faction has no stories**
   - **Impact**: One MVP system is entirely unbuilt; Alpha faction work is unmapped.
   - **Question**: Are these intentionally deferred, or should epics/stories be created?
   - **Suggested Action**: `/create-epics` + `/create-stories` for loot-economy;
     decide whether faction stays Alpha or is de-scoped.

8. **Zero playtests / no VS-REPORT**
   - **Impact**: The fun hypothesis and gate requirements remain unvalidated.
   - **Question**: Is the INT-4 playtest still planned, or is the slice paused?
   - **Suggested Action**: Complete INT-4, then run and document the playtest session.

### Nice-to-Have Gaps

9. **Missing design baselines** — `game-pillars.md`, `difficulty-curve.md`, level designs.
10. **No milestones or roadmap** — production lacks long-horizon planning artifacts.
11. **Prototype hook false positive** — concept folder has REPORT.md but no README.
12. **12 TODOs in `src/`** and **3 missing retrospectives** (sprints 3, 5, 6).

---

## Recommended Next Steps

### Immediate Priority (Do First)

1. **Commit the uncommitted changeset in logical batches** — reduces the single
   largest project risk and restores a reviewable history.
   - Suggested: manual git work; no skill required
   - Estimated effort: M (review + split + commit)

2. **Reconcile sprint 6 and plan sprint 7** — restore usable tracking.
   - Suggested skill: `/sprint-status`, then `/sprint-plan`
   - Estimated effort: S

3. **Get the test suite fully green** — triage the entity performance failure.
   - Suggested: re-run in isolation; `/diagnosing-bugs` if reproducible
   - Estimated effort: S–M

4. **Run smoke check and re-run the gate check** — replace the stale FAIL verdict.
   - Suggested skills: `/smoke-check`, then `/gate-check`
   - Estimated effort: S

### Short-Term (This Sprint/Week)

5. **Accept ADR-0016 and refresh the TR registry** — unblocks 9 party stories and
   clears stale conflict markers.
   - Suggested: `/architecture-decision` or ADR edit + registry update
   - Estimated effort: S

6. **Reconcile the 11 Needs Revision GDDs** — prioritize implemented systems
   (AI Dialogue, Combat, Investigation Journal, Dialogue/Combat UI, Stats).
   - Suggested: `/reverse-document design src/[system]` or `/design-review` per doc
   - Estimated effort: L

7. **Complete INT-4 and run the vertical-slice playtest (VS-REPORT)**
   - Suggested: manual playtest + `/playtest-report`
   - Estimated effort: M

### Medium-Term (Next Milestone)

8. **Decide Loot & Economy scope; create its epic and stories** (or explicitly de-scope).
   - Suggested skills: `/create-epics`, `/create-stories`
9. **Create milestone definitions and a roadmap** — the current sprint-only horizon
   makes the gate trajectory invisible.
   - Suggested skill: `/sprint-plan` supporting docs, `/milestone-review`
10. **Fill design baselines**: `game-pillars.md`, `difficulty-curve.md`, and the
    first region/level design.

---

## Follow-Up Skills to Run

- `/sprint-status` — reconcile sprint 6
- `/sprint-plan` — plan sprint 7
- `/smoke-check` — verify current build health
- `/gate-check` — replace the 2026-08-12 FAIL verdict
- `/reverse-document design src/[system]` — for GDDs marked Needs Revision
- `/architecture-decision` — accept ADR-0016 / document its hold
- `/create-epics` + `/create-stories` — Loot & Economy, Faction
- `/playtest-report` — for the pending vertical-slice session
- `/tech-debt` — triage the 12 TODOs

---

## Appendix: File Counts by Directory

```text
design/
  gdd/           26 files (23 system GDDs + concept + index + cross-review)
  narrative/      5 files (VO pack + recording docs)
  levels/         0 files
  ux/             8 files
  art/            (present)

src/            159 .go files, ~48,623 lines
  ui/            29
  ai_dialogue/   18
  combat/        13
  character_sheet/ 12
  game/          11
  investigation_journal/ 9
  entity/         8
  camera/         8
  audio/          6
  (16 more packages with 1–5 files each)

tests/           91 test files (56 unit, 35 integration), ~1,284 test functions
docs/architecture/ 16 ADRs + overview + 3 reviews + control manifest + TR registry
production/
  sprints/        6 plans
  epics/          22 epic dirs, 117 story files
  qa/             6 QA plans, 2 sign-offs, 2 smoke checks, 15 evidence files
  playtests/      1 plan + 1 template (0 sessions)
  milestones/     0 files
  retrospectives/ 3 files
  gate-checks/    1 file (Production → Polish, FAIL)
prototypes/       2 prototypes (concept + vertical slice, both documented)
```

---

*Report generated by `/project-stage-detect`. No files outside this report were modified.*

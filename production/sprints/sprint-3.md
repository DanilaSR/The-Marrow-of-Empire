# Sprint 3 — Core Gameplay Foundation

**Dates**: 2026-08-11 to 2026-08-17 (5 days)
**Milestone**: Production — Core Layer

## Sprint Goal
Establish the Core gameplay foundation: character stats (the math behind combat, dialogue checks, and progression), save/load orchestration (completing the persistence loop from EF-005), and CI/CD (closing infrastructure debt from Sprint 2).

## Capacity
- Total: 5 days (solo developer, ~30 hours)
- Buffer (20%): 6 hours reserved for formula verification and engine API spikes
- Available: ~24 hours

> **Estimation note**: Sprint 1 was overestimated by ~60%, Sprint 2 by ~40%. Sprint 3 estimates are calibrated using actual velocity: ~6 stories/day with subagent parallelism. Stats stories carry lower engine risk (pure Go math — no Kaiju APIs).

## Tasks

### Must Have (Critical Path)

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| STAT-001 | Attribute Model | gameplay-programmer | 2 | EF-001 | Six attributes STR/DEX/CON/INT/WIS/CHA, range 3-18, modifier = floor((A−10)/2), point-buy creation (30 pts, start 8, max 16) |
| STAT-002 | Derived Stats | gameplay-programmer | 2 | STAT-001 | MaxHealth = 20 + CON×2 + Level×CON_mod, AP = 4 + DEX_mod (min 2), Initiative = DEX_mod + WIS_mod, Evasion = 10 + DEX_mod |
| STAT-003 | XP & Leveling | gameplay-programmer | 2 | STAT-002 | XP_to_next = Level × 300, Total_XP = 150 × N × (N−1), level cap 20, stat gains on level-up |
| SAVE-001 | Save/Load Orchestration | gameplay-programmer | 3 | EF-002, EF-005 | Save() serializes all entity component stores + entity table via EF-005; Load() reconstructs all entities with identical state; slot metadata (character name, playtime, region, thumbnail, timestamp) |
| SAVE-002 | Save File I/O | gameplay-programmer | 2 | SAVE-001 | Version header, CRC32 checksum, .bak backup, platform-standard save paths (Linux $XDG_DATA_HOME, Windows %APPDATA%, macOS ~/Library/Application Support) |
| CI-001 | CI/CD Workflow Setup | devops-engineer | 1.5 | INFRA-001 | `.github/workflows/test.yml` runs `go test ./...` on push/PR; tests pass in CI; workflow file committed |

### Should Have

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| KG-001 | Knowledge Graph Data Model | ai-programmer | 3 | EF-001, EF-002 | YAML .kg file format parseable, fact types FACT/RUMOR/OPINION/SECRET stored in Knowledge component, trust gating (trust_required 0-3 filtered by current trust_level) |
| ENV-001 | Game Clock | gameplay-programmer | 2 | EF-001 | 24-hour clock, 10:1 time ratio (2.4 real hours = 1 game day), starts 8:00 AM Day, five light states (Dawn/Day/Dusk/Night/Deep Night), combat pauses clock |
| STAT-004 | Equipment Stat Modification | gameplay-programmer | 1.5 | STAT-001 | Items modify attributes via equipment slots; stat bonuses applied/removed on equip/unequip; derived stats recalculated |

### Nice to Have

| ID | Task | Agent/Owner | Est. Hrs | Dependencies | Acceptance Criteria |
|----|------|-------------|----------|--------------|-------------------|
| ASSET-001 | FBX → Kaiju Mesh Import | technical-artist | 4 | — | KitBash3D Dark Fantasy FBX imports via staging pipeline, scale normalised (1u = 1m), rotation corrected (Z-up → Y-up), reference object validation |

## Carryover from Previous Sprint

| Task | Reason | New Estimate |
|------|--------|-------------|
| SAVE-001 Save/Load Orchestration | Should-have in Sprint 2, not started — camera push consumed all capacity | 3 hrs (unchanged) |
| CI-001 CI/CD Workflow | Should-have in Sprint 2, not started | 1.5 hrs (unchanged) |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| STAT formulas conflict with GDD | Low | Medium | ADR-0006 governs all formulas — verify against `design/gdd/character-stats-progression.md` before each story |
| Save file format decisions | Medium | Low | EF-005 already handles pod serialization; SAVE stories are orchestration + file I/O only — no format invention |
| CI not running tests | Low | Medium | INFRA-001 fixed go.work — CI just needs to invoke `go test ./...` from the correct directories |
| .kg YAML format design iteration | Medium | Low | ADR-0005 defines the format; YAML parsing is fast (~10ms per NPC); iteration is on content, not parser |
| KitBash3D asset import complexity | High | Medium | First asset pipeline story — engine API verification required. Marked Nice to Have for risk management |

## Dependencies on External Factors
- Kaiju Engine source in `kaiju/` must be readable for save/load pod API verification (ADR-0002)
- CGO build environment must be functional (Vulkan SDK, Go 1.26)
- GitHub repository must exist for CI/CD workflow
- KitBash3D Dark Fantasy assets at `/home/danila/Downloads/KitBash3D - Dark Fantasy/` (ASSET-001)
- NPC-LLM-3_8B-Q5_K_M.gguf at project root for LLM integration (future AI stories)

## Definition of Done for this Sprint
- [ ] All 6 Must Have stories completed
- [ ] All acceptance criteria verified
- [ ] `go test ./...` passes from repo root (all Sprint 1 + Sprint 2 + Sprint 3 tests)
- [ ] Character stats module: all formulas match GDD, verified by unit tests
- [ ] Save/Load round-trip: 100 entities saved and loaded with identical state
- [ ] CI/CD: GitHub Actions workflow runs and passes on push
- [ ] QA plan exists (`production/qa/qa-plan-sprint-3.md`)
- [ ] Code reviewed and committed per-story

> ⚠️ **No QA Plan**: Run `/qa-plan sprint` before the last story is implemented.
> The Production → Polish gate requires a QA sign-off report, which requires a QA plan.

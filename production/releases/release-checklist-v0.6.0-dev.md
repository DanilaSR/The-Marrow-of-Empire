## Release Checklist: v0.6.0-dev — All Platforms
Generated: 2026-08-12

### Codebase Health
- **TODO count**: 12 (6 localization stubs, 5 CGo/lambda.cpp stubs, 1 game-layer wire, 1 CSS outline-offset)
- **FIXME count**: 0 — no known critical bugs marked
- **HACK count**: 0 — no technical shortcuts requiring review

**Top TODOs** (localization & CGo):
| File | Line | Note |
|------|------|------|
| `src/ui/combat_hud/render.go` | 386, 391 | User-facing combat strings need localization routing |
| `src/ui/combat_hud/action_bar.go` | 38 | Action bar labels need localization |
| `src/ui/combat_hud/combat_hud.go` | 70 | Combat HUD strings need localization |
| `src/ai_dialogue/tiny_cgo_backend.go` | 46, 64, 74, 125, 148 | llama.cpp CGo bindings still stubbed |
| `src/ui/title_screen/title_screen.go` | 587 | Game-layer wire — navigate to difficulty selection |
| `src/ui/title_screen/title_screen_focus.go` | 222 | CSS outline-offset rendering note |
| `src/ui/map/world_map.go` | 1628 | Localization system routing note |

### Build Verification
- [ ] Clean build succeeds on all target platforms — **⚠️ Currently FAILING**: `src/ui/map/world_map.go:411` — `SetInputEnabled` method missing on `LocalMap` (active work in Sprint 5)
- [ ] No compiler warnings (zero-warning policy) — go vet passes on all non-map packages
- [ ] All assets included and loading correctly
- [ ] Build size within budget ([TBD])
- [ ] Build version number correctly set ([TBD])
- [ ] Build is reproducible from tagged commit

### Quality Gates
- [ ] Zero S1 (Critical) bugs — **No open bugs in production/bugs/**
- [ ] Zero S2 (Major) bugs — or documented exceptions with producer approval
- [ ] All critical path features tested and signed off by QA
- [ ] Performance within budgets:
  - [ ] Target FPS met on minimum spec hardware (budget: 60 fps, 16.6ms frame)
  - [ ] Memory usage within budget ([TBD] — must account for 2.7GB LLM model + game memory)
  - [ ] Load times within budget
  - [ ] No memory leaks over extended play sessions
- [ ] No regression from previous build — **⚠️ map_system package (unit + integration) has build failures**
- [ ] Soak test passed (4+ hours continuous play)

### Test Suite Status
- **72 test files**, all pass except map_system (build failure — known work-in-progress)
- **Test packages**: 28 passing, 2 failing (map_system unit + integration, same root cause)
- **Completed epics**: Entity Framework (6/6), Character Stats (5/5), AI Dialogue Engine (5/5), Thread Investigation (5/5), Investigation Journal (5/5)
- **In-progress epics**: Audio System (1/5), Combat Engine (2/5+), NPC Knowledge Graph (1/5+), Question System (1/3), Trust & Relationship (1/3+), Main Menu & Settings (3/5+), Character & Inventory UI (5/9), Map System (2/3), Dialogue UI (1/3+), Save/Load System (1/5+)
- **Active sprint**: Sprint 5 — CI/CD Remediation & Feature Layer Kickoff (2026-08-12 to 2026-08-18)

### Content Complete
- [ ] All placeholder assets replaced with final versions
- [ ] All TODO/FIXME in content files resolved or documented
- [ ] All player-facing text proofread
- [ ] All text localization-ready — **⚠️ 6 localization TODOs remain in combat_hud package**
- [ ] Audio mix finalized and approved
- [ ] Credits complete and accurate

### Platform Requirements: PC
- [ ] Minimum and recommended specs verified and documented
- [ ] Keyboard+mouse controls fully functional
- [ ] Controller support tested (Xbox, PlayStation, generic)
- [ ] Resolution scaling tested (1080p, 1440p, 4K, ultrawide) — ref 1920×1080, min 1280×720
- [ ] Windowed, borderless, and fullscreen modes working
- [ ] Graphics settings save and load correctly
- [ ] Steam/Epic/GOG SDK integrated and tested
- [ ] Achievements functional
- [ ] Cloud saves functional
- [ ] Steam Deck compatibility verified (if targeting)

### Platform Requirements: Console
- [ ] TRC/TCR/Lotcheck requirements checklist complete
- [ ] Platform-specific controller prompts display correctly
- [ ] Suspend/resume works correctly
- [ ] User switching handled properly
- [ ] Network connectivity loss handled gracefully
- [ ] Storage full scenario handled
- [ ] Parental controls respected
- [ ] Platform-specific achievement/trophy integration tested
- [ ] First-party certification submission prepared

### Platform Requirements: Mobile
- [ ] App store guidelines compliance verified
- [ ] All required device permissions justified and documented
- [ ] Privacy policy linked and accurate
- [ ] Data safety/nutrition labels completed
- [ ] Touch controls tested on multiple screen sizes
- [ ] Battery usage within acceptable range
- [ ] Background behavior correct (pause, resume, terminate)
- [ ] Push notification permissions handled correctly
- [ ] In-app purchase flow tested (if applicable)
- [ ] App size within store limits

### Store / Distribution
- [ ] Store page metadata complete and proofread
  - [ ] Short description
  - [ ] Long description
  - [ ] Feature list
  - [ ] System requirements (PC)
- [ ] Screenshots up to date and per-platform resolution requirements met
- [ ] Trailers up to date
- [ ] Key art and capsule images current
- [ ] Age rating obtained and configured:
  - [ ] ESRB
  - [ ] PEGI
  - [ ] Other regional ratings as required
- [ ] Legal notices, EULA, and privacy policy in place
- [ ] Third-party license attributions complete
- [ ] Pricing configured for all regions

### Launch Readiness
- [ ] Analytics / telemetry verified and receiving data
- [ ] Crash reporting configured and dashboard accessible
- [ ] Day-one patch prepared and tested (if needed)
- [ ] On-call team schedule set for first 72 hours
- [ ] Community launch announcements drafted
- [ ] Press/influencer keys prepared for distribution
- [ ] Support team briefed on known issues and FAQ
- [ ] Rollback plan documented (if critical issues found post-launch)

### Go / No-Go: **NOT READY**

**Rationale:**
The project is in early Production (Feature Layer, Sprint 5). The CI/CD pipeline — a blocker for
gate check — has been carried across 3 sprints and is still not shipped. A `go build` failure exists
in `src/ui/map/world_map.go` (active work-in-progress). Approximately 60% of epic stories remain
unimplemented across 8 active epics. No release infrastructure (store pages, telemetry, crash
reporting) exists. This project is months from a release candidate. The checklist is generated as a
forward-looking template — revisit at the Production→Polish transition.

**Sign-offs Required:**
- [ ] QA Lead
- [ ] Technical Director
- [ ] Producer
- [ ] Creative Director

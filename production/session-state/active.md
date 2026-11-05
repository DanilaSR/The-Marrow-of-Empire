# Session State


## Session Extract — /sprint-plan new (Sprint 6) 2026-08-13
- Verdict: COMPLETE — sprint-6.md + sprint-status.yaml written (user approved both; review mode lean)
- Sprint 6: 2026-08-13 → 08-19. Goal: close in-flight stories + Party System foundation + carried FBX pipeline
- Must: MMS-007 keybindings finish (tests confirmed failing — click routing), DIALOGUE-UI-001 finish (panel exists from INT-2), PWS-005 world evolution rules, PARTY-001 roster, PRC-001 (accept ADR-0016 + amend ADR-0008/0009 → unblocks party 002/006/008/009)
- Should: ASSET-001 FBX (carryover), PARTY-002 state machine (blocked on ADR), PARTY-007 combat roster, VS-REPORT (human playtest)
- Nice: PWS-006, AUDIO-002, COMBAT-006, PARTY-003
- Carried statuses preserved in yaml: MMS-007 in-progress, DIALOGUE-UI-001 in-progress, PARTY-002/003 blocked
- QA plan: NONE for sprint-6 — user chose to run /qa-plan sprint before implementation
- Next: /qa-plan sprint → then MMS-007 finish (fix click routing) or PARTY-001 via /dev-story
- Facts: no production/milestones/ dir; no risk-register dir; backlog = 68 Complete / 36 Ready / 9 Blocked stories
*Updated: 2026-08-13*

## Session Extract — INT-4 Complete 2026-08-13 (Vertical Slice — code done, playtest pending)
- INT-4: UI rendering recipe found + applied, journal view, revelation proven, debug cleanup
- **THE UI RECIPE (applies to all production UI)**: markup + `position: absolute` + PX top/left/width/height + z-index; show/hide via entity `Activate()/Deactivate()` (console pattern — CSS display toggles DON'T re-apply layout); NO runtime UI tree mutations (static label pools — anyChildDirty panic); markup labels need zero-width-space content + `InnerLabel()` binding
- Files: dialogue_ui/panel.go (px markup + static 8-label pool), dialogue_ui/messages.go (pool rebuild), src/game/journal_view.go (NEW — slice journal, 14-label pool, J/tab toggle), src/game/{world,talk,game}.go (debug aids REMOVED: F1-F4, telemetry, cycle mode, debug_ui.go deleted), tests/unit/game/revelation_chain_test.go (2 tests — chain reaches revelation_ready after Vek→Tomas→Mira; contradiction flags facts)
- Live-verified: dialogue panel renders (233k parchment px bottom third), in-character responses, zero panics. Full 3-conversation live walkthrough blocked by OS focus flakiness — chain logic proven headlessly instead
- Findings for production: TryPlaneHit = camera-origin rays (custom world→screen projection math does NOT match it); ResolveAttack returns, callers apply damage; entity.Stats vs stats.StatsComponent duplication
- Tests: all INT suites PASS (game, dialogue_ui, ai_dialogue)
- **REMAINING (slice end)**: playtest Session 1 (human), then REPORT.md + prototypes/index.md verdict — vertical-slice skill Phases 6-8
- Build facts: standalone = cd kaiju/src && CGO_ENABLED=1 go build -tags="debug,ai_driver" -o ../../bin/game-standalone . ; run from PROJECT ROOT with LLAMA_SERVER_BIN env
*Updated: 2026-08-13*

## Session Extract — INT-2 Talk Loop Complete 2026-08-12 (Vertical Slice)
- Story: INT-2 (vertical slice integration) — Talk loop: 3 NPCs with .kg content, dialogue pipeline, LLM companion, thread wiring
- Files: assets/npc_knowledge/borderland/{blacksmith,miner,scholar}.kg (first game content), src/ai_dialogue/companion_codec.go + companion/main.go (stdin JSON ↔ llama-server), src/dialogue_ui/panel.go (REWRITTEN to markup) + messages.go (log/input/typewriter), src/game/talk.go (orchestration), world.go/world_config.go/game.go edits, tests: companion codec (11), PickNPC (5), WrapLines (5), .kg content validation (2)
- LIVE VERIFIED: standalone + ai_driver — dialogue opens on NPC click (PickNPC), F2 test question runs the full pipeline: MatchTopics → KG facts → BuildPrompt → StateMachine.Submit (text-mode fallback) → ProcessResponse → attributed in-character response ("I told the smith my ore had gone bad..." — Tomas). Evidence: production/qa/evidence/int2/
- FINDINGS (slice report material):
  1. **LLM too slow on dev machine**: capability probe gated — 1.2-1.4 tok/s vs 10 tok/s minimum → text mode. Real-model path WORKS (companion e2e test) but ~1 tok/s = ~70s per response. Performance constraint for real-time dialogue on this hardware.
  2. **Panel markup rendering bug**: dialogue panel (markup, absolute-positioned div) renders top-left at ~half size; backgrounds don't fill. Title screen (flow layout) + engine console (top:0 pattern) work. Diagnosis: absolute px/percent positioning in markup misrenders in this engine version. Deferred to targeted fix story; pipeline verified headlessly via logs.
  3. **Pick radius**: click ray ground-hit lands ~1.7u beyond NPC base (pitch through cube) — worldNPCClickRadius 1.5 → 3.0.
  4. **Matcher threshold**: 8-9 keywords/domain + 0.3 threshold = single-topic questions never match. Tuned slice to 0.1 (GDD range 0.1-0.6).
  5. Dialogue panel portrait never Init'd → engine panic (fixed: Init with square placeholder; also markup labels need InnerLabel() not ToLabel()).
  6. Dual ui.Manager: elements under a second manager don't render — panel uses game's shared manager (character_sheet pattern).
- Debug aids in updateWorld (remove for INT-4): F1 teleport to miner, F2 test question, per-second player/NPC screen-position logs
- Build facts: standalone run = CWD project root, LLAMA_SERVER_BIN env for companion; hooks fixed: settings.json hook commands anchored to $CLAUDE_PROJECT_DIR/.claude/hooks/ (cwd drift broke them)
- Next: INT-3 (COMBAT-003 attack resolution + mine encounter), then INT-4 (journal/revelation wiring + panel rendering fix + debug cleanup)
*Updated: 2026-08-12*

## Session Extract — /create-stories party-system 2026-08-12
- Verdict: COMPLETE — 14 stories written to production/epics/party-system/ (5 Ready, 9 Blocked)
- Review mode: lean — QL-STORY-READY gate skipped; QA plan check: no matching specs in production/qa/qa-plan-*.md
- Stories: 001 Roster (Logic, Ready), 002 State Machine (Logic, Blocked), 003 Recruitment (Integration, Ready), 004 Swap/Dismissal (Integration, Ready), 005 Re-recruitment (Logic, Ready), 006 Departure/Death (Integration, Blocked), 007 Combat Roster (Integration, Ready), 008 Behavior Modes Exploration (Logic, Blocked), 009 Combat Mode Translation (Integration, Blocked), 010 Scout (Logic, Blocked), 011 Formation (Logic, Blocked), 012 Knowledge Contribution (Integration, Blocked), 013 Rest-Point Gating (Logic, Blocked), 014 Management Screen (UI, Blocked)
- BLOCKERS (user-approved write-with-GAP-flags): TR-party-002..007 carry registry GAP "needs Party ADR"; 006/012 additionally need ADR-0008 revision (interjection interface); 008/009 need Party ADR or ADR-0009 extension. Unblock path: /architecture-decision → Party ADR, then flip stories to Ready
- User decision: write all 14 with GAPs flagged Blocked (not pause-for-ADR, not ready-only)
- Files: production/epics/party-system/story-001..014 (+EPIC.md stories table + requirements table updated; production/epics/index.md → "14 stories")
- Note: CR-5 split into two stories (008 exploration / 009 combat) per story sizing rule — 13 → 14 after presentation
- Next: /architecture-decision (Party ADR) to unblock 9 stories, or /story-readiness production/epics/party-system/story-001-party-roster-active-slots.md → /dev-story (001 is startable now)
- Other epics without stories: faction-system only. Sprint planning (/sprint-plan) viable once faction stories exist or party unblocks.
*Updated: 2026-08-12*

## Session Extract — INT-1 Game World Bootstrap Complete 2026-08-12 (Vertical Slice)
- Story: INT-1 (vertical slice integration) — Game World Bootstrap: New Game → debug-shape world, click-to-move, camera follow
- Files: src/game/world.go (world build, MoveToward, updateWorld), src/game/world_config.go (layout/tuning constants), src/game/game.go (+startNewGame, resize hook, world fields), src/game/title_screen.go (btn-new wired), tests/unit/game/world_movement_test.go (7 tests PASS)
- Verified APIs (against kaiju source): host.Cameras.Primary.ChangeCamera, StandardCamera.TryPlaneHit (ground picking), host.Window.Mouse (hid), NewMeshPlane/NewMeshCube + shader_data_registry basic color, Document.Destroy
- Verified LIVE via ai_driver: standalone build (debug,ai_driver — NOT editor tags; editor build shows the editor, not the game), run from PROJECT ROOT (`./bin/game-standalone`); title renders (4 buttons), New Game click → world renders, click-to-move works both directions (world shifts, camera follows player), no panics
- Evidence: production/qa/evidence/screenshots/int1-world-render.png + int1-player-moved.png
- Next: INT-2 (Talk loop — 3 NPCs with .kg content, dialogue UI wiring) then INT-3 (COMBAT-003 attack resolution + mine encounter), INT-4 (journal/revelation wiring)
- Key build facts: `make build-ai-driver` = editor+game (shows editor); standalone = `cd kaiju/src && CGO_ENABLED=1 go build -tags="debug,ai_driver" -o ../../bin/game-standalone .`, run with CWD=project root
*Updated: 2026-08-12*

## Session Extract — Investigation Journal Epic Complete 2026-08-12
- Epic: Investigation Journal — 5/5 Complete ✅ (001 Shell, 002 Chains/Threads, 003 Fact Log/Entities, 004 Contradictions/Revelations, 005 Map/Camera)
- Package: src/investigation_journal/ (9 files, ~1200 lines), 73 tests all PASS, go build + go vet clean
- Pending: manual walkthrough evidence, in-engine rendering, commit
*Updated: 2026-08-12*

## Session Extract — /architecture-review 2026-08-12
- Verdict: CONCERNS
- Requirements: ~267 total — ~128 covered (48%), ~61 partial (23%), ~78 gaps (29%)
- New TR-IDs registered: ~180 (TR registry v1 → v2)
- GDD revision flags: save-load-system.md (camera persistence conflict)
- Top ADR gaps: Audio System (8 TRs, no ADR), Loot & Economy (8 TRs, no ADR), AI Dialogue Engine revision (14 new TRs need ADR-0008 update), Map System (7 TRs, no ADR), Party System (6 TRs, no ADR)
- Report: docs/architecture/architecture-review-2026-08-12.md
- Blocking: Camera persistence conflict (Save/Load vs Camera GDD)

## Session Extract — /dev-story INVJ-003 2026-08-12
- Story: production/epics/investigation-journal/story-003-fact-log-entity-quickref.md — Fact Log & Entity Quick-Ref Bar
- Implemented directly (Story 003 — in progress, not yet /code-review or /story-done)
- Files created: src/investigation_journal/fact_log.go, src/investigation_journal/entity_chips.go, tests/integration/investigation_journal/fact_log_test.go
- Files modified: src/investigation_journal/state.go (+StateEntitySelected), src/investigation_journal/journal.go (+tabs, fact log, entity chips, quick-ref, nav stack)
- Test results: 58 total (20+25+13), all PASS, go build + go vet clean
- Status: Implementation complete, awaiting /code-review and /story-done
- Next: /code-review src/investigation_journal/ then /story-done production/epics/investigation-journal/story-003-fact-log-entity-quickref.md

## Session Extract — /story-done map-system 002 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/map-system/story-002-map-markers-pins.md — Map Markers & Pins
- Tech debt logged: None (advisory: H2 star stacking, H3 renderer coverage, provider-contract QA edges)
- Code review: APPROVED WITH SUGGESTIONS — all required fixes applied
- Test suite: 19 integration + 25 unit = 44 tests, all PASS, go build + go vet clean
- Map System epic: 2/3 Complete ✅
- Next recommended: MAP-003 (World Map & Region Travel) or remaining open epics

## Session Extract — /dev-story PWS 001 2026-08-12
- Story: production/epics/persistent-world-state/story-001-world-flag-store-query-api.md — World Flag Store & Query API
- Agent: gameplay-programmer (this session)
- Files created: src/persistent_world_state/flag_store.go (443 lines), tests/unit/persistent_world_state/flag_store_test.go (540 lines)
- Design decisions: (1) PersistentAccessor interface injection (not bare constructor) — needed for Persistent component existence check; (2) Added SetEntityRegion + ClearWorldFlag — region scoping and flag deletion gaps found during implementation; (3) Soft limit for max_flags_per_entity (warn + write, per EF pattern); (4) Storage delegates to entity.Persistent.StateFlags — preserves non-world.* namespaces
- Test results: 18/18 PASS, go build + go vet clean
- Blockers: None
- Next: /code-review src/persistent_world_state/flag_store.go then /story-done production/epics/persistent-world-state/story-001-world-flag-store-query-api.md

## Session Extract — /story-done map-system 003 2026-08-12
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/map-system/story-003-world-map-region-travel.md — World Map & Region Travel
- Tech debt logged: None (advisory: zoom navigation untested, click routing untested, hazard routes not implemented)
- Code review: APPROVED WITH SUGGESTIONS — S3 debounce test fix applied
- Test suite: 23 world map + 19 markers + 25 unit = 67 tests, all PASS, go build + go vet clean
- Map System epic: 3/3 Complete ✅
- Next recommended: Create stories for remaining epics (PWS, party-system, faction-system) or continue Investigation Journal epic

## Session Extract — /dev-story map-system 003 2026-08-12
- Story: production/epics/map-system/story-003-world-map-region-travel.md — World Map & Region Travel
- Agent: ui-programmer (2 passes — research then implementation)
- Files created: src/ui/map/world_map.go (1678 lines), tests/integration/map_system/world_map_test.go (23 tests)
- Files modified: src/ui/map/local_map.go (+WorldMap integration, world button), src/ui/map/render.go (+world button geometry), assets/ui/map.html (+world map overlay markup)
- Design decisions: (1) WorldMap follows LocalMap pattern — headless-testable, provider interfaces, paper-slide; (2) Region shapes via boundary LineSegments with state-based ink treatments (dashed/faint/solid); (3) Travel routes as dotted ink lines between region centers; (4) Faction territory as colored wash with hatch patterns (never hue alone); (5) Detail panel paper-slides from right side; (6) Navigation: World button on local map → zoom out, click region on world → zoom in
- Test results: 23 world map tests + 19 marker tests + 25 unit = 67 total, all PASS, go build + go vet clean
- Bugs found and fixed during implementation: (1) faction snapshot leak — empty entries created for regions without factions; (2) PointInRegion boundary exclusion — strict > instead of >=; (3) hatch segment floating-point escape from clip rect
- Not verified: in-engine rendering (manual QA evidence pending)
- Blockers: None
- Map System epic: 3/3 Complete ✅
- Next: /code-review src/ui/map/world_map.go then /story-done production/epics/map-system/story-003-world-map-region-travel.md

## Session Extract — /dev-story map-system 002 2026-08-11
- Story: production/epics/map-system/story-002-map-markers-pins.md — Map Markers & Pins
- Agent: ui-programmer (this session)
- Files created: src/ui/map/markers.go, tests/integration/map_system/map_markers_test.go
- Files modified: src/ui/map/local_map.go (+RegionID, +3 provider setters, +RefreshMarkers, +Markers, +snapshot), src/ui/map/render.go (+4 marker templates, +rebuildMarkers, +marker geometry), assets/ui/map.html (+4 marker templates + CSS)
- Design decisions: (1) RegionID on RegionMapData as provider query key — no silent Name fallback; (2) Setters for providers (not constructor args) — backward-compatible with Story 001; (3) Marker snapshot model — defensive copy at query time, RefreshMarkers is the only push path; (4) Amber star = 5-spoke shape (not just * glyph) — shape-distinct from dot for colorblind accessibility; (5) Ink X = two crossed arms at ±45° overlaying the location pin
- Test results: 17 integration tests PASS, 25 unit tests PASS (no regressions), go build + go vet clean
- Not verified: in-engine rendering (manual QA evidence pending)
- Blockers: None
- Next: /code-review src/ui/map/markers.go src/ui/map/render.go src/ui/map/local_map.go then /story-done production/epics/map-system/story-002-map-markers-pins.md

## Session Extract — /dev-story INVJ-002 2026-08-11
- Story: production/epics/investigation-journal/story-002-chain-list-thread-detail.md — Chain List Sidebar & Thread Detail View
- Implemented directly (ui-programmer agent pattern avoided after Story 001 research-loop)
- Files created: src/investigation_journal/chain_list.go, src/investigation_journal/thread_detail.go, tests/integration/investigation_journal/chain_thread_display_test.go
- Files modified: src/investigation_journal/state.go (+StateChainSelected), src/investigation_journal/journal.go (+LoadData, SelectChain, DeselectChain, sidebar/detail population)
- Test results: 45 total (20 story 001 + 25 new), all PASS, go build + go vet clean
- Design decisions: (1) Pure-Go display functions (BuildChainList, BuildThreadList, FormatThreadAge, ThreadStatusIcon/Label) for engine-free testing; (2) ChainSelected as flat enum state; (3) clearPanel is a VERIFY stub — children accumulate until Kaiju panel-clear API verified; (4) SourceNPC resolved as 0 placeholder — entity name lookup needs cross-system integration
- Blockers: None
- Next: /code-review src/investigation_journal/ then /story-done production/epics/investigation-journal/story-002-chain-list-thread-detail.md

## Session Extract — /story-done INVJ-001 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/investigation-journal/story-001-journal-shell-layout.md — Journal Shell & Three-Panel Layout
- Tech debt logged: None (advisory: hardcoded layout constants, J key un-gated, AC-4 camera wiring deferred to Story 005)
- Code review: APPROVED WITH SUGGESTIONS — 4 fixes applied (animation interrupt, real test calls, sidebar slide-in, AC-4 deferral)
- Test suite: 20 tests, all PASS, go build + go vet clean
- Next recommended: INVJ-002 (Chain List & Thread Detail) — now unblocked

## Session Extract — /story-done map-system 001 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/map-system/story-001-local-map-display.md — Local Map Display & Navigation
- Tech debt logged: None
- Test suite: 25 tests, all PASS, go build + go vet clean
- Code review: APPROVED WITH SUGGESTIONS — all fixes applied
- Advisory: manual walkthrough evidence pending (production/qa/evidence/)
- Next recommended: MAP-002 (Map Markers & Pins) — now unblocked

## Session Extract — /dev-story INVJ-001 2026-08-11
- Story: production/epics/investigation-journal/story-001-journal-shell-layout.md — Journal Shell & Three-Panel Layout
- Agent: ui-programmer (3 attempts, ~300k tokens — stuck in research loop; implemented directly)
- Files created: src/investigation_journal/state.go, src/investigation_journal/events.go, src/investigation_journal/journal.go, assets/ui/investigation_journal.html, tests/integration/investigation_journal/journal_shell_test.go
- Test results: 17/17 PASS, go build + go vet clean
- Design decisions: (1) Programmatic UI (dialogue_ui pattern) — HTML file retained as design reference for Story 002+; (2) EventBus as interface type, not pointer-to-interface; (3) UIUpdater for pre-render animation pass; (4) Esc only closes when Open — falls through to pause when Closed
- Not verified: in-engine rendering (screenshot evidence pending — Kaiju UI API compatibility verified against dialogue_ui pattern but runtime testing needed)
- Blockers: None
- Next: /code-review src/investigation_journal/ then /story-done production/epics/investigation-journal/story-001-journal-shell-layout.md

## Session Extract — /dev-story map-system 001 2026-08-11
- Story: production/epics/map-system/story-001-local-map-display.md — Local Map Display & Navigation
- Agent: ui-programmer (this session)
- Files created: src/ui/map/local_map.go, src/ui/map/render.go, src/ui/map/tween.go, assets/ui/map.html, tests/unit/map_system/local_map_display_test.go
- Design decisions: (1) Package `local_map` since `map` is a Go keyword (same convention as `combat_hud`); (2) Engine-free models + renderer pattern following combat_hud; (3) Paper-slide cubic-bezier tween Go-driven per frame (Kaiju CSS has no transitions); (4) Region geography as line segments with inline CSS rotate; (5) PositionProvider interface for player position (game layer implements)
- Test results: 16 unit tests PASS, go build + go vet clean
- Not verified: in-engine rendering (manual QA evidence pending — needs running game for M/Esc hotkeys, drag, edge-scroll, visual verification)
- Blockers: None
- Next: /code-review src/ui/map/ then /story-done production/epics/map-system/story-001-local-map-display.md

## Session Extract — /story-done AI-DIALOGUE-005 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/ai-dialogue-engine/story-005-tiered-routing-tiny-cgo.md — Tiered Routing & Tiny Model CGo Backend
- Tech debt logged: None (advisory: stub CGo generation, reload-after-panic deferred, full→fast fallback per ADR tension)
- Code review: APPROVED WITH SUGGESTIONS — 7 fixes applied
- Test suite: 44 tests, all PASS, go vet clean
- AI Dialogue Engine epic: 5/5 Complete ✅
- Next recommended: Dialogue UI epic (state machine events consumed by UI) or remaining open epics

## Session Extract — /dev-story AI-DIALOGUE-005 2026-08-11
- Story: production/epics/ai-dialogue-engine/story-005-tiered-routing-tiny-cgo.md — Tiered Routing & Tiny Model CGo Backend
- Files created: src/ai_dialogue/query_classifier.go, src/ai_dialogue/tiered_backend.go, src/ai_dialogue/tiny_cgo_backend.go
- Test written: tests/integration/ai_dialogue/tiered_backend_test.go (17 new tests)
- Test suite: 44 tests all PASS, go vet clean, go build OK
- Blockers: None
- Deviations: TinyCGoBackend uses stub generation — real CGo llama.cpp bindings not yet available (noted in TODO markers)
- Next: /code-review src/ai_dialogue/query_classifier.go src/ai_dialogue/tiered_backend.go src/ai_dialogue/tiny_cgo_backend.go then /story-done

## Session Extract — /story-done AI-DIALOGUE-004 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/ai-dialogue-engine/story-004-state-machine-degradation-memory.md — State Machine, Degradation & Conversation Memory
- Tech debt logged: None (advisory: TextFallbackGenerator unwired, PWS integration deferred, hardcoded config constants)
- Code review: CHANGES REQUIRED → 2 blocking bugs fixed (generation token ABA race, timeout classification), 2 regression tests added
- Test suite: 27 tests, all PASS, race-detector clean, go vet clean
- Next recommended: AI-DIALOGUE-005 (Tiered Routing & Tiny CGo) or Dialogue UI epic (state machine events consumed by UI)

## Session Extract — /story-done thread-investigation-001 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/thread-investigation/story-001-thread-data-model-auto-creation.md — Thread Data Model & Auto-Creation
- Tech debt logged: None
- Test suite: 23 tests, all PASS, benchmark ~8µs/op
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Next recommended: THREAD-002 (Auto-Chaining & Entity Overlap) — now unblocked

## Session Extract — /dev-story thread-investigation-002 2026-08-11
- Story: production/epics/thread-investigation/story-002-auto-chaining-entity-overlap.md — Auto-Chaining & Entity Overlap
- Agent: gameplay-programmer (background)
- Dependencies: Story 001 (Complete), ADR-0010 (Accepted)
- Files modified: src/thread_investigation/types.go (+OverlapFlag, +ChainStatusArchived), src/thread_investigation/events.go (+chain event types + ChainEvent envelope), src/thread_investigation/engine.go (+Step 3/3b, entityChainIndex, DrainChainEvents, GetOverlapFlags, WithAutoChainMerge)
- Files created: tests/unit/thread_investigation/auto_chaining_test.go (15 tests)
- Story 001 legacy: thread_creation_test.go updated — 23 tests pass with ChainID assertions adapted
- Total suite: 38 tests, all PASS, go vet clean
- Blockers: None
- Next: /code-review src/thread_investigation/ then /story-done

## Session Extract — /dev-story thread-investigation-001 2026-08-11
- Story: production/epics/thread-investigation/story-001-thread-data-model-auto-creation.md — Thread Data Model & Auto-Creation
- Agent: gameplay-programmer (background)
- Dependencies verified: KG-001 (Complete), KG-005 (Complete), EF-001 (Complete), EF-002 (Complete), ADR-0010 (Accepted)
- Manifest versions match: story 2026-08-09 = current 2026-08-09
- Files created: src/thread_investigation/types.go, src/thread_investigation/events.go, src/thread_investigation/engine.go
- Test file: tests/unit/thread_investigation/thread_creation_test.go (11 tests, all PASS, benchmark=6.9µs/op — ~290× under 2ms budget)
- go vet: clean
- Blockers: None
- Next: /code-review src/thread_investigation/ then /story-done

## Session Extract — /dev-story AI-DIALOGUE-004 2026-08-11
- Story: production/epics/ai-dialogue-engine/story-004-state-machine-degradation-memory.md — State Machine, Degradation & Conversation Memory
- Files created: src/ai_dialogue/clock.go, src/ai_dialogue/conversation_memory.go, src/ai_dialogue/text_fallback.go, src/ai_dialogue/state_machine.go
- Test written: tests/integration/ai_dialogue/state_machine_test.go (24 new + 4 existing = 27 total, all PASS, race-detector clean, go vet clean)
- Blockers: None
- Next: /code-review src/ai_dialogue/state_machine.go src/ai_dialogue/conversation_memory.go src/ai_dialogue/text_fallback.go src/ai_dialogue/clock.go then /story-done

## Session Extract — /story-done AI-DIALOGUE-003 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/ai-dialogue-engine/story-003-response-post-processor.md — Response Post-Processor
- Files created: src/ai_dialogue/response_processor.go (483 lines), tests/unit/ai_dialogue/response_processor_test.go (416 lines)
- Code review: CHANGES REQUIRED → 3 fixes applied (ReplaceAll word-boundary bug, AllNames hoisting, ProcessResponse complexity reduction, deadband test corrected)
- Test suite: 25 tests, all PASS
- Known deferrals: multi-word n-gram matching, pronoun/role substitution → later story (documented in header)
- AC-13 scope note: NeedsRetry flag emitted; actual re-query orchestration is Story 004
- Next recommended: AI-DIALOGUE-004 (State Machine Orchestration) — now unblocked

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/dialogue-ui/story-001-dialogue-panel-framework.md — Dialogue Panel Framework & Lifecycle
- Files created: src/dialogue_ui/panel.go, src/dialogue_ui/state.go, src/dialogue_ui/input_handler.go
- Test written: tests/integration/dialogue_ui/panel_framework_test.go (16 tests, all PASS)
- Blockers: None
- Deviations: Body text at 16px reference scales to 10.67px at 720p (below 12px minimum) — needs height-based font scaling with floor, deferred to font system story
- Next: /code-review src/dialogue_ui/ then /story-done production/epics/dialogue-ui/story-001-dialogue-panel-framework.md

## Session Extract — /story-done C&I UI 001 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/character-inventory-ui/story-001-character-sheet-display.md — Character Sheet Stat Display
- Tech debt logged: None (advisory: screenshot evidence pending, tween duplication noted)
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 4 tests, all PASS
- C&I UI epic: 1/9 Complete
- Next recommended: C&I UI 002 (XP Progress Bar & Level-Up Modal) — depends on Story 001 ✅

## Session Extract — /dev-story C&I UI 001 2026-08-11
- Story: production/epics/character-inventory-ui/story-001-character-sheet-display.md — Character Sheet Stat Display
- Agent: ui-programmer (background)
- Files created: src/character_sheet/sheet.go, src/character_sheet/slide.go, assets/ui/character_sheet.html, tests/unit/character_sheet/sheet_test.go
- Test results: 4/4 PASS, go vet clean
- Design decisions: bar fill = value/18 (satisfies AC-3 2.0 ratio), U+2212 minus sign, pure display functions for engine-free testing
- Blockers: None
- Manual evidence needed: screenshot at production/qa/evidence/story-001-character-sheet-display-evidence.md
- Next: /code-review src/character_sheet/ tests/unit/character_sheet/ then /story-done

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/dialogue-ui/story-001-dialogue-panel-framework.md — Dialogue Panel Framework & Lifecycle
- Agent: ui-programmer (background, ID: a617d30e462aba3cc)
- Dependencies verified: None (first story in epic), ADR-0012 (Accepted), manifest v2026-08-09 match
- Files expected: src/dialogue_ui/panel.go, tests/integration/dialogue_ui/panel_framework_test.go
- Blockers: None
- Next: /code-review then /story-done

## Session Extract — /story-done STAT-005 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/character-stats-progression/story-005-stats-component-integration.md — Stats Component Integration
- Tech debt logged: None (entity.Stats retirement planned in agent memory)
- Code review: APPROVED WITH SUGGESTIONS — all 4 fixes applied (comment dedup, xp≤0 test, event contract, entity.Stats plan)
- Test suite: 8 integration tests, all PASS
- Character Stats epic: 5/5 Complete ✅
- Next recommended: C&I UI Story 001 — fully unblocked (all 5 Stats stories Complete)

## Session Extract — /dev-story 2026-08-11 (AI Dialogue Engine — Prompt Builder)
- Story: production/epics/ai-dialogue-engine/story-002-prompt-builder.md — Prompt Builder
- Agent: ai-programmer (a8bc95ab17d115da6)
- Files created: src/ai_dialogue/prompt_builder.go (756 lines), tests/unit/ai_dialogue/prompt_builder_test.go (817 lines)
- Test: 21/21 PASS, p95=412µs (12× under 5ms budget), go vet clean
- Bugs found & fixed: factTier panic on empty MatchedDomains, AC-2 test counted IDs not content
- Design decisions: 495+marker char cap, IsFallback guard for tier-4 routing, EstimateTokens MVP (word-count, 4/3 ratio)
- Blockers: None
- Next: /code-review src/ai_dialogue/prompt_builder.go then /story-done

## Session Extract — /dev-story STAT-005 2026-08-11
- Story: production/epics/character-stats-progression/story-005-stats-component-integration.md — Stats Component Integration
- Agent: gameplay-programmer (background)
- Files created: src/stats/component.go, src/stats/events.go, tests/integration/stats/stats_integration_test.go
- Design decision: AwardXPAndNotify bridge (not inline event in AwardXP) — keeps stats pure-math
- Test results: 7/7 PASS (integration), full unit suite PASS (no regressions), go vet clean
- Cross-package: entity, knowledge, save-load integration tests all PASS
- Existing files untouched: attributes.go, xp.go, creation.go
- Blockers: None
- Next: /code-review src/stats/component.go src/stats/events.go tests/integration/stats/stats_integration_test.go then /story-done
- Then: C&I UI Story 001 fully unblocked

## Session Extract — /story-done STAT-003 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/character-stats-progression/story-003-character-creation.md — Character Creation
- Tech debt logged: None
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 7 tests, all PASS
- Next recommended: STAT-005 (Stats Component Integration) — now unblocked, then C&I UI Story 001

## Session Extract — /dev-story STAT-003 2026-08-11
- Story: production/epics/character-stats-progression/story-003-character-creation.md — Character Creation
- Agent: gameplay-programmer (background)
- Files created: src/stats/creation.go, tests/unit/stats/creation_test.go
- Test results: 7/7 PASS, go vet clean
- Blockers: None
- Next: /code-review src/stats/creation.go tests/unit/stats/creation_test.go then /story-done production/epics/character-stats-progression/story-003-character-creation.md
- Then: STAT-005 (Stats Component Integration) unblocks → C&I UI Story 001

## Session Extract — /dev-story 2026-08-11 (Trust & Relationship)
- Story: production/epics/trust-relationship/story-001-trust-levels-events.md — Trust Levels & Trust Events
- Agent: gameplay-programmer (adbe36499af786d68)
- Files created: src/trust_relationship/ (types.go, events.go, trust_system.go)
- Test: tests/unit/trust_relationship/trust_events_test.go — 18/18 PASS, go vet clean
- Design decisions: float64 game-minutes, record-all apply-highest, CR-5 caps in Config, priority-1 EventBus
- Refinements: sequential gain downgrade guard, one-time-per-thread check ordering
- Blockers: None
- Next: /code-review src/trust_relationship/ then /story-done

## Session Extract — /story-done 2026-08-11 (Combat Engine — Sprint 5)
- Verdict: COMPLETE WITH NOTES (Story 002)
- Story: production/epics/tactical-combat-engine/story-002-grid-movement-ap-economy.md — Grid Movement & AP Economy
- Tech debt logged: None (W1-W3 advisory — AP config drift, nil-pool, formula duplication)
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 14 unit + 23 integration, all PASS
- Sprint 5 Must Have: COMBAT-001 ✅, COMBAT-002 ✅ — both done
- Next recommended: COMBAT-003 (Attack Resolution d20) — now unblocked; COMBAT-004 (Positioning) also unblocked

## Session Extract — /story-done 2026-08-11 (Question System)
- Verdict: COMPLETE
- Story: production/epics/question-system/story-001-topic-matcher-domain-scoring.md — Topic Matcher & Domain Scoring
- Tech debt logged: None (advisory: entity-bonus reading of AC-1, FromContext field extension)
- Code review: APPROVED
- Test suite: 21 tests, all PASS, p95=157µs
- Next recommended: Q-002 (Question Input & State Machine) — depends on Thread Investigation epic (stories not yet created)

## Session Extract — /dev-story 2026-08-11 (Question System)
- Story: production/epics/question-system/story-001-topic-matcher-domain-scoring.md — Topic Matcher & Domain Scoring
- Agent: gameplay-programmer (a3ada85901a0ace18)
- Files created: src/question_system/ (types.go, matcher.go, keywords.go)
- Test: tests/unit/question_system/topic_matcher_test.go — 21/21 PASS, go vet clean
- Performance: p95 = 142µs (35× under 5ms budget)
- Design decisions: entity-bonus reading of AC-1 ("old mine" as location entity), FromContext field on DomainScore, EventQuestionMatched constant
- Pre-existing failure: tests/unit/combat/grid_movement_test.go:448 (syntax error, unrelated)
- Blockers: None
- Next: /code-review src/question_system/ then /story-done

## Session Extract — /story-done 2026-08-11 (AI Dialogue Engine)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/ai-dialogue-engine/story-001-llm-backend-ipc.md — LLM Backend Interface & IPC Subprocess
- Tech debt logged: None (advisory: ADR-0008 degradation table drift vs GDD CR-5)
- Code review: CHANGES REQUIRED → 3 fixes applied (restart-during-shutdown leak, SIGTERM alignment, nil-err guards)
- Test suite: 4 tests, all PASS, -race clean
- Next recommended: AI-DIALOGUE-002 (Prompt Builder) — now unblocked. Note: depends on Question System and Trust & Relationship epics having stories created.

## Session Extract — /story-done 2026-08-11 (Audio System)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/audio-system/story-001-audio-engine-foundation.md — Audio Engine Foundation & Spatial Listener
- Tech debt logged: None (advisory: AC-5 hot-unplug deferred — engine limitation)
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 14 tests, all PASS
- Next recommended: AUDIO-002 (Ambient Soundscapes) — now unblocked

## Session Extract — /story-done 2026-08-11 (Combat Engine)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/tactical-combat-engine/story-001-combat-state-machine-initiative.md — Combat State Machine & Initiative
- Tech debt logged: None (advisory: ADR-0011 event drift + Manager ISP)
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 31 test runs, all PASS
- Next recommended: COMBAT-002 (Grid Movement & AP Economy) — now unblocked

## Session Extract — /dev-story 2026-08-11 (Audio System)
- Story: production/epics/audio-system/story-001-audio-engine-foundation.md — Audio Engine Foundation & Spatial Listener
- Agent: gameplay-programmer (background, ID: a41120b63b315ee45)
- Files created: src/audio/ (audio.go, backend.go, bootstrap.go, config.go, listener.go, loader.go), tests/integration/audio/audio_foundation_test.go
- Test results: 14/14 PASS (0.023s), package compiles clean
- Engine gap documented: Kaiju Go bindings don't expose SoLoud 3D API (SpatialSink interface ready for engine patch)
- Blockers: None
- Next: /code-review src/audio/ then /story-done production/epics/audio-system/story-001-audio-engine-foundation.md

## Session Extract — /dev-story 2026-08-11 (Combat Engine)
- Story: production/epics/tactical-combat-engine/story-001-combat-state-machine-initiative.md — Combat State Machine & Initiative
- Agent: gameplay-programmer (a85d30e37645327ae)
- Files created: src/combat/ (combat.go, events.go, initiative.go, enemy_ai.go, config.go — 1035 lines)
- Test: tests/integration/combat/state_machine_test.go — 24/24 PASS, go vet clean
- Bugs found & fixed: RemoveCombatant TurnEnded emission order; flee test expectation; perf test round count
- Deviations: IsCombatActive() save gating not wired (cross-domain — save epic owns it)
- Blockers: None
- Next: /code-review src/combat/ then /story-done

## Session Extract — /dev-story 2026-08-11 (AI Dialogue Engine)
- Story: production/epics/ai-dialogue-engine/story-001-llm-backend-ipc.md — LLM Backend Interface & IPC Subprocess
- Agent: ai-programmer (a8f1cace7f38204cb)
- Files created: src/ai_dialogue/ (backend.go, ipc.go, config.go, process.go, probe.go, warmup.go, llm_backend.go)
- Test: tests/integration/ai_dialogue/llm_backend_test.go + stub/main.go — 4/4 PASS, go vet clean, -race clean
- Bug found & fixed: IPC read loop EOF raced with cmd.Wait — Generate leaked ErrProcessExited instead of ErrModelCrashed
- Full suite regression: only pre-existing integration/audio failure (missing assets/audio/ fixture, unrelated)
- Blockers: None
- Next: /code-review src/ai_dialogue/ then /story-done

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/tactical-combat-engine/story-001-combat-state-machine-initiative.md — Combat State Machine & Initiative
- Agent: gameplay-programmer (background, ID: a85d30e37645327ae)
- Dependencies verified: EF-001 (Complete), STAT-001 (Complete), ENV-001 (Complete), ADR-0009 (Accepted), ADR-0007 (Accepted), ADR-0011 (Accepted)
- Manifest versions match: story 2026-08-09 = current 2026-08-09
- TR-IDs registered: TR-combat-006 through TR-combat-009 added to tr-registry.yaml
- 8 stories written for tactical-combat-engine epic
- Blockers: None
- Next: Wait for agent to complete, then /code-review then /story-done

## Session Extract — /story-done 2026-08-10
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-002-component-model.md — Component Model
- Tech debt logged: None
- Next recommended: EF-003 (Spatial Index) — ready-for-dev, no blockers

## Session Extract — /story-done 2026-08-10 (#2)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-003-spatial-index.md — Spatial Index
- Tech debt logged: None
## Session Extract — /story-done 2026-08-10 (#3)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-004-write-arbitration.md — Write Arbitration
- Tech debt logged: None
- Next recommended: EF-005 (Serialization) — ready-for-dev, all blockers cleared

## Session Extract — /story-done 2026-08-10 (#4)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-005-serialization.md — Serialization
- Tech debt logged: None
- Next recommended: EF-006 (Performance Validation) — ready-for-dev, all blockers cleared

<!-- QA-PLAN: 2026-08-10 | System: sprint-2 | Plan written: production/qa/qa-plan-sprint-2-2026-08-10.md -->
<!-- STATUS -->
Epic: Save/Load System
Feature: Sprint 3 — Must Have stories
Task: SAVE-001 complete; SAVE-002 next
Task: QA plan written, ready for implementation
<!-- /STATUS -->

## Session Extract — /gate-check Production → Polish 2026-08-12
- Verdict: FAIL (unanimous NOT READY — all 4 directors)
- 15 blockers consolidated across specification, technical, production, visual domains
- 5/11 required artifacts present, 2/10 quality checks passing
- Critical gaps: 0 playtests, no runnable client, ADE GDD unapproved, CI never green, no milestone defined
- Report: production/gate-checks/gate-check-production-polish-2026-08-12.md

## Session Extract — Playtest Planning 2026-08-12
- Playtest plan created: production/playtests/playtest-plan-2026-08-12.md
- Session report template: production/playtests/session-report-template.md
- 3 sessions planned: New Player Experience, Mid-Game Systems, Difficulty & Depth
- Screenshot baseline requirements defined (18 screens)
- Fun hypothesis validation criteria defined
- Next: Fix build → run game client → execute Session 1

## Session Extract — First Game Launch 2026-08-12 ✅
- Game client RUNS — first in-engine render achieved (addresses gate blocker B3)
- Files created: src/game/game.go (GameInterface), src/game/title_screen.go (markup UI),
  src/game/content_db.go (extension-mapped asset DB), kaiju/src/main.marrow.go (getGame),
  game_content/ (renderer/shaders/fonts/textures/ui copied from engine content)
- kaiju/src/main.test.go → main.test.go.bak (avoided getGame conflict)
- Engine go.mod: +require/replace marrow-of-empire
- Title screen verified: soot backdrop #141210, bone-ivory title #E9E6DC, ink buttons —
  1054 unique colors, art bible palette exact match, screenshot saved to
  production/qa/evidence/screenshots/title-screen-2026-08-12.png
- Makefile: build-standalone / run-standalone targets (debug,ai_driver tags)
- Key lessons: ContentDatabase must be non-nil; engine content needs extension mapping;
  imperative UI Scale doesn't size panels — use markup system; editor runs from kaiju/,
  standalone from project root
- Skill created: .claude/skills/first-launch/ — full bootstrap workflow + failure catalog
- Next: playtest Session 1 can now run against the real client

## Current Task

- **Sprint 2 planned**: Camera System & Infrastructure Remediation (2026-08-11 to 2026-08-17)
- **Sprint plan**: production/sprints/sprint-2.md
- **QA plan**: production/qa/qa-plan-sprint-2-2026-08-10.md
- **Retrospective**: production/retrospectives/retro-sprint-1-2026-08-10.md
- **Next**: CAM-003 and CAM-004 both unblocked — can run in parallel

## Session Extract — /story-done 2026-08-10 (#7)
- Verdict: COMPLETE
- Story: production/epics/camera-system/story-002-zoom-system.md — Zoom System
- Tech debt logged: None
- Next recommended: CAM-003 (Follow & Framing) + CAM-004 (Occlusion & Boundaries) — both unblocked

## CAM-003 Implementation — 2026-08-10 (engine-programmer session)

- **Story**: production/epics/camera-system/story-003-follow-framing.md — Follow & Framing
- **Deliverables written**: src/camera/state.go (base states + overlays, one-deep combat stack, journal close returns to prior base), src/camera/follow.go (PositionSource, party/combat tracking, conversation + combat framing, off-screen marker data contract), src/camera/pan.go (drag pan, edge scroll w/ hysteresis, keyboard pan), src/camera/controller.go (composition root), src/camera/camera.go (added Focus/SetFocus/ViewportSize hooks), tests/integration/camera/follow_framing_test.go (17 tests, AC-1..AC-23 + CR-4)
- **Coordinator-approved decisions**: controller.go composition root; combat framing 40 u/s (GDD CR-7); party-spread threshold = O×AR
- **Bugs found & fixed during test pass**: controller.Update passed seconds to cam.Update (zoom API is milliseconds — 300ms transition advanced 0.3ms/frame); pan edge-scroll/keyboard speeds went through world-per-pixel (15/20 u/s became ~0.3 u/s) — added panByWorldDelta; test fixture Y/Z slot swaps in two combat fixtures; test kept-slice used unsorted registration order for the 75% trim expectation
- **Test results**: all green — tests/integration/camera (17 CAM-003 + 6 CAM-004 untouched), tests/unit/camera (CAM-002 zoom), tests/unit, tests/integration/entity; `go build ./... && go vet ./...` clean in both src and tests modules
- **Status**: awaiting /story-done review for CAM-003; CAM-004 files untouched (boundary.go, occlusion.go, occlusion_boundaries_test.go)

## Art Bible Progress

- [x] Section 1 (Visual Identity Statement) — one-line rule + 3 pillar-anchored principles + cross-cutting readability constraint
- [x] Section 3 (Lighting & Atmosphere) — mood targets per game state, user warmth directive accepted
- [x] Section 2 (Color Palette) — user-accepted forks: divergent UI register; palette cools with depth; moss = reclamation, health = amber
- [x] Section 4 (Character Art Direction) — 6 topics, written to file 2026-08-09
- [x] Section 5 (Environment & Level Art) — 6 topics, written to file
- [x] Section 6 (UI Visual Language) — 6.1 register split (world/desk/field), 6.2 typography (3 faces), 6.3 iconography (pen-and-ink), 6.4 motion language, 6.5 color application, 6.6 combat HUD, 6.7 accessibility & input contract + cross-GDD resolution record; written to file 2026-08-09
- [x] Section 7 (VFX & Particle Style) — 6 topics, written to file 2026-08-09
- [x] Section 8 (Asset Standards) — 8.1 formats, 8.2 naming, 8.3 texture tiers, 8.4 LOD, 8.5 export philosophy, 8.6 KitBash3D integration; written to file 2026-08-09; engine values pending technical-artist merge (marked in subsections)
- [x] Section 9 (Style Prohibitions) — 8 prohibitions (never/why-cited/instead), written to file 2026-08-09
- [x] Section 10 (Reference Direction) — 5 curated references (what-to-take / what-to-avoid / why-it-matters), additive rule, written to file 2026-08-09
- [x] Section 2b (Shape Language) — inserted between 2 and 3 (no renumbering; 34 Section refs + 138 subsection refs across 3–10 make renumbering high-risk); 6 subsections written to file 2026-08-09

## Section 6 Key Decisions

- **Three registers, one vocabulary**: desk (parchment overlays, world dims 30%), field (combat HUD, panel umber base), world (anchored UI, scales 4/10/20u, Pillar 3 lighting discipline). Thread-red line crosses registers — journal connectors and combat targeting line are the same visual.
- **Typography**: desk serif (Garamond class, warm, OFL-licensed, never blackletter), combat face (tabular grotesque for all HUD numerals), world lettering (painted, never typeset). Reference 1920×1080, min 1280×720; world-anchored text follows 4.4 LOD, ≥14px combat face at Far.
- **Iconography**: pen-and-ink, one stroke weight, no gradients; shape carries meaning, color carries accent (2.5 built into the drawing style). Map markers = thread vocabulary pins; resolved = ink check (no green/amber).
- **Motion**: paper slides (300ms in/200ms out), thread draws itself, pulses rationed (amber = reward, oxblood = tension), desk never trembles (screen-space immune to shake), no bounce/3D. Typewriter = word-chunk reveal 45–60 cps.
- **Color application**: hover = amber edge (lamp), selection/links = thread-red, destructive = oxblood, disabled = desaturated "bleached" (saturation is life), focus ring = oxblood-on-parchment / parchment-on-umber ≥3:1 (amber fails on parchment), ivory rationed to empire items, text never amber-on-parchment (fails AA).
- **Combat HUD**: panel umber 85–90% base; health = faction-fixed hue (amber allies / oxblood enemies), umber/soot track, level by length + number, low-HP pulse outline, NO hue shift as HP falls; AP = amber; turn order bar = panel umber (not parchment), current turn amber rim; damage numbers oxblood + soot outline, crits amber flash; targeting line thread-red; flanking arcs = cold indigo (ratified into UI palette as combat spatial cue, shape+motion distinct from target rim); cover = neutral ink shield; End Turn amber; victory/defeat returns to desk register.

## Section 7 Key Decisions

- **Particle aesthetic**: air is a material and a witness (Principle 1) — vocabulary from nature/hearth only (embers, ash, dust motes, pollen, mist, fireflies); legend filter at effect scale ("breath of the old world," never plasma bolt); material truth rule (behavior = meaning; ghost glow is the only exception); three strata of air (present/empire/wild) with family exclusivity.
- **Vocabulary by source** (each inherits color semantics + motion signature + rarity): present = flickering, irregular, anchored to visible source (lantern = player's portable emitter); empire = metronomic breath, no flicker/falloff/source, activation = "old god waking," rationed (one breathing glow per scene); wild = slowest air, ground-hugging mist, fireflies warm pinpricks in indigo night; corrupted = stutter in the empire's evenness, oxblood tendrils in pale light, spawned one at a time; combat = brief/sharp/weighted, crit flash = amber (matches 6.6 damage-number flash) with 0 particles + hit-stop, death dissolve = motion signature (beast settles / construct rises in even column / corrupted falls as red ash staining ground).
- **Performance (first draft, validate Section 8)**: per-effect tiers 16–32 / 64–128 / ambient ≤128 / hero ≤512 (State 4 only); scene soft cap 2,000 / hard 3,000, combat 1,500; ≤40 emitters/scene, ≤24 combat; effects ≤8s; **billboard two-orientation decision** — camera-facing for verticals, ground-aligned for horizontals (mist quad facing camera = wall); 3D meshes only for hero; additive blending rationed (empire/fireflies may glow, wild alpha only, no post bloom); one shared soft-sprite atlas.
- **Animation principles**: drift/float/settle — nothing pops, bounces, or moves like a screensaver; motion verbs = biography (rise/fall/hang/drift/settle/breathe); irregularity is life, regularity is the tell (empire metronome, corrupted stutter); settling leaves evidence (ash/blood/scorch persist — Pillar 2); ambient loops imperceptible within 30s except empire's (regularity is the point).
- **Weather (VS contract)**: rain ≤1,000 streaks + 60 ripples (favors present — wet ground reflects lantern pools), fog 8–16 ground sheets (favors empire — raises pale fraction; never above waist height in combat), snow ≤800 flakes with horizontal drift (clue sheet — tracks are testimony); weather never makes the collision, revelation can clear it; per-state dial (near-zero State 2, desk dry).
- **UI particles**: the desk does not breathe — no particle systems in the flat register (2.4); three moments as one-cycle 2D beats: journal dust = none, level-up = amber swell (~1s), trust-increase = portrait amber pulse (per 6.4 pulse vocabulary); world-anchored HUD may breathe at empire tempo as 2D animation; exception = title screen world particles (painting in motion).

## Section 8 Key Decisions

- **Formats**: FBX preferred interchange (OBJ accepted simple, BLEND exported to FBX first); PNG preferred textures (lossless carries the 5.2 grade; TGA legacy); no DCC material files enter the engine — materials = texture sets + authored values, assembled via mapping table. DCC's own formats are source-only. Engine-native formats = technical-artist contract (pending).
- **Naming**: kebab-case `[category]_[name]_[variant]_[size|role].[ext]`; prefixes prop_/env_/char_/tex_/mat_/ui_/vfx_; channel suffixes (_albedo/_normal/_roughness/_metalness/_ao/_emissive) on shared base; _lod0/1/2 suffixes; zero-padded variants; pack origin in pipeline metadata, never in name.
- **Texture tiers**: hero 2048×2048 / standard 1024×1024 / background+tiling 512×512 (tilesets to 4096 atlas) / UI min 2× display resolution; PoT always; tier = ceiling; **empire exception = 16-bit PNG, never down-tiered** (banding protection for near-achromatic value shifts, 2.6); textures reduce before geometry (4.4 order); VRAM scene budget — scenery down-tiers before witnesses.
- **LOD**: characters 3 levels (20k/10k/5k hero; 10k/5k/2.5k NPC — first draft), props 2 (5k/1.5k), small scenery 1 (≤500 + cull), terrain dynamic; thresholds anchored to zoom registers at 7u / 15u midpoints, ±1u hysteresis; 4.4 contract enforced (accent never shifts, silhouette never LODs, reduction order fixed, hostile rim-light readability to LOD2).
- **Export philosophy**: DCC authors appearance (grade authored into albedo — never re-applied, no double-grading; correct color-space tagging), engine delivers (compression, mips/streaming, assembly, LOD/instancing, collision/navmesh); exported once, read many; neutral exports; rig/animation budgets = engine constraints, art contract holds regardless.
- **KitBash3D integration**: packs = quarry, treatment = mason; foundation assets never final content; normalization per pipeline GDD CR-2 (scale 1u=1m door reference, Z-up→Y-up, material mapping tables, scale variance authored not a bug); mapping table = mechanical home of 2.1 color exclusivity; legend filter at import (treat or reject); both packs through same treatment; >80% single-pack validator = mechanical ally of 5.5 no-50/50 rule; pack assets scenery-by-default, witnesses earned by authoring.
- **Merge items marked [Technical-artist values pending — to be merged]**: engine-native formats (8.1), 16-bit texture support + VRAM limits + texel density (8.3), poly budgets + LOD switch + terrain LOD (8.4), compression targets + mip/streaming + draw call budgets + animation constraints + collision mesh requirements (8.5). Resolves with them: 1.3 art-pipeline decision (2.5D vs 3D) and 7.3 particle budget validation.

## Section 9 Key Decisions

- **Purpose**: anti-scope-creep section — each prohibition names the thing never done, the rule it would break (cited), and the true version instead; new visual ideas are judged by "which rule does it keep?"; a rule-breaking addition requires amending this document in writing.
- **The 8 prohibitions**: (1) never sci-fi — legend filter (1.1) bans plasma/neon/holograms/displays/chrome; ban is on first-sight reading, the machine may live under the surface (Principle 3, 2.6); (2) never generic fantasy — no dragons/elves/glowing swords/fireballs/wizard robes; supernatural vocabulary closed to 4.1's four families; investigator = journal + modest blade, not-a-chosen-one (4.5); spells act "the way the world breathes" (7.1); (3) never horror/gore for its own sake — no jump scares/gratuitous gore/body-horror decoration; horror notes rationed (4.1, 7.2, 4.6); blood-on-ivory is always a clue (2.2); (4) never UI outside the flat register — no 3D UI, no diegetic HUD, no skeuomorphic buttons (6.4, 6.3); flat world-anchored HUD remains (6.1) — the ban targets UI that pretends to live inside the painting; (5) never procedural generation — composition is authored (5.6), witness budget is a budget (5.3), moss authored per-asset (5.2); "a seed is not a writer"; (6) never filler combat — no random encounters/respawns/XP grind; respawn contradicts Pillar 2 (world keeps the mark, 5.4) and the finite-violence effects contract (7.2/7.3); (7) never comedy/parody/meta — no self-referential jokes, fourth-wall breaks, ironic detachment; warmth = human moments (State 2, 4.3 idles), not jokes; (8) never pure black or pure white — #000000/#FFFFFF banned everywhere; soot #141210 warm (2.1), bone ivory #E9E6DC (2.6), ink #1C1813 (2.4); pure black = void not unknown, pure white = sterile not sacred; painterly-grade reason (5.2).

## Section 10 Key Decisions

- **Selection rule**: additive — five references, five distinct domains (decay / composition + corruption / scale + warmth + craft / golden register / mythic-tech concept); no two point the same direction; the legend filter applies to references as to assets (take what serves the borderland, leave the rest). Section 10 is the formal home of inline citations (Section 3 warmth anchors, 2.1–2.2, 4.3, 5.4).
- **Per-reference splits**: Dark Souls = decay as narrative + ruined grandeur (Principle 1, 5.4), avoid grimdark grey/opacity/dead register; Diablo 1–2 = isometric crime scene + blood-on-white corrupted sacred space (5.6, 2.2, 2.3 false collision), avoid murk/grime/horror-gore; LOTR = inhuman scale + nature reclaiming stone + warmth against dark + practical costume (5.1, 2.1, 4.1, 4.5), avoid cinematic desaturated grade/heroic framing/fantasy iconography; Drakensang: The River of Time = the golden painterly baseline register (5.2, State 1, warmth directive), avoid cozy safety where danger never lands; One Piece = Void Century concept only — tech read as myth, misunderstanding is the story (Principle 3, 1.1, Pillar 4, 2.6), avoid all style + sci-fi legibility (9.1).

## Section 2b Key Decisions (Shape Language)

- **Placement**: inserted as Section 2b between Section 2 (Color) and Section 3 (Light) — no renumbering of Sections 3–10 (34 "Section 3–9" + 138 "N.M" shorthand references would require a high-risk sweep; user-sanctioned "2b" option). All existing cross-references remain valid.
- **Silhouette is the first read (2b.1)**: identity from form precedes 4.2's affiliation order (accent = whose side) — reconciled, not contradicted (accent needs a figure to attach to). Silhouette test = every figure/landmark/witness identifiable in pure silhouette at 20u (redesigned, not rescued); glance contract = frame parses as grammar (ground/figure/threat/witness). Pillars 3 + 1.
- **Three shape families (2b.2)**: present = angular, weathered, hand-wrought, human scale; empire = curved-perfect, unbroken lines, perfect symmetry, inhuman scale; wild = organic, irregular, patient. Rules: families never mix casually (6.1 register test in form); the unbroken line vs the broken line is the present/empire distinction (continuity = history, Principle 2); **symmetry is the empire's register** — nothing living is symmetric (4.3's small asymmetries; 5.2 clean is alien); fairy-tale balance in form (perfect curve = beautiful + wrong). Pillar 4.
- **Environment geometry (2b.3)**: angularity dominates by area (present = default, warm registers of 2.3); the perfect curve is the event, rationed like ivory (one circle per scene = statement; second = revelation or lie, mirroring 7.2); buried geometry resurfaces as pattern (5.5 — geometry, never color, betrays the plan, Pillar 2); verticality split: present = heavy human verticals, empire = rare vast unbroken verticals (5.6 accents).
- **Hero vs supporting shapes (2b.4)**: the eye is drawn by anomaly before color; hero shapes draw the eye by being wrong — always empire, player, or living (2.1 exclusivity in form); supporting shapes recede by repetition; witnesses sit where shape grammar points (5.6 focal point); player = unique carried profile (lantern/journal/satchel, 4.4 amber figure); **hostiles are never supporting shapes** (shape sibling of 2.5's hostiles-are-always-lit). Pillars 4 + 3.
- **UI shape grammar (2b.5)**: the desk draws with the investigator's hand — no empire geometry in the desk (no perfect-circle chrome, no unbroken-line frames; desk's only circles are "the hand's periods" — dots, knots, pins per 6.3); desk shapes = present's angular hand-wrought family (6.2 typography same rule in letterform); shape carries meaning before color (6.3 glyph rule). Pillar 4.
- **Character silhouettes (2b.6)**: cross-references 4.1–4.4, no duplication — one silhouette per role; thumbnail readability is the isometric contract (Pillar 1); **asymmetry is life, perfect symmetry is the tell** ("the only perfect figure in the game is the thing that was never alive"); hostility survives silhouette to the final LOD (Pillar 3).
- **Header**: "Sections 1–10 complete, plus Section 2b (Shape Language)". Noted during authoring: Section 10 (Reference Direction) authored concurrently — no overlap with shape content.

## Resolution Record (coordinator + ux-designer rulings, 2026-08-09)

- Health bars: faction-fixed hue, umber/soot track, length + numeric readout carry level, pulse outline low-HP, no hue shift
- Flanking arcs: cold indigo (#3E4A63) ratified into UI palette as combat spatial cue; shape + motion distinct from thread-red target rim
- Typewriter: 45–60 cps word-chunk reveal; skip (click/Enter/Space/gamepad); auto-complete on new question; "Instant" accessibility option; 10–15s timeout → Error + Retry; Cancel during Processing
- Resolved marker: ink check on parchment chip — no green, no amber
- Player questions: ink text + amber name-chip (never amber-on-parchment)
- Floating text: soot outline (numerals) or parchment backplate (labels)
- Turn order bar: panel umber, not parchment
- Focus ring: oxblood/umber ≥3:1; tab order + arrow-key chip navigation
- Esc: end conversation in dialogue; pause menu outside
- Resolution: min 1280×720, ref 1920×1080
- Auto-scroll: suspends on manual scroll-up, resumes at bottom
- PENDING ux-designer confirmation: ivory usage (save metadata ink-on-parchment, slider thumb amber/ink), destructive buttons oxblood not thread-red

## Files

- `/home/danila/BG2/design/art/art-bible.md` — Sections 1–10 + Section 2b complete; Section 8 engine values pending technical-artist merge. Header status line updated
- `/home/danila/BG2/design/gdd/camera-system.md` — revised, awaiting re-review

## Open Questions / Next

- Art bible complete (Sections 1–10 + Section 2b) — remaining: Section 8 engine values merge with technical-artist; AD-ART-BIBLE gate sign-off recorded when gate runs
- GDD deltas to apply (owner: ux-designer): dialogue-ui.md (typewriter speed/skip/instant/timeout, Esc, auto-scroll), combat-ui.md (health bars, flank arcs indigo, turn order umber, floating text backplate), investigation-journal.md (resolved = ink check), main-menu-and-settings.md (focus ring, ivory usage, destructive buttons, resolution floors)
- 2.4 follow-up: UI palette table should gain a cold indigo row (ratified in 6.6) when palette section next touched
- Section 8 engine values pending technical-artist merge (marked in subsections); resolves with them: art-pipeline decision 2.5D vs 3D (1.3 production note) and 7.3 particle budget validation
- AD-ART-BIBLE gate sign-off recorded when gate runs (main session)
- Camera re-review: `/clear` then `/design-review design/gdd/camera-system.md`; then update systems-index.md + review-log

<!-- QA RUN: 2026-08-10 | Sprint: sprint-1 | Verdict: APPROVED WITH CONDITIONS | Report: production/qa/qa-signoff-sprint-1-2026-08-10.md -->
<!-- QA RUN: 2026-08-11 | Sprint: sprint-2 | Verdict: APPROVED WITH CONDITIONS | Report: production/qa/qa-signoff-sprint-2-2026-08-11.md -->

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/save-load-system/story-001-save-load-orchestration.md — Save/Load Orchestration
- Agent: engine-programmer (background, ID: aff76f85157534cd3)
- Dependencies verified: EF-002 (Complete), EF-005 (Complete), ADR-0002 (Accepted)
- Manifest versions match: story 2026-08-09 = current 2026-08-09
- Files to create: src/save/save_manager.go, tests/integration/save-load/save_load_test.go
- Blockers: None
- Next: /code-review src/save/ then /story-done production/epics/save-load-system/story-001-save-load-orchestration.md


## Session Extract — /story-done 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/save-load-system/story-001-save-load-orchestration.md — Save/Load Orchestration
- Tech debt logged: None
- Test suite: 28 tests (18 integration + 10 unit), all PASS
- Code review: APPROVED (re-review after fixes)
- Next recommended: Check sprint-3 for next ready story

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/npc-knowledge-graph/story-001-knowledge-domain-fact-model.md — Knowledge Domain & Fact Model
- Agent: ai-programmer (background, ID: ae8dca2b2cd473bde)
- Dependencies verified: EF-001 (Complete), EF-002 (Complete), ADR-0005 (Accepted), ADR-0001 (Accepted)
- Manifest versions match: story 2026-08-09 = current 2026-08-09
- Files created: src/npc_knowledge_graph/types.go, tests/unit/npc_knowledge_graph/knowledge_graph_test.go
- Test results: 7/7 PASS, go vet clean
- Duplicate resolved: removed src/kg/ and tests/unit/kg/ (parallel agent, truncated data model)
- Blockers: None
- Next: /code-review src/npc_knowledge_graph/ then /story-done production/epics/npc-knowledge-graph/story-001-knowledge-domain-fact-model.md

## Session Extract — /story-done 2026-08-11
- Verdict: COMPLETE
- Story: production/epics/npc-knowledge-graph/story-001-knowledge-domain-fact-model.md — Knowledge Domain & Fact Model
- Tech debt logged: None
- Code review: APPROVED WITH SUGGESTIONS (lead-programmer + qa-tester)
- Test suite: 7 tests, all PASS
- Next recommended: KG-002 (YAML .kg File Parser) — now unblocked

## Session Extract — /dev-story C&I UI 005 2026-08-11
- Story: production/epics/character-inventory-ui/story-005-inventory-grid-weight-bar.md — Inventory Grid & Weight Bar
- Agent: ui-programmer (this session)
- Files created: src/character_sheet/inventory.go, tests/unit/character_sheet/inventory_test.go
- Files modified: src/character_sheet/equipment.go (Item += Description/Weight/Value/Quantity + 3 type prefix consts), src/character_sheet/sheet.go (inventory tab wiring: inventory field, showTab/setTabActive/setShown, bindElements, populate, Update, SetReducedMotion, inventoryFuncMap merge), assets/ui/character_sheet.html (tab bar + inv-panel markup + CSS, id on sheet-cols/equip-section)
- Design decisions: (1) Item component extended with additive fields — required data source for weight bar/detail/stack counts (flagged deviation); (2) AC-10 (95% pulses) vs GDD threshold table (>100% pulses) reconciled via WeightColor (bands per GDD) + WeightPulse (≥90% per AC-10/QA AC-12); (3) dynamic grid cells via document.DuplicateElementToParent from hidden #inv-cell-tpl
- Test results: 12 new unit tests PASS (AC-10/11/12/14 + thresholds + gold + filter/sort/weight), full suite 53 PASS, go build + go vet clean (src + tests incl. integration vet)
- Not verified: in-engine rendering (screenshot evidence pending — Kaiju UI risk HIGH: duplicated-cell click binding, Input onchange/onkeyup, overflow:scroll, grid-template-columns)
- Blockers: None
- Next: /code-review src/character_sheet/inventory.go then /story-done story-005; note Item component change needs pod serialization sanity check for old saves (ADR-0002)

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/main-menu-settings/story-001-navigation-transition-framework.md — Navigation Accessibility & Screen Transition Framework
- Agent: ui-programmer (background — agentId: a4571d804eb25a31d)
- Files expected: src/main_menu/ (screen_flow.go, focus_manager.go, transition_system.go, etc.), tests/unit/main_menu/navigation_transitions_test.go
- Blockers: None
- Next: /code-review src/main_menu/ then /story-done story-001

## Session Extract — /story-done 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/main-menu-settings/story-001-navigation-transition-framework.md — Navigation Accessibility & Screen Transition Framework
- Implementation: src/ui/mainmenu/ (8 files), tests/unit/main_menu/navigation_transitions_test.go (39 tests)
- Code review: APPROVED (2 bugs fixed, Dispatch refactored)
- Tech debt logged: None
- Advisory: Manual evidence file deferred until Story 002
- Next recommended: Story 002 — Title Screen & New Game Flow (now unblocked)

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/main-menu-settings/story-002-title-screen-new-game-flow.md — Title Screen & New Game Flow
- Agent: ui-programmer (background — agentId: aa4c37063e937560b)
- Files expected: src/ui/mainmenu/title_screen.go, tests/unit/main_menu/title_screen_test.go
- Depends on: Story 001 (Complete ✅) — builds on ScreenController, FocusManager, TransitionSystem
- Blockers: None
- Next: /code-review src/ui/mainmenu/title_screen.go then /story-done story-002

## Session Extract — /story-done 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/main-menu-settings/story-002-title-screen-new-game-flow.md — Title Screen & New Game Flow
- Implementation: src/ui/mainmenu/title_screen.go (602 lines), tests/unit/main_menu/title_screen_test.go (590 lines, 18 tests)
- Code review: APPROVED (no required changes)
- Tech debt logged: None
- Advisory: Manual evidence file deferred (same as Story 001)
- Next recommended: Story 003 — First-Launch Accessibility Prompt (now unblocked)

## Session Extract — /dev-story 2026-08-11
- Story: production/epics/main-menu-settings/story-003-first-launch-accessibility-prompt.md — First-Launch Accessibility Prompt
- Agent: ui-programmer (background — agentId: a05038d41835c618e)
- Files expected: src/ui/mainmenu/accessibility_prompt.go, tests/unit/main_menu/accessibility_prompt_test.go
- Depends on: Story 001 (Complete ✅), Story 002 (Complete ✅) — uses SetInteractive hook
- Blockers: None
- Next: /code-review src/ui/mainmenu/accessibility_prompt.go then /story-done story-003

## Session Extract — /story-done 2026-08-11
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/main-menu-settings/story-003-first-launch-accessibility-prompt.md — First-Launch Accessibility Prompt
- Implementation: src/ui/mainmenu/accessibility_prompt.go (512 lines), src/ui/mainmenu/screen_flow.go (refactored), tests/unit/main_menu/accessibility_prompt_test.go (13 tests)
- Code review: APPROVED (no required changes)
- Tech debt logged: None
- Advisory: Manual evidence file deferred (same as Stories 001-002)
- Next recommended: Story 004 — Save/Load Screens (Integration, depends on Story 001 ✅ + Save/Load System)

## Session Extract — /dev-story 2026-08-12
- Story: production/epics/main-menu-settings/story-004-save-load-screens.md — Save/Load Screens
- Agent: ui-programmer (background — agentId: abc5a37b37201fa4a)
- Type: Integration (BLOCKING test requirement)
- Files expected: src/ui/mainmenu/save_load_screen.go, tests/integration/main_menu/save_load_screens_test.go
- Depends on: Story 001 (Complete ✅), Save/Load System (external — stubbed data contract)
- Blockers: None
- Next: /code-review src/ui/mainmenu/save_load_screen.go then /story-done story-004

## Session Extract — /architecture-review 2026-08-12 (Review #2)
- Verdict: CONCERNS
- Requirements: 263 total — 222 covered, 0 partial, 41 gaps (incl. 17 gated behind Proposed ADRs)
- New TR-IDs registered: 0 (registry already complete)
- Registry updates: TR-ai-006..011/013 → ADR-0008 (revision verified in body); TR-save-009 conflict resolved (ADR-0002 amended); Audio/Loot TRs → ADR-0014/0015 (Proposed)
- GDD revision flags: save-load-system.md CR-4 (camera persistence — ADR amended, GDD pending), ai-dialogue-engine.md (registry synced)
- Top ADR gaps: Party System ADR (6 TRs), Faction System ADR (3 TRs), Map System ADR (2 TRs)
- Report: docs/architecture/architecture-review-2026-08-12b.md

## Session Extract — /dev-story 2026-08-12
- Story: production/epics/persistent-world-state/story-004-clue-path-protection.md — Clue-Path Protection
- Agent: gameplay-programmer (background, resumed 3× for gap-fix: missing deliverables, then 3 failing tests)
- Type: Integration (BLOCKING test requirement)
- Files changed: src/persistent_world_state/clue_path_protection.go (created), src/persistent_world_state/flag_store.go (tombstone storage + EntityRegion getter), src/persistent_world_state/event_pipeline.go (RecordDestructionEvent), tests/integration/persistent_world_state/clue_path_test.go (created, 17 tests)
- Design deviations (user-approved via AskUserQuestion): PWS-owned quest-relevance registry (not Persistent-component read — EF purges components before entity.destroyed); tombstone flag storage for destroyed entities (Story 006 will serialize); RecordDestructionEvent as destroyed-target pipeline path; clue_path_protection entries marked Protected:true
- Tests: 65/65 pass (33 integration + 32 unit PWS); src go build clean. Pre-existing unrelated failures exist in tests/unit/entity (flaky) and tests/integration/main_menu (keybindings) — neither imports persistent_world_state
- Blockers: None
- Next: /code-review src/persistent_world_state/clue_path_protection.go src/persistent_world_state/flag_store.go src/persistent_world_state/event_pipeline.go then /story-done production/epics/persistent-world-state/story-004-clue-path-protection.md

## Session Extract — /test-setup 2026-08-12
- Verdict: COMPLETE — infrastructure already existed (scaffolded 2026-08-09); verification + hardening pass
- Fixed: .github/workflows/test.yml missing CGO_ENABLED: 1 (tests import kaijuengine.com — CI would fail); tests/README.md CI path corrected (tests.yml → test.yml)
- Pre-gate checklist now fully green: tests/unit+integration ✅, CI workflow ✅, accessibility-requirements ✅, interaction-patterns ✅

## Session Extract — /clear mid-Story-007 2026-08-12
- Epic: main-menu-settings — Stories 001-006 COMPLETE, 007 in progress
- Story 007: Keybindings & Input Remapping — ui-programmer agent (agentId: ad316c9261196527d) resumed in background, mid-debug
- Last known state: tests failing on focus ID mismatch (test hardcoded "kb-row-move-forward-primary" with hyphens; impl registers "kb-row-move_forward-primary" with underscores) + ParseChord key-name validation issue
- Files: src/ui/mainmenu/keybindings_screen.go exists; tests/integration/main_menu/keybindings_test.go exists (24 tests, failing)
- Modifications pending verification: binding_registry.go (BindSlot/ClearSlot/BindingCount), screen_flow.go (ScreenKeybindings state), settings_screen.go (ControlNavigate + placeholder removal)
- Test status at last check: unit 70+ PASS, integration keybindings 20+ FAIL
- Next: wait for agent completion notification, verify tests pass, then /code-review + /story-done for story 007 (final epic story)

## Session Extract — ADR acceptance + Party ADR 2026-08-12
- ADR-0014 (Audio System): Proposed → Accepted (verified: all 13 sections present)
- ADR-0015 (Loot & Economy): Proposed → Accepted (verified: all 13 sections present)
- ADR-0016 (Party System): CREATED (Proposed) — covers TR-party-001 through TR-party-007
  - Resolves design gaps T0-2 (interjection interface → ADR-0008 amendment required) and T0-3 (knockout state → ADR-0009 amendment required)
- TR registry v4: party TRs → ADR-0016 (Proposed); (Proposed) markers removed from audio/loot/combat entries
- Coverage now: 247/263 (93.9%) covered by Accepted ADRs, 6 partial (ADR-0016 Proposed), 10 gaps
- Remaining gaps: PWS (2), Camera 3D audio (1), Stats cross-system (1), AI-019 open question (1), Faction (3), Map (2)

## Session Extract — /story-done 2026-08-12
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/persistent-world-state/story-004-clue-path-protection.md — Clue-Path Protection
- Tech debt logged: 4 items (reverse-index staleness, GetEntries since=0, silent region-miss, QA coverage gaps) → docs/tech-debt-register.md
- Code review: APPROVED WITH SUGGESTIONS (W1–W3 + 5 QA gaps)
- Next recommended: production/epics/persistent-world-state/story-005-world-evolution-rules.md (Ready)

<!-- QA-PLAN: 2026-08-13 | System: sprint-6 | Plan written: production/qa/qa-plan-sprint-6-2026-08-13.md -->

## Session Extract — /dev-story MMS-007 2026-08-13
- Story: production/epics/main-menu-settings/story-007-keybindings-input-remapping.md — Keybindings & Input Remapping (finish of in-flight work)
- Verdict: implementation verified — make test-all green (30/30 packages), go build + go vet clean in ./src
- Root cause: focus IDs registered snake_case (kb-row-move_forward-primary) vs kebab-case click contract (kb-row-move-forward-primary) → FocusManager.ActivateAt never matched; plus ParseChord accepted "???" as valid key (AC-8)
- Files changed: src/ui/mainmenu/keybindings_screen.go (bindingFocusID() kebab mapping + ParseChord validKeyName() tightening); tests/integration/main_menu/keybindings_test.go (2 test fixes: label map key direction, gamepad preset test missing navigateToKeybindings)
- Tests: 24/24 pass in main_menu integration; AC-1..AC-8 all mapped
- Agent note: story cites TR-ui-007 (accessibility) but keybinding requirement text lives in TR-menu-005 — registry re-reference follow-up suggested
- Next: /code-review keybindings files then /story-done production/epics/main-menu-settings/story-007-keybindings-input-remapping.md

## Session Extract — /code-review MMS-007 2026-08-13
- Review: 3 specialists (lead-programmer, ux-designer, qa-tester) — initial verdict CHANGES REQUIRED
- Blocking fixed (2 rounds): (1) restore path conflict resolution incl. loop-clear both slots + AC-6 gate + fixpoint restore-all; (2) "Unbound" persistence token (SettingUnboundValue) so cleared slots survive reboot; (3) reserved-key dialog keyboard routing (consume only Escape); + blockedMsg clearing, validKeyName spaces, gofmt, zz_debug_test.go deleted
- Tests: 34 pass in main_menu integration (-count=1 green); 5 new pinning tests + 1 discriminating replace + 2 updates
- Re-review verdict: APPROVED WITH SUGGESTIONS — suggestions logged for /story-done (amber text vs manifest, arrow-nav grid, FocusID suffix contract, focus jump, motion pulse, preset cycling, IBS scope overlap, sticky modifiers)
- Pre-existing flake surfaced: tests/unit/entity TestSpatialQueryPerformance fails under full sweep only (passes standalone x2) — separate tech debt
- Next: /story-done production/epics/main-menu-settings/story-007-keybindings-input-remapping.md

## Session Extract — /story-done MMS-007 2026-08-13
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/main-menu-settings/story-007-keybindings-input-remapping.md — Keybindings & Input Remapping
- Criteria: 8/8 auto-verified; 34 integration tests green; code review APPROVED WITH SUGGESTIONS (user-confirmed)
- Tech debt logged: 12 items (TR-menu-005 re-reference, amber-text vs manifest, IBS scope overlap, render contract, arrow-nav grid, focus jump, motion pulse, preset cycling, sticky modifiers, Bind/Unbind slot model, HandleKey adapter contract, entity perf flake)
- Sprint 6 must-haves remaining: DIALOGUE-UI-001 (in-progress), PWS-005 (ready), PARTY-001 (ready), PRC-001 (ADR-0016 acceptance)
- Next recommended: DIALOGUE-UI-001 finish or PARTY-001 via /dev-story

## Session Extract — /story-done PARTY-001 2026-08-13
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/party-system/story-001-party-roster-active-slots.md — Party Roster & Active Slots
- Criteria: 5/5 + edge auto-verified; 42 tests green; code review APPROVED WITH SUGGESTIONS (self-swap guard + 10 test items fixed)
- Tech debt logged: 6 items (ADR-0016 conflict → PRC-001, save-snapshot wiring → INT-6/7, dead config fields, AddToPool existence check, Swap reorder event, capacity-not-serialized)
- Also fixed this session: perf-flake thresholds (entity + ai_dialogue) — full sweep now deterministic 37/37; unused import in game.go removed
- Integration pass: INT-5..INT-10 planned, story files for INT-5..8 written, sprint-6.md + yaml updated
- Next: INT-5 (main-menu-settings/story-008-game-layer-wiring.md) — /dev-story

## Session Extract — пак озвучки 2026-11-04
- Задача: пользователь попросил диалоги между главными персонажами для самостоятельной записи озвучки
- Обнаружено: арт-библия уже готова (AD-ART-BIBLE одобрен 2026-08-10); папки `design/narrative/` не было; именованных игрока/спутников нет — использован существующий пограничный каст из `.kg`-файлов
- Результат: `design/narrative/dialogue/vo-scenes/` — 3 сцены / 72 реплики / ~6 минут, Век + Томас + Мира, цепочка расследования (руда → затопление → плита → Первый Кузнец); v1.1 — русская версия
  - `investigation-chain-vo-pack.md` — сценарий, связь с каноном, гид для одного исполнителя (постановка голоса, порядок записи, чек-лист непрерывности)
  - `lines-vek.md` / `lines-tomas.md` / `lines-mira.md` — тексты по персонажам с подсказками (23/24/25 реплик)
  - `recording-manifest.csv` — 72 строки: VO ID, имена файлов, подсказки, ключевые реплики, колонка `дубль_кипер`
  - `teleprompter.html` — автономный телепромптер: стрелки/пробел, фильтры 1–3, подсказка `c`, полный экран `f`, размер текста +/−
  - `audacity-recording-checklist.md` — пошаговая запись в Audacity: настройки, метки, Export Multiple, сверка с манифестом, типичные проблемы
- Якоря канона: `vek_ore_gone_wrong`, `vek_miner_gone_quiet`, `vek_forgemaster_oath`, `tomas_flood_not_rain`, `tomas_slab_hidden`, `tomas_lied_to_smith`, `mira_script_two_purposes`, `mira_smelting_formula`, `mira_stone_still_warm`
- Статус: одобрено к записи; конфликтов с каноном `.kg` нет; пользователь озвучивает все три роли
- Далее: запись (~2–2.5 часа). Возможные продолжения: интеграция озвучки в диалоговый UI, новые сцены/каст (Халла, Грета, Орин, Марен, Верен), оформление нарративных документов

## Session Extract — озвучка: запись и сборка 2026-11-05
- Записаны все реплики S01+S02: Век 11 · Томас S01 11 · Томас S02 13 · Мира S02 13 (всего 48)
- Источник: `~/vo/marrow/2026-11-04/marrow-vo.aup3` (Audacity, проект открыт во время обработки — снимок делался через SQLite backup)
- **Ключевой урок извлечения AUP3:** старт клипа хранится в поле `0a 23` **перед** именем клипа. Чтение поля после имени даёт старт следующего клипа — все реплики сдвигаются на одну. Исправлено и проверено по меткам/длительностям.
- Обработка: 48 кГц / 24 бит / моно; high-pass 80 Гц; afftdn 10 дБ / −40 dBFS; обрезка только ведущей тишины 150 мс; концы фраз полностью сохранены; фейды 5/10 мс; двухпроходный loudnorm −18 LUFS / TP −2 dBTP
- Концевые щелчки мыши (10–15 мс в последних 250 мс) удалены кроссфейдом; повторная проверка — 0 остатков
- Файлы: `~/vo/marrow/2026-11-04/{vek-s01,tomas-s01,tomas-s02,mira-s02}-{raw,processed}/`; в `*-raw/` исходники со щелчками (архив)
- Диалоги: `~/vo/marrow/2026-11-04/dialogues/` — `s01-dialogue.wav` 2:06, `s02-dialogue.wav` 2:21, `marrow-s01-s02-dialogue.wav` 4:28 + `*-manifest.txt`
- Отчёты: `mira-s02-processing-report.md`, `s01-s02-processing-report.md`
- Клиппинг в исходниках (переписать при возможности): `S02-TOMAS-014` 0.0 dBFS, `S01-TOMAS-010` −1.3, `S01-TOMAS-018` −1.5, `S02-TOMAS-004` −1.6, `S01-TOMAS-020` −2.4, `S01-TOMAS-004` −2.6
- Не закоммичено. Аудио вне репозитория; текстовый пак — `design/narrative/dialogue/vo-scenes/`
- Далее: интеграция озвучки в диалоговый UI, запись остальных персонажей/сцен, при необходимости — коммит VO-пака

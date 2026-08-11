# Session State

*Updated: 2026-08-11*

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

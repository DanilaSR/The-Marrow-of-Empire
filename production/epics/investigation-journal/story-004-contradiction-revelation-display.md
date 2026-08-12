# Story 004: Contradiction Display & Revelation Prompts

> **Epic**: Investigation Journal
> **Status**: Complete
> **Layer**: Presentation
> **Type**: Integration
> **Estimate**: 4-5 hours
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-12

## Context

**GDD**: `design/gdd/investigation-journal.md`
**Requirement**: `TR-invj-004`
*(Contradiction display: contradictory facts shown side-by-side with thread-red connector line and auto-created resolution thread. Revelation prompts: 3+ connected facts → amber glow on chain entry + prompt text, dismissible. TR-ID pending formal registration via /architecture-review.)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered in screen space via Kaiju HTML/CSS-like UI. Desk register parchment panels. Motion: thread draws itself (connector line animation), amber pulses rationed (reward signal). Paper-slide motion, no bounce/scale/spin/flip.

**Secondary ADRs**: ADR-0010 (Thread Investigation — contradiction detection and revelation threshold logic)

**Engine**: Kaiju Engine 26.07.29.1-nightly | **Risk**: HIGH
**Engine Notes**: Kaiju UI animation for connector line drawing and amber pulse glow. Post-cutoff APIs: CSS-like animation triggers, dynamic element insertion for connector rendering, glow/opacity animation. Verify against `kaiju/src/`.

**Control Manifest Rules (Presentation layer)**:
- Required: Kaiju HTML/CSS-like UI, screen space only; Paper-slide motion (no bounce/scale/spin/flip); Desk never trembles; Never encode game-critical info by hue alone (contradiction has both thread-red color AND connector line shape; revelation has both amber glow AND prompt text)
- Forbidden: Never amber-on-parchment text (amber glow is background/fill effect on chain entry, NOT amber text; prompt text is ink on parchment); Never 3D UI or diegetic HUD
- Guardrail: No frame drops during connector line animation or amber pulse

---

## Acceptance Criteria

*From GDD `design/gdd/investigation-journal.md`, scoped to this story:*

- [ ] **AC-1 (GDD AC-3)**: GIVEN two contradictory facts, WHEN the player views the chain detail (Facts tab), THEN the contradictory facts are displayed side by side with a thread-red connector line drawn between them and an auto-created resolution thread appears in the chain
- [ ] **AC-2 (GDD AC-2)**: GIVEN a chain with 3 connected facts pointing to the same target, WHEN the third fact is discovered, THEN the chain entry glows amber in the sidebar and a revelation prompt appears: "You have evidence linking [conclusion]. Confront [target NPC]?"
- [ ] **AC-3**: Revelation prompt is dismissible — player can dismiss it (continue investigating) or act on it (trigger confrontation). Dismissed prompts reappear when the journal is next opened
- [ ] **AC-4**: Contradiction connector line animates — draws from first fact to second fact (thread-red #C4483A line, ~500ms draw animation). Contradiction display shows both facts with source NPC names and the "↕ CONTRADICTS" label between them
- [ ] **AC-5**: Amber glow on revelation-ready chains is a subtle pulsing animation (gentle, not flashing — ~2s period, opacity oscillates between baseline and +15%). The glow stops when the revelation is acted upon or the chain is resolved
- [ ] **AC-6**: Audio cues fire for contradiction detection (sharp string accent — single tense note) and revelation threshold reached (rising chime — soft, expectant). Audio is triggered by events, not rendered by the journal itself — journal subscribes to Thread Investigation events

---

## Implementation Notes

*Derived from ADR-0012 and ADR-0010 Implementation Guidelines:*

**Contradiction detection flow**: Thread Investigation (ADR-0010) already handles contradiction detection — when `OnFactDiscovered` fires and a contradiction is found, a `ContradictionDetected` event is emitted. The journal subscribes to this event. When received, the journal: (1) updates the fact log to show the contradiction marker (Story 003's marker), (2) if the player is currently viewing the relevant chain's Facts tab, renders the side-by-side contradiction view with animated connector.

**Side-by-side contradiction display (CR-5)**: When viewing a chain's Facts tab and two facts have a contradiction relationship, render them as paired entries:

```
[NPC A name] says: "[fact A content]"
   ↕ CONTRADICTS
[NPC B name] says: "[fact B content]"
```

Layout: two fact cards side by side (or stacked on narrow layouts). Cards have a subtle thread-red (#C4483A) left border (2px). Between them, a vertical connector line (thread-red, 2px) draws from the bottom of the top card to the top of the bottom card — animate this line drawing over ~500ms using a CSS-like stroke-dashoffset or Go-driven line-reveal. The "↕ CONTRADICTS" label sits at the midpoint of the connector line, in thread-red, 12px, all-caps.

**Auto-created resolution thread**: When a contradiction is detected, Thread Investigation creates a resolution thread automatically (THREAD-002). The journal displays this thread in the chain's thread list with description "Resolve contradiction: [topic]" and an amber dot status indicator. This is standard thread display (Story 002) — the journal just needs to ensure the new thread appears without requiring a journal close/reopen. Subscribe to `ThreadCreated` events and update the thread list.

**Revelation threshold (CR-6)**: Thread Investigation tracks `revelation_ready` status on chains (THREAD-002: 3+ connected facts → chain status becomes `revelation_ready`). Journal subscribes to `RevelationReady` events. When received:
1. Chain entry in sidebar gets amber glow effect — CSS-like `box-shadow` or background gradient in amber (#D9A441), pulsing with ~2s period (opacity: baseline → baseline+15% → baseline). Use the gentle "breathing" pulse, not a rapid flash.
2. A revelation prompt panel appears at the top of the chain's detail view (or as a banner across the journal): "You have evidence linking [conclusion]. Confront [target NPC]?" — ink (#1C1813) on slightly-warm parchment, with two action chips: "Confront" (amber border — the call to action) and "Continue Investigating" (umber border — the dismiss).

**Prompt dismissal**: "Continue Investigating" dismisses the prompt for the current journal session. The dismissed state is stored in a session-local map (not persisted — it resets when the journal closes). When the journal is next opened, if the chain is still `revelation_ready`, the prompt reappears. "Confront" triggers the confrontation flow — emit a `RevelationActed` event (which Thread Investigation and Dialogue systems consume). The prompt is permanently dismissed for this chain after acting.

**Amber glow rules**: Amber (#D9A441) is used as a glow/fill effect on chain entries, never as text. The glow is subtle — it implies "look here" without screaming. It stops when: (a) player acts on revelation, (b) chain status changes to `resolved`, or (c) new contradictory evidence invalidates the revelation (chain status reverts to `active`).

**Audio integration**: Subscribe to `ContradictionDetected` → trigger sharp string accent SFX. Subscribe to `RevelationReady` → trigger rising chime SFX. Audio triggers are one-shot (play once, don't loop). Use the Audio System's event interface (Story 001 already complete). Audio files: `assets/audio/ui/contradiction_detected.wav`, `assets/audio/ui/revelation_chime.wav`.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Basic chain list and thread display (this story adds glow and contradiction formatting on top)
- Story 003: Fact log with contradiction flags, entity chips
- Story 005: Map centering on thread click
- Confrontation dialogue flow: emitting the event is in scope; the actual dialogue system response to `RevelationActed` is owned by the Dialogue UI epic
- Thread Investigation contradiction detection logic: already implemented (THREAD-002) — journal only displays results

---

## QA Test Cases

**Automated tests (Integration story — must have test file):**

- **AC-1**: Contradiction side-by-side display
  - Given: Two facts in the same chain with a contradiction relationship (fact A contradicts fact B). Player is viewing the chain's Facts tab
  - When: The journal renders the fact log for this chain
  - Then: Facts A and B are displayed as paired entries with a thread-red connector line between them. The connector line has a "↕ CONTRADICTS" label. An auto-created resolution thread appears in the chain's thread list with description containing the contradiction topic
  - Edge cases: Single fact with no contradictions (no connector rendered); 3+ mutually contradictory facts (pairwise connectors between each pair); contradiction in a different chain (no connector shown when viewing unrelated chain)

- **AC-2**: Revelation prompt on threshold
  - Given: A chain has 2 connected facts pointing to the same target NPC
  - When: A 3rd fact is discovered that points to the same target (revelation threshold reached)
  - Then: The chain entry in the sidebar glows amber (pulsing animation). A revelation prompt appears showing the conclusion and target NPC name. Prompt includes "Confront" and "Continue Investigating" actions
  - Edge cases: 4+ facts (threshold already reached — no duplicate prompt); facts pointing to different targets (no revelation until 3 share a target); chain already resolved (no prompt)

- **AC-3**: Prompt dismissal and re-appearance
  - Given: A revelation prompt is visible
  - When: Player clicks "Continue Investigating"
  - Then: Prompt dismisses. Chain entry still glows amber (revelation still available). Player closes journal. Player re-opens journal — prompt reappears
  - Edge cases: Player clicks "Confront" (prompt permanently dismissed, glow stops); chain becomes resolved before re-open (prompt does not reappear)

- **AC-4**: Connector line animation
  - Given: Contradiction side-by-side view is triggered
  - When: The connector line renders
  - Then: Line draws from top fact to bottom fact over ~500ms (animated reveal). Line color is thread-red (#C4483A). Animation does not cause frame drops
  - Edge cases: Player switches tabs during animation (animation completes or cancels cleanly); journal closed during animation (no crash, no dangling animation timer)

- **AC-5**: Amber glow animation
  - Given: Chain entry has amber glow active
  - When: Observe the glow over several seconds
  - Then: Glow pulses with ~2s period, opacity oscillates between baseline and +15%. Animation is smooth (no jitter). Glow stops when revelation acted upon or chain resolved
  - Edge cases: Multiple chains reach revelation simultaneously (all glow independently); player acts on one revelation (only that chain's glow stops)

- **AC-6**: Audio cue triggers
  - Given: Contradiction event is emitted by Thread Investigation
  - When: Journal receives the event
  - Then: Sharp string accent SFX plays once. Given: RevelationReady event is emitted. When: Journal receives the event. Then: Rising chime SFX plays once
  - Edge cases: Audio system not initialized (graceful no-op, no crash); multiple contradictions in rapid succession (each plays its own cue); journal closed when event fires (cue still plays — audio is ambient, not journal-scoped)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/investigation_journal/contradiction_revelation_test.go` — must exist and pass
**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 002 (Chain List & Thread Detail), Story 003 (Fact Log & Entity Quick-Ref), Thread Investigation Story 001 (Complete ✅), Thread Investigation Story 002 (Complete ✅), Audio System Story 001 (Complete ✅)
- Unlocks: None (final investigative feature story)

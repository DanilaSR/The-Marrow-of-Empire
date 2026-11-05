# Story 003: Conversation Log & Typewriter Display

> **Epic**: Dialogue UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 4h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/dialogue-ui.md`
**Requirement**: `TR-ui-001`, `TR-ui-004`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: All UI rendered through Kaiju's HTML/CSS-like UI system. Typewriter text reveal uses "ink appears" motion language — characters appear sequentially at 45-60 cps for journal, with the dialogue variant running at 20 chars/sec (designer-tuned for readability of NPC speech). Typewriter speed is an accessibility setting: Instant / Normal / Slow. Paper-slide motion only — no bounce, scale, spin, or flip.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI system — verify DOM-safe chunked text updates (incremental character appending), animation timer API, click event handling for skip-to-end. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Desk serif (Garamond class) for dialogue text; paper-slide motion: ink appears (typewriter 45-60 cps for journal — dialogue variant 20 cps); typewriter speed accessibility setting; desk never trembles (immune to camera shake)
- Forbidden: Never use amber-on-parchment text (amber is accent/fill only — NPC response text must be dark umber on parchment); never use bounce/scale/spin/flip motion
- Guardrail: Journal open/close transition 200-300ms; typewriter text: DOM-safe chunk updates; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/dialogue-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN a conversation with message history, WHEN the dialogue panel is visible, THEN the conversation log displays all messages in a scrolling area — player questions aligned amber-left, NPC responses aligned parchment-right — with the most recent message at the bottom and auto-scrolling on new messages.
- [ ] **AC-2**: GIVEN an NPC response is received from the AI Dialogue Engine, WHEN the response begins rendering, THEN text appears character by character at 20 chars/sec (typewriter effect) with a subtle soft tick sound per character, and the panel is in Response state.
- [ ] **AC-3**: GIVEN typewriter text is actively revealing, WHEN the player clicks anywhere on the response text or presses a skip key, THEN the full response text appears instantly and the soft tick sound stops.
- [ ] **AC-4**: GIVEN the AI Dialogue Engine is processing a question (Processing state), WHEN the panel is waiting for a response, THEN a thinking indicator shows three dots pulsing sequentially (0.5s per dot) next to the NPC portrait.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Conversation log container**: Scrolling div (overflow-y: auto or Kaiju equivalent) in the center section of the dialogue panel. Flex column layout, padding for readability. Messages appended as child elements. Auto-scroll: on each new message, scroll the container to bottom via `scrollTop = scrollHeight` or Kaiju equivalent.
- **Message alignment**: Player questions: `text-align: left` or `align-self: flex-start`, tinted amber (as accent, not the full text — text remains dark umber on parchment). NPC responses: `text-align: right` or `align-self: flex-end`, parchment tone. Each message is a distinct block with clear visual separation. Use Garamond-class serif for all dialogue text.
- **Message data model**: Each message in the log has `{speaker: "player"|"npc_id", text: string, timestamp: game_time}`. The conversation history is maintained by the AI Dialogue Engine; the UI subscribes to `NPCMessageReceived` and `PlayerMessageSent` events via EventBus to append to the log.
- **Typewriter effect**: Implement as a Go ticker or animation frame callback that appends characters to the NPC response text element at 20 chars/sec. Each character append triggers a soft tick audio event (very subtle — should not be fatiguing). The typewriter runs during the Response state.
  - **Chunk safety**: Do not append one character per DOM mutation — batch updates every ~50ms (roughly 1 character at 20 cps, or batch 2-3 chars every 100ms for safety) to stay within Kaiju's UI render budget (<1ms/frame).
  - **Accessibility**: Respect the typewriter speed setting. "Instant" → show full text immediately (no animation). "Normal" → 20 chars/sec. "Slow" → ~10 chars/sec. Read the setting from player preferences at response start.
- **Click-to-skip**: Attach a click handler to the response text area. On click during typewriter reveal: cancel the ticker, set the text element's content to the full response string, stop the tick sound. Also bind a keyboard shortcut (Space or Escape — but Esc is already mapped to end conversation; use Space or a dedicated skip key). The skip must not interfere with the End Conversation Esc mapping from Story 001.
- **Thinking indicator**: Three `<span>` dots or Kaiju-equivalent elements next to the NPC portrait in the top bar. CSS animation: each dot pulses opacity (0 → 1 → 0) with a 0.5s stagger. Dot 1 at 0ms, Dot 2 at 500ms, Dot 3 at 1000ms, looping. Show during Processing state, hide when Response state begins. The indicator also appears in the response area as "..." text while waiting.
- **Performance**: Log history can grow long. Cap visible messages at the last 50 exchanges; older messages scroll off the top but remain in the DOM (hidden) for scroll-back. If performance testing shows DOM bloat, implement virtual scrolling (render only visible + 5 messages above/below).

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Panel framework, show/hide lifecycle, state machine, NPC portrait, End Conversation button, Esc key
- Story 002: Text input field, topic suggestion chips, submit flow
- Story 004: Trust level amber border glow, degraded mode indicator (the degraded mode *display* of plain text is in Story 004 — this story handles normal typewriter rendering)

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (scrolling conversation log with alignment):
  - Setup: Have a conversation with 5+ exchanges (questions + responses)
  - Verify: All messages visible in the log area. Player messages aligned left (amber-tinted). NPC messages aligned right (parchment tone). Newest message at bottom. Log auto-scrolls to show newest message.
  - Pass condition: Messages correctly aligned and colored per speaker, auto-scroll works, log is scrollable

- **AC-2** (typewriter text reveal at 20 chars/sec):
  - Setup: Submit a question that generates a 100-character NPC response
  - Verify: Characters appear sequentially. At 20 cps, a 100-char response takes ~5 seconds to fully reveal. Soft tick audible per character (subtle, not fatiguing). Panel is in Response state during reveal.
  - Pass condition: Full response revealed at correct speed, tick sounds play, state is Response

- **AC-3** (click-to-skip typewriter):
  - Setup: During an active typewriter reveal (mid-response)
  - Verify: Click on the response text → full text appears instantly, tick sounds stop immediately. Response area shows complete text.
  - Pass condition: Full text visible instantly after click, no residual animation or sound

- **AC-4** (thinking indicator during Processing):
  - Setup: Submit a question, observe the UI during Processing state (before response arrives)
  - Verify: Three dots next to NPC portrait pulse sequentially — dot 1 fades in/out starting at 0ms, dot 2 at 500ms, dot 3 at 1000ms. Pattern repeats until response begins.
  - Pass condition: Dots visible and animating during Processing, hidden when Response starts

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/dialogue-ui-log-typewriter-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Dialogue Panel Framework — state machine and panel layout must exist), Story 002 (Text Input & Topic Chips — submit flow triggers the response pipeline that populates the log)
- Unlocks: Story 004 (Trust Indicator & Degraded Mode — degraded mode overrides the typewriter behaviour implemented here)

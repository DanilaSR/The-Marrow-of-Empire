# Story 004: Trust Indicator & Degraded Mode

> **Epic**: Dialogue UI
> **Status**: Ready
> **Layer**: Presentation
> **Type**: UI
> **Estimate**: 2h
> **Manifest Version**: 2026-08-09
> **Last Updated**: [set by /dev-story when implementation begins]

## Context

**GDD**: `design/gdd/dialogue-ui.md`
**Requirement**: `TR-ui-002`, `TR-ui-007`
*(Requirement text lives in `docs/architecture/tr-registry.yaml` — read fresh at review time)*

**ADR Governing Implementation**: ADR-0012: UI Framework
**ADR Decision Summary**: Amber is an accent/fill color — never used for text on parchment (fails WCAG AA at 1.5:1 contrast). All colors must have shape/icon/value/sound backup — never encode game-critical information by hue alone. Accessibility: focus ring ≥3:1 contrast, typewriter speed accessibility setting.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: Kaiju HTML/CSS-like UI system — verify CSS border-color animation, glow effects, conditional CSS class toggling. Post-cutoff APIs, no LLM training data exists.

**Control Manifest Rules (this layer)**:
- Required: Amber is accent/fill only; all colors have shape/icon/value/sound backup (never encode game-critical information by hue alone); accessibility — focus ring (≥3:1), typewriter speed setting
- Forbidden: Never allow amber-on-parchment text (1.5:1 contrast ratio fails WCAG AA); never encode game-critical information by hue alone
- Guardrail: UI render (static) <1ms/frame; no frame drops during UI animations

---

## Acceptance Criteria

*From GDD `design/gdd/dialogue-ui.md`, scoped to this story:*

- [ ] **AC-1**: GIVEN an NPC with trust level 0, WHEN conversation starts, THEN the portrait border shows no amber glow. GIVEN trust level 1-2, WHEN conversation is active, THEN the portrait border shows partial amber glow. GIVEN trust level 3 (confidant), THEN the portrait border shows full amber glow.
- [ ] **AC-2**: GIVEN a trust increase event during conversation, WHEN trust level changes, THEN the portrait border pulses amber for 500ms accompanied by a soft chime sound. The pulse is a brief brighten-dim cycle — not a bounce, scale, or spin.
- [ ] **AC-3**: GIVEN the LLM inference has failed over to degraded/fallback mode, WHEN an NPC response is generated, THEN the degraded mode indicator icon is shown next to the NPC portrait, and the response text appears as plain text (all at once, no typewriter effect).
- [ ] **AC-4**: GIVEN degraded mode is active, WHEN the LLM recovers and a normal response is generated, THEN the degraded indicator is removed and the typewriter effect resumes for subsequent responses.

---

## Implementation Notes

*Derived from ADR-0012:*

- **Trust level border glow**: Three visual states on the NPC portrait container's border:
  - Trust 0 (stranger): No glow — default parchment-border styling.
  - Trust 1-2 (acquaintance/trusted): Partial amber glow — use CSS `box-shadow` or border-color with a subdued amber (`#D4A040` or art-bible-defined amber). Glow is subtle, not a neon effect.
  - Trust 3 (confidant): Full amber glow — brighter border, more pronounced shadow/glow. Still must not compete with the text for attention.
  - **Accessibility note**: Trust level must also be conveyed by a non-color cue — display trust level as small filled/unfilled dots or a text label ("Stranger" / "Acquaintance" / "Trusted" / "Confidant") next to the NPC name. Never encode this information by hue alone.
- **Trust increase pulse**: On `TrustChanged` event from EventBus (emitted by Trust & Relationship system): add a CSS class `trust-pulse` to the portrait border. Animation: border-color brightens from current trust level amber to a brighter amber, then returns over 500ms (ease-out). After animation completes, update the border to the new trust level's static glow. Play soft chime via Audio System event. The pulse is a brightness modulation — no bounce, scale, spin, or flip (per ADR-0012 motion language).
- **Degraded mode indicator**: A small pen-and-ink warning icon (exclamation in circle, umber) positioned next to the NPC portrait in the top bar. Tooltip on hover: "AI response degraded — text-only fallback active." The icon must not be amber (amber is for trust, not warnings — but if using amber, pair with the icon shape).
- **Degraded mode text display**: When the AI Dialogue Engine signals degraded mode (via `LLMDegraded` event or a flag on the response message):
  1. Skip the typewriter effect — set the response text element's content to the full string immediately.
  2. Show the degraded indicator icon.
  3. Use a slightly different text treatment for the response: perhaps slightly desaturated or with a subtle dashed border to visually distinguish fallback text from normal LLM responses. But do NOT dim it to the point of reduced readability.
  4. The soft tick audio is suppressed — no typewriter, no tick.
- **Recovery from degraded mode**: When a normal (non-degraded) response arrives, remove the degraded indicator icon, remove any special text treatment, and resume normal typewriter rendering for that response and all subsequent responses.
- **Event subscriptions**: Subscribe to `TrustChanged` (entity ID, old level, new level) and `LLMDegraded` / `LLMRecovered` events via EventBus. Filter TrustChanged events for the currently conversing NPC's entity ID only.

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 001: Panel framework, NPC portrait rendering (the portrait itself), End Conversation button, state machine
- Story 002: Text input and topic chips
- Story 003: Conversation log, typewriter effect (this story overrides typewriter in degraded mode), thinking animation
- Trust level calculation: Handled by Trust & Relationship system (Core layer) — this story only displays the value

---

## QA Test Cases

*Written by qa-lead at story creation. The developer implements against these — do not invent new test cases during implementation.*

- **AC-1** (trust level border glow):
  - Setup: Start conversations with NPCs at trust levels 0, 1, 2, and 3 (use debug commands or test saves)
  - Verify: Trust 0 → no amber on portrait border. Trust 1-2 → partial/subdued amber glow. Trust 3 → full amber glow. Trust level also shown as text label or filled dots (non-color cue).
  - Pass condition: Correct glow level per trust tier, non-color backup present

- **AC-2** (trust increase pulse):
  - Setup: During conversation with an NPC, trigger a trust increase (e.g., resolve a thread, share a secret)
  - Verify: Portrait border briefly brightens then returns to new level's glow over 500ms (ease-out, no bounce/scale). Soft chime plays. After pulse, border settles at the new trust level's static glow.
  - Pass condition: 500ms pulse animation plays, chime sounds, border updates to new trust level, no bounce/scale/spin

- **AC-3** (degraded mode — indicator + plain text):
  - Setup: Trigger LLM degraded mode (kill the llama.cpp process or use a debug flag)
  - Verify: Warning icon appears next to NPC portrait. Submit a question — response appears as full plain text instantly (no typewriter, no tick sounds). Degraded indicator remains visible.
  - Pass condition: Icon visible, response is instant plain text, no typewriter animation or tick audio

- **AC-4** (recovery from degraded mode):
  - Setup: From degraded mode, restart the LLM process so normal inference resumes
  - Verify: Submit a question — degraded indicator disappears, response renders with normal typewriter effect (20 cps, tick sounds)
  - Pass condition: Indicator removed, typewriter effect restored for subsequent responses

---

## Test Evidence

**Story Type**: UI
**Required evidence**:
- UI: `production/qa/evidence/dialogue-ui-trust-degraded-evidence.md` or interaction test

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: Story 001 (Dialogue Panel Framework — NPC portrait must be rendered to attach border glow), Story 003 (Conversation Log & Typewriter — degraded mode overrides typewriter from Story 003)
- Unlocks: None (last story in epic)

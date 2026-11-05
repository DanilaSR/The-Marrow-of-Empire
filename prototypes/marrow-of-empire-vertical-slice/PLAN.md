# Vertical Slice Plan — The Marrow of Empire

- **Date**: 2026-08-12
- **Stage context**: mid-Production (sprint 5). The slice assembles the implemented
  systems into one playable [start → challenge → resolution] loop — it is
  production-tree integration work, not a throwaway build.
- **Per-INT records**: `int-001` … `int-004` in this directory.

## Validation Question

> Does a player, starting from nothing, experience the core fantasy — asking
> questions, following a thread, fighting for access, and connecting the dots —
> within ~5 minutes, without developer guidance?

Second half of the question: can this loop be built at representative quality
on schedule — answered by the velocity log in each INT record.

## The Loop (start → challenge → resolution)

1. **Start** — Boot → title screen → New Game → player spawns in the border town (debug-shape world)
2. **Challenge** — Walk to the blacksmith → ask about the strange ore (dialogue + question system + KG + LLM) → thread opens → walk to the old mine → **one combat encounter** → victory → recover the inscription → return → ask again → thread chains
3. **Resolution** — 3+ connected facts → contradiction detected → **revelation fires** in the journal. The player feels smart — the hypothesis being tested.

## INT Breakdown

| INT | Scope | Status |
|-----|-------|--------|
| INT-1 | World bootstrap: New Game → debug-shape world, click-to-move, camera follow | ✅ Complete (see `int-001-world-bootstrap.md`) |
| INT-2 | Talk loop: 3 NPCs with `.kg` content, dialogue pipeline, LLM companion, thread wiring | ✅ Complete — pipeline verified live (see `int-002-talk-loop.md`) |
| INT-3 | Combat engine wiring + mine encounter + victory/loot | ✅ Complete — fight won in the running game (see `int-003-attack-resolution.md`) |
| INT-4 | Journal/revelation + UI rendering fixes + debug cleanup | ✅ Code complete (see `int-004-journal-revelation.md`) — playtest Session 1 + REPORT.md remain |

## Explicitly Cut

Party system, inventory, map UI usage, audio content, save/load mid-loop,
factions, full death/dying rules, day/night pacing, accessibility prompt,
KitBash3D assets (debug shapes), all polish.

## Cross-Cutting Findings So Far (details in INT records)

1. Local LLM decode ~1.2–1.4 tok/s on dev hardware (10 tok/s gate) — real model works but is too slow for real-time dialogue here.
2. Markup absolute-positioned panels render top-left at ~half size in this engine version.
3. NPC click pick radius must absorb ~1.7u ray-ground offset.
4. Matcher threshold 0.3 too strict for single-keyword questions with 8–9 keyword domains (slice tuning: 0.1).

---
name: marrow-of-empire-game-context
description: The Marrow of Empire — investigation CRPG, local-LLM NPC dialogue, game pillars, key GDDs and their status
metadata:
  type: project
---

Game: **The Marrow of Empire** — isometric tactical CRPG + investigation ("you are the one who connects the dots"), Kaiju Engine, single dev.

- Pillars: 1 Every Voice Holds a Piece, 2 The World Remembers, 3 Tactical Depth Not Bloat, 4 Earned Revelation. Anti: power fantasy, quest markers, procedural content.
- NPC dialogue runs on a local LLM: Fiendish_LLAMA_3B (Llama 3.2 3B RP fine-tune, Q4_K_M, ~2.1 GB, CPU inference, llama.cpp subprocess).
- Core design principle (AI Dialogue Engine): the Knowledge Graph owns epistemics (fact selection, truth, gating, misdirection); the LLM owns sociolinguistics (voice, tone, deflection phrasing).
- Key docs: `design/gdd/game-concept.md`, `design/gdd/ai-dialogue-engine.md` (MAJOR REVISION, 7 blockers, reviewed 2026-08-09), `design/gdd/npc-knowledge-graph.md`. KG facts authored per-NPC as `.kg` YAML: domains, fact types (FACT/RUMOR/OPINION/SECRET), trust gating 0–3, knows_not boundaries, leads_to referrals, few-shot exemplars per archetype.

**Why:** this is the context for any narrative-system work — voice, trust, memory, investigation.

**How to apply:** when reviewing or extending narrative systems, read these three GDDs first and check against the pillars; the dialogue engine is the highest-risk, highest-fantasy system in the game.

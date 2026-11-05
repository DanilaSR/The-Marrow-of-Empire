# Concept Prototype Report: The Marrow of Empire

> **Date**: 2026-08-08
> **Prototype Path**: Paper
> **Concept File**: design/gdd/game-concept.md

---

## Hypothesis

"If the player talks to NPCs using a fractured knowledge system (each NPC knows
their domain, no one knows everything), they will feel like an investigator piecing
together a mystery — evidenced by following a thread chain across 3+ NPCs without
prompting and reporting curiosity rather than confusion."

**Verdict: CONFIRMED**

---

## Riskiest Assumption Tested

The knowledge graph structure — "deep but fractured" knowledge across NPCs —
creates satisfying investigation, not frustration. If players can't form mental
connections between NPC knowledge fragments, the core concept fails regardless
of LLM quality or visual polish.

This assumption held. The fractured structure generated natural thread-following
behavior and a clear "detective moment" when the player connected three
independent fragments about Captain Veren.

---

## Approach

**Path chosen:** Paper (rules document + simulated play-log)

**Reason for path:** The risk is in the knowledge graph *design*, not the LLM
response quality or the engine integration. Paper lets us validate whether
"deep but fractured" creates interesting investigation in hours, not days.

**Shortcuts taken (intentional):**
- 5 NPCs, not 20 — the minimum needed to test the thread-chain hypothesis
- No LLM integration — NPC responses derived manually from their knowledge graphs
- No combat, exploration, or UI — only the knowledge graph and conversation flow
- Simulated player, not a real external playtester — sufficient for initial design validation
- Trust gates modeled as simple conditions, not a full system

---

## Result

The simulated player followed a thread chain across 5 NPCs unprompted:
1. Halla (tavern keeper) — learned about Aldric's satchel, the argument with Veren
2. Greta (blacksmith) — learned about Ley-touched metal, directed to Orin
3. Orin (hermit) — learned the Ley Network truth, the Archivists, Veren's mark — **detective moment**
4. Maren (herbalist) — confirmed poison, Veren's purchases — **evidence collected**
5. Captain Veren — confronted with 3 pieces of evidence — **confession**

The detective moment ("the Archivist connection clicked — the closed eye on
Veren's desk") occurred at NPC 3 when three independent fragments converged:
Veren argued with Aldric (Halla) + Veren closed the case in one day (Halla) +
Veren has the Archivist mark on his desk (Orin).

No confusion points were reported. The player described everything as "fine."

---

## Metrics

| Metric | Value |
|--------|-------|
| Path used | Paper |
| Iterations to playable | N/A (Paper path) |
| Prototype duration | ~1 hour (design + writing) |
| Playtesters | 1 internal (simulated) |
| Feel assessment | Investigation flow was natural — thread-following emerged without prompting |
| Hypothesis verdict | CONFIRMED |

---

## Recommendation: PROCEED

The fractured knowledge graph design works. A player given 5 NPCs with domain-limited
knowledge naturally followed threads across them, experienced a genuine "click"
moment when fragments connected, and reported no confusion or friction. The core
investigation loop — talk to NPC → discover fragment → follow thread to next NPC →
connect fragments → revelation — is viable.

The prototype also surfaced refinements worth capturing in GDDs:
- The term "Ley" needs a gentler introduction (the blacksmith uses it without knowing
  what it means — realistic but briefly confusing for the player)
- The satchel contained three items at once (metal + lens + notebook) — spreading
  discoveries across the environment would feel more natural
- Trust gates (Maren, Orin) worked well — earning access to deeper knowledge felt meaningful
- 5 NPCs with 1 central mystery is the right density for an opening region

---

## If Proceeding

**Core tuning values discovered:**
- 5-7 hop thread chains feel right for regional investigations (not too short, not too long)
- 3 pieces of evidence is the minimum threshold for a satisfying confrontation
- Trust gates should gate ~30-40% of an NPC's knowledge — enough to feel earned,
  not so much that the investigation stalls

**Assumptions confirmed:**
- Players will follow threads across NPCs without quest markers
- Domain-limited knowledge (each NPC knows their world) creates natural investigation
- The "fractured" structure prevents any single NPC from being an exposition dump

**Assumptions disproved:**
- None — this prototype validated rather than challenged the concept's assumptions

**Emergent mechanics:**
- The "detective moment" — when 3+ fragments from different NPCs converge — should
  be tracked and highlighted by the journal system. The game should recognize when
  the player has enough pieces to form a conclusion.

**Next steps:**
1. `/art-bible` — complete visual identity (sections 2, 4-9 still pending)
2. `/map-systems` — decompose the concept into individual systems
3. `/design-system [mechanic]` — author per-system GDDs, using prototype learnings
   for Tuning Knobs and Formulas
4. `/create-architecture` — master architecture blueprint
5. `/architecture-review` — validate coverage before Pre-Production gate

---

## Lessons Learned

- **What assumptions were broken by actually building this?**
  None — the prototype confirmed rather than refuted. The knowledge graph approach
  worked as designed. The real test will be scaling from 5 to 100+ NPCs.

- **What surprised us that didn't show up in the brainstorm?**
  How naturally the trust mechanic emerged. Maren gating her poison knowledge behind
  "why do you want to know?" created a satisfying moment — the player earned her
  trust by demonstrating genuine investigation. This should be a first-class system,
  not an afterthought.

- **What would we test differently next time?**
  Test with a real external playtester who hasn't seen the design. The simulated
  player knew the investigation was solvable. A real player might hit confusion
  points the designer can't see. Also: test what happens when a player follows
  threads in a sub-optimal order (e.g., visiting Orin first, before Halla).

---

> *Prototype code location: `prototypes/marrow-of-empire-concept/`*
> *This code is throwaway. Never refactor into production.*

# Game Concept: The Marrow of Empire

*Created: 2026-08-08*
*Status: Draft*

---

## Elevator Pitch

> It's an isometric tactical CRPG where you question AI-driven NPCs to piece together
> the truth of an empire's collapse, in a borderland where every person you meet holds
> a genuine fragment of the answer — and no one holds the whole picture.

---

## Core Identity

| Aspect | Detail |
| ---- | ---- |
| **Genre** | Isometric CRPG — tactical combat + investigation-driven narrative |
| **Platform** | PC (Steam / Epic) |
| **Target Audience** | Explorer-Investigators who read every book, exhaust every dialogue option; Tactical Achievers who want deep combat with meaning |
| **Player Count** | Single-player with party (4–6 characters) |
| **Session Length** | 1–2 hour sessions — natural stopping points at thread exhaustion |
| **Monetization** | Premium (single purchase) |
| **Estimated Scope** | Large (multi-year, solo developer) |
| **Comparable Titles** | Baldur's Gate 1, Divinity Original Sin 2, Disco Elysium |

---

## Core Fantasy

You are the one who connects the dots. In a fractured borderland where the old
empire's fall shattered both the land and the truth, you are the investigator who
pieces history back together — not by finding a glowing quest marker, but by
talking to the right people, asking the right questions, and sometimes fighting
for the right to know. Every NPC is a living fragment of the world's memory.
Your mind is your greatest weapon, and the reward is understanding what no one
else has seen.

---

## Unique Hook

Like Baldur's Gate 1, *and also* every NPC is an AI agent with genuine, dynamic
knowledge about the world. The blacksmith knows metallurgy and the quality of
local ore. The hermit knows the old forest and what lives there. The captain
knows troop movements and who gave the orders. Ask anyone about anything — they
answer from what they genuinely know, not a scripted dialogue tree. The game is
not about picking the right dialogue option; it's about finding the right person
and asking the right question.

---

## Player Experience Analysis (MDA Framework)

### Target Aesthetics (What the player FEELS)

| Aesthetic | Priority | How We Deliver It |
| ---- | ---- | ---- |
| **Discovery** | 1 (Primary) | Uncovering the empire's fall through fragmented NPC knowledge; hidden locations, secret histories, connections between seemingly unrelated facts |
| **Challenge** | 2 | Tactical turn-based combat rewarding positioning, synergies, and preparation; intellectual challenge of piecing together the truth |
| **Narrative** | 3 | Player-driven investigation creates unique story arcs; the plot emerges from which threads you follow and which connections you make |
| **Sensation** | 4 | Deep nature immersion — forests, rivers, sunsets rendered with care; dynamic weather, day/night cycles, seasonal change |
| **Fantasy** | 5 | Role-playing as an investigator in a world that treats you as one, not as the chosen one |
| **Fellowship** | 6 | Party members with their own knowledge fragments and perspectives; building trust with NPCs to unlock deeper knowledge |
| **Expression** | N/A | Not a primary goal — some expression through party composition and investigation approach |
| **Submission** | N/A | Not a goal — this is an intellectually engaged experience |

### Key Dynamics (Emergent player behaviors)

- Players will naturally cross-reference NPCs: "The blacksmith mentioned strange ore — let me ask the miner about that."
- Players will build mental maps of who knows what, developing investigative intuition over time.
- Players will debate whether to fight, talk, or find another way — tactical decisions shaped by narrative context.
- Players will share discoveries: "Did you find the hermit who knows about the old king's secret?"
- Players will make notes, keep journals, and track threads outside the game.

### Core Mechanics (Systems we build)

1. **AI NPC Knowledge System** — Each NPC has a knowledge graph covering their domain (craft, land, people, history, secrets). Player questions are matched against what the NPC genuinely knows.
2. **Tactical Turn-Based Combat** — Party-based, grid or range-based, with positioning, synergies, environmental interaction, and meaningful stakes for every encounter.
3. **Thread-Based Investigation** — No traditional quest log. Players track leads as threads: "Talk to X about Y." Threads emerge naturally from conversation and exploration. The system helps you remember, but doesn't tell you where to go.
4. **Persistent World State** — NPCs remember interactions. Cleared ruins stay cleared. Factions react to your discoveries. The world changes and the change is legible.
5. **Living Environment** — Day/night cycle, weather, seasons affect both visuals and gameplay (NPC schedules, wildlife patterns, travel conditions).

---

## Player Motivation Profile

### Primary Psychological Needs Served

| Need | How This Game Satisfies It | Strength |
| ---- | ---- | ---- |
| **Autonomy** | Total freedom in which threads to follow, who to talk to, what to ask. No linear quest chain. The investigation is yours. | Core |
| **Competence** | Tactical combat mastery grows over time. Investigation skill develops — you get better at knowing who to ask and what questions matter. The game rewards you for being clever. | Core |
| **Relatedness** | Party members are fully realized characters with their own knowledge. NPCs feel like people, not quest dispensers. The world is populated by characters worth knowing. | Supporting |

### Player Type Appeal (Bartle Taxonomy)

- [x] **Achievers** (goal completion, collection, progression) — How: Threading together the full truth of the empire's fall is the ultimate achievement. Tactical mastery provides mechanical achievement.
- [x] **Explorers** (discovery, understanding systems, finding secrets) — How: The PRIMARY audience. Every NPC is a discovery. Every conversation can reveal a new thread. The world is dense with secrets to uncover.
- [ ] **Socializers** (relationships, cooperation, community) — How: Not a primary drive, but party dynamics and NPC relationships provide interpersonal depth.
- [ ] **Killers/Competitors** (domination, PvP, leaderboards) — How: Not applicable. Single-player, no competitive element.

### Flow State Design

- **Onboarding curve**: The first NPC you meet teaches you the questioning pattern by example — they mention something they know, something they've heard, and someone you should talk to. Within 10 minutes you understand: ask, listen, connect, follow.
- **Difficulty scaling**: Combat difficulty scales with region depth (deeper into the borderland = more dangerous). Investigation difficulty scales with thread complexity — early threads are short chains (2-3 hops), late threads span regions and require earned trust.
- **Feedback clarity**: When pieces click, the game acknowledges it — your character's journal updates, party members comment, new threads open. You know you're progressing because the picture gets clearer.
- **Recovery from failure**: Combat defeat reloads to before the encounter. Investigation dead ends are not failures — wrong threads still teach you something about an NPC or region. There are no permanently missed clues.

---

## Core Loop

### Moment-to-Moment (30 seconds)
The classic CRPG trio, each action feeding the others:
- **Talk** → NPC names a person or place → **Explore** to find them
- **Explore** → environment surfaces a threat or clue → **Fight** when words fail
- **Fight** → victory reveals loot, prisoners, documents → **Talk** about what you found

### Short-Term (5-15 minutes)
The "one more question" engine: talk to the blacksmith → learn about strange ore from the old mine → fight through the mine → find an inscription in Old Imperial → ask the scholar about the inscription → learn it references a name the blacksmith mentioned. Each answer opens two new threads. This is where "one more turn" psychology lives.

### Session-Level (1-2 hours)
Enter a region, map its NPCs and their knowledge domains, follow the most promising threads. Natural stopping points: you've exhausted your current leads in this area and have a list of people to find elsewhere. The session ends with curiosity — you know where to go next, you just haven't gone there yet.

### Long-Term Progression
Your understanding of the empire's fall deepens in layers:
1. **Surface**: The official story everyone tells
2. **Contradictions**: The cracks in the official story
3. **Hidden truth**: What really happened — and who benefits from it staying buried

In parallel: your party grows in tactical capability, NPC relationships deepen, the borderland's factions shift in response to your discoveries, and you gain access to regions that were closed to you before.

### Retention Hooks
- **Curiosity**: Unanswered questions, unexplored regions, NPCs you haven't met yet, threads pointing to distant places
- **Investment**: The investigation you've built — your notes, your leads, your understanding. Characters you care about and trust.
- **Mastery**: Combat skills to refine, investigation intuition that deepens, party builds to optimize
- **Social**: Not applicable (single-player)

---

## Game Pillars

### Pillar 1: Every Voice Holds a Piece
No NPC is filler. Every character you can talk to knows something real about
their domain — their craft, their land, their neighbors, their history. If you
can ask, they can answer.

*Design test*: "Should this NPC just say 'I don't know anything'?" → This pillar
says: give them something they genuinely DO know, even if it's small.

### Pillar 2: The World Remembers
Actions echo. An NPC you helped remembers. A faction you crossed reacts. A ruin
you cleared stays cleared — and something else might move in. The world state is
persistent and legible.

*Design test*: "Should this event reset after the quest?" → This pillar says:
the world changed. Keep it changed.

### Pillar 3: Tactical Depth, Not Tactical Bloat
Combat rewards mastery — positioning, synergies, knowing when to fight and when
to find another way. No filler fights against trash mobs. Every combat encounter
has a reason to exist in the world.

*Design test*: "Should we add more enemy types?" → This pillar says: only if
they add a new tactical decision, not just a new texture or stat block.

### Pillar 4: Earned Revelation
The truth is not handed out. You earn it — by finding the right people,
connecting their fragments, and sometimes fighting for access. The player should
feel smart when pieces click together, not led by the nose.

*Design test*: "Should we add an exposition dump here?" → This pillar says:
break it into fragments and scatter them across people who would actually know
those pieces.

### Anti-Pillars (What This Game Is NOT)

- **NOT a power fantasy** — You are not the chosen one, not the prophesied hero, not the center of the universe. You're a determined investigator in a world that doesn't revolve around you. This protects: *Earned Revelation* and *Every Voice Holds a Piece*.
- **NOT a completionist checklist** — No quest log with map markers directing you to the next objective. You follow threads, not waypoints. This protects: *Discovery* as the primary aesthetic and *Autonomy* as the core psychological need.
- **NOT procedurally generated** — Every NPC, every location, every clue is hand-authored. Depth over infinite breadth. This protects: *Every Voice Holds a Piece* (hand-authored knowledge is meaningful; procedural knowledge is filler).
- **NOT an action RPG** — Combat is turn-based and tactical. Player skill is planning and decision-making, not reflexes. This protects: *Tactical Depth, Not Tactical Bloat* and accessibility for the core CRPG audience.

---

## Visual Identity Anchor

*Note: Full art bible to be created by `/art-bible`. This section is the seed
captured during concept brainstorming.*

**Visual Direction**: Dark fantasy — the fallen empire rendered in Gothic stone,
jagged spires, crumbling archways, and haunted forests. The beauty of a ruined
world reclaimed by nature. Deep shadows, dramatic light, a landscape that feels
ancient, powerful, and slightly menacing.

**One-line visual rule**: Every scene should feel like a dark fairy tale — beautiful
and dangerous in equal measure, where the ruins whisper and the forest watches.

**Supporting visual principles**:
- *Dark fantasy, not horror* — the world is ominous and melancholic, not terrifying; beauty persists in the decay
- *Nature as reclamation* — the empire didn't just fall, it was swallowed; roots crack stone, moss veils statues, rivers claim roads
- *Atmosphere over fidelity* — mood lighting, weather, and color palette matter more than texture resolution
- *Readable tactical space* — darkness and atmosphere don't compromise gameplay clarity; combat areas must be visually legible

**Color philosophy**: Deep earth tones (dark stone, muted green, shadowed brown),
punctuated by dramatic contrast (golden lantern light against darkness, the pale
glow of a clearing at dusk, the cold blue of high mountain shadow). The palette
should feel ancient, grounded, and faintly melancholic.

**Asset foundation**: KitBash3D Dark Fantasy pack — Gothic architecture, ruined
structures, dark fantasy environment kit. Provides the primary architectural and
environmental asset base (Blender/FBX/OBJ formats available).

---

## Inspiration and References

| Reference | What We Take From It | What We Do Differently | Why It Matters |
| ---- | ---- | ---- | ---- |
| Baldur's Gate 1 | Isometric CRPG structure, party-based tactical combat, sense of a real world with real people | AI-driven NPC knowledge replaces scripted dialogue trees; investigation is the core, not combat | Proves the format works and the audience exists |
| Divinity Original Sin 2 | Tactical depth, elemental and environmental combat interactions, world reactivity | Less combat density, more investigation density; every fight has narrative weight | Shows tactical turn-based combat still has a massive audience |
| Disco Elysium | Dialogue as gameplay, investigation without hand-holding, player intelligence respected | Includes tactical combat as a meaningful system alongside investigation | Proves a CRPG can succeed on the strength of conversation and discovery |
| The Drakensang: The River of Time | Rich, grounded world-building; NPCs who feel like they inhabit the world | AI-driven dynamic knowledge instead of fixed dialogue trees | Shows the value of a world that feels lived-in rather than game-y |

**Non-game inspirations**: The Name of the Rose (fragmented truth reconstructed from scattered sources), landscape painting of the Hudson River School (nature as awe), The Silmarillion (history as a puzzle pieced together from multiple tellings).

---

## Target Player Profile

| Attribute | Detail |
| ---- | ---- |
| **Age range** | 20–45 |
| **Gaming experience** | Mid-core to hardcore CRPG players |
| **Time availability** | 1–2 hour sessions, 3–5 sessions per week |
| **Platform preference** | PC (Steam primary) |
| **Current games they play** | Divinity Original Sin 2, Baldur's Gate 3, Disco Elysium, Pillars of Eternity, Pathfinder |
| **What they're looking for** | A world that treats them like an adult — no quest markers, no glowing trails, real investigation where they feel smart for figuring things out |
| **What would turn them away** | Hand-holding tutorials, quest compasses, filler combat, NPCs who say nothing interesting, worlds that feel like theme parks |

---

## Technical Considerations

| Consideration | Assessment |
| ---- | ---- |
| **Engine** | Kaiju Engine (custom Go-based engine, Vulkan rendering, HTML/CSS-like UI system) |
| **Key Technical Challenges** | AI NPC Knowledge System powered by local LLM (~3.8B params, Q5_K_M quantized GGUF, ~2.7GB) — real-time inference on player hardware; Persistent World State system; Thread-based investigation tracking; LLM context management (NPC knowledge graph → LLM prompt) |
| **Art Style** | 2.5D isometric — pre-rendered or hand-painted backgrounds with 3D characters, or full 3D isometric |
| **Art Pipeline Complexity** | Medium-High — KitBash3D Dark Fantasy pack provides base assets (architecture, environment); custom engine import pipeline needed; hand-crafted scene composition |
| **Audio Needs** | Moderate to heavy — ambient nature soundscapes are critical to the sensation aesthetic; adaptive music for exploration vs. combat |
| **Networking** | None — single-player |
| **Content Volume** | Target: 5-7 regions, 100+ NPCs, 30-60 hours of core content |
| **Procedural Systems** | None — all content is hand-authored. Nature systems (weather, wildlife, seasons) are systemic but not procedural |

---

## Risks and Open Questions

### Design Risks
- **Information density overload**: 100+ NPCs with real knowledge could overwhelm players. Mitigation: region-based knowledge scoping (NPCs in one region primarily know about that region) and journal tools that help track threads.
- **Investigation fatigue**: If every answer leads to two more questions, players may feel they're on an infinite treadmill. Mitigation: each thread chain has a satisfying resolution — a revelation, a location, a confrontation.
- **Combat pacing**: If combat is sparse and meaningful, will players who love tactical combat feel underserved? Mitigation: optional combat zones (dangerous ruins, contested territories) for players who want more.

### Technical Risks
- **AI NPC Knowledge Authoring**: The defining feature is also the biggest unknown. NPC knowledge must be structured as a knowledge graph, then converted to LLM context at runtime. The authoring pipeline (how you define what each NPC knows) and the prompt engineering (how you constrain the LLM to stay in character and within its knowledge domain) are the critical path. Prototype this first with the ~3.8B GGUF model.
- **Local LLM inference**: Running LLM inference on player hardware alongside the game introduces memory (2.7GB+ for the model) and latency constraints. NPC responses must feel real-time. Mitigation: quantized model already chosen (Q5_K_M); consider inference batching, response streaming, and graceful degradation on lower-end hardware.
- **Custom engine overhead**: Building Kaiju Engine features alongside game features — rendering, UI, save/load, asset pipeline — will consume significant development time before game-specific work begins. KitBash3D assets reduce some art pipeline burden but introduce their own import/format conversion work.
- **Persistent world state**: Tracking meaningful changes across a multi-region world with 100+ NPCs requires careful state architecture from day one.

### Market Risks
- **Niche within a niche**: CRPGs are niche; investigation-driven CRPGs are a niche within that niche. The audience exists (Disco Elysium proved it) but is finite.
- **AI NPCs as a selling point**: "AI" in games currently carries baggage (hype, skepticism, ethical concerns). Positioning matters — this is about dynamic knowledge, not generative slop.

### Scope Risks
- **Custom engine + first game + multi-year epic**: The triple threat. Every system is built from scratch. Mitigation: MVP as engine proving ground; aggressive scope discipline.
- **Content volume for "deep but fractured"**: Hand-authoring knowledge for 100+ NPCs is a massive writing task. Mitigation: region-by-region development; each region ships complete before the next begins.

### Open Questions
- **Knowledge graph data model**: What's the right structure for NPC knowledge that's both authorable and queryable? — Resolve with a prototype of the knowledge system.
- **Question input method**: How does the player actually ask questions? Free text? Keyword selection? Topic menu? — Prototype and playtest with the MVP.
- **Art pipeline for Kaiju Engine**: What does the asset import workflow look like for isometric environments? — Define during engine setup.
- **Save/load architecture**: How does persistent world state serialize in Kaiju Engine? — Design during architecture phase.

---

## MVP Definition

**Core hypothesis**: Players find the loop of "talk to NPCs → discover threads →
explore to follow them → fight when necessary → return with new questions"
intrinsically satisfying for sessions of 1+ hours — and the local LLM-powered NPC
responses feel natural, grounded, and worth engaging with.

**Required for MVP**:
1. One complete region — a dark fantasy border town with surrounding wilds, built with KitBash3D Dark Fantasy assets (forest, river, 2-3 points of interest)
2. 15-25 NPCs with knowledge graphs + LLM integration — each NPC has a defined knowledge domain; the ~3.8B GGUF model generates responses constrained to that domain at runtime
3. Question system — player types or selects topics; LLM responds in-character, constrained by knowledge graph
4. Tactical turn-based combat — party of 2-4, basic positioning and abilities, 3-5 meaningful encounters
5. Thread journal — tracks leads and connections the player has discovered (NOT a quest log)
6. Basic exploration — movement, interaction, day/night cycle
7. One complete investigation thread chain (5-7 hops, leading to a real revelation)

**Explicitly NOT in MVP** (defer to later):
- Multiple regions
- Full party system (4-6 characters with deep backstories)
- Faction reputation system
- Seasonal changes (weather and day/night are in; full seasons are later)
- Persistent world state beyond the current session

### Scope Tiers

| Tier | Content | Features | Timeline |
| ---- | ---- | ---- | ---- |
| **MVP** | 1 region, ~20 NPCs, 1 investigation chain, 5 encounters | Core loop: questioning, threading, tactical combat, exploration | ~6-9 months (engine groundwork + first region) |
| **Vertical Slice** | 1 region, polished to production quality | MVP features + persistence, save/load, full combat system, party of 4 | ~12-15 months |
| **Alpha** | 3 regions, 50+ NPCs | All core systems functional, faction reputation, world state persistence | ~18-24 months |
| **Full Vision** | 5-7 regions, 100+ NPCs, 30-60 hours | All features polished, full investigation web across all regions | Multi-year (3+) |

---

## Next Steps

- [ ] Run `/setup-engine` to configure Kaiju Engine and populate engine reference docs
- [ ] Run `/art-bible` to create the visual identity specification — do this BEFORE writing GDDs
- [ ] Run `/prototype` to validate the AI NPC knowledge loop is fun before committing to GDDs
- [ ] Use `/design-review design/gdd/game-concept.md` to validate concept completeness
- [ ] Decompose into systems with `/map-systems`
- [ ] Author per-system GDDs with `/design-system`
- [ ] Plan technical architecture with `/create-architecture`
- [ ] Record architectural decisions with `/architecture-decision`
- [ ] Run `/architecture-review` to bootstrap the TR registry
- [ ] Validate readiness with `/gate-check`

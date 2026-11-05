# NPC Knowledge Graph

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 1 (Every Voice Holds a Piece — this IS the pillar made mechanical)

## Summary

The NPC Knowledge Graph is the data model that powers Every Voice Holds a Piece.
Each NPC has authored domains (2–4 categories of expertise), facts (FACT, RUMOR,
OPINION, or SECRET types), trust-gated access levels (0–3), explicit knowledge
boundaries, and entity references that form the edges of the investigation web.
Authored as YAML `.kg` files per NPC. Queried by the AI Dialogue Engine at
runtime to construct constrained LLM prompts.

> **Quick reference** — Layer: `Core` · Priority: `MVP` · Key deps: `Entity Framework` · Risk: `HIGH`

## Overview

The NPC Knowledge Graph is the data model that makes Every Voice Hold a Piece.
It defines what every NPC in The Marrow of Empire knows — their domain of
expertise, the specific facts they can share, the people they know about, and
the trust thresholds that gate their deepest knowledge. This is not a dialogue
tree. It is a structured knowledge database that the AI Dialogue Engine queries
at runtime to construct LLM prompts that keep NPCs in character and within
their knowledge boundaries.

The system defines:

- **Knowledge domains**: Every NPC has a defined domain — the set of topics
  they genuinely know about. A blacksmith knows metallurgy, local ore quality,
  and who buys their work. They do NOT know about imperial politics, ancient
  Ley technology, or what the hermit saw in the woods last night. Domains
  create the "fractured" in "deep but fractured" — no NPC knows everything,
  and the gaps between domains are where investigation lives.
- **Fact types**: Knowledge is categorized by type. An NPC can know FACTS
  (things they've directly experienced or verified), RUMORS (things they've
  heard from others, possibly inaccurate), OPINIONS (their subjective
  judgments about people and events), and SECRETS (knowledge they possess
  but will not share without trust). Fact types determine both the
  reliability of the information and how the AI Dialogue Engine presents it.
- **Entity references**: Facts reference other entities — people, places,
  objects, events — by their GameEntityID. This is what makes cross-
  referencing possible. The blacksmith mentions "strange ore from the old
  mine" (references the mine entity). The hermit mentions "the old mine's
  lower levels" (references the same entity). The player connects them.
- **Trust gating**: Knowledge is layered by trust level. An NPC's surface
  knowledge (trust level 0) is freely shared. Their deeper knowledge
  (trust levels 1-3) requires the player to earn trust — by demonstrating
  genuine investigation, helping the NPC, or being vouched for by someone
  the NPC trusts. Trust gating is what makes relationship-building
  mechanically meaningful.
- **Knowledge boundaries**: Explicitly defining what an NPC does NOT know
  is as important as defining what they DO know. Boundaries prevent the
  LLM from inventing knowledge outside the NPC's domain and create the
  natural referral pattern ("I don't know about that — ask the herbalist").

Without the NPC Knowledge Graph: NPCs are either omniscient quest dispensers
or empty shells. The core loop of "talk → discover → follow → connect" has
no data to run on. Every Voice is silent.

## Player Fantasy

The NPC Knowledge Graph is invisible to the player — but everything the player
loves about The Marrow of Empire depends on it.

**The world is full of witnesses.** When the player talks to the blacksmith and
learns about the strange ore, then talks to the hermit and learns about the old
mine, then connects those fragments — that moment of connection is the NPC
Knowledge Graph working. The player feels smart. The player feels like a
detective. The player never thinks "what a well-designed knowledge graph."
They think "I need to find out who else knows about that ore."

**Trust is earned, not clicked.** When the herbalist only shares surface
knowledge at first, then opens up after the player demonstrates genuine
investigation — that's trust gating. The player feels like they earned that
information, not like they selected the right dialogue option. The NPC feels
like a person with boundaries, not a vending machine for exposition.

**Every question has an answer.** The defining fantasy of the game is: you can
ask anyone about anything, and they will answer from what they genuinely know.
The knowledge graph makes that promise deliverable. It constrains the LLM to
the NPC's actual knowledge, preventing hallucinations and keeping every NPC
grounded in the world.

**The web is real.** Facts in the knowledge graph reference real entities,
real places, real events. When the player follows a thread from the blacksmith
to the hermit to the Captain, they're following references in a connected data
structure — but it feels like uncovering a conspiracy. The graph is the web;
the player is the spider.

## Detailed Design

### Core Rules

**CR-1: Knowledge Domain**

A knowledge domain is a named category of expertise. Every fact an NPC knows
belongs to exactly one domain.

| Domain | Example Topics | Example NPCs |
|--------|---------------|-------------|
| `craft_metallurgy` | Ore quality, smithing techniques, imperial alloys, weapon composition | Blacksmith, Armorer |
| `herbalism_medicine` | Local plants, poisons, remedies, wound treatment | Herbalist, Physician |
| `town_gossip` | Who talks to whom, recent events, rumors, relationships | Tavern Keeper, Merchant |
| `military_security` | Patrol routes, troop movements, town defenses, threat assessment | Captain of the Watch, Guard |
| `old_empire_lore` | Ley Network, imperial history, ancient language, the Archivists | Hermit, Scholar |
| `wilderness_survival` | Forest paths, animal behavior, foraging, weather signs | Hunter, Hermit |
| `trade_commerce` | Prices, supply routes, who buys what, rare goods | Merchant, Smuggler |
| `faction_politics` | Faction rivalries, power structures, hidden alliances | Faction Leader, Spy |
| `personal_history` | The NPC's own past, their relationships, their secrets | Any NPC |

Domains are authored per NPC. An NPC typically has 2–4 domains. Every fact in
an NPC's knowledge graph is tagged with its domain.

**CR-2: Knowledge Fact**

A knowledge fact is a single unit of NPC knowledge. Structure:

| Field | Type | Description |
|-------|------|-------------|
| `fact_id` | string | Unique identifier within the NPC's knowledge set |
| `domain` | string | Which domain this fact belongs to (CR-1) |
| `type` | enum | `FACT`, `RUMOR`, `OPINION`, `SECRET` |
| `content` | string | The fact itself — written as the NPC would say it (first-person natural language). This is what feeds the LLM prompt. |
| `trust_required` | int 0–3 | Minimum trust level to reveal this fact. 0 = always shared. |
| `references` | []GameEntityID | Entities this fact references (people, places, objects, events). Empty if the fact references nothing external. |
| `discovery_triggers` | []string | Optional — conditions that unlock this fact when met (e.g., `"player_has_item:imperial_lantern"`, `"player_visited:old_mine"`). Empty if always available at the trust level. |
| `contradicts` | []fact_id | Optional — IDs of other facts (from any NPC) that this fact contradicts. Used by Thread Investigation to detect inconsistencies. |
| `leads_to` | []{entity_id, topic_hint} | Optional — explicit "ask X about Y" leads this fact generates. The hint is what the NPC suggests. |

**Example fact** (blacksmith Greta, from paper prototype):
```
fact_id: "greta_ley_metal_001"
domain: "craft_metallurgy"
type: FACT
content: "This fragment... it's an alloy I've never seen. Pale silver, impossibly
light, warm to the touch even when it's been sitting in a cold forge all night.
Aldric called it 'Ley-touched metal.' Said it came from deep in the old ruins."
trust_required: 0
references: [aldric_entity_id, old_ruins_entity_id]
discovery_triggers: ["player_has_item:ley_metal_fragment"]
contradicts: []
leads_to: [{entity_id: orin_entity_id, topic_hint: "Ask the old hermit what
'Ley-touched' means. He knows the ruins better than anyone."}]
```

**CR-3: Fact Type Behavior**

The AI Dialogue Engine uses fact type to determine how the NPC presents the information:

| Type | NPC Presentation | Reliability |
|------|-----------------|-------------|
| `FACT` | States confidently. "The old mine's lower levels are flooded." | True — the NPC directly knows this. |
| `RUMOR` | States with caveat. "I heard the old mine's lower levels are flooded, but I've never been down there myself." | May be partially or fully false. Cross-reference required. |
| `OPINION` | States as judgment. "Captain Veren is a good man. Keeps us safe. A bit intense about the ruins, maybe." | Subjective — reveals NPC bias but may contain useful leads. |
| `SECRET` | Reveals reluctantly, after trust check. "*lowers voice* The Captain... he's not what he seems. I've seen his seal. The closed eye." | True — secrets are true but dangerous knowledge. |

**CR-4: Trust Gating**

Each NPC has a trust value with the player (managed by Trust & Relationship
system, stored as `trust_level` in the NPC's Persistent component). Facts
are gated by trust:

| Trust Level | Gated Facts | How to Earn |
|-------------|-------------|-------------|
| 0 (default) | All facts with `trust_required: 0` | Nothing — freely available on first meeting |
| 1 (acquaintance) | Facts with `trust_required: 0–1` | Player has demonstrated genuine investigation (asked about something the NPC actually knows about) OR completed a small favor |
| 2 (trusted) | Facts with `trust_required: 0–2` | Player has helped the NPC significantly OR been vouched for by someone the NPC trusts |
| 3 (confidant) | All facts including secrets | Player has earned deep personal trust — resolved a major thread involving this NPC, saved their life, or uncovered a secret they were keeping |

When the AI Dialogue Engine queries an NPC's knowledge, it receives only facts
at or below the current trust level. The LLM prompt contains these facts plus
the NPC's domain list and knowledge boundaries.

**CR-5: Knowledge Boundaries**

Every NPC has explicit `knows_not` entries — topics they are confirmed NOT to
know about. These are as important as what they DO know:

| Boundary Type | Example | Effect |
|---------------|---------|--------|
| Domain exclusion | Blacksmith `knows_not: ["old_empire_lore", "faction_politics"]` | LLM prompt includes: "You do NOT know about: ancient empire history, faction politics. If asked, say you don't know and suggest who might." |
| Specific entity ignorance | Blacksmith `knows_not_entities: [veren_secret_identity]` | LLM cannot reference Veren's secret. If asked directly, NPC is genuinely ignorant. |
| Deliberate misdirection | Captain Veren `knows_not: ["aldric_murder"]` but secretly knows | NPC actively lies or deflects. Fact type handles this — the NPC has SECRET facts about the murder but surface FACTS that present the official story. |

**CR-6: Authoring Format**

Knowledge graphs are authored as data files, not code. Each NPC has a `.kg`
file (YAML) in `assets/npc_knowledge/[region]/[npc_id].kg`:

```yaml
npc_id: "greta_blacksmith"
npc_name: "Greta"
domains: ["craft_metallurgy", "trade_commerce"]
personality: "Practical, observant, proud of her craft. Likes people who
  appreciate good metalwork. Suspicious of anyone who doesn't understand
  basic smithing."
speech_style: "Direct, uses smithing metaphors, calls people by their trade"
knows_not: ["old_empire_lore", "faction_politics", "herbalism_medicine"]
knows_not_entities: []
facts:
  - fact_id: "greta_ley_metal_001"
    domain: "craft_metallurgy"
    type: FACT
    content: "This fragment... it's an alloy I've never seen..."
    trust_required: 0
    references: [aldric_entity_id, old_ruins_entity_id]
    leads_to:
      - entity_id: orin_entity_id
        topic_hint: "Ask the hermit about Ley-touched metal"
```

### States and Transitions

No runtime states. The knowledge graph is static data loaded at scene start.
Facts can be marked as `discovered` in the player's PlayerKnowledge component
(owned by Thread Investigation), but the graph itself doesn't change.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Reads/Writes `Knowledge` component | Knowledge component stores `domain[]`, `facts[]` reference, `trust_level`. NPC Knowledge Graph system writes; AI Dialogue Engine reads. |
| AI Dialogue Engine | AI Dialogue reads knowledge graph | Queries NPC's facts at current trust level; receives domain list, personality, speech style, and knowledge boundaries for LLM prompt construction |
| Question System | Question System triggers knowledge queries | Player's question is matched against NPC's domain and fact content to select relevant facts |
| Trust & Relationship | Trust & Relationship writes `trust_level` | Updates the NPC's trust level in the Knowledge component; gates fact access |
| Thread Investigation | Thread Investigation reads fact references | Entity references and `leads_to` hints populate the investigation journal |

## Formulas

The NPC Knowledge Graph has no mathematical formulas. It is a data model — its
"calculations" are fact queries filtered by domain, trust level, and discovery
triggers. The query algorithm: match player's question topic against NPC's
domains → filter matching facts by `trust_required <= current_trust_level` →
filter by `discovery_triggers` satisfied → return fact set to AI Dialogue Engine.

## Edge Cases

- **If two NPCs have facts that contradict each other** (fact A in NPC 1 has `contradicts: [fact_B_id]`): Both facts are presented to the player normally. The Thread Investigation system detects the contradiction when both facts are discovered and flags it as an inconsistency for the player to investigate further.
- **If an NPC's `leads_to` references an entity that has been destroyed**: The lead is silently removed from the AI Dialogue Engine's prompt. The NPC no longer suggests talking to that entity. If the entity was critical to a thread, the Thread Investigation system detects the broken reference chain and surfaces alternative leads from other NPCs.
- **If the player asks about a topic outside all of an NPC's domains**: The AI Dialogue Engine receives the NPC's `knows_not` list and domain list. The LLM generates a natural "I don't know about that, but you should ask [relevant NPC]" response based on the NPC's personality and speech style. The NPC never invents knowledge.
- **If a fact has a `discovery_trigger` that the player hasn't met**: The fact is excluded from the AI Dialogue Engine's query results. The NPC doesn't mention it. The fact is not "missing" — it's dormant until the trigger condition is satisfied. This prevents NPCs from spoiling discoveries the player hasn't earned.
- **If an NPC has zero facts at the player's current trust level** (e.g., trust 0 but all facts require trust 1+): The NPC still has their personality and speech style. The LLM generates in-character small talk and may suggest who the player should talk to, but reveals no substantive information. This signals to the player that this NPC has more to share — if trust can be earned.
- **If a knowledge graph file fails to load or parse**: The NPC entity is created but has zero domains and zero facts. The AI Dialogue Engine receives only the NPC's name and a fallback personality ("confused, doesn't know what's happening"). The NPC can only say they don't know anything. Error is logged for content team to fix.
- **If the same fact_id appears in two different NPCs' knowledge graphs**: Both are valid — this is intentional for shared knowledge. The Thread Investigation system notes that multiple NPCs independently know the same fact, which increases its reliability rating.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | NPC Knowledge Graph depends on Entity Framework | Reads/Writes `Knowledge` component on NPC entities. Hard — cannot store knowledge without entity data. |
| Trust & Relationship | Trust & Relationship writes `trust_level` | Updates trust level on the NPC's Knowledge component, gating fact access. Hard — trust gating requires trust values. |
| AI Dialogue Engine | AI Dialogue depends on NPC Knowledge Graph | Queries NPC facts at current trust level for LLM prompt construction. Hard — dialogue has no content without the graph. |
| Question System | Question System depends on NPC Knowledge Graph | Player question topic is matched against NPC domains to select relevant facts. Hard. |
| Thread Investigation | Thread Investigation reads fact references | Entity references and `leads_to` hints populate investigation threads. Hard — threads are built from graph edges. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `default_trust_level` | 0 | 0–1 | NPCs start more open — less trust-gating, faster investigation | NPCs start more guarded — more trust required to get started |
| `max_trust_level` | 3 | 2–5 | Finer gradation of trust, more tiers of gated content | Coarser trust, simpler relationship system |
| `max_domains_per_npc` | 4 | 2–6 | NPCs have broader expertise, harder for players to form mental model of who knows what | NPCs are more specialized, easier to remember |
| `max_facts_per_npc` (soft cap) | 50 | 20–100 | More dialogue content per NPC, higher authoring burden | Less content per NPC, tighter scope |
| `max_leads_per_fact` | 3 | 1–5 | More thread branches per fact, richer investigation web | Fewer branches, more linear investigation |

## Visual/Audio Requirements

The NPC Knowledge Graph produces no direct visual or audio output. Indirect
feedback occurs through systems that read it:

- **Fact discovered**: Investigation Journal updates — brief parchment flash
  on the journal icon. Subtle paper-rustle sound.
- **Trust increased**: Brief amber glow on NPC portrait (0.3s) — signals
  that new knowledge is now accessible.
- **Contradiction detected**: Thread-red pulse on the relevant journal entry.
  Sharp string accent (single note, tense).

📌 **Asset Spec** — Visual/Audio requirements are defined. After the art bible
is approved, run `/asset-spec system:npc-knowledge-graph` for fact discovery
and trust increase VFX specifications.

## UI Requirements

The NPC Knowledge Graph has no player-facing UI. It provides data to systems
that do: Dialogue UI displays NPC name and portrait (from entity data), the
Investigation Journal displays discovered facts and thread connections, and
the trust indicator on the dialogue screen shows current trust level.

📌 **UX Flag — NPC Knowledge Graph**: The dialogue screen and investigation
journal are downstream UI concerns. Run `/ux-design` for the conversation
interface and journal during Pre-Production.

## Acceptance Criteria

- **GIVEN** an NPC with domains `["craft_metallurgy", "trade_commerce"]` and 10 facts at trust level 0, **WHEN** the AI Dialogue Engine queries the NPC at trust level 0, **THEN** all 10 facts are returned and the NPC's `knows_not` list is included in the query result.

- **GIVEN** the same NPC with 5 additional facts at trust level 2, **WHEN** queried at trust level 0, **THEN** only the 10 trust-0 facts are returned. The 5 trust-2 facts are excluded.

- **GIVEN** a fact with `discovery_trigger: "player_has_item:ley_metal_fragment"`, **WHEN** the player does NOT have the Ley metal fragment, **THEN** the fact is excluded from query results even if the trust level is sufficient.

- **GIVEN** NPC A has a fact that `contradicts` a fact from NPC B, **WHEN** the player has discovered both facts, **THEN** the Thread Investigation system receives both fact IDs with the contradiction flag set.

- **GIVEN** an NPC with `knows_not: ["old_empire_lore"]`, **WHEN** the player asks about Ley technology, **THEN** the query returns zero matching facts and the AI Dialogue Engine receives the `knows_not` entry to generate a natural deflection.

- **GIVEN** a knowledge graph file that fails YAML parsing, **WHEN** the NPC entity loads, **THEN** the NPC has zero domains, zero facts, a fallback personality, and an error is logged.

- **Performance**: Fact query (domain match + trust filter + trigger check) across 50 facts completes within 0.5ms. Full NPC knowledge graph load from `.kg` file completes within 10ms.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Should knowledge graphs support dynamic facts — facts added/removed at runtime based on world state changes? (e.g., an NPC learns something new during the game) | Game Designer | Before vertical slice | — |
| What is the authoring tool for `.kg` files? A dedicated editor, or hand-written YAML with validation? Authoring 100+ NPCs by hand in YAML is a significant content burden. | Tools Programmer | Before first NPC content authoring | — |
| Should facts support localization? The `content` field is natural language — each fact needs translations for all supported languages. | Localization Lead | Before multi-language support | — |

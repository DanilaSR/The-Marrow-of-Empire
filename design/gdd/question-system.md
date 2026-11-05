# Question System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 1 (Every Voice Holds a Piece — questions are how the player reaches those voices)

## Summary

The Question System is how the player asks NPCs about the world. It accepts player
input (free-text or topic selection), matches the question topic against the NPC's
knowledge domains, queries the NPC Knowledge Graph for relevant facts, and routes
the matched facts to the AI Dialogue Engine for response generation. It is the
input half of "ask anyone about anything."

> **Quick reference** — Layer: `Feature` · Priority: `MVP` · Key deps: `AI Dialogue Engine`, `NPC Knowledge Graph`

## Overview

The Question System defines how the player communicates with NPCs in The Marrow
of Empire. It is not a dialogue tree — there are no numbered options, no
branching paths, no "press 1 to ask about the ruins." The player types or
selects what they want to ask about, and the system finds the relevant knowledge.

The system provides:

- **Free-text input**: The player types their question in natural language.
  "What do you know about the old mine?" "Who else was with Aldric that night?"
  "Have you seen anyone strange in town lately?" This is the full "ask anyone
  about anything" promise.
- **Topic suggestions**: Alongside free text, the system surfaces suggested
  topics based on the NPC's known domains and the player's current investigation
  threads. These are prompts, not options — clicking one fills the text input.
  Suggestions help players who don't know what to ask, without restricting
  players who know exactly what they want.
- **Topic matching**: The player's question text is matched against the NPC's
  knowledge domains to determine which domains are relevant. Domain matching
  uses keyword overlap and entity reference detection — if the player mentions
  "ore" or "metal," the `craft_metallurgy` domain activates. If they mention
  "Captain Veren," facts referencing that entity are prioritized.
- **Fact routing**: Matched domains → query NPC Knowledge Graph for facts in
  those domains at the current trust level → route matching facts to AI
  Dialogue Engine for prompt construction.

Without the Question System: the player has no way to ask questions. The AI
Dialogue Engine has no input. The NPC Knowledge Graph has no query mechanism.
Every Voice is present but unreachable.

## Player Fantasy

The Question System is the player's voice in the world. It's the text input
box, the suggested topics, the feeling of "I can ask this person anything."

**You can ask what you want to know, not what the designer predicted.** Free-text
input means the player follows their own curiosity. They're not picking from a
list of questions a writer anticipated — they're asking what matters to THEM,
based on what they've discovered. The blacksmith mentioned "Ley-touched metal" —
the player types "What is Ley-touched metal?" because they want to know, not
because option 3 said "Ask about Ley-touched metal."

**The system helps without leading.** Topic suggestions surface what the NPC
can actually talk about, preventing the "I don't know anything" dead end. But
they're suggestions, not a menu. The player can ignore them and type their own
question. The system supports curiosity without scripting it.

**Questions have consequences.** What the player asks shapes what they learn.
Ask the blacksmith about ore, and you learn about the mine. Ask about Aldric,
and you learn about the scholar. Ask about the Captain, and you learn about
the argument. The player's choice of question IS their investigative strategy.

## Detailed Design

### Core Rules

**CR-1: Input Methods**

The player can ask questions through two complementary methods:

| Method | Input | Best For |
|--------|-------|----------|
| **Free text** | Player types their question into a text field. Natural language. Enter to submit. | Players who know what they want to ask; follow-up questions; specific entity/topic queries |
| **Topic suggestions** | System shows 3–6 suggested topics as clickable chips below the text field. Clicking a chip fills the text input with that question. | Players exploring what an NPC knows; first-time conversation; discovering available topics |

Both methods produce the same output: a text string that enters the topic
matching pipeline. Topic suggestions are generated from: the NPC's domain
list (e.g., "Ask about metallurgy"), the player's active investigation
threads (e.g., "Ask about the old mine"), and recently discovered facts
that reference this NPC (e.g., "Ask about Captain Veren").

**CR-2: Topic Matching**

The player's question text is matched against NPC domains to determine which
knowledge domains to query:

1. **Entity reference detection**: Scan the question for entity names (NPCs,
   locations, items, factions). If found, prioritize facts that reference
   those entities regardless of domain.
2. **Domain keyword matching**: Each domain has a set of keywords. The question
   text is scored against each domain's keyword set. Domains scoring above
   a threshold (0.3 relevance) are included in the query.
3. **Fallback**: If no domain scores above threshold, all domains are queried
   at low relevance. The NPC will likely deflect ("I don't know much about
   that"), but relevant facts may still surface.

| Domain | Keywords |
|--------|----------|
| `craft_metallurgy` | ore, metal, forge, smith, steel, iron, alloy, weapon, armor, blade, bronze, copper, silver |
| `herbalism_medicine` | herb, plant, poison, remedy, heal, wound, sickness, poultice, root, fungus |
| `town_gossip` | rumor, heard, people, town, happen, strange, recently, anyone, visitor, new |
| `military_security` | guard, patrol, attack, defend, danger, threat, safe, wall, gate, weapon, soldier |
| `old_empire_lore` | empire, ruin, ancient, old, Ley, imperial, history, past, fallen, technology, machine |
| `wilderness_survival` | forest, river, path, animal, beast, track, weather, hunt, forage, camp |
| `trade_commerce` | buy, sell, price, gold, trade, merchant, goods, supply, rare, valuable |
| `faction_politics` | faction, power, leader, alliance, enemy, rival, politics, control, influence |
| `personal_history` | you, your, past, before, born, family, childhood, name, why, how long |

**CR-3: Question Context**

The system maintains brief conversation context to improve matching:

- **Current NPC**: The NPC being spoken to (matches domains against their
  specific knowledge).
- **Conversation history**: Last 3 questions asked (any NPC). If the player
  asks "What kind of ore?" after previously asking "Tell me about the mine,"
  the system resolves "ore" against the previous topic context.
- **Active investigation threads**: Threads the player is currently following
  influence topic suggestion priority.

**CR-4: Topic Suggestion Generation**

Topic suggestions are generated fresh each time the player opens conversation:

1. **Domain-based** (always present): One suggestion per NPC domain, phrased
   as a natural question: "Ask about your craft" → "What do you know about
   smithing?"
2. **Thread-based** (if applicable): For each active investigation thread
   that intersects with this NPC's knowledge (i.e., the thread references
   an entity this NPC has facts about): "Ask about the old mine."
3. **Entity-based** (if applicable): For recently mentioned entities in
   conversation history: "Ask about Captain Veren."
4. **Capped at 6 suggestions total**. Priority: thread-based > entity-based >
   domain-based.

### States and Transitions

| State | Meaning |
|-------|---------|
| `Idle` | No conversation active. Question input hidden. |
| `Active` | Conversation with NPC active. Text input visible, topic suggestions shown. |
| `Processing` | Question submitted. Topic matching in progress. Input disabled. |
| `Waiting` | Facts routed to AI Dialogue Engine. Waiting for LLM response. "Thinking" indicator shown. |
| `Response` | NPC response displayed. Input re-enabled for follow-up. |

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| NPC Knowledge Graph | Question System reads knowledge graph | Queries NPC domains and facts for topic matching and suggestion generation |
| AI Dialogue Engine | Question System routes matched facts | Sends matched facts + player question text to AI Dialogue Engine for prompt construction |
| Thread Investigation | Question System reads active threads | Thread-based topic suggestions; question topics may create new threads |
| Dialogue UI | Dialogue UI reads Question System state | Displays text input, topic suggestions, conversation history |

## Formulas

No gameplay formulas. Topic relevance scoring: `score = (keyword_matches / total_keywords) + (entity_reference_bonus × 0.5)`. Threshold: 0.3. Entity reference bonus: 1.0 if any entity name detected in question.

## Edge Cases

- **If the player types an empty question or only whitespace**: Input rejected. Text field shakes briefly. Submit button disabled until non-whitespace content exists.
- **If no domains match the question (all scores < 0.3)**: All domains queried at low relevance. AI Dialogue Engine prompt includes all facts at current trust level but with a system note: "The player asked about [question]. This is outside your main expertise. Deflect naturally."
- **If the player asks the exact same question twice**: The conversation history contains the previous exchange. Topic matching returns the same facts. The LLM naturally varies the response (temperature).

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| NPC Knowledge Graph | Question System depends on Knowledge Graph | Reads domains and facts for topic matching. Hard. |
| AI Dialogue Engine | Question System depends on AI Dialogue Engine | Routes matched facts + question for response. Hard. |
| Thread Investigation | Question System depends on Thread Investigation | Reads active threads for topic suggestions. Soft. |
| Dialogue UI | Dialogue UI depends on Question System | Displays input field, suggestions, conversation state. Hard. |

## Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `relevance_threshold` | 0.3 | 0.1–0.6 | Lower = more domains matched, broader queries. Higher = stricter matching, more deflections. |
| `max_topic_suggestions` | 6 | 3–10 | More suggestions = more guidance, more UI clutter |
| `entity_reference_bonus` | 1.0 | 0.5–2.0 | Higher = entity mentions dominate domain matching more strongly |
| `conversation_history_length` | 3 | 1–5 | More history = better context resolution, more memory |

## Visual/Audio Requirements

| Event | Visual Feedback | Audio Feedback |
|-------|----------------|---------------|
| Conversation started | Text input slides up from bottom; topic suggestion chips animate in (staggered, 100ms each) | Soft parchment-unfurl sound |
| Question submitted | Text input briefly pulses amber; suggestions hide; input disabled (greyed out) | Subtle quill-scratch sound |
| Topic suggestion clicked | Chip highlights amber; text fills into input field with brief flash | Soft click |
| Empty question rejected | Text field shakes horizontally (3 oscillations, 150ms) | Low thud |
| Response received | Input re-enabled; suggestions refresh with updated context | None |

## Acceptance Criteria

- **GIVEN** an NPC with domains `["craft_metallurgy", "trade_commerce"]`, **WHEN** the player types "What do you know about the ore from the old mine?", **THEN** the topic matcher scores `craft_metallurgy` above threshold based on keyword overlap ("ore", "mine") and routes matching facts to the AI Dialogue Engine.

- **GIVEN** the same NPC, **WHEN** the player types "What happened to the old empire?", **THEN** no domain scores above threshold (NPC doesn't have `old_empire_lore`) and the system routes with fallback — all facts at current trust level with a deflection instruction.

- **GIVEN** a conversation with an NPC, **WHEN** the dialogue UI opens, **THEN** 3–6 topic suggestions are displayed based on the NPC's domains and the player's active investigation threads.

- **GIVEN** the player clicks a topic suggestion chip, **WHEN** the chip is clicked, **THEN** the suggestion text fills the input field and the player can edit before submitting.

- **GIVEN** empty input submission, **WHEN** the player attempts to submit, **THEN** the input is rejected with a shake animation.

- **Performance**: Topic matching completes within 5ms. Topic suggestions generate within 10ms. No frame drops during question processing.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should the player have a "notebook" of suggested questions collected from conversation (things NPCs explicitly told them to ask others about)? | UX Designer | Before vertical slice |
| Should there be a cooldown on repeated questions to the same NPC to prevent LLM inference spam? | Game Designer | Before MVP implementation |
# Thread Investigation

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 4 (Earned Revelation — threads ARE the path to revelation)

## Summary

Thread Investigation replaces the traditional quest log with a player-driven
web of leads and connections. Threads are created from NPC fact references
(`leads_to` hints and entity references), tracked across NPCs, and resolved
when the player connects enough fragments to reach a revelation. Contradictions
between NPC facts are detected and flagged as inconsistencies to investigate.
No quest markers, no objective lists — just the web the player weaves.

> **Quick reference** — Layer: `Feature` · Priority: `MVP` · Key deps: `Question System`, `Trust & Relationship`

## Overview

Thread Investigation is the mechanical heart of Earned Revelation. It does not
tell the player where to go or what to do. It records what the player has
discovered and helps them see connections — but the connections are made by
the player, not the system.

The system manages:

- **Thread creation**: When an NPC reveals a fact with `leads_to` hints or
  entity references, a thread is created: "Ask [target NPC] about [topic]."
  The player can also manually create threads from their own observations.
- **Thread tracking**: Active threads are tracked by the system. The player
  can view all active threads, mark threads as prioritized, and see which
  threads reference which NPCs. Threads are organized by region and by the
  investigation chain they belong to.
- **Cross-referencing**: When two facts from different NPCs reference the
  same entity, the system detects the overlap and surfaces it: "Both the
  blacksmith and the hermit mentioned the old mine." When two facts
  contradict each other (one has `contradicts: [other_fact_id]`), the
  system flags the inconsistency.
- **Thread resolution**: A thread is resolved when the player discovers the
  fact it pointed to, OR when the player manually marks it as resolved
  (dead end, NPC unavailable, etc.). Resolved threads move to the player's
  investigation history — a record of what they've learned and from whom.
- **Revelation detection**: When the player has discovered enough connected
  facts to form a conclusion, the system detects the cluster and flags it
  as a potential revelation: "You have evidence linking Captain Veren to
  Aldric's death. Confront him?" The player decides when to act.

Without Thread Investigation: the player has no way to track what they've
learned. Facts discovered from NPCs are forgotten between sessions. Cross-
references between NPCs are invisible. The investigation loop collapses
into "talk to everyone and hope you remember."

## Player Fantasy

Thread Investigation makes the player feel like a detective. Not a detective
in a game that tells you exactly where to go — a real detective who builds
their own web of leads and experiences the satisfaction of watching fragments
click into place.

**The web is yours.** The player decides which threads to follow, which to
deprioritize, which to mark as dead ends. The system helps track what you've
learned, but it never tells you what to do next. When you connect two facts
from different NPCs and realize they're about the same thing — that's your
discovery, not the game's.

**Contradictions are clues, not bugs.** When the blacksmith says the ore is
pure and the hermit says it's contaminated, the system flags the contradiction
but doesn't resolve it. The player investigates: who is right? Is someone
lying? Or are both seeing different sides of the same truth? Contradictions
are the engine of investigation — they're what make you question what you
think you know.

**Progress is visible.** The investigation journal (owned by Investigation
Journal system) shows the web growing. Threads connect to threads. Facts
connect to facts. The player sees their understanding of the world deepening
in real time — not as a checklist of completed objectives, but as a network
of discovered truths.

## Detailed Design

### Core Rules

**CR-1: Thread Creation**

A thread is created when:

| Trigger | Thread Content | Auto/Manual |
|---------|---------------|-------------|
| NPC fact with `leads_to` hint | "Ask [target NPC] about [topic]" — includes the hint text from the fact | Auto |
| NPC mentions an entity the player hasn't met | "Find [entity name] — mentioned by [NPC name]" | Auto |
| Player manually creates a thread | Player types their own thread: "Investigate the symbol on the Captain's desk" | Manual |
| Contradiction detected between two discovered facts | "Resolve contradiction: [NPC A] says [X], but [NPC B] says [Y]" | Auto |

Each thread has:

| Field | Description |
|-------|-------------|
| `thread_id` | Unique identifier |
| `source` | Which NPC/fact/observation created this thread |
| `target_entity` | Who or what the thread points to (may be null for open-ended threads) |
| `topic` | What to investigate/ask about |
| `status` | `active`, `prioritized`, `resolved`, `dead_end` |
| `created_at` | Game timestamp |
| `chain_id` | Which investigation chain this thread belongs to (auto-assigned by entity overlap) |

**CR-2: Investigation Chains**

Threads that reference the same entities or share `leads_to` targets are
grouped into investigation chains. A chain is a set of related threads
that together form a coherent line of inquiry.

- Threads are auto-assigned to a chain when they share at least one entity
  reference with an existing thread in that chain.
- Chains can merge: if two chains are discovered to be about the same thing
  (e.g., "The Strange Ore" chain and "Aldric's Death" chain both reference
  the old mine), they merge into one.
- Chains have a summary: the system generates a one-line description of
  what the chain is about based on the facts discovered so far. "You are
  investigating Aldric's death. Evidence points to Captain Veren."

**CR-3: Cross-Referencing**

The system continuously checks for connections between discovered facts:

- **Entity overlap**: When a newly discovered fact references an entity
  that also appears in a fact from a different NPC, the system flags the
  connection: "[NPC A] and [NPC B] both mentioned [entity]."
- **Contradiction detection**: When a discovered fact has `contradicts:
  [other_fact_id]` and the player has also discovered that other fact,
  the system creates a contradiction thread (CR-1) and flags both facts.
- **Revelation threshold**: When 3+ facts in the same chain all point to
  the same conclusion (same `leads_to` target, or all `contradicts` the
  same official story), the system suggests a revelation is available.

**CR-4: Thread Resolution**

A thread is resolved when:

| Resolution Type | Trigger | Effect |
|----------------|---------|--------|
| `discovered` | Player discovers the fact the thread pointed to | Thread moves to investigation history; fact is recorded as discovered |
| `confronted` | Player acts on accumulated evidence (e.g., confronts Veren) | All threads in the chain resolve; major revelation flagged |
| `dead_end` | Player manually marks thread as unresolvable (NPC dead, location destroyed, etc.) | Thread moves to history with dead-end marker; may be reactivated if new evidence emerges |
| `merged` | Thread's target was reached via a different thread | Thread resolves as merged; no duplicate tracking |

**CR-5: Player Knowledge Component**

The player entity carries a `PlayerKnowledge` component (Entity Framework)
that stores all discovered facts, active threads, resolved threads, and
investigation chains. This is what persists across save/load.

```
PlayerKnowledge:
  discovered_facts[]: {fact_id, source_npc, discovered_at}
  active_threads[]: {thread_id, status, chain_id}
  resolved_threads[]: {thread_id, resolution_type, resolved_at}
  chains[]: {chain_id, summary, facts[], status}
  known_entities[]: {entity_id, discovered_at, discovered_via}
  revelations[]: {revelation_id, chain_id, discovered_at, confronted_at}
```

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Question System | Thread Investigation reads discovered facts | When player asks a question that reveals a new fact, Thread Investigation checks for thread creation and cross-referencing |
| Trust & Relationship | Thread Investigation triggers trust events | Resolving threads that involve NPCs may trigger trust gains (Trust & Relationship CR-2) |
| NPC Knowledge Graph | Thread Investigation reads fact data | Reads `leads_to` hints and `contradicts` references for thread creation and contradiction detection |
| Investigation Journal | Investigation Journal reads thread state | Displays active threads, chains, and revelations to the player |
| Entity Framework | Thread Investigation reads/writes `PlayerKnowledge` component | All investigation state stored on player entity |
| Save/Load System | Save/Load serializes PlayerKnowledge | Investigation state persists across sessions |

## Formulas

No mathematical formulas. Thread creation and contradiction detection are
rule-based, not computed. The revelation threshold is a simple count: 3+
connected facts pointing to the same conclusion.

## Edge Cases

- **If a thread's target NPC dies before the player reaches them**: The thread auto-resolves as `dead_end` with the reason recorded. Other threads in the chain may still be resolvable through alternative NPCs.
- **If the player discovers the same fact via two different threads**: Both threads resolve as `merged`. The fact is recorded once. No duplicate tracking.
- **If a contradiction is detected but one of the contradicting facts is a RUMOR**: The contradiction is flagged as lower confidence. The rumor may be false. The player is told: "[NPC A] claims [X], but [NPC B] says [Y] — [NPC B]'s account may be a rumor."
- **If the player manually creates a thread that duplicates an auto-created thread**: The system detects the duplicate (same target entity + topic) and merges them.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Question System | Thread Investigation reads discovered facts | Receives fact discovery events for thread creation. Hard. |
| NPC Knowledge Graph | Thread Investigation reads fact data | Reads `leads_to`, `contradicts`, and entity references. Hard. |
| Trust & Relationship | Thread Investigation triggers trust events | Thread resolution fires trust gain events. Soft. |
| Investigation Journal | Investigation Journal reads thread state | All thread data displayed to player. Hard. |
| Entity Framework | Thread Investigation reads/writes PlayerKnowledge | Investigation state persisted on player entity. Hard. |

## Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `revelation_threshold` | 3 facts | 2–5 | Lower = revelations trigger faster, less investigation required |
| `max_active_threads` | 20 | 10–50 | More threads = more player choice, more UI clutter |
| `auto_chain_merge` | true | bool | Whether chains auto-merge on entity overlap |

## Acceptance Criteria

- **GIVEN** a fact with `leads_to: [{entity_id: orin_id, topic_hint: "Ley-touched metal"}]`, **WHEN** the player discovers that fact, **THEN** an active thread "Ask Orin about Ley-touched metal" is created.

- **GIVEN** two discovered facts from different NPCs that both reference the same entity, **WHEN** the second fact is discovered, **THEN** the system flags the entity overlap: "Both [NPC A] and [NPC B] mentioned [entity]."

- **GIVEN** a discovered fact with `contradicts: [other_fact_id]` where the other fact is also discovered, **WHEN** the second fact is discovered, **THEN** a contradiction thread is created and both facts are flagged.

- **GIVEN** 3 connected facts pointing to the same conclusion in one chain, **WHEN** the third fact is discovered, **THEN** the system suggests a revelation is available.

- **GIVEN** a thread's target NPC dies, **WHEN** the death event fires, **THEN** the thread auto-resolves as `dead_end` with the reason recorded.

- **Performance**: Thread creation completes within 5ms of fact discovery. Cross-reference check across 100 discovered facts completes within 1ms.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should the player be able to share threads/evidence with party members for their commentary? | Game Designer | Before vertical slice |
| Should there be a "theory board" where the player can manually draw connections between facts, or is auto-detection sufficient? | UX Designer | Before Investigation Journal GDD |
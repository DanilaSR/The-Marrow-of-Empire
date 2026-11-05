# Investigation Journal

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 4 (Earned Revelation — the journal is where the player sees their understanding grow)

## Summary

The Investigation Journal is the player-facing interface for the Thread
Investigation system. It displays active threads, investigation chains,
discovered facts, contradictions, and revelations — not as a quest log
with objectives and markers, but as a web of leads the player has woven.
Parchment-themed UI matching the art bible's desk-lamp aesthetic.

> **Quick reference** — Layer: `Presentation` · Priority: `MVP` · Key deps: `Thread Investigation`

## Overview

The Investigation Journal is where the player goes to think. It's the
investigator's desk — a parchment-and-ink interface that displays everything
the player has discovered and helps them see connections. It does not tell
the player where to go. It shows them what they know and lets them decide
what to pursue.

The journal provides:

- **Thread view**: All active threads, organized by investigation chain.
  Each thread shows: what to investigate, which NPC mentioned it, when it
  was created. Threads can be prioritized (pinned to top) or marked as
  dead ends. Clicking a thread centers the map on the relevant location
  or NPC.
- **Chain view**: Investigation chains — groups of related threads. Each
  chain shows a summary, the facts discovered so far, and how many threads
  remain active. Chains that have reached revelation threshold are
  highlighted with a subtle amber glow.
- **Fact log**: Every fact the player has discovered, organized by source
  NPC. Shows the fact content, when it was discovered, and which chain
  it belongs to. Contradictions are flagged with thread-red markers.
- **Entity index**: Every person, place, and object the player has learned
  about, with links to the facts and threads that reference them. "What
  do I know about Captain Veren?" — the entity index answers that.
- **Revelation prompts**: When 3+ connected facts point to the same
  conclusion, the journal suggests: "You have evidence suggesting [X].
  Confront [NPC]?" The player decides when to act.

The journal is accessed via a dedicated key (default: J) or from the main
menu. It opens as a full-screen overlay with the game world visible but
dimmed behind it. The camera freezes in Journal state.

Without the Investigation Journal: the Thread Investigation system tracks
everything, but the player can't see it. The web exists in code but not
in the player's experience.

## Player Fantasy

**This is your desk.** The journal feels like the investigator's private
workspace — parchment, ink, the quiet of ordered thought. It's the visual
break from the world that the art bible promises (Section 3, State 5):
"the desk lamp, ordered calm." When the player opens the journal, they're
not pausing the game — they're doing the work of investigation.

**The web is visible.** The player sees their investigation growing — a
thread here, a fact there, chains forming as connections emerge. This is
the visual reward for investigation. A full journal with connected chains
and resolved revelations is the player's trophy. It says: "I figured this
out."

**Contradictions demand attention.** Thread-red markers on contradictory
facts pull the eye. The player can't unsee them. They create the same
tension in the journal that they create in the investigation: something
is wrong here, and you need to find out what.

## Detailed Design

### Core Rules

**CR-1: Journal Layout**

The journal has three panels:

| Panel | Position | Content |
|-------|----------|---------|
| Chain list | Left sidebar (30% width) | All investigation chains, sorted by most recently updated. Active chains at top. Each entry shows chain summary + thread count. |
| Detail view | Center (70% width) | Selected chain's threads, facts, and contradictions. Or selected entity's details. |
| Entity quick-ref | Bottom bar | Recently referenced entities as clickable chips. "Captain Veren" "Old Mine" "Aldric" |

**CR-2: Thread Display**

Each thread in the detail view shows:

- Thread description: "Ask Orin about Ley-touched metal"
- Source: "Mentioned by Greta (Blacksmith)"
- Status indicator: active (amber dot), prioritized (amber star), dead end (grey x), resolved (green check)
- Age: "Discovered 2 days ago" (game time)
- Click: centers map on target entity (if location known), or opens entity's journal entry

**CR-3: Chain Display**

Each chain in the sidebar shows:

- Chain title: auto-generated summary (e.g., "Aldric's Death")
- Thread count: "3 active leads"
- Fact count: "7 facts discovered"
- Status: active, revelations_available (amber glow), resolved
- Click: opens chain detail in center panel

**CR-4: Fact Log**

The fact log (accessible via tab in the detail view) shows all discovered
facts for the selected chain:

- Fact content (as displayed to the player when discovered)
- Source NPC and trust level at time of discovery
- Timestamp (game time)
- Contradiction flag (thread-red marker if this fact contradicts another)
- Entity references (clickable chips)

**CR-5: Contradiction Display**

Contradictory facts are displayed side by side with a thread-red connector:

```
[NPC A] says: "The ore is pure — finest I've seen."
   ↕ CONTRADICTS
[NPC B] says: "The ore is contaminated. It's poisoning the river."
```

The contradiction creates an automatic thread: "Resolve contradiction:
ore quality." The player can't dismiss the contradiction without resolving
the thread or marking it as a dead end.

**CR-6: Revelation Prompts**

When a chain reaches revelation threshold (3+ connected facts pointing to
the same conclusion), the chain entry glows amber and a prompt appears:

> "You have evidence linking [conclusion]. Confront [target NPC]?"

The player can dismiss the prompt (continue investigating) or act on it
(trigger the confrontation). Dismissed prompts reappear when the journal
is next opened.

### States and Transitions

| State | Meaning |
|-------|---------|
| `Closed` | Journal not visible. Default exploration state. |
| `Open` | Journal full-screen. Camera frozen. World dimmed behind parchment overlay. |
| `ChainSelected` | A specific chain is selected in the sidebar. Detail panel shows chain content. |
| `EntitySelected` | An entity is selected (from fact reference or quick-ref bar). Detail shows entity's facts and connections. |

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Thread Investigation | Journal reads thread state | Displays threads, chains, facts, contradictions |
| Camera System | Journal triggers camera state | Camera enters Journal state (frozen) |
| Entity Framework | Journal reads entity data | Entity names, types for entity index and references |
| Map System | Journal may trigger map centering | Clicking a thread with a known location centers the map |

## Visual/Audio Requirements

📌 **Asset Spec** — UI system, Visual/Audio required.

| Event | Visual Feedback | Audio Feedback |
|-------|----------------|---------------|
| Journal opened | Full-screen parchment overlay slides in from bottom (300ms); world dims to 30% opacity behind it; chain list animates in from left | Parchment unfurl; ambient mutes |
| Journal closed | Reverse of open; slides out (200ms) | Parchment roll-up; ambient restores |
| New thread added | Brief amber pulse on the chain entry; "New" badge (fades after 10s) | Soft quill-scratch |
| Contradiction detected | Thread-red pulse on the chain entry; red connector line draws between contradictory facts | Sharp string accent (single tense note) |
| Revelation threshold reached | Chain entry glows amber; gentle pulsing until acted upon | Rising chime (soft, expectant) |
| Thread resolved | Thread fades to resolved state; brief green flash on chain entry | Soft paper-fold sound |

The journal uses the art bible's UI palette (Section 2.4): parchment `#E3D5B0`,
ink `#1C1813`, thread-red `#C4483A`, amber `#D9A441`.

## Acceptance Criteria

- **GIVEN** 3 active threads across 2 chains, **WHEN** the player opens the journal, **THEN** both chains are displayed in the sidebar with thread counts, sorted by most recently updated.

- **GIVEN** a chain with 3 connected facts pointing to the same target, **WHEN** the third fact is discovered, **THEN** the chain entry glows amber and a revelation prompt appears.

- **GIVEN** two contradictory facts, **WHEN** the player views the chain detail, **THEN** the contradictory facts are displayed side by side with a thread-red connector and an auto-created resolution thread.

- **GIVEN** the journal open, **WHEN** the player clicks a thread that references a known location, **THEN** the map centers on that location.

- **Performance**: Journal opens within 200ms. Chain detail renders within 50ms for up to 20 threads.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should the player be able to manually draw connections between facts (a "theory board"), or is auto-detection sufficient? | UX Designer | Before vertical slice |
| Should the journal support text search across all discovered facts? | UX Designer | Before vertical slice |
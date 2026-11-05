# Faction System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 2 (The World Remembers — factions react to player actions and shift over time)

## Summary

[To be designed]

> **Quick reference** — Layer: `Feature` · Priority: `Alpha` · Key deps: `Persistent World State`, `Trust & Relationship`

## Overview

The Faction System governs the political landscape of The Marrow of Empire —
the factions that control the borderland, their relationships with each other,
and their disposition toward the player. Each faction has a standing value with
the player (hostile to allied), internal power dynamics, and territorial claims.
Player actions — who they help, who they fight, what secrets they reveal — shift
faction standing, and those shifts propagate through the world. A faction that
trusts the player offers access, information, and safe passage. A faction that
fears the player may attack on sight. The Faction System makes the borderland's
political reality legible and responsive — not a static reputation meter, but a
living web of alliances and enmities that the player navigates with every choice.

> **Quick reference** — Layer: `Feature` · Priority: `Alpha` · Key deps: `Persistent World State` (reads/writes `world.faction.*` flags), `Trust & Relationship` (trust events may trigger faction standing changes) · Alpha scope: `multi-region faction dynamics, reputation propagation, territory control`

## Player Fantasy

The Faction System makes the player feel that **the borderland is a political
reality, and you are a force within it.**

Factions are not reputation meters with shops attached. They are organisations
with goals, enemies, and territory. When the player helps the Thornwell militia
hold a pass, the Marauder faction notices — and adjusts. When the player
discovers a secret that could destabilise a faction leader, the choice of
whether to reveal it is political. The player is not just an investigator
wandering through — they are an actor in the borderland's power struggles,
whether they intend to be or not.

**Reference**: Fallout: New Vegas (faction reputation as core narrative engine),
Tyranny (faction choices reshape the world and unlock exclusive paths),
Mount & Blade (dynamic faction territory and relationships).

## Detailed Design

### Core Rules

**CR-1: Faction Definition**

A faction is a named group with:
- **Faction ID**: Unique string identifier (e.g., `thornwell_militia`, `marauders`,
  `empire_remnant`, `free_settlements`).
- **Display Name**: Localizable name for UI and dialogue.
- **Standing Range**: `-100` (sworn enemy) to `+100` (honoured ally), with discrete
  tiers: Hostile (−100 to −51), Unfriendly (−50 to −1), Neutral (0 to 24),
  Friendly (25 to 74), Allied (75 to 100).
- **Relationships**: Standing values with other factions (same scale). Faction A's
  relationship with Faction B affects how Faction A reacts to player actions that
  benefit or harm Faction B.
- **Territory**: Regions or sub-regions the faction controls (Alpha-tier: factions
  control named territories; control shifts propagate to the Map System).
- **Home Region**: The faction's primary region. The faction is always present there.

Faction data is authored per region in `assets/factions/[faction_id].yaml`.

**CR-2: Player Standing**

The player has a standing value with each faction, stored in the entity's
`Faction` component (EF component #7). Standing changes via events:
- **Quest/thread resolution**: Resolving a thread that benefits a faction → +10
  to +30 depending on significance.
- **Combat**: Killing faction members → −5 per enemy, −20 for leaders.
- **Dialogue choices**: Major dialogue decisions (betraying trust, sharing
  secrets, making promises) → ±5 to ±15.
- **Faction event propagation**: A faction the player is allied with (+75)
  grants +10 bonus standing with that faction's allies and −10 penalty with
  that faction's enemies.

Standing tier thresholds determine NPC behavior:
- **Allied (75+)**: NPCs offer discounts, share secrets freely, may give
  quests without trust checks.
- **Friendly (25–74)**: NPCs are cooperative, trust checks at normal thresholds.
- **Neutral (0–24)**: Default. NPCs are cautious; trust must be earned normally.
- **Unfriendly (−50 to −1)**: NPCs are guarded; trust checks have penalty (−1 level).
- **Hostile (−100 to −51)**: NPCs may refuse dialogue, attack on sight, or require
  significant proof before engaging.

**CR-3: Faction Event Propagation**

When the player's standing with a faction changes significantly, the change
propagates to related factions:
- **Allies of the affected faction**: receive a standing shift of +50% of the
  player's change (same direction).
- **Enemies of the affected faction**: receive a standing shift of −50% of the
  player's change (opposite direction).
- **Neutral factions**: no automatic change, but may react to world events
  (e.g., a faction gaining power in a region triggers a standing adjustment).

Propagation is capped: no single event can cause more than ±15 standing through
propagation alone.

**CR-4: Territory Control (Alpha)**

Factions control named territories within regions. Territory changes hands via
major world events — a faction leader killed, a stronghold captured, a treaty
signed. Territory ownership is stored as world state flags
(`world.faction.[faction_id].controls.[territory_id]`). The Map System reads
territory flags to display faction borders.

**CR-5: Faction Perception**

NPCs within a faction are aware of the player's standing with their faction:
- NPC dialogue may reference standing ("You've been a friend to the militia")
  or ("The Marauders don't forget").
- Shop prices scale with standing: Allied = −25% cost, Hostile = +100% or
  refuse service.
- Faction guards at territory borders may grant or deny passage based on standing.

### States and Transitions

Faction standing is continuous (integer −100 to +100) with discrete tier
boundaries. Tier transitions (e.g., Neutral→Friendly at standing 25) trigger
a `faction_tier_changed` event that other systems can subscribe to.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Persistent World State | Mutual | Faction System reads/writes `world.faction.*` flags. Standing changes are submitted as `faction_event` world events. Territory changes recorded as `world.faction.[id].controls.[territory]` flags. |
| Trust & Relationship | Coordinates | Major trust events may trigger faction standing adjustments. Faction standing modifies trust checks (Friendly = normal, Hostile = −1 trust level penalty). |
| Entity Framework | Reads/writes `Faction` component | Each NPC carries a `Faction` component with `faction_id` and `standing`. Player entity's faction standings stored in the same component. |
| AI Dialogue Engine | Reads faction standing | NPC dialogue may reference player's faction reputation. Hostile NPCs may refuse dialogue. |
| Tactical Combat Engine | Reads faction standing | Hostile factions may attack on sight. Allied factions may assist in combat if present. |
| Map System | Reads territory control | Map displays faction territories and borders based on `world.faction.*.controls.*` flags. |
| Party System | Reads faction standing | Companions with strong faction affiliations may react to standing changes (departure risk if player becomes Hostile to their faction). |

## Formulas

**Standing Propagation**:
```
ally_shift = round(player_standing_change × 0.5)
enemy_shift = round(player_standing_change × −0.5)
clamp(ally_shift, −15, +15)
clamp(enemy_shift, −15, +15)
```

**Shop Price Modifier**: `price_multiplier = 1.0 − (standing / 400)`. At Allied (+100): 0.75×. At Hostile (−100): 1.25×. Refuse service at −76 and below.

## Edge Cases

- **If player standing with a faction changes while in conversation with a member
  of that faction**: The change is applied but the NPC's behavior in the current
  conversation does not change. The new standing takes effect for subsequent
  interactions.
- **If the player is Allied with two factions that become enemies**: The player
  is notified of the conflict. Standing with both is unchanged, but further
  actions that benefit one will penalise the other at 2× the normal rate until
  the player chooses a side (or brokers peace).
- **If a faction is destroyed** (all leaders killed, territory lost): The
  faction is marked `defeated`. Standing becomes irrelevant. NPCs from that
  faction become factionless or join allied factions.
- **If faction propagation would push standing below −100 or above +100**:
  Clamp at the boundary. Excess is discarded — no overflow.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Persistent World State | Mutual | Faction reads/writes `world.faction.*` flags. Standing events submitted as world events. **Hard.** |
| Trust & Relationship | Coordinates | Trust events may trigger standing changes; standing modifies trust checks. Soft. |
| Entity Framework | Reads/writes `Faction` component | **Hard.** |
| AI Dialogue Engine | Reads faction standing | Soft. |
| Tactical Combat Engine | Reads faction standing | Soft. |
| Map System | Reads territory data | Soft. |
| Party System | Reads faction standing for companion reactions | Soft. |

## Tuning Knobs

| Parameter | Default | Safe Range | Notes |
|-----------|---------|------------|-------|
| `standing_range` | −100 to +100 | ±50 to ±200 | Wider range = more granular reputation |
| `propagation_multiplier` | 0.5 | 0.0–1.0 | 0 = no propagation; 1 = full mirror |
| `propagation_cap` | ±15 | ±5 to ±50 | Max standing change from propagation per event |
| `hostile_refuse_service_threshold` | −76 | −100 to −25 | Standing below this → shops refuse service |
| `tier_thresholds` | [−51, −1, 25, 75] | configurable per faction | Tier boundary values |

## Visual/Audio Requirements

| Event | Visual | Audio |
|-------|--------|-------|
| Tier increase (e.g., Neutral→Friendly) | Brief faction emblem glow; notification toast | Positive sting |
| Tier decrease (e.g., Friendly→Unfriendly) | Faction emblem dims; notification toast | Negative sting |
| Faction territory change | Map update animation | — |
| Faction destroyed | Faction emblem shatters in UI | Sombre tone |

## UI Requirements

📌 **UX Flag — Faction System**: This system has UI requirements.

- **Faction screen** (Alpha, deferred from MVP): Accessible from journal or main
  menu. Shows all known factions with standing bars, tier labels, territory map
  overlay, and relationship web (who is allied/enemies with whom).
- **Faction reputation HUD element**: Minimal — current region's dominant faction
  standing shown as small icon + colour indicator in the HUD corner during
  exploration.
- **Dialogue indicators**: NPC dialogue UI shows faction affiliation (icon +
  standing tier colour).

## Acceptance Criteria

- **GIVEN** player standing 0 (Neutral) with Thornwell Militia, **WHEN** a thread
  resolution awards +30 standing, **THEN** standing becomes 30 (Friendly) and a
  tier-change notification fires.

- **GIVEN** player at Allied (+80) with Faction A, which is Allied with Faction B,
  **WHEN** player gains +20 standing with Faction A, **THEN** player gains +10
  (50% of 20, capped at ±15) standing with Faction B.

- **GIVEN** player at Hostile (−80) with a faction, **WHEN** attempting to trade
  with a shopkeeper of that faction, **THEN** the shopkeeper refuses service
  (standing below −76 threshold).

- **GIVEN** a faction leader is killed in combat, **WHEN** the death event fires,
  **THEN** the faction's territory control flag for that leader's territory is
  cleared and a `faction_event` is submitted to PWS.

- **GIVEN** faction propagation would produce a shift of +25, **WHEN** the
  propagation cap is ±15, **THEN** the applied shift is clamped to +15.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should factions have unique gameplay mechanics (e.g., Marauders raid player camps, Empire Remnant offers Ley-tech), or are they primarily narrative/reputation entities? | Game Designer + Creative Director | Before Alpha implementation |
| Can the player create or lead a faction (stronghold-style), or is faction leadership NPC-only? | Creative Director | Before Alpha implementation |
| Should faction territory control affect wildlife spawning and NPC schedules (e.g., Marauder-controlled forests are more dangerous)? | Game Designer + World Builder | Before Alpha implementation |

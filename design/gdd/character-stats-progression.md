# Character Stats & Progression

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 3 (Tactical Depth — stats drive meaningful build choices)

## Summary

The Character Stats & Progression system defines the six-attribute model
(STR/DEX/CON/INT/WIS/CHA, 3–18 range), derived stat calculations, point-buy
character creation (30 points, max 16 starting), and level progression (1–20,
XP from threads and combat). Attributes ARE the class — every stat serves both
combat and investigation, and the spread defines the build. The system writes
to the Entity Framework Stats component; all other systems read from it.

> **Quick reference** — Layer: `Core` · Priority: `MVP` · Key deps: `Entity Framework`

## Overview

The Character Stats & Progression system defines what a character IS in
mechanical terms. It provides the attribute model, derived stat calculations,
and level progression that make every character — player, party member, NPC,
enemy — a distinct tactical entity with strengths, weaknesses, and growth.

The system defines:

- **Primary attributes**: Six core attributes that define a character's
  physical, mental, and social capabilities. Every character has these;
  the values differentiate the warrior from the scholar.
- **Derived stats**: Values calculated from primary attributes — health,
  stamina/action points, carrying capacity, dialogue bonuses. These are
  what combat, exploration, and conversation systems actually read.
- **Progression**: XP acquisition, level advancement, attribute increases,
  and the curve that governs how characters grow from vulnerable newcomers
  to seasoned investigators.
- **Build identity**: The attribute spread defines a character's role —
  high Strength means melee damage and carrying capacity; high Intelligence
  means investigation bonuses and lore comprehension. No class system:
  attributes ARE the class. A character with 16 STR and 14 INT is a
  warrior-scholar. A character with 8 STR and 16 CHA is a silver-tongued
  diplomat who needs bodyguards.

Without this system: combat has no damage, no hit chance, no health. Dialogue
has no persuasion. Investigation has no comprehension checks. Every character
is mechanically identical — and that's not an RPG.

## Player Fantasy

Character stats are the player's first and most personal act of creation in
The Marrow of Empire. Before they talk to a single NPC, before they follow a
single thread, they decide: *who am I in this world?*

**Build identity — attributes ARE the class.** In a game without rigid class
archetypes, the six attributes define who you are. High Intelligence doesn't
just mean more skill points — it means you notice contradictions in NPC
testimony. High Charisma doesn't just mean better prices — it means NPCs
volunteer knowledge they'd hide from a less compelling investigator. Every
attribute has a combat use AND an investigation use — no stat is a dump stat
in a game where conversation and combat are equally core.

**Tactical mastery through build choices.** Pillar 3 demands that combat
reward mastery. Build choices are the first layer of mastery — the player who
understands that high Dexterity means more action points and earlier turn
order has already started thinking tactically before combat begins.

**Progression feels earned.** Levels are not handed out for attendance. XP
comes from uncovering truths, resolving threads, and winning tactical
encounters. When a character levels up, it's because they investigated well
or fought smart — the number going up is a record of accomplishment, not
a participation trophy.

**Reference**: Baldur's Gate 1-2 (six-attribute D&D spread, meaningful
attribute thresholds), Divinity Original Sin 2 (attributes as build identity
without rigid classes), Disco Elysium (mental/social stats as investigation
tools — INT and CHA matter outside combat).

## Detailed Design

### Core Rules

**CR-1: Primary Attributes**

Six attributes define every character. The range is 3–18 for mortals; 10 is the
human baseline (average, untrained). No class system — attribute values ARE the
build.

| Attribute | Abbrev | Governs (Combat) | Governs (Investigation/Social) |
|-----------|--------|------------------|-------------------------------|
| **Strength** | STR | Melee damage bonus, carrying capacity, heavy weapon requirements | Physical intimidation, forcing doors/objects |
| **Dexterity** | DEX | Action points, turn order, ranged accuracy, dodge/evasion | Lockpicking, trap disarming, stealth |
| **Constitution** | CON | Maximum health, health per level, stamina, resistance to poison/disease | Endurance (travel speed, exhaustion resistance) |
| **Intelligence** | INT | Magical/technological aptitude (Ley device operation), critical thinking under pressure | Contradiction detection in NPC testimony, lore comprehension, ancient language translation |
| **Wisdom** | WIS | Willpower saves, perception of hidden enemies | Insight (detecting lies/omissions), environmental clue detection, awareness of being followed |
| **Charisma** | CHA | Party morale, companion combat bonuses | NPC trust thresholds, persuasion, barter prices, how much knowledge NPCs volunteer unprompted |

Every attribute serves both combat and investigation. No dump stats in a game
where conversation and combat are equally core.

**CR-2: Attribute Modifiers**

Each attribute generates a modifier used by all derived calculations. The modifier
table uses even-number thresholds:

| Attribute Value | Modifier | Descriptor |
|----------------|----------|------------|
| 3 | −4 | Crippled |
| 4–5 | −3 | Severely Impaired |
| 6–7 | −2 | Weak |
| 8–9 | −1 | Below Average |
| 10–11 | +0 | Average |
| 12–13 | +1 | Above Average |
| 14–15 | +2 | Gifted |
| 16–17 | +3 | Exceptional |
| 18 | +4 | Peak Human |

`Modifier = floor((Attribute - 10) / 2)`, clamped to [−4, +4].

**CR-3: Derived Stats**

| Derived Stat | Formula | Purpose |
|-------------|---------|---------|
| **Maximum Health** | `20 + (CON × 2) + (Level × CON_modifier)` at level 1; `previous_max + CON_modifier` per additional level | How much damage a character can take before death |
| **Action Points (AP)** | `4 + DEX_modifier` (minimum 2) | Actions per combat turn. Move = 1 AP, standard attack = 2 AP, ability = 1–3 AP |
| **Carry Capacity** | `(STR × 10) + 30` lbs | Maximum inventory weight before encumbrance penalties |
| **Initiative** | `DEX_modifier + WIS_modifier` | Turn order bonus in combat |
| **Dialogue: Persuasion** | `CHA_modifier + 1` per 2 full investigation threads resolved | Bonus to persuasion checks in conversation |
| **Dialogue: Insight** | `WIS_modifier + 1` per 5 NPCs fully questioned | Bonus to detecting lies and omissions |
| **Investigation: Comprehension** | `INT_modifier + 1` per 3 lore fragments discovered | Bonus to understanding ancient texts and Ley technology |
| **Evasion** | `10 + DEX_modifier` | Base difficulty to hit this character in combat |

**CR-4: Character Creation**

The player character is created at the start of the game:

- **Point-buy**: 30 points to distribute across six attributes. Each attribute
  starts at 8. Raising an attribute to value N costs `N - 8` points (linear).
  Maximum starting value: 16 in any single attribute.
- **Example build (investigator)**: STR 10, DEX 12, CON 10, INT 16, WIS 14, CHA 12
  costs 2+4+2+8+6+4 = 26 points (4 remaining for customization).
- **Example build (warrior)**: STR 16, DEX 12, CON 14, INT 10, WIS 10, CHA 8
  costs 8+4+6+2+2+0 = 22 points (8 remaining).
- **Confirmation screen**: Player reviews derived stats before confirming. Can
  reallocate freely before confirming.

Party members and NPCs have fixed attribute spreads defined in their character
data. Enemies use simplified stat blocks (only the attributes relevant to their
combat role).

**CR-5: Level Progression**

XP is earned, not given:

| Source | XP Awarded |
|--------|-----------|
| Thread resolved (completed investigation chain) | 100 × thread_complexity (number of NPC hops in the chain) |
| Combat victory (all enemies defeated or fled) | 20 × total_enemy_level |
| Major revelation discovered | 200 × revelation_tier (1–3) |
| NPC trust milestone (deep knowledge unlocked) | 50 × trust_level_achieved |

Level thresholds use a linear progression:

| Level | Total XP Required | XP to Next |
|-------|-------------------|------------|
| 1 | 0 | 300 |
| 2 | 300 | 600 |
| 3 | 900 | 900 |
| 4 | 1800 | 1200 |
| 5 | 3000 | 1500 |
| ... | ... | ... |

`XP_to_next_level = Level × 300`. `Total_XP_for_level_N = 150 × (N × (N - 1))`.

**On level-up**: The character gains:
- Health increase: +CON_modifier (minimum +1)
- Every 4th level: +1 to one attribute of the player's choice
- Level cap: 20

MVP region (levels 1–5) takes approximately 9000 XP to reach level 5, which is
roughly: 6 resolved investigation chains + 15 combat encounters + 3 major
revelations + 4 trust milestones.

### States and Transitions

No runtime states. Characters have a level and an attribute array — these change
only on level-up. The Stats component on the entity holds the current values.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Reads/Writes `Stats` component | Stats component carries attributes[], derived[], level, xp. Character Stats system is the authoritative writer; all other systems read. |
| Tactical Combat Engine | Reads derived stats | Uses AP, initiative, health, evasion for combat calculations |
| Party System | Reads stats for party members | Displays party stats; uses CHA for party morale |
| AI Dialogue Engine | Reads INT, WIS, CHA for dialogue checks | Persuasion, insight, and comprehension bonuses affect conversation outcomes |
| Thread Investigation | Reads INT, WIS for investigation checks | Comprehension and insight affect thread discovery |
| Character & Inventory UI | Reads all stats for display | Character sheet, level-up screen |
| Save/Load System | Serializes via Entity Framework | Stats component data included in saves |

## Formulas

### Attribute Modifier Formula

The `attribute_modifier` formula converts a raw attribute value to its derived modifier:

`modifier = floor((attribute - 10) / 2)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| attribute | A | int | 3–18 | Raw attribute value |
| modifier | M | int | −4 to +4 | Derived modifier applied to all stat calculations |

**Output Range:** −4 (attribute 3) to +4 (attribute 18)
**Clamping:** Values below 3 are treated as 3. Values above 18 are treated as 18.
**Example:** Attribute 14 → (14−10)/2 = 2.0 → floor = +2 modifier.

### XP Progression Formula

The `xp_to_next_level` formula defines how much XP is needed to advance:

`xp_to_next = Level × 300`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| Level | L | int | 1–20 | Current character level |
| xp_to_next | X | int | 300–6000 | XP required to reach next level |

**Total XP for level N:** `Total_XP = 150 × N × (N − 1)`
**Output Range:** 0 XP (level 1) to 57,000 XP (level 20)
**Example:** Level 5 → xp_to_next = 5 × 300 = 1500. Total XP to reach level 5 = 150 × 5 × 4 = 3000.

### Maximum Health Formula

The `max_health` formula defines a character's health pool:

`MaxHealth_base = 20 + (CON × 2) + (Level × CON_mod)`
`MaxHealth_levelup = MaxHealth_previous + max(1, CON_mod)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| CON | C | int | 3–18 | Constitution attribute value |
| CON_mod | Cm | int | −4 to +4 | Constitution modifier |
| Level | L | int | 1–20 | Current character level |

**Output Range:** 26 HP (CON 3 at level 1) to 156 HP (CON 18 at level 20)
**Example:** CON 14 (+2 modifier), Level 3: 20 + 28 + (3 × 2) = 54 HP.

## Edge Cases

- **If CON_modifier is negative at level-up**: Health still increases by minimum +1 per level. A character with 6 CON (−2 modifier) gains 1 HP per level, never loses health.
- **If a character's attribute is reduced below 3 by a status effect**: Clamp to 3. The modifier is −4. The effect cannot reduce the attribute below 3 regardless of stacking.
- **If XP from multiple sources would grant 2+ levels at once**: Levels are processed sequentially. The character levels up, gains health/attribute points, THEN the remaining XP is applied against the next level threshold. No XP is lost across the multi-level jump.
- **If the player allocates 0 points to an attribute during creation**: Minimum starting value is 8 (costs 0 points). Cannot go below 8 at character creation.
- **If a party member joins after the player has leveled up**: Party members join at the player's current level with a fixed attribute spread appropriate to their role. They receive all level-up benefits (health, attribute points) as if they had leveled naturally.
- **If a character reaches level 20 (cap)**: XP continues to accumulate but no further level-ups occur. The total XP is preserved for display and for any post-cap content that checks XP thresholds.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | Character Stats depends on Entity Framework | Reads/Writes `Stats` component on entities. Hard — cannot store stats without entity data. |
| Tactical Combat Engine | Combat depends on Character Stats | Reads derived stats (AP, initiative, health, evasion) for combat. Hard — combat needs numbers. |
| Party System | Party depends on Character Stats | Reads stats for party member display; uses CHA for morale calculations. Hard. |
| AI Dialogue Engine | Dialogue depends on Character Stats | Reads INT, WIS, CHA-derived persuasion/insight/comprehension for conversation checks. Soft — dialogue functions without stat bonuses (just harder). |
| Thread Investigation | Investigation depends on Character Stats | Reads INT, WIS-derived comprehension/insight for investigation checks. Soft. |
| Character & Inventory UI | UI depends on Character Stats | Reads all stats for character sheet and level-up screen display. Hard. |
| Save/Load System | Save/Load serializes Character Stats | Stats component included in entity save data via Entity Framework. Hard. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect of Increase | Effect of Decrease |
|-----------|---------|------------|-------------------|-------------------|
| `starting_points` | 30 | 20–40 | More powerful starting characters, less growth tension | Weaker start, more reliance on level-up gains |
| `max_starting_attribute` | 16 | 14–18 | More specialized starting builds possible | More balanced starting builds enforced |
| `xp_per_thread_hop` | 100 | 50–200 | Investigation-heavy leveling, faster progression | Combat-heavy leveling, slower progression |
| `xp_per_enemy_level` | 20 | 10–50 | Combat-heavy leveling, faster progression | Investigation-heavy leveling, slower progression |
| `xp_level_multiplier` | 300 | 200–500 | Slower leveling, more content between levels | Faster leveling, quicker power growth |
| `level_cap` | 20 | 10–30 | More total power growth, longer progression curve | Tighter scope, faster max-level gameplay |
| `attribute_points_per_levels` | 4 | 2–6 | More frequent attribute gains, faster specialization | Slower specialization, attributes feel more fixed |
| `hp_per_con_mod` | 1.0 | 0.5–2.0 | CON more impactful on survivability | CON less dominant for tank builds |

## Visual/Audio Requirements

📌 **Asset Spec** — Visual/Audio requirements are defined below.

| Event | Visual Feedback | Audio Feedback | Priority |
|-------|----------------|---------------|----------|
| Level-up | Golden amber glow envelops character (800ms); floating "+1 [Attribute]" text rises and fades (1.5s); parchment flash on character portrait | Rising chime (3 ascending notes), soft | High |
| Attribute increase (every 4th level) | Larger golden glow (1200ms); attribute value pulses on character sheet; "+1" appears next to chosen attribute | Deeper chime (single resonant note) | High |
| XP gained | Brief "+XP" text near character portrait (800ms, fades) | Subtle paper-rustle sound | Low |
| Health increase on level-up | Health bar briefly flashes full with a pulsing amber fill | Soft heartbeat (single beat) | Medium |
| Character creation | Parchment UI (art bible Section 2.4); attribute sliders with amber highlight; derived stats update in real-time as points are allocated | Quiet ambient (menu music); soft click on point allocation | High |
| Damage taken (health decrease) | Health bar depletes with oxblood red; brief red flash on portrait | Impact sound; intensity scales with damage amount | High |

## UI Requirements

📌 **UX Flag — Character Stats & Progression**: This system has UI requirements.

- **Character creation screen**: Attribute allocation with sliders/buttons, derived stat preview, point-remaining counter, confirm/reset buttons. Full-screen overlay at game start.
- **Character sheet**: Accessible during exploration (default: C key). Shows all six attributes with modifiers, derived stats, level/XP progress bar, and attribute point history.
- **Level-up notification**: Modal overlay when XP threshold is reached. Shows health increase and (every 4 levels) attribute point allocation choice. Cannot be dismissed without allocating attribute point.
- **Party overview**: Compact view of all party members' key stats (health, AP, level). Accessible via Tab or party panel.
- **XP feedback**: Brief "+XP" indicator on screen during gameplay (no modal).

Run `/ux-design` for character-creation, character-sheet, and level-up screens during Pre-Production.

## Acceptance Criteria

- **GIVEN** character creation with 30 points, **WHEN** the player allocates STR 16, DEX 12, CON 14, INT 10, WIS 10, CHA 8, **THEN** the point counter shows 22 spent (8+4+6+2+2+0) and 8 remaining.

- **GIVEN** a character with CON 14 at level 3, **WHEN** the health formula is calculated, **THEN** MaxHealth = 20 + (14×2) + (3×2) = 54 HP.

- **GIVEN** a character with DEX 16 (+3), **WHEN** AP is calculated, **THEN** Action Points = 4 + 3 = 7.

- **GIVEN** a character at level 4 with 1800 total XP, **WHEN** they earn 1200 more XP (total 3000), **THEN** they reach level 5 and gain +CON_mod health and (if level 4 is a multiple of 4) +1 attribute point.

- **GIVEN** a character with CON 6 (−2) leveling from 1 to 2, **WHEN** health is recalculated, **THEN** MaxHealth increases by at least +1 (minimum gain rule), NOT −2.

- **GIVEN** a thread investigation chain of 7 NPC hops resolved, **WHEN** XP is awarded, **THEN** the character receives 100 × 7 = 700 XP.

- **GIVEN** a character at level 20 (cap), **WHEN** they earn additional XP, **THEN** XP accumulates but no further level-up occurs.

- **Performance**: Attribute modifier calculation completes within 0.01ms. XP threshold check completes within 0.01ms. Health recalculation on level-up completes within 0.1ms.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Should temporary stat boosts exist (potions, Ley artifacts, environmental effects)? If so, how do they interact with derived stats? | Game Designer | Before vertical slice | — |
| Should NPCs use the full six-attribute system or simplified stat blocks (only combat-relevant attributes)? | Game Designer | Before NPC Knowledge Graph GDD | — |
| Should the player character have a "background" choice at creation that pre-allocates some attribute points (e.g., "Scholar" → INT 14, WIS 12 preset)? | Narrative Director | Before vertical slice | — |

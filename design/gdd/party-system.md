# Party System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 3 (Tactical Depth — party composition as tactical choice), Pillar 1 (Every Voice — companions carry knowledge fragments)

## Summary

The Party System governs which characters fight alongside the player in The
Marrow of Empire — a pool of recruited companions with a limited number of
active slots (4 at Vertical Slice, 6 at Full Vision). Companions join through
dialogue when trust thresholds and story conditions are met; they carry
knowledge fragments that shape investigation possibilities and react to
revelations as witnesses. The player manages the party at rest points (swap,
dismiss, inspect, formation, scout designation), sets marching order and
formation, and configures companion AI behavior for exploration and combat.
Recruitment, party state, and companion metadata persist across save/load.

> **Quick reference** — Layer: `Feature` · Priority: `VS` · Key deps: `Character Stats & Progression`, `Trust & Relationship`

## Overview

The Party System governs which characters fight alongside the player, how they
are recruited and managed, and how they behave both in and out of combat. It
defines party size limits, recruitment conditions (companions join based on
trust and story events), formation rules for exploration and combat positioning,
and companion AI behavior modes (aggressive, defensive, passive) plus scout
designation. The party is the player's tactical toolkit — each companion brings
different attributes, abilities, and knowledge fragments that shape both combat
encounters and investigation possibilities. Choosing who to bring on an
expedition is a strategic decision with mechanical and narrative weight.

> **Quick reference** — Layer: `Feature` · Priority: `VS` · Key deps: `Character Stats & Progression` (reads attributes for party composition), `Trust & Relationship` (recruitment gated by trust level) · Party size: `up to 4 at VS, 6 at Full Vision`

## Player Fantasy

The Party System delivers the promise at the heart of every great CRPG:
**the people you travel with matter.**

**Choosing companions is the most important decision you make.** Every companion
is a person with their own knowledge fragments, trust requirements, and personal
agenda. You don't recruit stat blocks — you earn the trust of individuals, and
whether they join you depends on what you've done in the world. Some companions
will only follow you if you've proven yourself to their faction. Others will
leave if you betray their trust. The party is not a roster — it's a relationship.

**Your party is your tactical identity.** With a limited number of active slots
and a pool of potential companions, every expedition is a composition decision.
Bring the soldier for frontline durability and faction knowledge. Bring the
scholar for ancient language translation and Ley device operation. Leave the
hunter behind and you won't hear the wildlife warnings. The party you choose
shapes how you fight, what you can discover, and whose voice is present when a
revelation lands.

**Companions are witnesses.** When investigation threads click together, the
companions who are present react — because they carry their own knowledge
fragments. The scholar recognises the Old Imperial inscription. The soldier
knows who gave the order. The hunter reads the tracks. The party isn't just
your combat formation — it's your investigation team, and who you bring
determines what you can learn.

**Reference**: Baldur's Gate 1-2 (companions with distinct personalities who
react to your choices and each other), Dragon Age: Origins (companion approval
system where relationships unlock content), Divinity Original Sin 2 (party
composition as tactical identity — every companion brings unique capabilities).

## Detailed Design

### Core Rules

**CR-1: Party Size and Composition**

The player controls a party of characters. At VS tier: maximum 4 active members
(including the player character). At Full Vision: maximum 6. The player character
is always present and cannot be removed from the party.

Party slots beyond the player character are filled from a pool of recruited
companions. Companions not in the active party wait at the party's current camp,
inn, or most recent rest point. Pool companions relocate to each new rest point
automatically — they are always accessible for party management when the party
rests.

Departed or dismissed companions return to their home region (a per-companion
location defined in the companion's NPC data) and are NOT automatically
accessible at rest points. Re-recruitment requires the player to travel to the
companion's home region or encounter them in the world.

**CR-2: Companion Recruitment**

Companions are NPCs who can join the player's party. Recruitment requires:
- **Trust threshold**: The companion's trust level must meet their recruitment
  requirement (per-companion configurable, stored in the NPC's recruitment data).
  Typical threshold: trust level 1 for early companions, trust level 2 for
  companions with significant knowledge or faction ties. A companion recruited
  at trust level 2 retains trust level 2 after joining.
- **Story condition** (optional): Some companions require a specific world event
  — a thread resolved, a faction introduced, a location discovered. Stored as
  a `discovery_trigger` on the companion entity (same format as NPC Knowledge
  Graph discovery triggers).
- **Party slot available**: The player must have a free active slot. If the
  party is full, the companion is added to the recruitment pool and can be
  swapped in at a rest point.

Recruitment is initiated through dialogue — the AI Dialogue Engine handles the
conversation; the Party System validates conditions and processes the recruitment
event.

**CR-3: Party Management**

At any rest point (camp, inn, safe location — see Living Environment CR-3), the
player can manage their party. Party management is accessed via the wait/rest
dialog (a "Manage Party" option) or a dedicated keybinding at rest points.

- **Swap**: Move a companion from the active party to the pool, and bring another
  from the pool into the active party. Both companions are present at the rest
  point (pool companions automatically relocate to the current rest point per CR-1).
- **Dismiss**: Remove a companion from the recruitment pool. Dismissed companions
  return to their home region and may be re-recruited if trust and story conditions
  are still met. Dismissal does not reset trust. Requires confirmation dialog
  warning that the companion's knowledge domains will no longer be available.
- **Inspect**: View companion stats, equipment, trust level, and known knowledge
  domains.
- **Formation**: Set each companion's combat rank (front or back). Front rank
  deploys ahead of the player character; back rank deploys behind/side. At VS,
  default auto-assignment places melee-oriented companions in front and
  ranged/caster companions in back. Free-form grid editor deferred to Full Vision.
- **Scout designation**: Toggle scout mode on/off for one companion. Only one
  companion may be scout at a time. Default auto-assignment selects the companion
  with the highest WIS (ties broken by highest DEX). The player may override.

Party management is not available during combat, conversation, or region
transitions.

**CR-4: Party Formation**

During exploration, the party moves in formation:
- **Default formation**: The player character leads; companions follow in a loose
  diamond pattern at `formation_spacing` units behind the leader. For party sizes
  other than 4, companions fan out in a trailing arc (2 companions: line; 3:
  triangle; 5: pentagon arc; 6: double-row arc).
- **Marching order**: Front rank (melee companions) and back rank (ranged/caster
  companions). Configurable via the party management screen at rest points
  (front/back rank toggle per companion). Free-form formation editor deferred
  to Full Vision.
- **Scout mode**: One companion (auto-designated: highest WIS, tiebreak highest
  DEX; player may override at rest points) moves `scout_distance` units ahead of
  the party leader, detecting threats and environmental clues.

During combat (see Tactical Combat Engine GDD), formation determines initial
positioning. Companions deploy to their assigned formation positions relative to
the player character.

**Scout detection**: The scout detects hidden threats within `scout_detection_range`
units (see Formulas). Detection is an opposed check: scout WIS vs. threat stealth
or ambush DC. Visual obstructions (foliage, ruins, elevation) and lighting
conditions (darkness, weather) apply disadvantage (roll twice, take lower).
Detection is a forward 180° cone relative to the scout's facing — threats behind
the scout are not detected. Detection is staged: at detection range, the scout
enters "alerted" state (stops, hand-raised animation, "Something ahead" voice
line). The player may then choose: approach carefully, recall scout, or push
forward. If the player does nothing, the scout holds position.

Non-scout party members detect threats at `base_detection_range` (see Formulas)
with the same opposed-check rules, but without the scout's range bonus.

**CR-5: Companion AI Behavior**

Each companion has a behavior mode, configurable by the player. Modes are set
during exploration (not mid-combat) and persist across save/load. A newly
recruited companion defaults to **Defensive** mode.

| Mode | Exploration Behavior |
|------|---------------------|
| **Aggressive** | Moves ahead of party formation; attacks hostile entities within `aggressive_detection_range` units (15u default) on sight. Respects player stealth state — if the player is sneaking, the companion does not initiate combat. |
| **Defensive** | Stays within 5 units of the player character; does not initiate combat. Attacks only enemies that have already engaged the party (dealt damage or entered melee range of any party member). |
| **Passive** | Stays in formation; does not attack unless an enemy declares an attack directly against this companion (visible target indicator). Uses support abilities only (abilities tagged `support` in the ability data). Flees from threats: moves toward the nearest party member at 1.5× exploration speed until ≥15 units from the threat. A companion with zero support abilities remains at the player's position and does nothing — this is intentional; switch mode if combat is expected. |

**Combat behavior (turn-based)**:
- **Aggressive**: The AI targets the nearest enemy (nearest to the companion by
  path distance, ties broken by lowest current HP). Uses abilities on cooldown
  unless the ability consumes a limited resource (spell slots, per-rest charges)
  and remaining reserves are below 25% — above that threshold, abilities are used
  freely. Re-evaluates target each turn.
- **Defensive**: The AI targets enemies within 5 units of the player character
  or enemies that dealt damage to any party member in the previous round. Does
  not pursue enemies that retreat beyond 10 units from the player.
- **Passive**: Does not select attack targets. Uses support abilities on the
  lowest-HP party member within range. If directly targeted (enemy declares
  attack against this companion), may attack that enemy in self-defense.

The AI decision-making is owned by the Tactical Combat Engine — the Party System
provides the mode configuration; TCE translates it into target selection and
ability-use rules during `AllyTurn`. This contract must be reflected in both GDDs.

**Player override**: The player can override any companion action directly
during combat. An override persists until the ordered target dies or the player
issues a new explicit command to that companion, at which point the AI mode
resumes as the default.

**Mode and scout interaction**: Scout mode is independent of behavior mode.
When a companion is both scout and Aggressive, scout behavior takes precedence:
the scout stops and alerts on detection rather than attacking. The player then
decides whether to engage. An aggressive non-scout companion attacks detected
threats per the Aggressive exploration rule above.

**CR-6: Companion Knowledge Contribution**

Companions are investigation participants. Each companion has a `Knowledge`
component (EF component #5) with their own domain expertise. When a companion
is in the active party:
- Their knowledge domains are available for question matching (Question System).
- During conversation, companions interject if the dialogue topic matches their
  domain — see Interjection Rules below.
- When an investigation thread resolves, present companions with relevant
  knowledge contribute additional context (Thread Investigation).

**Interjection Rules:**
- **Trigger**: The AI Dialogue Engine evaluates each active companion's knowledge
  domains against the current dialogue topic. If domain match confidence exceeds
  `interjection_confidence_threshold` (default 0.7), the companion may interject.
- **Frequency cap**: Maximum `interjections_per_conversation_cap` (default 1)
  interjection per companion per conversation. Maximum `total_interjections_cap`
  (default 3) interjections total per conversation across all companions.
- **Priority**: When multiple companions qualify, the companion with the highest
  domain match confidence speaks. Ties broken by highest trust level, then by
  highest WIS.
- **Content source**: Interjection lines are sourced from the companion's
  `Dialogue` component (authored interjection lines keyed by domain). If no
  authored line exists for the matched topic, the AI Dialogue Engine generates
  one via LLM — this is a non-blocking async generation; the companion's portrait
  appears alongside the NPC portrait with a brief pause indicator.
- **Dialogue state**: Interjections occur at response-choice points in the
  conversation flow, never mid-sentence. The companion speaks after the NPC
  finishes; the player may then respond to the interjection or continue the
  main conversation.
- **Thread resolution**: When an investigation thread resolves, each present
  companion whose knowledge domains overlap with the thread's topic contributes
  one guaranteed reaction (dialogue line + optional journal entry). This is NOT
  subject to the frequency cap — thread resolutions are guaranteed witness
  moments.

**Knowledge exclusivity**: Some knowledge fragments are held by exactly one
companion (personal testimony, unique expertise, faction secrets). These
exclusive fragments are tagged `exclusive: true` in the companion's Knowledge
component. If that companion leaves the party, the exclusive fragment becomes
inaccessible until the companion is re-recruited. Non-exclusive domain knowledge
may be covered by other companions or NPCs per standard Question System matching.

> **Cross-document contract**: The AI Dialogue Engine GDD must define the
> interjection interface — a `companion_interject` trigger with domain match
> scoring, the frequency cap budget, and the async LLM fallback path. The
> Question System must gate domain access on companion presence for
> `exclusive: true` fragments. This dependency is **Hard** — the witness
> mechanic is Pillar 1's primary mechanical carrier and cannot function
> without coordinated implementation.

**CR-7: Companion Departure**

Companions may leave the party:
- **Trust betrayal**: If a companion's trust level drops to or below their
  departure threshold (typically trust level 0 — i.e., `trust_level <=
  departure_threshold`), they leave. The departure threshold may differ from
  the recruitment threshold. The Trust & Relationship system clamps trust at
  0 — reaching the clamp IS the departure event for threshold-0 companions.
- **Story departure**: Scripted departures tied to story events (companion's
  personal quest resolves, faction conflict forces a choice). Triggers a
  departure scene: final dialogue, reactions from other active companions,
  and a journal entry.
- **Death in combat**: At VS, companions at 0 HP are knocked out (unconscious,
  revive after combat with 1 HP). At Full Vision, `companion_death_enabled`
  becomes a configurable option — when enabled, companions follow TCE death
  rules (dying state → permanent death after 3 rounds without stabilization).
  Death is **permanent** — dead companions cannot be re-recruited. The
  `Departed` state with `cause: death` blocks the re-recruitment path.

A companion who departs via trust betrayal or story returns to their home region.

**Departure presentation (all departure types):**
- Goodbye dialogue (authored, context-specific — betrayed companions speak
  differently than those whose personal quest concluded)
- Reaction lines from other active companions (at least one reaction per
  companion with trust level ≥ 1 toward the departing companion)
- Journal entry recording the departure event
- Post-combat deferred departure: if departure triggers during combat, the
  companion fights normally for the remainder of the encounter. At combat
  resolution, a notification banner appears ("[Name] is leaving"), the
  departure scene plays, and the companion exits the formation.

**Re-recruitment:**
- After trust-betrayal departure: trust must be restored to `recruitment_threshold
  + departure_repentance_penalty`, clamped at `max_trust_level` (3). The Trust
  & Relationship trust freeze (72 game hours) applies — rebuilding trust after
  betrayal is a multi-day investment.
- After story departure: re-recruitment conditions are story-specific and
  defined in the companion's narrative data.
- After voluntary dismissal: re-recruitment at the original recruitment
  threshold with no additional penalty.
- After death: re-recruitment is **not possible**.

### States and Transitions

A companion's party state:

| State | Meaning |
|-------|---------|
| `Unrecruited` | NPC exists in the world but has not joined the party |
| `InParty` | Active party member — present in formation, participates in combat and conversation |
| `InPool` | Recruited but not in active party — waits at the party's current rest point |
| `Departed` | Left the party — returned to home region. `cause` field persisted: `trust_betrayal`, `story`, `death`, or `dismissed` |

**Transitions:**
- `Unrecruited → InParty`: Recruitment conditions met, party slot available.
- `Unrecruited → InPool`: Recruitment conditions met, party full → added to pool.
- `InPool → InParty`: Player swaps companion in at rest point.
- `InParty → InPool`: Player swaps companion out at rest point.
- `InParty → Departed`: Trust ≤ departure threshold, story departure, or death.
  `cause` field set to distinguish the departure type.
- `InPool → Departed`: Player dismisses (`cause: dismissed`), trust departure
  triggers (`cause: trust_betrayal`), or story departure triggers (`cause: story`).
- `Departed → Unrecruited`: Companion returns home; re-recruitment possible
  (unless `cause: death` — death blocks this transition permanently).
- `Departed → InParty`: Re-recruitment conditions met (trust restored above
  threshold + penalty for betrayal, or story conditions met for story departure),
  party slot available. Blocked for `cause: death`.
- `Departed → InPool`: Same as above, but party is full.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Reads/writes companion entities | Companions are NPC entities with `Stats`, `Knowledge`, `Faction`, `Combatant`, `Inventory` components. Active party members tagged `pc_controlled`. |
| Character Stats & Progression | Reads companion stats | Companions use the same stat system as the player. Party System reads attributes, level, and derived values for formation, AI behavior, and UI. |
| Trust & Relationship | Reads trust for recruitment/departure | Recruitment requires `trust_level >= recruitment_threshold`. Departure when `trust_level <= departure_threshold`. Trust events may fire on companion actions. T&R documents that party members have a departure threshold contract. |
| Tactical Combat Engine | Provides party composition | TCE reads active party member list for turn order, initiative, and companion AI (mode → target selection contract). Party System sets initial positioning from formation. Companion knockout state (VS) defined in both GDDs. |
| AI Dialogue Engine | Companions contribute knowledge | Active companions' `Knowledge` and `Dialogue` components drive interjections, domain-specific responses, and thread-resolution witness reactions. Hard dependency — ADE must implement the interjection interface defined in CR-6. |
| Living Environment | Reads rest points | Party management only available at `rest_point` entities (camp, inn, safe location). Wait/rest dialog includes "Manage Party" option. |
| Character & Inventory UI | UI reads party data | Party management screen, companion inspect, equipment management. |
| Save/Load System | Persists party state | Recruitment, party state, companion metadata (including `cause` field on Departed state), formation config, behavior modes, and scout designation persist across save/load. |

## Formulas

### Scout Detection

```
scout_detection_range = scout_detection_base + WIS_mod × scout_wis_scale

Where:
  scout_detection_base = 10 units
  scout_wis_scale     = 2 units per WIS modifier point
  WIS_mod             = (WIS - 10) / 2, rounded down (standard attribute modifier)

Example: WIS 16 → mod +3 → range = 10 + 3 × 2 = 16 units
Example: WIS 10 → mod  0 → range = 10 + 0 × 2 = 10 units
Example: WIS  6 → mod −2 → range = 10 − 2 × 2 =  6 units

Detection is an opposed roll:
  scout_perception = d20 + WIS_mod + scout_level_bonus
  threat_stealth   = d20 + threat_stealth_mod + terrain_cover_bonus
  Detection succeeds if scout_perception > threat_stealth

Disadvantage (roll twice, take lower) applies in: darkness, fog, heavy foliage,
or when the threat has full cover. The detection arc is a forward 180° cone.
```

### Base Detection (Non-Scout)

```
base_detection_range = 8 units (all party members)
Same opposed-roll mechanics as scout detection.
```

### Aggressive Mode Detection

```
aggressive_detection_range = 15 units
Triggers when a hostile entity enters this radius of an Aggressive-mode companion.
Subject to the same opposed-roll + LOS + lighting rules as scout detection.
Respects player stealth state.
```

### Re-Recruitment Requirement

```
re_recruitment_threshold = min(recruitment_threshold + departure_repentance_penalty, max_trust_level)

Where:
  max_trust_level = 3 (Trust & Relationship CR-1)

Example: threshold 1 + penalty 1 = 2 ✓
Example: threshold 2 + penalty 1 = 3 ✓ (at cap)
Example: threshold 3 + penalty 1 = 3 (clamped from 4)
Example: threshold 3 + penalty 2 = 3 (clamped from 5)
```

### Structural Constants

| Constant | Value | Source |
|----------|-------|--------|
| VS party size | 4 (including player) | CR-1 |
| Full Vision party size | 6 | CR-1 |
| Default formation spacing | 3 units | CR-4 |
| Scout advance distance | 10 units | CR-4 |
| Scout detection base | 10 units | CR-4 |
| Scout WIS scale | 2 units per mod | CR-4 |
| Base detection range (non-scout) | 8 units | CR-4 |
| Aggressive detection range | 15 units | CR-5 |
| Aggressive ability reserve threshold | 25% | CR-5 |
| Defensive proximity radius | 5 units | CR-5 |
| Interjection confidence threshold | 0.7 | CR-6 |
| Interjections per companion per conversation | 1 | CR-6 |
| Total interjections per conversation | 3 | CR-6 |
| Departure re-recruitment penalty | +1 trust level (clamped at 3) | CR-7 |
| Flee distance | 15 units from threat | CR-5 |

## Edge Cases

- **If the player's trust with a companion drops to or below the departure
  threshold during combat**: Departure is deferred until combat ends. The
  companion fights normally for the remainder of the encounter. At combat
  resolution, a notification banner appears and the departure scene plays.

- **If the player's trust with a companion drops to the departure threshold
  during conversation**: Departure is deferred until the conversation ends.
  The companion does not interject for the remainder of the conversation.
  After the conversation closes, the departure scene plays.

- **If the only active party member other than the player character departs
  mid-expedition**: The player continues solo. The party size drops to 1.
  The player can return to camp to bring companions from the pool.

- **If a companion in the recruitment pool is killed while waiting at camp**
  (world event, faction conflict): The companion transitions to `Departed`
  with cause `death`. The player is notified on next rest. Death is permanent
  — this companion cannot be re-recruited.

- **If the player attempts to recruit a companion whose trust is sufficient
  but a required story condition is unmet**: Recruitment is deferred — the
  companion indicates what needs to happen first (via dialogue). The condition
  is tracked in the companion's recruitment data.

- **If all companions in the pool depart or die**: The party consists of the
  player character only. New companions must be recruited from the world.

- **If a companion is in scout mode and triggers combat**: The companion is
  placed at their scout position. The rest of the party deploys at their
  formation positions relative to the player character. The scout may be
  isolated — this is the risk of scouting. Combat initiative proceeds normally.

- **If a companion with unique knowledge leaves the party**: Their knowledge
  domains are no longer available for question matching. Already-discovered
  facts from that companion remain in `PlayerKnowledge`. Exclusive fragments
  (`exclusive: true`) become inaccessible until re-recruitment. Non-exclusive
  domain knowledge may still be covered by other companions or NPCs.

- **If a companion reaches 0 HP at VS** (knockout): The companion is unconscious
  for the remainder of combat. After combat, they revive with 1 HP. No death
  countdown applies at VS. At Full Vision with `companion_death_enabled = true`,
  TCE death rules apply (dying → stabilize within 3 rounds or permanent death).

- **If combat ends with a companion at 0 HP** (Full Vision, permadeath enabled):
  The TCE dying countdown pauses at combat end. The player may attempt
  stabilization as a post-combat action (no AP cost, requires adjacency).
  If the companion was already at 0 rounds remaining, they die immediately
  at combat resolution.

- **If a companion dies and trust also drops below departure threshold in the
  same combat**: Death takes precedence. The companion is dead (permanent
  removal). The trust-betrayal departure path is not triggered.

- **If the player character is the only party member and enters scout mode**:
  Scout mode is disabled for solo parties. The player character cannot be
  designated as scout — scouting requires a companion.

- **If scout designation is toggled mid-expedition** (away from rest point):
  Scout designation can only be changed at rest points per CR-3. The current
  scout continues until the next rest.

- **If two companions have equal WIS and DEX for auto-scout selection**:
  The companion with the higher trust level is selected. If still tied, the
  companion who has been in the party longer is selected.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | Party System depends on EF | Reads/writes companion entities. Tags active members `pc_controlled`. **Hard.** |
| Character Stats & Progression | Party System depends on Char Stats | Reads companion attributes, level, and derived values. **Hard.** |
| Trust & Relationship | Party System depends on T&R | Reads trust levels for recruitment and departure thresholds. T&R documents party departure contract (clamp-at-0 = departure event). **Hard.** |
| Tactical Combat Engine | TCE depends on Party System | TCE reads active party list for combat setup and companion AI (mode → target selection). Party System provides initial formation positions. **Hard downstream.** |
| AI Dialogue Engine | AI Dialogue depends on Party System | Active companions' knowledge and dialogue components drive interjections, domain responses, and witness reactions. ADE must implement interjection interface per CR-6. **Hard.** |
| Living Environment | Coordinates | Party management gated behind `rest_point` entities. Wait/rest dialog includes party management entry point. Soft. |
| Character & Inventory UI | UI depends on Party System | Party management screen, companion inspect, equipment management. **Hard downstream.** |
| Save/Load System | Party System depends on Save/Load | Party state, companion metadata (including `cause` on Departed), formation config, behavior modes, scout designation persist. **Hard.** |

## Tuning Knobs

| Parameter | Default | Safe Range | Notes |
|-----------|---------|------------|-------|
| `vs_party_size` | 4 | 2–4 | Including player character. VS tier. |
| `full_vision_party_size` | 6 | 4–6 | Full Vision tier. |
| `formation_spacing` | 3u | 1–5 | Distance between companions in formation |
| `scout_distance` | 10u | 5–20 | How far ahead scout moves |
| `scout_detection_base` | 10u | 5–15 | Base detection range before WIS modifier |
| `scout_wis_scale` | 2 | 1–3 | Units per WIS modifier point |
| `base_detection_range` | 8u | 5–12 | Detection range for non-scout party members |
| `aggressive_detection_range` | 15u | 10–25 | Aggressive mode threat detection radius |
| `aggressive_ability_reserve` | 0.25 | 0.0–0.5 | Fraction of limited resources reserved (don't spend below) |
| `defensive_proximity_radius` | 5u | 3–8 | "Near player character" radius for Defensive mode |
| `flee_distance` | 15u | 10–30 | Distance Passive companions flee from threats |
| `interjection_confidence_threshold` | 0.7 | 0.5–0.95 | Domain match score required for interjection |
| `interjections_per_companion_cap` | 1 | 0–3 | Max interjections per companion per conversation |
| `total_interjections_cap` | 3 | 1–6 | Max total interjections per conversation |
| `departure_repentance_penalty` | 1 | 0–2 | Extra trust levels needed for re-recruitment after betrayal. Clamped at `max_trust_level` (3). |
| `companion_death_enabled` | false | true/false | VS default: false (knockout). True = permadeath per TCE rules (Full Vision). |
| `scout_detection_arc` | 180° | 90–360° | Forward cone for scout detection |

## Visual/Audio Requirements

📌 **Asset Spec** — Companion visual/audio required.

| Event | Visual | Audio |
|-------|--------|-------|
| Companion recruited | Brief portrait animation; companion joins formation | Affirmative voice line; party-join sting |
| Companion departs (trust) | Companion walks away from formation; fades at edge of viewport; notification banner | Disappointed/sad voice line; other companions may react |
| Companion departs (story) | Departure scene: companion + reacting companions framed; fades at edge of viewport | Departure dialogue; companion reactions |
| Companion departs (death) | Death animation; body remains; notification banner | Death sound; other companions may react |
| Party management at rest | Camp/inn scene with companions visible; swap animation | Ambient camp sounds |
| Scout detects threat | Scout stops; alert animation (hand raised) | "Something ahead" voice line; tension sting |
| Companion interjects in dialogue | Companion portrait appears alongside NPC portrait; brief pause indicator if LLM-generated | Companion voice line |
| Thread resolution witness moment | Companion portrait + NPC portrait framed together; journal entry animation | Guaranteed companion voice line |

## UI Requirements

📌 **UX Flag — Party System**: This system has UI requirements.

- **Party management screen**: Accessible at rest points via the wait/rest dialog
  ("Manage Party" option) or dedicated keybinding. Shows active party (4 slots,
  filled or empty), recruitment pool (scrollable list sorted by trust level),
  companion stats on hover/select (name, role, HP, trust level, knowledge domains).
  Click-to-swap between active and pool (keyboard: Tab-focus + Enter; gamepad:
  d-pad + A). Empty slots show a subdued "recruit to fill" affordance.
  Front/back rank toggle per active companion. Scout designation toggle.
  Confirm dialog for dismiss. Pause live world while open.
- **Formation editor** (deferred to Full Vision): Visual drag-and-drop grid
  for setting marching order and combat positioning. At VS: front/back rank
  toggle on the party management screen, with role-based auto-assignment
  (melee → front, ranged/caster → back).
- **Companion inspect**: Full character sheet — stats, equipment, knowledge
  domains, trust level (numeric + visual indicator, not color-only),
  recruitment story. Accessible from party management and from the Character
  & Inventory UI (shared component). C&I UI owns the sheet layout; Party System
  provides the trust/knowledge/recruitment sections.
- **Companion behavior mode**: Dropdown or cycle button per companion
  (Aggressive / Defensive / Passive), visible on the HUD during exploration.
  Mode glyph persists on the companion portrait for at-a-glance state legibility.
  Keyboard: d-pad through portraits + A to cycle. Behavior-mode HUD spec must
  be defined in a shared exploration HUD document (`design/ux/hud.md`).
- **Departure notification**: Post-combat/post-conversation banner: "[Name] has
  left the party." Triggers departure scene. Journal entry auto-created.
- **Reduce-motion**: All party animations (swap, departure fades, alert animations)
  respect a `reduce_motion` accessibility setting.

## Acceptance Criteria

- **AC-PS-01**: GIVEN a companion NPC with trust level 2 and `recruitment_threshold = 2`,
  WHEN the player asks them to join and a party slot is available, THEN the
  companion appears in the party screen's active slots, follows in formation
  during exploration, and is present in the next combat encounter's roster.
  The companion is tagged `pc_controlled` (verifiable via debug overlay).

- **AC-PS-02**: GIVEN a full party (4 members at VS), WHEN a companion with
  sufficient trust agrees to join, THEN the companion appears in the
  recruitment pool (party management screen) and is NOT added to the active
  formation. The companion is visible at the next rest point.

- **AC-PS-03a (swap-out)**: GIVEN a companion in `InParty` at a rest point,
  WHEN the player swaps them to the pool, THEN the companion is removed from
  the active formation, appears in the pool list on the party management
  screen, and is absent from the next combat encounter's roster.

- **AC-PS-03b (swap-in)**: GIVEN a companion in `InPool` at a rest point,
  WHEN the player swaps them into the active party, THEN the companion
  appears in the active formation, follows in exploration, and is present
  in the next combat encounter's roster.

- **AC-PS-04a (trust departure)**: GIVEN a companion with `departure_threshold = 1`
  at trust level 2, WHEN a trust-loss event reduces trust to 1 while outside
  combat and conversation, THEN the departure scene plays (goodbye dialogue +
  companion reactions), a journal entry is created, and the companion is
  removed from the active formation.

- **AC-PS-04b (departure deferred — combat)**: GIVEN the same trigger during
  an active combat encounter, WHEN the encounter ends, THEN a notification
  banner appears ("[Name] is leaving"), the departure scene plays, and the
  companion is removed from the party.

- **AC-PS-04c (departure deferred — conversation)**: GIVEN the same trigger
  during an active conversation, WHEN the conversation ends, THEN the
  departure scene plays and the companion is removed.

- **AC-PS-05a (knowledge available)**: GIVEN an active companion with knowledge
  domain `old_empire_lore` and an NPC with a question gated behind that domain,
  WHEN the companion is in the active party and the player opens dialogue,
  THEN the question option is present.

- **AC-PS-05b (knowledge removed)**: GIVEN the same companion NOT in the active
  party, WHEN the player opens dialogue with the same NPC, THEN the
  domain-gated question option is absent (unless another present companion
  or the NPC themselves covers the domain).

- **AC-PS-06 (formation deployment)**: GIVEN a party of 4 with front rank
  (soldier, scholar) and back rank (hunter, player) set via the party
  management screen, WHEN combat starts, THEN by the start of round 1 each
  companion is within `formation_spacing` (±1 unit) of their assigned position
  relative to the player character. Front-rank companions deploy ahead; back-rank
  behind or to the side. Verifiable via debug position readout.

- **AC-PS-07 (aggressive mode)**: GIVEN a companion set to Aggressive mode
  and the player NOT in stealth, WHEN hostile wildlife is detected within
  `aggressive_detection_range` (15 units) during exploration, THEN the
  companion begins moving toward the threat within 2 seconds with no player
  input, and continues until contact or combat begins.

- **AC-PS-07b (aggressive respects stealth)**: GIVEN a companion set to
  Aggressive mode and the player IS in stealth, WHEN hostile wildlife enters
  detection range, THEN the companion does NOT initiate combat.

- **AC-PS-08a (passive mode)**: GIVEN a companion set to Passive mode,
  WHEN the party is attacked and no enemy targets the companion for the
  first 3 rounds, THEN the companion performs no attacks. Support abilities
  on the lowest-HP party member are permitted.

- **AC-PS-08b (defensive mode)**: GIVEN a companion set to Defensive mode,
  WHEN the party is attacked but no enemy engages the companion (deals damage
  or enters melee range), THEN the companion stays within 5 units of the
  player character and does not initiate attacks.

- **AC-PS-09a (scout detection — WIS 16)**: GIVEN a test map with a hidden
  threat (stealth 10) at a known position and a scout with WIS 16 (+3),
  WHEN the party approaches, THEN the scout detects the threat at
  `scout_detection_base + WIS_mod × scout_wis_scale` (16) units ±1,
  enters alert state (stops, alert animation), and the player receives
  a prompt (approach / recall / push forward). Verifiable via debug readout.

- **AC-PS-09b (scout detection — WIS 10)**: GIVEN the same setup with
  WIS 10 (+0), WHEN the party approaches, THEN detection occurs at 10 units ±1.

- **AC-PS-10a (re-recruitment after betrayal)**: GIVEN a companion departed
  via trust betrayal (recruitment threshold 2, penalty 1), WHEN trust is
  restored to 3 and a party slot is free, THEN the companion's dialogue
  offers the join option. Accepting transitions them to `InParty` (or
  `InPool` if full).

- **AC-PS-10b (re-recruitment after dismissal)**: GIVEN a companion dismissed
  voluntarily (recruitment threshold 2), WHEN trust is at or above 2 and a
  slot is free, THEN the companion can be re-recruited at the original
  threshold with no penalty.

- **AC-PS-10c (death blocks re-recruitment)**: GIVEN a companion who died
  in combat (`cause: death`), WHEN the player travels to the companion's
  home region, THEN the companion is not present and cannot be re-recruited.

- **AC-PS-11 (dismissal)**: GIVEN a companion in `InPool`, WHEN the player
  dismisses them at a rest point and confirms the dismissal dialog, THEN
  the companion transitions to `Departed` with `cause: dismissed`, is
  removed from the pool list, and returns to their home region.

- **AC-PS-12 (party management gating)**: GIVEN the player is in combat,
  conversation, or a region transition, WHEN the player attempts to open
  party management, THEN the option is unavailable (greyed out or hidden).

- **AC-PS-13 (PC immovable)**: GIVEN any party state, WHEN the player
  attempts to remove or swap out the player character, THEN the action
  is rejected. The PC slot is always locked.

- **AC-PS-14 (scout isolation on combat)**: GIVEN a scout companion 10
  units ahead and hidden enemies between the scout and the party, WHEN
  combat triggers, THEN the scout deploys at their scout position
  (potentially isolated), the rest of the party deploys at formation
  positions, and initiative proceeds normally.

- **AC-PS-15 (story departure)**: GIVEN a companion with a story departure
  trigger satisfied, WHEN the trigger fires, THEN a departure scene plays
  (final dialogue + companion reactions), a journal entry is created, the
  companion transitions to `Departed` with `cause: story`, and the
  companion returns to their home region.

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should companions gain XP when not in the active party, or only when present? | Game Designer | Before VS implementation |
| Should companions have personal quest lines (like BG2 stronghold quests) or more reactive story integration? | Narrative Director | Before VS implementation |
| Should formation be a per-save setting or stored per region/encounter type? | UX Designer | Before VS implementation |

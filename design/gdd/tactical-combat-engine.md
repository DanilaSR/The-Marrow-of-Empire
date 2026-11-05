# Tactical Combat Engine

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 3 (Tactical Depth, Not Tactical Bloat)

## Summary

Turn-based tactical combat on an isometric grid. Characters spend Action Points
(AP) to move, attack, and use abilities. Turn order is determined by Initiative
(DEX + WIS modifiers). Every combat encounter has narrative weight — no filler
fights. Combat pauses the game clock and restricts saving to manual only.
Positioning, synergies, and environment interaction reward tactical mastery.

> **Quick reference** — Layer: `Feature` · Priority: `MVP` · Key deps: `Character Stats`, `Entity Framework`

## Overview

The Tactical Combat Engine is the combat half of The Marrow of Empire's core
loop. When words fail — when an investigation leads to violence, when the
ruins are guarded, when a confrontation turns deadly — combat resolves the
conflict through turn-based tactical play on the isometric battlefield.

The engine provides:

- **Turn-based AP system**: Each character has Action Points (derived from DEX:
  `4 + DEX_mod`, minimum 2). Actions cost AP: Move 1 AP per tile, standard
  attack 2 AP, ability 1–3 AP, interact 1 AP. Unused AP does not carry over.
- **Initiative-based turn order**: Turn order is determined by Initiative
  (`DEX_mod + WIS_mod`). Ties broken by DEX, then entity ID. Turn order is
  displayed in the Combat UI.
- **Grid-based positioning**: The isometric battlefield uses a grid (tile or
  hex — determined by ADR). Positioning matters: flanking, cover, elevation,
  and range are all tactical factors.
- **Attack resolution**: Attacks use accuracy vs. evasion. Damage is weapon
  base + STR_mod (melee) or DEX_mod (ranged) − target's armor. Critical
  hits on natural high rolls.
- **Environmental interaction**: The battlefield is interactive. Explosive
  barrels, collapsing structures, terrain hazards. Combat is not fought in
  an empty arena — it's fought in the same hand-crafted environments as
  exploration.
- **Death and defeat**: Characters reduced to 0 HP are dying — they can be
  stabilized by allies within 3 rounds or they die permanently. Enemies at
  0 HP are dead. Player character death is permanent (permadeath enabled
  by default; optional save-reload).

Without the Tactical Combat Engine: the game has no combat. The "talk →
explore → fight → talk" loop is broken. Tactical Depth has no delivery
mechanism.

## Player Fantasy

**Every fight matters.** Pillar 3 demands "Tactical Depth, Not Tactical
Bloat." There are no trash mobs, no random encounters for XP farming, no
filler fights to pad playtime. Every combat encounter exists because of a
narrative reason — guards protecting a secret, creatures defending their
territory, a confrontation turning violent. When the player draws their
weapon, something is at stake.

**Mastery is rewarded.** The player who understands positioning (flanking
bonus: +2 to hit from behind), action economy (save AP for a finishing blow
vs. spend it all on movement), and synergy (the fighter pins the enemy while
the rogue flanks) will win fights that a button-masher loses. The system
rewards thinking, not grinding.

**The battlefield tells the story.** Combat happens in the same hand-crafted
environments as exploration — the ruined machine-hall, the forest clearing,
the border-town street. Cover is a collapsed pillar. High ground is a fallen
statue. The environment is part of the tactical puzzle, and it tells the
story of where you're fighting.

## Detailed Design

### Core Rules

**CR-1: Combat Initiation**

Combat begins when:
- The player attacks a hostile entity
- A hostile entity detects the player and initiates combat
- A conversation reaches a violent conclusion

On combat start:
1. Game clock pauses (Living Environment CR-1)
2. Camera transitions to Combat state (Camera CR-3)
3. Initiative is rolled for all combatants
4. Turn order is displayed
5. First combatant in turn order begins their turn

**CR-2: Action Points**

Each combatant has a pool of AP per turn:

| Stat | Formula |
|------|---------|
| Base AP | 4 |
| AP Total | `4 + DEX_modifier` (minimum 2, maximum 8 at DEX 18) |

| Action | AP Cost |
|--------|---------|
| Move 1 tile | 1 |
| Standard attack (melee/ranged) | 2 |
| Ability (varies by ability) | 1–3 |
| Interact (open door, pull lever, use item) | 1 |
| Defend (end turn, +2 evasion until next turn) | Remaining AP (minimum 1) |
| Wait (delay turn — move to end of initiative order) | Remaining AP (minimum 1) |

Unused AP is lost at end of turn. There is no AP carryover.

**CR-3: Movement and Positioning**

Combat uses a grid-based movement system on the isometric battlefield:

- Each tile represents ~1 meter in world space
- Movement cost: 1 AP per tile. Diagonal movement allowed at 1 AP.
- Flanking: Attacking from the opposite side of an ally grants +2 accuracy.
  Flanking requires the ally to be adjacent to the target.
- Cover: Half cover (behind low wall, pillar) grants +2 evasion. Full cover
  (behind solid structure) blocks line of sight entirely.
- Elevation: Higher ground grants +1 accuracy and +1 range. Moving up 1
  elevation level costs 2 AP.
- Opportunity: Moving out of an enemy's melee range triggers a free attack
  from that enemy (once per turn per enemy).

**CR-4: Attack Resolution**

Attacks are resolved against the target's Evasion:

| Step | Calculation |
|------|-------------|
| Accuracy | Base weapon accuracy + attacker modifiers (flanking, elevation, ability bonuses) |
| Evasion | `10 + DEX_mod + cover_bonus + ability_bonuses` |
| Hit roll | `d20 + Accuracy` vs. target Evasion. Equal or exceed = hit. |
| Critical | Natural 20 on the d20 = critical hit (double damage). Natural 1 = automatic miss. |

Damage calculation (Melee):
`damage = weapon_base_damage + STR_modifier + ability_bonus − target_armor`

Damage calculation (Ranged):
`damage = weapon_base_damage + DEX_modifier + ability_bonus − target_armor`

Minimum damage: 1 (a hit always deals at least 1 damage).

**CR-5: Death and Dying**

| State | Condition | Effect |
|-------|-----------|--------|
| Alive | HP > 0 | Normal combat participation |
| Knocked Out (party, VS only) | HP = 0, `companion_death_enabled = false` | Unconscious for remainder of combat. Cannot act. Revives with 1 HP when combat ends. No stabilization required. |
| Dying (player/party, Full Vision) | HP = 0, `companion_death_enabled = true` | Cannot act. Must be stabilized by ally within `dying_rounds` (3) rounds or dies permanently. Stabilize costs 2 AP, requires adjacency. |
| Dead | HP = 0, `dying_rounds` elapsed without stabilization (FV) | Permanently removed from party. Save/Load is the only recovery mechanism. |
| Defeated (enemy) | HP = 0 | Enemy is dead. Drops loot if applicable. |

`companion_death_enabled` is a Party System configuration knob: `false` at VS (Knocked Out), configurable at Full Vision (Dying/Dead). The TCE reads this flag and routes 0-HP companions to the appropriate state.

**CR-6: Combat Resolution**

Combat ends when:
- All enemies are dead, fled, or surrendered
- All player characters are dead or dying (game over — reload)
- A scripted event ends combat (reinforcements arrive, environmental collapse, etc.)

On combat end:
1. Game clock resumes
2. Camera returns to Exploration state
3. XP is awarded (Character Stats CR-5)
4. Combat end events fire (trust gains for saving NPCs, thread updates, etc.)
5. Knocked Out companions (VS) revive with 1 HP
6. Dying companions (FV, `companion_death_enabled = true`): the round countdown pauses at combat end. The player may attempt stabilization as a post-combat action (no AP cost, requires adjacency). If the companion was at 0 rounds remaining, they die immediately at combat resolution. If stabilized, they revive with 1 HP.
7. Loot drops are requested from Loot & Economy: `request_loot_drop(enemy_type, region_tier)` — L&E returns items + gold. Items appear on enemy body for post-combat looting.

**CR-7: Enemy Stat Blocks**

Enemies use simplified stat blocks derived from region tier and archetype.
Unlike player characters, enemies do not use the full six-attribute system —
they have only the values needed for combat resolution.

**Region Tiers:**

| Region Tier | Name | Expected Party Level | Enemy Level Range |
|-------------|------|---------------------|-------------------|
| 1 | Borderland | 1–5 | 1–5 |
| 2 | Outer Reaches | 4–8 | 4–9 |
| 3 | Inner Marches | 7–12 | 7–13 |
| 4 | Core Provinces | 11–16 | 11–17 |
| 5 | Empire's Heart | 15–20 | 15–20 |

**Archetypes:**

| Archetype | Role | Relative Stats | Typical AP | XP Value |
|-----------|------|---------------|------------|----------|
| **Grunt** | Baseline enemy — dies in 2–3 hits, moderate damage | 1× health, 1× damage | 4–5 | `20 × enemy_level` |
| **Elite** | Tougher, higher threat — 2× grunt health, 1.5× damage | 2× health, 1.5× damage | 5–7 | `40 × enemy_level` |
| **Boss** | Set-piece enemy — 4× grunt health, 2× damage, unique ability | 4× health, 2× damage | 6–8 | `100 × enemy_level` |

**Stat Table by Region Tier (Grunt baseline):**

| Region Tier | HP | Damage | Accuracy Mod | Evasion | Armor |
|-------------|-----|--------|-------------|---------|-------|
| 1 (Borderland) | 12–20 | 2–5 | +0 to +2 | 8–10 | 0–2 |
| 2 (Outer Reaches) | 18–30 | 4–8 | +1 to +3 | 9–11 | 1–3 |
| 3 (Inner Marches) | 25–42 | 6–12 | +2 to +4 | 10–12 | 2–5 |
| 4 (Core Provinces) | 35–55 | 9–16 | +3 to +5 | 11–13 | 4–8 |
| 5 (Empire's Heart) | 45–70 | 12–20 | +4 to +7 | 12–15 | 6–12 |

**Archetype multipliers applied on top of grunt baseline:**
- Elite: HP ×2, Damage ×1.5, Accuracy +1, Evasion +1, Armor +2
- Boss: HP ×4, Damage ×2, Accuracy +2, Evasion +2, Armor +4, plus 1 unique ability

**Example:** Region tier 3 Elite: HP 50–84 (25–42 × 2), Damage 9–18 (6–12 × 1.5),
Accuracy +3 to +5, Evasion 11–13, Armor 4–7.

**Ability budgets by archetype:**
- Grunt: 0–1 abilities (basic attack, maybe one special)
- Elite: 1–2 abilities (e.g., Power Strike, Shield Bash)
- Boss: 2–4 abilities including 1 unique (e.g., Ley Pulse, Rally Reinforcements)

Enemy abilities use a simplified version of player abilities: fixed damage bonus
(+2 to +8), no resource cost (cooldown-based: 2–4 round cooldown), telegraphed
where thematically appropriate (boss "gathers energy" for one round before the
big attack).

**CR-8: Encounter Budgets**

Every combat encounter is designed, not generated. The "no filler fights" pillar
means each encounter must have a narrative reason to exist.

**Encounter composition budget** (total enemy power per encounter):

| Difficulty | Budget (Grunt Equivalents) | Typical Composition | Use |
|-----------|---------------------------|---------------------|-----|
| **Minor** | 2–3 GE | 2–3 grunts, or 1 elite | Guards at a door, wildlife defending nest |
| **Standard** | 4–6 GE | 1 elite + 2–3 grunts, or 4–5 grunts | Patrol encounter, ruin defenders |
| **Major** | 7–10 GE | 1 boss + 1–2 elites, or 3–4 elites | Thread-climax fight, faction showdown |
| **Set-Piece** | 11–15 GE | 1 boss + 2–3 elites + 2–4 grunts | Region boss, story-defining battle |

**Grunt Equivalent (GE) conversion:**
- 1 Grunt = 1 GE
- 1 Elite = 3 GE
- 1 Boss = 6 GE

**Budget validation rules:**
- Total enemy HP in an encounter ≤ `party_total_HP × 1.5`
- Total enemy damage output per round ≤ `party_total_HP × 0.5`
- Minimum 2 non-boss enemies in any encounter with a boss
- Maximum 8 total combatants per side (performance + readability ceiling)

**Encounter justification tags.** Each encounter must be tagged with its narrative
purpose:
- `thread_guard`: Enemy blocks access to a thread-critical location or NPC
- `faction_conflict`: Fight tied to faction standing (affects reputation on outcome)
- `territorial`: Wildlife or neutral faction defending territory (can be avoided)
- `ambush`: Enemy initiates combat (player may detect via scout before trigger)
- `story_climax`: Boss fight tied to thread resolution or major revelation

**Non-combat resolution.** Where thematically appropriate, encounters support
non-combat paths:
- Territorial: avoid via stealth (scout detection + party Stealth check)
- Thread guard: bypass via dialogue (persuasion, bribe, faction authority)
- Faction conflict: de-escalate via trust or faction standing

An encounter that offers no non-combat path must justify why in its design notes.

### States and Transitions

| State | Meaning |
|-------|---------|
| `Inactive` | No combat — exploration mode |
| `Initiating` | Combat starting — camera transition, initiative rolling |
| `PlayerTurn` | Player-controlled character's turn. AP displayed. Valid moves/attacks highlighted. |
| `EnemyTurn` | Enemy AI turn. Brief delay for readability. |
| `AllyTurn` | AI-controlled ally/party member turn |
| `Resolution` | Combat ended — XP, loot, state changes |

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Character Stats | Combat reads derived stats | AP, initiative, health, evasion from Stats component |
| Entity Framework | Combat reads/writes entity components | Reads Health, Stats, Combatant components; queries entities by spatial position for targeting |
| Camera System | Combat triggers camera state | Camera enters Combat state, frames all combatants |
| Living Environment | Combat pauses game clock | Time frozen during combat |
| Trust & Relationship | Combat may trigger trust events | Saving an NPC in combat, killing someone they care about |
| Thread Investigation | Combat may advance threads | Defeating a key enemy, discovering a clue on a body |
| Party System | Combat reads party composition | Which characters are in the active party |
| Combat UI | Combat UI reads combat state | Displays turn order, AP, health bars, targeting |
| Audio System | Combat triggers combat audio | Combat music, hit sounds, death sounds |
| Save/Load | Save restricted during combat | Manual save only; no quick-save or auto-save |
| Loot & Economy | Combat triggers loot drops | On enemy defeat, calls `request_loot_drop(enemy_type, region_tier)`. L&E returns items + gold placed on enemy body. **Hard** — combat has no reward without loot. |

## Formulas

### Damage Formula

`damage = max(1, weapon_base + attribute_mod + ability_bonus − target_armor)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| weapon_base | Wb | int | 2–20 | Weapon's base damage (e.g., dagger 2–4, greatsword 8–20) |
| attribute_mod | Am | int | −4 to +4 | STR_mod for melee, DEX_mod for ranged |
| ability_bonus | Ab | int | 0–10 | Bonus from active ability (Power Strike +4, etc.) |
| target_armor | Ta | int | 0–15 | Target's damage reduction from armor |

**Output Range:** 1 (minimum) to ~34 (maximum: 20 + 4 + 10 − 0)
**Example:** Longsword (6) + STR 16 (+3) + Power Strike (+4) vs. armor 2 = 6 + 3 + 4 − 2 = 11 damage.

### Hit Roll Formula

`hit_success = (d20 + accuracy >= target_evasion)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| d20 | D | int | 1–20 | Random roll |
| accuracy | Acc | int | varies | Base weapon accuracy + bonuses |
| target_evasion | Ev | int | 6–22 | 10 + DEX_mod + cover + ability bonus |

### Enemy Health Scaling

`grunt_hp_base = region_tier × 8 + 4`
`grunt_hp = grunt_hp_base + random(0, region_tier × 4)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| region_tier | Rt | int | 1–5 | Region tier |
| grunt_hp_base | Hb | int | 12–44 | Minimum grunt HP for this tier |
| grunt_hp | H | int | 12–70 | Final grunt HP after variance |

**Archetype multipliers:** Elite: `grunt_hp × 2`. Boss: `grunt_hp × 4`.
**Output Range:** 12 HP (tier 1 grunt min) to 280 HP (tier 5 boss max)
**Example:** Tier 3 grunt: base = 3 × 8 + 4 = 28, variance = random(0, 12) → 28–40 HP.

### Enemy Damage Scaling

`grunt_damage_min = region_tier × 2`
`grunt_damage_max = region_tier × 3 + 2`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| region_tier | Rt | int | 1–5 | Region tier |

**Archetype multipliers:** Elite: damage ×1.5. Boss: damage ×2.
**Output Range:** 2–5 (tier 1 grunt) to 24–40 (tier 5 boss)
**Example:** Tier 4 grunt: 8–14 damage. Tier 4 boss: 16–28 damage.

**Comparison to player curve:** At region tier 3, a grunt deals 6–11 damage against
a typical player HP of ~50 (CON 12 at level 8). The player survives ~5–8 grunt
hits — enough for tactical play without trivializing threats.

### Encounter Budget Formula

`encounter_budget_GE = sum(enemy_GE_values)`
`validation: enemy_total_HP ≤ party_total_HP × 1.5`
`validation: enemy_damage_per_round ≤ party_total_HP × 0.5`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| enemy_GE_values | GE | int[] | 1, 3, or 6 | Grunt=1, Elite=3, Boss=6 |
| enemy_total_HP | ΣHe | int | varies | Sum of all enemy max HP |
| party_total_HP | ΣHp | int | varies | Sum of all party member max HP |

**Example:** Standard encounter: 1 Elite (3 GE) + 3 Grunts (3 GE) = 6 GE budget.
Enemy HP = 60 (Elite) + 20+18+22 (Grunts) = 120. Party of 4 at level 6 (~40 HP each) = 160.
Validation: 120 ≤ 240 ✓ (1.5×). Damage/round: Elite(7) + 3×Grunts(5) = 22 vs party HP 160.
22 ≤ 80 ✓ (0.5×). Encounter passes both validation rules.

## Edge Cases

- **If all player characters are dying and no one can stabilize**: Game over screen with option to reload last save. Auto-save is not available during combat — player must have saved manually.
- **If an enemy flees combat and the player pursues**: Combat ends when the enemy leaves the combat area. The enemy becomes a neutral entity in exploration mode. If the player attacks again, new combat starts.
- **If a combatant is killed by environmental damage (falling debris, explosion)**: The kill is credited to the environment, not any combatant. No XP for environmental kills unless a player triggered the environment intentionally.
- **If two combatants have identical initiative**: Tiebreak: higher DEX goes first. If still tied: higher entity ID goes first.
- **If combat ends with a companion at 0 HP (VS, knockout)**: The companion revives with 1 HP immediately at combat end. No stabilization required.
- **If combat ends with a companion at 0 HP (Full Vision, permadeath enabled)**: The dying countdown pauses. The player may attempt stabilization as a post-combat action (no AP cost, adjacency required). If the companion was at 0 rounds remaining, they die at combat resolution.
- **If a companion dies and trust also drops below departure threshold in the same combat**: Death takes precedence. The companion is permanently removed. The trust-betrayal departure path is not triggered.
- **If the player character reaches 0 HP at VS (`companion_death_enabled = false`)**: The player character is knocked out (not killed). Combat ends in defeat — game over screen with reload option. The PC knockout rule is always game-over regardless of the `companion_death_enabled` setting; only companions are affected by the knockout/permadeath toggle.

- **If the player enters a region above their level range**: Enemy stat blocks use the region's tier regardless of player level. A level 3 party in a tier 3 region faces tier 3 enemies — the game does not scale enemies to player level. Region gating (travel costs, thread requirements, NPC warnings) is the primary soft-gate.

- **If an encounter's validation rules fail at design time** (enemy HP > party HP × 1.5 or damage/round > party HP × 0.5): The encounter is flagged in the editor with a warning. It can still be placed (set-piece bosses may intentionally exceed budgets) but requires designer sign-off.

- **If all enemies in an encounter are defeated except a fleeing enemy**: Combat ends when no hostile enemies remain in the combat area. Fleeing enemies exit the combat area and become neutral entities in exploration mode.

- **If an encounter offers non-combat resolution and the player succeeds**: Combat does not start. XP is awarded at 50% of combat value (partial reward for successful non-combat resolution). Thread advancement and loot are handled per the resolution path (dialogue rewards, stealth bypass, etc.).

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Character Stats | Combat reads Stats component | AP, initiative, health, evasion. Hard. |
| Entity Framework | Combat reads/writes entities | Spatial queries for targeting, Health component writes. Hard. |
| Camera System | Combat triggers camera state | Combat state with framing. Soft. |
| Living Environment | Combat pauses clock | Time frozen during combat. Soft. |
| Party System | Combat reads party composition + knockout config | Active party for turn order. Reads `companion_death_enabled` flag to route 0-HP companions to Knocked Out (VS) or Dying (FV). **Hard.** |
| Combat UI | Combat UI reads combat state | All combat data for display. Hard. |
| Audio System | Combat triggers audio | Combat music and SFX. Soft. |
| Loot & Economy | Combat triggers loot drops | On enemy defeat, calls `request_loot_drop()`. L&E returns items + gold. **Hard.** |

## Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `base_ap` | 4 | 3–6 | Higher = more actions per turn, faster combat |
| `flanking_bonus` | +2 | +1–+4 | Higher = positioning more impactful |
| `cover_evasion_bonus` | +2 (half), full (blocked) | +1–+4 | Higher = cover more impactful |
| `dying_rounds` | 3 | 2–5 | Higher = more time to save dying characters |
| `crit_multiplier` | 2× | 1.5×–3× | Higher = crits more decisive |
| `companion_death_enabled` | false (VS) | true/false | false = Knocked Out (VS default); true = Dying/Dead (Full Vision). Owned by Party System; TCE reads this flag. |
| `enemy_hp_per_tier` | 8 | 5–12 | Base HP multiplier: `region_tier × N + 4`. Higher = tankier enemies, longer fights |
| `enemy_hp_variance` | 4 | 2–8 | HP variance range: `random(0, region_tier × N)`. Higher = wider HP spread |
| `enemy_damage_min_per_tier` | 2 | 1–3 | Min damage multiplier: `region_tier × N`. Higher = deadlier enemies |
| `enemy_damage_max_per_tier` | 3 | 2–4 | Max damage formula: `region_tier × N + 2`. Higher = wider damage variance |
| `encounter_hp_ratio_cap` | 1.5 | 1.0–2.0 | Max enemy_total_HP / party_total_HP ratio. Higher = harder encounters allowed |
| `encounter_dpr_ratio_cap` | 0.5 | 0.3–0.75 | Max enemy damage-per-round / party HP ratio. Higher = deadlier encounters |
| `max_combatants_per_side` | 8 | 4–12 | Performance + readability ceiling. Higher = larger battles |
| `non_combat_xp_ratio` | 0.5 | 0.25–1.0 | XP awarded for non-combat resolution vs combat value. Higher = stealth/diplomacy more rewarding |

## Visual/Audio Requirements

📌 **Asset Spec** — Combat Visual/Audio required.

| Event | Visual Feedback | Audio Feedback |
|-------|----------------|---------------|
| Combat start | Camera snaps to combat framing; turn order UI appears | Combat music begins (tense, driving); ambient fades |
| Attack (hit) | Weapon swing animation; damage number floats up (oxblood red); target flinch animation | Weapon impact (type-specific: slash, crush, pierce) |
| Attack (miss) | Weapon swing; "Miss" text floats up (grey); target slight dodge | Whoosh (weapon swings through air) |
| Critical hit | Screen shake (2 units, 200ms); larger damage number (amber-gold flash); target stagger | Heavy impact + short rising chime |
| Character dying | Character slumps; portrait border flashes red; 3-round counter appears | Heartbeat (slowing over 3 rounds) |
| Character knocked out (VS) | Character slumps; portrait border dims (grey, no red flash); "KO" indicator | Brief impact sound; no heartbeat countdown |
| Character death | Character collapses; portrait greyed out; death animation | Final impact; music dips briefly |
| Combat end | Victory: brief fanfare, XP summary. Defeat: game over screen | Victory: music resolves, major chord. Defeat: silence then menu music. |

## Acceptance Criteria

- **GIVEN** a character with DEX 16 (+3), **WHEN** their turn begins, **THEN** they have 7 AP (4 + 3) available.

- **GIVEN** a flanking attack (+2), **WHEN** a melee character attacks from behind with an ally adjacent, **THEN** the accuracy roll includes the +2 flanking bonus.

- **GIVEN** a character at 0 HP with `companion_death_enabled = true`, **WHEN** 3 rounds elapse without stabilization, **THEN** the character dies permanently.

- **GIVEN** a companion at 0 HP with `companion_death_enabled = false` (VS), **WHEN** the companion reaches 0 HP, **THEN** they enter the Knocked Out state (cannot act, no death countdown) and revive with 1 HP when combat ends.

- **GIVEN** a companion at 0 HP with `companion_death_enabled = true` (FV) and combat ends with 1+ rounds remaining, **WHEN** the player performs a post-combat stabilization (adjacency required, no AP cost), **THEN** the companion revives with 1 HP. GIVEN 0 rounds remaining, **WHEN** combat ends, **THEN** the companion dies immediately.

- **GIVEN** combat, **WHEN** a manual save is attempted, **THEN** the save succeeds. Quick-save (F5) and auto-save are blocked during combat.

- **GIVEN** all enemies defeated, **WHEN** the last enemy reaches 0 HP, **THEN** combat ends, XP is awarded, the game clock resumes, and the camera returns to Exploration.

- **Performance**: Turn processing (AP calculation, initiative sort, AI decision) completes within 50ms. Attack resolution completes within 1ms.

**Enemy Scaling:**

- **AC-ES-1 — Grunt HP by tier**: GIVEN a region tier 3 grunt enemy, WHEN the enemy is instantiated, THEN its HP is in the range 28–40 (base = 3 × 8 + 4 = 28, variance 0–12).

- **AC-ES-2 — Elite multipliers**: GIVEN a region tier 2 grunt with HP 20, WHEN converted to Elite archetype, THEN HP = 40 (×2), Damage = 1.5× grunt damage, Accuracy = grunt + 1, Evasion = grunt + 1, Armor = grunt + 2.

- **AC-ES-3 — Boss multipliers**: GIVEN a region tier 4 grunt with damage 10–16, WHEN converted to Boss archetype, THEN Damage = 20–32 (×2) and the boss has at least 1 unique ability.

- **AC-ES-4 — Region tier gating**: GIVEN a tier 3 region, WHEN an enemy is placed, THEN the enemy's stats use the tier 3 row of the stat table regardless of the player's level (no level-scaling).

**Encounter Budgets:**

- **AC-EB-1 — Budget calculation**: GIVEN an encounter with 1 Boss (6 GE), 1 Elite (3 GE), and 2 Grunts (2 GE), WHEN the budget is calculated, THEN the total is 11 GE (Set-Piece difficulty).

- **AC-EB-2 — Validation pass**: GIVEN a Standard encounter (4 GE: 1 Elite + 1 Grunt) against a party of 4 at level 6 (~160 total HP), WHEN validation runs, THEN enemy_total_HP ≤ 240 (160 × 1.5) and enemy_damage_per_round ≤ 80 (160 × 0.5). Both pass.

- **AC-EB-3 — Validation fail**: GIVEN an encounter with enemy_total_HP = 500 against a party with total_HP = 200, WHEN validation runs, THEN the encounter is flagged with a warning (500 > 300).

- **AC-EB-4 — Non-combat XP**: GIVEN a territorial encounter resolved via stealth (no combat), WHEN XP is awarded, THEN the party receives 50% of the combat XP value for those enemies.

- **AC-EB-5 — Encounter justification tag**: GIVEN a designed encounter, WHEN it is saved to encounter data, THEN it has at least one narrative justification tag (`thread_guard`, `faction_conflict`, `territorial`, `ambush`, or `story_climax`).

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Grid type: square tiles or hex? Square is simpler for isometric, hex gives more tactical depth. | Game Designer | Before implementation (ADR) |
| Should there be a non-lethal combat option (knocking out enemies for interrogation)? | Game Designer | Before vertical slice — partially resolved: companion knockout exists at VS via `companion_death_enabled`. Enemy knockout remains an open design question. |
| Enemy AI: simple state machine (attack nearest, use ability on cooldown) or more sophisticated (flanking, targeting low-HP characters)? | AI Programmer | Before implementation |
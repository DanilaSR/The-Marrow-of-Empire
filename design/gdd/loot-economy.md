# Loot & Economy

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 2 (The World Remembers — what you find matters), Pillar 3 (Tactical Depth — equipment drives build choices)

## Overview

The Loot & Economy system is the resource layer of The Marrow of Empire — it
defines what the player finds, what it's worth, and what they can do with it.
On the data side, it owns the `Inventory` component's write path (items,
equipment_slots, gold), defines loot tables per enemy type and container type,
and establishes the equipment acquisition curve that gates combat progression.
On the player side, it's the reward engine of exploration: the sword in the
ruin, the gold in the bandit's chest, the merchant who recognizes your faction
standing and names a fair price — or doesn't.

The system provides:

- **Loot tables**: What enemies drop, what containers hold, and what
  investigating a scene yields. Tables are keyed by enemy type, region tier,
  and container category. Every drop has a weight and a conditional gate
  (minimum region, minimum player level, thread requirement).
- **Equipment curve**: Weapon and armor tiers mapped against region
  progression. The difference between a rusted borderland blade (2–4 damage)
  and an Old Empire Ley-forged sword (16–20 damage) is the combat power
  curve — equipment matters more than attributes, and loot gates the player's
  tactical options.
- **Gold economy**: Sources (enemy drops, thread rewards, barter sales)
  balanced against sinks (shop purchases, item identification, bribes,
  lodging, travel costs). Gold must feel earned and must have somewhere to
  go — unbounded accumulation makes gold meaningless.
- **Shop system**: Merchants with inventory keyed to region tier, faction
  standing, and trust level. Prices modified by faction standing and CHA
  barter. Some items only appear when the merchant trusts you enough.
- **Item rarity and identification**: Five rarity tiers (Common, Uncommon,
  Rare, Epic, Legendary) governing stat budgets and acquisition frequency.
  Unidentified items must be identified (cost in gold or Lore skill) before
  stats apply.

Without the Loot & Economy system: combat has no reward, inventory has no
input, gold accumulates forever with no purpose, and finding a sword in a
ruin is mechanically identical to buying one from a merchant — which is to say,
not an RPG.

## Player Fantasy

The Loot & Economy system serves two layers of a single promise: **what you find
means something.**

**"This was worth it."** Combat in The Marrow of Empire has real stakes —
permadeath at Full Vision, knockout at VS, every encounter a risk. The reward
must justify the risk. When the player searches a bandit captain's body and finds
a Ley-forged dagger with +3 damage and an Old Empire inscription, the fight
retroactively becomes a story about what they gained. When they find 12 gold and
a rusted blade, the fight becomes a story about what they risked for nothing.
The loot system's job is to make the first story common and the second story a
deliberate warning sign — "you're in the wrong region for your level."

**"I earned this."** Gold is not a score. It's a tool, and tools are measured by
what they unlock. The player who saves 500 gold for a suit of borderland plate
armor feels the weight of that purchase — those 500 gold represent cleared ruins,
resolved threads, and risks survived. The player who spends their last 50 gold
on a bribe to save a companion knows exactly what that gold was worth. Economic
choices with real trade-offs create the feeling of agency that a gold counter
alone never can.

**"This has a history."** Items in The Marrow of Empire are not procedurally
generated stat sticks. A sword found in an Old Empire ruin has a different
provenance than one bought from a borderland smith. Rarity communicates history:
Common items are everyday tools, Uncommon are well-made, Rare are pre-collapse
craftsmanship, Epic are named items with known histories, Legendary are
one-of-a-kind artifacts tied to thread chains. The identification mechanic
reinforces this — an unidentified Rare sword is a mystery; an identified one is
a discovery.

**"They know me here."** Shops are not vending machines. A merchant in a
faction-allied town gives better prices. A merchant who trusts you (trust level
2+) shows you the good stock. A merchant whose faction you've betrayed either
overcharges or refuses service. The economy is a social system — your reputation
precedes you, and it's priced in.

**Reference**: Baldur's Gate 1-2 (unique named items with lore — every +2 sword
has a story), Divinity Original Sin 2 (equipment as the primary progression
vector — finding new gear is transformative), Darkest Dungeon (the tension
between risking for loot and preserving your party).

## Detailed Design

### Core Rules

**CR-1: Item Definition**

Every item in The Marrow of Empire is defined by a data record with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `item_id` | string | Unique identifier (e.g., `"borderland_iron_sword"`) |
| `name` | string | Display name |
| `type` | enum | Weapon, Armor, Consumable, Quest, CraftingMaterial, Accessory, Readable |
| `subtype` | enum | For weapons: Sword, Axe, Dagger, Bow, Spear. For armor: Head, Chest, Hands, Legs |
| `rarity` | enum | Common, Uncommon, Rare, Epic, Legendary (CR-2) |
| `stats` | map[string]int | Stat modifiers applied when equipped (e.g., `{"damage": 4, "accuracy": 1}`) |
| `weight` | float | Weight in lbs — contributes to carry capacity |
| `value` | int | Base gold value (before faction/CHA modifiers) |
| `stack_size` | int | Max copies in one inventory cell (1 for equipment, >1 for consumables/materials) |
| `description` | string | Flavor text shown in item detail panel |
| `icon_ref` | string | Asset reference for the item icon |
| `equipment_slot` | string | Which slot this occupies if equipment (from Equipment Slots registry) |
| `required_attrs` | map[string]int | Minimum attributes to equip (e.g., `{"STR": 12}` for heavy weapons) |
| `identified` | bool | Whether stats are revealed (CR-7) |
| `region_tier` | int | Minimum region tier where this item can appear (1–5) |

**CR-2: Rarity Tiers**

Five rarity tiers govern stat budgets, acquisition frequency, and value:

| Rarity | Color | Drop Rate | Stat Budget (total mod points) | Value Multiplier | Typical Source |
|--------|-------|-----------|-------------------------------|-----------------|----------------|
| **Common** | Grey | 60% | 1–2 points | 1× | Any enemy, any container |
| **Uncommon** | Green | 25% | 3–4 points | 2.5× | Region tier 1+, elite enemies |
| **Rare** | Blue | 10% | 5–7 points | 5× | Region tier 2+, bosses, hidden caches |
| **Epic** | Purple | 4% | 8–10 points | 12× | Region tier 3+, named enemies, thread rewards |
| **Legendary** | Gold | 1% | 11–15 points | 25× | Unique — one per item, tied to thread chains or faction milestones |

**Stat budget**: Each point buys +1 damage, +1 armor, +1 accuracy, +1 to an
attribute, +5 HP, or equivalent. A Common sword with 2 points might be "+1 damage,
+1 accuracy." A Legendary sword with 14 points might be "+5 damage, +3 accuracy,
+2 STR, +10 HP, bleed on critical."

**CR-3: Loot Tables**

Loot tables are defined per source category and region tier:

| Source Category | Rolls | Guaranteed | Examples |
|----------------|-------|-----------|----------|
| `enemy_grunt` | 1–2 rolls | Gold (small) | Bandits, wildlife, militia |
| `enemy_elite` | 2–3 rolls | Gold (medium) + equipment (Uncommon+) | Captains, named enemies |
| `enemy_boss` | 3–5 rolls | Gold (large) + equipment (Rare+) + unique drop chance | Region bosses |
| `container_common` | 1 roll | Gold (small) or consumable | Barrels, crates, desks |
| `container_treasure` | 2–4 rolls | Gold (medium) + equipment chance | Locked chests, hidden caches |
| `container_thread` | 2–3 rolls + quest item | Quest item + gold + equipment chance | Investigation scenes |
| `merchant_inventory` | Per shop tier (CR-6) | Always has consumables | Shopkeepers |

Each roll independently selects an item from the weighted table for that
source + region_tier combination. Items with `region_tier > current_region`
are excluded.

**Conditional gates**: Individual items can specify additional gates:
- `requires_thread: "thread_id"` — only drops when thread is active
- `requires_trust_level: 2` — merchant only shows at trust 2+
- `requires_faction_standing: 50` — only drops for allied faction members
- `requires_discovery: "location_id"` — only in specific locations

**CR-4: Equipment Curve**

Weapon damage and armor values scale with region tier:

| Region Tier | Weapon Damage | Armor Range | Expected Level | Example |
|-------------|--------------|-------------|---------------|---------|
| 1 (Borderland) | 2–6 | 1–3 | 1–5 | Rusted Iron Sword (2–4, Common) |
| 2 (Outer Reaches) | 5–10 | 3–6 | 4–8 | Steel Broadsword (6–8, Uncommon) |
| 3 (Inner Marches) | 8–14 | 5–9 | 7–12 | Ley-Forged Blade (10–12, Rare) |
| 4 (Core Provinces) | 12–18 | 8–12 | 11–16 | Imperial Legion Sword (14–16, Epic) |
| 5 (Empire's Heart) | 16–20 | 11–15 | 15–20 | Emperor's Own Blade (18–20, Legendary) |

**Interaction with TCE damage formula**: TCE defines `damage = weapon_base +
STR_mod − target_armor + ability_bonus`. Equipment contributes ~60-70% of total
damage; attributes ~15-20%; abilities ~15-20%. Equipment is the primary
progression driver; attributes define build nuance.

**Minimum equip requirements**: Higher-tier equipment may require minimum
attributes (e.g., Imperial Legion Sword requires STR 14). This preserves the
"attributes ARE the class" identity — your build determines which equipment
curve you can access.

**CR-5: Gold Economy**

**Sources** (per typical encounter/event):

| Source | Amount | Frequency |
|--------|--------|-----------|
| Enemy grunt (drop or body loot) | 2–8 gold | Per enemy |
| Enemy elite | 15–40 gold | Per elite |
| Enemy boss | 50–200 gold | Per boss |
| Container common | 1–10 gold | Per container |
| Container treasure | 20–80 gold | Per container |
| Thread resolution reward | 50–200 gold | Per thread |
| Selling items to merchant | 25% of item value | Per transaction |
| Faction bounty | 100–500 gold | Per bounty |

**Sinks** (ongoing costs):

| Sink | Cost | Frequency |
|------|------|-----------|
| Shop purchase | Item value × price_multiplier | Per item |
| Item identification | 25 × rarity_tier × region_tier | Per item |
| Bribe (conversation) | 50–500 gold | Occasional |
| Lodging (inn rest) | 10–30 gold per night | Per rest |
| Travel (caravan/ferry) | 20–100 gold per crossing | Per crossing |
| Consumable restock | 5–25 gold per item | As needed |
| Faction tribute | 100–1000 gold | Per milestone |

**Balance target**: A player who clears one region (~15 encounters, 8 containers,
2 threads, 1 boss) should net approximately 400-800 gold after expenses — enough
for one significant equipment upgrade.

**CR-6: Shop System**

Merchants are NPCs with defined shop inventory tiers:

| Shop Tier | Region | Inventory Size | Rarity Ceiling | Restock Interval |
|-----------|--------|---------------|----------------|-----------------|
| 1 | Borderland | 8–12 items | Uncommon | 72 game hours |
| 2 | Outer Reaches | 12–18 items | Rare | 96 game hours |
| 3 | Inner Marches | 15–22 items | Epic | 120 game hours |
| 4 | Core Provinces | 18–25 items | Epic+ | 168 game hours |
| 5 | Empire's Heart | 20–30 items | Legendary (one) | 240 game hours |

**Pricing formula**: `final_price = base_value × price_multiplier × barter_modifier`

Where:
- `price_multiplier` from Faction System: `1.0 − standing/400` (range 0.75× to 1.25×)
- `barter_modifier` from CHA: `1.0 − (CHA_mod × 0.05)` (range 0.80× to 1.20×)
- Combined range: ~0.60× (allied + high CHA) to ~1.50× (hostile + low CHA)
- Minimum floor: 0.25× of base value (items cannot be free)
- Selling to merchant: always 25% of base value (unaffected by multipliers)

**Trust-gated inventory**: Specific items can require `trust_level ≥ N` to appear.
A village smith might have a Rare sword that only appears at trust 2.

**CR-7: Item Identification**

Items found in the world may be unidentified:

- **Auto-identify**: Common and Uncommon items auto-identify on pickup.
  Rare+ items from non-merchant sources are unidentified.
- **Identification cost**: `25 × rarity_tier × region_tier` gold at any merchant
  or camp. INT discount: `cost × (1.0 − INT_mod × 0.1)`, minimum 5 gold.
- **Unidentified behavior**: Can be equipped but provide NO stat bonuses.
  Display shows "???" for stats and "Unidentified [Type]" as the name.
- **Lore self-identify**: INT ≥ 14 can identify Rare items at camp (free,
  once per rest). INT ≥ 16 can identify Epic items. Legendary items always
  require a merchant or specific thread event.

**CR-8: Container Interaction**

Containers use the EF `Container` component and `InteractionType.loot`:

- **Loot interaction**: Click container → loot UI (container contents left,
  player inventory right). "Take All" transfers everything weight permits.
- **Body looting**: Enemy bodies persist after combat. Contain equipped items +
  gold drop. Despawn on region exit or after 72 game hours.
- **Locked containers**: Require key, lockpick (DEX check), or forced open
  (STR check, 50% chance to destroy one random item).
- **Already looted**: Container shows "Empty" on hover after looting.

### States and Transitions

**Item States:**

| State | Meaning |
|-------|---------|
| `InWorld` | Item exists in the game world (on ground, in container, on body) |
| `InInventory` | Item is in a character's inventory |
| `Equipped` | Item is in an equipment slot and stats are applied |
| `Identified` | Item stats are revealed to the player |
| `Sold` | Item has been sold to a merchant (enters merchant inventory until restock) |
| `Destroyed` | Item is permanently removed (forced container opening, quest consumption) |

**Transitions:**
- `InWorld → InInventory`: Player picks up or loots item.
- `InInventory → Equipped`: Player drags item to compatible equipment slot.
- `Equipped → InInventory`: Player unequips item.
- `Unidentified → Identified`: Pay identification cost or use Lore self-identify.
- `InInventory → Sold`: Sell to merchant. Item enters merchant buyback pool for 72h.
- `InInventory → InWorld`: Player drops item.
- `Any → Destroyed`: Locked container forced open, quest consumption, or player
  deliberately destroys (discard with confirmation for Rare+).

**Gold States:**

| State | Meaning |
|-------|---------|
| `InWorld` | Gold on a body, in a container, or as a world pickup |
| `InInventory` | Gold in player's inventory (displayed in C&I UI) |
| `Spent` | Gold removed via sink (purchase, bribe, etc.) |

Gold has no cap beyond the display cap (999,999). Gold is never "dropped" as a
world item — it's always auto-looted on interaction.

**Shop Interaction States:**

| State | Meaning |
|-------|---------|
| `Browsing` | Player viewing merchant inventory; can buy, sell, or identify |
| `TransactionConfirm` | Confirmation dialog for purchase/sale with final price shown |
| `Closed` | Shop UI closed; merchant returns to idle |

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | L&E depends on EF | Reads/writes `Inventory` component (items, equipment_slots, gold). Reads `Container` component (contents, capacity, is_locked). Reads `Stats` for equip requirements and identification checks. **Hard.** |
| Tactical Combat Engine | TCE depends on L&E | TCE triggers loot drops on enemy death via `request_loot_drop(enemy_type, region_tier)` → L&E returns items + gold. TCE reads equipped item stats for damage calculation. **Hard.** |
| Character Stats & Progression | L&E depends on Char Stats | Reads STR for carry capacity, equip requirements, and forced-open checks. Reads INT for Lore identification discount and self-identify threshold. Reads CHA for barter modifier. **Hard.** |
| Faction System | L&E depends on Faction | Reads `faction.standing` for price_multiplier and shop access (Hostile may refuse service). **Hard.** |
| Character & Inventory UI | C&I UI depends on L&E | UI reads item data, gold amount, equipment slot state. UI triggers equip/unequip/sell/drop/identify actions. L&E provides shop UI data. **Hard.** |
| Trust & Relationship | L&E depends on T&R | Reads `trust_level` for trust-gated shop inventory and merchant buyback availability. **Soft — shops function without trust gating.** |
| Save/Load System | L&E depends on Save/Load | Inventory and Container component data serialized via EF. Loot table state (already-rolled containers, restock timers) persisted. **Hard.** |
| Asset & Scene Pipeline | L&E depends on Pipeline | Item icon references (`icon_ref`) resolve to pipeline assets. **Soft — items function with placeholder icons.**

## Formulas

### Item Base Value

The `item_base_value` formula determines an item's gold value from rarity and region:

`base_value = rarity_value_multiplier × region_tier × 20`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| rarity_value_multiplier | Rm | int | 1–25 | Value multiplier from rarity tier (CR-2) |
| region_tier | Rt | int | 1–5 | Region tier the item belongs to |

**Rarity multipliers:** Common=1, Uncommon=2.5, Rare=5, Epic=12, Legendary=25
**Output Range:** 20 (Common, tier 1) to 2500 (Legendary, tier 5)
**Example:** Rare sword (Rm=5) at region tier 3: 5 × 3 × 20 = 300 gold base value.

### Final Price (Shop Purchase)

The `final_price` formula combines faction standing and CHA barter:

`final_price = base_value × faction_multiplier × barter_modifier`
`final_price = max(final_price, base_value × 0.25)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| base_value | V | int | 20–2500 | Item's base gold value |
| faction_multiplier | Fm | float | 0.75–1.25 | From Faction System: `1.0 − standing/400` |
| barter_modifier | Bm | float | 0.80–1.20 | From CHA: `1.0 − (CHA_mod × 0.05)` |

**Output Range:** 0.25× base_value (hard floor) to ~1.50× base_value (hostile + low CHA)
**Example:** Base 300g, faction standing 100 (Fm=0.75), CHA 14 (Bm=0.90). Price = 300 × 0.75 × 0.90 = 202.5 → 203 gold. Without faction/CHA: 300 gold. Saving: 97 gold.

### Selling Price

`sell_price = base_value × 0.25`

Faction and CHA modifiers do NOT affect selling price. Always 25% of base value.
**Example:** Item with base value 300 gold sells for 75 gold.

### Identification Cost

The `id_cost` formula determines how much it costs to identify an item:

`id_cost = 25 × rarity_tier × region_tier`
`id_cost = max(id_cost × (1.0 − INT_mod × 0.1), 5)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| rarity_tier | R | int | 3–5 | Only Rare(3), Epic(4), Legendary(5) need identification |
| region_tier | Rt | int | 1–5 | Region tier |
| INT_mod | Im | int | −4 to +4 | Intelligence modifier for discount |

**Output Range:** 15 gold (Rare, tier 1, INT 18) to 625 gold (Legendary, tier 5, INT 3)
**Example:** Epic item (R=4), tier 3, INT 14 (Im=+2): 25 × 4 × 3 = 300. Discount: 300 × 0.8 = 240 gold.

### Equipment Damage Scaling

The `weapon_damage_range` formula maps region tier to expected weapon damage:

`damage_min = max(2, floor(region_tier × 3.5 − 2))`
`damage_max = max(4, floor(region_tier × 3.5 + 2))`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| region_tier | Rt | int | 1–5 | Region tier |

**Output Range:** 2–6 (tier 1) to 16–20 (tier 5)
**Example:** Region tier 3: min = floor(10.5 − 2) = 8, max = floor(10.5 + 2) = 12. Range: 8–14.

### Loot Roll Probability

Each loot table roll uses weighted random selection:

`P(item) = item_weight / sum(all_weights_in_table)`

**Variables:**
| Variable | Symbol | Type | Range | Description |
|----------|--------|------|-------|-------------|
| item_weight | Wi | float | 0–100 | Weight adjusted by rarity drop rate × conditional modifiers |
| sum_weights | ΣW | float | >0 | Sum of all eligible item weights for this source + region |

**Eligibility**: Item excluded if `region_tier > current_region_tier` or conditional
gate not met.
**Example:** Enemy elite at tier 2 with table: Common sword (W=60), Uncommon axe (W=25),
Rare dagger (W=10), Epic bow (W=4), Legendary artifact (W=1). ΣW = 100.
P(Rare dagger) = 10/100 = 10%.

## Edge Cases

- **If the player's inventory is full when looting**: Items remain in the
  container/body. A toast appears: "Inventory full — [N] items remain."
  The container shows a "partially looted" state on hover. Items are not lost.

- **If a container is opened by multiple party members**: Any party member can
  loot any container. Items taken by one member are removed for all. Gold is
  deposited to the shared party gold pool regardless of which member loots.

- **If an enemy body despawns (72 game hours) before being looted**: Items on
  the body are lost. A journal notification is NOT generated — unlooted bodies
  are the player's choice. Exception: thread-critical items are auto-looted
  to a "recovered items" tab in inventory with a notification.

- **If the player sells a quest item**: Quest items are tagged `type: Quest`
  and cannot be sold or dropped. The sell/drop actions are greyed out with
  tooltip: "This item may be important."

- **If the player attempts to equip an item without meeting requirements**:
  The equipment slot shows a red X. Tooltip: "Requires STR 14 (current: 10)."
  The item can still be placed in inventory but not equipped. Stats are not
  applied.

- **If a merchant's faction standing changes to Hostile while the shop UI is
  open**: The shop remains open for the current session. Prices reflect the
  new standing on next open. If Hostile refuses service, the current session
  completes normally; the merchant is unavailable on next interaction.

- **If the player identifies an item they already have equipped**: The item
  immediately reveals its stats. If the stats include bonuses the player was
  already receiving (because they equipped unidentified), the stats "snap"
  to their real values retroactively — no double-dip or gap.

- **If gold would overflow the display cap (999,999)**: Display shows "999,999+".
  Gold accumulation continues beyond the cap. No gameplay limit on gold
  accumulation. The cap is purely cosmetic.

- **If a Legendary item's thread requirement is failed (thread dead_end)**:
  The item becomes unobtainable through that path. Alternative acquisition
  paths may exist (different thread chain, faction milestone). Legendary
  items are never permanently missable — at least one path always remains.

- **If two items grant bonuses to the same stat**: Bonuses stack additively.
  A +2 STR ring and +1 STR boots = +3 STR total. Equipment bonuses +
  temporary effects + base attribute = displayed total on character sheet.

- **If the player rests at an inn consecutively without adventuring**:
  Restock timers advance normally. Merchant gold resets. No penalty for
  consecutive rests — the time cost (game clock advancement) is the only
  soft deterrent.

- **If a container is destroyed (environmental damage, collapse)**: All
  contents are lost. No recovery. Containers with thread-critical items are
  tagged `indestructible: true` and cannot be destroyed.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| Entity Framework | L&E depends on EF | Reads/writes `Inventory` and `Container` components. Reads `Stats` for equip requirements and identification. **Hard** — cannot function without entity data. |
| Tactical Combat Engine | TCE depends on L&E | TCE triggers loot drops; L&E returns items + gold. TCE reads equipped item stats for damage. **Hard** — combat has no reward without loot. |
| Character Stats & Progression | L&E depends on Char Stats | Reads STR (equip requirements, carry capacity), INT (identification discount, self-identify), CHA (barter modifier). **Hard** — pricing and requirements need attributes. |
| Faction System | L&E depends on Faction | Reads `faction.standing` for price_multiplier. Hostile factions may refuse shop service. **Hard** — economy is faction-aware. |
| Character & Inventory UI | C&I UI depends on L&E | UI reads item data, gold, equipment state. Triggers equip/unequip/sell/drop/identify. Shop UI provided by L&E. **Hard** — inventory is empty without items. |
| Trust & Relationship | L&E depends on T&R | Reads `trust_level` for trust-gated shop inventory. **Soft** — shops function without trust gating; gating is additive. |
| Save/Load System | L&E depends on Save/Load | Inventory/Container data serialized via EF. Restock timers and looted-container state persisted. **Hard** — items must survive sessions. |
| Asset & Scene Pipeline | L&E depends on Pipeline | Item icons resolved through pipeline. **Soft** — placeholder icons are acceptable. |

## Tuning Knobs

| Parameter | Default | Safe Range | Effect |
|-----------|---------|------------|--------|
| `rarity_value_multipliers` | {C:1, U:2.5, R:5, E:12, L:25} | Adjust ±50% | Higher = larger gold gaps between rarities |
| `sell_price_ratio` | 0.25 | 0.10–0.50 | Higher = selling feels more rewarding, less incentive to shop |
| `id_cost_base` | 25 | 10–50 | Higher = identification more of a gold sink |
| `id_int_discount_per_mod` | 0.10 | 0.05–0.15 | Higher = INT matters more for identification |
| `barter_cha_per_mod` | 0.05 | 0.03–0.08 | Higher = CHA matters more for shop prices |
| `price_floor_ratio` | 0.25 | 0.10–0.50 | Higher = less discount possible; lower = faction/CHA more impactful |
| `weapon_damage_scale` | 3.5 | 2.5–4.5 | Per region tier. Higher = steeper equipment curve |
| `weapon_damage_spread` | 2 | 1–3 | ± from midpoint. Higher = wider damage variance per tier |
| `container_lock_break_destroy_chance` | 0.50 | 0.25–0.75 | Higher = riskier forced opening |
| `enemy_body_despawn_hours` | 72 | 24–168 | Longer = more time to return and loot |
| `merchant_restock_base_hours` | 72 | 48–240 | Base restock interval (scales with shop tier) |
| `merchant_buyback_hours` | 72 | 24–168 | How long sold items remain repurchaseable |
| `gold_small` | 2–8 | 1–15 | Gold range for grunt enemies / common containers |
| `gold_medium` | 15–40 | 10–80 | Gold range for elite enemies / treasure containers |
| `gold_large` | 50–200 | 30–500 | Gold range for bosses |
| `gold_thread_reward` | 50–200 | 25–500 | Gold range for thread resolution |
| `gold_faction_bounty` | 100–500 | 50–1000 | Gold range for faction bounties |
| `legendary_always_obtainable` | true | true/false | If true, at least one path to each Legendary always remains open |

## Visual/Audio Requirements

[To be designed]

## UI Requirements

[To be designed]

## Acceptance Criteria

**Item System:**

- **AC-1 — Item creation**: GIVEN an item definition with `item_id: "steel_broadsword"`,
  `type: Weapon`, `rarity: Uncommon`, `stats: {damage: 7, accuracy: 1}`,
  `region_tier: 2`, WHEN the item is instantiated, THEN all fields are populated
  and `identified` is false.

- **AC-2 — Rarity distribution**: GIVEN a loot table with 1000 rolls at region
  tier 2 using default drop rates, WHEN rolls are executed, THEN Common appears
  550-650 times, Uncommon 200-300 times, Rare 70-130 times, Epic 20-60 times,
  Legendary 5-25 times (±2σ tolerance).

- **AC-3 — Region tier gating**: GIVEN an item with `region_tier: 3` in a loot
  table, WHEN rolled at `current_region_tier: 2`, THEN the item is excluded from
  the eligible pool and cannot drop.

**Gold Economy:**

- **AC-4 — Gold source accumulation**: GIVEN a region 1 clear (15 grunts, 3 elites,
  1 boss, 8 containers, 2 threads), WHEN all loot is collected and sold, THEN
  total gold earned is in the range 350-950 gold (within ±30% of 600 gold target).

- **AC-5 — Gold sinks available**: GIVEN a player with 500 gold at region tier 2,
  WHEN they visit a tier 2 shop, buy one Uncommon item (~150g), identify one Rare
  item (~75g), and rest at an inn (~20g), THEN at least 200 gold remains for
  future purchases (sinks do not drain to zero in one visit).

**Shop System:**

- **AC-6 — Faction price modifier**: GIVEN an item with base value 200 gold at a
  merchant with faction standing +100 (allied), WHEN the player views the buy price,
  THEN the price is 200 × 0.75 = 150 gold (±1g rounding).

- **AC-7 — CHA barter modifier**: GIVEN the same item at a neutral merchant
  (standing 0, Fm=1.0) and a player with CHA 16 (+3), WHEN the player views the
  buy price, THEN the price is 200 × 1.0 × 0.85 = 170 gold (±1g rounding).

- **AC-8 — Combined modifiers**: GIVEN an item with base value 300 gold,
  faction standing −50 (Fm=1.125) and CHA 6 (−2, Bm=1.10), WHEN the player views
  the buy price, THEN the price is 300 × 1.125 × 1.10 = 371 gold (±1g rounding).

- **AC-9 — Price floor**: GIVEN an item with base value 100 gold, faction standing
  +200 (Fm=0.50) and CHA 18 (Bm=0.80), WHEN combined modifier would produce
  100 × 0.50 × 0.80 = 40 gold, THEN the price is floored at 100 × 0.25 = 25 gold.

- **AC-10 — Hostile refusal**: GIVEN a merchant whose faction standing is −76 or
  lower (Hostile), WHEN the player attempts to open the shop, THEN the merchant
  refuses service with dialogue and the shop UI does not open.

- **AC-11 — Trust-gated inventory**: GIVEN a tier 1 merchant with a Rare sword
  gated behind `requires_trust_level: 2` and player trust level 1, WHEN the shop
  is opened, THEN the Rare sword is absent from inventory.

**Identification:**

- **AC-12 — Unidentified item hides stats**: GIVEN an unidentified Rare sword with
  `damage: 8, accuracy: 2`, WHEN viewed in inventory, THEN the name reads
  "Unidentified Sword" and stats show "???".

- **AC-13 — Identification reveals stats**: GIVEN the same item, WHEN the player
  pays the identification cost (25 × 3 × 2 = 150 gold at region tier 2), THEN
  the name updates to its real name and stats display as "Damage: 8, Accuracy: 2".

- **AC-14 — INT discount**: GIVEN an Epic item (rarity 4) at region tier 3
  (base id_cost = 25 × 4 × 3 = 300) and a player with INT 16 (+3), WHEN the
  identification cost is calculated, THEN it is 300 × (1.0 − 0.3) = 210 gold.

- **AC-15 — Lore self-identify**: GIVEN a Rare item and a character with INT 14,
  WHEN the party rests at camp, THEN the player can identify the item for free
  (once per rest).

**Equipment Requirements:**

- **AC-16 — Requirement met**: GIVEN a sword requiring STR 14 and a character
  with STR 14, WHEN the player drags the sword to the Main Hand slot, THEN the
  sword equips and its stats are applied to derived calculations.

- **AC-17 — Requirement not met**: GIVEN a sword requiring STR 14 and a character
  with STR 10, WHEN the player drags the sword to the Main Hand slot, THEN the
  slot shows a red X, the item returns to inventory, and a tooltip displays
  "Requires STR 14 (current: 10)."

**Container Interaction:**

- **AC-18 — Container loot**: GIVEN a treasure chest with 3 items at region tier 1,
  WHEN the player opens it, THEN the loot UI shows container contents on the left,
  player inventory on the right, and a "Take All" button.

- **AC-19 — Locked container (key)**: GIVEN a locked chest and the player has the
  matching key, WHEN the player interacts, THEN the chest unlocks and opens normally.

- **AC-20 — Locked container (forced)**: GIVEN a locked chest with 4 items and a
  character with STR 16, WHEN the player forces it open, THEN the chest opens
  with a 50% chance that one random item is destroyed.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Should consumable items (healing poultices, bandages) exist as a distinct item category with their own acquisition curve, or are they folded into shop restocks only? | Game Designer | Before VS implementation | — |
| Should crafting exist as a system (combining materials into items), or is all equipment acquired through loot/shops? If crafting exists, it needs its own GDD. | Game Designer | Before Alpha | — |
| Should Legendary items have unique models/icons, or is the gold name color sufficient visual differentiation at MVP? | Art Director | Before VS implementation | — |
| Should merchants have persistent gold pools (limited buying power) or unlimited gold for player sales? Currently designed as unlimited — is this correct for economy balance? | Economy Designer | Before VS implementation | — |
| Should item durability/degradation exist as a gold sink, or are the existing sinks (identification, lodging, bribes, travel) sufficient? | Game Designer | Before VS implementation | — |
| Should there be a "junk" item category (vendor trash with no use except selling), or should every item have at least a theoretical use? | Game Designer | Before VS implementation | — |
| Should the loot UI be a separate screen or an overlay panel on top of the game world (like BG3's area loot)? | UX Designer | Before VS implementation | — |
| How should item data be authored — YAML/JSON files per region, or a single item database? → becomes an ADR. | Lead Programmer | Before implementation (ADR) | — |

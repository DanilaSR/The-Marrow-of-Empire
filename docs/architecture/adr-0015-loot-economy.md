# ADR-0015: Loot & Economy System

## Status
Accepted

## Date
2026-08-12
**Accepted**: 2026-08-12 — after /architecture-review verification

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core / Economy |
| **Knowledge Risk** | LOW — pure game logic and data structures |
| **References Consulted** | `design/gdd/loot-economy.md`; `design/gdd/character-stats-progression.md`; `design/gdd/tactical-combat-engine.md`; `design/gdd/faction-system.md`; `design/gdd/character-inventory-ui.md` |
| **Post-Cutoff APIs Used** | None — pure game logic |
| **Verification Required** | None — data model and formulas are engine-agnostic |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Framework — Inventory and Container components); ADR-0006 (Character Stats — STR/INT/CHA for equip requirements, identification, barter); ADR-0009 (Tactical Combat Engine — loot drop trigger, damage formula using equipped stats); ADR-0011 (Event Bus — loot-related events) |
| **Enables** | Character & Inventory UI (item display, equip/unequip, shop UI); Combat UI (post-combat loot interaction) |
| **Blocks** | Loot & Economy implementation — 8 TRs uncovered |
| **Ordering Note** | Must be Accepted after ADR-0006 and ADR-0009 (reads STR/INT/CHA, receives loot triggers from TCE). C&I UI depends on this ADR for item data schema and equip contract. |

## Context

### Problem Statement
The Loot & Economy system has no architectural specification. Eight technical requirements (TR-loot-001 through TR-loot-008) are uncovered by any ADR. The system must define: item data schema, rarity tiers with stat budgets, weighted loot tables with conditional gates, region-tiered equipment progression, gold economy with balanced sources and sinks, shop pricing driven by faction standing and character attributes, item identification mechanics, and container interaction contracts with Entity Framework.

### Constraints
- Equipment contributes ~60-70% of total damage (more than attributes + abilities combined) — the equipment curve IS the progression curve
- Must be data-driven: items authored as YAML files per region, not hardcoded in Go
- L&E owns the `Inventory` component write path; other systems read or trigger actions
- Gold must feel earned and must have sinks — unbounded accumulation makes gold meaningless
- Must function without faction/trust systems (soft dependencies — shops work without them)
- Quest items must be unsellable, undroppable
- Legendary items must remain obtainable through at least one path at all times

### Requirements
- Item data schema with 13+ fields (item_id, name, type, subtype, rarity, stats, weight, value, stack_size, description, icon_ref, equipment_slot, required_attrs, identified, region_tier)
- 5 rarity tiers with defined stat budgets, drop rates, and value multipliers
- Loot tables keyed by source category × region tier with weighted random selection and conditional gates
- Equipment damage/armor scaling across 5 region tiers
- Gold economy with defined sources and sinks, balanced to ~600g net per region clear
- Shop system with faction standing and CHA barter price modifiers, trust-gated inventory
- Item identification with INT discount and Lore self-identify at camp
- Container interaction: loot UI, body looting (72h persist), locking (key/lockpick/force)

## Decision

**Data-driven loot and economy system. Items are authored as YAML per region. Loot tables use weighted random selection with conditional gates (thread, trust, faction, discovery). Shop prices combine faction standing modifier and CHA barter modifier with a hard floor at 25% of base value. Identification gates stat access for Rare+ items, with INT-based discounts and camp self-identify. The L&E system owns the Inventory component write path — other systems read or trigger actions through a defined action API.**

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    LOOT & ECONOMY SYSTEM                     │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ Item Registry│  │ Loot Tables  │  │  Shop System     │   │
│  │              │  │              │  │                  │   │
│  │ YAML per     │  │ Weighted     │  │ faction × CHA    │   │
│  │ region       │  │ rolls by     │  │ pricing          │   │
│  │ (read-only)  │  │ source+tier  │  │ trust-gated      │   │
│  └──────┬───────┘  └──────┬───────┘  │ inventory        │   │
│         │                 │           └────────┬─────────┘   │
│         │                 │                    │              │
│  ┌──────┴─────────────────┴────────────────────┴─────────┐   │
│  │                    ACTION API                          │   │
│  │                                                        │   │
│  │  Equip(item, slot)    Unequip(slot)                    │   │
│  │  Sell(item, merchant)  Buy(item, merchant)             │   │
│  │  Identify(item)        Drop(item)                      │   │
│  │  Loot(container)       TakeAll(container)              │   │
│  │  RequestLootDrop(enemy_type, region_tier) → []Item     │   │
│  └──────────────────────┬──────────────────────────────────┘   │
│                         │                                     │
│  ┌──────────────────────┴──────────────────────────────────┐   │
│  │              INVENTORY COMPONENT (EF)                    │   │
│  │  items[]  equipment_slots{}  gold:int                   │   │
│  │  L&E OWNS the write path — all mutations via Action API │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

EXTERNAL SYSTEMS:
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│Tactical      │   │Character     │   │Faction       │
│Combat Engine │   │Stats         │   │System        │
│              │   │              │   │              │
│request_loot  │   │STR (equip,   │   │standing      │
│_drop() on    │   │ carry, force)│   │(price mult,  │
│enemy death   │   │INT (identify)│   │ refusal)     │
│reads equipped│   │CHA (barter)  │   │              │
│stats for dmg │   │              │   │              │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       ▼                  ▼                  ▼
  L&E Action API    L&E pricing/        L&E pricing/
  (loot trigger)    equip formulas      shop access
```

### Key Interfaces

```go
// Item is the runtime representation of an item definition.
// Loaded from YAML at region load; immutable at runtime except for identified flag.
type Item struct {
    ID             string            // unique, e.g. "borderland_iron_sword"
    Name           string            // display name
    Type           ItemType          // Weapon, Armor, Consumable, Quest, CraftingMaterial, Accessory, Readable
    Subtype        ItemSubtype       // Sword, Axe, Dagger, Bow, Spear | Head, Chest, Hands, Legs
    Rarity         Rarity            // Common, Uncommon, Rare, Epic, Legendary
    Stats          map[string]int    // stat modifiers (damage, armor, accuracy, attr bonuses, HP)
    Weight         float32           // lbs — contributes to carry capacity
    Value          int               // base gold value
    StackSize      int               // max copies per inventory cell (1 for equipment)
    Description    string            // flavor text
    IconRef        string            // asset reference
    EquipmentSlot  string            // from shared equipment slot registry
    RequiredAttrs  map[string]int    // minimum attributes to equip
    RegionTier     int               // 1-5, minimum region tier for this item to appear
    Identified     bool              // stats revealed (Common/Uncommon auto-identify)
}

// ItemType enum
type ItemType int
const (
    TypeWeapon ItemType = iota
    TypeArmor
    TypeConsumable
    TypeQuest
    TypeCraftingMaterial
    TypeAccessory
    TypeReadable
)

// Rarity enum with stat budgets and value multipliers.
type Rarity int
const (
    RarityCommon    Rarity = 0 // 1-2 stat points, 1× value, 60% drop rate
    RarityUncommon  Rarity = 1 // 3-4 stat points, 2.5× value, 25% drop rate
    RarityRare      Rarity = 2 // 5-7 stat points, 5× value, 10% drop rate
    RarityEpic      Rarity = 3 // 8-10 stat points, 12× value, 4% drop rate
    RarityLegendary Rarity = 4 // 11-15 stat points, 25× value, 1% drop rate
)

// LootTable is a weighted random selection table for a source × region_tier combination.
type LootTable struct {
    Source      LootSource   // enemy_grunt, enemy_elite, enemy_boss, container_common, container_treasure, container_thread, merchant_inventory
    RegionTier  int          // 1-5
    Entries     []LootEntry
}

// LootEntry is one entry in a loot table.
type LootEntry struct {
    ItemID      string            // references ItemRegistry
    Weight      float64           // relative weight in table
    Conditions  LootCondition     // optional gates
    Guaranteed  bool              // always drops (before random rolls)
}

// LootCondition gates item eligibility.
type LootCondition struct {
    RequiresThread         string  // thread_id must be active
    RequiresTrustLevel     int     // player trust with merchant must be ≥ N
    RequiresFactionStanding int    // faction standing must be ≥ N
    RequiresDiscovery      string  // location_id must be discovered
}

// LootSource enum
type LootSource int
const (
    SourceEnemyGrunt LootSource = iota
    SourceEnemyElite
    SourceEnemyBoss
    SourceContainerCommon
    SourceContainerTreasure
    SourceContainerThread
    SourceMerchantInventory
)

// Inventory component (owned by L&E, stored in Entity Framework).
type InventoryComponent struct {
    Items          []InventorySlot
    EquipmentSlots map[string]Item  // slot_name → equipped item
    Gold           int
}

// InventorySlot is one cell in the inventory grid.
type InventorySlot struct {
    Item     Item
    Quantity int  // >1 for stackable items
}

// Container component (owned by EF, read by L&E for looting).
type ContainerComponent struct {
    Contents      []Item
    Capacity      int
    IsLocked      bool
    LockID        string  // key required to unlock (empty = no key lock)
    Indestructible bool   // thread-critical containers
    IsLooted      bool
}

// EconomySystem is the top-level interface.
type EconomySystem interface {
    // Action API
    EquipItem(entityID uint64, itemID string, slot string) error
    UnequipItem(entityID uint64, slot string) error
    SellItem(entityID uint64, itemID string, merchantID uint64) (int, error) // returns gold received
    BuyItem(entityID uint64, itemID string, merchantID uint64) (int, error)  // returns gold spent
    IdentifyItem(entityID uint64, itemID string, identifierID uint64) (int, error) // returns gold cost
    DropItem(entityID uint64, itemID string) error

    // Loot API
    LootContainer(entityID uint64, containerID uint64) ([]Item, int, error) // items + gold
    TakeAll(entityID uint64, containerID uint64) ([]Item, int, error)
    RequestLootDrop(enemyType string, regionTier int, isBoss bool) ([]Item, int) // TCE trigger

    // Query API (read-only)
    GetInventory(entityID uint64) InventoryComponent
    GetEquipmentBonus(entityID uint64, stat string) int // sum of equipped item stats for this stat
    GetGold(entityID uint64) int
    GetShopPrice(itemID string, merchantID uint64, playerID uint64) int // computed price
    GetIdentificationCost(itemID string, playerID uint64) int

    // Shop API
    OpenShop(merchantID uint64, playerID uint64) (ShopState, error)
    GetBuybackPool(merchantID uint64) []Item
}

// ShopState represents a merchant shop session.
type ShopState struct {
    MerchantID    uint64
    Inventory     []Item       // available for purchase
    FactionStanding int        // cached at shop open
    PlayerTrust   int          // cached at shop open
    PlayerCHA     int          // cached at shop open
}
```

### Item Authoring Format

Items are authored as YAML files per region: `assets/items/{region_slug}.yaml`

```yaml
# assets/items/borderland.yaml
items:
  - item_id: "rusted_iron_sword"
    name: "Rusted Iron Sword"
    type: Weapon
    subtype: Sword
    rarity: Common
    stats:
      damage: 3
    weight: 3.5
    value: 20
    stack_size: 1
    description: "A simple blade, pitted from years in the borderland damp."
    icon_ref: "items/weapons/rusted_iron_sword"
    equipment_slot: "main_hand"
    required_attrs:
      STR: 8
    region_tier: 1

  - item_id: "steel_broadsword"
    name: "Steel Broadsword"
    type: Weapon
    subtype: Sword
    rarity: Uncommon
    stats:
      damage: 7
      accuracy: 1
    weight: 4.0
    value: 125
    stack_size: 1
    description: "Well-made borderland steel. Reliable."
    icon_ref: "items/weapons/steel_broadsword"
    equipment_slot: "main_hand"
    required_attrs:
      STR: 10
    region_tier: 2
```

### Key Formulas

**Item base value**: `base_value = rarity_multiplier × region_tier × 20`

| Rarity | Multiplier | Tier 1 | Tier 3 | Tier 5 |
|--------|-----------|--------|--------|--------|
| Common | 1× | 20g | 60g | 100g |
| Uncommon | 2.5× | 50g | 150g | 250g |
| Rare | 5× | 100g | 300g | 500g |
| Epic | 12× | 240g | 720g | 1200g |
| Legendary | 25× | 500g | 1500g | 2500g |

**Shop purchase price**: `final_price = max(base_value × faction_multiplier × barter_modifier, base_value × 0.25)`

Where:
- `faction_multiplier = 1.0 − standing/400` (range 0.75 to 1.25)
- `barter_modifier = 1.0 − CHA_mod × 0.05` (range 0.80 to 1.20)
- Hard floor at 25% of base value

**Selling price**: `sell_price = base_value × 0.25` (faction/CHA do not affect selling)

**Identification cost**: `id_cost = max(25 × rarity_tier × region_tier × (1.0 − INT_mod × 0.1), 5)`

**Equipment damage scaling** (per region tier): `damage_range = [max(2, floor(tier × 3.5 − 2)), max(4, floor(tier × 3.5 + 2))]`

| Tier | Damage Range | Armor Range |
|------|-------------|-------------|
| 1 | 2–6 | 1–3 |
| 2 | 5–10 | 3–6 |
| 3 | 8–14 | 5–9 |
| 4 | 12–18 | 8–12 |
| 5 | 16–20 | 11–15 |

### Equipment Slot Registry (shared across systems)

8 slots, consumed by L&E, C&I UI, and TCE:

| Slot | Compatible Types |
|------|-----------------|
| `head` | Armor (Head) |
| `chest` | Armor (Chest) |
| `hands` | Armor (Hands) |
| `legs` | Armor (Legs) |
| `main_hand` | Weapon (Sword, Axe, Dagger, Spear) |
| `off_hand` | Weapon (Dagger), Armor (Shield) |
| `accessory_1` | Accessory |
| `accessory_2` | Accessory |

## Alternatives Considered

### Alternative 1: Hardcoded item data in Go structs
- **Description**: Item definitions as Go source code — `var IronSword = Item{...}`.
- **Pros**: Compile-time validation. No parsing overhead. Simple toolchain.
- **Cons**: Every item change requires recompilation. Designers cannot iterate without a Go toolchain. Adding items bloats the binary. Separation of content from engine is an explicit project requirement (ADR-0005 establishes this for KG).
- **Rejection Reason**: Violates the data-driven principle. Content must be separable from engine code.

### Alternative 2: Single item database (SQLite or JSON blob)
- **Description**: All items in one file (`items.db` or `items.json`), queried at runtime.
- **Pros**: Single source of truth. Cross-region item referencing is trivial. Can express complex queries (all swords with damage > 5 at tier 3).
- **Cons**: Monolithic — changing one region's items risks affecting others. Merge conflicts in version control on a single large file. No natural scoping to region load boundaries.
- **Rejection Reason**: Per-region YAML files align with the region-based development model. A designer working on region 3 should not risk breaking region 1's items. The file-per-region pattern is already established by KG (`.kg` files) and PWS (evolution rules).

### Alternative 3: Runtime procedural generation from templates
- **Description**: Items generated procedurally from rarity + region tier templates with random stat allocation within budgets.
- **Pros**: Infinite variety. No authoring burden. Every playthrough has different loot.
- **Cons**: No authored flavor text or lore. Stats feel random, not designed. Cannot guarantee specific items for thread rewards. Violates the anti-pillar "NOT procedurally generated" — the game's content is hand-authored.
- **Rejection Reason**: The game is built on hand-authored content. Items with history and provenance require authored design. Procedural loot would undermine "Earned Revelation" — finding a specific sword with a known history is more meaningful than a randomly generated +2 blade.

## Consequences

### Positive
- Fully data-driven: designers can add, tune, and rebalance items without touching Go code
- Per-region YAML files prevent cross-region contamination and reduce merge conflicts
- Equipment is the primary progression driver (60-70% of combat power) — loot feels impactful
- Shop prices integrate faction standing and character attributes — the economy is part of the world simulation, not a disconnected storefront
- Identification creates discovery moments for Rare+ items
- Container locking mechanics (key/lockpick/force) create meaningful choices
- L&E's Action API provides a single write path for inventory mutations — prevents data corruption from multiple writers

### Negative
- Per-region YAML requires tooling for cross-region item validation (ensuring no unintended duplicates, consistent tier progression)
- Weighted loot tables require statistical validation (AC-2: ±2σ tolerance across 1000 rolls) — non-trivial to test deterministically
- Identification adds friction to the item acquisition loop — some players may find it tedious
- Gold economy balance requires ongoing tuning as other systems (combat density, thread rewards, travel costs) evolve

### Risks
- **Gold inflation**: If combat density or thread rewards increase without corresponding sink adjustments, gold accumulates without purpose. Mitigation: all sink costs are configurable via tuning knobs; economy is re-balanced per milestone.
- **Equipment power creep**: If higher-tier items are too accessible, combat difficulty collapses. Mitigation: region tier gating on all items; `required_attrs` prevent low-level characters from wielding endgame weapons.
- **Loot table validation drift**: As items are added or rarity rates adjusted, statistical guarantees may break. Mitigation: AC-2 enforces statistical validation; loot table generation is deterministic/seeded for reproducible CI testing.
- **Merchant gold pool question unresolved**: The GDD's open question on whether merchants have limited gold is not answered by this ADR. Mitigation: start with unlimited merchant gold (simpler); if economy analysis shows inflation, add merchant gold pools as a configurable knob.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| loot-economy.md | TR-loot-001: Item data schema with 13+ fields | `Item` struct with all GDD-specified fields; per-region YAML authoring format |
| loot-economy.md | TR-loot-002: Weighted loot tables with conditional gates | `LootTable` + `LootEntry` with `LootCondition` gates (thread, trust, faction, discovery); deterministic/seeded RNG for statistical ACs |
| loot-economy.md | TR-loot-003: Merchant/shop model with faction/CHA pricing, trust-gated stock, buyback | `ShopState`; `GetShopPrice()` with faction × CHA formula; `GetBuybackPool()`; restock timers 72-240 game hours |
| loot-economy.md | TR-loot-004: Identification state machine with INT discount and Lore self-identify | `IdentifyItem()` with cost formula; INT discount; camp self-identify (INT≥14 Rare, ≥16 Epic); Legendary always merchant/thread |
| loot-economy.md | TR-loot-005: Container interaction with EF Container component | `LootContainer()` / `TakeAll()` reading `ContainerComponent`; body looting (72h persist); locking (key/lockpick DEX/force STR 50% destroy) |
| loot-economy.md | TR-loot-006: Action API — L&E owns Inventory write path | `EconomySystem` interface with `EquipItem`, `UnequipItem`, `SellItem`, `BuyItem`, `IdentifyItem`, `DropItem`; `RequestLootDrop()` for TCE |
| loot-economy.md | TR-loot-007: Persistence — looted-container state, restock timers, buyback, shop stock | Inventory/Container components serialized via EF (ADR-0002); loot-table state persisted per region |
| loot-economy.md | TR-loot-008: Cross-system reads — faction.standing, trust_level, STR, INT, CHA | `GetShopPrice()` reads faction standing and CHA; `IdentifyItem()` reads INT; `EquipItem()` reads STR; trust-gated inventory reads trust_level |

## Performance Implications
- **CPU**: Loot roll (weighted random) < 0.1ms per roll (3-5 rolls per drop typical). Shop price calculation < 0.01ms (arithmetic). Identification cost < 0.01ms. Item stat summation for equipment bonus < 0.01ms (cached on equip). Total economy CPU budget: < 0.5ms/frame (only active during loot/shop interactions, not per-frame).
- **Memory**: Item registry: ~2KB per item definition (YAML parsed to Go structs). ~200 items across 5 regions = ~400KB. Loot tables: ~1KB per source × tier combination (~35 tables = ~35KB). Container state: ~200 bytes per container. Total economy memory: < 1MB.
- **Load Time**: Per-region YAML parsing: ~50ms for ~40 items. Loot table construction: ~10ms per region. Total economy load < 100ms per region.
- **Network**: None — single-player.

## Migration Plan
1. Implement `Item` struct and per-region YAML parser
2. Implement `InventoryComponent` and `ContainerComponent` in Entity Framework (ADR-0001)
3. Implement `LootTable` with weighted random selection and conditional gates
4. Implement `EconomySystem` Action API (Equip/Unequip/Sell/Buy/Identify/Drop)
5. Implement `RequestLootDrop()` integration with TCE (consumes ADR-0009 loot trigger)
6. Implement `GetShopPrice()` with faction standing and CHA barter modifiers
7. Implement identification with INT discount and Lore self-identify
8. Implement container looting with locking mechanics
9. Implement save/load persistence (ADR-0002 integration)
10. Item data authoring: create `assets/items/borderland.yaml` with ~40 items for region 1

## Validation Criteria
- Item YAML parses correctly and produces valid `Item` structs for all fields
- Loot table with 1000 rolls produces rarity distribution within ±2σ of expected rates
- Equipment bonus from 3 equipped items sums correctly (e.g., +2 STR ring + +1 STR boots + base 10 = 13)
- Shop price for 200g item at faction +100 (allied) is 150g (±1g)
- Identification cost for Rare item at tier 3 with INT 16 (+3) is 210g
- Container with 3 items shows all 3 in loot UI; Take All transfers all to inventory
- Locked container with correct key opens without item loss
- Quest items cannot be sold or dropped (action rejected with error)
- Inventory state survives save/load round-trip (all items, equipment slots, gold)

## Related Decisions
- ADR-0001: Entity Framework — Inventory and Container components
- ADR-0002: Save/Load — Inventory/Container serialization
- ADR-0006: Character Stats — STR (equip/carry), INT (identify), CHA (barter)
- ADR-0009: Tactical Combat Engine — loot drop trigger, equipped stat reads for damage
- ADR-0011: Event Bus — loot events (container opened, item identified, item equipped)
- ADR-0012: UI Framework — Character & Inventory UI reads item data; shop UI

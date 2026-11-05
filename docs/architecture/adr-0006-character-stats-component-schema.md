# ADR-0006: Character Stats & Component Schema

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Character Stats GDD |
| **Post-Cutoff APIs Used** | None — pure game logic |
| **Verification Required** | Stats component storage in Entity Framework |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — Stats component on entities) |
| **Enables** | ADR-0009 (Combat Engine — reads derived stats), Party System |
| **Blocks** | Combat, Progression, Character UI |
| **Ordering Note** | Must be Accepted before combat and character systems are implemented |

## Context

The Marrow of Empire uses a six-attribute CRPG stat system (STR/DEX/CON/INT/WIS/CHA, range 3-18) with derived stats (Health, AP, Initiative, Evasion, etc.) and level progression (1-20, XP from threads + combat). Point-buy character creation with 30 points, max 16 starting attribute. No class system — attributes ARE the class.

## Decision

**Six-attribute model with BG1-compatible modifier table.** `Modifier = floor((Attribute - 10) / 2)`, clamped to [-4, +4]. Derived stats calculated from attributes + modifiers. XP from threads and combat, linear level thresholds: `XP_to_next = Level × 300`.

### Key Formulas
- **Modifier**: `floor((A - 10) / 2)`, range [-4, +4]
- **Health**: `20 + (CON × 2) + (Level × CON_mod)`, minimum +1 HP/level
- **AP**: `4 + DEX_mod`, minimum 2, maximum 8
- **Initiative**: `DEX_mod + WIS_mod`
- **Evasion**: `10 + DEX_mod`
- **XP to next level**: `Level × 300`
- **Total XP for level N**: `150 × N × (N - 1)`

### Point-Buy
30 points. All attributes start at 8. Cost to raise: `N - 8`. Maximum starting: 16.

### Stats Component Schema
```go
type StatsComponent struct {
    Attributes [6]int  // [STR, DEX, CON, INT, WIS, CHA], 3-18
    Level      int     // 1-20
    XP         int     // total accumulated
}

func (s StatsComponent) Modifier(attr int) int {
    return clamp((s.Attributes[attr]-10)/2, -4, 4)
}
```

## GDD Requirements Addressed
Character Stats GDD: CR-1 through CR-5 (attributes, modifiers, derived stats, creation, progression), all formulas.

## Performance Implications
Attribute modifier: <0.01ms (integer arithmetic). Derived stat recalculation on level-up: <0.1ms. XP threshold check: <0.01ms.

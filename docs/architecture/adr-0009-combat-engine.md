# ADR-0009: Combat Engine

## Status
Accepted

## Date
2026-08-09

## Engine Compatibility

| Field | Value |
|-------|-------|
| **Engine** | Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) |
| **Domain** | Core / Physics |
| **Knowledge Risk** | HIGH — custom engine |
| **References Consulted** | Tactical Combat Engine GDD, Character Stats GDD, Camera System GDD |
| **Post-Cutoff APIs Used** | None specific — pure game logic |
| **Verification Required** | Grid movement and spatial queries against Entity Framework spatial index |

## ADR Dependencies

| Field | Value |
|-------|-------|
| **Depends On** | ADR-0001 (Entity Architecture — spatial queries for targeting), ADR-0006 (Character Stats — AP, initiative, damage formulas) |
| **Enables** | Combat UI, Party System (combat participation) |
| **Blocks** | Tactical Combat Engine implementation |
| **Ordering Note** | Must be Accepted before combat system development |

## Context

### Problem Statement
The Marrow of Empire uses turn-based tactical combat on an isometric grid. Characters spend Action Points to move, attack, and use abilities. Must choose grid type, formalize AP economy, define attack resolution (d20-based), and handle death/dying.

### Constraints
- Must work on the fixed isometric camera angle (30° pitch, 45° yaw)
- Must support the Camera System's three zoom registers (4u/10u/20u)
- Every combat encounter has narrative weight (Pillar 3 — no filler fights)
- Combat pauses the game clock
- Party size: 1 player + up to 5 companions = max 6 friendly combatants

### Requirements
- Grid-based movement and positioning
- AP system with clear action costs
- d20-based attack resolution against evasion
- Flanking, cover, and elevation bonuses
- Death/dying with 3-round stabilization window (player/party only)

## Decision

**Square grid (1 unit = 1 meter per tile), d20 attack resolution, 4+ DEX_mod AP per turn.** Square tiles align naturally with the isometric 45° camera and the fixed angle means diagonal movement is visually clear. The d20 system provides familiar CRPG combat math with predictable probability curves.

### Grid Decision

**Square grid** chosen over hex for isometric alignment. At 45° yaw, square tiles render as clean diamonds — visually distinct, easy to read at all zoom levels. Hex grids create diagonal ambiguity at isometric angles.

| Parameter | Value |
|-----------|-------|
| Grid type | Square tiles, 1 unit = 1 meter |
| Diagonal movement | Allowed, 1 AP per diagonal (same as orthogonal) |
| Flanking | +2 accuracy when attacking from opposite side of an adjacent ally |
| Cover | Half cover: +2 evasion. Full cover: blocks line of sight. |
| Elevation | Higher ground: +1 accuracy, +1 range. Climbing 1 level: 2 AP. |
| Opportunity | Moving out of enemy melee range triggers free attack (once/turn/enemy) |

### AP Economy

| Action | AP Cost |
|--------|---------|
| Move 1 tile | 1 |
| Standard attack | 2 |
| Ability | 1–3 (per ability definition) |
| Interact (door, lever, item) | 1 |
| Stabilize dying ally | 2 (requires adjacency) |
| Defend (+2 evasion until next turn) | Remaining AP (min 1) |
| Wait (move to end of initiative) | Remaining AP (min 1) |

AP per turn: `4 + DEX_modifier` (minimum 2, maximum 8 at DEX 18). Unused AP does not carry over.

### Attack Resolution (d20)

```
Hit roll: d20 + accuracy >= target evasion → hit
Natural 20: critical hit (double damage)
Natural 1: automatic miss
```

Accuracy = base weapon accuracy + STR_mod (melee) or DEX_mod (ranged) + flanking + elevation + ability bonuses.
Evasion = 10 + DEX_mod + cover + ability bonuses.

Damage = weapon_base + attribute_mod + ability_bonus − target_armor (minimum 1).

### Initiative

Initiative = `DEX_mod + WIS_mod`. Ties broken by DEX, then entity ID. Rolled once at combat start. Turn order is fixed for the encounter.

### Death and Dying

Player/party characters at 0 HP enter Dying state. They have 3 rounds to be stabilized by an adjacent ally (2 AP). If not stabilized within 3 rounds, permanent death. Enemies at 0 HP are immediately dead.

## Alternatives Considered

### Alternative 1: Square grid (CHOSEN)
- **Pros**: Natural isometric alignment. Diagonal movement visually clear. Familiar CRPG convention (BG1-2, DOS2). Easier spatial index integration.
- **Cons**: Diagonal movement cost = orthogonal (1 AP) can feel generous. No facing direction — flanking must be calculated from ally positions.
- **Rejection Reason**: N/A.

### Alternative 2: Hex grid
- **Pros**: More natural movement in all 6 directions. No diagonal ambiguity. More tactical depth in positioning.
- **Cons**: Misaligned with isometric 45° angle — hexes appear distorted at this camera angle. Harder to author environments for. Spatial index integration more complex.
- **Rejection Reason**: Isometric 45° camera makes square tiles the natural fit. The visual clarity advantage of square tiles at this angle outweighs hex's tactical depth.

### Alternative 3: Range-based (no grid)
- **Description**: Free movement measured in world units, no discrete grid.
- **Pros**: More natural movement. No grid artifacts.
- **Cons**: Ambiguous positioning — "adjacent" and "flanking" become fuzzy. Harder to communicate valid moves to player. AP-per-meter calculations less intuitive.
- **Rejection Reason**: Grid provides clarity for tactical decision-making. "Within 2 tiles" is unambiguous; "within 2 meters" invites edge cases.

## Consequences

### Positive
- Familiar d20 CRPG math — accessible to the target audience
- AP system rewards tactical thinking (save AP for positioning vs. spend on attacks)
- Square grid aligns with isometric camera for visual clarity
- Death/dying adds stakes without being punitive (3 rounds to save)

### Negative
- d20 swinginess: natural 1/20 creates 5% auto-miss/auto-crit — can feel random
- Square grid diagonal movement at 1 AP reduces tactical cost of positioning
- Fixed initiative order means no "delay turn" strategy beyond the Wait action

### Risks
- **AP inflation at high DEX**: DEX 18 characters have 8 AP vs. 2 AP for DEX 6 — 4× action economy gap. Mitigation: minimum 2 AP ensures even low-DEX characters can act; ability costs scale to consume AP.
- **d20 crit chains**: Multiple natural 20s in a row can swing fights dramatically. Mitigation: crits are double damage only (no additional effects); no "confirm crit" roll.

## GDD Requirements Addressed

| GDD System | Requirement | How This ADR Addresses It |
|------------|-------------|--------------------------|
| Tactical Combat Engine | CR-1: Combat Initiation — clock pause, camera transition, initiative roll | Combat start freezes game clock; camera enters Combat state |
| Tactical Combat Engine | CR-2: Action Points — 4 + DEX_mod, move 1 AP, attack 2 AP | AP economy table with costs per action type |
| Tactical Combat Engine | CR-3: Movement and Positioning — grid, flanking, cover, elevation, opportunity | Square grid with defined bonuses per positioning type |
| Tactical Combat Engine | CR-4: Attack Resolution — d20, accuracy vs. evasion, crit on 20 | d20 system with damage formula and crit/auto-miss rules |
| Tactical Combat Engine | CR-5: Death and Dying — 3-round stabilization, permanent death | Dying state with 3-round window and stabilize action (2 AP) |

## Performance Implications
- **CPU**: Attack resolution: <1ms (simple arithmetic). Turn order sort: <0.1ms for <20 combatants. AI decision: 10-50ms (deferred to AI system ADR).
- **Memory**: Combat state: ~2KB (turn order, AP values, status effects per combatant).
- **Load Time**: N/A — no additional load.
- **Network**: N/A.

## Migration Plan
No migration — first implementation. Grid and combat math are pure game logic with no existing code to replace.

## Validation Criteria
- DEX 16 character has 7 AP (4+3) at combat start
- Flanking attack (+2 accuracy) hits more often than frontal attack in 100-roll statistical test
- Dying character not stabilized within 3 rounds → permanent death
- Natural 20 → double damage; natural 1 → miss regardless of accuracy

## Related Decisions
- ADR-0001: Entity Architecture — spatial queries for targeting and movement validation
- ADR-0006: Character Stats — AP, initiative, and damage derived from attributes

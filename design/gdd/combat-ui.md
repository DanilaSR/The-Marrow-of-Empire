# Combat UI

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 3 (Tactical Depth — readable combat spaces)

## Summary

The Combat UI displays all tactical information during combat: turn order
(initiative bar), Action Points (current/max), health bars over combatants,
ability menu, targeting reticle, and damage numbers. Amber and oxblood
palette per the art bible. Clarity is the non-negotiable constraint —
"darkness never eats a decision."

> **Quick reference** — Layer: `Presentation` · Priority: `MVP` · Key deps: `Tactical Combat Engine`

## Overview

The Combat UI overlays tactical information on the isometric battlefield
during combat. It appears when combat starts and hides when combat ends.
All elements are designed for at-a-glance readability — the player should
never lose a fight because they couldn't see the tactical information.

## Detailed Design

**Layout:**
- **Top**: Turn order bar — horizontal row of combatant portraits in initiative order. Current turn highlighted with amber border. Upcoming turns dimmed. Hovering shows name + HP.
- **Bottom-left**: Selected character's AP bar (amber fill, depletes as AP is spent). Current AP / Max AP displayed numerically.
- **Bottom-center**: Ability menu — grid of available actions (Move, Attack, Abilities, Items, Defend, Wait). Greyed out if insufficient AP. Hotkeys 1-6.
- **Bottom-right**: End Turn button (amber, prominent).
- **World overlay**: Health bars above each combatant (amber fill, depletes to oxblood). Damage numbers float up on hit (oxblood) / miss text (grey). Critical hits: larger text + amber flash. Flanking indicator (blue arcs behind target when flanking bonus applies). Cover indicator (shield icon on combatants in cover).
- **Targeting**: When an attack/ability is selected, valid targets highlighted. Invalid targets dimmed. Range indicator (line from attacker to cursor). Hit chance displayed on hover (e.g., "75%").

**Combat start**: Initiative bar slides in from top (200ms). AP bar slides in from left. Ability grid slides in from bottom.

**Combat end**: All UI elements slide out (200ms). Victory/defeat screen overlay: XP summary for victory, "Game Over — Load Save" for defeat.

## Visual/Audio Requirements

| Event | Visual | Audio |
|-------|--------|-------|
| Turn change | Current portrait highlight slides to next; AP bar refills with amber pulse | Soft tick |
| AP spent | AP bar depletes in real-time; number counts down | Subtle click per AP |
| Attack selected | Targets highlight; invalid targets dim; range line appears | Soft select sound |
| Damage dealt | Damage number floats up (oxblood); health bar depletes | Impact (type-specific) |
| Critical hit | Larger number + amber flash + screen shake (2 units) | Heavy impact + chime |
| Enemy death | Health bar shatters; portrait greys in initiative bar | Death sound |
| Victory | XP summary screen overlay; party portraits with +XP | Fanfare |

## Acceptance Criteria

- **GIVEN** combat with 4 player characters and 3 enemies, **WHEN** initiative is rolled, **THEN** the turn order bar shows all 7 portraits in correct initiative order with the first combatant highlighted.
- **GIVEN** a character with 7 AP, **WHEN** they spend 2 AP on an attack, **THEN** the AP bar depletes from 7 to 5 with the number updating in real-time.
- **GIVEN** a flanking attack, **WHEN** targeting, **THEN** the flanking indicator (blue arcs) appears behind the target and the hit chance includes the +2 flanking bonus.

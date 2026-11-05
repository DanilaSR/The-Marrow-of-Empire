# Trust & Relationship

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: Pillar 1 (Every Voice Holds a Piece — trust gates the deepest pieces)

## Summary

Trust & Relationship manages the trust level between the player and each NPC
(0–3 scale), defines how trust is earned and lost, and persists relationship
state via the Persistent World State system. Trust gates access to deeper NPC
knowledge — it's the mechanical backbone of "trust is earned, not clicked."

> **Quick reference** — Layer: `Feature` · Priority: `MVP` · Key deps: `NPC Knowledge Graph`, `Persistent World State (VS — interface assumed)`

## Overview

Trust & Relationship is the system that makes NPC relationships feel real.
Every NPC starts at trust level 0 — surface knowledge only. As the player
demonstrates genuine investigation, helps the NPC, or is vouched for by
trusted allies, trust increases, unlocking deeper facts and secrets. Trust
can also be lost — betray an NPC's confidence, lie to them, or act against
their interests, and the door closes.

The system manages:

- **Trust levels**: A 0–3 integer per NPC, persisted across sessions. 0 =
  stranger (surface facts only). 1 = acquaintance (basic trust). 2 = trusted
  (significant knowledge). 3 = confidant (secrets revealed).
- **Trust events**: Actions that increase or decrease trust. These are
  triggered by gameplay events — resolving threads, completing favors,
  demonstrating knowledge, being vouched for, or betraying confidence.
- **Relationship metadata**: Beyond the number, the system tracks WHY the
  NPC trusts the player (which events earned it) and relationship flags
  (e.g., `owes_favor`, `was_betrayed`, `vouched_by:[entity_id]`). This
  metadata feeds NPC dialogue — an NPC who trusts you because you saved
  their child speaks differently than one who trusts you because the
  Captain vouched for you.
- **Persistence**: Trust levels and relationship metadata are stored in
  the NPC's Persistent component (Entity Framework), serialized by the
  Save/Load System.

Without Trust & Relationship: every NPC is equally open to every player.
There's no progression in relationships. Deep knowledge is either free
(undermining investigation) or permanently locked (frustrating players).
Trust is a number in the knowledge graph with no system to change it.

## Player Fantasy

**Trust is earned, not clicked.** This is the core fantasy. The player doesn't
select "Persuade" from a menu and watch a skill check resolve. They earn an
NPC's trust by DOING things — following investigation threads that matter to
that NPC, helping them with problems, demonstrating genuine knowledge. When
the herbalist goes from guarded to open, it's because the player showed her
they were really investigating Aldric's death, not just gossiping.

**Relationships feel individual.** The blacksmith trusts the player because
they appreciated her craft and asked intelligent questions about metallurgy.
The hermit trusts the player because they came to him already knowing about
the Ley Network — demonstrating they'd done their research. Each NPC's trust
feels earned for a specific reason, and the reason colors future interactions.

**Betrayal has weight.** If the player shares an NPC's secret with the wrong
person, or acts against their interests, trust drops. The NPC remembers.
Rebuilding broken trust is harder than earning it the first time. This makes
relationship decisions meaningful — who you trust, what you share, and who
you protect have consequences.

## Detailed Design

### Core Rules

**CR-1: Trust Levels**

| Level | Name | Unlocks | Visual Indicator |
|-------|------|---------|-----------------|
| 0 | Stranger | Facts with `trust_required: 0` | Neutral portrait, no indicator |
| 1 | Acquaintance | Facts with `trust_required: 0–1` | Subtle warm tint on portrait border |
| 2 | Trusted | Facts with `trust_required: 0–2` | Amber glow on portrait border; NPC greets player by name |
| 3 | Confidant | All facts including SECRETS | Full amber glow; NPC may proactively share new information |

**CR-2: Earning Trust**

Trust increases through specific, trackable events:

| Trust Event | Trust Gain | Condition |
|-------------|-----------|-----------|
| `demonstrated_investigation` | +1 to level 1 | Player asks NPC about a topic in their domain using knowledge gained from another NPC (proves they're investigating, not randomly asking) |
| `small_favor_completed` | +1 to level 1 | Player completes a minor task for the NPC (fetch item, deliver message, share relevant information) |
| `thread_resolved_involving_npc` | +1 (one-time) | Player resolves an investigation thread that directly involves or affects this NPC |
| `significant_help` | +1 to level 2 | Player helps the NPC in a meaningful way (saves them from danger, resolves a major problem, protects someone they care about) |
| `vouched_for` | +1 to level 2 | A character the NPC already trusts at level 2+ vouches for the player. The vouching character's trust must be genuine |
| `secret_discovered_and_kept` | +1 to level 3 | Player discovers the NPC's secret and does NOT share it with anyone the NPC wouldn't want to know |

Trust increases are sequential — you cannot skip from level 0 to level 2
without first reaching level 1. A `significant_help` event at level 0
raises trust to level 1, not level 2.

**CR-3: Losing Trust**

Trust can decrease. Losses are sharper than gains:

| Trust Event | Trust Loss | Condition |
|-------------|-----------|-----------|
| `secret_betrayed` | −2 levels (min 0) | Player shares an NPC's secret with someone the NPC explicitly wouldn't trust with it |
| `caught_lying` | −1 level | NPC catches the player in a direct lie (contradicts a fact the NPC knows) |
| `hostile_action` | −2 levels (min 0) | Player attacks, steals from, or deliberately harms the NPC or someone they care about |
| `broken_promise` | −1 level | Player promised to help and didn't (timeout on a favor) |
| `association_penalty` | −1 level | Player is seen working closely with an NPC's known enemy (faction-based) |

Lost trust is harder to regain. After a trust decrease, the NPC's trust
"freezes" for a period — no trust gains for 3 in-game days (72 game hours,
~7 real hours). After the freeze, trust can be earned again starting from
the reduced level.

**CR-4: Relationship Metadata**

Beyond the numeric trust level, the system stores relationship flags:

| Flag | Type | Example |
|------|------|---------|
| `trust_events[]` | array of {event_type, timestamp, thread_ref} | History of what earned/lost trust |
| `owes_favor` | bool | NPC owes the player a favor (can be called in) |
| `player_owes_favor` | bool | Player owes the NPC a favor (NPC may ask) |
| `vouched_by[]` | []entity_id | Who vouched for the player to this NPC |
| `vouched_for[]` | []entity_id | Who this NPC vouched for the player to |
| `knows_secret[]` | []fact_id | Which of this NPC's secrets the player has discovered |
| `betrayal_history[]` | array of {fact_id, timestamp} | Which secrets the player betrayed and when |

This metadata feeds the AI Dialogue Engine: an NPC who trusts the player
because `vouched_by: [orin_entity_id]` will reference Orin in conversation.
An NPC with `betrayal_history` containing a secret will be guarded and may
warn other NPCs about the player.

**CR-5: Initial Trust**

Most NPCs start at trust level 0. Exceptions:

- **Story-critical NPCs**: May start at trust level 1 if the narrative
  requires them to share basic information immediately (the first NPC
  the player meets, a patron who hired the player).
- **Antagonists**: Start at trust level 0 and cannot rise above level 2
  (they will never fully trust the player, and the player may not want
  their trust).
- **Party members**: Start at trust level 1 (they've already agreed to
  travel with the player; deeper trust must still be earned).
- **Faction enemies**: Start at trust level 0 and cannot rise above level
  1 without a major faction allegiance shift.

**CR-6: Trust Propagation**

Trust can spread through social networks:

- **Vouching** (CR-2): A trusted NPC vouches for the player to another NPC.
  The vouching NPC must be at trust level 2+ with the player. The target
  NPC gains +1 trust (to level 1 if at 0). Vouching is an active action —
  the player can ask a trusted NPC "Can you introduce me to [NPC]?" or it
  happens automatically when the trusted NPC's `leads_to` hints reference
  that NPC.
- **Reputation bleed**: If the player is at trust level 2+ with multiple
  NPCs in the same faction/town, other NPCs in that group start at trust
  level 1 instead of 0 (reputation precedes the player).
- **Betrayal bleed**: If the player betrays an NPC's secret and that NPC
  has `vouched_for` other NPCs, those NPCs also lose trust (−1 level)
  because the player was vouched for and proved untrustworthy.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| NPC Knowledge Graph | Trust & Relationship writes `trust_level` | Updates trust level on NPC's Knowledge component, gating fact access |
| Persistent World State | Trust & Relationship reads/writes persistence | Trust levels and relationship metadata stored in NPC's Persistent component. **Assumption**: Persistent World State provides `get_flags(entity_id) → map[string]interface{}` and `set_flags(entity_id, flags)`. These are provisional until Persistent World State GDD is written. |
| AI Dialogue Engine | AI Dialogue reads trust level and metadata | Trust level gates fact inclusion; relationship metadata colors NPC dialogue tone |
| Thread Investigation | Thread Investigation triggers trust events | Resolving threads that involve NPCs triggers trust gains |
| Entity Framework | Trust & Relationship reads/writes Persistent component | Relationship data stored as persistent flags on NPC entities |

## Formulas

No mathematical formulas. Trust level is discrete (0–3). Trust gain/loss
is event-driven, not computed from a formula.

## Edge Cases

- **If the player earns multiple trust events simultaneously** (e.g., resolves a thread that also counts as significant help): Apply the highest-value event only. Trust events don't stack in the same interaction.
- **If trust would increase above 3**: Clamp at 3. The excess event is recorded in `trust_events[]` but no further level change occurs.
- **If trust would decrease below 0**: Clamp at 0. NPC cannot go below stranger status.
- **If the player is vouched for by an NPC whose trust later drops below 2**: The vouching remains valid — it was genuine at the time. However, if the vouching NPC's trust dropped due to betrayal, the vouched-for NPC may question the player about it.
- **If an NPC dies while at trust level 2+**: The trust is memorialized — other NPCs who knew the deceased NPC may reference the relationship. No trust events can fire for the dead NPC.

## Dependencies

| System | Direction | Nature |
|--------|-----------|--------|
| NPC Knowledge Graph | Trust & Relationship writes trust_level | Updates trust on Knowledge component. Hard. |
| Persistent World State | Trust & Relationship writes persistent flags | Stores relationship metadata and trust level for session persistence. **Provisional** — Persistent World State is VS tier, not yet designed. MVP can store trust in NPC Knowledge component directly without full persistence. |
| AI Dialogue Engine | AI Dialogue reads trust level | Trust gates fact inclusion in prompts. Hard. |
| Thread Investigation | Thread Investigation triggers trust events | Thread resolution fires trust gain events. Soft — trust functions without threads. |
| Entity Framework | Trust & Relationship reads/writes Persistent component | Relationship flags stored on NPC entities. Hard. |

## Tuning Knobs

| Parameter | Default | Range | Effect |
|-----------|---------|-------|--------|
| `max_trust_level` | 3 | 2–5 | More gradation, more content gated behind trust |
| `trust_freeze_duration_hours` | 72 | 24–168 | Longer freeze = betrayal hurts more, trust is harder to rebuild |
| `reputation_bleed_threshold` | 2 | 1–3 | Higher = more NPCs must trust you before reputation spreads |
| `vouch_required_trust` | 2 | 1–3 | Higher = vouching is harder to access, more valuable |

## Acceptance Criteria

- **GIVEN** an NPC at trust level 0, **WHEN** the player asks about a topic in the NPC's domain using knowledge from another NPC (demonstrated_investigation), **THEN** trust increases to level 1.

- **GIVEN** an NPC at trust level 2, **WHEN** the player betrays the NPC's secret to someone they wouldn't trust, **THEN** trust drops to level 0 and the trust freeze activates for 72 game hours.

- **GIVEN** an NPC at trust level 2+, **WHEN** the player asks them to vouch for them to another NPC, **THEN** the target NPC gains +1 trust (to level 1 if at 0).

- **GIVEN** multiple trust events in one interaction, **WHEN** events would push trust from 0 to 2, **THEN** only the highest-value event applies — trust reaches level 1, not level 2.

- **GIVEN** an NPC with trust level 0 and `betrayal_history` non-empty, **WHEN** the player first meets them, **THEN** the AI Dialogue Engine receives the betrayal metadata and the NPC is guarded in conversation.

- **GIVEN** the player at trust 2+ with 3 NPCs in the same faction, **WHEN** the player meets a new NPC in that faction, **THEN** the new NPC starts at trust level 1 (reputation bleed).

## Open Questions

| Question | Owner | Deadline |
|----------|-------|----------|
| Should there be a visible trust indicator (portrait border), or should trust changes be communicated purely through NPC behavior/dialogue? | UX Designer | Before Dialogue UI GDD |
| Can the player spend trust as a resource (e.g., "cash in" a favor to get an NPC to do something risky)? | Game Designer | Before vertical slice |
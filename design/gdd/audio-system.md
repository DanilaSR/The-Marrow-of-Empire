# Audio System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Implements Pillar**: All four — audio carries mood (Pillar 1), memory (Pillar 2), tension (Pillar 3), and revelation (Pillar 4)

## Summary

The Audio System manages all sound in The Marrow of Empire: ambient nature
soundscapes (birdsong, wind, water), adaptive music (exploration vs. combat),
SFX (combat impacts, UI feedback, footstep types), and the spatial audio
listener tied to the camera position. Ambient audio is driven by the Living
Environment's time-of-day and light state. Combat audio is triggered by the
Tactical Combat Engine.

> **Quick reference** — Layer: `Presentation` · Priority: `VS` · Key deps: `Living Environment`, `Tactical Combat Engine`

## Overview

The Audio System creates the sonic identity of the borderland. It is the
reason the forest feels alive, the ruins feel abandoned, and combat feels
dangerous. It operates in layers:

- **Ambient layer**: Continuous environmental audio driven by time of day
  and region. Dawn birdsong, day wind, dusk crickets, night owls and
  distant howls. Region-specific ambience (river sounds near water, wind
  through ruins in empire spaces). Crossfades between states.
- **Music layer**: Adaptive music system. Exploration music: calm,
  melancholic, warm (amber palette in sound). Combat music: tense,
  driving, percussive. Transitions on combat start/end. Revelation music:
  reserved for State 4 moments — sparse, luminous, the audio equivalent
  of the two-light collision.
- **SFX layer**: Combat impacts (weapon type-specific), UI feedback
  (parchment unfurl, quill scratch, soft clicks), footstep types (stone,
  wood, grass, water), environmental (door creak, lantern light, fire
  crackle).
- **Voice layer** (deferred): NPC dialogue is text-only at MVP/VS. Voice
  acting or TTS is a future consideration.

**Spatial audio**: The camera position is the audio listener. Sounds are
spatialized in the isometric view — closer entities are louder, off-screen
sounds provide directional cues.

## Acceptance Criteria

- **GIVEN** Dawn light state, **WHEN** the time transitions to Day, **THEN** birdsong crossfades to day ambience over 5 seconds.
- **GIVEN** combat start, **WHEN** the encounter begins, **THEN** music transitions from exploration to combat over 1 second.
- **GIVEN** a critical hit, **WHEN** damage is dealt, **THEN** heavy impact + rising chime plays.

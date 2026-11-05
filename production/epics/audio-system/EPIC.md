# Epic: Audio System

> **Layer**: Presentation | **GDD**: audio-system.md | **Status**: Ready | **Stories**: 4 stories
> **Governing ADRs**: — (no specific audio ADR; pattern follows engine integration) | **Engine Risk**: HIGH | **Tier**: VS

## Overview
Implement game audio: ambient nature soundscapes driven by Living Environment time-of-day and light state, adaptive music (exploration calm/melancholic → combat tense/percussive → revelation sparse/luminous), SFX (combat impacts, UI feedback, footstep types), and spatial audio with camera as listener. VS tier.

## GDD Requirements
TR-ui-* (partial — audio is engine integration, not UI)

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Audio Engine Foundation & Spatial Listener | Integration | Complete | N/A — engine integration |
| 002 | Ambient Soundscapes | Integration | Ready | ADR-0007 |
| 003 | Adaptive Music System | Integration | Ready | ADR-0009, ADR-0011 |
| 004 | Combat & UI SFX | Integration | Ready | ADR-0009, ADR-0012 |

## Next Step
Run `/story-readiness production/epics/audio-system/story-001-audio-engine-foundation.md`

# Epic: Camera System

> **Layer**: Foundation
> **GDD**: design/gdd/camera-system.md
> **Architecture Module**: Camera System
> **Status**: Ready
> **Stories**: 4 stories

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | Isometric Projection Setup | Logic | Ready | ADR-0003 |
| 002 | Zoom System | Logic | Ready | ADR-0003 |
| 003 | Camera Follow & Framing | Integration | Ready | ADR-0003 |
| 004 | Occlusion & Boundaries | Integration | Ready | ADR-0003 |

## Overview

Implement the fixed isometric orthographic camera: 30° pitch, 45° yaw, three
discrete zoom levels (Close 4u / Default 10u / Far 20u), occlusion handling
(structures fade, roofs remove), aspect ratio strategy (16:9 ref, 21:9
supported), camera shake for combat feedback, and boundary enforcement
(no void space). Kaiju `StandardCameraOrthographic` with world-space extents.

## Governing ADRs

| ADR | Decision Summary | Engine Risk |
|-----|-----------------|-------------|
| ADR-0003: Isometric Camera | Orthographic 30°/45°, zoom registers, occlusion, aspect ratio | HIGH |

## GDD Requirements

| TR-ID | Requirement | ADR Coverage |
|-------|-------------|--------------|
| TR-cam-001 | Orthographic projection 30°/45° | ADR-0003 ✅ |
| TR-cam-002 | Three zoom registers (4u/10u/20u) | ADR-0003 ✅ |
| TR-cam-003 | Occlusion handling | ADR-0003 ✅ |
| TR-cam-004 | Aspect ratio strategy | ADR-0003 ✅ |

## Definition of Done

- All stories implemented, reviewed, and closed
- Camera renders isometric view at all three zoom levels
- Occlusion: structures fade, interiors render without roofs
- Camera never shows void space
- Zoom transitions smooth (300ms lerp)

## Next Step

Run `/create-stories camera-system`

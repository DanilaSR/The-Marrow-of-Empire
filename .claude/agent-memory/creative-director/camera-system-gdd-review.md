---
name: camera-system-gdd-review
description: Camera System GDD review history — 3rd review 2026-08-09 still MAJOR REVISION NEEDED; direction sound, contracts/math/state-model/ACs failing; EF position-cache unbuildable
metadata:
  type: project
---

Camera System GDD (`design/gdd/camera-system.md`) — review history.

**Review 1 (2026-08-09)**: MAJOR REVISION NEEDED. Zoom model nonfunctional (ortho + height only = no zoom), follow math unimplementable, conversation pan-lock + 2s auto-return fought the investigation fantasy. Rulings established: no auto-return, no camera lock on the investigation verb, shake in screen-space with mandatory disable, fixed ortho angle is identity, "no void" via art padding not coverage math.

**Review 3 (2026-08-09, 6 specialists: game-designer, systems-designer, ux-designer, qa-lead, performance-analyst, lead-programmer)**: MAJOR REVISION NEEDED (third round). Direction and prior rulings preserved and correct; failures are now precision-level:
- EF contract unbuildable: cached position lists rely on `on_component_changed` which does NOT fire for Transform; no position-change event; `pc_controlled` tag has no change event. Fix: per-frame Transform reads of tracked set + membership lifecycle events (verified by performance-analyst + lead-programmer against source).
- CR-6 padding math wrong ~50%: Far-zoom 45°-rotated viewport AABB half-extent = (80+71.1)/2×sin(45°) = 53.4u per axis, not ~36/40u. Same AABB bug breaks combat K1 fixture (30×30 region-axes = 42.4u > 36u Default inner-90%).
- Shake AC unsatisfiable: peak 1.5% asserted but envelope×sine(0-phase) sampled max ≈ 1.0-1.3%; fix = assert envelope amplitude. Max-of-signed-values composition picks weaker shake; division-by-zero at 0-height viewport.
- State model false: Manual (rank 4) activates FROM higher states; Journal is overlay not state; Transition always exits to Exploration, dropping combat framing on boundary cross; combat end → Exploration follow = hidden drift-back (game-designer B1).
- Outline-through-geometry NOT supported by Kaiju (UI CSS outlines only) — hard requirement in CR-8 on unproven capability. Creative ruling: primary combatant-visibility mechanism = extend occluder fade to any geometry occluding any combatant; outline optional pending engine spike.
- Zoom ACs stale (discrete stepping) vs CR-2 free wheel + detents; Strict Zoom Levels toggle absent from Settings table and ACs.

**Why:** The camera is foundation-layer and serves Pillars 3 (readable combat) and 1 (frames NPCs/clues); its GDD must be implementable before anything downstream (renderer, combat UI, markers) commits.

**How to apply:** Rulings to enforce in future camera reviews: camera is player-owned by default — the game may borrow it only in declared, bounded, interruptible classes that restore the player's frame (combat initial frame, conversation intro, region fade); combat end returns to the player's manual frame if they adjusted; fade-based visibility preferred over outline overlays (world-coherence); conversation re-frames must complete in the inter-line gap, never behind displayed text; framing zoom = tightest canonical level that fits, starting from persisted combat zoom. Related: [[entity-framework-gdd-review]] (EF verdict: wrap Kaiju, 6 ADRs; cache unbuildable — camera and EF GDDs must be corrected together).

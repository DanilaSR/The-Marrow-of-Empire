---
name: title-screen-visual
description: Title screen visual brief delivered 2026-08-09 — riverside camp scene overrides art bible State 6 "ruined vista"; parchment panel left-third; pending art bible amendments (State 6 wording, 6.1 no-dim exception, GDD dropdown selection thread-red)
metadata:
  type: project
---

Title screen visual design brief completed 2026-08-09 (delivered as conversation output to ux-design session; file placement of `design/ux/title-screen-visual.md` pending parent/user decision — NOT yet written to disk).

Key decisions:
- Scene = riverside camp at golden hour (Drakensang-inspired, user decision) — overrides art bible State 6 "storybook dusk, vast ruined vista" and GDD "ruined archway". Art bible amendment proposed but NOT yet user-approved.
- Layout Option B: parchment panel in left third (traveler's journal), scene right two-thirds unobstructed, version bottom-right.
- Conflicts flagged: (1) UX spec interaction tables say "amber edge on focus" but art bible 6.5 + UX spec's own accessibility section say oxblood ring — resolved to oxblood ring; (2) GDD dropdown "amber selection highlight" vs art bible 6.5 selection = thread-red — resolved to thread-red; (3) all "fade from black" transitions violate prohibition 9.8 — use warm soot #141210 instead; (4) title screen panel is a desk overlay that does NOT dim the world (6.1 says desk dims 30%) — needs explicit exception codified.
- Two-light tease (State 6 "collision teased at a distance"): proposed as static ambiguous river-mist patch in far forest (never breathes/shimmmers — would deliver the collision); option B = drop. Awaiting user decision.
- Panel at 92–95% flat opacity (uniform alpha is flat — no gradients); no drop shadows; non-uniform hand-cut corner radii (3/5/4/6px); ink hairline separators.
- Scaling: s = height/1080 for layout; text clamps at 6.2 floors (16px body, 13px captions) — at 720p body renders 16px not 10.7px. Art bible 6.2 floor language forces clamping.
- Scene painting: 2048×1024 PoT exact 16:9, PNG; calm left band for panel backdrop; focal band (fire ~62%w, ships ~78–88%w, horizon ~56%h) inside center-safe area; ambient particles ≤100 total (smoke 24–32, embers ≤8, river sparkle ≤24, leaves ≤8); optional ship sway layer.

**Why:** user explicitly chose the riverside camp over the GDD's ruined archway during the ux-design session; art bible is source of truth for appearance, so it must be amended to match or the conflict recurs.
**How to apply:** when reviewing title screen work or running AD gates, check the amendment proposals (Section 3 State 6 wording, 6.1 exception, GDD deltas) are resolved; the scene decision is user-made and not negotiable back to the archway. Related: [[visual-identity-brief]].

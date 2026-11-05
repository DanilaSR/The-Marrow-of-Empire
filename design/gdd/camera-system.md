# Camera System

> **Status**: In Design
> **Author**: Claude + user
> **Last Updated**: 2026-08-09
> **Review**: MAJOR REVISION NEEDED → revised 2026-08-09 (see review log)
> **Implements Pillar**: Pillar 3 (Tactical Depth — readable combat spaces), Pillar 1 (Every Voice — frames NPCs and environmental clues)
>
> **Revision note (2026-08-09)**: 19 fixes applied following six-specialist adversarial review + creative-director synthesis. Key changes: EF position-cache mechanism corrected (direct Transform reads), state machine restructured (base states + overlay flags), near clip fixed (−0.75×O, was +1), art padding formula corrected (AABB half-extent, not unrotated), off-screen marker spec added with action-state encoding, occlusion test changed to screen-space AABB, ivory highlight → parchment, Reduce Camera Motion expanded to ladder covering shake+transitions+re-frames, combat silhouettes shape-differentiated, occlusion fade hysteresis added, verticality addendum, Extra Close accessibility zoom (O=2.5u), input fixes (dedicated Center-on-Selected key, rebindable second pan binding), AC suite corrected (CR-1 block, shake peak math, timing tolerances, measurement methods), combat fixture split (18×18u fits-Default + oversized Far-fallback), enemy-turn info gap + joiner re-frame rules, Save/Load dependency added.

## Summary

The Camera System provides the isometric orthographic viewpoint through which the
player sees The Marrow of Empire — a fixed 30°/45° angle at three discrete zoom
levels (Close/Default/Far) plus an accessibility Extra Close level. It follows the
party centroid during exploration, frames combatants during tactical encounters,
frames NPCs during conversation (without locking player control), and freezes for
the journal. It is the visual voice of the CRPG genre: a living diptych that makes
the dark fantasy borderland legible, beautiful, and real.

> **Quick reference** — Layer: `Foundation` · Priority: `MVP` · Key deps: Entity Framework (soft reads), Rendering System (Kaiju, hard downstream), Save/Load System (serialization contract), Asset & Scene Pipeline (roof tagging convention, hard)

## Overview

The Camera System defines how the player sees the world of The Marrow of Empire.
It provides an isometric projection — a fixed-angle orthographic view that presents
the game world with consistent scale, no perspective distortion, and full tactical
legibility. This is the signature viewpoint of the CRPG genre.

The system provides:

- **Isometric projection**: An orthographic camera at a fixed angle (approximately
  30° pitch, 45° yaw) that renders the world without perspective foreshortening.
  Objects at the back of a room are the same visible size as objects at the front —
  this is what makes tactical positioning legible at a glance.
- **Viewport management**: The camera renders to the full game viewport. UI
  elements (dialogue panel, combat HUD, journal) overlay the camera view without
  obscuring the tactical space.
- **Pan and scroll**: The camera follows the player's focus — typically centered
  on the selected character or party. Edge-scrolling and drag-panning allow the
  player to look ahead. Pan speed is tuned for deliberate exploration, not
  twitch navigation. The camera stays where the player leaves it — there is no
  auto-return. A dedicated "Center on Party" command (default: Home key) instantly
  returns the camera to the party.
- **Zoom levels**: Three discrete zoom levels — default (full tactical awareness),
  close (character detail, conversation), and far (region overview) — plus an
  accessibility Extra Close level (opt-in via settings). Smooth transitions between
  levels. No free zoom wheel — discrete steps make every frame a composed painting
  and make the art padding/no-void contract provable. Zoom persists per context
  and across save/load.
- **Focus and framing**: During conversations, the camera frames the speaking NPC
  and the player character together — but the player retains full pan and zoom
  control. During combat, the camera frames all active combatants once at combat
  start, then the player controls framing freely; off-screen combatant actions
  are communicated via action-aware edge markers. During exploration, it follows
  the party.
- **Boundary enforcement**: The camera is constrained to the current region's
  playable area. Regions include art padding beyond playable bounds so the camera
  never shows void space. The "no void" rule is enforced by art padding sized to
  the rotated viewport AABB, not the unrotated half-extents.
- **Occlusion handling**: Structures between the camera and the party fade to
  transparent (screen-space AABB overlap test, not volume-column). Interiors
  render without roofs when occupied. Interactable entities highlight through
  geometry. Combatants render with shape-differentiated outlines (solid for
  friendly, dashed for hostile). A wall never eats a decision.

Without the Camera System: the game world exists but the player cannot see it.
The isometric view IS the game's visual identity — it's how the dark fantasy
borderland, the Gothic ruins, and the pale empire remnants are composed and
presented to the player.

## Player Fantasy

The isometric camera is not neutral — it is the visual voice of the CRPG genre.
At its best, an isometric view creates three feelings simultaneously:

**The world is real.** A fixed overhead angle makes the world feel like a place
that exists independently of the player. Unlike a follow-cam that centers the
player in the universe, the isometric view says: "This world was here before you
arrived, and it will be here after you leave." The player is an observer looking
in, not the axis the world rotates around. The camera stays where the player puts
it — the world doesn't yank you back. Combat-start framing and conversation
initial framing are the only sanctioned exceptions: brief, deliberate moves that
serve tactical clarity and narrative presence respectively.

**You can see everything you need.** The orthographic projection, consistent scale,
and elevated angle make tactical information legible at a glance. Distance, cover,
threat radius — all readable without rotating the camera or squinting through fog.
This serves Pillar 3 (Tactical Depth): clarity is a combat asset. Off-screen
combatant actions are communicated through action-aware edge markers — you may
need to re-frame to see the full picture, but you'll never miss that something
happened. A player should never lose a fight because they couldn't see the
battlefield — and never lose track of their party because a wall swallowed them whole.

**The frame is a painting.** The isometric angle composes the world the way a
landscape painter composes a canvas — the golden forests, Gothic ruins, and pale
empire remnants arranged in a single readable frame. The art bible's warm-gold
light language and the dark fantasy aesthetic are delivered THROUGH this camera.
The discrete zoom levels reinforce this: each level is a deliberate composition,
not an arbitrary scroll-wheel position.

**Reference touchstones**: Baldur's Gate 1-2 (the iconic isometric CRPG viewpoint —
the world as a living map), Divinity Original Sin 2 (zoom levels + environmental
storytelling through the isometric frame), Diablo 2 (the dark fantasy isometric
mood — every frame a Gothic painting).

## Detailed Design

### Core Rules

**CR-1: Isometric Projection**

The camera uses orthographic projection at a fixed angle. No perspective — objects
at all depths render at the same scale.

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Projection type | Orthographic | Required for tactical legibility — no foreshortening |
| Pitch angle | 30° from horizontal | Standard CRPG isometric — balances depth readability with ground visibility |
| Yaw angle | 45° from cardinal axes | Standard isometric — shows two faces of structures, diagonal grid alignment |
| Roll | 0° | No dutch angle — the horizon is flat |
| Near clip | −0.75 × O (zoom-derived) | Negative near plane prevents ground-plane clipping at the viewport bottom edge. Engine default is −1 (deliberate for ortho cameras); the original spec of +1 created a 22–28% void strip at every zoom level. `SetNearPlane` in `StandardCamera` accepts negative values. |
| Far clip | 1000 world units | Sufficient for all region sizes. Engine defaults overridden explicitly. |
| Reference aspect ratio | 16:9 | Primary target; 16:10 and 21:9 also supported |

The camera angle is FIXED. The player cannot rotate the view. This is a design
decision, not a limitation — fixed angle means every environment is composed for
one viewpoint, every asset is modeled to read from one angle, and tactical
information is never hidden behind a bad camera rotation. Occlusion from structures
at the fixed angle is handled by the Occlusion Policy (see CR-8).

**World-axis convention**: The Entity Framework defines X = east-west, Y = elevation,
Z = north-south (per `entity-framework.md` CR-1). The camera's position formula
uses this convention: camera height is on the Y axis.

**Orthographic projection in Kaiju Engine**: Kaiju supports orthographic projection
natively via `NewStandardCameraOrthographic(width, height, viewWidth, viewHeight,
position)` in `kaiju/src/engine/cameras/standard_camera.go`. The camera is created
with world-space extents (NOT pixel viewport dimensions). After any window resize,
the ortho extents are immediately re-applied from the current (possibly interpolated)
ortho half-height.

**Aspect ratio handling**: The orthographic half-height is the canonical zoom
parameter. The horizontal extent varies with aspect ratio. At 16:9 reference,
horizontal extent = half-height × 2 × (16/9). At 21:9, the player sees ~33% more
horizontal world — acceptable for a non-competitive game. Minimum supported aspect
ratio: 16:10.

**Resize policy**: After any window resize, immediately re-apply `SetWidth`/`SetHeight`
from the current zoom level's canonical ortho half-height (interpolated value if
mid-transition). The `sizeIsViewSize` engine behavior must be bypassed.

**CR-2: Zoom Levels**

Three discrete zoom levels plus one accessibility level. Transitions are smooth
(lerp over `zoom_transition_duration`, default 300ms, game-time, delta-time based).
No free scroll wheel — discrete steps make every frame a composed painting.

Zoom is defined by the orthographic half-height (O) — the world-space half-extent
in the camera's vertical axis. At 30° pitch, the visible ground footprint is:

```
ground_depth  = 2 × O / sin(30°) = 4 × O
ground_width  = 2 × O × aspect_ratio
```

Domain constraint: θ ∈ (0°, 90°]. At θ = 0°, sin(θ) = 0 → division by zero.
Pitch is fixed at 30°; this constraint guards future configuration changes.

| Level | Ortho Half-Height (O) | Ground Footprint (16:9) | Use | Activation |
|-------|----------------------|------------------------|-----|------------|
| Extra Close (accessibility) | 2.5u | 8.9 × 10u | Low-vision clue inspection, character close-up | Enabled via Accessibility setting; when enabled, mouse wheel up from Close |
| Close | 4u | 14.2 × 16u | Character detail, indoor conversations, inspecting environmental clues | Mouse wheel up from Default |
| Default | 10u | 35.6 × 40u | Standard exploration, outdoor conversation, most combat encounters | Game start, recenter |
| Far | 20u | 71.1 × 80u | Region overview, travel planning, large encounters | Mouse wheel down from Default |

All zoom transitions use delta-time-based lerp (`t += dt / duration`, clamped to
[0, 1]). Frame-rate independent. At frame times ≥ 45ms, the per-frame delta may
exceed a monotonic-progress bound; the duration stretches to maintain smoothness —
accepted as correct frame-rate-independent behavior.

**Zoom persistence**: The player's last zoom level per context (Exploration,
Conversation, Combat) is persisted and restored on re-entry. Zoom preference is
never silently discarded — with one documented exception: combat-start framing
enforces a minimum of Default zoom to guarantee tactical legibility at encounter
start (Pillar 3). The player's zoom is respected the moment they manually adjust.
On save/load, both camera position and zoom level are restored to their saved
values — the camera does NOT reset to the party centroid on load.

**CR-3: Camera Focus and Follow**

The camera has a focus point — the world position it is centered on. The focus
point moves to follow the player's attention. The state model uses **base states +
overlay flags** (see States and Transitions).

| Context (Base State) | Focus Target | Follow Behavior |
|----------------------|-------------|-----------------|
| Exploration | Party centroid (average position of all party members) | Constant-velocity move-toward at `follow_speed_exploration`. No snapping. When party spread > one viewport half-extent, follow the selected character; otherwise follow centroid. |
| Single character selected | Selected character position | Same smooth follow |
| Manual pan (overlay flag) | Player's drag/edge-scroll target | Camera stays at manual target. **No auto-return.** Player recovers via "Center on Party" (Home) or by selecting a party member. Manual is an overlay on the active base state — the base state's framing rules resume when the player re-centers. |
| Conversation | Midpoint between player character and speaking NPC | Frames on conversation start. Player retains full pan and zoom. Re-frame on speaker change only if player hasn't manually adjusted within `conversation_override_window` (5s) AND the new speaker is outside the inner 70% of the viewport. |
| Combat | Bounding rectangle of all active combatants (party + visible enemies) | Frames once at combat start (see CR-7). Player immediately controls framing freely. No centroid chase. Re-frames on turn boundaries only if player hasn't manually adjusted. New combatants joining mid-encounter trigger a re-frame (see CR-7). |
| Journal open (overlay flag) | Last known position | Camera freezes — journal overlays. Player can pan/zoom. On journal close, camera returns to the base state's framing. |

**Center on Party command**: Pressing Home (rebindable) instantly moves the camera
focus to the party centroid at the current zoom level. No lerp — immediate reposition
within one frame (≤33ms at 30fps, ≤16.7ms at 60fps). A dedicated "Center on Selected"
command (default: End key, rebindable) centers on the currently selected character.
No double-tap — each command has its own binding.

**Off-screen markers**: When any tracked entity (party member, combatant) is outside
the camera viewport, an edge-of-screen marker indicates their presence. Markers are
active whenever a tracked entity is outside the viewport, regardless of camera state.
See Off-Screen Markers spec (UI Requirements).

**Follow semantics**: Constant-velocity move-toward with distance clamping:
`offset = target - position; step = min(follow_speed × dt, |offset|);
position += normalize(offset) × step`. The camera reaches the target exactly.
This is NOT an exponential ease — it is a linear move-toward.

**Follow speed constraint**: `follow_speed_exploration` must exceed the maximum
party movement speed. The movement system GDD (not yet authored) must provide the
party walk speed so the lower bound (≥1.2× party speed) can be verified.

**Position data source**: The camera reads `Transform.Position()` directly for
tracked entities each frame (party 1–6, combatants ~10–30, speaking NPC 1). This
is a small fixed set — NOT a spatial-index range query. Membership changes arrive
via EF lifecycle events (`on_component_added`/`on_component_removed` for PartyMember
and Combatant components). This replaces the original `on_component_changed`
mechanism, which does not fire for Transform changes in Kaiju Engine.

**CR-4: Pan Controls**

The player can move the camera focus manually:

- **Edge scroll**: Cursor at screen edge pans at `pan_speed_edge` (default 15 u/s).
  Dead zone: `edge_scroll_deadzone` (default 16 logical px, configurable 4–64px, DPI-scaled).
  Hysteresis margin: `edge_scroll_hysteresis` (default +50% of deadzone) — cursor must
  retreat past deadzone + margin to stop pan, preventing jitter at boundary.
  Edge scroll DISABLED when cursor is over a UI panel.
- **Drag pan**: Hold pan button (default: middle mouse, rebindable) and drag.
  Point-under-cursor tracking — the world-space point at drag start remains under cursor.
  Pixel threshold: 8 logical px before engaging (prevents micro-drift).
  Drag Pan Mode: Hold / Toggle / Off.
  Sensitivity configurable (default 100%, range 25–200%).
  An additional rebindable pan binding is available (unassigned by default) — Alt+Left-drag
  is NOT hardcoded to avoid conflict with combat Alt+click commands.
- **Keyboard pan**: Arrow keys / WASD at `pan_speed_keyboard` (default 20 u/s). Rebindable.
  Suppressed when a text input field has focus (journal notes, search, character naming).
- **Keyboard zoom**: Z/X or PageUp/PageDown cycle zoom levels. Rebindable.
- **Gamepad pan**: Right stick at `pan_speed_gamepad` (default 15 u/s), configurable dead zone.
  Shoulder buttons cycle zoom. Right Stick click = Center on Party (rebindable).
- **Pan limit**: Camera cannot move beyond region boundary (see CR-6).

**CR-5: Camera Shake**

Combat and environmental events trigger camera shake for feedback. Shake is a
lateral offset along the screen-horizontal world axis `(−sin φ, cos φ)` where
φ = 45° yaw. Shake never adds offset along the camera's height axis.

**Shake is defined in screen-space percentages** — zoom-independent by construction.

| Event | Peak Displacement | Duration | Falloff |
|-------|------------------|----------|---------|
| Critical hit | 1.5% of viewport | 200ms | Linear: `falloff(t) = 1 − t/duration` |
| Large creature death | 3.0% of viewport | 400ms | Ease out: `falloff(t) = (1 − t/duration)²` |
| Environmental collapse (scripted) | 5.0% of viewport (hard cap) | 800ms | Ease out: `falloff(t) = (1 − t/duration)²` |
| Nearby explosion | 2.5% of viewport | 300ms | Linear: `falloff(t) = 1 − t/duration` |

At 1920×1080, the shorter dimension is 1080px — critical hit waveform peak: ~16px.

**Shake peak note**: The configured percentages are waveform amplitudes. The achieved
peak at the first sine crest (t ≈ 31.25ms) is approximately 0.844 × amplitude with
linear falloff (~1.27% for critical hit). The ACs assert the achieved peak.

**Shake composition**: At each frame, use the signed offset with the largest magnitude
across all active events. Preserve the sign of the winning offset. Events do not stack.

**Shake waveform**: Sine at 8 Hz with specified falloff envelope. Deterministic —
same event at same game-time = identical output (required for test reproducibility).

**Shake guard**: If `viewport_height_px = 0` (minimized window), skip shake — prevents NaN.

**Shake and boundary clamp**: Applied AFTER clamp. Amplitude reduced if it would show void.

**Accessibility — Reduce Camera Motion**: A unified ladder covering all camera motion:

| Setting | Shake | Zoom Transitions | Conversation Re-frames | Combat Framing |
|---------|-------|-----------------|----------------------|----------------|
| **Off** | Full per CR-5 | Smooth lerp | On speaker change (if outside inner 70%) | 40 u/s transition |
| **Reduced** (default on first launch) | 50% amplitude, 50% duration | Instant snap (no lerp) | Initial frame only; no re-frames | Snap to frame (no transition) |
| **Disabled** | None | Instant snap | Initial frame only | Snap to frame |

- Shake is **disabled entirely during Conversation state** regardless of setting —
  the player is reading text; the world must not vibrate behind it.
- Shake never uses rotation — permanently prohibited (strongest vestibular trigger).
- A **first-launch accessibility prompt** offers Reduced motion + subtitles, keyboard/
  gamepad-navigable, no timeout, Skip = accessible defaults (Reduced + subtitles ON).
- Reduce Camera Motion persists to profile, discoverable under a dedicated Accessibility
  settings page.

**CR-6: Boundary Enforcement**

The camera viewport must never show void space. Enforced by art padding + hard clamp:

- **Region art padding** (per world axis):
  ```
  padding ≥ (ground_width_max + ground_depth) / (2 × √2)
  ```
  At Far zoom, widest supported aspect (21:9):
  ```
  ground_width_max = 2 × 20 × (21/9) = 93.3u
  ground_depth     = 2 × 20 / sin(30°) = 80u
  padding ≥ (93.3 + 80) / (2 × 1.414) = 61.3u per world axis
  ```
  Round to ≥ 62u per side for production. At 16:9: ~53.4u. The widest aspect governs —
  padding is baked into art.

  **Production note**: This correction increases region art area by ~25% over the original
  ~36/40u estimate (which used unrotated half-extents). The producer must see this number
  before region art is scoped.

- **Hard clamp**: Camera position clamped to region bounds minus AABB viewport half-extent.
  At region edges, the party may render off-center — correct and expected.
- **Minimum region size**: Default-zoom viewport + padding × 2 ≈ 142 × 147u at 16:9.
  Smaller regions (interiors) use region transitions with their own boundary definitions.
- **Region transitions**: Fade to black (`region_fade_duration`, default 300ms), reposition
  to entry point, fade in (300ms). Movement, camera, and interaction inputs locked during
  fade; menu navigation remains functional. After fade-in, camera re-enters the pre-transition
  base state (not unconditionally Exploration). Combat/conversation events during fade are
  queued and processed after fade-in.
- **Shake margin**: Padding includes ~2.7u for shake at the boundary (5% of viewport shorter dimension at Far, 1080p).

**CR-7: Combat Framing**

Combat framing is a once-at-start operation, not a continuous follow:

1. **Combat start**: Camera frames the bounding rectangle of all active combatants
   (party + visible enemies). Bounding box is computed in the elevation plane of the
   encounter. Framing zoom = player's persisted Combat zoom, minimum Default —
   this documented exception serves Pillar 3 (tactical legibility at encounter start).
   Far is used if the box doesn't fit at Default with a 10% margin.
   **Single combatant**: pad box to minimum 8u × 8u in the XZ plane.
   **Doesn't fit at Far**: frame at Far, include party always, excess enemies get edge markers.
2. **After initial frame**: Player has full pan and zoom control. No centroid chase.
3. **Turn boundaries**: Camera re-frames (same rule) on turn advancement ONLY if the
   player hasn't manually panned/zoomed since combat start. Manual input blocks turn-boundary
   re-frames for that encounter — but does NOT block new-combatant re-frames (rule 7).
4. **Combat end**: Transition to Exploration at player's current zoom and position.
5. **Outlier combatants**: Beyond Far-zoom viewport extent → excluded from framing box,
   receive edge markers with action-state encoding.
6. **Combat start during manual pan**: Camera frames encounter at `combat_framing_speed`
   (40 u/s). For distances > 100u, use fade-to-black (same as region transition) and hold
   the first player turn until framing completes. Player can interrupt by panning.
7. **New combatants mid-encounter** (reinforcements, summons, ambush): Trigger re-frame on
   next turn boundary, or immediately if new count exceeds current framed count. This fires
   regardless of the manual-input flag — player's zoom preserved, framing updates.
8. **Off-screen combatant actions**: When a combatant outside the viewport takes an action,
   their edge marker flashes with an action indicator (see Off-Screen Markers spec).
   An optional "Follow Active Unit" setting (default off, under Accessibility) briefly pans
   to off-screen actors during their turn.

**CR-8: Occlusion Policy**

The fixed 30° pitch + Gothic architecture means structures will occlude. The following
guarantee "a wall never eats a decision":

- **Occlusion test (screen-space AABB overlap)**: For each structure in the camera-owned
  per-region structure list (built at region load, NOT queried from the EF spatial index
  per frame), project its world-space AABB to screen space. If the screen-space rect overlaps
  the party's screen-space rect (4 tests for 4 party members) and the structure is between
  the party and the camera in view-space depth, the structure is an occluder. This replaces
  the original volume-column test which faded every structure in the 3D column.

- **Occluder fade**: Fade per-mesh (not per-object — a cathedral doesn't ghost entirely
  because one arch pier occludes). Target alpha: `occluder_fade_alpha` (default 20% for
  solid geometry, 35% for foliage materials). Transition: `occluder_fade_duration` (150ms).
  Minimum alpha across all active fades when a structure occludes multiple entities.

- **Fade hysteresis**: Faded state holds for `occluder_fade_min_hold` (300ms). Occlusion
  condition must clear for `occluder_fade_min_clear` (300ms) before restore. This bounds
  toggle frequency below 2 Hz — prevents alpha oscillation in the 3–6 Hz WCAG 2.3.1 hazard
  band when walking alongside a wall.

- **Roof removal**: When the party enters an interior, roof/ceiling geometry above the
  party's elevation is culled. Roofs restore on exit. Multi-story: cull every horizontal
  slab above party's feet elevation. Requires semantic part tags ("roof", "ceiling") at
  asset import — **hard dependency on Asset & Scene Pipeline GDD** (see Dependencies).

- **Interactable highlight**: Occluded interactables render with a 2px solid outline
  (configurable 2/4/6px), color: parchment `#E3D5B0` (art bible UI register). This replaces
  the original pale ivory — ivory is rationed to empire-significant relics per art bible §2.4.
  Same highlight used for the Tab-key toggle. Outline renders unlit, screen-space, full opacity.

- **Combat silhouette**: Occluded combatants render with shape-differentiated outlines
  (not hue-only — satisfies WCAG 1.4.1 and the project's art-bible rule):
  friendly = solid outline (amber gold `#D9A441`), hostile = dashed outline (thread red
  `#C4483A`). Thickness configurable (2/4/6px, default 2px). Outlines unlit, screen-space,
  full opacity. Always active during combat — no key press. Combat silhouette takes precedence
  over interactable highlight for dual-role entities.

- **Rendering passes**: Silhouettes require a mask pass (combatant meshes, depth disabled,
  per-entity color uniform → mask RT) + fullscreen composite (dilate mask by N px, composite
  outline color over main image, before UI camera). Ghost pass uses blend-enabled pipeline
  variant with per-instance alpha; manual Sort per frame by view-space depth. **New engine
  infrastructure** — tracked as hard downstream dependency on Rendering System.

- **Shadow exclusion**: Faded structures must not cast shadows. Per-draw-instance shadow
  toggling exists in Kaiju (`ShaderDataBase.shadows`, `Activate()`/`Deactivate()`).
  Shadows disabled when alpha < 50% on fade-out, re-enabled when alpha > 95% on fade-in.

### States and Transitions

The camera uses **base states + overlay flags** rather than a flat state list.
The base state represents the active context (Exploration, Conversation, Combat).
Overlay flags (Manual, Journal) modify behavior on top of the base state.

**Base State Priority** (highest to lowest):
1. Transition (exclusive — no other state active during fade)
2. Combat
3. Conversation
4. Exploration

**Overlay Flags** (independent of base state):
- **Manual**: Camera at a player-chosen position. No auto-return. Overrides follow.
- **Journal**: Investigation journal open. Camera holds position. Player can pan/zoom.

| Base State | Entry Condition | Behavior | Exit Condition |
|------------|----------------|----------|----------------|
| `Exploration` | Game start, combat ends, conversation ends, journal closes | Smooth follow at party centroid; all pan modes enabled; player's last Exploration zoom | Combat starts, conversation starts, journal opens, player pans (Manual overlay) |
| `Conversation` | Player initiates dialogue | Frames PC+NPC midpoint at start; player retains full pan/zoom; re-frame only if speaker outside inner 70% AND player hasn't adjusted within 5s; shake disabled | Dialogue ends → Exploration |
| `Combat` | Combat encounter starts | Frames bounding box once at start (player's Combat zoom, Default min); player controls freely; turn-boundary re-frame if no manual input; joiner re-frame regardless; shake active | Combat ends → Exploration |
| `Transition` | Region boundary crossed | Fade to black (300ms) → reposition → fade in (300ms); input locked (movement/camera/interaction; menus functional) | Fade-in complete → restore pre-transition base state |

**Overlay behavior:**

| Overlay | Activation | Effect | Deactivation |
|---------|-----------|--------|--------------|
| `Manual` | Player drags, edge-scrolls, or keyboard-pans | Camera holds manual position; base state follow suspended; off-screen markers active; base state context preserved (combat re-frame rules, conversation override timer tracked) | Press Home, End, or select party member → return to base state framing |
| `Journal` | Player opens investigation journal | Camera holds position; player can pan/zoom; journal UI overlays; during Combat base state, combat framing persists with combat UI visible behind journal | Journal closes → return to base state framing (NOT unconditionally Exploration) |

**Preemption rules:**
- Combat preempts Conversation: combat starts mid-dialogue → Combat base state. Combat ends → return to Conversation (one-deep preemption stack).
- Combat preempts Journal: journal open, combat starts → Combat base state, journal dismissed. Combat ends → Exploration.
- Same-frame arbitration: Transition first, then Combat lifecycle, then Dialogue lifecycle, then player input. Combat wins over dialogue when both arrive same frame.
- During Transition: combat/conversation events queued, processed after fade-in.

**Manual flag + base state:**
- Manual during Exploration: follow suspended. Selecting party member or Home re-engages.
- Manual during Conversation: follow suspended, re-frame timer tracked. Dialogue end → Exploration (not stranded).
- Manual during Combat: follow suspended, turn-boundary re-frames suppressed (permanent). Joiner re-frames still fire. Combat end → Exploration.

### Verticality

**Elevation convention**: Per Entity Framework CR-1, Y = elevation in world space.
Camera position formula uses Y for height. Combat framing bounding box computed in
the XZ plane at the elevation of the majority of combatants.

**Ground plane as occluder**: For below-ground interiors (crypts, catacombs), the
ground plane is geometry between camera and focus. It is included in the camera-owned
region structure list and treated as an occluder per CR-8 — fades when party enters
a below-ground space. Region transition fade covers the transition.

**Multi-story interiors**: Each story is either a separate interior region (entered via
region transition) or a single region with story-indexed roof/ceiling culling (see
roof removal, CR-8). Level design chooses per region.

**Rooftop combat**: Combatants at different elevations use standard combat framing
(XZ plane bounding box). Occlusion from their standing structure handled by CR-8.

### Interactions with Other Systems

| System | Direction | Interaction |
|--------|-----------|-------------|
| Entity Framework | Reads entity transforms (SOFT) | Reads `Transform.Position()` directly for tracked entities each frame (party 1–6, combatants ~10–30, speaking NPC 1). Membership via `on_component_added`/`on_component_removed` for PartyMember and Combatant components. No per-frame spatial-index range queries — contract preserved. Camera registers in `Gameplay` priority band. |
| Tactical Combat Engine | Reads combat state (SOFT) | Enters/exits Combat base state; reads Combatant component set; receives new-combatant-joined events |
| Dialogue UI | Reads conversation state (SOFT) | Enters/exits Conversation base state; reads speaking NPC position |
| Investigation Journal | Reads journal state (SOFT) | Engages Journal overlay flag |
| Audio System | Audio reads camera position + orientation (HARD downstream) | Camera position = spatial audio listener (unshaken position). Camera exports forward/up vectors. Region transitions require audio cross-fade. **Engine gap**: Kaiju Go wrapper lacks 3D audio API; C layer has it. Requires engine programmer. |
| Rendering System (Kaiju) | Renderer reads camera matrices (HARD downstream) | Camera provides view/projection matrices each frame. Additionally: ghost pass (blend-enabled pipeline, per-instance alpha), silhouette mask + fullscreen composite passes, per-draw-instance shadow toggling. **New engine infrastructure** — requires technical-director. |
| Save/Load System | Mutual (HARD) | Camera serializes: position, zoom, per-context zoom preferences, base state, Manual flag, preemption stack. On load: restores all serialized state — NOT centroid reset. Save blocked during: in-flight zoom transitions, shake events, region fades, conversation framing transitions. Combat saves allowed (camera restores Combat state at saved position/zoom). **Save/Load GDD must be updated** — current revision has contradictory rules. |
| HUD System | Camera provides data (HARD downstream) | Provides viewport bounds + tracked entity list for off-screen markers. See Off-Screen Markers spec. |
| Asset & Scene Pipeline | Camera reads semantic part tags (HARD downstream) | Roof removal depends on per-mesh tags ("roof", "ceiling", "wall", "ground"). **Must be defined before first asset import** — blocking dependency. |
| Living Environment | None direct | Environment visuals render through camera viewport |
| Map System | Coordinates with | Map reads camera bounds; camera exposes `SetFocus` for map-click-to-center |

## Off-Screen Markers

Owned by the HUD system. Camera provides the data contract; HUD renders markers.
Markers are active whenever a tracked entity is outside the viewport, regardless
of camera state.

### Camera → HUD Data Contract

Per update, the camera provides:

| Field | Type | Description |
|-------|------|-------------|
| `viewport_bounds` | Rect (screen-space) | Current viewport in logical pixels |
| `tracked_entities` | Array of TrackedEntity | All tracked entities (party + combatants) |

Each `TrackedEntity`:

| Field | Type | Description |
|-------|------|-------------|
| `entity_id` | GameEntityID | Stable reference for selection/focus |
| `world_position` | Vector3 | Current world position |
| `display_name` | string | Character name for label |
| `entity_type` | enum | PartyMember / FriendlyCombatant / HostileCombatant / Neutral |
| `action_state` | enum | Idle / Moving / Targeting / Telegraphing / Acting |
| `action_target` | GameEntityID? | Target of current action |
| `action_label` | string? | Ability/action name for display |
| `is_selected` | bool | Current player selection |

### Marker Visual Spec (HUD-owned)

| Property | Spec |
|----------|------|
| **Shape** | Directional chevron (▲) pointing toward entity |
| **Size** | ≥ 16px at 1080p, configurable: Small 12px / Default 16px / Large 24px |
| **Contrast** | ≥ 3:1 — dark backing plate (#1A1A1A at 70% opacity) + light glyph |
| **Friendly (party)** | Solid chevron, amber gold `#D9A441` |
| **Hostile (enemy)** | Dashed chevron, thread red `#C4483A` |
| **Neutral** | Dotted chevron, parchment `#E3D5B0` |
| **Distance** | Label in world units below chevron (e.g., "12u"), UI-scale-aware, ≥ 14px |
| **Identity** | Display name above chevron; "???" for unidentified |
| **Action indicator** | Flash (pulse scale 1.0→1.3→1.0, 200ms) when `action_state` = Acting. Action label adjacent for 2s (e.g., "Arrow Shot") |
| **Animation** | Slow pulse ≤ 1 Hz idle. Faster 2 Hz pulse when Targeting/Telegraphing. All animations disable-able via Reduce Camera Motion |
| **Clustering** | ≥3 markers same edge within 40px → merge: chevron + count badge (e.g., "3"). Expand on hover |
| **Edge inset** | ≥ 32px from viewport edge; additional offset below UI panels (dialogue, combat HUD) |
| **Clickability** | Party member marker click = select + center. Enemy marker click = center only. Right-click = dismiss |
| **Settings** | Size, opacity (25–100%), distance labels (On/Off), animation (On/Off), always-show (party) |

## Occlusion Policy

See CR-8 in Detailed Design. This is a first-class system rule, not an
implementation detail — it directly serves Pillar 3 (Tactical Depth) and the
cross-cutting constraint "a wall never eats a decision."

## Formulas

### Orthographic Ground Footprint

At pitch angle θ and ortho half-height O:
```
ground_depth  = 2 × O / sin(θ)    (domain: θ ∈ (0°, 90°])
ground_width  = 2 × O × AR
```
At θ = 30°, sin(30°) = 0.5, so ground_depth = 4 × O.

### Zoom Level Extents

| Level | O (half-height) | Depth (16:9) | Width (16:9) | Width (21:9) |
|-------|----------------|-------------|-------------|-------------|
| Extra Close | 2.5u | 10u | 8.9u | 11.7u |
| Close | 4u | 16u | 14.2u | 18.7u |
| Default | 10u | 40u | 35.6u | 46.7u |
| Far | 20u | 80u | 71.1u | 93.3u |

### Near Clip (Zoom-Derived)

```
near = −0.75 × O
```
Extra Close: −1.875, Close: −3, Default: −7.5, Far: −15.
Guarantees ground plane reaches viewport bottom edge at all zoom levels.

### Region Art Padding

```
padding ≥ (ground_width_max + ground_depth) / (2 × √2)
```
At Far zoom, 21:9: `padding ≥ (93.3 + 80) / (2 × 1.414) = 61.3u` → round to ≥ 62u/side.

### Follow (Constant-Velocity Move-Toward)

```
offset = target_position - camera_position
distance = |offset|
step = min(follow_speed × dt, distance)
if distance > 0:
    camera_position += (offset / distance) × step
```
Move-toward with distance clamping — camera always reaches target exactly.
Not an exponential ease.

### Shake Displacement

Shake axis: screen-horizontal world direction `(−sin φ, cos φ)`, φ = 45°. XZ only.

```
shake_offset_pixels = (shake_percentage / 100) × min(viewport_width_px, viewport_height_px)
shake_world_units = shake_offset_pixels × (2 × O) / viewport_height_px
if viewport_height_px > 0:
    waveform = sin(2π × 8Hz × t) × falloff(t)
    camera_xz += shake_direction × shake_world_units × waveform
```
Falloffs: linear = `1 − t/duration`, ease-out = `(1 − t/duration)²`.
Achieved peak at first sine crest ≈ 0.844 × amplitude (linear) or ≈ 0.96 × amplitude (ease-out).
ACs assert the achieved peak.

### Camera Position from Focus

World-axis convention: X = east-west, Y = elevation, Z = north-south (EF CR-1).
For ortho pitch θ = 30°, yaw φ = 45°:
```
camera_x = focus_x - O × cos(θ) × sin(φ)
camera_y = focus_y + O × sin(θ)
camera_z = focus_z - O × cos(θ) × cos(φ)
```
Camera height = 0.5 × O: 1.25u (Extra Close) to 10u (Far).

### Combat Bounding Box Fit Test

A world-space AABB of side S (XZ plane) projects to screen-space:
```
screen_width = S × 1.414   (horizontal, yaw 45°)
screen_depth = S × 0.707   (vertical, foreshortened by sin 30°)
```
Binding constraint is horizontal. Fit at zoom O: `S × 1.414 ≤ 2 × O × AR × 0.9`.
Default (O=10, 16:9): S ≤ 22.6u. Far (O=20): S ≤ 45.3u.

## Edge Cases

- **If the player initiates conversation while an enemy is aggro'd but not yet in combat**: Camera stays in Conversation base state. If combat starts mid-conversation, Combat preempts Conversation (one-deep stack). Combat ends → return to Conversation.
- **If all combatants die simultaneously**: Camera transitions to Exploration at current zoom/position after a 1-second delay (death animation). Destroyed entities not queryable per EF CR-4.
- **If the party splits across a region boundary**: Camera follows selected character, or largest group if none selected. Off-screen party members show edge markers.
- **If party members split within same region** (looting 30u apart): Centroid may frame empty space. Follow selected character when party spread > one viewport half-extent; otherwise centroid. Markers always active for off-screen members.
- **If the player opens the journal during combat**: Journal overlay; camera stays in Combat framing with combat UI visible behind journal. Player can pan/zoom. Journal close → combat framing resumes.
- **If a conversation starts at Far zoom**: Camera zooms to last Conversation zoom (default: Close) while panning to PC+NPC midpoint — simultaneous over `conversation_snap_duration × 2` (400ms). Player retains full pan/zoom.
- **If the player's party centroid is at the edge of the playable area**: Hard boundary clamp — party may be off-center. No auto-return; use Center on Party.
- **If a window resize occurs mid-zoom-transition**: Ortho extents immediately re-applied from current interpolated O (not deferred). No visual pop.
- **If a shake event triggers at a region boundary**: Shake after clamp. Amplitude reduced if it would show void.
- **If combat starts while the player is manually panned away from the party**: Frame encounter at `combat_framing_speed` (40 u/s). Distance > 100u → fade-to-black + hold first turn. Player can interrupt by panning.
- **If multiple occlusion fades target the same structure**: Minimum alpha across all active fades. Hysteresis: min-hold 300ms, min-clear 300ms — prevents flicker during wall-walking.
- **If the player has zoomed to Close during combat and an enemy moves outside the viewport**: Edge markers with action-state encoding appear. Player responsible for re-framing. If enemy acts, marker flashes with action label.
- **If new combatants join mid-encounter** (reinforcements, summons): Re-frame on next turn boundary, or immediately if new count > current framed count. Overrides manual-input flag; player's zoom preserved.
- **If combat crosses a region boundary**: Camera enters Transition. Combat queued and restored after fade-in. Encounters should be region-locked by design.
- **If the game window is minimized**: `viewport_height_px = 0`. Skip shake. Camera update uses last known viewport dimensions; ortho extents re-applied on restore.
- **If multiple fade targets include foliage**: Foliage materials use higher alpha floor (35% vs 20% solid) to prevent canopy skeletonization.

## Dependencies

| System | Direction | Nature of Dependency |
|--------|-----------|---------------------|
| Entity Framework | Camera reads entity transforms (SOFT) | Reads `Transform.Position()` directly for tracked entities (~15 reads/frame). Membership via `on_component_added`/`on_component_removed`. No per-frame spatial-index range queries. |
| Tactical Combat Engine | Camera reads combat state (SOFT) | Enters/exits Combat base state; receives new-combatant-joined events |
| Dialogue UI | Camera reads conversation state (SOFT) | Enters/exits Conversation base state; reads speaking NPC position |
| Investigation Journal | Camera reads journal state (SOFT) | Engages Journal overlay flag |
| Audio System | Audio reads camera position + orientation (HARD, downstream) | Camera position = spatial audio listener (unshaken). **Engine gap**: Kaiju Go wrapper lacks 3D audio API. |
| Rendering System (Kaiju) | Renderer reads camera matrices (HARD, downstream) | View/projection matrices each frame. Additionally: ghost pass, silhouette mask + composite passes, shadow toggling. **New engine infrastructure**. |
| Save/Load System | Mutual (HARD) | Camera serializes: position, zoom, per-context zoom prefs, base state, manual flag, preemption stack. On load: restores all — NOT centroid reset. **Save/Load GDD must be updated**. |
| HUD System | Camera provides data (HARD, downstream) | Viewport bounds + tracked entity list for off-screen markers |
| Asset & Scene Pipeline | Camera reads semantic part tags (HARD, downstream) | Roof removal depends on per-mesh tags ("roof", "ceiling"). **Must be defined before first asset import**. |
| Living Environment | None direct | Environment visuals render through camera viewport |
| Map System | Coordinates with | Map reads camera bounds; camera exposes `SetFocus` |
| Main Menu & Settings | Camera reads settings (SOFT) | Reads player-configured speeds, dead zones, keybinds, accessibility toggles |

## Tuning Knobs

| Parameter | Default | Safe Range | Notes |
|-----------|---------|------------|-------|
| `ortho_half_height_extra_close` | 2.5u | 1.5–3 | Accessibility: enabled via toggle |
| `ortho_half_height_close` | 4u | 3–6 | |
| `ortho_half_height_default` | 10u | 7–14 | |
| `ortho_half_height_far` | 20u | 15–30 | |
| `follow_speed_exploration` | 5 u/s | ≥1.2× party speed – 15 | Lower bound TBD (movement GDD) |
| `zoom_transition_duration` | 300ms | 150–600 | |
| `pan_speed_edge` | 15 u/s | 5–30 | |
| `pan_speed_keyboard` | 20 u/s | 10–40 | |
| `pan_speed_gamepad` | 15 u/s | 5–30 | |
| `edge_scroll_deadzone` | 16 logical px | 4–64 | DPI-scaled |
| `edge_scroll_hysteresis` | +50% of deadzone | 25–100% | |
| `drag_pan_threshold` | 8 logical px | 4–32 | |
| `drag_pan_sensitivity` | 100% | 25–200% | |
| `conversation_snap_duration` | 200ms | 100–500 | |
| `conversation_override_window` | 5s | 2–10 | |
| `shake_intensity_global` | 1.0 | 0.0–2.0 | Capped by hard maxima for collapse events |
| `reduce_camera_motion` | Reduced | Off / Reduced / Disabled | Accessibility ladder |
| `region_fade_duration` | 300ms | 150–600 | |
| `occluder_fade_alpha` | 20% (solid) / 35% (foliage) | 10–40% | |
| `occluder_fade_duration` | 150ms | 75–300 | |
| `occluder_fade_min_hold` | 300ms | 150–600 | Hysteresis: minimum fade duration |
| `occluder_fade_min_clear` | 300ms | 150–600 | Hysteresis: minimum clear before restore |
| `combat_framing_speed` | 40 u/s | 20–80 | |
| `combat_framing_fade_distance` | 100u | 50–200 | |
| `outline_thickness` | 2px | 2/4/6px | Accessibility |
| `marker_size` | 16px (Default) | 12/16/24px | |
| `marker_opacity` | 100% | 25–100% | |

## Visual/Audio Requirements

- **Consistency**: Pitch 30° ± 0.1°, yaw 45° ± 0.1°, roll 0° at all times.
- **Boundary transitions**: Screen-space overlay (black quad, UI layer, topmost z-order).
- **Occlusion fades**: Faded structures must not cast shadows (per-draw-instance shadow toggle: off when alpha < 50%, on when alpha > 95%).
- **Silhouette outlines**: Unlit, screen-space, full opacity. Shape-differentiated: solid (friendly, amber `#D9A441`), dashed (hostile, red `#C4483A`). Color is redundant; shape is primary.
- **Interactable highlight**: Solid outline, parchment `#E3D5B0`.
- **Orthographic extents**: Immediately re-applied after any window resize from current O.
- **Near clip**: Must not clip ground plane — `near = −0.75 × O`.

## UI Requirements

- UI panels render in screen space — unaffected by camera shake or transitions.
- Edge scroll disabled when cursor is over any UI panel (requires UI "cursor over" query).
- Off-screen markers owned by HUD — camera provides viewport bounds + tracked entity data per the Off-Screen Markers spec.
- First-time off-screen marker triggers a one-time toast: "Party off-screen — press Home to center."
- First detected drag opportunity triggers a pan gesture hint. Both owned by UI/onboarding system.

### Controls & Settings

**Camera settings** (Settings → Camera):

| Setting | Type | Default |
|---------|------|---------|
| Edge Scroll | Dropdown | On (On / Hold-to-activate / Off) |
| Edge Scroll Dead Zone | Slider | 16 px (4–64) |
| Edge Scroll Speed | Slider | 15 u/s (5–30) |
| Keyboard Pan Speed | Slider | 20 u/s (10–40) |
| Gamepad Pan Speed | Slider | 15 u/s (5–30) |
| Gamepad Pan Dead Zone | Slider | 15% (5–40%) |
| Drag Pan Mode | Dropdown | Hold (Hold / Toggle / Off) |
| Drag Pan Sensitivity | Slider | 100% (25–200%) |
| Zoom Transition Speed | Slider | 300ms (150–600) |
| Invert Zoom Direction | Toggle | Off |
| Center on Party | Keybind | Home (rebindable) |
| Center on Selected | Keybind | End (rebindable) |
| Pan Up/Down/Left/Right | Keybinds | WASD / Arrows (rebindable) |
| Zoom In / Zoom Out | Keybinds | Z / X, PageUp / PageDown (rebindable) |
| Drag Pan Button | Keybind | Middle Mouse (rebindable) |
| Secondary Drag Pan Button | Keybind | (unassigned, rebindable) |
| Gamepad Pan | Axis | Right Stick (rebindable) |
| Gamepad Zoom | Buttons | LB / RB (rebindable) |
| Gamepad Center on Party | Button | Right Stick Click (rebindable) |

**Accessibility settings** (Settings → Accessibility):

| Setting | Type | Default |
|---------|------|---------|
| Reduce Camera Motion | Dropdown | Reduced (Off / Reduced / Disabled) |
| Extra Close Zoom | Toggle | Off |
| Follow Active Unit (combat) | Toggle | Off |
| Outline Thickness | Dropdown | 2px (2 / 4 / 6) |
| Marker Size | Dropdown | Default 16px (Small 12 / Default 16 / Large 24) |
| Marker Opacity | Slider | 100% (25–100%) |
| Marker Distance Labels | Toggle | On |
| Occluder Transparency | Slider | 20% (10–50%) |

## Acceptance Criteria

> **Note**: Tolerances are ±10% of target duration or ±1 frame at test capture rate,
> whichever is greater. "Within one frame" means ≤33ms at 30fps / ≤16.7ms at 60fps —
> the AC passes at both rates. All timing values reference tuning knobs by name.

### Projection (CR-1)

- **GIVEN** any state at any zoom, **WHEN** the camera renders, **THEN** view matrix encodes pitch = 30° ± 0.1°, yaw = 45° ± 0.1°, roll = 0° ± 0.1°, projection = orthographic. Verified by matrix dump.

- **GIVEN** Close zoom (O=4u), **WHEN** the camera renders, **THEN** the ground plane is visible to the viewport bottom edge — no void strip. The bottom 5% of the viewport contains rendered ground geometry. Repeat at Default and Far.

- **GIVEN** 16:9 aspect, **WHEN** window resized to 21:9, **THEN** ortho half-height unchanged (vertical preserved), horizontal extent increases proportionally. Verified by extents readout.

- **GIVEN** window resize mid-zoom-transition, **WHEN** the resize completes, **THEN** ortho extents immediately re-applied from current interpolated O. Verified by extents readout on the frame after resize.

- **GIVEN** unsupported aspect < 16:10, **WHEN** resize completes, **THEN** camera maintains 16:10 minimum (letterboxing or clamping).

### Zoom

- **GIVEN** game starts, **WHEN** first Exploration frame renders after world load, **THEN** camera in Exploration at Default zoom (O=10u), centered on party centroid within 32 screen pixels. Waived if centroid within one viewport half-extent of region boundary.

- **GIVEN** player scrolls wheel up at Default, **WHEN** wheel processed, **THEN** O transitions monotonically from 10u to 4u within `zoom_transition_duration` (300ms), no overshoot. Wheel up at Close has no effect (< 0.05u change over 500ms).

- **GIVEN** player scrolls wheel down at Default, **WHEN** processed, **THEN** O transitions to 20u within `zoom_transition_duration`. Wheel down at Far = no effect.

- **GIVEN** Extra Close Zoom enabled, **WHEN** player scrolls up at Close, **THEN** O transitions to 2.5u. When disabled, wheel up at Close = no effect.

- **GIVEN** rapid zoom input during transition, **WHEN** transition in flight, **THEN** at most one pending level queued. Intermediate levels skipped. Last-input-wins for 3+ detents.

- **GIVEN** player at Close (exploration), **WHEN** enter Conversation then exit, **THEN** camera returns to Close (last Exploration zoom), not Default.

- **GIVEN** game saved at Far zoom, **WHEN** loaded, **THEN** camera at Far at saved position (not party centroid).

### Conversation

- **GIVEN** dialogue with NPC at ≤ 10u, **WHEN** conversation starts, **THEN** camera frames PC+NPC within inner 90% of viewport (inset = 5% of viewport height per side). Uses last Conversation zoom if both fit; minimum zoom that fits otherwise.

- **GIVEN** conversation active, **WHEN** player pans/zooms, **THEN** camera responds without interrupting dialogue. Manual adjustment prevents re-frame on speaker change for `conversation_override_window` (5s).

- **GIVEN** conversation active, speaker changes, **WHEN** new speaker within inner 70% of viewport, **THEN** no re-frame occurs.

- **GIVEN** conversation starts at Far zoom, **WHEN** conversation triggers, **THEN** camera zooms to last Conversation zoom + pans to PC+NPC midpoint over `conversation_snap_duration × 2` (400ms).

- **GIVEN** conversation ends, **WHEN** dialogue closes, **THEN** camera → Exploration at last Exploration zoom within `zoom_transition_duration`.

### Combat

- **GIVEN** combat starts with 4 party + 6 enemies in 18×18u box, **WHEN** Combat activates, **THEN** camera frames all combatants within inner 90% at player's Combat zoom (minimum Default) within `combat_framing_speed` (40 u/s).

- **GIVEN** combat starts with 20 combatants in 60×60u box, **WHEN** Combat activates, **THEN** camera frames at Far zoom (doesn't fit at Default). Party + nearest 75% of enemies visible; remainder get edge markers.

- **GIVEN** player manually pans/zooms during combat, **WHEN** turn boundary occurs, **THEN** camera does NOT re-frame. Manual input suppresses turn-boundary re-frames for the encounter.

- **GIVEN** player has NOT manually adjusted, **WHEN** new combatants join mid-encounter, **THEN** camera re-frames on next turn boundary (or immediately if new count > current framed count). Overrides manual flag; player's zoom preserved.

- **GIVEN** combat ends, **WHEN** Combat exits, **THEN** camera → Exploration at current zoom/position within `zoom_transition_duration`.

### Pan & Manual Control

- **GIVEN** player drags with pan button ≥ 100ms, ≥ 8 logical px, releases, **WHEN** release occurs, **THEN** camera holds position (drift < 0.1u over 500ms). No auto-return. World-space point under cursor at drag start stays under cursor within 2 logical pixels.

- **GIVEN** player presses Home, **WHEN** key processed, **THEN** party centroid projects to within 0.5u of camera focus on first rendered frame after input. No intermediate positions. Zoom unchanged.

- **GIVEN** player presses End with character selected, **WHEN** key processed, **THEN** camera centers on selected character within one frame. No selection → no-op.

- **GIVEN** camera at region boundary, **WHEN** player pan-beyond, **THEN** camera clamps such that viewport shows no void (using AABB of 45°-rotated viewport rect). Automated check: all four edges + corners at 16:9 + 21:9.

- **GIVEN** cursor over UI panel, **WHEN** at screen edge, **THEN** edge scroll does NOT fire. Keyboard/drag/gamepad pan functional.

### Camera Shake

- **GIVEN** critical hit, **WHEN** hit connects, **THEN** lateral offset along screen-horizontal axis achieves peak of 1.27% ± 0.2% of shorter viewport dimension at the 8 Hz sine's first crest with linear falloff, decays to zero by 200ms (±20ms). Camera position = deterministic replay with shake disabled within 1 logical pixel from 200ms after start.

- **GIVEN** environmental collapse, **WHEN** event fires, **THEN** offset achieves peak 4.8% ± 0.3% with ease-out falloff, decays to zero by 800ms (±40ms). No achieved peak exceeds 5.0% — the hard cap.

- **GIVEN** Reduce Camera Motion = Disabled, **WHEN** any shake event triggers, **THEN** offset = zero.

- **GIVEN** Reduce Camera Motion = Reduced, **WHEN** shake triggers, **THEN** achieved peak = 50% ± 5% of standard, duration = 50% ± 10% of standard.

- **GIVEN** conversation active, **WHEN** any shake triggers, **THEN** offset = zero regardless of Reduce Camera Motion.

- **GIVEN** two shake events overlap (critical hit 200ms + explosion 300ms, 50ms apart), **WHEN** both active, **THEN** offset = signed offset with largest magnitude (not sum). Sign preserved.

- **GIVEN** shake at region boundary, **WHEN** offset would show void, **THEN** amplitude reduced for that frame. Waveform phase continues.

### Occlusion

- **GIVEN** structure's screen-space AABB overlaps party screen rect and structure between party and camera in depth, **WHEN** occlusion detected, **THEN** structure fades to `occluder_fade_alpha` (20%) within `occluder_fade_duration` (150ms). Verified via fade-alpha debug readout (not pixel-sampling).

- **GIVEN** structure in faded state, **WHEN** occlusion clears, **THEN** structure stays faded ≥ `occluder_fade_min_hold` (300ms), clear must persist ≥ `occluder_fade_min_clear` (300ms) before restore. Walking-alongside-wall test: no alpha cycle faster than one per 2s.

- **GIVEN** party enters interior, **WHEN** interior is active region, **THEN** roof/ceiling geometry above party elevation (tags "roof", "ceiling") absent from render (draw-call log). Interior entities visible at standard angle. Roofs restore within 150ms on exit.

- **GIVEN** interactable occluded, **WHEN** Tab pressed, **THEN** 2px solid outline in parchment `#E3D5B0` (±8/255), unlit, visible through geometry.

- **GIVEN** combatant occluded during combat, **WHEN** in frustum, **THEN** outline visible through geometry: solid (friendly, amber `#D9A441`), dashed (hostile, red `#C4483A`). Shape distinguishable by protanopia-simulating tester at 2px. Always active — no key press.

- **GIVEN** entity both combatant and interactable during Combat, **WHEN** Tab held, **THEN** combat silhouette wins (hostile dashed red / friendly solid amber); parchment suppressed.

- **GIVEN** faded structure, **WHEN** alpha < 50%, **THEN** shadow draw instances deactivated. Re-enabled when alpha > 95% on restore. Verified via shadow-pass draw-call log.

### State Machine

- **GIVEN** aggro'd enemy approaching, player initiates dialogue, **WHEN** combat starts mid-conversation, **THEN** Combat preempts Conversation within one frame. Combat ends → return to Conversation (one-deep stack).

- **GIVEN** journal open during combat, **WHEN** journal opens, **THEN** camera stays in Combat framing, combat UI visible behind journal. Journal close → combat framing resumes.

- **GIVEN** journal open, **WHEN** closes, **THEN** camera returns to base state active before journal opened (Exploration, Conversation, or Combat) — NOT unconditionally Exploration.

- **GIVEN** player crosses region boundary, **WHEN** trigger fires, **THEN** fade to black (300ms) → reposition → fade in (300ms). Movement/camera/interaction inputs ignored; menu navigation functional. Fade overlay in screen space (UI layer).

- **GIVEN** player pans during conversation, **WHEN** dialogue ends, **THEN** camera → Exploration (not stranded in Manual).

- **GIVEN** player pans during combat, **WHEN** combat ends, **THEN** camera → Exploration at current zoom/position.

### Off-Screen Markers

- **GIVEN** party member outside viewport, **WHEN** any camera state active, **THEN** edge marker at screen edge nearest entity position — solid chevron, amber `#D9A441`, display name.

- **GIVEN** hostile combatant outside viewport takes action, **WHEN** action fires, **THEN** marker flashes (200ms pulse) + action label adjacent for 2s.

- **GIVEN** ≥3 markers same edge within 40px, **WHEN** rendered, **THEN** merge into cluster (chevron + count badge). Expand on hover/focus.

- **GIVEN** Reduce Camera Motion = Disabled, **WHEN** markers rendered, **THEN** markers static (no pulse). Action flash suppressed.

### Performance

Reference conditions: release build (`editor`/`debug` tags OFF), 1920×1080 and 3440×1440 (21:9), reference hardware TBD (technical-director) — **BLOCKED until hardware pinned**. Fixture: combat AC fixture (4 party + 6 enemies in 18×18u) + 500 environment entities including 50 occlusion-eligible structures.

- Camera update (state machine + focus + follow + shake + marker data): ≤ 1.0ms p95 CPU over 3000-frame capture. Measured with Go benchmark instrumentation. Allocation-free (verified via `ReportAllocs`, zero allocs per frame).

- Whole-frame render: ≤ 16.6ms p95 at 1920×1080, ≤ 22.2ms p95 at 3440×1440, over 3000-frame capture including: idle exploration, party follow at walk speed, combat (6 silhouettes + 3 faded structures), conversation, one zoom transition (Default→Far→Default). CPU and GPU times reported separately. No GC/pause spikes > 33ms during any camera state transition.

- UI panels at correct screen-space positions during shake — bounding boxes identical shake on vs off. Automated screenshot comparison.

- Occlusion detection: zero spatial-index range queries per frame. Structure list built at region load. Verified via profiler instrumentation.

- Camera memory: ≤ 1MB total (position cache, shake ring buffer, fade state map with entity-destroy cleanup, occlusion structure list). Verified via heap profile.

### Accessibility

- **GIVEN** first launch (fresh profile), **WHEN** first frame renders, **THEN** accessibility prompt displayed: Reduced Camera Motion + Subtitles, keyboard/gamepad-navigable, no timeout. Escape/Skip = accessible defaults (Reduced + subtitles ON). Main menu visible/functional behind prompt.

- **GIVEN** Reduce Camera Motion = Disabled, **WHEN** relaunched, **THEN** setting persists from profile.

- **GIVEN** Reduce Camera Motion = Reduced (first-launch default), **WHEN** any state, **THEN** zoom transitions = instant snaps, no conversation re-frames after initial, combat framing = snaps.

- **GIVEN** Extra Close Zoom enabled, **WHEN** player zooms in from Close, **THEN** camera transitions to O = 2.5u.

## Open Questions

| Question | Owner | Deadline | Resolution |
|----------|-------|----------|-----------|
| Does Kaiju Engine's renderer support orthographic projection natively? | Engine Programmer | Before implementation | ✅ RESOLVED: Yes. `NewStandardCameraOrthographic` exists. |
| Should the camera support a "screenshot mode"? | UX Designer | Before vertical slice | Open. `snapshot_camera.go` exists as potential building block. |
| Camera shake accessibility | Accessibility Specialist | Before vertical slice | ✅ RESOLVED: `reduce_camera_motion` ladder, default Reduced. |
| Spatial audio listener API gap | Engine Programmer | Before vertical slice | Open. C layer has it; Go wrapper needs extension. |
| Entity Framework wrap decision | Technical Director | Before implementation | ✅ RESOLVED: EF wraps Kaiju entities. Camera reads Transforms directly. |
| Silhouette/outline rendering passes | Technical Director + Engine Programmer | Before vertical slice | Open. New engine infrastructure required. |
| Ghost pass rendering | Technical Director + Engine Programmer | Before vertical slice | Open. Blend-enabled pipeline variant + per-instance alpha needed. |
| Roof semantic part tagging | Technical Artist + Producer | Before first asset import | Open. Asset-scene-pipeline GDD must define tagging convention. |
| Shadow frustum scaling (50×50u < Far zoom 80u) | Engine Programmer | Before vertical slice | Open. Far-zoom shadows pop at region edges. |
| Movement system follow-speed constraint | Game Designer | Before camera tuning | Open. `follow_speed_exploration` lower bound ≥ 1.2× party walk speed. |

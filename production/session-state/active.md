# Session State

*Updated: 2026-08-10*

## Session Extract — /story-done 2026-08-10
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-002-component-model.md — Component Model
- Tech debt logged: None
- Next recommended: EF-003 (Spatial Index) — ready-for-dev, no blockers

## Session Extract — /story-done 2026-08-10 (#2)
- Verdict: COMPLETE WITH NOTES
- Story: production/epics/entity-framework/story-003-spatial-index.md — Spatial Index
- Tech debt logged: None
- Next recommended: EF-004 (Write Arbitration) — ready-for-dev, no blockers

<!-- CONSISTENCY-CHECK: 2026-08-09 | GDDs checked: 20 | Conflicts found: 3 (2 known/tracked, 1 registry fixed) | Report: docs/consistency-failures.md -->
<!-- STATUS -->
Epic: Art Direction
Feature: Art Bible
Task: Section 2b (Shape Language) written — Sections 1–10 + 2b complete; engine values pending technical-artist merge
<!-- /STATUS -->

## Current Task

- **Task**: Art bible Section 2b (Shape Language) — inserted between Section 2 (Color Palette) and Section 3 (Lighting & Atmosphere), written to file 2026-08-09
- **File**: design/art/art-bible.md (Sections 1–10 + Section 2b complete; Section 8 engine values pending technical-artist merge)
- **Status**: COMPLETE — inserted as "2b", no renumbering (34 "Section 3–9" + 138 "N.M" subsection references make a renumber sweep high-risk; user-sanctioned "2b" option chosen). 6 subsections (silhouette-first read, three shape families, environment geometry, hero vs supporting shapes, UI shape grammar, character silhouettes); every subsection anchored to Section 1 + a game pillar; header status line updated

## Art Bible Progress

- [x] Section 1 (Visual Identity Statement) — one-line rule + 3 pillar-anchored principles + cross-cutting readability constraint
- [x] Section 3 (Lighting & Atmosphere) — mood targets per game state, user warmth directive accepted
- [x] Section 2 (Color Palette) — user-accepted forks: divergent UI register; palette cools with depth; moss = reclamation, health = amber
- [x] Section 4 (Character Art Direction) — 6 topics, written to file 2026-08-09
- [x] Section 5 (Environment & Level Art) — 6 topics, written to file
- [x] Section 6 (UI Visual Language) — 6.1 register split (world/desk/field), 6.2 typography (3 faces), 6.3 iconography (pen-and-ink), 6.4 motion language, 6.5 color application, 6.6 combat HUD, 6.7 accessibility & input contract + cross-GDD resolution record; written to file 2026-08-09
- [x] Section 7 (VFX & Particle Style) — 6 topics, written to file 2026-08-09
- [x] Section 8 (Asset Standards) — 8.1 formats, 8.2 naming, 8.3 texture tiers, 8.4 LOD, 8.5 export philosophy, 8.6 KitBash3D integration; written to file 2026-08-09; engine values pending technical-artist merge (marked in subsections)
- [x] Section 9 (Style Prohibitions) — 8 prohibitions (never/why-cited/instead), written to file 2026-08-09
- [x] Section 10 (Reference Direction) — 5 curated references (what-to-take / what-to-avoid / why-it-matters), additive rule, written to file 2026-08-09
- [x] Section 2b (Shape Language) — inserted between 2 and 3 (no renumbering; 34 Section refs + 138 subsection refs across 3–10 make renumbering high-risk); 6 subsections written to file 2026-08-09

## Section 6 Key Decisions

- **Three registers, one vocabulary**: desk (parchment overlays, world dims 30%), field (combat HUD, panel umber base), world (anchored UI, scales 4/10/20u, Pillar 3 lighting discipline). Thread-red line crosses registers — journal connectors and combat targeting line are the same visual.
- **Typography**: desk serif (Garamond class, warm, OFL-licensed, never blackletter), combat face (tabular grotesque for all HUD numerals), world lettering (painted, never typeset). Reference 1920×1080, min 1280×720; world-anchored text follows 4.4 LOD, ≥14px combat face at Far.
- **Iconography**: pen-and-ink, one stroke weight, no gradients; shape carries meaning, color carries accent (2.5 built into the drawing style). Map markers = thread vocabulary pins; resolved = ink check (no green/amber).
- **Motion**: paper slides (300ms in/200ms out), thread draws itself, pulses rationed (amber = reward, oxblood = tension), desk never trembles (screen-space immune to shake), no bounce/3D. Typewriter = word-chunk reveal 45–60 cps.
- **Color application**: hover = amber edge (lamp), selection/links = thread-red, destructive = oxblood, disabled = desaturated "bleached" (saturation is life), focus ring = oxblood-on-parchment / parchment-on-umber ≥3:1 (amber fails on parchment), ivory rationed to empire items, text never amber-on-parchment (fails AA).
- **Combat HUD**: panel umber 85–90% base; health = faction-fixed hue (amber allies / oxblood enemies), umber/soot track, level by length + number, low-HP pulse outline, NO hue shift as HP falls; AP = amber; turn order bar = panel umber (not parchment), current turn amber rim; damage numbers oxblood + soot outline, crits amber flash; targeting line thread-red; flanking arcs = cold indigo (ratified into UI palette as combat spatial cue, shape+motion distinct from target rim); cover = neutral ink shield; End Turn amber; victory/defeat returns to desk register.

## Section 7 Key Decisions

- **Particle aesthetic**: air is a material and a witness (Principle 1) — vocabulary from nature/hearth only (embers, ash, dust motes, pollen, mist, fireflies); legend filter at effect scale ("breath of the old world," never plasma bolt); material truth rule (behavior = meaning; ghost glow is the only exception); three strata of air (present/empire/wild) with family exclusivity.
- **Vocabulary by source** (each inherits color semantics + motion signature + rarity): present = flickering, irregular, anchored to visible source (lantern = player's portable emitter); empire = metronomic breath, no flicker/falloff/source, activation = "old god waking," rationed (one breathing glow per scene); wild = slowest air, ground-hugging mist, fireflies warm pinpricks in indigo night; corrupted = stutter in the empire's evenness, oxblood tendrils in pale light, spawned one at a time; combat = brief/sharp/weighted, crit flash = amber (matches 6.6 damage-number flash) with 0 particles + hit-stop, death dissolve = motion signature (beast settles / construct rises in even column / corrupted falls as red ash staining ground).
- **Performance (first draft, validate Section 8)**: per-effect tiers 16–32 / 64–128 / ambient ≤128 / hero ≤512 (State 4 only); scene soft cap 2,000 / hard 3,000, combat 1,500; ≤40 emitters/scene, ≤24 combat; effects ≤8s; **billboard two-orientation decision** — camera-facing for verticals, ground-aligned for horizontals (mist quad facing camera = wall); 3D meshes only for hero; additive blending rationed (empire/fireflies may glow, wild alpha only, no post bloom); one shared soft-sprite atlas.
- **Animation principles**: drift/float/settle — nothing pops, bounces, or moves like a screensaver; motion verbs = biography (rise/fall/hang/drift/settle/breathe); irregularity is life, regularity is the tell (empire metronome, corrupted stutter); settling leaves evidence (ash/blood/scorch persist — Pillar 2); ambient loops imperceptible within 30s except empire's (regularity is the point).
- **Weather (VS contract)**: rain ≤1,000 streaks + 60 ripples (favors present — wet ground reflects lantern pools), fog 8–16 ground sheets (favors empire — raises pale fraction; never above waist height in combat), snow ≤800 flakes with horizontal drift (clue sheet — tracks are testimony); weather never makes the collision, revelation can clear it; per-state dial (near-zero State 2, desk dry).
- **UI particles**: the desk does not breathe — no particle systems in the flat register (2.4); three moments as one-cycle 2D beats: journal dust = none, level-up = amber swell (~1s), trust-increase = portrait amber pulse (per 6.4 pulse vocabulary); world-anchored HUD may breathe at empire tempo as 2D animation; exception = title screen world particles (painting in motion).

## Section 8 Key Decisions

- **Formats**: FBX preferred interchange (OBJ accepted simple, BLEND exported to FBX first); PNG preferred textures (lossless carries the 5.2 grade; TGA legacy); no DCC material files enter the engine — materials = texture sets + authored values, assembled via mapping table. DCC's own formats are source-only. Engine-native formats = technical-artist contract (pending).
- **Naming**: kebab-case `[category]_[name]_[variant]_[size|role].[ext]`; prefixes prop_/env_/char_/tex_/mat_/ui_/vfx_; channel suffixes (_albedo/_normal/_roughness/_metalness/_ao/_emissive) on shared base; _lod0/1/2 suffixes; zero-padded variants; pack origin in pipeline metadata, never in name.
- **Texture tiers**: hero 2048×2048 / standard 1024×1024 / background+tiling 512×512 (tilesets to 4096 atlas) / UI min 2× display resolution; PoT always; tier = ceiling; **empire exception = 16-bit PNG, never down-tiered** (banding protection for near-achromatic value shifts, 2.6); textures reduce before geometry (4.4 order); VRAM scene budget — scenery down-tiers before witnesses.
- **LOD**: characters 3 levels (20k/10k/5k hero; 10k/5k/2.5k NPC — first draft), props 2 (5k/1.5k), small scenery 1 (≤500 + cull), terrain dynamic; thresholds anchored to zoom registers at 7u / 15u midpoints, ±1u hysteresis; 4.4 contract enforced (accent never shifts, silhouette never LODs, reduction order fixed, hostile rim-light readability to LOD2).
- **Export philosophy**: DCC authors appearance (grade authored into albedo — never re-applied, no double-grading; correct color-space tagging), engine delivers (compression, mips/streaming, assembly, LOD/instancing, collision/navmesh); exported once, read many; neutral exports; rig/animation budgets = engine constraints, art contract holds regardless.
- **KitBash3D integration**: packs = quarry, treatment = mason; foundation assets never final content; normalization per pipeline GDD CR-2 (scale 1u=1m door reference, Z-up→Y-up, material mapping tables, scale variance authored not a bug); mapping table = mechanical home of 2.1 color exclusivity; legend filter at import (treat or reject); both packs through same treatment; >80% single-pack validator = mechanical ally of 5.5 no-50/50 rule; pack assets scenery-by-default, witnesses earned by authoring.
- **Merge items marked [Technical-artist values pending — to be merged]**: engine-native formats (8.1), 16-bit texture support + VRAM limits + texel density (8.3), poly budgets + LOD switch + terrain LOD (8.4), compression targets + mip/streaming + draw call budgets + animation constraints + collision mesh requirements (8.5). Resolves with them: 1.3 art-pipeline decision (2.5D vs 3D) and 7.3 particle budget validation.

## Section 9 Key Decisions

- **Purpose**: anti-scope-creep section — each prohibition names the thing never done, the rule it would break (cited), and the true version instead; new visual ideas are judged by "which rule does it keep?"; a rule-breaking addition requires amending this document in writing.
- **The 8 prohibitions**: (1) never sci-fi — legend filter (1.1) bans plasma/neon/holograms/displays/chrome; ban is on first-sight reading, the machine may live under the surface (Principle 3, 2.6); (2) never generic fantasy — no dragons/elves/glowing swords/fireballs/wizard robes; supernatural vocabulary closed to 4.1's four families; investigator = journal + modest blade, not-a-chosen-one (4.5); spells act "the way the world breathes" (7.1); (3) never horror/gore for its own sake — no jump scares/gratuitous gore/body-horror decoration; horror notes rationed (4.1, 7.2, 4.6); blood-on-ivory is always a clue (2.2); (4) never UI outside the flat register — no 3D UI, no diegetic HUD, no skeuomorphic buttons (6.4, 6.3); flat world-anchored HUD remains (6.1) — the ban targets UI that pretends to live inside the painting; (5) never procedural generation — composition is authored (5.6), witness budget is a budget (5.3), moss authored per-asset (5.2); "a seed is not a writer"; (6) never filler combat — no random encounters/respawns/XP grind; respawn contradicts Pillar 2 (world keeps the mark, 5.4) and the finite-violence effects contract (7.2/7.3); (7) never comedy/parody/meta — no self-referential jokes, fourth-wall breaks, ironic detachment; warmth = human moments (State 2, 4.3 idles), not jokes; (8) never pure black or pure white — #000000/#FFFFFF banned everywhere; soot #141210 warm (2.1), bone ivory #E9E6DC (2.6), ink #1C1813 (2.4); pure black = void not unknown, pure white = sterile not sacred; painterly-grade reason (5.2).

## Section 10 Key Decisions

- **Selection rule**: additive — five references, five distinct domains (decay / composition + corruption / scale + warmth + craft / golden register / mythic-tech concept); no two point the same direction; the legend filter applies to references as to assets (take what serves the borderland, leave the rest). Section 10 is the formal home of inline citations (Section 3 warmth anchors, 2.1–2.2, 4.3, 5.4).
- **Per-reference splits**: Dark Souls = decay as narrative + ruined grandeur (Principle 1, 5.4), avoid grimdark grey/opacity/dead register; Diablo 1–2 = isometric crime scene + blood-on-white corrupted sacred space (5.6, 2.2, 2.3 false collision), avoid murk/grime/horror-gore; LOTR = inhuman scale + nature reclaiming stone + warmth against dark + practical costume (5.1, 2.1, 4.1, 4.5), avoid cinematic desaturated grade/heroic framing/fantasy iconography; Drakensang: The River of Time = the golden painterly baseline register (5.2, State 1, warmth directive), avoid cozy safety where danger never lands; One Piece = Void Century concept only — tech read as myth, misunderstanding is the story (Principle 3, 1.1, Pillar 4, 2.6), avoid all style + sci-fi legibility (9.1).

## Section 2b Key Decisions (Shape Language)

- **Placement**: inserted as Section 2b between Section 2 (Color) and Section 3 (Light) — no renumbering of Sections 3–10 (34 "Section 3–9" + 138 "N.M" shorthand references would require a high-risk sweep; user-sanctioned "2b" option). All existing cross-references remain valid.
- **Silhouette is the first read (2b.1)**: identity from form precedes 4.2's affiliation order (accent = whose side) — reconciled, not contradicted (accent needs a figure to attach to). Silhouette test = every figure/landmark/witness identifiable in pure silhouette at 20u (redesigned, not rescued); glance contract = frame parses as grammar (ground/figure/threat/witness). Pillars 3 + 1.
- **Three shape families (2b.2)**: present = angular, weathered, hand-wrought, human scale; empire = curved-perfect, unbroken lines, perfect symmetry, inhuman scale; wild = organic, irregular, patient. Rules: families never mix casually (6.1 register test in form); the unbroken line vs the broken line is the present/empire distinction (continuity = history, Principle 2); **symmetry is the empire's register** — nothing living is symmetric (4.3's small asymmetries; 5.2 clean is alien); fairy-tale balance in form (perfect curve = beautiful + wrong). Pillar 4.
- **Environment geometry (2b.3)**: angularity dominates by area (present = default, warm registers of 2.3); the perfect curve is the event, rationed like ivory (one circle per scene = statement; second = revelation or lie, mirroring 7.2); buried geometry resurfaces as pattern (5.5 — geometry, never color, betrays the plan, Pillar 2); verticality split: present = heavy human verticals, empire = rare vast unbroken verticals (5.6 accents).
- **Hero vs supporting shapes (2b.4)**: the eye is drawn by anomaly before color; hero shapes draw the eye by being wrong — always empire, player, or living (2.1 exclusivity in form); supporting shapes recede by repetition; witnesses sit where shape grammar points (5.6 focal point); player = unique carried profile (lantern/journal/satchel, 4.4 amber figure); **hostiles are never supporting shapes** (shape sibling of 2.5's hostiles-are-always-lit). Pillars 4 + 3.
- **UI shape grammar (2b.5)**: the desk draws with the investigator's hand — no empire geometry in the desk (no perfect-circle chrome, no unbroken-line frames; desk's only circles are "the hand's periods" — dots, knots, pins per 6.3); desk shapes = present's angular hand-wrought family (6.2 typography same rule in letterform); shape carries meaning before color (6.3 glyph rule). Pillar 4.
- **Character silhouettes (2b.6)**: cross-references 4.1–4.4, no duplication — one silhouette per role; thumbnail readability is the isometric contract (Pillar 1); **asymmetry is life, perfect symmetry is the tell** ("the only perfect figure in the game is the thing that was never alive"); hostility survives silhouette to the final LOD (Pillar 3).
- **Header**: "Sections 1–10 complete, plus Section 2b (Shape Language)". Noted during authoring: Section 10 (Reference Direction) authored concurrently — no overlap with shape content.

## Resolution Record (coordinator + ux-designer rulings, 2026-08-09)

- Health bars: faction-fixed hue, umber/soot track, length + numeric readout carry level, pulse outline low-HP, no hue shift
- Flanking arcs: cold indigo (#3E4A63) ratified into UI palette as combat spatial cue; shape + motion distinct from thread-red target rim
- Typewriter: 45–60 cps word-chunk reveal; skip (click/Enter/Space/gamepad); auto-complete on new question; "Instant" accessibility option; 10–15s timeout → Error + Retry; Cancel during Processing
- Resolved marker: ink check on parchment chip — no green, no amber
- Player questions: ink text + amber name-chip (never amber-on-parchment)
- Floating text: soot outline (numerals) or parchment backplate (labels)
- Turn order bar: panel umber, not parchment
- Focus ring: oxblood/umber ≥3:1; tab order + arrow-key chip navigation
- Esc: end conversation in dialogue; pause menu outside
- Resolution: min 1280×720, ref 1920×1080
- Auto-scroll: suspends on manual scroll-up, resumes at bottom
- PENDING ux-designer confirmation: ivory usage (save metadata ink-on-parchment, slider thumb amber/ink), destructive buttons oxblood not thread-red

## Files

- `/home/danila/BG2/design/art/art-bible.md` — Sections 1–10 + Section 2b complete; Section 8 engine values pending technical-artist merge. Header status line updated
- `/home/danila/BG2/design/gdd/camera-system.md` — revised, awaiting re-review

## Open Questions / Next

- Art bible complete (Sections 1–10 + Section 2b) — remaining: Section 8 engine values merge with technical-artist; AD-ART-BIBLE gate sign-off recorded when gate runs
- GDD deltas to apply (owner: ux-designer): dialogue-ui.md (typewriter speed/skip/instant/timeout, Esc, auto-scroll), combat-ui.md (health bars, flank arcs indigo, turn order umber, floating text backplate), investigation-journal.md (resolved = ink check), main-menu-and-settings.md (focus ring, ivory usage, destructive buttons, resolution floors)
- 2.4 follow-up: UI palette table should gain a cold indigo row (ratified in 6.6) when palette section next touched
- Section 8 engine values pending technical-artist merge (marked in subsections); resolves with them: art-pipeline decision 2.5D vs 3D (1.3 production note) and 7.3 particle budget validation
- AD-ART-BIBLE gate sign-off recorded when gate runs (main session)
- Camera re-review: `/clear` then `/design-review design/gdd/camera-system.md`; then update systems-index.md + review-log

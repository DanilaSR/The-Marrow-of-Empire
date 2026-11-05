# Art Bible — The Marrow of Empire

*Created: 2026-08-08*
*Status: Complete — Sections 1–10 plus Section 2b approved at AD-ART-BIBLE gate (2026-08-10); Section 8 engine values pending technical-artist merge*

> **Art Director Sign-Off (AD-ART-BIBLE)**: APPROVED — 2026-08-10, all sections complete and consistent; open items tracked in-document (Section 8 engine values, 2.4 indigo row, pending ux-designer rulings)

---

## 1. Visual Identity Statement

### 1.1 One-Line Visual Rule

> **"Render it the way the legend remembers it — myth, never machine; beautiful enough to mourn, dangerous enough to fear."**

This single line carries two built-in tests that resolve the two axes of ambiguity in this game:

- **The Legend Filter (myth, never machine)** — Anything from the old empire's advanced age must be depicted the way a borderland storyteller would describe it: "a blade that drinks light," not "a plasma weapon." The *function* may be technological; the *appearance* must be mythic. If a prop, material, or effect reads as engineering, it fails the rule and gets redesigned.
- **The Fairy-Tale Balance (beautiful enough to mourn, dangerous enough to fear)** — When a scene drifts toward pure horror (grotesque, oppressive) or pure prettiness (picturesque, safe), this pulls it back to the tension point where the game's mood lives.

Spirit line (from concept brainstorm, kept for flavor): *"Every scene should feel like a dark fairy tale — beautiful and dangerous in equal measure, where the ruins whisper and the forest watches."*

### 1.2 Supporting Visual Principles

**Principle 1 — Everything visible is a witness.**
Every object in a scene knows something about the empire and can be read the way an NPC is questioned. A cracked mural is testimony; a child's toy in a collapsed nursery is an exhibit; the tool-marks on a fallen spire are a confession. Visual motifs repeat across people and places — a sigil on a shrine matches a sigil on a soldier's cloak — so the eye can connect fragments the way dialogue does.

- **Design test:** "When placing any object in a scene, ask what it knows — if the answer is nothing, it doesn't belong in the frame."
- **Pillar anchor — Pillar 1, Every Voice Holds a Piece:** The environment is the silent half of the witness list. The core fantasy — connecting fragments of testimony — extends to looking: the isometric camera hands the player the whole crime scene at once, and this principle guarantees the crime scene is actually full of evidence. It also reinforces the anti-pillar of hand-authored depth: every placed object implies a hand that placed it.

**Principle 2 — The world's skin is its memory.**
The look of the world is its history made visible, and it visibly records what the player does. Decay is stratified like rock: older wounds are overgrown, newer ones are raw. Paths wear where the player walks, fights leave ash, a lantern lit stays lit. Nature reclamation (roots cracking stone, moss veiling statues) is the world's slow memory; player marks are its fast memory.

- **Design test:** "When choosing how any surface looks, ask what history it wears and what the player did to it — if either is invisible, redesign it."
- **Pillar anchor — Pillar 2, The World Remembers:** Persistence that lives only in a journal is not legible persistence. The art language *is* the state vocabulary — growth stages, ash, blood, light, weather — that makes "actions echo" visible in the frame. The change must be legible.

**Principle 3 — The surface is the lie; the craft is the truth.**
The empire's technology reads as myth at first sight and as engineering only under study. Its peak is legend — so its relics appear as holy relics, oracle stones, impossible geometry — and the closer the player looks, the more the truth shows: tooling seams, calibration marks, function. The visuals lie the way history lies: mythic surface, mechanical truth.

- **Design test:** "When designing any relic of the old empire, ask: would a borderlander call this a miracle, and would an investigator find the machine beneath? If either answer is no, it's wrong."
- **Pillar anchor — Pillar 4, Earned Revelation:** The art mirrors the game's three layers of truth — surface (the legend), contradictions (seams that don't quite fit), hidden truth (the machine underneath). Revelation is earned with the eyes exactly as it is earned with questions; the mythic surface is the price of entry, the mechanical truth is the reward for attention.

**Cross-cutting constraint — Darkness never eats a decision (serves Pillar 3, Tactical Depth, Not Tactical Bloat).**
Not a competing principle but a hard limit applied to the other three: no atmospheric choice may reduce the player's ability to read a combat space — silhouettes, ground plane, threat, cover, line of sight.

- **Design test:** "If an atmospheric choice could make a player misread a tactical situation, the atmosphere must yield."
- Application: lantern light is both mood and gameplay color; unit silhouettes break against background value; terrain legibility beats texture fidelity.

### 1.3 The Synthesis — Gothic and Mythic-Tech

Every scene has two visual strata, and the gap between them *is* the game:

- **The present stratum — Gothic dark fantasy.** The borderland's own architecture and how it maintains, repurposes, and half-understands the ruins: shrines built into machine-halls, new gods carved over old ones, moss veiling statues. Stone, decay, nature reclaiming.
- **The empire stratum — mythic, never sci-fi.** The old age appears in materials that don't occur in nature (impossibly smooth stone, light-trapping metal that reads as "holy"), geometry that violates Gothic proportion (perfect circles, unbroken lines), and light that is pale where the present is warm. It reads as *inhuman* rather than futuristic.

**Light language:** warm gold = the living present (people, safety, the player); pale cold glow = the empire's remnants (dead, alien, beautiful); cold shadow = danger and the wild. Warm gold leads the world's default register — adventure soul. When warm and pale light meet in one frame — that is a revelation moment, and the palette should save that collision for it.

**Production note (not part of the statement):** the open art-pipeline choice (hand-painted 2.5D backgrounds vs. full 3D isometric, per game-concept.md) does not change this statement, but hand-painted backgrounds buy the most control over fairy-tale light, while full 3D buys the easiest visible world-state changes for Principle 2. Resolve with technical-artist when Section 8 (Asset Standards) is authored.

---

## 2. Color Palette

*The color system is the light language of Section 1.3 made into paint: warm gold is the living present, pale cold is the empire's marrow, cold shadow is the wild — and the collision of warm and pale is the revelation, reserved for it. Three structural rules govern every choice below:*

- **Saturation is life.** The present world is saturated and warm; the empire is bleached and cold. The more alive something is, the more color it holds.
- **Warmth leads.** Amber gold owns the default register of every scene — the adventure soul (Section 3 warmth directive). Coldness and pallor are events, never the baseline.
- **The families never mix casually.** Warm and pale share a frame only in State 4 (Revelation), per Section 3's hard rule. Outside it they are separated by territory, material, or time of day.

*All hex values are baseline references to be validated on target displays during Section 8 (Asset Standards). The roles are the contract; the hexes are the first draft.*

### 2.1 Primary Palette — Seven Colors, Seven Jobs

Each color has a narrative job, not an aesthetic one. Values are listed highlight / base / deep.

**1. Amber Gold** — `#F0C060` / `#D9A441` / `#A8742C`
- **Job — the living present.** Warmth, hearth, humanity, the player. The color of safety and of being alive. Leads the world's default register; everything the player loves is amber-lit.
- **Anchor:** Section 1.3 light language; Section 3 States 1, 2, 3, 6. **Exclusivity rule:** amber is never used on empire architecture or artifacts. When amber appears inside empire space it is an intrusion — and it is always the player or the living.

**2. Moss Green** — `#7C945F` / `#647A4E` / `#4A5A3A`
- **Job — nature reclaiming.** The world's slow memory (Pillar 2): roots crack stone, moss veils statues. Green is the land healing itself over the empire's corpse — beautiful, patient, watchful. Not "safe": the forest watches.
- **Anchor:** Principle 2 (the world's skin is its memory); LOTR nature-reclaims-stone; State 1's carrying element (sunlight on moss-veiled statues). **Semantic rule:** green never means "safe/heal" anywhere — health is amber (2.2).

**3. Deep Stone** — `#6B5F4F` / `#3E362E` / `#2A241E`
- **Job — the Gothic ground.** The borderland's own architecture: stone, timber, earth — the weight of ordinary life. The canvas the other colors sing against. Warm-tinted: even the present's stone is alive.
- **Anchor:** the Gothic stratum (Section 1.3); State 1's golden ruins. The palette's mid-value anchor for silhouette legibility (Pillar 3).

**4. Soot Black** — `#141210`
- **Job — the void between fragments.** Absence, the unknown, what the player has not learned yet. The empty spaces in the story made visible; the contrast anchor every other color is defined against.
- **Anchor:** Pillar 4 — the truth is earned out of the dark; State 2 ("the shadow holds what they have not said yet").

**5. Pale Ivory** — `#F2F0E8` / `#E9E6DC` / `#D8D4C8`
- **Job — the marrow.** The empire stratum's signature: bone-white, the inside of the dead thing. Reads as holy; is the empire's light — dead, alien, beautiful. Never appears on living things. The title of the game, given to a color.
- **Anchor:** Section 1.3 light language (pale cold glow); Pillar 4 — the marrow is the reward at the cold center. Full family in 2.6.

**6. Cold Indigo** — `#5A6A8C` / `#3E4A63` / `#232B40`
- **Job — the wild shadow.** Night, wilderness, the high places, the watchful edge. Danger's cold brother sets the mood, but never carries a threat alone (2.5): the wild warns with shape and motion; blue only sets the tone.
- **Anchor:** State 1 night ("cool indigo punctured by life"); the concept's "cold blue of high mountain shadow."

**7. Thread Red (Oxblood)** — `#C4483A` / `#A63A32` / `#6E2420`
- **Job — the thread and the wound.** Blood, urgency, connection. What binds: the investigator's thread, the web of knowing, the blood that makes a mystery worth solving. The only color whose meaning is identical in world and UI.
- **Anchor:** State 5's thread-red; Pillar 1 — the thread connects fragments; Diablo 1–2's blood-soaked sacred spaces. In UI it is the primary interactive color (2.4).

### 2.2 Semantic Color Vocabulary — What Each Color Means

Colors speak a language, and like the world's history (Principle 3: the surface is the lie, the craft is the truth) the language has two layers: **surface** — what a borderlander would say the color means — and **truth** — what it actually is. Both layers must be true at once.

| Color | Surface (what the world believes) | Truth (what it is) | Gameplay meaning |
|---|---|---|---|
| **Amber Gold** | "the fire that keeps us alive" | the living present; what the player protects | life, health, friendly, warmth — never on empire things |
| **Thread Red** | "blood and fire — the old wars" | the thread and the wound: what connects, what is at stake | damage, hostile, urgent; in UI: selection and thread links |
| **Pale Ivory** | "the holy light of the old gods" | the empire's marrow — dead, alien, beautiful | empire-significant objects (relics, oracles); rationed |
| **Moss Green** | "the land's health" | the world's slow memory; it outlives everything | nature-affiliated content; never safe/heal |
| **Cold Indigo** | "the cold beyond the fire" | the wild that watches | night, wilderness, atmosphere; never threat by itself |
| **Soot Black** | "nothing there" | what the player has not learned yet | absence, the unknown, the reveal's reward |
| **Deep Stone** | "the ground that holds us" | the present's own weight | structure, neutrality, the canvas |

Two composed meanings carry their own sentences:

- **Blood on white** — empire ivory stained with oxblood is the world's corruption signal: a witness to violence. Diablo 1–2's corrupted sacred spaces, translated: the holy thing that bled. Always a clue (Pillar 1: every surface knows something).
- **Dead gold** — gold that is not amber: tarnished, lustreless, greenish. The empire's only gold is a corpse of gold. Amber's exclusivity is the backbone of the system.

### 2.3 Per-Area Temperature Rules

Region names land in region GDDs; this section defines the grammar. Every area sits on one warm→cold gradient governed by three dials: **warm fraction** (how much of the frame amber owns), **saturation** (how much life-color the area holds), **pale fraction** (how much of the empire stratum surfaces).

**The depth rule:** temperature falls monotonically with depth. Deeper areas are never warmer than shallower ones — except for **settled refuges**: a hearth in the deep is allowed anywhere and is emotionally load-bearing. Warmth in the cold is precious, which is why State 2 (Interrogation) stays the warmest state wherever it happens.

**Why depth cools:** the concept's difficulty scaling ("deeper into the borderland = more dangerous") becomes visible — the palette is a danger telegraph (Pillar 3). And the empire promise: pale fraction rises with depth, so the marrow waits at the cold center (Pillar 4).

**Threshold rule:** crossing into a colder register is a visible event. Area boundaries shift hue and value in the first frames the player can see — the palette announces depth before the enemies do.

Area archetypes (5–7 regions planned; these are the temperature families, not final names):

| Area | Warm fraction | Saturation | Pale fraction | Character |
|---|---|---|---|---|
| **Borderland Settlements** (towns, farms, roads — the MVP region) | high | high | near zero | The warmest register: golden day, amber dusk, ochre stone, garden moss. Settlers keep the old things at arm's length. |
| **Old Forest / Wildwood** | medium (dappled) | high | low | Green-moderated warmth: gold becomes shafts and clearings; the shadows between are cold. The most beautiful greens and the coldest dark — the fairy-tale watchful edge. Empire leftovers show as pale stillnesses. |
| **High Places** (passes, the hermit's ridge) | low, concentrated | medium-low | low | The wild's cold: indigo and slate, thin light. A hermit's fire is an event. The empire never built here, or weather erased its works. |
| **Deep Ruins / Empire Strata** (mines, machine-halls, the buried city) | low, as pockets | low | rises with depth | The marrow zone: ivory, silver, celadon; saturation drains toward the achromatic. The only warm things are intrusions — settler lanterns, a shrine candle, the player's torch. The player carries the present into the tomb, and it shows. |
| **River Valley** | mirrors surroundings | mirrors surroundings | mirrors surroundings | Connective tissue: water doubles the register of its banks. A river town is warm twice over; a ruin by the water carries a pale reflection — the world's quiet way of teasing the collision without delivering it. |
| **Corrupted Sacred Space** (proposed area type) | low, sick | low | high, uneven | The two strata twisted together: ivory stained with oxblood, dead gold in the fittings, the glow uneven. The **false collision** — the world fakes revelation here, teaching the player to distrust cheap collisions, so the real one (State 4) is earned. The fake is noisy and stained; the real is still and clean. |

**The exclusivity rule, applied:** living amber never appears on empire material. If an empire surface must be gold, it is dead gold. When a player sees amber light on empire architecture, one of two things is true: the player is carrying it, or something living has moved into the ruin. Both read as sentences.

### 2.4 UI Palette — Divergent Register, Shared Vocabulary

**Decision:** the UI does not match the world's lighting; it uses its own flat, even register — State 5's deliberate break, so the mind can work. Reasons: readability across any world lighting; structural protection of the revelation collision (a separate register makes an accidental mini-collision impossible); the parchment-and-ink language is already canon. But the vocabulary is shared: every UI color means the same thing it means in the world.

| UI color | Hex | Role |
|---|---|---|
| **Parchment** | `#E3D5B0` | base surfaces; the desk |
| **Panel Umber** | `#2E2820` | dark panels, toolbars, the frame of the desk |
| **Ink** | `#1C1813` | text; ≥ 4.5:1 contrast on parchment (WCAG AA) |
| **Thread Red** | `#C4483A` | primary interactive: thread links, selection, current question |
| **Amber** | `#D9A441` | positive: health, rewards, friendly markers |
| **Oxblood** | `#7A2E28` | damage, hostile, danger states |
| **Pale Ivory** | `#E9E6DC` | rationed: empire-significant items only (relics, oracle fragments, empire sigils) |
| **Moss** | `#647A4E` | nature-affiliated content; never safe/heal |

Rules:
- World lighting never tints UI; the register is flat by design.
- Pale ivory in UI is the only empire-family presence in the interface, and it is rationed — an empire item on the desk is an event.
- Ivory and amber never combine in a single UI element (no mini-collisions).
- Game-critical states differ in value and shape, never in hue alone (2.5).
- The thread graph (State 5) is drawn in thread-red on parchment: the web of knowing as embroidery.

### 2.5 Colorblind Safety

**Hard rule:** no game-critical information anywhere — world or UI — is encoded by hue alone. Every semantic color carries a redundant cue: value, shape, icon, motion, or sound.

| Pair | Fails for | Backup cues |
|---|---|---|
| **Thread red vs moss green** | deuteranopia, protanopia | The most critical state never uses this pair — health is amber, not green. Where they coexist (hostile actors vs nature): value separation (oxblood is the palette's darkest chromatic family; moss is mid-light), shape (hostile actors are rim-lit and carry weapon silhouettes — Pillar 3). |
| **Pale ivory vs amber gold** | tritanopia; partially deuteranopia | The critical pair, because the revelation IS this pair. The empire glow is the only self-luminous pale light, and it is never still — it breathes and shimmers unnaturally (motion); it sits higher in value than any warm light (value); the revelation carries a sound cue (the world goes still, then hums); warm light is always anchored to a visible source while pale light has none. Outside State 4, the families are separated by geometry and texture, never blended. |
| **Cold indigo vs soot black** | tritanopia (dark blues read as black) | Cold shadow is atmosphere, never the sole carrier of threat; night scenes always hold warm pools (value pops); silhouettes break against backgrounds per Pillar 3. |
| **Oxblood vs deep umber stone** | protanopia | Threat markers carry icon and motion; hostile entities are always lit — rim light, fire accent, or blood-on-white composition — never dark-on-dark. |

**What the player never needs:** telling blue from purple, or reading a green/red health bar. Health is amber and damage is oxblood — a warm-dark pair that survives all three deficiencies.

### 2.6 The Empire Palette — The Marrow

The empire stratum has its own family, and the title of the game names it: the empire is *marrow* — bone-white, the inside of the dead thing, the reward at the cold center. The family is near-achromatic: **saturation is the present's privilege; the empire is bleached of life-color.**

| Color | Hex | Material |
|---|---|---|
| **Bone Ivory** | `#E9E6DC` | the light-trapping metal borderlanders call holy; reads as relic at first sight, seams under study (Principle 3: the craft shows in value shifts, not hue) |
| **Ghost Glow** | `#F5F3EC` | the only pale self-luminous light in the game; the empire's breath |
| **Silver** | `#C6CCCA` | moon-silver metal; liquid moonlight |
| **Porcelain** | `#9AA8AC` | the impossibly smooth stone that violates Gothic proportion |
| **Glaucous Deep** | `#5C6B72` | empire shadow; the cold dark empire light casts |
| **Dead Gold** | `#93865C` | tarnished, lustreless, greenish; the only gold the empire may wear — a corpse of gold |

**How the marrow contrasts with the present:**

| | Present | Empire |
|---|---|---|
| Hue | warm (amber, ochre, moss) | cold (ivory, silver, celadon) |
| Saturation | high — life | near zero — bleached |
| Key | mid-key; light falls from above (sun, hearth) | high-key; light comes from the material itself |
| Texture | rough, worn, patinated | impossibly smooth |
| Geometry | Gothic: irregular, hand-wrought | perfect circles, unbroken lines |
| Shadow | warm brown | glaucous cold |

Rules of the family:
- The families meet in one frame only in State 4 (Revelation). Anywhere else they are separated by territory (2.3), time of day, or material.
- Empire glow never behaves like fire: no flicker, no warm falloff — it breathes, slow and even.
- Warmth in empire space is always an intrusion, and always the player's or the living's. The player carries the present into the tomb, and the art direction never forgets it.
- The legend filter, in color: to a borderlander the ivory is holy; under study, tooling seams appear as near-imperceptible value shifts in the bone. The color lies the way history lies — mythic surface, mechanical truth (Principle 3).

### Quick reference — one line per concept

Amber = the living present, and it is exclusive; Moss = nature reclaiming, never "safe"; Deep stone = the ground, the canvas; Soot = the unknown, the not-yet-learned; Ivory = the marrow, rationed and reserved; Indigo = the wild, atmosphere not threat; Thread red = the wound and the web, the same color in world and UI. Depth cools the palette, and the empire promise deepens with it; the UI is parchment and ink — a deliberate break that still speaks the world's language; no critical state lives on hue alone; and the collision of amber and ivory happens exactly once per revelation, because the whole system saves it for that.

---

## 2b. Shape Language

*Before color, before texture, before detail — silhouette. The isometric camera shows the whole frame at once and nothing up close, and the eye's first pass on any scene is pure shape: what is a thing before it is a colored thing. This section makes shape a grammar as strict as Section 2's color grammar — three shape families, one hard rule (families never mix casually), one guarantee (the game reads in pure silhouette). The two strata of Section 1.3 are two geometries: the Gothic present is angular, weathered, hand-wrought, human-scale; the empire is perfect circles and unbroken lines at inhuman scale; the wild is organic, irregular, patient. The tension between the strata is the visual story, and it is told first in shape: the eye flags the empire as wrong before the mind names it alien.*

### 2b.1 Silhouette Is the First Read

**The eye reads identity from shape before it reads anything else.** The reading order of 4.2 — accent (whose side), silhouette (who they are), motion and stance (what they intend) — begins where this rule ends: the accent can only say whose side a figure is on once the silhouette has already said that it is a figure. For everything in the frame — a person, a ruin, a relic, a tree — identity comes from form, and form survives the conditions that kill every other cue: darkness, fog, snow (7.5), and colorblindness (2.5). Color is the second language, layered on top; it must never be the only language.

**The silhouette test.** Every figure, landmark, and witness must be identifiable in pure silhouette — no color, no texture, no interior detail — at its smallest readable register (Far zoom, 20u, per 4.4). A shape that needs its color to be recognized is a shape that fails in fog, at dusk, and for the colorblind — it is redesigned, not rescued. This is "darkness never eats a decision" (1.2) enforced on form, and it is why 4.4 orders that silhouette is never LOD'd: shape is the last cue standing, so it is the last cue to go.

**The glance contract.** The whole frame must parse on the eye's first pass — not every element, but the scene's grammar: what is ground, what is figure, what is threat, what is witness. The isometric frame is a diagram (5.6), and shape is what makes it a readable scene instead of a pattern. In combat (Pillar 3), the first read of a space is shapes: cover, height, threat, sightline. A frame that parses in silhouette parses in any weather.

> **Design test:** "Take any screenshot, remove all color, and ask: can you still tell what everything is? If the answer is no, the shape is wrong — color may add, but it never carries."

**Pillar anchors — Pillar 3, Tactical Depth, Not Tactical Bloat:** silhouette is the read that survives darkness, fog, and haste; the tactical map is drawn in shapes before it is lit in colors. **Pillar 1, Every Voice Holds a Piece:** the whole frame is evidence, and evidence must read as evidence at a glance — a shape the player cannot identify is a witness that never testifies.

### 2b.2 The Three Shape Families — Angular, Curved-Perfect, Organic

Three families, each with a sentence, each obeying Section 2's grammar in form:

| Family | Signature shapes | What it reads as | Where it lives |
|---|---|---|---|
| **The present** | angular, heavy verticals, broken lines, near-right angles, human scale | the mark of the hand — work, repair, life | the default frame: settlements, roads, the world's skin (2.3's warm registers) |
| **The empire** | perfect circles, unbroken lines, concentric inlays, perfect symmetry, inhuman scale | no human effort — miracle at first sight, machine under study (Principle 3) | the event: rationed by depth, like the pale fraction (2.3) |
| **The wild** | organic, irregular, asymmetric, patient | the world's slow memory (Pillar 2) | the frame: connective tissue, reclamation (5.5) |

**The present — angular, weathered, hand-wrought.** Cut stone, jutting gables, heavy verticals, irregular rooflines (5.1's skyline). The angle is the mark of the hand: no hand-wrought edge is perfectly straight, and joints, repairs, and weather break every line — the present's line is a line with history. Human scale: door frames, roof reaches, chimney heights — the body knows these shapes are for people.

**The empire — curved-perfect, unbroken, inhuman.** Perfect circles, unbroken lines, arcs with no joint. Nothing hand-wrought is perfectly round: the perfect curve is the shape of no human effort, which is why the eye flags it as wrong before the mind names it alien (1.3 — "geometry that violates Gothic proportion"). The empire's lines run unbroken where the present's are broken by mortar and weather, and the seam, when study finds it, is near-imperceptible (Principle 3 — the craft shows in the shape's continuity). Scale is the second tell: thresholds too wide, arcs that enclose a village (5.1's inhuman scale). The body knows before the mind does — this space was not made for people. **The fairy-tale balance, in form:** the perfect curve is beautiful enough to mourn and wrong enough to fear — one shape holding both halves of 1.1.

**The wild — organic, irregular, patient.** Roots, moss, vines, water-worn stone: nature's shapes, always asymmetric, always slow. The wild is the frame the other two families sit in — it covers the angle and the arc alike (5.2's overlay system is the shape system's brush), and its irregularity is the ground against which both present craft and empire perfection are read.

**Rules:**

- **Families never mix casually** — the shape version of Section 2's exclusivity. A curve inside angular stone is an event: it is either empire (and follows the empire's rationing, 2b.4) or organic (a root, a sagging beam — nature's curve). A shape that cannot name its family does not exist — the register test of 6.1, applied to form.
- **The unbroken line and the perfect curve are the empire's signature; the broken line and the near-right angle are the present's.** The difference between a present line and an empire line is continuity: joints, repairs, and weather are history made visible (Principle 2), and the empire has none.
- **Symmetry is the empire's register.** Nothing living is clean (5.2), and nothing living is symmetric: the present's buildings are nearly symmetric, its people carry 4.3's small asymmetries, and the wild is never symmetric. The empire's forms are perfect in their symmetry — the perfect circle is symmetry completed, the shape of something that was never alive.

**Pillar anchor — Pillar 4, Earned Revelation:** the geometry gap is the surface lie made visible. The empire's shape reads as miracle at first sight and as machine-precision under study, and the collision of the two geometries is what revelation resolves (Section 3, State 4). The synthesis of 1.3 is a shape sentence: "the gap between them is the game."

### 2b.3 Environment Geometry — The Dominant and the Event

**Angularity dominates by area; the perfect curve is rationed by scarcity.** The present's register is the world's default — settlements, roads, and forests hold most frames (the warm registers of 2.3), and the wild's organic forms are the connective tissue between them. The empire's geometry is the event, with the same economy as the pale fraction (2.3 — it rises with depth) and the same rationing as ivory (2.1): one perfect circle in a scene is a statement; a second is a revelation or a lie (2.3's corrupted sacred space fakes the event; the real one is State 4).

**The reclamation is a shape story.** Roots crack the angle and the arc alike (5.2), and buried geometry resurfaces as pattern — a ring of trees, a straight hedgerow, a clearing that is a perfect circle (5.5). The world's skin remembers (Pillar 2) in shape before color: **geometry, never color, betrays the buried plan.**

**Verticality belongs to the present; the vast unbroken vertical belongs to the empire.** The present's weight is in heavy, human-proportioned verticals — gables, chimneys, towers with joints and roofs. The empire's verticals are rare and unbroken: a dead tower, a colossal statue, a spire with no seam (5.6's vertical accents) — the skyline's shape events, marking the empire's height against the present's roofs (5.1). The difference is continuity and scale, never material alone.

> **Design test:** "Stand in any scene and name the three families: what is angular, what is curved-perfect, what is organic? If a frame cannot answer, it is a shape without grammar."

**Pillar anchors — Pillar 2, The World Remembers:** reclamation and resurfacing are shape memories; the world keeps its old geometry under its skin. **Pillar 1, Every Voice Holds a Piece:** geometry is testimony — "what made this shape?" is a question every circle in the wild answers (5.4's three questions, asked of form).

### 2b.4 Hero Shapes and Supporting Shapes — What the Eye Must Find

**The eye is drawn by anomaly before it is drawn by color.** In a frame of angular decay and organic growth, an unbroken arc is the one shape the eye cannot file away as background — the most complete shape in the field is the one the eye stops on, and in this world the shape that is most complete is the shape that does not belong. This is the shape version of the palette's collision (Section 2): **hero shapes are the figure; the world's angular and organic mass is the ground.**

- **Hero shapes draw the eye by being wrong.** The empire's perfect geometry is the game's visual hook: it reads as an event, an answer, a wound in the texture. The hero shape is always empire, or the player, or the living (2.1's exclusivity, in form): amber light in empire space is an intrusion and a sentence (2.1), and the unbroken arc in present space is the same sentence told by shape.
- **Supporting shapes recede by repetition.** The present's angularity and the wild's organic forms are designed to be scanned past — they repeat, they fill the frame, they are the ground the figure stands on. The witness budget (5.3) has a shape rule: witnesses sit where the shape grammar points — on diagonal axes, at the focal point (5.6), where the eye lands anyway — and supporting shapes never compete with the frame's one anomaly (5.6's one-focal-point rule, in form).
- **The player's shape.** The investigator's silhouette is deliberately un-heroic (4.5's test: "would anyone call this figure a warrior?") — but the carried shapes are unique: the lantern, the journal, the satchel give the player a profile no NPC shares, and at Far zoom the player is the moving amber figure (4.4) — the one warm color in the cold, and the one shape that holds its profile in any crowd.
- **Hostiles are never supporting shapes.** The hostile families (4.1) are hero shapes by duty: 2.5's "hostiles are always lit" has a shape sibling — a hostile silhouette that reads as background mass fails the hostile check (4.2) at any LOD. The rim light is the last cue at Far zoom (4.4); shape is the first.

**Pillar anchors — Pillar 4, Earned Revelation:** the anomaly is the reward for looking — the eye is drawn to what the story is about, and revelation (State 4) is the hero shape completing its sentence, the geometry collision resolving into truth. **Pillar 3, Tactical Depth:** hierarchy serves tactics — threat, cover, and the focal point must be the shapes the eye finds first, in any light.

### 2b.5 The UI's Shape Grammar — The Investigator's Hand

**The desk does not borrow the world's geometry.** The UI runs its own register (2.4, 6.1), and shape is part of the deliberate break: the desk draws with the investigator's hand — margin sketches (6.3), rectilinear paper, ink lines — never the empire's curves and never machine geometry.

- **No empire geometry in the desk.** No perfect-circle chrome, no unbroken-line frames, no cold arcs, no symmetry as design. The empire is never typeset (6.2) and never drawn as chrome: a perfect circle in the interface would be the empire's geometry in the player's own room (9.4's register ban, in form). The desk's only circles are the hand's periods — dots, knots, pins drawn small and imperfect (6.3's map markers): the difference between a dot and a perfect circle is the difference between writing and machining.
- **The desk's shapes are the present's shapes.** Hand-cut panels, pen-and-ink glyphs, slightly irregular line work — the angular, hand-wrought family (6.2's type is the same rule in letterform: a warm hand-set serif, never blackletter, never cold modern). State 5's order is arrangement, never machine geometry: the desk is where the truth is allowed to be orderly because the order is the player's own.
- **Shape carries meaning before color in the desk too.** Glyph shape IS the message (6.3: "a status is identified by its glyph before its color") — the 2.5 discipline in form: a burn is a flame line before it is anything colored.

**Pillar anchor — Pillar 4, Earned Revelation:** the desk is the player's own construction (Section 3, State 5); its shapes are the hand's because the order they keep is the player's, not the world's.

### 2b.6 Characters as Silhouettes — The Witness List at a Glance

Section 4 already contracts character shape in detail (4.1's party silhouettes, 4.2's cues, 4.4's LOD contract); this subsection states the shape principles behind them.

- **One silhouette per role.** The party contract of 4.1 is the witness list's shape rule: party members are told apart at Default zoom without a nameplate by height/width contrast, headwear, and carried shape. The archetype signatures of 4.1 are silhouette signatures first — a craftsperson's breadth, a soldier's block, a hermit's narrow layers — each an angular or organic profile that reads at a glance.
- **Thumbnail readability is the isometric contract.** At Far zoom the frame is a map (4.4, 5.6) and figures compress to accent-colored shapes; the witness list must still be distinguishable at thumbnail size, because Pillar 1's whole crime scene is visible at once — a player who cannot tell witnesses apart cannot connect their fragments. Two named figures that read the same at Far zoom are a shape error, not an art style.
- **Asymmetry is life; perfect symmetry is the tell.** 4.3's "small asymmetries" are the shape rule of the living: every living figure is slightly, naturally off-balance. The empire's wake is the only figure whose silhouette is perfect — smooth, symmetric, too regular (4.1's metronome, in form; 5.2's "clean is the alien state," in shape). A humanoid whose silhouette has no asymmetry is wrong in the way the empire is wrong.
- **Hostility survives silhouette.** The hostile check (4.2) is a shape test first: weapon posture — drawn vs. carried — must read in pure silhouette, and a hostile's profile must never be background-shaped (2b.4). The 4.4 contract holds: silhouette is never LOD'd, so the shape read survives to the final draw call.

**Pillar anchors — Pillar 1, Every Voice Holds a Piece:** the witness list is readable because its shapes are distinct — distinction is what lets the eye collect fragments. **Pillar 3, Tactical Depth:** hostile intent at a glance is a shape guarantee, in any light, at any LOD.

### Quick reference — one line per rule

Silhouette is the first read: form before side before intent, and the game must read in pure silhouette before color earns its place. Three shape families — the present is angular, weathered, hand-wrought; the empire is curved-perfect, unbroken, inhuman; the wild is organic and patient — and families never mix casually. Angularity dominates the frame, and the perfect curve is the event, rationed like ivory. Hero shapes draw the eye by being wrong; supporting shapes recede by repetition; hostiles are never background. The desk draws with the investigator's hand, never the empire's geometry. And the witness list is told apart by silhouette at thumbnail size: asymmetry is life, and the only perfect figure in the game is the thing that was never alive.

---

## 3. Lighting & Atmosphere

Mood targets by game state, built on Section 1: the light language (warm gold = living present, pale cold = empire remnants, collision = revelation), the fairy-tale balance, and the constraint that darkness never eats a decision.

**Warmth directive**: the borderland is a golden, lived-in world scarred by the fall. Adventure leads with warmth — the world invites you in — and danger follows behind. Reference anchors: The Drakensang: The River of Time for the painterly, inviting warmth; Dark Souls and Diablo 1–2 for the weight of the dark it pushes against.

### State 1 — Exploration (the borderland)

- **Emotion target:** Sunlit wonder with a watchful edge — the adventure soul. The world invites: golden ruins, warm forests, the sense that something worth finding is around the next archway. Curiosity with a chill, not a dread.
- **Lighting character:** Day is the signature hour: golden sunlight, warm shafts through canopy, moss and stone rendered in warm light, lush green. Warm-neutral baseline (sunlit amber) — not overcast grey. Dusk is the amber hour, the most beautiful time in the world. Night is cool indigo but punctured by life: lanterns, embers, lit windows — the dark is never empty of warmth. High ambient warmth by day; high local contrast (warm pools against cold dark) by night.
- **Descriptors:** ancient, golden, watchful, inviting, vast
- **Energy:** Measured and contemplative, with warm spikes of discovery and sharp spikes of danger.
- **Carrying element:** Sunlight on moss-veiled statues and crumbling archways — nature reclamation reads golden, not grey. Ruin silhouettes gilded at dusk on the horizon: witnesses you have not met yet.
- **Identity link:** The fairy tale with sunlight — beauty leads, danger follows. Warmth is the living present winning space back from the empire's cold memory and the wild's cold shadow.

### State 2 — Interrogation (questioning an NPC)

- **Emotion target:** Focused intimacy — the thrill of the hunt in a quiet room. Trust, weight, expectation. The most human moment in the game.
- **Lighting character:** The warmest state in the game, always. Hearth and candlelight at night; sun through a window, dust motes in the beam, by day. Tight pools of amber-gold, faces half-lit, corners falling to shadow (the shadow holds what they have not said yet). High local contrast, low ambient. Color temperature: warm (amber/ochre).
- **Descriptors:** intimate, charged, warm, close, expectant
- **Energy:** Low but coiled — silence is part of the conversation.
- **Carrying element:** The fire between you — literal and metaphorical. The NPC's face half in light, half in dark; the objects around them that are their witnesses (their tools, their home, their scars). The interrogation room is a crime scene you can read while they talk.
- **Identity link:** Pillar 1, Every Voice Holds a Piece — the game's warmth lives here, in the human contact. Talking to people is the safest and most alive thing in this world, and it should look it.

### State 3 — Combat (tactical encounters)

- **Emotion target:** Focused danger — the beauty of violence done carefully. Not horror, not chaos: a dangerous dance with clarity.
- **Lighting character:** Legibility first, always — and warmth helps, so combat by day is golden: high ambient sunlight, clear silhouettes breaking against sunlit ground and shadow. Combat by night is cool baseline with hot accents — fire, blood, the pale glow of empire relics entering the field — with contrast engineered for silhouette reading. Light sources ARE the tactical map (fire pools, lantern posts, burning ground). The clearest lighting in the game, day or night.
- **Descriptors:** sharp, taut, deadly, clear, electric
- **Energy:** High, but measured — turn-based tension, not panic.
- **Carrying element:** The light sources themselves as tactical terrain; the pale empire-glow marking a relic that is simultaneously treasure, threat, and question.
- **Identity link:** Pillar 3, Tactical Depth — "Darkness never eats a decision" holds unconditionally; the fairy-tale beauty must survive at full readability. Warmth is an ally of clarity here, never an excuse for murk.

### State 4 — Revelation (the moment fragments connect)

- **Emotion target:** Awe — the breath before understanding. Beauty and danger resolving into truth.
- **Lighting character:** THE collision. Warm gold and pale cold in the same frame — the only state where both light families share the image (per Section 1: the palette saves this collision for revelation). Because warmth is now the world's default, this moment's power comes from scarcity: the pale light floods the frame and the familiar gold answers it. Light behaves impossibly — myth, never machine. The game's highest contrast; a visual key change.
- **Descriptors:** mythic, still, luminous, immense, inevitable
- **Energy:** Zero-motion stillness — the world holds its breath.
- **Carrying element:** The revealed relic or oracle seen for what it is — surface falling away, craft showing (the seams, the calibration marks). The environment answers: weather clearing, birds falling silent, light bending in ways stone has no right to bend.
- **Identity link:** Pillar 4, Earned Revelation — the visual payoff is reserved and unmistakeable. Hard rule: this collision is never used anywhere else — not in menus, cutscenes, or title screens.

### State 5 — Journal / Threads / Menus (the investigator's desk)

- **Emotion target:** Ordered calm — the mind at work. The satisfaction of the web taking shape. Safe, deliberate, yours.
- **Lighting character:** Flat, even, neutral-warm — a desk lamp over parchment. Parchment tones, ink black, thread-red highlights. Low-medium contrast; no dramatic shadows. A deliberate break from the world's mood so the mind can work.
- **Descriptors:** quiet, ordered, papery, deliberate, warm
- **Energy:** Contemplative, zero pressure.
- **Carrying element:** The parchment-and-ink language; the thread graph as the player's web of knowing — the place where they feel smart.
- **Identity link:** Pillar 4 — the desk is the one place the truth is allowed to be orderly, because it is the player's own construction, not the world's.

### State 6 — Title screen (first impression)

- **Emotion target:** Invitation — a story about to be told, a road about to be walked. Melancholic promise with a golden horizon.
- **Lighting character:** Golden hour — a wide living river crossing gilded by a low sun; a campfire burns at the riverside, merchant ships are moored at the near bank, and the forest watches from the far shore. The world is alive and inviting, and it is waiting for you. The two-light collision is teased as a single static pale mist-patch in the far-forest shadow — unmoving, unresolved, atmosphere rather than empire material — and never delivered.
- **Descriptors:** storybook, vast, patient, golden, promising
- **Energy:** Still, expectant.
- **Carrying element:** The painted, storybook quality of the frame — the title screen is the fairy tale itself, before the game begins, and it is beautiful enough to make you want to step inside. The campfire is the warm center; the pale mist in the far forest is the quiet question the world hasn't answered yet.

### Layering note — weather and time as mood modifiers

Weather and day/night layer onto the base states rather than creating new ones — and because the warm register is the default, coldness becomes an *event*: fog muffles the gold and raises mystery, rain deepens melancholy, storm edges toward threat, clear night sharpens the cold. Seasonal change (later scope) shifts the world's base palette toward autumn gold richness, not toward grey. Each state must remain visually distinct through any layer — distinctness is the rule, mood is the variable.

### Quick reference — one line per state

Exploration = golden adventure by day, warm-pooled dark by night; Interrogation = the warmest hour, the human moment; Combat = sunlight clarity or fire-lit night; Revelation = the collision, the key change; Journal = the desk lamp, ordered calm; Title = the gilded storybook frame.

---

## 4. Character Art Direction

*People are read the way the world is read. The color grammar of Section 2 becomes a clothing grammar — accent says affiliation, saturation says vitality — and the readability discipline of "darkness never eats a decision" applies to bodies first: no character design may make a player misread intent. Four reading layers, in the order the eye takes them at any zoom: accent color (whose side), silhouette (who they are), motion and stance (what they intend). Faces are the fifth layer, and they are rationed — the interrogation camera of State 2 is where faces are allowed to do their work.*

### 4.1 Character Archetypes — The Witness List

**Design rule:** every character is dressed the way a crime scene is dressed — every garment and carried object is evidence an investigator could question. If a costume answers no question ("who are you, what do you do, what happened to you"), it fails Principle 1 and is redesigned. A character's look is a biography, not a wardrobe.

**The Present — townsfolk, craftspeople, soldiers, hermits, scholars, tavern keepers.** One-line signatures, each readable at Default zoom by silhouette plus carried shape:

- **Craftspeople** — aprons, soot, tools at belt; broadened by labor, hands always in use. Their shop is their portrait.
- **Soldiers of the borderland** — repaired uniforms, patched leather, their house sigil (4.6); disciplined stance that relaxes into weariness. They carry their duty and its cost.
- **Hermits** — layered salvaged cloth, weather-worn; stillness as a profession. They carry nothing shiny — the wild does not allow it.
- **Scholars** — ink-stained cuffs, books and bundles, spectacles on a cord; stooped from reading. They carry the weight of what they know.
- **Tavern keepers** — stained apron, keys, cloth over shoulder; the widest stance in the world — furniture that talks.

When the player interrogates any of them (State 2), their costume is the first exhibit.

**The Party — one silhouette per role.** Party members are told apart at Default zoom without a nameplate: contrast in height and width (no two share a build); distinct headwear (the fastest identity cue at isometric distance — a brim, a hood, a helm, a crown of braids); distinct carried shape (shield mass, staff height, blade line, lantern); distinct gait (weight, cadence, asymmetry). Final composition lands in character GDDs; the silhouette contract stands: if two party members can be confused in a Default-zoom screenshot, one is wrong.

**The Hostile — four families.** (1) **The wild's teeth** — beasts and wood-things: natural, never cartoonish, beautiful enough to mourn (fairy-tale balance). (2) **The blood-touched** — corrupted people and things, oxblood stains on faded base (the blood-on-white composition, 2.2) worn as a body. The game's horror notes — rationed. (3) **The empire's wake** — constructs and relics that still move: bone ivory, ghost glow, glaucous shadow, motion too smooth and too regular (2.6's breath, not fire). Not angry — they simply do not stop; the uncanny is their register. (4) **The fallen present** — bandits, cultists, deserters: human hostiles in the same warm register as everyone else, marked by what they've done — bloodied edges, brands, stolen finery worn wrong. Hostility is not a species; it is a choice — and it must be readable at a glance (4.2).

**Scale language:** the empire's wake and the wild's great beasts may exceed human scale; the larger the hostile, the more silhouette alone carries threat (Diablo 1–2's looming demons). Human-scale hostiles rely on the four-cue system.

### 4.2 Distinguishing Feature Rules — Reading a Figure at Isometric Distance

At the isometric camera, faces are unreadable and detail is a privilege of proximity. Intent must survive a glance. Four cues, in the order the eye takes them — **any hostile presents at least two hostile cues; every neutral suppresses all of them:**

**1. Accent color — whose side.** One accent per character (4.6): a small, consistent marker — sash, shield rim, eye-glow, brand, stain. The fastest read in the game, and never the only read (2.5).

**2. Silhouette + weapon posture — who they are and what they hold.** Identity from shape; intent from the weapon's state: *drawn* (in hand, edge presented, weight behind it) versus *carried* (sheathed, slung, or in use for work). A smith's hammer mid-strike on an anvil is work; the same silhouette with the hammer facing the player is threat. A hostile's weapon is always visible in silhouette — and hostiles are always lit (2.5: rim light, fire accent, or blood-on-white; never dark-on-dark).

**3. Motion signature — what they are doing.** Neutral motion is task-bound and cyclical — working, ambling, tending; the figure has a life that does not involve the player. Hostile motion is anticipatory — sharp starts, held tension, weight that tracks the player. The empire's wake moves on a metronome: no task, no rest, no startle — the wrongness is the tell.

**4. Stance — what they intend.** Weight forward, facing the player = intent; weight back, hands visible or occupied = indifference. A neutral facing away is neutral by definition; a hostile never turns its back.

> **The Hostile Check (design test):** "Remove every UI marker. If a player at Default zoom cannot tell this figure's intent at a glance — from accent, weapon state, motion, or stance — the figure is redesigned, not the UI." This is "darkness never eats a decision" (Pillar 3) and the 2.5 redundancy rule, enforced on bodies.
>
> **The Neutral Proof (design test):** "A neutral must never present a hostile cue by accident." A farmer carries the scythe over the shoulder, never presented; a hermit's staff is a walking stick, not a weapon. An idle animation that reads as a threat is a bug, not a choice.

**The wild's ambivalence:** moss never means safe (2.1). A moss-accented creature's allegiance is carried entirely by motion and stance — stalking wolf and grazing deer share a palette. The wild is a faction, not an alliance.

### 4.3 Expression and Pose Style — Posture-Readable, Face-Writable

**Style target:** grounded realism with legible intentions. Realistic proportions and materials (the LOTR register — people are people, not caricatures), painterly warmth in skin and cloth (Drakensang: The River of Time), and pose clarity that may exceed anatomical naturalism when readability demands it — the world is a painting, and a painted gesture can be clearer than a real one. Never stylized to cartoon; never stiff to mannequin. The rig's baseline is a working human body with weight, weariness, and small asymmetries — nothing stands like a statue except the things that should (4.1.3).

**Posture over face at Default.** The body carries the acting: tension in the shoulders, weight in the stance, curiosity in the lean — the investigator's forward lean matters as much as any line of dialogue. Faces are realistic, animated with restraint — micro-expressions, not mugging; the grief must be in the eyes, but only when the camera can see them.

**Faces are rationed to Close zoom — and that is State 2.** The interrogation camera (4u) is where faces finally matter: half-lit by hearthlight (Section 3, State 2), the eyes, the tells, the scars the costume cannot hide. The game's most human moment carries the game's most expensive animation. At every other zoom, facial expression is decorative, never load-bearing — no critical information travels on a face the camera cannot read (the 2.5 discipline, applied to animation).

**Idle is biography.** Every character has a task-bound idle: the smith checks his hammer, the scholar rereads a margin, the deserter counts stolen coins. Characters who "just stand" are characters the world has forgotten — and this world remembers (Pillar 2). Motion economy follows: weighty, readable cycles; nothing floaty or bouncy in a world of stone and labor.

### 4.4 LOD Philosophy — Three Zoom Registers

The camera's three registers (ortho half-height 4u / 10u / 20u, per camera-system GDD) define what a character owes each distance. **Detail is authored for Close; readability is authored for Default; legibility is authored for Far.**

| Register | Camera | The read | What the design guarantees |
|---|---|---|---|
| **Close (4u)** | interrogation, inspection | the face, the hands, the evidence | Full material and face detail; ink stains and tooling seams visible — Principle 3 applies to people: up close, the officer's silver shows its patina, the cultist's sash its weave |
| **Default (10u)** | gameplay baseline | accent + silhouette | Faces simplify to value-blocked heads; costumes read as mass and color-block, not fabric; props keep their shape (a hammer stays a hammer) but lose texture. The contractual reads: accent answers "whose side," silhouette answers "who/what" |
| **Far (20u)** | tactical overview | accent + motion + stance | Figures compress to accent-colored figures in stance; faces vanish; props reduce to silhouette lumps. The hostility cues of 4.2 must still fire — motion signature and weapon posture survive every LOD |

**The LOD contract:** the accent color never changes across LODs — a range-shifted hue is a faction error. Silhouette is never LOD'd — shape simplifies last, and the weapon-posture read survives to the final draw call. Reduction order is fixed: texture → face → props → costume detail → silhouette (never). Every LOD is the same figure with less surface; switching is seamless by construction.

**The lighting guarantee at range:** 2.5's rule that hostiles are always lit (rim light, fire accent, blood-on-white) is a LOD rule too — at Far zoom, the rim light is the last cue that a figure in shadow is hostile. A hostile that reads as a dark blob against dark ground at Far zoom fails the contract and is redesigned.

### 4.5 The Investigator — the Player Character

Not a knight. Not a chosen one. **A determined person who walks into the dark asking questions, and carries the answers home.** Three commitments:

**1. Practical clothing that tells the truth.** Worn traveler's gear in the canvas colors — deep stone, soot, ochre — patched, mended, broken in. Nothing shines that was not earned or found: no inherited sigil, no family crest, no destined birthright. The not-a-chosen-one rule is a visual rule. Wool, linen, leather — the borderland's own materials — so the player visibly belongs to the present stratum (1.3).

**2. Tools of the trade — "I ask questions."** Signature props, readable at Default zoom:
- **The journal** — the thread-web made physical: worn, bristling with markers and scraps. The investigator's weapon is a book, and it is drawn as carefully as any blade.
- **The lantern** — the portable amber light. The investigator carries the living present into cold places; every amber glow in empire space is the player or the living (2.1), made flesh. The lantern is gameplay light, mood light, and signature: at night or in the deep ruins, the player is the moving amber figure — the Far-zoom read (4.4).
- **The satchel** — fragments, relics, papers, examination tools. The investigator collects witnesses (Principle 1); the satchel is the crime scene, carried.
- **A modest blade** — knife or short sword, carried as a tool (cord, crates) before it is a weapon. Present in the silhouette, never dominating it: this figure says "I walk into danger," not "danger walks into me."

**3. A posture of inquiry, not combat.** The investigator leans, touches, inspects; idles are always reading — a wall, a person, a ruin. In combat: ready, never heroic — weight low, scanning, one hand near the lantern or journal.

> **Design test:** "In a silhouette test at Default zoom, would anyone call this figure a warrior? If yes, it is wrong."

**Accumulated witnesses (later scope):** as the investigation deepens, the costume visibly accumulates — a patch, a relic-clasp, a new strap. The World Remembers on the player's own body (Pillar 2). Scope note: post-MVP visual-reward system; rig and costume are authored now to allow it without rework.

### 4.6 Color on Characters — the Accent Grammar

The palette of Section 2 is a clothing grammar. **Every character wears one accent — a small, consistent marker of affiliation — following the palette's semantic rules exactly.** The surface/truth double (2.2) applies to every accent: what a borderlander reads is what the character tells the world; what the investigator reads is the truth.

| Accent | Worn by | Surface (what the world believes) | Truth (what it is) |
|---|---|---|---|
| **Amber** | the investigator only | "the fire that keeps us alive" | the living present, the witness — the most warmly lit figure in any cold frame |
| **Ochre / deep stone** | townsfolk, craftspeople, the party — most of the present | "ordinary people" | the canvas; the world's background voices |
| **Silver + dead gold** | the empire's reach: order soldiers, clergy, officials | "authority, holiness" | the empire's marrow reaching for the living — the least saturated accents in the living world |
| **Moss** | hermits, wood-things, the wild | "of the land" | the land watches; moss is never safe |
| **Oxblood** | the hostile: bandits, cultists, blood-touched, corrupted | "bloodied, dangerous" | the wound is a threat — oxblood on faded base |
| **Bone ivory + ghost glow** | no living character | — | — |

**The ivory exclusivity, applied to people (decision for consistency with 2.1):** pale ivory never appears on living things — the hard rule holds. The empire's living servants wear the marrow's *shadow*: moon-silver and dead gold trims, glaucous undertones, cold bleached cloth. Their accent reads "pale, cold, wrong-metal" at a glance without breaking the exclusivity that makes ivory meaningful. The exception is the horror beat: **a living person who has taken true ivory into themselves is no longer fully living** — a priest whose skin is porcelain, a relic-bearer whose cloak is bone — the corruption signal from 2.2 told in one garment, and the game's rarest character note. Blood on white stays the world's corruption sentence: a corrupted construct is bone ivory stained with oxblood.

**Saturation is life on bodies too.** The present is the most saturated family, the empire's reach is bleached, the corrupted are faded-and-stained, and the wild keeps nature's mid-key. Dead gold is the only gold the empire's reach may wear (2.2) — a tarnished officer's gilding tells the same sentence as a tarnished shrine fitting.

**The sigil rule (Principle 1):** an empire sigil on a soldier's cloak matches the sigil on a shrine; a cultist's brand repeats in their lair. Accents are connective tissue — the eye links a person to a place the way dialogue links a testimony to a fragment. A familiar accent in a new place is the world whispering.

**Rationing:** ivory-family accents are events, not defaults (like ivory in UI, 2.4) — one ivory-accented character per scene is a statement; a crowd is a plot point. Amber beyond the player is rationed the same way: a second amber-carrying figure in the dark is either the living present or a lie worth questioning.

**Redundancy (2.5):** accent color is never the sole carrier of allegiance or threat — every accent is paired with the silhouette, motion, and stance cues of 4.2 and the UI marker layer (Section 6). The accent is the fastest read; it is never the only read.

---

**Production note (not part of the statement):** characters are bespoke — the two KitBash3D packs (Dark Fantasy, Enchanted) are environment packs — but costume material language should echo the packs' texture grammar (Gothic stone greys, worn leather, enameled metal) so figures sit in their environments as if built from the same quarry. Enemy roster specifics beyond the four families land in creature/enemy GDDs; this section sets the visual contract they must satisfy.

### Quick reference — one line per concept

Accent = affiliation, silhouette = identity, motion = occupation, stance = intent — a hostile presents two of these at once, a neutral presents none; ivory never touches the living: the empire's servants wear silver and dead gold, and true ivory on a body means it is no longer living; oxblood marks the hostile and the corrupted, blood on white is the corruption sentence worn as a body; moss accents mean "of the wild," never "safe"; the investigator is the only amber-accented figure — journal, lantern, satchel, modest blade, no inherited shine; faces are rationed to Close zoom and State 2, posture carries the acting everywhere else; detail is authored for Close, readability for Default, legibility for Far, and the accent never shifts a hue.

---

## 5. Environment & Level Art

*The environment is the silent half of the witness list (Section 1, Principle 1). Every scene is built from the two strata of Section 1.3 — the Gothic present and the empire's pale marrow — and the fixed camera (camera-system GDD: 30° pitch, 45° yaw, orthographic, no rotation) is not a limitation but a storytelling device: it hands the player the whole crime scene at once, the way a diagram hands a reader the whole machine. Composition is authored, never improvised; every object is placed, and placement is testimony. Four rules govern the section: the present builds with the empire's bones (5.1); materials are PBR underneath and painterly on top (5.2); density is a witness budget (5.3); and the world answers three questions or it gets revised (5.4).*

### 5.1 Architectural Language — The Present Builds with the Empire's Bones

**The borderland's architecture is reuse, reverence, and repair — never demolition.** People build with the empire's bones because the stone is good and the old gods are still feared. Rules:

- **Reuse veils, it does not destroy.** Empire stone is re-cut into walls, floors, and barns; but inscribed faces are never smashed — they are turned inward, plastered over, or carried into shrines. The borderland does not kill the old gods; it houses them. This is the fairy-tale balance in masonry: the old face is hidden, not dead, and the hiding itself is readable (Principle 3 — the surface is the lie).
- **New gods are carved shallow; old gods were carved deep.** Present carving is hand-sized, eager, worn by hands that pray; empire carving is patient, inhuman, precise. Mark-making is the history lesson — compare any two surfaces and the era is legible in the tool marks (Principle 3: the craft is the truth).
- **Shrines are the present's colonization of the machine-hall.** A candle-lit shrine inside a vast cold hall is the scale contrast that carries the whole game: human-sized warmth inside inhuman architecture — the settled refuge of Section 2.3 in miniature. A pocket of amber inside the cold, allowed anywhere, always load-bearing.

**The empire's architecture is geometry, scale, and material the present cannot fake:**
- **Perfect geometry** — perfect circles, unbroken lines, concentric inlays. Nothing hand-wrought is straight; the empire's geometry is the first thing the eye flags as inhuman (Section 2.6 contrast row).
- **Inhuman scale** — doors too tall, thresholds too wide, ceilings unreachable. The body knows before the mind does: this space was not made for people.
- **Materials that don't occur in nature** — porcelain-smooth stone, light-trapping ivory metal. **The empire does not weather; it is covered.** Nature reclaims it (moss veils statues), but beneath the moss the bone stays smooth — the empire's decay is the world's doing, never its own. The clean surface under the green is the difference between "old" and "alien."

**The isometric reading splits the strata by plane.** The 30° camera shows the ground plane and the skyline, and the game assigns each stratum the plane it reads best on:

- **The skyline is the present.** Roofs, chimneys, gables, banners, smoke — the Gothic silhouette is what the player sees of living architecture, and the present's irregular, hand-wrought skyline is its signature.
- **The floor is the empire.** Floor mosaics, perfect circles, inlaid geometry — the empire's testimony is written on the ground because that is the plane the camera shows best.
- **Rare vast verticals** — a dead tower, a colossal statue, an unbroken spire — punch through the skyline as scale landmarks (5.6) and mark the empire's height against the present's roofs.

- **Design test:** if a scene's empire element is a tall smooth wall with no ground-plane signature, it reads as nothing at this angle — add the floor pattern, or cut the wall.

### 5.2 Texture Philosophy — PBR Bones, Painter's Skin

**Decision: PBR base materials with a global painterly treatment.** Not full hand-paint (out of scope for an indie team working from KitBash3D's PBR catalogue) and not raw PBR (reads as asset-flip and fights the fairy-tale light). The painterly treatment is a four-part system applied to every asset from both packs:

1. **The grade** — a shared warm amber color grade (Section 2: warmth leads) that unifies every material toward painted-canvas warmth; the world's light is soft, golden, directional — never clinical PBR reference lighting.
2. **Authored material variation** — albedo and roughness pushed per-material toward "painted" values: warmer darks (soot black is warm, never muddy blue-grey), softer roughness response, brush-scale variation visible on present materials.
3. **The moss/decay overlay system** — a universal overlay stack (moss, roots, grime, wounds) that is the game's brush: it paints time onto every surface (Principle 2). The overlay is authored per-asset, never random.
4. **The two material regimes** — the contract every texture obeys:
   - **Present:** rough, warm, patinated, irregular; craft marks visible (tool marks, mortar, timber grain); high roughness; mid-key.
   - **Empire:** impossibly smooth; near-zero roughness; high-key; the only near-emissive materials in the game (ghost glow, Section 2.6). **Clean is the alien state — nothing living is clean.**

**Legend filter at texture scale:** no text that reads as modern, no standardized panels, no cables, no safety geometry. Technical detail is redesigned as ornament or mystery — and the craft shows only under study, as value shifts, never hue shifts (Principle 3). The empire's seams are near-imperceptible changes in the bone.

**Legibility over fidelity:** texture fidelity never beats silhouette (Section 1 cross-cutting constraint). Ground stays mid-value (deep stone family), actors break against background value, and no texture is authored at the cost of a tactical read (Pillar 3).

**Production note:** the 2.5D hand-painted vs. full 3D pipeline choice remains open until Section 8 (per Section 1.3's production note). This contract holds under either: in 2.5D the treatment is the backgrounds themselves; in 3D it is grade + overlays + materials. Both packs must pass through the same treatment or the world reads as two games.

### 5.3 Prop Density — The Witness Budget

**Density is a witness budget, not a decoration budget.** Principle 1's test runs at placement time: what does this object know? Objects that know nothing are scenery — allowed, in modest share, for the texture of lived life (market goods, household clutter, rocks) — but scenery never outnumbers or buries the witnesses.

**Two classes of prop:**
- **Witnesses** — carry testimony (5.4); placed with intent; visible from the camera's default angle (5.6); the eye must be able to find them.
- **Scenery** — the world's lived texture; placed to serve a scene's mood and the density dial, never to hide a witness.

**The density dial by area archetype** (Section 2.3 temperature families — density follows the same curve as warmth):

| Area | Density | Character |
|---|---|---|
| **Borderland Settlements** | high, ordered | Lived clutter, tools, household objects; witnesses hide in plain sight among familiar things. Human scale makes clutter honest. |
| **Wildwood** | medium-low | Props are terrain — rocks, roots, fallen trees; the few empire leftovers read as pale stillnesses by scarcity. |
| **River Valley** | medium, linear | Props string along the waterline; the river is the spine, the bank is the shelf. |
| **High Places** | very low | Thin air; every object is an event — a cairn, a hermit's fire, a snapped branch. |
| **Deep Ruins** | low, heavy | The empire's halls are clean of clutter; **emptiness is the empire's prop**. The few objects — and the player's own intrusions — carry enormous weight. |

**The combat threshold (Pillar 3):** density never rises above legibility. Sub-knee clutter may be dense (it does not hide actors); waist-height clutter is deliberate and reads as cover, placed on tactical logic; full-height props are rare and are hard cover. Sightlines at waist height stay clean in combat areas, always.

**The crime-scene corollary:** evidence clusters where events happened — a fight leaves a cluster (blood, dropped gear, scorch); an abandonment leaves an arrangement (a still-set table); the empire's fall leaves absence. The clean hall is itself a placement.

### 5.4 Environmental Storytelling — The World Is a Crime Scene

**Every scene answers three questions: what happened here, who was here, and what did they leave unfinished?** If a scene cannot answer, it is scenery and gets cut or revised until it can.

**Evidence grammar:**
- **Wounds age by overgrowth (Principle 2).** Blood fades and darkens; scorch grows moss; cracks root. **Freshness is a clue** — a wound with no overgrowth means recent passage. Older = more green; newer = more raw.
- **The interrupted moment.** A meal half-eaten, a cart abandoned mid-turn, a forge left hot. The frozen instant is the most legible witness — Diablo 1–2's abandoned settlements translated into placement: the story stops mid-beat and the player walks in on it.
- **Paths are testimony.** Desire lines converge on doors, wells, shrines; a worn path into the wild is a question. Paths wear where the player walks, and the world keeps the mark (Pillar 2).
- **Thresholds are the beats.** What sits at a door, gate, or bridge announces what the space guards — a threshold is where a scene introduces itself.

**Placement rules:**
- **Rule of three.** A major event gets three corroborating details — a mural, a blood trail, a dropped object — telling one story. Witnesses must agree, the way NPC witnesses must (Pillar 1).
- **Contradiction is a hook.** One detail that does not fit the room — clean boots by a bloody door, a locked shrine with a broken lock — is a thread (the environment's version of Principle 3's surface-lie). Contradictions are where Pillar 4 lives in the world: the player who looks twice earns the truth.
- **Camera-facing placement.** Testimony sits on the ground plane or camera-facing verticals (5.6). North-facing walls carry deep secrets, revealed only by events.
- **The player's marks.** Ash, blood, and worn paths persist after combat (Pillar 2). Player-inflicted marks may violate density rules — they are the player's own testimony, and the world keeps them.

### 5.5 Mixing the Packs — The Borderland Signature

**The signature image is a ruined archway swallowed by an enchanted forest** — the Dark Fantasy pack's bones wearing the Enchanted pack's skin. The two packs overlap constantly but never balance.

**Ratio rule: no 50/50 frames.** Every frame has a dominant stratum — either the ruin owns the frame with green as its garment (roughly 60–80% ruin, 20–40% green), or the forest owns the frame with ruins as pale stillnesses (the inverse). Balance reads as a collision, and the collision is reserved for State 4 (Revelation). The packs interleave; they never tie.

**Interleaving rules:**
- **Contact.** Enchanted props attach to Dark Fantasy props — ivy, moss, and roots grow on the ruin. One pack is the structure, the other is the skin, and they touch; a forest without a ruin in it and a ruin without green on it are both unfinished.
- **The reclamation gradient.** Green density runs on a gradient by territory: at the forest edge, ruins are half-drowned; deeper in the wildwood, they surface as isolated islands (the wild keeps its secrets); in settled land, green is clipped back — the present prunes nature, and work is legible.
- **Buried geometry resurfaces as pattern.** The empire's ground-plan betrays itself in nature's own growth — a ring of trees, a straight line of hedgerow, a clearing that is a perfect circle. The world's skin remembers the empire (Principle 2). It is geometry, never color, that betrays it — ivory never touches living things (Section 2.1 exclusivity).
- **Temperature blending.** Use the Section 2.3 dials per frame: a wildwood frame is dappled warm with a low pale fraction; a ruin frame is high warm fraction with pale as accent. Area boundaries shift hue and value in the first frames the player can see (threshold rule) — the register change is the transition zone.

**Asset rules:**
- Both packs pass through the same treatment (5.2) — same grade, same overlay system — or the packs read as two games.
- KitBash3D props are used as-is only if they pass the legend filter (Section 1.1). Anything reading generic-fantasy or modern-engineering gets treated (overgrowth, re-paint, modification) or is replaced.
- **Roots are the deep's one green.** In Deep Ruins, the only Enchanted-family presence is roots — the world's slow fingers reaching down to the marrow. Green in the deep is memory, not life, and it is permitted everywhere.

### 5.6 Isometric Composition — Composing for the Fixed Camera

The camera (camera-system GDD) is orthographic, fixed at 30° pitch / 45° yaw, unrotatable, with zoom registers at 4u (Close), 10u (Default), and 20u (Far), and roof removal / occluder fade as the occlusion policy. The GDD calls the frame a painting; this subsection says how it is painted.

**What the angle gives:** the ground plane as a stage — the crime scene laid out like a map, the game's diagrammatic gift to Pillar 3 (distance, cover, line of sight read at a glance); rooftops and skyline; silhouettes against ground value.

**What it hides:** interiors (roofs must be cut or faded); north-facing walls; anything under a permanent overhang; true height — a tower reads as a column and needs help to read as tall.

**Composition rules:**
- **Ground-plane-first.** Compose for the floor: paths and geometry lead the eye, and the floor carries the empire's testimony (5.1). A scene whose story lives on walls alone is a scene that fails at this angle.
- **Diagonal axes.** The isometric grid has two strong diagonals (the 45° yaw). Major paths and sightlines run on them, converging on the focal point — the eye follows the diagonal the way it follows a thread (State 5's thread, drawn in space).
- **One focal point per screen.** Each view has a single visual anchor — a witness object, a threshold, a destination — with supporting details radiating from it. The eye lands on the anchor within a moment or two, then reads outward.
- **Vertical accents.** Punctuate the skyline with rare tall verticals — dead towers, colossal statues, unbroken spires. They are the scale landmarks that make the empire's inhuman height readable against human-height props, and a scene without one flattens.
- **The foreground frame.** Darkening, mist, or a foreground silhouette (a tree, an arch fragment) at the frame's edge pushes depth into a flat projection and pulls the eye inward. The frame edge is the camera's lens, and the painterly grade (5.2) owns it.
- **The north-wall rule.** The camera naturally hides north faces. Easy testimony goes to the ground plane and camera-facing walls; deep secrets live on hidden faces and are revealed only by events — a wall collapsing, a roof opening, the camera panning in a revelation (Pillar 4). Reaching the hidden face is an act, and the act is the reward.
- **Register-aware authorship.** Author composition for 20u (map reading: big silhouettes, low density), craft for 4u (Principle 3's seams visible at Close zoom — the craft is the close-zoom reward), balance for 10u (the default). No critical information exists at only one register, and the Extra Close register (2.5u, accessibility) must never be required to read a clue.
- **Occlusion discipline.** Design interiors with roof removal in mind: testimony on floors and camera-facing walls. Occluder fade (camera GDD, CR-8) is the last resort, not the plan.

- **Design test:** "Can the player find every witness from the default camera angle without rotating — and is the scene still worth close inspection at 4u?"

### Quick reference — one line per rule

The skyline is the present and the floor is the empire, and the empire never weathers — it is covered. PBR bones, painter's skin, and clean is the alien state. Density is a witness budget, and the combat threshold never moves. The world is a crime scene: three agreeing witnesses per event, contradictions are hooks, and freshness is a clue. The packs overlap but never balance, and geometry — never color — betrays the buried plan. Compose for the floor, run diagonals to one focal point, cut the roofs, hide the secrets on north walls, and author both the 20u map and the 4u craft.

---

## 6. UI Visual Language

*The UI is the investigator's desk, and the desk has three registers. The world is a painting the player walks through; the interface is where the painting is examined, annotated, and put to work. Section 2.4 established the contract: the UI runs its own flat, even register — State 5's deliberate break, so the mind can work — but it speaks the world's vocabulary: every UI color means what it means in the world. This section says how that register is drawn: what belongs where (6.1), the three faces of the interface (6.2), the pen-and-ink icon language (6.3), the desk's motion physics (6.4), the palette at work on real elements (6.5), how combat turns the desk into an instrument (6.6), and the accessibility and input contract (6.7). Three laws from earlier sections govern everything below: the register is flat (2.4), no critical state lives on hue alone (2.5), and the desk is where the truth is allowed to be orderly (Section 3, State 5).*

### 6.1 The Register Split — World, Desk, Field

The interface has three registers, and every UI element belongs to exactly one. If an element cannot name its register, it does not exist.

| Register | What lives there | Treatment |
|---|---|---|
| **The desk** — the investigator's thinking | Journal, main menu, settings, save/load, pause, victory record, tooltips | Parchment overlays on the flat register; the world dims to 30% behind them; the lamp is on (State 5) |
| **The desk, at the world's edge** — dialogue | The bottom-third conversation panel | The same parchment slip, sliding up from the bottom; **the world stays live above it** — no dim, because the room is part of the conversation (State 2: the interrogation room is a crime scene you can read while they talk) |
| **The field** — combat | Initiative bar, AP bar, ability menu, End Turn | The desk carried into the dark: panel-umber instrument surfaces, parchment rationed to text and chips (6.6) |
| **The world** — anchored | Health bars, damage numbers, status icons, targeting reticle, range line, cover icon, interact tags | Drawn in world space where the information lives, so the eye never leaves the crime scene (Pillar 1 — the whole frame is evidence); subject to Pillar 3 lighting discipline; scales with the 4u/10u/20u registers (4.4); floating text wears a soot outline or parchment backplate so it reads over any world value |
| **The thread** — crosses registers | Journal contradiction connectors, the web of knowing, the combat targeting line | The one visual allowed to exist on both sides of the desk: thread-red line. In the journal it connects facts; in combat it connects the player to the target. Connection is the same act in both places |

Boundary rules:

- **The desk holds thinking; the world holds state.** What the player knows, plans, and configures lives on the desk and is never diegetic — journal content never appears in the world. What the world is (health, threat, cover, target) lives in the world, anchored where it can be read without looking away from the scene.
- **The dim is the boundary.** Desk overlays dim the world to 30% over 200ms — the desk is an island of flat light over the living world. Dialogue is the exception: the world stays live, because watching the witness is part of the witness (4.3 — faces are rationed to State 2, and the portrait on the desk is its echo). **The title screen is the second exception:** the main menu panel does not dim the scene behind it — the world is the welcome. Modals and confirmation dialogs on the title screen dim as usual.
- **Screen-space UI never trembles and never tints.** Camera shake (crits, 2 units) moves the world only — the desk does not tremble. World lighting never touches UI (2.4 flat register). The ghost glow (2.6) is the empire's breath and never appears in the interface; the only self-luminous UI accent is amber, and it means lamp-light attention.
- **The world's lettering is painted, never typeset.** In-world signs, carvings, and inscriptions are part of the painting (5.2 legend filter). The desk is the only typeset place in the game — which is exactly why it reads as the player's own construction (State 5).
- **Portraits are the one painted element on the desk** — small witnesses in frames, the world's register invited in. They are not icons and are not typeset; they are the painting, at desk scale.

> **Design test:** "For any UI element, name its register: is it the desk (thinking), the field (deciding), or the world (seeing)? If it cannot name one, it does not exist."

### 6.2 Typography — Three Faces, One Voice

**Face 1 — The desk serif (journal, dialogue, menus, tooltips).** A humanist old-style serif in the Garamond class: warm, sturdy x-height, slightly hand-set rhythm — the scribe who writes for the investigator, not the printer who serves the empire. Candidates: EB Garamond, Cormorant, Source Serif (final pick with technical-artist in Section 8; must be OFL/licensed). **Not blackletter** — the grimoire is the empire's register, and even the empire is never typeset — and **not a cold modern serif**: the desk is warm by law (2.1, warmth leads).

**Face 2 — The combat face (all HUD numerals: damage, AP, HP values, hit chance, initiative).** A grotesque sans with tabular figures, wide apertures, and unambiguous digits — an instrument reading, not handwriting: the numbers of the fight are measured, not written. Candidates: Archivo, Inter. Numerals must read in a tenth of a second at Far zoom; tabular figures so HP/AP counts never jitter as they change. **Crits are never a new letterform** — same face, heavier weight, larger size, amber flash; the color and scale are the event.

**Face 3 — The world's lettering (in-world text).** Painted, etched, hand-cut — authored as texture, per 6.1 and 5.2. Never a UI font drawn into the world.

**Weight hierarchy — one job per weight.** Regular is the voice; italic is the aside (marginal notes, quoted testimony); bold is the shout, rationed like every accent. Small caps for headers and labels. There is no fourth weight; if a hierarchy needs one, the hierarchy is wrong.

**Size scale.** Screen-space text is fixed in screen pixels — the desk does not zoom. Reference baseline: 1920×1080; scale factor = screen height / 1080; **minimum supported resolution is 1280×720, and every legibility floor below must hold at it**. Body 16px; section headers ~1.5–2x body; captions and chips ≥ 13px; everything ≥ 4.5:1 contrast (2.4). **World-anchored text follows the 4.4 LOD contract:** authored in world units so it scales with the camera registers; legibility is contracted at Far (20u) — a damage number must read at 20u at its minimum on-screen size (≥ 14px combat face); as the camera pulls back, world text reduces by authored steps (numeral detail → fill-ratio reading), never by scaling below the floor. **Floating text never floats naked:** numerals (damage, miss, hit chance) wear a thin soot outline (`#141210`, 1–2px) so they hold against golden day and cold night alike; longer labels sit on parchment backplate chips. Text below the floor is redesigned, not shrunk.

### 6.3 Iconography — Ink Line, Flat Life

**The pen-and-ink rule.** Every glyph is drawn the way the investigator would sketch it in the margin: single-stroke ink line work, one weight (2px at a 24px base), flat fills only where the fill is a palette color with a job. No gradients, no bevels, no drop shadows, no outline glow. Shape carries the meaning; color carries the accent — the 2.5 discipline, built into the drawing style.

- **Ability icons** — margin sketches of the action on a parchment chip (desk) or umber chip (field). One accent allowed per icon, per role: amber rim (beneficial — the player's own power), oxblood (offensive), ink (utility). The icon says what the action does; the accent says whose side it is on.
- **Status effects** — small line glyphs whose shape IS the message: a flame (burn), a root (snare — moss accent), a chain (bind — ink), a drop (bleed — oxblood), a knot (contradiction — thread-red). A status is identified by its glyph before its color; color adds the 2.4 meaning.
- **Map markers** — the thread vocabulary made into pins on the parchment map: amber dot (active thread), amber star (prioritized), faded-ink x (dead end), ink check (resolved), thread-red knot (contradiction), pale ivory pin (empire-significant — rationed; an ivory pin on the map is an event). One status language across journal and map — the same marks in the same ink.
- **Warnings** — line glyph + oxblood accent; critical warnings add motion (6.4). Never oxblood alone (2.5).
- **Neutral information** (cover icon, hotkeys, structural chrome) — ink glyphs, no accent at all: neutrality is the absence of color-life.

**Legibility floor:** every glyph legible at 16px. An icon that fails the floor is redesigned, never scaled. Assets follow the studio naming convention (`ui_icon_[purpose]_[name]_[size].png`).

### 6.4 Motion Language — Paper, Ink, Thread, Lamp

The desk has material physics, and they come from State 5: **paper slides, ink appears, thread draws, light fades. Nothing springs, bounces, scales, spins, or flips** — the desk is flat by law, and 3D transforms in UI would break the promise of the flat register. Motion is a material property, not a flourish.

| Event | Motion | Source |
|---|---|---|
| Desk overlay in (journal, dialogue) | Slide up from the bottom edge, 300ms, ease-out | journal/dialogue GDDs — paper pulled onto the desk |
| Desk overlay out | Reverse path, 200ms — paper rolls away, never collapses | journal/dialogue GDDs |
| Field instruments in | From their edges: initiative from top, AP from left, ability grid from bottom, 200ms | combat GDD |
| Screen-to-screen (menus) | Crossfade, 250ms; title fades in 500ms; loading is a hard cut | main-menu GDD |
| Dim | World to 30% over 200ms — the lamp moves, the desk holds | main-menu GDD |
| Pulses | 500ms–1s gentle cycles — the only "alive" motion in the desk register | journal/dialogue GDDs |
| The thread draws itself | Connections animate as a line being stitched along the thread path — contradiction connectors, targeting line | journal GDD ("connector line draws") |
| Ink appears | Typewriter in word chunks, 45–60 chars/sec | resolution record, 2026-08-09 |

**The typewriter is the quill, and it writes in words.** Response text appears in word-chunk reveals at 45–60 chars/sec — the quill writing, not a character printer. Click, Enter, Space, or gamepad confirm skips to the completed text (the ink floods in); starting to type a new question auto-completes the current response; the accessibility "Instant" option bypasses the effect entirely. No information is gated behind animation.

**The pulse vocabulary is rationed like color.** Amber pulse = attention and reward (new thread, trust gain, revelation glow, current turn, AP refill). Oxblood pulse = tension (contradiction detected, critical health, error states). If everything pulses, nothing is urgent — no element animates longer than 1s without purpose, and never more than one dominant motion at a time. No critical information lives in motion alone — every pulsing state also carries a static cue (2.5).

**Dialogue failure is a broken thread.** If a response does not arrive within 10–15s, the slip shows an oxblood knot glyph — the thread broke — with a Retry chip; Cancel is available during Processing. The error is drawn in the desk's own vocabulary, never as a machine dialog.

**The desk does not tremble.** Screen-space UI is immune to camera shake and world weather — the flat register is a promise of stability, and the promise is what makes the world's violence legible. The crit shake (2 units) hits the world; the HUD holds still and lets the player read the result.

### 6.5 Color Application — The Palette at Work

The 2.4 vocabulary, applied to elements:

| Element | Treatment |
|---|---|
| **Panels** | Parchment `#E3D5B0` = paper surfaces (journal, cards, modals, tooltips). Panel umber `#2E2820` = the frame (toolbars, sidebars, menu bars, all combat chrome). Two text modes only: ink on parchment, parchment on umber |
| **Buttons** | Parchment face, ink label. Hover: amber edge — the lamp light finds the element. Active/selected: thread-red underline or rim — the thread ties it. Destructive: oxblood face or oxblood accent (resolution record). Disabled: desaturated to parchment/umber tones at ~40% opacity, no accent — **saturation is life, and a dead control is a bleached control** (the same rule that bleaches the empire, 2.6, applied to the interface) |
| **Links and threads** | Thread-red, always — anything that connects two things: journal links, entity chips, contradiction connectors, the targeting line. Thread-red is the color of connection, everywhere |
| **Selection** | Thread-red (current question, selected thread, valid target rim). Hover is amber; focus is not (below) |
| **Keyboard focus ring** | Oxblood `#7A2E28` on parchment surfaces; parchment `#E3D5B0` on umber surfaces — 2px, ≥ 3:1 contrast. Amber cannot carry focus: at ~2:1 on parchment it fails the floor. The lamp is for hover; the ring is for the hand that cannot hover |
| **The player's own** | Amber — health, AP, rewards, current turn, trust indicator. Amber is the living present, and in UI it marks what belongs to the player (2.1, applied) |
| **Damage and danger** | Oxblood — damage numbers, critical health, warnings, destructive actions. Warnings carry a glyph, and critical ones pulse; never hue alone (2.5) |
| **Combat spatial cue** | Cold indigo `#3E4A63` — ratified into the UI palette (resolution record): flanking arcs and the fight's geometry. Shape and motion carry the meaning first (6.6) |
| **Empire items** | Pale ivory, rationed: relic icons, oracle fragment markers, empire sigils in the journal. Ivory chrome is forbidden — no ivory buttons, sliders, borders, or text. Ivory and amber never share one element (2.4) |
| **Nature content** | Moss `#647A4E` — nature-affiliated entries only; never safe, never heal, never positive feedback (2.1) |
| **Text color rule** | Amber and oxblood are accent and fill colors, **never body text**: amber on parchment fails WCAG AA (~2:1). Text is ink on parchment or parchment on umber, always. The player's questions in dialogue are ink text with an amber name-chip — the player's voice is amber by attribution, not by letterform |

**Hover is lamp-light; selection is thread; destruction is wound.** Three sentences cover most of the interface: the lamp finds what you may touch, the thread marks what you have chosen, the wound marks what is dangerous.

### 6.6 The Combat HUD — The Desk in the Field

The combat HUD is the same desk, carried into the field: the same vocabulary, but the lamp is hard, the paper is rationed, and the dark wood dominates. It diverges from the parchment register for one structural reason: **it sits on the un-dimmed living world and must hold contrast against both the golden day and the cold night** (Pillar 3 — brightness never eats a decision, any more than darkness does). Parchment panels would glare against sunlit ground and vanish against ivory ruins; panel umber reads against both.

- **Base: panel umber at 85–90% opacity.** Dark frames over a bright world, dark frames over a dark world — the field register holds its contrast in both.
- **Turn order is the record, kept in dark wood.** The initiative bar is panel umber — never parchment, which would glare against the bright world (resolution record). Portraits ride on it as small painted vignettes; names in parchment-on-umber text. Current turn rimmed amber; upcoming turns dimmed (desaturated, 40% — the disabled rule). Hover shows name and HP in the combat face.
- **Health bars (world-anchored): faction-fixed hue on an umber/soot track.** The track is panel umber to soot — never parchment. Fill hue is fixed per faction: **amber for allies, oxblood for enemies — no hue shift as HP falls** (resolution record). The level is carried by bar length and the numeric HP readout (combat face, parchment-on-track or soot-outlined), never by hue. Low HP (below ~25%) adds a pulsing outline in the fill's hue — a motion cue, redundant to length and number (2.5). The faction pair is value-separated — amber is the palette's lightest chromatic family, oxblood the darkest — and the bar sits above the combatant it describes; the world's own cues (4.2) remain the primary read.
- **AP bar: amber fill** — the player's living resource (2.1). Number in combat face; refill pulses amber on turn change.
- **Damage numbers (world-anchored): combat face, oxblood, soot-outlined, float up.** Crits: ~1.5x size, heavier weight, amber flash — the same face, the event carried by color and scale. Miss: faded ink, soot-outlined. Hit chance and tactical readouts sit on parchment backplate chips — the desk's notes, handed to the player at the cursor.
- **Targeting: the thread, aimed.** The reticle and the range line are thread-red — the investigator's thread extended into the world. Valid targets rimmed thread-red; invalid targets desaturated. In the journal the thread connects facts; in combat it connects the player to the target; it is the same line (6.1).
- **Flanking arcs: cold indigo.** Ratified into the UI palette as the combat spatial cue — `#3E4A63`, the wild's own shadow (2.1, cold indigo). Arcs drawn behind the target, with a slow directional motion that reads "around the back"; shape and motion distinguish them from the static thread-red target rim, so hue is never the only difference (2.5). Spatial cues are the geometry of the fight — indigo is the battlefield's shape, never a combatant's state.
- **Cover indicator: neutral.** Ink shield glyph on a parchment chip, no accent — neutrality is the absence of color-life.
- **Ability menu:** umber panel, parchment glyphs; insufficient AP = desaturated; hotkey chips in ink on parchment.
- **End Turn:** amber, the largest fill in the HUD — the player's decision, made visible.
- **Victory and defeat: the desk returns.** The fight ends, the field register dissolves, and a parchment overlay records it — the record written down. The field exists only during the fight; the desk outlives it.
- **Pause during combat:** the combat UI renders above the dim layer (main-menu GDD) — the instrument stays readable while the desk opens.

> **Design test:** "Screenshot at Far zoom in the two worst cases — golden noon and cold night — and remove all color. If any tactical value (health, AP, turn, target, cover) cannot be read from value, shape, or position, the HUD is redesigned." (2.5, extended to the whole combat surface.)

### 6.7 Accessibility & Input Contract

*The desk must be reachable by every hand. These contracts are blocking; they are the interface half of 2.5 and the accessibility settings of the main-menu GDD, made visual.*

- **Focus management.** Full tab order through every screen; topic and journal chips navigate by arrow keys; the visible focus ring is oxblood on parchment and parchment on umber, 2px, ≥ 3:1 (6.5). Focus never disappears: if an element is reachable, its ring is visible.
- **Esc, one job per context.** In dialogue, Esc ends the conversation (the slip rolls away); outside dialogue, Esc opens the pause menu. Never both.
- **Resolution floors.** Minimum 1280×720; reference 1920×1080; UI scales by screen height and must hold every 6.2 floor at the minimum.
- **Text scale.** The accessibility text-size option scales the desk serif; no setting may push text below the legibility floor.
- **Instant text.** The typewriter's "Instant" option bypasses the word-chunk reveal (6.4) — no information is gated behind animation.
- **Auto-scroll manners.** The dialogue log's auto-scroll suspends when the player scrolls up manually and resumes only when the player returns to the bottom. The desk never steals the player's place.
- **Reduced motion.** The camera-motion accessibility settings also tame UI pulses; because no critical state lives in motion alone (6.4), the desk remains fully readable with motion off.

### Resolution record — cross-GDD rulings (2026-08-09)

*Where this section and the UI GDDs disagree, the rulings below stand; ux-designer updates the GDDs. The art bible is the source of truth for appearance.*

| Item | Ruling |
|---|---|
| Health bars | Faction-fixed hue (amber allies / oxblood enemies) on an umber/soot track; level by bar length + numeric readout; low-HP pulse outline; no hue shift as HP falls |
| Flanking arcs | Cold indigo arcs, ratified into the UI palette as the combat spatial cue; shape + motion distinct from the target rim |
| Typewriter | 45–60 chars/sec word-chunk reveal; skip on click/Enter/Space/gamepad confirm; auto-complete on new question; "Instant" accessibility option; 10–15s timeout → Error state with Retry; Cancel during Processing |
| Resolved marker | Ink check on parchment chip — settled, quiet, archived; no green, no amber |
| Player questions | Ink text with amber name-chip — never amber-on-parchment text |
| Floating text | Soot outline (numerals) or parchment backplate (labels) — never naked |
| Turn order bar | Panel umber, not parchment |
| Focus ring | Oxblood on parchment / parchment on umber, ≥ 3:1 (amber fails on parchment) |
| Esc | End conversation in dialogue; pause menu outside dialogue |
| Resolution | Min 1280×720; reference 1920×1080 |
| Auto-scroll | Suspends on manual scroll-up; resumes when the player returns to the bottom |
| Ivory usage (adopted from art-director draft; ux-designer confirmation pending) | Save-slot metadata ink-on-parchment; slider thumb amber or ink — ivory stays rationed to empire items |
| Destructive buttons (adopted from art-director draft; ux-designer confirmation pending) | Oxblood, not thread-red — thread-red remains connection/selection |

> **Follow-up:** 2.4's UI palette table should gain a cold indigo row (ratified here in 6.6) when the palette section is next touched.

### Quick reference — one line per concept

The world is a painting, the desk is the interface, and combat is the desk in the field — three registers, one vocabulary. The desk serif writes, the combat face measures, and the world's lettering is painted, never typeset. Icons are pen-and-ink: one weight, no gradients, shape carries meaning and color carries accent. The desk has material physics — paper slides, ink appears in word chunks, the thread draws itself, the lamp dims the world — and nothing bounces, and the desk never trembles. Saturation is life in the UI too: a disabled control is a bleached control. Amber is the lamp and the player's own; thread-red is the connection; oxblood is the wound; indigo is the fight's geometry; ivory stays rationed to the empire; text is ink and parchment, always. Combat trades paper for dark wood, keeps the record in umber, and aims the investigator's thread at the target.

**Production note (not part of the statement):** the UI is built in Kaiju's HTML/CSS-like system (main-menu GDD dependencies) — the flat register and CSS-style animation model agree; the durations in 6.4 are the tuning baseline. Resolution reference is 1920×1080 with height-based scaling (minimum 1280×720); fonts ship with the game (no runtime network fonts) and must be licensed for redistribution (OFL preferred). A text-scale accessibility option lives with ux-designer; the art contract is the legibility floor — no setting may render text below it.

---

## 7. VFX & Particle Style

*The two tests of Section 1 apply to air as surely as to stone: the legend filter at effect scale — a spell is "the breath of the old world," never a plasma bolt — and the fairy-tale balance, which keeps an ember beautiful because it dies. Particles are the world breathing. The air is a material with a memory: dust motes in a sunbeam prove the light is real, ash proves something burned, pollen proves the wild has been here — the air is a witness like any other (Principle 1). Every effect inherits three things from its source: the color family's semantics (Section 2), the source's motion signature (Section 4), and its rarity. Saturation is life in a particle as in a surface; the empire's light breathes where the present's flickers; and no effect may ever cost a tactical read (Pillar 3).*

### 7.1 The Particle Aesthetic — The Air Is a Material

Particle vocabulary is drawn from nature and the hearth, never from the laboratory: **embers, ash, dust motes, pollen, mist, fireflies, candle smoke, water rings.** No energy arcs, no hexagonal barriers, no straight-line sparks, no holographic readouts. When a spell is cast, the player should be able to describe it the way a borderlander would — "the ruin breathed," "the forest sighed," "the dead thing woke" — not "a laser fired."

**Material truth:** every particle is a small piece of matter with a story, and its behavior is its meaning. Ash is spent — it falls. An ember is still alive — it rises. Mist is patient — it hangs. A particle that does what its material cannot do (ash that glows, mist that pulses, fire that hums) is either empire family or wrong. The one exception is the empire's ghost glow — the only self-luminous pale light in the game (2.6) — and it is the exception precisely because the empire is the exception to everything natural.

**Three strata of air** (the two strata of 1.3 plus the wild): the present's warm breath — hearthsmoke, dust in golden shafts (State 2's dust motes are canon, Section 3), hearth embers; the empire's still exhalation — pale motes, breathing glow, even-strata haze; the wild's living fog — pollen, mist, fireflies. Air belongs to a family, and families never mix casually (Section 2): a puff of empire-mote in a warm room is an event, and it means the empire has reached this room.

**The fairy-tale balance at effect scale:** beauty and danger in the same breath of air. Hearthsmoke curls like a story told by a fire that could kill; fireflies are lovely and they mark the edge of the dark. A particle that is merely pretty is decoration; a particle that is only ominous is horror — both fail the balance. Moss never means safe (2.1), and neither does pollen.

> **Design test:** "Could this effect appear in a sci-fi shooter or a match-3? It fails. Could a borderlander describe it as the world breathing? It passes."

### 7.2 VFX Vocabulary by Source

Every family inherits three rules from its color grammar: **semantics** (Section 2's meanings hold in air), **motion signature** (Section 4's motion language extends to particles — the present is irregular, the empire is metronomic, the corrupted stutters), and **exclusivity** (amber effects never appear on empire material; corruption effects are rationed like the color). Motion is also the colorblind redundancy (2.5): the families must be separable by motion alone, with the screen muted.

| Source | Family | Signature particles | Motion signature | Rarity |
|---|---|---|---|---|
| **The living present** | amber gold, warm smoke-grey | hearth embers, candle flame, torch smoke, dust motes, lantern motes | organic, irregular, flickering; anchored to a visible source | default — the world's breathing |
| **The empire's remnants** | bone ivory, ghost glow, glaucous | breathing pale light, even-strata motes, ley channels | smooth, cold, too regular; a metronome's breath | events, never defaults |
| **The wild** | moss, cold indigo, warm pinpricks | pollen, spore puffs, ground mist, fireflies, water rings | organic, diffuse, patient; the slowest air in the game | everywhere it should be — and watching |
| **The corrupted** | oxblood on ivory | tendrils of red in pale light, sickly glow, falling red ash | the empire's evenness, broken — a stutter, an uneven pulse | the rarest effects in the game, rationed like the color |
| **Combat** | oxblood, white sparks, amber flash | hit sparks, blood spray, crit flash, death dissolve | brief, sharp, weighted, resolved | as frequent as violence — and as finite |

**The living present — the hearth's breath.** Fire is the present's signature particle, and it never repeats itself: flicker timing is noise-driven, embers rise on uneven convection, smoke curls with a lazy, asymmetric drift (torch smoke is warm grey with an ochre undertone — the color of fuel, not of a special effect). Warm light is always anchored to a visible source (2.5) — a hearth, a lantern, a wick — and the investigator's lantern carries a small comet of amber motes because the player is the light source that walks (4.5). Present effects appear in empire space only as intrusions, and always the player's or the living's (2.1): a hearth fire in a machine-hall is a sentence, and its embers are the period.

**The empire's remnants — the marrow's breath.** The empire's air is the uncanny in particle form: smooth, cold, too regular. Where the present flickers, the empire breathes — a slow, even swell and recede at a fixed tempo, no flicker, no warm falloff, no visible source (2.6, 2.5). Motes rise in perfect columns or hang in even strata; activation light spreads through channels the way blood returns to a body that should be dead — the old god waking, never a boot sequence (no hexagons, no circuit traces, no holograms: legend filter, 5.2). Empire effects obey the metronome of the empire's wake (4.1): no startle, no rest, no variation — the regularity is the tell. And they are rationed: one breathing glow per scene is a statement; a second is a revelation or a lie.

**The wild — the forest's sigh.** The wild's particles are organic, diffuse, and patient — the slowest air in the game, because the forest watches and has time. Pollen drifts on invisible currents in faint green-gold; spore clouds puff and hang; mist lies low and moves laterally — ground-hugging, which the isometric camera shows at its best (5.6). Fireflies are the night's signature: warm pinpricks in the cool indigo dark (State 1 — the dark is never empty of warmth), blinking at irregular, organic intervals; they are living things, so warmth is theirs (2.1's exclusivity guards empire material, not life). Water rings spread on the surface from the ground plane — the river doubles its banks' register (2.3), and a pale reflection of empire light on water is the world's quiet way of teasing the collision without delivering it.

**The corrupted — the wound in the marrow.** The rarest family, rationed like the color (2.2: blood on white is the world's corruption sentence). Corruption is the only thing that breaks the empire's motion: the breath stutters, the even tempo hiccups, the glow sickens and pools unevenly (2.3's corrupted sacred space — the false collision is noisy, the real one is still). Its signature is oxblood thread curling through pale light like a blood vessel in white marble — the marrow being fed upon, the sacred profaned. Corruption effects are spawned one at a time: one corrupted glow on screen is a statement; a scene full of them is a plot point.

**Combat — the brief register.** Combat effects are the exception to almost every patience rule: brief, sharp, weighted, and absolutely readable — the air must never obscure the tactical map (Pillar 3; the waist-height sightline contract of 5.3 extends to the air). **Hit sparks** are material-matched and tiny — steel on stone throws a white-grey flash of a few sparks for one or two frames; the flash is the read, the shower is not a firework. **Blood spray** is thread-red, heavy, finite — it arcs, falls, and is done; what remains is the stain on the ground, which persists (Pillar 2: the world keeps the mark). **Critical hit flash** is the only large combat moment, and it carries no particles at all — a brief amber flash (the event color, matching the damage number's flash, 6.6) plus a frame of hit-stop; sharpness, not spectacle; the wound itself stays oxblood. **Death dissolve** is a motion signature (4.1): a beast collapses and settles — gravity, no dissolve; an empire construct dissolves into pale motes that rise in a straight, even column, the metronome holding to the end, then stillness; a corrupted thing ends as falling red ash that stains where it lands. Combat feedback is never hue-only (2.5): every flash carries value and shape, and the sound system backs it.

### 7.3 Performance Budget

*Budgets below are the first draft of a contract; they are validated and tuned with technical-artist during Section 8, on target hardware. Kaiju is a custom Vulkan engine on a nightly pin (docs/engine-reference/kaiju/VERSION.md) with a particle system under active development (kaiju/src/rendering/vfx/) — implementation is technical-artist's; this section sets the visual contract the implementation must meet.*

**Per-effect budgets** (max concurrent particles):

| Effect tier | Examples | Budget |
|---|---|---|
| Ambient loop | dust motes, pollen, mist, ember trails | ≤ 128 |
| Small | hit sparks, lantern motes, blood spray, ripple rings | 16–32 |
| Medium | spell effect, machine activation, death dissolve | 64–128 |
| Hero | State 4 revelation only | ≤ 512, reserved |
| Critical flash | — | 0 particles (light pulse + hit-stop) |

**Per-scene budgets:** soft cap 2,000 active particles, hard cap 3,000; combat scenes cap at 1,500 (legibility and frametime — the air yields to the read). Emitter count is the real cost: ≤ 40 emitters per scene, ≤ 24 in combat. Gameplay-adjacent effects live ≤ 8 seconds — they must resolve out of the frame so the scene never accumulates haze. Ambient density LODs with zoom (4.4's discipline: reduction order — particle count first, emitters second, light third).

**Billboards, not volumes — two orientations.** Decision: billboarded 2D particles are the default for everything at gameplay scale. The isometric camera (30° pitch, 5.6) makes orientation a rule: **camera-facing billboards for vertical effects** (embers rising, sparks, rising motes — they read as columns with horizontal spread, which is what fire looks like from above), and **ground-aligned billboards for horizontal effects** (mist sheets, smoke pools, ripples, spore carpets — a camera-facing mist quad would read as a wall, and walls hide decisions). Two orientations cover the vocabulary; true 3D particle meshes are reserved for the rare hero effect (State 4), and even there, layered billboards usually suffice — technical-artist confirms on the pinned engine build.

**Light economy:** additive blending is rationed — empire ghost light and fireflies may glow; present fire is alpha-blended with a small additive core; wild effects are alpha only. Too much additive reads as sci-fi bloom (legend filter) and costs overdraw. There is no full-screen post bloom; glow lives in the material (2.6). Particles draw from one shared atlas of soft, organic sprites — no procedural shader particles except the light pulses, which are not particles at all.

### 7.4 Animation Principles — Drift, Float, Settle

**Nothing pops, nothing bounces, nothing moves like a screensaver.** The world's physics are a sentence: heavy ash falls, light embers rise, mist hangs, pollen drifts, dust settles. The forces are wind and convection, and they are gentle — a storm is the exception, and storms are events (7.5).

**Motion verbs are biography.** Rise (heat), fall (spent things and wounds), hang (patience), drift (the wild), settle (the world's memory), breathe (the empire). Weight is meaning: an ember rises because it is still alive; ash falls because it is done. A particle's trajectory is its life story, and it must be legible in half a second.

**Irregularity is life; regularity is the tell.** The present's motion is noise-driven — fire never repeats, embers deviate, smoke wanders. The empire's motion is a metronome — even spacing, even tempo, no variation (4.1's clock, obeyed by its light). The corrupted stutters — the empire's evenness broken by damage, the only irregularity born of harm rather than life.

**Settling is the rule.** Particles end by settling — falling, fading, or dissolving into stillness — never by popping out of existence. A particle that vanishes is a lie; a particle that settles is a story that ended. Spent particles leave evidence: ash piles, blood stains, scorch (5.4) — the world keeps the mark (Pillar 2), so death in the air becomes testimony on the ground. The one permitted sharpness is combat's brief register (7.2), and even there the effects are weighted and finite.

**Motion economy (4.3) applies to air.** Ambient effects are background actors: low amplitude, long periods, so a still frame still reads as a painting (the title screen is "a painting in motion," main-menu GDD). Ambient loops must not be perceptible as loops within 30 seconds — except the empire's, whose perceptible regularity is the point.

### 7.5 Weather Particles — Specified Now, Built for the Vertical Slice

*Weather systems are deferred to the vertical slice; this subsection is the contract they must meet, so weather and the light states of Section 3 are designed together. Weather layers onto states, never creates new ones (Section 3 layering note): distinctness is the rule, mood is the variable.*

- **Rain** — vertical streaks with a light spray haze, ground impact rings and puddle ripples (ground-plane effects are the isometric gift). Rain mutes the gold and deepens melancholy (Section 3); it darkens the frame's edges and keeps the center clear — value contrast for silhouettes never drops (Pillar 3). No full-screen post-process rain: the drops are in the world, not on the lens.
- **Fog and mist** — the wild's signature (7.2) scaled to a weather layer: ground-hugging sheets drifting on the diagonal axes (5.6). Fog muffles the gold and raises mystery (Section 3); it is the empire's ally — pale light spreads and softens in fog, raising the pale fraction. The 5.3 contract holds in air: fog never reaches waist height in a combat space.
- **Snow** (high places, later scope) — slow, sparse, settling; the most patient weather. Falling flakes need strong horizontal drift to read at the 30° camera (vertical streaks read poorly from above); the real story is the ground — snow accumulates and preserves, and a fresh blanket turns the world into a clue sheet: tracks, blood, and paths remain testimony under the white (Pillar 2; 5.4's freshness rule — tracks in fresh snow are the newest wound).
- **Weather and the states** — rain favors the present: wet ground is a reflective witness, puddles mirror their banks' register (2.3's river rule in miniature), and warm lantern pools spread on wet stone. Storm edges toward threat and justifies the indigo dark. Clear night sharpens the cold. **Weather never makes the collision** — only State 4 does — and the revelation can clear the weather: the environment answers (Section 3, State 4: weather clearing, birds falling silent). Weather intensity is a per-state dial: near-zero in State 2 (interiors), and the desk (State 5) is dry by design — weather is a world register (2.4).
- **Budgets** — rain ≤ 1,000 streaks plus ≤ 60 pooled ripple decals; fog: 8–16 large ground-aligned sheets (big soft quads, not particles — cheaper and gentler); snow ≤ 800 flakes with drift.

### 7.6 UI Particles — The Desk Does Not Breathe

**Rule: the flat register stays flat (2.4). The desk has no particle systems.** No floating dust on the parchment, no sparkle on buttons, no ember trails on the thread graph. The desk (State 5) is the player's own ordered construction — the one place the truth is allowed to be orderly (Section 3, State 5) — and its stillness is the point. Motion in the desk is the vocabulary of Section 6.4: paper slides, ink appears, thread draws, light fades, pulses breathe — authored 2D animation, never procedural effects. A particle in the desk would be magic in the one room that is pure craft.

The three moments that might tempt a particle resolve as single, authored, one-cycle 2D beats — one element, one breath, then still:

- **Journal parchment dust — none.** Pages turn, threads draw themselves, ink dries. The journal's motion is the hand's, not the air's.
- **Level-up — an amber swell.** A single warm-gold swell that rises and settles like a hearth's breath (amber = rewards, 2.4; the pulse vocabulary of 6.4), drawn in the desk's flat language: one ring, one cycle (~1s), then stillness. The hearth's ember translated to parchment, not a firework.
- **Trust-increase — the portrait's amber pulse.** Section 6.4 already names trust gain an amber pulse: one gentle cycle on the portrait's rim, then still — the connection warmed, not celebrated (2.5: value and shape carry the read; no critical state lives in motion alone).

World-anchored HUD (the field register, 6.1) may breathe where the world does: empire-derived statuses pulse at the empire's tempo — slow, even, metronomic — as authored 2D animation, never particle fields; motion is the redundant cue (2.5). Particles belong to the world register; the UI register owns animation. The single exception to all of the above is the title screen (State 6), where the world's own particles are allowed to fill the frame: drifting embers, pollen, dust motes in sunbeams — the painting in motion (main-menu GDD) — and still minimal: the fairy tale breathes; it does not sparkle.

### Quick reference — one line per rule

Air is a material and a witness: embers, ash, dust, pollen, mist, fireflies — the hearth's breath, the marrow's breath, the forest's sigh, and the wound. The present flickers, the empire breathes on a metronome, the corrupted stutters, and combat is brief, sharp, and never hides a decision. Particles settle, drift, or rise — nothing pops, nothing bounces, nothing moves like a screensaver. Billboards in two orientations: vertical effects face the camera, horizontal effects lie on the ground, and additive glow is rationed. Weather layers onto the states — rain favors the present, fog favors the empire, snow turns the world into a clue sheet — and only revelation can clear it. The desk does not breathe: no particles in the flat register, and the three big moments are one-cycle 2D beats, not effects.

**Production note (not part of the statement):** all budgets in 7.3 are first-draft contracts to be validated with technical-artist during Section 8 on the pinned Kaiju nightly (docs/engine-reference/kaiju/VERSION.md); the engine's particle system (kaiju/src/rendering/vfx/) is under active development, and the two-orientation billboard decision is the visual contract the implementation must meet.

---

## 8. Asset Standards

*The world's skin is its memory (Principle 2), and this section is how that skin is cut. Six standards govern every asset's journey from the quarry to the frame: the format it travels in (8.1), the name it carries (8.2), the resolution it is authored at (8.3), the LODs it shrinks through (8.4), the split between what the DCC authors and what the engine decides (8.5), and the KitBash3D foundation everything is quarried from (8.6). Together they guard the painterly treatment of 5.2 — the grade, the overlays, the two material regimes — because a standard that lets the treatment slip lets the world read as two games. Two earlier production notes resolve here: the open pipeline choice of 1.3 and the budget validation of 7.3 are settled jointly with technical-artist in this section's production note.*

### 8.1 File Format Preferences — FBX Travels, PNG Carries the Grade

**What the pipeline accepts, and what we prefer:**

| Content | Accepted | Preferred | Notes |
|---|---|---|---|
| Meshes | OBJ, BLEND | **FBX** | FBX is the interchange standard: it carries hierarchy, transforms, and material names. BLEND exports to FBX (headless or manual) before staging (pipeline CR-2). OBJ is accepted for simple, single-material meshes only — no hierarchy, no material references. |
| Textures | TGA | **PNG** | PNG is lossless and universal — and losslessness is an art requirement: the grade (5.2) is authored into the albedo, and a lossy detour degrades exactly the warm darks and pale value shifts the palette is built from. TGA is accepted for legacy pack assets. Both convert to engine native at import. |
| Materials | — | **Texture sets + authored values** | No DCC material files (.spp, .blend materials, .mat) enter the engine. Material intent leaves the DCC as texture sets plus authored values (roughness/metalness, emissive flags); assembly happens engine-side through the material mapping table (8.6). |

Rules:

- **The DCC's own formats are source-only.** .blend, .spp, .psd, and the packs' .max variants are the quarry, never the staging input. Staging speaks FBX and PNG.
- **Engine-native formats are technical-artist's contract** [Technical-artist values pending — to be merged]. Art direction sets what leaves the DCC; technical-artist sets what the engine consumes; the conversion in between is the pipeline's (CR-2).
- **Anything else the packs ship** (.tif, .exr, proprietary scene files) is converted at source or rejected — the staging directory is format-disciplined so validation has no surprises.

### 8.2 Naming Conventions — The Name Is the File's Witness

**Every asset is named so its name answers "what is this, and what is it for" with no documentation** — Principle 1 at file scale. The pipeline validates the convention (CR-3: kebab-case, no spaces, no special characters; failure = reject with rename suggestion).

Pattern: `[category]_[name]_[variant]_[size|role].[ext]`

| Category | Prefix | Examples |
|---|---|---|
| Props (placeable objects) | `prop_` | `prop_ruin_archway_01.fbx`, `prop_shrine_candle_01.fbx` |
| Environment structures (buildings, setpieces, terrain pieces) | `env_` | `env_tower_colossus_remnant_large.fbx` |
| Characters | `char_` | `char_party_investigator_01.fbx`, `char_npc_tavernkeeper_01.fbx` |
| Textures (texture sets) | `tex_` | `tex_stone_gothic_wall_albedo.png` |
| Materials (mapping-table names) | `mat_` | `mat_stone_empire_ivory_01` |
| UI | `ui_` | `ui_icon_ability_scorch_24.png` (per 6.3) |
| VFX (atlas sprites) | `vfx_` | `vfx_ember_rise_small.png` |

Rules:

- **Texture channels are suffixes on a shared base name:** `_albedo`, `_normal`, `_roughness`, `_metalness`, `_ao`, `_emissive`. A set is the base plus its channels.
- **LODs are suffixes on the mesh name:** `_lod0`, `_lod1`, `_lod2` (8.4).
- **Variant numbers are zero-padded** (`_01`) when a name recurs.
- **Pack origin lives in pipeline metadata, never in the name.** An imported asset is a borderland asset, not a "Dark Fantasy" asset — the name describes what it is now. (The mixing rule of 5.5 is a property of scenes, and the name must not prejudice it.)
- **A name that needs a lookup table is a name that failed.** If the witness is not legible in the filename, the file is misnamed.

### 8.3 Texture Resolution Tiers — Resolution Is a Legibility Budget

Resolution is authored for the closest register that will ever see the surface (4.4: detail authored for Close, readability for Default, legibility for Far). A 2048 texture that is only ever seen at Far zoom is a budget spent on nothing.

| Tier | Assets | Resolution |
|---|---|---|
| **Hero** | player character, party members, key NPCs, major setpieces, empire relic props | **2048×2048** |
| **Standard** | ordinary props, NPC costumes, minor setpieces | **1024×1024** |
| **Background / tiling** | terrain, tilesets, foliage atlases, repeated surfaces | **512×512** (tilesets may reach a 4096 atlas per the pipeline cap) |
| **UI** | icons, panels, portraits | **as needed, minimum 2× display resolution** — a texture that displays at 200×100 px is authored at 400×200 px minimum; the desk never scales above its authored size |

Rules:

- **Power of two, always.** Non-PoT textures pad at import with a warning (CR-3); we author PoT so padding never costs texel density.
- **The tier is a ceiling, not a target.** The pipeline downscales, never upscales.
- **The empire exception: 16-bit, and never down-tiered.** The empire's craft shows in near-imperceptible value shifts, not hue (2.6, Principle 3) — exactly what 8-bit banding destroys. Empire-family albedo and emissive export as **16-bit PNG** [engine 16-bit texture support: technical-artist values pending — to be merged] and hold their class's tier even when the asset is minor: a small ivory fitting still gets 1024, because its smoothness is its testimony.
- **Textures reduce before anything else** (4.4's reduction order): LOD1 hero surfaces run 1024, LOD2 runs 512 — resolution is the first LOD, geometry the last.
- **VRAM is a scene budget, and the tiers are its ceiling.** Per-scene VRAM limits [Technical-artist values pending — to be merged]; when a scene overruns, scenery down-tiers before witnesses — the witness budget (5.3) outranks texture tiers.
- **UV hygiene is part of the resolution contract:** island padding (4 px at 2048, scaled per tier) so mip bleeding never fuzzes the empire's clean surfaces; texel-density reference values per asset class [Technical-artist values pending — to be merged].

### 8.4 LOD Level Expectations — Three Registers, Two Cuts

The fixed orthographic camera (5.6) is a gift to LOD: the zoom registers are discrete and deterministic, so switching is never a surprise. Thresholds anchor to the camera zoom registers (4u Close / 10u Default / 20u Far), with hysteresis at the boundaries.

| Category | Levels | LOD0 | LOD1 | LOD2 |
|---|---|---|---|---|
| **Characters** (party, key NPCs, the empire's wake, the wild's beasts) | 3 | ≤ 20k tris (hero) / ≤ 10k (standard NPC) | ≤ 10k / ≤ 5k | ≤ 5k / ≤ 2.5k |
| **Props** | 2 | ≤ 5k tris (pipeline cap) | ≤ 1.5k | — |
| **Small scenery** | 1 | ≤ 500 tris + distance cull | — | — |
| **Terrain** | dynamic | technical-artist's scheme [Technical-artist values pending — to be merged]; art contract: seamless tile seams, no material pops, the ground plane's empire testimony (5.1) survives every LOD | | |

Tri budgets above are first-draft ceilings consistent with the Asset & Scene Pipeline's caps [poly count budgets: Technical-artist values pending — to be merged].

**Distance thresholds (first draft, with technical-artist)** [LOD switch implementation: Technical-artist values pending — to be merged]: LOD0 for half-heights below 7u (covers Close 4u — and the 2.5u accessibility register sees full detail, though 5.6's rule holds: no critical information exists at only one register); LOD1 for 7u–15u (Default 10u); LOD2 beyond 15u (Far 20u). Switches carry ±1u hysteresis so a register change never pops twice.

**The 4.4 contract is the LOD contract:**

- The accent color never changes across LODs — a range-shifted hue is a faction error.
- Silhouette is never LOD'd — shape simplifies last, and the weapon-posture read survives to the final draw call.
- Reduction order is fixed: texture → face → props → costume → silhouette (never).
- **Hostile readability survives to LOD2.** At Far zoom the rim light (2.5) is the last cue that a figure in shadow is hostile — a hostile that reads as a dark blob at Far fails the contract and is redesigned, not LOD'd away.

### 8.5 Export Settings Philosophy — The DCC Authors, the Engine Decides

The split is a rule, not a habit: **the DCC owns appearance; the engine owns delivery.** Nothing engine-specific is authored in the DCC, and nothing appearance-critical is decided at import.

| What | Owned by | Notes |
|---|---|---|
| Geometry, UVs, transforms | DCC | FBX, neutral and clean |
| Texture sets (albedo, normal, roughness, metalness, AO, emissive) | DCC | PNG; 16-bit for empire surfaces (8.3) |
| Material intent (roughness/metalness values, emissive flags) | DCC, as data | never as engine materials (8.1) |
| **The painterly grade** (5.2) | DCC, authored into the albedo | **never re-applied at import — no double-grading** |
| Color-space tagging | DCC, correct at export | albedo/emissive sRGB; roughness/normal/AO linear; flags confirmed with technical-artist |
| Compression (BCn / ETC2 / ASTC per platform) | Engine | one master set, compressed per target [compression targets: Technical-artist values pending — to be merged] |
| Mips, streaming | Engine | [Technical-artist values pending — to be merged] |
| Material assembly (shader, mapping table) | Engine / pipeline | 8.6 |
| LOD switching, instancing, culling, draw calls | Engine | 8.4 [draw call budgets: Technical-artist values pending — to be merged] |
| Collision, navmesh | Engine | never authored in the DCC [collision mesh requirements: Technical-artist values pending — to be merged] |

Rules:

- **Exported once, read many.** One master texture set and one mesh export serve every platform — no per-platform DCC exports. The engine compresses; the DCC never does.
- **The grade survives conversion untouched.** The warm darks are the contract's most fragile part — soot black is warm, never muddy blue-grey (5.2) — and a color-space slip at export or import is a palette violation, not a technical quirk.
- **Neutral exports:** no baked engine data, no hardcoded material slots, nothing that names an engine concept in the DCC file.
- **Rig and animation budgets are engine constraints** [animation constraints: Technical-artist values pending — to be merged] — bone counts, clip counts and lengths, blending. Art direction's contract holds regardless of budget: idles are task-bound and weighty (4.3), motion economy applies (7.4), and the hostile-motion signatures of 4.2 survive any LOD.

### 8.6 The KitBash3D Integration — The Quarry and the Mason

The packs are the quarry; the treatment is the mason. **Dark Fantasy and Enchanted are foundation assets — geometry and texture sets that the pipeline normalizes and the treatment completes — never final content.** An asset straight from the pack is a candidate, not a placement.

**What the pipeline expects from the packs** (per Asset & Scene Pipeline CR-2):

- **Models:** FBX/OBJ only; the packs' other DCC variants are rejected at source.
- **Textures:** PNG/TGA sets; converted to PNG in staging; non-PoT padded with a warning.
- **Scale normalization:** both packs normalized to the project scale (1 unit = 1 meter); a 2-unit human-height door is the reference object that validates consistency between packs. **Scale variance is authored, not a bug:** the colossal empire setpieces (5.6's vertical accents) are intentionally larger than the reference — normalization fixes the packs' *units*, never their *intent*.
- **Rotation normalization:** Z-up to Y-up (KitBash3D ships Z-up; Kaiju is Y-up), applied in staging.
- **Material remapping:** a hand-authored mapping table per pack (CR-2) — the pipeline never guesses material intent; unmapped materials flag for human review and load as default grey.

**What art direction adds to the pipeline's mechanics:**

- **The mapping table is where the color contract is enforced.** It is the mechanical home of the exclusivity rule (2.1): empire-family source materials map only into the empire material set (bone ivory, silver, porcelain, glaucous, dead gold — 2.6); present materials map into the warm stone/wood/metal families. A source material that maps into the wrong family is a palette violation, flagged before it reaches a scene. The two material regimes (5.2: present rough and warm; empire smooth and clean) are properties of the material sets, not of individual textures.
- **The legend filter runs at import.** Anything reading generic-fantasy or modern-engineering (5.5) is treated — overgrowth, re-paint, modification — or rejected. The palette validator's out-of-family warning is the filter's mechanical first pass; the human pass is the final one.
- **Both packs pass through the same treatment** — the same grade, the same overlay system — or the world reads as two games (5.5). There is no "Dark Fantasy look" and no "Enchanted look" inside the game; there is the borderland.
- **The ratio rule has a mechanical ally.** The validator's single-pack-dominance warning (>80% of visible surface area, CR-6) enforces 5.5's no-50/50-frames rule at scene load: a frame that ties the packs reads as a collision, and the collision is reserved for State 4.
- **Pack assets are scenery by default; witnesses are earned by authoring** (5.3). Import knows nothing about what an asset knows; the witness budget is decided at placement, by hand.

> **Design test:** "For any asset entering the pipeline: does it pass the legend filter as-is, or is it treated? Does its mapping place it in the right color family, and does its name say what it is? If any answer is no, it is a candidate, not content."

### Quick reference — one line per standard

FBX travels and PNG carries the grade, and the DCC's own formats never leave the quarry. A name answers what an asset is with no documentation — kebab-case, zero-padded, channel and LOD as suffixes, pack origin kept in metadata. Resolution is a legibility budget: hero 2048, standard 1024, background 512, UI at 2× display — and the empire's smoothness is 16-bit, always. Three registers, two cuts: characters get 3 LODs, props 2, terrain dynamic — and the accent never shifts a hue and the silhouette never LODs. The DCC authors appearance, the engine delivers it: exported once, read many, and the grade is never re-applied. The packs are the quarry and the treatment is the mason: normalized, remapped, filtered, and graded until Dark Fantasy and Enchanted are one world — a ruined archway swallowed by an enchanted forest, and not a single asset that remembers which pack it came from.

### Production note (not part of the statement)

Section 8 is authored jointly: **art direction owns the appearance contracts above; technical-artist owns the engine's hard constraints; where both appear, the stricter holds.** Engine values — native formats (8.1), 16-bit texture support, VRAM limits and texel density (8.3), poly count budgets, LOD switch implementation and terrain LOD (8.4), compression targets, mip/streaming, draw call budgets, animation constraints, collision mesh requirements (8.5) — are marked in their subsections as [Technical-artist values pending — to be merged]; they do not block authoring and are filled in when technical-artist completes. Two earlier production notes resolve with them: the 1.3 pipeline choice (2.5D hand-painted vs. full 3D — the 5.2 treatment contract holds under either, and this section is written so both choices honor it) and the 7.3 particle budget validation on the pinned Kaiju nightly.

---

## 9. Style Prohibitions

*Every section before this one says what the game is; this one says what it is not. Prohibitions exist because "wouldn't it be cool if…" is the most common failure mode in visual development: a cool idea is not a true idea, and the fastest way to break a visual identity is to add the one thing that does not belong to it. Each prohibition below names a thing this game never does, the rule it would break (cited), and the true version we do instead. These are not taste preferences; they are load-bearing. When a new visual idea arrives, the question is not "is it cool?" but "which rule does it keep?" — an idea that keeps no rule is rejected, and an idea that breaks a rule is accepted only by amending this document, in writing.*

### 9.1 Never Sci-Fi

**Never:** plasma, neon, holograms, digital displays, chrome — anything that reads as engineering at first sight.

The legend filter of Section 1.1 exists to catch exactly this: "myth, never machine," and "if a prop, material, or effect reads as engineering, it fails the rule and gets redesigned." The empire stratum is defined as *mythic, never sci-fi* — it reads as *inhuman* rather than futuristic (1.3). The rule holds in the air as in stone: the particle vocabulary is drawn from nature and the hearth, never from the laboratory — no energy arcs, no hexagonal barriers, no straight-line sparks, no holographic readouts (7.1); activation is "the old god waking, never a boot sequence" (7.2); additive blending is rationed because too much additive reads as sci-fi bloom (7.3).

The nuance that keeps the rule honest: the ban is on *first-sight reading*, not on the machine itself. The function may be technological; the appearance must be mythic (1.1), and Principle 3 lets the machine live under the surface — tooling seams, calibration marks, value shifts in the bone (2.6) — as the reward for study. A relic that shows its craft only up close is earned revelation; a relic that announces its engineering across the frame is a different game wearing this one's ruins.

**Instead:** the empire's technology reads as holy relics, oracle stones, impossible geometry — bone ivory, ghost glow, surfaces too smooth to be natural (2.6). If the player cannot describe an empire thing the way a borderlander would, it is wrong.

### 9.2 Never Generic Fantasy

**Never:** dragons, elves, glowing magic swords, fireballs, wizard robes — the ready-made vocabulary of fantasy.

The Marrow of Empire is dark fantasy grounded in a fallen technological civilization (1.3), and its supernatural vocabulary is closed: it belongs to the four hostile families of 4.1 and nothing outside them. The wild's teeth are beasts and wood-things — "natural, never cartoonish" (4.1); the great beasts may exceed human scale, but silhouette alone carries their threat (4.1) — they are not monsters of the week. "Hostility is not a species; it is a choice" (4.1): there are no fantasy races, only people wearing the same warm register as everyone else. The investigator is not a chosen one — "no inherited sigil, no family crest, no destined birthright" (4.5) — and carries no magic blade: a modest blade, "carried as a tool" (4.5), because the game's signature power object is a journal. There are no fireballs: when the old world acts at range, it acts the way the world breathes (7.1), not the way an RPG casts. The pack rule already polices the asset side: anything reading generic-fantasy or modern-engineering "gets treated (overgrowth, re-paint, modification) or is replaced" (5.5).

**Instead:** the four hostile families (4.1), the investigator's practical tools (4.5), grounded character design — "people are people, not caricatures" (4.3), and every costume is a biography, not a wardrobe (4.1). When an artist reaches for a familiar fantasy prop, ask what it would know in this world (Principle 1): a magic sword knows nothing here.

### 9.3 Never Horror or Gore for Its Own Sake

**Never:** jump scares, gratuitous gore, body horror as decoration.

The game is dark but not horror: the fairy-tale balance of 1.1 — "beautiful enough to mourn, dangerous enough to fear" — pulls every scene back from the two failure modes, pure horror and pure prettiness. Combat is "not horror, not chaos: a dangerous dance with clarity" (Section 3, State 3). The horror notes the game does allow are rationed by design: the blood-touched are "the game's horror notes — rationed" (4.1); corruption effects are "the rarest effects in the game, rationed like the color" (7.2); a living person who has taken true ivory into themselves is "the game's rarest character note" (4.6). Rationing is the difference between a register and a decoration — and the register has a job.

**Instead:** the blood-on-ivory corruption signal (2.2) — earned, rationed, always telling a story. Blood on white is always a clue — "Pillar 1: every surface knows something" (2.2) — and a wound is testimony that answers what happened here (5.4). Gore that answers no question is scenery that failed Principle 1: "if the answer is nothing, it doesn't belong in the frame" (1.2).

### 9.4 Never UI That Breaks the Flat Register

**Never:** 3D UI elements, diegetic HUD floating in world space, skeuomorphic buttons — anything that pretends the interface lives inside the painting.

The desk is flat by law: "nothing springs, bounces, scales, spins, or flips — the desk is flat by law, and 3D transforms in UI would break the promise of the flat register" (6.4). Icons are pen-and-ink with "no gradients, no bevels, no drop shadows, no outline glow" (6.3). And the interface never goes diegetic: the world is a painting, and the desk is the player's own ordered construction — the one place the truth is allowed to be orderly because it is the player's, not the world's (Section 3, State 5). The world-anchored register of 6.1 (health bars, damage numbers, the targeting line) is drawn in world space where the information lives, but it is flat UI anchored to the scene — never diegetic props: no holographic readouts (7.1 bans them at effect scale), no in-world menus, no buttons that belong to the painting. The boundary is the 6.1 split: "the desk holds thinking; the world holds state." An element that tries to be both breaks the split and is redesigned.

**Instead:** parchment panels, umber frames, pen-and-ink icons (6.3), paper-slide motion (6.4). The section's own test is the test: "for any UI element, name its register… if it cannot name one, it does not exist" (6.1).

### 9.5 Never Procedurally Generated Content

**Never:** procedural dungeons, random NPCs, generated quests — a seed is not a writer.

The game's depth is hand-authored by construction: "composition is authored, never improvised; every object is placed, and placement is testimony" (5.6). Pillar 1 — Every Voice Holds a Piece — is a promise that every witness was written by a hand, anchored by "the anti-pillar of hand-authored depth: every placed object implies a hand that placed it" (1.2). Density is "a witness budget, not a decoration budget" (5.3) — a budget is set per scene by a designer, never sampled by a function — and even the moss is "authored per-asset, never random" (5.2). A generator cannot produce testimony, only variation: the moment a witness's placement is algorithmic, Principle 1's question — "what does this object know?" — is answered by a seed instead of a writer, and the crime scene becomes a noise field. The prohibition extends to the knowledge graph itself: the web of knowing (State 5) is authored — fragments connect because a writer connected them.

**Instead:** authored scenes, authored NPCs, authored knowledge graphs — every room, every witness, every thread.

### 9.6 Never Filler Combat

**Never:** random encounters, respawning trash mobs, XP grind zones.

Pillar 3 is "Tactical Depth, Not Tactical Bloat" (1.2) — and bloat includes combat that exists to consume time. Every fight carries narrative weight: the world is a crime scene (5.4), and a hostile's presence is testimony — the wild's teeth guard territory, the blood-touched are evidence of what happened, the empire's wake "simply do not stop" (4.1). The respawn is doubly impossible. It contradicts Pillar 2: "ash, blood, and worn paths persist after combat" (5.4) — the world keeps the mark, and a respawned enemy is the world forgetting, which this world never does. And it contradicts the combat-effects contract: violence is "as frequent as violence — and as finite" (7.2), and effects resolve out of the frame so the scene never accumulates haze (7.3). Infinite violence would be haze.

**Instead:** authored combat encounters with story context — the palette announces danger before the enemies do (2.3 threshold rule), and encounters are framed by the world's evidence: the interrupted meal, the forge left hot (5.4). If a fight cannot answer what happened here (5.4's three questions), it does not happen.

### 9.7 Never Comedy, Parody, or Meta

**Never:** self-referential jokes, fourth-wall breaks, ironic detachment.

The tone is sincere, and sincerity is structural: the fairy tale "beautiful enough to mourn" (1.1) has no register for the joke, because a joke reminds the player this is a construction — the same way a sci-fi readout does (9.1). The desk is the one meta-adjacent space — the player's own construction (Section 3, State 5) — and even it stays in-world: it is the investigator's desk, paper and ink and thread, never the game winking at the player.

Warmth is not the same as comedy. The game's warmth lives in human moments: State 2's hearthlit intimacy — "the most human moment in the game" (Section 3); the biography in an idle — "the smith checks his hammer" (4.3); the interrupted meal (5.4). Those moments are warm because they are true, not because they are funny.

**Instead:** the world takes itself seriously — warmth comes from human moments, not jokes. When a line or a prop would make the player laugh *at* the world, it is cut; when it would make them smile *with* it, it stays.

### 9.8 Never Pure Black or Pure White

**Never:** `#000000` or `#FFFFFF` — in the world, in the UI, in text, in any material.

Every palette color has a narrative job, not an aesthetic one (2.1) — and the two pure values cannot hold a job. Soot black is `#141210`, warm-tinted: its job is "the void between fragments… what the player has not learned yet" (2.1), and an unknown with warm undertones is a mystery with texture, where pure black is a void with no promise — **pure black reads as void, not unknown.** Bone ivory is `#E9E6DC` (2.6), near-white but never pure: the marrow's job is to read as holy (2.1), and holiness is not sterility — **pure white reads as sterile, not sacred.** The discipline holds in the desk too: ink is `#1C1813`, never black (2.4); ghost glow is `#F5F3EC`, the lightest color in the game, and still warm at the edge (2.6). And the painterly grade (5.2) is the craft reason: the world is a painting, and paint is never a pure value — "soot black is warm, never muddy blue-grey" (5.2) — a pure black or pure white surface would jump out of the canvas like a hole cut in it.

There is also a meaning reason, and it is the load-bearing one. The game's darkness is never empty: "the shadow holds what they have not said yet" (Section 3, State 2) — so black must carry warmth, because the unknown is full of something. And the empire's light is a presence: the marrow fills the frame at revelation (State 4) — so white must carry bone, because the sacred is a substance, not an absence. The pure values would say the opposite of both sentences.

**Instead:** soot black `#141210` for the unknown, bone ivory `#E9E6DC` for the sacred, ink `#1C1813` for text — every dark warm, every light veined with warmth, the palette always one step inside the world's edge. If a material needs something blacker than soot or whiter than ivory, the material is wrong, not the rule.

### Quick reference — one line per prohibition

Never sci-fi: the empire is holy relics and oracle stones, and the machine shows only under study (1.1, 2.6). Never generic fantasy: the four hostile families, the investigator's journal and modest blade — no chosen ones, no glowing swords (4.1, 4.5). Never horror for its own sake: blood on ivory is testimony, rationed and earned, never decoration (2.2). Never UI outside the flat register: parchment, umber, pen-and-ink, paper-slide — no 3D, no diegetic HUD, no bevels (6.3, 6.4). Never procedural generation: every placement is authored, and a seed is not a writer (5.6). Never filler combat: every fight answers what happened here, and the world keeps the mark (5.4). Never comedy, parody, or meta: the world is sincere, and warmth is human moments, not jokes (State 2). Never pure black or pure white: soot is warm, ivory is bone, and the palette always lives one step inside the world's edge (2.1, 2.6).

---

## 10. Reference Direction

*Five references steer the visual identity. A reference is a direction, not a destination: each entry below names one specific thing to take — a technique, a color choice, a compositional rule, a design philosophy, never a whole aesthetic — the exact divergence that would turn the reference into a copy, and what the reference validates about this game. The selection rule is additive: no two references point in the same direction. Each owns one domain — decay, composition, scale, warmth, and the empire's concept — and together they cover the whole identity. The legend filter (1.1) applies to references as to assets: we take what serves the borderland, and everything else stays in the source. This section is the formal home of the references cited inline throughout the bible (Section 3's warmth anchors, 2.1–2.2's blood-on-white and nature-reclaims-stone, 4.3's LOTR register, 5.4's abandoned settlements); the inline citations point here.*

### 10.1 Dark Souls — Decay as Narrative

- **What to take:** Environmental storytelling through decay — the placement discipline that makes every ruin, stain, and broken thing evidence of a specific event, so the landscape reads as a witness (Principle 1). The ruined-grandeur register: wreckage that is terrible and still worth mourning, the fallen thing's original magnificence legible in its fall (1.1's fairy-tale balance). And the atmospheric principle that light is the mood's first instrument — light sculpting what the player can see and what the world keeps hidden (Section 3).
- **What to avoid:** The grimdark default. Dark Souls' grey desaturation, ash-toned light, and world-as-hostile-force register; its opacity — story buried where the player may never find it; its "everything is dead" tone. This borderland is amber-led (2.1: warmth leads) and alive — the living present outshines the dead (State 1) — and no darkness may eat a decision (1.2, Pillar 3). A Souls-graded frame would bleach the golden register off the game in a single pass.
- **Why it matters:** Validates Principle 1 and the evidence grammar of 5.4: a game can tell its story through the world's wounds with no dialogue at all, and decay can be beauty. Dark Souls is the proof that "beautiful enough to mourn" is a real register, not a slogan.

### 10.2 Diablo 1–2 — The Isometric Crime Scene and Blood on White

- **What to take:** The isometric gothic composition — the camera angle that hands the player the whole corrupted space at once, readable as a map (5.6's ground-plane-first; the whole-crime-scene gift of Section 5). The corrupted sacred space: a holy place profaned, sanctity still visible under the violation (2.2's blood-on-white — the world's corruption sentence). And moody lighting as menace: pools of glow against shadow, light as atmosphere with a job (Section 3, State 1's warm pools in indigo night).
- **What to avoid:** The murk. Diablo's signature darkness serves atmosphere before readability — silhouettes vanish into shadow and the player fights the lighting as much as the enemies. That fails the cross-cutting constraint (1.2) and Pillar 3 outright. Also avoid the grime-default — the permanently dingy brown-grey dungeon register — and the horror-saturated gore (9.3: horror for its own sake is banned; blood on white is always a clue, never decoration, 2.2).
- **Why it matters:** Validates the isometric camera as a storytelling device rather than a genre default, and the corruption signal (2.2) as the world's most legible sentence. Diablo proved a sacred space can carry a whole game's mood; here that mood has a job — the corrupted sacred space (2.3) is the false collision that teaches the player to distrust cheap reveals, so the real one (State 4) is earned.

### 10.3 The Lord of the Rings — Scale, Warmth, and Lived-In Craft

- **What to take:** The scale language — ancient architecture that dwarfs people, doors too tall, halls too vast, so the body knows the builders were not human before the mind does (5.1's inhuman scale). Nature reclaiming stone — moss, root, and time healing the old world's works (2.1's moss green, the world's slow memory; the 5.5 signature image of a ruin swallowed by forest). Warmth in the face of darkness — firelight, hearth, and golden hour alive inside the darkest acts (Section 3: the dark is never empty of warmth). And practical costume design: clothing worn, patched, and specific to a life (4.1: costume is biography; 4.5's not-a-chosen-one).
- **What to avoid:** The cinematic register. LOTR's muted, filmic, desaturated grade would fight the painterly saturation this game is built on (5.2: saturation is life). Avoid the epic-heroic framing — sweeping romantic heroism that would turn the investigator into a chosen one (4.5 bans inherited shine) — and the fantasy-familiar iconography: elves, dragons, glowing swords are prohibited vocabulary (9.2).
- **Why it matters:** Validates the two-strata world (1.3): the present's lived warmth set against the empire's inhuman scale is exactly the Shire-to-ancient-kingdoms relationship — and validates the practical, biographical costume language that keeps the world grounded instead of cosplayed.

### 10.4 Drakensang: The River of Time — The Golden Register

- **What to take:** The painterly, inviting warmth — a world rendered like a painted adventure, golden light as the default register, the frame itself welcoming (5.2's painter's skin; Section 3's warmth directive; State 1's golden adventure soul). Drakensang is the reference for the *baseline*: the register the player lives in most hours, the light the game returns to after every dark event. Its sequence is the game's sequence — the world invites you in, and danger follows behind.
- **What to avoid:** The comfort. Drakensang's world is warm, safe, and never truly dangerous — its danger is a stage prop. Here the invitation must carry a real cost: the register has to be able to break (2.3: temperature falls with depth; the collision is reserved for State 4), and the fairy-tale balance (1.1) bans pure prettiness. Avoid the drift to picturesque cosiness — taverns and fairs with no gothic weight, a holiday in the middle ages.
- **Why it matters:** Validates the art direction's most distinctive choice: adventure leads with warmth, and darkness is the exception that events must earn. Most dark fantasy starts cold and gets colder; this game starts golden, which is why its cold moments land (Section 3: when warmth leads, coldness is an event).

### 10.5 One Piece (the Void Century) — The Mythic-Tech Concept

- **What to take:** The concept, never the style: a precursor civilization so advanced that its achievements survive only as myth — its technology read as magic, its artifacts venerated as relics, its true function misunderstood by the present (Principle 3: the surface is the lie, the craft is the truth; the legend filter, 1.1). The Void Century is the proof that "a technological empire the world remembers as gods" is a coherent, magnetic premise — and that the misunderstanding is itself the story (Pillar 4: the machine under the miracle is the reward for study).
- **What to avoid:** Every visual element of One Piece — cartoonish proportions, manga energy, cel-shaded brightness, rubbery motion, and its tone of boundless optimism. And the concept's sci-fi temptations: power armor, energy weapons, holograms — anything whose engineering is legible at first sight (9.1: never sci-fi; 1.1: the ban is on first-sight reading). The empire must read as *inhuman*, never futuristic (1.3); the moment a relic reads as a machine, it is a Void Century artifact in a game that promised a legend.
- **Why it matters:** Validates the game's central visual wager — that the empire stratum (2.6: the marrow — bone ivory, ghost glow, impossible smoothness) can be *technological* while reading as *mythic*, and that the gap between the two readings is the engine of mystery. One Piece is the reference for what the empire *is*, not for what it looks like.

### Quick reference — one line per reference

Dark Souls = decay as narrative, beauty in the fall (Principle 1, 5.4) — never grey, never dead, never opaque. Diablo = the isometric crime scene and blood on white (5.6, 2.2) — never murk, never gore. LOTR = inhuman scale, nature reclaiming, warmth against the dark, worn craft (5.1, 2.1, 4.5) — never cinematic, never heroic, never elves. Drakensang = the golden painterly register that invites (5.2, State 1) — never comfortable, never safe. One Piece = the Void Century concept, technology remembered as myth (1.1, Principle 3, 2.6) — never a single frame of its style.

# Control Manifest

> **Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan)
> **Last Updated**: 2026-08-09
> **Manifest Version**: 2026-08-09
> **ADRs Covered**: ADR-0001 through ADR-0013
> **Status**: Active — regenerate with `/create-control-manifest update` when ADRs change

This manifest is a programmer's quick-reference extracted from all Accepted ADRs,
technical preferences, and engine reference docs. For the reasoning behind each
rule, see the referenced ADR.

---

## Foundation Layer Rules

*Applies to: entity architecture, camera, asset pipeline, save/load, event bus*

### Required Patterns
- **GameEntityID (uint64) maps to Kaiju EntityId (string)** — every game entity has both; GameEntityID is the universal reference, never reused after destruction — ADR-0001
- **Typed component stores**: `map[GameEntityID]ComponentType` per component, accessed via Go generics — ADR-0001
- **Uniform grid spatial index**: `map[[2]int32][]GameEntityID` for range queries — ADR-0001
- **Lifecycle bridge**: Kaiju `OnActivate/OnDeactivate/OnDestroy` → game-layer events via EventBus — ADR-0001, ADR-0011
- **Write arbitration**: per-frame delta accumulation, system priority resolution — ADR-0001
- **All component stores implement `EntityData`**, registered via `pod.Register` — ADR-0001, ADR-0002
- **Orthographic camera**: 30° pitch, 45° yaw, 0° roll — fixed, never rotated — ADR-0003
- **Three zoom registers**: Close 4u, Default 10u, Far 20u (ortho half-height) — ADR-0003
- **Occlusion**: structures between camera and player fade to transparent; interiors render without roofs — ADR-0003
- **FBX → Kaiju native conversion** via staging pipeline — ADR-0004
- **YAML material mapping tables** per KitBash3D pack — ADR-0004
- **Asset validation**: poly ≤5k static / ≤20k hero; texture ≤2048; materials ≤8; kebab-case naming — ADR-0004
- **Full-state pod serialization** for save/load — ADR-0002
- **Save versioning** with registered migration functions per version transition — ADR-0002
- **Checksum (CRC32) + .bak backup** — corrupted save detected, restored from backup — ADR-0002
- **Steam Cloud**: Auto-Cloud at launch, conflict resolution dialog — ADR-0002
- **Platform save paths**: Linux `$XDG_DATA_HOME`, Windows `%APPDATA%`, macOS `~/Library/Application Support` — ADR-0002
- **Synchronous event dispatch** via EventBus with system priority ordering — ADR-0011
- **Event subscribers register by event type + priority** (1 = highest) — ADR-0011

### Forbidden Approaches
- **Never define a parallel entity system** — use the game-layer-over-Kaiju pattern (ADR-0001 chose this over parallel system and direct Kaiju extension) — ADR-0001
- **Never reuse a GameEntityID** after entity destruction — ADR-0001
- **Never use JSON/text-based saves** — pod binary encoding is the standard (rejected for performance: 5-10× larger files, slower encode/decode) — ADR-0002
- **Never use incremental/delta saves at MVP** — full-state is simpler and sufficient for 500 entities (revisit if save times exceed 2s) — ADR-0002
- **Never allow the player to rotate the camera** — fixed isometric is a design decision, not a limitation — ADR-0003
- **Never allow the camera to show void space** — camera bounds always show ≥80% playable area — ADR-0003
- **Never skip material mapping** — unmapped materials load as default grey; the pipeline never guesses — ADR-0004
- **Never use asynchronous event dispatch** — synchronous with priority guarantees correct ordering (death must be recorded BEFORE trust events fire) — ADR-0011
- **Never use direct cross-system calls for state mutations** — all mutations go through the owning system's interface, with events for notification — ADR-0011

### Performance Guardrails
- **Entity creation**: <0.5ms — ADR-0001
- **Spatial query** (500 entities): <1ms — ADR-0001
- **Entity ID lookup**: ~50ns (map access) — ADR-0001
- **Save** (500 entities): <2 seconds — ADR-0002
- **Load** (500 entities): <3 seconds — ADR-0002
- **Camera matrix update**: <0.1ms/frame — ADR-0003
- **Asset import** (50MB FBX): ~30s — ADR-0004
- **Scene load** (200 placements): ~2s — ADR-0004
- **Event dispatch** (5 subscribers): <1µs — ADR-0011
- **All event handlers combined**: <2ms/frame — ADR-0011

---

## Core Layer Rules

*Applies to: character stats, NPC knowledge graph, living environment*

### Required Patterns
- **Six-attribute model**: STR/DEX/CON/INT/WIS/CHA, range 3-18, modifier = `floor((A-10)/2)` clamped [-4,+4] — ADR-0006
- **Point-buy creation**: 30 points, all start at 8, max 16 starting, cost = N-8 — ADR-0006
- **Derived stats**: MaxHealth, AP, Initiative, Evasion calculated from attributes — ADR-0006
- **XP from threads + combat**: linear thresholds, `XP_to_next = Level × 300`, cap 20 — ADR-0006
- **YAML .kg files per NPC** at `assets/npc_knowledge/{region}/{npc_id}.kg` — ADR-0005
- **Fact types**: FACT (confident), RUMOR (caveated), OPINION (subjective), SECRET (trust-gated) — ADR-0005
- **Trust gating**: facts tagged `trust_required` 0-3, queried by current `trust_level` — ADR-0005
- **Knowledge boundaries**: explicit `knows_not` (domains) and `knows_not_entities` per NPC — ADR-0005
- **Entity references**: facts reference entities by GameEntityID → enables cross-referencing — ADR-0005
- **Game clock**: 24-hour, 10:1 ratio (2.4 real hours = full day), starts 8:00 AM — ADR-0007
- **Five light states**: Dawn/Day/Dusk/Night/Deep Night with smooth interpolation — ADR-0007
- **Wait/pass-time**: rest_point interaction → select wake time → 100× fast-forward — ADR-0007
- **NPC schedules**: Schedule component with start/end times — ADR-0007
- **Combat pauses clock** — time frozen during combat — ADR-0007
- **Memory log**: append-only per entity, soft cap 50, protected entries never evicted — ADR-0013
- **World evolution rules**: YAML-authored condition→event, evaluated once on region load — ADR-0013

### Forbidden Approaches
- **Never use a class system** — attributes ARE the class — ADR-0006
- **Never embed NPC knowledge in Go code** — content must be separable from engine (rejected for requiring recompilation) — ADR-0005
- **Never use JSON for .kg files** — YAML's readability and comment support are required for designer authoring — ADR-0005
- **Never advance time during combat, conversation, menus, or journal** — clock pauses in these states — ADR-0007

### Performance Guardrails
- **Attribute modifier**: <0.01ms — ADR-0006
- **Derived stat recalculation** (level-up): <0.1ms — ADR-0006
- **.kg YAML parse**: ~10ms per NPC — ADR-0005
- **Fact query** (domain + trust + trigger filter): <0.5ms — ADR-0005
- **Clock update**: <0.01ms/frame — ADR-0007
- **Light state interpolation**: <0.1ms during transitions — ADR-0007
- **Memory log append**: <0.1ms — ADR-0013
- **Evolution rules** (20 rules): <1ms per region load — ADR-0013

---

## Feature Layer Rules

*Applies to: AI dialogue, question system, trust, thread investigation, combat, PWS, party, factions*

### Required Patterns
- **LLM inference via separate llama.cpp process** — stdin/stdout JSON IPC, crash isolation — ADR-0008
- **LLMBackend interface**: `Generate(ctx, prompt) → (string, error)` — swappable — ADR-0008
- **Prompt template**: [SYSTEM] character + rules → [FACTS] trust-filtered → [HISTORY] → [QUESTION] → [RESPONSE] — ADR-0008
- **Graceful degradation**: text fallback on model failure, auto-restart, timeout→retry — ADR-0008
- **Square grid combat**: 1 unit = 1 meter, diagonal 1 AP — ADR-0009
- **AP system**: 4 + DEX_mod (min 2, max 8), costs: Move 1, Attack 2, Ability 1-3 — ADR-0009
- **d20 attack resolution**: d20 + accuracy >= evasion → hit; nat20 = crit (2×); nat1 = auto-miss — ADR-0009
- **Death/dying**: 0 HP → Dying, 3 rounds to stabilize (2 AP, adjacent) — ADR-0009
- **Positioning**: flanking +2, half cover +2 evasion, full cover blocks LoS, elevation +1 acc/range — ADR-0009
- **Event-driven thread graph**: threads auto-created from fact references, auto-chained by entity overlap — ADR-0010
- **Contradiction detection**: discovered fact with `contradicts` → create thread, emit event — ADR-0010
- **Revelation threshold**: 3+ connected facts → chain `revelation_ready` — ADR-0010
- **PlayerKnowledge component**: persists discovered facts, threads, chains, revelations — ADR-0010

### Forbidden Approaches
- **Never use in-process CGo for LLM models exceeding 500M parameters** — a 3B+ model crash must never take down the game (use IPC). Tiny models (<500M, <400MB disk) may use CGo with `recover()` guard as a latency-critical fast-path for simple queries — ADR-0008 (revised 2026-08-11)
- **Never use HTTP for local LLM** — adds latency without benefit over stdin/stdout for local-only service — ADR-0008
- **Never use hex grid** — misaligned with isometric 45° camera (ADR-0009 chose square) — ADR-0009
- **Never use range-based (gridless) combat** — "within 2 meters" invites edge cases; "within 2 tiles" is unambiguous — ADR-0009
- **Never use a traditional quest log** — violates the anti-pillar "NOT a completionist checklist" — ADR-0010
- **Never rely on manual-only thread management** — auto-detection of connections is what makes the player feel smart — ADR-0010

### Performance Guardrails
- **LLM response**: <2s target for 50-200 tokens — ADR-0008
- **Model load**: ~15s during main menu (non-blocking) — ADR-0008
- **LLM memory**: ~3-4GB separate process — ADR-0008
- **Attack resolution**: <1ms — ADR-0009
- **Combat AI decision**: 10-50ms — ADR-0009
- **OnFactDiscovered** (thread creation + chain + contradiction check): <5ms — ADR-0010

---

## Presentation Layer Rules

*Applies to: all UI systems, rendering, audio*

### Required Patterns
- **Kaiju HTML/CSS-like UI** — all UI rendered in screen space — ADR-0012
- **Three registers**: Desk (parchment overlay, world dims 30%), Field (umber combat chrome), World (anchored bars/numbers) — ADR-0012
- **Desk serif** (Garamond class) for journal/dialogue/menus — ADR-0012
- **Combat face** (grotesque sans, tabular figures) for AP/HP/damage numbers — ADR-0012
- **Paper-slide motion**: enter 200-300ms ease-out, exit 200ms — no bounce/scale/spin/flip — ADR-0012
- **Desk never trembles** — UI immune to camera shake — ADR-0012
- **Minimum 1280×720**, reference 1920×1080 with height-based scaling — ADR-0012
- **Body text 16px at reference**, minimum 12px effective — ADR-0012
- **Accessibility**: tab order + focus ring (≥3:1), arrow-key navigation, typewriter speed setting, Esc mapping — ADR-0012
- **OFL-licensed fonts** shipped with game, no runtime network fonts — ADR-0012

### Forbidden Approaches
- **Never use 3D UI or diegetic HUD** — the desk is flat by law — ADR-0012
- **Never use skeuomorphic buttons** — pen-and-ink icons only — ADR-0012
- **Never allow amber-on-parchment text** — 1.5:1 contrast ratio fails WCAG AA (amber is accent/fill only) — ADR-0012
- **Never encode game-critical information by hue alone** — all colors have shape/icon/value/sound backup — ADR-0012

### Performance Guardrails
- **UI render** (static): <1ms/frame — ADR-0012
- **Journal open/close transition**: 200-300ms — ADR-0012
- **No frame drops during UI animations** — ADR-0012

---

## Global Rules (All Layers)

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, single word | `combat`, `ai`, `dialogue` |
| Exported types/functions | PascalCase | `PlayerController`, `TakeDamage` |
| Unexported types/functions | camelCase | `moveSpeed`, `updateState` |
| Interfaces | PascalCase, `-er` suffix for single-method | `Renderer`, `Saver` |
| Files | snake_case | `player_controller.go` |
| Scenes | PascalCase matching root entity | `ForestGrove.scene` |
| Constants | PascalCase (Go convention) | `MaxHealth`, `DefaultMoveSpeed` |

### Performance Budgets

| Target | Value |
|--------|-------|
| Framerate | 60 fps |
| Frame budget | 16.6ms |
| Draw calls | [TO BE CONFIGURED — Vulkan-dependent] |
| Memory ceiling | [TO BE CONFIGURED — must account for 2.7GB LLM model] |

### Forbidden Patterns
- None configured yet — add as architectural decisions are made

### Allowed Libraries / Addons
- None configured yet — add as dependencies are approved

### Engine Verification Required (All Layers)

Kaiju Engine is a custom engine with no LLM training data. Every ADR carries "Verification Required" items. Before implementing any system, verify the relevant engine APIs against `kaiju/src/`:
- `engine.Entity` lifecycle hooks — ADR-0001
- `matrix.Transform` dirty-flag behavior — ADR-0001
- `pod.Register` reflection requirements — ADR-0001, ADR-0002
- `cameras.NewStandardCameraOrthographic` API — ADR-0003
- Kaiju native mesh/texture formats — ADR-0004
- Kaiju HTML/CSS-like UI capabilities — ADR-0012
- AccessKit screen reader support — ADR-0012
- llama.cpp IPC latency with game running — ADR-0008

### Cross-Cutting Constraints
- **Downward-only dependencies**: higher layers depend on lower; lower never know about higher — architecture.md
- **Interface-first**: every cross-layer boundary is a Go interface — architecture.md
- **Save everything**: all game state in component stores; Save/Load orchestrates — ADR-0002
- **Verify against source**: no engine API used without checking Kaiju source first — architecture.md

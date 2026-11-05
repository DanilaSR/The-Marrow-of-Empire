# Kaiju Engine — Version Reference

| Field | Value |
|-------|-------|
| **Engine Version** | 26.07.29.1-nightly |
| **Release Date** | Nightly build (under active development) |
| **Project Pinned** | 2026-08-08 |
| **Last Docs Verified** | 2026-08-08 |
| **LLM Knowledge Cutoff** | May 2025 |
| **Risk Level** | HIGH — custom engine, no LLM training data exists |

## Knowledge Gap Warning

Kaiju Engine is a custom Go/Vulkan game engine. No LLM has training data
on Kaiju Engine APIs, patterns, or conventions. ALL engine guidance must
come from the primary reference sources listed below.

## Primary Reference Sources

1. **`kaiju/AGENTS.md`** — Comprehensive agent-facing documentation covering
   all engine subsystems, APIs, patterns, and conventions. This is the
   single most important reference file.
2. **`kaiju/CLAUDE.md`** — Build commands, build tags, project structure,
   and workflow documentation.
3. **`kaiju/docs/`** — Engine documentation (mkdocs-based).
4. **`kaiju/src/`** — Engine source code (module: `kaijuengine.com`, Go 1.26.0).
5. **`kaijuengine-game-dev` skill** — Essential reference skill for building
   games on Kaiju Engine.

## Engine Architecture

| Subsystem | Location | Purpose |
|-----------|----------|---------|
| Bootstrap | `kaiju/src/bootstrap/` | Engine initialization, game interface |
| Engine Core | `kaiju/src/engine/` | Core engine systems |
| Rendering | `kaiju/src/rendering/` | Vulkan-backed 2D/3D rendering |
| Matrix Math | `kaiju/src/matrix/` | Custom transform math library |
| UI System | `kaiju/src/engine/` | HTML/CSS-like UI (no JS runtime) |
| Editor | `kaiju/src/editor/` | Built-in editor (itself a Kaiju game) |
| Platform | `kaiju/src/platform/` | OS abstraction (Windows, Linux, Mac, Android) |
| Network | `kaiju/src/network/` | Networking systems |
| Framework | `kaiju/src/framework/` | Application framework |
| Audio | `kaiju/src/engine/` | Music, SFX, 3D sound sources |

## Key Conventions

- **Module path**: `kaijuengine.com`
- **Build**: CGO required, build tags control feature sets
- **Math**: Always use `kaijuengine.com/matrix` — never use `gonum/mathgl`
- **Game entry**: Implement `GameInterface` from bootstrap
- **Entities**: Custom matrix Transform (not standard 4x4)
- **UI**: HTML/CSS-like markup, no JavaScript runtime

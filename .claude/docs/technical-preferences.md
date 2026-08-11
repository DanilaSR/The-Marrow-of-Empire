# Technical Preferences

<!-- Populated by /setup-engine. Updated as the user makes decisions throughout development. -->
<!-- All agents reference this file for project-specific standards and conventions. -->

## Engine & Language

- **Engine**: Kaiju Engine (custom, Go/Vulkan, nightly 26.07.29.1)
- **Language**: Go
- **Rendering**: Vulkan (2D/3D)
- **Physics**: [TO BE CONFIGURED]

## Input & Platform

<!-- Written by /setup-engine. Read by /ux-design, /ux-review, /test-setup, /team-ui, and /dev-story -->
<!-- to scope interaction specs, test helpers, and implementation to the correct input methods. -->

- **Target Platforms**: PC (Steam / Epic)
- **Input Methods**: Keyboard/Mouse, Gamepad (partial)
- **Primary Input**: Keyboard/Mouse (isometric CRPG — point-and-click movement, tactical grid selection)
- **Gamepad Support**: Partial (recommended but not primary)
- **Touch Support**: None
- **Platform Notes**: UI must support mouse-driven interaction as primary. Gamepad navigation for menus and combat as secondary.

## Naming Conventions

- **Packages**: lowercase, single word (e.g., `combat`, `ai`, `dialogue`)
- **Exported types/functions**: PascalCase (e.g., `PlayerController`, `TakeDamage`)
- **Unexported types/functions**: camelCase (e.g., `moveSpeed`, `updateState`)
- **Interfaces**: PascalCase, `-er` suffix for single-method (e.g., `Renderer`, `Saver`)
- **Files**: snake_case (e.g., `player_controller.go`)
- **Scenes/Prefabs**: PascalCase matching root entity (e.g., `ForestGrove.scene`)
- **Constants**: PascalCase (Go convention — no UPPER_SNAKE_CASE)

## Performance Budgets

- **Target Framerate**: 60 fps
- **Frame Budget**: 16.6ms
- **Draw Calls**: [TO BE CONFIGURED — Vulkan-dependent, profile after first region built]
- **Memory Ceiling**: [TO BE CONFIGURED — must account for 2.7GB LLM model + game memory]

## Testing

- **Framework**: Go native `testing` package + `kaijuengine.com` integration test harness
- **Minimum Coverage**: [TO BE CONFIGURED]
- **Required Tests**: Combat formulas, AI NPC knowledge graph queries, investigation thread logic, save/load round-trip

## Forbidden Patterns

<!-- Add patterns that should never appear in this project's codebase -->
- [None configured yet — add as architectural decisions are made]

## Allowed Libraries / Addons

<!-- Add approved third-party dependencies here -->
- [None configured yet — add as dependencies are approved]

## Architecture Decisions Log

<!-- Quick reference linking to full ADRs in docs/architecture/ -->
- [No ADRs yet — use /architecture-decision to create one]

## Engine Specialists

<!-- Written by /setup-engine when engine is configured. -->
<!-- Read by /code-review, /architecture-decision, /architecture-review, and team skills -->
<!-- to know which specialist to spawn for engine-specific validation. -->

- **Primary**: kaijuengine-game-dev (skill-based, covers all engine systems)
- **Language/Code Specialist**: lead-programmer (Go code review, architecture)
- **Shader Specialist**: technical-artist (GLSL/Vulkan shaders, materials)
- **UI Specialist**: ux-designer + kaijuengine-game-dev (HTML/CSS-like UI system)
- **Additional Specialists**: ai-programmer (NPC LLM integration), gameplay-programmer (combat/exploration systems)
- **Routing Notes**: No dedicated Kaiju subagent exists. The `kaijuengine-game-dev` skill provides engine guidance. All engine-specific code should invoke this skill. For general Go architecture, invoke `lead-programmer`. For rendering/shaders, invoke `technical-artist`. For Kaiju UI (HTML/CSS-like markup), use `kaijuengine-game-dev` skill + `ux-designer`.

### File Extension Routing

<!-- Skills use this table to select the right specialist per file type. -->

| File Extension / Type | Specialist to Spawn |
|-----------------------|---------------------|
| Game code (.go files) | lead-programmer + kaijuengine-game-dev skill |
| Shader / material files (.glsl, .frag, .vert) | technical-artist |
| UI / screen files (.html, .css — Kaiju UI markup) | ux-designer + kaijuengine-game-dev skill |
| Scene / prefab / level files (.scene, .prefab) | kaijuengine-game-dev skill |
| Entity / component files (.entity, .component) | kaijuengine-game-dev skill |
| Asset / import config files | kaijuengine-game-dev skill |
| General architecture review | lead-programmer + kaijuengine-game-dev skill |

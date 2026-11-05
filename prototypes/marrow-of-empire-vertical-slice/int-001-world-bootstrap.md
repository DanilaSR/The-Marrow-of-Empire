# INT-1 — Game World Bootstrap (Complete)

- **Date**: 2026-08-12
- **Status**: ✅ Complete — exit criterion met (walkable world in a running build)

## Scope

New Game → debug-shape scene (ground, town structures, mine, NPC markers),
fixed isometric camera (ADR-0003), click-to-move player, camera follow.

**Cut**: NPC interaction (INT-2), combat (INT-3), journal (INT-4), character
creation, transitions, polish.

## Files

| File | Action | Content |
|------|--------|---------|
| `src/game/world.go` | new | world build, `MoveToward`, `updateWorld`, NPC cubes (INT-2 edit) |
| `src/game/world_config.go` | new | layout/tuning constants, art-bible colors |
| `src/game/game.go` | edit | `startNewGame`, resize hook, world/talk fields |
| `src/game/title_screen.go` | edit | New Game button → world |
| `tests/unit/game/world_movement_test.go` | new | 7 tests: MoveToward math |

## Engine APIs Verified Against Source

- `host.Cameras.Primary.ChangeCamera` (camera container swap)
- `StandardCamera.TryPlaneHit` (screen → ground-plane picking)
- `host.Window.Mouse` (`hid.Mouse.Pressed` / `ScreenPosition`)
- `rendering.NewMeshPlane/NewMeshCube` (unit meshes; scale via transform)
- `shader_data_registry.Create("basic")` color + `MaterialDefinitionBasic` + `TextureSquare`
- `document.Document.Destroy` (title → world transition)

## Live Verification (standalone + AI Driver)

- Build: `cd kaiju/src && go build -tags="debug,ai_driver" -o ../../bin/game-standalone .`
- Run: **from the project root**, `LLAMA_SERVER_BIN=... ./bin/game-standalone`
- Title screen renders (4 buttons located by pixel forensics)
- New Game click → world renders (ground, buildings, ivory player cube, mine)
- Click-to-move both directions (8,269 px changed on first walk; camera follows — player stays centered)
- Zero panics

**Evidence**: `production/qa/evidence/screenshots/int1-world-render.png`,
`int1-player-moved.png`

## Findings

1. **Editor vs standalone build**: the `make build-ai-driver` target uses editor
   tags — it launches the Kaiju *editor*, not the game. The standalone build
   (`debug,ai_driver`, no `editor` tag) runs `getGame()`; it must run from the
   project root (content DB is CWD-relative).
2. AI Driver input is dropped while the game window lacks OS focus (first
   session had `focused:false`; the standalone launch gained focus on its own).

## Velocity

One working session including engine API verification, implementation, unit
tests, and two live-verification rounds (editor/standalone confusion + focus
hand-off).

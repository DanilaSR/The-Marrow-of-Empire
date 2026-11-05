---
name: first-launch
description: "Launch a Kaiju Engine game for the first time. Codifies the full bootstrap path: GameInterface implementation, engine getGame() wiring, game_content setup with extension-mapped content database, standalone build, and AI Driver verification. Run when a game module exists but the engine has never run it, or when the game window shows nothing / crashes at startup."
argument-hint: "[--with-ai-driver] [--editor]"
user-invocable: true
allowed-tools: Read, Glob, Grep, Bash, Write, Edit, AskUserQuestion
model: sonnet
---

# Kaiju First Launch

Bootstraps a Kaiju Engine project from "library code exists" to "game window
renders". Based on the verified bootstrap path from Marrow of Empire
(2026-08-12): every step below was validated against Kaiju nightly
26.07.29.1, and every failure mode in Phase 4 was hit and fixed in a real
first launch.

**When to run**: the game module (`src/`) compiles but the engine has never
displayed game content, or `./bin/game` crashes at startup with asset/GPU
errors.

**Success criterion**: game window opens, AI Driver health check returns
`{"ok":true}`, screenshot shows game content (not just the clear color).

---

## Phase 1: Verify Module Wiring

The engine and game are separate Go modules linked by `replace` directives.
Both directions must exist.

```bash
# 1. Game module requires engine
cat src/go.mod    # require kaijuengine.com v0.0.0
                  # replace kaijuengine.com => ../kaiju/src

# 2. Engine module requires game (for standalone builds)
cat kaiju/src/go.mod  # require marrow-of-empire v0.0.0
                      # replace marrow-of-empire => ../../src
```

If the engine's `go.mod` lacks the game requirement, add it:

```bash
cd kaiju/src
go mod edit -require="marrow-of-empire@v0.0.0"
go mod edit -replace="marrow-of-empire=../../src"
```

**Circularity check**: this is NOT circular. The engine's `main` package
imports the game; the game imports engine packages. `main` ≠ engine library
packages, so the module graph stays acyclic.

## Phase 2: Create the GameInterface

The engine calls three methods on `bootstrap.GameInterface`:

1. **`ContentDatabase()`** — MUST return a non-nil database. Returning nil
   panics with `asset database cannot be nil` during window creation.
2. **`Launch(host)`** — build initial UI, register updates.
3. **`PluginRegistry()`** — nil is fine for pure-Go games.

Create `src/game/game.go` (package `game`):

```go
func (g *Game) ContentDatabase() (assets.Database, error) {
    base, err := assets.NewFileDatabase("game_content")
    if err != nil { return nil, err }
    return &contentDB{base: base}, nil
}

func (g *Game) Launch(host *engine.Host) {
    g.host = host
    g.uiMan.Init(host)           // REQUIRED before any UI element
    g.buildTitleScreen()          // markup-based (see Phase 5)
    _ = host.Updater.AddUpdate(g.update)
}

func GetGame() bootstrap.GameInterface { return &Game{} }
```

## Phase 3: Wire getGame() in the Engine

The engine's `getGame()` is defined per build-tag file in `kaiju/src/`:

| File | Build tag | Purpose |
|------|-----------|---------|
| `main.ed.go` | `editor && !rawsrc` | Editor with embedded FS |
| `main.ed.dbg.go` | `editor && rawsrc` | Editor reading from disk |
| `main.test.go` | `!editor` | Stock demo game (bouncing sphere) |
| `main.std.go` | all | `_main()` calling `getGame()` |

For a standalone game, **back up** `main.test.go` (it also defines
`getGame()` for `!editor` — two definitions won't compile):

```bash
mv kaiju/src/main.test.go kaiju/src/main.test.go.bak
```

Create `kaiju/src/main.marrow.go` (or `<project>.go`):

```go
//go:build !editor

package main

import (
    "kaijuengine.com/bootstrap"
    "marrow-of-empire/game"
)

func getGame() bootstrap.GameInterface { return game.GetGame() }
```

## Phase 4: Set Up game_content/

The `FileDatabase` reads keys **flat from the root** — but engine lookups use
extension-routed paths (`swapchain.renderpass` lives at
`renderer/passes/swapchain.renderpass`). Two options:

**Option A (recommended): extension-mapping wrapper.** Wrap the flat
database with the editor's `toEmbedPath` mapping. See
`src/game/content_db.go` in Marrow of Empire for the full implementation.
Key rules:

| Extension | Directory |
|-----------|-----------|
| `.bin` | `fonts/` |
| `.fbx` `.gltf` | `meshes/` |
| `.png` | search `textures/` → `fonts/` → `meshes/` in order |
| `.css` `.html` | `ui/` |
| `.material` | `renderer/materials/` |
| `.renderpass` | `renderer/passes/` |
| `.shaderpipeline` | `renderer/pipelines/` |
| `.shader` | `renderer/shaders/` |
| `.spv` | `renderer/spv/` |

**Option B: flatten everything** into `game_content/` root — works but
filenames can collide across subdirectories.

Then copy the engine's essential content (all paths relative to
`kaiju/src/editor/editor_embedded_content/editor_content/`):

```bash
mkdir -p game_content/{renderer/passes,renderer/spv,renderer/materials,renderer/shaders,renderer/pipelines,textures,fonts,meshes,ui}
cp editor_content/renderer/passes/*.renderpass   game_content/renderer/passes/
cp editor_content/renderer/spv/*                 game_content/renderer/spv/
cp editor_content/renderer/materials/*.material  game_content/renderer/materials/
cp editor_content/renderer/shaders/*.shader      game_content/renderer/shaders/
cp editor_content/renderer/pipelines/*.shaderpipeline game_content/renderer/pipelines/
cp editor_content/textures/*                     game_content/textures/
cp editor_content/fonts/*                        game_content/fonts/
cp editor_content/ui/*                           game_content/ui/
cp editor_content/meshes/*                       game_content/meshes/
```

## Phase 5: Build the UI with Markup, NOT Imperative Layout

**Critical lesson**: imperative `Layout().Scale(w, h)` on panels does not
size rendered geometry as expected in this engine version (panels rendered
at ~half-window size anchored top-left regardless of Scale values). The
markup system (`DocumentFromHTMLString`) handles root sizing, flex layout,
and CSS→layout mapping correctly — it is the path the engine's own console
uses.

```go
const titleHTML = `<!DOCTYPE html>
<html><head><style>
body { margin:0; padding:0; background-color:#141210;
       display:flex; flex-direction:column; align-items:center; }
</style></head>
<body> ... </body></html>`

doc := markup.DocumentFromHTMLString(&g.uiMan, titleHTML, "", nil, nil, nil)
if elm, ok := doc.GetElementById("my-btn"); ok {
    elm.UI.AddEvent(ui.EventTypeClick, func() { ... })
}
```

Signature: `DocumentFromHTMLString(uiMan, html, css string, withData any,
funcMap map[string]func(*document.Element), parent *document.Element)`.
`elm.UI` is already `*ui.UI` — no `.Base()` call.

## Phase 6: Build and Run

```makefile
build-standalone:
	cd kaiju/src && CGO_ENABLED=1 go build -tags="debug,ai_driver" -o ../../bin/game-standalone .
```

Run the **standalone** binary from the **project root** (its
`game_content/` path is CWD-relative). The **editor** build must run from
`kaiju/` (its `rawsrc` init does `os.OpenRoot("src")`).

```bash
./bin/game-standalone &
sleep 3
curl -s http://127.0.0.1:7777/v1/health
# → {"ok":true,"service":"kaiju-aidriver","version":"1"}
```

Then verify visually:

```bash
curl -s "http://127.0.0.1:7777/v1/screenshot?settle=3" --output /tmp/first.png
```

**If you cannot view images** (no vision support), verify content via pixel
analysis: `magick /tmp/first.png -unique-colors txt: | wc -l` — more than a
few hundred unique colors means text/UI is rendering; a count < 200 means
only flat fills. Sample expected palette colors with `-crop 1x1+X+Y`.

## Phase 7: Failure Mode Catalog

| Symptom | Cause | Fix |
|---------|-------|-----|
| `panic: asset database cannot be nil` | ContentDatabase returned nil | Return a real database (Phase 2) |
| `failed to initialize the GPU ... openat swapchain.renderpass` | No extension mapping / missing renderer content | Phase 4 wrapper + copies |
| `failed to load the text3d material ... openat text3d.shader` | Shaders not copied | Copy `renderer/shaders/` |
| `panic: assignment to entry in nil map` in `FontCache.createLetterMesh` | Font PNGs unreachable (textures/fonts/meshes search missing) | Phase 4 `.png` search order |
| `panic: invalid memory address` in `console.initialize` | `console.html` missing from `ui/` | Copy `editor_content/ui/` |
| `Could not find validation layer VK_LAYER_KHRONOS_validation` | Benign warning — no Vulkan SDK validation layers | Ignore (engine falls back) |
| Editor runs, game shows nothing | `rawsrc` editor build needs CWD=`kaiju/` | Run editor from `kaiju/` dir |
| Uniform gray/clear-color screen | UI not rendering (imperative Scale misuse) or content DB failing silently | Use markup (Phase 5); check logs |

## Phase 8: Record Evidence

- Save the first screenshot to `production/qa/evidence/screenshots/first-launch-[date].png`
- Note in `production/session-state/active.md`:
  - Binary name and build command
  - CWD requirement for each build type
  - Content database root and wrapper file
- Quit gracefully when done: `curl -s -X POST http://127.0.0.1:7777/v1/quit`

## Notes

- **AI Driver**: build with `ai_driver` tag; logs `AI Driver started
  addr=127.0.0.1:7777` at startup. Health endpoint confirms liveness.
- **Validation layers**: `VK_LAYER_KHRONOS_validation` missing is a WARN on
  Linux without Vulkan SDK — harmless for development.
- **Editor vs standalone**: the editor build (`editor` tag) embeds all
  engine content and doesn't need `game_content/` — but it shows the
  *editor*, not the game. Standalone (`!editor`) runs `getGame()` directly.
- **go.work**: root `go.work` includes `./src` and `./tests` but NOT
  `kaiju/src` — engine builds happen inside `kaiju/src/` with its own
  module context.
